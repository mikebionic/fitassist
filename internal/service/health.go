package service

import (
	"context"
	"time"

	"github.com/mike/fitassist/internal/model"
	"github.com/mike/fitassist/internal/repository"
)

type HealthService struct {
	repo *repository.HealthRepository
}

func NewHealthService(repo *repository.HealthRepository) *HealthService {
	return &HealthService{repo: repo}
}

func (s *HealthService) GetDashboard(ctx context.Context, userID string) (*repository.DashboardSummary, error) {
	today := time.Now().Truncate(24 * time.Hour)
	return s.repo.GetDashboardSummary(ctx, userID, today)
}

func (s *HealthService) GetSteps(ctx context.Context, userID string, from, to time.Time) ([]model.HealthSteps, error) {
	return s.repo.GetSteps(ctx, userID, from, to)
}

func (s *HealthService) GetSleep(ctx context.Context, userID string, from, to time.Time) ([]model.HealthSleep, error) {
	return s.repo.GetSleep(ctx, userID, from, to)
}

func (s *HealthService) GetHeartRate(ctx context.Context, userID string, from, to time.Time) ([]model.HealthHeartRate, error) {
	return s.repo.GetHeartRate(ctx, userID, from, to)
}

func (s *HealthService) GetSpO2(ctx context.Context, userID string, from, to time.Time) ([]model.HealthSpO2, error) {
	return s.repo.GetSpO2(ctx, userID, from, to)
}

func (s *HealthService) GetWorkouts(ctx context.Context, userID string, from, to time.Time) ([]model.HealthWorkout, error) {
	return s.repo.GetWorkouts(ctx, userID, from, to)
}

func (s *HealthService) GetWorkoutByID(ctx context.Context, userID string, id int64) (*model.HealthWorkout, error) {
	return s.repo.GetWorkoutByID(ctx, userID, id)
}

func (s *HealthService) GetStress(ctx context.Context, userID string, from, to time.Time) ([]model.HealthStress, error) {
	return s.repo.GetStress(ctx, userID, from, to)
}
