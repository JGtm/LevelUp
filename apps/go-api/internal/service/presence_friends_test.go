package service

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"levelup/go-api/internal/domain/title"
)

const (
	titleIDInfinite = "2043073184"
	titleIDHalo5    = "219630713"
	titleIDOther    = "2076696971" // jeu non suivi
)

// twoTitleRegistry : les deux titres suivis par LevelUp.
func twoTitleRegistry() *title.Registry {
	reg := title.NewRegistry()
	reg.Register(&title.TitleDescriptor{
		Slug: "halo_infinite", Name: "Halo Infinite", Status: title.StatusActive, XboxTitleID: titleIDInfinite,
	})
	reg.Register(&title.TitleDescriptor{
		Slug: "halo_5", Name: "Halo 5", Status: title.StatusActive, XboxTitleID: titleIDHalo5,
	})
	return reg
}

// counterFor assemble un compteur avec des sources en dur ; fetchCalls compte
// les allers-retours Xbox (le cache TTL en dépend).
func counterFor(t *testing.T, gamertags []string, resolved map[string]string,
	presences []FriendPresence, fetchErr error, fetchCalls *int,
) *FriendPresenceCounter {
	t.Helper()
	return NewFriendPresenceCounter(
		func(context.Context) []string { return gamertags },
		func(context.Context, []string) (map[string]string, error) { return resolved, nil },
		func(context.Context, []string) ([]FriendPresence, error) {
			if fetchCalls != nil {
				*fetchCalls++
			}
			return presences, fetchErr
		},
		twoTitleRegistry(),
	)
}

// Un ami est « en jeu » sur N'IMPORTE quel titre suivi — Halo 5 compte autant
// qu'Infinite. Un jeu hors registre ne compte pas.
func TestFriendsCount_AnyTrackedTitleCounts(t *testing.T) {
	c := counterFor(t,
		[]string{"Ami1", "Ami2", "Ami3"},
		map[string]string{"Ami1": "111", "Ami2": "222", "Ami3": "333"},
		[]FriendPresence{
			{XUID: "111", TitleID: titleIDInfinite},
			{XUID: "222", TitleID: titleIDHalo5},
			{XUID: "333", TitleID: titleIDOther},
		}, nil, nil)

	if got := c.Count(context.Background()); got != 2 {
		t.Errorf("Count = %d, attendu 2 (Infinite + Halo 5 ; l'autre jeu ne compte pas)", got)
	}
}

// Présence MASQUÉE (privacy) : l'ami arrive sans titre, ou n'arrive pas du tout.
// Dans les deux cas il n'est pas compté, et ce n'est PAS une erreur.
func TestFriendsCount_MaskedPresenceIgnored(t *testing.T) {
	c := counterFor(t,
		[]string{"Visible", "Masque", "Absent"},
		map[string]string{"Visible": "111", "Masque": "222", "Absent": "333"},
		[]FriendPresence{
			{XUID: "111", TitleID: titleIDInfinite},
			{XUID: "222", TitleID: ""}, // présence masquée : aucun titre rendu
			// "333" : absent de la réponse Xbox
		}, nil, nil)

	if got := c.Count(context.Background()); got != 1 {
		t.Errorf("Count = %d, attendu 1 (masqué et absent ignorés)", got)
	}
}

// Un ami jamais croisé en match n'a pas de xuid connu : il est ignoré, sans
// empêcher le comptage des autres.
func TestFriendsCount_UnknownGamertagIgnored(t *testing.T) {
	c := counterFor(t,
		[]string{"Connu", "JamaisCroise"},
		map[string]string{"Connu": "111"},
		[]FriendPresence{{XUID: "111", TitleID: titleIDInfinite}}, nil, nil)

	if got := c.Count(context.Background()); got != 1 {
		t.Errorf("Count = %d, attendu 1", got)
	}
}

// Xbox indisponible : compteur à zéro, jamais d'erreur remontée. L'échec n'est
// pas mémorisé COMME RÉSULTAT — le compteur ne servira jamais un chiffre issu
// d'un échec — mais il pose un BACKOFF : l'appel suivant rend zéro sans
// réémettre. Sans lui, une panne Xbox fait repartir chaque poll du shell (30 s)
// et chaque onglet vers un service qu'on sait indisponible, en payant sa latence
// d'échec sur une requête que l'utilisateur attend.
func TestFriendsCount_FetchErrorReturnsZeroAndBacksOff(t *testing.T) {
	calls := 0
	c := counterFor(t, []string{"Ami"}, map[string]string{"Ami": "111"},
		nil, errors.New("xbox 503"), &calls)

	if got := c.Count(context.Background()); got != 0 {
		t.Errorf("Count = %d, attendu 0", got)
	}
	if got := c.Count(context.Background()); got != 0 {
		t.Errorf("Count (2e appel) = %d, attendu 0", got)
	}
	if calls != 1 {
		t.Errorf("appels Xbox = %d, attendu 1 (le 2e appel est en backoff)", calls)
	}
	// Rien n'a été mis en cache comme résultat : le backoff écoulé, on retente.
	if c.cacheKey != "" {
		t.Errorf("cacheKey = %q, attendu vide (un échec n'est pas un résultat)", c.cacheKey)
	}
}

// Le backoff est COURT et il expire : passé son délai, le compteur retente.
func TestFriendsCount_RetriesAfterBackoffExpires(t *testing.T) {
	calls := 0
	c := counterFor(t, []string{"Ami"}, map[string]string{"Ami": "111"},
		nil, errors.New("xbox 503"), &calls)

	c.Count(context.Background())
	// Vieillissement à la main : le backoff est daté, pas minuté.
	c.failedAt = time.Now().Add(-FriendsPresenceFailureBackoff - time.Second)

	c.Count(context.Background())
	if calls != 2 {
		t.Errorf("appels Xbox = %d, attendu 2 (backoff expiré → nouvelle tentative)", calls)
	}
}

// Un succès efface la mémoire d'échec : le backoff ne survit pas à la reprise.
func TestFriendsCount_SuccessClearsBackoff(t *testing.T) {
	calls := 0
	var boom error = errors.New("xbox 503")
	c := NewFriendPresenceCounter(
		func(context.Context) []string { return []string{"Ami"} },
		func(context.Context, []string) (map[string]string, error) {
			return map[string]string{"Ami": "111"}, nil
		},
		func(context.Context, []string) ([]FriendPresence, error) {
			calls++
			if boom != nil {
				return nil, boom
			}
			return []FriendPresence{{XUID: "111", TitleID: titleIDInfinite}}, nil
		},
		twoTitleRegistry(),
	)

	c.Count(context.Background())
	boom = nil
	c.failedAt = time.Now().Add(-FriendsPresenceFailureBackoff - time.Second)
	if got := c.Count(context.Background()); got != 1 {
		t.Fatalf("Count après reprise = %d, attendu 1", got)
	}
	if c.failedKey != "" {
		t.Errorf("failedKey = %q, attendu vide après un succès", c.failedKey)
	}
}

// Changer la liste d'amis relance immédiatement, backoff ou pas : la mémoire
// d'échec porte la MÊME clé que le cache.
func TestFriendsCount_ListChangeBypassesBackoff(t *testing.T) {
	calls := 0
	gamertags := []string{"Ami1"}
	c := NewFriendPresenceCounter(
		func(context.Context) []string { return gamertags },
		func(context.Context, []string) (map[string]string, error) {
			return map[string]string{"Ami1": "111", "Ami2": "222"}, nil
		},
		func(context.Context, []string) ([]FriendPresence, error) {
			calls++
			return nil, errors.New("xbox 503")
		},
		twoTitleRegistry(),
	)

	c.Count(context.Background())
	gamertags = []string{"Ami1", "Ami2"}
	c.Count(context.Background())
	if calls != 2 {
		t.Errorf("appels Xbox = %d, attendu 2 (la liste a changé)", calls)
	}
}

// Le cache évite un appel Xbox par poll du shell (30 s) : deux appels
// rapprochés ne touchent Xbox qu'une fois.
func TestFriendsCount_CachedWithinTTL(t *testing.T) {
	calls := 0
	c := counterFor(t, []string{"Ami"}, map[string]string{"Ami": "111"},
		[]FriendPresence{{XUID: "111", TitleID: titleIDInfinite}}, nil, &calls)

	first := c.Count(context.Background())
	second := c.Count(context.Background())
	if first != 1 || second != 1 {
		t.Errorf("Count = %d puis %d, attendu 1 et 1", first, second)
	}
	if calls != 1 {
		t.Errorf("appels Xbox = %d, attendu 1 (2e servi par le cache)", calls)
	}
}

// Changer la liste d'amis des Réglages doit invalider le cache : la clé EST la
// liste des xuids interrogés.
func TestFriendsCount_ListChangeInvalidatesCache(t *testing.T) {
	calls := 0
	gamertags := []string{"Ami1"}
	resolved := map[string]string{"Ami1": "111", "Ami2": "222"}
	c := NewFriendPresenceCounter(
		func(context.Context) []string { return gamertags },
		func(context.Context, []string) (map[string]string, error) { return resolved, nil },
		func(_ context.Context, xuids []string) ([]FriendPresence, error) {
			calls++
			out := make([]FriendPresence, 0, len(xuids))
			for _, x := range xuids {
				out = append(out, FriendPresence{XUID: x, TitleID: titleIDInfinite})
			}
			return out, nil
		},
		twoTitleRegistry(),
	)

	if got := c.Count(context.Background()); got != 1 {
		t.Fatalf("Count initial = %d, attendu 1", got)
	}
	gamertags = []string{"Ami1", "Ami2"}
	if got := c.Count(context.Background()); got != 2 {
		t.Errorf("Count après ajout = %d, attendu 2", got)
	}
	if calls != 2 {
		t.Errorf("appels Xbox = %d, attendu 2 (la liste a changé)", calls)
	}
}

// Deux gamertags pointant le même compte, ou un doublon de la réponse Xbox, ne
// comptent qu'une fois : un ami reste un ami.
func TestFriendsCount_DeduplicatesByXUID(t *testing.T) {
	c := counterFor(t,
		[]string{"Ami", "AmiRenomme"},
		map[string]string{"Ami": "111", "AmiRenomme": "111"},
		[]FriendPresence{
			{XUID: "111", TitleID: titleIDInfinite},
			{XUID: "111", TitleID: titleIDHalo5},
		}, nil, nil)

	if got := c.Count(context.Background()); got != 1 {
		t.Errorf("Count = %d, attendu 1", got)
	}
}

// Une présence rendue par Xbox pour un xuid NON demandé n'entre pas au compte.
func TestFriendsCount_UnrequestedXUIDIgnored(t *testing.T) {
	c := counterFor(t,
		[]string{"Ami"},
		map[string]string{"Ami": "111"},
		[]FriendPresence{
			{XUID: "111", TitleID: titleIDInfinite},
			{XUID: "999", TitleID: titleIDInfinite},
		}, nil, nil)

	if got := c.Count(context.Background()); got != 1 {
		t.Errorf("Count = %d, attendu 1", got)
	}
}

// Réglages sans aucun ami : zéro, sans toucher ni la base ni Xbox.
func TestFriendsCount_NoFriendsConfigured(t *testing.T) {
	calls := 0
	c := counterFor(t, nil, nil, nil, nil, &calls)
	if got := c.Count(context.Background()); got != 0 {
		t.Errorf("Count = %d, attendu 0", got)
	}
	if calls != 0 {
		t.Errorf("appels Xbox = %d, attendu 0", calls)
	}
}

// ─── LE CACHE : SA CLÉ, SON EXPIRATION, ET LE SINGLEFLIGHT ──────────────────────

// F6 — dans le TTL, la RÉSOLUTION ne repart pas non plus. La clé de cache est la
// liste de GAMERTAGS ; avec l'ancienne clé (les xuids résolus), il fallait faire
// la requête DuckDB de résolution avant même de pouvoir consulter le cache, donc
// à chaque poll du shell, pour une liste qui ne bouge qu'à la main.
func TestFriendsCount_ResolutionIsBehindTheCache(t *testing.T) {
	resolves, fetches := 0, 0
	c := NewFriendPresenceCounter(
		func(context.Context) []string { return []string{"Ami"} },
		func(context.Context, []string) (map[string]string, error) {
			resolves++
			return map[string]string{"Ami": "111"}, nil
		},
		func(context.Context, []string) ([]FriendPresence, error) {
			fetches++
			return []FriendPresence{{XUID: "111", TitleID: titleIDInfinite}}, nil
		},
		twoTitleRegistry(),
	)

	for i := 0; i < 3; i++ {
		if got := c.Count(context.Background()); got != 1 {
			t.Fatalf("Count #%d = %d, attendu 1", i, got)
		}
	}
	if resolves != 1 {
		t.Errorf("résolutions = %d, attendu 1 (les suivantes sont derrière le cache)", resolves)
	}
	if fetches != 1 {
		t.Errorf("appels Xbox = %d, attendu 1", fetches)
	}
}

// L'ORDRE de la liste des Réglages ne doit pas invalider le cache : la clé est la
// liste TRIÉE et dédoublonnée.
func TestFriendsCount_ListReorderKeepsCache(t *testing.T) {
	calls := 0
	gamertags := []string{"Ami1", "Ami2"}
	c := NewFriendPresenceCounter(
		func(context.Context) []string { return gamertags },
		func(context.Context, []string) (map[string]string, error) {
			return map[string]string{"Ami1": "111", "Ami2": "222"}, nil
		},
		func(context.Context, []string) ([]FriendPresence, error) {
			calls++
			return []FriendPresence{{XUID: "111", TitleID: titleIDInfinite}}, nil
		},
		twoTitleRegistry(),
	)

	c.Count(context.Background())
	gamertags = []string{"Ami2", " Ami1 ", "Ami2"} // réordonnée, espacée, dupliquée
	c.Count(context.Background())
	if calls != 1 {
		t.Errorf("appels Xbox = %d, attendu 1 (même liste, autre ordre)", calls)
	}
}

// F15b — le TTL EXPIRE. `cachedAt` est vieilli à la main : le cache est daté, pas
// minuté, et rien d'autre ne prouve qu'il finit par lâcher.
func TestFriendsCount_CacheExpiresAfterTTL(t *testing.T) {
	calls := 0
	c := counterFor(t, []string{"Ami"}, map[string]string{"Ami": "111"},
		[]FriendPresence{{XUID: "111", TitleID: titleIDInfinite}}, nil, &calls)

	c.Count(context.Background())
	c.cachedAt = time.Now().Add(-FriendsPresenceTTL - time.Second)

	if got := c.Count(context.Background()); got != 1 {
		t.Errorf("Count après expiration = %d, attendu 1", got)
	}
	if calls != 2 {
		t.Errorf("appels Xbox = %d, attendu 2 (TTL écoulé → nouvel appel)", calls)
	}
}

// F4 — SINGLEFLIGHT : N requêtes simultanées à cache froid ne font PARTIR QU'UN
// SEUL lot. C'est le scénario nominal au réveil du shell (plusieurs onglets), et
// sans lui chacune émettait le sien.
func TestFriendsCount_ConcurrentCallsIssueASingleFetch(t *testing.T) {
	const concurrents = 8
	var mu sync.Mutex
	calls := 0
	entre := make(chan struct{})

	c := NewFriendPresenceCounter(
		func(context.Context) []string { return []string{"Ami"} },
		func(context.Context, []string) (map[string]string, error) {
			return map[string]string{"Ami": "111"}, nil
		},
		func(context.Context, []string) ([]FriendPresence, error) {
			mu.Lock()
			calls++
			premier := calls == 1
			mu.Unlock()
			if premier {
				// Tient la place le temps que les autres arrivent : sans
				// singleflight, elles émettraient toutes ici.
				<-entre
			}
			return []FriendPresence{{XUID: "111", TitleID: titleIDInfinite}}, nil
		},
		twoTitleRegistry(),
	)

	var wg sync.WaitGroup
	resultats := make([]int, concurrents)
	for i := 0; i < concurrents; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			resultats[idx] = c.Count(context.Background())
		}(i)
	}
	time.Sleep(20 * time.Millisecond) // laisse les concurrentes s'empiler sur le verrou
	close(entre)
	wg.Wait()

	if calls != 1 {
		t.Errorf("appels Xbox = %d, attendu 1 pour %d requêtes simultanées", calls, concurrents)
	}
	for i, n := range resultats {
		if n != 1 {
			t.Errorf("requête %d : Count = %d, attendu 1 (résultat du premier)", i, n)
		}
	}
}

// Dépendance manquante (démo, watcher off, shared indisponible) : pas de
// compteur du tout — le service rendra 0 sans jamais tenter d'appel.
func TestNewFriendPresenceCounter_NilWhenDependencyMissing(t *testing.T) {
	cases := map[string]*FriendPresenceCounter{
		"sans gamertags": NewFriendPresenceCounter(nil,
			func(context.Context, []string) (map[string]string, error) { return nil, nil },
			func(context.Context, []string) ([]FriendPresence, error) { return nil, nil },
			twoTitleRegistry()),
		"sans résolveur": NewFriendPresenceCounter(func(context.Context) []string { return nil },
			nil,
			func(context.Context, []string) ([]FriendPresence, error) { return nil, nil },
			twoTitleRegistry()),
		"sans fetch": NewFriendPresenceCounter(func(context.Context) []string { return nil },
			func(context.Context, []string) (map[string]string, error) { return nil, nil },
			nil, twoTitleRegistry()),
		"sans registre": NewFriendPresenceCounter(func(context.Context) []string { return nil },
			func(context.Context, []string) (map[string]string, error) { return nil, nil },
			func(context.Context, []string) ([]FriendPresence, error) { return nil, nil }, nil),
	}
	for name, c := range cases {
		if c != nil {
			t.Errorf("%s : compteur non nil", name)
		}
	}
}
