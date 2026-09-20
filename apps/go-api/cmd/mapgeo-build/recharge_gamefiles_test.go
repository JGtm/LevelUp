//go:build gamefiles

package main

// recharge_gamefiles_test.go — LE TEMOIN QUI OUVRE LE JEU, isole sous son tag.
//
// Il cuit UNE carte (Recharge, la carte de calibrage) et verifie ce qu'une mesure
// geometrique doit tenir pour valoir quelque chose : des triangles, des noeuds, les ancres
// d'objectif posees sur un sol reconstruit, un graphe d'un seul tenant (au petit reste
// pres), des rayons lances. Tag `gamefiles` + suffixe `_gamefiles_test.go` : regle de
// CLAUDE.md, garde-rail `internal/archlint/gamefiles_tag_test.go`. Duree mesuree : ~5 s.
//
// Il lit les donnees de reference de `LEVELUP_DATA_ROOT` (la racine qui CONTIENT `data/`)
// ou, a defaut, de la racine du depot.

import (
	"context"
	"os"
	"testing"

	"levelup/go-api/internal/analysis/powerpos/geo"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/himap"
	"levelup/go-api/internal/testutil"
)

func TestCuissonRecharge(t *testing.T) {
	if _, err := himap.DeployRoot(); err != nil {
		t.Skip(err)
	}
	racine := os.Getenv("LEVELUP_DATA_ROOT")
	if racine == "" {
		r, err := testutil.RepoRoot()
		if err != nil {
			t.Fatalf("racine du depot introuvable : %v", err)
		}
		racine = r
	}
	res := title.NewPathResolver(racine)
	cibles, err := ResoutCibles(res, title.DefaultSlug, []string{"recharge"})
	if err != nil {
		t.Skip("recharge non resolvable ici :", err)
	}
	c, err := Cuit(context.Background(), cibles[0], geo.ParametresParDefaut(), geo.ReglageGeoV1())
	if err != nil {
		t.Fatal(err)
	}
	b := c.Bilan
	if c.Triangles.Triangles < 1_000_000 {
		t.Errorf("triangles = %d, au moins un million attendus sur Recharge", c.Triangles.Triangles)
	}
	// 2 901 noeuds mesures le 2026-09-20 (apres l'elagage des 1 188 noeuds sans retour du
	// niveau bas de la halle) : un plancher a 2 500 attrape un sol qui se troue, pas une
	// variation de decor.
	if len(c.Resultat.Noeuds) < 2500 {
		t.Errorf("noeuds = %d, au moins 2 500 attendus", len(c.Resultat.Noeuds))
	}
	if b.Sol.AncresSansNoeud > 0 {
		t.Errorf("%d germe(s) sans noeud : le sol reconstruit ne porte pas toutes les ancres", b.Sol.AncresSansNoeud)
	}
	total := 0
	for _, t := range b.Sol.TaillesComposantes {
		total += t
	}
	if total == 0 || b.Sol.TaillesComposantes[0] < 9*total/10 {
		t.Errorf("composante principale %v : le sol est en morceaux", b.Sol.TaillesComposantes)
	}
	if b.Rayons < 1_000_000 || b.Cibles < 500 {
		t.Errorf("rayons %d, cibles %d : la visibilite n'a pas ete mesuree", b.Rayons, b.Cibles)
	}
	if !c.FrontiereAppliquee {
		t.Error("la coquille de mort n'a pas ete appliquee sur Recharge")
	}
	if len(c.Positions) == 0 {
		t.Error("aucune position retenue")
	}
	t.Logf("noeuds %d, composantes %v, rayons %d, positions %d, duree %s",
		len(c.Resultat.Noeuds), b.Sol.TaillesComposantes, b.Rayons, len(c.Positions), c.DureeTotale)
}
