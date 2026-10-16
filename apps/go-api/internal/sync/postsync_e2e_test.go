//go:build integration

package sync

// postsync_e2e_test.go — simulation de bout en bout d'un match d'escouade fraîchement
// synchronisé : premier écrit par le chemin de production (fetchAndPersistMatches), puis
// post-sync complet (runPostSyncPipeline), sur des bases migrées.
//
// Client de test (mockHaloClient) et chunk de film v41 versionné : la simulation tourne en CI.
// Le faux serveur HTTP (halotest) ne convient pas ici, ses fichiers (chunks, JSON d'API) étant
// exclus du dépôt par .gitignore.

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"slices"
	"testing"

	"levelup/go-api/internal/migration"
)

const (
	e2eMatchID = "aabbccdd-0000-4000-8000-0000000000e2"
	e2eOwner   = "0000000000000000" // Player0, équipe 0 (makeMatchJSON : équipe = indice % 2)
	e2eFriend  = "0000000000000002" // Player2, même équipe que le propriétaire
)

type psEnv struct {
	e      *SyncEngine
	client *mockHaloClient
	shared *sql.DB
	player *sql.DB
}

func newPSEnv(t *testing.T, amis []string) *psEnv {
	t.Helper()
	shared := openBatchPathTestDB(t, migration.TargetShared)
	player := openBatchPathTestDB(t, migration.TargetPlayer)
	meta := openFixtureDB(t, buildMetadataDDL())
	(&pipelineFixture{metadata: meta}).insertMetadataData(t)
	// Une citation par médaille du propriétaire (Player0 : médailles 100 et 200).
	mustExec(t, meta, `INSERT INTO citation_mappings (citation_name_norm, citation_name_display, mapping_type, medal_id, enabled)
		VALUES ('e2e_medal', 'Medaille E2E', 'medal', 100, TRUE)`)
	// Le compte ami est connu de la base partagée (matchs précédents) : c'est ce que le
	// premier écrit résout.
	mustExec(t, shared, `INSERT INTO xuid_aliases (xuid, gamertag) VALUES (?, 'Player2')`, e2eFriend)

	client := &mockHaloClient{
		statsBody:             map[string]map[string]any{e2eMatchID: makeMatchJSON(e2eMatchID, 8)},
		highlightChunkData:    loadHighlightFixture(t),
		highlightChunkVersion: 41,
		highlightChunkFound:   true,
	}
	e := &SyncEngine{gamertag: "Player0", xuid: e2eOwner, titleSlug: "halo_infinite", metaDB: meta}
	if amis != nil {
		e.friendsLoader = func() ([]string, error) { return amis, nil }
	}
	return &psEnv{e: e, client: client, shared: shared, player: player}
}

// psFinal : valeurs du match lues en base, comparables entre deux versions du code. Les
// horodatages sont exclus (non déterministes) ; leur présence est un booléen.
type psFinal struct {
	IsWithFriends    bool             `json:"is_with_friends"`
	DominanceFlag    *int64           `json:"dominance_flag"`
	SessionPresent   bool             `json:"session_present"`
	PerformanceSet   bool             `json:"performance_score_set"`
	SnapshotReady    bool             `json:"snapshot_ready_at_set"`
	PartialReasons   *string          `json:"partial_reasons"`
	LUSRRows         []string         `json:"lusr_rows"`
	Citations        map[string]int64 `json:"citations"`
	EventsLoaded     bool             `json:"events_loaded"`
	HighlightEvents  int              `json:"highlight_events"`
	KillEventsLatest int              `json:"kill_events_latest"`
	Participants     int              `json:"participants"`
}

func (env *psEnv) lireFinal(t *testing.T) psFinal {
	t.Helper()
	var f psFinal
	var dom sql.NullInt64
	var perf sql.NullFloat64
	var sess, partial sql.NullString
	var snap sql.NullTime
	if err := env.player.QueryRow(`SELECT COALESCE(is_with_friends, FALSE), dominance_flag, session_id, performance_score, snapshot_ready_at, partial_reasons
		FROM player_match_enrichment_latest WHERE match_id = ?`, e2eMatchID).Scan(&f.IsWithFriends, &dom, &sess, &perf, &snap, &partial); err != nil {
		t.Fatalf("lecture enrichissement: %v", err)
	}
	if dom.Valid {
		v := dom.Int64
		f.DominanceFlag = &v
	}
	f.SessionPresent, f.PerformanceSet, f.SnapshotReady = sess.Valid && sess.String != "", perf.Valid, snap.Valid
	if partial.Valid {
		f.PartialReasons = &partial.String
	}
	rows, err := env.player.Query(`SELECT rating_type, COALESCE(CAST(ROUND(rating_value, 6) AS VARCHAR), 'NULL'), COALESCE(tier_label, '')
		FROM match_skill_rank_latest WHERE match_id = ? ORDER BY rating_type`, e2eMatchID)
	if err != nil {
		t.Fatalf("lecture rating: %v", err)
	}
	for rows.Next() {
		var rt, rv, tl string
		if err := rows.Scan(&rt, &rv, &tl); err != nil {
			t.Fatalf("scan rating: %v", err)
		}
		f.LUSRRows = append(f.LUSRRows, fmt.Sprintf("%s=%s/%s", rt, rv, tl))
	}
	_ = rows.Close()
	f.Citations = map[string]int64{}
	crows, err := env.player.Query(`SELECT citation_name_norm, value FROM match_citations_latest WHERE match_id = ?`, e2eMatchID)
	if err != nil {
		t.Fatalf("lecture citations: %v", err)
	}
	for crows.Next() {
		var n string
		var v int64
		if err := crows.Scan(&n, &v); err != nil {
			t.Fatalf("scan citations: %v", err)
		}
		f.Citations[n] = v
	}
	_ = crows.Close()
	_ = env.shared.QueryRow(`SELECT COALESCE(events_loaded, FALSE) FROM match_registry WHERE match_id = ?`, e2eMatchID).Scan(&f.EventsLoaded)
	_ = env.shared.QueryRow(`SELECT COUNT(*) FROM highlight_events WHERE match_id = ?`, e2eMatchID).Scan(&f.HighlightEvents)
	_ = env.shared.QueryRow(`SELECT COUNT(*) FROM match_kill_events_latest WHERE match_id = ?`, e2eMatchID).Scan(&f.KillEventsLatest)
	_ = env.shared.QueryRow(`SELECT COUNT(*) FROM match_participants WHERE match_id = ?`, e2eMatchID).Scan(&f.Participants)
	return f
}

// jouer exécute le premier écrit puis le post-sync complet. Rend l'état lu ENTRE les deux et
// l'ordre des étapes du post-sync.
func (env *psEnv) jouer(t *testing.T) (avant psFinal, ordre []string) {
	t.Helper()
	ctx := context.Background()
	result, n := passePremierEcrit(ctx, env.e, env.client, env.shared, env.player, []string{e2eMatchID})
	if n != 1 || len(result.InsertedMatchIDs) != 1 {
		t.Fatalf("premier écrit: %d persisté(s), insérés=%v", n, result.InsertedMatchIDs)
	}
	avant = env.lireFinal(t)
	r := env.e.runPostSyncPipeline(ctx, env.player, NewPinnedSharedAccess(env.shared), env.client, result.InsertedMatchIDs)
	return avant, etapesDuResultat(r)
}

// TestPostSyncE2E_SquadMatch (2b.1) : avec un coéquipier réel dans la liste d'amis, la ligne lue
// par la vue _latest vaut is_with_friends = TRUE APRÈS le premier écrit et AVANT le post-sync ;
// après le post-sync : LUSR présent, domination calculée, citations présentes, ordre D1.
func TestPostSyncE2E_SquadMatch(t *testing.T) {
	env := newPSEnv(t, []string{"Player2"})
	avant, ordre := env.jouer(t)

	if !avant.IsWithFriends {
		t.Error("après le premier écrit et avant le post-sync: is_with_friends = FALSE, want TRUE")
	}
	apres := env.lireFinal(t)
	if !apres.IsWithFriends {
		t.Error("après le post-sync: is_with_friends = FALSE, want TRUE (recalcul convergent)")
	}
	if len(apres.LUSRRows) == 0 {
		t.Error("aucune ligne de rating (LUSR) après le post-sync")
	}
	if apres.DominanceFlag == nil {
		t.Error("dominance_flag NULL après le post-sync, want une valeur calculée")
	}
	if apres.Citations["e2e_medal"] == 0 {
		t.Errorf("citation e2e_medal absente ou nulle: %v", apres.Citations)
	}
	if !apres.EventsLoaded || apres.HighlightEvents == 0 {
		t.Errorf("événements du film non chargés: loaded=%v n=%d", apres.EventsLoaded, apres.HighlightEvents)
	}
	if !slices.Equal(ordre, ordrePostSyncAttendu) {
		t.Errorf("ordre des étapes\n got  %v\n want %v", ordre, ordrePostSyncAttendu)
	}
}

// TestPostSyncE2E_NoFriendList (2b.2) : sans liste d'amis, is_with_friends reste FALSE au premier
// écrit ; le reste du post-sync est identique au cas de l'escouade.
func TestPostSyncE2E_NoFriendList(t *testing.T) {
	squad := newPSEnv(t, []string{"Player2"})
	_, _ = squad.jouer(t)
	sq := squad.lireFinal(t)

	env := newPSEnv(t, nil)
	avant, _ := env.jouer(t)
	apres := env.lireFinal(t)

	if avant.IsWithFriends || apres.IsWithFriends {
		t.Errorf("sans liste d'amis: is_with_friends avant=%v après=%v, want FALSE et FALSE", avant.IsWithFriends, apres.IsWithFriends)
	}
	sq.IsWithFriends = false
	if a, b := mustJSON(t, sq), mustJSON(t, apres); a != b {
		t.Errorf("hors is_with_friends, le post-sync diffère entre escouade et sans liste\n escouade: %s\n sans liste: %s", a, b)
	}
}

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("json: %v", err)
	}
	return string(b)
}
