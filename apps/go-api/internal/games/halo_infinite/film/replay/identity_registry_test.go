package replay

import (
	"testing"

	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/games/halo_infinite/film/filmdec"
)

// identity_registry_test.go — LES PROPRIETES DU REGISTRE, pas des valeurs figees.
//
// Chaque test porte une regle de la doctrine, et chacune a sa MUTATION jouee au journal du lot :
// l'elimination retiree, deux candidats acceptes, un film entierement nomme qui bougerait.

// filmDeuxJoueurs fabrique un film synthetique : le slot 100 meurt trois fois (il sera nomme
// par le fil des morts), le slot 200 ne meurt JAMAIS (c'est le cas de `d9781168`).
func filmDeuxJoueurs() ([]filmdec.BipedPosition, []Death, PlayerIndexTable) {
	var pos []filmdec.BipedPosition
	// Slot 100 : trois sejours separes par plus de lifeGapUS, chacun clos par une mort.
	for i, debut := range []uint64{1_000_000, 20_000_000, 40_000_000} {
		_ = i
		for t := debut; t <= debut+3_000_000; t += 500_000 {
			pos = append(pos, posAt(100, t, 1, 1, 1))
		}
	}
	// Slot 200 : une seule vie continue, du debut a la fin. Aucune mort ne la termine.
	for t := uint64(1_000_000); t <= 43_000_000; t += 500_000 {
		pos = append(pos, posAt(200, t, 2, 2, 2))
	}
	deaths := []Death{
		{XUID: 111, Gamertag: "MORTEL", TimeMS: 4_000},
		{XUID: 111, Gamertag: "MORTEL", TimeMS: 23_000},
		{XUID: 111, Gamertag: "MORTEL", TimeMS: 43_000},
	}
	idx := PlayerIndexTable{ByXUID: map[uint64]int{111: 0}, Readings: 26}
	return pos, deaths, idx
}

func entreeDeuxJoueurs(roster []uint64) IdentityInput {
	pos, deaths, idx := filmDeuxJoueurs()
	return IdentityInput{
		Positions: pos, Deaths: deaths, PlayerIndices: idx, RosterXUIDs: roster,
		Clock:   IdentityClock{OriginUS: 1_000_000, StepUS: 100_000, FrameCount: 421},
		MatchID: "test",
	}
}

// TestRegistreNommeLeSlotMuetParElimination : un seul xuid du roster sans vie, un seul slot sans
// vie nommee — le registre les apparie, et la vie porte `elimination`, jamais une mort.
//
// MUTATION : retirer `resolveByRosterElimination` -> les vies du slot 200 restent anonymes, rouge.
func TestRegistreNommeLeSlotMuetParElimination(t *testing.T) {
	reg := BuildIdentityRegistry(entreeDeuxJoueurs([]uint64{111, 222}))
	var nommees, deduites int
	for i, l := range reg.Vies() {
		if l.slot != 200 {
			continue
		}
		if l.xuid == 222 {
			nommees++
		}
		if reg.VieDeduite(i) {
			deduites++
		}
		if l.nomPar != NomParElimination {
			t.Fatalf("vie du slot 200 : nomPar = %q, attendu %q", l.nomPar, NomParElimination)
		}
		if l.cause == CauseVieMort {
			t.Fatal("l'elimination a fabrique une MORT — elle ajoute une presence, jamais une fin")
		}
	}
	if nommees == 0 {
		t.Fatal("aucune vie du slot muet n'a ete nommee par elimination")
	}
	if deduites != nommees {
		t.Fatalf("vies deduites = %d, nommees = %d : une deduction non signalee prouverait une absence",
			deduites, nommees)
	}
}

// TestRegistreSeTaitADeuxCandidats : deux xuids libres au roster — aucune vie n'est nommee.
//
// MUTATION : prendre « le premier » candidat -> une identite inventee, rouge.
func TestRegistreSeTaitADeuxCandidats(t *testing.T) {
	reg := BuildIdentityRegistry(entreeDeuxJoueurs([]uint64{111, 222, 333}))
	for _, l := range reg.Vies() {
		if l.slot == 200 && l.xuid != 0 {
			t.Fatalf("un candidat a ete choisi malgre l'ambiguite : xuid %d", l.xuid)
		}
	}
	// ET LE NON RESOLU EST PUBLIE, jamais jete : c'est la doctrine §0.2.
	if reg.Section.Coverage.BipedSlot.Unresolved == 0 {
		t.Fatal("aucun lien non resolu publie alors que le slot 200 reste anonyme")
	}
}

// TestRegistreSansRosterNeDeduitRien : sans roster de la base, l'elimination n'a pas de candidat
// — l'artefact reste publiable hors ligne, exactement comme avant le lot.
func TestRegistreSansRosterNeDeduitRien(t *testing.T) {
	reg := BuildIdentityRegistry(entreeDeuxJoueurs(nil))
	for _, l := range reg.Vies() {
		if l.slot == 200 && l.xuid != 0 {
			t.Fatalf("une identite est apparue sans roster : xuid %d", l.xuid)
		}
	}
}

// TestRegistreFilmEntierementNommeInchange : quand toutes les vies sont deja nommees par la
// lecture, l'elimination ne touche a RIEN — le registre est identique a ce que le pont rendait.
func TestRegistreFilmEntierementNommeInchange(t *testing.T) {
	var pos []filmdec.BipedPosition
	for _, debut := range []uint64{1_000_000, 20_000_000} {
		for t := debut; t <= debut+3_000_000; t += 500_000 {
			pos = append(pos, posAt(100, t, 1, 1, 1))
		}
	}
	in := IdentityInput{
		Positions:     pos,
		Deaths:        []Death{{XUID: 111, TimeMS: 4_000}, {XUID: 111, TimeMS: 23_000}},
		PlayerIndices: PlayerIndexTable{ByXUID: map[uint64]int{111: 0}, Readings: 26},
		RosterXUIDs:   []uint64{111},
		Clock:         IdentityClock{OriginUS: 1_000_000, StepUS: 100_000, FrameCount: 231},
	}
	reg := BuildIdentityRegistry(in)
	for i, l := range reg.Vies() {
		if l.xuid != 111 {
			t.Fatalf("vie %d non nommee par la lecture : %+v", i, l)
		}
		if l.nomPar != NomParMort {
			t.Fatalf("vie %d nommee par %q au lieu de la lecture", i, l.nomPar)
		}
		if reg.VieDeduite(i) {
			t.Fatalf("vie %d marquee deduite alors qu'elle est LUE", i)
		}
	}
	if reg.Section.Coverage.BipedSlot.Unresolved != 0 {
		t.Fatalf("liens non resolus = %d sur un film entierement nomme",
			reg.Section.Coverage.BipedSlot.Unresolved)
	}
}

// TestRegistrePublieLeLienDirectDeLIndexDeJoueur : l'index de joueur lu dans le film est publie
// `direct`, avec son nombre de lectures concordantes — jamais `deduit`.
func TestRegistrePublieLeLienDirectDeLIndexDeJoueur(t *testing.T) {
	reg := BuildIdentityRegistry(entreeDeuxJoueurs([]uint64{111, 222}))
	var direct, externe int
	for _, p := range reg.Section.Players {
		switch p.Link.Source {
		case canonical.LinkDirect:
			direct++
			if p.Link.Method != canonical.MethodPlayerIndexTable || p.Link.Readings != 26 {
				t.Fatalf("lien direct sans sa preuve : %+v", p.Link)
			}
		case canonical.LinkExternal:
			externe++
		}
	}
	if direct != 1 {
		t.Fatalf("liens directs = %d, attendu 1 (le seul xuid de la table d'index)", direct)
	}
	if externe != 1 {
		t.Fatalf("liens externes = %d, attendu 1 (le joueur que seule la feuille connait)", externe)
	}
}

// TestRegistreBorneChaqueLienDeSlot : un slot RECYCLE porte plusieurs liens bornes, jamais un
// lien aplati. C'est ce qui interdit de rejouer le defaut P0-2.
func TestRegistreBorneChaqueLienDeSlot(t *testing.T) {
	reg := BuildIdentityRegistry(entreeDeuxJoueurs([]uint64{111, 222}))
	n := 0
	for _, b := range reg.Section.BipedSlots {
		if b.Slot != 100 {
			continue
		}
		n++
		if b.Link.To < b.Link.From {
			t.Fatalf("bornes inversees : %+v", b.Link)
		}
	}
	if n < 3 {
		t.Fatalf("le slot 100 porte %d liens, attendu 3 (une par vie)", n)
	}
}

// TestBidDuBotEstPublie : le `bid(N.0)` est LU depuis toujours ; il doit sortir jusqu'au roster,
// sans quoi la jointure web retombe sur le nom nu et deux bots homonymes fusionnent.
func TestBidDuBotEstPublie(t *testing.T) {
	bots := []BotIdentity{{FilmIndex: 8, Name: "343 Aloysius [bot]", BotID: 39}}
	roster := buildRoster(PlayerIndexTable{ByXUID: map[uint64]int{111: 0}},
		map[uint64]string{111: "HUMAIN"}, bots)
	var vu bool
	for _, e := range roster {
		if !e.Bot {
			continue
		}
		vu = true
		if e.Bid != "bid(39.0)" {
			t.Fatalf("bid publie = %q, attendu \"bid(39.0)\"", e.Bid)
		}
	}
	if !vu {
		t.Fatal("aucune entree de bot au roster")
	}
	// UN BOT SANS IDENTIFIANT DECLARE NE RECOIT PAS DE `bid(0.0)` : ce serait une jointure fausse.
	if b := (BotIdentity{FilmIndex: 8, Name: "x"}).Bid(); b != "" {
		t.Fatalf("bid fabrique pour un bot sans identifiant : %q", b)
	}
}

// regDe fabrique un registre autour d'un rapport de pont deja construit. Sert aux tests des
// calques, qui exercent un rapport monte a la main plutot qu'un film complet.
func regDe(r OwnerReport) IdentityRegistry {
	return IdentityRegistry{own: r, deducedLives: map[int]bool{}}
}
