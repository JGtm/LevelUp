package filmdec

import (
	"sort"

	"levelup/go-api/internal/analysis/filmsource"
)

// keyframe_loadout.go — ARMES PORTÉES par chaque biped, lues dans les keyframes type-2.
//
// PRINCIPE, et pourquoi ce n'est PAS la chaîne de composants. L'identifiant d'arme est un
// entier 64 bits dont la moitié HAUTE est la famille (cf. weaponv3.CanonWeaponID) ;
// weapon-state-type-info le porte en clair (RE FUN_1407f06bc : porte R(1) + handle R(32) +
// nom de variante R(32)). On ne tente donc PAS de dérouler la chaîne de composants jusqu'à
// i43 : cette voie a été mesurée et REFUSÉE (le masque de présence lu à cette position dit
// « i43 présent » dans 15 records sur 184, alors que 8 joueurs portent 2 armes à chaque
// keyframe — le masque lu n'est pas le vrai masque). On balaye la charge utile à la
// recherche des 32 bits de famille et on attribue chaque occurrence au record qui la
// CONTIENT, bornes données par WalkKeyframeWorld (déjà validé 249/250 entités, 8/8 bipeds).
//
// GRAMMAIRE MESURÉE sur 150 records biped du film 000d5950 : deux emplacements d'arme par
// record, à position stable — 1er identifiant à +1950 bits (médiane) du début du record,
// son ALIAS (le même canon sous son second id) à +97, le 2e emplacement à +203 puis son
// alias à +97 ; record entier ~2800 bits. Après repli des alias sur la tête de famille :
// 2 armes dans 90 records, 3 dans 55, 4 dans 5 — le compte attendu (primaire + secondaire)
// domine.
//
// ANCRAGE ANTI-HASARD : 911 occurrences de famille sur 29 997 624 bits balayés, contre 0,52
// attendue par pur hasard (74 familles connues sur 2^32), soit ~1750x. Réparties par
// archétype du record porteur : ti=35 biped 495, ti=42 arme au sol 397, ti=21/45 divers 19 —
// répartition sémantiquement juste, et qui n'est imposée par rien dans la méthode.
//
// CE QUE CE DÉCODEUR NE DONNE PAS : ni les grenades, ni la capacité d'armure, ni les
// munitions (leurs identifiants n'ont pas d'ancre comparable) ; ni QUELLE des deux armes est
// dégainée. Un keyframe toutes les ~20 s : c'est un ÉTAT DE RÉFÉRENCE, pas un suivi continu.

// keyframeBipedTI est l'archétype (typeIndex) des records de biped joueur dans la table
// keyframe. Les armes au sol (ti=42) portent aussi un identifiant de famille : les retenir ICI
// donnerait « l'arme posée par terre » comme arme d'un joueur. Elles se lisent séparément, sous
// leur propre archétype (cf. keyframe_ground_weapons.go).
const keyframeBipedTI = 35

// KeyframeLoadout est l'ensemble des identifiants de FAMILLE d'arme trouvés dans le record
// biped d'un slot, à l'instant d'un keyframe.
type KeyframeLoadout struct {
	// TimestampUS est l'horodatage du paquet keyframe — MÊME horloge que
	// BipedPosition.TimestampUS et FireEvent.TimestampUS.
	TimestampUS uint64
	// Chunk / PacketIndex localisent le keyframe dans le film.
	Chunk, PacketIndex int
	// Slot est le slot du biped porteur (celui des trajectoires).
	Slot uint32
	// Families liste les familles (high-32 du weapon-id) dans l'ORDRE DES BITS du record.
	// Les alias ne sont PAS repliés ici : deux familles distinctes peuvent désigner le même
	// canon. Le repli est une question de NOMMAGE, il appartient à la couche qui possède le
	// catalogue d'armes (cf. replay/loadouts.go).
	Families []uint32
}

// ScanFilmKeyframeLoadouts décode les armes portées de tous les keyframes du film de dir.
// `known` est le prédicat d'appartenance au catalogue de familles : c'est LUI qui fait la
// sélectivité du balayage (un prédicat trop large rendrait du bruit — cf. l'ancrage
// anti-hasard en tête de fichier, qui suppose un catalogue de l'ordre de la centaine).
//
// HORS LIGNE (I/O disque sur tout le film) — jamais depuis un chemin de requête.
//
// ScanFilmKeyframeLoadouts est l'ENVELOPPE D2, HORS PRODUCTION ; la cuisson appelle
// [ScanKeyframeLoadouts].
func ScanFilmKeyframeLoadouts(dir string, known map[uint32]bool) ([]KeyframeLoadout, error) {
	if len(known) == 0 {
		return nil, nil // catalogue vide : rien a chercher, et rien a charger
	}
	film, err := filmsource.LoadDir(dir, nil)
	if err != nil {
		return nil, err
	}
	return ScanKeyframeLoadouts(film, known)
}

// ScanKeyframeLoadouts décode les armes portées aux images-clés d'un film DEJA CHARGE.
func ScanKeyframeLoadouts(film *filmsource.Film, known map[uint32]bool) ([]KeyframeLoadout, error) {
	if len(known) == 0 {
		return nil, nil
	}
	var out []KeyframeLoadout
	read := 0
	for _, c := range FilmChunkNumbers(film) {
		chunk, pks, ok := FilmChunkAt(film, c)
		if !ok {
			continue
		}
		read++
		for _, p := range pks {
			if p.Type != PacketTypeKeyframe {
				continue
			}
			for _, l := range keyframeLoadouts(p.Payload(chunk), known) {
				l.TimestampUS, l.Chunk, l.PacketIndex = p.TimestampUS, c, p.Index
				out = append(out, l)
			}
		}
	}
	if read == 0 {
		return nil, ErrNoReadableFilmChunk
	}
	return out, nil
}

// keyframeLoadouts balaye un payload de keyframe et rend un loadout par record biped
// porteur d'au moins une famille connue. PUR (aucune I/O).
func keyframeLoadouts(pay []byte, known map[uint32]bool) []KeyframeLoadout {
	rf := familiesByRecord(pay, known, keyframeBipedTI)
	if len(rf) == 0 {
		return nil
	}
	out := make([]KeyframeLoadout, 0, len(rf))
	for _, r := range rf {
		out = append(out, KeyframeLoadout{Slot: uint32(r.Rec.Slot), Families: r.Families})
	}
	return out
}

// recordFamilies porte les familles d'arme trouvées DANS un record de keyframe, avec le record
// qui les contient (son archétype, son slot, sa position en bits).
type recordFamilies struct {
	Rec      KeyframeRec
	Families []uint32
}

// familiesByRecord attribue chaque occurrence de famille connue au record de keyframe qui la
// CONTIENT et ne retient que les records d'archétype wantTI. L'ordre de sortie est celui de la
// PREMIÈRE occurrence de famille dans chaque record ; les alias ne sont PAS repliés (cf.
// KeyframeLoadout.Families). PUR (aucune I/O).
//
// C'est le cœur partagé des deux lectures d'armes du keyframe : les armes PORTÉES
// (wantTI = keyframeBipedTI, cf. keyframeLoadouts) et les armes AU SOL
// (wantTI = keyframeGroundWeaponTI, cf. keyframe_ground_weapons.go). Le balayage bit à bit
// est identique — seul l'archétype retenu change.
func familiesByRecord(pay []byte, known map[uint32]bool, wantTI int) []recordFamilies {
	recs := WalkKeyframeWorld(pay)
	if len(recs) == 0 {
		return nil
	}
	// L'index de recherche binaire ci-dessous exige des débuts de record CROISSANTS. Le
	// walker les émet déjà dans cet ordre ; on le garantit ici plutôt que de le supposer.
	sort.Slice(recs, func(i, j int) bool { return recs[i].Bit < recs[j].Bit })
	starts := make([]int, len(recs))
	for i, r := range recs {
		starts[i] = r.Bit
	}
	byRec := map[int][]uint32{}
	var order []int
	total := len(pay) * 8
	var w uint32
	for b := 0; b < total; b++ {
		w = w<<1 | uint32(kfBitAt(pay, b))
		if b < 31 || !known[w] {
			continue
		}
		ri := recordContaining(starts, b-31)
		if ri < 0 || recs[ri].TI != wantTI {
			continue
		}
		if byRec[ri] == nil {
			order = append(order, ri)
		}
		byRec[ri] = append(byRec[ri], w)
	}
	out := make([]recordFamilies, 0, len(order))
	for _, ri := range order {
		out = append(out, recordFamilies{Rec: recs[ri], Families: byRec[ri]})
	}
	return out
}

// recordContaining renvoie l'index du record dont l'emprise contient le bit `at`, c'est-à-dire
// le DERNIER record commençant à `at` ou avant. -1 si `at` précède le premier record.
func recordContaining(starts []int, at int) int {
	lo, hi := 0, len(starts)
	for lo < hi { // premier index dont le début est > at
		mid := (lo + hi) / 2
		if starts[mid] > at {
			hi = mid
		} else {
			lo = mid + 1
		}
	}
	return lo - 1
}
