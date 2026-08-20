package domain

import "time"

type ParticipantSourceType string

const (
	ParticipantSourceComment      ParticipantSourceType = "comment"
	ParticipantSourceStoryMention ParticipantSourceType = "story_mention"
)

// Participant es alguien que interactuó con una campaña (comentario o
// mención/compartido en historia) y quedó registrado como candidato a
// ganar. No se borra ante una exclusión manual: se marca IsExcluded para
// trazabilidad, pero el motor de sorteo lo ignora.
type Participant struct {
	ID              string
	CampaignID      string
	InstagramUserID string
	Username        string
	SourceType      ParticipantSourceType
	IsExcluded      bool
	ExcludedReason  string
	CreatedAt       time.Time
}

// Exclude implementa UC-2.3. No valida el estado actual porque no hay
// transición inválida: incluso sobre una campaña ya sorteada se permite
// guardar la exclusión para futuros re-sorteos (FA-2.3.2).
func (p *Participant) Exclude(reason string) {
	p.IsExcluded = true
	p.ExcludedReason = reason
}
