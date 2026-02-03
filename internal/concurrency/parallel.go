package concurrency

import (
	"context"
	"sync"
)

// ParallelOptions configura el comportamiento del procesamiento paralelo
type ParallelOptions struct {
	// MaxWorkers es el número máximo de trabajadores en paralelo
	MaxWorkers int
}

// DefaultOptions devuelve opciones predeterminadas para procesamiento paralelo
func DefaultOptions() ParallelOptions {
	return ParallelOptions{
		MaxWorkers: 10,
	}
}

// ProcessParallel procesa elementos en paralelo usando la función de trabajo proporcionada
// itemFunc se llama para cada elemento y debe devolver un resultado y/o error
// Devuelve los resultados en el mismo orden que los elementos de entrada
func ProcessParallel[T any, R any](
	ctx context.Context,
	items []T,
	opts ParallelOptions,
	itemFunc func(ctx context.Context, index int, item T) (R, error),
) ([]R, []error) {
	if len(items) == 0 {
		return []R{}, nil
	}

	maxWorkers := opts.MaxWorkers
	if maxWorkers <= 0 {
		maxWorkers = 10 // Default to 10 workers if not specified
	}

	// Use fewer workers if we have fewer items
	if maxWorkers > len(items) {
		maxWorkers = len(items)
	}

	// Create channels for work distribution and result collection
	jobs := make(chan int, len(items))
	results := make(chan struct {
		index  int
		result R
		err    error
	}, len(items))

	// Start workers
	var wg sync.WaitGroup
	for w := 0; w < maxWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for jobIndex := range jobs {
				select {
				case <-ctx.Done():
					return
				default:
					result, err := itemFunc(ctx, jobIndex, items[jobIndex])
					results <- struct {
						index  int
						result R
						err    error
					}{jobIndex, result, err}
				}
			}
		}()
	}

	// Send jobs to workers
	for i := range items {
		jobs <- i
	}
	close(jobs)

	// Wait for all workers to finish in a separate goroutine
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	resultList := make([]R, len(items))
	var errors []error

	for i := 0; i < len(items); i++ {
		res := <-results
		if res.err != nil {
			errors = append(errors, res.err)
		}
		resultList[res.index] = res.result
	}

	return resultList, errors
}

// ForEach ejecuta una función para cada elemento en paralelo, sin recolectar resultados
// Útil cuando solo necesitas efectos secundarios y no te importan los resultados
func ForEach[T any](
	ctx context.Context,
	items []T,
	opts ParallelOptions,
	itemFunc func(ctx context.Context, index int, item T) error,
) []error {
	if len(items) == 0 {
		return nil
	}

	maxWorkers := opts.MaxWorkers
	if maxWorkers <= 0 {
		maxWorkers = 10 // Default to 10 workers if not specified
	}

	// Use fewer workers if we have fewer items
	if maxWorkers > len(items) {
		maxWorkers = len(items)
	}

	// Create channels for work distribution and result collection
	jobs := make(chan int, len(items))
	errors := make(chan error, len(items))

	// Start workers
	var wg sync.WaitGroup
	for w := 0; w < maxWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for jobIndex := range jobs {
				select {
				case <-ctx.Done():
					return
				default:
					err := itemFunc(ctx, jobIndex, items[jobIndex])
					if err != nil {
						errors <- err
					}
				}
			}
		}()
	}

	// Send jobs to workers
	for i := range items {
		jobs <- i
	}
	close(jobs)

	// Wait for all workers to finish
	wg.Wait()
	close(errors)

	// Collect errors
	var errorList []error
	for err := range errors {
		errorList = append(errorList, err)
	}

	return errorList
}
