package replay

// ground_weapon_rules.go — LES RÈGLES de la chaîne des ARMES AU SOL : ce qu'est une apparition,
// comment elle se classe, comment les apparitions font un SOCLE, comment une disparition se
// BORNE et se DATE, et quand un CYCLE est établi.
//
// D'OÙ ELLES VIENNENT. Elles ont été écrites AVANT chaque mesure du plan
// `.ai/V7.5/replay2d/PLAN_ARMES_AU_SOL_2E_LECTURE.md` (phases 1 et 2, 2026-08-17) et n'ont pas
// bougé depuis : les seuils du plan ne se rebaissent pas après coup. Elles vivaient dans les
// instruments sous garde ; la phase 3 les amène en production sans les recopier — les
// instruments les APPELLENT désormais. Une seconde copie de la règle de grappe ou du verdict de
// stabilité aurait divergé au premier correctif, et c'est exactement ce que la règle des deux
// copies du dépôt interdit.
//
// LE PRÉFIXE `gw` (ground weapon) EST LE VOCABULAIRE DU CHANTIER : `gwPad*` pour ce qui fait un
// socle, `gwPickup*` pour ce qui fait une disparition. Il est gardé tel quel pour que les
// mesures publiées au plan (lignes `PAD`, `PICKUP`, `PADSTATE`, `PADCYCLE`) désignent les mêmes
// fonctions que le code qui publie l'artefact.
//
// PUR (aucune I/O, aucun accès film) : le décodage vit dans `filmdec`, l'assemblage dans
// `ground_weapon_pads.go`.

import (
	"fmt"
	"math"
	"sort"

	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/analysis/weaponv3"
)

// LES SEUILS, ÉCRITS AVANT LA MESURE (plan, items 1.2-1.4, 2.1, et décisions 2-4).

// gwPadRadiusM est le rayon d'agglomération d'une grappe, en mètres : deux apparitions de MÊME
// famille à moins d'un mètre sont au MÊME socle. Un mètre est la tolérance du plan ; c'est aussi
// l'ordre de grandeur du quantum de position sur les cartes d'arène.
const gwPadRadiusM = 1.0

// gwPadMinHits est le nombre d'apparitions en deçà duquel une grappe n'est PAS un socle. Deux,
// parce qu'un socle est une RÉCURRENCE (décision 2) : une apparition unique ne récurre pas.
const gwPadMinHits = 2

// gwPadCycleMinGaps est le nombre d'écarts en deçà duquel un cycle ne se juge pas. Deux écarts
// (donc trois occupations) : un écart unique n'a pas d'écart-type, et le publier stable serait
// une affirmation sans mesure.
const gwPadCycleMinGaps = 2

// gwPadCycleMaxCV est l'écart-type maximal, en fraction de la médiane, pour qu'un cycle soit
// ÉTABLI (décision 3 : « écart-type > 20 % de la médiane » -> cycle non établi).
const gwPadCycleMaxCV = 0.20

// gwPickupTrackTolUS est la tolérance d'appariement entre un record de CRÉATION et la piste
// delta de la même vie : 200 ms, soit `originDropWindowUS`. Le record de création porte lui-même
// la position i0, donc la piste commence en principe au MÊME paquet ; la tolérance couvre le cas
// où le premier point répliqué suit d'une frame ou deux.
const gwPickupTrackTolUS = originDropWindowUS

// gwPadKindWeapon / gwPadKindPowerup : les deux natures d'apparition que la chaîne sait mesurer.
//
// LA PRODUCTION NE PUBLIE QUE LES ARMES, et le second n'est pas mort pour autant : la règle de
// grappe groupe par (nature, famille), et le NÉGATIF DE CORPUS des power-ups de socle — 5
// apparitions sur 8 films, toutes `dropped`, aucune grappe, aucun socle (plan, phase 1) — est un
// résultat que son instrument rejoue par ce même vocabulaire.
const (
	gwPadKindWeapon  = "weapon"
	gwPadKindPowerup = "powerup"
)

// gwClassDropped / gwClassSpawned : les deux classes de l'item 1.1. `dropped` réutilise le
// vocabulaire des poses d'équipement (`OriginDropped`) — c'est la MÊME règle mesurée, appliquée
// à toutes les vies de bipède au lieu du seul poseur.
const (
	gwClassDropped = OriginDropped
	gwClassSpawned = "spawned"
)

// gwPickupStatus* : les trois issues d'une occupation de socle. Vocabulaire STABLE du chantier —
// la couverture de l'artefact les publie tels quels.
const (
	gwPickupStatusDated   = "dated"   // un joueur est passé à < 1,5 m dans l'intervalle
	gwPickupStatusUnknown = "unknown" // personne n'est passé : la disparition reste bornée
	gwPickupStatusNever   = "never"   // encore recensé à la dernière image-clé du film
)

// gwPadApparition est UNE apparition d'objet au sol : où, quand, quelle famille, et comment elle
// est née.
type gwPadApparition struct {
	Kind, Family string
	X, Y, Z      float32
	TUS          uint64
	Class        string
	// HasDelta dit que la vie de l'objet a laissé des échantillons de position dans les
	// paquets delta. Son absence est le candidat « apparu au repos » : un objet qui naît
	// immobile à son socle n'émet jamais de position.
	HasDelta bool
}

// gwPadCluster est une GRAPPE : des apparitions de même nature et de même famille, à moins de
// `gwPadRadiusM` les unes des autres. Sa position est le CENTROÏDE de ses apparitions.
type gwPadCluster struct {
	Kind, Family string
	X, Y, Z      float32
	// TS porte les instants des apparitions, TRIÉS. C'est la matière du cycle.
	TS []uint64
}

// gwPadCycle est le cycle mesuré d'une grappe. `Established` est faux quand la mesure refuse de
// conclure — jamais un chiffre instable publié comme s'il était stable (décision 3 du plan).
type gwPadCycle struct {
	MedianS, P10S, P90S, SDS float64
	Gaps                     int
	Established              bool
}

// gwPadsCluster agglomère les apparitions par (nature, famille) et proximité.
//
// LA RÈGLE EST « LE PLUS PROCHE DANS LE RAYON », pas « le premier trouvé » : sur une carte
// d'arène deux socles voisins peuvent être à deux mètres l'un de l'autre, et le premier trouvé
// les fusionnerait selon l'ordre d'arrivée. L'ordre d'entrée est rendu TOTAL avant la marche
// (instant, puis position) pour que deux exécutions rendent les mêmes grappes.
func gwPadsCluster(app []gwPadApparition, radiusM float64) []gwPadCluster {
	out, _ := gwPadsClusterAssign(app, radiusM)
	return out
}

// gwPadsClusterAssign agglomère ET rend, pour chaque apparition D'ENTRÉE (même index), la
// grappe qui la porte.
//
// POURQUOI L'ASSIGNATION EXISTE. Le cycle se mesure DEPUIS LE RAMASSAGE : il faut, socle par
// socle, quel objet est apparu et quand il a disparu. Retrouver la grappe d'une apparition après
// coup en comparant sa position au centroïde serait une SECONDE règle de grappe, qui divergerait
// de celle-ci au premier correctif.
func gwPadsClusterAssign(app []gwPadApparition, radiusM float64) ([]gwPadCluster, []int) {
	ord := make([]int, len(app))
	for i := range ord {
		ord[i] = i
	}
	sort.Slice(ord, func(i, j int) bool { return gwPadsLess(app[ord[i]], app[ord[j]]) })
	assign := make([]int, len(app))
	var out []gwPadCluster
	for _, src := range ord {
		a := app[src]
		best, bestD := -1, math.Inf(1)
		for i := range out {
			if out[i].Kind != a.Kind || out[i].Family != a.Family {
				continue
			}
			d := gwPadsDist(out[i].X, out[i].Y, out[i].Z, a.X, a.Y, a.Z)
			if d <= radiusM && d < bestD {
				best, bestD = i, d
			}
		}
		if best < 0 {
			out = append(out, gwPadCluster{
				Kind: a.Kind, Family: a.Family, X: a.X, Y: a.Y, Z: a.Z, TS: []uint64{a.TUS},
			})
			assign[src] = len(out) - 1
			continue
		}
		n := float32(len(out[best].TS))
		out[best].X = (out[best].X*n + a.X) / (n + 1)
		out[best].Y = (out[best].Y*n + a.Y) / (n + 1)
		out[best].Z = (out[best].Z*n + a.Z) / (n + 1)
		out[best].TS = append(out[best].TS, a.TUS)
		assign[src] = best
	}
	for i := range out {
		sort.Slice(out[i].TS, func(a, b int) bool { return out[i].TS[a] < out[i].TS[b] })
	}
	// Les grappes sont rendues dans un ordre TOTAL (et non celui de leur découverte) : les
	// index d'assignation déjà distribués doivent suivre la permutation, sans quoi ils
	// désigneraient une grappe voisine.
	return gwPadsSortClusters(out, assign)
}

// gwPadsSortClusters range les grappes dans l'ordre total et réécrit les assignations.
func gwPadsSortClusters(out []gwPadCluster, assign []int) ([]gwPadCluster, []int) {
	perm := make([]int, len(out))
	for i := range perm {
		perm[i] = i
	}
	sort.Slice(perm, func(i, j int) bool { return gwPadsClusterLess(out[perm[i]], out[perm[j]]) })
	inv := make([]int, len(out))
	sorted := make([]gwPadCluster, len(out))
	for newIdx, oldIdx := range perm {
		inv[oldIdx] = newIdx
		sorted[newIdx] = out[oldIdx]
	}
	for i := range assign {
		assign[i] = inv[assign[i]]
	}
	return sorted, assign
}

// gwPadsLess est l'ordre TOTAL des apparitions : instant, puis nature, famille et position.
func gwPadsLess(a, b gwPadApparition) bool {
	switch {
	case a.TUS != b.TUS:
		return a.TUS < b.TUS
	case a.Kind != b.Kind:
		return a.Kind < b.Kind
	case a.Family != b.Family:
		return a.Family < b.Family
	case a.X != b.X:
		return a.X < b.X
	case a.Y != b.Y:
		return a.Y < b.Y
	}
	return a.Z < b.Z
}

// gwPadsClusterLess est l'ordre TOTAL des grappes rendues.
func gwPadsClusterLess(a, b gwPadCluster) bool {
	switch {
	case a.Kind != b.Kind:
		return a.Kind < b.Kind
	case a.Family != b.Family:
		return a.Family < b.Family
	case a.X != b.X:
		return a.X < b.X
	case a.Y != b.Y:
		return a.Y < b.Y
	}
	return a.Z < b.Z
}

// gwPadsKeep ne garde que les grappes d'au moins `min` apparitions — les SOCLES.
func gwPadsKeep(in []gwPadCluster, min int) []gwPadCluster {
	out := make([]gwPadCluster, 0, len(in))
	for _, c := range in {
		if len(c.TS) >= min {
			out = append(out, c)
		}
	}
	return out
}

// gwPadsCycle mesure le cycle d'une grappe : les écarts entre apparitions successives.
//
// L'ÉCART EST PUBLIÉ DÈS QU'IL EST MESURÉ, MAIS « ÉTABLI » EXIGE DEUX ÉCARTS. Un socle vu deux
// fois donne UN intervalle : c'est une mesure, et la taire rendrait un `0,0 s` qui se lirait
// comme un cycle nul. Ce n'est pourtant pas un cycle — un intervalle unique n'a pas d'écart-type,
// donc rien ne dit qu'il se répète. Les deux faits se publient séparément : `MedianS` porte la
// mesure, `Established` porte le verdict (décision 3 du plan).
func gwPadsCycle(ts []uint64) gwPadCycle {
	if len(ts) < 2 {
		return gwPadCycle{}
	}
	gaps := make([]float64, 0, len(ts)-1)
	for i := 1; i < len(ts); i++ {
		gaps = append(gaps, float64(ts[i]-ts[i-1])/1e6)
	}
	return gwPadsCycleFromGaps(gaps)
}

// gwPadsCycleFromGaps juge un cycle à partir des ÉCARTS déjà calculés. Le cycle publié se mesure
// du RAMASSAGE à la réapparition suivante (item 2.4 : 24 socles établis sur 57 contre 4 pour
// l'horloge d'apparition, aux mêmes règles de stabilité) : ces écarts ne sont donc PAS des
// différences d'instants successifs, et le verdict reste écrit une seule fois.
func gwPadsCycleFromGaps(gaps []float64) gwPadCycle {
	if len(gaps) == 0 {
		return gwPadCycle{}
	}
	c := gwPadCycle{
		Gaps:    len(gaps),
		MedianS: gwPadsQuantile(gaps, 0.5),
		P10S:    gwPadsQuantile(gaps, 0.10),
		P90S:    gwPadsQuantile(gaps, 0.90),
		SDS:     gwPadsStdDev(gaps),
	}
	c.Established = c.Gaps >= gwPadCycleMinGaps && c.MedianS > 0 &&
		c.SDS <= gwPadCycleMaxCV*c.MedianS
	return c
}

// gwPadsQuantile rend le quantile d'ordre q, par la MÊME convention d'index que les autres
// mesures du paquet (`origineQ`) : deux mesures du dépôt qui disent « p90 » doivent dire la même
// chose.
func gwPadsQuantile(v []float64, q float64) float64 {
	if len(v) == 0 {
		return 0
	}
	s := append([]float64(nil), v...)
	sort.Float64s(s)
	return s[int(q*float64(len(s)-1))]
}

// gwPadsStdDev rend l'écart-type de POPULATION — les écarts mesurés sont la population entière
// du cycle de ce socle, pas un échantillon tiré d'une population plus large.
func gwPadsStdDev(v []float64) float64 {
	if len(v) == 0 {
		return 0
	}
	var sum float64
	for _, x := range v {
		sum += x
	}
	m := sum / float64(len(v))
	var acc float64
	for _, x := range v {
		acc += (x - m) * (x - m)
	}
	return math.Sqrt(acc / float64(len(v)))
}

// gwPadsClass classe une apparition : `dropped` si une vie de bipède S'ACHÈVE à moins de
// `originDropWindowUS` (2 frames) et `originDropMaxDist` (1,5 m) d'elle, `spawned` sinon.
//
// LES DEUX CONSTANTES SONT CELLES DES POSES (equipment_placements.go), et pas des jumelles : la
// règle du 2026-08-18 est la MÊME règle, appliquée ici à toutes les vies au lieu du seul poseur
// mesuré — un objet apparu à son socle n'a pas de poseur à moins de 3 m, donc le chemin
// `equipmentOwner` le classerait `unknown` au lieu de `spawned`.
//
// MESURE (plan, item 1.1) : 1 275 `dropped` sur 1 790 apparitions retenues, part maximale sur un
// Super Fiesta (82,3 %) et minimale sur les arènes classiques (62,3 % et 64,9 %) — le témoin que
// le plan avait écrit. `dropped` implique une vie delta 1 275 fois sur 1 275.
func gwPadsClass(lives map[uint32][]equipLife, a gwPadApparition) string {
	for _, vs := range lives {
		for _, v := range vs {
			if equipTimeGap(a.TUS, v.to) > originDropWindowUS {
				continue
			}
			if gwPadsDist(a.X, a.Y, a.Z, v.x, v.y, v.z) < originDropMaxDist {
				return gwClassDropped
			}
		}
	}
	return gwClassSpawned
}

// gwPadsIdentity rend le mot MPP de 32 bits du record de création — l'identité de l'arme.
//
// C'EST LE FILTRE, ET IL VIENT D'UNE DÉCOUVERTE (plan, découverte 2) : le témoin FANTÔME du
// balayage de créations n'est PAS discriminant à lui seul (sur `00162144`, la bande fantôme rend
// 398 créations acceptées contre 366 pour la bande réelle). Après le filtre d'identité : 13
// fantômes croisées contre 1 785 réelles sur huit films, un facteur 137.
func gwPadsIdentity(c filmdec.EquipmentCreation) (uint32, bool) {
	if !c.MPPPresent[filmdec.MPPWord32] {
		return 0, false
	}
	return uint32(c.MPPVal[filmdec.MPPWord32]), true
}

// gwPadsWeaponFamily rend le nom CANONIQUE de l'arme — le repli des alias, sans quoi un même
// socle rendrait deux familles pour un seul canon (piège documenté par keyframe_loadout.go).
// À défaut de nom, l'identifiant brut : une famille sans nom reste une famille.
func gwPadsWeaponFamily(w uint32) string {
	if n := weaponv3.WeaponName(w); n != "" {
		return n
	}
	return fmt.Sprintf("0x%08x", w)
}

func gwPadsDist(ax, ay, az, bx, by, bz float32) float64 {
	dx, dy, dz := float64(ax-bx), float64(ay-by), float64(az-bz)
	return math.Sqrt(dx*dx + dy*dy + dz*dz)
}

// gwPickupBounds encadre la disparition d'un objet au sol : ce que le recensement des images-clés
// permet d'affirmer, et rien de plus.
type gwPickupBounds struct {
	// LowUS : dernier instant où l'objet est PROUVÉ présent (dernière image-clé qui le
	// recense, à défaut l'instant de sa création).
	LowUS uint64
	// HighUS : premier instant où son absence est PROUVÉE (première image-clé qui ne le
	// recense plus), à défaut la fin de sa vie de clé ou la fin du film.
	HighUS uint64
	// NeverPicked : l'objet est encore recensé à la DERNIÈRE image-clé du film. Il n'a pas été
	// ramassé — ou il l'a été après la dernière image-clé, ce que rien ne dit.
	NeverPicked bool
	// NoLaterKF : aucune image-clé ne suit l'instant de référence. La borne haute est alors la
	// fin du film, et l'intervalle n'est pas une mesure de recensement.
	NoLaterKF bool
	// SeenKF : nombre d'images-clés qui recensent l'objet. Zéro = ne PAS conclure qu'il n'a
	// jamais existé : il a pu naître et disparaître entre deux images-clés (24,0 % des
	// apparitions mesurées).
	SeenKF int
}

// WidthS rend la largeur de l'intervalle de bornage, en secondes.
func (b gwPickupBounds) WidthS() float64 {
	if b.HighUS <= b.LowUS {
		return 0
	}
	return float64(b.HighUS-b.LowUS) / 1e6
}

// gwPickupBoundsFrom encadre la disparition d'un objet né à `t0`, dont la clé (slot, gen) est
// reprise par une autre création à `lifeEnd` (ou `filmEnd` si elle ne l'est pas).
//
// `kfTimes` sont les instants de TOUTES les images-clés du film, triés ; `seen` ceux qui
// RECENSENT l'objet, triés, déjà restreints à [t0, lifeEnd).
//
// LA CLÉ EST REPRISE, ET C'EST LA DONNÉE : la génération ne fait que 2 bits, donc un même couple
// (slot, gen) porte plusieurs objets successifs au cours d'un match. Sans la borne `lifeEnd`, le
// recensement du SUIVANT prouverait la survie du PRÉCÉDENT.
func gwPickupBoundsFrom(t0, lifeEnd, filmEnd uint64, kfTimes, seen []uint64) gwPickupBounds {
	b := gwPickupBounds{LowUS: t0, HighUS: filmEnd, SeenKF: len(seen)}
	if len(seen) > 0 {
		b.LowUS = seen[len(seen)-1]
	}
	if len(kfTimes) > 0 && len(seen) > 0 && seen[len(seen)-1] == kfTimes[len(kfTimes)-1] {
		b.NeverPicked = true
		return b
	}
	next, ok := gwPickupNextAfter(kfTimes, b.LowUS)
	switch {
	case ok:
		b.HighUS = next
	default:
		b.NoLaterKF = true
	}
	if lifeEnd < b.HighUS {
		b.HighUS = lifeEnd
	}
	if b.HighUS < b.LowUS {
		b.HighUS = b.LowUS
	}
	return b
}

// gwPickupNextAfter rend le premier instant de `sorted` strictement supérieur à `at`.
func gwPickupNextAfter(sorted []uint64, at uint64) (uint64, bool) {
	i := sort.Search(len(sorted), func(i int) bool { return sorted[i] > at })
	if i >= len(sorted) {
		return 0, false
	}
	return sorted[i], true
}

// gwPickupSeenWithin restreint le recensement d'une clé à la vie [t0, lifeEnd).
func gwPickupSeenWithin(seen []uint64, t0, lifeEnd uint64) []uint64 {
	lo := sort.Search(len(seen), func(i int) bool { return seen[i] >= t0 })
	hi := sort.Search(len(seen), func(i int) bool { return seen[i] >= lifeEnd })
	if hi < lo {
		hi = lo
	}
	return seen[lo:hi]
}

// gwPickupHit est un passage de joueur près d'un objet : qui, quand, à quelle distance.
type gwPickupHit struct {
	Slot  uint32
	TUS   uint64
	DistM float64
	Found bool
}

// gwPickupNearestPass rend le PREMIER passage d'un bipède à moins de `maxDistM` de `pos` dans la
// fenêtre [lowUS, highUS]. `samples` doit être TRIÉ par instant.
//
// « LE PREMIER », PAS « LE PLUS PROCHE » : le plan tranche l'ambiguïté dans ce sens (phase 2.1
// amendée — « si plusieurs : le premier »). À instant égal, le plus proche l'emporte, puis le
// slot le plus bas : sans ces deux départages, deux exécutions rendraient deux ramasseurs.
//
// LA DISTANCE RENDUE N'EST PAS UNE MESURE (découverte 9 du plan) : c'est celle à laquelle le
// seuil est franchi, et elle vaut mécaniquement 1,46 à 1,50 m sur tous les sous-ensembles. Elle
// ne se publie pas.
func gwPickupNearestPass(
	pos [3]float32, lowUS, highUS uint64, samples []filmdec.BipedPosition, maxDistM float64,
) gwPickupHit {
	var out gwPickupHit
	i := sort.Search(len(samples), func(i int) bool { return samples[i].TimestampUS >= lowUS })
	for ; i < len(samples); i++ {
		p := samples[i]
		if p.TimestampUS > highUS {
			break
		}
		if !p.HasWorld {
			continue
		}
		d := gwPadsDist(pos[0], pos[1], pos[2], p.X, p.Y, p.Z)
		if d >= maxDistM {
			continue
		}
		if !out.Found || p.TimestampUS < out.TUS ||
			(p.TimestampUS == out.TUS && (d < out.DistM || (d == out.DistM && p.Slot < out.Slot))) {
			out = gwPickupHit{Slot: p.Slot, TUS: p.TimestampUS, DistM: d, Found: true}
		}
		if out.Found && p.TimestampUS > out.TUS {
			break
		}
	}
	return out
}

// gwPickupRefPos rend la position où l'objet SE TROUVE quand on le ramasse : le dernier point de
// sa piste delta s'il a bougé dans sa vie, sa position de création sinon.
func gwPickupRefPos(
	c filmdec.EquipmentCreation, lifeEnd uint64, tracks []filmdec.ProjectileTrack,
) ([3]float32, bool) {
	best, bestGap := -1, uint64(math.MaxUint64)
	for i, tr := range tracks {
		if len(tr.Pts) == 0 {
			continue
		}
		t0 := tr.Pts[0].TimestampUS
		if t0 >= lifeEnd || t0+gwPickupTrackTolUS < c.TimestampUS {
			continue
		}
		if g := equipTimeGap(t0, c.TimestampUS); g < bestGap {
			best, bestGap = i, g
		}
	}
	if best < 0 {
		return [3]float32{c.X, c.Y, c.Z}, false
	}
	p := tracks[best].Pts[len(tracks[best].Pts)-1]
	return [3]float32{p.X, p.Y, p.Z}, true
}
