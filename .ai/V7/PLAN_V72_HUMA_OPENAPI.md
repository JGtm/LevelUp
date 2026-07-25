# PLAN V72-01 — openapi.yaml généré par Huma (« exploiter Huma à son potentiel »)

Chantier parent : PLAN_V72_NOTION_BATCH.md (item V72-01). Recon : agent Sonnet 2026-07-24.
Revue plan-review (agent Opus, 2026-07-25) : GO avec amendements, CONDITIONNÉ au spike
H0.5 — amendements INTÉGRÉS ci-dessous. Exécution EN DERNIER des gros lots v7.2.

Contrat : skill `plan-execution`. Branche : `feat/v7.2-notion-batch`. Statuts d'items
`[x]/[~]/[!]`, aucune case vide à la clôture ; reprise à la première case non statuée ;
ZÉRO fix hors périmètre (H1 touche ~71 handlers — interdiction de « nettoyer au
passage »). NB : chemins Go préfixés `apps/go-api/`.

## État des lieux (vérifié sur pièces, contre-vérifié par la revue)

- **203 routes Huma exactement** (108 GET / 72 POST / 13 PATCH / 9 DELETE / 1 PUT).
  Compte de référence : call-sites `huma.X(api,` hors tests (un grep naïf `huma.Get`
  donne ~254 à cause du drift test et des commentaires — NE PAS s'y fier).
  **71 méthodes `Mount()`** réelles (55 invocations dans `server_apiv1.go`) + 3 routes
  inline `internal/api/huma_routes.go`. ~20 routes chi brut dont seules les **7 routes
  `groups.go`** (`server_apiv1.go:320-328`) sont migrables ; le reste (assets images,
  media upload/serve, export CSV, redirects OAuth ×4) reste manuel → YAML composite.
- **Pas d'instance `huma.API` unique** : chaque `Mount()` appelle `humacore.NewAPI(r)`
  (`internal/api/humacore/humacore.go:265-288`), registres de schémas jetés.
  Docs/OpenAPI natifs désactivés (`humacore.go:271-273`).
- **Contrainte architecturale critique (revue)** : une `huma.API` est liée À UN routeur ;
  les handlers sont montés sur des sous-routeurs hétérogènes (middlewares `r.With`/
  `r.Group(RequireAuth)`, path params parents `/players/{player_slug}`,
  `/profiles/{player_slug}/titles/{slug}` — `server_apiv1.go:320,333,341,549,556`).
  Une instance unique naïve sur le routeur racine perdrait middlewares et params
  parents. **L'objectif est donc « UN DOCUMENT OpenAPI unique (registre partagé) »,
  PAS « une instance unique »** : une `NewAPI` par sous-routeur pointant sur le même
  document/registre.
- 0 usage de `huma.Operation{}`, 0 tag `doc:`/`enum:`/`default:`/`example:`,
  0 securityScheme. 16 endpoints `RawBody` (contrat d'erreur 400 historique) sans schéma
  de body dérivable → leurs request bodies RESTENT dans le fragment manuel.
- `api/openapi.yaml` : 17 822 lignes, 175 paths, 518 schémas. Sémantique manuelle
  UNIQUEMENT : 402 description, 21 enum, 30 default, 6 example. Le drift test ne gate
  que MISSING (`openapi_schema_drift_test.go:111-120`).
- Dettes vivantes : double schéma d'erreur `ApiErrorSchema`/`ApiError` ; en-tête 3.1
  style 3.0 ; commentaire oapi-codegen obsolète (`openapi_schema_drift_test.go:~216`) ;
  `api/openapi_fastapi_reference.yaml` inerte à archiver.

## Plan de bascule (étapes, effort, gates)

- [x] **H0 — Baseline diff sémantique (S).** Outillage TRANCHÉ : script Go kin-openapi
      dans le repo (pas de binaire externe en CI). Rapport baseline commité.
      FAIT (2026-07-25) : outil `apps/go-api/cmd/openapi-diff/` (kin-openapi v0.144.0,
      chargement sans validation stricte car doc 3.1.0 corps 3.0-style ; modes défaut/
      `-out`/`-diff`). Baseline `.ai/V7/openapi_baseline_v72.txt` (176 paths / 182 ops /
      520 schémas ; enum=26 default=30 example=124 desc-nodes=478 resp-desc=374). Sortie
      DÉTERMINISTE (2 runs byte-identiques) ; `-diff` self-check = exit 0. Couverture :
      paths+méthodes+operationId+summary+tags+params(+enum/default/example inline)+
      requestBody+responses(+desc)+media-type example ; schémas walkés RÉCURSIVEMENT
      (inline uniquement ; $ref nommés listés une fois en section SCHEMAS, jamais
      expansés → pas de cycle). Gate H0 : `go build`/`go vet ./cmd/openapi-diff/` verts,
      baseline générée. [x]
- [x] **H0.5 — SPIKE de dérisquage (M, BLOQUANT).** Prouver le partage d'un seul
      `*huma.OpenAPI` (registre de schémas) entre plusieurs adaptateurs par-sous-routeur
      (une `NewAPI(r, sharedDoc)` par Mount) : middlewares et params parents préservés.
      Gate : `TestHumaNestedSubrouterProbe` vert + un POC 2 sous-routeurs qui émet un
      document fusionné correct. **Si le spike échoue → STOP, replanifier (NO-GO
      implicite de la revue sans ce spike).**
      **VERDICT (2026-07-25) : spike CONCLUANT.** Mécanisme prouvé (huma v2.38.0) :
      (1) PARTAGE DU DOC — `huma.Config` incorpore `*huma.OpenAPI` et `api.OpenAPI()`
      renvoie `config.OpenAPI` (api.go:327) ; passer la MÊME `huma.Config` à plusieurs
      `humachi.New` fait pointer tous les adaptateurs vers LE MÊME doc + registre
      (huma.NewAPI initialise `Components`/`Schemas` de façon idempotente à travers ce
      pointeur, api.go:475-489). (2) PRÉFIXE PARENT (le point dur) — huma enregistre le
      MÊME `op.Path` dans le doc (`AddOperation`) ET sur le routeur chi
      (`chiAdapter.Handle` → `MethodFunc`) : découplage via `huma.NewGroup(api, prefix)`
      qui préfixe le chemin du DOCUMENT + régénère l'operationID (group.go, PrefixModifier
      + ModifyOperation), l'adaptateur sous-jacent étant un `prefixStripAdapter` qui RETIRE
      le préfixe avant `MethodFunc` (le sous-routeur chi est déjà monté sous ce préfixe →
      chemin LOCAL, middlewares + params parents intacts). Ajouts humacore OPT-IN
      (défaut `NewAPI` inchangé) : `NewSharedConfig`, `NewAPIWithConfig`, `NewSubrouterAPI`.
      POC `internal/api/humacore/shared_doc_poc_test.go` : (a) params parents + middleware
      témoin + gate 401 OK ; (b) doc fusionné porte les paths ABSOLUS
      (`/api/v1/players/{player_slug}/pages/probe`, `/matches/{match_id}`), le chemin
      local nu ABSENT ; (c) `CommonMeta` enregistré 1×, `$ref` depuis les DTOs des DEUX
      groupes, operationIDs uniques. Gate : `go test ./internal/api/humacore/`
      (POC + existants) + `./internal/api/` (incl. `TestHumaNestedSubrouterProbe`,
      drift, contract, route-collision) + `go build ./...` — TOUS verts. H1-H8 = GO.
- [ ] **H1 — Document OpenAPI partagé (L).** Généraliser le mécanisme du spike aux
      71 `Mount()`. Gate : contract_test + openapi_schema_drift_test verts (inchangés),
      `TestHumaNestedSubrouterProbe` vert.
- [ ] **H2 — Tags/Summary/OperationID (M).** Les 203 routes (`huma.X(api,`) passent en
      `huma.Register` + `huma.Operation`. Gate : spec en mémoire porte les tags.
- [ ] **H3 — Instrumenter les DTOs (L).** Porter 402 descriptions / 21 enums /
      30 defaults / 6 examples en tags struct Go. LIVRABLE SUPPLÉMENTAIRE (revue) :
      inventaire FERMÉ des sémantiques NON portables en tags (descriptions par-contexte
      sur type partagé, 16 request bodies RawBody, double schéma d'erreur) avec leur
      destination = fragment manuel. Gate : diff sémantique H0 = 0 perte (hors items
      inventoriés).
- [ ] **H4 — Modèle d'erreur unifié (M).** Fusion ApiErrorSchema/ApiError sur
      `humacore.apiError` enrichi. Gate : call-sites NewError OK.
- [ ] **H5 — Migrer groups.go vers Huma (S).** 7 routes writeJSON → typées.
- [ ] **H6 — Pipeline génération + golden (M). NE DÉMARRE PAS tant que H2, H3, H4, H5
      ne sont pas CLOS** (tous items statués, gates verts) — sinon le golden fige une
      spec appauvrie. `api.OpenAPI().YAML()` + fusion fragment manuel VERSIONNÉ
      (précédence DÉFINIE : le fragment manuel gagne sur toute collision de path) →
      écrit api/openapi.yaml. Drift test remplacé par golden `TestOpenAPIYAMLIsUpToDate`
      + assertion que les ~20 paths chi-brut du fragment survivent à la régénération.
      Filtrer les wrappers anonymes par-route. Archiver openapi_fastapi_reference.yaml.
- [ ] **H7 — generate-types (S→M).** Diff massif attendu dans generated.ts (14 010 L).
      GARDE SÉMANTIQUE (revue) : snapshot des noms de types exportés + membres d'enums
      de generated.ts (test dédié) — pas seulement tsc ; `response-types.guard.test.ts`
      rendu assertif sur les types critiques. Attention 3.1 `type:[X,"null"]` vs 3.0
      `nullable:` → vérifier la forme TS produite.
- [ ] **H8 — Non-régression (S).** Diff sémantique 0 breaking vs H0 ; contract tests ;
      golden en CI ; vitest ; tsc -b.
- Bonus post-bascule (optionnels, S) : `/docs` gaté hors prod, securitySchemes,
  validation auto des inputs, résolveurs cross-champs.

## Journal

- 2026-07-24 : recon + plan posés. Séquencement : dernier des gros lots v7.2.
- 2026-07-25 : revue plan-review intégrée : spike H0.5 bloquant ajouté, H1 reformulé
  « document partagé » (pas instance unique), comptages corrigés (71 Mount), inventaire
  fermé H3, précédence + golden du fragment H6, garde sémantique generated.ts H7,
  outillage H0 tranché (kin-openapi), blindage plan-execution.
- 2026-07-25 : **H0 + H0.5 EXÉCUTÉS (agent Opus).** H0 : outil `cmd/openapi-diff/`
  (kin-openapi v0.144.0, direct dep après `go mod tidy`) + baseline
  `.ai/V7/openapi_baseline_v72.txt` (176 paths / 182 ops / 520 schémas), déterministe,
  self-diff = 0. H0.5 : spike CONCLUANT (détail ci-dessus). Mécanisme H1 retenu :
  UNE `NewSharedConfig()` construite au point de composition (`server_apiv1.go`),
  chaque Mount reçoit soit `NewAPIWithConfig(r, cfg)` (routeur racine / groupe
  middleware-only), soit `NewSubrouterAPI(r, cfg, docPrefix)` (sous-routeur à préfixe).
  RISQUE RÉSIDUEL H1 : le `docPrefix` absolu de chaque Mount doit être fourni depuis
  `server_apiv1.go` (chi n'expose pas le préfixe de montage à l'enregistrement) — les
  méthodes `Mount(r)` doivent recevoir leur préfixe (nouveau paramètre ou table de
  préfixes au point de composition). Périmètre : humacore.go + POC + go.mod/go.sum +
  ce plan (ZÉRO autre fichier). Aucun commit (superviseur).
