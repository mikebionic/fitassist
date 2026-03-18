package mifit

import (
	"fmt"
	"time"
)

// BandDataResponse is the response from /v1/data/band_data.json.
type BandDataResponse struct {
	Code    int            `json:"code"`
	Message string         `json:"message"`
	Data    []BandDataItem `json:"data"`
}

// BandDataItem is a single day's data from band_data endpoint.
type BandDataItem struct {
	DateStr    string `json:"date_time"` // "2026-03-17"
	Summary    string `json:"summary"`   // Base64-encoded JSON
	DataHR     string `json:"data_hr"`   // Base64-encoded binary heart rate data
	DataSleep  string `json:"data_sleep"`
	DataStress string `json:"data_stress"`
	DataSpO2   string `json:"data_spo2"`
}

// DecodedSummary is the decoded structure from the Base64 summary field.
type DecodedSummary struct {
	Goal int `json:"goal"`
	TZ   int `json:"tz"` // timezone offset in minutes

	// Steps
	Steps struct {
		Total    int `json:"ttl"`
		Distance int `json:"dis"` // meters
		Calories int `json:"cal"`
		Stages   []struct {
			Start int `json:"start"` // minutes from midnight
			Stop  int `json:"stop"`
			Steps int `json:"step"`
			Mode  int `json:"mode"` // 1=walk, 2=run, etc
		} `json:"stage"`
	} `json:"stp"`

	// Sleep
	Sleep struct {
		Start  int `json:"st"`  // minutes from midnight (e.g., 1380 = 23:00)
		End    int `json:"ed"`  // minutes from midnight (e.g., 480 = 08:00)
		Deep   int `json:"dp"`  // deep sleep minutes
		Light  int `json:"lt"`  // light sleep minutes
		REM    int `json:"rem"` // REM minutes (if available)
		Awake  int `json:"wk"`  // awake minutes
		Stages []struct {
			Start int `json:"start"`
			Stop  int `json:"stop"`
			Mode  int `json:"mode"` // 4=light, 5=deep, 6=REM, 7=awake
		} `json:"stage"`
	} `json:"slp"`
}

// WorkoutHistoryResponse is the response from /v1/sport/run/history.json.
type WorkoutHistoryResponse struct {
	Code    int              `json:"code"`
	Message string           `json:"message"`
	Data    WorkoutHistoryData `json:"data"`
}

type WorkoutHistoryData struct {
	Summary []WorkoutSummary `json:"summary"`
}

// WorkoutSummary is a single workout entry from history.
type WorkoutSummary struct {
	TrackID   int64  `json:"trackid"`
	Type      int    `json:"type"`       // workout type code
	StartTime int64  `json:"start_time"` // unix timestamp (seconds)
	EndTime   int64  `json:"end_time"`
	Distance  int    `json:"dis"`        // meters
	Calories  int    `json:"cal"`
	AvgHR     int    `json:"avg_heart_rate"`
	MaxHR     int    `json:"max_heart_rate"`
	AvgPace   int    `json:"avg_pace"`   // seconds per km
}

// WorkoutDetailResponse is the response from /v1/sport/run/detail.json.
type WorkoutDetailResponse struct {
	Code    int           `json:"code"`
	Message string        `json:"message"`
	Data    WorkoutDetail `json:"data"`
}

// WorkoutDetail contains detailed workout data.
type WorkoutDetail struct {
	TrackID   int64   `json:"trackid"`
	Source    string  `json:"source"`
	Distance  int     `json:"dis"`
	Calories  int     `json:"cal"`
	StartTime int64   `json:"start_time"`
	EndTime   int64   `json:"end_time"`
	AvgHR     int     `json:"avg_heart_rate"`
	MaxHR     int     `json:"max_heart_rate"`
	AvgPace   int     `json:"avg_pace"`
	GPXData   string  `json:"longitude_latitude"` // GPS track data
	HRData    string  `json:"heart_rate"`          // heart rate during workout
}

// ParsedHeartRatePoint represents a single heart rate measurement.
type ParsedHeartRatePoint struct {
	Time time.Time
	BPM  int
}

// WorkoutTypeNames maps workout type codes to human-readable names.
var WorkoutTypeNames = map[int]string{
	1:   "running",
	6:   "walking",
	9:   "treadmill",
	10:  "cycling",
	16:  "yoga",
	17:  "elliptical",
	25:  "strength",
	27:  "hiking",
	29:  "swimming_pool",
	31:  "rowing",
	32:  "jump_rope",
	33:  "free_training",
	48:  "dance",
	74:  "stretching",
	80:  "outdoor_cycling",
	258: "tennis",
}

func WorkoutTypeName(code int) string {
	if name, ok := WorkoutTypeNames[code]; ok {
		return name
	}
	return fmt.Sprintf("unknown_%d", code)
}
