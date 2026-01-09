package providers

import (
	"context"
	"course-sync/internal/domain"
)

type CourseProvider interface {
	Name() string
	ListCourses(ctx context.Context) ([]domain.UnifiedCourse, error)
}
