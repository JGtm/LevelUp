package temporal

import "levelup/go-api/internal/games/canonical"

// engagementEventWeight — poids d'un event highlight dans le RYTHME de la courbe
// d'engagement (« meneur » vs « en retrait »). Pondère l'action MENÉE plutôt que
// subie / logistique :
//   - "mode" (objectif)  1.5 : prime meneur d'objectif (porter > fragger). String
//     brute Infinite (pas de constante canonical) ; H5 = impulses objectif (allowlist).
//   - assist             0.5 : support (action menée, PAS un double comptage du frag ;
//     H5 natif ; Infinite absent du timeline).
//   - death              0.0 : subie, jamais une action menée. Un frag d'un côté est
//     déjà une mort de l'autre (double comptage kill/mort) — le compter côté victime
//     ferait « répondre » un joueur qui se fait farmer. Chaque affrontement compte une
//     fois, côté acteur (décision user 2026-07-07).
//   - défaut (kill, medal, first_kill/death, finisher, clutch…) 1.0 : neutre. Le
//     medal à 1.0 ADDITIF encode l'intensité (un double kill = kill(s) + medal →
//     pèse plus que deux kills isolés — décision user, on ne dé-duplique pas).
//
// RECENSEMENT (pondération appliquée à UN SEUL endroit, par cohérence) : ce poids
// est sommé dans buildEngagementCurve (paces) → propage au résidu, au score ET aux
// coefficients (RatioSample dérivés des means de la courbe). Restent BRUTS
// volontairement : MatchIntensity (densité d'events = descripteur de chaos du match,
// pas du leadership) et PlayerActivity (K+A+D, simple filtre AFK). Impact INFINITE :
// death↓ (1.0→0.0) + objectif↑ (1.0→1.5) → re-backfill des 2 titres requis. Baisse
// mécanique de ~25 % des paces sur un mix kills≈morts (seuils de filtre abaissés en
// conséquence, cf. engagement_coefficients.go).
//
// Poids validés user : objectif 1.5 / assist 0.5 (2026-06-26), death 0.0 (2026-07-07,
// modèle lobby-anchored). Calibrables, mêmes valeurs les 2 titres.
func engagementEventWeight(eventType string) float64 {
	switch eventType {
	case "mode": // objectif (Infinite event_type "mode" + impulses objectif H5)
		return 1.5
	case string(canonical.EventAssist):
		return 0.5
	case string(canonical.EventDeath):
		return 0.0
	default:
		return 1.0
	}
}
