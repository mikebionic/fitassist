package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mike/fitassist/internal/crypto"
	"github.com/mike/fitassist/internal/mifit"
	"github.com/mike/fitassist/internal/model"
	"github.com/mike/fitassist/internal/repository"
)

type MiFitService struct {
	mifitRepo *repository.MiFitRepository
	syncSvc   *SyncService
	apiBase   string
	encKey    string
}

func NewMiFitService(
	mifitRepo *repository.MiFitRepository,
	syncSvc *SyncService,
	apiBase, encKey string,
) *MiFitService {
	return &MiFitService{
		mifitRepo: mifitRepo,
		syncSvc:   syncSvc,
		apiBase:   apiBase,
		encKey:    encKey,
	}
}

type LinkRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LinkResult struct {
	AccountID string `json:"account_id"`
	MiUserID  string `json:"mi_user_id"`
}

// Link validates Mi Fitness credentials and stores them encrypted.
func (s *MiFitService) Link(ctx context.Context, userID string, req LinkRequest) (*LinkResult, error) {
	if s.encKey == "" {
		return nil, fmt.Errorf("encryption key not configured")
	}

	// Test login first
	client := mifit.NewClient(s.apiBase)
	authResult, err := client.Login(req.Email, req.Password)
	if err != nil {
		return nil, fmt.Errorf("Mi Fitness login failed: %w", err)
	}

	// Encrypt password
	encryptedPassword, err := crypto.Encrypt([]byte(req.Password), s.encKey)
	if err != nil {
		return nil, fmt.Errorf("encrypting password: %w", err)
	}

	// Check if account already exists
	existing, _ := s.mifitRepo.GetByUserID(ctx, userID)
	if existing != nil {
		// Update existing
		_ = s.mifitRepo.Delete(ctx, existing.ID)
	}

	acc := &model.MiFitAccount{
		UserID:     userID,
		MiEmail:    req.Email,
		MiPassword: encryptedPassword,
		SyncEnabled: true,
	}

	if err := s.mifitRepo.Create(ctx, acc); err != nil {
		return nil, fmt.Errorf("saving account: %w", err)
	}

	// Store the token
	if err := s.mifitRepo.UpdateToken(ctx, acc.ID, authResult.AppToken, authResult.UserIDMi); err != nil {
		return nil, fmt.Errorf("storing token: %w", err)
	}

	// Store auth method and Xiaomi credentials if applicable
	if authResult.AuthMethod == "xiaomi" && authResult.XiaomiAuth != nil {
		xiaomiJSON, err := json.Marshal(authResult.XiaomiAuth)
		if err != nil {
			return nil, fmt.Errorf("serializing xiaomi auth: %w", err)
		}
		encXiaomiAuth, err := crypto.Encrypt(xiaomiJSON, s.encKey)
		if err != nil {
			return nil, fmt.Errorf("encrypting xiaomi auth: %w", err)
		}
		if err := s.mifitRepo.UpdateAuthMethod(ctx, acc.ID, "xiaomi", encXiaomiAuth); err != nil {
			return nil, fmt.Errorf("storing auth method: %w", err)
		}
	}

	return &LinkResult{
		AccountID: acc.ID,
		MiUserID:  authResult.UserIDMi,
	}, nil
}

// TriggerSync manually triggers a sync for a user's Mi Fitness account.
func (s *MiFitService) TriggerSync(ctx context.Context, userID string) error {
	acc, err := s.mifitRepo.GetByUserID(ctx, userID)
	if err != nil {
		return fmt.Errorf("account not found: %w", err)
	}

	return s.syncSvc.SyncAccount(ctx, acc)
}

type AccountStatus struct {
	Linked      bool   `json:"linked"`
	Email       string `json:"email,omitempty"`
	SyncEnabled bool   `json:"sync_enabled"`
	LastSync    string `json:"last_sync,omitempty"`
}

// GetStatus returns the connection status for a user's Mi Fitness account.
func (s *MiFitService) GetStatus(ctx context.Context, userID string) (*AccountStatus, error) {
	acc, err := s.mifitRepo.GetByUserID(ctx, userID)
	if err != nil {
		return &AccountStatus{Linked: false}, nil
	}

	status := &AccountStatus{
		Linked:      true,
		Email:       acc.MiEmail,
		SyncEnabled: acc.SyncEnabled,
	}

	if acc.LastSync != nil {
		status.LastSync = acc.LastSync.Format("2006-01-02T15:04:05Z")
	}

	return status, nil
}
