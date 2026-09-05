package filmdec

// game_entities_bands_test.go — LES BANDES DE SLOTS ET LEURS TEMOINS.
//
// Second morceau de l'instrument des entites non corporelles (`game_entities_scan_test.go` en
// porte l'en-tete et le contrat) : le recensement des images-cles, la construction des bandes
// reelles, et les DEUX bandes de controle sans lesquelles aucun chiffre de l'ancrage ne se
// juge. Scinde du premier pour tenir le seuil de 500 lignes par fichier.

// gameEntityCensus est le recensement des images-cles : slot -> archetypes vus.
type gameEntityCensus struct {
	slotTIs map[uint32]map[int]bool
}

// gameEntityKeyframeCensus lit TOUS les records d'image-cle du film et rend, pour chaque
// slot, l'ensemble des archetypes qu'il a portes.
func gameEntityKeyframeCensus(dir string, n int) gameEntityCensus {
	c := gameEntityCensus{slotTIs: map[uint32]map[int]bool{}}
	for ch := 1; ch <= n; ch++ {
		data, err := ReadFilmChunk(dir, ch)
		if err != nil {
			continue
		}
		for _, pk := range WalkPackets(data) {
			if pk.Type != PacketTypeKeyframe {
				continue
			}
			for _, r := range WalkKeyframeWorld(pk.Payload(data)) {
				if r.Slot < 0 {
					continue
				}
				s := uint32(r.Slot)
				if c.slotTIs[s] == nil {
					c.slotTIs[s] = map[int]bool{}
				}
				c.slotTIs[s][r.TI] = true
			}
		}
	}
	return c
}

// gameEntityBands construit les bandes reelles et les deux bandes de controle.
//
// LA BANDE EST OBSERVEE, PAS COMBLEE — et c'est le contraire du choix fait pour les
// projectiles. Un projectile vit moins d'une seconde et n'apparait dans presque aucune
// image-cle : sa plage doit etre comblee pour ne pas perdre 90 % des vies
// (`worldObjectSlotBand`). Le moteur de partie et les entites joueur vivent TOUTE la partie :
// elles sont presentes a CHAQUE image-cle, donc combler ne recupere aucune couverture et ne
// peut qu'avaler les slots voisins. Le comblement est tout de meme CHIFFRE (`Filled`) pour
// que ce raisonnement soit verifiable et non seulement affirme.
func gameEntityBands(c gameEntityCensus) (map[int]map[uint32]bool, int, int) {
	bands := map[int]map[uint32]bool{
		GameEngineTypeIndex:   {},
		PlayerEngineTypeIndex: {},
		ProbeWitnessTypeIndex: {},
	}
	ambiguous, taken := 0, map[uint32]bool{}
	lo := uint32(kfTableCap)
	for slot, tis := range c.slotTIs {
		target := -1
		for ti := range tis {
			if _, ok := bands[ti]; ok {
				target = ti
			}
		}
		if target < 0 {
			continue
		}
		if len(tis) > 1 { // slot recycle : non attribuable, ecarte
			ambiguous++
			continue
		}
		bands[target][slot] = true
		taken[slot] = true
		if slot < lo {
			lo = slot
		}
	}
	filled := gameEntityFilledExcess(bands)
	real := len(bands[GameEngineTypeIndex]) + len(bands[PlayerEngineTypeIndex])
	bands[GameEntityClassNeighbour] = gameEntityControlBand(c, taken, lo, real, false)
	bands[GameEntityClassVoid] = gameEntityControlBand(c, taken, lo, real, true)
	return bands, ambiguous, filled
}

// gameEntityFilledExcess compte les slots qu'un comblement de plage AJOUTERAIT aux deux
// bandes reelles — la mesure qui justifie de ne pas combler.
func gameEntityFilledExcess(bands map[int]map[uint32]bool) int {
	excess := 0
	for _, ti := range []int{GameEngineTypeIndex, PlayerEngineTypeIndex} {
		for _, s := range fillSlotBand(bands[ti]).Slots() {
			if !bands[ti][s] {
				excess++
			}
		}
	}
	return excess
}

// gameEntityControlBand tire `size` slots LIBRES (jamais vus dans une image-cle, et non
// deja pris par une bande) : depuis `lo` vers le haut pour le temoin de voisinage, depuis le
// sommet de l'espace de slots vers le bas pour le temoin de vide.
func gameEntityControlBand(
	c gameEntityCensus, taken map[uint32]bool, lo uint32, size int, void bool,
) map[uint32]bool {
	out := map[uint32]bool{}
	if size <= 0 {
		return out
	}
	free := func(s uint32) bool { return !taken[s] && c.slotTIs[s] == nil }
	if void {
		for s := uint32(kfTableCap - 1); s > 0 && len(out) < size; s-- {
			if free(s) {
				out[s], taken[s] = true, true
			}
		}
		return out
	}
	for s := lo; s < kfTableCap && len(out) < size; s++ {
		if free(s) {
			out[s], taken[s] = true, true
		}
	}
	return out
}

// ScanFilmGameEntities balaye les paquets delta du film de `dir` et rend les records de ti=0
// et ti=5, avec les temoins d'ancrage (purete ti=4, deux bandes de controle).
