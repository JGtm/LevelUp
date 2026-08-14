package replay

// neutral_deaths.go — LES MORTS QUE PERSONNE NE REVENDIQUE, côté assemblage.
//
// Ce fichier ne DÉCODE rien : le type d'une telle mort est résolu par l'appelant (cf.
// Options.NeutralDeaths et le commentaire qui dit pourquoi). Il ne fait qu'appliquer au calque
// la règle commune à tous les autres — on ne publie que ce qui rencontrera une trajectoire.

import "log/slog"

// keepNeutralDeathsOfPublishedTracks écarte les morts dont le joueur n'a AUCUNE trajectoire
// publiée.
//
// POURQUOI CE FILTRE EXISTE : le client ne dessine pas ces lignes à partir de cette table. Il
// les déduit de SES PISTES (une fin de vie qu'aucun kill ne consomme) et vient ensuite y
// chercher de quoi le joueur est mort. Une entrée sans piste ne rencontrerait donc jamais de
// ligne à décorer — elle ne serait pas fausse, elle serait morte. C'est la même règle que pour
// les tirs, les lancers et les actions d'objectif, et elle est appliquée au même endroit.
//
// L'écart est JOURNALISÉ, jamais tu : un calque qui perd des entrées en silence laisse croire
// que le film n'en portait pas.
func keepNeutralDeathsOfPublishedTracks(deaths []NeutralDeath, tracks []Track) []NeutralDeath {
	if len(deaths) == 0 {
		return nil
	}
	published := map[string]bool{}
	for _, tr := range tracks {
		if tr.XUID != "" {
			published[tr.XUID] = true
		}
	}
	out := make([]NeutralDeath, 0, len(deaths))
	for _, d := range deaths {
		if d.Kind == "" || !published[d.XUID] {
			continue
		}
		out = append(out, d)
	}
	if dropped := len(deaths) - len(out); dropped > 0 {
		slog.Info("rejeu 2D : morts sans revendication écartées (joueur sans trajectoire publiée, "+
			"ou type non établi)", "ecartees", dropped, "publiees", len(out))
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
