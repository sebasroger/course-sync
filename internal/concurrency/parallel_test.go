package concurrency

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions()
	if opts.MaxWorkers != 10 {
		t.Errorf("Expected MaxWorkers to be 10, got %d", opts.MaxWorkers)
	}
}

func TestProcessParallel(t *testing.T) {
	ctx := context.Background()

	// Test with empty slice
	results, errs := ProcessParallel(ctx, []int{}, DefaultOptions(), func(ctx context.Context, index int, item int) (string, error) {
		return "", nil
	})
	if len(results) != 0 {
		t.Errorf("Expected empty results for empty input, got %d items", len(results))
	}
	if errs != nil {
		t.Errorf("Expected nil errors for empty input, got %v", errs)
	}

	// Test with normal operation
	input := []int{1, 2, 3, 4, 5}
	results, errs = ProcessParallel(ctx, input, DefaultOptions(), func(ctx context.Context, index int, item int) (string, error) {
		return string(rune('a' + item - 1)), nil
	})
	if len(results) != len(input) {
		t.Errorf("Expected %d results, got %d", len(input), len(results))
	}
	if len(errs) != 0 {
		t.Errorf("Expected no errors, got %d", len(errs))
	}
	expected := []string{"a", "b", "c", "d", "e"}
	for i, res := range results {
		if res != expected[i] {
			t.Errorf("Expected result at index %d to be %s, got %s", i, expected[i], res)
		}
	}

	// Test with errors
	results, errs = ProcessParallel(ctx, input, DefaultOptions(), func(ctx context.Context, index int, item int) (string, error) {
		if item%2 == 0 {
			return "", errors.New("even number error")
		}
		return string(rune('a' + item - 1)), nil
	})
	if len(results) != len(input) {
		t.Errorf("Expected %d results, got %d", len(input), len(results))
	}
	if len(errs) != 2 {
		t.Errorf("Expected 2 errors, got %d", len(errs))
	}

	// Test with custom options
	opts := ParallelOptions{MaxWorkers: 2}
	results, errs = ProcessParallel(ctx, input, opts, func(ctx context.Context, index int, item int) (string, error) {
		return string(rune('a' + item - 1)), nil
	})
	if len(results) != len(input) {
		t.Errorf("Expected %d results, got %d", len(input), len(results))
	}
	if len(errs) != 0 {
		t.Errorf("Expected no errors, got %d", len(errs))
	}

	// Test with invalid MaxWorkers
	opts = ParallelOptions{MaxWorkers: -1}
	results, errs = ProcessParallel(ctx, input, opts, func(ctx context.Context, index int, item int) (string, error) {
		return string(rune('a' + item - 1)), nil
	})
	if len(results) != len(input) {
		t.Errorf("Expected %d results, got %d", len(input), len(results))
	}
	if len(errs) != 0 {
		t.Errorf("Expected no errors, got %d", len(errs))
	}

	// Test with context cancellation
	cancelCtx, cancel := context.WithCancel(ctx)
	cancel() // Cancel immediately
	results, errs = ProcessParallel(cancelCtx, input, DefaultOptions(), func(ctx context.Context, index int, item int) (string, error) {
		// This should not be executed due to context cancellation
		time.Sleep(100 * time.Millisecond)
		return string(rune('a' + item - 1)), nil
	})
	// We still expect results array to be fully populated, but with zero values
	if len(results) != len(input) {
		t.Errorf("Expected %d results, got %d", len(input), len(results))
	}
}

func TestForEach(t *testing.T) {
	ctx := context.Background()

	// Test with empty slice
	errs := ForEach(ctx, []int{}, DefaultOptions(), func(ctx context.Context, index int, item int) error {
		return nil
	})
	if errs != nil {
		t.Errorf("Expected nil errors for empty input, got %v", errs)
	}

	// Test with normal operation
	input := []int{1, 2, 3, 4, 5}
	results := make([]string, len(input))
	errs = ForEach(ctx, input, DefaultOptions(), func(ctx context.Context, index int, item int) error {
		results[index] = string(rune('a' + item - 1))
		return nil
	})
	if len(errs) != 0 {
		t.Errorf("Expected no errors, got %d", len(errs))
	}
	expected := []string{"a", "b", "c", "d", "e"}
	for i, res := range results {
		if res != expected[i] {
			t.Errorf("Expected result at index %d to be %s, got %s", i, expected[i], res)
		}
	}

	// Test with errors
	errs = ForEach(ctx, input, DefaultOptions(), func(ctx context.Context, index int, item int) error {
		if item%2 == 0 {
			return errors.New("even number error")
		}
		return nil
	})
	if len(errs) != 2 {
		t.Errorf("Expected 2 errors, got %d", len(errs))
	}

	// Test with custom options
	opts := ParallelOptions{MaxWorkers: 2}
	errs = ForEach(ctx, input, opts, func(ctx context.Context, index int, item int) error {
		return nil
	})
	if len(errs) != 0 {
		t.Errorf("Expected no errors, got %d", len(errs))
	}

	// Test with invalid MaxWorkers
	opts = ParallelOptions{MaxWorkers: -1}
	errs = ForEach(ctx, input, opts, func(ctx context.Context, index int, item int) error {
		return nil
	})
	if len(errs) != 0 {
		t.Errorf("Expected no errors, got %d", len(errs))
	}

	// Test with context cancellation
	cancelCtx, cancel := context.WithCancel(ctx)
	cancel() // Cancel immediately
	errs = ForEach(cancelCtx, input, DefaultOptions(), func(ctx context.Context, index int, item int) error {
		// This should not be executed due to context cancellation
		time.Sleep(100 * time.Millisecond)
		return nil
	})
	// We don't expect any errors since the context cancellation should prevent execution
	if len(errs) != 0 {
		t.Errorf("Expected no errors with cancelled context, got %d", len(errs))
	}
}

// Test to ensure results are returned in the correct order despite parallel execution
func TestProcessParallelOrder(t *testing.T) {
	ctx := context.Background()
	input := []int{5, 3, 1, 4, 2} // Unordered input

	results, errs := ProcessParallel(ctx, input, DefaultOptions(), func(ctx context.Context, index int, item int) (int, error) {
		// Simulate varying processing times
		time.Sleep(time.Duration(item) * 10 * time.Millisecond)
		return item, nil
	})

	if len(errs) != 0 {
		t.Errorf("Expected no errors, got %d", len(errs))
	}

	// Results should match the original input order, not execution completion order
	for i, res := range results {
		if res != input[i] {
			t.Errorf("Expected result at index %d to be %d, got %d", i, input[i], res)
		}
	}

	// No need to verify if input was sorted or not, we just need to ensure
	// that the results match the input order regardless of execution time
}
