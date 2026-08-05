package replay

import (
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/analysis/weaponv3"
)

// Familles réelles du catalogue de production, choisies pour ce que le test doit prouver :
// famHammer et famHammerAlias n'existent pas (une seule famille par canon dans le catalogue),
// donc l'alias est simulé par deux familles qui résolvent au MÊME nom — c'est exactement le
// cas Shock Rifle / Shock Rifle (Ranked) et M41 SPNKr / Fuel Rod SPNKr.
const (
	famShockRifle       = 0x9387A8B9 // "Shock Rifle"
	famShockRifleRanked = 0x1A22FEE6 // "Shock Rifle" aussi (variante classée)
	famSkewer           = 0x0D20C469 // "Skewer"
	famInconnue         = 0x00000001 // absente du catalogue
)

func TestLoadoutFamilies_DeriveDuCatalogueDeProduction(t *testing.T) {
	fams := loadoutFamilies()
	if len(fams) != len(weaponv3.KnownWeaponHigh32) {
		t.Fatalf("le set interrogé doit être exactement le catalogue : %d vs %d",
			len(fams), len(weaponv3.KnownWeaponHigh32))
	}
	if !fams[famSkewer] {
		t.Fatalf("famille %08X (Skewer) absente du set interrogé", famSkewer)
	}
	if fams[famInconnue] {
		t.Fatalf("famille %08X hors catalogue acceptée : le balayage rendrait du bruit", famInconnue)
	}
}

// TestBuildLoadouts_ReplieLesAliasEtIgnoreLInconnu : deux identifiants résolvant au même nom
// sont UN canon (sinon le client afficherait deux fois le même fusil) ; un identifiant hors
// catalogue est écarté sans faire disparaître le loadout.
func TestBuildLoadouts_ReplieLesAliasEtIgnoreLInconnu(t *testing.T) {
	raw := []filmdec.KeyframeLoadout{{
		TimestampUS: 1_500_000, Slot: 512,
		Families: []uint32{famShockRifle, famShockRifleRanked, famSkewer, famInconnue},
	}}
	got := buildLoadouts(raw, 500_000, 100_000)
	if len(got) != 1 {
		t.Fatalf("1 loadout attendu, %d obtenus", len(got))
	}
	if got[0].T != 10 || got[0].Slot != 512 {
		t.Fatalf("frame/slot faux : %+v", got[0])
	}
	if len(got[0].W) != 2 {
		t.Fatalf("2 armes attendues après repli des alias, %d obtenues : %v", len(got[0].W), got[0].W)
	}
	if got[0].W[0] != "0x9387A8B9" || got[0].W[1] != "0x0D20C469" {
		t.Fatalf("identifiants ou ordre de lecture faux : %v", got[0].W)
	}
}

// TestBuildLoadouts_EcarteCeQuiPrecedeLOrigine : un keyframe antérieur au premier paquet de
// position n'a pas de frame sur l'axe du rejeu — il ne doit pas devenir la frame 0.
func TestBuildLoadouts_EcarteCeQuiPrecedeLOrigine(t *testing.T) {
	raw := []filmdec.KeyframeLoadout{
		{TimestampUS: 100_000, Slot: 512, Families: []uint32{famSkewer}},
		{TimestampUS: 700_000, Slot: 513, Families: []uint32{famSkewer}},
	}
	got := buildLoadouts(raw, 500_000, 100_000)
	if len(got) != 1 || got[0].Slot != 513 || got[0].T != 2 {
		t.Fatalf("seul le loadout postérieur à l'origine doit survivre : %+v", got)
	}
}

// TestBuildLoadouts_SansArmeConnueNePublieRien : un record dont aucune famille n'est au
// catalogue ne doit pas produire un loadout VIDE (le client l'afficherait comme « connu »).
func TestBuildLoadouts_SansArmeConnueNePublieRien(t *testing.T) {
	raw := []filmdec.KeyframeLoadout{{TimestampUS: 600_000, Slot: 512, Families: []uint32{famInconnue}}}
	if got := buildLoadouts(raw, 500_000, 100_000); got != nil {
		t.Fatalf("aucun loadout publiable attendu, obtenu %+v", got)
	}
}

func TestKeepLoadoutsOfPublishedTracks(t *testing.T) {
	in := []Loadout{{T: 1, Slot: 512, W: []string{"0x0D20C469"}}, {T: 1, Slot: 999, W: []string{"0x0D20C469"}}}
	got := keepLoadoutsOfPublishedTracks(in, []Track{{Slot: 512}})
	if len(got) != 1 || got[0].Slot != 512 {
		t.Fatalf("le loadout d'un slot sans trajectoire doit être écarté : %+v", got)
	}
}

// TestFormatWeaponFamily : 8 chiffres, pas 16 — la longueur dit qu'une famille n'est PAS un
// weapon-id complet, et un zéro de tête ne doit pas disparaître.
func TestFormatWeaponFamily(t *testing.T) {
	if got := formatWeaponFamily(famSkewer); got != "0x0D20C469" {
		t.Fatalf("format faux : %s", got)
	}
}
