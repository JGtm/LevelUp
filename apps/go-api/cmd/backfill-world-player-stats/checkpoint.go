//go:build cgo

package main

// checkpoint.go — reprise du backfill world : un fichier JSON mémorise par saison
// les gamertags déjà traités (succès dans "done", échecs dans "failed"). Permet de
// relancer la même commande sans re-scanner les joueurs faits, et d'empêcher un
// échec persistant de rebloquer indéfiniment une saison (il est compté comme tenté).

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

type seasonProgress struct {
	Done      []string `json:"done"`
	Failed    []string `json:"failed,omitempty"`
	Completed bool     `json:"completed"`
}

type checkpoint struct {
	Seasons map[string]*seasonProgress `json:"seasons"`
	// ResolvedXUIDs : associations gamertag->xuid déjà résolues (PeopleHub), persistées
	// entre runs. Sans ça on re-résout les MÊMES joueurs à chaque run (les tops jouent
	// plusieurs saisons → fort recouvrement) et on rebrûle le quota PeopleHub. Le cache
	// les réutilise → résolution une seule fois par joueur, jamais re-faite.
	ResolvedXUIDs map[string]string `json:"resolved_xuids,omitempty"`
	mu            sync.Mutex
}

// resolvedXUIDsSeed retourne une COPIE des associations connues (amorce du
// CachingResolver). Thread-safe.
func (c *checkpoint) resolvedXUIDsSeed() map[string]string {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make(map[string]string, len(c.ResolvedXUIDs))
	for k, v := range c.ResolvedXUIDs {
		out[k] = v
	}
	return out
}

// setResolvedXUID mémorise une NOUVELLE association gamertag->xuid (persistée au save).
func (c *checkpoint) setResolvedXUID(gamertag, xuid string) {
	if gamertag == "" || xuid == "" {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.ResolvedXUIDs == nil {
		c.ResolvedXUIDs = map[string]string{}
	}
	c.ResolvedXUIDs[gamertag] = xuid
}

// loadCheckpoint lit le fichier de reprise (vide si absent ou si force).
func loadCheckpoint(path string, force bool) *checkpoint {
	cp := &checkpoint{Seasons: map[string]*seasonProgress{}}
	if force {
		return cp
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return cp
	}
	_ = json.Unmarshal(b, cp)
	if cp.Seasons == nil {
		cp.Seasons = map[string]*seasonProgress{}
	}
	return cp
}

func (c *checkpoint) get(season string) *seasonProgress {
	sp := c.Seasons[season]
	if sp == nil {
		sp = &seasonProgress{}
		c.Seasons[season] = sp
	}
	return sp
}

func (c *checkpoint) completed(season string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	sp := c.Seasons[season]
	return sp != nil && sp.Completed
}

// remaining retourne les gamertags non encore traités (ordre stable), tronqué à
// limit (0 = pas de limite). La limite s'applique au total de la saison. Les
// joueurs DÉJÀ tentés (done ∪ failed) sont exclus : sans ça un échec persistant
// (ex. gros historique qui 429 en boucle, xuid non résolu) rebloque la saison à
// chaque reprise. retryFailed ré-inclut les échecs pour une tentative explicite.
func (c *checkpoint) remaining(season string, gamertags []string, limit int, retryFailed bool) []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	sp := c.get(season)
	skip := map[string]bool{}
	for _, gt := range sp.Done {
		skip[gt] = true
	}
	if !retryFailed {
		for _, gt := range sp.Failed {
			skip[gt] = true
		}
	}
	pool := gamertags
	if limit > 0 && limit < len(pool) {
		pool = pool[:limit]
	}
	var out []string
	for _, gt := range pool {
		if !skip[gt] {
			out = append(out, gt)
		}
	}
	return out
}

// doneCount compte les gamertags de la saison déjà traités (intersection avec le
// checkpoint). Sert à un affichage de progression correct (≠ exclus par -limit).
func (c *checkpoint) doneCount(season string, gamertags []string) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	sp := c.Seasons[season]
	if sp == nil {
		return 0
	}
	done := make(map[string]bool, len(sp.Done))
	for _, gt := range sp.Done {
		done[gt] = true
	}
	n := 0
	for _, gt := range gamertags {
		if done[gt] {
			n++
		}
	}
	return n
}

// attemptedCount compte les gamertags de la saison déjà TENTÉS (done ∪ failed),
// intersection avec les gamertags courants. La complétude d'une saison se décide
// là-dessus (pas sur les seuls succès) : une saison est complète quand tous ses
// joueurs ont été tentés, même si quelques-uns ont échoué (échecs acceptés).
func (c *checkpoint) attemptedCount(season string, gamertags []string) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	sp := c.Seasons[season]
	if sp == nil {
		return 0
	}
	attempted := make(map[string]bool, len(sp.Done)+len(sp.Failed))
	for _, gt := range sp.Done {
		attempted[gt] = true
	}
	for _, gt := range sp.Failed {
		attempted[gt] = true
	}
	n := 0
	for _, gt := range gamertags {
		if attempted[gt] {
			n++
		}
	}
	return n
}

func (c *checkpoint) markDone(season string, gamertags []string) {
	if len(gamertags) == 0 {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	sp := c.get(season)
	sp.Done = append(sp.Done, gamertags...)
	// Un joueur re-tenté avec succès (-retry-failed) sort de la liste des échecs.
	if len(sp.Failed) > 0 {
		nowDone := make(map[string]bool, len(gamertags))
		for _, gt := range gamertags {
			nowDone[gt] = true
		}
		kept := sp.Failed[:0]
		for _, gt := range sp.Failed {
			if !nowDone[gt] {
				kept = append(kept, gt)
			}
		}
		sp.Failed = kept
	}
}

// markFailed enregistre des gamertags tentés mais en échec (dé-dupliqué). Ils ne
// sont plus re-tentés à la reprise (sauf -retry-failed) et comptent vers la
// complétude de la saison — un échec persistant ne rebloque plus le backfill.
func (c *checkpoint) markFailed(season string, gamertags []string) {
	if len(gamertags) == 0 {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	sp := c.get(season)
	existing := make(map[string]bool, len(sp.Failed))
	for _, gt := range sp.Failed {
		existing[gt] = true
	}
	for _, gt := range gamertags {
		if !existing[gt] {
			sp.Failed = append(sp.Failed, gt)
			existing[gt] = true
		}
	}
}

func (c *checkpoint) markCompleted(season string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.get(season).Completed = true
}

// save écrit le checkpoint de façon atomique (tmp + rename).
func (c *checkpoint) save(path string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	if dir := filepath.Dir(path); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
