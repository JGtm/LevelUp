package replaylabels

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"levelup/go-api/internal/analysis/weaponv3"
	"levelup/go-api/internal/games/weapons"
)

// repoRoot remonte jusqu'au répertoire qui porte config/titles.
func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("répertoire courant : %v", err)
	}
	for i := 0; i < 8; i++ {
		if _, err := os.Stat(filepath.Join(dir, "config", "titles")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatal("racine du dépôt introuvable")
	return ""
}

func TestLoad_CatalogueReelDuTitre(t *testing.T) {
	cat, err := Load(repoRoot(t), "halo_infinite")
	if err != nil {
		t.Fatalf("chargement du catalogue : %v", err)
	}
	if cat.Empty() {
		t.Fatal("catalogue vide : les mappings du titre ne sont plus lus")
	}
	if len(cat.Grenades) != 4 {
		t.Errorf("%d rang(s) de grenade, attendu 4", len(cat.Grenades))
	}
	// Les deux langues sont obligatoires PARTOUT : un artefact est construit une fois et
	// servi aux deux locales.
	for i, g := range cat.Grenades {
		if g.En == "" || g.Fr == "" {
			t.Errorf("rang de grenade %d incomplet : %+v", i, g)
		}
	}
	// Les palettes portent leurs MARQUEURS avec leurs noms : sans marqueur, l'assemblage ne
	// pourrait classer aucun film, et la palette serait de la donnée morte.
	for _, p := range cat.Abilities {
		if p.ID == "" || len(p.Markers) == 0 {
			t.Errorf("palette de capacités sans identifiant ou sans marqueur : %+v", p)
		}
		for rank, a := range p.Ranks {
			if a.En == "" || a.Fr == "" {
				t.Errorf("capacité %d de %q incomplète : %+v", rank, p.ID, a)
			}
		}
	}
	for fam, w := range cat.Weapons {
		if w.En == "" || w.Fr == "" {
			t.Errorf("famille %08X : libellé incomplet %+v", fam, w)
		}
	}
	// Les ICÔNES EXTRAITES suivent le catalogue (fiches joueur du rejeu). Toutes les familles
	// n'en ont pas — le repli libellé est la règle — mais un catalogue réel sans AUCUNE icône
	// dirait que la jointure famille -> tag `weap` est cassée.
	if len(cat.Icons) == 0 {
		t.Fatal("aucune icône d'arme dans le catalogue : la jointure par tag weap ne rend plus rien")
	}
	for fam, ic := range cat.Icons {
		if ic.URL == "" {
			t.Errorf("famille %08X : icône à URL vide — une entrée sans visuel ne doit pas entrer", fam)
		}
		if _, nommee := cat.Weapons[fam]; !nommee {
			// Une icône sans libellé ne s'affiche pas (buildWeaponLabels n'entre que les armes
			// nommées) : signal d'un registre incomplet, pas une erreur de jointure.
			t.Logf("famille %08X : icône sans libellé (registre incomplet ?) : %s", fam, ic.URL)
		}
	}
}

// TestLoad_TitreSansMappings — un titre sans ces fichiers ÉCHOUE, il ne dégrade pas en
// silence. Un rejeu muet est indistinguable, à l'écran, d'un rejeu dont toutes les armes
// seraient inconnues : c'est à l'outil de build de refuser, pas au lecteur de deviner.
func TestLoad_TitreSansMappings(t *testing.T) {
	if _, err := Load(repoRoot(t), "halo_5"); err == nil {
		t.Error("un titre sans replay_labels.toml doit faire échouer le chargement")
	}
	if _, err := Load(t.TempDir(), "halo_infinite"); err == nil {
		t.Error("une racine sans config/titles doit faire échouer le chargement")
	}
}

// corpusObjetsEquipement — LES IDENTIFIANTS `eqip` OBSERVÉS DANS LES POSES DU CORPUS de films,
// avec l'effectif qui les a fait entrer. Recopiés de la mesure amont
// (`himap/sonde_ti37_gamefiles_test.go`, `ti37Observes` — la sonde qui les résout contre les
// modules du jeu) : c'est la SECONDE et dernière copie de cette liste, et elle existe pour que
// le manifeste ne puisse pas prendre du retard sur la mesure en silence.
//
// LE GARDE-RAIL EST BILATÉRAL. Un identifiant mesuré mais absent du manifeste se publierait
// `other` sans que personne le sache — c'est exactement l'état d'avant le 2026-08-18, où 19 des
// 21 tombaient dans `other`. Et un identifiant du manifeste qui n'est plus au corpus est une
// ligne morte : le corpus a bougé, la table doit suivre.
var corpusObjetsEquipement = map[uint32]string{
	0xbcabbe43: "9 films · 933 poses",
	0x0f5716ff: "8 films · 306",
	0xcaaadcb0: "9 films · 293",
	0xaada07f3: "6 films · 177",
	0x273fe0eb: "5 films · 105",
	0x8c77ffe7: "3 films · 83",
	0x7ca85adc: "4 films · 77",
	0xeef5d48d: "3 films · 70",
	0x72199cba: "3 films · 60",
	0x4396db42: "5 films · 51",
	0x32d97758: "2 films · 48",
	0x528fce46: "3 films · 45",
	0x72b63d69: "2 films · 43",
	0x8e2dc574: "3 films · 42",
	0x2974c233: "3 films · 41",
	0x430dda48: "2 films · 33",
	0x686b40c9: "3 films · 25",
	0x4744d742: "1 film · 4",
	0xb781197a: "1 film · 1",
	0x730dc70f: "1 film · 1",
	0xe7be9f5c: "1 film · 1",
}

func TestPariteObjetsEquipementDuCorpus(t *testing.T) {
	cat, err := Load(repoRoot(t), "halo_infinite")
	if err != nil {
		t.Fatalf("chargement du catalogue : %v", err)
	}
	var manquants, morts []string
	for id, effectif := range corpusObjetsEquipement {
		if _, ok := cat.EquipmentFamilies[id]; !ok {
			manquants = append(manquants, fmt.Sprintf("0x%08x (%s)", id, effectif))
		}
	}
	for id := range cat.EquipmentFamilies {
		if _, ok := corpusObjetsEquipement[id]; !ok {
			morts = append(morts, fmt.Sprintf("0x%08x", id))
		}
	}
	sort.Strings(manquants)
	sort.Strings(morts)
	if len(manquants) > 0 {
		t.Errorf("identifiant(s) `eqip` mesuré(s) au corpus mais ABSENT(S) de "+
			"[[equipment_objects]] : %v — ils se publieraient `other` sans que rien ne le dise. "+
			"Ajouter une ligne, même avec la famille `other` et la provenance `aucune`", manquants)
	}
	if len(morts) > 0 {
		t.Errorf("ligne(s) [[equipment_objects]] dont l'identifiant n'est plus au corpus : %v — "+
			"mettre à jour corpusObjetsEquipement, ou retirer la ligne", morts)
	}
	// Le garde-rail doit pouvoir échouer : une table vide passerait les deux boucles ci-dessus
	// si le corpus l'était aussi.
	if len(cat.EquipmentFamilies) == 0 {
		t.Fatal("aucun objet d'équipement au catalogue : la section [[equipment_objects]] n'est plus lue")
	}
}

// famillesSansWeaponKey — LE TROU CONNU, FIGÉ POUR QU'IL NE GROSSISSE PAS EN SILENCE.
//
// Le décodeur connaît des familles d'arme que le REGISTRE du titre ne porte pas : ce sont
// des curiosités de bac à sable (Fiesta). Elles n'ont donc pas de weapon_key, donc pas de
// libellé, donc elles gardent leur hexadécimal à l'écran — ce qui est la règle du
// chantier, mais qui doit rester une liste COURTE et VUE.
//
// Aucune n'apparaît dans les trois artefacts construits à ce jour (mesuré le 2026-08-02).
var famillesSansWeaponKey = map[string]bool{
	"Sandwich":        true,
	"Mythic Sandwich": true,
	"Mutilator":       true,
}

func TestFamillesDArmeConnuesDuDecodeurSontAuRegistre(t *testing.T) {
	registre := weapons.FilmshellWeaponKeysByFamily()
	var nouvelles []string
	for high, nom := range weaponv3.KnownWeaponHigh32 {
		if _, ok := registre[high]; ok {
			continue
		}
		if famillesSansWeaponKey[nom] {
			continue
		}
		nouvelles = append(nouvelles, nom)
	}
	sort.Strings(nouvelles)
	if len(nouvelles) > 0 {
		t.Errorf("famille(s) d'arme connues du décodeur mais ABSENTES du registre du titre : %v — "+
			"elles n'auront aucun libellé dans le rejeu. Les ajouter au registre, ou les inscrire "+
			"dans famillesSansWeaponKey avec la raison", nouvelles)
	}
	// Le garde-rail doit pouvoir échouer : si l'allowlist ne décrit plus la réalité,
	// c'est elle qui est périmée (leçon J4.0 — un garde qui ne peut pas échouer ne garde rien).
	for nom := range famillesSansWeaponKey {
		trouvee := false
		for high, n := range weaponv3.KnownWeaponHigh32 {
			if n != nom {
				continue
			}
			trouvee = true
			if _, ok := registre[high]; ok {
				t.Errorf("%q est désormais AU REGISTRE : la retirer de famillesSansWeaponKey", nom)
			}
		}
		if !trouvee {
			t.Errorf("%q n'est plus connue du décodeur : entrée périmée dans famillesSansWeaponKey", nom)
		}
	}
}
