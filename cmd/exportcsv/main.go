package main

import (
	"context"
	"flag"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"course-sync/internal/config"
	"course-sync/internal/domain"
	"course-sync/internal/export"
	"course-sync/internal/providers/pluralsight"
	"course-sync/internal/providers/udemy"
	"course-sync/internal/sftpclient"
)

type provResult struct {
	name    string
	courses []domain.UnifiedCourse
	err     error
}

func main() {
	var (
		outPath    = flag.String("out", "COURSE-MAIN_ALL.csv", "output csv path")
		udemyPages = flag.Int("udemy-max-pages", 1, "max pages to fetch from udemy (0 = all)")
		psPages    = flag.Int("ps-max-pages", 1, "max pages to fetch from pluralsight (0 = all)")
		pageSize   = flag.Int("page-size", 100, "page size for providers (when supported)")
		uploadSFTP = flag.Bool("sftp", false, "upload the generated CSV via SFTP")
	)
	flag.Parse()

	// timeout general grande
	rootCtx, rootCancel := context.WithTimeout(context.Background(), 8*time.Hour)
	defer rootCancel()

	cfg := config.Load()

	// asegura dir de salida
	if dir := filepath.Dir(*outPath); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			log.Fatal(err)
		}
	}

	u := udemy.New(cfg.UdemyBaseURL, cfg.UdemyClientID, cfg.UdemyClientSecret)
	p := pluralsight.New(cfg.PluralsightBaseURL, cfg.PluralsightToken)

	resultsCh := make(chan provResult, 2)

	// Udemy en paralelo (ctx propio)
	go func() {
		ctx, cancel := context.WithTimeout(rootCtx, 6*time.Hour)
		defer cancel()

		udProv := udemy.Provider{
			C:        u,
			PageSize: *pageSize,
			MaxPages: *udemyPages,
		}
		courses, err := udProv.ListCourses(ctx)
		resultsCh <- provResult{name: "udemy", courses: courses, err: err}
	}()

	// Pluralsight en paralelo (ctx propio)
	go func() {
		ctx, cancel := context.WithTimeout(rootCtx, 6*time.Hour)
		defer cancel()

		psProv := pluralsight.Provider{
			C:        p,
			First:    *pageSize,
			MaxPages: *psPages,
		}
		courses, err := psProv.ListCourses(ctx)
		resultsCh <- provResult{name: "pluralsight", courses: courses, err: err}
	}()

	var all []domain.UnifiedCourse
	totalByProvider := map[string]int{}

	for i := 0; i < 2; i++ {
		r := <-resultsCh
		totalByProvider[r.name] = len(r.courses)

		if r.err != nil {
			log.Printf("WARN: %s failed: %v (using %d courses fetched)", r.name, r.err, len(r.courses))
		}
		all = append(all, r.courses...)
	}

	filtered := filterCoursesByLang(all, map[string]bool{
		"es": true,
		"en": true,
		"pt": true,
	})

	if err := export.WriteEightfoldCourseCSV(*outPath, filtered); err != nil {
		log.Fatal(err)
	}

	log.Printf(
		"wrote %d courses to %s (udemy=%d, pluralsight=%d, merged=%d)",
		len(filtered),
		*outPath,
		totalByProvider["udemy"],
		totalByProvider["pluralsight"],
		len(all),
	)

	if *uploadSFTP {
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

func filterCoursesByLang(courses []domain.UnifiedCourse, allowed map[string]bool) []domain.UnifiedCourse {
	out := make([]domain.UnifiedCourse, 0, len(courses))
	for _, c := range courses {
		lang := normalizeLang(c.Language)
		if allowed[lang] {
			out = append(out, c)
		}
	}
	return out
}

func normalizeLang(lang string) string {
	s := strings.TrimSpace(strings.ToLower(lang))
	if s == "" {
		return ""
	}
	s = strings.ReplaceAll(s, "_", "-")

	switch s {
	case "english":
		return "en"
	case "spanish", "español", "espanol":
		return "es"
	case "portuguese", "português", "portugues":
		return "pt"
	}

	if len(s) >= 2 {
		return s[:2]
	}
	return s
}
