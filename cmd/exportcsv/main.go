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

// splitCSV splits a comma-separated string into a slice of strings,
// trimming whitespace and removing empty entries.
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

func main() {
	var (
		outPath = flag.String("out", "out/COURSE-MAIN_ALL.csv", "output csv path")
		upload  = flag.Bool("upload", false, "upload to SFTP after generating the file")

		udemyPages = flag.Int("udemy-max-pages", 1, "max pages to fetch from udemy (0 = all)")
		psPages    = flag.Int("ps-max-pages", 1, "max pages to fetch from pluralsight (0 = all)")
		pageSize   = flag.Int("page-size", 100, "page size for providers (Udemy page_size / Pluralsight first). Udemy will be clamped to its max.")

		// Eligibility tags temporalmente deshabilitados
		// udemyTags = flag.String("udemy-tags", "IC1,IC2,IC3,IC4", "eligibility tags for Udemy courses (comma-separated)")
		// psTags    = flag.String("pluralsight-tags", "IC5,IC6,IC7,M1,M2,M3", "eligibility tags for Pluralsight courses (comma-separated)")
	)
	flag.Parse()

	rootCtx, cancel := context.WithTimeout(context.Background(), 6*time.Hour)
	defer cancel()

	cfg := config.Load()

	start := time.Now()
	defer func() {
		log.Printf("job finished in %s", time.Since(start))
	}()

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

	// Eligibility tags temporalmente deshabilitados
	// tagCfg := export.CourseTagConfig{
	// 	EligibilityTagsFieldName: "eligibility_tags",
	// 	TagsBySource: map[string][]string{
	// 		"udemy":       splitCSV(*udemyTags),
	// 		"pluralsight": splitCSV(*psTags),
	// 	},
	// }

	// Configuración vacía ya que no se usarán tags por ahora
	tagCfg := export.CourseTagConfig{}

	// Use the CSV writer with the tag configuration
	if err := export.WriteEightfoldCourseCSV(*outPath, filtered, tagCfg); err != nil {
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

	if *upload {
		remoteName := filepath.Base(*outPath)

		// Verificar que el archivo local existe antes de intentar subirlo
		if _, err := os.Stat(*outPath); os.IsNotExist(err) {
			log.Fatalf("Error: El archivo local %s no existe", *outPath)
		}

		log.Printf("Iniciando subida SFTP del archivo %s", *outPath)

		// Usar la ruta completa que sabemos que funciona
		upCfg := sftpclient.Config{
			Host:                  cfg.SFTPHost,
			Port:                  cfg.SFTPPort,
			User:                  cfg.SFTPUser,
			Pass:                  cfg.SFTPPass,
			RemoteDir:             "/ef-sftp/femsa-sandbox/home/inbound", // Usar la ruta completa con el directorio inbound
			InsecureIgnoreHostKey: cfg.SFTPInsecureIgnoreHostKey,
			HostKey:               cfg.SFTPHostKey,
			KeyPath:               cfg.SFTPKeyPath,
			KeyPassphrase:         cfg.SFTPKeyPassphrase,
		}

		// Mostrar la configuración SFTP (sin mostrar contraseñas)
		log.Printf("Configuración SFTP: Host=%s, Port=%d, User=%s, RemoteDir=%s",
			upCfg.Host, upCfg.Port, upCfg.User, upCfg.RemoteDir)

		upCtx, upCancel := context.WithTimeout(rootCtx, 5*time.Minute)
		defer upCancel()

		log.Printf("Subiendo archivo %s a %s:%d%s/%s...", *outPath, upCfg.Host, upCfg.Port, upCfg.RemoteDir, remoteName)
		if err := sftpclient.UploadFile(upCtx, upCfg, *outPath, remoteName); err != nil {
			log.Fatalf("Error al subir archivo: %v", err)
		}
		log.Printf("¡Subida exitosa! Archivo disponible en sftp://%s:%d%s/%s", upCfg.Host, upCfg.Port, upCfg.RemoteDir, remoteName)
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
