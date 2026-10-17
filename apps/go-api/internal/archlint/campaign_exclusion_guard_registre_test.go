package archlint

import "testing"

// TestEstLecteurCampagne_RegistreSurFenetreDeTemps : le critère de D-5 voit aussi le registre lu
// SEUL sur une fenêtre de temps (le filtre joueur vient d'ailleurs, par match_id), et ne prend pas
// un `<>` ou une jointure pour une borne (E5.6 du plan backlog 2026-10-16).
func TestEstLecteurCampagne_RegistreSurFenetreDeTemps(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		nom, sql string
		want     bool
	}{
		{"registre, bornes >= et <=",
			"SELECT mr.match_id FROM match_registry mr WHERE ts >= ? AND ts <= ?", true},
		{"registre, borne stricte",
			"SELECT match_id FROM match_registry WHERE start_time > ?", true},
		{"vue du registre, BETWEEN",
			"SELECT match_id FROM v_match_full WHERE start_time BETWEEN ? AND ?", true},
		{"registre, différent de (pas une borne)",
			"SELECT r.match_id FROM match_registry r WHERE r.read_path <> ?", false},
		{"registre, égalité (pas une fenêtre)",
			"SELECT match_id FROM match_registry WHERE playlist_id = ?", false},
		{"registre sans paramètre",
			"SELECT COUNT(*) FROM match_registry WHERE start_time >= '2026-01-01'", false},
		{"lecteur joueur d'origine",
			"SELECT match_id FROM match_participants WHERE xuid = ?", true},
	} {
		if got := estLecteurCampagne(c.sql); got != c.want {
			t.Errorf("%s : estLecteurCampagne = %v, want %v", c.nom, got, c.want)
		}
	}
}
