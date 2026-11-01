package main

import (
	"errors"
	"flag"
	"strings"
	"testing"
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
		{name: "mode refresh-only refuse", args: []string{"-mode=refresh-only"},
			wantErr: "mode non disponible dans cette version", is: errModeUnavailable},
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
