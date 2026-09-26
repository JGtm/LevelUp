# Plan — Multi-titre : PÉRIPHÉRIE (axes hors chemin data-lecture)

> **Rôle** : specs détaillées des axes multi-titre **oubliés** par le master title-agnostic
> (audit 2026-06-13/14) — tout ce qui entoure le chemin data-lecture du match :
> ingestion, acquisition auth, scheduler, settings, achievements, world-stats, outcome,
> observabilité, Discord, cycle de vie, registre de migrations, garde-fous.
>
> **Index / registre complet** : [PLAN_MULTITITRE_INDEX.md](PLAN_MULTITITRE_INDEX.md) (`MT-01..MT-26` + carte des phases).
> **Master data-path** : [PLAN_TITLE_AGNOSTIC_REFACTORING.md](PLAN_TITLE_AGNOSTIC_REFACTORING.md) + [tracker](PLAN_TITLE_AGNOSTIC_TRACKER.md).
> **Créé** : 2026-06-14 · **Génération** : 16 agents (re-vérification du code AVANT rédaction) + relecteur de cohérence (audit `wpdukv3sr` → rédaction `wbmehosde`/`wmlrszolc`).

---

## ⚠ Doctrine RE-VÉRIFIER (obligatoire avant l'exécution de TOUTE phase)

Les pointeurs `file:line` et l'analyse de chaque phase sont une **carte datée (2026-06-13/14), PAS une vérité figée**.
La re-vérification de cet audit a DÉJÀ corrigé des erreurs de la passe initiale — preuve que les seeds ne sont pas fiables tels quels :

- `weapon_labels` DDL = `steps_metadata.go:564-568`, **pas** `:463-640` (l'audit pointait `applyWeaponLabels`).
- `discovery_client.go:20` = `discoveryUGCHost` (gamecms-hacs), **pas** `defaultStatsHost` ; le host stats inline est à `:82`.
- audience Spartan Halo `"urn:343:s3:services"` (`halo_exchange.go:182`) **manquée** par l'audit, c'est un 4ᵉ leg auth.
- `BuildEngine` (`auto_sync.go:255`) est le point de câblage **UNIQUE** watcher + scheduler + HTTP (le watcher ne construit pas son propre engine).
- `MetadataDBPath(slug)` isole **déjà** metadata par chemin → MT-16 (`title_id` colonnes) devient une **décision** defense-in-depth, pas une migration obligatoire.

**Avant de démarrer une phase, l'agent DOIT** : (1) re-grep/re-lire chaque évidence contre `HEAD` ;
(2) re-valider que le couplage existe encore et que les `file:line` sont à jour ;
(3) re-scoper (le gap a pu rétrécir/grossir) ; (4) consigner la dérive dans la PR.
Chaque section **Dérive re-vérif** ci-dessous date du 2026-06-14 — à refaire, pas à croire.

---

## Méthode imposée : `expand → parity-gate → contract`

Idiome du projet (parallel-change / strangler ; déjà appliqué en 1.6 / 1.7a / 1.9). Trois temps par axe :

1. **Expand (factoriser sans swap)** — introduire le seam title-aware avec `halo_infinite` câblé aux **valeurs ACTUELLES** (zéro changement de comportement). Couches arch-rules : descripteur/port dans `domain/title` ou `internal/games`, résolution dans `platform/*`, routage `api/*` ; **jamais** `slug == "halo_infinite"` (garde `archlint/no_slug_comparison`), tout chemin via `PathResolver`, gating par `HasCapability()`.
2. **Parity-gate — ORACLE DOUBLE (non négociable)** :
   - **(a) Parité Halo byte-identique** : test de caractérisation / golden sur la sortie actuelle (URL, body, chemins DB, JSON…) **à travers le seam** pour `halo_infinite`. Zéro diff = comportement inchangé.
   - **(b) Exercice `synthetic_test_title`** : un titre fixture qui prouve que le seam route **vraiment** différemment **ET** dégrade proprement quand la capability/donnée est absente. **Non optionnel** : sans (b), la factorisation est cosmétique (abstraction à une seule implémentation — elle compile, Halo passe, et le 2ᵉ titre révèle que le seam est faux).
3. **Contract (swap)** — retirer le hardcode / basculer les callers en **PR MINCE par axe**, jamais un méga-swap (cf. le revert du cluster `steps_shared.go`). La garde lint reste verte à chaque PR.

**Ordre imposé par les dépendances** (cf. DAG) : les 2 bloquants se factorisent **en premier** (seule porte pour exercer end-to-end le titre synthétique).

---

## DAG de dépendances (ordre d'exécution)

```
        PMT-2 (acquisition auth)          PMT-1 (hosts ingestion)
                  \                              /
                   \____________________________/
                                │  (acquérir/fetch un 2e titre devient possible)
                                ▼
                   PMT-3 (scheduler/sync : threader titleSlug → écriture)
                                │  (écriture per-title sûre — sinon un 2e titre écrase les DB Halo)
            ┌──────────────┬────┴─────┬──────────────┬───────────────┐
            ▼              ▼          ▼              ▼               ▼
       EXT-1.5         PMT-4       PMT-7          EXT-2          PMT-9
   (DDL/ops/meta)   (settings)  (world-stats)  (career/LUSR)  (migrations/titre)
                                │
   PMT-12 (cutoffs MT-09) : séquencer APRÈS PMT-3 (ne pas livrer le validateur avant le threading slug).

Indépendants (parité Halo seule, aucun prérequis dur) :
   PMT-5 (outcome) · PMT-6 (achievements) · PMT-8 (lifecycle) · PMT-10 (observabilité)
   · PMT-11 (Discord) · EXT-5 (front) · PMT-13 (mineurs, décision documentée)
```

**Racine du DAG = PMT-1 + PMT-2** (bloquants). Rien d'aval n'est *exerçable* end-to-end (oracle b) tant qu'on ne peut pas fetch+authentifier un 2e titre.

---

## Specs (17 phases — re-vérifiées 2026-06-14)

---

### PMT-1 — Ingestion title-aware (hosts API)  (sévérité: blocker)

**Axes** : MT-01

**Statut couverture actuelle** : gap — chaque host Halo est un `const` package-level ou un littéral inline ; aucun seam ne dérive l'host du titre. `HaloProvider.WithTitleSlug` stocke le slug mais ne l'utilise jamais pour choisir un host (provider.go:230-251), et `FetchServiceRecord(ctx, gamertag, titleSlug)` reçoit déjà un slug qu'il ignore (compare_provider.go:70-82).

**Évidence (⚠ RE-VÉRIFIER avant exécution — pointeurs vérifiés 2026-06-14)** :
- `internal/sync/halo_client.go:46-47` — `haloStatsHost`, `haloGameCMSHost` const ; consommés halo_client.go:214,253 (matches/stats) + 152-153 (NewHaloAPIClient initialise `economyBaseURL`/`gameCMSBaseURL` en dur).
- `internal/sync/halo_client_career.go:169-181` — `economyHost()` / `gameCMSHost()` : littéral `https://economy.svc.halowaypoint.com` (171) + fallback `haloGameCMSHost` (178). Appelés 23,70,85.
- `internal/sync/halo_skill.go:46` — `haloSkillHost` const (skill.svc).
- `internal/sync/halo_client_film.go:20` — `haloUGCHost = discovery-infiniteugc.svc` const.
- `internal/platform/halo/provider.go:98-99` — `defaultEconomyHost`, `defaultChallengesHost` const ; utilisés 301 (battlepass), 413,423 (challenges), clé singleflight 411-415.
- `internal/platform/halo/privacy_provider.go:68` — `defaultStatsHost` const ; utilisé 88 (matches-privacy).
- `internal/platform/halo/compare_provider.go:80,173` — réutilise `defaultStatsHost` pour servicerecord ; `FetchServiceRecord` a un param `titleSlug` non câblé.
- `internal/platform/halo/discovery_client.go:20` — `discoveryUGCHost = gamecms-hacs.svc` (⚠ seed disait `defaultStatsHost` — FAUX) ; **+ littéral inline halostats** à discovery_client.go:82 (stats par match).
- Hosts hors seed à inclure : `internal/assets/fetcher_gamecms.go:13` (`defaultGameCMSBase`), `internal/platform/halo/season_provider.go:19` (`defaultGameCMSHost`), `internal/sync/spartan_nameplate_resolver.go:34` (`nameplateHost`). **Exclus de PMT-1** (auth) : `internal/platform/auth/halo_exchange.go:30-31` (spartan-token/clearance settings.svc) → PMT-2.
- Pas de seam existant : aucun `constants.toml`, pas de champ `[endpoints]` sur `TitleDescriptor` (registry.go:44-56), pas d'`EndpointResolver`. Le loader mappings lit déjà `config/titles/{slug}/mappings/*.toml` via `mappings.Registry.LoadFromConfigDir(repoRoot, slugs, logger)` (registry.go:43-49) — point d'ancrage naturel pour un `constants.toml`.

**Méthode (expand → parity → contract)** :
- **Expand** : introduire `internal/games/mappings` → un `EndpointSet` (typé : `Stats`, `GameCMS`, `Economy`, `Skill`, `UGCFilm`, `DiscoveryUGC`, `Challenges`, `Nameplate` — un host par axe ingestion) chargé depuis une nouvelle section `[endpoints]` d'un `config/titles/{slug}/mappings/constants.toml`, parsé par `LoadEndpointsFromFile`/`...FromBytes` sur le modèle de loader.go, stocké dans la `mappings.Registry` (`r.endpoints[slug]`). Exposer un `EndpointResolver` (port côté `internal/games`) avec `HostFor(slug string, key EndpointKey) (string, bool)` — **clé d'endpoint, jamais comparaison de slug** (archlint no_slug_comparison). Câbler halo_infinite sur les valeurs ACTUELLES byte-pour-byte (mêmes hosts, mêmes `:443`, même schéma). Injecter le resolver dans `HaloAPIClient` (champ `endpoints EndpointResolver` + `titleSlug`, défaut = `title.DefaultSlug`) et dans `HaloProvider`/discovery/compare/privacy via un `WithEndpoints(...)`. `WithTitleSlug` existant devient le porteur effectif du slug consommé par le resolver. Couche arch-rules : resolver = port dans `games/`, consommé par `platform/halo` + `internal/sync` ; zéro accès DB ; logging `slog.*Context` clé `title`.
- **Parity-gate (ORACLE DOUBLE, les deux obligatoires)** :
  - (a) **Parité Halo golden** : test de caractérisation qui, pour `slug=halo_infinite`, snapshot l'URL complète construite par CHAQUE call-site migré (history, stats, skill, film manifest, economy/battlepass, challenges decks, matches-privacy, servicerecord, discovery asset, nameplate) et assert byte-identique à la chaîne produite par le code const actuel (golden figé). Aucune requête réseau — on compare les URLs assemblées.
  - (b) **Exercice `synthetic_test_title`** : fixture `config/titles/synthetic_test_title/mappings/constants.toml` avec des hosts distincts (ex. `https://stats.example.test`) + un endpoint volontairement ABSENT (ex. `[endpoints].skill` omis). Le test prouve (1) que pour `slug=synthetic_test_title` les URLs routent vers `example.test` (le seam route VRAIMENT, pas cosmétique) et (2) que `HostFor(..., Skill)` retourne `ok=false` → le call-site dégrade proprement (skip + `slog.WarnContext` `capability_absent`, pas de panic, pas de fallback silencieux vers l'host Halo). Réutiliser le pattern d'isolation de `synthetic_title_b/isolation_test.go`.
- **Contract** : PR mince PAR AXE (1 axe = 1 commit) : retirer le `const`/littéral d'un seul host à la fois en basculant son call-site sur `endpoints.HostFor`. Ordre : (1) stats/history, (2) skill, (3) film/UGC, (4) economy/battlepass + challenges, (5) privacy/servicerecord, (6) discovery + gamecms + nameplate. Le littéral inline discovery_client.go:82 bascule en même temps que l'axe stats. Garde lint `no_slug_comparison` active sur chaque PR ; les hosts ne réintroduisent JAMAIS de `slug == "halo_infinite"`. Const supprimés seulement quand 0 référence restante (vérif grep par PR).

**Tests (par couche)** :
- `mappings/` : `LoadEndpointsFromBytes` — TOML valide → `EndpointSet` complet ; section `[endpoints]` absente → erreur explicite ; clé inconnue → erreur ; host vide → erreur ; URL non-https → warn/erreur (parité avec validateField).
- `games/` : `EndpointResolver.HostFor` — slug connu+clé connue → host ; clé absente → `ok=false` ; slug inconnu → fallback DefaultSlug ou `ok=false` (décision explicitée dans le test).
- `platform/halo` + `internal/sync` (caractérisation) : oracle (a) golden URLs halo_infinite byte-identique ; oracle (b) synthetic route vers `example.test` + dégrade sur endpoint absent.
- `archlint` : `no_slug_comparison` reste vert (aucun nouveau gating slug introduit par le câblage host).
- Note CGO/race : packages touchant DuckDB indirects — exécuter avec `-gcflags=all=-d=checkptr=0` si `-race` (réf mémoire driver DuckDB).

**Logging** : `slog.InfoContext` `endpoints_loaded` (clés `title`, `count`, `schema_version`) au boot ; `slog.WarnContext` `endpoint_missing` (clés `title`, `endpoint_key`) quand `HostFor` rend `ok=false` au point de consommation ; `slog.DebugContext` `endpoint_resolved` (clés `title`, `endpoint_key`, `host`) sur chaque résolution. Toujours propager la clé `title`.

**Exit gate** : Halo byte-identique (oracle (a) golden URLs verts sur tous les call-sites) + `synthetic_test_title` route effectivement vers ses propres hosts (oracle (b)) + endpoint capability-absente dégrade proprement (skip + warn, zéro panic, zéro fallback Halo silencieux) + `no_slug_comparison` vert + 0 `const`/littéral host Halo restant hors des fichiers de défaut du resolver.

**Dérive re-vérif** : a changé — (1) seed `discovery_client.go:20 defaultStatsHost` est FAUX : la ligne 20 est `discoveryUGCHost = gamecms-hacs.svc` ; le host stats inline réel pour discovery est un littéral à `discovery_client.go:82`. (2) `economyHost` est à `halo_client_career.go:169-181` (pas seulement :171). (3) `provider.go` economy/challenges sont à `:98-99` et consommés `:301,413,423` (le `:232-281` du seed = mauvais bloc ; WithTitleSlug réel `:230-251`). (4) Hosts manqués par le seed à inclure dans l'axe : `assets/fetcher_gamecms.go:13`, `season_provider.go:19`, `spartan_nameplate_resolver.go:34`, + littéral `discovery_client.go:82`. (5) Le seam reco « TitleDescriptor ou constants.toml » : `TitleDescriptor` (registry.go:44-56) n'a PAS de champ endpoints et `constants.toml` n'existe PAS encore — net-new ; mais l'infra de chargement `config/titles/{slug}/mappings/` + `mappings.Registry.LoadFromConfigDir` existe déjà (registry.go:43-49) et est le bon point d'ancrage. (6) Le titre synthétique de référence enregistré est `synthetic_title_b` (games/synthetic_title_b/) ; la spec crée un fixture `synthetic_test_title` dédié à l'oracle (b) endpoints. aucune autre dérive.

---

### PMT-2 — Acquisition auth par titre  (sévérité: blocker)

**Axes** : MT-02
**Statut couverture actuelle** : **done ✅** (4 legs : XSTS/Spartan/Clearance/SISU/scopes via `AuthDescriptor`). **Leg 5 (store namespacé titre `873637195`) ANNULÉE (2026-06-25)** : les tokens auth sont attachés au compte (xuid), title-agnostic ; ils sont partagés par tous les titres (Halo Infinite ET Halo 5 réutilisent le même pool SpartanToken), donc les ranger par titre dupliquerait inutilement le même `{xuid}.json`. Store global `data/auth/watcher_tokens/` rétabli ; `WatcherTokensDir()` repointé global ; `WatcherTokensDirFor`/`LegacyWatcherTokensDir`/`MigrateWatcherTokens` supprimés — cf. branche `fix/auth-tokens-title-agnostic`. *Gap initial (historique)* : toute la chaîne d'acquisition était hardcodée Halo dans `internal/platform/auth` ; aucun `TitleDescriptor`/TOML ne portait de section auth. (Le `MultiUserTokenStore` reste **délibérément global** — voir annulation ci-dessus.)

**Évidence (⚠ RE-VÉRIFIER avant exécution — pointeurs datés 2026-06-13/14, re-vérifiés)** :
- `internal/platform/auth/halo_exchange.go:30-33` — consts `spartanTokenURL` (settings.svc.halowaypoint.com/spartan-token), `clearanceURL` (`oban/flight-configurations/titles/hi/audiences/RETAIL/active` — le `hi` est Halo-specific), `xstsHaloAudience = "https://prod.xsts.halowaypoint.com/"`. (Le seed pointait 188/203 = corps des appelants `requestSpartanToken`/`requestClearanceToken` ; les consts sont en 30-33.)
- `internal/platform/auth/halo_exchange.go:182` — `requestSpartanToken` body `"Audience": "urn:343:s3:services"` HARDCODÉ Halo (manqué par le seed).
- `internal/platform/auth/halo_exchange.go:56` — `ExchangeAccessToken` passe `xstsHaloAudience` en dur à `requestXSTSToken`.
- `internal/platform/auth/xsts.go:26,62` — `xboxLiveRelyingParty = "http://xboxlive.com"` (RP RTA, partageable cross-titre) ; usage en 62.
- `internal/platform/auth/sisu_provider.go:23-26` — `SISUDefaultAppID = "000000004c20a908"`, `SISUDefaultTitleID = "144209987"` (= Halo Infinite title id) ; `NewSISUProvider()` 76-81 force ces défauts ; `NewSISUProviderWithIDs(appID,titleID)` 84-86 existe DÉJÀ (point d'injection prêt).
- `internal/platform/auth/msal_client.go:42` — `var XboxScopes = []string{"Xboxlive.signin", "Xboxlive.offline_access"}` ; `internal/platform/auth/oauth_refresh.go:33` — `const xboxScopes = "Xboxlive.signin Xboxlive.offline_access"` ; `internal/platform/auth/auth_code.go:64` — réutilise `xboxScopes`. (Ces scopes Xbox sont vraisemblablement constants cross-titre ; à modéliser quand même dans le descripteur pour ne pas re-hardcoder.)
- `internal/api/handlers/auth_xbox_oauth.go:129` — handler `Callback` singulier (1 seul flow, 0 routage titre).
- `internal/platform/auth/multi_user_token_store.go:3,87-99` + `internal/domain/title/registry.go:342` (`WatcherTokensDir()` = `data/auth/watcher_tokens` relatif repoRoot, PAS `data/titles/{slug}/...`) — store global, fichier `{xuid}.json` sans dimension titre ; Spartan/Clearance ne sont d'ailleurs PAS persistés ici (seulement XSTS/RT/MSAL).
- `cmd/server/main.go:77-91` (`buildTokenProvider` → `auth.NewSISUProvider()` défaut), `:589,1452,1471,1616,1653,1795,1895` (≈8 sites `NewMultiUserTokenStore(...WatcherTokensDir())`) — les callers à basculer.
- `internal/domain/title/registry.go:43-66` — `TitleDescriptor{Slug,Name,Provider,Status,Capabilities,XboxTitleID,SteamAppID}` + `HasCapability`. Aucun champ auth ⇒ extension neuve. `config/titles/halo_infinite/constants.toml` a un `[meta]` mais aucune `[auth]`.
- `internal/archlint/no_slug_comparison_test.go:26-35` — garde lint active (allowlist = `api/registry.go`, `api/registry_career.go` ; tout nouveau gate slug hors allowlist casse le test).

**Méthode (expand → parity → contract)** :
- **Expand** : introduire un `title.AuthDescriptor` (value object, package `internal/domain/title` ou sous-package `title/authspec`) portant : `XSTSAudience` (audience XSTS du titre, ex `https://prod.xsts.halowaypoint.com/`), `SpartanAudience` (`urn:343:s3:services`), `ClearanceURL`/`ClearanceTitlePath` (segment `titles/hi`), `SISUAppID`, `SISUTitleID`, `OAuthScopes []string`, `XboxLiveRelyingParty` (RTA, défaut commun). Chargé depuis `config/titles/{slug}/auth.toml` (nouveau, section `[auth]`) avec fallback au défaut Halo câblé aux valeurs ACTUELLES (zéro changement de comportement). Câbler le seam SANS bouger les consts : `ExchangeAccessToken`, `requestSpartanToken`, `requestClearanceToken`, `requestXSTSToken` reçoivent un `AuthDescriptor` (ou un `AuthContext{Descriptor, TitleSlug}`) — nouvelle signature `*WithDescriptor`, les anciennes signatures délèguent au descripteur Halo par défaut. `SISUProvider` : le constructeur de prod passe par `NewSISUProviderWithIDs(desc.SISUAppID, desc.SISUTitleID)`. Persistance : `MultiUserTokenStore` namespacé titre via `PathResolver.WatcherTokensDir(titleSlug)` (nouvelle signature title-aware ; ajouter aussi champs `SpartanToken`/`ClearanceToken`/`TitleSlug` dans `UserTokens` pour persister la sortie Halo par titre). Couche arch-rules : descripteur = `domain/title` (port), résolution = `platform/auth` (platform), routage = `api` ; gating par `HasCapability`/présence d'`AuthDescriptor`, JAMAIS `slug ==`.
- **Parity-gate (ORACLE DOUBLE, non négociable)** :
  - (a) **Parité Halo golden** : test de caractérisation qui sérialise les *requêtes sortantes* (URL + body JSON + headers) produites à travers le seam pour le descripteur `halo_infinite`, et les compare byte-identique à un golden capturé sur le code AVANT seam — couvre les 4 legs (XBL user inchangé, XSTS audience, spartan Audience, clearance URL). Via `httptest.Server` interceptant les POST/GET (msalTokenURL est déjà une var overridable ; rendre les URLs Halo overridables de la même façon pour le test). Zéro diff = parité prouvée.
  - (b) **Exercice `synthetic_test_title`** : fixture `config/titles/synthetic_title_b/auth.toml` avec des valeurs DISTINCTES (audience/app id/title id/clearance path bidon) + test prouvant que (1) le seam route réellement ces valeurs différentes dans les requêtes sortantes (audience ≠ Halo), et (2) capability/descripteur ABSENT (titre sans `[auth]`) → dégradation propre : erreur typée `ErrAuthNotConfigured` (pas de panic, pas de fallback silencieux vers Halo). Sans ce 2e oracle la factorisation reste cosmétique.
- **Contract** : retirer le hardcode par PR MINCE par axe, jamais un méga-swap : PR-1 bascule `requestXSTSToken`+spartan Audience sur le descripteur ; PR-2 clearance URL/title path ; PR-3 SISU app/title id via `buildTokenProvider` → `NewSISUProviderWithIDs(desc...)` ; PR-4 scopes OAuth (`auth_code.go`/`oauth_refresh.go`/`XboxScopes`) ; ~~PR-5 persistance namespacée titre (`WatcherTokensDir(slug)` + champs Spartan/Clearance/TitleSlug) sur les ≈8 callers `cmd/server/main.go`~~ **PR-5 ANNULÉE (2026-06-25)** : tokens account-level partagés inter-titres, store global `data/auth/watcher_tokens/` conservé — cf. branche `fix/auth-tokens-title-agnostic`. Chaque PR garde le défaut Halo identique et laisse le golden (a) vert. Garde lint `no_slug_comparison` active ; aucun nouveau `slug ==` introduit (le routage passe par le descripteur résolu via Registry).

**Tests (par couche)** :
- domain/title : `AuthDescriptor` chargé depuis TOML, défauts Halo == valeurs actuelles, `synthetic_title_b` ≠ Halo, titre sans `[auth]` → erreur typée.
- platform/auth : golden requêtes sortantes Halo byte-identique (a) ; routage descripteur synthétique (b) ; `requestSpartanToken/Clearance/XSTS` honorent l'audience/URL injectée ; `SISUProvider` utilise app/title id du descripteur.
- platform (store) : `WatcherTokensDir(slug)` isole 2 titres (fichiers sous dossiers distincts, pas de collision xuid cross-titre) ; `UserTokens` persiste Spartan/Clearance/TitleSlug ; merge-preserve RT/MSAL inchangé.
- api/handlers : `Callback` résout le descripteur du titre courant (défaut Halo) ; e2e SSO inchangé pour Halo.
- archlint : `TestNoNewSlugComparison` reste vert.

**Logging** : slog `*Context` avec clé `title` sur chaque leg : `auth.exchange.xsts` (`title`, `xsts_audience`), `auth.exchange.spartan` (`title`, `spartan_audience`), `auth.exchange.clearance` (`title`, `clearance_url`), `auth.sisu.init` (`title`, `sisu_app_id`, `sisu_title_id`), `auth.store.persist` (`title`, `xuid`). JAMAIS de token/secret en valeur. Event ids via `logging.WithEvent` (cf. xsts.go:48).

**Exit gate** : golden Halo byte-identique sur les 4 legs (audience XSTS + spartan Audience + clearance URL/title path + scopes inchangés) ; `synthetic_test_title` route un descripteur auth distinct prouvé dans les requêtes sortantes ; titre sans `[auth]` dégrade proprement (`ErrAuthNotConfigured`, zéro fallback Halo silencieux, zéro panic) ; lint `no_slug_comparison` vert. (~~store namespacé titre sans collision xuid~~ **critère ANNULÉ 2026-06-25** : store global title-agnostic conservé, tokens account-level partagés inter-titres — cf. branche `fix/auth-tokens-title-agnostic`.)

**Dérive re-vérif** : a bougé / a changé — (1) Seed pointait `halo_exchange.go:188,203` pour le spartan/clearance endpoint ; en réalité ce sont les CONSTS en `:30-33` (188/203 = corps des appelants `requestSpartanToken`/`requestClearanceToken`). (2) Seed a MANQUÉ l'audience spartan Halo hardcodée `"urn:343:s3:services"` en `halo_exchange.go:182` — ajoutée à l'évidence et à l'oracle (4e leg). (3) `xsts.go` : RP `http://xboxlive.com` const en `:26`, usage en `:62` (seed disait 26,62 — exact). (4) Scopes : seed citait `msal_client.go:41-42` ; la `var XboxScopes` est précisément en `:42`, et il existe DEUX autres définitions du même scope (`oauth_refresh.go:33` const `xboxScopes`, réutilisée par `auth_code.go:64`) — à unifier dans le descripteur, pas seulement msal_client. (5) `NewSISUProviderWithIDs(appID,titleID)` EXISTE déjà (sisu_provider.go:84) — le point d'injection est prêt, seul `buildTokenProvider` (main.go:77-91) force `NewSISUProvider()` (défauts). (6) `WatcherTokensDir()` (registry.go:342) est relatif repoRoot, PAS title-aware ; le store ne persiste PAS Spartan/Clearance (seulement XSTS/RT/MSAL) — la reco « persistance Spartan/clearance namespacée titre » implique d'ÉTENDRE `UserTokens` ET la signature de `WatcherTokensDir`. (7) `TitleDescriptor` n'a aucun champ auth et aucune `[auth]` dans les TOML → extension neuve confirmée. (8) PMT-1 n'existe pas encore comme spec écrite (référencé seulement dans le seed) — la dépendance « lié à PMT-1 » reste à matérialiser.

---

### PMT-3 — Scheduler/auto-sync titleSlug threading  (sévérité: blocker)

**Axes** : MT-11

**Statut couverture actuelle** : gap — le moteur de sync écrit toujours dans les DB `DefaultSlug` quel que soit le titre du joueur ; le slug porté par le profil est résolu pour la seule garde `os.Stat` puis jeté (dette explicitée en commentaire `auto_sync.go:838-841`).

**Évidence (⚠ RE-VÉRIFIER avant exécution — pointeurs datés 2026-06-13, RE-VÉRIFIÉS 2026-06-14)** :
- `internal/sync/engine_options.go:31-49` — `NewSyncEngine(repoRoot, gamertag, xuid string, tokens, provider)` : AUCUN paramètre slug ; lignes 36-44 hardcodent `titlePkg.DefaultSlug` pour `titleSlug` + les 4 chemins DB (player/shared/metadata) + globalXuid. C'est l'unique point de naissance du slug du moteur.
- `internal/sync/engine.go:53` — champ `titleSlug` déjà présent et **profondément consommé** : CSR (`engine_postsync_csr.go:99-100,213`), batch path (`engine_batch_path.go:91`), scoring LUSR (`engine_postsync_scoring.go:94`), asset names (`assetnames_wiring.go:56`), prestige hook (`engine.go:636`), catalog (`engine_postsync.go:291`). → la seule chose manquante est de lui INJECTER la bonne valeur.
- `internal/scheduler/auto_sync.go:68` — `DeltaRunnerFactory func(ctx, gamertag, xuid string) DeltaRunner` : signature sans slug.
- `internal/scheduler/auto_sync.go:255-329` — `BuildEngine(_ ctx, gamertag, xuid)` appelle `NewSyncEngine(...)` :256 sans slug. **UNIQUE source of truth du wiring** : câblée par `main.go:1729` `syncTrigger.WithEngineFactory(autoScheduler.BuildEngine)` → couvre scheduler + watcher + HTTP simultanément.
- `internal/scheduler/auto_sync.go:723` — `runner := factory(ctx, p.Gamertag, p.XUID)` : `p.TitleSlug` (présent sur le `PlayerSummary`) est **droppé** ici.
- `internal/scheduler/auto_sync.go:842-846` + commentaire dette `:838-841` — `slug := p.TitleSlug; if slug=="" { slug = DefaultSlug }` utilisé QUE pour `os.Stat(PlayerDBPath(slug, gamertag))`.
- `internal/scheduler/auto_sync.go:891-899` — `runOnceV2` mappe `PlayerSummary → syncv2.PlayerProfile` sans porter le slug.
- `internal/sync/v2/types.go:10-14` — `PlayerProfile{Gamertag, XUID, PlayerSlug}` : **pas de champ `TitleSlug`**.
- `internal/sync/coordinator.go:72-95,122,232` — `SyncGate.TryClaim(gamertag)` + `Coordinator.TryClaim` clé `normGT(gamertag)` (gamertag-only) → collision cross-titre potentielle (même gamertag, 2 titres).
- `cmd/server/sync_v2_wiring.go:239-306` — `buildSyncEngineFactoryParityComplete` : 2ᵉ appelant de `NewSyncEngine` (:241) sans slug, **doit rester en parité** avec `BuildEngine`.
- `internal/domain/bootstrap.go:35-43` — `PlayerSummary.TitleSlug` existe (rempli par `config/config_players.go:55,62,132,166`).
- Pré-existants réutilisables : `internal/ctxkeys/ctxkeys.go:36-44` (`WithTitleSlug`/`TitleSlug`) ; `internal/archlint/no_slug_comparison_test.go` (garde lint) ; `internal/domain/title/registry.go:79,85` (`DefaultSlug`, `XboxTitleIDFor`).

**Méthode (expand → parity → contract)** :
- **Expand** : introduire le seam slug-aware SANS changer le comportement Halo.
  1. Ajouter un constructeur seam `NewSyncEngineForTitle(repoRoot, titleSlug, gamertag, xuid string, tokens, provider) *SyncEngine` dans `engine_options.go` qui résout TOUS les chemins via `PathResolver` + le titleSlug fourni. Faire de l'ancien `NewSyncEngine(...)` un wrapper mince `→ NewSyncEngineForTitle(repoRoot, titlePkg.DefaultSlug, ...)` (zéro caller cassé). Garde arch-rules : tous les chemins via `PathResolver`, jamais de littéral slug, branchement futur via `HasCapability()` côté consommateurs déjà existants.
  2. Élargir le seam de la factory : `DeltaRunnerFactory func(ctx, titleSlug, gamertag, xuid string) DeltaRunner` ; `BuildEngine(_ ctx, titleSlug, gamertag, xuid string)` appelle `NewSyncEngineForTitle`. À l'appel `:723`, passer `p.TitleSlug` (fallback `DefaultSlug` si vide, même règle que `:842-846` — factoriser en helper `resolveSlug(p)`).
  3. Ajouter `TitleSlug string` à `syncv2.PlayerProfile` (types.go) ; `runOnceV2` le remplit depuis `p.TitleSlug` ; `buildSyncEngineFactoryParityComplete` passe `p.TitleSlug` à `NewSyncEngineForTitle` (parité V1↔V2 maintenue).
  4. Clé `SyncGate` composite : `TryClaim(titleSlug, gamertag)` (et `IsInFlight`, `GateClaimInfo.TitleSlug`) ; clé interne `normGT(gamertag)+"|"+titleSlug`. Le watcher (Submit) prend la même clé composite. Couche : seam dans `internal/sync` (port/service), aucune logique métier titre hors `domain/title`.
  5. Propager `ctxkeys.WithTitleSlug(ctx, slug)` dans `syncPlayer` (avant le `factory(...)`) pour que les logs/sous-modules héritent du slug.
- **Parity-gate** : ORACLE DOUBLE obligatoire.
  - (a) **Parité Halo golden** : test de caractérisation `engine_title_seam_test.go` — pour `titleSlug="halo_infinite"`, `NewSyncEngineForTitle` produit des chemins DB byte-identiques à l'actuel `NewSyncEngine` (player/shared/metadata/globalXuid) ET `e.titleSlug == "halo_infinite"`. Golden sur la sortie des 4 chemins + le slug injecté à travers le seam factory (`BuildEngine` → engine). Comportement runtime inchangé : un cycle scheduler/V2 sur halo_infinite donne la même `RunOnceResult`/`CycleResult`.
  - (b) **Fixture `synthetic_test_title`** (NON optionnel) : profil `PlayerSummary{TitleSlug:"synthetic_test_title"}` → assert que `BuildEngine`/factory route l'engine vers `data/titles/synthetic_test_title/...` (chemins distincts de halo_infinite), que `SyncGate.TryClaim("synthetic_test_title","GT")` et `TryClaim("halo_infinite","GT")` n'entrent PAS en collision (2 claims concomitants accordés), et que `runOnceV2` mappe bien le slug. Dégradation propre : titre sans capability concernée (ex. LUSR absent via `slugHasLUSR`) → l'engine skip la phase sans erreur (réutilise les gates capability déjà en place).
- **Contract** : PR MINCE par axe, jamais de méga-swap.
  - PR-1 : seam `NewSyncEngineForTitle` + wrapper `NewSyncEngine` (no-op runtime).
  - PR-2 : `BuildEngine`/`DeltaRunnerFactory`/`defaultRunnerFactory` threadés + `syncPlayer` passe `p.TitleSlug` ; retirer le commentaire de dette `auto_sync.go:838-841`.
  - PR-3 : `PlayerProfile.TitleSlug` + `runOnceV2` + `buildSyncEngineFactoryParityComplete`.
  - PR-4 : `SyncGate` clé composite (interface + Coordinator + Nop + watcher Submit + handler HTTP).
  - Garde lint `archlint/no_slug_comparison` reste verte (aucun `== "halo_infinite"` introduit ; le seul littéral `DefaultSlug` autorisé reste dans le wrapper rétrocompat + helper fallback).

**Tests (par couche)** :
- **engine (sync)** : `engine_title_seam_test.go` — chemins DB par slug, parité halo_infinite golden, slug injecté consommé par CSR/batch/scoring (assert `e.titleSlug`).
- **scheduler** : factory mock recevant le slug ; `syncPlayer` passe `p.TitleSlug` ; fallback slug vide→DefaultSlug ; synthetic_test_title route distinctement.
- **sync/v2** : `PlayerProfile.TitleSlug` propagé `runOnceV2`→factory ; parité `buildSyncEngineFactoryParityComplete` vs `BuildEngine`.
- **coordinator** : `SyncGate` clé composite — non-collision cross-titre même gamertag ; dédup intra-titre préservée (adapter `gate_test.go`).
- **archlint** : `no_slug_comparison` reste vert.

**Logging** : slog `*Context` clés `title` (= titleSlug) ajoutée sur `auto_sync: démarrage sync delta`, `traitement joueur démarré`, `DB joueur absente`, et les logs gate ; `ctxkeys.WithTitleSlug(ctx, slug)` posé dans `syncPlayer` pour héritage transverse. Clés existantes conservées (`gamertag`, `xuid`, `event`).

**Exit gate** : Halo byte-identique (golden chemins + slug halo_infinite à travers le seam, `RunOnceResult`/`CycleResult` inchangés) + `synthetic_test_title` route vers ses propres DB et claims sans collision cross-titre + capability-absente (ex. LUSR) dégrade proprement (skip sans erreur) + `no_slug_comparison` vert.

**Dérive re-vérif** : a bougé (numéros de ligne) + a changé (substance) — détail complet dans le champ `drift` : signature `NewSyncEngine` sans paramètre slug confirmée ; `BuildEngine` est le point unique câblé watcher+scheduler+HTTP (la Phase 1.9 watcher ne construit pas son propre engine, contrairement à l'implicite du seed) ; 2ᵉ caller `buildSyncEngineFactoryParityComplete` à threader (absent du seed) ; `SyncGate.TryClaim` vit dans `coordinator.go` (pas `auto_sync.go`) et sa clé gamertag-only est un risque cross-titre ; `syncv2.PlayerProfile` n'a pas de champ `TitleSlug` (à ajouter) ; `domain.PlayerSummary.TitleSlug` + `ctxkeys.WithTitleSlug` + lint `no_slug_comparison` déjà présents et réutilisables.

---

### PMT-4 — Settings par titre (overlay)  (sévérité: major)

**Axes** : MT-04 (+ config Discord)
**Statut couverture actuelle** : **partiel** (⚠ re-vérif 2026-06-17) — la primitive + les overlays runtime-critiques sont LIVRÉS : PR-0 `settings.Store.ResolveForTitle(overlayPath)` + `PathResolver.TitleSettingsPath(slug)` (`22649f23e`) ; PR-1 `CSRSeasonIDForTitle` (overlay CSR season, le seul bloquant runtime : sync CSR ne pointe plus une saison Halo inexistante — `afef1195f`) ; PR-2 `notify.LoadNotifyConfigForTitle` (overlay webhook/lang/toggles Discord — `83953208f`) ; PR-3a CSR season UI read-path (`ae7a1627b`). **Reste** (UX, non bloquant runtime) : les settings métier restants lisent encore `store.Load()` global au lieu de `ResolveForTitle(TitleSettingsPath(slug))` — `readCoachProactiveMode` (post_sync_deltas.go), `SessionGapMinutes`/`ShowProgression`/`OutcomeExclude*` (handlers/settings.go), `FriendGamertags` (server.go/registry_pages.go/sync_handler.go). Décision à acter : `FriendGamertags` est-il per-titre ou cross-titre-global (amis = personnes, pas titre) ?

**Évidence (⚠ RE-VÉRIFIER avant exécution — pointeurs datés 2026-06-13, dérive corrigée 2026-06-14)** :
- `internal/config/config.go:96-99` — champ unique `CurrentCSRSeasonID string` (valeur exacte L99) ; loader `loadCSRSeasonID` à `config.go:302-320` (pas 304-320) ; câblage `config.go:190`.
- `internal/config/config.go:113-116` + `internal/config/config_settings.go:73-94` — `PrestigeEnabled` (loader réel dans config_settings, **pas** config.go:304-320 qui est loadCSRSeasonID).
- `internal/platform/settings/store.go:17-85` — `AppSettings` global ; `FriendGamertags` L51, `SessionGap*` L54-56, `OutcomeExcludeBotMatchesFrom{Badges,Records}` L59-60 (**deux** champs), `ShowProgression` L63-64, `CoachProactiveMode` L66-68 ; `Store` n'est clé que par `path` (un seul fichier).
- `internal/api/post_sync_deltas.go:116-130` — `readCoachProactiveMode(reg)` lit le toggle global via `reg.SettingsStore().Load()`.
- `internal/api/handlers/settings.go:410,443` — `h.cfg.LoadPlayers()` sans filtre titre ; `config.PlayerDBPath(h.cfg, "", p.Gamertag)` (slug `""` déjà passé → déjà title-blind, à brancher sur slug courant).
- `internal/notify/discord.go:64-87` (struct `NotifyConfig`), `:91-127` (`LoadNotifyConfig(settingsPath)`) — **un seul** webhook global ; `discord_lang` ne fait que choisir les strings i18n. Construit à 5 sites prod (cmd_notify 27/53, refresh-metadata 413, server/main 1801, friends_orchestrator_service 156, handlers/settings 239).
- `internal/domain/title/registry.go:325` — `AppSettingsPath()` du `PathResolver` est **global**, non namespacé par titre (point d'extension du seam).
- Seam capability DÉJÀ en place : `internal/api/middleware/require_capability.go` (`RequireCapability` → 503 `capability_unavailable`), `TitleDescriptor.HasCapability` (registry.go:59), `config/titles/halo_infinite/mappings/capabilities.toml`. archlint `internal/archlint/no_slug_comparison_test.go` allowliste seulement `api/registry.go` + `api/registry_career.go`.

**Méthode (expand → parity → contract)** :
- **Expand** : introduire un résolveur d'overlay `settings.TitleResolver` (couche `platform/settings`, sans dépendance DuckDB ni slug littéral). Modèle = **base globale + overlay par titre** : nouveau bloc `[titles.<slug>]` dans `app_settings.json` (ou fichier `data/titles/<slug>/settings.json` résolu via un nouveau `PathResolver.TitleSettingsPath(slug)` à ajouter en miroir d'`AppSettingsPath()`). API : `func (s *Store) ResolveForTitle(slug string) (*AppSettings, error)` qui charge le global puis applique l'overlay champ-présent-only (préservation `raw` inchangée). Idem `notify.LoadNotifyConfigForTitle(settingsPath, slug)` → overlay webhook/lang/toggles. Et `cfg.CSRSeasonIDForTitle(slug)` (résolveur, pas champ). **halo_infinite câblé aux valeurs ACTUELLES** : overlay vide ⇒ overlay == global ⇒ zéro changement de comportement, byte-identique. Le titre courant vient de `ctxkeys.TitleSlug(ctx)` (fallback `DefaultSlug`), jamais d'un slug en dur. **Feature-matrix gate la DISPO, l'overlay gate la VALEUR** : si capability absente (ex. `CapRanked` ⇒ pas de CSR season), le résolveur retourne la valeur dégradée (saison vide) sans erreur ; il ne décide JAMAIS de la dispo d'une feature.
- **Parity-gate (oracle double, obligatoire)** :
  - (a) **Parité Halo golden** : test de caractérisation `settings_overlay_parity_test.go` — pour `slug="halo_infinite"` SANS overlay déclaré, `Store.ResolveForTitle` == `Store.Load` (deep-equal sur tous les champs typés + `raw`), `LoadNotifyConfigForTitle == LoadNotifyConfig`, `CSRSeasonIDForTitle("halo_infinite") == CurrentCSRSeasonID`. Golden sur le JSON sérialisé du `NotifyConfig` résolu (webhook/lang/toggles byte-identiques). Gate dans le CI Go existant.
  - (b) **Exercice `synthetic_test_title`** : fixture `app_settings.json` avec un overlay `[titles.synthetic_title_b]` qui surcharge `csr_season_id`, `discord_webhook_url`, `session_gap_minutes`, `friend_gamertags`. Prouver : (i) `ResolveForTitle("synthetic_title_b")` route réellement (valeurs ≠ global), (ii) overlay PARTIEL → champs non-surchargés héritent du global, (iii) **dégradation propre** quand le titre n'a pas `CapRanked` → `CSRSeasonIDForTitle` retourne `""` (sync CSR skippé, pas de crash) même si l'overlay déclare une saison, (iv) webhook overlay pris en compte uniquement si `discord_notifications_enabled` global+overlay résolu. Le titre synthétique n'est PAS optionnel : c'est la seule preuve que le seam route et ne fait pas que recopier le global.
- **Contract** : PR minces par axe (jamais un méga-swap) :
  1. CSR season : basculer les ~12 lecteurs `cfg.CurrentCSRSeasonID` → `cfg.CSRSeasonIDForTitle(ctxkeys.TitleSlug(ctx))` (registry_pages, registry_career, auto_sync, sync_v2_wiring, cmd_sync, sync_handler), supprimer le champ global après le dernier caller.
  2. Discord : basculer les 5 sites `LoadNotifyConfig` → `…ForTitle(slug)`.
  3. Settings métier : `readCoachProactiveMode`/`SessionGapMinutes`/`FriendGamertags`/`ShowProgression`/`OutcomeExclude*` → résolution overlay au point d'usage.
  Garde lint : aucun nouveau `slug == "halo_infinite"` introduit ; le résolveur reste hors allowlist `no_slug_comparison_test.go` (route par capability + map d'overlay, pas par comparaison de slug).

**Tests (par couche)** :
- platform/settings : `ResolveForTitle` (overlay vide=parité, overlay partiel=merge, overlay absent=global), préservation `raw` au Save.
- notify : `LoadNotifyConfigForTitle` parité halo + override synthetic ; webhook overlay gaté par `discord_notifications_enabled`.
- config : `CSRSeasonIDForTitle` parité + dégradation `CapRanked` absente → `""`.
- middleware/integration : requête `ctxkeys.TitleSlug="synthetic_title_b"` → overlay routé bout-en-bout ; capability absente → 503 inchangé (le seam settings ne remplace pas RequireCapability).
- archlint : `TestNoNewSlugComparison` reste vert (aucun ajout à l'allowlist).

**Logging** : `slog.*Context` clé `title` sur toute résolution d'overlay (`op=settings.resolve_overlay`, `title`, `overlay_applied=bool`) ; clé `title` + `capability` quand la dégradation CSR s'active (`op=csr_season.degraded`, `title`, `reason=cap_ranked_absent`) ; webhook résolu logge `title` + `webhook_present=bool` (jamais l'URL).

**Exit gate** : Halo byte-identique (oracle (a) deep-equal global vs overlay-vide sur settings/notify/CSR) + `synthetic_test_title` route correctement (valeurs surchargées distinctes, merge partiel correct) + capability-absente dégrade proprement (CSR season vide, sync skippé, zéro crash/erreur remontée) + `no_slug_comparison_test.go` vert sans nouvelle entrée d'allowlist.

**Dérive re-vérif** : a bougé — voir champ `drift` (line drift sans changement de contenu sur tous les pointeurs) ; a changé — la prémisse seed « webhook 1/langue » est inexacte : il y a UN webhook global, `discord_lang` ne sélectionne que les strings i18n ; le loader `PrestigeEnabled` est dans `config_settings.go:73-94`, pas `config.go:304-320` (ce range = `loadCSRSeasonID`) ; `OutcomeExcludeBotMatches` = deux champs (Badges + Records) ; `handlers/settings.go` passe déjà slug `""` à `PlayerDBPath` donc est title-blind (à brancher, pas à introduire) ; le seam capability runtime (RequireCapability + capabilities.toml) existe déjà et doit être réutilisé tel quel (la feature-matrix gate la dispo, l'overlay la valeur).

---

### PMT-5 — Canonicalisation Outcome  (sévérité: major)

**Axes** : MT-06

**Statut couverture actuelle** : gap — le canonique `canonical.Outcome` (win/loss/tie/dnf) et le `mappings.OutcomeMappingSet` (labels/couleurs) existent, mais AUCUN seam ne traduit le raw int Halo `2/3/1/4` ↔ canonique : le code int 2/3/1/4 est ré-encodé en dur dans ~20 sites SQL/Go + 2 sites front, hors de tout adapter titre.

**Évidence (⚠ RE-VÉRIFIER avant exécution — pointeurs datés 2026-06-14)** :
- `internal/domain/outcomes.go:14-20` — constantes int brutes `OutcomeUnknown=0, OutcomeDraw=1, OutcomeWin=2, OutcomeLoss=3, OutcomeDNF=4` (CONFIRMÉ). C'est l'encodage Halo de référence.
- `internal/games/canonical/enums.go:9-14` — enum canonique cible `Outcome` string `win/loss/tie/dnf` (CONFIRMÉ). Pas de fonction int→canonical.
- `internal/games/mappings/outcomes.go:10-14` — `OutcomeMapping{Key, Labels, ColorToken}` : PAS de champ `code int`. `OutcomeMappingSet.Get(key)` indexe par clé string uniquement. **C'est le trou à combler.**
- `config/titles/halo_infinite/mappings/outcomes.toml:15-29` — sections `[outcomes.win|loss|tie|dnf]` sans champ numérique. À étendre avec un `raw_code` par outcome.
- `internal/games/adapter.go:105-112` — `TitleSemanticAdapter.Outcomes() *mappings.OutcomeMappingSet` (CONFIRMÉ : déjà exposé, peut être nil → callers dégradent).
- `internal/games/halo_infinite/citations_custom.go:76,92,107,121` — `if ctx.Outcome != 2` (CONFIRMÉ, 4 occurrences ; `domain.CitationContext.Outcome int`, cf. `domain/citations.go:168`).
- `internal/analysis/sql_fragments.go:36,44-46` — `const SQLIsWin = "outcome = 2"` + `SQLWinRateExpr` (existent mais quasi inutilisés ; les repos inlinent `outcome = 2`).
- SQL en dur `outcome=2/3/1` à router : `platform/duckdb/compare_repo.go:38,191,193` ; `explorer_repo.go:107-109` ; `match_history_repo.go:256` ; `queries_squad.go:34,256,341` ; `queries_career_encounters.go:30-33` ; `queries_match_detail.go:120-123` ; `squad_repo_mapstats.go:56` ; `career_repo_top_matches.go:117` ; `api/post_sync_deltas_snapshot.go:218` ; `api/post_sync_progression_queries.go:300`.
- Comparaisons int Go à router : `analysis/patterns/context.go:75,145` ; `behavioral.go:37` ; `service/squad_service.go:211-213` ; `career_service_encounters.go:274` ; `match_history_service_enrich.go:49` ; `sync/skill_v2_quit_penalty.go:51` ; `sync/assists_model.go:70` (`outcome != 4`).
- Front : `apps/web/src/lib/outcome-color.ts:18-22` (`OUTCOME_KEY {2:win,1:draw,3:loss}`) + `features/media/fallback.i18n.ts:13-18` (`{2:Victoire...}`). Seam déjà présent : `lib/i18n/fieldMappings.ts:190-201` (`useOutcomeLabel(key)`), `/field-mappings` expose les outcomes par clé.
- `internal/archlint/no_slug_comparison_test.go:33-35` — regex de garde (gating slug interdit). Modèle de ratchet réutilisable pour un nouveau test "no raw outcome literal".

**Méthode (expand → parity → contract)** :
- **Expand** : (1) Ajouter `raw_code int` à `outcomeEntryTOML`/`OutcomeMapping` (`mappings/outcomes.go` + `loader_outcomes.go`), peupler `outcomes.toml` (`win=2, loss=3, tie=1, dnf=4`) ; validation : codes uniques + ⊂{1,2,3,4}. (2) Sur `OutcomeMappingSet` ajouter `Canonical(rawCode int) (canonical.Outcome, bool)` et `RawCode(canonical.Outcome) (int, bool)` (table double-sens construite au load). (3) Exposer 2 SQL builders title-aware dans `OutcomeMappingSet` : `SQLIsWinExpr(col string) string` → `col = 2` pour HI, et `SQLOutcomeCase(col)` ; le titre fournit les littéraux, le repo ne les connaît plus. Couche arch-rules : la traduction vit dans `internal/games/mappings` (domaine/sémantique), consommée via `Resolver.Semantic(slug).Outcomes()` ; AUCUN nouveau `slug ==`. (4) Front : aucun nouveau seam — réutiliser `useOutcomeLabel`/`/field-mappings`. **halo_infinite câblé aux valeurs ACTUELLES → zéro changement de comportement.**
- **Parity-gate** : ORACLE DOUBLE. (a) **Golden Halo byte-identique** : test de caractérisation `outcomes_parity_golden_test.go` qui, pour chaque site basculé, asserte que `OutcomeMappingSet(HI).SQLIsWinExpr("outcome")` == `"outcome = 2"` littéral, que `Canonical(2)==win, 3==loss, 1==tie, 4==dnf`, et un golden sur la sortie d'au moins 1 repo (`explorer_repo` wins/losses/draws + 1 citation `computeWinsCTF`) AVANT/APRÈS = identique sur dataset figé. (b) **Fixture synthetic_title_b** : fournir un `synth_outcomes.toml` qui INVERSE les codes (ex. `win=7, loss=9`) et un test prouvant que `OutcomeMappingSet(synth).SQLIsWinExpr("outcome")=="outcome = 7"` ≠ HI → le seam route VRAIMENT différemment ; + cas `Outcomes()==nil` (titre sans outcomes.toml) → le caller dégrade proprement (fallback `domain.OutcomeWin` ou skip, jamais panic). Le titre synthétique n'est PAS optionnel.
- **Contract** : retirer le hardcode par PR MINCE par axe, jamais méga-swap. Ordre : PR-a (les 2 bloquants d'abord) = sites **ingestion/sync** (`assists_model.go`, `skill_v2_quit_penalty.go`, `post_sync_*`) + le builder SQL ; PR-b SQL repos lecture (`compare/explorer/match_history/squad/career_encounters/match_detail/mapstats`) ; PR-c citations (`citations_custom.go` ×4 → `Outcomes().Canonical(ctx.Outcome)==OutcomeWin`) + `analysis/patterns` + `sql_fragments.go` (déprécier `SQLIsWin` const au profit du builder) ; PR-d front (supprimer `OUTCOME_KEY`/`OUTCOME_LABELS_FALLBACK_FR` int-maps, dériver via `/field-mappings`). Garde : étendre l'archlint avec un test ratchet "no raw outcome literal" (regex `outcome\s*=\s*[1-4]` / `Outcome\s*[!=]=\s*[1-4]` hors `internal/games/mappings` + allowlist des sites non-encore-migrés, qui rétrécit à chaque PR).

**Tests (par couche)** :
- `mappings/` : round-trip `Canonical(RawCode(o))==o` ∀ o ; rejet code dupliqué / hors {1..4} ; `SQLIsWinExpr` exact.
- `games/halo_infinite/` : `adapter_semantic_test` — `Outcomes().Canonical(2..4)` == valeurs Halo ; SQL == littéraux actuels.
- `games/synthetic_title_b/isolation_test.go` : codes inversés routent différemment ; `Outcomes()==nil` dégrade.
- `platform/duckdb/` : golden wins/losses/draws sur dataset figé, avant/après identique (table-driven).
- `analysis/` + `service/` : `computeWinsCTF`/`squad`/`encounters` inchangés sur fixture.
- `archlint/` : le nouveau test échoue si un littéral outcome réapparaît hors allowlist.
- Front : test que `outcomeKey`/labels dérivent du DTO `/field-mappings` (mock), plus de table int en dur.

**Logging** : slog `*Context` clé `title` ; `mappings_loaded kind=outcomes title_slug outcomes_count schema_version` (existant `registry.go:95-100`) ; ajouter au load un debug `outcome_codes_indexed title_slug count` ; côté caller en cas de dégradation `outcome_mapping_absent title_slug fallback_used=true`.

**Exit gate** : Halo byte-identique (golden SQL/citations/repos avant==après) + synthetic_title_b route ses codes inversés différemment (preuve non-cosmétique) + `Outcomes()==nil` dégrade proprement (pas de panic, fallback déterministe) + archlint "no raw outcome literal" vert avec allowlist en décroissance + `no_slug_comparison` toujours vert.

**Dérive re-vérif** : voir champ `drift` — 5 corrections (double `outcomes.go` ; `world_stats.go:28` = champ struct pas SQL ; empreinte ~3× plus large que le seed ; `OutcomeMappingSet` sans mapping int↔canonical = le vrai seam manquant ; seam front `useOutcomeLabel` déjà présent, seule la table int est en dérive).

---

### PMT-6 — Achievements par titre  (sévérité: major)

**Axes** : MT-08

**Statut couverture actuelle** : **done ✅** (PR1 lecture `GetAchievementDefinitions(ctx, slug)` filtre `title_id=?` + PR2 `XboxTitleIDFor` registry-driven, `e7f06fe71`). **Reste** : PR3 flag CLI `--title` + propagation engine — différé (write-side, exerçable seulement avec un 2e titre). *Gap initial* : lecture filtrait en dur `title_id='halo_infinite'` en ignorant le slug du service ; `XboxTitleIDFor` dupliquait `TitleDescriptor.XboxTitleID`.

**Évidence (⚠ RE-VÉRIFIER avant exécution — pointeurs datés 2026-06-13, re-vérifiés 2026-06-14)** :
- `internal/platform/duckdb/metadata_achievements_repo.go:29` — `WHERE title_id = 'halo_infinite'` en dur dans `GetAchievementDefinitions` ; le slug n'est ni paramètre ni lu. **C'est LE gap lecture.**
- `internal/port/achievements.go:22-24` — `MetadataAchievementsRepository.GetAchievementDefinitions(ctx)` sans `titleSlug` → l'interface à étendre (le seam).
- `internal/service/achievements_service.go:26,41,60,79-82,88,117` — service déjà title-aware (champ `titleSlug`, `WithTitleSlug`, fallback `defaultAchievementsTitleSlug`), slug PASSÉ à la catégorie (l.88) mais PAS au metaRepo (l.60).
- `internal/api/registry_career.go:177` — `NewAchievementsService(repo, metaRepo).WithTitleSlug(pdb.TitleSlug)` : le slug est déjà disponible au point de câblage.
- `internal/domain/achievement_categories.go:27-29,40-49` — map keyée par slug + `AchievementCategoryFor` dégrade DÉJÀ (`("",false)` si titre absent). À NE PAS toucher, juste prouver via synthetic.
- `internal/domain/title/registry.go:85-91` — `XboxTitleIDFor(slug)` = switch hardcodé `DefaultSlug→"2043073184"`, alors que `TitleDescriptor.XboxTitleID` (l.54, peuplé l.117) porte déjà la valeur → **doublon de vérité à supprimer**.
- `cmd/levelup/cmd_sync_achievements.go:52,64,111,120,139` — `titlePkg.DefaultSlug` hardcodé ×5, aucun flag `--title`.
- `internal/sync/engine_options.go:31-49` — `NewSyncEngine` force `titleSlug: DefaultSlug` (pas de param) → bloque le passage du titre depuis le CLI.
- `internal/sync/achievements.go:146,181,188` + `internal/sync/xbox_client.go:164` + `internal/sync/engine_postsync_csr.go:99` — write side DÉJÀ title-aware (sert d'oracle de parité : ne pas régresser).
- `internal/api/server.go:905` + `internal/archlint/no_slug_comparison_test.go:34-35` — gate `CapAchievements` posée + lint garde active.

**Méthode (expand → parity → contract)** :
- **Expand** : (1) Étendre le port `MetadataAchievementsRepository.GetAchievementDefinitions(ctx, titleSlug string)` (couche `port`) ; l'impl `duckdb` paramètre le `WHERE title_id = ?`. (2) Le service passe `s.titleSlug` (avec fallback `DefaultSlug`) à cet appel — couche `service`, AUCUN slug littéral, lecture du champ existant. (3) `NewSyncEngine(..., titleSlug string)` (nouveau param en queue, défaut conservé via les callers) OU helper `WithTitleSlug(slug)` sur l'engine, pour propager le titre au CLI ; couche `sync`. (4) `XboxTitleIDFor` devient registry-driven : `func (r *Registry) XboxTitleIDFor(slug) string { d:=r.Get(slug); return d.XboxTitleID }` (ou variante via Registry injecté), supprime le switch. Câblage à HI = valeurs ACTUELLES, zéro changement de comportement.
- **Parity-gate** : oracle DOUBLE. **(a) parité Halo (golden)** : test de caractérisation sur `GetAchievementsPage` pour `halo_infinite` — la réponse JSON (summary + entries triées + catégories) doit être **byte-identique** à la sortie actuelle ; idem golden sur la requête SQL effective du repo (`title_id='halo_infinite'`) et sur `XboxTitleIDFor("halo_infinite")=="2043073184"` (inchangé). **(b) fixture `synthetic_test_title`** : enregistrer un titre synthétique SANS `CapAchievements` ET un avec, dans `metadata.duckdb` insérer des defs `title_id='synthetic_test_title'` → prouver que (i) le filtre route vraiment (defs du titre synthétique retournées, defs HI exclues, et inversement), (ii) `XboxTitleIDFor("synthetic_test_title")` renvoie l'XboxTitleID du descripteur synthétique (≠ HI), (iii) catégorie absente du registre → `("",false)` → entries avec `Category==""` → preuve de dégradation propre. Le titre synthétique est obligatoire (seule preuve non-cosmétique).
- **Contract** : PR mince par axe — (PR1) port+repo paramétrés + service qui transmet le slug, callers HI inchangés ; (PR2) `XboxTitleIDFor` registry-driven, suppression du switch ; (PR3) flag `--title` au CLI + propagation engine, défaut `halo_infinite`. Aucun méga-swap. Garde : `no_slug_comparison_test.go` reste vert (ne JAMAIS réintroduire `slug == "halo_infinite"` ; le filtre passe par paramètre/`HasCapability`, pas par comparaison littérale).

**Tests (par couche)** :
- `port`/`duckdb` : `GetAchievementDefinitions(ctx, slug)` filtre correctement (table multi-titres → seul le slug demandé revient ; slug inconnu → slice vide).
- `service` : golden parité HI (réponse identique) ; titleSlug vide → fallback `DefaultSlug` ; titleSlug synthétique → defs synthétiques + `Category==""`.
- `domain` : `XboxTitleIDFor` registry-driven (HI="2043073184", synthetic=≠, slug inconnu="").
- `sync` (intégration, existant `achievements_integration_test.go`) : ne pas régresser (write `title_id` depuis slug, filtre API `titleId`).
- `archlint` : `no_slug_comparison_test.go` vert sur les fichiers modifiés.
- `cmd` : `--title` parse + propagation au PathResolver/engine (dry-run).

**Logging** : `slog.*Context` avec clé `title` (canonique arch-rules) en plus de l'existant `titleSlug`/`title_id` — uniformiser sur `title=<slug>` dans : `achievements service: page built` (service l.101), `GetAchievementDefinitions` (ajouter un Debug avec `title` + `count`), `XboxTitleIDFor` miss (Warn `title` `reason=unknown_slug` si "" retourné côté caller), CLI sync (`title` dans les lignes OK/FAIL/SKIP).

**Exit gate** : Halo byte-identique (golden page + golden SQL + `XboxTitleIDFor("halo_infinite")` inchangés) + `synthetic_test_title` route réellement (defs filtrées par titre, XboxTitleID distinct) + capability/catégorie absente dégrade proprement (`Category==""`, front masque le filtre, slice vide sans erreur) + `no_slug_comparison` lint vert.

**Dérive re-vérif** : a bougé / a changé — détail complet dans le champ `drift`. Résumé : le write-side et la garde `CapAchievements` sont DÉJÀ title-aware (non vus par le seed) ; le seul vrai trou est la **lecture** (`metadata_achievements_repo.go:29`) + l'absence de paramètre `titleSlug` sur le port (`achievements.go:22-24`) + CLI sans `--title` + `XboxTitleIDFor` (registry.go:85-91) qui duplique `TitleDescriptor.XboxTitleID`. La catégorisation (`achievement_categories.go`) dégrade déjà — ne pas la modifier, seulement la prouver via synthetic.

---

### PMT-7 — World-stats / leaderboard par titre  (sévérité: major)

**Axes** : MT-03

**Statut couverture actuelle** : gap — la colonne `title_slug` existe partout dans le schéma + les types, mais aucun chemin de bout-en-bout ne la THREADE : le handler/service portent déjà `req.TitleSlug` mais le repo le jette et re-câble `defaultLeaderboardTitleSlug = "halo_infinite"` ; analyse, enricher et 3 CLIs sont mono-titre en dur.

**Évidence (⚠ RE-VÉRIFIER avant exécution — pointeurs datés 2026-06-14)** :
- `internal/platform/duckdb/world_player_stats_repo.go:209` — `const defaultLeaderboardTitleSlug = "halo_infinite"` ; lu en `:175` (bind du WHERE de `worldPlayerStatsQuery`), `:195` (force `s.TitleSlug`), et `:55` (fallback du INSERT `InsertPlayerSeasonStats`). `GetWorldPlayerSeasonStats(:157)` / `queryWorldPlayerStats(:174)` n'acceptent PAS de slug en paramètre. CONFIRMÉ.
- `internal/platform/duckdb/leaderboard_world_repo.go:253-254` — `resolvePlaylistNamesFromCatalog` binde `defaultLeaderboardTitleSlug` dans `WHERE title_slug = ?` (catalogue noms). `GetCSRWorldLeaderboard(:32)`, `enrichWorldEntries(:105)`, `GetWorldLeaderboardCatalog(:159)` ne prennent PAS de slug. (Seed disait `:175,254` — `:175` est un simple `scanIDColumn` sans slug ; la vraie liaison est `:253-254`.) CONFIRMÉ avec dérive de ligne.
- `internal/analysis/world_stats.go:163` — `AccumulateWorldStats` fixe `TitleSlug: "halo_infinite"` en dur dans le bucket. Aucune autre passe ne ré-attribue le slug → l'enricher hérite de ce défaut. CONFIRMÉ.
- `internal/service/world_stats_enricher.go` — n'écrit JAMAIS `TitleSlug` (le seed `:21-28` est `RankedPlaylistSet`, hors-sujet slug). Le slug vient uniquement de `analysis` ci-dessus. Pointeur seed à corriger.
- `internal/migration/steps_shared_world_player_season_stats.go:37` — `title_slug VARCHAR NOT NULL DEFAULT 'halo_infinite'` (+ vue `_latest` partitionne par `title_slug`, `:66`). ⚠ Le chemin du seed (`internal/platform/duckdb/migration/...`) est FAUX — le fichier vit sous `internal/migration/`. CONFIRMÉ après correction de chemin.
- `cmd/backfill-world-player-stats/main.go:173` — `SharedDBPath(titlepkg.DefaultSlug)` ; `:476` `MetadataDBPath(titlepkg.DefaultSlug)` ; pas de flag `-title`. (Seed `:126,286` = `NormalizeSeasonID`, sans rapport slug ; lignes dérivées.) CONFIRMÉ avec dérive.
- `cmd/snapshot-world-leaderboard/main.go:43` — défaut `-shared-db` LITTÉRAL `"data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb"` (court-circuite PathResolver). NON listé au seed — NOUVEAU.
- `cmd/probe-world-stats/main.go:613` — `WHERE title_id = 'halo_infinite'` sur `csr_season_calendars` (colonne `title_id`, pas `title_slug`). CONFIRMÉ.
- `internal/worldenrich/wiring.go:32` — `PlayerDBPath(title.DefaultSlug, gamertag)` en dur (résolveur title-aware mais slug figé). NOUVEAU.
- `internal/api/handlers/leaderboard.go:51` + `internal/service/leaderboard_service.go:58` — `req.TitleSlug` lu du query param et RÉ-ÉMIS dans la réponse, mais JAMAIS passé à `GetCSRWorldLeaderboard/GetStatLeaderboard/GetWorldLeaderboardCatalog` (port `repository_data.go:371,375,378` : signatures sans slug). Le fil est coupé à la frontière service→repo. NOUVEAU (axe central de la phase).
- `internal/domain/title/registry.go:79` `DefaultSlug`, `:59` `HasCapability`, `:206` `SharedDBPath(slug)` — seam title-aware disponible ; `CapRanked` existe (`:35`) mais aucune capability « world leaderboard » dédiée.

**Méthode (expand → parity → contract)** :
- **Expand** : introduire le slug comme paramètre EXPLICITE, halo_infinite câblé à la valeur actuelle, zéro changement de comportement.
  1. Étendre le port (`repository_data.go`) : `GetCSRWorldLeaderboard(ctx, titleSlug, season, playlist, limit)`, `GetStatLeaderboard(ctx, titleSlug, category, playlist, season, limit)`, `GetWorldLeaderboardCatalog(ctx, titleSlug)` + propager dans le noop. Threader `titleSlug` jusqu'à `queryWorldPlayerStats` / `resolvePlaylistNamesFromCatalog` / `loadPrevSeasonRanks` (ce dernier ajoute `WHERE title_slug=?` sur la vue _latest une fois la colonne disponible côté CSR — sinon laisser inchangé tant que `world_csr_leaderboard*` n'a pas la colonne, voir Dérive).
  2. `analysis.AccumulateWorldStats(titleSlug, gamertag, stats)` reçoit le slug en argument (plus de littéral `:163`). `ExtractWorldPlayersFromMatch` reste pur.
  3. `WorldStatsEnricher.EnrichSeason` + `WorldStatsAggregator` portent le slug et le posent sur chaque `WorldPlayerSeasonStats`. Le cron (`world_leaderboard_cron.go`) résout le slug via `provider`/config (un seul titre actif → `title.DefaultSlug`), pas de littéral.
  4. Couche service : `LeaderboardService.GetPage/GetCatalog` valident `req.TitleSlug` (vide → `title.DefaultSlug`) via `PathResolver.ValidateTitle`, puis le passent au repo. Le handler le lit déjà (`:51`).
  5. CLIs : ajouter `-title` (défaut `title.DefaultSlug`) à backfill/snapshot/probe ; dériver TOUS les chemins via `PathResolver.SharedDBPath(title)/MetadataDBPath(title)` (supprime le littéral `snapshot:43` et le `DefaultSlug` figé de `worldenrich/wiring.go:32`). Le slug devient une colonne de l'`INSERT` (plus de fallback `:55`).
  Couche arch-rules : slug = donnée threadée (port→service→repo→analysis), JAMAIS un `if slug == "halo_infinite"`. Le gating « ce titre publie-t-il un leaderboard mondial ? » passe par une capability NOUVELLE `CapWorldLeaderboard` (registre `registry.go`) consommée via `HasCapability` au point d'entrée service (titre sans la cap → réponse vide + 200, pas 500).
- **Parity-gate (ORACLE DOUBLE, non négociable)** :
  - (a) **Parité Halo golden** : test de caractérisation sur `GetCSRWorldLeaderboard`/`GetWorldPlayerSeasonStats`/`GetWorldLeaderboardCatalog` à travers le nouveau seam, slug résolu = `halo_infinite`, comparé byte-identique à un golden capturé AVANT refactor (mêmes entrées, mêmes ratios dérivés win_rate/kda/kills_per_min, mêmes trends, même delta rang). Plus test unitaire `AccumulateWorldStats` (déjà existant, `world_stats_test.go:115` attend `TitleSlug=="halo_infinite"`) re-paramétré pour prouver l'identité sur halo_infinite.
  - (b) **Exercice `synthetic_test_title`** (PAS optionnel) : fixture qui (1) insère `world_player_season_stats` + `world_csr_leaderboard_snapshots` sous `title_slug = "synthetic_title_b"`, prouve que `GetCSRWorldLeaderboard(synthetic_title_b, …)` route SUR ces lignes et NE FUIT PAS vers halo_infinite (isolation, calquée sur `resolver_test.go:145 TestStaticResolver_Catalog_Isolation`) ; (2) un titre SANS `CapWorldLeaderboard` (descripteur synthétique sans la cap) → service renvoie catalogue/entries vides + dégrade proprement (aucun panic, aucun fallback halo_infinite), prouvant que la factorisation est réelle et pas cosmétique.
- **Contract** : retirer le hardcode par PR MINCE PAR AXE (jamais un méga-swap) :
  - PR-1 : port + repo (`world_player_stats_repo.go` / `leaderboard_world_repo.go`) threadés ; suppression du `const defaultLeaderboardTitleSlug` une fois aucun caller restant.
  - PR-2 : `analysis` + `enricher` + cron threadés ; suppression du littéral `:163`.
  - PR-3 : service/handler câblent `req.TitleSlug` au repo + gating `CapWorldLeaderboard`.
  - PR-4 : 3 CLIs (`-title` + PathResolver) + `worldenrich/wiring.go`.
  - PR-5 : retirer `DEFAULT 'halo_infinite'` du schéma (`steps_shared_world_player_season_stats.go:37`) une fois TOUS les writers passant le slug explicitement (migration de durcissement, INSERT sans défaut). Garde : `archlint/no_slug_comparison_test.go` reste vert (aucun nouveau `slug == "halo_infinite"` introduit ; le gating se fait par cap).

**Tests (par couche)** :
- analysis : `AccumulateWorldStats` paramétré slug (halo_infinite = golden ; synthetic_title_b = bucket distinct).
- repo (duckdb, build cgo) : golden parité halo_infinite + isolation synthetic_title_b (2 jeux de lignes sous 2 slugs, requêtes croisées ne fuient pas) + `loadPrevSeasonRanks` filtré par slug.
- service : `GetPage` vide→DefaultSlug ; `CapWorldLeaderboard` absente → entries/catalog vides + 200.
- handler : `title_slug` query param propagé au repo (table-driven).
- cron : slug résolu = DefaultSlug, pas de littéral ; enrich persiste avec le bon slug.
- archlint : `TestNoNewSlugComparison` reste vert sans nouvelle allowlist.
- CLIs : test de résolution de chemin (`-title` → `PathResolver.SharedDBPath`), pas de littéral `snapshot:43`.

**Logging** : slog `*Context` avec clé `title` (slug résolu) sur chaque entrée — repo (`logModuleLeaderboard`), cron (`ModuleLeaderboard`), CLIs (`InstallCLI`). Clés existantes conservées : `season`, `playlist`, `entries`, `scraped`, `inserted`. Ajouter `title` aux logs `world_leaderboard_cron: cycle terminé` / `enrichissement terminé` et aux warns d'enrichissement.

**Exit gate** : Halo byte-identique (golden parité sur les 3 lectures + ratios dérivés) ✓ + `synthetic_title_b` route sur ses propres lignes sans fuite ✓ + titre sans `CapWorldLeaderboard` dégrade proprement (vide + 200, zéro panic, zéro fallback halo_infinite) ✓ + `archlint/no_slug_comparison_test.go` vert ✓ + `DEFAULT 'halo_infinite'` retiré du schéma (PR-5) ✓.

**Dérive re-vérif** : a bougé : (1) migration sous `internal/migration/steps_shared_world_player_season_stats.go:37`, PAS `internal/platform/duckdb/migration/...` (chemin seed faux). (2) Liaison slug du catalogue à `leaderboard_world_repo.go:253-254`, pas `:175` (qui est un `scanIDColumn` sans slug). (3) Backfill : lignes slug réelles `:173` (`SharedDBPath`) + `:476` (`MetadataDBPath`) ; les `:126,286` du seed sont `NormalizeSeasonID` (hors-sujet). a changé / nouveau : (4) `snapshot-world-leaderboard/main.go:43` a un défaut de chemin LITTÉRAL court-circuitant PathResolver (absent du seed). (5) `req.TitleSlug` existe déjà dans le handler/service (`leaderboard.go:51`, `leaderboard_service.go:58`) mais n'est PAS threadé au repo (fil coupé service→repo) — c'est l'axe central, non capturé par le seed. (6) `worldenrich/wiring.go:32` fige `DefaultSlug` pour le player DB. (7) probe utilise la colonne `title_id` (pas `title_slug`) sur `csr_season_calendars`. (8) Aucune capability « world leaderboard » n'existe (`CapRanked` proche mais distincte) → à créer pour le gating par capability exigé par arch-rules. (9) ⚠ Les tables CSR mondiales (`world_csr_leaderboard_snapshots`/`_latest`) n'ont PAS de colonne `title_slug` (seul `world_player_season_stats` l'a) — la migration `world_csr_leaderboard*` doit d'abord ajouter la colonne avant de threader le slug dans `GetCSRWorldLeaderboard`/`loadPrevSeasonRanks` ; à vérifier dans `steps_world_csr_leaderboard*.go` au moment de l'exécution.

---

### PMT-8 — Cycle de vie du titre (Status enforcement)  (sévérité: major)

**Axes** : MT-22
**Statut couverture actuelle** : partial — `TitleDescriptor.Status` est défini, persisté dans le registre et SÉRIALISÉ vers le front (`BuildAvailableTitles` → `TitleSummary.Status` → bootstrap/session_context), mais jamais ENFORCÉ : aucun gate middleware (un titre `coming_soon`/`archived` répond comme `active`), aucun filtrage de liste, et le front ne lit pas le signal.

**Évidence (⚠ RE-VÉRIFIER avant exécution — pointeurs datés 2026-06-13, RE-VÉRIFIÉS 2026-06-14)** :
- `internal/domain/title/registry.go:18-25` — `type Status` + `StatusActive`/`StatusComingSoon`/`StatusArchived`. CONFIRMÉ.
- `internal/domain/title/registry.go:49,110` — champ `Status` sur `TitleDescriptor` ; HI enregistré `StatusActive`. CONFIRMÉ.
- `internal/api/middleware/title.go:30-47` — `resolveTitleSlug` route via header/session/fallback en n'appelant QUE `registry.Exists()` ; jamais `.Status`. CONFIRMÉ (le seam d'injection).
- `internal/api/middleware/require_capability.go:39-67` — template du gate (503 `capability_unavailable`, lecture `ctxkeys.TitleSlug` + `registry.Get`) ; AUCUN équivalent Status. (seed disait « l.46 » → en réalité test l.47.) ⚠ Note : log clé `titleSlug` l.48-52 → à corriger en `title`.
- `internal/service/bootstrap_service.go:411-432` — `BuildAvailableTitles()` LIT `t.Status` (l.426) → `domain.TitleSummary.Status`. ⚠ Le seed le ratait : Status est lu pour sérialisation, pas filtré.
- `internal/domain/bootstrap.go:51-59` + `internal/api/gen/types.gen.go:371-379,1650-1661` — DTO `TitleSummary.Status` + enum OpenAPI `TitleSummaryStatus` déjà générés.
- `apps/web/src/stores/appShellStore.ts:122` — `availableTitles` (avec `.status`) stocké mais aucun consommateur downstream ne le lit. CONFIRMÉ.
- `internal/archlint/no_slug_comparison_test.go:26-35` — garde lint pour la phase contract.
- `cmd/levelup/cmd_title.go:214` — string d'aide (générateur de snippet), pas une lecture runtime. CONFIRMÉ.

**Méthode (expand → parity → contract)** :
- **Expand** : introduire `middleware.RequireActiveTitle(registry *title.Registry) func(http.Handler) http.Handler` dans `internal/api/middleware/` (jumeau de `RequireCapability`) : lit `ctxkeys.TitleSlug`, `registry.Get(slug)`, et si `desc == nil || desc.Status != title.StatusActive` répond `503` body `{code:"title_unavailable", title_slug, status, message, retryable:false}` (`coming_soon` → message « bientôt disponible » ; `archived` → « plus maintenu »). Ajouter une méthode pure `(*Registry).Active() []*TitleDescriptor` ET une méthode prédicat `(*TitleDescriptor).IsActive() bool` dans le domaine title (couche domain, zéro IO). Brancher le middleware HI aux valeurs ACTUELLES = no-op de comportement (HI est `StatusActive`, tout passe). NE PAS encore filtrer `BuildAvailableTitles` (expand = seam, zéro changement observable). Gating sur `HasCapability`/Status via le registre, jamais sur un slug littéral (arch-rules).
- **Parity-gate** : ORACLE DOUBLE. (a) **Parité Halo golden** : test de caractérisation montant le routeur réel avec le `NewRegistry()` de prod ; assert que toutes les routes gatées par `RequireActiveTitle` répondent byte-identique (status code + body) AVANT/APRÈS le câblage pour HI (HI=active → 0 régression) ; golden sur le JSON `available_titles` de `/bootstrap` inchangé. (b) **Synthetic** : enregistrer dans un `*title.Registry` de test (pattern `matcher_test.go:55-63` / `multititle_test.go:41-46`) un descripteur `Slug:"synthetic_b", Status:StatusComingSoon` et un autre `Status:StatusArchived` ; prouver que `RequireActiveTitle` renvoie `503 title_unavailable` avec le bon `status` pour chacun, que `Registry.Active()` les exclut, et qu'un descripteur `StatusActive` passe — DÉGRADATION PROPRE = body machine-readable, pas un panic ni un 500. Le titre synthétique n'est pas optionnel : c'est la seule preuve que le seam route vraiment sur Status.
- **Contract** : PR mince par axe — (1) appliquer `RequireActiveTitle` aux sous-arbres de routes data/sync title-scoped ; (2) filtrer `BuildAvailableTitles` → ne lister que `Active()` PLUS les `coming_soon` annotés (garder le status pour que le front affiche « bientôt »), exclure `archived` ; (3) front : `appShellStore` masque/désactive les entrées non-`active` dans le futur title-switcher (signal déjà présent). Retirer toute tentation de `slug == "..."` ; le test `no_slug_comparison_test.go` reste la garde. Jamais un méga-swap.

**Tests (par couche)** :
- domain : `registry_test.go` — `Active()` exclut coming_soon/archived ; `IsActive()` par statut.
- middleware : `require_active_title_test.go` — active=passe, coming_soon/archived=503 body+code+status, titre inconnu=503, log clé `title`.
- service : `bootstrap_service_test.go` — `BuildAvailableTitles` filtre archived, conserve status sur coming_soon ; golden JSON parité HI.
- handler : caractérisation route gatée (parité Halo) + cas synthetic_b (route différemment).
- front (vitest hors sandbox) : `appShellStore.test.ts` — entrée non-active masquée/désactivée, active inchangée.
- archlint : `no_slug_comparison_test.go` reste vert (aucun nouveau gate par slug).

**Logging** : `slog.WarnContext(ctx, "title_rejected", "title", slug, "status", string(desc.Status), "path", r.URL.Path)` — clé `title` (PAS `titleSlug`) ; corriger au passage la dette `titleSlug` de `require_capability.go:48`.

**Exit gate** : Halo byte-identique (HI `StatusActive` → routes + `/bootstrap` inchangés, golden vert) + synthetic_b `coming_soon`/`archived` route correctement vers `503 title_unavailable` + capability/Status-absente dégrade proprement (body machine-readable, jamais 500/panic) + `no_slug_comparison_test` vert.

**Dérive re-vérif** : a changé — le cadrage seed « Status jamais LU en prod » est inexact : `BuildAvailableTitles` (bootstrap_service.go:411-432) lit déjà `t.Status` et le sérialise au front (DTO + enum OpenAPI générés). Le gap réel = ENFORCEMENT (gate middleware) + FILTRAGE de liste + consommation front, pas l'exposition. a bougé — pointeur require_capability « l.46 » → test réel l.47, logique 39-67. Précision — le fixture synthétique du parity-gate s'enregistre dans un `*title.Registry` (pas le package `synthetic_title_b` qui opère au niveau mappings) ; pas de slug littéral `synthetic_test_title` dans le code. Dette annexe relevée : log clé `titleSlug` (camelCase) en require_capability.go:48 viole la règle logging `title`.

---

### PMT-9 — Registre de migrations par titre + schema_version par titre  (sévérité: major)

**Axes** : MT-23

**Statut couverture actuelle** : **done ✅** (`743f9467c` : `RunForTitleDB(db, slug, target)` route par set enregistré (`TitleMigrationSet`/`RegisterMigrationSet`) ou retombe sur le défaut Halo byte-identique ; ledger `title_schema_version` + colonne `title_slug` sur `schema_migrations` ; oracle b `synthetic_title_b`). **2 déviations doc** : PK `name` conservée (DB per-titre → pas de collision), `canonicalOrder` reste dans le runner (ordre unifié global+title). *Gap initial* : liste globale Halo unique exécutée contre la DB de chaque titre, runner sans paramètre `titleSlug`.

**Évidence (⚠ RE-VÉRIFIER avant exécution — pointeurs datés 2026-06-13, re-vérifiés 2026-06-14)** :
- `internal/migration/registry.go:42-49` — `Register()`/`All()` peuplent un `var registry []Migration` (:39) global, sans dimension titre. Confirmé.
- `internal/migration/registry.go:68-99` — `ensureMigrationTable` crée `schema_migrations(name VARCHAR PRIMARY KEY, …)` ; `getApplied` lit par `name` seul. Aucune colonne `title_slug` ni `schema_version`. Confirmé.
- `internal/migration/registry.go:108-195` — `RunForDB(db, target)` → `RunSteps(db, target, stepsForTarget(target))`. Signature paramétrée par `TargetDB` UNIQUEMENT, jamais par slug. Confirmé.
- `internal/migration/order.go:15-167` — `canonicalOrder` = UNE liste globale ordonnée mêlant contenu Halo (`add_asset_translations` :21, `add_weapon_labels` :26, `seed_ranked_playlists_catalog` :52, `fix_super_fiesta_fr_label` :39) et structure transverse, tous targets confondus. ~150 steps (PAS 54). Confirmé.
- `internal/games/halo_infinite/migrations/steps.go:26-177` — `Steps()` contient déjà 6 steps title-owned déplacés (`add_pve_schema`, `shared_add_t0_quality`, `shared_add_participation_info_booleans`, `shared_add_participation_timestamps`, `add_shared_match_csrs`, `shared_seed_tier_boundaries_v2`). `StepsFor(target)` (:165-177) filtre par `TargetDB` SEUL, jamais par slug. ⚠ Le seed disait « no-op, aucun step déplacé » : FAUX aujourd'hui.
- `internal/migration/title_steps.go:18-33` — `titleStepsProvider func(TargetDB) []Migration` posé via `SetTitleStepsProvider` (signature SANS slug). `stepsForTarget` combine registre global + provider par target. Confirmé : pas de dimension titre dans la signature.
- `cmd/server/main.go:1360-1413` — `SetTitleStepsProvider(halomigrations.StepsFor)` puis `RunForDB(db, Target…)` pour metadata/shared/pve/social ; `main.go:1426` pour player. Aucun slug passé. Confirmé.
- `internal/domain/title/registry.go:194-247` — `PathResolver.{Metadata,Shared,SharedPVE,SharedSocial,Player}DBPath(titleSlug)` : les DB SONT déjà per-titre (`data/titles/{slug}/warehouse/…`). Donc le même set s'exécute physiquement contre N fichiers de titres distincts. Confirmé.

**Méthode (expand → parity → contract)** :

- **Expand** : introduire le seam title-aware SANS changer le comportement Halo.
  - Nouveau type `migration.TitleMigrationSet` (couche `domain`-pure, zéro DuckDB) : `{ Slug string; CanonicalOrder []string; Steps func(TargetDB) []Migration }`. Un `MigrationSetProvider func(slug string) (TitleMigrationSet, bool)` remplace l'actuel `titleStepsProvider func(TargetDB)` (qui ignore le slug).
  - Élargir le runner : `RunForTitleDB(db, slug, target)` (nouveau) qui résout le set du titre, applique SON `canonicalOrder` (plus la constante package globale), et trace dans un ledger par titre. `RunForDB(db, target)` devient un wrapper `RunForTitleDB(db, title.DefaultSlug, target)` — comportement Halo strictement identique.
  - Ledger `schema_version` par titre : nouvelle table `title_schema_version(title_slug VARCHAR, target VARCHAR, version INTEGER, applied_at TIMESTAMP, PRIMARY KEY(title_slug, target))`, écrite à la fin d'un cycle réussi. `schema_migrations` reste keyé par `name` MAIS gagne une colonne `title_slug` (DEFAULT 'halo_infinite' à la migration de la table elle-même → idempotent sur les DB Halo existantes, PK devient `(title_slug, name)`). Le `version` = `len(set.CanonicalOrder)` (monotone, dérivé) tant qu'on n'a pas de numérotation explicite.
  - Côté Halo : `halo_infinite/migrations` expose son `CanonicalOrder()` (la liste actuelle déménage ici depuis `order.go`, le package `migration` ne garde QUE l'algorithme de tri + le ledger). Capability-keyed : aucune comparaison de slug dans le runner ; le routage passe par le `MigrationSetProvider` (clé = slug en data, pas en littéral codé). Respecte `archlint/no_slug_comparison`.

- **Parity-gate** : ORACLE DOUBLE, les deux obligatoires.
  - (a) **Parité Halo golden** : test de caractérisation qui ouvre une DuckDB `:memory:`, lance l'ancien chemin (`RunForDB`) et le nouveau (`RunForTitleDB(_, DefaultSlug, _)`) pour chacun des 5 targets, et compare BYTE-IDENTIQUE : (i) la liste ordonnée des steps réellement appliqués, (ii) le `information_schema` complet (tables, colonnes, types, ordre), (iii) le contenu de `schema_migrations` (mêmes `name`, même cardinalité). Golden = snapshot de l'ordre actuel `CanonicalOrder()` figé en fixture. Tout réordonnancement ou step manquant échoue.
  - (b) **Exercice synthetic_test_title** : enregistrer un `MigrationSetProvider` pour `synthetic_title_b` avec un `CanonicalOrder` PROPRE (≠ Halo) — p.ex. 2 steps `synthb_create_base_shared` + `synthb_add_score_ms` — et UN target volontairement non couvert (capability absente). Prouver : (1) `RunForTitleDB(dbB, "synthetic_title_b", TargetShared)` applique les steps de B et JAMAIS les steps Halo (`add_asset_translations` absent de la DB de B) ; (2) le ledger `title_schema_version` enregistre `(synthetic_title_b, shared, 2)` distinct de la version Halo ; (3) pour un target où B n'a aucun step, le cycle est un no-op propre (0 step, 0 erreur, pas de table Halo créée) — dégradation. Sans ce titre synthétique, la factorisation reste cosmétique : il est NON optionnel.

- **Contract** : retirer le hardcode en PR minces par axe, jamais un méga-swap.
  - PR-1 : ledger seam (table `title_schema_version` + colonne `title_slug` sur `schema_migrations`) + `RunForTitleDB` wrappant `RunForDB`, callers inchangés. Garde : parité (a) verte.
  - PR-2 : déménager `canonicalOrder` du package `migration` vers `halo_infinite/migrations.CanonicalOrder()` ; `migration` ne référence plus aucune constante de contenu Halo. `order_audit_test.go` bascule pour interroger le set du titre.
  - PR-3 : changer la signature du provider `func(TargetDB)` → `func(slug)` et basculer les ~6 call-sites `RunForDB` (server/main.go, apply_shared_migrations, backfill-world, diag_bot_resolution, snapshot-world, cmd_data) vers `RunForTitleDB(db, slug, target)`, où `slug` vient du `PathResolver`/contexte d'ouverture de la DB — PAS d'un littéral. Garde : `archlint/no_slug_comparison` reste vert (routage data-driven).
  - Recoupe Phase 1.5 : Phase 1.5 a déplacé des FICHIERS DDL (6 steps title-owned) ; PMT-9 déplace le REGISTRE/ORDRE/LEDGER. Les deux convergent quand `canonicalOrder` vit côté titre.

**Tests (par couche)** :
- `domain` (migration) : `RunForTitleDB` route le bon set ; tri canonique stable ; ledger `title_schema_version` écrit la bonne version par `(slug,target)` ; `schema_migrations` PK `(title_slug,name)` ne collisionne pas entre titres sur une même DB partagée hypothétique.
- `games/halo_infinite/migrations` : `CanonicalOrder()` couvre exactement `All()∪Steps()` (reprise de `order_audit_test.go`) ; parité golden (oracle a).
- `games/synthetic_title_b` : set propre appliqué, set Halo absent, dégradation target-vide (oracle b) — dans `isolation_test.go` ou un nouveau `migration_isolation_test.go`.
- `archlint` : `no_slug_comparison` reste vert après PR-3 (aucun nouveau `slug ==`).
- `cmd/server` : smoke boot — les 5 cycles passent par `RunForTitleDB(_, DefaultSlug, _)` sans régression de migration count.

**Logging** : `slog.*Context` avec clé `title` (slug) ajoutée à TOUS les logs du cycle (aujourd'hui seul `target` est présent, registry.go:122/144/172/188) ; ajouter `schema_version` (int du ledger) sur « migration: cycle terminé » ; event id `migration.run:{slug}:{target}` (étendre la clé actuelle `migration.run:{target}` registry.go:120). Un WARN `title_schema_set_empty` si le provider renvoie un set vide pour un titre non-default (dégradation visible).

**Exit gate** : Halo byte-identique (oracle a : même ordre de steps, même `information_schema`, même `schema_migrations` que `RunForDB` aujourd'hui) + `synthetic_title_b` route son propre set et JAMAIS les steps Halo, avec un `title_schema_version` distinct + un target sans step dégrade en no-op propre (oracle b) + `no_slug_comparison` vert.

**Dérive re-vérif** : a changé / a bougé — voir le champ `drift` ci-dessus. Synthèse : (1) le provider title-owned N'EST PLUS un no-op (6 steps Halo déjà déplacés, `StepsFor` filtre par target seul) ; (2) `canonicalOrder` fait ~150 entrées jusqu'à order.go:167, pas :54 ; (3) `schema_migrations` PK=`name` seul confirmé sans dimension titre ; (4) aucun `schema_version` par titre (les occurrences sont le catalogue Prestige) ; (5) les DB sont DÉJÀ per-titre via `PathResolver`, ce qui rend le gap concret (même set N fois) ; (6) trou structurel clé : `RunForDB`/`SetTitleStepsProvider` n'ont AUCUN paramètre slug.

---

### PMT-10 — Observability — dimension titre  (sévérité: major)

**Axes** : MT-05
**Statut couverture actuelle** : partial — la dimension titre existe côté lecture haute (`MonitoringOverview`/`ConvergenceReport` prennent déjà `titleSlug`) mais est absente des collecteurs process-wide (expvar, error_collector, player_api_collector), des endpoints `/perf` + `/errors`, des logs (ContextHandler ne câble PAS `title_slug` malgré son doc-comment), et du file-logging (`LogsDir` unique).

**Évidence (⚠ RE-VÉRIFIER avant exécution — pointeurs datés 2026-06-13, re-vérifiés 2026-06-14)** :
- `internal/observability/expvar_metrics.go:23-34,41-124` — `var counters/durations sync.Map` globaux, noms de métrique = strings nus (`metricsMap.Set(name, …)`), zéro clé titre. (confirmé)
- `internal/observability/error_collector.go:51,147-154` — `key := r.Level.String()+"|"+r.Message` (l.51) ; singleton `defaultErrorColl` (l.147) ; API `ErrorBuckets()`/`ResetErrorBuckets()` (l.149-154). Pas de titre. (confirmé)
- `internal/observability/player_api_collector.go:52,131-140` — `key := call+"\x00"+player` (l.52) ; `RecordPlayerAPICall(call,player,ms,isErr)` (l.135) / `PlayerAPIStats()` (l.140). Pas de titre. (confirmé)
- `internal/observability/context_handler.go:44-51` — ⚠ DÉRIVE : doc dit `request_id + title_slug`, mais `Handle` n'ajoute que `request_id` (l.45) + `event_id` (l.48). `title_slug` jamais attaché alors que `ctxkeys.TitleSlug(ctx)` existe (`ctxkeys.go:43`).
- `internal/api/registry_monitoring.go:251,306` — `PerfStats(_ context.Context)` et `ErrorStats(_ context.Context)` : aucun `titleSlug`. (confirmé) — à comparer à `MonitoringOverview` l.50 / `ConvergenceReport` l.181 qui l'ont déjà.
- `internal/api/handlers/admin_monitoring.go:36,40,67,80,113` — `PerfStatsRunner`/`ErrorStatsRunner` sans param titre ; `GetPerf`/`GetErrors` n'appellent pas `titleOrDefault(r)` (helper existant l.113). Point d'injection `?title=`.
- `internal/domain/admin_monitoring.go:124-207` — `PerfCallStats`/`AdminPerfStats`/`AdminErrorStats`/`AdminErrorBucket` : aucun champ titre. (confirmé)
- `internal/observability/logging/config.go:24,65-80` — `LogsDir` unique (`LEVELUP_LOGS_DIR` ou `<repoRoot>/logs`), pas de namespacing par titre. (confirmé)
- `internal/games/synthetic_title_b/{adapter.go,isolation_test.go}` — titre synthétique enregistré → support de l'oracle (b).
- `internal/archlint/no_slug_comparison_test.go:26-35` — lint actif ; allowlist 2 fichiers.

**Méthode (expand → parity → contract)** :
- **Expand** : introduire le SEAM `title` dans les 3 collecteurs sans changer le comportement Halo (titre par défaut == valeur actuelle).
  - Couche `internal/observability` (port/platform — PAS de dépendance domain/title pour éviter un cycle ; le titre arrive en string, fourni par le caller via `ctxkeys.TitleSlug(ctx)`).
  - Nouvelle API titre-aware en parallèle de l'existante : `RecordDurationMST(title, name, ms)`, `IncCounterT(title,name)`, `RecordPlayerAPICallT(title,call,player,ms,isErr)`, et `record()` du error_collector clé = `title|level|message`. La clé expvar interne devient `title + "." + name` ; pour `title == title.DefaultSlug` (lu via une const string locale "halo_infinite" injectée par le caller, jamais comparée en dur dans le collecteur) on conserve **le nom nu actuel** => octets `/debug/vars` inchangés pour Halo. Les helpers historiques (`RecordDurationMS`, etc.) deviennent des wrappers `…T(defaultTitle, …)`.
  - Étendre les snapshots : `PlayerAPIStat`/`ErrorBucket`/`durationStats` exposent `Title`. DTOs `PerfCallStats`/`AdminErrorBucket`/`PerfPlayerCallStats` gagnent `Title string \`json:"title,omitempty"\`` (omitempty => sortie Halo identique tant que vide n'est pas forcé ; pour Halo on émet "" => clé absente, byte-identique).
  - Logs : câbler `title_slug` dans `ContextHandler.Handle` via `ctxkeys.TitleSlug(ctx)` mais **uniquement si présent dans le ctx** (ne pas forcer le fallback "halo_infinite" dans le handler, sinon tous les logs background gagnent un attribut => sortie non byte-identique). Clé slog = `"title"` (convention CLAUDE.md / arch-rules).
- **Parity-gate (oracle DOUBLE, obligatoire)** :
  - (a) **Parité Halo golden** : test de caractérisation `expvar_parity_golden_test.go` — émettre une séquence fixe via les helpers legacy (`RecordDurationMS`/`IncCounter`/`RecordPlayerAPICall`) + records WARN sans `title` dans le ctx, dumper `expvar.Get("levelup")` en JSON trié + `PerfStats(ctx)`/`ErrorStats(ctx)` sérialisés, comparer **byte-identique** au golden capturé sur le code AVANT seam. Idem un golden des lignes ContextHandler (ctx sans title) prouvant qu'aucun attribut `title` n'apparaît.
  - (b) **Exercice synthetic_title_b** : test `observability_title_routing_test.go` — émettre les MÊMES métriques sous `title="synthetic_title_b"` ET `"halo_infinite"`, prouver (1) clés expvar distinctes (`synthetic_title_b.<name>` vs `<name>` nu), (2) `PerfStats`/`ErrorStats` filtrés par `?title=synthetic_title_b` ne renvoient QUE les buckets de ce titre, (3) **dégradation propre** : un caller qui n'émet jamais avec titre (ou un titre inconnu sans capability d'observabilité) ne panique pas, renvoie des agrégats vides, et le fallback `ctxkeys.TitleSlug` ne pollue pas la vue Halo. Ce titre synthétique est la SEULE preuve que la factorisation route réellement.
- **Contract** (PR minces par axe, lint `no_slug_comparison` en garde, jamais de méga-swap) :
  - PR-1 (la plus mince) : câbler `title_slug` dans `ContextHandler.Handle` + corriger le doc-comment mensonger. Zéro changement Halo (ctx background sans title).
  - PR-2 : ajouter `titleSlug` aux signatures `PerfStats(ctx, titleSlug)` / `ErrorStats(ctx, titleSlug)` + runner types + `GetPerf`/`GetErrors` passent `titleOrDefault(r)` (réutilise le helper l.113). Filtrage des collecteurs par titre côté `registry_monitoring`.
  - PR-3 : basculer les call-sites d'émission (sync, persist, pooled_client_metrics, ~65 occ. dans `internal/sync`) du helper legacy vers la variante `…T` en passant `ctxkeys.TitleSlug(ctx)`. Tant que ces call-sites tournent sur Halo, sortie inchangée (clé nue).
  - PR-4 (optionnelle, différable) : namespacing `LogsDir` par titre (`<LogsDir>/<title>/{module}.log`) gardé derrière `len(registry)>1` — ne s'active que quand un 2e titre est réellement servi.
  - Aucune nouvelle entrée d'allowlist `no_slug_comparison` : le titre transite par ctx/param, jamais par `slug == "halo_infinite"`.

**Tests (par couche)** :
- `observability/` : golden parité (a) + routing synthétique (b) ; test cap/éviction inchangé sous clé titrée ; test que `RecordDurationMS` legacy == `RecordDurationMST(default,…)`.
- `api/handlers/` : `admin_monitoring_test.go` — étendre `okPerfRunner`/`okErrorRunner` pour asserter que `GetPerf?title=synthetic_title_b` et `GetErrors?title=…` propagent bien le slug (miroir des tests overview/convergence existants).
- `api/` (registry) : `PerfStats`/`ErrorStats` filtrent par titre ; `MonitoringOverview` inchangé.
- `archlint` : `TestNoNewSlugComparison` reste vert (preuve : pas de gating slug introduit).
- `domain` : sérialisation `omitempty` du champ `title` (vide => absent).

**Logging** : slog clés — `title` (slug du titre, via `ctxkeys.TitleSlug`), aux côtés de `request_id`/`event_id` existants. Sur les warns monitoring : `slog.*Context(ctx, "admin_monitoring: perf failed", "title", titleSlug, "err", err)` (aligné sur GetOverview l.126 / GetConvergence l.157). Jamais de log de la valeur de capability brute.

**Exit gate** : `/debug/vars` + `/admin/monitoring/perf` + `/errors` + lignes slog Halo **byte-identiques** au golden (oracle a) ; `synthetic_title_b` route vers des clés/buckets distincts et `?title=` filtre correctement (oracle b) ; un titre sans observabilité (ou ctx sans title) dégrade proprement (agrégats vides, zéro panic, vue Halo non polluée) ; `no_slug_comparison` vert sans nouvelle allowlist.

**Dérive re-vérif** : a bougé — voir le champ `drift` (dérive majeure : `context_handler.go` n'attache PAS `title_slug` malgré son doc ; numéros de ligne du seed recalés sur expvar/error/player collectors ; helper `titleOrDefault` + fixture `synthetic_title_b` + DTOs `domain/admin_monitoring.go` confirmés comme points d'ancrage existants).

---

### PMT-11 — Discord notifications title-aware (contenu)  (sévérité: major)

**Axes** : MT-26

**Statut couverture actuelle** : **done ✅** (outcomes — périmètre minimal) (`b571f1df5` : seam `NotifyLabels`/`OutcomeSource`, `BuildSyncEmbedWithLabels`, `NotifyConfig.Labels` ; oracle a Halo byte-identique + oracle b `synthetic_title_b` → « Triomphe »). **Reste** : footer + libellés backfill (pas de manifeste i18n par titre — hors scope minimal). *Gap initial* : contenu 100% Halo codé en dur (`discord.go` strings + `embeds.go` rendu), aucun slug dans le chemin de notification.

**Évidence (⚠ RE-VÉRIFIER avant exécution — pointeurs re-vérifiés 2026-06-14)** :
- `internal/notify/discord.go:188-297` — map `discordStrings` codée en dur ; footer l.238 (`LevelUp · Halo Infinite Stats`, FR=EN), outcomes l.189-192, KDA l.241, ranked tag l.237, libellés backfill l.218-228, `discord_last_match` l.236. **Confirmé exact.**
- `internal/notify/embeds.go:211-250` — `lastMatchLines` : map int→clé outcome l.213-218 (`1→draw 2→win 3→loss 4→quit`), icônes l.212, ranked tag l.227-230, KDA `T("discord_kda")` l.231, layout map/variant/playlist l.237-243. **C'est ici que le contenu Halo est rendu — fichier non cité par le seed.**
- `internal/notify/embeds.go:184-209` — `backfillLines` : table 11 libellés backfill Halo (LUSR/CSR/médailles/KVP/PvE…).
- `internal/notify/embeds.go:71,121` — `BuildSyncEmbed(op, …, lang)` : aucun paramètre slug ; footer via `T("discord_footer", lang)`.
- `internal/games/adapter.go:96-112` — `TitleSemanticAdapter` expose déjà `Outcomes() *mappings.OutcomeMappingSet` (peut être nil → dégrader). Seam prêt.
- `internal/games/mappings/outcomes.go:16` — `OutcomeMapping.Label(locale)` (fallback locale→en→key) ; clés canoniques `win|loss|tie|dnf` (loader_outcomes.go:25-30, enums.go OutcomeDNF).
- `config/titles/{halo_infinite,synthetic_title_b}/mappings/outcomes.toml` — les deux existent + peuplés ; synthetic diverge (Triomphe/Victory, Match nul/Draw, Forfait/Forfeit) → oracle (b).
- `internal/service/friends_orchestrator_service.go:133,157` — chemin friends thread déjà `slug` jusqu'à `NotifyFriendSyncCompleted` (libellés encore en dur).
- `internal/archlint/no_slug_comparison_test.go:26-35` — garde lint ; allowlist = registry.go + registry_career.go seulement.

**Méthode (expand → parity → contract)** :
- **Expand** : introduire un seam de présentation title-aware **sans changer aucun octet** pour Halo.
  1. Définir dans `internal/notify` une interface mince `NotifyLabels` (couche présentation, dépend de `games`, JAMAIS d'un slug littéral) : `Outcome(canonicalKey, lang) string`, `Footer(lang) string`, `BackfillLabel(canonicalKey, count, lang) string`, `RankedTag(lang) string`, `KDA(...)`. Une impl `haloLabels` câblée sur les valeurs ACTUELLES (footer figé, KDA F/D/A, outcomes via map int→clé canonique 1→tie/2→win/3→loss/4→dnf puis lookup `discordStrings`).
  2. Une impl `semanticLabels{sem games.TitleSemanticAdapter, fallback NotifyLabels}` : outcomes via `sem.Outcomes().Get(key).Label(lang)`, footer/backfill via fallback Halo tant que le manifeste i18n du titre ne les porte pas. **Aucune surface produit ne change** car au boot on injecte `haloLabels` (identique aux strings actuels).
  3. Threader un `titleSlug string` (défaut `resolver.DefaultSlug()`) à travers `PlayerSyncResult` / `BuildSyncEmbed` / `NotifySync` et un `Resolver` optionnel dans `NotifyConfig` (failsafe : nil → `haloLabels`). Gating UNIQUEMENT par `HasCapability`/présence d'adapter, jamais `slug == "halo_infinite"`.
- **Parity-gate (oracle DOUBLE, obligatoire)** :
  - (a) **Parité Halo golden/byte-identique** : test de caractérisation sur `BuildSyncEmbed` (+ `NotifyReauthRequired`, embed media, friend embeds) avec un `PlayerSyncResult` fixture riche (les 4 outcomes, ranked on/off, tous compteurs backfill, FR+EN) → sérialiser le `WebhookPayload` JSON et comparer à un golden figé pris AVANT le refactor. Zéro diff = comportement Halo inchangé.
  - (b) **Exercice synthetic_test_title** : construire un `semanticLabels` adossé au `SemanticAdapter` de `synthetic_title_b` (outcomes.toml = Triomphe/Match nul/Forfait) → asserter que l'embed rend bien « Triomphe » et PAS « Victoire » (preuve que le seam route vraiment) ; PUIS un cas `Outcomes()==nil` (capability/TOML absent) → asserter dégradation propre vers `haloLabels`/clé brute, aucun panic (le package est failsafe).
- **Contract** : PR MINCE par axe.
  1. Basculer le rendu des **outcomes** d'`embeds.go` du lookup `discordStrings` vers `NotifyLabels.Outcome(canonicalKey,…)` ; le pont int→clé canonique reste localisé. Garder les entrées `discord_outcome_*` comme fallback Halo.
  2. Basculer **footer** et **backfill labels** vers `NotifyLabels` (footer non-Halo dérivé du titre quand dispo).
  3. Câbler le `Resolver` + threader le slug réel depuis les vrais call-sites (`cmd/levelup/cmd_notify.go`, `friends_orchestrator` qui a déjà le slug, futur déclencheur sync). Lint `no_slug_comparison` reste vert (aucun nouveau gate slug introduit).

**Tests (par couche)** :
- notify (présentation) : golden parity Halo (a) FR+EN ; routing synthetic (b) ; dégradation `Outcomes()==nil` ; pont int→clé canonique (4→dnf) ; failsafe nil-resolver → haloLabels.
- mappings (existant, vérifier non-régression) : `OutcomeMapping.Label` fallback locale→en→key déjà couvert (loader_outcomes_test.go).
- archlint : `TestNoNewSlugComparison` doit rester vert (le seam n'ajoute aucun `slug ==`/`!=`).
- intégration : `NotifySync` end-to-end avec un fixture multi-joueurs hétérogène (FR), webhook stub, assert payload.

**Logging** : slog clés `title` (slug résolu), `op`, `lang`, `outcome_fallback_used` (bool, quand `Outcomes()` nil ou clé absente → fallback Halo), `notify_labels_source` (`halo`|`semantic`). Tous via `slog.*Context` ; conserver le caractère failsafe (warn, jamais d'erreur propagée).

**Exit gate** : golden Halo byte-identique (FR+EN, payload JSON) + synthetic_title_b rend « Triomphe/Match nul/Forfait » via le seam + capability/TOML outcomes absent dégrade proprement vers libellés Halo/clé brute sans panic + `no_slug_comparison` vert.

**Dérive re-vérif** : a bougé — pointeurs `discord.go` du seed tous exacts MAIS le contenu Halo est aussi (surtout) rendu dans `embeds.go` (non cité) : la surface contract est bicéphale. a changé (en mieux) — le seam `TitleSemanticAdapter.Outcomes()` + `outcomes.toml` (halo + synthetic, divergent) existe déjà ; les outcomes ne demandent aucune infra neuve, juste câblage + un pont int→clé canonique (quit Discord = `dnf` canonique). manque — `BuildSyncEmbed`/`NotifySync`/`PlayerSyncResult` ne portent aucun slug aujourd'hui ; à threader (le chemin friends thread déjà `slug`). footer/backfill labels n'ont PAS de seam manifeste i18n par titre → restent en fallback Halo jusqu'à extension du manifeste (hors scope minimal de cette phase).

---

### PMT-12 — Garde-fous & validateurs  (sévérité: major)

**Axes** : MT-21 (validateur boot required-TOML par titre) + MT-09 (cutoffs DefaultSlug → lookup registre + doc boot 2e adapter) + lint MT-12 (lint front anti-littéral `halo_infinite`)

**Statut couverture actuelle** : **gap** — aucun fail-fast boot si un TOML requis manque pour un titre enregistré (capabilities/outcomes/assets silencieusement optionnels) ; 2 cutoffs gating-par-slug encore en dur côté factories ; aucune garde front contre le littéral `halo_infinite`.

**Évidence (⚠ RE-VÉRIFIER avant exécution — pointeurs datés 2026-06-13, re-vérifiés 2026-06-14)** :
- `internal/games/mappings/registry.go:36-121` — `LoadFromConfigDir` : seul `fields.toml` obligatoire ; `loadAssetsIfExists`/`loadOutcomesIfExists`/`loadCapabilitiesIfExists` (l.123-142) retournent `nil,nil` sur `ErrNotExist` → absence silencieuse. **Pas de notion de required-set par titre.**
- `internal/api/server.go:186-190` — boot : `fieldMappingsRegistry.LoadFromConfigDir(...)` ; les erreurs ne font que `slog.Warn("field_mappings_load_warning")`. **Pas de fail-fast.**
- `internal/api/server.go:187` — `multiTitleSlugs := []string{titlePkg.DefaultSlug}` : un seul titre booté ; le validateur doit itérer sur les titres **du registre** (`title.Registry.All()`), pas sur cette liste figée.
- `internal/api/registry.go:367-376` — `dataAdapterForPDB` : `if pdb==nil || pdb.TitleSlug != title.DefaultSlug { return nil }` → **cutoff slug #1**.
- `internal/api/registry_career.go:190-205` — `TitleDataAdapter` : `if pdb.TitleSlug != title.DefaultSlug { return ErrTitleNotResolved... }` (l.195-198) → **cutoff slug #2**.
- `internal/archlint/no_slug_comparison_test.go:26-29` — `slugCompareAllowlist` = {`api/registry.go`, `api/registry_career.go`} ; commentaire « à retirer quand un 2e titre enregistre son adapter via le resolver ». C'est le **ratchet de contract** (test Go, pas golangci).
- `internal/domain/title/registry.go:100-121` — `NewRegistry()` ne registre que `halo_infinite` ; `synthetic_title_b` n'y est PAS (test-only, `isolation_test.go`).
- `config/titles/synthetic_title_b/mappings/` — 3 TOML (assets/fields/outcomes) ; **manque capabilities.toml + awards.toml** vs halo (5 TOML).
- `apps/web/eslint.config.js:8,31-38` + `apps/web/eslint-rules/no-hardcoded-strings.js` — infra de règle custom `@levelup/*` en `warn` ; **aucune règle anti-littéral-slug**.

**Méthode (expand → parity → contract)** :
- **Expand** :
  1. *Validateur required-TOML* — nouveau type `mappings.RequiredSet` (ou `RequiredTOMLPolicy`) et méthode `Registry.ValidateRequired(repoRoot string, slugs []string, policy RequiredSetResolver) []error`, où `RequiredSetResolver(slug) []string` retourne la liste des fichiers obligatoires **dérivée des capabilities du `title.TitleDescriptor`** (ex: `CapCareer`→pas de fichier requis dédié, `CapAssetImages`→`assets.toml`, présence systématique→`fields.toml`+`capabilities.toml`+`outcomes.toml`). Couche : `internal/games/mappings` (pur, 0 DB, 0 slug littéral). Le mapping capability→fichiers requis vit dans `internal/games` (consomme `title.Capability`), JAMAIS un `switch slug`.
  2. *Lookup registre vs DefaultSlug* — introduire le seam : les factories `dataAdapterForPDB`/`TitleDataAdapter` interrogent `r.titleResolver.Data(pdb.TitleSlug)` (déjà existant, resolver.go:65) au lieu de comparer à `title.DefaultSlug`. Halo reste câblé à l'identique (il EST enregistré au boot, server.go:273) → zéro changement de comportement.
  3. *Lint front* — nouvelle règle `apps/web/eslint-rules/no-title-slug-literal.js` calquée sur `no-hardcoded-strings.js`, enregistrée sous `@levelup`, bannissant le `Literal`/`TemplateElement` valant `halo_infinite` dans `src/features/**` et `src/components/**`, avec `allowlist` (tests, i18n fallback) ; démarrage en `warn`.
- **Parity-gate (ORACLE DOUBLE, non négociable)** :
  - *(a) parité Halo golden* : test de caractérisation `mappings_validation_golden_test.go` qui appelle `ValidateRequired(repoRoot, ["halo_infinite"], policy)` et asserte **0 erreur** (halo a déjà ses 5 TOML) ; + snapshot de la liste de fichiers requis calculée pour halo (golden), pour figer que le mapping capability→fichiers ne dérive pas. Pour les cutoffs : test `registry_career_parity_test.go` prouvant que `TitleDataAdapter(ctx,"halo_infinite")` retourne le **même** `*halo.DataAdapter` (capabilities byte-identiques via `Capabilities()`) avant/après bascule lookup.
  - *(b) exercice synthetic_test_title* : **compléter `config/titles/synthetic_title_b/mappings/` avec un `capabilities.toml`** déclarant un profil divergent (ex. `match.skill.snapshot=not_exposed`, `career.progression=not_exposed`) et **enregistrer `synthetic_title_b` dans une fixture de `title.Registry`** (test-only). Tests prouvant : (i) `ValidateRequired` **échoue proprement** si on retire `capabilities.toml` du fixture (erreur explicite `missing_required_toml` nommant slug+fichier, pas un panic) ; (ii) après bascule lookup, résoudre un PlayerDB `TitleSlug="synthetic_title_b"` **route vers l'adapter synthétique** (et non nil/halo), et `LoadMatchDetail`/`LoadCareerSnapshot` **dégradent** en `ErrCapabilityNotSupported`. Le titre synthétique est la seule preuve que le seam route réellement.
- **Contract** : PR mince **par axe**, jamais un méga-swap :
  - PR-A : valider + fail-fast boot. Remplacer `slog.Warn` (server.go:189) par : `ValidateRequired` sur `title.Registry.All()` → si erreur sur un titre `StatusActive`, `log.Fatal`/`os.Exit(1)` (fail-fast) ; titres `coming_soon` → `slog.Error` non bloquant. Itère sur le **registre**, pas sur `multiTitleSlugs`.
  - PR-B : basculer cutoff #1 (registry.go:368) vers lookup resolver, **retirer `api/registry.go` de l'allowlist** archlint.
  - PR-C : basculer cutoff #2 (registry_career.go:195) vers lookup resolver, **retirer `api/registry_career.go` de l'allowlist** → allowlist vide = garde mordante.
  - PR-D : lint front (`warn`) + `docs/` sur le boot d'un 2e adapter (procédure d'enregistrement registre→resolver→TOML requis).

**Tests (par couche)** :
- *mappings (pur)* : `registry_required_test.go` — required-set dérivé des capabilities ; erreur nommée sur fichier manquant ; halo OK / synthetic-sans-capabilities KO.
- *games (resolver)* : `resolver_test.go` (étendre) — lookup retourne l'adapter du slug demandé, `ErrTitleNotResolved` si absent (déjà couvert, ajouter cas synthetic enregistré).
- *api (factories)* : `registry_career_parity_test.go` (halo identique) + cas synthetic routé + dégradation `ErrCapabilityNotSupported`.
- *api (boot)* : `server_boot_validation_test.go` — fail-fast si un titre actif a un TOML requis manquant (via fixture `RepoRoot` temporaire).
- *archlint* : `no_slug_comparison_test.go` doit passer **avec allowlist vide** en fin de contract.
- *front* : `no-title-slug-literal.test.js` (calqué sur `no-hardcoded-strings.test.js`) — flag littéral, respecte allowlist.

**Logging** : slog clés `title` (obligatoire arch-rules), `event` :
- `required_toml_missing` (`title`, `path`, `required_by` = capability) — niveau Error.
- `boot_validation_failed` (`title`, `errors_count`) avant `os.Exit` pour titre actif.
- `adapter_routed` (`title`, `kind`, `source`="resolver") au lookup, pour tracer la bascule cutoff→lookup.
- réutiliser `mappings_validation_failed` existant (registry.go:55) — ne pas dupliquer.

**Exit gate** : Halo byte-identique (parité golden `ValidateRequired`=0 erreur + capabilities adapter inchangées + snapshot required-set figé) + synthetic_title_b route correctement (PlayerDB synthetic → adapter synthétique, pas nil/halo) + capability-absente dégrade proprement (`ErrCapabilityNotSupported` sur Load* non supportés, `required_toml_missing` explicite sans panic) + **allowlist archlint vide** (les 2 cutoffs slug retirés) + lint front actif (`warn`).

**Dérive re-vérif** : voir champ `drift` — résumé : (1) cutoffs présents mais lignes décalées (registry.go:368, registry_career.go:195) ; (2) resolver.go:23-26 = fallback constructeur, PAS un cutoff à retirer (la résolution n'a pas de défaut-halo silencieux) ; (3) validateur required-TOML confirmé absent + capabilities.toml bien optionnel ; (4) synthetic_title_b non enregistré au registre/boot (test-only) et manque capabilities.toml + awards.toml ; (5) no_slug_comparison = test Go (allowlist = les 2 cutoffs) ; (6) lint front absent, infra `@levelup` réutilisable.

---

### PMT-13 — Mineurs & bénins (décision documentée)  (sévérité: minor)

**Axes** : MT-24 (backup restic global) + MT-25 (cache HTTP / rate-limit) + MT-20 (self-identity de l'adapter Halo)
**Statut couverture actuelle** : partial — les trois axes sont déjà soit title-aware par construction (MT-24 découverte, MT-20 registre), soit cross-titre par nature (MT-25) ; aucun ne porte de *gap fonctionnel* — c'est une phase de **décision documentée + garde-fous**, pas d'implémentation forcée.

**Évidence (⚠ RE-VÉRIFIER avant exécution — pointeurs datés 2026-06-13, RE-VÉRIFIÉS 2026-06-14)** :
- `internal/ops/backup_service.go:53-100` — `discoverLevelUpDBs(pr)` scanne `data/titles/<slug>/` et **est déjà title-aware** : chaque target porte une clé `slug+":"+...` (lignes 73-96) pour shared/metadata/pve/social + player DBs. Confirmé.
- `internal/ops/backup_service.go:19-41` — `NewLevelUpBackupScheduler` + `toPkgConfig` : **une seule** politique (Enabled/Interval/KeepDaily/Weekly/Monthly/ResticRepo) enveloppe la découverte multi-titre → rétention/repo **globaux** (pas par titre). Confirmé.
- `pkg/duckdbbackup/scheduler.go:15-30,68-70,198-216` — `Scheduler{cfg, discover, restic}` mono-politique ; `RunOnce`/`Status` ne prennent **aucun** paramètre titre. Confirmé.
- `internal/config/config.go:132-140` — `BackupConfig` est plat (un seul jeu de seuils), pas de map par slug. Confirmé.
- `internal/api/handlers/settings_backup.go:11-36` — `GetBackupStatus` / `PostBackupRun` : **aucun param titre**, déclenchent un cycle global. Confirmé.
- `cmd/server/main.go:919` + `cmd/backup-once/main.go:25` — seuls call-sites : `NewLevelUpBackupScheduler(cfg.Backup, pr)`, instanciation unique. Confirmé.
- `internal/api/middleware/http_cache.go:18-58` (⚠ **déplacé** depuis `internal/middleware/`) — `ETagFromBytes` = SHA-256 du **body** → identité d'ETag indépendante du titre ; `CacheMaxAge`/`NoStore` posent des headers sans connaître le titre. **OK cross-titre** : la varianciation par titre est déjà portée par l'URL (le body change avec le titre, donc l'ETag aussi). Confirmé.
- `internal/api/middleware/rate_limit.go:37-57` (⚠ **déplacé**) — `LimitByIP` (IP-global, **pas** par titre) ; exemption en dur `/static/` + `/api/v1/assets/` (lignes 49-50). **Bénin** : le rate-limit IP est une protection transverse, pas une frontière de titre ; l'exemption `/api/v1/assets/` est un préfixe title-agnostic (le slug est dans le path *après*). Confirmé.
- `internal/games/halo_infinite/adapter_data.go:66` — `TitleSlug() { return titlePkg.DefaultSlug }` : **CORRECT** — l'adapter Halo *est* `halo_infinite`, il ne ment pas, il déclare son identité ; le routage par titre est porté par le registre (`internal/domain/title/registry.go:59 HasCapability`), pas par l'adapter qui s'auto-désigne. NO-ACTION fonctionnelle. Confirmé.

**Méthode (expand → parity → contract)** :
- **Expand** :
  - *MT-24* — N'introduire AUCUN seam de comportement. Le seul ajout est un **point d'extension dormant** : transformer `toPkgConfig(cfg)` en `toPkgConfig(cfg, slug)` SEULEMENT si la décision retient le « par titre optionnel ». Forme retenue par défaut = **NO-OP documenté** : ajouter dans `internal/ops/backup_service.go` un commentaire de doc-décision référençant cette spec + une constante `backupRetentionScope = "global"` (string, pas de slug littéral) qui rend explicite que la rétention est volontairement transverse. Si plus tard un titre exige une rétention dédiée, le seam est `discover()` qui retourne déjà des targets clés par slug → on filtrera/régira par `Target.Key`-prefix, jamais par comparaison de slug (couche `pkg/duckdbbackup`, ADR arch-rules : pas de littéral slug).
  - *MT-25* — Aucun seam. Ajouter un commentaire de décision au-dessus de `RateLimit` (rate-limit IP = transverse, l'exemption `/api/v1/assets/` est title-agnostic par préfixe) et au-dessus de `WriteJSONCached` (ETag = body-derived → naturellement title-correct). Optionnellement, durcir le test pour figer l'invariant (cf. Tests).
  - *MT-20* — Aucun seam. Ajouter un commentaire au-dessus de `TitleSlug()` clarifiant que retourner `DefaultSlug` est l'**auto-identité** correcte de l'adapter Halo (≠ gating), pour éviter qu'un futur lecteur le « corrige » par erreur.
  - Couche arch-rules : tout reste dans `internal/ops` (platform/ops) et `internal/api/middleware` ; zéro nouvelle dépendance ; aucune comparaison de slug introduite (garde `no_slug_comparison`).
- **Parity-gate** :
  - **(a) Parité Halo golden** : test de caractérisation sur les sorties *actuelles*, byte-identique. (1) `duckdbbackup` : golden sur la liste de `Target.Key` produite par `discoverLevelUpDBs` pour une arborescence `data/titles/halo_infinite/...` fixture → l'ensemble des clés et chemins doit rester identique au comportement actuel (preuve que la doc-décision n'altère rien). (2) middleware : golden sur les headers `Cache-Control` + valeur d'`ETag` pour un body fixe (inchangé), et sur la décision exempté/throttlé pour `/api/v1/assets/x`, `/static/y`, `/api/v1/career` (inchangé).
  - **(b) Exercice synthetic_test_title (NON optionnel)** : créer/garnir une fixture `data/titles/synthetic_test_title/` (au moins `shared_matches_v2.duckdb` + un dossier `players/<gt>/`) et prouver que `discoverLevelUpDBs` **route réellement par titre** : les targets émises portent le préfixe `synthetic_test_title:` ET coexistent avec `halo_infinite:` sans collision de clé. **Dégradation propre** : si `synthetic_test_title` n'a PAS de `shared_pve.duckdb` (capability PvE absente), la cible correspondante est silencieusement omise (lignes 78-82 : `os.Stat` gate) — le cycle ne casse pas. Pour MT-25/MT-20, le synthetic title prouve que rate-limit/cache/ETag sont **invariants au titre** (même comportement quel que soit le slug dans le path), ce qui est précisément la propriété qu'on documente comme bénigne.
- **Contract** : Rien à retirer (aucun hardcode de slug à supprimer ici — l'allowlist `no_slug_comparison` (`api/registry.go`, `api/registry_career.go`) n'est PAS touchée par cette phase). Le « contrat » de PMT-13 = **figer la décision** : (1) une entrée dans `.ai/thought_log.md` + (2) commentaires de décision in-code (3 fichiers) + (3) éventuel test-ratchet figeant l'invariant title-agnostic du middleware. PR **mince par axe** (3 micro-PR ou 1 PR à 3 commits : `docs(backup)`, `docs(middleware)`, `docs(adapter)`), aucune avec changement de comportement. Garde lint `no_slug_comparison` reste verte (on n'introduit aucune comparaison).

**Tests (par couche)** :
- **platform/ops** (`pkg/duckdbbackup` + `internal/ops`) : test golden sur la liste `Target` pour fixture `halo_infinite` (parité) ; test multi-titre `halo_infinite`+`synthetic_test_title` vérifiant préfixes de clé distincts + omission propre d'une cible PvE absente ; assertion que `toPkgConfig` produit une `Config` identique (rétention globale inchangée).
- **api/middleware** : table-test `RateLimit` (exempté pour `/static/`, `/api/v1/assets/<slug-quelconque>/...` ; throttlé pour endpoints applicatifs) — paramétré sur deux slugs pour prouver l'invariance ; test `WriteJSONCached`/`ETagFromBytes` : même body → même ETag, body différent (donc titre différent) → ETag différent (304 cohérent).
- **games/halo_infinite** : assertion `TitleSlug() == title.DefaultSlug` + un commentaire de test liant à la doc-décision (anti-régression « correction » erronée).
- **archlint** : confirmer que `TestNoNewSlugComparison` reste vert après les 3 commits (aucun nouveau littéral slug introduit).

**Logging** : slog clés — `slog.*Context(ctx, ...)` avec `"title"` = slug à chaque target de backup (déjà implicite via `Target.Key` ; ajouter `"title"` explicite dans les logs `backup: fingerprint skip` / `intégrité DB dégradée` de `scheduler.go:102,113` en dérivant le slug du préfixe de clé — amélioration d'observabilité, pas de comportement) ; `"retention_scope"="global"` une fois au démarrage du scheduler pour tracer la décision MT-24 ; middleware : aucun nouveau log (transverse, ne loggue pas le titre par design).

**Exit gate** : Halo byte-identique (golden Target-list + headers/ETag inchangés) + `synthetic_test_title` route correctement (préfixes de clé distincts, coexistence sans collision, invariance rate-limit/ETag au slug) + capability-absente dégrade proprement (cible PvE absente omise sans erreur de cycle) + `no_slug_comparison` vert + entrée `thought_log.md` posée + 3 commentaires de décision in-code mergés.

**Dérive re-vérif** :
- **a bougé** : les middlewares ont migré de `internal/middleware/` → `internal/api/middleware/` (`http_cache.go`, `rate_limit.go`). Pointeurs du seed corrigés.
- **a changé (numéros de ligne)** : `backup_service.go` — `discoverLevelUpDBs` est à **:53-100** (seed disait 53-100, OK) mais `NewLevelUpBackupScheduler`/`toPkgConfig` sont à **:19-41** (le seed citait `29-41` pour la « politique unique », exact pour `toPkgConfig`). `http_cache.go` ETag à **:39-58** (seed `39-58` ≈ OK). `rate_limit.go` IP-global + exemption à **:37-57** / exemption `/api/v1/assets/` lignes **49-50** (seed `16-57`, légère dérive). `adapter_data.go:66` `TitleSlug` → `DefaultSlug` **confirmé exact**.
- **aucun changement sémantique** : les trois constats du seed tiennent — MT-24 découverte title-aware mais politique globale ; MT-25 bénin (ETag body-derived + rate-limit IP transverse + exemption assets title-agnostic) ; MT-20 self-identity correcte (résolu par le registre `registry.go:59 HasCapability`). Confirmé aussi : `BackupConfig` (`config.go:132-140`) est plat (pas de map par titre) et les 2 seuls call-sites (`cmd/server/main.go:919`, `cmd/backup-once/main.go:25`) instancient une politique unique.

---

### EXT-1.5 — Extension Phase 1.5 — au-delà du déplacement DDL  (sévérité: major)

**Axes** : MT-16 + MT-10 + MT-18 + MT-17

**Statut couverture actuelle** : partial — la Phase 1.5 master ne couvre QUE (a) le déplacement des DDL vers `internal/games/halo_infinite/ddl/`, (b) la paramétrisation du `MigrationRunner` par `titleSlug`, et (c) un audit `internal/ops/` restreint à backup/restore/diagnose. Les 4 axes ci-dessous sont les ITEMS RESTANTS non couverts (title_id colonnes + décision, audit ops élargi à tout `cmd/*`+healthcheck+gate, seed démo, scoping notif).

**Évidence (⚠ RE-VÉRIFIER avant exécution — pointeurs RÉ-ALIGNÉS le 2026-06-14, le code avait dérivé)** :
- `internal/migration/steps_metadata.go:466-471` — DDL `mode_name_tr (mode_en, lang, name)` : PAS de `title_id` (confirmé).
- `internal/migration/steps_metadata.go:564-568` — DDL `weapon_labels (weapon_id, name_en, name_fr)` : PAS de `title_id` (seed pointait :463-640, FAUX).
- `internal/migration/steps_metadata.go:243-253` — DDL `citation_mappings` : PAS de `title_id`.
- `internal/migration/steps_metadata.go:333` — DDL `career_rank_translations` (PAS `career_rank_data.go`, qui n'est que la donnée des 272 rangs).
- `internal/domain/title/registry.go:222` — `MetadataDBPath(slug)` isole DÉJÀ metadata par chemin → aucune metadata.duckdb globale ⇒ MT-16 = décision, pas migration obligatoire.
- `cmd/populate-assets/main.go:56,98,102,112` — flag `--title-id` DÉCLARÉ mais `run()` l'IGNORE et hardcode `DefaultSlug` (bug latent, param mort).
- `cmd/refresh-metadata/main.go:344`, `cmd/migrate-static-maps/main.go:49` — `MetadataDBPath(DefaultSlug)` pinné (pas de chemin brut ; seed obsolète sur ce point).
- `internal/ops/healthcheck.go:77,78,85,86,92,98,104` — littéral `"halo_infinite"` (n'itère pas `registry.All()`).
- `internal/validation/gate.go:119,126,133,140` (littéral) + `:162` (`DefaultSlug`) — checks pinnés au titre par défaut.
- `internal/archlint/no_slug_comparison_test.go:33-35,45-56` — lint ne matche que `slug ==/!=`, PAS `f("halo_infinite")` en argument, et ne walk QUE `internal/` (cmd/* hors périmètre).
- `internal/ops/seed_demo_media.go:32` — `const haloInfinitePrefix = "Halo Infinite"` (filtre captures Halo-pinné).
- `internal/ops/seed_demo.go:398-408` — résolution xuid priorise `Profiles["halo_infinite"]` (littéral).
- `internal/migration/steps_player_notifications.go:54-61` — `notification_preferences (xuid, category)` dans `shared_social` (TargetSharedSocial), zéro dimension titre ; `SharedSocialDBPath(slug)` isole déjà par chemin.
- `internal/domain/title/registry.go:146` — `Registry.All() []*TitleDescriptor` = le seam d'itération multi-titre à utiliser partout.

**Méthode (expand → parity → contract)** :

- **Expand** :
  - *MT-16 (décision + seam)* : trancher d'abord — puisque `MetadataDBPath(slug)` isole déjà physiquement, `title_id` en colonne est défense-en-profondeur, PAS une nécessité d'isolation. Décision par défaut recommandée : NE PAS ajouter de colonne `title_id` aux 4 tables (mode_name_tr/weapon_labels/citation_mappings/career_rank_translations), documenter dans ADR 0008 que l'isolation metadata = par chemin (parité avec shared/player/pve). Le seul seam à introduire est un helper de lecture déjà title-scopé : exposer ces tables via un repo qui prend le `metadataDB` résolu par titre (déjà le cas), aucun filtre WHERE title_id requis. Couche : `internal/platform/duckdb` (port/repo). Si l'équipe veut quand même la colonne (titres futurs partageant un référentiel), la faire en step ADDITIF `ALTER TABLE … ADD COLUMN IF NOT EXISTS title_id VARCHAR DEFAULT 'halo_infinite'` (valeur ACTUELLE pour parité zéro-changement) + index, et router via `TitleDataAdapter`.
  - *MT-10 (seam d'itération)* : introduire un helper `title.Registry.All()`-driven dans chaque outil ops/cmd qui balaie aujourd'hui un seul titre — câblé pour itérer `[]{DefaultSlug}` initialement (comportement identique). Pour `populate-assets`, brancher le param `titleID` mort (lignes 102/112) sur le flag déjà déclaré. Couche : `cmd/*` + `internal/ops` + `internal/validation` (platform/ops), via `PathResolver` partout, jamais de littéral slug.
  - *MT-18* : remplacer `haloInfinitePrefix` const par une valeur lue depuis le `TitleSemanticAdapter`/`TitleDescriptor.Name` du titre démo résolu (Halo Infinite reste la valeur ACTUELLE). Remplacer la priorisation `Profiles["halo_infinite"]` par `title.DefaultSlug` puis itération `registry` (résultat identique tant qu'un seul titre). Couche : `internal/ops` (service de seed).
  - *MT-17* : décision de scoping AVANT tout code. Choix par défaut recommandé : `notification_preferences` RESTE dans `shared_social` per-title (path-isolé) — une prérence notif est par-titre (un user peut vouloir des notifs Halo mais pas Reach). Si décision inverse (prefs globales user), migrer la table vers une DB globale type `xbox_aliases.duckdb` (cf. `GlobalXuidAliasesDBPath`). Documenter le choix dans l'ADR notifications. Couche : `internal/migration` (DDL) + ADR.

- **Parity-gate (ORACLE DOUBLE — les 2 obligatoires)** :
  - **(a) Parité Halo golden (byte-identique)** : test de caractérisation sur la sortie ACTUELLE traversant chaque seam.
    - MT-16 : golden des lignes lues de `mode_name_tr`/`weapon_labels`/`citation_mappings`/`career_rank_translations` via le repo title-scopé pour `halo_infinite` == dump actuel (octet-identique).
    - MT-10 : pour `populate-assets`/`refresh-metadata`/`migrate-static-maps`/`healthcheck`/`gate` exécutés avec slug résolu = `halo_infinite`, les chemins DB ouverts et le rapport produit == comportement actuel (golden sur paths + sortie healthcheck/gate).
    - MT-18 : seed démo produit la MÊME sélection de médias + mêmes xuid résolus qu'avant (golden corpus).
  - **(b) Exercice `synthetic_test_title` (NON optionnel)** : étendre le fixture `synthetic_title_b` (config + adapter existants) pour prouver le routage RÉEL + dégradation propre.
    - MT-16 : repo metadata pour `synthetic_title_b` lit depuis SA metadata.duckdb (chemin distinct) — aucune fuite des libellés Halo ; si la table est absente (capability `assetImages`/`career` non déclarée), le repo retourne vide proprement (pas de panic).
    - MT-10 : `populate-assets --title-id synthetic_title_b` ouvre `data/titles/synthetic_title_b/warehouse/metadata.duckdb` (chemin ≠ Halo) ; `healthcheck`/`gate` itérant `registry.All()` listent les 2 titres et marquent `synthetic_title_b` dégradé/absent sans échec dur.
    - MT-18 : seed démo pour `synthetic_title_b` utilise SON `Name` de préfixe et SA résolution xuid, ou skip propre si capability `media` absente.
    - MT-17 : si la table notif est per-title, prouver que les prefs d'un xuid sous `synthetic_title_b` n'apparaissent pas sous `halo_infinite` (isolation par chemin) ; si globale, prouver le partage.

- **Contract (PR MINCE par axe, jamais de méga-swap, garde lint)** :
  - PR-A (MT-10 cmd) : brancher `populate-assets` sur son flag `titleID` + convertir `refresh-metadata`/`migrate-static-maps` à `titleID` honoré (defaut `DefaultSlug`). Retirer le param mort.
  - PR-B (MT-10 ops/validation) : `healthcheck`/`gate` itèrent `registry.All()` au lieu des littéraux `"halo_infinite"`. Remplacer chaque littéral par `DefaultSlug`/itération.
  - PR-C (MT-18) : seed démo dé-hardcodé (préfixe via adapter, xuid via `DefaultSlug`+registry).
  - PR-D (MT-16) : appliquer la DÉCISION (no-op documenté OU step additif `title_id`). Mince soit doc-only, soit 1 migration additive.
  - PR-E (MT-17) : appliquer la DÉCISION (rester per-title doc-only OU migration vers DB globale).
  - **Garde lint** : ÉTENDRE `no_slug_comparison_test.go` pour (1) aussi flaguer les littéraux `"halo_infinite"` passés en argument aux méthodes `PathResolver` (nouveau regex sur `PathResolver`/`pr.\w+\("halo_infinite"`), (2) inclure `cmd/` dans le walk. Allowlister transitoirement les sites non encore basculés, retirer l'allowlist à mesure que chaque PR merge. C'est la garde anti-régression du contract.

**Tests (par couche)** :
- `internal/platform/duckdb` : golden repo metadata title-scopé (MT-16, oracle a) + isolation synthetic (oracle b).
- `cmd/*` (`populate-assets`, `refresh-metadata`, `migrate-static-maps`) : test de résolution de chemin par `titleID` (Halo == actuel ; synthetic == chemin distinct ; param plus jamais ignoré).
- `internal/ops` : `healthcheck`/`seed_demo` itèrent `registry.All()` ; golden Halo + cas synthetic dégradé.
- `internal/validation` : `gate` multi-titre (Halo OK, synthetic absent → dégradé non bloquant).
- `internal/archlint` : extension du test `no_slug_comparison` (littéraux PathResolver + walk cmd/) — DOIT échouer sur l'état actuel avant les PR, passer après.
- `internal/migration` : MT-16 (si colonne) step additif idempotent ; MT-17 step de migration de table (si décision globale) + idempotence.

**Logging** : slog `*Context` avec clé `title` systématique sur chaque chemin touché (`populate-assets`, `refresh-metadata`, `migrate-static-maps`, `healthcheck`, `gate`, seed démo) ; clés additionnelles `slug`, `metadata_db_path`/`db_path`, `degraded` (bool, pour capability absente côté synthetic), `seed_prefix` (MT-18). Aucun log ne doit afficher un slug littéral codé en dur — la valeur vient du titre résolu.

**Exit gate** : Halo byte-identique (goldens MT-16/MT-10/MT-18 inchangés) + `synthetic_test_title` route vers ses propres chemins/données (aucune fuite Halo, healthcheck/gate le listent) + capability-absente dégrade proprement (repo vide, seed skip, gate non bloquant, zéro panic) + lint `no_slug_comparison` ÉTENDU (littéraux PathResolver + cmd/) au vert avec allowlist réduite à zéro pour les axes traités. Décisions MT-16 et MT-17 tranchées et consignées en ADR.

**Dérive re-vérif** : a bougé (branche courante = fix/reauth-banner-transient-false-positive, j'y reste) ; a changé (line:numbers ré-alignés — weapon_labels :564-568 pas :463-640 ; career = career_rank_translations:333 pas career_rank_data.go ; ops/healthcheck :77-104 pas :75-105 ; gate :119-162 pas :119-281) ; reframing (MT-16 : path-isolation rend title_id non-obligatoire → item de décision ; MT-10 : cmd tools utilisent déjà PathResolver+DefaultSlug, le vrai gap = pin DefaultSlug + flag --title-id mort dans populate-assets, PAS des chemins data/warehouse bruts ; lint actuel n'attrape pas les littéraux-argument ni cmd/).

---

### EXT-2 — Extension Phase 2 — au-delà des données match  (sévérité: major)

**Axes** : MT-07 + MT-15 + MT-14 + MT-19

**Statut couverture actuelle** : partial — la Phase 2 a factorisé l'extraction *match* canonique (DataAdapter/SemanticAdapter, RankCatalog, capabilities.toml) ; restent NON couverts les 4 sous-systèmes dérivés du match qui embarquent encore des constantes/chaînes/chemins Halo en dur : grille tiers carrière (MT-07), chaîne LUSR + poids perf (MT-15), extraction JSON participant + routing persist mono-DB (MT-14), slug progression/prestige littéral (MT-19).

**Évidence (⚠ RE-VÉRIFIER avant exécution — pointeurs datés 2026-06-14)** :
- `internal/sync/skill_config.go:11` — import direct `games/halo_infinite` (couplage titre au cœur de l'algo LUSR ; cible du contract).
- `internal/sync/skill_config.go:85-94` + `:119-133` — `CompositeWeights` / `RelativeWeights` (poids perf en dur, somme renormalisée).
- `internal/sync/skill_config.go:182-251` — `GetLUSRChain` + `lusrChainForOther`/`lusrChainForAssassin` (4 chaînes en dur via `halo_infinite.InferModeCategoryFromPairName`).
- `internal/sync/skill_config.go:285-292` — `SkillTiers` (échelle legacy 1000-2000, 6 tiers + sous-tiers Halo).
- `internal/sync/skill_config.go:37` — `CSRPlacementThresholdDefault = 5` (le "placement=5" du seed ; orthogonal aux tiers).
- `internal/analysis/skill_v2/tier.go:48-72` — `DefaultTierBoundaries()` (grille μ→tier v2, **zéro** title-routing — confirmé grep slug vide).
- `internal/analysis/combat_yield.go:20-24` — `OffensiveConversionP80=0.83` / `DefensiveResistanceP80=1.59` / clip 1.5× (P80 Halo calibrés ; pas sous skill_v2/).
- `internal/analysis/mode_label.go:55-110` — `NormalizeModeLabel` (déjà pure, mais `playlistIdentityPrefixes` Halo en dur l.49-53 ; déjà appelé par lusrChainForAssassin).
- `internal/sync/transforms.go:308-372` — extraction CoreStats + ParticipationInfo (~21 champs JSON Halo en dur : `Kills`/`Deaths`/`KDA`/`Accuracy`/`AverageLifeDuration` PT-duration/`ParticipationInfo`).
- `internal/sync/assists_model.go:25,28-49` — modèle OLS expected_assists (`minAssistsSamples=15`, features kills/deaths/dmg/mmr_delta — toutes FieldKey Halo).
- `internal/persist/batch.go:16-30,44` — `DBTarget{shared,player,pve,metadata}` mono-DB ; `MatchBatch.TitleSlug` existe mais le routing PVE ne le consomme pas (`data/titles/{slug}/` non appliqué au target 'pve').
- `internal/api/post_sync_deltas.go:101,112-114` — `defaultProgressionTitleSlug()` littéral `"halo_infinite"`.
- `internal/api/post_sync_progression.go:108-111` — `EvaluateProgressionAfterSync(ctx, pdb, titleSlug, …)` **déjà title-threadé** (le slug est paramétré, seul le call-site le hardcode).
- `internal/api/prestige_setup.go:57-71` — `titleSlug := titlePkg.DefaultSlug` (l.59) → ouvre **1 SEULE** shared_social + metadata (l.65/71) ; pas de bundle par titre.
- `internal/games/halo_infinite/adapter_semantic.go:64` — `SemanticAdapter.Ranks() *mappings.RankCatalog` (seam rangs DÉJÀ exposé) ; synthetic_title_b retourne un `RankCatalog` vide (dégrade).
- `apps/web/src/lib/skillTiers.ts:52-79` — `LUSR_TIER_GRID`/`CSR_TIER_GRID` (couplage manuel Go↔TS documenté l.50) ; + `apps/web/src/lib/charts/skillTierBands.ts`.
- `internal/domain/title/registry.go:30-66` — `Capability` consts + `HasCapability` (CapLUSR, CapCareer, CapEngagement existent déjà) ; capabilities.toml l.26-36.
- `internal/archlint/no_slug_comparison_test.go` — garde lint (existe).

**Méthode (expand → parity → contract)** :

- **Expand** :
  - *MT-07 (tiers/rangs)* : introduire `internal/games/titlerating.TierScale` (interface : `Tiers() []TierBand`, `PlacementThreshold() int`, `Scale() RatingScale{legacy|native_mu}`) résolue par titre. halo_infinite câble exactement `SkillTiers` (legacy 1000-2000) + `DefaultTierBoundaries()` (native μ) aux valeurs actuelles. Source de vérité = `config/titles/{slug}/mappings/rating.toml` (nouveau, lu par `games/mappings/loader_rating.go`), branché sur le `SemanticAdapter.Ranks()` existant pour les libellés. Couche : `games/` (adapter) + `analysis/skill_v2` consomme une `[]TierBoundary` injectée (signature déjà compatible, cf. `InferTier(mu, boundaries)`).
  - *MT-15 (chaîne LUSR + poids)* : introduire `titlerating.RatingProfile` (interface : `LUSRChain(pairName string) string`, `PerfChain(pairName string, ranked, ff bool) string`, `CompositeWeights() map[string]float64`, `RelativeWeights() map[string]float64`, `CombatYieldP80() (oc, dr, clip float64)`). halo_infinite implémente via les fonctions actuelles (`GetLUSRChain`, weights, P80 l.20-24). `skill_config.go` cesse d'importer `halo_infinite` : il reçoit un `RatingProfile`. Couche : `games/halo_infinite` (impl) ← injecté dans `sync` (port).
  - *MT-14 (extraction JSON + routing)* : l'extraction participant passe derrière `TitleDataAdapter` (déjà l'interface canonique) — `transforms.go` devient l'**impl halo_infinite** de `LoadMatchScoreboard`/projection canonique, pas un helper sync global. Le routing persist : `BatchBuilder` consomme `MatchBatch.TitleSlug` pour résoudre les chemins PVE/shared via `PathResolver.SharedPVEDBPath(slug)` au lieu d'un pool 'pve' global mono-DB. Couche : `games/halo_infinite` (extraction) + `persist` (routing title-aware, mais pas de comparaison slug — résolution par PathResolver).
  - *MT-19 (progression/prestige)* : remplacer `defaultProgressionTitleSlug()` par lecture du slug porté dans le contexte sync (`ctxkeys.TitleSlug`) — le pipeline est déjà paramétré. `PrestigeBundle` devient `PrestigeBundleSet` indexé par slug (factory par titre, ouvre shared_social/metadata via `PathResolver.SharedSocialDBPath(slug)` par titre actif du Registry). Couche : `api` (bootstrap) — seul endroit où le slug est autorisé.

- **Parity-gate (ORACLE DOUBLE — obligatoire)** :
  - **(a) Parité Halo golden** : tests de caractérisation byte-identiques AVANT/APRÈS le seam, capturés sur sorties actuelles : (i) `GetTierForRating`/`FormatTierLabel` sur balayage rating 200→2500 ; (ii) `InferTier`/`FormatTierLabel` v2 sur μ 0→30 ; (iii) `GetLUSRChain`/`GetPerformanceChain` sur le corpus de pair_names réels (fixture `testdata/pairnames_halo.json`) ; (iv) `computeCompositeScore`/score relatif sur un match fixture (mêmes float64) ; (v) projection canonique de `transforms.go` sur un match JSON Halo réel (golden `testdata/match_*.json` → snapshot 21 champs). Tout diff = échec.
  - **(b) Exercice synthetic_test_title** : étendre `synthetic_title_b` avec un `rating.toml` minimal (échelle native différente, 3 tiers, sans CapLUSR sur certains modes) + fixtures de match. Le test prouve que (1) le seam route une grille DIFFÉRENTE (sub-tiers et bornes ≠ Halo), (2) un titre SANS `CapLUSR`/sans `RatingProfile` **dégrade proprement** : pas de panic, LUSR omis, perf_score nil, prestige bundle absent (capability `not_exposed`), et que `LoadMatchScoreboard` retourne `ErrCapabilityNotSupported` sans casser le sync. Le titre synthétique est la SEULE preuve que la factorisation n'est pas cosmétique.

- **Contract** : retrait du hardcode en PR MINCES, 1 par axe, jamais de méga-swap :
  1. PR MT-07 : supprimer la dépendance directe des consommateurs aux `SkillTiers`/`DefaultTierBoundaries` globaux → injection `TierScale` ; front : remplacer le couplage manuel `skillTiers.ts:52-79` par un endpoint/manifest généré depuis `rating.toml`.
  2. PR MT-15 : retirer `import games/halo_infinite` de `skill_config.go` (l.11) → `RatingProfile` injecté ; le lint `no_slug_comparison` garde l'absence de slug littéral réintroduit.
  3. PR MT-14 : basculer le routing persist PVE sur `MatchBatch.TitleSlug` + `PathResolver` (retirer le pool 'pve' global) ; `transforms.go` devient impl adapter.
  4. PR MT-19 : remplacer `defaultProgressionTitleSlug()` littéral par `ctxkeys.TitleSlug` ; `PrestigeBundle` → set par titre. Garde : `no_slug_comparison_test.go` doit rester vert (aucun `== "halo_infinite"`).

**Tests (par couche)** :
- **domain/games** : golden parité (a) sur tier/chain/weights/P80 + projection participant ; table-test capability absente → dégrade.
- **analysis/skill_v2** : `InferTier` injecté (boundaries Halo vs synthetic) ; `combat_yield` P80 injectés.
- **sync** : `skill_config` sans import halo_infinite (test de compilation/archlint) ; OLS assists inchangé (parité coefficients).
- **persist** : routing PVE résolu via PathResolver (titre Halo → chemin actuel ; synthetic → chemin distinct) ; pas de régression ART (append-only).
- **api** : `EvaluateProgressionAfterSync` reçoit slug du contexte ; `PrestigeBundleSet` ouvre N bundles ; titre sans prestige → bundle nil sans panic.
- **archlint** : `no_slug_comparison_test.go` étendu aux nouveaux packages.
- **front (vitest)** : grille tiers dérivée du manifest = identique à l'ancienne `LUSR_TIER_GRID`/`CSR_TIER_GRID` (snapshot) ; lancer hors sandbox.

**Logging** : `slog.*Context` clé `title` systématique sur : résolution `RatingProfile`/`TierScale` (`title`, `rating_scale`, `placement_threshold`), routing persist (`title`, `db_target`, `resolved_path`), dégradation capability (`title`, `capability`, `degrade_reason="not_exposed"`), bootstrap prestige par titre (`title`, `shared_social_path`). Aucun log sans clé `title`.

**Exit gate** : Halo byte-identique (tous les goldens tier/chain/weights/P80/projection participant verts, zéro diff) + synthetic_test_title route une grille/chaîne DIFFÉRENTE prouvée + capability-absente (CapLUSR/RatingProfile/Prestige manquants) dégrade proprement (pas de panic, omission propre, `ErrCapabilityNotSupported`) + `no_slug_comparison` lint vert sur tous les packages touchés + front grille dérivée == ancienne.

**Dérive re-vérif** : a bougé — voir champ `drift` (combat_yield hors skill_v2 ; lignes skill_config décalées ; "placement=5"=CSRPlacementThresholdDefault pas SkillTiers ; pipeline progression DÉJÀ title-threadé, seul le littéral reste ; seam `Ranks()`/RankCatalog déjà exposé côté semantic). Aucun pointeur invalidé sur le fond ; corrections de lignes/chemins intégrées ci-dessus.

---

### EXT-5 — Extension Phase 5 — au-delà des hooks  (sévérité: major)

**Axes** : MT-12 (constantes littérales `'halo_infinite'`) + MT-13 (tables Halo client-side)
**Statut couverture actuelle** : partial — l'infra seam est livrée (store `currentTitleSlug`, `setApiTitleSlug`/header `X-LevelUp-Title`, `useFieldMappings`/`useAssetMapping`/`useOutcomeLabel` sur `/titles/{slug}/field-mappings`, kind `challenge_tier` déjà dans les 2 manifestes). Phase 5 a câblé la plomberie et migré `PrestigeSquadProgress` (lit `currentTitleSlug`), mais il RESTE (a) ~4 surfaces qui re-hardcodent le slug au lieu de lire le store, (b) 5 tables Halo encore en dur côté JS dont 2 ont déjà un manifeste, 3 sans kind manifeste, et (c) aucune garde lint front contre la réintroduction.

**Évidence (⚠ RE-VÉRIFIER avant exécution — pointeurs datés 2026-06-14)** :
- `apps/web/src/lib/staticAssets.ts:26` — `DEFAULT_TITLE_SLUG = 'halo_infinite'` : LA constante canonique fallback à réutiliser partout (ne pas dupliquer le littéral).
- `apps/web/src/features/ascension/AscensionProfileTab.tsx:22` — `TITLE_SLUG='halo_infinite'` consommé l.57/58/92/223/224/225/250/258 → remplacer par `useAppShellStore(s=>s.currentTitleSlug)`.
- `apps/web/src/features/ascension/AscensionRealisationsTab.tsx:26` — idem, consommé l.33.
- `apps/web/src/features/home/HomePage.tsx:320` — prop inline `titleSlug="halo_infinite"` sur `<HomePrestigeSection>` → lire le store.
- `apps/web/src/features/squad/SquadFocusStrip.tsx:178` — `SQUAD_TITLE_SLUG='halo_infinite'` (commentaire « V1 mono-titre ») → lire le store.
- `apps/web/src/features/ascension/PrestigeSquadProgress.tsx:30,68` — DÉJÀ branché `currentTitleSlug || TITLE_FALLBACK` : ne reste qu'à remplacer le littéral fallback par `DEFAULT_TITLE_SLUG` importé (cosmétique).
- `apps/web/src/lib/api/client.ts:64,72` — défaut + `!== 'halo_infinite'` pour omettre le header : légitime, mais remplacer le littéral par `DEFAULT_TITLE_SLUG` (cohérence + 1 seule source).
- `apps/web/src/lib/prestige.ts:48-71` — `TIER_COLORS` (hex) + `TIER_LABELS_FR/EN` : le kind `challenge_tier` existe DÉJÀ (color_token) dans les 2 manifestes → migrer vers `useAssetMapping('challenge_tier', tier)`.
- `apps/web/src/lib/medalDifficulty.ts:12-19` — glow RGBA par difficulté : même kind `challenge_tier` (Normal/Heroic/Legendary/Mythic), même migration.
- `apps/web/src/lib/skillTiers.ts:52-78` — `LUSR_TIER_GRID`/`CSR_TIER_GRID` : PAS de kind manifeste → créer le kind `skill_tier` (bornes+sous-paliers+style) avant externalisation.
- `apps/web/src/lib/halo/teamNames.ts:17-27` — `TEAM_NAMES_HALO_INFINITE` (9 équipes) : PAS de kind manifeste → créer kind `team_name`.
- `apps/web/src/lib/halo/outline-colors.ts:19-36` — `HALO_OUTLINE_COLORS` (16 hex) : PAS de kind manifeste → créer kind `outline_color` (hex tolérés par exception §20).
- `apps/web/src/lib/staticAssets.ts:54-57` + `apps/go-api/internal/games/halo_infinite/adapter_asset_urls.go:209-215` — préfixe `120px-HINF-CSR_*` : le **slug** est déjà paramètre (`a.titleSlug`/`titleSlug`), seul le templ `HINF-CSR_*` reste Halo-spécifique → déplacer le pattern d'ID dans l'AssetURLAdapter du titre (Go = source, JS = miroir).
- `apps/web/src/components/ui/combat-yield-bar.tsx:18-21,28` — `225` vit en docstring; la vraie magie `225 HP` est côté Go (`combat_yield.go`) → si externalisé, cible = `config/titles/halo_infinite/constants.toml` (déjà créé), HORS scope front.
- `apps/go-api/internal/archlint/no_slug_comparison_test.go:26-35` — garde Go (test ratchet). Côté front, l'équivalent à étendre est `apps/web/eslint-rules/no-hardcoded-strings.js` + `apps/web/eslint.config.js:38`.

**Méthode (expand → parity → contract)** :
- **Expand** :
  - *MT-12* : 1 seul accès canonique au slug courant — `useAppShellStore(s=>s.currentTitleSlug)` (hooks React) et `DEFAULT_TITLE_SLUG` (modules non-React/fallback). Aucun nouveau seam à inventer ; le store ET `staticAssets.DEFAULT_TITLE_SLUG` existent. Couche arch : lib/store (front), pas de slug dans `features/*`.
  - *MT-13 (déjà-manifesté)* : réutiliser `useAssetMapping('challenge_tier', tier)` (DTO `{label, color_token, icon}`). Le `color_token` traverse `tokenCssVar()`/`resolveToken()` (respect §20) au lieu des hex/RGBA inline.
  - *MT-13 (nouveaux kinds)* : étendre le schéma `AssetMappingDTO.extra` pour transporter les données structurées non-textuelles : `skill_tier` (`extra.min/max/sub_tiers/sub_tier_style`), `team_name` (label par id 0..8), `outline_color` (`extra.hex` + label). Loader Go `mappings` + endpoint field-mappings émettent ces kinds sans nouveau type Go (généricité `assets[kind][id]`). Le templ d'ID CSR (`HINF-CSR_*`) devient une méthode/const de l'`AssetURLAdapter` (Go) consommée via l'asset-URL adapter, jamais re-hardcodée JS.
  - Garde arch-rules : aucun gating par slug introduit ; tout routage = présence de la clé dans le manifeste / capability (`HasCapability`), conforme `no_slug_comparison`.
- **Parity-gate (oracle DOUBLE)** :
  - *(a) Parité Halo byte-identique* : test de caractérisation/golden sur chaque surface migrée — snapshot AVANT/APRÈS pour `halo_infinite` doit être identique : `TIER_COLORS`⟷`challenge_tier.color_token` résolus → même hex final ; `LUSR_TIER_GRID`/`CSR_TIER_GRID` reconstruits depuis `skill_tier` manifeste → mêmes bornes/sous-paliers (réutiliser `skillTiers.test.ts`/`skillTierBands.test.ts`) ; `teamNames`/`outline-colors` → mêmes noms FR/EN et hex ; URL CSR badge → string identique à `120px-HINF-CSR_*` (réutiliser `staticAssets.test.ts`). Côté Go : golden sur `field-mappings` JSON de `halo_infinite` (les 3 nouveaux kinds présents, valeurs = ancien hardcode JS).
  - *(b) Exercice synthetic_test_title* (NON optionnel) : `challenge_tier` est DÉJÀ divergent dans `synthetic_title_b` (Standard/Hard/Epic/Insane) → test prouvant que `useAssetMapping` route ces labels et PAS « Normal/Heroic ». Ajouter au corpus `synthetic_title_b/mappings/assets.toml` des entrées `skill_tier`/`team_name`/`outline_color` volontairement divergentes (ex. tiers « Rookie/Pro », équipes « Red/Blue », 1 couleur), + un test : (i) titre B route les valeurs B ; (ii) kind ABSENT (retirer p.ex. `team_name` du corpus B) → dégradation propre = retour de l'`id` brut/fallback codé, JAMAIS un crash ni une fuite des valeurs Halo.
- **Contract** : retirer le hardcode par PR MINCE, 1 axe = 1 PR :
  - PR-1 (MT-12 slug) : basculer les 4 surfaces (`AscensionProfileTab`, `AscensionRealisationsTab`, `HomePage:320`, `SquadFocusStrip`) sur le store ; aligner `client.ts`/`PrestigeSquadProgress` sur `DEFAULT_TITLE_SLUG`. Supprimer les const locales `TITLE_SLUG`/`SQUAD_TITLE_SLUG`.
  - PR-2 (challenge_tier) : remplacer `TIER_COLORS`/`TIER_LABELS_*`/`DIFFICULTY_GLOW_COLOR` par `useAssetMapping('challenge_tier', …)` ; supprimer les tables JS une fois les consommateurs (ChallengeCard, MomentCard, ObjectiveRow, StatsGlobales, LeaderboardPP) migrés.
  - PR-3 (skill_tier), PR-4 (team_name), PR-5 (outline_color) : créer le kind manifeste (les 2 titres) → endpoint l'émet → front lit le manifeste → supprimer la table JS.
  - PR-6 (badge HINF) : pousser le templ d'ID dans l'AssetURLAdapter (Go) + miroir `staticAssets.ts` lisant la valeur du titre, retirer `HINF` codé en dur.
  - Garde lint (dernière PR de chaque axe) : étendre `eslint-rules/no-hardcoded-strings.js` (ou règle sœur `no-title-slug-literal`) pour interdire le littéral `'halo_infinite'` hors `staticAssets.ts`/store/tests, passer la règle de `warn`→`error` sur le périmètre migré. Côté Go, le ratchet `no_slug_comparison_test.go` reste la garde (rien à retirer de l'allowlist tant que l'adapter d'un 2e titre prod n'est pas enregistré).

**Tests (par couche)** :
- domain/config (Go) : `mappings` loader parse les nouveaux kinds `skill_tier`/`team_name`/`outline_color` (halo + synthetic), valide `extra` typé ; parité TOML⟷fallback.
- service/api (Go) : golden JSON `/titles/halo_infinite/field-mappings` (kinds présents, valeurs = ex-hardcode) + `/titles/synthetic_title_b/...` (valeurs divergentes) + kind absent → omis proprement.
- lib (front, Vitest hors sandbox) : `staticAssets.test.ts` (URL CSR inchangée), `skillTiers.test.ts`/`skillTierBands.test.ts` (grille reconstruite identique), nouveau test `fieldMappings` pour `challenge_tier`/`skill_tier`/`team_name`/`outline_color` (route B ≠ A, fallback si absent).
- features (front) : snapshot de caractérisation des composants prestige/médaille AVANT/APRÈS migration (byte-identique pour `halo_infinite`).
- lint : test de la règle ESLint (`no-hardcoded-strings.test.js` étendu) — un littéral `'halo_infinite'` dans `features/*` échoue ; toléré dans store/staticAssets/tests.

**Logging** : slog `*Context` côté Go avec clé `title` (slug) sur l'endpoint field-mappings (kinds émis, fallback déclenché). Pas de log front nouveau ; en dev, `console.warn` une seule fois si `useAssetMapping` tombe en fallback pour un kind attendu (id retourné brut) afin de tracer un manifeste incomplet. Clés : `title`, `kind`, `asset_id`, `fallback=true`.

**Exit gate** : Halo byte-identique (goldens (a) verts : URLs/labels/hex/grilles identiques au pré-migration) + `synthetic_title_b` route correctement ses valeurs divergentes pour les 4 kinds + un kind retiré du corpus B dégrade proprement (id/fallback, zéro fuite Halo, zéro crash) + lint front `no-title-slug-literal` en `error` sur le périmètre migré + `no_slug_comparison_test.go` toujours vert. Les 5 tables JS et les 4 const slug locales sont supprimées (pas conservées « au cas où » — anti-pattern « dead code museum »).

**Dérive re-vérif** : a bougé / a changé — voir le champ `drift` détaillé. Synthèse : PrestigeSquadProgress déjà partiellement migré (fallback only) ; skillTiers.ts/medalDifficulty.ts/prestige.ts vivent sous `lib/` (pas `lib/halo/`) ; le `225 HP` est une docstring miroir d'une constante Go (hors scope front) ; le seam front (`useFieldMappings`/`useAssetMapping`) et le kind `challenge_tier` (2 titres) existent déjà — l'extension est surtout du *contract* (brancher+supprimer) plus 3 nouveaux kinds manifeste ; la garde « lint no_slug_comparison » est Go-only, l'équivalent front est `eslint-rules/no-hardcoded-strings.js` à étendre.

---

### PMT-14 — Admin : gestion des titres (+ réhabilitation Lab)  (sévérité: major)

**Axes** : MT-22 (lifecycle Status, productisé) · réutilise 1.7a/b (capabilities/feature-matrix) · productise Phase 1.8 (diagnostic déclaré-vs-DB) dans l'**admin** (pas un Lab dev) · ponts MT-04 (PMT-4 settings overlay). Trois volets : **A** (feature admin Titres), **B** (réutilisation Lab → partage), **C** (réhabilitation Lab cassé).

> Nature hybride : **A/B = feature** (oracle = réutilise endpoints existants + capability/role-gated + marche pour `synthetic_test_title`/`synthetic_title_b`) ; **C = fix** (caractériser la casse → réparer → test anti-régression d'intégration serveur). Le gabarit `expand → parity-gate → contract` s'applique au diagnostic titre (volet A) ; volet C suit un cycle fix classique.

**Statut couverture actuelle** :
- **A** : gap — aucune surface admin titres. Le Registry expose `All()`/`Get()`/`Exists()` (registry.go:131-149) ; `Status` (active/coming_soon/archived, :19-24) est **défini mais jamais lu/gaté** (MT-22, `resolveTitleSlug` title.go:30-47 ne consulte jamais `Status`). Pas de `GET /api/v1/titles` (liste) — uniquement via `BuildAvailableTitles()` (bootstrap_service.go:413) injecté dans `/bootstrap` + `/session/context`. Capabilities/feature-matrix montés derrière `MULTI_TITLE_API_ENABLED` (server.go:587-592), **non** auth-gatés admin. Phase 1.8 (diagnostic déclaré-vs-DB + export TOML draft + CLI `levelup-titles diagnose`) = ⬜ todo (index :41) ; `LabDiagnosticsResponse` n'a aucun champ titre. CLI : seul `levelup add-title` existe (cmd_title.go), `levelup-titles diagnose` absent.
- **B** : gap — atoms Lab (`_labShared.tsx` : StatusBadge, MetricCard, JsonViewer, FileStatusRow, GuardRow, TabButton, formatters) + structure handler/service/provider Lab sont réutilisables mais cloisonnés `features/lab/`.
- **C** : **broken** — backend Lab implémenté mais NON monté (`server.go` 0 occurrence `/lab`) ; front appelle 3 endpoints qui renvoient 404 ; casse masquée par MSW (handlers.ts) + tests chi-local.

**Évidence (⚠ RE-VÉRIFIER avant exécution — pointeurs vérifiés 2026-06-14)** :
- *Registry / lifecycle* : `internal/domain/title/registry.go:19-24` (`Status` enum), `:44-56` (`TitleDescriptor`), `:59-66` (`HasCapability`), `:131-149` (`Get`/`Exists`/`All`), `:85-91` (`XboxTitleIDFor`). `internal/api/middleware/title.go:30-47` — `resolveTitleSlug` ne lit JAMAIS `Status` (MT-22 confirmé).
- *Liste titres* : `internal/service/bootstrap_service.go:413-432` (`BuildAvailableTitles() []TitleSummary{Slug,Name,IconURL,Status,Capabilities,IsDefault}`). Pas d'endpoint dédié.
- *1.7a/b à réutiliser* : `internal/api/handlers/capabilities.go`, `internal/api/handlers/feature_matrix.go` ; montés `server.go:587-592` sous `cfg.MultiTitleAPIEnabled` (:576). `internal/games/mappings/registry.go` (`GetCapabilities`, `Slugs()`).
- *Admin (cible du montage A)* : `internal/api/server.go:713-741` — `r.Route("/admin")` + `middleware.RequireAuth` + `middleware.RequireAdmin`. Front : `apps/web/src/features/admin/AdminLayout.tsx`, routes file-based `apps/web/src/routes/admin/*`, gate `useAppShellStore(s=>s.isAdmin)`, i18n `lib/i18n/manifests/admin.toml`. `features/admin/sections/*` (UsersSection, InvariantsSection…), `AdminActionButton`, `middleware.NoStore`.
- *Lab cassé (volet C)* : `internal/api/handlers/lab.go:20` (`NewLabHandler`), `internal/service/lab_service.go:29-71` (`requireAccess()` lit `can_manage_instance`, **return nil si LoadAppSettings échoue** = permissif), `internal/platform/lab/provider.go`. `server.go` : **0** montage `/lab`. Front : `apps/web/src/routes/lab.tsx`, `features/lab/LabPage.tsx:30,80`, `queries.ts:53,62,71`. Masquage : `apps/web/src/test/handlers.ts` (MSW mocke `/lab/*`), `lab_test.go` (chi-local). NavL1.tsx:243 (tab gaté `can_manage_instance`, distinct de `isAdmin` :222,244).
- *Phase 1.8 (à productiser dans A)* : master `.ai/PLAN_TITLE_AGNOSTIC_REFACTORING.md:412-501` (couches `domain/diagnostic`, `port.TableInspector`, `port.TitleDiagnosticService`, impl DuckDB read-only, handler, CLI, `TitleDiagnosticSection.tsx`, **D10 : aucune écriture serveur, export presse-papier uniquement**, lint « pas d'`os.WriteFile` dans handlers Lab »). Exit Gate :1035-1054.
- *Fixtures titre synthétique* : `internal/games/synthetic_title_b/{adapter.go,isolation_test.go}` enregistré ; `config/titles/synthetic_title_b/mappings/{assets,fields,outcomes}.toml` présents **mais PAS de `capabilities.toml`** (idéal pour l'oracle « capability/donnée absente → dégradation propre »).

**Volet A — Section admin « Titres » (feature ; productise Phase 1.8 + MT-22)** :
- **Approche (expand → parity → contract)** :
  - *Expand* : nouveau sous-arbre admin `r.Route("/admin/titles")` (sous `RequireAuth`+`RequireAdmin`, server.go:713) :
    1. `GET /admin/titles` — liste : réutiliser `Registry.All()` + `BuildAvailableTitles()` (NE PAS dupliquer la projection ; extraire un helper partagé si besoin). Inclut `Status` (MT-22 enfin LU côté admin).
    2. `GET /admin/titles/{slug}` — détail : descripteur + résumé capabilities (proxy/réutilise la logique `capabilities.go`/`feature_matrix.go`, PAS un recalcul — appeler le même `FeatureChecker`/registry mappings). Pont PMT-4 : afficher l'overlay settings résolu pour le titre (lien `settings.ResolveForTitle(slug)` quand PMT-4 livré ; sinon section « hérite du global »).
    3. `GET /admin/titles/{slug}/diagnostic` — **productisation Phase 1.8** : `domain/diagnostic` + `port.TableInspector` (CountRows/ListExpectedTables read-only) + `port.TitleDiagnosticService.RunDiagnostic` comparant TOML déclaré (capabilities/fields) vs réalité DB (rows par table via `PathResolver(slug)`). Réutiliser le `FeatureChecker` de 1.7b. Handler **admin-gated** (≠ Lab `can_manage_instance`).
    4. `GET /admin/titles/{slug}/toml-draft` — export bloc `[data]`/`[capabilities]` draft, **string renvoyée telle quelle, ZÉRO écriture serveur** (décision D10). Front copie via `navigator.clipboard`.
    - Procédure « enregistrer un 2e titre » = page d'aide read-only qui documente le workflow existant (`levelup add-title` → snippet `registry.go` → `make build`), réutilisant `LabHelp`-style notice. Pas d'écriture de registre via HTTP.
    - Gating : `RequireAdmin` (role) + flag dédié (réutiliser `MultiTitleAPIEnabled` ou un nouveau `ADMIN_TITLES_ENABLED`) ; **jamais** `slug == "halo_infinite"` (lint `no_slug_comparison`) ; route par capability/registry.
  - *Parity-gate (oracle double)* :
    - (a) **Parité** : pour `halo_infinite`, `GET /admin/titles/{slug}/diagnostic` doit refléter les capabilities déclarées dans `capabilities.toml` sans drift faux-positif (golden sur le rapport) ; `GET /admin/titles` retourne le set actuel (Halo seul) byte-stable.
    - (b) **`synthetic_title_b`/`synthetic_test_title`** : enregistrer le titre synthétique (déjà fixture) → `GET /admin/titles` le liste avec son `Status` ; son diagnostic montre le DRIFT (capabilities.toml ABSENT ⇒ tout déclaré false vs DB éventuellement peuplée) ET dégrade proprement (table/DB absente → `actual=0 rows`, pas de panic, `slog.Warn capability_absent`). `toml-draft` produit un bloc collable non vide. C'est la preuve que le seam route vraiment.
  - *Contract* : PR minces : PR-1 endpoint liste + détail (réutilise 1.7a/b) ; PR-2 diagnostic (port+service+impl) ; PR-3 toml-draft + CLI `levelup-titles diagnose` (appelle `service.RunDiagnostic` direct, output text-table/`--format=json`, golden) ; PR-4 front `AdminTitlesPage` + route `routes/admin/titles*`. Garde lint verte à chaque PR.
- **Oracle de complétion A** : réutilise capabilities/feature-matrix (pas de recalcul divergent) + admin-gated + marche pour le titre synthétique (liste + diagnostic drift + dégradation propre + toml-draft non vide) + Status enfin lu/affiché.

**Volet B — Réutilisation Lab (partage, pas duplication)** :
- **Recensement** (à PORTER/partager, pas copier) :
  - Front : `_labShared.tsx` atoms (`StatusBadge`, `MetricCard`, `JsonViewer`, `FileStatusRow`, `GuardRow`, `TabButton`, formatters `formatDate/Number/Bytes`, `getStatusVariant`) → promouvoir vers un emplacement partagé (`components/ui/` ou `lib/`) consommé par Lab ET admin/titles. `LabHelp` (`LabNotice`/`LabToolSectionCard`) → pattern d'aide contextuelle pour la page « enregistrer un titre ». `DiagnosticsPanel` (cards + FileStatusRow + GuardRow) → gabarit du `<DataCapabilitiesTable>`/`<FeatureDiscrepanciesTable>` de la Phase 1.8.
  - Back : pattern handler/service/provider Lab (thin handler → service gate → provider read-only DuckDB+FS) = exactement le modèle pour `TitleDiagnosticService`+`TableInspector`. Réutiliser `writeJSON`/`writeError`, `middleware.NoStore` (diagnostics = état courant).
- **Approche** : extraire d'abord les atoms partagés (PR sans changement de comportement : Lab importe la nouvelle source), puis A les consomme. Aucune copie : si un atom diverge, le partager via prop, pas un fork (cf. règle anti-`ExplorerBanner fork`).
- **Oracle B** : 0 duplication d'atom (un seul `StatusBadge`/`MetricCard` dans le repo) ; Lab et admin/titles importent la même source ; tests Lab existants restent verts après extraction.

**Volet C — Réhabilitation Lab (FIX : caractériser → réparer → anti-régression)** :
- **Caractérisation de la casse** (cf. lab_health_detail) : backend complet mais NON monté → 3 endpoints 404 ; masquage MSW (handlers.ts) + tests chi-local ; `requireAccess()` permissif si `LoadAppSettings` échoue ; modèle d'accès = `can_manage_instance` (≠ admin role).
- **Décision panneau par panneau** : Resources = RÉPARER (monter) ; Diagnostics = RÉPARER (monter, sert d'ancrage au diagnostic titre exposé côté admin) ; Contracts (diff OpenAPI vs FastAPI legacy) = MONTER tel quel mais MARQUER pour retrait/repurpose (cutover Go fait, faible valeur) — ne pas investir ; ChartsShowcase = SAIN, laisser.
- **Réparation** : monter `LabHandler`/`LabService`/`LabProvider` dans `server.go` (instancier provider via `lab.NewProvider`, injecter dans la `ServiceRegistry`, `r.Route("/lab")` derrière la garde d'accès). **Durcir `requireAccess`** : échec `LoadAppSettings` → refuser (fail-closed), pas autoriser. Trancher le modèle d'accès : conserver `can_manage_instance` pour le Lab dev (cohérent avec NavL1) OU aligner sur `RequireAdmin` — documenter la décision (le diagnostic TITRE, lui, vit côté admin role via volet A, indépendamment).
- **Oracle C / anti-régression** : un **test d'intégration serveur** (route réelle montée, PAS chi-local, PAS MSW) prouvant `/lab/resources|contracts|diagnostics` → 200 pour un manager autorisé et 403 sinon — c'est précisément le test absent qui aurait attrapé la casse. Retirer/ajuster le mock MSW pour qu'il ne masque plus un 404.

**Tests (par couche)** :
- *domain/diagnostic* : `RunDiagnostic` 4 scénarios (no drift / drift data / drift feature / cascade) — réutiliser plan §1.8.5.
- *port/platform (TableInspector)* : DuckDB `:memory:` — table absente / présente vide / présente avec rows ; `PathResolver(slug)` route la bonne DB.
- *service (TitleDiagnosticService)* : compose TableInspector+FeatureChecker+fields ; parité halo_infinite ; drift `synthetic_title_b`.
- *api/handlers* : `/admin/titles*` admin-gated (401 sans session, 403 non-admin) ; `/admin/titles` liste avec Status ; `/admin/titles/{slug}/diagnostic` golden ; `toml-draft` string non vide ; **lint : aucun `os.WriteFile` dans handlers titres/Lab** (garde-fou D10).
- *intégration serveur (volet C)* : `/lab/*` monté répond 200/403 (anti-régression de la casse) — NON mocké.
- *CLI* : `levelup-titles diagnose --slug …` golden text-table + `--format=json`.
- *front (Vitest, hors sandbox cf. mémoire)* : `AdminTitlesPage` 3 états diagnostic (no drift / drifts / vide) + clipboard mocké + hidden si non-admin ; `_labShared` atoms partagés rendent identiquement dans Lab et admin.
- *archlint* : `no_slug_comparison` reste vert (aucun `slug == "halo_infinite"`).

**Logging (clé `title`)** : `slog.InfoContext "admin.titles.list"` (`count`) ; `slog.InfoContext "title_diagnostic.report"` (`title`, `data_drifts`, `feature_drifts`) à chaque run (pas de Warn par drift — bruit) ; `slog.WarnContext "title_diagnostic.capability_absent"` (`title`, `table`/`capability`) quand DB/donnée absente ; `slog.InfoContext "admin.titles.toml_draft"` (`title`) ; volet C montage : `slog.InfoContext "lab_routes_mounted"`. Jamais de secret/token en valeur ; toujours propager `title`.

**Exit gate** :
- A : `/admin/titles` (liste + Status MT-22 lu) + `/admin/titles/{slug}` (détail réutilisant 1.7a/b sans recalcul) + `/admin/titles/{slug}/diagnostic` (parité Halo + drift `synthetic_title_b` + dégradation propre) + `/admin/titles/{slug}/toml-draft` (presse-papier, ZÉRO écriture serveur, lint `os.WriteFile` vert) + CLI `levelup-titles diagnose` golden + page « enregistrer un 2e titre » documentée, le tout admin-gated, `no_slug_comparison` vert.
- B : 0 duplication d'atom (source unique partagée Lab↔admin), tests Lap existants verts post-extraction.
- C : `/lab/*` monté et fonctionnel (3 panneaux 200 pour manager autorisé), `requireAccess` fail-closed, test d'INTÉGRATION serveur anti-régression vert, MSW ne masque plus le 404, décision d'accès + statut Contracts documentés.

**Dérive re-vérif (2026-06-14)** :
- a changé / précisé : (1) Le seed « Lab implémenté mais non monté » est CONFIRMÉ — `server.go` n'a 0 occurrence `/lab` ni `NewLabHandler`/`NewLabService`/`lab.NewProvider` ; `NewLabHandler` n'est référencé que par `lab.go`+`lab_test.go`. La stack existe dans l'arbre principal (pas uniquement le worktree cité par le seed `lab_backend_health`). (2) **Modèle d'accès du Lab = `can_manage_instance` (capability), PAS admin role** — distinction non explicitée dans le seed, déterminante pour A (admin role) vs C (capability) ; à trancher au montage. (3) `LabService.requireAccess()` est permissif sur erreur de chargement settings (`return nil`) — fail-open à corriger. (4) `LabDiagnosticsResponse` ne contient AUCUN champ diagnostic titre → Phase 1.8 reste entièrement à construire ; le diagnostic doit être exposé côté ADMIN (volet A), pas enfermé dans le Lab dev (correction de cap vs master qui le plaçait en « Lab »). (5) `synthetic_title_b` est le bon fixture (mappings présents, `capabilities.toml` ABSENT = oracle dégradation) — préférer-le au `synthetic_test_title` net-new sauf besoin d'isolation supplémentaire. (6) Capabilities/feature-matrix montés sous `MULTI_TITLE_API_ENABLED` et NON admin-gatés : volet A les CONSOMME mais ajoute sa propre garde admin sur les nouveaux endpoints. (7) `Status` jamais lu (`title.go` resolveTitleSlug) reste vrai (MT-22) ; PMT-14 est la première surface qui le LIT (affichage), le GATING runtime du Status reste PMT-8.

