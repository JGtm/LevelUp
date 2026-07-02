# Skill : plan-review — Grille de revue d'un plan d'implémentation

Utiliser ce skill pour valider qu'un plan est complet, cohérent avec l'architecture, et livrable.

---

## 1. Structure du plan — est-il bien formé ?

- [ ] Objectif clair et critère de succès défini ("quasi ISO", "feature X opérationnelle", etc.)
- [ ] Phases ou étapes ordonnées (risque/effort croissant de préférence)
- [ ] Blockers identifiés et documentés avec justification et workaround éventuel
- [ ] Effort estimé ou au moins catégorisé (rapide / moyen / lourd)
- [ ] Branche Git cible nommée

## 2. Architecture Go — les couches sont-elles respectées ?

- [ ] Les algos purs prévus dans `internal/analysis/`
- [ ] Les types de résultat prévus dans `internal/domain/` ou `internal/games/canonical/`
- [ ] L'orchestration prévue dans `internal/service/`
- [ ] Les interfaces nécessaires ajoutées dans `internal/port/`
- [ ] Les handlers HTTP dans `internal/api/handlers/` (pas de logique métier)
- [ ] Aucun accès DuckDB direct prévu dans handlers ou services (tout via adapter ou repo)

## 3. Multi-titres — le plan est-il title-aware ?

- [ ] Tous les chemins de fichiers passent par `PathResolver` (pas de `filepath.Join` direct)
- [ ] Les nouvelles features branchées sur `HasCapability()` / `CapabilityMap.Has()`, pas sur le slug
- [ ] Si nouveau champ stats : `config/titles/halo_infinite/mappings/fields.toml` prévu
- [ ] Si nouvel outcome/asset : `assets.toml` / `outcomes.toml` prévus
- [ ] Si dégradation pour certains titres : `ErrCapabilityNotSupported` mentionné

## 4. Adapters — le plan utilise-t-il la bonne interface ?

- [ ] Données lues via `TitleDataAdapter.Load*()` (pas de SQL inline dans service)
- [ ] Types de retour sont des types canoniques (`canonical.MatchSummary`, `canonical.PlayerStats`, etc.)
- [ ] Si nouvel adapter : `internal/games/{title}/adapter_data.go` et `adapter_semantic.go` prévus
- [ ] Si nouveau `FieldKey` : ajouté dans `canonical/fields.go` ET dans le TOML correspondant

## 5. Tests — sont-ils planifiés à chaque couche ?

- [ ] `internal/analysis/` : tests unitaires purs prévus
- [ ] `internal/service/` : tests avec mock `port.Repository` prévus
- [ ] `internal/api/handlers/` : tests `httptest` prévus ou existants couvrent les nouveaux cas
- [ ] `platform/duckdb/` : tests avec DuckDB `:memory:` prévus si nouveau repo
- [ ] Frontend : typecheck + tests hooks si logic complexe

## 6. Logging — est-il prévu ?

- [ ] `slog.InfoContext` ou `slog.DebugContext` pour les opérations significatives
- [ ] `slog.ErrorContext(ctx, "...", "err", err)` pour toutes les erreurs non-triviales
- [ ] Pas de `fmt.Println` dans le code prévu

## 7. Frontend — si la feature touche apps/web

- [ ] Nouvelles routes ajoutées dans `apps/web/src/routes/` (file-based, pas dans routeTree.gen)
- [ ] Query keys prévues dans `apps/web/src/lib/query/keys.ts`
- [ ] Strings UI prévues dans `i18n.ts` (FR + EN)
- [ ] Labels de stats via `useFieldLabel()` / `useOutcomeLabel()` (pas hardcodés)
- [ ] Pas de couleurs hex/Tailwind prévues directement dans les composants

## 8. Livraison — le plan définit-il une "done definition" ?

- [ ] Critères de complétion par phase
- [ ] `thought_log.md` mentionné (ou au moins implicite à chaque phase)
- [ ] Pas de dépendance externe bloquante non documentée

## 9. Exécutabilité par un agent — le plan résiste-t-il aux dérives d'exécution ?

Un plan destiné à être exécuté par un agent (Opus ou autre) doit être blindé contre les
dérives connues : exécution non séquentielle, traitement partiel, étapes différées au lieu
d'être faites, périmètre adapté en silence.

- [ ] Périmètre FERMÉ par étape : liste exhaustive d'items cochables, pas de « etc. »
      ni de « entre autres »
- [ ] Gate vérifiable par étape : commandes exactes à exécuter (tests, greps), pas de
      critère subjectif (« le code est propre »)
- [ ] Statuts d'item définis (`[x]` / `[~]` réf / `[!]` justifié) et règle « aucune case
      vide à la clôture »
- [ ] Règle d'ordre explicite (étape N close avant N+1) + définition de « clos »
- [ ] Interdiction des fixes hors périmètre + section « Découvertes » où les consigner
- [ ] Décisions produit TRANCHÉES avant l'exécution (pas de « à décider en cours de
      route » qui arrêtera ou fera dériver l'agent)
- [ ] Protocole de reprise de session (où lire l'avancement, où reprendre)
- [ ] Renvoi explicite au skill `plan-execution` (contrat par défaut)

---

## Questions à poser sur un plan spécifique

1. **Quelle couche est manquante ?** — Passer mentalement la liste handler→service→analysis→domain→platform
2. **Y a-t-il du code title-specific là où il devrait y avoir une capability ?**
3. **Les types de retour des services sont-ils canoniques ou title-specific ?**
4. **Les tests couvrent-ils les cas de dégradation (`ErrCapabilityNotSupported`) ?**
5. **Est-ce que chaque phase est livrable indépendamment ?** (si non, noter la dépendance)
