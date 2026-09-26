package replay

// identity_registry_entites_test.go — LE CORPS D'UN INDEX PARTAGE NOMME PAR L'ENTITE QUI VIT A SA
// CREATION (lot M2.3, 2026-09-23).
//
//	R-ENTITE     trois bots se relaient sur l'index 8 (gabarit de `b1ad85eb`, sonde P4) : chaque
//	             corps prend le `bid` du bot dont l'entite vit a sa creation — y compris le corps de
//	             `343 PardonMy`, cree AVANT son premier paquet BOT_METADATA ;
//	R-SILENCE    sans entite lue, rien ne change : les corps restent refuses (`index_hors_table`),
//	             comptes `IndexBot`, comme avant le lot ;
//	R-PISTES     une piste anonyme prend le NOM du bot dont une vie du meme slot porte le `bid`.

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
)

// entreeTroisBots : un humain (slot 100, index 0) et trois corps de bots sur l'index 8.
func entreeTroisBots(avecEntites bool) IdentityInput {
	var pos []grammar.BipedPosition
	corps := []struct {
		slot     uint32
		de, a    uint64
		index    uint32
		creation uint64
	}{
		{100, 500_000, 400_000_000, 0, 500_000},
		{512, 500_000, 27_000_000, 8, 500_000},
		{526, 77_400_000, 82_900_000, 8, 77_400_000},
		{564, 323_600_000, 400_000_000, 8, 323_600_000},
	}
	var creations []grammar.BipedCreation
	for _, c := range corps {
		for t := c.de; t <= c.a; t += 500_000 {
			pos = append(pos, posAt(c.slot, t, 1, 1, 0))
		}
		creations = append(creations, creationDe(c.slot, c.creation, c.index))
	}
	in := IdentityInput{
		Positions: pos, BipedCreations: creations,
		PlayerIndices: PlayerIndexTable{ByXUID: map[uint64]int{111: 0}, Readings: 20},
		Bots: []BotIdentity{
			{FilmIndex: 8, Name: "343 Hundy [bot]", BotID: 16, Declarations: [][2]uint64{{1_200_000, 27_300_000}}},
			{FilmIndex: 8, Name: "343 PardonMy [bot]", BotID: 7, Declarations: [][2]uint64{{81_200_000, 83_100_000}}},
			{FilmIndex: 8, Name: "343 Brew Dog [bot]", BotID: 19, Declarations: [][2]uint64{{315_500_000, 0}}},
		},
		Clock:   IdentityClock{OriginUS: 500_000, StepUS: 100_000, FrameCount: 4000},
		MatchID: "temoin",
	}
	if avecEntites {
		in.Entities = scanTroisBots()
	}
	return in
}

// scanTroisBots : les entites de l'index 8 et de l'humain, une image-cle toutes les 20 s.
func scanTroisBots() grammar.PlayerEntityScan {
	s := grammar.PlayerEntityScan{Scanned: true}
	for t := uint64(1_200_000); t <= 401_200_000; t += 20_000_000 {
		s.KeyframesUS = append(s.KeyframesUS, t)
	}
	fin := len(s.KeyframesUS) - 1
	s.Entities = []grammar.PlayerEntity{
		{Slot: 1297, Index: 0, Team: 0, FirstKF: 0, LastKF: fin, Seen: fin + 1},
		{Slot: 1530, Index: 8, Team: 0, FirstKF: 0, LastKF: 1, Seen: 2},           // 1,2 s .. 21,2 s
		{Slot: 1687, Index: 8, Team: 0, FirstKF: 4, LastKF: 4, Seen: 1},           // 81,2 s
		{Slot: 2145, Index: 8, Team: 1, FirstKF: 16, LastKF: fin, Seen: fin - 15}, // 321,2 s ..
	}
	return s
}

// bidsParSlot rend le `bid` que le registre a pose sur la premiere vie de chaque slot.
func bidsParSlot(reg IdentityRegistry) map[uint32]string {
	out := map[uint32]string{}
	for _, l := range reg.Vies() {
		if _, vu := out[l.slot]; !vu {
			out[l.slot] = l.bid
		}
	}
	return out
}

func TestCorpsDIndexPartageNommeParLEntiteASaCreation(t *testing.T) {
	reg := BuildIdentityRegistry(entreeTroisBots(true))
	want := map[uint32]string{512: "bid(16.0)", 526: "bid(7.0)", 564: "bid(19.0)"}
	got := bidsParSlot(reg)
	for slot, bid := range want {
		if got[slot] != bid {
			t.Errorf("corps %d : bid %q, attendu %q (l'entite vivante a sa creation)", slot, got[slot], bid)
		}
	}
	if reg.creation.ParEntite != 3 || reg.creation.IndexBot != 0 {
		t.Errorf("par entite %d, indexBot %d : attendu 3 et 0", reg.creation.ParEntite, reg.creation.IndexBot)
	}
	for _, l := range reg.Vies() {
		if l.slot == 526 && l.nomPar != NomParCreation {
			t.Errorf("corps 526 : voie %q, attendu %q — une LECTURE du film", l.nomPar, NomParCreation)
		}
	}
}

func TestCorpsDIndexPartageSansEntiteRestentRefuses(t *testing.T) {
	reg := BuildIdentityRegistry(entreeTroisBots(false))
	for slot, bid := range bidsParSlot(reg) {
		if slot != 100 && bid != "" {
			t.Errorf("corps %d : bid %q sans entite lue — la lecture devait se taire", slot, bid)
		}
	}
	if reg.creation.ParEntite != 0 || reg.creation.IndexBot == 0 {
		t.Errorf("par entite %d, indexBot %d : sans entite, le refus d'avant", reg.creation.ParEntite,
			reg.creation.IndexBot)
	}
}

func TestPistesDeBotNommeesParLeBidDeLeurVie(t *testing.T) {
	vies := []lifeSpan{
		{slot: 512, from: 500_000, to: 27_000_000, bid: "bid(16.0)"},
		{slot: 526, from: 77_400_000, to: 82_900_000, bid: "bid(7.0)"},
	}
	tracks := []Track{
		{Slot: 512, StartFrame: 0, EndFrame: 265},
		{Slot: 526, StartFrame: 769, EndFrame: 824},
		{Slot: 526, StartFrame: 769, EndFrame: 824, XUID: "111"}, // deja nommee : jamais ecrasee
	}
	bots := entreeTroisBots(true).Bots
	nommerLesPistesDeBotParLeurVie(tracks, vies, bots, replayClock{origin: 500_000, step: 100_000, frames: 4000})
	if tracks[0].Bot != "343 Hundy [bot]" || tracks[1].Bot != "343 PardonMy [bot]" || tracks[2].Bot != "" {
		t.Fatalf("pistes %+v : chaque piste prend le nom du bot de SA vie, une piste nommee ne bouge pas",
			tracks)
	}
}
