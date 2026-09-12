package filmdec

import (
	"math/rand"
	"testing"
)

// TestCaptureConsumesSameBitsAsDispatch — GARDE-RAIL de la double étiquette.
//
// Le nom de chaque composant capturé apparaît dans DEUX switch : consumeByName (le
// dispatch historique) et consumeByNameCapturing (la couche de capture). Les deux doivent
// consommer EXACTEMENT le même nombre de bits sur n'importe quelle entrée, sinon la capture
// désynchroniserait le reste du record — le pire des bugs de ce décodeur, et le plus
// silencieux.
//
// Le test balaie captureNames : ajouter un nom à la couche de capture sans l'aligner sur le
// dispatch fait échouer ce test, sans qu'on ait à penser à l'ajouter ici.
func TestCaptureConsumesSameBitsAsDispatch(t *testing.T) {
	rng := rand.New(rand.NewSource(20260726))
	buf := make([]byte, 64)
	for _, name := range captureNames {
		for iter := 0; iter < 500; iter++ {
			for i := range buf {
				buf[i] = byte(rng.Intn(256))
			}
			a := NewBitReader(buf)
			_, _, portedA := consumeByName(a, name, BipedTypeIndex, 0)
			b := NewBitReader(buf)
			_, _, payload, portedB := consumeByNameCapturing(b, name, BipedTypeIndex, 0)
			if portedA != portedB {
				t.Fatalf("%s : ported diverge (dispatch=%v capture=%v)", name, portedA, portedB)
			}
			if a.BitPos() != b.BitPos() {
				t.Fatalf("%s (itération %d) : le dispatch consomme %d bits, la capture %d",
					name, iter, a.BitPos(), b.BitPos())
			}
			if payload == nil {
				t.Fatalf("%s : la couche de capture ne rend aucune valeur", name)
			}
		}
	}
}

// TestCaptureAccessorsTypeSafety vérifie que les accesseurs typés ne se confondent pas :
// un ShieldOf sur une capture de santé doit rendre false, pas une valeur muette.
func TestCaptureAccessorsTypeSafety(t *testing.T) {
	body := CompResult{Payload: BodyVitality{Health: 0.5}}
	if _, ok := body.ShieldOf(); ok {
		t.Fatal("ShieldOf accepte une BodyVitality")
	}
	if v, ok := body.BodyOf(); !ok || v.Health != 0.5 {
		t.Fatalf("BodyOf rend %v, %v", v, ok)
	}
	if _, ok := (CompResult{}).RespawnOf(); ok {
		t.Fatal("RespawnOf accepte un Payload nil")
	}
}

// TestDecodeShieldVitalityBitCost fige les quatre coûts en bits du composant i5 selon ses
// deux portes internes (29 / 31 / 43 / 55), tels que relus au désassemblage de FUN_140d50cbc.
// Un changement de grammaire non voulu se voit ici avant de désynchroniser un film.
func TestDecodeShieldVitalityBitCost(t *testing.T) {
	// Construit un flux dont on maîtrise les portes : q(8) | présence(1) [| g0(1) [12] | g1(1) [12]] | 16 | 4
	cases := []struct {
		presence, g0, g1 bool
		want             int
	}{
		// Coûts NOMINAUX (sans le R(1) de corruption-check, désactivé hors mode film) :
		// 29 / 31 / 43 / 55 bits. Les commentaires historiques du paquet annoncent
		// « 25 / 27 / 39 / 51 » : MÊMES ÉCARTS (+2, +12, +12), décalés de 4 — ils oublient
		// les quatre R(1) de queue. Le CODE, lui, les a toujours lus ; c'est donc bien la
		// note qui était fausse, pas la consommation. Corrigée dans components_object.go.
		{false, false, false, 8 + 1 + 16 + 4},                // 29
		{true, false, false, 8 + 1 + 1 + 1 + 16 + 4},         // 31
		{true, true, false, 8 + 1 + 1 + 12 + 1 + 16 + 4},     // 43
		{true, true, true, 8 + 1 + 1 + 12 + 1 + 12 + 16 + 4}, // 55
	}
	for _, c := range cases {
		bits := []int{}
		push := func(v uint64, n int) {
			for i := n - 1; i >= 0; i-- {
				bits = append(bits, int((v>>uint(i))&1))
			}
		}
		push(0xA5, 8)
		push(b2u(c.presence), 1)
		if c.presence {
			push(b2u(c.g0), 1)
			if c.g0 {
				push(0x123, 12)
			}
			push(b2u(c.g1), 1)
			if c.g1 {
				push(0x456, 12)
			}
		}
		push(0xBEEF, 16)
		push(0b1010, 4)
		buf := make([]byte, (len(bits)+7)/8)
		for i, b := range bits {
			if b != 0 {
				buf[i>>3] |= 1 << (7 - uint(i&7))
			}
		}
		br := NewBitReader(buf)
		got := decodeObjectShieldVitality(br)
		if br.BitPos() != c.want {
			t.Fatalf("portes %v/%v/%v : %d bits consommés, attendu %d", c.presence, c.g0, c.g1, br.BitPos(), c.want)
		}
		if got.Q != 0xA5 || got.Block64 != 0xBEEF {
			t.Fatalf("portes %v/%v/%v : champs mal placés (Q=%02x Block64=%04x)", c.presence, c.g0, c.g1, got.Q, got.Block64)
		}
		if got.F66 != true || got.F67 != false || got.F69 != true || got.F68 != false {
			t.Fatalf("ordre des drapeaux de queue cassé : %v %v %v %v", got.F66, got.F67, got.F69, got.F68)
		}
	}
}

func b2u(b bool) uint64 {
	if b {
		return 1
	}
	return 0
}
