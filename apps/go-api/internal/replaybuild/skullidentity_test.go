package replaybuild

import (
	"testing"

	"levelup/go-api/internal/analysis/objectiveevents"
)

// skullidentity_test.go — LE PONT D'IDENTITE DESCEND JUSQU'AU PORTEUR DU CRANE d'Oddball.
//
// CE QUI EST EN JEU, ET COMBIEN IL COUTAIT. Le calque du crane nommait son porteur par les SEULS
// instants de mort, qui en exigent TROIS coincidents : un joueur qui meurt moins de trois fois
// dans la manche lui echappe par construction — et en Oddball c'est souvent le porteur, que son
// equipe protege. Son train de tics etait compte `noBridge` et AUCUN intervalle n'etait publie.
// Mesure du 2026-09-10 sur les quatre films Oddball du parc, contre l'oracle API
// `time_as_skull_carrier_seconds` : `43716616` perdait 62,3 s sur son PLUS GROS porteur.
//
// CES TESTS FIGENT LE CABLAGE, ET RIEN D'AUTRE — le pont est pose sur un film Oddball, il ne
// l'est PAS ailleurs, et sans lignes de match il retombe sur les seules morts. La correction du
// pont lui-meme vit a sa source (`objectiveevents/slotidentity_completion_test.go`).

// monoRoundOddballFixture — un film SYNTHETIQUE d'une manche avec les trois populations de slots
// qui font l'interet du lot :
//
//	slot 10  3 morts -> LE PONT PAR MORTS LE NOMME ("aaa") ;
//	slot 12  2 morts -> il lui ECHAPPE, et c'est LE PORTEUR (il porte les tics de score de mode) ;
//	slot 14  compteurs AGREGES (9/5/4) qui ne designent AUCUNE ligne : personne ne le nomme.
func monoRoundOddballFixture() ([]objectiveevents.StatRecord, []objectiveevents.DeathInstant,
	[]objectiveevents.PlayerLine) {
	// LES DEUX CANAUX D'UNE EMISSION SORTENT ENSEMBLE : le composant 2 porte les frags en A et
	// les morts en B, dans le MEME enregistrement (n'ecrire qu'un canal poserait un zero sur
	// l'autre, et la plus longue suite non decroissante du canal muet ecraserait la vraie serie).
	tueMort := func(t, slot int, kills, deaths int64) objectiveevents.StatRecord {
		return objectiveevents.StatRecord{TimeMS: t, Slot: slot, Round: 0,
			Comps: map[int]objectiveevents.StatValue{2: {A: kills, B: deaths}}}
	}
	sideA := func(t, slot, comp int, v int64) objectiveevents.StatRecord {
		return objectiveevents.StatRecord{TimeMS: t, Slot: slot, Round: 0,
			Comps: map[int]objectiveevents.StatValue{comp: {A: v}}}
	}
	recs := []objectiveevents.StatRecord{
		// Slot 10 = "aaa" : 4 frags, 3 morts, 1 assistance — nomme par les morts seules.
		tueMort(500, 10, 1, 0), tueMort(1000, 10, 1, 1), tueMort(2000, 10, 2, 2),
		tueMort(3000, 10, 3, 3), tueMort(3500, 10, 4, 3),
		sideA(4000, 10, 3, 1),
		// Slot 12 = "bbb" : 7 frags, 2 morts, 1 assistance. DEUX morts : sous `deathInstantMin`
		// = 3, le pont par morts se tait. C'est LUI qui porte le crane (comp 0 A).
		tueMort(600, 12, 3, 0), tueMort(5000, 12, 5, 1), tueMort(6000, 12, 7, 2),
		sideA(7000, 12, 3, 1),
		sideA(8000, 12, 0, 1), sideA(9000, 12, 0, 2), sideA(10000, 12, 0, 3),
		// Slot 14 : compteurs AGREGES (9/5/4), appariables a aucune ligne.
		tueMort(20000, 14, 9, 5),
		sideA(20500, 14, 3, 4),
	}
	deaths := []objectiveevents.DeathInstant{
		{XUID: "aaa", TimeMS: 1000}, {XUID: "aaa", TimeMS: 2000}, {XUID: "aaa", TimeMS: 3000},
		{XUID: "bbb", TimeMS: 5000}, {XUID: "bbb", TimeMS: 6000},
	}
	lines := []objectiveevents.PlayerLine{
		{XUID: "aaa", Kills: 4, Deaths: 3, Assists: 1},
		{XUID: "bbb", Kills: 7, Deaths: 2, Assists: 1},
	}
	return recs, deaths, lines
}

// TestSkullInputPoseLePontCompleteSurUnFilmOddball — LE CŒUR DU LOT.
//
// Le porteur (slot 12, 2 morts) est nomme, le slot que les morts nommaient deja garde son nom,
// et le slot agrege reste muet. LA MUTATION EST DANS LE TEST : sans lignes de match, le meme
// appel laisse le slot 12 sans nom — c'est bien la completion qui le nomme, pas autre chose.
func TestSkullInputPoseLePontCompleteSurUnFilmOddball(t *testing.T) {
	recs, deaths, lines := monoRoundOddballFixture()

	got := skullInput(recs, true, &pontParManche{recs: recs, deaths: deaths, lines: lines})
	if !got.Scanned {
		t.Fatalf("entree non balayee sur un film Oddball : le calque ne serait pas construit")
	}
	if !got.Identity.Resolved() {
		t.Fatalf("aucun pont pose : le calque retomberait sur les seules morts")
	}
	if x := got.Identity.AtRound(0, 12); x != "bbb" {
		t.Errorf("porteur du slot 12 = %q, attendu \"bbb\" — le joueur qui meurt DEUX fois est "+
			"hors de portee du pont par morts, et c'est lui qui porte le crane", x)
	}
	if x := got.Identity.AtRound(0, 10); x != "aaa" {
		t.Errorf("slot 10 = %q, attendu \"aaa\" : la completion COMPLETE, elle ne contredit pas", x)
	}
	if x := got.Identity.AtRound(0, 14); x != "" {
		t.Errorf("slot 14 = %q, attendu vide : un slot AGREGE ne designe aucune ligne de match", x)
	}

	// MUTATION — sans lignes, la completion s'abstient et le trou se rouvre.
	sansLignes := skullInput(recs, true, &pontParManche{recs: recs, deaths: deaths})
	if x := sansLignes.Identity.AtRound(0, 12); x != "" {
		t.Errorf("sans lignes de match, slot 12 = %q, attendu vide : ce test ne prouverait rien "+
			"si le pont par morts savait deja le nommer", x)
	}
	if x := sansLignes.Identity.AtRound(0, 10); x != "aaa" {
		t.Errorf("sans lignes de match, slot 10 = %q, attendu \"aaa\" : le pont par morts reste", x)
	}
}

// TestSkullInputHorsOddballNeResoutRien — LA GARDE DE MODE EST DANS CETTE FONCTION, pas chez
// son appelant.
//
// Hors Oddball, l'entree est VIDE et le pont n'est meme pas DEMANDE : le resolveur paresseux
// reste non resolu. La propriete tient pour elle-meme — dans `readFilmStats`, `statborgIdentity`
// reveille de toute facon le resolveur, tous modes confondus — et elle protege le jour ou un
// autre appelant assemblera une entree de crane sans avoir cette raison-la. Le deroulage du
// compteur de morts sur un film d'une autre grammaire a coute 19-22 Go le 2026-08-18 : une
// fonction qui le declenche « au cas ou » est un piege qu'on ne pose pas.
func TestSkullInputHorsOddballNeResoutRien(t *testing.T) {
	recs, deaths, lines := monoRoundOddballFixture()
	pont := &pontParManche{recs: recs, deaths: deaths, lines: lines}

	got := skullInput(recs, false, pont)
	if got.Scanned || got.Records != nil || got.Identity.Resolved() {
		t.Errorf("hors Oddball : entree = %+v, attendue vide (ni balayage, ni records, ni pont)", got)
	}
	if pont.resolu {
		t.Errorf("hors Oddball, le pont a ete RESOLU : la garde de memoire du 2026-08-18 est levee")
	}
}
