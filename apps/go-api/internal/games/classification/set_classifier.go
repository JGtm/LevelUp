package classification

import "strings"

// SetClassifier — stratégie #1 : classification par APPARTENANCE à des sets
// autoritatifs d'ids de playlist. Réutilisable data-only par tout titre publiant
// ses listes d'ids classées / PvE (cf. doc package + handoff §7).
//
// Sémantique des verdicts (CONSERVATRICE — ne jamais minter un faux « classé ») :
//   - set VIDE (aucune donnée autoritative encore publiée) → nil pour tout id
//     (INDETERMINE). Préserve le comportement « avant data » : aucune décision.
//   - set PEUPLE → &(id ∈ set). Le set publié est réputé EXHAUSTIF : « absent du
//     set » = &false (non-classé décidé), « présent » = &true.
//   - id vide → nil (indéterminé).
type SetClassifier struct {
	ranked map[string]struct{}
	pve    map[string]struct{}
}

var _ RankedClassifier = (*SetClassifier)(nil)

// NewSetClassifier construit le classifier depuis les listes d'ids (classées, PvE).
// Trim + dédup ; slices vides → sets nil (verdicts nil). Récepteur nil toléré par
// les méthodes (verdicts nil) — un titre sans config se dégrade proprement.
func NewSetClassifier(rankedIDs, pveIDs []string) *SetClassifier {
	return &SetClassifier{
		ranked: toSet(rankedIDs),
		pve:    toSet(pveIDs),
	}
}

func toSet(ids []string) map[string]struct{} {
	if len(ids) == 0 {
		return nil
	}
	out := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		if id = strings.TrimSpace(id); id != "" {
			out[id] = struct{}{}
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// IsRanked applique la sémantique d'appartenance au set classé (cf. SetClassifier).
func (c *SetClassifier) IsRanked(playlistID string) *bool {
	if c == nil {
		return nil
	}
	return membership(c.ranked, playlistID)
}

// IsPvE applique la sémantique d'appartenance au set PvE (cf. SetClassifier).
func (c *SetClassifier) IsPvE(playlistID string) *bool {
	if c == nil {
		return nil
	}
	return membership(c.pve, playlistID)
}

// membership : len(set)==0 → nil (pas de donnée autoritative) ; sinon &(id ∈ set).
func membership(set map[string]struct{}, id string) *bool {
	if len(set) == 0 {
		return nil
	}
	if id = strings.TrimSpace(id); id == "" {
		return nil
	}
	_, ok := set[id]
	return &ok
}
