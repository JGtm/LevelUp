package filmdec

// zone_census_report_test.go — L'ENTREE et les SORTIES du recensement du lot C (phase 0,
// items C.0.1 et C.0.3). Le balayage vit dans zone_census_scan_test.go ; ici on charge le
// film, on joue les quatre mesures dans l'ordre du plan, on ecrit les TSV et on journalise.
//
// L'ORACLE DES EVENEMENTS D'OBJECTIF vient d'`objectiveevents.NamedEvents` sur la source
// disque canonique (`filmcache.Open`) — et NON d'une implementation locale de `FilmSource`,
// que le garde-rail `filmcache_guard_test.go` interdit. Cet import n'existe que dans un
// fichier de test : aucun code de production d'`analysis/` ne prend de dependance vers
// `games/halo_infinite/`.

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/objectiveevents"
	"levelup/go-api/internal/games/halo_infinite/film/filmcache"
)

// zcArchInfo porte la grammaire d'un archetype telle que le registre du FILM la declare.
type zcArchInfo struct {
	present    bool
	components []string
	levels     []uint32
	ported     []bool
}

// zcLoadGrammar lit le registre du film (chunk_00) et rend la grammaire des archetypes
// cibles. Le statut de portage est demande au DISPATCH REEL (`consumeByName`), jamais a une
// liste ecrite a la main : une liste diverge, le dispatch fait foi.
func zcLoadGrammar(t *testing.T, dir string) (map[int]zcArchInfo, *Registry) {
	t.Helper()
	raw, err := ReadFilmChunk(dir, 0)
	if err != nil {
		t.Fatalf("lecture de chunk_00 (registre) : %v", err)
	}
	reg, err := ParseRegistryChunk(raw)
	if err != nil {
		t.Fatalf("analyse du registre : %v", err)
	}
	out := map[int]zcArchInfo{}
	for _, ti := range zcTargetTIs {
		arch, ok := reg.Archetype(ti)
		if !ok {
			out[ti] = zcArchInfo{}
			continue
		}
		info := zcArchInfo{present: true, components: arch.Components}
		for i := range arch.Components {
			lv := arch.Level(i)
			info.levels = append(info.levels, lv)
			info.ported = append(info.ported, ti11ComponentIsPorted(arch.Components[i], uint32(ti), lv))
		}
		out[ti] = info
	}
	return out, reg
}

// zcObjTypeEnv porte la FAMILLE D'OBJECTIF du film : `zone` (Strongholds), `flag` (CTF) ou
// `none`. Elle est FOURNIE, jamais devinee.
//
// POURQUOI ELLE NE PEUT PAS ETRE DEVINEE. `namedStatSlots` (named.go:77-105) n'a de table que
// pour `zone` et `flag`, et le sens d'un emplacement DEPEND DU MODE : la table `flag`
// appliquee a un film KOTH rend des chiffres d'apparence solide et faux (mesure : 267
// « evenements de drapeau » dont 199 `flag_returns` sur le film KOTH 606d9844, qui n'a jamais
// vu un drapeau). Choisir la famille « celle qui rend le plus d'evenements » est donc un piege
// mesure, pas une commodite. KOTH et Oddball n'ont AUCUNE table : `none`, et le dire.
const zcObjTypeEnv = "ZONE_OBJTYPE"

// zcOracle porte les evenements d'objectif nommes du statborg et la famille employee.
type zcOracle struct {
	family string
	// times ne porte QUE les evenements d'OBJECTIF (captures, securisations, prises...) :
	// c'est ce que le gate 0 nomme « les captures ». `kills` et `assists`, que la meme table
	// porte, en sont EXCLUS — non par commodite mais parce qu'ils saturent la frise (117
	// frags + 58 assistances sur le film Strongholds 7344d24f : la fenetre +/- 3 s couvrait
	// 76 % du match, ce qui rend toute comparaison de densite vide de sens).
	times []int
	// counts recense TOUTES les statistiques rendues, y compris celles ecartees des fenetres,
	// pour que le denominateur reel soit lisible.
	counts  map[string]int
	nCombat int
}

// zcCombatStats sont les statistiques de COMBAT que la table nommee porte aussi, et qui ne
// sont pas des evenements d'objectif.
var zcCombatStats = map[string]bool{
	objectiveevents.StatKills:   true,
	objectiveevents.StatAssists: true,
}

// zcLoadOracle charge les evenements nommes du film pour la famille declaree. `none` (ou une
// famille sans table) rend un oracle VIDE : la densite est alors declaree non mesurable, ce
// qui est un resultat et non une panne.
func zcLoadOracle(t *testing.T, dir string) zcOracle {
	t.Helper()
	fam := os.Getenv(zcObjTypeEnv)
	o := zcOracle{family: fam, counts: map[string]int{}}
	if fam == "" {
		t.Fatalf("%s absent : declarer la famille d'objectif du film (zone | flag | none)", zcObjTypeEnv)
	}
	if fam != objectiveevents.ObjectiveTypeZone && fam != objectiveevents.ObjectiveTypeFlag {
		return o // `none`, KOTH, Oddball, Slayer : aucune table nommee
	}
	src := zcOpenSource(t, dir)
	for _, e := range objectiveevents.NamedEvents(src, fam) {
		o.counts[e.Stat]++
		if zcCombatStats[e.Stat] {
			o.nCombat++
			continue
		}
		o.times = append(o.times, e.TimeMS)
	}
	return o
}

// zcOpenSource ouvre la source disque CANONIQUE du film (`filmcache`). Aucune implementation
// locale de `FilmSource` : le garde-rail `filmcache_guard_test.go` l'interdit, et il a raison.
func zcOpenSource(t *testing.T, dir string) *filmcache.Source {
	t.Helper()
	root := filepath.Dir(filepath.Dir(dir))
	short := filepath.Base(dir)
	src, ok, err := filmcache.Open(root, short)
	if err != nil {
		t.Fatalf("ouverture du manifeste de film (%s / %s) : %v", root, short, err)
	}
	if !ok {
		t.Fatalf("manifeste absent pour %s sous %s : l'horloge du film est indisponible", short, root)
	}
	return src
}

// zcLoadClock rend l'horloge du film : start_ms par chunk, lu dans le manifeste.
func zcLoadClock(t *testing.T, dir string) zcClock {
	t.Helper()
	clk := zcClock{startMS: map[int]int{}}
	for _, m := range zcOpenSource(t, dir).Chunks() {
		clk.startMS[m.Index] = m.StartMS
	}
	return clk
}

// TestZoneCensusLotC est LE recensement de la phase 0 du lot C. Un film par processus.
func TestZoneCensusLotC(t *testing.T) {
	dir := zcDir(t)
	out := zcOutDir(t)
	short := filepath.Base(dir)
	release := LockProcessDecode() // les bascules de grammaire sont des globaux de paquet
	defer release()

	gram, reg := zcLoadGrammar(t, dir)
	c := zcKeyframeCensus(dir)
	t.Logf("FILM %s — %d chunks · %d tables d'image-cle · %d records d'image-cle · %d archetypes distincts",
		short, c.chunks, c.keyframes, c.totalRecords(), len(c.recordsTI))

	bands := zcBuildBands(c)
	zcReportKeyframes(t, c, bands, gram, out, short)
	zcReportKeyframeWalk(t, c, reg, out, short)

	oracle := zcLoadOracle(t, dir)
	t.Logf("ORACLE — famille declaree %q · %d evenements d'OBJECTIF retenus pour les fenetres"+
		" (+ %d evenements de combat ECARTES)", oracle.family, len(oracle.times), oracle.nCombat)
	for _, k := range zcSortedStrings(oracle.counts) {
		mark := ""
		if zcCombatStats[k] {
			mark = "   (ecarte des fenetres)"
		}
		t.Logf("    %-24s %d%s", k, oracle.counts[k], mark)
	}

	grammarLen := map[int]int{}
	for ti, g := range gram {
		grammarLen[ti] = len(g.components)
	}
	win := newZCWindows(oracle.times, zcWindowMS)
	res := zcScanDelta(c, bands, zcLoadClock(t, dir), win, grammarLen)
	zcReportDelta(t, res, bands, gram, out, short)
	zcReportDensity(t, res, gram, oracle, out, short)
}

// zcReportKeyframes journalise et ecrit le recensement d'image-cle (C.0.1 parties 1 et 4).
func zcReportKeyframes(t *testing.T, c zcCensus, b zcBands, gram map[int]zcArchInfo, out, short string) {
	t.Helper()
	var sb strings.Builder
	sb.WriteString("ti\tarchetype_present\tcomposants\trecords_image_cle\tslots_distincts\tslots_bande\tslots\n")
	t.Logf("IMAGE-CLE — records et slots par archetype (denominateur %d records) :", c.totalRecords())
	for _, ti := range zcTargetTIs {
		mark := ""
		if !gram[ti].present {
			mark = " (archetype ABSENT du registre du film)"
		}
		zcKeyframeRow(t, &sb, c, b, gram, ti, mark)
	}
	t.Logf("  les 3 premiers archetypes HORS perimetre, par volume d'image-cle :")
	for _, ti := range c.topOtherTIs(3) {
		zcKeyframeRow(t, &sb, c, b, gram, ti, " (hors perimetre)")
	}
	t.Logf("  BANDES — %d slots reels · ambigus ecartes %d · temoins : inconnu %d, occupe %d,"+
		" vide %d (plus grand slot vu en image-cle : %d)",
		b.union, b.ambigus, b.nInconnu, b.nOccupe, b.nVide, zcMaxSlotSeen(c))
	zcWriteFile(t, filepath.Join(out, short+"_kf_slots.tsv"), sb.String())
}

// zcKeyframeRow journalise et ecrit UNE ligne du recensement d'image-cle.
func zcKeyframeRow(t *testing.T, sb *strings.Builder, c zcCensus, b zcBands,
	gram map[int]zcArchInfo, ti int, mark string,
) {
	t.Helper()
	slots := zcSortedSlots(c.slotsTI[ti])
	t.Logf("    ti=%-3d %8d records · %3d slots distincts%s · slots %v",
		ti, c.recordsTI[ti], len(slots), mark, zcTrim(slots, 40))
	sb.WriteString(fmt.Sprintf("%d\t%t\t%d\t%d\t%d\t%d\t%s\n",
		ti, gram[ti].present, len(gram[ti].components), c.recordsTI[ti], len(slots),
		len(b.perTI[ti]), zcJoinInts(slots)))
}

// zcReportKeyframeWalk joue le walker DETERMINISTE de la table d'image-cle (C.0.1 partie 2).
// Il est attendu qu'il s'arrete tot : le depot a deja mesure que le corps d'un record
// d'image-cle n'est PAS celui d'un record NEW (keyframe_record_walk.go:259-273). Ce qu'on
// recense ici, ce sont les records qu'il atteint AVANT de lacher, et la cause d'arret.
func zcReportKeyframeWalk(t *testing.T, c zcCensus, reg *Registry, out, short string) {
	t.Helper()
	perTI := map[int]int{}
	stops := map[string]int{}
	total, tables := 0, 0
	for ch := 1; ch <= c.chunks; ch++ {
		data, err := ReadFilmChunk(c.dir, ch)
		if err != nil {
			continue
		}
		for _, pk := range WalkPackets(data) {
			if pk.Type != PacketTypeKeyframe {
				continue
			}
			tables++
			recs, stop := WalkKeyframeRecords(pk.Payload(data), reg)
			stops[stop.String()]++
			total += len(recs)
			for _, r := range recs {
				perTI[r.TI]++
			}
		}
	}
	t.Logf("IMAGE-CLE (walker deterministe) — %d tables · %d records parses avant arret · causes %v",
		tables, total, stops)
	var sb strings.Builder
	sb.WriteString("ti\trecords_parses_walker\n")
	for _, ti := range zcSortedKeysInt(perTI) {
		sb.WriteString(fmt.Sprintf("%d\t%d\n", ti, perTI[ti]))
		if zcIsTarget(ti) {
			t.Logf("    ti=%-3d %6d records parses", ti, perTI[ti])
		}
	}
	sb.WriteString(fmt.Sprintf("# tables=%d records_total=%d causes=%v\n", tables, total, stops))
	zcWriteFile(t, filepath.Join(out, short+"_kf_walk.tsv"), sb.String())
}

// zcReportDelta journalise et ecrit le recensement des ANNONCES au masque (C.0.1 partie 3
// et C.0.3). C'est la mesure centrale du lot.
func zcReportDelta(t *testing.T, res zcScanResult, b zcBands, gram map[int]zcArchInfo, out, short string) {
	t.Helper()
	t.Logf("DELTA — %d paquets balayes (%d dans une fenetre d'evenement · %d sans horloge)",
		res.packets, res.packetsInWin, res.packetsNoClock)
	for _, cl := range []int{zcClassVide, zcClassInconnu, zcClassOccupe} {
		s := res.byClass[cl]
		t.Logf("  TEMOIN %-9s %8d records · %3d slots peuples · %6.1f records/slot ·"+
			" plancher de bruit %.1f annonces/index", zcClassName(cl), s.records,
			len(s.slots), zcRecordsPerSlot(s), zcNoiseFloor(s))
	}
	var sb strings.Builder
	sb.WriteString("ti\ti\tcomposant\tniveau\tstatut\tannonces\tpct_records_ti\tplancher_bruit\t" +
		"exces_sur_plancher\ttemoin_vide\ttemoin_inconnu\ttemoin_occupe\n")
	for _, ti := range zcTargetTIs {
		s := res.byClass[ti]
		if s == nil {
			continue
		}
		t.Logf("  ti=%-3d %8d records · %3d/%3d slots peuples · %.1f records/slot · plancher %.1f"+
			" · hors grammaire %.2f %% · masques %s",
			ti, s.records, len(s.slots), len(b.perTI[ti]), zcRecordsPerSlot(s), zcNoiseFloor(s),
			100*zcRate(s.outOfGrammar, s.records), ti11Histogram(s.maskCount))
		if s.records == 0 {
			continue
		}
		zcWriteTIRows(&sb, t, ti, s, res, gram[ti])
	}
	zcWriteFile(t, filepath.Join(out, short+"_delta_masques.tsv"), sb.String())
}

// zcWriteTIRows ecrit une ligne par composant annonce d'un archetype. Le JOURNAL ne retient
// que ce qui se detache du plancher de bruit (facteur >= zcExcessMin) : le reste est ecrit au
// TSV mais n'a rien a dire.
func zcWriteTIRows(sb *strings.Builder, t *testing.T, ti int, s *zcDeltaStats, res zcScanResult, g zcArchInfo) {
	t.Helper()
	floor := zcNoiseFloor(s)
	for i := 0; i < worldObjectMaxComponent; i++ {
		if s.byIndex[i] == 0 {
			continue
		}
		ex := zcExcess(s.byIndex[i], floor)
		sb.WriteString(fmt.Sprintf("%d\t%d\t%s\t%s\t%s\t%d\t%.2f\t%.1f\t%.1f\t%d\t%d\t%d\n",
			ti, i, zcName(g, i), zcLevel(g, i), zcStatus(g, i), s.byIndex[i],
			100*zcRate(s.byIndex[i], s.records), floor, ex,
			res.byClass[zcClassVide].byIndex[i], res.byClass[zcClassInconnu].byIndex[i],
			res.byClass[zcClassOccupe].byIndex[i]))
		if ex >= zcExcessMin {
			t.Logf("      i%-2d %-10s %8d  %5.1f %%  exces %6.1fx  %s", i, zcStatus(g, i),
				s.byIndex[i], 100*zcRate(s.byIndex[i], s.records), ex, zcName(g, i))
		}
	}
}

// zcExcessMin est le facteur au-dessus du plancher de bruit a partir duquel une annonce est
// jugee DISTINGUABLE du bruit. Ecrit avant la mesure. 3x est le meme ordre de grandeur que le
// seuil de densite du gate 0 ; en-dessous, un ecart s'explique par la variance du tirage.
const zcExcessMin = 3.0

// zcClassName nomme une bande de controle.
func zcClassName(cl int) string {
	switch cl {
	case zcClassInconnu:
		return "INCONNU"
	case zcClassOccupe:
		return "OCCUPE"
	case zcClassVide:
		return "VIDE"
	}
	return fmt.Sprintf("ti=%d", cl)
}

// zcReportDensity ecrit la densite des annonces DANS et HORS des fenetres d'evenement — la
// grandeur du gate 0. Les denominateurs (secondes couvertes) sont publies avec.
func zcReportDensity(t *testing.T, res zcScanResult, gram map[int]zcArchInfo, o zcOracle, out, short string) {
	t.Helper()
	t.Logf("DENSITE — horloge de match couverte [%d ; %d] ms · fenetre du gate (+/- %d ms autour"+
		" de %d evenements d'objectif) %.1f s · hors fenetre %.1f s · diagnostic serre (+/- %d ms)"+
		" %.1f s / %.1f s",
		res.tMinMS, res.tMaxMS, zcWindowMS, len(o.times), res.secInWin, res.secOutWin,
		zcTightMS, res.secInTight, res.secOutTight)
	var sb strings.Builder
	sb.WriteString("ti\ti\tcomposant\tstatut\tannonces_fenetre\tannonces_hors\tsecondes_fenetre\t" +
		"secondes_hors\tdensite_fenetre\tdensite_hors\trapport\trapport_serre_1s\n")
	if res.secInWin <= 0 || res.secOutWin <= 0 {
		sb.WriteString("# denominateur nul : aucune fenetre exploitable sur ce film\n")
		zcWriteFile(t, filepath.Join(out, short+"_densite.tsv"), sb.String())
		t.Logf("  AUCUNE FENETRE EXPLOITABLE (oracle vide ou horloge absente) : la densite n'est"+
			" pas mesurable sur ce film. Famille d'oracle : %q, %d evenements.", o.family, len(o.times))
		return
	}
	for _, ti := range zcTargetTIs {
		s := res.byClass[ti]
		if s == nil || s.records == 0 {
			continue
		}
		floor := zcNoiseFloor(s)
		for i := 0; i < worldObjectMaxComponent; i++ {
			if s.inWin[i]+s.outWin[i] == 0 {
				continue
			}
			dIn := float64(s.inWin[i]) / res.secInWin
			dOut := float64(s.outWin[i]) / res.secOutWin
			tight := zcRatio(float64(s.inTight[i])/res.secInTight, float64(s.outTight[i])/res.secOutTight)
			sb.WriteString(fmt.Sprintf("%d\t%d\t%s\t%s\t%d\t%d\t%.1f\t%.1f\t%.3f\t%.3f\t%s\t%s\n",
				ti, i, zcName(gram[ti], i), zcStatus(gram[ti], i), s.inWin[i], s.outWin[i],
				res.secInWin, res.secOutWin, dIn, dOut, zcRatio(dIn, dOut), tight))
			// Le gate 0 porte sur les composants NON PORTES qui se detachent du bruit ET
			// depassent 100 annonces : le journal ne retient que ceux-la.
			if zcStatus(gram[ti], i) != "porte" && s.byIndex[i] >= zcGateMinAnnonces &&
				zcExcess(s.byIndex[i], floor) >= zcExcessMin {
				t.Logf("      ti=%-3d i%-2d %-46s annonces %6d (fen %5d / hors %6d) densite %8.3f"+
					" vs %8.3f -> %s (serre 1 s : %s)", ti, i, zcName(gram[ti], i), s.byIndex[i],
					s.inWin[i], s.outWin[i], dIn, dOut, zcRatio(dIn, dOut), tight)
			}
		}
	}
	zcWriteFile(t, filepath.Join(out, short+"_densite.tsv"), sb.String())
}

// zcGateMinAnnonces est le volume minimal d'annonces par film que le gate 0 du lot C exige
// d'un composant non porte de ti=10 ou ti=12. Ecrit avant la mesure (plan, Gate 0).
const zcGateMinAnnonces = 100

// --- petites fonctions de mise en forme -------------------------------------------------

func zcIsTarget(ti int) bool {
	for _, v := range zcTargetTIs {
		if v == ti {
			return true
		}
	}
	return false
}

func zcName(g zcArchInfo, i int) string {
	if i < len(g.components) {
		return g.components[i]
	}
	return "(hors grammaire)"
}

func zcLevel(g zcArchInfo, i int) string {
	if i < len(g.levels) {
		return fmt.Sprintf("%d", g.levels[i])
	}
	return "-"
}

func zcStatus(g zcArchInfo, i int) string {
	if i >= len(g.ported) {
		return "hors_grammaire"
	}
	if g.ported[i] {
		return "porte"
	}
	return "non_porte"
}

func zcRate(a, b int) float64 {
	if b == 0 {
		return 0
	}
	return float64(a) / float64(b)
}

func zcRatio(a, b float64) string {
	if b == 0 {
		if a == 0 {
			return "0/0"
		}
		return "hors-fenetre-nul"
	}
	return fmt.Sprintf("%.2fx", a/b)
}

func zcTrim(v []int, n int) []int {
	if len(v) > n {
		return v[:n]
	}
	return v
}

func zcJoinInts(v []int) string {
	parts := make([]string, 0, len(v))
	for _, x := range v {
		parts = append(parts, fmt.Sprintf("%d", x))
	}
	return strings.Join(parts, ",")
}

func zcSortedKeysInt(m map[int]int) []int {
	out := make([]int, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Ints(out)
	return out
}

func zcSortedStrings(m map[string]int) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func zcWriteFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("ecriture de %s : %v", path, err)
	}
	t.Logf("  ecrit : %s (%d octets)", path, len(content))
}
