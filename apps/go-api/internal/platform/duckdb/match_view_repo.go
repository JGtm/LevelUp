// Package duckdb — MatchViewRepo : données pour la vue détail d'un match.
//
// Le code est découpé en fichiers thématiques pour respecter la limite des
// 500 lignes par fichier (CLAUDE.md). Ce fichier contient le type repo, les
// helpers de résolution d'assets et les 2 lectures de base (meta, player stats,
// enrichment) + le sanitize_f64 partagé. Les autres responsabilités vivent
// dans :
//
//   - match_view_repo_options.go         — constructeur, options With*, viewer,
//     sharedRead (déplacé le 2026-09-05, sans changement de logique : l'arrivée
//     de WithBombStats faisait franchir les 500 lignes à ce fichier)
//   - match_view_repo_scoreboard.go      — scoreboard + objective score
//   - match_view_repo_medals.go          — médailles (single + bulk + lookup)
//   - match_view_repo_weapons.go         — armes (single + bulk + lookup helpers)
//   - match_view_repo_neighbors_skill.go — navigation + CSR/LUSR + shared CSRs
//   - match_view_repo_extras.go          — events, KV pairs, encounters, media,
//     expected stats, history avg, assists model
package duckdb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"strings"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/port"
)

// MatchViewRepo implémente port.MatchViewRepository.
type MatchViewRepo struct {
	pdb  *PlayerDB
	xuid string
	// sharedReader (optionnel) override le SharedReader des lectures shared — câblé au
	// pilote snapshot (lecture decouplee du B-swap, fallback live). nil = comportement
	// historique (pdb.SharedReadDB() = live B-swap). SCOPED a MatchView : seules les
	// lectures shared MATCH-IMMUTABLES de cette page passent par le snapshot ; les
	// relations shared non-match-immutables (ex. world_csr_leaderboard) lues par
	// d'autres repos restent sur le live. Media (SharedSocial) + player (ReadDB) idem.
	sharedReader SharedReader
	// modeTax : classification des modes du titre, injectée au wiring pour éviter
	// le couplage platform/duckdb → games/halo_infinite dans le filtrage neighbors
	// (F15-2). Zéro-value = pas de classification (clause ModeCategory omise).
	modeTax analysis.ModeTaxonomy
	// forceLive (scoped requête) : quand true, sharedRead() ignore l'override snapshot
	// et sert le live. Armé par GetMatchMeta lorsqu'un match EXISTE dans le shared live
	// mais est ABSENT du snapshot immuable courant (joué après le dernier cut, ou exclu
	// comme "partial" au moment du cut). Sans ça, la page 404 (meta introuvable) ou
	// s'affiche à moitié vide (scoreboard/events snapshot vides) alors que le live a la
	// donnée complète — cf. bouton « Voir les matchs » dont la liste vient du live.
	// Le repo est instancié PAR REQUÊTE (ServiceRegistry.MatchView) : ce champ n'est
	// jamais partagé entre requêtes, et il est armé (écrit) dans GetMatchMeta AVANT le
	// fan-out parallèle des autres lectures (loadMatchViewDataParallel) — pas de course.
	forceLive bool
	// killSourceClassifier : le traducteur « source de degat du film -> cle du registre
	// d armes » du titre, injecte au cablage (nil pour un titre qui n en fournit pas).
	// Non nil = les armes du match se lisent dans `match_kill_events_latest` plutot que
	// dans `v_weapon_kills` (bascule du 2026-09-01). Aucun `slug ==` : c est la presence
	// du traducteur qui decide, et elle vient d une capability.
	killSourceClassifier port.KillSourceClassifier
	// stripPlaylistCategory : le titre déclare-t-il CapPlaylistCategoryStrip
	// (libellés de playlist préfixés d'une catégorie matchmaking à retirer pour
	// l'affichage) ? Câblé au wiring depuis la CapabilityMap du titre — jamais de
	// slug ==. Zéro-value false = pas de strip (un titre dont les noms officiels
	// n'ont pas de préfixe, ex. Halo 5, garde "Super Fiesta Fête" entier).
	stripPlaylistCategory bool
	// bombStats : le titre déclare-t-il `film.bomb_stats` ? Câblé au wiring depuis la
	// CapabilityMap du titre (jamais un slug). Faux = la SECONDE requête du scoreboard
	// (Q12cBombStats) n'est même pas payée, et aucune colonne d'Assaut n'est exposée — Halo 5
	// n'a pas de décodeur de film, donc rien à lire.
	bombStats bool
	// playlistLabelOverrides : table data-driven nom brut -> libelle court, chargee
	// depuis config/titles/{slug}/mappings/playlist_labels.toml (ex. Halo 5
	// "Super Fiesta Fete" -> "Super Fiesta"). nil/vide = no-op. Appliquee APRES le
	// strip eventuel : correction explicite par nom, jamais heuristique.
	playlistLabelOverrides map[string]string
	// viewerSlug : joueur qui REGARDE la page match (session), injecté par requête
	// au wiring. Détermine l'état du cœur des médias de l'onglet Médias (Q24) :
	// `liked` = « liké PAR CE VIEWER ». Distinct du joueur dont on consulte la
	// page. Vide → repli sur ce dernier, cf. viewer().
	viewerSlug string
}

// GetMatchMeta retourne les métadonnées du match (Q13).
// Exécutée sur SharedReader (ADR 0016) — Q13 lit match_registry (shared-only).
//
// Fallback snapshot→live : le reader par défaut de MatchView est le snapshot immuable
// (découplé du B-swap). Un match présent dans le shared LIVE mais absent du snapshot
// courant (joué après le dernier cut, ou exclu comme "partial") renverrait sql.ErrNoRows
// → 404, alors que les surfaces qui lisent le live (filters/timeseries/history, d'où
// vient la liste du bouton « Voir les matchs ») le proposent. On re-tente donc sur le
// live et, en cas de succès, on ARME forceLive pour que TOUTES les lectures shared
// suivantes de cette requête (scoreboard/events/médailles…) viennent aussi du live —
// sinon la page s'afficherait à moitié vide.
func (r *MatchViewRepo) GetMatchMeta(ctx context.Context, matchID string) (*domain.MatchMetaRaw, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	row, err := r.scanMatchMeta(ctx, r.sharedRead(), matchID)
	if err != nil && r.sharedReader != nil && !r.forceLive {
		// LA BASCULE COUVRE TOUTE ERREUR DU SNAPSHOT, pas seulement la ligne absente.
		// Un snapshot est un artefact FIGE d'un schema passe : toute colonne ajoutee a
		// Q13 apres son cut le fait echouer en Binder Error jusqu'au cut suivant — vecu
		// le 2026-08-29 (colonnes de manches team_*_rounds_won ajoutees a Q13 le jour
		// meme, snapshot du 27/08 sans elles : 404 « match introuvable » sur TOUS les
		// matchs, l'erreur etant avalee en amont par le mapping not_found du service).
		// Le live, lui, porte toujours le schema courant : c'est la degradation juste.
		if errors.Is(err, sql.ErrNoRows) {
			slog.WarnContext(ctx, "match_view: match absent du snapshot immuable → bascule lecture live",
				"match_id", matchID, "title", r.pdb.TitleSlug)
			observability.IncCounterT(r.pdb.TitleSlug, "match_view_snapshot_miss_live_fallback_total")
		} else {
			// ERREUR et pas Warn : chaque requete de vue de match paiera ce detour
			// jusqu'au prochain cut de snapshot — c'est un signal d'exploitation.
			slog.ErrorContext(ctx, "match_view: requete snapshot en echec (schema en retard ?) → bascule lecture live",
				"match_id", matchID, "title", r.pdb.TitleSlug, "err", err)
			observability.IncCounterT(r.pdb.TitleSlug, "match_view_snapshot_stale_schema_live_fallback_total")
		}
		r.forceLive = true
		row, err = r.scanMatchMeta(ctx, r.pdb.SharedReadDB(), matchID)
	}
	if err != nil {
		return nil, fmt.Errorf("MatchViewRepo.GetMatchMeta: %w", err)
	}
	// Résolution unifiée des noms d'asset via MetadataRepo.ResolveAssetName.
	// Cascade FR-FR → fr → en-US → en (PreferredLangsForLocale("fr")), une seule
	// requête SQL par asset, source unique de vérité (asset_translations).
	if row.MapAssetID != nil {
		row.MapNameFR = r.resolveAssetName(ctx, "map", *row.MapAssetID)
		row.MapNameEN = r.resolveAssetNameEN(ctx, "map", *row.MapAssetID)
		row.MapImageURL = r.lookupMapImageURL(ctx, *row.MapAssetID)
		// Pas d'image locale curée + version connue → endpoint framework KindMapImage
		// (DiscoveryUGC, lazy + cache). ?v = version requise par DiscoveryUGC. UUIDs
		// URL-safe (pas d'échappement). Le header builder respecte ce MapImageURL.
		if (row.MapImageURL == nil || *row.MapImageURL == "") &&
			row.MapVersionID != nil && *row.MapVersionID != "" {
			u := fmt.Sprintf("/api/v1/assets/maps/%s/%s/image?v=%s",
				r.pdb.TitleSlug, *row.MapAssetID, *row.MapVersionID)
			row.MapImageURL = &u
		}
	}
	if row.PlaylistAssetID != nil {
		row.PlaylistNameFR = r.resolveAssetName(ctx, "playlist", *row.PlaylistAssetID)
		// Libellé d'affichage via le CHOKEPOINT UNIQUE partagé (strip de catégorie
		// title-aware + override data-driven playlist_labels.toml) — même résolution
		// que les tuiles home / sessions / Explorer (analysis.DisplayPlaylistLabel).
		// Halo 5 « Super Fiesta Fête » → « Super Fiesta » ; Infinite « Arène delta :
		// Héritage » → « Delta : Héritage ». No-op si nom absent.
		if row.PlaylistNameFR != nil {
			disp := analysis.DisplayPlaylistLabel(*row.PlaylistNameFR, r.stripPlaylistCategory, r.playlistLabelOverrides)
			row.PlaylistNameFR = &disp
		}
	}
	// Résolution du libellé de mode — même cascade que applyMatchHistoryFRTranslations :
	// ResolveAssetNamesBulk(pair) → loadModeFRBatch(mode_name_tr) → ResolvePairNameFR.
	// pair_name_fr est toujours NULL en DB (non écrit par sync) : seul ce chemin
	// produit "Capture du drapeau" au lieu de l'EN normalisé.
	{
		var pairAssetName string
		if r.pdb.Metadata != nil && row.PairAssetID != nil {
			metaRepo := NewMetadataRepoFromDB(r.pdb.Metadata)
			langs := PreferredLangsForLocale("fr")
			pairNames, _ := metaRepo.ResolveAssetNamesBulk(ctx, "pair", []string{*row.PairAssetID}, langs)
			pairAssetName = strings.TrimSpace(pairNames[*row.PairAssetID])
		}

		rawPairName := derefString(row.PairName)
		normRaw := analysis.NormalizeModeLabel(rawPairName)
		normAsset := analysis.NormalizeModeLabel(pairAssetName)

		// Rattrapage des variantes non canoniques ("Legacy Slayer BR" → "Slayer")
		// via extraction du mode connu (liste mode_en de mode_name_tr). Appliqué
		// APRÈS NormalizeModeLabel pour préserver les préfixes d'identité de
		// playlist (Super Fiesta…). Cf. analysis.ExtractKnownMode.
		knownModes := loadKnownModesEN(ctx, r.pdb.Metadata)
		extracted := analysis.ExtractKnownMode(normRaw, knownModes)
		if extracted == "" {
			extracted = analysis.ExtractKnownMode(normAsset, knownModes)
		}

		modeENSet := make(map[string]struct{})
		for _, en := range []string{normRaw, normAsset, extracted} {
			if en != "" {
				modeENSet[en] = struct{}{}
			}
		}
		modeFR := loadModeFRBatch(ctx, r.pdb, modeENSet)

		// Priorité au mode canonique extrait + traduit (variante → mode connu FR),
		// sinon cascade historique ResolvePairNameFR (modes standards / cas limites).
		if extracted != "" && modeFR[extracted] != "" {
			fr := modeFR[extracted]
			row.ModeNameFR = &fr
		} else if fr := analysis.ResolvePairNameFR(rawPairName, derefString(row.PairNameFR), pairAssetName, modeFR); fr != "" {
			row.ModeNameFR = &fr
		}
	}
	// Fallback mode data-driven (title-agnostic) : quand le titre n'a PAS de pair
	// (ni ID ni nom — cas Halo 5, PairMode nil dans l'adapter) mais porte un
	// game_variant, le mode est résolu via asset_translations type "game_variant"
	// (même cascade FR-FR→fr→en-US→en). Sans ça mode_ui reste vide sur 100 % des
	// matchs H5. N'affecte pas les titres à pair (Infinite : PairAssetID présent →
	// condition fausse, comportement inchangé).
	if (row.ModeNameFR == nil || strings.TrimSpace(*row.ModeNameFR) == "") &&
		row.PairAssetID == nil && strings.TrimSpace(derefString(row.PairName)) == "" &&
		row.GameVariantAssetID != nil && strings.TrimSpace(*row.GameVariantAssetID) != "" {
		if gv := r.resolveAssetName(ctx, "game_variant", *row.GameVariantAssetID); gv != nil {
			row.ModeNameFR = gv
		}
	}
	// Durée de jeu observée (socle du flag « Prolongation »). Lecture SÉPARÉE de
	// Q13 et best-effort : elle dépend de colonnes de participation qui peuvent
	// manquer sur une DB non migrée — hors de question de faire tomber la Match
	// View entière pour un badge (leçon G1).
	r.attachElapsedSeconds(ctx, &row)
	return &row, nil
}

// attachElapsedSeconds renseigne row.ElapsedSeconds (durée de jeu observée) via
// l'estimateur partagé. Best-effort : sur échec, le champ reste nil et le service
// n'expose simplement pas de flag « Prolongation ».
func (r *MatchViewRepo) attachElapsedSeconds(ctx context.Context, row *domain.MatchMetaRaw) {
	sharedDB, release, err := r.sharedRead().Get(ctx)
	if err != nil {
		slog.WarnContext(ctx, "match_view: durée de jeu observée indisponible (shared reader)",
			"match_id", row.MatchID, "err", err)
		return
	}
	defer release()
	if secs, ok := LoadElapsedSecondsForMatch(ctx, sharedDB, row.MatchID); ok {
		row.ElapsedSeconds = &secs
	}
}

// scanMatchMeta exécute Q13 sur le reader fourni et scanne les 22 colonnes brutes.
// Isolé pour permettre le fallback snapshot→live de GetMatchMeta (on ré-exécute la
// même query sur le live quand le snapshot immuable ne contient pas le match). Renvoie
// l'erreur de Scan telle quelle (sql.ErrNoRows non enveloppé) pour que l'appelant
// puisse la détecter via errors.Is. Le handle shared est relâché à la sortie : la
// résolution d'assets qui suit dans GetMatchMeta n'utilise que la metadata (pas ce
// handle), le release anticipé est donc sûr.
func (r *MatchViewRepo) scanMatchMeta(ctx context.Context, reader SharedReader, matchID string) (domain.MatchMetaRaw, error) {
	var row domain.MatchMetaRaw
	sharedDB, release, err := reader.Get(ctx)
	if err != nil {
		return row, fmt.Errorf("shared reader: %w", err)
	}
	defer release()

	err = sharedDB.QueryRowContext(ctx, Q13MatchMeta, matchID).Scan(
		&row.MatchID,
		&row.StartTime,
		&row.DurationSeconds,
		&row.MapName,
		&row.PairName,
		&row.PlaylistName,
		&row.IsFirefight,
		&row.IsRanked,
		&row.PlayableDurationSeconds,
		&row.MapAssetID,
		&row.GameVariantName,
		&row.PlaylistAssetID,
		&row.Team0Score,
		&row.Team1Score,
		&row.Team0RoundsWon,
		&row.Team1RoundsWon,
		&row.RoundsTotal,
		&row.PairNameFR,
		&row.PairAssetID,
		&row.GameVariantAssetID,
		&row.T0Ms,
		&row.MapVersionID,
	)
	return row, err
}

// resolveAssetName est un helper qui appelle MetadataRepo.ResolveAssetName avec
// les préférences locale=FR par défaut. Retourne nil si la métadonnée DB n'est
// pas disponible ou si l'asset n'a aucune traduction.
func (r *MatchViewRepo) resolveAssetName(ctx context.Context, assetType, assetID string) *string {
	if r.pdb.Metadata == nil {
		return nil
	}
	meta := NewMetadataRepoFromDB(r.pdb.Metadata)
	name, _, ok, err := meta.ResolveAssetName(ctx, assetType, assetID, PreferredLangsForLocale("fr"))
	if err != nil || !ok || strings.TrimSpace(name) == "" {
		return nil
	}
	return &name
}

// resolveAssetNameEN retourne le nom EN canonique d'un asset (sans cascade
// FR) — utilisé pour les lookups d'image qui dépendent du nom EN canonique
// (l'adapter `AssetURLAdapter` indexe `static/maps/{titleSlug}/{name}.{ext}`
// par nom EN, pas par nom localisé).
func (r *MatchViewRepo) resolveAssetNameEN(ctx context.Context, assetType, assetID string) *string {
	if r.pdb.Metadata == nil {
		return nil
	}
	meta := NewMetadataRepoFromDB(r.pdb.Metadata)
	name, _, ok, err := meta.ResolveAssetName(ctx, assetType, assetID, PreferredLangsForLocale("en"))
	if err != nil || !ok || strings.TrimSpace(name) == "" {
		return nil
	}
	return &name
}

// lookupMapImageURL retourne l'URL de l'image de map depuis map_images_registry
// par map_id (UUID stable). Pattern identique à home_repo.loadHomeMapImageURLs.
// Nil si map_id absent du registry ou si metadata.duckdb indisponible.
func (r *MatchViewRepo) lookupMapImageURL(ctx context.Context, mapAssetID string) *string {
	if r.pdb.Metadata == nil {
		return nil
	}
	const q = `
		SELECT local_path FROM map_images_registry
		WHERE title_id = ? AND map_id = ?
		  AND TRIM(local_path) != ''
		LIMIT 1`
	rows, err := r.pdb.Metadata.Query(ctx, q, r.pdb.TitleSlug, mapAssetID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	if rows.Next() {
		var path string
		if err := rows.Scan(&path); err == nil && path != "" {
			return &path
		}
	}
	return nil
}

// GetPlayerMatchStats retourne les stats du joueur pour ce match (Q17).
// Exécutée sur SharedReader (ADR 0016) — Q17 lit match_participants (shared-only).
func (r *MatchViewRepo) GetPlayerMatchStats(ctx context.Context, xuid, matchID string) (*domain.PlayerMatchStatsRaw, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	sharedDB, release, err := r.sharedRead().Get(ctx)
	if err != nil {
		return &domain.PlayerMatchStatsRaw{}, nil //nolint:nilerr
	}
	defer release()

	var s domain.PlayerMatchStatsRaw
	err = sharedDB.QueryRowContext(ctx, Q17PlayerMatchStats, matchID, xuid).Scan(
		&s.OutcomeCode,
		&s.TeamID,
		&s.RankInTeam,
		&s.Kills,
		&s.Deaths,
		&s.Assists,
		&s.KDA,
		&s.Accuracy,
		&s.PersonalScore,
		&s.AvgLifeSeconds,
		&s.TimePlayedSeconds,
		&s.ShotsFired,
		&s.ShotsHit,
		&s.DamageDealt,
		&s.DamageTaken,
		&s.TeamMMR,
		&s.EnemyMMR,
		&s.HeadshotKills,
		&s.MaxKillingSpree,
		&s.BackfillBits,
	)
	if err != nil {
		// Le joueur peut ne pas avoir participé → retourner une stats vide
		return &domain.PlayerMatchStatsRaw{}, nil
	}
	return &s, nil
}

// IsParticipant indique si le joueur (xuid) figure dans match_participants pour
// ce match. Sert au gating "match non-participé" (ADR 0029, Couche B). EXISTS
// léger — ne scanne pas les 31 colonnes de Q17. Exécutée sur SharedReader.
func (r *MatchViewRepo) IsParticipant(ctx context.Context, xuid, matchID string) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	sharedDB, release, err := r.sharedRead().Get(ctx)
	if err != nil {
		return false, err
	}
	defer release()

	var exists bool
	if err := sharedDB.QueryRowContext(ctx, Q17bIsParticipant, matchID, xuid).Scan(&exists); err != nil {
		return false, err
	}
	return exists, nil
}

// GetMatchEnrichment retourne l'enrichissement pour ce match (Q18).
func (r *MatchViewRepo) GetMatchEnrichment(ctx context.Context, matchID string) (*domain.MatchEnrichmentRaw, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var e domain.MatchEnrichmentRaw
	rows, err := r.pdb.ReadDB().QueryRowRecovered(ctx, Q18MatchEnrichment, matchID)
	if err != nil {
		// Pas d'enrichissement (ou Reopen concurrent) → retourner vide
		return &domain.MatchEnrichmentRaw{}, nil
	}
	defer rows.Close()
	if err := rows.Scan(
		&e.PerformanceScore,
		&e.IsWithFriends,
		&e.IsExcluded,
		&e.DominanceFlag,
	); err != nil {
		return &domain.MatchEnrichmentRaw{}, nil
	}
	return &e, nil
}

// sanitizeF64 remplace les valeurs NaN/Inf par nil. json.Marshal rejette NaN
// et +/-Inf (non représentables en JSON), ce qui provoque un corps HTTP vide
// quand writeJSON ignorait silencieusement l'erreur.
func sanitizeF64(f *float64) *float64 {
	if f == nil || math.IsNaN(*f) || math.IsInf(*f, 0) {
		return nil
	}
	return f
}
