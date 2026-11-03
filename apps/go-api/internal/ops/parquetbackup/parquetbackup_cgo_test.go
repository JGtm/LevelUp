//go:build cgo

package parquetbackup

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/dblease"
	"levelup/go-api/internal/platform/duckdb"
)

const testSlug = "halo_infinite"

// fakeToken : un faux jeton construit à l'exécution (aucun littéral de secret dans le dépôt).
func fakeToken() string { return "tok-" + strings.Repeat("a1b2", 8) }

// seedDB crée une base de fichier avec les instructions de stmts.
func seedDB(t *testing.T, path string, stmts ...string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("duckdb", path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			t.Fatalf("%s: %v", s, err)
		}
	}
}

func countRows(t *testing.T, path, table string) int {
	t.Helper()
	db, err := sql.Open("duckdb", path+"?access_mode=read_only")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	var n int
	if err := db.QueryRow("SELECT count(*) FROM " + table).Scan(&n); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	return n
}

func writeText(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

// seedRoot peuple une racine de données : bases partagée, métadonnées, joueur, fichiers de configuration.
func seedRoot(t *testing.T) (*title.PathResolver, string) {
	t.Helper()
	root := t.TempDir()
	pr := title.NewPathResolverRoots(title.Roots{Resources: root, Data: filepath.Join(root, "data"), DataSet: true})
	seedDB(t, pr.SharedDBPath(testSlug),
		`CREATE SEQUENCE seq_ids START 100`,
		`CREATE TABLE match_registry (match_id VARCHAR PRIMARY KEY, id BIGINT DEFAULT nextval('seq_ids'), map_name VARCHAR NOT NULL)`,
		`INSERT INTO match_registry (match_id, map_name) VALUES ('m1','Aquarius'),('m2','Streets'),('m3','Recharge')`,
		`CREATE VIEW v_maps AS SELECT map_name FROM match_registry`)
	seedDB(t, pr.MetadataDBPath(testSlug),
		`CREATE TABLE medal_definitions (medal_id BIGINT, name VARCHAR)`,
		`INSERT INTO medal_definitions VALUES (1,'Killing Spree'),(2,'Double Kill')`)
	seedDB(t, pr.PlayerDBPath(testSlug, "Spartan B"),
		`CREATE TABLE player_match_enrichment (match_id VARCHAR PRIMARY KEY, performance_score DOUBLE)`,
		`INSERT INTO player_match_enrichment VALUES ('m1', 71.5),('m2', 64.0)`)
	authDir := filepath.Join(pr.DataRoot(), "auth")
	writeText(t, pr.DBProfilesPath(), `{"profiles":{}}`)
	writeText(t, pr.AppSettingsPath(), `{"lang":"fr"}`)
	writeText(t, filepath.Join(authDir, "users.json"), `{"users":[]}`)
	writeText(t, filepath.Join(authDir, "groups.json"), `{"groups":[]}`)
	writeText(t, filepath.Join(pr.WatcherTokensDir(), "2533274800000001.json"), `{"oauth_refresh_token":"`+fakeToken()+`"}`)
	return pr, authDir
}

func newOptions(pr *title.PathResolver, authDir, dest string) Options {
	return Options{Paths: pr, AuthDir: authDir, Dest: dest, Keep: 7, AppVersion: "9.9.9-test",
		LeaseWait: 50 * time.Millisecond, RetryWait: 100 * time.Millisecond}
}

func TestRunThenRestore_SameRows(t *testing.T) {
	ctx := context.Background()
	pr, authDir := seedRoot(t)
	dest := t.TempDir()

	m, err := Run(ctx, newOptions(pr, authDir, dest))
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if m.Partial || m.AppVersion != "9.9.9-test" || len(m.Databases) != 3 {
		t.Fatalf("manifeste inattendu : partial=%v version=%s bases=%d", m.Partial, m.AppVersion, len(m.Databases))
	}
	if len(m.Config) != 5 {
		t.Fatalf("fichiers de configuration = %d, want 5 (profils, réglages, comptes, groupes, 1 jeton)", len(m.Config))
	}
	if m.TotalRows != 3+2+2 {
		t.Fatalf("lignes du manifeste = %d, want 7", m.TotalRows)
	}
	setDir := filepath.Join(dest, m.Name)
	if err := m.Verify(setDir); err != nil {
		t.Fatalf("Verify : %v", err)
	}
	if !m.hasExcluded(ExcludedFilmArchive) {
		t.Fatalf("le manifeste doit dire que l'archive des films n'est pas copiée : %v", m.Excluded)
	}

	// Restauration dans une autre racine.
	root2 := t.TempDir()
	pr2 := title.NewPathResolverRoots(title.Roots{Resources: root2, Data: filepath.Join(root2, "data"), DataSet: true})
	auth2 := filepath.Join(pr2.DataRoot(), "auth")
	res, err := Restore(ctx, RestoreOptions{From: setDir, Paths: pr2, AuthDir: auth2})
	if err != nil {
		t.Fatalf("Restore : %v", err)
	}
	if len(res.Import.Restored) != 3 || len(res.ConfigRestored) != 5 {
		t.Fatalf("restauré : %d bases, %d fichiers", len(res.Import.Restored), len(res.ConfigRestored))
	}
	for _, c := range []struct {
		path, table string
		want        int
	}{
		{pr2.SharedDBPath(testSlug), "match_registry", 3},
		{pr2.MetadataDBPath(testSlug), "medal_definitions", 2},
		{pr2.PlayerDBPath(testSlug, "Spartan B"), "player_match_enrichment", 2},
	} {
		if got := countRows(t, c.path, c.table); got != c.want {
			t.Errorf("%s.%s : %d lignes, want %d", filepath.Base(c.path), c.table, got, c.want)
		}
	}
	// Le schéma de l'époque est rendu : la clé primaire, la séquence et la vue.
	db, err := sql.Open("duckdb", pr2.SharedDBPath(testSlug)+"?access_mode=read_only")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	var views int
	if err := db.QueryRow(`SELECT count(*) FROM v_maps`).Scan(&views); err != nil || views != 3 {
		t.Fatalf("vue v_maps restaurée : %d lignes, err=%v", views, err)
	}
	for _, p := range []string{pr2.DBProfilesPath(), pr2.AppSettingsPath(), filepath.Join(auth2, "users.json"),
		filepath.Join(pr2.WatcherTokensDir(), "2533274800000001.json")} {
		if _, err := os.Stat(p); err != nil {
			t.Errorf("fichier de configuration non restauré : %v", err)
		}
	}
}

func (m *Manifest) hasExcluded(what string) bool {
	for _, e := range m.Excluded {
		if e == what {
			return true
		}
	}
	return false
}

func TestRestore_RefusesNonEmptyTargetWithoutForce(t *testing.T) {
	ctx := context.Background()
	pr, authDir := seedRoot(t)
	dest := t.TempDir()
	m, err := Run(ctx, newOptions(pr, authDir, dest))
	if err != nil {
		t.Fatal(err)
	}
	// Restauration SUR la racine source : ses bases existent.
	_, err = Restore(ctx, RestoreOptions{From: filepath.Join(dest, m.Name), Paths: pr, AuthDir: authDir})
	if !errors.Is(err, ErrTargetNotEmpty) {
		t.Fatalf("restauration sur des bases existantes : %v, want ErrTargetNotEmpty", err)
	}
}

func TestRestore_RefusesAlteredSet(t *testing.T) {
	ctx := context.Background()
	pr, authDir := seedRoot(t)
	dest := t.TempDir()
	m, err := Run(ctx, newOptions(pr, authDir, dest))
	if err != nil {
		t.Fatal(err)
	}
	setDir := filepath.Join(dest, m.Name)
	victim := filepath.Join(setDir, filepath.FromSlash(m.Databases[0].Tables[0].File))
	if err := os.WriteFile(victim, []byte("corrompu"), 0o644); err != nil {
		t.Fatal(err)
	}
	root2 := t.TempDir()
	pr2 := title.NewPathResolverRoots(title.Roots{Resources: root2, Data: filepath.Join(root2, "data"), DataSet: true})
	if _, err := Restore(ctx, RestoreOptions{From: setDir, Paths: pr2, AuthDir: filepath.Join(root2, "auth")}); err == nil ||
		!strings.Contains(err.Error(), "altéré") {
		t.Fatalf("un jeu altéré doit être refusé : %v", err)
	}
}

func TestRun_BusyBaseGivesPartialSet(t *testing.T) {
	ctx := context.Background()
	pr, authDir := seedRoot(t)
	dest := t.TempDir()
	release, err := dblease.AcquireLease(pr.SharedDBPath(testSlug), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	m, err := Run(ctx, newOptions(pr, authDir, dest))
	if err != nil {
		t.Fatalf("Run : %v", err)
	}
	if !m.Partial {
		t.Fatalf("une base occupée doit rendre le jeu partiel")
	}
	var skipped *DatabaseEntry
	for i := range m.Databases {
		if m.Databases[i].Skipped {
			skipped = &m.Databases[i]
		}
	}
	if skipped == nil || skipped.Reason != reasonBusy || !strings.Contains(skipped.Key, "shared_matches_v2") {
		t.Fatalf("base sautée attendue (busy, shared) : %+v", skipped)
	}
}

func TestRun_NothingExportedKeepsNothing(t *testing.T) {
	ctx := context.Background()
	pr, authDir := seedRoot(t)
	dest := t.TempDir()
	for _, p := range []string{pr.SharedDBPath(testSlug), pr.MetadataDBPath(testSlug), pr.PlayerDBPath(testSlug, "Spartan B")} {
		release, err := dblease.AcquireLease(p, time.Second)
		if err != nil {
			t.Fatal(err)
		}
		defer release()
	}
	if _, err := Run(ctx, newOptions(pr, authDir, dest)); !errors.Is(err, ErrNothingExported) {
		t.Fatalf("Run : %v, want ErrNothingExported", err)
	}
	entries, _ := os.ReadDir(dest)
	if len(entries) != 0 {
		t.Fatalf("un échec ne laisse rien dans le dossier choisi : %v", entries)
	}
}

func TestRun_RotationKeepsOnlyKeepAndOnlyLevelupSets(t *testing.T) {
	ctx := context.Background()
	pr, authDir := seedRoot(t)
	dest := t.TempDir()
	foreign := filepath.Join(dest, "mes-photos")
	writeText(t, filepath.Join(foreign, "a.txt"), "x")
	writeText(t, filepath.Join(dest, "levelup-notes.txt"), "x") // fichier au préfixe : jamais touché
	base := time.Date(2026, 11, 3, 8, 0, 0, 0, time.Local)
	for i := 0; i < 4; i++ {
		o := newOptions(pr, authDir, dest)
		o.Keep = 2
		at := base.Add(time.Duration(i) * time.Hour)
		o.Now = func() time.Time { return at }
		if _, err := Run(ctx, o); err != nil {
			t.Fatalf("Run %d : %v", i, err)
		}
	}
	sets, err := SetDirs(dest)
	if err != nil || len(sets) != 2 {
		t.Fatalf("jeux gardés = %d (%v), want 2", len(sets), err)
	}
	if filepath.Base(sets[0]) != "levelup-20261103-1100" || filepath.Base(sets[1]) != "levelup-20261103-1000" {
		t.Fatalf("les deux plus récents attendus, obtenu %v", sets)
	}
	for _, p := range []string{foreign, filepath.Join(dest, "levelup-notes.txt")} {
		if _, err := os.Stat(p); err != nil {
			t.Errorf("la rotation a touché %s", p)
		}
	}
}

func TestRun_SameMinuteGetsSuffix(t *testing.T) {
	ctx := context.Background()
	pr, authDir := seedRoot(t)
	dest := t.TempDir()
	at := time.Date(2026, 11, 3, 8, 0, 0, 0, time.Local)
	o := newOptions(pr, authDir, dest)
	o.Now = func() time.Time { return at }
	m1, err := Run(ctx, o)
	if err != nil {
		t.Fatal(err)
	}
	m2, err := Run(ctx, o)
	if err != nil {
		t.Fatal(err)
	}
	if m1.Name == m2.Name || !strings.HasSuffix(m2.Name, "-2") {
		t.Fatalf("noms %q et %q : le second doit porter un suffixe", m1.Name, m2.Name)
	}
}

func TestRun_ReadsHeldHandleWithoutWriting(t *testing.T) {
	// Une base TENUE en RW par le processus (cache process-wide) se lit sans seconde ouverture.
	ctx := context.Background()
	pr, authDir := seedRoot(t)
	dest := t.TempDir()
	held, err := duckdb.OpenReadWrite(pr.MetadataDBPath(testSlug))
	if err != nil {
		t.Fatalf("OpenReadWrite : %v", err)
	}
	defer func() { _ = held.Close() }()
	m, err := Run(ctx, newOptions(pr, authDir, dest))
	if err != nil || m.Partial {
		t.Fatalf("Run avec une base tenue en RW : err=%v partial=%v", err, m != nil && m.Partial)
	}
	var got int
	if err := held.SQLDb().QueryRow(`SELECT count(*) FROM medal_definitions`).Scan(&got); err != nil || got != 2 {
		t.Fatalf("la base source a changé : %d lignes, err=%v", got, err)
	}
}

func TestValidateDest(t *testing.T) {
	root := t.TempDir()
	pr := title.NewPathResolverRoots(title.Roots{Resources: root, Data: filepath.Join(root, "data"), DataSet: true})
	if err := os.MkdirAll(pr.DataRoot(), 0o755); err != nil {
		t.Fatal(err)
	}
	good := t.TempDir()
	cases := []struct {
		name string
		dest string
		want error
	}{
		{"relatif", "dossier/relatif", ErrDestNotAbsolute},
		{"absent", filepath.Join(good, "nexistepas"), ErrDestMissing},
		{"sous les données", filepath.Join(pr.DataRoot()), ErrDestInsideData},
		{"valide", good, nil},
	}
	for _, c := range cases {
		if _, err := ValidateDest(c.dest, pr); !errors.Is(err, c.want) {
			t.Errorf("%s : %v, want %v", c.name, err, c.want)
		}
	}
	sub := filepath.Join(pr.DataRoot(), "titles")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := ValidateDest(sub, pr); !errors.Is(err, ErrDestInsideData) {
		t.Errorf("sous-dossier des données : %v, want ErrDestInsideData", err)
	}
}

// seedSet pose un jeu factice (dossier + manifeste) dans dest.
func seedSet(t *testing.T, dest, name string, partial bool) {
	t.Helper()
	dir := filepath.Join(dest, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := writeManifest(dir, &Manifest{Format: ManifestFormat, Name: name, Partial: partial}); err != nil {
		t.Fatal(err)
	}
}

// TestRotate_KeepsTheLastFullSetEvenWhenNewerOnesArePartial : une suite de jeux partiels ne chasse jamais le
// dernier jeu complet (S6.b) ; dès qu'un jeu complet plus récent est gardé, l'ancien peut partir.
func TestRotate_KeepsTheLastFullSetEvenWhenNewerOnesArePartial(t *testing.T) {
	dest := t.TempDir()
	seedSet(t, dest, "levelup-20261103-0700", true)
	seedSet(t, dest, "levelup-20261103-0800", false) // le dernier complet
	seedSet(t, dest, "levelup-20261103-0900", true)
	seedSet(t, dest, "levelup-20261103-1000", true)
	if removed := rotate(context.Background(), dest, 2); removed != 1 {
		t.Fatalf("jeux supprimés = %d, attendu 1 (le partiel de 07:00 ; le complet de 08:00 est gardé)", removed)
	}
	sets, _ := SetDirs(dest)
	var names []string
	for _, s := range sets {
		names = append(names, filepath.Base(s))
	}
	want := []string{"levelup-20261103-1000", "levelup-20261103-0900", "levelup-20261103-0800"}
	if len(names) != len(want) {
		t.Fatalf("jeux gardés %v, attendu %v", names, want)
	}
	for i := range want {
		if names[i] != want[i] {
			t.Fatalf("jeux gardés %v, attendu %v", names, want)
		}
	}
	// Un jeu complet plus récent arrive : l'ancien complet n'est plus protégé.
	seedSet(t, dest, "levelup-20261103-1100", false)
	rotate(context.Background(), dest, 2)
	sets, _ = SetDirs(dest)
	if len(sets) != 2 || filepath.Base(sets[0]) != "levelup-20261103-1100" {
		t.Fatalf("avec un complet récent, deux jeux gardés attendus : %v", sets)
	}
}

// emptyTarget : une racine cible vierge.
func emptyTarget(t *testing.T) (*title.PathResolver, string) {
	t.Helper()
	root := t.TempDir()
	pr := title.NewPathResolverRoots(title.Roots{Resources: root, Data: filepath.Join(root, "data"), DataSet: true})
	return pr, filepath.Join(pr.DataRoot(), "auth")
}

// TestRestore_RefusesAFileTheManifestDoesNotList : un Parquet ajouté au jeu après coup n'a aucune somme ; il ne
// s'importe pas (S4.A), même quand il porterait l'horodatage le plus récent d'une table.
func TestRestore_RefusesAFileTheManifestDoesNotList(t *testing.T) {
	ctx := context.Background()
	pr, authDir := seedRoot(t)
	dest := t.TempDir()
	m, err := Run(ctx, newOptions(pr, authDir, dest))
	if err != nil {
		t.Fatal(err)
	}
	setDir := filepath.Join(dest, m.Name)
	listed := m.Databases[0].Tables[0].File
	stranger := filepath.Join(setDir, filepath.FromSlash(filepath.ToSlash(filepath.Dir(listed))), "match_registry_29990101_000000.parquet")
	writeText(t, stranger, "pas du parquet")
	target, targetAuth := emptyTarget(t)
	_, err = Restore(ctx, RestoreOptions{From: setDir, Paths: target, AuthDir: targetAuth})
	if !errors.Is(err, ErrUnlistedFile) {
		t.Fatalf("fichier hors manifeste : %v, attendu ErrUnlistedFile", err)
	}
	if _, serr := os.Stat(target.SharedDBPath(testSlug)); serr == nil {
		t.Fatal("rien ne doit être écrit quand le jeu est refusé")
	}
}

// TestRestore_ResidualWALIsRefusedThenMovedWithItsBaseUnderForce : un WAL résiduel de la racine cible fait refuser
// la restauration sans --force ; avec --force il est écarté AVEC sa base (jamais rejoué par la base neuve, S4.B).
func TestRestore_ResidualWALIsRefusedThenMovedWithItsBaseUnderForce(t *testing.T) {
	ctx := context.Background()
	pr, authDir := seedRoot(t)
	dest := t.TempDir()
	m, err := Run(ctx, newOptions(pr, authDir, dest))
	if err != nil {
		t.Fatal(err)
	}
	setDir := filepath.Join(dest, m.Name)
	target, targetAuth := emptyTarget(t)
	seedDB(t, target.SharedDBPath(testSlug), `CREATE TABLE old_only (x INTEGER)`)
	wal := target.SharedDBPath(testSlug) + ".wal"
	writeText(t, wal, "journal d'une application tuée")

	_, err = Restore(ctx, RestoreOptions{From: setDir, Paths: target, AuthDir: targetAuth})
	if !errors.Is(err, ErrResidualWAL) {
		t.Fatalf("WAL résiduel sans --force : %v, attendu ErrResidualWAL", err)
	}
	if !strings.Contains(err.Error(), "WAL") {
		t.Errorf("le message doit dire quoi faire : %v", err)
	}
	if _, serr := os.Stat(wal); serr != nil {
		t.Fatal("le refus ne doit pas toucher au WAL")
	}

	if _, err := Restore(ctx, RestoreOptions{From: setDir, Paths: target, AuthDir: targetAuth, Force: true}); err != nil {
		t.Fatalf("Restore --force : %v", err)
	}
	if _, serr := os.Stat(wal); serr == nil {
		t.Fatal("le WAL de l'ancienne base doit avoir été écarté avec elle : la base neuve l'aurait rejoué")
	}
	if got := countRows(t, target.SharedDBPath(testSlug), "match_registry"); got != 3 {
		t.Fatalf("lignes restaurées = %d, attendu 3", got)
	}
}
