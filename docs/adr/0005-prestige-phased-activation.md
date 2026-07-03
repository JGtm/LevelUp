# ADR 0005 — Prestige module : phased activation

**Status** — Accepted (2026-07-03) — Prestige activé par défaut ; gate UNIQUE sourcé `app_settings.json` avec override `PRESTIGE_ENABLED`. (Initialement Proposed 2026-04-29, code review axe 11 + verification-finale-scaffolding.md cas 10.)

**Deciders** — Guillaume (GS).

## Update — 2026-07-03 : activation confirmée + gate unifié (C7 / DEC-4)

Prestige est **activé par défaut**. La source de vérité UNIQUE est `prestige_enabled`
dans `app_settings.json` (le fichier racine ship `true`), avec la variable
d'environnement `PRESTIGE_ENABLED` comme override d'urgence (valeurs falsy
`0`/`false`/`no`/`off` = désactivé). Il existe exactement UN gate,
`prestige.IsEnabled(settingsPath string)` (`apps/go-api/internal/prestige/sync_hook.go`),
consommé par toutes les surfaces via `cfg.PrestigeEnabled` (montage des routes HTTP +
boot bundle dans `server.go`, `config.go`) et par le hook post-sync via
`PrestigeBundle.RunPostSync` / `b.enabled` (`prestige_setup.go`). Le doublon env-only
(`prestige.IsEnabled()` sans argument) et `config.loadPrestigeEnabled` sont **supprimés**.

L'alternative **A) Activation immédiate** est désormais retenue (après validation staging).
La **clause d'expiration 2026-09-30 est annulée** : la feature étant activée, le garde-fou
de date n'a plus d'objet — le test `apps/go-api/internal/config/prestige_expiry_test.go` est
supprimé dans le même commit (sinon dead-guard / doc inversée).

## Context

Le module Prestige est scaffoldé dans le repo depuis le sprint Phase 14 (cf. thought_log [2026-04-29] « documentation Objectifs / Prestige / Notifications ») :

- Backend Go : ~30 fichiers dans `internal/prestige/`, `internal/api/prestige_setup.go`, ~21 routes derrière `PRESTIGE_ENABLED` (lecture env dans `apps/go-api/internal/prestige/sync_hook.go:23`).
- Frontend : 8 composants React dans `apps/web/src/features/prestige/`, 2 routes TanStack (`objectifs/index.tsx` + `palmares/prestige.tsx`) montées dans `NavL1`.
- Configuration : `tuning.toml` + templates + preset arcs Halo packagés.
- État au 2026-04-29 : `PRESTIGE_ENABLED=false` par défaut, **désormais documenté** dans `.env.local.example:57-62` (mise à jour ce jour).

Trois composants exportés sans importateur (`MomentCard`, `ArcSummary`, `StatsGlobales` — cf. axe 4 amendé) restent orphelins même flag activé.

Sans décision claire, on hérite du pattern « scaffolding then forget » identifié sur 12 instances dans la revue. Le coût de maintenance est payé chaque jour (typecheck, tests qui compilent les composants Prestige, deps dans le bundle), pour zéro valeur utilisateur en prod.

## Decision

**Activation phasée du module Prestige** (option A des arbitrages de revue) :

1. **Phase staging (immédiate)** : `PRESTIGE_ENABLED=true` par défaut en CI et staging. Tests smoke obligatoires en mode flag ON (cf. P3.4 du `PLAN_ACTION.md`).
2. **Branchement des composants orphelins** : `MomentCard`, `ArcSummary`, `StatsGlobales` câblés dans la vue Prestige active (P6.5 du plan). Si un composant n'a pas vocation à être utilisé, le supprimer.
3. **Phase production (différée)** : activation prod conditionnée à 1 sprint complet en staging sans incident + tests E2E Playwright verts sur `/objectifs` et `/palmares/prestige`.
4. **Date d'expiration** : ré-évaluation **fin Q3 2026** (2026-09-30). Si non activé en prod à cette date, le module est archivé en `_archive/prestige/` ou supprimé.

## Consequences

### Positive

- Sort le module de l'état dormant, paie la dette d'un coup.
- Aligne la doc env avec l'implémentation (déjà fait).
- Force les tests smoke flag ON, créant un précédent réutilisable pour les autres flags (`MULTI_TITLE_API_ENABLED`).
- Date d'expiration applique la règle CLAUDE.md anti-pattern « Compatibility guard forever ».

### Negative

- 1 sprint de coordination produit (validation fonctionnelle Prestige avant prod).
- Risque que les 3 composants orphelins révèlent un design incomplet à brancher.
- Backfill des données Prestige (PP, arcs, défis) à anticiper sur l'instance staging.

## Alternatives evaluated

| Alternative | Rejected because |
|---|---|
| **A) Activation immédiate prod** | Pas de tests smoke flag ON, risque de régressions visibles utilisateurs. |
| **B) Mise en veille (deferred indefinite)** | Pattern « scaffolding then forget » : devient permanent, dette grandit. |
| **C) Suppression complète du module** | Perte du travail Sprint Phase 14 ; module utile produit-side. |

## References

- Code review : `.ai/V7/review/2026-04-29/axe-11-feature-flags.md` (BLOQUANT 1).
- Verification : `.ai/V7/review/2026-04-29/verification-finale-scaffolding.md` cas 10.
- Plan d'action : `.ai/V7/review/2026-04-29/PLAN_ACTION.md` P1.1, P3.4, P6.5, P7.2.
- Composants orphelins : `apps/web/src/features/prestige/components/{MomentCard,ArcSummary,StatsGlobales}.tsx`.
