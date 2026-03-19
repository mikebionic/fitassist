package mifit

import (
	"errors"
	"fmt"
)

// GetWorkoutHistory fetches the list of all workouts.
func (c *Client) GetWorkoutHistory() (*WorkoutHistoryResponse, error) {
	path := "/v1/sport/run/history.json"
	params := map[string]string{
		"source": "run.band",
		"userid": c.userIDMi,
	}

	body, err := c.doDataRequest("GET", path, params)
	if errors.Is(err, ErrNoData) {
		return &WorkoutHistoryResponse{Code: 1, Message: "ok"}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("fetching workout history: %w", err)
	}

	var resp WorkoutHistoryResponse
	if err := parseJSON(body, &resp); err != nil {
		return nil, err
	}

	if resp.Code != 1 {
		return nil, fmt.Errorf("workout history API error: code=%d message=%s", resp.Code, resp.Message)
	}

	return &resp, nil
}

// GetWorkoutDetail fetches detailed data for a specific workout.
func (c *Client) GetWorkoutDetail(trackID int64) (*WorkoutDetailResponse, error) {
	path := "/v1/sport/run/detail.json"
	params := map[string]string{
		"source":  "run.band",
		"userid":  c.userIDMi,
		"trackid": fmt.Sprintf("%d", trackID),
	}

	body, err := c.doDataRequest("GET", path, params)
	if errors.Is(err, ErrNoData) {
		return &WorkoutDetailResponse{Code: 1, Message: "ok"}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("fetching workout detail: %w", err)
	}

	var resp WorkoutDetailResponse
	if err := parseJSON(body, &resp); err != nil {
		return nil, err
	}

	if resp.Code != 1 {
		return nil, fmt.Errorf("workout detail API error: code=%d message=%s", resp.Code, resp.Message)
	}

	return &resp, nil
}
