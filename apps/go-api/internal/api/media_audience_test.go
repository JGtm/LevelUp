package api

import (
	"context"
	"errors"
	"testing"

	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/domain"
)

type noAccounts struct{}

func (noAccounts) Get(string) (*domain.User, error)       { return nil, errors.New("absent") }
func (noAccounts) GetByXUID(string) (*domain.User, error) { return nil, errors.New("absent") }

// TestMediaAudienceResolver : la page d'un joueur montre les médias de son groupe ; sans
// groupe, les siens seuls ; un lecteur ne voit pas, par un ami commun, les autres
// groupes de cet ami.
func TestMediaAudienceResolver(t *testing.T) {
	players := []domain.PlayerSummary{
		{PlayerSlug: "solo", Gamertag: "Solo", XUID: "1"},
		{PlayerSlug: "ami", Gamertag: "Ami", XUID: "2"},
		{PlayerSlug: "moi", Gamertag: "Moi", XUID: "3"},
		{PlayerSlug: "autre", Gamertag: "Autre", XUID: "4"},
	}
	// Groupes : {ami, moi} et {ami, autre}.
	groups := map[string]map[string]bool{
		"2": {"2": true, "3": true, "4": true},
		"3": {"2": true, "3": true},
		"4": {"2": true, "4": true},
	}
	coMembers := func(x string) (map[string]bool, error) { return groups[x], nil }
	load := func(string) ([]domain.PlayerSummary, error) { return players, nil }
	session := func(xuid string) context.Context {
		return middleware.InjectSession(context.Background(),
			&domain.SessionData{LinkedHaloIdentity: &domain.HaloIdentity{XUID: xuid}})
	}

	cases := []struct {
		name     string
		enforced bool
		ctx      context.Context
		page     string
		allowed  []string
		refused  []string
	}{
		{"sans groupe : ses médias seuls", true, session("1"), "solo", []string{"solo", "Solo"}, []string{"ami", "moi", "autre"}},
		{"groupe : les co-membres", true, session("3"), "moi", []string{"moi", "ami"}, []string{"autre", "solo"}},
		{"page d'un ami : son autre groupe retiré", true, session("3"), "ami", []string{"ami", "moi"}, []string{"autre"}},
		{"mono-utilisateur : groupe entier de la page", false, context.Background(), "ami", []string{"ami", "moi", "autre"}, []string{"solo"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a, err := mediaAudienceResolver(tc.enforced, load, coMembers, noAccounts{})(tc.ctx, tc.page)
			if err != nil {
				t.Fatal(err)
			}
			for _, n := range tc.allowed {
				if !a.Allows(n) {
					t.Errorf("%s refusé, attendu visible (audience %v)", n, a.Authors())
				}
			}
			for _, n := range tc.refused {
				if a.Allows(n) {
					t.Errorf("%s visible, attendu refusé (audience %v)", n, a.Authors())
				}
			}
		})
	}
}
