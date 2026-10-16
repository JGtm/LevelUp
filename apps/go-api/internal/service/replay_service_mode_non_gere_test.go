package service

// replay_service_mode_non_gere_test.go — LE REJEU NE SERT PAS UN MODE QU'IL NE GERE PAS
// (Firefight, decision du 2026-10-15), MEME SI UN ARTEFACT TRAINE DANS LE CACHE.

import (
	"context"
	"errors"
	"os"
	"testing"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/port"
)

const (
	idPvE = "edc5daf6-0df8-4eb0-b578-b8eb892ce184"
	idPvP = "39910eb1-8c01-4e26-bc60-965cb2dcc789"
)

// cartesAvecModes : un ReplayMapNameRepo qui porte AUSSI la capacite ReplayModeGate.
type cartesAvecModes struct {
	nonGeres map[string]bool // cle : match_id complet
	err      error
}

func (cartesAvecModes) MapKeysForMatch(context.Context, string) (port.MatchMapKeys, error) {
	return port.MatchMapKeys{}, port.ErrMatchMapUnknown
}

func (cartesAvecModes) MapKeysForMap(context.Context, string) (port.MatchMapKeys, error) {
	return port.MatchMapKeys{}, port.ErrMatchMapUnknown
}

func (c cartesAvecModes) ModeNonGere(_ context.Context, matchID string) (bool, error) {
	return c.nonGeres[matchID], c.err
}

func (c cartesAvecModes) ModesNonGeresShortIDs(context.Context) (map[string]struct{}, error) {
	out := map[string]struct{}{}
	for id, v := range c.nonGeres {
		if v {
			out[title.FilmShortMatchID(id)] = struct{}{}
		}
	}
	return out, c.err
}

// serviceAvecDeuxArtefacts pose un artefact PvP et un artefact PvE sous la meme racine.
func serviceAvecDeuxArtefacts(t *testing.T, maps port.ReplayMapNameRepo) port.ReplayService {
	t.Helper()
	root := artefactAt(t, title.DefaultSlug, idPvP, `{"schemaVersion":1}`)
	// Le second artefact est pose a la main sous la meme racine.
	for _, id := range []string{idPvE} {
		root2 := artefactAt(t, title.DefaultSlug, id, `{"schemaVersion":1}`)
		copierArtefact(t, root2, root, id)
	}
	return NewReplayService(title.DefaultSlug, root, maps)
}

func TestReplayService_ModeNonGere_IndisponibleMemeAvecArtefact(t *testing.T) {
	maps := cartesAvecModes{nonGeres: map[string]bool{idPvE: true}}
	svc := serviceAvecDeuxArtefacts(t, maps)
	ctx := context.Background()

	if svc.IsAvailable(ctx, idPvE) {
		t.Error("IsAvailable : un match Firefight ne doit pas etre annonce disponible")
	}
	if !svc.IsAvailable(ctx, idPvP) {
		t.Error("IsAvailable : un match gere garde son rejeu")
	}
	set, err := svc.AvailableSet(ctx)
	if err != nil {
		t.Fatalf("AvailableSet : %v", err)
	}
	if set.Has(idPvE) || !set.Has(idPvP) {
		t.Errorf("AvailableSet = %v : le Firefight doit en sortir, le match gere y rester", set)
	}
	// MEME CONTRAT QUE L'ABSENCE D'ARTEFACT : la sentinelle (404 cote handler), pas une erreur.
	if _, err := svc.GetReplay(ctx, idPvE); !errors.Is(err, port.ErrReplayNotAvailable) {
		t.Errorf("GetReplay Firefight : %v, attendu ErrReplayNotAvailable", err)
	}
}

// TestReplayService_ModeNonGere_RegistreIlisibleNeRetireRien — fail-open : une panne de lecture
// du registre ne fait pas disparaitre les rejeux des modes geres.
func TestReplayService_ModeNonGere_RegistreIlisibleNeRetireRien(t *testing.T) {
	svc := serviceAvecDeuxArtefacts(t, cartesAvecModes{err: errors.New("registre indisponible")})
	ctx := context.Background()
	if !svc.IsAvailable(ctx, idPvP) || !svc.IsAvailable(ctx, idPvE) {
		t.Error("registre illisible : aucun filtrage attendu")
	}
	set, err := svc.AvailableSet(ctx)
	if err != nil || len(set) != 2 {
		t.Errorf("AvailableSet = %v, %v : attendu les deux artefacts", set, err)
	}
}

// copierArtefact recopie l'artefact d'un match d'une racine de depot a une autre.
func copierArtefact(t *testing.T, de, vers, matchID string) {
	t.Helper()
	src := title.NewPathResolver(de).ReplayArtifactPath(title.DefaultSlug, matchID)
	dst := title.NewPathResolver(vers).ReplayArtifactPath(title.DefaultSlug, matchID)
	raw, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, raw, 0o644); err != nil {
		t.Fatal(err)
	}
}
