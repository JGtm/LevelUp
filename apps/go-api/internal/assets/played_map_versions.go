package assets

import (
	"context"
	"errors"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

// PlayedMapVersionLoader parcourt les couples (carte, version) des matchs enregistrés d'un
// titre (base partagée du titre) et appelle visit pour chacun.
type PlayedMapVersionLoader func(ctx context.Context, titleID string, visit func(mapID, versionID string)) error

// playedVersionsMinReload est l'intervalle minimal entre deux lectures de la base partagée
// pour un même titre.
const playedVersionsMinReload = time.Minute

// playedVersionsLoadTimeout borne une lecture des versions jouées d'un titre.
const playedVersionsLoadTimeout = 30 * time.Second

// PlayedMapVersions dit si une version de carte figure dans des matchs réellement joués d'un
// titre. Les couples (carte, version) d'un titre sont tenus en mémoire ; une absence ne
// relit la base qu'au plus une fois par playedVersionsMinReload, si bien qu'une version
// inventée coûte une recherche en mémoire et jamais une lecture de la base par requête.
// Un match synchronisé depuis la dernière lecture est vu à la première absence qui suit
// l'intervalle.
//
// La lecture d'un titre se fait hors verrou, une à la fois par titre (singleflight) : les
// demandes simultanées la partagent, celles d'un autre titre ne l'attendent pas. Elle tourne
// sous un contexte détaché du demandeur (ses valeurs, pas son annulation), borné par
// playedVersionsLoadTimeout : un client parti n'interrompt ni ne fait échouer la lecture
// servie aux suivants. Chaque demandeur attend au plus jusqu'à l'annulation de son contexte.
type PlayedMapVersions struct {
	load        PlayedMapVersionLoader
	minReload   time.Duration
	loadTimeout time.Duration
	flights     singleflight.Group

	mu      sync.Mutex
	byTitle map[string]*playedVersionSet
}

// playedVersionSet est l'état d'un titre : les couples lus, la date de la dernière lecture
// terminée et son erreur.
type playedVersionSet struct {
	pairs     map[string]struct{}
	attempted time.Time
	err       error
}

// NewPlayedMapVersions construit la source sur load.
func NewPlayedMapVersions(load PlayedMapVersionLoader) *PlayedMapVersions {
	return &PlayedMapVersions{
		load:        load,
		minReload:   playedVersionsMinReload,
		loadTimeout: playedVersionsLoadTimeout,
		byTitle:     map[string]*playedVersionSet{},
	}
}

func playedKey(mapID, versionID string) string { return mapID + "\x00" + versionID }

// Played dit si (mapID, versionID) figure dans un match enregistré du titre. Une erreur de
// lecture est rendue telle quelle (la dernière, jusqu'à la prochaine tentative ; une erreur
// d'annulation n'est jamais retenue) ; les couples déjà lus restent servis. L'annulation de
// ctx pendant l'attente d'une lecture rend ctx.Err().
func (p *PlayedMapVersions) Played(ctx context.Context, titleID, mapID, versionID string) (bool, error) {
	key := playedKey(mapID, versionID)
	if found, fresh, err := p.lookup(titleID, key); found || fresh {
		return found, err
	}
	flight := p.flights.DoChan(titleID, func() (any, error) {
		return nil, p.reload(context.WithoutCancel(ctx), titleID)
	})
	select {
	case res := <-flight:
		if res.Err != nil {
			return false, res.Err
		}
		found, _, _ := p.lookup(titleID, key)
		return found, nil
	case <-ctx.Done():
		return false, ctx.Err()
	}
}

// lookup cherche key parmi les couples lus du titre. fresh dit que la dernière lecture date de
// moins de minReload (aucune relecture due) ; err est alors son erreur.
func (p *PlayedMapVersions) lookup(titleID, key string) (found, fresh bool, err error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	set := p.byTitle[titleID]
	if set == nil {
		return false, false, nil
	}
	if _, ok := set.pairs[key]; ok {
		return true, true, nil
	}
	if !set.attempted.IsZero() && time.Since(set.attempted) < p.minReload {
		return false, true, set.err
	}
	return false, false, nil
}

// reload relit les couples du titre sous sa propre échéance, sauf si une lecture vient de se
// terminer (demande arrivée entre le constat d'absence et la fin d'une lecture précédente).
// Le résultat est retenu jusqu'à la prochaine relecture, sauf une erreur d'annulation.
func (p *PlayedMapVersions) reload(ctx context.Context, titleID string) error {
	p.mu.Lock()
	if set := p.byTitle[titleID]; set != nil && !set.attempted.IsZero() && time.Since(set.attempted) < p.minReload {
		err := set.err
		p.mu.Unlock()
		return err
	}
	p.mu.Unlock()

	loadCtx, cancel := context.WithTimeout(ctx, p.loadTimeout)
	defer cancel()
	pairs := map[string]struct{}{}
	err := p.load(loadCtx, titleID, func(m, v string) { pairs[playedKey(m, v)] = struct{}{} })
	if errors.Is(err, context.Canceled) {
		return err
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	set := p.byTitle[titleID]
	if set == nil {
		set = &playedVersionSet{}
		p.byTitle[titleID] = set
	}
	set.attempted = time.Now()
	set.err = err
	if err == nil {
		set.pairs = pairs
	}
	return err
}
