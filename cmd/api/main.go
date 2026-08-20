package main

import (
	"context"
	"log"

	"github.com/gofiber/fiber/v2"

	"github.com/luismoralesarg/instagram-giveaways-api/internal/application/usecase"
	"github.com/luismoralesarg/instagram-giveaways-api/internal/config"
	"github.com/luismoralesarg/instagram-giveaways-api/internal/infrastructure/auth"
	fiberhttp "github.com/luismoralesarg/instagram-giveaways-api/internal/infrastructure/http/fiber"
	"github.com/luismoralesarg/instagram-giveaways-api/internal/infrastructure/instagram"
	"github.com/luismoralesarg/instagram-giveaways-api/internal/infrastructure/persistence/postgres"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	ctx := context.Background()
	pool, err := postgres.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("conexión a postgres: %v", err)
	}
	defer pool.Close()

	userRepo := postgres.NewUserRepository(pool)
	campaignRepo := postgres.NewCampaignRepository(pool)
	participantRepo := postgres.NewParticipantRepository(pool)
	hasher := auth.NewBcryptHasher()
	tokens := auth.NewJWTIssuer(cfg.JWTSecret, cfg.JWTExpiration)
	instagramClient := instagram.NewGraphClient(cfg.InstagramAccessToken)

	loginUC := usecase.NewLogin(userRepo, hasher, tokens)
	createCampaignUC := usecase.NewCreateCampaign(campaignRepo)
	activateCampaignUC := usecase.NewActivateCampaign(campaignRepo)
	closeCampaignUC := usecase.NewCloseCampaign(campaignRepo)
	syncCommentsUC := usecase.NewSyncComments(campaignRepo, participantRepo, instagramClient)
	handleStoryMentionUC := usecase.NewHandleStoryMention(campaignRepo, participantRepo)
	excludeParticipantUC := usecase.NewExcludeParticipant(participantRepo)
	listParticipantsUC := usecase.NewListParticipants(campaignRepo, participantRepo)

	app := fiber.New()
	handlers := fiberhttp.Handlers{
		Auth:        fiberhttp.NewAuthHandler(loginUC),
		Campaign:    fiberhttp.NewCampaignHandler(createCampaignUC, activateCampaignUC, closeCampaignUC),
		Participant: fiberhttp.NewParticipantHandler(syncCommentsUC, excludeParticipantUC, listParticipantsUC),
		Webhook:     fiberhttp.NewWebhookHandler(handleStoryMentionUC, cfg.InstagramWebhookVerifyToken, cfg.InstagramAppSecret),
	}
	fiberhttp.NewRouter(app, handlers, tokens)

	log.Printf("escuchando en :%s", cfg.Port)
	if err := app.Listen(":" + cfg.Port); err != nil {
		log.Fatalf("servidor: %v", err)
	}
}
