package killcollector

// category_headshot_test.go — LE VERROU D'ÉGALITÉ entre le décodeur et le comparateur partagé
// (G.1, 2026-08-30). Même montage que film_read_paths_test.go, pour la MÊME raison : la valeur
// a deux domiciles structurels —
//
//	games/halo_infinite/film/killsource   `Category` — l'énumération, chez le décodeur qui la
//	                                      produit. Paquet title-specific : ni `persist` ni
//	                                      `platform/duckdb` ne peuvent l'importer.
//	domain/killscope                      la CHAÎNE, dans une feuille sans import, lisible par
//	                                      le lecteur du kill feed (`platform/duckdb`).
//
// Ce paquet est le SEUL du dépôt qui importe les deux. CE QUE LA DÉRIVE COÛTERAIT : si
// `killscope.CategoryHeadshot` divergeait d'un caractère de `killsource.CategoryHeadshot.Name()`,
// le filtre de lecture cesserait de reconnaître AUCUNE ligne de tir à la tête — silencieusement,
// sans erreur ni compteur (le kill feed afficherait juste « pas de headshot » partout).

import (
	"testing"

	"levelup/go-api/internal/domain/killscope"
	"levelup/go-api/internal/games/halo_infinite/film/killsource"
)

func TestCategoryHeadshotEgaleAuDecodeur(t *testing.T) {
	if got, want := killscope.CategoryHeadshot, killsource.CategoryHeadshot.Name(); got != want {
		t.Errorf("killscope.CategoryHeadshot = %q, decodeur = %q — la lecture du tir a la tete "+
			"cesserait de reconnaitre TOUTE ligne, sans erreur ni compteur", got, want)
	}
	// Témoin négatif : la valeur EXCLUE par doctrine (G.0, oracle 84,4 % avec elle, 99,3 % sans)
	// doit rester DIFFÉRENTE. Un decodeur qui renommerait HeadshotMultiplier en "Headshot" ferait
	// silencieusement rentrer la population interdite par le filtre strict.
	if killscope.CategoryHeadshot == killsource.CategoryHeadshotMultiplier.Name() {
		t.Fatalf("killscope.CategoryHeadshot (%q) egale desormais killsource.CategoryHeadshotMultiplier.Name() "+
			"— le decodeur a change de nommage, et le filtre STRICT du rapport G.0 (99,3%% vs 84,4%% "+
			"d'accord oracle) laisserait rentrer la population interdite", killscope.CategoryHeadshot)
	}
}
