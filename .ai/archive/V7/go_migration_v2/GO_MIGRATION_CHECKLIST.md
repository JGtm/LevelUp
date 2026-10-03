# GO_MIGRATION_CHECKLIST.md - Suivi vivant du chantier Go

> [!CAUTION]
> **DÉPRÉCIÉ depuis le Sprint 49 (2026-07-25).**
> Ce document est obsolète et n'est plus maintenu. Le suivi du chantier Go
> est assuré par [`SPRINT_ROADMAP.md`](SPRINT_ROADMAP.md).
> Les informations ci-dessous reflètent un état antérieur (Phase 1-5) et ne
> correspondent plus à la réalité du projet (Phase 11 atteinte).

> [!WARNING]
> Ne pas utiliser ce document seul.
> Lire aussi [PLAN_MIGRATION_PYTHON_TO_GO_V2.md](PLAN_MIGRATION_PYTHON_TO_GO_V2.md), [MATRIX.md](MATRIX.md), [OPS_COMPAT_CHECKLIST.md](OPS_COMPAT_CHECKLIST.md) et [ZERO_PYTHON_STRATEGY.md](ZERO_PYTHON_STRATEGY.md).

## Role du document

Cette checklist suit l'avancement reel du chantier Go :

1. quels lots sont ouverts ;
2. leur statut courant ;
3. la preuve attendue pour les faire avancer ;
4. les blocages a lever avant la phase suivante.

## Regles d'usage

1. Un lot actif ou planifie = une ligne visible dans ce document.
2. Toute evolution de statut doit mettre a jour la ligne du lot le jour meme.
3. Aucun lot ne passe en `en_cours` sans reference de contrat, corpus ou preuve attendue.
4. Aucun lot d'une phase N+1 ne passe en `en_cours` tant que le gate de la phase N n'est pas valide, sauf exploration ou POC explicitement signales.
5. Aucun morceau Python ne doit etre supprime si le lot Go correspondant n'est pas marque `termine`.

## Statuts a utiliser

| Statut | Sens |
|--------|------|
| `non_demarre` | lot identifie mais non ouvert |
| `cadre` | contrat, corpus et gate de sortie definis |
| `en_cours` | implementation active |
| `en_verif_parite` | verification Python vs Go en cours |
| `pret_integration` | parite utile demontree, lot pret pour revue ou bascule controlee |
| `termine` | gate passe, suivi a jour, lot cloture |
| `bloque` | prerequis manquant ou dependance non resolue |

## Etat global

| Champ | Valeur |
|-------|--------|
| Statut programme | `en_cours` |
| Lot actif | Phase 1 partielle — Sprint 5 (repositories read-only + pool DuckDB) à ouvrir |
| Derniere mise a jour | 2026-04-15 |
| Journal technique | [../thought_log.md](../thought_log.md) |

## Decision d'arret documentaire

1. Le prerequis 0 documentaire est considere comme gele.
2. Aucun nouveau document n'est requis avant le Sprint 0, sauf ecart bloquant constate en spike.
3. Les prochains changements attendus concernent des preuves techniques, pas un nouveau cycle de planification abstraite.

## Backlog ordonne minimal

| Ordre | Lot | Phase | Statut | Preuve attendue | Derniere mise a jour | Blocage / prochaine action |
|------:|-----|-------|--------|-----------------|----------------------|----------------------------|
| 0 | Prerequis 0 : corpus documentaire Go gele | Prerequis | `termine` | charte + contrats Halo + taxonomie erreurs + OpenAPI MVP P0/P1 + matrice documentes | 2026-04-14 | Ouvrir le Sprint 0 ; plus de doc prealable generale |
| 1 | Phase 0.0 : modele canonique Halo + capability map par titre | Phase 0 | `termine` | contrat canonique + capability map + bootstrap + mapping/adapters materialises | 2026-04-14 | Lot documentaire clos ; reference a utiliser telle quelle |
| 2 | Sprint 0 : POC DuckDB, HTTP, MSAL | Sprint 0 | `pret_integration` | DuckDB Go + HTTP chi + MSAL device code flow validés Windows — `/health`, `/api/v1/bootstrap`, `/api/v1/players` cohérents avec Python — toolchain ucrt64 documentée dans Makefile — build `go build ./...` 0 erreur | 2026-04-15 | Lot clos. Ouvrir Phase 0.2 corpus golden values (lot 4) |
| 3 | Phase 0.1 : freeze OpenAPI MVP et taxonomie d'erreurs | Phase 0 | `pret_integration` | `apps/go-api/api/openapi.yaml` — 14 endpoints P0/P1, tous schémas dérivés de Python, commité Sprint 1 | 2026-04-15 | Prêt — oapi-codegen l'utilisera en Sprint 4 |
| 4 | Phase 0.2 : corpus golden values complet | Phase 0 | `pret_integration` | 10 fixtures JSON schema-conformant + `capture.py` + README dans `tests/fixtures/golden_values/` | 2026-04-15 | Prêt — à régénérer via capture.py avant Sprint 6 |
| 4b | Phase 0.3 : baselines de performance | Phase 0 | `pret_integration` | `tests/fixtures/baselines.json` — 8 endpoints mesurés p50/p95/p99, script `benchmark_python_api.py` | 2026-04-15 | Prêt — à remesurer sur API prod avant Sprint 7 |
| 4c | Sprint 4 : squelette HTTP + middleware + oapi-codegen | Phase 1 | `pret_integration` | CORS, rate-limit, slog, oapi-codegen types générés, CI go-build Windows+Linux, `go build ./...` 0 erreur | 2026-04-15 | Prêt — Sprint 5 peut ouvrir |
| 5 | Phase 1 (S04-S07) : squelette HTTP + repositories read-only + parite | Phase 1 | `non_demarre` | service Go runnable + requetes critiques sous test + 0 ecart | 2026-04-14 | Dependance : lots 3-4 |
| 6 | Phase 2 (S08-S13) : parcours read-only complets | Phase 2 | `non_demarre` | tous parcours read-only en parite utile | 2026-04-14 | Dependance : gate phase 1 |
| 7 | Phase 3 (S14-S17) : auth, session, settings, jobs persistants | Phase 3 | `non_demarre` | onboarding complet sans Python | 2026-04-14 | Dependance : gate phase 2 |
| 8 | Phase 4 (S18-S25) : sync, backfill, migrations, CLI, media | Phase 4 | `non_demarre` | cycle complet sync/backfill equivalent Python | 2026-04-14 | Dependance : gate phase 3 |
| 9 | Phase 5 (S26-S28) : bascule et extinction Python | Phase 5 | `non_demarre` | 3 cycles reels clean + runbook prod sans Python | 2026-04-14 | Dependance : gate phase 4 |

## Detail des lots par sprint (a derouler a l'ouverture de chaque phase)

Les lots ci-dessus sont a gros grain. Au moment de l'ouverture de chaque phase, les decomposer en lots Sprint dans ce tableau :

| Lot | Sprint | Statut | Preuve attendue | Derniere MAJ |
|-----|--------|--------|-----------------|--------------|
| Phase 0.0 : modele canonique Halo + capability map produit | Phase 0 | `termine` | contrat canonique + capability map versionnes | 2026-04-14 |
| S04 : squelette HTTP + config + middleware | Phase 1 | `pret_integration` | CORS/rate-limit/slog branchés, oapi-codegen types générés, CI GitHub Actions, `go vet/build` OK | 2026-04-15 |
| S05 : repositories read-only + pool DuckDB | Phase 1 | `non_demarre` | requetes Q1-Q16 sous test golden values | 2026-04-14 |
| S06 : bootstrap, players, filters, career, history | Phase 1 | `non_demarre` | endpoints fonctionnels et compares | 2026-04-14 |
| S07 : validation parite Phase 1 | Phase 1 | `non_demarre` | 0 ecart non justifie sur corpus | 2026-04-14 |
| S08 : Explorer + Match View + killer/victim | Phase 2 | `non_demarre` | parcours explorer en parite | 2026-04-14 |
| S09 : Sessions (algorithme 6, 2 modes) | Phase 2 | `non_demarre` | decoupe sessions identique au Python | 2026-04-14 |
| S10 : Stats/Series + perf score + LUSR | Phase 2 | `non_demarre` | 5 onglets × 2 modes en parite | 2026-04-14 |
| S11 : Accueil/Home read-only + socle provider Halo | Phase 2 | `non_demarre` | hero card + provider Halo sur fixtures + dégradation pré-auth explicite | 2026-04-14 |
| S12 : Escouade + Synthese | Phase 2 | `non_demarre` | 13 sous-modules en parite | 2026-04-14 |
| S13 : Citations + Medias | Phase 2 | `non_demarre` | galerie + citations en parite | 2026-04-14 |
| S14 : session/cookies | Phase 3 | `non_demarre` | sessions persistantes fonctionnelles | 2026-04-14 |
| S15 : device code flow + MSAL Go | Phase 3 | `non_demarre` | auth complete sans Python | 2026-04-14 |
| S16 : settings/setup | Phase 3 | `non_demarre` | GET/PATCH settings, smoke test | 2026-04-14 |
| S17 : jobs longs persistants | Phase 3 | `non_demarre` | jobs persistes au redemarrage | 2026-04-14 |
| S18 : moteur sync minimal (12 mixins, ~13K LOC) | Phase 4 | `non_demarre` | delta sync fonctionnel | 2026-04-14 |
| S19 : pipeline post-sync | Phase 4 | `non_demarre` | perf score + LUSR + mv refresh | 2026-04-14 |
| S20 : backfill complet (96 champs, ~120 args CLI) | Phase 4 | `non_demarre` | backfill identique au Python | 2026-04-14 |
| S21 : migrations DuckDB (35 steps) | Phase 4 | `non_demarre` | idempotence et auto-apply OK | 2026-04-14 |
| S22 : weapon parsing | Phase 4 | `non_demarre` | parser binaire ou bridge valide | 2026-04-14 |
| S23 : PvE Firefight | Phase 4 | `non_demarre` | sync PvE fonctionnel | 2026-04-14 |
| S24 : scripts d'exploitation | Phase 4 | `non_demarre` | backup/restore/healthcheck Go | 2026-04-14 |
| S25 : notifications Discord | Phase 4 | `non_demarre` | embeds post-sync fonctionnels | 2026-04-14 |

## Blocages ouverts

1. Le Sprint 0 n'a pas encore valide DuckDB Go, HTTP minimal et MSAL sur le terrain.
2. Le corpus golden values de parite n'est pas encore centralise pour les surfaces prioritaires.
3. La strategie finale de port direct du client Halo ou de bridge transitoire etroit reste a prouver en spike.
4. Les references documentaires sont desormais suffisantes ; la suite doit etre pilotee par preuve technique et non par nouveau cadrage general.

## Regles de maintenance

1. Mettre a jour ce fichier avant l'ouverture d'un lot.
2. Changer le statut des lots des qu'une preuve concrete est obtenue ou qu'un blocage apparait.
3. Reporter toute decision technique structurante dans [../thought_log.md](../thought_log.md).
4. Ne marquer un lot `termine` que si la preuve de parite ou de sortie est referencable.

## Checklist de cloture d'un lot

- [ ] Le lot est reference dans `MATRIX.md`
- [ ] Le statut d'avancement est a jour dans ce document
- [ ] Le gate de sortie est ecrit et verifiable
- [ ] La preuve de parite ou l'ecart volontaire est documente
- [ ] `OPS_COMPAT_CHECKLIST.md` est mis a jour si auth/jobs/runtime/package sont touches
- [ ] `../thought_log.md` est mis a jour si une decision technique a change le plan
- [ ] Le destin du morceau Python remplace est explicite : garde temporaire, bridge etroit, suppression
