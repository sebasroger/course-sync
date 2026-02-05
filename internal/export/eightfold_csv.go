package export

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"strings"

	"course-sync/internal/domain"
)

var header = []string{
	"systemId",
	"title",
	"description",
	"courseUrl",
	"durationHours",
	"category",
	"imageUrl",
	"language",
	"publishedDate",
	"lmsCourseId",
	"difficulty",
	"provider",
	"status",
	// "eligibility_tags", // Temporalmente deshabilitado, se implementará en el futuro
}

func WriteEightfoldCourseCSV(outPath string, courses []domain.UnifiedCourse, cfg CourseTagConfig) error {
	f, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("export: create csv: %w", err)
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	if err := w.Write(header); err != nil {
		return fmt.Errorf("export: write header: %w", err)
	}

	for _, c := range courses {
		systemID := buildSystemID(c.Source, c.SourceID)
		status := c.Status
		if status == "" {
			status = "active"
		}

		// Eligibility tags temporalmente deshabilitados
		// tags := cfg.TagsBySource[strings.ToLower(strings.TrimSpace(c.Source))]
		// tagsStr := strings.Join(compactStrings(tags), ",")

		row := []string{
			systemID,
			c.Title,
			c.Description,
			c.CourseURL,
			floatToString(c.DurationHours),
			c.Category,
			c.ImageURL,
			firstNonEmpty(c.Language), // si viene vacío igual queda vacío
			c.PublishedDate,
			c.SourceID, // lmsCourseId
			c.Difficulty,
			c.Source, // provider
			status,
			// tagsStr, // Temporalmente deshabilitado
		}

		if err := w.Write(row); err != nil {
			return fmt.Errorf("export: write row: %w", err)
		}
	}

	if err := w.Error(); err != nil {
		return fmt.Errorf("export: csv error: %w", err)
	}

	return nil
}

func buildSystemID(source, sourceID string) string {
	switch strings.ToLower(source) {
	case "udemy":
		return "UDM+" + sourceID
	case "pluralsight":
		return "PLS+" + sourceID
	default:
		prefix := strings.ToUpper(source)
		if prefix == "" {
			prefix = "SRC"
		}
		return prefix + "+" + sourceID
	}
}

func floatToString(v float64) string {
	// Ej: 1.5, 2, 0
	return strconv.FormatFloat(v, 'f', -1, 64)
}

func firstNonEmpty(v string) string { return v }
