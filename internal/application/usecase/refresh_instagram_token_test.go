package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/luismoralesarg/instagram-giveaways-api/internal/domain"
)

func TestRefreshInstagramToken_SkipsWhenFarFromExpiry(t *testing.T) {
	fixedNow := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	tokens := &fakeInstagramTokenRepo{token: &domain.InstagramToken{
		AccessToken: "current",
		ExpiresAt:   fixedNow.Add(30 * 24 * time.Hour), // muy lejos del refreshWindow (10 días)
	}}
	refresher := &fakeInstagramTokenRefresher{}

	uc := NewRefreshInstagramTokenIfNeeded(tokens, refresher)
	uc.now = func() time.Time { return fixedNow }

	out, err := uc.Execute(context.Background())
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if out.Refreshed {
		t.Error("Refreshed = true, no debería haber refrescado (todavía falta mucho para vencer)")
	}
	if refresher.calledWith != "" {
		t.Error("no debería haber llamado al refresher")
	}
}

func TestRefreshInstagramToken_RefreshesWhenCloseToExpiry(t *testing.T) {
	fixedNow := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	tokens := &fakeInstagramTokenRepo{token: &domain.InstagramToken{
		AccessToken: "current",
		ExpiresAt:   fixedNow.Add(3 * 24 * time.Hour), // dentro del refreshWindow
	}}
	refresher := &fakeInstagramTokenRefresher{newAccessToken: "fresh", expiresIn: 60 * 24 * time.Hour}

	uc := NewRefreshInstagramTokenIfNeeded(tokens, refresher)
	uc.now = func() time.Time { return fixedNow }

	out, err := uc.Execute(context.Background())
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !out.Refreshed {
		t.Fatal("Refreshed = false, quiero true")
	}
	if refresher.calledWith != "current" {
		t.Errorf("refresher llamado con %q, quiero %q", refresher.calledWith, "current")
	}

	wantExpiresAt := fixedNow.Add(60 * 24 * time.Hour)
	if !out.ExpiresAt.Equal(wantExpiresAt) {
		t.Errorf("ExpiresAt = %v, quiero %v", out.ExpiresAt, wantExpiresAt)
	}

	saved, _ := tokens.Get(context.Background())
	if saved.AccessToken != "fresh" {
		t.Errorf("token guardado = %q, quiero %q", saved.AccessToken, "fresh")
	}
}

func TestRefreshInstagramToken_AlreadyExpired_StillRefreshes(t *testing.T) {
	fixedNow := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	tokens := &fakeInstagramTokenRepo{token: &domain.InstagramToken{
		AccessToken: "vencido",
		ExpiresAt:   fixedNow.Add(-24 * time.Hour),
	}}
	refresher := &fakeInstagramTokenRefresher{newAccessToken: "fresh", expiresIn: 60 * 24 * time.Hour}

	uc := NewRefreshInstagramTokenIfNeeded(tokens, refresher)
	uc.now = func() time.Time { return fixedNow }

	out, err := uc.Execute(context.Background())
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !out.Refreshed {
		t.Fatal("Refreshed = false, quiero true (un token vencido igual debería intentar refrescarse)")
	}
}

func TestRefreshInstagramToken_NoTokenSeeded(t *testing.T) {
	uc := NewRefreshInstagramTokenIfNeeded(&fakeInstagramTokenRepo{}, &fakeInstagramTokenRefresher{})
	_, err := uc.Execute(context.Background())
	if !errors.Is(err, domain.ErrInstagramTokenNotFound) {
		t.Fatalf("err = %v, quiero %v", err, domain.ErrInstagramTokenNotFound)
	}
}

func TestRefreshInstagramToken_RefresherError_Propagates(t *testing.T) {
	fixedNow := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	tokens := &fakeInstagramTokenRepo{token: &domain.InstagramToken{
		AccessToken: "current",
		ExpiresAt:   fixedNow.Add(time.Hour),
	}}
	boom := errors.New("graph api caída")
	refresher := &fakeInstagramTokenRefresher{err: boom}

	uc := NewRefreshInstagramTokenIfNeeded(tokens, refresher)
	uc.now = func() time.Time { return fixedNow }

	_, err := uc.Execute(context.Background())
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v, quiero que envuelva %v", err, boom)
	}
}
