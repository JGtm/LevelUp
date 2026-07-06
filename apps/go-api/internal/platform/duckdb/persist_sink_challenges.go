// Package duckdb — persist_sink_challenges.go : persistance synchrone des defis
// (waypoint_assets_raw + challenge_snapshots). Extrait de persist_sink.go (K3f god-file
// split, 2026-07-06), meme package. INSERT-only ART-safe inchange (ADR 0019).
package duckdb

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/dblease"
)

// ---------------------------------------------------------------------------
// Challenges
// ---------------------------------------------------------------------------

// PersistChallengesSync persiste les défis de manière SYNCHRONE dans
// waypoint_assets_raw (metadata) et challenge_snapshots (player). L'écriture est
// liée au ctx appelant (HTTP / ticker live_refresh) au lieu d'une goroutine
// détachée en context.Background() — garantit qu'elle se termine avant le
// shutdown (lifecycle, W6 revue 2026-06-01).
// items porte les ChallengeItem RENDUS (titre/description/image résolus côté provider) :
// ils sont persistés sur les snapshots actifs pour que le cache reconstruise de vraies
// cartes hors-ligne (au lieu de « Défis indisponibles ») quand le live est indisponible.
func (s *PersistSink) PersistChallengesSync(ctx context.Context, rawBody []byte, items []domain.ChallengeItem) error {
	if len(rawBody) == 0 {
		return nil
	}
	return s.writeChallenges(ctx, rawBody, items)
}

// deckChallengeRaw est le struct de parsing best-effort d'un challenge depuis /decks.
// Les champs sont lenients : si un champ est absent, il reste à zéro/vide.
type deckChallengeRaw struct {
	TrackingID      string `json:"TrackingId"`
	XPReward        int    `json:"XPReward"`
	SecXPReward     int    `json:"SecondaryXpReward"`
	Threshold       int    `json:"Threshold"`
	CurrentProgress int    `json:"CurrentProgress"`
	IsCompleted     bool   `json:"IsCompleted"`
	CanReroll       bool   `json:"CanReroll"`
	Expiration      struct {
		ISO8601Date string `json:"ISO8601Date"`
	} `json:"Expiration"`
}

// writeChallenges effectue les écritures dans metadata.duckdb et stats.duckdb.
// renderByTracking (issu des items rendus) alimente title/description/image_url des
// snapshots actifs → cartes reconstructibles depuis le cache.
func (s *PersistSink) writeChallenges(ctx context.Context, body []byte, items []domain.ChallengeItem) error {
	hash := persistHash(body)
	now := time.Now()
	renderByTracking := challengeRenderMap(items)

	// Structure /decks telle que parsée par le provider.
	var raw struct {
		AssignedDecks []struct {
			Expiration struct {
				ISO8601Date string `json:"ISO8601Date"`
			} `json:"Expiration"`
			ActiveChallenges    []json.RawMessage `json:"ActiveChallenges"`
			CompletedChallenges []json.RawMessage `json:"CompletedChallenges"`
		} `json:"AssignedDecks"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return fmt.Errorf("parse challenges body: %w", err)
	}

	// 1. Sauvegarder le blob brut dans waypoint_assets_raw (metadata.duckdb).
	if s.MetaPath != "" {
		relMeta, leaseErr := dblease.AcquireLease(s.MetaPath, dblease.MetadataLeaseTimeout)
		if leaseErr != nil {
			slog.Warn("persist_sink: meta lease for challenges failed", "err", leaseErr)
		} else {
			defer relMeta()
			db, err := OpenReadWrite(s.MetaPath)
			if err != nil {
				slog.Warn("persist_sink: open meta rw for challenges failed", "err", err)
			} else {
				defer db.Close()
				if err := upsertWaypointAsset(ctx, db,
					s.TitleSlug,
					s.XUID+"/challenge_deck",
					"challenge_deck",
					hash,
					string(body),
					now,
				); err != nil {
					slog.Warn("persist_sink: waypoint_assets_raw challenges upsert failed",
						"xuid", s.XUID, "err", err)
				}
			}
		}
	}

	// 2. Persister les snapshots dans challenge_snapshots (stats.duckdb).
	if s.PlayerPath == "" || s.XUID == "" {
		return nil
	}

	relPlayer, err := dblease.AcquireLease(s.PlayerPath, dblease.PlayerLeaseTimeout)
	if err != nil {
		return fmt.Errorf("writeChallenges lease player: %w", err)
	}
	defer relPlayer()

	pdb, err := OpenReadWrite(s.PlayerPath)
	if err != nil {
		return fmt.Errorf("open player rw: %w", err)
	}
	defer pdb.Close()

	for _, deck := range raw.AssignedDecks {
		deckExpiry := deck.Expiration.ISO8601Date
		for _, rawCh := range deck.ActiveChallenges {
			if err := s.insertSnapshot(ctx, pdb, rawCh, "Active", deckExpiry, now, renderByTracking); err != nil {
				slog.Warn("persist_sink: snapshot insert failed",
					"status", "Active", "xuid", s.XUID, "err", err)
			}
		}
		for _, rawCh := range deck.CompletedChallenges {
			if err := s.insertSnapshot(ctx, pdb, rawCh, "Completed", deckExpiry, now, nil); err != nil {
				slog.Warn("persist_sink: snapshot insert failed",
					"status", "Completed", "xuid", s.XUID, "err", err)
			}
		}
	}
	return nil
}

// challengeRenderMap indexe les items rendus par TrackingID (clé stable des snapshots).
// Permet à insertSnapshot d'attacher title/description/image_url au bon défi actif.
func challengeRenderMap(items []domain.ChallengeItem) map[string]domain.ChallengeItem {
	if len(items) == 0 {
		return nil
	}
	m := make(map[string]domain.ChallengeItem, len(items))
	for _, it := range items {
		if it.TrackingID != nil && *it.TrackingID != "" {
			m[*it.TrackingID] = it
		}
	}
	return m
}

// insertSnapshot insère un snapshot de défi si l'état a changé depuis le dernier
// enregistrement (déduplication par state_hash sur les dernières 24h).
// Retourne nil sans insérer si le challenge n'a pas de TrackingID.
func (s *PersistSink) insertSnapshot(
	ctx context.Context,
	db *DB,
	rawCh json.RawMessage,
	status, deckExpiry string,
	at time.Time,
	renderByTracking map[string]domain.ChallengeItem,
) error {
	var ch deckChallengeRaw
	if err := json.Unmarshal(rawCh, &ch); err != nil {
		return nil // skip malformed
	}
	if ch.TrackingID == "" {
		return nil // pas d'identifiant stable → skip
	}

	chPath := "Challenges/Tracking/" + ch.TrackingID
	stateHash := persistHash(rawCh)

	// Déduplication : ne pas insérer si un snapshot identique existe dans les 24h.
	var existing int
	err := db.QueryRow(ctx, `
		SELECT COUNT(*) FROM challenge_snapshots
		WHERE xuid = ? AND challenge_path = ? AND state_hash = ?
		  AND snapshot_at > CURRENT_TIMESTAMP - INTERVAL 1 DAY`,
		s.XUID, chPath, stateHash,
	).Scan(&existing)
	if err == nil && existing > 0 {
		return nil // état inchangé, pas besoin d'insérer
	}

	// Choix de l'expiration : priorité au champ du challenge, fallback sur le deck.
	expiry := deckExpiry
	if ch.Expiration.ISO8601Date != "" {
		expiry = ch.Expiration.ISO8601Date
	}

	var expiresAt interface{}
	if expiry != "" {
		expiresAt = expiry
	}

	// Métadonnées de rendu (titre/description/image + vrai chemin GameCMS) du défi actif
	// correspondant, pour reconstruire des cartes depuis le cache. nil pour les complétés.
	// display_path = le vrai path (...DailyChallenges/...) → le front dérive la cadence ;
	// challenge_path reste la clé de dedup synthétique.
	var title, description, imageURL, displayPath interface{}
	if item, ok := renderByTracking[ch.TrackingID]; ok {
		if item.Title != "" {
			title = item.Title
		}
		if item.Description != nil && *item.Description != "" {
			description = *item.Description
		}
		if item.ImageURL != nil && *item.ImageURL != "" {
			imageURL = *item.ImageURL
		}
		if item.ChallengePath != "" {
			displayPath = item.ChallengePath
		}
	}

	_, err = db.Exec(ctx, `
		INSERT INTO challenge_snapshots
			(snapshot_at, xuid, challenge_path, challenge_id,
			 status, progress_current, progress_target, xp_reward,
			 can_reroll, expires_at, state_hash, title, description, image_url, display_path)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		at, s.XUID, chPath, ch.TrackingID,
		status, ch.CurrentProgress, ch.Threshold, ch.XPReward,
		ch.CanReroll, expiresAt, stateHash, title, description, imageURL, displayPath,
	)
	return err
}

// ---------------------------------------------------------------------------
// Helper commun
// ---------------------------------------------------------------------------

// upsertWaypointAsset écrit ou met à jour un enregistrement dans waypoint_assets_raw.
func upsertWaypointAsset(
	ctx context.Context,
	db *DB,
	titleID, assetID, assetType, contentHash, rawJSON string,
	fetchedAt time.Time,
) error {
	// SELECT-then-write anti-ART (cf. (*DB).UpsertNoConflict) : pas d'ON CONFLICT
	// DO UPDATE sur metadata.duckdb, qui FATAL-invalide le handle partagé.
	// version_id = contentHash ici (clé naturelle de la version).
	err := db.UpsertNoConflict(ctx,
		`SELECT 1 FROM waypoint_assets_raw WHERE title_id = ? AND asset_id = ? AND version_id = ?`,
		[]any{titleID, assetID, contentHash},
		`UPDATE waypoint_assets_raw SET raw_json = ?, fetched_at = ?, content_hash = ?
		 WHERE title_id = ? AND asset_id = ? AND version_id = ?`,
		[]any{rawJSON, fetchedAt, contentHash, titleID, assetID, contentHash},
		`INSERT INTO waypoint_assets_raw
			(title_id, asset_id, asset_type, version_id,
			 name, description, raw_json, fetched_at, content_hash)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		[]any{titleID, assetID, assetType, contentHash,
			"", "", rawJSON, fetchedAt, contentHash},
	)
	if err != nil {
		return fmt.Errorf("upsertWaypointAsset %s: %w", assetID, err)
	}
	return nil
}
