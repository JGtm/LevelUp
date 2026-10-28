// Package ops — seed_demo_prestige_objectives.go : arcs et objectifs du plan Prestige démo.
//
// Chaque objectif démo est ADOSSÉ à un modèle du catalogue du titre et chaque arc à un preset :
// l'objectif porte le template_id du modèle et, comme à la création réelle d'un défi depuis un
// modèle (service.AdoptPresetArc), sa mesure, sa fenêtre, sa cadence, son type d'évaluation et
// la copie de départ de son libellé (Template.StoredLabel) ; l'arc porte le preset_id, le titre
// et la description copiés du preset (PresetArc.StoredTitle / StoredDescription). Le serveur
// sert alors ces textes dans la langue de la requête (ADR 0040 D-3).
//
// Un titre sans catalogue (Halo 5) ou un modèle absent du catalogue fait retomber l'élément sur
// son texte de repli : il est écrit sans origine et reste dans une seule langue.
package ops

import (
	"fmt"
	"time"

	"levelup/go-api/internal/prestige"
)

// arcIDs démo — stables entre deux reseeds (les pages profondes /arcs/{id} restent
// atteignables par lien direct).
const (
	demoArcAscensionID = "demo-arc-ascension"
	demoArcPrecisionID = "demo-arc-precision"
)

// Suffixes des presets du catalogue (identifiant complet "<slug>.<suffixe>").
const (
	demoPresetSlayer   = "slayer"
	demoPresetMarksman = "marksman"
)

// metricHeadshotKills : clé de métrique des tirs à la tête dans les textes de repli.
const metricHeadshotKills = "headshot_kills"

// demoArcSpec décrit un arc démo : le preset dont il est l'adoption (suffixe d'identifiant du
// catalogue), si l'arc est complété, et le texte de repli d'un titre sans ce preset.
type demoArcSpec struct {
	ID          string
	PresetID    string
	Completed   bool
	Title       string
	Description string
}

// demoArcSpecs : un arc en cours (Le Slayer) et un arc complété (Le Tireur d'élite).
func demoArcSpecs() []demoArcSpec {
	return []demoArcSpec{
		{
			ID: demoArcAscensionID, PresetID: demoPresetSlayer,
			Title:       "Ascension du Spartan",
			Description: "Enchaîne les objectifs pour gravir les paliers de progression.",
		},
		{
			ID: demoArcPrecisionID, PresetID: demoPresetMarksman, Completed: true,
			Title:       "Maîtrise du tir",
			Description: "Un arc court centré sur la précision et la conclusion des duels.",
		},
	}
}

// demoObjectiveFallback est le contenu d'un objectif SANS origine (titre sans catalogue) :
// une mesure du corpus, un libellé dans une seule langue, et la cible soit dérivée du corpus
// (Ratio < 1 sous le réalisé, > 1 au-dessus), soit posée (Target).
type demoObjectiveFallback struct {
	Metric   string
	Label    string
	Tier     prestige.Tier
	EvalType prestige.EvalType
	Ratio    float64
	Target   float64
}

// demoObjectiveSpec décrit un objectif démo : son modèle (suffixe d'identifiant du catalogue),
// le palier visé, son statut et, pour une mesure cumulative du corpus, le rapport de la cible au
// réalisé (0 = cible du palier du modèle).
type demoObjectiveSpec struct {
	ArcID      string
	Position   int
	TemplateID string
	Tier       prestige.Tier
	Status     prestige.ChallengeStatus
	Ratio      float64
	Fallback   demoObjectiveFallback
}

// demoObjectiveSpecs : 9 objectifs couvrant tous les stades du cycle de vie. L'arc en cours
// compte 2 étapes acquises et 2 en cours (« Étape 2/4 »), l'arc complété 3 étapes acquises ;
// hors arc, un objectif expiré et un objectif abandonné. Le repli de chaque objectif est son
// ancien contenu à mesures du corpus.
func demoObjectiveSpecs() []demoObjectiveSpec {
	done, active := prestige.StatusCompleted, prestige.StatusActive
	return []demoObjectiveSpec{
		{ArcID: demoArcAscensionID, Position: 0, TemplateID: "daily.kdr_session",
			Tier: prestige.TierNormal, Status: done,
			Fallback: demoObjectiveFallback{Metric: metricMatchesPlayed, Label: "Mise en jambe — parties jouées",
				Tier: prestige.TierNormal, Ratio: 0.5}},
		{ArcID: demoArcAscensionID, Position: 1, TemplateID: "weekly.kda_3sessions",
			Tier: prestige.TierHeroic, Status: done,
			Fallback: demoObjectiveFallback{Metric: metricKills, Label: "Montée en puissance — éliminations",
				Tier: prestige.TierHeroic, Ratio: 0.55}},
		{ArcID: demoArcAscensionID, Position: 2, TemplateID: "weekly.kills_vs_expected",
			Tier: prestige.TierLegendary, Status: active,
			Fallback: demoObjectiveFallback{Metric: metricHeadshotKills, Label: "Précision létale — tirs à la tête",
				Tier: prestige.TierHeroic, Ratio: 1.30}},
		{ArcID: demoArcAscensionID, Position: 3, TemplateID: "weekly.damage_per_min",
			Tier: prestige.TierHeroic, Status: active,
			Fallback: demoObjectiveFallback{Metric: metricWins, Label: "Série victorieuse — victoires",
				Tier: prestige.TierLegendary, Ratio: 1.25}},
		{ArcID: demoArcPrecisionID, Position: 0, TemplateID: "weekly.accuracy_3sessions",
			Tier: prestige.TierHeroic, Status: done,
			Fallback: demoObjectiveFallback{Metric: "accuracy", Label: "Viser juste — précision moyenne",
				Tier: prestige.TierHeroic, EvalType: prestige.EvalThreshold, Target: 0.45}},
		{ArcID: demoArcPrecisionID, Position: 1, TemplateID: "monthly.accuracy_30d",
			Tier: prestige.TierLegendary, Status: done,
			Fallback: demoObjectiveFallback{Metric: metricKills, Label: "Conclure les duels — éliminations",
				Tier: prestige.TierNormal, Ratio: 0.35}},
		{ArcID: demoArcPrecisionID, Position: 2, TemplateID: "weekly.headshots_3sessions",
			Tier: prestige.TierNormal, Status: done,
			Fallback: demoObjectiveFallback{Metric: metricAssists, Label: "Jouer collectif — assistances",
				Tier: prestige.TierHeroic, Ratio: 0.40}},
		{TemplateID: "weekly.kda_3sessions", Tier: prestige.TierLegendary, Status: prestige.StatusExpired,
			Fallback: demoObjectiveFallback{Metric: "kda", Label: "Tenir la moyenne — FDA sur 10 parties",
				Tier: prestige.TierLegendary, EvalType: prestige.EvalThreshold, Target: 1.8}},
		{TemplateID: "monthly.matches_played", Tier: prestige.TierMythic, Status: prestige.StatusAbandoned, Ratio: 2.0,
			Fallback: demoObjectiveFallback{Metric: metricMatchesPlayed, Label: "Marathon hebdomadaire — parties jouées",
				Tier: prestige.TierMythic, Ratio: 2.0}},
	}
}

// buildDemoArcsAndChallenges construit les arcs et les objectifs du plan depuis le catalogue du
// titre et le corpus. Les cibles cumulatives du corpus sont dérivées de lui (sous le réalisé pour
// un objectif complété, au-dessus pour un objectif encore en cours ou abandonné) ; les autres
// reprennent la cible du palier du modèle, comme une création réelle.
func buildDemoArcsAndChallenges(t prestige.Tuning, cat demoPrestigeCatalog,
	st demoCorpusStats) ([]demoPrestigeArc, []demoPrestigeChallenge) {
	arcStart, end := st.First, st.Last
	done := demoNotBefore(end.Add(-72*time.Hour), arcStart)
	// Bornées au début du corpus : sur une fenêtre courte (fixtures de test), un simple retrait
	// de N heures daterait un objectif AVANT le premier match démo.
	terminal := map[prestige.ChallengeStatus]*time.Time{
		prestige.StatusCompleted: &done,
		prestige.StatusExpired:   timePtr(demoNotBefore(end.Add(-120*time.Hour), arcStart)),
		prestige.StatusAbandoned: timePtr(demoNotBefore(end.Add(-48*time.Hour), arcStart)),
	}

	arcs := make([]demoPrestigeArc, 0, len(demoArcSpecs()))
	for _, a := range demoArcSpecs() {
		var completedAt *time.Time
		if a.Completed {
			completedAt = &done
		}
		arcs = append(arcs, resolveDemoArc(cat, a, arcStart, completedAt))
	}

	specs := demoObjectiveSpecs()
	out := make([]demoPrestigeChallenge, 0, len(specs))
	for i, s := range specs {
		c := resolveDemoObjective(cat, s, st)
		c.Status = s.Status
		c.TerminalAt = terminal[s.Status]
		out = append(out, finalizeDemoChallenge(t, c, i, arcStart))
	}
	return arcs, out
}

// resolveDemoArc rend l'arc démo : copie du preset (titre et description de départ) quand le
// catalogue le porte, sinon texte de repli sans origine (jamais un preset_id inventé).
func resolveDemoArc(cat demoPrestigeCatalog, a demoArcSpec, created time.Time,
	completedAt *time.Time) demoPrestigeArc {
	arc := demoPrestigeArc{ID: a.ID, CreatedAt: created, CompletedAt: completedAt}
	if p, ok := cat.preset(a.PresetID); ok {
		arc.Title, arc.Description = p.StoredTitle(), p.StoredDescription()
		arc.IsPreset, arc.PresetID = true, p.ID
		return arc
	}
	arc.Title, arc.Description = a.Title, a.Description
	return arc
}

// resolveDemoObjective rend l'objectif démo tel qu'une création depuis son modèle l'écrirait
// (service.AdoptPresetArc), ou son repli sans origine quand le catalogue n'a pas le modèle.
func resolveDemoObjective(cat demoPrestigeCatalog, s demoObjectiveSpec,
	st demoCorpusStats) demoPrestigeChallenge {
	tmpl, ok := cat.template(s.TemplateID)
	if !ok {
		return fallbackDemoObjective(s, st)
	}
	c := demoPrestigeChallenge{
		ArcID: s.ArcID, Position: s.Position, TemplateID: tmpl.ID,
		Metric: tmpl.Metric, Label: tmpl.StoredLabel(), Tier: s.Tier,
		Cadence: tmpl.Cadence, EvalType: tmpl.EvalType,
		WindowType: tmpl.WindowType, WindowValue: tmpl.WindowValue,
		Target: demoTierTarget(tmpl, s.Tier),
	}
	if s.Ratio != 0 && tmpl.EvalType == prestige.EvalCumulative {
		if v, known := demoCorpusTarget(st, tmpl.Metric, s.Ratio); known {
			c.Target = v
		}
	}
	return c
}

// fallbackDemoObjective rend l'objectif sans origine : texte et mesure du repli.
func fallbackDemoObjective(s demoObjectiveSpec, st demoCorpusStats) demoPrestigeChallenge {
	f := s.Fallback
	c := demoPrestigeChallenge{
		ArcID: s.ArcID, Position: s.Position, Metric: f.Metric, Label: f.Label,
		Tier: f.Tier, EvalType: f.EvalType, Target: f.Target,
	}
	if f.Ratio != 0 {
		if v, known := demoCorpusTarget(st, f.Metric, f.Ratio); known {
			c.Target = v
		}
	}
	return c
}

// demoTierTarget : cible du palier d'un modèle (même lecture que la création réelle).
func demoTierTarget(t prestige.Template, tier prestige.Tier) float64 {
	switch tier {
	case prestige.TierHeroic:
		return t.HeroicTarget
	case prestige.TierLegendary:
		return t.LegendaryTarget
	case prestige.TierMythic:
		return t.MythicTarget
	}
	return t.NormalTarget
}

// demoCorpusTarget dérive une cible du réalisé du corpus pour une mesure cumulative : sous le
// réalisé quand ratio < 1, au-dessus sinon. known=false pour une mesure que le corpus ne cumule
// pas (moyennes, mesures de session).
func demoCorpusTarget(st demoCorpusStats, metric string, ratio float64) (target float64, known bool) {
	var realized int
	switch metric {
	case metricMatchesPlayed:
		realized = st.Matches
	case metricKills:
		realized = st.Kills
	case metricAssists:
		realized = st.Assists
	case metricHeadshotKills:
		realized = st.HeadshotKills
	case metricWins:
		realized = st.Wins
	default:
		return 0, false
	}
	if ratio < 1 {
		return demoTargetBelow(realized, ratio), true
	}
	return demoTargetAbove(realized, ratio), true
}

// finalizeDemoChallenge complète les champs dérivés d'un objectif démo (id, fenêtre et cadence
// par défaut d'un objectif sans origine, PP). NB : la table `challenge` n'a PAS de colonne
// d'échéance — l'expiration est portée par le statut + expired_at (cf. steps_player_base.go).
func finalizeDemoChallenge(t prestige.Tuning, c demoPrestigeChallenge, i int,
	arcStart time.Time) demoPrestigeChallenge {
	c.ID = fmt.Sprintf("demo-objective-%02d", i+1)
	c.CreatedAt = arcStart
	if c.EvalType == "" {
		c.EvalType = prestige.EvalCumulative
	}
	if c.Cadence == "" {
		c.Cadence = prestige.CadenceFree
		c.WindowType = prestige.WindowLastNMatches
		c.WindowValue = "20"
	}
	c.PP = prestige.PPForCompletion(t, c.Tier, false, prestige.DataFull)
	return c
}
