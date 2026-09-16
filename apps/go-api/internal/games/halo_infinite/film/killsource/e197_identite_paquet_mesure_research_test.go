//go:build research

package killsource

// e197_identite_paquet_mesure_research_test.go — LOT 1.9.7, LA MESURE AVANT DE CODER.
//
// # LA QUESTION, POSEE AVANT TOUTE LIGNE DE PRODUCTION
//
// L appariement d une mort lue au DEAD-STATE avec une ligne du KILL-FEED se decide aujourd hui
// par une FENETRE TEMPORELLE de 2,5 s ([tolMS]), a cinq endroits ([decodeCtx.matchExact],
// [decodeCtx.matchVictim], [decodeCtx.apparierMortDeBot], [decodeCtx.resolveBotKillerDeaths],
// [pass.runUnclaimed]). La valeur n a jamais ete derivee d une mesure : son commentaire dit
// « valeur historique du chantier, employee par TOUTES les mesures publiees ».
//
// Or LES DEUX COTES PORTENT UNE IDENTITE DE PAQUET :
//
//	le dead-state    [candidate] porte `(chunk, pidx)` — des le scan (`scan.go`) comme depuis la
//	                 marche (`walk.go`, `candidates()`) ;
//	le kill-feed     un instant du feed n a que son horodatage... MAIS le lot 1.9.3 lui associe
//	                 deja un KILL-EVENT 85, qui porte `(chunk, pidx)` ([killEventRec],
//	                 `assist.go`). L association existe (`feed_couples.go`), elle est JETEE
//	                 (`pris[j] = true` et rien d autre).
//
// Avant de convertir, il faut CHIFFRER, film par film :
//
//	pour chaque appariement decide aujourd hui par la fenetre, l identite de paquet des deux
//	  cotes est-elle DISPONIBLE, EGALE, DIFFERENTE ?
//	ce que rendrait l appariement PAR IDENTITE SEULE : accord avec la fenetre, desaccord, muet.
//
// # CE QU IL NE FAIT PAS
//
// Aucune ecriture, aucune base, aucun artefact, aucun replay-equiv : il charge des films en
// lecture seule et compte. Il deroule en revanche la passe complete ([decodeCtx.prepare] puis
// [decodeCtx.run]) — la question porte sur les dead-states, donc sur la marche ET le scan.
//
// GARDE CHUNK00_FILMS (repertoires de film absolus separes par « ; ») pour les films entiers ;
// la bobine versionnee testdata/minibobine_000d5950 est mesuree sans garde.
//
//	CHUNK00_FILMS=<dir1>;<dir2> CGO_ENABLED=0 \
//	  go test -tags research ./internal/games/halo_infinite/film/killsource/ \
//	  -run TestE197IdentiteDePaquetContreFenetre -v -count=1 -timeout 3600s

import (
	"context"
	"fmt"
	"sort"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/source"
)

// e197Temps : le temps de l hybride qui a decide l appariement. Domaine FERME.
type e197Temps string

const (
	e197Fort      e197Temps = "1-2_couple_exact"
	e197Auto      e197Temps = "3_source_auto"
	e197MortBot   e197Temps = "4_mort_de_bot"
	e197TueurBot  e197Temps = "5_mort_par_bot"
	e197NonRevend e197Temps = "6_non_revendiquee"
)

// e197Ordre : l ordre d impression des temps.
var e197Ordre = []e197Temps{e197Fort, e197Auto, e197MortBot, e197TueurBot, e197NonRevend}

// e197Verdict : ce que l identite de paquet dit d un appariement decide par la fenetre.
type e197Verdict string

const (
	// e197IDEgale : les deux cotes portent la MEME identite de paquet. La fenetre et l identite
	// disent la meme chose ; l identite est simplement plus forte.
	e197IDEgale e197Verdict = "id_egale"
	// e197IDDifferente : les deux cotes portent une identite DIFFERENTE. Soit la fenetre
	// appariait faux, soit le dead-state et le kill-event ne vivent pas dans le meme paquet.
	e197IDDifferente e197Verdict = "id_differente"
	// e197IDAbsente : le cote feed ne porte aucune identite (aucun kill-event 85 ne lui est
	// associe). C est le diagnostic typé qui ouvrira le repli (D14 b).
	e197IDAbsente e197Verdict = "id_absente"
)

var e197Verdicts = []e197Verdict{e197IDEgale, e197IDDifferente, e197IDAbsente}

// e197Seul : ce que rendrait l appariement PAR IDENTITE SEULE, confronte a ce que la fenetre
// rend aujourd hui.
type e197Seul string

const (
	// e197Accord : l identite designe EXACTEMENT le dead-state que la fenetre a retenu.
	e197Accord e197Seul = "accord"
	// e197Desaccord : l identite designe un AUTRE dead-state.
	e197Desaccord e197Seul = "desaccord"
	// e197Ambigu : plusieurs dead-states du meme paquet satisfont la contrainte de couple.
	e197Ambigu e197Seul = "ambigu"
	// e197Muet : aucun dead-state du paquet designe ne satisfait la contrainte. L identite ne
	// decide pas — la fenetre reste le seul moyen.
	e197Muet e197Seul = "muet"
	// e197SansID : le cote feed n a pas d identite : l appariement par identite n est meme pas
	// tentable.
	e197SansID e197Seul = "sans_id"
)

var e197Seuls = []e197Seul{e197Accord, e197Desaccord, e197Ambigu, e197Muet, e197SansID}

// e197Ligne : la mesure d un film.
type e197Ligne struct {
	Film string
	Err  error
	// Instants : instants du kill-feed ; InstantsID : ceux auxquels un kill-event 85 est associe.
	Instants, InstantsID int
	// Appariements : par temps, puis par verdict / par ce que rendrait l identite seule.
	ParVerdict map[e197Temps]map[e197Verdict]int
	ParSeul    map[e197Temps]map[e197Seul]int
	// Detail : les appariements que l identite NE confirme PAS, nommes un par un — ce sont eux
	// que le corpus gate devra classer.
	Detail []string
}

func (l *e197Ligne) compte(t e197Temps, v e197Verdict, s e197Seul) {
	if l.ParVerdict == nil {
		l.ParVerdict, l.ParSeul = map[e197Temps]map[e197Verdict]int{}, map[e197Temps]map[e197Seul]int{}
	}
	if l.ParVerdict[t] == nil {
		l.ParVerdict[t], l.ParSeul[t] = map[e197Verdict]int{}, map[e197Seul]int{}
	}
	l.ParVerdict[t][v]++
	l.ParSeul[t][s]++
}

// e197Paquet : l identite de paquet, et si elle est disponible.
type e197Paquet struct {
	chunk, pidx int
	ok          bool
}

// e197Mesure : l etat d une mesure de film. Une structure plutot que six parametres.
type e197Mesure struct {
	c *decodeCtx
	p *pass
	// idFeed : l identite de paquet d un instant du feed, par `timeMS`. Elle vient du KILL-EVENT
	// 85 que [killFeed.resoudreCouples] lui a associe (lot 1.9.3).
	idFeed map[int]e197Paquet
	ligne  *e197Ligne
}

// e197AssocierLesKillEvents : LE MIROIR EN LECTURE SEULE de [killFeed.resoudreCouples].
//
// Il rejoue EXACTEMENT son ordre de consommation — les couples ecrits au meme instant d abord,
// les kills sans mort en face ensuite — et retient l identite de paquet du kill-event retenu au
// lieu de la jeter. `kf.events` n est pas modifie par la production : le rejeu est fidele.
func e197AssocierLesKillEvents(kf *killFeed, recs []killEventRec, r *roster) map[int]e197Paquet {
	out := map[int]e197Paquet{}
	pris := make([]bool, len(recs))
	garder := func(ms, j int) {
		pris[j] = true
		out[ms] = e197Paquet{chunk: recs[j].chunk, pidx: recs[j].pidx, ok: true}
	}
	for i := range kf.events {
		e := kf.events[i]
		if e.killer == "" || e.victim == "" {
			continue
		}
		if j := e197ChercherCoupleEcrit(recs, pris, r, e); j >= 0 {
			garder(e.timeMS, j)
		}
	}
	for i := range kf.events {
		e := kf.events[i]
		if e.killer == "" || e.victim != "" {
			continue
		}
		if j, ambigu := e197LireVictime(recs, pris, r, e); j >= 0 && !ambigu {
			garder(e.timeMS, j)
		}
	}
	return out
}

// e197ChercherCoupleEcrit : miroir de `chercherCoupleEcrit`.
func e197ChercherCoupleEcrit(recs []killEventRec, pris []bool, r *roster, e feedEvent) int {
	for i := range recs {
		if pris[i] || !dansLaFenetre(recs[i].ms, e.timeMS) {
			continue
		}
		k, okK := r.nomEpingle(recs[i].fields.killer)
		v, okV := r.nomEpingle(recs[i].fields.victim)
		if okK && okV && k == e.killer && v == e.victim {
			return i
		}
	}
	return -1
}

// e197LireVictime : miroir de `lireVictime` — il ne rend que l indice de l enregistrement.
func e197LireVictime(recs []killEventRec, pris []bool, r *roster, e feedEvent) (rec int, ambigu bool) {
	rec, victime := -1, -1
	for i := range recs {
		if pris[i] || !dansLaFenetre(recs[i].ms, e.timeMS) {
			continue
		}
		k, okK := r.nomEpingle(recs[i].fields.killer)
		if !okK || k != e.killer {
			continue
		}
		v := recs[i].fields.victim
		if _, okV := r.nomEpingle(v); !okV {
			continue
		}
		if rec >= 0 && v != victime {
			return -1, true
		}
		victime, rec = v, i
	}
	return rec, false
}

// e197Contrainte : la contrainte de COUPLE du temps qui a apparie. Elle est rejouee telle quelle
// sur les candidats du paquet designe par l identite — l identite remplace la FENETRE, jamais la
// contrainte de couple.
type e197Contrainte func(cd candidate) bool

// noter : un appariement decide par la fenetre, avec la contrainte qui l a produit.
func (m *e197Mesure) noter(t e197Temps, ms int, cd candidate, ok e197Contrainte) {
	id := m.idFeed[ms]
	if !id.ok {
		m.ligne.compte(t, e197IDAbsente, e197SansID)
		return
	}
	v := e197IDDifferente
	if id.chunk == cd.chunk && id.pidx == cd.pidx {
		v = e197IDEgale
	}
	s := m.parIdentiteSeule(id, cd, ok)
	m.ligne.compte(t, v, s)
	if v != e197IDEgale || s != e197Accord {
		m.ligne.Detail = append(m.ligne.Detail, fmt.Sprintf(
			"%-18s ms=%-8d feed=(%d,%d) dead=(%d,%d,bit %d) %s / %s",
			t, ms, id.chunk, id.pidx, cd.chunk, cd.pidx, cd.bit, v, s))
	}
}

// parIdentiteSeule : ce que rendrait l appariement PAR IDENTITE SEULE — les candidats du paquet
// designe qui satisfont la contrainte de couple, confrontes a celui que la fenetre a retenu.
func (m *e197Mesure) parIdentiteSeule(id e197Paquet, cd candidate, ok e197Contrainte) e197Seul {
	n, memeQueLaFenetre := 0, false
	for _, a := range m.p.all {
		if a.chunk != id.chunk || a.pidx != id.pidx || !ok(a.candidate) {
			continue
		}
		n++
		if a.chunk == cd.chunk && a.pidx == cd.pidx && a.bit == cd.bit {
			memeQueLaFenetre = true
		}
	}
	switch {
	case n == 0:
		return e197Muet
	case n > 1:
		return e197Ambigu
	case memeQueLaFenetre:
		return e197Accord
	default:
		return e197Desaccord
	}
}

// TestE197IdentiteDePaquetContreFenetre : LA MESURE. Aucune assertion de valeur — il imprime les
// tableaux que le lot colle au §5 du plan.
func TestE197IdentiteDePaquetContreFenetre(t *testing.T) {
	dirs := e193Films()
	lignes := make([]e197Ligne, 0, len(dirs))
	for _, d := range dirs {
		lignes = append(lignes, e197Mesurer(d))
	}
	sort.Slice(lignes, func(i, j int) bool { return lignes[i].Film < lignes[j].Film })
	e197TableauParFilm(t, lignes)
	e197TableauParTemps(t, lignes)
	e197TableauDetail(t, lignes)
}

// e197TableauDetail : tableau [3] — les appariements que l identite ne confirme pas, un par un.
func e197TableauDetail(t *testing.T, lignes []e197Ligne) {
	t.Logf("")
	t.Logf("==== [3] LES APPARIEMENTS QUE L IDENTITE NE CONFIRME PAS (hors id_absente) ====")
	n := 0
	for _, l := range lignes {
		for _, d := range l.Detail {
			t.Logf("%-14s %s", l.Film, d)
			n++
		}
	}
	t.Logf("TOTAL : %d appariement(s) a classer", n)
}

// e197Mesurer : un film. La passe complete, puis la confrontation.
func e197Mesurer(dir string) e197Ligne {
	l := e197Ligne{Film: e193Nom(dir)}
	src, err := source.LoadDir(dir, nil)
	if err != nil {
		l.Err = err
		return l
	}
	o := DefaultOptions()
	c := &decodeCtx{name: l.Film, opts: o}
	if err := c.prepare(context.Background(), src); err != nil {
		l.Err = err
		return l
	}
	m := &e197Mesure{c: c, p: c.run(), ligne: &l,
		idFeed: e197AssocierLesKillEvents(c.feed, c.killEvents.recs, c.roster)}
	l.Instants, l.InstantsID = len(c.feed.events), len(m.idFeed)
	m.temps12()
	m.temps3()
	m.temps4()
	m.temps5()
	m.temps6()
	return l
}

// e197ParLaFenetre : LE REGIME D AVANT LE LOT, FIGE ICI.
//
// Les fonctions de production apparient desormais par l IDENTITE DE PAQUET, la fenetre n etant
// qu un repli : les appeler ne mesurerait plus ce que cet instrument mesure. La demi-fenetre de
// 2,5 s est donc rejouee ici, et l instrument reste reproductible APRES la conversion — ce qui
// est toute la raison d etre d une mesure AVANT.
//
// Rend l indice du premier element de la fenetre satisfaisant la contrainte de couple, -1 sinon.
func e197ParLaFenetre(n, ms int, instant func(int) int, couple func(int) bool) int {
	for i := 0; i < n; i++ {
		dt := instant(i) - ms
		if dt < -tolMS || dt > tolMS {
			continue
		}
		if couple(i) {
			return i
		}
	}
	return -1
}

// temps12 : le critere FORT — couple exact, fenetre de 2,5 s.
func (m *e197Mesure) temps12() {
	for _, cd := range m.p.all {
		if cd.victim == cd.killer || m.c.isBotSide(cd.candidate) {
			continue
		}
		vic, kil := m.c.roster.nameOf(cd.victim), m.c.roster.nameOf(cd.killer)
		pairs := m.c.feed.pairs
		couple := func(a candidate) bool {
			return m.c.roster.nameOf(a.victim) == vic && m.c.roster.nameOf(a.killer) == kil
		}
		j := e197ParLaFenetre(len(pairs), cd.ms, func(i int) int { return pairs[i].timeMS },
			func(i int) bool { return pairs[i].victim == vic && pairs[i].killer == kil })
		if j >= 0 {
			m.noter(e197Fort, pairs[j].timeMS, cd.candidate, couple)
		}
	}
}

// temps3 : la source auto-infligee — victime seule, fenetre de 2,5 s.
//
// LES DEUX DISCRIMINANTS D ABORD : un candidat que [decodeCtx.selfSourceOK] rejette ne decide
// aucune ligne. Le compter ferait passer pour « apparie par la fenetre » ce que la passe jette.
func (m *e197Mesure) temps3() {
	for _, cd := range m.p.all {
		if cd.victim != cd.killer || !m.c.selfSourceOK(cd.candidate, m.p.mult) {
			continue
		}
		vic := m.c.roster.nameOf(cd.victim)
		pairs := m.c.feed.pairs
		couple := func(a candidate) bool {
			return a.victim == a.killer && m.c.roster.nameOf(a.victim) == vic
		}
		j := e197ParLaFenetre(len(pairs), cd.ms, func(i int) int { return pairs[i].timeMS },
			func(i int) bool { return pairs[i].victim == vic })
		if j >= 0 {
			m.noter(e197Auto, pairs[j].timeMS, cd.candidate, couple)
		}
	}
}

// e197PopulationDeBot : la population du temps 4, recopiee de [decodeCtx.resolveBotDeaths] —
// seul l APPARIEMENT est rejoue a la fenetre, la population ne bouge pas.
func (m *e197Mesure) e197PopulationDeBot() []botMatch {
	kf := m.c.feed
	out := make([]botMatch, 0, len(kf.orphK)+len(kf.fab)+len(kf.botLus))
	for _, b := range kf.botLus {
		out = append(out, botMatch{event: b.ev, victimeLue: b.victime})
	}
	for _, e := range kf.orphK {
		out = append(out, botMatch{event: e, victimeLue: -1})
	}
	for _, e := range kf.fab {
		out = append(out, botMatch{event: e, fab: true, victimeLue: -1})
	}
	return out
}

// temps4 : la mort DU bot.
func (m *e197Mesure) temps4() {
	for _, b := range m.e197PopulationDeBot() {
		cands := m.c.scanCands
		couple := func(a candidate) bool { return m.c.coupleDeMortDeBot(&b, a) }
		j := e197ParLaFenetre(len(cands), b.event.timeMS, func(i int) int { return cands[i].ms },
			func(i int) bool { return couple(cands[i]) })
		if j >= 0 {
			m.noter(e197MortBot, b.event.timeMS, cands[j], couple)
		}
	}
}

// temps5 : la mort infligee PAR un bot.
func (m *e197Mesure) temps5() {
	for _, e := range m.c.feed.orphD {
		vic := e.victim
		couple := func(a candidate) bool {
			return m.c.roster.isBotIndex(a.killer) && m.c.roster.nameOf(a.victim) == vic
		}
		j := e197ParLaFenetre(len(m.p.all), e.timeMS, func(i int) int { return m.p.all[i].ms },
			func(i int) bool { return couple(m.p.all[i].candidate) })
		if j >= 0 {
			m.noter(e197TueurBot, e.timeMS, m.p.all[j].candidate, couple)
		}
	}
}

// temps6 : les morts que personne ne revendique — le plus proche EN TEMPS gagne
// (`repli_mort_non_revendiquee_la_plus_proche`).
func (m *e197Mesure) temps6() {
	for _, e := range m.c.feed.orphD {
		best, bestDT, trouve := candidate{}, tolMS+1, false
		for _, cd := range m.p.all {
			dt := e.timeMS - cd.ms
			if dt < -tolMS || dt > tolMS {
				continue
			}
			if cd.victim != cd.killer || m.c.roster.nameOf(cd.victim) != e.victim {
				continue
			}
			if d := absMS(dt); d < bestDT {
				bestDT, best, trouve = d, cd.candidate, true
			}
		}
		if !trouve {
			continue
		}
		vic := e.victim
		m.noter(e197NonRevend, e.timeMS, best, func(a candidate) bool {
			return a.victim == a.killer && m.c.roster.nameOf(a.victim) == vic
		})
	}
}

// e197TableauParFilm : tableau [1] — l identite de paquet des deux cotes, film par film.
func e197TableauParFilm(t *testing.T, lignes []e197Ligne) {
	t.Logf("==== [1] L IDENTITE DE PAQUET DES DEUX COTES, FILM PAR FILM ====")
	t.Logf("%-14s %8s %8s %8s %10s %12s %10s", "film", "instants", "inst-id",
		"appar", "id_egale", "id_differente", "id_absente")
	var tot [4]int
	for _, l := range lignes {
		if l.Err != nil {
			t.Logf("%-14s ERREUR : %v", l.Film, l.Err)
			continue
		}
		var n [3]int
		for _, tp := range e197Ordre {
			for i, v := range e197Verdicts {
				n[i] += l.ParVerdict[tp][v]
			}
		}
		s := n[0] + n[1] + n[2]
		t.Logf("%-14s %8d %8d %8d %10d %12d %10d", l.Film, l.Instants, l.InstantsID, s, n[0], n[1], n[2])
		tot[0] += s
		for i := range n {
			tot[i+1] += n[i]
		}
	}
	t.Logf("%-14s %8s %8s %8d %10d %12d %10d", "TOTAL", "", "", tot[0], tot[1], tot[2], tot[3])
}

// e197TableauParTemps : tableau [2] — ce que rendrait l appariement PAR IDENTITE SEULE.
func e197TableauParTemps(t *testing.T, lignes []e197Ligne) {
	t.Logf("")
	t.Logf("==== [2] CE QUE RENDRAIT L APPARIEMENT PAR IDENTITE SEULE, PAR TEMPS DE L HYBRIDE ====")
	entete := fmt.Sprintf("%-18s %8s", "temps", "appar")
	for _, s := range e197Seuls {
		entete += fmt.Sprintf(" %10s", s)
	}
	t.Logf("%s", entete)
	cumul := map[e197Seul]int{}
	total := 0
	for _, tp := range e197Ordre {
		n, ligne := 0, map[e197Seul]int{}
		for _, l := range lignes {
			for _, s := range e197Seuls {
				ligne[s] += l.ParSeul[tp][s]
				cumul[s] += l.ParSeul[tp][s]
				n += l.ParSeul[tp][s]
			}
		}
		total += n
		txt := fmt.Sprintf("%-18s %8d", tp, n)
		for _, s := range e197Seuls {
			txt += fmt.Sprintf(" %10d", ligne[s])
		}
		t.Logf("%s", txt)
	}
	txt := fmt.Sprintf("%-18s %8d", "TOTAL", total)
	for _, s := range e197Seuls {
		txt += fmt.Sprintf(" %10d", cumul[s])
	}
	t.Logf("%s", txt)
}

// ==== MESURE 2 — L ASSISTANT : QUEL KILL-EVENT 85 DECRIT CETTE MORT ? ====
//
// [decodeCtx.killEventsFor] apparie une ligne PUBLIEE aux kill-events 85 par la MEME fenetre de
// 2,5 s, et c est ce qui remplit l assistant et les DEUX PARTS DE DEGATS. La question est la
// meme qu a la mesure 1, du cote de la sortie : le paquet du dead-state qui a produit la ligne
// designe-t-il les memes enregistrements que la fenetre ?

// e197AssistLigne : la mesure d un film, pour l appariement de l assistant.
type e197AssistLigne struct {
	Film  string
	Err   error
	Kills int
	// MemePick : le kill-event RETENU (apres [pickAssistHit]) est le meme des deux cotes.
	MemePick int
	// AutrePick : les deux cotes retiennent des enregistrements DIFFERENTS.
	AutrePick int
	// IDMuette : le paquet de la ligne ne porte aucun kill-event satisfaisant le couple, la
	// fenetre si — le repli sert.
	IDMuette int
	// FenetreMuette : la fenetre ne trouve rien non plus. Rien a comparer.
	FenetreMuette int
	// IDSeule : l identite trouve la ou la fenetre ne trouvait rien (gain net).
	IDSeule int
}

// e197HitsParFenetre : le regime d AVANT, fige ici (miroir de `killEventsFor` avant le lot).
func e197HitsParFenetre(c *decodeCtx, k *Kill, recs []killEventRec, used []bool) []int {
	var hits []int
	for j := range recs {
		if used[j] || !dansLaFenetre(recs[j].ms, k.TimeMS) || !e197CoupleAssist(c, k, recs[j]) {
			continue
		}
		hits = append(hits, j)
	}
	return hits
}

// e197HitsParIdentite : les kill-events du PAQUET de la ligne qui satisfont le meme couple.
func e197HitsParIdentite(c *decodeCtx, k *Kill, recs []killEventRec, used []bool) []int {
	var hits []int
	for j := range recs {
		if used[j] || !k.paquet.memeQue(recs[j].chunk, recs[j].pidx) || !e197CoupleAssist(c, k, recs[j]) {
			continue
		}
		hits = append(hits, j)
	}
	return hits
}

// e197CoupleAssist : la contrainte de couple de `killEventsFor`, inchangee.
func e197CoupleAssist(c *decodeCtx, k *Kill, r killEventRec) bool {
	if c.roster.nameOf(r.fields.victim) != k.Victim {
		return false
	}
	return !k.Feed.Present || c.roster.nameOf(r.fields.killer) == k.Feed.Killer
}

// e197MesurerAssist : un film.
func e197MesurerAssist(dir string) e197AssistLigne {
	l := e197AssistLigne{Film: e193Nom(dir)}
	src, err := source.LoadDir(dir, nil)
	if err != nil {
		l.Err = err
		return l
	}
	c := &decodeCtx{name: l.Film, opts: DefaultOptions()}
	if err := c.prepare(context.Background(), src); err != nil {
		l.Err = err
		return l
	}
	kills := c.run().kills()
	l.Kills = len(kills)
	recs := c.killEvents.recs
	usedF, usedI := make([]bool, len(recs)), make([]bool, len(recs))
	for i := range kills {
		e197ComparerUneLigne(&l, c, &kills[i], recs, usedF, usedI)
	}
	return l
}

// e197ComparerUneLigne : les deux regimes sur une ligne publiee, chacun avec SON `used` — comme
// en production, ou un enregistrement retenu ne ressert pas.
func e197ComparerUneLigne(l *e197AssistLigne, c *decodeCtx, k *Kill, recs []killEventRec, usedF, usedI []bool) {
	hf := e197HitsParFenetre(c, k, recs, usedF)
	hi := e197HitsParIdentite(c, k, recs, usedI)
	pf, pi := -1, -1
	if len(hf) > 0 {
		pf, _ = pickAssistHit(recs, hf)
		usedF[pf] = true
	}
	if len(hi) > 0 {
		pi, _ = pickAssistHit(recs, hi)
		usedI[pi] = true
	}
	switch {
	case pf < 0 && pi < 0:
		l.FenetreMuette++
	case pf < 0:
		l.IDSeule++
	case pi < 0:
		l.IDMuette++
	case pf == pi:
		l.MemePick++
	default:
		l.AutrePick++
	}
}

// TestE197AssistantParIdentiteDePaquet : LA MESURE 2. Aucune assertion de valeur.
func TestE197AssistantParIdentiteDePaquet(t *testing.T) {
	dirs := e193Films()
	lignes := make([]e197AssistLigne, 0, len(dirs))
	for _, d := range dirs {
		lignes = append(lignes, e197MesurerAssist(d))
	}
	sort.Slice(lignes, func(i, j int) bool { return lignes[i].Film < lignes[j].Film })

	t.Logf("==== [4] L ASSISTANT : LE PAQUET DE LA LIGNE CONTRE LA FENETRE DE 2,5 s ====")
	t.Logf("%-14s %7s %10s %10s %10s %12s %9s", "film", "lignes", "meme-pick", "autre-pick",
		"id-muette", "fen-muette", "id-seule")
	var tot [6]int
	for _, l := range lignes {
		if l.Err != nil {
			t.Logf("%-14s ERREUR : %v", l.Film, l.Err)
			continue
		}
		t.Logf("%-14s %7d %10d %10d %10d %12d %9d", l.Film, l.Kills, l.MemePick, l.AutrePick,
			l.IDMuette, l.FenetreMuette, l.IDSeule)
		for i, v := range []int{l.Kills, l.MemePick, l.AutrePick, l.IDMuette, l.FenetreMuette, l.IDSeule} {
			tot[i] += v
		}
	}
	t.Logf("%-14s %7d %10d %10d %10d %12d %9d", "TOTAL", tot[0], tot[1], tot[2], tot[3], tot[4], tot[5])
}
