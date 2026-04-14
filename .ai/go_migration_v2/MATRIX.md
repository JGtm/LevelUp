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
| `a_remplacer` | disparait en tant que code Python et est remplace par une implementation Go plus adaptee |
| `a_porter_plus_tard` | porte apres les surfaces plus critiques |
| `a_analyser` | surface mixte dont le destin final doit etre tranche entre portage, remplacement ou suppression |
| `a_auditer` | surface auxiliaire a inventorier finement avant decision de portage |
| `a_adapter` | enveloppe, packaging ou scripts a realigner sur la cible Go, sans portage 1:1 |
| `a_conserver` | reste en place telle quelle pendant et apres la migration Go |
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
   Statut : `a_remplacer`
   Strategie : supprimer la dependance Python et la remplacer directement par un client HTTP Go natif.
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
| `apps/api/` | ~65 | ~12 000 | `a_remplacer` | `go-api/internal/api/` | Haute |
| `src/data/repositories/` | ~15 | ~4000 | `a_porter` | `go-api/internal/platform/duckdb/` | Haute |
| `src/data/services/` | ~8 | ~1500 | `a_porter` | `go-api/internal/{domain}/` | Moyenne |
| `src/data/sync/` (dont `transformers/` ~2 400 LOC) | ~46 | ~13 000 | `a_porter_plus_tard` | `go-api/internal/sync/` | Tres haute |
| `src/data/migration/` | ~40 | ~2000 | `a_porter` | `go-api/internal/platform/migrations/` | Haute |
| `src/analysis/` | ~62 | ~14 000 | `a_porter` | `go-api/internal/analysis/` | Tres haute |
| `src/auth/` | 5 | ~900 | `a_porter` | `go-api/internal/auth/` | Haute |
| `src/app/` | ~25 | ~2000 | `a_analyser` | Majoritairement supprime (Streamlit) | Basse |
| `src/ports/` | 2 | ~200 | Interfaces de reference | Interfaces Go equivalentes | Basse |
| `src/config.py` | 1 | ~150 | `a_porter` | `go-api/internal/platform/config/` | Basse |
| `src/visualization/` | 47 | ~12 000 | `a_porter` (charting server-side) | `go-api/internal/domain/chart/` + `service/charts/` | Tres haute |
| `src/ui/components/` (radars, KPI, annotations) | 13 | ~1 200 | `a_porter` partiellement | `go-api/internal/domain/chart/` + `api/dto/` | Moyenne |
| `src/ui/` (hors components/) | ~27 | ~4 800 | `a_supprimer` | N/A | N/A |
| `src/ai/` | ~6 | ~1200 | `hors_scope` | Reste Python ou separe | N/A |
| `src/utils/` (Discord + helpers runtime utiles) | ~4 | ~600 | `a_auditer` | `go-api/internal/platform/` | Basse |
| `scripts/` | ~25 | ~3000 | `a_porter_plus_tard` | `go-api/cmd/` | Haute |
| `spnkr_pr/` | ~8 | ~800 | `a_remplacer` | Client HTTP Go direct | Moyenne |
| `launcher.py` | 1 | ~500 | `a_adapter` | `go-api/cmd/levelup/` | Moyenne |

**Ordre de grandeur** : ~55 000 LOC Python à porter (dont ~12K LOC `src/visualization/` charting server-side, auparavant classé "N/A") → ~45-65 000 LOC Go estimés.

> **Note** : les estimations précédentes (~25K LOC) sous-estimaient sévèrement `src/analysis/` (14K, pas 3K),
> `apps/api/` (12K, pas 3K), `src/data/sync/` (13K, pas 6-7K) et omettaient `src/visualization/` (12K).
> Chiffres vérifiés le 2026-04-14.

## Scripts et outillage

| Script Python | Usage | Portage Go | Priorite |
|---------------|-------|------------|:--------:|
| `scripts/sync.py` | Sync delta/full | `levelup sync` | P4 |
| `scripts/backfill_data.py` | Backfill selectif (~120 flags) | `levelup backfill` | P4 |
| `scripts/backup_player.py` | Backup DB joueur | `levelup backup` | P3 |
| `scripts/restore_player.py` | Restore DB joueur | `levelup restore` | P3 |
| `scripts/healthcheck_db.py` | Diagnostic integrite | `levelup healthcheck` | P3 |
| `scripts/index_media.py` | Indexation videos | `levelup index-media` | P3 |
| `scripts/check_env.py` | Validation environnement | `levelup check-env` | P2 |
| `scripts/diagnose_player_db.py` | Debug schemas | `levelup diagnose` | P3 |
| `scripts/post_sync_compute.py` | Post-sync pipeline | Integre dans sync | P4 |
| `scripts/archive_season.py` | Archivage Parquet | `levelup archive` | P4 |
| `scripts/populate_*.py` | Seed metadata | `levelup seed` | P2 |
| `launcher.py` | Orchestrateur principal | `levelup api` | P1 |

## Backfill bitmask : valeurs a reproduire exactement

Le systeme de bitmask est persiste en DB. Le portage Go doit reprendre exactement les memes valeurs, sans "equivalent" approximatif. La source de verite est double :

1. `BACKFILL_FLAGS` historiques dans `src/data/sync/migrations.py` pour les bits 0-15 et un bit 18 legacy obsolet.
2. `MatchBits` dans `src/data/sync/constants.py` pour les bits 16-22 effectivement utilises en production au niveau match.

| Bit | Source | Champ | Valeur | Note |
|-----|--------|-------|--------|------|
| 0 | `BACKFILL_FLAGS` | medals | 1 | historique |
| 1 | `BACKFILL_FLAGS` | events | 2 | historique |
| 2 | `BACKFILL_FLAGS` | skill | 4 | historique |
| 3 | `BACKFILL_FLAGS` | personal_scores | 8 | historique |
| 5 | `BACKFILL_FLAGS` | accuracy | 32 | bit 4 absent intentionnellement |
| 6 | `BACKFILL_FLAGS` | shots | 64 | historique |
| 7 | `BACKFILL_FLAGS` | enemy_mmr | 128 | historique |
| 8 | `BACKFILL_FLAGS` | assets | 256 | historique, distinct de `MatchBits.ASSETS` |
| 9 | `BACKFILL_FLAGS` | participants | 512 | historique |
| 10 | `BACKFILL_FLAGS` | participants_scores | 1024 | historique |
| 11 | `BACKFILL_FLAGS` | participants_kda | 2048 | historique |
| 12 | `BACKFILL_FLAGS` | participants_shots | 4096 | historique |
| 13 | `BACKFILL_FLAGS` | participants_damage | 8192 | historique |
| 14 | `BACKFILL_FLAGS` | aliases | 16384 | historique, distinct de `MatchBits.ALIASES` |
| 15 | `BACKFILL_FLAGS` | participants_avg_life | 32768 | historique |
| 16 | `MatchBits` | events_loaded | 65536 | `highlight_events` charges |
| 17 | `MatchBits` | assets_loaded | 131072 | metadonnees match resolues |
| 18 | `MatchBits` | aliases_loaded | 262144 | `xuid_aliases` extraits |
| 19 | `MatchBits` | killer_victim_loaded | 524288 | global |
| 20 | `MatchBits` | pve_stats | 1048576 | tentative PvE effectuee |
| 21 | `MatchBits` | weapon_kills | 2097152 | source de verite moderne |
| 22 | `MatchBits` | weapon_kills_no_film | 4194304 | chunks indisponibles |

**Attention** :

- le bit 4 reste absent intentionnellement ;
- `migrations.py` conserve un ancien `weapon_kills = 1 << 18` uniquement pour retrocompatibilite des tests ; ce bit legacy ne doit jamais etre re-ecrit comme s'il etait la source de verite moderne ;
- les bits 16, 17 et 18 existent bel et bien en production via `MatchBits`.

## Regle de maintenance

Avant de toucher une nouvelle surface :

1. l'ajouter ici ;
2. lui donner un statut explicite ;
3. lier la surface a son impact dans `OPS_COMPAT_CHECKLIST.md` si elle touche auth, jobs, DEMO_MODE, packaging ou runbook.
