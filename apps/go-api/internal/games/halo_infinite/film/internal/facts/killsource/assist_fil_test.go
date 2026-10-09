package killsource

// assist_fil_test.go — L ASSISTANT DES MORTS SANS KILL-EVENT, LU AU FIL DES EVENEMENTS
// (`assist_fil.go`). Chaque test nomme la mutation qui doit le faire rougir.

import "testing"

// Les types d evenement des tests : celui que le jeu adresse a l assistant, et celui qu il adresse au
// tueur. Leurs valeurs ne disent rien : le type d assistance se lit par film.
const (
	typeAssistance = 70
	typeAuTueur    = 68
)

// destinatairesFixes : un message du fil adresse aux participants donnes.
type destinatairesFixes []int

func (d destinatairesFixes) Destine(i int) bool {
	for _, v := range d {
		if v == i {
			return true
		}
	}
	return false
}

// rosterAuFil : K (0), V (1) et W (2) epingles par la table du film, les bots A (3) et B (4) par
// BOT_METADATA, et un nom de remplissage sur l indice 5.
func rosterAuFil() *roster {
	return &roster{
		names:   []string{"K", "V", "W", "343 A" + BotSuffix, "343 B" + BotSuffix, "?5"},
		perm:    []int{0, 1, 2, 3, 4, 5},
		pin:     map[int]int{0: 0, 1: 1, 2: 2, 3: 3, 4: 4},
		seatPin: map[int]bool{0: true, 1: true, 2: true},
		nPlay:   6,
	}
}

// messageDuFil : un message du fil de la trame (1, pidx).
func messageDuFil(pidx int, typ uint32, tueur, victime int, complet bool, dest ...int) filRec {
	return filRec{chunk: 1, pidx: pidx, complet: complet, typ: typ, tueur: tueur, victime: victime,
		evenement: destinatairesFixes(dest)}
}

// contexteAuFil : un decodage dont les kill-events sont `recs` et le fil `msgs`.
func contexteAuFil(recs []killEventRec, msgs ...filRec) *decodeCtx {
	f := &filScan{parPaquet: map[[2]int][]filRec{}}
	for _, m := range msgs {
		k := [2]int{m.chunk, m.pidx}
		f.parPaquet[k] = append(f.parPaquet[k], m)
	}
	return &decodeCtx{roster: rosterAuFil(), film: &film{}, killEvents: &assistScan{recs: recs}, fil: f}
}

// killEventAuFil : le kill-event de K sur V, assiste par W, dans la trame (1, 1) — ce qui apprend
// au film son type d assistance.
func killEventAuFil() killEventRec {
	return killEventRec{chunk: 1, pidx: 1, fields: killEventFields{killer: 0, victim: 1, assist: 2}}
}

// mortDeBot : une mort du bot `victime`, creditee a K, dont le dead-state est dans la trame (1, pidx).
func mortDeBot(victime string, pidx int) Kill {
	return Kill{TimeMS: pidx * 1000, Victim: victime, Feed: FeedTruth{Killer: "K"},
		paquet: paquetID{chunk: 1, pidx: pidx, ok: true}}
}

// TestLAssistantDUneMortDeBotSeLitAuFil — le film apprend son type d assistance du kill-event de
// (1, 1), puis lit les morts de bot :
//
//	(1, 2)  assistance adressee a W            W, lu au fil
//	(1, 3)  fil du couple sans assistance      PAS d assistant, mesure (trame entiere)
//	(1, 4)  idem, trame arretee                on ne sait pas
//	(1, 5)  assistance adressee a W et au bot A W nomme, `Extra` a zero, multiple compte
//	(1, 6)  assistance adressee au bot B       le bot B
//	(1, 7)  aucun message au couple            on ne sait pas
//
// MUTATIONS QUI DOIVENT LE FAIRE ROUGIR : retirer la garde `complet` de [decodeCtx.lireLAssistantDuFil]
// (la trame arretee publie « pas d assistant ») ; comparer le couple sans la victime (la mort de
// (1, 7) prend le fil d une autre).
func TestLAssistantDUneMortDeBotSeLitAuFil(t *testing.T) {
	a, b := "343 A"+BotSuffix, "343 B"+BotSuffix
	c := contexteAuFil([]killEventRec{killEventAuFil()},
		messageDuFil(1, typeAssistance, 0, 1, true, 2), messageDuFil(1, typeAuTueur, 0, 1, true, 0),
		messageDuFil(2, typeAuTueur, 0, 3, true, 0), messageDuFil(2, typeAssistance, 0, 3, true, 2),
		messageDuFil(3, typeAuTueur, 0, 3, true, 0),
		messageDuFil(4, typeAuTueur, 0, 3, false, 0),
		messageDuFil(5, typeAssistance, 0, 4, true, 2), messageDuFil(5, typeAssistance, 0, 4, true, 3),
		messageDuFil(6, typeAssistance, 0, 3, true, 4),
		messageDuFil(7, typeAuTueur, 0, 4, true, 0))
	kills := []Kill{mortDeBot(a, 2), mortDeBot(a, 3), mortDeBot(a, 4), mortDeBot(b, 5), mortDeBot(a, 6),
		mortDeBot(a, 7)}
	st := c.attachAssistsDuFil(kills)
	attendu := []struct {
		connu bool
		nom   string
		extra int
	}{{true, "W", 0}, {true, "", 0}, {false, "", 0}, {true, "W", 0}, {true, b, 0}, {false, "", 0}}
	for i, w := range attendu {
		got := kills[i].Assist
		if got.Known != w.connu || got.Name != w.nom || got.Extra != w.extra || kills[i].AssistLuAuFil != w.connu {
			t.Errorf("mort %d : connu %v nom %q surplus %d fil %v — attendu connu %v nom %q surplus %d",
				i, got.Known, got.Name, got.Extra, kills[i].AssistLuAuFil, w.connu, w.nom, w.extra)
		}
	}
	if st.Nommes != 3 || st.SansAssistant != 1 || st.TramesArretees != 1 || st.AssistantsMultiples != 1 ||
		st.MortsAuFil != 5 || st.Apprentissages != 1 {
		t.Errorf("comptes %+v", st)
	}
}

// TestUneMortAKillEventGardeSonAssistant — une mort a laquelle un kill-event est attache n est pas
// relue au fil, meme quand le fil de sa trame nomme un autre assistant.
//
// MUTATION QUI DOIT LE FAIRE ROUGIR : retirer la garde `k.Assist.Known` de
// [decodeCtx.attachAssistsDuFil].
func TestUneMortAKillEventGardeSonAssistant(t *testing.T) {
	c := contexteAuFil([]killEventRec{killEventAuFil()},
		messageDuFil(1, typeAssistance, 0, 1, true, 2),
		messageDuFil(2, typeAssistance, 0, 1, true, 4))
	k := Kill{Victim: "V", Feed: FeedTruth{Killer: "K", Present: true}, paquet: paquetID{chunk: 1, pidx: 2, ok: true}}
	k.Assist.Known, k.Assist.Index = true, -1
	kills := []Kill{k}
	c.attachAssistsDuFil(kills)
	if got := kills[0].Assist; !got.Known || got.Name != "" || kills[0].AssistLuAuFil {
		t.Fatalf("la mort a kill-event a change : %+v", got)
	}
}

// TestUnFilQuiContreditSesKillEventsNeSeLitPas — sur une seconde mort a kill-event (assistant W),
// le message d assistance du couple va a B : le film contredit son type, aucune mort ne se lit au
// fil. Meme chose quand deux types different s adressent a l assistant.
//
// MUTATION QUI DOIT LE FAIRE ROUGIR : publier quand `LexiqueRetenu` est faux, ou apprendre le premier
// type venu ([apprendreLeLexique] sans la condition d unicite).
func TestUnFilQuiContreditSesKillEventsNeSeLitPas(t *testing.T) {
	desaccord := killEventRec{chunk: 1, pidx: 9, fields: killEventFields{killer: 0, victim: 1, assist: 2}}
	cas := map[string]*decodeCtx{
		"desaccord": contexteAuFil([]killEventRec{killEventAuFil(), desaccord},
			messageDuFil(1, typeAssistance, 0, 1, true, 2), messageDuFil(9, typeAssistance, 0, 1, true, 4),
			messageDuFil(2, typeAssistance, 0, 3, true, 2)),
		"deux types": contexteAuFil([]killEventRec{killEventAuFil(), desaccord},
			messageDuFil(1, typeAssistance, 0, 1, true, 2), messageDuFil(9, typeAssistance, 0, 1, true, 2),
			messageDuFil(9, typeAuTueur, 0, 1, true, 2), messageDuFil(2, typeAssistance, 0, 3, true, 2)),
	}
	for nom, c := range cas {
		kills := []Kill{mortDeBot("343 A"+BotSuffix, 2)}
		st := c.attachAssistsDuFil(kills)
		if kills[0].Assist.Known {
			t.Errorf("%s : la mort de bot se lit au fil (%+v) ; comptes %+v", nom, kills[0].Assist, st)
		}
		if st.MortsAuFil != 1 {
			t.Errorf("%s : MortsAuFil = %d, attendu 1", nom, st.MortsAuFil)
		}
	}
}

// TestUnAssistantSansEvenementFermeLeLexique — un second kill-event a assistant (W) dont la trame,
// lue entiere, porte le fil du couple sans le type appris : le film contredit son lexique, aucune
// mort ne se lit au fil. Trame arretee : la contradiction ne compte pas, la mort de bot se lit.
//
// MUTATIONS QUI DOIVENT LE FAIRE ROUGIR : retirer `AssistantsSansEvenement == 0` de
// `LexiqueRetenu` ([apprendreLeLexique]) ; remplacer `tousComplets(auCouple)` par vrai dans
// [lexiqueDuFil.confronter].
func TestUnAssistantSansEvenementFermeLeLexique(t *testing.T) {
	second := killEventRec{chunk: 1, pidx: 8, fields: killEventFields{killer: 0, victim: 1, assist: 2}}
	for _, complet := range []bool{true, false} {
		c := contexteAuFil([]killEventRec{killEventAuFil(), second},
			messageDuFil(1, typeAssistance, 0, 1, true, 2), messageDuFil(8, typeAuTueur, 0, 1, complet, 0),
			messageDuFil(2, typeAssistance, 0, 3, true, 2))
		kills := []Kill{mortDeBot("343 A"+BotSuffix, 2)}
		st := c.attachAssistsDuFil(kills)
		if lue := kills[0].Assist.Known; lue == complet || st.LexiqueRetenu == complet {
			t.Errorf("trame du second kill-event entiere %v : mort lue %v, lexique retenu %v ; comptes %+v",
				complet, lue, st.LexiqueRetenu, st)
		}
	}
}

// TestLesRefusDuFil — l evenement d assistance adresse au tueur, a la victime ou a un nom de
// remplissage : la mort est connue, l assistant refuse sous le meme motif qu un kill-event.
//
// MUTATION QUI DOIT LE FAIRE ROUGIR : retirer l un des trois refus de [decodeCtx.lireLAssistantDuFil].
func TestLesRefusDuFil(t *testing.T) {
	for dest, motif := range map[int]string{0: AssistRejectSelf, 3: AssistRejectVictim, 5: AssistRejectRoster} {
		c := contexteAuFil([]killEventRec{killEventAuFil()},
			messageDuFil(1, typeAssistance, 0, 1, true, 2), messageDuFil(2, typeAssistance, 0, 3, true, dest))
		kills := []Kill{mortDeBot("343 A"+BotSuffix, 2)}
		st := c.attachAssistsDuFil(kills)
		if a := kills[0].Assist; !a.Known || a.Name != "" || a.Rejected != motif || st.Rejetes != 1 {
			t.Errorf("destinataire %d : %+v ; comptes %+v — attendu connu, refuse %q", dest, a, st, motif)
		}
	}
}

// TestLeFilDUnAutreTueurNeSeLitPas — la trame porte le fil d un kill du bot A par W, la ligne credite
// K : le couple ne correspond pas, la mort reste inconnue.
//
// MUTATION QUI DOIT LE FAIRE ROUGIR : retirer la comparaison du tueur du predicat de couple de
// [decodeCtx.attachAssistsDuFil].
func TestLeFilDUnAutreTueurNeSeLitPas(t *testing.T) {
	c := contexteAuFil([]killEventRec{killEventAuFil()},
		messageDuFil(1, typeAssistance, 0, 1, true, 2), messageDuFil(2, typeAssistance, 2, 3, true, 4))
	kills := []Kill{mortDeBot("343 A"+BotSuffix, 2)}
	st := c.attachAssistsDuFil(kills)
	if kills[0].Assist.Known || st.MortsAuFil != 0 {
		t.Fatalf("la mort lit le fil d un autre tueur : %+v ; comptes %+v", kills[0].Assist, st)
	}
}
