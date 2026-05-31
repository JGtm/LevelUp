# Audit — écritures shared en post-sync (RO bug) + inventaire complet

> Phase 0 + Phase 1bis du plan `les-matchs-d-hier`. Date : 2026-05-31.

## Symptôme confirmé (données réelles)
Matchs d'hier (`3e9967f6…`, `4cb4a8d0…`) : `highlight_events=0`, `killer_victim_pairs=0`,
`weapon_kills=0`, `events_loaded=false`, `backfill_completed=0`, `map_name`=GUID. Tableaux OK
(participants/medals/MMR présents). → graphiques de combat + citations vides.

## Cause racine (Phase 0)
Logs `sync.log`, à **chaque** cycle :
```
InsertHighlightEvents: Cannot execute statement of type "INSERT" on database
"shared_matches_v2" which is attached in read-only mode!
```
+ même erreur sur `SkillV2Repo.UpsertState` (LUSR v2 shadow).

Le film est téléchargé + parsé (le heal trouve le film) : **seule l'écriture échoue**. Donc la connexion
`sharedDB` utilisée par la complétion post-sync a `shared_matches_v2` ouvert **read-only** au moment de
l'INSERT (message DuckDB = base catalogue en read-only).

Divergence architecturale :
- **Chemin primaire** : `collect.go` → `persist.BatchBuilder` → `SharedPersister` via writer RW du
  `SharedDBProvider` → écrit highlight_events + killer_victim. **Marche** (matchs antérieurs complets).
- **Complétion post-sync** : `events_heal.go` → `ProcessHighlightEvents` → `InsertHighlightEvents`
  (`writes.go:406`) en `db.Exec` **direct**, sur le `sharedDB` transmis qui n'est **pas garanti RW** au
  moment de l'exécution. `events_heal.go` n'apparaît PAS dans la liste des appelants de
  `acquireSharedWriter` (contrairement à `backfill_weapons.go`, `events_replay.go`, `citations_backfill.go`).

Nuance à confirmer en impl (instrumentation runtime) : le weapon-heal réussit parfois (`healed:4`) avec le
MÊME `sharedDB` que le events-heal qui échoue dans le même run → l'état RO/RW du handle post-sync est
**instable** (B-swap ADR 0016 : ré-acquisition writer après drain async `engine.go` ~480-505, et/ou
re-swap RO concurrent par un reader pool). Le fix ne dépend PAS de trancher cette nuance.

## Fix (robuste, indépendant de la nuance)
Router TOUTES les écritures shared de complétion via `persist` sur un **writer RW tenu** pour toute la
durée de l'opération (même mécanisme que le chemin primaire qui, lui, réussit). Règle absolue : zéro
`db.Exec` direct sur table shared hors package `persist`.

## Inventaire exhaustif des écritures shared hors `persist` (Phase 1bis)
Tables shared = `shared_matches_v2` : match_registry, match_participants, medals_earned,
highlight_events, weapon_kills, killer_victim_pairs, xuid_aliases, match_csrs (+ LUSR v2 si shared).
(`match_skill_rank`, `player_csr_snapshots` = player DB → hors scope shared.)

Sites trouvés (`internal/sync/**`, hors *_test.go) — à migrer/valider :
- `writes.go:41` INSERT match_registry (primaire — déjà via persister ? vérifier)
- `writes.go:116` INSERT match_participants
- `writes.go:193` INSERT OR IGNORE medals_earned
- `writes.go:217` INSERT xuid_aliases
- `writes.go:320` INSERT weapon_kills
- `writes.go:357/494/507/553/566` UPDATE match_registry / match_participants (bitmasks, scores)
- `writes.go:411` INSERT OR IGNORE highlight_events  ← **cassé RO en post-sync**
- `writes.go:473` INSERT killer_victim_pairs        ← **cassé RO en post-sync**
- `events_replay.go:225` UPDATE match_registry (events_loaded reset)
- `engagement.go:478` UPDATE match_registry (match_intensity)
- `pve.go:306` UPDATE match_registry
- `csr_shared_writes.go:124` INSERT match_csrs
- `backfill_registry_names.go:142` UPDATE match_registry (noms metadata)
- LUSR v2 : `platform/duckdb/skill_v2_repo.go` UpsertState (shared ?) ← **cassé RO**

> NB : la plupart des `writes.go` sont appelés par le chemin LEGACY `insertFetchedMatch` (non-batch) et/ou
> par les heals. Le chemin batch primaire passe déjà par persist. Le scope critique = supprimer les
> écritures shared exécutées sur un handle non garanti RW (complétion/heal), pas casser le primaire.

## Décision writer RW post-sync
Le post-sync tient déjà un writer (ré-acquis après drain, `engine.go` ~497). Le dblease est
**non-réentrant** → NE PAS ré-acquérir dans les heals. Faire en sorte que les heals reçoivent et utilisent
le handle RW déjà tenu (le persister doit écrire sur CE handle), et garantir qu'il est bien RW (assert).
