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

// FriendsPresenceFailureBackoff est la mémoire d'ÉCHEC du lot d'amis.
//
// Sans elle, une panne Xbox transforme chaque onglet ouvert en marteau : le
// résultat n'étant pas mis en cache, chaque poll du shell (30 s) et chaque
// onglet repartent vers un service qu'on sait indisponible — et paient sa
// latence d'échec (souvent le timeout complet) au passage, sur une requête que
// l'utilisateur attend.
//
// 30 s, soit MOINS que le TTL de succès : un incident d'une seconde ne doit pas
// geler l'affichage aussi longtemps qu'une mesure réussie. L'échec n'est donc
// pas « mis en cache » comme un résultat — le compteur ne rend jamais un chiffre
// issu d'un échec, il rend zéro sans réémettre.
const FriendsPresenceFailureBackoff = 30 * time.Second

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

// FriendPresenceCounter compte les amis en jeu, avec cache TTL, singleflight et
// mémoire d'échec.
type FriendPresenceCounter struct {
	gamertags FriendGamertagsFunc
	resolve   FriendXUIDResolver
	fetch     FriendPresenceFetcher
	titles    *title.Registry

	// mu SÉRIALISE LE CALCUL ENTIER, appel Xbox compris — c'est le singleflight.
	//
	// Un verrou pris seulement autour des champs de cache laisserait N requêtes
	// simultanées trouver le cache froid ensemble et partir toutes vers Xbox : le
	// scénario nominal au démarrage du shell (plusieurs onglets, ou un onglet qui
	// remonte après une veille). En le tenant PENDANT le fetch, les suivantes
	// attendent le résultat de la première et le lisent au cache.
	//
	// Ce que cela coûte : un appelant peut attendre la durée du fetch. C'est borné
	// en amont — PresenceService.GetSnapshot appelle Count sous un contexte de
	// friendsCountBudget (3 s), justement pour que /presence ne dépende jamais de
	// la latence de Xbox.
	mu         sync.Mutex
	cacheKey   string
	cacheCount int
	cachedAt   time.Time
	// failedKey / failedAt : la mémoire d'échec, portée par la MÊME clé que le
	// cache — changer la liste d'amis relance immédiatement, sans purger.
	failedKey string
	failedAt  time.Time
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
// Xbox indisponible. Un échec ne devient JAMAIS un résultat en cache ; il pose
// seulement un backoff court (FriendsPresenceFailureBackoff) pendant lequel le
// compteur rend zéro sans réémettre.
//
// LA CLÉ DE CACHE EST LA LISTE DE GAMERTAGS, pas les xuids résolus. Avec les
// xuids, il fallait TOUJOURS faire la requête DuckDB de résolution avant même de
// pouvoir consulter le cache : le chemin chaud payait une lecture de base à
// chaque poll du shell, pour une liste qui ne bouge qu'à la main dans les
// Réglages. Avec les gamertags, la résolution est DERRIÈRE la porte du cache.
// (La lecture des Réglages, elle, reste en amont : c'est elle qui détecte que la
// liste a changé, et c'est un chargement local, pas une requête de base.)
func (c *FriendPresenceCounter) Count(ctx context.Context) int {
	friends := normalizedFriendList(c.gamertags(ctx))
	if len(friends) == 0 {
		return 0
	}
	key := strings.Join(friends, "\n")

	// Verrou tenu jusqu'au retour, appel Xbox compris : cf. godoc du champ `mu`.
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.cacheKey == key && time.Since(c.cachedAt) <= FriendsPresenceTTL {
		return c.cacheCount
	}
	if c.failedKey == key && time.Since(c.failedAt) <= FriendsPresenceFailureBackoff {
		slog.DebugContext(ctx, "presence: lot d'amis en backoff après échec — compteur à zéro",
			"friends", len(friends), "backoff", FriendsPresenceFailureBackoff)
		return 0
	}

	n, err := c.measure(ctx, friends)
	if err != nil {
		c.failedKey, c.failedAt = key, time.Now()
		return 0
	}
	c.cacheKey, c.cacheCount, c.cachedAt = key, n, time.Now()
	// Un succès efface la mémoire d'échec : le backoff ne survit pas à la reprise.
	c.failedKey, c.failedAt = "", time.Time{}
	return n
}

// measure fait le travail réel : résolution puis lot Xbox. Appelée SOUS le verrou,
// uniquement quand ni le cache ni le backoff n'ont répondu.
//
// Une liste d'amis dont AUCUN n'a de xuid connu est un résultat (zéro), pas un
// échec : elle est mise en cache comme telle, sinon la requête de résolution
// repartirait à chaque poll pour rendre le même zéro.
func (c *FriendPresenceCounter) measure(ctx context.Context, friends []string) (int, error) {
	xuids, err := c.resolveXUIDs(ctx, friends)
	if err != nil {
		slog.WarnContext(ctx, "presence: résolution des amis échouée — compteur à zéro", "err", err)
		return 0, err
	}
	if len(xuids) == 0 {
		return 0, nil
	}
	presences, err := c.fetch(ctx, xuids)
	if err != nil {
		slog.WarnContext(ctx, "presence: lot d'amis indisponible — compteur à zéro",
			"friends", len(xuids), "err", err)
		return 0, err
	}
	return c.countInGame(ctx, presences, xuids), nil
}

// normalizedFriendList rend la liste des Réglages en forme canonique : entrées
// vidées de leurs blancs, sans doublon, TRIÉE. C'est cette forme qui sert de clé
// de cache — sans le tri, réordonner la liste dans les Réglages invaliderait un
// cache encore valable ; sans le dédoublonnage, coller deux fois le même ami en
// ferait autant.
func normalizedFriendList(gamertags []string) []string {
	seen := make(map[string]struct{}, len(gamertags))
	out := make([]string, 0, len(gamertags))
	for _, gt := range gamertags {
		gt = strings.TrimSpace(gt)
		if gt == "" {
			continue
		}
		if _, dup := seen[gt]; dup {
			continue
		}
		seen[gt] = struct{}{}
		out = append(out, gt)
	}
	sort.Strings(out)
	return out
}

// resolveXUIDs traduit les gamertags des Réglages en xuids, triés et
// dédoublonnés. Un ami jamais croisé en match n'a pas de xuid connu : il est
// ignoré, avec une trace en Debug. Une ERREUR de résolution remonte (elle vaut
// backoff) ; une liste vide n'en est pas une.
func (c *FriendPresenceCounter) resolveXUIDs(ctx context.Context, gamertags []string) ([]string, error) {
	byGamertag, err := c.resolve(ctx, gamertags)
	if err != nil {
		return nil, err
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
	return out, nil
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
