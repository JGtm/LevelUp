// Package profile — service minimal exposant le LUSR + tendance LOWESS d'un joueur
// pour la couche coach (V2 Ascension).
//
// Le plan original V2 référence un « PlayerProfile V1 » qui n'a jamais été
// matérialisé comme service unique : les briques existaient en pièces détachées
// (match_skill_rank, skill_config tiers, temporal.LowessSmooth). Ce package
// agrège ce qu'il faut pour les alertes coach (LUSRTierApproach +
// LOWESSPositive) sans dépendre d'une infra absente.
//
// Réutilisable par UI future (page profil progression).
package profile

import "time"

// PlayerProfile rassemble l'état de progression d'un joueur sur un titre.
type PlayerProfile struct {
	UserID    string
	TitleSlug string
	UpdatedAt time.Time

	// LUSR : état courant. Empty si pas assez de matchs (MinMatchesForRating).
	LUSR LUSRState

	// Tier courant + prochain sub-tier (vide si tier max).
	Tier     TierState
	NextTier TierState

	// Tendance LOWESS sur μ (l'agrégat LUSR composite). Window indique la
	// taille effective de l'observation. Slope > 0 = amélioration sur la
	// fenêtre.
	MuTrend LOWESSTrend
}

// LUSRState capture l'état LUSR courant.
type LUSRState struct {
	Mu    float64
	Sigma float64
	// MatchesCount : nombre de matchs ayant contribué au rating (pour gating).
	MatchesCount int
	// LastMatchAt : timestamp du dernier match avec rating (séparé de
	// l'horloge serveur pour gérer les imports décalés).
	LastMatchAt *time.Time
}

// TierState décrit un tier + sub-tier LUSR.
type TierState struct {
	Name    string  // "Diamond"
	NameFR  string  // "Diamant"
	SubTier int     // 1..6 (0 si tier sans sub-tier comme Onyx)
	Label   string  // "Diamond III"
	LowerMu float64 // entrée du sub-tier (inclusive)
	UpperMu float64 // sortie du sub-tier (= entrée du sub-tier suivant, exclusive)
}

// IsEmpty retourne true si le TierState n'est pas renseigné.
func (t TierState) IsEmpty() bool { return t.Name == "" }

// LOWESSTrend décrit une tendance lissée sur une métrique.
type LOWESSTrend struct {
	Metric string  // "mu" pour la tendance globale LUSR
	Slope  float64 // diff (lastSmoothed - firstSmoothed) sur la fenêtre
	Window int     // nombre de points effectifs dans la fenêtre LOWESS
}

// IsPositive retourne true si la tendance est positive ET suffisamment longue.
func (t LOWESSTrend) IsPositive(minWindow int) bool {
	return t.Slope > 0 && t.Window >= minWindow
}

// MinMatchesForRating est le seuil sous lequel on considère le rating non
// fiable (aligné sur sync.MinMatchesForRating mais dupliqué ici pour éviter
// la dépendance cyclique entre progression et sync).
const MinMatchesForRating = 10

// LOWESSAlpha est le paramètre de lissage passé à temporal.LowessSmooth.
// 0.3 = fenêtre ~30% du dataset (cohérent avec defaults Python statsmodels).
const LOWESSAlpha = 0.3
