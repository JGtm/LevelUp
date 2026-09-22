//go:build research

package grammar

// avant_chassis_54_research_test.go — LOT 5.4 : L AVANT DU CHASSIS, MESURE.
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
// DEUX PORTES, ET UNE SEULE LIT LE CACHE :
//
//	go test -tags=research ./internal/games/halo_infinite/film/internal/grammar/ \
//	  -run TestAvantChassisMiniBobines -v -count=1                      (aucun film du cache)
//
//	AVANT_FILM=<abs>/data/cache/film_chunks/4f77afc1 AVANT_CARTE="Flood Gulch" \
//	  go test -tags=research ./internal/games/halo_infinite/film/internal/grammar/ \
//	  -run TestAvantChassisFilm -v -count=1 -timeout 60m                (UN film a la fois)
//
// LECTURE SEULE : aucun fichier ecrit, AUCUNE base DuckDB ouverte — l instrument ne lit que les
// chunks du film et le catalogue de bornes versionne (`map_quant_bounds.json`).

import (
	"math"
	"os"
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
	slot     uint32
	tUS      uint64
	mode     uint8
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
	lents      []avantEchantillon // vitesse SOUS le seuil : le regime « a l arret »
}

// ---------------------------------------------------------------------------------------------
// LA PORTE QUI LIT UN FILM DU CACHE — un seul a la fois.
// ---------------------------------------------------------------------------------------------

func TestAvantChassisFilm(t *testing.T) {
	dir, carte := os.Getenv("AVANT_FILM"), os.Getenv("AVANT_CARTE")
	if dir == "" || carte == "" {
		t.Skip("instrument de mesure : AVANT_FILM et AVANT_CARTE requis")
	}
	fc, plage := avantContexteDe(t, dir, carte)
	if fc == nil {
		t.Fatalf("contexte illisible")
	}
	if restore, err := InstallFilmFormatMPP(fc); err == nil {
		defer restore()
	}
	v := avantBalayer(t, fc, plage)
	if v == nil {
		t.Fatalf("nuage ti=40 illisible")
	}
	nom := filepath.Base(dir)
	avantJournaliser(t, nom, v)
	avantParMode(t, v)
	avantRegimeLent(t, v)
}

// ---------------------------------------------------------------------------------------------
// LA PORTE SANS CACHE — les mini-bobines.
// ---------------------------------------------------------------------------------------------

func TestAvantChassisMiniBobines(t *testing.T) {
	total := avantVentilation{}
	for _, court := range closureMiniFilms() {
		fc, plage := avantContexte(t, court)
		if fc == nil {
			continue
		}
		v := avantBalayer(t, fc, plage)
		if v == nil {
			t.Logf("  %-10s nuage ti=40 vide", court)
			continue
		}
		avantJournaliser(t, court, v)
		avantCumuler(&total, v)
	}
	t.Log("================ CUMUL DES MINI-BOBINES ================")
	avantJournaliser(t, "CUMUL", &total)
}

// avantBalayer rend la ventilation du nuage `ti=40` d un contexte deja ouvert.
func avantBalayer(t *testing.T, fc *FilmContext, wr *profile.Vec3Range) *avantVentilation {
	t.Helper()
	kf := ScanWorldObjectKeyframes(fc.Film(), VehicleTypeIndex)
	if len(kf.Band) == 0 {
		return nil
	}
	opt := DefaultScanFilmOptions()
	opt.RequireTag1 = false
	opt.CaptureDirs = true
	opt.DynPrecOrientation = true
	if l := fc.ImposedLayout(); l != nil {
		opt.Layout = l
	}
	opt.WorldRange = wr
	pos, err := ScanBipedPositionsForBand(fc, NewSlotBand(kf.Band), opt)
	if err != nil {
		t.Logf("   nuage illisible : %v", err)
		return nil
	}
	if len(pos) == 0 {
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
		e := avantEchantillon{
			slot: p.Slot, tUS: p.TimestampUS, mode: p.FwdMode,
			capAvant: avantAzimut(float64(fwd[0]), float64(fwd[1])),
			capVel:   avantAzimut(float64(vel[0]), float64(vel[1])),
			vitesse:  math.Hypot(float64(vel[0]), float64(vel[1])),
		}
		if e.vitesse >= avantSeuilVitesseMPS {
			v.ech = append(v.ech, e)
		} else {
			v.lents = append(v.lents, e)
		}
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
	dst.lents = append(dst.lents, src.lents...)
}

func avantJournaliser(t *testing.T, nom string, v *avantVentilation) {
	t.Helper()
	t.Logf("  %-10s %6d position(s) ti=40 — i2 lu %d, mode0 %d, mode1 %d, mode2 %d, roulis %d, a plat %d",
		nom, v.positions, v.avecI2, v.mode0, v.mode1, v.mode2, v.avecRoulis, v.aPlat)
	t.Logf("  %-10s avant reconstructible %d, velocite %d, couples rapides %d, couples lents %d",
		"", v.avecAvant, v.avecVel, len(v.ech), len(v.lents))
	if len(v.zHaut) > 0 {
		t.Logf("  %-10s |z| du HAUT : mediane %.3f, part au-dessus de 0,9 = %.1f %%",
			"", avantMediane(v.zHaut), 100*avantPart(v.zHaut, func(z float64) bool { return z > 0.9 }))
	}
	avantPopulation(t, "TOUTE VITESSE >= 5 m/s", v.ech)
	avance := avantFiltrer(v.ech, func(e avantEchantillon) bool {
		return math.Cos(avantRad(avantEcart(e.capAvant, e.capVel))) > 0
	})
	avantPopulation(t, "AVANCE (scalaire > 0)", avance)
	avantTemoin(t, v.ech)
	for _, mult := range []float64{2, 3} {
		vite := avantFiltrer(v.ech, func(e avantEchantillon) bool {
			return e.vitesse >= mult*avantSeuilVitesseMPS
		})
		avantPopulation(t, avantTitre(mult), vite)
	}
}

// avantParMode : le tableau PAR MODE. Les deux chemins reels ont des largeurs differentes
// (19/8 contre 30/30) ; si la reconstruction est juste, les deux doivent tomber au meme endroit.
func avantParMode(t *testing.T, v *avantVentilation) {
	t.Helper()
	t.Log("  ---- PAR MODE ----")
	for _, m := range []uint8{0, 1} {
		ech := avantFiltrer(v.ech, func(e avantEchantillon) bool { return e.mode == m })
		if len(ech) == 0 {
			t.Logf("      mode %d : aucun echantillon rapide", m)
			continue
		}
		t.Logf("      mode %d (dir %d bits, roulis %d bits)", m, FwdUpDirBits(m), FwdUpRollBits(m))
		avantPopulation(t, "  TOUTE VITESSE >= 5 m/s", ech)
		avance := avantFiltrer(ech, func(e avantEchantillon) bool {
			return math.Cos(avantRad(avantEcart(e.capAvant, e.capVel))) > 0
		})
		avantPopulation(t, "  AVANCE (scalaire > 0)", avance)
		avantTemoin(t, ech)
	}
}

// avantRegimeLent : LE COMPORTEMENT A L ARRET ET EN MARCHE ARRIERE.
//
// A l arret, la velocite n est plus un oracle (direction d un vecteur quasi nul) : ce qui se
// mesure est la STABILITE du cap reconstruit d un echantillon au suivant, sur le MEME slot. Un
// cap lu dans le film doit y rester pose ; un cap tire d une velocite de bruit sauterait.
// En marche arriere, l ecart a la velocite doit se loger pres de 180 deg, pas se disperser.
func avantRegimeLent(t *testing.T, v *avantVentilation) {
	t.Helper()
	t.Log("  ---- REGIME LENT ET MARCHE ARRIERE ----")
	if n := len(v.ech); n > 0 {
		arriere := avantFiltrer(v.ech, func(e avantEchantillon) bool {
			return avantEcart(e.capAvant, e.capVel) > 135
		})
		t.Logf("      MARCHE ARRIERE (ecart > 135 deg) : %d / %d = %.1f %% des echantillons rapides",
			len(arriere), n, 100*float64(len(arriere))/float64(n))
		if len(arriere) > 0 {
			ec := make([]float64, 0, len(arriere))
			for _, e := range arriere {
				ec = append(ec, avantEcart(e.capAvant, e.capVel))
			}
			t.Logf("      ... et leur ecart se loge a mediane %.1f deg (180 = nez a l oppose du mouvement)",
				avantMediane(ec))
		}
	}
	t.Logf("      stabilite du cap entre echantillons consecutifs du MEME slot, |delta| median :")
	t.Logf("        a l arret (< 5 m/s, n = %d) : %.2f deg", len(v.lents), avantStabilite(v.lents))
	t.Logf("        en mouvement (>= 5 m/s, n = %d) : %.2f deg", len(v.ech), avantStabilite(v.ech))
}

// avantStabilite rend la mediane de |delta cap| entre echantillons consecutifs d un meme slot.
func avantStabilite(ech []avantEchantillon) float64 {
	par := map[uint32][]avantEchantillon{}
	for _, e := range ech {
		par[e.slot] = append(par[e.slot], e)
	}
	var d []float64
	for _, s := range par {
		sort.Slice(s, func(i, j int) bool { return s[i].tUS < s[j].tUS })
		for i := 1; i < len(s); i++ {
			d = append(d, avantEcart(s[i].capAvant, s[i-1].capAvant))
		}
	}
	return avantMediane(d)
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
	t.Logf("      %-26s n = %6d, mediane %6.1f deg, p90 %6.1f, sous 15 deg %5.1f %%",
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
	t.Logf("      %-26s n = %6d, mediane %6.1f deg (attendu proche de 90)",
		"TEMOIN PAR PERMUTATION", len(ec), avantMediane(ec))
}

// ---------------------------------------------------------------------------------------------
// OUVERTURE DES CONTEXTES
// ---------------------------------------------------------------------------------------------

func avantContexte(t *testing.T, court string) (*FilmContext, *profile.Vec3Range) {
	t.Helper()
	dir := filepath.Join("..", "..", "replay", "testdata", "minifilm_"+court)
	return avantContexteDe(t, dir, e191bCarteDeBobine[court])
}

// avantContexteDe ouvre le contexte sous l entree de catalogue de la carte, comme la cuisson.
func avantContexteDe(t *testing.T, dir, carte string) (*FilmContext, *profile.Vec3Range) {
	t.Helper()
	film, err := source.LoadDir(dir, nil)
	if err != nil {
		t.Logf("  LoadDir %s : %v", dir, err)
		return nil, nil
	}
	cat, err := profile.LoadMapQuantCatalog(e191bCatalogue())
	if err != nil {
		t.Logf("  catalogue : %v", err)
		return nil, nil
	}
	entree, err := cat.Lookup(carte)
	if err != nil {
		t.Logf("  carte %q hors catalogue : %v", carte, err)
		return nil, nil
	}
	fc := NewFilmContextForMap(film, &entree, nil)
	bal := fc.ProfilDeBalayage()
	bal.PoserLargeursObjetDuMondeDepuisDecoupage(entree.Layout())
	fc.PoserProfilDeBalayage(bal)
	plage := entree.Range()
	return fc, &plage
}

// ---------------------------------------------------------------------------------------------
// PETITE STATISTIQUE
// ---------------------------------------------------------------------------------------------

func avantTitre(mult float64) string {
	switch mult {
	case 2:
		return "VITESSE >= 10 m/s"
	default:
		return "VITESSE >= 15 m/s"
	}
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
