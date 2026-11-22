//go:build research

package replayartifacts

// medailles_bouclier_research_test.go — INSTRUMENT des mesures de bouclier et de vengeance du parc.
//
// TestMedaillesBouclierBris : par vie (une piste nommée d'un document de rejeu), les bris de
// bouclier (la mesure `Sh` passe d'une valeur > 0 à 0, la première mesure de la vie à 0 compte) et
// les recharges complètes (après un bris, `Sh` >= 1 dans la même vie, une au plus par bris) ;
// l'issue de chaque bris ; la distribution par joueur et par match ; la part des documents sans
// aucune mesure. Écrit, si MEDBOUCLIER_SORTIE est posé, une ligne `match_id<TAB>xuid<TAB>bris<TAB>
// recharges` par joueur et par match (entrée de TestImpactBareme, paquet platform/duckdb).
//
// TestMedaillesVengeur : la vengeance (un frag de K sur V quand V est le tueur de la dernière mort
// de K, une vengeance par mort) rejouée sur le fil des morts, comparée aux lignes héritées de la
// médaille 9000000001.
//
// LECTURE SEULE, SUR UNE COPIE de la base partagée ; les documents sont lus un par un.
//
//	MEDBOUCLIER_ARTEFACTS=<dossier des rejeux> MEDBOUCLIER_SHARED=<copie shared_matches_v2.duckdb> \
//	MEDBOUCLIER_SORTIE=<fichier tsv> \
//	  go test -tags research -count=1 -run '^TestMedailles' -v ./internal/sync/replayartifacts/

import (
	"bufio"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"github.com/JGtm/halo-infinite-film-decoder/replay"
)

var medBouclierDocRe = regexp.MustCompile(`^([0-9a-f]{8})\.json(\.gz)?$`)

// medBouclierSansIssueMS : au-delà de ce délai sans recharge complète, un bris est « sans issue ».
const medBouclierSansIssueMS = 10000

// Les issues d'un bris. L'horizon d'un bris est le bris suivant de la vie, ou la fin de la vie.
const (
	issueRecharge     = "recharge complete"
	issueMortCourt    = "mort dans les 10 s"
	issueMortLong     = "mort apres plus de 10 s sans recharge"
	issueFinCourt     = "fin de vie sans mort dans les 10 s"
	issueFinLong      = "fin de vie sans mort apres plus de 10 s sans recharge"
	issueRebrisCourt  = "nouveau bris dans les 10 s (remontee partielle)"
	issueRebrisLong   = "nouveau bris apres plus de 10 s (remontee partielle)"
	medBouclierVengID = 9000000001
)

type medBouclierJoueur struct{ bris, recharges int }

type medBouclierBilan struct {
	docs, docsSansSh, docsSansOrigine int
	vies, viesAvecSh                  int
	issues                            map[string]int
	issuesSeuil95                     map[string]int // meme classement, recharge = Sh >= 0,95
	delaisRecharge                    []int          // ms entre le bris et la recharge complete
	maxApresLong                      []float64      // Sh maximal revu apres un bris sans issue
	parJoueur                         []medBouclierJoueur
	brisParMatch, rechargesParMatch   []int
	premiereMesureZero                int
	zeroNaissance                     int // dont : mesure dans la seconde qui suit la naissance de la vie
	docsSansBouclier                  int // documents dont au moins 80 % des bris sont une premiere mesure a 0
	brisDocsSansBouclier              int
	rechargesDocsSansBouclier         int
}

func TestMedaillesBouclierBris(t *testing.T) {
	dir, shared := os.Getenv("MEDBOUCLIER_ARTEFACTS"), os.Getenv("MEDBOUCLIER_SHARED")
	if dir == "" || shared == "" {
		t.Skip("MEDBOUCLIER_ARTEFACTS et MEDBOUCLIER_SHARED requis")
	}
	db, err := sql.Open("duckdb", shared+"?access_mode=read_only")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var sortie *bufio.Writer
	if p := os.Getenv("MEDBOUCLIER_SORTIE"); p != "" {
		f, err := os.Create(p)
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		sortie = bufio.NewWriter(f)
		defer sortie.Flush()
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	b := medBouclierBilan{issues: map[string]int{}, issuesSeuil95: map[string]int{}}
	for _, e := range entries {
		if medBouclierDocRe.FindStringSubmatch(e.Name()) == nil {
			continue
		}
		doc, err := LireArtefactRange(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Logf("%s : %v", e.Name(), err)
			continue
		}
		b.docs++
		if doc.OriginMs == nil || doc.FrameIntervalMS <= 0 {
			b.docsSansOrigine++
			continue
		}
		morts := medBouclierMorts(t, db, doc.MatchID)
		zeroAvant := b.premiereMesureZero
		joueurs := medBouclierMesurerDoc(doc, morts, &b)
		if joueurs == nil {
			b.docsSansSh++
			continue
		}
		totB, totR := 0, 0
		xuids := make([]string, 0, len(joueurs))
		for x := range joueurs {
			xuids = append(xuids, x)
		}
		sort.Strings(xuids)
		for _, x := range xuids {
			j := joueurs[x]
			b.parJoueur = append(b.parJoueur, *j)
			totB += j.bris
			totR += j.recharges
			if sortie != nil {
				fmt.Fprintf(sortie, "%s\t%s\t%d\t%d\n", doc.MatchID, x, j.bris, j.recharges)
			}
		}
		if z := b.premiereMesureZero - zeroAvant; totB > 0 && z*5 >= totB*4 {
			b.docsSansBouclier++
			b.brisDocsSansBouclier += totB
			b.rechargesDocsSansBouclier += totR
		}
		b.brisParMatch = append(b.brisParMatch, totB)
		b.rechargesParMatch = append(b.rechargesParMatch, totR)
	}
	medBouclierRapport(t, &b)
}

func medBouclierRapport(t *testing.T, b *medBouclierBilan) {
	t.Helper()
	t.Logf("documents lus : %d ; sans origine : %d ; sans aucune mesure de bouclier : %d (%.1f %%)",
		b.docs, b.docsSansOrigine, b.docsSansSh, pct(b.docsSansSh, b.docs))
	t.Logf("vies nommees : %d ; avec au moins une mesure : %d (%.1f %%)", b.vies, b.viesAvecSh, pct(b.viesAvecSh, b.vies))
	ordre := []string{issueRecharge, issueMortCourt, issueMortLong, issueFinCourt, issueFinLong, issueRebrisCourt, issueRebrisLong}
	for _, jeu := range []struct {
		nom string
		m   map[string]int
	}{{"seuil 1,00", b.issues}, {"seuil 0,95", b.issuesSeuil95}} {
		total := 0
		for _, n := range jeu.m {
			total += n
		}
		t.Logf("--- issues des bris (%s) : %d bris", jeu.nom, total)
		for _, k := range ordre {
			t.Logf("    %-55s %7d  %5.1f %%", k, jeu.m[k], pct(jeu.m[k], total))
		}
		long := jeu.m[issueMortLong] + jeu.m[issueFinLong] + jeu.m[issueRebrisLong]
		t.Logf("    GATE : sans issue alors que la vie continue plus de 10 s : %d (%.1f %%)", long, pct(long, total))
	}
	t.Logf("bris dont la premiere mesure de la vie est deja a 0 : %d, dont dans la seconde qui suit la naissance : %d", b.premiereMesureZero, b.zeroNaissance)
	t.Logf("documents dont au moins 80 %% des bris sont une premiere mesure a 0 (mode sans bouclier) : %d ; ils portent %d bris et %d recharges",
		b.docsSansBouclier, b.brisDocsSansBouclier, b.rechargesDocsSansBouclier)
	sort.Ints(b.delaisRecharge)
	if n := len(b.delaisRecharge); n > 0 {
		t.Logf("delai bris -> recharge complete (ms) : p10 %d, mediane %d, p90 %d, p99 %d, max %d",
			b.delaisRecharge[n/10], b.delaisRecharge[n/2], b.delaisRecharge[n*9/10], b.delaisRecharge[n*99/100], b.delaisRecharge[n-1])
	}
	sort.Float64s(b.maxApresLong)
	if n := len(b.maxApresLong); n > 0 {
		t.Logf("bouclier maximal revu apres un bris sans issue : p10 %.2f, mediane %.2f, p90 %.2f (n=%d)",
			b.maxApresLong[n/10], b.maxApresLong[n/2], b.maxApresLong[n*9/10], n)
	}
	bris := make([]int, len(b.parJoueur))
	rech := make([]int, len(b.parJoueur))
	for i, j := range b.parJoueur {
		bris[i], rech[i] = j.bris, j.recharges
	}
	t.Logf("par joueur et par match (n=%d) — bris : %s", len(bris), medBouclierQuantiles(bris))
	t.Logf("par joueur et par match (n=%d) — recharges (Cible difficile) : %s", len(rech), medBouclierQuantiles(rech))
	t.Logf("par match (n=%d) — bris : %s", len(b.brisParMatch), medBouclierQuantiles(b.brisParMatch))
	t.Logf("par match (n=%d) — recharges : %s", len(b.rechargesParMatch), medBouclierQuantiles(b.rechargesParMatch))
}

func medBouclierQuantiles(v []int) string {
	if len(v) == 0 {
		return "aucune valeur"
	}
	s := append([]int(nil), v...)
	sort.Ints(s)
	somme, zeros := 0, 0
	for _, x := range s {
		somme += x
		if x == 0 {
			zeros++
		}
	}
	n := len(s)
	return fmt.Sprintf("moyenne %.2f, p25 %d, mediane %d, p75 %d, p90 %d, p99 %d, max %d, a zero %.1f %%",
		float64(somme)/float64(n), s[n/4], s[n/2], s[n*3/4], s[n*9/10], s[n*99/100], s[n-1], pct(zeros, n))
}

// medBouclierMorts rend, par xuid, les instants (ms, horloge du film) de ses morts.
func medBouclierMorts(t *testing.T, db *sql.DB, matchID string) map[string][]int {
	t.Helper()
	out := map[string][]int{}
	if matchID == "" {
		return out
	}
	rows, err := db.Query(`SELECT time_ms, victim_xuid FROM match_kill_events_latest
		WHERE match_id = ? AND victim_xuid IS NOT NULL ORDER BY time_ms`, matchID)
	if err != nil {
		t.Fatalf("%s : %v", matchID, err)
	}
	defer rows.Close()
	for rows.Next() {
		var ms int
		var v string
		if err := rows.Scan(&ms, &v); err != nil {
			t.Fatal(err)
		}
		out[v] = append(out[v], ms)
	}
	return out
}

// medBouclierMesurerDoc classe les bris du document ; nil si le document ne porte aucune mesure.
func medBouclierMesurerDoc(doc *replay.ReplayDocument, morts map[string][]int, b *medBouclierBilan) map[string]*medBouclierJoueur {
	pas, origine := doc.FrameIntervalMS, int(*doc.OriginMs)
	avecSh := false
	joueurs := map[string]*medBouclierJoueur{}
	for i := range doc.Tracks {
		tr := &doc.Tracks[i]
		if tr.XUID == "" {
			continue
		}
		b.vies++
		if joueurs[tr.XUID] == nil {
			joueurs[tr.XUID] = &medBouclierJoueur{}
		}
		fin := citBouclierFin(tr)
		mort := tr.EndReason == "released"
		for _, ms := range morts[tr.XUID] {
			if d := (ms-origine)/pas - fin; d >= -45 && d <= 45 {
				mort = true
			}
		}
		nb, nr, vu := medBouclierVie(tr, fin, pas, mort, 1.0, b.issues, b)
		medBouclierVie(tr, fin, pas, mort, 0.95, b.issuesSeuil95, nil)
		if vu {
			avecSh = true
			b.viesAvecSh++
		}
		joueurs[tr.XUID].bris += nb
		joueurs[tr.XUID].recharges += nr
	}
	if !avecSh {
		return nil
	}
	return joueurs
}

// medBouclierVie parcourt les mesures d'une vie : rend ses bris, ses recharges et si elle porte une
// mesure. `detail` non nil reçoit les délais et les maxima.
func medBouclierVie(tr *replay.Track, fin, pas int, mort bool, seuil float32, issues map[string]int, detail *medBouclierBilan) (int, int, bool) {
	bris, recharges, vu := 0, 0, false
	var prec *float32
	enAttente, frameBris := false, 0
	var maxApres float32
	clore := func(frameHorizon int, rebris bool) {
		long := (frameHorizon-frameBris)*pas > medBouclierSansIssueMS
		var k string
		switch {
		case rebris && long:
			k = issueRebrisLong
		case rebris:
			k = issueRebrisCourt
		case mort && long:
			k = issueMortLong
		case mort:
			k = issueMortCourt
		case long:
			k = issueFinLong
		default:
			k = issueFinCourt
		}
		issues[k]++
		if long && detail != nil {
			detail.maxApresLong = append(detail.maxApresLong, float64(maxApres))
		}
	}
	for j := range tr.Points {
		p := &tr.Points[j]
		if p.Sh == nil {
			continue
		}
		vu = true
		sh := *p.Sh
		switch {
		case sh == 0 && (prec == nil || *prec > 0):
			if enAttente {
				clore(p.T, true)
			}
			if prec == nil && detail != nil {
				detail.premiereMesureZero++
				if (p.T-citBouclierDebut(tr))*pas <= 1000 {
					detail.zeroNaissance++
				}
			}
			bris++
			enAttente, frameBris, maxApres = true, p.T, 0
		case enAttente && sh >= seuil:
			recharges++
			enAttente = false
			issues[issueRecharge]++
			if detail != nil {
				detail.delaisRecharge = append(detail.delaisRecharge, (p.T-frameBris)*pas)
			}
		case enAttente && sh > maxApres:
			maxApres = sh
		}
		prec = p.Sh
	}
	if enAttente {
		clore(fin, false)
	}
	return bris, recharges, vu
}

// ─── Vengeur ────────────────────────────────────────────────────────────────────────────────────

type medVengeurFrag struct {
	match, tueur, victime string
}

func TestMedaillesVengeur(t *testing.T) {
	shared := os.Getenv("MEDBOUCLIER_SHARED")
	if shared == "" {
		t.Skip("MEDBOUCLIER_SHARED requis")
	}
	db, err := sql.Open("duckdb", shared+"?access_mode=read_only")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	equipes := map[string]int{} // match|xuid -> team_id
	rows, err := db.Query(`SELECT match_id, xuid, COALESCE(team_id, -1) FROM match_participants WHERE xuid IS NOT NULL AND xuid <> ''`)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var m, x string
		var e int
		if err := rows.Scan(&m, &x, &e); err != nil {
			t.Fatal(err)
		}
		equipes[m+"|"+x] = e
	}
	rows.Close()

	rows, err = db.Query(`SELECT match_id, COALESCE(feed_killer_xuid, ''), COALESCE(victim_xuid, '')
		FROM match_kill_events_latest ORDER BY match_id, time_ms, id`)
	if err != nil {
		t.Fatal(err)
	}
	derive := map[string]int{} // match|xuid -> vengeances
	matchs := map[string]bool{}
	var courant string
	dernierTueur := map[string]string{}
	frags, suicides, trahisons, bots, sansEquipe := 0, 0, 0, 0, 0
	for rows.Next() {
		var f medVengeurFrag
		if err := rows.Scan(&f.match, &f.tueur, &f.victime); err != nil {
			t.Fatal(err)
		}
		if f.match != courant {
			courant, dernierTueur = f.match, map[string]string{}
			matchs[f.match] = true
		}
		frags++
		valide := true
		switch {
		case f.tueur == "" || f.victime == "":
			bots++
			valide = false
		case f.tueur == f.victime:
			suicides++
			valide = false
		default:
			et, okT := equipes[f.match+"|"+f.tueur]
			ev, okV := equipes[f.match+"|"+f.victime]
			if !okT || !okV {
				sansEquipe++
			} else if et == ev && et >= 0 {
				trahisons++
				valide = false
			}
		}
		if valide && dernierTueur[f.tueur] == f.victime {
			derive[f.match+"|"+f.tueur]++
			delete(dernierTueur, f.tueur) // la mort est consommee
		}
		if f.victime != "" {
			if valide {
				dernierTueur[f.victime] = f.tueur
			} else {
				delete(dernierTueur, f.victime) // derniere mort sans tueur vengeable
			}
		}
	}
	rows.Close()
	t.Logf("fil des morts : %d matchs, %d lignes ; ecartees : %d bot ou non resolu, %d suicides, %d trahisons ; %d sans equipe connue (gardees)",
		len(matchs), frags, bots, suicides, trahisons, sansEquipe)

	// Distribution par joueur et par match : tous les participants nommes des matchs du fil.
	var parJoueur []int
	parMatch := map[string]int{}
	for cle := range equipes {
		m := cle[:36]
		if !matchs[m] {
			continue
		}
		parJoueur = append(parJoueur, derive[cle])
		parMatch[m] += derive[cle]
	}
	pm := make([]int, 0, len(parMatch))
	for _, n := range parMatch {
		pm = append(pm, n)
	}
	t.Logf("Vengeur derive, par joueur et par match (n=%d) : %s", len(parJoueur), medBouclierQuantiles(parJoueur))
	t.Logf("Vengeur derive, par match (n=%d) : %s", len(pm), medBouclierQuantiles(pm))

	// Lignes heritees.
	herite := map[string]int{}
	matchsHerites := map[string]bool{}
	rows, err = db.Query(`SELECT match_id, xuid, count FROM medals_earned WHERE medal_name_id = ?`, medBouclierVengID)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var m, x string
		var n int
		if err := rows.Scan(&m, &x, &n); err != nil {
			t.Fatal(err)
		}
		herite[m+"|"+x] = n
		matchsHerites[m] = true
	}
	rows.Close()
	medVengeurDates(t, db)
	communs, heritesSeuls := 0, 0
	for m := range matchsHerites {
		if matchs[m] {
			communs++
		} else {
			heritesSeuls++
		}
	}
	t.Logf("lignes heritees %d : %d lignes sur %d matchs ; matchs avec les deux sources : %d ; herites sans fil des morts : %d ; fil des morts sans ligne heritee : %d",
		medBouclierVengID, len(herite), len(matchsHerites), communs, heritesSeuls, len(matchs)-communs)
	egal, plus, moins, sommeH, sommeD := 0, 0, 0, 0, 0
	ecarts := map[int]int{}
	couples := map[[2]int]int{}
	for cle := range equipes {
		m := cle[:36]
		if !matchs[m] || !matchsHerites[m] {
			continue
		}
		h, d := herite[cle], derive[cle]
		sommeH += h
		sommeD += d
		ecarts[d-h]++
		couples[[2]int{h, d}]++
		switch {
		case d == h:
			egal++
		case d > h:
			plus++
		default:
			moins++
		}
	}
	n := egal + plus + moins
	t.Logf("comparaison sur les matchs communs, par joueur (n=%d) : egal %d (%.1f %%), derive > herite %d, derive < herite %d ; total herite %d, total derive %d",
		n, egal, pct(egal, n), plus, moins, sommeH, sommeD)
	medVengeurCouples(t, couples)
	cles := make([]int, 0, len(ecarts))
	for k := range ecarts {
		cles = append(cles, k)
	}
	sort.Ints(cles)
	for _, k := range cles {
		t.Logf("    ecart derive - herite = %+d : %d", k, ecarts[k])
	}
}

// medVengeurCouples journalise les couples (herite, derive) les plus frequents.
func medVengeurCouples(t *testing.T, couples map[[2]int]int) {
	t.Helper()
	type c struct {
		hd [2]int
		n  int
	}
	var l []c
	for k, n := range couples {
		l = append(l, c{k, n})
	}
	sort.Slice(l, func(i, j int) bool {
		if l[i].n != l[j].n {
			return l[i].n > l[j].n
		}
		return l[i].hd[0]*1000+l[i].hd[1] < l[j].hd[0]*1000+l[j].hd[1]
	})
	for i := 0; i < len(l) && i < 30; i++ {
		t.Logf("    herite %d, derive %d : %d joueurs-matchs", l[i].hd[0], l[i].hd[1], l[i].n)
	}
}

// medVengeurDates situe dans le temps les matchs du fil des morts, avec et sans ligne heritee.
func medVengeurDates(t *testing.T, db *sql.DB) {
	t.Helper()
	rows, err := db.Query(`WITH fil AS (SELECT DISTINCT match_id FROM match_kill_events_latest),
		her AS (SELECT DISTINCT match_id FROM medals_earned WHERE medal_name_id = ?)
		SELECT (h.match_id IS NOT NULL) AS herite, count(*), CAST(min(r.start_time) AS VARCHAR), CAST(max(r.start_time) AS VARCHAR)
		FROM fil f JOIN match_registry r ON r.match_id = f.match_id LEFT JOIN her h ON h.match_id = f.match_id
		GROUP BY 1 ORDER BY 1`, medBouclierVengID)
	if err != nil {
		t.Logf("dates des matchs : %v", err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var herite bool
		var n int
		var min, max sql.NullString
		if err := rows.Scan(&herite, &n, &min, &max); err != nil {
			t.Fatal(err)
		}
		t.Logf("matchs du fil des morts, ligne heritee=%v : %d, du %s au %s", herite, n, min.String, max.String)
	}
}
