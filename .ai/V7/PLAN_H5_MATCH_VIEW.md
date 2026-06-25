# PLAN — Match View Halo 5 (G9 / S.8)

> STATUT (2026-06-21) : **LIVRÉ**. Part A (`LoadMatchDetail` carnage→canonical,
> commit 57ced69b7) + Part B (voie canonique service + routage repo-first +
> capability `match.detail.core=supported` + câblage viewer, commit 10ebc92c1).
> La page vue-match est fonctionnelle pour h5 (header + scoreboard + summary,
> `IsPartial` pour les onglets riches). HINF byte-identique (voie repo conservée).
> V11 (WaypointURL HINF figée) devient MOOT pour h5 : la voie canonique ne pose
> pas de WaypointURL (pas de page Waypoint match h5) ; seule la voie repo HINF la pose.

> Rend la page vue-match fonctionnelle pour h5. Aujourd'hui `match.detail.core =
> not_exposed` (capabilities.toml h5) + `MatchViewService` lit le repo DuckDB
> (vide pour h5 live-only) → page vide. Stratégie : payload CORE depuis la carnage
> via l'adapter + `IsPartial` (bandeau « sync incomplet ») pour les onglets riches.

## Données disponibles (h5, live)
- `h5Source.GetPlayerMatches(gamertag, start, count)` → liste (map/mode/playlist/startTime/isRanked par entrée).
- `h5Source.GetMatchCarnage(matchID, mode)` → roster complet (`H5CarnageResponse` : PlayerStats + TeamStats + IsTeamGame).
- `h5Source.GetMatchEvents(matchID)` → kill-feed (déjà `supported` via MatchEventsService, indépendant).
- Cœur réutilisable : `mapCarnageParticipants` (mapping_carnage.go) projette PlayerStats → stats complètes (K/D/A/headshots/shots/damage/melee/grenade/power/timeplayed/avglife/outcome/KDA net). `participantOutcome`, `winningTeamID`, `h5GameModeSegment` réutilisables.

## Cible
- `canonical.MatchDetail` (match.go:68) : MatchID, StartedAtUTC, Map/Playlist/GameVariant (`AssetReference`), IsRanked, MatchType, Participants ([]MatchParticipant), Teams ([]TeamSnapshot), Skill, Limitations.
- `domain.MatchViewResponse` (match_view.go:10) : Header + Rank + 4 onglets + `IsPartial`/`PartialReasons` (mécanisme de dégradation EXISTANT — front affiche un bandeau, pas de crash).

## PART A — h5 `DataAdapter.LoadMatchDetail` (carnage → canonical.MatchDetail)
- Stub actuel : `adapter_data.go:270` retourne `ErrCapabilityNotSupported`.
- **Point de design à résoudre** : le viewer (gamertag) n'est pas dans le ctx (seul `ctxkeys.HaloXUID` existe ; carnage gamertag-keyée, Player.Xuid null). Pour résoudre le MODE (carnage) + les refs header (map/playlist/startTime/isRanked), il faut l'entrée de la liste de matchs → `GetPlayerMatches(gamertag)`. Résoudre le gamertag du viewer (depuis le ctx/identité de requête, ou via le slug joueur, ou mode-iteration arena→warzone si le gamertag est introuvable). À investiguer + documenter.
- Construire `canonical.MatchDetail` : refs depuis l'entrée de liste (réutiliser le mapping summary existant `mapping.go`), Participants/Teams/outcome depuis la carnage (nouveau mapper carnage→canonical, parallèle à `mapCarnageParticipants`). Skill best-effort (CSR arena pré/post = Phase ultérieure → nil). `Limitations` = gaps connus.
- Best-effort + dégradation : token absent / 404 / match introuvable → `ErrCapabilityNotSupported` (le service retombe sur le repo, comportement actuel).
- Test adapter (fake source : liste + carnage → MatchDetail attendu).

## PART B — `MatchViewService` voie canonical + routage + capability
- `buildMatchViewFromCanonical(ctx, detail) domain.MatchViewResponse` : Header (map/mode/outcome/score/duration/isRanked + MapImageURL via `s.assetURL`), Rank (depuis detail.Skill, nil → vide), TeamTab/scoreboard (depuis Participants+Teams), SummaryTab KPIs (participant `s.xuid`), `IsPartial=true` + `PartialReasons` (combat narrative / citations / media indisponibles live). CombatTab/events restent servis par l'endpoint events (h5 supported).
- `GetMatchView` : si `s.dataAdapter != nil` et `LoadMatchDetail` réussit (non-nil, != ErrCapabilityNotSupported) → `buildMatchViewFromCanonical` ; sinon voie repo actuelle (HINF byte-identique).
- `capabilities.toml` h5 : `match.detail.core` → `supported`.
- Vérifier que le front (`MatchViewPage.tsx`) dégrade bien sur `is_partial` + onglets vides (ne pas crasher).

## Garde-fous
- HINF byte-identique : la voie repo reste le défaut ; la voie canonical n'est prise que si l'adapter sert `match.detail.core`.
- Pas de méthode par jeu côté service : le routage est piloté par la présence/capability de l'adapter, pas par le slug.
