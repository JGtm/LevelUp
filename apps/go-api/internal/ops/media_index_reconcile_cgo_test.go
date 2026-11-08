//go:build cgo

package ops

// media_index_reconcile_cgo_test.go — IndexMedia aligné sur le disque : renommage,
// déplacement, noms identiques dans deux dossiers, vidéo convertie dont l'original
// disparaît, et état de la racine rendu au bandeau. Les vidéos sont de vraies vidéos
// courtes produites par ffmpeg (absent : les tests qui en ont besoin sont ignorés).

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	platform_duckdb "levelup/go-api/internal/platform/duckdb"
)

const (
	indexPlayer = "spartan"
	indexTitle  = "test_title"
)

type indexSetup struct {
	dir        string
	socialPath string
	base       string
	capDir     string
}

// newIndexSetup prépare une shared_social aux tables médias et de likes, et le dossier
// de captures du joueur.
func newIndexSetup(t *testing.T) *indexSetup {
	t.Helper()
	dir := t.TempDir()
	s := &indexSetup{
		dir: dir, socialPath: filepath.Join(dir, "shared_social.duckdb"),
		base: filepath.Join(dir, "captures"),
	}
	s.capDir = filepath.Join(s.base, indexPlayer)
	if err := os.MkdirAll(s.capDir, 0o755); err != nil {
		t.Fatal(err)
	}
	seed, err := sql.Open("duckdb", s.socialPath)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := ensureMediaTables(ctx, seed); err != nil {
		t.Fatal(err)
	}
	if _, err := seed.ExecContext(ctx, reconcileLikesDDL); err != nil {
		t.Fatal(err)
	}
	if err := seed.Close(); err != nil {
		t.Fatal(err)
	}
	return s
}

// opts : options d'une passe de scan (racine fiable), à modifier par mod.
func (s *indexSetup) opts(mod ...func(*MediaIndexOptions)) MediaIndexOptions {
	o := MediaIndexOptions{
		PlayerDBPath: filepath.Join(s.dir, "player.duckdb"), SharedSocialDBPath: s.socialPath,
		CapturesDir: s.capDir, CapturesBase: s.base, Gamertag: indexPlayer, Timezone: "Europe/Paris",
		TitleSlug: indexTitle, ForeignFilenamePrefixes: []string{}, MissingPolicy: MissingMark,
	}
	for _, m := range mod {
		m(&o)
	}
	return o
}

func (s *indexSetup) index(t *testing.T, mod ...func(*MediaIndexOptions)) MediaIndexResult {
	t.Helper()
	res, err := IndexMedia(context.Background(), s.opts(mod...))
	if err != nil {
		t.Fatalf("IndexMedia: %v", err)
	}
	return res
}

// read exécute fn sur une lecture de la base.
func (s *indexSetup) read(t *testing.T, fn func(db *sql.DB)) {
	t.Helper()
	db, release, err := platform_duckdb.OpenReadForQuery(s.socialPath)
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	fn(db)
}

// indexedRow : ce qu'un test relit d'une ligne.
type indexedRow struct {
	ID                                                 int64
	FilePath, SourcePath, HLSPath, Thumb, Status, Hash string
}

func (s *indexSetup) rows(t *testing.T) []indexedRow {
	t.Helper()
	var out []indexedRow
	s.read(t, func(db *sql.DB) {
		rs, err := db.Query(`SELECT id, COALESCE(file_path,''), COALESCE(source_path,''), COALESCE(hls_path,''),
			COALESCE(thumbnail_path,''), COALESCE(status,''), COALESCE(file_hash,'')
			FROM media_files ORDER BY id`)
		if err != nil {
			t.Fatal(err)
		}
		defer rs.Close()
		for rs.Next() {
			var r indexedRow
			if err := rs.Scan(&r.ID, &r.FilePath, &r.SourcePath, &r.HLSPath, &r.Thumb, &r.Status, &r.Hash); err != nil {
				t.Fatal(err)
			}
			out = append(out, r)
		}
	})
	return out
}

func requireFFmpeg(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg absent")
	}
}

// genMP4Tone produit un MP4 web-natif d'une seconde dont le contenu dépend de la
// fréquence : deux fréquences, deux empreintes.
func genMP4Tone(t *testing.T, path string, freq int) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("ffmpeg", "-hide_banner", "-loglevel", "error", "-y",
		"-f", "lavfi", "-i", "testsrc=size=320x240:rate=15:duration=1",
		"-f", "lavfi", "-i", fmt.Sprintf("sine=frequency=%d:duration=1", freq),
		"-map", "0:v", "-map", "1:a", "-c:v", "libx264", "-preset", "ultrafast", "-c:a", "aac",
		"-movflags", "+faststart", path)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("génération MP4: %v\n%s", err, out)
	}
}

func TestIndexMedia_RenameKeepsRowAndThumbnail(t *testing.T) {
	requireFFmpeg(t)
	s := newIndexSetup(t)
	genMP4Tone(t, filepath.Join(s.capDir, "clip.mp4"), 440)

	first := s.index(t)
	if first.NewFiles != 1 {
		t.Fatalf("première passe : NewFiles = %d, want 1 (erreurs %v)", first.NewFiles, first.Errors)
	}
	before := s.rows(t)
	if len(before) != 1 || before[0].Thumb == "" || !fileExists(filepath.Join(s.base, filepath.FromSlash(before[0].Thumb))) {
		t.Fatalf("après la première passe : %+v, want une ligne avec sa miniature sur le disque", before)
	}

	if err := os.Rename(filepath.Join(s.capDir, "clip.mp4"), filepath.Join(s.capDir, "clip renomme.mp4")); err != nil {
		t.Fatal(err)
	}
	second := s.index(t)
	if second.NewFiles != 0 {
		t.Errorf("seconde passe : NewFiles = %d, want 0 (un renommage n'est pas un nouveau média)", second.NewFiles)
	}
	after := s.rows(t)
	if len(after) != 1 {
		t.Fatalf("après renommage : %d lignes, want 1 (%+v)", len(after), after)
	}
	if after[0].ID != before[0].ID || after[0].FilePath != indexPlayer+"/clip renomme.mp4" ||
		after[0].SourcePath != after[0].FilePath || after[0].Status != "" {
		t.Errorf("ligne après renommage = %+v, want même id, file_path et source_path %q, status NULL", after[0], indexPlayer+"/clip renomme.mp4")
	}
	if after[0].Thumb != before[0].Thumb || !fileExists(filepath.Join(s.base, filepath.FromSlash(after[0].Thumb))) {
		t.Errorf("miniature %q -> %q, want conservée et présente", before[0].Thumb, after[0].Thumb)
	}
}

func TestIndexMedia_ForceRescanAfterRenameNoDuplicate(t *testing.T) {
	requireFFmpeg(t)
	s := newIndexSetup(t)
	genMP4Tone(t, filepath.Join(s.capDir, "clip.mp4"), 440)
	s.index(t)

	if err := os.Rename(filepath.Join(s.capDir, "clip.mp4"), filepath.Join(s.capDir, "autre nom.mp4")); err != nil {
		t.Fatal(err)
	}
	// ForceRescan compte chaque fichier présenté à l'insertion (NewFiles) même quand elle
	// le reconnaît ; ce qui compte ici est l'absence de seconde ligne.
	if res := s.index(t, func(o *MediaIndexOptions) { o.ForceRescan = true }); len(res.Errors) != 0 {
		t.Errorf("erreurs %v, want aucune", res.Errors)
	}
	if rows := s.rows(t); len(rows) != 1 || rows[0].FilePath != indexPlayer+"/autre nom.mp4" {
		t.Errorf("lignes = %+v, want une seule, sur le nouveau nom", rows)
	}
}

func TestIndexMedia_SameNameInTwoFoldersKeepsBoth(t *testing.T) {
	requireFFmpeg(t)
	s := newIndexSetup(t)
	genMP4Tone(t, filepath.Join(s.capDir, "a", "clip.mp4"), 440)
	genMP4Tone(t, filepath.Join(s.capDir, "b", "clip.mp4"), 880)

	res := s.index(t)
	if res.NewFiles != 2 {
		t.Fatalf("NewFiles = %d, want 2 (erreurs %v)", res.NewFiles, res.Errors)
	}
	rows := s.rows(t)
	if len(rows) != 2 || rows[0].FilePath == rows[1].FilePath {
		t.Fatalf("lignes = %+v, want deux médias distincts", rows)
	}
	if rows[0].Thumb == "" || rows[1].Thumb == "" || rows[0].Thumb == rows[1].Thumb {
		t.Fatalf("miniatures %q et %q, want deux miniatures distinctes", rows[0].Thumb, rows[1].Thumb)
	}
	for _, r := range rows {
		if !fileExists(filepath.Join(s.base, filepath.FromSlash(r.Thumb))) {
			t.Errorf("miniature %q absente du disque", r.Thumb)
		}
	}
	again := s.index(t)
	if again.NewFiles != 0 || len(s.rows(t)) != 2 {
		t.Errorf("seconde passe : NewFiles = %d, %d lignes, want 0 et 2", again.NewFiles, len(s.rows(t)))
	}
}

// TestIndexMedia_AfterHLSTranscodeDeletingSource : la conversion qui retire l'original
// détache la fiche de lui ; la passe suivante ne marque rien introuvable.
func TestIndexMedia_AfterHLSTranscodeDeletingSource(t *testing.T) {
	requireFFmpeg(t)
	s := newIndexSetup(t)
	mkv := genHLSSourceMKV(t, s.capDir, "multi.mkv")
	first := s.index(t)
	if first.NewFiles != 1 || first.Thumbnails != 1 {
		t.Fatalf("première passe : NewFiles = %d, Thumbnails = %d, want 1 et 1 (erreurs %v)", first.NewFiles, first.Thumbnails, first.Errors)
	}
	before := s.rows(t)
	if len(before) != 1 || before[0].SourcePath != indexPlayer+"/multi.mkv" {
		t.Fatalf("avant conversion : %+v, want source_path renseigné", before)
	}

	outDir, hlsRel := HLSPathsFor(s.capDir, s.base, indexPlayer, mkv)
	if err := RunHLSTranscode(context.Background(), HLSTranscodeParams{
		SourceAbs: mkv, OutDir: outDir, DBPath: s.socialPath,
		FileRel: before[0].FilePath, HLSRel: hlsRel, DeleteSource: true,
	}); err != nil {
		t.Fatalf("RunHLSTranscode: %v", err)
	}
	if fileExists(mkv) {
		t.Fatal("l'original devait être retiré après la conversion")
	}

	res := s.index(t)
	if res.NewFiles != 0 {
		t.Errorf("NewFiles = %d, want 0", res.NewFiles)
	}
	after := s.rows(t)
	if len(after) != 1 || after[0].Status != "" || after[0].SourcePath != "" ||
		after[0].FilePath != hlsRel || after[0].HLSPath != hlsRel {
		t.Errorf("après conversion : %+v, want visible, source_path NULL, file_path et hls_path = %q", after, hlsRel)
	}
}

// seedPhantomRows écrit n lignes dont aucun fichier n'existe.
func (s *indexSetup) seedPhantomRows(t *testing.T, n int) {
	t.Helper()
	db, err := sql.Open("duckdb", s.socialPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for i := 0; i < n; i++ {
		name := fmt.Sprintf("fantome %02d.mp4", i)
		if _, err := db.Exec(`INSERT INTO media_files (player_slug, file_path, file_name, file_stem, file_ext, kind,
			file_hash, source_path) VALUES (?, ?, ?, ?, '.mp4', 'video', ?, ?)`,
			indexPlayer, indexPlayer+"/"+name, name, fmt.Sprintf("fantome %02d", i),
			fmt.Sprintf("h%02d", i), indexPlayer+"/"+name); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.Exec("CHECKPOINT"); err != nil {
		t.Fatal(err)
	}
}

func (s *indexSetup) countStatus(t *testing.T, status string) int {
	t.Helper()
	n := 0
	for _, r := range s.rows(t) {
		if r.Status == status {
			n++
		}
	}
	return n
}

// TestIndexMedia_BulkDisappearanceWaitsForConfirmation : une disparition massive est
// suspendue et rendue au bandeau ; la confirmation la fait marquer à la passe suivante,
// une seule fois.
func TestIndexMedia_BulkDisappearanceWaitsForConfirmation(t *testing.T) {
	s := newIndexSetup(t)
	s.seedPhantomRows(t, 20)

	s.index(t)
	if got := s.countStatus(t, "missing"); got != 0 {
		t.Fatalf("%d lignes marquées, want 0 (garde-fou)", got)
	}
	if st := GetRootState(indexTitle, indexPlayer); st.Unavailable || st.BulkPending != 20 {
		t.Fatalf("état = %+v, want {false 20}", st)
	}

	ConfirmBulkMissing(indexTitle, indexPlayer)
	s.index(t)
	if got := s.countStatus(t, "missing"); got != 20 {
		t.Fatalf("%d lignes marquées après confirmation, want 20", got)
	}
	if st := GetRootState(indexTitle, indexPlayer); st.Unavailable || st.BulkPending != 0 {
		t.Errorf("état après confirmation = %+v, want {false 0}", st)
	}
	if ConsumeBulkConfirmation(indexTitle, indexPlayer) {
		t.Error("la confirmation doit être consommée par la passe")
	}
}

// TestIndexMedia_UnavailableRootMarksNothing : racine indisponible, aucune ligne n'est
// marquée et l'état le dit ; un défaut MissingIgnore n'enregistre rien.
func TestIndexMedia_UnavailableRootMarksNothing(t *testing.T) {
	s := newIndexSetup(t)
	s.seedPhantomRows(t, 3)
	RecordRootState(indexTitle, indexPlayer, RootState{})

	s.index(t, func(o *MediaIndexOptions) { o.MissingPolicy = MissingIgnore })
	if st := GetRootState(indexTitle, indexPlayer); st != (RootState{}) {
		t.Errorf("MissingIgnore a enregistré %+v, want rien", st)
	}

	s.index(t, func(o *MediaIndexOptions) { o.MissingPolicy = MissingRootUnavailable })
	if got := s.countStatus(t, "missing"); got != 0 {
		t.Errorf("%d lignes marquées, want 0", got)
	}
	if st := GetRootState(indexTitle, indexPlayer); !st.Unavailable || st.BulkPending != 0 {
		t.Errorf("état = %+v, want {true 0}", st)
	}

	s.index(t)
	if got := s.countStatus(t, "missing"); got != 3 {
		t.Errorf("racine fiable : %d lignes marquées, want 3", got)
	}
	if st := GetRootState(indexTitle, indexPlayer); st.Unavailable {
		t.Errorf("état = %+v, want la racine de nouveau disponible", st)
	}
}

// TestIndexMedia_ReconcileFailureIsFatal : sans table de lignes lisible, l'indexation
// rend l'erreur au lieu de poursuivre avec des empreintes qu'elle n'a pas.
func TestIndexMedia_ReconcileFailureIsFatal(t *testing.T) {
	s := newIndexSetup(t)
	s.seedPhantomRows(t, 1)
	genFake := filepath.Join(s.capDir, "renomme.mp4")
	if err := os.WriteFile(genFake, []byte("contenu"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Une ligne dont l'empreinte est celle du fichier : la relocalisation écrit des
	// likes, dont la vue est retirée pour provoquer l'échec de la passe.
	db, err := sql.Open("duckdb", s.socialPath)
	if err != nil {
		t.Fatal(err)
	}
	hash, err := HashFile(genFake)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE media_files SET file_hash = ?`, hash); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`DROP VIEW media_likes_latest`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("CHECKPOINT"); err != nil {
		t.Fatal(err)
	}
	db.Close()

	if _, err := IndexMedia(context.Background(), s.opts()); err == nil {
		t.Fatal("IndexMedia a réussi alors que la passe de réconciliation échoue, want une erreur")
	}
}
