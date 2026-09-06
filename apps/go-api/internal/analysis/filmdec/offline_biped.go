// Décodage 100 % OFFLINE des positions absolues des bipeds (joueurs) depuis les seuls
// chunks d'un film — zéro capture Cheat Engine.
//
// RÉFÉRENCE DE GRAMMAIRE : .ai/V7.5/film_re/GRAMMAIRE_RECORD_FILM.md fait foi sur l'en-tête d'un record,
// sur le fait qu'idLow est une valeur de RUNTIME variable d'un film à l'autre (11 ici, 14 sur
// le film de la capture live), et sur les bornes de jeu utilisables comme critères absolus.
// À lire avant tout nouveau décodage.
//
// GRAMMAIRE DU RECORD BIPED (validée au quantum exact, 99,99 %, contre une table de
// vérité live ; cf. .ai/thought_log_replay.md) :
//
//	[1 préfixe=1][idLow = slot][2 tag][1 gate=0][3 maskCount]
//	[6 bits x maskCount indices de composants, croissants, le premier = 0]
//	puis les composants ; offset(i0) = début + 21 + 6*maskCount
//	i0 = [GateBits à 0][X][Y][Z]
//
// ATTENTION — il n'existe AUCUN bit « maskSel » (arbitrage 2026-07-26, sur pièces). Le
// masque est bit-exactement FUN_1406d7610 : R(1) porte ; porte=0 -> R(3) count +
// count x R(6) index ; porte=1 -> R(64) dense. La fonction RENVOIE elle-même le nombre de
// bits consommés (4, 6*count+4, ou 0x41) : la question est close côté binaire. Le 21 se
// décompose donc 1 + idLow(14) + 2 + 1 + 3, PAS 1 + 13 + 2 + 1 + 1 + 3 : le bit « en
// trop » est le 14e bit du CHAMP ID, pas une porte de masque. `idLow` est une valeur de
// RUNTIME (FUN_1406d3140 -> FUN_1406d310c sur DAT_1451f98d0/d4, peuplée au chargement de
// carte) : elle peut différer d'un film à l'autre, et le fenêtrage 13 bits utilisé par le
// détecteur ci-dessous est un MOTIF de reconnaissance, pas le champ id lui-même.
// Vérité terrain (capture live filmdec_delta_capture, 807 855 lignes) : en-tête de record
// DELTA = 21 bits sur 93,89 % de 166 800 paires de records consécutifs, facteur « 6 bits
// par index » confirmé indépendamment pour count=1..7. Mesure : cmd/tmp_hdrtruth.
//
// Les LARGEURS D'AXE d'i0 sont une CONSTANTE PAR CARTE, pas du décodeur : elles se lisent
// dans le film lui-même via DetectI0Layout (cf. i0_layout.go). 13/13/14 est la valeur de
// Cliffhanger, PAS une largeur universelle (Catalyst = 15/15/15, Streets = 12/12/12...).
//
// Déquantification : v = min + step*(q + 0.5), step = (max-min)/2^w. Les BORNES sont
// PROPRES À LA CARTE (AABB du BSP, tag scenario_structure_bsp du module) : elles doivent
// être fournies par l'appelant via ScanFilmOptions.WorldRange, sinon aucune coordonnée
// monde n'est émise (cf. map_bounds.go).
//
// CE FICHIER PORTE LA GRAMMAIRE. L'ENTRÉE PAR BANDE de slots et la plomberie du balayage
// (choix des chunks, découpage i0, relevé de la bande `ti=35`) vivent dans
// offline_biped_band.go — un DÉPLACEMENT du 2026-09-05, sans une ligne de logique changée,
// pour repasser sous le seuil de 500 lignes.
package filmdec

import (
	"fmt"

	"levelup/go-api/internal/analysis/filmsource"
)

// BipedTypeIndex est le typeIndex (ti) des entités biped (joueurs) dans le registre film.
const BipedTypeIndex = 35

// Tailles (en bits) de la grammaire du record biped.
const (
	bipedHeaderBits = 21 // préfixe(1) + idLow(14) + tag(2) + gate(1) + maskCount(3) — cf. en-tête de fichier
	bipedIndexBits  = 6  // un index de composant
	bipedMinMaskCnt = 2
	bipedMaxMaskCnt = 7
	bipedSlotBits   = 13
)

// BipedPosition est une position absolue de biped décodée offline.
type BipedPosition struct {
	// Slot est l'identifiant bas (13 bits) de l'entité. Il MIGRE aux respawns : un slot
	// correspond à UNE VIE, pas à un joueur (l'attribution slot -> joueur n'est pas faite).
	Slot uint32
	// Chunk / PacketIndex localisent l'échantillon dans le film.
	Chunk, PacketIndex int
	// TimestampUS est l'horodatage du paquet porteur, en microsecondes (horloge du film).
	TimestampUS uint64
	// X, Y, Z sont en unités monde ; Z est l'axe vertical (étages). Ils ne sont renseignés
	// que si HasWorld : sans les bornes de la carte, un quantum n'est PAS une coordonnée.
	X, Y, Z float32
	// HasWorld indique que X/Y/Z sont de vraies coordonnées monde (bornes de la carte
	// fournies). À false, seul Q est exploitable.
	HasWorld bool
	// Q est le triplet d'INDICES DE QUANTUM BRUTS (X sur 13 bits, Y sur 13, Z sur 14) lu
	// dans le film, AVANT déquantification. Il ne dépend d'aucune plage : c'est la seule
	// donnée réellement décodée. Le conserver permet de re-déquantifier avec une autre
	// plage (per-map) sans re-balayer le film.
	Q [3]uint32
	// Directions capturées dans le MÊME record (composants i1/i2 qui suivent i0), quand
	// ScanFilmOptions.CaptureDirs est actif. Cf. offline_aim.go.
	componentDirs
	// Vitalité (santé i4, bouclier i5) capturée dans le MÊME record que la position —
	// donc même slot, même instant, même paquet. Renseignée sous la même option
	// CaptureDirs : c'est le même balayage, poursuivi de deux composants.
	componentVitals
}

// HealthAt rend la fraction de vie [0,1] décodée dans ce record, et sa validité.
func (p BipedPosition) HealthAt() (float32, bool) {
	if !p.HasBody {
		return 0, false
	}
	return HealthFraction(p.Body.Health), true
}

// ShieldAt rend la fraction de bouclier [0,1] décodée dans ce record, et sa validité.
func (p BipedPosition) ShieldAt() (float32, bool) {
	if !p.HasShield {
		return 0, false
	}
	return ShieldFraction(p.Shield.Shield), true
}

// ScanFilmOptions règle le balayage offline.
type ScanFilmOptions struct {
	// Chunks à balayer ; vide -> tous les chunks présents (chunk_01..chunk_NN).
	Chunks []int
	// RequireTag1 exige tag == 1 dans l'en-tête (filtre éprouvé sur 000d5950).
	RequireTag1 bool
	// DropSaturated rejette les positions dont un axe tombe dans le bucket extrême
	// (q == 0 ou q == 2^w-1) : ce sont des valeurs écrêtées, pas des positions réelles
	// (9 points sur 171 116 pour 000d5950, mais ils multipliaient le span Z par 8).
	DropSaturated bool
	// MaxSpeedMPS écarte les téléportations : une position dont la vitesse depuis la
	// précédente position ACCEPTÉE du même slot dépasse ce seuil est un faux positif du
	// balayage bit à bit (le motif d'en-tête peut coïncider avec du bruit). 0 = désactivé.
	MaxSpeedMPS float64
	// IsolationGapMS écarte les échantillons temporellement isolés dans leur slot (cf.
	// DropIsolated). 0 = désactivé.
	IsolationGapMS int
	// TeleportExemptions lève le filtre MaxSpeedMPS à ±200 ms d'un événement 117 du même
	// slot (cf. TeleportExemptions — décision D2) : une téléportation de translocateur est
	// un déplacement RÉEL à 193-1540 m/s, pas un faux positif du balayage. nil (défaut) =
	// aucun événement fourni, filtre strictement identique à l'actuel.
	TeleportExemptions TeleportExemptions
	// CaptureDirs décode en plus les composants i1 (vélocité) et i2 (forward/cap) qui
	// suivent i0 dans le même record. Ne change AUCUNE position émise (le curseur repart
	// toujours de la fin d'i0) : le décodage des directions est en lecture seule.
	CaptureDirs bool
	// Layout force le découpage binaire d'i0. nil (défaut) = le découpage est LU dans le
	// film par DetectI0Layout — c'est le mode normal, car les largeurs d'axe sont propres à
	// la carte. Ne renseigner que pour rejouer un découpage connu (tests, comparaisons).
	Layout *I0Layout
	// WorldRange porte les BORNES DE LA CARTE (AABB du BSP principal). Obligatoire pour
	// produire des coordonnées monde : nil -> ScanFilmBipedPositions échoue avec
	// ErrUnknownMapBounds, sauf si QuantaOnly. Cf. MapQuantCatalog.
	WorldRange *Vec3Range
	// DynPrecOrientation choisit la grammaire d'i2/i3 sous CaptureDirs. false (défaut) =
	// celle du BIPÈDE. true = celle des archétypes qui portent les variantes
	// `-dynamic-precision-` de forward-and-up et angular-velocity : ti=38 (corps rigide),
	// 39, **40 (véhicule)** et 43 (dispositif). Ce sont QUATRE désérialiseurs distincts
	// (FUN_14076e278/FUN_140d70998 contre FUN_140c5f7ec/FUN_140d87740), résolus
	// statiquement le 2026-09-03 — cf. components_dynprec_orientation.go. Laisser ce
	// drapeau à false sur la bande ti=40 fait atteindre i4/i5 avec un curseur décalé :
	// c'est la cause racine du bruit de vitalité mesuré au lot V2b.
	DynPrecOrientation bool
	// QuantaOnly autorise un balayage SANS bornes : les positions ne portent alors que Q
	// (HasWorld=false), et les filtres exprimés en m/s sont inopérants — donc désactivés.
	// Réservé aux outils d'analyse du bitstream.
	QuantaOnly bool
}

// DefaultMaxSpeedMPS : borne haute très large (le déplacement le plus rapide d'un Spartan,
// grappin ou véhicule, reste sous ~35 m/s). Ne sert qu'à couper les aberrations, pas à
// lisser le mouvement.
const DefaultMaxSpeedMPS = 100

// maxRejectStreak : après ce nombre de rejets consécutifs sur un slot, on se réancre sur la
// position courante — sinon une ancre elle-même fausse condamnerait tout le reste du slot.
const maxRejectStreak = 3

// DefaultScanFilmOptions est le réglage validé sur 000d5950.
func DefaultScanFilmOptions() ScanFilmOptions {
	return ScanFilmOptions{
		RequireTag1:    true,
		DropSaturated:  true,
		MaxSpeedMPS:    DefaultMaxSpeedMPS,
		IsolationGapMS: DefaultIsolationGapMS,
	}
}

// ScanFilmBipedPositions décode toutes les positions absolues de bipeds des chunks du
// film situé dans dir. Les chunks illisibles sont ignorés (le film peut être partiel) ;
// une erreur n'est renvoyée que si AUCUN chunk n'a pu être lu.
//
// ScanFilmBipedPositions est l'ENVELOPPE D2, HORS PRODUCTION : elle charge le film puis appelle
// [ScanBipedPositions]. La cuisson passe un film deja charge (une seule decompression).
func ScanFilmBipedPositions(dir string, opt ScanFilmOptions) ([]BipedPosition, error) {
	// LE REFUS DE BORNES PRECEDE LE CHARGEMENT, et c'est l'ordre d'origine : un appelant sans
	// bornes doit recevoir ErrUnknownMapBounds, pas une erreur de lecture de repertoire. Le
	// message garde le chemin du film, que la forme `Scan*(film)` n'a plus.
	if opt.WorldRange == nil && !opt.QuantaOnly {
		return nil, fmt.Errorf("%w (film %s) : renseigner ScanFilmOptions.WorldRange, ou QuantaOnly pour n'obtenir que les quanta", ErrUnknownMapBounds, dir)
	}
	film, err := filmsource.LoadDir(dir, nil)
	if err != nil {
		return nil, err
	}
	return ScanBipedPositions(film, opt)
}

// ScanBipedPositions décode les positions absolues de bipeds d'un film DEJA CHARGE. Cf.
// [ScanFilmBipedPositions] pour la doctrine du balayage.
//
// C'est l'entrée BIPÈDE de [ScanBipedPositionsForBand] : elle ne fait qu'y ajouter le relevé
// de la bande de slots `ti=35` (bipedSlotBand). Aucun décodage ne lui est propre.
func ScanBipedPositions(film *filmsource.Film, opt ScanFilmOptions) ([]BipedPosition, error) {
	chunks, err := bipedScanChunks(film, opt)
	if err != nil {
		return nil, err
	}
	band := bipedSlotBand(film, chunks)
	if band.Count() == 0 {
		return nil, fmt.Errorf("aucun slot biped (ti=%d) dans les keyframes du film", BipedTypeIndex)
	}
	return ScanBipedPositionsForBand(film, band, opt)
}

// ScanBipedRecords balaie un payload de paquet delta bit à bit et renvoie les positions
// absolues des records biped reconnus. PUR (aucune I/O) : c'est le cœur testable du
// décodeur. Les champs Chunk/PacketIndex/TimestampUS sont laissés à zéro (remplis par
// l'appelant).
func ScanBipedRecords(payload []byte, slots SlotBand, lay I0Layout, opt ScanFilmOptions) []BipedPosition {
	i0Bits := lay.TotalBits()
	var out []BipedPosition
	// UN SEUL lecteur de bits pour tout le payload : `scanRecordDirs` le repositionne par
	// `SetBitPos` a chaque composant de vitalite, la ou il en allouait deux PAR RECORD.
	br := NewBitReader(payload)
	// LA GRAMMAIRE D'ORIENTATION EST RESOLUE UNE FOIS PAR PAYLOAD, hors de la boucle : elle ne
	// depend que des options (l'archetype decide d'i2 ET d'i3 a la fois), jamais du record.
	g := dirsGrammar{}
	if opt.DynPrecOrientation {
		g = dynPrecOrientationGrammar()
	}
	// L'ANCRAGE EST CELUI DU MARCHEUR UNIQUE (delta_biped_walk.go) : ce balayage etait la
	// neuvieme copie de la meme triple boucle. L'avance de curseur, la borne et le pas d'echec
	// sont les siens ; ne reste ici que la LECTURE du record.
	walkDeltaBipedPayload(payload, slots, lay, opt.RequireTag1, func(r deltaBipedRecord) {
		var q [3]uint32
		for ax := 0; ax < 3; ax++ {
			q[ax] = readBitsAt(payload, r.I0+lay.AxisOffset(ax), int(lay.AxisW[ax]))
		}
		if opt.DropSaturated && saturatedQuantum(q, lay) {
			return
		}
		rec := BipedPosition{Slot: r.Slot, Q: q}
		if opt.WorldRange != nil {
			rec.HasWorld = true
			rec.X = DequantBipedAxis(q[0], 0, lay, *opt.WorldRange)
			rec.Y = DequantBipedAxis(q[1], 1, lay, *opt.WorldRange)
			rec.Z = DequantBipedAxis(q[2], 2, lay, *opt.WorldRange)
		}
		if opt.CaptureDirs {
			rec.componentDirs, rec.componentVitals = scanRecordDirs(br, r.I0+i0Bits, r.Total, r.Mask, g)
			rec.MaskBits, rec.MaskOver = maskBitsOf(r.Mask)
			if recordMaskHook != nil {
				recordMaskHook(r.Mask, payload, r.I0+i0Bits)
			}
		}
		out = append(out, rec)
	})
	return out
}

// matchBipedHeader teste la grammaire d'en-tête biped à la position bit p, exige un i0
// ABSOLU (spine + useDefault nuls) DE LA RÉGION ATTENDUE (lay.Region — zéro partout sauf
// sur les cartes dont la région jouée n'est pas la première du bloc structure-BSP) et
// renvoie l'offset bit de i0, le slot et la liste des index de composants du masque.
// Un record d'une autre région est écarté : ses quanta vivent dans une autre AABB.
func matchBipedHeader(pay []byte, p, total int, slots SlotBand, needTag1 bool, lay I0Layout) (int, uint32, []int, bool) {
	i0, slot, idx, ok := matchBipedHeaderRaw(pay, p, total, slots, needTag1, lay.TotalBits())
	if !ok {
		return 0, 0, nil, false
	}
	const preGate = i0SpineBits + i0UseDefaultBits
	if readBitsAt(pay, i0, preGate) != 0 { // i0 absolu : spine + useDefault nuls
		return 0, 0, nil, false
	}
	if readBitsAt(pay, i0+preGate, lay.GateBits-preGate) != lay.Region {
		return 0, 0, nil, false
	}
	return i0, slot, idx, true
}

// matchBipedHeaderRaw teste la seule grammaire d'EN-TÊTE (préfixe, slot, tag, masque) et
// vérifie que needBits bits restent lisibles après le début d'i0. Il ne suppose RIEN du
// contenu d'i0 : c'est le point d'entrée du détecteur de découpage (i0_layout.go), qui doit
// justement mesurer i0 sans en présupposer la structure.
func matchBipedHeaderRaw(pay []byte, p, total int, slots SlotBand, needTag1 bool, needBits int) (int, uint32, []int, bool) {
	if readBitsAt(pay, p, 1) != 1 {
		return 0, 0, nil, false
	}
	slot := readBitsAt(pay, p+1, bipedSlotBits)
	if !slots.Has(slot) {
		return 0, 0, nil, false
	}
	if needTag1 && readBitsAt(pay, p+14, 2) != 1 {
		return 0, 0, nil, false
	}
	if readBitsAt(pay, p+16, 2) != 0 { // (14e bit id ou LSB tag) + gate — PAS un maskSel
		return 0, 0, nil, false
	}
	mc := int(readBitsAt(pay, p+18, 3))
	if mc < bipedMinMaskCnt || mc > bipedMaxMaskCnt {
		return 0, 0, nil, false
	}
	i0 := p + bipedHeaderBits + bipedIndexBits*mc
	if i0+needBits > total {
		return 0, 0, nil, false
	}
	idx, ok := ascendingFromZero(pay, p+bipedHeaderBits, mc)
	if !ok {
		return 0, 0, nil, false
	}
	return i0, slot, idx, true
}

// ascendingFromZero valide la liste d'indices de composants (premier = 0, strictement
// croissants — invariant fort du masque) et la renvoie.
//
// LA VALIDATION PASSE AVANT L'ALLOCATION. Ce test est joue pour CHAQUE position de bit
// candidate d'un payload delta et il echoue dans l'immense majorite des cas : allouer la
// tranche d'abord, c'etait une allocation jetee par candidat rejete. Les index sont donc
// rassembles dans un tampon de pile (`count` vaut au plus [bipedMaxMaskCnt]) et la tranche
// n'est allouee qu'une fois la liste validee.
func ascendingFromZero(pay []byte, at, count int) ([]int, bool) {
	var buf [bipedMaxMaskCnt]int
	if count < 0 || count > len(buf) {
		return nil, false
	}
	prev := -1
	for k := 0; k < count; k++ {
		idx := int(readBitsAt(pay, at+bipedIndexBits*k, bipedIndexBits))
		if (k == 0 && idx != 0) || idx <= prev {
			return nil, false
		}
		prev = idx
		buf[k] = idx
	}
	out := make([]int, count)
	copy(out, buf[:count])
	return out, true
}

// saturatedQuantum signale un axe dans son bucket extrême (valeur écrêtée).
func saturatedQuantum(q [3]uint32, lay I0Layout) bool {
	for i, w := range lay.AxisW {
		if q[i] == 0 || q[i] == uint32(1)<<w-1 {
			return true
		}
	}
	return false
}

// DequantBipedAxis déquantifie l'axe ax (0=X, 1=Y, 2=Z) d'une position absolue biped avec
// les largeurs du découpage lay et les BORNES DE LA CARTE world. Calcul en float64 pour ne
// pas décaler l'indice de quantum sur les arrondis float32.
//
// world DOIT être l'AABB du BSP de la carte du film (cf. MapQuantCatalog) : appliquer les
// bornes d'une autre carte produit une coordonnée fausse, pas approximative.
func DequantBipedAxis(q uint32, ax int, lay I0Layout, world Vec3Range) float32 {
	rng := world[ax]
	step := (float64(rng.Max) - float64(rng.Min)) / float64(uint64(1)<<lay.AxisW[ax])
	return float32(float64(rng.Min) + step*(float64(q)+quantCenter))
}

// readBitsAt lit n bits MSB-first à la position bit pos (n <= 32).
//
// AUCUNE GARDE, ET C'EST VOULU : une lecture hors du tampon PANIQUE (`index out of range`).
// Les balayages qui l'appellent bornent eux-memes leur fenetre ; une panique ici denonce un
// balayage qui a perdu ses bornes, et la faire taire masquerait le defaut. Cette convention
// est INCHANGEE par la lecture par mot : le chemin rapide n'est pris que quand les n bits
// tiennent entierement dans le tampon, tout le reste retombe sur la boucle d'origine — qui
// panique aux memes positions qu'avant.
func readBitsAt(b []byte, pos, n int) uint32 {
	if pos >= 0 && n > 0 && n <= 64 && pos+n <= len(b)*8 {
		return uint32(wordBitsAt(b, pos, uint(n)))
	}
	var v uint32
	for i := 0; i < n; i++ {
		p := pos + i
		v = v<<1 | uint32(b[p>>3]>>(7-uint(p&7))&1)
	}
	return v
}
