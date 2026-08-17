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

[[ability_palettes]]
id = "famille_a"
markers = [2, 4]
[ability_palettes.ranks]
"2" = { en = "Drop Wall", fr = "mur portatif" }

[[ability_palettes]]
id = "famille_b"
markers = [19, 20]
[ability_palettes.ranks]
"20" = { en = "Grappleshot", fr = "grappin" }

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
	pal := set.AbilityPalettes()
	if len(pal) != 2 || pal[0].ID != "famille_a" || pal[1].ID != "famille_b" {
		t.Fatalf("palettes mal lues: %+v", pal)
	}
	if got := pal[0].Ranks[2]; got.En != "Drop Wall" || got.Fr != "mur portatif" {
		t.Errorf("capacité 2 de famille_a mal lue: %+v", got)
	}
	// Les MARQUEURS voyagent avec les noms : sans eux, aucun film ne se classe.
	if len(pal[1].Markers) != 2 || pal[1].Markers[0] != 19 {
		t.Errorf("marqueurs de famille_b perdus: %+v", pal[1].Markers)
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
		{"rang non numérique", "[meta]\ntitle_slug=\"x\"\nschema_version=1\n[[ability_palettes]]\nid=\"p\"\nmarkers=[1]\n[ability_palettes.ranks]\n\"abc\"={en=\"A\",fr=\"A\"}\n"},
		{"capacité sans en", "[meta]\ntitle_slug=\"x\"\nschema_version=1\n[[ability_palettes]]\nid=\"p\"\nmarkers=[3]\n[ability_palettes.ranks]\n\"3\"={fr=\"A\"}\n"},
		{"palette sans id", "[meta]\ntitle_slug=\"x\"\nschema_version=1\n[[ability_palettes]]\nmarkers=[1]\n"},
		// Une palette sans marqueur ne pourrait JAMAIS etre reconnue : de la donnée morte.
		{"palette sans marqueur", "[meta]\ntitle_slug=\"x\"\nschema_version=1\n[[ability_palettes]]\nid=\"p\"\n"},
		{"palette dupliquée", "[meta]\ntitle_slug=\"x\"\nschema_version=1\n[[ability_palettes]]\nid=\"p\"\nmarkers=[1]\n[[ability_palettes]]\nid=\"p\"\nmarkers=[2]\n"},
		// Un marqueur partagé rendrait le classement ambigu sur tout film qui le montre.
		{"marqueur partagé", "[meta]\ntitle_slug=\"x\"\nschema_version=1\n[[ability_palettes]]\nid=\"a\"\nmarkers=[4]\n[[ability_palettes]]\nid=\"b\"\nmarkers=[4]\n"},
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

// objetEquipementTOML fabrique une section [[equipment_objects]] d'une seule ligne.
func objetEquipementTOML(famille, provenance, kind string) string {
	return fmt.Sprintf("[meta]\ntitle_slug=\"x\"\nschema_version=1\n"+
		"[[equipment_objects]]\nid=\"0x528fce46\"\nfamily=%q\nname_id=\"0xedebd7b7\"\n"+
		"provenance=%q\nkind=%q\n", famille, provenance, kind)
}

// TestLoadReplayLabels_NatureEquipement — L'INVARIANT `kind` x `provenance`, DANS LES DEUX
// SENS, ET IL DOIT POUVOIR ETRE VU ROUGE.
//
// CE QU'IL PROTEGE, mesure a l'appui (PLAN_ORIGINE_POSES_ET_FAMILLES G.3) : `kind = deployed`
// autorise le rendu a dessiner l'objet. Le declarer sur un identifiant que la structure du jeu
// ne rattache pas comme une piece ENGENDREE (`sofa_parent`) ferait dessiner un mur a l'endroit
// ou un joueur est mort en portant son appareil — l'appareil du mur n'est deploye que dans
// 13,0 a 29,4 % de ses poses, quand les panneaux le sont dans 97,7 et 97,9 %.
func TestLoadReplayLabels_NatureEquipement(t *testing.T) {
	const famMur = "wall"
	ok := []struct{ nom, famille, prov, kind string }{
		{"panneaux deployes", famMur, equipProvSofaParent, equipKindDeployed},
		{"appareil porte", famMur, equipProvSofaStringID, equipKindCarried},
		{"grenade portee", "grenade_frag", "gggl_entree", equipKindCarried},
	}
	for _, c := range ok {
		if _, err := LoadReplayLabelsFromBytes("t.toml",
			[]byte(objetEquipementTOML(c.famille, c.prov, c.kind))); err != nil {
			t.Errorf("%s : refuse alors qu'il est coherent : %v", c.nom, err)
		}
	}
	ko := []struct{ nom, famille, prov, kind string }{
		{"nature absente", famMur, equipProvSofaStringID, ""},
		{"nature inconnue", famMur, equipProvSofaStringID, "dotation"},
		// LES DEUX SENS : sans eux `kind` serait un commentaire.
		{"deployed sans sofa_parent", famMur, equipProvSofaStringID, equipKindDeployed},
		{"sofa_parent declare carried", famMur, equipProvSofaParent, equipKindCarried},
	}
	for _, c := range ko {
		if _, err := LoadReplayLabelsFromBytes("t.toml",
			[]byte(objetEquipementTOML(c.famille, c.prov, c.kind))); err == nil {
			t.Errorf("%s : accepte alors qu'il doit etre refuse", c.nom)
		}
	}
}
