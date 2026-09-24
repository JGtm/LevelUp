package mappings

// loader_vehicle_weapons_test.go — LE GARDE-RAIL DU REGISTRE DES ARMES DE VEHICULE (retours du
// rejeu 2026-09-23, lot M4a). Il reprend, cote Go, celui que le lot L1.5 avait pose sur les trois
// tables client (`vehicleWeaponTags.guard.test.ts`, supprime avec elles), sur la MEME fixture datee
// du parc :
//
//  1. toute cle du registre est un tag OBSERVE dans un document (fixture datee) ;
//  2. tout tag de vehicule observe au moins `seuilObserve` fois a une entree, ou une ligne
//     `[[unknown]]` motivee ;
//  3. tout son nomme existe dans les assets servis, tout vehicule nomme a son sprite.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// seuilObserve : un tag observe moins souvent peut attendre sa ligne (le Wasp en a 4 et l a).
const seuilObserve = 5

type tagObserve struct {
	Tag   string `json:"tag"`
	Shots int    `json:"shots"`
}

type fixtureObservee struct {
	GeneratedAt string `json:"generatedAt"`
	Source      string `json:"source"`
	Corpus      struct {
		Documents int `json:"documents"`
	} `json:"corpus"`
	Tags []tagObserve `json:"tags"`
}

func vwRepoRoot(t *testing.T) string {
	t.Helper()
	_, thisFile, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..", "..")
}

func vwRegistre(t *testing.T) *VehicleWeaponSet {
	t.Helper()
	set, err := LoadVehicleWeaponsFromFile(filepath.Join(vwRepoRoot(t), "config", "titles",
		"halo_infinite", "mappings", "vehicle_weapons.toml"))
	if err != nil {
		t.Fatalf("registre : %v", err)
	}
	return set
}

func vwFixture(t *testing.T) fixtureObservee {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", "vehicle_weapons", "vehicle_weapon_tags_observed.json"))
	if err != nil {
		t.Fatal(err)
	}
	var f fixtureObservee
	if err := json.Unmarshal(raw, &f); err != nil {
		t.Fatal(err)
	}
	return f
}

// tagDeShot : `0x<TAG>00000000` -> `TAG`, ou vide si ce n est pas le gabarit d une arme de vehicule.
func tagDeShot(w string) string {
	if len(w) != 18 || !strings.HasPrefix(w, "0x") || !strings.HasSuffix(w, "00000000") {
		return ""
	}
	return w[2:10]
}

func TestRegistreArmesVehicule_FixtureDateeEtRegenerable(t *testing.T) {
	f := vwFixture(t)
	if f.GeneratedAt != "2026-09-23" || f.Corpus.Documents != 111 || len(f.Tags) == 0 {
		t.Fatalf("fixture = %s / %d documents / %d tags, attendu 2026-09-23 / 111 / >0",
			f.GeneratedAt, f.Corpus.Documents, len(f.Tags))
	}
	const instrument = "sweep_vehicle_weapon_tags.mjs"
	if !strings.Contains(f.Source, instrument) {
		t.Errorf("la fixture doit citer son instrument : %q", f.Source)
	}
	if _, err := os.Stat(filepath.Join("testdata", "vehicle_weapons", instrument)); err != nil {
		t.Errorf("instrument absent : %v", err)
	}
}

func TestRegistreArmesVehicule_ToutesLesClesSontObservees(t *testing.T) {
	set, f := vwRegistre(t), vwFixture(t)
	observes := map[string]bool{}
	for _, o := range f.Tags {
		observes[tagDeShot(o.Tag)] = true
	}
	for _, tag := range set.Tags() {
		if !observes[tag] {
			t.Errorf("%s : au registre sans avoir ete observe dans un document", tag)
		}
	}
	for tag := range set.Unknown() {
		if !observes[tag] {
			t.Errorf("%s : ligne inconnue sur un tag jamais observe", tag)
		}
	}
}

func TestRegistreArmesVehicule_ToutTagFrequentANomOuRaison(t *testing.T) {
	set, f := vwRegistre(t), vwFixture(t)
	inconnus := set.Unknown()
	for _, o := range f.Tags {
		if o.Shots < seuilObserve {
			continue
		}
		tag := tagDeShot(o.Tag)
		if _, ok := set.Weapon(tag); !ok && inconnus[tag] == "" {
			t.Errorf("%s (%d tirs) : ni entree ni ligne inconnue motivee", tag, o.Shots)
		}
	}
}

func TestRegistreArmesVehicule_SonsEtSpritesServis(t *testing.T) {
	set, root := vwRegistre(t), vwRepoRoot(t)
	for _, tag := range set.Tags() {
		w, _ := set.Weapon(tag)
		if w.Sound != "" {
			p := filepath.Join(root, "static", "sounds", "halo_infinite", w.Sound+".wav")
			if _, err := os.Stat(p); err != nil {
				t.Errorf("%s : son %q absent des assets servis", tag, w.Sound)
			}
		}
		p := filepath.Join(root, "static", "vehicles-assets", "halo_infinite", "replay", w.Vehicle+".png")
		if _, err := os.Stat(p); err != nil {
			t.Errorf("%s : vehicule %q sans sprite (vocabulaire des familles / variantes)", tag, w.Vehicle)
		}
	}
}

// TestRegistreArmesVehicule_ValidationFermee — les listes fermees et les champs obligatoires.
func TestRegistreArmesVehicule_ValidationFermee(t *testing.T) {
	const base = `tag = "121B4009"
vehicle = "wraith"
en = "a"
fr = "b"
fire = "single"
fx = "plasma"
tint = "plasma_hot"
proof = "p"
`
	cas := map[string]string{
		"tag minuscule":      strings.Replace(base, "121B4009", "121b4009", 1) + "sound = \"s\"\n",
		"regime inconnu":     strings.Replace(base, "single", "burst", 1) + "sound = \"s\"\n",
		"forme inconnue":     strings.Replace(base, "\"plasma\"", "\"laser\"", 1) + "sound = \"s\"\n",
		"son ET silence":     base + "sound = \"s\"\nsilence = \"r\"\n",
		"ni son ni silence":  base,
		"sans preuve":        strings.Replace(base, "proof = \"p\"", "proof = \"\"", 1) + "sound = \"s\"\n",
		"montage hors cadre": base + "sound = \"s\"\nmount = { aim = \"fixed\", ax = 0.7, ay = 0.0 }\n",
		"visee inconnue":     base + "sound = \"s\"\nmount = { aim = \"free\", ax = 0.0, ay = 0.0 }\n",
	}
	for nom, entree := range cas {
		if _, err := LoadVehicleWeaponsFromBytes("t.toml", []byte("[[weapons]]\n"+entree)); err == nil {
			t.Errorf("%s : accepte, attendu refuse", nom)
		}
	}
	doublon := "[[weapons]]\n" + base + "sound = \"s\"\n[[unknown]]\ntag = \"121B4009\"\nreason = \"r\"\n"
	if _, err := LoadVehicleWeaponsFromBytes("t.toml", []byte(doublon)); err == nil {
		t.Error("tag a la fois arme et inconnu : accepte, attendu refuse")
	}
	if _, err := LoadVehicleWeaponsFromBytes("t.toml", []byte("[[weapons]]\n"+base+"sound = \"s\"\n")); err != nil {
		t.Errorf("entree valide refusee : %v", err)
	}
}
