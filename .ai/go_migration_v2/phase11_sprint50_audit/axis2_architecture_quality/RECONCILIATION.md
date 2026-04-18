# Axe 2 · Réconciliation — Architecture & qualité code

> **Statut** : ✅ Complété
> **Date de réconciliation** : `2026-04-18`

## 1. Méthodologie

Cf. PROTOCOL.md §Étape 4. Claude a effectué un audit automatisé (grep violations, wc -l, import graph). ChatGPT a effectué une lecture ciblée des fichiers les plus à risque. La divergence clé porte sur une violation hexagonale identifiée uniquement par ChatGPT.

## 2. Convergences

| Item | Section | Classif commune | Fichier:ligne | Note |
|------|---------|:---------------:|---------------|------|
| Architecture hexagonale globalement propre | A.2 | 🟢 | `internal/domain/`, `port/`, `analysis/` | Couches distinctes, zéro violation généralisée |
| `SetupPage.tsx` (467L) — god file onboarding | C.3 | 🟠 | `features/setup/SetupPage.tsx` | Découpage en étapes/state-machine recommandé |
| Cross-feature `settings` → `setup/queries` | B.1 | 🟡 | `features/settings/SettingsPage.tsx:9` | Fuite à corriger |
| TODO `media/reset-index` sans ticket ni TTL | D | 🟡 | `handlers/settings.go:105` | À ticketer |
| `interface{}`/`any` aux frontières dynamiques — usage partiel discutable | A.3 | 🟡 | `domain/job.go:70-72`, `config/config.go:51` | Non abus massif |
| Handlers Go : 1 fichier par resource | — | 🟢 | `internal/api/handlers/` | Conventions respectées |
| Feature flags Go centralisés sans flag orphelin | G | 🟢 | `internal/config/feature_flags.go` | Sain |

## 3. Divergences de classification

| Item | Classif Claude | Classif ChatGPT | Classif finale | Justification |
|------|:--------------:|:---------------:|:--------------:|---------------|
| **`fanout_service.go` importe `platform/duckdb`** | ❌ non détecté | 🟠 | **🟠** | Vérification confirmée : `fanout_service.go:17` importe `levelup/go-api/internal/platform/duckdb`, manipule `*duckdb.PlayerDB` — violation hexagonale réelle |
| `MediaPage.tsx` lignes | 388L → 🟡 | 583L → 🟠 | **🟡** | Mesure réelle au SHA `93c3cd66` : 391L — sous le seuil 500L. 🟡 mais à surveiller. |
| DTOs HTTP distincts des entités domaine | 🟡 (partiel) | 🟡 (partiel) | **🟡** | Convergence : handlers sérialisent souvent `domain.*` directement |

## 4. Items identifiés par un seul LLM

| Item | Par qui | Vérif manuelle | Retenu ? | Classif |
|------|:-------:|----------------|:--------:|:-------:|
| `fanout_service.go:17` import direct `platform/duckdb` | ChatGPT | ✅ Confirmé par grep | ✅ | 🟠 |
| TODO `provider.go:116,124` live call spartan_token | ChatGPT | Confirmé présent | ✅ | 🟡 |
| ErrorBoundary React absent — crash = page blanche | Claude | Confirmé — aucun `ErrorBoundary` dans `src/` | ✅ | 🟠 |
| 27 fonctions >80L (sync/migration/analysis) | Claude | Non re-vérifié par ChatGPT | ✅ | 🟡 |
| `interface{}` vs `any` Go pre-1.18 dans 6+ fichiers | Claude | Confirmé stylistiquement | ✅ | 🟡 |

## 5. Synthèse finale

| Niveau | Nombre | Descriptions |
|--------|:------:|---|
| 🔴 | 0 | — |
| 🟠 | 3 | fanout_service violation hexagonale, ErrorBoundary React absent, SetupPage 467L god file |
| 🟡 | 8 | cross-feature settings→setup, types doublon, TODO media, 27 fonctions >80L, interface{} style, MediaPage 391L, provider TODOs, DTOs couplés |
| 🟢 | 8+ | hexagonal propre, feature flags sains, conventions handlers, pas de dead code, Zustand propre |

## 6. Top 5 dettes prioritaires (consolidées)

| # | Dette | Fichier:ligne | Effort (S/M/L) | Impact |
|--:|-------|---------------|:--------------:|--------|
| 1 | `fanout_service.go` importe `*duckdb.PlayerDB` — violation hexagonale | `internal/service/fanout_service.go:17` | M | Architecture, testabilité |
| 2 | Pas d'`ErrorBoundary` React — crash composant = page blanche | `apps/web/src/` | S | UX prod, fiabilité |
| 3 | `SetupPage.tsx` (467L) — toute la machine onboarding dans 1 fichier | `features/setup/SetupPage.tsx` | M | Maintenabilité |
| 4 | `settings` dépend de `setup/queries` — fuite cross-feature | `features/settings/SettingsPage.tsx:9` | S | Couplage inter-features |
| 5 | TODO `media/reset-index` sans ticket — fonctionnalité incomplète | `handlers/settings.go:105` | S | Feature manquante |

## 7. Recommandation go / no-go pour l'axe 2

- [x] Aucun écart bloquant
- [ ] Violation hexagonale `fanout_service` à corriger (🟠, M)
- [ ] ErrorBoundary React à ajouter (🟠, S)

**Décision** : **GO conditionnel** — la violation hexagonale est isolée à `fanout_service.go` et ne compromet pas la bascule, mais doit être corrigée avant release. L'ErrorBoundary est applicable immédiatement (effort S).
