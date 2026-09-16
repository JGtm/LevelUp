// Package revision porte LE MECANISME D EMPREINTE DE SOURCES partage par les revisions de
// couche du decodeur de film (lot 2.6.0 du PLAN_DECODEUR_FILM_2026-09-13).
//
// # POURQUOI CE PAQUET EXISTE, ET POURQUOI AVANT 2.6.1
//
// Deux gates du depot font AUJOURD HUI la meme chose, chacun avec sa copie du code :
//
//	filmdec/grammar_rev_fingerprint_test.go      hache filmdec/ + killsource/ + objectives/
//	sync/killcollector/decoder_rev_fingerprint_test.go  hache killsource/
//
// La decision V15 (11) du plan pose QUATRE revisions, une par couche (`source`, `profile`,
// `grammar`, `facts`). Les ecrire a l identique ferait QUATRE copies du meme motif, alors que
// CLAUDE.md regle 6 impose de centraliser des la TROISIEME — et d y poser un garde-rail, sans
// quoi la factorisation re-diverge. Ce paquet est cette centralisation ; le garde-rail est
// `archlint/no_ad_hoc_source_fingerprint_test.go`.
//
// # CE QUE CE PAQUET NE FAIT PAS
//
// Il ne porte AUCUNE revision, AUCUNE racine, AUCUN golden : ce sont les couches qui les
// declarent. Il ne connait pas non plus `testing` — un paquet de production qui declarerait un
// drapeau de test le poserait sur le binaire du serveur. La porte de regeneration ([Porte])
// DECIDE a partir d un drapeau et d une variable que l appelant lui passe ; c est le fichier de
// test de la couche qui declare `flag.Bool`.
//
// # POURQUOI ICI, ET PAS SOUS `film/internal/`
//
// Le paquet doit etre importable par les futurs `film/internal/{source,profile,grammar,facts}`
// ET par `sync/killcollector` tant que la constante de revision des faits y vit (elle descend
// en `facts/` au lot 2.6.1, note de preparation §3.1). Sous `film/internal/`, le compilateur
// fermerait la porte a `killcollector`. Il est donc classe `horsCoucheFilm` dans
// `archlint/film_layers_deps_test.go` : outillage de revision, ni decodage ni publication — il
// ne lit aucun octet de film, il hache des octets de SOURCE.
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
	"sort"
	"strings"
)

// Cadre : LA FORME DES OCTETS SOUMIS AU HACHAGE — c est-a-dire le CONTRAT de l empreinte.
//
// Deux cadres coexistent le temps du lot 2.6.1, et ce n est pas une preference de style : deux
// empreintes calculees par deux cadres differents ne se comparent pas, donc changer de cadre
// sur une couche en place renumeroterait sa revision pour rien.
type Cadre int

const (
	// CadreRacine — LE CONTRAT COURANT : le chemin hache a cote du contenu est RELATIF A LA
	// RACINE de la couche.
	//
	// ARBITRAGE (lot 2.6.0, pour le pas 5). Les deux mecanismes existants ne hachent pas le meme
	// chemin :
	//
	//	killsource   `decode.go`           (relatif a la racine)
	//	grammaire    `filmdec/decode.go`   (relatif au PARENT de la racine : le nom du dossier
	//	                                    de racine est prefixe au chemin)
	//
	// Le pas 5 deplace les couches EN BLOC (`git mv filmdec film/internal/grammar`). Sous le
	// cadre de la grammaire, ce deplacement change les 141 chemins haches et donc l empreinte,
	// alors qu AUCUN octet de grammaire n a bouge : il faudrait soit monter la revision pour
	// rien — ce qui, pour `facts`, rouvre un backlog de redecodage (V15 (16)) — soit regenerer
	// le golden sur la branche « revision inchangee, empreinte differente », c est-a-dire faire
	// taire le ratchet dans le cas precis pour lequel il existe.
	//
	// Le chemin relatif A LA RACINE survit au `git mv` du dossier entier et mord toujours sur ce
	// que le prefixe attrapait : un fichier renomme DANS la couche. C est donc le contrat
	// courant. Ce que le cadre perd — deux racines qui portent le meme nom de fichier ne sont
	// plus distinguees par leur dossier — est sans effet : les racines sont hachees dans l ordre
	// ou l appelant les donne, et la longueur du contenu encadre chaque fichier (voir
	// [Calculer]).
	CadreRacine Cadre = iota

	// CadreHeriteGrammaire — LE CONTRAT HERITE de `grammar_rev_fingerprint_test.go` : le chemin
	// hache est prefixe du nom du dossier de racine, et l encadrement est `chemin`, octet NUL,
	// contenu (sans longueur).
	//
	// KILL-SWITCH, ET IL EST DATE. Bascule du defaut : jamais — ce cadre n est le defaut de
	// personne, il ne sert qu a PROUVER (`equivalence_test.go`) que le mecanisme central rend
	// EXACTEMENT l empreinte figee dans `filmdec/testdata/grammar_rev.golden`, donc qu au lot
	// 2.6.1 `grammar.Rev` heritera sans renumerotation gratuite. Cible de retrait : lot 2.6.1,
	// dans le commit qui fait passer `grammar` a [CadreRacine] avec la montee de revision qui
	// l accompagne. Critere mesurable : plus aucun appelant de [EmpreinteHeritee] hors de son
	// propre test de non-regression.
	CadreHeriteGrammaire
)

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

// Empreinte hache les sources `.go` de production des racines, sous le contrat courant
// ([CadreRacine]).
//
// `exclure` recoit le chemin de chaque fichier RELATIF A SA RACINE, en slash, et rend vrai pour
// les fichiers a ecarter — nominalement le fichier qui PORTE la revision, qui decrit la couche
// sans en faire partie. Nil n exclut rien.
//
// `valeursAmont` porte la VALEUR des revisions dont la couche depend (decision V15 (12) :
// `facts.Rev` hache la valeur de `grammar.Rev`). Elles sont hachees EN TETE, avant toute source.
func Empreinte(racines []string, exclure func(rel string) bool, valeursAmont ...string) (string, error) {
	r, err := Calculer(CadreRacine, racines, exclure, valeursAmont...)
	return r.Empreinte, err
}

// EmpreinteHeritee hache les memes sources sous le contrat [CadreHeriteGrammaire].
//
// A N APPELER QUE depuis la preuve d equivalence : voir la note de retrait de la constante.
func EmpreinteHeritee(racines []string, exclure func(rel string) bool, valeursAmont ...string) (string, error) {
	r, err := Calculer(CadreHeriteGrammaire, racines, exclure, valeursAmont...)
	return r.Empreinte, err
}

// Calculer rend l empreinte ET le nombre de fichiers, sous le cadre demande.
//
// # CE QUI ENTRE DANS LE HACHAGE, DANS L ORDRE
//
//  1. Les valeurs amont, encadrees `amont:<longueur>\n<valeur>\n`. AUCUN octet n est ecrit
//     quand il n y en a pas : c est cette propriete qui rend l empreinte d une couche SANS
//     amont identique a celle des deux mecanismes d avant, donc l heritage sans
//     renumerotation. Une valeur amont ne peut pas se confondre avec un fichier : les chemins
//     haches finissent tous par `.go`.
//  2. Pour chaque racine, DANS L ORDRE DONNE, ses sources triees par chemin relatif.
//
// # CE QUI EST ECARTE
//
// Les `_test.go` et tout ce qui vit sous un `testdata/` : un test ajoute ne change pas une
// ligne produite, et exiger une montee de revision pour lui viderait la revision de son sens.
// Les fins de ligne sont normalisees en LF — sans quoi un checkout mal configure rendrait le
// gate vert en CI et rouge sur le poste, pour une raison etrangere a la couche.
//
// # CE QUE CE MECANISME NE FAIT PAS, ET C EST ASSUME
//
// Il ne distingue pas un changement de decodage d une reformulation de commentaire : le hachage
// porte sur les OCTETS. Un garde-rail qui ne mordrait que sur le « significatif » devrait
// comprendre le decodeur — il rendrait des faux negatifs, c est-a-dire le defaut meme qu il
// existe pour fermer. Un faux positif coute une ligne a mettre a jour.
func Calculer(cadre Cadre, racines []string, exclure func(rel string) bool, valeursAmont ...string) (Resultat, error) {
	h := sha256.New()
	for _, v := range valeursAmont {
		_, _ = fmt.Fprintf(h, "amont:%d\n%s\n", len(v), v)
	}
	total := 0
	for _, racine := range racines {
		lus, err := sourcesDe(racine, exclure)
		if err != nil {
			return Resultat{}, err
		}
		if len(lus) == 0 {
			return Resultat{}, fmt.Errorf("%w : %s", ErrRacineSansSource, racine)
		}
		ecrireSources(h, cadre, racine, lus)
		total += len(lus)
	}
	return Resultat{Empreinte: hex.EncodeToString(h.Sum(nil)), Fichiers: total}, nil
}

// sourceLue : un fichier retenu, son chemin relatif a sa racine (en slash) et son texte
// normalise.
type sourceLue struct {
	rel   string
	texte string
}

// sourcesDe rend les sources `.go` de production d une racine, triees par chemin relatif.
func sourcesDe(racine string, exclure func(rel string) bool) ([]sourceLue, error) {
	var lus []sourceLue
	err := filepath.WalkDir(racine, func(chemin string, d fs.DirEntry, errMarche error) error {
		if errMarche != nil {
			return errMarche
		}
		if d.IsDir() {
			if d.Name() == "testdata" {
				return filepath.SkipDir
			}
			return nil
		}
		nom := d.Name()
		if !strings.HasSuffix(nom, ".go") || strings.HasSuffix(nom, "_test.go") {
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
		blob, errLire := os.ReadFile(chemin) //nolint:gosec // chemin construit depuis une racine fournie par l appelant
		if errLire != nil {
			return errLire
		}
		lus = append(lus, sourceLue{rel: rel, texte: strings.ReplaceAll(string(blob), "\r\n", "\n")})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(lus, func(i, j int) bool { return lus[i].rel < lus[j].rel })
	return lus, nil
}

// ecrireSources verse les sources d une racine dans le hachage, selon le cadre.
//
// Le tri se fait sur le chemin relatif a la racine dans les DEUX cadres : le prefixe du cadre
// herite etant constant pour une racine donnee, il ne change aucun ordre relatif.
func ecrireSources(h hash.Hash, cadre Cadre, racine string, lus []sourceLue) {
	for _, f := range lus {
		if cadre == CadreHeriteGrammaire {
			_, _ = h.Write([]byte(path.Join(filepath.Base(racine), f.rel)))
			_, _ = h.Write([]byte{0})
			_, _ = h.Write([]byte(f.texte))
			continue
		}
		// La longueur encadre le contenu : sans elle, deux decoupages differents des memes
		// octets rendraient la meme empreinte.
		_, _ = fmt.Fprintf(h, "%s\n%d\n", f.rel, len(f.texte))
		_, _ = h.Write([]byte(f.texte))
	}
}
