// Package analysis — expected_fda.go : FDA attendu et écart au FDA réel.
//
// Source UNIQUE de la formule « FDA attendu » (kills_expected +
// assists_expected/3 − deaths_expected) et de l'écart réel − attendu. Toutes les
// surfaces (Timeseries, Sessions, Escouade) passent par ces deux helpers plutôt
// que de réécrire l'arithmétique inline — garde-rail
// archlint/no_inline_expected_fda_test.go (interdit l'arithmétique
// *Expected .../3 hors de ce package).
//
// ADR 0006 : le FDA réel per-match est la valeur API native (jamais recalculé) ;
// ces helpers ne produisent QUE l'attendu et son écart.
package analysis

// ExpectedFDA calcule le FDA attendu d'un match :
// kills_expected + assists_expected/3 − deaths_expected.
//
// Retourne nil si kills_expected ou deaths_expected est absent (nil) ou non fini
// (NaN/±Inf) — un attendu sans son terme K ou D n'a pas de sens (des lignes
// kills_expected=+Inf existent en base, garde OBLIGATOIRE). assists_expected nil
// (ou non fini) est traité comme un terme 0 : l'attendu dégrade proprement en
// K/D pur, car le modèle d'assists attendus est best-effort (absent pour les
// joueurs sous le seuil d'échantillons).
func ExpectedFDA(killsExp, deathsExp, assistsExp *float64) *float64 {
	if killsExp == nil || deathsExp == nil {
		return nil
	}
	if IsBadFloat(*killsExp) || IsBadFloat(*deathsExp) {
		return nil
	}
	assistsTerm := 0.0
	if assistsExp != nil && !IsBadFloat(*assistsExp) {
		assistsTerm = *assistsExp / 3.0
	}
	v := *killsExp + assistsTerm - *deathsExp
	return &v
}

// FDADiff calcule l'écart FDA réel − FDA attendu. Retourne nil si l'un des deux
// termes est absent (nil) ou non fini — pas d'écart sans les deux valeurs.
// actualKDA est la valeur API native (StatsMatchRow.KDA, ADR 0006) ; expectedFDA
// vient d'ExpectedFDA (déjà fini par construction, la garde reste défensive).
func FDADiff(actualKDA, expectedFDA *float64) *float64 {
	if actualKDA == nil || expectedFDA == nil {
		return nil
	}
	if IsBadFloat(*actualKDA) || IsBadFloat(*expectedFDA) {
		return nil
	}
	d := *actualKDA - *expectedFDA
	return &d
}
