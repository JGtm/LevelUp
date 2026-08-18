package replay

import (
	"encoding/json"
	"math"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

// build_aim_test.go — LE CONTRAT D ECRITURE DES DEUX ANGLES DE VISEE (lot E phase 1).
//
// Ce que ces tests ferment : l arrondi, le SENS, l omission, et le CABLAGE. Les quatre se
// verifient separement parce qu ils tombent separement — un arrondi juste sur un champ jamais
// alimente rendrait un document a plat sans qu aucun test unitaire ne bronche.

// TestPitchForJSONArrondiAuDixieme : la valeur publiee est celle du dixieme le plus proche.
func TestPitchForJSONArrondiAuDixieme(t *testing.T) {
	for _, c := range []struct{ in, want float32 }{
		{12.34, 12.3},
		{12.36, 12.4},
		{-12.34, -12.3},
		{-12.36, -12.4},
		{0.0879, 0.1},   // le plus petit pas positif que la source puisse produire
		{-0.0879, -0.1}, // et son symetrique
	} {
		if got := pitchForJSON(c.in); got != c.want {
			t.Errorf("pitchForJSON(%v) = %v, attendu %v", c.in, got, c.want)
		}
	}
}

// TestPitchForJSONOmetLaVisceAPlat : sous 0,05 deg, le champ RETOURNE ZERO, donc `omitempty`
// l omet — et l absence se lit « a plat ».
//
// C est le contraire du geste de headingForJSON, teste ici cote a cote pour que l opposition
// soit verrouillee : un cap qui s arrondit a 0 est REPECHE en 360, une elevation qui s arrondit
// a 0 est LAISSEE a 0. Inverser l un des deux casse ce test.
func TestPitchForJSONOmetLaViseeAPlat(t *testing.T) {
	for _, in := range []float32{0, 0.04, -0.04, 0.049, -0.049} {
		if got := pitchForJSON(in); got != 0 {
			t.Errorf("pitchForJSON(%v) = %v, attendu 0 (donc omis : l absence dit « a plat »)", in, got)
		}
		if math.Signbit(float64(pitchForJSON(in))) {
			t.Errorf("pitchForJSON(%v) rend un ZERO NEGATIF : il se serialiserait en \"-0\" au lieu "+
				"de s omettre", in)
		}
	}
	if got := headingForJSON(0); got != 360 {
		t.Errorf("headingForJSON(0) = %v, attendu 360 : le CAP, lui, se repeche — les deux regles "+
			"ne doivent jamais converger", got)
	}
}

// TestPointOmetPQuandLaViseeEstAPlat : la regle d ecriture et la SERIALISATION disent la meme
// chose. Un test sur la seule fonction laisserait passer un tag JSON sans `omitempty`.
func TestPointOmetPQuandLaViseeEstAPlat(t *testing.T) {
	plat, err := json.Marshal(Point{T: 1, X: 2, Y: 3, H: 90, P: pitchForJSON(0.04)})
	if err != nil {
		t.Fatalf("serialisation : %v", err)
	}
	if strings.Contains(string(plat), `"p"`) {
		t.Errorf("le point a plat porte quand meme `p` : %s", plat)
	}
	if !strings.Contains(string(plat), `"h"`) {
		t.Errorf("le point a plat a perdu son CAP : %s — « a plat » n est pas « sans visee »", plat)
	}
	penchee, err := json.Marshal(Point{T: 1, X: 2, Y: 3, H: 90, P: pitchForJSON(-12.36)})
	if err != nil {
		t.Fatalf("serialisation : %v", err)
	}
	if !strings.Contains(string(penchee), `"p":-12.4`) {
		t.Errorf("la plongee n est pas publiee telle quelle : %s", penchee)
	}
}

// TestPitchPublieEstToujoursHorsDeLaBandeDOmission : CE QUE LA SOURCE PEUT PRODUIRE.
//
// La regle d omission est le contrat du CLIENT (absent = a plat), et ce test mesure ce que le
// PRODUCTEUR en fait reellement : sur les 2 048 valeurs que le champ R(11) peut prendre, AUCUNE
// ne tombe dans la bande omise. La formule vaut 360*(raw + 0,5)/2048 - 180, et le demi-pas
// empeche tout raw de rendre exactement 0 : le plus proche vaut +/- 0,0879 deg, soit 1,76 fois
// le seuil. `p` accompagne donc TOUJOURS `h` dans un artefact cuit depuis un film.
//
// Ce n est pas une raison de retirer la regle : elle protege le client d un producteur futur
// (autre titre, autre largeur de champ) et elle donne son sens a l absence. Mais elle DOIT etre
// ecrite, sans quoi le cout du champ serait sous-estime a la lecture du code.
func TestPitchPublieEstToujoursHorsDeLaBandeDOmission(t *testing.T) {
	var omis int
	minAbs := float32(math.MaxFloat32)
	for raw := uint32(0); raw < 2048; raw++ {
		var p filmdec.BipedPosition
		p.HasYaw = true
		p.PitchRaw = raw
		v, ok := p.AimPitchDeg()
		if !ok {
			t.Fatalf("raw %d : accesseur invalide alors que HasYaw est vrai", raw)
		}
		if abs := float32(math.Abs(float64(v))); abs < minAbs {
			minAbs = abs
		}
		if pitchForJSON(v) == 0 {
			omis++
		}
	}
	if omis != 0 {
		t.Errorf("%d valeur(s) de raw sur 2048 tombent dans la bande omise — la mesure disait 0", omis)
	}
	if minAbs < 0.08 || minAbs > 0.09 {
		t.Errorf("plus petite elevation absolue = %v deg, attendue ~0,0879 (le demi-pas de la "+
			"formule) : la convention de decodage a change", minAbs)
	}
}

// TestDocumentPortePEtSonSigne : LE CABLAGE. Sans la ligne de `decimateTracks`, tout le reste
// reste vert et l artefact est a plat.
//
// Les trois positions portent le MEME cap et trois elevations distinctes ; le test verifie le
// signe (au-dessus de 1024 = vers le haut), la valeur, et le fait que `h` et `p` voyagent
// ensemble.
func TestDocumentPortePEtSonSigne(t *testing.T) {
	mk := func(ts uint64, x float32, pitch uint32) filmdec.BipedPosition {
		var p filmdec.BipedPosition
		p.Slot, p.TimestampUS, p.X, p.Y, p.Z, p.HasWorld = 7, ts, x, 1, 0, true
		p.HasYaw, p.YawRaw, p.PitchRaw = true, 1024, pitch
		return p
	}
	doc := BuildFromPositions("m", "halo_infinite", []filmdec.BipedPosition{
		mk(1_000_000, 1, 1500), // vers le HAUT
		mk(1_200_000, 2, 500),  // vers le BAS
		mk(1_400_000, 3, 1024), // quasi a plat, mais publie (cf. test precedent)
	}, nil, Options{})

	if len(doc.Tracks) != 1 {
		t.Fatalf("%d trace(s) publiee(s), attendu 1", len(doc.Tracks))
	}
	pts := doc.Tracks[0].Points
	if len(pts) != 3 {
		t.Fatalf("%d point(s), attendu 3", len(pts))
	}
	for i, want := range []float32{83.8, -92, 0.1} {
		if pts[i].P != want {
			t.Errorf("point %d : p = %v, attendu %v", i, pts[i].P, want)
		}
		if pts[i].H == 0 {
			t.Errorf("point %d : cap perdu alors que l elevation est publiee — les deux angles "+
				"viennent du MEME composant", i)
		}
	}
	if pts[0].P <= 0 || pts[1].P >= 0 {
		t.Errorf("le SIGNE est inverse : raw 1500 rend %v et raw 500 rend %v, alors que au-dessus "+
			"de 1024 = vers le HAUT", pts[0].P, pts[1].P)
	}
}

// TestDocumentSansViseeNePubliePasDElevation : un record sans i21 ne fabrique aucune elevation.
// Le temoin negatif du cablage : sans lui, un `pt.P = pitchForJSON(...)` pose hors du `if`
// publierait -180 deg partout (PitchRaw valant 0 par defaut).
func TestDocumentSansViseeNePubliePasDElevation(t *testing.T) {
	mk := func(ts uint64, x float32) filmdec.BipedPosition {
		var p filmdec.BipedPosition
		p.Slot, p.TimestampUS, p.X, p.Y, p.Z, p.HasWorld = 7, ts, x, 1, 0, true
		return p
	}
	doc := BuildFromPositions("m", "halo_infinite", []filmdec.BipedPosition{
		mk(1_000_000, 1), mk(1_200_000, 2),
	}, nil, Options{})
	if len(doc.Tracks) != 1 {
		t.Fatalf("%d trace(s) publiee(s), attendu 1", len(doc.Tracks))
	}
	for i, pt := range doc.Tracks[0].Points {
		if pt.P != 0 || pt.H != 0 {
			t.Errorf("point %d : h = %v, p = %v — un record sans i21 ne porte AUCUN des deux angles",
				i, pt.H, pt.P)
		}
	}
}
