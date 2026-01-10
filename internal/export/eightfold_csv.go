package export

import (
    "course-sync/internal/domain"
    "encoding/csv"
    "io"
    "strconv"
    "strings"
)

// Eightfold CSV template (based on COURSE-MAIN_PLURAL.csv).
// Keep header order EXACT.
var eightfoldHeader = []string{
    "COURSE_ID",
    "COURSE_TITLE",
    "COURSE_DESCRIPTION",
    "COURSE_URL",
    "DURATION_HOURS",
    "CATEGORY",
    "REQUIREMENTS",
    "COURSE_TYPE",
    "LANGUAGE",
    "DIFFICULTY",
    "STATUS",
    "IMAGE_URL",
    "SKILLS",
    "PROFICIENCY_LIST",
    "PROVIDER",
    "FEE_VALUE",
    "FEE_CURRENCY",
    "LOCATION",
    "PUBLISHED_TS",
    "SENIORITY_LIST",
    "JOB_FUNCTION_LIST",
    "JOB_CODE_LIST",
    "BUSINESS_UNIT_LIST",
    "RATING",
    "RATING_COUNT",
}

// WriteEightfoldCSV writes courses in the Eightfold import format.
// It intentionally leaves non-mapped columns empty.
func WriteEightfoldCSV(w io.Writer, courses []domain.UnifiedCourse) error {
    cw := csv.NewWriter(w)
    // match typical templates
    cw.UseCRLF = true

    if err := cw.Write(eightfoldHeader); err != nil {
        return err
    }

    for _, c := range courses {
        row := toEightfoldRow(c)
        if err := cw.Write(row); err != nil {
            return err
        }
    }
    cw.Flush()
    return cw.Error()
}

func toEightfoldRow(c domain.UnifiedCourse) []string {
    // defaults
    courseType := "Online"
    status := c.Status
    if status == "" {
        status = "active"
    }

    // joiners
    skills := ""
    if len(c.Skills) > 0 {
        // avoid commas to keep CSV clean
        skills = strings.Join(cleanStrings(c.Skills), " | ")
    }

    duration := ""
    if c.DurationHours > 0 {
        // keep it simple: plain decimal
        duration = strconv.FormatFloat(c.DurationHours, 'f', -1, 64)
    }

    // Provider: keep first letter uppercase, rest lowercase ("udemy" -> "Udemy")
    provider := strings.TrimSpace(c.Source)
    if provider != "" {
        provider = strings.ToUpper(provider[:1]) + strings.ToLower(provider[1:])
    }

    // NOTE: published format is whatever the provider gives (ISO string). If unknown, keep empty.
    published := strings.TrimSpace(c.PublishedDate)

    return []string{
        c.SourceID,          // COURSE_ID
        c.Title,             // COURSE_TITLE
        c.Description,       // COURSE_DESCRIPTION
        c.CourseURL,         // COURSE_URL
        duration,            // DURATION_HOURS
        c.Category,          // CATEGORY
        "",                 // REQUIREMENTS
        courseType,          // COURSE_TYPE
        c.Language,          // LANGUAGE
        c.Difficulty,        // DIFFICULTY
        status,              // STATUS
        c.ImageURL,          // IMAGE_URL
        skills,              // SKILLS
        "",                 // PROFICIENCY_LIST
        provider,            // PROVIDER
        "",                 // FEE_VALUE
        "",                 // FEE_CURRENCY
        "",                 // LOCATION
        published,           // PUBLISHED_TS
        "",                 // SENIORITY_LIST
        "",                 // JOB_FUNCTION_LIST
        "",                 // JOB_CODE_LIST
        "",                 // BUSINESS_UNIT_LIST
        "",                 // RATING
        "",                 // RATING_COUNT
    }
}

func cleanStrings(in []string) []string {
    out := make([]string, 0, len(in))
    for _, s := range in {
        s = strings.TrimSpace(s)
        if s == "" {
            continue
        }
        // avoid newlines / commas
        s = strings.ReplaceAll(s, "\n", " ")
        s = strings.ReplaceAll(s, "\r", " ")
        out = append(out, s)
    }
    return out
}
