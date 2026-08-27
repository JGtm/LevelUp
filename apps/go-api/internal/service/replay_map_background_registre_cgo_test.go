//go:build cgo

package service

// replay_map_background_registre_cgo_test.go — LA RE-MESURE SUR LA BASE VIVANTE.
//
// Le garde-rail de couverture (replay_map_background_registre_test.go) travaille sur un
// inventaire GELÉ : c'est ce qui le rend jouable en CI, où aucune DuckDB n'existe. Il ne peut
// donc rien dire des matchs synchronisés DEPUIS le gel — et c'est précisément là qu'apparaît
// une nouvelle dérive d'identifiant d'asset. Ce test-ci referme la boucle : il relit le
// registre partagé et dit CE QUI A BOUGÉ.
//
// COMMENT LE LANCER (serveur local ARRÊTÉ — un second process ne peut pas ouvrir une DuckDB
// qu'un autre tient en écriture) :
//
//	FOND_REGISTRE=data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb \
//	  go test ./internal/service/ -run TestRegistreVivant -v
//
// Il ÉCHOUE dans deux cas, et rend dans les deux le texte à copier :
//   - une carte jouée absente de l'inventaire gelé -> la ligne JSON à ajouter au testdata ;
//   - une carte jouée sans fond et hors allowlist -> son map_id, ses noms, son poids.

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/testutil"
)

// fondRegistreEnv est la garde : sans elle, le test se saute proprement.
const fondRegistreEnv = "FOND_REGISTRE"

// litCartesJouees relit le registre partagé et rend l'inventaire réel.
//
// La requête est celle qui a produit le testdata gelé — la garder ici et là serait deux
// vérités : c'est CE fichier qui la porte, le testdata la cite dans son champ `source`.
func litCartesJouees(t *testing.T, db *sql.DB) []carteJouee {
	t.Helper()
	rows, err := db.QueryContext(t.Context(), `
		SELECT map_id,
		       list(DISTINCT map_name ORDER BY map_name)
		         FILTER (WHERE map_name IS NOT NULL AND map_name <> '' AND map_name <> map_id),
		       count(*),
		       strftime(max(start_time), '%Y-%m-%d')
		FROM match_registry
		GROUP BY map_id
		ORDER BY count(*) DESC, map_id`)
	if err != nil {
		t.Fatalf("lecture de match_registry : %v", err)
	}
	defer rows.Close()

	var out []carteJouee
	for rows.Next() {
		var c carteJouee
		var noms any
		var dernier sql.NullString
		if err := rows.Scan(&c.MapID, &noms, &c.Matchs, &dernier); err != nil {
			t.Fatalf("scan d'une ligne de match_registry : %v", err)
		}
		c.Noms = nomsDeListe(noms)
		c.Dernier = dernier.String
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("parcours de match_registry : %v", err)
	}
	if len(out) == 0 {
		t.Fatal("aucune carte dans match_registry — la base pointée n'est pas le registre partagé")
	}
	return out
}

// nomsDeListe convertit la colonne LIST du driver en tranche de chaînes, quelle que soit la
// forme rendue ([]any d'interfaces ou []string).
func nomsDeListe(v any) []string {
	switch l := v.(type) {
	case nil:
		return nil
	case []string:
		return l
	case []any:
		out := make([]string, 0, len(l))
		for _, e := range l {
			if s, ok := e.(string); ok && s != "" {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

// TestRegistreVivantCouvertureDesFonds relit le registre et confronte l'inventaire gelé + la
// résolution réelle à l'état du jour.
func TestRegistreVivantCouvertureDesFonds(t *testing.T) {
	chemin := strings.TrimSpace(os.Getenv(fondRegistreEnv))
	if chemin == "" {
		t.Skipf("%s non défini — re-mesure du registre non demandée", fondRegistreEnv)
	}
	root, err := testutil.RepoRoot()
	if err != nil {
		t.Fatalf("racine du dépôt introuvable : %v", err)
	}
	db, release, err := duckdb.OpenReadForQuery(chemin)
	if err != nil {
		// Cause de loin la plus fréquente : le serveur local tient la base en écriture.
		t.Skipf("registre %s non ouvrable en lecture (serveur local en cours ?) : %v", chemin, err)
	}
	defer release()

	cartes := litCartesJouees(t, db)
	inv := chargeInventaireRegistre(t)
	geles := make(map[string]bool, len(inv.Cartes))
	for _, c := range inv.Cartes {
		geles[c.MapID] = true
	}

	var nouvelles, orphelines []string
	matchsTotal, resolues := 0, 0
	for _, c := range cartes {
		matchsTotal += c.Matchs
		if !geles[c.MapID] {
			ligne, mErr := json.Marshal(c)
			if mErr != nil {
				t.Fatalf("sérialisation de %s : %v", c.MapID, mErr)
			}
			nouvelles = append(nouvelles, string(ligne))
		}
		cle, errRes := resoutFondDeCarte(t, root, c)
		if errRes == nil && cle != "" {
			resolues++
			continue
		}
		if !errors.Is(errRes, port.ErrMapBackgroundNotAvailable) {
			t.Errorf("%s (%v) : erreur inattendue : %v", c.MapID, c.Noms, errRes)
			continue
		}
		if _, admis := fondsManquantsAdmis[c.MapID]; admis {
			continue
		}
		orphelines = append(orphelines, fmt.Sprintf("%s  %-28s %4d matchs  dernier %s",
			c.MapID, strings.Join(c.Noms, " / "), c.Matchs, c.Dernier))
	}

	if len(nouvelles) > 0 {
		sort.Strings(nouvelles)
		t.Errorf("cartes JOUÉES absentes de l'inventaire gelé (%s) — %d entrées à ajouter, "+
			"telles quelles :\n    %s",
			inventaireRegistreFichier, len(nouvelles), strings.Join(nouvelles, ",\n    "))
	}
	if len(orphelines) > 0 {
		sort.Strings(orphelines)
		t.Errorf("cartes JOUÉES sans fond résolvable et hors allowlist (%d).\n"+
			"VÉRIFIER D'ABORD LA DÉRIVE D'IDENTIFIANT : si un fond porte déjà ce nom de carte, "+
			"c'est son identité qui manque dans `mapNames`.\n  %s",
			len(orphelines), strings.Join(orphelines, "\n  "))
	}
	t.Logf("registre vivant : %d cartes jouées, %d matchs ; %d cartes ont un fond",
		len(cartes), matchsTotal, resolues)
}
