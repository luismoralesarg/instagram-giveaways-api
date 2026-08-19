package fiber

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"github.com/luismoralesarg/instagram-giveaways-api/internal/application/usecase"
	"github.com/luismoralesarg/instagram-giveaways-api/internal/domain"
)

type AuthHandler struct {
	login *usecase.Login
}

func NewAuthHandler(login *usecase.Login) *AuthHandler {
	return &AuthHandler{login: login}
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginResponse struct {
	Token string `json:"token"`
}

// Login implementa POST /auth/login (UC-0.1).
func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req loginRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "body inválido")
	}
	if req.Username == "" || req.Password == "" {
		return fiber.NewError(fiber.StatusBadRequest, "username y password son requeridos")
	}

	out, err := h.login.Execute(c.Context(), usecase.LoginInput{
		Username: req.Username,
		Password: req.Password,
	})
	if err != nil {
		if errors.Is(err, domain.ErrInvalidCredentials) {
			return fiber.NewError(fiber.StatusUnauthorized, "credenciales inválidas")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "error interno")
	}

	return c.JSON(loginResponse{Token: out.Token})
}
