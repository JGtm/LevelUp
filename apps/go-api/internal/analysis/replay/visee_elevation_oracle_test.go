package replay

// visee_elevation_oracle_test.go — L'ORACLE DU KILL de l'item E.0.1. L'instrument et la
// distribution sont dans `visee_elevation_test.go` ; ce fichier porte la seule piece qui ne
// partage RIEN avec le decodage de l'angle : la geometrie entre un tueur et sa victime a
// l'instant du kill.
//
// Il vit a part parce que les deux moities repondent a deux questions distinctes — « quelle
// forme a le champ » et « le champ dit-il la verite » — et parce que reunies elles poussaient
// le fichier au-dela du seuil de 500 lignes du depot.

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/filmdec"
)

// aimCouple est un kill du fil : un tueur, une victime, un instant (horloge du fil).
type aimCouple struct {
	tueur, victime uint64
	tMS            int64
}

// aimCouples reconstitue les kills du chunk highlight du film.
//
// AUCUN COUPLE N'EST FABRIQUE : on ne retient qu'un instant portant EXACTEMENT un event `kill`
// et un event `death`, avec deux identites distinctes. Les instants ambigus (deux morts a la
// meme milliseconde, un kill orphelin) sont COMPTES et ecartes — un couple invente serait un
// point d'oracle faux, et un oracle faux vaut moins que pas d'oracle.
func aimCouples(t *testing.T, dir string) ([]aimCouple, int, int) {
	t.Helper()
	n := filmdec.CountFilmChunks(dir)
	raw, err := os.ReadFile(filepath.Join(dir, fmt.Sprintf("chunk_%02d.bin", n)))
	if err != nil {
		t.Fatalf("chunk highlight (%d) : %v", n, err)
	}
	evs, err := analysis.ParseHighlightEvents(raw, 0)
	if err != nil {
		t.Fatalf("chunk highlight (%d) : %v", n, err)
	}
	kills := map[int][]uint64{}
	deaths := map[int][]uint64{}
	for _, e := range evs {
		switch e.EventType {
		case analysis.EventTypeKill:
			kills[e.TimeMS] = append(kills[e.TimeMS], e.XUID)
		case analysis.EventTypeDeath:
			deaths[e.TimeMS] = append(deaths[e.TimeMS], e.XUID)
		}
	}
	var out []aimCouple
	ambigus := 0
	for ms, ks := range kills {
		ds := deaths[ms]
		if len(ks) != 1 || len(ds) != 1 || ks[0] == ds[0] {
			ambigus++
			continue
		}
		out = append(out, aimCouple{tueur: ks[0], victime: ds[0], tMS: int64(ms)})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].tMS < out[j].tMS })
	return out, len(kills), ambigus
}

// aimPoint est UN kill situe : l'elevation brute du tueur et la geometrie vers sa victime.
type aimPoint struct {
	pas     int     // PitchRaw - centre theorique
	dz, dxy float64 // metres
	elevDeg float64 // atan2(dz, dxy) en degres : l'angle REELLEMENT vise
	// viseeDeg est ce que rend l'ACCESSEUR de production (`AimPitchDeg`). Le comparer a
	// elevDeg est le controle de bout en bout : si les deux divergent, c'est la formule de
	// l'accesseur qui est fausse, et le mesurer ici evite qu'elle derive en silence.
	viseeDeg float64
}

// aimBilan agrege le resultat de l'oracle.
type aimBilan struct {
	// couples : kills reconstruits ; sansTueur / sansVictime : ecartes faute d'echantillon.
	couples, sansTueur, sansVictime, ecartTrop int
	// sousSeuilDZ / sousSeuilDXY : ecartes par les deux seuils, chacun pour son oracle.
	sousSeuilDZ, sousSeuilDXY int
	// retenus : kills entres dans l'oracle de SIGNE ; accords : signes concordants.
	retenus, accords int
	// temoin : accords apres permutation deterministe des elevations.
	temoin int
	// dzPositifs : nombre de dz > 0 parmi les retenus. Sans lui, on ne peut pas dire si un
	// accord eleve vient de la mesure ou d'une population constante.
	dzPositifs int
	// ecarts : |dt| entre l'echantillon du tueur et celui de la victime, en ms.
	ecarts []int64
	// pts : la population de l'oracle ANGULAIRE (seuil dxy, pas de seuil dz).
	pts []aimPoint
}

// aimOracle confronte le signe de l'elevation du tueur au signe de dz vers sa victime.
func aimOracle(t *testing.T, dir string, pos []filmdec.BipedPosition) {
	t.Helper()
	couples, nKills, ambigus := aimCouples(t, dir)
	deaths, err := ScanFilmDeaths(dir)
	if err != nil {
		t.Fatalf("fil des morts : %v", err)
	}
	debut := time.Now()
	idx, err := ScanFilmPlayerIndices(dir, rosterFromDeaths(deaths))
	if err != nil {
		t.Fatalf("index de joueur : %v", err)
	}
	t.Logf("COUT — ScanFilmPlayerIndices : %s", time.Since(debut).Round(time.Millisecond))
	table, collisions := injectiveOrEmpty(idx)
	tracks := indexBySlot(pos)
	own := buildOwners(tracks, deaths, table, nil)
	t.Logf("ORACLE — fil : %d instants de kill, %d couples retenus, %d instants ambigus ecartes",
		nKills, len(couples), ambigus)
	t.Logf("  pont slot->xuid : %d slots nommes sur %d vies · decalage d'horloge %d ms"+
		" (%d fins de vie appariees) · collisions d'index %d",
		len(own.SlotXUID), own.LivesTotal, own.DeathOffsetMS, own.DeathOffsetMatches, collisions)
	if len(own.SlotXUID) == 0 {
		t.Fatalf("pont vide : aucun kill ne peut etre situe sur la carte")
	}
	parXUID := map[uint64][]uint32{}
	for slot, x := range own.SlotXUID {
		parXUID[x] = append(parXUID[x], slot)
	}
	b := aimEvalueCouples(couples, tracks, parXUID, own.DeathOffsetMS)
	aimJournaliseOracle(t, b)
	aimEcrisOracleTSV(t, dir, b)
}

// aimEcrisOracleTSV depose la population de l'oracle, kill par kill : c'est la piece qui permet
// de refaire la regression ailleurs sans re-decoder le film.
func aimEcrisOracleTSV(t *testing.T, dir string, b aimBilan) {
	t.Helper()
	out := os.Getenv(aimTSVEnv)
	if out == "" {
		return
	}
	var sb strings.Builder
	sb.WriteString("pas\tpitch_raw\tdz_m\tdxy_m\televation_geo_deg\n")
	for _, p := range b.pts {
		fmt.Fprintf(&sb, "%d\t%d\t%.3f\t%.3f\t%.4f\n",
			p.pas, p.pas+aimPitchCentreTheorique, p.dz, p.dxy, p.elevDeg)
	}
	path := filepath.Join(out, filepath.Base(dir)+"_E01_oracle.tsv")
	if err := os.WriteFile(path, []byte(sb.String()), 0o600); err != nil {
		t.Fatalf("ecriture TSV : %v", err)
	}
	t.Logf("  TSV oracle : %s", path)
}

// aimEvalueCouples parcourt les kills et remplit le bilan. Les elevations retenues sont
// permutees d'un cran pour fabriquer le temoin : meme population, meme distribution, appariement
// FAUX. Un oracle qui tient par construction donnerait le meme chiffre aux deux.
func aimEvalueCouples(couples []aimCouple, tracks map[uint32]slotTrack,
	parXUID map[uint64][]uint32, offMS int64) aimBilan {
	b := aimBilan{couples: len(couples)}
	var signes []aimPoint
	for _, c := range couples {
		tFilm := c.tMS + offMS
		tueur, okT := aimDernierEchantillon(tracks, parXUID[c.tueur], tFilm, true)
		if !okT {
			b.sansTueur++
			continue
		}
		victime, okV := aimEchantillonProche(tracks, parXUID[c.victime], int64(tueur.TimestampUS)/1000)
		if !okV {
			b.sansVictime++
			continue
		}
		ecart := absI64(int64(tueur.TimestampUS)/1000 - int64(victime.TimestampUS)/1000)
		b.ecarts = append(b.ecarts, ecart)
		if ecart > aimPairGapMS {
			b.ecartTrop++
			continue
		}
		p := aimPointDe(tueur, victime)
		switch {
		case p.dxy < aimMinDXYM:
			b.sousSeuilDXY++
		default:
			b.pts = append(b.pts, p)
		}
		if math.Abs(p.dz) < aimMinDZM {
			b.sousSeuilDZ++
			continue
		}
		signes = append(signes, p)
	}
	b.retenus = len(signes)
	for i, p := range signes {
		if p.dz > 0 {
			b.dzPositifs++
		}
		if aimSigneAccorde(p.pas, p.dz) {
			b.accords++
		}
		if aimSigneAccorde(signes[(i+1)%len(signes)].pas, p.dz) {
			b.temoin++
		}
	}
	return b
}

// aimPointDe compose un point d'oracle a partir des deux echantillons contemporains.
func aimPointDe(tueur, victime filmdec.BipedPosition) aimPoint {
	dz := float64(victime.Z - tueur.Z)
	dx := float64(victime.X - tueur.X)
	dy := float64(victime.Y - tueur.Y)
	dxy := math.Hypot(dx, dy)
	visee, _ := tueur.AimPitchDeg()
	return aimPoint{
		pas: int(tueur.PitchRaw) - aimPitchCentreTheorique,
		dz:  dz, dxy: dxy,
		elevDeg:  math.Atan2(dz, dxy) * 180 / math.Pi,
		viseeDeg: float64(visee),
	}
}

// aimSigneAccorde dit si l'elevation brute et l'ecart d'altitude ont le meme signe, sous
// l'hypothese « au-dessus du centre = vers le haut ». C'est CETTE hypothese que l'oracle teste :
// un accord voisin de 0 % la refuterait en designant la convention inverse.
func aimSigneAccorde(pas int, dz float64) bool { return (pas > 0) == (dz > 0) }

// aimDernierEchantillon rend le DERNIER echantillon d'un joueur dans la fenetre amont [t-300, t],
// tous ses slots confondus (un joueur change de slot a chaque vie ; seul le slot vivant emet).
func aimDernierEchantillon(tracks map[uint32]slotTrack, slots []uint32, tMS int64,
	exigeVisee bool) (filmdec.BipedPosition, bool) {
	var best filmdec.BipedPosition
	found := false
	for _, s := range slots {
		for _, p := range tracks[s].pts {
			ms := int64(p.TimestampUS) / 1000
			if ms > tMS || ms < tMS-aimWindowMS {
				continue
			}
			if exigeVisee && !p.HasYaw {
				continue
			}
			if !found || p.TimestampUS > best.TimestampUS {
				best, found = p, true
			}
		}
	}
	return best, found
}

// aimEchantillonProche rend l'echantillon d'un joueur le plus proche d'un instant, dans la
// fenetre de l'oracle. La victime n'a pas a porter de visee : seule son altitude compte.
func aimEchantillonProche(tracks map[uint32]slotTrack, slots []uint32,
	tMS int64) (filmdec.BipedPosition, bool) {
	var best filmdec.BipedPosition
	bd := int64(aimWindowMS + 1)
	found := false
	for _, s := range slots {
		for _, p := range tracks[s].pts {
			d := absI64(int64(p.TimestampUS)/1000 - tMS)
			if d < bd {
				bd, best, found = d, p, true
			}
		}
	}
	return best, found
}

// aimJournaliseOracle publie le bilan, ses denominateurs et son verdict.
func aimJournaliseOracle(t *testing.T, b aimBilan) {
	t.Helper()
	t.Logf("  attrition : %d couples · %d sans echantillon de tueur · %d sans echantillon de"+
		" victime · %d ecart tueur/victime > %d ms · %d sous |dz| >= %.1f m (oracle de signe)"+
		" · %d sous dxy >= %.1f m (oracle angulaire)",
		b.couples, b.sansTueur, b.sansVictime, b.ecartTrop, aimPairGapMS, b.sousSeuilDZ,
		aimMinDZM, b.sousSeuilDXY, aimMinDXYM)
	if len(b.ecarts) > 0 {
		sort.Slice(b.ecarts, func(i, j int) bool { return b.ecarts[i] < b.ecarts[j] })
		t.Logf("  ecart tueur/victime : mediane %d ms · p95 %d ms · max %d ms",
			b.ecarts[len(b.ecarts)/2], b.ecarts[minInt(len(b.ecarts)-1, 95*len(b.ecarts)/100)],
			b.ecarts[len(b.ecarts)-1])
	}
	aimJournaliseSigne(t, b)
	aimJournaliseAngle(t, b)
}

// aimJournaliseSigne publie l'oracle de SIGNE tel que le plan l'enonce, AVEC son plancher :
// la part de la modalite majoritaire de dz. Un accord qui ne depasse pas ce plancher est celui
// d'un predicteur constant, et ne prouve rien.
func aimJournaliseSigne(t *testing.T, b aimBilan) {
	t.Helper()
	if b.retenus == 0 {
		t.Logf("  SIGNE : aucun kill retenu — oracle de signe non mesurable sur ce film.")
		return
	}
	accord := float64(b.accords) / float64(b.retenus)
	temoin := float64(b.temoin) / float64(b.retenus)
	part := float64(b.dzPositifs) / float64(b.retenus)
	plancher := math.Max(part, 1-part)
	t.Logf("  SIGNE : accord %d / %d = %.1f %% (seuil %.0f %%) · temoin permute %.1f %%"+
		" · plancher du predicteur constant %.1f %% (dz > 0 dans %d cas sur %d)",
		b.accords, b.retenus, 100*accord, 100*aimAccordSeuil, 100*temoin, 100*plancher,
		b.dzPositifs, b.retenus)
	switch {
	case accord >= aimAccordSeuil && accord > plancher:
		t.Logf("    -> TENU ET DISCRIMINANT : au-dessus du centre = viser vers le HAUT.")
	case accord >= aimAccordSeuil:
		t.Logf("    -> tenu MAIS NON DISCRIMINANT (le plancher constant fait aussi bien).")
	case 1-accord >= aimAccordSeuil:
		t.Logf("    -> CONVENTION INVERSE (accord de l'hypothese inverse : %.1f %%).", 100*(1-accord))
	default:
		t.Logf("    -> NON TENU : ni l'hypothese ni son inverse n'atteignent %.0f %%.", 100*aimAccordSeuil)
	}
}

// aimJournaliseAngle publie l'oracle ANGULAIRE : la pente degres-par-pas, sa correlation, et sa
// stabilite par tranche d'amplitude (une convention lineaire en ANGLE garde la meme pente
// partout ; une convention lineaire en sinus verrait la pente decroitre aux grandes valeurs).
func aimJournaliseAngle(t *testing.T, b aimBilan) {
	t.Helper()
	if len(b.pts) < 5 {
		t.Logf("  ANGLE : %d points — population trop maigre pour une pente.", len(b.pts))
		return
	}
	pente, ord, r := aimRegression(b.pts)
	t.Logf("  ANGLE : %d kills · pente %.5f deg/pas · ordonnee %.2f deg · correlation r = %.3f",
		len(b.pts), pente, ord, r)
	t.Logf("    reference : le quantum du CAP vaut 360/4096 = %.5f deg/pas ; une plage +/- 90 deg"+
		" sur R(11) donnerait 180/2048 = %.5f deg/pas", 360.0/4096, 180.0/2048)
	// Rapport median angle/pas par tranche d'amplitude : c'est le controle de linearite. Il est
	// BIAISE aux courtes portees (cf. aimAjuste) — il est publie pour cette raison meme.
	tranches := [][2]int{{5, 20}, {20, 60}, {60, 500}}
	for _, tr := range tranches {
		var ratios []float64
		for _, p := range b.pts {
			if a := absInt(p.pas); a >= tr[0] && a < tr[1] {
				ratios = append(ratios, p.elevDeg/float64(p.pas))
			}
		}
		if len(ratios) == 0 {
			continue
		}
		sort.Float64s(ratios)
		t.Logf("    |pas| dans [%d,%d[ : %3d kills · rapport median %.5f deg/pas (biais de hauteur non corrige)",
			tr[0], tr[1], len(ratios), ratios[len(ratios)/2])
	}
	aimJournaliseAjustement(t, b.pts)
}

// aimJournaliseAjustement publie le quantum ajuste, le decalage de hauteur qui l'accompagne, et
// la comparaison aux deux conventions candidates.
func aimJournaliseAjustement(t *testing.T, pts []aimPoint) {
	t.Helper()
	c, h, r2 := aimAjuste(pts)
	t.Logf("  AJUSTEMENT dz = dxy·tan(c·pas) − h sur %d kills :", len(pts))
	t.Logf("    c = %.6f deg/pas · h = %.3f m (hauteur oeil du tueur − point vise) · R2 = %.3f",
		c, h, r2)
	for _, cand := range aimQuantumCandidats {
		_, sse := aimResidu(pts, cand.degs)
		_, sseOpt := aimResidu(pts, c)
		t.Logf("    candidat %-40s c = %.6f · ecart au meilleur : SSE ×%.2f",
			cand.nom, cand.degs, sse/math.Max(sseOpt, 1e-12))
	}
	// Portee LONGUE seulement : le biais de hauteur y devient negligeable, donc le rapport
	// median y est un second estimateur du quantum, independant de l'ajustement.
	var ratios []float64
	for _, p := range pts {
		if p.dxy >= 8 && absInt(p.pas) >= 20 {
			ratios = append(ratios, p.elevDeg/float64(p.pas))
		}
	}
	if len(ratios) > 0 {
		sort.Float64s(ratios)
		t.Logf("    second estimateur (dxy >= 8 m et |pas| >= 20) : %d kills · rapport median %.6f deg/pas",
			len(ratios), ratios[len(ratios)/2])
	}
	aimJournaliseAccesseur(t, pts)
}

// aimJournaliseAccesseur confronte l'ACCESSEUR de production a la geometrie, sur les kills a
// longue portee (ceux ou le biais de hauteur ne masque plus rien).
func aimJournaliseAccesseur(t *testing.T, pts []aimPoint) {
	t.Helper()
	var ecarts []float64
	for _, p := range pts {
		if p.dxy >= 8 {
			ecarts = append(ecarts, math.Abs(p.viseeDeg-p.elevDeg))
		}
	}
	if len(ecarts) == 0 {
		return
	}
	sort.Float64s(ecarts)
	t.Logf("    ACCESSEUR AimPitchDeg vs geometrie (dxy >= 8 m, %d kills) : ecart median %.2f deg"+
		" · p90 %.2f deg", len(ecarts), ecarts[len(ecarts)/2],
		ecarts[minInt(len(ecarts)-1, 90*len(ecarts)/100)])
}
