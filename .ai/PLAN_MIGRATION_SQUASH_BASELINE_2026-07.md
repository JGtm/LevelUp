# PLAN — Squash des migrations : baseline bit-identique prouvée (chantier N4)

> Date : 2026-07-10. Auteur : Fable (supervision). Exécutant prévu : Opus.
> Origine : politique N4 documentée dans `internal/migration/doc.go` (2026-07-05,
> PROPOSITION) + exigence utilisateur 2026-07-10 : « une solution propre qui garantit
> AUCUNE perte ». Ce plan livre exactement ça : l'OUTILLAGE et la PREUVE d'abord (test
> d'invariant bit-identique), le squash lui-même en dernier, derrière un GO humain.
>
> PRINCIPE CENTRAL (à garder en tête à chaque phase) : les migrations sont des RECETTES
> de schéma, pas des données. Un squash ne touche AUCUNE donnée (matchs, joueurs, DBs
> existantes — les DBs déjà provisionnées ne rejouent pas les steps passés). La seule
> chose remplacée : ce qu'une DB VIERGE rejoue au boot. La garantie zéro-perte = un test
> qui prouve que (baseline + steps restants) produit un schéma BIT-IDENTIQUE à
> (historique complet). Tant que ce test n'est pas vert, RIEN n'est committé.
>
> **Contrat d'exécution : skill `plan-execution`, intégralement** (ordre strict, statuts
> [x]/[~]/[!], vérifier sur pièces, zéro fix hors périmètre → §Découvertes, thought_log +
> MAJ de CE fichier par phase). Reprise : relire ce header, puis §J, première case non statuée.
>
> **Branche** : `refactor/migration-squash-baseline` (nouvelle, depuis main APRÈS le merge
> de la campagne d'audits). JAMAIS main direct (deploy auto).

## 1. Objectif et critères de succès

Objectif : donner au projet la CAPACITÉ de squasher ses migrations (~100+ steps cumulés)
avec une garantie mécanique zéro-perte, puis exécuter le premier squash si (et seulement
si) l'opérateur donne le GO — conformément à la politique N4 (déclenchement MANUEL).

Critères de succès :
- Un outil de **snapshot de schéma normalisé** existe et est déterministe (2 runs → même
  sortie octet pour octet).
- Un **test d'invariant** compare schéma(historique complet) vs schéma(baseline + reste)
  et échoue au moindre écart — prouvé mordant DANS LES DEUX SENS avant tout squash.
- Les steps squashés sont **archivés** (`.ai/migrations/squashed/<version>/`), jamais perdus.
- Une DB vierge boote plus vite (mesure avant/après consignée) ; les DBs existantes ne
  voient AUCUNE différence (le ledger des steps appliqués reste cohérent).
- La suite `-tags=integration -p 1` complète reste verte ; SeedDemo end-to-end vert.

## 2. Décisions PRÉ-TRANCHÉES

| # | Décision | Choix ferme |
|---|---|---|
| DM-1 | Ordre | Outillage + test d'invariant AVANT toute baseline (TDD). Le squash réel = dernière phase, gated GO humain (politique N4 point 1 : manuel) |
| DM-2 | Fenêtre préservée | Les 10 derniers steps de chaque registre restent hors baseline (politique N4 point 3 — fenêtre de rollback récent) |
| DM-3 | Archivage | `.ai/migrations/squashed/<version>/` : fichiers source des steps squashés copiés tels quels + un README (provenance, date, hash du commit d'origine) — politique N4 point 4 |
| DM-4 | Transition b23/b25 | Le squash NE traverse PAS la frontière global→title-owned tant que la transition ADR 0025 Phase 1.5 n'est pas statuée stable (M0 l'établit). Si instable : squasher UNIQUEMENT à l'intérieur de chaque registre, jamais en fusionnant les deux mondes |
| DM-5 | Ledger des steps appliqués | Les DBs EXISTANTES ont déjà les steps historiques marqués appliqués. La baseline porte un ID nouveau ; le runner doit la considérer comme DÉJÀ satisfaite sur une DB qui a passé le dernier step squashé (règle d'équivalence explicite, testée) — sinon une DB prod re-exécuterait la baseline au boot |
| DM-6 | Périmètre v1 | Le PREMIER squash vise le registre le plus volumineux et le plus stable (M0 le désigne — attendu : player base / shared core). Un registre à la fois, un commit par registre. Pas de big-bang tous-registres |

## 3. Phases (ordre strict ; commits `squash(MX):`)

### M0 — Cartographie et décision de périmètre (READ-ONLY, consignée)
- [ ] M0a — Inventaire des registres : `internal/migration/order.go` (canonicalOrder) +
  `internal/migration/steps_*.go` (globaux) + `games/halo_infinite/migrations/steps.go` +
  `games/halo_5/migrations/steps.go` (title-owned). Pour chacun : nombre de steps, cible
  (metadata / shared / player / social), dépendances d'ordre inter-registres.
- [ ] M0b — État de la transition b23/b25 (ADR 0025 Phase 1.5, ownership global→title-owned,
  raison du report E7) : quels `create_base_*_schema` existent en double, la bascule
  est-elle terminée ? VERDICT écrit : la frontière est-elle stable ? → applique DM-4.
- [ ] M0c — Mécanique du ledger : comment le runner marque un step appliqué (table de
  suivi ? nom ? `RunForDB`) — établir précisément ce que DM-5 exige.
- [ ] M0d — Mesure de référence : temps de provisioning d'une DB vierge par cible
  (player, shared, metadata, social — 2 titres), consigné (baseline de la mesure de succès).
- [ ] M0e — DÉSIGNER le registre v1 (DM-6) + la borne de baseline (dernier step inclus,
  en respectant DM-2 : les 10 derniers restent dehors). Point d'étape utilisateur.
- Gate M0 : rapport complet au Journal §J ; aucune modification de code.

### M1 — Outil de snapshot de schéma normalisé
- [ ] M1a — `cmd/schema-snapshot` (ou fonction test-only dans `internal/migration` si plus
  simple — décider en M0, documenter) : ouvre une DB DuckDB, extrait le schéma COMPLET et
  le sérialise NORMALISÉ : tables (colonnes, types, defaults, NOT NULL, PK), index, vues
  (définition SQL normalisée), séquences — trié de façon déterministe (ordre lexical),
  indépendant de l'ordre d'exécution des steps là où l'ordre n'a pas d'effet observable.
  Sources DuckDB : `duckdb_tables()`, `duckdb_columns()`, `duckdb_views()`,
  `duckdb_constraints()`, `duckdb_indexes()` (vérifier la disponibilité réelle dans la
  version embarquée).
- [ ] M1b — Déterminisme prouvé : 2 provisionings vierges successifs (même historique) →
  snapshots identiques octet pour octet (test).
- [ ] M1c — Sensibilité prouvée : altérer artificiellement un schéma (1 colonne, 1 vue)
  → diff non vide (test des deux sens, sonde jetable supprimée ensuite).
- Gate M1 : tests M1b/M1c verts ; build+vet 0.

### M2 — Test d'invariant bit-identique (le verrou central — AVANT toute baseline)
- [ ] M2a — `internal/migration/squash_invariant_test.go` : provisionne DEUX DBs vierges
  par cible visée — (A) historique complet actuel, (B) chemin candidat (baseline + steps
  restants ; tant que M3 n'existe pas, B = A et le test tourne en mode « harnais prêt ») —
  snapshot M1 des deux, comparaison stricte, `t.Errorf` avec diff lisible par section.
- [ ] M2b — Brancher le test en CI (il tourne avec la suite `-tags=integration -p 1` —
  vérifier la durée ; s'il est lourd, le scoper au registre en cours de squash).
- Gate M2 : harnais vert en mode A=B ; morsure prouvée (schéma altéré → rouge).

### M3 — Génération de la baseline (registre v1 désigné en M0e)
- [ ] M3a — Écrire le step `create_baseline_<cible>_v<version>` : schéma « à plat »
  produisant EXACTEMENT l'état cumulé des steps squashés (s'appuyer sur le snapshot M1 de
  référence comme spec ; le step est écrit à la main ou généré — décider et documenter —
  mais RELU dans les deux cas).
- [ ] M3b — Règle d'équivalence ledger (DM-5) : une DB portant le dernier step squashé
  est réputée porter la baseline (implémentation dans le runner + TEST dédié : DB
  provisionnée à l'ancienne → boot avec le nouveau registre → AUCUN step rejoué, schéma
  intact).
- [ ] M3c — Câbler le registre : baseline + 10 derniers steps ; `order_audit_test.go`
  (audit d'ordre) mis à jour.
- [ ] M3d — Le test d'invariant M2 passe en mode RÉEL (A = historique complet archivé,
  B = baseline + reste) → VERT exigé.
- Gate M3 : invariant vert ; test ledger M3b vert ; `-tags=integration -p 1` cible verte.

### M4 — Archivage + documentation
- [ ] M4a — Copier les steps squashés dans `.ai/migrations/squashed/<version>/` + README
  (DM-3). Les fichiers source des steps sont RETIRÉS du registre actif seulement à ce
  stade (git garde tout de toute façon — l'archive est une commodité d'audit).
- [ ] M4b — `internal/migration/doc.go` : la politique passe de « PROPOSITION » à
  « APPLIQUÉE le <date> (registre <X>, version <V>) » + renvoi vers ce plan et l'archive.
- [ ] M4c — Mesure après (M0d rejouée) : temps de provisioning vierge avant/après consigné.
- Gate M4 : build+vet+tests migration verts ; archive en place.

### M5 — Vérifications end-to-end (avant tout GO)
- [ ] M5a — Suite complète `-tags=integration -p 1 -timeout 900s ./...` → exit 0.
- [ ] M5b — SeedDemo end-to-end (intégration ops) → vert (provisionne des DBs vierges,
  c'est le consommateur réel du chemin baseline).
- [ ] M5c — Répétition sur COPIE PROD (celle de `LevelUp-prod-copy`, restaurée V10a — la
  rafraîchir via restic si périmée) : booter le binaire de la branche sur la copie →
  AUCUN step rejoué (DM-5), schéma intact (snapshot M1 avant/après identiques), pages OK.
- Gate M5 : les 3 verts, consignés.

### M6 — GO opérateur puis merge (politique N4 : déclenchement manuel)
- [ ] M6a — Point d'étape utilisateur : mesures (M0d vs M4c), diff de registre, résultat
  M5c. DEMANDER le GO explicite (c'est LA décision opérateur de la politique).
- [ ] M6b — Si GO : merge selon les règles projet (prévenir, deploy auto). Si NO-GO :
  la branche reste (l'outillage M1/M2 est réutilisable même sans squash — il sert aussi
  au chantier E7 futur), consigner.

## 4. Synergies (à noter, PAS à traiter ici)
- **E7** (DDL bootstrap `sync/schema.go` → migration, différé « après b23/b25 ») : le
  snapshot M1 + l'invariant M2 sont exactement l'outillage qui dé-risquera E7 (prouver
  que Ensure* et les migrations produisent le même schéma). Le noter dans DETTE_ASSUMEE
  à la clôture de M2.
- **V9d** (rebuild DROP colonnes mortes) : indépendant — le rebuild est une opération de
  DONNÉES (ADR 0026), pas de registre. Ne pas mélanger.

## 5. Hors périmètre
- E7 lui-même ; tout changement de schéma fonctionnel ; le rebuild V9d ; les registres
  non désignés en M0e (ils suivront le même chemin, chantiers ultérieurs).

## 6. Journal §J
- (vide au démarrage)

## 7. Découvertes hors périmètre (NE PAS traiter)
- (vide au démarrage)
