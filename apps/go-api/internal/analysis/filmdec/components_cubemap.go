package filmdec

import "math"

// Cubemap direction decode (PISTE 2) — port of FUN_1406d8288 (HaloInfinite.exe).
//
// The dynamic-precision vec3 family (velocity i1/i3, aim vectors, …) encodes a UNIT
// DIRECTION as a single packed integer of `width` bits via a cube→sphere mapping:
// pick one of 6 cube faces (the dominant signed axis), then two grid coordinates on
// that face. FUN_14076d528 reads R(width) then calls FUN_1406d8288(packed, out, width);
// the caller multiplies the returned unit vector by the scalar magnitude (FUN_14076d6dc)
// to obtain the full vector. This is the "cubemap" the external RE dev had at ~90%
// (blocked on the 2nd grid coord); the decompile gives BOTH divmods, so it is complete.
//
// Encoder (FUN_1407eaf1c, confirms the layout):
//   encoded = face*faceSize + u*gridSize + v   (faceSize is NOT gridSize² — see below)
// so decode is: face = enc/faceSize ; rem = enc%faceSize ; u = rem/gridSize ; v = rem%gridSize.
//
// Constants read from the .exe .rdata (static float32):
//   DAT_143cd8374 = 1.0   DAT_143cd8934 = 2.0 (coord range)   DAT_143cd84b0 = 0.5
//   DAT_143cd84ec = -1.0  DAT_143cd837c = 1e-4 (normalize epsilon)  DAT_143cd8370 = 0.0
//
// faceSize/gridSize live in the table DAT_1447084d0/DAT_1447084d4 indexed by `width`.
// CORRECTION 2026-07-25 : the earlier offline derivation here (faceSize = gridSize²,
// gridSize = max g with 6g² <= 2^width) was WRONG on BOTH columns — the real values are
// faceSize = floor(2^width/6) (read byte-for-byte in the .exe) and gridSize =
// floor(sqrt(2^width/6)) - 1 (FUN_14038bc40 initialiser). The single source of truth is
// now cubemapTable/DecodeAimVectorChecked in aim_vector.go; this file only wires the
// velocity family to it.

// DecodeDynPrecDir decodes a dynamic-precision packed direction (as delivered by the
// dynPrec capture hook) of `width` bits into a UNIT world direction via the cubemap.
// `width` is the family's magnitude-bit count (translational/angular velocity, aim = 19).
// The caller multiplies by the scalar magnitude to obtain the full vector.
func DecodeDynPrecDir(packed uint64, width uint) [3]float32 {
	v, _ := DecodeAimVectorChecked(uint32(packed), width)
	return v
}

// Translational-velocity magnitude range (world-units/s), read from the .exe: the
// FUN_14076d4d0 call site passes DAT_143cd88f8=0.03 (min) and DAT_143cd88fc=350.0 (max)
// to FUN_14076d528 -> FUN_14076d6dc, with scale width 10.
const (
	velMagMin       = float32(0.03)
	velMagMax       = float32(350.0)
	velScaleBitsDef = uint(10)
)

// DecodeVelocityMagnitude decodes the translational-velocity speed scalar from its
// `width`-bit quantized `scale`, mirroring FUN_14076d6dc: a LOG/EXP interpolation between
// velMagMin and velMagMax. scale==0 -> min ; scale>=2^width-1 -> max ; else
// exp((scale+0.5)·log(1-min+max)/2^width) - (1-min).
func DecodeVelocityMagnitude(scale uint64, width uint) float32 {
	n := uint64(1) << width
	if scale == 0 {
		return velMagMin
	}
	if scale >= n-1 {
		return velMagMax
	}
	oneMinusMin := float32(1.0) - velMagMin
	step := float32(math.Log(float64(oneMinusMin+velMagMax))) / float32(n)
	return float32(math.Exp(float64(float32(scale)*step+step*0.5))) - oneMinusMin
}

// DecodeVelocity combines the cubemap direction (19-bit packed dir) and the log/exp
// magnitude (10-bit scale) into a full translational-velocity vector (world-units/s),
// as FUN_14076d528 does (unit dir * FUN_14076d6dc magnitude). This is the datum that
// dead-reckons a player's position through keep-baseline frames.
func DecodeVelocity(packedDir, scale uint64) [3]float32 {
	d := DecodeDynPrecDir(packedDir, velocityMagBits)
	m := DecodeVelocityMagnitude(scale, velScaleBitsDef)
	return [3]float32{d[0] * m, d[1] * m, d[2] * m}
}

func absf(f float32) float32 {
	if f < 0 {
		return -f
	}
	return f
}

func clampi(x, lo, hi int) int {
	if x < lo {
		return lo
	}
	if x > hi {
		return hi
	}
	return x
}
