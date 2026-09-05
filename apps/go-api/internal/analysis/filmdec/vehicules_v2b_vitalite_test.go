package filmdec

// vehicules_v2b_vitalite_test.go — INSTRUMENT DE MESURE (lot V2b, signal 2 : DESTRUCTION par la
// VITALITE du vehicule i4/i5). LECTURE SEULE, garde par V2B_FILMS.
//
// L'IDEE (insight utilisateur). Le recensement des images-cles borne la fin de vie d'un vehicule a
// +/-20 s, jamais datee (item 4 du lot V2 : mort-coincidente au hasard, i14 sous plancher). Mais la
// VITALITE de corps i4 (object-body-vitality) est un composant PORTE dont la valeur est capturee par
// le meme balayage que les positions (offline_aim.go : componentVitals, sous CaptureDirs) : le
// vehicule qui prend des degats emet i4, et la destruction = i4 -> 0, datee A LA MILLISECONDE sur
// l'horloge du film. C'est le signal DATE qui manquait.
//
// CE QUE CET INSTRUMENT REUTILISE, sans rien recopier :
//   - la bande de slots ti=40 + le recensement des vies : ScanFilmWorldObjectKeyframes(dir, 40) ;
//   - la CAPTURE de la vitalite i4/i5 : ScanFilmBipedPositionsForBand(dir, NewSlotBand(band), {CaptureDirs}) —
//     ti=40 porte la meme grammaire dyn.-prec. que le bipede (i0), et i4/i5 la suivent dans le
//     masque ; HealthAt()/ShieldAt() rendent les fractions [0,1] deja portees (vitality.go) ;
//   - les bornes de carte : LoadMapQuantCatalog (pour un balayage monde ; ici QuantaOnly suffit).
//
// LES SEUILS, ECRITS AVANT LA MESURE.
//   - i4 "a zero" : HealthFraction(i4) <= v2bZeroFrac (0), c.-a-d. Body.Health <= 0. On releve aussi
//     le quantum brut Body.Q (sans hypothese de dequant) et le seuil "proche" v2bNearFrac.
//   - CLASSEMENT d'une vie recensee (slot,gen), fenetre [firstSeen, goneBy] :
//       DETRUIT   : >= 1 echantillon i4 a zero dans la fenetre ;
//       DESPAWN   : >= 1 echantillon i4, aucun a zero (le vehicule a disparu sans i4->0) ;
//       SANS_I4   : 0 echantillon i4 (indecidable).
//   - GATE (destruction datee) : pour une vie DETRUIT, l'instant du PREMIER i4->0 doit tomber dans
//     la fenetre et PRECEDER/COINCIDER avec la fin bornee (lastSeen/goneBy). On mesure gap =
//     lastSeen - firstZero (signe).
//   - TEMOIN de terminalite : un i4->0 de destruction est TERMINAL (aucune image-cle ne recense plus
//     la vie apres lui). Une vie DETRUIT dont le vehicule est REVU apres firstZero est le signal
//     qu'un zero mi-vie n'est PAS la destruction. Gate : la part TERMINALE des vies DETRUIT est
//     elevee ; si les zeros sont uniformement repartis dans la vie, ce n'est pas la destruction.
//
// UN SEUL decodage filmdec par process : le verrou est pris pour tout le test.
//
// USAGE (depuis apps/go-api, cache Go ISOLE) :
//
//	$env:GOCACHE='<scratch>\gocache_v2b'
//	CGO_ENABLED=0 V2B_FILM_ROOT=<repo>/data/cache \
//	  V2B_FILMS="0d76e8f1:behemoth,fccc61cd:launch site" \
//	  V2B_BOUNDS=<repo>/data/titles/halo_infinite/reference/map_quant_bounds.json \
//	  go test ./internal/analysis/filmdec/ -run '^TestV2bVitalite$' -v -timeout 180m

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// SEUILS, ecrits AVANT toute mesure.
const (
	v2bZeroFrac = 0.0  // i4 "a zero" : HealthFraction <= 0 (Body.Health <= 0)
	v2bNearFrac = 0.05 // i4 "proche de zero" : corroboration
)

type v2bFilmSpec struct{ short8, mapKey string }

// v2bLifeStat porte le verdict d'UNE vie de vehicule.
type v2bLifeStat struct {
	nBody, nShield int
	minFrac        float64 // fraction de vie minimale observee (1 si aucun i4)
	minShieldFrac  float64
	hasZero        bool
	firstZeroUS    uint64
	terminal       bool // aucune image-cle ne recense la vie apres firstZeroUS
	firstSeen      uint64
	lastSeen       uint64
	goneBy         uint64
	lastQ          uint8   // quantum i4 du DERNIER echantillon de la vie
	qTrail         []uint8 // trajectoire des quanta i4 (diagnostic)
}

// v2bMapAgg agrege tous les films d'une carte.
type v2bMapAgg struct {
	films                      int
	lives, destroyed, despawn  int
	noI4, terminalDestroyed    int
	gapToLastSeenS             []float64 // (lastSeen - firstZero) en s, sur les vies DETRUIT
	minFracs                   []float64 // fraction min par vie AYANT au moins un i4
	bodySamples, shieldSamples int
	livesWithShield            int
	qHist                      [8]int // histogramme des quanta i4 (buckets de 32)
	lastQZero                  int    // vies decidables dont le DERNIER i4 est <= near
	downSteps, upSteps         int    // monotonicite des quanta i4 dans une vie
}

func TestV2bVitalite(t *testing.T) {
	films := v2bParseFilms(t)
	root := v2bRoot()
	cat := v2bLoadBounds(t)

	release := LockProcessDecode()
	defer release()

	aggs := map[string]*v2bMapAgg{}
	for _, f := range films {
		entry, err := cat.Lookup(f.mapKey)
		if err != nil {
			t.Fatalf("%s : bornes de %q introuvables : %v", f.short8, f.mapKey, err)
		}
		ag := aggs[f.mapKey]
		if ag == nil {
			ag = &v2bMapAgg{}
			aggs[f.mapKey] = ag
		}
		dir := filepath.Join(root, "film_chunks", f.short8)
		t.Run(f.short8, func(t *testing.T) { v2bProcessFilm(t, dir, f.short8, entry, ag) })
	}

	keys := make([]string, 0, len(aggs))
	for k := range aggs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		v2bReport(t, k, aggs[k])
	}
}

func v2bProcessFilm(t *testing.T, dir, short8 string, entry MapQuantEntry, ag *v2bMapAgg) {
	kf := ScanFilmWorldObjectKeyframes(dir, VehicleTypeIndex)
	if len(kf.Band) == 0 {
		t.Fatalf("%s : aucun slot ti=%d aux images-cles", short8, VehicleTypeIndex)
	}
	// Balayage vitalite : QuantaOnly (pas besoin de coordonnees monde) + CaptureDirs (porte i4/i5).
	// MaxSpeed/Isolation a 0 : aucun echantillon de vitalite n'est ecarte par un filtre de position.
	// DynPrecOrientation : ti=40 porte les variantes `-dynamic-precision-` d'i2 et i3
	// (FUN_140c5f7ec / FUN_140d87740), pas celles du bipede. Sans ce drapeau le curseur
	// arrive decale sur i4 — c'est la cause racine du bruit mesure le 2026-09-01.
	// V2B_LEGACY_I2I3=1 rejoue l'ancienne grammaire (temoin de la correction).
	opt := ScanFilmOptions{RequireTag1: false, DropSaturated: true, CaptureDirs: true, QuantaOnly: true,
		DynPrecOrientation: os.Getenv("V2B_LEGACY_I2I3") == ""}
	lay := entry.Layout()
	if lay.Valid() {
		opt.Layout = &lay
	}
	var masks [][]int
	if os.Getenv("V2B_MASK") != "" {
		prev := recordMaskHook
		SetRecordMaskHook(func(idx []int, _ []byte, _ int) { masks = append(masks, append([]int{}, idx...)) })
		defer SetRecordMaskHook(prev)
	}
	pos, err := ScanFilmBipedPositionsForBand(dir, NewSlotBand(kf.Band), opt)
	if err != nil {
		t.Fatalf("%s : balayage vitalite : %v", short8, err)
	}
	if len(masks) == len(pos) && len(masks) > 0 {
		v2bMaskDiag(t, short8, pos, masks)
	}
	bySlot := v2bBodyBySlot(pos)

	// TEMOIN DE CONTROLE : la MEME lecture i4 sur la bande BIPEDE (ti=35), dont la sante EST
	// validee en production (HealthAt sert le rejeu). Si la sante bipede est concentree pres du
	// plein et monotone, mais celle du vehicule uniforme et chaotique, alors i4 du VEHICULE est
	// mal aligne (la valeur ne suit pas la grammaire dyn.-prec. du bipede — cf. i2 refute V1a).
	if os.Getenv("V2B_CONTROL") != "" {
		v2bControlBiped(t, dir, short8, entry)
	}

	stats := v2bClassify(kf, bySlot)
	nBody, nShield := 0, 0
	for _, p := range pos {
		if p.HasBody {
			nBody++
		}
		if p.HasShield {
			nShield++
		}
	}
	ag.films++
	ag.bodySamples += nBody
	ag.shieldSamples += nShield
	dump := os.Getenv("V2B_DUMP") != ""
	dCount, spCount, nCount, term := 0, 0, 0, 0
	for _, s := range stats {
		ag.lives++
		for i, q := range s.qTrail {
			ag.qHist[int(q)/32]++
			if i > 0 {
				switch {
				case int(q) < int(s.qTrail[i-1]):
					ag.downSteps++
				case int(q) > int(s.qTrail[i-1]):
					ag.upSteps++
				}
			}
		}
		if s.nBody > 0 && HealthFraction(DequantEndpoint(uint64(s.lastQ), VitalityBodyMin, VitalityBodyMax, 8, true, true)) <= v2bNearFrac {
			ag.lastQZero++
		}
		switch {
		case s.nBody == 0:
			ag.noI4++
			nCount++
		case s.hasZero:
			ag.destroyed++
			dCount++
			ag.gapToLastSeenS = append(ag.gapToLastSeenS,
				(float64(s.lastSeen)-float64(s.firstZeroUS))/1e6)
			if s.terminal {
				ag.terminalDestroyed++
				term++
			}
			ag.minFracs = append(ag.minFracs, s.minFrac)
		default:
			ag.despawn++
			spCount++
			ag.minFracs = append(ag.minFracs, s.minFrac)
		}
		if s.nShield > 0 {
			ag.livesWithShield++
		}
		if dump && s.nBody > 0 {
			t.Logf("    vie slot? n=%d minFrac=%.2f lastQ=%d hasZero=%v terminal=%v vie=%.0f..%.0fs Q=%v",
				s.nBody, s.minFrac, s.lastQ, s.hasZero, s.terminal,
				float64(s.firstSeen)/1e6, float64(s.lastSeen)/1e6, v2bTrailSample(s.qTrail))
		}
	}
	t.Logf("V2b %s — %d vies recensees · i4: %d ech / i5: %d ech · DETRUIT %d (terminal %d) · DESPAWN %d · SANS_I4 %d",
		short8, len(stats), nBody, nShield, dCount, term, spCount, nCount)
}

// v2bControlBiped scanne la bande BIPEDE avec la MEME lecture i4 et journalise histogramme des
// quanta + part de pas DECROISSANTS (une vraie sante ne remonte pas). Temoin de non-regression :
// sur le meme film, la sante du JOUEUR (validee) doit etre concentree et monotone.
func v2bControlBiped(t *testing.T, dir, short8 string, entry MapQuantEntry) {
	opt := ScanFilmOptions{RequireTag1: true, DropSaturated: true, CaptureDirs: true, QuantaOnly: true}
	lay := entry.Layout()
	if lay.Valid() {
		opt.Layout = &lay
	}
	pos, err := ScanFilmBipedPositions(dir, opt)
	if err != nil {
		t.Logf("    [CONTROLE bipede] balayage impossible : %v", err)
		return
	}
	var hist [8]int
	n := 0
	bySlot := v2bBodyBySlot(pos)
	down, up := 0, 0
	for _, ps := range bySlot {
		var prev int = -1
		for _, p := range ps {
			if !p.HasBody {
				continue
			}
			hist[int(p.Body.Q)/32]++
			n++
			if prev >= 0 {
				if int(p.Body.Q) < prev {
					down++
				} else if int(p.Body.Q) > prev {
					up++
				}
			}
			prev = int(p.Body.Q)
		}
	}
	t.Logf("    [CONTROLE bipede %s] i4=%d ech · histo Q %v · pas decroissants %d / croissants %d (%.0f %% down)",
		short8, n, hist, down, up, 100*v2bFrac(down, down+up))
}

// v2bMaskDiag partage les echantillons i4 selon que le MASQUE porte i2 ou i3 AVANT i4. Si le
// sous-ensemble PROPRE (ni i2 ni i3) donne des quanta concentres et le sous-ensemble contamine des
// quanta uniformes, la desynchronisation vient de la grammaire d'i2/i3 pour ti=40 (i1 est valide
// par V1a). Zip par index : sous QuantaOnly, les positions sortent dans l'ordre du hook (pas de
// DropTeleports, DropIsolated inerte).
func v2bMaskDiag(t *testing.T, short8 string, pos []BipedPosition, masks [][]int) {
	var clean, dirty [8]int
	nc, nd := 0, 0
	for i, p := range pos {
		if !p.HasBody {
			continue
		}
		has2, has3 := false, false
		for _, id := range masks[i] {
			if id >= 4 {
				break
			}
			if id == 2 {
				has2 = true
			}
			if id == 3 {
				has3 = true
			}
		}
		if has2 || has3 {
			dirty[int(p.Body.Q)/32]++
			nd++
		} else {
			clean[int(p.Body.Q)/32]++
			nc++
		}
	}
	t.Logf("    [MASQUE %s] i4 PROPRE (sans i2/i3 avant) n=%d histo %v", short8, nc, clean)
	t.Logf("    [MASQUE %s] i4 CONTAMINE (i2 ou i3 avant) n=%d histo %v", short8, nd, dirty)
}

// v2bBodyBySlot regroupe les echantillons PORTANT i4 par slot, tries par instant.
func v2bBodyBySlot(pos []BipedPosition) map[uint32][]BipedPosition {
	out := map[uint32][]BipedPosition{}
	for _, p := range pos {
		if !p.HasBody && !p.HasShield {
			continue
		}
		out[p.Slot] = append(out[p.Slot], p)
	}
	for s := range out {
		sort.Slice(out[s], func(i, j int) bool { return out[s][i].TimestampUS < out[s][j].TimestampUS })
	}
	return out
}

// v2bClassify borne chaque vie (slot,gen) par le recensement et classe sa vitalite.
func v2bClassify(kf WorldObjectKeyframes, bySlot map[uint32][]BipedPosition) []v2bLifeStat {
	var out []v2bLifeStat
	for key, seen := range kf.SeenUS {
		if len(seen) == 0 {
			continue
		}
		st := v2bLifeStat{minFrac: 1, minShieldFrac: 1, firstSeen: seen[0], lastSeen: seen[len(seen)-1]}
		st.goneBy = st.lastSeen
		for _, ts := range kf.TimesUS {
			if ts > st.lastSeen {
				st.goneBy = ts
				break
			}
		}
		v2bScanLife(&st, bySlot[key.Slot], seen)
		out = append(out, st)
	}
	return out
}

// v2bScanLife balaye les echantillons du slot dans [firstSeen, goneBy] et remplit la vitalite.
func v2bScanLife(st *v2bLifeStat, samples []BipedPosition, seen []uint64) {
	for _, p := range samples {
		if p.TimestampUS < st.firstSeen || p.TimestampUS > st.goneBy {
			continue
		}
		if h, ok := p.HealthAt(); ok {
			st.nBody++
			st.lastQ = p.Body.Q
			st.qTrail = append(st.qTrail, p.Body.Q)
			if float64(h) < st.minFrac {
				st.minFrac = float64(h)
			}
			if float64(h) <= v2bZeroFrac && !st.hasZero {
				st.hasZero, st.firstZeroUS = true, p.TimestampUS
			}
		}
		if s, ok := p.ShieldAt(); ok {
			st.nShield++
			if float64(s) < st.minShieldFrac {
				st.minShieldFrac = float64(s)
			}
		}
	}
	// terminalite : aucune image-cle ne recense la vie strictement apres firstZeroUS.
	if st.hasZero {
		st.terminal = true
		for _, ts := range seen {
			if ts > st.firstZeroUS {
				st.terminal = false
				break
			}
		}
	}
}

func v2bReport(t *testing.T, mapKey string, ag *v2bMapAgg) {
	t.Logf("\n############## VITALITE i4/i5 — CARTE %q (%d films) ##############", mapKey, ag.films)
	t.Logf("  echantillons i4=%d · i5=%d · vies=%d (avec i5 sur au moins un ech: %d)",
		ag.bodySamples, ag.shieldSamples, ag.lives, ag.livesWithShield)
	if ag.lives == 0 {
		return
	}
	fd := v2bFrac(ag.destroyed, ag.lives)
	fs := v2bFrac(ag.despawn, ag.lives)
	fn := v2bFrac(ag.noI4, ag.lives)
	t.Logf("  DETRUIT (i4->0) : %d/%d = %.0f %% · DESPAWN (i4 sans zero) : %d/%d = %.0f %% · SANS_I4 : %d/%d = %.0f %%",
		ag.destroyed, ag.lives, 100*fd, ag.despawn, ag.lives, 100*fs, ag.noI4, ag.lives, 100*fn)
	decidable := ag.destroyed + ag.despawn
	if decidable > 0 {
		t.Logf("  part DETRUIT parmi les vies DECIDABLES (avec i4) : %d/%d = %.0f %%",
			ag.destroyed, decidable, 100*v2bFrac(ag.destroyed, decidable))
	}
	if ag.destroyed > 0 {
		t.Logf("  TERMINALITE (gate) : %d/%d = %.0f %% des DETRUIT ont un i4->0 TERMINAL (aucune image-cle apres)",
			ag.terminalDestroyed, ag.destroyed, 100*v2bFrac(ag.terminalDestroyed, ag.destroyed))
		sort.Float64s(ag.gapToLastSeenS)
		g := ag.gapToLastSeenS
		t.Logf("  DATATION : (lastSeen - firstZero) en s — mediane %.1f · p10 %.1f · p90 %.1f · min %.1f · max %.1f",
			g[len(g)/2], v2bQuantile(g, 0.10), v2bQuantile(g, 0.90), g[0], g[len(g)-1])
	}
	if len(ag.minFracs) > 0 {
		sort.Float64s(ag.minFracs)
		m := ag.minFracs
		t.Logf("  fraction de vie MIN par vie decidable : mediane %.2f · p10 %.2f (une vie non detruite garde une reserve)",
			m[len(m)/2], v2bQuantile(m, 0.10))
	}
	t.Logf("  DERNIER i4 <= %.2f : %d/%d vies decidables (classement alternatif : la vie FINIT bas)",
		v2bNearFrac, ag.lastQZero, decidable)
	t.Logf("  HISTOGRAMME quanta i4 (buckets de 32, 0..255) : %v", ag.qHist)
	t.Logf("  MONOTONICITE i4 : pas decroissants %d / croissants %d = %.0f %% down (une vraie sante ne remonte pas ; ~50 %% = bruit)",
		ag.downSteps, ag.upSteps, 100*v2bFrac(ag.downSteps, ag.downSteps+ag.upSteps))
}

// v2bTrailSample rend un echantillonnage lisible d'une trajectoire de quanta (debut..fin).
func v2bTrailSample(q []uint8) []uint8 {
	if len(q) <= 12 {
		return q
	}
	out := append([]uint8{}, q[:6]...)
	return append(out, q[len(q)-6:]...)
}

// ------------------------------------------------------------------ helpers

func v2bFrac(a, b int) float64 {
	if b == 0 {
		return 0
	}
	return float64(a) / float64(b)
}

func v2bQuantile(sorted []float64, q float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	i := int(q * float64(len(sorted)-1))
	if i < 0 {
		i = 0
	}
	if i >= len(sorted) {
		i = len(sorted) - 1
	}
	return sorted[i]
}

func v2bParseFilms(t *testing.T) []v2bFilmSpec {
	raw := os.Getenv("V2B_FILMS")
	if raw == "" {
		t.Skipf("V2B_FILMS absent : instrument vitalite saute")
	}
	var out []v2bFilmSpec
	for _, tok := range strings.Split(raw, ",") {
		tok = strings.TrimSpace(tok)
		if tok == "" {
			continue
		}
		i := strings.Index(tok, ":")
		if i < 0 {
			t.Fatalf("V2B_FILMS : entree %q sans ':'", tok)
		}
		out = append(out, v2bFilmSpec{strings.TrimSpace(tok[:i]), strings.TrimSpace(tok[i+1:])})
	}
	if len(out) == 0 {
		t.Skipf("V2B_FILMS vide")
	}
	return out
}

func v2bRoot() string {
	if r := os.Getenv("V2B_FILM_ROOT"); r != "" {
		return r
	}
	return `C:\Users\Guillaume\Projects\LevelUp\data\cache`
}

func v2bLoadBounds(t *testing.T) *MapQuantCatalog {
	path := os.Getenv("V2B_BOUNDS")
	if path == "" {
		path = `C:\Users\Guillaume\Projects\LevelUp\data\titles\halo_infinite\reference\map_quant_bounds.json`
	}
	cat, err := LoadMapQuantCatalog(path)
	if err != nil {
		t.Fatalf("catalogue de bornes illisible : %v", err)
	}
	return cat
}
