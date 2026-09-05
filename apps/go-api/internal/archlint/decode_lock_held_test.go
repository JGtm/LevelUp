package archlint

// decode_lock_held_test.go — TOUT CHEMIN DE PRODUCTION QUI BALAIE UN FILM TIENT LE VERROU DE
// DECODAGE (lot E, item E.5 du PLAN_V2_REJEU_FILM, 2026-09-05).
//
// # LE CONTRAT, ET POURQUOI IL EST DUR
//
// `internal/analysis/filmdec/decode_gate.go:16-18` l'ecrit noir sur blanc : « tout chemin qui
// enchaine les balayages de ce paquet (`Scan*`, walk killsource) acquiert ce verrou pour TOUTE la
// duree du decodage d'un film — jamais par sous-appel ». La raison n'est pas theorique : les
// parametres de replication du decodeur de bits sont des GLOBAUX DE PAQUET, et le paquet chiffre
// lui-meme ce qu'un entrelacement coute — « le score d'un film passe de 1111 a 1214 selon l'ordre
// d'appel » (`killsource/decode.go:91-93`).
//
// # CE QUI EXISTAIT, ET CE QUE CE FICHIER GENERALISE
//
// Un garde-rail du verrou existait deja, mais BORNE A UNE FONCTION :
// `replay/world_object_precision_guard_test.go` lit la source de `BuildFromFilm` et echoue si
// `LockProcessDecode` n'y apparait pas AVANT l'installation des largeurs. Il ne couvrait donc pas
// `killcollector/positions.go`, qui enchainait QUATRE balayages sans le verrou — l'ecart releve
// au registre (constat E5) et corrige par le meme lot. Ce ratchet-ci est la generalisation : il
// ne regarde plus une fonction, il regarde le MODULE.
//
// # LA REGLE, EXACTEMENT
//
// Dans chaque paquet de production hors `filmdec` lui-meme, une fonction est dite SOUS VERROU si
// elle appelle `filmdec.LockProcessDecode()`, ou si TOUS ses appelants du meme paquet sont eux
// memes sous verrou (point fixe, en remontant). Toute fonction qui appelle un balayage `filmdec`
// doit etre sous verrou.
//
// LA RECIPROQUE EST AUSSI IMPORTANTE : le mutex N'EST PAS REENTRANT. Prendre le verrou dans une
// fonction dont l'appelant le tient deja bloquerait le process. La regle n'exige donc PAS que
// chaque balayeur le prenne — elle exige qu'il soit couvert.
//
// RETRAIT CIBLE : le jour ou `filmdec` est de-globalise (decision D10, meme critere que
// `filmdec_package_vars_test.go`) — `LockProcessDecode` disparait, et ce ratchet avec lui.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// paquetsQuiDecodent : les paquets de production dont les fonctions balaient un film. La liste
// est FERMEE et se re-mesure : `grep -rl "filmdec\.Scan" --include=*.go internal cmd` hors
// `internal/analysis/filmdec`. Un paquet neuf qui balaie doit entrer ici — c'est un geste
// conscient, pas un effet de bord.
var paquetsQuiDecodent = []string{
	"internal/analysis/replay",
	"internal/games/halo_infinite/film/killsource",
	"internal/sync/killcollector",
}

// balayageFilmdec dit si `sel` est un balayage de film du paquet `filmdec` : les familles
// `Scan*` et les decodeurs de trame, qui lisent tous les globaux de replication.
func balayageFilmdec(pkg, fun string) bool {
	if pkg != "filmdec" {
		return false
	}
	return strings.HasPrefix(fun, "Scan") || strings.HasPrefix(fun, "DecodeFrame") ||
		strings.HasPrefix(fun, "TraverseEntity")
}

// TestBalayagesFilmdecSousVerrou — LE RATCHET.
func TestBalayagesFilmdecSousVerrou(t *testing.T) {
	racine := apiRootDepuisIci(t)
	for _, rel := range paquetsQuiDecodent {
		t.Run(rel, func(t *testing.T) {
			verifierPaquetSousVerrou(t, filepath.Join(racine, filepath.FromSlash(rel)), rel)
		})
	}
}

// fonctionDuPaquet porte ce que le ratchet a besoin de savoir d'une fonction.
type fonctionDuPaquet struct {
	nom      string
	fichier  string
	ligne    int
	prend    bool     // appelle filmdec.LockProcessDecode()
	balaie   bool     // appelle un balayage filmdec
	appelles []string // noms de fonctions du MEME paquet qu'elle appelle
}

// verifierPaquetSousVerrou lit un paquet, calcule le point fixe « sous verrou » et exige que
// tout balayeur y soit.
func verifierPaquetSousVerrou(t *testing.T, dir, rel string) {
	t.Helper()
	fns := lireFonctionsDuPaquet(t, dir)
	if len(fns) == 0 {
		t.Fatalf("%s : aucune fonction lue — le ratchet ne mesure plus rien", rel)
	}
	sousVerrou := pointFixeSousVerrou(fns)

	var manquants []string
	for _, nom := range nomsTries(fns) {
		f := fns[nom]
		if f.balaie && !sousVerrou[nom] {
			manquants = append(manquants, fnSituee(rel, f))
		}
	}
	if len(manquants) > 0 {
		t.Fatalf("ces fonctions balaient un film SANS que le verrou de decodage soit tenu :\n  %s\n"+
			"Contrat : filmdec/decode_gate.go:16-18. Prendre le verrou en tete de la fonction\n"+
			"(`release := filmdec.LockProcessDecode(); defer release()`) OU s'assurer que TOUS ses\n"+
			"appelants du paquet le tiennent. ATTENTION : le mutex n'est pas reentrant — ne pas le\n"+
			"prendre deux fois sur le meme chemin.", strings.Join(manquants, "\n  "))
	}
}

// pointFixeSousVerrou rend l'ensemble des fonctions couvertes par le verrou : celles qui le
// prennent, plus celles dont TOUS les appelants du paquet sont couverts (et qui en ont au moins
// un). L'iteration s'arrete quand l'ensemble ne grandit plus.
func pointFixeSousVerrou(fns map[string]*fonctionDuPaquet) map[string]bool {
	appelants := map[string][]string{}
	for nom, f := range fns {
		for _, appele := range f.appelles {
			if _, ok := fns[appele]; ok {
				appelants[appele] = append(appelants[appele], nom)
			}
		}
	}
	couvert := map[string]bool{}
	for nom, f := range fns {
		if f.prend {
			couvert[nom] = true
		}
	}
	for bouge := true; bouge; {
		bouge = false
		for nom := range fns {
			if couvert[nom] {
				continue
			}
			appelantsDe := appelants[nom]
			if len(appelantsDe) == 0 {
				continue // point d'entree du paquet : il doit prendre le verrou lui-meme
			}
			tous := true
			for _, a := range appelantsDe {
				if !couvert[a] {
					tous = false
					break
				}
			}
			if tous {
				couvert[nom], bouge = true, true
			}
		}
	}
	return couvert
}

// lireFonctionsDuPaquet analyse les sources non-test d un paquet et rend ses fonctions indexees
// par leur nom SIMPLE (methodes comprises, sans recepteur) : c est la seule forme qui se recoupe
// avec les noms d appel lus dans les corps, ou `c.collectPositions()` ne donne que `collectPositions`.
// Deux methodes homonymes de types differents se confondraient — le cas n existe dans aucun des
// trois paquets mesures, et une confusion ne peut que RESSERRER la regle (les deux devraient etre
// couvertes), jamais la relacher.
func lireFonctionsDuPaquet(t *testing.T, dir string) map[string]*fonctionDuPaquet {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("paquet %s introuvable (%v) : s'il a DEMENAGE, deplacer ce ratchet avec lui", dir, err)
	}
	fset := token.NewFileSet()
	out := map[string]*fonctionDuPaquet{}
	for _, e := range entries {
		nom := e.Name()
		if e.IsDir() || !strings.HasSuffix(nom, ".go") || strings.HasSuffix(nom, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, filepath.Join(dir, nom), nil, 0)
		if err != nil {
			t.Fatalf("analyse de %s : %v", nom, err)
		}
		for _, d := range f.Decls {
			fd, ok := d.(*ast.FuncDecl)
			if !ok || fd.Body == nil {
				continue
			}
			fn := &fonctionDuPaquet{nom: fd.Name.Name, fichier: nom, ligne: fset.Position(fd.Pos()).Line}
			remplirAppels(fd.Body, fn)
			out[fn.nom] = fn
		}
	}
	return out
}

// remplirAppels parcourt un corps et note : prise du verrou, balayage filmdec, et les appels
// aux fonctions ou methodes du meme paquet.
func remplirAppels(body *ast.BlockStmt, fn *fonctionDuPaquet) {
	ast.Inspect(body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		switch f := call.Fun.(type) {
		case *ast.Ident: // appel d'une fonction du meme paquet
			fn.appelles = append(fn.appelles, f.Name)
		case *ast.SelectorExpr:
			pkg, estIdent := f.X.(*ast.Ident)
			switch {
			case !estIdent:
				// appel sur une expression (`c.x.y()`) : le recepteur n'est pas resoluble
				// syntaxiquement, on ne le compte pas.
			case pkg.Name == "filmdec" && f.Sel.Name == "LockProcessDecode":
				fn.prend = true
			case balayageFilmdec(pkg.Name, f.Sel.Name):
				fn.balaie = true
			default:
				// methode d'un recepteur du paquet : `c.collectPositions()` -> on tente les deux
				// formes, la resolution exacte du type demanderait le type-checker complet.
				fn.appelles = append(fn.appelles, f.Sel.Name)
			}
		}
		return true
	})
}

// nomsTries rend les cles triees, pour un message d'echec deterministe.
func nomsTries(fns map[string]*fonctionDuPaquet) []string {
	out := make([]string, 0, len(fns))
	for nom := range fns {
		out = append(out, nom)
	}
	sort.Strings(out)
	return out
}

// fnSituee rend « paquet/fichier:ligne fonction », de quoi ouvrir le site directement.
func fnSituee(rel string, f *fonctionDuPaquet) string {
	return rel + "/" + f.fichier + ":" + itoa(f.ligne) + " " + f.nom
}
