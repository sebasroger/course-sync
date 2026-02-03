package eightfold

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"course-sync/internal/httpx"
)

// Response shape #1 (common in Eightfold): {"data": [...], "meta": {...}}
type employeesResponseDataMeta struct {
	Data []map[string]any `json:"data"`
	Meta struct {
		PageStartIndex int `json:"pageStartIndex"`
		PageTotalCount int `json:"pageTotalCount"`
		TotalCount     int `json:"totalCount"`
	} `json:"meta"`
}

// Response shape #2: {"results": [...], "next": "..."}
type employeesResponseResultsNext struct {
	Results []map[string]any `json:"results"`
	Next    string           `json:"next"`
	Count   int              `json:"count"`
}

type employeesErrorResponse struct {
	Message string `json:"message"`
}

// ListAllEmployees fetches all employees from /api/v2/core/employees.
//
// Tenant behavior observed:
// - NO acepta pageStartIndex / pageSize (400 validating query parameters)
// - Primera página funciona SIN params y devuelve meta.totalCount/pageTotalCount
//
// Paginación implementada:
// 1) Primera llamada SIN params.
// 2) Si viene meta (totalCount/pageTotalCount), pagina con start+limit:
//   - start = pageTotalCount (offset)
//   - limit = pageTotalCount (cap a 100 por seguridad)
//
// También soporta results/next si el endpoint devuelve ese formato.
func (c *Client) ListAllEmployees(ctx context.Context, pageSizeHint int) ([]map[string]any, error) {
	if strings.TrimSpace(c.BearerToken) == "" {
		return nil, errors.New("eightfold: missing bearer token (set EIGHTFOLD_BEARER_TOKEN or call Authenticate)")
	}

	base, err := url.Parse(strings.TrimRight(c.BaseURL, "/") + "/api/v2/core/employees")
	if err != nil {
		return nil, fmt.Errorf("eightfold: invalid base url: %w", err)
	}

	// -------- First call: NO params --------
	body0, status0, err := c.getRaw(ctx, base.String())
	if err != nil {
		return nil, err
	}
	if status0 < 200 || status0 >= 300 {
		return nil, fmt.Errorf("list employees failed: url=%s status=%d body=%s", base.String(), status0, string(body0))
	}

	// Try shape #1 (data/meta)
	var dm0 employeesResponseDataMeta
	if err := json.Unmarshal(body0, &dm0); err == nil && dm0.Data != nil {
		all := make([]map[string]any, 0, max(dm0.Meta.TotalCount, len(dm0.Data)))
		all = append(all, dm0.Data...)

		// If meta doesn't give paging hints, return what we got.
		if dm0.Meta.TotalCount <= 0 || dm0.Meta.PageTotalCount <= 0 {
			return all, nil
		}

		total := dm0.Meta.TotalCount
		limit := dm0.Meta.PageTotalCount

		// Safety cap (Eightfold suele limitar a 100)
		if limit <= 0 {
			limit = 100
		}
		if limit > 100 {
			limit = 100
		}

		// If pageSizeHint is provided, keep it but cap to 100.
		if pageSizeHint > 0 {
			limit = pageSizeHint
			if limit > 100 {
				limit = 100
			}
		}

		// start is OFFSET, not page number
		start := len(dm0.Data)

		for start < total {
			u := *base
			q := u.Query()
			q.Set("start", strconv.Itoa(start))
			q.Set("limit", strconv.Itoa(limit))
			u.RawQuery = q.Encode()

			b, st, err := c.getRaw(ctx, u.String())
			if err != nil {
				return nil, err
			}
			if st < 200 || st >= 300 {
				return nil, fmt.Errorf("list employees failed: url=%s status=%d body=%s", u.String(), st, string(b))
			}

			var dm employeesResponseDataMeta
			if err := json.Unmarshal(b, &dm); err != nil {
				return nil, fmt.Errorf("list employees: json parse error: %w body=%s", err, string(b))
			}
			if dm.Data == nil {
				return nil, fmt.Errorf("list employees: unexpected response body=%s", string(b))
			}

			all = append(all, dm.Data...)

			// advance by actual received count (más robusto)
			got := len(dm.Data)
			if got == 0 {
				break
			}
			start += got
		}

		return all, nil
	}

	// Try shape #2 (results/next)
	var rn0 employeesResponseResultsNext
	if err := json.Unmarshal(body0, &rn0); err == nil && rn0.Results != nil {
		all := make([]map[string]any, 0)
		all = append(all, rn0.Results...)

		next := strings.TrimSpace(rn0.Next)
		for next != "" {
			b, st, err := c.getRaw(ctx, next)
			if err != nil {
				return nil, err
			}
			if st < 200 || st >= 300 {
				return nil, fmt.Errorf("list employees failed: url=%s status=%d body=%s", next, st, string(b))
			}

			var rn employeesResponseResultsNext
			if err := json.Unmarshal(b, &rn); err != nil {
				return nil, fmt.Errorf("list employees: json parse error: %w body=%s", err, string(b))
			}
			if rn.Results == nil {
				return nil, fmt.Errorf("list employees: unexpected response body=%s", string(b))
			}

			all = append(all, rn.Results...)
			next = strings.TrimSpace(rn.Next)
		}

		return all, nil
	}

	return nil, fmt.Errorf("list employees: unsupported response body=%s", string(body0))
}

func (c *Client) getRaw(ctx context.Context, urlStr string) ([]byte, int, error) {
	resp, body, err := httpx.DoWithRetry(
		ctx,
		c.HTTP,
		func(ctx context.Context) (*http.Request, error) {
			r, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
			if err != nil {
				return nil, err
			}
			r.Header.Set("Accept", "application/json")
			r.Header.Set("Authorization", "Bearer "+c.BearerToken)
			return r, nil
		},
		httpx.DefaultRetryConfig(),
	)
	if err != nil {
		return nil, 0, fmt.Errorf("eightfold: request failed: %w", err)
	}
	if resp == nil {
		return body, 0, nil
	}
	return body, resp.StatusCode, nil
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// ListEmployeesFields fetches all employees from /api/v2/core/employees and filters to only include specified fields.
// This is an optimized version of ListAllEmployees that only returns the fields you need.
func (c *Client) ListEmployeesFields(ctx context.Context, pageSizeHint int, fields []string) ([]map[string]any, error) {
	// Get all employees using the standard method
	allEmployees, err := c.ListAllEmployees(ctx, pageSizeHint)
	if err != nil {
		return nil, err
	}

	// If no fields specified, return all data
	if len(fields) == 0 {
		return allEmployees, nil
	}

	// Create a map for faster field lookup
	fieldMap := make(map[string]bool, len(fields))
	for _, field := range fields {
		fieldMap[field] = true
	}

	// Filter each employee to only include the specified fields
	result := make([]map[string]any, len(allEmployees))
	for i, employee := range allEmployees {
		filtered := make(map[string]any)

		// Only include fields that were requested
		for _, field := range fields {
			if value, exists := employee[field]; exists {
				filtered[field] = value
			}
		}

		result[i] = filtered
	}

	return result, nil
}
