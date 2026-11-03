package main

import (
	"errors"
	"flag"
	"os"
	"regexp"
	"strings"
	"testing"

	"levelup/go-api/internal/ops/parquetbackup"
)

func TestParseServerFlags(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    serverFlags
		wantErr string // sous-chaîne attendue ; vide : pas d'erreur
		is      error  // erreur attendue par errors.Is, si renseignée
	}{
		{name: "aucun argument", args: nil, want: serverFlags{}},
		{name: "health-check seul", args: []string{"-health-check"}, want: serverFlags{HealthCheck: true}},
		{name: "arret sur stdin", args: []string{"-arret-sur-stdin"}, want: serverFlags{StopOnStdin: true}},
		{name: "arret sur stdin et mode vide", args: []string{"-arret-sur-stdin", "-mode", ""},
			want: serverFlags{StopOnStdin: true}},
		{name: "double tiret", args: []string{"--arret-sur-stdin"}, want: serverFlags{StopOnStdin: true}},
		{name: "mode desktop", args: []string{"-mode", "desktop"}, want: serverFlags{Mode: serverModeDesktop}},
		{name: "desktop avec arret sur stdin", args: []string{"-mode", "desktop", "-arret-sur-stdin"},
			want: serverFlags{Mode: serverModeDesktop, StopOnStdin: true}},
		{name: "mode refresh-only", args: []string{"-mode=refresh-only"}, want: serverFlags{Mode: serverModeRefreshOnly}},
		{name: "mode restore", args: []string{"-mode", "restore", "--from", "jeu", "--to", "racine", "--dry-run"},
			want: serverFlags{Mode: serverModeRestore, Restore: parquetbackup.RestoreCommand{From: "jeu", To: "racine", DryRun: true}}},
		{name: "restore avec force", args: []string{"-mode=restore", "-from", "jeu", "-to", "racine", "-force"},
			want: serverFlags{Mode: serverModeRestore, Restore: parquetbackup.RestoreCommand{From: "jeu", To: "racine", Force: true}}},
		{name: "restore sans dossiers", args: []string{"-mode", "restore"}, wantErr: "usage"},
		{name: "restore sans racine cible", args: []string{"-mode", "restore", "--from", "jeu"}, wantErr: "usage"},
		{name: "drapeaux de restauration hors du mode", args: []string{"--from", "jeu"}, wantErr: "n'existent qu'avec -mode restore"},
		{name: "mode inconnu", args: []string{"-mode", "wat"}, wantErr: `-mode "wat" inconnu`},
		{name: "drapeau inconnu", args: []string{"-nope"}, wantErr: "not defined"},
		{name: "argument positionnel", args: []string{"extra"}, wantErr: `argument inattendu "extra"`},
		{name: "aide", args: []string{"-h"}, wantErr: "help requested", is: flag.ErrHelp},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseServerFlags(tt.args)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("erreur inattendue : %v", err)
				}
				if got != tt.want {
					t.Errorf("got %+v, attendu %+v", got, tt.want)
				}
				return
			}
			if err == nil {
				t.Fatalf("erreur attendue (%q), obtenu %+v", tt.wantErr, got)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("erreur %q, attendu une sous-chaîne %q", err, tt.wantErr)
			}
			if tt.is != nil && !errors.Is(err, tt.is) {
				t.Errorf("erreur %q : errors.Is(%v) faux", err, tt.is)
			}
			if got != (serverFlags{}) {
				t.Errorf("valeurs rendues malgré l'erreur : %+v", got)
			}
		})
	}
}

// Le mode enfant de cuisson reste intercepté par main avant le parseur, sur la ligne brute :
// -film-child n'est donc pas un drapeau du serveur.
func TestParseServerFlags_FilmChildNEstPasUnDrapeauDuServeur(t *testing.T) {
	if _, err := parseServerFlags([]string{"-film-child"}); err == nil {
		t.Error("-film-child accepté par le parseur du serveur : il doit rester l'affaire de replaychild.IsChild")
	}
}

// TestParseServerFlags_AucunDrapeauNePorteLeSecret : le secret du lanceur ne voyage JAMAIS en argument
// de ligne de commande (visible dans la liste des processus) ; il passe par l'environnement
// (LEVELUP_DESKTOP_SECRET). Aucun drapeau de ce nom n'existe, et le parseur n'en déclare aucun dont le
// nom évoque un secret (ADR 0045 D-1 et D-3, lot L13.4).
func TestParseServerFlags_AucunDrapeauNePorteLeSecret(t *testing.T) {
	for _, name := range []string{"secret", "desktop-secret", "desktop_secret", "session-secret", "token", "password"} {
		for _, args := range [][]string{{"-" + name, "valeur"}, {"-" + name + "=valeur"}, {"--" + name, "valeur"}} {
			if _, err := parseServerFlags(args); err == nil || !strings.Contains(err.Error(), "not defined") {
				t.Errorf("parseServerFlags(%q) = %v, attendu un drapeau inconnu", args, err)
			}
		}
	}
	src, err := os.ReadFile("flags.go")
	if err != nil {
		t.Fatalf("lecture de flags.go : %v", err)
	}
	for _, line := range strings.Split(string(src), "\n") {
		if strings.Contains(line, "fs.") && strings.Contains(line, "Var(") && secretish.MatchString(line) {
			t.Errorf("flags.go déclare un drapeau qui évoque un secret : %s", strings.TrimSpace(line))
		}
	}
}

var secretish = regexp.MustCompile(`(?i)secret|token|passw`)
