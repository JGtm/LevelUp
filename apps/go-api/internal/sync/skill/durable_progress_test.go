package skill

import (
	"context"
	"errors"
	"testing"
)

// TestCommitThenAdvance couvre les 4 cas de l'invariant durable-avant-progrès,
// en isolation (closures fake, aucune DB). Prouve qu'AdvanceProgress n'est jamais
// appelé sans écriture durable réussie, et la sémantique Held vs Err.
func TestCommitThenAdvance(t *testing.T) {
	ctx := context.Background()
	errBoom := errors.New("boom")

	t.Run("succès complet → Advanced, advance appelé, groupe non tenu", func(t *testing.T) {
		advanced := false
		held := map[string]bool{}
		out := CommitThenAdvance(ctx, held, DurableStep{
			GroupKey:        "g",
			WriteDurable:    func(context.Context) error { return nil },
			AdvanceProgress: func(context.Context) error { advanced = true; return nil },
		})
		if !out.Advanced || out.Held || out.Err != nil {
			t.Fatalf("out = %+v, attendu Advanced", out)
		}
		if !advanced {
			t.Error("AdvanceProgress aurait dû être appelé")
		}
		if held["g"] {
			t.Error("le groupe ne doit pas être tenu sur succès")
		}
	})

	t.Run("échec durable → Held, advance JAMAIS appelé, groupe tenu", func(t *testing.T) {
		advanceCalled := false
		held := map[string]bool{}
		out := CommitThenAdvance(ctx, held, DurableStep{
			GroupKey:        "g",
			WriteDurable:    func(context.Context) error { return errBoom },
			AdvanceProgress: func(context.Context) error { advanceCalled = true; return nil },
		})
		if !out.Held || out.Advanced || out.Err == nil {
			t.Fatalf("out = %+v, attendu Held+Err", out)
		}
		if advanceCalled {
			t.Error("AdvanceProgress NE doit PAS être appelé après un échec durable")
		}
		if !held["g"] {
			t.Error("le groupe doit être tenu après un échec durable")
		}
	})

	t.Run("groupe déjà tenu → skip total (rien appelé)", func(t *testing.T) {
		writeCalled, advanceCalled := false, false
		held := map[string]bool{"g": true}
		out := CommitThenAdvance(ctx, held, DurableStep{
			GroupKey:        "g",
			WriteDurable:    func(context.Context) error { writeCalled = true; return nil },
			AdvanceProgress: func(context.Context) error { advanceCalled = true; return nil },
		})
		if !out.Held || out.Advanced {
			t.Fatalf("out = %+v, attendu Held", out)
		}
		if writeCalled || advanceCalled {
			t.Error("aucune closure ne doit être appelée si le groupe est déjà tenu")
		}
	})

	t.Run("échec advance après durable OK → Err sans Held (retry idempotent)", func(t *testing.T) {
		held := map[string]bool{}
		out := CommitThenAdvance(ctx, held, DurableStep{
			GroupKey:        "g",
			WriteDurable:    func(context.Context) error { return nil },
			AdvanceProgress: func(context.Context) error { return errBoom },
		})
		if out.Advanced || out.Held || out.Err == nil {
			t.Fatalf("out = %+v, attendu Err sans Held", out)
		}
		if held["g"] {
			t.Error("le groupe NE doit PAS être tenu : l'écriture durable a réussi")
		}
	})
}
