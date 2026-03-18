package mifit

import (
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"
)

func TestDecodeSummary(t *testing.T) {
	summary := DecodedSummary{}
	summary.Steps.Total = 8500
	summary.Steps.Distance = 6200
	summary.Steps.Calories = 320
	summary.Sleep.Start = 1380 // 23:00
	summary.Sleep.End = 480    // 08:00
	summary.Sleep.Deep = 90
	summary.Sleep.Light = 180
	summary.Sleep.REM = 60
	summary.Sleep.Awake = 20

	data, _ := json.Marshal(summary)
	encoded := base64.StdEncoding.EncodeToString(data)

	decoded, err := DecodeSummary(encoded)
	if err != nil {
		t.Fatalf("DecodeSummary: %v", err)
	}

	if decoded.Steps.Total != 8500 {
		t.Errorf("steps total: got %d, want 8500", decoded.Steps.Total)
	}
	if decoded.Steps.Distance != 6200 {
		t.Errorf("distance: got %d, want 6200", decoded.Steps.Distance)
	}
	if decoded.Sleep.Deep != 90 {
		t.Errorf("deep sleep: got %d, want 90", decoded.Sleep.Deep)
	}
	if decoded.Sleep.REM != 60 {
		t.Errorf("REM: got %d, want 60", decoded.Sleep.REM)
	}
}

func TestDecodeSummaryEmpty(t *testing.T) {
	_, err := DecodeSummary("")
	if err == nil {
		t.Error("expected error on empty summary")
	}
}

func TestDecodeSummaryInvalidBase64(t *testing.T) {
	_, err := DecodeSummary("!!!not-base64!!!")
	if err == nil {
		t.Error("expected error on invalid base64")
	}
}

func TestDecodeHeartRate(t *testing.T) {
	// Simulate 5 bytes of HR data: [0, 72, 75, 255, 80]
	data := []byte{0, 72, 75, 255, 80}
	encoded := base64.StdEncoding.EncodeToString(data)
	date := time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)

	points, err := DecodeHeartRate(encoded, date)
	if err != nil {
		t.Fatalf("DecodeHeartRate: %v", err)
	}

	// 0 and 255 are skipped, so we expect 3 points
	if len(points) != 3 {
		t.Fatalf("expected 3 points, got %d", len(points))
	}

	if points[0].BPM != 72 {
		t.Errorf("point 0 BPM: got %d, want 72", points[0].BPM)
	}
	// Minute 1 of the day
	expected := time.Date(2026, 3, 15, 0, 1, 0, 0, time.UTC)
	if !points[0].Time.Equal(expected) {
		t.Errorf("point 0 time: got %v, want %v", points[0].Time, expected)
	}

	if points[1].BPM != 75 {
		t.Errorf("point 1 BPM: got %d, want 75", points[1].BPM)
	}

	if points[2].BPM != 80 {
		t.Errorf("point 2 BPM: got %d, want 80", points[2].BPM)
	}
	// Minute 4 of the day
	expected4 := time.Date(2026, 3, 15, 0, 4, 0, 0, time.UTC)
	if !points[2].Time.Equal(expected4) {
		t.Errorf("point 2 time: got %v, want %v", points[2].Time, expected4)
	}
}

func TestDecodeHeartRateEmpty(t *testing.T) {
	date := time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)
	points, err := DecodeHeartRate("", date)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if points != nil {
		t.Error("expected nil for empty input")
	}
}

func TestDecodeSpO2(t *testing.T) {
	// Values: [0, 98, 97, 69, 96] — 69 is below threshold (70), skipped
	data := []byte{0, 98, 97, 69, 96}
	encoded := base64.StdEncoding.EncodeToString(data)
	date := time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)

	points, err := DecodeSpO2(encoded, date)
	if err != nil {
		t.Fatalf("DecodeSpO2: %v", err)
	}

	if len(points) != 3 {
		t.Fatalf("expected 3 points, got %d", len(points))
	}

	if points[0].Value != 98 {
		t.Errorf("point 0 value: got %d, want 98", points[0].Value)
	}
	if points[1].Value != 97 {
		t.Errorf("point 1 value: got %d, want 97", points[1].Value)
	}
	if points[2].Value != 96 {
		t.Errorf("point 2 value: got %d, want 96", points[2].Value)
	}
}

func TestDecodeStress(t *testing.T) {
	data := []byte{0, 42, 255, 55}
	encoded := base64.StdEncoding.EncodeToString(data)
	date := time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)

	points, err := DecodeStress(encoded, date)
	if err != nil {
		t.Fatalf("DecodeStress: %v", err)
	}

	if len(points) != 2 {
		t.Fatalf("expected 2 points, got %d", len(points))
	}
	if points[0].Value != 42 {
		t.Errorf("got %d, want 42", points[0].Value)
	}
	if points[1].Value != 55 {
		t.Errorf("got %d, want 55", points[1].Value)
	}
}

func TestSleepMinutesToTime(t *testing.T) {
	date := time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		minutes  int
		expected time.Time
	}{
		{
			name:     "morning wakeup 8:00",
			minutes:  480,
			expected: time.Date(2026, 3, 15, 8, 0, 0, 0, time.UTC),
		},
		{
			name:     "evening bedtime 23:00 (1380 min)",
			minutes:  1380,
			expected: time.Date(2026, 3, 14, 23, 0, 0, 0, time.UTC), // previous day
		},
		{
			name:     "overflow next day 1500 min",
			minutes:  1500,
			expected: time.Date(2026, 3, 16, 1, 0, 0, 0, time.UTC), // next day 1:00
		},
		{
			name:     "midnight",
			minutes:  0,
			expected: time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := SleepMinutesToTime(tc.minutes, date)
			if !got.Equal(tc.expected) {
				t.Errorf("SleepMinutesToTime(%d) = %v, want %v", tc.minutes, got, tc.expected)
			}
		})
	}
}

func TestGenerateDateList(t *testing.T) {
	from := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 3, 13, 0, 0, 0, 0, time.UTC)

	dates := GenerateDateList(from, to)
	expected := []string{"2026-03-10", "2026-03-11", "2026-03-12", "2026-03-13"}

	if len(dates) != len(expected) {
		t.Fatalf("expected %d dates, got %d", len(expected), len(dates))
	}

	for i, d := range dates {
		if d != expected[i] {
			t.Errorf("date[%d] = %s, want %s", i, d, expected[i])
		}
	}
}

func TestGenerateDateListSameDay(t *testing.T) {
	day := time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)
	dates := GenerateDateList(day, day)
	if len(dates) != 1 {
		t.Fatalf("expected 1 date, got %d", len(dates))
	}
	if dates[0] != "2026-03-15" {
		t.Errorf("got %s, want 2026-03-15", dates[0])
	}
}

func TestWorkoutTypeName(t *testing.T) {
	if name := WorkoutTypeName(1); name != "running" {
		t.Errorf("type 1: got %q, want %q", name, "running")
	}
	if name := WorkoutTypeName(6); name != "walking" {
		t.Errorf("type 6: got %q, want %q", name, "walking")
	}
	if name := WorkoutTypeName(9999); name != "unknown_9999" {
		t.Errorf("type 9999: got %q, want %q", name, "unknown_9999")
	}
}
