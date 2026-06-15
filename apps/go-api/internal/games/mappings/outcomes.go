package mappings

import (
	"fmt"
	"sort"

	"levelup/go-api/internal/games/canonical"
)

// OutcomeMapping est la projection d'une section [outcomes.{key}] du TOML.
//
// Les outcomes décrivent les issues de match canoniques (win, loss, tie, dnf).
// La couleur est exposée via un token du design system (`color_token`) plutôt
// qu'un hex direct, pour que le frontend résolve via son thème.
type OutcomeMapping struct {
	Key        string            // "win" | "loss" | "tie" | "dnf"
	Labels     map[string]string // locale → libellé
	ColorToken string            // ex : "outcome.positive" | "outcome.negative" | "outcome.neutral"
	RawCode    int               // MT-06 : code brut du titre (Halo : win=2…). 0 = absent.
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
	byRawCode     map[int]string // MT-06 : raw_code (non nul) → key. Sens int→canonique.
}

// NewOutcomeMappingSet construit un OutcomeMappingSet (utilisé par le loader et
// les tests). Les arguments sont consommés tels quels ; l'index inverse
// raw_code→key (MT-06) est dérivé des RawCode non nuls.
func NewOutcomeMappingSet(titleSlug string, schemaVersion int, byKey map[string]OutcomeMapping) *OutcomeMappingSet {
	if byKey == nil {
		byKey = make(map[string]OutcomeMapping)
	}
	byRawCode := make(map[int]string, len(byKey))
	for key, o := range byKey {
		if o.RawCode != 0 {
			byRawCode[o.RawCode] = key
		}
	}
	return &OutcomeMappingSet{
		titleSlug:     titleSlug,
		schemaVersion: schemaVersion,
		byKey:         byKey,
		byRawCode:     byRawCode,
	}
}

// Canonical traduit un code brut du titre (ex. Halo 2/3/1/4) vers l'Outcome
// canonique (MT-06). Retourne (_, false) si le code n'est pas mappé (titre sans
// raw_code, ou code inconnu) → le caller dégrade.
func (s *OutcomeMappingSet) Canonical(rawCode int) (canonical.Outcome, bool) {
	if s == nil {
		return "", false
	}
	key, ok := s.byRawCode[rawCode]
	if !ok {
		return "", false
	}
	return canonical.Outcome(key), true
}

// RawCode traduit un Outcome canonique vers le code brut du titre (MT-06).
// Retourne (0, false) si l'outcome n'a pas de raw_code pour ce titre.
func (s *OutcomeMappingSet) RawCode(o canonical.Outcome) (int, bool) {
	if s == nil {
		return 0, false
	}
	m, ok := s.byKey[string(o)]
	if !ok || m.RawCode == 0 {
		return 0, false
	}
	return m.RawCode, true
}

// SQLIsWinExpr construit l'expression SQL title-aware « la colonne vaut le code
// d'un win » (MT-06) — ex. `outcome = 2` pour Halo. Le repo ne connaît plus le
// littéral. Retourne (_, false) si le titre n'a pas de raw_code pour win → le
// caller dégrade (skip filtre ou fallback). `col` doit être un nom de colonne
// de confiance (pas une entrée utilisateur) ; seul un entier est interpolé.
func (s *OutcomeMappingSet) SQLIsWinExpr(col string) (string, bool) {
	code, ok := s.RawCode(canonical.OutcomeWin)
	if !ok {
		return "", false
	}
	return fmt.Sprintf("%s = %d", col, code), true
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
