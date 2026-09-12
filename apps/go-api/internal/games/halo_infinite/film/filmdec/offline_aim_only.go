package filmdec

// offline_aim_only.go — LA VISEE D'UN BIPEDE QUI NE REPLIQUE PLUS SA POSITION.
//
// # LE POINT AVEUGLE QUE CE FICHIER OUVRE (lot V11, 2026-09-03)
//
// `ScanBipedRecords` n'accepte un record que s'il porte un `i0` ABSOLU de la region attendue
// ET si le premier index de son masque vaut 0 (`ascendingFromZero`). C'est une exigence du
// DETECTEUR, pas du format : le masque d'un record delta n'a aucune obligation de declarer
// `i0`. Un record qui ne replique que la visee lui est donc STRUCTURELLEMENT invisible.
//
// Or c'est exactement ce que fait un occupant de vehicule. Le modele V1a.4 dit qu'il cesse de
// repliquer sa POSITION monde — c'est la primitive du « trou » sur laquelle repose toute
// l'attribution des episodes d'occupation. Il ne dit rien de sa VISEE, et la mesure du lot V11
// tranche : le flux `i21 unit-desired-aiming-vector` CONTINUE pendant le trou.
//
// # CE QUE LA MESURE ETABLIT (V11_ORIENTATION_TOURELLE_2026-09-03.md)
//
//	PRESENCE    17 838 / 4 052 / 17 911 lectures sur `0d76e8f1` / `fccc61cd` / `4898d586`,
//	            soit 173,2 / 39,0 / 172,2 par slot bipede — contre 0,5 / 0,2 / 0,1 par slot sur
//	            une bande FANTOME de meme cardinalite (slots jamais vus porter un archetype).
//	            Rapport signal / plancher de bruit : x347 a x1722.
//	JUSTESSE    appariee a la lecture `i21` AVEC i0 du meme slot a moins de 200 ms, l'ecart
//	            median de cap vaut 0,4 deg (R 0,975 / 0,983 / 0,989). TEMOIN par melange
//	            deterministe : 86,5 / 87,0 deg, R 0,047 / 0,041.
//	UTILITE     sur les episodes d'occupation ATTESTES par la sortie, 24 / 25 portent au moins
//	            une visee a bord, a 5 a 46 lectures par seconde, la ou le meme episode ne porte
//	            0 ou 1 lecture `i21` AVEC position. C'est la visee du CONDUCTEUR, de
//	            l'ARTILLEUR et du PASSAGER — chacun sur son propre slot bipede.
//
// # AUCUNE GRAMMAIRE N'EST REECRITE ICI
//
// L'en-tete est celle de `matchBipedHeaderRaw` (meme prefixe, meme slot 13 bits, meme tag == 1,
// meme couple de zeros, meme compteur de masque), a la SEULE exception documentee : le premier
// index du masque n'est pas contraint a zero. Les composants qui precedent `i21` sont consommes
// par leurs detenteurs existants (`readVelocityComponent`, `readForwardComponent`,
// `readAngularVelocityComponent`, `readBodyVitalityComponent`, `readShieldVitalityComponent`)
// et `i21` lui-meme par `readAimingVectorComponent` — le meme code, valide en production, qui
// lit la visee dans un record porteur de position.

import (
	"fmt"

	"levelup/go-api/internal/analysis/filmsource"
)

// BipedAim est une lecture de VISEE issue d'un record qui ne porte AUCUNE position.
//
// Elle ne remplace pas `BipedPosition` : elle la COMPLETE la ou celle-ci ne peut rien dire,
// c'est-a-dire pendant qu'un joueur est a bord d'un vehicule (ou, plus generalement, chaque
// fois que le moteur ne re-emet pas la position dans le meme record que la visee).
type BipedAim struct {
	// Slot est l'identifiant bas (13 bits) de l'entite — meme convention que BipedPosition.
	Slot uint32
	// Chunk / PacketIndex localisent la lecture dans le film.
	Chunk, PacketIndex int
	// TimestampUS est l'horodatage du paquet porteur (horloge du film), donc la MEME horloge
	// que BipedPosition, VehicleEvent et FireEvent.
	TimestampUS uint64
	// YawRaw / PitchRaw sont les deux scalaires bruts d'`i21` : R(12) puis R(11).
	YawRaw, PitchRaw uint32
}

// AimHeadingDeg rend le cap de visee en degres, avec la convention MESUREE et deja publiee de
// `BipedPosition.AimHeadingDeg` — le meme calcul, un seul detenteur.
func (a BipedAim) AimHeadingDeg() float32 { return aimHeadingDegFromRaw(a.YawRaw) }

// AimPitchDeg rend l'elevation de visee en degres, avec la convention MESUREE et deja publiee
// de `BipedPosition.AimPitchDeg` (reserve comprise) — le meme calcul, un seul detenteur.
func (a BipedAim) AimPitchDeg() float32 { return aimPitchDegFromRaw(a.PitchRaw) }

// ScanFilmBipedAimOnly est l'ENVELOPPE D2, HORS PRODUCTION : elle charge le film puis appelle
// [ScanBipedAimOnly]. La cuisson passe un contexte deja ouvert (une seule decompression).
func ScanFilmBipedAimOnly(dir string) ([]BipedAim, error) {
	film, err := filmsource.LoadDir(dir, nil)
	if err != nil {
		return nil, err
	}
	return ScanBipedAimOnly(NewFilmContext(film))
}

// ScanBipedAimOnly decode, sur tous les chunks d'un film DEJA CHARGE, les lectures de visee des
// records bipedes qui NE PORTENT PAS de position. La bande de slots est celle des images-cles
// `ti=35`, exactement comme [ScanBipedPositions] — et c'est la MEME, relevee une seule fois par
// le contexte du film (lot 2 de PLAN_CUISSON_PERF).
func ScanBipedAimOnly(fc *FilmContext) ([]BipedAim, error) {
	nums := fc.ChunkNumbers()
	if len(nums) == 0 {
		return nil, ErrNoFilmChunk
	}
	band := fc.BipedSlots()
	if band.Count() == 0 {
		return nil, fmt.Errorf("aucun slot biped (ti=%d) dans les keyframes du film", BipedTypeIndex)
	}
	var out []BipedAim
	read := 0
	for _, c := range nums {
		data, pks, ok := fc.ChunkAt(c)
		if !ok {
			continue
		}
		read++
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta {
				continue
			}
			for _, a := range ScanBipedAimRecords(pk.Payload(data), band) {
				a.Chunk, a.PacketIndex, a.TimestampUS = c, pk.Index, pk.TimestampUS
				out = append(out, a)
			}
		}
	}
	if read == 0 {
		return nil, ErrNoReadableFilmChunk
	}
	return out, nil
}

// ScanBipedAimRecords balaie un payload de paquet delta bit a bit et renvoie les visees des
// records bipedes SANS position. PUR (aucune I/O) : c'est le coeur testable. Les champs
// Chunk/PacketIndex/TimestampUS sont laisses a zero (remplis par l'appelant).
func ScanBipedAimRecords(payload []byte, slots SlotBand) []BipedAim {
	total := len(payload) * 8
	var out []BipedAim
	// UN SEUL lecteur de bits pour tout le payload : les composants de vitalite traverses avant
	// `i21` le repositionnent par `SetBitPos`, la ou ils en allouaient un chacun PAR CANDIDAT.
	br := NewBitReader(payload)
	for p := 0; p+bipedHeaderBits <= total; {
		at, slot, ok := matchAimOnlyRecord(br, payload, p, total, slots)
		if !ok {
			p++
			continue
		}
		var d componentDirs
		readAimingVectorComponent(payload, at, total, &d)
		if !d.HasYaw {
			p++
			continue
		}
		out = append(out, BipedAim{Slot: slot, YawRaw: d.YawRaw, PitchRaw: d.PitchRaw})
		p = at + 1 + aimYawBits + aimPitchBits // pas de re-scan chevauchant
	}
	return out
}

// matchAimOnlyRecord teste l'en-tete d'un record delta bipede dont le masque NE COMMENCE PAS
// par `i0`, consomme les composants qui precedent `i21`, et rend le bit ou commence `i21`.
//
// La seule difference avec `matchBipedHeaderRaw` est l'ABSENCE de la contrainte « premier index
// du masque = 0 » ; tout le reste (prefixe, slot, tag == 1, couple de zeros, compteur) est
// identique, et c'est ce qui rend le plancher de faux positifs mesurable par une bande fantome.
func matchAimOnlyRecord(br *BitReader, pay []byte, p, total int, slots SlotBand) (int, uint32, bool) {
	if readBitsAt(pay, p, 1) != 1 {
		return 0, 0, false
	}
	slot := readBitsAt(pay, p+1, bipedSlotBits)
	if !slots.Has(slot) {
		return 0, 0, false
	}
	if readBitsAt(pay, p+14, 2) != 1 { // tag == 1 : le filtre bipede eprouve
		return 0, 0, false
	}
	if readBitsAt(pay, p+16, 2) != 0 { // 14e bit d'id + selecteur de baseline
		return 0, 0, false
	}
	mc := int(readBitsAt(pay, p+18, 3))
	if mc < 1 || mc > bipedMaxMaskCnt {
		return 0, 0, false
	}
	if p+bipedHeaderBits+bipedIndexBits*mc > total {
		return 0, 0, false
	}
	idx, ok := ascendingMask(pay, p+bipedHeaderBits, mc)
	if !ok || idx[0] == 0 {
		return 0, 0, false // un masque qui declare i0 releve de ScanBipedRecords
	}
	at, ok := aimOnlyCursorToI21(br, pay, p+bipedHeaderBits+bipedIndexBits*mc, total, idx)
	if !ok {
		return 0, 0, false
	}
	return at, slot, true
}

// ascendingMask valide une liste d'index de composants STRICTEMENT croissants, sans exiger que
// le premier vaille zero (c'est `ascendingFromZero` moins cette seule contrainte).
func ascendingMask(pay []byte, at, count int) ([]int, bool) {
	out := make([]int, 0, count)
	prev := -1
	for k := 0; k < count; k++ {
		idx := int(readBitsAt(pay, at+bipedIndexBits*k, bipedIndexBits))
		if idx <= prev {
			return nil, false
		}
		prev = idx
		out = append(out, idx)
	}
	return out, true
}

// aimOnlyCursorToI21 consomme les composants qui PRECEDENT `i21` dans le masque et rend le bit
// ou `i21` commence. Chaque composant est consomme par SON detenteur existant : aucune largeur
// n'est reecrite ici. Un index non modelise avant `i21` arrete la lecture (le curseur ne serait
// plus fiable) — c'est la meme regle que `scanRecordDirs`.
func aimOnlyCursorToI21(br *BitReader, pay []byte, at, total int, idx []int) (int, bool) {
	var d componentDirs
	var v componentVitals
	for _, id := range idx {
		if id == 21 {
			return at, true
		}
		var ok bool
		switch id {
		case 1:
			at, ok = readVelocityComponent(pay, at, total, &d)
		case 2:
			at, ok = readForwardComponent(pay, at, total, &d)
		case 3:
			at, ok = readAngularVelocityComponent(pay, at, total)
		case 4:
			at, ok = readBodyVitalityComponent(br, at, total, &v)
		case 5:
			at, ok = readShieldVitalityComponent(br, at, total, &v)
		default:
			return at, false
		}
		if !ok {
			return at, false
		}
	}
	return at, false // i21 absent du masque
}
