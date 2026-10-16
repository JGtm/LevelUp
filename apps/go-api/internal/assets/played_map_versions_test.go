package assets

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"
)

// countingLoader sert des couples par titre et compte ses lectures.
type countingLoader struct {
	pairs map[string][][2]string
	err   error
	loads int
}

func (l *countingLoader) load(_ context.Context, titleID string, visit func(string, string)) error {
	l.loads++
	if l.err != nil {
		return l.err
	}
	for _, p := range l.pairs[titleID] {
		visit(p[0], p[1])
	}
	return nil
}

// Une absence relit la base au plus une fois par intervalle : N versions inventées coûtent
// une lecture, pas N.
func TestPlayedMapVersions_InventedVersionsDoNotReloadEachTime(t *testing.T) {
	l := &countingLoader{pairs: map[string][][2]string{"halo_infinite": {{"map-a", "v-1"}}}}
	p := NewPlayedMapVersions(l.load)
	ctx := context.Background()
	if ok, err := p.Played(ctx, "halo_infinite", "map-a", "v-1"); !ok || err != nil {
		t.Fatalf("couple joué : %v, %v", ok, err)
	}
	for _, v := range []string{"v-x", "v-y", "v-z"} {
		if ok, err := p.Played(ctx, "halo_infinite", "map-a", v); ok || err != nil {
			t.Fatalf("version inventée %s : %v, %v", v, ok, err)
		}
	}
	if ok, _ := p.Played(ctx, "halo_infinite", "map-b", "v-1"); ok {
		t.Fatal("version jouée sur une autre carte reconnue")
	}
	if l.loads != 1 {
		t.Fatalf("%d lecture(s) de la base, attendu 1", l.loads)
	}
}

// Passé l'intervalle, une absence relit la base : un match synchronisé depuis est vu.
func TestPlayedMapVersions_ReloadsAfterTheInterval(t *testing.T) {
	l := &countingLoader{pairs: map[string][][2]string{}}
	p := NewPlayedMapVersions(l.load)
	p.minReload = time.Millisecond
	ctx := context.Background()
	if ok, _ := p.Played(ctx, "halo_infinite", "map-a", "v-2"); ok {
		t.Fatal("version pas encore jouée reconnue")
	}
	l.pairs["halo_infinite"] = [][2]string{{"map-a", "v-2"}}
	time.Sleep(5 * time.Millisecond)
	if ok, err := p.Played(ctx, "halo_infinite", "map-a", "v-2"); !ok || err != nil {
		t.Fatalf("match synchronisé depuis : %v, %v", ok, err)
	}
}

// Les titres sont séparés : un couple joué dans un titre ne l'est pas dans un autre.
func TestPlayedMapVersions_TitlesAreSeparate(t *testing.T) {
	l := &countingLoader{pairs: map[string][][2]string{"halo_infinite": {{"map-a", "v-1"}}}}
	p := NewPlayedMapVersions(l.load)
	if ok, _ := p.Played(context.Background(), "halo_5", "map-a", "v-1"); ok {
		t.Fatal("couple d'un autre titre reconnu")
	}
}

// Une lecture en échec rend l'erreur, sans relire avant l'intervalle.
func TestPlayedMapVersions_ReadErrorIsReturned(t *testing.T) {
	boom := errors.New("base partagée en bascule")
	l := &countingLoader{err: boom}
	p := NewPlayedMapVersions(l.load)
	for range 3 {
		if ok, err := p.Played(context.Background(), "halo_infinite", "map-a", "v-1"); ok || !errors.Is(err, boom) {
			t.Fatalf("Played = %v, %v ; attendu false, %v", ok, err, boom)
		}
	}
	if l.loads != 1 {
		t.Fatalf("%d lecture(s), attendu 1", l.loads)
	}
}

// Avec les versions jouées, le catalogue reconnaît une version jouée (carte au catalogue ou
// non) ; une version jamais jouée reste invérifiable pour une carte connue et inconnue pour une
// carte inconnue ; une base partagée illisible ne fait ni connaître ni mémoriser un refus.
func TestMetadataCatalog_ClassifyPlayedMapVersion(t *testing.T) {
	store := NewDuckDBIndexStore(filepath.Join(t.TempDir(), "metadata.duckdb"))
	t.Cleanup(store.Close)
	db, err := store.openDB()
	if err != nil {
		t.Fatalf("openDB: %v", err)
	}
	for _, stmt := range []string{
		`CREATE TABLE maps_catalog (title_slug VARCHAR, map_asset_id VARCHAR, current_version_id VARCHAR)`,
		`INSERT INTO maps_catalog VALUES ('halo_infinite', 'map-known', 'v-current')`,
	} {
		if _, err := db.Exec(context.Background(), stmt); err != nil {
			t.Fatalf("%s: %v", stmt, err)
		}
	}
	l := &countingLoader{pairs: map[string][][2]string{"halo_infinite": {
		{"map-known", "v-played"}, {"map-new", "v-new"},
	}}}
	cat := NewMetadataCatalog(store, func(id string) bool { return id == "halo_infinite" }).
		WithPlayedMapVersions(NewPlayedMapVersions(l.load))
	cases := []struct {
		name    string
		id, ver string
		want    Verdict
	}{
		{"version courante", "map-known", "v-current", VerdictKnown},
		{"version jouée", "map-known", "v-played", VerdictKnown},
		{"carte hors catalogue, version jouée", "map-new", "v-new", VerdictKnown},
		{"version jamais jouée d'une carte connue", "map-known", "v-invented", VerdictUnverifiable},
		{"version jamais jouée d'une carte inconnue", "map-invented", "v-invented", VerdictUnknown},
		{"carte hors catalogue, sans version", "map-new", "", VerdictUnknown},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ref := Ref{Kind: KindMapImage, TitleID: "halo_infinite", ID: tc.id, Variant: tc.ver}
			if got := cat.Classify(context.Background(), ref); got != tc.want {
				t.Fatalf("Classify(%s) = %v, attendu %v", ref, got, tc.want)
			}
		})
	}

	broken := NewMetadataCatalog(store, func(string) bool { return true }).
		WithPlayedMapVersions(NewPlayedMapVersions((&countingLoader{err: errors.New("illisible")}).load))
	for _, tc := range []struct {
		id   string
		want Verdict
	}{
		{"map-known", VerdictUnverifiable}, // repli sur le jugement du catalogue seul
		{"map-invented", VerdictUnavailable},
	} {
		ref := Ref{Kind: KindMapImage, TitleID: "halo_infinite", ID: tc.id, Variant: "v-any"}
		if got := broken.Classify(context.Background(), ref); got != tc.want {
			t.Errorf("base partagée illisible, Classify(%s) = %v, attendu %v", ref, got, tc.want)
		}
	}
}

// New branche la source des versions jouées dans le catalogue de la porte quand elle est
// fournie, et seulement alors.
func TestNew_WiresPlayedMapVersions(t *testing.T) {
	for _, withSource := range []bool{true, false} {
		dir := t.TempDir()
		cfg := AssetConfig{
			CacheRootDir: filepath.Join(dir, "cache"),
			MetaDBPath:   filepath.Join(dir, "metadata.duckdb"),
			TitleKnown:   func(string) bool { return true },
		}
		if withSource {
			cfg.PlayedMapVersions = (&countingLoader{}).load
		}
		r, err := New(cfg)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = r.Close(context.Background()) })
		cat, ok := r.catalog.(*MetadataCatalog)
		if !ok {
			t.Fatalf("catalogue %T, attendu *MetadataCatalog", r.catalog)
		}
		if got := cat.played != nil; got != withSource {
			t.Errorf("source fournie=%v : versions jouées branchées=%v", withSource, got)
		}
	}
}
