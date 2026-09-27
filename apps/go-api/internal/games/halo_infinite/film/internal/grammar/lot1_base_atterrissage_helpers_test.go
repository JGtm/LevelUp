package grammar

// lot1_base_atterrissage_helpers_test.go — LA BASE D'ATTERRISSAGE BIPEDE DES DEGATS, calculee pour
// les instruments seulement (J10.2, GB-4 de l'audit du 2026-09-24).
//
// `ScanFilmWeaponDamages` rendait cette base en production : il rejouait le monde de chaque chunk
// (images-cles, puis `DecodeFrameRecords` dont l'erreur etait jetee) pour compter, par base
// candidate, les index bruts qui atterrissent sur un bipede. Aucun appelant de production ne la
// lisait. Le calcul vit desormais ici, a l'identique, pour les sondes qui s'en servent encore
// (`lot1_sonde_precision`, `lot1_attrib_arme_tir`) et pour les instruments `lot1_*` qui
// appellent `lot1chIsBiped` / `lot1ArgmaxBase` sur leur propre monde.

import "testing"

// lot1chIsBiped indique si l'index brut idx, rapporte a la base, atterrit sur un slot lie a un
// bipede dans le monde reconstruit w.
func lot1chIsBiped(w *World, base, idx int) bool {
	if idx < 0 {
		return false
	}
	slot := base + idx
	if slot < 0 || slot >= 8192 {
		return false
	}
	ti, ok := w.ArchetypeForSlot(uint32(slot))
	return ok && ti == BipedTypeIndex
}

// lot1ArgmaxBase rend la base a l'atterrissage bipede maximal (base la plus basse en cas d'egalite).
func lot1ArgmaxBase(hits map[int]int) int {
	best, bestN := lot1chBases[0], -1
	for _, b := range lot1chBases {
		if hits[b] > bestN {
			best, bestN = b, hits[b]
		}
	}
	return best
}

// lot1BaseAtterrissageDegats rend l'argmax des bases ou les degats des chunks 1..n atterrissent sur
// un bipede, dans le monde de FIN de chaque chunk (le calcul que `ScanFilmWeaponDamages` faisait
// avant J10.2). Une trame que `DecodeFrameRecords` ne traverse pas est comptee et journalisee.
func lot1BaseAtterrissageDegats(t *testing.T, dir string, reg *Registry, n int) int {
	t.Helper()
	cfg := DefaultFrameConfig()
	hit := map[int]int{}
	enErreur := 0
	for c := 1; c <= n; c++ {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			t.Fatalf("chunk_%02d illisible : %v", c, err)
		}
		pks := WalkPackets(data)
		w := NewWorld(reg)
		for _, pk := range pks {
			if pk.Type != PacketTypeKeyframe {
				continue
			}
			for _, r := range WalkKeyframeWorld(pk.Payload(data)) {
				w.BindFull(uint32((r.Gen<<30)|r.Slot), uint32(r.TI))
			}
		}
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta || pk.Size < 1 {
				continue
			}
			if pay := pk.Payload(data); pay[0]&0x40 == 0 {
				if _, err := DecodeFrameRecords(LecteurSur(pay), w, cfg); err != nil {
					enErreur++
				}
			}
		}
		for _, d := range scanChunkDamages(pks, data, nil) {
			for _, b := range lot1chBases {
				if lot1chIsBiped(w, b, d.VictimIdx) {
					hit[b]++
				}
				if lot1chIsBiped(w, b, d.ResponsibleIdx) {
					hit[b]++
				}
			}
		}
	}
	if enErreur > 0 {
		t.Logf("base d'atterrissage : %d trames non traversees par DecodeFrameRecords", enErreur)
	}
	return lot1ArgmaxBase(hit)
}
