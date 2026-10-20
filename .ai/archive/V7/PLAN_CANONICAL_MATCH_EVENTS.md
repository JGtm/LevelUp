# PLAN — Canonical MatchEvents (timeline d'événements de match, multi-titre)

> **Créé** : 2026-06-19 (suite à la sonde Halo 5 events live, cf. `HANDOFF_HALO5_EXPERIMENTAL.md` §0-quater).
> **Branche** : `feat/multititre-peripherie` (séquentiel avec l'arc multi-titre ; commits dédiés).
> **GATE** : à finaliser (Phase 0+1+3) **AVANT l'activation Halo 5** (1b) — pour activer Halo 5 avec sa richesse native, pas une v1 dégradée à refaire. Réf gate : `HANDOFF_MULTITITRE_ACTIVATION.md` §5.
> **Origine** : la sonde a prouvé que Halo 5 sert NATIVEMENT (`/h5/matches/{id}/events`, sans segment de mode) une timeline horodatée — Death (tueur·victime·**arme**·type·position·instant), Medal, WeaponPickup/Drop, Spawn, Round. = ce que le décodage film d'Infinite reconstruit à la main, **plus** les positions monde.

## 1. Objectif + critère de succès

**Objectif** : un modèle CANONIQUE d'événements de match qui (a) capte la richesse native Halo 5, (b) **unifie** les events Infinite existants (`highlight_events` + Timeline T0 + film-derived) sous le même canon, (c) reste LÉGER pour le cœur (surface séparée, on-demand), (d) sert de **cible canonique** au RE Infinite (le décodeur film a enfin une forme exacte à produire).

**Critère de succès** : une surface « timeline / kill-feed de match » qui affiche les events horodatés (a) pour Halo 5 = données réelles API (kill-feed + arme-par-kill + médailles), (b) pour Infinite = depuis `highlight_events`/film (dégradé là où le RE est incomplet), via **UN seul type canonique + une méthode adapter**, **sans alourdir `MatchDetail`** ni le bootstrap.

## 2. « Très gros canonique ? » — NON, et voici pourquoi (cœur du design)

La crainte est légitime (220 Ko / ~700 events par match). La réponse :

1. **Le TYPE est petit ; le VOLUME est géré par le lazy-load.** `canonical.MatchEvent` = un struct discriminé par `Type` + champs optionnels. La timeline n'est PAS un champ de `MatchDetail` : c'est une **surface séparée** chargée par un appel DÉDIÉ (`LoadMatchEvents(matchID)`), uniquement quand l'utilisateur ouvre la timeline/kill-feed d'un match. Jamais dans le bootstrap, jamais dans la liste de matchs.
2. **Filtrage par type d'event.** Le contrat `LoadMatchEvents` accepte un filtre (`types []MatchEventType`). Un kill-feed ne demande que `Death`+`Medal` ; pas besoin de transférer les 269 `WeaponPickup`. Réduit drastiquement la charge utile par usage.
3. **Pas de persistance par défaut.** Halo 5 = fetch API on-demand (live read, cache court TanStack/serveur) — zéro stockage initial. Infinite = `highlight_events` est DÉJÀ persisté (réutilisé, pas dupliqué). La persistance étendue (si besoin) est une décision Phase 4, pas un prérequis.
4. **On unifie, on ne double pas.** Infinite a déjà des events (`highlight_events`, T0, `CorrectEvents`/`CorrectImpactEvents`). Le canon GÉNÉRALISE ces sources ; il ne crée pas un 2e système parallèle.

→ Le « canonique » ajouté = **1 fichier de types + 1 méthode adapter + N adapters**. Léger. Le poids vit dans le transport on-demand, pas dans le modèle.

## 3. Architecture (couches Go respectées)

| Élément | Emplacement | Rôle |
|---|---|---|
| Types canoniques | `internal/games/canonical/events.go` | `MatchEvent`, `MatchEventTimeline`, enums `MatchEventType`/`KillKind`, `Vec3` |
| Contrat adapter | `internal/games/adapter.go` | `TitleDataAdapter.LoadMatchEvents(ctx, matchID, opts) (*MatchEventTimeline, error)` + `ErrCapabilityNotSupported` |
| Capabilities fines | `config/titles/{slug}/mappings/capabilities.toml` + `internal/games/feature.go` | `match.events.timeline`, `match.killfeed.per_kill`, `match.events.spatial` |
| Adapter Halo 5 | `internal/games/halo_5/events.go` (+ `events_dto.go`, mapper pur) | client `/h5/matches/{id}/events` → canonical |
| Adapter Infinite | mapper `highlight_events`/T0 → canonical (réutilise le pattern Timeline T0) | dégradé où le RE est incomplet |
| Orchestration | `internal/service/match_events_service.go` | on-demand, gating capability, dégradation gracieuse |
| Handler HTTP | `internal/api/handlers/match_events.go` (Huma) | `GET /players/{player_slug}/matches/{match_id}/events?types=` |
| Front | `features/match/` query + composant timeline/kill-feed | lazy, query key dédiée, i18n, color tokens |

### Forme canonique proposée

```go
type MatchEventType string // "kill" | "medal" | "weapon_pickup" | "weapon_drop" | "spawn" | "round_start" | "round_end"
type KillKind string       // "weapon" | "headshot" | "melee" | "groundpound" | "shoulderbash"

type MatchEvent struct {
    Type    MatchEventType
    TimeMs  int            // depuis TimeSinceStart (ISO8601 → ms), aligné sur le pipeline T0 existant (skip TimeMs<0)
    Killer  *PlayerIdentity // kill
    Victim  *PlayerIdentity // kill
    Weapon  *AssetReference // kill : StockId → arme canonique (résolveur asset existant)
    Kind    KillKind        // kill : drapeaux Halo5 → enum (Infinite : partiel)
    KillerLoc, VictimLoc *Vec3 // Halo 5 plein ; Infinite not_exposed
    MedalID *string         // medal
    Player  *PlayerIdentity // medal/spawn/weapon
}
type MatchEventTimeline struct {
    MatchID     string
    Events      []MatchEvent
    Limitations []CapabilityGap // ex. "per-kill weapon degraded (Infinite RE pending)"
}
```
**Discipline** : un struct discriminé par `Type` + champs nullables (convention canonical existante). Pas de sum-type. Tout champ canonique est adossé à **≥ 1 source réelle** (Halo 5 back tout ; pas de champ fantôme).

## 4. Phases (ordre = risque/effort croissant ; gate activation)

### Phase 0 — Contrat canonique + capabilities (PRE-ACTIVATION) — petit, central
- `canonical/events.go` (types + enums) ; `LoadMatchEvents` sur `TitleDataAdapter` (+ stub `ErrCapabilityNotSupported` partout) ; 3 capability keys déclarées (`supported` h5, `not_exposed` Infinite par défaut). **Inerte** (aucun adapter concret).
- Oracle : build + vet + test de types ; Halo Infinite byte-identique (additif).

### Phase 1 — Adapter Halo 5 events (PRE-ACTIVATION) — autonome
- `halo_5/events_dto.go` (shape réelle, cf. §0-quater) + `halo_5/events.go` (client `/h5/matches/{id}/events` SANS segment de mode + mapper PUR → canonical). Filtre `types`. Gamertag-keyé. StockId → `AssetReference` (résolveur asset).
- Oracle : test mapper pur sur un fixture JSON réel (capturé par la sonde) ; le kill-feed Halo 5 mappe correctement (Death → kill avec arme/positions).

### Phase 2 — Unification Infinite (PARTIEL à l'activation, complété au rythme du RE)
- Mapper `highlight_events` + events film-derived (`CorrectEvents`/`CorrectImpactEvents`, lecture T0 via `canonical/Q30`) → `MatchEventTimeline`. Réutilise le pattern Timeline T0 (skip `TimeMs<0`).
- `match.killfeed.per_kill` Infinite = **degraded** tant que le décodeur film n'attribue pas l'arme ; `match.events.spatial` = **not_exposed**. Le canon est la **cible** du RE — pas de blocage mutuel.
- ⚠️ NE PAS régresser les affichages d'events Infinite LEGACY existants (ils restent sur leurs chemins ; le canon est une surface additive).

### Phase 3 — Surface produit (PRE-ACTIVATION pour Halo 5) — service + handler + front
- `match_events_service.go` (on-demand, gating) ; `GET .../matches/{match_id}/events` (Huma, OpenAPI + regen types) ; composant front timeline/kill-feed (lazy, gaté `FeatureGate`/`RouteCapabilityGate`).
- Oracle : page kill-feed Halo 5 (données réelles) ; Infinite dégradé proprement (pas vide → « indisponible pour ce titre » si not_exposed).

### Phase 4a — DURABILITÉ : persister les events Halo 5 (à faire AVEC activation 1b)
**Rationale (ÉLEVÉ 2026-06-20, point user — corrige le cadrage « optim » initial)** : les events Halo 5 ne viennent QUE de l'API cryptum (interne, fragile — 343/MS peut la fermer à tout moment, Halo 5 est un vieux titre). Fetch-live-seul ⇒ **si l'API meurt, le kill-feed / arme-par-kill / positions Halo 5 est PERDU à jamais** — et c'est IRREMPLAÇABLE (Infinite ne produit même pas l'arme-par-kill sans le RE film). Infinite est déjà à l'abri (`highlight_events` persisté au sync). Donc persister Halo 5 = **archiver l'irremplaçable depuis une source fragile**, PAS une optimisation. → ce n'est pas « différé sine die ».
- **Capture-on-fetch append-only** : quand `LoadMatchEvents` Halo 5 fetch l'API, persister la timeline en write-through dans une table dédiée du warehouse Halo 5, **append-only** (doctrine `project_append_only_eradication_campaign` : zéro DELETE/UPDATE-indexé). Lecture ultérieure = table d'abord, API en refresh/fallback.
- **Couplé à activation 1b** (le bon moment, pas « plus tard ») : Halo 5 n'a PAS de DuckDB tant que non activé (le provisioning 1b crée le warehouse) ; le write-path se conçoit/teste contre la vraie shape live. La construire AVANT activation = à l'aveugle, sans DB ni data.
- Oracle : un match Halo 5 fetché une fois reste lisible après coupure simulée de l'API (la table sert la timeline).

### Phase 4b — VOLUME / perf (POST, vraie décision différée)
- Cache (déjà en place côté lecture TanStack/serveur). Pagination si une timeline dépasse un seuil (~700 events). Recalibration engagement Halo 5 (events présents maintenant) — `project_halo5_experimental_direction`. Décidable seulement avec du volume réel **post-activation**.

## 5. Tests (par couche — delivery-checklist)
- `canonical/` : test de types/enums purs.
- `halo_5/events` : mapper pur sur fixture JSON réel (Death/Medal/Weapon).
- Infinite mapper : `highlight_events` → canonical (réaligné sur T0).
- `service` : mock adapter, dégradation `ErrCapabilityNotSupported`.
- handler : httptest (200 Halo 5 ; 503/dégradé Infinite) + contrat OpenAPI (routes + schémas).
- front : query + composant (vitest hors sandbox).

## 6. Blockers / dépendances
- **Infinite arme-par-kill** : dépend de la maturité du décodeur film (chantier RE long, cf. `project_kill_feed_frame_decoder`, `reference_killfeed_deadstate_fields`). → `degraded` en attendant ; le canon LUI DONNE une cible exacte (gain pour le RE).
- **Halo 5** : autonome (API sert tout) mais réellement servi à l'activation (status active).
- **Aucun blocage mutuel** : Phase 0/1/3 (canon + h5 + surface) livrables sans le RE Infinite.

## 7. Done definition / sequencing
- **Gate activation** = Phase 0 + 1 + 3 vertes (canon + adapter h5 + surface) → Halo 5 peut s'activer AVEC ses events. ✅ **LIVRÉ 2026-06-20** (Phases 0→3b, audit 4/4). Phase 2 (Infinite) partielle OK à l'activation, complétée ensuite.
- **Phase 4a (durabilité — persister les events Halo 5 append-only) = à faire AVEC l'activation 1b** : source cryptum fragile + irremplaçable (cf. §4a). Phase 4b (volume/perf) = POST.
- Thought_log à chaque phase. Mémoire `project_halo5_experimental_direction` (events natifs) reste la référence direction.
- Réfs : `HANDOFF_HALO5_EXPERIMENTAL.md` §0-quater (shape réelle), `reference_match_timeline_t0`, `reference_film_chunks_structure`, `reference_halo_hud_viewmodels`.
