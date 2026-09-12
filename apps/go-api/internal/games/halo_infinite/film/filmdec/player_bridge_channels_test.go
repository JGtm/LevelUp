package filmdec

// player_bridge_channels_test.go — LES TROIS MESURES de `player_bridge_measure_test.go` (qui
// porte l'en-tete, le contrat et les seuils) : P.0.3 l'arme en main, P.0.4 la seconde source de
// visee, et le compte des fenetres actives de reapparition. Scinde du premier pour tenir le
// seuil de 500 lignes par fichier.

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// pbHeldWeapon mesure P.0.3 : au frame de chaque tir, l'arme en main est-elle de la famille du
// tir ? La recherche porte sur TOUS les slots (cf. l'en-tete de `player_bridge_measure_test.go`) :
// le chiffre rendu est une BORNE SUPERIEURE de la couverture reelle.
func pbHeldWeapon(t *testing.T, in pbInputs) {
	t.Helper()
	bySlot := map[uint32][]HeldWeaponSample{}
	for _, h := range in.held {
		bySlot[h.Slot] = append(bySlot[h.Slot], h)
	}
	all := make([]uint32, 0, len(bySlot))
	for s := range bySlot {
		sort.Slice(bySlot[s], func(i, j int) bool {
			return bySlot[s][i].TimestampUS < bySlot[s][j].TimestampUS
		})
		all = append(all, s)
	}
	var total, withHeld, agree int
	changes := pbWeaponChanges(bySlot)
	for _, sh := range in.shots {
		if sh.WeaponID == 0 {
			continue
		}
		total++
		fam, ok := pbHeldAt(bySlot, all, sh.TimestampUS)
		if !ok {
			continue
		}
		withHeld++
		if fam == uint32(sh.WeaponID>>32) {
			agree++
		}
	}
	t.Logf("P.0.3 ARME EN MAIN · tirs %d · tirs avec une arme en main connue %d (%.1f %%, "+
		"BORNE SUPERIEURE) · ACCORD famille %d (%.1f %% des tirs couverts, %.1f %% de tous les "+
		"tirs) · seuil 90 %% · %s", total, withHeld, pbPct(withHeld, total), agree,
		pbPct(agree, withHeld), pbPct(agree, total), pbVerdict(agree, withHeld, 90))
	t.Logf("P.0.3 CHANGEMENTS D ARME · %d transitions relevees sur %d slots · latence mediane "+
		"entre deux lectures du meme slot %.2f s", changes.n, len(bySlot), changes.medianS)
	pbHeldWeaponCensus(t, in)
}

// pbHeldWeaponCensus dit POURQUOI le canal rend ce qu'il rend : sur les records de bipede
// certains, les quatre composants d'identite d'arme (i43-i46) sont-ils seulement ANNONCES ?
// Un canal jamais annonce et un canal annonce mais illisible appellent des conclusions
// opposees, et seul ce recensement les separe.
func pbHeldWeaponCensus(t *testing.T, in pbInputs) {
	t.Helper()
	var parts []string
	for i := 40; i <= 47; i++ {
		parts = append(parts, fmt.Sprintf("i%d:%d", i, in.chain.BipedMask[i]))
	}
	top := 0
	for i := 0; i < len(in.chain.BipedMask); i++ {
		if in.chain.BipedMask[i] > top {
			top = in.chain.BipedMask[i]
		}
	}
	t.Logf("P.0.3 ANNONCES AU MASQUE (records de bipede certains %d, composant le plus annonce "+
		"%d fois) · %s", in.chain.BipedRecords, top, strings.Join(parts, " "))
}

// pbWeaponChangeStats resume les transitions d'arme en main.
type pbWeaponChangeStats struct {
	n       int
	medianS float64
}

// pbWeaponChanges compte les transitions et l'ecart median entre deux lectures d'un slot —
// c'est la CADENCE du canal, la grandeur qui borne toute latence mesurable.
func pbWeaponChanges(bySlot map[uint32][]HeldWeaponSample) pbWeaponChangeStats {
	var gaps []float64
	n := 0
	for _, hs := range bySlot {
		for i := 1; i < len(hs); i++ {
			gaps = append(gaps, float64(hs[i].TimestampUS-hs[i-1].TimestampUS)/1e6)
			if hs[i].Family != hs[i-1].Family {
				n++
			}
		}
	}
	sort.Float64s(gaps)
	med := 0.0
	if len(gaps) > 0 {
		med = gaps[len(gaps)/2]
	}
	return pbWeaponChangeStats{n: n, medianS: med}
}

// pbHeldAt rend la derniere famille d'arme lue sur l'un des slots avant `at`.
func pbHeldAt(bySlot map[uint32][]HeldWeaponSample, slots []uint32, at uint64) (uint32, bool) {
	var best uint64
	var fam uint32
	found := false
	for _, s := range slots {
		hs := bySlot[s]
		i := sort.Search(len(hs), func(k int) bool { return hs[k].TimestampUS > at })
		if i == 0 {
			continue
		}
		h := hs[i-1]
		if at-h.TimestampUS > pbShotWindowUS {
			continue
		}
		if !found || h.TimestampUS > best {
			best, fam, found = h.TimestampUS, h.Family, true
		}
	}
	return fam, found
}

// pbAimPairs mesure P.0.4 : la visee du JOUEUR (ti=5 i17, cubemap 19 bits) s'accorde-t-elle
// avec celle du CORPS (i21, cap quantifie sur 12 bits) quand les deux tombent a <= 100 ms ?
func pbAimPairs(t *testing.T, in pbInputs) {
	t.Helper()
	bridge, quality := pbBridge(t, in)
	if len(bridge) == 0 {
		t.Log("P.0.4 VISEE · aucun pont entite joueur -> bipede : item NON MESURABLE sur ce film")
		return
	}
	byBiped := map[uint32][]BipedPosition{}
	for _, p := range in.pos {
		if p.HasYaw {
			byBiped[p.Slot] = append(byBiped[p.Slot], p)
		}
	}
	pairs, within, deltas := pbCountAimPairs(in, bridge, byBiped)
	sort.Float64s(deltas)
	med := 0.0
	if len(deltas) > 0 {
		med = deltas[len(deltas)/2]
	}
	t.Logf("P.0.4 VISEE · pont %d entites joueur (qualite %s) · paires a <= 100 ms %d "+
		"· |delta cap| <= 5° %d (%.1f %%) · ecart median %.1f° · seuil 90 %% · %s",
		len(bridge), quality, pairs, within, pbPct(within, pairs), med,
		pbVerdict(within, pairs, 90))
	noYaw := 0
	for _, p := range in.pos {
		if !p.HasYaw {
			noYaw++
		}
	}
	t.Logf("P.0.4 COUVERTURE · points sans cap de corps %d / %d (%.1f %%) · lectures de visee "+
		"joueur disponibles %d — couverture AJOUTEE au mieux %.2f %%", noYaw, len(in.pos),
		pbPct(noYaw, len(in.pos)), pairs, pbPct(pairs, noYaw))
}

// pbCountAimPairs apparie les lectures d'i17 aux lectures d'i21 du meme joueur.
func pbCountAimPairs(
	in pbInputs, bridge map[uint32]uint32, byBiped map[uint32][]BipedPosition,
) (pairs, within int, deltas []float64) {
	for _, r := range in.player {
		if r.TI != PlayerEngineTypeIndex || !r.PlayerSeen[PlayerControlAiming] {
			continue
		}
		if !r.PlayerPresent[PlayerControlAiming] || len(r.PlayerVal[PlayerControlAiming]) == 0 {
			continue
		}
		bslot, ok := bridge[r.Slot]
		if !ok {
			continue
		}
		p, ok := pbNearestYaw(byBiped[bslot], r.TimestampUS)
		if !ok {
			continue
		}
		v, ok := DecodeAimVectorChecked(uint32(r.PlayerVal[PlayerControlAiming][0]), aimDirBits)
		if !ok {
			continue
		}
		body, ok := p.AimHeadingDeg()
		if !ok {
			continue
		}
		pairs++
		d := pbAngleDelta(pbHeadingDeg(v), float64(body))
		deltas = append(deltas, d)
		if d <= 5 {
			within++
		}
	}
	return pairs, within, deltas
}

// pbHeadingDeg projette un vecteur unitaire monde sur le cap [0,360[, meme convention que
// `BipedPosition.AimHeadingDeg` (atan2(Y, X) des positions dequantifiees).
func pbHeadingDeg(v [3]float32) float64 {
	d := math.Atan2(float64(v[1]), float64(v[0])) * 180 / math.Pi
	if d < 0 {
		d += 360
	}
	return d
}

// pbNearestYaw rend la position de bipede la plus proche dans le temps, si elle tombe dans la
// fenetre d'appariement.
func pbNearestYaw(ps []BipedPosition, at uint64) (BipedPosition, bool) {
	best, found := BipedPosition{}, false
	var bd uint64 = pbPairWindowUS + 1
	for _, p := range ps {
		d := at - p.TimestampUS
		if p.TimestampUS > at {
			d = p.TimestampUS - at
		}
		if d < bd {
			bd, best, found = d, p, true
		}
	}
	if bd > pbPairWindowUS {
		return BipedPosition{}, false
	}
	return best, found
}

// pbBridge apparie chaque slot d'entite joueur au slot de bipede dont les fins de vie
// coincident le mieux avec ses fenetres de reapparition ACTIVES.
//
// LE RESULTAT EST RENDU AVEC SA QUALITE, jamais seul : un appariement construit sur trois
// coincidences n'est pas un pont, et le presenter comme tel ferait passer du hasard pour une
// mesure.
func pbBridge(t *testing.T, in pbInputs) (map[uint32]uint32, string) {
	t.Helper()
	active := map[uint32][]uint64{}
	for _, r := range in.player {
		if r.TI == PlayerEngineTypeIndex && r.HasRespawn && r.Respawn.Active {
			active[r.Slot] = append(active[r.Slot], r.TimestampUS)
		}
	}
	out := map[uint32]uint32{}
	hits, tries := 0, 0
	for ps, times := range active {
		bestSlot, bestN := uint32(0), 0
		for bs, es := range in.ends {
			if n := pbCoincidences(times, es); n > bestN {
				bestN, bestSlot = n, bs
			}
		}
		tries += len(times)
		if bestN == 0 {
			continue
		}
		hits += bestN
		out[ps] = bestSlot
	}
	q := fmt.Sprintf("%d fenetres actives appariees sur %d", hits, tries)
	t.Logf("PONT ti=5 -> BIPEDE · entites joueur avec au moins une fenetre active %d "+
		"· appariees %d · %s", len(active), len(out), q)
	return out, q
}

// pbCoincidences compte les instants de `times` qui tombent a portee d'une fin de vie.
func pbCoincidences(times, ends []uint64) int {
	n := 0
	for _, at := range times {
		for _, e := range ends {
			if pbAbsU(at, e) <= pbBridgeWindowUS {
				n++
				break
			}
		}
	}
	return n
}

// pbRespawnWindows compte les fenetres ACTIVES du compte a rebours — la moitie de B.0.4 qui
// vient de la chaine. L'autre moitie (delai mort -> reapparition, temps mort cumule) vit dans
// `replay`, parce qu'elle a besoin du fil des morts pour nommer les vies.
func pbRespawnWindows(t *testing.T, in pbInputs) {
	t.Helper()
	var reads, active int
	hist := map[string]int{}
	for _, r := range in.player {
		if r.TI != PlayerEngineTypeIndex || !r.HasRespawn {
			continue
		}
		reads++
		if r.Respawn.Active {
			active++
		}
		hist[fmt.Sprintf("%v/%d/%d", r.Respawn.Active, r.Respawn.T0, r.Respawn.T1)]++
	}
	t.Logf("B.0.4 (moitie chaine) COMPTE A REBOURS · lectures %d dont ACTIVES %d (%.2f %%) "+
		"· valeurs distinctes %d · %s", reads, active, pbPct(active, reads), len(hist),
		gameTopValues(hist, 6))
}

// pbDump depose les echantillons bruts sous GAME_OUT (le meme repertoire que l'autre
// instrument du paquet : deux variables pour un seul dossier ne serviraient qu'a le perdre).
func pbDump(t *testing.T, in pbInputs) {
	t.Helper()
	out := os.Getenv(gameOutEnv)
	if out == "" {
		return
	}
	if err := os.MkdirAll(out, 0o755); err != nil {
		t.Fatalf("repertoire de sortie %s : %v", out, err)
	}
	rows := make([]string, 0, len(in.held))
	for _, h := range in.held {
		rows = append(rows, fmt.Sprintf("%d\t%d\t%d", h.Slot, h.TimestampUS, h.Family))
	}
	gameWriteTSV(t, filepath.Join(out, in.short+"_arme_en_main.tsv"), "slot\tt_us\tfamille", rows)
	t.Logf("TSV arme en main depose dans %s (%d lignes)", out, len(in.held))
}

func pbPct(num, den int) float64 {
	if den <= 0 {
		return 0
	}
	return 100 * float64(num) / float64(den)
}

// pbVerdict rend TENU / NON TENU contre un seuil en pourcentage, ou « denominateur nul ».
func pbVerdict(num, den, seuil int) string {
	if den <= 0 {
		return "NON MESURABLE (denominateur nul)"
	}
	if pbPct(num, den) >= float64(seuil) {
		return "TENU"
	}
	return "NON TENU"
}

func pbAbsU(a, b uint64) uint64 {
	if a > b {
		return a - b
	}
	return b - a
}

// pbAngleDelta rend l'ecart angulaire minimal entre deux caps en degres.
func pbAngleDelta(a, b float64) float64 {
	d := a - b
	for d > 180 {
		d -= 360
	}
	for d < -180 {
		d += 360
	}
	if d < 0 {
		return -d
	}
	return d
}
