package filmdec

import "testing"

// ability_rank_test.go — LE HOOK D'i48 PUBLIE-T-IL CE QUE LE DÉSERIALISEUR LIT ?
//
// Contrôle PUR (aucun film, aucune I/O) : il vaut en CI, contrairement à l'instrument
// i48_rank_test.go qui exige un film. Les deux se répondent — celui-ci fixe la grammaire sur
// des bits construits à la main, celui-là la confronte au corpus réel.
//
// CE QU'IL VERROUILLE, et pourquoi c'est le point sensible du lot du 2026-08-14 : le
// déserialiseur consommait les six bits de l'identité pour rester aligné et les JETAIT. La
// correction ne doit RIEN changer au nombre de bits consommés (4 porte ouverte, 10 porte
// fermée) — un décalage d'un seul bit désynchroniserait tout le reste du record.

// abilityBits construit un i48 isolé : R(3) compteur, R(1) porte, puis R(6) rang si la porte
// est fermée. La porte est INVERSÉE — le rang n'est présent que si son bit vaut 0.
func abilityBits(counter uint64, gate uint64, rank uint64) []byte {
	w := &bitWriter{}
	w.bits(counter, i48CounterBits)
	w.bits(gate, 1)
	if gate == 0 {
		w.bits(rank, i48RankBits)
	}
	w.bits(0x2A, 8) // queue de garde : un désalignement la déplacerait
	return w.buf
}

func TestConsumeBipedDesiredAbilitySetPublieLeRang(t *testing.T) {
	cases := []struct {
		nom       string
		counter   uint64
		gate      uint64
		rank      uint64
		wantRank  int
		wantWidth int
	}{
		{"porte fermée, rang 23 (champ de réparation)", 5, 0, 23, 23, 10},
		{"porte fermée, rang 8 (camouflage)", 0, 0, 8, 8, 10},
		{"porte fermée, rang 0", 7, 0, 0, 0, 10},
		{"porte fermée, rang maximal", 3, 0, 63, 63, 10},
		{"porte OUVERTE : aucune identité transmise", 6, 1, 0, AbilitySetNoRank, 4},
	}
	for _, c := range cases {
		t.Run(c.nom, func(t *testing.T) {
			var got struct {
				counter uint64
				rank    int
				width   int
				calls   int
			}
			prev := abilitySetHook
			SetAbilitySetHook(func(counter uint64, rank, width int) {
				got.counter, got.rank, got.width, got.calls = counter, rank, width, got.calls+1
			})
			defer SetAbilitySetHook(prev)

			br := NewBitReader(abilityBits(c.counter, c.gate, c.rank))
			consumeBipedDesiredAbilitySet(br)

			if got.calls != 1 {
				t.Fatalf("hook appelé %d fois, attendu 1", got.calls)
			}
			if br.BitPos() != c.wantWidth {
				t.Errorf("BITS CONSOMMÉS = %d, attendu %d — le déserialiseur s'est désaligné",
					br.BitPos(), c.wantWidth)
			}
			if got.width != c.wantWidth {
				t.Errorf("largeur publiée = %d, attendu %d", got.width, c.wantWidth)
			}
			if got.counter != c.counter {
				t.Errorf("compteur publié = %d, attendu %d", got.counter, c.counter)
			}
			if got.rank != c.wantRank {
				t.Errorf("rang publié = %d, attendu %d", got.rank, c.wantRank)
			}
		})
	}
}

// TestConsumeBipedDesiredAbilitySetSansHook vérifie que le déserialiseur consomme le MÊME
// nombre de bits quand aucune sonde n'est installée : la publication ne doit pas être ce qui
// fait avancer le curseur.
func TestConsumeBipedDesiredAbilitySetSansHook(t *testing.T) {
	prev := abilitySetHook
	SetAbilitySetHook(nil)
	defer SetAbilitySetHook(prev)

	for _, c := range []struct {
		gate uint64
		want int
	}{{0, 10}, {1, 4}} {
		br := NewBitReader(abilityBits(2, c.gate, 19))
		consumeBipedDesiredAbilitySet(br)
		if br.BitPos() != c.want {
			t.Errorf("porte %d : %d bits consommés, attendu %d", c.gate, br.BitPos(), c.want)
		}
	}
}
