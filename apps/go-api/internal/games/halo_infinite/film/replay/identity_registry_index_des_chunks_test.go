package replay

import (
	"context"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/fallback"
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// identity_registry_index_des_chunks_test.go — L'INDEX QUE LES CHUNKS LISENT POUR UN JOUEUR QUE LA
// TABLE DU FILM PLACE AILLEURS (identity_registry_index_des_chunks.go).
//
// La figure est celle du parc : la table du film assied le joueur 222 a l'index 1, les chunks le
// lisent a l'index 8, et un corps porte l'index 8. Chaque test porte une condition de la regle.

// filmIndexDesChunks : trois corps. Slot 100 (index 0, joueur 111) tout le film ; slot 200 (index 1,
// joueur 222 par la table) de 1 a 6 s ; slot 300 (index 8) a partir de `debut300`.
func filmIndexDesChunks(debut300 uint64, controle222 int) IdentityInput {
	var pos []grammar.BipedPosition
	for t := uint64(1_000_000); t <= 18_000_000; t += 500_000 {
		pos = append(pos, posAt(100, t, 1, 1, 1))
	}
	for t := uint64(1_000_000); t <= 6_000_000; t += 500_000 {
		pos = append(pos, posAt(200, t, 2, 2, 2))
	}
	for t := debut300; t <= 18_000_000; t += 500_000 {
		pos = append(pos, posAt(300, t, 3, 3, 3))
	}
	return IdentityInput{
		Positions: pos,
		BipedCreations: []grammar.BipedCreation{
			creationDe(100, 1_000_000, 0),
			creationDe(200, 1_000_000, 1),
			creationDe(300, debut300, 8),
		},
		FilmTable: tableDuFilm(
			grammar.FilmPlayerSeat{FilmIndex: 0, XUID: 111, Gamertag: "Alpha"},
			grammar.FilmPlayerSeat{FilmIndex: 1, XUID: 222, Gamertag: "Bravo"},
		),
		PlayerIndices: types.PlayerIndexTable{ByXUID: map[uint64]int{111: 0, 222: controle222}, Readings: 26},
		Clock:         IdentityClock{OriginUS: 1_000_000, StepUS: 100_000, FrameCount: 171},
		MatchID:       "test",
		Fallbacks:     fallback.NouveauCompteur(),
	}
}

// vieDuSlotChunks rend l'identite de la vie du slot donne (le slot n'en porte qu'une dans ces figures).
func vieDuSlotChunks(t *testing.T, reg IdentityRegistry, slot uint32) lifeSpan {
	t.Helper()
	for _, l := range reg.Vies() {
		if l.slot == slot {
			return l
		}
	}
	t.Fatalf("aucune vie sur le slot %d", slot)
	return lifeSpan{}
}

// L'index des chunks nomme le corps que la table tait, le siege de la table reste, le repli se compte.
//
// MUTATION : ne pas etendre `versXUID` par l'index des chunks -> le slot 300 reste sans nom, rouge.
func TestIndexDesChunksNommeLeCorpsQueLaTableTait(t *testing.T) {
	in := filmIndexDesChunks(10_000_000, 8)
	reg := BuildIdentityRegistry(context.Background(), in)
	if got := vieDuSlotChunks(t, reg, 300).xuid; got != 222 {
		t.Fatalf("slot 300 (index 8) nomme %d, attendu 222 : les chunks lisent 222 a l'index 8", got)
	}
	if got := vieDuSlotChunks(t, reg, 200).xuid; got != 222 {
		t.Fatalf("slot 200 (index 1, siege de la table) nomme %d, attendu 222", got)
	}
	// LE PONT SLOT -> INDEX DIT L'INDEX QUE LE CORPS PORTE (8), pas le siege du joueur (1) : c'est par
	// lui que se lit l'equipe d'un corps quand le siege n'en porte aucune. MUTATION : retirer
	// `garderLIndexDesCorpsParChunks` -> 1, rouge.
	if got := reg.IndexParSlot()[300]; got != 8 {
		t.Fatalf("pont slot 300 -> index %d, attendu 8 (l'index du record de creation)", got)
	}
	if got := reg.TableDIndex().ByXUID[222]; got != 1 {
		t.Fatalf("la table effective assied 222 a l'index %d, attendu 1 : la regle ne retire aucun siege", got)
	}
	if n := in.Fallbacks.Compte(fallback.NomIndexDesChunksHorsTable); n != 1 {
		t.Fatalf("repli compte %d fois, attendu 1 (une vie nommee)", n)
	}
	if c := reg.Section.Coverage.BipedSlot.UnresolvedByCause.IndexOutOfTable; c != 0 {
		t.Fatalf("index_hors_table = %d, attendu 0", c)
	}
}

// Un corps de l'index des chunks qui chevauche un corps du meme joueur nomme par la table : l'une des
// deux lectures est fausse, l'index des chunks est retire et le refus se compte.
//
// MUTATION : retirer la garde (`indexDesChunksEnConflit` rendant vide) -> le slot 300 est nomme, rouge.
func TestIndexDesChunksSeTaitQuandLeJoueurAuraitDeuxCorps(t *testing.T) {
	in := filmIndexDesChunks(3_000_000, 8)
	reg := BuildIdentityRegistry(context.Background(), in)
	if got := vieDuSlotChunks(t, reg, 300).xuid; got != 0 {
		t.Fatalf("slot 300 nomme %d alors qu'il chevauche le corps du slot 200 du meme joueur", got)
	}
	if reg.creation.IndexDesChunksRefuses != 1 {
		t.Fatalf("refus = %d, attendu 1", reg.creation.IndexDesChunksRefuses)
	}
	if c := reg.Section.Coverage.BipedSlot.UnresolvedByCause.IndexOutOfTable; c != 1 {
		t.Fatalf("index_hors_table = %d, attendu 1 (le refus garde sa cause)", c)
	}
	if n := in.Fallbacks.Compte(fallback.NomIndexDesChunksHorsTable); n != 0 {
		t.Fatalf("repli compte %d fois apres refus, attendu 0", n)
	}
}

// Les chunks lisent 222 a l index 0, que la table donne a 111 que les chunks ne contredisent pas : deux
// lectures du film se disputent l index, rien n est pose et le refus se compte.
func TestIndexDesChunksSeTaitSurUnIndexTenuParLaTable(t *testing.T) {
	in := filmIndexDesChunks(10_000_000, 0)
	got := composerTableDIndex(in)
	if len(got.indexDesChunks) != 0 || got.indexTenus != 1 || got.chaines != 0 {
		t.Fatalf("index des chunks = %v, tenus = %d, chaines = %d ; attendus vide, 1 et 0", got.indexDesChunks, got.indexTenus, got.chaines)
	}
	reg := BuildIdentityRegistry(context.Background(), in)
	if v := vieDuSlotChunks(t, reg, 100).xuid; v != 111 {
		t.Fatalf("slot 100 nomme %d, attendu 111 (le siege de la table reste)", v)
	}
}

// Un bot que BOT_METADATA declare a l'index des chunks le tient : le corps n'est pas donne a l'humain.
func TestIndexDesChunksNePrendPasLIndexDUnBotDeclare(t *testing.T) {
	in := filmIndexDesChunks(10_000_000, 8)
	in.Bots = []BotIdentity{{FilmIndex: 8, Name: "343 Flippant [bot]", BotID: 7, BotIDLu: true}}
	reg := BuildIdentityRegistry(context.Background(), in)
	if got := vieDuSlotChunks(t, reg, 300).xuid; got != 0 {
		t.Fatalf("le corps d'un bot declare a l'index 8 porte le xuid %d", got)
	}
}

// filmChaine : la table assied 111 a 0, 222 a 1, 333 a 2 ; les chunks lisent 222 a 2 (le siege de
// 333) et 333 a 5. Corps : slot 100 index 0, slot 200 index 2, slot 300 index 5, sans chevauchement
// entre 200 et 300.
func filmChaine() IdentityInput {
	var pos []grammar.BipedPosition
	for t := uint64(1_000_000); t <= 18_000_000; t += 500_000 {
		pos = append(pos, posAt(100, t, 1, 1, 1), posAt(200, t, 2, 2, 2), posAt(300, t, 3, 3, 3))
	}
	return IdentityInput{
		Positions: pos,
		BipedCreations: []grammar.BipedCreation{
			creationDe(100, 1_000_000, 0), creationDe(200, 1_000_000, 2), creationDe(300, 1_000_000, 5),
		},
		FilmTable: tableDuFilm(
			grammar.FilmPlayerSeat{FilmIndex: 0, XUID: 111},
			grammar.FilmPlayerSeat{FilmIndex: 1, XUID: 222},
			grammar.FilmPlayerSeat{FilmIndex: 2, XUID: 333},
		),
		PlayerIndices: types.PlayerIndexTable{ByXUID: map[uint64]int{111: 0, 222: 2, 333: 5}, Readings: 26},
		Clock:         IdentityClock{OriginUS: 1_000_000, StepUS: 100_000, FrameCount: 171},
		MatchID:       "test",
		Fallbacks:     fallback.NouveauCompteur(),
	}
}

// Une chaine : le siege de table de 333 est l'index des chunks de 222, et 333 est lui-meme contredit.
// Les deux membres prennent leur index des chunks ; le corps d'index 2 est a 222, celui d'index 5 a 333.
//
// MUTATION : `membresDesChaines` rendant vide -> le slot 200 reste a 333 (table), et le slot 300, qui
// donnerait a 333 deux corps a la fois, reste sans nom par la garde du chevauchement, rouge.
func TestChaineLesChunksFontFoiPourSesMembres(t *testing.T) {
	in := filmChaine()
	reg := BuildIdentityRegistry(context.Background(), in)
	if got := reg.TableDIndex().ByXUID; got[222] != 2 || got[333] != 5 || got[111] != 0 {
		t.Fatalf("table effective = %v, attendue 111:0 222:2 333:5", got)
	}
	if v := vieDuSlotChunks(t, reg, 200).xuid; v != 222 {
		t.Fatalf("slot 200 (index 2) nomme %d, attendu 222", v)
	}
	if v := vieDuSlotChunks(t, reg, 300).xuid; v != 333 {
		t.Fatalf("slot 300 (index 5) nomme %d, attendu 333", v)
	}
	if c := reg.CouvertureTableDuFilm(); c.Direct != 1 || c.Fallback != 2 || c.Contradiction != 2 {
		t.Fatalf("couverture = %+v, attendus direct 1, repli 2, contradictions 2", c)
	}
}

// Une chaine dont un membre lirait son index des chunks sur le siege d'un joueur NON contredit : tout
// ou rien, la table reste telle qu'elle est ecrite et la chaine se compte.
func TestChaineNonResolueQuandUnIndexEstTenuParUnJoueurNonContredit(t *testing.T) {
	in := filmChaine()
	in.FilmTable = tableDuFilm(
		grammar.FilmPlayerSeat{FilmIndex: 0, XUID: 111},
		grammar.FilmPlayerSeat{FilmIndex: 1, XUID: 222},
		grammar.FilmPlayerSeat{FilmIndex: 2, XUID: 333},
		grammar.FilmPlayerSeat{FilmIndex: 5, XUID: 444}, // muet dans les chunks : non contredit
	)
	got := composerTableDIndex(in)
	if got.chaines != 1 || got.membresDeChaine != 0 {
		t.Fatalf("chaines = %d, membres = %d ; attendus 1 et 0", got.chaines, got.membresDeChaine)
	}
	if got.table.ByXUID[333] != 2 || got.table.ByXUID[222] != 1 {
		t.Fatalf("table effective = %v, attendue telle que le film l'ecrit", got.table.ByXUID)
	}
}

// Le siege de table d'un joueur contredit est l'index que les chunks donnent a un joueur que la table
// n'assied pas : sans la chaine, le complement poserait une collision qui retirerait les DEUX liens.
// La chaine porte le titulaire a son index des chunks, le complement garde le sien.
func TestChaineLeveLaCollisionDuComplement(t *testing.T) {
	in := filmChaine()
	in.FilmTable = tableDuFilm(
		grammar.FilmPlayerSeat{FilmIndex: 0, XUID: 111},
		grammar.FilmPlayerSeat{FilmIndex: 2, XUID: 333},
	)
	// 222 n'a pas de siege : les chunks le lisent a 2, le siege de 333, que les chunks lisent a 5.
	got := composerTableDIndex(in)
	if got.couverture.IndexCollisions != 0 {
		t.Fatalf("collisions = %d, attendu 0 : la chaine explique le conflit", got.couverture.IndexCollisions)
	}
	if got.table.ByXUID[222] != 2 || got.table.ByXUID[333] != 5 {
		t.Fatalf("table effective = %v, attendue 222:2 333:5", got.table.ByXUID)
	}
}
