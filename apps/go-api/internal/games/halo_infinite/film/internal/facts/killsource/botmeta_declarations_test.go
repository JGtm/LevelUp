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
	"reflect"
	"testing"
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
		nom := s + botSlotBackBits/8
		for i, c := range []byte(b.Name) {
			pl[nom+2*i+1] = c
		}
	}
	return pl
}

func ecrireU32BE(d []byte, at int, v uint32) {
	d[at], d[at+1], d[at+2], d[at+3] = byte(v>>24), byte(v>>16), byte(v>>8), byte(v)
}

func filmDePaquetsBotMeta(paquets ...packet) *film {
	for i := range paquets {
		paquets[i].typ = packetTypeBotMeta
	}
	return &film{packets: paquets}
}

// TestBotMetaDeclarationsDesRelais execute DECL-RELAIS.
func TestBotMetaDeclarationsDesRelais(t *testing.T) {
	hundy := bot{Slot: 8, BotID: 16, Name: "343 Hundy"}
	pardon := bot{Slot: 8, BotID: 7, Name: "343 PardonMy"}
	brew := bot{Slot: 8, BotID: 19, Name: "343 Brew Dog"}
	f := filmDePaquetsBotMeta(
		packet{chunk: 0, ts: 1_000, payload: payloadBotMeta(1, hundy)},
		packet{chunk: 1, ts: 21_000, payload: payloadBotMeta(1, hundy)},
		packet{chunk: 1, ts: 27_300, payload: payloadBotMeta(0)}, // Hundy retire : son depart
		packet{chunk: 4, ts: 81_300, payload: payloadBotMeta(1, pardon)},
		packet{chunk: 4, ts: 83_100, payload: payloadBotMeta(0)},
		packet{chunk: 17, ts: 315_500, payload: payloadBotMeta(1, brew)}, // paquet de changement
		packet{chunk: 18, ts: 321_300, payload: payloadBotMeta(1, brew)},
	)
	m := loadBotMeta(f)
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
	f := filmDePaquetsBotMeta(
		packet{ts: 1_000, payload: payloadBotMeta(2, a, b)},
		packet{ts: 2_000, payload: payloadBotMeta(2, a)}, // nbBots dit 2, une seule entree lue
		packet{ts: 3_000, payload: payloadBotMeta(1, a)}, // complet : b est parti ici
	)
	m := loadBotMeta(f)
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
	f := &film{packets: []packet{
		{chunk: 6, idx: 1, typ: packetTypeKeyframe, ts: 1_000_000},
		{chunk: 6, idx: 4, typ: packetTypeBotMeta, ts: 1_000_390, payload: payloadBotMeta(1, pardon)},
		{chunk: 6, idx: 5, typ: packetType0, ts: 1_100_000},
		{chunk: 6, idx: 9, typ: packetTypeBotMeta, ts: 2_800_000, payload: payloadBotMeta(0)},
	}}
	m := loadBotMeta(f)
	want := []BotDeclaration{{FromUS: 1_000_000, ToUS: 2_800_000}}
	if len(m.Bots) != 1 || !reflect.DeepEqual(m.Bots[0].entree().Declarations, want) {
		t.Fatalf("declarations %+v, attendu %+v : l ouverture en tete de chunk est l instant de "+
			"l image-cle, la fermeture en milieu de chunk garde le sien", m.Bots, want)
	}
}
