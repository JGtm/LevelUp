package objectiveevents

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

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
// Le `continue` sur film absent etait un FAUX VERT (constat D-P2, 5e occurrence du motif
// dans le chantier) : sans aucun film en cache, ce test passait au VERT en n'ayant rien
// confronte. Il compte desormais ce qu'il a reellement joue, et se declare SKIP plutot que
// PASS quand il n'a rien pu verifier.
func TestNamedEventsCrossCheck(t *testing.T) {
	confrontes := 0
	for film, objectiveType := range map[string]string{
		"696a9d7c": ObjectiveTypeZone, "1bc77d2e": ObjectiveTypeFlag,
	} {
		src, ok := newDiskFilmSource(t, film)
		if !ok {
			t.Logf("film %s absent du cache local — non confronte", film)
			continue
		}
		confrontes++
		for slot, byStat := range CrossCheckNamedEvents(src, objectiveType) {
			for stat, pair := range byStat {
				t.Errorf("%s slot %d : %s = %d sur l'emplacement canonique mais %d sur le "+
					"redondant", film, slot, stat, pair[0], pair[1])
			}
		}
	}
	if confrontes == 0 {
		t.Skip("aucun film de reference dans le cache local — aucun recoupement joue")
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

// TestTableStatborgConcordeAvecNamedStatSlots — la table figee `.ai/refs/TABLE_STATS_STATBORG.tsv`
// et `namedStatSlots` doivent dire LA MEME CHOSE, dans les deux sens.
//
// Pourquoi ce test existe (arbitrage du 2026-08-05, integration piste C-bis). La revue
// avait porte deux affirmations incompatibles : « la TSV concorde ligne a ligne avec
// namedStatSlots » (constat R1) et « zone 2 B = deaths y est mais pas dans la table »
// (dette consignee). Les deux etaient vraies en meme temps parce que la premiere etait mal
// formulee : la TSV est le registre de TOUS les emplacements decodes du statborg, tous
// consommateurs confondus, et `namedStatSlots` n'en est qu'un. La colonne `lecteur` de la
// TSV dit desormais qui lit quoi, et ce test verifie la seule concordance qui ait un sens —
// celle des lignes dont le lecteur declare est `named.go`.
//
// Il ferme la question pour de bon : jusqu'ici la concordance etait CRUE, elle est mesuree.
// Ajouter une entree a `namedStatSlots` sans la consigner dans la TSV (ou l'inverse) fait
// echouer ce test.
func TestTableStatborgConcordeAvecNamedStatSlots(t *testing.T) {
	const lecteurNommage = "named.go"
	lignes := lireTableStatborg(t)

	// Sens 1 : toute ligne dont le lecteur est le nommage doit exister dans namedStatSlots,
	// avec la MEME statistique.
	attendu := map[string]map[statSlotKey]string{}
	for _, l := range lignes {
		if !strings.Contains(l.lecteur, lecteurNommage) {
			continue
		}
		if attendu[l.mode] == nil {
			attendu[l.mode] = map[statSlotKey]string{}
		}
		attendu[l.mode][statSlotKey{l.comp, l.side}] = l.stat
	}
	for mode, cles := range attendu {
		table, ok := namedStatSlots[mode]
		if !ok {
			t.Errorf("mode %q present dans la TSV (lecteur %s) mais absent de namedStatSlots",
				mode, lecteurNommage)
			continue
		}
		for cle, stat := range cles {
			got, ok := table[cle]
			if !ok {
				t.Errorf("%s comp %d %v : consigne dans la TSV pour %s, absent de namedStatSlots",
					mode, cle.Comp, cle.Side, lecteurNommage)
				continue
			}
			if got.Stat != stat {
				t.Errorf("%s comp %d %v : TSV dit %q, namedStatSlots dit %q",
					mode, cle.Comp, cle.Side, stat, got.Stat)
			}
		}
	}

	// Sens 2 : reciproquement, aucune entree de namedStatSlots ne doit manquer a la TSV —
	// c'est ce sens qui empeche une cle d'entrer dans le code sans passer par le releve.
	for mode, table := range namedStatSlots {
		for cle, slot := range table {
			if attendu[mode][cle] != slot.Stat {
				t.Errorf("%s comp %d %v (%s) : dans namedStatSlots, pas dans la TSV avec le "+
					"lecteur %s — toute cle nommee doit etre consignee",
					mode, cle.Comp, cle.Side, slot.Stat, lecteurNommage)
			}
		}
	}
}

// ligneStatborg est une ligne utile de la table figee.
type ligneStatborg struct {
	mode    string
	comp    int
	side    string
	stat    string
	lecteur string
}

// lireTableStatborg lit `.ai/refs/TABLE_STATS_STATBORG.tsv`. Les lignes de commentaire, la
// ligne d'en-tete et les lignes sans emplacement (`hill`) sont ecartees.
//
// La table est VERSIONNEE : son absence est une anomalie du depot, pas une fixture
// optionnelle — donc t.Fatal, jamais t.Skip.
func lireTableStatborg(t *testing.T) []ligneStatborg {
	t.Helper()
	path := filepath.Join("..", "..", "..", "..", "..", ".ai", "refs", "TABLE_STATS_STATBORG.tsv")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("table figee %s illisible: %v", path, err)
	}
	var out []ligneStatborg
	for i, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimRight(line, "\r")
		if i == 0 || line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		champs := strings.Split(line, "\t")
		if len(champs) < 6 {
			t.Fatalf("ligne %d : %d colonnes, attendu 6 — la table a change de forme", i+1, len(champs))
		}
		comp, err := strconv.Atoi(champs[1])
		if err != nil {
			continue // ligne sans emplacement (hill)
		}
		side := champs[2]
		if side != sideA && side != sideB {
			t.Fatalf("ligne %d : cote %q inconnu", i+1, side)
		}
		out = append(out, ligneStatborg{
			mode: champs[0], comp: comp, side: side, stat: champs[3], lecteur: champs[4],
		})
	}
	if len(out) == 0 {
		t.Fatal("aucune ligne exploitable dans la table figee")
	}
	return out
}
