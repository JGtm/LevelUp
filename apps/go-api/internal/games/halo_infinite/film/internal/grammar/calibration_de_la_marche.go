package grammar

// calibration_de_la_marche.go — LE CRITERE DE LA CALIBRATION DE `killsource`, SOUS LE MONDE DES
// PRELIMINAIRES DE LA MARCHE DES TRAMES.
//
// killsource calibre deux grandeurs que le film ne porte pas : l oracle de largeur d axe et la
// largeur du mot de poignee. Son critere compte, pour chaque cadre candidat, les records de bipede lus
// sans desynchronisation sur un echantillon fige de trames sans evenement, marchees depuis leur bit 2
// (aucun localisateur), sous le monde que la marche des trames construit avant ses trames — chunk
// par chunk, la table anticipee puis la liaison des images-cles du chunk
// ([lierLesImagesClesDuChunk]) —, le monde restaure apres chaque essai. La decision (dominance de la
// mediane) reste a killsource.

import (
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// EchantillonDeCalibration fixe l echantillon du critere : les `Taille` premieres trames delta du
// film, dans l ordre de la source, sans liste d evenements et d au moins `OctetsMin` octets.
type EchantillonDeCalibration struct{ Taille, OctetsMin int }

// ScoresDeCalibration rend, pour chaque cadre de `cadres`, le nombre de records de bipede lus sans
// desynchronisation sur l echantillon, chaque trame marchee depuis son bit 2 sur `vues` vues sous le
// monde des preliminaires de la marche ; et la taille de l echantillon. Les cadres ne different que
// par leur profil : l identifiant bas de chacun est celui de l en-tete de la marche ([EnTete]).
func (c *FilmContext) ScoresDeCalibration(cadres []FrameConfig, ech EchantillonDeCalibration, vues int) (
	[]int, int, error,
) {
	m, err := c.nouveauMarcheurDesTrames(nil)
	if err != nil {
		return nil, 0, err
	}
	parPosition, n := echantillonParChunk(c.Film(), ech)
	scores := make([]int, len(cadres))
	m.parcourirLesPreliminaires(func(num int) {
		for _, p := range parPosition[filmChunkPos(c.Film(), num)] {
			for k := range cadres {
				cadre := cadres[k]
				cadre.IDLowBits = m.cfg.IDLowBits
				scores[k] += recordsBipedesPropres(p.Payload, m.monde, cadre, vues)
			}
		}
	})
	return scores, n, nil
}

// echantillonParChunk rend l echantillon range par position de chunk, et sa taille.
func echantillonParChunk(f *source.Film, ech EchantillonDeCalibration) (map[int][]types.Packet, int) {
	out := map[int][]types.Packet{}
	n := 0
	for _, p := range f.AllPackets() {
		if n >= ech.Taille {
			break
		}
		if p.Type != int(PacketTypeDelta) || len(p.Payload) < ech.OctetsMin || source.BitAt(p.Payload, 1) != 0 {
			continue
		}
		out[p.Chunk] = append(out[p.Chunk], p)
		n++
	}
	return out, n
}

// recordsBipedesPropres marche `pay` depuis son bit 2 sur `vues` vues sous `cadre`, le monde restaure
// ensuite, et rend le nombre de records de bipede entierement portes.
func recordsBipedesPropres(pay []byte, w *World, cadre FrameConfig, vues int) int {
	snap := w.Snapshot()
	defer w.Restore(snap)
	br := LecteurSur(pay)
	br.Skip(2)
	n := 0
	for v := 0; v < vues && len(pay)*8-br.BitPos() >= 8; v++ {
		recs, err := DecodeFrameRecords(br, w, cadre)
		for k := range recs {
			if recs[k].DesyncAt == -1 && recs[k].TypeIndex == BipedTypeIndex {
				n++
			}
		}
		if err != nil {
			break
		}
	}
	return n
}
