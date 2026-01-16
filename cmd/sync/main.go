package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"course-sync/internal/config"
	"course-sync/internal/domain"
	"course-sync/internal/export"
	"course-sync/internal/providers/eightfold"
	"course-sync/internal/providers/pluralsight"
	"course-sync/internal/providers/udemy"
	syncx "course-sync/internal/sync"
)

// Sync command:
// - Fetch Udemy + Pluralsight catalog
// - Fetch Eightfold existing courses
// - Diff -> upsert XML + delete XML
// - Optional mock-dir for deterministic runs

func main() {
	var (
		// Backward compatible combined file (create+update). If provided, we also write it.
		outUpsert = flag.String("out-upsert", "", "(deprecated) output xml path for combined upserts (creates+updates). If set, we also write this file")

		// New: separate files.
		outAdd    = flag.String("out-add", "out/ef_course_add.xml", "output xml path for creates (Eightfold ef_course_add format)")
		outUpdate = flag.String("out-update", "out/ef_course_update.xml", "output xml path for updates (Eightfold ef_course_update format)")
		outDelete = flag.String("out-delete", "out/ef_course_delete.xml", "output xml path for deletes (Eightfold ef_course_delete format)")

		systemID = flag.String("system-id", "successfactors", "value to write into <system_id>. Use empty string to keep legacy prefixed ids")

		udemyPages = flag.Int("udemy-max-pages", 1, "max pages to fetch from udemy (0 = all)")
		psPages    = flag.Int("ps-max-pages", 1, "max pages to fetch from pluralsight (0 = all)")
		pageSize   = flag.Int("page-size", 100, "page size for providers (Udemy page_size / Pluralsight first). Udemy will be clamped to its max.")

		udemyTags = flag.String("udemy-tags", "IC1,IC2,IC3,IC4", "eligibility tags for Udemy courses (comma-separated)")
		psTags    = flag.String("pluralsight-tags", "IC5,IC6,IC7,M1,M2,M3", "eligibility tags for Pluralsight courses (comma-separated)")
		op        = flag.String("operation", "upsert", "EF_Course @operation attribute value (empty to omit)")

		mockDir     = flag.String("mock-dir", "", "read catalogs from JSON snapshots in this directory (udemy.json, pluralsight.json, eightfold.json) instead of calling APIs")
		snapshotDir = flag.String("snapshot-dir", "", "if set, write JSON snapshots (udemy.json, pluralsight.json, eightfold.json) to this directory")
		dryRun      = flag.Bool("dry-run", false, "do not write XML files; only print counts")
	)
	flag.Parse()

	rootCtx, cancel := context.WithTimeout(context.Background(), 6*time.Hour)
	defer cancel()

	start := time.Now()
	defer func() { log.Printf("job finished in %s", time.Since(start)) }()

	// Fetch
	var (
		providerCourses []domain.UnifiedCourse
		efCourses       []syncx.EFCourse
		err             error
	)

	if strings.TrimSpace(*mockDir) != "" {
		providerCourses, efCourses, err = loadFromMocks(*mockDir)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		cfg := config.Load()

		// Eightfold envs (password grant)
		if cfg.EightfoldBasicAuth == "" || cfg.EightfoldUser == "" || cfg.EightfoldPass == "" {
			log.Fatal("missing env: EIGHTFOLD_BASIC_AUTH / EIGHTFOLD_USERNAME / EIGHTFOLD_PASSWORD")
		}

		ef := eightfold.New(cfg.EightfoldBaseURL)
		ps := pluralsight.New(cfg.PluralsightBaseURL, cfg.PluralsightToken)
		ud := udemy.New(cfg.UdemyBaseURL, cfg.UdemyClientID, cfg.UdemyClientSecret)

		if err := ef.Authenticate(rootCtx, cfg.EightfoldBasicAuth, eightfold.AuthRequest{
			GrantType: "password",
			Username:  cfg.EightfoldUser,
			Password:  cfg.EightfoldPass,
		}); err != nil {
			log.Fatalf("eightfold auth error: %v", err)
		}

		// Providers
		providerCourses, err = fetchProviders(rootCtx, ud, ps, *pageSize, *udemyPages, *psPages)
		if err != nil {
			log.Fatalf("providers fetch error: %v", err)
		}

		// Eightfold
		efCourses, err = syncx.FetchEightfoldCourses(rootCtx, ef, 100, 0) // limit=100; maxPages=0 means auto until done (best effort)
		if err != nil {
			log.Fatalf("eightfold list error: %v", err)
		}
	}

	// Optional snapshots
	if strings.TrimSpace(*snapshotDir) != "" {
		if err := writeSnapshots(*snapshotDir, providerCourses, efCourses); err != nil {
			log.Fatalf("write snapshots error: %v", err)
		}
	}

	// Diff
	create, update, del := syncx.Diff(providerCourses, efCourses)

	log.Printf("diff: create=%d update=%d delete=%d (providers=%d, eightfold=%d)", len(create), len(update), len(del), len(providerCourses), len(efCourses))

	if *dryRun {
		return
	}

	tagCfg := export.CourseTagConfig{
		Operation:                strings.TrimSpace(*op),
		SystemID:                 strings.TrimSpace(*systemID),
		EligibilityTagsFieldName: "eligibility_tags",
		TagsBySource: map[string][]string{
			"udemy":       splitCSV(*udemyTags),
			"pluralsight": splitCSV(*psTags),
		},
	}

	// Separate files (recommended)
	if err := export.WriteEFCourseXML(*outAdd, create, tagCfg); err != nil {
		log.Fatal(err)
	}
	if err := export.WriteEFCourseXML(*outUpdate, update, tagCfg); err != nil {
		log.Fatal(err)
	}
	if err := export.WriteEFCourseDeleteXML(*outDelete, del); err != nil {
		log.Fatal(err)
	}

	// Optional combined file for backward compatibility
	if strings.TrimSpace(*outUpsert) != "" {
		upserts := append(create, update...)
		if err := export.WriteEFCourseXML(*outUpsert, upserts, tagCfg); err != nil {
			log.Fatal(err)
		}
	}
}

func fetchProviders(
	ctx context.Context,
	ud *udemy.Client,
	ps *pluralsight.Client,
	pageSize int,
	udemyPages int,
	psPages int,
) ([]domain.UnifiedCourse, error) {
	type provResult struct {
		name    string
		courses []domain.UnifiedCourse
		err     error
	}
	resultsCh := make(chan provResult, 2)

	go func() {
		uctx, cancel := context.WithTimeout(ctx, 6*time.Hour)
		defer cancel()
		udProv := udemy.Provider{C: ud, PageSize: pageSize, MaxPages: udemyPages}
		courses, err := udProv.ListCourses(uctx)
		resultsCh <- provResult{name: "udemy", courses: courses, err: err}
	}()

	go func() {
		pctx, cancel := context.WithTimeout(ctx, 6*time.Hour)
		defer cancel()
		psProv := pluralsight.Provider{C: ps, First: pageSize, MaxPages: psPages}
		courses, err := psProv.ListCourses(pctx)
		resultsCh <- provResult{name: "pluralsight", courses: courses, err: err}
	}()

	var all []domain.UnifiedCourse
	for i := 0; i < 2; i++ {
		r := <-resultsCh
		if r.err != nil {
			// keep partial results
			log.Printf("WARN: %s failed: %v (using %d courses fetched)", r.name, r.err, len(r.courses))
		}
		all = append(all, r.courses...)
	}
	return all, nil
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		v := strings.TrimSpace(p)
		if v != "" {
			out = append(out, v)
		}
	}
	return out
}

func loadFromMocks(dir string) ([]domain.UnifiedCourse, []syncx.EFCourse, error) {
	read := func(name string, v any) error {
		p := filepath.Join(dir, name)
		b, err := os.ReadFile(p)
		if err != nil {
			return fmt.Errorf("mock: read %s: %w", p, err)
		}
		if err := json.Unmarshal(b, v); err != nil {
			return fmt.Errorf("mock: decode %s: %w", p, err)
		}
		return nil
	}

	var ud []domain.UnifiedCourse
	var ps []domain.UnifiedCourse
	var ef []syncx.EFCourse
	if err := read("udemy.json", &ud); err != nil {
		return nil, nil, err
	}
	if err := read("pluralsight.json", &ps); err != nil {
		return nil, nil, err
	}
	if err := read("eightfold.json", &ef); err != nil {
		return nil, nil, err
	}
	all := append(ud, ps...)
	return all, ef, nil
}

func writeSnapshots(dir string, prov []domain.UnifiedCourse, ef []syncx.EFCourse) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	// Split provider snapshots by source for convenience.
	var ud, ps []domain.UnifiedCourse
	for _, c := range prov {
		switch strings.ToLower(strings.TrimSpace(c.Source)) {
		case "udemy":
			ud = append(ud, c)
		case "pluralsight":
			ps = append(ps, c)
		}
	}

	write := func(name string, v any) error {
		b, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(dir, name), b, 0o644)
	}

	if err := write("udemy.json", ud); err != nil {
		return err
	}
	if err := write("pluralsight.json", ps); err != nil {
		return err
	}
	if err := write("eightfold.json", ef); err != nil {
		return err
	}
	return nil
}
