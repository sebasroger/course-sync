package main

import (
	"context"
	"course-sync/internal/config"
	"course-sync/internal/providers/eightfold"
	"course-sync/internal/providers/pluralsight"
	"flag"
	"fmt"
	"log"
	"time"
)

func main() {
	var (
		limit  = flag.Int("limit", 100, "limit page size hint (default 100 = max)")
		dryRun = flag.Bool("dry-run", false, "fetch data but do not update Eightfold")
	)
	flag.Parse()

	start := time.Now()

	err := run(*limit, *dryRun)

	log.Printf("Execution finished in %s", time.Since(start))

	if err != nil {
		log.Fatalf("Job failed: %v", err)
	}
}

func run(limit int, dryRun bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Hour)
	defer cancel()

	cfg := config.Load()

	// 1. Init Eightfold
	if cfg.EightfoldBasicAuth == "" || cfg.EightfoldUser == "" || cfg.EightfoldPass == "" {
		return fmt.Errorf("missing env: EIGHTFOLD_BASIC_AUTH / EIGHTFOLD_USERNAME / EIGHTFOLD_PASSWORD")
	}
	ef := eightfold.New(cfg.EightfoldBaseURL)
	log.Printf("Authenticating with Eightfold...")
	if err := ef.Authenticate(ctx, cfg.EightfoldBasicAuth, eightfold.AuthRequest{
		GrantType: "password",
		Username:  cfg.EightfoldUser,
		Password:  cfg.EightfoldPass,
	}); err != nil {
		return fmt.Errorf("eightfold auth error: %w", err)
	}

	// 2. Init Pluralsight
	if cfg.PluralsightBaseURL == "" || cfg.PluralsightToken == "" {
		return fmt.Errorf("missing env: PLURALSIGHT_GQL_URL / PLURALSIGHT_TOKEN")
	}
	ps := pluralsight.New(cfg.PluralsightBaseURL, cfg.PluralsightToken)

	// 3. Fetch all EF users
	log.Printf("Fetching all employees from Eightfold...")
	users, err := ef.ListAllEmployees(ctx, limit)
	if err != nil {
		return fmt.Errorf("fetch employees error: %w", err)
	}
	log.Printf("Fetched %d users from Eightfold", len(users))

	// 4. Iterate and Sync
	for i, u := range users {
		profileID, _ := u["id"].(string) // or "employeeId" depending on API, usually "id" in core/employees
		if profileID == "" {
			// fallback
			profileID, _ = u["employeeId"].(string)
		}

		email, _ := u["email"].(string)
		if email == "" {
			email, _ = u["username"].(string)
		}

		if email == "" || profileID == "" {
			log.Printf("[%d/%d] SKIP: missing email or id (id=%s email=%s)", i+1, len(users), profileID, email)
			continue
		}

		log.Printf("[%d/%d] Processing %s (%s)...", i+1, len(users), email, profileID)

		// A. Get PS User ID
		psUser, err := ps.GetUserByEmail(ctx, email)
		if err != nil {
			log.Printf("  ERR: pluralsight user lookup failed: %v", err)
			continue
		}
		if psUser == nil {
			log.Printf("  SKIP: user not found in Pluralsight")
			continue
		}

		// B. Get PS Course Progress
		progressList, err := ps.GetCourseProgress(ctx, psUser.PsUserID)
		if err != nil {
			log.Printf("  ERR: pluralsight progress fetch failed: %v", err)
			continue
		}
		if len(progressList) == 0 {
			log.Printf("  INFO: no course progress found")
			continue
		}

		// C. Map to EF CourseAttendance
		var attendance []eightfold.CourseAttendance
		for _, p := range progressList {
			// Mapping:
			// "lmsCourseId": courseId
			// "title": course.title
			// "percentageCompletion": percentComplete
			// "status": "in_progress" (unless 100)
			// "startTs": firstViewedClipOn (parsed)
			// "durationHours": courseSeconds (converted to hours)
			// "provider": "Pluralsight"

			status := "in_progress"
			if p.PercentComplete >= 100.0 {
				status = "completed"
			}

			var startTs int64
			if p.FirstViewedClipOn != "" {
				if t, err := time.Parse(time.RFC3339, p.FirstViewedClipOn); err == nil {
					startTs = t.Unix()
				}
			}

			// Convert seconds to hours for durationHours
			durationHours := p.CourseSeconds / 3600.0

			attendance = append(attendance, eightfold.CourseAttendance{
				LmsCourseID:          p.CourseID,
				Title:                p.Course.Title,
				PercentageCompletion: p.PercentComplete,
				Status:               status,
				StartTs:              startTs,
				DurationHours:        durationHours,
				Provider:             "Pluralsight",
			})
		}

		// D. Patch EF User
		if len(attendance) > 0 {
			req := eightfold.UpdateEmployeeRequest{
				CourseAttendance: attendance,
			}

			if dryRun {
				log.Printf("  [DRY-RUN] Would patch %d courses for %s", len(attendance), email)
				// debug print one
				// b, _ := json.Marshal(attendance[0])
				// log.Printf("  Sample: %s", string(b))
			} else {
				if err := ef.UpdateEmployee(ctx, profileID, req); err != nil {
					log.Printf("  ERR: failed to update eightfold employee: %v", err)
				} else {
					log.Printf("  OK: updated %d courses", len(attendance))
				}
			}
		} else {
			log.Printf("  INFO: no courses to sync")
		}
	}
	return nil
}
