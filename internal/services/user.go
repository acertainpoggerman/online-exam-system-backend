package services

import (
	"context"
	"fmt"
	"time"

	"github.com/acertainpoggerman/online-exam-system/internal/models"
	"github.com/acertainpoggerman/online-exam-system/internal/tools"
)

// Registration service.
func (svc *Service) Register(
	ctx context.Context,
	data models.RegisterRequestBody,
) (string, error) {

	passwordHash, err := tools.GeneratePasswordHash(data.Password)
	if err != nil {
		return "", err
	}

	userID, err := svc.repo.CreateUser(ctx, models.User{
		// [User Data]
		Role:         data.Role,
		Email:        data.Email,
		PasswordHash: passwordHash,
		// [PTA]
		OpCode:    "system",
		CreatedAt: time.Now(),
	})
	if err != nil {
		return "", err
	}

	user, _ := svc.repo.FindUserByID(ctx, userID)
	tokenString, err := tools.CreateJWT(tools.UserData{
		ID:    user.ID.Hex(),
		Email: user.Email,
		Role:  user.Role,
	}, []byte(svc.jwtSecretKey), svc.jwtExpiryTime)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// Log in service.
func (svc *Service) Login(
	ctx context.Context,
	data models.LoginRequestBody,
) (string, error) {

	user, err := svc.repo.FindUserByEmail(ctx, data.Email)
	if err != nil {
		return "", err
	}

	if !tools.VerifyPassword(data.Password, user.PasswordHash) {
		return "", fmt.Errorf("invalid credentials provided")
	}

	tokenString, err := tools.CreateJWT(tools.UserData{
		ID:    user.ID.Hex(),
		Email: user.Email,
		Role:  user.Role,
	}, []byte(svc.jwtSecretKey), svc.jwtExpiryTime)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// Get user information.
func (svc *Service) GetUserByID(ctx context.Context, userID string) (*models.User, error) {

	user, err := svc.repo.FindUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	return user, nil
}

// Delete user information
func (svc *Service) DeleteUserByID(ctx context.Context, userID string) error {

	err := svc.repo.DeleteUserByID(ctx, userID)
	return err
}
