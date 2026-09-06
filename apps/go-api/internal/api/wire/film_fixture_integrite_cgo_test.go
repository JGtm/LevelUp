//go:build cgo

// Package api — film_fixture_integrite_cgo_test.go : LE MINI-FILM VERSIONNÉ EST-IL CE QUE LE CDN
// SERT ?
//
// POURQUOI CE GARDE-RAIL EXISTE. Le fixture `testdata/film_e2e/c0a82e88` est le SEUL film que la
// CI décode réellement (`TestOuvrierReel_ConstruitEtLivre`). Il n'a de valeur que s'il reproduit
// ce que l'ouvrier reçoit en production : des morceaux servis par le CDN Azure, chacun sous UNE
// couche zlib, que `filmsource` décompresse une fois.
//
// IL NE LE REPRODUISAIT PAS. Généré le 2026-08-25 en compressant chaque fichier du cache local,
// il a hérité d'un cache HÉTÉROGÈNE : ses morceaux de jeu (1 à 6) y étaient stockés DÉJÀ
// décompressés — les compresser une fois était juste — mais ses morceaux 00 (registre ECS) et 07
// (pied) y étaient stockés compressés, et les compresser à leur tour leur a mis DEUX couches.
//
// CELA N'A RIEN CASSÉ PENDANT HUIT JOURS parce que `filmdec.ParseRegistryChunk` décompressait
// elle-même au besoin. Le 2026-09-02, `c17f4941f` (lot 1a de PLAN_CUISSON_PERF) lui a retiré cet
// inflate — décision juste, la décompression se fait une fois par film — et le registre du
// fixture est devenu VIDE : plus d'archétype biped 35, arme au sol 42, équipement 37, véhicule
// 40, objet d'objectif. Onze calques du rejeu sont tombés d'un coup SUR CE FILM, et la CI est
// restée verte parce que l'épreuve E2E n'assertait que la forme du document.
//
// Ce test fige donc la propriété qui manquait : UNE couche zlib par morceau, et un registre qui
// se lit et porte l'empreinte de référence. Il est DÉLIBÉRÉMENT autonome (il ne partage rien avec
// l'épreuve E2E, qui est derrière le tag `integration`) pour tourner dès que CGO est actif —
// c'est-à-dire bien plus souvent que la cuisson qu'il protège.
package wire

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/analysis/filmsource"
)

// filmPreuveChunks rend le dossier des morceaux du mini-film versionné, résolu PAR LE PAQUET.
func filmPreuveChunks(t *testing.T) string {
	t.Helper()
	_, ici, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("chemin du test introuvable")
	}
	return filepath.Join(filepath.Dir(ici), "testdata", "film_e2e", "c0a82e88", "chunks")
}

// filmPreuveMorceaux liste les fichiers de morceaux du fixture, triés par nom.
func filmPreuveMorceaux(t *testing.T) []string {
	t.Helper()
	fichiers, err := filepath.Glob(filepath.Join(filmPreuveChunks(t), "chunk_*.bin"))
	if err != nil || len(fichiers) == 0 {
		t.Fatalf("morceaux du fixture introuvables (%v) — ils doivent être VERSIONNÉS", err)
	}
	return fichiers
}

// TestFixtureFilmUneSeuleCoucheZlib : chaque morceau du fixture porte EXACTEMENT une couche zlib,
// comme le CDN les sert. Zéro couche serait un cache local recopié tel quel ; deux couches, le
// défaut de génération du 2026-08-25.
func TestFixtureFilmUneSeuleCoucheZlib(t *testing.T) {
	for _, chemin := range filmPreuveMorceaux(t) {
		nom := filepath.Base(chemin)
		brut, err := os.ReadFile(chemin)
		if err != nil {
			t.Fatalf("morceau %s illisible : %v", nom, err)
		}
		// Une couche : la décompression change la taille (les morceaux de ce film ne sont jamais
		// incompressibles — le moins compressible du fixture gagne encore un facteur 3).
		unePasse := filmsource.Inflate(brut)
		if len(unePasse) == len(brut) {
			t.Errorf("%s : AUCUNE couche zlib (%d octets) — le fixture doit porter les morceaux "+
				"tels que le CDN les sert, pas la copie décompressée du cache local", nom, len(brut))
			continue
		}
		// Pas deux : une seconde passe ne doit plus rien décompresser.
		deuxPasses := filmsource.Inflate(unePasse)
		if len(deuxPasses) != len(unePasse) {
			t.Errorf("%s : DEUX couches zlib (%d -> %d -> %d octets) — le décodage n'en pèle "+
				"qu'une, tout ce qui lit ce morceau lira du zlib au lieu de sa charge",
				nom, len(brut), len(unePasse), len(deuxPasses))
		}
	}
}

// TestFixtureFilmRegistreECSLisible : le morceau 00 décompressé EST un registre ECS, il porte
// l'empreinte du build de référence, et les quatre archétypes dont dépendent les calques du rejeu
// y sont. C'est l'assertion qui aurait arrêté `c17f4941f` sur ce fixture.
func TestFixtureFilmRegistreECSLisible(t *testing.T) {
	chemin := filepath.Join(filmPreuveChunks(t), "chunk_00.bin")
	brut, err := os.ReadFile(chemin)
	if err != nil {
		t.Fatalf("morceau du registre illisible : %v", err)
	}
	reg, err := filmdec.ParseRegistryChunk(filmsource.Inflate(brut))
	if err != nil {
		t.Fatalf("registre ECS du fixture illisible : %v", err)
	}
	if fp := filmdec.RegistryFingerprint(reg); fp != filmdec.KnownRegistryFingerprint {
		t.Fatalf("empreinte du registre = %d, attendu %d (le build de référence) — le décodage "+
			"tournerait sur une grammaire de composants qui n'est pas celle que la table décrit",
			fp, filmdec.KnownRegistryFingerprint)
	}
	// Les quatre archétypes que la cuisson interroge nommément. Leur absence est exactement ce
	// que le registre vide produisait, sous le message trompeur « archétype N absent du registre ».
	for _, ti := range []int{35, 37, 40, 42} {
		arch, ok := reg.Archetype(ti)
		if !ok || len(arch.Components) == 0 {
			t.Errorf("archétype ti=%d absent du registre du fixture (présent=%v, composants=%d)",
				ti, ok, len(arch.Components))
		}
	}
}
