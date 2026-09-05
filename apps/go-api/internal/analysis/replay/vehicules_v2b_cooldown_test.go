package replay

// vehicules_v2b_cooldown_test.go — INSTRUMENT DE MESURE (lot V2b, signal 3 : COOLDOWN par la
// METHODE DES SOCLES). LECTURE SEULE, garde par V2B_CD_FILMS.
//
// L'IDEE (insight utilisateur) : le depot a deja resolu le cycle des SOCLES d'arme
// (ground_weapon_pads.go) — le cycle se mesure de la DISPARITION DATEE (le ramassage, un joueur
// passe) a la reapparition, PAS de mesure brute a +/-20 s. Les pads VEHICULES sont fixes (V2 item 1).
// On REUTILISE gwPadsCycleFromGaps (la regle de production, seuil gwPadCycleMaxCV=0,20) et on la
// nourrit d'ECARTS calcules avec le meilleur ancrage DATE disponible.
//
// TROIS ANCRAGES DE FIN DE VIE, compares sur les MEMES pads (ecrits AVANT mesure) :
//   1. NAISSANCE->NAISSANCE : ecart entre naissances successives au meme pad (horloge d'apparition).
//   2. CENSUS goneBy->NAISSANCE : la borne du recensement (+/-20 s) — c'est l'ancrage du lot V2
//      item 3, qui a ECHOUE (IQR/mediane 0,87). Reproduit ici comme temoin negatif.
//   3. DEPART DU PAD->NAISSANCE : dernier echantillon de position du vehicule ENCORE dans le rayon
//      du pad (le pad devient libre), date a ~0,5 s par la trace de position VALIDEE (V1a).
//
// NOTE HONNETE : signal 2 (destruction datee par i4->0) est REFUTE (valeur i4 = bruit pour ti=40,
// cf. vehicules_v2b_vitalite_test.go) — l'ancrage "destruction" n'est donc pas disponible ; le
// depart du pad (ancrage 3) est le meilleur substitut date.
//
// GATE : par pad (>= 2 ecarts), la part de pads dont le cycle est ETABLI (gwPadsCycleFromGaps :
// CV <= 0,20). L'ancrage 3 doit ETABLIR plus de pads que l'ancrage 2 (census), sinon le cooldown
// reste non resolu.
//
// CE QUI EST REUTILISE, sans copie : filmdec.ScanFilmVehicleCreations (naissances + position monde),
// ScanFilmWorldObjectKeyframes (census), ScanFilmBipedPositionsForBand (trace de position),
// indexBySlot/slotTrack (shots.go), gwPadsCycleFromGaps + gwPadCycleMaxCV (ground_weapon_rules.go).
//
// UN SEUL decodage filmdec par process (LockProcessDecode).
//
// USAGE (depuis apps/go-api, cache Go ISOLE) :
//
//	$env:GOCACHE='<scratch>\gocache_v2b'
//	CGO_ENABLED=0 V2B_CD_ROOT=<repo>/data/cache \
//	  V2B_CD_FILMS="0d76e8f1:behemoth,fccc61cd:launch site" \
//	  V2B_CD_BOUNDS=<repo>/data/titles/halo_infinite/reference/map_quant_bounds.json \
//	  go test ./internal/analysis/replay/ -run '^TestV2bCooldown$' -v -timeout 90m

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

const (
	v2cClusterM = 2.0 // rayon d'amas des pads (item 1 : rayon reel 0,00 m, 2 m est large)
	v2cPadRadM  = 4.0 // rayon "sur le pad" pour dater le depart
)

type v2cFilmSpec struct{ short8, mapKey string }

// v2cLife porte une vie de vehicule bornee + datee.
type v2cLife struct {
	slot, gen uint32
	birthUS   uint64
	pos       [3]float64
	goneByUS  uint64 // borne census (premiere image-cle apres lastSeen)
	departUS  uint64 // dernier echantillon de position dans le rayon du pad
	hasDepart bool
}

// v2cMapAgg agrege une carte.
type v2cMapAgg struct {
	films           int
	pads            int
	padsGE2         [3]int // pads avec >= 2 ecarts, par ancrage
	padsEstablished [3]int // pads au cycle etabli (CV <= seuil), par ancrage
	pooledCV        [3][]float64
	establishedMed  [3][]float64 // mediane (s) des cycles ETABLIS, par ancrage
}

var v2cAnchorNames = [3]string{"naissance->naissance", "census goneBy->naissance", "depart pad->naissance"}

func TestV2bCooldown(t *testing.T) {
	films := v2cParseFilms(t)
	root := v2cRoot()
	cat := v2cLoadBounds(t)

	release := filmdec.LockProcessDecode()
	defer release()

	aggs := map[string]*v2cMapAgg{}
	for _, f := range films {
		entry, err := cat.Lookup(f.mapKey)
		if err != nil {
			t.Fatalf("%s : bornes de %q : %v", f.short8, f.mapKey, err)
		}
		ag := aggs[f.mapKey]
		if ag == nil {
			ag = &v2cMapAgg{}
			aggs[f.mapKey] = ag
		}
		dir := filepath.Join(root, "film_chunks", f.short8)
		t.Run(f.short8, func(t *testing.T) { v2cProcessFilm(t, dir, f.short8, entry, ag) })
	}
	keys := make([]string, 0, len(aggs))
	for k := range aggs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		v2cReport(t, k, aggs[k])
	}
}

func v2cProcessFilm(t *testing.T, dir, short8 string, entry filmdec.MapQuantEntry, ag *v2cMapAgg) {
	wr := entry.Range()
	cre, st, err := filmdec.ScanFilmVehicleCreations(dir, &wr)
	if err != nil {
		t.Fatalf("%s : creations : %v", short8, err)
	}
	lives := v2cLivesPerBirth(cre)

	kf := filmdec.ScanFilmWorldObjectKeyframes(dir, filmdec.VehicleTypeIndex)
	v2cAttachCensus(lives, kf)

	opt := filmdec.ScanFilmOptions{WorldRange: &wr, RequireTag1: false, DropSaturated: true}
	if lay := entry.Layout(); lay.Valid() {
		opt.Layout = &lay
	}
	pos, err := filmdec.ScanFilmBipedPositionsForBand(dir, filmdec.NewSlotBand(kf.Band), opt)
	if err != nil {
		t.Fatalf("%s : trace position : %v", short8, err)
	}
	tracks := indexBySlot(pos)

	pads := v2cCluster(lives)
	for _, pad := range pads {
		v2cDatePadDeparture(pad, tracks)
	}
	v2cScorePads(pads, ag)
	ag.films++
	ag.pads += len(pads)
	t.Logf("V2b-cd %s — creations acceptees %d -> %d vies · %d pads", short8, st.Accepted, len(lives), len(pads))
}

// v2cLivesPerBirth dedup une naissance par vie (slot,gen), la plus precoce, en metres monde.
func v2cLivesPerBirth(cre []filmdec.EquipmentCreation) []*v2cLife {
	best := map[[2]uint32]*v2cLife{}
	for _, c := range cre {
		key := [2]uint32{c.Slot, c.Gen}
		if prev, ok := best[key]; ok && prev.birthUS <= c.TimestampUS {
			continue
		}
		best[key] = &v2cLife{slot: c.Slot, gen: c.Gen, birthUS: c.TimestampUS,
			pos: [3]float64{float64(c.X), float64(c.Y), float64(c.Z)}}
	}
	out := make([]*v2cLife, 0, len(best))
	for _, l := range best {
		out = append(out, l)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].birthUS < out[j].birthUS })
	return out
}

// v2cAttachCensus pose goneBy (premiere image-cle apres le dernier recensement) par vie.
func v2cAttachCensus(lives []*v2cLife, kf filmdec.WorldObjectKeyframes) {
	for _, l := range lives {
		seen := kf.SeenUS[filmdec.EquipmentLifeKey{Slot: l.slot, Gen: l.gen}]
		if len(seen) == 0 {
			continue
		}
		last := seen[len(seen)-1]
		l.goneByUS = last
		for _, ts := range kf.TimesUS {
			if ts > last {
				l.goneByUS = ts
				break
			}
		}
	}
}

// v2cDatePadDeparture date, pour chaque vie du pad, le dernier echantillon de position ENCORE dans
// le rayon du pad (le pad devient libre). Ancrage a ~0,5 s (trace de position validee).
func v2cDatePadDeparture(pad *v2cPad, tracks map[uint32]slotTrack) {
	for _, l := range pad.lives {
		tr, ok := tracks[l.slot]
		if !ok {
			continue
		}
		hi := l.goneByUS
		if hi == 0 {
			hi = ^uint64(0)
		}
		for _, p := range tr.pts {
			if p.TimestampUS < l.birthUS || p.TimestampUS > hi || !p.HasWorld {
				continue
			}
			if v2cDist([3]float64{float64(p.X), float64(p.Y), float64(p.Z)}, pad.center) <= v2cPadRadM {
				if p.TimestampUS >= l.departUS {
					l.departUS, l.hasDepart = p.TimestampUS, true
				}
			}
		}
	}
}

// v2cPad est un amas de naissances au meme lieu (le pad).
type v2cPad struct {
	center [3]float64
	lives  []*v2cLife
}

// v2cCluster regroupe les naissances en pads (algorithme du meneur, seuil v2cClusterM).
func v2cCluster(lives []*v2cLife) []*v2cPad {
	var pads []*v2cPad
	for _, l := range lives {
		var host *v2cPad
		for _, p := range pads {
			if v2cDist(l.pos, p.center) <= v2cClusterM {
				host = p
				break
			}
		}
		if host == nil {
			host = &v2cPad{center: l.pos}
			pads = append(pads, host)
		}
		host.lives = append(host.lives, l)
		host.center = v2cCentroid(host.lives)
	}
	return pads
}

// v2cScorePads calcule, par pad et par ancrage, les ecarts et le verdict de cycle.
func v2cScorePads(pads []*v2cPad, ag *v2cMapAgg) {
	for _, pad := range pads {
		ls := append([]*v2cLife{}, pad.lives...)
		sort.Slice(ls, func(i, j int) bool { return ls[i].birthUS < ls[j].birthUS })
		var gaps [3][]float64
		for i := 0; i+1 < len(ls); i++ {
			a, b := ls[i], ls[i+1]
			// 1) naissance->naissance
			gaps[0] = append(gaps[0], float64(b.birthUS-a.birthUS)/1e6)
			// 2) census goneBy->naissance
			if a.goneByUS != 0 && b.birthUS > a.goneByUS {
				gaps[1] = append(gaps[1], float64(b.birthUS-a.goneByUS)/1e6)
			}
			// 3) depart pad->naissance
			if a.hasDepart && b.birthUS > a.departUS {
				gaps[2] = append(gaps[2], float64(b.birthUS-a.departUS)/1e6)
			}
		}
		for k := 0; k < 3; k++ {
			if len(gaps[k]) < 2 {
				continue
			}
			ag.padsGE2[k]++
			c := gwPadsCycleFromGaps(gaps[k])
			if c.Established {
				ag.padsEstablished[k]++
				ag.establishedMed[k] = append(ag.establishedMed[k], c.MedianS)
			}
			if c.MedianS > 0 {
				ag.pooledCV[k] = append(ag.pooledCV[k], c.SDS/c.MedianS)
			}
		}
	}
}

func v2cReport(t *testing.T, mapKey string, ag *v2cMapAgg) {
	t.Logf("\n############## COOLDOWN (methode des socles) — CARTE %q (%d films, %d pads) ##############",
		mapKey, ag.films, ag.pads)
	for k := 0; k < 3; k++ {
		cvMed := 0.0
		if len(ag.pooledCV[k]) > 0 {
			s := append([]float64{}, ag.pooledCV[k]...)
			sort.Float64s(s)
			cvMed = s[len(s)/2]
		}
		cdMed := 0.0
		if len(ag.establishedMed[k]) > 0 {
			s := append([]float64{}, ag.establishedMed[k]...)
			sort.Float64s(s)
			cdMed = s[len(s)/2]
		}
		t.Logf("  ANCRAGE %d (%s) : pads >=2 ecarts %d · cycle ETABLI (CV<=%.2f) %d · CV median inter-pad %.2f · cooldown median des etablis %.0f s",
			k+1, v2cAnchorNames[k], ag.padsGE2[k], gwPadCycleMaxCV, ag.padsEstablished[k], cvMed, cdMed)
	}
	t.Logf("  RAPPEL lot V2 item 3 (ancrage census, brut) : IQR/mediane 0,87 = ECHEC. Gate : l'ancrage 3 doit etablir plus de pads que l'ancrage 2.")
}

// ------------------------------------------------------------------ helpers

// v2cDist : adaptateur d'une ligne vers dist3 (l'unique formule du paquet, geometry.go). Les
// positions viennent de float32 (monde film) ; la reconversion est sans perte a l'echelle du match.
func v2cDist(a, b [3]float64) float64 {
	return dist3([3]float32{float32(a[0]), float32(a[1]), float32(a[2])}, [3]float32{float32(b[0]), float32(b[1]), float32(b[2])})
}

func v2cCentroid(ls []*v2cLife) [3]float64 {
	var c [3]float64
	for _, l := range ls {
		c[0] += l.pos[0]
		c[1] += l.pos[1]
		c[2] += l.pos[2]
	}
	n := float64(len(ls))
	return [3]float64{c[0] / n, c[1] / n, c[2] / n}
}

func v2cParseFilms(t *testing.T) []v2cFilmSpec {
	raw := os.Getenv("V2B_CD_FILMS")
	if raw == "" {
		t.Skipf("V2B_CD_FILMS absent : instrument cooldown saute")
	}
	var out []v2cFilmSpec
	for _, tok := range strings.Split(raw, ",") {
		tok = strings.TrimSpace(tok)
		if tok == "" {
			continue
		}
		i := strings.Index(tok, ":")
		if i < 0 {
			t.Fatalf("V2B_CD_FILMS : entree %q sans ':'", tok)
		}
		out = append(out, v2cFilmSpec{strings.TrimSpace(tok[:i]), strings.TrimSpace(tok[i+1:])})
	}
	if len(out) == 0 {
		t.Skipf("V2B_CD_FILMS vide")
	}
	return out
}

func v2cRoot() string {
	if r := os.Getenv("V2B_CD_ROOT"); r != "" {
		return r
	}
	return `C:\Users\Guillaume\Projects\LevelUp\data\cache`
}

func v2cLoadBounds(t *testing.T) *filmdec.MapQuantCatalog {
	path := os.Getenv("V2B_CD_BOUNDS")
	if path == "" {
		path = `C:\Users\Guillaume\Projects\LevelUp\data\titles\halo_infinite\reference\map_quant_bounds.json`
	}
	cat, err := filmdec.LoadMapQuantCatalog(path)
	if err != nil {
		t.Fatalf("catalogue de bornes illisible : %v", err)
	}
	return cat
}
