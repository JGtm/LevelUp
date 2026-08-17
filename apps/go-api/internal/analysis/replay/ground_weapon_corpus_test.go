package replay

// ground_weapon_corpus_test.go — RECENSEMENT DE CORPUS : sur quels films `ti=42` est-il
// PRÉSENT, et en quelle quantité.
//
// POURQUOI IL EXISTE. La réfutation des positions d'armes au sol du 2026-08-12 n'a été jouée
// que sur UN film (`000d5950`). La rejouer sur plusieurs cartes suppose de CHOISIR des films
// où l'archétype est présent — et la contre-liste du lot R6 le dit déjà pour trois films
// (`ti=42` x4 à x21 par chunk sur `000d5950`, x0 sur `00502e52` et `07aa428d`). Choisir sur
// une intuition de mode de jeu reproduirait ce piège ; ce recensement le remplace par un
// compte.
//
// CE QU'IL MESURE, par film : le nombre d'images-clés, le nombre de records `ti=42` qu'elles
// portent, le nombre de slots distincts, et la BANDE de slots (`GroundWeaponSlotBand`) — la
// bande est ce que le décodeur de positions consomme, donc c'est elle qui décide si une mesure
// de position a une matière.
//
// LECTURE SEULE, aucune base, aucune écriture. SOUS GARDE D'ENVIRONNEMENT (GW_CORPUS).
//
// USAGE (depuis apps/go-api) :
//
//	CGO_ENABLED=0 GW_CORPUS=<repo>/data/cache/film_chunks/000d5950,<repo>/data/cache/film_chunks/... \
//	  go test ./internal/analysis/replay/ -run GroundWeaponCorpus -timeout 60m -v

import (
	"os"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

const gwCorpusEnv = "GW_CORPUS" // liste de répertoires de films, séparés par des virgules

func TestGroundWeaponCorpusCensus(t *testing.T) {
	raw := os.Getenv(gwCorpusEnv)
	if raw == "" {
		t.Skipf("%s absent : recensement de corpus sauté", gwCorpusEnv)
	}
	release := filmdec.LockProcessDecode()
	defer release()

	for _, dir := range strings.Split(raw, ",") {
		dir = strings.TrimSpace(dir)
		if dir == "" {
			continue
		}
		c := gwCensusOnly(dir)
		band := filmdec.GroundWeaponSlotBand(dir)
		t.Logf("%-64s images-clés %3d · records ti=42 %5d · slots ti=42 %4d · bande %4d"+
			" · records ti=37 %5d · records ti=35 %5d",
			dir, c.keyframes, c.recordsTI[filmdec.GroundWeaponTypeIndex],
			len(c.slotsTI[filmdec.GroundWeaponTypeIndex]), len(band),
			c.recordsTI[filmdec.EquipmentTypeIndex], c.recordsTI[35])
	}
}

// gwCensusOnly recense les records d'image-clé par archétype SANS `testing.T` : le
// recensement de corpus ne doit pas s'arrêter sur un film illisible, il doit le dire.
func gwCensusOnly(dir string) gwCensus {
	c := gwCensus{recordsTI: map[int]int{}, slotsTI: map[int]map[uint32]bool{}, allSlots: map[uint32]bool{}}
	n := filmdec.CountFilmChunks(dir)
	for i := 1; i <= n; i++ {
		data, err := filmdec.ReadFilmChunk(dir, i)
		if err != nil {
			continue
		}
		for _, p := range filmdec.WalkPackets(data) {
			if p.Type != filmdec.PacketTypeKeyframe {
				continue
			}
			c.keyframes++
			for _, r := range filmdec.WalkKeyframeWorld(p.Payload(data)) {
				c.recordsTI[r.TI]++
				if c.slotsTI[r.TI] == nil {
					c.slotsTI[r.TI] = map[uint32]bool{}
				}
				c.slotsTI[r.TI][uint32(r.Slot)] = true
				c.allSlots[uint32(r.Slot)] = true
			}
		}
	}
	return c
}
