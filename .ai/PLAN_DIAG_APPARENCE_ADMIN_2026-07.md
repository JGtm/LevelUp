# PLAN — Diagnostic apparence Spartan ID dans la page admin — 2026-07

> Plan d'exécution pour agent (Opus). Contrat : skill `plan-execution` (ordre strict,
> une étape à la fois, gate par lot, statuts [x]/[~]/[!], zéro fix hors périmètre).
> Rédigé le 2026-07-08 après l'épisode « bannière JGtm figée » — toutes les décisions
> produit ci-dessous sont TRANCHÉES, ne pas les rouvrir.

## Contexte et objectif

Épisode 2026-07-08 (thought_log même date) : la bannière de JGtm ne « se mettait pas à
jour ». Root cause : l'emblème équipé (`3806589-SpartanEmblem`, nouvelle génération)
n'a AUCUNE image de nameplate publiée par Microsoft (absent de `mapping.json`, aucune
cfg positive au CMS, 404 CDN). L'app servait correctement la dernière bannière connue
(directive : champs d'apparence indépendants, jamais vides) — mais rien ne rendait ce
verdict visible : le diagnostic a nécessité 3 CLI (`diag_appearance`,
`diag_emblem_mapping`, `diag_emblem_colors`) et la lecture des logs prod.

**Objectif** : exposer dans la page ADMIN un diagnostic à la demande, par joueur suivi,
des composants du Spartan ID (bannière, emblème, backdrop, service tag) avec un
VERDICT ACTIONNABLE par composant : ce qui est servi, ce que le live résout, pourquoi,
et surtout s'il y a quelque chose à faire ou non.

**Critère de succès global** : depuis la page admin, un clic « Diagnostiquer » sur un
joueur suivi affiche en < 10 s le verdict des 4 composants avec explication FR/EN ;
le cas JGtm (emblème sans nameplate upstream) s'affiche « Rien à faire — dernière
bannière connue servie par design » ; suites Go + web vertes.

**Décisions produit actées (ne pas rediscuter)** :
- Surface = page **admin** (introspection à la demande). PAS le monitoring (chantier
  refonte séparé, PLAN_MONITORING_*_2026-07) ; aucune alerte/notification émise.
- Périmètre v1 = **joueurs suivis** (`db_profiles.json`) uniquement — pas de tiers.
- Calcul **à la demande** (bouton par joueur), pas de polling ni de persistance
  nouvelle. L'historique passif existant reste `career_progression.last_fetch_status`.
- Verdicts (enum fermé, 5 valeurs) :
  - `ok` — le live résout, la valeur servie est à jour ;
  - `upstream_missing` — définitif côté Microsoft (ex. nameplate absente de
    mapping.json + aucune cfg positive) → « Rien à faire, dernière valeur connue
    servie par design ; se réparera seul si Microsoft publie l'image » ;
  - `transient` — échec réseau/HTTP/parse indéterminé → « Se répare seul au prochain
    refresh » ;
  - `auth_required` — tokens absents/morts (`auth_missing`, 401/403 owner) →
    « Action requise : réauthentification » (renvoyer vers le flux SSO existant) ;
  - `not_supported` — le titre ne fournit pas ce composant via ce pipeline
    (capability absente) → dégradation propre, jamais un faux « cassé ».
- La sémantique de LECTURE de l'apparence ne bouge PAS : champs 100 % indépendants,
  chacun sert sa dernière valeur non vide, jamais vide (directive verrouillée par
  tests le 2026-07-08 — toute condition croisée bannière↔emblème est interdite).
- Pas de synthèse locale de bannière dans ce plan (piste future distincte).

## Références de code (vérifier sur pièces avant chaque lot — le code a pu bouger)

| Élément | Emplacement |
|---|---|
| Fetch appearance + résolution images | `apps/go-api/internal/sync/haloclient/halo_client_career.go` (`GetSpartanCustomization`, `resolveCustomizationImageURL`) |
| Resolver nameplate (verdict définitif/transitoire DÉJÀ distingué) | `apps/go-api/internal/sync/haloclient/spartan_nameplate_resolver.go` (`ResolveNameplateURL`, `resolvePositiveEmblemCfg` → retourne `(cfg, definitive)`) |
| Statuts fetch persistés | `apps/go-api/internal/service/career_live_partial.go` (`FetchStatus` : ok/api_empty/forbidden_403/auth_missing/failed) |
| Dernières valeurs servies | `apps/go-api/internal/platform/duckdb/career_live_repo.go` (`LoadLastCareerRank`) |
| Tokens serveur par joueur | `internal/platform/auth` `MultiUserTokenStore` + `RefreshHaloTokensViaStoreFirst` (ADR 0023) ; modèle d'usage : `cmd/diag_appearance/main.go` |
| Rate budget par compte | `internal/platform/ratebudget` (`ForXUID`) — cf. `CareerFetcherFactoryFromTokens` |
| Endpoints admin existants (modèle de montage + auth) | `apps/go-api/internal/api/handlers/` routes `/_admin/*` (ex. `_admin/progression/backfill/{player_slug}`) + wiring `internal/api/wire/` |
| Garde-rail hosts | `internal/archlint/no_halowaypoint_literal_test.go` (aucun host en dur — passer par le resolver d'endpoints) |
| Capabilities | `HasCapability` / `CapabilityMap` + `config/titles/{slug}/mappings/capabilities.toml` (H5 : `spartan_customizer` — bannière = rendu Spartan, pipeline `games/halo_5/livesync/appearance_persist.go`) |
| Panneau admin front existant (modèle) | `apps/web/src/features/admin/overview/DataHealthPanel.tsx` |
| i18n admin | `apps/web/src/lib/i18n/manifests/admin.toml` → regen `node apps/web/scripts/build_i18n_manifests.mjs` |
| Query keys | `apps/web/src/lib/query/keys.ts` |
| OpenAPI (manuel, migration Huma terminée) | `apps/go-api/api/openapi.yaml` → `make generate-types` |
| Cas de test canonique | emblème `Inventory/Spartan/Emblems/3806589-SpartanEmblem.json`, cfg `-1766636888` → `upstream_missing` (payloads réels dans le thought_log 2026-07-08) |

## Branche et livraison

- Branche : **`feat/admin-diag-apparence`**, créée depuis `main` À JOUR.
- **Dépendance dure** : le commit `diag(spartan-id)` de la branche
  `refactor/audits-2026-07` (resolver `(cfg, definitive)`, tests directive apparence)
  doit être mergé dans `main` AVANT de commencer. Sinon : le signaler à l'utilisateur
  et attendre — ne pas cherry-pick.
- 1 tâche = 1 branche, N commits (un par lot). Prévenir avant tout merge dans `main`
  (= deploy prod auto).

## Lot A — haloclient : résolution diagnostique structurée (Go)

Objectif : exposer le POURQUOI de la résolution sans dupliquer la logique existante
(règle ≤ 2 copies : refactorer, ne pas copier).

- [ ] A1. Créer dans `internal/sync/haloclient/` un type exporté
      `AppearanceDiagnosis` : par composant {`ServedFrom` (live/carry), `ResolvedURL`,
      `Verdict` (enum 5 valeurs ci-dessus), `Detail` (clé technique : mapping_miss,
      no_positive_cfg, cms_http_error, no_banner_field, …)}.
- [ ] A2. Extraire de `ResolveNameplateURL` une fonction interne unique qui retourne
      (url, verdict, detail) ; `ResolveNameplateURL` devient un wrapper qui n'en garde
      que l'url (comportement byte-identique, logs inchangés). Ajouter une fonction
      exportée `DiagnoseNameplate(ctx, emblemPath, cfg, tokens…)` sur cette même
      interne. INTERDIT : dupliquer le fetch mapping/CMS.
- [ ] A3. Pendant emblème/backdrop : diagnostiquer via `resolveCustomizationImageURL`
      (succès/échec HTTP → ok/transient). Service tag : présent/absent du payload.
- [ ] A4. Tests unitaires haloclient : verdicts sur cache mapping seedé
      (`seedEmblemMappingCacheForTest`), cas `upstream_missing` (JSON CMS réel du cas
      3806589 en fixture), cas transient (HTTP KO), cas cfg positive directe.

**Gate A** : `cd apps/go-api && go test ./internal/sync/haloclient/... -count=1 &&
go vet ./internal/sync/...` — exit 0, zéro FAIL. Vérifier que
`TestResolveNameplateURL_*` existants passent SANS modification de leurs attentes.

## Lot B — service + endpoint admin (Go)

- [ ] B1. Service `internal/service/` (nouveau fichier, ex. `appearance_diag_service.go`) :
      pour un `player_slug` → xuid via db_profiles ; tokens via
      `RefreshHaloTokensViaStoreFirst` (échec → verdict global `auth_required`, pas
      d'erreur 500) ; fetch `GetSpartanCustomization` + diagnostics lot A ; lecture
      des valeurs SERVIES via `LoadLastCareerRank` + dernier `last_fetch_status` ;
      assemblage DTO. Limiteur `ratebudget.ForXUID`. Aucune écriture DB.
- [ ] B2. Multi-titre : brancher sur capabilities (JAMAIS `slug ==`). Titre avec
      `spartan_customizer` (H5) : bannière/emblème = pipeline appearance dédié →
      verdict `not_supported` pour la résolution nameplate + affichage des valeurs
      servies uniquement. Dégradation `ErrCapabilityNotSupported` → réponse partielle
      propre, pas de 500.
- [ ] B3. Handler Huma `GET /_admin/diag/appearance/{player_slug}` dans
      `internal/api/handlers/` (zéro logique métier), wiring dans `internal/api/wire/`,
      même auth/gating que les autres routes `/_admin/*`. OpenAPI manuel + DTO.
- [ ] B4. Logging : `slog.InfoContext` début/fin (player, titleSlug, duration),
      `ErrorContext` avec `"err"` sur échec non-trivial. Pas de token en clair.
- [ ] B5. Tests service avec mock fetcher (verdicts par scénario, y compris tokens
      absents et capability absente) + test handler `httptest` (200 nominal, 404 slug
      inconnu).

**Gate B** : `cd apps/go-api && go test ./internal/service/... ./internal/api/... -count=1
&& go vet ./...` exit 0 ; `grep -rn "slug ==" internal/service/appearance_diag*` vide ;
route visible dans `api/openapi.yaml` ; `make generate-types` sans diff front inattendu.

## Lot C — panneau admin (front)

- [ ] C1. Section « Diagnostic apparence Spartan » dans la page admin (calquer la
      structure de `DataHealthPanel.tsx`) : sélecteur joueur suivi + bouton
      « Diagnostiquer » (mutation à la demande, pas de refetch auto).
- [ ] C2. Rendu par composant (bannière/emblème/backdrop/service tag) : vignette de la
      valeur servie, badge verdict, explication dépliable (le POURQUOI + « quoi
      faire » : rien / attendre / réauthentifier). Aucune couleur hex/Tailwind :
      tokens sémantiques uniquement (skill `color-tokens`).
- [ ] C3. i18n FR **et** EN dans `admin.toml` (labels verdicts + explications + CTA),
      regen manifests. Pas d'anglicisme côté FR.
- [ ] C4. Query key dans `lib/query/keys.ts` ; types depuis `generated.ts` (pas de
      types manuels dupliqués).
- [ ] C5. Test vitest du composant (au minimum : rendu des 5 verdicts depuis des
      fixtures, mock API).

**Gate C** : `cd apps/web && Remove-Item -Recurse -Force node_modules\.tmp ;
npm run typecheck && npm run lint && npm run test` — exit 0 (purge tsbuildinfo
OBLIGATOIRE avant le typecheck de clôture).

## Lot D — clôture

- [ ] D1. Vérification manuelle dev local : diagnostiquer JGtm → bannière =
      `upstream_missing` avec l'explication « rien à faire » ; un joueur sain → `ok`.
- [ ] D2. Skill `delivery-checklist` complet (dont : aucun TODO introduit, fichiers
      ≤ 500 L, fonctions ≤ 80 L).
- [ ] D3. Entrée `thought_log.md` + mise à jour de ce plan (statuts).
- [ ] D4. CI de branche verte (`gh run list --branch feat/admin-diag-apparence`).

## Découvertes (à consigner ici, NE PAS traiter hors périmètre)

- (vide)

## Protocole de reprise de session

Lire : ce plan (statuts des cases) → `thought_log.md` entrées `feat/admin-diag-apparence`
→ `git log --oneline -10` sur la branche. Reprendre au premier item non coché du
premier lot non clos. Un lot est « clos » quand tous ses items sont statués ET son
gate est passé (code de sortie vérifié, pas seulement la sortie filtrée).

## Estimation

Moyen : ~1 journée agent (A/B = gros du travail ; C cadré par les modèles existants).
