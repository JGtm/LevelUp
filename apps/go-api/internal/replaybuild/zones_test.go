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
	"levelup/go-api/internal/analysis/replay/mapvar"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/testutil"
)

// vagabondMapID est la carte des deux temoins de Bastion du lot C-bis.
const vagabondMapID = "105f5d84-8de1-4908-af3a-1c4f3bf9d642"

// catalystMapID est la carte du temoin KOTH `01e1f945` (lot C-ter) : 3 zones de Bastion, 5
// d'Extraction, 2 livraisons de drapeau en cylindre et 6 collines au catalogue.
const catalystMapID = "f7e8cde9-0c0a-487c-94a3-61bfa0f20465"

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

// TestMatchZonesModesSansZone : un mode sans zone TENUE n'apparie rien — et le balayage de
// `ti=13` n'est donc pas paye — MEME quand la carte declare des volumes sous d'autres roles :
// Catalyst porte 2 livraisons de drapeau en cylindre (servies au client en CTF) et 5 zones
// d'Extraction. Avant le lot C-ter, `Arena:CTF` y rendait 2 zones et payait le balayage pour
// une couverture vide (revue de la phase 2b, P2) ; le test ne passait pour Vagabond que parce
// qu'elle n'a pas de `flag_delivery` avec forme.
func TestMatchZonesModesSansZone(t *testing.T) {
	b, _ := zonesTestBuilder(t)
	for _, mapID := range []string{vagabondMapID, catalystMapID} {
		for _, variant := range []string{"Arena:Slayer", "Arena:CTF", "CTF:Arena", "Arena:Oddball",
			"Arena:Extraction", ""} {
			if zones, roles := b.matchZones("m", mapID, variant); len(zones) != 0 {
				t.Errorf("carte %s, variante %q : %d zone(s) appariee(s) (%v), attendu aucune",
					mapID, variant, len(zones), roles)
			}
		}
	}
}

// TestMatchZonesKOTHViennentDuRoleHill : en KOTH les zones du match sont les COLLINES du role
// `hill` (lot C-ter volet 2), dans les deux ordres de libelle, et RIEN d'autre — plus de repli
// sur les formes de Bastion/Extraction. Sur Catalyst le catalogue porte 6 collines ; la liste
// est celle que le service sert (meme garde que le test de jointure).
func TestMatchZonesKOTHViennentDuRoleHill(t *testing.T) {
	b, cat := zonesTestBuilder(t)
	for _, variant := range []string{"Arena:King of the Hill", "KOTH:Arena", "Ranked:King of the Hill"} {
		zones, roles := b.matchZones("m", catalystMapID, variant)
		if len(zones) != 6 {
			t.Fatalf("variante %q : %d zone(s), attendu les 6 collines de Catalyst (%v)", variant, len(zones), roles)
		}
		if roles != "hill" {
			t.Errorf("variante %q : roles publies %q, attendu « hill » seul", variant, roles)
		}
		for _, z := range zones {
			if z.Role != mapvar.RoleHill {
				t.Errorf("variante %q : zone de role %q dans le catalogue du match", variant, z.Role)
			}
		}
	}
	entry, err := cat.Lookup(catalystMapID)
	if err != nil {
		t.Fatalf("Catalyst absente du catalogue : %v", err)
	}
	zones, _ := b.matchZones("m", catalystMapID, "KOTH:Arena")
	served := replay.BuildMapObjectives(entry, []replay.ObjectiveRoleSpec{{Role: mapvar.RoleHill, Neutral: true}})
	if served == nil || len(served.Zones) != len(zones) {
		t.Fatalf("le service sert %d colline(s), l'artefact en apparie %d", zonesServedCount(served), len(zones))
	}
	for i, z := range zones {
		if dto := served.Zones[i]; float32(z.Center.X) != dto.X || float32(z.Center.Y) != dto.Y {
			t.Errorf("colline %d : artefact (%.2f ; %.2f) contre service (%.2f ; %.2f)", i, z.Center.X, z.Center.Y, dto.X, dto.Y)
		}
	}
}

// TestMatchZonesBastionSansCollines : sur une carte qui porte les deux roles tenus (Catalyst :
// 3 zones de Bastion, 6 collines), Bastion ne rend que ses zones — la table decide, pas la carte.
func TestMatchZonesBastionSansCollines(t *testing.T) {
	b, _ := zonesTestBuilder(t)
	zones, roles := b.matchZones("m", catalystMapID, "Arena:Strongholds")
	if len(zones) != 3 || roles != "strongholds_zone" {
		t.Fatalf("Bastion sur Catalyst : %d zone(s), roles %q — attendu 3 zones, « strongholds_zone »", len(zones), roles)
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

// zonesTestBuilder monte le constructeur sur les fichiers VERSIONNES du depot (catalogue
// d'objectifs, table des roles, bornes de quantification).
//
// AUCUN SKIP : la racine vient de testutil.RepoRoot() (deduite de l'arbre source, pas d'un
// marqueur gitignore ni de LEVELUP_REPO_ROOT) et tout ce que ce test lit est versionne. Un
// skip ici a rendu ces gardes muettes en CI pendant tout le lot (revue ronde 1, R1-1).
func zonesTestBuilder(t *testing.T) (*Builder, *replay.MapObjectivesCatalog) {
	t.Helper()
	repoRoot, err := testutil.RepoRoot()
	if err != nil {
		t.Fatalf("racine du depot introuvable : %v", err)
	}
	b, err := NewBuilder(repoRoot, title.DefaultSlug)
	if err != nil {
		t.Fatalf("NewBuilder : %v", err)
	}
	cat := b.objectivesCatalog()
	if cat == nil {
		t.Fatal("catalogue d'objectifs versionne illisible — cf. le log de objectivesCatalog()")
	}
	return b, cat
}

func zonesServedCount(mo *replay.MapObjectives) int {
	if mo == nil {
		return 0
	}
	return len(mo.Zones)
}
