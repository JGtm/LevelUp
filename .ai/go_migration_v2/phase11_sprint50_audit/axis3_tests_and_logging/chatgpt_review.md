# Axe 3 · Review ChatGPT — Tests & logging

## Métadonnées du passage

| Champ | Valeur |
|-------|--------|
| Auteur LLM | ChatGPT |
| Date du passage | `2026-04-18` |
| SHA Go | `93c3cd66` |
| SHA React | `93c3cd66` |
| Coverage Go globale mesurée | `78.8 %` (`go tool cover -func=coverage.out`) |
| Coverage React globale mesurée | `non mesurable dans cette passe` |
| Durée de l'analyse | `session courante` |

## Synthèse exécutive (150 mots max)

Le backend Go dispose d'un socle de tests dense sur la branche courante : 30 fichiers de tests handlers, 12 fichiers middleware, tests de contrat YAML et 16 specs Playwright côté web. Le workflow CI actuel a aussi été durci : `.github/workflows/ci.yml` exécute une couverture complète CGO avec `-coverpkg=./...`, puis ratchette contre `apps/go-api/coverage_baseline.txt`, aujourd'hui à `76.0`.

Le point faible confirmé est désormais le frontend unitaire mesuré : Vitest est bien en place, MSW aussi, mais la mesure de couverture échoue immédiatement car `@vitest/coverage-v8` n'est pas installé comme dépendance directe. On peut donc constater l'existence de tests frontend, mais pas prouver une couverture React >= 70 % sur cette passe.

---

## A. Couverture unitaire Go

### A.1 Cible vs réalité (rappel Phase 10)

| Package | Cible Phase 10 | Coverage mesurée | Écart | Classif |
|---------|:--------------:|:----------------:|:-----:|:-------:|
| `internal/api/handlers/` | ≥ 75% | `75.4 %` (profil `coverage.out` agrégé) | atteint | 🟢 |
| `internal/api/middleware/` | ≥ 80% | `84.6 %` (profil `coverage.out` agrégé) | atteint | 🟢 |
| `internal/sync/` | ≥ 70% | `non mesuré dans l'artefact actuel` | preuve absente | 🟡 |
| `internal/migration/` | ≥ 75% | `81.1 %` (profil `coverage.out` agrégé) | atteint | 🟢 |
| `internal/platform/duckdb/` | ≥ 70% | `75.4 %` (profil `coverage.out` agrégé) | atteint | 🟢 |
| `internal/validation/` | ≥ 70% | `88.4 %` (profil `coverage.out` agrégé) | atteint | 🟢 |
| `internal/analysis/` | (variable) | `non mesuré dans l'artefact actuel` | preuve absente | 🟡 |
| `internal/service/` | (variable) | `non mesuré dans l'artefact actuel` | preuve absente | 🟡 |
| `internal/ops/` | (variable) | `non mesuré dans l'artefact actuel` | preuve absente | 🟡 |
| `internal/domain/` | (variable) | `non mesuré dans l'artefact actuel` | preuve absente | 🟡 |
| **Global** | ≥ 70% | `78.8 %` (`coverage.out`) | atteint | 🟢 |

### A.2 Zones à 0% ou quasi-0%

| Fichier | Coverage | Criticité | Classif |
|---------|:--------:|-----------|:-------:|
| `apps/go-api/internal/api/handlers/auth.go:136` (`pollDeviceFlow`) | `0.0 %` | auth onboarding | 🟠 |
| `apps/go-api/internal/api/handlers/match_view.go:21` (`NewMatchViewHandler`) | `0.0 %` | faible isolément | 🟡 |
| `apps/go-api/internal/api/handlers/media.go:146` (`parseUploadedFiles`) | `0.0 %` | upload média | 🟡 |
| `apps/go-api/internal/migration/steps_player.go:317` (`applyFixMvSessionStats`) | `0.0 %` | migration | 🟡 |
| `apps/go-api/internal/platform/duckdb/pool.go:56`, `:94`, `:141`, `:160` | `0.0 %` | ouverture/attach DB | 🟠 |

### A.3 Qualité des tests (pas juste la quantité)

| Question | Réponse | Classif |
|----------|---------|:-------:|
| Tests table-driven généralisés ? | Présents dans plusieurs paquets, mais non audités exhaustivement ici. | 🟢 |
| Fixtures DB partagées via `testutil/` ? | Oui, existence de `internal/sync/testutil/` documentée par le repo et la stratégie coverage. | 🟢 |
| Tests d'intégration séparés des unitaires (tag build) ? | Oui côté CI : `-tags=integration` dans `.github/workflows/ci.yml:250-256`. | 🟢 |
| Mocks via interfaces ou librairie (gomock/mockery) ? | Côté frontend : MSW. Côté Go : plutôt helpers/httptest que framework de mocks. | 🟢 |
| Tests de concurrence (goroutines) sur le write lease ? | Non revérifié directement dans cette passe. | 🟡 |
| Tests de migration idempotents (appliqués deux fois sans crash) ? | Le repo contient des tests migration et une couverture mesurée sur `internal/migration`, mais l'idempotence n'a pas été rerun ici. | 🟡 |

---

## B. Couverture React

| Feature | Fichier test | Tests présents | Cas couverts | Cas manquants | Classif |
|---------|--------------|:--------------:|--------------|---------------|:-------:|
| Career | `apps/web/src/features/career/CareerPage.test.tsx` | ✅ | smoke + rendu principal | couverture chiffrée absente | 🟢 |
| Explorer | `apps/web/src/features/explorer/ExplorerPage.test.tsx` | ✅ | smoke + rendu principal | couverture chiffrée absente | 🟢 |
| Home | `apps/web/src/features/home/HomePage.test.tsx` | ✅ | spinner + KPIs + nom joueur | couverture chiffrée absente | 🟢 |
| Match history | `apps/web/src/features/match-history/MatchHistoryPage.test.tsx` | ✅ | rendu principal | couverture chiffrée absente | 🟢 |
| Media | `apps/web/src/features/media/MediaPage.test.tsx` | ✅ | rendu principal | couverture chiffrée absente | 🟢 |
| Settings | `apps/web/src/features/settings/SettingsPage.test.tsx` | ✅ | rendu principal | couverture chiffrée absente | 🟢 |
| Setup | `apps/web/src/features/setup/SetupPage.test.tsx` | ✅ | étapes auth de base | couverture chiffrée absente | 🟢 |
| Squad | `apps/web/src/features/squad/SquadPage.test.tsx` | ✅ | rendu principal | couverture chiffrée absente | 🟢 |
| Synthesis | `apps/web/src/features/synthesis/SynthesisPage.test.tsx` | ✅ | rendu principal | couverture chiffrée absente | 🟢 |
| Match view / Last match / Session compare / Citations / Timeseries / Changelog | `aucun test unitaire dédié trouvé` | ❌ | Playwright seulement pour plusieurs de ces pages | tests unitaires absents | 🟠 |

### B.1 Qualité des tests React

| Question | Réponse | Classif |
|----------|---------|:-------:|
| Testing Library / Vitest en place ? | Oui : `apps/web/package.json` + `apps/web/vite.config.ts:27-34`. | 🟢 |
| MSW ou équivalent pour mock API ? | Oui : `apps/web/src/test/setup.ts:6-18` et `apps/web/src/test/handlers.ts`. | 🟢 |
| Tests d'accessibilité (axe-core) ? | Non trouvés dans cette passe. | 🟡 |
| Tests de snapshots maîtrisés (pas de snapshot géant) ? | Aucun snapshot significatif trouvé dans cette passe. | 🟢 |

---

## C. Tests de non-régression (parité Python)

### C.1 Scénarios critiques issus de la version Python

| Scénario | Test Python (fichier) | Test Go/React équivalent | Présent ? | Classif |
|----------|----------------------|--------------------------|:---------:|:-------:|
| Sync delta sur joueur existant | non recherché ici | non vérifié | ⚪ | 🟡 |
| Backfill LUSR + CSR sur match classé | non recherché ici | non vérifié | ⚪ | 🟡 |
| Backfill citations PvE | non recherché ici | non vérifié | ⚪ | 🟡 |
| Backfill weapon_kills + bit 18 posé | non recherché ici | non vérifié | ⚪ | 🟡 |
| Détection bots (`xuid LIKE 'bid(%'`) | non recherché ici | non vérifié | ⚪ | 🟡 |
| Write lease : 2 writers concurrents | non recherché ici | non vérifié | ⚪ | 🟡 |
| Restore depuis backup corrompu | non recherché ici | non vérifié | ⚪ | 🟡 |
| Home : computation battle pass | non recherché ici | `HomePage.test.tsx` couvre le rendu mais pas la non-régression métier complète | partiel | 🟡 |
| Career : calcul LUSR rolling window | non recherché ici | non vérifié | ⚪ | 🟡 |
| Session compare : 2 sessions overlapping | non recherché ici | spec Playwright présente | partiel | 🟡 |
| Teammates trio : f2_xuid optionnel | non recherché ici | non vérifié | ⚪ | 🟡 |
| Comeback badges narrative (v6.2) | non recherché ici | non vérifié | ⚪ | 🟡 |
| i18n : résolution Accept-Language | non recherché ici | non vérifié | ⚪ | 🟡 |
| Media : association média ↔ match | non recherché ici | tests page / handler présents, non audités en profondeur | partiel | 🟡 |

### C.2 Golden values

| Algorithme | Golden Python | Golden Go | Parité bit-exact ? | Classif |
|------------|:-------------:|:---------:|:------------------:|:-------:|
| Performance score | non relu ici | non relu ici | non vérifié | 🟡 |
| LUSR | non relu ici | non relu ici | non vérifié | 🟡 |
| CSR | non relu ici | non relu ici | non vérifié | 🟡 |
| Sessions | non relu ici | non relu ici | non vérifié | 🟡 |
| Citations | non relu ici | non relu ici | non vérifié | 🟡 |
| Killer/victim | non relu ici | non relu ici | non vérifié | 🟡 |
| Weapon parser | non relu ici | non relu ici | non vérifié | 🟡 |

### C.3 Tests E2E Playwright (Sprint 36)

| Spec | Présente ? | Passe en CI ? | Classif |
|------|:----------:|:-------------:|:-------:|
| 16 specs Playwright référencées | ✅ | workflow `e2e-react` présent dans `.github/workflows/ci.yml:380-456` | 🟢 |
| Onboarding E2E (auth → home) | ✅ (`slice-9-onboarding.spec.ts`) | workflow présent, non relancé ici | 🟢 |

---

## D. Logging & observabilité

### D.1 Points de log attendus (côté Go)

| Flux | Log présent ? | Niveau (DEBUG/INFO/WARN/ERROR) | Structuré (slog JSON) ? | Classif |
|------|:-------------:|:-------------------------------:|:----------------------:|:-------:|
| Démarrage serveur | oui, partiellement vérifié via câblage | INFO | oui | 🟢 |
| Requête HTTP entrante (middleware) | oui (`slog_logger.go`) | INFO | oui | 🟢 |
| Request ID propagé dans tous les logs | présent dans le middleware | INFO | oui | 🟢 |
| Début/fin sync joueur | non revérifié ici | N/D | N/D | 🟡 |
| Début/fin backfill (par flag) | non revérifié ici | N/D | N/D | 🟡 |
| Écriture DB (writes.go) — succès + échec | non revérifié ici | N/D | N/D | 🟡 |
| Acquisition/libération write lease | non revérifié ici | N/D | N/D | 🟡 |
| Timeout lease | non revérifié ici | N/D | N/D | 🟡 |
| Appel API Halo (succès/échec/retry) | partiellement présent dans les adapters platform | WARN/ERROR | oui | 🟢 |
| Taxonomie d'erreur provider appliquée | documentairement présente | N/D | N/D | 🟢 |
| Migration appliquée (steps_*.go) | non revérifié ici | N/D | N/D | 🟡 |
| Device code flow (init, poll, complete) | partiellement observable via handlers/auth | N/D | N/D | 🟡 |

### D.2 Logs trop verbeux / à nettoyer

| Emplacement | Volume | Suggestion | Classif |
|-------------|--------|------------|:-------:|
| Aucune verbosité problématique confirmée dans cette passe | — | — | 🟢 |

### D.3 Logs absents ou manquants sur flux critiques

| Flux | Quoi logger | Où l'ajouter | Classif |
|------|-------------|--------------|:-------:|
| `pollDeviceFlow` | état de résolution identité / cause d'échec | `apps/go-api/internal/api/handlers/auth.go:136` | 🟠 |
| Ouverture/attach pool DuckDB | succès / erreur / chemin DB | `apps/go-api/internal/platform/duckdb/pool.go:56-160` | 🟠 |

### D.4 Logging côté React

| Aspect | Observation | Classif |
|--------|-------------|:-------:|
| Console.log résiduels en prod ? | Aucun `console.log`, `console.error`, `console.warn` confirmé dans `apps/web/src` ou `e2e`. | 🟢 |
| Error boundary avec télémétrie ? | Aucune `ErrorBoundary` explicite trouvée dans `src/`. | 🟡 |
| Traçage requêtes API (request id round-trip) ? | Non vérifié dans cette passe. | 🟡 |

---

## E. CI & qualité d'exécution

| Aspect | Observation | Classif |
|--------|-------------|:-------:|
| Job coverage Go en CI (CGO activé) | Oui : `.github/workflows/ci.yml:250-257` exécute `go test` avec `CGO_ENABLED=1`, `-coverpkg=./...` et `-tags=integration`. | 🟢 |
| Seuil CI à 70% effectif ? | Oui, indirectement : ratchet contre `apps/go-api/coverage_baseline.txt`, actuellement `76.0`. | 🟢 |
| Ratchet positif (pas de régression) ? | Oui : `apps/go-api/scripts/coverage_check.sh`. | 🟢 |
| Temps d'exécution suite Go < 5 min local / < 10 min CI ? | Non mesuré ici. | 🟡 |
| Temps d'exécution suite React < 2 min ? | Non mesuré ici. | 🟡 |
| Flakiness observée ? | Non mesurée ici. | 🟡 |

---

## F. Récap classifications

| Niveau | Nombre d'items |
|--------|:--------------:|
| 🔴 Bloquant | 1 |
| 🟠 Majeur | 4 |
| 🟡 Mineur | 17 |
| 🟢 Toléré | 13 |

## G. Top 10 trous de couverture à combler en priorité

| # | Zone | Package/feature | Effort (S/M/L) | Impact |
|--:|------|-----------------|:--------------:|--------|
| 1 | Frontend coverage | installer `@vitest/coverage-v8` et produire un vrai rapport React | S | rend enfin la cible React mesurable |
| 2 | Auth | `pollDeviceFlow` | S | onboarding / auth |
| 3 | DuckDB pool | `GetOrOpen`, `openPlayerDB`, `attachShared`, `attachMeta` | M | stabilité runtime |
| 4 | React unit tests | `MatchViewPage` | S | surface centrale non couverte en unit |
| 5 | React unit tests | `LastMatchPage` | S | surface visible non couverte en unit |
| 6 | React unit tests | `SessionComparePage` | S | surface visible non couverte en unit |
| 7 | React unit tests | `TimeseriesPage` | S | surface visible non couverte en unit |
| 8 | React unit tests | `CitationsPage` | S | surface visible non couverte en unit |
| 9 | React unit tests | `ChangelogPage` | S | nouvelle surface non couverte en unit |
| 10 | Media upload | `parseUploadedFiles` | S | chemin d'entrée utilisateur à 0% |

## H. Observations libres

Le constat principal a changé par rapport aux audits anciens : la faiblesse de cette branche n'est plus le backend Go, mais la preuve de qualité frontend. Le backend dispose de tests, d'un workflow coverage ratchetté, de middleware testés et d'un contrat OpenAPI contrôlé. En revanche, le frontend reste dans une zone intermédiaire : vrais tests unitaires déjà présents, MSW correctement branché, Playwright en CI, mais pas de couverture mesurable parce que le provider Vitest n'est pas installé. Pour Sprint 50, c'est la plus grosse dette de preuve encore visible de mon côté.

---

**Fin du template axe 3.**
