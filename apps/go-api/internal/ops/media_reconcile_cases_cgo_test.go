//go:build cgo

// Package ops — media_reconcile_cases_cgo_test.go : suite du tableau de
// ReconcileMediaWithDisk (R11 à R25) : vidéos converties, garde-fou de disparition
// massive, politiques, isolation par joueur, fichiers frais, écriture concurrente.
// Socle et helpers : media_reconcile_cgo_test.go.
package ops

import (
	"fmt"
	"reflect"
	"testing"
	"time"
)

const (
	hlsMasterRel = "hls/clip/master.m3u8"
	hlsMasterTxt = "#EXTM3U"
)

// hlsRow ajoute une vidéo convertie : copie de l'app (master), original au
// chemin source (vide = aucun original attendu).
func (s *reconcileSetup) hlsRow(t *testing.T, source, hash string) int64 {
	t.Helper()
	s.write(t, hlsMasterRel, hlsMasterTxt)
	return s.addRow(t, rowSpec{rel: hlsMasterRel, name: "clip.mp4", hash: hash, source: source, hls: hlsMasterRel})
}

func TestReconcile_R11_ConvertedOriginalRenamed(t *testing.T) {
	s := newReconcileSetup(t)
	s.write(t, "clip.mp4", "a")
	id := s.hlsRow(t, "clip.mp4", "h:a")
	s.like(t, s.stored(hlsMasterRel), "alice", true)
	events := s.likeEvents(t)
	s.move(t, "clip.mp4", "clip2.mp4")
	if res := s.run(t, MissingMark); res.Relinked != 1 {
		t.Errorf("Relinked = %d, want 1", res.Relinked)
	}
	r := s.row(t, id)
	if r.SourcePath != s.stored("clip2.mp4") || r.FilePath != s.stored(hlsMasterRel) {
		t.Errorf("source_path=%q file_path=%q, want source à jour et file_path inchangé", r.SourcePath, r.FilePath)
	}
	if got := s.likeEvents(t); got != events {
		t.Errorf("événements de likes %d -> %d, want aucun INSERT", events, got)
	}
}

func TestReconcile_R12_ConvertedOriginalDeletedStaysVisible(t *testing.T) {
	s := newReconcileSetup(t)
	s.write(t, "clip.mp4", "a")
	id := s.hlsRow(t, "clip.mp4", "h:a")
	s.remove(t, "clip.mp4")
	res := s.run(t, MissingMark)
	r := s.row(t, id)
	if res.SourcesCleared != 1 || res.MarkedMissing != 0 {
		t.Errorf("SourcesCleared=%d MarkedMissing=%d, want 1 et 0", res.SourcesCleared, res.MarkedMissing)
	}
	if r.Status != "" || r.SourcePath != "" || r.SourceSize != -1 || r.FilePath != s.stored(hlsMasterRel) {
		t.Errorf("row = %+v, want visible, source NULL, file_path inchangé", r)
	}
}

func TestReconcile_R12b_ConvertedBothAbsent(t *testing.T) {
	s := newReconcileSetup(t)
	id := s.addRow(t, rowSpec{rel: hlsMasterRel, name: "clip.mp4", hash: "h:a", source: "clip.mp4", hls: hlsMasterRel})
	if res := s.run(t, MissingMark); res.MarkedMissing != 1 {
		t.Errorf("MarkedMissing = %d, want 1", res.MarkedMissing)
	}
	if r := s.row(t, id); r.Status != "missing" {
		t.Errorf("status = %q, want missing", r.Status)
	}
}

func TestReconcile_R12c_OriginalPutBackElsewhere(t *testing.T) {
	s := newReconcileSetup(t)
	s.write(t, "clip.mp4", "a")
	id := s.hlsRow(t, "clip.mp4", "h:a")
	s.remove(t, "clip.mp4")
	s.run(t, MissingMark)
	s.write(t, "back/orig.mp4", "a")
	s.run(t, MissingMark)
	r := s.row(t, id)
	if r.SourcePath != s.stored("back/orig.mp4") || r.FilePath != s.stored(hlsMasterRel) || r.Status != "" {
		t.Errorf("row = %+v, want source par empreinte, file_path inchangé", r)
	}
}

func TestReconcile_R13_ConvertedNoSourceNoOriginal(t *testing.T) {
	s := newReconcileSetup(t)
	s.hlsRow(t, "", "h:a")
	before := s.rows(t)
	res := s.run(t, MissingMark)
	if res.writes != 0 {
		t.Errorf("writes = %d, want 0 (cas serveur)", res.writes)
	}
	sameRows(t, before, s.rows(t))
}

func TestReconcile_R14_ConvertedNoSourceOriginalPresent(t *testing.T) {
	s := newReconcileSetup(t)
	id := s.hlsRow(t, "", "h:a")
	s.write(t, "clip.mp4", "a")
	s.run(t, MissingMark)
	if r := s.row(t, id); r.SourcePath != s.stored("clip.mp4") || r.FilePath != s.stored(hlsMasterRel) {
		t.Errorf("row = %+v, want source_path renseigné", r)
	}
}

// bulkRows crée n lignes (fichier présent, original renseigné) et retire le
// fichier des absent premières.
func (s *reconcileSetup) bulkRows(t *testing.T, n, absent int) {
	t.Helper()
	for i := 0; i < n; i++ {
		rel := fmt.Sprintf("f%02d.mp4", i)
		s.write(t, rel, fmt.Sprintf("c%02d", i))
		s.addRow(t, rowSpec{rel: rel, hash: fmt.Sprintf("h:c%02d", i), source: rel})
	}
	for i := 0; i < absent; i++ {
		s.remove(t, fmt.Sprintf("f%02d.mp4", i))
	}
}

func (s *reconcileSetup) countStatus(t *testing.T, status string) int {
	t.Helper()
	n := 0
	for _, r := range s.rows(t) {
		if r.Status == status {
			n++
		}
	}
	return n
}

func TestReconcile_R15_BulkSuspended(t *testing.T) {
	s := newReconcileSetup(t)
	s.bulkRows(t, 20, 10)
	before := s.rows(t)
	res := s.run(t, MissingMark)
	if res.BulkPending != 10 || res.MarkedMissing != 0 {
		t.Errorf("BulkPending=%d MarkedMissing=%d, want 10 et 0", res.BulkPending, res.MarkedMissing)
	}
	sameRows(t, before, s.rows(t))
}

func TestReconcile_R15b_BulkBelowThresholdTwoPasses(t *testing.T) {
	s := newReconcileSetup(t)
	s.bulkRows(t, 40, 14)
	first := s.run(t, MissingMark)
	if first.MarkedMissing != 14 || first.BulkPending != 0 || s.countStatus(t, "missing") != 14 {
		t.Errorf("1re passe: %+v missing=%d, want 14 marquées", first, s.countStatus(t, "missing"))
	}
	second := s.run(t, MissingMark)
	if second.BulkPending != 0 || second.writes != 0 {
		t.Errorf("2e passe: BulkPending=%d writes=%d, want 0 et 0", second.BulkPending, second.writes)
	}
}

func TestReconcile_R15c_BulkConfirmed(t *testing.T) {
	s := newReconcileSetup(t)
	s.bulkRows(t, 20, 10)
	res := s.run(t, MissingMark, func(in *ReconcileInput) { in.AllowBulk = true })
	if res.MarkedMissing != 10 || res.BulkPending != 0 || s.countStatus(t, "missing") != 10 {
		t.Errorf("res=%+v missing=%d, want 10 marquées", res, s.countStatus(t, "missing"))
	}
}

func TestReconcile_R16_BelowHalf(t *testing.T) {
	s := newReconcileSetup(t)
	s.bulkRows(t, 20, 9)
	res := s.run(t, MissingMark)
	if res.MarkedMissing != 9 || res.BulkPending != 0 || s.countStatus(t, "missing") != 9 {
		t.Errorf("res=%+v missing=%d, want 9 marquées", res, s.countStatus(t, "missing"))
	}
}

func TestReconcile_R17_PoliciesThatNeverMark(t *testing.T) {
	for name, policy := range map[string]MissingPolicy{"ignore": MissingIgnore, "root-unavailable": MissingRootUnavailable} {
		t.Run(name, func(t *testing.T) {
			s := newReconcileSetup(t)
			s.bulkRows(t, 20, 10)
			before := s.rows(t)
			res := s.run(t, policy)
			if res.BulkPending != 0 || res.MarkedMissing != 0 {
				t.Errorf("BulkPending=%d MarkedMissing=%d, want 0 et 0", res.BulkPending, res.MarkedMissing)
			}
			sameRows(t, before, s.rows(t))
		})
	}
}

func TestReconcile_R18_OtherPlayerUntouched(t *testing.T) {
	s := newReconcileSetup(t)
	idOther := s.addRow(t, rowSpec{player: "Other", rel: "clip.mp4", hash: "h:a"})
	s.addRow(t, rowSpec{rel: "gone.mp4", hash: "h:z"})
	s.write(t, "new.mp4", "a")
	before := s.row(t, idOther)
	s.run(t, MissingMark)
	if after := s.row(t, idOther); after != before {
		t.Errorf("ligne de l'autre joueur modifiée: %+v -> %+v", before, after)
	}
}

func TestReconcile_R19_CopyNotClaimed(t *testing.T) {
	s := newReconcileSetup(t)
	s.write(t, "clip.mp4", "a")
	s.write(t, "copy.mp4", "a")
	s.addRow(t, rowSpec{rel: "clip.mp4", hash: "h:a", source: "clip.mp4"})
	before := s.rows(t)
	res := s.run(t, MissingMark)
	if res.writes != 0 {
		t.Errorf("writes = %d, want 0 (la copie ne déplace pas la ligne)", res.writes)
	}
	sameRows(t, before, s.rows(t))
	if _, ok := res.Hashes[pathKey(s.abs("copy.mp4"))]; !ok {
		t.Error("la copie devrait figurer dans Hashes (elle est indexable)")
	}
}

func TestReconcile_R20_DeletedNeverTouched(t *testing.T) {
	s := newReconcileSetup(t)
	s.addRow(t, rowSpec{rel: "clip.mp4", hash: "h:a", status: "deleted"})
	s.write(t, "elsewhere/x.mp4", "a")
	before := s.rows(t)
	res := s.run(t, MissingMark)
	if res.writes != 0 {
		t.Errorf("writes = %d, want 0", res.writes)
	}
	sameRows(t, before, s.rows(t))
}

func TestReconcile_R21_FreshFileSkipped(t *testing.T) {
	s := newReconcileSetup(t)
	s.write(t, "clip.mp4", "a")
	s.addRow(t, rowSpec{rel: "clip.mp4", hash: "h:a"})
	now := time.Now()
	if err := chtimes(s.abs("clip.mp4"), now); err != nil {
		t.Fatal(err)
	}
	before := s.rows(t)
	res := s.run(t, MissingMark, func(in *ReconcileInput) { in.SkipFresh = true; in.Now = now.Add(10 * time.Second) })
	key := pathKey(s.abs("clip.mp4"))
	if s.hashCalls != 0 || !res.Fresh[key] {
		t.Errorf("hashCalls=%d Fresh=%v, want 0 et fichier dans Fresh", s.hashCalls, res.Fresh[key])
	}
	if _, hashed := res.Hashes[key]; hashed {
		t.Error("un fichier frais ne doit pas avoir d'empreinte")
	}
	sameRows(t, before, s.rows(t))
}

func TestReconcile_R22_ModifiedInPlace(t *testing.T) {
	s := newReconcileSetup(t)
	s.write(t, "clip.mp4", "a")
	id := s.addRow(t, rowSpec{rel: "clip.mp4", hash: "h:a", source: "clip.mp4"})
	before := s.row(t, id)
	s.write(t, "clip.mp4", "bb")
	if err := chtimes(s.abs("clip.mp4"), time.Now().Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	s.run(t, MissingMark)
	after := s.row(t, id)
	if after.SourceSize != 2 || after.SourceMtime == before.SourceMtime || after.FileHash != "h:bb" {
		t.Errorf("row = %+v (avant %+v), want taille 2, date et empreinte à jour", after, before)
	}
}

func TestReconcile_R23_IdempotentAfterRename(t *testing.T) {
	s := newReconcileSetup(t)
	s.write(t, "clip.mp4", "a")
	s.addRow(t, rowSpec{rel: "clip.mp4", hash: "h:a", source: "clip.mp4"})
	s.move(t, "clip.mp4", "renamed.mp4")
	s.run(t, MissingMark)
	before := s.rows(t)
	res := s.run(t, MissingMark)
	if res.writes != 0 {
		t.Errorf("writes = %d, want 0 (idempotence)", res.writes)
	}
	sameRows(t, before, s.rows(t))
}

func TestReconcile_R24_ConvertedMasterAbsentRevertsToDirect(t *testing.T) {
	s := newReconcileSetup(t)
	s.write(t, "clip.mp4", "a")
	id := s.addRow(t, rowSpec{rel: hlsMasterRel, name: "clip.mp4", hash: "h:a", source: "clip.mp4", hls: hlsMasterRel})
	master := s.stored(hlsMasterRel)
	s.like(t, master, "alice", true)
	s.like(t, master, "bob", true)
	if res := s.run(t, MissingMark); res.Reverted != 1 {
		t.Errorf("Reverted = %d, want 1", res.Reverted)
	}
	r := s.row(t, id)
	if r.FilePath != r.SourcePath || r.FilePath != s.stored("clip.mp4") || r.HLSPath != "" || r.Transcode != "" {
		t.Errorf("row = %+v, want lecture directe sur l'original", r)
	}
	if got := s.likers(t, s.stored("clip.mp4")); !reflect.DeepEqual(got, []string{"alice", "bob"}) {
		t.Errorf("likers = %v, want [alice bob] transférés", got)
	}
	if got := s.likers(t, master); len(got) != 0 {
		t.Errorf("likers de l'ancien chemin = %v, want aucun", got)
	}
}

func TestReconcile_R25_RowDeletedBetweenReadAndWrite(t *testing.T) {
	s := newReconcileSetup(t)
	s.write(t, "clip.mp4", "a")
	id := s.addRow(t, rowSpec{rel: "clip.mp4", hash: "h:a", source: "clip.mp4"})
	oldPath := s.stored("clip.mp4")
	s.like(t, oldPath, "alice", true)
	s.move(t, "clip.mp4", "renamed.mp4")
	s.onHash = func() {
		if _, err := s.db.Exec(`UPDATE media_files SET status = 'deleted' WHERE id = ?`, id); err != nil {
			t.Error(err)
		}
		s.onHash = nil
	}
	res := s.run(t, MissingMark)
	r := s.row(t, id)
	if r.Status != "deleted" || r.FilePath != oldPath {
		t.Errorf("row = %+v, want deleted et file_path inchangé", r)
	}
	if res.Relinked != 0 {
		t.Errorf("Relinked = %d, want 0 (aucune ligne affectée)", res.Relinked)
	}
	if got := s.likers(t, s.stored("renamed.mp4")); len(got) != 0 {
		t.Errorf("likers du nouveau chemin = %v, want aucun transfert", got)
	}
	if got := s.likers(t, oldPath); !reflect.DeepEqual(got, []string{"alice"}) {
		t.Errorf("likers de l'ancien chemin = %v, want [alice] intacts", got)
	}
}
