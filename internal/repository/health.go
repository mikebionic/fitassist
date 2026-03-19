package repository

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/mike/fitassist/internal/model"
)

type HealthRepository struct {
	db *sqlx.DB
}

func NewHealthRepository(db *sqlx.DB) *HealthRepository {
	return &HealthRepository{db: db}
}

// Steps

func (r *HealthRepository) UpsertSteps(ctx context.Context, s *model.HealthSteps) error {
	query := `
		INSERT INTO health_steps (user_id, date, total_steps, distance_m, calories, active_minutes, stages)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (user_id, date) DO UPDATE SET
			total_steps = EXCLUDED.total_steps,
			distance_m = EXCLUDED.distance_m,
			calories = EXCLUDED.calories,
			active_minutes = EXCLUDED.active_minutes,
			stages = EXCLUDED.stages
		RETURNING id`
	return r.db.QueryRowxContext(ctx, query,
		s.UserID, s.Date, s.TotalSteps, s.DistanceM, s.Calories, s.ActiveMinutes, s.Stages,
	).Scan(&s.ID)
}

func (r *HealthRepository) GetSteps(ctx context.Context, userID string, from, to time.Time) ([]model.HealthSteps, error) {
	var steps []model.HealthSteps
	err := r.db.SelectContext(ctx, &steps,
		`SELECT * FROM health_steps WHERE user_id = $1 AND date >= $2 AND date <= $3 ORDER BY date`,
		userID, from, to)
	return steps, err
}

// Sleep

func (r *HealthRepository) UpsertSleep(ctx context.Context, s *model.HealthSleep) error {
	query := `
		INSERT INTO health_sleep (user_id, date, sleep_start, sleep_end, duration_min, deep_min, light_min, rem_min, awake_min, stages)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (user_id, date) DO UPDATE SET
			sleep_start = EXCLUDED.sleep_start,
			sleep_end = EXCLUDED.sleep_end,
			duration_min = EXCLUDED.duration_min,
			deep_min = EXCLUDED.deep_min,
			light_min = EXCLUDED.light_min,
			rem_min = EXCLUDED.rem_min,
			awake_min = EXCLUDED.awake_min,
			stages = EXCLUDED.stages
		RETURNING id`
	return r.db.QueryRowxContext(ctx, query,
		s.UserID, s.Date, s.SleepStart, s.SleepEnd, s.DurationMin, s.DeepMin, s.LightMin, s.REMMin, s.AwakeMin, s.Stages,
	).Scan(&s.ID)
}

func (r *HealthRepository) GetSleep(ctx context.Context, userID string, from, to time.Time) ([]model.HealthSleep, error) {
	var sleep []model.HealthSleep
	err := r.db.SelectContext(ctx, &sleep,
		`SELECT * FROM health_sleep WHERE user_id = $1 AND date >= $2 AND date <= $3 ORDER BY date`,
		userID, from, to)
	return sleep, err
}

// Heart Rate

func (r *HealthRepository) BatchInsertHeartRate(ctx context.Context, records []model.HealthHeartRate) (int, error) {
	if len(records) == 0 {
		return 0, nil
	}

	query := `
		INSERT INTO health_heartrate (user_id, measured_at, bpm, type)
		VALUES (:user_id, :measured_at, :bpm, :type)
		ON CONFLICT (user_id, measured_at) DO NOTHING`

	result, err := r.db.NamedExecContext(ctx, query, records)
	if err != nil {
		return 0, err
	}
	rows, _ := result.RowsAffected()
	return int(rows), nil
}

func (r *HealthRepository) GetHeartRate(ctx context.Context, userID string, from, to time.Time) ([]model.HealthHeartRate, error) {
	var hr []model.HealthHeartRate
	err := r.db.SelectContext(ctx, &hr,
		`SELECT * FROM health_heartrate WHERE user_id = $1 AND measured_at >= $2 AND measured_at <= $3 ORDER BY measured_at`,
		userID, from, to)
	return hr, err
}

func (r *HealthRepository) GetLatestHeartRate(ctx context.Context, userID string) (*model.HealthHeartRate, error) {
	var hr model.HealthHeartRate
	err := r.db.GetContext(ctx, &hr,
		`SELECT * FROM health_heartrate WHERE user_id = $1 ORDER BY measured_at DESC LIMIT 1`,
		userID)
	if err != nil {
		return nil, err
	}
	return &hr, nil
}

// SpO2

func (r *HealthRepository) BatchInsertSpO2(ctx context.Context, records []model.HealthSpO2) (int, error) {
	if len(records) == 0 {
		return 0, nil
	}

	query := `
		INSERT INTO health_spo2 (user_id, measured_at, value)
		VALUES (:user_id, :measured_at, :value)
		ON CONFLICT (user_id, measured_at) DO NOTHING`

	result, err := r.db.NamedExecContext(ctx, query, records)
	if err != nil {
		return 0, err
	}
	rows, _ := result.RowsAffected()
	return int(rows), nil
}

func (r *HealthRepository) GetSpO2(ctx context.Context, userID string, from, to time.Time) ([]model.HealthSpO2, error) {
	var spo2 []model.HealthSpO2
	err := r.db.SelectContext(ctx, &spo2,
		`SELECT * FROM health_spo2 WHERE user_id = $1 AND measured_at >= $2 AND measured_at <= $3 ORDER BY measured_at`,
		userID, from, to)
	return spo2, err
}

// Workouts

func (r *HealthRepository) UpsertWorkout(ctx context.Context, w *model.HealthWorkout) error {
	query := `
		INSERT INTO health_workouts (user_id, workout_type, started_at, ended_at, duration_sec, distance_m, calories, avg_heartrate, max_heartrate, avg_pace, route_data, details)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (user_id, started_at) DO UPDATE SET
			workout_type = EXCLUDED.workout_type,
			ended_at = EXCLUDED.ended_at,
			duration_sec = EXCLUDED.duration_sec,
			distance_m = EXCLUDED.distance_m,
			calories = EXCLUDED.calories,
			avg_heartrate = EXCLUDED.avg_heartrate,
			max_heartrate = EXCLUDED.max_heartrate,
			avg_pace = EXCLUDED.avg_pace,
			route_data = EXCLUDED.route_data,
			details = EXCLUDED.details
		RETURNING id`
	return r.db.QueryRowxContext(ctx, query,
		w.UserID, w.WorkoutType, w.StartedAt, w.EndedAt, w.DurationSec, w.DistanceM,
		w.Calories, w.AvgHeartRate, w.MaxHeartRate, w.AvgPace, w.RouteData, w.Details,
	).Scan(&w.ID)
}

func (r *HealthRepository) GetWorkouts(ctx context.Context, userID string, from, to time.Time) ([]model.HealthWorkout, error) {
	var workouts []model.HealthWorkout
	err := r.db.SelectContext(ctx, &workouts,
		`SELECT * FROM health_workouts WHERE user_id = $1 AND started_at >= $2 AND started_at <= $3 ORDER BY started_at DESC`,
		userID, from, to)
	return workouts, err
}

func (r *HealthRepository) GetWorkoutByID(ctx context.Context, userID string, id int64) (*model.HealthWorkout, error) {
	var w model.HealthWorkout
	err := r.db.GetContext(ctx, &w,
		`SELECT * FROM health_workouts WHERE user_id = $1 AND id = $2`,
		userID, id)
	if err != nil {
		return nil, err
	}
	return &w, nil
}

// Stress

func (r *HealthRepository) BatchInsertStress(ctx context.Context, records []model.HealthStress) (int, error) {
	if len(records) == 0 {
		return 0, nil
	}

	query := `
		INSERT INTO health_stress (user_id, measured_at, value)
		VALUES (:user_id, :measured_at, :value)
		ON CONFLICT (user_id, measured_at) DO NOTHING`

	result, err := r.db.NamedExecContext(ctx, query, records)
	if err != nil {
		return 0, err
	}
	rows, _ := result.RowsAffected()
	return int(rows), nil
}

func (r *HealthRepository) GetStress(ctx context.Context, userID string, from, to time.Time) ([]model.HealthStress, error) {
	var stress []model.HealthStress
	err := r.db.SelectContext(ctx, &stress,
		`SELECT * FROM health_stress WHERE user_id = $1 AND measured_at >= $2 AND measured_at <= $3 ORDER BY measured_at`,
		userID, from, to)
	return stress, err
}

// GetLatestWorkout returns the most recent workout for a user.
func (r *HealthRepository) GetLatestWorkout(ctx context.Context, userID string) (*model.HealthWorkout, error) {
	var w model.HealthWorkout
	err := r.db.GetContext(ctx, &w,
		`SELECT * FROM health_workouts WHERE user_id = $1 ORDER BY started_at DESC LIMIT 1`,
		userID)
	if err != nil {
		return nil, err
	}
	return &w, nil
}

// GetLatestSleep returns the most recent sleep record for a user.
func (r *HealthRepository) GetLatestSleep(ctx context.Context, userID string) (*model.HealthSleep, error) {
	var s model.HealthSleep
	err := r.db.GetContext(ctx, &s,
		`SELECT * FROM health_sleep WHERE user_id = $1 ORDER BY date DESC LIMIT 1`,
		userID)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// Dashboard summary

type DashboardSummary struct {
	StepsToday    *int     `json:"steps_today"`
	CaloriesToday *int     `json:"calories_today"`
	DistanceToday *int     `json:"distance_today"`
	SleepLastNight *int    `json:"sleep_last_night_min"`
	AvgHRToday    *float64 `json:"avg_hr_today"`
	LastHR        *int16   `json:"last_hr"`
}

func (r *HealthRepository) GetDashboardSummary(ctx context.Context, userID string, today time.Time) (*DashboardSummary, error) {
	summary := &DashboardSummary{}

	// Steps today
	_ = r.db.GetContext(ctx, &summary.StepsToday,
		`SELECT total_steps FROM health_steps WHERE user_id = $1 AND date = $2`, userID, today)
	_ = r.db.GetContext(ctx, &summary.CaloriesToday,
		`SELECT calories FROM health_steps WHERE user_id = $1 AND date = $2`, userID, today)
	_ = r.db.GetContext(ctx, &summary.DistanceToday,
		`SELECT distance_m FROM health_steps WHERE user_id = $1 AND date = $2`, userID, today)

	// Sleep last night
	_ = r.db.GetContext(ctx, &summary.SleepLastNight,
		`SELECT duration_min FROM health_sleep WHERE user_id = $1 AND date = $2`, userID, today)

	// Average HR today
	_ = r.db.GetContext(ctx, &summary.AvgHRToday,
		`SELECT AVG(bpm)::float FROM health_heartrate WHERE user_id = $1 AND measured_at >= $2 AND measured_at < $3`,
		userID, today, today.AddDate(0, 0, 1))

	// Last HR
	_ = r.db.GetContext(ctx, &summary.LastHR,
		`SELECT bpm FROM health_heartrate WHERE user_id = $1 ORDER BY measured_at DESC LIMIT 1`, userID)

	return summary, nil
}
