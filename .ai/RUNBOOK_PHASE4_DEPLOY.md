# Runbook — Déploiement Phase 4 Collect→Persist sur DB legacy

**Pour qui** : ops + devs qui déploient la branche `refactor/collect-persist` sur une instance LevelUp **existante** (DB héritée du mode legacy `LEVELUP_PERSIST_BATCH=0`).

**Pré-requis absolu** : la branche `refactor/collect-persist` doit être buildée + déployée sur la machine cible. Voir la liste de commits Phase 4 dans `thought_log.md`.

---

## 1. Pourquoi un runbook spécifique ?

Phase 4 (commits `14dfd135` → `157d80a8`) refactorise le path INSERT et le post-sync compute en batch INSERT-only. **Le default flip en Phase 4.7** (commit `82d0aa1a`) active le path batch sans flag.

**Limitation connue** : le path batch **n'élimine pas une corruption ART pré-existante** — il PRÉVIENT les futures. Une DB héritée a très probablement des entries ART orphelines dans 3 tables critiques :
- `shared.match_participants`
- `player_match_enrichment` (par joueur)
- `match_skill_rank` (par joueur)

Sans rebuild initial, le 1er cycle après upgrade va déclencher des `FATAL Error: Invalid Input Error: Failed to delete all rows from index. Only deleted 0 out of N rows`.

---

## 2. Procédure de déploiement

### Étape 1 — Backup préventif (5 min)

```bash
# Toujours backup avant rebuild ART (CTAS swap = atomic mais 0 réseau de sécurité)
./bin/levelup backup --gamertag <chaque_joueur>
# OU snapshot disque / VM
```

### Étape 2 — Arrêter le serveur (1 min)

```bash
# Le serveur doit être STOPPÉ pour que force_rebuild_art puisse ouvrir
# les DBs en RW exclusif (DuckDB refuse 2 ouvertures RW simultanées).
taskkill /F /IM levelup-api.exe  # Windows
# ou systemctl stop levelup-api  # Linux
```

### Étape 3 — Rebuild ART complet (1-2 min selon volume)

```bash
./bin/force_rebuild_art --all true
```

Le CLI rebuild **3 tables** :
- `shared.match_participants` (1 fois)
- `player_match_enrichment` (1 par player DB)
- `match_skill_rank` (1 par player DB)

Output attendu :
```
══ force_rebuild_art (shared) ══
Rows avant : 25216
Rows après : 25216
✓ Rebuild shared terminé sans perte.

--- Player : Madina97294 ---
Rows pre-rebuild : 1115
✓ Rebuild player (player_match_enrichment) terminé sans perte.
match_skill_rank rows : avant=1109 après=1109
✓ Rebuild player (match_skill_rank) terminé sans perte.
...
✓ Rebuild --all terminé.
```

**Critère go/no-go** : `avant == après` pour chaque table. Si divergence → backup restore + investiguer.

### Étape 4 — Démarrer le serveur (defaults Phase 4)

```bash
./bin/levelup-api.exe
# OU systemctl start levelup-api
```

Les defaults Phase 4 sont automatiquement actifs :
- `LEVELUP_PERSIST_BATCH != "0"` → batch INSERT-only path
- `LEVELUP_POSTSYNC_INSERT_ONLY != "0"` → post-sync 5 sites batch
- `LEVELUP_PERSIST_BATCH_ASYNC != "0"` → BatchQueue + WAL durable

Pour fallback en mode legacy (urgence rollback) :
```bash
LEVELUP_PERSIST_BATCH=0 ./bin/levelup-api.exe
```

### Étape 5 — Smoke test (10 min)

Observer les logs `logs/sync.log` sur les premiers cycles :
- ✅ `writeSessionAssignmentsBatch: sessions persistées (INSERT-only path) updated=N`
- ✅ `batchComputePerformanceScores: batch terminé`
- ✅ `upsertLUSRRatingsBatch: batch terminé (INSERT-only path)`
- ✅ `persist: BatchQueue activée (async path)` au boot
- ❌ **AUCUN** `FATAL Error: Invalid Input Error: Failed to delete all rows from index`

Si FATAL observé → 1 table ART qui n'a pas été rebuild correctement → re-run force_rebuild_art ciblé.

---

## 3. Validation empirique (référence)

Sur le dataset de test (4 joueurs : Chocoboflor 410 matchs, JGtm 832, Madina97294 1115, XxDaemonGamerxX 32) :

| Cycle | Pré-rebuild ART | Post-rebuild ART (pme only) | Post-rebuild ART (pme + msr) |
|---|---|---|---|
| 1 | 2/4 FATAL | — | — |
| 2 | — | sessions OK, LUSR FATAL | — |
| 3 | — | — | **4/4 OK, 0 FATAL** |
| 4-9 | — | — | **4/4 OK, 0 FATAL** (chaque) |

Total : 9 cycles consécutifs × 4 joueurs = **36 syncs / 0 FATAL ART** après rebuild complet.

Performance cycle 9 (sequentiel, post-revert acad4603 + flip defaults) :
- Chocoboflor : 14.7s
- JGtm : 70.5s
- Madina97294 : 87.9s
- XxDaemonGamerxX : 2.6s

---

## 4. Procédure de rollback (si incident)

Si Phase 4 cause un incident inattendu en prod :

```bash
# 1. Arrêter le serveur
taskkill /F /IM levelup-api.exe

# 2. Relancer avec flags legacy (path UPSERT direct, expose ART bug mais
#    comportement connu d'avant Phase 4)
LEVELUP_PERSIST_BATCH=0 LEVELUP_POSTSYNC_INSERT_ONLY=0 LEVELUP_PERSIST_BATCH_ASYNC=0 \
  ./bin/levelup-api.exe

# 3. Observer + alerter team
```

Note : le rollback fait revenir au comportement legacy d'avant Phase 4. Le bug ART **peut revenir** sur les UPDATE concurrents. Pour le contourner manuellement : `./bin/force_rebuild_art --all true` (sans avoir besoin de re-flipper les flags).

---

## 5. Items hors-scope ce runbook

- **Rollback structurel** (revert toute la branche) : `git revert <commits>` puis redeploy. Pas couvert ici.
- **Migration auth** (E.v2 / PR 2.5b) : transparente pour l'utilisateur, pas de procédure spécifique.
- **Données utilisateur** : le rebuild ART préserve toutes les rows (CTAS swap atomic). Aucun risque de perte.
