package mappings

import "sort"

// OutcomeMapping est la projection d'une section [outcomes.{key}] du TOML.
//
// Les outcomes décrivent les issues de match canoniques (win, loss, tie, dnf).
// La couleur est exposée via un token du design system (`color_token`) plutôt
// qu'un hex direct, pour que le frontend résolve via son thème.
type OutcomeMapping struct {
	Key        string            // "win" | "loss" | "tie" | "dnf"
	Labels     map[string]string // locale → libellé
	ColorToken string            // ex : "outcome.positive" | "outcome.negative" | "outcome.neutral"
}

// Label retourne le libellé pour la locale demandée + fallback locale → en → key.
func (o OutcomeMapping) Label(locale string) (label string, usedFallback bool) {
	if v, ok := o.Labels[locale]; ok && v != "" {
		return v, false
	}
	if v, ok := o.Labels[LocaleEN]; ok && v != "" {
		return v, true
	}
	return o.Key, true
}

// OutcomeMappingSet est l'ensemble des OutcomeMapping d'un titre.
type OutcomeMappingSet struct {
	titleSlug     string
	schemaVersion int
	byKey         map[string]OutcomeMapping
}

// NewOutcomeMappingSet construit un OutcomeMappingSet (utilisé par le loader et
// les tests). Les arguments sont consommés tels quels.
func NewOutcomeMappingSet(titleSlug string, schemaVersion int, byKey map[string]OutcomeMapping) *OutcomeMappingSet {
	if byKey == nil {
		byKey = make(map[string]OutcomeMapping)
	}
	return &OutcomeMappingSet{
		titleSlug:     titleSlug,
		schemaVersion: schemaVersion,
		byKey:         byKey,
	}
}

// TitleSlug retourne le slug du titre porteur.
func (s *OutcomeMappingSet) TitleSlug() string { return s.titleSlug }

// SchemaVersion retourne la version du schéma TOML.
func (s *OutcomeMappingSet) SchemaVersion() int { return s.schemaVersion }

// Get retourne le mapping d'un outcome s'il existe.
func (s *OutcomeMappingSet) Get(key string) (OutcomeMapping, bool) {
	if s == nil {
		return OutcomeMapping{}, false
	}
	o, ok := s.byKey[key]
	return o, ok
}

// All retourne tous les outcomes triés par key.
func (s *OutcomeMappingSet) All() []OutcomeMapping {
	if s == nil {
		return nil
	}
	out := make([]OutcomeMapping, 0, len(s.byKey))
	for _, o := range s.byKey {
		out = append(out, o)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out
}

// Keys retourne les keys présentes dans cet ensemble.
func (s *OutcomeMappingSet) Keys() []string {
	if s == nil {
		return nil
	}
	out := make([]string, 0, len(s.byKey))
	for k := range s.byKey {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
