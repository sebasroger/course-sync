package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"time"

	"course-sync/internal/config"
	"course-sync/internal/domain"
	"course-sync/internal/export"
	"course-sync/internal/providers/eightfold"
	"course-sync/internal/sftpclient"
)

func main() {
	var (
		outPath  = flag.String("out", "out/ef_emp_update.xml", "output xml path (Eightfold EF_Employee_List format)")
		upload   = flag.Bool("upload", false, "upload to SFTP after generating the file")
		pageSize = flag.Int("page-size", 500, "page size for Eightfold employees endpoint (if supported)")

		fieldName  = flag.String("field", "course_eligibility_tags", "custom_info field_name to set")
		badgeMerge = flag.String("badge-merge-strategy", "latest", "EF_Employee_List @badge_merge_strategy (empty to omit)")
	)
	flag.Parse()

	if *pageSize > 100 {
		log.Printf("page-size %d > 100, capping to 100 (Eightfold limit)", *pageSize)
		*pageSize = 100
	}

	rootCtx, cancel := context.WithTimeout(context.Background(), 6*time.Hour)
	defer cancel()

	cfg := config.Load()

	if strings.TrimSpace(cfg.EightfoldBaseURL) == "" {
		log.Fatal("missing env: EIGHTFOLD_BASE_URL")
	}

	start := time.Now()
	defer func() { log.Printf("job finished in %s", time.Since(start)) }()

	ef := eightfold.New(cfg.EightfoldBaseURL)

	// Auth: prefer bearer token if provided (matches your curl usage).
	ef.BearerToken = strings.TrimSpace(cfg.EightfoldBearerToken)
	if ef.BearerToken == "" {
		if cfg.EightfoldBasicAuth == "" || cfg.EightfoldUser == "" || cfg.EightfoldPass == "" {
			log.Fatal("missing auth: set EIGHTFOLD_BEARER_TOKEN or (EIGHTFOLD_BASIC_AUTH + EIGHTFOLD_USERNAME + EIGHTFOLD_PASSWORD)")
		}
		authCtx, authCancel := context.WithTimeout(rootCtx, 2*time.Minute)
		defer authCancel()
		if err := ef.Authenticate(authCtx, cfg.EightfoldBasicAuth, eightfold.AuthRequest{
			GrantType: "password",
			Username:  cfg.EightfoldUser,
			Password:  cfg.EightfoldPass,
		}); err != nil {
			log.Fatalf("eightfold auth failed: %v", err)
		}
	}

	listCtx, listCancel := context.WithTimeout(rootCtx, 6*time.Hour)
	defer listCancel()

	empMaps, err := ef.ListAllEmployees(listCtx, *pageSize)
	if err != nil {
		log.Fatal(err)
	}

	emps := make([]domain.UnifiedEmployee, 0, len(empMaps))
	missingID := 0
	for _, m := range empMaps {
		eid := pickString(m, "employee_id", "employeeId", "employeeID")
		uid := pickString(m, "user_id", "userId", "userID", "id")
		lvl := pickString(m, "level", "job_level", "jobLevel", "career_level", "careerLevel")

		emails := pickEmails(m)

		if strings.TrimSpace(eid) == "" {
			// Some tenants only expose user_id as the primary key.
			// We still keep the row, but count it so it's visible.
			missingID++
			eid = uid
		}

		emps = append(emps, domain.UnifiedEmployee{
			EmployeeID: eid,
			UserID:     uid,
			Level:      lvl,
			Emails:     emails,
		})
	}

	if missingID > 0 {
		log.Printf("WARN: %d employees had empty employee_id (used user_id instead)", missingID)
	}

	xCfg := export.EmployeeTagConfig{
		BadgeMergeStrategy: strings.TrimSpace(*badgeMerge),
		FieldName:          strings.TrimSpace(*fieldName),
	}
	if err := export.WriteEFEmployeeUpdateXML(*outPath, emps, xCfg); err != nil {
		log.Fatal(err)
	}

	log.Printf("wrote %d employees to %s", len(emps), *outPath)

	if *upload {
		remoteName := filepath.Base(*outPath)
		upCfg := sftpclient.Config{
			Host:                  cfg.SFTPHost,
			Port:                  cfg.SFTPPort,
			User:                  cfg.SFTPUser,
			Pass:                  cfg.SFTPPass,
			RemoteDir:             cfg.SFTPDir,
			InsecureIgnoreHostKey: cfg.SFTPInsecureIgnoreHostKey,
		}

		upCtx, upCancel := context.WithTimeout(rootCtx, 5*time.Minute)
		defer upCancel()

		if err := sftpclient.UploadFile(upCtx, upCfg, *outPath, remoteName); err != nil {
			log.Fatal(err)
		}
		log.Printf("uploaded to sftp://%s:%d%s/%s", upCfg.Host, upCfg.Port, upCfg.RemoteDir, remoteName)
	}
}

func pickString(m map[string]any, keys ...string) string {
	for _, k := range keys {
		v, ok := m[k]
		if !ok || v == nil {
			continue
		}
		s := anyToString(v)
		if strings.TrimSpace(s) != "" {
			return strings.TrimSpace(s)
		}
	}
	return ""
}

func pickEmails(m map[string]any) []string {
	// common keys
	keys := []string{"email", "emails", "email_list", "emailList", "email_list"}
	for _, k := range keys {
		if v, ok := m[k]; ok && v != nil {
			out := anyToStringSlice(v)
			if len(out) > 0 {
				return out
			}
		}
	}
	return nil
}

func anyToString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case fmt.Stringer:
		return t.String()
	default:
		return fmt.Sprint(v)
	}
}

func anyToStringSlice(v any) []string {
	out := []string{}
	switch t := v.(type) {
	case string:
		if strings.TrimSpace(t) != "" {
			out = append(out, strings.TrimSpace(t))
		}
	case []any:
		for _, item := range t {
			if item == nil {
				continue
			}
			// string
			if s, ok := item.(string); ok {
				s = strings.TrimSpace(s)
				if s != "" {
					out = append(out, s)
				}
				continue
			}
			// map with "email"
			if mm, ok := item.(map[string]any); ok {
				if e, ok := mm["email"]; ok {
					es := strings.TrimSpace(anyToString(e))
					if es != "" {
						out = append(out, es)
					}
				}
			}
		}
	case map[string]any:
		// Sometimes comes as {"email": "a@b"} or {"data": [...]}.
		if e, ok := t["email"]; ok {
			es := strings.TrimSpace(anyToString(e))
			if es != "" {
				out = append(out, es)
			}
		}
		if list, ok := t["data"]; ok {
			out = append(out, anyToStringSlice(list)...)
		}
	}

	// de-dupe
	seen := map[string]bool{}
	uniq := []string{}
	for _, s := range out {
		if s == "" {
			continue
		}
		if seen[s] {
			continue
		}
		seen[s] = true
		uniq = append(uniq, s)
	}
	return uniq
}
