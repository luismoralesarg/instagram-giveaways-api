package domain

import "time"

// User es el administrador que se autentica contra la API. No participa en
// las reglas de negocio de sorteos.
type User struct {
	ID           string
	Username     string
	PasswordHash string
	CreatedAt    time.Time
}
