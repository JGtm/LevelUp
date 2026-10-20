# Phase 5 — Cleanup anti-ART (post-validation Phase 3)

**Pré-requis** : Phase 3 du runbook validée (10 cycles prod sans FATAL ART, default `LEVELUP_PERSIST_BATCH=1` flipé).
**Branche cible** : `cleanup/anti-art-removal` (depuis main une fois Phase 2 mergée).
**Effort estimé** : ~2-3h.

---

## Objectif

Supprimer le code de contournement de l'ART qui devient inutile dès lors que le path Collect → Persist est le défaut. Le legacy `insertFetchedMatch` + tous ses workarounds étaient là pour pallier le bug ART. Le path INSERT-only le supprime par construction.

**Critère go/no-go** : 10 cycles prod consécutifs avec `LEVELUP_PERSIST_BATCH=1` sans aucun `FATAL Error: Invalid Input Error: Failed to delete all rows from index`. Avant ça, **NE PAS** lancer cette phase — le legacy doit rester intact pour permettre un rollback.

---

## Items à supprimer

### 1. `singleflight` dans `internal/sync/writes.go::InsertParticipants`

**Statut** : ADR 0018 documente cette dépendance. Empiriquement testé 2026-05-23 — réduit la fréquence du bug ART sans l'éliminer.

**Action** :
- Retirer `participantsSF *singleflight.Group` package-level variable
- Retirer la closure `participantsSF.Do(key, ...)` dans `InsertParticipants`
- Garder `insertParticipantRow` direct (sans singleflight wrapping)
- Retirer les compteurs `singleflight_dedupe_total` (ou les laisser à 0 pour observabilité historique)

**Fichiers touchés** : `internal/sync/writes.go` (lignes ~109-135).

**Tests à valider** : `internal/sync` full pass après retrait. Le path legacy `insertFetchedMatch` ne sera plus utilisé en prod, donc peu de risque, mais conserver les tests pour la trace.

### 2. `CHECKPOINT` post-sync (Plan J)

**Statut** : ADR/commit `ae82901e` — testé en prod 2026-05-22, **inactif** car le FATAL ART crashait AVANT d'atteindre le CHECKPOINT.

**Action** :
- Retirer la fonction `runCheckpoint` dans `internal/sync/engine_postsync.go`
- Retirer l'appel dans le pipeline post-sync
- Retirer le compteur `checkpoint_post_sync_*` si présent

**Fichiers touchés** : `internal/sync/engine_postsync.go`.

### 3. `BootARTGuard` auto-heal automatique

**Statut** : ADR 0017 + commit `487eea4e`. Détecte et auto-heal la corruption ART au boot.

**Décision** : **GARDER la détection** (log WARNING utile pour observabilité), **RETIRER l'auto-heal automatique** (devient redondant).

**Action** :
- Conserver `BootARTGuard.Detect()` (log WARNING)
- Retirer `BootARTGuard.Heal()` automatique au boot
- Conserver la possibilité d'heal manuel via CLI (`cmd/force_rebuild_art/`)

**Fichiers touchés** : `internal/ops/boot_art_guard.go` (à confirmer le chemin exact).

### 4. `RebuildMatchParticipantsART` / `RebuildPlayerMatchEnrichmentART` runtime

**Statut** : Outils de migration runtime. Toujours utiles comme ops one-shot.

**Décision** : **GARDER comme outils ops one-shot** (déjà CLI `cmd/force_rebuild_art/`). Ne plus appeler depuis le code de prod.

**Action** : Retirer les call sites runtime éventuels (chercher `RebuildMatchParticipantsART(` dans le code).

### 5. `force_rebuild_art` CLI

**Décision** : **GARDER** comme outil ops manuel. Pas de changement.

### 6. Migrations UPDATE-then-INSERT (commit `acad4603`)

**Statut** : ADR/commit `acad4603` — patch Phase 2 stabilisation. Pattern `UPDATE WHERE pk = ? ; if RowsAffected==0 then INSERT`.

**Décision** : **REVERT** le commit OU réécrire `insertParticipantRow` en INSERT pur (le legacy insertFetchedMatch ne sera plus exécuté en prod après le flip default).

**Action**:
- Option A — revert direct du commit `acad4603` (5 migrations touchées)
- Option B — réécrire `insertParticipantRow` en INSERT-only (cf. submitMatchAsBatch pour le pattern)

Recommander Option A pour simplicité.

### 7. Bits MBit* / PBit* (à confirmer)

**Statut** : Les bitmasks (`backfill_completed`, `backfill_bits`, `pve_bits`) sont actuellement positionnés par `Mark*` UPDATE post-INSERT dans le legacy. Le path batch les positionne à l'INSERT (cf. `MatchRegistryRow.BackfillCompleted` etc., champs pointer).

**Décision** : **GARDER** la sémantique bits (utiles aux backfills heal). Mais les Mark* fonctions ne seront plus appelées en path batch.

**Action** : Aucune. Les Mark* legacy restent appelables pour les backfill CLI (cmd/backfill_all qui n'utilise pas submitMatchAsBatch).

---

## Pattern de test

Pour chaque suppression :
1. Vérifier que `go test ./...` passe encore (le legacy path est testé même s'il n'est plus utilisé en prod)
2. `go vet ./...` clean
3. Pas de compteur expvar référencé qui n'existe plus côté front (chercher dans `apps/web/` les usages)

---

## Garde-fous

- **NE PAS supprimer** `insertFetchedMatch` ni les fonctions qu'elle appelle (`InsertRegistryIfNotExists`, `InsertParticipants`, etc.) — le path legacy doit rester appelable pour rollback en cas d'incident post-Phase 3.
- **NE PAS supprimer** le feature flag `WithBatchPersistMode` ni la lecture env var `LEVELUP_PERSIST_BATCH` — utile pour basculer en cas d'incident.
- **GARDER** ADR 0017 et 0018 dans le repo (marqués obsolètes) pour la trace historique.

---

## Suivi

Une fois Phase 5 livrée :
- ✅ Update `INCIDENT_ART_CORRUPTION_DUCKDB.md` status → CLOSED.
- ✅ Update ADR 0017 status → CLOSED (replaced by 0019).
- ✅ Update ADR 0018 status → CLOSED (replaced by 0019).
- ✅ Mention dans le CHANGELOG / release notes.
