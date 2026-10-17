package filmproc

// solo_observe_test.go — LIRE LE DETENTEUR SANS JAMAIS TOUCHER AU VERROU.

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func ecrireDetenteur(t *testing.T, dir, contenu string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, soloHolderFileName), []byte(contenu), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestReadDecodeHolder(t *testing.T) {
	pidVivant := os.Getpid()
	cas := []struct {
		nom      string
		contenu  *string
		wantOK   bool
		wantID   string
		wantTool string
	}{
		{nom: "fichier absent"},
		{nom: "json illisible", contenu: ptr(`{pas du json`)},
		{nom: "fichier vide", contenu: ptr(``)},
		{nom: "pid absent", contenu: ptr(`{"tool":"x","matchId":"m1"}`)},
		{nom: "pid mort (aucun systeme n'attribue 2147483646)", contenu: ptr(`{"tool":"x","pid":2147483646,"matchId":"m1"}`)},
		{nom: "pid vivant, autre match", contenu: ptr(jsonDetenteur(pidVivant, "outil-a", "m-autre")),
			wantOK: true, wantID: "m-autre", wantTool: "outil-a"},
		{nom: "pid vivant, ce match", contenu: ptr(jsonDetenteur(pidVivant, "outil-b", "m1")),
			wantOK: true, wantID: "m1", wantTool: "outil-b"},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			dir := t.TempDir()
			if c.contenu != nil {
				ecrireDetenteur(t, dir, *c.contenu)
			}
			h, ok := ReadDecodeHolder(dir)
			if ok != c.wantOK || h.MatchID != c.wantID || h.Tool != c.wantTool {
				t.Fatalf("ReadDecodeHolder = (%+v, %v), want ok=%v match=%q outil=%q", h, ok, c.wantOK, c.wantID, c.wantTool)
			}
			if got := NewDecodeObserver(dir).DecodingMatchID(); got != c.wantID {
				t.Fatalf("DecodingMatchID = %q, want %q", got, c.wantID)
			}
		})
	}
}

func TestReadDecodeHolder_RacineVideEtObservateurNil(t *testing.T) {
	if _, ok := ReadDecodeHolder(""); ok {
		t.Fatal("racine vide : aucun detenteur")
	}
	var o *DecodeObserver
	if o.DecodingMatchID() != "" {
		t.Fatal("observateur nil : aucun match")
	}
}

// Le verrou reel : tant qu'un decodeur le tient, l'observateur le nomme ; la lecture ne prend
// jamais le verrou (un second decodeur reste refuse, et la liberation est constatee).
func TestDecodeObserver_SuitLeVerrouReel(t *testing.T) {
	dir := t.TempDir()
	obs := NewDecodeObserver(dir)
	if got := obs.DecodingMatchID(); got != "" {
		t.Fatalf("rien ne decode : %q", got)
	}
	lock, err := AcquireSolo(dir, "outil-test", "match-reel")
	if err != nil {
		t.Fatal(err)
	}
	if got := obs.DecodingMatchID(); got != "match-reel" {
		t.Fatalf("decodage en cours : %q", got)
	}
	// La lecture n'a pas pris le verrou : il est toujours tenu par le premier.
	if second, err := AcquireSolo(dir, "outil-2", "autre"); err == nil {
		second.Release()
		t.Fatal("le verrou aurait du rester tenu")
	}
	lock.Release()
	if got := obs.DecodingMatchID(); got != "" {
		t.Fatalf("verrou rendu : %q", got)
	}
	// Et un decodeur peut le reprendre : l'observation ne l'a pas retenu.
	lock2, err := AcquireSolo(dir, "outil-3", "m3")
	if err != nil {
		t.Fatalf("reprise du verrou apres observation: %v", err)
	}
	lock2.Release()
}

func ptr(s string) *string { return &s }

func jsonDetenteur(pid int, outil, match string) string {
	return fmt.Sprintf(`{"tool":%q,"pid":%d,"matchId":%q,"startedAt":"2026-10-17T10:00:00Z"}`, outil, pid, match)
}
