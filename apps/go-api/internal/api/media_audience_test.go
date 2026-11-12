package api

import (
	"context"
	"errors"
	"testing"

	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/service"
)

type noAccounts struct{}

func (noAccounts) Get(string) (*domain.User, error)       { return nil, errors.New("absent") }
func (noAccounts) GetByXUID(string) (*domain.User, error) { return nil, errors.New("absent") }

// TestMediaAudienceResolver_LitLaSessionDuContexte : le résolveur câblé aux handlers passe au
// service la session posée par le middleware ; un lecteur dont la session est dans le groupe de
// la page voit ses co-membres, un contexte sans session ne voit que la page.
func TestMediaAudienceResolver_LitLaSessionDuContexte(t *testing.T) {
	players := []domain.PlayerSummary{
		{PlayerSlug: "ami", Gamertag: "Ami", XUID: "2"},
		{PlayerSlug: "moi", Gamertag: "Moi", XUID: "3"},
	}
	groups := map[string]map[string]bool{"2": {"2": true, "3": true}, "3": {"2": true, "3": true}}
	svc := service.NewMediaAudienceService(true,
		func(string) ([]domain.PlayerSummary, error) { return players, nil },
		func(x string) (map[string]bool, error) { return groups[x], nil },
		noAccounts{})
	resolve := mediaAudienceResolver(svc)

	withSession := middleware.InjectSession(context.Background(),
		&domain.SessionData{LinkedHaloIdentity: &domain.HaloIdentity{XUID: "3"}})
	a, err := resolve(withSession, "ami")
	if err != nil {
		t.Fatal(err)
	}
	if !a.Allows("moi") || !a.Allows("ami") {
		t.Errorf("lecteur du groupe : audience %v, attendu ami et moi", a.Authors())
	}

	a, err = resolve(context.Background(), "ami")
	if err != nil {
		t.Fatal(err)
	}
	if a.Allows("moi") {
		t.Errorf("sans session : audience %v, attendu la page seule", a.Authors())
	}
}
