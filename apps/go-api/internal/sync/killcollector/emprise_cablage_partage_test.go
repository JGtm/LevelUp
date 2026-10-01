//go:build integration || research

package killcollector

// emprise_cablage_partage_test.go — le cablage du collecteur et le rejeu de sa passe, PARTAGES
// par les instruments d emprise (`emprise_v0_passe_research_test.go`,
// `emprise_v1_temoin_research_test.go`, tag `research`) et la garde d integration du placement
// des vies (`placement_des_vies_integration_test.go`, tag `integration`). Extraits a la fusion de
// J12 (2026-10-01) : une garde reelle ne depend jamais d un fichier tague `research` (regle du
// J12.7 bis, `archlint/research_tag_test.go`).

import (
	"context"
	"database/sql"
	"testing"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/halo_infinite/film/decfilm"
	"levelup/go-api/internal/persist"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/sync/haloclient"
	"levelup/go-api/internal/testutil"
)

// v0Env porte les entrees du protocole, lues dans l'environnement.
type v0Env struct {
	films        []string
	dir, db, cac string
	tours        int
}

// v0Lecteur adapte un handle au port de lecture partagee du resolveur de carte.
type v0Lecteur struct{ db *sql.DB }

func (l v0Lecteur) Get(context.Context) (*sql.DB, func(), error) { return l.db, func() {}, nil }

// v0Collecteur cable le collecteur comme la production : les trois capabilities du titre
// (`capabilities.toml` : film.kill_source, film.weapon_shots, film.kill_positions), le cache
// disque comme source de films, le roster par defaut (jointure par match) sur la copie, la
// capture de positions par `CaptureDepuisCatalogue` et le resolveur de carte de production.
// `ConfigureFilmAccuracy` n'est PAS appele : aucun appelant de production ne l'appelle.
func v0Collecteur(t *testing.T, e v0Env, lecture, ecriture *sql.DB) *KillSourceCollector {
	t.Helper()
	repoRoot, err := testutil.RepoRoot()
	if err != nil {
		t.Fatalf("racine du depot : %v", err)
	}
	deps, err := CaptureDepuisCatalogue(repoRoot, title.DefaultSlug,
		duckdb.NewReplayMapRepo(v0Lecteur{lecture}, nil))
	if err != nil {
		t.Fatalf("capture : %v", err)
	}
	caps := games.CapabilityMap{
		games.CapFilmKillSource:    games.CapSupported,
		games.CapFilmWeaponShots:   games.CapSupported,
		games.CapFilmKillPositions: games.CapSupported,
	}
	return NewKillSourceCollector(NewLocalCacheFilms(haloclient.NewLocalFilmCache(e.cac)),
		NewSharedRoster(lecture), func(context.Context) (*sql.DB, func(), error) {
			return ecriture, func() {}, nil
		}, caps, 0).AvecCapture(deps)
}

// v1Passe : ce que la passe du collecteur a en main quand le placement se calcule.
type v1Passe struct {
	mat      materiauDIsolement
	ids      MatchIdentities
	fusionne persist.KillSourceBatch
}

// v1RejouerLaPasse rejoue `collect` jusqu'aux faits d'isolement, par les fonctions de production,
// sans rien écrire.
func v1RejouerLaPasse(t *testing.T, ctx context.Context, col *KillSourceCollector, lecture *sql.DB,
	id string) v1Passe {
	t.Helper()
	chunks, _, err := FilmChunksForMatch(ctx, col.client, id)
	if err != nil {
		t.Fatalf("%s : chunks : %v", id, err)
	}
	film, err := FilmOf(chunks)
	if err != nil {
		t.Fatalf("%s : film : %v", id, err)
	}
	opts := decfilm.DefaultOptions()
	carte, err := col.carteDuMatch(ctx, id)
	if err != nil {
		t.Fatalf("%s : carte : %v", id, err)
	}
	opts.Carte = carte
	res, err := decfilm.Decode(ctx, id, film, &opts)
	if err != nil {
		t.Fatalf("%s : decodage : %v", id, err)
	}
	ids, err := col.roster.IdentitiesForMatch(ctx, id)
	if err != nil {
		t.Fatalf("%s : roster : %v", id, err)
	}
	passeFilm := BuildKillSourceBatch(ctx, id, res, ids)
	base, err := persist.CreditBaseForMatch(ctx, lecture, id)
	if err != nil {
		t.Fatalf("%s : base credit : %v", id, err)
	}
	fusionne, _, err := persist.MergeCreditAndFilm(base, passeFilm)
	if err != nil {
		t.Fatalf("%s : fusion : %v", id, err)
	}
	entry, err := col.resolveMapBounds(ctx, id)
	if err != nil {
		t.Fatalf("%s : carte : %v", id, err)
	}
	_, mat, err := buildPositionRows(ctx, film, res, entry, ids, killRefsFromDeaths(passeFilm.Deaths), id)
	if err != nil {
		t.Fatalf("%s : positions : %v", id, err)
	}
	return v1Passe{mat: mat, ids: ids, fusionne: fusionne}
}
