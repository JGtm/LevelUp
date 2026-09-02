package filmdec

// equipment_state_test.go — INSTRUMENT DE MESURE de l'ENTITÉ D'ÉQUIPEMENT (ti=37).
//
// LA QUESTION POSÉE, en trois morceaux qui se répondent séparément :
//  1. sait-on DATER l'activation d'un équipement ?   -> transitions d'`equipment-activated`
//  2. sait-on QUI l'active ?                          -> `equipment-creator`, et son contrôle
//  3. sait-on OÙ l'objet est posé ?                   -> ScanFilmWorldObjects(ti=37)
//
// POURQUOI CE CANAL PLUTÔT QU'`i56`/`i57`. Les deux canaux BIPÈDE mesurés les 13 et 14/08
// ne tranchent pas : `i57` (l'état actif, R(2)) bouge 1 257 fois sur 1 414 lectures mais
// aucune de ses quatre valeurs n'est nommée, et son association aux épisodes `i54` ne vaut
// que 2,1x les témoins décalés ; `i56` (l'énergie) n'avait été lu que sur 176 records d'UN
// SEUL film. Le film porte pourtant une ENTITÉ DU MONDE dédiée, dont les composants sont
// nommés en clair dans le registre — et dont les quatre désers JETAIENT leurs valeurs
// (corrigé par equipment_state.go, même défaut qu'i48 le 14/08).
//
// LES CONTRÔLES QUI PEUVENT ÉCHOUER, ÉNONCÉS AVANT LA MESURE :
//
//	C1 (prescrit) — le R(5) d'`equipment-creator` doit tomber dans la BANDE DES SLOTS DE
//	   BIPÈDE du même film. S'il en sort, ce n'est pas un identifiant de bipède et il ne
//	   faut pas le présenter comme tel. Réserve écrite d'avance : le champ ne fait que
//	   5 bits (0..31) alors que les slots de bipède se comptent en milliers — le contrôle
//	   a donc toutes les chances d'échouer PAR CONSTRUCTION, et c'est le résultat qui sera
//	   publié, pas une réinterprétation de la question.
//	C2 (plus faible, dit comme tel) — à défaut d'être un slot, le champ pourrait être un
//	   INDEX COMPACT de joueur : ses valeurs seraient alors bornées par le nombre de slots
//	   de bipède du film. Un champ qui déborde cette borne n'est ni l'un ni l'autre.
//	C3 (géométrie) — les positions de ti=37 doivent tomber DANS l'emprise du nuage des
//	   bipèdes du même film, en coordonnées NORMALISÉES [0,1] de l'AABB de la carte. Un
//	   nuage collé à l'origine, ou débordant largement le nuage des joueurs, dirait que la
//	   déquantification est fausse. Ce contrôle n'exige AUCUNE base : les deux nuages sont
//	   exprimés dans le même repère quantifié, lu dans le film.
//
// LECTURE SEULE, gardé par EQUIP_FILM, sauté partout ailleurs (CI comprise). UN SEUL
// décodage filmdec par process (globaux de paquet) : le verrou est pris pour tout le test.
//
// USAGE (depuis apps/go-api) :
//
//	CGO_ENABLED=0 EQUIP_FILM=<repo>/data/cache/film_chunks/000d5950 \
//	  go test ./internal/analysis/filmdec/ -run '^TestEquipmentEntityState$' -timeout 30m -v

import (
	"fmt"
	"os"
	"sort"
	"testing"
)

const equipFilmEnv = "EQUIP_FILM"

func TestEquipmentEntityState(t *testing.T) {
	dir := os.Getenv(equipFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument de mesure sauté", equipFilmEnv)
	}
	release := LockProcessDecode()
	defer release()

	arch := equipLogArchetype(t, dir)
	lay := equipSetPrecision(t, dir)

	samples, st, err := ScanFilmEquipmentState(dir)
	if err != nil {
		t.Fatalf("balayage de l'état d'équipement impossible : %v", err)
	}
	t.Logf("MESURE A1 — RECORDS delta ti=%d %d · slots retenus %d · masque∋(un des 4) %d "+
		"· marche ABOUTIE %d · marche CASSÉE %d",
		EquipmentTypeIndex, st.Records, st.Slots, st.WithAny, st.Walked, st.Broken)
	for f := 0; f < EquipmentFieldCount; f++ {
		fl := EquipmentField(f)
		t.Logf("  %-32s i%-2d · masque %6d · LU %6d · porte fermée %6d",
			fl, arch.indicesOfFirst(fl.String()), st.WithField[f], st.Read[f], st.Gated[f])
	}
	equipLogCensus(t, arch, st)
	equipAssertNewFieldsSeen(t, st)
	if len(samples) == 0 {
		t.Log("VERDICT A : aucune lecture exploitable — ni datation ni auteur par ce canal")
		return
	}
	equipLogValues(t, samples)
	equipLogTransitions(t, samples)
	equipLogDatableEvents(t, samples)
	equipCreatorControl(t, dir, samples)
	equipPositions(t, dir, lay)
}

// indicesOfFirst rend le premier index d'itérateur portant ce nom, ou -1.
func (a Archetype) indicesOfFirst(name string) int {
	if ids := a.indicesOf(name); len(ids) > 0 {
		return ids[0]
	}
	return -1
}

// equipLogArchetype publie la liste ORDONNÉE des composants de ti=37 telle que le film la
// déclare. C'est la pièce : les index cités par le plan sont vérifiés ici, pas supposés.
func equipLogArchetype(t *testing.T, dir string) Archetype {
	t.Helper()
	arch, err := EquipmentArchetype(dir)
	if err != nil {
		t.Fatalf("archétype ti=%d illisible : %v", EquipmentTypeIndex, err)
	}
	t.Logf("== ARCHÉTYPE ti=%d : %d composants ==", EquipmentTypeIndex, len(arch.Components))
	for i, c := range arch.Components {
		t.Logf("  i%-2d %s", i, c)
	}
	return arch
}

// equipSetPrecision installe les largeurs d'axe LUES DANS LE FILM pour le chemin
// world-object, et les restaure à la sortie.
//
// DÉCOUVERTE À NOTER, NON TRAITÉE ICI : aucun chemin de production n'appelle
// SetWorldObjectPrecisionFromLayout — le défaut {13,13,14} (largeurs de Cliffhanger) sert
// donc partout, y compris sur les cartes dont DetectI0Layout mesure d'autres largeurs.
func equipSetPrecision(t *testing.T, dir string) I0Layout {
	t.Helper()
	lay, _, err := DetectI0Layout(dir)
	if err != nil {
		t.Fatalf("découpage i0 illisible dans %s : %v", dir, err)
	}
	prev := WorldObjectPrecision
	t.Cleanup(func() { WorldObjectPrecision = prev })
	SetWorldObjectPrecisionFromLayout(lay)
	t.Logf("largeurs d'axe LUES DANS LE FILM : %v (défaut du paquet : %v)",
		lay.AxisW, prev.AxisW)
	return lay
}

// equipLogCensus publie, pour CHAQUE composant de l'archétype, le nombre de records dont le
// masque l'annonce. C'est le recensement qui dit où le signal se trouve — et il ne coûte
// aucun décodage de valeur : un composant jamais annoncé ne portera jamais rien.
func equipLogCensus(t *testing.T, arch Archetype, st EquipmentStateStats) {
	t.Logf("== MESURE A1 — RECENSEMENT AU MASQUE des %d composants de ti=%d "+
		"(dénominateur : %d records) ==", len(arch.Components), EquipmentTypeIndex, st.Records)
	for i, name := range arch.Components {
		if i >= worldObjectMaxComponent || st.MaskCensus[i] == 0 {
			continue
		}
		t.Logf("  i%-2d %-52s %6d (%.3f %%)", i, name, st.MaskCensus[i],
			100*float64(st.MaskCensus[i])/float64(st.Records))
	}
	var absents []string
	for i, name := range arch.Components {
		if i < worldObjectMaxComponent && st.MaskCensus[i] == 0 {
			absents = append(absents, fmt.Sprintf("i%d %s", i, name))
		}
	}
	t.Logf("  JAMAIS annoncés (%d) : %v", len(absents), absents)
}

// equipLogDatableEvents répond à la question « sait-on DATER quelque chose ? » avec un
// dénominateur : combien de VIES d'objet portent, au moins une fois, une valeur de chaque
// champ — et à quel instant cette première valeur arrive dans la vie de l'objet.
//
// UNE PREMIÈRE TRANSMISSION N'EST PAS UN ÉVÉNEMENT. Un champ transmis une seule fois par vie
// date la RÉPLICATION du champ, pas le geste du joueur : sans une seconde transmission de
// valeur différente, rien ne distingue « l'objet vient d'être activé » de « l'objet est
// répliqué pour la première fois, dans l'état où il se trouve ». C'est pourquoi ce bloc et
// celui des transitions se lisent ensemble.
func equipLogDatableEvents(t *testing.T, samples []EquipmentStateSample) {
	type key struct{ slot, gen uint32 }
	first := map[key]map[int]uint64{}
	birth := map[key]uint64{}
	for _, s := range samples {
		k := key{s.Slot, s.Gen}
		if b, ok := birth[k]; !ok || s.TimestampUS < b {
			birth[k] = s.TimestampUS
		}
		for f := 0; f < EquipmentFieldCount; f++ {
			if !s.Present[f] {
				continue
			}
			if first[k] == nil {
				first[k] = map[int]uint64{}
			}
			if ts, ok := first[k][f]; !ok || s.TimestampUS < ts {
				first[k][f] = s.TimestampUS
			}
		}
	}
	t.Logf("== MESURE A2 — PREMIÈRE VALEUR PAR VIE D'OBJET (%d vies) ==", len(birth))
	for f := 0; f < EquipmentFieldCount; f++ {
		n, late := 0, 0
		for k, m := range first {
			ts, ok := m[f]
			if !ok {
				continue
			}
			n++
			if ts > birth[k] {
				late++
			}
		}
		t.Logf("  %-32s valeur transmise dans %d vies sur %d · dont %d APRÈS le premier "+
			"record de la vie", EquipmentField(f), n, len(birth), late)
	}
}

// equipLogValues publie la distribution des valeurs transmises, champ par champ. Sans elle,
// un nombre de transitions ne veut rien dire : un champ à valeur unique n'en produit aucune.
func equipLogValues(t *testing.T, samples []EquipmentStateSample) {
	t.Log("== MESURE A2 — DISTRIBUTION DES VALEURS TRANSMISES ==")
	for f := 0; f < EquipmentFieldCount; f++ {
		hist := map[uint64]int{}
		n := 0
		for _, s := range samples {
			if s.Present[f] {
				hist[s.Val[f]]++
				n++
			}
		}
		if n == 0 {
			t.Logf("  %-32s aucune valeur transmise", EquipmentField(f))
			continue
		}
		t.Logf("  %-32s %d valeurs · %s", EquipmentField(f), n, equipRenderU64(hist))
	}
}

// equipLogTransitions compte les CHANGEMENTS DE VALEUR par vie d'objet (slot, génération).
// C'est la transition qui date un événement, pas la valeur : un champ constant sur toute la
// vie de l'objet ne date rien, même lu des milliers de fois.
func equipLogTransitions(t *testing.T, samples []EquipmentStateSample) {
	type key struct{ slot, gen uint32 }
	series := map[key][]EquipmentStateSample{}
	for _, s := range samples {
		k := key{s.Slot, s.Gen}
		series[k] = append(series[k], s)
	}
	var trans, pairs [EquipmentFieldCount]int
	rise := map[int]int{}
	for _, ss := range series {
		sort.Slice(ss, func(a, b int) bool { return ss[a].TimestampUS < ss[b].TimestampUS })
		for i := 1; i < len(ss); i++ {
			for f := 0; f < EquipmentFieldCount; f++ {
				if !ss[i-1].Present[f] || !ss[i].Present[f] {
					continue
				}
				pairs[f]++
				if ss[i-1].Val[f] != ss[i].Val[f] {
					trans[f]++
					if ss[i].Val[f] > ss[i-1].Val[f] {
						rise[f]++
					}
				}
			}
		}
	}
	t.Logf("== MESURE A2 — TRANSITIONS, sur %d vies d'objet (slot, génération) ==", len(series))
	for f := 0; f < EquipmentFieldCount; f++ {
		if pairs[f] == 0 {
			t.Logf("  %-32s aucune paire consécutive : transition non calculable", EquipmentField(f))
			continue
		}
		t.Logf("  %-32s %d transitions sur %d paires consécutives (%.2f %%), dont %d en hausse",
			EquipmentField(f), trans[f], pairs[f],
			100*float64(trans[f])/float64(pairs[f]), rise[f])
	}
}

// equipCreatorControl joue les contrôles C1 et C2 énoncés en tête de fichier.
func equipCreatorControl(t *testing.T, dir string, samples []EquipmentStateSample) {
	n := CountFilmChunks(dir)
	chunks := make([]int, 0, n)
	for i := 1; i <= n; i++ {
		chunks = append(chunks, i)
	}
	band := bipedSlotBandDir(dir, chunks)
	lo, hi := uint32(1)<<31, uint32(0)
	for s := range band {
		if s < lo {
			lo = s
		}
		if s > hi {
			hi = s
		}
	}
	var vals []uint64
	hist := map[uint64]int{}
	for _, s := range samples {
		if s.Present[EquipCreator] {
			vals = append(vals, s.Val[EquipCreator])
			hist[s.Val[EquipCreator]]++
		}
	}
	t.Logf("== MESURE A4 — CONTRÔLE DU CHAMP `creator` ==")
	t.Logf("  bande de slots de BIPÈDE du film : %d slots, de %d à %d", len(band), lo, hi)
	if len(vals) == 0 {
		t.Log("  C1/C2 NON CALCULABLES : aucune valeur de creator transmise")
		return
	}
	inBand := 0
	maxVal := uint64(0)
	for _, v := range vals {
		if band[uint32(v)] {
			inBand++
		}
		if v > maxVal {
			maxVal = v
		}
	}
	t.Logf("  C1 (prescrit) : %d valeurs sur %d tombent dans la bande des slots de bipède "+
		"-> %s", inBand, len(vals), equipVerdict(inBand == len(vals)))
	t.Logf("  C2 (plus faible) : valeur maximale %d, pour %d slots de bipède dans le film "+
		"-> %s", maxVal, len(band), equipVerdict(int(maxVal) < len(band)))
	t.Logf("  distribution des valeurs de creator : %s", equipRenderU64(hist))
}

func equipVerdict(ok bool) string {
	if ok {
		return "CONTRÔLE PASSÉ"
	}
	return "CONTRÔLE ÉCHOUÉ"
}

// equipPositions décode les trajectoires de ti=37 en coordonnées NORMALISÉES [0,1] de l'AABB
// de la carte, et les confronte au nuage des bipèdes du même film (contrôle C3).
//
// POURQUOI DES COORDONNÉES NORMALISÉES ET PAS MONDE : les bornes monde exigent le nom de la
// carte, donc une base. La normalisation par l'AABB suffit au contrôle demandé — les deux
// nuages sont quantifiés sur le MÊME repère, lu dans le film.
func equipPositions(t *testing.T, dir string, lay I0Layout) {
	unit := Vec3Range{{Min: 0, Max: 1}, {Min: 0, Max: 1}, {Min: 0, Max: 1}}
	tracks, err := ScanFilmWorldObjects(dir, &unit, EquipmentTypeIndex)
	if err != nil {
		t.Logf("MESURE A3 : trajectoires ti=%d indisponibles : %v", EquipmentTypeIndex, err)
		return
	}
	pts := 0
	var box equipBox
	for _, tr := range tracks {
		pts += len(tr.Pts)
		for _, p := range tr.Pts {
			box.add([3]float32{p.X, p.Y, p.Z})
		}
	}
	t.Logf("== MESURE A3 — POSITIONS DES OBJETS ti=%d (coordonnées normalisées [0,1]) ==",
		EquipmentTypeIndex)
	t.Logf("  %d trajectoires · %d échantillons · emprise %s", len(tracks), pts, box)
	if pts == 0 {
		return
	}
	bip := equipBipedBox(t, dir, lay)
	if bip.n == 0 {
		t.Log("  C3 NON CALCULABLE : aucun nuage de bipède à confronter")
		return
	}
	t.Logf("  nuage des BIPÈDES du même film : %d positions · emprise %s", bip.n, bip)
	in := 0
	for _, tr := range tracks {
		for _, p := range tr.Pts {
			if bip.contains([3]float32{p.X, p.Y, p.Z}) {
				in++
			}
		}
	}
	t.Logf("  C3 : %d échantillons d'équipement sur %d tombent dans l'emprise des bipèdes "+
		"(%.1f %%) -> %s", in, pts, 100*float64(in)/float64(pts),
		equipVerdict(in > pts*9/10))
}

// equipBipedBox lit le nuage des bipèdes en QUANTA et le normalise par 2^W, axe par axe —
// le même repère que celui des objets du monde.
func equipBipedBox(t *testing.T, dir string, lay I0Layout) equipBox {
	t.Helper()
	opt := DefaultScanFilmOptions()
	opt.QuantaOnly = true
	pos, err := ScanFilmBipedPositions(dir, opt)
	if err != nil {
		t.Logf("  nuage des bipèdes indisponible : %v", err)
		return equipBox{}
	}
	var box equipBox
	for _, p := range pos {
		var v [3]float32
		for a := 0; a < 3; a++ {
			v[a] = (float32(p.Q[a]) + 0.5) / float32(uint64(1)<<lay.AxisW[a])
		}
		box.add(v)
	}
	return box
}

// equipBox est l'emprise d'un nuage de points normalisés.
type equipBox struct {
	n        int
	min, max [3]float32
}

func (b *equipBox) add(v [3]float32) {
	if b.n == 0 {
		b.min, b.max = v, v
	}
	for a := 0; a < 3; a++ {
		if v[a] < b.min[a] {
			b.min[a] = v[a]
		}
		if v[a] > b.max[a] {
			b.max[a] = v[a]
		}
	}
	b.n++
}

// contains teste l'appartenance à l'emprise, avec une marge de 5 % de l'étendue par axe :
// un objet posé contre un mur peut sortir de quelques centimètres du nuage des pas.
func (b equipBox) contains(v [3]float32) bool {
	for a := 0; a < 3; a++ {
		m := 0.05 * (b.max[a] - b.min[a])
		if v[a] < b.min[a]-m || v[a] > b.max[a]+m {
			return false
		}
	}
	return true
}

func (b equipBox) String() string {
	if b.n == 0 {
		return "(vide)"
	}
	return fmt.Sprintf("X[%.3f %.3f] Y[%.3f %.3f] Z[%.3f %.3f]",
		b.min[0], b.max[0], b.min[1], b.max[1], b.min[2], b.max[2])
}

// equipHistCap borne l'affichage d'un histogramme. 64 : la plus large plage qu'un champ de
// ce fichier puisse couvrir sans être un compteur (`creator` est sur 5 bits, `activated` sur
// 3). Une valeur plus basse TRONQUERAIT silencieusement la queue d'un champ de 5 bits — le
// défaut corrigé le 2026-08-15, après qu'un premier balayage eut publié une distribution
// amputée sans le dire.
const equipHistCap = 64

func equipRenderU64(h map[uint64]int) string {
	keys := make([]uint64, 0, len(h))
	for k := range h {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(a, b int) bool { return keys[a] < keys[b] })
	out := ""
	for i, k := range keys {
		if i == equipHistCap {
			out += fmt.Sprintf(" … TRONQUÉ, %d valeurs distinctes au total", len(keys))
			break
		}
		if out != "" {
			out += " "
		}
		out += fmt.Sprintf("%d:%d", k, h[k])
	}
	return out
}

// equipAssertNewFieldsSeen — LES DEUX CANAUX AJOUTÉS AU LOT 0 SONT-ILS VRAIMENT LÀ ?
//
// POURQUOI UNE ASSERTION ET PAS UNE LIGNE DE JOURNAL. i26 (`energy-delay-ticks-left`) et i27
// (`charges-remaining`) ont été ajoutés au hook par la plomberie de publication (lot 0 item
// 0.6) sur la promesse qu'ils sont les DEUX CANAUX LES PLUS BAVARDS de l'archétype. Une
// promesse qui ne peut pas échouer n'est pas une mesure : si un jour ces deux champs cessent
// d'être lus — parce que le désérialiseur a été débranché, ou que la marche s'arrête avant eux
// — le journal continuerait d'afficher « 0 » sans que personne ne le remarque.
//
// MESURE DE RÉFÉRENCE (`000d5950`, 2026-08-17) : i26 820 lectures pour 629 vies d'objet, i27
// 883 pour 599 vies — contre 129 / 62 / 33 / 48 pour les quatre champs d'origine. Le plancher
// est volontairement à UN : le compte exact dépend du film, la PRÉSENCE ne doit pas en dépendre.
func equipAssertNewFieldsSeen(t *testing.T, st EquipmentStateStats) {
	t.Helper()
	for _, f := range []EquipmentField{EquipEnergyDelay, EquipCharges} {
		if st.WithField[f] == 0 {
			t.Errorf("%s : AUCUN record ne l'annonce au masque — soit le film ne le porte pas, "+
				"soit l'index a été résolu sur un autre nom", f)
			continue
		}
		if st.Read[f] == 0 {
			t.Errorf("%s : annoncé %d fois au masque mais JAMAIS lu — la marche s'arrête avant "+
				"lui, ou le déser ne publie plus", f, st.WithField[f])
		}
	}
}
