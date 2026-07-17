package coach_advisor

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

// synthesis_grammar.go — chargement et représentation de l'allowlist
// (config/coach_advisor/synthesis_grammar.toml), cf. ADR 0028.

// ErrMetricNotSynthesizable est retournée par le synthesizer si une
// combinaison (metric, window_type, window_value, eval_type) n'est pas
// présente dans l'allowlist.
var ErrMetricNotSynthesizable = errors.New("coach_advisor: metric/window/eval not in synthesis grammar")

// ErrSignalTooWeak est retournée par le synthesizer si signal.Strength
// est en-dessous de tuning.SynthesisMinStrength (invariant I5 ADR 0020).
var ErrSignalTooWeak = errors.New("coach_advisor: signal strength below synthesis_min_strength")

// SynthesisGrammar définit ce que le synthesizer peut produire.
//
// Construite via LoadSynthesisGrammar(toml_path) ou DefaultSynthesisGrammar()
// (vide — refuse toute synthèse).
type SynthesisGrammar struct {
	// keyed by metric → []allowedWindowEval
	rules map[string][]allowedWindowEval
}

// allowedWindowEval est une combinaison (window_type, window_value, eval_type)
// autorisée pour une métrique donnée.
type allowedWindowEval struct {
	WindowType  string
	WindowValue string
	EvalType    string
}

// DefaultSynthesisGrammar retourne une grammaire vide — toute synthèse est
// refusée. Utiliser LoadSynthesisGrammar pour charger l'allowlist réelle.
func DefaultSynthesisGrammar() SynthesisGrammar {
	return SynthesisGrammar{rules: map[string][]allowedWindowEval{}}
}

// IsAllowed retourne true si (metric, window_type, window_value, eval_type)
// est dans l'allowlist. window_value peut être vide (cas "session").
func (g SynthesisGrammar) IsAllowed(metric, windowType, windowValue, evalType string) bool {
	allowed, ok := g.rules[metric]
	if !ok {
		return false
	}
	for _, a := range allowed {
		if a.WindowType == windowType && a.WindowValue == windowValue && a.EvalType == evalType {
			return true
		}
	}
	return false
}

// AllowedWindows retourne la liste des combinaisons autorisées pour cette
// métrique. Permet au synthesizer de "choisir" la fenêtre adaptée au signal
// quand celui-ci n'en impose pas une explicitement (V2 : on prend la première).
func (g SynthesisGrammar) AllowedWindows(metric string) []allowedWindowEval {
	allowed := g.rules[metric]
	out := make([]allowedWindowEval, len(allowed))
	copy(out, allowed)
	return out
}

// Metrics liste les métriques connues de la grammaire (utile pour tests
// d'introspection).
func (g SynthesisGrammar) Metrics() []string {
	out := make([]string, 0, len(g.rules))
	for k := range g.rules {
		out = append(out, k)
	}
	return out
}

// WindowSpecs retourne les fenêtres autorisées pour une métrique sous la forme
// "window_type:window_value" (ou "session" pour une fenêtre sans valeur), telle
// qu'écrite dans synthesis_grammar.toml. Accesseur d'introspection externe :
// permet à l'analyseur de tuning (cmd/prestige-tuning-analyze) de relier une
// recommandation aux entrées réelles de la grammaire sans re-parser le TOML.
// Retourne nil si la métrique est absente de la grammaire.
func (g SynthesisGrammar) WindowSpecs(metric string) []string {
	allowed, ok := g.rules[metric]
	if !ok {
		return nil
	}
	out := make([]string, 0, len(allowed))
	for _, a := range allowed {
		if a.WindowValue == "" {
			out = append(out, a.WindowType)
			continue
		}
		out = append(out, a.WindowType+":"+a.WindowValue)
	}
	return out
}

// ─── Loader TOML ───

// synthesisGrammarTOML est la projection brute du fichier.
type synthesisGrammarTOML struct {
	Allow []synthesisAllowEntryTOML `toml:"allow"`
}

type synthesisAllowEntryTOML struct {
	Metric   string   `toml:"metric"`
	Windows  []string `toml:"windows"` // "window_type:window_value" ou "session" (window_value vide)
	EvalType string   `toml:"eval_type"`
}

// LoadSynthesisGrammar parse un fichier TOML d'allowlist.
//
// Format : voir config/coach_advisor/synthesis_grammar.toml. Erreur si :
//   - le fichier n'existe pas / n'est pas lisible
//   - le TOML est invalide
//   - une entrée a un metric ou eval_type vide
//   - une window est mal formée (attendu "type:value" ou juste "session")
func LoadSynthesisGrammar(path string) (SynthesisGrammar, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return SynthesisGrammar{}, fmt.Errorf("coach_advisor: load grammar %q: %w", path, err)
	}
	var raw synthesisGrammarTOML
	if err := toml.Unmarshal(data, &raw); err != nil {
		return SynthesisGrammar{}, fmt.Errorf("coach_advisor: parse grammar %q: %w", path, err)
	}
	g := SynthesisGrammar{rules: map[string][]allowedWindowEval{}}
	for i, e := range raw.Allow {
		if e.Metric == "" {
			return SynthesisGrammar{}, fmt.Errorf("coach_advisor: grammar entry #%d has empty metric", i)
		}
		if e.EvalType == "" {
			return SynthesisGrammar{}, fmt.Errorf("coach_advisor: grammar entry #%d (metric=%s) has empty eval_type", i, e.Metric)
		}
		for _, w := range e.Windows {
			wt, wv, err := parseWindowSpec(w)
			if err != nil {
				return SynthesisGrammar{}, fmt.Errorf("coach_advisor: grammar entry #%d (metric=%s) window %q: %w", i, e.Metric, w, err)
			}
			g.rules[e.Metric] = append(g.rules[e.Metric], allowedWindowEval{
				WindowType:  wt,
				WindowValue: wv,
				EvalType:    e.EvalType,
			})
		}
	}
	return g, nil
}

// parseWindowSpec parse "session" (window_value vide) ou "type:value"
// (ex: "rolling_days:14", "last_n_matches:10"). Pour "type:value", value doit
// être un entier positif.
func parseWindowSpec(spec string) (windowType, windowValue string, err error) {
	if spec == "session" {
		return "session", "", nil
	}
	parts := strings.SplitN(spec, ":", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("expected \"type:value\" or \"session\"")
	}
	if _, parseErr := strconv.Atoi(parts[1]); parseErr != nil {
		return "", "", fmt.Errorf("window_value must be int: %w", parseErr)
	}
	return parts[0], parts[1], nil
}
