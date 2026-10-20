package v2

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/persist"
)

// statsEquipe construit un payload GetMatchStats minimal : 4 joueurs, indice pair = équipe 0.
func statsEquipe(matchID string, xuids []string) map[string]any {
	players := make([]any, 0, len(xuids))
	for i, x := range xuids {
		players = append(players, map[string]any{
			"PlayerId": fmt.Sprintf("xuid(%s)", x), "PlayerName": fmt.Sprintf("P%d", i),
			"LastTeamId": float64(i % 2), "Outcome": float64(2), "Rank": float64(i + 1),
			"ParticipationInfo": map[string]any{"TimePlayed": "PT10M0S"},
			"PlayerTeamStats": []any{map[string]any{"Stats": map[string]any{"CoreStats": map[string]any{
				"Kills": float64(10), "Deaths": float64(5), "Assists": float64(3),
			}}}},
		})
	}
	return map[string]any{
		"MatchId": matchID,
		"MatchInfo": map[string]any{
			"StartTime": "2025-01-15T14:30:00Z", "EndTime": "2025-01-15T14:40:00Z",
			"MapVariant":          map[string]any{"AssetId": "map-1", "PublicName": "Aquarius"},
			"GameVariantCategory": float64(9), "PlaylistExperience": "Arena:Slayer",
			"Playlist":       map[string]any{"AssetId": "pl-1", "PublicName": "Arena"},
			"UgcGameVariant": map[string]any{"AssetId": "gv-1", "PublicName": "Slayer"},
			"Duration":       "PT10M0S", "PlayableDuration": "PT9M30S", "LifecycleMode": float64(3),
		},
		"Players": players,
	}
}

// TestPersistCycle_PosesIsWithFriendsOnFirstWrite : le persister de cycle V2 pose
// is_with_friends sur la ligne d'enrichissement de CHAQUE match, pour le joueur qui l'a récupéré
// (amis résolus une fois, sur l'emprunt de la base partagée) ; sans source d'amis, rien n'est posé.
func TestPersistCycle_PosesIsWithFriendsOnFirstWrite(t *testing.T) {
	const owner, mate, foe = "1000000000000001", "1000000000000003", "1000000000000002"
	// Équipe 0 : owner (indice 0) et mate (indice 2) ; équipe 1 : foe, x4.
	xuids := []string{owner, foe, mate, "1000000000000004"}

	shared, err := sql.Open("duckdb", "")
	if err != nil {
		t.Fatalf("open shared: %v", err)
	}
	t.Cleanup(func() { _ = shared.Close() })
	if _, err := shared.Exec(`CREATE TABLE xuid_aliases (xuid VARCHAR, gamertag VARCHAR);
		INSERT INTO xuid_aliases VALUES ('` + mate + `', 'Mate'), ('` + foe + `', 'Foe')`); err != nil {
		t.Fatalf("seed aliases: %v", err)
	}
	// Le getter en cache rend une base VIDE : lire par lui ne résoudrait aucun ami. Seul l'emprunt
	// (compté, avec son release) sert la résolution.
	enCache, err := sql.Open("duckdb", "")
	if err != nil {
		t.Fatalf("open cache: %v", err)
	}
	t.Cleanup(func() { _ = enCache.Close() })
	var emprunts, liberes int
	borrow := func(context.Context) (*sql.DB, func(), error) {
		emprunts++
		return shared, func() { liberes++ }, nil
	}

	run := func(t *testing.T, amis []string) *bool {
		t.Helper()
		q, err := persist.NewBatchQueue(persist.BatchQueueConfig{WALDir: t.TempDir(), ChanBufSize: 10})
		if err != nil {
			t.Fatalf("queue: %v", err)
		}
		t.Cleanup(func() { _ = q.Close() })
		got := make(chan *bool, 1)
		go func() {
			for b := range q.Channel() {
				got <- b.PlayerData.Enrichment.IsWithFriends
				_ = q.ACK(b.BatchID)
				q.RecordPersistResult(true)
			}
		}()
		p := NewCycleBatchPersister("halo_infinite", q, 5*time.Second, nil, nil, func() *sql.DB { return enCache })
		if amis != nil {
			p = WithFriendsSource(p, func(x string) ([]string, error) {
				if x != owner {
					return nil, fmt.Errorf("amis demandés pour %s, want %s", x, owner)
				}
				return amis, nil
			}, borrow)
		}
		err = p.PersistCycle(context.Background(), CycleBatch{
			CycleID: "c1",
			Matches: map[string]SharedMatchData{"m-friends": {MatchID: "m-friends", Fetcher: "owner", Stats: statsEquipe("m-friends", xuids)}},
			PlayerBySlug: map[string]PlayerProfile{
				"owner": {Gamertag: "Owner", XUID: owner, PlayerSlug: "owner"},
			},
		})
		if err != nil {
			t.Fatalf("PersistCycle: %v", err)
		}
		select {
		case v := <-got:
			return v
		case <-time.After(5 * time.Second):
			t.Fatal("aucun lot reçu de la file")
			return nil
		}
	}

	if v := run(t, []string{"Mate"}); v == nil || !*v {
		t.Errorf("ami coéquipier: IsWithFriends = %v, want TRUE", v)
	}
	if v := run(t, []string{"Foe"}); v == nil || *v {
		t.Errorf("ami adverse: IsWithFriends = %v, want FALSE explicite", v)
	}
	if v := run(t, nil); v != nil {
		t.Errorf("sans source d'amis: IsWithFriends = %v, want non posé", *v)
	}
	if emprunts != 2 || liberes != emprunts {
		t.Errorf("emprunts de la base partagée = %d, libérés = %d, want 2 et 2 (un emprunt par cycle avec source d'amis, toujours relâché)", emprunts, liberes)
	}
}
