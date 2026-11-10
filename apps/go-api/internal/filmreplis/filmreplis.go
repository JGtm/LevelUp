// Package filmreplis — LA PART DE L'HOTE DU REGISTRE DES REPLIS du decodeur de film (ADR 0034
// D-10).
//
// Le registre de la brique (`decfilm.Table`) declare chaque repli : son nom, ce qu'il decide, son
// critere de retrait, son compteur. Il ne nomme que des fichiers de la brique, relatifs a son
// module. Un repli qui s'execute aussi, ou seulement, dans LevelUp (le collecteur des morts, le
// constructeur de rejeu, la regle du decor) y est marque `CompleteeParLHote` : ses sites dans go-api,
// et les appelants d'un outil hors production, sont declares ICI. Aucun nom de repli ne vit dans
// ce paquet : un nom se compte sur les compteurs de la brique et se publie dans son document.
//
// Les garde-rails du registre (`archlint/no_unregistered_fallback*_test.go`,
// `archlint/fallback_versement_test.go`) lisent les sites par [Sites] et les appelants par
// [Appelants], qui ouvrent chaque chemin depuis la racine de SON registre, et rejouent
// [Verifier].
package filmreplis

import (
	"fmt"
	"sort"
	"strings"

	"github.com/JGtm/halo-infinite-film-decoder/decfilm"
)

// Racines des deux registres, relatives a la racine du depot : un site de la brique est relatif a
// son module, un site de l'hote au module go-api.
const (
	racineDeLaBrique = "libs/film/"
	racineDeLHote    = "apps/go-api/"
)

// Sites rend les sites du repli `nom`, chemins relatifs a la racine du depot : ceux que la brique
// declare (`deLaBrique`, relatifs a son module), puis ceux que l'hote declare.
func Sites(nom decfilm.Nom, deLaBrique []decfilm.Site) []decfilm.Site {
	out := make([]decfilm.Site, 0, len(deLaBrique)+len(sitesDeLHote[nom]))
	for _, s := range deLaBrique {
		s.Fichier = racineDeLaBrique + s.Fichier
		out = append(out, s)
	}
	for _, s := range sitesDeLHote[nom] {
		s.Fichier = racineDeLHote + s.Fichier
		out = append(out, s)
	}
	return out
}

// Appelants rend les repertoires, relatifs a la racine du depot, ou l'outil hors production du
// repli `nom` est appele : ceux que la brique declare (`deLaBrique`), puis ceux de l'hote.
func Appelants(nom decfilm.Nom, deLaBrique []string) []string {
	out := make([]string, 0, len(deLaBrique)+len(appelantsDeLHote[nom]))
	for _, a := range deLaBrique {
		out = append(out, racineDeLaBrique+a)
	}
	for _, a := range appelantsDeLHote[nom] {
		out = append(out, racineDeLHote+a)
	}
	return out
}

// Verifier rend les defauts de la part de l'hote, nil quand elle tient : un complement pour un nom
// absent du registre de la brique, une entree marquee `CompleteeParLHote` que l'hote ne complete
// pas, un complement pour une entree non marquee, un site incomplet, des appelants pour une entree
// qui n'est pas un outil hors production.
func Verifier() []string {
	var pbs []string
	marquees := map[decfilm.Nom]bool{}
	horsProduction := map[decfilm.Nom]bool{}
	for _, r := range decfilm.Table() {
		marquees[r.Nom] = r.CompleteeParLHote
		horsProduction[r.Nom] = r.HorsProduction != nil
		if r.CompleteeParLHote && len(sitesDeLHote[r.Nom]) == 0 && len(appelantsDeLHote[r.Nom]) == 0 {
			pbs = append(pbs, fmt.Sprintf("%s : marquee CompleteeParLHote, mais l hote ne declare ni site ni appelant", r.Nom))
		}
	}
	for nom, sites := range sitesDeLHote {
		pbs = append(pbs, verifierComplement(nom, marquees)...)
		for i, s := range sites {
			if strings.TrimSpace(s.Fichier) == "" || strings.TrimSpace(s.Ancre) == "" {
				pbs = append(pbs, fmt.Sprintf("%s : site de l hote #%d incomplet (fichier %q, ancre %q)", nom, i, s.Fichier, s.Ancre))
			}
		}
	}
	for nom := range appelantsDeLHote {
		pbs = append(pbs, verifierComplement(nom, marquees)...)
		if !horsProduction[nom] {
			pbs = append(pbs, fmt.Sprintf("%s : appelants declares par l hote pour une entree qui n est pas un outil hors production", nom))
		}
	}
	sort.Strings(pbs)
	return pbs
}

// verifierComplement : le nom complete par l'hote existe au registre de la brique et y est marque.
func verifierComplement(nom decfilm.Nom, marquees map[decfilm.Nom]bool) []string {
	marquee, connu := marquees[nom]
	switch {
	case !connu:
		return []string{fmt.Sprintf("%s : complete par l hote, mais absent du registre de la brique", nom)}
	case !marquee:
		return []string{fmt.Sprintf("%s : complete par l hote, mais non marque CompleteeParLHote dans la brique", nom)}
	}
	return nil
}
