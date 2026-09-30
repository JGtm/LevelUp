package main

// backfill_aide_test.go — l'aide de `levelup` (revue L7.5, RV1 et RV2, 2026-10-01).
//
// RV2 : l'insertion de `backfill-pad-tiers` puis de `backfill-vehicle-takes` au milieu de
// l'entree de `backfill-flag-grabs-net` avait laisse les trois lignes de celle-ci sous la
// derniere : `backfill-vehicle-takes` se lisait comme projetant les prises de drapeau.
// RV1 : la ligne `--dry-run` de `backfill-vehicle-takes` collait ses deux raisons.

import (
	"io"
	"os"
	"strings"
	"testing"

	"levelup/go-api/internal/persist"
)

// aideCapturee rend ce que `printUsage` ecrit sur la sortie standard.
func aideCapturee(t *testing.T) string {
	t.Helper()
	ancienne := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe : %v", err)
	}
	os.Stdout = w
	defer func() { os.Stdout = ancienne }()
	// Lecture CONCURRENTE : le tube d un poste Windows ne tient que quelques ko, l aide en fait plus.
	lu := make(chan []byte, 1)
	go func() {
		out, _ := io.ReadAll(r)
		lu <- out
	}()
	printUsage()
	_ = w.Close()
	return string(<-lu)
}

// entreesBackfill decoupe l'aide en entrees `backfill-*` : la ligne de tete et ses lignes de
// continuation (indentation plus profonde que celle des noms de commande).
func entreesBackfill(aide string) map[string]string {
	out := map[string]string{}
	var courant string
	for _, ligne := range strings.Split(aide, "\n") {
		switch {
		case strings.HasPrefix(ligne, "  ") && !strings.HasPrefix(ligne, "   "):
			courant = ""
			if nom := strings.Fields(ligne); len(nom) > 0 && strings.HasPrefix(nom[0], "backfill-") {
				courant = nom[0]
				out[courant] = ligne
			}
		case courant != "" && strings.HasPrefix(ligne, "   "):
			out[courant] += "\n" + ligne
		default:
			courant = ""
		}
	}
	return out
}

func TestAideBackfillChaqueEntreePorteSonTexte(t *testing.T) {
	entrees := entreesBackfill(aideCapturee(t))
	// Chaque entree cite la table qu'elle projette dans son propre bloc.
	tables := map[string]string{
		"backfill-flag-grabs-net": "match_flag_grabs_net",
		"backfill-vehicle-takes":  "match_vehicle_takes",
		"backfill-bomb-stats":     "match_bomb_stats",
		"backfill-usage-summary":  "match_usage_players",
	}
	for nom, table := range tables {
		bloc, ok := entrees[nom]
		if !ok {
			t.Errorf("%s : absente de l'aide", nom)
			continue
		}
		if !strings.Contains(bloc, table) {
			t.Errorf("%s : son bloc ne cite pas %s :\n%s", nom, table, bloc)
		}
		// Aucun bloc ne cite la table d'une AUTRE commande (le defaut constate : les lignes de
		// backfill-flag-grabs-net sous backfill-vehicle-takes).
		for autre, tableAutre := range tables {
			if autre != nom && strings.Contains(bloc, tableAutre) {
				t.Errorf("%s cite la table de %s (%s) :\n%s", nom, autre, tableAutre, bloc)
			}
		}
	}
	// Chaque bloc se ferme sur la parenthese de ses options, pas au milieu d'une phrase.
	for _, nom := range []string{"backfill-flag-grabs-net", "backfill-vehicle-takes"} {
		lignes := strings.Split(entrees[nom], "\n")
		if dernier := strings.TrimSpace(lignes[len(lignes)-1]); !strings.HasSuffix(dernier, ")") {
			t.Errorf("%s : le bloc ne se ferme pas sur la liste des options : %q", nom, dernier)
		}
	}
}

func TestLigneDryRunVehiculesSepareLesRaisons(t *testing.T) {
	ligne := ligneDryRunVehicules("match-1", persist.VehicleTakesBatch{
		Reason: "schema_before_67", DocSchema: 61, FragsReason: "takes_not_measured",
	})
	if strings.Contains(ligne, "schema_before_67takes_not_measured") {
		t.Fatalf("raisons collees : %q", ligne)
	}
	for _, attendu := range []string{"raison=schema_before_67", "frags_raison=takes_not_measured"} {
		if !strings.Contains(ligne, attendu) {
			t.Errorf("la ligne ne contient pas %q : %q", attendu, ligne)
		}
	}
	// Une raison vide se lit `-`, pas un vide qui recolle.
	if vide := ligneDryRunVehicules("m", persist.VehicleTakesBatch{Measured: true}); !strings.Contains(vide, "raison=- frags_raison=-") {
		t.Errorf("raisons vides : %q", vide)
	}
}
