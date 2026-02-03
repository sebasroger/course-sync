package main

import (
	"context"
	"course-sync/internal/config"
	"course-sync/internal/providers/eightfold"
	"course-sync/internal/providers/pluralsight"
	"course-sync/internal/providers/udemy"
	"flag"
	"fmt"
	"log"
	"strings"
	"time"
)

// Estructura para mantener los clientes inicializados
type clients struct {
	eightfold   *eightfold.Client
	pluralsight *pluralsight.Client
	udemy       *udemy.Client
}

func main() {
	var (
		limit  = flag.Int("limit", 100, "limit page size hint (default 100 = max)")
		dryRun = flag.Bool("dry-run", false, "fetch data but do not update Eightfold")
	)
	flag.Parse()

	// Medir tiempo total de ejecución
	start := time.Now()

	err := run(*limit, *dryRun)

	log.Printf("Execution finished in %s", time.Since(start))

	if err != nil {
		log.Fatalf("Job failed: %v", err)
	}
}

// Inicializa todos los clientes necesarios
func initializeClients(ctx context.Context, cfg *config.Config) (*clients, error) {
	// 1. Init Eightfold
	if cfg.EightfoldBasicAuth == "" || cfg.EightfoldUser == "" || cfg.EightfoldPass == "" {
		return nil, fmt.Errorf("missing env: EIGHTFOLD_BASIC_AUTH / EIGHTFOLD_USERNAME / EIGHTFOLD_PASSWORD")
	}

	ef := eightfold.New(cfg.EightfoldBaseURL)
	log.Printf("Authenticating with Eightfold...")
	if err := ef.Authenticate(ctx, cfg.EightfoldBasicAuth, eightfold.AuthRequest{
		GrantType: "password",
		Username:  cfg.EightfoldUser,
		Password:  cfg.EightfoldPass,
	}); err != nil {
		return nil, fmt.Errorf("eightfold auth error: %w", err)
	}

	// 2. Init Pluralsight
	var psClient *pluralsight.Client
	if cfg.PluralsightBaseURL != "" && cfg.PluralsightToken != "" {
		psClient = pluralsight.New(cfg.PluralsightBaseURL, cfg.PluralsightToken)
		log.Printf("Pluralsight client initialized")
	} else {
		log.Printf("Skipping Pluralsight integration: missing env variables")
	}

	// 3. Init Udemy
	var udemyClient *udemy.Client
	if cfg.UdemyBaseURL != "" && cfg.UdemyClientID != "" && cfg.UdemyClientSecret != "" {
		udemyClient = udemy.New(cfg.UdemyBaseURL, cfg.UdemyClientID, cfg.UdemyClientSecret)
		log.Printf("Udemy client initialized")
	} else {
		log.Printf("Skipping Udemy integration: missing env variables")
	}

	return &clients{
		eightfold:   ef,
		pluralsight: psClient,
		udemy:       udemyClient,
	}, nil
}

// Helper function to process Pluralsight courses for a user
func processPluralsightCourses(ctx context.Context, ps *pluralsight.Client, psUser *pluralsight.UserNode) ([]eightfold.CourseAttendance, error) {
	progressList, err := ps.GetCourseProgress(ctx, psUser.PsUserID)
	if err != nil {
		return nil, err
	}

	if len(progressList) == 0 {
		return nil, nil
	}

	var attendance []eightfold.CourseAttendance
	for _, p := range progressList {
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

	return attendance, nil
}

// Helper function to process Udemy courses for a user
func processUdemyCourses(ctx context.Context, uClient *udemy.Client, email string) ([]eightfold.CourseAttendance, error) {
	// 1. Look up the user in Udemy by email
	udemyUser, err := uClient.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("udemy user lookup failed: %w", err)
	}
	if udemyUser == nil {
		// User not found in Udemy
		return []eightfold.CourseAttendance{}, nil
	}

	// 2. Get the user's course progress
	progressList, err := uClient.GetCourseProgress(ctx, udemyUser.UdemyUserID)
	if err != nil {
		return nil, fmt.Errorf("udemy course progress fetch failed: %w", err)
	}
	if len(progressList) == 0 {
		return []eightfold.CourseAttendance{}, nil
	}

	// 3. Convert to eightfold.CourseAttendance format
	var attendance []eightfold.CourseAttendance
	for _, p := range progressList {
		status := "in_progress"
		if p.IsCourseCompleted || p.PercentComplete >= 100.0 {
			status = "completed"
		}

		var startTs int64
		if p.FirstViewedLectureOn != "" {
			if t, err := time.Parse(time.RFC3339, p.FirstViewedLectureOn); err == nil {
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
			Provider:             "Udemy",
		})
	}

	return attendance, nil
}

func run(limit int, dryRun bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Hour)
	defer cancel()

	// Medir tiempo de inicialización
	initStart := time.Now()

	cfg := config.Load()

	// 1. Inicializar clientes
	clients, err := initializeClients(ctx, &cfg)
	if err != nil {
		return err
	}

	log.Printf("Clients initialized in %s", time.Since(initStart))

	// 2. Fetch all EF users with only the fields we need
	fetchStart := time.Now()
	log.Printf("Fetching all employees from Eightfold...")
	// Solo traemos los campos que necesitamos: id, email, username
	users, err := clients.eightfold.ListEmployeesFields(ctx, limit, []string{"id", "email", "username", "employeeId"})
	if err != nil {
		return fmt.Errorf("fetch employees error: %w", err)
	}
	log.Printf("Fetched %d users from Eightfold in %s", len(users), time.Since(fetchStart))

	// Estructura para resultados de procesamiento de usuario
	type userProcessResult struct {
		index       int
		email       string
		profileID   string
		attendance  []eightfold.CourseAttendance
		processTime time.Duration
		err         error
	}

	// 3. Iterate and Sync con procesamiento paralelo
	syncStart := time.Now()
	processed := 0
	skipped := 0
	updated := 0
	errorCount := 0

	// Configuración de paralelismo
	workers := 10 // Número de trabajadores en paralelo
	sem := make(chan struct{}, workers)
	resultsCh := make(chan userProcessResult, len(users))

	// Función para procesar un usuario
	processUser := func(i int, u map[string]any) {
		defer func() { <-sem }()

		// Medir tiempo de procesamiento por usuario
		userStart := time.Now()

		profileID, _ := u["id"].(string) // or "employeeId" depending on API, usually "id" in core/employees
		if profileID == "" {
			// fallback
			profileID, _ = u["employeeId"].(string)
		}

		email, _ := u["email"].(string)
		if email == "" {
			email, _ = u["username"].(string)
		}

		// Temporary patch: Remove "-sandbox" from email addresses
		email = strings.Replace(email, "-sandbox", "", 1)

		if email == "" || profileID == "" {
			resultsCh <- userProcessResult{
				index:     i,
				email:     email,
				profileID: profileID,
				err:       fmt.Errorf("missing email or id"),
			}
			return
		}

		// Crear un contexto específico para este usuario
		userCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
		defer cancel()

		// Initialize attendance array for this user
		var attendance []eightfold.CourseAttendance

		// Process Pluralsight courses
		if clients.pluralsight != nil {
			psUser, err := clients.pluralsight.GetUserByEmail(userCtx, email)
			if err == nil && psUser != nil {
				psAttendance, err := processPluralsightCourses(userCtx, clients.pluralsight, psUser)
				if err == nil && len(psAttendance) > 0 {
					attendance = append(attendance, psAttendance...)
				}
			}
		}

		// Process Udemy courses
		if clients.udemy != nil {
			udemyAttendance, err := processUdemyCourses(userCtx, clients.udemy, email)
			if err == nil && len(udemyAttendance) > 0 {
				attendance = append(attendance, udemyAttendance...)
			}
		}

		// Enviar resultado
		resultsCh <- userProcessResult{
			index:       i,
			email:       email,
			profileID:   profileID,
			attendance:  attendance,
			processTime: time.Since(userStart),
		}
	}

	// Lanzar procesamiento en paralelo
	for i, u := range users {
		sem <- struct{}{}
		go processUser(i, u)
	}

	// Recoger resultados y actualizar Eightfold
	for range users {
		result := <-resultsCh
		i := result.index
		email := result.email
		profileID := result.profileID
		attendance := result.attendance

		if result.err != nil {
			log.Printf("[%d/%d] SKIP: %v (id=%s email=%s)", i+1, len(users), result.err, profileID, email)
			skipped++
			continue
		}

		log.Printf("[%d/%d] Processing %s (%s)...", i+1, len(users), email, profileID)
		processed++

		// Mostrar resultados de cursos
		psCount := 0
		udemyCount := 0
		for _, course := range attendance {
			if course.Provider == "Pluralsight" {
				psCount++
			} else if course.Provider == "Udemy" {
				udemyCount++
			}
		}

		if psCount > 0 {
			log.Printf("  INFO: found %d Pluralsight courses", psCount)
		}
		if udemyCount > 0 {
			log.Printf("  INFO: found %d Udemy courses", udemyCount)
		}

		// Patch EF User with combined courses
		if len(attendance) > 0 {
			req := eightfold.UpdateEmployeeRequest{
				Email: email,
				CandidateData: eightfold.CandidateData{
					CourseAttendance: attendance,
				},
			}

			if dryRun {
				log.Printf("  [DRY-RUN] Would patch %d courses for %s", len(attendance), email)
			} else {
				if err := clients.eightfold.UpdateEmployee(ctx, profileID, req); err != nil {
					log.Printf("  ERR: failed to update eightfold employee: %v", err)
					errorCount++
				} else {
					log.Printf("  OK: updated %d courses", len(attendance))
					updated++
				}
			}
		} else {
			log.Printf("  INFO: no courses to sync")
		}

		log.Printf("  Processed in %s", result.processTime)
	}

	// Resumen final
	totalTime := time.Since(syncStart)
	log.Printf("Sync summary: processed=%d, updated=%d, skipped=%d, errors=%d, total_time=%s",
		processed, updated, skipped, errorCount, totalTime)
	return nil
}
