package service

import (
	"context"
	"errors"
	"testing"

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
	session := func(xuid string) *domain.SessionData {
		return &domain.SessionData{LinkedHaloIdentity: &domain.HaloIdentity{XUID: xuid}}
	}

	cases := []struct {
		name     string
		enforced bool
		sess     *domain.SessionData
		page     string
		allowed  []string
		refused  []string
	}{
		{"sans groupe : ses médias seuls", true, session("1"), "solo", []string{"solo", "Solo"}, []string{"ami", "moi", "autre"}},
		{"groupe : les co-membres", true, session("3"), "moi", []string{"moi", "ami"}, []string{"autre", "solo"}},
		{"page d'un ami : son autre groupe retiré", true, session("3"), "ami", []string{"ami", "moi"}, []string{"autre"}},
		{"mono-utilisateur : groupe entier de la page", false, nil, "ami", []string{"ami", "moi", "autre"}, []string{"solo"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a, err := NewMediaAudienceService(tc.enforced, load, coMembers, noAccounts{}).Resolve(context.Background(), tc.sess, tc.page)
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

// TestMediaAudienceService_Erreurs : une panne de lecture des profils ou des groupes remonte (le
// handler se replie alors sur la page seule et le journalise) ; une page sans profil suivi ne montre
// que ses médias, sans lire les groupes.
func TestMediaAudienceService_Erreurs(t *testing.T) {
	players := []domain.PlayerSummary{{PlayerSlug: "moi", Gamertag: "Moi", XUID: "3"}}
	okPlayers := func(string) ([]domain.PlayerSummary, error) { return players, nil }
	okGroups := func(string) (map[string]bool, error) { return nil, nil }
	boom := errors.New("groups.json illisible")
	sess := &domain.SessionData{LinkedHaloIdentity: &domain.HaloIdentity{XUID: "3"}}
	ctx := context.Background()

	_, err := NewMediaAudienceService(true,
		func(string) ([]domain.PlayerSummary, error) { return nil, boom }, okGroups, noAccounts{}).Resolve(ctx, sess, "moi")
	if !errors.Is(err, boom) {
		t.Errorf("profils illisibles : %v, attendu l'erreur de la source", err)
	}
	_, err = NewMediaAudienceService(true, okPlayers,
		func(string) (map[string]bool, error) { return nil, boom }, noAccounts{}).Resolve(ctx, sess, "moi")
	if !errors.Is(err, boom) {
		t.Errorf("groupes de la page illisibles : %v, attendu l'erreur de la source", err)
	}
	calls := 0
	a, err := NewMediaAudienceService(true, okPlayers,
		func(string) (map[string]bool, error) { calls++; return nil, nil }, noAccounts{}).Resolve(ctx, sess, "inconnu")
	if err != nil || !a.Allows("inconnu") || a.Allows("moi") || calls != 0 {
		t.Errorf("page sans profil : audience %v, erreur %v, %d lecture(s) de groupes ; attendu la page seule sans lecture", a.Authors(), err, calls)
	}
}
