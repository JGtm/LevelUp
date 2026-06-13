# PLAN — Bruit terminal auth/boot : root causes + observabilité dashboard

> Statut : IMPLÉMENTÉ (2026-06-11) — voir « Déviations en implémentation » ci-dessous
> Déviations :
> 1. Branche : exécuté sur `feat/achievements-category-filter` (demande user — WIP LUSR
>    concurrent dans le working tree interdisait le checkout depuis main).
> 2. BUG DÉCOUVERT ET CORRIGÉ en Phase 3 : la boucle boot de `pool.NewPool` (`i--` +
>    `poolSize--`) retentait le MÊME index après un échec et ABANDONNAIT silencieusement
>    toutes les sources suivantes (explique le burst de 7 tentatives DankerGlue ET des
>    comptes sains absents du pool). Test de régression : TestNewPool_SkipsFailingSourceAndKeepsTheRest.
> 3. Phase 4b (purge sync_meta post-migration) VOLONTAIREMENT REPORTÉE : le double-write
>    compat `onRotated` (store + sync_meta, retiré Phase 5 ADR-0023) ré-écrirait le RT à
>    chaque rotation — purger au boot serait du churn contradictoire. Le fix 4a (warn
>    only-when-adopted) suffit à éteindre le bruit. La purge se fera avec la Phase 5
>    ADR-0023 (retrait des lectures/écritures legacy).
> Branche cible : `fix/auth-warning-noise` (créée depuis `main` — tâche indépendante des achievements)
> Objectif : ~95 % du bruit WARN récurrent éliminé en traitant les causes (pas en filtrant les logs),
> et l'état de santé auth visible dans le dashboard admin au lieu du terminal.

## Critère de succès global

1. Un boot serveur + 1 h de fonctionnement ne produit plus que : 1 ligne config dev,
   éventuellement 1 WARN par compte à problème (première occurrence), et les one-shots
   transitoires légitimes (réseau).
2. Le dashboard admin montre, par joueur : santé tokens (existant) + classe du dernier
   échec OAuth + source de credentials.
3. Aucun changement de comportement pour les 4 comptes principaux (refresh OK aujourd'hui).

## Rappel du diagnostic (2026-06-11)

| Famille | Cause racine | Fichier clé |
|---|---|---|
| 1. AADSTS90023 ×5 comptes, toutes les ~15 min (~80 % du bruit) | `client_secret` envoyé pour des RT émis en flux client public (token-capture) ; échec permanent retraité comme transitoire (aucun cache négatif) | `internal/platform/auth/oauth_refresh.go:106`, `internal/platform/auth/pool/resolver.go` |
| 2. « legacy sync_meta utilisée » ×4 comptes à chaque boot | Warn émis dès qu'une valeur legacy est LUE même si non utilisée ; résidus sync_meta jamais purgés (Phase 5 ADR-0023 différée) | `internal/platform/auth/pool/discovery.go:147-158, 203-220` |
| 3. « migrations player échouées » ×5 comptes à chaque boot | Les comptes watcher (db_path vide, donneurs de tokens) n'ont pas de player DB ; le boot tente quand même la migration → IO Error | `cmd/server/main.go:375-390` |
| 4. « configuration non sûre » ×3 lignes à chaque boot | Intentionnel en dev (garde prod existe) mais 3 lignes au lieu d'1 | `cmd/server/main.go:281-289`, `internal/config/config.go:191-213` |

Comptes : 4 principaux (JGtm, Chocoboflor, Madina97294, XxDaemonGamerxX — refresh OK,
RT flux web confidentiel) ; 5 watcher (DankerGlue, Trimbutton, QuiteSiren, UppedJoker,
GeleJugefi — RT flux public, `db_path` vide dans db_profiles.json).

---

## Phase 1 — Quick wins boot (familles 3 + 4) — effort : rapide, risque : faible

### 1a. Skip migrations player quand la DB n'existe pas

- `cmd/server/main.go` (boucle ~375-390) : avant `RunPlayerMigrations`, `os.Stat` sur
  `pr.PlayerDBPath(titleSlug, p.Gamertag)`. Fichier absent → `slog.Debug("migrations player
  ignorées — player DB absente (compte token-only)", "gamertag", ...)` et continue.
- Ne PAS créer la DB : la création appartient au chemin sync/onboarding, pas au boot.
- La migration reste tentée (et peut échouer en WARN) si le fichier EXISTE — un vrai lock
  reste visible.

### 1b. Consolidation des 3 warns config en 1 ligne

- `cmd/server/main.go:285-289` : remplacer la boucle par UN `slog.Warn("configuration non
  sûre pour un déploiement multi-user exposé", "issues_count", n, "issues", strings.Join(...,
  " | "), "prod_guard", ...)`.
- `config.SecurityWarnings()` inchangé (testé, source unique).

**Tests** : helper extrait `shouldRunPlayerMigrations(path) bool` si trivialement testable,
sinon vérification manuelle au boot. Pas de test pour 1b (formatage de log).
**Done** : boot sans WARN famille 3/4 (sauf 1 ligne config consolidée).

---

## Phase 2 — Root cause AADSTS90023 + erreurs typées + compteurs expvar — effort : moyen, risque : moyen

### 2a. Erreur OAuth typée

- `internal/platform/auth/oauth_refresh.go` : nouvelle struct `OAuthExchangeError{ErrorCode,
  Description string}` avec `Error()`, exposée via `errors.As`. Helper de classification :
  `func (e *OAuthExchangeError) Class() AuthErrorClass` →
  - `AuthErrorConfig` : `invalid_request` + description contenant `AADSTS90023` (ou
    `invalid_client`, `unauthorized_client`)
  - `AuthErrorRevoked` : `invalid_grant`
  - `AuthErrorTransient` : tout le reste (réseau, 5xx, JSON invalide)
- Les retours d'erreur de `ExchangeRefreshTokenWithRotation` portent ce type.

### 2b. Retry sans client_secret sur AADSTS90023 (stateless)

- Dans `ExchangeRefreshTokenWithRotation` : si la 1re tentative (avec secret, comportement
  actuel conservé — les 4 comptes principaux en dépendent) échoue avec classe Config/90023
  ET que le secret avait été envoyé → retenter UNE fois sans `client_secret`.
  - Succès → log `slog.InfoContext` « oauth_refresh: secret refusé (AADSTS90023), retry
    public OK » + retour normal. Coût : 1 round-trip HTTP supplémentaire par refresh
    (~toutes les 3 h 30 par compte concerné) → pas de mémorisation nécessaire.
  - Échec (probablement `invalid_grant` → RT mort) → retourner la 2e erreur typée.
- Refactor extraction : la construction+envoi de la requête part dans une sous-fonction
  `postTokenRequest(ctx, body)` pour rester < 80 lignes.
- Testabilité : promouvoir `msalTokenURL` en `var` package (override par les tests httptest).

### 2c. Compteurs expvar (pattern `pool/metrics.go`, ADR 0009)

- Nouveau `internal/platform/auth/metrics.go` :
  - `levelup.auth.oauth_refresh_total`, `levelup.auth.oauth_refresh_fail_total`
  - `expvar.Map` `levelup.auth.oauth_refresh_fail_by_class` (config / revoked / transient)
  - `levelup.auth.oauth_refresh_retry_public_total` (compteur du fallback 2b)
- Incrémentés dans `ExchangeRefreshTokenWithRotation`.

**Tests** (`oauth_refresh_test.go`, httptest) :
- succès direct avec secret ; 90023 → retry sans secret → succès (vérifier que le 2e POST
  ne contient pas `client_secret`) ; 90023 → retry → `invalid_grant` → erreur classe Revoked ;
  pas de secret env → aucun retry ; classification unitaire des 3 classes.
**Done** : si les RT des 5 watchers sont encore vivants, ils refreshent à nouveau ;
sinon l'erreur devient `invalid_grant` (classe Revoked) → traitée par la Phase 3.
**Issue possible documentée** : RT watcher morts (rotation perdue pendant la période 90023)
→ action user : `go run ./cmd/token-capture/ <Gamertag>` ×5. Le plan reste valide,
le dashboard (Phase 5) l'affichera.

---

## Phase 3 — Cache négatif resolver + 1 seul WARN + persistance de l'erreur — effort : moyen, risque : moyen

### 3a. Cache négatif par gamertag dans le resolver

- `internal/platform/auth/pool/resolver.go` : nouveau champ `negative map[string]*negativeEntry`
  (`{class, err, until time.Time, warned bool}`), protégé par le mutex existant.
- Dans `resolveExpensive` : erreur de classe **Config** ou **Revoked** → entrée négative
  (TTL : 1 h config, 6 h revoked). Classe Transient → pas de cache négatif (comportement
  actuel conservé).
- Dans `Resolve` (avant singleflight) : entrée négative non expirée → retour immédiat de
  l'erreur mémorisée, log `slog.DebugContext` (« resolve court-circuité — échec permanent
  récent », avec classe + until). PAS d'appel réseau, PAS de WARN.
- Invalidation : `Refresh(gamertag)` (rotation post-token-capture / SSO) purge l'entrée
  négative du gamertag — c'est déjà le chemin appelé après re-capture. Vérifier le câblage
  `halo.InvalidateCachedPlayerTokens` → si la re-capture ne passe pas par `Refresh`,
  exposer `InvalidateNegative(gamertag)` et l'appeler au même endroit.

### 3b. Un seul WARN par échec (au lieu de 4 lignes empilées)

- `oauth_refresh.go:80,135` : les deux WARN deviennent `slog.DebugContext` (l'erreur est
  propagée et loggée plus haut ; les compteurs expvar 2c gardent la trace quantitative).
- `sisu_provider.go:253` (`TryOAuthRefreshWithRotation erreur`) : WARN → Debug (même raison).
- `resolver.go:219` (`pool/resolver: TryOAuthRefresh erreur`) : RESTE le WARN unique,
  enrichi avec `"class"`. Émis seulement à la PREMIÈRE occurrence de la fenêtre négative
  (flag `warned` de l'entrée) ; les occurrences suivantes sont en Debug.
- Audit rapide des autres consommateurs directs du provider (watcher daemon, presence
  rest_poller) : s'ils appellent `TryOAuthRefresh*` sans passer par le resolver, vérifier
  qu'ils ne re-créent pas une pile de WARN (au besoin router leur log au niveau Debug).

### 3c. Persistance de l'erreur dans le store (donnée du dashboard)

- `internal/platform/auth/multi_user_token_store.go` — `UserTokens` += 3 champs (omitempty) :
  `LastAuthErrorClass string`, `LastAuthError string` (message court, SANS token),
  `LastAuthErrorAt time.Time`.
- Resolver : nouveau callback optionnel `onAuthError(ctx, gamertag, xuid, class, msg)`
  (même pattern que `onReauth`/PR-B, câblé au même endroit dans `cmd/server/main.go`) →
  écrit ces champs dans le store. Un refresh réussi les efface (même cycle de vie que
  `ReauthRequired`).
- `invalid_grant` continue de déclencher `onReauth` (existant, inchangé).

**Tests** (`resolver_test.go`, mock provider) :
- erreur Config → entrée négative ; 2e Resolve dans la fenêtre → 0 appel provider (compteur
  d'appels du mock) + même erreur ; après TTL → nouvel appel.
- erreur Transient → PAS de cache négatif.
- `Refresh()` purge l'entrée négative.
- callback `onAuthError` invoqué avec la bonne classe ; effacé après succès.
**Done** : le burst boot (7 tentatives DankerGlue) tombe à 1 WARN ; les cycles 15 min
deviennent silencieux (Debug) tant que l'état n'a pas changé.

---

## Phase 4 — Discovery : warns legacy honnêtes + purge post-migration (Phase 5 ADR-0023 partielle) — effort : rapide/moyen, risque : faible

### 4a. Warn « legacy utilisée » seulement si la valeur est retenue

- `discovery.go` : `readLegacyDuckDB` ne logue plus ; il retourne `(msal, oauth, ok)`.
  Le caller (`scanPlayer`) émet le WARN uniquement pour les champs réellement adoptés
  (`msal == "" && dbMsal != ""` / `oauth == "" && dbOauth != ""`), avec un attribut
  `"fields"` explicite (`msal`, `oauth`).
- Optimisation associée : si le store a déjà fourni le RT et qu'il ne manque que le MSAL,
  l'ouverture sync_meta reste utile (lecture MSAL) mais ne doit plus warner pour le RT ignoré.

### 4b. Purge des résidus sync_meta après migration confirmée

- `internal/platform/auth/migration.go` (`MigrateLegacyTokens`) : quand le store possède
  déjà un RT pour le joueur (cas `PlayersSkipped` actuel) OU après copie réussie →
  effacer `sync_meta.oauth_refresh_token` / `sync_meta.msal_token_cache` dans la player DB
  (UPDATE → '' via `SELECT-then-UPDATE` — player DB legacy sans PK, cf. mémoire projet ;
  pas d'ON CONFLICT).
  - Garde-fou : ne purger que si la valeur store est non vide ET `UpdatedAt` du store
    plus récent que le boot précédent (le store est la source de vérité post-rotation ;
    les valeurs sync_meta sont des RT morts depuis longtemps — rotation Microsoft).
  - Log `slog.InfoContext` « auth_migration: résidus sync_meta purgés », une fois par joueur.
- `.env.local` : les 4 `SPNKR_OAUTH_REFRESH_TOKEN_*` ne sont PAS touchées par le code.
  Livrer dans le message de fin la liste exacte des lignes à supprimer manuellement
  (action user, fichier non versionné). Le warn env var (discovery prio 3) reste, il est
  déjà « only-when-used ».

**Tests** : discovery (store RT + sync_meta RT → 0 warn ; store vide + sync_meta RT → 1 warn
champ oauth) ; migration (purge effectuée quand store déjà peuplé ; sync_meta intact si
store vide). DuckDB `:memory:` ou fixture player DB temporaire.
**Done** : boot sans warn famille 2 pour les 4 comptes principaux dès le 2e démarrage.

---

## Phase 5 — Observabilité dashboard admin — effort : moyen, risque : faible

### 5a. Backend — exposer classe d'erreur + source de credentials

- `internal/domain/admin_token_health.go` — `PlayerTokenHealth` += `LastAuthErrorClass`,
  `LastAuthError`, `LastAuthErrorAt` (depuis les champs store de 3c) et `CredentialSource`.
- Source de credentials SANS I/O supplémentaire : nouveau snapshot en mémoire dans `pool` —
  `Scan()` enregistre `map[gamertag]{source, at}` (mutex package ou struct Discovery),
  exposé via `pool.LastScanSnapshot()`. Le handler `admin_token_health.go` merge :
  source = snapshot si présent, sinon `"unknown"` (pas de scan depuis le boot).
- Pas de nouvelle route : enrichissement de `GET /admin/token-health` existant.
- Architecture : handler → service/port existants ; aucun accès DuckDB ajouté.

### 5b. Frontend — colonne Source + badge erreur dans TokenHealthSection

- `apps/web/src/lib/api/types.ts` : champs ajoutés sur le type TokenHealth existant.
- `AdminPage.tsx` (`TokenHealthSection`, ~566) :
  - colonne « Source » (libellé court : `store`, `sync_meta`, `env`, `legacy`, `unknown`) —
    valeur ≠ `store/watcher_*` rendue en `tokenCssVar('warning')` (dette ADR-0023 visible) ;
  - si `last_auth_error_class` présent : badge destructive/warning selon la classe
    (`revoked`/`config` → destructive, `transient` → warning) avec tooltip
    (message + date + action suggérée : « re-capture requise » pour revoked).
- i18n : clés `common.admin.token_source_*`, `common.admin.token_error_*` ajoutées dans le
  manifest TOML commun (FR + EN) + régénération du manifest (pipeline existant).
  Aucune couleur hex ; uniquement `tokenCssVar`.

**Tests** : Go — handler/service token-health (champ source mergé, erreurs exposées) ;
front — typecheck + extension du test AdminPage s'il existe, sinon test léger du mapping
statut→couleur. Vitest hors sandbox.
**Done** : un coup d'œil au dashboard répond à « pourquoi ce compte ne refresh plus ? »
et « qui n'est pas encore migré au store ? » sans ouvrir le terminal.

---

## Découpage commits / livraison

1 branche, 5 commits (1 par phase), chaque phase livrable indépendamment :
1. `fix(boot): migrations player skippées sans DB + warns config consolidés`
2. `fix(auth): retry public sur AADSTS90023 + erreurs OAuth typées + expvar`
3. `fix(auth): cache négatif resolver — 1 WARN par échec permanent + persistance store`
4. `fix(auth): warns legacy only-when-used + purge sync_meta post-migration`
5. `feat(admin): santé tokens — classe d'erreur + source de credentials`

Dépendances : 3 dépend de 2 (classes d'erreur) ; 5 dépend de 3 (champs store) ;
1 et 4 indépendants.

## Vérifications transverses (chaque phase)

- `go build ./...`, `go vet ./...`, `go test ./internal/platform/auth/... ./internal/...`
  (CGO ucrt64) ; front : typecheck + lint + vitest (hors sandbox).
- Pas de `fmt.Println` ; clés slog standard (`err`, `player`/`gamertag`, `titleSlug`, `class`).
- JAMAIS de token/secret dans un message de log ou un champ store d'erreur.
- Validation finale réelle : redémarrer le serveur local, comparer le volume de WARN sur
  boot + 1 h vs Term_log.txt de référence (2026-06-11). Vérifier que les 4 comptes
  principaux refreshent toujours (fichiers store rotatés).
- thought_log : 1 entrée par phase commitée ; accord user avant chaque commit.

## Hors scope (documenté)

- Suppression complète des lectures legacy (Phase 5 ADR-0023 totale : retirer sync_meta/env
  du discovery) — après une période d'observation sans warn « legacy utilisée ».
- Re-capture des 5 RT watcher si morts (action user, `cmd/token-capture`).
- `world_leaderboard_cron` / `rest_poller` : déjà bien gérés, transitoires.
