package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// artefactMinimal : un match d'un joueur, deux vies, une mort ; `vieEnTrop` ajoute une vie qui
// recouvre la premiere (V-4, un FAUX).
func artefactMinimal(vieEnTrop bool) string {
	pistes := `{"slot":512,"xuid":"111","startFrame":0,"endFrame":100,"points":[{"t":0,"x":0,"y":0}]},
		{"slot":513,"xuid":"111","startFrame":150,"endFrame":300,"points":[{"t":150,"x":0,"y":0}]}`
	if vieEnTrop {
		pistes += `,{"slot":514,"xuid":"111","startFrame":90,"endFrame":120,"points":[{"t":90,"x":0,"y":0}]}`
	}
	return `{"schemaVersion":76,"matchId":"m-1","frameCount":302,"frameIntervalMs":100,
		"bounds":{"minX":0,"minY":0,"maxX":10,"maxY":10},"tracks":[` + pistes + `],
		"coverage":{"continuousFire":{"packets":10,"closed":8}}}`
}

const faitsMinimaux = `{"players":[{"xuid":"111","kills":0,"deaths":1,"assists":0,"teamId":0}],"matchId":"m-1"}`

func ecrire(t *testing.T, dir, nom, contenu string) string {
	t.Helper()
	p := filepath.Join(dir, nom)
	if err := os.WriteFile(p, []byte(contenu), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

// TestLancer_CodesDeSortie : ok (0), un faux nouveau (1), un drapeau manquant (2).
func TestLancer_CodesDeSortie(t *testing.T) {
	dir := t.TempDir()
	avant := ecrire(t, dir, "a.json", artefactMinimal(false))
	apres := ecrire(t, dir, "b.json", artefactMinimal(true))
	faits := ecrire(t, dir, "f.facts.json", faitsMinimaux)
	oracle := ecrire(t, dir, "o.oracle.json", `{"matchId":"m-1","players":[{"xuid":"111","personalScore":0}]}`)
	vide := ecrire(t, dir, "v.oracle.json", `{"matchId":"m-1"}`)
	cas := []struct {
		nom  string
		args []string
		veut int
		dans string
	}{
		{"identique", []string{"-avant", avant, "-apres", avant, "-faits", faits, "-temoin", "t1"}, codeOK, "BANC DE VERITE — t1 : ok"},
		{"vie en trop", []string{"-avant", avant, "-apres", apres, "-faits", faits}, codeVerdict, "[FAUX] V-4 deux corps"},
		{"sans faits", []string{"-avant", avant, "-apres", apres}, codeUsage, ""},
		{"avec oracle", []string{"-avant", avant, "-apres", avant, "-faits", faits, "-oracle", oracle}, codeOK, "O-S3 score personnel"},
		{"oracle vide", []string{"-avant", avant, "-apres", avant, "-faits", faits, "-oracle", vide}, codeUsage, ""},
	}
	for _, k := range cas {
		var out, errw bytes.Buffer
		if got := lancer(k.args, &out, &errw); got != k.veut {
			t.Errorf("%s : code %d, veut %d (%s%s)", k.nom, got, k.veut, out.String(), errw.String())
		}
		if !strings.Contains(out.String(), k.dans) {
			t.Errorf("%s : %q absent de\n%s", k.nom, k.dans, out.String())
		}
	}
}

// TestLancer_RegistreEtRegistreAvant : `-registre` ecrit le registre de la revision courante, et il
// se relit par `-registre-avant`.
func TestLancer_RegistreEtRegistreAvant(t *testing.T) {
	var out, errw bytes.Buffer
	if got := lancer([]string{"-registre"}, &out, &errw); got != codeOK {
		t.Fatalf("code %d : %s", got, errw.String())
	}
	var reg map[string]bool
	if err := json.Unmarshal(out.Bytes(), &reg); err != nil || len(reg) < 50 {
		t.Fatalf("registre illisible ou court (%d entrees) : %v", len(reg), err)
	}
	dir := t.TempDir()
	a := ecrire(t, dir, "a.json", artefactMinimal(false))
	f := ecrire(t, dir, "f.facts.json", faitsMinimaux)
	r := ecrire(t, dir, "r.json", out.String())
	out.Reset()
	if got := lancer([]string{"-avant", a, "-apres", a, "-faits", f, "-registre-avant", r}, &out, &errw); got != codeOK {
		t.Fatalf("code %d : %s %s", got, out.String(), errw.String())
	}
}
