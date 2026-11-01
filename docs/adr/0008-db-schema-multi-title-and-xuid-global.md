# ADR 0008 — DB schema multi-title : isolation par chemin FS, xuid_aliases globalisé

**Status** — Proposed (2026-04-29), amended 2026-11-01 (two roots: see the last section). Triggered by code review axe 2 (BLOQUANT « schéma DuckDB transverse sans `title_id` »).

**Deciders** — Guillaume (GS).

## Context

L'audit axe 2 a soulevé que les tables transverses (`match_registry`, `match_participants`, `medals_earned`, `killer_victim_pairs`, `highlight_events`) **ne portent pas de colonne `title_id`**. L'isolation est purement filesystem via `data/titles/{slug}/warehouse/shared_matches_v2.duckdb`.

Question initiale : faut-il ajouter `title_id` partout pour préparer le multi-titres ?

Analyse complémentaire pendant la revue :

- Chaque titre a sa propre arborescence DB sous `data/titles/{slug}/`. Le `PathResolver` (`internal/domain/title/registry.go`) compose ces chemins.
- Une connexion DuckDB ouvre **un seul fichier** à la fois (sauf `ATTACH` explicite).
- Une query `SELECT ... FROM match_registry` sur la DB Halo Infinite **ne peut physiquement pas** retourner des matchs d'un autre titre — ils sont dans un autre fichier sur disque.

**Cas particulier `xuid_aliases`** : le `xuid` est un identifiant **Microsoft/Xbox global** par construction. Le même compte Xbox utilisé sur Halo Infinite et Halo MCC produit le même xuid. Avoir 2 tables `xuid_aliases` dupliquées dans 2 DB par titre :
- duplique les données (gamertag, last_seen)
- crée un risque de divergence (gamertag mis à jour sur un titre, pas l'autre)
- ne reflète pas la sémantique réelle (le xuid n'appartient pas au jeu)

## Decision

**1. Pas de colonne `title_id` sur les tables transverses.**

L'isolation par chemin FS est suffisante. Coûts évités :
- Migration de schéma + backfill sur tables avec ~millions de lignes
- Risque de bugs (oubli `WHERE title_id = ?` dans un SELECT, INSERT incomplet)
- Storage cost : 4-8 bytes/ligne × N

**Cas cross-title** (futurs, ex: comparer KDA d'un joueur sur 2 jeux) : traités par `ATTACH` multi-DB et `'halo_infinite' AS title` ajouté **à la requête**, pas stocké en colonne.

```sql
-- Pattern cross-title sans title_id colonne
ATTACH 'data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb' AS hi;
ATTACH 'data/titles/halo_mcc/warehouse/shared_matches_v2.duckdb'      AS mcc;
SELECT 'halo_infinite' AS title, * FROM hi.match_participants WHERE xuid = ?
UNION ALL
SELECT 'halo_mcc'      AS title, * FROM mcc.match_participants WHERE xuid = ?;
```

**Règle architecturale du projet** : isolation par chemin FS, **pas par colonne**.

**2. Exception `xuid_aliases` : déplacement vers DB globale.**

Nouveau fichier : `data/global/xbox_aliases.duckdb` (chemin unique, **pas paramétré par titre**).

`PathResolver` gagne une méthode dédiée :

```go
// PathResolver
func (r *PathResolver) GlobalXuidAliasesDBPath() string  // data/global/xbox_aliases.duckdb
```

C'est la seule méthode du resolver qui ne prend pas de `titleSlug` en paramètre — elle marque sémantiquement le fait que les xuid sont globaux.

**3. Migration via `cmd/migrate-xuid-aliases-global`** (script one-shot) :

- Lire `xuid_aliases` de chaque `data/titles/{slug}/warehouse/shared_matches_v2.duckdb`
- Consolider dans `data/global/xbox_aliases.duckdb` avec dédup sur `xuid` (max `last_seen`)
- Drop les tables locales `xuid_aliases` après vérification + dry-run
- Idempotent (réexécutable sans corruption)
- Test sur fixture multi-titres (`synthetic_title_b` + `halo_infinite`)

**4. Refactor consommateurs** : tous les services qui lisent ou écrivent `xuid_aliases` (sync ingestion, `engagement_score_service`, lookups xuid → gamertag) migrent vers la DB globale.

## Consequences

### Positive

- Pas de migration de schéma sur les tables transverses (économie ~1 sem).
- Sémantique correcte : le xuid est global, pas game-dependent.
- Pas de duplication / risque de divergence sur les `xuid_aliases`.
- Cohérent avec la règle « isolation par chemin FS, pas par colonne ».

### Negative

- `PathResolver` gagne une singularité (méthode sans `titleSlug`) — nécessite documentation explicite.
- 1 fichier DB de plus à backuper (`data/global/xbox_aliases.duckdb`).
- Migration one-shot à exécuter sur toutes les instances (locale, staging, prod).
- Dégrade si on voulait à un moment passer à un schéma `title_id` colonne — il faudrait migrer les autres tables aussi (mais c'est précisément ce qu'on évite).

## Alternatives evaluated

| Alternative | Rejected because |
|---|---|
| **A) `title_id` colonne sur toutes les tables transverses** | Redondance avec path FS. Migration coûteuse pour aucun bénéfice immédiat. Risque bugs élevé. |
| **B) `xuid_aliases` dupliqué par titre (status quo)** | Sémantiquement incorrect, divergences possibles. |
| **C) `xuid_aliases` dans `metadata.duckdb`** | Mélange référentiels titre-spécifiques et identifiants globaux. Confusion. |
| **D) Service externe d'identité** | Surdimensionné pour le besoin. Pas d'autres backends à servir. |

## References

- Code review : `axe-2-multi-titres.md` (BLOQUANT schéma DuckDB transverse).
- Plan d'action : `PLAN_ACTION.md` P1.4, P5.
- Skill `db-schema` : `.claude/skills/db-schema/SKILL.md` (à mettre à jour avec ce nouveau chemin).
- PathResolver : `apps/go-api/internal/domain/title/registry.go`.

## Amendment 2026-11-01 — two roots

**Source** — decisions D-14 and D-15 of `.ai/V8/PLAN_DESKTOP_2026-11-01.md`, audit worksite C8, lot L11.
**Relates to** — ADR 0044 (deployment profile: the desktop profile is what sets the roots), ADR 0045 (the shell
launches the server with the three roots in its environment).

The decision above stands: isolation by file-system path, every path through `PathResolver`. What changes is the origin
of the paths. `PathResolver` is built from one root (`NewPathResolver(repoRoot)`) and each data path is
`filepath.Join(repoRoot, "data", ...)`; `config.RepoRoot` is read at some 190 sites of `internal/` outside the tests.
A checkout of the repository is therefore both the place the program reads its versioned resources (`config/`,
`static/`, the web build, the replay reference files) and the place it writes its state. A per-user Windows installation
(program under `%LOCALAPPDATA%\Programs\LevelUp`, state elsewhere, never deleted by the uninstaller) needs the two apart.
Two variables were already declared and read by nothing (`LEVELUP_ROOT`, `LEVELUP_DATA` in the Dockerfile and the
compose file); `LEVELUP_REPO_ROOT` and `LEVELUP_LOGS_DIR` already exist and keep working. The demo already separates the
two in its own way (`config_demo.go`: versioned configuration read from the repository, state under the demo root).

**A-1 — Three roots.**

- `ResourcesRoot`: what ships with the program, read-only by convention (not by ACL, D-14). Variable
  `LEVELUP_RESOURCES_DIR`, then `LEVELUP_REPO_ROOT` (kept as the compatible name), then the current detection
  (`config.autoDetectRepoRoot` for the server, `title.FindRepoRoot` for the offline tools; both walk upwards looking for
  checkout markers, which an installed tree does not need because the shell sets the variable).
- `DataRoot`: everything the program creates or modifies. Variable `LEVELUP_DATA_DIR`; default
  `<ResourcesRoot>/data`, which is today's `data/`.
- `LogsRoot`: journals. Variable `LEVELUP_LOGS_DIR` (existing); default as today, `<ResourcesRoot>/logs`.

Explicit variables are environment only (a `.env.local` cannot set the root it is read from). Resolution happens once, in
`internal/config`, and the roots are given to `PathResolver`; in the `desktop` profile the shell passes all three.

**A-2 — Which family lives under which root.**

| Root | Paths |
|---|---|
| `ResourcesRoot` | `config/` (manifests, `locales.toml`), `static/`, the web build (`dist`), `metadata-prebuilt.zip`, the replay reference files `data/titles/{slug}/reference/` |
| `DataRoot` | the warehouse and player databases (`titles/{slug}/...`, `global/xbox_aliases.duckdb`), `db_profiles.json`, `app_settings.json`, `.env.local`, `sessions/`, `auth/` (tokens, users, groups, `session_secret`), caches (film chunks and manifests, replay artifacts), `backups/migrations/` (`MigrationBackupsDir`), `desktop/port`, `levelup.lock` |
| `LogsRoot` | `server.log`, per-category journals, `server.crash.log`, `desktop.log` |

Resource paths keep their relative shape under `ResourcesRoot` (`data/titles/{slug}/reference/...`,
`data/titles/{slug}/warehouse/metadata-prebuilt.zip`), so that a shipped tree and a checkout look alike; the resolver
exposes them through resource accessors, not through `TitleDataDir`.

**A-3 — Default layout reproduces today's paths.** With `LEVELUP_DATA_DIR` unset, every path the resolver returns is
the path of before (witness T4: the sorted list of the paths of every exported method, identical with default roots), and
the three files that sit at the top of the checkout today (`db_profiles.json`, `app_settings.json`, `.env.local`) stay
there. With `LEVELUP_DATA_DIR` set, they live at the top of `DataRoot`.

**A-4 — Per-user installation (D-14).** Program and resources under `%LOCALAPPDATA%\Programs\LevelUp`, data under
`%LOCALAPPDATA%\LevelUp` (logs in its `logs/`), no elevation, so that the updater needs no UAC prompt. The uninstaller
never removes `DataRoot` (36 to 100 GB of films and databases on the maintainer's machine); the Tauri identifier
(`info.lvelup.desktop`) is frozen from P1 because it also names the WebView2 profile folder.

**A-5 — Invariants.** With distinct roots: no resource path is under `DataRoot`, no data path is under `ResourcesRoot`.
`RepoRoot` is no longer read outside `internal/config` and `PathResolver` (a dated allowlist, which only goes down, holds
the readers not yet migrated). The existing rule stands: no `filepath.Join(..., "data", ...)` by hand.

**A-6 — Child processes.** The replay build children (`replaychild`, `filmproc`) receive the three roots in their
environment, so that a child resolves the same paths as its parent, whatever its working directory.

**A-7 — Docker.** The image sets the three variables explicitly, the dead `LEVELUP_ROOT` and `LEVELUP_DATA` are removed
from the Dockerfile and the compose files, and the resources are mounted read-only. The data volume stays `./data`.

**Implementation notes (L11, 2026-11-01).**

- *Generated overlays are data.* `data/titles/{slug}/reference/generated/` (map weapon pads and callouts caught up by
  the runtime) is written while the server runs: it lives under `DataRoot`, with the same relative path, not under
  `ResourcesRoot`. Everything else under `reference/` stays a resource. This refines A-2.
- *Transitional bridge.* About a hundred services receive a bare root (`repoRoot string`) and build their own
  `PathResolver`. `internal/config` installs the process roots once (`title.InstallProcessRoots`);
  `NewPathResolver(root)` returns them when `root` is that resources root and the default layout otherwise. The bridge
  goes away when the allowlist of the A-5 ratchet (`archlint/repo_root_readers_test.go`) is empty.
- *Docker.* Setting `LEVELUP_DATA_DIR` moves `db_profiles.json` and `app_settings.json` to the top of `DataRoot` (A-3);
  the compose service keeps its host files mounted under `/app` through `LEVELUP_DB_PROFILES` and
  `LEVELUP_APP_SETTINGS`. The `reference/` directory is not mounted read-only on the production service, because
  `reference/generated/` shares it.
- *Children.* `filmproc.NewRunnerRoots` imposes the three roots (an empty root is removed from the child environment);
  the data root is imposed only when it was set explicitly.

**Consequences.** `PathResolver` gains a second root and resource accessors; the 41 hand-built joins of `"data"` and the
readers of `RepoRoot` move in three sub-lots (L11a table of path to root, L11b resources and data, L11c Docker and
deploy scripts), each proved by T4. The journals directory is created before any writer opens a file in it, which also
removes the lost crash capture on a fresh root (`crash_output` opens `server.crash.log` before `logs/` exists).
**Guard-rails** (to create at L11): a `PathResolver` test with distinct roots for the two invariants; the `RepoRoot`
ratchet in `internal/archlint/`; the existing `filepath.Join(..., "data", ...)` guard stays green.
