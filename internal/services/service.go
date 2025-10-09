package services

import (
	"time"

	"github.com/acertainpoggerman/online-exam-system/internal/repository"
)

type Service struct {
	repo          repository.Repository
	jwtSecretKey  string
	jwtExpiryTime time.Duration
}

func NewService(r repository.Repository, jwtSecretKey string, jwtExpiryTime time.Duration) *Service {
	return &Service{
		repo:          r,
		jwtSecretKey:  jwtSecretKey,
		jwtExpiryTime: jwtExpiryTime,
	}
}
