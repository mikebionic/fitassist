package mifit

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"
)

// DecodeSummary decodes the base64-encoded summary field from band_data.
func DecodeSummary(encoded string) (*DecodedSummary, error) {
	if encoded == "" {
		return nil, fmt.Errorf("empty summary")
	}

	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		// Try URL-safe base64
		data, err = base64.URLEncoding.DecodeString(encoded)
		if err != nil {
			return nil, fmt.Errorf("base64 decode: %w", err)
		}
	}

	var summary DecodedSummary
	if err := json.Unmarshal(data, &summary); err != nil {
		return nil, fmt.Errorf("JSON unmarshal summary: %w (data: %s)", err, truncate(string(data), 200))
	}

	return &summary, nil
}

// DecodeHeartRate decodes the base64-encoded binary heart rate data.
// The binary format is: each byte represents a heart rate reading.
// Readings are taken at regular intervals (typically 1 minute).
// A value of 0 or 255 means no measurement.
func DecodeHeartRate(encoded string, date time.Time) ([]ParsedHeartRatePoint, error) {
	if encoded == "" {
		return nil, nil
	}

	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		data, err = base64.URLEncoding.DecodeString(encoded)
		if err != nil {
			return nil, fmt.Errorf("base64 decode HR: %w", err)
		}
	}

	if len(data) == 0 {
		return nil, nil
	}

	// Each byte = 1 minute of HR data, starting from midnight
	var points []ParsedHeartRatePoint
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())

	for i, bpm := range data {
		if bpm == 0 || bpm == 255 {
			continue // no measurement
		}

		t := startOfDay.Add(time.Duration(i) * time.Minute)
		points = append(points, ParsedHeartRatePoint{
			Time: t,
			BPM:  int(bpm),
		})
	}

	return points, nil
}

// DecodeSpO2 decodes the base64-encoded SpO2 data.
// Similar binary format to heart rate — each byte is a percentage value.
func DecodeSpO2(encoded string, date time.Time) ([]struct {
	Time  time.Time
	Value int
}, error) {
	if encoded == "" {
		return nil, nil
	}

	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		data, err = base64.URLEncoding.DecodeString(encoded)
		if err != nil {
			return nil, fmt.Errorf("base64 decode SpO2: %w", err)
		}
	}

	if len(data) == 0 {
		return nil, nil
	}

	var points []struct {
		Time  time.Time
		Value int
	}
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())

	for i, val := range data {
		if val == 0 || val == 255 || val < 70 {
			continue // invalid or no measurement
		}

		t := startOfDay.Add(time.Duration(i) * time.Minute)
		points = append(points, struct {
			Time  time.Time
			Value int
		}{Time: t, Value: int(val)})
	}

	return points, nil
}

// DecodeStress decodes the base64-encoded stress data.
func DecodeStress(encoded string, date time.Time) ([]struct {
	Time  time.Time
	Value int
}, error) {
	if encoded == "" {
		return nil, nil
	}

	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		data, err = base64.URLEncoding.DecodeString(encoded)
		if err != nil {
			return nil, fmt.Errorf("base64 decode stress: %w", err)
		}
	}

	if len(data) == 0 {
		return nil, nil
	}

	var points []struct {
		Time  time.Time
		Value int
	}
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())

	for i, val := range data {
		if val == 0 || val == 255 {
			continue
		}

		t := startOfDay.Add(time.Duration(i) * time.Minute)
		points = append(points, struct {
			Time  time.Time
			Value int
		}{Time: t, Value: int(val)})
	}

	return points, nil
}

// SleepMinutesToTime converts minutes-from-midnight to a time.Time on the given date.
// Handles values > 1440 (next day) and values interpreted as previous-day sleep start.
func SleepMinutesToTime(minutes int, date time.Time) time.Time {
	baseDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())

	if minutes >= 1440 {
		// Overflows into next day
		return baseDay.Add(time.Duration(minutes) * time.Minute)
	}

	if minutes > 720 {
		// After noon — this is previous day's sleep start (e.g., 23:00 = 1380 min)
		return baseDay.Add(time.Duration(minutes-1440) * time.Minute)
	}

	// Morning — wake up time
	return baseDay.Add(time.Duration(minutes) * time.Minute)
}
