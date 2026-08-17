package filmdec

// delta_walk_witness_test.go — LE TEMOIN CHIFFRE DE LA MARCHE DELTA, sur les films du corpus.
//
// A QUOI IL SERT, ET POURQUOI IL EST UNIQUE. Deux besoins distincts demandaient la meme
// mesure, et une seule instance existe donc pour les deux :
//
//	(a) le golden de decodage delta de la polarite d'i9 (lot 0, item 0.1) — un compte FIGE de
//	    records dont la traversee ABOUTIT ; une inversion de porte le fait bouger ;
//	(b) le controle « les hooks ne changent pas un bit » de la plomberie de publication
//	    (lot 0, item 0.6) — les memes comptes, AVANT et APRES le deplacement des `case`.
//
// CE QU'IL MESURE. Pour un film donne, sur les `deltaWitnessChunks` PREMIERS chunks de
// replication : le nombre de paquets delta lus, le nombre de records rendus par
// `DecodeFrameRecords`, et parmi eux le nombre dont `DesyncAt == -1` (traversee aboutie). Le
// complement — `records - walked` — est le compte de records `ported=false` que le plan
// nomme.
//
// CE QU'IL N'EST PAS, et ne pretend pas etre. Le monde est amorce par les declarations
// d'image-cle DU CHUNK COURANT, dans l'ordre du chunk. Ce n'est pas la reconstruction
// chronologique de `killsource.timeline` (qui gere le recyclage de slot) : un slot recycle
// peut donc etre lu sous le mauvais archetype. C'est DELIBERE — l'instrument est un TEMOIN DE
// COMPARABILITE, deterministe et sensible, pas un oracle de justesse. Sa valeur absolue ne
// dit rien de la qualite du decodage ; seule sa VARIATION entre deux versions du code compte.
//
// UN SEUL FILM PAR PROCESS (memoire du depot, deux plantages machine en aout) : la garde
// nomme UN chemin, jamais une liste. LECTURE SEULE, aucune ecriture disque. Le verrou de
// process est pris : la marche ecrit des globaux de paquet.
//
// USAGE (depuis apps/go-api, un film a la fois, en avant-plan) :
//
//	CGO_ENABLED=0 DELTA_WITNESS_FILM=C:/Users/Guillaume/Projects/LevelUp/data/cache/film_chunks/000d5950 \
//	  go test ./internal/analysis/filmdec/ -run TestDeltaWalkWitness -v

import (
	"os"
	"path/filepath"
	"testing"
)

// deltaWitnessFilmEnv nomme le repertoire du film a mesurer (chemin ABSOLU).
const deltaWitnessFilmEnv = "DELTA_WITNESS_FILM"

// deltaWitnessChunks : nombre de chunks de replication parcourus. Douze suffisent a rendre
// des comptes a quatre ou cinq chiffres sur les trois films de reference — le premier tiers du
// corpus rend 276 records seulement sur 06dfe6d9, trop peu pour qu une derive s y voie —
// et bornent le cout a quelques secondes par film.
const deltaWitnessChunks = 12

// deltaWitnessCounts : ce qu'une passe rend.
type deltaWitnessCounts struct {
	packets, records, walked int
}

// deltaWitnessFrozen : les comptes FIGES, par identifiant de film (nom du repertoire).
//
// MESURES LE 2026-08-17 (lot 0, item 0.1), sur les trois films de reference du corpus. Ces
// valeurs ne se « rafraichissent » pas : si l'une bouge, c'est la GRAMMAIRE qui a bouge, et
// c'est ce qu'il faut expliquer avant de reecrire le chiffre.
var deltaWitnessFrozen = map[string]deltaWitnessCounts{
	"000d5950": {packets: 14350, records: 38860, walked: 30058}, // 77,349 % aboutis
	"06dfe6d9": {packets: 6606, records: 10607, walked: 8494},   // 80,079 % aboutis
	"64e8adfa": {packets: 14357, records: 39776, walked: 31934}, // 80,285 % aboutis
}

// TestDeltaWalkWitness : la mesure, confrontee au fige quand le film est connu.
func TestDeltaWalkWitness(t *testing.T) {
	dir := os.Getenv(deltaWitnessFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : temoin de marche delta saute", deltaWitnessFilmEnv)
	}
	release := LockProcessDecode()
	defer release()

	got := deltaWitnessMeasure(t, dir)
	id := filepath.Base(filepath.Clean(dir))
	t.Logf("== FILM %s (%d premier(s) chunk(s) de replication) ==", id, deltaWitnessChunks)
	t.Logf("  paquets delta lus : %d", got.packets)
	t.Logf("  records rendus : %d · traversee ABOUTIE (DesyncAt == -1) : %d (%.3f %%) · "+
		"ported=false : %d", got.records, got.walked,
		deltaWitnessPct(got.walked, got.records), got.records-got.walked)

	want, ok := deltaWitnessFrozen[id]
	if !ok {
		t.Logf("  film HORS TABLE FIGEE : la mesure est publiee, rien n'est confronte")
		return
	}
	if got != want {
		t.Fatalf("les comptes ONT BOUGE sur %s : mesure {paquets %d records %d aboutis %d}, "+
			"fige {paquets %d records %d aboutis %d} — la grammaire de la traversee a change",
			id, got.packets, got.records, got.walked, want.packets, want.records, want.walked)
	}
	t.Logf("  CONFORME au compte fige")
}

// deltaWitnessMeasure parcourt les chunks et agrege. Le monde est amorce par les images-cles
// du chunk courant avant que ses paquets delta ne soient lus.
func deltaWitnessMeasure(t *testing.T, dir string) deltaWitnessCounts {
	t.Helper()
	raw, err := ReadFilmChunk(dir, 0)
	if err != nil {
		t.Fatalf("chunk_00 (registre) illisible dans %s : %v", dir, err)
	}
	reg, err := ParseRegistryChunk(raw)
	if err != nil {
		t.Fatalf("registre illisible dans %s : %v", dir, err)
	}
	n := CountFilmChunks(dir)
	if n < deltaWitnessChunks {
		t.Fatalf("%s ne porte que %d chunk(s) de replication, %d attendus", dir, n, deltaWitnessChunks)
	}
	cfg := DefaultFrameConfig()
	var out deltaWitnessCounts
	for c := 1; c <= deltaWitnessChunks; c++ {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			t.Fatalf("chunk_%02d de %s illisible : %v", c, dir, err)
		}
		w := NewWorld(reg)
		pks := WalkPackets(data)
		for _, pk := range pks {
			if pk.Type != PacketTypeKeyframe {
				continue
			}
			for _, r := range WalkKeyframeWorld(pk.Payload(data)) {
				w.BindFull(uint32((r.Gen<<30)|r.Slot), uint32(r.TI))
			}
		}
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta {
				continue
			}
			out.packets++
			br := NewBitReader(pk.Payload(data))
			recs, _ := DecodeFrameRecords(br, w, cfg)
			for i := range recs {
				out.records++
				if recs[i].DesyncAt == -1 {
					out.walked++
				}
			}
		}
	}
	return out
}

// deltaWitnessPct : pourcentage a denominateur jamais nul.
func deltaWitnessPct(num, den int) float64 {
	if den <= 0 {
		return 0
	}
	return 100 * float64(num) / float64(den)
}
