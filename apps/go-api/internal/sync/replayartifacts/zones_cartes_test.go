package replayartifacts

import (
	"context"
	"database/sql"
	"reflect"
	"testing"
)

// TestCartesJoueesJcJ — la liste du premier passage : une entrée par carte, les plus jouées
// d'abord, sans les matchs Firefight ni les lignes sans map_id ; le nom brut du registre sert de
// candidat quand le catalogue d'assets n'est pas câblé.
func TestCartesJoueesJcJ(t *testing.T) {
	db, err := sql.Open("duckdb", "")
	if err != nil {
		t.Fatalf("duckdb en mémoire : %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec(`CREATE TABLE match_registry (match_id VARCHAR, map_id VARCHAR, map_name VARCHAR, is_firefight BOOLEAN)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO match_registry VALUES
		('m1', 'aaa', 'Lattice - Ranked', FALSE),
		('m2', 'aaa', NULL, NULL),
		('m3', 'bbb', 'Origin', FALSE),
		('m4', 'ccc', 'Vallaheim Firefight', TRUE),
		('m5', NULL, 'Sans carte', FALSE),
		('m6', '', 'Vide', FALSE)`); err != nil {
		t.Fatal(err)
	}
	cartes, err := CartesJoueesJcJ(context.Background(), db, nil)
	if err != nil {
		t.Fatalf("CartesJoueesJcJ : %v", err)
	}
	attendu := []CarteJouee{
		{MapID: "aaa", Noms: []string{"Lattice - Ranked"}, Matchs: 2},
		{MapID: "bbb", Noms: []string{"Origin"}, Matchs: 1},
	}
	if !reflect.DeepEqual(cartes, attendu) {
		t.Fatalf("cartes = %+v\nattendu %+v", cartes, attendu)
	}
}
