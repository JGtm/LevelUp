package filmdec

// world_object_precision_test.go — INSTRUMENT DE MESURE de la PRÉCISION DU CHEMIN
// WORLD-OBJECT (`object-position-component` : projectiles ti=41, équipement ti=37, armes au
// sol ti=42, corps rigides ti=38).
//
// LA QUESTION POSÉE. `WorldObjectPrecision` (traverse.go) est un GLOBAL de paquet dont le
// défaut `{13,13,14}` EST l'entrée `cliffhanger`/`ridgeline` du catalogue de bornes
// (`map_bounds_test.go` le vérifie), et aucun chemin de production ne l'installe depuis la
// carte du match. Les largeurs DU CATALOGUE — celles-là mêmes qui fournissent déjà
// `opt.WorldRange` à `BuildFromFilm` — font-elles MIEUX ? Qu'elles DIFFÈRENT ne le prouve
// pas : `AxisWidths` est DÉDUIT des bornes par la loi du moteur, et l'hypothèse « le chemin
// world-object emploie les largeurs de la carte, les mêmes que l'absolu du bipède » est
// ÉCRITE dans traverse.go sans avoir jamais été vérifiée par ses conséquences.
//
// LE CRITÈRE, ÉNONCÉ AVANT LA MESURE, et repris TEL QUEL du lot d4be4ab95 (97,2 % sur ti=37,
// 12 films) : la part d'échantillons qui tombe dans l'EMPRISE DU NUAGE DES BIPÈDES DU MÊME
// FILM, en coordonnées NORMALISÉES [0,1] de l'AABB (`equipBipedBox`, `equipBox.contains`).
// Non circulaire — le nuage de référence vient d'un canal indépendant, l'absolu du bipède —
// et SANS BASE : aucune borne de carte n'est nécessaire, les deux nuages sont exprimés dans
// le même repère quantifié, lu dans le film.
//
// LE CONTRÔLE DE COHÉRENCE, joué à chaque film : `DetectI0Layout` lit le découpage d'i0 DANS
// le bitstream. C'est un CONTRÔLE, jamais une entrée — le commentaire d'`AxisWidths` le dit
// déjà. Un désaccord avec le catalogue signalerait que ce sont les BORNES qui sont fausses.
//
// CE QUE LE CRITÈRE NE PEUT PAS DIRE, dit d'avance : l'emprise est une BOÎTE. Un film dont
// les bipèdes couvrent presque tout [0,1] sur un axe ne discrimine rien sur cet axe, et un
// taux élevé n'y vaut pas certificat de justesse. Le dénominateur et l'emprise sont donc
// publiés à côté du taux, jamais le taux seul.
//
// UNE CARTE HORS CATALOGUE NE SE MESURE PAS COMME SI ELLE Y ÉTAIT : l'instrument le dit et
// s'arrête sur ce film, il ne se rabat pas sur le découpage lu dans le film.
//
// APRÈS LE CORRECTIF DE LA PHASE 2, LES DEUX COLONNES DEVIENNENT IDENTIQUES : le chemin de
// production installe alors lui-même les largeurs de l'entrée de catalogue, et la colonne
// « défaut » n'existe plus. C'est la signature du branchement, pas une panne de l'instrument.
//
// LECTURE SEULE, gardé par WORLDPREC_FILM, sauté partout ailleurs (CI comprise). UN SEUL
// décodage filmdec par process (globaux de paquet) : le verrou est pris pour tout le test.
//
// USAGE (depuis apps/go-api) :
//
//	CGO_ENABLED=0 WORLDPREC_FILM=<repo>/data/cache/film_chunks/000d5950 \
//	  WORLDPREC_BOUNDS=<repo>/data/titles/halo_infinite/reference/map_quant_bounds.json \
//	  WORLDPREC_MAP=Cliffhanger \
//	  go test ./internal/analysis/filmdec/ -run '^TestWorldObjectPrecisionImpact$' -timeout 60m -v
//
//	CGO_ENABLED=0 WORLDPREC_FILM=... \
//	  go test ./internal/analysis/filmdec/ -run '^TestWorldObjectPrecisionLayout$' -timeout 10m -v

import (
	"fmt"
	"os"
	"testing"
)

const (
	worldPrecFilmEnv   = "WORLDPREC_FILM"
	worldPrecBoundsEnv = "WORLDPREC_BOUNDS"
	worldPrecMapEnv    = "WORLDPREC_MAP"
)

// worldPrecArchetypes : les archétypes mesurés. ti=41 EST le chemin de production
// (`BuildFromFilm` -> `ScanFilmProjectiles`) ; ti=37 est celui dont le taux de référence
// (97,2 %) est publié, il sert d'ancrage comparable.
var worldPrecArchetypes = []int{ProjectileTypeIndex, EquipmentTypeIndex}

// TestWorldObjectPrecisionLayout est la SONDE DE CORPUS : elle publie le découpage d'i0 lu
// dans le film, sans rien décoder d'autre. Coût borné (6 chunks) — c'est elle qui sert à
// choisir des films couvrant plusieurs valeurs de largeurs, et à jouer le contrôle de
// cohérence sans payer un balayage complet.
func TestWorldObjectPrecisionLayout(t *testing.T) {
	dir := os.Getenv(worldPrecFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument de mesure sauté", worldPrecFilmEnv)
	}
	release := LockProcessDecode()
	defer release()

	lay, rep, err := DetectI0Layout(dir)
	if err != nil {
		t.Logf("LAYOUT %s : NON DÉTECTÉ (%v) · %d records · %d paires · frontières %v",
			dir, err, rep.Records, rep.Pairs, rep.Boundaries)
		return
	}
	t.Logf("LAYOUT %s : %v · %s · %d records · %d paires · frontières %v",
		dir, lay.AxisW, lay, rep.Records, rep.Pairs, rep.Boundaries)
}

// TestWorldObjectPrecisionImpact balaie DEUX FOIS chaque archétype d'objet du monde — aux
// largeurs par DÉFAUT du paquet, puis aux largeurs DU CATALOGUE de la carte du match — et
// confronte les deux nuages au même nuage de bipèdes.
func TestWorldObjectPrecisionImpact(t *testing.T) {
	dir := os.Getenv(worldPrecFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument de mesure sauté", worldPrecFilmEnv)
	}
	release := LockProcessDecode()
	defer release()

	prev := WorldObjectPrecision
	t.Cleanup(func() { WorldObjectPrecision = prev })

	t.Logf("== FILM %s ==", dir)
	entry, ok := worldPrecCatalogEntry(t)
	lay := worldPrecCoherence(t, dir, entry, ok)
	if !ok {
		t.Log("  CARTE HORS CATALOGUE : ce film est compté à part et N'EST PAS mesuré — se")
		t.Log("  rabattre sur le découpage lu dans le film ferait passer un contrôle pour une")
		t.Log("  entrée, ce que la décision n°1 du plan interdit.")
		return
	}
	t.Logf("  largeurs par DÉFAUT du paquet : %v · largeurs DU CATALOGUE : %v (module %s)",
		prev.AxisW, entry.AxisWidths, entry.Module)

	bip := equipBipedBox(t, dir, lay)
	if bip.n == 0 {
		t.Fatalf("aucun nuage de bipède dans %s : le critère n'est pas calculable", dir)
	}
	t.Logf("  nuage des BIPÈDES du même film : %d positions · emprise %s", bip.n, bip)

	for _, ti := range worldPrecArchetypes {
		def := worldPrecRun(t, dir, ti, prev.AxisW, bip)
		got := worldPrecRun(t, dir, ti, entry.AxisWidths, bip)
		t.Logf("== ARCHÉTYPE ti=%d ==", ti)
		worldPrecReport(t, fmt.Sprintf("DÉFAUT    %v", prev.AxisW), def)
		worldPrecReport(t, fmt.Sprintf("CATALOGUE %v", entry.AxisWidths), got)
		worldPrecDelta(t, def, got)
	}
}

// worldPrecCatalogEntry résout l'entrée de catalogue de la carte du match. C'est la MÊME
// entrée qui fournit `opt.WorldRange` à `BuildFromFilm` : les bornes et les largeurs ne se
// dissocient pas.
func worldPrecCatalogEntry(t *testing.T) (MapQuantEntry, bool) {
	t.Helper()
	boundsPath, mapName := os.Getenv(worldPrecBoundsEnv), os.Getenv(worldPrecMapEnv)
	if boundsPath == "" || mapName == "" {
		t.Skipf("%s / %s absents : la source des largeurs est le CATALOGUE, pas le film",
			worldPrecBoundsEnv, worldPrecMapEnv)
	}
	cat, err := LoadMapQuantCatalog(boundsPath)
	if err != nil {
		t.Fatalf("catalogue de bornes %s : %v", boundsPath, err)
	}
	entry, err := cat.Lookup(mapName)
	if err != nil {
		t.Logf("  carte %q ABSENTE du catalogue : %v", mapName, err)
		return MapQuantEntry{}, false
	}
	t.Logf("  carte %q -> module %s · bornes X[%.2f %.2f] Y[%.2f %.2f] Z[%.2f %.2f]",
		mapName, entry.Module, entry.Min[0], entry.Max[0], entry.Min[1], entry.Max[1],
		entry.Min[2], entry.Max[2])
	return entry, true
}

// worldPrecCoherence joue le CONTRÔLE : le découpage lu dans le bitstream doit égaler les
// largeurs déduites des bornes. Un désaccord dit que les BORNES sont fausses — il se logge,
// il ne se contourne pas. Rend le découpage lu, qui sert à normaliser le nuage de bipèdes
// (c'est celui sous lequel `ScanFilmBipedPositions` produit ses quanta).
func worldPrecCoherence(t *testing.T, dir string, entry MapQuantEntry, ok bool) I0Layout {
	t.Helper()
	lay, _, err := DetectI0Layout(dir)
	if err != nil {
		t.Fatalf("découpage i0 illisible dans %s : %v", dir, err)
	}
	if !ok {
		t.Logf("  CONTRÔLE non jouable (carte hors catalogue) · découpage lu dans le film : %v",
			lay.AxisW)
		return lay
	}
	verdict := "ACCORD"
	if lay.AxisW != entry.AxisWidths {
		verdict = "DÉSACCORD — les BORNES du catalogue sont suspectes, pas le découpage"
	}
	t.Logf("  CONTRÔLE catalogue %v <-> découpage lu dans le film %v -> %s",
		entry.AxisWidths, lay.AxisW, verdict)
	return lay
}

// worldPrecKey identifie une vie d'objet. AVERTISSEMENT SUR CETTE CLÉ : deux segments d'un
// même couple (slot, génération) peuvent naître au même instant quand tous les points du
// premier tiennent dans un seul paquet et qu'un `at-rest` le clôt. La clé collisionne alors,
// et les comptes « changent / identiques » sont approchés à cette fraction près (mesuré :
// 2 collisions sur 477 trajectoires, 0,4 %). Le taux du critère, lui, est exact : il est
// compté sur les échantillons, pas sur les clés.
type worldPrecKey struct {
	slot, gen uint32
	birthUS   uint64
}

// worldPrecSig est la signature compacte d'une trajectoire : de quoi dire si elle a CHANGÉ
// sans conserver tous les points des deux mesures en mémoire simultanément.
type worldPrecSig struct {
	pts     int
	x, y, z float32
}

// worldPrecMeasure est le résultat d'un balayage à des largeurs données.
type worldPrecMeasure struct {
	axisW    [3]uint
	tracks   int
	samples  int
	in       int
	box      equipBox
	sigs     map[worldPrecKey]worldPrecSig
	walk     worldPrecWalk
	posBits  int
	inTracks int
}

// worldPrecWalk compte la MARCHE du décodeur sur le composant de position, avec ses deux
// causes d'échec séparées. Le parcours reproduit `scanProjectileRecords` à l'identique
// (même reconnaissance d'en-tête, même avance `p += posBits` sur record accepté) : l'écart
// entre `accepted` et le nombre d'échantillons rendus par `ScanFilmWorldObjects` n'est donc
// que le rejet des vies de moins de trois points par `splitLives`. Un `accepted` INFÉRIEUR
// au nombre d'échantillons signalerait que ce parcours a divergé — c'est vérifié.
type worldPrecWalk struct {
	cand     int // en-têtes reconnus portant i0 au masque
	gated    int // porte non nulle (precHigh / index-sel / index de région)
	sat      int // au moins un axe au quantum saturé (0 ou 2^w-1)
	accepted int
}

// worldPrecRun installe des largeurs, balaie l'archétype, et mesure.
func worldPrecRun(t *testing.T, dir string, ti int, axisW [3]uint, bip equipBox) worldPrecMeasure {
	t.Helper()
	WorldObjectPrecision.AxisW = axisW
	m := worldPrecMeasure{axisW: axisW, posBits: projPosBits(), sigs: map[worldPrecKey]worldPrecSig{}}
	m.walk = worldPrecWalkStats(dir, ti)

	unit := Vec3Range{{Min: 0, Max: 1}, {Min: 0, Max: 1}, {Min: 0, Max: 1}}
	tracks, err := ScanFilmWorldObjects(dir, &unit, ti)
	if err != nil {
		t.Logf("  balayage ti=%d aux largeurs %v impossible : %v", ti, axisW, err)
		return m
	}
	m.tracks = len(tracks)
	for _, tr := range tracks {
		m.samples += len(tr.Pts)
		allIn := true
		for _, p := range tr.Pts {
			v := [3]float32{p.X, p.Y, p.Z}
			m.box.add(v)
			if bip.contains(v) {
				m.in++
			} else {
				allIn = false
			}
		}
		if allIn {
			m.inTracks++
		}
		k := worldPrecKey{slot: tr.Slot, gen: tr.Gen, birthUS: tr.Pts[0].TimestampUS}
		m.sigs[k] = worldPrecSig{pts: len(tr.Pts), x: tr.Pts[0].X, y: tr.Pts[0].Y, z: tr.Pts[0].Z}
	}
	if m.walk.accepted < m.samples {
		t.Fatalf("parcours de contrôle DIVERGENT (ti=%d, largeurs %v) : %d acceptés < %d "+
			"échantillons rendus", ti, axisW, m.walk.accepted, m.samples)
	}
	return m
}

// worldPrecWalkStats compte les issues de la marche sur tout le film.
func worldPrecWalkStats(dir string, typeIndex int) worldPrecWalk {
	var w worldPrecWalk
	n := CountFilmChunks(dir)
	if n == 0 {
		return w
	}
	band := worldObjectSlotBand(dir, n, typeIndex)
	if len(band) == 0 {
		return w
	}
	for c := 1; c <= n; c++ {
		chunk, err := ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, p := range WalkPackets(chunk) {
			if p.Type != PacketTypeDelta {
				continue
			}
			worldPrecWalkPayload(p.Payload(chunk), band, &w)
		}
	}
	return w
}

// worldPrecWalkPayload reproduit le parcours de `scanProjectileRecords` sur un payload, en
// séparant les deux causes d'échec de la marche.
func worldPrecWalkPayload(pay []byte, band map[uint32]bool, w *worldPrecWalk) {
	unit := Vec3Range{{Min: 0, Max: 1}, {Min: 0, Max: 1}, {Min: 0, Max: 1}}
	posBits := projPosBits()
	limit := len(pay)*8 - (worldObjectHeaderBits + worldObjectIndexBits + posBits)
	for p := 0; p <= limit; p++ {
		rec, ok := matchWorldObjectRecord(pay, p, band)
		if !ok || rec.Idx[0] != 0 {
			continue
		}
		w.cand++
		if PeekBits(pay, rec.After, 3) != 0 {
			w.gated++
			continue
		}
		if _, ok := decodeWorldObjectPos(pay, rec.After, &unit); !ok {
			w.sat++
			continue
		}
		w.accepted++
		p += posBits
	}
}

// worldPrecReport publie une mesure avec TOUS ses dénominateurs.
func worldPrecReport(t *testing.T, label string, m worldPrecMeasure) {
	t.Helper()
	t.Logf("  %s · i0 = %d bits", label, m.posBits)
	t.Logf("    marche : %d records candidats · porte fermée %d · quantum saturé %d · "+
		"acceptés %d", m.walk.cand, m.walk.gated, m.walk.sat, m.walk.accepted)
	if m.samples == 0 {
		t.Log("    aucun échantillon rendu : critère non calculable")
		return
	}
	t.Logf("    %d trajectoires · %d échantillons · emprise %s", m.tracks, m.samples, m.box)
	t.Logf("    CRITÈRE : %d échantillons sur %d dans l'emprise des bipèdes (%.2f %%) · "+
		"%d trajectoires sur %d entièrement dedans (%.2f %%)",
		m.in, m.samples, 100*float64(m.in)/float64(m.samples),
		m.inTracks, m.tracks, 100*float64(m.inTracks)/float64(m.tracks))
}

// worldPrecDelta publie l'écart en clair : trajectoires qui apparaissent, disparaissent,
// changent ; échantillons qui entrent ou sortent de l'emprise.
func worldPrecDelta(t *testing.T, a, b worldPrecMeasure) {
	t.Helper()
	var appear, vanish, changed, same int
	for k, sb := range b.sigs {
		sa, ok := a.sigs[k]
		switch {
		case !ok:
			appear++
		case sa != sb:
			changed++
		default:
			same++
		}
	}
	for k := range a.sigs {
		if _, ok := b.sigs[k]; !ok {
			vanish++
		}
	}
	t.Logf("  ÉCART %v -> %v", a.axisW, b.axisW)
	t.Logf("    trajectoires : %d apparaissent · %d disparaissent · %d changent · "+
		"%d identiques", appear, vanish, changed, same)
	t.Logf("    échantillons dans l'emprise : %d -> %d (%+d) · hors emprise : %d -> %d (%+d)",
		a.in, b.in, b.in-a.in, a.samples-a.in, b.samples-b.in,
		(b.samples-b.in)-(a.samples-a.in))
	if a.samples > 0 && b.samples > 0 {
		ra := 100 * float64(a.in) / float64(a.samples)
		rb := 100 * float64(b.in) / float64(b.samples)
		t.Logf("    VERDICT : %.2f %% -> %.2f %% (%+.2f point) -> %s",
			ra, rb, rb-ra, worldPrecVerdict(rb, ra))
	}
}

// worldPrecVerdict tranche dans le sens prescrit par le plan : à égalité, le défaut gagne
// (on ne corrige pas ce qui ne s'améliore pas).
func worldPrecVerdict(catalog, def float64) string {
	if catalog > def {
		return "les largeurs DU CATALOGUE font mieux"
	}
	return "le DÉFAUT fait aussi bien ou mieux"
}
