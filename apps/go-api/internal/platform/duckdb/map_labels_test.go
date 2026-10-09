// Package duckdb — map_labels_test.go : LE libellé d'une carte (map_labels.go), lu par
// l'Explorateur, l'onglet Tactique et le score d'engagement.
//
// TestResolveMapNameFR garde son nom (baseline de tests) : il teste désormais la traduction
// canonique (traductionsDeCartes), qui a remplacé la résolution propre au score d'engagement.
package duckdb

import (
	"context"
	"testing"
)

func metaDesTraductions(t *testing.T) *DB {
	t.Helper()
	meta, err := OpenReadWrite(":memory:")
	if err != nil {
		t.Fatalf("OpenReadWrite metadata: %v", err)
	}
	t.Cleanup(func() { _ = meta.Close() })
	ctx := context.Background()
	for _, s := range []string{
		`CREATE TABLE asset_translations (
			asset_id    VARCHAR,
			asset_type  VARCHAR,
			lang        VARCHAR,
			name        VARCHAR,
			description VARCHAR,
			fetched_at  TIMESTAMP
		)`,
		// The Pit : EN "The Pit" / FR "La fosse" (cas EN != FR).
		`INSERT INTO asset_translations VALUES ('648ae7aa','map','fr-FR','La fosse','',now())`,
		`INSERT INTO asset_translations VALUES ('648ae7aa','map','en-US','The Pit','',now())`,
		// Aquarius : FR == EN (cas degenere mais valide).
		`INSERT INTO asset_translations VALUES ('33c0766c','map','fr-FR','Aquarius','',now())`,
		// Carte avec seulement un repli 'fr' (pas 'fr-FR').
		`INSERT INTO asset_translations VALUES ('ffff0001','map','fr','Repli','',now())`,
		// Bruit : une selection FR ne doit PAS repondre pour asset_type='map'.
		`INSERT INTO asset_translations VALUES ('648ae7aa','playlist','fr-FR','NE PAS PRENDRE','',now())`,
	} {
		if _, err := meta.Exec(ctx, s); err != nil {
			t.Fatalf("seed %q: %v", s, err)
		}
	}
	return meta
}

func TestResolveMapNameFR(t *testing.T) {
	meta := metaDesTraductions(t)
	traductions := traductionsDeCartes(context.Background(), meta,
		[]string{"648ae7aa", "33c0766c", "ffff0001", "deadbeef"})
	cases := []struct {
		name      string
		mapID     string
		wantFR    string
		wantFound bool
	}{
		{"EN!=FR -> FR", "648ae7aa", "La fosse", true},
		{"FR==EN", "33c0766c", "Aquarius", true},
		{"fallback lang 'fr'", "ffff0001", "Repli", true},
		{"map_id inconnu -> fallback EN", "deadbeef", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := traductions[tc.mapID]
			if ok != tc.wantFound {
				t.Fatalf("found = %v, want %v (got %q)", ok, tc.wantFound, got)
			}
			if ok && got != tc.wantFR {
				t.Errorf("FR = %q, want %q", got, tc.wantFR)
			}
		})
	}
}

// Metadata nil -> aucune traduction, jamais de panique.
func TestResolveMapNameFR_NilMetadata(t *testing.T) {
	if got := traductionsDeCartes(context.Background(), nil, []string{"648ae7aa"}); got != nil {
		t.Errorf("attendu nil sans Metadata, got %v", got)
	}
}

// TestLibelleDeCarte_Regle : le nom FR du registre quand il diffère de l'EN, sinon la traduction
// rognée, sinon le registre.
func TestLibelleDeCarte_Regle(t *testing.T) {
	cases := []struct {
		nom, registreFR, registreEN, traduction, want string
	}{
		{"registre FR distinct : gardé", "La Fosse (registre)", "The Pit", "La fosse", "La Fosse (registre)"},
		{"registre FR absent : la traduction", "", "The Pit", "La fosse", "La fosse"},
		{"registre FR égal à l'EN : la traduction", "The Pit", "The Pit", "La fosse", "La fosse"},
		{"traduction rognée", "", "The Pit", "  La fosse ", "La fosse"},
		{"aucune traduction : le registre FR", "The Pit", "The Pit", "", "The Pit"},
		{"aucune traduction ni FR : l'EN", "", "The Pit", "", "The Pit"},
		{"rien", "", "", "", ""},
	}
	for _, c := range cases {
		if got := libelleDeCarte(c.registreFR, c.registreEN, c.traduction); got != c.want {
			t.Errorf("%s : %q, want %q", c.nom, got, c.want)
		}
	}
}
