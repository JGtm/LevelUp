package filmdec

// vitesse_filtre_outils_research_test.go — outillage de TestVitesseFiltre (lot R3, plan
// PLAN_LECTURE_FIABLE_EQUIPEMENT_2026-09-03) : recensement des têtes d'événement 117,
// fenêtrage par chunks, rejeu décision par décision de la sémantique de DropTeleports,
// profondeur de corroboration (option B), catalogue de déquantification et identification
// de carte par calibration, lecture de l'artefact publié. La question de recherche et
// l'usage sont documentés en tête de vitesse_filtre_research_test.go.

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"
)

// vitfCtx porte l'environnement d'une exécution : film, origine d'horloge, artefact,
// catalogue de bornes, entrée de carte (donnée ou identifiée) et grammaire de décodage.
type vitfCtx struct {
	dir     string
	origine uint64
	art     *vitfArt
	cat     *MapQuantCatalog
	entree  *MapQuantEntry
	nom     string
	lay     I0Layout
}

// vitfSetup lit l'environnement : catalogue OBLIGATOIRE (le piège des bornes d'affichage
// est documenté en tête de fichier), carte et artefact optionnels — mais l'un des deux est
// nécessaire pour déquantifier (l'identification s'appuie sur l'artefact).
func vitfSetup(t *testing.T, dir string) *vitfCtx {
	t.Helper()
	ctx := &vitfCtx{dir: dir}
	origine, err := vitfOrigine(dir)
	if err != nil {
		t.Fatalf("origine d'horloge illisible : %v", err)
	}
	ctx.origine = origine
	catPath := strings.TrimSpace(os.Getenv(vitfCatalogueEnv))
	if catPath == "" {
		t.Fatalf("%s absent : les bornes de déquantification viennent du catalogue de production (map_quant_bounds.json), jamais du champ bounds de l'artefact (cadrage d'affichage)", vitfCatalogueEnv)
	}
	ctx.cat, err = LoadMapQuantCatalog(catPath)
	if err != nil {
		t.Fatalf("%s : %v", vitfCatalogueEnv, err)
	}
	if nom := strings.TrimSpace(os.Getenv(vitfCarteEnv)); nom != "" {
		e, err := ctx.cat.Lookup(nom)
		if err != nil {
			t.Fatalf("%s=%q : %v", vitfCarteEnv, nom, err)
		}
		ctx.entree, ctx.nom = &e, NormalizeMapName(nom)
		t.Logf("== CARTE (donnée) : %s · bornes %v -> %v · largeurs %v ==", ctx.nom, e.Min, e.Max, e.AxisWidths)
	}
	ctx.art = vitfChargerArtefact(t)
	if ctx.entree == nil && ctx.art == nil {
		t.Fatalf("ni %s ni %s : impossible de déquantifier (carte inconnue, pas d'artefact pour l'identifier)", vitfCarteEnv, vitfArtefactEnv)
	}
	return ctx
}

// vitfEvent est une tête d'événement 117 EquipmentTranslocatorTeleportEffects (R1 §4.2 :
// premier octet 0xFA, ref0 = slot du bipède — porte 1 bit, index 8 bits base 512, gen 2).
type vitfEvent struct {
	ts     uint64 // horloge MOTEUR (celle des paquets)
	slot   uint32
	chunk  int
	paquet int
}

// vitfSpan est l'étendue temporelle des paquets delta d'un chunk (pour borner les
// décodages aux seuls chunks couvrant une fenêtre — jamais un film entier sans filtre).
type vitfSpan struct {
	chunk    int
	min, max uint64
}

// vitfScanEvenements recense les têtes de type 117 du film ENTIER — même canal et même
// décodage de ref0 que TestFailleActivationEvenements (RAPPORT_R1 §4), lecture O(1) par
// paquet, aucun balayage bit à bit. Rend aussi l'étendue de chaque chunk et le nombre de
// paquets delta (le dénominateur du recensement).
func vitfScanEvenements(t *testing.T, dir string) ([]vitfEvent, []vitfSpan, int) {
	t.Helper()
	n := CountFilmChunks(dir)
	if n == 0 {
		t.Fatalf("aucun chunk film dans %s", dir)
	}
	var evs []vitfEvent
	var spans []vitfSpan
	deltas := 0
	for c := 1; c <= n; c++ {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		sp := vitfSpan{chunk: c}
		for _, pk := range WalkPackets(data) {
			if pk.Type != PacketTypeDelta || pk.Size < 2 {
				continue
			}
			if sp.min == 0 || pk.TimestampUS < sp.min {
				sp.min = pk.TimestampUS
			}
			if pk.TimestampUS > sp.max {
				sp.max = pk.TimestampUS
			}
			deltas++
			pay := pk.Payload(data)
			if pay[0]&0xC0 != 0xC0 { // bit config + bit de continuation : liste non vide
				continue
			}
			if typ := int(pay[0]&0x3F)<<1 | int(pay[1]>>7); typ != 117 {
				continue
			}
			br := NewBitReader(pay)
			br.Skip(9)
			if !br.ReadBit() {
				t.Logf("  tête 117 @%d us SANS ref0 — ignorée (slot indécodable)", pk.TimestampUS)
				continue
			}
			slot := uint32(br.ReadBits(8)) + 512
			evs = append(evs, vitfEvent{ts: pk.TimestampUS, slot: slot, chunk: c, paquet: pk.Index})
		}
		if sp.max > 0 {
			spans = append(spans, sp)
		}
	}
	return evs, spans, deltas
}

// vitfGroupe : des événements dont les fenêtres se recouvrent, décodés en UNE passe.
type vitfGroupe struct {
	evs    []vitfEvent
	chunks []int
	lo, hi uint64
}

// vitfGroupes fusionne les fenêtres [ts-5 s, ts+10 s] qui se recouvrent et liste les
// chunks dont l'étendue intersecte chaque fenêtre fusionnée.
func vitfGroupes(evs []vitfEvent, spans []vitfSpan) []vitfGroupe {
	tri := append([]vitfEvent(nil), evs...)
	sort.Slice(tri, func(i, j int) bool { return tri[i].ts < tri[j].ts })
	var out []vitfGroupe
	for _, ev := range tri {
		lo := uint64(0)
		if ev.ts > vitfFenAvantUS {
			lo = ev.ts - vitfFenAvantUS
		}
		hi := ev.ts + vitfFenApresUS
		if len(out) > 0 && lo <= out[len(out)-1].hi {
			g := &out[len(out)-1]
			g.evs = append(g.evs, ev)
			if hi > g.hi {
				g.hi = hi
			}
			continue
		}
		out = append(out, vitfGroupe{evs: []vitfEvent{ev}, lo: lo, hi: hi})
	}
	for i := range out {
		for _, sp := range spans {
			if sp.max >= out[i].lo && sp.min <= out[i].hi {
				out[i].chunks = append(out[i].chunks, sp.chunk)
			}
		}
	}
	return vitfFusionnerParChunk(out)
}

// vitfFusionnerParChunk fusionne les groupes consécutifs qui PARTAGENT un chunk : sans
// cela le chunk commun serait décodé deux fois, et les rejets d'arrivée d'un groupe
// compteraient comme « bruit hors zone » dans l'autre (vu sur 1b2d9e08, chunks 17-18/18-19).
func vitfFusionnerParChunk(groupes []vitfGroupe) []vitfGroupe {
	var out []vitfGroupe
	for _, g := range groupes {
		if len(out) > 0 && vitfChunksCommuns(out[len(out)-1].chunks, g.chunks) {
			d := &out[len(out)-1]
			d.evs = append(d.evs, g.evs...)
			for _, c := range g.chunks {
				if d.chunks[len(d.chunks)-1] < c {
					d.chunks = append(d.chunks, c)
				}
			}
			if g.hi > d.hi {
				d.hi = g.hi
			}
			continue
		}
		out = append(out, g)
	}
	return out
}

func vitfChunksCommuns(a, b []int) bool {
	for _, x := range a {
		for _, y := range b {
			if x == y {
				return true
			}
		}
	}
	return false
}

// vitfDecodeQuanta décode les records bipèdes en QUANTA BRUTS (QuantaOnly, sans filtre de
// vitesse ni d'isolement), sur les SEULS chunks donnés (le balayage sans filtre d'un film
// entier tue le process — mesure du 2026-08-18, cf. translocateur_test.go). La
// déquantification est faite ensuite, avec l'entrée de catalogue choisie.
func vitfDecodeQuanta(t *testing.T, dir string, lay *I0Layout, chunks []int) []BipedPosition {
	t.Helper()
	opt := DefaultScanFilmOptions()
	opt.MaxSpeedMPS = 0
	opt.IsolationGapMS = 0
	opt.QuantaOnly = true
	opt.Layout = lay
	opt.Chunks = chunks
	raw, err := ScanFilmBipedPositions(dir, opt)
	if err != nil {
		t.Fatalf("décodage sans filtre impossible (chunks %v) : %v", chunks, err)
	}
	return raw
}

// vitfDequantTous convertit les quanta en coordonnées monde avec les bornes de l'entrée de
// catalogue — le même DequantBipedAxis que la production.
func vitfDequantTous(qs []BipedPosition, lay I0Layout, wr Vec3Range) []BipedPosition {
	out := append([]BipedPosition(nil), qs...)
	for i := range out {
		out[i].X = DequantBipedAxis(out[i].Q[0], 0, lay, wr)
		out[i].Y = DequantBipedAxis(out[i].Q[1], 1, lay, wr)
		out[i].Z = DequantBipedAxis(out[i].Q[2], 2, lay, wr)
		out[i].HasWorld = true
	}
	return out
}

// vitfChoisirEntree identifie la carte : parmi les entrées du catalogue dont le découpage
// égale la grammaire décodée, celle qui minimise l'écart médian 2D entre les positions
// déquantifiées du slot et la piste PUBLIÉE, sur les frames d'AVANT saut des événements du
// groupe (la production y est fiable). Fatal si le meilleur score dépasse 1 m : la mesure
// serait fausse en silence.
func vitfChoisirEntree(t *testing.T, ctx *vitfCtx, qs []BipedPosition, g vitfGroupe) {
	t.Helper()
	type cand struct {
		nom   string
		e     MapQuantEntry
		score float64
		n     int
	}
	var cands []cand
	noms := make([]string, 0, len(ctx.cat.Maps))
	for nom := range ctx.cat.Maps {
		noms = append(noms, nom)
	}
	sort.Strings(noms)
	for _, nom := range noms {
		e := ctx.cat.Maps[nom]
		if e.Layout() != ctx.lay {
			continue
		}
		score, n := vitfScoreCalibration(ctx, qs, g, e.Range())
		if n >= 10 {
			cands = append(cands, cand{nom: nom, e: e, score: score, n: n})
		}
	}
	if len(cands) == 0 {
		t.Fatalf("aucune entrée de catalogue au découpage %+v avec assez de paires piste/film : donner %s", ctx.lay, vitfCarteEnv)
	}
	sort.Slice(cands, func(i, j int) bool { return cands[i].score < cands[j].score })
	for i, c := range cands {
		if i == 3 {
			break
		}
		t.Logf("  candidat carte %q : écart médian %.3f m sur %d paires", c.nom, c.score, c.n)
	}
	best := cands[0]
	if best.score > 1.0 {
		t.Fatalf("identification de carte impossible : meilleur écart %.2f m (%q) — donner %s", best.score, best.nom, vitfCarteEnv)
	}
	e := best.e
	ctx.entree, ctx.nom = &e, best.nom
	t.Logf("== CARTE (identifiée par calibration) : %s (module %s) · bornes %v -> %v · écart médian %.3f m sur %d paires ==",
		best.nom, e.Module, e.Min, e.Max, best.score, best.n)
	t.Log("   (des cartes Forge partagent le même canevas : à bornes identiques, le nom est indécidable et sans effet sur la mesure)")
}

// vitfScoreCalibration apparie les points publiés des slots des événements (frames d'avant
// saut, de -30 à -1) à l'échantillon brut le plus proche en temps (<= 60 ms) et rend
// l'écart médian 2D après déquantification par wr, et le nombre de paires.
func vitfScoreCalibration(ctx *vitfCtx, qs []BipedPosition, g vitfGroupe, wr Vec3Range) (float64, int) {
	var ecarts []float64
	for _, ev := range g.evs {
		filmMS := (int64(ev.ts) - int64(ctx.origine)) / 1000
		frameEv := int((filmMS - *ctx.art.OriginMs) / int64(ctx.art.FrameIntervalMS))
		tr := vitfPiste(ctx.art, ev.slot, frameEv)
		if tr == nil {
			continue
		}
		var idx []int
		for i, p := range qs {
			if p.Slot == ev.slot {
				idx = append(idx, i)
			}
		}
		for _, pt := range tr.Points {
			if pt.T < frameEv-30 || pt.T >= frameEv {
				continue
			}
			ptUS := uint64((*ctx.art.OriginMs+int64(pt.T)*int64(ctx.art.FrameIntervalMS))*1000) + ctx.origine
			if i := vitfPlusProche(qs, idx, ptUS); i >= 0 {
				x := DequantBipedAxis(qs[i].Q[0], 0, ctx.lay, wr)
				y := DequantBipedAxis(qs[i].Q[1], 1, ctx.lay, wr)
				ecarts = append(ecarts, vitfDist2D(x, y, pt.X, pt.Y))
			}
		}
	}
	return vitfMediane(ecarts), len(ecarts)
}

// vitfPlusProche rend l'indice de l'échantillon de idx le plus proche de ts (<= 60 ms).
func vitfPlusProche(qs []BipedPosition, idx []int, ts uint64) int {
	best, bestD := -1, uint64(60_000)
	for _, i := range idx {
		d := qs[i].TimestampUS - ts
		if qs[i].TimestampUS < ts {
			d = ts - qs[i].TimestampUS
		}
		if d <= bestD {
			best, bestD = i, d
		}
	}
	return best
}

// vitfSim est le verdict décision par décision d'une passe de filtre.
type vitfSim struct {
	accepte []bool
	// motif : 0 = décision de production pure · 'e' = accepté par exemption (option A) ·
	// 's' = réancrage aveugle de production (streak épuisé, vitesse toujours > seuil).
	motif []byte
}

type vitfSimOpts struct {
	maxSpeed float64
	// exemption (option A) : lever le filtre à ±vitfExemptionUS d'un événement 117 du
	// même slot. nil = production pure.
	exemption []vitfEvent
}

// vitfSimuler rejoue la sémantique EXACTE de DropTeleports (offline_filters.go : ancre par
// slot = dernière position acceptée, rejet si vitesse > maxSpeed, réancrage aveugle après
// maxRejectStreak rejets consécutifs), en notant chaque décision. L'option A remplace un
// rejet par une acceptation (et un réancrage) quand l'échantillon est exempté.
func vitfSimuler(samples []BipedPosition, o vitfSimOpts) vitfSim {
	res := vitfSim{accepte: make([]bool, len(samples)), motif: make([]byte, len(samples))}
	type ancre struct {
		p      BipedPosition
		ok     bool
		streak int
	}
	ancres := map[uint32]*ancre{}
	for i, p := range samples {
		a := ancres[p.Slot]
		if a == nil {
			a = &ancre{}
			ancres[p.Slot] = a
		}
		if a.ok && speedFrom(a.p, p) > o.maxSpeed {
			if a.streak < maxRejectStreak {
				if !vitfExempte(o.exemption, p.Slot, p.TimestampUS) {
					a.streak++
					continue
				}
				res.motif[i] = 'e'
			} else {
				res.motif[i] = 's'
			}
		}
		a.p, a.ok, a.streak = p, true, 0
		res.accepte[i] = true
	}
	return res
}

func vitfExempte(evs []vitfEvent, slot uint32, ts uint64) bool {
	for _, ev := range evs {
		if ev.slot != slot {
			continue
		}
		if d := int64(ts) - int64(ev.ts); d >= -vitfExemptionUS && d <= vitfExemptionUS {
			return true
		}
	}
	return false
}

// vitfProfondeur mesure la corroboration de l'échantillon d'indice at : le nombre
// d'échantillons SUIVANTS du même slot (plafonné à vitfCorrobMax) qui s'enchaînent depuis
// lui sous maxSpeed m/s. Une aberration du balayage bit à bit est un point sans suite
// (profondeur 0-1) ; une vraie arrivée est suivie de toute la trajectoire (profondeur au
// plafond). C'est le « k » mesurable du réancrage par corroboration (option B).
func vitfProfondeur(samples []BipedPosition, idx []int, at int, maxSpeed float64) int {
	pos := -1
	for k, i := range idx {
		if i == at {
			pos = k
			break
		}
	}
	if pos < 0 {
		return 0
	}
	depth := 0
	prev := samples[at]
	for k := pos + 1; k < len(idx) && depth < vitfCorrobMax; k++ {
		next := samples[idx[k]]
		if speedFrom(prev, next) > maxSpeed {
			break
		}
		depth++
		prev = next
	}
	return depth
}

// vitfBruit compte les rejets de production HORS zones d'arrivée et leur profondeur de
// corroboration — le dénominateur du risque de l'option B. Les rejets hors zone
// CORROBORÉS (profondeur >= 2) sont listés un par un : ce sont les points que l'option B
// publierait en plus, il faut pouvoir les regarder.
type vitfBruit struct {
	origine               uint64
	ech, rejets, horsZone int
	prof                  [vitfCorrobMax + 1]int
	exemples              []string
}

// vitfCompterBruit reçoit les événements du FILM ENTIER (pas ceux du groupe) : une zone
// d'arrivée d'un autre groupe n'est pas du bruit.
func vitfCompterBruit(b *vitfBruit, samples []BipedPosition, prod vitfSim, evs []vitfEvent) {
	parSlot := map[uint32][]int{}
	for i, p := range samples {
		parSlot[p.Slot] = append(parSlot[p.Slot], i)
	}
	b.ech += len(samples)
	for i, p := range samples {
		if prod.accepte[i] {
			continue
		}
		b.rejets++
		if vitfDansZone(evs, p.Slot, p.TimestampUS) {
			continue
		}
		b.horsZone++
		prof := vitfProfondeur(samples, parSlot[p.Slot], i, DefaultMaxSpeedMPS)
		b.prof[prof]++
		if prof >= 2 {
			b.exemples = append(b.exemples, fmt.Sprintf(
				"slot %d @%d ms (%.1f, %.1f, %.1f) profondeur %d",
				p.Slot, (int64(p.TimestampUS)-int64(b.origine))/1000, p.X, p.Y, p.Z, prof))
		}
	}
}

func vitfDansZone(evs []vitfEvent, slot uint32, ts uint64) bool {
	for _, ev := range evs {
		if ev.slot != slot {
			continue
		}
		if d := int64(ts) - int64(ev.ts); d >= -vitfZoneAvantUS && d <= vitfZoneApresUS {
			return true
		}
	}
	return false
}

// vitfOrigine rend l'horodatage moteur du PREMIER paquet du film — le zéro de l'horloge
// FILM (même lecture que replay.ScanFilmClockOrigin, recopiée car filmdec ne peut pas
// importer replay ; les instruments R1 non committés portent la même copie, on ne s'y
// couple pas).
func vitfOrigine(dir string) (uint64, error) {
	raw, err := ReadFilmChunk(dir, 1)
	if err != nil {
		return 0, err
	}
	packets := WalkPackets(raw)
	if len(packets) == 0 {
		return 0, errVitfChunk1Vide
	}
	return packets[0].TimestampUS, nil
}

var errVitfChunk1Vide = &vitfErreur{"aucun paquet lisible dans le chunk 1"}

type vitfErreur struct{ s string }

func (e *vitfErreur) Error() string { return e.s }
