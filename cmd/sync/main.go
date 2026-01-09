package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"course-sync/internal/config"
	"course-sync/internal/devutil"
	"course-sync/internal/providers/eightfold"
	"course-sync/internal/providers/pluralsight"
	"course-sync/internal/providers/udemy"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	cfg := config.Load()

	// Eightfold envs (modo password grant)
	if cfg.EightfoldBasicAuth == "" || cfg.EightfoldUser == "" || cfg.EightfoldPass == "" {
		log.Fatal("missing env: EIGHTFOLD_BASIC_AUTH / EIGHTFOLD_USERNAME / EIGHTFOLD_PASSWORD")
	}

	ef := eightfold.New(cfg.EightfoldBaseURL)
	ps := pluralsight.New(cfg.PluralsightBaseURL, cfg.PluralsightToken)
	ud := udemy.New(cfg.UdemyBaseURL, cfg.UdemyClientID, cfg.UdemyClientSecret)

	if err := ef.Authenticate(ctx, cfg.EightfoldBasicAuth, eightfold.AuthRequest{
		GrantType: "password",
		Username:  cfg.EightfoldUser,
		Password:  cfg.EightfoldPass,
	}); err != nil {
		log.Fatalf("eightfold auth error: %v", err)
	}

	var cursor *string
	first := 50 // chico para no sobrecargar
	totalFetched := 0

	for page := 1; page <= 2; page++ { // probemos 2 páginas
		res, err := ps.ListCoursesPage(ctx, first, cursor)
		if err != nil {
			log.Fatalf("list page error: %v", err)
		}

		fmt.Println("total:", res.Data.CourseCatalog.TotalCount)

		time.Sleep(300 * time.Millisecond) // pequeño “respeto” al server
	}

	fmt.Println("done. fetched:", totalFetched)

	courses, err := ud.ListCourses(ctx, 50, 2)
	if err != nil {
		log.Fatalf("list courses error: %v", err)
	}
	for i, c := range courses {
		fmt.Printf("%d) %v\n", i+1, devutil.Pick(c, "title", "id", "url", "language"))
	}
}
