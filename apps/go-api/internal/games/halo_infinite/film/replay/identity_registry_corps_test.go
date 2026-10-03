package replay

import (
	"context"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/fallback"
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
)

// identity_registry_corps_test.go — LE REGISTRE RAISONNE PAR CORPS (slot, generation) (lot J5.4 du
// plan de suite de l'audit du decodeur, 2026-09-27).

// creationGen fabrique un record de creation lu d'une GENERATION donnee : le corps que le slot
// porte a partir de cette date.
func creationGen(slot uint32, tUS uint64, index, gen uint32) grammar.BipedCreation {
	c := creationDe(slot, tUS, index)
	c.Generation = gen
	return c
}

// TestRegistre_RecordDeCreationNOuvreQueSaVie (RA2-1) : sur un slot RECYCLE, le record d'un corps
// n'ouvre qu'une vie de CE corps. Ici le premier corps du slot 100 (generation 1, index 0) ne
// replique aucune position ; le second (generation 2, index 1) nait a 12 s et vit [14 s..18 s].
//
// ROUGE AVANT : `viesOuvertes` donnait au premier record « la premiere vie non terminee a sa date »
// — la vie du corps SUIVANT —, publiee `direct` sous le joueur de l'index 0.
//
// MUTATION : retirer la garde de corps de `viesOuvertes` -> la vie porte 111, rouge.
func TestRegistre_RecordDeCreationNOuvreQueSaVie(t *testing.T) {
	var pos []grammar.BipedPosition
	for ts := uint64(14_000_000); ts <= 18_000_000; ts += 500_000 {
		pos = append(pos, posAt(100, ts, 1, 1, 1))
	}
	for ts := uint64(1_000_000); ts <= 18_000_000; ts += 500_000 {
		pos = append(pos, posAt(200, ts, 2, 2, 2))
	}
	in := filmDeuxCorps()
	in.Positions = pos
	in.BipedCreations = []grammar.BipedCreation{
		creationGen(100, 500_000, 0, 1),
		creationGen(100, 12_000_000, 1, 2),
		creationDe(200, 1_000_000, 1),
	}
	in.PlayerIndices.ByXUID = map[uint64]int{111: 0, 222: 1, 333: 2}
	in.BipedCreations[2].ParticipantIndex = 2
	reg := BuildIdentityRegistry(context.Background(), in)
	var vue bool
	for _, l := range reg.Vies() {
		if l.slot != 100 {
			continue
		}
		vue = true
		if l.xuid != 222 {
			t.Fatalf("vie 100[%d..%d] : xuid = %d, attendu 222 — le record du corps precedent a "+
				"ouvert la vie du corps suivant", l.from, l.to, l.xuid)
		}
		if l.nomPar != NomParCreation {
			t.Fatalf("vie 100 : voie %q, attendu %q (le record de SON corps l'ouvre)", l.nomPar,
				NomParCreation)
		}
	}
	if !vue {
		t.Fatal("aucune vie sur le slot 100 : la decoupe a change")
	}
}

// entreeSlotRecycle : le slot 100 porte deux corps — generation 1 (index 0, xuid 111) qui vit
// [1 s..4 s], puis generation 2 (index `index2`) nee a 12 s qui vit [14 s..18 s] ; le slot 200
// porte l'index 2 (xuid 333) tout le film.
func entreeSlotRecycle(index2 uint32) IdentityInput {
	in := filmDeuxCorps()
	in.BipedCreations = []grammar.BipedCreation{
		creationGen(100, 500_000, 0, 1),
		creationGen(100, 12_000_000, index2, 2),
		creationDe(200, 1_000_000, 2),
	}
	in.PlayerIndices.ByXUID = map[uint64]int{111: 0, 333: 2}
	return in
}

// pistesDuSlotRecycle : les pistes publiees du slot 100 (axe 1 s + 100 ms), la premiere nommee par
// sa vie, la seconde — celle du corps de generation 2 — sans nom.
func pistesDuSlotRecycle() []Track {
	return []Track{
		{Slot: 100, StartFrame: 0, EndFrame: 30, XUID: "111"},
		{Slot: 100, StartFrame: 130, EndFrame: 170},
	}
}

// TestViesSansNom_NeFranchissentPasUneFrontiereDeCorps (RA2-2) : le nommage final « par occupation
// du slot » ne prend jamais l'identite d'un AUTRE corps du meme slot. Deux figures : le second
// corps porte un index hors de la table (aucun joueur publie), puis l'index d'un BOT declare.
//
// ROUGE AVANT : la piste du second corps prenait le xuid 111 du corps precedent (`occupantPrevious`),
// et le bot ne recevait pas son nom (`nameBotTracks` lisait le pont aplati, index 0 pour le slot).
//
// MUTATIONS : retirer le filtre de corps de `nameRemainingLives` -> la piste porte 111, rouge ;
// faire lire a `nameBotTracks` le pont aplati -> la piste du bot reste sans nom, rouge.
func TestViesSansNom_NeFranchissentPasUneFrontiereDeCorps(t *testing.T) {
	t.Run("index hors table", func(t *testing.T) {
		in := entreeSlotRecycle(7)
		fb := fallback.NouveauCompteur()
		in.Fallbacks = fb
		reg := BuildIdentityRegistry(context.Background(), in)
		tracks := pistesDuSlotRecycle()
		nameBotTracks(context.Background(), tracks, reg.Occupants(), in.Bots, in.Clock.OriginUS, in.Clock.StepUS)
		rep := nameRemainingLives(tracks, reg, in.Clock.OriginUS, in.Clock.StepUS)
		if tracks[1].XUID != "" || tracks[1].Bot != "" {
			t.Fatalf("la piste du corps de generation 2 a pris l'identite %q/%q du corps precedent",
				tracks[1].XUID, tracks[1].Bot)
		}
		if rep.remaining != 1 {
			t.Fatalf("residu = %d, attendu 1 : la vie sans nom se compte (%+v)", rep.remaining, rep)
		}
		if n := fb.Compte(fallback.NomIdentiteVieParOccupationDuCorps); n != 0 {
			t.Fatalf("repli par occupation declenche %d fois, attendu 0 (rien n'a ete nomme)", n)
		}
	})
	t.Run("bot declare", func(t *testing.T) {
		in := entreeSlotRecycle(5)
		in.Bots = []BotIdentity{{FilmIndex: 5, Name: "343 Bot [bot]"}}
		reg := BuildIdentityRegistry(context.Background(), in)
		tracks := pistesDuSlotRecycle()
		nameBotTracks(context.Background(), tracks, reg.Occupants(), in.Bots, in.Clock.OriginUS, in.Clock.StepUS)
		nameRemainingLives(tracks, reg, in.Clock.OriginUS, in.Clock.StepUS)
		if tracks[1].XUID != "" || tracks[1].Bot != "343 Bot [bot]" {
			t.Fatalf("piste du corps de generation 2 : xuid %q bot %q, attendu le bot de SON corps",
				tracks[1].XUID, tracks[1].Bot)
		}
		if tracks[0].Bot != "" || tracks[0].XUID != "111" {
			t.Fatalf("piste du corps de generation 1 : %+v, attendu 111 inchange", tracks[0])
		}
	})
}

// TestViesSansNom_LeRepliParOccupationSeCompte : dans son corps, le nommage par occupation reste —
// c'est un REPLI, inscrit au registre et compte a chaque piste qu'il nomme.
func TestViesSansNom_LeRepliParOccupationSeCompte(t *testing.T) {
	in := entreeSlotRecycle(0) // les deux corps portent le meme joueur : aucune frontiere d'identite
	// Un troisieme sejour du corps de generation 2 : une piste sans nom tombe ENTRE deux vies
	// nommees de CE corps.
	for ts := uint64(24_000_000); ts <= 28_000_000; ts += 500_000 {
		in.Positions = append(in.Positions, posAt(100, ts, 1, 1, 1))
	}
	fb := fallback.NouveauCompteur()
	in.Fallbacks = fb
	reg := BuildIdentityRegistry(context.Background(), in)
	tracks := []Track{
		{Slot: 100, StartFrame: 130, EndFrame: 170, XUID: "111"},
		{Slot: 100, StartFrame: 190, EndFrame: 210},
	}
	nameRemainingLives(tracks, reg, in.Clock.OriginUS, in.Clock.StepUS)
	if tracks[1].XUID != "111" {
		t.Fatalf("la piste sans nom du MEME corps devait etre nommee 111, obtenu %q", tracks[1].XUID)
	}
	if n := fb.Compte(fallback.NomIdentiteVieParOccupationDuCorps); n != 1 {
		t.Fatalf("repli par occupation compte %d fois, attendu 1", n)
	}
}
