package main

import (
    "context"
    "flag"
    "log"
    "os"
    "time"

    "course-sync/internal/config"
    "course-sync/internal/domain"
    "course-sync/internal/export"
    "course-sync/internal/providers/pluralsight"
    "course-sync/internal/providers/udemy"
)

func main() {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
    defer cancel()

    outPath := flag.String("out", "eightfold-courses.csv", "output csv path")
    udemyPageSize := flag.Int("udemy-page-size", 100, "udemy page size")
    udemyMaxPages := flag.Int("udemy-max-pages", 0, "udemy max pages (<=0 = all)")
    psFirst := flag.Int("ps-first", 100, "pluralsight page size")
    psMaxPages := flag.Int("ps-max-pages", 0, "pluralsight max pages (<=0 = all)")
    flag.Parse()

    cfg := config.Load()

    psClient := pluralsight.New(cfg.PluralsightBaseURL, cfg.PluralsightToken)
    udClient := udemy.New(cfg.UdemyBaseURL, cfg.UdemyClientID, cfg.UdemyClientSecret)

    psProv := pluralsight.Provider{C: psClient, First: *psFirst, MaxPages: *psMaxPages}
    udProv := udemy.Provider{C: udClient, PageSize: *udemyPageSize, MaxPages: *udemyMaxPages}

    courses := make([]domain.UnifiedCourse, 0, 4096)

    psCourses, err := psProv.ListCourses(ctx)
    if err != nil {
        log.Fatalf("pluralsight list error: %v", err)
    }
    courses = append(courses, psCourses...)

    udCourses, err := udProv.ListCourses(ctx)
    if err != nil {
        log.Fatalf("udemy list error: %v", err)
    }
    courses = append(courses, udCourses...)

    f, err := os.Create(*outPath)
    if err != nil {
        log.Fatalf("create csv: %v", err)
    }
    defer f.Close()

    if err := export.WriteEightfoldCSV(f, courses); err != nil {
        log.Fatalf("write csv: %v", err)
    }

    log.Printf("done. wrote %d courses to %s\n", len(courses), *outPath)
}
