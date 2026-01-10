package udemy

import "encoding/json"

type Category struct {
	Title string `json:"title"`
	Name  string `json:"name"`
}

// Categories puede venir como:
// - "Development" (string)
// - {title,name} (obj)
// - ["Dev","IT"] (array de strings)
// - [{title,name}, ...] (array de objetos)
type Categories []Category

func (c *Categories) UnmarshalJSON(b []byte) error {
	if len(b) == 0 || string(b) == "null" {
		*c = nil
		return nil
	}

	// string: "Development"
	if b[0] == '"' {
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return err
		}
		if s == "" {
			*c = nil
			return nil
		}
		*c = Categories{{Title: s, Name: s}}
		return nil
	}

	// object: { ... }
	if b[0] == '{' {
		var one Category
		if err := json.Unmarshal(b, &one); err != nil {
			return err
		}
		*c = Categories{one}
		return nil
	}

	// array: [ ... ] (puede ser de objetos o de strings)
	if b[0] == '[' {
		var objs []Category
		if err := json.Unmarshal(b, &objs); err == nil {
			*c = objs
			return nil
		}

		var strs []string
		if err := json.Unmarshal(b, &strs); err != nil {
			return err
		}
		out := make(Categories, 0, len(strs))
		for _, s := range strs {
			if s == "" {
				continue
			}
			out = append(out, Category{Title: s, Name: s})
		}
		*c = out
		return nil
	}

	*c = nil
	return nil
}

// LocaleValue puede venir como:
// - "es_ES" (string)
// - { code: "es_ES" } (obj)
// - { locale: "es_ES" } (obj)
type LocaleValue string

func (l *LocaleValue) UnmarshalJSON(b []byte) error {
	if len(b) == 0 || string(b) == "null" {
		*l = ""
		return nil
	}

	// string: "es_ES"
	if b[0] == '"' {
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return err
		}
		*l = LocaleValue(s)
		return nil
	}

	// object: { ... }
	if b[0] == '{' {
		var m map[string]any
		if err := json.Unmarshal(b, &m); err != nil {
			return err
		}
		for _, k := range []string{"locale", "code", "name", "title", "id"} {
			if v, ok := m[k]; ok {
				if s, ok := v.(string); ok {
					*l = LocaleValue(s)
					return nil
				}
			}
		}
		*l = ""
		return nil
	}

	*l = ""
	return nil
}
