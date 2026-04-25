# Plan : TTL dynamique BP + Challenges selon la présence active

**Date** : 2026-04-24  
**Branche cible** : branche active/courante
**Statut** : ✅ Implémenté et testé (2026-04-24)

---

## Contexte et motivation

### Problème observé

La page Season Pass (battlepass + défis) ne se rafraîchit pas après une session de jeu si
le joueur visite la page moins d'1h après la dernière synchro.

### Comportement actuel

| Donnée | TTL cache | Source live | Comportement |
|---|---|---|---|
| Battlepass | `1h` (constante) | `GetBattlePassWithRaw` | Snapshot affiché tel quel jusqu'à expiration |
| Challenges | `1h` (même constante) | `GetChallengesWithRaw` | Idem |

Le ticker `live_refresh` tourne toutes les **5 min** pendant la session, mais si le snapshot
n'a pas changé (même `state_hash`), il est ignoré — et une fois la session terminée, le TTL
1h bloque les appels live côté page.

### Comportement cible

| Contexte | TTL BP | TTL Challenges |
|---|---|---|
| Joueur hors session | 1 heure | 1 heure |
| Joueur en session active | 5 minutes | 5 minutes |

---

## Architecture de la solution

### Principe général

Remplacer la constante `battlePassCacheTTL` (partagée BP + Challenges) par un TTL
**atomique dynamique** sur `HomeService`, piloté depuis le `PlayerLiveRefresher` via
une interface minimaliste définie dans `port/`.

```
PlayerLiveRefresher
    ↓  SetSessionActive(bool)
port.SessionNotifier  ←── interface (évite le cycle watcher → service)
    ↑  implémenté par
HomeService.sessionTTL  (atomic.Int64, nanosecondes)
    ↓  consommé par
GetBattlePass()  +  GetChallenges()
```

---

## Fichiers impactés

| Fichier | Modification |
|---|---|
| `internal/port/services.go` | Ajouter interface `SessionNotifier` |
| `internal/service/home_service.go` | Champ `sessionTTL atomic.Int64` + méthode `SetSessionActive` |
| `internal/watcher/live_refresh.go` | Champ `notifier port.SessionNotifier` + appels dans `OnPresenceActive/Inactive` |
| `internal/watcher/daemon.go` | Passer le `HomeService` comme notifier dans `LiveRefreshFactory` |
| `internal/api/registry.go` | Exposer le `*service.HomeService` concret pour le brancher dans la factory |
| Tests | Couvrir les 4 cas (BP/challenges × session active/inactive) |

---

## Détail des modifications

### Étape 1 — `internal/port/services.go`

Ajouter **après** l'interface `HomeService` :

```go
// SessionNotifier est notifié des changements de présence active d'un joueur.
// Implémenté par *service.HomeService pour ajuster le TTL cache BP/Challenges.
type SessionNotifier interface {
    SetSessionActive(active bool)
}
```

Pas de dépendance circulaire : `port/` n'importe ni `service/` ni `watcher/`.

---

### Étape 2 — `internal/service/home_service.go`

**a) Imports à ajouter**

```go
"sync/atomic"
```

**b) Constantes**

```go
const (
    battlePassCacheTTLDefault = 1 * time.Hour
    battlePassCacheTTLActive  = 5 * time.Minute
)
```

Supprimer l'ancienne `const battlePassCacheTTL = 1 * time.Hour`.

**c) Champ sur la struct**

```go
type HomeService struct {
    // ... champs existants ...
    sessionTTL atomic.Int64  // TTL en nanosecondes; 0 = valeur par défaut (1h)
}
```

**d) Méthode `SetSessionActive`** (implémente `port.SessionNotifier`)

```go
// SetSessionActive bascule le TTL cache BP/Challenges selon la présence du joueur.
// true → 5 min (symétrie avec liveRefreshInterval), false → 1 h (défaut).
func (s *HomeService) SetSessionActive(active bool) {
    if active {
        s.sessionTTL.Store(battlePassCacheTTLActive.Nanoseconds())
    } else {
        s.sessionTTL.Store(0) // 0 = défaut (1h)
    }
}
```

**e) Méthode helper privée**

```go
func (s *HomeService) currentTTL() time.Duration {
    if ns := s.sessionTTL.Load(); ns > 0 {
        return time.Duration(ns)
    }
    return battlePassCacheTTLDefault
}
```

**f) Utilisation dans `GetBattlePass` et `GetChallenges`**

Remplacer les deux occurrences de `battlePassCacheTTL` par `s.currentTTL()` :

```go
// GetBattlePass
if cached, hit, err := s.cacheRepo.LoadCachedBattlePass(ctx, s.currentTTL()); ...

// GetChallenges
if cached, hit, err := s.cacheRepo.LoadCachedChallenges(ctx, s.currentTTL()); ...
```

---

### Étape 3 — `internal/watcher/live_refresh.go`

**a) Nouveau champ sur `PlayerLiveRefresher`**

```go
type PlayerLiveRefresher struct {
    // ... champs existants ...
    notifier port.SessionNotifier // nil si non configuré
}
```

**b) Méthode de configuration (chaînage)**

```go
// WithSessionNotifier configure le notifier de présence.
func (r *PlayerLiveRefresher) WithSessionNotifier(n port.SessionNotifier) *PlayerLiveRefresher {
    r.notifier = n
    return r
}
```

**c) Appels dans `OnPresenceActive` et `OnPresenceInactive`**

```go
func (r *PlayerLiveRefresher) OnPresenceActive(ctx context.Context) {
    if r.notifier != nil {
        r.notifier.SetSessionActive(true)
    }
    // ... logique ticker existante ...
}

func (r *PlayerLiveRefresher) OnPresenceInactive(ctx context.Context) {
    // ... arrêt ticker existant ...
    if r.notifier != nil {
        r.notifier.SetSessionActive(false)
    }
}
```

> **Ordre important** : `SetSessionActive(true)` **avant** le démarrage du ticker ;
> `SetSessionActive(false)` **après** l'arrêt du ticker.

---

### Étape 4 — `internal/api/registry.go`

Le `HomeService` est créé dans `HomeCtxWithAuth` et `SeasonPassCtxWithAuth`.
Pour permettre au daemon de récupérer le bon notifier, deux options :

#### Option A — Registry de notifiers (recommandée)

Ajouter un champ `notifiers sync.Map` sur `ServiceRegistry` :

```go
type ServiceRegistry struct {
    // ... champs existants ...
    notifiers sync.Map // xuid → port.SessionNotifier
}
```

Dans `HomeCtxWithAuth` et `SeasonPassCtxWithAuth`, après la création du `homeSvc` :

```go
r.notifiers.Store(pdb.XUID, port.SessionNotifier(svc))
```

Exposer une méthode :

```go
func (r *ServiceRegistry) GetSessionNotifier(xuid string) port.SessionNotifier {
    if v, ok := r.notifiers.Load(xuid); ok {
        return v.(port.SessionNotifier)
    }
    return nil
}
```

#### Option B — Notifier créé avant les services (plus simple, moins flexible)

Créer un `HomeService` persistant par joueur dans la registry (nécessite de changer
l'architecture de `ServiceRegistry` qui crée actuellement les services à la demande).
**Option A préférable** car elle ne change pas le cycle de vie des services.

---

### Étape 5 — `cmd/server/main.go`

Modifier la `LiveRefreshFactory` pour brancher le notifier :

```go
LiveRefreshFactory: func(gamertag, xuid string) watcher.LiveRefreshTrigger {
    wMetaPath := watcherPR.MetadataDBPath(watcherSlug)
    wPlayerPath := watcherPR.PlayerDBPath(watcherSlug, gamertag)
    sink := duckdb.NewPersistSink(wMetaPath, wPlayerPath, xuid)
    refresher := watcher.NewPlayerLiveRefresher(gamertag, xuid, sink, nil)
    if n := registry.GetSessionNotifier(xuid); n != nil {
        refresher = refresher.WithSessionNotifier(n)
    }
    return refresher
},
```

> **Note** : La factory est appelée au démarrage du daemon. Si le notifier n'est pas
> encore enregistré (aucun appel HTTP pour ce joueur), il sera `nil` → comportement
> identique à l'état actuel (TTL 1h par défaut).  
> Alternative : enregistrer le notifier au démarrage via un appel synthétique, ou
> accepter que le TTL dynamique ne soit effectif qu'après le premier accès HTTP.

---

## Logging

### Dans `SetSessionActive` (`home_service.go`)

```go
func (s *HomeService) SetSessionActive(active bool) {
    if active {
        s.sessionTTL.Store(battlePassCacheTTLActive.Nanoseconds())
        slog.Info("home_service: session active — TTL cache réduit",
            "ttl", battlePassCacheTTLActive,
            "player", s.playerSlug,
        )
    } else {
        s.sessionTTL.Store(0)
        slog.Info("home_service: session inactive — TTL cache restauré",
            "ttl", battlePassCacheTTLDefault,
            "player", s.playerSlug,
        )
    }
}
```

### Dans `GetBattlePass` et `GetChallenges`

Enrichir les logs de cache hit/miss existants avec le TTL effectif :

```go
// Avant l'appel LoadCachedBattlePass :
ttl := s.currentTTL()
slog.DebugContext(ctx, "home: GetBattlePass cache check", "ttl", ttl)

// Sur cache hit (ligne existante à enrichir) :
slog.DebugContext(ctx, "home: BattlePass servi depuis cache DB", "ttl_used", ttl)

// Sur cache miss (ligne existante) :
slog.DebugContext(ctx, "home: BattlePass cache miss → appel live", "ttl_used", ttl)
```

Même pattern pour `GetChallenges`.

### Dans `OnPresenceActive` / `OnPresenceInactive` (`live_refresh.go`)

```go
// OnPresenceActive — après SetSessionActive(true) :
slog.InfoContext(ctx, "live_refresh: notifier session active",
    "gamertag", r.gamertag,
    "notifier_configured", r.notifier != nil,
)

// OnPresenceInactive — après SetSessionActive(false) :
slog.InfoContext(ctx, "live_refresh: notifier session inactive",
    "gamertag", r.gamertag,
)
```

Si `r.notifier == nil` au moment de l'appel → log `Warn` une seule fois (flag `notifierWarnOnce sync.Once`) :

```go
slog.WarnContext(ctx, "live_refresh: aucun SessionNotifier configuré — TTL dynamique inactif",
    "gamertag", r.gamertag,
)
```

---

## Tests à écrire

### `internal/service/home_service_test.go`

#### Tests TTL — comportement de base

| Test | Scénario | Type |
|---|---|---|
| `TestHomeService_DefaultTTL_Is1Hour` | Sans appel `SetSessionActive` → `currentTTL()` = 1h | Unitaire |
| `TestHomeService_SetSessionActive_True_Reduces_TTL` | Après `SetSessionActive(true)` → `currentTTL()` = 5 min | Unitaire |
| `TestHomeService_SetSessionActive_False_Restores_TTL` | `true` puis `false` → retour 1h | Unitaire |
| `TestHomeService_SetSessionActive_Idempotent_True` | Deux `SetSessionActive(true)` consécutifs → TTL reste 5 min | Unitaire (non-régression) |
| `TestHomeService_SetSessionActive_Idempotent_False` | Deux `SetSessionActive(false)` sans `true` préalable → TTL 1h, pas de panique | Unitaire (non-régression) |

#### Tests TTL — impact sur GetBattlePass

| Test | Scénario | Type |
|---|---|---|
| `TestGetBattlePass_SessionActive_CacheHitAt4Min` | Session active + snapshot à 4 min → cache hit (TTL 5 min) | Intégration |
| `TestGetBattlePass_SessionActive_CacheMissAt6Min` | Session active + snapshot à 6 min → cache miss + appel live | Intégration |
| `TestGetBattlePass_SessionInactive_CacheHitAt50Min` | Session inactive + snapshot à 50 min → cache hit (TTL 1h) | Intégration |
| `TestGetBattlePass_SessionInactive_CacheMissAt70Min` | Session inactive + snapshot à 70 min → cache miss | Intégration |

#### Tests TTL — impact sur GetChallenges (symétrie)

Mêmes 4 tests que ci-dessus, préfixés `TestGetChallenges_*`.

#### Non-régression comportement existant

| Test | Ce qu'il vérifie |
|---|---|
| `TestGetBattlePass_NoSink_StillWorks` | Sans `PersistSink`, `GetBattlePass` ne panique pas après `SetSessionActive(true)` |
| `TestGetChallenges_NoSink_StillWorks` | Idem pour challenges |
| `TestGetBattlePass_NoCacheRepo_LiveFallback` | Sans `cacheRepo`, le TTL dynamique ne casse pas le fallback live |

#### Test de concurrence

| Test | Scénario |
|---|---|
| `TestHomeService_ConcurrentSetSessionActive` | N goroutines alternent `true/false` pendant 50 ms + M goroutines lisent `currentTTL()` → pas de data race (détecté par `-race`) |

---

### `internal/watcher/live_refresh_test.go`

#### Non-régression notifier

| Test | Scénario | Type |
|---|---|---|
| `TestPlayerLiveRefresher_NilNotifier_PresenceActive_NoOp` | Sans notifier → `OnPresenceActive` ne panique pas | Non-régression |
| `TestPlayerLiveRefresher_NilNotifier_PresenceInactive_NoOp` | Sans notifier → `OnPresenceInactive` ne panique pas | Non-régression |
| `TestPlayerLiveRefresher_NotifiesOnPresenceActive` | Notifier configuré → `SetSessionActive(true)` appelé | Comportement |
| `TestPlayerLiveRefresher_NotifiesOnPresenceInactive` | Notifier configuré → `SetSessionActive(false)` appelé | Comportement |
| `TestPlayerLiveRefresher_NotifyOrder_ActiveBeforeTicker` | `SetSessionActive(true)` appelé **avant** le démarrage du ticker | Ordre |
| `TestPlayerLiveRefresher_NotifyOrder_InactiveAfterTicker` | `SetSessionActive(false)` appelé **après** l'arrêt du ticker | Ordre |
| `TestPlayerLiveRefresher_DoubleActive_IdempotentNotify` | Deux `OnPresenceActive` consécutifs → `SetSessionActive(true)` appelé une seule fois (ticker déjà en cours) | Non-régression |

---

## Séquence temporelle après implémentation

```
T+0:00  Joueur détecté en jeu (RTA)
           → SetSessionActive(true)  → TTL = 5 min
           → Ticker démarré (5 min)

T+0:05  Ticker → GetBattlePassWithRaw + GetChallengesWithRaw
           → PersistBattlePassSync  (snapshot en DB)
           → PersistChallenges      (snapshot en DB)

T+0:06  Joueur ouvre la page Season Pass
           → TTL = 5 min → cache miss si snapshot > 5 min → appel live
           → Rang à jour affiché

T+2:00  Joueur quitte le jeu (RTA inactif)
           → Ticker arrêté
           → SetSessionActive(false) → TTL = 1h

T+2:01  Joueur ouvre la page Season Pass
           → TTL = 1h → cache hit si snapshot < 1h
           → Données de fin de session affichées
```

---

## Points de vigilance

1. **Thread-safety** : `atomic.Int64` suffit pour 1 writer (watcher goroutine) + N readers
   (handlers HTTP). Pas de mutex nécessaire sur le TTL.

2. **Cold start** : Si le daemon démarre et l'utilisateur est déjà en jeu, `SetSessionActive`
   ne sera appelé qu'après la première connexion RTA. Acceptable — la condition de warm start
   se résout au premier event de présence reçu.

3. **Multi-joueurs** : Chaque `HomeService` est une instance par joueur (créée à chaque requête
   dans la registry actuelle). Le `notifier` dans le refresher doit pointer sur l'instance
   **partagée** entre les requêtes → nécessite Option A (registry de notifiers).

4. **`LiveRefreshFactory` appelée au démarrage** : La factory est exécutée dans `initPlayers`
   avant tout appel HTTP. Si `registry.GetSessionNotifier(xuid)` retourne `nil` à ce moment,
   le notifier peut être injecté **lazily** dans `OnPresenceActive` via un lookup tardif.
   Alternative plus simple : pré-créer un `HomeService` partagé par joueur dans la registry
   et le réutiliser dans les handlers → changement d'architecture plus lourd.

5. **Symétrie challenges** : Le seul changement pour les challenges est le remplacement de
   `battlePassCacheTTL` par `s.currentTTL()` dans `GetChallenges` — même TTL dynamique que
   le battlepass.
