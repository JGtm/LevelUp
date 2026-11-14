package service

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"levelup/go-api/internal/port"
)

// fauxPlaylistsSeen rend une liste fixe de playlists.
type fauxPlaylistsSeen struct {
	playlists []port.PlaylistRef
	err       error
}

func (f fauxPlaylistsSeen) PlaylistsSeenInMatches(context.Context, string) ([]port.PlaylistRef, error) {
	return f.playlists, f.err
}

// fauxChildStore retient ce qu'on lui écrit.
type fauxChildStore struct {
	ensureErr  error
	enqueueErr map[string]error
	enqueued   []string
	weights    []string
}

func (f *fauxChildStore) EnsureSchema(context.Context) error { return f.ensureErr }

func (f *fauxChildStore) EnqueuePair(_ context.Context, slug, pairID, versionID string) error {
	if err := f.enqueueErr[pairID]; err != nil {
		return err
	}
	f.enqueued = append(f.enqueued, slug+"|"+pairID+"|"+versionID)
	return nil
}

func (f *fauxChildStore) RecordWeight(_ context.Context, slug, playlistID, pairID string, _ float64) error {
	f.weights = append(f.weights, slug+"|"+playlistID+"|"+pairID)
	return errors.New("poids ignorés : best-effort")
}

func fetcherDe(children map[string][]port.PlaylistChild, failing ...string) port.PlaylistConfigFetcher {
	return port.PlaylistConfigFetcherFunc(func(_ context.Context, p port.PlaylistRef) ([]port.PlaylistChild, error) {
		for _, f := range failing {
			if f == p.ID {
				return nil, errors.New("429")
			}
		}
		return children[p.ID], nil
	})
}

func depsDe(seen fauxPlaylistsSeen, store *fauxChildStore, fetcher port.PlaylistConfigFetcher, trace *[]string) PlaylistExpansionDeps {
	return PlaylistExpansionDeps{
		Playlists: seen,
		OpenChildren: func(context.Context, string) (port.PlaylistChildStore, func(), error) {
			*trace = append(*trace, "ouvre metadata")
			return store, func() { *trace = append(*trace, "rend metadata") }, nil
		},
		OpenFetcher: func(context.Context, string) (port.PlaylistConfigFetcher, error) {
			*trace = append(*trace, "jetons")
			return fetcher, nil
		},
	}
}

// TestPlaylistExpansion_EnfileLesEnfantsEtIgnoreLesPoids : un fetch raté n'interrompt pas les autres, un couple
// sans identifiant est ignoré, un enfilage en échec n'est pas compté, et l'échec d'un poids est sans effet.
func TestPlaylistExpansion_EnfileLesEnfantsEtIgnoreLesPoids(t *testing.T) {
	store := &fauxChildStore{enqueueErr: map[string]error{"p-ko": errors.New("insert")}}
	var trace []string
	svc := NewPlaylistExpansionService(depsDe(
		fauxPlaylistsSeen{playlists: []port.PlaylistRef{{ID: "pl-1", Version: "v1"}, {ID: "pl-2", Version: "v2"}, {ID: "pl-3", Version: "v3"}}},
		store,
		fetcherDe(map[string][]port.PlaylistChild{
			"pl-1": {{PairAssetID: "p-a", VersionID: "va", Weight: 2}, {PairAssetID: ""}, {PairAssetID: "p-ko"}},
			"pl-3": {{PairAssetID: "p-b", VersionID: "vb", Weight: 1}},
		}, "pl-2"),
		&trace))

	n, err := svc.ExpandPlaylistChildren(context.Background(), "halo_infinite")
	if err != nil || n != 2 {
		t.Fatalf("enfilés = %d, %v ; attendu 2", n, err)
	}
	if got := strings.Join(store.enqueued, ","); got != "halo_infinite|p-a|va,halo_infinite|p-b|vb" {
		t.Errorf("enfilés = %s", got)
	}
	if got := strings.Join(store.weights, ","); got != "halo_infinite|pl-1|p-a,halo_infinite|pl-3|p-b" {
		t.Errorf("poids inscrits = %s (le couple dont l'enfilage échoue n'a pas de poids)", got)
	}
	if got := strings.Join(trace, ","); got != "ouvre metadata,jetons,rend metadata" {
		t.Errorf("ordre = %s : metadata ouverte avant les jetons, rendue en sortant", got)
	}
}

// TestPlaylistExpansion_ArretAvantLeReseau : une lecture des playlists en échec, metadata indisponible ou un
// schéma impossible à créer arrêtent l'expansion AVANT la prise de jetons.
func TestPlaylistExpansion_ArretAvantLeReseau(t *testing.T) {
	boom := errors.New("boom")
	cas := map[string]func() (PlaylistExpansionDeps, *[]string){
		"playlists illisibles": func() (PlaylistExpansionDeps, *[]string) {
			var tr []string
			return depsDe(fauxPlaylistsSeen{err: boom}, &fauxChildStore{}, fetcherDe(nil), &tr), &tr
		},
		"schéma impossible": func() (PlaylistExpansionDeps, *[]string) {
			var tr []string
			return depsDe(fauxPlaylistsSeen{}, &fauxChildStore{ensureErr: boom}, fetcherDe(nil), &tr), &tr
		},
		"metadata indisponible": func() (PlaylistExpansionDeps, *[]string) {
			var tr []string
			d := depsDe(fauxPlaylistsSeen{}, &fauxChildStore{}, fetcherDe(nil), &tr)
			d.OpenChildren = func(context.Context, string) (port.PlaylistChildStore, func(), error) { return nil, nil, boom }
			return d, &tr
		},
	}
	for nom, mk := range cas {
		deps, trace := mk()
		n, err := NewPlaylistExpansionService(deps).ExpandPlaylistChildren(context.Background(), "halo_infinite")
		if !errors.Is(err, boom) || n != 0 {
			t.Errorf("%s : %d, %v ; attendu 0 et l'erreur d'origine", nom, n, err)
		}
		for _, e := range *trace {
			if e == "jetons" {
				t.Errorf("%s : les jetons ont été pris malgré l'échec", nom)
			}
		}
	}
}

// TestPlaylistExpansion_ContexteAnnule : l'annulation en cours de boucle rend ce qui est déjà enfilé et l'erreur
// du contexte.
func TestPlaylistExpansion_ContexteAnnule(t *testing.T) {
	store := &fauxChildStore{}
	var trace []string
	ctx, cancel := context.WithCancel(context.Background())
	fetcher := port.PlaylistConfigFetcherFunc(func(_ context.Context, _ port.PlaylistRef) ([]port.PlaylistChild, error) {
		cancel() // annulé pendant la première playlist
		return []port.PlaylistChild{{PairAssetID: "p-a", VersionID: "va"}}, nil
	})
	svc := NewPlaylistExpansionService(depsDe(
		fauxPlaylistsSeen{playlists: []port.PlaylistRef{{ID: "pl-1", Version: "v1"}, {ID: "pl-2", Version: "v2"}}}, store, fetcher, &trace))
	n, err := svc.ExpandPlaylistChildren(ctx, "halo_infinite")
	if !errors.Is(err, context.Canceled) || n != 1 {
		t.Fatalf("%d, %v ; attendu 1 et context.Canceled", n, err)
	}
}

// Une passe dont des configurations de playlist échouent en 500 écrit UNE ligne de synthèse WARN qui
// dit la cause et le statut HTTP ; le détail par playlist reste en DEBUG.
func TestPlaylistExpansion_SyntheseDitLaCauseEtLeStatut(t *testing.T) {
	var buf strings.Builder
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})))
	t.Cleanup(func() { slog.SetDefault(prev) })

	var trace []string
	fetcher := port.PlaylistConfigFetcherFunc(func(context.Context, port.PlaylistRef) ([]port.PlaylistChild, error) {
		return nil, errors.New("PlaylistChildren: HTTP 500: https://exemple/playlist")
	})
	svc := NewPlaylistExpansionService(depsDe(
		fauxPlaylistsSeen{playlists: []port.PlaylistRef{{ID: "pl-1"}, {ID: "pl-2"}}},
		&fauxChildStore{}, fetcher, &trace))

	if _, err := svc.ExpandPlaylistChildren(context.Background(), "halo_infinite"); err != nil {
		t.Fatalf("ExpandPlaylistChildren: %v", err)
	}
	logs := buf.String()
	if n := strings.Count(logs, "level=WARN"); n != 1 {
		t.Fatalf("%d lignes WARN, attendu 1 :\n%s", n, logs)
	}
	for _, want := range []string{"catalog_expand: terminé avec échecs", "playlists_failed=2", "statuts_http=[500]", "HTTP 500"} {
		if !strings.Contains(logs, want) {
			t.Errorf("synthèse sans %q :\n%s", want, logs)
		}
	}
}
