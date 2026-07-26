// Package campaign — ImprovementCampaign V1 (PLAN_PLAYER_PROFILE_ASCENSION §4.5).
//
// Une campagne d'amélioration est un mini-objectif personnel volontaire sur
// 1 axe (radar 6 axes OU 1 des 8 composantes LUSR). Pas de deadline, pas de
// pénalité. La progression visible est sa propre récompense.
//
// Principe clé : opt-in, jamais imposé. Pause/abandon/clôture libres.
//
// Cf. plan §4.5 — 5 raffinements algorithmiques :
//
//	R1 : delta lissé LOWESS (pas brut)
//	R2 : phrasing strict — pas de causalité revendiquée
//	R3 : filtre playlist optionnel
//	R4 : test Mann-Whitney U pour milestone "progression confirmée"
//	R5 : auto-suggestion de clôture (jamais auto-fermeture)
package campaign

import "time"

// CampaignStatus est l'état courant d'une campagne.
type CampaignStatus string

const (
	StatusActive    CampaignStatus = "active"
	StatusPaused    CampaignStatus = "paused"
	StatusCompleted CampaignStatus = "completed"
	StatusAbandoned CampaignStatus = "abandoned"
)

// Valid retourne true si le statut appartient à l'énum.
//
// La colonne `improvement_campaign.status` est un VARCHAR sans contrainte : une
// valeur inventée s'insère sans erreur et disparaît simplement des lectures, qui
// filtrent sur des littéraux ('active' pour GetActive, IN ('completed',
// 'abandoned') pour ListEnded). Tout producteur de lignes doit donc valider
// AVANT d'écrire — sinon l'invisibilité est le seul symptôme.
func (s CampaignStatus) Valid() bool {
	switch s {
	case StatusActive, StatusPaused, StatusCompleted, StatusAbandoned:
		return true
	}
	return false
}

// AxisKind différencie un axe radar (1 des 6 narrative) d'une composante LUSR
// (1 des 8 du composite_score). Le tracking sera différent selon le kind :
//
//   - radar          : moyenne d'un raw axis (combat, survival, …) post-snapshot
//   - lusr_component : moyenne d'une composante composite (kills_vs_expected, …)
type AxisKind string

const (
	AxisKindRadar         AxisKind = "radar"
	AxisKindLUSRComponent AxisKind = "lusr_component"
)

// ImprovementCampaign capture l'état d'une campagne d'amélioration.
//
// Persistance : table `improvement_campaign` dans stats.duckdb (par joueur).
type ImprovementCampaign struct {
	ID                   string         `json:"id"`
	UserID               string         `json:"user_id"`
	TitleSlug            string         `json:"title_slug"`
	Axis                 string         `json:"axis"`
	AxisKind             AxisKind       `json:"axis_kind"`
	StartedAt            time.Time      `json:"started_at"`
	EndedAt              *time.Time     `json:"ended_at,omitempty"`
	Status               CampaignStatus `json:"status"`
	PlaylistGroup        string         `json:"playlist_group"`
	SnapshotValue        float64        `json:"snapshot_value"`
	SnapshotSample       int            `json:"snapshot_sample"`
	CurrentValueRaw      *float64       `json:"current_value_raw,omitempty"`
	CurrentValueLOWESS   *float64       `json:"current_value_lowess,omitempty"`
	MatchesSinceStart    int            `json:"matches_since_start"`
	LastEvaluatedAt      *time.Time     `json:"last_evaluated_at,omitempty"`
	MannWhitneyP         *float64       `json:"mann_whitney_p,omitempty"`
	ProgressionConfirmed bool           `json:"progression_confirmed"`
	AutoClosureSuggested bool           `json:"auto_closure_suggested"`
	AutoClosureReason    string         `json:"auto_closure_reason,omitempty"`

	// LinkedChallengeIDs : défis tagués campaign_id = this.ID. Hydraté à la
	// demande (GET /campaigns/{id}). Vide en lecture liste.
	LinkedChallengeIDs []string `json:"linked_challenge_ids,omitempty"`
}

// IsActive retourne true si la campagne est en cours.
func (c *ImprovementCampaign) IsActive() bool { return c.Status == StatusActive }

// IsEnded retourne true si la campagne est terminée (completed ou abandoned).
func (c *ImprovementCampaign) IsEnded() bool {
	return c.Status == StatusCompleted || c.Status == StatusAbandoned
}

// FinalValue retourne la meilleure estimation de la valeur finale de l'axe :
// LOWESS lissé si disponible, sinon la valeur brute. nil si la campagne n'a
// jamais été évaluée (aucune des deux valeurs).
func (c *ImprovementCampaign) FinalValue() *float64 {
	if c.CurrentValueLOWESS != nil {
		v := *c.CurrentValueLOWESS
		return &v
	}
	if c.CurrentValueRaw != nil {
		v := *c.CurrentValueRaw
		return &v
	}
	return nil
}

// Delta retourne l'écart valeur finale − snapshot de départ (progression sur
// l'axe). nil si la valeur finale est inconnue (campagne non évaluée).
func (c *ImprovementCampaign) Delta() *float64 {
	final := c.FinalValue()
	if final == nil {
		return nil
	}
	d := *final - c.SnapshotValue
	return &d
}

// MinMatchesForTrend est le seuil sous lequel le delta lissé n'est pas affiché.
// Cf. R1 (delta lissé LOWESS) — l'UI montre "joue encore N matchs" en dessous.
const MinMatchesForTrend = 20

// MinMatchesForMannWhitney est le seuil sous lequel le test stat n'est pas
// calculé. Cf. R4 — déclenche la notif "progression confirmée".
const MinMatchesForMannWhitney = 20

// MannWhitneyThreshold est le p-value en-dessous duquel on déclare
// `ProgressionConfirmed = true`. Cf. R4 — p < 0.05.
const MannWhitneyThreshold = 0.05

// LOWESSAlpha est le paramètre de lissage utilisé pour le delta.
// Aligné sur profile.LOWESSAlpha (0.3 = fenêtre ~30%).
const LOWESSAlpha = 0.3

// PlateauWindowDays est la durée sans variation significative au-delà de
// laquelle on suggère la clôture (cf. R5).
const PlateauWindowDays = 60
