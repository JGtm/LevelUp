package assets

import (
	"context"
	"sync"
	"time"
)

// PlayedMapVersionLoader parcourt les couples (carte, version) des matchs enregistrés d'un
// titre (base partagée du titre) et appelle visit pour chacun.
type PlayedMapVersionLoader func(ctx context.Context, titleID string, visit func(mapID, versionID string)) error

// playedVersionsMinReload est l'intervalle minimal entre deux lectures de la base partagée
// pour un même titre.
const playedVersionsMinReload = time.Minute

// PlayedMapVersions dit si une version de carte figure dans des matchs réellement joués d'un
// titre. Les couples (carte, version) d'un titre sont tenus en mémoire ; une absence ne
// relit la base qu'au plus une fois par playedVersionsMinReload, si bien qu'une version
// inventée coûte une recherche en mémoire et jamais une lecture de la base par requête.
// Un match synchronisé depuis la dernière lecture est vu à la première absence qui suit
// l'intervalle.
type PlayedMapVersions struct {
	load      PlayedMapVersionLoader
	minReload time.Duration

	mu      sync.Mutex
	byTitle map[string]*playedVersionSet
}

// playedVersionSet est l'état d'un titre : les couples lus, la date de la dernière tentative
// de lecture et son erreur.
type playedVersionSet struct {
	pairs     map[string]struct{}
	attempted time.Time
	err       error
}

// NewPlayedMapVersions construit la source sur load.
func NewPlayedMapVersions(load PlayedMapVersionLoader) *PlayedMapVersions {
	return &PlayedMapVersions{load: load, minReload: playedVersionsMinReload, byTitle: map[string]*playedVersionSet{}}
}

func playedKey(mapID, versionID string) string { return mapID + "\x00" + versionID }

// Played dit si (mapID, versionID) figure dans un match enregistré du titre. Une erreur de
// lecture est rendue telle quelle (la dernière, jusqu'à la prochaine tentative) ; les couples
// déjà lus restent servis.
func (p *PlayedMapVersions) Played(ctx context.Context, titleID, mapID, versionID string) (bool, error) {
	key := playedKey(mapID, versionID)
	p.mu.Lock()
	defer p.mu.Unlock()
	set := p.byTitle[titleID]
	if set == nil {
		set = &playedVersionSet{}
		p.byTitle[titleID] = set
	}
	if _, ok := set.pairs[key]; ok {
		return true, nil
	}
	if !set.attempted.IsZero() && time.Since(set.attempted) < p.minReload {
		return false, set.err
	}
	set.attempted = time.Now()
	pairs := map[string]struct{}{}
	set.err = p.load(ctx, titleID, func(m, v string) { pairs[playedKey(m, v)] = struct{}{} })
	if set.err != nil {
		return false, set.err
	}
	set.pairs = pairs
	_, ok := pairs[key]
	return ok, nil
}
