package filmdec

// equipment_changes_test.go — les deux règles de lecture du canal d'équipement : le témoin de
// complétude porté par le compteur de rotation, et le classement d'une émission.
//
// Ces règles ne sont couvertes par AUCUN autre test : le balayage lui-même a besoin d'un film,
// que le dépôt ne versionne pas. Ce qui se teste sans film, c'est ce qui décide.

import "testing"

func TestCompteurDeRotationDenonceLesEmissionsManquees(t *testing.T) {
	cases := []struct {
		nom              string
		from, to         uint32
		repeats          int
		jumps, estimated int
	}{
		{"pas de 1 : rien de manque", 5, 6, 0, 0, 0},
		{"repli du modulo 8", 7, 0, 0, 0, 0},
		{"saut de 2 : une emission manquee", 5, 7, 0, 1, 1},
		{"saut par-dessus le repli", 7, 1, 0, 1, 1},
		{"saut de 4 : trois manquees", 1, 5, 0, 1, 3},
		{"compteur immobile : contredit le canal", 5, 5, 1, 0, 0},
	}
	for _, c := range cases {
		t.Run(c.nom, func(t *testing.T) {
			var st EquipmentChangeStats
			countEquipmentCounterStep(&st, c.from, c.to)
			if st.Repeats != c.repeats || st.CounterJumps != c.jumps ||
				st.MissedEstimate != c.estimated {
				t.Errorf("de c%d a c%d : repetitions=%d sauts=%d manquees=%d ; "+
					"attendu %d/%d/%d", c.from, c.to,
					st.Repeats, st.CounterJumps, st.MissedEstimate,
					c.repeats, c.jumps, c.estimated)
			}
		})
	}
}

func TestClassementDUneEmissionDEquipement(t *testing.T) {
	const birth = uint64(10_000_000)
	born := func(uint32) (uint64, bool) { return birth, true }

	t.Run("porte ouverte = consommation", func(t *testing.T) {
		ch := EquipmentChange{TimestampUS: birth + 30_000_000, Rank: AbilitySetNoRank}
		if got := classifyEquipmentChange(ch, true, born); got != EquipmentSpent {
			t.Errorf("nature = %q, attendu %q : un emplacement qui se vide est une "+
				"consommation — la mesure exclut la mort", got, EquipmentSpent)
		}
	})
	t.Run("emission suivante = ramassage", func(t *testing.T) {
		ch := EquipmentChange{TimestampUS: birth, Rank: 6}
		if got := classifyEquipmentChange(ch, true, born); got != EquipmentTaken {
			t.Errorf("nature = %q, attendu %q : une vie qui a deja emis ne peut plus "+
				"reapparaitre", got, EquipmentTaken)
		}
	})
	t.Run("premiere emission a la naissance = reapparition", func(t *testing.T) {
		ch := EquipmentChange{TimestampUS: birth, Rank: 4}
		if got := classifyEquipmentChange(ch, false, born); got != EquipmentSpawned {
			t.Errorf("nature = %q, attendu %q", got, EquipmentSpawned)
		}
	})
	t.Run("premiere emission tardive = ramassage", func(t *testing.T) {
		ch := EquipmentChange{TimestampUS: birth + equipmentSpawnWindowUS + 1, Rank: 4}
		if got := classifyEquipmentChange(ch, false, born); got != EquipmentTaken {
			t.Errorf("nature = %q, attendu %q : hors de la fenetre de naissance, le joueur "+
				"est alle CHERCHER cet equipement", got, EquipmentTaken)
		}
	})
	t.Run("sans temoin de naissance, la premiere emission est un ramassage", func(t *testing.T) {
		ch := EquipmentChange{TimestampUS: birth, Rank: 4}
		if got := classifyEquipmentChange(ch, false, nil); got != EquipmentTaken {
			t.Errorf("nature = %q, attendu %q : sans temoin le balayage SURESTIME les "+
				"ramassages, et le contrat de ScanFilmEquipmentChanges le dit", got, EquipmentTaken)
		}
	})
}
