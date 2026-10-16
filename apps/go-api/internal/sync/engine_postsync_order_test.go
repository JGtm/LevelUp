package sync

import (
	"context"
	"slices"
	"testing"

	"levelup/go-api/internal/domain"
)

// ordrePostSyncAttendu : l'ordre d'exécution du pipeline post-sync. Il est écrit ICI, en
// littéral, et non relu de postSyncStepNames : le test échoue si la liste publiée ou
// l'ordre réel des étapes dérive. Les contraintes de dépendance qui le fixent sont dans le
// commentaire « ORDRE DES ÉTAPES » de runPostSyncPipeline.
var ordrePostSyncAttendu = []string{
	"enrichment_rows",
	"scoring",
	"convergence_events",
	"convergence_psa",
	"friends",
	"dominance",
	"skill_rating",
	"csr_snapshots",
	"aggregates",
	"kill_source",
	"replay_artifacts",
	"citations",
	"media_scan",
	"achievements",
	"snapshot_readiness",
}

func etapesDuResultat(r domain.PostSyncResult) []string {
	out := make([]string, 0, len(r.StepTimings))
	for _, st := range r.StepTimings {
		out = append(out, st.Step)
	}
	return out
}

// TestRunPostSyncPipeline_StepOrder épingle l'ordre des étapes : escouade, LUSR, CSR et
// agrégats AVANT la source du kill (étape film lente), citations APRÈS le rejeu, préparation
// du match en dernier.
func TestRunPostSyncPipeline_StepOrder(t *testing.T) {
	playerDB, sharedDB := newInMemoryDBs(t)
	e := &SyncEngine{gamertag: "TestPlayer", xuid: "1234567890123456"}

	r := e.runPostSyncPipeline(context.Background(), playerDB, NewPinnedSharedAccess(sharedDB), &mockHaloClient{}, nil)

	got := etapesDuResultat(r)
	if !slices.Equal(got, ordrePostSyncAttendu) {
		t.Fatalf("ordre des étapes du post-sync\n got  %v\n want %v", got, ordrePostSyncAttendu)
	}
	if pub := PostSyncStepNames(); !slices.Equal(pub, ordrePostSyncAttendu) {
		t.Fatalf("PostSyncStepNames (étapes publiées au tableau de bord) ne suit pas l'ordre d'exécution\n got  %v\n want %v",
			pub, ordrePostSyncAttendu)
	}
}
