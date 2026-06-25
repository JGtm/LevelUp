# PLAN — Durabilité convergente + lectures sur substrat immuable

> Statut : PROPOSITION (non démarré). Auteur : agent IA. Date : 2026-06-25.
> Déclencheur : revue de l'article "DuckDB + Kafka micro-batches that feel real-time"
> confronté à notre architecture (ART, persist workers, B-swap) + insight produit
> de l'utilisateur : « un match récupéré et enrichi est immuable, sauf exclusion ou
> changement d'escouade ».
> Branche cible proposée : `refactor/durabilite-snapshot-immuable` (à confirmer avant tout code).

---

## 0. Résumé exécutif

L'article ne propose pas d'architecture supérieure : il **valide** notre direction
(writer unique + micro-batch + commit atomique + append-only + merge-on-read). Trois
idées sont néanmoins transférables, classées par ROI une fois confronté à l'existant
réellement vérifié dans le code (2026-06-25) :

1. **« Commit l'offset seulement après la durabilité »** — déjà appliqué pour LUSR
   (`canonicalGate`), mais **à la main, LUSR-only, non testé comme invariant**. À
   généraliser + verrouiller par un garde-fou. ROI : moyen, risque faible.
2. **Éradiquer le dernier UPDATE indexé sur `shared.match_registry`** (`dominance_flag`).
   ROI : élevé, risque faible, ferme la campagne append-only sur la base la plus contendue.
3. **Servir les lectures historiques depuis un substrat immuable** (Parquet versionné)
   plutôt que la base RW → ces lectures ne subissent plus les stalls du B-swap. ROI :
   potentiellement élevé (prédictibilité), **coût moyen-élevé, à dé-risquer par un pilote mesuré**.

**Recommandation** : exécuter **Phase 0 (mesure) + Phase 1 (hardening haute confiance)
inconditionnellement**. La décision d'engager les Phases 2-4 (lectures sur snapshot) est
**gated sur les chiffres de la Phase 0** — on ne fait pas le gros pari sans preuve de gain.

Non-objectifs : introduire Kafka ; un big-bang sur les ~150 sites de lecture ; un flag
qui laisse une demi-feature OFF en prod (tout flag de pilote a un critère de retrait explicite).

---

## 1. État vérifié de l'existant (preuves)

| Constat | Preuve (fichier:ligne) | Conséquence pour le plan |
|---|---|---|
| Discipline durable-write-avant-watermark déjà là pour LUSR | `internal/sync/skill_v2_shadow.go:305-358` (`canonicalGate`, `heldGroups`) | Ne PAS réimplémenter ; généraliser + tester |
| Persist = WAL JSON → channel → `BeginTx`→INSERT→`Commit`, idempotence par `EXISTS`, retry au boot | `internal/persist/queue.go`, `shared_persister.go`, `player_persister.go`, `worker.go` | Substrat durable solide, réutilisable comme "journal" |
| Enrichissements = INSERT-only par `stage` + vue `_latest` | `internal/persist/player_persister.go`, `migration/steps_player_append_only_match_enrichment.go` | Faits dérivés déjà append-only |
| Faits bruts immuables après 1er sync | `match_registry`/`match_participants`/`medals_earned`/`highlight_events`/`killer_victim_pairs` (INSERT-only) | Candidats snapshot immuable |
| **`dominance_flag` = UPDATE direct sur `shared.match_registry`** | `internal/sync/comeback.go:BackfillDominanceFlags()` | Dernier UPDATE indexé sur shared → Phase 1 |
| B-swap = contournement mono-process pour les ÉCRITURES ; les lectures stallent pendant RW | `internal/platform/duckdb/sharedprovider/provider_writer.go`, mémoire `project_bswap_root_cause...` | Sortir les lectures de la base RW = angle neuf |
| Producteur Parquet + manifest JSON existe (à froid, par-joueur) | `internal/ops/archive.go` (`COPY ... TO parquet`, `archive_index.json`) | Embryon réutilisable pour Phase 2 |
| Chemins title-aware centralisés | `internal/domain/title/registry.go` (`PathResolver`) | Snapshot dir par titre obligatoire |

Mutations possibles d'un match APRÈS enrichissement (audit exhaustif) :
- **Rares, user-driven, append-only** : `is_excluded` (exclusion), `is_with_friends`
  (escouade), `match_favorites`, `media_likes`.
- **Systématiques, recalcul, append-only** : `performance_score`, LUSR, `engagement_*`,
  `session_id`, `match_citations`.
- **Exception non append-only** : `dominance_flag` sur `shared.match_registry` (UPDATE direct).

Conclusion : l'hypothèse "faits bruts immuables" tient. Les colonnes recalculées et
mutations user vivent toutes en player DB / shared_social en append-only, SAUF
`dominance_flag` — d'où sa priorité.

---

## 1bis. Raffinement (2026-06-25) — les DÉRIVÉS sont immuables-ancrés, pas seulement les faits bruts

Précision validée avec l'utilisateur (concepteur du modèle) : les dérivés par match
(`performance_score`, LUSR écrit, `session_id`, citations) sont **ancrés** — calculés une
fois à partir de l'historique disponible AU MOMENT du match, écrits, puis **non réécrits**
quand des matchs ultérieurs arrivent (le watermark saute les matchs déjà vus). Ce ne sont
PAS des métriques glissantes qui réécriraient le passé. Donc, pour un match ancien en
régime stable, le dérivé est figé au même titre que le fait brut.

Les seuls mouvements possibles d'un dérivé :
1. **Régénération de version** : changement délibéré de formule / algo / mapping metadata
   → recalcul global. **Rare, intentionnel.** → Dans le modèle snapshot, ce n'est PAS un
   problème : c'est **produire la version N+1** et basculer le manifest. On gagne en prime
   le **rollback** (garder vN tant que vN+1 n'est pas validée).
2. **Effet de frontière** : sessions (peuvent s'étendre au dernier match) — n'affecte que
   la zone récente, jamais l'historique.
3. **Toggles user** : exclusion / favoris / likes — rares, servis par une couche overlay
   légère par-dessus le snapshot.

**Conséquence sur le plan** : le snapshot ne se limite plus aux faits bruts shared — il
devient une **version cohérente et figée de TOUT le dataset (faits + dérivés)**, identifiée
par un numéro (modèle lakehouse). Les lectures de la **player DB** (dérivés) se découplent
alors elles aussi du B-swap. La zone "live" résiduelle se réduit (en Option B, cf. §1ter)
à : **les toggles user** (exclusion/favoris, servis par overlay léger) + un transitoire le
temps que le producteur cut la version — historique et dérivés ancrés sont tous sur l'immuable.

Subtilité assumée : le LUSR est intrinsèquement relatif (lit l'état courant des coéquipiers,
`skill_v2_shadow.go:96-99`), mais sa ligne historique n'est pas réécrite en live → figée en
pratique, ne bouge qu'au recompute délibéré = nouvelle version. Rentre dans le cadre.

---

## 1ter. Contrat de fraîcheur & UX (Option B retenue)

**Garantie centrale : une lecture utilisateur n'est JAMAIS bloquée par une écriture.**
L'app lit des fichiers immuables (version N du manifest) ; le sync écrit ailleurs (live DB)
et le producteur écrit de nouveaux fichiers. Aucun verrou partagé sur le chemin de lecture →
on peut enchaîner autant de syncs qu'on veut sans jamais immobiliser l'app.

**Scénario "un user navigue, un nouveau match arrive" :**

```
T+0       Sync ingère le match → live DB.            UX : aucun impact (fichier différent).
T+0..~1s  Enrichissement + dérivés terminés → match COMPLET.
          Producteur cut version N+1 (partition + manifest flip atomique).  UX : aucun impact.
T+ poll   Front re-fetch (React Query) → API résout sur N+1 → le match apparaît.
          UX : rafraîchissement de données normal. Pas de reload, pas d'écran blanc.
```

**Option B retenue (lecture snapshot seule)** — justification : la règle produit veut qu'un
match ne soit affiché que **complet** (fetché + enrichi + dérivé). Or "complet" = "snapshotable".
Donc dès qu'un match est affichable il entre dans une version → pas besoin d'une frontière
live (Option A). L'app lit UNIQUEMENT le snapshot courant ; un match est visible dès le cut.

**Indispo / délais (résumé UX) :**

| Évènement | UX |
|---|---|
| Pendant une écriture/sync | Aucun stall, app fluide (vs aujourd'hui : spinner jusqu'à ~qq s) |
| Apparition d'un nouveau match | ~1 s après complétion (cut en fin d'enrichissement) |
| Bascule de version (manifest) | Atomique, imperceptible ; lectures en cours sur N finissent sur N |
| Recompute d'algo | Bascule tout-ou-rien sur N+1 + rollback (pas d'état intermédiaire visible) |
| Page détail d'un match ouverte | Donnée immuable identique entre versions → aucune perturbation |

**Engagements qu'impose l'Option B (à tenir, sinon repli Option A) :**
- Cut **incrémental + peu coûteux**, déclenché **en fin d'enrichissement** (après libération
  du write-lease du sync), pas sur timer.
- **Prédicat de complétude** : seuls les matchs complets (présents shared ET player avec tous
  les `_latest`) entrent dans une version ; un match mi-enrichi est exclu jusqu'au prochain cut.
- **Compaction de fond** des petites partitions (job d'arrière-plan, n'impacte pas la visibilité).
- **Surveillance du lag producteur** : B fait du producteur une dépendance unique de la
  visibilité du frais (s'il traîne, le frais gèle — l'app reste fluide). Match complet non
  snapshoté depuis > seuil → alerte.

---

## 2. Gains visés

- **Stabilité** : zéro UPDATE indexé sur `shared.match_registry` → ferme par construction
  le dernier vecteur ART #23046 sur la base la plus écrite.
- **Prédictibilité** : un invariant testé "le progrès n'avance jamais avant l'écriture
  durable" → plus de gap silencieux possible (récidive de l'incident JGtm 03/06).
- **Performance/prédictibilité de lecture** (si Phases 2-4 retenues) : les lectures
  d'historique ne stallent plus pendant les fenêtres RW du B-swap.

---

## 3. Phases

### Phase 0 — Mesure & garde-fous (PRÉ-REQUIS, décisionnel)

Objectif : chiffrer le gain potentiel des lectures-sur-snapshot AVANT de l'engager.

- Instrumenter le B-swap : exposer en expvar le temps cumulé de stall lecteur dû aux
  fenêtres RW (drain + swap + write + reopen) et le nb de `Get()` retardés
  (`sharedprovider/provider_writer.go`).
- Recenser les endpoints de lecture qui ne lisent QUE des faits immuables (pas de
  colonne recalculée fraîche) vs ceux mixant immuable + dérivé frais.
- Mesurer la part de lectures portant sur des matchs "anciens" (> N jours) éligibles snapshot.

Done-def : note chiffrée dans `.ai/` (stall p50/p95, % lectures éligibles, liste
endpoints). **Si stall négligeable → Phases 2-4 abandonnées, on s'arrête à Phase 1.**

### Phase 1 — Hardening haute confiance (inconditionnel, indépendamment livrable)

**1.a — `dominance_flag` en append-only.**
- Migrer la production de `dominance_flag` hors d'un UPDATE sur `match_registry`.
  Deux options à arbitrer en conception :
  (i) colonne dérivée sur une table d'état append-only dédiée + vue `_latest` (pattern
  `internal/migration/append_only_rebuild.go`), lue par jointure ; ou
  (ii) calcul à la lecture si peu coûteux.
- Mettre à jour les lecteurs de `dominance_flag` vers la nouvelle source.
- Étendre l'allowlist/garde `internal/sync/no_art_patterns_test.go` (retirer la tolérance).

**1.b — Invariant "durabilité avant progrès" généralisé.**
- Extraire le pattern de `canonicalGate` en un helper nommé réutilisable
  (`internal/sync/...` ou `internal/persist/...`) : "n'avance le marqueur de progrès
  qu'après confirmation durable de l'écriture associée".
- Audit des autres points d'avance de progrès cross-store (sync match, enrichissements,
  CSR) : confirmer l'ordre write-then-advance ; documenter ceux déjà conformes.
- Garde-fou : test prouvant qu'un échec d'écriture player-DB ne fait pas avancer le
  watermark shared (réutiliser le harnais `append_only_state_guard_test.go`).

Done-def : `dominance_flag` ne génère plus d'UPDATE indexé ; test anti-régression vert ;
campagne append-only fermée sur shared (`project_append_only_eradication_campaign`).

### Phase 2 — Producteur de snapshot immuable "roulant" (conditionnel Phase 0)

- Étendre `internal/ops/archive.go` d'un export à froid par-joueur vers un **snapshot
  versionné d'une version cohérente du dataset** (cf. §1bis) :
  - Faits bruts shared : `match_registry` (sans `dominance_flag`, désormais ailleurs),
    `match_participants`, `medals_earned`, `highlight_events`, `killer_victim_pairs`.
  - Dérivés ancrés (player DB) : `player_match_enrichment_latest`, `match_skill_rank_latest`
    (LUSR), `match_citations_latest`. Lus depuis les vues `_latest` au moment du cut.
- Versionnement : `manifest.json` (version monotone, borne temporelle haute = dernier
  `start_time` couvert, checksums). Aucun DELETE — purement additif. **Rétention de N
  versions** pour permettre le rollback.
- **Régénération d'algo = nouvelle version** : un recompute délibéré (changement de
  formule LUSR/perf, mapping citations) produit une version N+1 ; bascule manifest atomique ;
  vN conservée jusqu'à validation de vN+1 (rollback gratuit).
- **Prédicat de complétude** : seuls les matchs entièrement fetchés + enrichis + dérivés
  (présents shared ET player avec tous les `_latest`, `performance_score`/`engagement_score`
  non NULL, `events_loaded`) entrent dans une version. Un match mi-enrichi est exclu jusqu'au
  prochain cut.
  - **NB — ce n'est PAS le comportement actuel** : aujourd'hui la liste home vient de shared
    (Phase A = "source de vérité de la liste") et les dérivés sont mergés best-effort (Phase B,
    `home_repo_matches.go:32-74`). Un match d'escouade inséré par le watcher d'un coéquipier
    **apparaît avec dérivés NULL** jusqu'à convergence. L'Option B fait du prédicat d'inclusion
    snapshot la définition unique de "complet = affichable" → **corrige** ce flash de dérivés
    vides. Changement de comportement assumé : match visible quand complet (~1 s), pas
    instantanément à moitié vide.
- **Cut déclenché en fin d'enrichissement** (juste après la libération du write-lease du
  sync), pas sur timer → "complet" et "snapshoté" coïncident, latence de visibilité ~1 s.
- **Compaction de fond** : petites partitions par-batch fusionnées périodiquement (ex. par
  mois) → nouvelle version remplace les petits fichiers, swap atomique, perf de lecture
  préservée. N'impacte jamais la visibilité.
- Production HORS fenêtre RW (lecture RO via le chemin 2-phases existant, écriture fichiers).
  Title-aware via `PathResolver` : `data/titles/{slug}/warehouse/snapshots/`.

Done-def : un snapshot interrogeable par `read_parquet`, manifest cohérent, produit
sans toucher au verrou RW ; test DuckDB `:memory:` sur un dataset hétérogène
(`feedback_integration_tests_realistic_datasets`).

**Gestion des matchs incomplets — readiness marker + grâce (cœur de la robustesse B) :**

Piège à éviter : « complet » ≠ « toutes les valeurs non NULL ». Certains matchs sont
*légitimement* partiels et ne le seront jamais (film perdu 404/410 → events/weapons ;
FFA/3+ équipes → pas de LUSR ; etc.). Un prédicat naïf les cacherait à jamais.

Mécanisme retenu — **marqueur de readiness unique, posé en fin de post-sync** :
- `snapshot_ready_at` (append-only) posé quand **chaque** dérivation du match est TERMINALE
  (calculée OU terminalement-absente) — réutilise les marqueurs existants : `events_loaded`,
  no-film 30j, `psa_checked_at`, bits weapon, éligibilité LUSR dérivée des faits.
- `partial_reasons` : dérivations terminalement absentes (ex. `["no_film","lusr_ineligible"]`).
- Le producteur inclut les matchs où `snapshot_ready_at IS NOT NULL`. Prédicat trivial/testable ;
  la logique "est-ce terminal ?" par-dérivation vit à un seul endroit.

Trois tiers d'incomplétude :
1. **Transitoire** (dérivés pas encore calculés) → pas `ready` → absent de la version courante,
   ramassé au prochain cut. Cas nominal, ~1 s pour un match live. Aucune action.
2. **Terminalement partiel** (film perdu, FFA sans LUSR) → `ready` avec `partial_reasons` →
   snapshoté **avec ce qu'on a**, jamais caché. UI : note discrète possible sur ce match.
3. **Bloqué** (devrait converger, traîne) → après **grâce bornée** (délai / N essais, réutilise
   l'horizon de convergence) on pose `snapshot_ready_at` **de force** + `partial_reasons` → un
   seul dérivé en échec ne gèle jamais l'affichage.

Monitoring (opérateur, pas utilisateur — expvar `levelup`, ADR 0009) : étendre les compteurs
de convergence avec `snapshot_pending_total`, `snapshot_pending_oldest_age_seconds`,
distribution de `partial_reasons`. Alerte si le plus vieux pending dépasse un seuil ou si le
taux terminal-partial grimpe (= régression d'ingestion). **L'utilisateur n'investigue jamais**
(doctrine app autonome / zéro perte silencieuse) ; la grâce garantit l'affichage partiel.

### Phase 3 — Pilote lecture (1 endpoint, mesuré, décisionnel)

- Choisir UN endpoint read-heavy identifié Phase 0, lisant des faits immuables
  d'historique.
- **Option B (lecture snapshot seule)** : l'adapter sert UNIQUEMENT depuis le snapshot
  (`read_parquet`) de la version courante du manifest. Un match complet est visible dès le
  cut (fin d'enrichissement). Pas de lecture deux-zones. Toggles user via overlay léger.
- **Option A (frontière sur live DB) = repli** seulement si Phase 0/2 montre qu'on ne peut
  pas rendre le cut assez prompt/peu coûteux.
- Flag de pilote AVEC critère de retrait explicite : à l'issue de Phase 3, soit on
  adopte (Phase 4, flag retiré, lecture snapshot par défaut), soit on retire le code.
  Pas d'état demi-on permanent (`feedback_no_flags_partial_features`).

Done-def : réduction de stall mesurée sur l'endpoint pilote ; latence de visibilité d'un
nouveau match (complet → affiché) mesurée et bornée (~1 s) ; prédicat de complétude prouvé
(aucun match mi-enrichi affiché, aucun match complet manquant) ; lag du producteur surveillé
(match complet non snapshoté > seuil = alerte).

### Phase 4 — Rollout conditionné

- Si le pilote prouve le gain : généraliser l'adapter aux autres sites de lecture
  immuable, retirer le flag, documenter (ADR).
- Sinon : retirer le code du pilote, conserver Phases 1-2 (snapshot reste utile pour
  l'archivage/export), clore.

---

## 4. Risques & points d'attention

- **Frontière de fraîcheur** : un match récent doit toujours être servi par le live tant
  qu'il n'est pas dans le manifest. C'est le risque #1 (servir du périmé) → testé en Phase 3.
- **Dérivés ancrés (cf. §1bis)** : perf/LUSR/citations sont figés par match → inclus dans
  le snapshot via leurs vues `_latest`. Un recompute délibéré = nouvelle version, pas une
  mutation en place. Reste à confirmer en Phase 0 l'étendue de la sensibilité-frontière des
  sessions (zone "live" = matchs récents non snapshotés) ; `engagement_*` dépend de
  coefficients ré-estimés → traiter sa cadence de ré-estimation comme un événement de version.
- **`dominance_flag`** doit être traité (Phase 1.a) AVANT le snapshot, sinon `match_registry`
  n'est pas un fait immuable pur.
- **Multi-titre** : tout chemin via `PathResolver` ; snapshot par titre ; dégradation
  gracieuse si un titre n'a pas de snapshot (fallback live).
- **Coût disque** : snapshots additifs → prévoir une rétention (garder N versions).

---

## 5. Conformité (grille plan-review / delivery-checklist)

- Couches Go : producteur snapshot = `internal/ops/` ; adapter lecture = repo
  `internal/platform/duckdb/` ; aucun SQL inline en handler/service.
- Logging : `slog.InfoContext` (production snapshot, bascule lecture), `slog.ErrorContext`
  sur erreurs ; pas de `fmt.Println`.
- Tests par couche : migration `dominance_flag` (DuckDB `:memory:`), invariant (guard test),
  snapshot (dataset réaliste PVE+PVP), frontière de fraîcheur (Phase 3).
- Frontend : aucun impact (backend/données uniquement).
- `thought_log.md` : entrée à chaque phase.
- Done-def par phase : définie ci-dessus.

---

## 6. Séquencement recommandé

1. **Phase 0** (mesure) — décide de la suite.
2. **Phase 1** (hardening) — en parallèle, inconditionnel, livrable seul.
3. Si Phase 0 positive → **Phases 2 → 3 → 4** séquentielles, décision de rollout après Phase 3.

Une seule branche, commits par phase (règle "1 tâche = 1 branche, N commits").

---

## 7. Mise à jour [2026-06-25] — passe d'implémentation (ultracode, branche refactor/durabilite-snapshot-immuable)

Conception verrouillée par un swarm design + vérif-adversariale (9 agents) AVANT tout code.
Corrections de cap importantes (le code est la source autoritaire — "carte datée, pas vérité") :

- **Phase 0 (instrumentation) — LIVRÉE.** 5 signaux ajoutés au B-swap provider
  (`internal/platform/duckdb/sharedprovider/`) : `shared_provider_reader_stall_ns_total`
  (vrai stall lecteur côté `Get`, distinct du drain moteur `get_wait_ms_total`),
  `…_reader_delayed_total`, `…_rw_window_ms` (fenêtre RW stricte, **avg/max** — l'infra
  observability ne fait pas de p50/p95), `…_swap_failures_total{drain_timeout}`
  (désambiguïsé d'`acquire_writer`). Exposés sur `/debug/vars` + `Snapshot()`. Test dédié
  `reader_stall_metrics_test.go`. `go vet` clean, tests verts. C'est l'outil qui **gate
  la décision Phases 2-4** (lire ces compteurs en prod).

- **Phase 1.a (`dominance_flag` append-only) — ANNULÉE (tâche fantôme).** Prémisse FAUSSE :
  `dominance_flag` n'est pas sur `shared.match_registry` mais sur `player_match_enrichment`
  (player DB), **déjà append-only** depuis 2026-06-21 (`steps_player_append_only_match_enrichment.go`,
  `stage='dominance'`). Aucune écriture UPDATE indexée sur shared pour ce flag. Rien à migrer.
  (Le caveat "match_registry sauf dominance_flag" du présent plan est donc à ignorer.)

- **Phase 1.b (invariant durable-avant-progrès) — DÉJÀ CORRECT, descopé en refactor optionnel.**
  Seul vrai watermark cross-store = LUSR v2 (`player_skill_state_v2`), **déjà géré** par
  `canonicalGate` (skill_v2_shadow.go, fix 2026-06-07, 2 tests e2e). Le chemin base-enrichment
  (shared écrit avant player, 2 TX séparées) **n'est PAS une perte** : `ensurePlayerEnrichmentRows`
  (étape -2 post-sync) re-crée la baseline de tout match orphelin depuis `shared.match_participants`
  (correctif incident 2026-05-27) → **auto-réparant**. 1.b se réduit donc à extraire/nommer/tester
  le pattern `canonicalGate` existant (clarté + garde anti-régression), PAS un bug fix, PAS de
  consolidation de transaction cross-DB.

- **Phase 0 inventory (lecture) confirmé** : >95% des lectures shared sont immuable-pur ;
  meilleurs pilotes Phase 3 = MatchView / Home / Explorer.

Reste : déployer Phase 0 + mesurer en prod pour trancher Phases 2-4 ; 1.b (refactor de clarté) optionnel.

### Mise à jour [2026-06-25] — Phase 2 LIVRÉE en entier (sur directive utilisateur « faire le plan au complet »)

L'utilisateur a explicitement écarté le gating Phase-0-d'abord et demandé la feature complète. Phase 2 = **readiness marker (étapes 1-6) + producteur + monitoring (étapes 7-13)** livrés sur la branche.

- **Readiness marker** : `snapshot_ready_at`/`partial_reasons` append-only (stage `'snapshot'`) + prédicat pur `isMatchSnapshotReady` (10 cas) + `evaluateSnapshotReadiness` (post-sync étape 6, grâce 60j) + `CapWeaponKills`.
- **Producteur** (`internal/ops/snapshot*.go`) : `ProduceSnapshot` versionné (`vNNN…/` + `CURRENT.json` flip os.Rename atomique + manifest checksummé + rétention). Lit shared + chaque player DB en RO via `OpenReadForQuery` (zéro ATTACH), filtre aux matchs `snapshot_ready_at IS NOT NULL`.
- **Câblage** : Phase 6bis du cycle V2 (`v2.SnapshotProducer` + `WithSnapshotProducer`, nil-guard, best-effort), pont `sync.SnapshotCutter`, inconditionnel.
- **Monitoring** : métriques cut (enum fermé via sentinelles `ops.ErrSnapshot*`) + gauges backlog global-par-titre + section `AdminMonitoringOverview.Snapshot` (zéro I/O DuckDB).

**Déviations assumées vs §3 de ce plan** (documentées, justifiées échelle perso) :
1. **Full re-export change-gated** au lieu d'incrémental par-batch → **pas de compaction** (`compactMonth` non écrit : aurait été du code mort).
2. **Un fichier Parquet par table** (shared) / **par (table, joueur)** (dérivés) au lieu d'une **partition mensuelle** : à l'échelle perso (quelques milliers de matchs), le partitionnement mensuel ajoute de la complexité (globbing, comptage par fichier) pour un gain de pruning négligeable. Réintroductible en **Phase 3** si le profilage lecture le justifie.

### Mise à jour [2026-06-25] — Phase 3 LIVRÉE (lecture sur snapshot, pilote MatchView)

- **Couche de lecture** (`ops.OpenSnapshotForPlayer` / `OpenSnapshotShared`) : ouvre la version courante en DuckDB :memory: avec les vues read_parquet aux NOMS LIVE (faits shared + vues `v_gamertag_lookup` canonique / `v_match_full` / `v_killer_victim_full` / `v_weapon_kills` / `match_csrs_latest`). `ErrNoSnapshot` / `ErrSnapshotIncomplete` → fallback live. Producteur étendu (`xuid_aliases` + `weapon_kills` + `match_csrs`). **Test de fidélité** : snapshot == live.
- **Reader** (`sync.SnapshotPreferredSharedReader`) : snapshot-préféré + fallback live (pas de flag), cache versionné (garde 3 queriers), métriques `snapshot_read_served/live_fallback_total`.
- **Câblage = Option B SCOPED MatchView** (pilote du plan) : injecté uniquement sur `MatchViewRepo` (singleton par titre). Les 17 lectures shared de MatchView basculent sur le snapshot ; médias (SharedSocial) + lectures player (ReadDB) restent live. Câblage GLOBAL rejeté (audit exhaustif requis + risque de casse sans fallback). Métriques read dans `AdminMonitoringOverview.Snapshot`.

### Mise à jour [2026-06-25] — Phase 4 LIVRÉE (cutover GLOBAL des lectures shared)

Sur directive user « généralise direct ». Toutes les lectures shared de l'app sont désormais snapshot-préférées (fallback live), pas seulement MatchView.

- **Schéma shared COMPLET reconstruit** : `OpenSnapshotShared` matérialise les 8 tables de base depuis les Parquet puis recrée TOUTES les vues via les fonctions canoniques (`migration.ApplyResolutionViews` + `ApplyMvPlayerMatchesView`, zéro divergence) + DDL inline v_weapon_kills/match_csrs_latest. Producteur étendu (`weapon_kills`/`match_csrs` en RAW). Sûr par construction : schéma complet OU `ErrSnapshotIncomplete` → fallback live global.
- **Câblage GLOBAL** : `config.sharedReaderForTitle` enveloppe le SharedReader live de chaque titre via le hook `AppConfig.SnapshotReaderWrapper` (impl cmd/server, singleton par titre). Le pilote scoped MatchView (Phase 3) est reverté (subsumé).
- **OpenAPI** : `MonitoringSnapshotSummary` documenté (drift test vert).

### Mise à jour [2026-06-25] — vérification adversariale → GLOBAL REVERTÉ, scoped MatchView retenu

Revue ultracode (28 findings confirmés) : le câblage GLOBAL casse le **classement mondial** (`world_csr_leaderboard_latest`/`world_player_season_stats_latest` lues via le même SharedReader, absentes du snapshot → Catalog Error sans fallback ; données mondiales non-match-immutables). → **global abandonné**, retour au **scoped MatchView** (lectures shared 100% match-immutables, fidélité-testées). Remédiation : dead code dérivé retiré, logging dédié `logs/snapshot.log` + échecs silencieux loggés (fallback/incomplete/force-ready, negative-cache), `rows.Err()` propagé (plus de set ready tronqué silencieux), tests ajoutés (change-gate re-cut, rétention, métriques, fallback-incomplet).

**État livré** : **lecture-snapshot SCOPED MatchView** — sûre, monitorée, loggée. Un cutover GLOBAL sûr exigerait une reconstruction complete-by-construction de TOUT le schéma shared (export dynamique de toutes les tables shared + application des migrations sur la :memory) → phase future NON engagée.

**Reste** : **déploiement prod = décision utilisateur** (downtime ; merge main = auto-deploy). Au deploy : producteur peuple les snapshots → MatchView bascule (fallback live avant), mesurable `snapshot_read_served/live_fallback_total` + `shared_provider_reader_stall_ns_total`. Revert = redéployer le binaire précédent (additif, zéro migration destructive). Suivi non bloquant : régénérer `apps/web` generated-types quand le front consommera la section snapshot du monitoring.
