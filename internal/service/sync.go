package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/mike/fitassist/internal/crypto"
	"github.com/mike/fitassist/internal/mifit"
	"github.com/mike/fitassist/internal/model"
	"github.com/mike/fitassist/internal/repository"
)

type SyncService struct {
	healthRepo   *repository.HealthRepository
	mifitRepo    *repository.MiFitRepository
	syncLogRepo  *repository.SyncLogRepository
	apiBaseURL   string
	encKey       string
	postSyncHook func(ctx context.Context, userID string)
}

// SetPostSyncHook sets a callback that runs after each successful account sync.
func (s *SyncService) SetPostSyncHook(fn func(ctx context.Context, userID string)) {
	s.postSyncHook = fn
}

func NewSyncService(
	healthRepo *repository.HealthRepository,
	mifitRepo *repository.MiFitRepository,
	syncLogRepo *repository.SyncLogRepository,
	apiBaseURL, encryptionKey string,
) *SyncService {
	return &SyncService{
		healthRepo:  healthRepo,
		mifitRepo:   mifitRepo,
		syncLogRepo: syncLogRepo,
		apiBaseURL:  apiBaseURL,
		encKey:      encryptionKey,
	}
}

// SyncAll syncs data for all enabled Mi Fitness accounts.
func (s *SyncService) SyncAll(ctx context.Context) {
	accounts, err := s.mifitRepo.ListSyncEnabled(ctx)
	if err != nil {
		slog.Error("listing sync-enabled accounts", "error", err)
		return
	}

	for _, acc := range accounts {
		if err := s.SyncAccount(ctx, &acc); err != nil {
			slog.Error("syncing account", "user_id", acc.UserID, "error", err)
		}
	}
}

// SyncAccount syncs data for a single Mi Fitness account.
func (s *SyncService) SyncAccount(ctx context.Context, acc *model.MiFitAccount) error {
	syncType := "incremental"
	status := "success"

	log := &model.SyncLog{
		UserID:   acc.UserID,
		SyncType: &syncType,
		Status:   &status,
	}
	if err := s.syncLogRepo.Create(ctx, log); err != nil {
		return fmt.Errorf("creating sync log: %w", err)
	}

	totalRecords := 0
	var syncErr error

	defer func() {
		st := "success"
		var errMsg *string
		if syncErr != nil {
			st = "error"
			e := syncErr.Error()
			errMsg = &e
		}
		_ = s.syncLogRepo.Finish(ctx, log.ID, st, totalRecords, errMsg)
	}()

	// Create Mi Fitness API client
	client := mifit.NewClient(s.apiBaseURL)

	// If we have a stored token, use it
	if acc.AuthToken != nil && *acc.AuthToken != "" {
		client.SetAuth(*acc.AuthToken, ptrVal(acc.UserIDMi))

		// Restore Xiaomi auth credentials if this is a Xiaomi account
		if acc.AuthMethod == "xiaomi" && len(acc.XiaomiAuthData) > 0 {
			xiaomiAuth, err := s.decryptXiaomiAuth(acc.XiaomiAuthData)
			if err != nil {
				slog.Warn("failed to restore Xiaomi auth, will re-authenticate", "error", err)
			} else {
				client.SetXiaomiAuth(xiaomiAuth)
			}
		}
	} else {
		// Need to re-authenticate
		n, err := s.reAuth(ctx, acc, client)
		if err != nil {
			syncErr = fmt.Errorf("re-authentication: %w", err)
			return syncErr
		}
		totalRecords += n
	}

	// Determine date range for sync
	from := time.Now().AddDate(0, 0, -7) // default: last 7 days
	if acc.LastSync != nil {
		from = *acc.LastSync
	}
	to := time.Now()
	dates := mifit.GenerateDateList(from, to)

	if len(dates) == 0 {
		return nil
	}

	// Sync band data (steps, sleep, HR) in batches of 10 days
	for i := 0; i < len(dates); i += 10 {
		end := i + 10
		if end > len(dates) {
			end = len(dates)
		}
		batch := dates[i:end]

		n, err := s.syncBandData(ctx, client, acc.UserID, batch, from)
		if err != nil {
			slog.Warn("band data sync error", "user_id", acc.UserID, "error", err)
			syncErr = err
			continue
		}
		totalRecords += n
	}

	// Sync workouts
	n, err := s.syncWorkouts(ctx, client, acc.UserID)
	if err != nil {
		slog.Warn("workout sync error", "user_id", acc.UserID, "error", err)
		if syncErr == nil {
			syncErr = err
		}
	} else {
		totalRecords += n
	}

	// Update last sync time
	_ = s.mifitRepo.UpdateLastSync(ctx, acc.ID)

	slog.Info("sync completed", "user_id", acc.UserID, "records", totalRecords)

	if s.postSyncHook != nil {
		go s.postSyncHook(context.Background(), acc.UserID)
	}

	return syncErr
}

// reAuth re-authenticates the account by decrypting stored credentials.
func (s *SyncService) reAuth(ctx context.Context, acc *model.MiFitAccount, client *mifit.Client) (int, error) {
	if s.encKey == "" {
		return 0, fmt.Errorf("encryption key not configured")
	}

	password, err := crypto.Decrypt(acc.MiPassword, s.encKey)
	if err != nil {
		return 0, fmt.Errorf("decrypting password: %w", err)
	}

	result, err := client.Login(acc.MiEmail, string(password))
	if err != nil {
		return 0, fmt.Errorf("Mi Fitness login: %w", err)
	}

	// Store the new token
	if err := s.mifitRepo.UpdateToken(ctx, acc.ID, result.AppToken, result.UserIDMi); err != nil {
		return 0, fmt.Errorf("storing token: %w", err)
	}

	// Store Xiaomi auth credentials if applicable
	if result.AuthMethod == "xiaomi" && result.XiaomiAuth != nil {
		xiaomiJSON, _ := json.Marshal(result.XiaomiAuth)
		encXiaomiAuth, err := crypto.Encrypt(xiaomiJSON, s.encKey)
		if err == nil {
			_ = s.mifitRepo.UpdateAuthMethod(ctx, acc.ID, "xiaomi", encXiaomiAuth)
		}
	}

	return 0, nil
}

// decryptXiaomiAuth decrypts stored Xiaomi auth credentials.
func (s *SyncService) decryptXiaomiAuth(encData []byte) (*mifit.XiaomiAuth, error) {
	decrypted, err := crypto.Decrypt(encData, s.encKey)
	if err != nil {
		return nil, fmt.Errorf("decrypting xiaomi auth: %w", err)
	}
	var auth mifit.XiaomiAuth
	if err := json.Unmarshal(decrypted, &auth); err != nil {
		return nil, fmt.Errorf("parsing xiaomi auth: %w", err)
	}
	return &auth, nil
}

// syncBandData syncs steps, sleep, heart rate, SpO2, and stress data.
func (s *SyncService) syncBandData(ctx context.Context, client *mifit.Client, userID string, dates []string, fromDate time.Time) (int, error) {
	records := 0

	// Get summary data (steps, sleep)
	summary, err := client.GetBandData(dates)
	if err != nil {
		return 0, fmt.Errorf("fetching summary: %w", err)
	}

	// Get detail data (HR, SpO2, stress)
	detail, err := client.GetBandDataDetail(dates)
	if err != nil {
		slog.Warn("fetching detail data failed, continuing with summary only", "error", err)
	}

	// Build detail lookup by date
	detailByDate := make(map[string]*mifit.BandDataItem)
	if detail != nil {
		for i := range detail.Data {
			detailByDate[detail.Data[i].DateStr] = &detail.Data[i]
		}
	}

	for _, item := range summary.Data {
		date, err := time.Parse("2006-01-02", item.DateStr)
		if err != nil {
			slog.Warn("invalid date in band data", "date", item.DateStr)
			continue
		}

		// Decode summary (steps + sleep)
		decoded, err := mifit.DecodeSummary(item.Summary)
		if err != nil {
			slog.Warn("decoding summary", "date", item.DateStr, "error", err)
			continue
		}

		// Save steps
		if decoded.Steps.Total > 0 {
			stagesJSON, _ := json.Marshal(decoded.Steps.Stages)
			rawStages := json.RawMessage(stagesJSON)
			steps := &model.HealthSteps{
				UserID:     userID,
				Date:       date,
				TotalSteps: &decoded.Steps.Total,
				DistanceM:  &decoded.Steps.Distance,
				Calories:   &decoded.Steps.Calories,
				Stages:     &rawStages,
			}
			if err := s.healthRepo.UpsertSteps(ctx, steps); err != nil {
				slog.Warn("upserting steps", "date", item.DateStr, "error", err)
			} else {
				records++
			}
		}

		// Save sleep
		if decoded.Sleep.Deep > 0 || decoded.Sleep.Light > 0 {
			sleepStart := mifit.SleepMinutesToTime(decoded.Sleep.Start, date)
			sleepEnd := mifit.SleepMinutesToTime(decoded.Sleep.End, date)
			duration := decoded.Sleep.Deep + decoded.Sleep.Light + decoded.Sleep.REM

			stagesJSON, _ := json.Marshal(decoded.Sleep.Stages)
			rawStages := json.RawMessage(stagesJSON)

			sleep := &model.HealthSleep{
				UserID:      userID,
				Date:        date,
				SleepStart:  &sleepStart,
				SleepEnd:    &sleepEnd,
				DurationMin: &duration,
				DeepMin:     &decoded.Sleep.Deep,
				LightMin:    &decoded.Sleep.Light,
				REMMin:      &decoded.Sleep.REM,
				AwakeMin:    &decoded.Sleep.Awake,
				Stages:      &rawStages,
			}
			if err := s.healthRepo.UpsertSleep(ctx, sleep); err != nil {
				slog.Warn("upserting sleep", "date", item.DateStr, "error", err)
			} else {
				records++
			}
		}

		// Process detail data for this date
		detailItem, ok := detailByDate[item.DateStr]
		if !ok {
			continue
		}

		// Save heart rate
		hrPoints, err := mifit.DecodeHeartRate(detailItem.DataHR, date)
		if err != nil {
			slog.Warn("decoding HR", "date", item.DateStr, "error", err)
		} else if len(hrPoints) > 0 {
			hrRecords := make([]model.HealthHeartRate, len(hrPoints))
			for i, p := range hrPoints {
				hrRecords[i] = model.HealthHeartRate{
					UserID:     userID,
					MeasuredAt: p.Time,
					BPM:        int16(p.BPM),
					Type:       "auto",
				}
			}
			n, err := s.healthRepo.BatchInsertHeartRate(ctx, hrRecords)
			if err != nil {
				slog.Warn("inserting HR", "date", item.DateStr, "error", err)
			} else {
				records += n
			}
		}

		// Save SpO2
		spo2Points, err := mifit.DecodeSpO2(detailItem.DataSpO2, date)
		if err != nil {
			slog.Warn("decoding SpO2", "date", item.DateStr, "error", err)
		} else if len(spo2Points) > 0 {
			spo2Records := make([]model.HealthSpO2, len(spo2Points))
			for i, p := range spo2Points {
				spo2Records[i] = model.HealthSpO2{
					UserID:     userID,
					MeasuredAt: p.Time,
					Value:      int16(p.Value),
				}
			}
			n, err := s.healthRepo.BatchInsertSpO2(ctx, spo2Records)
			if err != nil {
				slog.Warn("inserting SpO2", "date", item.DateStr, "error", err)
			} else {
				records += n
			}
		}

		// Save stress
		stressPoints, err := mifit.DecodeStress(detailItem.DataStress, date)
		if err != nil {
			slog.Warn("decoding stress", "date", item.DateStr, "error", err)
		} else if len(stressPoints) > 0 {
			stressRecords := make([]model.HealthStress, len(stressPoints))
			for i, p := range stressPoints {
				stressRecords[i] = model.HealthStress{
					UserID:     userID,
					MeasuredAt: p.Time,
					Value:      int16(p.Value),
				}
			}
			n, err := s.healthRepo.BatchInsertStress(ctx, stressRecords)
			if err != nil {
				slog.Warn("inserting stress", "date", item.DateStr, "error", err)
			} else {
				records += n
			}
		}
	}

	return records, nil
}

// syncWorkouts syncs workout history and details.
func (s *SyncService) syncWorkouts(ctx context.Context, client *mifit.Client, userID string) (int, error) {
	history, err := client.GetWorkoutHistory()
	if err != nil {
		return 0, fmt.Errorf("fetching workout history: %w", err)
	}

	records := 0

	for _, ws := range history.Data.Summary {
		startedAt := time.Unix(ws.StartTime, 0)
		endedAt := time.Unix(ws.EndTime, 0)
		durationSec := int(ws.EndTime - ws.StartTime)
		avgHR := int16(ws.AvgHR)
		maxHR := int16(ws.MaxHR)
		avgPace := float64(ws.AvgPace) / 60.0 // convert to min/km

		workout := &model.HealthWorkout{
			UserID:       userID,
			WorkoutType:  mifit.WorkoutTypeName(ws.Type),
			StartedAt:    startedAt,
			EndedAt:      &endedAt,
			DurationSec:  &durationSec,
			DistanceM:    &ws.Distance,
			Calories:     &ws.Calories,
			AvgHeartRate: &avgHR,
			MaxHeartRate: &maxHR,
			AvgPace:      &avgPace,
		}

		if err := s.healthRepo.UpsertWorkout(ctx, workout); err != nil {
			slog.Warn("upserting workout", "track_id", ws.TrackID, "error", err)
			continue
		}
		records++
	}

	return records, nil
}

func ptrVal(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
