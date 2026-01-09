package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	eightfold "course-sync/internal/providers/eightfold"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	baseURL := getenv("EIGHTFOLD_BASE_URL", "https://apiv2.eightfold.ai")
	ef := eightfold.New(baseURL)

	basic := os.Getenv("EIGHTFOLD_BASIC_AUTH")
	user := os.Getenv("EIGHTFOLD_USERNAME")
	pass := os.Getenv("EIGHTFOLD_PASSWORD")

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

	fmt.Println("OK: got token (len):", len(ef.BearerToken))

	courses, err := ef.ListCourses(ctx, 50)
	if err != nil {
		log.Fatalf("list courses error: %v", err)
	}

	fmt.Printf("OK: fetched %d courses\n", len(courses))
	for i, c := range courses {
		fmt.Printf("%d) %v\n", i+1, pick(c, "title", "courseId", "lmsCourseId", "provider", "status"))
	}
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
