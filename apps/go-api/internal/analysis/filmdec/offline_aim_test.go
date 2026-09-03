package filmdec

import (
	"math"
	"testing"
)

// writeBipedRecordWithDirs écrit un record biped dont le masque déclare [i0, i1, i21] et
// dont la queue porte la vélocité (direction cubemap + magnitude) puis la visée
// (cap 12 bits + élévation 11 bits), conformément à la grammaire mesurée sur le film.
func writeBipedRecordWithDirs(w *bitWriter, slot uint32, velDir, velScale, yaw, pitch uint64) {
	w.bits(1, 1)
	w.bits(uint64(slot), bipedSlotBits)
	w.bits(1, 2) // tag
	w.bits(0, 2) // gate + maskSel
	w.bits(3, 3) // maskCount = 3
	for _, id := range []uint64{0, 1, 21} {
		w.bits(id, bipedIndexBits)
	}
	w.bits(0, cliffLayout.GateBits) // i0 absolu
	w.bits(4096, int(cliffLayout.AxisW[0]))
	w.bits(5000, int(cliffLayout.AxisW[1]))
	w.bits(8192, int(cliffLayout.AxisW[2]))
	w.bits(0, i0TailBits) // queue i0 : handleSel + regionPresent
	// i1 : outer=0 (chemin dynamic-precision), absent=0 (direction présente)
	w.bits(0, 1)
	w.bits(0, 1)
	w.bits(velDir, int(aimDirBits))
	w.bits(velScale, velScaleBits)
	// i21 : flag0 puis cap R(12) et élévation R(11)
	w.bits(0, 1)
	w.bits(yaw, aimYawBits)
	w.bits(pitch, aimPitchBits)
	w.bits(0, 32) // queue
}

// TestScanBipedRecords_CaptureDirs : la capture des directions retrouve exactement les
// champs écrits, et reste inerte quand l'option est désactivée.
func TestScanBipedRecords_CaptureDirs(t *testing.T) {
	const slot uint32 = 517
	const velDir, velScale, yaw, pitch = 123456, 512, 1024, 700
	w := &bitWriter{}
	w.bits(0, 5) // bruit de tête
	writeBipedRecordWithDirs(w, slot, velDir, velScale, yaw, pitch)

	opt := scanOptWorld()
	opt.CaptureDirs = true
	got := ScanBipedRecords(w.buf, map[uint32]bool{slot: true}, cliffLayout, opt)
	if len(got) != 1 {
		t.Fatalf("attendu 1 record, obtenu %d", len(got))
	}
	r := got[0]
	if !r.HasVel || r.VelRaw != velDir || r.VelScale != velScale {
		t.Errorf("vélocité = (%v, %d, %d), attendu (true, %d, %d)", r.HasVel, r.VelRaw, r.VelScale, velDir, velScale)
	}
	if !r.HasYaw || r.YawRaw != yaw || r.PitchRaw != pitch {
		t.Errorf("visée = (%v, %d, %d), attendu (true, %d, %d)", r.HasYaw, r.YawRaw, r.PitchRaw, yaw, pitch)
	}
	h, ok := r.AimHeadingDeg()
	wantH := float32(360 * (float64(yaw) + 0.5) / 4096)
	if !ok || math.Abs(float64(h-wantH)) > 1e-3 {
		t.Errorf("cap = %.4f (ok=%v), attendu %.4f", h, ok, wantH)
	}
	if v, ok := r.VelocityVector(); !ok || v == [3]float32{} {
		t.Errorf("vecteur vélocité = %v (ok=%v)", v, ok)
	}

	// CaptureDirs désactivé : mêmes positions, aucune direction (non-régression).
	off := scanOptWorld()
	plain := ScanBipedRecords(w.buf, map[uint32]bool{slot: true}, cliffLayout, off)
	if len(plain) != 1 || plain[0].X != r.X || plain[0].Y != r.Y || plain[0].Z != r.Z {
		t.Fatalf("les positions diffèrent selon CaptureDirs : %+v vs %+v", plain, got)
	}
	if plain[0].HasVel || plain[0].HasYaw {
		t.Errorf("directions capturées alors que CaptureDirs est faux : %+v", plain[0])
	}
}

// TestScanRecordDirs_StopsOnUnknownComponent : un composant non modélisé entre i0 et la
// direction interrompt la capture au lieu de décoder du bruit.
func TestScanRecordDirs_StopsOnUnknownComponent(t *testing.T) {
	pay := make([]byte, 64)
	out, vit := scanRecordDirs(pay, 0, len(pay)*8, []int{0, 7, 21}, dirsGrammar{})
	if out.HasVel || out.HasYaw || out.HasAim {
		t.Errorf("capture après un composant inconnu : %+v", out)
	}
	if vit.HasBody || vit.HasShield {
		t.Errorf("vitalité capturée après un composant inconnu : %+v", vit)
	}
}

// TestScanRecordVitals_ReachedThroughAngularVelocity : i4/i5 se lisent bien APRÈS i3, dont
// le seul rôle ici est d'être traversé. Le flux est fabriqué bit à bit — c'est la seule
// façon de vérifier l'ENCHAÎNEMENT (le film réel ne dit pas où commence i4).
func TestScanRecordVitals_ReachedThroughAngularVelocity(t *testing.T) {
	var w bitWriter
	w.bits(0, i0TailBits) // queue d'i0
	w.bits(1, 1)          // i3 : gate == 1 -> absent, zéro bit de charge utile
	w.bits(254, 8)        // i4 : quantum de santé = max
	w.bits(0, 3)          // i4 : trois drapeaux
	w.bits(255, 8)        // i5 : quantum de bouclier = max
	w.bits(0, 1)          // i5 : pas de bloc de regen
	w.bits(0x1234, 16)    // i5 : mot inline
	w.bits(0b1010, 4)     // i5 : quatre drapeaux
	_, vit := scanRecordDirs(w.buf, 0, len(w.buf)*8, []int{0, 3, 4, 5}, dirsGrammar{})
	if !vit.HasBody || !vit.HasShield {
		t.Fatalf("vitalité non atteinte à travers i3 : %+v", vit)
	}
	if vit.Body.Health != VitalityBodyMax {
		t.Errorf("santé = %v, attendu %v (q=254 = extrémité exacte)", vit.Body.Health, VitalityBodyMax)
	}
	if vit.Shield.Shield != VitalityShieldMax {
		t.Errorf("bouclier = %v, attendu %v (q=255 = extrémité exacte)", vit.Shield.Shield, VitalityShieldMax)
	}
	if vit.Shield.Block64 != 0x1234 {
		t.Errorf("mot inline du bouclier = %04x — l'enchaînement des bits est décalé", vit.Shield.Block64)
	}
}
