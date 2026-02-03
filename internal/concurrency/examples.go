package concurrency

import (
	"context"
	"fmt"
	"log"
	"time"
)

// Este archivo contiene ejemplos de cómo usar las utilidades de procesamiento paralelo
// para diferentes casos de uso en la aplicación course-sync.

// UserProcessResult es un ejemplo de estructura de resultado para procesamiento de usuarios
type UserProcessResult struct {
	UserID      string
	Email       string
	Courses     []interface{}
	ProcessTime time.Duration
	Error       error
}

// CourseProcessResult es un ejemplo de estructura de resultado para procesamiento de cursos
type CourseProcessResult struct {
	CourseID    string
	Title       string
	Provider    string
	ProcessTime time.Duration
	Error       error
}

// EjemploSincronizacionEmpleados muestra cómo usar ProcessParallel para sincronizar empleados
func EjemploSincronizacionEmpleados(
	ctx context.Context,
	users []map[string]interface{},
	maxWorkers int,
	procesadorUsuario func(ctx context.Context, userID string, email string) ([]interface{}, error),
) ([]UserProcessResult, []error) {
	// Configurar opciones de paralelismo
	opts := ParallelOptions{
		MaxWorkers: maxWorkers,
	}

	// Procesar todos los usuarios en paralelo
	results, errors := ProcessParallel(
		ctx,
		users,
		opts,
		func(ctx context.Context, i int, user map[string]interface{}) (UserProcessResult, error) {
			// Medir tiempo de procesamiento por usuario
			userStart := time.Now()

			// Extraer información del usuario
			userID, _ := user["id"].(string)
			if userID == "" {
				// Intentar con employeeId como fallback
				userID, _ = user["employeeId"].(string)
			}

			email, _ := user["email"].(string)
			if email == "" {
				// Intentar con username como fallback
				email, _ = user["username"].(string)
			}

			if email == "" || userID == "" {
				return UserProcessResult{}, fmt.Errorf("missing email or id for user at index %d", i)
			}

			// Crear un contexto específico para este usuario con timeout
			userCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
			defer cancel()

			// Procesar el usuario (esta función la proporciona el llamador)
			courses, err := procesadorUsuario(userCtx, userID, email)
			
			// Devolver resultado
			return UserProcessResult{
				UserID:      userID,
				Email:       email,
				Courses:     courses,
				ProcessTime: time.Since(userStart),
				Error:       err,
			}, nil
		},
	)

	// Procesar y registrar resultados
	for i, result := range results {
		if i < len(errors) && errors[i] != nil {
			log.Printf("[%d/%d] Error processing user %s: %v", 
				i+1, len(users), result.Email, errors[i])
			continue
		}

		log.Printf("[%d/%d] Processed user %s in %s with %d courses", 
			i+1, len(users), result.Email, result.ProcessTime, len(result.Courses))
	}

	return results, errors
}

// EjemploSincronizacionCursos muestra cómo usar ProcessParallel para sincronizar cursos
func EjemploSincronizacionCursos(
	ctx context.Context,
	courses []map[string]interface{},
	maxWorkers int,
	procesadorCurso func(ctx context.Context, courseID string, title string) error,
) []error {
	// Configurar opciones de paralelismo
	opts := ParallelOptions{
		MaxWorkers: maxWorkers,
	}

	// Procesar todos los cursos en paralelo sin recolectar resultados
	return ForEach(
		ctx,
		courses,
		opts,
		func(ctx context.Context, i int, course map[string]interface{}) error {
			// Medir tiempo de procesamiento por curso
			courseStart := time.Now()

			// Extraer información del curso
			courseID, _ := course["id"].(string)
			title, _ := course["title"].(string)

			if courseID == "" {
				return fmt.Errorf("missing course ID for course at index %d", i)
			}

			// Crear un contexto específico para este curso con timeout
			courseCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
			defer cancel()

			// Procesar el curso (esta función la proporciona el llamador)
			err := procesadorCurso(courseCtx, courseID, title)
			
			// Registrar resultado
			if err != nil {
				log.Printf("[%d/%d] Error processing course %s: %v", 
					i+1, len(courses), title, err)
			} else {
				log.Printf("[%d/%d] Processed course %s in %s", 
					i+1, len(courses), title, time.Since(courseStart))
			}
			
			return err
		},
	)
}

// Ejemplo de uso:
/*
func main() {
	ctx := context.Background()
	
	// 1. Ejemplo de sincronización de empleados
	users := []map[string]interface{}{
		{"id": "1", "email": "user1@example.com"},
		{"id": "2", "email": "user2@example.com"},
		// ...
	}
	
	results, errors := concurrency.EjemploSincronizacionEmpleados(
		ctx,
		users,
		10, // maxWorkers
		func(ctx context.Context, userID string, email string) ([]interface{}, error) {
			// Aquí iría la lógica para procesar un usuario
			// Por ejemplo, obtener sus cursos de Pluralsight y Udemy
			return []interface{}{
				map[string]interface{}{"courseId": "c1", "title": "Course 1"},
				// ...
			}, nil
		},
	)
	
	// 2. Ejemplo de sincronización de cursos
	courses := []map[string]interface{}{
		{"id": "c1", "title": "Course 1"},
		{"id": "c2", "title": "Course 2"},
		// ...
	}
	
	errors := concurrency.EjemploSincronizacionCursos(
		ctx,
		courses,
		10, // maxWorkers
		func(ctx context.Context, courseID string, title string) error {
			// Aquí iría la lógica para procesar un curso
			return nil
		},
	)
}
*/
