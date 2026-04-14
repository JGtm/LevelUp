# Go Migration Master — Corpus restructuré et exhaustif local

> Point d'entrée du corpus documentaire restructuré pour le chantier Python -> Go.
> Les documents originaux restent intacts dans [../go_migration](../go_migration).

## Pourquoi ce dossier existe

Le corpus initial est riche mais le plan principal est devenu trop large pour jouer seul le rôle de document maître.

Ce dossier introduit une hiérarchie plus lisible sans perdre le niveau de détail d'origine :

1. un point d'entrée court ;
2. une charte de programme stable ;
3. une référence technique de parité ;
4. des copies exhaustives locales des documents détaillés utiles au pilotage.

## État global

| Champ | Valeur |
|-------|--------|
| Statut programme | `cadre` |
| Lot actif | aucun |
| Prochain gate utile | formaliser le prérequis 0 puis lancer le Sprint 0 |
| Corpus détaillé de travail | ce dossier |
| Archive d'origine conservée | [../go_migration](../go_migration) |

## Lecture recommandée

### Vue d'ensemble rapide

1. [PROGRAM_CHARTER.md](PROGRAM_CHARTER.md)
2. [DOC_GOVERNANCE.md](DOC_GOVERNANCE.md)
3. [ZERO_PYTHON_TARGET.md](ZERO_PYTHON_TARGET.md)
4. [PORTING_REFERENCE.md](PORTING_REFERENCE.md)
5. [HALO_CANONICAL_MODEL.md](HALO_CANONICAL_MODEL.md)
6. [HALO_INFINITE_CAPABILITY_MAP.md](HALO_INFINITE_CAPABILITY_MAP.md)
7. [HALO_BOOTSTRAP_CONTRACT.md](HALO_BOOTSTRAP_CONTRACT.md)
8. [HALO_GO_TYPE_BLUEPRINT.md](HALO_GO_TYPE_BLUEPRINT.md)

### Références exhaustives locales

1. [PLAN_MIGRATION_PYTHON_TO_GO_V2.md](PLAN_MIGRATION_PYTHON_TO_GO_V2.md)
2. [ZERO_PYTHON_STRATEGY.md](ZERO_PYTHON_STRATEGY.md)
3. [MATRIX.md](MATRIX.md)
4. [OPS_COMPAT_CHECKLIST.md](OPS_COMPAT_CHECKLIST.md)
5. [SPRINT_ROADMAP.md](SPRINT_ROADMAP.md)
6. [GO_MIGRATION_CHECKLIST.md](GO_MIGRATION_CHECKLIST.md)

## Sources historiques utiles

Les fichiers originaux sont conservés comme archive de la première itération documentaire.

1. [../go_migration/PLAN_MIGRATION_PYTHON_TO_GO.md](../go_migration/PLAN_MIGRATION_PYTHON_TO_GO.md)
2. [../go_migration/ZERO_PYTHON_STRATEGY.md](../go_migration/ZERO_PYTHON_STRATEGY.md)
3. [../go_migration/SPRINT_ROADMAP.md](../go_migration/SPRINT_ROADMAP.md)
4. [../go_migration/GO_MIGRATION_CHECKLIST.md](../go_migration/GO_MIGRATION_CHECKLIST.md)

## Source de vérité par sujet

| Sujet | Vue d'ensemble | Référence exhaustive locale |
|-------|----------------|----------------------------|
| Vision, périmètre, méthode, phases, gates | [PROGRAM_CHARTER.md](PROGRAM_CHARTER.md) | [PLAN_MIGRATION_PYTHON_TO_GO_V2.md](PLAN_MIGRATION_PYTHON_TO_GO_V2.md) |
| Gouvernance documentaire | [DOC_GOVERNANCE.md](DOC_GOVERNANCE.md) | ce dossier |
| Cible finale zéro Python | [ZERO_PYTHON_TARGET.md](ZERO_PYTHON_TARGET.md) | [ZERO_PYTHON_STRATEGY.md](ZERO_PYTHON_STRATEGY.md) |
| Inventaire métier et oracles de parité | [PORTING_REFERENCE.md](PORTING_REFERENCE.md) | [PLAN_MIGRATION_PYTHON_TO_GO_V2.md](PLAN_MIGRATION_PYTHON_TO_GO_V2.md) |
| Modèle canonique Halo | [HALO_CANONICAL_MODEL.md](HALO_CANONICAL_MODEL.md) | [HALO_CANONICAL_MODEL.md](HALO_CANONICAL_MODEL.md) |
| Capability map Halo Infinite | [HALO_INFINITE_CAPABILITY_MAP.md](HALO_INFINITE_CAPABILITY_MAP.md) | [HALO_INFINITE_CAPABILITY_MAP.md](HALO_INFINITE_CAPABILITY_MAP.md) |
| Contrat bootstrap Halo | [HALO_BOOTSTRAP_CONTRACT.md](HALO_BOOTSTRAP_CONTRACT.md) | [HALO_BOOTSTRAP_CONTRACT.md](HALO_BOOTSTRAP_CONTRACT.md) |
| Blueprint types Go canoniques | [HALO_GO_TYPE_BLUEPRINT.md](HALO_GO_TYPE_BLUEPRINT.md) | [HALO_GO_TYPE_BLUEPRINT.md](HALO_GO_TYPE_BLUEPRINT.md) |
| Couverture package/script/commande | [MATRIX.md](MATRIX.md) | [MATRIX.md](MATRIX.md) |
| Compat runtime, auth, jobs, packaging | [OPS_COMPAT_CHECKLIST.md](OPS_COMPAT_CHECKLIST.md) | [OPS_COMPAT_CHECKLIST.md](OPS_COMPAT_CHECKLIST.md) |
| Ordre d'exécution détaillé par sprint | [SPRINT_ROADMAP.md](SPRINT_ROADMAP.md) | [SPRINT_ROADMAP.md](SPRINT_ROADMAP.md) |
| Avancement vivant et preuves attendues | [GO_MIGRATION_CHECKLIST.md](GO_MIGRATION_CHECKLIST.md) | [GO_MIGRATION_CHECKLIST.md](GO_MIGRATION_CHECKLIST.md) |

## Sujets spécialisés à ne pas perdre de vue

| Sujet | Où regarder d'abord | Pourquoi c'est critique |
|-------|----------------------|-------------------------|
| Mode de travail en worktree et discipline pré-merge | [PROGRAM_CHARTER.md](PROGRAM_CHARTER.md) | évite de confondre vitesse locale de refactor et qualité d'intégration |
| Déploiement cible, runbook et migration utilisateur | [OPS_COMPAT_CHECKLIST.md](OPS_COMPAT_CHECKLIST.md) | packaging, sessions, cache MSAL, compat utilisateur existante |
| Multi-joueurs et pools par gamertag | [PORTING_REFERENCE.md](PORTING_REFERENCE.md) | isole correctement les DB player et les write leases |
| Couverture métier complémentaire : i18n, PvE, bitmask, Discord, media indexing | [PORTING_REFERENCE.md](PORTING_REFERENCE.md) | évite un portage concentré uniquement sur les écrans P1 |
| Architecture API multi-titre Halo | [PORTING_REFERENCE.md](PORTING_REFERENCE.md) | garde une façade produit stable si un nouveau jeu Halo remplace Infinite |
| Modèle canonique Halo | [HALO_CANONICAL_MODEL.md](HALO_CANONICAL_MODEL.md) | fixe la frontière entre provider, produit et analytics métier |
| Capability map `halo_infinite` | [HALO_INFINITE_CAPABILITY_MAP.md](HALO_INFINITE_CAPABILITY_MAP.md) | rend explicites les surfaces supportées, dégradées ou absentes |
| Contrat bootstrap Halo | [HALO_BOOTSTRAP_CONTRACT.md](HALO_BOOTSTRAP_CONTRACT.md) | fixe ce que le consommateur doit savoir au démarrage, sans exposer la mécanique 343i |
| Blueprint types Go | [HALO_GO_TYPE_BLUEPRINT.md](HALO_GO_TYPE_BLUEPRINT.md) | prépare les structs et interfaces cibles sans coder trop tôt |
| Zéro Python et extinction des bridges | [ZERO_PYTHON_TARGET.md](ZERO_PYTHON_TARGET.md) | garde la cible finale visible pendant tout le programme |
| Arbitrages de pilotage solo et kill switches | [PROGRAM_CHARTER.md](PROGRAM_CHARTER.md) | évite de transformer le chantier en tunnel sans preuve ni seuil d'arrêt |

## Routine d'ouverture d'un lot

1. Vérifier qu'une ligne existe dans [MATRIX.md](MATRIX.md) et dans [GO_MIGRATION_CHECKLIST.md](GO_MIGRATION_CHECKLIST.md).
2. Identifier le contrat observable, l'oracle Python et le corpus de vérification.
3. Vérifier [OPS_COMPAT_CHECKLIST.md](OPS_COMPAT_CHECKLIST.md) si le lot touche auth, jobs, runtime, packaging ou migration utilisateur.
4. Si le lot touche l'intégration Halo, vérifier aussi le modèle canonique, la matrice de capabilities par titre et la politique de dégradation dans [PORTING_REFERENCE.md](PORTING_REFERENCE.md).
5. N'ouvrir un lot qu'avec un gate de sortie explicite.
6. Reporter toute décision structurante dans [../thought_log.md](../thought_log.md).

## Prochaines actions concrètes

1. Rendre explicite le prérequis 0 : corpus de référence Go gelé, contrats, golden values, matrice.
2. Détailler le mapping documentaire `titles/haloinfinite -> canonical` pour préparer le futur provider sans écrire encore de code.
3. Détailler l'adaptateur `canonical -> bootstrap/OpenAPI` pour éviter que les handlers Go n'inventent leur propre contrat.
4. Exécuter le Sprint 0 comme filtre de faisabilité, pas comme début de production.
5. Ne lancer aucun lot de Phase 1 tant que le résultat du Sprint 0 n'est pas capturé noir sur blanc.

## Règle de maintenance

Le dossier v2 assume désormais deux couches distinctes.

La règle est simple :

1. les vues d'ensemble vivent dans [PROGRAM_CHARTER.md](PROGRAM_CHARTER.md), [PORTING_REFERENCE.md](PORTING_REFERENCE.md) et [ZERO_PYTHON_TARGET.md](ZERO_PYTHON_TARGET.md) ;
2. les références exhaustives locales vivent dans [PLAN_MIGRATION_PYTHON_TO_GO_V2.md](PLAN_MIGRATION_PYTHON_TO_GO_V2.md), [ZERO_PYTHON_STRATEGY.md](ZERO_PYTHON_STRATEGY.md), [MATRIX.md](MATRIX.md), [OPS_COMPAT_CHECKLIST.md](OPS_COMPAT_CHECKLIST.md), [SPRINT_ROADMAP.md](SPRINT_ROADMAP.md) et [GO_MIGRATION_CHECKLIST.md](GO_MIGRATION_CHECKLIST.md) ;
3. l'archive d'origine reste conservée dans [../go_migration](../go_migration), mais ne doit plus être nécessaire pour travailler ;
4. si une information change les invariants du programme, la vue d'ensemble et la référence exhaustive locale doivent être réalignées ensemble.
