# HANDOFF — H5 mécaniques de kill (LIVRÉ) + Weapon taxonomy registry (PLANIFIÉ) — 2026-06-23

> Doc de reprise post-compact. Branche : `integration/h5-x-livefetch` (worktree `levelup-multititre`).
> Runtime/données : `LevelUp-go-migration`. Push integration ≠ deploy (deploy part de `main`).

## 0. État en une page

| Sujet | État | Détail |
|---|---|---|
| **H5 mécaniques de kill** (assassinats + compétences spartiate : ground pound, shoulder bash) | **LIVRÉ + POUSSÉ** | 4 surfaces, capability-gated, tests verts. Commits `c081e1c36`→`fe9790496`. |
| **Cards cumulées** (assassinats + 2 compétences) | **LIVRÉ** | Sur Synthesis (1 card/mécanique, cumul scope). |
| **Précision PAR ARME** | **RÉSOLU (impossible)** | `WeaponStats[]` carnage H5 servi VIDE (sonde live `n=0`) ; aucune source tirs/arme. Substitut headshot-rate/arme = non fait (optionnel). |
| **Sweep shared tables `create_base vs add_*`** (note collègue) | **VÉRIFIÉ : déjà fait** | weapon_kills/match_csrs/player_match_enrichment résolus + guards. Rien à coder. |
| **Weapon taxonomy** (registre BDD) | **PLANIFIÉ + DATA VÉRIFIÉE, PAS construit** | Plan `.ai/PLAN_WEAPON_TAXONOMY.md` ; attend GO build (P2→P4). UI différée (narration TBD). |
| **Kill-feed decoder** | **À VIRER du projet (consigne d'origine, NON fait)** | « ça n'a rien à voir, c'est en pause ». Code = `feat/weapon-attribution-v3` worktree + `internal/analysis/weaponv3` + `cmd/diag_weapons_v3`. |
| **Backfill H5** (peupler les 3 colonnes sur matchs existants) | **AUTRE AGENT** | `cmd/h5-backfill` modifié en // (LEVELUP_H5_AUTH_AS). DOIT tourner AVEC mes commits (migration+mapper+persist) sinon colonnes restent 0. |
| **Land `integration`→`main`** | **EN ATTENTE (accord user)** | Auto-deploy prod. Branche ~280 commits derrière main. |

## 1. H5 mécaniques de kill — LIVRÉ (4 surfaces)

Données natives du carnage H5 (`TotalAssassinations`, `TotalGroundPoundKills`, `TotalShoulderBashKills` — confirmées par
sonde live `probe-h5`, distinctes de `TotalMeleeKills`). Capability **`native_kill_mechanics`** (registry Go + allowlist +
`config/titles/halo_5/title.toml` + miroir front `TITLE_CAPABILITIES`) → gate title-agnostic (invisible Infinite, fail-open).

| Surface | Ajout | Commit |
|---|---|---|
| Synthesis | Donut « répartition des frags » +3 slices + **cards cumulées** | `c081e1c36` |
| Match-view | Donut kill-types du viewer (ligne `is_me` scoreboard) | `88d2a9bda` |
| Timeseries | Donut sur le **1er onglet** (summary, pas distributions — consigne user) | `b0b16d065` |
| Escouade | **Barres empilées** par coéquipier (`LoadKillMechanics` GROUP BY xuid) | `fe9790496` |

- **Fondation** : DTO `H5CarnagePlayer` +3 ; `canonical.MatchParticipant` + `domain.MatchParticipantRow` +3 ; migration
  dédiée `add_h5_kill_mechanics_columns` (3 colonnes `match_participants`, ordonnée APRÈS create_base) ; persist + read-path
  `player_matches` (débloque Synthesis+Timeseries via `Self`) ; openapi + types + i18n FR/EN.
- **Fix au passage** : whitelist `notifications_title_ready` (test shared_social pré-existant cassé, MT-19).
- Tests : Go (migration/title/h5/canonical/persist/duckdb/service) + typecheck + eslint + vitest (synthesis/match-view/
  timeseries/squad). TOUS VERTS à chaque commit (ratchets pre-push OK).
- **CAVEAT** : colonnes peuplées seulement pour matchs RE-synchronisés → backfill requis (autre agent, cf. §0).

## 2. Weapon taxonomy registry — PLANIFIÉ (plan = `.ai/PLAN_WEAPON_TAXONOMY.md`)

Vision user : **registre d'armes canonique EN BDD = passage PRINCIPAL** (pas un TOML à côté). 3 tables append-only
(`weapons` + `weapon_ids` [N ids/arme : module/film_chunk/stock_id/filmshell] + `weapon_families`), 4 dimensions par arme
(**class** poing/épaule/lourde, **family** rôle cross-titre, **faction** humaine/covenant/forerunner/parias, **damage_type**)
+ `extra` JSON extensible (TTK & co plus tard). Resolver `(titre, id_kind, id) → weapon_key → tout`, migration anti-corruption
des lookups épars (`weapon_labels` + `weapon_data.go` + stock ids H5).

- **Table §6 du plan = VÉRIFIÉE** contre halopedia.org + wiki.halo.fr (4 agents, 2026-06-23, sources citées par arme).
  Corrections clés vs mémoire : Cindershot/Heatwave = **forerunner** (pas paria) ; « Diminisher of Hope » = Gravity Axe
  mêlée (PAS variante Skewer) ; Mk50 (pas Mk51) ; Fuel Rod SPNKr = banished ; faction = **origine** (covenant restent
  covenant même portées par Parias) ; FR officiels (Empaleur, Crémator, Calcineur, Ravageur, Disrupteur, Déchiqueteur…).
- **Phases** : P0 plan ✓ · P1 vérif ✓ · **P2 schéma+seed · P3 resolver+tests · P4 migration lecteurs** (à construire) ·
  ~~P5 UI~~ DIFFÉRÉ (narration/branchement non décidés par le user).
- **Prochain pas** : attendre le GO du user pour construire P2→P4 (registre + resolver + migration, SANS UI).

## 3. Sujets « du début » encore ouverts (à ne pas perdre)

1. **Kill-feed decoder = à VIRER du projet** (consigne d'origine, jamais faite). C'est `feat/weapon-attribution-v3`
   (worktree `.claude/worktrees/weapon-attribution-v3`) + `internal/analysis/weaponv3/` + `cmd/diag_weapons_v3/` +
   `weapon_kills_v3*`. « Rien à voir, en pause. » → décider : suppression du code/worktree, ou archivage. NON destructif
   sans confirmation explicite (gros code RE).
2. **Backfill H5** : autre agent (`cmd/h5-backfill` + `LEVELUP_H5_AUTH_AS`). Coordination : doit tourner avec les commits
   `c081e1c36`→`fe9790496` pour peupler assassination/ground_pound/shoulder_bash sur l'existant.
3. **Land `integration/h5-x-livefetch` → `main`** : gros merge (~280 commits derrière), auto-deploy prod → accord user requis.
4. **Sanity-check runtime** LUSR/combat (optionnel, non bloquant).

## 4. Contraintes permanentes (rappel)
Répondre FR ; jamais re-capture tokens ; pas de Python (Go-only) ; pas d'emojis dans les fichiers versionnés ; jamais
git stash ; prévenir avant op prod/VPS ; demander avant commit ; push main = auto-deploy ; clé `LEVELUP_HALOAPI_KEY`
jamais committée. Skills `.claude/skills/{delivery-checklist, arch-rules, …}` avant tout commit.
