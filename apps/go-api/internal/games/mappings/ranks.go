package mappings

import (
	"sort"
	"strings"
)

// RankEntry porte les libellés localisés d'un rang de carrière.
//
// Les map sont indexées par locale courte ("en", "fr", "de", ...) — la
// normalisation depuis les codes Waypoint ("fr-FR", "de-DE") est faite par le
// loader avant insertion.
type RankEntry struct {
	ID       int
	Title    map[string]string
	Subtitle map[string]string
	Tier     map[string]string
	// XPRequired : XP nécessaire pour COMPLÉTER ce rang (career_ranks.xp_required,
	// = xp_for_next_rank côté affichage). 0 si inconnu. Sert de fallback quand la
	// source (DB locale) ne stocke pas la valeur (cf. buildHomeCareerRank).
	XPRequired int
}

// FullLabel concatène title + subtitle + tier dans la locale demandée, avec
// fallback EN par segment si la locale est manquante.
//
// Retourne ("", true) si aucun segment EN n'est disponible (entrée vide).
func (e RankEntry) FullLabel(locale string) (string, bool) {
	parts := make([]string, 0, 3)
	usedFallback := false
	for _, m := range []map[string]string{e.Title, e.Subtitle, e.Tier} {
		if v, ok := m[locale]; ok && v != "" {
			parts = append(parts, v)
			continue
		}
		if v, ok := m[LocaleEN]; ok && v != "" {
			parts = append(parts, v)
			usedFallback = true
		}
	}
	return strings.Join(parts, " "), usedFallback
}

// RankCatalog est l'ensemble des RankEntry d'un titre, exposé en lecture seule.
//
// L'ordre d'itération (Next) suit l'ordre numérique de `rank_id` qui correspond
// à la progression de carrière côté Halo Waypoint.
//
// Un *RankCatalog nil est valide et se comporte comme un catalog vide : le
// chargement est best-effort (metadata absente en mode démo / titre sans rangs)
// et les consommateurs ne re-vérifient pas le pointeur.
type RankCatalog struct {
	titleSlug string
	byID      map[int]RankEntry
}

// NewRankCatalog construit un catalog à partir d'entrées (typiquement issues
// d'une lecture DB). titleSlug est purement informatif (lookup/observabilité).
func NewRankCatalog(titleSlug string, entries []RankEntry) *RankCatalog {
	byID := make(map[int]RankEntry, len(entries))
	for _, e := range entries {
		byID[e.ID] = e
	}
	return &RankCatalog{titleSlug: titleSlug, byID: byID}
}

// TitleSlug retourne le slug du titre porteur du catalog.
func (c *RankCatalog) TitleSlug() string {
	if c == nil {
		return ""
	}
	return c.titleSlug
}

// Len retourne le nombre de rangs chargés.
func (c *RankCatalog) Len() int {
	if c == nil {
		return 0
	}
	return len(c.byID)
}

// Get retourne le RankEntry pour rank_id (ok = false si absent).
func (c *RankCatalog) Get(id int) (RankEntry, bool) {
	if c == nil {
		return RankEntry{}, false
	}
	e, ok := c.byID[id]
	return e, ok
}

// Next retourne le rang suivant dans la progression (rank_id + 1).
// Retourne (zero, false) si id est le dernier rang du catalog.
func (c *RankCatalog) Next(id int) (RankEntry, bool) {
	if c == nil {
		return RankEntry{}, false
	}
	e, ok := c.byID[id+1]
	return e, ok
}

// CumulativeXPRequired somme XPRequired des rangs 1..uptoRankInclusive — soit l'XP
// totale pour ATTEINDRE le rang (uptoRankInclusive+1). Utilisé pour l'XP de carrière
// cumulée au rang max (où la progression intra-rang est nulle). 0 si le catalog n'a
// pas les seuils (XPRequired non chargé).
func (c *RankCatalog) CumulativeXPRequired(uptoRankInclusive int) int {
	if c == nil {
		return 0
	}
	total := 0
	for id := 1; id <= uptoRankInclusive; id++ {
		if e, ok := c.byID[id]; ok {
			total += e.XPRequired
		}
	}
	return total
}

// MaxRank retourne l'entrée du rang SOMMET du titre (rank_id le plus élevé chargé).
// Sert à résoudre le libellé du rang maximum (« Héros » pour Halo Infinite) sans
// coder en dur l'identifiant du rang max. Retourne (zero, false) si le catalog est
// vide/nil.
func (c *RankCatalog) MaxRank() (RankEntry, bool) {
	if c == nil || len(c.byID) == 0 {
		return RankEntry{}, false
	}
	maxID := -1
	for id := range c.byID {
		if id > maxID {
			maxID = id
		}
	}
	e, ok := c.byID[maxID]
	return e, ok
}

// FullLabel résout le libellé complet d'un rang dans la locale demandée.
// Retourne ("", false) si rank_id est absent.
func (c *RankCatalog) FullLabel(id int, locale string) (string, bool) {
	if c == nil {
		return "", false
	}
	e, ok := c.byID[id]
	if !ok {
		return "", false
	}
	label, _ := e.FullLabel(locale)
	return label, true
}

// IDs retourne la liste triée des rank_id chargés (utile pour debug/tests).
func (c *RankCatalog) IDs() []int {
	if c == nil {
		return nil
	}
	out := make([]int, 0, len(c.byID))
	for id := range c.byID {
		out = append(out, id)
	}
	sort.Ints(out)
	return out
}

// NormalizeLang collapses "fr-FR" → "fr", "de-DE" → "de", etc. Conserve "en"
// tel quel. Le préfixe avant `-` est retourné en lowercase.
//
// Exposée pour que les loaders (DB ou TOML) normalisent avant insertion.
func NormalizeLang(lang string) string {
	if i := strings.Index(lang, "-"); i > 0 {
		return strings.ToLower(lang[:i])
	}
	return strings.ToLower(lang)
}
