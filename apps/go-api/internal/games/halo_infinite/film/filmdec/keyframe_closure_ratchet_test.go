package filmdec

// keyframe_closure_ratchet_test.go — LA COUVERTURE D'IMAGE-CLE NE DESCEND JAMAIS (lot 0.A.3).
//
// # CE QUE CE RATCHET GARDE
//
// `KeyframeClosure` mesure, par archetype, combien de records d'image-cle FERMENT — c'est-a-dire
// combien atterrissent exactement sur le premier bit du record suivant. C'est le seul oracle qui
// prouve une largeur de composant sans capture live. Le golden fige ces comptes sur les bobines
// par build ; le test rougit des qu'un compte DESCEND.
//
// # POURQUOI IL ROUGIT SUR LA BAISSE ET PAS SUR LA HAUSSE
//
// Le chantier du lot 3.6 est un PORT de composants : chaque composant porte debloque des records
// qui butaient dessus, donc les comptes MONTENT, lot apres lot. Rougir sur une hausse ferait
// echouer le progres qu'on cherche (la lecon de `replaydiff/polarite.go` : un compteur qu'on lit
// a l'envers refuse exactement ce qu'il devait garder). Une hausse est donc acceptee et
// SIGNALEE — le golden se regenere alors explicitement, ce qui laisse la trace du gain dans le
// diff.
//
// # LA PREUVE QUE LE RATCHET MORD
//
// Fausser une largeur du registre — par exemple `R(6)` en `R(7)` sur ti=6 — decale tous les
// composants qui suivent, donc la marche n'atterrit plus sur la frontiere et la fermeture tombe.
// Verifie le 2026-09-13 (cf. le rapport du lot).
//
// REGENERATION (jamais d edition a la main) :
//
//	go test ./internal/games/halo_infinite/film/filmdec/ -run KeyframeClosureRatchet -update-keyframe-closure

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/filmsource"
)

// updateFermeture : LA PORTE DE REGENERATION DE CE GOLDEN, ET D AUCUN AUTRE.
//
// ELLE EST NOMMEE, ET C EST LE CORRECTIF DE LA REVUE R1 (P1-1). Ce golden a d abord ete
// accroche au `-update` du corpus de graines du fuzz, sous le pretexte qu un second drapeau
// ferait paniquer `flag`. C ETAIT FAUX : `flag` ne panique que sur un NOM deja pris, et le
// paquet declare deja `-update-golden-familles` (`golden_minibobine_test.go`). Le cout de
// l erreur etait exactement celui que le lot est cense empecher : avec une largeur faussee en
// place, `go test ./...filmdec/ -update` (sans `-run`) repondait `ok` et REECRIVAIT ce golden
// avec la grammaire cassee — le ratchet ne le disait que par un `t.Logf` invisible sans `-v`.
// C est la reouverture du defaut C5 ferme le 2026-09-06 : « une porte de regeneration doit
// nommer CE qu elle regenere » (`fuzz_records_test.go`).
var updateFermeture = flag.Bool("update-keyframe-closure", false,
	"reecrire testdata/keyframe_closure.golden (lot 0.A.3) — CE golden seulement")

// closureGoldenPath : le golden, a cote des autres references du paquet.
const closureGoldenPath = "testdata/keyframe_closure.golden"

// closureMiniFilms : les bobines par build, celles du lot 0.A.2. Elles portent toutes leur
// `chunk_00`, sans lequel il n'y aurait ni registre ni archetypes — donc aucune fermeture a
// mesurer. La bobine historique `minifilm_000d5950` n'y est PAS : elle n'a pas de `chunk_00`.
func closureMiniFilms() []string {
	return []string{
		"a521164d", // HI_1_4_1
		"60ae07c4", // HI_1_8_0
		"11de8353", // HI_1_9_0
		"111fa685", // HI_1_10_0
		"e5adf7b2", // HI_1_11_0
		"bcb6d393", // HI_1_12_0
		"fb1a1a72", // HI_1_13_0
	}
}

// TestKeyframeClosureRatchet : la couverture par archetype ne descend jamais.
func TestKeyframeClosureRatchet(t *testing.T) {
	// `KeyframeClosure` installe le profil MPP du build : globaux de paquet, donc verrou.
	relVerrou := LockProcessDecode()
	defer relVerrou()
	got := mesurerFermetureBobines(t)
	if *updateFermeture {
		if err := os.MkdirAll(filepath.Dir(closureGoldenPath), 0o750); err != nil {
			t.Fatalf("creation de testdata : %v", err)
		}
		if err := os.WriteFile(closureGoldenPath, []byte(got), 0o600); err != nil {
			t.Fatalf("ecriture du golden : %v", err)
		}
		// UNE PORTE DE REGENERATION NE REND JAMAIS `ok` (revue R2, C1). `go test` JETTE la sortie
		// d un paquet qui PASSE : un `t.Logf`, et meme une ecriture directe sur stderr, sont
		// INVISIBLES avec la commande documentee (sans `-v`). Mesure : le golden etait reecrit,
		// stdout rendait `ok`, stderr restait vide. Terminer en ECHEC est la seule forme qui rende
		// la reecriture visible ET qui empeche de confondre une regeneration avec un run vert.
		t.Fatalf("1 reference(s) reecrite(s) : %s (%d octets) ; "+
			"relancer sans -update-keyframe-closure pour verifier", closureGoldenPath, len(got))
	}
	brut, err := os.ReadFile(closureGoldenPath) //nolint:gosec // chemin fige dans le code
	if err != nil {
		t.Fatalf("golden absent (%s) : %v — regenerer avec -update-keyframe-closure",
			closureGoldenPath, err)
	}
	comparerFermeture(t, string(brut), got)
}

// comparerFermeture confronte le golden a la mesure, ligne a ligne.
//
// Une BAISSE est une erreur ; une HAUSSE est un gain a figer ; une ligne qui DISPARAIT est une
// erreur (un archetype cesse d'etre mesure), une ligne NEUVE est un gain.
func comparerFermeture(t *testing.T, want, got string) {
	t.Helper()
	fige, obtenu := lignesFermeture(want), lignesFermeture(got)
	for cle, ref := range fige {
		cur, ok := obtenu[cle]
		if !ok {
			t.Errorf("%s : l'archetype a DISPARU de la mesure (fige : %d/%d fermes)",
				cle, ref.closed, ref.total)
			continue
		}
		if cur.closed < ref.closed {
			t.Errorf("%s : fermeture en BAISSE, %d/%d fige contre %d/%d obtenu — bloquant %q.\n"+
				"Une largeur a bouge : la corriger, ou regenerer le golden si la baisse est voulue "+
				"et justifiee dans le commit.",
				cle, ref.closed, ref.total, cur.closed, cur.total, cur.blocking)
		}
		if cur.closed > ref.closed {
			t.Logf("%s : fermeture en HAUSSE, %d/%d -> %d/%d — figer par -update-keyframe-closure",
				cle, ref.closed, ref.total, cur.closed, cur.total)
		}
	}
	for cle, cur := range obtenu {
		if _, ok := fige[cle]; !ok {
			t.Logf("%s : archetype NEUF dans la mesure (%d/%d) — figer par -update-keyframe-closure",
				cle, cur.closed, cur.total)
		}
	}
}

// compteFermeture est une ligne de golden relue.
type compteFermeture struct {
	closed, total int
	blocking      string
}

// lignesFermeture relit le golden en table `film ti=N` -> comptes. Les lignes de commentaire et
// les lignes vides sont ignorees.
func lignesFermeture(s string) map[string]compteFermeture {
	out := map[string]compteFermeture{}
	for _, l := range strings.Split(s, "\n") {
		l = strings.TrimSpace(l)
		if l == "" || strings.HasPrefix(l, "#") {
			continue
		}
		cols := strings.Split(l, "\t")
		if len(cols) < 4 {
			continue
		}
		closed, err1 := strconv.Atoi(cols[2])
		total, err2 := strconv.Atoi(cols[3])
		if err1 != nil || err2 != nil {
			continue
		}
		bloquant := ""
		if len(cols) > 4 {
			bloquant = cols[4]
		}
		out[cols[0]+" "+cols[1]] = compteFermeture{closed: closed, total: total, blocking: bloquant}
	}
	return out
}

// mesurerFermetureBobines rend le rendu textuel de la mesure sur toutes les bobines par build.
func mesurerFermetureBobines(t *testing.T) string {
	t.Helper()
	var b strings.Builder
	b.WriteString("# FERMETURE DES RECORDS D'IMAGE-CLE PAR ARCHETYPE — golden du lot 0.A.3.\n")
	b.WriteString("# Une colonne par mesure : film, archetype, fermes, total, bloquant le plus frequent.\n")
	b.WriteString("# Le ratchet rougit sur une BAISSE de `fermes`. Regeneration :\n")
	b.WriteString("#   go test ./internal/games/halo_infinite/film/filmdec/ -run KeyframeClosureRatchet -update-keyframe-closure\n")
	b.WriteString("#\n")
	b.WriteString("# HISTORIQUE DES REGENERATIONS — une ligne par lot, avec CE QUI MONTE ET POURQUOI.\n")
	b.WriteString("# Un golden de couverture qu'on refige sans dire ce qu'il gagne ne garde plus rien.\n")
	b.WriteString("#\n")
	b.WriteString("#   2026-09-13 lot 0.A.3 : creation, 7 bobines par build, 217 lignes de mesure.\n")
	b.WriteString("#   2026-09-14 lot 1.3   : les CINQ etats par defaut manquants relus chez l'ecrivain\n")
	b.WriteString("#     (ti14 `V ; R(5)` FUN_140FED6F4 · ti17 `V ; R(7)` FUN_14101A0A4 · ti21 `R(18)`\n")
	b.WriteString("#     FUN_141133C24 · ti29 `V` seul FUN_14116F514 · ti47 `V ; R(5)` FUN_1410F44F8).\n")
	b.WriteString("#     21 lignes MONTENT, 0 descend, aucune ne disparait, aucun total ne bouge :\n")
	b.WriteString("#       ti=14    0/3520 -> 3520/3520 (100 %, les 7 bobines)\n")
	b.WriteString("#       ti=17    0/3729 -> 3729/3729 (100 %, les 7 bobines)\n")
	b.WriteString("#       ti=29    0/110  ->  102/110  (100 % sur 6 bobines, 22/30 sur 60ae07c4)\n")
	b.WriteString("#     ti=21 (0/357) et ti=47 (0/1716) ne bougent PAS, et c'etait prevu : leur largeur\n")
	b.WriteString("#     est prouvee par deux chaines, mais un composant reste faux — ti=47 bute sur\n")
	b.WriteString("#     `i2 personal-ai-data-component`, ti=21 sous-lit. Une largeur juste ne remplace\n")
	b.WriteString("#     pas un deserialiseur manquant (lot 3.6).\n")
	b.WriteString("#     Meme geste sur les 6 films de recherche : 8 796/62 686 (14,0 %) -> 19 337/62 686\n")
	b.WriteString("#     (30,8 %), delta = 5 024 + 5 379 + 138, et le plancher de hasard DESCEND de\n")
	b.WriteString("#     529 a 391. Mesure rejouable : `TestImageCleFermetureParArchetype` sous CHUNK00_FILMS.\n")
	for _, court := range closureMiniFilms() {
		dir := filepath.Join("..", "replay", "testdata", "minifilm_"+court)
		stats := fermetureDUneBobine(t, dir)
		tis := make([]int, 0, len(stats))
		for ti := range stats {
			tis = append(tis, int(ti))
		}
		sort.Ints(tis)
		for _, ti := range tis {
			s := stats[uint32(ti)] //nolint:gosec // ti vient d'une cle uint32
			fmt.Fprintf(&b, "%s\tti=%d\t%d\t%d\t%s\n", court, ti, s.Closed, s.Total, s.Blocking)
		}
	}
	return b.String()
}

// fermetureDUneBobine charge une bobine et rend sa fermeture par archetype.
func fermetureDUneBobine(t *testing.T, dir string) map[uint32]KeyframeClosureStat {
	t.Helper()
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("bobine absente (%s) : %v — regenerer les bobines du lot 0.A.2", dir, err)
	}
	film, err := filmsource.LoadDir(dir, nil)
	if err != nil {
		t.Fatalf("LoadDir %s : %v", dir, err)
	}
	stats, err := KeyframeClosure(NewFilmContext(film))
	if err != nil {
		t.Fatalf("KeyframeClosure %s : %v", dir, err)
	}
	return stats
}
