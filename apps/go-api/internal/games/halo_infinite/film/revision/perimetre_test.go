package revision_test

// perimetre_test.go — LE PERIMETRE D UNE COUCHE EST LA FERMETURE DE SES IMPORTS (lot J3.2 du
// PLAN_SUITE_AUDIT_DECODEUR_FILM_2026-09-25, decision DU-2 (b), constat SRC-1).

import (
	"errors"
	"flag"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/revision"
)

// updatePerimetres : LA PORTE DE REGENERATION DES GOLDENS DE PERIMETRE, et d aucun autre. Deux
// verrous, comme les goldens de revision : le drapeau NOMME et `LEVELUP_UPDATE_PERIMETRES=1`.
var updatePerimetres = flag.Bool("update-perimetres", false,
	"reecrire testdata/<couche>_perimetre.golden de chaque couche revisee")

// moduleDeFixture : un module `exemple.test/m` ou la couche `a` importe `b`, qui importe `c` et un
// sous-paquet de la couche revisee `z`. `d` n est importe que par un TEST ; `b/sub` n est importe
// par personne.
func moduleDeFixture(t *testing.T, retouches map[string]string) (revision.Module, []revision.Couche) {
	t.Helper()
	fichiers := map[string]string{
		"go.mod":              "module exemple.test/m\n\ngo 1.22\n",
		"couches/a/a.go":      "package a\n\nimport (\n\t_ \"exemple.test/m/b\"\n\t_ \"fmt\"\n)\n",
		"couches/a/sous/s.go": "package sous\n\nconst S = 1\n",
		"b/b.go":              "package b\n\nimport (\n\t_ \"exemple.test/m/c\"\n\t_ \"exemple.test/m/couches/z/zz\"\n)\n",
		"b/b_test.go":         "package b\n\nimport _ \"exemple.test/m/d\"\n",
		"b/sub/x.go":          "package sub\n\nconst X = 1\n",
		"c/c.go":              "package c\n\nconst C = 1\n",
		"d/d.go":              "package d\n\nconst D = 1\n",
		"couches/z/z.go":      "package z\n\nconst Z = 1\n",
		"couches/z/zz/zz.go":  "package zz\n\nconst ZZ = 1\n",
	}
	for nom, texte := range retouches {
		fichiers[nom] = texte
	}
	m, err := revision.ModuleDe(couche(t, fichiers))
	if err != nil {
		t.Fatalf("module de fixture : %v", err)
	}
	return m, []revision.Couche{{Nom: "a", Racine: "couches/a"}, {Nom: "z", Racine: "couches/z"}}
}

// empreinteDeA rend l empreinte de la couche `a` du module de fixture.
func empreinteDeA(t *testing.T, retouches map[string]string, valeurZ string) string {
	t.Helper()
	m, couches := moduleDeFixture(t, retouches)
	res, _, err := m.CalculerCouche("a", couches, nil, map[string]string{"z": valeurZ})
	if err != nil {
		t.Fatalf("empreinte de a : %v", err)
	}
	return res.Empreinte
}

func TestPerimetre_FermetureDesImports(t *testing.T) {
	m, couches := moduleDeFixture(t, nil)
	p, err := m.Fermeture("a", couches)
	if err != nil {
		t.Fatalf("fermeture : %v", err)
	}
	attendu := revision.Perimetre{
		Paquets: []string{"b", "c", "couches/a", "couches/a/sous"},
		Amonts:  []string{"z"},
	}
	if !p.Egal(attendu) {
		t.Fatalf("fermeture de a = %+v, attendu %+v (A -> B -> C, la couche z par sa VALEUR, ni le "+
			"paquet importe par un test ni le sous-paquet que personne n importe)", p, attendu)
	}

	ref := empreinteDeA(t, nil, "z-1")
	mord := map[string]map[string]string{
		"C, importe par B": {"c/c.go": "package c\n\nconst C = 2\n"},
		"l arbre de A":     {"couches/a/sous/s.go": "package sous\n\nconst S = 2\n"},
	}
	for quoi, retouche := range mord {
		if empreinteDeA(t, retouche, "z-1") == ref {
			t.Errorf("%s change et l empreinte de a ne bouge pas : un paquet de la fermeture "+
				"n est pas hache", quoi)
		}
	}
	neMordPas := map[string]map[string]string{
		"les OCTETS de la couche z":   {"couches/z/zz/zz.go": "package zz\n\nconst ZZ = 2\n"},
		"d, importe par un test":      {"d/d.go": "package d\n\nconst D = 2\n"},
		"b/sub, importe par personne": {"b/sub/x.go": "package sub\n\nconst X = 2\n"},
	}
	for quoi, retouche := range neMordPas {
		if empreinteDeA(t, retouche, "z-1") != ref {
			t.Errorf("%s change l empreinte de a : il n est pas dans son perimetre", quoi)
		}
	}
	if empreinteDeA(t, nil, "z-2") == ref {
		t.Error("la VALEUR de la couche z change et l empreinte de a ne bouge pas")
	}
}

func TestPerimetre_ValeursAmontExactes(t *testing.T) {
	m, couches := moduleDeFixture(t, nil)
	for quoi, valeurs := range map[string]map[string]string{
		"valeur manquante": {},
		"valeur superflue": {"z": "z-1", "fantome": "f-1"},
		"valeur vide":      {"z": ""},
	} {
		if _, _, err := m.CalculerCouche("a", couches, nil, valeurs); !errors.Is(err, revision.ErrPerimetre) {
			t.Errorf("%s : err = %v, attendu ErrPerimetre — une declaration qui diverge de la "+
				"fermeture doit echouer bruyamment", quoi, err)
		}
	}
	if _, err := m.Fermeture("inconnue", couches); !errors.Is(err, revision.ErrPerimetre) {
		t.Errorf("couche inconnue : err = %v, attendu ErrPerimetre", err)
	}
}

func TestCalculer_FichierEmbarqueCompte(t *testing.T) {
	source := "package p\n\nimport _ \"embed\"\n\n//go:embed data/table.tsv\nvar Table string\n"
	ref := empreinteDe(t, couche(t, map[string]string{"p.go": source, "data/table.tsv": "1\ta\n"}))
	if got := empreinteDe(t, couche(t, map[string]string{"p.go": source, "data/table.tsv": "1\tb\n"})); got == ref {
		t.Error("une ligne de la table EMBARQUEE change et l empreinte ne bouge pas (SRC-1 : les " +
			"donnees de `damagetag`)")
	}
	if got := empreinteDe(t, couche(t, map[string]string{"p.go": source, "data/table.tsv": "1\ta\r\n"})); got != ref {
		t.Error("la meme table en CRLF rend une autre empreinte")
	}
	if _, err := revision.Calculer([]string{couche(t, map[string]string{"p.go": source})}, nil); err == nil {
		t.Error("un motif //go:embed sans fichier est accepte : le gate hacherait du vide")
	}
}

// TestPerimetreDeChaqueCoucheEgaleSonGolden : la fermeture de chaque couche revisee est celle que
// son golden fige. Un import ajoute dans une couche le fait rougir.
func TestPerimetreDeChaqueCoucheEgaleSonGolden(t *testing.T) {
	m, err := revision.ModuleDe(racineAPI(t))
	if err != nil {
		t.Fatalf("module : %v", err)
	}
	porte := revision.Porte{Nom: "perimetres"}
	for _, c := range revision.CouchesRevisees() {
		t.Run(c.Nom, func(t *testing.T) {
			calcule, err := m.Fermeture(c.Nom, revision.CouchesRevisees())
			if err != nil {
				t.Fatalf("fermeture de %s : %v", c.Nom, err)
			}
			chemin := filepath.Join(m.Racine, filepath.FromSlash(c.Racine), "testdata", c.Nom+"_perimetre.golden")
			if *updatePerimetres {
				if ouverte, raison := porte.Ouverte(true, os.Getenv(porte.Variable())); !ouverte {
					t.Skip(raison)
				}
				ecrit, err := porte.ReecrirePerimetre(chemin, calcule)
				if err != nil {
					t.Fatalf("regeneration : %v", err)
				}
				t.Fatal(ecrit)
			}
			fige, err := revision.LirePerimetre(chemin)
			if err != nil {
				t.Fatalf("golden de perimetre de %s : %v", c.Nom, err)
			}
			if !calcule.Egal(fige) {
				t.Fatalf("LE PERIMETRE DE LA COUCHE %s A CHANGE.\n  fige    : %+v\n  calcule : %+v\n"+
					"Un paquet entre (ou sort) de la fermeture de ses imports : il decide desormais "+
					"(ou plus) de la sortie de la couche. Le verifier, puis regenerer :\n"+
					"  %s=1 go test ./internal/games/halo_infinite/film/revision/ "+
					"-run TestPerimetreDeChaqueCoucheEgaleSonGolden -%s",
					c.Nom, fige, calcule, porte.Variable(), porte.Drapeau())
			}
		})
	}
}
