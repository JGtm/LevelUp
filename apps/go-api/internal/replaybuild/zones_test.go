package replaybuild

// zones_test.go — LA GARDE DE LA JOINTURE : l'artefact et le service doivent construire LA MEME
// liste de zones, sinon `zoneStates[].zoneRef` teinterait la mauvaise zone.
//
// C'est le seul garde-fou possible : les deux listes sont produites par deux chemins differents
// (ici pour l'artefact, `replay.BuildMapObjectives` pour la requete), et rien dans le type ne les
// oblige a coincider. L'erreur, elle, serait invisible — une zone teintee reste credible.

import (
	"testing"

	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/domain/title"
)

// vagabondMapID est la carte des deux temoins de Bastion du lot C-bis.
const vagabondMapID = "105f5d84-8de1-4908-af3a-1c4f3bf9d642"

// TestMatchZonesSuitLOrdreDuCalqueServi : la liste de l'artefact et celle du service sont la
// meme, zone par zone, sur le catalogue VERSIONNE.
func TestMatchZonesSuitLOrdreDuCalqueServi(t *testing.T) {
	b, cat := zonesTestBuilder(t)
	zones, roles := b.matchZones("m", vagabondMapID, "Arena:Strongholds")
	if len(zones) == 0 {
		t.Fatalf("aucune zone rendue pour Bastion sur Vagabond — le catalogue en porte")
	}
	if roles != "strongholds_zone" {
		t.Errorf("roles publies %q, attendu « strongholds_zone »", roles)
	}
	entry, err := cat.Lookup(vagabondMapID)
	if err != nil {
		t.Fatalf("carte %s absente du catalogue : %v", vagabondMapID, err)
	}
	specs := []replay.ObjectiveRoleSpec{{Role: "strongholds_zone", Neutral: true}}
	served := replay.BuildMapObjectives(entry, specs)
	if served == nil || len(served.Zones) != len(zones) {
		t.Fatalf("le service sert %d zone(s), l'artefact en apparie %d",
			zonesServedCount(served), len(zones))
	}
	for i, z := range zones {
		dto := served.Zones[i]
		if float32(z.Center.X) != dto.X || float32(z.Center.Y) != dto.Y {
			t.Errorf("zone %d : artefact (%.2f ; %.2f) contre service (%.2f ; %.2f) —"+
				" l'index publie ne designerait pas la meme zone",
				i, z.Center.X, z.Center.Y, dto.X, dto.Y)
		}
	}
}

// TestMatchZonesLitLeModeDansLesDeuxOrdres : `game_variant_name` ecrit `Strongholds:Arena`, le
// registre ecrit ses `pair_name` `Arena:Strongholds`. Les deux doivent rendre les memes roles —
// normaliser d'abord garderait « Arena » sur le premier, et le mode serait perdu en silence.
func TestMatchZonesLitLeModeDansLesDeuxOrdres(t *testing.T) {
	b, _ := zonesTestBuilder(t)
	direct, _ := b.matchZones("m", vagabondMapID, "Strongholds:Arena")
	inverse, _ := b.matchZones("m", vagabondMapID, "Arena:Strongholds")
	if len(direct) == 0 {
		t.Fatalf("aucune zone pour la variante « Strongholds:Arena » — l'ordre du registre est perdu")
	}
	if len(direct) != len(inverse) {
		t.Errorf("%d zone(s) pour « Strongholds:Arena » contre %d pour « Arena:Strongholds »",
			len(direct), len(inverse))
	}
}

// TestMatchZonesModesSansZone : un mode dont la table ne sert aucun role surfacique n'apparie
// rien — et le balayage de `ti=13` n'est donc pas paye.
func TestMatchZonesModesSansZone(t *testing.T) {
	b, _ := zonesTestBuilder(t)
	for _, variant := range []string{"Arena:Slayer", "Arena:CTF", "Arena:Oddball", ""} {
		if zones, roles := b.matchZones("m", vagabondMapID, variant); len(zones) != 0 {
			t.Errorf("variante %q : %d zone(s) appariee(s) (%v), attendu aucune",
				variant, len(zones), roles)
		}
	}
}

// TestMatchZonesReplieSurLesRolesSurfaciquesEnColline : la table du titre ne sert aucun role en
// KOTH (le catalogue n'a pas de role de colline) ; l'artefact se replie sur les roles
// surfaciques, DANS UN ORDRE FIXE, et le publie.
func TestMatchZonesReplieSurLesRolesSurfaciquesEnColline(t *testing.T) {
	b, _ := zonesTestBuilder(t)
	zones, roles := b.matchZones("m", vagabondMapID, "Arena:King of the Hill")
	if len(zones) == 0 {
		t.Fatalf("aucune zone rendue en KOTH : le repli des roles surfaciques ne joue pas")
	}
	if roles != "strongholds_zone,extraction_zone" {
		t.Errorf("roles publies %q, attendu « strongholds_zone,extraction_zone » dans cet ordre", roles)
	}
}

// TestMatchZonesSansMapID : sans clé de carte, aucune jointure n'existe — et c'est une absence
// propre, pas une erreur.
func TestMatchZonesSansMapID(t *testing.T) {
	b, _ := zonesTestBuilder(t)
	if zones, _ := b.matchZones("m", "", "Arena:Strongholds"); zones != nil {
		t.Errorf("%d zone(s) appariee(s) sans map_id", len(zones))
	}
}

func zonesTestBuilder(t *testing.T) (*Builder, *replay.MapObjectivesCatalog) {
	t.Helper()
	repoRoot, err := title.FindRepoRoot()
	if err != nil {
		t.Skipf("racine repo introuvable : %v", err)
	}
	b, err := NewBuilder(repoRoot, title.DefaultSlug)
	if err != nil {
		t.Fatalf("NewBuilder : %v", err)
	}
	cat := b.objectivesCatalog()
	if cat == nil {
		t.Skipf("catalogue d'objectifs indisponible")
	}
	return b, cat
}

func zonesServedCount(mo *replay.MapObjectives) int {
	if mo == nil {
		return 0
	}
	return len(mo.Zones)
}
