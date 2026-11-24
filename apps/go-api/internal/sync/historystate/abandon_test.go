package historystate_test

import (
	"context"
	"testing"
	"time"

	"levelup/go-api/internal/sync/historystate"
	"levelup/go-api/internal/sync/syncmeta"
)

// fakeClock remplace l'horloge du paquet ; le rendu avance d'une durée.
func fakeClock(t *testing.T) (advance func(time.Duration)) {
	t.Helper()
	now := time.Date(2026, 11, 22, 10, 0, 0, 0, time.UTC)
	old := historystate.Now
	historystate.Now = func() time.Time { return now }
	t.Cleanup(func() { historystate.Now = old })
	return func(d time.Duration) { now = now.Add(d) }
}

func TestAbandonRetryAfter_Vaut24h(t *testing.T) {
	if historystate.AbandonRetryAfter != 24*time.Hour {
		t.Errorf("AbandonRetryAfter = %v, want 24h (décision de l'utilisateur)", historystate.AbandonRetryAfter)
	}
}

// Un abandon échoit après 24 h : il sort de l'ensemble actif et passe dans les échus ; un introuvable
// reste actif pour de bon.
func TestSkipSet_AbandonEchoitApres24h_IntrouvableJamais(t *testing.T) {
	advance := fakeClock(t)
	db := openMemDB(t, true)
	ctx := context.Background()
	historystate.AddSkips(ctx, db, []historystate.Skip{
		{MatchID: "abandonne", Reason: historystate.SkipAbandoned},
		{MatchID: "introuvable", Reason: historystate.SkipNotFound},
	})

	advance(historystate.AbandonRetryAfter - time.Second)
	if got := historystate.LoadSkipSet(ctx, db); !got.Active["abandonne"] || !got.Active["introuvable"] || len(got.Due) != 0 {
		t.Fatalf("avant 24 h : %+v, want les deux actifs et aucun échu", got)
	}
	advance(2 * time.Second)
	got := historystate.LoadSkipSet(ctx, db)
	if got.Active["abandonne"] || !got.Active["introuvable"] || len(got.Due) != 1 || got.Due[0] != "abandonne" {
		t.Fatalf("après 24 h : %+v, want l'abandonné échu et l'introuvable actif", got)
	}
	advance(100 * historystate.AbandonRetryAfter)
	if got := historystate.LoadSkipSet(ctx, db); !got.Active["introuvable"] {
		t.Errorf("longtemps après : %+v, un introuvable reste actif pour de bon", got)
	}
}

// Une entrée sans date (format d'avant) vaut la date de la dernière écriture ; sans date du tout, elle
// n'échoit pas (abandonnée « maintenant »).
func TestSkipSet_EntreeSansDate(t *testing.T) {
	advance := fakeClock(t)
	ctx := context.Background()
	saved := historystate.Now().Format(time.RFC3339)

	withSaved := openMemDB(t, true)
	if err := syncmeta.Set(ctx, withSaved, historystate.SkipSetKey,
		`{"skipped":[{"id":"a","why":"abandoned"},{"id":"b","why":"not_found"}],"saved_at":"`+saved+`"}`); err != nil {
		t.Fatal(err)
	}
	if got := historystate.LoadSkipSet(ctx, withSaved); !got.Active["a"] || len(got.Due) != 0 {
		t.Fatalf("juste après l'écriture : %+v, want a actif", got)
	}
	advance(historystate.AbandonRetryAfter + time.Minute)
	if got := historystate.LoadSkipSet(ctx, withSaved); len(got.Due) != 1 || got.Due[0] != "a" || !got.Active["b"] {
		t.Fatalf("24 h après l'écriture : %+v, want a échu et b actif", got)
	}

	noDate := openMemDB(t, true)
	if err := syncmeta.Set(ctx, noDate, historystate.SkipSetKey, `{"skipped":[{"id":"a","why":"abandoned"}]}`); err != nil {
		t.Fatal(err)
	}
	advance(1000 * historystate.AbandonRetryAfter)
	if got := historystate.LoadSkipSet(ctx, noDate); !got.Active["a"] || len(got.Due) != 0 {
		t.Errorf("sans aucune date : %+v, want a actif (abandonné maintenant, n'échoit pas)", got)
	}
}

// Un abandon retenté puis abandonné de nouveau renouvelle sa date ; un introuvable rend l'abandon définitif ;
// les entrées sans date reçoivent celle de la dernière écriture.
func TestAddSkips_RenouvelleLAbandon_IntrouvableLeRendDefinitif(t *testing.T) {
	advance := fakeClock(t)
	db := openMemDB(t, true)
	ctx := context.Background()
	historystate.AddSkips(ctx, db, []historystate.Skip{
		{MatchID: "a", Reason: historystate.SkipAbandoned},
		{MatchID: "b", Reason: historystate.SkipAbandoned},
	})

	advance(historystate.AbandonRetryAfter + time.Hour)
	if got := historystate.LoadSkipSet(ctx, db); len(got.Due) != 2 {
		t.Fatalf("échus %v, want a et b", got.Due)
	}
	historystate.AddSkips(ctx, db, []historystate.Skip{
		{MatchID: "a", Reason: historystate.SkipAbandoned}, // retenté en vain : date renouvelée
		{MatchID: "b", Reason: historystate.SkipNotFound},  // introuvable au retour : définitif
	})
	got := historystate.LoadSkipSet(ctx, db)
	if !got.Active["a"] || !got.Active["b"] || len(got.Due) != 0 {
		t.Fatalf("après renouvellement : %+v, want a et b actifs", got)
	}
	advance(historystate.AbandonRetryAfter + time.Hour)
	got = historystate.LoadSkipSet(ctx, db)
	if len(got.Due) != 1 || got.Due[0] != "a" || !got.Active["b"] {
		t.Errorf("24 h après : %+v, want a échu de nouveau et b toujours actif (introuvable)", got)
	}
}

// Une clé vide rangée par Reopen n'est pas une fin d'historique ; Reopen efface aussi marque-page et trou.
func TestReopen_RetireLaFinEtLEtat(t *testing.T) {
	db := openMemDB(t, true)
	ctx := context.Background()
	historystate.MarkComplete(ctx, db, "gt")
	historystate.SaveCursor(ctx, db, historystate.Cursor{Offset: 10, LastMatchID: "m-10"})
	historystate.SaveStreak(ctx, db, historystate.Streak{Subject: "m-10", Attempts: 2})
	historystate.NoteHole(ctx, db)
	if !mustComplete(t, ctx, db) {
		t.Fatal("préparation : la fin devait être rangée")
	}
	if !historystate.Reopen(ctx, db, "gt") {
		t.Fatal("Reopen doit rendre vrai")
	}
	if mustComplete(t, ctx, db) {
		t.Error("après Reopen : Complete = vrai, want faux (clé vide)")
	}
	if _, ok := historystate.LoadCursor(ctx, db); ok {
		t.Error("le marque-page doit être effacé")
	}
	if s := historystate.LoadStreak(ctx, db, "m-10"); s.Attempts != 0 {
		t.Errorf("compte d'échecs %+v, want effacé", s)
	}
	if historystate.HoleDue(ctx, db) {
		t.Error("le trou doit être effacé")
	}
	// Une fin rangée de nouveau prime.
	historystate.MarkComplete(ctx, db, "gt")
	if !mustComplete(t, ctx, db) {
		t.Error("après un nouveau MarkComplete : Complete = faux, want vrai")
	}
}

// Le trou le plus ancien date l'échéance ; un second trou ne la repousse pas.
func TestHole_NoteEtEcheance(t *testing.T) {
	advance := fakeClock(t)
	db := openMemDB(t, true)
	ctx := context.Background()
	if historystate.HoleDue(ctx, db) {
		t.Fatal("aucun trou noté : HoleDue doit être faux")
	}
	if !historystate.NoteHole(ctx, db) {
		t.Fatal("NoteHole doit rendre vrai")
	}
	advance(historystate.AbandonRetryAfter - time.Hour)
	historystate.NoteHole(ctx, db) // second trou : n'avance pas l'échéance
	if historystate.HoleDue(ctx, db) {
		t.Fatal("avant 24 h : HoleDue doit être faux")
	}
	advance(2 * time.Hour)
	if !historystate.HoleDue(ctx, db) {
		t.Error("24 h après le premier trou : HoleDue doit être vrai")
	}
	if historystate.NoteHole(ctx, nil) || historystate.HoleDue(ctx, nil) {
		t.Error("base absente : NoteHole et HoleDue doivent rendre faux")
	}
}
