package usecase

import (
	"context"
	"errors"

	"github.com/luismoralesarg/instagram-giveaways-api/internal/domain"
)

type LoginInput struct {
	Username string
	Password string
}

type LoginOutput struct {
	Token string
}

// Login implementa UC-0.1: autentica usuario/contraseña y emite un JWT.
type Login struct {
	users  domain.UserRepository
	hasher domain.PasswordHasher
	tokens domain.TokenIssuer
}

func NewLogin(users domain.UserRepository, hasher domain.PasswordHasher, tokens domain.TokenIssuer) *Login {
	return &Login{users: users, hasher: hasher, tokens: tokens}
}

func (uc *Login) Execute(ctx context.Context, in LoginInput) (LoginOutput, error) {
	user, err := uc.users.FindByUsername(ctx, in.Username)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return LoginOutput{}, domain.ErrInvalidCredentials
		}
		return LoginOutput{}, err
	}

	if err := uc.hasher.Compare(user.PasswordHash, in.Password); err != nil {
		return LoginOutput{}, domain.ErrInvalidCredentials
	}

	token, err := uc.tokens.Issue(user.ID)
	if err != nil {
		return LoginOutput{}, err
	}

	return LoginOutput{Token: token}, nil
}
