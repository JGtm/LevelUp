package replay

// porteurs_au_sync_test.go — LA GARDE DE MODE ET LA RELECTURE DES CALQUES DE L'ENTREE DU SYNC
// (lot V1.4 du plan `.ai/PLAN_EMPRISE_VIES_2026-09-28.md`). La fidelite aux documents de rejeu
// sur de vrais films est mesuree par `emprise_v1_porteurs_research_test.go`.

import (
	"context"
	"testing"
)

// TestPortagesAuSync_HorsModeAPorteurNeLitRien — LA GARDE REND AVANT LE PREMIER OCTET.
//
// Film et contexte sont NIL : la moindre lecture paniquerait. Un Team Slayer ne paie donc rien,
// ce que le V0 a mesure sur les dix films du 22/09 qui ne sont pas d'un mode a porteur.
func TestPortagesAuSync_HorsModeAPorteurNeLitRien(t *testing.T) {
	for _, variante := range []string{"Team Slayer:Arena", "Slayer:Arena Super Fiesta", "KOTH:Arena", ""} {
		t.Run(variante, func(t *testing.T) {
			portages, b := PortagesAuSync(context.Background(), EntreePorteursAuSync{
				MatchID: "m", Variante: variante,
			})
			// UN BILAN VIERGE : ni lecture ni ASSEMBLAGE (un document assemble sans calage dirait
			// `SansCalage`) — le cout d'un match sans porteur est nul, pas seulement faible.
			if portages != nil || b != (BilanPortages{}) {
				t.Fatalf("variante %q : portages %v, bilan %+v — attendu aucune lecture ni assemblage",
					variante, portages, b)
			}
		})
	}
}

// TestGardesDeLaVariante_Familles — une famille par variante, et la meme garde que la cuisson.
func TestGardesDeLaVariante_Familles(t *testing.T) {
	for _, c := range []struct {
		variante string
		attendu  GardesDesPorteurs
	}{
		{"CTF:Arena", GardesDesPorteurs{Drapeau: true}},
		{"Big Team Battle:CTF", GardesDesPorteurs{Drapeau: true}},
		{"Oddball:Arena", GardesDesPorteurs{Crane: true}},
		{"Assault:Neutral Bomb", GardesDesPorteurs{Bombe: true}},
		{"Arena:VIP", GardesDesPorteurs{VIP: true}},
		{"Team Slayer:Arena", GardesDesPorteurs{}},
	} {
		if got := GardesDeLaVariante(c.variante); got != c.attendu {
			t.Errorf("GardesDeLaVariante(%q) = %+v, attendu %+v", c.variante, got, c.attendu)
		}
	}
}

// docDePorteurs : un document a l'axe de 100 ms, cale de 4 000 ms (horloge du film = horloge du
// match + 4 000).
func docDePorteurs() ReplayDocument {
	calage := int64(4_000)
	xa, xb := "111", "222"
	return ReplayDocument{
		FrameIntervalMS: 100,
		Coverage:        &Coverage{Bridge: BridgeHealth{DeathOffsetMs: &calage}},
		FlagCarries: []FlagCarry{{Spans: []FlagSpan{
			{State: FlagStateCarried, XUID: &xa, T0: 10, T1: 20},
			{State: FlagStateCarriedOpen, XUID: &xb, T0: 30, T1: 90},
			{State: FlagStateDropped, T0: 21, T1: 29},
		}}},
		SkullCarries: []SkullCarry{{XUID: "222", T0: 50, T1: 60}, {XUID: "bid(1.0)", T0: 1, T1: 2}},
	}
}

// TestPortagesDuDocument_HorlogeDuMatch — les frames deviennent des ms du MATCH par l'origine du
// document et SON calage ; `carried_open` et les porteurs non decimaux sont ecartes et comptes.
//
// Origine 10 000 000 µs (10 s de film) : la frame 10 est a 11 s de film, donc 7 000 ms de match.
func TestPortagesDuDocument_HorlogeDuMatch(t *testing.T) {
	var b BilanPortages
	got := portagesDuDocument(docDePorteurs(), 10_000_000, &b)
	if l := got[111]; len(l) != 1 || l[0] != (IntervalleDePort{DebutMS: 7_000, FinMS: 8_000}) {
		t.Fatalf("drapeau de 111 = %+v, attendu [7 000, 8 000] ms du match", l)
	}
	if l := got[222]; len(l) != 1 || l[0] != (IntervalleDePort{DebutMS: 11_000, FinMS: 12_000}) {
		t.Fatalf("portages de 222 = %+v, attendu le seul crane [11 000, 12 000] (le drapeau "+
			"`carried_open` est une borne haute)", l)
	}
	if b.Intervalles != 2 || b.DrapeauOuverts != 1 || b.XUIDIllisibles != 1 {
		t.Fatalf("bilan %+v, attendu 2 intervalles, 1 drapeau ouvert, 1 xuid illisible", b)
	}
}

// TestPortagesDuDocument_SansCalage — sans calage, aucune date : rien ne sort, et le bilan le dit.
func TestPortagesDuDocument_SansCalage(t *testing.T) {
	doc := docDePorteurs()
	doc.Coverage.Bridge.DeathOffsetMs = nil
	var b BilanPortages
	if got := portagesDuDocument(doc, 10_000_000, &b); got != nil || !b.SansCalage {
		t.Fatalf("portages %v, sans calage = %v : attendu rien et le refus dit", got, b.SansCalage)
	}
}

// TestPortagesAuSync_ChaqueFamilleNePaieQueSesLectures — la couronne ne paie que le statborg, le
// crane le statborg (et son pont), la bombe le seul canal des armes tenues. Les lectures du
// drapeau (equipes, objets du monde) sont les plus cheres du lot V0 : elles ne se paient QUE sur
// un CTF. Film et contexte nuls : les balayages rendent vide, seul le fait de lire est mesure.
func TestPortagesAuSync_ChaqueFamilleNePaieQueSesLectures(t *testing.T) {
	for _, c := range []struct {
		variante string
		lu       LecturesDesPorteurs
	}{
		{"Arena:VIP", LecturesDesPorteurs{Statborg: true}},
		{"Oddball:Arena", LecturesDesPorteurs{Statborg: true}},
		{"Assault:Neutral Bomb", LecturesDesPorteurs{ArmesTenues: true}},
	} {
		t.Run(c.variante, func(t *testing.T) {
			_, b := PortagesAuSync(context.Background(), EntreePorteursAuSync{MatchID: "m", Variante: c.variante})
			if b.Lectures != c.lu {
				t.Fatalf("lectures %+v, attendu %+v", b.Lectures, c.lu)
			}
		})
	}
}
