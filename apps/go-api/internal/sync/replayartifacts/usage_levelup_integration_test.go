//go:build integration

package replayartifacts

// usage_levelup_integration_test.go — LES MESURES DE LEVELUP DE BOUT EN BOUT : artefact, fil des
// morts et équipes en base -> lecture avant tout segment d'écriture -> projection -> colonnes
// `shield_breaks`, `shield_recharges`, `revenge_kills` et `levelup_rev` servies par les vues
// `_latest`. Un match sans fil des morts garde ses vengeances NULL, bouclier mesuré.

import (
	"context"
	"database/sql"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/replay"
)

// artefactCycle : le joueur 111 casse son bouclier (image 10) puis le recharge (image 30) ;
// 222 n'a aucune mesure. Les deux ont une piste.
func artefactCycle(t *testing.T, dir, matchID string) string {
	t.Helper()
	origin := int64(0)
	zero, plein := float32(0), float32(1)
	pts := make([]replay.Point, 0, 100)
	for i := 0; i < 100; i++ {
		p := replay.Point{T: i}
		switch i {
		case 5:
			p.Sh = &plein
		case 10:
			p.Sh = &zero
		case 30:
			p.Sh = &plein
		}
		pts = append(pts, p)
	}
	doc := replay.ReplayDocument{
		SchemaVersion: replay.SchemaVersion, MatchID: matchID,
		FrameIntervalMS: 100, FrameCount: 100, DurationMS: 10000, OriginMs: &origin,
		Tracks: []replay.Track{
			{Slot: 1, XUID: "111", Points: pts},
			{Slot: 2, XUID: "222", Points: []replay.Point{{T: 0}}},
		},
	}
	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("marshal artefact: %v", err)
	}
	path := filepath.Join(dir, matchID+".json.gz")
	if err := poserCompresse(path, raw, 0o644); err != nil {
		t.Fatalf("write artefact: %v", err)
	}
	return path
}

func insererEquipe(t *testing.T, db *sql.DB, matchID, xuid string, team int) {
	t.Helper()
	if _, err := db.Exec(`INSERT INTO match_participants (match_id, xuid, team_id) VALUES (?, ?, ?)`,
		matchID, xuid, team); err != nil {
		t.Fatalf("insert participant %s/%s: %v", matchID, xuid, err)
	}
}

func TestResumeUsage_MesuresDeLevelUp(t *testing.T) {
	db := baseRegistre(t)
	dir := t.TempDir()
	acquis, relaches := 0, 0
	d := depsUsage(t, db, "halo_infinite", &acquis, &relaches)

	// m-lu : 222 tue 111, puis 111 tue 222 (111 se venge) ; 111 a un cycle de bouclier.
	insererEquipe(t, db, "m-lu", "111", 0)
	insererEquipe(t, db, "m-lu", "222", 1)
	insererFrag(t, db, "m-lu", "222", "111", 1000)
	insererFrag(t, db, "m-lu", "111", "222", 2000)
	// m-sans-fil : bouclier mesure, aucun fil des morts.
	insererEquipe(t, db, "m-sans-fil", "111", 0)
	insererEquipe(t, db, "m-sans-fil", "222", 1)

	lot := lus(d,
		ArtefactRange{MatchID: "m-lu", Path: artefactCycle(t, dir, "m-lu")},
		ArtefactRange{MatchID: "m-sans-fil", Path: artefactCycle(t, dir, "m-sans-fil")},
	)
	preparerFragsDuResume(context.Background(), d, lot)
	persisterResumesUsage(context.Background(), d, &bilanDerivations{}, lot)

	lire := func(match, xuid string) (b, r, v sql.NullInt64) {
		t.Helper()
		if err := db.QueryRow(`SELECT shield_breaks, shield_recharges, revenge_kills
			FROM match_usage_players_latest WHERE match_id = ? AND xuid = ?`, match, xuid).Scan(&b, &r, &v); err != nil {
			t.Fatalf("lecture %s/%s: %v", match, xuid, err)
		}
		return b, r, v
	}
	b, r, v := lire("m-lu", "111")
	if !b.Valid || b.Int64 != 1 || !r.Valid || r.Int64 != 1 || !v.Valid || v.Int64 != 1 {
		t.Errorf("m-lu/111 = %+v %+v %+v, attendu 1 bris / 1 recharge / 1 vengeance", b, r, v)
	}
	// 222 est a zero partout et n'avait aucune ligne de resume : il n'en recoit pas (il se lit zero
	// parce que son match est mesure par la ligne de 111).
	var lignes222 int
	if err := db.QueryRow(`SELECT count(*) FROM match_usage_players_latest
		WHERE match_id = 'm-lu' AND xuid = '222'`).Scan(&lignes222); err != nil {
		t.Fatalf("lecture m-lu/222: %v", err)
	}
	if lignes222 != 0 {
		t.Errorf("m-lu/222 : %d ligne(s), attendu aucune (zero partout, pas de ligne ajoutee)", lignes222)
	}
	b, r, v = lire("m-sans-fil", "111")
	if !b.Valid || b.Int64 != 1 || v.Valid {
		t.Errorf("m-sans-fil/111 = %+v %+v %+v, attendu bouclier mesure et vengeance NULL (fil des morts absent)", b, r, v)
	}

	// Révision : posée sur le match mesuré en entier, laissée VIDE sur celui dont le fil des morts n'a
	// pas été lu (le rattrapage le reprendra, il ne le croit pas à jour).
	for id, attendue := range map[string]string{"m-lu": UsageLevelUpRev, "m-sans-fil": ""} {
		var rev sql.NullString
		if err := db.QueryRow(`SELECT levelup_rev FROM match_usage_films_latest WHERE match_id = ?`, id).Scan(&rev); err != nil {
			t.Fatalf("levelup_rev %s: %v", id, err)
		}
		if rev.String != attendue {
			t.Errorf("%s : levelup_rev = %+v, attendu %q", id, rev, attendue)
		}
	}
}

// Deux frags au même instant se lisent dans le même ordre quel que soit l'ordre dans lequel ils ont été
// écrits : le tri n'a pas de départage caché dans l'ordre d'insertion.
func TestLireFragsDuFilDesMorts_OrdreDeterministeALegalite(t *testing.T) {
	db := baseRegistre(t)
	insererFrag(t, db, "m-ab", "A", "B", 5000)
	insererFrag(t, db, "m-ab", "C", "D", 5000)
	insererFrag(t, db, "m-ba", "C", "D", 5000)
	insererFrag(t, db, "m-ba", "A", "B", 5000)

	frags, err := LireFragsDuFilDesMorts(context.Background(), db, []string{"m-ab", "m-ba"})
	if err != nil {
		t.Fatalf("LireFragsDuFilDesMorts: %v", err)
	}
	ab, ba := frags["m-ab"].Kills, frags["m-ba"].Kills
	if len(ab) != 2 || len(ba) != 2 {
		t.Fatalf("frags lus : %d et %d, attendu 2 et 2", len(ab), len(ba))
	}
	for i := range ab {
		if ab[i] != ba[i] {
			t.Fatalf("ordre dépendant de l'insertion : %+v contre %+v", ab, ba)
		}
	}
}
