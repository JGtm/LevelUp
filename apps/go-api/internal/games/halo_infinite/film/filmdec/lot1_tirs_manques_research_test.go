package filmdec

// lot1_tirs_manques_research_test.go — LOT 1 : LES TIRS MANQUES (demande utilisateur). Un tir
// manque = un action_weapon_fire (type 36, octet 0xD2) qui ne touche personne, c.-a-d. dont la
// LISTE DE CIBLES est vide. Le compte de cibles est un champ du type 36 ; il devient fiable une
// fois le cadrage complet du type 36 ferme (agent Ghidra en cours sur la visee/preambule).
//
// CE QUI EST MESURABLE MAINTENANT, sans le cadrage complet : le DENOMINATEUR de la precision —
// le nombre de TIRS par tireur. L'attaquant du tir est lu par le decodeur de production
// (fire_events : R(5) a l'offset FIXE bit 36, valeur >> 1 = index de tireur 0..15). On compte
// les tirs par tireur, et on publie le total de tirs vs le total d'evenements de degat
// (damage_aftermath) — le rapport donne une premiere idee de la part de tirs qui portent.
//
// LA DEFINITION FINE (miss = tir a 0 cible) attend le decodage complet du type 36 ; cette etape
// pose le denominateur et le cadre. Garde LOT1_TRAME_FILM.

import (
	"os"
	"testing"
)

func TestLot1TirsManques(t *testing.T) {
	dir := os.Getenv(lot1TrameFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument saute", lot1TrameFilmEnv)
	}
	release := LockProcessDecode()
	defer release()
	n := CountFilmChunks(dir)
	if n > deltaWitnessChunks {
		n = deltaWitnessChunks
	}
	var (
		tirs, degats int
		parTireur    = map[uint64]int{}
	)
	for c := 1; c <= n; c++ {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, pk := range WalkPackets(data) {
			if pk.Type != PacketTypeDelta || pk.Size < 4 {
				continue
			}
			pay := pk.Payload(data)
			switch pay[0] {
			case 0xD2: // action_weapon_fire
				br := NewBitReader(pay)
				br.Skip(2)
				if br.ReadBits(7) != 36 {
					continue
				}
				tirs++
				// Attaquant : offset FIXE de production (fire_events), R(5)>>1 a bit 36.
				if len(pay)*8 >= 41 {
					parTireur[uint64(readBitsAt(pay, 36, 5))>>1]++
				}
			case 0xC0: // damage_aftermath (type 0)
				br := NewBitReader(pay)
				br.Skip(2)
				if br.ReadBits(7) == 0 {
					degats++
				}
			}
		}
	}
	t.Logf("== tirs (action_weapon_fire) : %d · evenements de degat (damage_aftermath) : %d ==", tirs, degats)
	t.Logf("  rapport degats/tirs = %.1f %% (proxy BRUT : un tir peut causer plusieurs degats et un degat plusieurs ticks — PAS la precision)",
		lot1Pct(degats, tirs))
	t.Logf("TIRS PAR TIREUR (index fire_events 0..15) : %d tireurs distincts", len(parTireur))
	// Repartition des tirs par tireur (top).
	t.Logf("  distribution : %s", lot1TopU64(parTireur, 12))
	t.Logf("ETAPE MISS a fermer : miss = action_weapon_fire a 0 cible ; le compte de cibles devient")
	t.Logf("  fiable une fois le cadrage complet du type 36 ferme (agent Ghidra en cours).")
}
