package hinavmesh

import (
	"fmt"
	"testing"
)

// region3 rend le fichier-tag de la region 3 (hkaiTraversalAnnotationLibrary).
func region3(t *testing.T, asset string) *fichierTag {
	t.Helper()
	d := chargeRegionsTemoin(t, asset)
	for _, r := range d {
		if len(r) < 8 || string(r[4:8]) != "TAG0" {
			continue
		}
		f, err := lireFichierTag(r)
		if err != nil {
			continue
		}
		rac, err := f.racine()
		if err != nil {
			continue
		}
		if f.nomType(rac.Type) == "hkaiTraversalAnnotationLibrary" {
			return f
		}
	}
	t.Fatal("region 3 introuvable")
	return nil
}

func TestZZTypeTableRegion3(t *testing.T) {
	f := region3(t, "01af558d-53ab-4f05-ba68-92d805fc6260")
	t.Log("=== TABLE DES TYPES, region 3 (Isolation) ===")
	for i := 1; i < len(f.types); i++ {
		ty := f.types[i]
		mod := ""
		if len(ty.Modeles) > 0 {
			for k, v := range ty.Modeles {
				mod += fmt.Sprintf(" %s=%d(%s)", k, v, f.nomType(v))
			}
		}
		t.Logf("[%2d] %-45s parent=%-2d taille=%-4d membres=%d%s", i, ty.Nom, ty.Parent, ty.Taille, len(ty.Membres), mod)
		for _, m := range ty.Membres {
			t.Logf("        +%-4d %-28s type=%d (%s) [taille %d]", m.Offset, m.Nom, m.Type, f.nomType(m.Type), f.types.taille(m.Type))
		}
	}
}
