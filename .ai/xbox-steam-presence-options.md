# Surveillance de présence joueur — Xbox & Steam

> Objectif : déclencher automatiquement un sync LevelUp dès qu'un joueur tracké termine une partie,
> sans intervention manuelle. L'app tourne sur un serveur headless en continu.

-----

## Vue d'ensemble des approches

|Approche              |Plateforme|Méthode |Latence       |Complexité |Open source|
|----------------------|----------|--------|--------------|-----------|-----------|
|OpenXBL REST          |Xbox      |Polling |~60s          |Faible     |✗          |
|`xbox-webapi-python`  |Xbox      |Polling |~60s          |Faible     |✅          |
|**Xbox RTA WebSocket**|**Xbox**  |**Push**|**Instantané**|**Élevée** |**✅**      |
|Steam Web API         |Steam     |Polling |~60s          |Faible     |✅          |

**Choix retenu : Option 3 (RTA WebSocket) — 100% Go, sans dépendance Python.**

-----

## Option retenue — Xbox RTA WebSocket (Go natif)

**Type** : Protocole WebSocket officiel Xbox Live
**Langue** : Go (`gorilla/websocket`)
**Endpoint** : `wss://rta.xboxlive.com/connect`

### Principe

S'abonner au topic de présence de chaque joueur tracké via une connexion WebSocket persistante.
Xbox pousse un event dès que l'état change — aucun polling, zéro overhead à vide.
Le watcher tourne en daemon sur le serveur, réagit aux events et déclenche les syncs.

### Topic à subscribe (un par joueur tracké)

```
https://userpresence.xboxlive.com/users/xuid(<XUID>)/richpresence
```

### Payload reçu à chaque changement

```json
{
  "xuid": "1234567890123456",
  "presenceState": "Online",
  "presenceDetails": [
    {
      "titleid": "1144039928",
      "titleName": "Halo Infinite",
      "isGame": true,
      "isPrimary": true,
      "device": "PC",
      "state": "Active"
    }
  ]
}
```

> Le champ `titleid` est la source de vérité pour identifier le jeu. `presenceState: "Offline"` ou
> absence de `presenceDetails` = joueur déconnecté.

-----

## Architecture Go — séparation des responsabilités

```
cmd/
    server/
        main.go            ← point d'entrée unique (Option B) : API + watcher comme goroutine

internal/
    auth/
        xsts.go            ← OAuth2 → XBL → XSTS (flow complet, sans lib externe)
        token_store.go     ← persistence token sur disque (refresh sans redémarrage)
        refresh_loop.go    ← goroutine de refresh automatique avant expiration

    presence/
        rta_client.go      ← connexion WebSocket RTA, subscribe, read loop
        reconnect.go       ← reconnexion automatique avec backoff exponentiel
        event_parser.go    ← parsing payload → PresenceEvent struct
        steam_poller.go    ← poll Steam API toutes les 60s (fallback joueurs Steam)

    titles/
        registry.go        ← catalogue multi-titre (Halo, CS2, Fortnite…)
        matcher.go         ← titleid / steam_app_id → TitleConfig

    watcher/
        state_machine.go   ← FSM par joueur : Idle/Watching/Syncing/Cooling
        match_poller.go    ← GET /hi/players/xuid/matches, détection nouveaux IDs
        match_queue.go     ← chan []string : file d'attente match_ids à syncer
        player_watcher.go  ← goroutine par joueur, orchestre RTA + Steam + sync trigger
        provider.go        ← implémente WatcherStateProvider (exposé à l'API via interface)

    sync/
        trigger.go         ← appel sync Go (in-process, même binaire)
        coordinator.go     ← gestion concurrence multi-joueurs (voir § dédié)

    config/
        profiles.go        ← lecture db_profiles.json → PlayerProfile (xuid + steam_id)
        titles.go          ← lecture titles.json → TitleConfig par titleid/steam_app_id

    notify/
        discord.go         ← webhook Discord
        notifier.go        ← interface Notifier (Discord, Telegram, ntfy…)
```

**Règles d'architecture :**
- `internal/auth/` ne connaît pas `internal/watcher/` ni `internal/sync/`
- `internal/titles/` ne connaît ni la DB ni le sync — catalogue statique pur
- `internal/presence/steam_poller.go` ne connaît pas la FSM — émet des callbacks
- `internal/watcher/` orchestre RTA + Steam mais ne fait pas de sync → délègue à `internal/sync/`
- `internal/sync/` ne connaît pas la présence — reçoit juste une liste de match_ids
- `internal/watcher/provider.go` est le seul point de contact entre watcher et API HTTP
- Toute dépendance circulaire = erreur de design

-----

## Auth XSTS — flow complet en Go (sans lib Python)

Le token XSTS expire ~90 min. Sur un serveur headless, le refresh doit être **entièrement automatique**.

### Flow d'obtention

```
1. POST login.live.com/oauth20_token.srf
       → access_token, refresh_token (expiry: 1h)

2. POST user.auth.xboxlive.com/authenticate
       body: { Token: access_token, TokenType: "JWT" }
       → XBL token, userhash

3. POST xsts.auth.xboxlive.com/authorize
       body: { Tokens: [xbl_token], RelyingParty: "http://xboxlive.com" }
       → XSTS token (expiry: ~90 min)

4. Connexion RTA avec header:
       Authorization: XBL3.0 x=<userhash>;<xsts_token>
```

### Persistence et refresh automatique (token_store.go)

Les tokens sont persistés sur disque (`data/auth/tokens.json`) :

```json
{
  "access_token": "...",
  "refresh_token": "...",       ← permanent, survit aux redémarrages
  "xbl_token": "...",
  "xsts_token": "...",
  "xsts_expires_at": "2026-04-20T15:30:00Z",
  "oauth_expires_at": "2026-04-20T14:00:00Z"
}
```

**Stratégie refresh (refresh_loop.go) :**

```
Toutes les 5 minutes, vérifier :
  - Si oauth_expires_at - now < 10 min  → refresher le OAuth access_token via refresh_token
  - Si xsts_expires_at - now < 15 min   → re-générer XBL + XSTS depuis l'access_token frais
  - Si la connexion RTA est ouverte      → la fermer proprement et reconnecter avec le nouveau XSTS
```

> **Première connexion** : nécessite une auth interactive (browser OAuth). Le token `refresh_token`
> résultant est permanent (valide des mois). Tous les refreshs suivants sont automatiques.
> Un script CLI `cmd/auth-setup/main.go` gère cette étape unique.

-----

## Catalogue multi-titre (internal/titles/)

La surveillance n'est pas limitée à Halo. Chaque titre déclenche un comportement configurable.

### TitleConfig (titles/registry.go)

```go
type TitleConfig struct {
    TitleID      string        // ex: "1144039928"
    Name         string        // ex: "Halo Infinite"
    SyncTarget   string        // ex: "halo", "cs2" — identifie l'orchestrateur sync
    PollInterval time.Duration // intervalle poll matchs pendant une session active
    CooldownTime time.Duration // délai après déconnexion avant d'arrêter le poll
}
```

### titles.json (config externe)

```json
[
  {
    "title_id": "1144039928",
    "name": "Halo Infinite",
    "sync_target": "halo",
    "poll_interval_s": 30,
    "cooldown_s": 300
  },
  {
    "title_id": "730",
    "name": "Counter-Strike 2",
    "sync_target": "cs2",
    "poll_interval_s": 60,
    "cooldown_s": 120
  }
]
```

Le `matcher.go` résout un `titleid` reçu dans un event RTA vers le `TitleConfig` correspondant.
Si le titleid n'est pas dans le catalogue → event ignoré silencieusement.

-----

## State machine par joueur (watcher/state_machine.go)

Chaque joueur tracké a sa propre goroutine avec une FSM indépendante.

```
         ┌─────────────────────────────────────────┐
         │                                         │
   PresenceEvent(titre connu)              PresenceEvent(offline)
         │                                         │
    ┌────▼────┐    nouveau match_id     ┌───────────┴──────┐
    │ Watching ├────────────────────────► Syncing          │
    │  (poll) │◄────────────────────────┤ (poll suspendu)  │
    └────┬────┘    sync terminé         └──────────────────┘
         │
   PresenceEvent(offline / titre inconnu)
         │
    ┌────▼────┐   cooldown_s écoulé    ┌──────────┐
    │ Cooling ├────────────────────────► Idle      │
    └─────────┘                        │ (aucune  │
                                       │ activité)│
                                       └──────────┘
```

**États :**
- `Idle` — aucune activité, aucun poll, connexion RTA active (écoute passive)
- `Watching` — joueur actif sur un titre connu, poll API matchs actif
- `Syncing` — nouveau(x) match(s) détecté(s), poll suspendu, sync en cours
- `Cooling` — joueur déconnecté, poll encore actif pendant `cooldown_s` secondes

**Transitions :**
- RTA event avec `titleid` connu → `Idle → Watching`
- `match_id` non connu détecté par le poller → `Watching → Syncing`
- Sync terminé → `Syncing → Watching`
- RTA event `Offline` ou titre inconnu → `Watching/Syncing → Cooling`
- Timer `cooldown_s` expiré → `Cooling → Idle`

-----

## File d'attente matchs (watcher/match_queue.go)

Si plusieurs matchs nouveaux sont détectés en rafale (ex : joueur revient après 5 parties non syncées) :

```go
type MatchQueue struct {
    pending chan string   // match_ids à syncer, bufferisé
    seen    map[string]struct{} // dédoublonnage en mémoire
}
```

**Comportement :**
- Le poller détecte N nouveaux IDs → les enfile tous dans `pending`
- Le sync trigger dépile **un par un** et les passe au sync (ou en batch si le sync le supporte)
- Le sync existant n'est **pas modifié** — il reçoit juste une liste de match_ids à traiter
- Pendant que `pending` est non vide, l'état reste `Syncing`
- Retour à `Watching` uniquement quand `pending` est vide ET le dernier sync est confirmé

Le `seen` map est réinitialisé à chaque session (transition `Idle → Watching`) pour éviter de blocker indéfiniment sur des IDs anciens.

-----

## Concurrence multi-joueurs (sync/coordinator.go)

### Problème

Deux joueurs peuvent être actifs en même temps **sans jouer ensemble** (sessions indépendantes).
Chacun détecte ses propres matchs et veut déclencher un sync simultanément.

### Comportement du sync Python existant

Le sync (`python scripts/sync.py --delta --gamertag X`) est **par joueur** et indépendant.
Deux syncs de joueurs différents peuvent tourner en parallèle sans conflit DB (chacun écrit dans
`data/players/{gamertag}/stats.duckdb` + les tables shared sont en append-only).

### Décision : syncs parallèles autorisés, avec limite de concurrence

```go
type SyncCoordinator struct {
    sem       chan struct{}    // sémaphore, ex: max 3 syncs simultanés
    inFlight  map[string]bool // gamertag → sync en cours (pour éviter doublon)
    mu        sync.Mutex
}
```

**Règles :**
1. Un seul sync actif **par joueur** à la fois (le `inFlight` map le garantit)
2. Maximum N syncs simultanés au total (sémaphore configurable, défaut = 3)
3. Si un sync est déjà en cours pour un joueur et qu'un nouveau match est détecté → le match_id est
   mis en file d'attente, le sync en cours le prendra ou un nouveau sync sera lancé à sa fin
4. Les syncs de joueurs **différents** s'exécutent en parallèle sans coordination particulière

### Cas : deux joueurs jouent à des jeux différents

Chacun a sa propre FSM, son propre poller, sa propre file. Le `SyncTarget` du `TitleConfig` 
détermine quel orchestrateur de sync est appelé. Aucune interaction entre eux.

-----

## Halo Infinite via Steam — détection hybride

### Le problème

Halo Infinite lancé via Steam démarre bien le client Xbox Live en arrière-plan (obligatoire pour
le matchmaking), mais la remontée de présence Xbox RTA est **inconsistante** : certains joueurs
apparaissent avec le `titleid` Xbox, d'autres non, selon la version du Xbox overlay installée.

### Stratégie : RTA en priorité, Steam en fallback par joueur

```
Pour chaque joueur dans db_profiles.json :
  - Si xuid présent      → subscribe RTA WebSocket (source primaire)
  - Si steam_id présent  → poll Steam API toutes les 60s (fallback)
  - Si les deux          → RTA en priorité ; Steam active la FSM si RTA silencieux
```

**Règle de bascule :** si Steam détecte `gameid=1336960` (Halo Infinite sur Steam) et que RTA
n'a pas émis d'event depuis plus de 2 minutes → la FSM passe en `Watching` sur signal Steam.
Dès qu'un event RTA arrive, il reprend la priorité.

### steam_id dans db_profiles.json

```json
{
  "profiles": {
    "MonGamertag": {
      "xuid": "1234567890123456",
      "steam_id": "76561198xxxxxxxxx"   ← optionnel, uniquement pour joueurs Steam
    }
  }
}
```

Si `steam_id` est absent → pas de fallback Steam, RTA uniquement.
Si `xuid` est absent mais `steam_id` présent → Steam uniquement (joueur sans compte Xbox actif).

### Nouveau fichier : `internal/presence/steam_poller.go`

```go
type SteamPoller struct {
    steamID     string
    apiKey      string          // STEAM_API_KEY dans .env.local
    interval    time.Duration   // 60s
    onActive    func(titleName string)   // callback → signal FSM
    onInactive  func()
}
```

Poll `https://api.steampowered.com/ISteamUser/GetPlayerSummaries/v2/` toutes les 60s.
Si `gameid` présent dans la réponse → appelle `onActive(gameextrainfo)`.
Si `gameid` absent → appelle `onInactive()`.

Le profil Steam doit être **public** (même contrainte que Xbox pour la visibilité).

### Nouveau champ dans TitleConfig

```go
type TitleConfig struct {
    TitleID      string
    SteamAppID   string        // ex: "1336960" pour Halo Infinite — "" si non Steam
    Name         string
    SyncTarget   string
    PollInterval time.Duration
    CooldownTime time.Duration
}
```

### titles.json mis à jour

```json
[
  {
    "title_id": "1144039928",
    "steam_app_id": "1336960",
    "name": "Halo Infinite",
    "sync_target": "halo",
    "poll_interval_s": 30,
    "cooldown_s": 300
  },
  {
    "title_id": "730",
    "steam_app_id": "730",
    "name": "Counter-Strike 2",
    "sync_target": "cs2",
    "poll_interval_s": 60,
    "cooldown_s": 120
  }
]
```

### Variable d'environnement à ajouter

```bash
# .env.local
STEAM_API_KEY=...    # clé gratuite sur steamcommunity.com/dev/apikey
```

-----

## Source des joueurs trackés — db_profiles.json

Le watcher lit `db_profiles.json` au démarrage pour obtenir la liste des joueurs à surveiller.
Pas de config supplémentaire nécessaire : si un joueur a un profil, il est tracké.

```go
// config/profiles.go
type PlayerProfile struct {
    Gamertag string `json:"gamertag"`
    XUID     string `json:"xuid"`       // si absent → résolu via API Xbox au démarrage
    SteamID  string `json:"steam_id"`   // optionnel — fallback si RTA silencieux
}

func LoadWatchedPlayers(path string) ([]PlayerProfile, error)
```

> Si `xuid` est absent, le watcher le résout au démarrage via l'API Xbox et le cache en mémoire.
> Si `steam_id` est présent, un `SteamPoller` est instancié en parallèle de l'abonnement RTA.

-----

## Détection de nouveaux matchs (watcher/match_poller.go)

```go
type MatchPoller struct {
    lastKnownID string          // dernier match_id connu, persisté sur disque
    titleConfig TitleConfig
    queue       *MatchQueue
}
```

**Algorithme :**
1. GET `/hi/players/xuid({xuid})/matches?count=5` (les 5 derniers suffisent)
2. Pour chaque match_id retourné, si absent du `seen` map → nouveau match
3. Enfile dans `queue`
4. Met à jour `lastKnownID`

**Persistence du `lastKnownID` :**
Stocké dans `data/watcher/{gamertag}/state.json` pour survivre aux redémarrages du daemon.
Au redémarrage, si le joueur est déjà en jeu (RTA event reçu immédiatement), le poller
reprend depuis le dernier ID connu et ne re-syncera pas les matchs déjà traités.

-----

## Reconnexion WebSocket automatique (presence/reconnect.go)

```go
type ReconnectPolicy struct {
    InitialDelay time.Duration  // 1s
    MaxDelay     time.Duration  // 5min
    Multiplier   float64        // 2.0 (backoff exponentiel)
}
```

**Déclencheurs de reconnexion :**
- Connexion WebSocket coupée (timeout, erreur réseau)
- Token XSTS expiré (détecté par la `refresh_loop`) → reconnecter avec le nouveau token
- `close frame` reçu du serveur RTA

La reconnexion resouscrit automatiquement à tous les topics (un par joueur tracké).
Les états des FSM joueurs **ne sont pas réinitialisés** lors d'une reconnexion : si un joueur
était en `Watching`, il y reste — le poller continue pendant que le WebSocket se reconnecte.

-----

## Notification (notify/)

Interface `Notifier` permettant de brancher plusieurs canaux :

```go
type Notifier interface {
    Notify(ctx context.Context, event NotifyEvent) error
}

type NotifyEvent struct {
    Gamertag  string
    TitleName string
    EventType string    // "session_start", "match_synced", "session_end"
    MatchIDs  []string  // pour "match_synced"
}
```

Implémentations initiales : `DiscordNotifier` (webhook). Extensible à Telegram, ntfy, etc.

-----

## State machine — transitions Steam

Les transitions Steam s'ajoutent à la FSM existante sans modifier les transitions RTA :

```
Signal Steam onActive(titre connu) ET RTA silencieux depuis >2min → Idle → Watching
Signal Steam onInactive()          ET état = Watching/Cooling      → Cooling (si RTA aussi silencieux)
Event RTA reçu                                                     → reprend la priorité, Steam ignoré
```

La FSM ne distingue pas l'origine du signal (RTA ou Steam) une fois en `Watching` —
le match poller fonctionne identiquement dans les deux cas.

-----

## Roadmap d'implémentation

| Phase | Composant | Priorité |
|-------|-----------|----------|
| 1 | `cmd/auth-setup` + `internal/auth/` (XSTS flow + token store) | Critique |
| 2 | `internal/presence/rta_client.go` + `reconnect.go` | Critique |
| 3 | `internal/titles/registry.go` + `config/profiles.go` (xuid + steam_id) | Haute |
| 4 | `internal/watcher/state_machine.go` + `match_poller.go` | Haute |
| 5 | `internal/watcher/match_queue.go` + `sync/coordinator.go` | Haute |
| 6 | `internal/sync/trigger.go` (appel sync in-process) | Haute |
| 7 | `internal/presence/steam_poller.go` + intégration FSM | Haute |
| 8 | `internal/watcher/provider.go` (WatcherStateProvider in-process) | Haute |
| 9 | `internal/notify/discord.go` | Moyenne |
| 10 | Tests d'intégration (reconnexion, multi-joueurs, fallback Steam) | Haute |
