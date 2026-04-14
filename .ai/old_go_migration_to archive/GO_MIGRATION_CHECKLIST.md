# GO_MIGRATION_CHECKLIST.md - Suivi vivant du chantier Go

> [!WARNING]
> Ne pas utiliser ce document seul.
> Lire aussi [PLAN_MIGRATION_PYTHON_TO_GO.md](PLAN_MIGRATION_PYTHON_TO_GO.md), [MATRIX.md](MATRIX.md), [OPS_COMPAT_CHECKLIST.md](OPS_COMPAT_CHECKLIST.md) et [ZERO_PYTHON_STRATEGY.md](ZERO_PYTHON_STRATEGY.md).

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
| Statut programme | `cadre` |
| Lot actif | Aucun |
| Derniere mise a jour | 2026-04-14 |
| Journal technique | [../thought_log.md](../thought_log.md) |

## Backlog ordonne minimal

| Ordre | Lot | Phase | Statut | Preuve attendue | Derniere mise a jour | Blocage / prochaine action |
|------:|-----|-------|--------|-----------------|----------------------|----------------------------|
| 0 | Prerequis : corpus de reference Go gele | Prerequis | `non_demarre` | facade web cible + contrats + golden values + matrice geles | 2026-04-14 | Formaliser le corpus de depart avant tout lot technique |
| 1 | Phase 0.0 : modele canonique Halo + capability map par titre | Phase 0 | `non_demarre` | contrat canonique + matrice de capabilities versionnes | 2026-04-14 | A cadrer avant toute vraie implementation provider |
| 2 | Sprint 0 : POC DuckDB, HTTP, MSAL | Sprint 0 | `non_demarre` | POC Windows/Linux valide + JSON bootstrap coherent | 2026-04-14 | Demarrer une fois le prerequis 0 explicite |
| 3 | Phase 0.1 : freeze OpenAPI et contrats de reference | Phase 0 | `non_demarre` | schema versionne + contrats documentes | 2026-04-14 | Dependance : lots 1-2 |
| 4 | Phase 0.2 : corpus golden values complet | Phase 0 | `non_demarre` | corpus rejouable couvrant surfaces prioritaires | 2026-04-14 | Dependance : lot 3 |
| 5 | Phase 1.1-1.2 : squelette HTTP + repositories read-only | Phase 1 | `non_demarre` | service Go runnable + requetes critiques sous test | 2026-04-14 | Dependance : lots 3-4 |
| 6 | Phase 1.3-1.4 : bootstrap, players, filters + verif parite | Phase 1 | `non_demarre` | 0 ecart non justifie sur le corpus Phase 1 | 2026-04-14 | Dependance : lot 5 |
| 7 | Phase 2 : Career, History, Explorer, Match View | Phase 2 | `non_demarre` | parcours read-only prioritaires en parite utile | 2026-04-14 | Dependance : gate phase 1 |
| 8 | Phase 3 : auth, session, settings, jobs persistants | Phase 3 | `non_demarre` | onboarding complet sans Python | 2026-04-14 | Dependance : gate phase 2 |
| 9 | Phase 4 : sync, backfill, migrations, CLI, media | Phase 4 | `non_demarre` | cycle complet sync/backfill equivalent Python | 2026-04-14 | Dependance : gate phase 3 |
| 10 | Phase 5 : bascule et extinction Python | Phase 5 | `non_demarre` | 3 cycles reels clean + runbook prod sans Python | 2026-04-14 | Dependance : gate phase 4 |

## Detail des lots par sprint (a derouler a l'ouverture de chaque phase)

Les lots ci-dessus sont a gros grain. Au moment de l'ouverture de chaque phase, les decomposer en lots Sprint dans ce tableau :

| Lot | Sprint | Statut | Preuve attendue | Derniere MAJ |
|-----|--------|--------|-----------------|--------------|
| Phase 0.0 : modele canonique Halo + capability map produit | Phase 0 | `non_demarre` | contrat canonique + capability map versionnes | 2026-04-14 |
| Sprint 1.1 : squelette HTTP + config + middleware | Phase 1 | `non_demarre` | service Go runnable, `/health` OK | 2026-04-14 |
| Sprint 1.2 : repositories read-only + pool DuckDB | Phase 1 | `non_demarre` | requetes Q1-Q16 sous test golden values | 2026-04-14 |
| Sprint 1.3 : bootstrap, players, filters, career, history | Phase 1 | `non_demarre` | endpoints fonctionnels et compares | 2026-04-14 |
| Sprint 1.4 : validation parite Phase 1 | Phase 1 | `non_demarre` | 0 ecart non justifie sur corpus | 2026-04-14 |
| Sprint 2.1 : Explorer + Match View + killer/victim | Phase 2 | `non_demarre` | parcours explorer en parite | 2026-04-14 |
| Sprint 2.2 : Stats/Series + sessions + perf score | Phase 2 | `non_demarre` | 5 onglets × 2 modes en parite | 2026-04-14 |
| Sprint 2.3 : Accueil/Home + socle provider Halo | Phase 2 | `non_demarre` | hero card + provider Halo live fonctionnels | 2026-04-14 |
| Sprint 2.4 : Escouade + Synthese | Phase 2 | `non_demarre` | 13 sous-modules en parite | 2026-04-14 |
| Sprint 2.5 : Citations + Medias | Phase 2 | `non_demarre` | galerie + citations en parite | 2026-04-14 |
| Sprint 3.1 : session/cookies | Phase 3 | `non_demarre` | sessions persistantes fonctionnelles | 2026-04-14 |
| Sprint 3.2 : device code flow + MSAL Go | Phase 3 | `non_demarre` | auth complete sans Python | 2026-04-14 |
| Sprint 3.3 : settings/setup | Phase 3 | `non_demarre` | GET/PATCH settings, smoke test | 2026-04-14 |
| Sprint 3.4 : jobs longs persistants | Phase 3 | `non_demarre` | jobs persistes au redemarrage | 2026-04-14 |
| Sprint 4.1 : moteur sync minimal (11 mixins) | Phase 4 | `non_demarre` | delta sync fonctionnel | 2026-04-14 |
| Sprint 4.2 : pipeline post-sync | Phase 4 | `non_demarre` | perf score + LUSR + mv refresh | 2026-04-14 |
| Sprint 4.3 : backfill complet (94 champs, ~120 args CLI) | Phase 4 | `non_demarre` | backfill identique au Python | 2026-04-14 |
| Sprint 4.4 : migrations DuckDB (35 steps) | Phase 4 | `non_demarre` | idempotence et auto-apply OK | 2026-04-14 |
| Sprint 4.5 : weapon parsing | Phase 4 | `non_demarre` | parser binaire ou bridge valide | 2026-04-14 |
| Sprint 4.6 : PvE Firefight | Phase 4 | `non_demarre` | sync PvE fonctionnel | 2026-04-14 |
| Sprint 4.7 : scripts d'exploitation | Phase 4 | `non_demarre` | backup/restore/healthcheck Go | 2026-04-14 |
| Sprint 4.8 : notifications Discord | Phase 4 | `non_demarre` | embeds post-sync fonctionnels | 2026-04-14 |

## Blocages ouverts

1. Aucun lot Go n'est ouvert tant que les preuves de faisabilite du Sprint 0 ne sont pas capturees.
2. Le modele canonique Halo et la capability map par titre ne sont pas encore figes dans le corpus de cadrage.
3. Le corpus de parite n'est pas encore centralise dans ce dossier.
4. Les references documentaires ont ete isolees ici, mais les statuts doivent etre maintenus a partir du premier lot reel.

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
