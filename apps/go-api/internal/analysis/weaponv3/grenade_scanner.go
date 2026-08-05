package weaponv3

// grenade_scanner.go — scanner des lancers de grenade depuis un chunk film
// (réf §C). Le marqueur 24-bit 0x4c0c00 est un marqueur object/equipment-id
// GÉNÉRAL : seuls les hits dont l'id décode dans l'allowlist 4-grenades sont de
// vrais lancers. La confiance qu'un hit valide = vraie grenade est HAUTE
// (faux positifs ~0 parmi les valides) ; la complétude est basse (markers sparse).
//
// Le marqueur grenade n'expose AUCUN player_index validé (§C) — GrenadeThrow ne
// porte donc pas de PI (ne pas en inventer un).

import "encoding/binary"

// Allowlist des grenades (high-32 directement, réf §C). Seules ces valeurs sont
// renvoyées : tout autre id décodé après le marqueur est un object-id générique.
const (
	GrenadeFrag   uint32 = 0xB0171062
	GrenadePlasma uint32 = 0xC0E34C44
	GrenadeShock  uint32 = 0x3B2567D4
	GrenadeSpike  uint32 = 0x9212E428
)

// grenadeMarker — préfixe 3 octets du marqueur object-id (0x4c 0x0c 0x00).
var grenadeMarker = [3]byte{0x4c, 0x0c, 0x00}

// grenadeAllowlist — set des grenades valides (clé = id 32-bit).
var grenadeAllowlist = map[uint32]struct{}{
	GrenadeFrag:   {},
	GrenadePlasma: {},
	GrenadeShock:  {},
	GrenadeSpike:  {},
}

// isGrenadeID indique si un id 32-bit appartient à l'allowlist 4-grenades.
func isGrenadeID(id uint32) bool {
	_, ok := grenadeAllowlist[id]
	return ok
}

// ScanGrenadeThrows parcourt un chunk DÉCOMPRESSÉ byte par byte, repère le
// marqueur 0x4c0c00 et lit le weapon32 juste APRÈS (marqueur+3 octets). Seuls les
// ids de l'allowlist sont renvoyés. est convertit une position d'octet en ms.
//
// Orientation : §C verrouille le weapon-id à marqueur+24 bits ; en lecture
// byte-alignée cela donne BigEndian.Uint32(chunk[p+3:p+7]). L'invariant testé est
// que TOUT hit renvoyé décode dans l'allowlist — l'orientation BE est confirmée
// par les hits Frag/Plasma observés (cf. grenade_scanner_test.go).
func ScanGrenadeThrows(chunk []byte, est func(int) float64) []GrenadeThrow {
	var throws []GrenadeThrow
	// Borne : il faut le marqueur (3 o) + le weapon32 (4 o) après p.
	for p := 0; p+7 <= len(chunk); p++ {
		if chunk[p] != grenadeMarker[0] || chunk[p+1] != grenadeMarker[1] || chunk[p+2] != grenadeMarker[2] {
			continue
		}
		id := binary.BigEndian.Uint32(chunk[p+3 : p+7])
		if !isGrenadeID(id) {
			continue
		}
		throws = append(throws, GrenadeThrow{
			TimeMS:   int(est(p)),
			WeaponID: id,
		})
	}
	return throws
}
