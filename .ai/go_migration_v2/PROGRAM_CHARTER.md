# PROGRAM_CHARTER.md — Charte du programme Python -> Go

> Ce document fixe le cadre stable du chantier.
> Il ne remplace pas les documents de détail ; il évite que le plan maître redevienne un document totalisant.

## Objectif

Remplacer progressivement le runtime backend Python de LevelUp par Go, sans rouvrir un chantier produit parallèle et sans casser les invariants métier actuels :

1. DuckDB v6 comme source de vérité ;
2. façade web actuelle comme consommateur contractuel de référence ;
3. golden values et corpus de parité comme oracle de validation ;
4. extinction de Python seulement en fin de programme.

## Cible terminale zéro Python

La cible finale du programme est détaillée dans [ZERO_PYTHON_TARGET.md](ZERO_PYTHON_TARGET.md).

Les invariants minimaux sont les suivants :

1. aucun runtime Python dans le chemin produit ;
2. aucun fichier `.py` exécuté par le produit final ;
3. aucun `pip` ou `venv` requis pour l'installation du produit ;
4. aucun bridge Python dans le chemin produit final ;
5. `src/ai/` hors scope du packaging et du runtime produit.

## Ce que le programme couvre

1. API backend aujourd'hui portée par Python.
2. Services métier et orchestration backend.
3. Repositories et accès DuckDB.
4. Auth, sessions, jobs longs.
5. Sync, backfill, migrations, scripts d'exploitation, media indexing.

## Ce que le programme ne couvre pas

1. Refonte produit parallèle.
2. Refonte gratuite des payloads et contrats HTTP.
3. Refonte DuckDB en même temps que le changement de runtime.
4. Réécriture de la façade web comme sous-projet du portage Go.

## Invariants non négociables

1. Pas de big bang.
2. Parité avant bascule.
3. Read-only avant auth et jobs ; auth et jobs avant sync/backfill.
4. SQL-first quand c'est viable, Go explicite quand SQL ne suffit pas.
5. Aucun morceau Python supprimé tant que son équivalent Go n'a pas passé son gate.

## Méthode

### Principes de portage

1. `contract-first` : figer d'abord la sortie observable.
2. `python-as-oracle` : Python sert de référence de comportement, pas de blueprint ligne à ligne.
3. `read-only-first` : commencer par les surfaces déterministes et visibles.
4. `stateful-last` : porter en dernier auth, sessions, jobs, sync et backfill.
5. `sql-first` : déplacer les calculs implicites Python vers SQL quand cela clarifie la vérité métier.
6. `parity-before-switch` : aucune surface n'est basculée sans corpus de vérification.
7. `one-way-replacement` : coexistence transitoire pendant l'intégration, pas double backend canonique durable.

### Anti-patterns interdits

1. Traduire un module Python complexe ligne à ligne.
2. Reproduire des coercions implicites non documentées.
3. Refaire Polars en Go à l'intuition.
4. Changer contrat JSON, logique métier et implémentation dans la même tranche.
5. Commencer par sync ou auth sans avoir validé le socle read-only.

## Architecture cible minimale

Le corpus v2 ne remplace pas le détail d'architecture existant, mais il fixe les garde-fous suivants.

> [!IMPORTANT]
> L'architecture logicielle formelle (hexagonale, couches, interfaces Go, injection de dépendances,
> pare-feu linter) est définie dans [GO_ARCHITECTURE_RULES.md](GO_ARCHITECTURE_RULES.md).
> Ce document-ci fixe les principes produit ; GO_ARCHITECTURE_RULES fixe les contraintes de code.
> Les deux sont contraignants.

1. Le service Go expose des contrats stables pour la façade web existante ; il ne redéfinit pas le produit.
2. Les accès DuckDB passent par un composant central avec pool read-only borné et write lease explicite.
3. Les jobs longs ont un modèle explicite : démarrage, statut, résultat, reprise après redémarrage si nécessaire.
4. Les migrations DuckDB restent idempotentes et automatiques.
5. Le service doit supporter un graceful shutdown propre : drain HTTP, fermeture des connexions et aucune interruption d'un sync en pleine écriture.
6. Le packaging cible reste orienté vers un binaire principal à sous-commandes, avec build Windows et Linux reproductible.
7. La façade API reste orientée parcours produit ; elle ne reflète jamais directement les endpoints d'un titre Halo particulier.
8. L'intégration Halo est structurée en deux niveaux : un socle provider générique, puis un adaptateur par titre mappant vers un modèle canonique LevelUp.
9. Les zones spécifiques au jeu restent isolées : auth Waypoint/XSTS, registre d'endpoints, refdata, films, skill, assets et constantes gameplay.
10. Le modèle canonique Halo doit être défini hors de toute couche d'orchestration sync legacy, pour pouvoir accueillir plusieurs titres sans héritage implicite du pipeline Python actuel.
11. Les capabilities produit par titre doivent être explicites, versionnées et consultables avant tout lot touchant la façade API.

## Préparation documentaire immédiate du chantier multi-titre

Avant d'écrire la première ligne de Go spécifique au provider Halo, le corpus doit contenir :

1. un modèle canonique Halo produit : identité joueur, historique de matchs, match détaillé, progression carrière, assets, films et erreurs ;
2. une matrice de capabilities par titre et par surface produit ;
3. un registre des zones spécifiques au jeu à isoler ;
4. une politique de dégradation des surfaces absentes ou incomplètes.

## Découpage du programme

| Étape | But | Sortie attendue |
|------|-----|------------------|
| Prérequis 0 | geler la référence de départ | contrats, corpus, golden values et matrice gelés |
| Sprint 0 | tester la faisabilité technique | DuckDB Go + HTTP + MSAL validés ou plan stoppé |
| Phase 0 | cadrer et figer la référence | OpenAPI, golden values, DoD, baselines |
| Phase 1 | prouver le socle Go read-only | bootstrap, players, filters, career, history en parité |
| Phase 2 | porter les parcours read-only prioritaires | explorer, match view, stats, home, citations, médias |
| Phase 3 | porter les chemins stateful intermédiaires | auth, sessions, settings, jobs persistants |
| Phase 4 | porter les traitements les plus risqués | sync, backfill, migrations, CLI, media, Discord |
| Phase 5 | basculer et éteindre Python | cycles réels clean, runbook Go autonome |

## Gates du programme

### Gate 1 — Faisabilité technique minimale

Passage autorisé seulement si :

1. DuckDB Go est valide sur les OS cibles.
2. Le chemin read-only a une parité démontrable.
3. La stratégie de pool, de lock et de toolchain Windows est documentée.

### Gate 2 — Viabilité produit

Passage autorisé seulement si :

1. Career, History, Explorer et Match View tournent en Go avec parité utile.
2. La façade V7 consomme les contrats Go sans changement de sémantique.
3. Les écarts restants sont documentés et volontairement assumés.

### Gate 3 — Viabilité d'exploitation

Passage autorisé seulement si :

1. Settings, sessions, auth et jobs tournent sans Python dans le chemin nominal.
2. L'observabilité et les procédures d'échec sont testées.
3. Le build et le déploiement sont reproductibles.

### Jalon ZP — Runtime produit sans Python

Passage requis avant l'entrée en Phase 5 :

1. aucun `.py` n'est exécuté dans le chemin produit ;
2. aucun `pip install` ou bootstrap `venv` n'est requis dans le packaging final ;
3. le client Go Halo est validé (tous endpoints testés sur fixtures, 3 cycles sync clean) ;
4. `src/ai/` est explicitement exclu du build et du packaging produit.

### Gate 4 — Extinction Python

Passage autorisé seulement si :

1. Sync, backfill, migrations DuckDB et scripts critiques ont leur équivalent Go.
2. Des cycles réels complets ont été observés sans régression majeure.
3. Python n'est plus requis par le runbook de prod.

## Mode de travail en worktree

Le programme assume un worktree dédié pour les gros refactors, mais cette liberté a des limites explicites.

### Ce qui peut être cassé localement pendant un lot

1. l'organisation interne des packages Go ;
2. une couche complète de repositories ou de services en cours de remplacement ;
3. des scripts internes ou points d'entrée temporaires tant qu'ils ne sont pas la preuve finale du lot.

### Ce qui doit revenir propre avant revue, merge ou bascule

1. les chemins critiques touchant schéma DuckDB et migrations ;
2. l'auth, les secrets, les cookies et caches de tokens ;
3. les endpoints déjà consommés par la façade V7 ;
4. sync, backfill, backup, restore, smoke tests et media indexing.

### Discipline minimale à respecter

1. Un lot peut casser localement, mais il doit redevenir runnable ou testable avant de changer de statut.
2. Les golden values, les tests de parité et les scripts minimaux utiles à la revue doivent être rebranchés avant intégration.
3. Aucun lot ne doit devenir structurellement opaque sous prétexte qu'il est développé dans un worktree séparé.

## Notes de pilotage solo

Le chantier peut être allégé pour un seul développeur, mais pas vidé de ses garde-fous.

### Ce qui peut être simplifié

1. le shadow mode complet peut être remplacé par des comparaisons Python versus Go ciblées sur un corpus représentatif ;
2. les soak tests très longs peuvent être remplacés par quelques cycles réels propres ;
3. la bureaucratie documentaire quotidienne doit rester minimale tant que les preuves et les décisions sont traçables.

### Ce qui reste non négociable

1. le Sprint 0 de faisabilité ;
2. les golden values et les tests de parité ;
3. la discipline de branche et de worktree ;
4. la preuve avant suppression d'un morceau Python ;
5. les kill switches si le coût du portage dépasse clairement sa valeur.

Les opportunités spécifiques de Go, comme la distribution simplifiée, la concurrence native ou les outils de détection de race, sont des bonus à capturer pendant le portage, pas une justification suffisante pour lancer le programme sans gates.

## Décisions structurantes gelées

1. packaging : binaire unique `levelup` à sous-commandes ;
2. contrats API : schema-first à partir de la surface Python existante ;
3. sessions : fichiers JSON + cookie signé HMAC-SHA256, sans JWT ;
4. auth : MSAL Go natif + support refresh tokens de compatibilité ;
5. charting : séparation stricte entre `ChartPayload` renderer-agnostic et rendu frontend.

## Points de pilotage restants

1. weapon parser : portage natif Go ou fallback subprocess étroit si échec ;
2. stratégie ATTACH avec `database/sql` au Sprint 0 ;
3. feature flags de bascule et observabilité minimale.

## Critères d'abandon

Le programme doit être stoppé, pas seulement ralenti, si l'un de ces cas devient réel sans contournement raisonnable :

1. le Sprint 0 échoue sur DuckDB Go ou le toolchain Windows ;
2. la Phase 1 dépasse massivement son estimation sans produire de parcours read-only crédibles ;
3. l'API Halo change au point d'invalider le contrat cible ;
4. le driver DuckDB Go n'a plus de voie de maintenance crédible ;
5. le produit Python dérive plus vite que le corpus de parité ;
6. le coût humain du portage solo dépasse clairement la valeur attendue.

## Documents détaillés à consulter ensuite

1. [PORTING_REFERENCE.md](PORTING_REFERENCE.md) pour l'inventaire métier et la parité.
2. [ZERO_PYTHON_TARGET.md](ZERO_PYTHON_TARGET.md) pour la cible terminale du produit.
3. [SPRINT_ROADMAP.md](SPRINT_ROADMAP.md) pour l'ordre détaillé d'exécution.
4. [GO_MIGRATION_CHECKLIST.md](GO_MIGRATION_CHECKLIST.md) pour le suivi vivant.
5. [MATRIX.md](MATRIX.md) pour la couverture complète des surfaces.
6. [OPS_COMPAT_CHECKLIST.md](OPS_COMPAT_CHECKLIST.md) pour auth, jobs, runtime et packaging.
