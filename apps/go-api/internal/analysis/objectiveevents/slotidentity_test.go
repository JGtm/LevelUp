package objectiveevents

import "testing"

// slotidentity_test.go — la verite terrain du pont slot d'entite -> joueur.
//
// L oracle est celui des HUIT joueurs, defini dans named_test.go (oracle8) : le triplet
// (frags, morts, assistances) de match_participants, joint aux compteurs d objectif.

// TestSlotIdentityResolvesAllEightPlayers — l'appariement doit rendre les HUIT slots, et
// aucun xuid ne doit apparaitre deux fois.
//
// C'est le test qui autorise le collage au rejeu 2D : sans lui, les evenements seraient
// attribues par un numero de slot qui n'a pas le meme sens dans les deux mondes.
//
// Le `continue` sur film absent etait un FAUX VERT (constat D-P2) : sans aucun film en
// cache, ce test passait au VERT en n'ayant apparie personne. Il compte ce qu'il a joue et
// se declare SKIP plutot que PASS quand il n'a rien pu apparier.
func TestSlotIdentityResolvesAllEightPlayers(t *testing.T) {
	apparies := 0
	for _, film := range []string{"696a9d7c", "1bc77d2e"} {
		lines := linesOf(film)
		src, ok := newDiskFilmSource(t, film)
		if !ok {
			t.Logf("film %s absent du cache local — aucun appariement joue dessus", film)
			continue
		}
		apparies++
		got := slotIdentityFrom(StatRecords(src), lines)
		if len(got) != 8 {
			t.Errorf("%s : %d slots apparies, attendu 8", film, len(got))
		}
		seen := map[string]int{}
		for slot, xuid := range got {
			if prev, dup := seen[xuid]; dup {
				t.Errorf("%s : xuid %s attribue aux slots %d ET %d", film, xuid, prev, slot)
			}
			seen[xuid] = slot
			if slot < 10 || slot > 24 || slot%2 != 0 {
				t.Errorf("%s : slot %d hors des slots de joueur (10..24 pairs)", film, slot)
			}
		}
	}
	if apparies == 0 {
		t.Skip("aucun film de reference dans le cache local — aucun appariement joue")
	}
}

// TestSlotIdentityMatchesKnownOwners — l'appariement doit retrouver les identites etablies
// par une AUTRE methode (coincidence des increments de +100 avec les instants de frag,
// etat de l'art §16.2). Deux chemins independants qui tombent d'accord.
func TestSlotIdentityMatchesKnownOwners(t *testing.T) {
	src, ok := newDiskFilmSource(t, "1bc77d2e")
	if !ok {
		t.Skip("film 1bc77d2e absent du cache local")
	}
	got := slotIdentityFrom(StatRecords(src), linesOf("1bc77d2e"))
	// slotOwners (named_test.go) donne slot 18 = JGtm, dont le xuid est etabli en §16.2.
	for slot, want := range map[int]string{
		18: "2533274823110022", // JGtm
		24: "2535469190789936", // Chocoboflor
		16: "2533274858283686", // Madina97294
	} {
		if got[slot] != want {
			t.Errorf("slot %d -> %q, attendu %q", slot, got[slot], want)
		}
	}
}

// TestSlotIdentityRefusesAmbiguousTriplets — deux joueurs sur le meme triplet ne prouvent
// rien : aucun des deux slots ne doit etre apparie.
//
// Se taire est le comportement voulu. Attribuer les frags d'un joueur a un autre serait,
// sur une carte, une erreur invisible et credible.
func TestSlotIdentityRefusesAmbiguousTriplets(t *testing.T) {
	recs := []StatRecord{
		{TimeMS: 1000, Slot: 10, Comps: map[int]StatValue{2: {A: 5, B: 3}, 3: {A: 2}}},
		{TimeMS: 1000, Slot: 12, Comps: map[int]StatValue{2: {A: 5, B: 3}, 3: {A: 2}}},
		{TimeMS: 1000, Slot: 14, Comps: map[int]StatValue{2: {A: 9, B: 1}, 3: {A: 4}}},
	}
	lines := []PlayerLine{
		{XUID: "a", Kills: 5, Deaths: 3, Assists: 2},
		{XUID: "b", Kills: 5, Deaths: 3, Assists: 2},
		{XUID: "c", Kills: 9, Deaths: 1, Assists: 4},
	}
	got := slotIdentityFrom(recs, lines)
	if _, ok := got[10]; ok {
		t.Error("slot 10 apparie alors que deux joueurs partagent son triplet")
	}
	if _, ok := got[12]; ok {
		t.Error("slot 12 apparie alors que deux joueurs partagent son triplet")
	}
	if got[14] != "c" {
		t.Errorf("slot 14 -> %q, attendu \"c\" (triplet unique)", got[14])
	}
}

// TestSlotIdentityRefuseUnXuidRevendiqueParDeuxSlots — la SECONDE passe, qui n'etait
// couverte par aucun test (constat D-P2).
//
// Le test voisin (triplets ambigus) est rejete des la PREMIERE passe : deux LIGNES y
// partagent le triplet, donc aucun slot ne designe une ligne unique et rien n'entre dans
// `claim`. La seconde passe garde un cas different et symetrique — deux SLOTS designent
// chacun sans ambiguite LA MEME ligne. Chacun sort donc de la premiere passe avec une
// revendication legitime, et c'est seulement en les confrontant qu'on voit qu'ils ne
// peuvent pas avoir raison tous les deux. Sans elle, le xuid serait attribue aux deux
// slots — et les evenements d'un joueur apparaitraient en double a l'ecran.
//
// Mutation qui doit le faire rougir : renvoyer `claim` au lieu de `out`.
func TestSlotIdentityRefuseUnXuidRevendiqueParDeuxSlots(t *testing.T) {
	recs := []StatRecord{
		{TimeMS: 1000, Slot: 10, Comps: map[int]StatValue{2: {A: 5, B: 3}, 3: {A: 2}}},
		{TimeMS: 1000, Slot: 12, Comps: map[int]StatValue{2: {A: 5, B: 3}, 3: {A: 2}}},
		{TimeMS: 1000, Slot: 14, Comps: map[int]StatValue{2: {A: 9, B: 1}, 3: {A: 4}}},
	}
	// UNE seule ligne porte le triplet (5, 3, 2) : les slots 10 et 12 le revendiquent
	// chacun sans ambiguite, la premiere passe les accepte donc tous les deux.
	lines := []PlayerLine{
		{XUID: "a", Kills: 5, Deaths: 3, Assists: 2},
		{XUID: "c", Kills: 9, Deaths: 1, Assists: 4},
	}
	got := slotIdentityFrom(recs, lines)
	if _, ok := got[10]; ok {
		t.Error("slot 10 apparie alors qu'il se dispute le xuid \"a\" avec le slot 12")
	}
	if _, ok := got[12]; ok {
		t.Error("slot 12 apparie alors qu'il se dispute le xuid \"a\" avec le slot 10")
	}
	if got[14] != "c" {
		t.Errorf("slot 14 -> %q, attendu \"c\" : la dispute des autres ne doit pas le penaliser", got[14])
	}
}

// TestIdentifyNamedEventsDropsUnmatchedSlots — un evenement dont le slot n'est pas apparie
// est ECARTE, jamais attribue par defaut.
func TestIdentifyNamedEventsDropsUnmatchedSlots(t *testing.T) {
	evs := []NamedEvent{
		{TimeMS: 500, Slot: 10, Stat: StatZoneCaptures, Comp: 20, Side: sideB},
		{TimeMS: 900, Slot: 12, Stat: StatZoneSecures, Comp: 21, Side: sideA},
	}
	got := IdentifyNamedEvents(evs, map[int]string{10: "xuid-a"})
	if len(got) != 1 {
		t.Fatalf("%d evenements identifies, attendu 1 (le slot 12 n'est pas apparie)", len(got))
	}
	if got[0].XUID != "xuid-a" || got[0].Stat != StatZoneCaptures {
		t.Errorf("evenement = %s par %s, attendu zone_captured par xuid-a", got[0].Stat, got[0].XUID)
	}
}

// TestIdentifyNamedEventsOnRealFilm — bout en bout : les evenements nommes d'un vrai film,
// attribues a de vrais xuid, prets a etre superposes aux positions du rejeu.
func TestIdentifyNamedEventsOnRealFilm(t *testing.T) {
	src, ok := newDiskFilmSource(t, "1bc77d2e")
	if !ok {
		t.Skip("film 1bc77d2e absent du cache local")
	}
	evs := NamedEvents(src, ObjectiveTypeFlag)
	identity := slotIdentityFrom(StatRecords(src), linesOf("1bc77d2e"))
	got := IdentifyNamedEvents(evs, identity)

	// Les 8 slots etant apparies, aucun evenement ne doit etre perdu.
	if len(got) != len(evs) {
		t.Errorf("%d evenements identifies sur %d, attendu aucune perte", len(got), len(evs))
	}
	// L'ordre chronologique est la propriete dont depend le collage aux positions.
	for i := 1; i < len(got); i++ {
		if got[i].TimeMS < got[i-1].TimeMS {
			t.Fatalf("evenements non chronologiques a l'index %d", i)
		}
	}
	// JGtm doit porter ses 4 vols de drapeau, sous son xuid.
	steals := 0
	for _, e := range got {
		if e.XUID == "2533274823110022" && e.Stat == StatFlagSteals {
			steals++
		}
	}
	if steals != 4 {
		t.Errorf("JGtm : %d flag_stolen, attendu 4 (personal_score_awards)", steals)
	}
}
