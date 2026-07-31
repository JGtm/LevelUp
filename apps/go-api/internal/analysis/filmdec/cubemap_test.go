package filmdec

import (
	"math"
	"math/rand"
	"testing"
)

// TestCubemapTableLaws vérifie que la table figée respecte les deux lois lues dans l'exe :
// faceSize(p) = floor(2^p/6) (colonne lue octet à octet) et gridSize(p) =
// floor(sqrt(2^p/6)) - 1 (initialiseur FUN_14038bc40), plus les invariants de capacité.
func TestCubemapTableLaws(t *testing.T) {
	for w := cubemapMinWidth; w <= cubemapMaxWidth; w++ {
		fs, n, ok := cubemapParams(w)
		if !ok {
			t.Fatalf("width %d: hors table", w)
		}
		wantFS := int(uint64(1)<<w) / 6
		if fs != wantFS {
			t.Errorf("faceSize(%d) = %d, attendu %d", w, fs, wantFS)
		}
		wantN := int(math.Sqrt(float64(uint64(1)<<w)/6.0)) - 1
		if n != wantN {
			t.Errorf("gridSize(%d) = %d, attendu %d", w, n, wantN)
		}
		if 6*uint64(fs) > uint64(1)<<w {
			t.Errorf("width %d: 6*faceSize dépasse 2^%d", w, w)
		}
		if (n-2)*n+(n-2) >= fs {
			t.Errorf("width %d: la grille (%d) déborde de la face (%d)", w, n, fs)
		}
	}
	if _, _, ok := cubemapParams(5); ok {
		t.Error("width 5 devrait être rejetée (entrée (0,0) dans l'exe)")
	}
	if _, _, ok := cubemapParams(31); ok {
		t.Error("width 31 devrait être rejetée (hors table)")
	}
}

// TestDecodeAimVectorBlogSample décode l'exemple publié par le dev externe :
//
//	code = (0x6240e840e0 >> 5) & 0x3FFFFFFF = 302465543, largeur 30.
//
// Sa version (2e coordonnée forcée à 0, FACE_SIZE=0xAAA8000) donne (0.3556, 0.9346, 0).
// La nôtre récupère la 2e coordonnée via le 2e divmod : l'écart attendu est ~10° ici
// (et jusqu'à 45° aux coutures). Le test DOCUMENTE cet écart, il ne force pas l'égalité.
func TestDecodeAimVectorBlogSample(t *testing.T) {
	const code = uint32((0x6240e840e0 >> 5) & 0x3FFFFFFF)
	if code != 302465543 {
		t.Fatalf("code attendu 302465543, obtenu %d", code)
	}
	v, ok := DecodeAimVectorChecked(code, 30)
	if !ok {
		t.Fatal("décodage invalide")
	}
	want := [3]float32{0.3503, 0.9200, 0.1758} // valeur de l'exe (grammaire triangulée)
	for i, c := range want {
		if math.Abs(float64(v[i]-c)) > 1e-3 {
			t.Errorf("composante %d = %.4f, attendu %.4f (v=%v)", i, v[i], c, v)
		}
	}
	// Écart chiffré vs la version communautaire (b=0) : ~10 degrés sur cet échantillon.
	flat, _ := DecodeAimVectorFlat(code, 30)
	cos := float64(v[0]*flat[0] + v[1]*flat[1] + v[2]*flat[2])
	deg := math.Acos(math.Min(1, cos)) * 180 / math.Pi
	if deg < 5 || deg > 15 {
		t.Errorf("écart avec la version 2e-coordonnée-nulle = %.2f°, attendu ~10°", deg)
	}
	t.Logf("exe=%v communautaire=%v écart=%.2f°", v, flat, deg)
}

// TestDecodeAimVectorUnit : toute direction décodée doit être unitaire.
func TestDecodeAimVectorUnit(t *testing.T) {
	for _, w := range []uint{6, 12, 19, 24, 30} {
		fs, _, _ := cubemapParams(w)
		for _, code := range []int{0, 1, fs / 2, fs, 3 * fs, 6*fs - 1} {
			v, ok := DecodeAimVectorChecked(uint32(code), w)
			if !ok {
				t.Errorf("width %d code %d: invalide", w, code)
				continue
			}
			m := math.Sqrt(float64(v[0]*v[0] + v[1]*v[1] + v[2]*v[2]))
			if math.Abs(m-1) > 1e-5 {
				t.Errorf("width %d code %d -> %v, |v|=%.6f", w, code, v, m)
			}
		}
		// face >= 6 : la sentinelle de l'exe (0,0,1) + ok=false (cliquet d'alignement).
		if v, ok := DecodeAimVectorChecked(uint32(6*fs), w); ok || v != [3]float32{0, 0, 1} {
			t.Errorf("width %d: face 6 devrait être invalide, obtenu %v ok=%v", w, v, ok)
		}
	}
}

// TestDecodeAimVectorRoundTrip : encode(decode) doit retomber sur la même direction à la
// résolution de cellule près, sur toutes les largeurs légales (test le plus fort : il
// casse immédiatement si faceSize ou gridSize est faux).
func TestDecodeAimVectorRoundTrip(t *testing.T) {
	rnd := rand.New(rand.NewSource(1))
	for w := cubemapMinWidth; w <= cubemapMaxWidth; w++ {
		_, n, _ := cubemapParams(w)
		// ~1 cellule d'erreur max, avec un plancher de 0,1° : au-delà de p=28 la cellule
		// est plus fine que la résolution du float32 utilisé par l'exe (et par ce port).
		errRad := math.Max(2.2*math.Atan(2.0/float64(n-1)), 0.1*math.Pi/180)
		tol := math.Cos(errRad)
		worst := 1.0
		for i := 0; i < 3000; i++ {
			v := randomUnit(rnd)
			enc, ok := EncodeAimVector(v, w)
			if !ok {
				t.Fatalf("width %d: encodage refusé", w)
			}
			dec, ok := DecodeAimVectorChecked(enc, w)
			if !ok {
				t.Fatalf("width %d: décodage de %d invalide", w, enc)
			}
			cos := float64(v[0]*dec[0] + v[1]*dec[1] + v[2]*dec[2])
			if cos < worst {
				worst = cos
			}
		}
		if worst < tol {
			t.Errorf("width %d: pire cos aller-retour %.6f < %.6f (%.2f° d'erreur)",
				w, worst, tol, math.Acos(worst)*180/math.Pi)
		}
	}
}

// TestDecodeAimVectorFlatIsWorse chiffre le gain de la 2e coordonnée : sur des directions
// aléatoires, l'aller-retour avec b=0 (version communautaire) doit être NETTEMENT moins
// fidèle que le décodage complet.
func TestDecodeAimVectorFlatIsWorse(t *testing.T) {
	rnd := rand.New(rand.NewSource(7))
	var sumFull, sumFlat float64
	const n = 5000
	for i := 0; i < n; i++ {
		v := randomUnit(rnd)
		enc, _ := EncodeAimVector(v, 19)
		full, _ := DecodeAimVectorChecked(enc, 19)
		flat, _ := DecodeAimVectorFlat(enc, 19)
		sumFull += angleDeg(v, full)
		sumFlat += angleDeg(v, flat)
	}
	mFull, mFlat := sumFull/n, sumFlat/n
	t.Logf("erreur angulaire moyenne : complet %.3f° | 2e coord nulle %.3f°", mFull, mFlat)
	if mFull > 0.5 {
		t.Errorf("décodage complet trop imprécis : %.3f°", mFull)
	}
	if mFlat < 10*mFull {
		t.Errorf("le gain de la 2e coordonnée n'apparaît pas (%.3f° vs %.3f°)", mFlat, mFull)
	}
}

func randomUnit(rnd *rand.Rand) [3]float32 {
	z := rnd.Float64()*2 - 1
	th := rnd.Float64() * 2 * math.Pi
	r := math.Sqrt(1 - z*z)
	return [3]float32{float32(r * math.Cos(th)), float32(r * math.Sin(th)), float32(z)}
}

func angleDeg(a, b [3]float32) float64 {
	cos := float64(a[0]*b[0] + a[1]*b[1] + a[2]*b[2])
	if cos > 1 {
		cos = 1
	}
	if cos < -1 {
		cos = -1
	}
	return math.Acos(cos) * 180 / math.Pi
}
