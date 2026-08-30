package replay

// visee_camera_pov_research_test.go — A QUI APPARTIENT LA CAMERA DU TYPE 97 ?
//
// CE QUI EST DEJA ETABLI (phase 4, `visee_canal_zoom_research_test.go`) : le paquet de type 97
// porte une camera premiere personne complete — une position (largeur RUNTIME, non decodee ici)
// puis DEUX angles dequantises de 20 bits chacun, adjacents dans le flux :
//
//	tangage  R(20) sur [-1,49226 ; +1,49226] rad (= ±85,5°, la butee de camera d'un FPS)
//	cap      R(20) sur [0 ; 2π[
//
// (bornes lues dans l'exe : 143d13010 / 143d13488 / 143cd842c ; largeurs aux sites d'appel
// 142f16111 / 142f16136 : `MOV dword ptr [RSP+0x20], 0x14`). Et son FLUX S'EFFONDRE dans les
// fenetres des kills zoomes (43 % de presence contre 88-95 % pour les deux temoins).
//
// CE QUE CET INSTRUMENT TRANCHE : la camera suit-elle UN joueur ? L'appariement se fait par les
// ANGLES, pas par la position : en premiere personne, la camera d'un joueur EST sa visee — or la
// visee de chaque joueur est deja decodee et validee par l'oracle du kill (i21, `AimHeadingDeg` /
// `AimPitchDeg`, r = 0,93-0,97). Si un offset de bits fait coincider (cap, tangage) du paquet avec
// la visee d'UN joueur, paquet apres paquet, l'attribution est prouvee sans rien supposer de
// l'enveloppe.
//
// BALAYAGE D'OFFSET AUTO-VERROUILLANT (recette du depot, cf. l'i21 de tmp_aimsweep2) : la
// position qui precede les angles est de largeur inconnue, donc l'offset des angles l'est aussi.
// On essaie TOUS les offsets de bits candidats ; le vrai se reconnait tout seul.
//
// SEUILS, ECRITS AVANT LA MESURE :
//
//	accord paquet<->joueur : |Δcap| < 10° (circulaire) ET |Δtangage| < 7°, sur l'echantillon
//	                         i21 du joueur le plus proche dans le temps (fenetre ±150 ms) ;
//	offset RETENU           : >= 60 % des paquets accordes a au moins un joueur, ET au moins
//	                         2 fois le score du meilleur offset hors de son voisinage ±4 bits ;
//	sans offset retenu      : NEGATIF publie — les angles ne sont pas la, ou pas sous cette
//	                         forme ; l'attribution par position restera a faire.
//
// SOUS GARDE (CAM_FILM). Prend le verrou de decodage du paquet filmdec : contrairement aux
// instruments de la phase 4, celui-ci appelle ScanFilmBipedPositions (globaux de decodage).
//
// USAGE (depuis apps/go-api) :
//
//	CGO_ENABLED=0 CAM_FILM=<repo>/data/cache/film_chunks/000d5950 \
//	  go test ./internal/analysis/replay/ -run TestViseeCameraPOV -v -timeout 30m

import (
	"fmt"
	"math"
	"os"
	"sort"
	"strings"
	"testing"
	"time"

	"levelup/go-api/internal/analysis/filmdec"
)

const (
	camFilmEnv = "CAM_FILM"

	camAngleBits     = 20
	camPitchMinRad   = -1.49225652217865
	camPitchMaxRad   = 1.49225652217865
	camYawMaxRad     = 6.2831854820251465
	camAccordCapDeg  = 10.0
	camAccordTangDeg = 7.0
	camFenetreMS     = 150
	camOffsetScore   = 0.60
	camOffsetDomin   = 2.0
	camOffsetMin     = 8   // avant : le type 7 bits + le drapeau de variante
	camOffsetMax     = 160 // au-dela : la tete des paquets observes est passee
)

// camPaquet est UN paquet type 97 : l'instant et le payload brut.
type camPaquet struct {
	tMS int64
	pay []byte
}

// camVisee est un echantillon de visee d'UN joueur : l'instant, le slot et les deux angles.
type camVisee struct {
	tMS      int64
	slot     uint32
	cap, tng float64
}

// TestViseeCameraPOV mesure l'attribution angulaire sur UN film.
func TestViseeCameraPOV(t *testing.T) {
	dir := os.Getenv(camFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument saute", camFilmEnv)
	}
	release := filmdec.LockProcessDecode()
	defer release()

	scan := filmdec.DefaultScanFilmOptions()
	scan.CaptureDirs = true
	scan.QuantaOnly = true
	debut := time.Now()
	pos, err := filmdec.ScanFilmBipedPositions(dir, scan)
	if err != nil {
		t.Fatalf("balayage des positions : %v", err)
	}
	visees := camVisees(pos)
	paquets := camPaquets(dir)
	t.Logf("COUT — %d visees i21, %d paquets type 97 en %s",
		len(visees), len(paquets), time.Since(debut).Round(time.Millisecond))
	if len(paquets) < 30 || len(visees) == 0 {
		t.Fatalf("population insuffisante : %d paquets, %d visees", len(paquets), len(visees))
	}

	scores := make([]float64, camOffsetMax+1)
	for o := camOffsetMin; o <= camOffsetMax; o++ {
		scores[o] = camScoreOffset(paquets, visees, o)
	}
	best := camOffsetMin
	for o := camOffsetMin; o <= camOffsetMax; o++ {
		if scores[o] > scores[best] {
			best = o
		}
	}
	horsVoisinage := 0.0
	for o := camOffsetMin; o <= camOffsetMax; o++ {
		if (o < best-4 || o > best+4) && scores[o] > horsVoisinage {
			horsVoisinage = scores[o]
		}
	}
	t.Logf("OFFSET v1 (accord strict cap+tangage) — meilleur o=%d : %5.1f %% ; hors ±4 bits :"+
		" %5.1f %%", best, 100*scores[best], 100*horsVoisinage)
	if scores[best] >= camOffsetScore && scores[best] >= camOffsetDomin*horsVoisinage {
		camRapport(t, paquets, visees, best)
		return
	}
	t.Log("OFFSET v1 non retenu — passage au score v2 (tangage seul + concentration de l'ecart" +
		" de cap, insensible a la convention de zero du cap)")
	camScoreV2(t, paquets, visees)
}

// camScoreV2 : pour chaque offset, chaque paquet est apparie par TANGAGE SEUL (|Δ| < 3°, meme
// convention des deux cotes : 0 = a plat) ; l'ecart de CAP vers ce joueur est ensuite verse dans
// un histogramme a 5°. Au bon offset, cet ecart est CONSTANT (le decalage de convention, quel
// qu'il soit) : le score est la fraction de paquets dans le bin modal. Un offset faux disperse
// l'ecart uniformement (~5/360 = 1,4 %).
func camScoreV2(t *testing.T, paquets []camPaquet, visees []camVisee) {
	t.Helper()
	type resultat struct {
		o, accords, modal int
		delta             float64
	}
	var meilleurs []resultat
	for o := camOffsetMin; o <= camOffsetMax; o++ {
		hist := map[int]int{}
		accords := 0
		for _, p := range paquets {
			tng, cap, ok := camAngles(p.pay, o)
			if !ok {
				continue
			}
			i := sort.Search(len(visees), func(i int) bool { return visees[i].tMS >= p.tMS-camFenetreMS })
			bestD, bestCap, trouve := 3.0, 0.0, false
			for ; i < len(visees) && visees[i].tMS <= p.tMS+camFenetreMS; i++ {
				d := math.Abs(visees[i].tng - tng)
				if d < bestD {
					bestD, bestCap, trouve = d, visees[i].cap, true
				}
			}
			if !trouve {
				continue
			}
			accords++
			dc := math.Mod(cap-bestCap+360, 360)
			hist[int(dc/5)]++
		}
		modal := 0
		delta := 0.0
		for bin, n := range hist {
			if n > modal {
				modal, delta = n, float64(bin*5)+2.5
			}
		}
		meilleurs = append(meilleurs, resultat{o, accords, modal, delta})
	}
	sort.Slice(meilleurs, func(i, j int) bool { return meilleurs[i].modal > meilleurs[j].modal })
	for i := 0; i < 5 && i < len(meilleurs); i++ {
		r := meilleurs[i]
		frac := 0.0
		if r.accords > 0 {
			frac = float64(r.modal) / float64(r.accords)
		}
		t.Logf("  v2 o=%3d : %4d apparies par tangage, bin modal de l'ecart de cap = %3d"+
			" (%4.1f %% des apparies) a Δ=%5.1f°", r.o, r.accords, r.modal, 100*frac, r.delta)
	}
	tete := meilleurs[0]
	if tete.accords >= 100 && float64(tete.modal) >= 0.30*float64(tete.accords) {
		t.Logf("VERDICT v2 — CONCENTRATION TROUVEE a o=%d (Δ de convention %.1f°) : la camera"+
			" suit bien la visee d'un joueur ; verrouiller l'offset et industrialiser"+
			" l'attribution.", tete.o, tete.delta)
	} else {
		t.Log("VERDICT v2 — PAS DE CONCENTRATION : meme par tangage seul, aucun offset ne fait" +
			" coller la camera a la visee d'un joueur. Soit l'offset varie paquet par paquet" +
			" (position a largeur variable en tete), soit cette camera ne suit pas les joueurs." +
			" Prochaine voie : decoder la POSITION (largeur runtime DAT_144632be0) et apparier" +
			" par distance aux bipedes.")
	}
}

// camVisees extrait les echantillons i21 par instant, tries.
func camVisees(pos []filmdec.BipedPosition) []camVisee {
	var out []camVisee
	for _, p := range pos {
		if !p.HasYaw {
			continue
		}
		capDeg, _ := p.AimHeadingDeg()
		tngDeg, _ := p.AimPitchDeg()
		out = append(out, camVisee{
			tMS: int64(p.TimestampUS / 1000), slot: p.Slot,
			cap: float64(capDeg), tng: float64(tngDeg),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].tMS < out[j].tMS })
	return out
}

// camPaquets rend les paquets type 97 du film (payload copie).
func camPaquets(dir string) []camPaquet {
	n := filmdec.CountFilmChunks(dir)
	var out []camPaquet
	for c := 1; c <= n; c++ {
		chunk, err := filmdec.ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, p := range filmdec.WalkPackets(chunk) {
			if p.Type != filmdec.PacketTypeDelta || p.Size < 4 {
				continue
			}
			pay := p.Payload(chunk)
			if int(pay[0]>>1) != 97 {
				continue
			}
			cp := camPaquet{tMS: int64(p.TimestampUS / 1000), pay: append([]byte(nil), pay...)}
			out = append(out, cp)
		}
	}
	return out
}

// camAngles decode (tangage, cap) en degres a l'offset o, ok=false si le payload est trop court.
// L'ordre du flux est TANGAGE puis CAP (l'ordre des deux appels du deser).
func camAngles(pay []byte, o int) (tngDeg, capDeg float64, ok bool) {
	if (o+2*camAngleBits+7)/8 > len(pay) {
		return 0, 0, false
	}
	qt := filmdec.ReadBitsAtForDiag(pay, o, camAngleBits)
	qc := filmdec.ReadBitsAtForDiag(pay, o+camAngleBits, camAngleBits)
	span := float64(int64(1)<<camAngleBits - 1)
	tng := camPitchMinRad + (camPitchMaxRad-camPitchMinRad)*float64(qt)/span
	cap := camYawMaxRad * float64(qc) / span
	return tng * 180 / math.Pi, cap * 180 / math.Pi, true
}

// camScoreOffset rend la fraction de paquets accordes a au moins un joueur pour un offset.
func camScoreOffset(paquets []camPaquet, visees []camVisee, o int) float64 {
	accord := 0
	for _, p := range paquets {
		if _, _, ok := camAccorde(p, visees, o); ok {
			accord++
		}
	}
	return float64(accord) / float64(len(paquets))
}

// camAccorde cherche le joueur dont la visee contemporaine colle aux angles du paquet.
func camAccorde(p camPaquet, visees []camVisee, o int) (uint32, float64, bool) {
	tng, cap, ok := camAngles(p.pay, o)
	if !ok {
		return 0, 0, false
	}
	i := sort.Search(len(visees), func(i int) bool { return visees[i].tMS >= p.tMS-camFenetreMS })
	bestSlot, bestEcart, trouve := uint32(0), math.MaxFloat64, false
	for ; i < len(visees) && visees[i].tMS <= p.tMS+camFenetreMS; i++ {
		v := visees[i]
		dc := math.Abs(v.cap - cap)
		if dc > 180 {
			dc = 360 - dc
		}
		dt := math.Abs(v.tng - tng)
		if dc < camAccordCapDeg && dt < camAccordTangDeg && dc+dt < bestEcart {
			bestSlot, bestEcart, trouve = v.slot, dc+dt, true
		}
	}
	return bestSlot, bestEcart, trouve
}

// camRapport publie l'attribution paquet par paquet a l'offset retenu.
func camRapport(t *testing.T, paquets []camPaquet, visees []camVisee, o int) {
	t.Helper()
	parSlot := map[uint32]int{}
	accordes := 0
	var instants []string
	for _, p := range paquets {
		slot, _, ok := camAccorde(p, visees, o)
		if !ok {
			continue
		}
		accordes++
		parSlot[slot]++
		if len(instants) < 12 {
			instants = append(instants, fmt.Sprintf("%d->slot %d", p.tMS, slot))
		}
	}
	t.Logf("ATTRIBUTION — offset %d : %d/%d paquets accordes a un joueur", o, accordes, len(paquets))
	type sc struct {
		s uint32
		n int
	}
	var l []sc
	for s, n := range parSlot {
		l = append(l, sc{s, n})
	}
	sort.Slice(l, func(i, j int) bool { return l[i].n > l[j].n })
	var sb strings.Builder
	for i, e := range l {
		if i == 10 {
			break
		}
		fmt.Fprintf(&sb, " slot%d=%d", e.s, e.n)
	}
	t.Logf("  repartition par slot (top 10) :%s", sb.String())
	t.Logf("  premiers accords : %s", strings.Join(instants, " · "))
	t.Log("LECTURE — une repartition sur PLUSIEURS slots = la camera suit successivement" +
		" plusieurs joueurs (flux POV multi-sujets) ; un slot unique = une seule camera suivie." +
		" Dans les deux cas l'attribution angulaire fonctionne, et la mesure des TROUS du flux" +
		" pendant les periodes zoomees devient possible par joueur.")
}
