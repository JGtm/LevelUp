package replay

// visee_elevation_test.go — INSTRUMENT DE MESURE du DEUXIEME AXE DE LA VISEE (i21 `PitchRaw`),
// item E.0.1 du plan `.ai/V7.5/replay2d/PLAN_EXPLOITATION_REGISTRE_FILM.md`.
//
// CE QU'IL MESURE, ET POURQUOI IL EXISTE. `unit-desired-aiming-vector` (ti=35 i21) porte DEUX
// scalaires : un cap R(12), lu et publie depuis longtemps (`AimHeadingDeg`), et une elevation
// R(11) qui est LUE, STOCKEE dans `BipedPosition`, et que personne n'a jamais interpretee. Sa
// convention (ou est « a plat » ? quel signe regarde vers le haut ? quelle plage ?) n'est ecrite
// nulle part : elle ne peut donc pas etre supposee, elle doit etre MESUREE. Cet instrument le
// fait en deux temps :
//
//	1. DISTRIBUTION — la forme brute de `PitchRaw` sur le film (bornes, mode, quantiles,
//	   histogramme). Une convention lineaire centree se voit dans la forme : un mode net pres du
//	   centre de la plage, un support borne, une symetrie approximative.
//	2. ORACLE — le seul controle qui ne partage AUCUNE piece avec le decodage de l'angle : a
//	   chaque kill du fil, le SIGNE de l'elevation du tueur doit s'accorder avec le SIGNE de
//	   l'ecart d'altitude vers sa victime (dz = z(victime) - z(tueur)). Si l'un tire vers le
//	   haut, l'autre est au-dessus. Seuil du plan : accord >= 80 % sur les kills a |dz| >= 1 m.
//	   Temoin obligatoire : les memes signes d'elevation apparies A D'AUTRES kills (permutation
//	   deterministe) doivent retomber a ~50 %.
//	3. ORACLE ANGULAIRE — ajoute apres la premiere passe, et c'est lui qui porte la CONVENTION.
//	   Le test de signe seul s'est revele NON DISCRIMINANT sur le film temoin (accord 100 %,
//	   mais temoin permute a 86,7 % : la population etait quasi constante, un predicteur
//	   constant aurait obtenu le meme chiffre). Le signe ne dit d'ailleurs RIEN de l'echelle,
//	   alors que la convention demandee est une FORMULE EN DEGRES. Au moment du kill, le
//	   reticule du tueur est SUR sa victime : l'angle d'elevation geometrique
//	   atan2(dz, distance horizontale) est donc l'angle vise, en degres, mesure sans toucher au
//	   champ. La pente de la regression (degres par pas de quantum) EST la convention, et la
//	   correlation dit si elle est lineaire en angle.
//
// D'OU VIENNENT LES PIECES, ET POURQUOI D'ICI :
//
//	positions + elevation  `filmdec.ScanFilmBipedPositions` (CaptureDirs) — un seul balayage.
//	fil des morts          `ScanFilmDeaths` (chunk highlight du film).
//	fil des KILLS          `analysis.ParseHighlightEvents` sur le MEME chunk : la victime est un
//	                       event `death`, le tueur un event `kill` au MEME instant. On ne prend
//	                       QUE les instants qui portent exactement un de chaque — aucun couple
//	                       n'est reconstruit, aucun orphelin n'est recolle (`killsource` le fait,
//	                       mais il ne rend que des NOMS, et il coute un second decodage complet
//	                       du film : deux raisons de ne pas l'appeler ici).
//	pont slot -> xuid      `buildOwners` — le pont de PRODUCTION du rejeu, nomme par la mort qui
//	                       termine chaque vie (cf. lives.go). C'est lui aussi qui rend le
//	                       decalage d'horloge entre le fil (horloge du match) et les positions
//	                       (horloge du film).
//	bornes de la carte     `filmdec.MapQuantCatalog` — le nom de carte est FOURNI par
//	                       l'operateur (AIM_MAP), jamais devine, et le catalogue est CONTROLE
//	                       contre le decoupage lu dans le film (`DetectI0Layout`) : sans ce
//	                       controle, un dz en metres serait un dz dans une autre unite.
//
// IL NE MODIFIE RIEN : lecture seule du film, aucune base, aucun artefact. SOUS GARDE
// D'ENVIRONNEMENT (AIM_FILM), donc saute partout ailleurs, CI comprise.
//
// UN SEUL FILM PAR PROCESSUS (regle D17 du plan : la machine paie les decodages).
//
// USAGE (depuis apps/go-api) :
//
//	CGO_ENABLED=0 AIM_FILM=<repo>/data/cache/film_chunks/000d5950 AIM_MAP=Cliffhanger \
//	  AIM_BOUNDS=<repo>/data/titles/halo_infinite/reference/map_quant_bounds.json \
//	  AIM_TSV=<repo>/.ai/V7.5/replay2d/registre_film/lotEF \
//	  go test ./internal/analysis/replay/ -run TestViseeElevation -v -timeout 60m

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

const (
	aimFilmEnv   = "AIM_FILM"   // repertoire des chunks du film
	aimMapEnv    = "AIM_MAP"    // nom AFFICHE de la carte (cle du catalogue de bornes)
	aimBoundsEnv = "AIM_BOUNDS" // chemin de map_quant_bounds.json
	aimTSVEnv    = "AIM_TSV"    // repertoire de sortie des TSV (facultatif)
)

// Les quatre seuils de l'oracle, ECRITS AVANT LA MESURE (regle D13 du plan).
const (
	// aimWindowMS : fenetre AMONT dans laquelle on retient la derniere visee du tueur. 300 ms
	// est l'enonce du plan ; a ~60 Hz de replication elle contient une vingtaine de records.
	aimWindowMS = 300
	// aimPairGapMS : ecart maximal accepte entre l'echantillon du tueur et celui de la victime.
	// Au-dela, « au meme instant » serait un abus de langage et dz melangerait deux instants.
	aimPairGapMS = 150
	// aimMinDZM : ecart vertical minimal, en metres, pour qu'un kill entre dans l'oracle. En
	// deca, le signe de dz n'est pas une information sur la visee (deux joueurs a la meme
	// altitude se visent a plat).
	aimMinDZM = 1.0
	// aimAccordSeuil : le gate du plan.
	aimAccordSeuil = 0.80
	// aimMinDXYM : distance HORIZONTALE minimale, en metres, pour que l'angle d'elevation
	// geometrique vers la victime soit defini. A bout portant, atan2 explose sur du bruit de
	// quantum ; deux metres suffisent a le stabiliser sans vider la population.
	aimMinDXYM = 2.0
)

// aimPitchSpan est le nombre de valeurs distinctes de `PitchRaw` (R(11)).
const aimPitchSpan = 1 << 11

// aimPitchCentreTheorique : le centre de la plage R(11). L'hypothese de depart — a CONFIRMER
// par la mesure, jamais a supposer — est que l'elevation est encodee comme le cap : lineaire,
// centree, au MEME quantum angulaire (360/4096 = 180/2048 = 0,0879 deg par pas).
const aimPitchCentreTheorique = aimPitchSpan / 2

// TestViseeElevation execute les deux mesures de E.0.1 sur UN film.
func TestViseeElevation(t *testing.T) {
	dir, mapName, cat := aimEntrees(t)
	entry, err := cat.Lookup(mapName)
	if err != nil {
		t.Fatalf("carte %q absente du catalogue de bornes : %v", mapName, err)
	}
	release := filmdec.LockProcessDecode()
	defer release()

	// CONTROLE AVANT MESURE : le decoupage d'i0 lu dans le film doit egaler celui que le
	// catalogue deduit des bornes. S'ils different, les bornes ne sont pas celles de cette
	// carte et tout dz en metres serait faux — on refuse de mesurer plutot que de publier un
	// chiffre dans une unite inconnue.
	lay, _, err := filmdec.DetectI0Layout(dir)
	if err != nil {
		t.Fatalf("decoupage d'i0 illisible : %v", err)
	}
	if lay.AxisW != entry.AxisWidths {
		t.Fatalf("DESACCORD DE BORNES : catalogue %v (module %s) vs film %v — le nom de carte"+
			" fourni (%q) n'est pas celui du film", entry.AxisWidths, entry.Module, lay.AxisW, mapName)
	}
	t.Logf("FILM %s · carte %q (module %s) · decoupage d'axes %v CONCORDANT",
		filepath.Base(dir), mapName, entry.Module, lay.AxisW)

	wr := entry.Range()
	scan := filmdec.DefaultScanFilmOptions()
	scan.CaptureDirs = true
	scan.WorldRange = &wr
	debut := time.Now()
	pos, err := filmdec.ScanFilmBipedPositions(dir, scan)
	cout := time.Since(debut)
	if err != nil {
		t.Fatalf("balayage des positions : %v", err)
	}
	t.Logf("COUT — ScanFilmBipedPositions : %d positions en %s", len(pos), cout.Round(time.Millisecond))

	dist := aimDistribution(t, pos)
	aimEcrisDistributionTSV(t, dir, dist)
	aimOracle(t, dir, pos)
}

// aimEntrees resout les trois entrees de l'instrument, ou declare le test saute.
func aimEntrees(t *testing.T) (string, string, *filmdec.MapQuantCatalog) {
	t.Helper()
	dir := os.Getenv(aimFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument de mesure saute", aimFilmEnv)
	}
	mapName := os.Getenv(aimMapEnv)
	if mapName == "" {
		t.Skipf("%s absent : le nom de carte est FOURNI, jamais devine — mesure sautee", aimMapEnv)
	}
	boundsPath := os.Getenv(aimBoundsEnv)
	if boundsPath == "" {
		t.Skipf("%s absent : sans bornes, un quantum n'est pas une altitude — mesure sautee", aimBoundsEnv)
	}
	cat, err := filmdec.LoadMapQuantCatalog(boundsPath)
	if err != nil {
		t.Fatalf("catalogue de bornes : %v", err)
	}
	return dir, mapName, cat
}

// aimDist porte la forme brute de `PitchRaw` sur un film.
type aimDist struct {
	// total / avecVisee : denominateurs. Une elevation n'existe que dans les records dont le
	// masque annonce i21 ET dont la marche l'a atteint (HasYaw).
	total, avecVisee int
	// hist[q] compte les occurrences de la valeur brute q.
	hist     []int
	min, max int
	// quantiles aux rangs 1, 5, 25, 50, 75, 95, 99 %.
	quant [7]int
	mode  int
}

// aimDistribution calcule et journalise la distribution brute de `PitchRaw`.
func aimDistribution(t *testing.T, pos []filmdec.BipedPosition) aimDist {
	t.Helper()
	d := aimDist{total: len(pos), hist: make([]int, aimPitchSpan), min: aimPitchSpan, max: -1}
	vals := make([]int, 0, len(pos))
	for _, p := range pos {
		if !p.HasYaw {
			continue
		}
		q := int(p.PitchRaw)
		if q < 0 || q >= aimPitchSpan {
			t.Fatalf("PitchRaw hors de R(11) : %d — la largeur du champ n'est pas celle qu'on croit", q)
		}
		d.avecVisee++
		d.hist[q]++
		vals = append(vals, q)
		d.min, d.max = minInt(d.min, q), maxInt(d.max, q)
	}
	if d.avecVisee == 0 {
		t.Fatalf("aucun record ne porte i21 sur ce film : rien a mesurer")
	}
	sort.Ints(vals)
	for i, r := range [7]float64{0.01, 0.05, 0.25, 0.50, 0.75, 0.95, 0.99} {
		d.quant[i] = vals[minInt(len(vals)-1, int(r*float64(len(vals))))]
	}
	for q, n := range d.hist {
		if n > d.hist[d.mode] {
			d.mode = q
		}
	}
	aimJournaliseDistribution(t, d)
	return d
}

// aimJournaliseDistribution ecrit la distribution au journal du test.
func aimJournaliseDistribution(t *testing.T, d aimDist) {
	t.Helper()
	t.Logf("DISTRIBUTION de PitchRaw (R(11), 0..%d) — denominateurs : %d positions, %d portent i21 (%.1f %%)",
		aimPitchSpan-1, d.total, d.avecVisee, 100*float64(d.avecVisee)/float64(maxInt(1, d.total)))
	t.Logf("  bornes observees : [%d, %d] · mode %d · centre theorique %d",
		d.min, d.max, d.mode, aimPitchCentreTheorique)
	t.Logf("  quantiles  p1=%d  p5=%d  p25=%d  p50=%d  p75=%d  p95=%d  p99=%d",
		d.quant[0], d.quant[1], d.quant[2], d.quant[3], d.quant[4], d.quant[5], d.quant[6])
	// Symetrie : sous une convention centree, autant de valeurs de part et d'autre du centre.
	var sous, sur int
	for q, n := range d.hist {
		switch {
		case q < aimPitchCentreTheorique:
			sous += n
		case q > aimPitchCentreTheorique:
			sur += n
		}
	}
	t.Logf("  autour du centre theorique %d : %d sous (%.1f %%) · %d sur (%.1f %%) · %d dessus exactement",
		aimPitchCentreTheorique, sous, 100*float64(sous)/float64(d.avecVisee),
		sur, 100*float64(sur)/float64(d.avecVisee), d.hist[aimPitchCentreTheorique])
	t.Logf("  histogramme en 32 classes de 64 valeurs (classe : borne basse · compte · %% ) :")
	for c := 0; c < 32; c++ {
		n := 0
		for q := c * 64; q < (c+1)*64; q++ {
			n += d.hist[q]
		}
		if n == 0 {
			continue
		}
		t.Logf("    %5d  %8d  %5.2f %%  %s", c*64, n, 100*float64(n)/float64(d.avecVisee),
			strings.Repeat("#", minInt(60, 1+60*n/maxInt(1, d.hist[d.mode]*64))))
	}
}

// aimEcrisDistributionTSV depose l'histogramme complet, valeur par valeur.
func aimEcrisDistributionTSV(t *testing.T, dir string, d aimDist) {
	t.Helper()
	out := os.Getenv(aimTSVEnv)
	if out == "" {
		return
	}
	var b strings.Builder
	b.WriteString("pitch_raw\tcompte\tpart\n")
	for q, n := range d.hist {
		if n == 0 {
			continue
		}
		fmt.Fprintf(&b, "%d\t%d\t%.6f\n", q, n, float64(n)/float64(d.avecVisee))
	}
	path := filepath.Join(out, filepath.Base(dir)+"_E01_pitch_hist.tsv")
	if err := os.WriteFile(path, []byte(b.String()), 0o600); err != nil {
		t.Fatalf("ecriture TSV : %v", err)
	}
	t.Logf("  TSV : %s", path)
}

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

// aimQuantumCandidats : les deux conventions candidates, en degres par pas.
//
//	180/2048 = le quantum du CAP (360/4096) : plage +/- 90 deg sur TOUT le champ R(11).
//	360/2048 = deux fois ce quantum : le champ couvre +/- 180 deg et le tangage, borne a
//	           +/- 90 deg par le jeu, n'en occupe que la MOITIE (1024 pas utiles).
var aimQuantumCandidats = []struct {
	nom  string
	degs float64
}{
	{"180/2048 (quantum du cap)", 180.0 / 2048},
	{"360/2048 (deux fois le quantum du cap)", 360.0 / 2048},
}

// aimAjuste estime le quantum angulaire EN CORRIGEANT le biais de hauteur de visee.
//
// POURQUOI UN AJUSTEMENT A DEUX PARAMETRES, ET PAS LA SIMPLE PENTE. `elevDeg` est calcule entre
// deux ORIGINES de bipede, alors que le tir part de l'OEIL du tueur et arrive sur le CORPS de la
// victime : il manque une hauteur h, constante, que la geometrie transforme en une erreur
// d'angle INVERSEMENT proportionnelle a la distance. C'est elle qui ecrase le rapport
// angle/pas aux courtes portees et qui faisait varier la pente d'un facteur trois selon la
// tranche d'amplitude. Le modele l'absorbe :
//
//	dz = dxy * tan(c * pas) - h
//
// c (degres par pas) et h (metres) sortent ensemble. h est un CONTROLE PHYSIQUE : s'il tombe
// autour d'un demi-metre, le modele decrit bien la geometrie du tir ; s'il tombe a dix metres,
// c'est le modele qui est faux, et c alors ne vaut rien.
func aimAjuste(pts []aimPoint) (c, h, r2 float64) {
	meilleur := math.Inf(1)
	for pas := 0; pas <= 3000; pas++ {
		cand := 0.02 + float64(pas)*0.0001
		hh, sse := aimResidu(pts, cand)
		if sse < meilleur {
			meilleur, c, h = sse, cand, hh
		}
	}
	var moy float64
	for _, p := range pts {
		moy += p.dz
	}
	moy /= float64(len(pts))
	var sst float64
	for _, p := range pts {
		sst += (p.dz - moy) * (p.dz - moy)
	}
	if sst > 0 {
		r2 = 1 - meilleur/sst
	}
	return c, h, r2
}

// aimResidu rend le decalage de hauteur optimal pour un quantum donne, et la somme des carres.
func aimResidu(pts []aimPoint, c float64) (h, sse float64) {
	res := make([]float64, len(pts))
	for i, p := range pts {
		res[i] = p.dz - p.dxy*math.Tan(c*float64(p.pas)*math.Pi/180)
	}
	for _, v := range res {
		h += v
	}
	h /= float64(len(res))
	for _, v := range res {
		sse += (v - h) * (v - h)
	}
	return h, sse
}

// aimRegression rend la pente, l'ordonnee a l'origine et la correlation de elevDeg sur pas.
func aimRegression(pts []aimPoint) (pente, ord, r float64) {
	n := float64(len(pts))
	var sx, sy, sxx, syy, sxy float64
	for _, p := range pts {
		x, y := float64(p.pas), p.elevDeg
		sx, sy, sxx, syy, sxy = sx+x, sy+y, sxx+x*x, syy+y*y, sxy+x*y
	}
	varX, varY := n*sxx-sx*sx, n*syy-sy*sy
	cov := n*sxy - sx*sy
	if varX == 0 || varY == 0 {
		return 0, 0, 0
	}
	pente = cov / varX
	ord = (sy - pente*sx) / n
	r = cov / math.Sqrt(varX*varY)
	return pente, ord, r
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
