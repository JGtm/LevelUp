package replay

// geometry_ligne_invalide_test.go — UNE LIGNE CSV ILLISIBLE EST UNE ERREUR NOMMEE (lot J2.10,
// constat RA1-7, 2026-09-26).
//
// `LoadGeometry` sautait en silence une ligne dont le `type_id` ne se lisait pas et lisait 0 pour
// une coordonnee illisible : un prop dessine a l origine, ou un prop disparu, sans une ligne de
// journal. La regle (CLAUDE.md regle 3) : l erreur remonte, TYPEE avec son fichier, sa ligne et
// sa colonne, et c est l appelant qui journalise avant de degrader.

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// ecrireGeometrie pose un catalogue de types et un CSV de props dans un repertoire temporaire, et
// rend (mapDir, typesDir).
func ecrireGeometrie(t *testing.T, types, props string) (string, string) {
	t.Helper()
	typesDir := filepath.Join(t.TempDir(), "map_geometry")
	mapDir := filepath.Join(typesDir, "ridgeline")
	if err := os.MkdirAll(mapDir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(typesDir, ObjectTypesFile), []byte(types), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(mapDir, MapObjectsFile), []byte(props), 0o600); err != nil {
		t.Fatal(err)
	}
	return mapDir, typesDir
}

func TestLoadGeometry_LigneCSVInvalideRendUneErreurNommee(t *testing.T) {
	const typesOK = "type_id,dx,dy,geom\n42,2.0,3.0,ok\n"
	const propsOK = "type_id,x,y,z,yaw_deg\n42,1.5,2.5,3.5,90\n"
	cas := []struct {
		nom, types, props, fichier, colonne string
		ligne                               int
	}{
		{"type_id de prop illisible", typesOK, propsOK + "4x2,1,2,3,0\n", MapObjectsFile, "type_id", 3},
		{"coordonnee illisible", typesOK, propsOK + "42,1,deux,3,0\n", MapObjectsFile, "y", 3},
		{"lacet vide", typesOK, "type_id,x,y,z,yaw_deg\n42,1,2,3,\n", MapObjectsFile, "yaw_deg", 2},
		{"type_id du catalogue illisible", typesOK + "?,1,1,ok\n", propsOK, ObjectTypesFile, "type_id", 3},
		{"emprise mesuree illisible", "type_id,dx,dy,geom\n42,2.0,trois,ok\n", propsOK, ObjectTypesFile, "dy", 2},
	}
	for _, c := range cas {
		mapDir, typesDir := ecrireGeometrie(t, c.types, c.props)
		_, _, err := LoadGeometry(mapDir, typesDir)
		var ligne *LigneDeGeometrieError
		if !errors.As(err, &ligne) {
			t.Errorf("%s : erreur %v, attendu *LigneDeGeometrieError", c.nom, err)
			continue
		}
		if filepath.Base(ligne.Fichier) != c.fichier || ligne.Ligne != c.ligne || ligne.Colonne != c.colonne {
			t.Errorf("%s : %s ligne %d colonne %q, attendu %s ligne %d colonne %q", c.nom,
				filepath.Base(ligne.Fichier), ligne.Ligne, ligne.Colonne, c.fichier, c.ligne, c.colonne)
		}
	}
}

// TestLoadGeometry_EmpriseFactivePeutEtreVide : une ligne de catalogue NON mesuree (modele vide)
// est ecartee sans lire son emprise — l exiger lisible refuserait un catalogue valide.
func TestLoadGeometry_EmpriseFactivePeutEtreVide(t *testing.T) {
	mapDir, typesDir := ecrireGeometrie(t, "type_id,dx,dy,geom\n42,2.0,3.0,ok\n7,,,modele_vide\n",
		"type_id,x,y,z,yaw_deg\n42,1.5,2.5,3.5,90\n7,0,0,0,0\n")
	objs, ecartes, err := LoadGeometry(mapDir, typesDir)
	if err != nil || len(objs) != 1 || ecartes != 1 {
		t.Fatalf("objets %d, ecartes %d, erreur %v : attendu 1, 1, nil", len(objs), ecartes, err)
	}
}
