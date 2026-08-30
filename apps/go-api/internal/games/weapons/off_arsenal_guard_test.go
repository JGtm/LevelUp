//go:build cgo

// off_arsenal_guard_test.go — LE garde-rail anti-double-comptage du lot « kills hors
// arme a feu » (2026-08-29).
//
// CE QU'IL PROTEGE. Deux voies independantes comptent les kills : l'attribution
// arme-a-feu (weapon_kills, qui part d'un `weapon_id` numerique) et la source de degat du
// film (match_kill_events, qui part d'un tag `jpt!`). Si une meme arme etait visible des
// deux, ses kills compteraient DEUX FOIS et l'invariant du sunburst (somme des classes ==
// total) sauterait.
//
// Le filtre qui l'interdit vit dans platform/duckdb/weapon_resolver.go
// (resolveOffArsenalKeys) et il est STRUCTUREL : ne remonter par la seconde voie que les
// cles SANS identifiant numerique. Ce test verrouille la propriete qui rend ce filtre
// correct — l'ensemble des cles Halo Infinite sans id numerique est EXACTEMENT les six
// entrees hors arsenal, ni plus, ni moins.
//
// SI CE TEST DEVIENT ROUGE : quelqu'un a ajoute au registre une arme Halo Infinite sans
// lui donner d'id filmshell. Soit c'est un oubli (donner l'id), soit c'est une nouvelle
// source hors arsenal (l'ajouter a la liste CI-DESSOUS, en connaissance de cause). Ne
// JAMAIS elargir la liste sans se demander si l'arme est deja comptee par weapon_kills.
package weapons

import (
	"sort"
	"testing"
)

// horsArsenalHINF : les six cles Halo Infinite qui ne se resolvent PAS par weapon_id,
// mais par la source de degat du film (pont killicon). Cf. registry.go.
var horsArsenalHINF = []string{
	"hinf_coil_hardlight",
	"hinf_coil_kinetic",
	"hinf_coil_plasma",
	"hinf_coil_shock",
	"hinf_environment",
	"hinf_repulsor",
}

func TestHorsArsenalHINFSansIdNumerique(t *testing.T) {
	db := openWeaponRegistryDB(t)

	rows, err := db.Query(`
		SELECT w.weapon_key
		FROM weapons w
		LEFT JOIN weapon_ids wi
		  ON wi.title_slug = w.title_slug AND wi.weapon_key = w.weapon_key
		WHERE w.title_slug = 'halo_infinite' AND wi.weapon_key IS NULL
		ORDER BY w.weapon_key`)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	defer rows.Close()

	var got []string
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			t.Fatalf("scan: %v", err)
		}
		got = append(got, k)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows: %v", err)
	}

	want := append([]string(nil), horsArsenalHINF...)
	sort.Strings(want)
	if len(got) != len(want) {
		t.Fatalf("cles HINF sans id numerique = %v, want %v — lire l'en-tete de ce fichier "+
			"AVANT de corriger la liste", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("cle sans id numerique [%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

// TestHorsArsenalHINFClassesAttendues : ces six cles portent bien les DEUX classes du lot
// (equipment pour le repulseur, environmental pour les cinq autres). Sans ca, le sunburst
// les rangerait dans une classe d'arme et le decoupage du residu serait faux.
func TestHorsArsenalHINFClassesAttendues(t *testing.T) {
	db := openWeaponRegistryDB(t)
	for key, wantClass := range map[string]string{
		"hinf_repulsor":       "equipment",
		"hinf_coil_kinetic":   "environmental",
		"hinf_coil_plasma":    "environmental",
		"hinf_coil_shock":     "environmental",
		"hinf_coil_hardlight": "environmental",
		"hinf_environment":    "environmental",
	} {
		var class string
		err := db.QueryRow(
			"SELECT class FROM weapons WHERE title_slug = 'halo_infinite' AND weapon_key = ?", key,
		).Scan(&class)
		if err != nil {
			t.Errorf("%s: %v", key, err)
			continue
		}
		if class != wantClass {
			t.Errorf("%s: class = %q, want %q", key, class, wantClass)
		}
	}
}
