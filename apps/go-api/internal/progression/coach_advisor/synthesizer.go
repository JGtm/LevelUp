package coach_advisor

import (
	"encoding/hex"
	"fmt"
	"hash/fnv"
	"sort"
	"strconv"
	"time"

	"levelup/go-api/internal/prestige"
)

// synthesizer.go — production dynamique de prestige.Template à partir d'un
// Signal coach_advisor (cf. ADR 0028).
//
// Contraintes (cf. ADR 0028 §"Garde-fous globaux") :
//   - signal.Strength >= synthesisMinStrength (I5)
//   - (metric, window, eval_type) doit être dans la SynthesisGrammar
//   - targets contiennent les stretch ratios standards (1.08/1.25/1.50/2.00),
//     pas des valeurs absolues — Prestige matérialise via baseline au
//     CreateChallenge (I1+I2)
//   - ID déterministe via hash → dédup cross-joueurs automatique
//
// Fonction PURE — pas d'I/O, déterministe pour des entrées identiques.

// SynthesisConfig regroupe les seuils numériques du synthesizer. Valeurs par
// défaut via DefaultSynthesisConfig().
type SynthesisConfig struct {
	// MinStrength : seuil dur I5 — un signal en-dessous ne déclenche jamais
	// de synthèse (ErrSignalTooWeak). Défaut 0.6.
	MinStrength float64
	// StretchFactors : stretch ratios appliqués à la baseline par Prestige
	// au moment du CreateChallenge. Ordre : Normal, Heroic, Legendary, Mythic.
	// Défaut : [1.08, 1.25, 1.50, 2.00] (aligné sur prestige/tuning.toml).
	StretchFactors [4]float64
}

// DefaultSynthesisConfig retourne les valeurs canoniques de l'ADR.
func DefaultSynthesisConfig() SynthesisConfig {
	return SynthesisConfig{
		MinStrength:    0.6,
		StretchFactors: [4]float64{1.08, 1.25, 1.50, 2.00},
	}
}

// Synthesizer produit des prestige.Template depuis des Signals. Stateless —
// la configuration et la grammaire sont injectées au constructeur.
type Synthesizer struct {
	grammar SynthesisGrammar
	cfg     SynthesisConfig
}

// NewSynthesizer construit un Synthesizer avec la grammaire et la config
// fournies. Utiliser DefaultSynthesisConfig() pour les valeurs canoniques.
func NewSynthesizer(grammar SynthesisGrammar, cfg SynthesisConfig) *Synthesizer {
	return &Synthesizer{grammar: grammar, cfg: cfg}
}

// Synthesize produit un prestige.Template depuis un Signal.
//
// Erreurs :
//   - ErrSignalTooWeak si signal.Strength < cfg.MinStrength (I5).
//   - ErrMetricNotSynthesizable si aucune combinaison metric+window+eval_type
//     ne match la grammaire pour la métrique inférée du signal.
//
// Fenêtre choisie : la première de la grammaire pour cette métrique (V2 — un
// arbitrage plus malin viendra avec V2.1 selon kind/nature du signal).
func (s *Synthesizer) Synthesize(sig Signal, titleSlug string, now time.Time) (prestige.Template, error) {
	if sig.Strength < s.cfg.MinStrength {
		return prestige.Template{}, fmt.Errorf("%w: strength=%f, min=%f",
			ErrSignalTooWeak, sig.Strength, s.cfg.MinStrength)
	}
	metric := inferMetric(sig)
	if metric == "" {
		return prestige.Template{}, fmt.Errorf("%w: signal kind=%q has no inferred metric",
			ErrMetricNotSynthesizable, sig.Kind)
	}

	allowed := s.grammar.AllowedWindows(metric)
	if len(allowed) == 0 {
		return prestige.Template{}, fmt.Errorf("%w: metric=%q", ErrMetricNotSynthesizable, metric)
	}

	// V2 : prend la première combinaison autorisée. Une heuristique plus fine
	// (préférer rolling_days pour LOWESS, session pour patterns combat, etc.)
	// viendra en V2.1 si la télémétrie le justifie.
	choice := allowed[0]

	lusrComponents := nonEmptyList(sig.LUSRComponent)
	radarAxes := nonEmptyList(sig.RadarAxis)
	cadence := cadenceForWindow(choice.WindowType, choice.WindowValue)

	t := prestige.Template{
		ID:              templateIDFromComponents(metric, choice.WindowType, choice.WindowValue, cadence, choice.EvalType, lusrComponents, radarAxes),
		TitleSlug:       titleSlug,
		Metric:          metric,
		WindowType:      prestige.WindowType(choice.WindowType),
		WindowValue:     choice.WindowValue,
		Cadence:         prestige.Cadence(cadence),
		EvalType:        prestige.EvalType(choice.EvalType),
		ModeFilter:      "universal",
		LabelEN:         synthesizedLabelEN(metric, choice.WindowType, choice.WindowValue),
		LabelFR:         synthesizedLabelFR(metric, choice.WindowType, choice.WindowValue),
		DescriptionEN:   synthesizedDescriptionEN(metric, choice.WindowType, choice.WindowValue),
		DescriptionFR:   synthesizedDescriptionFR(metric, choice.WindowType, choice.WindowValue),
		NormalTarget:    s.cfg.StretchFactors[0], // stretch ratio (cf. I1/I2)
		HeroicTarget:    s.cfg.StretchFactors[1],
		LegendaryTarget: s.cfg.StretchFactors[2],
		MythicTarget:    s.cfg.StretchFactors[3],
		LUSRComponents:  lusrComponents,
		RadarAxes:       radarAxes,
		IsLongTerm:      isLongTermWindow(choice.WindowType, choice.WindowValue),
		Source:          "coach_synthesized",
		SchemaVersion:   1,
		UpdatedAt:       now.UTC(),
	}
	return t, nil
}

// inferMetric retourne la métrique principale pour un signal.
//
// Précédence : Metric explicite (record/milestone) > LUSRComponent
// (LOWESS/combat). Retourne "" si le signal n'a ni l'un ni l'autre.
func inferMetric(s Signal) string {
	if s.Metric != "" {
		return s.Metric
	}
	if s.LUSRComponent != "" {
		return s.LUSRComponent
	}
	return ""
}

// cadenceForWindow déduit une cadence raisonnable d'une fenêtre.
//
//   - session                       → daily   (suggérer après chaque session)
//   - last_n_matches:N              → daily   (signal court, suivi rapide)
//   - rolling_days:N (N ≤ 7)        → daily
//   - rolling_days:N (8 ≤ N ≤ 31)   → weekly
//   - rolling_days:N (N > 31)       → monthly
//   - défaut                        → weekly
func cadenceForWindow(windowType, windowValue string) string {
	switch windowType {
	case "session":
		return "daily"
	case "last_n_matches":
		return "daily"
	case "rolling_days":
		n, err := strconv.Atoi(windowValue)
		if err != nil {
			return "weekly"
		}
		switch {
		case n <= 7:
			return "daily"
		case n <= 31:
			return "weekly"
		default:
			return "monthly"
		}
	}
	return "weekly"
}

// isLongTermWindow retourne true pour les fenêtres rolling_days ≥ 14 ou
// last_n_matches ≥ 20 — utilisé par le PlayerProfile pour favoriser ces
// templates en campagne durable (cohérent avec catalog_loader.go).
func isLongTermWindow(windowType, windowValue string) bool {
	n, _ := strconv.Atoi(windowValue)
	switch windowType {
	case "rolling_days":
		return n >= 14
	case "last_n_matches":
		return n >= 20
	}
	return false
}

// nonEmptyList retourne []string{s} si s != "", sinon nil.
func nonEmptyList(s string) []string {
	if s == "" {
		return nil
	}
	return []string{s}
}

// templateIDFromComponents calcule un ID déterministe préfixé "syn_" suivi
// d'un hash FNV-64 hex (16 chars). Le hash garantit la dédup cross-joueurs :
// deux signaux qui produisent les mêmes composantes → même ID → UPSERT
// idempotent côté Prestige.
func templateIDFromComponents(metric, windowType, windowValue, cadence, evalType string, lusrComponents, radarAxes []string) string {
	h := fnv.New64a()
	// Sort pour déterminisme indépendamment de l'ordre des listes.
	lusrSorted := append([]string(nil), lusrComponents...)
	sort.Strings(lusrSorted)
	radarSorted := append([]string(nil), radarAxes...)
	sort.Strings(radarSorted)
	for _, s := range []string{
		metric, windowType, windowValue, cadence, evalType,
		joinPipe(lusrSorted), joinPipe(radarSorted),
	} {
		_, _ = h.Write([]byte(s))
		_, _ = h.Write([]byte{0}) // séparateur pour éviter les collisions de concaténation
	}
	return "syn_" + hex.EncodeToString(h.Sum(nil))
}

func joinPipe(items []string) string {
	out := ""
	for i, s := range items {
		if i > 0 {
			out += "|"
		}
		out += s
	}
	return out
}

// ─── Labels paramétrés (i18n minimal en attendant i18n service complet) ───

func synthesizedLabelEN(metric, windowType, windowValue string) string {
	return fmt.Sprintf("Improve %s over %s", humanMetric(metric), humanWindow(windowType, windowValue, "en"))
}

func synthesizedLabelFR(metric, windowType, windowValue string) string {
	return fmt.Sprintf("Améliore %s sur %s", humanMetric(metric), humanWindow(windowType, windowValue, "fr"))
}

func synthesizedDescriptionEN(metric, windowType, windowValue string) string {
	return fmt.Sprintf("Push your %s above your personal baseline over %s.",
		humanMetric(metric), humanWindow(windowType, windowValue, "en"))
}

func synthesizedDescriptionFR(metric, windowType, windowValue string) string {
	return fmt.Sprintf("Pousse ton %s au-dessus de ta moyenne personnelle sur %s.",
		humanMetric(metric), humanWindow(windowType, windowValue, "fr"))
}

func humanMetric(m string) string {
	switch m {
	case "kda":
		return "KDA"
	case "kills_vs_expected":
		return "kills vs expected"
	case "deaths_vs_expected":
		return "deaths vs expected"
	case "accuracy_delta":
		return "accuracy delta"
	case "assists_per_kill":
		return "assists per kill"
	case "performance_score":
		return "performance score"
	}
	// Fallback : kebab/snake → words (basique, suffisant pour labels neutres)
	return m
}

func humanWindow(windowType, windowValue, lang string) string {
	if windowType == "session" {
		if lang == "fr" {
			return "une session"
		}
		return "a session"
	}
	if windowType == "last_n_matches" {
		if lang == "fr" {
			return fmt.Sprintf("les %s derniers matchs", windowValue)
		}
		return fmt.Sprintf("the last %s matches", windowValue)
	}
	if windowType == "rolling_days" {
		if lang == "fr" {
			return fmt.Sprintf("%s jours", windowValue)
		}
		return fmt.Sprintf("%s days", windowValue)
	}
	return windowType
}
