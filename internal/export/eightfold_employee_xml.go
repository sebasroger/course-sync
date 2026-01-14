package export

import (
	"encoding/xml"
	"fmt"
	"os"
	"strings"

	"course-sync/internal/domain"
)

/*
Example from Eightfold docs/sample:

<EF_Employee_List badge_merge_strategy='latest'>
  <EF_Employee>
    <employee_id>...</employee_id>
    <user_id>...</user_id>
    <email_list><email>...</email></email_list>
    <level>IC5</level>
    <custom_info>
      <custom_field>
        <field_name>course_eligibility_tags</field_name>
        <data_type>string</data_type>
        <field_value>UDEMY</field_value>
      </custom_field>
    </custom_info>
  </EF_Employee>
</EF_Employee_List>
*/

type efEmployeeList struct {
	XMLName           xml.Name     `xml:"EF_Employee_List"`
	BadgeMergeStrategy string       `xml:"badge_merge_strategy,attr,omitempty"`
	Employees         []efEmployee  `xml:"EF_Employee"`
}

type efEmployee struct {
	EmployeeID string `xml:"employee_id"`
	UserID     string `xml:"user_id,omitempty"`
	EmailList  *efEmailList `xml:"email_list,omitempty"`
	Level      string `xml:"level,omitempty"`

	CustomInfo *efCustomInfo `xml:"custom_info,omitempty"`
}

type efEmailList struct {
	Emails []string `xml:"email"`
}

// efCustomInfo / efCustomField are also used by course XML exporter.
// (they are defined in eightfold_xml.go)

type EmployeeTagConfig struct {
	BadgeMergeStrategy string
	FieldName          string // default: course_eligibility_tags
}

func WriteEFEmployeeUpdateXML(outPath string, emps []domain.UnifiedEmployee, cfg EmployeeTagConfig) error {
	fieldName := strings.TrimSpace(cfg.FieldName)
	if fieldName == "" {
		fieldName = "course_eligibility_tags"
	}

	out := efEmployeeList{
		BadgeMergeStrategy: strings.TrimSpace(cfg.BadgeMergeStrategy),
		Employees:          make([]efEmployee, 0, len(emps)),
	}

	for _, e := range emps {
		row := efEmployee{
			EmployeeID: strings.TrimSpace(e.EmployeeID),
			UserID:     strings.TrimSpace(e.UserID),
			Level:      strings.TrimSpace(e.Level),
		}

		emails := compactStrings(e.Emails)
		if len(emails) > 0 {
			row.EmailList = &efEmailList{Emails: emails}
		}

		// always emit the course eligibility field
		tag := EligibilityProviderFromLevel(e.Level)
		row.CustomInfo = &efCustomInfo{Fields: []efCustomField{{
			FieldName:  fieldName,
			DataType:   "string",
			FieldValue: tag,
		}}}

		out.Employees = append(out.Employees, row)
	}

	b, err := xml.MarshalIndent(out, "", "  ")
	if err != nil {
		return fmt.Errorf("export: marshal employee xml: %w", err)
	}
	if err := os.WriteFile(outPath, append([]byte(xml.Header), b...), 0o644); err != nil {
		return fmt.Errorf("export: write employee xml: %w", err)
	}
	return nil
}

// EligibilityProviderFromLevel returns UDEMY if level starts with IC (case-insensitive), otherwise PLURALSIGHT.
func EligibilityProviderFromLevel(level string) string {
	lvl := strings.ToUpper(strings.TrimSpace(level))
	if strings.HasPrefix(lvl, "IC") {
		return "UDEMY"
	}
	return "PLURALSIGHT"
}
