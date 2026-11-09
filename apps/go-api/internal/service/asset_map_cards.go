// Package service — asset_map_cards.go : grain d'AFFICHAGE des cartes du tiroir d'assets.
package service

import (
	"sort"
	"strings"

	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/i18n"
)

// oneCardPerImage réduit une liste de cartes (image déjà résolue) à UNE entrée par
// visuel, puis la trie pour l'affichage.
//
// Pourquoi ici (lot rr/L4, 2026-09-23) : le dépôt rend une ligne par ASSET (décision D15
// du 2026-09-13, `TestListMapsByTitle_DedupeParAssetIDPasParNom`), mais plusieurs assets
// distincts portent le même visuel — versions republiées d'une carte, copies Forge. Le
// tiroir n'affiche que le nom et l'image : ces assets y donnaient des cartes identiques
// (mesuré sur le catalogue réel le 2026-09-23 : 157 cartes pour 93 images, « Solution »
// en trois exemplaires). La clé du visuel est l'URL d'image RÉSOLUE, connue seulement
// après la résolution faite par ListMaps — d'où ce regroupement dans le service, et non
// dans le dépôt (grain de données) ni dans le client (qui garde son filet par id).
// Aucune règle propre à un titre : seule l'URL rendue par l'adapter du titre compte.
//
// Représentant d'un visuel, par ordre TOTAL (résultat indépendant de l'ordre reçu), dans la
// langue `loc` de la requête :
//  1. un libellé TRADUIT (non vide et différent du nom canonique) — décision utilisateur Q27
//     du 2026-09-23 : Starboard/Tribord s'affiche « Tribord » en français ;
//  2. puis un libellé non vide dans la langue ;
//  3. puis le plus petit ID (puis les textes de toutes les langues, pour un départage complet).
//
// Une entrée sans image (URL vide) n'a pas d'identité visuelle : elle n'est jamais
// fusionnée avec une autre (ListMaps n'en produit pas ; la garde protège un appelant
// futur contre la fusion silencieuse de cartes sans rapport).
//
// Le nom CANONIQUE est le nom du jeu dans la langue de repli (`i18n.Fallback`) : clé de
// l'image et de l'ordre. Tri de sortie : nom canonique sans casse, puis nom canonique, puis
// ID (ordre identique dans toutes les langues). La fonction ne réordonne ni ne modifie la
// tranche reçue : elle rend une tranche neuve. (Cela ne rend pas immuable la tranche du
// dépôt en amont : c'est l'affaire de l'appelant.)
func oneCardPerImage(items []canonical.AssetMeta, loc i18n.Locale) []canonical.AssetMeta {
	bestByImage := make(map[string]int, len(items))
	out := make([]canonical.AssetMeta, 0, len(items))
	for _, item := range items {
		if item.ImageURL == "" {
			out = append(out, item)
			continue
		}
		idx, seen := bestByImage[item.ImageURL]
		if !seen {
			bestByImage[item.ImageURL] = len(out)
			out = append(out, item)
			continue
		}
		if representsBetter(item, out[idx], loc) {
			out[idx] = item
		}
	}
	sort.Slice(out, func(i, j int) bool { return displayedBefore(out[i], out[j]) })
	return out
}

// labelRank classe le libellé d'une carte dans la langue `loc` : 0 = traduit, 1 = présent mais
// égal au nom canonique, 2 = absent.
func labelRank(m canonical.AssetMeta, loc i18n.Locale) int {
	switch {
	case m.Names[loc] != "" && m.Names[loc] != m.CanonicalName():
		return 0
	case m.Names[loc] != "":
		return 1
	default:
		return 2
	}
}

// representsBetter dit si a doit remplacer b comme représentant d'un même visuel, dans la
// langue `loc`.
func representsBetter(a, b canonical.AssetMeta, loc i18n.Locale) bool {
	if ra, rb := labelRank(a, loc), labelRank(b, loc); ra != rb {
		return ra < rb
	}
	if a.ID != b.ID {
		return a.ID < b.ID
	}
	return a.Names.Compare(b.Names) < 0
}

// displayedBefore est l'ordre d'affichage du tiroir : nom canonique sans casse, puis nom
// canonique exact, puis ID.
func displayedBefore(a, b canonical.AssetMeta) bool {
	ca, cb := a.CanonicalName(), b.CanonicalName()
	if la, lb := strings.ToLower(ca), strings.ToLower(cb); la != lb {
		return la < lb
	}
	if ca != cb {
		return ca < cb
	}
	return a.ID < b.ID
}
