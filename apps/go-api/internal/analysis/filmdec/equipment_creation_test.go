package filmdec

// equipment_creation_test.go — INSTRUMENT DE MESURE du RECORD DE CRÉATION de ti=37.
//
// LA QUESTION POSÉE, et elle est falsifiable : le default-state de l'archétype d'équipement
// porte un identifiant 32 bits (« ability-enabled-id ») et une référence d'entité 5 bits, tous
// deux jetés jusqu'ici. Est-ce que l'un des deux DISCRIMINE — c'est-à-dire : l'identifiant
// prend-il une POIGNÉE de valeurs, les mêmes d'un film à l'autre, ou autant de valeurs que
// d'entités ?
//
// PRÉDICTION ÉCRITE AVANT LA MESURE (plan PLAN_IDENTITE_TI37, phase 0.2) : moins de 30 valeurs
// distinctes, dont deux dominent sur les films dont un joueur porte un mur ou un capteur.
//
// CE QUE LA MESURE A RENDU (2026-08-17) — la prédiction est REFUTÉE, et le résultat est
// ailleurs. Les DEUX champs du default-state ont leur porte FERMÉE sur 503 records de création
// sur 503 (274 sur `000d5950`, 229 sur `00162144`) : ni identifiant de capacité, ni référence
// de créateur, jamais. Mais le MÊME record porte le bloc `object-multiplayer-properties`, dont
// le mot de 32 bits INCONDITIONNEL prend 8 valeurs sur 274 records — et ces valeurs sont les
// GlobalID des tags `eqip` du jeu (11 sur 11 résolues, cf. himap/sonde_ti37_gamefiles_test.go).
// C'est pourquoi ce fichier publie AUSSI les quatre champs du bloc MPP : chercher l'identité
// de l'objet ailleurs que dans son propre record de naissance n'aurait pas de sens.
//
// LE TÉMOIN QUI PEUT TOUT ANNULER : le même balayage sur une bande FANTÔME de même cardinalité,
// faite de slots JAMAIS vus porter ti=37. S'il rend autant de records, l'ancre ne reconnaît pas
// des créations d'équipement mais du bruit — et aucune distribution ne vaut.
//
// LECTURE SEULE, gardé par EQUIP_CREATION_FILM, sauté partout ailleurs (CI comprise). UN SEUL
// décodage filmdec par process (globaux de paquet) : le verrou est pris pour tout le test.
//
// USAGE (depuis apps/go-api) :
//
//	CGO_ENABLED=0 EQUIP_CREATION_FILM=<repo>/data/cache/film_chunks/000d5950 \
//	  go test ./internal/analysis/filmdec/ -run '^TestEquipmentCreationRecord$' -timeout 60m -v
//
// `EQUIP_MPP_LEAD=<n>` force la largeur du PREMIER champ du bloc MPP au lieu de calibrer les
// deux. Il ne sert plus qu'à rejouer une hypothèse à la main : depuis le 2026-08-18 la
// calibration par oracle de position tranche le DÉCOUPAGE COMPLET (premier champ / champ
// inline) sur 11 films du corpus sur 12, et c'est elle que la production emploie. Forcer une
// largeur seule reproduit le défaut que la calibration a corrigé — un total juste avec un
// identifiant lu trois bits trop tôt.

import (
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
	"testing"
)

const equipCreationFilmEnv = "EQUIP_CREATION_FILM"

// equipCreationUnitRange : coordonnées NORMALISÉES [0,1] de l'AABB de la carte. Les bornes
// MONDE exigeraient le nom de la carte, donc une base ; la normalisation suffit à tout ce que
// cet instrument juge (les deux nuages sont quantifiés sur le MÊME repère, lu dans le film).
var equipCreationUnitRange = Vec3Range{{Min: 0, Max: 1}, {Min: 0, Max: 1}, {Min: 0, Max: 1}}

func TestEquipmentCreationRecord(t *testing.T) {
	dir := os.Getenv(equipCreationFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument de mesure sauté", equipCreationFilmEnv)
	}
	release := LockProcessDecode()
	defer release()

	lay, _, err := DetectI0Layout(dir)
	if err != nil {
		t.Fatalf("découpage i0 illisible dans %s : %v", dir, err)
	}
	prev := WorldObjectPrecision
	t.Cleanup(func() { WorldObjectPrecision = prev })
	SetWorldObjectPrecisionFromLayout(lay)
	t.Logf("FILM %s · largeurs d'axe lues dans le film %v", dir, lay.AxisW)

	real, phantom := equipCreationBands(t, dir)
	prevW := CurrentMPPWidths()
	t.Cleanup(func() { SetMPPWidths(prevW) })
	if forced := os.Getenv("EQUIP_MPP_LEAD"); forced != "" {
		w, err := strconv.Atoi(forced)
		if err != nil {
			t.Fatalf("EQUIP_MPP_LEAD=%q illisible : %v", forced, err)
		}
		SetMPPWidths(MPPWidths{Lead: w, Index: prevW.Index})
		t.Logf("largeur du premier champ MPP FORCÉE par l'environnement : %d bits", w)
	} else {
		tracks, err := ScanFilmWorldObjects(dir, &equipCreationUnitRange, EquipmentTypeIndex)
		if err != nil {
			t.Fatalf("trajectoires ti=%d illisibles (oracle de calibration) : %v",
				EquipmentTypeIndex, err)
		}
		cal, ok := CalibrateMPPWidths(dir, &equipCreationUnitRange, real, EquipmentLifeSpans(tracks))
		t.Logf("découpage du bloc MPP — calibration : %s", cal)
		if !ok {
			t.Logf("calibration NON concluante : défaut %s conservé", prevW)
		} else {
			SetMPPWidths(cal.Widths)
		}
	}
	cre, st, err := ScanFilmEquipmentCreationsForBand(dir, &equipCreationUnitRange, real)
	if err != nil {
		t.Fatalf("balayage des créations impossible : %v", err)
	}
	equipCreationLogStats(t, "BANDE RÉELLE", st)
	_, pst, err := ScanFilmEquipmentCreationsForBand(dir, &equipCreationUnitRange, phantom)
	if err != nil {
		t.Fatalf("balayage témoin impossible : %v", err)
	}
	equipCreationLogStats(t, "TÉMOIN FANTÔME", pst)
	if len(cre) == 0 {
		t.Log("VERDICT PHASE 0 : aucun record de création lu — le canal ne porte rien ici")
		return
	}
	clean := equipCreationFilter(cre, func(c EquipmentCreation) bool { return c.MaskFull && c.MaskHasI0 })
	equipCreationLogFields(t, "TOUS", cre)
	equipCreationLogFields(t, "COHORTE PROPRE (masque plein + i0)", clean)
	equipCreationLogLives(t, clean)
	equipCreationCrossCheck(t, dir, cre)
}

// equipCreationBands rend la bande RÉELLE de ti=37 et une bande FANTÔME de même cardinalité,
// tirée des slots vus porter un AUTRE archétype. Le témoin passe par le même code que la mesure.
func equipCreationBands(t *testing.T, dir string) (real, phantom map[uint32]bool) {
	t.Helper()
	n := CountFilmChunks(dir)
	real = worldObjectSlotBandDir(dir, n, EquipmentTypeIndex)
	if len(real) == 0 {
		t.Fatalf("aucun slot ti=%d dans les images-clés de %s", EquipmentTypeIndex, dir)
	}
	others := map[uint32]bool{}
	for c := 1; c <= n; c++ {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, pk := range WalkPackets(data) {
			if pk.Type != PacketTypeKeyframe {
				continue
			}
			for _, r := range WalkKeyframeWorld(pk.Payload(data)) {
				if r.TI != EquipmentTypeIndex && !real[uint32(r.Slot)] {
					others[uint32(r.Slot)] = true
				}
			}
		}
	}
	keys := make([]uint32, 0, len(others))
	for s := range others {
		keys = append(keys, s)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	phantom = map[uint32]bool{}
	for _, s := range keys {
		if len(phantom) >= len(real) {
			break
		}
		phantom[s] = true
	}
	return real, phantom
}

func equipCreationLogStats(t *testing.T, label string, st EquipmentCreationStats) {
	t.Helper()
	t.Logf("== %s — %d slots · ancres NEW ti=%d %d · ACCEPTÉS %d ==",
		label, st.Slots, EquipmentTypeIndex, st.Anchors, st.Accepted)
	t.Logf("   rejets : default-state hors payload %d · masque invalide %d · position rejetée %d",
		st.Overflow, st.MaskBad, st.PosBad)
	t.Logf("   acceptés : masque éparse %d · masque PLEIN %d · sans i0 au masque %d",
		st.MaskSparse, st.MaskFull, st.NoI0)
	t.Logf("   portes ouvertes : entity-ref-index5 %d · ability-enabled-id %d",
		st.WithRef, st.WithID)
}

// equipCreationLogFields publie la DISTRIBUTION des deux champs. C'est la mesure du gate 0 :
// une poignée de valeurs stables dit une énumération ou un identifiant de définition ; autant
// de valeurs que d'entités dit du bruit, et le plan s'arrête là.
func equipCreationLogFields(t *testing.T, label string, cre []EquipmentCreation) {
	t.Helper()
	ids := map[uint32]int{}
	refs := map[uint32]int{}
	widths := map[uint32]int{}
	for _, c := range cre {
		widths[uint32(c.DefaultStateBits)]++
		if c.HasID {
			ids[c.AbilityID]++
		}
		if c.HasRef {
			refs[c.Ref]++
		}
	}
	t.Logf("== %s (%d records) — `ability-enabled-id` transmis %d fois, %d valeurs distinctes ==",
		label, len(cre), equipCreationSum(ids), len(ids))
	for _, e := range equipCreationTop(ids, 40) {
		t.Logf("   %#010x (%10d)  %6d", e.key, e.key, e.count)
	}
	t.Logf("   `entity-ref-index5` transmis %d fois, %d valeurs distinctes :%s",
		equipCreationSum(refs), len(refs), equipCreationLine(refs, 32))
	t.Logf("   largeurs du default-state (%d distinctes) :%s", len(widths),
		equipCreationLine(widths, 12))
	equipCreationLogMPP(t, cre)
}

// equipCreationLogMPP publie la distribution des quatre champs du bloc
// `object-multiplayer-properties` du MÊME record. C'est là que se trouve le SEUL champ de
// 32 bits transmis sur tous les records de création : si l'identité de l'objet est écrite
// quelque part dans son record de naissance, c'est le premier endroit où la chercher.
func equipCreationLogMPP(t *testing.T, cre []EquipmentCreation) {
	t.Helper()
	for f := 0; f < MPPFieldCount; f++ {
		h := map[uint32]int{}
		lives := map[uint32]map[equipCreationLifeKey]bool{}
		for _, c := range cre {
			if !c.MPPPresent[f] {
				continue
			}
			v := uint32(c.MPPVal[f])
			h[v]++
			if lives[v] == nil {
				lives[v] = map[equipCreationLifeKey]bool{}
			}
			lives[v][equipCreationLifeKey{c.Slot, c.Gen}] = true
		}
		t.Logf("   %-18s transmis %d fois · %d valeurs distinctes", MPPField(f),
			equipCreationSum(h), len(h))
		for _, e := range equipCreationTop(h, 12) {
			t.Logf("      %#010x (%10d) %5d records · %d vies", e.key, e.key, e.count,
				len(lives[e.key]))
		}
	}
}

func equipCreationSum(h map[uint32]int) int {
	n := 0
	for _, v := range h {
		n += v
	}
	return n
}

func equipCreationLine(h map[uint32]int, limit int) string {
	out := ""
	for _, e := range equipCreationTop(h, limit) {
		out += fmt.Sprintf(" %d:%d", e.key, e.count)
	}
	return out
}

// equipCreationLogLives compte les VIES d'objet (slot, génération) et dit combien portent un
// identifiant — le dénominateur sans lequel « N records » ne se juge pas.
func equipCreationLogLives(t *testing.T, cre []EquipmentCreation) {
	t.Helper()
	type key struct{ slot, gen uint32 }
	lives := map[key]bool{}
	withID := map[key]bool{}
	idPerLife := map[key]map[uint32]bool{}
	for _, c := range cre {
		k := key{c.Slot, c.Gen}
		lives[k] = true
		if c.HasID {
			withID[k] = true
			if idPerLife[k] == nil {
				idPerLife[k] = map[uint32]bool{}
			}
			idPerLife[k][c.AbilityID] = true
		}
	}
	multi := 0
	for _, m := range idPerLife {
		if len(m) > 1 {
			multi++
		}
	}
	t.Logf("== VIES D'OBJET — %d couples (slot, génération) · %d portent un identifiant"+
		" · %d en portent PLUSIEURS ==", len(lives), len(withID), multi)
}

// equipCreationCrossCheck confronte les créations aux VIES décodées des paquets delta
// (ScanFilmWorldObjects) : une création dont le couple (slot, génération) n'existe dans aucune
// trajectoire, ou dont la position ne retombe pas sur le premier point de la vie, est un faux
// positif. C'est la chaîne indépendante qui juge l'ancre.
func equipCreationCrossCheck(t *testing.T, dir string, cre []EquipmentCreation) {
	t.Helper()
	tracks, err := ScanFilmWorldObjects(dir, &equipCreationUnitRange, EquipmentTypeIndex)
	if err != nil {
		t.Logf("CONTRÔLE CROISÉ non calculable : %v", err)
		return
	}
	firstPt := map[equipCreationLifeKey]ProjectileSample{}
	for _, tr := range tracks {
		k := equipCreationLifeKey{tr.Slot, tr.Gen}
		if p, ok := firstPt[k]; !ok || tr.Pts[0].TimestampUS < p.TimestampUS {
			firstPt[k] = tr.Pts[0]
		}
	}
	t.Logf("== CONTRÔLE CROISÉ — %d créations contre %d vies delta ==", len(cre), len(firstPt))
	equipCreationCohort(t, "TOUTES", cre, firstPt)
	equipCreationCohort(t, "masque PLEIN", equipCreationFilter(cre,
		func(c EquipmentCreation) bool { return c.MaskFull }), firstPt)
	equipCreationCohort(t, "masque éparse", equipCreationFilter(cre,
		func(c EquipmentCreation) bool { return !c.MaskFull }), firstPt)
	equipCreationCohort(t, "i0 AU masque", equipCreationFilter(cre,
		func(c EquipmentCreation) bool { return c.MaskHasI0 }), firstPt)
	equipCreationCohort(t, "i0 ABSENT du masque", equipCreationFilter(cre,
		func(c EquipmentCreation) bool { return !c.MaskHasI0 }), firstPt)
	equipCreationCohort(t, "PROPRE + id transmis", equipCreationFilter(cre,
		func(c EquipmentCreation) bool { return c.MaskFull && c.MaskHasI0 && c.HasID }), firstPt)
}

type equipCreationLifeKey struct{ slot, gen uint32 }

// equipCreationCohort juge UNE cohorte de créations contre les vies delta : part appariée,
// antériorité, et écart de position. Séparer les cohortes est ce qui dit LAQUELLE des formes
// acceptées porte du signal — une moyenne globale les mélangerait.
func equipCreationCohort(
	t *testing.T, label string, cre []EquipmentCreation,
	firstPt map[equipCreationLifeKey]ProjectileSample,
) {
	t.Helper()
	if len(cre) == 0 {
		t.Logf("   %-22s aucune création", label)
		return
	}
	matched, before := 0, 0
	var d []float64
	for _, c := range cre {
		p, ok := firstPt[equipCreationLifeKey{c.Slot, c.Gen}]
		if !ok {
			continue
		}
		matched++
		if c.TimestampUS <= p.TimestampUS {
			before++
		}
		d = append(d, equipCreationDist(c, p))
	}
	if len(d) == 0 {
		t.Logf("   %-22s %d créations · 0 appariée à une vie delta", label, len(cre))
		return
	}
	sort.Float64s(d)
	t.Logf("   %-22s %d créations · %d appariées (%.1f %%) · %d antérieures ou égales"+
		" · écart médian %.5f · p90 %.5f", label, len(cre), matched,
		100*float64(matched)/float64(len(cre)), before, d[len(d)/2], d[(len(d)*9)/10])
}

func equipCreationFilter(cre []EquipmentCreation, keep func(EquipmentCreation) bool) []EquipmentCreation {
	var out []EquipmentCreation
	for _, c := range cre {
		if keep(c) {
			out = append(out, c)
		}
	}
	return out
}

func equipCreationDist(c EquipmentCreation, p ProjectileSample) float64 {
	dx, dy, dz := float64(c.X-p.X), float64(c.Y-p.Y), float64(c.Z-p.Z)
	return math.Sqrt(dx*dx + dy*dy + dz*dz)
}

type equipCreationEntry struct {
	key   uint32
	count int
}

// equipCreationTop rend les entrées les plus fréquentes, décroissant, bornées à limit.
func equipCreationTop(h map[uint32]int, limit int) []equipCreationEntry {
	out := make([]equipCreationEntry, 0, len(h))
	for k, n := range h {
		out = append(out, equipCreationEntry{key: k, count: n})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].count != out[j].count {
			return out[i].count > out[j].count
		}
		return out[i].key < out[j].key
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out
}
