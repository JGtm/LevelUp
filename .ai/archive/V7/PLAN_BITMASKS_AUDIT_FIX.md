# PLAN — Bitmasks audit + fix des dead writes

> **Date d'ouverture** : 2026-05-08
> **Branche cible** : `feat/token-pool-parallel-sync` (continuité — la branche est stable après PLAN_HIGHLIGHT_EVENTS_BACKFILL livré + plan jumeau complété)
> **Statut** : à valider — plan en attente d'exécution
> **Référence amont** : `.ai/thought_log.md` entrée [2026-05-08] "Audit lecture seule des bitmasks de sync (Phase 1ter)" + entrée [2026-05-08] "PLAN_HIGHLIGHT_EVENTS_BACKFILL — implémentation complète"

---

## 1. Contexte

### 1.1 État découvert

Trois jeux de bits coexistent dans la même colonne `match_registry.backfill_completed` (et `match_participants.backfill_bits`) :

| Famille | Bits | Origine | Écrit en prod ? |
|---|---|---|---|
| `BackfillFlags` legacy (`backfill_flags.go::BackfillFlags`) | `mr.backfill_completed` 1<<0..1<<14 | port Python | **❌ aucun** |
| `MBit*` | `mr.backfill_completed` 1<<16..1<<22 | convention Go récente | seuls 4/7 écrits |
| `PBit*` | `mp.backfill_bits` 1<<0..1<<18 | granularité per-player | **❌ aucun** |

Sur les **21 bits** déclarés :
- **4 effectivement utilisés** : `MBitEvents`, `MBitKillerVictim`, `MBitWeaponKills`, `MBitWeaponKillsNoFilm`
- **17 du folklore** : déclarés et parfois lus, mais jamais écrits → leur filtre detection match TOUS les matchs

### 1.2 Filtres effectivement cassés

Le READ-side a été audité dans `internal/sync/backfill.go::findMatchesInSharedAll`. Trois filtres reposent **exclusivement** sur des bits jamais écrits, sans fallback colonne :

| Filtre | SQL detect | Conséquence |
|---|---|---|
| **`--skill`** | `(mp.backfill_bits & 1) = 0 AND (mr.backfill_completed & 4) = 0` (lignes 256-257) | re-fetch `GetMatchSkill` pour TOUS les matchs à chaque run |
| **`--participants`** | `1=1 + doneGuard("participants", 1<<9)` (ligne 335) | re-traitement participants pour tous (impact moindre car `InsertParticipants` est OR REPLACE idempotent) |
| **`--PVE stats`** (firefights) | `mr.is_firefight=TRUE AND (mr.backfill_completed & MBitPVEStats) = 0` (ligne 346) | re-fetch PvE stats pour tous les firefights |

Les autres filtres ont soit un guard column-based (`mp.accuracy IS NULL`, `mp.shots_fired IS NULL`, etc.), soit un guard table-based (`NOT IN medals_earned`, `playerDoneGuard()`), soit s'appuient sur des bits effectivement écrits (`MBitWeaponKills`/`NoFilm`).

### 1.3 Impact mesuré

Coût API pour un user normal (~25 matchs/jour) :
- skill : ~25 appels `GetMatchSkill` par sync, dont ~24 inutiles → ~600 appels/mois gaspillés
- PVE : ne s'applique que sur les firefights, marginal
- participants : marginal (déjà inséré)

Pour un user power (~200 matchs/jour) ou un backfill historique : le ratio empire (chaque `--skill` retraite tout l'historique).

**Pas une corruption de données — un coût API gaspillé + temps de traitement.**

### 1.4 Folklore à éliminer

Constantes déclarées mais entièrement non-utilisées (ni READ ni WRITE) :
- `MBitAssets` (1<<17), `MBitAliases` (1<<18) — `MarkAssetsLoaded` / `MarkAliasesLoaded` n'existent pas
- `MarkPveStatsDone` existe (`pve.go:287`) mais n'a aucun caller en prod
- `BackfillFlags["medals"]`, `["events"]`, `["personal_scores"]`, `["accuracy"]`, `["shots"]`, `["enemy_mmr"]`, `["aliases"]` — tous lus uniquement via `doneGuard()` qui retourne string vide ou guard sur un bit jamais écrit

### 1.5 Hors-scope

- Les fixes events+kvp déjà livrés en `PLAN_HIGHLIGHT_EVENTS_BACKFILL.md::Phase 1bis` ne sont PAS retouchés ici.
- Les RC4/5/6 du plan jumeau (i18n, alias backfill cross-match, graceful 404) sont indépendants.

---

## 2. Objectifs et critères de succès

| Critère | Mesure |
|---|---|
| Tous les filtres detection ont une garde efficace | `findMatchesInSharedAll` audit table verte (chaque branche → soit bit écrit en prod, soit fallback colonne) |
| Aucun bit déclaré mais jamais lu ni écrit | grep `MBitX` ou `PBitX` dans `internal/sync/` retourne au moins une utilisation effective |
| Test anti-dead-write | nouveau test scanne le source et fail si un bit est défini sans aucun call-site WRITE/READ |
| Reset post-deploy contrôlé | one-shot SQL fenêtré (matchs où la colonne sous-jacente est non-NULL → mark le bit) ; documenté dans le plan |
| Aucune régression | `go test ./...` 100% pass + `go vet` clean |

---

## 3. Phases

### Phase 1 — Audit terrain empirique

**Effort** : ~30 min
**Livrable indépendant** : oui (lecture seule)

Une sonde mesure sur la prod combien de matchs seraient effectivement re-traités par chaque filtre cassé. Permet de valider l'analyse et de prioriser.

**Fichier nouveau** : `cmd/diag_bitmask_coverage/main.go` (~150 L) — read-only inspection sur shared_matches_v2.duckdb :

```sql
-- skill : matchs avec team_mmr déjà rempli mais bit non set
SELECT COUNT(*) FROM match_participants mp
WHERE mp.team_mmr IS NOT NULL
  AND (COALESCE(mp.backfill_bits, 0) & 1) = 0;

-- participants : tous les matchs sont à 1=1
-- (pas mesurable directement, marginalement intéressant)

-- PVE stats : firefights avec lignes pve_match_stats déjà présentes mais bit non set
SELECT COUNT(*) FROM match_registry mr
WHERE mr.is_firefight = TRUE
  AND EXISTS (SELECT 1 FROM shared_pve.pve_match_stats WHERE match_id = mr.match_id)
  AND (COALESCE(mr.backfill_completed, 0) & 1048576) = 0;
```

**Critère de complétion** : tableau de chiffres prod ajouté à `thought_log.md`. Ces chiffres serviront de baseline pour valider le reset Phase 4.

---

### Phase 2 — Fix par cas (écriture des bits manquants)

**Effort** : ~1 h
**Livrable indépendant** : oui (chaque cas indépendant)
**Pourquoi avant Phase 3** : on écrit les bits AVANT de retirer les constantes orphelines, sinon on perd les définitions.

#### 2.a Skill — `PBitTeamMMR` + `mr.backfill_completed bit 2`

Au moment de l'appel `MergeSkillIntoParticipants` (où l'API `GetMatchSkill` a renvoyé team_mmr/enemy_mmr/kills_expected), positionner :
- `mp.backfill_bits |= PBitTeamMMR | PBitEnemyMMR | PBitKillsExp | PBitDeathsExp` pour chaque participant updated
- `mr.backfill_completed |= 4` (bit 2, "skill" legacy)

**Fonction nouvelle** : `MarkSkillLoaded(db *sql.DB, matchID string, xuids []string)` dans `internal/sync/writes.go`. UPDATE participant + UPDATE registry.

**Call-site** : `engine.go::processFetchedMatch` après `MergeSkillIntoParticipants`, conditional sur la skill data réellement présente (pas de mark menteur sur `skillData=empty`).

**Idempotent** : `|=` est commutatif, donc safe à re-exécuter.

#### 2.b Participants — `mr.backfill_completed bit 9`

Après `InsertParticipants` réussi, positionner `mr.backfill_completed |= 1<<9` (`BackfillFlags["participants"]`).

**Fonction nouvelle** : `MarkParticipantsDone(db *sql.DB, matchID string)` dans `internal/sync/writes.go`.

**Call-site** : `engine.go::processFetchedMatch` après l'insert participants succès.

#### 2.c PVE stats — appeler `MarkPveStatsDone` (déjà existante)

`pve.go:287` définit `MarkPveStatsDone` mais aucun caller. Ajouter l'appel dans la pipeline PvE après `InsertPveMatchStats` succès.

**Recherche d'abord** : où `InsertPveMatchStats` est appelée pour identifier le bon call-site.

#### 2.d Tests

`bitmask_honesty_test.go` étendu :
- `TestMarkSkillLoaded_ConditionalOnInsertSuccess` (mock DB qui fait échouer Update → bit reste 0)
- `TestMarkParticipantsDone_ConditionalOnInsertSuccess`
- `TestMarkPveStatsDone_CalledAfterInsert` (vérifier que le call-site fait bien le mark)

**Critère de complétion** : 3 nouvelles fonctions Mark + 3 call-sites + tests verts.

---

### Phase 3 — Cleanup du folklore

**Effort** : ~30-45 min
**Livrable indépendant** : oui (post-Phase 2 pour ne pas casser les WRITE)

Suppression des constantes orphelines :

- `MBitAssets`, `MBitAliases` : aucune utilisation. Supprimer + commentaire dans `backfill_flags.go` indiquant "réservé pour futur usage" ou simplement les retirer.
- `MarkPveStatsDone` : maintenant utilisée (Phase 2.c) → reste en place. Annuler l'item "DEAD CODE" du verdict initial.
- `BackfillFlags` map : entrées non utilisées en READ → marquer chacune avec un commentaire ou retirer les inutiles. Garder uniquement celles consommées par `doneGuard()`.

**Décision case-by-case** :
- Si une constante n'est plus référencée nulle part → suppression
- Si elle est référencée par un test → décision : utile ou meta-test ?
- Si commentaire-de-future-usage → suppression (le folklore augmente la dette)

**Test ajouté** : `TestNoDeadBitDeclaration` dans `bitmask_honesty_test.go` — scanne les sources Go pour trouver des `Mbit*` ou `PBit*` ou `BackfillFlags["..."]` déclarés mais jamais utilisés ailleurs. **Garde-fou**.

**Critère de complétion** : suppressions appliquées + nouveau test vert.

---

### Phase 4 — Reset post-deploy contrôlé

**Effort** : ~30 min
**Livrable indépendant** : conditionne le déploiement Phase 2

Une fois Phase 2 livrée et déployée, le sync va commencer à écrire les bits sur les NOUVEAUX matchs. Mais l'historique est tout à 0 → le premier `levelup backfill --skill --participants` va re-traiter tout l'historique (comportement actuel).

**One-shot SQL** : positionner les bits sur les matchs déjà OK :

```sql
-- Skill : marquer les participants avec team_mmr déjà rempli + le registry
UPDATE match_participants
SET backfill_bits = COALESCE(backfill_bits, 0) | 15  -- PBitTeamMMR|PBitEnemyMMR|PBitKillsExp|PBitDeathsExp
WHERE team_mmr IS NOT NULL;

UPDATE match_registry
SET backfill_completed = COALESCE(backfill_completed, 0) | 4  -- BackfillFlags["skill"]
WHERE EXISTS (
  SELECT 1 FROM match_participants mp
  WHERE mp.match_id = match_registry.match_id AND mp.team_mmr IS NOT NULL
);

-- Participants : marquer les matchs avec ≥ 1 participant inséré
UPDATE match_registry
SET backfill_completed = COALESCE(backfill_completed, 0) | 512  -- BackfillFlags["participants"]
WHERE EXISTS (SELECT 1 FROM match_participants WHERE match_id = match_registry.match_id);

-- PVE stats : marquer les firefights avec ≥ 1 ligne pve_match_stats
UPDATE match_registry
SET backfill_completed = COALESCE(backfill_completed, 0) | 1048576  -- MBitPVEStats
WHERE is_firefight = TRUE
  AND EXISTS (SELECT 1 FROM shared_pve.pve_match_stats WHERE match_id = match_registry.match_id);
```

**Implémentation** : nouvelle sous-commande `levelup reset-bitmasks --dry-run|--apply` dans `cmd/levelup/cmd_reset_bitmasks.go`. Idempotent. Affiche les chiffres avant/après.

**Critère de complétion** : commande implémentée, dry-run aligné avec les chiffres Phase 1, --apply OK une fois (ou différé si user préfère ne pas modifier la prod immédiatement).

---

### Phase 6 — Étendre le garde-fou pour la map `BackfillFlags`

**Effort** : ~30 min (ajouté après livraison Phase 5 sur question utilisateur)
**Livrable indépendant** : oui

**Contexte** : Le test `TestNoDeadBitDeclaration` (Phase 3) parse les **constantes** `MBit*`/`PBit*`/`PveBit*` mais ignore la map `BackfillFlags map[string]int`. Une key orpheline ajoutée demain ne déclencherait pas le garde-fou.

Trois catégories de keys découvertes :
- **9 keys héritage Python non consommées en Go** (`medals`, `events`, `skill`, `personal_scores`, `accuracy`, `shots`, `enemy_mmr`, `aliases`, `weapon_kills`) — la detection Go utilise des column/table guards directs, pas la map. Ces bits sont quand même positionnés en DB par l'ancien code Python sur l'historique.
- **7 keys consommées via `doneGuard()`** (`assets`, `participants`, `participants_scores`, `participants_kda`, `participants_shots`, `participants_damage`, `participants_avg_life`) — le filtre column principal fait le job, le bit est un fast-skip cosmétique tant qu'il n'est pas écrit.
- **1 key écrite par Phase 2** (`participants` via `MarkParticipantsDone`).

**Fichier modifié** : `internal/sync/bitmask_dead_declarations_test.go`

Ajout de :
- `TestNoDeadBackfillFlagKey` : extrait les keys via regex sur le bloc `BackfillFlags = map[string]int{...}`, vérifie pour chacune une occurrence de `"key"` dans un `.go` du package (couvre `doneGuard("key", ...)`, `ComputeBackfillMask("key", ...)`, et toute autre référence textuelle). Whitelist explicite des 9 keys héritage avec justification écrite.
- `extractBackfillFlagsKeys(path)` helper.

**Sanity check** : ajout temporaire d'une key bidon `"this_should_fail_test"` → test fail comme attendu → key retirée → test repasse vert.

**Critère de complétion** :
- `TestNoDeadBackfillFlagKey` vert
- `TestNoDeadBitDeclaration` continue de passer (régression)
- Whitelist documente la raison de chaque exception

---

### Phase 5 — Vérifs finales + thought_log

**Effort** : ~15 min

| Check | Méthode |
|---|---|
| `go build ./...` | passe |
| `go test ./...` | 100% |
| `go vet ./...` | clean |
| Test anti-dead-write | `TestNoDeadBitDeclaration` vert |
| Audit table READ → impact réel | tableau revisité, tous les filtres ont une garde |
| Re-exécution Phase 1 sonde après Phase 4 | les chiffres "matchs avec garde absente" tombent à 0 |
| Entrée `thought_log.md` `[2026-05-08]` | clôture du plan, mention des commits |

---

## 4. Découpage en commits

| # | Phase | Message |
|---|---|---|
| 1 | Phase 1 | `feat(diag): add diag_bitmask_coverage tool + audit prod findings` |
| 2 | Phase 2 | `fix(sync): write skill + participants + PVE stats bitmasks at insert` |
| 3 | Phase 3 | `refactor(sync): remove orphan bitmask constants + add anti-dead-write test` |
| 4 | Phase 4 | `feat(cli): add levelup reset-bitmasks for one-shot rétroactif` |
| 5 | Phase 5 | `docs(ai): thought_log — clôture PLAN_BITMASKS_AUDIT_FIX` |

Chaque commit livrable indépendamment ; `go test ./...` reste vert entre chaque.

---

## 5. Architecture — checks plan-review

| Check | État |
|---|---|
| Algos purs dans `internal/analysis/` | N/A — pas de nouvel algo |
| Types résultat dans `domain/` ou `canonical/` | N/A |
| Orchestration dans `internal/service/` | N/A — sync-spécifique reste dans `internal/sync/` |
| Handlers HTTP | N/A — pas de touche API HTTP |
| Aucun SQL dans handler / service | Oui — tout dans writes.go ou cmd/ |
| Multi-titres : `PathResolver` | Phase 1 et Phase 4 utilisent `PathResolver` pour shared/pve DB |
| Multi-titres : `HasCapability()` | PvE est une capability ; Phase 2.c doit vérifier `HasCapability("pve")` avant d'appeler `MarkPveStatsDone` (sinon erreur sur titres sans firefight) |
| Tests à chaque couche | sync (Phase 2 + 3) ; CLI (smoke test Phase 4) ; pas de handler |
| Logging via `slog` | Oui — `slog.WarnContext` sur erreur de Mark |
| Frontend impacté | Non |

---

## 6. Risques et hors-scope

### Risques identifiés

| Risque | Mitigation |
|---|---|
| Phase 4 reset : un match avec `team_mmr` rempli mais `kills_expected` manquant verra le PBit positionné à tort | Reset SQL utilise `WHERE team_mmr IS NOT NULL` — c'est une approximation. Documenter dans le commit que le bit signifie "tentative effectuée" pas "100% rempli" |
| `MarkSkillLoaded` appelé sur un mock où l'API a renvoyé skill data vide | Conditional sur `len(skillData) > 0` dans le call-site (déjà le pattern de processFetchedMatch) |
| Phase 3 cleanup : suppression d'une constante encore référencée par un test futur | Test `TestNoDeadBitDeclaration` est aussi un fail-safe : il fail si une constante orpheline ré-apparaît |
| Concurrence : un sync simultané pendant le reset Phase 4 | `levelup reset-bitmasks` acquiert le lease shared, comme les autres outils |

### Hors-scope explicite

- Refonte du système bitmask vers du column-based pur (option β des audits initiaux) — décidé en faveur de l'option α (compléter le mécanisme) car l'investissement WRITE est faible et les guards bit sont plus rapides que les guards colonne sur de gros volumes
- Migration vers `match_registry.skill_loaded BOOLEAN` ou similaire (parallèle à `events_loaded`) — overkill pour le bénéfice
- Performance benchmarks après le fix (mesurer le gain réel d'API saved) — peut être ajouté à thought_log post-Phase 5 si la curiosité prime

---

## 7. Effort total estimé

| Phase | Effort |
|---|---|
| Phase 1 — sonde audit terrain | 30 min |
| Phase 2 — fix par cas + tests | 1 h |
| Phase 3 — cleanup folklore + test anti-dead | 30-45 min |
| Phase 4 — CLI reset-bitmasks | 30 min |
| Phase 5 — vérifs + thought_log | 15 min |
| **Total** | **~3 h** |

---

## 8. Done definition globale

- [ ] Phase 1 : `cmd/diag_bitmask_coverage` opérationnel + tableau prod dans thought_log
- [ ] Phase 2 : 3 nouvelles fonctions `Mark*` + 3 call-sites dans `engine.go::processFetchedMatch` + 3 tests honesty
- [ ] Phase 3 : constantes orphelines supprimées + `TestNoDeadBitDeclaration` vert
- [ ] Phase 4 : `levelup reset-bitmasks` opérationnel (--dry-run + --apply)
- [ ] Phase 5 : `go test ./...` clean + thought_log entrée `[2026-05-08]`

---

## 9. Référence croisée

- Plan parent : `.ai/PLAN_HIGHLIGHT_EVENTS_BACKFILL.md` (Phase 1ter audit lecture seule)
- Plan jumeau : `.ai/PLAN_RECENT_MATCH_REGRESSION_FIX.md` — Phases A/B/C indépendantes (status : complété par autre agent)
- Commits amont : `b6b31062` (audit Phase 1ter qui a découvert le folklore)
- Constantes touchées : `internal/sync/backfill_flags.go::{MBitAssets, MBitAliases, BackfillFlags map, PBit*}`
- Fonctions touchées : `internal/sync/writes.go::Mark*`, `internal/sync/pve.go::MarkPveStatsDone`
