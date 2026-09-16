package archlint

// decode_lock_interdit_test.go — LE VERROU DE DECODAGE N EXISTE PLUS, ET NE DOIT PAS REVENIR
// (lot 2.3 du PLAN_DECODEUR_FILM, item 2.3.2 ; REMPLACE `decode_lock_held_test.go`).
//
// # CE QUE CE RATCHET REMPLACE, ET POURQUOI IL EST INVERSE
//
// Jusqu au lot 2.3, `grammar` portait un verrou de PAQUET (`LockProcessDecode`) et le ratchet
// precedent exigeait que TOUT chemin de production enchainant des balayages le tienne. La raison
// etait ecrite dans `decode_gate.go` : les parametres de replication du decodeur de bits etaient
// des VARIABLES DE PAQUET, et le paquet chiffrait lui-meme ce qu un entrelacement coutait — « le
// score d un film passe de 1111 a 1214 selon l ordre d appel ».
//
// CES VARIABLES N EXISTENT PLUS. Le profil de balayage voyage avec le lecteur de bits
// ([grammar.ProfilDeBalayage]), l observateur aussi ([grammar.Observation]), et le ratchet
// `filmdec_package_vars_test.go` mesure ZERO variable de paquet ECRITE. Le verrou n a donc plus
// de raison d etre — et le garder serait pire qu inutile : il SERIALISERAIT deux decodages que
// plus rien n oblige a se suivre.
//
// # LA REGLE, EXACTEMENT
//
// Aucun fichier du module (production ou test) ne declare ni n appelle un verrou de decodage de
// PAQUET. Trois noms sont interdits, et ce sont ceux qui ont existe :
//
//	LockProcessDecode   la porte publique du verrou de paquet ;
//	processDecodeMu     le mutex lui-meme ;
//	decode_gate.go      le fichier qui les portait.
//
// # CE QUI N EST PAS VISE, ET NE DOIT PAS L ETRE
//
// `filmproc.AcquireSolo` est le verrou INTER-PROCESSUS de la machine : il borne le poste a un
// decodage a la fois pour une raison de MEMOIRE, pas de correction. Il reste, et ce ratchet ne le
// regarde pas. De meme, un `sync.Mutex` qui protege autre chose qu un etat de decodage (un cache,
// un compteur de service) n est pas concerne : la regle ne parle que des trois noms ci-dessus.
//
// # LA MUTATION QUI DOIT ROUGIR
//
// Reintroduire `func LockProcessDecode()` dans `grammar`, ou un `var processDecodeMu sync.Mutex`,
// ou recreer `decode_gate.go` : ce test nomme le fichier fautif et refuse.
//
// # POURQUOI LE CODE, ET PAS LES COMMENTAIRES
//
// Le balayage se fait sur l AST, IDENTIFIANTS SEULS. Un ratchet qui grepperait le texte brut
// interdirait aux chroniques de revision (`filmdec/grammar_rev.go`,
// `killcollector/killsource_decoder_rev.go`) de NOMMER ce que leur lot a retire — c est-a-dire
// qu il effacerait la trace du geste qu il protege. Un verrou qui revient revient en code.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
)

// nomsDeVerrouDeDecodage : les identifiants qui ont porte le verrou de PAQUET, et qui sont
// desormais interdits. Le nom du FICHIER en fait partie : le recreer vide serait deja le premier
// pas du retour.
var nomsDeVerrouDeDecodage = map[string]bool{"LockProcessDecode": true, "processDecodeMu": true}

// fichierDeVerrouDeDecodage : le fichier supprime au lot 2.3.
const fichierDeVerrouDeDecodage = "decode_gate.go"

// TestAucunVerrouDeDecodageDePaquet — LE RATCHET INVERSE.
func TestAucunVerrouDeDecodageDePaquet(t *testing.T) {
	racine := apiRootDepuisIci(t)
	var fautes []string
	for _, sous := range []string{"internal", "cmd"} {
		err := filepath.WalkDir(filepath.Join(racine, sous), func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(d.Name(), ".go") {
				return nil
			}
			rel, _ := filepath.Rel(racine, p)
			rel = filepath.ToSlash(rel)
			// CE FICHIER-CI EST LE SEUL A CITER LES NOMS INTERDITS, et c est sa raison d etre.
			if strings.HasSuffix(rel, "archlint/decode_lock_interdit_test.go") {
				return nil
			}
			if d.Name() == fichierDeVerrouDeDecodage {
				fautes = append(fautes, rel+" : le fichier du verrou de decodage est de retour")
				return nil
			}
			// PARSE SANS COMMENTAIRES : seuls les identifiants comptent (cf. l en-tete).
			f, err := parser.ParseFile(token.NewFileSet(), p, nil, parser.SkipObjectResolution)
			if err != nil {
				// Un fichier qui ne parse pas (build tag exotique, source generee) n est pas
				// une faute de ce ratchet — d autres gates le disent.
				return nil //nolint:nilerr // parse impossible = hors perimetre de ce ratchet
			}
			ast.Inspect(f, func(n ast.Node) bool {
				id, ok := n.(*ast.Ident)
				if !ok || !nomsDeVerrouDeDecodage[id.Name] {
					return true
				}
				fautes = append(fautes, rel+" : declare ou appelle "+id.Name)
				return false
			})
			return nil
		})
		if err != nil {
			t.Fatalf("balayage de %s : %v", sous, err)
		}
	}
	if len(fautes) == 0 {
		return
	}
	t.Fatalf("LE VERROU DE DECODAGE DE PAQUET EST DE RETOUR (%d) :\n  %s\n\n"+
		"Il a ete retire au lot 2.3 parce que `grammar` n a plus AUCUNE variable de paquet\n"+
		"ecrite (ratchet `filmdec_package_vars_test.go`) : deux films peuvent se decoder en\n"+
		"parallele, et un verrou de paquet ne ferait que les serialiser. Ce qu il faut a la\n"+
		"place : passer le profil et l observateur par `filmdec.ContexteDeLecture`.\n\n"+
		"Le verrou INTER-PROCESSUS de la machine (`filmproc.AcquireSolo`) n est PAS vise : il\n"+
		"borne la memoire du poste, pas la correction du decodage.",
		len(fautes), strings.Join(fautes, "\n  "))
}
