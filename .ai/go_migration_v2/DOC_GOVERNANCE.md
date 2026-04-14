# DOC_GOVERNANCE.md — Gouvernance du corpus Go v2

> Ce document définit la hiérarchie documentaire du chantier Python -> Go.
> Il organise un corpus v2 qui contient à la fois une couche d'entrée et des références exhaustives locales.

## Problème à corriger

Le corpus initial est solide mais son plan principal concentre trop de rôles à la fois :

1. charte stratégique ;
2. inventaire métier ;
3. catalogue technique ;
4. ordre d'exécution ;
5. suivi de projet ;
6. contraintes d'exploitation.

Quand un document joue tous ces rôles, il devient précis mais difficile à piloter.

## Nouvelle répartition des rôles

| Rôle | Document principal | Ce qui ne doit pas y vivre |
|------|--------------------|----------------------------|
| Point d'entrée | [README.md](README.md) | redites complètes de tout le corpus |
| Charte du programme | [PROGRAM_CHARTER.md](PROGRAM_CHARTER.md) | backlog vivant, matrices, tables exhaustives |
| Cible terminale zéro Python | [ZERO_PYTHON_TARGET.md](ZERO_PYTHON_TARGET.md) | détail module par module exhaustif |
| Référence de portage | [PORTING_REFERENCE.md](PORTING_REFERENCE.md) | statut quotidien des lots |
| Plan exhaustif du programme | [PLAN_MIGRATION_PYTHON_TO_GO_V2.md](PLAN_MIGRATION_PYTHON_TO_GO_V2.md) | micro-suivi quotidien |
| Stratégie zéro Python exhaustive | [ZERO_PYTHON_STRATEGY.md](ZERO_PYTHON_STRATEGY.md) | résumé très court de position |
| Couverture complète | [MATRIX.md](MATRIX.md) | suivi d'avancement détaillé |
| Compat runtime / ops | [OPS_COMPAT_CHECKLIST.md](OPS_COMPAT_CHECKLIST.md) | reformulation générale des phases |
| Ordre détaillé d'exécution | [SPRINT_ROADMAP.md](SPRINT_ROADMAP.md) | inventaires métier exhaustifs hors sprints |
| Suivi vivant | [GO_MIGRATION_CHECKLIST.md](GO_MIGRATION_CHECKLIST.md) | doctrine ou annexe technique longue |

## Règles anti-duplication

1. Une table de sprint détaillée ne doit vivre que dans [SPRINT_ROADMAP.md](SPRINT_ROADMAP.md) ou [GO_MIGRATION_CHECKLIST.md](GO_MIGRATION_CHECKLIST.md).
2. Une règle auth/jobs/runtime détaillée ne doit pas être maintenue en double dans [PROGRAM_CHARTER.md](PROGRAM_CHARTER.md).
3. Une matrice package/script/commande ne doit vivre qu'à un seul endroit : [MATRIX.md](MATRIX.md).
4. Les docs d'entrée ne doivent pas devenir des copies compactes ambiguës des références exhaustives.
5. La duplication v1 -> v2 est assumée uniquement pour rendre le corpus v2 autonome avec le même niveau de détail local.

## Règles de mise à jour

### Quand mettre à jour le corpus v2

1. si le cadre du programme change ;
2. si un gate change ;
3. si la hiérarchie documentaire devient floue ;
4. si une vue d'ensemble n'est plus alignée avec sa référence exhaustive locale.

### Quand mettre à jour le corpus original

1. seulement si l'on veut maintenir une archive d'origine réalignée ;
2. seulement si une divergence historique doit être documentée explicitement ;
3. par défaut, le corpus de travail prioritaire est désormais le v2.

## Politique de coexistence avec les originaux

Les originaux sont conservés comme archive de la première itération documentaire.

Le dossier v2 constitue désormais un corpus autosuffisant pour piloter le chantier.
Il sert à :

1. remettre une hiérarchie claire au-dessus du corpus actuel ;
2. conserver localement le même niveau de détail utile au pilotage ;
3. fournir son propre pilotage quotidien via une roadmap, une checklist, une matrice et une checklist ops v2 ;
4. conserver les originaux comme références historiques seulement.

## Stratégie de consolidation future

Si le chantier Go démarre réellement, la consolidation devrait suivre cet ordre :

1. garder [README.md](README.md), [PROGRAM_CHARTER.md](PROGRAM_CHARTER.md), [ZERO_PYTHON_TARGET.md](ZERO_PYTHON_TARGET.md) et [PORTING_REFERENCE.md](PORTING_REFERENCE.md) comme couche d'entrée ;
2. utiliser [PLAN_MIGRATION_PYTHON_TO_GO_V2.md](PLAN_MIGRATION_PYTHON_TO_GO_V2.md), [ZERO_PYTHON_STRATEGY.md](ZERO_PYTHON_STRATEGY.md), [MATRIX.md](MATRIX.md), [OPS_COMPAT_CHECKLIST.md](OPS_COMPAT_CHECKLIST.md), [SPRINT_ROADMAP.md](SPRINT_ROADMAP.md) et [GO_MIGRATION_CHECKLIST.md](GO_MIGRATION_CHECKLIST.md) comme références de travail détaillées ;
3. garder les originaux pour l'historique et la vérification, pas comme dépendances structurelles ;
4. ne jamais réintroduire une version condensée comme seule source de vérité.

## Règle simple

Si une information change tous les jours, elle n'appartient probablement pas à la charte.

Si une information est un invariant du programme, elle n'appartient probablement pas à la checklist vivante.
