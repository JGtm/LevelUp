package killsource

// world_precision_test.go — INSTRUMENT DE MESURE : LES LARGEURS D AXE DE LA CARTE
// CHANGENT-ELLES CE QUE `Decode` PUBLIE ?
//
// LA QUESTION, ET POURQUOI ELLE SE POSE ICI. `filmdec.WorldObjectPrecision` est un GLOBAL de
// paquet dont le defaut `{13,13,14}` EST l entree `cliffhanger` du catalogue de bornes. Depuis
// le 2026-08-15, `replay.BuildFromFilm` installe les largeurs de la carte du match pour toute
// la duree de son decodage — mais `killsource.Decode` est une chaine d appel DISTINCTE, qui ne
// recoit aucune entree de catalogue : ses films sont donc lus aux largeurs de Cliffhanger,
// quelle que soit la carte.
//
// CE N EST PAS UNE QUESTION DE POSITION, C EST UNE QUESTION DE CURSEUR. La marche
// (`walkFrom` -> `filmdec.DecodeFrameRecords`) deroule la boucle de records d un paquet ; le
// deser d `object-position-component` (traverse.go, chemin world-object) AVANCE le curseur de
// `1 + 1 [+IndexW] + AxisW[0]+AxisW[1]+AxisW[2] + 2` bits. Sur une carte `[17 17 16]` le vrai
// record fait 10 bits de plus que ce que le defaut lit : tout ce qui suit dans le paquet est
// alors decale. Si un tel record precede un dead-state de bipede, la marche meurt avant lui.
//
// CE QUE L INSTRUMENT MESURE, ET AVEC QUELS DENOMINATEURS. Il rejoue `Decode` DEUX FOIS sur le
// meme film — aux largeurs par DEFAUT, puis a celles du CATALOGUE — et confronte les
// denominateurs que le paquet publie DEJA (`Coverage`, `Stats`, `Health`), plus la table des
// morts ligne par ligne. Aucun critere neuf n est invente pour l occasion.
//
// LE TEMOIN EST DANS L INSTRUMENT : sur Cliffhanger les largeurs du catalogue EGALENT le
// defaut, donc les deux passes doivent rendre EXACTEMENT la meme chose. Un ecart y serait de
// la non-reproductibilite, pas un effet des largeurs — et il invaliderait la mesure sur les
// autres films.
//
// LA SONDE D ABSURDE EST LE CONTROLE DU NEGATIF, et sans elle un « aucun ecart » ne vaudrait
// rien. Un instrument mal branche rend TOUJOURS zero ; une troisieme passe a des largeurs
// grossierement fausses ([6 6 6], 26 bits de moins par record d objet du monde) distingue les
// deux lectures d un resultat nul :
//
//	sonde qui CHANGE quelque chose  -> le chemin est bien exerce, et l ecart catalogue/defaut
//	                                   est nul parce que la sortie y est INSENSIBLE ;
//	sonde qui ne change RIEN        -> `Decode` ne depend d AUCUNE largeur d objet du monde,
//	                                   et c est un negatif STRUCTUREL, pas une coincidence.
//
// LECTURE SEULE, garde par KSPREC_FILM, saute partout ailleurs (CI comprise). UN SEUL FILM PAR
// PROCESS : `Decode` prend `filmdec.LockProcessDecode` et remet les globaux de replication a
// leur valeur d origine, mais la comparabilite entre films n est pas l objet ici.
//
// USAGE (depuis apps/go-api) :
//
//	CGO_ENABLED=0 KSPREC_FILM=<repo>/data/cache/film_chunks/00502e52 \
//	  KSPREC_BOUNDS=<repo>/data/titles/halo_infinite/reference/map_quant_bounds.json \
//	  KSPREC_MAP=Bazaar \
//	  go test ./internal/games/halo_infinite/film/killsource/ \
//	  -run '^TestKillSourceWorldPrecisionImpact$' -timeout 60m -v

import (
	"context"
	"fmt"
	"os"
	"sort"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/analysis/filmsource"
)

const (
	ksPrecFilmEnv   = "KSPREC_FILM"
	ksPrecBoundsEnv = "KSPREC_BOUNDS"
	ksPrecMapEnv    = "KSPREC_MAP"
)

// TestKillSourceWorldPrecisionImpact : la mesure A/B.
func TestKillSourceWorldPrecisionImpact(t *testing.T) {
	dir := os.Getenv(ksPrecFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument de mesure saute", ksPrecFilmEnv)
	}
	entry := ksPrecEntry(t)
	def := filmdec.WorldObjectPrecision.AxisW
	t.Logf("== FILM %s ==", dir)
	t.Logf("  largeurs par DEFAUT du paquet : %v · largeurs DU CATALOGUE : %v (module %s)",
		def, entry.AxisWidths, entry.Module)
	if entry.AxisWidths == def {
		t.Log("  TEMOIN : les deux jeux de largeurs sont IDENTIQUES — les deux passes doivent")
		t.Log("  rendre exactement la meme chose, sans quoi la mesure n est pas reproductible.")
	}

	a := ksPrecRun(t, dir, def)
	b := ksPrecRun(t, dir, entry.AxisWidths)
	c := ksPrecRun(t, dir, ksPrecAbsurdWidths)
	ksPrecReport(t, fmt.Sprintf("DEFAUT    %v", def), a)
	ksPrecReport(t, fmt.Sprintf("CATALOGUE %v", entry.AxisWidths), b)
	ksPrecReport(t, fmt.Sprintf("SONDE D ABSURDE %v", ksPrecAbsurdWidths), c)
	ksPrecDelta(t, a, b)
	ksPrecDelta(t, a, c)
	t.Logf("  LECTURE DU NEGATIF : %s", ksPrecNegativeReading(a, b, c))
}

// ksPrecAbsurdWidths : des largeurs grossierement fausses, hors de toute entree de catalogue
// (le minimum observe est [13 12 11] sur Aquarius). Elles retirent 26 bits par record d objet
// du monde : si meme celles-la ne changent rien, aucune largeur ne change rien.
var ksPrecAbsurdWidths = [3]uint{6, 6, 6}

// ksPrecNegativeReading : ce qu un ecart nul veut dire, selon ce que la sonde a fait.
//
// LE TROISIEME CAS NE DIT PAS << le chemin ne sert pas >>, et c est la nuance qui compte :
// [TestKillSourceWalkArchetypes] montre la MARCHE changer sous ces memes largeurs. Ce qui est
// insensible, c est la SORTIE PUBLIEE — le filtre de credibilite, l appariement par couple
// exact et le rattrapage du scan absorbent l ecart avant publication.
func ksPrecNegativeReading(def, cat, absurd ksPrecMeasure) string {
	switch {
	case !ksPrecIdentical(def, cat):
		return "sans objet — l ecart catalogue/defaut n est PAS nul"
	case !ksPrecIdentical(def, absurd):
		return "la sortie ne bouge PAS sous les largeurs du catalogue, mais elle bouge sous des " +
			"largeurs grossierement fausses : la marge existe, elle est seulement plus large " +
			"que l ecart entre cartes reelles"
	default:
		return "SORTIE INSENSIBLE — meme [6 6 6] ne deplace aucune ligne publiee. Ce n est PAS " +
			"que le chemin ne serve pas (cf. TestKillSourceWalkArchetypes : la marche, elle, " +
			"change) : c est que la credibilite et l hybride absorbent l ecart avant publication"
	}
}

// ksPrecIdentical : deux passes rendent-elles exactement la meme chose ?
func ksPrecIdentical(a, b ksPrecMeasure) bool {
	return a.res.Coverage == b.res.Coverage &&
		a.res.Stats.Walk == b.res.Stats.Walk && a.res.Stats.Scan == b.res.Stats.Scan &&
		len(a.lignes) == len(b.lignes) && ksPrecSameLines(a, b)
}

// TestKillSourceWalkArchetypes : LA CAUSE DU NEGATIF, comptee et non supposee.
//
// La mesure A/B dit que `Decode` ne bouge pas d un pouce quelles que soient les largeurs
// d objet du monde. Ce test dit POURQUOI, et il commence par verifier que l instrument mord :
// il rejoue la marche a DEUX jeux de largeurs et publie les deux histogrammes. Si les deux
// etaient identiques, l installation du global ne prendrait pas effet et le negatif de la
// mesure A/B ne vaudrait rien.
//
// LE COMPTEUR QUI DECIDE est `dead apres position d objet du monde` : un dead-state de bipede
// lu APRES un composant dont la largeur est fausse est un dead-state POTENTIELLEMENT perdu.
// A zero, la sortie ne peut pas dependre de ces largeurs, quel que soit le desalignement
// qu elles causent plus loin dans le paquet.
//
// Meme garde et meme usage que la mesure A/B ; `KSPREC_BOUNDS` / `KSPREC_MAP` inutiles ici.
func TestKillSourceWalkArchetypes(t *testing.T) {
	dir := os.Getenv(ksPrecFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument de mesure saute", ksPrecFilmEnv)
	}
	release := filmdec.LockProcessDecode()
	defer release()
	resetGlobals()
	prev := filmdec.WorldObjectPrecision
	t.Cleanup(func() { filmdec.WorldObjectPrecision = prev })

	src, err := filmsource.LoadDir(dir, nil)
	if err != nil {
		t.Fatalf("chunks de %s : %v", dir, err)
	}
	f, err := loadFilm(src)
	if err != nil {
		t.Fatalf("film %s : %v", dir, err)
	}
	tl, err := newTimeline(f)
	if err != nil {
		t.Fatalf("timeline de %s : %v", dir, err)
	}
	tl.rewind()
	calib := calibrate(f, tl, DefaultOptions().Views)
	t.Logf("== FILM %s ==", dir)
	t.Logf("  calibration : %s", calib)

	largeurs := [][3]uint{prev.AxisW}
	if os.Getenv(ksPrecBoundsEnv) != "" && os.Getenv(ksPrecMapEnv) != "" {
		largeurs = append(largeurs, ksPrecEntry(t).AxisWidths)
	}
	largeurs = append(largeurs, ksPrecAbsurdWidths)

	hs := make([]ksPrecHist, 0, len(largeurs))
	for _, w := range largeurs {
		filmdec.WorldObjectPrecision.AxisW = w
		h := ksPrecWalkHistogram(f, tl, DefaultOptions().Views)
		hs = append(hs, h)
		t.Logf("  -- largeurs %v --", w)
		t.Logf("    records atteints par la MARCHE : %d · dont porteurs d un composant de "+
			"position d objet du monde : %d (%.2f %%)",
			h.records, h.worldPos, ksPrecPct(h.worldPos, h.records))
		t.Logf("    records de bipede (ti=%d) : %d · dead-states lus : %d",
			bipedArchetype, h.biped, h.dead)
		t.Logf("    APRES un composant de position d objet du monde, dans le meme paquet : "+
			"%d records · %d bipedes · %d DEAD-STATES",
			h.afterWorldPos, h.bipedAfterWorldPos, h.deadAfterWorldPos)
		for _, ti := range h.sortedTypes() {
			t.Logf("      ti=%-3d : %6d records", ti, h.byType[ti])
		}
	}
	last := len(hs) - 1
	if hs[0].records == hs[last].records && hs[0].worldPos == hs[last].worldPos {
		t.Fatalf("INSTRUMENT MUET : la marche rend exactement les memes comptes a %v et a %v — "+
			"le global n est pas installe, et le negatif de la mesure A/B serait sans valeur",
			prev.AxisW, ksPrecAbsurdWidths)
	}
	t.Logf("  L INSTRUMENT MORD : la marche elle-meme change entre %v et %v (%d -> %d records "+
		"lus, %d -> %d composants de position d objet du monde, %d -> %d dead-states)",
		prev.AxisW, ksPrecAbsurdWidths, hs[0].records, hs[last].records,
		hs[0].worldPos, hs[last].worldPos, hs[0].dead, hs[last].dead)
}

// ksPrecHist : l histogramme des archetypes atteints par la marche.
type ksPrecHist struct {
	records            int
	biped              int
	dead               int // records portant un composant dead-state
	worldPos           int // records dont un composant lu est `object-position-component`
	afterWorldPos      int // records lus APRES un tel composant, dans le meme paquet
	bipedAfterWorldPos int
	deadAfterWorldPos  int
	byType             map[uint32]int
}

func (h ksPrecHist) sortedTypes() []uint32 {
	out := make([]uint32, 0, len(h.byType))
	for ti := range h.byType {
		out = append(out, ti)
	}
	sort.Slice(out, func(i, j int) bool { return h.byType[out[i]] > h.byType[out[j]] })
	if len(out) > 12 {
		out = out[:12]
	}
	return out
}

// worldObjectPositionComponent : le nom, dans le registre ECS du film, du composant dont le
// deser lit `filmdec.WorldObjectPrecision` (traverse.go, chemin world-object d i0).
const worldObjectPositionComponent = "object-position-component"

// ksPrecWalkHistogram rejoue EXACTEMENT le parcours de `runWalk` — meme localisateur, meme
// restauration du monde — mais compte au lieu de filtrer.
func ksPrecWalkHistogram(f *film, tl *timeline, views int) ksPrecHist {
	tl.rewind()
	cfg := filmdec.DefaultFrameConfig()
	h := ksPrecHist{byType: map[uint32]int{}}
	for i := range f.t0 {
		p := &f.t0[i]
		w := tl.advanceTo(p.ts)
		start := 2
		if hasEvents(p) {
			s := locateRecords(p.payload, w, cfg)
			if s < 0 {
				continue
			}
			start = s
		}
		snap := w.Snapshot()
		recs := walkFrom(p.payload, w, cfg, start, views)
		w.Restore(snap)
		vuWorldPos := false
		for k := range recs {
			r := &recs[k]
			h.records++
			h.byType[r.TypeIndex]++
			mort := r.Trace.Dead != nil
			if r.TypeIndex == bipedArchetype {
				h.biped++
			}
			if mort {
				h.dead++
			}
			if vuWorldPos {
				h.afterWorldPos++
				if r.TypeIndex == bipedArchetype {
					h.bipedAfterWorldPos++
				}
				if mort {
					h.deadAfterWorldPos++
				}
			}
			for _, c := range r.Trace.Comps {
				if c.Name == worldObjectPositionComponent {
					h.worldPos++
					vuWorldPos = true
					break
				}
			}
		}
	}
	tl.rewind()
	return h
}

// ksPrecEntry : l entree de catalogue de la carte du match. C est la MEME entree qui porte les
// bornes — bornes et largeurs ne se dissocient pas.
func ksPrecEntry(t *testing.T) filmdec.MapQuantEntry {
	t.Helper()
	boundsPath, mapName := os.Getenv(ksPrecBoundsEnv), os.Getenv(ksPrecMapEnv)
	if boundsPath == "" || mapName == "" {
		t.Skipf("%s / %s absents : la source des largeurs est le CATALOGUE, pas le film",
			ksPrecBoundsEnv, ksPrecMapEnv)
	}
	cat, err := filmdec.LoadMapQuantCatalog(boundsPath)
	if err != nil {
		t.Fatalf("catalogue de bornes %s : %v", boundsPath, err)
	}
	entry, err := cat.Lookup(mapName)
	if err != nil {
		t.Fatalf("carte %q ABSENTE du catalogue : %v — une carte hors catalogue ne se mesure "+
			"pas comme si elle y etait", mapName, err)
	}
	return entry
}

// ksPrecMeasure : ce qu une passe rend, reduit a ce qui se compare.
type ksPrecMeasure struct {
	axisW  [3]uint
	res    *Result
	lignes map[ksPrecKey]ksPrecLine
}

// ksPrecKey identifie une mort publiee : instant + victime. Le kill-feed date les deux passes
// de la meme facon (il ne depend d aucune largeur), donc la cle est stable par construction.
type ksPrecKey struct {
	ms     int
	victim string
}

// ksPrecLine : ce que la ligne dit, sous forme comparable.
type ksPrecLine struct {
	killer   string
	tag      uint32
	origin   Origin
	path     Path
	diverges bool
}

// ksPrecRun installe des largeurs et decode. `Decode` prend le verrou de process lui-meme :
// l installation se fait AVANT l appel, et le test est le seul decodeur du process.
func ksPrecRun(t *testing.T, dir string, axisW [3]uint) ksPrecMeasure {
	t.Helper()
	prev := filmdec.WorldObjectPrecision
	t.Cleanup(func() { filmdec.WorldObjectPrecision = prev })
	filmdec.WorldObjectPrecision.AxisW = axisW

	src, err := filmsource.LoadDir(dir, nil)
	if err != nil {
		t.Fatalf("chunks de %s : %v", dir, err)
	}
	res, err := Decode(context.Background(), dir, src, nil)
	if err != nil {
		t.Fatalf("decodage de %s aux largeurs %v : %v", dir, axisW, err)
	}
	m := ksPrecMeasure{axisW: axisW, res: res, lignes: map[ksPrecKey]ksPrecLine{}}
	for _, k := range res.Kills {
		m.lignes[ksPrecKey{k.TimeMS, k.Victim}] = ksPrecLine{
			killer: k.Feed.Killer, tag: k.Source.Tag, origin: k.Read.Origin,
			path: k.Read.Path, diverges: k.Diverges,
		}
	}
	return m
}

// ksPrecReport publie une passe avec TOUS ses denominateurs, ceux du paquet.
func ksPrecReport(t *testing.T, label string, m ksPrecMeasure) {
	t.Helper()
	c, s, h := m.res.Coverage, m.res.Stats, m.res.Health
	t.Logf("  %s", label)
	t.Logf("    calibration retenue : %s", m.res.Calibration)
	t.Logf("    COUVERTURE : %d morts publiees sur %d couples REELS (%.2f %%) · "+
		"reconstruits %d · fantomes %d · meme instant %d",
		c.Covered, c.RealPairs, ksPrecPct(c.Covered, c.RealPairs),
		c.ReconstructedPairs, c.GhostPairs, c.SameInstantPairs)
	t.Logf("    feed : %d kills · %d morts · bots tues %d · tues PAR un bot %d",
		c.FeedKills, c.FeedDeaths, c.BotDeaths, c.BotKillerDeaths)
	t.Logf("    voie MARCHE : population %d · apparies %d (%.2f %%) · publies %d",
		s.Walk.Population, s.Walk.Matched, 100*s.Walk.Ratio(), s.Walk.Published)
	t.Logf("    voie SCAN   : population %d · apparies %d (%.2f %%) · publies %d",
		s.Scan.Population, s.Scan.Matched, 100*s.Scan.Ratio(), s.Scan.Published)
	t.Logf("    auto-infligees : marche %d/%d publiees %d · scan %d/%d publiees %d",
		s.SelfWalk.Matched, s.SelfWalk.Population, s.SelfWalk.Published,
		s.SelfScan.Matched, s.SelfScan.Population, s.SelfScan.Published)
	t.Logf("    paquets a events %d · localises %d (%.2f %%)",
		s.PacketsWithEvents, s.PacketsLocated, ksPrecPct(s.PacketsLocated, s.PacketsWithEvents))
	t.Logf("    accord des deux voies %d · DESACCORD %d · redondants %d · sans bit %d · "+
		"multi-candidat %d", s.Agree, s.Disagree, s.Redundant, s.NoBit, s.MultiCandidate)
	t.Logf("    divergences credit/source : %d · morts non revendiquees : %d",
		ksPrecDiverges(m), len(m.res.UnclaimedDeaths))
	t.Logf("    sante : candidats %d · publies %d · inexpliques couple %d / soi %d / bot %d · "+
		"hors roster %d · tag hors catalogue marche %d",
		h.Candidates, h.Published, h.UnexplainedPair, h.UnexplainedSelf, h.UnexplainedBotIdx,
		h.OutOfRoster, h.TagOutOfCatalogueWalk)
	t.Logf("    marge de bijection %d · publiable ligne a ligne : %v · alertes %v",
		m.res.BijectionMargin, m.res.LineByLinePublishable(), h.Alerts())
}

// ksPrecDiverges : nombre de lignes ou le credit et la source divergent.
func ksPrecDiverges(m ksPrecMeasure) int {
	n := 0
	for _, k := range m.res.Kills {
		if k.Diverges {
			n++
		}
	}
	return n
}

// ksPrecPct : pourcentage a denominateur jamais nul.
func ksPrecPct(num, den int) float64 {
	if den <= 0 {
		return 0
	}
	return 100 * float64(num) / float64(den)
}

// ksPrecDelta publie l ecart en clair, ligne a ligne, et le verdict.
func ksPrecDelta(t *testing.T, a, b ksPrecMeasure) {
	t.Helper()
	var apparues, disparues, changees, identiques int
	var details []string
	for k, lb := range b.lignes {
		la, ok := a.lignes[k]
		switch {
		case !ok:
			apparues++
			details = append(details, fmt.Sprintf("      + %6d ms %-20s tag %08x %s",
				k.ms, k.victim, lb.tag, lb.origin))
		case la != lb:
			changees++
			details = append(details, fmt.Sprintf("      ~ %6d ms %-20s tag %08x -> %08x · %s -> %s",
				k.ms, k.victim, la.tag, lb.tag, la.path, lb.path))
		default:
			identiques++
		}
	}
	for k, la := range a.lignes {
		if _, ok := b.lignes[k]; !ok {
			disparues++
			details = append(details, fmt.Sprintf("      - %6d ms %-20s tag %08x %s",
				k.ms, k.victim, la.tag, la.origin))
		}
	}
	sort.Strings(details)

	t.Logf("  ECART %v -> %v", a.axisW, b.axisW)
	t.Logf("    lignes : %d apparaissent · %d disparaissent · %d changent · %d identiques",
		apparues, disparues, changees, identiques)
	for _, d := range details {
		t.Log(d)
	}
	ca, cb := a.res.Coverage, b.res.Coverage
	t.Logf("    couverture : %d/%d (%.2f %%) -> %d/%d (%.2f %%) · %+d morts publiees",
		ca.Covered, ca.RealPairs, ksPrecPct(ca.Covered, ca.RealPairs),
		cb.Covered, cb.RealPairs, ksPrecPct(cb.Covered, cb.RealPairs), cb.Covered-ca.Covered)
	t.Logf("    voie MARCHE apparies : %d -> %d (%+d) · population %d -> %d (%+d)",
		a.res.Stats.Walk.Matched, b.res.Stats.Walk.Matched,
		b.res.Stats.Walk.Matched-a.res.Stats.Walk.Matched,
		a.res.Stats.Walk.Population, b.res.Stats.Walk.Population,
		b.res.Stats.Walk.Population-a.res.Stats.Walk.Population)
	// La sante se publie A PART de l identite des lignes : elle compte des CANDIDATS, pas des
	// morts. Elle peut bouger de quelques unites sans qu une seule ligne change — c est une
	// trace supplementaire que les largeurs mordent quelque part, et il faut la voir.
	t.Logf("    sante : candidats %d -> %d (%+d) · inexpliques soi %d -> %d · couple %d -> %d",
		a.res.Health.Candidates, b.res.Health.Candidates,
		b.res.Health.Candidates-a.res.Health.Candidates,
		a.res.Health.UnexplainedSelf, b.res.Health.UnexplainedSelf,
		a.res.Health.UnexplainedPair, b.res.Health.UnexplainedPair)
	t.Logf("    VERDICT : %s", ksPrecVerdict(a, b))
}

// ksPrecVerdict tranche dans le sens prescrit par le plan : a egalite stricte de sortie, le
// defaut reste (on ne modifie pas ce qui ne change rien).
func ksPrecVerdict(a, b ksPrecMeasure) string {
	if ksPrecIdentical(a, b) {
		return "AUCUN ECART — ces largeurs ne mordent pas sur ce que Decode publie"
	}
	return "ECART REEL — ces largeurs changent ce que Decode publie"
}

// ksPrecSameLines : les deux tables de morts sont-elles identiques, ligne pour ligne ?
func ksPrecSameLines(a, b ksPrecMeasure) bool {
	for k, la := range a.lignes {
		if lb, ok := b.lignes[k]; !ok || la != lb {
			return false
		}
	}
	return true
}
