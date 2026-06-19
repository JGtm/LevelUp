package sync

// contract_cycle_v1_test.go — invariants contractuels d'un cycle de persist,
// testés contre le chemin V1 réel (processMatch + loadKnownMatchIDs) avec des
// DB DuckDB in-memory. Active les invariants laissés en t.Skip dans
// contract_test.go (scaffold de la suite V2 jamais livrée) sur l'implémentation
// V1 qui tourne en prod.
//
// Dataset hétérogène PVP + PVE (Firefight) conformément à la règle « datasets
// réalistes » : le match PVE (GameVariantCategory 22) exerce le chemin
// is_firefight=true du registry, distinct du PVP (Slayer, catégorie 9).

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
)

// makeFirefightMatchJSON dérive un match PVE (Firefight) de makeMatchJSON :
// GameVariantCategory 22 (Firefight Arcade) — reconnu par isFirefightMatch.
func makeFirefightMatchJSON(matchID string, players int) map[string]any {
	m := makeMatchJSON(matchID, players)
	mi := m["MatchInfo"].(map[string]any)
	mi["GameVariantCategory"] = float64(22)
	mi["UgcGameVariant"] = map[string]any{"AssetId": "gv-ff", "PublicName": "Firefight"}
	mi["PlaylistExperience"] = "Firefight"
	return m
}

// dupCount retourne le nombre de groupes de PK ayant > 1 ligne (0 = PK saine).
func dupCount(t *testing.T, db *sql.DB, table, pkCols string) int {
	t.Helper()
	var n int
	q := "SELECT COUNT(*) FROM (SELECT " + pkCols + " FROM " + table +
		" GROUP BY " + pkCols + " HAVING COUNT(*) > 1)"
	if err := db.QueryRow(q).Scan(&n); err != nil {
		t.Fatalf("dupCount(%s): %v", table, err)
	}
	return n
}

// failingStatsClient enrobe mockHaloClient pour faire échouer GetMatchStats sur
// un match précis (simulation d'une 500 API ciblée). L'embedding promeut les
// autres méthodes HaloClient.
type failingStatsClient struct {
	*mockHaloClient
	failMatch string
}

func (c *failingStatsClient) GetMatchStats(ctx context.Context, matchID string) (map[string]any, error) {
	if matchID == c.failMatch {
		return nil, errors.New("simulated API 500")
	}
	return c.mockHaloClient.GetMatchStats(ctx, matchID)
}

// TestContract_NoDuplicateRows_V1 : rejouer le même dataset (PVP + PVE) ne doit
// JAMAIS créer de doublon de PK sur les tables critiques — anti-régression
// directe du pattern ART (ON CONFLICT DO UPDATE / ré-INSERT, ADR 0019).
func TestContract_NoDuplicateRows_V1(t *testing.T) {
	playerDB, sharedDB := newInMemoryDBs(t)
	e := &SyncEngine{gamertag: "P0", xuid: "0000000000000000"}
	opts := domain.DefaultSyncOptions()

	pvp := "aabbccdd-0000-4000-8000-000000000001"
	pve := "aabbccdd-0000-4000-8000-000000000002"
	mock := &mockHaloClient{statsBody: map[string]map[string]any{
		pvp: makeMatchJSON(pvp, 4),
		pve: makeFirefightMatchJSON(pve, 4),
	}}

	// 2 cycles : le 2e simule un re-sync (delta qui retraite les mêmes matchs).
	for cycle := 0; cycle < 2; cycle++ {
		for _, mid := range []string{pvp, pve} {
			res := domain.SyncResult{StartedAt: time.Now()}
			if err := e.processMatch(context.Background(), mock, sharedDB, playerDB, &res, mid, opts); err != nil {
				t.Fatalf("processMatch(%s) cycle %d: %v", mid, cycle, err)
			}
		}
	}

	if d := dupCount(t, sharedDB, "match_registry", "match_id"); d != 0 {
		t.Errorf("match_registry: %d match_id dupliqués (PK violée → pattern ART)", d)
	}
	if d := dupCount(t, sharedDB, "match_participants", "match_id, xuid"); d != 0 {
		t.Errorf("match_participants: %d (match_id,xuid) dupliqués", d)
	}
	if d := dupCount(t, playerDB, "player_match_enrichment", "match_id"); d != 0 {
		t.Errorf("player_match_enrichment: %d match_id dupliqués", d)
	}
	if n := countRows(t, sharedDB, "match_registry"); n != 2 {
		t.Errorf("match_registry: attendu 2 (PVP+PVE), obtenu %d", n)
	}
}

// TestContract_CycleIdempotent_V1 : rejouer le même cycle immédiatement produit
// le même état DB (aucune ligne en plus, aucune erreur PK).
func TestContract_CycleIdempotent_V1(t *testing.T) {
	playerDB, sharedDB := newInMemoryDBs(t)
	e := &SyncEngine{gamertag: "P0", xuid: "0000000000000000"}
	opts := domain.DefaultSyncOptions()
	mid := "aabbccdd-0000-4000-8000-000000000010"
	mock := &mockHaloClient{statsBody: map[string]map[string]any{mid: makeMatchJSON(mid, 4)}}

	run := func(pass int) {
		res := domain.SyncResult{StartedAt: time.Now()}
		if err := e.processMatch(context.Background(), mock, sharedDB, playerDB, &res, mid, opts); err != nil {
			t.Fatalf("processMatch pass %d: %v", pass, err)
		}
	}
	run(1)
	reg1 := countRows(t, sharedDB, "match_registry")
	part1 := countRows(t, sharedDB, "match_participants")
	enr1 := countRows(t, playerDB, "player_match_enrichment")
	run(2) // rejoue à l'identique
	if reg2, part2, enr2 := countRows(t, sharedDB, "match_registry"),
		countRows(t, sharedDB, "match_participants"),
		countRows(t, playerDB, "player_match_enrichment"); reg2 != reg1 || part2 != part1 || enr2 != enr1 {
		t.Errorf("cycle non-idempotent: registry %d→%d, participants %d→%d, enrichment %d→%d",
			reg1, reg2, part1, part2, enr1, enr2)
	}
}

// TestContract_CrossPlayerDedup_V1 : un match partagé par P0 et P1, une fois
// synchronisé par P0, écrit X0 ET X1 dans shared.match_participants.
// loadKnownMatchIDs(X1) doit alors le retourner (source 2) — donc P1 SKIP
// l'appel API. C'est l'invariant qui justifie la dédup cross-player (incident
// 14 jours mai 2026 : sans cette reconnaissance, P1 refait l'appel).
func TestContract_CrossPlayerDedup_V1(t *testing.T) {
	ctx := context.Background()
	sharedDB := openMemDB(t)
	if err := EnsureSharedSchema(ctx, sharedDB); err != nil {
		t.Fatalf("EnsureSharedSchema: %v", err)
	}
	playerDB0 := openMemDB(t)
	playerDB1 := openMemDB(t)
	for _, pdb := range []*sql.DB{playerDB0, playerDB1} {
		if err := EnsurePlayerSchema(ctx, pdb); err != nil {
			t.Fatalf("EnsurePlayerSchema: %v", err)
		}
	}

	e := &SyncEngine{gamertag: "P0", xuid: "0000000000000000"}
	opts := domain.DefaultSyncOptions()
	mid := "aabbccdd-0000-4000-8000-000000000020"
	mock := &mockHaloClient{statsBody: map[string]map[string]any{mid: makeMatchJSON(mid, 2)}}
	res := domain.SyncResult{StartedAt: time.Now()}
	if err := e.processMatch(ctx, mock, sharedDB, playerDB0, &res, mid, opts); err != nil {
		t.Fatalf("processMatch P0: %v", err)
	}

	// P1 (xuid 0000000000000001) a une DB joueur vierge mais partage sharedDB.
	known, err := loadKnownMatchIDs(ctx, playerDB1, sharedDB, "0000000000000001")
	if err != nil {
		t.Fatalf("loadKnownMatchIDs: %v", err)
	}
	if !known[mid] {
		t.Errorf("cross-player dedup cassé: M=%s absent du known set de P1 → P1 referait l'appel API (incident 14j)", mid)
	}
	// Contrôle négatif : un xuid non-participant ne doit pas connaître M.
	if other, _ := loadKnownMatchIDs(ctx, playerDB1, sharedDB, "0000000000009999"); other[mid] {
		t.Error("faux positif: un xuid non-participant ne doit pas voir M comme connu")
	}
}

// TestContract_PartialFailureIsolation_V1 : un match dont le fetch API échoue ne
// doit (a) laisser aucune ligne partielle, (b) ni empêcher le traitement d'un
// match valide suivant.
func TestContract_PartialFailureIsolation_V1(t *testing.T) {
	playerDB, sharedDB := newInMemoryDBs(t)
	e := &SyncEngine{gamertag: "P0", xuid: "0000000000000000"}
	opts := domain.DefaultSyncOptions()

	bad := "aabbccdd-0000-4000-8000-000000000030"
	good := "aabbccdd-0000-4000-8000-000000000031"
	mock := &mockHaloClient{statsBody: map[string]map[string]any{good: makeMatchJSON(good, 4)}}
	client := &failingStatsClient{mockHaloClient: mock, failMatch: bad}

	resBad := domain.SyncResult{StartedAt: time.Now()}
	if err := e.processMatch(context.Background(), client, sharedDB, playerDB, &resBad, bad, opts); err == nil {
		t.Fatal("attendu une erreur pour le match en échec (fetch API 500)")
	}
	if n := countRows(t, sharedDB, "match_registry"); n != 0 {
		t.Errorf("l'échec API a laissé %d ligne(s) registry — écriture partielle", n)
	}

	resGood := domain.SyncResult{StartedAt: time.Now()}
	if err := e.processMatch(context.Background(), client, sharedDB, playerDB, &resGood, good, opts); err != nil {
		t.Fatalf("le match valide doit réussir malgré l'échec précédent (isolation): %v", err)
	}
	if n := countRows(t, sharedDB, "match_registry"); n != 1 {
		t.Errorf("match_registry: attendu 1 (le bon match), obtenu %d", n)
	}
}
