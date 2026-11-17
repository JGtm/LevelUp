// Package profile — service PlayerProfile (V1 Ascension).
//
// Depuis 2026-05-18 (V1 PlayerProfile commit 4) ce package porte le service
// complet décrit dans `PLAN_PLAYER_PROFILE_ASCENSION.md` §4-§5 :
//   - Section A1 : radar narrative 6 axes + rôles
//   - Section A2 : style FK/FD + engagement
//   - Section B : LUSR tier + 8 composantes + tendance LOWESS
//   - Section C : leviers + suggestions de défis
//
// Historique : précédente version V2-commit-6 ne portait que LUSRState +
// TierState + MuTrend (mini-service workaround). V1 commit 4 enrichit
// le `PlayerProfile` avec les nouveaux champs sans casser les callers V2
// (la closure post-sync continue à lire `p.LUSR.Mu`, `p.NextTier.Label`
// et `p.MuTrend`).
package profile

import (
	"time"
)

// MinMatchesForProfile est le seuil sous lequel on considère que le profil
// n'a pas assez de données pour les insights (Section A2 + B + C). Section A1
// dégrade gracieusement (radar avec seulement les axes calculables).
const MinMatchesForProfile = 30

// muPoint = un rating LUSR daté, source de la sparkline de tendance (interne).
type muPoint struct {
	at    time.Time
	value float64
}

// SkillTrendWindowDays : fenêtre FIXE (90 j) de la sparkline de tendance LUSR
// (DEC-5). Indépendante du window_days du profil (sections A1/A2/B/C) : la
// sparkline porte une sémantique « tendance sur 90 jours » stable.
const SkillTrendWindowDays = 90

// MinMatchesForRating est le seuil sous lequel on considère le rating non
// fiable (aligné sur sync.MinMatchesForRating mais dupliqué ici pour éviter
// la dépendance cyclique entre progression et sync).
const MinMatchesForRating = 10

// LOWESSAlpha est le paramètre de lissage passé à temporal.LowessSmooth.
// 0.3 = fenêtre ~30% du dataset (cohérent avec defaults Python statsmodels).
const LOWESSAlpha = 0.3
