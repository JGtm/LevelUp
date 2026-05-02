# Axe 2 · CHECKLIST — Architecture & qualité code

> Cocher au fur et à mesure. Fichier:ligne obligatoire pour chaque écart.

## Phase de préparation

- [ ] SHAs figés dans `SCOPE.md`
- [ ] `GO_ARCHITECTURE_RULES.md` relu
- [ ] Template copié vers `claude_review.md` ou `chatgpt_review.md`
- [ ] `golangci-lint run ./...` exécuté, sortie archivée
- [ ] `tsc --noEmit` + `eslint` exécutés côté React, sortie archivée

---

## Bloc 1 — Hexagonal Go (section A du template)

### 1.1 Identification des couches

- [ ] Chaque package `internal/*` rattaché à une couche (domain/port/service/adapter/analysis/cross)
- [ ] Aucune couche manquante par rapport au plan

### 1.2 Règles de dépendance

- [ ] `internal/domain/` : `go list -f '{{.Imports}}' ./internal/domain/...` ne contient **aucun** `internal/platform`, `internal/service`, `internal/api`, `database/sql`, `net/http`
- [ ] `internal/port/` : importe uniquement `internal/domain/` et std
- [ ] `internal/analysis/` : ne contient pas `database/sql`, `net/http`, `os.File`
- [ ] `internal/service/` : dépend des interfaces `port`, pas des implémentations `platform`
- [ ] `internal/api/handlers/` : zéro import `internal/platform/duckdb` direct
- [ ] `internal/api/middleware/` : pas de logique métier (pas d'import `internal/service`)
- [ ] Pas de cycle d'import (`go list -deps ./...` sans erreur)

### 1.3 Qualité des abstractions

- [ ] `port.Services` : découpée en sous-interfaces par domaine (ou bien une seule justifiée)
- [ ] Chaque repo DuckDB a une interface consommable ailleurs si utile
- [ ] DTOs HTTP vs entités domaine distinguées (pas de struct domaine retournée telle quelle)
- [ ] Zéro `interface{}` / `any` non motivé par un commentaire
- [ ] `JobMeta` est un type structuré (cf. Sprint 49 §volet C)

---

## Bloc 2 — Architecture React (section B du template)

### 2.1 Structure par feature

- [ ] `src/features/<domaine>/` isolé : grep croisé `from '../features/'` retourne 0 import cross-feature
- [ ] `src/routes/` : fichiers de composition uniquement (pas de logique métier)
- [ ] `src/components/ui/` : composants de design system, zéro import `features/`, `routes/`
- [ ] Data fetching isolé (hooks `useX` ou React Query keys centralisées)
- [ ] Un hook par feature pour encapsuler state + fetch

### 2.2 Abstractions

- [ ] Client API typé (idéalement généré depuis OpenAPI, un seul point d'entrée)
- [ ] Couche i18n commune à toutes les pages
- [ ] Error boundaries en place sur chaque route racine
- [ ] État global : 1 solution unique (pas Context + Zustand + Redux en même temps)

---

## Bloc 3 — Duplications & factorisation (section C du template)

### 3.1 Duplications Go

- [ ] Handlers HTTP : patterns d'extraction paramètre / erreur centralisés via `helpers.go`
- [ ] Repos DuckDB : patterns row-scan factorisés (pas de boilerplate répété 13 fois)
- [ ] Middlewares : pas de helpers dupliqués
- [ ] Sync/writes : pas de 3 versions du même INSERT

### 3.2 Duplications React

- [ ] Cards KPI : composant unique réutilisé
- [ ] Wrappers de page (loading/error) factorisés
- [ ] Hooks de fetch typés via génération, pas copiés

### 3.3 Modules trop gros

- [ ] `find apps/go-api/internal -name '*.go' -not -name '*_test.go'` : aucun > 500L (hors `gen/`)
- [ ] `find apps/web/src -name '*.tsx' -not -name '*.test.tsx'` : aucun > 400L
- [ ] Fichiers "god" identifiés : découpage proposé documenté

### 3.4 Fonctions trop longues

- [ ] Toute fonction > 80 L a un `//nolint:funlen` avec raison
- [ ] Toute fonction > 150 L est considérée comme un écart 🔴 ou 🟠

---

## Bloc 4 — Workarounds & fallbacks (section D du template)

### 4.1 Grep systématique

- [ ] `// TODO` — liste tous les TODO Go + React, chacun doit avoir un ticket ou un owner
- [ ] `// FIXME` — zéro toléré
- [ ] `// HACK` — zéro toléré
- [ ] `// XXX` — zéro toléré
- [ ] `//nolint` — chacun a une raison en commentaire
- [ ] `panic(` hors `main` / init : justifié
- [ ] `_ = err` — chacun justifié (erreur volontairement ignorée)

### 4.2 Fallbacks dangereux

- [ ] `if err != nil { return nil }` silencieux : liste
- [ ] Conversions `any` non protégées
- [ ] Branches legacy (`if legacyBackend`, flags de migration) avec date d'expiration en commentaire
- [ ] `strconv.Atoi` sans check d'erreur

### 4.3 Commentaires de fallback

- [ ] `// fallback`, `// default value`, `// just in case` : chacun justifié ou à supprimer

---

## Bloc 5 — Anti-patterns CLAUDE.md transposés (section E du template)

- [ ] Dead code museum Go : zéro fonction exportée non appelée (hors `port.Services` interface — normal)
- [ ] Dead code museum React : zéro composant exporté non utilisé (vérifier via tree-shaking report)
- [ ] God file : aucun module > 500 L mono-responsabilité diluée
- [ ] Swiss-army function : aucun handler qui init + call + transform + write
- [ ] Magic number : toute constante numérique nommée (seuils, timeouts, tailles de batch)
- [ ] Bare connect : `sql.Open` sans `defer db.Close()` ou sans gestion de pool
- [ ] `map[string]any` : chaque occurrence justifiée
- [ ] Shadow / legacy guard : `feature_flags` — chaque flag a une date d'expiration
- [ ] Alias inutile (`var _ = x.Y`) : supprimer ou justifier

---

## Bloc 6 — Nommage & conventions (section F du template)

- [ ] Packages Go : noms courts, lowercase, sans underscore
- [ ] Handlers : `handlers/<feature>.go` (un fichier par feature racine)
- [ ] Services : `service/<feature>_service.go`
- [ ] Repos : `platform/duckdb/<feature>_repo.go`
- [ ] React composants : PascalCase, 1 composant par fichier principal
- [ ] React hooks : `useXxx` camelCase
- [ ] Types TS : colocalisés si feature-specific, partagés sinon

---

## Bloc 7 — Config & feature flags (section G du template)

- [ ] `internal/config/feature_flags.go` : chaque flag a un usage actuel (grep)
- [ ] Pas de flag orphelin (défini mais jamais lu)
- [ ] Pas de flag toujours-vrai ou toujours-faux (code mort)
- [ ] Variables d'env listées dans `README.md` ou `docs/`
- [ ] Côté React : `import.meta.env.*` cohérent avec `.env.example`
- [ ] `VITE_API_BASE_URL` (ou équiv.) aligné avec config backend

---

## Bloc 8 — Signature de code

- [ ] Top 5 dettes techniques identifiées, classées par impact × effort
- [ ] Chaque dette a : fichier:ligne, effort S/M/L, impact

---

## Validation finale de l'axe

- [ ] Template rempli à 100%
- [ ] Tous les écarts ont une classif 🔴🟠🟡🟢
- [ ] Récap §H cohérent avec sections A-G
- [ ] Top 5 §I rempli
- [ ] Commit sur branche `phase11/sprint50-triple-audit`
