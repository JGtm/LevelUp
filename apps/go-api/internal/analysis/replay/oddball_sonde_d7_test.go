package replay

// oddball_sonde_d7_test.go — D7 : LA SONDE DIAGNOSTIQUE. Ou passe la moitie manquante ?
//
// LE PROTOCOLE EST ECRIT ET COMMITE AVANT CE FICHIER (`.ai/V7.5/PLAN_OBJECTIFS_ETAT_VIVANT_
// 2026-08.md`, section « D7 »). Ce qui suit l'applique.
//
// # CE QUE CETTE SONDE N'EST PAS
//
// Elle ne reconstruit rien, ne rejoue aucun verdict, ne touche aucun seuil de D6 et ne publie
// rien dans l'artefact. Elle rend TROIS signatures, chacune assortie de ce qui la confirmerait
// ET de ce qui l'ecarterait — ecrits avant de mesurer, faute de quoi une sonde trouve toujours
// ce qu'elle cherche.
//
// REGIME : gardes `ATT_FILM` + `ODDBALL_FILM` + `ODDBALL_ORACLE`, UN FILM PAR PROCESSUS, lecture
// seule, AUCUNE base ouverte.
//
//	go test ./internal/analysis/replay/ -run OddballSondeDiagnostique -v

import (
	"encoding/json"
	"math"
	"os"
	"sort"
	"testing"

	"levelup/go-api/internal/filmproc"
)

const (
	// d7FenetresImages : les fenetres de fraicheur sondees, en IMAGES. Zero = le protocole de
	// D6 tel quel ; les suivantes elargissent la recherche autour de l'instant du silence.
	d7ImageUS = 100_000 // une image de rejeu vaut 100 ms
	// d7GainConfirme / d7GainEcarte : les deux bornes de la signature S1, en POINTS de
	// pourcentage gagnes entre W = 0 et W = 5 images.
	d7GainConfirme = 15.0
	d7GainEcarte   = 5.0
	// d7PartVieCourte / d7PartNaissanceInexpliquee : les bornes des signatures S2 et S3.
	d7PartVieCourte             = 0.20
	d7PartNaissanceInexpliquee  = 0.30
	d7PartNaissanceInexpliqueeK = 0.10
)

// d7Fenetres : les largeurs sondees, en images.
var d7Fenetres = []int{0, 1, 2, 5, 10}

// TestOddballSondeDiagnostique — LA SONDE. Un film par processus.
func TestOddballSondeDiagnostique(t *testing.T) {
	root := attRequireRoot(t)
	id := os.Getenv(d4FilmEnv)
	if id == "" {
		t.Skipf("mesure non demandee : %s vide", d4FilmEnv)
	}
	maxAPI, ok := d7MaxPortageAPI(t, id)
	if !ok {
		return
	}
	g := filmproc.Arm("d7-sonde", filmproc.MeasureLimitGiB, func(peak uint64) {
		t.Errorf("PLAFOND MEMOIRE DEPASSE %s : %.2f Gio", id, float64(peak)/(1<<30))
	})
	defer func() {
		g.Disarm()
		t.Logf("%s : pic memoire observe %.2f Gio", id, float64(g.Peak())/(1<<30))
	}()

	vies, socles, ok := d6ViesEtSocles(t, root, id)
	if !ok {
		return
	}
	wr, lay, ok := d6Bornes(t, root, id)
	if !ok {
		return
	}
	dir := objChunkDir(root, id)
	pont := objBridgeOf(t, root, id)
	pos, err := d6Positions(dir, wr, lay)
	if err != nil {
		t.Fatalf("%s : positions de bipede illisibles : %v", id, err)
	}
	tracks := indexBySlot(pos)
	deaths, err := ScanFilmDeaths(dir)
	if err != nil {
		t.Fatalf("%s : fil des morts illisible : %v", id, err)
	}
	t.Logf("%s : %d vie(s) libre(s), %d slot(s) nomme(s), plus long portage API %.0f s",
		id, len(vies), len(pont.SlotXUID), maxAPI)

	d7SignatureS1(t, id, vies, tracks, pont)
	d7SignatureS2(t, id, vies, socles, tracks, pont, deaths, maxAPI)
	d7SignatureS3(t, id, vies, socles, tracks, pont)
}

// d7MaxPortageAPI lit, dans l'oracle fige, le plus long portage d'un joueur sur ce film.
func d7MaxPortageAPI(t *testing.T, id string) (float64, bool) {
	t.Helper()
	path := os.Getenv(d6OracleEnv)
	if path == "" {
		t.Skipf("mesure non demandee : %s vide (oracle API fige)", d6OracleEnv)
	}
	raw, err := os.ReadFile(path) //nolint:gosec // chemin fourni par l'operateur
	if err != nil {
		t.Fatalf("oracle illisible : %v", err)
	}
	var all map[string]map[string]float64
	if err := json.Unmarshal(raw, &all); err != nil {
		t.Fatalf("oracle invalide : %v", err)
	}
	best := 0.0
	for _, s := range all[id] {
		if s > best {
			best = s
		}
	}
	if best <= 0 {
		t.Logf("NON EXPLOITABLE %s : aucun temps de portage API.", id)
		return 0, false
	}
	return best, true
}

// d7SignatureS1 — LA DERNIERE POSITION EST-ELLE PERIMEE ?
//
// On rejoue la recherche du plus proche sur des fenetres de plus en plus larges autour de
// l'instant du silence. Si le joueur est « un peu plus loin dans le temps », la part de trous
// couverts doit MONTER ; si elle ne bouge pas, il n'est nulle part et la piste tombe.
func d7SignatureS1(t *testing.T, id string, vies []flagFreeLife,
	tracks map[uint32]slotTrack, pont objBridge,
) {
	t.Helper()
	trous := d7Trous(vies)
	if len(trous) == 0 {
		t.Logf("S1 %s : aucun trou confrontable", id)
		return
	}
	parts := make([]float64, 0, len(d7Fenetres))
	for _, w := range d7Fenetres {
		sous := 0
		for _, tr := range trous {
			if d7MinSurFenetre(tracks, pont, tr, w) <= d6RayonRamassageM {
				sous++
			}
		}
		p := 100 * float64(sous) / float64(len(trous))
		parts = append(parts, p)
		t.Logf("S1 %s :   W = %2d image(s) -> %d/%d trou(s) avec un joueur sous %.1f m (%.1f %%)",
			id, w, sous, len(trous), d6RayonRamassageM, p)
	}
	gain := parts[3] - parts[0] // W = 5 contre W = 0, tel qu'ecrit au protocole
	verdict := "INDECISE"
	switch {
	case gain >= d7GainConfirme:
		verdict = "CONFIRMEE"
	case gain < d7GainEcarte:
		verdict = "ECARTEE"
	}
	t.Logf("S1 %s : gain W=0 -> W=5 : %+.1f point(s) (confirme >= %.0f, ecarte < %.0f) — %s",
		id, gain, d7GainConfirme, d7GainEcarte, verdict)
	d7Vitesses(t, id, vies)
}

// d7Vitesses publie la vitesse de l'objet sur ses trois dernieres images avant le silence : une
// derniere position ne peut etre PERIMEE que si l'objet BOUGEAIT.
func d7Vitesses(t *testing.T, id string, vies []flagFreeLife) {
	t.Helper()
	var v []float64
	for _, l := range vies {
		n := len(l.Pts)
		if n < 2 {
			continue
		}
		i := n - 4
		if i < 0 {
			i = 0
		}
		dt := float64(l.Pts[n-1].TUS-l.Pts[i].TUS) / 1e6
		if dt <= 0 {
			continue
		}
		d := math.Hypot(float64(l.Pts[n-1].X)-float64(l.Pts[i].X),
			float64(l.Pts[n-1].Y)-float64(l.Pts[i].Y))
		v = append(v, d/dt)
	}
	if len(v) == 0 {
		t.Logf("S1 %s : aucune vitesse mesurable", id)
		return
	}
	sort.Float64s(v)
	t.Logf("S1 %s : vitesse de l objet sur ses 3 dernieres images — mediane %.2f m/s, "+
		"q75 %.2f, q90 %.2f, max %.2f (%d vies)",
		id, v[len(v)/2], v[len(v)*3/4], v[len(v)*9/10], v[len(v)-1], len(v))
}

// d7SignatureS2 — DEUX PORTAGES FUSIONNES EN UN ?
func d7SignatureS2(t *testing.T, id string, vies []flagFreeLife, socles []PointObjective,
	tracks map[uint32]slotTrack, pont objBridge, deaths []Death, maxAPI float64,
) {
	t.Helper()
	depassent, portes := 0, 0
	var plusLong float64
	for _, tr := range d6Reconstruit(vies, socles, tracks, pont, deaths, nil) {
		if tr.classe != "porte" {
			continue
		}
		portes++
		d := tr.dureeS()
		if d > plusLong {
			plusLong = d
		}
		if d > maxAPI {
			depassent++
		}
	}
	courtes := 0
	for _, l := range vies {
		if l.T1US <= l.T0US {
			courtes++
		}
	}
	partCourtes := float64(courtes) / float64(len(vies))
	t.Logf("S2 %s : %d portage(s) reconstruit(s), le plus long %.0f s contre un maximum API de "+
		"%.0f s ; %d le depasse(nt). Vies libres reduites a un instant : %d/%d = %.1f %%",
		id, portes, plusLong, maxAPI, depassent, courtes, len(vies), 100*partCourtes)
	verdict := "ECARTEE"
	if depassent > 0 || partCourtes >= d7PartVieCourte {
		verdict = "CONFIRMEE"
	}
	t.Logf("S2 %s : (confirme si un portage depasse le max API, ou si les vies d un instant "+
		"pesent >= %.0f %%) — %s", id, 100*d7PartVieCourte, verdict)
}

// d7SignatureS3 — DES VIES NAISSENT-ELLES SANS ETRE APPARIEES ?
func d7SignatureS3(t *testing.T, id string, vies []flagFreeLife, socles []PointObjective,
	tracks map[uint32]slotTrack, pont objBridge,
) {
	t.Helper()
	n := map[string]int{}
	for i, l := range vies {
		x, y := l.First()
		var precX, precY float32
		aPrec := false
		if i > 0 {
			precX, precY = vies[i-1].Last()
			aPrec = true
		}
		_, dist, _ := d6PlusProche(tracks, pont, l.T0US, x, y)
		switch {
		case d6NaitAuSocle(l, socles):
			n["socle"]++
		case dist <= d6RayonRamassageM:
			n["joueur"]++
		case aPrec && math.Hypot(float64(x)-float64(precX), float64(y)-float64(precY)) <= d6SocleM:
			n["silence"]++
		default:
			n["inexpliquee"]++
		}
	}
	part := float64(n["inexpliquee"]) / float64(len(vies))
	verdict := "INDECISE"
	switch {
	case part >= d7PartNaissanceInexpliquee:
		verdict = "CONFIRMEE"
	case part <= d7PartNaissanceInexpliqueeK:
		verdict = "ECARTEE"
	}
	t.Logf("S3 %s : naissances — %d au socle, %d aux pieds d un joueur, %d au lieu du silence "+
		"precedent, %d INEXPLIQUEES sur %d (%.1f %%) — %s",
		id, n["socle"], n["joueur"], n["silence"], n["inexpliquee"], len(vies), 100*part, verdict)
}

// d7Silence est UN silence d'objet : quand il s'est tu, et OU il etait a ce moment-la.
//
// UN TYPE A LUI PLUTOT QUE DE DETOURNER `d6Trou` : la sonde n'a besoin ni de classe, ni de
// porteur, ni de fin de portage, et loger deux flottants dans des champs prevus pour autre chose
// aurait rendu la lecture fausse au premier regard.
type d7Silence struct {
	auUS uint64
	x, y float32
}

// d7Trous rend les silences : l'instant ou chaque vie se tait, et la derniere position emise.
func d7Trous(vies []flagFreeLife) []d7Silence {
	out := make([]d7Silence, 0, len(vies))
	for i := 0; i+1 < len(vies); i++ {
		if vies[i+1].T0US <= vies[i].T1US {
			continue
		}
		x, y := vies[i].Last()
		out = append(out, d7Silence{auUS: vies[i].T1US, x: x, y: y})
	}
	return out
}

// d7MinSurFenetre rend la distance minimale a un joueur nomme sur `t1 +/- w` images.
//
// LA POSITION DE L'OBJET NE BOUGE PAS DANS LA FENETRE : c'est la derniere qu'il ait emise, et
// c'est precisement l'hypothese qu'on teste. Seul l'instant ou l'on regarde les JOUEURS varie.
func d7MinSurFenetre(tracks map[uint32]slotTrack, pont objBridge, s d7Silence, w int) float64 {
	best := math.MaxFloat64
	for k := -w; k <= w; k++ {
		at := int64(s.auUS) + int64(k)*d7ImageUS
		if at < 0 {
			continue
		}
		if _, d, _ := d6PlusProche(tracks, pont, uint64(at), s.x, s.y); d < best {
			best = d
		}
	}
	return best
}
