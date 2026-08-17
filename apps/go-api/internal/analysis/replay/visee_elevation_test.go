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
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

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
