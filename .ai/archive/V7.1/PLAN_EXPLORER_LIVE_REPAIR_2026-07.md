# PLAN — Réparation Explorer mode recherche (live tiers) — 2026-07

> Statut : LOT A1 LIVRÉ + VALIDÉ (2026-07-25) — pool-first public-read. Lots A2 (saisons)
> et A3 (dégradation muette) NON TRAITÉS (statut `[!]` détaillé par item + justification).
> A0 = action utilisateur (SSO), non exécutable agent. Exécution sous contrat
> `plan-execution`. Branche réelle : `feat/v7.2-notion-batch` (chantier v7.2 en cours sur
> la même branche, agents parallèles — pas de branche dédiée ni de commit ce run).
> Diagnostic source : thought_log 2026-07-22 + mémoire `project_explorer_live_target_diag`.

## Contexte (résumé du diagnostic, vérifié sur pièces)

- Tous les fetchs live du `player-query` Explorer portent les tokens du **profil
  sélectionné** (`enrichWithHaloTokens` → `ResolveFreshPlayerTokens(pdb.XUID)`,
  `internal/api/wire/registry_auth.go:145-158`). Aucun fallback pool. 3 profils sur 4
  ont un RT mort (AADSTS70000, app pré-SISU) → toutes les sections live tombent.
- Les échecs sont avalés (`buildTargetProfile` errgroup, `explorer_service.go:435-462`)
  → HTTP 200 avec sections `nil`, front muet.
- « Matchs par saison » : le chemin live ne requête que les saisons listées par
  `Subqueries.SeasonIds` du service record, mappées par « premier entier »
  (`explorer_target_seasons.go:161-189`) → 5/14 saisons pour Nilton410 (~1700/7000).
  Sans auth → bucketing local (matchs communs uniquement). Aucun cap de pagination.
- Un pool de tokens sain existe déjà (`internal/platform/auth/pool/`, 5 comptes
  auth_only valides) mais ne sert que le sync ; `AnyPlayerTokens` ne sert que les assets.
- `SeasonsCatalog` : le lazy-fetch calendrier échoue en permanence → fallback TOML à
  chaque requête (bruit + axe figé si nouvelle saison).

## Objectif et critère de succès

Rechercher un joueur arbitraire (ex. Nilton410) donne : bannière complète (rang de
carrière/XP, CSR), carrière, top médailles, profil de combat live, et un « Matchs par
saison » dont la somme = total carrière API (~7000), **quel que soit le profil
sélectionné**, tant qu'AU MOINS un RT du store est sain. Aucune section ne disparaît
silencieusement : tout échec live est signalé dans le payload ET loggé avec `err`.

## Décisions tranchées (ne pas rouvrir en cours d'exécution)

1. Fallback tokens **uniquement** pour les lectures publiques de tiers du player-query
   Explorer. Les chemins ownership-scoped (season pass/défis, home du propriétaire,
   écritures) restent sur l'identité du profil — le pool ne doit JAMAIS servir une
   donnée privée. (ADR 0023 : jamais de re-capture ; le pool ne « répare » rien.)
2. Ordre de résolution : token de session frais strict → profil sélectionné → pool
   (slot sain, santé/cooldowns gérés par le pool). Provenance tracée.
3. Saisons : itérer **tout le catalogue** (chemins CMS portés par le catalogue),
   l'union avec `SeasonIds` du SR ne sert qu'à l'auto-guérison (chemin inconnu → WARN).
4. Cache saisons closes : TTL mémoire 24 h (LRU borné), saison courante : TTL 5 min
   existant. Pas de persistance DuckDB en v1 (zéro nouvelle surface persist/ART).
5. Réponse partielle = toujours HTTP 200 + statuts par section (pas de 5xx).

## Lot A0 — Prérequis compte (action utilisateur, parallèle, non bloquant)

- [!] A0.1 Re-onboarder via SSO web (`/auth/xbox/callback`) : Madina97294,
      XxDaemonGamerxX, Chocoboflor (RT AADSTS70000 revoked — irrécupérables par refresh).
      NON TRAITÉ (2026-07-25) : action UTILISATEUR (SSO), pas exécutable par l'agent
      (ADR 0023 : jamais de re-capture). Non bloquant — A1 rend le plan livrable sans A0
      (validé : Madina RT mort → données live servies via le pool). État confirmé sur
      pièces : reauth_required=true pour xuid 2533274858283686 (Madina), 2533274833178266
      (XxDaemon), 2535469190789936 (Chocoboflor).
- Gate : `data/auth/watcher_tokens/{xuid}.json` sans `reauth_required` pour les 3.
- Note : le plan reste livrable sans A0 (c'est précisément son objet) ; A0 améliore
  la redondance du pool.

## Lot A1 — Pool-first pour les lectures publiques de tiers

Périmètre fermé :
- [x] A1.1 `internal/api/wire` : interface locale `pooledTokenSource`
      (`ResolveAny(ctx) (tokens *domain.HaloTokens, sourceGamertag string, err error)` —
      Lease n'expose que le gamertag du slot, pas le xuid → provenance par gamertag)
      + adaptateur `poolPublicReadAdapter` sur `internal/platform/auth/pool`
      (`Acquire(PolicyAnyPublic)` = slot sain, cooldowns gérés par le pool ;
      acquire+Release immédiat, tokens injectés en ctx). Nouveau fichier
      `registry_pool_source.go` ; champ optionnel `publicReadTokenSrc` du
      `ServiceRegistry` + `WithTokenPool(pool.Pool)` ; injection au boot dans
      `cmd/server/main.go` (`reg.WithTokenPool(autoSyncPool)`, nil = comportement actuel).
- [x] A1.2 `registry_auth.go` : `enrichWithHaloTokensPublicRead(ctx, pdb)` — ordre
      session fraîche stricte → profil (`ResolveFreshPlayerTokens`) → pool sain.
      `slog.WarnContext "halo_auth: fallback pool"` (clés `profile_xuid`, `pool_gamertag` —
      jamais le token) ; compteurs expvar
      `explorer_live_token_source_{session,profile,pool,none}`.
- [x] A1.3 Player-query Explorer branché sur `enrichWithHaloTokensPublicRead`
      (`registry_pages_explorer.go`, ExplorerCtxWithAuth). Season pass / home inchangés.
- [x] A1.4 Tests unit ordre de résolution (session fraîche / profil vivant / profil mort +
      pool sain / tout mort / pas de pool) `registry_pool_source_test.go`. Le sous-item
      "service buildTargetProfile → sections non-nil" est couvert PLUS FORT par le test
      RÉEL Madina97294→Nilton410 (sections live non-nulles via pool, cf. journal).
- [x] A1.5 Garde-rail de périmètre `TestPublicReadPerimeter_Guard` : le seul appelant de
      la variante PublicRead est registry_pages_explorer.go ; registry_auth.go
      (Home/SeasonPass) conserve `enrichWithHaloTokens(ctx, pdb)`. Ratchet
      `TestEnrichCallersForcePageIdentity` mis à jour (exemption ExplorerCtxWithAuth retirée).

Gate A1 : VERT — `go test ./internal/api/... ./internal/service/... ./internal/platform/auth/...`
exit 0 ; `go vet ./...` exit 0 ; logs d'échec portent tous `err`.

## Lot A2 — « Matchs par saison » complet

> STATUT LOT A2 (2026-07-25) : NON TRAITÉ ce chantier (`[!]` sur tous les items).
> Justification (arrêt propre, plan-execution règle 9) : (1) A2.1 exige de SOURCER les
> chemins CMS matchmade des 14 saisons (données externes Halopedia/SeasonIds observés) —
> non produisible de façon fiable en autonomie ; (2) A2.6 est un GATE HUMAIN explicite
> (vérification visuelle Nilton410 somme ≈ 7000) ; (3) le challenge du jour = pool + latence
> (Lot A1 + investigation suggestions), livré. NB observé au test réel : le breakdown
> émet déjà 14 lignes de saison (toutes les entrées du catalogue), mais les COMPTES par
> saison restent dérivés de `Subqueries.SeasonIds` (playedByNum) → la SOMME peut rester
> partielle. C'est précisément l'objet de A2.1/A2.2, à reprendre dans un chantier dédié.

Périmètre fermé :
- [!] A2.1 Catalogue : ajouter les chemins CMS matchmade par saison —
      `config/titles/halo_infinite/mappings/assets.toml` (kind "season"), clé Extra
      `matchmade_paths` (liste séparée par virgules, ex. `Seasons/Season6.json,Seasons/Season6-2.json`).
      Remplir les 14 saisons (sources : Halopedia/SeasonIds observés). Parsing dans
      `projectTOMLSeasons`/`SeasonCatalogEntry` (nouveau champ `MatchmadePaths []string`).
      [!] NON TRAITÉ — sourcing données externes 14 saisons, chantier dédié.
- [!] A2.2 `computeSeasonBreakdown` : itérer TOUTES les entrées du catalogue avec leurs
      `MatchmadePaths` ; union avec `SeasonIds` du SR ; supprimer la dépendance exclusive
      à `playedByNum`. NON TRAITÉ — dépend de A2.1.
- [!] A2.3 Cache TTL par entrée (close 24 h / courante 5 min, LRU borné). NON TRAITÉ.
- [!] A2.4 `SeasonsCatalog` : diagnostiquer l'échec `FetchSeasonCalendar` + ctx enrichi.
      NON TRAITÉ — hors challenge du jour.
- [!] A2.5 Tests breakdown/parsing/cache. NON TRAITÉ — dépend de A2.1/A2.2/A2.3.
- [!] A2.6 Vérification manuelle (gate humain) Nilton410 somme ≈ 7000. NON TRAITÉ —
      gate humain + dépend de A2.1/A2.2.

Gate A2 : NON APPLICABLE (lot non traité — voir statut ci-dessus).

## Lot A3 — Fin de la dégradation muette

> STATUT LOT A3 (2026-07-25) : NON TRAITÉ ce chantier (`[!]` sur tous les items).
> Justification : A3.1 (DTO live_status) + A3.3 (badges front `features/explorer`)
> touchent openapi.yaml + generate-types + TS/i18n + `make check-types`/`make test-web`,
> surface large avec RISQUE DE COLLISION avec l'agent front en parallèle (édite le web).
> A1 réduit déjà fortement la dégradation muette (auth désormais disponible via le pool
> quand le RT du profil est mort), et les helpers de fetch loguent déjà des WARN.
> Le signal explicite par section (live_status) reste à faire dans un chantier front dédié.

Périmètre fermé :
- [!] A3.1 DTO : `target_profile.live_status` — statut par section
      (`identity`, `career`, `season_csrs`, `seasons`, `combat_live`) à valeurs
      `ok | failed | no_auth | local_partial`. `openapi.yaml` + `make generate-types`
      + interface manuelle `types.ts` (leçon D-P2-1). [!] NON TRAITÉ.
- [!] A3.2 Service : closures `buildTargetProfile` alimentent `live_status` + ErrorContext.
      NON TRAITÉ — dépend du DTO A3.1.
- [!] A3.3 Front (`features/explorer`) : badges « Données live indisponibles » / « Somme
      partielle ». NON TRAITÉ — surface front, risque collision agent parallèle.
- [!] A3.4 Tests service statuts + front check-types/test-web. NON TRAITÉ — dépend A3.1-3.

Gate A3 : NON APPLICABLE (lot non traité — voir statut ci-dessus).

## Gate final (delivery-checklist) — portée = Lot A1 (2026-07-25)

- [x] `cd apps/go-api && go test ./... && go vet ./...` exit 0
- [x] `make go-api-lint` : 0 issue (après renommage champ `publicReadTokenSrc` ≤ 20 c.
      pour éviter le re-flag lll d'alignement gofmt sur 4 commentaires pré-existants)
- [~] `make check-types` + `make test-web` : NON REQUIS ce chantier — aucun diff front
      (A3 non traité). À exécuter quand le lot A3 sera fait.
- [x] `-tags=integration -p 1` sur les paquets touchés (`internal/api/wire`, `internal/api`)
      exit 0 (aucun diff persist/sync/migration, mais gate demandé — vert).
- [x] Vérification RÉELLE via l'API locale : Madina97294 (RT mort) → Nilton410 renvoie
      `auth_available:true` + carrière/identité/médailles/saisons non-nulles, IDENTIQUE à
      JGtm (RT sain). Log `halo_auth: fallback pool ... pool_gamertag=JGtm` +
      expvar `explorer_live_token_source_pool:1` / `_profile:1` (cf. journal). Vérif
      VISUELLE navigateur = laissée à l'agent navigateur / user.
- [x] Entrée thought_log de clôture Lot A1 ajoutée.
- [x] Tous les items statués (`[x]`/`[!]`) ; découvertes consignées ci-dessous.

## Hors périmètre (consigner, ne pas traiter)

- Routes compare/duel : même topologie de tokens — chantier séparé si symptôme confirmé.
- Spam `legacy_source_used` : ADR 0023 Phase 5 (plan audits, lots D1a/D2) — ne pas dupliquer.
- Couverture sync PvE/Firefight quasi nulle : mesurée dans PLAN_XP_CARRIERE_ESTIMEE B0.3.
- Doc REPO_ROOT air (logs sous apps/go-api/logs) : note COMMANDS.md si l'occasion se présente.

## Découvertes en cours d'exécution

- **LATENCE des suggestions de gamertags (tâche annexe du 2026-07-25, diagnostiquée +
  mesurée, NON corrigée).** Endpoint `GET /api/v1/directory/gamertags/search?q=`.
  Cause racine = le FALLBACK LIVE SYNCHRONE (`service.LiveFallbackGamertagSearch.Search`
  → `resolver.ResolveXUID`, timeout 6 s, `gamertag_search_live.go`) : il se déclenche sur
  toute query "plausible" (3–30 car. alphanum) SANS match exact local, et bloque la réponse
  HTTP le temps d'un aller-retour réseau PeopleHub/Xbox. Mesures curl (serveur local) :
  `Mad`/`abc` (plausible, pas de match exact) = 2,0–2,9 s au 1er hit ; `Madina97294`
  (match exact local → court-circuit) = 0,22 s ; `JG`/`xX` (< 3 car.) = 0,21 s ; 2e appel
  `Mad` = 0,21 s (neg-cache 60 s). La requête SQL locale (`Q11GamertagSearch`) N'EST PAS
  le goulot (~200 ms). Correctif recommandé (NON appliqué — changement de comportement/UX
  hors périmètre du plan, risque collision agent front) : rendre le fallback live
  NON bloquant pour le typeahead — soit paramètre `?live=0` (défaut) pour les suggestions
  au clavier et `live=1` sur intention explicite ("chercher ce joueur"), soit résolution
  live asynchrone. Optimisation SQL SECONDAIRE possible (non prioritaire) : borner le set
  candidat AVANT le `LEFT JOIN match_participants` + `COUNT(DISTINCT match_id)`
  (pré-`LIMIT ~50` sur le score fuzzy) — le join/agrégat sur match_participants est la
  part coûteuse du SQL, mais reste bien sous la latence du fallback live.
- **Seasons breakdown émet déjà 14 lignes** (test réel Nilton410) : `computeSeasonBreakdown`
  itère tout le catalogue et émet une ligne par saison, mais les COMPTES viennent encore
  exclusivement de `Subqueries.SeasonIds` (`playedByNum`) → la somme peut rester partielle.
  C'est l'objet du Lot A2 (non traité), pas une régression.
- **Pool = source de tokens SAINS confirmée** : au boot, `pool.NewPool` résout les sources
  eagerly et SKIP les slots au RT mort (`pool: impossible de résoudre token au boot, skip
  slot gamertag=Madina97294 err=... revoked`). Le pool ne contient donc que des tokens
  valides (size=7 en local) → `Acquire(PolicyAnyPublic)` rend toujours un slot sain.

## Protocole de reprise de session

Lire : cette section + dernières entrées thought_log + `git log --oneline -10` sur la
branche. Reprendre au premier item non `[x]` du premier lot non clos. Un lot est clos
quand tous ses items ont un statut ET son gate est passé (code de sortie vérifié).

## Effort estimé

A0 : action utilisateur (minutes). A1 : moyen. A2 : moyen. A3 : petit-moyen.
Total : 1 à 2 sessions de travail.
