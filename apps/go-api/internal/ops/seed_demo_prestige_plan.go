// Package ops — seed_demo_prestige_plan.go : PLAN Prestige démo (données pures).
//
// Ce fichier ne touche AUCUNE base : il transforme les agrégats du corpus démo
// déjà extrait (demoCorpusStats) en un jeu de lignes Prestige cohérent. Les
// écritures vivent dans seed_demo_prestige.go (player DB) et
// seed_demo_prestige_social.go (shared_social).
//
// Invariants de cohérence portés par ce plan (vérifiés par
// seed_demo_prestige_plan_test.go) :
//  1. Chaque objectif appartient à un arc déclaré, ou à aucun (arc_id vide).
//  2. Un arc marqué complété n'a QUE des objectifs terminaux ; un arc actif a au
//     moins un objectif actif et au moins un objectif complété (progression
//     visible « Étape k/n »).
//  3. Les cibles sont dérivées du corpus : un objectif COMPLÉTÉ vise en-deçà du
//     réalisé du joueur démo, un objectif ACTIF vise au-delà.
//  4. TotalPP == somme des pp_amount des événements, et Level == LevelFromPP.
//  5. Aucune date hors de la fenêtre du corpus (pas d'objectif « créé » avant le
//     premier match de la démo).
//  6. Aucune identité réelle : user_id = slug démo, xuid = xuid démo.
package ops

import (
	"fmt"
	"math"
	"time"

	"levelup/go-api/internal/campaign"
	"levelup/go-api/internal/i18n"
	"levelup/go-api/internal/prestige"
	"levelup/go-api/internal/progression/records"
	"levelup/go-api/internal/progression/streaks"
)

// demoCorpusStats agrège le corpus démo (match_participants du joueur démo
// principal) — source unique des cibles et des dates du plan Prestige.
type demoCorpusStats struct {
	Matches       int
	Kills         int
	Deaths        int
	Assists       int
	HeadshotKills int
	Wins          int
	First         time.Time
	Last          time.Time
}

// demoPrestigeArc décrit un arc démo.
type demoPrestigeArc struct {
	ID          string
	Title       string
	Description string
	PresetID    string
	IsPreset    bool
	CreatedAt   time.Time
	CompletedAt *time.Time
}

// demoPrestigeChallenge décrit un objectif démo. TerminalAt alimente la colonne
// correspondant au Status (completed_at / expired_at / abandoned_at).
type demoPrestigeChallenge struct {
	ID       string
	ArcID    string
	Position int
	// TemplateID : origine du libellé dans le catalogue du titre (vide = objectif sans origine,
	// servi tel quel dans une seule langue). Voir seed_demo_prestige_objectives.go.
	TemplateID  string
	Metric      string
	Label       string
	Target      float64
	Tier        prestige.Tier
	Cadence     prestige.Cadence
	EvalType    prestige.EvalType
	WindowType  prestige.WindowType
	WindowValue string
	Status      prestige.ChallengeStatus
	CreatedAt   time.Time
	TerminalAt  *time.Time
	PP          int
}

// demoPrestigeCampaign décrit une campagne d'amélioration démo.
//
// Status est TYPÉ campaign.CampaignStatus : la table improvement_campaign ne
// porte pas de contrainte d'énum, et la lecture (CampaignRepo.GetActive /
// ListEnded) filtre sur des littéraux ('active', puis IN ('completed',
// 'abandoned')). Un statut hors énum ne casse donc RIEN à l'écriture — il rend
// simplement la ligne invisible, silencieusement. C'est ce qui est arrivé à la
// campagne close de la démo, seedée « closed » (2026-07-26). Le typage seul ne
// suffit pas (une constante chaîne non typée se convertit) : la garde réelle est
// validateDemoCampaigns, appelée à l'écriture.
type demoPrestigeCampaign struct {
	ID            string
	Axis          string
	AxisKind      string
	Status        campaign.CampaignStatus
	StartedAt     time.Time
	EndedAt       *time.Time
	SnapshotValue float64
	SnapshotN     int
	CurrentValue  float64
	MatchesSince  int
	Confirmed     bool
}

// demoPrestigeStreak décrit une série démo (table append-only streak_history).
type demoPrestigeStreak struct {
	ID          string
	Type        streaks.StreakType
	Current     int
	Best        int
	ShieldsUsed int
	Status      streaks.StreakStatus
	StartedAt   time.Time
	LastAt      time.Time
	BrokenAt    *time.Time
}

// demoPrestigeRecord décrit un record personnel démo : une valeur COURANTE (le PB)
// et la valeur PRÉCÉDENTE qu'elle a battue. Les deux alimentent à la fois la
// timeline record_history (player DB) et le PB player_records_history (social).
type demoPrestigeRecord struct {
	Metric         records.TrackedMetric
	Period         records.RecordPeriod
	Value          float64
	AchievedAt     time.Time
	PrevValue      float64
	PrevAchievedAt time.Time
}

// demoPrestigeEvent décrit un gain de points de progression (prestige_events).
type demoPrestigeEvent struct {
	ID         string
	SourceType string
	SourceID   string
	PP         int
	Tier       string
	CreatedAt  time.Time
}

// demoPrestigePlan est le jeu complet à écrire pour un titre.
//
// UserID (slug de route) et XUID sont DEUX clés distinctes, comme en production :
// le module Prestige (arc, challenge, campagne, baseline, events, escouade) est
// clé par player_slug ; la couche Progression V2 (streak, record, jalon, PB) est
// clée par xuid. Les mélanger rendrait la moitié des pages vides.
type demoPrestigePlan struct {
	UserID    string
	XUID      string
	Gamertag  string
	TitleSlug string
	// Anchor : instant de référence des lignes « d'état » (baseline, jalons,
	// escouade, snapshot de total). Ancré sur le DERNIER match du corpus — jamais
	// time.Now(), sans quoi le générateur synthétique perdrait son byte-identique
	// run-à-run (TestSeedDemoSynthetic_Deterministic).
	Anchor     time.Time
	Arcs       []demoPrestigeArc
	Challenges []demoPrestigeChallenge
	Campaigns  []demoPrestigeCampaign
	Streaks    []demoPrestigeStreak
	Records    []demoPrestigeRecord
	Events     []demoPrestigeEvent
	TotalPP    int
	Level      int
}

// demoMilestoneCount : nombre de jalons du catalogue marqués « débloqués » pour la
// démo (les N plus accessibles par seuil croissant). Assez pour que la grille
// Réalisations montre du débloqué ET du verrouillé.
const demoMilestoneCount = 6

// buildDemoPrestigePlan construit le plan complet depuis le corpus démo.
// matchPP = corpus (source des événements PP « match »), won = résultat par match.
// cat = catalogue de modèles et de presets du titre : arcs et objectifs en sont des adoptions
// (seed_demo_prestige_objectives.go) ; un catalogue vide donne des textes de repli sans origine.
//
// L'identité du joueur de démo n'est PAS paramétrable : slug, xuid et gamertag
// sont fixés par construction. C'est délibéré — les rendre variables laisserait
// croire qu'on peut générer un plan pour un joueur RÉEL, alors que tout le
// contenu ci-dessous est synthétique et que l'identité d'un vrai joueur
// fuiterait dans les fixtures publiques de la démo (invariant n°6, vérifié par
// TestDemoPrestigePlan_NoRealIdentityLeak).
//
// Les DEUX identités restent distinctes dans le plan : `UserID` est un
// player_slug (clé des données Prestige), `XUID` est l'identifiant Xbox (clé de
// Progression V2 — séries, records, jalons). Les confondre viderait la moitié
// des pages ; c'est la confusion que verrouille le garde-rail
// xuid_identity_ratchet_test côté lecture.
func buildDemoPrestigePlan(titleSlug string, cat demoPrestigeCatalog,
	st demoCorpusStats, matchPP []demoMatchPP) demoPrestigePlan {
	tuning := prestige.DefaultTuning()
	plan := demoPrestigePlan{
		UserID:    DefaultDemoMainSlug,
		XUID:      demoXUIDForIndex(0),
		Gamertag:  DefaultDemoMainGamertag,
		TitleSlug: titleSlug,
		Anchor:    st.Last,
	}
	plan.Arcs, plan.Challenges = buildDemoArcsAndChallenges(tuning, cat, st)
	plan.Campaigns = buildDemoCampaigns(st)
	plan.Streaks = buildDemoStreaks(st)
	plan.Records = buildDemoRecords(st)
	plan.Events = buildDemoEvents(tuning, plan.Challenges, plan.Arcs, matchPP, st)
	for _, ev := range plan.Events {
		plan.TotalPP += ev.PP
	}
	plan.Level = prestige.LevelFromPP(tuning, plan.TotalPP, i18n.Default).Index
	return plan
}

// demoMatchPP porte le minimum nécessaire d'un match du corpus pour émettre son
// événement PP (identifiant + résultat + date).
type demoMatchPP struct {
	MatchID string
	Won     bool
	At      time.Time
}

// demoTargetBelow retourne une cible SOUS le réalisé du corpus (objectif atteint).
// ratio doit être < 1. L'arrondi se fait vers le BAS : sur un corpus minuscule
// (fixtures de test), un arrondi au palier supérieur produirait une cible au-dessus
// du réalisé — donc un objectif « complété » qu'on n'a jamais atteint.
func demoTargetBelow(realized int, ratio float64) float64 {
	return demoRoundTargetDown(float64(realized) * ratio)
}

// demoTargetAbove retourne une cible AU-DESSUS du réalisé (objectif en cours).
// ratio doit être > 1, et l'arrondi se fait vers le HAUT (jamais en-deçà du
// réalisé, sinon l'objectif serait affiché « en cours » alors qu'il est atteint).
func demoTargetAbove(realized int, ratio float64) float64 {
	return demoRoundTargetUp(float64(realized) * ratio)
}

// demoNotBefore borne un instant au plancher fourni (début du corpus démo).
func demoNotBefore(at, floor time.Time) time.Time {
	if at.Before(floor) {
		return floor
	}
	return at
}

// demoTargetStep retourne le palier d'arrondi lisible d'une cible (5 sous 100,
// 10 sous 500, 25 au-delà).
func demoTargetStep(v float64) float64 {
	switch {
	case v >= 500:
		return 25
	case v >= 100:
		return 10
	}
	return 5
}

// demoRoundTargetDown arrondit vers le bas au palier lisible, plancher 1.
func demoRoundTargetDown(v float64) float64 {
	step := demoTargetStep(v)
	r := math.Floor(v/step) * step
	if r < 1 {
		r = 1
	}
	return r
}

// demoRoundTargetUp arrondit vers le haut au palier lisible, plancher = palier.
func demoRoundTargetUp(v float64) float64 {
	step := demoTargetStep(v)
	r := math.Ceil(v/step) * step
	if r < step {
		r = step
	}
	return r
}

// buildDemoCampaigns : 1 campagne d'amélioration en cours + 1 close et confirmée.
//
// La close porte campaign.StatusCompleted et NON « closed » : ce dernier
// n'appartient pas à l'énum (active|paused|completed|abandoned) et
// CampaignRepo.ListEnded filtre `status IN ('completed','abandoned')` — la ligne
// était écrite puis jamais servie, laissant la page Profil sans aucune campagne
// visible (2026-07-26). Elle a un EndedAt et Confirmed : « completed » est le
// statut terminal qui la décrit.
func buildDemoCampaigns(st demoCorpusStats) []demoPrestigeCampaign {
	ended := demoNotBefore(st.Last.Add(-96*time.Hour), st.First)
	return []demoPrestigeCampaign{
		{
			ID: "demo-campaign-accuracy", Axis: "accuracy", AxisKind: "metric",
			Status: campaign.StatusActive, StartedAt: st.First, SnapshotValue: 0.42, SnapshotN: 20,
			CurrentValue: 0.46, MatchesSince: st.Matches / 2,
		},
		{
			ID: "demo-campaign-kda", Axis: "kda", AxisKind: "metric",
			Status: campaign.StatusCompleted, StartedAt: st.First, EndedAt: &ended,
			SnapshotValue: 1.10, SnapshotN: 20, CurrentValue: 1.45,
			MatchesSince: st.Matches, Confirmed: true,
		},
	}
}

// validateDemoCampaigns refuse d'écrire une campagne dont le statut sort de
// l'énum campaign.CampaignStatus.
//
// Garde POSITIONNÉE À L'ÉCRITURE et non dans le builder : c'est l'INSERT qui
// grave la valeur, et un statut hors énum ne produit aucune erreur SQL (la
// colonne est un VARCHAR libre) — seulement une ligne que les lectures filtrées
// n'atteignent jamais. Sans cette garde, le prochain statut inventé repartirait
// pour un tour de silence.
func validateDemoCampaigns(campaigns []demoPrestigeCampaign) error {
	for _, c := range campaigns {
		if !c.Status.Valid() {
			return fmt.Errorf("campagne démo %s : statut %q hors énum "+
				"(active|paused|completed|abandoned) — la ligne serait écrite puis "+
				"jamais servie par CampaignRepo", c.ID, c.Status)
		}
	}
	return nil
}

// buildDemoStreaks : 2 séries en cours + 1 cassée (la grille Séries montre les
// trois états et le multiplicateur de points associé).
func buildDemoStreaks(st demoCorpusStats) []demoPrestigeStreak {
	broken := demoNotBefore(st.Last.Add(-48*time.Hour), st.First)
	return []demoPrestigeStreak{
		{ID: "demo-streak-daily-play", Type: streaks.StreakTypeDailyPlay,
			Current: 5, Best: 12, Status: streaks.StreakStatusActive,
			StartedAt: demoNotBefore(st.Last.Add(-5*24*time.Hour), st.First), LastAt: st.Last},
		{ID: "demo-streak-weekly-play", Type: streaks.StreakTypeWeeklyPlay,
			Current: 3, Best: 6, Status: streaks.StreakStatusActive,
			StartedAt: demoNotBefore(st.Last.Add(-21*24*time.Hour), st.First), LastAt: st.Last},
		{ID: "demo-streak-daily-perf", Type: streaks.StreakTypeDailyPerf,
			Current: 0, Best: 8, ShieldsUsed: streaks.MaxShieldsPerMonth,
			Status: streaks.StreakStatusBroken, StartedAt: st.First, LastAt: broken, BrokenAt: &broken},
	}
}

// buildDemoRecords : un PB par métrique suivie, sur 2 fenêtres (30 j + carrière).
// Les valeurs restent dans les bornes de vraisemblance (records.IsPlausibleValue)
// et la valeur carrière domine toujours la valeur 30 jours.
func buildDemoRecords(st demoCorpusStats) []demoPrestigeRecord {
	base := map[records.TrackedMetric][2]float64{
		records.MetricPerformanceScore: {74.5, 88.0},
		records.MetricKDA:              {2.4, 3.6},
		records.MetricKPM:              {1.15, 1.62},
		records.MetricAccuracy:         {0.48, 0.61},
		records.MetricPSPM:             {320, 455},
	}
	// La date « précédente » doit rester STRICTEMENT antérieure à la date du PB,
	// y compris sur un corpus d'une journée (fixtures de test).
	recent := demoNotBefore(st.Last.Add(-24*time.Hour), st.First)
	older := st.First
	if !older.Before(recent) {
		older = recent.Add(-time.Hour)
	}
	out := make([]demoPrestigeRecord, 0, 2*len(records.DefaultTrackedMetrics()))
	for _, m := range records.DefaultTrackedMetrics() {
		v, ok := base[m]
		if !ok {
			continue
		}
		out = append(out,
			demoPrestigeRecord{Metric: m, Period: records.RecordPeriod30d,
				Value: v[0], AchievedAt: recent, PrevValue: v[0] * 0.9, PrevAchievedAt: older},
			demoPrestigeRecord{Metric: m, Period: records.RecordPeriodAllTime,
				Value: v[1], AchievedAt: recent, PrevValue: v[1] * 0.92, PrevAchievedAt: older},
		)
	}
	return out
}

// buildDemoEvents construit le journal des points de progression. L'ordre des
// sources reflète le modèle réel : un événement par match du corpus (participation
// + bonus victoire), un par objectif complété, un bonus par arc complété, un bonus
// de série. TotalPP en découle — il n'est jamais posé « à la main ».
func buildDemoEvents(t prestige.Tuning, chs []demoPrestigeChallenge, arcs []demoPrestigeArc,
	matches []demoMatchPP, st demoCorpusStats) []demoPrestigeEvent {
	out := make([]demoPrestigeEvent, 0, len(matches)+len(chs)+len(arcs)+1)
	for i, m := range matches {
		out = append(out, demoPrestigeEvent{
			ID: fmt.Sprintf("demo-pe-match-%04d", i+1), SourceType: prestige.SourceMatch,
			SourceID: m.MatchID, PP: prestige.PPForMatch(t, m.Won), CreatedAt: m.At,
		})
	}
	arcObjectivePP := map[string]int{}
	for i, c := range chs {
		if c.Status != prestige.StatusCompleted {
			continue
		}
		arcObjectivePP[c.ArcID] += c.PP
		at := st.Last
		if c.TerminalAt != nil {
			at = *c.TerminalAt
		}
		out = append(out, demoPrestigeEvent{
			ID: fmt.Sprintf("demo-pe-obj-%02d", i+1), SourceType: prestige.SourceChallenge,
			SourceID: c.ID, PP: c.PP, Tier: string(c.Tier), CreatedAt: at,
		})
	}
	for i, a := range arcs {
		if a.CompletedAt == nil {
			continue
		}
		bonus := prestige.PPForArcCompletion(t, arcObjectivePP[a.ID])
		if bonus <= 0 {
			continue
		}
		out = append(out, demoPrestigeEvent{
			ID: fmt.Sprintf("demo-pe-arc-%02d", i+1), SourceType: prestige.SourceArc,
			SourceID: a.ID, PP: bonus, CreatedAt: *a.CompletedAt,
		})
	}
	out = append(out, demoPrestigeEvent{
		ID: "demo-pe-streak-01", SourceType: prestige.SourceStreak,
		SourceID: "demo-streak-daily-play", PP: prestige.PPForStreak(t), CreatedAt: st.Last,
	})
	return out
}
