package main

// langues.go — LE PONT ENTRE LES LANGUES DE L'APPLICATION ET CELLES DU JEU.
//
// Les langues de l'application sont celles de `config/locales.toml` (source unique, ADR 0040) ; le jeu a les
// siennes : un index dans ses listes de chaînes, un dossier de voix. Le pont passe par l'étiquette Halo que
// `locales.toml` donne à chaque langue (`halo = ["fr-FR"]`). Ajouter une langue à l'application sans
// l'ajouter ici fait échouer la commande par un message qui dit quoi écrire, jamais produire une table
// amputée.

import (
	"fmt"

	"levelup/go-api/internal/i18n"
)

// langueJeu : une langue que le jeu sait dire. Etiquette est l'étiquette Halo, Blob l'index de la langue dans
// le blob d'une liste de chaînes (uslg.go), Dossier le dossier des voix sous `Sound/win`.
type langueJeu struct {
	Etiquette string
	Blob      int
	Dossier   string
}

// languesDuJeu : les langues dont l'index et le dossier sont vérifiés (jeu installé, build HI_1_13_0). Les
// voix anglaises ne sont installées qu'en écran d'accueil : le dossier existe, ses paquets de voix de Spartan
// non (`medias` le dit au lieu de le taire).
var languesDuJeu = []langueJeu{
	{Etiquette: "en-US", Blob: 0, Dossier: "English(US)"},
	{Etiquette: "fr-FR", Blob: 3, Dossier: "French(France)"},
}

// langueApp : une langue de l'application avec sa langue du jeu.
type langueApp struct {
	Code    string
	Requise bool
	Jeu     langueJeu
}

// languesDeLApp résout chaque langue de `locales.toml` vers une langue du jeu. Une langue requise sans
// langue du jeu connue est une erreur ; une langue facultative sans équivalent est écartée.
func languesDeLApp() ([]langueApp, error) {
	var out []langueApp
	for _, l := range i18n.Locales() {
		jeu, ok := langueDuJeuDe(i18n.HaloTags(l))
		switch {
		case ok:
			out = append(out, langueApp{Code: string(l), Requise: i18n.Required(l), Jeu: jeu})
		case i18n.Required(l):
			return nil, fmt.Errorf("langue %q : aucune étiquette Halo %v dans languesDuJeu (langues.go) ; "+
				"ajouter son index de blob et son dossier de voix", l, i18n.HaloTags(l))
		}
	}
	return out, nil
}

// langueDuJeuDe rend la première langue du jeu qui porte une des étiquettes.
func langueDuJeuDe(etiquettes []string) (langueJeu, bool) {
	for _, e := range etiquettes {
		for _, j := range languesDuJeu {
			if j.Etiquette == e {
				return j, true
			}
		}
	}
	return langueJeu{}, false
}
