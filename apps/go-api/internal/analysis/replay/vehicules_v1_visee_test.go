package replay

// vehicules_v1_visee_test.go — ITEM 2 du lot V1 (suite) : LA VISEE i21 SUR LE VEHICULE.
//
// LA QUESTION, REPRIORISEE PAR L'UTILISATEUR : en 2D on veut la DIRECTION et la VISEE (« ou porte
// le regard »), pas le haut/bas de i2 (parque, refute par V1a.3). i21 = unit-desired-aiming-vector
// porte un CAP DE VISEE (R(12), tour complet) et une elevation (R(11)). Le deserialiseur est deja
// porte (`readAimingVectorComponent`) et VALIDE sur le bipede (offline_aim.go : ecart moyen au
// deplacement nul a moins de 2 deg). La question est : i21 est-il present et fiable sur `ti=40` ?
//
// UN PIEGE DE MESURE, EXPLICITE. `scanRecordDirs` s'arrete au premier index de masque non modelise
// (seuls i1/i2/i3/i4/i5/i21 sont geres). Or i18/i19/i20 (unit-control/actor) sont < 21 et NON
// modelises : un record dont le masque les porte AVANT i21 ne fait jamais atteindre i21 au
// curseur. La CAPTURE DE VALEUR (HasYaw) SOUS-COMPTE donc la PRESENCE au masque. On mesure les
// DEUX : la presence au masque (hook `SetRecordMaskHook`) et la capture de valeur.
//
// TROIS CONFRONTATIONS, SEUILS ECRITS AVANT :
//
//	PRESENCE       i21 est "present dans le flux" si sa presence au masque depasse 1 % des records
//	               `ti=40` acceptes (plancher de faux positifs 0,17 %, cadrage § 1.4).
//	CONCENTRATION  une vraie visee a une elevation CONCENTREE pres de l'horizontale ; le bruit est
//	               uniforme. Part des tangages dans +/-45 deg : >= 60 % pour une vraie visee,
//	               ~25 % (90/360) pour un champ uniforme. Reference : le bipede, valide.
//	CONTINUITE     un cap de visee reel varie LENTEMENT dans le temps ; le bruit saute au hasard.
//	               Mediane du pas |dyaw| (pas <= 2 s) : < 45 deg pour un signal, TEMOIN par melange
//	               deterministe > 70 deg (un decalage ne fait pas un temoin sur serie autocorrelee,
//	               lecon V1a.3.4).
//	MOUVEMENT      informatif : le cap de visee suit-il la direction de deplacement (conducteur)
//	               ou en est-il independant (canonnier) ? Meme oracle que V1a.3, temoin par melange.
//
// LECTURE SEULE : aucun fichier ecrit, aucune base ouverte.
//
//	CGO_ENABLED=0 ATT_FILM=<depot>/data/cache \
//	  V0_FILMS="0d76e8f1:behemoth,fccc61cd:launch site,4898d586:behemoth" \
//	  go test ./internal/analysis/replay/ -run TestV1Visee -v -timeout 120m

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

// Seuils de l'item 2, ecrits avant mesure.
const (
	v1vPresenceMin         = 0.01 // presence au masque > 1 % (plancher faux positifs 0,17 %)
	v1vPitchWindowDeg      = 45.0 // demi-fenetre de concentration du tangage
	v1vPitchConcMin        = 0.60 // part dans la fenetre pour une vraie visee
	v1vPitchUniform        = 0.25 // 90/360 : la meme part sous un champ uniforme
	v1vYawStepMaxDeg       = 45.0 // mediane du pas de cap pour un signal continu
	v1vYawStepTemoinMinDeg = 70.0 // le melange doit remonter le pas au-dela de ce seuil
)

// TestV1ViseeI21 mesure la visee i21 sur le corpus.
func TestV1ViseeI21(t *testing.T) {
	root := attRequireRoot(t)
	for _, f := range v0Corpus(t) {
		v1vUnFilm(t, root, f)
	}
}

// v1vUnFilm mesure UN film.
func v1vUnFilm(t *testing.T, root string, f v0Film) {
	t.Helper()
	dir := objChunkDir(root, f.ID)
	if filmdec.CountFilmChunks(dir) == 0 {
		t.Logf("%s : film absent du cache — saute", f.ID)
		return
	}
	release := filmdec.LockProcessDecode()
	defer release()
	prev := filmdec.WorldObjectPrecision
	defer func() { filmdec.WorldObjectPrecision = prev }()
	wr, ok := v0Bornes(t, root, f.Carte)
	if !ok {
		return
	}
	bande := v1aBandeVehicule(dir)
	if len(bande) == 0 {
		t.Logf("V1v %s (%s) — bande ti=%d vide : rien a mesurer", f.ID, f.Carte, attVehiculeTI)
		return
	}
	lay, _, err := filmdec.DetectI0Layout(dir)
	if err != nil {
		t.Logf("V1v %s : decoupage i0 illisible : %v", f.ID, err)
		return
	}
	vehPos, total, hist := v1vScanAvecMasque(dir, bande, &wr, lay)
	bipPos := v1vScanBiped(dir, &wr, lay)
	v1vPresence(t, f, vehPos, total, hist, bipPos)
	v1vConcentration(t, f, vehPos, bipPos)
	v1vContinuite(t, f, vehPos)
	v1vMouvement(t, f, vehPos)
}

// v1vScanAvecMasque balaie la bande vehicule avec capture des directions ET histogramme les
// composants AU MASQUE via le hook. Le decoupage i0 est FORCE (opt.Layout) pour que le seul
// balayage qui declenche le hook soit celui-ci (DetectI0Layout ne le rejoue pas). Flux brut :
// hook 1:1 positions. Rendre l'histogramme entier (pas seulement i21) prouve que le hook lit de
// VRAIS masques : i21 absent au milieu de i1/i2/i3/i25 presents n'est alors pas un bug de hook.
func v1vScanAvecMasque(dir string, band map[uint32]bool, wr *filmdec.Vec3Range, lay filmdec.I0Layout) (
	[]filmdec.BipedPosition, int, map[int]int) {
	total := 0
	hist := map[int]int{}
	filmdec.SetRecordMaskHook(func(idx []int, _ []byte, _ int) {
		total++
		for _, id := range idx {
			hist[id]++
		}
	})
	defer filmdec.SetRecordMaskHook(nil)
	opt := v1aOptions(wr, false)
	opt.CaptureDirs, opt.Layout = true, &lay
	pos, err := filmdec.ScanFilmBipedPositionsForBand(dir, band, opt)
	if err != nil {
		return nil, total, hist
	}
	return pos, total, hist
}

// v1vScanBiped balaie le bipede (reference validee) avec capture des directions.
func v1vScanBiped(dir string, wr *filmdec.Vec3Range, lay filmdec.I0Layout) []filmdec.BipedPosition {
	opt := filmdec.DefaultScanFilmOptions()
	opt.WorldRange, opt.CaptureDirs, opt.Layout = wr, true, &lay
	pos, err := filmdec.ScanFilmBipedPositions(dir, opt)
	if err != nil {
		return nil
	}
	return pos
}

// v1vCompteYaw rend le nombre d'echantillons portant i21 (HasYaw).
func v1vCompteYaw(pos []filmdec.BipedPosition) int {
	n := 0
	for _, p := range pos {
		if p.HasYaw {
			n++
		}
	}
	return n
}

// v1vPresence publie la presence au masque et la capture de valeur, contre le bipede.
func v1vPresence(t *testing.T, f v0Film, veh []filmdec.BipedPosition, total int, hist map[int]int,
	bip []filmdec.BipedPosition) {
	t.Helper()
	masqI21 := hist[21]
	valYaw := v1vCompteYaw(veh)
	bipYaw := v1vCompteYaw(bip)
	verdict := "ABSENT/BRUIT"
	if attPart(masqI21, total) >= v1vPresenceMin {
		verdict = "PRESENT"
	}
	t.Logf("V1v %s (%s) — PRESENCE i21 : %d records ti=%d acceptes · %d au MASQUE (%.2f %%, seuil "+
		"%.0f %%) · %d captures de VALEUR (%.2f %%) · reference bipede %d/%d captures (%.1f %%) · %s",
		f.ID, f.Carte, total, attVehiculeTI, masqI21, 100*attPart(masqI21, total), 100*v1vPresenceMin,
		valYaw, 100*attPart(valYaw, total), bipYaw, len(bip), 100*attPart(bipYaw, len(bip)), verdict)
	t.Logf("V1v %s —   MASQUE ti=%d (le hook lit de VRAIS masques) : %s", f.ID, attVehiculeTI,
		v1vHistoLisible(hist, total))
	if masqI21 > valYaw {
		t.Logf("V1v %s —   NOTE : %d records portent i21 au masque mais n'atteignent pas le curseur "+
			"(composant non modelise entre i0 et i21, ex. i18/i19/i20 unit-control).", f.ID, masqI21-valYaw)
	}
}

// v1vHistoLisible rend l'histogramme des composants du masque, tries par index, part >= 0,3 %
// (au-dessus du plancher de faux positifs 0,17 % du cadrage § 1.4).
func v1vHistoLisible(hist map[int]int, total int) string {
	idx := make([]int, 0, len(hist))
	for k := range hist {
		idx = append(idx, k)
	}
	sort.Ints(idx)
	parts := make([]string, 0, len(idx))
	for _, k := range idx {
		if p := attPart(hist[k], total); p >= 0.003 {
			parts = append(parts, fmt.Sprintf("i%d=%.1f%%", k, 100*p))
		}
	}
	return strings.Join(parts, " ")
}

// v1vPitchConc rend la part des tangages dans +/-v1vPitchWindowDeg et leur nombre.
func v1vPitchConc(pos []filmdec.BipedPosition) (float64, int) {
	dans, n := 0, 0
	for _, p := range pos {
		pitch, ok := p.AimPitchDeg()
		if !ok {
			continue
		}
		n++
		if math.Abs(float64(pitch)) <= v1vPitchWindowDeg {
			dans++
		}
	}
	return attPart(dans, n), n
}

// v1vConcentration confronte la concentration du tangage vehicule a l'uniforme et au bipede.
func v1vConcentration(t *testing.T, f v0Film, veh, bip []filmdec.BipedPosition) {
	t.Helper()
	pv, nv := v1vPitchConc(veh)
	pb, nb := v1vPitchConc(bip)
	if nv == 0 {
		t.Logf("V1v %s (%s) — CONCENTRATION : aucun tangage i21 sur le vehicule, rien a juger", f.ID, f.Carte)
		return
	}
	verdict := "UNIFORME (bruit)"
	if pv >= v1vPitchConcMin {
		verdict = "CONCENTRE (visee reelle)"
	}
	t.Logf("V1v %s (%s) — CONCENTRATION du tangage : vehicule %.1f %% dans +/-%.0f deg (%d valeurs, "+
		"seuil %.0f %%) · uniforme %.0f %% · reference bipede %.1f %% (%d valeurs) · %s",
		f.ID, f.Carte, 100*pv, v1vPitchWindowDeg, nv, 100*v1vPitchConcMin, 100*v1vPitchUniform,
		100*pb, nb, verdict)
}

// v1vContinuite mesure la continuite temporelle du cap de visee, avec temoin par melange.
func v1vContinuite(t *testing.T, f v0Film, pos []filmdec.BipedPosition) {
	t.Helper()
	a, b := v1vPairesYawConsecutives(pos)
	if len(a) == 0 {
		t.Logf("V1v %s (%s) — CONTINUITE : aucune paire de caps consecutifs", f.ID, f.Carte)
		return
	}
	sig := make(v1aEcarts, len(a))
	tem := make(v1aEcarts, len(a))
	perm := v1aPermutation(len(a))
	for i := range a {
		sig[i] = v1aWrap180(b[i] - a[i])
		tem[i] = v1aWrap180(b[perm[i]] - a[i])
	}
	_, medSig, _ := sig.v1aStats()
	_, medTem, _ := tem.v1aStats()
	verdict := "BRUIT"
	if medSig < v1vYawStepMaxDeg && medTem > v1vYawStepTemoinMinDeg {
		verdict = "CONTINU (cap exploitable)"
	}
	t.Logf("V1v %s (%s) — CONTINUITE du cap : %d pas · mediane |dyaw| %.1f deg (seuil < %.0f) · "+
		"TEMOIN melange %.1f deg (seuil > %.0f) · %s",
		f.ID, f.Carte, len(a), medSig, v1vYawStepMaxDeg, medTem, v1vYawStepTemoinMinDeg, verdict)
}

// v1vPairesYawConsecutives rend, par slot, les couples de caps de visee consecutifs (pas <= 2 s).
func v1vPairesYawConsecutives(pos []filmdec.BipedPosition) (a, b []float64) {
	parSlot := map[uint32][]filmdec.BipedPosition{}
	for _, p := range pos {
		if p.HasYaw {
			parSlot[p.Slot] = append(parSlot[p.Slot], p)
		}
	}
	slots := make([]uint32, 0, len(parSlot))
	for s := range parSlot {
		slots = append(slots, s)
	}
	sort.Slice(slots, func(i, j int) bool { return slots[i] < slots[j] })
	for _, s := range slots {
		ech := parSlot[s]
		sort.SliceStable(ech, func(i, j int) bool { return ech[i].TimestampUS < ech[j].TimestampUS })
		for i := 0; i+1 < len(ech); i++ {
			if !v0PasCompte(ech[i].TimestampUS, ech[i+1].TimestampUS) {
				continue
			}
			ya, _ := ech[i].AimHeadingDeg()
			yb, _ := ech[i+1].AimHeadingDeg()
			a = append(a, float64(ya))
			b = append(b, float64(yb))
		}
	}
	return a, b
}

// v1vMouvement confronte le cap de visee a la direction de deplacement (velocite i1), informatif.
func v1vMouvement(t *testing.T, f v0Film, pos []filmdec.BipedPosition) {
	t.Helper()
	var caps, ref v1aEcarts
	var vit []float64
	for _, p := range pos {
		if !p.HasYaw || !p.HasVel {
			continue
		}
		v, ok := p.VelocityVector()
		if !ok {
			continue
		}
		sp := dist3(v, [3]float32{})
		if sp <= v1aVitesseMinMPS {
			continue
		}
		yaw, _ := p.AimHeadingDeg()
		caps = append(caps, float64(yaw))
		ref = append(ref, v1aCapDeg(v[0], v[1]))
		vit = append(vit, sp)
	}
	if len(caps) == 0 {
		t.Logf("V1v %s (%s) — MOUVEMENT : aucune paire (i21 + velocite > %.0f m/s)", f.ID, f.Carte, v1aVitesseMinMPS)
		return
	}
	ecarts := make(v1aEcarts, len(caps))
	temoin := make(v1aEcarts, len(caps))
	perm := v1aPermutation(len(caps))
	for i := range caps {
		ecarts[i] = v1aWrap180(caps[i] - ref[i])
		temoin[i] = v1aWrap180(caps[perm[i]] - ref[i])
	}
	moy, med, r := ecarts.v1aStats()
	tmoy, tmed, tr := temoin.v1aStats()
	t.Logf("V1v %s (%s) — MOUVEMENT (cap visee i21 contre deplacement) : %d paires · moyenne "+
		"circ. %+.1f deg · mediane |ecart| %.1f deg · R %.3f · TEMOIN melange moyenne %+.1f deg "+
		"mediane %.1f deg R %.3f",
		f.ID, f.Carte, len(caps), moy, med, r, tmoy, tmed, tr)
}
