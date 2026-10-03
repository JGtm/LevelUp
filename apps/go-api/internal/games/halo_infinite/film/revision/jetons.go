package revision

// jetons.go — CE QUI ENTRE DANS L EMPREINTE D UNE SOURCE : SES JETONS, PAS SES OCTETS (lot J3.1 du
// PLAN_SUITE_AUDIT_DECODEUR_FILM_2026-09-25, decision DU-2 (a), constat « faiblesse 1 » de
// `.ai/AUDIT_DECODEUR_FILM_2026-09-24.md`).
//
// # LE DEFAUT QUE CE FICHIER FERME
//
// Jusqu au lot J3.1 l empreinte hachait les OCTETS : une phrase reformulee dans un commentaire
// faisait rougir le gate de sa couche, et le remettre au vert demandait soit une montee de
// revision — qui, pour les faits, rouvre un backlog de redecodage de toutes les lignes de kill en
// base — soit une regeneration « a revision constante » ecrite a la main, c est-a-dire l habitude
// meme de faire taire le gate. Mesure de l audit : 46 % des lignes des couches sont des
// commentaires.
//
// # CE QUI COMPTE, ET CE QUI NE COMPTE PAS
//
//	compte       tout jeton du langage tel que `go/scanner` le rend : mots-cles, identifiants,
//	             operateurs, litteraux (une chaine qui contient `//` ou `/*` reste une chaine,
//	             au caractere pres), et les points-virgules — explicites ou inseres a la fin
//	             d une ligne, ecrits pareil parce qu ils ont le meme sens.
//	compte       les DIRECTIVES `//go:` (`//go:build`, `//go:embed`, `//go:noinline`...) : ce
//	             sont des commentaires pour le lexeur et des instructions pour le compilateur.
//	ne compte pas  les commentaires ordinaires, les blancs et la mise en page entre jetons.
//
// UN COMMENTAIRE QUI CONTIENT UN SAUT DE LIGNE INSERE UN POINT-VIRGULE, exactement comme le fait
// le compilateur (`go/scanner`, regle d insertion) : le flux rendu est celui que le compilateur
// lit, commentaires en moins.

import (
	"fmt"
	"go/scanner"
	"go/token"
	"strings"
)

// prefixeDirective : ce qui distingue une directive du compilateur d un commentaire ordinaire.
const prefixeDirective = "//go:"

// directiveEmbed : la directive dont les motifs designent des fichiers qui entrent dans le binaire,
// donc dans l empreinte (cf. [embarquer]).
const directiveEmbed = "//go:embed "

// jetonsDe rend le flux de jetons d une source Go, commentaires ordinaires retires, directives
// `//go:` conservees — une ligne par jeton, `<nom du jeton> <longueur> <litteral>` — ET les motifs
// de ses directives `//go:embed`, que [sourcesDe] resout en fichiers.
//
// LA LONGUEUR ENCADRE LE LITTERAL, pour la meme raison qu elle encadre chaque fichier : sans elle,
// un litteral qui contiendrait un saut de ligne se confondrait avec deux jetons. Le NOM du jeton
// (`token.Token.String`) et pas sa valeur entiere : la numerotation interne de `go/token` n est
// pas un contrat entre versions de Go, son nom l est.
//
// Une source que le lexeur refuse rend une ERREUR, jamais une empreinte : hacher un fichier qui
// ne se compile pas rendrait un gate vert sur une couche cassee.
func jetonsDe(rel, texte string) (string, []string, error) {
	src := []byte(texte)
	fichier := token.NewFileSet().AddFile(rel, -1, len(src))
	var premiere error
	var s scanner.Scanner
	s.Init(fichier, src, func(pos token.Position, msg string) {
		if premiere == nil {
			premiere = fmt.Errorf("source %s illisible par le lexeur : %s : %s", rel, pos, msg)
		}
	}, scanner.ScanComments)
	var b strings.Builder
	var motifs []string
	for {
		_, tok, lit := s.Scan()
		if tok == token.EOF {
			break
		}
		if tok == token.COMMENT && !strings.HasPrefix(lit, prefixeDirective) {
			continue
		}
		if reste, ok := strings.CutPrefix(lit, directiveEmbed); ok {
			motifs = append(motifs, strings.Fields(reste)...)
		}
		if tok == token.SEMICOLON {
			// `;` ecrit et `\n` insere sont le MEME jeton pour le compilateur.
			lit = ";"
		}
		_, _ = fmt.Fprintf(&b, "%s %d %s\n", tok, len(lit), lit)
	}
	if premiere != nil {
		return "", nil, premiere
	}
	return b.String(), motifs, nil
}
