package replay

// ctf_research_test.go — INSTRUMENT DE RECHERCHE (v7.5 voie B, décision #2 du master plan).
//
// CE QU'IL MESURE. Le constat d'origine — « 564 tirs perdus en CTF contre 44 en Slayer » —
// est un compte ABSOLU sur deux films de tailles différentes. Cet instrument le remplace par
// des taux, puis VENTILE le rejet « slot introuvable » en sous-causes départageables :
//
//	vie non nommée      un tir tombe pendant une vie que le fil des morts n'a pas nommée
//	                    (le pont ne peut pas connaître son slot) -> chantier du PONT
//	trou de position    le joueur est au pont, mais aucun échantillon de position de ses
//	                    slots ne tombe à moins de 120 ms du tir -> chantier de la RÉPLICATION
//	joueur hors pont    aucun slot du film n'est rattaché à cet index de joueur
//
// CE QU'IL NE FAIT PAS. Il ne modifie RIEN : ni le décodeur, ni l'assemblage. Il rejoue le
// même enchaînement que BuildFromFilm et compte à côté. Il est SOUS GARDE D'ENVIRONNEMENT
// (CTF_RESEARCH_FILMS) et se saute partout ailleurs, CI comprise.
//
// USAGE (depuis apps/go-api) :
//
//	CGO_ENABLED=0 FILM_CACHE_ROOT=<mainrepo>/data/cache CTF_RESEARCH_OUT=<dir> \
//	  CTF_RESEARCH_FILMS="64e8adfa:Catalyst,9aeca4b3:Catalyst" \
//	  go test ./internal/analysis/replay/ -run CTFLostShots -timeout 60m -v
//
// Le film est en LECTURE SEULE, et la sortie n'est jamais écrite dans data/.

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/domain/title"
)

const (
	ctfFilmsEnv = "CTF_RESEARCH_FILMS" // "short:Carte,short:Carte"
	ctfCacheEnv = "FILM_CACHE_ROOT"    // racine contenant film_chunks/
	ctfOutEnv   = "CTF_RESEARCH_OUT"   // répertoire des rapports
)

// ctfGapBuckets borne les écarts entre un tir rejeté et l'échantillon de position le plus
// proche de son tireur. Le premier seuil est la tolérance du rattachement (shotPosToleranceUS) :
// en deçà, le tir serait rattaché. Les suivants séparent « la réplication est clairsemée »
// (centaines de ms) de « ce joueur n'a aucune trace ici » (secondes).
var ctfGapBuckets = []uint64{120_000, 250_000, 500_000, 1_000_000, 2_000_000,
	5_000_000, 15_000_000, math.MaxUint64}

// shotDiag est le verdict porté sur UN événement de tir.
type shotDiag struct {
	tUS      uint64
	pi       int
	weapon   uint64
	class    string // "rattache" | "ambigu" | "sans-slot"
	cause    string // sous-cause du rejet
	gapUS    uint64 // écart au plus proche échantillon d'un slot du joueur
	inUnnamd bool   // une vie NON NOMMÉE couvre cet instant
	slots    int    // nombre de slots du pont appartenant à ce joueur
}

// filmReport agrège les verdicts d'un film.
type filmReport struct {
	short, mapName            string
	firstUS, lastUS           uint64
	diags                     []shotDiag
	lives, named              int
	slots                     int
	posSamples                int
	medianStepUS, p90StepUS   uint64
	unnamedLives              []lifeSpan
	attached, ambig, noSlotN  int
	interSampleOver120msRatio float64
}

func TestCTFLostShotsResearch(t *testing.T) {
	spec := os.Getenv(ctfFilmsEnv)
	if spec == "" {
		t.Skipf("recherche CTF non demandée : %s vide (format \"short:Carte,short:Carte\")", ctfFilmsEnv)
	}
	cache := os.Getenv(ctfCacheEnv)
	if cache == "" {
		t.Fatalf("%s est requis (racine contenant film_chunks/)", ctfCacheEnv)
	}
	outDir := os.Getenv(ctfOutEnv)
	if outDir == "" {
		t.Fatalf("%s est requis (répertoire des rapports — jamais data/)", ctfOutEnv)
	}
	cat := loadCTFQuantCatalog(t)
	for _, item := range strings.Split(spec, ",") {
		short, mapName, ok := strings.Cut(strings.TrimSpace(item), ":")
		if !ok {
			t.Fatalf("entrée mal formée %q (attendu short:Carte)", item)
		}
		t.Run(short, func(t *testing.T) {
			rep := analyzeCTFFilm(t, cat, filepath.Join(cache, "film_chunks", short), short, mapName)
			writeCTFReport(t, outDir, rep)
		})
	}
}

// ctfReadingOnlyOwners reproduit le pont TEL QU'IL ÉTAIT AVANT les fermetures : la seule lecture
// du fil des morts. Il existe parce que `buildOwners` ferme désormais le pont en production
// (closures.go) — mesurer le « avant » exige donc de reconstruire explicitement l'état antérieur,
// et non de dupliquer la fermeture, qui vit dans le code de production et nulle part ailleurs.
func ctfReadingOnlyOwners(tracks map[uint32]slotTrack, deaths []Death,
	idx PlayerIndexTable) (map[uint32]int, []lifeSpan, int64) {
	lives := buildLifeSpans(tracks)
	off, _ := bestDeathOffset(lives, deaths)
	nameLivesByDeaths(lives, deaths, off)
	owners, _, _ := ownersFromLives(lives, idx.ByXUID)
	return owners, lives, off
}

func loadCTFQuantCatalog(t *testing.T) *filmdec.MapQuantCatalog {
	t.Helper()
	root, err := title.FindRepoRoot()
	if err != nil {
		t.Fatalf("racine du dépôt : %v", err)
	}
	path := title.NewPathResolver(root).MapQuantBoundsPath(title.DefaultSlug)
	cat, err := filmdec.LoadMapQuantCatalog(path)
	if err != nil {
		t.Fatalf("catalogue de bornes %s : %v", path, err)
	}
	return cat
}

// analyzeCTFFilm rejoue l'enchaînement de BuildFromFilm et compte à côté de lui.
func analyzeCTFFilm(t *testing.T, cat *filmdec.MapQuantCatalog, dir, short, mapName string) filmReport {
	t.Helper()
	entry, err := cat.Lookup(mapName)
	if err != nil {
		t.Fatalf("bornes de %s : %v", mapName, err)
	}
	world := entry.Range()
	scan := filmdec.DefaultScanFilmOptions()
	scan.WorldRange, scan.CaptureDirs = &world, true
	pos, err := filmdec.ScanFilmBipedPositions(dir, scan)
	if err != nil {
		t.Fatalf("positions %s : %v", short, err)
	}
	fire, err := filmdec.ScanFilmFireEvents(dir)
	if err != nil {
		t.Fatalf("tirs %s : %v", short, err)
	}
	deaths, err := ScanFilmDeaths(dir)
	if err != nil {
		t.Fatalf("fil des morts %s : %v", short, err)
	}
	idx, err := ScanFilmPlayerIndices(dir, rosterFromDeaths(deaths))
	if err != nil {
		t.Fatalf("index de joueur %s : %v", short, err)
	}
	table, _ := injectiveOrEmpty(idx)

	sorted := append([]filmdec.BipedPosition(nil), pos...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].TimestampUS < sorted[j].TimestampUS })
	tracks := indexBySlot(sorted)
	// Pont de LECTURE SEULE : cet instrument mesure l'état antérieur aux fermetures, qui est le
	// point de comparaison de tout le verdict.
	owners, lives, _ := ctfReadingOnlyOwners(tracks, deaths, table)
	named := 0
	for _, l := range lives {
		if l.xuid != 0 {
			named++
		}
	}

	rep := filmReport{
		short: short, mapName: mapName,
		lives: len(lives), named: named, slots: len(owners), posSamples: len(sorted),
	}
	if len(sorted) > 0 {
		rep.firstUS, rep.lastUS = sorted[0].TimestampUS, sorted[len(sorted)-1].TimestampUS
	}
	for _, l := range lives {
		if l.xuid == 0 {
			rep.unnamedLives = append(rep.unnamedLives, l)
		}
	}
	rep.medianStepUS, rep.p90StepUS, rep.interSampleOver120msRatio = ctfSampleSteps(tracks)
	rep.diags = ctfClassify(tracks, owners, rep.unnamedLives, fire)
	for _, d := range rep.diags {
		switch d.class {
		case "rattache":
			rep.attached++
		case "ambigu":
			rep.ambig++
		default:
			rep.noSlotN++
		}
	}
	return rep
}

// ctfClassify porte un verdict sur chaque tir : rattaché, ambigu, ou rejeté avec sa cause.
func ctfClassify(tracks map[uint32]slotTrack, owner map[uint32]int,
	unnamed []lifeSpan, fire []filmdec.FireEvent) []shotDiag {
	bySlots := map[int][]uint32{}
	for slot, pi := range owner {
		bySlots[pi] = append(bySlots[pi], slot)
	}
	out := make([]shotDiag, 0, len(fire))
	for _, e := range fire {
		d := shotDiag{tUS: e.TimestampUS, pi: e.FilmIndex, weapon: e.WeaponID,
			gapUS: math.MaxUint64, slots: len(bySlots[e.FilmIndex])}
		n := 0
		for _, slot := range bySlots[e.FilmIndex] {
			_, gap := tracks[slot].at(e.TimestampUS)
			if gap < d.gapUS {
				d.gapUS = gap
			}
			if gap <= shotPosToleranceUS {
				n++
			}
		}
		d.inUnnamd = coveredByUnnamedLife(unnamed, e.TimestampUS)
		switch {
		case n == 1:
			d.class, d.cause = "rattache", "-"
		case n > 1:
			d.class, d.cause = "ambigu", "vies qui se recouvrent"
		default:
			d.class = "sans-slot"
			d.cause = ctfNoSlotCause(d)
		}
		out = append(out, d)
	}
	return out
}

// ctfNoSlotCause départage les sous-causes du rejet « slot introuvable ».
//
// L'ORDRE EST UNE PRÉSÉANCE, PAS UN HASARD : « aucun slot » est un fait sur le pont, il prime ;
// « vie non nommée » l'explique quand une vie sans identité couvre l'instant ; le trou de
// position ne reste qu'en dernier, quand le joueur EST au pont et qu'une vie nommée l'entoure.
func ctfNoSlotCause(d shotDiag) string {
	switch {
	case d.slots == 0:
		return "joueur hors pont"
	case d.inUnnamd:
		return "vie non nommee"
	default:
		return "trou de position"
	}
}

// coveredByUnnamedLife dit si une vie sans identité couvre l'instant.
func coveredByUnnamedLife(unnamed []lifeSpan, tUS uint64) bool {
	t := int64(tUS)
	for _, l := range unnamed {
		if t >= l.from && t <= l.to {
			return true
		}
	}
	return false
}

// ctfSampleSteps mesure la DENSITÉ de réplication des positions : médiane et p90 de l'écart
// entre deux échantillons consécutifs d'un même slot, et la part de ces écarts qui dépasse la
// tolérance du rattachement. C'est le témoin direct de l'hypothèse « réplication clairsemée ».
func ctfSampleSteps(tracks map[uint32]slotTrack) (uint64, uint64, float64) {
	var steps []uint64
	over := 0
	for _, tr := range tracks {
		for i := 1; i < len(tr.pts); i++ {
			d := tr.pts[i].TimestampUS - tr.pts[i-1].TimestampUS
			if d > lifeGapUS { // changement de vie : ce n'est pas un pas de réplication
				continue
			}
			steps = append(steps, d)
			if d > shotPosToleranceUS {
				over++
			}
		}
	}
	if len(steps) == 0 {
		return 0, 0, 0
	}
	sort.Slice(steps, func(i, j int) bool { return steps[i] < steps[j] })
	return steps[len(steps)/2], steps[len(steps)*9/10], float64(over) / float64(len(steps))
}

// writeCTFReport écrit le rapport lisible et la table des rejets, et journalise le résumé.
func writeCTFReport(t *testing.T, outDir string, r filmReport) {
	t.Helper()
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatalf("répertoire de sortie : %v", err)
	}
	var b strings.Builder
	total := len(r.diags)
	fmt.Fprintf(&b, "film\t%s\ncarte\t%s\n", r.short, r.mapName)
	fmt.Fprintf(&b, "duree_s\t%.1f\n", float64(r.lastUS-r.firstUS)/1e6)
	fmt.Fprintf(&b, "tirs_disponibles\t%d\nrattaches\t%d\tambigus\t%d\tsans_slot\t%d\n",
		total, r.attached, r.ambig, r.noSlotN)
	fmt.Fprintf(&b, "taux_rattachement\t%.4f\ntaux_sans_slot\t%.4f\n",
		ratio(r.attached, total), ratio(r.noSlotN, total))
	fmt.Fprintf(&b, "vies\t%d\tnommees\t%d\tnon_nommees\t%d\tslots_pont\t%d\n",
		r.lives, r.named, r.lives-r.named, r.slots)
	fmt.Fprintf(&b, "echantillons_position\t%d\npas_median_ms\t%.1f\npas_p90_ms\t%.1f\npart_pas_sup_120ms\t%.4f\n",
		r.posSamples, float64(r.medianStepUS)/1000, float64(r.p90StepUS)/1000, r.interSampleOver120msRatio)
	writeCTFCauses(&b, r)
	writeCTFGaps(&b, r)
	writeCTFTemporal(&b, r)
	writeCTFPerPlayer(&b, r)
	writeCTFWeapons(&b, r)
	path := filepath.Join(outDir, r.short+"_rapport.txt")
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		t.Fatalf("écriture rapport : %v", err)
	}
	t.Logf("\n%s", b.String())
}

func writeCTFCauses(b *strings.Builder, r filmReport) {
	counts := map[string]int{}
	for _, d := range r.diags {
		if d.class == "sans-slot" {
			counts[d.cause]++
		}
	}
	fmt.Fprintf(b, "\n# ventilation des rejets sans-slot\n")
	for _, k := range ctfSortedKeys(counts) {
		fmt.Fprintf(b, "cause\t%s\t%d\t%.4f\n", k, counts[k], ratio(counts[k], len(r.diags)))
	}
}

func writeCTFGaps(b *strings.Builder, r filmReport) {
	hist := make([]int, len(ctfGapBuckets))
	for _, d := range r.diags {
		if d.class != "sans-slot" || d.slots == 0 {
			continue
		}
		for i, lim := range ctfGapBuckets {
			if d.gapUS <= lim {
				hist[i]++
				break
			}
		}
	}
	fmt.Fprintf(b, "\n# ecart au plus proche echantillon du tireur (rejets sans-slot, joueur au pont)\n")
	for i, lim := range ctfGapBuckets {
		fmt.Fprintf(b, "ecart_max_ms\t%s\t%d\n", limLabel(lim), hist[i])
	}
}

func writeCTFTemporal(b *strings.Builder, r filmReport) {
	const buckets = 10
	span := r.lastUS - r.firstUS
	if span == 0 {
		return
	}
	avail, lost := make([]int, buckets), make([]int, buckets)
	for _, d := range r.diags {
		i := int((d.tUS - r.firstUS) * buckets / span)
		if i >= buckets {
			i = buckets - 1
		}
		avail[i]++
		if d.class == "sans-slot" {
			lost[i]++
		}
	}
	fmt.Fprintf(b, "\n# repartition temporelle (deciles du film)\n")
	for i := 0; i < buckets; i++ {
		fmt.Fprintf(b, "decile\t%d\tdisponibles\t%d\tsans_slot\t%d\ttaux\t%.4f\n",
			i+1, avail[i], lost[i], ratio(lost[i], avail[i]))
	}
}

func writeCTFPerPlayer(b *strings.Builder, r filmReport) {
	avail, lost, slots := map[int]int{}, map[int]int{}, map[int]int{}
	for _, d := range r.diags {
		avail[d.pi]++
		slots[d.pi] = d.slots
		if d.class == "sans-slot" {
			lost[d.pi]++
		}
	}
	fmt.Fprintf(b, "\n# par index de joueur du film\n")
	keys := make([]int, 0, len(avail))
	for k := range avail {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	for _, k := range keys {
		fmt.Fprintf(b, "joueur\t%d\tslots\t%d\tdisponibles\t%d\tsans_slot\t%d\ttaux\t%.4f\n",
			k, slots[k], avail[k], lost[k], ratio(lost[k], avail[k]))
	}
}

func writeCTFWeapons(b *strings.Builder, r filmReport) {
	avail, lost := map[uint64]int{}, map[uint64]int{}
	for _, d := range r.diags {
		avail[d.weapon]++
		if d.class == "sans-slot" {
			lost[d.weapon]++
		}
	}
	type row struct {
		id         uint64
		a, l       int
		lossRatioV float64
	}
	rows := make([]row, 0, len(avail))
	for id, a := range avail {
		rows = append(rows, row{id, a, lost[id], ratio(lost[id], a)})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].a > rows[j].a })
	fmt.Fprintf(b, "\n# par arme (identifiant brut du film)\n")
	for i, rw := range rows {
		if i >= 15 {
			break
		}
		fmt.Fprintf(b, "arme\t0x%016X\tdisponibles\t%d\tsans_slot\t%d\ttaux\t%.4f\n",
			rw.id, rw.a, rw.l, rw.lossRatioV)
	}
}

func limLabel(lim uint64) string {
	if lim == math.MaxUint64 {
		return "infini"
	}
	return fmt.Sprintf("%.0f", float64(lim)/1000)
}

func ratio(n, d int) float64 {
	if d == 0 {
		return 0
	}
	return float64(n) / float64(d)
}

func ctfSortedKeys(m map[string]int) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
