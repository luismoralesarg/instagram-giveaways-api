package fiber

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"github.com/luismoralesarg/instagram-giveaways-api/internal/application/usecase"
	"github.com/luismoralesarg/instagram-giveaways-api/internal/domain"
)

type ParticipantHandler struct {
	syncComments *usecase.SyncComments
	exclude      *usecase.ExcludeParticipant
	list         *usecase.ListParticipants
}

func NewParticipantHandler(syncComments *usecase.SyncComments, exclude *usecase.ExcludeParticipant, list *usecase.ListParticipants) *ParticipantHandler {
	return &ParticipantHandler{syncComments: syncComments, exclude: exclude, list: list}
}

type syncCommentsResponse struct {
	NewParticipants int `json:"new_participants"`
}

// SyncComments implementa POST /campaigns/:id/sync-comments (UC-2.1).
func (h *ParticipantHandler) SyncComments(c *fiber.Ctx) error {
	out, err := h.syncComments.Execute(c.Context(), usecase.SyncCommentsInput{CampaignID: c.Params("id")})
	if err != nil {
		return participantError(err)
	}
	return c.JSON(syncCommentsResponse{NewParticipants: out.NewParticipants})
}

type excludeParticipantRequest struct {
	Reason string `json:"reason"`
}

// Exclude implementa PATCH /campaigns/:id/participants/:pid/exclude (UC-2.3).
func (h *ParticipantHandler) Exclude(c *fiber.Ctx) error {
	var req excludeParticipantRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "body inválido")
	}

	err := h.exclude.Execute(c.Context(), usecase.ExcludeParticipantInput{
		CampaignID:    c.Params("id"),
		ParticipantID: c.Params("pid"),
		Reason:        req.Reason,
	})
	if err != nil {
		return participantError(err)
	}
	return c.SendStatus(fiber.StatusOK)
}

type participantResponse struct {
	ID              string `json:"id"`
	InstagramUserID string `json:"instagram_user_id"`
	Username        string `json:"username"`
	SourceType      string `json:"source_type"`
	IsExcluded      bool   `json:"is_excluded"`
	ExcludedReason  string `json:"excluded_reason,omitempty"`
}

// List implementa GET /campaigns/:id/participants.
func (h *ParticipantHandler) List(c *fiber.Ctx) error {
	participants, err := h.list.Execute(c.Context(), c.Params("id"))
	if err != nil {
		return participantError(err)
	}

	resp := make([]participantResponse, 0, len(participants))
	for _, p := range participants {
		resp = append(resp, participantResponse{
			ID:              p.ID,
			InstagramUserID: p.InstagramUserID,
			Username:        p.Username,
			SourceType:      string(p.SourceType),
			IsExcluded:      p.IsExcluded,
			ExcludedReason:  p.ExcludedReason,
		})
	}
	return c.JSON(resp)
}

func participantError(err error) error {
	switch {
	case errors.Is(err, domain.ErrCampaignNotFound), errors.Is(err, domain.ErrParticipantNotFound):
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	case errors.Is(err, domain.ErrCampaignNotActive), errors.Is(err, domain.ErrCampaignNotPost):
		return fiber.NewError(fiber.StatusConflict, err.Error())
	default:
		return fiber.NewError(fiber.StatusInternalServerError, "error interno")
	}
}
