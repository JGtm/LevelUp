package revision_test

// jetons_test.go — L EMPREINTE PORTE SUR LES JETONS DU LANGAGE, PAS SUR LES OCTETS (lot J3.1 du
// PLAN_SUITE_AUDIT_DECODEUR_FILM_2026-09-25, decision DU-2 (a)).
//
// Ce que l audit du 2026-09-24 mesure : 46 % des lignes des couches sont des commentaires, et
// chacune d elles faisait monter une revision — c est-a-dire, pour les faits, rouvrir un backlog
// de redecodage pour une phrase reformulee. Les quatre cas ci-dessous tiennent la frontiere :
// un commentaire est sans effet, un jeton en a un, une chaine qui contient `//` reste une chaine,
// et une directive `//go:` — qui change ce que le compilateur fait — reste comptee.

import (
	"strings"
	"testing"
)

// empreinteDUnFichier rend l empreinte d une couche d un seul fichier `p.go`.
func empreinteDUnFichier(t *testing.T, texte string) string {
	t.Helper()
	return empreinteDe(t, couche(t, map[string]string{"p.go": texte}))
}

func TestCalculer_CommentaireSansEffet(t *testing.T) {
	ref := empreinteDUnFichier(t, "package p\n\nconst Largeur = 5\n\nfunc Lire() int { return Largeur }\n")
	commente := empreinteDUnFichier(t, "// Package p decrit la couche.\npackage p\n\n"+
		"// Largeur : la largeur lue, EN BITS (reformulee trois fois).\nconst Largeur = 5 // en bits\n\n"+
		"/* bloc\n   de plusieurs lignes */\nfunc Lire() int { return Largeur /* rendu tel quel */ }\n")
	if commente != ref {
		t.Errorf("deux arbres qui ne different que par des commentaires rendent deux empreintes :\n"+
			"  sans : %s\n  avec : %s\nune reformulation ferait monter la revision (DU-2 (a))", ref, commente)
	}
}

func TestCalculer_UnJetonChangeLEmpreinte(t *testing.T) {
	ref := empreinteDUnFichier(t, "package p\n\nconst Largeur = 5\n")
	for nom, texte := range map[string]string{
		"litteral":      "package p\n\nconst Largeur = 6\n",
		"identifiant":   "package p\n\nconst Hauteur = 5\n",
		"operateur":     "package p\n\nconst Largeur = -5\n",
		"mot-cle":       "package p\n\nvar Largeur = 5\n",
		"jeton-ajoute":  "package p\n\nconst Largeur = 5 + 0\n",
		"chaine-brute":  "package p\n\nconst Largeur = 5\n\nconst S = `a  b`\n",
		"chaine-brute2": "package p\n\nconst Largeur = 5\n\nconst S = `a b`\n",
	} {
		if got := empreinteDUnFichier(t, texte); got == ref {
			t.Errorf("%s : un jeton change et l empreinte ne bouge pas", nom)
		}
	}
	a := empreinteDUnFichier(t, "package p\n\nconst S = `a  b`\n")
	b := empreinteDUnFichier(t, "package p\n\nconst S = `a b`\n")
	if a == b {
		t.Error("un espace A L INTERIEUR d une chaine brute est un jeton : il doit compter")
	}
}

func TestCalculer_ChaineContenantDeuxBarres(t *testing.T) {
	// Une chaine qui porte `//` n est pas un commentaire : ce qui suit les deux barres est du
	// CONTENU. Un decoupage naif « tout ce qui suit // est commentaire » rendrait ces deux
	// fichiers identiques.
	a := empreinteDUnFichier(t, "package p\n\nconst URL = \"http://a.example/x\"\n")
	b := empreinteDUnFichier(t, "package p\n\nconst URL = \"http://b.example/y\"\n")
	if a == b {
		t.Error("le texte apres `//` DANS une chaine a ete traite comme un commentaire")
	}
	c := empreinteDUnFichier(t, "package p\n\nconst R = `/* pas un commentaire */`\n")
	d := empreinteDUnFichier(t, "package p\n\nconst R = `/* autre chose */`\n")
	if c == d {
		t.Error("le texte entre `/*` et `*/` DANS une chaine brute a ete traite comme un commentaire")
	}
}

func TestCalculer_DirectiveGoCompte(t *testing.T) {
	ref := empreinteDUnFichier(t, "package p\n\nvar B []byte\n")
	for nom, texte := range map[string]string{
		"go:build":    "//go:build research\n\npackage p\n\nvar B []byte\n",
		"go:embed":    "package p\n\nimport _ \"embed\"\n\n//go:embed x.txt\nvar B []byte\n",
		"go:noinline": "package p\n\nvar B []byte\n\n//go:noinline\nfunc F() {}\n",
	} {
		sans := empreinteDUnFichier(t, retirerDirectives(texte))
		avec := empreinteDUnFichier(t, texte)
		if avec == sans {
			t.Errorf("%s : la directive ne change pas l empreinte — elle change pourtant ce que "+
				"le compilateur fait", nom)
		}
	}
	e1 := empreinteDUnFichier(t, "//go:build research\n\npackage p\n\nvar B []byte\n")
	e2 := empreinteDUnFichier(t, "//go:build gamefiles\n\npackage p\n\nvar B []byte\n")
	if e1 == e2 || e1 == ref {
		t.Error("deux contraintes de construction differentes rendent la meme empreinte")
	}
}

// retirerDirectives rend le texte sans ses lignes `//go:` — la version « commentaire ordinaire »
// du meme fichier, pour prouver que la directive, elle, compte.
func retirerDirectives(texte string) string {
	var garde []string
	for _, ligne := range strings.Split(texte, "\n") {
		if !strings.HasPrefix(ligne, "//go:") {
			garde = append(garde, ligne)
		}
	}
	return strings.Join(garde, "\n")
}
