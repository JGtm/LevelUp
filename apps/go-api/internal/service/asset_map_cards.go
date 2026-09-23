// Package service — asset_map_cards.go : grain d'AFFICHAGE des cartes du tiroir d'assets.
package service

import (
	"sort"
	"strings"

	"levelup/go-api/internal/games/canonical"
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
// Représentant d'un visuel, par ordre TOTAL (résultat indépendant de l'ordre reçu) :
//  1. un libellé FR TRADUIT (NameFR non vide et différent de NameEN) — décision
//     utilisateur Q27 du 2026-09-23 : Starboard/Tribord s'affiche « Tribord » ;
//  2. puis un libellé FR non vide ;
//  3. puis le plus petit ID (puis NameFR, puis NameEN, pour un départage complet).
//
// Une entrée sans image (URL vide) n'a pas d'identité visuelle : elle n'est jamais
// fusionnée avec une autre (ListMaps n'en produit pas ; la garde protège un appelant
// futur contre la fusion silencieuse de cartes sans rapport).
//
// Tri de sortie : NameEN sans casse, puis NameEN, puis ID. La tranche reçue n'est pas
// modifiée (elle peut être l'instantané en mémoire du StaticAssetMetaRepo).
func oneCardPerImage(items []canonical.AssetMeta) []canonical.AssetMeta {
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
		if representsBetter(item, out[idx]) {
			out[idx] = item
		}
	}
	sort.Slice(out, func(i, j int) bool { return displayedBefore(out[i], out[j]) })
	return out
}

// frLabelRank classe le libellé FR d'une carte : 0 = traduit, 1 = présent mais égal au
// nom anglais, 2 = absent.
func frLabelRank(m canonical.AssetMeta) int {
	switch {
	case m.NameFR != "" && m.NameFR != m.NameEN:
		return 0
	case m.NameFR != "":
		return 1
	default:
		return 2
	}
}

// representsBetter dit si a doit remplacer b comme représentant d'un même visuel.
func representsBetter(a, b canonical.AssetMeta) bool {
	if ra, rb := frLabelRank(a), frLabelRank(b); ra != rb {
		return ra < rb
	}
	if a.ID != b.ID {
		return a.ID < b.ID
	}
	if a.NameFR != b.NameFR {
		return a.NameFR < b.NameFR
	}
	return a.NameEN < b.NameEN
}

// displayedBefore est l'ordre d'affichage du tiroir : nom anglais sans casse, puis nom
// anglais exact, puis ID.
func displayedBefore(a, b canonical.AssetMeta) bool {
	if la, lb := strings.ToLower(a.NameEN), strings.ToLower(b.NameEN); la != lb {
		return la < lb
	}
	if a.NameEN != b.NameEN {
		return a.NameEN < b.NameEN
	}
	return a.ID < b.ID
}
