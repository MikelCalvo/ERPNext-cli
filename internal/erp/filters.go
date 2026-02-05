package erp

import (
	"encoding/json"
	"fmt"
	"net/url"
)

// encodeFilters builds a safe, URL-escaped filters string for ERPNext APIs.
func encodeFilters(filters [][]interface{}) (string, error) {
	if len(filters) == 0 {
		return "", nil
	}

	encoded, err := json.Marshal(filters)
	if err != nil {
		return "", fmt.Errorf("failed to encode filters: %w", err)
	}

	return url.QueryEscape(string(encoded)), nil
}
