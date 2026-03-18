package profile

import (
	"database/sql"
	"errors"
)

type Service struct {
	repo *Repository
}

func NewService(db *sql.DB) *Service {
	return &Service{
		repo: NewRepository(db),
	}
}

// GetProfile gets user profile
func (s *Service) GetProfile(userID string) (*UserProfile, error) {
	profile, err := s.repo.GetProfile(userID)
	if err == sql.ErrNoRows {
		return nil, errors.New("profile not found")
	}
	return profile, err
}

// UpdateProfile updates user profile
func (s *Service) UpdateProfile(userID string, req UpdateProfileRequest) (*UserProfile, error) {
	return s.repo.UpdateProfile(userID, req)
}
