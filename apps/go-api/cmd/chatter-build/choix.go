package main

// choix.go — QUELLE VARIANTE PAR SON, ET QUELS SONS SONT PARLES.
//
// Un son de réplique a plusieurs variantes (une à six) : autant d'enregistrements de la même intention, avec
// leur propre phrase. Le film ne dit pas laquelle a joué. La table retient UNE variante par son, et son texte
// ET sa voix viennent de cette même variante : la phrase affichée est celle qu'on entend.
//
// Une réplique « parlée » est un son dont la variante retenue a un texte dans chaque langue requise de
// `config/locales.toml`. Les autres sons (cris de bouclier percé, atterrissage dur, recul) ont une voix et
// jamais de texte : ils ne s'affichent ni ne se jouent (décision du propriétaire, 2026-11-24).

import (
	"sort"

	"levelup/go-api/internal/i18n"
)

// BilanChoix : ce que la sélection a gardé et écarté, pour le compte rendu.
type BilanChoix struct {
	Joues       int            `json:"joues"`
	SansBanque  int            `json:"sans_banque"`
	SansTexte   int            `json:"sans_texte"`
	Parles      int            `json:"parles"`
	AvecVoix    map[string]int `json:"avec_voix"`
	SansVariant int            `json:"sans_variante"`
}

// choisir retient, pour chaque son joué, la variante parlée à verser. La variante retenue est la première (dans
// l'ordre du tag) qui a un texte dans chaque langue requise ; si elle n'a pas de voix dans la langue par défaut
// de l'application alors qu'une autre variante en a une, c'est celle-là. Un média n'est
// retenu que s'il existe dans une langue : la variante garde le premier média de l'évènement.
func choisir(joues SonsJoues, textes []SonTextes, banques []SonBanque, langues []langueApp) ([]Choix, BilanChoix) {
	bilan := BilanChoix{AvecVoix: map[string]int{}}
	banqueDe := map[string]SonBanque{}
	for _, b := range banques {
		banqueDe[b.Son] = b
	}
	var out []Choix
	for _, t := range textes {
		if joues.Sons[t.Son] == 0 {
			continue
		}
		bilan.Joues++
		b := banqueDe[t.Son]
		if b.Absente {
			bilan.SansBanque++
		}
		if len(t.Variantes) == 0 {
			bilan.SansVariant++
			continue
		}
		c, ok := meilleureVariante(t, b, langues)
		if !ok {
			bilan.SansTexte++
			continue
		}
		bilan.Parles++
		for code := range c.Voix {
			bilan.AvecVoix[code]++
		}
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Son < out[j].Son })
	return out, bilan
}

// meilleureVariante : voir `choisir`. `ok` est faux quand aucune variante n'a de texte dans toutes les
// langues requises.
func meilleureVariante(t SonTextes, b SonBanque, langues []langueApp) (Choix, bool) {
	evenements := map[string]EvenementBanque{}
	for _, e := range b.Evenements {
		evenements[e.Evenement] = e
	}
	var premier *Choix
	for _, v := range t.Variantes {
		if !texteComplet(v.Textes, langues) {
			continue
		}
		c := Choix{Son: t.Son, Evenement: v.Evenement, Cle: v.Cle, Textes: v.Textes, Banque: t.Banque, Voix: map[string]MediaVoix{}}
		for _, l := range langues {
			if medias := evenements[v.Evenement].Langues[l.Code]; len(medias) > 0 {
				c.Voix[l.Code] = medias[0]
			}
		}
		if premier == nil {
			cp := c
			premier = &cp
		}
		if _, ok := c.Voix[string(i18n.Default)]; ok {
			return c, true
		}
	}
	if premier == nil {
		return Choix{}, false
	}
	return *premier, true
}

// texteComplet : un texte non vide dans chaque langue requise.
func texteComplet(textes map[string]string, langues []langueApp) bool {
	for _, l := range langues {
		if l.Requise && textes[l.Code] == "" {
			return false
		}
	}
	return true
}
