# Axe 3 — Template tests & logging

> **À remplir à l'identique** par Claude puis par ChatGPT. Ne pas modifier la structure.

## Métadonnées du passage

| Champ | Valeur |
|-------|--------|
| Auteur LLM | Claude \| ChatGPT (entourer) |
| Date du passage | `YYYY-MM-DD` |
| SHA Go | `xxxxxxx` |
| SHA React | `xxxxxxx` |
| Coverage Go globale mesurée | `XX.X %` |
| Coverage React globale mesurée | `XX.X %` |
| Durée de l'analyse | `Nh` |

## Synthèse exécutive (150 mots max)

> État général de la couverture, qualité des tests de non-régression, état du logging.

---

## A. Couverture unitaire Go

### A.1 Cible vs réalité (rappel Phase 10)

| Package | Cible Phase 10 | Coverage mesurée | Écart | Classif |
|---------|:--------------:|:----------------:|:-----:|:-------:|
| `internal/api/handlers/` | ≥ 75% | | | |
| `internal/api/middleware/` | ≥ 80% | | | |
| `internal/sync/` | ≥ 70% | | | |
| `internal/migration/` | ≥ 75% | | | |
| `internal/platform/duckdb/` | ≥ 70% | | | |
| `internal/validation/` | ≥ 70% | | | |
| `internal/analysis/` | (variable) | | | |
| `internal/service/` | (variable) | | | |
| `internal/ops/` | (variable) | | | |
| `internal/domain/` | (variable) | | | |
| **Global** | ≥ 70% | | | |

### A.2 Zones à 0% ou quasi-0%

| Fichier | Coverage | Criticité | Classif |
|---------|:--------:|-----------|:-------:|
| | | | |

### A.3 Qualité des tests (pas juste la quantité)

| Question | Réponse | Classif |
|----------|---------|:-------:|
| Tests table-driven généralisés ? | | |
| Fixtures DB partagées via `testutil/` ? | | |
| Tests d'intégration séparés des unitaires (tag build) ? | | |
| Mocks via interfaces ou librairie (gomock/mockery) ? | | |
| Tests de concurrence (goroutines) sur le write lease ? | | |
| Tests de migration idempotents (appliqués deux fois sans crash) ? | | |

---

## B. Couverture React

| Feature | Fichier test | Tests présents | Cas couverts | Cas manquants | Classif |
|---------|--------------|:--------------:|--------------|---------------|:-------:|
| Career | `CareerPage.test.tsx` | | | | |
| Explorer | `ExplorerPage.test.tsx` | | | | |
| Home | `HomePage.test.tsx` | | | | |
| Match history | `MatchHistoryPage.test.tsx` | | | | |
| Media | `MediaPage.test.tsx` | | | | |
| Settings | `SettingsPage.test.tsx` | | | | |
| Setup | `SetupPage.test.tsx` | | | | |
| Squad | `SquadPage.test.tsx` | | | | |
| Synthesis | `SynthesisPage.test.tsx` | | | | |
| Match view / Last match / Session compare / Citations / Timeseries / Changelog | (à lister) | | | | |

### B.1 Qualité des tests React

| Question | Réponse | Classif |
|----------|---------|:-------:|
| Testing Library / Vitest en place ? | | |
| MSW ou équivalent pour mock API ? | | |
| Tests d'accessibilité (axe-core) ? | | |
| Tests de snapshots maîtrisés (pas de snapshot géant) ? | | |

---

## C. Tests de non-régression (parité Python)

> L'objectif est que **tout scénario critique validé côté Python** ait un test équivalent côté Go ou React.

### C.1 Scénarios critiques issus de la version Python

| Scénario | Test Python (fichier) | Test Go/React équivalent | Présent ? | Classif |
|----------|----------------------|--------------------------|:---------:|:-------:|
| Sync delta sur joueur existant | | | | |
| Backfill LUSR + CSR sur match classé | | | | |
| Backfill citations PvE | | | | |
| Backfill weapon_kills + bit 18 posé | | | | |
| Détection bots (`xuid LIKE 'bid(%'`) | | | | |
| Write lease : 2 writers concurrents | | | | |
| Restore depuis backup corrompu | | | | |
| Home : computation battle pass | | | | |
| Career : calcul LUSR rolling window | | | | |
| Session compare : 2 sessions overlapping | | | | |
| Teammates trio : f2_xuid optionnel | | | | |
| Comeback badges narrative (v6.2) | | | | |
| i18n : résolution Accept-Language | | | | |
| Media : association média ↔ match | | | | |

### C.2 Golden values

| Algorithme | Golden Python | Golden Go | Parité bit-exact ? | Classif |
|------------|:-------------:|:---------:|:------------------:|:-------:|
| Performance score | | | | |
| LUSR | | | | |
| CSR | | | | |
| Sessions | | | | |
| Citations | | | | |
| Killer/victim | | | | |
| Weapon parser | | | | |

### C.3 Tests E2E Playwright (Sprint 36)

| Spec | Présente ? | Passe en CI ? | Classif |
|------|:----------:|:-------------:|:-------:|
| 15 specs Playwright référencées | | | |
| Onboarding E2E (auth → home) | | | |

---

## D. Logging & observabilité

### D.1 Points de log attendus (côté Go)

| Flux | Log présent ? | Niveau (DEBUG/INFO/WARN/ERROR) | Structuré (slog JSON) ? | Classif |
|------|:-------------:|:-------------------------------:|:----------------------:|:-------:|
| Démarrage serveur | | | | |
| Requête HTTP entrante (middleware) | | | | |
| Request ID propagé dans tous les logs | | | | |
| Début/fin sync joueur | | | | |
| Début/fin backfill (par flag) | | | | |
| Écriture DB (writes.go) — succès + échec | | | | |
| Acquisition/libération write lease | | | | |
| Timeout lease | | | | |
| Appel API Halo (succès/échec/retry) | | | | |
| Taxonomie d'erreur provider appliquée | | | | |
| Migration appliquée (steps_*.go) | | | | |
| Device code flow (init, poll, complete) | | | | |

### D.2 Logs trop verbeux / à nettoyer

| Emplacement | Volume | Suggestion | Classif |
|-------------|--------|------------|:-------:|
| | | | |

### D.3 Logs absents ou manquants sur flux critiques

| Flux | Quoi logger | Où l'ajouter | Classif |
|------|-------------|--------------|:-------:|
| | | | |

### D.4 Logging côté React

| Aspect | Observation | Classif |
|--------|-------------|:-------:|
| Console.log résiduels en prod ? | | |
| Error boundary avec télémétrie ? | | |
| Traçage requêtes API (request id round-trip) ? | | |

---

## E. CI & qualité d'exécution

| Aspect | Observation | Classif |
|--------|-------------|:-------:|
| Job coverage Go en CI (CGO activé) | | |
| Seuil CI à 70% effectif ? | | |
| Ratchet positif (pas de régression) ? | | |
| Temps d'exécution suite Go < 5 min local / < 10 min CI ? | | |
| Temps d'exécution suite React < 2 min ? | | |
| Flakiness observée ? | | |

---

## F. Récap classifications

| Niveau | Nombre d'items |
|--------|:--------------:|
| 🔴 Bloquant | |
| 🟠 Majeur | |
| 🟡 Mineur | |
| 🟢 Toléré | |

## G. Top 10 trous de couverture à combler en priorité

| # | Zone | Package/feature | Effort (S/M/L) | Impact |
|--:|------|-----------------|:--------------:|--------|
| 1 | | | | |
| ... | | | | |
| 10 | | | | |

## H. Observations libres

> Max 300 mots.

---

**Fin du template axe 3.**
