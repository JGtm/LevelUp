package service

import (
	"context"
	"errors"
	"hash/fnv"
	"log/slog"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/canonical"
)

// computeTargetCombatProfileLocal retourne les N derniers matchs PvP de la cible
// présents en base locale (shared.match_participants — surtout les matchs communs).
// Alimente le toggle "local" de la section. nil si erreur / aucun match.
func (s *ExplorerService) computeTargetCombatProfileLocal(ctx context.Context, targetXUID string) []domain.ExplorerTargetRecentMatch {
	rows, err := s.loadTargetRecentRows(ctx, targetXUID, explorerCombatProfileLimit)
	if err != nil {
		slog.WarnContext(ctx, "explorer_target_combat_profile_local_failed", "xuid", targetXUID, "err", err)
		return nil
	}
	slog.DebugContext(ctx, "explorer_target_combat_profile", "xuid", targetXUID, "matches", len(rows), "source", "local")
	return rows
}

// loadTargetRecentRows centralise la résolution repo/adapter du profil de combat
// récent (HIGH-B). Adapter-first via LoadTargetRecentMatches + reconstitution
// byte-identique ; fallback repo.GetTargetRecentMatches sur capability absente.
func (s *ExplorerService) loadTargetRecentRows(ctx context.Context, xuid string, limit int) ([]domain.ExplorerTargetRecentMatch, error) {
	if s.dataAdapter != nil {
		canonicalRows, err := s.dataAdapter.LoadTargetRecentMatches(ctx, xuid, limit)
		if err == nil {
			return recentMatchesFromCanonical(canonicalRows), nil
		}
		if !errors.Is(err, games.ErrCapabilityNotSupported) {
			return nil, err
		}
	}
	return s.repo.GetTargetRecentMatches(ctx, xuid, limit)
}

// loadParticipantStats centralise la résolution repo/adapter de l'agrégat sample
// (HIGH-B). Adapter-first via LoadParticipantStats + reconstitution byte-identique ;
// fallback repo.GetParticipantStatsForMatches sur capability absente.
func (s *ExplorerService) loadParticipantStats(ctx context.Context, xuid string, matchIDs []string) (*domain.ParticipantStatsAggregate, error) {
	if s.dataAdapter != nil {
		c, err := s.dataAdapter.LoadParticipantStats(ctx, xuid, matchIDs)
		if err == nil {
			return participantStatsFromCanonical(c), nil
		}
		if !errors.Is(err, games.ErrCapabilityNotSupported) {
			return nil, err
		}
	}
	return s.repo.GetParticipantStatsForMatches(ctx, xuid, matchIDs)
}

// participantStatsFromCanonical projette *canonical.PlayerMatchSetStats →
// *domain.ParticipantStatsAggregate (nil → nil).
func participantStatsFromCanonical(c *canonical.PlayerMatchSetStats) *domain.ParticipantStatsAggregate {
	if c == nil {
		return nil
	}
	return &domain.ParticipantStatsAggregate{
		Kills:             c.Kills,
		Deaths:            c.Deaths,
		Assists:           c.Assists,
		Wins:              c.Wins,
		Losses:            c.Losses,
		Draws:             c.Draws,
		ShotsFired:        c.ShotsFired,
		ShotsHit:          c.ShotsHit,
		DamageDealt:       c.DamageDealt,
		DamageTaken:       c.DamageTaken,
		HeadshotKills:     c.HeadshotKills,
		MeleeKills:        c.MeleeKills,
		PowerWeaponKills:  c.PowerWeaponKills,
		GrenadeKills:      c.GrenadeKills,
		TimePlayedSeconds: c.TimePlayedSeconds,
		PersonalScore:     c.PersonalScore,
	}
}

// recentMatchesFromCanonical projette []canonical.RecentMatchRow →
// []domain.ExplorerTargetRecentMatch (copie profonde de Rank). ModePairAssetID reste
// vide (transient live, hors canonique). Retourne nil pour une entrée vide.
func recentMatchesFromCanonical(cs []canonical.RecentMatchRow) []domain.ExplorerTargetRecentMatch {
	if len(cs) == 0 {
		return nil
	}
	out := make([]domain.ExplorerTargetRecentMatch, 0, len(cs))
	for _, c := range cs {
		r := domain.ExplorerTargetRecentMatch{
			MatchID:         c.MatchID,
			StartTime:       c.StartTime,
			MapUI:           c.MapUI,
			ModeUI:          c.ModeUI,
			Outcome:         c.Outcome,
			Kills:           c.Kills,
			Deaths:          c.Deaths,
			Assists:         c.Assists,
			KDA:             c.KDA,
			Score:           c.Score,
			DamageDealt:     c.DamageDealt,
			DamageTaken:     c.DamageTaken,
			MaxKillingSpree: c.MaxKillingSpree,
			PerfectKills:    c.PerfectKills,
		}
		if c.Rank != nil {
			v := *c.Rank
			r.Rank = &v
		}
		out = append(out, r)
	}
	return out
}

// computeTargetCombatProfileLive fetche en LIVE les ~20 derniers matchs PvP de la
// cible (RecentMatchesProvider, borné par liveCtx, cache 20 min). C'est la source
// AFFICHÉE PAR DÉFAUT. nil si pas de provider / pas d'auth / pas de xuid / erreur.
func (s *ExplorerService) computeTargetCombatProfileLive(liveCtx context.Context, targetXUID string, hasAuth bool) []domain.ExplorerTargetRecentMatch {
	if s.deps.RecentMatches == nil || !hasAuth || targetXUID == "" {
		return nil
	}
	live, err := s.deps.RecentMatches.FetchRecentMatches(liveCtx, targetXUID, explorerCombatProfileLimit)
	if err != nil {
		slog.WarnContext(liveCtx, "explorer_target_combat_profile_live_failed", "xuid", targetXUID, "err", err)
		return nil
	}
	// Traduit les sous-modes EN normalisés en FR (mode_name_tr) — MÊME résolution
	// que la source locale, pour un donut "Répartition des modes" homogène.
	s.repo.TranslateModeUIsFR(liveCtx, live)
	slog.DebugContext(liveCtx, "explorer_target_combat_profile", "xuid", targetXUID, "matches", len(live), "source", "live")
	return live
}

// computeMatchesPerSeason agrège les matchs du target par saison (calcul local
// DuckDB, indépendant des tokens) : start_times depuis shared.match_participants
// rangés dans les plages de saison. nil si pas de saisons câblées / pas de matchs.
func (s *ExplorerService) computeMatchesPerSeason(ctx context.Context, targetXUID string) []domain.SeasonMatchCount {
	if len(s.deps.Seasons) == 0 {
		return nil
	}
	starts, err := s.repo.GetMatchStartTimesForXUID(ctx, targetXUID)
	if err != nil {
		slog.WarnContext(ctx, "explorer_target_matches_per_season_failed", "xuid", targetXUID, "err", err)
		return nil
	}
	return buildMatchesPerSeason(starts, s.deps.Seasons)
}

// fetchTargetIdentityRaw résout l'identité Spartan BRUTE du target : DB locale
// d'abord (si joueur suivi), sinon live pour un xuid arbitraire (career rank/
// emblem/backdrop). Applique le fallback bannière → backdrop. Retourne nil si
// rien. La conversion vers le DTO snake_case (BuildSpartanIdentity) est faite
// par l'appelant buildTargetProfile.
func (s *ExplorerService) fetchTargetIdentityRaw(ctx context.Context, targetXUID, targetGamertag string, hasAuth bool) *domain.HomeSpartanIdentityRow {
	if s.deps.LocalIdentity != nil {
		if id := s.deps.LocalIdentity.LocalSpartanIdentity(ctx, targetGamertag); id != nil {
			// Joueur suivi : on garde sa vraie bannière (preferPool=false).
			s.applyBannerFallbacks(ctx, id, targetXUID, false)
			return id
		}
	}
	if hasAuth && s.deps.LiveIdentity != nil && targetXUID != "" {
		id, err := s.deps.LiveIdentity.FetchLiveIdentity(ctx, targetXUID)
		if err != nil {
			slog.WarnContext(ctx, "explorer_target_identity_live_failed", "xuid", targetXUID, "err", err)
			// On ne retourne pas : on tente quand même la bannière de repli ci-dessous.
		} else if id != nil {
			// Cible non-locale : nameplate live souvent 404 → préférer le pool (preferPool=true).
			s.applyBannerFallbacks(ctx, id, targetXUID, true)
			return id
		}
	}
	// Aucune identité exploitable (cible non suivie + pas d'auth / live indisponible
	// ou en erreur). On synthétise une identité minimale ne portant qu'une nameplate
	// de repli déterministe du pool local, pour qu'une cible non-locale ne soit
	// jamais affichée sans bannière. nil si pas de xuid / pool vide → l'appelant
	// retombe sur le placeholder "identité indisponible".
	if id := s.fallbackBannerOnlyIdentity(ctx, targetXUID); id != nil {
		return id
	}
	slog.DebugContext(ctx, "explorer_target_identity_unavailable", "gamertag", targetGamertag)
	return nil
}

// fallbackBannerOnlyIdentity construit une identité minimale ne portant qu'une
// bannière du pool local (déterministe par xuid, cf. pickDeterministicBanner).
// Retourne nil si le pool n'est pas câblé, si le xuid est vide, ou si le pool est
// vide (aucun joueur suivi avec bannière).
func (s *ExplorerService) fallbackBannerOnlyIdentity(ctx context.Context, targetXUID string) *domain.HomeSpartanIdentityRow {
	if s.deps.LocalBannerPool == nil || targetXUID == "" {
		return nil
	}
	b := pickDeterministicBanner(targetXUID, s.deps.LocalBannerPool(ctx))
	if b == "" {
		return nil
	}
	slog.DebugContext(ctx, "explorer_target_banner_pool_only_fallback", "xuid", targetXUID)
	return &domain.HomeSpartanIdentityRow{BannerImageURL: &b}
}

// applyBannerFallbacks résout la bannière de la cible. preferPool=true (cible
// NON-locale, identité live) : la nameplate live pointe quasi-systématiquement
// vers un asset Waypoint non résolvable par notre CDN (404) → on PRÉFÈRE d'emblée
// une nameplate du pool local (déterministe par xuid, issue des joueurs suivis,
// assets en cache), tout en gardant emblem/rang live. preferPool=false (cible
// suivie, identité locale) : on garde sa vraie bannière, fallback backdrop, puis
// pool seulement si rien. Mutation en place, best-effort.
func (s *ExplorerService) applyBannerFallbacks(ctx context.Context, id *domain.HomeSpartanIdentityRow, targetXUID string, preferPool bool) {
	if id == nil {
		return
	}
	if preferPool && s.deps.LocalBannerPool != nil && targetXUID != "" {
		if b := pickDeterministicBanner(targetXUID, s.deps.LocalBannerPool(ctx)); b != "" {
			id.BannerImageURL = &b
			slog.DebugContext(ctx, "explorer_target_banner_pool_preferred", "xuid", targetXUID)
			return
		}
	}
	applyBannerFallback(id)
	if hasBanner(id) || s.deps.LocalBannerPool == nil || targetXUID == "" {
		return
	}
	if b := pickDeterministicBanner(targetXUID, s.deps.LocalBannerPool(ctx)); b != "" {
		id.BannerImageURL = &b
		slog.DebugContext(ctx, "explorer_target_banner_pool_fallback", "xuid", targetXUID)
	}
}

// hasBanner indique si l'identité porte déjà une bannière non vide.
func hasBanner(id *domain.HomeSpartanIdentityRow) bool {
	return id.BannerImageURL != nil && *id.BannerImageURL != ""
}

// pickDeterministicBanner choisit une bannière du pool de façon stable par xuid
// (hash FNV-32a → index) — un même joueur retombe toujours sur la même nameplate
// (pas de random non-déterministe). "" si le pool est vide.
func pickDeterministicBanner(xuid string, pool []string) string {
	if len(pool) == 0 {
		return ""
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(xuid))
	return pool[int(h.Sum32())%len(pool)]
}

// applyBannerFallback : si la bannière (nameplate background) est absente — cas
// fréquent pour une cible non locale, l'API officielle n'expose pas de
// nameplate dédié — on retombe sur le backdrop du joueur (présent dans
// l'appearance). Mutation en place, best-effort.
func applyBannerFallback(id *domain.HomeSpartanIdentityRow) {
	if id == nil {
		return
	}
	if (id.BannerImageURL == nil || *id.BannerImageURL == "") &&
		id.BackdropImageURL != nil && *id.BackdropImageURL != "" {
		id.BannerImageURL = id.BackdropImageURL
	}
}

// fetchTargetServiceRecord fetch le service record live (stats + temps de jeu +
// médailles lifetime) et en dérive les stats carrière + le top médailles + la
// liste des saisons jouées (Subqueries.SeasonIds, pour le breakdown par saison).
// Skip sans auth ; (nil, nil, nil) en cas d'erreur.
func (s *ExplorerService) fetchTargetServiceRecord(ctx context.Context, targetGamertag string, hasAuth bool) (*domain.NormalizedPlayerStats, []domain.MedalDigestItem, []string, []string) {
	if s.deps.RemoteStats == nil {
		return nil, nil, nil, nil
	}
	if !hasAuth {
		slog.DebugContext(ctx, "explorer_target_career_skipped",
			"gamertag", targetGamertag, "reason", "no_auth_tokens")
		return nil, nil, nil, nil
	}
	rec, err := s.deps.RemoteStats.FetchServiceRecord(ctx, targetGamertag, s.deps.TitleSlug)
	if err != nil {
		slog.WarnContext(ctx, "explorer_target_career_failed", "gamertag", targetGamertag, "err", err)
		return nil, nil, nil, nil
	}
	if rec == nil {
		return nil, nil, nil, nil
	}
	stats := rec.Stats
	medals := buildTargetTopMedals(ctx, s.deps.MedalDefs, rec.Medals, s.deps.TitleSlug, ctxkeys.Locale(ctx))
	return &stats, medals, rec.SeasonIDs, rec.PlaylistAssetIDs
}

// fetchTargetCSR fetch les classements CSR de la saison courante de la cible
// (endpoint skill public — tout xuid). Skip sans auth / sans provider / sans
// saison. nil en cas d'erreur (logguée).
func (s *ExplorerService) fetchTargetCSR(ctx context.Context, targetXUID string, hasAuth bool) []domain.CareerPlaylistCSR {
	if s.deps.CSR == nil || !hasAuth || targetXUID == "" || s.deps.CurrentSeasonID == "" {
		return nil
	}
	csrs, err := s.deps.CSR.SeasonCSRs(ctx, targetXUID, s.deps.CurrentSeasonID)
	if err != nil {
		slog.WarnContext(ctx, "explorer_target_csr_failed", "xuid", targetXUID, "err", err)
		return nil
	}
	return csrs
}

// computeTargetSampleStats agrège les stats du target sur les matchs communs
// (calcul local DuckDB, indépendant des tokens). nil si aucun match commun.
func (s *ExplorerService) computeTargetSampleStats(ctx context.Context, targetXUID string, rawMatches []domain.CommonMatchRaw) *domain.ExplorerTargetSampleStats {
	matchIDs := extractCommonMatchIDs(rawMatches)
	if len(matchIDs) == 0 {
		return nil
	}
	agg, err := s.loadParticipantStats(ctx, targetXUID, matchIDs)
	if err != nil {
		slog.WarnContext(ctx, "explorer_target_sample_stats_failed", "xuid", targetXUID, "err", err)
		return nil
	}
	medals, mErr := s.repo.GetMedalCountsForMatches(ctx, targetXUID, matchIDs)
	if mErr != nil {
		slog.WarnContext(ctx, "explorer_target_medals_failed", "xuid", targetXUID, "err", mErr)
		// medals est nil → BuildSampleStats l'ignorera, ce n'est pas bloquant.
	}
	slug := ctxkeys.TitleSlug(ctx)
	sample := analysis.BuildSampleStats(agg, medals, len(matchIDs), games.EffectiveHpToKill(slug))
	if sample != nil {
		// Top 3 armes (par kills) sur les matchs communs — best-effort.
		weapons, wErr := s.repo.GetTopWeaponsForMatches(ctx, targetXUID, matchIDs, 3)
		if wErr != nil {
			slog.WarnContext(ctx, "explorer_target_top_weapons_failed", "xuid", targetXUID, "err", wErr)
		}
		sample.TopWeapons = weapons
	}
	return sample
}

// extractCommonMatchIDs extrait la liste des match_id d'un slice de common matches.
