# Watcher RTA Auth Refresh - Reprise de chantier

> Statut: ouvert
> Priorite: haute
> Origine: session Copilot interrompue `2e5ac0fa-bdb9-48b2-9f7d-07282a67c40d`
> Date d'arret constatee: 2026-04-20

## Resume executif

Le chantier watcher/RTA est presque complet, mais le chainage final
`status=3 -> refresh XSTS -> UpdateAuth -> reconnexion`
 n'est pas encore cable de bout en bout.

Le code actuel dispose deja des briques suivantes:

- `internal/presence/rta_client.go` detecte `status=3`, positionne `authExpired`,
  et ferme la connexion pour forcer une reconnexion.
- `internal/presence/reconnect.go` expose un callback `OnAuthExpired`,
  mais `RunWithReconnect()` ne l'utilise pas encore.
- `internal/watcher/daemon.go` instancie `ReconnectManager`,
  mais ne lui branche pas de callback de refresh d'auth.
- `cmd/server/main.go` fait un refresh XSTS proactif au demarrage et met deja
  a jour le daemon via la `RefreshLoop`, ce qui fournit la base du comportement voulu.

Conclusion: il manque surtout le cablage on-demand au moment ou le serveur RTA
refuse un subscribe a cause d'un token XSTS expire.

## Perimetre exact

### Dans le scope

- `apps/go-api/internal/presence/rta_client.go`
- `apps/go-api/internal/presence/reconnect.go`
- `apps/go-api/internal/watcher/daemon.go`
- `apps/go-api/cmd/server/main.go`
- tests associes sous `apps/go-api/internal/presence/` et `apps/go-api/internal/watcher/`

### Hors scope

- UI React du watcher
- fallback Steam
- FSM joueur
- queue de matchs / coordinator
- notifications

## Etat actuel du code

### 1. Signal d'expiration deja emis cote client RTA

Dans `apps/go-api/internal/presence/rta_client.go`:

- le champ `authExpired atomic.Bool` existe
- `Connect()` reinitialise ce flag a `false`
- `UpdateAuth()` existe deja
- `IsAuthExpired()` et `ResetAuthExpired()` existent deja
- `handleSubscribeResponse()` positionne `authExpired=true` sur `status == 3`
- `handleSubscribeResponse()` ferme aussi la connexion pour casser la boucle RTA courante

Autrement dit, le point de detection est deja implemente.

### 2. ReconnectManager annonce le bon hook, mais ne l'utilise pas encore

Dans `apps/go-api/internal/presence/reconnect.go`:

- `ReconnectManager` contient deja:

```go
OnAuthExpired func(ctx context.Context) error
```

- le commentaire dit explicitement que ce callback doit:
  - obtenir un XSTS frais
  - appeler `client.UpdateAuth`
  - appliquer un delai de 30s si le callback est absent ou echoue

Mais aujourd'hui, `RunWithReconnect()`:

- tente `connectFunc(ctx)`
- lance `ReadLoop(ctx)`
- repart en boucle apres la deconnexion

Sans jamais verifier `r.client.IsAuthExpired()` et sans jamais appeler `OnAuthExpired`.

### 3. Le daemon ne branche pas le callback

Dans `apps/go-api/internal/watcher/daemon.go`:

- `connectAndSubscribe()` cree bien un `ReconnectManager`
- mais ne renseigne pas `reconnectMgr.OnAuthExpired`
- `DaemonConfig` ne fournit encore aucun hook de refresh d'auth RTA a la demande

Le daemon sait deja mettre a jour son header via:

```go
func (d *Daemon) UpdateAuth(authHeader string)
```

mais cette mise a jour n'est aujourd'hui utilisee que par la boucle de refresh proactive.

### 4. Le bootstrap main.go a deja la logique reusable

Dans `apps/go-api/cmd/server/main.go`:

- un refresh XSTS proactif est deja fait avant `daemon.Start(...)`
- `auth.NewRefreshLoop(...)` appelle deja `daemon.UpdateAuth(result.AuthHeader())`

Donc le chantier ne consiste pas a inventer une nouvelle voie de refresh,
mais a reutiliser cette capacite quand le RTA signale un `status=3`.

## Gap fonctionnel exact

Scenario cible:

1. le token XSTS expire
2. un `Subscribe` RTA recoit `status=3`
3. `RTAClient` positionne `authExpired=true` et ferme la connexion
4. la boucle de reconnexion detecte cet etat
5. un refresh XSTS on-demand est execute
6. `UpdateAuth()` installe le nouveau header
7. la reconnexion repart avec un token frais

Scenario actuel:

1. le token XSTS expire
2. un `Subscribe` RTA recoit `status=3`
3. `RTAClient` positionne `authExpired=true` et ferme la connexion
4. la boucle de reconnexion repart
5. aucun refresh on-demand n'est execute
6. le reconnect peut reutiliser un header stale
7. risque de boucle `reconnect -> subscribe refuse -> reconnect`

## Decision technique recommandee

### Option recommandee: callback de refresh injecte au daemon

Conserver l'architecture actuelle et finir le cablage minimal:

1. ajouter dans `watcher.DaemonConfig` un callback explicite, par exemple:

```go
RefreshRTAAuth func(ctx context.Context) error
```

2. dans `cmd/server/main.go`, fournir une closure qui:
   - recharge les tokens via `store.Load()`
   - verifie la presence d'un `AccessToken`
   - appelle `auth.AcquireXSTSForRTA(ctx, tokens.AccessToken)`
   - persiste le resultat via `store.UpdateXSTS(result, 55*time.Minute)`
   - pousse le nouveau header dans le daemon via `daemon.UpdateAuth(result.AuthHeader())`

3. dans `watcher.connectAndSubscribe()`, brancher:

```go
reconnectMgr.OnAuthExpired = d.cfg.RefreshRTAAuth
```

4. dans `ReconnectManager.RunWithReconnect()`, verifier en debut de tour:
   - si `r.client.IsAuthExpired()` est `false` -> flux actuel inchange
   - si `true` -> appeler `OnAuthExpired(ctx)` avant `connectFunc(ctx)`
   - si succes -> `r.client.ResetAuthExpired()`
   - si callback absent ou en erreur -> log + delai fixe 30s + retry

Cette option garde les responsabilites propres:

- `presence/` detecte et orchestre
- `watcher/` cable les composants
- `main.go` connait le `TokenStore` et le refresh auth concret

## Plan d'implementation propose

### Etape 1 - Finir `ReconnectManager`

Fichier: `apps/go-api/internal/presence/reconnect.go`

Ajouter:

- une constante de retry auth, par exemple `authRefreshRetryDelay = 30 * time.Second`
- un helper interne du style:

```go
func (r *ReconnectManager) refreshAuthIfNeeded(ctx context.Context) error
```

Comportement attendu:

- si `!r.client.IsAuthExpired()` -> return `nil`
- si `OnAuthExpired == nil` -> log + return erreur sentinelle ou appliquer le delai ici
- si callback echoue -> log + return erreur
- si callback reussit -> `r.client.ResetAuthExpired()` puis return `nil`

Puis appeler ce helper juste avant `connectFunc(ctx)`.

### Etape 2 - Exposer le callback via le daemon

Fichier: `apps/go-api/internal/watcher/daemon.go`

Ajouter dans `DaemonConfig`:

```go
RefreshRTAAuth func(ctx context.Context) error
```

Puis, dans `connectAndSubscribe()`:

```go
reconnectMgr.OnAuthExpired = d.cfg.RefreshRTAAuth
```

### Etape 3 - Fournir le refresh concret dans `main.go`

Fichier: `apps/go-api/cmd/server/main.go`

Au moment du `watcher.NewDaemon(...)`, injecter une closure qui:

1. relit le `TokenStore`
2. prend `tokens.AccessToken`
3. appelle `auth.AcquireXSTSForRTA(...)`
4. persiste `UpdateXSTS(...)`
5. appelle `daemon.UpdateAuth(...)`

Point d'attention:

- la closure a besoin d'une reference au daemon.
- le plus simple est souvent de construire le daemon, puis d'affecter un champ de config
  ou de le fournir via une option si la construction actuelle ne le permet pas directement.

### Etape 4 - Garder la boucle proactive intacte

Le refresh loop existant dans `main.go` doit rester en place.

Le comportement cible devient:

- refresh proactif normal par `RefreshLoop`
- refresh reactif on-demand en cas de `status=3`

Les deux ne sont pas concurrents sur le plan fonctionnel; le reactive couvre juste
la fenetre ou le token expire entre deux checks.

## Tests a ajouter ou etendre

### Tests deja presents

Le repo contient deja des tests utiles dans
`apps/go-api/internal/presence/presence_test.go`:

- `TestRTAClient_UpdateAuth`
- `TestReconnectManager_BackoffDelay`
- test de `status=3` cote `RTAClient`

### Tests manquants prioritaires

#### `presence_test.go`

1. `TestReconnectManager_RunWithReconnect_AuthExpired_CallsOnAuthExpiredBeforeConnect`
   - simuler `authExpired=true`
   - verifier que `OnAuthExpired` est appele avant `connectFunc`

2. `TestReconnectManager_RunWithReconnect_AuthExpired_ResetAfterSuccess`
   - apres succes du callback, verifier `IsAuthExpired() == false`

3. `TestReconnectManager_RunWithReconnect_AuthExpired_CallbackError`
   - verifier qu'un echec du callback n'appelle pas `connectFunc` immediatement

4. `TestReconnectManager_RunWithReconnect_AuthExpired_NoCallback`
   - verifier le comportement quand `OnAuthExpired` est nil

#### `daemon_test.go`

5. test que `connectAndSubscribe()` branche bien le callback fourni par config

### Facilitation de tests recommandee

Pour eviter un vrai `sleep 30s` dans les tests, injecter un helper de temporisation
dans `ReconnectManager`, par exemple:

```go
waitFn func(ctx context.Context, d time.Duration) bool
```

avec implementation par defaut basee sur `time.After`, et version testable sans attente reelle.

## Criteres d'acceptation

Le chantier sera considere termine quand les conditions suivantes seront vraies:

1. un `status=3` sur un subscribe ne provoque plus de boucle infinie de reconnexion
2. un refresh XSTS on-demand est execute avant la tentative de reconnexion suivante
3. `daemon.UpdateAuth()` recoit bien le nouveau header avant `Connect()`
4. le flag `authExpired` est reset uniquement apres succes du refresh
5. la boucle proactive `RefreshLoop` continue de fonctionner sans regression
6. les tests cibles passent

## Commandes de validation ciblees

Depuis `apps/go-api/`:

```bash
go test ./internal/presence ./internal/watcher ./internal/platform/auth -count=1
```

Si un test de scenario plus large existe ensuite:

```bash
go test ./... -count=1
```

## Risques et points de vigilance

### Access token absent ou invalide

Si `store.Load()` retourne un `AccessToken` vide ou deja inutilisable,
le callback de refresh doit echouer proprement et laisser des logs clairs.

### Double source de refresh

Le refresh proactif et le refresh reactif peuvent se croiser.
Ce n'est pas un probleme si:

- `UpdateXSTS()` reste idempotent au niveau metier
- `UpdateAuth()` peut etre appele plusieurs fois avec le dernier header valide

### Granularite du signal `status=3`

Le code actuel traite `status=3` comme un signal global d'expiration d'auth.
Vu qu'une seule connexion partage le meme header pour tous les joueurs,
ce choix reste coherent et acceptable.

## Definition de done

Le chantier est termine seulement si:

- le code est cable de bout en bout
- les tests presence/watcher/auth sont au vert
- les logs permettent d'identifier clairement le chemin `status=3 -> refresh -> reconnect`
- le comportement est documente implicitement dans le code via noms et commentaires,
  sans avoir besoin de re-lire ce markdown pour comprendre le flux
