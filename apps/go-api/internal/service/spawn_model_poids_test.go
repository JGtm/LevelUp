package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/observability"
)

func TestChargerPoidsReapparition(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	ecrire := func(nom, contenu string) string {
		p := filepath.Join(dir, nom)
		if err := os.WriteFile(p, []byte(contenu), 0o644); err != nil {
			t.Fatal(err)
		}
		return p
	}
	cas := []struct {
		nom     string
		path    string
		attendu bool
	}{
		{"valide", ecrire("ok.json", `{"revision":2,"date":"2026-11-15","familles":{"tous":{"poids":{"frequence_log":0.5}}}}`), true},
		{"absent", filepath.Join(dir, "absent.json"), false},
		{"illisible", ecrire("mauvais.json", `{`), false},
		{"autre revision", ecrire("rev.json", `{"revision":99}`), false},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			avant := observability.LoadCounterT("t", CompteurSpawnModelRepli)
			p, ok := ChargerPoidsReapparition(ctx, "t", c.path)
			if ok != c.attendu {
				t.Fatalf("ok = %v, attendu %v", ok, c.attendu)
			}
			apres := observability.LoadCounterT("t", CompteurSpawnModelRepli)
			if ok && p.Familles[domain.FamilleToutesModes].Poids["frequence_log"] != 0.5 {
				t.Errorf("poids non relus : %+v", p)
			}
			if !ok && apres != avant+1 {
				t.Errorf("le repli doit se compter : %d -> %d", avant, apres)
			}
			if ok && apres != avant {
				t.Errorf("une lecture reussie ne compte pas de repli")
			}
		})
	}
}
