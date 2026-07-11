# CLAUDE.md - Instructions pour agents IA

> Ce fichier est lu par Claude Code et autres agents IA au début de chaque session.
> Réécrit le 2026-07-02 (purge du monde Python supprimé, chemins v7 multi-titre, règles
> d'exécution des plans). Si une affirmation de ce fichier contredit le code : le code fait
> foi, corriger ce fichier dans le même commit.

## Contexte Projet

**LevelUp** — Dashboard de statistiques multi-titres Halo (Halo Infinite + Halo 5, extensible) :

| Composant | Stack | Emplacement |
|---|---|---|
| API + sync + analyse | **Go** (chi + Huma, slog, DuckDB) | `apps/go-api/` |
| Frontend | **React/TypeScript** (Vite, TanStack Router/Query/Table, ECharts) | `apps/web/` |
| Stockage | **DuckDB** par titre + Parquet (archives) | `data/titles/{slug}/` |
| Config | JSON + TOML | `db_profiles.json`, `app_settings.json`, `.env.local`, `config/titles/` |

La migration Python→Go est **terminée** : les anciens dossiers Python (code source +
environnement virtuel) n'existent plus. Il reste 3
scripts `.py` utilitaires (2 générateurs de fixtures sous `apps/go-api/tests/`, 1 analyse
ponctuelle sous `scripts/`). **Ne pas écrire de nouveau Python** : tout code applicatif est
en Go ou TypeScript. **SQLite interdit** : DuckDB uniquement.

## Workflow Agentique

**AVANT TOUTE ACTION** :
- `.ai/thought_log.md` — journal des décisions (lire les entrées récentes)
- Le plan actif du chantier en cours (`.ai/V7/PLAN_*.md`) s'il existe
- `docs/ARCHITECTURE_V6.md` + `docs/FOUNDATIONS_GUIDE.md` (onboarding)
- `.ai/project_map.md` — cartographie (vérifier la date : doctrine RE-VÉRIFIER, les
  documents `.ai/` rotent plus vite qu'ils ne sont maintenus)

**APRÈS CHAQUE MODIFICATION SIGNIFICATIVE** : mettre à jour ces fichiers.

**THOUGHT LOG — RÈGLE OBLIGATOIRE** : avant tout commit (ou à défaut avant de rendre la
main), ajouter une entrée dans `.ai/thought_log.md` avec : date `[YYYY-MM-DD]`, titre,
statut (En cours / Complété), décision technique principale, résultats observés,
conclusion / prochaine étape. L'absence d'entrée = tâche non terminée. **Rotation
trimestrielle** : quand un trimestre est clos, déplacer ses entrées vers
`.ai/archive/thought_log_<AAAA>-Q<N>.md` — le journal actif ne garde que le
trimestre courant + le précédent.

**Skills agent** (`.claude/skills/*/SKILL.md`) — invoquer le skill AVANT d'agir dans son domaine :

| Skill | Quand |
|---|---|
| `plan-execution` | Dès qu'on exécute un plan multi-étapes (OBLIGATOIRE — anti-partiel, anti-report) |
| `plan-review` | Avant de finaliser un plan d'implémentation |
| `delivery-checklist` | Avant tout commit / PR / « c'est livré » |
| `arch-rules` | Code Go (couches, ports, multi-titre, logging) |
| `canonical-types` | Types inter-titres (`internal/games/canonical/`) |
| `db-schema` | Requêtes / schéma DuckDB |
| `go-features` | Avant d'implémenter un algo (vérifier l'existant) |
| `frontend-patterns` | Code React/TS |
| `color-tokens` | Toute couleur côté web |
| `foundations-usage` | Nouvelle page ou nouveau chart |
| `halo-modes` | Normalisation modes Halo Infinite |

## Exécution des plans — règle critique

Tout plan multi-étapes s'exécute sous le contrat du skill `plan-execution`. Le résumé
tient en 5 règles (le skill fait foi) :

1. **Ordre strict, une étape à la fois** — ne jamais commencer l'étape N+1 tant que
   l'étape N n'est pas terminée ET vérifiée (gate passé).
2. **Ne jamais différer une étape exécutable maintenant.** « Je ferai X plus tard », un
   TODO, ou une étape sautée « pour avancer » = tâche non terminée. Les seuls reports
   valides : dépendance explicite du plan, blocage nécessitant l'utilisateur, délai
   d'observation prescrit (soak).
3. **Statuer chaque item** : fait `[x]` / couvert ailleurs `[~]` (référence) / non traité
   `[!]` (justification écrite). Aucun item sans statut à la clôture.
4. **Vérifier sur pièces** avant de coder et avant de cocher (rouvrir le fichier/la ligne
   cible — le code a pu bouger).
5. **Zéro fix opportuniste hors périmètre** : noter la découverte, ne pas la traiter.

## Architecture des Données (v7 — isolation par titre, ADR 0008)

Tous les chemins passent par `PathResolver` (`internal/domain/title/registry.go`).
**Jamais** de `filepath.Join(..., "data", ...)` à la main.

| Donnée | Chemin |
|---|---|
| Matchs partagés (tous joueurs) | `data/titles/{slug}/warehouse/shared_matches_v2.duckdb` |
| Référentiels (modes, armes, rangs) | `data/titles/{slug}/warehouse/metadata.duckdb` |
| Stats PvE Firefight | `data/titles/halo_infinite/warehouse/shared_pve.duckdb` |
| Social (followers, activité) | `data/titles/{slug}/warehouse/shared_social.duckdb` |
| Enrichissements joueur | `data/titles/{slug}/players/{gamertag}/stats.duckdb` |
| Aliases Xbox globaux | `data/global/xbox_aliases.duckdb` |
| Tokens auth (source unique) | `data/auth/watcher_tokens/{xuid}.json` |
| Sessions HTTP | `data/sessions/` |
| Manifests par titre | `config/titles/{slug}/title.toml` + `mappings/{fields,assets,outcomes,capabilities}.toml` |

Détail des tables : skill `db-schema`. Slugs actifs : `halo_infinite` (défaut), `halo_5`.

## Règles critiques — écritures DuckDB (anti-corruption ART)

Contexte : le bug DuckDB ART #23046 (`Failed to delete all rows from index`) a corrompu
des DBs en prod. L'éradication (ADR 0019/0026) repose sur des invariants NON NÉGOCIABLES :

1. **Toute écriture per-match sur une DB partagée** (shared, player, pve, metadata) passe
   par `internal/persist/BatchBuilder.Submit()` → `persist.*Persister.Persist()`
   (INSERT-only). Jamais d'UPSERT/`ON CONFLICT DO UPDATE` concurrent sur les tables
   critiques.
2. **Tables append-only** (`match_skill_rank`, `match_csrs`, `player_csr_snapshots`,
   `pve_match_stats`, ...) : écriture = INSERT pur avec `written_at` ; **lecture = vue
   `<table>_latest` UNIQUEMENT** (une lecture brute sert des lignes périmées — piège
   documenté ADR 0026). Recette d'ajout d'une table : ADR 0026 +
   `internal/migration/append_only_rebuild.go`.
3. **Garde-rails** : `internal/sync/no_art_patterns_test.go` (allowlist explicite) — ne
   jamais allowlister sans justification datée.
4. **Un seul process writer par DB** (ADR 0013 dblease + ADR 0016 B-swap) : RO et RW sur
   le même fichier dans deux process = interdit. Pour lire une DB potentiellement tenue
   RW : `OpenReadForQuery` (jamais `OpenReadOnly` forcé). Écritures `sync_meta` : sous
   `AcquirePlayerWriterTimeout`.
5. **shared_social** : écritures via `SharedSocialPersister` + `CHECKPOINT` (ADR 0022 —
   sans CHECKPOINT le WAL peut être perdu).
6. Piège : `CREATE TABLE IF NOT EXISTS` n'ajoute jamais une PK à une table existante →
   `ON CONFLICT` échoue. Player DBs legacy sans contraintes : pattern
   `SELECT-then-UPDATE-or-INSERT`.
7. **Ajout d'un enrichment local** sur `player_match_enrichment` : 3 étapes (migration
   ALTER + champ pointer dans `EnrichmentRow` + if-block dans `enrichmentFields()`) —
   ADR 0019 + `internal/persist/doc.go`.

## Règle auth tokens (ADR 0023)

- **Source unique** : `data/auth/watcher_tokens/{xuid}.json` via `*auth.MultiUserTokenStore`
  (`OAuthRefreshToken` + `MSALCacheJSON` par xuid).
- **JAMAIS de re-capture de token** pour « réparer » une auth : un refresh token valide se
  rafraîchit ; s'il est mort, diagnostiquer la cause (rotation perdue, mauvais xuid) avant
  tout. `AADSTS70000` = vieille app / RT étranger, pas une raison de re-capturer.
- **Onboarding** : SSO web (`/auth/xbox/callback`) ; avancé : `go run ./cmd/token-capture/ <GT>`
  ou `token-import` (RT sur stdin). Pré-requis : joueur déclaré dans `db_profiles.json`.
- **Cache process** : après rotation externe d'un RT, appeler
  `halo.InvalidateCachedPlayerTokens(xuid)` (sinon le cache 50 min sert l'ancien chain).
- **Fallbacks legacy en transition** (`sync_meta.*`, `SPNKR_OAUTH_REFRESH_TOKEN_*`) :
  encore lus ; la télémétrie `legacy_source_used` puis leur suppression (Phase 5) sont
  planifiées (plan audits, lots D1a/D2). Aucune logique métier dans le package `auth`.
- Helper canonique CLI : `auth.RefreshHaloTokensViaStoreFirst(...)`.

## Multi-titre — title-agnostic (règle transverse)

- Brancher sur **capabilities** (`HasCapability`, `CapabilityMap`, clés fines
  `capabilities.toml`), **jamais** sur `slug == "..."` (ratchet
  `no_slug_comparison_test.go`).
- Libellés/assets/outcomes via `TitleSemanticAdapter` + TOML `config/titles/{slug}/mappings/`
  — jamais de label FR/EN en dur côté Go.
- Dégradation gracieuse : `ErrCapabilityNotSupported` → réponse partielle/503 propre,
  jamais de panic ni de données d'un autre titre.
- Détail : skill `arch-rules` (adapters, PathResolver, TitleRegistry).

## Commandes Utiles

```bash
# Backend (depuis la racine)
make go-api-test            # tests Go rapides (domain/analysis/contracttest)
cd apps/go-api && go test ./...                      # suite complète
cd apps/go-api && go test -tags=integration ./...    # inclut les tests persist anti-ART (OBLIGATOIRE avant livraison sync/persist)
make go-api-lint            # golangci-lint

# Frontend
make check-types            # tsc
make test-web               # vitest
make generate-types         # openapi.yaml -> apps/web/src/lib/api/generated.ts

# Dev servers
make dev                    # go-api (air) + vite
# Redémarrage du serveur air local : Start-Process détaché, puis vérifier le port 8000

# Requêtes DuckDB ad hoc (pas de Python)
duckdb data/titles/halo_infinite/warehouse/metadata.duckdb "SELECT ..."
go run apps/go-api/cmd/inspect_bp/main.go            # outil Go (CGO : gcc msys64)

# CLI principal
go run ./apps/go-api/cmd/levelup --help              # sync, backfill, diag
```

Référence complète des commandes : `docs/COMMANDS.md`. Déploiement : `docs/RUNBOOK_GO_LIVE*`
— **push sur `main` = déploiement prod automatique** : prévenir l'utilisateur avant.

## Règles

1. **Répondre en français.** UI : FR sans anglicismes (« série » pas « streak »,
   « Taux de victoire » pas « WR ») ; toute string UI en FR **et** EN (`i18n.ts`,
   parité par typage `Record<Locale, T>`).
2. **Go/TS uniquement.** Pas de nouveau Python, pas de SQLite, pas de Pandas/Polars (morts
   avec la migration).
3. **Logging Go** : `slog.InfoContext/ErrorContext(ctx, "...", "err", err)` structuré.
   Jamais `fmt.Println`/`log.Printf`. Jamais d'erreur avalée en silence : logger AVANT
   toute dégradation best-effort.
4. **Pas d'emojis dans les fichiers versionnés.**
5. **Seuils** : fichier ≤ 500 L, fonction ≤ 80 L, ≤ 5 paramètres, complexité ≤ 12.
   Au-delà : extraire (mixins/sous-fichiers/sous-fonctions) ou exemption justifiée par
   commentaire. Dette existante gelée par baseline lint — ne pas l'accroître.
6. **≤ 2 copies d'un même pattern** : à la 3e, centraliser dans un helper ET ajouter un
   garde-rail (test grep) qui interdit l'ancien littéral — une factorisation sans
   garde-rail re-diverge (leçon : prédicat bot passé de 8 à 36 copies après
   centralisation).
7. **0 code mort** : ce qu'on débranche du routing/des callers, on le supprime avec ses
   tests et imports. Pas de « au cas où » — git garde l'historique.
8. **Timezone canonique** : `COALESCE(x.start_time_utc, x.start_time AT TIME ZONE 'UTC')`
   via le fragment SQL partagé — jamais `start_time` brut dans un filtre/tri temporel.
9. **KDA/KDR (ADR 0006)** : KDA n'est JAMAIS le quotient. Per-match = valeur API native ;
   agrégat = `((frags + assists/3) − morts) / nb_matchs`. KDR = frags/morts, distinct.
   Réutiliser les KPI dérivés existants, ne pas recalculer ad hoc.
10. **Ratings** : lecteurs → vues `_latest` uniquement (règle ART n°2). LUSR : chemin v2
    canonique (`RecomputeLUSRCanonicalForPlayer`) — v1 est mort.
11. **Feature flags** : pas de flag qui laisse une feature OFF « pour plus tard » —
    corriger la cause et livrer actif. Tout kill-switch porte en commentaire : date du
    basculement de défaut + date cible de retrait + critère mesurable (modèle :
    `platform/duckdb/shared_reader_legacy.go`).
12. **Couleurs web** : aucune valeur hex ni classe Tailwind couleur dans `features/` /
    `components/` — tokens sémantiques uniquement (skill `color-tokens`).
13. **Tableaux interactifs** : TanStack Table. **Query keys** : `lib/query/keys.ts`,
    jamais inline. **Routes** : file-based, ne jamais éditer `routeTree.gen.ts`.
14. **Vérifier l'existant avant d'implémenter** : `internal/analysis/` + skill
    `go-features` + grep des exports. Beaucoup d'algos existent déjà.
15. **docs/FR — politique** : guides majeurs (`FOUNDATIONS_GUIDE`, `COMMANDS`,
    `SYNC_GUIDE`, `ARCHITECTURE_V6`) = bilingues, toute modif EN inclut la MAJ FR dans le
    même PR. **ADRs et runbooks = EN-only** (pas de traduction à créer ni maintenir).
16. **Git** : jamais `git stash` (commit WIP à la place) ; demander avant tout commit ;
    jamais travailler sur `main` ; ne pas changer de branche si un travail est en cours.

## Diagnostic de revue de code — anti-patterns interdits

1. **Dead code museum** — conserver du code mort « au cas où » (le pire : avec des tests
   verts qui entretiennent l'illusion).
2. **Compatibility guard forever** — flag/branche legacy sans date d'expiration.
3. **God file / god function** — > 500 L / > 80 L mêlant des responsabilités.
4. **Copy-paste config** — même littéral en 3+ endroits → constante + garde-rail.
5. **Bare connect** — ouverture DuckDB hors provider/lease (viole le modèle mono-process).
6. **Magic number** — `outcome == 2` → constante/enum ; seuils nommés.
7. **Logique métier dans un handler HTTP ou un composant React** — extraire
   (service/analysis côté Go ; hook/`*_logic.ts` côté web).
8. **Factorisation abandonnée** — créer le helper canonique sans migrer les copies ni
   poser le garde-rail (la dette re-croît).
9. **Doc inversée** — commentaire qui décrit l'ancien défaut d'un flag ; la doc d'un
   kill-switch se met à jour dans le commit qui bascule le défaut.
10. **Swallowed error** — `_ = f()` / `continue` sur erreur sans log ni compteur.

## Stratégie de branches Git

### Règle fondamentale : 1 tâche = 1 branche, N commits

```
# Correct — phases séquentielles = commits sur une branche
git checkout -b refactor/cleanup-all
git commit -m "refactor(phase1): ..."
git commit -m "refactor(phase2): ..."

# Interdit — une branche par phase séquentielle (oblige rebase/merge manuels)
```

1. Vérifier la branche courante avant de committer : `git branch --show-current`
2. **JAMAIS travailler sur `main`** — aucune exception (push main = deploy prod auto)
3. Nouvelle feature/fix → nouvelle branche depuis la branche appropriée
4. Ne jamais changer de branche si un travail différent est en cours — informer l'utilisateur
5. Pas de nom fourni → en proposer un avant de créer
6. Entre sessions : `git log --oneline -10` pour reprendre au bon endroit
7. Plusieurs branches uniquement pour des tâches réellement indépendantes/parallèles

## Décisions architecturales (ADRs) — `docs/adr/`

- `0001` charts ECharts · `0002` canonical PlayerMatchRow · `0003` i18n manifests + lint ·
  `0004` narrative engine · `0005` Prestige activation phasée · `0006` indicateurs
  canoniques KDA/KDR/WinRate/Accuracy + unité 0..1 · `0007` migration canonical big-bang ·
  `0008` isolation par chemin FS + xuid global · `0009` monitoring expvar ·
  `0010` binning serveur · `0011` frontière canonical vs semantic vs asset-URL adapters ·
  `0012` extraction adapters Halo-only (`internal/games/halo_infinite/`) ·
  `0013` LeasedWriter/dblease (un seul writer RW) · `0014` progression V2 Ascension ·
  `0015` PlayerProfile V1 partiel · `0016` SharedDBProvider B-swap RO↔RW ·
  `0017`/`0018` CLOSED (supersedés par 0019) · `0019` **Collect→Persist anti-ART** ·
  `0020` coach→pont Prestige · `0021` recovery WAL shared_social ·
  `0022` shared_social Collect→Persist · `0023` **tokens source unique** ·
  `0024` LUSR v2 TrueSkill2 · `0025` refactor title-agnostic (master :
  `.ai/V7/PLAN_TITLE_AGNOSTIC_REFACTORING.md`) · `0026` **append-only ART eradication**
  (+ vues `_latest`) · `0027` sync pipeline V2 cycle orchestrator ·
  `0028` template synthesis coach · `0029` ownership joueur multi-user ·
  `0030` **persist write aggregates** (durcissement compile-time anti-ART : batch opaque,
  allowlist datée `OpenReadWrite`, garde-rail lecture `_latest`) · `0031` frontière source
  de données par titre (mutualisation HTTP `platform/httpx`, `TitleSyncRunner` ; amende 0027).

READMEs catalogues : `apps/go-api/internal/analysis/{temporal,breakdown,narrative}/README.md`,
`apps/web/src/components/charts/README.md` (wrappers ECharts).

## Serveurs MCP disponibles

**duckdb** — SQL direct sur les données :
```sql
ATTACH 'data/titles/halo_infinite/warehouse/metadata.duckdb' AS meta (READ_ONLY);
```
Attention au modèle mono-process : ne jamais ouvrir en RW une DB que le serveur tient.

**browser** — tester l'app visuellement (dev local sur :8000 / vite).
