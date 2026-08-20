package fiber

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"github.com/luismoralesarg/instagram-giveaways-api/internal/application/usecase"
	"github.com/luismoralesarg/instagram-giveaways-api/internal/domain"
)

type CampaignHandler struct {
	create   *usecase.CreateCampaign
	activate *usecase.ActivateCampaign
	closeUC  *usecase.CloseCampaign
}

func NewCampaignHandler(create *usecase.CreateCampaign, activate *usecase.ActivateCampaign, closeUC *usecase.CloseCampaign) *CampaignHandler {
	return &CampaignHandler{create: create, activate: activate, closeUC: closeUC}
}

type createCampaignRequest struct {
	Type        string `json:"type"`
	MediaID     string `json:"media_id"`
	Name        string `json:"name"`
	MustFollow  bool   `json:"must_follow"`
	MinMentions int    `json:"min_mentions"`
}

type createCampaignResponse struct {
	ID string `json:"id"`
}

// Create implementa POST /campaigns (UC-1.1).
func (h *CampaignHandler) Create(c *fiber.Ctx) error {
	var req createCampaignRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "body inválido")
	}

	out, err := h.create.Execute(c.Context(), usecase.CreateCampaignInput{
		Type:        domain.CampaignType(req.Type),
		MediaID:     req.MediaID,
		Name:        req.Name,
		MustFollow:  req.MustFollow,
		MinMentions: req.MinMentions,
	})
	if err != nil {
		return campaignError(err)
	}

	return c.Status(fiber.StatusCreated).JSON(createCampaignResponse{ID: out.CampaignID})
}

// Activate implementa POST /campaigns/:id/activate (UC-1.2).
func (h *CampaignHandler) Activate(c *fiber.Ctx) error {
	if err := h.activate.Execute(c.Context(), c.Params("id")); err != nil {
		return campaignError(err)
	}
	return c.SendStatus(fiber.StatusOK)
}

// Close implementa POST /campaigns/:id/close (UC-1.3).
func (h *CampaignHandler) Close(c *fiber.Ctx) error {
	if err := h.closeUC.Execute(c.Context(), c.Params("id")); err != nil {
		return campaignError(err)
	}
	return c.SendStatus(fiber.StatusOK)
}

func campaignError(err error) error {
	switch {
	case errors.Is(err, domain.ErrInvalidCampaignType), errors.Is(err, domain.ErrMediaIDRequired):
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	case errors.Is(err, domain.ErrCampaignNotFound):
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	case errors.Is(err, domain.ErrDuplicateMediaID),
		errors.Is(err, domain.ErrCampaignNotDraft),
		errors.Is(err, domain.ErrCampaignNotActive),
		errors.Is(err, domain.ErrActiveStoryCampaignExists):
		return fiber.NewError(fiber.StatusConflict, err.Error())
	default:
		return fiber.NewError(fiber.StatusInternalServerError, "error interno")
	}
}
