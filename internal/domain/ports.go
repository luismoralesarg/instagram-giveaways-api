package domain

import "context"

// UserRepository persiste y consulta usuarios administradores. No hay
// método de alta: los usuarios se crean manualmente (ver cmd/seedadmin).
type UserRepository interface {
	FindByUsername(ctx context.Context, username string) (*User, error)
}

// PasswordHasher hashea y compara contraseñas. Implementado en
// infrastructure/auth con bcrypt.
type PasswordHasher interface {
	Hash(password string) (string, error)
	Compare(hash, password string) error
}

// TokenIssuer emite y valida los tokens de acceso (JWT) usados para
// autenticar al administrador en las rutas protegidas.
type TokenIssuer interface {
	Issue(userID string) (string, error)
	Parse(token string) (userID string, err error)
}
