package startupentry

import (
	"errors"
	"testing"
)

// fakeStore : un registre simulé (valeurs Run et approbations) qui note ses suppressions.
type fakeStore struct {
	run       map[string]string
	approved  map[string]bool
	deleted   []string
	readErr   error
	deleteErr error
}

func newFake(run string) *fakeStore {
	f := &fakeStore{run: map[string]string{}, approved: map[string]bool{}}
	if run != "" {
		f.run[EntryName] = run
		f.approved[EntryName] = true
	}
	return f
}

func (f *fakeStore) RunValue(name string) (string, bool, error) {
	if f.readErr != nil {
		return "", false, f.readErr
	}
	v, ok := f.run[name]
	return v, ok, nil
}

func (f *fakeStore) DeleteRunValue(name string) error {
	if f.deleteErr != nil {
		return f.deleteErr
	}
	delete(f.run, name)
	f.deleted = append(f.deleted, "run")
	return nil
}

func (f *fakeStore) DeleteApprovedValue(name string) error {
	delete(f.approved, name)
	f.deleted = append(f.deleted, "approved")
	return nil
}

const installDir = `C:\Users\Eve\AppData\Local\Programs\LevelUp`

func TestRemoveForInstall_RemovesTheEntryOfThisInstall(t *testing.T) {
	f := newFake(`"` + installDir + `\levelup-desktop.exe" --minimized`)
	out, err := RemoveForInstall(f, installDir)
	if err != nil || out != OutcomeRemoved {
		t.Fatalf("issue = %q, err = %v ; attendu %q", out, err, OutcomeRemoved)
	}
	if _, ok := f.run[EntryName]; ok || f.approved[EntryName] {
		t.Fatalf("valeur Run ou approbation encore présente : run=%v approved=%v", f.run, f.approved)
	}
}

func TestRemoveForInstall_AbsentWritesNothing(t *testing.T) {
	f := newFake("")
	out, err := RemoveForInstall(f, installDir)
	if err != nil || out != OutcomeAbsent {
		t.Fatalf("issue = %q, err = %v ; attendu %q", out, err, OutcomeAbsent)
	}
	if len(f.deleted) != 0 {
		t.Fatalf("suppressions %v sans entrée", f.deleted)
	}
}

func TestRemoveForInstall_ForeignEntryIsKept(t *testing.T) {
	f := newFake(`"D:\dev\LevelUp\levelup-desktop.exe" --minimized`)
	out, err := RemoveForInstall(f, installDir)
	if err != nil || out != OutcomeForeign {
		t.Fatalf("issue = %q, err = %v ; attendu %q", out, err, OutcomeForeign)
	}
	if len(f.deleted) != 0 || f.run[EntryName] == "" {
		t.Fatalf("l'entrée d'un autre dossier a été touchée : %v", f.deleted)
	}
}

func TestRemoveForInstall_Errors(t *testing.T) {
	boom := errors.New("accès refusé")
	read := newFake("x")
	read.readErr = boom
	if _, err := RemoveForInstall(read, installDir); !errors.Is(err, boom) {
		t.Fatalf("lecture : err = %v", err)
	}
	del := newFake(`"` + installDir + `\levelup-desktop.exe"`)
	del.deleteErr = boom
	if _, err := RemoveForInstall(del, installDir); !errors.Is(err, boom) {
		t.Fatalf("suppression : err = %v", err)
	}
	if len(del.deleted) != 0 {
		t.Fatalf("l'approbation ne se retire pas quand la valeur Run reste : %v", del.deleted)
	}
}

func TestCommandTargets(t *testing.T) {
	cases := []struct {
		name, command, dir string
		want               bool
	}{
		{"entre guillemets avec argument", `"` + installDir + `\levelup-desktop.exe" --minimized`, installDir, true},
		{"sans guillemets", installDir + `\levelup-desktop.exe`, installDir, true},
		{"casse et barres obliques ignorées", `"c:/users/eve/appdata/local/programs/levelup/LevelUp-Desktop.exe"`, installDir, true},
		{"dossier avec barre finale", `"` + installDir + `\levelup-desktop.exe"`, installDir + `\`, true},
		{"dossier voisin de même préfixe", `"` + installDir + `2\levelup-desktop.exe"`, installDir, false},
		{"autre dossier", `"D:\LevelUp\levelup-desktop.exe"`, installDir, false},
		{"dossier vide", `"` + installDir + `\levelup-desktop.exe"`, "", false},
		{"commande vide", "", installDir, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := CommandTargets(c.command, c.dir); got != c.want {
				t.Fatalf("CommandTargets(%q, %q) = %v, attendu %v", c.command, c.dir, got, c.want)
			}
		})
	}
}
