package filmdec

// vitesse_filtre_research_test.go — R3 : que coûte le filtre MaxSpeedMPS=100 de la
// production sur les téléportations du translocateur, et que vaudrait son remplacement ?
// Plan : .ai/PLAN_LECTURE_FIABLE_EQUIPEMENT_2026-09-03.md, lot R3.
//
// CE QUE LA PRODUCTION FAIT (offline_biped.go / offline_filters.go, lu sur pièces) :
// DropTeleports rejette toute position dont la vitesse depuis la DERNIÈRE POSITION ACCEPTÉE
// du même slot dépasse DefaultMaxSpeedMPS=100 — une téléportation (200-400 m/s) est donc
// rejetée. MAIS le rejet en cascade est BORNÉ : après maxRejectStreak=3 rejets consécutifs,
// le filtre se RÉANCRE AVEUGLÉMENT sur l'échantillon suivant. Le coût par téléportation est
// donc au plus 3 échantillons bruts, pas la fin de la vie. C'est ce coût que R3.1 mesure,
// contre l'artefact publié ET contre le film décodé sans filtre.
//
// LA MESURE (par événement 117 EquipmentTranslocatorTeleportEffects, découvert en R1 —
// RAPPORT_R1_FAILLE_ACTIVATION_2026-09-03.md §4 : l'événement date la téléportation, 5 à
// 80 ms avant la discontinuité, ref0 = slot du bipède) :
//   1. recensement des têtes 117 du film ENTIER (lecture O(1) par paquet, aucun décodage) ;
//   2. décodage SANS filtre (MaxSpeedMPS=0, IsolationGapMS=0) des SEULS chunks couvrant
//      [événement-5 s, événement+10 s] — jamais un film entier sans filtre (le nuage tue le
//      process, mesure du 2026-08-18, cf. translocateur_test.go) ;
//   3. rejeu de la sémantique EXACTE de DropTeleports décision par décision (vitfSimuler),
//      plus deux variantes : option A (filtre levé à ±200 ms d'un événement 117 du même
//      slot) et option B (profondeur de corroboration : combien d'échantillons suivants
//      s'enchaînent depuis le point rejeté — le k du réancrage se MESURE ici) ;
//   4. lecture de l'ARTEFACT PUBLIÉ (.tracks[], VITESSE_ARTEFACT) : retard de la première
//      position publiée au lieu d'arrivée, trous de frames autour du saut, calibration.
//
// DÉQUANTIFICATION — LE PIÈGE DÉCOUVERT À LA PREMIÈRE EXÉCUTION : le champ `bounds` de
// l'artefact est un CADRAGE D'AFFICHAGE (l'étendue des pistes), PAS l'AABB de
// déquantification de la carte. Déquantifier avec lui compresse tout d'un facteur ~10
// (mesuré sur Dynasty : saut de 22,1 m rendu 2,05 m). L'instrument lit donc l'entrée de
// catalogue de la production (VITESSE_CATALOGUE = map_quant_bounds.json). La carte est soit
// donnée (VITESSE_CARTE), soit IDENTIFIÉE par calibration : parmi les entrées du catalogue
// dont le découpage égale celui lu dans le film, celle qui minimise l'écart médian entre
// les positions déquantifiées et la piste publiée du slot avant le saut. L'identification
// est rapportée avec son score — un score > 1 m invaliderait la mesure.
//
// LIMITES ASSUMÉES (dites, pas cachées) : la simulation opère sur la fenêtre, pas sur le
// film entier — l'ancre de chaque slot à l'entrée de fenêtre peut différer de celle de la
// production (5 s de trajectoire dense l'égalisent avant l'événement) ; DropIsolated sur la
// fenêtre peut écarter un échantillon de bord que la production aurait gardé (loin de la
// zone d'arrivée). « Rejeté à tort » = rejeté par la production ET corroboré par la suite
// de la trajectoire (>= 2 échantillons suivants s'enchaînent sous 100 m/s depuis le point).
//
// LECTURE SEULE, gardé par VITESSE_FILM, CGO_ENABLED=0, un seul décodage par process.
//
// USAGE (depuis apps/go-api) :
//
//	CGO_ENABLED=0 VITESSE_FILM=<repo>/data/cache/film_chunks/1b2d9e08 \
//	  VITESSE_CATALOGUE=<repo>/data/titles/halo_infinite/reference/map_quant_bounds.json \
//	  VITESSE_ARTEFACT=<repo>/data/cache/replays/halo_infinite/1b2d9e08.json \
//	  go test ./internal/analysis/filmdec/ -run '^TestVitesseFiltre$' -timeout 30m -v
//
// Témoin SANS translocateur (option A prouvée par le recensement ; option B mesurée sur une
// fenêtre bornée par VITESSE_CHUNKS ; la carte doit être donnée, l'identification par
// artefact exige des événements) :
//
//	CGO_ENABLED=0 VITESSE_FILM=<repo>/data/cache/film_chunks/696a9d7c \
//	  VITESSE_CATALOGUE=<...>/map_quant_bounds.json VITESSE_CARTE=vagabond \
//	  VITESSE_CHUNKS=8,9,10 \
//	  go test ./internal/analysis/filmdec/ -run '^TestVitesseFiltre$' -timeout 30m -v

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"
)

const (
	vitfFilmEnv      = "VITESSE_FILM"
	vitfCatalogueEnv = "VITESSE_CATALOGUE"
	vitfCarteEnv     = "VITESSE_CARTE"
	vitfArtefactEnv  = "VITESSE_ARTEFACT"
	vitfChunksEnv    = "VITESSE_CHUNKS"
)

const (
	// vitfFenAvantUS / vitfFenApresUS : la fenêtre de décodage autour d'un événement —
	// 5 s avant (l'ancre du slot se stabilise sur de la trajectoire dense), 10 s après.
	vitfFenAvantUS = 5_000_000
	vitfFenApresUS = 10_000_000
	// vitfZoneAvantUS / vitfZoneApresUS : la « zone d'arrivée » d'un événement, où un rejet
	// de production est confronté à la corroboration (l'arrivée et sa cascade y tombent).
	vitfZoneAvantUS = 500_000
	vitfZoneApresUS = 2_000_000
	// vitfExemptionUS : la fenêtre de l'option A — ±200 ms autour d'un événement 117 du
	// même slot, valeur du plan (l'événement précède le saut de 5 à 80 ms, R1 §4.1).
	vitfExemptionUS = 200_000
	// vitfCorrobMax : plafond de la profondeur de corroboration mesurée (option B).
	vitfCorrobMax = 8
	// vitfArriveeM : rayon « au lieu d'arrivée » pour l'artefact — le même 2 m que
	// translocArriveM (translocateur_test.go), pour comparer les mesures entre elles.
	vitfArriveeM = 2.0
)

// vitfMesure est la ligne de la table R3.1 pour UNE téléportation datée par l'événement.
type vitfMesure struct {
	ev     vitfEvent
	filmMS int64
	nEch   int
	// cadenceMS : dt médian entre échantillons bruts du slot dans la fenêtre.
	cadenceMS float64
	// saut/dtSautMS/vSaut : la discontinuité (plus grand pas consécutif arrivant dans
	// [événement-100 ms, +600 ms]) ; tArr/posArr son point d'arrivée, posAvant le départ.
	saut     float64
	dtSautMS float64
	vSaut    float64
	tArr     uint64
	posArr   [3]float32
	posAvant [3]float32
	// rejets de la production dans la zone d'arrivée : total, corroborés (à tort), dont
	// datés à ±200 ms de l'événement (couverture du POINT par l'option A) ; et rejets du
	// slot HORS zone dans la fenêtre (les aberrations que le filtre vise vraiment).
	rejZone         int
	rejTort         int
	rejTortA200     int
	rejSlotHorsZone int
	// retardProdMS : premier échantillon ACCEPTÉ par la production au nouveau lieu, en ms
	// après l'arrivée réelle (-1 = jamais) ; deplProdM sa distance à l'arrivée réelle.
	retardProdMS int64
	deplProdM    float64
	// profArr : profondeur de corroboration de l'échantillon d'arrivée (le k mesurable).
	profArr int
	// rejZoneA / exemptesTort : la même zone sous l'option A, et les exemptions accordées
	// par A à des points NON corroborés (le risque de l'option A).
	rejZoneA     int
	exemptesTort int
	// artefact publié : frame de l'événement, retard de la première position publiée à
	// <= 2 m de l'arrivée (-1 = jamais), pas max et son retard, trous de frames à ±10,
	// distance du 1er point publié à l'arrivée réelle, calibration piste/film avant saut.
	artOK        bool
	frameEv      int
	retardArrFr  int
	sautArtM     float64
	retardSautFr int
	trous        int
	trousListe   []int
	deplArtM     float64
	calibM       float64
}

func TestVitesseFiltre(t *testing.T) {
	dir := os.Getenv(vitfFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument de mesure sauté", vitfFilmEnv)
	}
	release := LockProcessDecode()
	defer release()
	ctx := vitfSetup(t, dir)
	evs, spans, deltas := vitfScanEvenements(t, dir)
	t.Logf("== FILM %s : %d paquets delta · %d têtes de type 117 ==", dir, deltas, len(evs))
	for _, ev := range evs {
		t.Logf("  événement 117 @%d ms (film) · slot %d · chunk %d paquet %d",
			(int64(ev.ts)-int64(ctx.origine))/1000, ev.slot, ev.chunk, ev.paquet)
	}
	vitfChoisirLayout(t, ctx)
	if len(evs) == 0 {
		t.Log("AUCUN événement 117 : film témoin — l'option A n'y change RIEN par construction (le filtre n'y est jamais levé).")
		vitfTemoin(t, ctx)
		return
	}
	var mesures []vitfMesure
	bruit := vitfBruit{origine: ctx.origine}
	for _, g := range vitfGroupes(evs, spans) {
		t.Logf("== GROUPE de %d événement(s) · chunks %v ==", len(g.evs), g.chunks)
		qs := vitfDecodeQuanta(t, ctx.dir, &ctx.lay, g.chunks)
		if ctx.entree == nil {
			vitfChoisirEntree(t, ctx, qs, g)
		}
		samples := vitfDequantTous(qs, ctx.lay, ctx.entree.Range())
		samples = DropIsolated(samples, DefaultIsolationGapMS)
		prod := vitfSimuler(samples, vitfSimOpts{maxSpeed: DefaultMaxSpeedMPS})
		optA := vitfSimuler(samples, vitfSimOpts{maxSpeed: DefaultMaxSpeedMPS, exemption: g.evs})
		for _, ev := range g.evs {
			m := vitfAnalyser(ev, samples, prod, optA, ctx.origine)
			if ctx.art != nil {
				vitfMesureArtefact(t, ctx.art, &m, ctx.origine)
			}
			mesures = append(mesures, m)
			vitfLigne(t, &m)
		}
		// Zones d'exclusion : les événements du FILM ENTIER — une arrivée d'un autre
		// groupe n'est pas du bruit (les groupes ne partagent plus aucun chunk).
		vitfCompterBruit(&bruit, samples, prod, evs)
	}
	vitfAgregats(t, mesures)
	vitfRapportBruit(t, &bruit, "fenêtres de ce film, hors zones d'arrivée")
}

// vitfChoisirLayout arrête la grammaire de décodage : celle du CATALOGUE quand la carte est
// donnée (l'entrée de la production — le découpage lu dans le film reste le contrôle), la
// détection sinon (l'identification de carte n'acceptera que des entrées au même découpage).
func vitfChoisirLayout(t *testing.T, ctx *vitfCtx) {
	t.Helper()
	detecte, _, errDet := DetectI0Layout(ctx.dir)
	if ctx.entree != nil {
		ctx.lay = ctx.entree.Layout()
		if errDet == nil && detecte != ctx.lay {
			t.Logf("  CONTRÔLE : découpage détecté %+v != catalogue %+v — le catalogue reste l'entrée", detecte, ctx.lay)
		}
		return
	}
	if errDet != nil {
		t.Fatalf("découpage i0 illisible dans %s (et %s absent) : %v", ctx.dir, vitfCarteEnv, errDet)
	}
	ctx.lay = detecte
}

// vitfAnalyser mesure UNE téléportation contre les échantillons bruts de son groupe et les
// deux passes de filtre (production et option A) déjà simulées sur ces échantillons.
func vitfAnalyser(ev vitfEvent, samples []BipedPosition, prod, optA vitfSim, origine uint64) vitfMesure {
	m := vitfMesure{ev: ev, filmMS: (int64(ev.ts) - int64(origine)) / 1000, retardProdMS: -1, retardArrFr: -1}
	lo := uint64(0)
	if ev.ts > vitfFenAvantUS {
		lo = ev.ts - vitfFenAvantUS
	}
	hi := ev.ts + vitfFenApresUS
	var idx []int
	for i, p := range samples {
		if p.Slot == ev.slot && p.TimestampUS >= lo && p.TimestampUS <= hi {
			idx = append(idx, i)
		}
	}
	sort.Slice(idx, func(a, b int) bool { return samples[idx[a]].TimestampUS < samples[idx[b]].TimestampUS })
	m.nEch = len(idx)
	if len(idx) < 2 {
		return m
	}
	m.cadenceMS = vitfCadence(samples, idx)
	// La discontinuité : le plus grand pas consécutif dont l'ARRIVÉE tombe dans
	// [événement-100 ms, +600 ms] — l'événement précède le saut de 5 à 80 ms (R1 §4.1),
	// la marge basse tolère l'imprécision d'horodatage de paquet.
	iArr := -1
	for k := 1; k < len(idx); k++ {
		p0, p1 := samples[idx[k-1]], samples[idx[k]]
		d := int64(p1.TimestampUS) - int64(ev.ts)
		if d < -100_000 || d > 600_000 {
			continue
		}
		if dd := translocDist(vitfPos(p0), vitfPos(p1)); dd > m.saut {
			m.saut, m.tArr = dd, p1.TimestampUS
			m.dtSautMS = float64(p1.TimestampUS-p0.TimestampUS) / 1000
			m.posArr, m.posAvant = vitfPos(p1), vitfPos(p0)
			iArr = k
		}
	}
	if iArr < 0 {
		return m
	}
	if m.dtSautMS > 0 {
		m.vSaut = m.saut / (m.dtSautMS / 1000)
	}
	m.profArr = vitfProfondeur(samples, idx, idx[iArr], DefaultMaxSpeedMPS)
	vitfRejets(&m, samples, idx, prod, optA)
	return m
}

// vitfRejets compte, pour la mesure m, les décisions de rejet des deux passes sur les
// échantillons du slot, et localise le premier point accepté par la production au nouveau
// lieu (critère : à plus de saut/2 du point de départ, à partir de l'arrivée réelle).
func vitfRejets(m *vitfMesure, samples []BipedPosition, idx []int, prod, optA vitfSim) {
	zLo, zHi := int64(m.ev.ts)-vitfZoneAvantUS, int64(m.ev.ts)+vitfZoneApresUS
	for _, i := range idx {
		p := samples[i]
		ts := int64(p.TimestampUS)
		dansZone := ts >= zLo && ts <= zHi
		if !prod.accepte[i] {
			if !dansZone {
				m.rejSlotHorsZone++
			} else {
				m.rejZone++
				if vitfProfondeur(samples, idx, i, DefaultMaxSpeedMPS) >= 2 {
					m.rejTort++
					if d := ts - int64(m.ev.ts); d >= -vitfExemptionUS && d <= vitfExemptionUS {
						m.rejTortA200++
					}
				}
			}
		}
		if dansZone && !optA.accepte[i] {
			m.rejZoneA++
		}
		if optA.motif[i] == 'e' && vitfProfondeur(samples, idx, i, DefaultMaxSpeedMPS) < 2 {
			m.exemptesTort++
		}
		if m.retardProdMS < 0 && prod.accepte[i] && p.TimestampUS >= m.tArr &&
			translocDist(vitfPos(p), m.posAvant) > m.saut/2 {
			m.retardProdMS = (ts - int64(m.tArr)) / 1000
			m.deplProdM = translocDist(vitfPos(p), m.posArr)
		}
	}
}

// vitfLigne imprime la ligne de table d'une téléportation.
func vitfLigne(t *testing.T, m *vitfMesure) {
	t.Helper()
	t.Logf("-- @%d ms slot %d : %d échantillons bruts (cadence médiane %.0f ms)",
		m.filmMS, m.ev.slot, m.nEch, m.cadenceMS)
	if m.saut == 0 {
		t.Log("   AUCUNE discontinuité mesurable dans [-100,+600] ms de l'événement")
		return
	}
	t.Logf("   saut %.2f m en %.0f ms (%.0f m/s) · arrivée %+d ms après l'événement",
		m.saut, m.dtSautMS, m.vSaut, (int64(m.tArr)-int64(m.ev.ts))/1000)
	t.Logf("   film : rejets zone %d (à tort %d, dont ±200 ms %d) · rejets slot hors zone %d · 1er point prod %s (dépl. %.2f m) · profondeur corrob. arrivée %d",
		m.rejZone, m.rejTort, m.rejTortA200, m.rejSlotHorsZone, vitfMSTxt(m.retardProdMS), m.deplProdM, m.profArr)
	if m.artOK {
		t.Logf("   artefact : frame évén. %d · arrivée publiée %s · pas max %.2f m @+%d fr · trous de frames à ±10 : %d %v · dépl. 1er point publié %.2f m · calibration %.2f m",
			m.frameEv, vitfFrTxt(m.retardArrFr), m.sautArtM, m.retardSautFr, m.trous, m.trousListe, m.deplArtM, m.calibM)
	}
	t.Logf("   option A : rejets zone restants %d · exemptions non corroborées %d", m.rejZoneA, m.exemptesTort)
}

// vitfAgregats imprime les agrégats R3.1 et la comparaison chiffrée A/B de R3.2.
func vitfAgregats(t *testing.T, ms []vitfMesure) {
	t.Helper()
	var retArt, rejTort, retProd []float64
	tortTotal, tort200, resteA, exTort, sansSaut, rejetees := 0, 0, 0, 0, 0, 0
	profMin := vitfCorrobMax + 1
	for _, m := range ms {
		if m.saut == 0 {
			sansSaut++
			continue
		}
		if m.artOK && m.retardArrFr >= 0 {
			retArt = append(retArt, float64(m.retardArrFr))
		}
		rejTort = append(rejTort, float64(m.rejTort))
		if m.retardProdMS >= 0 {
			retProd = append(retProd, float64(m.retardProdMS))
		}
		tortTotal += m.rejTort
		tort200 += m.rejTortA200
		resteA += m.rejZoneA
		exTort += m.exemptesTort
		if m.rejZone > 0 {
			rejetees++
			if m.profArr < profMin {
				profMin = m.profArr
			}
		}
	}
	t.Logf("== AGRÉGATS (%d téléportations datées, %d sans discontinuité mesurable) ==", len(ms), sansSaut)
	t.Logf("  retard artefact au lieu d'arrivée (frames) : médiane %.1f · pire %.0f (sur %d mesurables)",
		vitfMediane(retArt), vitfMax(retArt), len(retArt))
	t.Logf("  rejets à tort par téléportation : médiane %.1f · pire %.0f · total %d (dont %d à ±200 ms de l'événement)",
		vitfMediane(rejTort), vitfMax(rejTort), tortTotal, tort200)
	t.Logf("  retard du 1er point accepté par la production (ms après l'arrivée réelle) : médiane %.1f · pire %.0f",
		vitfMediane(retProd), vitfMax(retProd))
	t.Logf("== OPTION A (filtre levé à ±%d ms d'un événement 117 du même slot) : rejets zone restants %d (production : %d à tort) · exemptions non corroborées %d ==",
		vitfExemptionUS/1000, resteA, tortTotal, exTort)
	if rejetees == 0 {
		t.Log("== OPTION B : aucune arrivée rejetée par la production dans ces fenêtres — rien à récupérer ==")
	} else {
		t.Logf("== OPTION B : %d téléportations à rejets · profondeur de corroboration minimale de l'arrivée %d — tout k <= %d les récupère toutes (cf. bruit ci-dessous pour le prix) ==",
			rejetees, profMin, profMin)
	}
}

// vitfTemoin : sur un film SANS événement 117, mesurer ce que l'option B changerait —
// chaque rejet de production réancré par corroboration serait un point publié EN PLUS sur
// un film sans translocateur. L'option A, elle, n'y change rien par construction.
func vitfTemoin(t *testing.T, ctx *vitfCtx) {
	t.Helper()
	chunks := vitfChunks(t)
	if len(chunks) == 0 {
		t.Logf("%s absent : pas de mesure option B sur ce témoin (le recensement 117 ci-dessus prouve déjà l'option A)", vitfChunksEnv)
		return
	}
	if ctx.entree == nil {
		t.Fatalf("film sans événement : %s obligatoire (l'identification par artefact exige des événements)", vitfCarteEnv)
	}
	qs := vitfDecodeQuanta(t, ctx.dir, &ctx.lay, chunks)
	samples := vitfDequantTous(qs, ctx.lay, ctx.entree.Range())
	samples = DropIsolated(samples, DefaultIsolationGapMS)
	prod := vitfSimuler(samples, vitfSimOpts{maxSpeed: DefaultMaxSpeedMPS})
	b := vitfBruit{origine: ctx.origine}
	vitfCompterBruit(&b, samples, prod, nil)
	vitfRapportBruit(t, &b, "témoin sans translocateur, chunks bornés")
}

// vitfRapportBruit : les rejets de production HORS zones d'arrivée (les aberrations que le
// filtre vise), et combien l'option B en réancrerait pour chaque k.
func vitfRapportBruit(t *testing.T, b *vitfBruit, titre string) {
	t.Helper()
	t.Logf("== BRUIT (%s) : %d échantillons · %d rejets production · %d hors zone d'arrivée ==",
		titre, b.ech, b.rejets, b.horsZone)
	for k := 1; k <= vitfCorrobMax; k++ {
		n := 0
		for d := k; d <= vitfCorrobMax; d++ {
			n += b.prof[d]
		}
		t.Logf("  option B k=%d : %d rejet(s) hors zone seraient réancrés (profondeur >= %d)", k, n, k)
	}
	for _, ex := range b.exemples {
		t.Logf("  rejet hors zone CORROBORÉ (l'option B le publierait) : %s", ex)
	}
}

func vitfMSTxt(v int64) string {
	if v < 0 {
		return "JAMAIS (aucun point accepté au nouveau lieu)"
	}
	return fmt.Sprintf("+%d ms", v)
}

func vitfFrTxt(v int) string {
	if v < 0 {
		return fmt.Sprintf("JAMAIS à <= %.0f m de l'arrivée", vitfArriveeM)
	}
	return fmt.Sprintf("+%d frame(s)", v)
}

func vitfPos(p BipedPosition) [3]float32 { return [3]float32{p.X, p.Y, p.Z} }

func vitfCadence(samples []BipedPosition, idx []int) float64 {
	var dts []float64
	for k := 1; k < len(idx); k++ {
		dts = append(dts, float64(samples[idx[k]].TimestampUS-samples[idx[k-1]].TimestampUS)/1000)
	}
	return vitfMediane(dts)
}

func vitfMediane(v []float64) float64 {
	if len(v) == 0 {
		return 0
	}
	s := append([]float64(nil), v...)
	sort.Float64s(s)
	return s[len(s)/2]
}

func vitfMax(v []float64) float64 {
	m := 0.0
	for _, x := range v {
		if x > m {
			m = x
		}
	}
	return m
}

// vitfChunks lit VITESSE_CHUNKS (fenêtre bornée du témoin), vide = absent.
func vitfChunks(t *testing.T) []int {
	t.Helper()
	raw := strings.TrimSpace(os.Getenv(vitfChunksEnv))
	if raw == "" {
		return nil
	}
	var out []int
	for _, p := range strings.Split(raw, ",") {
		n, err := strconv.Atoi(strings.TrimSpace(p))
		if err != nil {
			t.Fatalf("%s=%q : %q n'est pas un numéro de chunk", vitfChunksEnv, raw, p)
		}
		out = append(out, n)
	}
	return out
}
