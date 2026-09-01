//go:build cgo

// off_arsenal_guard_test.go — LE garde-rail anti-double-comptage du lot « kills hors
// arme a feu » (2026-08-29).
//
// CE QU'IL PROTEGEAIT, ET CE QU'IL PROTEGE DEPUIS LE 2026-09-01.
//
// A l'origine, DEUX voies independantes comptaient les kills : l'attribution arme-a-feu
// (`weapon_kills`, qui part d'un `weapon_id` numerique) et la source de degat du film
// (`match_kill_events`, qui part d'un tag `jpt!`). Une meme arme visible des deux aurait
// compte DEUX FOIS, et l'invariant du sunburst (somme des classes == total) aurait saute.
// Le filtre qui l'interdisait vivait dans `resolveOffArsenalKeys` : ne remonter par la
// seconde voie que les cles SANS identifiant numerique.
//
// LES DEUX VOIES ONT FUSIONNE (decision D11) et `weapon_kills` a disparu du fichier Halo
// Infinite : il n'y a plus qu'une source, donc plus rien a departager, et ce filtre a ete
// supprime avec sa fonction. CE TEST, LUI, RESTE — il verrouille desormais la FORME du
// registre, qui est ce qui rendait le filtre correct et qui reste vraie : l'ensemble des
// cles Halo Infinite sans identifiant numerique est EXACTEMENT la liste ci-dessous, ni
// plus, ni moins. C'est aussi ce qui garantirait le non-double-comptage si une seconde
// voie revenait un jour.
//
// SI CE TEST DEVIENT ROUGE : quelqu'un a ajoute au registre une arme Halo Infinite sans
// lui donner d'id filmshell. Soit c'est un oubli (donner l'id), soit c'est une nouvelle
// source hors arsenal — objet, chassis, tourelle — qui se resout par le pont `killicon`
// et n'a donc pas d'id : l'ajouter a la liste CI-DESSOUS, en connaissance de cause.
package weapons

import (
	"sort"
	"testing"
)

// horsArsenalHINF : les cles Halo Infinite qui ne se resolvent PAS par weapon_id, mais par
// la source de degat du film (pont killicon). Cf. registry.go.
//
// ELARGIE DE SIX A VINGT LE 2026-09-01 (etape A6) : les 14 chassis et tourelles rejoignent
// les 6 entrees d origine. LA JUSTIFICATION EST LA MEME, et elle est structurelle — aucune
// de ces sources n emet de record de degat `0xd2`, donc aucune n a d identifiant filmshell,
// donc aucune n est visible de `weapon_kills`. Le non-double-comptage n est pas une regle
// qu on applique, c est une propriete qu on constate.
//
// ⚠ L EPEE ET LE MARTEAU N Y SONT PAS, ET C EST NORMAL : reclasses `heavy` le meme jour,
// ils gardent leurs SEPT identifiants numeriques (variantes et skins) — ils se resolvent
// par les deux voies, comme toute arme de l arsenal.
var horsArsenalHINF = []string{
	"hinf_banshee",
	"hinf_chopper",
	"hinf_coil_hardlight",
	"hinf_coil_kinetic",
	"hinf_coil_plasma",
	"hinf_coil_shock",
	"hinf_environment",
	"hinf_falcon_gl",
	"hinf_falcon_lmg",
	"hinf_ghost",
	"hinf_pelican",
	"hinf_phantom",
	"hinf_repulsor",
	"hinf_rockethog",
	"hinf_scorpion",
	"hinf_turret_machinegun",
	"hinf_turret_plasma",
	"hinf_turret_shade",
	"hinf_wasp",
	"hinf_wraith",
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

// TestHorsArsenalHINFClassesAttendues : ces cles portent bien les QUATRE classes du lot
// (equipment pour le repulseur, environmental pour la chute et les bobines, vehicle pour
// les chassis, turret pour les tourelles). Sans ca, le sunburst les rangerait dans une
// classe d'arme et le decoupage du residu serait faux.
//
// L EPEE ET LE MARTEAU sont verifies ici AUSSI, en `heavy` : ils ne sont pas hors arsenal
// (ils ont des identifiants numeriques) mais leur classe est le point exact de la decision
// du 2026-09-01 — les remettre en `melee` les ferait ecarter par le lecteur, et leurs
// 9 741 frags retomberaient dans « Non attribue » sans qu aucun compteur API ne les serve.
func TestHorsArsenalHINFClassesAttendues(t *testing.T) {
	db := openWeaponRegistryDB(t)
	for key, wantClass := range map[string]string{
		"hinf_repulsor":          "equipment",
		"hinf_coil_kinetic":      "environmental",
		"hinf_coil_plasma":       "environmental",
		"hinf_coil_shock":        "environmental",
		"hinf_coil_hardlight":    "environmental",
		"hinf_environment":       "environmental",
		"hinf_ghost":             "vehicle",
		"hinf_banshee":           "vehicle",
		"hinf_wraith":            "vehicle",
		"hinf_phantom":           "vehicle",
		"hinf_chopper":           "vehicle",
		"hinf_wasp":              "vehicle",
		"hinf_scorpion":          "vehicle",
		"hinf_rockethog":         "vehicle",
		"hinf_pelican":           "vehicle",
		"hinf_falcon_lmg":        "turret",
		"hinf_falcon_gl":         "turret",
		"hinf_turret_machinegun": "turret",
		"hinf_turret_plasma":     "turret",
		"hinf_turret_shade":      "turret",
		"hinf_energy_sword":      "heavy",
		"hinf_gravity_hammer":    "heavy",
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
