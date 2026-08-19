package domain

import "errors"

var (
	// ErrUserNotFound indica que no existe un User con el username buscado.
	ErrUserNotFound = errors.New("usuario no encontrado")

	// ErrInvalidCredentials cubre tanto username inexistente como contraseña
	// incorrecta — no se distinguen para no revelar qué usernames existen.
	ErrInvalidCredentials = errors.New("credenciales inválidas")
)
