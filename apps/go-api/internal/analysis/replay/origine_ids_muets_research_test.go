package replay

// origine_ids_muets_research_test.go — ADDENDUM : NOMMER LES DEUX IDENTIFIANTS DE CLASSE ARME
// QUE LE CATALOGUE NE NOMME PAS.
//
// LES DEUX SUSPECTS. `00007ca9` et `e9e7ff79` sont portés par des ramassages natifs de classe
// ARME (0 ou 1) mais ne figurent dans aucune table de libellés. `00007ca9` est le tout premier
// identifiant que le chantier a décodé à la main, sur le premier paquet du film de référence ;
// sa forme est ATYPIQUE — valeur basse (31 913), pas l'allure d'un hash de tag comme les
// autres (2 à 4 milliards).
//
// LA MÉTHODE, EMPIRIQUE ET BON MARCHÉ : pour chaque ramassage portant l'un de ces identifiants,
// regarder ce que le RAMASSEUR TIENT JUSTE APRÈS. Le canal i43..i46 publie l'identité de l'arme
// par emplacement à chaque changement ; si une arme apparaît en main du bon slot dans la
// foulée, elle nomme l'identifiant.
//
// SEUILS ÉCRITS AVANT LA MESURE :
//
//	M1 — une émission i43..i46 du MÊME slot dans les 2 s qui suivent le ramassage nomme
//	     l'identifiant. Si une seule famille sort sur toutes les occurrences, l'identifiant est
//	     NOMMÉ ; si plusieurs sortent, on publie la distribution sans trancher.
//	M2 — TÉMOIN : la même recherche sur des instants décalés. Elle doit rendre nettement moins,
//	     sinon on ne mesure que la densité des changements d'arme.
//	M3 — un identifiant sans AUCUNE émission dans la fenêtre reste MUET, et c'est un résultat :
//	     il désigne alors probablement un objet qui n'occupe pas un emplacement d'arme.
//
// Garde ORIGINE_FILM (le répertoire d'UN film).

import (
	"fmt"
	"os"
	"sort"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

// oriIdsMuets : les deux identifiants à élucider, en hexadécimal minuscule (convention de
// `Pickup.W` et `WeaponChange.W`).
var oriIdsMuets = map[uint32]string{0x00007ca9: "00007ca9", 0xe9e7ff79: "e9e7ff79"}

// oriSuiteUS est la fenêtre « juste après » : deux secondes.
const oriSuiteUS = 2_000_000

// oriTenuApres rend les familles d'arme que le slot reçoit en main dans la fenêtre qui suit
// `at` (décalée de decalUS pour le témoin).
func oriTenuApres(chg []filmdec.HeldWeaponChange, slot uint32, at uint64, decalUS int64) []uint32 {
	var out []uint32
	for _, c := range chg {
		if c.Slot != slot || c.Family == filmdec.NoWeaponVariant {
			continue
		}
		d := int64(c.TimestampUS) + decalUS - int64(at)
		if d >= 0 && d <= oriSuiteUS {
			out = append(out, c.Family)
		}
	}
	return out
}

func TestOrigineIdentifiantsMuets(t *testing.T) {
	dir := os.Getenv("ORIGINE_FILM")
	if dir == "" {
		t.Skip("ORIGINE_FILM absent : instrument de mesure sauté")
	}
	release := filmdec.LockProcessDecode()
	defer release()

	pickups, _, err := filmdec.ScanFilmBipedPickups(dir)
	if err != nil {
		t.Fatalf("ramassages natifs illisibles : %v", err)
	}
	kf, err := filmdec.ScanFilmKeyframeLoadouts(dir, loadoutFamilies())
	if err != nil {
		t.Fatalf("images-clés illisibles : %v", err)
	}
	chg, _, err := filmdec.ScanFilmHeldWeaponChanges(dir, spawnSetFrom(kf))
	if err != nil {
		t.Fatalf("changements d'arme illisibles : %v", err)
	}
	t.Logf("== ADDENDUM — LES DEUX IDENTIFIANTS MUETS · %s ==", dir)
	t.Logf("ramassages natifs : %d · émissions i43..i46 : %d", len(pickups), len(chg))

	// Toutes les familles vues par i43..i46 : elles disent ce que le catalogue NOMME.
	connues := map[uint32]int{}
	for _, c := range chg {
		if c.Family != filmdec.NoWeaponVariant {
			connues[c.Family]++
		}
	}
	for id, hex := range oriIdsMuets {
		var occ int
		suites := map[uint32]int{}
		temoin := 0
		classes := map[uint8]int{}
		for _, p := range pickups {
			if p.CatalogID != id {
				continue
			}
			occ++
			classes[p.Class]++
			for _, f := range oriTenuApres(chg, p.Slot, p.TimestampUS, 0) {
				suites[f]++
			}
			for _, d := range []int64{37_000_000, -53_000_000, 91_000_000} {
				if len(oriTenuApres(chg, p.Slot, p.TimestampUS, d)) > 0 {
					temoin++
					break
				}
			}
		}
		if occ == 0 {
			t.Logf("  %s : ABSENT de ce film", hex)
			continue
		}
		keys := make([]uint32, 0, len(suites))
		for k := range suites {
			keys = append(keys, k)
		}
		sort.Slice(keys, func(i, j int) bool { return suites[keys[i]] > suites[keys[j]] })
		s := ""
		for i, k := range keys {
			if i >= 5 {
				break
			}
			if i > 0 {
				s += " · "
			}
			s += fmt.Sprintf("%08x x%d", k, suites[k])
		}
		if s == "" {
			s = "AUCUNE — le ramasseur ne reçoit RIEN en main dans les 2 s"
		}
		t.Logf("  %s : %d occurrence(s), classes %v", hex, occ, classes)
		t.Logf("      armes reçues en main dans les 2 s : %s", s)
		t.Logf("      TÉMOIN (instants décalés) : %d / %d occurrences ont une émission", temoin, occ)
		// L'identifiant est-il lui-même une famille que i43..i46 publie ailleurs ?
		t.Logf("      cet identifiant est-il une famille vue par i43..i46 ? %v (%d émissions)",
			connues[id] > 0, connues[id])
	}
}
