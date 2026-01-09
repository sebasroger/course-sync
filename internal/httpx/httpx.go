package httpx

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func DoJSON(
	ctx context.Context,
	client *http.Client,
	req *http.Request,
	out any,
) error {
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("http error: %d %s", resp.StatusCode, string(body))
	}

	if out != nil {
		return json.Unmarshal(body, out)
	}
	return nil
}
