# PLAN — Réparation Explorer mode recherche (live tiers) — 2026-07

> Statut : PRÊT, non exécuté. Exécution sous contrat du skill `plan-execution`
> (ordre strict, gates par lot, statuts `[x]`/`[~]`/`[!]`, zéro fix hors périmètre).
> Branche cible : `fix/explorer-live-pool-seasons` (1 tâche = 1 branche, N commits).
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

- [ ] A0.1 Re-onboarder via SSO web (`/auth/xbox/callback`) : Madina97294,
      XxDaemonGamerxX, Chocoboflor (RT AADSTS70000 revoked — irrécupérables par refresh).
- Gate : `data/auth/watcher_tokens/{xuid}.json` sans `reauth_required` pour les 3.
- Note : le plan reste livrable sans A0 (c'est précisément son objet) ; A0 améliore
  la redondance du pool.

## Lot A1 — Pool-first pour les lectures publiques de tiers

Périmètre fermé :
- [ ] A1.1 `internal/api/wire` : interface locale `pooledTokenSource`
      (`ResolveAny(ctx) (tokens *halo.HaloTokens, sourceXUID string, err error)`),
      implémentée par un adaptateur sur `internal/platform/auth/pool` (slot sain
      uniquement, respect cooldowns). Injection au boot (`api/server.go`), champ
      optionnel du `ServiceRegistry` (nil = comportement actuel).
- [ ] A1.2 `registry_auth.go` : nouvelle fonction `enrichWithHaloTokensPublicRead(ctx, pdb)`
      — ordre décision n°2, log `slog.WarnContext` quand on bascule sur le pool
      (`"halo_auth: fallback pool"`, clés `profile_xuid`, `pool_xuid` — jamais le token),
      compteurs expvar `explorer_live_token_source_{session,profile,pool,none}`.
- [ ] A1.3 Brancher **uniquement** le player-query Explorer
      (`registry_pages_explorer.go:92`) sur `enrichWithHaloTokensPublicRead`.
      Les autres appels de `enrichWithHaloTokens` restent inchangés (season pass, home).
- [ ] A1.4 Tests : unit ordre de résolution (session fraîche / profil mort + pool sain /
      tout mort) avec mocks ; test service `buildTargetProfile` avec tokens pool présents
      → sections live non-nil (client Halo mocké).
- [ ] A1.5 Test garde-rail de périmètre : assertion que `SeasonPassCtxWithAuth` et le
      chemin home n'appellent PAS la variante PublicRead (grep test sur le wiring).

Gate A1 : `cd apps/go-api && go test ./internal/api/... ./internal/service/... ./internal/platform/auth/...`
exit 0 ; `go vet ./...` ; aucun nouveau `slog` sans `err` sur les chemins d'échec.

## Lot A2 — « Matchs par saison » complet

Périmètre fermé :
- [ ] A2.1 Catalogue : ajouter les chemins CMS matchmade par saison —
      `config/titles/halo_infinite/mappings/assets.toml` (kind "season"), clé Extra
      `matchmade_paths` (liste séparée par virgules, ex. `Seasons/Season6.json,Seasons/Season6-2.json`).
      Remplir les 14 saisons (sources : Halopedia/SeasonIds observés). Parsing dans
      `projectTOMLSeasons`/`SeasonCatalogEntry` (nouveau champ `MatchmadePaths []string`).
- [ ] A2.2 `computeSeasonBreakdown` : itérer TOUTES les entrées du catalogue avec leurs
      `MatchmadePaths` ; union avec les chemins `SeasonIds` du SR non couverts
      (rattachés via `extractSeasonNumber`, WARN `"explorer_seasons: chemin SR non mappé"`
      si aucun rattachement). Supprimer la dépendance exclusive à `playedByNum` ;
      `hasAuth=false` → fallback local inchangé.
- [ ] A2.3 Cache : étendre le cache season-SR (`remote_stats_cache.go`) avec TTL par
      entrée — saison close (End non-nil et < now−7 j) : 24 h ; sinon 5 min. LRU borné
      (max 4096 entrées) pour les cibles arbitraires.
- [ ] A2.4 `SeasonsCatalog` : diagnostiquer l'échec permanent de `FetchSeasonCalendar`
      (capturer l'erreur complète une fois — vraisemblablement absence de token sur ce
      chemin) ; correctif : passer le ctx enrichi (profil→pool) au lazy-fetch ; en cas
      d'échec persistant, log limité (1/h) au lieu d'un log par requête.
- [ ] A2.5 Tests : unit breakdown (mock SeasonSR) — catalogue complet parcouru, saisons
      non jouées à 0, chemin SR inconnu rattaché ; unit parsing `matchmade_paths` ;
      test cache TTL close vs courante.
- [ ] A2.6 Vérification manuelle (gate humain) : Nilton410 — somme des barres ≈ total
      carrière affiché (~7000) ; profil sélectionné ≠ JGtm.

Gate A2 : tests A2.5 verts + suite `go test ./...` (apps/go-api) exit 0 + A2.6 validé.

## Lot A3 — Fin de la dégradation muette

Périmètre fermé :
- [ ] A3.1 DTO : `target_profile.live_status` — statut par section
      (`identity`, `career`, `season_csrs`, `seasons`, `combat_live`) à valeurs
      `ok | failed | no_auth | local_partial`. `openapi.yaml` + `make generate-types`
      + interface manuelle `types.ts` (leçon D-P2-1).
- [ ] A3.2 Service : les closures de `buildTargetProfile` alimentent `live_status` et
      loggent `slog.ErrorContext(ctx, "...", "err", err)` (fin des `nil` muets).
      `career_live_fetcher.go` : le « API silent skip » remonte l'erreur sous-jacente
      dans le log + statut (`api_empty`/`forbidden_403`/`auth_missing` déjà typés).
- [ ] A3.3 Front (`features/explorer`) : badge « Données live indisponibles » par carte
      concernée quand `live_status` ≠ ok ; « Somme partielle (matchs observés) » sur le
      graphe saisons en `local_partial`. i18n FR **et** EN (`Record<Locale, T>`),
      aucune couleur hex (tokens sémantiques).
- [ ] A3.4 Tests : unit service statuts (échec identité → `identity=failed`) ;
      front `make check-types` (purge `node_modules\.tmp` avant) + `make test-web`.

Gate A3 : `make generate-types && make check-types && make test-web` exit 0 ;
grep : aucun nouveau littéral hex dans `features/` ; i18n parité par typage.

## Gate final (delivery-checklist)

- [ ] `cd apps/go-api && go test ./... && go vet ./...` exit 0
- [ ] `make check-types` (cache purgé) + `make test-web` + `make go-api-lint`
- [ ] Aucun test intégration requis (pas de diff persist/sync/migration) — sinon
      `-tags=integration -p 1` obligatoire
- [ ] Vérification visuelle : recherche Nilton410 avec chaque profil sélectionné
      (JGtm ET un profil au RT mort si A0 non fait) — bannière + saisons + statuts
- [ ] Entrée thought_log par lot clos + entrée de clôture
- [ ] Aucun item sans statut ; découvertes consignées ci-dessous, non traitées

## Hors périmètre (consigner, ne pas traiter)

- Routes compare/duel : même topologie de tokens — chantier séparé si symptôme confirmé.
- Spam `legacy_source_used` : ADR 0023 Phase 5 (plan audits, lots D1a/D2) — ne pas dupliquer.
- Couverture sync PvE/Firefight quasi nulle : mesurée dans PLAN_XP_CARRIERE_ESTIMEE B0.3.
- Doc REPO_ROOT air (logs sous apps/go-api/logs) : note COMMANDS.md si l'occasion se présente.

## Découvertes en cours d'exécution

(vide — à remplir pendant l'exécution)

## Protocole de reprise de session

Lire : cette section + dernières entrées thought_log + `git log --oneline -10` sur la
branche. Reprendre au premier item non `[x]` du premier lot non clos. Un lot est clos
quand tous ses items ont un statut ET son gate est passé (code de sortie vérifié).

## Effort estimé

A0 : action utilisateur (minutes). A1 : moyen. A2 : moyen. A3 : petit-moyen.
Total : 1 à 2 sessions de travail.
