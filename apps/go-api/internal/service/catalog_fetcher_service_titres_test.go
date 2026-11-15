package service

// catalog_fetcher_service_titres_test.go — le drain du catalogue d'un titre sans adapter de catalogue
// (Halo 5) échoue proprement, et la classification « classée » est celle de l'adapter du titre : le
// service n'applique la référence d'aucun titre.

import (
	"context"
	"errors"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/canonical"
)

// referenceClasseeHalo : un asset_id de la référence des playlists classées d'Halo Infinite.
const referenceClasseeHalo = "edfef3ac-9cbe-4fa2-b949-8f29deafd483"

// ecrivainCatalogueFaux enregistre ce que le service persiste.
type ecrivainCatalogueFaux struct {
	pending     []domain.CatalogQueueEntry
	playlists   []canonical.CanonicalPlaylist
	isRanked    []bool
	experiences []string
}

func (w *ecrivainCatalogueFaux) SelectPending(context.Context, string) ([]domain.CatalogQueueEntry, error) {
	return w.pending, nil
}

func (w *ecrivainCatalogueFaux) UpsertPlaylist(_ context.Context, _ string, pl canonical.CanonicalPlaylist, isRanked bool, experience string) error {
	w.playlists = append(w.playlists, pl)
	w.isRanked = append(w.isRanked, isRanked)
	w.experiences = append(w.experiences, experience)
	return nil
}

func (w *ecrivainCatalogueFaux) UpsertPair(context.Context, string, canonical.CanonicalPair) error {
	return nil
}
func (w *ecrivainCatalogueFaux) UpsertMap(context.Context, string, canonical.CanonicalMap) error {
	return nil
}
func (w *ecrivainCatalogueFaux) UpsertGameVariant(context.Context, string, canonical.CanonicalGameVariant) error {
	return nil
}

// adapterPlaylistFixe rend toujours la même playlist, telle que l'adapter de SON titre l'a classée.
type adapterPlaylistFixe struct {
	games.TitleCatalogAdapter
	pl canonical.CanonicalPlaylist
}

func (a adapterPlaylistFixe) FetchPlaylist(context.Context, string, string) (canonical.CanonicalPlaylist, error) {
	return a.pl, nil
}

// resolveurCataloguePar rend l'adapter des titres de la table, games.ErrTitleNotResolved pour les autres.
type resolveurCataloguePar map[string]games.TitleCatalogAdapter

func (r resolveurCataloguePar) Data(string) (games.TitleDataAdapter, error) { return nil, nil }
func (r resolveurCataloguePar) Semantic(string) (games.TitleSemanticAdapter, error) {
	return nil, nil
}
func (r resolveurCataloguePar) AssetURL(string) (games.TitleAssetURLAdapter, error) {
	return nil, nil
}
func (r resolveurCataloguePar) Catalog(slug string) (games.TitleCatalogAdapter, error) {
	if a, ok := r[slug]; ok {
		return a, nil
	}
	return nil, games.ErrTitleNotResolved
}
func (r resolveurCataloguePar) DefaultSlug() string { return "halo_infinite" }

// TestCatalogFetcherService_Drain_Halo5SansAdapter : Halo 5 n'a pas d'adapter de catalogue. Le drain rend
// l'erreur du résolveur, sans lire la file ni rien écrire.
func TestCatalogFetcherService_Drain_Halo5SansAdapter(t *testing.T) {
	w := &ecrivainCatalogueFaux{pending: []domain.CatalogQueueEntry{{AssetType: games.AssetKindPlaylist, AssetID: "pl-1"}}}
	svc := NewCatalogFetcherService(w, resolveurCataloguePar{"halo_infinite": adapterPlaylistFixe{}})

	_, err := svc.Drain(context.Background(), "halo_5")
	if !errors.Is(err, games.ErrTitleNotResolved) {
		t.Fatalf("err = %v, attendu games.ErrTitleNotResolved", err)
	}
	if len(w.playlists) != 0 {
		t.Errorf("%d playlist(s) écrite(s), attendu aucune", len(w.playlists))
	}
}

// TestCatalogFetcherService_Drain_PersisteLaClassificationDeLAdapter : un titre dont l'adapter range une
// playlist en `social` la voit persistée `social`, même si son identifiant figure dans la référence des
// playlists classées d'Halo Infinite : la référence est affaire de l'adapter d'Halo Infinite.
func TestCatalogFetcherService_Drain_PersisteLaClassificationDeLAdapter(t *testing.T) {
	w := &ecrivainCatalogueFaux{pending: []domain.CatalogQueueEntry{{AssetType: games.AssetKindPlaylist, AssetID: referenceClasseeHalo}}}
	adapter := adapterPlaylistFixe{pl: canonical.CanonicalPlaylist{
		AssetID: referenceClasseeHalo, Experience: canonical.ExperienceSocial, IsRanked: false,
	}}
	svc := NewCatalogFetcherService(w, resolveurCataloguePar{"halo_5": adapter})

	res, err := svc.Drain(context.Background(), "halo_5")
	if err != nil {
		t.Fatalf("Drain : %v", err)
	}
	if res.Playlists != 1 || len(w.playlists) != 1 {
		t.Fatalf("playlists traitées %d, écrites %d ; attendu 1 et 1", res.Playlists, len(w.playlists))
	}
	if w.isRanked[0] || w.experiences[0] != string(canonical.ExperienceSocial) {
		t.Errorf("persistée isRanked=%v experience=%q ; attendu false et %q (la classification de l'adapter)",
			w.isRanked[0], w.experiences[0], canonical.ExperienceSocial)
	}
}
