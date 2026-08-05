package weaponv3

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// rushdownHigh32 — high-32 du Gravity Hammer / Rushdown Hammer (arme de pickup
// partagée entre joueurs au fil du match).
const rushdownHigh32 = 0x841AC5E5

// filmManifestChunk — un chunk du manifeste film (champs utiles uniquement).
type filmManifestChunk struct {
	Index   int `json:"index"`
	StartMS int `json:"start_ms"`
}

// filmManifest — manifeste film (data/cache/film_manifests/<short8>.json).
type filmManifest struct {
	Chunks []filmManifestChunk `json:"chunks"`
}

// loadFilmManifest lit le manifeste film d'un match (skip-friendly : renvoie nil
// si absent). Réutilise la résolution de racine du cache des tests pi_resolver.
func loadFilmManifest(t *testing.T, short8 string) *filmManifest {
	t.Helper()
	chunkRoot := filmCacheRoot(t)
	if chunkRoot == "" {
		return nil
	}
	// Les manifests sont dans data/cache/film_manifests, soeur de film_chunks.
	manifestPath := filepath.Join(filepath.Dir(chunkRoot), "film_manifests", short8+".json")
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil
	}
	var m filmManifest
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("manifest %s illisible : %v", short8, err)
	}
	return &m
}

// scanAllMeleeHits agrège ScanMeleeHits sur tous les chunks d'un match (estimateur
// µs ancré sur le start_ms de chaque chunk). Renvoie nil si cache/manifeste absent.
func scanAllMeleeHits(t *testing.T, short8 string) []MeleeHit {
	t.Helper()
	manifest := loadFilmManifest(t, short8)
	if manifest == nil {
		return nil
	}
	var all []MeleeHit
	for _, c := range manifest.Chunks {
		chunk := loadCachedChunk(t, short8, fmt.Sprintf("chunk_%02d.bin", c.Index))
		if chunk == nil {
			continue
		}
		est := USEstimator(chunk, c.StartMS)
		all = append(all, ScanMeleeHits(chunk, est)...)
	}
	return all
}

// TestScanMeleeHits_000d5950 — sur le match validé (§K-bis : 56 swings weapon-
// validés), le scan agrégé doit retomber dans une bande raisonnable (~56, accepté
// 40..72) et exposer le Rushdown/Gravity Hammer (high-32 0x841ac5e5) tenu par au
// moins 2 joueurs distincts (arme de pickup).
func TestScanMeleeHits_000d5950(t *testing.T) {
	hits := scanAllMeleeHits(t, "000d5950")
	if hits == nil {
		t.Skip("cache/manifeste film 000d5950 absent — skip melee ground-truth")
	}

	if len(hits) < 40 || len(hits) > 72 {
		t.Fatalf("swings melee weapon-validés = %d, hors bande [40,72] (vise ~56)", len(hits))
	}

	hammerPI := make(map[int]struct{})
	for _, h := range hits {
		high, known := CanonWeaponID(h.WeaponID)
		if !known {
			t.Fatalf("swing avec arme inconnue (high 0x%08x) — le filtre high-32 a fui", high)
		}
		if h.PI < 0 || h.PI > meleeMaxPI {
			t.Fatalf("pi hors borne [0,%d] : %d", meleeMaxPI, h.PI)
		}
		if high == rushdownHigh32 {
			hammerPI[h.PI] = struct{}{}
		}
		// HitType doit être l'un des type-bytes connus §K-bis (le scanner ne garde
		// que ces trois). Le Gravity Hammer est un HIT marteau (0x47) — pas un whiff.
		switch h.HitType {
		case meleeHitMiss, meleeHitHammer, meleeHitSword:
		default:
			t.Fatalf("HitType inattendu 0x%02x (pi=%d high=0x%08x)", h.HitType, h.PI, high)
		}
		if high == rushdownHigh32 && h.HitType != meleeHitHammer {
			t.Fatalf("Gravity/Rushdown Hammer attendu HitType 0x47, obtenu 0x%02x", h.HitType)
		}
	}
	if _, ok := KnownWeaponHigh32[rushdownHigh32]; !ok {
		t.Fatalf("pré-condition : high-32 0x%08x absent de KnownWeaponHigh32", rushdownHigh32)
	}
	if len(hammerPI) < 2 {
		t.Fatalf("Gravity/Rushdown Hammer attendu sur >=2 pi distincts (arme de pickup), obtenu %d", len(hammerPI))
	}
	t.Logf("melee 000d5950 : %d swings, Hammer sur %d pi distincts", len(hits), len(hammerPI))
}

// TestScanMeleeHits_Empty — un chunk vide ne panique pas et renvoie 0 hit.
func TestScanMeleeHits_Empty(t *testing.T) {
	if got := ScanMeleeHits(nil, func(int) float64 { return 0 }); got != nil {
		t.Fatalf("chunk nil attendu nil, obtenu %v", got)
	}
	if got := ScanMeleeHits([]byte{0x34, 0x35, 0x00}, func(int) float64 { return 0 }); len(got) != 0 {
		t.Fatalf("chunk trop court attendu 0 hit, obtenu %d", len(got))
	}
}

// TestMeleeHitLethal — seuls 0x47 (hammer) et 0x60 (sword/powered) sont létaux ;
// 0x42 (miss/unpowered) et tout autre byte sont non-létaux (§K-bis).
func TestMeleeHitLethal(t *testing.T) {
	cases := map[byte]bool{
		meleeHitMiss:   false, // 0x42 whiff
		meleeHitHammer: true,  // 0x47 hit
		meleeHitSword:  true,  // 0x60 hit
		0x00:           false,
		0x47 + 1:       false,
	}
	for hitType, want := range cases {
		if got := MeleeHitLethal(hitType); got != want {
			t.Fatalf("MeleeHitLethal(0x%02x)=%v, attendu %v", hitType, got, want)
		}
	}
}

// TestLethalMeleeHits_FiltersWhiffs — le filtre létal retire les 0x42 et conserve
// l'ordre chronologique des HIT.
func TestLethalMeleeHits_FiltersWhiffs(t *testing.T) {
	in := []MeleeHit{
		{PI: 3, TimeMS: 100, WeaponID: 0x841AC5E500000000, HitType: meleeHitMiss},
		{PI: 3, TimeMS: 200, WeaponID: 0x841AC5E500000000, HitType: meleeHitHammer},
		{PI: 5, TimeMS: 300, WeaponID: 0x4FF3937E00000000, HitType: meleeHitSword},
		{PI: 5, TimeMS: 400, WeaponID: 0x4FF3937E00000000, HitType: meleeHitMiss},
	}
	got := lethalMeleeHits(in)
	if len(got) != 2 {
		t.Fatalf("attendu 2 hits létaux, obtenu %d", len(got))
	}
	if got[0].TimeMS != 200 || got[1].TimeMS != 300 {
		t.Fatalf("ordre/contenu inattendu : %+v", got)
	}
}
