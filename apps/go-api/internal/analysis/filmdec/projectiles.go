package filmdec

import (
	"fmt"
	"sort"
)

// projectiles.go — TRAJECTOIRES DE PROJECTILE, décodées des paquets delta.
//
// CE QUE CE FICHIER ÉTABLIT, et qui contredit deux affirmations antérieures du projet :
//
//  1. Un projectile EST une entité répliquée ordinaire. Le registre d'archétypes du film
//     (chunk_00) contient un archétype dont les composants s'appellent littéralement
//     `projectile-at-rest-state`, `projectile-tether-state`, `projectile-command_tick`,
//     `projectile-deceleration-disabled-state`, et dont les quatre premiers sont
//     `object-position-component`, `object-translational-velocity-component`,
//     `forward-and-up`, `angular-velocity`. Sur ce build c'est **ti=41** — un archétype que
//     nos notes classaient « divers, mal caractérisé ». Personne n'avait cherché de
//     projectile : il était dans le registre depuis le début.
//
//  2. `0x4C0C00` N'EST PAS UN MARQUEUR (cf. grenade_events.go, dont l'en-tête est corrigé).
//     C'est le milieu d'un record de CRÉATION d'entité : `[5 bits bas de typeIndex=41]` suivi
//     de 19 bits constants d'amorce de l'état par défaut. Le « marqueur » fonctionnait par
//     accident heureux — il reconnaissait la naissance d'un projectile.
//
// TÉMOINS (film 000d5950, 70 lancers de grenade de la vérité terrain) :
//
//	appariement lancer -> naissance d'entité   70/70   contre 5,3 % par décalage circulaire
//	distance naissance <-> bipède lanceur      0,77 u  contre 6,4 u (instant permuté)
//	                                                   et 33,9 u (bipède tiré au hasard)
//	direction initiale <-> cap de visée        1,0°    contre 16,7 % d'accord au hasard
//	durée de vie par type de grenade           Shock 3,05 s · Frag 1,40 · Plasma 1,22 ·
//	                                           Spike 0,77 — p < 5e-4 sur 20 000 permutations
//
// Le point de naissance n'est pas « près d'un joueur » : il est à 0,4 unité DEVANT et 0,64
// AU-DESSUS du repère du bipède, maximum 1,37 sur 70. C'est la main, à la bonne hauteur.
//
// CE QUE CE DÉCODEUR NE DONNE PAS : **l'impact**. Il n'existe aucun événement de détonation
// dans le film. Ce qu'on lit est la DERNIÈRE POSITION RÉPLIQUÉE, et pour les grenades à
// fragmentation la réplication cesse ~1,4 s après le lancer alors que la mèche court jusqu'à
// ~3 s : le dernier point approche le point d'explosion parce que l'objet ne bouge plus, pas
// parce qu'on lit l'explosion. À l'écran, écrire « dernière position connue », jamais
// « impact ». Seul `projectile-at-rest-state` (i18) CERTIFIE une fin de vol : il est porté
// par le dernier record de la vie dans 78 cas sur 79.

// ProjectileTypeIndex est l'archétype des projectiles. À VÉRIFIER par le nom des composants
// du registre plutôt qu'à câbler : c'est un index de build, pas une constante du format.
const ProjectileTypeIndex = 41

// projectileRestComponent est l'index du composant `projectile-at-rest-state` : sa présence
// dans le masque certifie que le record est le dernier de la vie du projectile.
//
// LE MÊME INDEX PORTE UN COMPOSANT DE MÊME SENS DANS L'ARCHÉTYPE D'ÉQUIPEMENT, et ce n'est pas
// une supposition : le registre de ti=37 nomme i18 `item-at-rest-component` (mesuré sur
// 000d5950 et 00ba2e1c le 2026-08-18, `equipment_life_end_test.go`). La coupure ci-dessous a
// donc un sens pour les deux archétypes que `ScanFilmWorldObjects` sert. Elle n'est pourtant
// PAS ce qui borne une vie d'équipement : sur les deux films, la distribution des durées est
// IDENTIQUE avec et sans elle, parce que le flux de position s'arrête de lui-même quand
// l'objet s'immobilise (cf. splitLives).
const projectileRestComponent = 18

// projPosBits est la longueur d'`object-position-component` sur le chemin dominant :
// 3 de porte + les trois axes + 2 de queue. Voir WorldObjectPrecision.
func projPosBits() int {
	p := WorldObjectPrecision
	return 3 + int(p.AxisW[0]+p.AxisW[1]+p.AxisW[2]) + 2
}

// ProjectileSample est une position de projectile à un instant.
type ProjectileSample struct {
	TimestampUS uint64
	Chunk       int
	X, Y, Z     float32
	// AtRest signale que ce record porte `projectile-at-rest-state` : le vol est fini.
	AtRest bool
}

// ProjectileTrack est la vie d'un projectile : un slot, une génération, une suite de positions.
type ProjectileTrack struct {
	// Slot et Gen identifient la vie. LA PAIRE, pas le slot seul : le pool de slots reboucle
	// et un même slot sert plusieurs projectiles au cours du match.
	Slot uint32
	Gen  uint32
	Pts  []ProjectileSample
}

// EquipmentTypeIndex est l'archétype des objets d'ÉQUIPEMENT déployés (mur, champ de
// réparation, détecteur, écran). Comme les projectiles, ce sont des entités du monde : elles
// ont une position, et donc un endroit où le joueur les a posées.
//
// À VÉRIFIER par le nom des composants du registre plutôt qu'à câbler : ses composants sont
// equipment-deployed / activated / creator / energy / charges-remaining.
const EquipmentTypeIndex = 37

// ScanFilmProjectiles décode les trajectoires de projectile du film de dir.
//
// HORS LIGNE (I/O disque sur tout le film) — jamais depuis un chemin de requête.
func ScanFilmProjectiles(dir string, wr *Vec3Range) ([]ProjectileTrack, error) {
	return ScanFilmWorldObjects(dir, wr, ProjectileTypeIndex)
}

// ScanFilmWorldObjects décode les trajectoires d'un archétype d'OBJET DU MONDE quelconque.
//
// Ces archétypes (projectile ti=41, équipement ti=37, arme au sol ti=42, corps rigide ti=38)
// partagent le même composant de position `object-position-component` : position ABSOLUE
// quantifiée à chaque image, aux largeurs de la CARTE — à la différence du bipède, qui envoie
// des deltas. Un seul décodeur les sert donc tous, avec l'archétype en paramètre.
//
// HORS LIGNE (I/O disque sur tout le film) — jamais depuis un chemin de requête.
func ScanFilmWorldObjects(dir string, wr *Vec3Range, typeIndex int) ([]ProjectileTrack, error) {
	if wr == nil {
		return nil, fmt.Errorf("bornes monde absentes : sans elles le décodeur ne rend que des quanta")
	}
	n := CountFilmChunks(dir)
	if n == 0 {
		return nil, fmt.Errorf("aucun chunk film dans %s", dir)
	}
	band := worldObjectSlotBand(dir, n, typeIndex)
	if len(band) == 0 {
		return nil, fmt.Errorf("aucun slot d'archétype ti=%d dans les keyframes de %s",
			typeIndex, dir)
	}
	type key struct{ slot, gen uint32 }
	lives := map[key][]ProjectileSample{}
	for c := 1; c <= n; c++ {
		chunk, err := ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, p := range WalkPackets(chunk) {
			if p.Type != PacketTypeDelta {
				continue
			}
			pay := p.Payload(chunk)
			for _, s := range scanProjectileRecords(pay, band, wr) {
				s.TimestampUS, s.Chunk = p.TimestampUS, c
				k := key{s.slot, s.gen}
				lives[k] = append(lives[k], s.ProjectileSample)
			}
		}
	}
	out := make([]ProjectileTrack, 0, len(lives))
	for k, pts := range lives {
		// Tri TOTAL des échantillons d'une vie : l'instant seul laisse des ex æquo (plusieurs
		// records du même projectile dans un même paquet), et un départage arbitraire ferait
		// dépendre le premier point — donc la naissance — de l'ordre d'arrivée.
		sort.Slice(pts, func(i, j int) bool { return lessSample(pts[i], pts[j]) })
		for _, seg := range splitLives(pts) {
			out = append(out, ProjectileTrack{Slot: k.slot, Gen: k.gen, Pts: seg})
		}
	}
	// Tri TOTAL des vies. `lives` est une MAP : son ordre d'itération change à chaque exécution,
	// et un tri sur le seul instant de naissance laissait cet aléa décider du rang des ex æquo.
	// Ce n'était pas cosmétique : `replay.projectileBirths` retrie ces naissances et
	// `birthNear` prend celle d'un INDICE donné — l'ordre choisissait donc la position publiée
	// pour un lancer de grenade. Le couple (slot, gen) est unique par construction (c'est la clé
	// de `lives`), l'instant de naissance sépare les segments d'une même clé : l'ordre est total.
	sort.Slice(out, func(i, j int) bool { return lessTrack(out[i], out[j]) })
	return out, nil
}

// lessSample : ordre total sur les échantillons d'une vie (instant, puis position).
func lessSample(a, b ProjectileSample) bool {
	if a.TimestampUS != b.TimestampUS {
		return a.TimestampUS < b.TimestampUS
	}
	if a.X != b.X {
		return a.X < b.X
	}
	if a.Y != b.Y {
		return a.Y < b.Y
	}
	return a.Z < b.Z
}

// lessTrack : ordre total sur les vies (naissance, puis slot, puis génération).
func lessTrack(a, b ProjectileTrack) bool {
	if a.Pts[0].TimestampUS != b.Pts[0].TimestampUS {
		return a.Pts[0].TimestampUS < b.Pts[0].TimestampUS
	}
	if a.Slot != b.Slot {
		return a.Slot < b.Slot
	}
	return a.Gen < b.Gen
}

// projectileGapUS est le trou temporel au-delà duquel deux échantillons d'un même couple
// (slot, génération) appartiennent à DEUX vies différentes.
//
// POURQUOI CETTE ÉTAPE EST INDISPENSABLE : le pool de slots de projectile reboucle, et la
// génération ne fait que 2 bits — le couple (slot, gen) revient donc plusieurs fois dans un
// match. Sans découpage, on obtient des « trajectoires » de 300 à 460 SECONDES, c'est-à-dire
// la concaténation de dizaines de vols distincts. Un projectile est répliqué à ~60 Hz : un
// trou de plus de 250 ms est une frontière de vie, pas une lacune.
const projectileGapUS = 250_000

// splitLives découpe une suite d'échantillons triés en vies séparées, et écarte celles de
// moins de trois points — deux points ne dessinent pas une trajectoire.
//
// CE QUE LA VIE AINSI DÉCOUPÉE EST, ET CE QU'ELLE N'EST PAS (mesuré le 2026-08-18 sur
// 000d5950 et 00ba2e1c, `equipment_life_end_test.go`). Ce balayage ne retient que les records
// qui portent `object-position-component` : la vie décodée est donc la vie MOBILE de l'entité.
// Un objet d'équipement qui se pose cesse de bouger, donc cesse d'émettre sa position, et sa
// vie décodée se ferme sur le trou de 250 ms — alors que l'entité, elle, reste répliquée. Le
// recensement des keyframes le PROUVE : 101 poses sur 295 (000d5950) et 228 sur 537
// (00ba2e1c) y figurent encore plus d'une seconde après la fin de leur flux de position.
// La dernière borne d'une vie est donc une MISE AU REPOS, jamais une disparition.
func splitLives(pts []ProjectileSample) [][]ProjectileSample {
	var out [][]ProjectileSample
	start := 0
	flush := func(end int) {
		if end-start >= 3 {
			seg := make([]ProjectileSample, end-start)
			copy(seg, pts[start:end])
			out = append(out, seg)
		}
	}
	for i := 1; i < len(pts); i++ {
		// Un record `at rest` clôt la vie : le projectile ne bougera plus.
		if pts[i].TimestampUS-pts[i-1].TimestampUS > projectileGapUS || pts[i-1].AtRest {
			flush(i)
			start = i
		}
	}
	flush(len(pts))
	return out
}

// projSample porte le slot et la génération le temps du regroupement en vies.
type projSample struct {
	ProjectileSample
	slot, gen uint32
}

// scanProjectileRecords balaye un payload delta. PUR (aucune I/O).
//
// DEUX ÉCARTS AVEC LE BALAYAGE DU BIPÈDE, tous deux nécessaires et mesurés :
//   - AUCUN filtre sur le tag. Le tag de 2 bits est la GÉNÉRATION du handle, et ses quatre
//     valeurs sont légitimes. Le bipède est à tag=1 dans 99,9 % des cas, d'où le filtre
//     historique ; le projectile utilise les quatre, et 16 des 70 trajectoires de grenade sont
//     INTÉGRALEMENT en tag=0. Filtrer les perdrait toutes.
//   - maskCount minimal à 1 (et non 2) : les records de projectile sont courts.
func scanProjectileRecords(pay []byte, band map[uint32]bool, wr *Vec3Range) []projSample {
	var out []projSample
	posBits := projPosBits()
	limit := len(pay)*8 - (worldObjectHeaderBits + worldObjectIndexBits + posBits)
	for p := 0; p <= limit; p++ {
		rec, ok := matchWorldObjectRecord(pay, p, band)
		if !ok || rec.Idx[0] != 0 { // i0 doit être présent : c'est la position
			continue
		}
		v, ok := decodeWorldObjectPos(pay, rec.After, wr)
		if !ok {
			continue
		}
		rest := false
		for _, i := range rec.Idx {
			if i == projectileRestComponent {
				rest = true
			}
		}
		out = append(out, projSample{
			ProjectileSample: ProjectileSample{X: v[0], Y: v[1], Z: v[2], AtRest: rest},
			slot:             rec.Slot, gen: rec.Gen,
		})
		p += posBits // un record accepté n'est pas re-balayé
	}
	return out
}

// Découpage de l'en-tête d'un record delta d'OBJET DU MONDE (ti=37/38/41/42) :
// préfixe(1) + slot(13) + génération(2) + porte de masque(2) + nombre de composants(3),
// puis un index de composant par entrée. Nommé parce que deux balayages s'en servent
// (projectiles et équipement) : trois copies du littéral 21 auraient re-divergé.
const (
	worldObjectHeaderBits = 21
	worldObjectIndexBits  = 6
	worldObjectMaxMaskCnt = 7
)

// WorldObjectRecord est l'en-tête reconnu d'un record delta d'objet du monde.
type WorldObjectRecord struct {
	// Slot et Gen identifient la vie de l'objet — LA PAIRE : le pool de slots reboucle.
	Slot, Gen uint32
	// Idx est la liste des index de composant du masque, strictement croissante.
	Idx []int
	// After est la position du premier bit APRÈS le masque, c'est-à-dire le début d'i0.
	After int
}

// matchWorldObjectRecord reconnaît un en-tête de record delta d'objet du monde à la position
// de bit p. PUR (aucune I/O).
//
// DEUX ÉCARTS AVEC LE BALAYAGE DU BIPÈDE, tous deux nécessaires et mesurés :
//   - AUCUN filtre sur le tag. Le tag de 2 bits est la GÉNÉRATION du handle, et ses quatre
//     valeurs sont légitimes. Le bipède est à tag=1 dans 99,9 % des cas, d'où le filtre
//     historique ; le projectile utilise les quatre, et 16 des 70 trajectoires de grenade sont
//     INTÉGRALEMENT en tag=0. Filtrer les perdrait toutes.
//   - maskCount minimal à 1 (et non 2) : les records d'objet du monde sont courts.
func matchWorldObjectRecord(pay []byte, p int, band map[uint32]bool) (WorldObjectRecord, bool) {
	var rec WorldObjectRecord
	if PeekBits(pay, p, 1) != 1 { // préfixe de record DELTA
		return rec, false
	}
	slot := uint32(PeekBits(pay, p+1, 13))
	if !band[slot] {
		return rec, false
	}
	if PeekBits(pay, p+16, 2) != 0 { // porte de masque = 0 -> branche éparse
		return rec, false
	}
	mc := int(PeekBits(pay, p+18, 3))
	if mc < 1 || mc > worldObjectMaxMaskCnt {
		return rec, false
	}
	idx, ok := ascendingComponents(pay, p+worldObjectHeaderBits, mc)
	if !ok {
		return rec, false
	}
	rec.Slot, rec.Gen = slot, uint32(PeekBits(pay, p+14, 2))
	rec.Idx, rec.After = idx, p+worldObjectHeaderBits+worldObjectIndexBits*mc
	return rec, true
}

// ascendingComponents lit les mc index de composant de 6 bits et exige qu'ils soient
// STRICTEMENT croissants — c'est cette contrainte qui fait la sélectivité du balayage.
func ascendingComponents(pay []byte, at, mc int) ([]int, bool) {
	idx := make([]int, mc)
	prev := -1
	for k := 0; k < mc; k++ {
		v := int(PeekBits(pay, at+6*k, 6))
		if v <= prev {
			return nil, false
		}
		idx[k], prev = v, v
	}
	return idx, true
}

// decodeWorldObjectPos lit et déquantifie `object-position-component` sur le chemin dominant.
//
// REJET DES QUANTA SATURÉS : un axe à 0 ou à 2^w-1 est écarté. Sans cette règle, une vie sur
// soixante-dix finit au plancher du BSP (z = -84 m) — un quantum saturé n'est pas une position,
// c'est une valeur de garde.
func decodeWorldObjectPos(pay []byte, at int, wr *Vec3Range) ([3]float32, bool) {
	var v [3]float32
	if PeekBits(pay, at, 3) != 0 { // porte : precHigh, index-sel, index de région tous nuls
		return v, false
	}
	off := at + 3
	for a := 0; a < 3; a++ {
		w := WorldObjectPrecision.AxisW[a]
		q := PeekBits(pay, off, int(w))
		if q == 0 || q == (uint64(1)<<w)-1 {
			return v, false
		}
		lo, hi := wr[a].Min, wr[a].Max
		v[a] = lo + (float32(q)+0.5)*(hi-lo)/float32(uint64(1)<<w)
		off += int(w)
	}
	return v, true
}

// worldObjectSlotBand rend les slots utilisables pour un archétype, lus dans les keyframes.
//
// NI L'UN NI L'AUTRE DES DEUX EXTRÊMES N'EST JUSTE — c'est la leçon du 2026-07-26, obtenue en
// les essayant tous les deux :
//
//	COMBLER la plage (min..max), comme pour le bipède : contamine massivement. Le bipède a une
//	plage remplie à 91 %, mais le projectile à 8 %, l'équipement à 23 %, le corps rigide à 10 %
//	-- et les plages de ti=37 [1462,2660] et ti=38 [1280,2641] se recouvrent presque entièrement.
//	Symptôme mesuré : les trois archétypes rendaient des chiffres identiques à 1 % près sur cinq
//	critères indépendants (nombre de vies, durée, immobilité, convergence, vitesse d'approche).
//
//	S'EN TENIR À L'OBSERVÉ : rate l'essentiel. Un projectile vit moins d'une seconde et les
//	keyframes sont espacés de 20 s : seuls 19 slots de projectile y apparaissent, contre 57 vies
//	décodées au lieu de 580.
//
// LA FORME JUSTE est donc : combler la plage de l'archétype, PUIS retirer tout slot vu porter un
// AUTRE archétype. On récupère la couverture sans la contamination, et le retrait est fondé sur
// une observation, pas sur une heuristique.
func worldObjectSlotBand(dir string, n int, typeIndex int) map[uint32]bool {
	seen := map[uint32]bool{}
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
				if r.TI == typeIndex {
					seen[uint32(r.Slot)] = true
				} else {
					others[uint32(r.Slot)] = true
				}
			}
		}
	}
	band := fillSlotBand(seen)
	for s := range others {
		delete(band, s)
	}
	return band
}
