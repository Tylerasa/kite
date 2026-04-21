package auth

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/kite/internal/domain/exceptions"
	"github.com/kite/internal/domain/models"
	"github.com/kite/internal/domain/ports/in"
	"github.com/kite/internal/domain/ports/out"
	"golang.org/x/crypto/bcrypt"
)

const bcryptCost = 12

type UseCase struct {
	users      out.UserRepository
	accounts   out.AccountRepository
	jwtSecret  []byte
}

func NewUseCase(users out.UserRepository, accounts out.AccountRepository, jwtSecret string) *UseCase {
	return &UseCase{
		users:     users,
		accounts:  accounts,
		jwtSecret: []byte(jwtSecret),
	}
}

func (uc *UseCase) Signup(ctx context.Context, cmd in.SignupCommand) (*in.TokenResult, error) {
	email := strings.ToLower(strings.TrimSpace(cmd.Email))
	if email == "" || !strings.Contains(email, "@") {
		return nil, exceptions.ErrInvalidCredentials.WithDetails(map[string]interface{}{
			"field": "email", "reason": "invalid email format",
		})
	}
	if len(cmd.Password) < 8 {
		return nil, exceptions.ErrInvalidCredentials.WithDetails(map[string]interface{}{
			"field": "password", "reason": "must be at least 8 characters",
		})
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(cmd.Password), bcryptCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	user := &models.User{
		ID:           uuid.New(),
		Email:        email,
		PasswordHash: string(hash),
		CreatedAt:    time.Now().UTC(),
	}

	if err := uc.users.Create(ctx, user); err != nil {
		// Treat unique violation as user already exists
		return nil, exceptions.ErrUserAlreadyExists
	}

	// Create all 5 currency wallet accounts at signup — never lazily.
	for _, currency := range models.SupportedCurrencies {
		uid := user.ID
		account := &models.Account{
			ID:        uuid.New(),
			UserID:    &uid,
			Currency:  currency,
			Type:      models.AccountTypeUserWallet,
			Name:      fmt.Sprintf("Wallet - %s", currency),
			CreatedAt: time.Now().UTC(),
		}
		if err := uc.accounts.Create(ctx, account); err != nil {
			return nil, fmt.Errorf("create wallet account %s: %w", currency, err)
		}
	}

	token, err := uc.generateToken(user.ID)
	if err != nil {
		return nil, err
	}

	return &in.TokenResult{Token: token, UserID: user.ID}, nil
}

func (uc *UseCase) Login(ctx context.Context, cmd in.LoginCommand) (*in.TokenResult, error) {
	email := strings.ToLower(strings.TrimSpace(cmd.Email))
	user, err := uc.users.GetByEmail(ctx, email)
	if err != nil {
		// Return same error for not-found and wrong-password to prevent user enumeration.
		return nil, exceptions.ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(cmd.Password)); err != nil {
		return nil, exceptions.ErrInvalidCredentials
	}

	token, err := uc.generateToken(user.ID)
	if err != nil {
		return nil, err
	}

	return &in.TokenResult{Token: token, UserID: user.ID}, nil
}

func (uc *UseCase) generateToken(userID uuid.UUID) (string, error) {
	claims := jwt.MapClaims{
		"sub": userID.String(),
		"exp": time.Now().Add(24 * time.Hour).Unix(),
		"iat": time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(uc.jwtSecret)
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}
	return signed, nil
}
