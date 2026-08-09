package main

// assets.go — LE CATALOGUE DE TAGS DE DEGAT SERT DE VOCABULAIRE.
//
// POURQUOI CETTE SOURCE, ET PAS UNE DE PLUS. `internal/.../film/damagetag` porte, deja versionne
// et deja embarque, le NOM D ASSET INTERNE de chaque source de degat :
// `sb_010_grn_un_lightninggrenade`, `sb_008_exp_single_small_hardlight`,
// `sb_010_veh_un_falcongrenadelauncher`. Ce sont les mots du jeu pour les memes objets que le
// kill feed dessine — et aucune moisson ne les rendait, parce qu ils ne sont ni dans le binaire
// ni dans le vocabulaire commercial.
//
// CE QU ELLE A RENDU. `killfeed_lightning_grenade` (index 48 — la grenade Dynamo s appelle
// « lightning grenade » en interne, ce qui explique pourquoi `dynamo_grenade` echouait) et
// `killfeed_gatling_mortar` (index 14). Elle retrouve aussi `killfeed_falcon_grenade_launcher`
// de facon INDEPENDANTE, ce qui vaut controle : la source et l oeil humain concordent.
//
// LA TRANSFORMATION. Les noms du catalogue sont COLLES (`lightninggrenade`) la ou le kill feed
// separe (`lightning_grenade`). On genere donc toutes les insertions d un ou deux underscores,
// sur le nom entier et sur chacune de ses queues — le prefixe `sb_010_grn_un_` est un classement,
// pas un nom. Aucune heuristique de dictionnaire : quelques dizaines de milliers de formes, et
// seule une egalite exacte sort.

import (
	"regexp"
	"strings"

	"levelup/go-api/internal/games/halo_infinite/film/damagetag"
)

var reAssetName = regexp.MustCompile(`[a-z][a-z0-9_]{4,}`)

const maxAssetTail = 28

// assetVocabulary rend les formes derivees des noms d assets du catalogue de degats.
func assetVocabulary() []string {
	noms := map[string]bool{}
	for _, id := range damagetag.IDs() {
		l, ok := damagetag.Lookup(id)
		if !ok {
			continue
		}
		for _, champ := range []string{l.Detail, l.Name, l.Reserve} {
			for _, m := range reAssetName.FindAllString(strings.ToLower(champ), -1) {
				if strings.Contains(m, "_") {
					noms[m] = true
				}
			}
		}
	}
	seen := map[string]bool{}
	var out []string
	for nom := range noms {
		for _, q := range assetTails(nom) {
			if len(q) > maxAssetTail {
				continue
			}
			for _, v := range withUnderscores(q) {
				if !seen[v] {
					seen[v] = true
					out = append(out, v)
				}
			}
		}
	}
	return out
}

// assetTails rend le nom entier puis chacune de ses queues apres un `_`.
func assetTails(name string) []string {
	out := []string{name}
	parts := strings.Split(name, "_")
	for i := 1; i < len(parts); i++ {
		out = append(out, strings.Join(parts[i:], "_"))
	}
	return out
}

// withUnderscores rend s, puis s avec un underscore insere a chaque position, puis a chaque paire.
func withUnderscores(s string) []string {
	out := []string{s}
	for i := 1; i < len(s); i++ {
		a := s[:i] + "_" + s[i:]
		out = append(out, a)
		for j := i + 2; j < len(a); j++ {
			out = append(out, a[:j]+"_"+a[j:])
		}
	}
	return out
}
