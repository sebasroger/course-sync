package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	eightfold "course-sync/internal/providers/eightfold"
	pluralsight "course-sync/internal/providers/pluralsight"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	token := os.Getenv("PLURALSIGHT_TOKEN")
	basic := os.Getenv("EIGHTFOLD_BASIC_AUTH")
	user := os.Getenv("EIGHTFOLD_USERNAME")
	pass := os.Getenv("EIGHTFOLD_PASSWORD")

	baseEightfoldURL := getenv("EIGHTFOLD_BASE_URL", "https://apiv2.eightfold.ai")
	basePluralsightURL := getenv("PLURALSIGHT_GQL_URL", "https://paas-api.pluralsight.com/graphql")

	if token == "" {
		log.Fatal("missing env var: PLURALSIGHT_TOKEN")
	}

	ef := eightfold.New(baseEightfoldURL)
	ps := pluralsight.New(basePluralsightURL, token)

	if basic == "" || user == "" || pass == "" {
		log.Fatal("missing env vars: EIGHTFOLD_BASIC_AUTH / EIGHTFOLD_USERNAME / EIGHTFOLD_PASSWORD")
	}

	if err := ef.Authenticate(ctx, basic, eightfold.AuthRequest{
		GrantType: "password",
		Username:  user,
		Password:  pass,
	}); err != nil {
		log.Fatalf("auth error: %v", err)
	}
	/*
		fmt.Println("OK: got token (len):", len(ef.BearerToken))

		courses, err := ef.ListCourses(ctx, 50)
		if err != nil {
			log.Fatalf("list courses error: %v", err)
		}

		fmt.Printf("OK: fetched %d courses\n", len(courses))
		for i, c := range courses {
			fmt.Printf("%d) %v\n", i+1, pick(c, "title", "courseId", "lmsCourseId", "provider", "status"))
		}
	*/
	var cursor *string
	first := 50 // chico para no sobrecargar
	totalFetched := 0

	for page := 1; page <= 2; page++ { // probemos 2 páginas
		res, err := ps.ListCoursesPage(ctx, first, cursor)
		if err != nil {
			log.Fatalf("list page error: %v", err)
		}

		fmt.Println("total:", res.Data.CourseCatalog.TotalCount)
		fmt.Println("hasNext:", res.Data.CourseCatalog.PageInfo.HasNextPage)
		fmt.Println("endCursor:", res.Data.CourseCatalog.PageInfo.EndCursor)
		fmt.Println("nodes:", len(res.Data.CourseCatalog.Nodes))

		// Si querés verlo EXACTO como JSON:
		b, _ := json.MarshalIndent(res, "", "  ")
		fmt.Println(string(b))

		time.Sleep(300 * time.Millisecond) // pequeño “respeto” al server
	}

	fmt.Println("done. fetched:", totalFetched)
}

func getenv(k, def string) string {
	v := os.Getenv(k)
	if v == "" {
		return def
	}
	return v
}

func pick(m map[string]any, keys ...string) map[string]any {
	out := map[string]any{}
	for _, k := range keys {
		if v, ok := m[k]; ok {
			out[k] = v
		}
	}
	return out
}
