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
	udemy "course-sync/internal/providers/udemy"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	pluralsightToken := os.Getenv("PLURALSIGHT_TOKEN")
	eightfoldBasicAuth := os.Getenv("EIGHTFOLD_BASIC_AUTH")
	eightfoldUser := os.Getenv("EIGHTFOLD_USERNAME")
	eightfoldPass := os.Getenv("EIGHTFOLD_PASSWORD")
	udemyClientId := os.Getenv("UDEMY_CLIENT_ID")
	udemyClientSecret := os.Getenv("UDEMY_CLIENT_SECRET")

	baseEightfoldURL := getenv("EIGHTFOLD_BASE_URL", "https://apiv2.eightfold.ai")
	basePluralsightURL := getenv("PLURALSIGHT_GQL_URL", "https://paas-api.pluralsight.com/graphql")
	baseUdemyURL := getenv("UDEMY_BASE_URL", "https://femsa.udemy.com/api-2.0/organizations/243186")

	if pluralsightToken == "" {
		log.Fatal("missing env var: PLURALSIGHT_TOKEN")
	}

	if eightfoldBasicAuth == "" || eightfoldUser == "" || eightfoldPass == "" {
		log.Fatal("missing env vars: EIGHTFOLD_BASIC_AUTH / EIGHTFOLD_USERNAME / EIGHTFOLD_PASSWORD")
	}

	ef := eightfold.New(baseEightfoldURL)
	ps := pluralsight.New(basePluralsightURL, pluralsightToken)
	ud := udemy.New(baseUdemyURL, udemyClientId, udemyClientSecret)

	if err := ef.Authenticate(ctx, eightfoldBasicAuth, eightfold.AuthRequest{
		GrantType: "password",
		Username:  eightfoldUser,
		Password:  eightfoldPass,
	}); err != nil {
		log.Fatalf("auth error: %v", err)
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
		fmt.Printf("%d) %v\n", i+1, pick(c, "title", "id", "url", "language"))
	}
}

func getenv(k, def string) string {
	v := os.Getenv(k)
	if v == "" {
		return def
	}
	return v
}

func pick(v any, keys ...string) map[string]any {
	b, err := json.Marshal(v)
	if err != nil {
		return map[string]any{}
	}

	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return map[string]any{}
	}

	out := map[string]any{}
	for _, k := range keys {
		if v, ok := m[k]; ok {
			out[k] = v
		}
	}
	return out
}
