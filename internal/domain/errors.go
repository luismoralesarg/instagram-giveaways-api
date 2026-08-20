package domain

import "errors"

var (
	// ErrUserNotFound indica que no existe un User con el username buscado.
	ErrUserNotFound = errors.New("usuario no encontrado")

	// ErrInvalidCredentials cubre tanto username inexistente como contraseña
	// incorrecta — no se distinguen para no revelar qué usernames existen.
	ErrInvalidCredentials = errors.New("credenciales inválidas")

	ErrInvalidCampaignType = errors.New("tipo de campaña inválido: debe ser 'post' o 'historia'")
	ErrMediaIDRequired     = errors.New("media_id es requerido")
	ErrCampaignNotFound    = errors.New("campaña no encontrada")
	ErrCampaignNotDraft    = errors.New("la campaña debe estar en estado 'draft'")
	ErrCampaignNotActive   = errors.New("la campaña debe estar en estado 'activa'")

	// ErrDuplicateMediaID cubre la regla de UC-1.1: un media_id no puede
	// tener más de una campaña activa/draft simultánea.
	ErrDuplicateMediaID = errors.New("ya existe una campaña activa o en draft para este media_id")

	// ErrActiveStoryCampaignExists cubre la regla acordada para UC-2.2: el
	// webhook de mención de historia no trae en el payload un identificador
	// que permita desambiguar a qué campaña corresponde si hay más de una
	// campaña de tipo historia activa a la vez, así que solo se permite una.
	ErrActiveStoryCampaignExists = errors.New("ya hay una campaña de tipo historia activa")

	ErrCampaignNotPost = errors.New("la campaña debe ser de tipo 'post'")

	ErrParticipantNotFound = errors.New("participante no encontrado")
)
