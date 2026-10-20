// Package revision porte LE MECANISME D EMPREINTE DE SOURCES partage par les revisions de
// couche du decodeur de film (lot 2.6.0 du PLAN_DECODEUR_FILM_2026-09-13).
//
// # POURQUOI CE PAQUET EXISTE
//
// Deux gates du depot faisaient la meme chose, chacun avec sa copie du code : celui de la
// grammaire (`filmdec/grammar_rev_fingerprint_test.go`, qui hachait cinq racines) et celui des
// faits (`sync/killcollector/decoder_rev_fingerprint_test.go`, qui hachait `killsource/`).
//
// La decision V15 (11) du plan pose QUATRE revisions, une par couche (`source`, `profile`,
// `grammar`, `facts`). Les ecrire a l identique aurait fait QUATRE copies du meme motif, alors
// que CLAUDE.md regle 6 impose de centraliser des la TROISIEME — et d y poser un garde-rail, sans
// quoi la factorisation re-diverge. Ce paquet est cette centralisation ; le garde-rail est
// `archlint/no_ad_hoc_source_fingerprint_test.go`, dont l allowlist est VIDE depuis que la
// grammaire a herite (lot 2.6.1) : les quatre couches passent par ici, et aucune copie ne reste.
//
// # CE QUE CE PAQUET NE FAIT PAS
//
// Il ne porte AUCUNE revision ni AUCUN golden : ce sont les couches qui les declarent. Il porte,
// depuis le lot J3.2, la LISTE des couches revisees — leurs noms et leurs racines, pas leurs
// valeurs ([CouchesRevisees]) — parce que la fermeture des imports de chacune doit s arreter a
// toutes les autres, et que cinq copies de cette liste dans cinq gates re-divergeraient. Il ne connait pas non plus `testing` — un paquet de production qui declarerait un
// drapeau de test le poserait sur le binaire du serveur. La porte de regeneration ([Porte])
// DECIDE a partir d un drapeau et d une variable que l appelant lui passe ; c est le fichier de
// test de la couche qui declare `flag.Bool`.
//
// # POURQUOI ICI, ET PAS SOUS `film/internal/`
//
// Il etait hors de `film/internal/` pour rester importable par `sync/killcollector`, qui portait
// la constante de revision des faits. CETTE RAISON A DISPARU au volet facts du lot 2.6.1 : la
// constante a descendu en `film/internal/facts/rev.go`, et depuis le volet grammaire les seuls
// importateurs de ce paquet sont les quatre gates de couche, tous sous `film/`. Le paquet
// POURRAIT donc passer sous `film/internal/` ; il ne le fait pas dans ce lot, qui ne deplace
// rien (D5 (2.6), consigne au §4 du plan).
//
// Il est classe `horsCoucheFilm` dans `archlint/film_layers_deps_test.go` : outillage de
// revision, ni decodage ni publication — il ne lit aucun octet de film, il hache des octets de
// SOURCE.
package revision

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
)

// LE CADRE DU HACHAGE — c est-a-dire le CONTRAT de l empreinte — EST UNIQUE : le chemin hache a
// cote du contenu est RELATIF A LA RACINE de la couche.
//
// ARBITRAGE (lot 2.6.0, pour le pas 5). Les deux mecanismes d avant ne hachaient pas le meme
// chemin :
//
//	killsource   `decode.go`           (relatif a la racine)
//	grammaire    `filmdec/decode.go`   (relatif au PARENT de la racine : le nom du dossier de
//	                                    racine etait prefixe au chemin)
//
// Le pas 5 deplace les couches EN BLOC (`git mv filmdec film/internal/grammar`). Sous le cadre de
// la grammaire, ce deplacement changeait les 141 chemins haches et donc l empreinte, alors
// qu AUCUN octet de grammaire n avait bouge : il aurait fallu soit monter la revision pour rien —
// ce qui, pour `facts`, rouvre un backlog de redecodage (V15 (16)) — soit regenerer le golden sur
// la branche « revision inchangee, empreinte differente », c est-a-dire faire taire le ratchet
// dans le cas precis pour lequel il existe.
//
// Le chemin relatif A LA RACINE survit au `git mv` du dossier entier et mord toujours sur ce que
// le prefixe attrapait : un fichier renomme DANS la couche. Ce que le cadre perd — deux racines
// qui portent le meme nom de fichier ne sont plus distinguees par leur dossier — est sans effet :
// les racines sont hachees dans l ordre ou l appelant les donne, et la longueur du contenu
// encadre chaque fichier (voir [empreinteur]).
//
// LE CADRE HERITE DE LA GRAMMAIRE A ETE SUPPRIME AU LOT 2.6.1, dans le commit qui a fait heriter
// `grammar.Rev` — c etait sa cible de retrait datee, et son critere mesurable est tenu : plus
// aucun appelant. Ce qu il servait a prouver est acquis : l heritage de l OUTILLAGE ne coute
// aucune renumerotation (l empreinte heritee des cinq racines egalait `7994ce19…`, figee au rang
// `.38`). Ce qui a fait monter le rang au `.39` est le PERIMETRE, pas le cadre.

// ErrRacineSansSource : une racine ne porte AUCUNE source `.go` de production.
//
// C est une ERREUR et pas un zero : une couche dont l arborescence a bouge doit echouer
// bruyamment plutot que hacher du vide et rendre un gate vert qui ne garde plus rien (meme
// doctrine que les planchers de balayage des ratchets d `archlint`).
var ErrRacineSansSource = errors.New("racine sans source .go de production")

// Resultat : l empreinte et le nombre de fichiers qui l ont produite.
//
// Le COMPTE fait partie du resultat parce que les messages d echec le citent (« 150 fichiers ») :
// c est lui qui distingue « la grammaire a change » de « le balayage n a rien trouve ».
type Resultat struct {
	// Empreinte : le SHA-256 en hexadecimal minuscule.
	Empreinte string
	// Fichiers : le nombre de sources hachees, toutes racines confondues.
	Fichiers int
}

// empreinteur : UN hachage en cours, et le compte des fichiers qui y sont entres. C est LE cadre
// de l empreinte, ecrit une fois : [Module.CalculerCouche] est son seul point d entree de
// production, et chaque couche cite le compte dans ses messages d echec.
//
// `exclure` (cf. [empreinteur.arbre]) recoit le chemin de chaque fichier RELATIF A SA RACINE, en
// slash, et rend vrai pour les fichiers a ecarter — nominalement le fichier qui PORTE la
// revision, qui decrit la couche sans en faire partie. Nil n exclut rien.
//
// # CE QUI ENTRE DANS LE HACHAGE, DANS L ORDRE
//
//  1. Les valeurs amont, encadrees `amont:<longueur>\n<valeur>\n`. AUCUN octet n est ecrit
//     quand il n y en a pas : c est cette propriete qui rend l empreinte d une couche SANS
//     amont identique a celle des deux mecanismes d avant, donc l heritage sans
//     renumerotation. Une valeur amont ne peut pas se confondre avec un fichier : les chemins
//     haches finissent tous par `.go`, ou commencent par `embed:`.
//  2. Les racines, DANS L ORDRE DONNE, chacune ses sources triees par nom hache.
//
// # CE QUI EST ECARTE
//
// Les `_test.go` et tout ce qui vit sous un `testdata/` : un test ajoute ne change pas une
// ligne produite, et exiger une montee de revision pour lui viderait la revision de son sens.
// Les fins de ligne sont normalisees en LF — sans quoi un checkout mal configure rendrait le
// gate vert en CI et rouge sur le poste, pour une raison etrangere a la couche.
//
// LES COMMENTAIRES ORDINAIRES ET LA MISE EN PAGE SONT ECARTES DEPUIS LE LOT J3.1 (2026-09-26,
// decision DU-2 (a)) : le contenu d un fichier est son FLUX DE JETONS ([jetonsDe]), directives
// `//go:` comprises, et les fichiers qu il EMBARQUE entrent a cote de lui depuis le lot J3.2
// ([collecte.embarquer]). Le cadre, lui, n a pas change — nom relatif et longueur du contenu.
//
// # CE QUE CE MECANISME NE FAIT PAS, ET C EST ASSUME
//
// Il ne distingue pas un changement de decodage d un renommage de variable locale : le hachage
// porte sur les JETONS. Un garde-rail qui ne mordrait que sur le « significatif » devrait
// comprendre le decodeur — il rendrait des faux negatifs, c est-a-dire le defaut meme qu il
// existe pour fermer. Un faux positif coute une ligne a mettre a jour.
type empreinteur struct {
	h        hash.Hash
	fichiers int
}

// nouvelEmpreinteur ouvre un hachage et y verse les valeurs amont, EN TETE et dans l ordre donne.
// AUCUN OCTET n est ecrit quand il n y en a pas (cf. [empreinteur]).
func nouvelEmpreinteur(valeursAmont []string) *empreinteur {
	h := sha256.New()
	for _, v := range valeursAmont {
		_, _ = fmt.Fprintf(h, "amont:%d\n%s\n", len(v), v)
	}
	return &empreinteur{h: h}
}

// arbre verse toute l arborescence `racine`, chemins relatifs a la racine. Une racine sans source
// de production est une ERREUR ([ErrRacineSansSource]).
func (e *empreinteur) arbre(racine string, exclure func(rel string) bool) error {
	lus, err := sourcesDe(racine, "", true, exclure)
	if err != nil {
		return err
	}
	if len(lus) == 0 {
		return fmt.Errorf("%w : %s", ErrRacineSansSource, racine)
	}
	e.ecrire(lus)
	return nil
}

// paquet verse UN paquet importe — le dossier `rel` du module, sans ses sous-dossiers — chemins
// prefixes par `rel` : c est le paquet qui nomme ce qu il apporte.
func (e *empreinteur) paquet(racineModule, rel string) error {
	lus, err := sourcesDe(filepath.Join(racineModule, filepath.FromSlash(rel)), rel, false, nil)
	if err != nil {
		return err
	}
	if len(lus) == 0 {
		return fmt.Errorf("%w : %s", ErrRacineSansSource, rel)
	}
	e.ecrire(lus)
	return nil
}

// resultat ferme le hachage.
func (e *empreinteur) resultat() Resultat {
	return Resultat{Empreinte: hex.EncodeToString(e.h.Sum(nil)), Fichiers: e.fichiers}
}

// ecrire verse des sources lues dans le hachage.
//
// LA RACINE N EST PAS HACHEE, et c est le cadre lui-meme : seul le chemin RELATIF entre dans les
// octets haches, de sorte qu un `git mv` de la couche entiere ne coute rien (voir l en-tete de ce
// fichier).
func (e *empreinteur) ecrire(lus []sourceLue) {
	for _, f := range lus {
		// La longueur encadre le contenu : sans elle, deux decoupages differents des memes
		// octets rendraient la meme empreinte.
		_, _ = fmt.Fprintf(e.h, "%s\n%d\n", f.rel, len(f.texte))
		_, _ = e.h.Write([]byte(f.texte))
	}
	e.fichiers += len(lus)
}

// sourceLue : un fichier retenu, son nom HACHE (chemin relatif, en slash, eventuellement prefixe)
// et son contenu normalise — le flux de jetons d une source, les octets d un fichier embarque.
type sourceLue struct {
	rel   string
	texte string
}

// sourcesDe rend les sources `.go` de production de `racine` — et les fichiers qu elles
// EMBARQUENT — triees par nom hache. `recursif` faux s arrete au dossier lui-meme ; `prefixe`
// non vide prefixe chaque nom hache ; `exclure` recoit le chemin relatif a la racine.
func sourcesDe(racine, prefixe string, recursif bool, exclure func(rel string) bool) ([]sourceLue, error) {
	c := &collecte{racine: racine, prefixe: prefixe, deja: map[string]bool{}}
	err := filepath.WalkDir(racine, func(chemin string, d fs.DirEntry, errMarche error) error {
		if errMarche != nil {
			return errMarche
		}
		if d.IsDir() {
			if d.Name() == "testdata" || (!recursif && chemin != racine) {
				return filepath.SkipDir
			}
			return nil
		}
		if !estSourceDeProduction(d.Name()) {
			return nil
		}
		rel, errRel := filepath.Rel(racine, chemin)
		if errRel != nil {
			return errRel
		}
		rel = filepath.ToSlash(rel)
		if exclure != nil && exclure(rel) {
			return nil
		}
		source, motifs, err := lireSource(chemin, rel)
		if err != nil {
			return err
		}
		source.rel = nomHache(prefixe, rel)
		c.lus = append(c.lus, source)
		return c.embarquer(chemin, motifs)
	})
	if err != nil {
		return nil, err
	}
	slices.SortFunc(c.lus, func(a, b sourceLue) int { return strings.Compare(a.rel, b.rel) })
	return c.lus, nil
}

// lireSource lit une source et rend son flux de jetons et ses motifs `//go:embed`.
func lireSource(chemin, rel string) (sourceLue, []string, error) {
	blob, err := os.ReadFile(chemin) //nolint:gosec // chemin construit depuis une racine fournie par l appelant
	if err != nil {
		return sourceLue{}, nil, err
	}
	jetons, motifs, err := jetonsDe(rel, strings.ReplaceAll(string(blob), "\r\n", "\n"))
	if err != nil {
		return sourceLue{}, nil, err
	}
	return sourceLue{texte: jetons}, motifs, nil
}

// nomHache rend le nom sous lequel un fichier entre dans le hachage.
func nomHache(prefixe, rel string) string {
	if prefixe == "" {
		return rel
	}
	return path.Join(prefixe, rel)
}
