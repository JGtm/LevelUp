# Skill : delivery-checklist — Go/no-go avant livraison

Invoquer ce skill avant tout PR, merge, ou "c'est livré".

---

## 0. Complétude — la livraison couvre-t-elle TOUT ce qui était demandé ?

À vérifier AVANT les checks techniques (c'est le point le plus souvent raté) :

- [ ] Chaque item de la demande / de l'étape du plan est traité — pas de sous-ensemble
      livré comme si c'était le tout
- [ ] Aucune étape « différée » qui était exécutable maintenant (skill `plan-execution`
      règle 3) ; les reports restants sont justifiés par écrit
- [ ] Aucun TODO/FIXME introduit par ce diff (dette assumée : `TODO(expiry:YYYY-MM-DD)`
      + justification)
- [ ] Ce qui a été débranché (route, caller, feature) est SUPPRIMÉ avec ses tests, ses
      imports, ses types et ses entrées openapi/i18n — pas laissé en zombie testé
- [ ] Tout helper canonique créé/complété embarque son garde-rail anti-régression
      (test grep interdisant l'ancien littéral) dans le MÊME commit
- [ ] Le message final à l'utilisateur décrit fidèlement ce qui est fait ET ce qui ne
      l'est pas — pas de « terminé » approximatif
- [ ] État des runs CI de la branche vérifié (`gh run list --branch <branche>`) AVANT de
      déclarer un lot clos — un gate rejoué LOCALEMENT ne couvre pas les jobs CI (baseline
      Go Linux, type-check + build Vite front). Un signal rouge public ignoré = lot non
      clos (leçon VF-16 : CI de branche rouge sur K2a.. pendant toute la fin de campagne)

## 1. Tests et qualité — Go

```bash
# Depuis apps/go-api/
go test ./...                   # tous les tests
go test -tags=integration -p 1 ./...  # OBLIGATOIRE si le diff touche persist/, sync/ ou
                                # migration/ (les tests anti-ART critiques sont derrière ce
                                # tag — un run nu donne un FAUX VERT). `-p 1` NON NÉGOCIABLE :
                                # le driver DuckDB est mono-process ; en parallèle (défaut),
                                # les packages platform/duckdb + sync se contendent, avortent
                                # avec des durées FANTÔMES (~28000 s) et MASQUENT les vraies
                                # défaillances (piège vécu : gate LOT B « vert » à tort,
                                # 20 tests rouges non vus — 2026-07-03).
go vet ./...                    # analyse statique
# -race : incompatible driver DuckDB tel quel — ajouter -gcflags=all=-d=checkptr=0

# Si un package spécifique
go test ./internal/service/... -v -run TestMatchView
```

> PIÈGE DE FILTRE : pour lister les échecs, ancrer le motif — `Select-String "^--- FAIL:"`
> ou `grep '^--- FAIL:'`. Un filtre `FAIL` nu attrape aussi les logs applicatifs
> (« **Fail**ure while replaying WAL ») et, combiné à un `tail`, fait défiler les vraies
> lignes `--- FAIL:` hors écran → FAUX VERT. Vérifier aussi le code de sortie (`$LASTEXITCODE`
> / `echo $?`), jamais la seule sortie filtrée.

Critères go/no-go :
- [ ] `go test ./...` passe sans erreur
- [ ] `-tags=integration -p 1` lancé si persist/sync/migration touchés — sérialisé, code de
      sortie 0 vérifié (pas seulement la sortie filtrée)
- [ ] `go vet ./...` sans warning
- [ ] Pas de test ignoré (`t.Skip`) sans commentaire justificatif
- [ ] Aucun garde-rail affaibli pour passer (allowlist agrandie, regex assouplie, seuil
      relevé) sans justification datée dans le diff
- [ ] Les nouveaux chemins de code ont un test (adapter, service, handler)

## 2. Tests — Frontend

```bash
# Depuis apps/web/
Remove-Item -Recurse -Force node_modules\.tmp   # purge le tsBuildInfo AVANT (cf. piège)
npm run typecheck     # tsc -b
npm run lint          # eslint
npm run test          # vitest
```

> PIÈGE DE CACHE : `npm run typecheck` = `tsc -b` (build INCRÉMENTAL, `.tsbuildinfo`
> sous `node_modules/.tmp`). Un cache chaud peut rendre un FAUX VERT — tsc re-typecheck
> seulement ce qu'il croit modifié et rate des erreurs pré-existantes. Purger
> `node_modules\.tmp` (ou `tsc -b --force`) AVANT de conclure un typecheck vert.
> Symétrique du faux vert `-p 1` côté Go (§1) : dans les deux cas, l'outillage
> incrémental/parallèle masque des rouges. Vitest, lui, ne typecheck pas (esbuild) —
> un vitest vert NE couvre PAS le typecheck.

Critères :
- [ ] Cache `.tsbuildinfo` purgé avant le typecheck de clôture (pas de faux vert incrémental)
- [ ] TypeScript compile sans erreur
- [ ] Pas de `any` non justifié introduit
- [ ] Nouveau hook/query a un test ou au minimum un type correct

## 3. Logging — Go

- [ ] Toute erreur non-triviale loggée avec `slog.ErrorContext(ctx, "...", "err", err)`
- [ ] Pas de `fmt.Println` ni `log.Printf` introduits
- [ ] Les opérations lentes/importantes ont un `slog.InfoContext` ou `slog.DebugContext`
- [ ] Clés structurées standards : `"err"`, `"match_id"`, `"player"`, `"titleSlug"`, `"duration"`

## 4. Multi-titres

- [ ] Chemins via `PathResolver` — aucun `filepath.Join(repoRoot, "data", ...)` direct
- [ ] Dégradation gracieuse sur `ErrCapabilityNotSupported` (pas de panic, pas d'erreur 500)
- [ ] Branché sur `HasCapability()` / `CapabilityMap.Has()`, jamais sur `slug == "halo_infinite"`
- [ ] Si nouveau champ stats : section ajoutée dans `config/titles/halo_infinite/mappings/fields.toml`
- [ ] Si nouvel outcome/asset : `assets.toml` / `outcomes.toml` mis à jour

## 5. Architecture

- [ ] Aucune colonne DuckDB title-specific dans un service (tout via adapter)
- [ ] Aucun fichier dépassant 500 lignes
- [ ] Aucune fonction dépassant 80 lignes
- [ ] Pas de code mort laissé (fonctions jamais appelées, branches toujours fausses)
- [ ] Feature flags nettoyés si la feature est en prod ; tout kill-switch conservé porte
      date de basculement de défaut + date cible de retrait + critère mesurable
      (modèle : `platform/duckdb/shared_reader_legacy.go`)
- [ ] Écritures per-match via `persist.BatchBuilder`/Persister (jamais d'UPSERT concurrent
      sur les tables critiques) ; lectures de tables append-only via les vues `_latest`
- [ ] Si un défaut de flag a basculé dans ce diff : les commentaires/doc du flag sont mis
      à jour dans le MÊME commit (une doc inversée sur un kill-switch = risque opérationnel)

## 6. Frontend — règles couleurs et i18n

- [ ] Pas de hex `#RRGGBB` ni classe Tailwind couleur dans `features/` ou `components/`
- [ ] Nouvelles strings UI ajoutées dans `i18n.ts` pour **FR et EN**
- [ ] Nouvelles query keys ajoutées dans `apps/web/src/lib/query/keys.ts`
- [ ] `routeTree.gen.ts` non édité manuellement

## 7. Thought log

- [ ] Entrée ajoutée dans `.ai/thought_log.md` avec :
  - Date `[YYYY-MM-DD]`
  - Titre de la tâche
  - Statut (Complété)
  - Décision technique principale
  - Résultats observés
  - Prochaine étape

---

## Commande de vérification rapide

```bash
# Go — tout en une fois
cd apps/go-api && go test ./... && go vet ./...

# Frontend — tout en une fois
cd apps/web && npm run typecheck && npm run lint

# Chercher les fmt.Println oubliés
grep -r "fmt\.Println\|log\.Printf\|log\.Println" apps/go-api/internal/

# Chercher les filepath.Join directs sur data/
grep -r 'filepath\.Join.*"data"' apps/go-api/internal/
```
