//go:build cgo

package ops

// media_reconcile_deleted_cgo_test.go — aucune écriture de la passe de réconciliation ne
// touche une fiche supprimée : chaque requête de media_reconcile_apply.go, exécutée sur une
// ligne 'deleted', n'affecte aucune ligne et la laisse intacte.

import (
	"context"
	"testing"

	"levelup/go-api/internal/domain"
)

func TestReconcileWrites_NeverTouchADeletedRow(t *testing.T) {
	cases := []struct {
		name string
		op   reconcileOp
	}{
		{"relocalisation d'un fichier non converti", reconcileOp{kind: opRelocate, path: "p/neuf.mp4", size: 9, mtime: 9, name: "neuf.mp4", ext: ".mp4", hash: "h:neuf"}},
		{"relocalisation d'une vidéo convertie", reconcileOp{kind: opRelocate, path: "p/neuf.mp4", size: 9, mtime: 9, name: "neuf.mp4", ext: ".mp4", hash: "h:neuf", hasHLS: true}},
		{"rattachement de l'original", reconcileOp{kind: opAttachSource, path: "p/neuf.mp4", size: 9, mtime: 9}},
		{"relevé de l'original", reconcileOp{kind: opRefreshSource, size: 9, mtime: 9, hash: "h:neuf"}},
		{"détachement de l'original", reconcileOp{kind: opClearSource}},
		{"retour en lecture directe", reconcileOp{kind: opRevert}},
		{"retour du fichier", reconcileOp{kind: opRestore}},
		{"marquage introuvable", reconcileOp{kind: opMarkMissing}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := newReconcileSetup(t)
			s.write(t, "clip.mp4", "a")
			s.write(t, hlsMasterRel, hlsMasterTxt)
			id := s.addRow(t, rowSpec{
				rel: hlsMasterRel, name: "clip.mp4", hash: "h:a", source: "clip.mp4",
				hls: hlsMasterRel, status: domain.MediaStatusDeleted,
			})
			ctx := context.Background()
			snapshot := func() string {
				var out string
				if err := s.db.QueryRowContext(ctx, `SELECT CAST(t AS VARCHAR) FROM media_files t WHERE id = ?`, id).Scan(&out); err != nil {
					t.Fatal(err)
				}
				return out
			}
			before := snapshot()

			r := &reconciler{ctx: ctx, db: s.db}
			op := c.op
			op.id = id
			tx, err := s.db.BeginTx(ctx, nil)
			if err != nil {
				t.Fatal(err)
			}
			res, err := r.execOp(tx, op)
			if err != nil {
				t.Fatalf("execOp: %v", err)
			}
			n, err := res.RowsAffected()
			if err != nil {
				t.Fatal(err)
			}
			if err := tx.Commit(); err != nil {
				t.Fatal(err)
			}
			if n != 0 {
				t.Errorf("%d ligne(s) affectée(s), want 0 sur une fiche supprimée", n)
			}
			if after := snapshot(); after != before {
				t.Errorf("la fiche supprimée a changé :\n avant %s\n après %s", before, after)
			}
		})
	}
}
