package weaponv3

// melee_scanner.go — scanner des coups de melee depuis un chunk film (réf §K-bis,
// VALIDÉ sur 000d5950 = 56 swings weapon-validés).
//
// Principe (§K-bis) : le payload film est un BITSTREAM non byte-aligné. Un coup de
// melee est ancré sur une fenêtre de 8 bits valant 0x34/0x35, précédée du préfixe
// 3 bits 0b101 (obligatoire : sans lui le marqueur sur-déclenche). L'ancre est donc
// cherchée à TOUTE position de bit (pas seulement byte-alignée — le scan strictement
// byte par byte ne trouve qu'1 swing sur 56, vérifié). Les champs sont ensuite lus à
// des offsets EN BITS relatifs à l'ancre : type d'animation @+76 (∈ {0x42,0x47,0x60}),
// weapon-id 64-bit à un offset dépendant du type, player_index dans les 5 bits BAS
// de l'octet @+20 (filtré pi<=15). Le FILTRE DÉCISIF (écrase ~98 % du bruit) est la
// validation du high-32 contre KnownWeaponHigh32 (CanonWeaponID).

const (
	// meleeAnchorLo / meleeAnchorHi — valeurs d'octet d'ancre acceptées (8 bits).
	meleeAnchorLo = 0x34
	meleeAnchorHi = 0x35
	// meleeTypeOff — offset (BITS) du type-byte d'animation, relatif à l'ancre.
	meleeTypeOff = 76
	// meleePIOff — offset (BITS) de l'octet portant le player_index (5 bits bas).
	meleePIOff = 20
	// meleeMaxPI — borne haute du player_index melee (§K-bis : pi<=15).
	meleeMaxPI = 15
	// meleeDedupMS — fenêtre de déduplication d'ancres redondantes (même pi+arme).
	meleeDedupMS = 50
	// meleePrefixBits / meleePrefixVal — préfixe 0b101 attendu avant l'ancre.
	meleePrefixBits = 3
	meleePrefixVal  = 0b101
	// meleeAnchorStart — 1er bit candidat (le préfixe 0b101 occupe les 3 bits avant).
	meleeAnchorStart = meleePrefixBits

	// Type-bytes @+76 (§K-bis) — nature du coup de melee.
	meleeHitMiss   byte = 0x42 // miss / unpowered : NON-LÉTAL (whiff, pistol-whip raté)
	meleeHitHammer byte = 0x47 // HIT marteau (Gravity/Rushdown Hammer)
	meleeHitSword  byte = 0x60 // HIT épée / coup chargé (powered hit)
)

// meleeWeaponOffsets — offsets (BITS) candidats du weapon-id 64-bit par type-byte
// d'animation (relatifs à l'ancre, réf §K-bis). 0x60 a deux candidats à tester.
var meleeWeaponOffsets = map[byte][]int{
	meleeHitMiss:   {88},
	meleeHitHammer: {86},
	meleeHitSword:  {101, 103},
}

// ScanMeleeHits parcourt un chunk DÉCOMPRESSÉ et renvoie les coups de melee dont
// l'arme est validée (high-32 connu). est convertit une position d'octet en ms.
//
// L'ancre 0x34/0x35 est cherchée à toute position de bit (fenêtre 8 bits) précédée
// du préfixe 0b101. Filtrage en cascade : ancre + préfixe → type-byte@+76 ∈
// {0x42,0x47,0x60} → weapon-id high-32 connu → pi<=15. La déduplication retire les
// ancres voisines produisant le même (pi, weaponID) à moins de meleeDedupMS.
func ScanMeleeHits(chunk []byte, est func(int) float64) []MeleeHit {
	if len(chunk) == 0 {
		return nil
	}
	br := newBitReader(chunk)
	maxBitOff := maxMeleeBitOffset()
	var hits []MeleeHit
	for abit := meleeAnchorStart; abit+maxBitOff+64 <= br.total; abit++ {
		v := br.readBits(abit, 8)
		if v != meleeAnchorLo && v != meleeAnchorHi {
			continue
		}
		if br.readBits(abit-meleePrefixBits, meleePrefixBits) != meleePrefixVal {
			continue
		}
		typeByte := byte(br.readBits(abit+meleeTypeOff, 8))
		offs, ok := meleeWeaponOffsets[typeByte]
		if !ok {
			continue
		}
		if hit, ok := decodeMeleeAt(br, abit, typeByte, offs); ok {
			hit.TimeMS = int(est(abit / 8)) // est attend une position d'OCTET
			hits = append(hits, hit)
		}
	}
	return dedupMeleeHits(hits)
}

// decodeMeleeAt tente de décoder un coup de melee à la position de bit de l'ancre
// pour les offsets weapon candidats. Renvoie le premier offset dont le high-32 est
// connu et le player_index valide (pi<=15). hitType est le type-byte @+76 déjà lu
// par l'appelant (conservé dans MeleeHit.HitType). TimeMS est posé par l'appelant.
func decodeMeleeAt(br bitReader, abit int, hitType byte, offs []int) (MeleeHit, bool) {
	pi := int(br.readBits(abit+meleePIOff, 8) & 0x1f) // 5 bits BAS
	if pi > meleeMaxPI {
		return MeleeHit{}, false
	}
	for _, off := range offs {
		if abit+off+64 > br.total {
			continue
		}
		weaponID := br.readBits(abit+off, 64)
		if _, known := CanonWeaponID(weaponID); !known {
			continue
		}
		return MeleeHit{
			PI:       pi,
			WeaponID: weaponID,
			HitType:  hitType,
			AnimType: byte(br.readBits(abit+off-4, 4) & 0x0f), // nibble avant le weapon-id
		}, true
	}
	return MeleeHit{}, false
}

// dedupMeleeHits retire les coups voisins (<meleeDedupMS) produisant le même
// (pi, weaponID, hitType) — ancres qui se chevauchent au bit près produisant le
// même coup. Le hitType entre dans la clé : un HIT létal (0x47/0x60) et un miss
// (0x42) du même couple (pi,arme) sont des coups DISTINCTS et ne doivent pas se
// dédupliquer l'un l'autre (sinon un swing létal disparaît au profit d'un whiff
// voisin scanné en premier). Conserve le premier de chaque groupe (ordre de scan
// = chronologique).
func dedupMeleeHits(hits []MeleeHit) []MeleeHit {
	if len(hits) <= 1 {
		return hits
	}
	type key struct {
		pi      int
		wid     uint64
		hitType byte
	}
	last := make(map[key]int, len(hits))
	out := hits[:0:0]
	for _, h := range hits {
		k := key{pi: h.PI, wid: h.WeaponID, hitType: h.HitType}
		if prev, seen := last[k]; seen && abs(h.TimeMS-prev) < meleeDedupMS {
			continue
		}
		last[k] = h.TimeMS
		out = append(out, h)
	}
	return out
}

// maxMeleeBitOffset renvoie le plus grand offset weapon (en bits) pour borner le scan.
func maxMeleeBitOffset() int {
	m := meleeTypeOff
	for _, offs := range meleeWeaponOffsets {
		for _, o := range offs {
			if o > m {
				m = o
			}
		}
	}
	return m
}

// abs — valeur absolue d'un int (pas de dépendance math/float).
func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
