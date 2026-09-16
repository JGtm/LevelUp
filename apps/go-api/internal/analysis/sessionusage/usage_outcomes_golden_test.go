package sessionusage

// usage_outcomes_golden_test.go — LE GOLDEN DES FAMILLES -> ISSUE, figé AVANT le
// portage des quatre symboles d'usage hors du paquet de titre (item 2.5.f du
// PLAN_DECODEUR_FILM_2026-09-13, décision V15 (5)).
//
// # CE QU'IL FIGE, ET POURQUOI CE N'EST PAS COUVERT PAR LES TESTS EXISTANTS
//
// Les tests de comportement (usage_outcomes_test.go) épinglent le résultat sur les
// DEUX familles que leurs fixtures nomment (mur et capteur). Le portage, lui, change
// la MAISON de la table de reconnaissance ([equipmentBilanFamilies]) et de la bascule
// « utilisé » ([equipmentUsedOf]) : ce qui doit être prouvé inchangé est le
// classement de CHAQUE famille du bilan, y compris celles qu'aucune fixture ne cite.
//
// # LA LECTURE DU GOLDEN
//
// Une ligne par famille, DANS L'ORDRE de la table de reconnaissance — cet ordre est
// celui où la page cite les familles, il fait donc partie du contrat. Les quatre
// colonnes sont les issues rendues par [equipmentOutcomeOf] sur une ligne de base
// dont CHAQUE canal porte une valeur sentinelle distincte : la colonne « utilisé »
// nomme donc à elle seule le canal lu (11 = épisodes de camouflage, 22 = épisodes de
// surbouclier, 33 = poses `deployed`, 44 = charges consommées `spent`). Un portage
// qui recopierait la liste des familles à pièce engendrée, ou qui perdrait une
// famille, déplace une valeur de cette table.
//
// Ce fichier NE DOIT PAS BOUGER au commit de portage : c'est le gate de 2.5.f.

import (
	"fmt"
	"strings"
	"testing"
)

// goldenIssuesParFamille — l'état mesuré le 2026-09-17 sur la base 88f1a1115, AVANT
// le déplacement. Colonnes : famille, utilisé, gardé, lâché, pris.
const goldenIssuesParFamille = `
wall                 33 55 66 77
sensor               44 55 66 77
translocator_beacon  44 55 66 77
shroud_screen        44 55 66 77
threat_seeker        44 55 66 77
repair_field         44 55 66 77
powerup_camo         11 55 66 77
powerup_overshield   22 55 66 77
`

// ligneDeBaseSentinelle — une ligne joueur dont les six canaux d'issue portent des
// valeurs DISTINCTES, pour que la sortie nomme le canal lu et pas seulement un
// nombre. Les ventilations par famille sont remplies pour TOUTES les familles du
// bilan : la table de reconnaissance est le sujet du test, pas son entrée.
func ligneDeBaseSentinelle(familles []string) *PlayerRow {
	p := &PlayerRow{
		XUID:               "P",
		CamoEpisodes:       11,
		OvershieldEpisodes: 22,
		DeployedByFamily:   map[string]int{},
		SpentByFamily:      map[string]int{},
		KeptByFamily:       map[string]int{},
		DroppedByFamily:    map[string]int{},
		TakenByFamily:      map[string]int{},
	}
	for _, f := range familles {
		p.DeployedByFamily[f] = 33
		p.SpentByFamily[f] = 44
		p.KeptByFamily[f] = 55
		p.DroppedByFamily[f] = 66
		p.TakenByFamily[f] = 77
	}
	return p
}

func TestGoldenIssuesParFamilleNeBougePas(t *testing.T) {
	familles := equipmentBilanFamilies
	if len(familles) == 0 {
		t.Fatal("la table de reconnaissance du bilan est vide — le golden ne garderait rien")
	}
	p := ligneDeBaseSentinelle(familles)

	var b strings.Builder
	b.WriteString("\n")
	for _, f := range familles {
		c := equipmentOutcomeOf(p, f)
		fmt.Fprintf(&b, "%-20s %d %d %d %d\n", f, c.used, c.kept, c.dropped, c.taken)
	}

	if got, want := normaliseGolden(b.String()), normaliseGolden(goldenIssuesParFamille); got != want {
		t.Errorf("le classement des familles du bilan a changé.\n--- attendu ---\n%s\n--- obtenu ---\n%s",
			want, got)
	}
}

// normaliseGolden réduit chaque ligne à ses champs séparés par une espace : le
// golden reste lisible en colonnes sans que l'alignement en fasse partie.
func normaliseGolden(s string) string {
	var lignes []string
	for _, l := range strings.Split(s, "\n") {
		if champs := strings.Fields(l); len(champs) > 0 {
			lignes = append(lignes, strings.Join(champs, " "))
		}
	}
	return strings.Join(lignes, "\n")
}
