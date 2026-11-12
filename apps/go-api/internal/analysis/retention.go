// Package analysis — retention.go : la borne de la fenêtre de rétention des artefacts de rejeu.
//
// Calcul pur (aucun SQL) : les prédicats qui l'emploient (sqlfrag.SQLDansFenetreRetention,
// sqlfrag.SQLEligibleALaCuisson) vivent dans platform/duckdb/sqlfrag.
package analysis

import "time"

// BorneRetention rend l'instant a partir duquel un match est DANS la fenetre, et `true` si
// la fenetre est bornee.
//
// `mois <= 0` VEUT DIRE ILLIMITEE, jamais « zero mois » : c'est le reglage par defaut
// (`ReplayRetentionMonths`), et un match n'en sort alors jamais.
func BorneRetention(mois int) (time.Time, bool) {
	return BorneRetentionDepuis(time.Now().UTC(), mois)
}

// BorneRetentionDepuis est [BorneRetention] avec un instant de reference explicite.
//
// ELLE EXISTE POUR L'HORLOGE INJECTEE du cron de purge, qui doit pouvoir se placer a une
// date choisie dans ses tests. Le calcul reste ICI : c'est lui, et non l'appel a
// `time.Now`, qui doit rester unique — un `AddDate` recopie chez l'appelant est exactement
// la divergence que le garde-rail `no_retention_window_inline_test` interdit.
func BorneRetentionDepuis(now time.Time, mois int) (time.Time, bool) {
	if mois <= 0 {
		return time.Time{}, false
	}
	return now.AddDate(0, -mois, 0), true
}
