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
	"000d5950": {packets: 14350, records: 38878, walked: 30080}, // 77,370 % aboutis
	"06dfe6d9": {packets: 6606, records: 10613, walked: 8502},   // 80,109 % aboutis
	"64e8adfa": {packets: 14357, records: 39806, walked: 31973}, // 80,322 % aboutis
}

// POURQUOI CES COMPTES ONT BOUGE LE 2026-08-18, PREMIERE FOIS (lot C phase 1b, item C.1b.1) — la
// raison est exigee par le contrat ci-dessus, et elle est de la seule espece acceptable : la
// grammaire a GAGNE des composants. Trois desers ont ete portes
// (`managed-navpoint-radial-progress`, `managed-object-boundary-color-component`,
// `managed-object-rtpc-component`), donc des records qui desynchronisaient aboutissent.
// Mesure sur les douze films, avant -> apres : aucun film ne recule, sept progressent.
// 000d5950 30 058 -> 30 060 · 64e8adfa 31 934 -> 31 935 · 06dfe6d9 8 494 (inchange)
// 7344d24f 25 016 -> 25 021 · 696a9d7c 24 652 -> 24 653 · 01e1f945 29 138 -> 29 140
// 530820e5 26 238 -> 26 241 · 53ce4390 28 584 -> 28 586 · 0a247154, 606d9844, 8076f97f,
// 24dbb67d inchanges. Detail : `.ai/V7.5/replay2d/registre_film/LOTC_PHASE1B.md`.
//
// POURQUOI CES COMPTES ONT BOUGE LE 2026-08-18, DEUXIEME FOIS (lot C-bis phase 1, item CB.1.1) —
// meme espece de raison que la premiere : la grammaire a GAGNE des composants. Les deux desers de
// ti=13 ont ete portes (`managed-object-property-component` en mode A et
// `managed-object-player-masked-property-component` en mode B, les 32 instances), donc des records
// qui desynchronisaient aboutissent, et la marche sequentielle qui s arretait sur eux continue et
// decouvre des records de plus loin dans le paquet.
//
// Mesure sur les DOUZE films du corpus, avant -> apres : AUCUN film ne recule, LES DOUZE
// progressent (le lot C n en avait fait progresser que sept).
// 7344d24f 25 021 -> 25 149 (+128) · 8076f97f 24 889 -> 24 943 (+54) · 696a9d7c 24 653 -> 24 693
// (+40) · 64e8adfa 31 935 -> 31 973 (+38) · 24dbb67d 30 917 -> 30 954 (+37) · 530820e5 26 241 ->
// 26 273 (+32) · 53ce4390 28 586 -> 28 613 (+27) · 01e1f945 29 140 -> 29 162 (+22) · 000d5950
// 30 060 -> 30 080 (+20) · 0a247154 24 468 -> 24 484 (+16) · 606d9844 26 642 -> 26 657 (+15) ·
// 06dfe6d9 8 494 -> 8 502 (+8). Detail : `.ai/V7.5/replay2d/registre_film/LOTCBIS_PHASE1.md`.
//
// Le gain reste MODESTE, et pour la meme raison qu au lot C : une traversee n aboutit que si TOUS
// les composants annonces sont portes, et ti=13 n a que 34 composants sur un trafic domine par le
// bipede. Ce qui est neuf, c est que le gain touche LES DOUZE films, y compris le temoin Slayer ou
// ti=13 est pourtant quasi muet — signe que les records concernes ne sont pas seulement ceux de la
// bande ti=13 mais aussi ceux qui la SUIVENT dans le paquet.

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
