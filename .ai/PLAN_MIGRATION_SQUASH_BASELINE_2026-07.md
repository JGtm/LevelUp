# PLAN — Squash des migrations : baseline bit-identique prouvée (chantier N4)

> **STATUT : M3-M5 COMPLÉTÉS le 2026-07-12 (1er squash réel LIVRÉ : baseline PLAYER v1).
> M6 (merge = deploy prod auto) EN ATTENTE DU TRAIN DE MERGE SUPERVISEUR** — hors mandat
> de l'exécutant. M0/M1/M2 étaient déjà mergés (PR #54). GO opérateur donné 2026-07-12
> (périmètre v1 confirmé = cible player, bloc title-owned contigu). Tout est vert sur la
> branche `refactor/migration-squash-m3` : baseline `create_baseline_player_v1` (33 steps
> squashés), preuve zéro-perte bit-identique (golden), DM-5 (équivalence ledger), archive
> `.ai/migrations/squashed/player_v1/`. Le plan reste dans `.ai/` tant que M6 n'est pas
> exécuté. Détail : §J (entrée M3-M5, 2026-07-12).
>
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
- [x] M2a — `games/halo_infinite/migrations/squash_invariant_test.go` (DÉVIATION documentée
  de l'emplacement plan : le provisioning complet exige StepsFor, non importable depuis
  `internal/migration` — cycle ; même raison qu'order_audit_test.go). Provisionne DEUX DBs
  vierges par cible (metadata/shared/pve/social/player) via `provisionFullHistory` (oracle)
  et `provisionCandidate` (runner actif) ; snapshot M1 des deux ; comparaison stricte avec
  `firstDiff` lisible. SEAM en place : aujourd'hui A=B (mode harnais) ; post-M3
  provisionFullHistory rejouera le fixture des steps squashés.
- [x] M2b — CI : le test est integration-tagged dans le package titre → tourne
  automatiquement sous `-tags=integration -p 1 ./...` (gate CI). Durée mesurée 3.0s (2.5s
  invariant 5 cibles + 0.4s morsure) — léger, pas de scoping nécessaire.
- Gate M2 : harnais VERT en mode A=B (5 cibles PASS) ; morsure prouvée
  (`TestSquashInvariant_BiteProof` : colonne ajoutée → snapshots divergents, détecté). [x]
  Synergie E7 notée dans `DETTE_ASSUMEE_2026-Q3.md` (§4 du plan).

### M3 — Génération de la baseline (registre v1 désigné en M0e)
> GO opérateur reçu 2026-07-12 (périmètre v1 = cible player, bloc title-owned contigu).
> Baseline GÉNÉRÉE depuis le golden (spec) puis RELUE. Exécuté 2026-07-12.
- [x] M3a — Borne figée SUR PIÈCES (classification machine-vérifiée) : bloc
  `create_base_player_schema`→`player_append_only_csr_snapshots_v1` = 33 steps ; 1er
  GLOBAL suivant `player_append_only_match_citations_v1` (DM-4 respecté ; bloc = préfixe →
  DM-2 satisfait). Baseline `create_baseline_player_v1` (`steps_player_baseline.go`) :
  DDL « à plat » du schéma cumulé, GÉNÉRÉ depuis le golden. Reproduit les quirks
  (career_progression sans id/PK + séquence présente ; media_files net-absente).
- [x] M3b — DM-5 : champ `Migration.SupersededByAll` + `supersededBaselineSatisfied`
  (registry.go) → DB portant la sentinelle (dernier step squashé) réputée porter la
  baseline, DDL non rejoué. Test décisif « poison » (`internal/migration/squash_dm5_test.go` :
  skip-DDL si sentinelle présente ; DDL exécuté sur DB vierge). Verts.
- [x] M3c — Registre câblé : baseline en tête du bloc player (canonicalOrder), 33 noms
  retirés, steps post-borne préservés. `order_audit_test.go` (TestCanonicalCoversGlobalAndTitle)
  vert. Garde anti-réintroduction (`TestSquashInvariant_PlayerSquashedStepsRemoved`).
- [x] M3d — Invariant RÉEL : `TestSquashInvariant_PlayerBaselineEquivalent` —
  `SchemaSnapshot(baseline) == golden` (historique 33 steps, capturé avant retrait), octet
  pour octet + bite proof. Preuve compositionnelle (post-borne inchangé → provisioning
  complet identique). DÉVIATION SEAM documentée : le bloc étant archivé/retiré du code
  actif, le golden figé EST l'oracle historique (au lieu d'un rejeu live).
- Gate M3 : invariant vert ; DM-5 vert ; `-tags=integration -p 1` migration+player verts. [x]

### M4 — Archivage + documentation
- [x] M4a — `.ai/migrations/squashed/player_v1/` : README (33 steps, provenance HEAD
  9296496c9, DM-5, preuve) + `source/presquash_*.go` (4 fichiers pré-squash) + golden.
  Les 33 steps + 4 helpers orphelins retirés du registre actif.
- [x] M4b — `internal/migration/doc.go` : politique N4 « PROPOSITION » → « APPLIQUÉE le
  2026-07-12 (baseline player v1) » + points 1-6 (dont DM-5) + renvoi archive/plan.
- [x] M4c — Boot player vierge (:memory:, best-of) : ~229 ms (M0d) → ~111-117 ms (best-of-5) ;
  61→29 steps ; schema_version 194→162.
- Gate M4 : build+vet+lint(0)+tests migration verts ; archive en place. [x]

### M5 — Vérifications end-to-end (avant tout GO)
- [x] M5a — Suite complète `-tags=integration -p 1 -timeout 900s ./...` → exit 0 (voir §J).
- [x] M5b — SeedDemo end-to-end → vert (consommateur réel du chemin baseline, DBs vierges).
- [x] M5c — Répétition sur COPIE PROD `../LevelUp-prod-copy` : voir §J (fraîcheur + verdict).
- Gate M5 : les 3 verts, consignés. [x]

### M6 — GO opérateur puis merge (politique N4 : déclenchement manuel)
- [!] M6a — Point d'étape utilisateur : mesures (M0d vs M4c), diff de registre, résultat
  M5c. DEMANDER le GO explicite (c'est LA décision opérateur de la politique).
- [!] M6b — Si GO : merge selon les règles projet (prévenir, deploy auto). Si NO-GO :
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

### Clôture PARTIELLE (2026-07-11)

M0/M1/M2 COMPLÉTÉS et livrés sur `refactor/migration-squash-baseline` (commits `squash(M0)`
823c09e68, `squash(M1)` 949d70eb2 + fix noctx ae294e566, `squash(M2)` 7830cfafb). Objectif
#1 (CAPACITÉ + PREUVE zéro-perte) atteint : `migration.SchemaSnapshot` + invariant
bit-identique (5 cibles, mode A=B, morsure prouvée).

**Gates de livraison (tous exécutés cette session, verts)** :
- `golangci-lint run --new-from-rev=origin/main` = 0 issue.
- `go test ./...` = exit 0.
- `go test -tags=integration -p 1 -timeout 900s ./...` = exit 0 (aucun FAIL).
- CI de branche (run 29165659241) = TOUS les jobs `success` (E2E skipped) — Go Lint inclus
  (only-new-issues), Baseline non-régression, Build+Test ubuntu+windows, Coverage complet.
- Aucun test supprimé/renommé → baseline `tests_pre_migration.jsonl` inchangée.

**M3→M6 EN ATTENTE GO OPÉRATEUR** (dépendance décisionnelle + prod, pas report de
commodité) : politique N4 point 1 (déclenchement MANUEL), DM-1, M0e (point d'étape user),
M5c (copie prod), M6a (GO explicite avant merge = deploy auto). Périmètre v1 désigné (player,
bloc title-owned contigu) + approche baseline générée recommandée : prêt à exécuter en
session dédiée dès le GO.

**Vérifications utilisateur requises AVANT le 1er squash réel** (M6a) :
1. Confirmer le registre v1 (player) et valider la borne de baseline que M3a figera.
2. Donner le GO explicite pour lancer M3→M5 (génération baseline + invariant réel vert).
3. M5c : autoriser la répétition sur la copie prod (`../LevelUp-prod-copy`, à rafraîchir via
   restic si périmée) — lecture seule, aucun écrit prod.
4. Au merge (M6b) : push main = DEPLOY AUTO — prévenir avant.

### M3-M5 — 1er squash réel : baseline PLAYER v1 (2026-07-12, GO opérateur)

**M3a — Borne (sur pièces, machine-vérifiée).** Bloc contigu title-owned player =
`create_base_player_schema` → `player_append_only_csr_snapshots_v1` = **33 steps** (préfixe
du tier player dans canonicalOrder). 1er step GLOBAL suivant = `player_append_only_match_citations_v1`
→ DM-4 respecté (pas de traversée global↔title). Bloc = préfixe ⇒ DM-2 satisfait (tous les
steps post-borne préservés). Sources réparties sur 3 fonctions (`playerBaseSteps` 26,
engagement de `playerSteps` 5, `playerMatchSkillRankSteps` 1 [player_add_expected_win_prob],
`appendOnlyMiscSteps` 1). Correctif cartographie M0 confirmé : `create_base_player_schema`
EST title-owned (b25).

**M3b — Baseline + DM-5.** `create_baseline_player_v1` (steps_player_baseline.go) : DDL
« à plat » GÉNÉRÉ depuis le golden (capturé de l'historique réel via RunSteps sur DB vierge,
AVANT retrait → oracle indépendant). Reproduit les quirks : career_progression SANS id/PK
mais séquence `career_progression_id_seq` présente ; media_files net-absente (créée puis
droppée). Équivalence bit-identique golden↔baseline **dès le 1er essai**. DM-5 : champ
`Migration.SupersededByAll` (33 noms) + `supersededBaselineSatisfied`/`recordSupersededBaseline`
(registry.go) → DB portant la sentinelle réputée porter la baseline, DDL non rejoué. Tests
« poison » décisifs (`internal/migration/squash_dm5_test.go`).

**M3c/M3d — Registre + invariant.** 33 noms retirés de canonicalOrder + StepsFor (baseline
en tête du bloc) ; order_audit vert ; garde anti-réintroduction. Invariant :
`TestSquashInvariant_PlayerBaselineEquivalent` (SchemaSnapshot(baseline) == golden octet pour
octet) + bite proof. DÉVIATION SEAM documentée : le golden figé (historique archivé) EST
l'oracle (le bloc étant retiré du code actif, rejeu live impossible). Preuve compositionnelle :
post-borne inchangé (byte-identique) ⇒ provisioning complet identique.

**M4 — Archive + doc + mesure.** `.ai/migrations/squashed/player_v1/` (README + 4 sources
pré-squash HEAD 9296496c9 + golden). doc.go N4 → APPLIQUÉE. Boot player vierge (:memory:,
best-of-5) : ~229 ms (M0d) → ~111-117 ms ; 61→29 steps ; schema_version 194→162.

**Robustesse E7 (découverte en M5a, corrigée).** Le CREATE-IF-NOT-EXISTS à plat no-opait
quand une table pré-existait partielle (sync `EnsureSchema` / bootstraps de test créant
match_skill_rank sans start_time, player_match_enrichment sans colonnes engagement). Les
steps historiques la patchaient via AddColumnIfMissing → la baseline reproduit ce contrat
idempotent-additif (`ensureBaselinePlayerV1AdditiveColumns`). No-op sur DB vierge (golden
préservé). Fix des 14 échecs convergence/batch (sync + platform/duckdb).

**M5 — End-to-end.**
- M5a : `-tags=integration -p 1 -timeout 900s ./...` → exit 0 (après fix E7 + reword doc.go
  pour le garde `TestNoUnauthorizedSharedSocialMention`).
- M5b : SeedDemo end-to-end couvert par `internal/ops` (consommateur réel du chemin baseline,
  DBs vierges) → vert.
- M5c : rehearsal sur les 4 player DB de `../LevelUp-prod-copy` (copie temp, runner réel de la
  branche) → sentinel_present=true, **schéma intact (before==after), seule nouvelle ligne
  ledger = create_baseline_player_v1 (0 DDL rejoué)** pour les 4. FRAÎCHEUR : copie à
  schema_version 190 (pré-squash 194, ~4 steps de retard) — NON bloquant pour la cible player
  (sentinelle présente, DM-5 opérant). Sonde M5c supprimée (non committée, modèle M0d).

**Gates M3-M5 (verts cette session)** : go build ./... = 0 ; golangci-lint
--new-from-rev=origin/main (packages changés) = 0 ; invariant + DM-5 verts ;
`-tags=integration -p 1 ./...` = exit 0 ; baseline CI mise à jour (3 tests obsolètes retirés).

**M6 (merge = deploy prod auto) : HORS mandat exécutant → train de merge superviseur.**

## 7. Découvertes hors périmètre (NE PAS traiter)
- **Doc drift mineur (non traité)** : ~6 commentaires (sync_meta_repo.go, notifications_boot.go,
  media_store.go, known_loader_test.go, e2e_test.go) citent la migration `create_base_player_schema`
  par nom — step désormais squashé (les tables restent créées, par la baseline). Purement
  documentaire, aucun impact fonctionnel ; laissé tel quel (hors périmètre).
- **CI « Go Lint » = ONLY-NEW-ISSUES** : le job golangci-lint ne fait échouer QUE sur des
  issues NEUVES vs base (comportement observé : 1er push RED sur le seul `noctx` de mon
  code → corrigé → 2e run VERT). Il SURFACE néanmoins en annotations informationnelles une
  dette baseline PRÉ-EXISTANTE (funlen `MetadataSteps` 204>80, `MapParticipants`,
  `LoadFromConfigDir`, `applyModeNameTr`, `registerHalo5Adapters`, `StartInitialSync`,
  `handleCreatePlayer`, `handlePatchSettings` ; errcheck `os.Remove`/`os.RemoveAll`). Cette
  dette est GELÉE (CLAUDE.md règle 5) et NE fait PAS échouer la CI de branche — non traitée.
- **Hook pre-commit go-vet** : imprime de nombreux `build constraints exclude all Go files`
  pour `cmd/*` + `duckdb-go-bindings/lib/windows-amd64` (tags/CGO Windows) — bruit
  pré-existant, le hook PASSE quand même. Non traité.
- **Note M3 (pas une découverte, une contrainte confirmée)** : DM-4 (frontière instable,
  M0b) impose que la baseline player ne couvre que le bloc title-owned CONTIGU précédant le
  1er step GLOBAL player. La borne exacte est à figer sur pièces en M3a (post-GO) car les 26
  steps globaux append-only s'intercalent dans l'ordre player.
