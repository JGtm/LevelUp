//go:build integration

package replayartifacts

// raster_integration_test.go — LE CABLAGE DU SIDECAR DANS `Run`, EXECUTE POUR DE VRAI.
//
// # POURQUOI CE FICHIER EXISTE (constat C5 de la revue)
//
// La ligne qui appelle `projeterRastersTactiques` dans `Run` n'etait traversee par AUCUN
// test : la supprimer laissait toute la suite verte, et plus aucun sidecar ne serait ne a
// la cuisson. Les tests unitaires de raster_test.go eprouvent la PROJECTION ; ils
// n'eprouvent pas qu'on l'appelle.
//
// # CE QUE LA PASSE FAIT ICI, ET CE QU'ELLE NE FAIT PAS
//
// AUCUN FILM N'EST DECODE. `Deps.BuildOne` est le seam prevu pour cela (cf. buildone.go) :
// le test rend les OCTETS d'un document forge a la main, exactement comme le ferait
// l'enfant decodeur, et le parent les range par `StoreArtifact` — donc tout le chemin
// « artefact range -> projection -> sidecar sur disque » est joue, sans une seconde de
// decodage ni un processus enfant.
//
// TOUT VIT SOUS UN REPERTOIRE TEMPORAIRE : la racine porte les manifestes du titre (pour la
// porte de capability) et un catalogue de bornes MINIMAL (pour la sonde de titre de `Run`).
// Rien n'est ecrit sous le `data/` du depot.

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"levelup/go-api/internal/analysis/replay"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/replaybuild"
	"levelup/go-api/internal/sync/haloclient"
)

// fetcherRaster : un client film minimal — le pont disque doit trouver de quoi persister,
// il ne decode rien.
type fetcherRaster struct{}

func (fetcherRaster) GetFilmChunks(context.Context, string) ([]haloclient.FilmChunk, bool, error) {
	return []haloclient.FilmChunk{{Index: 0, ChunkType: 2, Data: []byte("x")}}, true, nil
}

// racineCuisson prepare une racine temporaire complete : manifestes du titre + catalogue de
// bornes minimal (la sonde `replaybuild.NewBuilder` de `Run` s'arrete sans lui).
func racineCuisson(t *testing.T) string {
	t.Helper()
	racine := racineAvecTitres(t, titlePkg.DefaultSlug)
	ref := filepath.Join(racine, "data", "titles", titlePkg.DefaultSlug, "reference")
	if err := os.MkdirAll(ref, 0o755); err != nil {
		t.Fatalf("mkdir reference: %v", err)
	}
	if err := os.WriteFile(filepath.Join(ref, "map_quant_bounds.json"),
		[]byte(`{"schemaVersion":1,"maps":{}}`), 0o644); err != nil {
		t.Fatalf("write map_quant_bounds: %v", err)
	}
	return racine
}

// documentCuit : les octets qu'un enfant decodeur rendrait — un joueur immobile 2 s.
func documentCuit(t *testing.T, matchID string) []byte {
	t.Helper()
	doc := replay.ReplayDocument{
		SchemaVersion:   replay.SchemaVersion,
		MatchID:         matchID,
		FrameIntervalMS: 100,
		FrameCount:      21,
		Tracks: []replay.Track{{
			Slot: 1, Team: -1, XUID: "111", StartFrame: 0, EndFrame: 20,
			Points: []replay.Point{{T: 0, X: 0.25, Y: 0.25}, {T: 20, X: 0.25, Y: 0.25}},
		}},
	}
	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("marshal document: %v", err)
	}
	return raw
}

// depsCuisson : le cablage LOCAL complet, sans base d'ecriture (le sidecar n'en veut pas).
func depsCuisson(t *testing.T, racine string, db *sql.DB, build BuildOneFunc) Deps {
	t.Helper()
	return Deps{
		Gamertag:  "testeur",
		RepoRoot:  racine,
		TitleSlug: titlePkg.DefaultSlug,
		CacheRoot: t.TempDir(),
		Placement: replaybuild.PlacementLocal,
		Fetcher:   fetcherRaster{},
		WithRead:  func(_ context.Context, _ string, fn func(*sql.DB)) { fn(db) },
		BuildOne:  build,
	}
}

// TestRun_DeposeLeSidecarDeRaster — LE CABLAGE, DE BOUT EN BOUT.
//
// Supprimer l'appel a `projeterRastersTactiques` dans `Run` doit faire tomber ce test :
// c'est la seule chose qui garantit qu'un sidecar NAIT a la cuisson.
func TestRun_DeposeLeSidecarDeRaster(t *testing.T) {
	db := baseRegistre(t)
	racine := racineCuisson(t)
	const matchID = "aaaaaaaa-1111-4000-8000-000000000000"
	inscrireAuRegistre(t, db, matchID, time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC), 0)

	d := depsCuisson(t, racine, db, func(_ context.Context, req BuildOneRequest) (BuildOneResult, error) {
		return BuildOneResult{Blob: documentCuit(t, req.MatchID)}, nil
	})
	Run(context.Background(), d, []string{matchID})

	sidecar := titlePkg.NewPathResolver(racine).TacticalRasterPath(titlePkg.DefaultSlug, matchID)
	raw, err := os.ReadFile(sidecar)
	if err != nil {
		t.Fatalf("aucun sidecar de raster depose par la cuisson (%s) : %v — le cablage de "+
			"`Run` n'est traverse par rien d'autre", sidecar, err)
	}
	var sc struct {
		SchemaVersion int    `json:"schema_version"`
		MatchID       string `json:"match_id"`
		Joueurs       []struct {
			XUID     string `json:"xuid"`
			Cellules []struct {
				Echantillons int `json:"echantillons"`
			} `json:"cellules"`
		} `json:"joueurs"`
	}
	if err := json.Unmarshal(raw, &sc); err != nil {
		t.Fatalf("sidecar illisible: %v", err)
	}
	if sc.MatchID != matchID {
		t.Fatalf("match_id = %q, attendu %q", sc.MatchID, matchID)
	}
	if len(sc.Joueurs) != 1 || len(sc.Joueurs[0].Cellules) != 1 ||
		sc.Joueurs[0].Cellules[0].Echantillons != 8 {
		t.Fatalf("occupation = %+v, attendu 8 echantillons (2 s / 250 ms)", sc.Joueurs)
	}
	// LE SIDECAR EST DANS SON SOUS-DOSSIER, jamais au premier niveau du dossier
	// d'artefacts : sinon `AvailableSet` le compterait pour un match et la purge le
	// supprimerait.
	if filepath.Base(filepath.Dir(sidecar)) != "rasters" {
		t.Fatalf("sidecar hors du sous-dossier rasters : %s", sidecar)
	}
}

// TestRun_ProjectionEnEchecNArretePasLeCycle — un artefact range mais illisible compte en
// echec et laisse le reste du cycle intact. Le sidecar est best-effort : il ne doit jamais
// casser une synchronisation, mais il ne doit jamais se taire non plus.
func TestRun_ProjectionEnEchecNArretePasLeCycle(t *testing.T) {
	db := baseRegistre(t)
	racine := racineCuisson(t)
	const sain = "cccccccc-3333-4000-8000-000000000000"
	const casse = "dddddddd-4444-4000-8000-000000000000"
	inscrireAuRegistre(t, db, sain, time.Date(2026, 6, 2, 12, 0, 0, 0, time.UTC), 0)
	inscrireAuRegistre(t, db, casse, time.Date(2026, 6, 2, 13, 0, 0, 0, time.UTC), 0)

	// L'enfant rend un document valide pour le premier match, des octets ILLISIBLES pour le
	// second : `StoreArtifact` refuse ces derniers, donc rien n'est range — c'est la
	// projection du lot qui doit rester saine pour l'autre match.
	d := depsCuisson(t, racine, db, func(_ context.Context, req BuildOneRequest) (BuildOneResult, error) {
		return BuildOneResult{Blob: documentCuit(t, req.MatchID)}, nil
	})
	Run(context.Background(), d, []string{sain, casse})

	avantEchecs := observability.LoadCounterT("", CompteurRastersEchecs)
	avantEcrits := observability.LoadCounterT("", CompteurRastersEcrits)
	pr := titlePkg.NewPathResolver(racine)
	// Un artefact CORROMPU APRES RANGEMENT : la situation reelle d'un fichier abime sur
	// disque. La projection doit le compter en echec ET traiter le suivant.
	if err := os.WriteFile(pr.ReplayArtifactPath(titlePkg.DefaultSlug, casse),
		[]byte(`{"schemaVersion":`), 0o644); err != nil {
		t.Fatalf("corrompre l'artefact: %v", err)
	}
	projeterRastersTactiques(context.Background(), d, []artefactCuit{
		{matchID: casse, path: pr.ReplayArtifactPath(titlePkg.DefaultSlug, casse)},
		{matchID: sain, path: pr.ReplayArtifactPath(titlePkg.DefaultSlug, sain)},
	})

	if apres := observability.LoadCounterT("", CompteurRastersEchecs); apres != avantEchecs+1 {
		t.Fatalf("compteur d'echecs = %d (avant %d), attendu +1 : un artefact illisible doit "+
			"se compter, jamais se taire", apres, avantEchecs)
	}
	if apres := observability.LoadCounterT("", CompteurRastersEcrits); apres != avantEcrits+1 {
		t.Fatalf("compteur d'ecrits = %d (avant %d), attendu +1 : l'echec d'un match ne doit "+
			"pas empecher les suivants", apres, avantEcrits)
	}
}
