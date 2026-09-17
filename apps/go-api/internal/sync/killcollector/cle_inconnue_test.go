package killcollector

// cle_inconnue_test.go — LA POLITIQUE « CLE INCONNUE = FILM MIS DE COTE », PROUVEE COTE
// KILLCOLLECTOR (lot 3.1.1, D-4 d ADR 0034, critere S7 du PLAN_DECODEUR_FILM).
//
// # POURQUOI LA PREUVE PASSE PAR UNE MUTATION
//
// AUCUN des 1 351 films du cache n a de cle hors profil (recensement du 2026-09-17, comme celui
// du 2026-09-14 avant lui) : la politique ne se declenche sur AUCUNE donnee reelle, et c est
// exactement ce qu on veut — elle est ecrite pour la PROCHAINE mise a jour du jeu (V20 (2)). La
// seule facon de prouver qu elle mord est donc de fabriquer le film qui n existe pas encore :
// une mini-bobine COMMISE dont on remplace le nom de build par un nom que la table ignore. Meme
// geste que `replay.TestScanFilmPlayerTableCableLeCompteurDeBuildInconnu`.

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/halo_infinite/film/decfilm"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/sync/haloclient"
	"levelup/go-api/internal/testutil"
)

const (
	// bobineCleConnue : la mini-bobine du build de REFERENCE, commise au depot.
	bobineCleConnue = "fb1a1a72"
	// buildEcritParLaBobine / buildHorsTable : la mutation. Les deux chaines ont la MEME
	// longueur — la section 2 est a champs de largeur fixe, un remplacement plus court
	// deplacerait tout ce qui suit et prouverait autre chose.
	buildEcritParLaBobine = "HI_1_13_0"
	buildHorsTable        = "HI_9_99_0"
	// compteurCleRefusee : le compteur PAR CLE que `grammar` nomme et que la politique cable.
	compteurCleRefusee = "filmdec_unknown_build_hi_9_99_0"
)

// TestKillCollectorEcarteUnFilmACleInconnue : LE TEST NOMME DE L ITEM 3.1.1-a, cote
// `sync/killcollector`.
//
// Il prouve les quatre moities de la politique d un coup : le film est ECARTE (outcome dedie),
// AUCUN fait n est ecrit (le collecteur n a meme pas de base branchee — un seul appel a
// l ecriture paniquerait), le refus est COMPTE par cle, et la passe multi-matchs le range dans
// sa propre colonne au lieu de le confondre avec une panne.
func TestKillCollectorEcarteUnFilmACleInconnue(t *testing.T) {
	chunks := bobineEnChunks(t, bobineCleConnue, true)
	client := &clientDeBobine{chunks: chunks}
	col := NewKillSourceCollector(client, rosterVide{}, nil, capsAvecFilmPourCle(), 0)

	avant := observability.LoadCounter(compteurCleRefusee)
	avantPasses := observability.LoadCounter(metricUnknownKey)

	outcome, morts, err := col.CollectMatch(context.Background(), "m1")
	if err != nil {
		t.Fatalf("CollectMatch : erreur rendue %v — une cle inconnue est un ETAT, pas une panne", err)
	}
	if outcome != OutcomeUnknownKey {
		t.Fatalf("outcome = %q, attendu %q — le film n a PAS ete mis de cote", outcome, OutcomeUnknownKey)
	}
	if morts != 0 {
		t.Errorf("morts = %d, attendu 0 — un film ecarte ne publie AUCUN fait", morts)
	}
	if apres := observability.LoadCounter(compteurCleRefusee); apres != avant+1 {
		t.Errorf("compteur %q : %d -> %d, attendu +1 — le refus est INVISIBLE en production",
			compteurCleRefusee, avant, apres)
	}
	if apres := observability.LoadCounter(metricUnknownKey); apres != avantPasses+1 {
		t.Errorf("compteur %q : %d -> %d, attendu +1", metricUnknownKey, avantPasses, apres)
	}

	// LA SYNTHESE DE PASSE RANGE L ECARTE A PART : ni un ecrit, ni une erreur. Les confondre
	// ferait chercher une panne la ou il faut ecrire une ligne de table.
	sum := col.CollectMatches(context.Background(), []string{"m1"})
	if sum.UnknownKey != 1 || sum.Errors != 0 || sum.Written != 0 || sum.NoFilm != 0 {
		t.Errorf("synthese : ecartes=%d erreurs=%d ecrits=%d sans_film=%d, attendu 1/0/0/0",
			sum.UnknownKey, sum.Errors, sum.Written, sum.NoFilm)
	}
}

// TestKillCollectorNEcartePasUnFilmACleConnue : LE CONTROLE NEGATIF, sans lequel le test
// ci-dessus passerait aussi sur une garde qui ecarte TOUT.
//
// La mini-bobine NON mutee traverse la porte : l outcome n est pas `ecarte-cle-inconnue` et
// AUCUN compteur de cle ne bouge. Ce qu elle devient ensuite (une mini-bobine de trois chunks
// n a pas de kill-feed exploitable) n est pas le sujet — la porte de la cle est en amont.
func TestKillCollectorNEcartePasUnFilmACleConnue(t *testing.T) {
	chunks := bobineEnChunks(t, bobineCleConnue, false)
	client := &clientDeBobine{chunks: chunks}
	col := NewKillSourceCollector(client, rosterVide{}, nil, capsAvecFilmPourCle(), 0)

	avantRef := observability.LoadCounter("filmdec_unknown_build_hi_1_13_0")
	avantPasses := observability.LoadCounter(metricUnknownKey)

	outcome, _, _ := col.CollectMatch(context.Background(), "m1")
	if outcome == OutcomeUnknownKey {
		t.Fatalf("le build de reference est ECARTE — la porte refuse ce que le profil connait")
	}
	if apres := observability.LoadCounter("filmdec_unknown_build_hi_1_13_0"); apres != avantRef {
		t.Errorf("compteur de cle refusee incremente sur une cle CONNUE : %d -> %d", avantRef, apres)
	}
	if apres := observability.LoadCounter(metricUnknownKey); apres != avantPasses {
		t.Errorf("compteur de passes ecartees incremente sur une cle CONNUE : %d -> %d",
			avantPasses, apres)
	}
}

// bobineEnChunks lit une mini-bobine COMMISE et rend ses chunks sous la forme que le pont
// fournit. `muter` remplace le nom de build par un nom hors table, dans les octets DECOMPRESSES.
//
// LA MUTATION PORTE SUR LE CHUNK_00 INFLATE, et `source.Load` laisse passer un chunk deja
// decompresse : recompresser pour le faire re-decompresser ne prouverait rien de plus.
func bobineEnChunks(t *testing.T, court string, muter bool) []haloclient.FilmChunk {
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
	out := make([]haloclient.FilmChunk, 0, len(noms))
	for idx, nom := range noms {
		brut, err := os.ReadFile(nom)
		if err != nil {
			t.Fatalf("lecture de %s : %v", nom, err)
		}
		data := decfilm.Inflate(brut)
		if idx == 0 && muter {
			data = muterLeBuild(t, data)
		}
		out = append(out, haloclient.FilmChunk{Index: idx, Data: data,
			ChunkType: typeDeChunkDeBobine(idx, len(noms))})
	}
	return out
}

// muterLeBuild remplace le nom de build ECRIT par un nom que la table de profil ignore, et
// VERIFIE que la mutation a pris : une substitution silencieusement sans effet rendrait ce test
// vert sur un film de cle connue.
func muterLeBuild(t *testing.T, chunk0 []byte) []byte {
	t.Helper()
	mute := []byte(strings.ReplaceAll(string(chunk0), buildEcritParLaBobine, buildHorsTable))
	if strings.Contains(string(mute), buildEcritParLaBobine) {
		t.Fatalf("mutation sans effet : %q est encore ecrit dans chunk_00", buildEcritParLaBobine)
	}
	return mute
}

// typeDeChunkDeBobine : le type deduit de la position, comme le pont le fournirait — chunk 0
// en-tete, dernier temps forts, le reste replication (cf. `chargerFilmDeFixture`).
func typeDeChunkDeBobine(idx, total int) int {
	switch {
	case idx == 0:
		return 1
	case idx == total-1:
		return 3
	default:
		return 2
	}
}

// clientDeBobine : le pont film reduit a une seule mini-bobine. Fakes PROPRES a ce fichier —
// ceux de `collector_test.go` vivent derriere le tag `integration`, et ce test doit tourner
// dans les DEUX regimes : la politique de cle inconnue est une garde de production, pas une
// preuve d integration.
type clientDeBobine struct{ chunks []haloclient.FilmChunk }

func (c *clientDeBobine) GetFilmChunks(_ context.Context, _ string) ([]haloclient.FilmChunk, bool, error) {
	return c.chunks, true, nil
}

// rosterVide : aucune identite. Un film ECARTE ne doit jamais l interroger ; un film traverse
// la porte et s arrete plus loin, faute de kill-feed.
type rosterVide struct{}

func (rosterVide) IdentitiesForMatch(_ context.Context, _ string) (MatchIdentities, error) {
	return MatchIdentities{ParNom: map[string]string{}, ParXUID: map[string]string{},
		ShotsFired: map[string]int{}}, nil
}

// capsAvecFilmPourCle : la capability de la source de kill, activee.
func capsAvecFilmPourCle() games.CapabilityMap {
	return games.CapabilityMap{games.CapFilmKillSource: games.CapSupported}
}
