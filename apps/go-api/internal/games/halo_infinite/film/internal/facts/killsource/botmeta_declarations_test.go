package killsource

// botmeta_declarations_test.go — BOT_METADATA GARDE L INSTANT DE SES PAQUETS (lot M2.1, 2026-09-23).
//
//	DECL-RELAIS   Au gabarit de la sonde P4 (`b1ad85eb`) : trois bots qui se relaient sur l index
//	              8, chacun declare puis retire par un paquet de 4 octets (`nbBots=0`) — chaque
//	              bot garde SON intervalle, date au paquet pres. L agregat (bots, ordre, NBots)
//	              reste celui d avant : aucune ligne de kill ne bouge.
//	DECL-INCOMPLET Un paquet dont le scan ne retrouve pas `nbBots` entrees n est pas un depart :
//	              il ne ferme rien, et il se compte.

import (
	"encoding/binary"
	"reflect"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// strideEntreeBotMeta : l ecart, en octets, entre deux entrees fabriquees. Le lecteur est
// bit-precis et ne suppose aucun stride (cf. botmeta.go) : des entrees alignees a l octet lui
// suffisent, et c est ce qui rend la fabrique simple.
const strideEntreeBotMeta = 2076

// payloadBotMeta fabrique un payload de type 12 : `nbBots` puis une entree par bot (slot et bid
// en big-endian, nom en UTF-16BE ferme par 0x0000, aux offsets que le lecteur mesure).
func payloadBotMeta(nbBots int, bots ...bot) []byte {
	if nbBots == 0 && len(bots) == 0 {
		return []byte{0, 0, 0, 0}
	}
	pl := make([]byte, 4+len(bots)*strideEntreeBotMeta+256)
	ecrireU32BE(pl, 0, uint32(nbBots))
	for k, b := range bots {
		s := 4 + k*strideEntreeBotMeta
		ecrireU32BE(pl, s, uint32(b.Slot))
		ecrireU32BE(pl, s+4, uint32(b.BotID))
		nom := s + nomApresLeSlot
		for i, c := range []byte(b.Name) {
			pl[nom+2*i+1] = c
		}
	}
	return pl
}

func ecrireU32BE(d []byte, at int, v uint32) {
	d[at], d[at+1], d[at+2], d[at+3] = byte(v>>24), byte(v>>16), byte(v>>8), byte(v)
}

// nomApresLeSlot : l ecart, en octets, du slot au nom d une entree (`0x74`, cf. `grammar/bot_metadata.go`).
const nomApresLeSlot = 0x74

// paquetDeTest decrit un paquet d un film synthetique : son chunk, son rang, son type, son
// horodatage et son payload.
type paquetDeTest struct {
	chunk, idx, typ int
	ts              uint64
	payload         []byte
}

// sourceDePaquets fabrique un film en memoire dont chaque chunk porte, dans leur ordre, les paquets
// `ps` de sa position : en-tetes de 16 octets `[u16 type][2 octets][u32 taille][u64 horodatage]`,
// petit-boutistes. Un payload vide est porte a un octet : la source arrete un chunk sur un en-tete de
// taille nulle qui n est pas son terminateur.
func sourceDePaquets(t *testing.T, ps ...paquetDeTest) *source.Film {
	t.Helper()
	n := 0
	for _, p := range ps {
		n = max(n, p.chunk+1)
	}
	chunks := make(source.MemoryChunks, n)
	for _, p := range ps {
		pl := p.payload
		if len(pl) == 0 {
			pl = []byte{0}
		}
		b := make([]byte, enTeteDePaquet+len(pl))
		binary.LittleEndian.PutUint16(b, uint16(p.typ))       //nolint:gosec // type de paquet de test
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

// botsDePaquets rend l agregat des bots d un film synthetique de paquets BOT_METADATA, lus par la
// grammaire.
func botsDePaquets(t *testing.T, paquets ...paquetDeTest) botMeta {
	t.Helper()
	for i := range paquets {
		paquets[i].typ = grammar.PacketTypeBotMetadata
	}
	return loadBotMeta(grammar.PaquetsBotMetadata(sourceDePaquets(t, paquets...), 0, false))
}

// TestBotMetaDeclarationsDesRelais execute DECL-RELAIS.
func TestBotMetaDeclarationsDesRelais(t *testing.T) {
	hundy := bot{Slot: 8, BotID: 16, Name: "343 Hundy"}
	pardon := bot{Slot: 8, BotID: 7, Name: "343 PardonMy"}
	brew := bot{Slot: 8, BotID: 19, Name: "343 Brew Dog"}
	m := botsDePaquets(t,
		paquetDeTest{chunk: 0, ts: 1_000, payload: payloadBotMeta(1, hundy)},
		paquetDeTest{chunk: 1, ts: 21_000, payload: payloadBotMeta(1, hundy)},
		paquetDeTest{chunk: 1, ts: 27_300, payload: payloadBotMeta(0)}, // Hundy retire : son depart
		paquetDeTest{chunk: 4, ts: 81_300, payload: payloadBotMeta(1, pardon)},
		paquetDeTest{chunk: 4, ts: 83_100, payload: payloadBotMeta(0)},
		paquetDeTest{chunk: 17, ts: 315_500, payload: payloadBotMeta(1, brew)}, // paquet de changement
		paquetDeTest{chunk: 18, ts: 321_300, payload: payloadBotMeta(1, brew)},
	)
	if m.NBots != 1 || m.NPkt != 7 || m.Incomplets != 0 {
		t.Fatalf("agregat : NBots %d NPkt %d incomplets %d, attendu 1 / 7 / 0", m.NBots, m.NPkt, m.Incomplets)
	}
	noms := make([]string, 0, len(m.Bots))
	got := map[string][]BotDeclaration{}
	for _, b := range m.Bots {
		noms = append(noms, b.Name)
		got[b.Name] = b.entree().Declarations
	}
	if !reflect.DeepEqual(noms, []string{"343 Hundy", "343 PardonMy", "343 Brew Dog"}) {
		t.Fatalf("ordre des bots %v : l agregat d avant les rendait dans l ordre de decouverte", noms)
	}
	want := map[string][]BotDeclaration{
		"343 Hundy":    {{FromUS: 1_000, ToUS: 27_300}},
		"343 PardonMy": {{FromUS: 81_300, ToUS: 83_100}},
		"343 Brew Dog": {{FromUS: 315_500}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("declarations :\n obtenu %+v\n attendu %+v", got, want)
	}
}

// TestBotMetaPaquetIncompletNeFermeRien execute DECL-INCOMPLET.
func TestBotMetaPaquetIncompletNeFermeRien(t *testing.T) {
	a := bot{Slot: 8, BotID: 16, Name: "343 Hundy"}
	b := bot{Slot: 9, BotID: 7, Name: "343 PardonMy"}
	m := botsDePaquets(t,
		paquetDeTest{ts: 1_000, payload: payloadBotMeta(2, a, b)},
		paquetDeTest{ts: 2_000, payload: payloadBotMeta(2, a)}, // nbBots dit 2, une seule entree lue
		paquetDeTest{ts: 3_000, payload: payloadBotMeta(1, a)}, // complet : b est parti ici
	)
	if m.Incomplets != 1 {
		t.Fatalf("paquets incomplets : %d, attendu 1", m.Incomplets)
	}
	for _, bt := range m.Bots {
		d := bt.entree().Declarations
		switch bt.Name {
		case "343 Hundy":
			if !reflect.DeepEqual(d, []BotDeclaration{{FromUS: 1_000}}) {
				t.Errorf("Hundy : %+v, attendu une declaration ouverte depuis 1000", d)
			}
		case "343 PardonMy":
			if !reflect.DeepEqual(d, []BotDeclaration{{FromUS: 1_000, ToUS: 3_000}}) {
				t.Errorf("PardonMy : %+v — le paquet incomplet de 2000 ne devait RIEN fermer", d)
			}
		}
	}
}

// TestBotMetaInstantaneDeTete (DECL-TETE) : le BOT_METADATA de tete de chunk porte l etat A
// L IMAGE-CLE (mesure `b1ad85eb` : ecrit 390 us apres elle) ; un paquet de changement ecrit apres
// la premiere trame garde son propre instant.
func TestBotMetaInstantaneDeTete(t *testing.T) {
	pardon := bot{Slot: 8, BotID: 7, Name: "343 PardonMy"}
	m := loadBotMeta(grammar.PaquetsBotMetadata(sourceDePaquets(t,
		paquetDeTest{chunk: 6, idx: 1, typ: int(grammar.PacketTypeKeyframe), ts: 1_000_000},
		paquetDeTest{chunk: 6, idx: 4, typ: grammar.PacketTypeBotMetadata, ts: 1_000_390, payload: payloadBotMeta(1, pardon)},
		paquetDeTest{chunk: 6, idx: 5, typ: packetType0, ts: 1_100_000},
		paquetDeTest{chunk: 6, idx: 9, typ: grammar.PacketTypeBotMetadata, ts: 2_800_000, payload: payloadBotMeta(0)},
	), 0, false))
	want := []BotDeclaration{{FromUS: 1_000_000, ToUS: 2_800_000}}
	if len(m.Bots) != 1 || !reflect.DeepEqual(m.Bots[0].entree().Declarations, want) {
		t.Fatalf("declarations %+v, attendu %+v : l ouverture en tete de chunk est l instant de "+
			"l image-cle, la fermeture en milieu de chunk garde le sien", m.Bots, want)
	}
}
