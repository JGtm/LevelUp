//go:build integration

package replayartifacts

// derivations_raster_ancien_integration_test.go — UN FICHIER TACTIQUE D'UN ANCIEN FORMAT SE
// REPROJETTE PAR LE CHEMIN NORMAL, SANS LE FILM.
//
// Le fichier tactique d'un match est une dérivation de son rejeu rangé (`projeterRastersTactiques`,
// quatrième famille de `Deriver`). Un format qui monte rend le fichier du parc inexploitable
// (`domain.SidecarRasterCourant`) : le match sort de la Tactique. Le seul mécanisme qui refait ce
// fichier sans recuire le film est la reprise des dérivations, jugée sur la révision de la marque
// (`replaybuild.DerivationsRev`). Ce test fige la chaîne entière : un rejeu présent, son fichier au
// format précédent et une marque de l'ancienne révision sont jugés « à refaire », le rattrapage
// reprojette le fichier au format courant, puis le rejeu est jugé à jour.

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/replaybuild"
)

// revisionDesDerivationsAvantLeFormat8 est la révision que portent les marques posées par la
// version livrée avant le format 8 du fichier tactique : le test la pose telle quelle, sans la
// déduire de la constante courante, pour que ne pas monter celle-ci rougisse le test.
const revisionDesDerivationsAvantLeFormat8 = "derivations-2026-09-06"

// formatTactiqueAvantLe8 : le format du fichier tactique du parc livré avant le format 8.
const formatTactiqueAvantLe8 = 7

// poserFichierTactiqueAncien dépose le fichier tactique d'un match à un ancien format, tel que le
// parc le porte.
func poserFichierTactiqueAncien(t *testing.T, d Deps, matchID string) string {
	t.Helper()
	chemin := titlePkg.NewPathResolver(d.RepoRoot).TacticalRasterPath(d.TitleSlug, matchID)
	ancien := domain.TacticalRasterSidecar{
		SchemaVersion:    formatTactiqueAvantLe8,
		MatchID:          matchID,
		ShortID:          titlePkg.FilmShortMatchID(matchID),
		PasM:             domain.TacticalRasterPasM,
		PasEchantillonMs: domain.TacticalRasterPasEchantillonMs,
	}
	if err := EcrireSidecarRaster(chemin, ancien); err != nil {
		t.Fatalf("écriture du fichier tactique ancien: %v", err)
	}
	return chemin
}

// poserMarqueDeLAncienneRevision marque l'artefact comme dérivé à l'ancienne révision, sur sa
// taille d'aujourd'hui : seule la révision diffère de la marque courante.
func poserMarqueDeLAncienneRevision(t *testing.T, artefact string) {
	t.Helper()
	st, err := os.Stat(artefact)
	if err != nil {
		t.Fatalf("stat de l'artefact: %v", err)
	}
	blob, err := json.Marshal(replaybuild.DerivationsMark{
		Rev: revisionDesDerivationsAvantLeFormat8, ArtifactBytes: int(st.Size()), At: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("marque ancienne: %v", err)
	}
	if err := os.WriteFile(replaybuild.DerivationsMarkPath(artefact), blob, 0o600); err != nil {
		t.Fatalf("écriture de la marque ancienne: %v", err)
	}
}

// lireFichierTactique relit le fichier tactique tel que le service le lit.
func lireFichierTactique(t *testing.T, chemin string) domain.TacticalRasterSidecar {
	t.Helper()
	raw, err := os.ReadFile(chemin) //nolint:gosec // chemin posé par le test
	if err != nil {
		t.Fatalf("lecture du fichier tactique: %v", err)
	}
	var sc domain.TacticalRasterSidecar
	if err := json.Unmarshal(raw, &sc); err != nil {
		t.Fatalf("lecture du fichier tactique: %v", err)
	}
	return sc
}

// TestRattrapage_ReprojetteUnFichierTactiqueAncien_SansLeFilm : un rejeu présent, un fichier
// tactique au format 7 et la marque de l'ancienne révision. Aucun client de film n'est câblé : seul
// l'artefact rangé sert.
func TestRattrapage_ReprojetteUnFichierTactiqueAncien_SansLeFilm(t *testing.T) {
	const matchID = "0000ae05"
	_, d, _ := monterLeLot(t, matchID)
	artefact := cheminDuLot(d, matchID)
	fichier := poserFichierTactiqueAncien(t, d, matchID)
	poserMarqueDeLAncienneRevision(t, artefact)

	if domain.SidecarRasterCourant(&domain.TacticalRasterSidecar{
		SchemaVersion: formatTactiqueAvantLe8, PasM: domain.TacticalRasterPasM,
		PasEchantillonMs: domain.TacticalRasterPasEchantillonMs,
	}) {
		t.Fatalf("le format %d est encore tenu pour courant : le test ne prouve rien", formatTactiqueAvantLe8)
	}
	if replaybuild.DerivationsUpToDate(artefact) {
		t.Fatalf("rejeu jugé à jour avec la marque de l'ancienne révision %q : le fichier tactique "+
			"au format %d ne serait jamais reprojeté (le match sortirait de toute la Tactique)",
			revisionDesDerivationsAvantLeFormat8, formatTactiqueAvantLe8)
	}
	if d.Placement != "" || d.Fetcher != nil {
		t.Fatal("le test câble un lieu de construction ou un client de film : il ne prouverait plus « sans le film »")
	}

	rattraperDerivations(context.Background(), d)

	sc := lireFichierTactique(t, fichier)
	if !domain.SidecarRasterCourant(&sc) {
		t.Errorf("fichier tactique au format %d après le rattrapage, attendu le format %d",
			sc.SchemaVersion, domain.TacticalRasterSchemaVersion)
	}
	if !replaybuild.DerivationsUpToDate(artefact) {
		t.Error("rejeu non jugé à jour après le rattrapage : les mêmes artefacts reviendraient à chaque cycle")
	}
}
