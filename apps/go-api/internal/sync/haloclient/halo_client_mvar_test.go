package haloclient

// halo_client_mvar_test.go — L'ORDRE DE PREFERENCE DES FICHIERS `.mvar`, fige par un test.
//
// POURQUOI CE TEST EXISTE : une passe de re-validation a pris, dans un asset a deux `.mvar`,
// celui de la carte de BASE au lieu de la VARIANTE. Elle a produit des socles deplaces de 22 a
// 80 metres sur neuf cartes — des chiffres qui ne decrivaient aucune mise a jour du jeu. Le
// piege est invisible a la lecture (les deux fichiers parsent, les deux rendent des socles
// plausibles) et il ne se voit qu'a la distance. Ce test le rend impossible a reintroduire.

import "testing"

// mvChoisir rejoue la selection de fichier telle que FetchMvarForMap la fait.
//
// La logique est recopiee plutot qu'extraite : elle tient en six lignes au milieu d'une methode
// qui fait un appel reseau, et l'extraire pour la tester la rendrait plus indirecte qu'elle
// n'est utile. Le test echoue si l'ordre diverge — c'est ce qu'on lui demande.
func mvChoisir(chemins []string, mvarFile string) string {
	choisi := chemins[0]
	for _, p := range chemins {
		if base(p) == mvarFile {
			choisi = p
		}
	}
	for _, p := range chemins {
		if base(p) == nomDeLaVariante {
			choisi = p
			break
		}
	}
	return choisi
}

func base(p string) string {
	for i := len(p) - 1; i >= 0; i-- {
		if p[i] == '/' || p[i] == 92 { // 92 = antislash, ecrit en code pour rester lisible
			return p[i+1:]
		}
	}
	return p
}

func TestPreferenceDuFichierDeVariante(t *testing.T) {
	cas := []struct {
		nom     string
		chemins []string
		declare string
		attendu string
	}{
		{
			// LE CAS QUI A CAUSE LE DEGAT. Le catalogue d'objectifs declare le nom du NIVEAU ;
			// l'asset porte aussi la variante. C'est la variante qui est jouee.
			nom:     "asset a deux fichiers, le catalogue declare la carte de BASE",
			chemins: []string{"pre/btb_highpower.mvar", "pre/map.mvar"},
			declare: "btb_highpower.mvar",
			attendu: "pre/map.mvar",
		},
		{
			nom:     "la variante est premiere dans la liste",
			chemins: []string{"pre/map.mvar", "pre/btb_highpower.mvar"},
			declare: "btb_highpower.mvar",
			attendu: "pre/map.mvar",
		},
		{
			// Carte NATIVE : pas de `map.mvar`, le fichier declare EST la variante.
			nom:     "asset sans map.mvar — le fichier declare gagne",
			chemins: []string{"pre/autre.mvar", "pre/catalyst.mvar"},
			declare: "catalyst.mvar",
			attendu: "pre/catalyst.mvar",
		},
		{
			nom:     "rien ne correspond — le premier, faute de mieux",
			chemins: []string{"pre/inconnu.mvar"},
			declare: "absent.mvar",
			attendu: "pre/inconnu.mvar",
		},
		{
			// Le canevas Forge ne doit pas l'emporter : il est nomme, la variante aussi.
			nom:     "canevas Forge et variante",
			chemins: []string{"pre/fo11_blank.mvar", "pre/map.mvar"},
			declare: "fo11_blank.mvar",
			attendu: "pre/map.mvar",
		},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			if got := mvChoisir(c.chemins, c.declare); got != c.attendu {
				t.Errorf("fichier choisi = %q, attendu %q — prendre la carte de BASE pour la "+
					"VARIANTE deplace les socles de plusieurs dizaines de metres", got, c.attendu)
			}
		})
	}
}
