package sync

import (
	"context"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/rankedplaylists"
)

// augmentStubClient implémente HaloClient via embedding (méthodes non utilisées
// = nil, paniqueraient si appelées). Seul GetPlaylistCsr est surchargé.
type augmentStubClient struct {
	HaloClient
	got  []string
	resp map[string]*PlayerPlaylistCSR
}

func (m *augmentStubClient) GetPlaylistCsr(_ context.Context, playlistID, _, _ string) (*PlayerPlaylistCSR, error) {
	m.got = append(m.got, playlistID)
	return m.resp[playlistID], nil
}

// TestAugmentWithActiveRankedCSRs vérifie que le fetch par-playlist complète les
// playlists classées actives manquantes (avec nom de la référence), ne re-fetch
// pas celles déjà présentes (player-level), et ignore celles sans réponse API
// (laissées à la synthèse catalogue-first "Non classé").
func TestAugmentWithActiveRankedCSRs(t *testing.T) {
	active := rankedplaylists.Active()
	if len(active) < 2 {
		t.Skip("référence < 2 playlists actives")
	}

	// pre : la 1ère active déjà fournie par le player-level.
	pre := []PlayerPlaylistCSR{{PlaylistID: active[0].AssetID, PlaylistName: "deja"}}
	// L'API ne renvoie un CSR que pour la 2e active ; nil (jamais jouée) pour le reste.
	stub := &augmentStubClient{resp: map[string]*PlayerPlaylistCSR{
		active[1].AssetID: {
			PlaylistID: active[1].AssetID,
			Current:    CSRRankSnapshot{Tier: "Gold", SubTier: 3, Value: 1234},
		},
	}}

	// activePlaylists nil → fallback rankedplaylists.Active() (la référence testée ici).
	out := AugmentWithActiveRankedCSRs(context.Background(), stub, "123", "CsrSeason13-1", pre, "en", nil)

	// La playlist déjà présente ne doit pas être interrogée.
	for _, id := range stub.got {
		if id == active[0].AssetID {
			t.Errorf("playlist déjà présente re-fetchée: %s", id)
		}
	}

	byID := make(map[string]PlayerPlaylistCSR, len(out))
	for _, c := range out {
		byID[c.PlaylistID] = c
	}
	if _, ok := byID[active[0].AssetID]; !ok {
		t.Error("entrée player-level perdue")
	}
	got2, ok := byID[active[1].AssetID]
	if !ok {
		t.Fatal("2e active non ajoutée")
	}
	if got2.PlaylistName != active[1].NameEN {
		t.Errorf("nom = %q, attendu référence %q", got2.PlaylistName, active[1].NameEN)
	}
	if got2.Current.Tier != "Gold" {
		t.Errorf("tier perdu après augment: %q", got2.Current.Tier)
	}
	// pre (1) + 1 ajout ; les actives sans réponse API ne sont PAS ajoutées.
	if len(out) != 2 {
		t.Errorf("len(out)=%d, attendu 2 (1 player-level + 1 ajout API)", len(out))
	}
}

// TestAugmentWithActiveRankedCSRs_UsesProvidedList vérifie que l'augment interroge la
// LISTE FOURNIE (playlists actives découvertes dynamiquement, A3) et non la référence
// statique — une playlist active hors référence est bien couverte.
func TestAugmentWithActiveRankedCSRs_UsesProvidedList(t *testing.T) {
	const dynID = "dyn-active-playlist-0001" // absente de rankedplaylists.Active()
	stub := &augmentStubClient{resp: map[string]*PlayerPlaylistCSR{
		dynID: {PlaylistID: dynID, Current: CSRRankSnapshot{Tier: "Onyx", Value: 1500}},
	}}
	provided := []rankedplaylists.Playlist{{AssetID: dynID, NameEN: "Dynamic"}}

	out := AugmentWithActiveRankedCSRs(context.Background(), stub, "123", "CsrSeason13-1", nil, "en", provided)

	if len(stub.got) != 1 || stub.got[0] != dynID {
		t.Fatalf("l'augment doit interroger la liste fournie (%q), got %v", dynID, stub.got)
	}
	if len(out) != 1 || out[0].PlaylistID != dynID {
		t.Fatalf("sortie = %+v, attendu 1 entrée pour %q", out, dynID)
	}
	if out[0].PlaylistName != "Dynamic" {
		t.Errorf("nom = %q, attendu 'Dynamic' (de la liste fournie)", out[0].PlaylistName)
	}
}
