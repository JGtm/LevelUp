# HANDOFF — Halo 5: Guardians comme 2e titre EXPERIMENTAL

> **Créé** : 2026-06-18 (session multi-titre péripherie). **Statut** : faisabilité PROUVÉE, pas commencé.
> **À faire APRÈS** : finir l'auto-gen openapi (shims types.ts + 69 DIVERGENT) + le vocab Halo cosmétique.
> **Branche** : `feat/multititre-peripherie`. **User de test** : **JGtm** (l'utilisateur).
> Mémoire liée : `project_halo5_experimental_direction`.

## 0. TL;DR pour reprendre

Halo 5 = premier VRAI 2e titre, en mode **experimental**. **Toute l'infra d'accueil est déjà prête** (registre config, switcher câblé, provisioning DB au boot, gating capability, drift-detector de contrat). Ce qui reste = le **title-specific** : auth + client + adapter de données + mappings + **feature-matrix fine** (déterminer quoi est affichable).

## 1. Décision de source de données (TRANCHÉE)

**Source retenue = endpoints internes façon `cryptum-halodotapi`, PAS l'API officielle.**

| | `developer.haloapi.com` (officiel) | `cryptum-halodotapi` (interne) |
|---|---|---|
| Auth | header `Ocp-Apim-Subscription-Key` (simple) | **SpartanToken Xbox Live** v2/v3 |
| Données | service records + metadata, **DÉGRADÉ** (sonde 2026-06-18 : Snip3down tout à zéro) | matchs + **events**, Arena/Warzone/Campaign/Custom, customisation, REQ, UGC — **bien plus complet** |
| Langage | REST (n'importe quel client) | JS/Node (= **référence** pour notre adapter Go) |

**Pourquoi cryptum/interne** : (1) données bien plus riches ; (2) **le projet a DÉJÀ la machinerie Xbox→SpartanToken** (pool d'auth Halo Infinite) → on la **réutilise** pour Halo 5 (audience/relying-party différente à câbler). cryptum = doc des URLs/shapes/audience.

**Clé API officielle** (subscription key haloapi.com) : fournie par l'user 2026-06-18, **en clair dans le chat → À RÉGÉNÉRER**. Si jamais utilisée (fallback) : **env/`.env.local` gitignoré, JAMAIS versionnée**. Repo cryptum : https://github.com/Alexis-Bize/cryptum-halodotapi

### Sonde de faisabilité (faite, OK)
- `GET https://www.haloapi.com/metadata/h5/metadata/playlists` (header clé) → 200, playlists Halo 5 (Free-for-All, SWAT, Breakout…).
- `GET https://www.haloapi.com/stats/h5/servicerecords/arena?players=<gt>` → 200 mais **stats à zéro** (API officielle dégradée) → confirme qu'il faut les endpoints internes.

## 2. ENJEU ARCHITECTURAL MAJEUR — feature-matrix fine par titre

**Halo 5 a des données LIMITÉES** : RIEN au niveau des **highlight events** ni des **chunks de films** qu'on exploite pour Halo Infinite (kill-feed, engagement score, intensité par phase, timeline T0, arme-par-kill, etc.). **Beaucoup de cards / graphes / tableaux ne pourront PAS être générés** (partiellement ou du tout).

→ **Il FAUT déterminer en amont, par titre, ce qui est dispo, pour que le frontend sache quoi présenter.** C'est le système de **capabilities / feature-matrix MAIS à granularité fine** — au-delà des 11 capabilities actuelles (matchmaking/firefight/forge/media/ranked/career/asset.images/achievements/engagement/lusr/world.leaderboard), il faut une granularité **par surface** (card/chart/colonne).

Briques existantes à exploiter :
- **Capability gating front** : `apps/web/src/lib/capabilities/` (`useCapability`/`FeatureGate`/`RouteCapabilityGate`). Déjà câblé NO-OP halo_infinite ; Explorer + Timeseries rang gatés ce jour.
- **Feature-matrix backend** : `internal/games/mappings/` (capabilities.toml fines : `match.history`, `match.detail.core`, `match.skill.snapshot`, `career.progression`, `pve.firefight_stats`, `analytics.timeseries`, `match.scoreboard.extra`, `citations.engine`, `engagement.score`) + valeurs `supported`/`degraded`/`not_exposed`.
- **Drift-detector de contrat** (Lever B, ce jour) : `internal/api/openapi_schema_drift_test.go` — sait quels schémas/types existent.

**Travail Halo 5** : mapper chaque surface produit → capability fine, déclarer Halo 5 en `not_exposed` pour tout ce qui dépend des events/films. Le front masque déjà via gating. **À étendre** là où la granularité 11-capabilities est trop grosse (ex : un chart de timeseries qui marche pour kills mais pas pour engagement).

Capabilities Halo 5 plausibles : `matchmaking` ✅ (Arena+Warzone), `ranked` ✅ (CSR Arena), `career` ✅ (Spartan Rank / SR), `asset.images` ✅ (maps/playlists/emblems via metadata). **PAS** : `lusr` (calcul local sur events Infinite), `engagement` (dépend events), `forge`/`media` (pas via API), highlight-events, films.

## 3. Infra d'accueil PRÊTE (ne pas reconstruire)

- **Registre piloté par config** : déposer `config/titles/halo_5/` (`title.toml` + `mappings/{capabilities,fields,assets,outcomes}.toml` + `constants.toml` + `auth.toml`) → découvert au boot par `title.LoadTitlesIntoRegistry` (`internal/domain/title/config_loader.go`). Modèle = `config/titles/synthetic_title_b/`.
- **Switcher UI** : `apps/web/src/components/shell/TitleSwitcher.tsx` (menu Paramètres NavL1). Halo 5 apparaîtra automatiquement dès qu'il est dans `bootstrap.available_titles` ; `coming_soon` → « Bientôt disponible » (désactivé) ; `active` → sélectionnable.
- **Provisioning DB au boot** : `cmd/server/main.go` `provisionAdditionalActiveTitles` (itère `reg.Active()`, crée+migre les DB par titre). Ne provisionne PAS les `coming_soon`.
- **Adapters** (le vrai travail title-specific) : implémenter `games.TitleDataAdapter` (Load*) + `TitleSemanticAdapter` dans `internal/games/halo_5/` (modèle = `internal/games/halo_infinite/`). Types de retour = **canonical** (`internal/games/canonical/` : MatchSummary, PlayerStats, CareerSnapshot…).
- **Validateur boot** : `internal/games/mappings/validate.go` (RequiredTOMLFor : fields+capabilities toujours ; +assets si CapAssetImages ; +outcomes si CapMatchmaking). N'`os.Exit` que sur titres ACTIFS.

## 4. Plan staged

### Phase 0 — Skeleton (≈30 min, faible risque)
`config/titles/halo_5/` complet, **status `coming_soon`**, capabilities RÉELLES Halo 5 (matchmaking, ranked, career, asset.images — pas lusr/engagement/firefight). → Halo 5 visible dans le switcher « Bientôt disponible ». Aucun adapter requis (pas servi). Modèle exact = synthetic_title_b mais métadonnées Halo 5 réelles.
- **Oracle** : test type `internal/games/halo_5/skeleton_test.go` (registre découvre coming_soon + capabilities + endpoints distincts) ; `go test ./internal/games/mappings/` (validateur).

### Phase 1 — Experimental read-only (≈1-2 sessions, risque moyen)
1. **Auth Halo 5** : étendre le pool Xbox→SpartanToken pour l'audience Halo 5 (relying-party différente de Halo Infinite). Réf : cryptum (auth v2/v3). Réutiliser `internal/platform/halo` / `auth.MultiUserTokenStore`.
2. **Client interne** Halo 5 : `internal/games/halo_5/client.go` (endpoints stats/matchs/service-record ; réf cryptum pour URLs+shapes).
3. **Adapter data** minimal : 1 surface d'abord (recommandé **historique de matchs** = plus parlant ; ou service record = plus simple). Mapper JSON Halo 5 → canonical `MatchSummary`/`PlayerStats`.
4. **Feature-matrix** : déclarer `not_exposed` tout ce qui dépend events/films.
5. **Statut → `active`** (servi + provisionné) une fois l'adapter en place.
- **Oracle** : page front s'affiche avec données réelles JGtm, surfaces non-dispo masquées (pas vides).

### Phase 2+ — Complet (multi-sessions)
Sync/ingestion DB, toutes surfaces possibles, nettoyage vocab cosmétique Halo→neutre (cf. tâche en cours).

## 5. Pièges connus / notes
- L'API officielle est dégradée → ne PAS se baser dessus pour la richesse. cryptum/interne = la voie.
- SpartanToken Halo 5 ≠ Halo Infinite (audience). C'est le point d'effort auth.
- Ne JAMAIS comparer `slug == "halo_5"` (archlint `no_slug_comparison`) — tout par capability.
- Vocab Halo cosmétique (admin « API Halo », Lab « Waypoint », `HINF-CSR`) : à neutraliser ; Halo 5 va l'exercer (ses badges CSR ≠ HINF).
- Réfs : `.ai/PLAN_MULTITITRE_INDEX.md` (statut infra), `.ai/PLAN_TITLE_AGNOSTIC_TRACKER.md`, mémoire `project_halo5_experimental_direction`.
