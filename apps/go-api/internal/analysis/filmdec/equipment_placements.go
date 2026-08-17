package filmdec

// equipment_placements.go — LES POSES d'objets d'équipement (mur de protection, capteur de
// menaces, grappin, propulseur, et les objets du monde qui partagent l'archétype).
//
// CE QU'UNE POSE EST, ET D'OÙ CHAQUE MORCEAU VIENT :
//
//	quoi      le GlobalID du tag `eqip`, lu dans le bloc `object-multiplayer-properties` du
//	          record de CRÉATION (cf. equipment_creation.go ; identité prouvée le 2026-08-17)
//	où        la position i0 du MÊME record — le lieu exact de la pose
//	quand     t0 = l'instant du record de création ; t1 = le dernier point de la vie d'objet
//	          décodée des paquets delta (ScanFilmWorldObjects), donc la disparition
//
// LE FILTRE QUI FAIT LA MESURE, et sans lui rien n'est publiable. L'en-tête NEW
// (`[0][01][slot:13][gen:2][ti:6=37]`) n'est PAS sélectif : sur un film BTB, la bande compte
// 2 000 à 3 500 slots sur 8 192, si bien qu'un quart des positions de bit tirées au hasard
// passent le test de bande — mesuré le 2026-08-17, 20 657 ancres et 57 248 ancres pour quelques
// dizaines de créations réelles. Le corps qui suit filtre un peu (masque valide, position
// décodable), pas assez : les records acceptés portaient 353 identifiants `eqip` distincts pour
// 393 records, c'est-à-dire du bruit. Une pose n'est donc RETENUE que si sa position retombe sur
// le premier point de la vie que son en-tête ANNONCE — une coïncidence sur trois axes quantifiés
// qui ne s'obtient pas par hasard, et qui vient d'un décodage (les paquets delta) qui ne sait
// rien du record de création.
//
// UNE POSE PAR VIE. Un même objet peut être annoncé plusieurs fois (retransmission de l'état
// complet) : on garde le record le PLUS ANCIEN de chaque vie, c'est lui qui date la pose.
//
// CE QUE CE FICHIER NE FAIT PAS : nommer (le manifeste du titre le fait, par GlobalID) et
// attribuer un poseur (l'assemblage le fait, il a le nuage des bipèdes). Le décodeur rend ce
// que le film DIT ; l'interprétation vit une couche plus haut.
//
// HORS LIGNE (I/O disque sur tout le film) — jamais depuis un chemin de requête.

import (
	"fmt"
	"sort"
)

// EquipmentPlacement est UNE pose d'objet d'équipement, telle que le film la porte.
type EquipmentPlacement struct {
	// Life identifie la vie d'objet (slot, génération) — la clé qui relie le record de création
	// à la trajectoire décodée des paquets delta.
	Life EquipmentLifeKey
	// T0US est l'instant du record de création : la pose. T1US est le dernier point de la vie :
	// la disparition. Les deux sur l'horloge des paquets, celle des positions de bipède.
	T0US, T1US uint64
	// X, Y, Z est la position du record de création, en coordonnées MONDE (bornes de la carte).
	X, Y, Z float32
	// GlobalID est le GlobalID du tag `eqip` de l'objet : son IDENTITÉ, telle que le jeu la
	// définit. Le nom se résout par le manifeste du titre, jamais ici.
	GlobalID uint32
	// Points est le nombre d'échantillons de la vie décodée — le dénominateur d'une trajectoire.
	Points int
}

// EquipmentPlacementStats compte ce que le balayage a rencontré. Sans ces dénominateurs, « N
// poses » ne se juge pas : c'est l'écart entre les ancres, les records acceptés et les records
// CONFIRMÉS par l'oracle qui dit à quel point l'en-tête seul est peu sélectif.
type EquipmentPlacementStats struct {
	// Calibration est la mesure de la largeur du bloc MPP sur ce film — la pièce justificative
	// de tout le reste. Bits == 0 : le film n'a pas tranché, et rien n'est publié.
	Calibration MPPCalibration
	// Lives est le nombre de vies d'objet ti=37 décodées des paquets delta.
	Lives int
	// Slots est la taille de la bande de slots de l'archétype.
	Slots int
	// Anchors est le nombre d'en-têtes NEW ti=37 reconnus (bruit compris).
	Anchors int
	// Accepted est le nombre de records dont le corps s'est déroulé (masque + position).
	Accepted int
	// Confirmed est le nombre de records dont la position retombe sur la vie annoncée.
	Confirmed int
	// Placements est le nombre de poses rendues (une par vie confirmée).
	Placements int
	// ByID compte les poses par GlobalID `eqip` — la distribution qui se croise avec le
	// manifeste et avec le rang de capacité du poseur.
	ByID map[uint32]int
}

// ScanFilmEquipmentPlacements décode les POSES d'objets d'équipement du film de dir.
//
// Rend une liste VIDE et des statistiques renseignées quand la calibration ne tranche pas :
// l'appelant doit le dire (log) et publier une liste vide, jamais des poses décodées à une
// largeur devinée. C'est le gate 0 du plan, tenu par construction.
//
// UN SEUL DÉCODAGE filmdec À LA FOIS PAR PROCESS : ce balayage installe les sondes de paquet et
// écrit `mppLeadBits`. L'appelant doit détenir LockProcessDecode ; les globaux sont restaurés.
//
// HORS LIGNE (I/O disque sur tout le film) — jamais depuis un chemin de requête.
func ScanFilmEquipmentPlacements(
	dir string, wr *Vec3Range,
) ([]EquipmentPlacement, EquipmentPlacementStats, error) {
	st := EquipmentPlacementStats{ByID: map[uint32]int{}}
	if wr == nil {
		return nil, st, fmt.Errorf("bornes monde absentes : sans elles le décodeur ne rend que des quanta")
	}
	n := CountFilmChunks(dir)
	if n == 0 {
		return nil, st, fmt.Errorf("aucun chunk film dans %s", dir)
	}
	band := worldObjectSlotBand(dir, n, EquipmentTypeIndex)
	if len(band) == 0 {
		return nil, st, fmt.Errorf("aucun slot d'archétype ti=%d dans les keyframes de %s",
			EquipmentTypeIndex, dir)
	}
	st.Slots = len(band)
	tracks, err := ScanFilmWorldObjects(dir, wr, EquipmentTypeIndex)
	if err != nil {
		return nil, st, err
	}
	spans := EquipmentLifeSpans(tracks)
	st.Lives = len(spans)

	defer SetMPPWidths(CurrentMPPWidths())
	cal, ok := CalibrateMPPWidths(dir, wr, band, spans)
	st.Calibration = cal
	if !ok {
		return nil, st, nil // le film n'a pas tranché : aucune pose, et les stats le disent
	}
	SetMPPWidths(cal.Widths)

	cre, cst, err := ScanFilmEquipmentCreationsForBand(dir, wr, band)
	if err != nil {
		return nil, st, err
	}
	st.Anchors, st.Accepted = cst.Anchors, cst.Accepted
	out := confirmPlacements(cre, spans, EquipmentPosEps(wr), &st)
	for _, p := range out {
		st.ByID[p.GlobalID]++
	}
	return out, st, nil
}

// confirmPlacements garde les records que l'oracle de position confirme, et n'en rend qu'un par
// VIE — le plus ancien, celui qui date la pose.
//
// La clé de déduplication est (slot, génération, début de la vie) et non le seul couple
// (slot, génération) : la génération ne fait que 2 bits, donc un slot repasse par la même paire
// au cours d'un match. Dédupliquer sur la paire seule fondrait deux poses distinctes du même
// socle en une, et c'est la deuxième — la plus tardive — qui disparaîtrait.
func confirmPlacements(
	cre []EquipmentCreation, spans map[EquipmentLifeKey][]EquipmentLifeSpan,
	eps [3]float32, st *EquipmentPlacementStats,
) []EquipmentPlacement {
	type lifeInstance struct {
		key   EquipmentLifeKey
		spanT uint64
	}
	best := map[lifeInstance]EquipmentPlacement{}
	for _, c := range cre {
		k := EquipmentLifeKey{c.Slot, c.Gen}
		life, hit := MatchEquipmentLife(spans[k], [3]float32{c.X, c.Y, c.Z}, eps, c.TimestampUS)
		if !hit {
			continue
		}
		st.Confirmed++
		id := lifeInstance{key: k, spanT: life.T0US}
		if cur, seen := best[id]; seen && cur.T0US <= c.TimestampUS {
			continue
		}
		best[id] = EquipmentPlacement{
			Life: k, T0US: c.TimestampUS, T1US: life.T1US,
			X: c.X, Y: c.Y, Z: c.Z,
			GlobalID: uint32(c.MPPVal[MPPWord32]), Points: life.Points,
		}
	}
	out := make([]EquipmentPlacement, 0, len(best))
	for _, p := range best {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].T0US != out[j].T0US {
			return out[i].T0US < out[j].T0US
		}
		if out[i].Life.Slot != out[j].Life.Slot {
			return out[i].Life.Slot < out[j].Life.Slot
		}
		return out[i].Life.Gen < out[j].Life.Gen
	})
	st.Placements = len(out)
	return out
}
