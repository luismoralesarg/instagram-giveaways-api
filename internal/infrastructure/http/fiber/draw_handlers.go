package fiber

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"github.com/luismoralesarg/instagram-giveaways-api/internal/application/usecase"
	"github.com/luismoralesarg/instagram-giveaways-api/internal/domain"
)

type DrawHandler struct {
	run *usecase.RunDraw
	get *usecase.GetDrawResult
}

func NewDrawHandler(run *usecase.RunDraw, get *usecase.GetDrawResult) *DrawHandler {
	return &DrawHandler{run: run, get: get}
}

type runDrawRequest struct {
	WinnersCount int `json:"winners_count"`
}

type winnerResponse struct {
	ParticipantID   string `json:"participant_id"`
	InstagramUserID string `json:"instagram_user_id"`
	Username        string `json:"username"`
	Position        int    `json:"position,omitempty"`
}

type drawResponse struct {
	ID           string           `json:"id"`
	WinnersCount int              `json:"winners_count"`
	RandomSeed   int64            `json:"random_seed"`
	Winners      []winnerResponse `json:"winners"`
}

// toDrawResponse omite `position` cuando el sorteo tiene un único ganador
// (UC-3.1: "sin posición si es un único ganador").
func toDrawResponse(d domain.Draw) drawResponse {
	resp := drawResponse{
		ID:           d.ID,
		WinnersCount: d.WinnersCount,
		RandomSeed:   d.RandomSeed,
		Winners:      make([]winnerResponse, 0, len(d.Winners)),
	}
	for _, w := range d.Winners {
		wr := winnerResponse{
			ParticipantID:   w.ParticipantID,
			InstagramUserID: w.InstagramUserID,
			Username:        w.Username,
		}
		if d.WinnersCount > 1 {
			wr.Position = w.Position
		}
		resp.Winners = append(resp.Winners, wr)
	}
	return resp
}

// Run implementa POST /campaigns/:id/draws (UC-3.1).
func (h *DrawHandler) Run(c *fiber.Ctx) error {
	var req runDrawRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "body inválido")
	}

	out, err := h.run.Execute(c.Context(), usecase.RunDrawInput{
		CampaignID:   c.Params("id"),
		WinnersCount: req.WinnersCount,
	})
	if err != nil {
		return drawError(err)
	}

	return c.Status(fiber.StatusCreated).JSON(toDrawResponse(out.Draw))
}

// Get implementa GET /campaigns/:id/draws/:draw_id (UC-3.2).
func (h *DrawHandler) Get(c *fiber.Ctx) error {
	draw, err := h.get.Execute(c.Context(), usecase.GetDrawResultInput{
		CampaignID: c.Params("id"),
		DrawID:     c.Params("draw_id"),
	})
	if err != nil {
		return drawError(err)
	}
	return c.JSON(toDrawResponse(draw))
}

func drawError(err error) error {
	switch {
	case errors.Is(err, domain.ErrInvalidWinnersCount):
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	case errors.Is(err, domain.ErrCampaignNotFound), errors.Is(err, domain.ErrDrawNotFound):
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	case errors.Is(err, domain.ErrCampaignNotDrawable), errors.Is(err, domain.ErrNotEnoughParticipants):
		return fiber.NewError(fiber.StatusConflict, err.Error())
	default:
		return fiber.NewError(fiber.StatusInternalServerError, "error interno")
	}
}
