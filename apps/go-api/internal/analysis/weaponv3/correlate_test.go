package weaponv3

// correlate_test.go — smoke de l'orchestrateur v3 sur le match cache 000d5950.
//
// Skip-if-missing : si le cache film n'est pas présent (CI sans data), le test
// est SKIPPÉ (pas d'échec). La validation chiffrée (recall/précision) est du
// ressort de la couche CLI, pas de ce smoke. Réutilise filmCacheRoot/
// loadCachedChunk (pi_resolver_test.go) pour la résolution main-tree du cache.

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"levelup/go-api/internal/analysis"
)

// smokeMatchShort8 — 8 premiers chars de l'UUID du match de référence.
const smokeMatchShort8 = "000d5950"

// smokeChunkManifest — manifeste film avec les champs nécessaires à l'orchestrateur
// (chunk_type + duration_ms en plus de index/start_ms). Distinct de filmManifest
// (melee_scanner_test.go) qui n'expose pas ces deux champs.
type smokeChunkManifest struct {
	Chunks []struct {
		Index      int `json:"index"`
		ChunkType  int `json:"chunk_type"`
		StartMS    int `json:"start_ms"`
		DurationMS int `json:"duration_ms"`
	} `json:"chunks"`
}

// loadSmokeChunkInputs charge le manifeste + les chunks décompressés du match de
// référence en ChunkInput, ou nil si le cache est absent (déclenche skip côté test).
func loadSmokeChunkInputs(t *testing.T) []ChunkInput {
	t.Helper()
	root := filmCacheRoot(t)
	if root == "" {
		return nil
	}
	// Les manifests sont dans data/cache/film_manifests, soeur de film_chunks.
	manPath := filepath.Join(filepath.Dir(root), "film_manifests", smokeMatchShort8+".json")
	mb, err := os.ReadFile(manPath)
	if err != nil {
		return nil
	}
	var m smokeChunkManifest
	if err := json.Unmarshal(mb, &m); err != nil {
		t.Fatalf("manifest %s illisible: %v", smokeMatchShort8, err)
	}
	var chunks []ChunkInput
	for _, c := range m.Chunks {
		data := loadCachedChunk(t, smokeMatchShort8, fmt.Sprintf("chunk_%02d.bin", c.Index))
		if data == nil {
			continue
		}
		chunks = append(chunks, ChunkInput{
			Index:      c.Index,
			Data:       data,
			StartMS:    c.StartMS,
			DurationMS: c.DurationMS,
			ChunkType:  c.ChunkType,
		})
	}
	return chunks
}

// synthesizeSmokeKills fabrique des kills plausibles répartis dans la fenêtre
// gameplay (un par joueur du roster, 30s → 345s). On ne flagge ni melee ni grenade :
// la corrélation fire-event joue, et les overlays décident le cas échéant.
func synthesizeSmokeKills() []analysis.Kill {
	kills := make([]analysis.Kill, 0, len(piverifyXuids))
	for i, x := range piverifyXuids {
		kills = append(kills, analysis.Kill{
			MatchID: smokeMatchShort8,
			XUID:    strconv.FormatUint(x, 10),
			TimeMS:  30000 + i*45000,
		})
	}
	return kills
}

// hammerWID / swordWID — weapon-ids 64-bit (high-32 connu + suffixe arbitraire)
// utilisés par les tests purs de récupération melee.
const (
	hammerWID uint64 = 0x841AC5E5_D8D07CA1 // Rushdown Hammer
	swordWID  uint64 = 0x4FF3937E_42C9679F // Energy Sword
)

// ptrU64 — helper pointeur pour fabriquer des attributions de test.
func ptrU64(v uint64) *uint64 { return &v }

// TestRecoverMeleeOnNone_GatesAndClaims vérifie le coeur de la récupération melee
// (pure, sans cache) : (1) seuls les kills NONE sont touchés ; (2) un kill déjà
// attribué fire est INTOUCHÉ ; (3) le pi doit matcher ; (4) la fenêtre serrée
// gate ; (5) claim-and-remove (un swing = un kill) ; (6) les whiffs 0x42 sont
// ignorés.
func TestRecoverMeleeOnNone_GatesAndClaims(t *testing.T) {
	xuidToPI := map[string]int{
		"alice": 3, // a un MeleeHit létal proche → doit récupérer
		"bob":   5, // kill déjà fire → intouché
		"carol": 7, // MeleeHit hors fenêtre → reste NONE
		"dave":  9, // seulement un whiff 0x42 → reste NONE
	}
	attrs := []analysis.KillAttribution{
		{XUID: "alice", TimeMS: 1000},
		{XUID: "bob", TimeMS: 2000},
		{XUID: "carol", TimeMS: 3000},
		{XUID: "dave", TimeMS: 4000},
		{XUID: "alice", TimeMS: 1200}, // 2e kill alice : pas de 2e hammer libre → NONE
	}
	out := []AttributionV3{
		{XUID: "alice", TimeMS: 1000},                           // NONE
		{XUID: "bob", TimeMS: 2000, WeaponID: ptrU64(swordWID)}, // déjà résolu (fire)
		{XUID: "carol", TimeMS: 3000},                           // NONE
		{XUID: "dave", TimeMS: 4000},                            // NONE
		{XUID: "alice", TimeMS: 1200},                           // NONE
	}
	melees := []MeleeHit{
		{PI: 3, TimeMS: 1050, WeaponID: hammerWID, HitType: meleeHitHammer}, // alice@1000 : Δ50 OK
		{PI: 5, TimeMS: 1980, WeaponID: swordWID, HitType: meleeHitSword},   // bob : mais bob déjà résolu
		{PI: 7, TimeMS: 3700, WeaponID: swordWID, HitType: meleeHitSword},   // carol : Δ700 > 500 → out
		{PI: 9, TimeMS: 4010, WeaponID: hammerWID, HitType: meleeHitMiss},   // dave : whiff → ignoré
	}

	recoverMeleeOnNone(out, attrs, xuidToPI, melees)

	// alice@1000 récupéré en HIGH melee, arme = hammer.
	a0 := out[0]
	if a0.SourceSignal != SignalMelee || a0.Confidence != confidenceHighV3 {
		t.Fatalf("alice@1000 : attendu melee/high, obtenu %s/%s", a0.SourceSignal, a0.Confidence)
	}
	if a0.WeaponID == nil || *a0.WeaponID != hammerWID || a0.HighWeaponID == nil {
		t.Fatalf("alice@1000 : arme hammer attendue, obtenu %+v", a0)
	}
	// bob intouché (déjà fire) : pas de SourceSignal melee, arme inchangée.
	if out[1].SourceSignal == SignalMelee || out[1].WeaponID == nil || *out[1].WeaponID != swordWID {
		t.Fatalf("bob : kill fire ne doit JAMAIS être reclassé melee, obtenu %+v", out[1])
	}
	// carol hors fenêtre → reste NONE.
	if out[2].WeaponID != nil || out[2].SourceSignal == SignalMelee {
		t.Fatalf("carol : hit hors fenêtre, doit rester NONE, obtenu %+v", out[2])
	}
	// dave whiff → reste NONE.
	if out[3].WeaponID != nil || out[3].SourceSignal == SignalMelee {
		t.Fatalf("dave : whiff 0x42 ne doit pas récupérer, obtenu %+v", out[3])
	}
	// alice@1200 : le seul hammer libre a été claim par alice@1000 (plus proche) →
	// claim-and-remove garantit qu'il n'est pas réutilisé.
	if out[4].WeaponID != nil || out[4].SourceSignal == SignalMelee {
		t.Fatalf("alice@1200 : hit déjà réclamé, doit rester NONE, obtenu %+v", out[4])
	}
}

// TestRecoverMeleeOnNone_NearestWins vérifie que chaque hit va au kill le plus
// proche en temps quand plusieurs kills NONE du même pi sont candidats.
func TestRecoverMeleeOnNone_NearestWins(t *testing.T) {
	xuidToPI := map[string]int{"alice": 3}
	attrs := []analysis.KillAttribution{
		{XUID: "alice", TimeMS: 1000},
		{XUID: "alice", TimeMS: 1400},
	}
	out := []AttributionV3{
		{XUID: "alice", TimeMS: 1000},
		{XUID: "alice", TimeMS: 1400},
	}
	// Un seul hammer à 1380 : plus proche de 1400 (Δ20) que de 1000 (Δ380).
	melees := []MeleeHit{{PI: 3, TimeMS: 1380, WeaponID: hammerWID, HitType: meleeHitHammer}}

	recoverMeleeOnNone(out, attrs, xuidToPI, melees)

	if out[1].SourceSignal != SignalMelee {
		t.Fatalf("kill@1400 (le plus proche) doit récupérer le hit, obtenu %+v", out[1])
	}
	if out[0].SourceSignal == SignalMelee {
		t.Fatalf("kill@1000 (plus loin) ne doit pas voler le hit, obtenu %+v", out[0])
	}
}

func TestBuildV3Attributions_Smoke000d5950(t *testing.T) {
	chunks := loadSmokeChunkInputs(t)
	if len(chunks) == 0 {
		t.Skip("cache film 000d5950 absent — smoke skippé (validation chiffrée = couche CLI)")
	}

	kills := synthesizeSmokeKills()
	in := V3Input{
		MatchID:     smokeMatchShort8,
		Kills:       kills,
		RosterXuids: piverifyXuids,
		Chunks:      chunks,
	}

	out := BuildV3Attributions(in) // ne doit pas paniquer

	if len(out) != len(kills) {
		t.Fatalf("cardinalité: out=%d, kills=%d (attendu égal)", len(out), len(kills))
	}

	// Au moins une attribution doit porter une arme via un signal DIRECT/fire :
	// HighWeaponID non nil + SourceSignal ∈ {fire, melee}.
	var withWeapon int
	for _, v := range out {
		signalOK := v.SourceSignal == SignalFire || v.SourceSignal == SignalMelee
		if v.HighWeaponID != nil && signalOK {
			withWeapon++
			// Cohérence : un high-32 exposé doit être une arme connue (anti-bruit).
			if WeaponName(*v.HighWeaponID) == "" {
				t.Errorf("HighWeaponID=0x%08x exposé mais inconnu de KnownWeaponHigh32 (kill xuid=%s @%dms)",
					*v.HighWeaponID, v.XUID, v.TimeMS)
			}
		}
	}
	if withWeapon == 0 {
		t.Fatalf("aucune attribution fire/melee avec HighWeaponID (sur %d kills) — pipeline cassé", len(out))
	}
	t.Logf("smoke OK: %d/%d kills attribués via fire/melee avec arme connue", withWeapon, len(out))
}
