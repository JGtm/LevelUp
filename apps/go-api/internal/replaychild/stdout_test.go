package replaychild

// stdout_test.go — en -mode desktop, la coquille lit la sortie standard du serveur ligne a ligne
// comme des evenements JSON (ADR 0045 D-2) : une cuisson lancee par le serveur ne doit RIEN y
// ecrire. Le binaire de test joue l'enfant (TestMain), comme le serveur re-execute son binaire.

import (
	"bytes"
	"context"
	"io"
	"os"
	"testing"
)

// TestMain : lance avec le drapeau de l'enfant, le binaire de test EST l'enfant de cuisson.
func TestMain(m *testing.M) {
	if IsChild(os.Args) {
		os.Exit(RunChild(context.Background(), os.Args[1:]))
	}
	os.Exit(m.Run())
}

// capture redirige *target vers un tube et rend la fonction qui restitue ce qui y a ete ecrit.
func capture(t *testing.T, target **os.File) func() string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	saved := *target
	*target = w
	done := make(chan string)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		done <- buf.String()
	}()
	return func() string {
		*target = saved
		_ = w.Close()
		return <-done
	}
}

// Une cuisson (ici refusee par l'enfant : aucun film) ne laisse rien sur la sortie standard du
// parent ; ce que l'enfant dit passe par la sortie d'erreur, lignes de protocole exclues.
func TestSpawn_NEcritRienSurLaSortieStandard(t *testing.T) {
	stdout := capture(t, &os.Stdout)
	stderr := capture(t, &os.Stderr)

	_, err := Spawn(context.Background(), Request{
		RepoRoot: t.TempDir(), TitleSlug: "halo_infinite",
		MatchID: "64e8adfa-0000-0000-0000-000000000000", MapNames: []string{"Catalyst"},
	})

	out, errOut := stdout(), stderr()
	if err == nil {
		t.Fatal("la cuisson d'un film inexistant a reussi")
	}
	if out != "" {
		t.Errorf("la sortie standard du parent porte %q : en -mode desktop elle ne porte que des evenements", out)
	}
	if errOut == "" {
		t.Error("le compte rendu de l'enfant n'a pas ete relaye sur la sortie d'erreur")
	}
	if bytes.Contains([]byte(errOut), []byte("__levelup_")) {
		t.Errorf("une ligne de protocole a ete relayee : %q", errOut)
	}
}
