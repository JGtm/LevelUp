package duckdb

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/migration"
)

func openBattlePassTestDB(t *testing.T, filename string, target migration.TargetDB) *DB {
	t.Helper()
	path := filepath.Join(t.TempDir(), filename)
	db, err := OpenReadWrite(path)
	if err != nil {
		t.Fatalf("OpenReadWrite(%s): %v", filename, err)
	}
	t.Cleanup(func() { _ = db.Close() })
	_ = migration.All()
	if err := migration.RunForDB(db.SQLDb(), target); err != nil {
		t.Fatalf("RunForDB(%s): %v", filename, err)
	}
	return db
}

func sampleBattlePassPayload() []byte {
	return []byte(`{
		"ActiveOperationRewardTrackPath":"RewardTracks/TrackA",
		"OperationRewardTracks":[
			{
				"RewardTrackPath":"RewardTracks/TrackA",
				"CurrentProgress":{
					"Rank":12,
					"PartialProgress":300,
					"IsOwned":true,
					"HasReachedMaxRank":false
				},
				"IsOwned":true,
				"BaseXp":1200,
				"BoostXp":200
			},
			{
				"RewardTrackPath":"RewardTracks/TrackB",
				"CurrentProgress":{
					"Rank":20,
					"PartialProgress":0,
					"IsOwned":true,
					"HasReachedMaxRank":true
				},
				"IsOwned":true,
				"BaseXp":2000,
				"BoostXp":0
			}
		]
	}`)
}

func TestPersistSinkWriteBattlePass_PersistsAndDeduplicatesSnapshots(t *testing.T) {
	ctx := context.Background()
	meta := openBattlePassTestDB(t, "metadata.duckdb", migration.TargetMetadata)
	player := openBattlePassTestDB(t, "player.duckdb", migration.TargetPlayer)

	sink := NewPersistSink(meta.Path(), player.Path(), "xuid-1")
	if err := sink.writeBattlePass(ctx, "RewardTracks/TrackA", sampleBattlePassPayload()); err != nil {
		t.Fatalf("writeBattlePass first call: %v", err)
	}
	if err := sink.writeBattlePass(ctx, "RewardTracks/TrackA", sampleBattlePassPayload()); err != nil {
		t.Fatalf("writeBattlePass second call: %v", err)
	}

	var count int
	if err := player.QueryRow(ctx, "SELECT COUNT(*) FROM battlepass_snapshots").Scan(&count); err != nil {
		t.Fatalf("count battlepass_snapshots: %v", err)
	}
	if count != 2 {
		t.Fatalf("battlepass_snapshots count = %d, want 2", count)
	}

	var isActive bool
	var rank, partial, baseXP, boostXP int
	if err := player.QueryRow(ctx, `
		SELECT is_active, current_rank, partial_progress, base_xp, boost_xp
		FROM battlepass_snapshots
		WHERE reward_track_path = 'RewardTracks/TrackA'
		ORDER BY snapshot_at DESC
		LIMIT 1`).Scan(&isActive, &rank, &partial, &baseXP, &boostXP); err != nil {
		t.Fatalf("select active snapshot: %v", err)
	}
	if !isActive || rank != 12 || partial != 300 || baseXP != 1200 || boostXP != 200 {
		t.Fatalf("snapshot TrackA invalide: active=%v rank=%d partial=%d base=%d boost=%d", isActive, rank, partial, baseXP, boostXP)
	}
}

func TestHomeRepoLoadCachedBattlePass_UsesPlayerSnapshots(t *testing.T) {
	ctx := context.Background()
	meta := openBattlePassTestDB(t, "metadata.duckdb", migration.TargetMetadata)
	player := openBattlePassTestDB(t, "player.duckdb", migration.TargetPlayer)

	_, err := player.Exec(ctx, `
		INSERT INTO battlepass_snapshots
			(snapshot_at, xuid, reward_track_path, is_active, current_rank, partial_progress,
			 is_owned, has_reached_max_rank, base_xp, boost_xp, state_hash, raw_payload_json)
		VALUES (CURRENT_TIMESTAMP, 'xuid-1', 'RewardTracks/TrackA', TRUE, 14, 450, TRUE, FALSE, 1400, 250, 'state-a', '{}')`)
	if err != nil {
		t.Fatalf("insert battlepass snapshot: %v", err)
	}

	repo := NewHomeRepo(&PlayerDB{Player: player, Metadata: meta, XUID: "xuid-1"})
	resp, hit, err := repo.LoadCachedBattlePass(ctx, time.Hour)
	if err != nil {
		t.Fatalf("LoadCachedBattlePass: %v", err)
	}
	if !hit || resp == nil {
		t.Fatal("LoadCachedBattlePass: attendu un hit")
	}
	if resp.RewardTrack == nil || *resp.RewardTrack != "RewardTracks/TrackA" {
		t.Fatalf("reward_track = %v, want RewardTracks/TrackA", resp.RewardTrack)
	}
	if resp.Rank == nil || *resp.Rank != 14 {
		t.Fatalf("rank = %v, want 14", resp.Rank)
	}
	if resp.Progress == nil || *resp.Progress != 450 {
		t.Fatalf("progress = %v, want 450", resp.Progress)
	}
	if !resp.FromCache {
		t.Fatal("FromCache devrait etre true")
	}
}

func TestSeasonPassRepoLoadSeasonPassTracks_UsesPlayerSnapshots(t *testing.T) {
	ctx := context.Background()
	meta := openBattlePassTestDB(t, "metadata.duckdb", migration.TargetMetadata)
	player := openBattlePassTestDB(t, "player.duckdb", migration.TargetPlayer)

	_, err := meta.Exec(ctx, `
		INSERT INTO battlepass_track_definitions
			(reward_track_path, content_hash, xp_per_rank, battlepass_image_path, background_image_path,
			 raw_payload_json, is_current, first_seen_at, last_seen_at)
		VALUES
			('RewardTracks/TrackA', 'hash-a', 1000, 'progression/track-a.png', 'progression/bg-a.png', '{"Name":{"fr":"Operation Alpha"},"Description":{"fr":"Escalade principale"},"BattlePassImage":"progression/track-a.png","BackgroundImagePath":"progression/bg-a.png","XpPerRank":1000,"Ranks":[{"Rank":1,"FreeRewards":{"InventoryRewards":[{"InventoryItemPath":"Inventory/Reward-1.json"}]},"PaidRewards":{"InventoryRewards":[]}},{"Rank":2,"FreeRewards":{"InventoryRewards":[]},"PaidRewards":{"InventoryRewards":[{"InventoryItemPath":"Inventory/Reward-2.json"}]}},{"Rank":3,"FreeRewards":{"InventoryRewards":[]},"PaidRewards":{"InventoryRewards":[]}}]}', TRUE, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
			('RewardTracks/TrackB', 'hash-b', 1000, 'https://img/track-b.png', 'https://img/bg-b.png', '{}', FALSE, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`)
	if err != nil {
		t.Fatalf("insert track definitions: %v", err)
	}
	_, err = meta.Exec(ctx, `
		INSERT INTO battlepass_track_translations
			(reward_track_path, content_hash, lang, track_name, first_seen_at, last_seen_at)
		VALUES
			('RewardTracks/TrackA', 'hash-a', 'fr', 'Operation Alpha', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
			('RewardTracks/TrackB', 'hash-b', 'fr', 'Operation Beta', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`)
	if err != nil {
		t.Fatalf("insert track translations: %v", err)
	}
	_, err = meta.Exec(ctx, `
		INSERT INTO battlepass_item_definitions
			(inventory_item_path, content_hash, quality, item_type, display_path, raw_payload_json, is_current, first_seen_at, last_seen_at)
		VALUES
			('Inventory/Reward-1.json', 'item-1', 'rare', 'ArmorCoating', 'progression/items/reward-1.png', '{}', TRUE, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
			('Inventory/Reward-2.json', 'item-2', 'epic', 'ArmorEffect', 'progression/items/reward-2.png', '{}', TRUE, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`)
	if err != nil {
		t.Fatalf("insert item definitions: %v", err)
	}
	_, err = meta.Exec(ctx, `
		INSERT INTO battlepass_item_translations
			(inventory_item_path, content_hash, lang, title, description, first_seen_at, last_seen_at)
		VALUES
			('Inventory/Reward-1.json', 'item-1', 'fr', 'Récompense 1', 'Description 1', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
			('Inventory/Reward-2.json', 'item-2', 'fr', 'Récompense 2', 'Description 2', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`)
	if err != nil {
		t.Fatalf("insert item translations: %v", err)
	}

	_, err = player.Exec(ctx, `
		INSERT INTO battlepass_snapshots
			(snapshot_at, xuid, reward_track_path, is_active, current_rank, partial_progress,
			 is_owned, has_reached_max_rank, base_xp, boost_xp, state_hash, raw_payload_json)
		VALUES
			(CURRENT_TIMESTAMP, 'xuid-1', 'RewardTracks/TrackA', TRUE, 12, 300, TRUE, FALSE, 1200, 200, 'state-a', '{}'),
			(CURRENT_TIMESTAMP, 'xuid-1', 'RewardTracks/TrackB', FALSE, 20, 0, TRUE, TRUE, 2000, 0, 'state-b', '{}')`)
	if err != nil {
		t.Fatalf("insert player snapshots: %v", err)
	}

	repo := NewSeasonPassRepo(&PlayerDB{Player: player, Metadata: meta, XUID: "xuid-1"})
	tracks, err := repo.LoadSeasonPassTracks(ctx, "xuid-1", "halo_infinite")
	if err != nil {
		t.Fatalf("LoadSeasonPassTracks: %v", err)
	}
	if len(tracks) != 2 {
		t.Fatalf("len(tracks) = %d, want 2", len(tracks))
	}

	trackByPath := map[string]domain.SeasonPassTrackSummary{}
	for _, track := range tracks {
		trackByPath[track.RewardTrackPath] = track
	}

	active := trackByPath["RewardTracks/TrackA"]
	if !active.IsActive || active.Status != domain.SeasonPassStatusActive || active.CurrentRank != 12 {
		t.Fatalf("TrackA invalide: %+v", active)
	}
	if active.ImageURL == nil || *active.ImageURL != "/api/v1/assets/battlepass/tracks/progression/track-a.png" {
		t.Fatalf("TrackA image_url invalide: %+v", active.ImageURL)
	}
	if active.Description == nil || *active.Description != "Escalade principale" {
		t.Fatalf("TrackA description invalide: %+v", active.Description)
	}
	if active.ActiveTierRank == nil || *active.ActiveTierRank != 3 {
		t.Fatalf("TrackA active_tier_rank invalide: %+v", active.ActiveTierRank)
	}
	if active.ActiveTierProgressPercent == nil || *active.ActiveTierProgressPercent != 30 {
		t.Fatalf("TrackA active_tier_progress_percent invalide: %+v", active.ActiveTierProgressPercent)
	}
	if len(active.Tiers) != 3 {
		t.Fatalf("TrackA tiers len = %d, want 3", len(active.Tiers))
	}
	if active.Tiers[0].ImageURL == nil || *active.Tiers[0].ImageURL != "/api/v1/assets/battlepass/tier/progression/items/reward-1.png" {
		t.Fatalf("TrackA tier 1 image invalide: %+v", active.Tiers[0].ImageURL)
	}
	if !active.Tiers[1].IsObtained || active.Tiers[1].IsCurrent {
		t.Fatalf("TrackA tier 2 etat invalide: %+v", active.Tiers[1])
	}
	if !active.Tiers[2].IsCurrent || active.Tiers[2].Title != "Palier 3" {
		t.Fatalf("TrackA tier 3 invalide: %+v", active.Tiers[2])
	}

	completed := trackByPath["RewardTracks/TrackB"]
	if completed.Status != domain.SeasonPassStatusCompleted || !completed.HasReachedMaxRank || completed.CurrentRank != 20 {
		t.Fatalf("TrackB invalide: %+v", completed)
	}
}

// TestSeasonPassRepoLoadSeasonPassTracks_FallbackTitleFromRawPayload couvre le
// cas où battlepass_item_translations est vide (items pré-déploiement de la
// table translations) mais raw_payload_json contient bien CommonData.Title.
// Le COALESCE doit extraire le titre via json_extract_string et éviter le
// fallback "path brut comme titre".
func TestSeasonPassRepoLoadSeasonPassTracks_FallbackTitleFromRawPayload(t *testing.T) {
	ctx := context.Background()
	meta := openBattlePassTestDB(t, "metadata.duckdb", migration.TargetMetadata)
	player := openBattlePassTestDB(t, "player.duckdb", migration.TargetPlayer)

	_, err := meta.Exec(ctx, `
		INSERT INTO battlepass_track_definitions
			(reward_track_path, content_hash, xp_per_rank, battlepass_image_path, background_image_path,
			 raw_payload_json, is_current, first_seen_at, last_seen_at)
		VALUES
			('RewardTracks/TrackC', 'hash-c', 1000, 'progression/track-c.png', 'progression/bg-c.png',
			 '{"BattlePassImage":"progression/track-c.png","BackgroundImagePath":"progression/bg-c.png","XpPerRank":1000,"Ranks":[{"Rank":1,"FreeRewards":{"InventoryRewards":[{"InventoryItemPath":"Inventory/HelmetMarkVB.json"}]},"PaidRewards":{"InventoryRewards":[]}}]}',
			 TRUE, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`)
	if err != nil {
		t.Fatalf("insert track: %v", err)
	}

	// Item définition avec raw_payload_json contenant CommonData.Title.translations
	// MAIS sans entrée correspondante dans battlepass_item_translations.
	rawItem := `{"CommonData":{"Title":{"value":"EOD GEN1","translations":{"fr-FR":"Casque EOD GEN1 FR","en-US":"EOD GEN1 EN"}},"Description":{"value":"Helmet desc","translations":{"fr-FR":"Description casque FR"}},"Quality":"Epic","DisplayPath":{"Media":{"MediaUrl":{"Path":"progression/Inventory/Armor/Helmets/eod.png"}}}}}`
	_, err = meta.Exec(ctx, `
		INSERT INTO battlepass_item_definitions
			(inventory_item_path, content_hash, quality, item_type, display_path,
			 raw_payload_json, is_current, first_seen_at, last_seen_at)
		VALUES
			('Inventory/HelmetMarkVB.json', 'item-c', 'Epic', '',
			 'progression/Inventory/Armor/Helmets/eod.png', ?,
			 TRUE, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`, rawItem)
	if err != nil {
		t.Fatalf("insert item def: %v", err)
	}

	_, err = player.Exec(ctx, `
		INSERT INTO battlepass_snapshots
			(snapshot_at, xuid, reward_track_path, is_active, current_rank, partial_progress,
			 is_owned, has_reached_max_rank, base_xp, boost_xp, state_hash, raw_payload_json)
		VALUES
			(CURRENT_TIMESTAMP, 'xuid-2', 'RewardTracks/TrackC', TRUE, 1, 0, TRUE, FALSE, 0, 0, 'state-c', '{}')`)
	if err != nil {
		t.Fatalf("insert snapshot: %v", err)
	}

	repo := NewSeasonPassRepo(&PlayerDB{Player: player, Metadata: meta, XUID: "xuid-2"})
	tracks, err := repo.LoadSeasonPassTracks(ctx, "xuid-2", "halo_infinite")
	if err != nil {
		t.Fatalf("LoadSeasonPassTracks: %v", err)
	}
	if len(tracks) != 1 || len(tracks[0].Tiers) == 0 {
		t.Fatalf("tracks/tiers manquants: %+v", tracks)
	}

	tier := tracks[0].Tiers[0]
	if tier.Title != "Casque EOD GEN1 FR" {
		t.Errorf("tier.Title = %q, want %q (extraction depuis raw_payload_json fr-FR)", tier.Title, "Casque EOD GEN1 FR")
	}
	if tier.Description == nil || *tier.Description != "Description casque FR" {
		t.Errorf("tier.Description = %v, want %q", tier.Description, "Description casque FR")
	}
	if len(tracks[0].Tiers[0].FreeRewards) == 0 {
		t.Fatalf("free_rewards manquants: %+v", tracks[0].Tiers[0])
	}
	freeReward := tracks[0].Tiers[0].FreeRewards[0]
	if freeReward.Title != "Casque EOD GEN1 FR" {
		t.Errorf("free_reward.Title = %q, want %q (pas le path brut)", freeReward.Title, "Casque EOD GEN1 FR")
	}
}
