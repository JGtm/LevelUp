# PLAN V72-01 — openapi.yaml généré par Huma (« exploiter Huma à son potentiel »)

Chantier parent : PLAN_V72_NOTION_BATCH.md (item V72-01). Recon : agent Sonnet 2026-07-24.
Statut : plan posé, revue plan-review en cours. Exécution EN DERNIER des gros lots v7.2
(touche ~79 points de montage — éviter les conflits avec les autres correctifs Go).

## État des lieux (vérifié sur pièces)

- **203 routes Huma** (108 GET / 72 POST / 13 PATCH / 9 DELETE / 1 PUT) sur ~79 points
  d'enregistrement (`Mount()` par handler, `internal/api/handlers/*.go` + 3 inline
  `internal/api/huma_routes.go:33,57,84`). **~20 routes chi brut** dont seules les
  **7 routes `groups.go`** (JSON via writeJSON, `server_apiv1.go:322-328`) sont une vraie
  dette migrable ; le reste (5 assets images, media upload/serve, export CSV, redirects
  OAuth ×4) est structurellement hors scope JSON Huma → **le YAML restera composite**.
- **Pas d'instance `huma.API` unique** : chaque `Mount()` appelle `humacore.NewAPI(r)`
  (`internal/api/humacore/humacore.go:265-288`) → 79 registres de schémas jetés après
  enregistrement. `api.OpenAPI()` inexploitable en l'état. Docs/OpenAPI natifs
  explicitement désactivés (`humacore.go:271-273`).
- **0 usage** de `huma.Operation{}` (tags/summary/operationID), **0 tag** `doc:`/`enum:`/
  `default:`/`example:` sur les structs, **0 securityScheme**. 16 endpoints en `RawBody`
  (contrat d'erreur 400 historique) sans schéma de body dérivable.
- `api/openapi.yaml` : 17 822 lignes, 175 paths, 518 schémas. Sémantique manuelle
  UNIQUEMENT : 402 description, 21 enum, 30 default, 6 example → une génération naïve
  les perd EN SILENCE (le drift test ne gate que la PRÉSENCE des schémas, pas leur forme :
  MISSING=erreur, DIVERGENT/EXTRA=logés seulement, `openapi_schema_drift_test.go:111-120`).
- Preuves de dette vivante : double schéma d'erreur `ApiErrorSchema` (riche, manuel) vs
  `ApiError` (auto-émis pauvre) ; en-tête `openapi: 3.1.0` mais style 3.0 (`nullable:`) ;
  commentaire obsolète oapi-codegen (`openapi_schema_drift_test.go:216` — outillage mort,
  le vrai consommateur est openapi-typescript@7.13 qui supporte 3.1) ;
  `api/openapi_fastapi_reference.yaml` = artefact FastAPI inerte à archiver.

## Plan de bascule (étapes, effort, gates)

- [ ] **H0 — Baseline diff sémantique (S)** : outiller un diff OpenAPI (oasdiff ou script
      Go kin-openapi) sur l'état actuel. Gate : rapport baseline commité.
- [ ] **H1 — Instance `huma.API` unique (L, PRÉALABLE ABSOLU)** : créer l'instance dans
      server_apiv1.go, injecter dans chaque `Mount(r, api)` (~76 signatures). Risque :
      héritage middleware par sous-routeur (`humacore.go:258-261`) à préserver.
      Gate : contract_test + openapi_schema_drift_test verts (inchangés à ce stade).
- [ ] **H2 — Tags/Summary/OperationID (M)** : passer les 203 routes en `huma.Register`
      + `huma.Operation`. Gate : spec générée en mémoire porte les tags du bloc manuel.
- [ ] **H3 — Instrumenter les DTOs (L)** : reporter 402 descriptions / 21 enums /
      30 defaults / 6 examples du YAML manuel en tags struct Go (`doc:`, `enum:`, ...).
      Gate : diff sémantique = 0 perte vs baseline H0.
- [ ] **H4 — Modèle d'erreur unifié (M)** : fusionner ApiErrorSchema/ApiError sur
      `humacore.apiError` enrichi (details, field_errors). Gate : call-sites NewError OK.
- [ ] **H5 — Migrer groups.go vers Huma (S)** : 7 routes writeJSON → typées.
- [ ] **H6 — Pipeline génération + golden test (M)** : `api.OpenAPI().YAML()` + fusion
      fragment manuel (routes binaires) → écrit api/openapi.yaml. Drift test remplacé par
      golden `TestOpenAPIYAMLIsUpToDate` (régénère, diff avec le commité). Filtrer les
      wrappers anonymes par-route du components.schemas. Archiver openapi_fastapi_reference.
- [ ] **H7 — generate-types (S)** : diff massif attendu dans generated.ts (14 010 L) —
      à review en bloc. Gate : tsc -b + response-types.guard.test.ts verts.
- [ ] **H8 — Non-régression (S)** : diff sémantique 0 breaking vs H0 ; contract tests ;
      golden ; vitest.
- Bonus post-bascule (optionnels, S chacun) : `/docs` interactif (gater hors prod),
  securitySchemes déclarés (RequireAuth/RequireAdmin), validation auto des inputs
  (contraintes sur structs), résolveurs cross-champs.

## Journal

- 2026-07-24 : recon + plan posés. Décision de séquencement : dernier des gros lots v7.2.
