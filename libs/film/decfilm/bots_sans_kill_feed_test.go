package decfilm

// bots_sans_kill_feed_test.go — LE ROSTER DE BOTS D UN FILM SANS KILL-FEED, sur des films
// synthetiques de paquets BOT_METADATA (type 12).
//
//	SKF-RELAIS     trois bots qui se relaient sur l index 8 : chacun garde SON intervalle, l ordre est
//	               celui du slot puis de la decouverte (les memes attendus que le test de relais de
//	               `killsource`, dont ce fichier est le jumeau) ;
//	SKF-INCOMPLET  un paquet dont le balayage ne retrouve pas `nbBots` entrees ne ferme rien ;
//	SKF-GABARIT    le gabarit de `5da6fd30` : trois bots arrives un a un, un quatrieme au dernier paquet ;
//	SKF-NON-EPINGLE un slot hors des 32 indices est dit non epingle, et reste au roster ;
//	SKF-AUCUN-BOT  aucun bot declare, ou pas de film : nil, comme le resultat d avant.

import (
	"encoding/binary"
	"reflect"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/facts/killsource"
	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar"
	"github.com/JGtm/halo-infinite-film-decoder/internal/source"
)

const (
	// strideEntreeBotMeta : l ecart, en octets, entre deux entrees fabriquees (le lecteur est
	// bit-precis et ne suppose aucun stride).
	strideEntreeBotMeta = 2076
	// nomApresLeSlot : l ecart, en octets, du slot au nom d une entree (`0x74`).
	nomApresLeSlot = 0x74
	// enTeteDePaquet : `[u16 type][2 octets][u32 taille][u64 horodatage]`, petit-boutiste.
	enTeteDePaquet = 16
)

// botDeTest : un bot declare par un paquet fabrique.
type botDeTest struct {
	slot, bid int
	nom       string
}

// paquetDeTest : un paquet BOT_METADATA fabrique, a son chunk et son horodatage.
type paquetDeTest struct {
	chunk int
	ts    uint64
	nb    int
	bots  []botDeTest
}

func payloadBotMeta(nb int, bots []botDeTest) []byte {
	if nb == 0 && len(bots) == 0 {
		return []byte{0, 0, 0, 0}
	}
	pl := make([]byte, 4+len(bots)*strideEntreeBotMeta+256)
	binary.BigEndian.PutUint32(pl, uint32(nb)) //nolint:gosec // compte de test borne
	for k, b := range bots {
		s := 4 + k*strideEntreeBotMeta
		binary.BigEndian.PutUint32(pl[s:], uint32(b.slot))  //nolint:gosec // slot de test borne
		binary.BigEndian.PutUint32(pl[s+4:], uint32(b.bid)) //nolint:gosec // bid de test borne
		nom := s + nomApresLeSlot
		for i, c := range []byte(b.nom) {
			pl[nom+2*i+1] = c
		}
	}
	return pl
}

// filmDePaquets fabrique un film en memoire dont chaque chunk porte, dans leur ordre, les paquets de
// type 12 de sa position.
func filmDePaquets(t *testing.T, ps ...paquetDeTest) *source.Film {
	t.Helper()
	n := 0
	for _, p := range ps {
		n = max(n, p.chunk+1)
	}
	chunks := make(source.MemoryChunks, n)
	for _, p := range ps {
		pl := payloadBotMeta(p.nb, p.bots)
		b := make([]byte, enTeteDePaquet+len(pl))
		binary.LittleEndian.PutUint16(b, uint16(grammar.PacketTypeBotMetadata))
		binary.LittleEndian.PutUint32(b[4:], uint32(len(pl))) //nolint:gosec // payload de test borne
		binary.LittleEndian.PutUint64(b[8:], p.ts)
		copy(b[enTeteDePaquet:], pl)
		chunks[p.chunk] = append(chunks[p.chunk], b...)
	}
	f, err := source.Load(chunks, nil)
	if err != nil {
		t.Fatalf("film synthetique : %v", err)
	}
	return f
}

func entreesDe(r *killsource.Result) []killsource.BotEntry {
	if r == nil {
		return nil
	}
	return r.Roster.Bots
}

func TestBotsSansKillFeedRelais(t *testing.T) {
	hundy := botDeTest{8, 16, "343 Hundy"}
	pardon := botDeTest{8, 7, "343 PardonMy"}
	brew := botDeTest{8, 19, "343 Brew Dog"}
	r := resultatDeBotsSansKillFeed(filmDePaquets(t,
		paquetDeTest{chunk: 0, ts: 1_000, nb: 1, bots: []botDeTest{hundy}},
		paquetDeTest{chunk: 1, ts: 21_000, nb: 1, bots: []botDeTest{hundy}},
		paquetDeTest{chunk: 1, ts: 27_300, nb: 0},
		paquetDeTest{chunk: 4, ts: 81_300, nb: 1, bots: []botDeTest{pardon}},
		paquetDeTest{chunk: 4, ts: 83_100, nb: 0},
		paquetDeTest{chunk: 17, ts: 315_500, nb: 1, bots: []botDeTest{brew}},
		paquetDeTest{chunk: 18, ts: 321_300, nb: 1, bots: []botDeTest{brew}},
	))
	if r == nil {
		t.Fatal("resultat nil : trois bots declares")
	}
	got := map[string][]killsource.BotDeclaration{}
	var noms []string
	for _, b := range entreesDe(r) {
		noms = append(noms, b.Name)
		got[b.Name] = b.Declarations
		if b.Team != nil {
			t.Errorf("%s : equipe %d lue sans largeur de personnalisation au profil", b.Name, *b.Team)
		}
	}
	if want := []string{"343 Hundy", "343 PardonMy", "343 Brew Dog"}; !reflect.DeepEqual(noms, want) {
		t.Fatalf("ordre des bots %v, attendu %v", noms, want)
	}
	want := map[string][]killsource.BotDeclaration{
		"343 Hundy":    {{FromUS: 1_000, ToUS: 27_300}},
		"343 PardonMy": {{FromUS: 81_300, ToUS: 83_100}},
		"343 Brew Dog": {{FromUS: 315_500}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("declarations %+v, attendu %+v", got, want)
	}
	if b := r.Roster.BotEquipes; b.Illisibles != 3 || !b.PersoInconnue {
		t.Errorf("bilan des equipes %+v : trois bots illisibles, personnalisation inconnue attendues", b)
	}
	if len(r.Roster.UnpinnedBots) != 0 {
		t.Errorf("bots non epingles %+v, attendu aucun", r.Roster.UnpinnedBots)
	}
	if !reflect.DeepEqual(r.ProfilCalibre, killsource.ProfilDeDepart()) {
		t.Errorf("profil %+v : le resultat porte le profil de depart, comme le resultat nil d avant", r.ProfilCalibre)
	}
}

func TestBotsSansKillFeedIncomplet(t *testing.T) {
	a, b := botDeTest{8, 59, "343 Hollis"}, botDeTest{9, 19, "343 Brew Dog"}
	r := resultatDeBotsSansKillFeed(filmDePaquets(t,
		paquetDeTest{chunk: 0, ts: 1_000, nb: 2, bots: []botDeTest{a, b}},
		paquetDeTest{chunk: 1, ts: 21_000, nb: 2, bots: []botDeTest{a}}, // il en annonce deux, n en porte qu un
	))
	if r == nil || r.Roster.BotPaquetsIncomplets != 1 {
		t.Fatalf("resultat %+v : un paquet incomplet attendu", r)
	}
	for _, e := range r.Roster.Bots {
		if want := []killsource.BotDeclaration{{FromUS: 1_000}}; !reflect.DeepEqual(e.Declarations, want) {
			t.Errorf("%s : declarations %+v, attendu %+v (un paquet incomplet ne ferme rien)", e.Name, e.Declarations, want)
		}
	}
}

func TestBotsSansKillFeedGabaritDe5da6fd30(t *testing.T) {
	hollis, brew := botDeTest{8, 59, "343 Hollis"}, botDeTest{9, 19, "343 Brew Dog"}
	referee, grapple := botDeTest{10, 51, "343 The Referee"}, botDeTest{11, 13, "343 GrappleMans"}
	r := resultatDeBotsSansKillFeed(filmDePaquets(t,
		paquetDeTest{chunk: 1, ts: 5_279_835_622, nb: 0},
		paquetDeTest{chunk: 2, ts: 5_306_466_741, nb: 1, bots: []botDeTest{hollis}},
		paquetDeTest{chunk: 3, ts: 5_319_823_878, nb: 3, bots: []botDeTest{hollis, brew, referee}},
		paquetDeTest{chunk: 3, ts: 5_323_688_182, nb: 4, bots: []botDeTest{hollis, brew, referee, grapple}},
	))
	if r == nil {
		t.Fatal("resultat nil : quatre bots declares")
	}
	type lu struct {
		slot, bid int
		depuis    uint64
	}
	var got []lu
	for _, b := range r.Roster.Bots {
		if len(b.Declarations) != 1 || b.Declarations[0].ToUS != 0 {
			t.Fatalf("%s : declarations %+v, attendu une seule, encore ouverte", b.Name, b.Declarations)
		}
		got = append(got, lu{b.Slot, b.BotID, b.Declarations[0].FromUS})
	}
	want := []lu{{8, 59, 5_306_466_741}, {9, 19, 5_319_823_878}, {10, 51, 5_319_823_878}, {11, 13, 5_323_688_182}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("bots %+v, attendu %+v", got, want)
	}
}

func TestBotsSansKillFeedSlotHorsDesTrenteDeuxIndices(t *testing.T) {
	r := resultatDeBotsSansKillFeed(filmDePaquets(t,
		paquetDeTest{chunk: 0, ts: 1_000, nb: 2, bots: []botDeTest{{8, 59, "343 Hollis"}, {40, 7, "343 Hors Table"}}},
	))
	if r == nil || len(r.Roster.Bots) != 2 {
		t.Fatalf("resultat %+v : deux bots au roster attendus", r)
	}
	if len(r.Roster.UnpinnedBots) != 1 || r.Roster.UnpinnedBots[0].Slot != 40 {
		t.Fatalf("bots non epingles %+v, attendu le seul slot 40", r.Roster.UnpinnedBots)
	}
}

func TestBotsSansKillFeedAucunBot(t *testing.T) {
	if r := resultatDeBotsSansKillFeed(nil); r != nil {
		t.Errorf("film nil : %+v, attendu nil", r)
	}
	if r := resultatDeBotsSansKillFeed(filmDePaquets(t,
		paquetDeTest{chunk: 0, ts: 1_000, nb: 0}, paquetDeTest{chunk: 1, ts: 2_000, nb: 0})); r != nil {
		t.Errorf("aucun bot declare : %+v, attendu nil", r)
	}
	if r := ResultatDeBotsSansKillFeed(nil); r != nil {
		t.Errorf("renvoi de la facade, film nil : %+v, attendu nil", r)
	}
}
