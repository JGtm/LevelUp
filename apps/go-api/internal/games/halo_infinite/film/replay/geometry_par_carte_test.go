package replay

// geometry_par_carte_test.go — LES PROPS APPARTIENNENT A UNE CARTE, PAS AU TITRE.
//
// LE DEFAUT QUE CE LOT CORRIGE. `MapGeometryDir` ne prenait que le titre et rendait UN
// repertoire : le CSV qui s'y trouvait etait dessine sur CHAQUE match, quelle que soit la carte
// jouee. Mesure du 2026-09-11 sur le parc : 382 props IDENTIQUES sur les 76 artefacts, cartes
// confondues. Tant que le sol reconstruit etait un amas de rectangles, ces props etaient les
// seuls reperes lisibles et les servir partout se defendait ; avec un fond correct, un decor
// appartenant a une AUTRE carte est une donnee fausse sur un outil ou l'on mesure des positions.
//
// DEUX REPERTOIRES, PARCE QUE LES DEUX FICHIERS N'ONT PAS LA MEME PORTEE : les props
// appartiennent a une carte, le catalogue des types (quel identifiant a quelle emprise) vaut
// pour tout le titre. Les lire au meme endroit obligerait a recopier le catalogue sous chaque
// carte.

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadGeometryCarteSansFichierRendNilSansErreur(t *testing.T) {
	// UNE CARTE SANS PROPS EST LE CAS NOMINAL, pas une panne : personne n'a extrait les props
	// des 79 cartes du catalogue. Le chargeur doit rendre zero prop et AUCUNE erreur, sinon
	// chaque match d'une carte non extraite journaliserait un avertissement.
	racine := t.TempDir()
	typesDir := filepath.Join(racine, "map_geometry")
	if err := os.MkdirAll(typesDir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(typesDir, ObjectTypesFile),
		[]byte("type_id,dx,dy,geom\n42,2.0,3.0,ok\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	objs, ecartes, err := LoadGeometry(filepath.Join(typesDir, "carte_sans_props"), typesDir)
	if err != nil {
		t.Fatalf("une carte sans fichier de props n'est pas une erreur : %v", err)
	}
	if objs != nil || ecartes != 0 {
		t.Errorf("zero prop attendu, obtenu %d objets / %d ecartes", len(objs), ecartes)
	}
}

func TestLoadGeometryLitLesPropsDeLaCarteEtLeCatalogueDuTitre(t *testing.T) {
	// Le catalogue des types vit UN CRAN AU-DESSUS des props : c'est ce qui evite de le recopier
	// sous chaque carte.
	racine := t.TempDir()
	typesDir := filepath.Join(racine, "map_geometry")
	mapDir := filepath.Join(typesDir, "ridgeline")
	if err := os.MkdirAll(mapDir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(typesDir, ObjectTypesFile),
		[]byte("type_id,dx,dy,geom\n42,2.0,3.0,ok\n7,0.001,0.001,modele_vide\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(mapDir, MapObjectsFile),
		[]byte("type_id,x,y,z,yaw_deg\n42,1.5,2.5,3.5,90\n7,0,0,0,0\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	objs, ecartes, err := LoadGeometry(mapDir, typesDir)
	if err != nil {
		t.Fatalf("LoadGeometry : %v", err)
	}
	if len(objs) != 1 || objs[0].TypeID != 42 || objs[0].X != 1.5 {
		t.Fatalf("un seul prop dessinable attendu (type 42), obtenu %+v", objs)
	}
	if ecartes != 1 {
		t.Errorf("le type sans emprise mesuree doit etre COMPTE : 1 attendu, obtenu %d", ecartes)
	}
}

func TestLoadGeometryCatalogueDeTypesAbsentEstUneErreur(t *testing.T) {
	// LE CATALOGUE, LUI, N'EST PAS OPTIONNEL : sans lui aucun prop n'a d'emprise, et se taire
	// rendrait indiscernables « cette carte n'a pas de props » et « le titre a perdu sa table
	// des emprises ».
	racine := t.TempDir()
	if _, _, err := LoadGeometry(filepath.Join(racine, "ridgeline"), racine); err == nil {
		t.Error("un catalogue de types absent doit remonter une erreur")
	}
}
