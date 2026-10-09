package replaybuild

// cle_inconnue_test.go — LA POLITIQUE « CLE INCONNUE = FILM MIS DE COTE », PROUVEE COTE
// CUISSON (lot 3.1.1, D-4 d ADR 0034, critere S7 du PLAN_DECODEUR_FILM).
//
// # POURQUOI IL MORD SUR `ecarterSiCleInconnue` ET NON SUR `BuildBytes`
//
// `BuildBytes` exige un `Builder` construit, donc le catalogue de bornes, les libelles du titre
// et la table de reglement — trois fichiers de donnees du depot. La politique, elle, ne depend
// d AUCUN des trois : elle se place AVANT eux, sur le film seul. La prouver a travers un Builder
// ferait dependre un test de comportement de l etat de `data/`, sans rien prouver de plus. Le
// point d insertion dans `BuildBytes`, lui, est garde par la compilation : `ecarterSiCleInconnue`
// n a qu un seul appelant.
//
// # POURQUOI LA PREUVE PASSE PAR UNE MUTATION
//
// Aucun des 1 351 films du cache n a de cle hors profil (recensement du 2026-09-17) : la
// politique ne se declenche sur aucune donnee reelle — elle est ecrite pour la PROCHAINE mise a
// jour du jeu (V20 (2)). On fabrique donc le film qui n existe pas encore, en remplacant le nom
// de build ECRIT d une mini-bobine commise par un nom que la table ignore.

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/decfilm"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/testutil"
)

const (
	// bobineDeReference : la mini-bobine du build de reference, commise au depot.
	bobineDeReference = "fb1a1a72"
	// buildEcrit / buildHorsTable : la mutation, a longueur EGALE (la section 2 est a champs de
	// largeur fixe : un remplacement plus court deplacerait tout ce qui suit).
	buildEcrit     = "HI_1_13_0"
	buildHorsTable = "HI_9_99_0"
	// compteurDeLaCleRefusee : le compteur PAR CLE que `grammar` nomme et que la politique cable.
	compteurDeLaCleRefusee = "filmdec_unknown_build_hi_9_99_0"
)

// TestCuissonEcarteUnFilmACleInconnue : LE TEST NOMME DE L ITEM 3.1.1-a, cote `replaybuild`.
//
// Il prouve les trois moities de la politique : l erreur rendue est la SENTINELLE (donc aucun
// artefact n est cuit — `BuildBytes` sort sur elle avant la premiere lecture), le refus est
// COMPTE par cle, et le message porte la cle refusee.
func TestCuissonEcarteUnFilmACleInconnue(t *testing.T) {
	film := bobineChargee(t, bobineDeReference, true)
	avant := observability.LoadCounter(compteurDeLaCleRefusee)

	err := ecarterSiCleInconnue(context.Background(), "m1", film)
	if err == nil {
		t.Fatal("cle hors table : aucune erreur — le film serait cuit au profil des invariants")
	}
	if !errors.Is(err, ErrUnknownFilmKey) {
		t.Fatalf("erreur %v : ce n est pas ErrUnknownFilmKey — les producteurs la classeraient "+
			"en ECHEC au lieu d ECARTE", err)
	}
	if !strings.Contains(err.Error(), buildHorsTable) {
		t.Errorf("le message ne NOMME pas la cle refusee : %q", err.Error())
	}
	if apres := observability.LoadCounter(compteurDeLaCleRefusee); apres != avant+1 {
		t.Errorf("compteur %q : %d -> %d, attendu +1 — le refus est INVISIBLE en production",
			compteurDeLaCleRefusee, avant, apres)
	}
}

// TestLErreurDeCleTraverseLaFrontiereDeProcessus : LE GARDE-RAIL DE LA CLASSIFICATION A DISTANCE.
//
// La cuisson part dans un ENFANT (`replaychild`, `cmd/levelup backfill-replay`) : le parent ne
// recoit qu un code de sortie et un `stderr`, donc `sync/replayartifacts` classe sur le TEXTE de
// la sentinelle, exactement comme il le fait pour `ErrMapNotInCatalog`. Si le message enveloppe
// cessait de contenir celui de la sentinelle, la classification retomberait silencieusement en
// ECHEC — et personne ne verrait la difference avant de chercher une panne inexistante.
func TestLErreurDeCleTraverseLaFrontiereDeProcessus(t *testing.T) {
	err := ecarterSiCleInconnue(context.Background(), "m1", bobineChargee(t, bobineDeReference, true))
	if err == nil {
		t.Fatal("cle hors table : aucune erreur")
	}
	if !strings.Contains(err.Error(), ErrUnknownFilmKey.Error()) {
		t.Fatalf("le message enveloppe ne contient pas celui de la sentinelle :\n  %q\n  %q",
			err.Error(), ErrUnknownFilmKey.Error())
	}
}

// TestCuissonNEcartePasUnFilmACleConnue : LE CONTROLE NEGATIF. Sans lui, une garde qui ecarte
// TOUT passerait le test ci-dessus.
func TestCuissonNEcartePasUnFilmACleConnue(t *testing.T) {
	film := bobineChargee(t, bobineDeReference, false)
	avant := observability.LoadCounter("filmdec_unknown_build_hi_1_13_0")

	if err := ecarterSiCleInconnue(context.Background(), "m1", film); err != nil {
		t.Fatalf("le build de reference est ECARTE : %v", err)
	}
	if apres := observability.LoadCounter("filmdec_unknown_build_hi_1_13_0"); apres != avant {
		t.Errorf("compteur de cle refusee incremente sur une cle CONNUE : %d -> %d", avant, apres)
	}
}

// TestUneBobineSansRegistreNEstPasEcartee : UNE BOBINE PARTIELLE N EST PAS UNE CLE INCONNUE.
//
// Un film sans `chunk_00` n ecrit aucune cle : la refuser transformerait un diagnostic de
// lecture deja nomme en « cle inconnue », qui serait FAUX. C est la meme regle que
// `grammar.journaliserProfilIncomplet`, et elle vaut ici aussi — sinon toute bobine partielle du
// parc deviendrait un film ecarte.
func TestUneBobineSansRegistreNEstPasEcartee(t *testing.T) {
	film, err := decfilm.Load(decfilm.MemoryChunks([][]byte{nil}),
		[]decfilm.ChunkMeta{{Index: 0, ChunkType: 2}})
	if err != nil {
		t.Fatalf("chargement d une bobine sans registre : %v", err)
	}
	if err := ecarterSiCleInconnue(context.Background(), "m1", film); err != nil {
		t.Fatalf("bobine sans chunk_00 ECARTEE pour cle inconnue : %v", err)
	}
	if err := ecarterSiCleInconnue(context.Background(), "m1", nil); err != nil {
		t.Fatalf("film nil ECARTE pour cle inconnue : %v", err)
	}
}

// bobineChargee lit une mini-bobine COMMISE et rend le film charge. `muter` remplace le nom de
// build par un nom hors table, dans les octets DECOMPRESSES (`source.Load` laisse passer un
// chunk deja decompresse).
func bobineChargee(t *testing.T, court string, muter bool) *decfilm.Film {
	t.Helper()
	racine, err := testutil.RepoRoot()
	if err != nil {
		t.Fatalf("racine du depot : %v", err)
	}
	dir := filepath.Join(racine, "apps", "go-api", "internal", "games", "halo_infinite", "film",
		"replay", "testdata", "minifilm_"+court)
	noms, err := filepath.Glob(filepath.Join(dir, "chunk_*.bin"))
	if err != nil || len(noms) == 0 {
		t.Fatalf("mini-bobine %s : aucun chunk lu (%v)", dir, err)
	}
	sort.Strings(noms)
	seq := make([][]byte, len(noms))
	meta := make([]decfilm.ChunkMeta, len(noms))
	for i, nom := range noms {
		brut, err := os.ReadFile(nom)
		if err != nil {
			t.Fatalf("lecture de %s : %v", nom, err)
		}
		seq[i] = decfilm.Inflate(brut)
		if i == 0 && muter {
			mute := strings.ReplaceAll(string(seq[i]), buildEcrit, buildHorsTable)
			if strings.Contains(mute, buildEcrit) {
				t.Fatalf("mutation sans effet : %q est encore ecrit dans chunk_00", buildEcrit)
			}
			seq[i] = []byte(mute)
		}
		meta[i] = decfilm.ChunkMeta{Index: i, ChunkType: typeDeChunk(i, len(noms))}
	}
	film, err := decfilm.Load(decfilm.MemoryChunks(seq), meta)
	if err != nil {
		t.Fatalf("chargement de la mini-bobine %s : %v", court, err)
	}
	return film
}

// typeDeChunk : le type deduit de la position — chunk 0 en-tete, dernier temps forts, le reste
// replication.
func typeDeChunk(idx, total int) int {
	switch {
	case idx == 0:
		return 1
	case idx == total-1:
		return 3
	default:
		return 2
	}
}
