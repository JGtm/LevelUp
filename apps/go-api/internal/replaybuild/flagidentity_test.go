package replaybuild

import (
	"testing"

	"levelup/go-api/internal/analysis/objectiveevents"
	"levelup/go-api/internal/analysis/replay"
)

// flagidentity_test.go — LE PONT D'IDENTITE DESCEND JUSQU'AU CALQUE DU DRAPEAU (schema 42).
//
// CE QUI EST EN JEU. Le calque du drapeau nommait son porteur par les SEULS instants de mort,
// qui en exigent TROIS coincidents : un joueur qui meurt moins de trois fois lui echappe par
// construction — et c'est celui qui porte le drapeau. Ses prises etaient comptees `noBridge` et
// AUCUN intervalle n'etait publie. Le pont COMPLETE par le triplet, deja en service pour les
// actions d'objectif, est desormais pose sur l'entree du calque par [withFlagIdentity].
//
// CES TESTS FIGENT LA REGLE, ET RIEN D'AUTRE. La correction du pont lui-meme (les trois gardes
// de `CompletedByLines`) est prouvee a sa source, dans
// `objectiveevents/slotidentity_completion_test.go` ; ce que l'on fige ici est le CABLAGE : le
// pont est pose sur un film de CTF, il ne l'est PAS ailleurs, et il n'est resolu qu'une fois.

// monoRoundCTFFixture — un film SYNTHETIQUE d'une manche que les trois signaux reconnaissent
// comme du CTF, avec les trois populations de slots qui font tout l'interet du lot :
//
//	slot 10  3 morts -> LE PONT PAR MORTS LE NOMME ("aaa") ;
//	slot 12  2 morts -> il lui ECHAPPE, et c'est le porteur (7 frags) : seul le triplet le nomme ;
//	slot 14  compteurs AGREGES (9/5/4) qui ne designent AUCUNE ligne : personne ne le nomme.
//
// Rend les enregistrements, le fil des morts, les lignes de match et les bursts de capture.
func monoRoundCTFFixture() ([]objectiveevents.StatRecord, []objectiveevents.DeathInstant,
	[]objectiveevents.PlayerLine, []int) {
	// LES DEUX CANAUX D'UNE EMISSION SORTENT ENSEMBLE, et la fixture doit le respecter : le
	// composant 2 porte les frags en A et les morts en B, dans le MEME enregistrement. N'ecrire
	// qu'un canal poserait un zero sur l'autre, et la plus longue suite non decroissante du
	// canal muet ecraserait la vraie serie.
	tueMort := func(t, slot int, kills, deaths int64) objectiveevents.StatRecord {
		return objectiveevents.StatRecord{TimeMS: t, Slot: slot, Round: 0,
			Comps: map[int]objectiveevents.StatValue{2: {A: kills, B: deaths}}}
	}
	sideA := func(t, slot, comp int, v int64) objectiveevents.StatRecord {
		return objectiveevents.StatRecord{TimeMS: t, Slot: slot, Round: 0,
			Comps: map[int]objectiveevents.StatValue{comp: {A: v}}}
	}
	recs := []objectiveevents.StatRecord{
		// Slot 10 = "aaa" : 4 frags, 3 morts, 1 assistance. Ses TROIS morts coincident avec le
		// fil ci-dessous, donc le pont PAR MORTS le nomme sans l'aide du triplet.
		tueMort(500, 10, 1, 0), tueMort(1000, 10, 1, 1), tueMort(2000, 10, 2, 2),
		tueMort(3000, 10, 3, 3), tueMort(3500, 10, 4, 3),
		sideA(4000, 10, 3, 1),
		// Slot 12 = "bbb" : 7 frags, 2 morts, 1 assistance. DEUX morts seulement : sous
		// `deathInstantMin` = 3, le pont par morts se tait. C'est LUI qui vole et capture.
		tueMort(600, 12, 3, 0), tueMort(5000, 12, 5, 1), tueMort(6000, 12, 7, 2),
		sideA(7000, 12, 3, 1),
		sideA(8000, 12, 24, 1), // flag_steals
		sideA(9000, 12, 21, 1), // flag_captures
		// Slot 14 : compteurs AGREGES (9/5/4), appariables a aucune ligne, et ses morts ne
		// coincident avec aucun fil. Il prend un drapeau (comp 22 A) qui restera SANS PONT.
		tueMort(20000, 14, 9, 5),
		sideA(20500, 14, 3, 4),
		sideA(21000, 14, 22, 1), // flag_grabs
	}
	deaths := []objectiveevents.DeathInstant{
		{XUID: "aaa", TimeMS: 1000}, {XUID: "aaa", TimeMS: 2000}, {XUID: "aaa", TimeMS: 3000},
		{XUID: "bbb", TimeMS: 5000}, {XUID: "bbb", TimeMS: 6000},
	}
	lines := []objectiveevents.PlayerLine{
		{XUID: "aaa", Kills: 4, Deaths: 3, Assists: 1},
		{XUID: "bbb", Kills: 7, Deaths: 2, Assists: 1},
	}
	// UNE capture au film, donc au moins un burst : le verdict de mode exige
	// `bursts > 0 && captures > 0 && captures <= bursts && steals > 0`.
	return recs, deaths, lines, []int{9000}
}

// TestWithFlagIdentityPoseLePontCompleteSurUnFilmCTF — LE CŒUR DU LOT.
//
// Le porteur (slot 12, 2 morts) est nomme, le slot que les morts nommaient deja garde son nom,
// et le slot agrege reste muet. LA MUTATION EST DANS LE TEST : sans lignes de match, le meme
// appel laisse le slot 12 sans nom — c'est bien la completion qui le nomme, pas autre chose.
func TestWithFlagIdentityPoseLePontCompleteSurUnFilmCTF(t *testing.T) {
	recs, deaths, lines, bursts := monoRoundCTFFixture()
	in := replay.FlagInput{Scanned: true, Records: recs, Bursts: bursts}

	got := withFlagIdentity(in, &pontParManche{recs: recs, deaths: deaths, lines: lines})
	if !got.Identity.Resolved() {
		t.Fatalf("aucun pont pose sur un film de CTF : le calque retomberait sur les seules morts")
	}
	if x := got.Identity.At(12, 8000); x != "bbb" {
		t.Errorf("porteur du slot 12 = %q, attendu \"bbb\" — le joueur qui meurt DEUX fois est "+
			"hors de portee du pont par morts, et c'est lui qui porte le drapeau", x)
	}
	if x := got.Identity.At(10, 1000); x != "aaa" {
		t.Errorf("slot 10 = %q, attendu \"aaa\" : la completion COMPLETE, elle ne contredit pas", x)
	}
	if x := got.Identity.At(14, 21000); x != "" {
		t.Errorf("slot 14 = %q, attendu vide : un slot AGREGE ne designe aucune ligne de match", x)
	}

	// MUTATION — sans lignes, la completion s'abstient et le trou se rouvre.
	sansLignes := withFlagIdentity(in, &pontParManche{recs: recs, deaths: deaths})
	if x := sansLignes.Identity.At(12, 8000); x != "" {
		t.Errorf("sans lignes de match, slot 12 = %q, attendu vide : ce test ne prouverait rien "+
			"si le pont par morts savait deja le nommer", x)
	}
	if x := sansLignes.Identity.At(10, 1000); x != "aaa" {
		t.Errorf("sans lignes de match, slot 10 = %q, attendu \"aaa\" : le pont par morts reste", x)
	}
}

// TestWithFlagIdentityNeResoutRienHorsCTF — LA GARDE DE COUT, ET ELLE N'EST PAS COSMETIQUE.
//
// Le deroulage du compteur de morts sur un film dont la grammaire n'est PAS celle du CTF montait
// a 19-22 Go avant la garde du 2026-08-18. Le pont ne doit donc etre resolu que sur un film que
// les TROIS SIGNAUX DU FILM reconnaissent — jamais sur la foi d'un nom de variante.
func TestWithFlagIdentityNeResoutRienHorsCTF(t *testing.T) {
	recs, deaths, lines, bursts := monoRoundCTFFixture()
	for nom, in := range map[string]replay.FlagInput{
		"aucun burst de capture": {Scanned: true, Records: recs},
		"aucun evenement nomme":  {Scanned: true, Bursts: bursts},
	} {
		t.Run(nom, func(t *testing.T) {
			got := withFlagIdentity(in, &pontParManche{recs: recs, deaths: deaths, lines: lines})
			if got.Identity.Resolved() {
				t.Errorf("pont resolu hors CTF : %d slot(s) nomme(s) — le calque ne publiera rien "+
					"de ce film, et ce travail est paye pour rien", got.Identity.NamedCount())
			}
		})
	}
}

// TestPontParMancheNeResoutQuUneFois — UN SEUL DEROULAGE PAR CUISSON, PARTAGE PAR LES DEUX
// CALQUES.
//
// Les actions d'objectif et le drapeau vivant consomment LE MEME pont. Le resoudre deux fois
// serait deux deroulages complets du compteur de morts sur les memes enregistrements. La preuve
// est faite en VIDANT la source apres le premier appel : si le second re-resolvait, il rendrait
// une table vide.
func TestPontParMancheNeResoutQuUneFois(t *testing.T) {
	recs, deaths, lines, _ := monoRoundCTFFixture()
	p := &pontParManche{recs: recs, deaths: deaths, lines: lines}
	if x := p.identite().At(12, 8000); x != "bbb" {
		t.Fatalf("premier appel : slot 12 = %q, attendu \"bbb\"", x)
	}
	p.recs, p.deaths, p.lines = nil, nil, nil
	if x := p.identite().At(12, 8000); x != "bbb" {
		t.Errorf("second appel : slot 12 = %q, attendu \"bbb\" — le pont a ete re-resolu au lieu "+
			"d'etre memorise", x)
	}
}
