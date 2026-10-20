package wire

import (
	"context"
	"log/slog"
	"sync"
)

// participationCacheMax borne le nombre de participations mémorisées. Une entrée tient
// trois chaînes courtes : la borne coûte quelques centaines de Ko au plus, et un joueur
// qui parcourt ses matchs en reste très en deçà.
const participationCacheMax = 4096

// participationKey identifie une participation : le titre (une base partagée par
// titre), le joueur et le match.
type participationKey struct {
	title   string
	xuid    string
	matchID string
}

// participationCache mémorise les participations CONFIRMÉES à un match. Le fait est
// immuable (un joueur ne cesse pas d'avoir joué un match) : une entrée n'expire donc
// jamais, elle ne sort que par éviction du plus ancien quand la borne est atteinte.
//
// Seul un `true` est mémorisé. Un `false` (le match peut encore être synchronisé) et une
// erreur (budget dépassé, base indisponible) repassent par la requête à chaque appel :
// le refus reste le fait de la base, jamais d'un cache, et le 503 fermé du middleware
// n'est pas masqué.
//
// La valeur zéro est utilisable. Sûr pour un usage concurrent.
type participationCache struct {
	// max est la borne ; 0 prend participationCacheMax (les tests posent une borne
	// petite).
	max int

	mu    sync.Mutex
	seen  map[participationKey]struct{}
	order []participationKey // anneau d'insertion, du plus ancien au plus récent à partir de next
	next  int                // position du plus ancien quand l'anneau est plein
}

func (c *participationCache) limit() int {
	if c.max > 0 {
		return c.max
	}
	return participationCacheMax
}

// check rend la participation de `key` : depuis la mémoire si elle y est, sinon par
// `load`, dont un résultat `true` sans erreur est mémorisé.
func (c *participationCache) check(
	ctx context.Context, key participationKey,
	load func(context.Context) (bool, error),
) (bool, error) {
	if c.has(key) {
		return true, nil
	}
	participated, err := load(ctx)
	if err != nil || !participated {
		return participated, err
	}
	c.add(ctx, key)
	return true, nil
}

func (c *participationCache) has(key participationKey) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	_, ok := c.seen[key]
	return ok
}

func (c *participationCache) add(ctx context.Context, key participationKey) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.seen == nil {
		c.seen = make(map[participationKey]struct{})
	}
	if _, ok := c.seen[key]; ok {
		return
	}
	if len(c.order) < c.limit() {
		c.order = append(c.order, key)
		c.seen[key] = struct{}{}
		return
	}
	evicted := c.order[c.next]
	delete(c.seen, evicted)
	c.order[c.next] = key
	c.next = (c.next + 1) % len(c.order)
	c.seen[key] = struct{}{}
	slog.DebugContext(ctx, "participation: entrée la plus ancienne évincée",
		"title", evicted.title, "match_id", evicted.matchID, "capacity", len(c.order))
}
