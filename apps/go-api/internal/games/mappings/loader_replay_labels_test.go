package mappings

import (
	"fmt"
	"testing"
)

const replayLabelsValide = `
[meta]
title_slug     = "halo_infinite"
schema_version = 1

[[grenades]]
en = "Frag"
fr = "Fragmentation"

[[grenades]]
en = "Plasma"
fr = "Plasma"

[abilities]
"3" = { en = "Drop Wall", fr = "mur portatif" }

[shot_effects]
hinf_ma40_ar = "ballistic"
`

func TestLoadReplayLabels_Valide(t *testing.T) {
	set, err := LoadReplayLabelsFromBytes("t.toml", []byte(replayLabelsValide))
	if err != nil {
		t.Fatalf("chargement: %v", err)
	}
	if set.TitleSlug() != "halo_infinite" || set.SchemaVersion() != 1 {
		t.Errorf("meta mal lue: slug=%q version=%d", set.TitleSlug(), set.SchemaVersion())
	}
	// L'ORDRE EST LA DONNÉE : le rang d'une grenade est sa position dans le fichier.
	ranks := set.GrenadeRanks()
	if len(ranks) != 2 || ranks[0].En != "Frag" || ranks[1].En != "Plasma" {
		t.Errorf("ordre des rangs perdu: %+v", ranks)
	}
	if got := set.Abilities()[3]; got.En != "Drop Wall" || got.Fr != "mur portatif" {
		t.Errorf("capacité 3 mal lue: %+v", got)
	}
	if got := set.ShotEffects()["hinf_ma40_ar"]; got != "ballistic" {
		t.Errorf("effet de tir mal lu: %q", got)
	}
}

// TestLoadReplayLabels_Refus — la validation est TOUT-OU-RIEN : un rejeu à moitié nommé
// se lit comme une donnée absente alors que c'est une configuration incomplète.
func TestLoadReplayLabels_Refus(t *testing.T) {
	cas := []struct{ nom, toml string }{
		{"sans meta", "[[grenades]]\nen = \"Frag\"\nfr = \"Fragmentation\"\n"},
		{"version nulle", "[meta]\ntitle_slug=\"x\"\nschema_version=0\n"},
		{"grenade sans fr", "[meta]\ntitle_slug=\"x\"\nschema_version=1\n[[grenades]]\nen=\"Frag\"\n"},
		{"grenade sans en", "[meta]\ntitle_slug=\"x\"\nschema_version=1\n[[grenades]]\nfr=\"Frag\"\n"},
		{"capacité non numérique", "[meta]\ntitle_slug=\"x\"\nschema_version=1\n[abilities]\n\"abc\"={en=\"A\",fr=\"A\"}\n"},
		{"capacité sans en", "[meta]\ntitle_slug=\"x\"\nschema_version=1\n[abilities]\n\"3\"={fr=\"A\"}\n"},
		{"effet inconnu", "[meta]\ntitle_slug=\"x\"\nschema_version=1\n[shot_effects]\nk=\"laser\"\n"},
	}
	for _, c := range cas {
		if _, err := LoadReplayLabelsFromBytes("t.toml", []byte(c.toml)); err == nil {
			t.Errorf("%s : accepté alors qu'il devrait être refusé", c.nom)
		}
	}
}

// TestLoadReplayLabels_EffetInconnuRefuse — POURQUOI LA LISTE EST FERMÉE : une valeur
// libre ferait tomber l'arme sur le rendu neutre EN SILENCE, ce qui est indistinguable
// d'une arme volontairement non cataloguée.
func TestLoadReplayLabels_EffetInconnuRefuse(t *testing.T) {
	base := "[meta]\ntitle_slug=\"x\"\nschema_version=1\n[shot_effects]\nhinf_x=%q\n"
	for _, fam := range []string{"ballistic", "plasma", "light", "shock", "explosive", "melee", "needles"} {
		if _, err := LoadReplayLabelsFromBytes("t.toml", []byte(fmt.Sprintf(base, fam))); err != nil {
			t.Errorf("famille admise %q refusée : %v", fam, err)
		}
	}
	// `plain` n'est PAS une famille : c'est l'absence de famille connue, et elle
	// s'exprime en n'écrivant pas la ligne. L'accepter ici ferait deux façons de dire
	// la même chose, dont une qui se lit comme un choix.
	for _, fam := range []string{"plain", "", "Ballistic", "kinetic"} {
		if _, err := LoadReplayLabelsFromBytes("t.toml", []byte(fmt.Sprintf(base, fam))); err == nil {
			t.Errorf("famille %q acceptée alors qu'elle n'est pas dans la liste fermée", fam)
		}
	}
}
