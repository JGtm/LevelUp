package wire

import (
	"context"
	"errors"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/platform/auth"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/port"
)

type fakeGenQueueWire struct {
	got []domain.ReplayGenerationDemand
}

func (q *fakeGenQueueWire) Enqueue(d domain.ReplayGenerationDemand) (domain.ReplayGenerationStatus, bool, error) {
	q.got = append(q.got, d)
	return domain.ReplayGenerationStatus{State: domain.ReplayGenerationQueued, Position: 1}, true, nil
}

func (q *fakeGenQueueWire) DemandStatus(string, string) (domain.ReplayGenerationStatus, bool) {
	return domain.ReplayGenerationStatus{}, false
}

type fakeGenProbeWire struct{}

func (fakeGenProbeWire) ProbeReplayMatch(context.Context, string, string) (port.ReplayMatchProbeResult, error) {
	return port.ReplayMatchProbeResult{FullID: "0a0a0a0a-1-2-3-4", Known: true, FilmArchived: true}, nil
}

// La fabrique lie le service au profil du CHEMIN : la demande porte le xuid et le pseudo de ce profil, résolu par la
// fabrique du registre (la casse du slug n'est jamais supposée), et aucun autre.
func TestReplayGeneration_FabriqueLieLaDemandeAuProfilDuChemin(t *testing.T) {
	var resolved []string
	queue := &fakeGenQueueWire{}
	pdb := &duckdb.PlayerDB{XUID: "xuid-profil-de-la-page", Gamertag: "UnPseudo"}
	r := &ServiceRegistry{
		resolve: func(_ context.Context, slug string) (*duckdb.PlayerDB, error) {
			resolved = append(resolved, slug)
			return pdb, nil
		},
		hiCapabilities: games.CapabilityMap{games.CapFilmReplayArtifact: games.CapSupported},
	}
	if _, err := r.ReplayGeneration(context.Background(), "UnPseudo"); err != nil {
		t.Fatal(err)
	}
	if len(resolved) != 1 || resolved[0] != "UnPseudo" {
		t.Fatalf("slug résolu = %v, attendu le slug du chemin tel quel", resolved)
	}
	svc := r.replayGenerationFor(pdb, "UnPseudo", queue, fakeGenProbeWire{})
	if _, created, err := svc.RequestGeneration(context.Background(), "halo_infinite", "0a0a0a0a"); err != nil || !created {
		t.Fatalf("RequestGeneration: created=%v err=%v", created, err)
	}
	if len(queue.got) != 1 {
		t.Fatalf("demandes = %d", len(queue.got))
	}
	d := queue.got[0]
	if d.XUID != "xuid-profil-de-la-page" || d.Gamertag != "UnPseudo" || d.PlayerSlug != "UnPseudo" || d.MatchID != "0a0a0a0a-1-2-3-4" {
		t.Fatalf("demande = %+v", d)
	}
}

func TestReplayGeneration_FabriqueRendLErreurDeResolution(t *testing.T) {
	boom := errors.New("profil inconnu")
	r := &ServiceRegistry{resolve: func(context.Context, string) (*duckdb.PlayerDB, error) { return nil, boom }}
	if _, err := r.ReplayGeneration(context.Background(), "x"); !errors.Is(err, boom) {
		t.Fatalf("err = %v", err)
	}
}

type fakeAuthStoreWire struct {
	users map[string]*auth.UserTokens
	err   error
}

func (s fakeAuthStoreWire) Load(xuid string) (*auth.UserTokens, error) {
	if s.err != nil {
		return nil, s.err
	}
	u, ok := s.users[xuid]
	if !ok {
		return nil, auth.ErrUserTokensNotFound
	}
	return u, nil
}
func (fakeAuthStoreWire) UpdateOAuthRefreshToken(string, string) error { return nil }
func (fakeAuthStoreWire) ClearReauthRequired(string) error             { return nil }

// La présence d'un jeton est une lecture du magasin pour CE xuid : refresh token non vide, rien d'autre.
func TestReplayTokenPresence(t *testing.T) {
	r := &ServiceRegistry{authStore: fakeAuthStoreWire{users: map[string]*auth.UserTokens{
		"avec":  {OAuthRefreshToken: "rt"},
		"vide":  {OAuthRefreshToken: "  "},
		"autre": {OAuthRefreshToken: "rt-autre"},
	}}}
	for xuid, want := range map[string]bool{"avec": true, "vide": false, "absent": false, "": false} {
		if got := r.replayTokenPresence(xuid); got != want {
			t.Errorf("replayTokenPresence(%q) = %v, attendu %v", xuid, got, want)
		}
	}
	if (&ServiceRegistry{}).replayTokenPresence("avec") {
		t.Error("sans magasin, aucun jeton")
	}
	if (&ServiceRegistry{authStore: fakeAuthStoreWire{err: errors.New("illisible")}}).replayTokenPresence("avec") {
		t.Error("un magasin illisible se lit « pas de jeton »")
	}
}
