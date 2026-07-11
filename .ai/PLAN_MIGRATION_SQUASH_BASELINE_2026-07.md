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
- [x] M0a — Inventaire des registres (cf. §J/M0a). 193 steps dans canonicalOrder (43 metadata,
  61 shared, 60 player, 27 shared_social, 2 shared_pve) ; 26 steps GLOBAUX (registre
  `internal/migration` via init/Register — append-only ADR 0026 + drop-ART, cross-titre),
  167 title-owned Halo Infinite (`games/halo_infinite/migrations`), 12 title-owned Halo 5
  (set ISOLÉ metadata seule). Ordre imposé par `canonicalOrder` (défaut) / `set.CanonicalOrder`.
- [x] M0b — VERDICT : frontière b23/b25 NON stable (E7 explicitement gaté « après
  stabilisation b23/b25 », DETTE_ASSUMEE §7). Les `create_base_*_schema` sont title-owned
  SANS doublon global (transition des bases faite), mais les 26 steps globaux (append-only)
  s'INTERCALENT dans l'ordre de chaque cible → DM-4 s'applique : le 1er squash NE FUSIONNE
  PAS les deux mondes ; il ne squashe qu'un bloc CONTIGU d'UN SEUL monde.
- [x] M0c — Ledger (cf. `registry.go`) : `schema_migrations` (PK `name`) trace chaque step
  appliqué ; le runner SKIPPE un step dont le `name` est déjà présent. `title_schema_version`
  (PK title_slug+target) porte version=len(order). DM-5 exige donc : une baseline au NOM
  NOUVEAU serait rejouée sur une DB prod existante (name absent) — MAIS elle est
  `CREATE ... IF NOT EXISTS` idempotente (no-op sur schéma déjà présent) ; l'équivalence
  ledger (M3b) marquera la baseline comme satisfaite si le dernier step squashé est présent,
  pour éviter tout DDL rejoué.
- [x] M0d — Provisioning DB vierge (:memory:, best-of-3, tag integration) : metadata 697ms
  (dominé par les SEEDS), player 229ms, shared 196ms, shared_social 92ms, shared_pve 16ms.
  Sonde jetable supprimée (non committée). DuckDB introspection dispo :
  duckdb_tables/columns/views/constraints/indexes/sequences (toutes OK).
- [x] M0e — DÉSIGNÉ : registre v1 = **cible player, bloc title-owned contigu** partant de
  `create_base_player_schema` jusqu'au dernier step title-owned PRÉCÉDANT le 1er step
  GLOBAL de la cible player (borne exacte figée en M3a sur pièces). Respecte DM-4 (un seul
  monde), DM-2 (prefix → les 10 derniers steps player restent hors baseline), schéma-only
  (pas de seed data à perdre — contrairement à metadata). RECOMMANDÉ vs shared (bases plus
  tardives, 58 vues, blocs plus entrelacés) et vs halo_5 (isolé mais 12 steps, faible
  valeur, seed milestones data). Point d'étape utilisateur = décision opérateur (M6a GO).
- Gate M0 : rapport complet au Journal §J ; aucune modification de code committée. [x]

### M1 — Outil de snapshot de schéma normalisé
- [x] M1a — DÉCIDÉ (M0) : fonction LIBRAIRIE réutilisable `migration.SchemaSnapshot(db)`
  (`internal/migration/schema_snapshot.go`), pas un cmd — appelable par le test d'invariant
  M2 (même package + package titre) ET par un futur `cmd/schema-snapshot` pour M5c. Extrait
  tables (schema.table + PK flag), colonnes (POSITIONNEL, ordre observable préservé), types,
  defaults, nullable, contraintes, index (sql normalisé), vues (sql normalisé), séquences.
  Objets de 1er niveau triés lexicalement ; SCHÉMA SEUL (zéro donnée lue). Sources vérifiées
  dispo : `duckdb_tables/columns/constraints/indexes/views/sequences()`.
- [x] M1b — Déterminisme prouvé : `TestSchemaSnapshot_DeterministicIdenticalSchema` +
  `TestSchemaSnapshot_DeterministicRunForDB` (2 provisionings RunForDB → snapshot identique
  octet pour octet ; les lignes ledger, données, ne sont pas capturées).
- [x] M1c — Sensibilité prouvée (morsure 2 sens) : `TestSchemaSnapshot_SensitiveToMutations`
  (6 dimensions : colonne, default, table, vue, index, séquence — chacune → diff) +
  `TestSchemaSnapshot_ColumnOrderIsObservable` (garde-fou fausse équivalence par tri
  colonnes). Tests PERMANENTS (le tool est du code livré), pas de sonde résiduelle.
- Gate M1 : tests M1b/M1c verts (8 sous-tests PASS) ; `go vet` + `go build` migration = 0. [x]

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

### M0 — Cartographie (2026-07-11, READ-ONLY)

**M0a — Inventaire des registres.** Source d'ordre unique : `canonicalOrder`
(`internal/migration/order.go`), 193 entrées. Répartition par cible (commentaires de
l'ordre) : metadata 43, shared 61, player 60, shared_social 27, shared_pve 2.

Trois SOURCES de steps, dédupliquées par `Name` (title-owned override le global) puis triées
par `canonicalOrder` :
- **Registre global** (`internal/migration/steps_*.go`, `Register()` en `init()`) : 26 steps,
  TOUS de nature « maintenance ART » (ADR 0026) : conversions append-only
  (`*_append_only_v1`), drops d'index ART (`drop_*_art_*`), rebuilds
  (`rebuild_match_participants_defeat_art_corruption`, `rebuild_catalog_fetch_queue_*`),
  markers reset skill v2. Cross-titre par design (pas en transition).
- **Title-owned Halo Infinite** (`games/halo_infinite/migrations/`, via
  `SetTitleStepsProvider(StepsFor)`) : 167 steps (bases `create_base_*_schema`, ALTER,
  familles metadata, prestige, progression, world leaderboard…).
- **Title-owned Halo 5** (`games/halo_5/migrations/`, `RegisterMigrationSet`) : 12 steps
  metadata, set ISOLÉ (`OwnsTarget==metadata` seul ; shared/player/social/pve héritent du
  fallback HINF). CanonicalOrder propre (`metadataStepNames`).

Routage runner (`RunForTitleDB`) : si un set est enregistré pour le slug ET possède la cible
→ steps+ordre DU SET (isolation totale) ; sinon fallback legacy (registre global +
titleStepsProvider, ordonné par canonicalOrder), byte-identique au défaut Halo.

**M0b — Frontière b23/b25.** VERDICT : **NON stable**. Preuve : E7 (« DDL bootstrap
sync/schema.go → migration », le refactor couplé à la transition b23/b25 title-ownership) est
statué `[!]` dans DETTE_ASSUMEE_2026-Q3 §7 avec condition de reprise « chantier dédié APRÈS
stabilisation b23/b25 ». Constat de code : les `create_base_{player,shared,shared_social}_schema`
sont title-owned SANS doublon dans le registre global (la transition des bases elle-même est
faite), MAIS les 26 steps globaux (append-only/ART) s'INTERCALENT dans l'ordre d'exécution de
chaque cible (ex. player : base title-owned en tête, puis `player_append_only_match_*_v1`
GLOBAUX plus loin). Conséquence DM-4 : le 1er squash NE traverse PAS la frontière — il ne
fusionne pas un bloc mêlant steps globaux et title-owned. On squashe un bloc CONTIGU d'un
SEUL monde (title-owned).

**M0c — Ledger (DM-5).** `ensureMigrationTable` crée `schema_migrations(name PK, …,
title_slug)`. `runSteps` charge `getApplied` puis, pour chaque step, `if !exists` applique le
DDL et INSERT la ligne ; **si le name existe déjà → step SKIPPÉ** (aucun DDL). Version
courante tracée dans `title_schema_version(title_slug,target)` = `len(order)`. Implication
DM-5 : une baseline au NOM NOUVEAU n'est PAS dans `schema_migrations` d'une DB prod
existante → elle serait « rejouée » au prochain boot. Comme le DDL baseline est
`CREATE … IF NOT EXISTS` (idempotent), c'est un no-op fonctionnel ; néanmoins M3b posera une
règle d'équivalence explicite (marquer la baseline satisfaite si le dernier step squashé est
présent) pour garantir « zéro DDL rejoué » et un ledger cohérent.

**M0d — Mesures de référence** (provisioning DB :memory: vierge, best-of-3, tag integration,
sonde jetable supprimée non-committée) :

| Cible | Temps (best-of-3) | Note |
|---|---|---|
| metadata | 697 ms | dominé par les SEEDS (ranked playlists, milestones, career ranks, csr thresholds) |
| player | 229 ms | 60 steps, bases + ALTER + append-only |
| shared | 196 ms | 61 steps, bases + 58 vues |
| shared_social | 92 ms | 27 steps |
| shared_pve | 16 ms | 2 steps |

Introspection DuckDB (embarqué v2) : `duckdb_tables()`, `duckdb_columns()`, `duckdb_views()`,
`duckdb_constraints()`, `duckdb_indexes()`, `duckdb_sequences()` TOUTES disponibles → M1 peut
s'appuyer dessus.

**M0e — Périmètre v1 DÉSIGNÉ.** Registre v1 = **cible player, bloc title-owned contigu** de
`create_base_player_schema` jusqu'au dernier step title-owned précédant le 1er step GLOBAL
player (borne exacte figée en M3a). Justification :
- DM-4 (frontière instable) : bloc title-owned pur, ne fusionne pas les mondes.
- DM-2 : un PREFIX → les 10 derniers steps player restent hors baseline (fenêtre rollback).
- Schéma-only : player = CREATE/ALTER/append-only, sans seed data (contrairement à metadata
  dont M1 — schéma seul — ne capturerait pas les seeds : risque de perte silencieuse).
- Valeur : les player DBs sont NOMBREUSES (une par joueur) → gain de boot multiplié.
Alternatives écartées : shared (bases plus tardives, 58 vues, blocs plus entrelacés →
génération bit-identique plus fragile) ; halo_5 metadata (monde propre isolé mais 12 steps,
faible valeur, seed milestones = data). La désignation finale + le GO restent une décision
OPÉRATEUR (politique N4 point 1 ; M6a).

Gate M0 : OK — rapport ci-dessus, aucune modification de code committée (sonde M0d/M1
supprimée).

## 7. Découvertes hors périmètre (NE PAS traiter)
- (vide au démarrage)
