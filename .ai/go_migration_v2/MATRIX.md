# MATRIX.md - Couverture Python -> Go

> [!WARNING]
> Ne pas utiliser ce document seul.
> Lire aussi [PLAN_MIGRATION_PYTHON_TO_GO_V2.md](PLAN_MIGRATION_PYTHON_TO_GO_V2.md), [OPS_COMPAT_CHECKLIST.md](OPS_COMPAT_CHECKLIST.md) et [GO_MIGRATION_CHECKLIST.md](GO_MIGRATION_CHECKLIST.md).

## Role du document

Cette matrice porte la couverture technique du chantier Go :

1. quels packages et scripts existent vraiment ;
2. quel est leur statut de couverture ;
3. ce qui est hors scope, supprime, garde temporairement ou porte ;
4. quelles surfaces ne doivent jamais etre touchees sans decision explicite.

L'avancement vivant du chantier ne se suit pas ici mais dans [GO_MIGRATION_CHECKLIST.md](GO_MIGRATION_CHECKLIST.md). La matrice dit quoi couvrir ; la checklist dit ou on en est reellement.

## Statuts de couverture a utiliser

| Statut | Sens |
|--------|------|
| `a_porter` | doit avoir un equivalent Go explicite |
| `a_porter_plus_tard` | porte apres les surfaces plus critiques |
| `a_garder_temporairement` | reste Python pendant une phase transitoire assume |
| `a_supprimer` | disparait avec la fin de Streamlit / du legacy |
| `hors_scope` | n'entre pas dans le chantier Go principal |
| `sorti_du_scope_go` | comportement reel du repo, mais explicitement exclu du programme principal |

## Statuts d'avancement a utiliser

Ces statuts d'avancement sont suivis dans [GO_MIGRATION_CHECKLIST.md](GO_MIGRATION_CHECKLIST.md), pas dans la colonne de couverture de cette matrice.

| Statut | Sens |
|--------|------|
| `non_demarre` | lot identifie mais non ouvert |
| `cadre` | contrat, corpus et gate de sortie definis |
| `en_cours` | implementation active |
| `en_verif_parite` | verification Python vs Go en cours |
| `pret_integration` | parite utile demontree, lot pret pour revue ou bascule controlee |
| `termine` | gate passe, suivi a jour, lot cloture |
| `bloque` | prerequis manquant ou dependance non resolue |

## Regle simple

Si un fichier, un package, une commande ou un comportement n'apparait pas ici, il est considere comme non couvert. Il doit etre ajoute avant toute modification profonde.

## Surfaces prioritaires deja visibles

1. `apps/api/`
   Statut : `a_porter`
   Strategie : port direct des routes et schemas utiles au frontend React.
2. `src/data/repositories/`
   Statut : `a_porter`
   Strategie : extraire les requetes critiques d'abord, SQL-first quand possible.
3. `src/data/services/`
   Statut : `a_porter`
   Strategie : porter par parcours metier, pas en bloc.
4. `src/analysis/`
   Statut : `a_porter`
   Strategie : ne porter que ce qui ne vit pas proprement en SQL DuckDB.
5. `src/auth/`
   Statut : `a_porter`
   Strategie : MSAL canonique + support refresh tokens de compatibilite.
6. `spnkr_pr/`
   Statut : `a_porter_plus_tard`
   Strategie : anti-corruption layer, puis remplacement progressif par client HTTP Go.
7. `src/app/`
   Statut : `a_analyser`
   Strategie : distribuer entre suppression legacy et portage des rares logiques encore actives.
8. `src/utils/` (partiellement)
   Statut : `a_auditer`
   Strategie : porter les helpers d'exploitation et retirer les helpers purement UI.

## Surfaces a traiter tard ou a remplacer plutot qu'a porter

1. `src/data/sync/`
   Statut : `a_porter_plus_tard`
   Strategie : une fois read-only, settings, auth et jobs stabilises.
2. `scripts/sync.py` et `scripts/backfill_data.py`
   Statut : `a_porter_plus_tard`
   Strategie : repartir des usages reels, pas d'une traduction ligne a ligne.
3. `scripts/backup_player.py`, `scripts/restore_player.py`, `scripts/check_env.py`, `scripts/diagnose_player_db.py`
   Statut : `a_porter_plus_tard`
   Strategie : prioriser ce qui est necessaire a l'exploitation et au support.
4. `launcher.py`, `LevelUp.sh`, `LevelUp.bat`, `packaging/`
   Statut : `a_adapter`
   Strategie : converger vers les nouveaux binaires Go et le frontend conserve.

## Surfaces a supprimer plutot qu'a porter

1. `src/ui/`, `streamlit_app.py`, `streamlit_app_v7.py`
   Statut : `a_supprimer`
   Strategie : retirer via la migration React, pas reimplementer en Go.
2. `apps/web/`
   Statut : `a_conserver`
   Strategie : stabiliser les contrats, pas changer de pile web.
3. `tests/parity/`, fixtures, golden values
   Statut : `a_conserver`
   Strategie : enrichir si besoin, jamais traiter comme du legacy a supprimer trop tot.

## Surfaces explicitement sorties du scope Go principal

| Surface | Statut | Decision |
|---------|--------|----------|
| `src/app/media_watcher.py` | `sorti_du_scope_go` | watcher Linux inotify non bloquant ; a traiter a part ou a supprimer, pas un prerequis du portage backend |
| `src/utils/tailscale.py` | `sorti_du_scope_go` | helper d'exposition Tailscale ; non critique pour la parite metier |
| `src/app/media_background.py` (scheduler / watcher) | `sorti_du_scope_go` | le coeur index-media peut etre porte plus tard, le scheduling legacy ne bloque pas le programme Go |
| `src/ai/` | `hors_scope` | outillage developpeur, pas logique produit Halo |

## Matrice detaillee Python -> Go

| Package | Fichiers | LOC approx | Statut | Cible Go | Difficulte |
|---------|:--------:|:----------:|--------|----------|:----------:|
| `apps/api/` | ~30 | ~3000 | A remplacer | `go-api/internal/api/` | Moyenne |
| `src/data/repositories/` | ~15 | ~4000 | A porter | `go-api/internal/platform/duckdb/` | Haute |
| `src/data/services/` | ~8 | ~1500 | A porter | `go-api/internal/{domain}/` | Moyenne |
| `src/data/sync/` | ~20 | ~6000-7000 | A porter (P4) | `go-api/internal/sync/` | Tres haute |
| `src/data/migration/` | ~40 | ~2000 | A porter | `go-api/internal/platform/migrations/` | Haute |
| `src/analysis/` | ~12 | ~3000 | A porter | `go-api/internal/analysis/` | Tres haute |
| `src/auth/` | ~6 | ~1000 | A porter | `go-api/internal/auth/` | Haute |
| `src/app/` | ~25 | ~2000 | A analyser + distribuer | Majoritairement supprime (Streamlit) | Basse |
| `src/ports/` | 2 | ~200 | Interfaces de reference | Interfaces Go equivalentes | Basse |
| `src/config.py` | 1 | ~150 | A porter | `go-api/internal/platform/config/` | Basse |
| `src/ui/` | ~40 | ~6000 | A supprimer (React) | N/A | N/A |
| `src/ai/` | ~6 | ~1200 | Hors scope | Reste Python ou separe | N/A |
| `src/utils/` (Discord + helpers runtime utiles) | ~4 | ~600 | A porter partiellement | `go-api/internal/platform/` | Basse |
| `scripts/` | ~25 | ~3000 | A reconstituer | `go-api/cmd/` | Haute |
| `spnkr_pr/` | ~8 | ~800 | A remplacer | Client HTTP Go direct | Moyenne |
| `launcher.py` | 1 | ~500 | A remplacer | `go-api/cmd/levelup-api/` | Moyenne |

**Ordre de grandeur** : ~25 000 LOC Python -> ~25-35 000 LOC Go.

## Scripts et outillage

| Script Python | Usage | Portage Go | Priorite |
|---------------|-------|------------|:--------:|
| `scripts/sync.py` | Sync delta/full | `cmd/levelup-sync/` | P4 |
| `scripts/backfill_data.py` | Backfill selectif (~120 flags) | `cmd/levelup-sync/ --backfill` | P4 |
| `scripts/backup_player.py` | Backup DB joueur | `cmd/levelup-tools backup` | P3 |
| `scripts/restore_player.py` | Restore DB joueur | `cmd/levelup-tools restore` | P3 |
| `scripts/healthcheck_db.py` | Diagnostic integrite | `cmd/levelup-tools healthcheck` | P3 |
| `scripts/index_media.py` | Indexation videos | `cmd/levelup-tools index-media` | P3 |
| `scripts/check_env.py` | Validation environnement | `cmd/levelup-tools check-env` | P2 |
| `scripts/diagnose_player_db.py` | Debug schemas | `cmd/levelup-tools diagnose` | P3 |
| `scripts/post_sync_compute.py` | Post-sync pipeline | Integre dans sync | P4 |
| `scripts/archive_season.py` | Archivage Parquet | `cmd/levelup-tools archive` | P4 |
| `scripts/populate_*.py` | Seed metadata | `cmd/levelup-tools seed` | P2 |
| `launcher.py` | Orchestrateur principal | `cmd/levelup-api/` | P1 |

## Backfill bitmask : valeurs a reproduire exactement

Le systeme de bitmask est persiste en DB. Le portage Go doit reprendre exactement les memes valeurs, sans "equivalent" approximatif.

```go
const (
    BackfillMedals           = 1 << 0   // 1
    BackfillEvents           = 1 << 1   // 2
    BackfillSkill            = 1 << 2   // 4
    BackfillPersonalScores   = 1 << 3   // 8
    // Bit 4 intentionnellement absent
    BackfillAccuracy         = 1 << 5   // 32
    BackfillShots            = 1 << 6   // 64
    BackfillEnemyMmr         = 1 << 7   // 128
    BackfillAssets           = 1 << 8   // 256
    BackfillParticipants     = 1 << 9   // 512
)
```

Valeurs critiques deja identifiees :

| Bit | Champ | Valeur |
|-----|-------|--------|
| 0 | medals | 1 |
| 1 | events | 2 |
| 2 | skill | 4 |
| 3 | personal_scores | 8 |
| 5 | accuracy | 32 |
| 6 | shots | 64 |
| 7 | enemy_mmr | 128 |
| 8 | assets | 256 |
| 9 | participants | 512 |
| 10 | participants_scores | 1024 |
| 11 | participants_kda | 2048 |
| 12 | participants_shots | 4096 |
| 13 | participants_damage | 8192 |
| 14 | aliases | 16384 |
| 15 | participants_avg_life | 32768 |
| 19 | killer_victim | 524288 |
| 20 | pve_stats | 1048576 |
| 21 | weapon_kills | 2097152 |
| 22 | weapon_kills_no_film | 4194304 |

**Attention** : les bits 4, 16, 17, 18 sont absents intentionnellement.

## Regle de maintenance

Avant de toucher une nouvelle surface :

1. l'ajouter ici ;
2. lui donner un statut explicite ;
3. lier la surface a son impact dans `OPS_COMPAT_CHECKLIST.md` si elle touche auth, jobs, DEMO_MODE, packaging ou runbook.
