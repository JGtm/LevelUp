package killsource

// e193_couple_evenement_mesure_research_test.go — LOT 1.9.3, LA MESURE AVANT DE CODER.
//
// # LA QUESTION, POSEE AVANT TOUTE LIGNE DE PRODUCTION
//
// Le couple (tueur, victime) d un frag se DECIDE aujourd hui par [reconstructPairs] : quand le
// kill-feed ne porte pas les deux champs au meme instant, le kill orphelin est RECOLLE sur la
// mort d un voisin immediat (fenetre de 2 instants). Or le film ECRIT le couple : le kill-event
// de code 85 porte victime ET tueur dans le MEME enregistrement
// (victime(E5) tueur(E5) [% TUEUR] R1 assistant(E5) [% ASSISTANT], [readKillEvent]) — et ce
// lecteur existe deja, mais il n est lu que pour l ASSISTANT ([decodeCtx.attachAssists]).
//
// Avant de convertir, il faut CHIFFRER, film par film :
//
//	combien de couples sont RECOLLES aujourd hui, et combien sont ecrits au meme instant ;
//	ce que le kill-event 85 ECRIT en face de chacun (accord / desaccord / silence) ;
//	combien de couples RECOLLES ont en realite une victime BOT — la fabrication que le lot
//	  doit supprimer (« les vies anonymes n existent pas », decision utilisateur du 2026-09-06).
//
// # LE LIEN INDICE -> JOUEUR EMPLOYE ICI EST LA PART *LUE*, ET RIEN D AUTRE
//
// Un kill-event porte des INDICES de replication, pas des noms. Les traduire par la bijection
// resolue serait CIRCULAIRE : la bijection est ajustee sur [killFeed.pairs], donc sur le
// recollage que l on veut juger. La mesure n emploie donc que les indices EPINGLES — la table
// des joueurs de chunk_00 (lots 1.5 et 1.8) et BOT_METADATA —, qui sont poses AVANT toute
// inference et ne dependent d aucun couple. C est exactement le lien que la conversion pourra
// employer en production, et la mesure dit combien de films en disposent.
//
// # CE QU IL NE FAIT PAS
//
// Aucune ecriture, aucune base, aucun artefact, aucun replay-equiv : il charge des films en
// lecture seule et compte. Il ne deroule NI la marche des morts NI le scan de dead-states — la
// question porte sur le kill-feed et sur la liste d evenements, pas sur la source du degat.
//
// GARDE CHUNK00_FILMS (repertoires de film absolus separes par « ; ») pour les films entiers ;
// la bobine versionnee testdata/minibobine_000d5950 est mesuree SANS garde, en CI.
//
//	CHUNK00_FILMS=<dir1>;<dir2> CGO_ENABLED=0 \
//	  go test ./internal/games/halo_infinite/film/facts/killsource/ \
//	  -run TestE193CoupleEcritContreRecollage -v -count=1 -timeout 3600s

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/source"
)

// e193Verdict : ce que le kill-event 85 dit d un instant du kill-feed, confronte a ce que la
// reconstruction en fait. Domaine FERME — un instant tombe dans un et un seul.
type e193Verdict string

const (
	// e193Accord : le couple ecrit est celui que la reconstruction rend.
	e193Accord e193Verdict = "accord"
	// e193DesaccordVictime : le couple ecrit nomme une AUTRE victime humaine.
	e193DesaccordVictime e193Verdict = "desaccord_victime"
	// e193VictimeBot : le couple ecrit nomme un BOT en victime. Sur un couple RECOLLE, c est la
	// FABRICATION que le lot supprime ; sur un kill orphelin, c est la mort de bot deja connue.
	e193VictimeBot e193Verdict = "victime_bot"
	// e193Ambigu : plusieurs kill-events de la fenetre nomment le meme tueur et DES VICTIMES
	// DIFFERENTES. La lecture ne tranche pas ; elle ne decide donc pas.
	e193Ambigu e193Verdict = "ambigu"
	// e193Muet : aucun kill-event de la fenetre ne nomme ce tueur par un indice EPINGLE.
	e193Muet e193Verdict = "muet"
	// e193SansMortAuFeed : le couple ecrit nomme une victime HUMAINE dont le kill-feed ne porte
	// aucune mort dans le voisinage. Le feed est humain-seul et porte TOUTES les morts humaines :
	// une lecture qui tombe ici se contredit avec lui, et elle se COMPTE au lieu de decider.
	e193SansMortAuFeed e193Verdict = "sans_mort_au_feed"
)

// e193Population : d ou vient l instant juge, dans le vocabulaire de [killFeed.split].
type e193Population string

const (
	// e193Reel : le kill-feed porte le kill ET la mort au MEME instant. Ce n est pas une
	// reconstruction ; le kill-event y sert de CONTROLE.
	e193Reel e193Population = "couple_meme_instant"
	// e193Recolle : le couple est RECOLLE sur la mort d un voisin — la population que le lot
	// convertit.
	e193Recolle e193Population = "couple_recolle"
	// e193Orphelin : le kill n a trouve AUCUN voisin a consommer.
	e193Orphelin e193Population = "kill_orphelin"
	// e193KillSansMort : LA POPULATION QUE LA CONVERSION VISE, sous le regime d ASSIGNATION.
	// Elle reunit [e193Recolle] et [e193Orphelin] — tout instant du feed qui porte un kill sans
	// mort en face — et elle est jugee APRES que les couples du MEME INSTANT ont consomme leurs
	// kill-events (voir [e193Assigner]).
	e193KillSansMort e193Population = "kill_sans_mort_assigne"
)

// e193Ligne : la mesure d un film.
type e193Ligne struct {
	Film       string
	Build      string
	Refus      FilmTableRefusal
	Epingles   int // indices epingles (table du film + BOT_METADATA)
	Bots       int
	Instants   int
	Kills      int
	Morts      int
	Pairs      int
	Reels      int
	Recolles   int
	Orphelins  int
	MortsOrph  int
	KillEvents int
	// Par : la ventilation par population puis par verdict.
	Par map[e193Population]map[e193Verdict]int
	Err error
}

func (l *e193Ligne) compte(p e193Population, v e193Verdict) {
	if l.Par == nil {
		l.Par = map[e193Population]map[e193Verdict]int{}
	}
	if l.Par[p] == nil {
		l.Par[p] = map[e193Verdict]int{}
	}
	l.Par[p][v]++
}

func (l e193Ligne) n(p e193Population, v e193Verdict) int { return l.Par[p][v] }

// e193Films : les repertoires de film a mesurer. La bobine versionnee toujours, les films
// entiers du cache quand CHUNK00_FILMS les nomme.
func e193Films() []string {
	dirs := []string{filepath.Join("testdata", "minibobine_000d5950")}

	for _, d := range strings.Split(os.Getenv("CHUNK00_FILMS"), ";") {
		if d = strings.TrimSpace(d); d != "" {
			dirs = append(dirs, d)
		}
	}
	return dirs
}

// e193Nom : le nom court d un film, depuis son repertoire.
func e193Nom(dir string) string {
	base := filepath.Base(strings.TrimRight(dir, `/\`))
	if s, ok := strings.CutPrefix(base, "minibobine_"); ok {
		return s + "(bob)" // la BOBINE et le film ENTIER portent le meme identifiant court
	}
	return base
}

// e193Epinglage : le lien indice -> nom EPINGLE (table du film + BOT_METADATA), et les indices
// que BOT_METADATA tient. C est la part LUE de la bijection, disponible avant toute inference.
func e193Epinglage(r *roster) (nom map[int]string, bot map[int]bool) {
	nom, bot = map[int]string{}, map[int]bool{}
	for idx, pos := range r.pin {
		if pos >= 0 && pos < len(r.names) {
			nom[idx] = r.names[pos]
		}
		if !r.seatPin[idx] {
			bot[idx] = true
		}
	}
	return nom, bot
}

// e193Lu : ce que les kill-events de la fenetre ecrivent pour un tueur donne.
type e193Lu struct {
	Victimes map[string]bool
	Bot      bool
}

// e193LireCouple : les victimes que le film ECRIT pour ce tueur a cet instant. Un kill-event
// n est retenu que si SES DEUX indices sont epingles — sans quoi il ne nomme rien.
func e193LireCouple(recs []killEventRec, ms int, tueur string, nom map[int]string, bot map[int]bool) e193Lu {
	out := e193Lu{Victimes: map[string]bool{}}
	for i := range recs {
		r := &recs[i]
		if dt := r.ms - ms; dt < -tolMS || dt > tolMS {
			continue
		}
		nk, okk := nom[r.fields.killer]
		nv, okv := nom[r.fields.victim]
		if !okk || !okv || nk != tueur {
			continue
		}
		out.Victimes[nv] = true
		if bot[r.fields.victim] {
			out.Bot = true
		}
	}
	return out
}

// e193Verdicter : le verdict d un instant, depuis ce que le film ecrit et ce que la
// reconstruction rend.
func e193Verdicter(lu e193Lu, victimeRecollee string) e193Verdict {
	switch {
	case len(lu.Victimes) == 0:
		return e193Muet
	case len(lu.Victimes) > 1:
		return e193Ambigu
	case lu.Bot:
		return e193VictimeBot
	case victimeRecollee != "" && lu.Victimes[victimeRecollee]:
		return e193Accord
	default:
		return e193DesaccordVictime
	}
}

// e193Assigner : LE REGIME QUE LA CONVERSION EMPLOIERA, mesure ici avant d etre code.
//
// DEUX TEMPS, ET L ORDRE EST LE RESULTAT. La fenetre d appariement vaut 2,5 s de part et
// d autre : un joueur qui tue deux fois en 2,5 s y produit DEUX kill-events au meme nom de
// tueur, et une lecture qui ne regarderait que le tueur ne trancherait pas. On donne donc
// d abord a chaque couple ECRIT AU MEME INSTANT par le kill-feed le kill-event dont LE COUPLE
// ENTIER correspond — ce n est pas une decision, c est une CONSOMMATION —, puis on lit le
// couple des kills orphelins dans ce qui RESTE.
//
// Rend la ventilation des kills sans mort en face, par verdict.
func e193Assigner(kf *killFeed, recs []killEventRec, nom map[int]string, bot map[int]bool) map[e193Verdict]int {
	pris := make([]bool, len(recs))
	for _, e := range kf.real {
		if j := e193ChercherCouple(recs, pris, e.timeMS, e.killer, e.victim, nom); j >= 0 {
			pris[j] = true
		}
	}
	out := map[e193Verdict]int{}
	for i := range kf.events {
		e := kf.events[i]
		if e.killer == "" || e.victim != "" {
			continue
		}
		v, j := e193LireVictime(recs, pris, e.timeMS, e.killer, nom)
		if j >= 0 {
			pris[j] = true
		}
		out[e193VerdictAssigne(kf, i, v, nom, bot, j >= 0)]++
	}
	return out
}

// e193ChercherCouple : le premier kill-event NON PRIS de la fenetre dont les deux indices
// epingles nomment exactement ce couple. -1 quand il n y en a pas.
func e193ChercherCouple(recs []killEventRec, pris []bool, ms int, tueur, victime string, nom map[int]string) int {
	for i := range recs {
		if pris[i] {
			continue
		}
		if dt := recs[i].ms - ms; dt < -tolMS || dt > tolMS {
			continue
		}
		if nom[recs[i].fields.killer] == tueur && nom[recs[i].fields.victim] == victime {
			return i
		}
	}
	return -1
}

// e193LireVictime : la victime que le film ECRIT pour ce tueur a cet instant, prise dans les
// kill-events NON PRIS. Rend l indice de victime et celui du kill-event, ou (-1, -1) quand la
// lecture se tait ou ne tranche pas (deux victimes distinctes).
func e193LireVictime(recs []killEventRec, pris []bool, ms int, tueur string, nom map[int]string) (victime, rec int) {
	victime, rec = -1, -1
	for i := range recs {
		if pris[i] {
			continue
		}
		if dt := recs[i].ms - ms; dt < -tolMS || dt > tolMS {
			continue
		}
		nk, okk := nom[recs[i].fields.killer]
		nv, okv := nom[recs[i].fields.victim]
		if !okk || !okv || nk != tueur {
			continue
		}
		if rec >= 0 && nv != nom[victime] {
			return -2, -1 // deux victimes distinctes : la lecture ne tranche pas
		}
		victime, rec = recs[i].fields.victim, i
	}
	return victime, rec
}

// e193VerdictAssigne : le verdict d un kill sans mort en face, sous le regime d assignation.
func e193VerdictAssigne(kf *killFeed, i, victime int, nom map[int]string, bot map[int]bool, lu bool) e193Verdict {
	switch {
	case victime == -2:
		return e193Ambigu
	case !lu:
		return e193Muet
	case bot[victime]:
		return e193VictimeBot
	default:
		return e193VerdictHumain(kf, i, nom[victime])
	}
}

// e193VerdictHumain : le verdict quand le film nomme une victime HUMAINE — reste a savoir si le
// kill-feed porte la mort correspondante dans le voisinage, et si c est celle que le recollage
// aurait prise.
func e193VerdictHumain(kf *killFeed, i int, lue string) e193Verdict {
	recollee := ""
	for d := 1; d <= 2 && i+d < len(kf.events); d++ {
		if o := kf.events[i+d]; o.victim != "" && o.killer == "" {
			recollee = o.victim
			break
		}
	}
	switch {
	case recollee == lue:
		return e193Accord
	case e193MortAuVoisinage(kf, i, lue):
		return e193DesaccordVictime
	default:
		return e193SansMortAuFeed
	}
}

// e193MortAuVoisinage : le kill-feed porte-t-il une mort de ce joueur dans la fenetre ?
func e193MortAuVoisinage(kf *killFeed, i int, victime string) bool {
	for j := range kf.events {
		if dt := kf.events[j].timeMS - kf.events[i].timeMS; dt < -tolMS || dt > tolMS {
			continue
		}
		if kf.events[j].victim == victime {
			return true
		}
	}
	return false
}

// e193Mesurer : un film.
func e193Mesurer(dir string) e193Ligne {
	l := e193Ligne{Film: e193Nom(dir)}
	src, err := source.LoadDir(dir, nil)
	if err != nil {
		l.Err = err
		return l
	}
	f, err := loadFilm(src)
	if err != nil {
		l.Err = err
		return l
	}
	kf, err := loadKillFeed(f)
	if err != nil {
		l.Err = err
		return l
	}
	table := readFilmTable(f)
	r := buildRoster(kf, loadBotMeta(f), true, table)
	nom, bot := e193Epinglage(r)
	recs := scanKillEvents(f).recs

	l.Build, l.Refus = table.Build, table.Refusal
	l.Epingles, l.Bots = len(nom), len(bot)
	l.Instants, l.Kills, l.Morts = len(kf.events), kf.nKills, kf.nDeaths
	l.Pairs, l.Reels, l.Recolles = len(kf.pairs), len(kf.real), len(kf.fab)
	l.Orphelins, l.MortsOrph = len(kf.orphK), len(kf.orphD)
	l.KillEvents = len(recs)

	for _, pop := range []struct {
		nom  e193Population
		evts []feedEvent
	}{
		{e193Reel, kf.real}, {e193Recolle, kf.fab}, {e193Orphelin, kf.orphK},
	} {
		for _, e := range pop.evts {
			lu := e193LireCouple(recs, e.timeMS, e.killer, nom, bot)
			l.compte(pop.nom, e193Verdicter(lu, e.victim))
		}
	}
	for v, n := range e193Assigner(kf, recs, nom, bot) {
		for k := 0; k < n; k++ {
			l.compte(e193KillSansMort, v)
		}
	}
	return l
}

// TestE193CoupleEcritContreRecollage : LA MESURE. Aucune assertion de valeur — il imprime le
// tableau que le lot colle au §5 du plan.
func TestE193CoupleEcritContreRecollage(t *testing.T) {
	dirs := e193Films()
	lignes := make([]e193Ligne, 0, len(dirs))
	for _, d := range dirs {
		lignes = append(lignes, e193Mesurer(d))
	}
	sort.Slice(lignes, func(i, j int) bool { return lignes[i].Film < lignes[j].Film })

	t.Logf("==== [1] LE KILL-FEED ET SES COUPLES, FILM PAR FILM ====")
	t.Logf("%-10s %-12s %-14s %5s %5s %6s %6s %6s %6s %6s %6s %6s %7s",
		"film", "build", "table", "epin", "bots", "kills", "morts", "pairs", "meme-i",
		"recol", "orphK", "orphD", "ev85")
	var tp, tr, tf, to, td int
	for _, l := range lignes {
		if l.Err != nil {
			t.Logf("%-10s ERREUR : %v", l.Film, l.Err)
			continue
		}
		t.Logf("%-10s %-12s %-14s %5d %5d %6d %6d %6d %6d %6d %6d %6d %7d",
			l.Film, e193Court(l.Build), e193Court(string(l.Refus)), l.Epingles, l.Bots,
			l.Kills, l.Morts, l.Pairs, l.Reels, l.Recolles, l.Orphelins, l.MortsOrph, l.KillEvents)
		tp += l.Pairs
		tr += l.Reels
		tf += l.Recolles
		to += l.Orphelins
		td += l.MortsOrph
	}
	t.Logf("TOTAL : %d couples reconstruits, dont %d au MEME instant et %d RECOLLES ; "+
		"%d kill(s) orphelin(s), %d mort(s) orpheline(s)", tp, tr, tf, to, td)

	e193Tableau(t, "[2] LE COUPLE RECOLLE CONTRE LE COUPLE ECRIT (la population que le lot convertit)",
		lignes, e193Recolle)
	e193Tableau(t, "[3] LE COUPLE DU MEME INSTANT CONTRE LE COUPLE ECRIT (controle : rien ne doit bouger)",
		lignes, e193Reel)
	e193Tableau(t, "[4] LE KILL ORPHELIN CONTRE LE COUPLE ECRIT (aucun voisin consomme)",
		lignes, e193Orphelin)
	e193Tableau(t, "[5] LE REGIME D ASSIGNATION — TOUT KILL SANS MORT EN FACE, "+
		"les couples du meme instant ayant consomme leurs kill-events", lignes, e193KillSansMort)
}

// e193Court raccourcit une colonne du tableau, et rend un tiret pour le vide.
func e193Court(s string) string {
	if s == "" {
		return "-"
	}
	if len(s) > 14 {
		return s[:14]
	}
	return s
}

// e193Verdicts : l ordre des colonnes des tableaux [2] a [4].
var e193Verdicts = []e193Verdict{e193Accord, e193DesaccordVictime, e193VictimeBot,
	e193SansMortAuFeed, e193Ambigu, e193Muet}

// e193Tableau imprime la ventilation d une population par verdict.
func e193Tableau(t *testing.T, titre string, lignes []e193Ligne, p e193Population) {
	t.Logf("")
	t.Logf("==== %s ====", titre)
	entete := fmt.Sprintf("%-10s %6s", "film", "total")
	for _, v := range e193Verdicts {
		entete += fmt.Sprintf(" %18s", v)
	}
	t.Logf("%s", entete)
	cumul := map[e193Verdict]int{}
	total := 0
	for _, l := range lignes {
		if l.Err != nil {
			continue
		}
		n := 0
		for _, v := range e193Verdicts {
			n += l.n(p, v)
			cumul[v] += l.n(p, v)
		}
		if n == 0 {
			continue
		}
		total += n
		ligne := fmt.Sprintf("%-10s %6d", l.Film, n)
		for _, v := range e193Verdicts {
			ligne += fmt.Sprintf(" %18d", l.n(p, v))
		}
		t.Logf("%s", ligne)
	}
	ligne := fmt.Sprintf("%-10s %6d", "TOTAL", total)
	for _, v := range e193Verdicts {
		ligne += fmt.Sprintf(" %18d", cumul[v])
	}
	t.Logf("%s", ligne)
}
