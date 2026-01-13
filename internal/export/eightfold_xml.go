package export

import (
	"encoding/xml"
	"fmt"
	"os"
	"strings"

	"course-sync/internal/domain"
)

/*
Eightfold XML examples (from ef_course_add.xml / ef_course_update.xml):

<EF_Course_List>
  <EF_Course operation="upsert">
    <title>...</title>
    <lms_course_id>...</lms_course_id>
    <description>...</description>
    <course_type>Course</course_type>
    <language>en</language>
    <course_url>...</course_url>
    <duration_hours>1.5</duration_hours>
    <difficulty>...</difficulty>
    <category>...</category>
    <provider>Udemy</provider>
    <status>active</status>
    <published_ts>2020-10-01T07:19:45</published_ts>
    <skills_list>
      <skill>...</skill>
    </skills_list>
    <custom_multi_value_list>
      <custom_mv_field>
        <field_name>eligibility_tags</field_name>
        <data_type>string</data_type>
        <data_list>
          <field_value>IC1</field_value>
        </data_list>
      </custom_mv_field>
    </custom_multi_value_list>
  </EF_Course>
</EF_Course_List>
*/

type efCourseList struct {
	XMLName xml.Name   `xml:"EF_Course_List"`
	Courses []efCourse `xml:"EF_Course"`
}

type efCourse struct {
	Operation string `xml:"operation,attr,omitempty"`

	Title       string `xml:"title,omitempty"`
	LMSCourseID string `xml:"lms_course_id"`
	Description string `xml:"description,omitempty"`

	CourseType string `xml:"course_type,omitempty"`
	Language   string `xml:"language,omitempty"`

	CourseURL     string `xml:"course_url,omitempty"`
	DurationHours string `xml:"duration_hours,omitempty"`

	Difficulty string `xml:"difficulty,omitempty"`
	Category   string `xml:"category,omitempty"`
	Provider   string `xml:"provider,omitempty"`

	Status      string `xml:"status,omitempty"`
	PublishedTS string `xml:"published_ts,omitempty"`

	SkillsList *efSkillsList `xml:"skills_list,omitempty"`

	CustomInfo          *efCustomInfo          `xml:"custom_info,omitempty"`
	CustomMultiValueList *efCustomMultiValueList `xml:"custom_multi_value_list,omitempty"`
}

type efSkillsList struct {
	Skills []string `xml:"skill"`
}

type efCustomInfo struct {
	Fields []efCustomField `xml:"custom_field"`
}

type efCustomField struct {
	FieldName  string `xml:"field_name"`
	DataType   string `xml:"data_type"`
	FieldValue string `xml:"field_value"`
}

type efCustomMultiValueList struct {
	Fields []efCustomMVField `xml:"custom_mv_field"`
}

type efCustomMVField struct {
	FieldName string    `xml:"field_name"`
	DataType  string    `xml:"data_type"`
	DataList  efDataList `xml:"data_list"`
}

type efDataList struct {
	FieldValues []string `xml:"field_value"`
}

type CourseTagConfig struct {
	// If Operation is set, it will be included as EF_Course @operation="...".
	Operation string

	// eligibility_tags custom field (multi value list)
	EligibilityTagsFieldName string // default: "eligibility_tags"

	// Maps course.Source -> tags (e.g. "udemy" -> {"IC1","IC2"...})
	TagsBySource map[string][]string
}

// WriteEFCourseXML writes a single XML file (ef_course_add/update) including eligibility_tags.
// This matches Eightfold's single-file XML option for course ingestion.
func WriteEFCourseXML(outPath string, courses []domain.UnifiedCourse, cfg CourseTagConfig) error {
	fieldName := strings.TrimSpace(cfg.EligibilityTagsFieldName)
	if fieldName == "" {
		fieldName = "eligibility_tags"
	}

	out := efCourseList{
		Courses: make([]efCourse, 0, len(courses)),
	}

	for _, c := range courses {
		lmsID := buildSystemID(c.Source, c.SourceID)

		lang := normalizeLang(c.Language)
		provider := strings.Title(strings.ToLower(c.Source)) // "Udemy", "Pluralsight"

		row := efCourse{
			Operation:   strings.TrimSpace(cfg.Operation),
			Title:       strings.TrimSpace(c.Title),
			LMSCourseID: lmsID,
			Description: strings.TrimSpace(c.Description),

			CourseType: "Course",
			Language:   lang,

			CourseURL: strings.TrimSpace(c.CourseURL),

			Difficulty: strings.TrimSpace(c.Difficulty),
			Category:   strings.TrimSpace(c.Category),
			Provider:   provider,

			Status:      strings.TrimSpace(c.Status),
			PublishedTS: strings.TrimSpace(c.PublishedDate),
		}

		if c.DurationHours > 0 {
			row.DurationHours = floatToString(c.DurationHours)
		}

		if len(c.Skills) > 0 {
			row.SkillsList = &efSkillsList{Skills: c.Skills}
		}

		// eligibility_tags per source
		tags := cfg.TagsBySource[strings.ToLower(strings.TrimSpace(c.Source))]
		tags = compactStrings(tags)

		if len(tags) == 1 {
			row.CustomInfo = &efCustomInfo{
				Fields: []efCustomField{{
					FieldName:  fieldName,
					DataType:   "string",
					FieldValue: tags[0],
				}},
			}
		} else if len(tags) > 1 {
			row.CustomMultiValueList = &efCustomMultiValueList{
				Fields: []efCustomMVField{{
					FieldName: fieldName,
					DataType:  "string",
					DataList:  efDataList{FieldValues: tags},
				}},
			}
		}

		out.Courses = append(out.Courses, row)
	}

	b, err := xml.MarshalIndent(out, "", "  ")
	if err != nil {
		return fmt.Errorf("export: marshal xml: %w", err)
	}

	if err := os.WriteFile(outPath, append([]byte(xml.Header), b...), 0o644); err != nil {
		return fmt.Errorf("export: write xml: %w", err)
	}

	return nil
}

func compactStrings(in []string) []string {
	out := make([]string, 0, len(in))
	seen := map[string]bool{}
	for _, s := range in {
		v := strings.TrimSpace(s)
		if v == "" {
			continue
		}
		if seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	return out
}

// normalizeLang maps provider language strings to short tags ("en","es","pt") and keeps existing tags.
func normalizeLang(lang string) string {
	s := strings.TrimSpace(strings.ToLower(lang))
	if s == "" {
		return ""
	}
	s = strings.ReplaceAll(s, "_", "-")

	switch s {
	case "english":
		return "en"
	case "spanish", "español", "espanol":
		return "es"
	case "portuguese", "português", "portugues":
		return "pt"
	}

	// Accept variants like en-us, pt-br, es-mx
	if strings.HasPrefix(s, "en") {
		return "en"
	}
	if strings.HasPrefix(s, "es") {
		return "es"
	}
	if strings.HasPrefix(s, "pt") {
		return "pt"
	}
	return s
}
