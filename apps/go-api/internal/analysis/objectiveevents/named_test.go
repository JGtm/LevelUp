package objectiveevents

import "testing"

// named_test.go — la verite terrain de la lecture par COMPOSANT.
//
// Les tests adosses au cache film SKIPPENT proprement si le film est absent (meme
// convention qu'extract_test.go), donc verts en CI sans films. Les tests purs, eux,
// tournent partout : ce sont eux qui gardent les pieges deja payes.
//
// # L'oracle est celui des HUIT joueurs, et c'est ce qui donne sa force a ce fichier
//
// `match_objective_stats_latest` (base partagee) porte les compteurs d'objectif des huit
// joueurs d'un match, la ou `personal_score_awards` ne couvre que les quatre joueurs suivis.
// Confronter les huit ne double pas la preuve, il la change de nature : un decodage qui
// n'aurait marche que sur les joueurs suivis n'y survivrait pas.
//
// C'est aussi cet oracle qui a corrige un NOM FAUX : `comp 22 A`, nomme `flag_taken` d'apres
// les recompenses, est en realite `flag_grabs` — ses valeurs sont exactement celles de la
// colonne `flag_grabs`, slot par slot. Les compteurs du film sont des STATISTIQUES, pas les
// recompenses de score que le serveur en derive.

// playerStat est la ligne complete d'un joueur : son triplet d'identite et les compteurs
// d'objectif que l'oracle a huit joueurs lui donne.
type playerStat struct {
	line  PlayerLine
	stats map[string]int
}

// oracle8 fige `match_objective_stats_latest` JOINT a `match_participants`, pour les HUIT
// joueurs des deux films de reference (releve le 2026-08-02).
//
// Un compteur absent de la carte vaut ZERO et ce zero compte : il interdit au decodeur
// d'inventer des evenements que l'API ne connait pas.
var oracle8 = map[string][]playerStat{
	// 696a9d7c — Strongholds. Les xuid ne sont pas necessaires ici : le triplet suffit a
	// designer chaque joueur, et le test porte sur l'appariement, pas sur la forme de la cle.
	"696a9d7c": {
		{PlayerLine{"z16", 16, 13, 4}, map[string]int{StatZoneCaptures: 10, StatZoneSecures: 6}},
		{PlayerLine{"z15a", 15, 14, 3}, map[string]int{StatZoneCaptures: 4, StatZoneSecures: 2}},
		{PlayerLine{"z15b", 15, 17, 4}, map[string]int{StatZoneCaptures: 9, StatZoneSecures: 1}},
		{PlayerLine{"z15c", 15, 9, 10}, map[string]int{StatZoneCaptures: 6, StatZoneSecures: 3}},
		{PlayerLine{"madina", 15, 11, 5}, map[string]int{StatZoneCaptures: 10, StatZoneSecures: 0}},
		{PlayerLine{"z9a", 9, 12, 4}, map[string]int{StatZoneCaptures: 6, StatZoneSecures: 1}},
		{PlayerLine{"z9b", 9, 15, 7}, map[string]int{StatZoneCaptures: 7, StatZoneSecures: 2}},
		{PlayerLine{"z8", 8, 12, 8}, map[string]int{StatZoneCaptures: 9, StatZoneSecures: 1}},
	},
	// 1bc77d2e — CTF, avec les vrais xuid.
	"1bc77d2e": {
		{PlayerLine{"2533274823110022", 24, 13, 2}, map[string]int{ // JGtm
			StatFlagCaptures: 1, StatFlagReturns: 2, StatFlagSteals: 4,
			StatFlagCaptureAssists: 0, StatFlagCarriersKilled: 2, StatFlagGrabs: 3}},
		{PlayerLine{"2535433601851512", 20, 15, 7}, map[string]int{
			StatFlagCaptures: 1, StatFlagReturns: 1, StatFlagSteals: 2,
			StatFlagCaptureAssists: 1, StatFlagCarriersKilled: 3, StatFlagGrabs: 3}},
		{PlayerLine{"2535421262359392", 16, 13, 5}, map[string]int{
			StatFlagCaptures: 1, StatFlagReturns: 1, StatFlagSteals: 0,
			StatFlagCaptureAssists: 1, StatFlagCarriersKilled: 3, StatFlagGrabs: 13}},
		{PlayerLine{"2535469190789936", 16, 15, 7}, map[string]int{ // Chocoboflor
			StatFlagCaptures: 1, StatFlagReturns: 1, StatFlagSteals: 0,
			StatFlagCaptureAssists: 2, StatFlagCarriersKilled: 1, StatFlagGrabs: 6}},
		{PlayerLine{"2535415145546162", 11, 11, 2}, map[string]int{
			StatFlagCaptures: 0, StatFlagReturns: 3, StatFlagSteals: 1,
			StatFlagCaptureAssists: 1, StatFlagCarriersKilled: 2, StatFlagGrabs: 2}},
		{PlayerLine{"2533274858283686", 11, 12, 13}, map[string]int{ // Madina97294
			StatFlagCaptures: 0, StatFlagReturns: 0, StatFlagSteals: 2,
			StatFlagCaptureAssists: 2, StatFlagCarriersKilled: 1, StatFlagGrabs: 16}},
		{PlayerLine{"2535436340554308", 10, 18, 5}, map[string]int{
			StatFlagCaptures: 0, StatFlagReturns: 1, StatFlagSteals: 3,
			StatFlagCaptureAssists: 1, StatFlagCarriersKilled: 1, StatFlagGrabs: 1}},
		{PlayerLine{"2535456897775421", 6, 17, 4}, map[string]int{
			StatFlagCaptures: 1, StatFlagReturns: 0, StatFlagSteals: 2,
			StatFlagCaptureAssists: 1, StatFlagCarriersKilled: 0, StatFlagGrabs: 2}},
	},
}

// linesOf rend les lignes de match d'un film, pour l'appariement des slots.
func linesOf(film string) []PlayerLine {
	out := make([]PlayerLine, 0, len(oracle8[film]))
	for _, p := range oracle8[film] {
		out = append(out, p.line)
	}
	return out
}

// checkAgainstOracle8 confronte les comptes decodes a l'oracle des HUIT joueurs, compteur
// par compteur. L'appariement slot -> joueur passe par le triplet (cf. slotidentity.go).
func checkAgainstOracle8(t *testing.T, film, objectiveType string) {
	t.Helper()
	src, ok := newDiskFilmSource(t, film)
	if !ok {
		t.Skipf("film %s absent du cache local", film)
	}
	recs := StatRecords(src)
	identity := slotIdentityFrom(recs, linesOf(film))
	if len(identity) != 8 {
		t.Fatalf("%s : %d slots apparies, attendu 8", film, len(identity))
	}
	counts := CountsBySlot(namedEventsFrom(recs, objectiveType))

	byXUID := map[string]map[string]int{}
	for _, p := range oracle8[film] {
		byXUID[p.line.XUID] = p.stats
	}
	checked := 0
	for slot, xuid := range identity {
		for stat, want := range byXUID[xuid] {
			checked++
			if got := counts[slot][stat]; got != want {
				t.Errorf("%s slot %d (%s) : %s = %d, attendu %d (oracle 8 joueurs)",
					film, slot, xuid, stat, got, want)
			}
		}
	}
	t.Logf("%s : %d confrontations sur les 8 joueurs", film, checked)
}

// TestNamedEventsZoneAgainstEightPlayers — Strongholds `696a9d7c`, les HUIT joueurs.
//
// Ce que ce test prouve et que la lecture par valeur ne pouvait pas prouver :
// `zone_captures` et `zone_secures` valent tous deux 25 points par action, donc aucune
// lecture du score ne pouvait les separer. Ils vivent dans deux emplacements distincts
// (comp 20 B et comp 21 A) et s'y lisent sans ambiguite.
func TestNamedEventsZoneAgainstEightPlayers(t *testing.T) {
	checkAgainstOracle8(t, "696a9d7c", ObjectiveTypeZone)
}

// TestNamedEventsCTFAgainstEightPlayers — CTF `1bc77d2e`, les HUIT joueurs et les SIX
// compteurs de drapeau.
//
// C'est le test qui ferme l'ambiguite « l'un de trois noms » : `flag_returns`,
// `flag_steals` et `flag_carriers_killed` valent tous 25 points et se separent ici
// parfaitement. Et c'est lui qui garde le nom corrige — `flag_grabs`, pas `flag_taken` :
// 16 pour Madina97294 la ou la RECOMPENSE `flag_taken` dit 4.
func TestNamedEventsCTFAgainstEightPlayers(t *testing.T) {
	checkAgainstOracle8(t, "1bc77d2e", ObjectiveTypeFlag)
}

// TestNamedEventsZoneTotalsMatchAPI — le controle d'ensemble, qui ne depend d'AUCUNE
// correspondance slot -> joueur : sur `696a9d7c`, comp 20 B totalise 61 et comp 21 A 16,
// somme 77, exactement le total du match. Meme si l'identite des slots derivait, ce total
// tiendrait.
func TestNamedEventsZoneTotalsMatchAPI(t *testing.T) {
	src, ok := newDiskFilmSource(t, "696a9d7c")
	if !ok {
		t.Skip("film 696a9d7c absent du cache local")
	}
	total := map[string]int{}
	for _, e := range NamedEvents(src, ObjectiveTypeZone) {
		total[e.Stat]++
	}
	if total[StatZoneCaptures] != 61 {
		t.Errorf("zone_captures total = %d, attendu 61", total[StatZoneCaptures])
	}
	if total[StatZoneSecures] != 16 {
		t.Errorf("zone_secures total = %d, attendu 16", total[StatZoneSecures])
	}
	if sum := total[StatZoneCaptures] + total[StatZoneSecures]; sum != 77 {
		t.Errorf("zone_captures + zone_secures = %d, attendu 77 (total du match)", sum)
	}
}

// TestNamedEventsCrossCheck — les emplacements REDONDANTS doivent reproduire exactement
// leur emplacement canonique. C'est un controle interne au film, sans oracle externe : le
// statborg duplique certaines statistiques, donc un desaccord signale un decodage qui a
// derape sur ce slot.
//
// Ce test a deja servi : il a demasque une valeur parasite a -115 sur comp 0 A qui faisait
// remonter le compteur en 116 evenements au lieu d'1.
func TestNamedEventsCrossCheck(t *testing.T) {
	for film, objectiveType := range map[string]string{
		"696a9d7c": ObjectiveTypeZone, "1bc77d2e": ObjectiveTypeFlag,
	} {
		src, ok := newDiskFilmSource(t, film)
		if !ok {
			continue
		}
		for slot, byStat := range CrossCheckNamedEvents(src, objectiveType) {
			for stat, pair := range byStat {
				t.Errorf("%s slot %d : %s = %d sur l'emplacement canonique mais %d sur le "+
					"redondant", film, slot, stat, pair[0], pair[1])
			}
		}
	}
}

// TestNamedEventsUnknownModeIsSilent — un mode sans table ne rend rien, et surtout
// n'invente aucun nom.
//
// KOTH est dans ce cas DEFINITIVEMENT, et ce n'est pas un trou a combler : les recompenses
// de colline ne sont repliquees dans AUCUN emplacement du statborg (mesure sur deux films,
// etat de l'art §21), le binaire ne declare aucune famille de stats KOTH, et la base n'a
// aucune colonne `hill_*`. Trois sources concordantes.
func TestNamedEventsUnknownModeIsSilent(t *testing.T) {
	recs := []StatRecord{{TimeMS: 1000, Slot: 10, Comps: map[int]StatValue{20: {A: 0, B: 3}}}}
	for _, mode := range []string{ObjectiveTypeHill, ObjectiveTypeSkull, "", "slayer"} {
		if got := namedEventsFrom(recs, mode); got != nil {
			t.Errorf("mode %q : %d evenements rendus, attendu aucun", mode, len(got))
		}
	}
}

// TestNamedEventsIgnoresNegativeValues — non-regression du piege le plus couteux du
// fichier. Une emission parasite a valeur negative ne doit produire AUCUN evenement, et ne
// doit pas non plus permettre de recompter les unites deja acquises.
//
// Sans ce garde-fou, la suite ci-dessous (1 puis -115 puis 1) rendait 116 evenements.
func TestNamedEventsIgnoresNegativeValues(t *testing.T) {
	recs := []StatRecord{
		{TimeMS: 1000, Slot: 10, Comps: map[int]StatValue{23: {A: 1}}},
		{TimeMS: 2000, Slot: 10, Comps: map[int]StatValue{23: {A: -115}}},
		{TimeMS: 3000, Slot: 10, Comps: map[int]StatValue{23: {A: 1}}},
	}
	got := namedEventsFrom(recs, ObjectiveTypeFlag)
	if len(got) != 1 {
		t.Fatalf("%d evenements rendus, attendu 1 (la valeur negative est un parasite)", len(got))
	}
	if got[0].Stat != StatFlagReturns || got[0].TimeMS != 1000 {
		t.Errorf("evenement = %s a t=%d, attendu flag_returns a t=1000", got[0].Stat, got[0].TimeMS)
	}
}

// TestNamedEventsRedundantSlotsDoNotDoubleCount — un emplacement redondant ne doit jamais
// emettre : sinon chaque frag serait compte deux fois (comp 2 A et comp 12 A portent tous
// deux le nombre de frags).
func TestNamedEventsRedundantSlotsDoNotDoubleCount(t *testing.T) {
	recs := []StatRecord{{
		TimeMS: 1000, Slot: 10,
		Comps: map[int]StatValue{2: {A: 3}, 12: {A: 3, B: 2}, 3: {A: 2}},
	}}
	counts := CountsBySlot(namedEventsFrom(recs, ObjectiveTypeZone))
	if got := counts[10][StatKills]; got != 3 {
		t.Errorf("kills = %d, attendu 3 (comp 12 A redouble comp 2 A)", got)
	}
	if got := counts[10][StatAssists]; got != 2 {
		t.Errorf("assists = %d, attendu 2 (comp 12 B redouble comp 3 A)", got)
	}
}

// TestNamedEventsRepeatedValueIsNotAnEvent — un composant est reemis des que l'UNE de ses
// deux valeurs bouge. Une reemission a valeur inchangee n'est donc pas un evenement.
func TestNamedEventsRepeatedValueIsNotAnEvent(t *testing.T) {
	recs := []StatRecord{
		{TimeMS: 1000, Slot: 10, Comps: map[int]StatValue{20: {B: 1}}},
		{TimeMS: 2000, Slot: 10, Comps: map[int]StatValue{20: {B: 1}}},
		{TimeMS: 3000, Slot: 10, Comps: map[int]StatValue{20: {B: 2}}},
	}
	got := namedEventsFrom(recs, ObjectiveTypeZone)
	if len(got) != 2 {
		t.Fatalf("%d evenements rendus, attendu 2 (la reemission a valeur egale n'en est pas un)",
			len(got))
	}
	if got[0].TimeMS != 1000 || got[1].TimeMS != 3000 {
		t.Errorf("instants = %d et %d, attendus 1000 et 3000", got[0].TimeMS, got[1].TimeMS)
	}
}

// TestKnownStatsCoverBothModes — l'inventaire d'un mode dit ce que la lecture couvre. Il
// garde aussi le piege central du chantier : `comp 21 A` vaut `zone_secures` en zones et
// `flag_captures` en CTF, donc les deux inventaires DOIVENT differer.
func TestKnownStatsCoverBothModes(t *testing.T) {
	flag, zone := KnownStats(ObjectiveTypeFlag), KnownStats(ObjectiveTypeZone)
	for _, name := range []string{StatFlagReturns, StatFlagSteals, StatFlagCarriersKilled} {
		if !flag[name] {
			t.Errorf("CTF : %s absent de l'inventaire", name)
		}
		if zone[name] {
			t.Errorf("zones : %s ne devrait pas y figurer", name)
		}
	}
	if !zone[StatZoneCaptures] || !zone[StatZoneSecures] {
		t.Error("zones : zone_captures / zone_secures absents de l'inventaire")
	}
	if len(KnownStats(ObjectiveTypeHill)) != 0 {
		t.Error("hill : l'inventaire doit rester vide — aucun compteur de colline n'est replique")
	}
}
