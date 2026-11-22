//go:build gamefiles

package main

// cmd/mapfond-build — maillage_partage_gamefiles_test.go : LE MAILLAGE D'UNE CARTE DONT LA GEOMETRIE
// VIT DANS UN MODULE PARTAGE, LU DANS L'INSTALLATION.
//
// Live Fire (`sgh_interlock`) n'a pas de maillage dans son module `any` : le sien est dans le module
// `any` homonyme du module de géométrie partagé (levier `moduleGeometrie`). Une carte ordinaire, elle,
// garde le maillage de son module. Le test ouvre le jeu : il porte le tag `gamefiles` (garde-rail :
// `internal/archlint`, `gamefiles_tag_test.go`).

import (
	"context"
	"testing"

	"levelup/go-api/internal/domain/title"

	"github.com/JGtm/halo-infinite-map-images/himap"
)

// cibleNative rend la cible native `cle` du catalogue d'objectifs installé.
func cibleNative(t *testing.T, e *environnement, cle string) cible {
	t.Helper()
	for _, c := range e.ciblesNatives() {
		if c.cle == cle {
			return c
		}
	}
	t.Skipf("la carte %s n'est pas installée", cle)
	return cible{}
}

func TestMaillageNatifTrouveCeluiDeLiveFire(t *testing.T) {
	if _, err := himap.DeployRoot(); err != nil {
		t.Skip(err)
	}
	e, err := prepare(title.DefaultSlug, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	c := cibleNative(t, e, "sgh_interlock")
	m, degradation, err := e.maillageNatif(context.Background(), c)
	if err != nil {
		t.Fatalf("maillageNatif : %v", err)
	}
	if m == nil || degradation != "" {
		t.Fatalf("Live Fire sans maillage (dégradation %q) : sa géométrie partagée n'a pas été cherchée", degradation)
	}
	if len(m.Faces) == 0 {
		t.Error("le maillage de Live Fire n'a aucune face")
	}
}

// Une carte ordinaire garde le maillage de son module, le même que celui de la brique pour un module
// `any` : le levier de module partagé ne la concerne pas.
func TestMaillageNatifDUneCarteOrdinaireEstInchange(t *testing.T) {
	racine, err := himap.DeployRoot()
	if err != nil {
		t.Skip(err)
	}
	e, err := prepare(title.DefaultSlug, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	c := cibleNative(t, e, "sgh_blueprint")
	m, degradation, err := e.maillageNatif(context.Background(), c)
	if err != nil || m == nil || degradation != "" {
		t.Fatalf("Recharge : maillage %v, dégradation %q, erreur %v", m != nil, degradation, err)
	}
	direct, _, err := himap.ChargeMaillageNatif(context.Background(), racine, c.cle)
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Faces) != len(direct.Faces) || len(m.Sommets) != len(direct.Sommets) {
		t.Errorf("maillage de Recharge : %d faces / %d sommets, lecture directe %d / %d",
			len(m.Faces), len(m.Sommets), len(direct.Faces), len(direct.Sommets))
	}
}
