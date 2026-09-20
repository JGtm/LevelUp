//go:build research

package grammar

// avant_chassis_54_research_test.go — LOT 5.4 : L AVANT DU CHASSIS, MESURE SUR MINI-BOBINES.
//
// AUCUN FILM DU CACHE, AUCUNE BASE : les sept mini-bobines par build de `replay/testdata`.
//
// # CE QUE CET INSTRUMENT MESURE
//
// Le lot 5.2b.2 a mesure l azimut de la DIRECTION ecrite par `i2` et l a trouve indiscernable du
// hasard face au deplacement. La cause est desormais NOMMEE et relue au decompile : la direction
// ecrite est le vecteur HAUT, et l AVANT est la perpendiculaire que `FUN_1406d8678` reconstruit
// a partir d elle ET d un ANGLE DE ROULIS que le depot lisait puis JETAIT.
//
// L oracle est le meme que celui de 5.2b.2, et c est le seul honnete : la DIRECTION DE
// DEPLACEMENT sur les echantillons ou le vehicule avance nettement. Le temoin est le meme :
// l avant d un AUTRE echantillon, qui doit rendre une mediane d environ 90 degres.
//
//	go test ./internal/games/halo_infinite/film/internal/grammar/ -run TestAvantChassisMiniBobines -v -count=1

import (
	"math"
	"path/filepath"
	"sort"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// avantSeuilVitesseMPS : le seuil de l oracle V1a.3, celui sous lequel `vehicle_tracks.go`
// refuse deja de faire un cap d une velocite.
const avantSeuilVitesseMPS = 5.0

type avantEchantillon struct {
	capAvant float64 // degres, azimut de l avant reconstruit
	capVel   float64 // degres, azimut de la velocite
	vitesse  float64
}

type avantVentilation struct {
	positions  int
	avecI2     int // i2 lu (un mode connu)
	mode0      int
	mode1      int
	mode2      int
	avecRoulis int
	aPlat      int // porte de direction posee : haut = (0,0,1)
	avecAvant  int // avant reconstructible
	avecVel    int
	zHaut      []float64
	ech        []avantEchantillon
}

func TestAvantChassisMiniBobines(t *testing.T) {
	total := avantVentilation{}
	for _, court := range closureMiniFilms() {
		v := avantMesurerBobine(t, court)
		if v == nil {
			continue
		}
		avantJournaliser(t, court, v)
		avantCumuler(&total, v)
	}
	t.Log("================ CUMUL DES MINI-BOBINES ================")
	avantJournaliser(t, "CUMUL", &total)
}

// TestAvantFeuille4MiniBobines — LA FEUILLE 4 DE L ETAT PAR DEFAUT DE `ti=40`, LUE EN VALEUR.
//
// Les mini-bobines ne portent AUCUN paquet delta exploitable pour `ti=40` (le nuage de positions
// y est vide, mesure de ce lot) : la cadence d i2 ne s y mesure pas. Ce qu elles portent, ce sont
// les records d IMAGE-CLE — donc la feuille 4, derriere la porte `bVar14`.
//
// Et la feuille 4 porte LE MEME COUPLE (direction, angle) qu i2, a la meme convention : une
// position absolue (`FUN_14076e494`), puis `FUN_140c1e79c` = R(1) porte ; si 0 -> R(19) direction
// cubemap ; puis R(8) angle. Ce test la lit EN VALEUR et rend la verticalite du vecteur : si la
// direction ecrite est bien le HAUT, |z| doit etre proche de 1 sur des chassis au sol.
func TestAvantFeuille4MiniBobines(t *testing.T) {
	var zs []float64
	var caps []float64
	total, aPlat := 0, 0
	for _, court := range closureMiniFilms() {
		fc := avantContexte(t, court)
		if fc == nil {
			continue
		}
		if restore, err := InstallFilmFormatMPP(fc); err == nil {
			defer restore()
		}
		ctx := fc.ContexteDeLecture()
		n, plats := 0, 0
		for _, num := range fc.ChunkNumbers() {
			data, packets, ok := fc.ChunkAt(num)
			if !ok {
				continue
			}
			for _, pk := range packets {
				if pk.Type != PacketTypeKeyframe {
					continue
				}
				avantFeuille4UnPaquet(pk.Payload(data), ctx, &n, &plats, &zs, &caps)
			}
		}
		t.Logf("  %-10s feuille 4 lue sur %4d record(s) ti=40, dont %d a plat (haut = (0,0,1))",
			court, n, plats)
		total, aPlat = total+n, aPlat+plats
	}
	if total == 0 {
		t.Skip("aucune feuille 4 lisible sur les mini-bobines")
	}
	t.Logf("  CUMUL      %d feuille(s) 4, %d a plat (%.1f %%)", total, aPlat,
		100*float64(aPlat)/float64(total))
	if len(zs) > 0 {
		t.Logf("  |z| de la direction lue : mediane %.3f, part au-dessus de 0,9 = %.1f %% (n = %d)",
			avantMediane(zs), 100*avantPart(zs, func(z float64) bool { return z > 0.9 }), len(zs))
	}
	avantHistogrammeCaps(t, caps)
}

// avantFeuille4UnPaquet lit la feuille 4 de chaque record `ti=40` d image-cle dont la porte
// `bVar14` est posee, et rend la direction et l angle EN VALEUR.
func avantFeuille4UnPaquet(pay []byte, ctx ContexteDeLecture, n, plats *int, zs, caps *[]float64) {
	for _, b := range keyframeBornesToutes(pay) {
		if b.TI != VehicleTypeIndex {
			continue
		}
		br := LecteurSur(pay)
		br.PoserContexte(ctx)
		br.SetBitPos(b.Bit + br.cadre().EnTeteBits)
		mot := uint(br.cadre().MotDeTailleBits) //nolint:gosec // largeur de profil, bornee a 32
		if int32(br.ReadBits(mot)) <= 0 {       //nolint:gosec // n1, comparaison SIGNEE
			continue
		}
		consumeVersionPrefix(br)              // feuille 1
		consumeMultiplayerPropertiesBlock(br) // feuille 2
		if !br.ReadBit() {                    // feuille 3 : la porte bVar14
			continue
		}
		consumeSimStateHandleTail(br) // feuille 4, premiere moitie : la position absolue
		// feuille 4, seconde moitie : FUN_140c1e79c = R(1) porte ; si 0 -> R(19) ; puis R(8).
		up := [3]float32{0, 0, 1}
		if br.ReadBit() {
			*plats++
		} else {
			v, ok := DecodeAimVectorChecked(uint32(br.ReadBits(19)), 19)
			if !ok {
				continue
			}
			up = v
		}
		roll := RollAngleFromRaw(uint32(br.ReadBits(8)), 8)
		*n++
		*zs = append(*zs, math.Abs(float64(up[2])))
		f := ForwardFromUpRoll(up, roll)
		*caps = append(*caps, avantAzimut(float64(f[0]), float64(f[1])))
	}
}

// avantHistogrammeCaps : un cap RECONSTRUIT doit se repartir sur le tour. Une lecture a la
// mauvaise largeur, elle, s effondre sur quelques valeurs ou sur une seule face du cubemap.
func avantHistogrammeCaps(t *testing.T, caps []float64) {
	t.Helper()
	if len(caps) == 0 {
		return
	}
	var cases [12]int
	for _, c := range caps {
		i := int(c / 30)
		if i > 11 {
			i = 11
		}
		cases[i]++
	}
	t.Logf("  caps reconstruits, par secteur de 30 deg (n = %d) : %v", len(caps), cases)
}

func avantContexte(t *testing.T, court string) *FilmContext {
	t.Helper()
	dir := filepath.Join("..", "..", "replay", "testdata", "minifilm_"+court)
	film, err := source.LoadDir(dir, nil)
	if err != nil {
		t.Logf("  %-10s LoadDir : %v", court, err)
		return nil
	}
	cat, err := profile.LoadMapQuantCatalog(e191bCatalogue())
	if err != nil {
		t.Logf("  %-10s catalogue : %v", court, err)
		return nil
	}
	entree, err := cat.Lookup(e191bCarteDeBobine[court])
	if err != nil {
		t.Logf("  %-10s carte hors catalogue : %v", court, err)
		return nil
	}
	fc := NewFilmContextForMap(film, &entree, nil)
	bal := fc.ProfilDeBalayage()
	bal.PoserLargeursObjetDuMondeDepuisDecoupage(entree.Layout())
	fc.PoserProfilDeBalayage(bal)
	return fc
}

func avantMesurerBobine(t *testing.T, court string) *avantVentilation {
	t.Helper()
	dir := filepath.Join("..", "..", "replay", "testdata", "minifilm_"+court)
	film, err := source.LoadDir(dir, nil)
	if err != nil {
		t.Logf("  %-10s LoadDir : %v", court, err)
		return nil
	}
	cat, err := profile.LoadMapQuantCatalog(e191bCatalogue())
	if err != nil {
		t.Logf("  %-10s catalogue : %v", court, err)
		return nil
	}
	entree, err := cat.Lookup(e191bCarteDeBobine[court])
	if err != nil {
		t.Logf("  %-10s carte hors catalogue : %v", court, err)
		return nil
	}
	// MEME OUVERTURE QUE LA CUISSON (cf. `bv14Contexte`) : le decoupage d i0 vient du CATALOGUE,
	// pas de l auto-detection — une mini-bobine peut n avoir aucun bipede a detecter, et un
	// decoupage faux desynchronise le curseur AVANT i1 et i2.
	fc := NewFilmContextForMap(film, &entree, nil)
	bal := fc.ProfilDeBalayage()
	bal.PoserLargeursObjetDuMondeDepuisDecoupage(entree.Layout())
	fc.PoserProfilDeBalayage(bal)
	kf := ScanWorldObjectKeyframes(film, VehicleTypeIndex)
	if len(kf.Band) == 0 {
		t.Logf("  %-10s aucun slot ti=40 aux images-cles", court)
		return nil
	}
	plage := entree.Range()
	decoupage := entree.Layout()
	opt := DefaultScanFilmOptions()
	opt.RequireTag1 = false
	opt.CaptureDirs = true
	opt.DynPrecOrientation = true
	opt.WorldRange = &plage
	opt.Layout = &decoupage
	pos, err := ScanBipedPositionsForBand(fc, NewSlotBand(kf.Band), opt)
	if err != nil {
		t.Logf("  %-10s nuage illisible : %v", court, err)
		return nil
	}
	return avantVentiler(pos)
}

func avantVentiler(pos []BipedPosition) *avantVentilation {
	v := &avantVentilation{positions: len(pos)}
	for _, p := range pos {
		if p.HasAim || p.AimDefault || p.HasRoll {
			v.avecI2++
		}
		switch p.FwdMode {
		case 0:
			v.mode0++
		case 1:
			v.mode1++
		case 2:
			v.mode2++
		}
		if p.HasRoll {
			v.avecRoulis++
		}
		if p.AimDefault {
			v.aPlat++
		}
		if up, ok := p.AimVector(); ok {
			v.zHaut = append(v.zHaut, math.Abs(float64(up[2])))
		}
		fwd, okF := p.ChassisForwardVector()
		if okF {
			v.avecAvant++
		}
		vel, okV := p.VelocityVector()
		if okV {
			v.avecVel++
		}
		if !okF || !okV {
			continue
		}
		v.ech = append(v.ech, avantEchantillon{
			capAvant: avantAzimut(float64(fwd[0]), float64(fwd[1])),
			capVel:   avantAzimut(float64(vel[0]), float64(vel[1])),
			vitesse:  math.Hypot(float64(vel[0]), float64(vel[1])),
		})
	}
	return v
}

func avantCumuler(dst, src *avantVentilation) {
	dst.positions += src.positions
	dst.avecI2 += src.avecI2
	dst.mode0, dst.mode1, dst.mode2 = dst.mode0+src.mode0, dst.mode1+src.mode1, dst.mode2+src.mode2
	dst.avecRoulis += src.avecRoulis
	dst.aPlat += src.aPlat
	dst.avecAvant += src.avecAvant
	dst.avecVel += src.avecVel
	dst.zHaut = append(dst.zHaut, src.zHaut...)
	dst.ech = append(dst.ech, src.ech...)
}

func avantJournaliser(t *testing.T, nom string, v *avantVentilation) {
	t.Helper()
	t.Logf("  %-10s %6d position(s) ti=40 — i2 lu %d, mode0 %d, mode1 %d, mode2 %d, roulis %d, a plat %d",
		nom, v.positions, v.avecI2, v.mode0, v.mode1, v.mode2, v.avecRoulis, v.aPlat)
	t.Logf("  %-10s avant reconstructible %d, velocite %d, couples (avant ET velocite) %d",
		"", v.avecAvant, v.avecVel, len(v.ech))
	if len(v.zHaut) > 0 {
		t.Logf("  %-10s |z| du HAUT : mediane %.3f, part au-dessus de 0,9 = %.1f %%",
			"", avantMediane(v.zHaut), 100*avantPart(v.zHaut, func(z float64) bool { return z > 0.9 }))
	}
	rapide := avantFiltrer(v.ech, func(e avantEchantillon) bool { return e.vitesse >= avantSeuilVitesseMPS })
	avantPopulation(t, "TOUTE VITESSE >= 5 m/s", rapide)
	avance := avantFiltrer(rapide, func(e avantEchantillon) bool {
		return math.Cos(avantRad(avantEcart(e.capAvant, e.capVel))) > 0
	})
	avantPopulation(t, "AVANCE (scalaire > 0)", avance)
	avantTemoin(t, rapide)
}

func avantPopulation(t *testing.T, titre string, ech []avantEchantillon) {
	t.Helper()
	if len(ech) == 0 {
		t.Logf("      %-26s aucun echantillon", titre)
		return
	}
	ec := make([]float64, 0, len(ech))
	for _, e := range ech {
		ec = append(ec, avantEcart(e.capAvant, e.capVel))
	}
	t.Logf("      %-26s n = %5d, mediane %6.1f deg, p90 %6.1f, sous 15 deg %5.1f %%",
		titre, len(ec), avantMediane(ec), avantPercentile(ec, 0.90),
		100*avantPart(ec, func(d float64) bool { return d < 15 }))
}

// avantTemoin : l avant d un AUTRE echantillon, decale de la moitie de la population. Une
// mesure qui ne mesure rien rend ici la MEME mediane que la population vraie.
func avantTemoin(t *testing.T, ech []avantEchantillon) {
	t.Helper()
	if len(ech) < 4 {
		return
	}
	d := len(ech) / 2
	ec := make([]float64, 0, len(ech))
	for i, e := range ech {
		ec = append(ec, avantEcart(ech[(i+d)%len(ech)].capAvant, e.capVel))
	}
	t.Logf("      %-26s n = %5d, mediane %6.1f deg (attendu proche de 90)",
		"TEMOIN PAR PERMUTATION", len(ec), avantMediane(ec))
}

func avantAzimut(x, y float64) float64 {
	d := math.Atan2(y, x) * 180 / math.Pi
	if d < 0 {
		d += 360
	}
	return d
}

func avantEcart(a, b float64) float64 {
	d := math.Abs(a - b)
	if d > 180 {
		d = 360 - d
	}
	return d
}

func avantRad(d float64) float64 { return d * math.Pi / 180 }

func avantFiltrer(in []avantEchantillon, ok func(avantEchantillon) bool) []avantEchantillon {
	var out []avantEchantillon
	for _, e := range in {
		if ok(e) {
			out = append(out, e)
		}
	}
	return out
}

func avantMediane(v []float64) float64 { return avantPercentile(v, 0.5) }

func avantPercentile(v []float64, p float64) float64 {
	if len(v) == 0 {
		return 0
	}
	c := append([]float64(nil), v...)
	sort.Float64s(c)
	return c[int(p*float64(len(c)-1))]
}

func avantPart(v []float64, ok func(float64) bool) float64 {
	if len(v) == 0 {
		return 0
	}
	n := 0
	for _, x := range v {
		if ok(x) {
			n++
		}
	}
	return float64(n) / float64(len(v))
}
