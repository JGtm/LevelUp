package killsource

// index_motif_test.go — LES TEMOINS DU LOT 5.2b.1 : UN REMPLACANT ENTRE AU ROSTER, ET SES MORTS
// NE FERMENT PLUS LA PUBLICATION DU MATCH.
//
// Le cas synthetique reproduit `b1ad85eb` a la structure pres : dix joueurs, NEUF indices connus
// de la table de `chunk_00` (zero..huit) et UN remplacant arrive en cours, que la table ignore et
// que le motif du xuid nomme a l indice 10.

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
)

// rosterAvecRemplacant : la table du film nomme les sieges 0..7, BOT_METADATA tient le 8, et le
// motif du xuid apporte le remplacant a l indice 10.
func rosterAvecRemplacant(t *testing.T) *roster {
	t.Helper()
	kf := &killFeed{names: []string{"A", "B", "C", "D", "E", "F", "G", "H", "Remplacant"}}
	bm := botMeta{NBots: 1, Bots: []bot{{Slot: 9, BotID: 7, Name: "343 Bot"}}}
	table := FilmTable{Build: "b", Seats: map[int]string{
		0: "A", 1: "B", 2: "C", 3: "D", 4: "E", 5: "F", 6: "G", 7: "H",
	}}
	motif := indexParMotif{
		nomParIndex: map[int]string{0: "A", 7: "H", 10: "Remplacant"},
		lectures:    27,
	}
	return buildRoster(kf, bm, true, table, motif)
}

// TestLeMotifEpingleLeRemplacantQueLaTableIgnore — LA ROUTE, ET SA MORSURE.
//
// Mutation qui doit le faire rougir : retirer `r.pinMotifSeats(m)` de [buildRoster] (la borne
// reste a 10 et l indice 10 n est nomme par personne), ou faire passer le motif AVANT la table
// (les accords deviennent des epinglages et la provenance ment).
func TestLeMotifEpingleLeRemplacantQueLaTableIgnore(t *testing.T) {
	r := rosterAvecRemplacant(t)
	if r.nPlay != 11 {
		t.Fatalf("borne des indices = %d, attendu 11 — le remplacant n etend pas le roster", r.nPlay)
	}
	if got := r.table.MotifPinned; got != 1 {
		t.Errorf("MotifPinned = %d, attendu 1 (le seul indice que la table ne tient pas)", got)
	}
	if got := r.table.MotifAgree; got != 2 {
		t.Errorf("MotifAgree = %d, attendu 2 — le motif CONTROLE les indices deja lus", got)
	}
	if nom, ok := r.nomEpingle(10); !ok || nom != "Remplacant" {
		t.Errorf("indice 10 epingle = %q (%v), attendu \"Remplacant\"", nom, ok)
	}
	if got := r.originOf(10); got != OriginXUIDMotif {
		t.Errorf("provenance de l indice 10 = %q, attendu %q", got, OriginXUIDMotif)
	}
	// LE PIEGE DU LOT 1.8, REJOUE POUR LE NOUVEL EPINGLAGE : un indice pose par le motif n est
	// PAS un bot. Le confondre reclasserait ses morts en « mort de bot », population jamais
	// publiee (la mesure d alors : dix lignes devenues deux).
	if r.isBotIndex(10) {
		t.Error("l indice pose par le motif est pris pour un bot")
	}
	if !r.isBotIndex(9) {
		t.Error("le slot de BOT_METADATA doit rester un bot")
	}
}

// TestLeMotifNeCorrigeJamaisLaTable — la table de `chunk_00` est la lecture la plus eprouvee
// (314 accords sur 322 sieges, 30 films) : une contradiction du motif se COMPTE, elle ne deplace
// rien. Meme doctrine que le controle par les votes du kill-feed (D14 b).
func TestLeMotifNeCorrigeJamaisLaTable(t *testing.T) {
	kf := &killFeed{names: []string{"A", "B"}}
	table := FilmTable{Build: "b", Seats: map[int]string{0: "A", 1: "B"}}
	motif := indexParMotif{nomParIndex: map[int]string{0: "B"}, lectures: 3}
	r := buildRoster(kf, botMeta{}, true, table, motif)
	if r.table.MotifContradict != 1 || r.table.MotifPinned != 0 {
		t.Fatalf("provenance = %+v, attendu 1 contradiction et 0 epinglage", r.table)
	}
	if nom, _ := r.nomEpingle(0); nom != "A" {
		t.Errorf("l indice 0 vaut %q : la contradiction a DEPLACE la lecture de la table", nom)
	}
}

// TestLeMotifRefuseUnNomDejaEpingle — un meme nom ne peut porter deux indices, exactement comme
// pour les sieges de la table.
func TestLeMotifRefuseUnNomDejaEpingle(t *testing.T) {
	kf := &killFeed{names: []string{"A"}}
	table := FilmTable{Build: "b", Seats: map[int]string{0: "A"}}
	motif := indexParMotif{nomParIndex: map[int]string{4: "A"}, lectures: 3}
	r := buildRoster(kf, botMeta{}, true, table, motif)
	if r.table.MotifDuplicate != 1 || r.table.MotifPinned != 0 {
		t.Fatalf("provenance = %+v, attendu 1 doublon refuse", r.table)
	}
}

// TestDeuxXUIDAuMemeIndiceNePublientRien — un index partage rangerait les morts de deux joueurs
// sous un seul nom. Les deux lectures sont laches et comptees, comme `replay.injectiveOrEmpty`.
func TestDeuxXUIDAuMemeIndiceNePublientRien(t *testing.T) {
	m := indexParMotif{nomParIndex: map[int]string{}}
	m.retenir(4, "A")
	m.retenir(4, "B")
	if len(m.nomParIndex) != 0 {
		t.Fatalf("index retenus = %v, attendu aucun : la collision doit lacher les deux", m.nomParIndex)
	}
	if m.desaccords != 1 {
		t.Errorf("desaccords = %d, attendu 1 — la collision doit se compter", m.desaccords)
	}
}

// TestBotsSuccessifsNeFabriquentPasDeNomLibre — LE DEFAUT QUE LE LOT 5.2b.1 A MESURE SUR
// `b1ad85eb` : trois bots declares sur le MEME slot ajoutaient TROIS noms au roster pour UN seul
// indice. Les deux perdants devenaient des NOMS LIBRES, donc de la matiere a inference — et deux
// noms libres pour un indice libre rendent `AffectationUnique` faux, ce qui ferme la publication
// ligne par ligne de tout le match.
func TestBotsSuccessifsNeFabriquentPasDeNomLibre(t *testing.T) {
	kf := &killFeed{names: []string{"A", "B"}}
	bm := botMeta{NBots: 3, Bots: []bot{
		{Slot: 2, BotID: 7, Name: "343 Un"},
		{Slot: 2, BotID: 16, Name: "343 Deux"},
		{Slot: 2, BotID: 19, Name: "343 Trois"},
	}}
	r := buildRoster(kf, bm, true, FilmTable{Build: "b", Seats: map[int]string{0: "A", 1: "B"}},
		indexParMotif{})
	if r.botsSuccedes != 2 {
		t.Fatalf("successions comptees = %d, attendu 2", r.botsSuccedes)
	}
	// LE VAINQUEUR NE CHANGE PAS : c est toujours le DERNIER declare, comme avant le lot.
	if nom, _ := r.nomEpingle(2); nom != "343 Trois"+BotSuffix {
		t.Errorf("indice 2 = %q, attendu le DERNIER bot declare", nom)
	}
	_, nomsLibres := r.freeSlots()
	if len(nomsLibres) != 0 {
		t.Errorf("noms libres = %v, attendu aucun — les bots perdants sont redevenus de la "+
			"matiere a inference", nomsLibres)
	}
	if r.public().BotsSuccedes != 2 {
		t.Error("la succession n est pas publiee : elle resterait invisible a un relecteur")
	}
}

// TestUnHorsRosterNEteintPlusLeMatch — LE SECOND FILET DU LOT, du cote de la sante : une mort
// dont l indice deborde le roster est refusee SEULE (le filtre de credibilite l ecarte avant
// qu elle ne devienne un candidat), et les autres continuent de publier.
//
// Mutation qui doit le faire rougir : remettre `OutOfRoster` dans [grammar.KillSourceHealth.Alerts].
func TestUnHorsRosterNEteintPlusLeMatch(t *testing.T) {
	res := &Result{
		Health: grammar.KillSourceHealth{Film: "temoin", Candidates: 100, Published: 93,
			UnexplainedPair: 4, UnexplainedSelf: 3, OutOfRoster: 8,
			DeathsReal: 93, DeathsCovered: 93},
		BijectionDetermined: true,
	}
	if !res.LineByLinePublishable() {
		t.Fatalf("publication ligne par ligne refusee : alertes %v", res.Health.Alerts())
	}
	if len(res.Health.Degradations()) != 1 {
		t.Errorf("degradations = %v, attendu une seule, NOMMEE", res.Health.Degradations())
	}
	if got := res.Health.Verdict(); got != grammar.VerdictHorsDomaine {
		t.Errorf("verdict = %q, attendu %q : un participant non compte doit rester visible",
			got, grammar.VerdictHorsDomaine)
	}
}

// TestUneLectureQuiSeContreditNEpinglRien — LE TEMOIN DU NEGATIF MESURE SUR `a349fea8` : le
// resolveur y rend l index 0 pour les vingt-cinq xuids du film. Les collisions lachent
// vingt-quatre lectures ; la vingt-cinquieme, lue a un AUTRE index donc sans collision,
// survivait et epinglait du bruit — 28 lignes publiees perdues.
//
// Mutation qui doit le faire rougir : retirer l appel a `refuserSiElleSeContredit`.
func TestUneLectureQuiSeContreditNEpinglRien(t *testing.T) {
	m := indexParMotif{nomParIndex: map[int]string{}}
	m.retenir(0, "A")
	m.retenir(0, "B") // collision : les deux sont lachees, un desaccord est compte
	m.retenir(7, "C") // lecture SANS collision : elle survivait au filtre par xuid
	if len(m.nomParIndex) != 1 {
		t.Fatalf("temoin sans valeur : %v — la lecture isolee doit survivre au filtre par xuid",
			m.nomParIndex)
	}
	m.refuserSiElleSeContredit()
	if len(m.nomParIndex) != 0 {
		t.Errorf("index retenus = %v, attendu aucun : une lecture qui se contredit ne pose rien",
			m.nomParIndex)
	}
	if m.desaccords != 1 {
		t.Errorf("desaccords = %d, attendu 1 — le refus doit rester EXPLIQUE", m.desaccords)
	}
}
