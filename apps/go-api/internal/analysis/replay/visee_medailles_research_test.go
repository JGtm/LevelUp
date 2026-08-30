package replay

// visee_medailles_research_test.go — RECENSEMENT DE L'ORACLE DES MEDAILLES.
//
// POURQUOI. Deux medailles du jeu disent l'etat de lunette EN TOUTES LETTRES, et le film les
// porte, datees a la milliseconde et attribuees a un xuid :
//
//	No Scope (Tir au juge)     code film (type_hint=100, medal_type=114), id 2602963073.
//	                           « Kill an enemy with a Power sniper rifle WITHOUT ZOOMING »
//	                           -> au moment de ce kill, le TUEUR N'ETAIT PAS zoome.
//	Counter-snipe (Sniper snipe) code film (100, 168), id 1477806194.
//	                           « Headshot an enemy while YOU BOTH ARE ZOOMED with Power sniper
//	                           rifles » -> au moment de ce kill, LE TUEUR ET LA VICTIME etaient
//	                           zoomes tous les deux.
//
// Deux etiquettes OPPOSEES, sur la MEME classe d'arme, a des instants EXACTS. C'est l'oracle que
// le proxy « kill au sniper => sans doute zoome » n'etait pas.
//
// Correspondance (type_hint, medal_type) -> medaille : `.ai/refs/TABLE_MEDAILLES_FILM.tsv`.
// Libelles et descriptions : `medal_definitions` (metadata.duckdb).
//
// CE QUE CE FICHIER FAIT, ET RIEN DE PLUS. Il compte. Il ne decode aucun paquet de position : il
// ne lit que le DERNIER chunk de chaque film (le flux d'evenements), ce qui coute quelques
// millisecondes par film au lieu de plusieurs secondes. Sa sortie repond a une seule question :
// combien d'instants etiquetes le corpus donne-t-il, et dans quels films ? Sans ce chiffre, lancer
// un balayage de bits sur 954 films serait tirer a l'aveugle.
//
// SOUS GARDE D'ENVIRONNEMENT (ADS_FILMS_DIR), donc saute partout ailleurs, CI comprise.
//
// USAGE (depuis apps/go-api) :
//
//	CGO_ENABLED=0 ADS_FILMS_DIR=<repo>/data/cache/film_chunks \
//	  ADS_TSV=<repo>/.ai/V7.5/film_re \
//	  go test ./internal/analysis/replay/ -run TestViseeMedaillesRecensement -v -timeout 120m

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/filmdec"
)

const adsFilmsDirEnv = "ADS_FILMS_DIR"

// Codes film des trois medailles utiles. Ils viennent de la table du depot, pas d'une supposition.
const (
	adsTypeHintSkill = 50  // medailles « skill » simples
	adsTypeHintMulti = 100 // la famille qui porte No Scope et Counter-snipe
	adsMedalNoScope  = 114 // No Scope       -> tueur NON zoome
	adsMedalCounter  = 168 // Counter-snipe  -> tueur ET victime zoomes
	adsMedalSnipe    = 108 // Snipe (avec adsTypeHintSkill) : contexte, pas une etiquette
)

// adsMedailleFilm : ce qu'un film porte comme instants etiquetes.
type adsMedailleFilm struct {
	film             string
	noScope, counter int
	snipe            int
	instantsNoScope  []int
	instantsCounter  []int
	xuidNoScope      []uint64
	xuidCounter      []uint64
	// gtNoScope / gtCounter : le gamertag porte par l evenement, dans le MEME ordre que les
	// instants. Il ne sert a aucun calcul : il sert a pouvoir ALLER VOIR la scene dans Theater.
	gtNoScope        []string
	gtCounter        []string
	evenementsMedaux int
	chunkIntrouvable bool
}

// TestViseeMedaillesRecensement compte les instants etiquetes sur tout le corpus de films.
func TestViseeMedaillesRecensement(t *testing.T) {
	root := os.Getenv(adsFilmsDirEnv)
	if root == "" {
		t.Skipf("%s absent : recensement saute", adsFilmsDirEnv)
	}
	dirs := adsListeFilms(t, root)
	t.Logf("CORPUS — %d repertoires de film sous %s", len(dirs), root)

	debut := time.Now()
	var fiches []adsMedailleFilm
	illisibles := 0
	for _, d := range dirs {
		f, ok := adsRecenseFilm(root, d)
		if !ok {
			illisibles++
			continue
		}
		fiches = append(fiches, f)
	}
	t.Logf("COUT — %d films lus en %s (%d illisibles)", len(fiches),
		time.Since(debut).Round(time.Millisecond), illisibles)

	adsJournaliseRecensement(t, fiches)
	adsEcrisRecensementTSV(t, fiches)
}

// adsListeFilms rend les sous-repertoires du corpus, tries.
func adsListeFilms(t *testing.T, root string) []string {
	t.Helper()
	ents, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("corpus illisible : %v", err)
	}
	var out []string
	for _, e := range ents {
		if e.IsDir() {
			out = append(out, e.Name())
		}
	}
	sort.Strings(out)
	return out
}

// adsRecenseFilm lit le SEUL chunk d'evenements d'un film et compte ses medailles utiles.
func adsRecenseFilm(root, name string) (adsMedailleFilm, bool) {
	dir := filepath.Join(root, name)
	n := filmdec.CountFilmChunks(dir)
	if n == 0 {
		return adsMedailleFilm{film: name, chunkIntrouvable: true}, false
	}
	raw, err := os.ReadFile(filepath.Join(dir, fmt.Sprintf("chunk_%02d.bin", n)))
	if err != nil {
		return adsMedailleFilm{film: name, chunkIntrouvable: true}, false
	}
	evs, err := analysis.ParseHighlightEvents(raw, 0)
	if err != nil {
		return adsMedailleFilm{film: name}, false
	}
	f := adsMedailleFilm{film: name}
	for _, e := range evs {
		if e.EventType != analysis.EventTypeMedal {
			continue
		}
		f.evenementsMedaux++
		switch {
		case e.TypeHint == adsTypeHintMulti && e.MedalType == adsMedalNoScope:
			f.noScope++
			f.instantsNoScope = append(f.instantsNoScope, e.TimeMS)
			f.xuidNoScope = append(f.xuidNoScope, e.XUID)
			f.gtNoScope = append(f.gtNoScope, e.Gamertag)
		case e.TypeHint == adsTypeHintMulti && e.MedalType == adsMedalCounter:
			f.counter++
			f.instantsCounter = append(f.instantsCounter, e.TimeMS)
			f.xuidCounter = append(f.xuidCounter, e.XUID)
			f.gtCounter = append(f.gtCounter, e.Gamertag)
		case e.TypeHint == adsTypeHintSkill && e.MedalType == adsMedalSnipe:
			f.snipe++
		}
	}
	return f, true
}

// adsJournaliseRecensement publie les denominateurs de l'oracle.
func adsJournaliseRecensement(t *testing.T, fiches []adsMedailleFilm) {
	t.Helper()
	var totNo, totCo, totSn, filmsNo, filmsCo, filmsDeux, filmsAvecMedaille int
	for _, f := range fiches {
		totNo += f.noScope
		totCo += f.counter
		totSn += f.snipe
		if f.evenementsMedaux > 0 {
			filmsAvecMedaille++
		}
		if f.noScope > 0 {
			filmsNo++
		}
		if f.counter > 0 {
			filmsCo++
		}
		if f.noScope > 0 && f.counter > 0 {
			filmsDeux++
		}
	}
	t.Logf("MEDAILLES — %d films portent au moins une medaille lisible", filmsAvecMedaille)
	t.Logf("  No Scope      (tueur NON zoome) : %d instants sur %d films", totNo, filmsNo)
	t.Logf("  Counter-snipe (les DEUX zoomes) : %d instants sur %d films", totCo, filmsCo)
	t.Logf("  Snipe         (contexte, zoom non dit) : %d instants", totSn)
	t.Logf("  films portant LES DEUX etiquettes : %d", filmsDeux)

	if totNo == 0 || totCo == 0 {
		t.Log("VERDICT DE RECENSEMENT — une des deux etiquettes est absente du corpus :" +
			" l'oracle a deux faces ne peut pas etre monte tel quel.")
		return
	}
	t.Logf("VERDICT DE RECENSEMENT — l'oracle est MONTABLE : %d instants « pas zoome » contre"+
		" %d instants « zoome », etiquetes par le jeu lui-meme, dates a la ms.", totNo, totCo)
}

// adsEcrisRecensementTSV depose la liste des films utiles : c'est elle qui bornera le balayage de
// bits, pour ne decoder que les films qui portent une etiquette.
func adsEcrisRecensementTSV(t *testing.T, fiches []adsMedailleFilm) {
	t.Helper()
	out := os.Getenv(adsTSVEnv)
	if out == "" {
		return
	}
	var sb strings.Builder
	sb.WriteString("film\tno_scope\tcounter_snipe\tsnipe\tinstants_no_scope_ms\tinstants_counter_ms\n")
	for _, f := range fiches {
		if f.noScope == 0 && f.counter == 0 {
			continue
		}
		fmt.Fprintf(&sb, "%s\t%d\t%d\t%d\t%s\t%s\n", f.film, f.noScope, f.counter, f.snipe,
			adsJoinInts(f.instantsNoScope), adsJoinInts(f.instantsCounter))
	}
	path := filepath.Join(out, "visee_medailles_recensement.tsv")
	if err := os.WriteFile(path, []byte(sb.String()), 0o644); err != nil {
		t.Fatalf("ecriture du recensement : %v", err)
	}
	t.Logf("RELEVE — %s", path)
}

// adsJoinInts formate une liste d'instants pour le releve.
func adsJoinInts(v []int) string {
	if len(v) == 0 {
		return "-"
	}
	parts := make([]string, len(v))
	for i, x := range v {
		parts[i] = fmt.Sprint(x)
	}
	return strings.Join(parts, ",")
}

// TestViseeMedaillesScenes imprime des SCENES A ALLER VOIR : film, joueur, instant, etiquette.
//
// POURQUOI CE TEST EXISTE. Toute cette campagne repose sur une premisse VISUELLE — « on voit
// l'epaulement dans Theater » — et cette premisse n'a jamais ete testee, ni par moi ni par
// personne. Or le film porte 2378 instants ou le jeu AFFIRME que le tueur n'etait PAS a la
// lunette (No Scope) et 295 ou il affirme qu'il y ETAIT (Counter-snipe). Ouvrir Theater sur un
// instant de chaque et regarder la pose du tueur repond en deux minutes a une question que
// des heures de decodage n'ont pas tranchee :
//
//	si le tueur EST epaule sur un No Scope   -> l'epaulement n'est PAS la lunette, et toute la
//	                                            recherche d'un bit de zoom repose sur une
//	                                            observation mal interpretee ;
//	si les deux poses DIFFERENT visiblement  -> le client SAIT, l'information est bien quelque
//	                                            part, et il faut continuer a chercher — mais on
//	                                            saura enfin que la cible existe.
//
// Il ne mesure rien et ne conclut rien : il fabrique une liste de rendez-vous.
//
//	CGO_ENABLED=0 ADS_FILMS_DIR=<repo>/data/cache/film_chunks \
//	  go test ./internal/analysis/replay/ -run TestViseeMedaillesScenes -v
func TestViseeMedaillesScenes(t *testing.T) {
	root := os.Getenv(adsFilmsDirEnv)
	if root == "" {
		t.Skipf("%s absent : liste de scenes sautee", adsFilmsDirEnv)
	}
	const voulues = 6
	n := 0
	for _, d := range adsListeFilms(t, root) {
		f, ok := adsRecenseFilm(root, d)
		if !ok || f.noScope == 0 || f.counter == 0 {
			continue // on veut LES DEUX poses dans le MEME film : meme carte, meme partie
		}
		t.Logf("FILM %s — les deux etiquettes dans la meme partie", f.film)
		adsImprimeScenes(t, "SANS LUNETTE (No Scope)", f.gtNoScope, f.instantsNoScope)
		adsImprimeScenes(t, "AVEC LUNETTE (Counter-snipe)", f.gtCounter, f.instantsCounter)
		n++
		if n >= voulues {
			break
		}
	}
	if n == 0 {
		t.Log("aucun film ne porte les deux etiquettes")
		return
	}
	t.Log("MODE D'EMPLOI — ouvrir le film dans Theater, se placer sur le joueur nomme, avancer a" +
		" l'instant indique (temps de match, en ms depuis le debut du film) et regarder sa pose au" +
		" moment du tir. Comparer une ligne SANS LUNETTE et une ligne AVEC LUNETTE du meme film.")
}

// adsImprimeScenes ecrit les rendez-vous d'une etiquette.
func adsImprimeScenes(t *testing.T, etiquette string, gts []string, instants []int) {
	t.Helper()
	for i, ms := range instants {
		gt := "?"
		if i < len(gts) && gts[i] != "" {
			gt = gts[i]
		}
		t.Logf("    %-30s %-18s a %d ms (%02d:%02d.%03d)", etiquette, gt, ms,
			ms/60000, (ms/1000)%60, ms%1000)
	}
}
