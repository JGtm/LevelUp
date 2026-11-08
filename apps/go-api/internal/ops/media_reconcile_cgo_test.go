//go:build cgo

// Package ops — media_reconcile_cgo_test.go : socle et premiers cas de
// ReconcileMediaWithDisk (renommage, déplacement, conversion, disparition,
// retour). Les cas des vidéos converties, du garde-fou et de la concurrence sont
// dans media_reconcile_cases_cgo_test.go.
//
// Le hachage est injecté : l'empreinte d'un fichier est "h:" + son contenu, et
// chaque appel est compté, ce qui permet de prouver qu'un fichier inchangé n'est
// pas relu.
package ops

import (
	"context"
	"database/sql"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
)

const reconcilePlayer = "Tester"

// reconcileLikesDDL : substrat des likes, créé par les migrations en production.
const reconcileLikesDDL = `
	CREATE SEQUENCE IF NOT EXISTS media_likes_history_id_seq START 1;
	CREATE TABLE IF NOT EXISTS media_likes_history (
		id BIGINT PRIMARY KEY DEFAULT nextval('media_likes_history_id_seq'),
		media_path VARCHAR NOT NULL, liker_slug VARCHAR NOT NULL, liker_gamertag VARCHAR,
		is_liked BOOLEAN NOT NULL, liked_at TIMESTAMP,
		written_at TIMESTAMP NOT NULL DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP)
	);
	CREATE OR REPLACE VIEW media_likes_latest AS
		SELECT id, media_path, liker_slug, liker_gamertag, is_liked, liked_at, written_at
		FROM media_likes_history
		QUALIFY ROW_NUMBER() OVER (PARTITION BY media_path, liker_slug
			ORDER BY written_at DESC, id DESC) = 1;`

type reconcileSetup struct {
	db        *sql.DB
	base      string // racine des captures
	dir       string // dossier du joueur
	store     MediaPathStore
	hashCalls int
	// onHash : crochet appelé avant chaque empreinte (simule une écriture concurrente).
	onHash func()
}

func newReconcileSetup(t *testing.T) *reconcileSetup {
	t.Helper()
	_, db := openDiagDB(t)
	ctx := context.Background()
	if err := ensureMediaTables(ctx, db); err != nil {
		t.Fatalf("ensureMediaTables: %v", err)
	}
	if _, err := db.ExecContext(ctx, reconcileLikesDDL); err != nil {
		t.Fatalf("DDL des likes: %v", err)
	}
	base := t.TempDir()
	dir := filepath.Join(base, reconcilePlayer)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	return &reconcileSetup{db: db, base: base, dir: dir, store: MediaPathStore{CapturesBase: base}}
}

// abs rend le chemin absolu d'un chemin relatif au dossier du joueur.
func (s *reconcileSetup) abs(rel string) string { return filepath.Join(s.dir, filepath.FromSlash(rel)) }

// stored rend le chemin tel que la base le stocke.
func (s *reconcileSetup) stored(rel string) string { return reconcilePlayer + "/" + rel }

// write dépose un fichier, daté d'il y a deux heures (donc stable).
func (s *reconcileSetup) write(t *testing.T, rel, content string) {
	t.Helper()
	p := s.abs(rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	past := time.Now().Add(-2 * time.Hour)
	if err := os.Chtimes(p, past, past); err != nil {
		t.Fatal(err)
	}
}

// chtimes date un fichier (accès et modification).
func chtimes(p string, at time.Time) error { return os.Chtimes(p, at, at) }

func (s *reconcileSetup) remove(t *testing.T, rel string) {
	t.Helper()
	if err := os.Remove(s.abs(rel)); err != nil {
		t.Fatal(err)
	}
}

func (s *reconcileSetup) move(t *testing.T, from, to string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(s.abs(to)), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(s.abs(from), s.abs(to)); err != nil {
		t.Fatal(err)
	}
}

// fakeHash : "h:" + contenu, compté.
func (s *reconcileSetup) fakeHash(p string) (string, error) {
	s.hashCalls++
	if s.onHash != nil {
		s.onHash()
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return "", err
	}
	return "h:" + string(b), nil
}

// rowSpec décrit une ligne media_files de départ. Les chemins sont relatifs au
// dossier du joueur ; la taille et la date de l'original sont relevées sur le
// disque s'il y existe déjà.
type rowSpec struct {
	player string // défaut : reconcilePlayer
	rel    string // file_path
	name   string // file_name ; défaut : nom de rel
	hash   string // file_hash ; vide = NULL
	source string // source_path ; vide = NULL
	hls    string // hls_path ; vide = NULL
	status string // vide = NULL
}

func nullable(v string) any {
	if v == "" {
		return nil
	}
	return v
}

func (s *reconcileSetup) addRow(t *testing.T, sp rowSpec) int64 {
	t.Helper()
	player := sp.player
	if player == "" {
		player = reconcilePlayer
	}
	name := sp.name
	if name == "" {
		name = path.Base(sp.rel)
	}
	ext := path.Ext(name)
	var size, mtime any
	if sp.source != "" {
		if st, err := os.Stat(s.abs(sp.source)); err == nil {
			size, mtime = st.Size(), st.ModTime().UnixNano()
		}
	}
	srcStored := ""
	if sp.source != "" {
		srcStored = player + "/" + sp.source
	}
	hlsStored, transcode := "", ""
	if sp.hls != "" {
		hlsStored, transcode = player+"/"+sp.hls, "ready"
	}
	var id int64
	err := s.db.QueryRowContext(context.Background(), `
		INSERT INTO media_files (player_slug, file_path, file_name, file_stem, file_ext, kind,
			file_hash, source_path, source_size, source_mtime_ns, hls_path, transcode_status, status)
		VALUES (?, ?, ?, ?, ?, 'video', ?, ?, ?, ?, ?, ?, ?) RETURNING id`,
		player, player+"/"+sp.rel, name, name[:len(name)-len(ext)], ext,
		nullable(sp.hash), nullable(srcStored), size, mtime,
		nullable(hlsStored), nullable(transcode), nullable(sp.status)).Scan(&id)
	if err != nil {
		t.Fatalf("INSERT fixture: %v", err)
	}
	return id
}

// run lance une passe sur les médias du dossier du joueur.
func (s *reconcileSetup) run(t *testing.T, policy MissingPolicy, mod ...func(*ReconcileInput)) ReconcileResult {
	t.Helper()
	files, err := walkMediaDir(s.dir)
	if err != nil {
		t.Fatal(err)
	}
	in := ReconcileInput{
		PlayerSlug: reconcilePlayer, Files: files, Store: s.store, Policy: policy, Hash: s.fakeHash,
	}
	for _, m := range mod {
		m(&in)
	}
	res, err := ReconcileMediaWithDisk(context.Background(), s.db, in)
	if err != nil {
		t.Fatalf("ReconcileMediaWithDisk: %v", err)
	}
	return res
}

type rowState struct {
	ID                                                                  int64
	Player, FilePath, FileName, FileExt, FileStem, FileHash, SourcePath string
	HLSPath, Transcode, Status                                          string
	SourceSize, SourceMtime                                             int64
}

// rows rend toutes les lignes, dans l'ordre des id.
func (s *reconcileSetup) rows(t *testing.T) []rowState {
	t.Helper()
	rs, err := s.db.QueryContext(context.Background(), `
		SELECT id, COALESCE(player_slug,''), COALESCE(file_path,''), COALESCE(file_name,''),
		       COALESCE(file_ext,''), COALESCE(file_stem,''), COALESCE(file_hash,''),
		       COALESCE(source_path,''), COALESCE(hls_path,''), COALESCE(transcode_status,''),
		       COALESCE(status,''), COALESCE(source_size,-1), COALESCE(source_mtime_ns,-1)
		FROM media_files ORDER BY id`)
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Close()
	var out []rowState
	for rs.Next() {
		var r rowState
		if err := rs.Scan(&r.ID, &r.Player, &r.FilePath, &r.FileName, &r.FileExt, &r.FileStem,
			&r.FileHash, &r.SourcePath, &r.HLSPath, &r.Transcode, &r.Status,
			&r.SourceSize, &r.SourceMtime); err != nil {
			t.Fatal(err)
		}
		out = append(out, r)
	}
	return out
}

func (s *reconcileSetup) row(t *testing.T, id int64) rowState {
	t.Helper()
	for _, r := range s.rows(t) {
		if r.ID == id {
			return r
		}
	}
	t.Fatalf("ligne %d introuvable", id)
	return rowState{}
}

func (s *reconcileSetup) like(t *testing.T, mediaPath, liker string, liked bool) {
	t.Helper()
	q := `INSERT INTO media_likes_history (media_path, liker_slug, liker_gamertag, is_liked, liked_at)
		VALUES (?, ?, ?, TRUE, CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP))`
	if !liked {
		q = `INSERT INTO media_likes_history (media_path, liker_slug, liker_gamertag, is_liked, liked_at)
		VALUES (?, ?, ?, FALSE, NULL)`
	}
	if _, err := s.db.ExecContext(context.Background(), q, mediaPath, liker, liker); err != nil {
		t.Fatal(err)
	}
}

// likers rend les likers actifs d'un chemin stocké, lus par media_likes_latest.
func (s *reconcileSetup) likers(t *testing.T, mediaPath string) []string {
	t.Helper()
	rs, err := s.db.QueryContext(context.Background(),
		`SELECT liker_slug FROM media_likes_latest WHERE media_path = ? AND is_liked ORDER BY liker_slug`, mediaPath)
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Close()
	var out []string
	for rs.Next() {
		var l string
		if err := rs.Scan(&l); err != nil {
			t.Fatal(err)
		}
		out = append(out, l)
	}
	sort.Strings(out)
	return out
}

func (s *reconcileSetup) likeEvents(t *testing.T) int {
	t.Helper()
	var n int
	if err := s.db.QueryRowContext(context.Background(), `SELECT count(*) FROM media_likes_history`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func sameRows(t *testing.T, before, after []rowState) {
	t.Helper()
	if !reflect.DeepEqual(before, after) {
		t.Errorf("lignes modifiées:\navant %+v\naprès %+v", before, after)
	}
}

// expectRelocated vérifie l'état d'une ligne relocalisée vers rel : chemin, nom,
// extension et original à jour, identité (id, file_stem) intacte.
func expectRelocated(t *testing.T, before, after rowState, rel string) {
	t.Helper()
	if after.FilePath != reconcilePlayer+"/"+rel || after.SourcePath != reconcilePlayer+"/"+rel {
		t.Errorf("file_path=%q source_path=%q, want %q", after.FilePath, after.SourcePath, reconcilePlayer+"/"+rel)
	}
	if after.FileName != path.Base(rel) || after.FileExt != path.Ext(rel) {
		t.Errorf("file_name=%q file_ext=%q, want %q %q", after.FileName, after.FileExt, path.Base(rel), path.Ext(rel))
	}
	if after.ID != before.ID || after.FileStem != before.FileStem {
		t.Errorf("identité modifiée: id %d->%d file_stem %q->%q", before.ID, after.ID, before.FileStem, after.FileStem)
	}
	if after.Status != "" {
		t.Errorf("status=%q, want NULL", after.Status)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Les six cas de l'ancienne passe d'orphelins, sur la nouvelle fonction.
// ─────────────────────────────────────────────────────────────────────────────

func TestReconcile_FilePresentNoChange(t *testing.T) {
	s := newReconcileSetup(t)
	s.write(t, "clip.mp4", "x")
	id := s.addRow(t, rowSpec{rel: "clip.mp4"})
	res := s.run(t, MissingMark)
	if res.Relinked+res.Converted != 0 {
		t.Errorf("relocalisations = %d, want 0 (fichier toujours présent)", res.Relinked+res.Converted)
	}
	if r := s.row(t, id); r.FilePath != s.stored("clip.mp4") || r.FileExt != ".mp4" || r.Status != "" {
		t.Errorf("entrée modifiée par erreur: %+v", r)
	}
}

func TestReconcile_OrphanWithUniqueMkvMatch(t *testing.T) {
	s := newReconcileSetup(t)
	id := s.addRow(t, rowSpec{rel: "clip.mp4"})
	s.write(t, "clip.mkv", "x")
	res := s.run(t, MissingMark)
	if res.Converted != 1 {
		t.Errorf("Converted = %d, want 1", res.Converted)
	}
	r := s.row(t, id)
	if r.FileExt != ".mkv" || r.FilePath != s.stored("clip.mkv") {
		t.Errorf("entrée = %+v, want clip.mkv", r)
	}
}

func TestReconcile_OrphanNoMatch(t *testing.T) {
	s := newReconcileSetup(t)
	id := s.addRow(t, rowSpec{rel: "clip.mp4"})
	res := s.run(t, MissingMark)
	if res.MarkedMissing != 1 {
		t.Errorf("MarkedMissing = %d, want 1 (pas de match)", res.MarkedMissing)
	}
	r := s.row(t, id)
	if r.Status != "missing" || r.FilePath != s.stored("clip.mp4") || r.FileExt != ".mp4" {
		t.Errorf("entrée = %+v, want missing et chemin intact", r)
	}
}

func TestReconcile_AmbiguousMatchesSkipped(t *testing.T) {
	s := newReconcileSetup(t)
	id := s.addRow(t, rowSpec{rel: "clip.mp4"})
	s.write(t, "clip.mkv", "x")
	s.write(t, "clip.mov", "y")
	res := s.run(t, MissingMark)
	if res.Relinked+res.Converted != 0 {
		t.Errorf("relocalisations = %d, want 0 (ambigu)", res.Relinked+res.Converted)
	}
	if r := s.row(t, id); r.FilePath != s.stored("clip.mp4") || r.FileExt != ".mp4" {
		t.Errorf("entrée modifiée par erreur: %+v", r)
	}
}

func TestReconcile_Idempotent(t *testing.T) {
	s := newReconcileSetup(t)
	s.addRow(t, rowSpec{rel: "clip.mp4"})
	s.write(t, "clip.mkv", "x")
	if r1 := s.run(t, MissingMark); r1.Converted != 1 {
		t.Errorf("1er run Converted = %d, want 1", r1.Converted)
	}
	r2 := s.run(t, MissingMark)
	if r2.Converted != 0 || r2.writes != 0 {
		t.Errorf("2e run Converted=%d writes=%d, want 0 (idempotent)", r2.Converted, r2.writes)
	}
}

func TestReconcile_OnlyTargetsRequestedPlayer(t *testing.T) {
	s := newReconcileSetup(t)
	s.addRow(t, rowSpec{rel: "clip_PlayerA.mp4"})
	idB := s.addRow(t, rowSpec{player: "PlayerB", rel: "clip_PlayerB.mp4"})
	s.write(t, "clip_PlayerA.mkv", "x")
	before := s.row(t, idB)
	res := s.run(t, MissingMark)
	if res.Converted != 1 {
		t.Errorf("Converted = %d, want 1 (seulement le joueur demandé)", res.Converted)
	}
	if after := s.row(t, idB); after != before {
		t.Errorf("PlayerB modifié par erreur: %+v -> %+v", before, after)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Tableau du plan : R1 à R10.
// ─────────────────────────────────────────────────────────────────────────────

func TestReconcile_R1_UnchangedTwoPasses(t *testing.T) {
	s := newReconcileSetup(t)
	s.write(t, "clip.mp4", "a")
	s.addRow(t, rowSpec{rel: "clip.mp4", hash: "h:a"})
	first := s.run(t, MissingMark)
	if first.Backfilled != 1 || s.hashCalls != 1 {
		t.Fatalf("1re passe: Backfilled=%d hashCalls=%d, want 1 et 1", first.Backfilled, s.hashCalls)
	}
	before := s.rows(t)
	second := s.run(t, MissingMark)
	if second.writes != 0 {
		t.Errorf("2e passe: writes = %d, want 0", second.writes)
	}
	if s.hashCalls != 1 {
		t.Errorf("2e passe: hashCalls = %d, want 1 (fichier inchangé non relu)", s.hashCalls)
	}
	sameRows(t, before, s.rows(t))
}

func TestReconcile_R2_RenamedSameDir(t *testing.T) {
	s := newReconcileSetup(t)
	s.write(t, "clip.mp4", "a")
	id := s.addRow(t, rowSpec{rel: "clip.mp4", hash: "h:a", source: "clip.mp4"})
	before := s.row(t, id)
	s.move(t, "clip.mp4", "renamed.mp4")
	if res := s.run(t, MissingMark); res.Relinked != 1 {
		t.Errorf("Relinked = %d, want 1", res.Relinked)
	}
	expectRelocated(t, before, s.row(t, id), "renamed.mp4")
}

func TestReconcile_R3_MovedToSubdir(t *testing.T) {
	s := newReconcileSetup(t)
	s.write(t, "clip.mp4", "a")
	id := s.addRow(t, rowSpec{rel: "clip.mp4", hash: "h:a", source: "clip.mp4"})
	before := s.row(t, id)
	s.move(t, "clip.mp4", "sub/clip.mp4")
	if res := s.run(t, MissingMark); res.Relinked != 1 {
		t.Errorf("Relinked = %d, want 1", res.Relinked)
	}
	expectRelocated(t, before, s.row(t, id), "sub/clip.mp4")
}

func TestReconcile_R4_RenamedAndMoved(t *testing.T) {
	s := newReconcileSetup(t)
	s.write(t, "clip.mp4", "a")
	id := s.addRow(t, rowSpec{rel: "clip.mp4", hash: "h:a", source: "clip.mp4"})
	before := s.row(t, id)
	s.move(t, "clip.mp4", "deep/er/other name.mp4")
	if res := s.run(t, MissingMark); res.Relinked != 1 {
		t.Errorf("Relinked = %d, want 1", res.Relinked)
	}
	expectRelocated(t, before, s.row(t, id), "deep/er/other name.mp4")
}

func TestReconcile_R5_ExtensionChangedUniqueCandidate(t *testing.T) {
	s := newReconcileSetup(t)
	id := s.addRow(t, rowSpec{rel: "clip.mp4", hash: "h:a"})
	s.write(t, "clip.mkv", "b")
	before := s.row(t, id)
	if res := s.run(t, MissingMark); res.Converted != 1 {
		t.Errorf("Converted = %d, want 1", res.Converted)
	}
	after := s.row(t, id)
	expectRelocated(t, before, after, "clip.mkv")
	if after.FileHash != "h:b" {
		t.Errorf("file_hash = %q, want h:b (nouvelle empreinte)", after.FileHash)
	}
}

func TestReconcile_R6_TwoCandidatesSameName(t *testing.T) {
	s := newReconcileSetup(t)
	id := s.addRow(t, rowSpec{rel: "clip.mp4", hash: "h:a"})
	s.write(t, "clip.mkv", "b")
	s.write(t, "clip.mov", "c")
	res := s.run(t, MissingMark)
	r := s.row(t, id)
	if res.Relinked+res.Converted != 0 || r.FilePath != s.stored("clip.mp4") || r.Status != "missing" {
		t.Errorf("res=%+v row=%+v, want aucune relocalisation et missing", res, r)
	}
}

func TestReconcile_R7_Deleted(t *testing.T) {
	s := newReconcileSetup(t)
	s.write(t, "clip.mp4", "a")
	id := s.addRow(t, rowSpec{rel: "clip.mp4", hash: "h:a", source: "clip.mp4"})
	s.remove(t, "clip.mp4")
	if res := s.run(t, MissingMark); res.MarkedMissing != 1 {
		t.Errorf("MarkedMissing = %d, want 1", res.MarkedMissing)
	}
	if r := s.row(t, id); r.Status != "missing" || r.FilePath != s.stored("clip.mp4") {
		t.Errorf("row = %+v, want missing et file_path inchangé", r)
	}
}

func TestReconcile_R8_ReturnsSamePlace(t *testing.T) {
	s := newReconcileSetup(t)
	s.write(t, "clip.mp4", "a")
	id := s.addRow(t, rowSpec{rel: "clip.mp4", hash: "h:a", source: "clip.mp4"})
	s.remove(t, "clip.mp4")
	s.run(t, MissingMark)
	s.write(t, "clip.mp4", "a")
	res := s.run(t, MissingMark)
	if res.Restored != 1 {
		t.Errorf("Restored = %d, want 1", res.Restored)
	}
	if r := s.row(t, id); r.Status != "" {
		t.Errorf("status = %q, want NULL", r.Status)
	}
}

func TestReconcile_R9_ReturnsElsewhere(t *testing.T) {
	s := newReconcileSetup(t)
	s.write(t, "clip.mp4", "a")
	id := s.addRow(t, rowSpec{rel: "clip.mp4", hash: "h:a", source: "clip.mp4"})
	before := s.row(t, id)
	s.remove(t, "clip.mp4")
	s.run(t, MissingMark)
	s.write(t, "elsewhere/back.mp4", "a")
	if res := s.run(t, MissingMark); res.Relinked != 1 {
		t.Errorf("Relinked = %d, want 1", res.Relinked)
	}
	expectRelocated(t, before, s.row(t, id), "elsewhere/back.mp4")
}

func TestReconcile_R10_LikesFollowRename(t *testing.T) {
	s := newReconcileSetup(t)
	s.write(t, "clip.mp4", "a")
	s.addRow(t, rowSpec{rel: "clip.mp4", hash: "h:a", source: "clip.mp4"})
	oldPath, newPath := s.stored("clip.mp4"), s.stored("renamed.mp4")
	s.like(t, oldPath, "alice", true)
	s.like(t, oldPath, "bob", true)
	s.like(t, oldPath, "carol", true)
	s.like(t, oldPath, "carol", false)
	s.move(t, "clip.mp4", "renamed.mp4")
	s.run(t, MissingMark)
	if got := s.likers(t, newPath); !reflect.DeepEqual(got, []string{"alice", "bob"}) {
		t.Errorf("likers du nouveau chemin = %v, want [alice bob]", got)
	}
	if got := s.likers(t, oldPath); len(got) != 0 {
		t.Errorf("likers de l'ancien chemin = %v, want aucun", got)
	}
}
