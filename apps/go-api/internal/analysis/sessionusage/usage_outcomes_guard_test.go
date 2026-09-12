package sessionusage

// usage_outcomes_guard_test.go — GARDE-RAIL DE LA BASCULE « UTILISÉ » (correction C1
// de la revue de la vague 5, 2026-09-10).
//
// # CE QU'IL EMPÊCHE
//
// [equipmentUsedOf] est la DEUXIÈME et dernière écriture tolérée de la bascule
// « utilisé » (règle CLAUDE.md n°6) : la première est `usageUsedOf`
// (internal/games/halo_infinite/film/replay/usage_summary_outcomes.go), qui décide de la même chose
// sur une ligne de PROJECTION quand celle-ci décide sur une ligne de BASE. Les deux
// ont divergé une fois, en silence, et pendant tout un lot : le résumé est passé aux
// CONSOMMATIONS en `us6` (2026-09-10) et l'agrégat de session est resté sur les
// POSES — la page Sessions affichait « utilisé 0 » là où la vue match affichait
// « utilisé 2 » pour le même match.
//
// Ce que la divergence rendait possible : une liste de familles RECOPIÉE ici. Le
// garde-rail interdit la recopie et exige l'accès partagé
// ([replay.UsageFamilySpawnsPiece], recollée au manifeste par le garde-rail du
// paquet replay), plus la branche de repli sur les consommations.
//
// # POURQUOI UN GREP ET PAS UN TEST DE COMPORTEMENT
//
// Les tests de comportement (usage_outcomes_test.go) épinglent le RÉSULTAT sur les
// familles que leurs fixtures nomment. Ils ne diraient rien d'une famille NOUVELLE
// du manifeste, ni d'une réécriture qui recopierait la liste : c'est le lien vers la
// source unique que ce fichier surveille, pas la valeur.

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

func TestBasculeUtiliseLitLaSourceUniqueDesFamilles(t *testing.T) {
	raw, err := os.ReadFile("usage_outcomes.go")
	if err != nil {
		t.Fatalf("lecture de usage_outcomes.go: %v", err)
	}
	src := string(raw)

	// 1. LA CONNAISSANCE VIENT DU PAQUET replay, jamais d'une liste réécrite.
	if !strings.Contains(src, "replay.UsageFamilySpawnsPiece(") {
		t.Error("la bascule « utilisé » n'appelle plus replay.UsageFamilySpawnsPiece : " +
			"la liste des familles à pièce engendrée est recollée au manifeste dans le paquet " +
			"replay, une seconde écriture ici re-divergerait au premier objet `kind = deployed` ajouté")
	}

	// 2. LE REPLI EST LES CONSOMMATIONS. Sans cette branche, un capteur pris puis
	// consommé retombe à « utilisé 0 » — le défaut exact du constat C1.
	if !strings.Contains(src, "p.SpentByFamily[family]") {
		t.Error("la bascule « utilisé » ne lit plus p.SpentByFamily : tout déployable qui " +
			"n'engendre pas de pièce se lit sur ses CHARGES CONSOMMÉES (rapport E0 du " +
			"2026-09-10, question 5), jamais sur ses poses")
	}

	// 3. AUCUNE FAMILLE EN DUR dans la décision : le seul vocabulaire admis ici est
	// celui des constantes exportées du paquet replay (les deux bonus). Un littéral
	// "wall" / "sensor" / ... signerait la liste recopiée que le point 1 interdit.
	//
	// Les commentaires sont retirés avant l'examen : ils CITENT ces familles, et
	// c'est leur rôle (la règle s'explique en nommant le mur et le capteur).
	interdits := regexp.MustCompile(`"(wall|sensor|shroud_screen|threat_seeker|repair_field|translocator_beacon)"`)
	if m := interdits.FindString(sansCommentaires(src)); m != "" {
		t.Errorf("famille %s écrite en dur dans usage_outcomes.go — le périmètre du bilan vient "+
			"de replay.EquipmentOutcomeFamilies et la bascule de replay.UsageFamilySpawnsPiece", m)
	}
}

// sansCommentaires retire les commentaires de ligne d'une source Go. Suffisant ici :
// le fichier surveillé n'a aucun commentaire de bloc ni aucune chaîne portant `//`.
func sansCommentaires(src string) string {
	var b strings.Builder
	for _, ligne := range strings.Split(src, "\n") {
		if i := strings.Index(ligne, "//"); i >= 0 {
			ligne = ligne[:i]
		}
		b.WriteString(ligne)
		b.WriteString("\n")
	}
	return b.String()
}
