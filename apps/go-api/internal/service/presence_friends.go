// Package service — presence_friends.go : « combien d'amis sont en jeu ».
//
// « Amis » = la liste `friend_gamertags` des Réglages (décision produit du
// 2026-08-24), PAS la liste d'amis Xbox : c'est le cercle que l'utilisateur a
// lui-même déclaré, et le seul dont on connaisse déjà les gamertags.
//
// Chemin : gamertags des Réglages → xuids (vue partagée v_gamertag_lookup) →
// un appel batch Xbox → compte de ceux vus sur un titre suivi. Le tout à la
// demande, derrière un cache TTL court : le shell interroge l'endpoint toutes
// les 30 s et il n'y a pas de poller dédié aux amis.
package service

import (
	"context"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"

	"levelup/go-api/internal/domain/title"
)

// FriendsPresenceTTL est la durée de validité du compteur d'amis en jeu. 45 s :
// plus long que l'intervalle de rafraîchissement du shell (30 s) pour qu'un
// onglet ouvert ne déclenche pas un appel Xbox à chaque poll, assez court pour
// qu'un ami qui lance une partie apparaisse en moins d'une minute.
const FriendsPresenceTTL = 45 * time.Second

// FriendPresence est la présence brute d'un ami. Type neutre : le service ne
// dépend pas du client Xbox (l'adaptation se fait au composition root).
// TitleID est l'identifiant Xbox du titre actif, vide si aucun ou si la
// présence de cet ami est masquée.
type FriendPresence struct {
	XUID    string
	TitleID string
}

// FriendGamertagsFunc rend la liste `friend_gamertags` des Réglages.
type FriendGamertagsFunc func(ctx context.Context) []string

// FriendXUIDResolver résout des gamertags en xuids. Clé de la map = le gamertag
// demandé ; un gamertag inconnu est simplement absent du résultat.
type FriendXUIDResolver func(ctx context.Context, gamertags []string) (map[string]string, error)

// FriendPresenceFetcher rend la présence Xbox de plusieurs xuids en un appel.
type FriendPresenceFetcher func(ctx context.Context, xuids []string) ([]FriendPresence, error)

// FriendPresenceCounter compte les amis en jeu, avec cache TTL.
type FriendPresenceCounter struct {
	gamertags FriendGamertagsFunc
	resolve   FriendXUIDResolver
	fetch     FriendPresenceFetcher
	titles    *title.Registry

	mu         sync.Mutex
	cacheKey   string
	cacheCount int
	cachedAt   time.Time
}

// NewFriendPresenceCounter crée le compteur d'amis en jeu. Retourne nil si une
// dépendance manque — le service rend alors friends_in_game = 0 sans jamais
// tenter d'appel (démo, watcher désactivé, pas de settings store).
func NewFriendPresenceCounter(
	gamertags FriendGamertagsFunc,
	resolve FriendXUIDResolver,
	fetch FriendPresenceFetcher,
	titles *title.Registry,
) *FriendPresenceCounter {
	if gamertags == nil || resolve == nil || fetch == nil || titles == nil {
		return nil
	}
	return &FriendPresenceCounter{gamertags: gamertags, resolve: resolve, fetch: fetch, titles: titles}
}

// Count retourne le nombre d'amis actuellement en jeu sur un titre suivi.
//
// Toute défaillance rend 0 : Réglages vides, gamertags inconnus de la base,
// Xbox indisponible. Un échec n'est PAS mis en cache — le prochain appel
// retentera, au pire une fois toutes les 30 s.
func (c *FriendPresenceCounter) Count(ctx context.Context) int {
	gts := c.gamertags(ctx)
	if len(gts) == 0 {
		return 0
	}

	xuids := c.resolveXUIDs(ctx, gts)
	if len(xuids) == 0 {
		return 0
	}

	key := strings.Join(xuids, ",")
	if n, ok := c.cached(key); ok {
		return n
	}

	presences, err := c.fetch(ctx, xuids)
	if err != nil {
		slog.WarnContext(ctx, "presence: lot d'amis indisponible — compteur à zéro",
			"friends", len(xuids), "err", err)
		return 0
	}

	n := c.countInGame(ctx, presences, xuids)
	c.store(key, n)
	return n
}

// resolveXUIDs traduit les gamertags des Réglages en xuids, triés et
// dédoublonnés (l'ordre stable sert de clé de cache). Un ami jamais croisé en
// match n'a pas de xuid connu : il est ignoré, avec une trace en Debug.
func (c *FriendPresenceCounter) resolveXUIDs(ctx context.Context, gamertags []string) []string {
	byGamertag, err := c.resolve(ctx, gamertags)
	if err != nil {
		slog.WarnContext(ctx, "presence: résolution des amis échouée — compteur à zéro", "err", err)
		return nil
	}
	seen := make(map[string]struct{}, len(byGamertag))
	out := make([]string, 0, len(byGamertag))
	for _, gt := range gamertags {
		xuid, ok := byGamertag[gt]
		if !ok || xuid == "" {
			slog.DebugContext(ctx, "presence: ami sans xuid connu — non compté", "gamertag", gt)
			continue
		}
		if _, dup := seen[xuid]; dup {
			continue
		}
		seen[xuid] = struct{}{}
		out = append(out, xuid)
	}
	sort.Strings(out)
	return out
}

// countInGame compte les présences portant un titre du registre. Un ami dont la
// présence est masquée (privacy Xbox) arrive sans titre, ou n'arrive pas du
// tout : il n'est pas compté et ce n'est PAS une erreur — l'utilisateur ne doit
// jamais voir d'échec parce qu'un ami protège sa vie privée.
func (c *FriendPresenceCounter) countInGame(ctx context.Context, presences []FriendPresence, asked []string) int {
	expected := make(map[string]struct{}, len(asked))
	for _, x := range asked {
		expected[x] = struct{}{}
	}
	counted := make(map[string]struct{}, len(presences))
	for _, p := range presences {
		if p.TitleID == "" {
			slog.DebugContext(ctx, "presence: ami sans titre actif (hors jeu ou présence masquée)", "xuid", p.XUID)
			continue
		}
		// Xbox peut renvoyer un enregistrement pour un xuid non demandé (ou
		// vide) : on ne compte que ce qu'on a demandé, une seule fois.
		if _, ok := expected[p.XUID]; !ok {
			continue
		}
		if _, dup := counted[p.XUID]; dup {
			continue
		}
		// « En jeu » = sur N'IMPORTE quel titre suivi par LevelUp, pas seulement
		// le titre courant de l'utilisateur : un ami sur Halo 5 est en jeu.
		if c.titles.MatchPresence(p.TitleID) == nil {
			continue
		}
		counted[p.XUID] = struct{}{}
	}
	return len(counted)
}

// cached retourne le compteur mémorisé si la clé (liste d'amis) est la même et
// que le TTL n'est pas écoulé. Changer la liste des Réglages invalide de fait.
func (c *FriendPresenceCounter) cached(key string) (int, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cacheKey != key || time.Since(c.cachedAt) > FriendsPresenceTTL {
		return 0, false
	}
	return c.cacheCount, true
}

func (c *FriendPresenceCounter) store(key string, count int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cacheKey = key
	c.cacheCount = count
	c.cachedAt = time.Now()
}
