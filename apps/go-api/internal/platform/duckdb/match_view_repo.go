// Package duckdb — MatchViewRepo : données pour la vue détail d'un match.
//
// Le code est découpé en fichiers thématiques pour respecter la limite des
// 500 lignes par fichier (CLAUDE.md). Ce fichier contient le type repo, le
// constructeur, les helpers de résolution d'assets et les 2 lectures de base
// (meta, player stats, enrichment) + le sanitize_f64 partagé. Les autres
// responsabilités vivent dans :
//
//   - match_view_repo_scoreboard.go      — scoreboard + objective score
//   - match_view_repo_medals.go          — médailles (single + bulk + lookup)
//   - match_view_repo_weapons.go         — armes (single + bulk + lookup helpers)
//   - match_view_repo_neighbors_skill.go — navigation + CSR/LUSR + shared CSRs
//   - match_view_repo_extras.go          — events, KV pairs, encounters, media,
//     expected stats, history avg, assists model
package duckdb

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
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
}

// NewMatchViewRepo crée un MatchViewRepo.
func NewMatchViewRepo(pdb *PlayerDB, xuid string) *MatchViewRepo {
	return &MatchViewRepo{pdb: pdb, xuid: xuid}
}

// WithSharedReader injecte un SharedReader override pour les lectures shared (pilote
// snapshot scoped). Retourne le repo pour chaînage. nil = no-op (reste sur pdb.SharedReadDB()).
func (r *MatchViewRepo) WithSharedReader(sr SharedReader) *MatchViewRepo {
	r.sharedReader = sr
	return r
}

// WithModeTaxonomy injecte la classification des modes du titre (préfixes pair_name
// par catégorie) pour le filtrage neighbors. Sans injection, la clause ModeCategory
// est omise (dégradation gracieuse). Câblé au wiring depuis games/halo_infinite (F15-2).
func (r *MatchViewRepo) WithModeTaxonomy(t analysis.ModeTaxonomy) *MatchViewRepo {
	r.modeTax = t
	return r
}

// sharedRead retourne le SharedReader effectif : l'override snapshot s'il est câblé,
// sinon le reader live du pool (pdb.SharedReadDB()).
func (r *MatchViewRepo) sharedRead() SharedReader {
	if r.sharedReader != nil {
		return r.sharedReader
	}
	return r.pdb.SharedReadDB()
}

// GetMatchMeta retourne les métadonnées du match (Q13).
// Exécutée sur SharedReader (ADR 0016) — Q13 lit match_registry (shared-only).
func (r *MatchViewRepo) GetMatchMeta(ctx context.Context, matchID string) (*domain.MatchMetaRaw, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	sharedDB, release, err := r.sharedRead().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("MatchViewRepo.GetMatchMeta: shared reader: %w", err)
	}
	defer release()

	var row domain.MatchMetaRaw
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
		&row.PairNameFR,
		&row.PairAssetID,
		&row.GameVariantAssetID,
		&row.T0Ms,
		&row.MapVersionID,
	)
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
		// Retire la catégorie matchmaking de tête ("Arène delta : Héritage" →
		// "Delta : Héritage") pour l'affichage.
		if row.PlaylistNameFR != nil {
			norm := analysis.NormalizePlaylistLabel(*row.PlaylistNameFR)
			row.PlaylistNameFR = &norm
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
	return &row, nil
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
	err := r.pdb.ReadDB().QueryRow(ctx, Q18MatchEnrichment, matchID).Scan(
		&e.PerformanceScore,
		&e.IsWithFriends,
		&e.IsExcluded,
		&e.DominanceFlag,
	)
	if err != nil {
		// Pas d'enrichissement → retourner vide
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
