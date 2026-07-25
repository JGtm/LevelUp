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
- [x] **H1 — Document OpenAPI partagé (L).** Généraliser le mécanisme du spike aux
      71 `Mount()`. Gate : contract_test + openapi_schema_drift_test verts (inchangés),
      `TestHumaNestedSubrouterProbe` vert.
      FAIT (2026-07-25, agent Opus). Ergonomie : `NewAPI` rendu VARIADIC
      (`NewAPI(r, opts ...MountOption)`) — sans option = comportement legacy (doc isolé,
      jeté), avec `WithSharedDoc(cfg, docPrefix)` = document PARTAGÉ. Choix motivé :
      préserve la signature `Mount(r)` des ~130 call-sites de tests handlers (variadic →
      `Mount(r)` compile inchangé, ZÉRO modif d'assertion). Chaque `Mount` devient
      `Mount(r chi.Router, opts ...humacore.MountOption)` + `NewAPI(r, opts...)` (2 lignes
      mécaniques × 74 fichiers, 77 méthodes). Config partagée créée UNE fois dans
      `NewRouter` (`humacore.NewSharedConfig`), threadée via `apiV1Inputs`/`apiV1Deps`,
      injectée par option scopée à côté de chaque `r.Route` (préfixe absolu = source
      unique `apiV1BasePath` + suffixe du r.Route). Accesseur H6 : `NewRouter` retourne
      un 3e résultat `*huma.OpenAPI` (document fusionné, non exposé en HTTP). Points durs
      résolus : (a) `MarkRequestBodyOptional` (15 handlers, chemin LOCAL) — le
      sous-routeur expose `DocPrefix()`, la fonction résout l'absolu ; (b) garde-rail
      anti-collision — sous doc partagé huma écrase silencieusement, la lecture du doc
      final ne voit plus la collision → nouveau hook `OnOperationRegistered` (par
      enregistrement, chemin absolu) ; `TestNoDuplicateRouteRegistration` réécrit dessus
      (détection PLUS fine, couvre l'ancienne LIMITE r.With/r.Group). Nouveau garde-rail
      FIDÉLITÉ : `TestSharedOpenAPIDocCoversAllHumaRoutes` (166 ops Huma fusionnées →
      162 paths ; chi total 208 : les 42 restants = chi-brut connus, exclusions
      documentées) — vérifie doc ⟺ chi.Walk (chemins absolus), pas de fantôme, pas
      d'oubli de `WithSharedDoc`. Gates : gofmt vide, `go build ./...`, `go vet ./...`,
      `go test ./internal/api/...` (contract/drift/nested/collision/fidélité), `go test
      ./...` complet (exit 0, 118 ok, 0 fail), `make go-api-lint` (0 issue) — TOUS verts.
      DocsPath/OpenAPIPath restent désactivés (aucune route HTTP auto). [x]
- [x] **H2 — Tags/Summary/OperationID (M).** Les 204 call-sites (`huma.X(api,`) portent
      OperationID + Summary + Tags stables. Gate : spec en mémoire porte les tags.
      FAIT (2026-07-25, agent Opus). Ergonomie : helper mince `humacore.Op(opID, summary,
      tags...)` (nouveau `humacore/operation.go`) passé en DERNIER argument variadique de
      `huma.Get/Post/...` — PAS de conversion vers `huma.Register` (préserve la signature
      des 204 call-sites + le type inference générique ; poser OperationID/Summary
      EXPLICITEMENT désactive la régén auto de huma, y compris le PrefixModifier du
      sous-routeur qui ne re-préfixe l'ID que s'il est resté celui de la convenience —
      seul le chemin gagne le préfixe absolu). Application MÉCANIQUE des 204 call-sites via
      un générateur go/ast jetable (scratchpad, insertion à l'offset du Rparen, forme
      multi-ligne + summary concaténé quand la ligne dépasse 210 char pour lll). Décompte
      exact (AST) : **204 call-sites** (108 GET / 72 POST / 13 PATCH / 9 DELETE / 1 PUT —
      révise le « 203 » estimé), dont **159 en parité EXACTE** avec api/openapi.yaml
      (operationId/summary/tags repris verbatim) et **45 supplémentés** (cf. Découvertes).
      RÉCONCILIATION 204 statique → 166 runtime (doc de démo) : delta 38 = 29 Prestige
      (bundle nil en démo) + 3 catalog (`/titles/{slug}/catalog/*`, metadata DB indispo)
      + 3 diag auto-sync (`autoSyncScheduler` nil dans le routeur de test) + 3
      assets_metadata (2 branches mutuellement exclusives : 6 call-sites → 3 routes montées).
      Gate H2 : nouveau `openapi_operation_metadata_test.go` (charge le yaml via kin-openapi,
      vérifie (a) chaque op du doc partagé a OperationID+Summary+≥1 Tag, (b) parité
      operationId+tags sur les chemins COMMUNS). Verdict : 166 ops, metadata OK=166,
      parité vérifiée 156, échecs 0. Gates : gofmt -l vide, `go build ./...` (0),
      `go vet ./...` (0), `go test ./internal/api/...` (ok), `go test ./...` (0),
      `make go-api-lint` (0 issues) — TOUS verts. Aucun commit (superviseur). [x]
- [x] **H3 — Instrumenter les DTOs (L).** FAIT (2026-07-25, agent Opus). Réalité mesurée
      sur pièces (script kin-openapi jetable) : les cibles « 402/21/30/6 » du recon sont des
      comptages BRUTS de tout le doc — 175 des 232 `description:` du yaml sont des
      descriptions de RÉPONSE/paramètre (niveau opération, PAS schéma.champ) ;
      enum=21/default=30/example=6 confirmés au grep. La sémantique de schéma.champ PORTABLE
      en tag Huma (doc/enum/default/example de CHAMP) se limite à 31 schémas ; parmi eux
      seuls **11 ont un type Go HUMA-GÉNÉRÉ** (output ou imbriqué dans un output) — les
      « Request » sont décodés à la main (`RawBody` / `var req domain.X`) donc JAMAIS générés
      par Huma, et 10 schémas n'ont aucun type Go (chi-brut / champ Go scalaire / legacy
      Python). **POSÉ : 19 descriptions + 11 enums + 5 defaults sur 10 types Go** (bootstrap,
      auth, engagement_score, media_audio_config, settings, admin_monitoring, explorer,
      career). Vérifié empiriquement : Huma INLINE les enums des types string custom
      (`MediaAudioMode`/`AudioTrackRole`) — pas de `$ref`, l'enum surgit sur le champ. Enums
      d'INPUT : AUCUN risque 422 — tous les types instrumentés sont des OUTPUTS (non validés)
      ou des inputs `RawBody` non typés Huma (vérifié call-site par call-site) ; aucun enum
      posé sur un `Body` typé Huma. Descriptions de SCHÉMA RACINE non portables (Huma n'a pas
      de tag type-level ; seuls `SchemaProvider`/`SchemaTransformer`, hors périmètre tag) →
      inventaire. 2 descriptions raccourcies pour lll ≤ 220 (`provides_max_killing_spree`,
      `media_delete_source_after_transcode`) — le texte long peut retourner au fragment H6.
      Gate H3 : nouveau `openapi_schema_semantics_test.go` — **Partie A** (indépendante du
      routage : régénère chaque type via un registre Huma neuf → 35 sémantiques vérifiées /
      10 types, couvre les types dont la route n'est pas montée en démo) + **Partie B** (doc
      partagé en mémoire vs yaml, schéma.champ COMMUN, allowlist = inventaire + compteurs
      anti-caducité : 28 en parité, 5 pertes = 5 descriptions racine allowlistées, **0 perte
      hors inventaire**). Gates : gofmt -l vide, `go build ./...`, `go vet ./...`,
      `go test ./internal/api/...`, `go test ./...` (118 pkg ok, 0 fail), `make go-api-lint`
      (0 issue) — TOUS verts. Aucun commit (superviseur). Inventaire fermé ci-dessous. [x]
- [x] **H4 — Modèle d'erreur unifié (M).** FAIT (2026-07-25, agent Opus). Cartographie :
      `humacore.apiError` (internal/api/humacore/humacore.go) = SEUL type d'erreur, SEUL
      constructeur `NewError(status, code, message)` (458 occurrences `NewError(` sur 81
      fichiers, tous `humacore.NewError` au contrat {code,message,retryable}, signature
      INCHANGÉE). Deux schémas yaml coexistaient : `ApiErrorSchema` (riche, manuel, 7 `$ref`)
      et `ApiError` (auto-émis par Huma depuis `apiError`, pauvre, 0 `$ref`). **Champs
      unifiés** : `apiError` enrichi de `Details any` + `FieldErrors []FieldError` (nouveau
      type exporté `FieldError{Field,Message,Code}`), tous deux `omitempty` → **contrat
      runtime {code,message,retryable} INCHANGÉ** (nil absent du corps ; `TestNewError_*`
      verts sans modif). Tags Huma posés (style H3) : `code` doc+example `player_not_found`,
      `message` doc, `retryable` doc, `details` doc, `field_errors` doc. **Compat front
      VÉRIFIÉE** : `apps/web/src/lib/api/client.ts` lit déjà `code/message/retryable/details/
      field_errors` (interface ApiError + FieldError présentes) → `errorBody.details ?? …`
      tolère l'absence ; AUCUN changement de contrat runtime. **YAML convergé** (édition
      ciblée) : les 7 `$ref` ApiErrorSchema → `ApiError` ; `ApiError` réécrit fidèle au riche
      (dérivé du type Go via emit Huma : code/message/retryable/details/field_errors→FieldError,
      required [code,message,retryable]) ; `FieldError` ajouté (MISSING résorbé) ;
      `ApiErrorSchema` + `FieldErrorSchema` supprimés (0 réf orpheline). Drift : MISSING=0
      (gate) ; `ApiError` reste DIVERGENT non-gaté (représentation `example`/`examples`,
      `details:any` vs `oneOf` — non exprimable en tag, cf. inventaire). Semantics H3 Partie B :
      `ApiError` désormais COMMUN doc∩yaml, parité code(desc+example)/message(desc)/details(desc)/
      field_errors(desc) VÉRIFIÉE ; `rootDescAllowlist` INCHANGÉE (5 entrées, anti-caducité OK) ;
      inventaire du test mis à jour (ApiErrorSchema résolu). **5 exemples d'erreur (item 7
      inventaire)** : exemples de RÉPONSE (components.responses BadRequest/Unauthorized/NotFound/
      InternalError/Conflict), NON exprimables en tag struct (full-object, 5 distincts) →
      restent au fragment manuel H6 ; SEUL l'exemple de CHAMP `code=player_not_found` est porté
      en tag. Gates : gofmt -l vide, `go build ./...` (0), `go vet ./...` (0), `go test
      ./internal/api/...` (ok : drift/semantics/metadata/contract/fidélité), `go test ./...`
      (117 ok, 0 fail), `make go-api-lint` (0 issue) — TOUS verts. Périmètre : humacore.go,
      openapi.yaml (édition ciblée), openapi_schema_semantics_test.go (commentaire inventaire).
      Aucun commit (superviseur). [x]
- [x] **H5 — Migrer groups.go vers Huma (S).** FAIT (2026-07-25, agent Opus). Les 7 routes
      groups (`server_apiv1.go` : `r.Get/Post/Patch/Delete("/groups...")` writeJSON manuel)
      passent en huma typé : structs input/output (`groupIDInput`, `groupBodyInput`,
      `groupIDBodyInput`, `groupMemberInput` ; sorties `groupsListOutput`/`groupCreatedOutput`
      (201)/`groupOutput` (200)/`inviteCreatedOutput` (201)/`groupsNoContent` (204)), corps via
      `RawBody` + décodage maison (préserve le 400 {invalid_body}, PAS le 422 Huma) ; erreurs
      via le modèle UNIFIÉ H4 (`humacore.NewError`). Enregistrement via le document PARTAGÉ :
      `groupsHandler.Mount(r, apiOpt)` dans le groupe middleware-only RequireAuth (docPrefix
      `/api/v1`, comme les voisins settings/setup/sync). `humacore.Op` en PARITÉ avec le yaml
      (operationId listMyGroups/createGroup/renameGroup/deleteGroup/createGroupInvite/leaveGroup/
      removeGroupMember, tag `groups`) ; `MarkRequestBodyOptional` sur POST /groups/{id}/invites
      (corps optionnel). **MÊME contrat JSON runtime** : les 5 tests groups passent avec ZÉRO
      changement d'assertion (harness `groups_test.go` : `h.Mount(r)` + URLs `/groups` sans
      trailing-slash — Content-Type NON requis pour RawBody ici). openapi.yaml : les 7 routes
      étaient déjà documentées (parité vérifiée) ; Huma dérive désormais les schémas de sortie
      `Group` + `GroupMember` (ajoutés à components.schemas, MISSING résorbé). Test de fidélité
      H1 : les 7 routes comptent MAINTENANT comme Huma (exclusion chi-brut `/api/v1/groups`
      RETIRÉE de `isKnownChiBrut`) — 173 ops Huma fusionnées (166+7), 167 paths (162+5).
      Metadata H2 : 173 ops, metadata OK=173, parité 163 (0 échec). Gates : gofmt -l vide,
      `go build ./...` (0), `go vet ./...` (0), `go test ./internal/api/...` (ok : contract_test
      + TestSharedOpenAPIDocCoversAllHumaRoutes + openapi_operation_metadata_test + drift +
      semantics), `go test ./...` (green), `make go-api-lint` : 0 issue SUR MES FICHIERS (les
      issues résiduelles goconst/gofmt/staticcheck/unparam sont dans les fichiers d'un agent
      parallèle — halo_5/medal_category*, scheduler/auto_sync_notify* — hors périmètre H5, non
      corrigées). Périmètre : handlers/groups.go (réécrit), handlers/groups_test.go (harness),
      server_apiv1.go (Mount), shared_openapi_doc_test.go (exclusion retirée), openapi.yaml
      (Group/GroupMember). Aucun commit (superviseur). [x]
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

## Découvertes (H2 — à traiter en H6, PAS maintenant)

- **45 call-sites hors parité EXACTE**, statués mais à intégrer au yaml régénéré en H6 :
  - **6 divergences de nom de PARAMÈTRE** (le CODE fait foi, CLAUDE.md) — operationId +
    summary + tags REPRIS du yaml sémantique équivalent :
    `/admin/users/{username}` (yaml `{user_id}`), `/admin/users/{username}/role`,
    `/admin/users/{username}/password`, `/admin/invites/{code}` (yaml `{invite_id}`),
    `/players/{player_slug}/notifications/{id}` + `.../{id}/unread` (yaml
    `{notification_id}`). H6 régénérera ces chemins avec le nom de param Go → diff attendu.
  - **1 divergence SÉMANTIQUE** : le yaml documente `GET /watcher/auth/{provider}`
    (`getWatcherAuthCallback`, « Callback OAuth ») mais la vraie route Go est
    `GET /watcher/auth/{attempt_id}` = `handleGetAuthStatus` (statut d'une tentative).
    L'entrée yaml est OBSOLÈTE → operationId INVENTÉ `getWatcherAuthStatus` (tag auth).
  - **38 routes Go-only NON documentées dans le yaml manuel** (operationId/summary/tags
    inventés, convention du yaml suivie) : **29 Prestige** (challenges/arcs/prestige/
    templates/squads/squad-challenges/pilot-mode) → NOUVEAU tag `prestige` ; **6 multi-titre**
    (`/titles/{slug}/field-mappings|capabilities|feature-matrix|catalog/{playlists,pairs,maps}`)
    → NOUVEAU tag `titles` ; **3 diag auto-sync** (`/_diag/auto-sync/{snapshot,run,probe}`)
    → tag `diagnostics`.
- **Nouveaux tags introduits** (`prestige`, `titles`) : H6 devra ajouter leur description
  au bloc `tags:` régénéré (ces routes n'existaient pas dans le yaml manuel).
- **Incohérence pré-existante du yaml manuel préservée pour parité** : le tag `jobs`
  (op `GET /jobs/{job_id}`) est UTILISÉ sans être DÉCLARÉ dans le bloc `tags:`. Non corrigé
  (hors périmètre H2) — H6 régénérera le bloc complet.

## Inventaire fermé H3 — sémantiques NON exprimables en tags struct

Livrable de revue. Chaque entrée = sémantique du yaml manuel qui NE peut PAS être portée
en tag Huma de champ ; destination indiquée. Le garde-rail Partie B allowliste UNIQUEMENT
les items observés comme perdus sur un schéma présent dans le doc de démo (les descriptions
racine) ; le reste n'apparaît pas dans le doc généré (schémas non-Huma) donc n'est pas
« perdu » au sens du diff — il vit dans le fragment manuel H6.

1. **Descriptions de SCHÉMA RACINE** (Huma n'a AUCUN tag type-level ; `SchemaProvider`
   remplace la génération, `SchemaTransformer` ajoute une méthode — les deux hors périmètre
   « tag »). 7 au total ; allowlistées quand le schéma est généré (5 démo-présents) :
   `DeviceFlowStartResponse`, `DeviceFlowStatusResponse`, `MatchScoreboardObjective`,
   `ObjectiveAggregate`, `PlayerMediaAudioConfig` (allowlist Partie B) ; +`SettingsResponse`,
   `CascadeInput` (schéma non démo-présent). **Destination** : `SchemaTransformer` léger
   (mécanisme identifié) OU fragment manuel H6.
2. **Schémas SANS type Go Huma-généré** (chi-brut / champ Go scalaire / legacy) — leur
   schéma yaml est écrit à la main, Huma ne le régénère pas → **fragment manuel H6** :
   - `ApiErrorSchema` (code desc+example, message desc, retryable default) → **RÉSOLU H4** :
     fusionné dans le type Go huma-généré `ApiError` (enrichi details/field_errors,
     code desc+example + message desc portés en tags) ; `ApiErrorSchema`/`FieldErrorSchema`
     supprimés, 7 `$ref` repointés. `retryable default:false` = trivial (zéro), NON posé
     (cohérent item 5). `details:oneOf[object,array]` non exprimable en tag (`any` → `{}`)
     → seul reliquat au fragment/SchemaTransformer H6 si l'on veut restaurer le `oneOf`.
   - `FreshnessInfo` (source enum+default, sync_status enum+default) — le champ Go est
     `*string` / `interface{}`, pas un objet.
   - `SortSpec` (direction enum+default), `MatchHistoryExportRequest` (format default) —
     route export chi-brut, aucun type Go.
   - `PlotlyFigurePayload` (config_key enum+default, figure desc) — legacy Python (mort).
   - `BackupStatusResponse` (8 desc), `BackupRunResult` (4 desc), `IntegrityCheckResult`
     (3 desc) — admin backup chi-brut, aucun type Go.
   - `AssociatedMediaItem`, `MediaItemRow` (liked default false) — default trivial (zéro).
3. **Schémas d'ENTRÉE décodés à la main** (`RawBody` / `var req domain.X`) — jamais générés
   par Huma → fragment manuel H6 : `CompareRequest` (title_slug default "hi"),
   `MatchHistoryQueryRequest` (include_export_hint default), `PaginationRequest` (page/
   page_size defaults), `MediaPageRequest` (page/page_size defaults), `SessionContextRequest`
   (locale enum), `ExplorerMatchesQueryRequest` (squad_scope enum+default). NB : les enums
   de ces inputs seraient de toute façon posés avec prudence (validation) — ici sans objet
   puisque le type n'est pas généré.
4. **Dérives de NOM yaml↔Go** — la sémantique portable a été posée sur le VRAI type Go ; le
   nom yaml diverge (à réconcilier H6/H7) : `ExplorerMatchRow`→`ExplorerMatchesRow` (posé :
   had_bot_teammate desc, experience_type_label default), `GamertagSuggestion`→
   `GamertagSearchResult` (posé : score default), `CascadeInput`→`CascadeFilter` (root desc
   → cat. 1), `CareerTopMatch` yaml (variant enum) SANS équivalent DTO (le `canonical.
   CareerTopMatch` n'a ni json ni `variant`) → fragment manuel.
5. **Defaults TRIVIAUX (valeur zéro)** NON posés (no-op : Huma n'applique pas de default en
   sortie et la valeur zéro est déjà false/0) : `ExplorerMatchesRow` deaths/kills/kda/
   is_with_friends/had_bot_teammate, `GamertagSearchResult` exact_match, `AssociatedMediaItem`/
   `MediaItemRow` liked, `ApiErrorSchema` retryable.
6. **Sémantique de PARAMÈTRE** (query/path, hors schéma.champ) → **H6** (Huma régénère les
   params depuis les input structs ; enums d'input à poser avec prudence sur ces structs) :
   `default: 50` (limit), `default: ""` (recherche q), `default: 8` (limit) ; `enum: [fr,en]`
   (lang field-mappings), `enum: [pending,accepted,dismissed,superseded,obsoleted,stale]`
   (proposals status), `enum: [xbox,steam]` (watcher provider — sur chemin yaml OBSOLÈTE, cf.
   Découvertes H2). **Enums d'input NON posés par prudence** : les 3 ci-dessus (validation
   422 potentielle + exhaustivité non garantie côté route).
7. **Exemples de RÉPONSE** (5 exemples d'erreur dans `components.responses`, désormais sur
   l'`ApiError` unifié H4) : bad_request, auth_required, player_not_found, internal_error,
   last_active_title. Full-object, 5 distincts, NON exprimables en tag struct → **restent au
   fragment manuel H6** (components.responses préservé). L'exemple de CHAMP
   `code=player_not_found` est, lui, porté en tag Huma (H4).
8. **16 request bodies `RawBody`** (contrat d'erreur 400 historique, non dérivable) →
   fragment manuel H6 (déjà acté dans l'état des lieux).

## Journal

- 2026-07-24 : recon + plan posés. Séquencement : dernier des gros lots v7.2.
- 2026-07-25 : revue plan-review intégrée : spike H0.5 bloquant ajouté, H1 reformulé
  « document partagé » (pas instance unique), comptages corrigés (71 Mount), inventaire
  fermé H3, précédence + golden du fragment H6, garde sémantique generated.ts H7,
  outillage H0 tranché (kin-openapi), blindage plan-execution.
- 2026-07-25 : **H4 + H5 EXÉCUTÉS (agent Opus).** H4 : modèle d'erreur UNIFIÉ — `humacore.apiError`
  enrichi (`Details any` + `FieldErrors []FieldError`, omitempty → contrat runtime {code,message,
  retryable} inchangé, tags doc/example H3-style) ; yaml convergé (7 `$ref` ApiErrorSchema →
  `ApiError` réécrit fidèle au riche via emit Huma, `FieldError` ajouté, ApiErrorSchema/
  FieldErrorSchema supprimés) ; compat front `client.ts` vérifiée (lit déjà details/field_errors) ;
  5 exemples d'erreur = response-level → restent fragment H6 (seul l'exemple champ
  code=player_not_found porté en tag). H5 : 7 routes groups.go writeJSON → Huma typé
  (Mount + humacore.Op parité yaml + RawBody + modèle d'erreur H4), schémas Group/GroupMember
  dérivés ajoutés au yaml, exclusion chi-brut retirée (fidélité H1 : 173 ops/167 paths). Contrat
  JSON runtime IDENTIQUE (5 tests groups, 0 changement d'assertion). Gates : gofmt/build/vet/
  `go test ./internal/api/...`/`go test ./...` verts ; lint 0 issue sur mes fichiers (résidus =
  fichiers d'un agent parallèle V72-31, hors périmètre, notés non corrigés). Aucun commit. H6 = suivant.
- 2026-07-25 : **H3 EXÉCUTÉ (agent Opus).** Sémantique de schéma.champ portée en tags
      Huma sur 10 types Go DTO : **19 descriptions + 11 enums + 5 defaults** (bootstrap/auth/
      engagement_score/media_audio_config/settings/admin_monitoring/explorer/career). Constat
      mesuré : les cibles 402/21/30/6 du recon = comptages bruts de tout le doc (175/232 desc
      = réponses/params, niveau opération) ; le portable schéma.champ = 31 schémas dont 11
      seulement ont un type Go Huma-généré (les « Request » = RawBody/décodage manuel, non
      générés ; 10 schémas sans type Go). Enums d'input = 0 risque 422 (types instrumentés =
      outputs ou RawBody, vérifié call-site). Descriptions racine non portables (pas de tag
      type-level Huma) → inventaire fermé (section dédiée du plan, 8 catégories). Gate :
      nouveau `openapi_schema_semantics_test.go` — Partie A (registre neuf, 35 sémantiques /
      10 types) + Partie B (doc partagé vs yaml, allowlist=inventaire+compteurs : 28 parité,
      5 pertes = descriptions racine allowlistées, 0 perte hors inventaire). gofmt/build/vet/
      `go test ./internal/api/...`/`go test ./...` (118 ok)/`make go-api-lint` (0) verts. 2
      descriptions raccourcies (lll ≤ 220). Périmètre : 8 fichiers domain + le test — ZÉRO
      autre fichier Go, ZÉRO apps/web. Aucun commit (superviseur). H4 = suivant.
- 2026-07-25 : **H2 EXÉCUTÉ (agent Opus).** 204 call-sites `huma.X(api,` portent
  operationID + summary + tags via le helper variadique `humacore.Op` (nouveau
  `humacore/operation.go`), appliqué mécaniquement (générateur go/ast jetable). 159 en
  parité exacte avec api/openapi.yaml, 45 supplémentés (6 renommages de param + 1
  divergence watcher + 38 routes Go-only : Prestige/multi-titre/diag) — détail section
  Découvertes. Gate : nouveau `openapi_operation_metadata_test.go` (166 ops, metadata
  OK=166, parité 156 vérifiée, 0 échec). Réconciliation 204 statique → 166 runtime : 29
  Prestige + 3 catalog + 3 diag auto-sync (non montés en démo) + 3 assets_metadata (double
  branche). gofmt/build/vet/`go test ./...`/`make go-api-lint` TOUS verts. Aucun commit.
  Périmètre : `humacore/operation.go`, `huma_routes.go` (+import), 74 handlers,
  `openapi_operation_metadata_test.go` — ZÉRO autre fichier Go. H3 = suivant.
- 2026-07-25 : **H1 EXÉCUTÉ (agent Opus).** Document OpenAPI PARTAGÉ câblé sur les
  71 `Mount()` + 3 registres inline via `NewAPI` variadic + `WithSharedDoc`. Périmètre :
  humacore.go (MountOption/WithSharedDoc, `OnOperationRegistered` + observingAdapter,
  `subrouterAPI.DocPrefix`, `MarkRequestBodyOptional` prefix-aware) ; huma_setup.go
  (newHumaAPI variadic) ; 74 fichiers handlers (signature Mount) ; server.go +
  server_apiv1.go (config partagée, options scopées, `apiV1BasePath`, 3e retour doc) ;
  wire/server_admin_monitoring.go (param option) ; cmd/server/main.go +
  contract_helpers_test.go (3e retour) ; nouveau `shared_openapi_doc_test.go` (fidélité) ;
  route_collision_test.go (`TestNoDuplicateRouteRegistration` sur le nouveau hook). Aucun
  commit (superviseur). Comportement HTTP INCHANGÉ (0 modif d'assertion). H2 = suivant.
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
