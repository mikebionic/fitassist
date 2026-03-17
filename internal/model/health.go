package model

import (
	"encoding/json"
	"time"
)

type HealthSteps struct {
	ID            int64            `db:"id" json:"id"`
	UserID        string           `db:"user_id" json:"user_id"`
	Date          time.Time        `db:"date" json:"date"`
	TotalSteps    *int             `db:"total_steps" json:"total_steps"`
	DistanceM     *int             `db:"distance_m" json:"distance_m"`
	Calories      *int             `db:"calories" json:"calories"`
	ActiveMinutes *int             `db:"active_minutes" json:"active_minutes"`
	Stages        *json.RawMessage `db:"stages" json:"stages,omitempty"`
}

type HealthSleep struct {
	ID          int64            `db:"id" json:"id"`
	UserID      string           `db:"user_id" json:"user_id"`
	Date        time.Time        `db:"date" json:"date"`
	SleepStart  *time.Time       `db:"sleep_start" json:"sleep_start,omitempty"`
	SleepEnd    *time.Time       `db:"sleep_end" json:"sleep_end,omitempty"`
	DurationMin *int             `db:"duration_min" json:"duration_min"`
	DeepMin     *int             `db:"deep_min" json:"deep_min"`
	LightMin    *int             `db:"light_min" json:"light_min"`
	REMMin      *int             `db:"rem_min" json:"rem_min"`
	AwakeMin    *int             `db:"awake_min" json:"awake_min"`
	Stages      *json.RawMessage `db:"stages" json:"stages,omitempty"`
}

type HealthHeartRate struct {
	ID         int64     `db:"id" json:"id"`
	UserID     string    `db:"user_id" json:"user_id"`
	MeasuredAt time.Time `db:"measured_at" json:"measured_at"`
	BPM        int16     `db:"bpm" json:"bpm"`
	Type       string    `db:"type" json:"type"`
}

type HealthSpO2 struct {
	ID         int64     `db:"id" json:"id"`
	UserID     string    `db:"user_id" json:"user_id"`
	MeasuredAt time.Time `db:"measured_at" json:"measured_at"`
	Value      int16     `db:"value" json:"value"`
}

type HealthWorkout struct {
	ID           int64            `db:"id" json:"id"`
	UserID       string           `db:"user_id" json:"user_id"`
	WorkoutType  string           `db:"workout_type" json:"workout_type"`
	StartedAt    time.Time        `db:"started_at" json:"started_at"`
	EndedAt      *time.Time       `db:"ended_at" json:"ended_at,omitempty"`
	DurationSec  *int             `db:"duration_sec" json:"duration_sec"`
	DistanceM    *int             `db:"distance_m" json:"distance_m"`
	Calories     *int             `db:"calories" json:"calories"`
	AvgHeartRate *int16           `db:"avg_heartrate" json:"avg_heartrate"`
	MaxHeartRate *int16           `db:"max_heartrate" json:"max_heartrate"`
	AvgPace      *float64         `db:"avg_pace" json:"avg_pace"`
	RouteData    *json.RawMessage `db:"route_data" json:"route_data,omitempty"`
	Details      *json.RawMessage `db:"details" json:"details,omitempty"`
}

type HealthStress struct {
	ID         int64     `db:"id" json:"id"`
	UserID     string    `db:"user_id" json:"user_id"`
	MeasuredAt time.Time `db:"measured_at" json:"measured_at"`
	Value      int16     `db:"value" json:"value"`
}
