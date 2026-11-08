package ops

import "testing"

func TestRootState_RecordAndGet(t *testing.T) {
	if got := GetRootState("t-root", "nobody"); got != (RootState{}) {
		t.Errorf("état inconnu = %+v, want zéro", got)
	}
	RecordRootState("t-root", "p1", RootState{Unavailable: true})
	RecordRootState("t-root", "p2", RootState{BulkPending: 7})
	if got := GetRootState("t-root", "p1"); !got.Unavailable || got.BulkPending != 0 {
		t.Errorf("p1 = %+v, want {true 0}", got)
	}
	if got := GetRootState("t-root", "p2"); got.Unavailable || got.BulkPending != 7 {
		t.Errorf("p2 = %+v, want {false 7}", got)
	}
	if got := GetRootState("autre-titre", "p1"); got != (RootState{}) {
		t.Errorf("même joueur, autre titre = %+v, want zéro (clé = titre et joueur)", got)
	}
	RecordRootState("t-root", "p1", RootState{})
	if got := GetRootState("t-root", "p1"); got != (RootState{}) {
		t.Errorf("p1 après remise à zéro = %+v", got)
	}
}

func TestRootState_ConfirmationIsConsumedOnce(t *testing.T) {
	if ConsumeBulkConfirmation("t-conf", "p") {
		t.Fatal("aucune confirmation en attente, want false")
	}
	ConfirmBulkMissing("t-conf", "p")
	if ConsumeBulkConfirmation("t-conf", "autre") {
		t.Error("la confirmation d'un joueur ne vaut pas pour un autre")
	}
	if !ConsumeBulkConfirmation("t-conf", "p") {
		t.Fatal("confirmation en attente, want true")
	}
	if ConsumeBulkConfirmation("t-conf", "p") {
		t.Error("une confirmation ne vaut que pour une passe, want false à la lecture suivante")
	}
}

func TestFilterForeignMedia(t *testing.T) {
	files := []string{
		"/c/Halo_5_Guardians-1.mp4", "/c/Halo Infinite 1.mp4", "/c/sub/halo_5_guardians-2.mkv",
	}
	kept, skipped := filterForeignMedia(files, MediaIndexOptions{ForeignFilenamePrefixes: []string{"Halo_5_Guardians"}})
	if skipped != 2 || len(kept) != 1 || kept[0] != "/c/Halo Infinite 1.mp4" {
		t.Errorf("kept=%v skipped=%d, want un fichier gardé et deux écartés", kept, skipped)
	}
	kept, skipped = filterForeignMedia(files, MediaIndexOptions{ForeignFilenamePrefixes: []string{}})
	if skipped != 0 || len(kept) != 3 {
		t.Errorf("sans préfixe : kept=%v skipped=%d, want tout gardé", kept, skipped)
	}
}
