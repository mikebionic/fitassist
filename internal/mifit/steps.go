package mifit

import (
	"fmt"
	"net/url"
	"strings"
	"time"
)

// GetBandData fetches band data (steps, sleep, HR) for the given date range.
// dates should be in "YYYY-MM-DD" format.
func (c *Client) GetBandData(dates []string) (*BandDataResponse, error) {
	params := url.Values{
		"query_type": {"summary"},
		"device_type": {"0"},
		"userid":     {c.userIDMi},
		"date_list":  {strings.Join(dates, ",")},
	}

	body, err := c.doRequest("GET", "/v1/data/band_data.json", params)
	if err != nil {
		return nil, fmt.Errorf("fetching band data: %w", err)
	}

	var resp BandDataResponse
	if err := parseJSON(body, &resp); err != nil {
		return nil, err
	}

	if resp.Code != 1 {
		return nil, fmt.Errorf("band data API error: code=%d message=%s", resp.Code, resp.Message)
	}

	return &resp, nil
}

// GetBandDataDetail fetches detailed band data (including HR binary data).
func (c *Client) GetBandDataDetail(dates []string) (*BandDataResponse, error) {
	params := url.Values{
		"query_type": {"detail"},
		"device_type": {"0"},
		"userid":     {c.userIDMi},
		"date_list":  {strings.Join(dates, ",")},
	}

	body, err := c.doRequest("GET", "/v1/data/band_data.json", params)
	if err != nil {
		return nil, fmt.Errorf("fetching band data detail: %w", err)
	}

	var resp BandDataResponse
	if err := parseJSON(body, &resp); err != nil {
		return nil, err
	}

	if resp.Code != 1 {
		return nil, fmt.Errorf("band data detail API error: code=%d message=%s", resp.Code, resp.Message)
	}

	return &resp, nil
}

// GenerateDateList creates a list of date strings between from and to (inclusive).
func GenerateDateList(from, to time.Time) []string {
	var dates []string
	current := from
	for !current.After(to) {
		dates = append(dates, current.Format("2006-01-02"))
		current = current.AddDate(0, 0, 1)
	}
	return dates
}
