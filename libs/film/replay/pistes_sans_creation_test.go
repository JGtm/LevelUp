package replay

import (
	"context"
	"reflect"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/identity"
	"github.com/JGtm/halo-infinite-film-decoder/internal/facts/fallback"
	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar"
)

// pistes_sans_creation_test.go — LA REGLE « UN SLOT SANS RECORD DE CREATION NI NOM N'EST PAS UN
// CORPS » (schema 108). Chaque test porte une condition de la regle, et chacun a sa MUTATION.

// slotParasite est un slot que ni le film de test ni aucune de ses creations n'ouvre.
const slotParasite uint32 = 36

// positionsAvecParasite rend celles de `filmDeuxCorps` plus quatre positions du slot parasite.
func positionsAvecParasite() IdentityInput {
	in := filmDeuxCorps()
	for t := uint64(5_000_000); t <= 5_300_000; t += 100_000 {
		in.Positions = append(in.Positions, posAt(slotParasite, t, 50, 60, 0))
	}
	return in
}

// documentDuFilmDeTest assemble le film de test sous les options donnees ; les creations et la
// table d'index viennent de `filmDeuxCorps`.
func documentDuFilmDeTest(t *testing.T, avecParasite bool, opt Options) (ReplayDocument, *fallback.Compteur) {
	t.Helper()
	in := filmDeuxCorps()
	if avecParasite {
		in = positionsAvecParasite()
	}
	fb := fallback.NouveauCompteur()
	opt.FrameIntervalMS = 100
	opt.BipedCreations = in.BipedCreations
	opt.PlayerIndices = in.PlayerIndices
	opt.Fallbacks = fb
	doc := BuildFromPositions(context.Background(), "test", "halo_infinite", in.Positions, nil, opt)
	return doc, fb
}

func pisteDuSlot(doc ReplayDocument, slot uint32) bool {
	for _, tr := range doc.Tracks {
		if tr.Slot == slot {
			return true
		}
	}
	return false
}

// TestSlotSansCreationNiNomNEstPasPublieSurUnFilmComplet : le film lit ses creations, son morceau des
// temps forts est lu, le slot n'a aucun record et rien ne le nomme : aucune piste, un repli compte, et
// les pistes des vrais corps sont celles du film sans parasite.
//
// MUTATION : retirer l'appel a `sansLesSlots` -> la piste du slot 36 reste publiee, rouge.
func TestSlotSansCreationNiNomNEstPasPublieSurUnFilmComplet(t *testing.T) {
	avec, fb := documentDuFilmDeTest(t, true, Options{DeathsFeed: DeathsFeedRead})
	sans, _ := documentDuFilmDeTest(t, false, Options{DeathsFeed: DeathsFeedRead})
	if pisteDuSlot(avec, slotParasite) {
		t.Fatalf("le slot %d, sans record de creation ni nom, est publie : %+v", slotParasite, avec.Tracks)
	}
	if got := fb.Compte(fallback.NomPositionDeBipedeSansCreationEcartee); got != 4 {
		t.Fatalf("positions ecartees comptees = %d, attendu 4", got)
	}
	if !reflect.DeepEqual(avec.Tracks, sans.Tracks) {
		t.Fatalf("les pistes des vrais corps changent avec le parasite :\n avec %+v\n sans %+v", avec.Tracks, sans.Tracks)
	}
	if avec.Coverage.Bridge.UnnamedLives != 0 {
		t.Fatalf("vies sans identite = %d, attendu 0", avec.Coverage.Bridge.UnnamedLives)
	}
}

// TestSlotSansCreationResteSiLeFilmNEstPasProuveComplet : sans morceau des temps forts lu, un record a
// pu manquer faute de chunk, et la regle se tait (verdict absent, puis verdict `unreadable`).
//
// MUTATION : retirer la garde `filmDesTempsFortsLu` -> le slot disparait, rouge.
func TestSlotSansCreationResteSiLeFilmNEstPasProuveComplet(t *testing.T) {
	for _, verdict := range []string{"", DeathsFeedUnreadable} {
		doc, fb := documentDuFilmDeTest(t, true, Options{DeathsFeed: verdict})
		if !pisteDuSlot(doc, slotParasite) {
			t.Fatalf("verdict %q : le slot %d est ecarte alors que la completude n'est pas prouvee", verdict, slotParasite)
		}
		if got := fb.Compte(fallback.NomPositionDeBipedeSansCreationEcartee); got != 0 {
			t.Fatalf("verdict %q : %d positions comptees, attendu 0", verdict, got)
		}
	}
}

// TestFilmSansRecordDeCreationLuResteInchange : un film dont aucune creation n'est lue n'a aucun slot
// « sans record » au sens de la regle : ses pistes se publient comme avant.
//
// MUTATION : retirer la sortie `!lus` de `slotsSansRecordDeCreation` -> tous les slots disparaissent, rouge.
func TestFilmSansRecordDeCreationLuResteInchange(t *testing.T) {
	in := positionsAvecParasite()
	fb := fallback.NouveauCompteur()
	doc := BuildFromPositions(context.Background(), "test", "halo_infinite", in.Positions, nil,
		Options{FrameIntervalMS: 100, DeathsFeed: DeathsFeedRead, Fallbacks: fb})
	if !pisteDuSlot(doc, slotParasite) || !pisteDuSlot(doc, 100) || !pisteDuSlot(doc, 200) {
		t.Fatalf("un film sans creation lue perd des pistes : %+v", doc.Tracks)
	}
	if got := fb.Compte(fallback.NomPositionDeBipedeSansCreationEcartee); got != 0 {
		t.Fatalf("%d positions comptees, attendu 0", got)
	}
}

// TestSlotsSansRecordDeCreation : les slots candidats sont ceux qui portent des positions sans aucun
// record, indexe ou non ; un film sans record indexe n'en designe aucun.
func TestSlotsSansRecordDeCreation(t *testing.T) {
	pos := []grammar.BipedPosition{posAt(100, 1, 0, 0, 0), posAt(36, 2, 0, 0, 0), posAt(37, 3, 0, 0, 0)}
	indexe := creationDe(100, 0, 0)
	sansIndex := grammar.BipedCreation{Slot: 37, Generation: 1, TimestampUS: 0}
	if got := slotsSansRecordDeCreation(pos, nil); got != nil {
		t.Fatalf("sans creation : %v, attendu nil", got)
	}
	if got := slotsSansRecordDeCreation(pos, []grammar.BipedCreation{sansIndex}); got != nil {
		t.Fatalf("aucun record indexe : %v, attendu nil", got)
	}
	got := slotsSansRecordDeCreation(pos, []grammar.BipedCreation{indexe, sansIndex})
	if !reflect.DeepEqual(got, map[uint32]struct{}{36: {}}) {
		t.Fatalf("candidats = %v, attendu {36}: un record sans index ouvre quand meme son slot", got)
	}
}

// TestSlotsSansRecordNiNomGardeLesSlotsNommesOuSansVie : un slot qu'une vie nommee touche, ou qui n'a
// aucune vie au registre, n'est jamais ecarte ; seul l'est celui dont TOUTES les vies sont sans nom
// et sans record.
//
// MUTATION : compter une seule vie sans nom au lieu de toutes -> le slot 3 est ecarte, rouge.
func TestSlotsSansRecordNiNomGardeLesSlotsNommesOuSansVie(t *testing.T) {
	reg := IdentityRegistry{
		own: OwnerReport{lives: []lifeSpan{
			{slot: 1},                           // 0 : sans nom, sans record
			{slot: 2, xuid: 7},                  // 1 : nommee
			{slot: 3}, {slot: 3, bid: "bid(1)"}, // 2 et 3 : une sans nom, un bot
			{slot: 5}, // 4 : sans nom mais cause autre que sans_record
		}},
		creation: creationReport{causes: map[int]identity.LinkMethod{
			0: identity.MethodNoCreationRecord, 1: identity.MethodNoCreationRecord,
			2: identity.MethodNoCreationRecord, 3: identity.MethodNoCreationRecord,
			4: identity.MethodIndexOutOfTable,
		}},
	}
	got := reg.slotsSansRecordNiNom(map[uint32]struct{}{1: {}, 2: {}, 3: {}, 4: {}, 5: {}})
	if !reflect.DeepEqual(got, map[uint32]struct{}{1: {}}) {
		t.Fatalf("slots ecartes = %v, attendu {1} seul (4 n'a aucune vie, 2 est nomme, 3 porte un bot, 5 a un autre refus)", got)
	}
}
