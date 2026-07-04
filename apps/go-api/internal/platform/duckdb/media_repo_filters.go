// Package duckdb - media_repo_filters.go : LoadMediaFilterOptions +
// translatePlaylistFilterOptions + LoadMatchCandidatesForMedia +
// loadMatchLobbies + normalizeModeLabel. Decoupe de media_repo.go
// (god-file split, refactor 2026-05-27).
package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
)

func (r *MediaRepo) LoadMediaFilterOptions(ctx context.Context, filters domain.MediaFilters) (domain.MediaFilterOptions, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	mapEnriched, err := r.runMediaPipeline(ctx, filters, mediaWhereConfig{
		includeMapFilter:      false,
		includeModeFilter:     true,
		includePlaylistFilter: true,
	}, 0, 0)
	if err != nil {
		return domain.MediaFilterOptions{}, fmt.Errorf("LoadMediaFilterOptions maps: %w", err)
	}
	maps := r.translateMapFilterOptions(ctx, extractMapPairs(mapEnriched))

	modeEnriched, err := r.runMediaPipeline(ctx, filters, mediaWhereConfig{
		includeMapFilter:      true,
		includeModeFilter:     false,
		includePlaylistFilter: true,
	}, 0, 0)
	if err != nil {
		return domain.MediaFilterOptions{}, fmt.Errorf("LoadMediaFilterOptions modes: %w", err)
	}
	modes := r.translateModeFilterOptions(ctx, extractModePairs(modeEnriched))

	playlistEnriched, err := r.runMediaPipeline(ctx, filters, mediaWhereConfig{
		includeMapFilter:      true,
		includeModeFilter:     true,
		includePlaylistFilter: false,
	}, 0, 0)
	if err != nil {
		return domain.MediaFilterOptions{}, fmt.Errorf("LoadMediaFilterOptions playlists: %w", err)
	}
	playlists := r.translatePlaylistFilterOptions(ctx, extractPlaylistPairs(playlistEnriched))

	return domain.MediaFilterOptions{Playlists: playlists, Maps: maps, Modes: modes}, nil
}

// ListMediaAuthors retourne les player_slug distincts ayant au moins un média,
// avec leur compte. Source UNIQUE = shared_social.media_files — exactement la
// table que lit la galerie (cf. loadMediaCandidates) — pour garantir que la liste
// d'auteurs proposée au filtre soit cohérente avec ce que la galerie peut afficher.
//
// Remplace l'ancien scan filesystem (countMediaInDir) du handler GetMediaAuthors,
// qui ratait les auteurs dont les fichiers n'étaient pas présents sur le disque
// local (multi-user) ou rangés en sous-dossiers.
//
// Gamertag et is_self ne sont PAS résolus ici : le caller (service/handler) les
// enrichit depuis db_profiles.json et le slug courant. On retourne PlayerSlug +
// MediaCount uniquement.
//
// Schéma legacy (SharedSocial nil, pas de colonne player_slug) : on ne peut pas
// distinguer les auteurs → on retourne uniquement le joueur courant, équivalent
// au comportement antérieur pour les DB non migrées.
func (r *MediaRepo) ListMediaAuthors(ctx context.Context) ([]domain.MediaAuthor, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if r.pdb.SharedSocial == nil {
		slug := r.CurrentPlayerSlug()
		if slug == "" {
			return []domain.MediaAuthor{}, nil
		}
		return []domain.MediaAuthor{{PlayerSlug: slug, MediaCount: 0}}, nil
	}

	// Pas de filtre status : la galerie en schéma shared_social ne filtre pas non
	// plus (cf. mediaQueryConfig.baseWhereClause, branche default) — la plupart des
	// lignes ont status NULL. On reste cohérent avec ce que la galerie affiche.
	rows, err := r.socialDB().QueryRecovered(ctx, `
		SELECT player_slug, COUNT(*) AS n
		FROM media_files
		WHERE player_slug IS NOT NULL AND player_slug <> ''
		GROUP BY player_slug
		ORDER BY n DESC, player_slug ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("ListMediaAuthors: %w", err)
	}
	defer rows.Close()

	authors := make([]domain.MediaAuthor, 0, 8)
	for rows.Next() {
		var slug string
		var n int
		if err := rows.Scan(&slug, &n); err != nil {
			return nil, fmt.Errorf("ListMediaAuthors: scan: %w", err)
		}
		authors = append(authors, domain.MediaAuthor{PlayerSlug: slug, MediaCount: n})
	}
	return authors, rows.Err()
}

// translatePlaylistFilterOptions enrichit les libellÃ©s de playlists en FR via
// asset_translations (asset_type='playlist') + dÃ©dup par playlist_id. Value =
// playlist_id (stable) ; sinon fallback label brut.
func (r *MediaRepo) translatePlaylistFilterOptions(ctx context.Context, pairs []mediaFilterOptionPair) []domain.LabelValue {
	if len(pairs) == 0 {
		return []domain.LabelValue{}
	}
	idsSet := make(map[string]struct{})
	for _, p := range pairs {
		if p.id != "" {
			idsSet[p.id] = struct{}{}
		}
	}
	ids := make([]string, 0, len(idsSet))
	for id := range idsSet {
		ids = append(ids, id)
	}
	translations := r.loadAssetTranslationNames(ctx, "playlist", ids)

	seenIDs := make(map[string]bool)
	seenLabels := make(map[string]bool)
	options := make([]domain.LabelValue, 0, len(pairs))
	for _, p := range pairs {
		labelFR := translations[p.id]
		if labelFR == "" {
			labelFR = p.label
		}
		value := p.id
		if value == "" {
			value = p.label
			if seenLabels[value] {
				continue
			}
			seenLabels[value] = true
		} else {
			if seenIDs[value] {
				continue
			}
			seenIDs[value] = true
		}
		options = append(options, domain.LabelValue{Label: labelFR, Value: value})
	}
	sort.Slice(options, func(i, j int) bool { return options[i].Label < options[j].Label })
	return options
}

// CurrentPlayerSlug retourne le slug (== gamertag) du joueur dont on lit la galerie.
func (r *MediaRepo) CurrentPlayerSlug() string {
	if r.pdb == nil {
		return ""
	}
	return r.pdb.Gamertag
}

// LoadMatchCandidatesForMedia retourne les matchs du joueur courant dans la
// fenÃªtre temporelle [capture_start - window, capture_start + window].
// Inclut les KPIs du joueur pour aider Ã  reconnaÃ®tre le bon match.
// Si capture_start_utc est nul â†’ fallback mtime, sinon liste vide.
//
// cross-DB match scan (shared + player KPIs) → assemblage candidates. La complexité
// reflète les nombreux fallbacks (capture_start_utc → mtime, NULL playlists, etc.).
// Splitter éclaterait la cohésion du flow.
//
//nolint:funlen,gocyclo // Pipeline ENTIER : lookup media → fenêtre temporelle →
func (r *MediaRepo) LoadMatchCandidatesForMedia(ctx context.Context, filePath string, windowMinutes int) (domain.MediaMatchCandidatesResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if windowMinutes <= 0 {
		windowMinutes = 15
	}

	if r.pdb.SharedSocial == nil {
		return domain.MediaMatchCandidatesResponse{}, nil
	}

	// Lire capture_start_utc + association actuelle du mÃ©dia.
	// Match flexible : soit file_path exact (DB absolute), soit file_name
	// (basename â€” pour quand le frontend envoie l'URL transformÃ©e et qu'on
	// reÃ§oit “.../foo.mp4” au lieu du chemin DB original).
	basename := filepath.Base(filePath)
	var captureUTC sql.NullTime
	var currentMatchID sql.NullString
	err := r.socialDB().QueryRow(ctx, `
		SELECT
			COALESCE(mf.capture_start_utc, mf.capture_end_utc, mf.mtime) AS cap,
			mma.match_id
		FROM media_files mf
		LEFT JOIN media_match_associations_latest mma ON mma.media_file_id = mf.id
		WHERE mf.file_path = ? OR mf.file_name = ?
		LIMIT 1
	`, filePath, basename).Scan(&captureUTC, &currentMatchID)
	if err != nil {
		return domain.MediaMatchCandidatesResponse{}, fmt.Errorf("LoadMatchCandidatesForMedia: lookup capture: %w", err)
	}

	resp := domain.MediaMatchCandidatesResponse{
		FilePath:      filePath,
		WindowMinutes: windowMinutes,
		Candidates:    []domain.MediaMatchCandidate{},
	}
	if !captureUTC.Valid {
		return resp, nil
	}
	cap := captureUTC.Time
	resp.CaptureUTC = &cap

	// Charger les matchs du joueur dans la fenÃªtre.
	// start_time_utc est TIMESTAMPTZ UTC garanti (migration add_start_time_utc_to_match_registry).
	// Fallback sur start_time AT TIME ZONE 'UTC' pour les matchs synchro aprÃ¨s le fix DuckDB
	// (first_sync_at >= 2026-03-01 â†’ start_time dÃ©jÃ  UTC) qui n'auraient pas encore start_time_utc.
	// query shared-only via SharedReader (root-level
	// naming). start_time_utc est TIMESTAMPTZ UTC garanti.
	sharedDB, releaseShared, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return resp, fmt.Errorf("LoadMatchCandidatesForMedia: shared reader: %w", err)
	}
	defer releaseShared()

	rows, err := sharedDB.QueryContext(ctx, fmt.Sprintf(`
		SELECT
			r.match_id,
			`+StartTimeCanonicalSQL("r")+` AS start_utc,
			COALESCE(r.end_time_utc,   r.end_time   AT TIME ZONE 'UTC') AS end_utc,
			COALESCE(r.map_name_fr, r.map_name) AS map_name,
			COALESCE(r.map_id, '') AS map_id,
			COALESCE(r.pair_name_fr, r.pair_name) AS pair_name,
			COALESCE(r.playlist_name_fr, r.playlist_name) AS playlist_name,
			COALESCE(r.playlist_id, '') AS playlist_id,
			mp.outcome,
			mp.team_id,
			r.team_0_score,
			r.team_1_score,
			ABS(DATEDIFF('second', ?, `+StartTimeCanonicalSQL("r")+`)) AS delta_s
		FROM match_registry r
		JOIN match_participants mp
			ON mp.match_id = r.match_id AND mp.xuid = ?
		WHERE `+StartTimeCanonicalSQL("r")+`
		        BETWEEN (? - INTERVAL '%d minutes')
		            AND (? + INTERVAL '%d minutes')
		ORDER BY ABS(DATEDIFF('second', ?, `+StartTimeCanonicalSQL("r")+`)) ASC
		LIMIT 50
	`, windowMinutes, windowMinutes), cap, r.pdb.XUID, cap, cap, cap)
	if err != nil {
		return resp, fmt.Errorf("LoadMatchCandidatesForMedia: query: %w", err)
	}
	defer rows.Close()

	matchIDs := []string{}
	mapIDByMatch := map[string]string{}
	mapIDSet := map[string]struct{}{}
	modeEnSet := map[string]struct{}{}
	playlistIDByMatch := map[string]string{}
	playlistIDSet := map[string]struct{}{}
	for rows.Next() {
		var c domain.MediaMatchCandidate
		var mapName, mapID, pairName, playlistName, playlistID sql.NullString
		var outcome sql.NullInt64
		var deltaS sql.NullInt64
		var startT, endT sql.NullTime
		var teamID, team0Score, team1Score sql.NullInt64
		if err := rows.Scan(&c.MatchID, &startT, &endT, &mapName, &mapID, &pairName, &playlistName,
			&playlistID, &outcome, &teamID, &team0Score, &team1Score, &deltaS); err != nil {
			continue
		}
		// Scores POV joueur (team_id 0 → own=team_0_score, sinon swap).
		// Nil si l'un des deux team_X_score est NULL côté DB (FFA, modes objectif sans score).
		if teamID.Valid && team0Score.Valid && team1Score.Valid {
			own := int(team0Score.Int64)
			enemy := int(team1Score.Int64)
			if teamID.Int64 != 0 {
				own, enemy = enemy, own
			}
			c.OwnScore = &own
			c.EnemyScore = &enemy
		}
		if mapID.Valid && mapID.String != "" {
			mapIDByMatch[c.MatchID] = mapID.String
			mapIDSet[mapID.String] = struct{}{}
		}
		if playlistID.Valid && playlistID.String != "" {
			playlistIDByMatch[c.MatchID] = playlistID.String
			playlistIDSet[playlistID.String] = struct{}{}
		}
		if startT.Valid {
			t := startT.Time
			c.StartTime = &t
		}
		if endT.Valid {
			t := endT.Time
			c.EndTime = &t
		}
		if mapName.Valid {
			s := strings.TrimSpace(mapName.String)
			if s != "" {
				c.MapName = &s
			}
		}
		if pairName.Valid {
			// Sous-mode EN canonique pour le picker (Slayer/CTF/KOTH/etc.) :
			// l'utilisateur a besoin du DÃ‰TAIL du mode pour distinguer entre 4
			// matchs candidats â€” pas de la catÃ©gorie parente. cf. mode_label.go
			// (NormalizeModeLabel) vs mode_category.go (InferModeCategoryFromPairName).
			if en := analysis.NormalizeModeLabel(pairName.String); en != "" {
				c.ModeName = &en
				modeEnSet[en] = struct{}{}
			}
		}
		if playlistName.Valid {
			s := strings.TrimSpace(playlistName.String)
			if s != "" {
				c.PlaylistName = &s
			}
		}
		if outcome.Valid {
			n := int(outcome.Int64)
			c.Outcome = &n
		}
		if deltaS.Valid {
			n := int(deltaS.Int64)
			c.DeltaSeconds = &n
		}
		c.IsCurrent = currentMatchID.Valid && currentMatchID.String == c.MatchID
		resp.Candidates = append(resp.Candidates, c)
		matchIDs = append(matchIDs, c.MatchID)
	}

	// Traduction FR des modes (ex: "Slayer" â†’ "Assassin") via mode_name_tr.
	// Si pair_name_fr Ã©tait dÃ©jÃ  rempli en DB, on le prÃ©serve quand mÃªme
	// puisqu'on substitue uniquement si une traduction existe.
	if len(modeEnSet) > 0 {
		enList := make([]string, 0, len(modeEnSet))
		for en := range modeEnSet {
			enList = append(enList, en)
		}
		translations := r.loadModeNameTranslations(ctx, enList)
		for i := range resp.Candidates {
			if resp.Candidates[i].ModeName == nil {
				continue
			}
			// ModeName est dÃ©jÃ  le sous-mode EN canonique (cf. boucle ci-dessus) â†’
			// lookup direct, pas besoin de re-normaliser.
			if fr, ok := translations[*resp.Candidates[i].ModeName]; ok && fr != "" {
				resp.Candidates[i].ModeName = &fr
			}
		}
	}

	// Résolution map (nom + image) via maps_catalog (name_canonical TOUJOURS peuplé)
	// + asset_translations FR — même source fiable que la page match / ListMapsByTitle.
	// Corrige (a) le nom de map affiché en GUID brut quand match_registry n'est pas
	// enrichi, (b) l'absence de miniature : map_images_registry souvent vide ET le nom
	// EN d'asset_translations absent → on bascule sur le name_canonical du catalogue
	// pour le fallback AssetURLAdapter (câblé via WithAssetURL, registry_media.go).
	if len(mapIDSet) > 0 {
		ids := make([]string, 0, len(mapIDSet))
		for id := range mapIDSet {
			ids = append(ids, id)
		}
		urls := r.loadMapImageURLsByID(ctx, ids)
		catNames := r.loadMapCatalogNames(ctx, ids)
		missing := 0
		for i := range resp.Candidates {
			mid := mapIDByMatch[resp.Candidates[i].MatchID]
			if mid == "" {
				continue
			}
			cat := catNames[mid]
			// Nom affiché : FR préféré, sinon name_canonical EN ; jamais l'UUID brut.
			switch {
			case cat.fr != "":
				fr := cat.fr
				resp.Candidates[i].MapName = &fr
			case cat.en != "":
				en := cat.en
				resp.Candidates[i].MapName = &en
			case resp.Candidates[i].MapName != nil && looksLikeAssetID(*resp.Candidates[i].MapName):
				resp.Candidates[i].MapName = nil
			}
			// Image : map_images_registry d'abord, sinon adapter sur le name_canonical EN.
			if u, ok := urls[mid]; ok && u != "" {
				localCopy := u
				resp.Candidates[i].MapImageURL = &localCopy
				continue
			}
			if r.assetURL != nil && cat.en != "" {
				if u := r.assetURL.MapImageURL(cat.en); u != "" {
					localCopy := u
					resp.Candidates[i].MapImageURL = &localCopy
					continue
				}
			}
			missing++
		}
		if missing > 0 {
			slog.WarnContext(ctx, "media_picker: miniature map non résolue pour certains candidats",
				"missing", missing, "total", len(resp.Candidates))
		}
	}

	// Traduction FR des playlists via asset_translations (asset_type='playlist').
	if len(playlistIDSet) > 0 {
		ids := make([]string, 0, len(playlistIDSet))
		for id := range playlistIDSet {
			ids = append(ids, id)
		}
		playlistTr := r.loadAssetTranslationNames(ctx, "playlist", ids)
		for i := range resp.Candidates {
			pid := playlistIDByMatch[resp.Candidates[i].MatchID]
			if pid == "" {
				continue
			}
			if fr, ok := playlistTr[pid]; ok && fr != "" {
				resp.Candidates[i].PlaylistName = &fr
			}
		}
	}

	// 2e query batch : lobby (max 12 joueurs par match â€” assez pour 4v4 + spectateurs)
	if len(matchIDs) > 0 {
		lobbies := r.loadMatchLobbies(ctx, matchIDs)
		for i := range resp.Candidates {
			resp.Candidates[i].Lobby = lobbies[resp.Candidates[i].MatchID]
		}
	}

	return resp, nil
}

// loadMatchLobbies retourne le lobby (max 12 joueurs/match) pour un set de match_ids.
// Marque is_self pour le joueur courant (xuid match).
func (r *MediaRepo) loadMatchLobbies(ctx context.Context, matchIDs []string) map[string][]domain.MediaMatchLobbyEntry {
	if len(matchIDs) == 0 {
		return nil
	}
	placeholders := make([]string, len(matchIDs))
	args := make([]any, 0, len(matchIDs)+1)
	args = append(args, r.pdb.XUID)
	for i, id := range matchIDs {
		placeholders[i] = "?"
		args = append(args, id)
	}
	// Résolveur canonique : v_gamertag_lookup gère bots + cascade
	// xuid_aliases / match_participants. Fallback masqué "Joueur ####" (jamais de
	// xuid brut, miroir analysis.MaskedXuidLabelSQL) pour les orphelins + garantit
	// gamertag NON NULL pour le scan. Le JOIN xuid_aliases est devenu redondant
	// (la vue intègre déjà xuid_aliases) → supprimé.
	// is_bot : aligné sur Q12 (queries_match.go) — pour badge "Bot" dans le picker.
	// query shared-only via SharedReader (root-level naming).
	q := `
		SELECT mp.match_id,
			COALESCE(vg.gamertag, ('Joueur ' || RIGHT(mp.xuid, 4))) AS gamertag,
			mp.team_id,
			(mp.xuid = ?) AS is_self,
			(` + analysis.SQLIsBotCol("mp.xuid") + `) AS is_bot
		FROM match_participants mp
		LEFT JOIN v_gamertag_lookup vg ON vg.xuid = mp.xuid
		WHERE mp.match_id IN (` + joinStrings(placeholders) + `)
		ORDER BY mp.match_id, mp.team_id, mp.gamertag
	`
	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil
	}
	defer release()

	rows, err := db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()

	out := make(map[string][]domain.MediaMatchLobbyEntry, len(matchIDs))
	for rows.Next() {
		var matchID string
		var gamertag sql.NullString
		var teamID sql.NullInt64
		var isSelf, isBot bool
		if err := rows.Scan(&matchID, &gamertag, &teamID, &isSelf, &isBot); err != nil {
			continue
		}
		if len(out[matchID]) >= 12 {
			continue
		}
		name := "?"
		if gamertag.Valid {
			name = strings.TrimSpace(gamertag.String)
			if name == "" {
				name = "?"
			}
		}
		entry := domain.MediaMatchLobbyEntry{Gamertag: name, IsSelf: isSelf, IsBot: isBot}
		if teamID.Valid {
			t := int(teamID.Int64)
			entry.TeamID = &t
		}
		out[matchID] = append(out[matchID], entry)
	}
	return out
}

// Miroir Go de q37MediaModeLabelExpr : extraction du mode parent.
//   - Si le label contient ":" â†’ prÃ©fixe avant (Arena:Slayer â†’ Arena)
//   - Sinon â†’ strip suffixes carte/Forge/Ranked
var (
	modeLabelOnRe     = regexp.MustCompile(`(?i)\s+on\s+.+$`)
	modeLabelForgeRe  = regexp.MustCompile(`(?i)\s*-\s*Forge\b.*$`)
	modeLabelRankedRe = regexp.MustCompile(`(?i)\s*-\s*Ranked\b.*$`)
)

func normalizeModeLabel(s string) string {
	if idx := strings.Index(s, ":"); idx >= 0 {
		return strings.TrimSpace(s[:idx])
	}
	s = modeLabelOnRe.ReplaceAllString(s, "")
	s = modeLabelForgeRe.ReplaceAllString(s, "")
	s = modeLabelRankedRe.ReplaceAllString(s, "")
	return strings.TrimSpace(s)
}

// SetMediaMatchAssociation force l'association d'un mÃ©dia Ã  un match prÃ©cis.
// Supprime l'association existante (si prÃ©sente) et insÃ¨re la nouvelle.
// Retourne (mapName, modeName) pour permettre au handler d'enrichir la rÃ©ponse.
