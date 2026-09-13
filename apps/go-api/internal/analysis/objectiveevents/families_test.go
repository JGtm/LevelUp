package objectiveevents

import "testing"

// families_test.go — la frontiere entre une statistique D'OBJECTIF et une statistique que les
// tables portent pour une autre raison (ancre d'identite, controle croise).

// TestStatsNommeesPortentLeurFamille est LE GARDE-RAIL de la liste blanche de families.go :
// toute statistique d'une table de `namedStatSlots` porte sa famille en prefixe, SAUF les deux
// qui ne sont pas des objectifs et qui sont nommees ici, une par une.
//
// MUTATION : nommer un emplacement `flagSecures` (sans souligne) ou ajouter une famille dans
// `namedStatSlots` sans l'ajouter a `objectiveFamilies` -> rouge. C'est exactement le jour ou
// le filtre du calque deviendrait silencieusement faux.
func TestStatsNommeesPortentLeurFamille(t *testing.T) {
	horsObjectif := map[string]bool{StatKills: true, StatAssists: true}
	for objectiveType, table := range namedStatSlots {
		for key, slot := range table {
			if horsObjectif[slot.Stat] {
				if IsObjectiveFamilyStat(slot.Stat) {
					t.Errorf("%s %v : `%s` est comptee comme un objectif", objectiveType, key, slot.Stat)
				}
				continue
			}
			if !IsObjectiveFamilyStat(slot.Stat) {
				t.Errorf("%s %v : `%s` ne porte aucune famille d'objectif connue — ajouter la "+
					"famille a objectiveFamilies, ou la statistique a horsObjectif",
					objectiveType, key, slot.Stat)
			}
		}
		// La famille du mode elle-meme doit etre connue : une table dont le type d'objectif
		// n'est pas dans la liste blanche ne produirait que des actions « hors objectif ».
		if !IsObjectiveFamilyStat(objectiveType + "_x") {
			t.Errorf("famille `%s` absente d'objectiveFamilies", objectiveType)
		}
	}
}

// TestIsObjectiveFamilyStatExigeLeSeparateur — le prefixe est une FAMILLE, pas une suite de
// lettres : sans le souligne, un nom voisin n'en est pas.
func TestIsObjectiveFamilyStatExigeLeSeparateur(t *testing.T) {
	for _, stat := range []string{"flagrant", "zones", "vip", "", "deaths", StatKills, StatAssists} {
		if IsObjectiveFamilyStat(stat) {
			t.Errorf("`%s` ne devrait pas etre une statistique d'objectif", stat)
		}
	}
	for _, stat := range []string{StatFlagGrabs, StatZoneSecures, StatVipSelected,
		StatBombDetonations, "skull_grabs", ObjectiveTypeHill + "_x"} {
		if !IsObjectiveFamilyStat(stat) {
			t.Errorf("`%s` devrait etre une statistique d'objectif", stat)
		}
	}
}

// TestCountObjectiveFamilyCompteLesDeuxFormes — le compteur sert le calque (IdentifiedEvent)
// et le pont amont (NamedEvent) par le meme code.
func TestCountObjectiveFamilyCompteLesDeuxFormes(t *testing.T) {
	named := []NamedEvent{
		{Stat: StatFlagGrabs}, {Stat: StatKills}, {Stat: StatAssists}, {Stat: StatZoneCaptures},
	}
	if n := CountObjectiveFamily(named); n != 2 {
		t.Errorf("nommes : %d, attendu 2", n)
	}
	identified := []IdentifiedEvent{
		{NamedEvent: NamedEvent{Stat: StatKills}, XUID: "a"},
		{NamedEvent: NamedEvent{Stat: StatBombDetonations}, XUID: "a"},
	}
	if n := CountObjectiveFamily(identified); n != 1 {
		t.Errorf("identifies : %d, attendu 1", n)
	}
	if n := CountObjectiveFamily([]NamedEvent(nil)); n != 0 {
		t.Errorf("liste vide : %d, attendu 0", n)
	}
}
