package profile

// frontiere_test.go — CE QUE LA COUCHE `profile` PROMET, PROUVE DANS LA COUCHE ELLE-MEME.
//
// # POURQUOI CE FICHIER EXISTE (D11 du plan, lot 2.5.e-d)
//
// A sa naissance (lot 2.5.b) la couche `profile` portait DIX fichiers de production et ZERO
// `_test.go` : son comportement etait couvert, mais depuis la couche du DESSUS — la table par
// `grammar/build_profile_test.go`, les invariants par `grammar/profile_globales_test.go`, la
// resolution par `grammar/profile_test.go`. C etait tenable tant que les deux paquets vivaient
// cote a cote ; ca ne l est plus maintenant que `grammar` ne peut plus etre importe de
// l exterieur du decodeur : une couche dont AUCUN test ne vit chez elle est une couche dont
// personne ne peut verifier la promesse sans ouvrir la couche voisine.
//
// # CE QUE CES TROIS TESTS GARDENT, ET CE QU ILS NE DOUBLENT PAS
//
// Ils gardent LA FRONTIERE, pas le contenu : que la couche est une FEUILLE, qu elle rend une
// erreur TYPEE sur une cle inconnue plutot que de mentir, et que sa table est lisible
// d elle-meme. Le CONTENU de la table (quelle largeur pour quel build) reste garde la ou il est
// mesure — chez `grammar`, qui le confronte a des films reels ; le doubler ici serait la copie
// de garde-rail que CLAUDE.md regle 6 interdit.

import (
	"errors"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

// TestCoucheProfileEstUneFeuille : `profile` n importe RIEN du depot.
//
// C EST LA PROMESSE LA PLUS FORTE DE LA COUCHE, et celle qui la rend utilisable par tout le
// monde : elle ne lit aucun octet de film (`source`), elle ne connait aucun decodeur
// (`grammar`), donc elle ne peut pas ramener un cycle. Le sens unique
// `source -> profile -> grammar -> facts -> replay` est garde au niveau du decodeur par
// `archlint/film_layers_deps_test.go` ; ce test-ci est la meme verite dite DANS la couche, et
// il rougit sans qu on ait a ouvrir `internal/archlint`.
//
// Il lit les imports par `go/parser` et non par `grep` : les en-tetes de ce paquet CITENT
// `source` et `grammar` en toutes lettres (c est ainsi qu ils expliquent le sens unique), et un
// grep rougirait sur la documentation.
func TestCoucheProfileEstUneFeuille(t *testing.T) {
	const prefixeDepot = "levelup/go-api/"
	fset := token.NewFileSet()
	entrees, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("lecture du paquet : %v", err)
	}
	var vus int
	var fautifs []string
	for _, e := range entrees {
		nom := e.Name()
		if e.IsDir() || !strings.HasSuffix(nom, ".go") || strings.HasSuffix(nom, "_test.go") {
			continue
		}
		vus++
		f, perr := parser.ParseFile(fset, nom, nil, parser.ImportsOnly)
		if perr != nil {
			t.Fatalf("analyse de %s : %v", nom, perr)
		}
		for _, imp := range f.Imports {
			chemin := strings.Trim(imp.Path.Value, `"`)
			if strings.HasPrefix(chemin, prefixeDepot) {
				fautifs = append(fautifs, nom+" -> "+chemin)
			}
		}
	}
	if vus < 8 {
		t.Fatalf("balayage muet : %d fichier(s) de production vus, au moins 8 attendus — la "+
			"couche a bouge ou le filtre est casse, et ce test ne garderait plus rien", vus)
	}
	if len(fautifs) > 0 {
		t.Errorf("la couche `profile` n est plus une FEUILLE (%d import(s) du depot) :\n  %s\n"+
			"Elle porte la DONNEE, pas le lecteur : tout ce qui LIT un octet de film vit en "+
			"`grammar` et rend un type de ce paquet (inversion de dependance du lot 2.5.b). "+
			"Un import vers `grammar` ici est exactement l arete que ce lot a supprimee.",
			len(fautifs), strings.Join(fautifs, "\n  "))
	}
}

// TestCleInconnueRendUneErreurTypee : une cle absente de la table ne MENT pas.
//
// ADR 0034 D-4 : lire un film au profil du build voisin est interdit — pas le lire du tout.
// [Resoudre] rend donc un profil dont [Profile.Err] porte une erreur TYPEE, que l appelant
// consulte par `errors.Is`, et le reste du profil (invariants, carte) est pose quand meme.
// Tester la CHAINE de l erreur serait fragile et faux : c est la SENTINELLE qui est le contrat.
func TestCleInconnueRendUneErreurTypee(t *testing.T) {
	p := Resoudre(ClesDuFilm{
		RegistrePresent: true,
		Format:          999999,
		Majeure:         41,
		MajeureLue:      true,
		Identite:        FilmIdentity{Build: "HI_BUILD_QUI_N_EXISTE_PAS"},
		IdentiteLue:     true,
	}, nil)
	err := p.Err()
	if err == nil {
		t.Fatal("cle inconnue : `Err()` est nil — le profil a servi un repli silencieux, ce que " +
			"D-4 d ADR 0034 interdit")
	}
	if !errors.Is(err, ErrUnknownFormat) {
		t.Errorf("format inconnu : `errors.Is(err, ErrUnknownFormat)` est faux (err = %v)", err)
	}
	if !errors.Is(err, ErrUnknownBuild) {
		t.Errorf("build inconnu : `errors.Is(err, ErrUnknownBuild)` est faux (err = %v)", err)
	}
}

// TestTableProfilEstLisibleDElleMeme : chaque ligne de la table porte ses cinq champs.
//
// La table EST la documentation du profil — c est ce que `docs/RUNBOOK_FILM_PROFILES.md` promet
// a qui doit comprendre pourquoi un film se lit d une facon plutot que d une autre. Une ligne
// sans provenance ni date n est pas une ligne de moins : c est une ligne qui a l air d etre
// prouvee et ne l est pas.
func TestTableProfilEstLisibleDElleMeme(t *testing.T) {
	table := TableProfil()
	if len(table) < 16 {
		t.Fatalf("table de profil : %d ligne(s), au moins 16 attendues — la table a ete videe "+
			"ou le montage des quatre sous-tables est casse", len(table))
	}
	for i, l := range table {
		for nom, v := range map[string]string{
			"Cle": l.Cle, "Champ": l.Champ, "Valeur": l.Valeur, "Preuve": l.Preuve, "Date": l.Date,
		} {
			if strings.TrimSpace(v) == "" {
				t.Errorf("ligne %d (%s / %s) : champ %s vide — une ligne de profil dit ce "+
					"qu elle pose ET d ou elle le tient", i, l.Cle, l.Champ, nom)
			}
		}
		switch l.Source {
		case ProvenanceRelue, ProvenanceMesuree, ProvenancePresumee:
		default:
			t.Errorf("ligne %d (%s / %s) : provenance %q hors des trois declarees",
				i, l.Cle, l.Champ, l.Source)
		}
		if !dateBienFormee(l.Date) {
			t.Errorf("ligne %d (%s / %s) : date %q n est pas au format AAAA-MM-JJ",
				i, l.Cle, l.Champ, l.Date)
		}
	}
}

// dateBienFormee : la date d une ligne de table, au seul format que la table declare.
func dateBienFormee(s string) bool {
	if len(s) != len("2026-09-17") || s[4] != '-' || s[7] != '-' {
		return false
	}
	for i, c := range s {
		if i == 4 || i == 7 {
			continue
		}
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
