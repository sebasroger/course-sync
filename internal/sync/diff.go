package sync

import (
	"math"
	"strings"

	"course-sync/internal/domain"
	"course-sync/internal/export"
)

// Diff compares provider courses (Udemy + Pluralsight) with the current Eightfold catalog.
// Returns:
// - create: present in providers but not in Eightfold
// - update: present in both but changed
// - del: present in Eightfold but not in providers (only for managed IDs, i.e. UDM+/PLS+)
func Diff(provider []domain.UnifiedCourse, eightfold []EFCourse) (create []domain.UnifiedCourse, update []domain.UnifiedCourse, del []export.DeleteCourse) {
	provByID := map[string]domain.UnifiedCourse{}
	for _, c := range provider {
		id := BuildSystemID(c.Source, c.SourceID)
		if strings.TrimSpace(id) == "" {
			continue
		}
		provByID[id] = c
	}

	efByID := map[string]EFCourse{}
	for _, c := range eightfold {
		id := strings.TrimSpace(firstNonEmpty(c.LMSCourseID, c.SystemID))
		if id == "" {
			continue
		}
		efByID[id] = c
	}

	// create/update
	for id, pc := range provByID {
		efc, ok := efByID[id]
		if !ok {
			create = append(create, pc)
			continue
		}
		if needsUpdate(pc, efc) {
			update = append(update, pc)
		}
	}

	// deletes
	for id, efc := range efByID {
		if _, ok := provByID[id]; ok {
			continue
		}
		del = append(del, export.DeleteCourse{Title: strings.TrimSpace(efc.Title), LMSCourseID: id})
	}

	return create, update, del
}

func needsUpdate(p domain.UnifiedCourse, e EFCourse) bool {
	// Normalize provider fields similarly to export layer.
	// If Eightfold data is missing a field, we avoid triggering update on that field.
	pTitle := norm(p.Title)
	eTitle := norm(e.Title)
	if eTitle != "" && pTitle != eTitle {
		return true
	}

	pDesc := norm(p.Description)
	eDesc := norm(e.Description)
	if eDesc != "" && pDesc != eDesc {
		return true
	}

	pURL := norm(p.CourseURL)
	eURL := norm(e.CourseURL)
	if eURL != "" && pURL != eURL {
		return true
	}

	pLang := normLang(p.Language)
	eLang := normLang(e.Language)
	if eLang != "" && pLang != eLang {
		return true
	}

	pCat := norm(p.Category)
	eCat := norm(e.Category)
	if eCat != "" && pCat != eCat {
		return true
	}

	pDiff := norm(p.Difficulty)
	eDiff := norm(e.Difficulty)
	if eDiff != "" && pDiff != eDiff {
		return true
	}

	// Duration: tolerate small float formatting differences
	if e.DurationHours > 0 && p.DurationHours > 0 {
		if math.Abs(e.DurationHours-p.DurationHours) > 0.01 {
			return true
		}
	}

	// PublishedDate
	pPub := norm(p.PublishedDate)
	ePub := norm(e.PublishedDate)
	if ePub != "" && pPub != ePub {
		return true
	}

	// ImageURL
	pImg := norm(p.ImageURL)
	eImg := norm(e.ImageURL)
	if eImg != "" && pImg != eImg {
		return true
	}

	// Status: we usually keep Eightfold active; only update if Eightfold has a value.
	pStatus := norm(p.Status)
	eStatus := norm(e.Status)
	if eStatus != "" && pStatus != "" && pStatus != eStatus {
		return true
	}

	return false
}

func norm(s string) string {
	return strings.TrimSpace(strings.ToLower(s))
}

func normLang(s string) string {
	v := strings.TrimSpace(strings.ToLower(s))
	v = strings.ReplaceAll(v, "_", "-")
	switch v {
	case "english":
		return "en"
	case "spanish", "español", "espanol":
		return "es"
	case "portuguese", "português", "portugues":
		return "pt"
	}
	if strings.HasPrefix(v, "en") {
		return "en"
	}
	if strings.HasPrefix(v, "es") {
		return "es"
	}
	if strings.HasPrefix(v, "pt") {
		return "pt"
	}
	return v
}
