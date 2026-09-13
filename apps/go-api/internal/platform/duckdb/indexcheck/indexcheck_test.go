package indexcheck

// indexcheck_test.go — la RÈGLE et ses invariants, sans DuckDB. Les chemins SQL
// sont exercés sur de vraies bases par les deux consommateurs :
// `internal/scheduler/data_health_msr_index_test.go` (mode échantillon) et
// `cmd/repair_msr_index/diag_test.go` (mode exhaustif, fixture aux migrations).

import (
	"errors"
	"strings"
	"testing"
)

func TestCompareRendLesEcartsEtLesLignesIndexees(t *testing.T) {
	refs := []KeyCount{
		{Key: []string{"h5_arena"}, Count: 1826},
		{Key: []string{"btb"}, Count: 40},
		{Key: []string{"arena_slayer"}, Count: 3},
	}
	indexed := map[string]int{"h5_arena": 22, "btb": 40, "arena_slayer": 9}

	rows, divergences, err := Compare(refs, func(key []string) (int, error) {
		return indexed[key[0]], nil
	})
	if err != nil {
		t.Fatalf("Compare: %v", err)
	}
	if rows != 22+40+9 {
		t.Errorf("IndexedRows = %d, want %d", rows, 22+40+9)
	}
	if len(divergences) != 2 {
		t.Fatalf("écarts = %d, want 2 (déficit + excédent) : %+v", len(divergences), divergences)
	}
	// Seul le DÉFICIT compte en lignes manquantes : l'excédent (arena_slayer, 9
	// servies pour 3 réelles) est une autre pathologie et ne doit pas être masqué
	// par une soustraction négative.
	rep := Report{Divergences: divergences}
	if got := rep.RowsMissing(); got != 1804 {
		t.Errorf("RowsMissing = %d, want 1804 (le seul déficit)", got)
	}
	if rep.OK() {
		t.Error("OK() = true alors que deux clés divergent")
	}
}

func TestCompareRemonteLErreurDeLookup(t *testing.T) {
	boom := errors.New("handle fermé")
	_, _, err := Compare([]KeyCount{{Key: []string{"m1"}, Count: 1}},
		func([]string) (int, error) { return 0, boom })
	if !errors.Is(err, boom) {
		t.Fatalf("erreur attendue %v, obtenue %v", boom, err)
	}
}

func TestIndexesToRebuildUnionOrdonneeEtDedoublonnee(t *testing.T) {
	axes := []Axis{
		{Name: "a", Indexes: []string{"idx_b"}},
		{Name: "b", Indexes: []string{"idx_a", "idx_b"}},
		{Name: "c", Indexes: []string{"idx_c"}},
	}
	ecart := []Divergence{{Key: []string{"k"}, Scanned: 2, Indexed: 1}}

	if got := IndexesToRebuild([]Report{{}, {}, {}}, axes); len(got) != 0 {
		t.Errorf("tous les axes sains : index = %v, want aucun", got)
	}
	got := IndexesToRebuild([]Report{{Divergences: ecart}, {Divergences: ecart}, {}}, axes)
	want := "idx_a,idx_b"
	if strings.Join(got, ",") != want {
		t.Errorf("index = %v, want %s (union ordonnée, sans doublon)", got, want)
	}
}

// TestMatchSkillRankAxesEstUneCopieDefensive — la carte partagée ne doit pas
// pouvoir être modifiée par un appelant : les deux consommateurs la liraient
// alors différemment selon l'ordre d'exécution.
func TestMatchSkillRankAxesEstUneCopieDefensive(t *testing.T) {
	a := MatchSkillRankAxes()
	if len(a) != 3 {
		t.Fatalf("axes = %d, want 3 (un par index posé par la migration)", len(a))
	}
	a[0].Name = "saboté"
	a[0].LookupWhere = "1 = 1"
	b := MatchSkillRankAxes()
	if b[0].Name == "saboté" || b[0].LookupWhere == "1 = 1" {
		t.Fatal("la carte partagée a été modifiée par l'appelant — copie non défensive")
	}
}

// TestReferenceSQLForceLeScanEtBorneLEchantillon — les deux formes de la requête
// de référence. Le scan est FORCÉ par l'expression de clé ; le mode échantillon
// enveloppe le regroupement, donc le tirage porte sur les CLÉS, pas sur les lignes.
func TestReferenceSQLForceLeScanEtBorneLEchantillon(t *testing.T) {
	a := MatchSkillRankAxes()[0]
	opts := Options{Table: MatchSkillRankTable, MaxKeys: 200}

	exhaustif := referenceSQL(a, opts)
	if !strings.Contains(exhaustif, "playlist_group || ''") {
		t.Errorf("le scan n'est pas forcé par une expression : %s", exhaustif)
	}
	if strings.Contains(exhaustif, "USING SAMPLE") {
		t.Errorf("mode exhaustif : aucun tirage attendu : %s", exhaustif)
	}
	if !strings.Contains(exhaustif, "ORDER BY") {
		t.Errorf("mode exhaustif : ordre déterministe attendu (diagnostic reproductible) : %s", exhaustif)
	}

	opts.Sample = true
	echantillon := referenceSQL(a, opts)
	if !strings.Contains(echantillon, "USING SAMPLE 200 ROWS (reservoir)") {
		t.Errorf("mode échantillon : tirage réservoir attendu : %s", echantillon)
	}
	if !strings.Contains(echantillon, "GROUP BY") ||
		strings.Index(echantillon, "GROUP BY") > strings.Index(echantillon, "USING SAMPLE") {
		t.Errorf("le tirage doit ENVELOPPER le regroupement (tirer des clés, pas des lignes) : %s", echantillon)
	}
}
