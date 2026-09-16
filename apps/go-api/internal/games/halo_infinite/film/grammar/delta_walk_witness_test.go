package grammar

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
//	  go test ./internal/games/halo_infinite/film/filmdec/ -run TestDeltaWalkWitness -v

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
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
// MESURES LE 2026-08-17 (lot 0, item 0.1), RE-FIGEES LE 2026-09-14 (item 1.4.0) sur les trois
// films de reference du corpus. Ces valeurs ne se « rafraichissent » pas : si l'une bouge,
// c'est la GRAMMAIRE qui a bouge, et c'est ce qu'il faut expliquer avant de reecrire le
// chiffre — l'attribution des quatre marches du 2026-08-18 au 2026-09-14 est ecrite plus bas,
// et c'est la SEULE forme acceptable d'un re-figeage de ce fichier.
//
// CE FICHIER EST SOUS GARDE D'ENVIRONNEMENT (`DELTA_WITNESS_FILM`) : la CI ne le joue JAMAIS.
// Un lot qui touche la traversee doit donc le jouer A LA MAIN sur les trois films, et
// consigner ce qu'il y voit. Ne pas le jouer, c'est laisser la derive s'accumuler jusqu'a ce
// qu'elle ne soit plus attribuable — c'est exactement ce qui s'est produit entre le
// 2026-08-18 et le 2026-09-14, et ce que l'item 1.4.0 a du defaire par bisection.
var deltaWitnessFrozen = map[string]deltaWitnessCounts{
	"000d5950": {packets: 14350, records: 38945, walked: 30118}, // 77,335 % aboutis
	"06dfe6d9": {packets: 6606, records: 10636, walked: 8505},   // 79,964 % aboutis
	"64e8adfa": {packets: 14357, records: 39936, walked: 31933}, // 79,960 % aboutis
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
//
// POURQUOI CES COMPTES ONT BOUGE ENTRE LE 2026-08-18 ET LE 2026-09-14 (item 1.4.0 du
// PLAN_DECODEUR_FILM, attribution exigee par la decouverte D1 (1.3)) — QUATRE fois, par QUATRE
// lots, dont AUCUN n'a tenu le contrat ci-dessus. Le temoin est sous garde d'environnement :
// la CI ne le joue jamais, et c'est pour cela que la derive a pu traverser 0.A a 1.3 sans etre
// consignee. Les comptes sont re-figes ici a l'etat du commit d'integration `15309e89e`
// (cloture du lot 1.3), APRES attribution de chaque marche.
//
// LA BISECTION (chaine premier-parent 4ad72a4a1..8f35efb72, 690 points, temoin `06dfe6d9`) —
// records rendus / traversees abouties :
//
//	4ad72a4a1 2026-08-18  10 613 / 8 502   le fige d'origine, CONFORME
//	62ba098b8 2026-09-01  10 615 / 8 504   merge wt/bombe-visuel (la bombe d'Assaut au rejeu)
//	8f309ce86 2026-09-02  10 610 / 8 497   merge feat/precision-arme (acquis backend)
//	736ccf3c3 2026-09-05  10 610 / 8 489   merge cuisson-perf + vehicules (schema 39)
//	ffb27238c 2026-09-11  10 627 / 8 499   merge wt/munitions-objet (grammaire d'i9 retablie)
//	8f35efb72 .. 191933992               10 627 / 8 499   M0, 0.D, 1.0, 1.1 : AUCUN mouvement
//	783ae680d 2026-09-14  10 629 / 8 499   lot 1.2 (le registre a l'octet 8)
//	15309e89e 2026-09-14  10 636 / 8 505   lot 1.3 (les cinq etats par defaut)
//
// LE « SENS QUE LE CONTRAT REFUSE » EST UN ARTEFACT D'AGREGATION, ET C'EST LE RESULTAT DE
// L'ITEM. D1 (1.3) relevait que sur `06dfe6d9` les records MONTAIENT (+16) pendant que les
// traversees abouties DESCENDAIENT (-3). Aucune marche ne fait cela : pris un a un, chaque
// point est coherent (+2/+2, -5/-7, 0/-8, +17/+10, +2/0, +7/+6). Ce n'est pas un lot qui a
// produit un sens impossible, c'est un temoin fige laisse en place pendant QUATRE lots.
//
// CHAQUE MARCHE EST UNE DIVERGENCE (une lecture du jeu ajoutee ou corrigee), AUCUNE N'EST UNE
// REGRESSION, et le point le plus suspect est celui qui le prouve le mieux :
//
//	8f309ce86 (-7 aboutis) — la sonde par archetype ne laisse QU'UNE ligne changer de verdict :
//	  `ti=49` passe de 0/7 non portes a 7/7. Cause mesuree : avant ce merge, `parseRegistry`
//	  decoupait le chunk_00 par « taille du fichier / taille d'un bloc » et resolvait 64
//	  archetypes sur `06dfe6d9`, dont ti=49 a 63 avec ZERO composant — du bourrage. Une
//	  traversee sur un archetype a zero composant se termine sans rien lire, donc ELLE
//	  ABOUTISSAIT. Depuis, le registre s'arrete a sa fin STRUCTURELLE : 49 archetypes sur ce
//	  film, ti=49 ABSENT. Les 7 traversees perdues ne lisaient rien — les perdre est le
//	  correctif, pas la regression.
//	62ba098b8 (+2/+2) et ffb27238c (+17/+10) — des composants portes en plus (objectif/bombe,
//	  puis la grammaire d'i9 relue au desassemblage) : l'espece de derive que le contrat accepte.
//	736ccf3c3 (-8 aboutis) — les largeurs mesurees du chantier vehicules (dont l'etat par defaut
//	  de ti=40) et les bandes de slots ; la perte se concentre sur ti=0 (-6), ti=33 (-1) et
//	  ti=38 (-1), aucune ligne ne bascule en bloc.
//	783ae680d (lot 1.2, +2/0) et 15309e89e (lot 1.3, +7/+6) — les deux lots du chantier, dont
//	  l'effet etait attendu et deja chiffre dans leur propre journal.
//
// AUCUN DE CES QUATRE POINTS N'EST DANS LE PERIMETRE DU CADRE D'IMAGE-CLE (lot 1.4) : ils
// vivent tous dans la marche DELTA et dans le registre. Le report est consigne au plan (§4 et
// bloc « Cloture M1 »).

// TestDeltaWalkWitness : la mesure, confrontee au fige quand le film est connu.
func TestDeltaWalkWitness(t *testing.T) {
	dir := os.Getenv(deltaWitnessFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : temoin de marche delta saute", deltaWitnessFilmEnv)
	}

	got, parTI := deltaWitnessMeasure(t, dir)
	id := filepath.Base(filepath.Clean(dir))
	t.Logf("== FILM %s (%d premier(s) chunk(s) de replication) ==", id, deltaWitnessChunks)
	t.Logf("  paquets delta lus : %d", got.packets)
	t.Logf("  records rendus : %d · traversee ABOUTIE (DesyncAt == -1) : %d (%.3f %%) · "+
		"ported=false : %d", got.records, got.walked,
		deltaWitnessPct(got.walked, got.records), got.records-got.walked)
	t.Logf("  records NEW par archetype (ceux qui jouent l'etat par defaut) : %s",
		deltaWitnessHisto(parTI))

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
//
// LE SECOND RENDU EST L'HISTOGRAMME DES RECORDS `recNew` PAR ARCHETYPE (lot 1.3, 2026-09-14).
// C'est la POPULATION EXACTE qu'une entree de `defaultStateDeserByTI` change : `TraverseEntity`
// est le seul lecteur de production qui consulte cette table, et il ne la consulte que sur un
// record NEW. L'histogramme dit donc, film par film, combien de records un etat par defaut neuf
// touche — avant d'ecrire la moindre ligne. Il n'est PAS fige (il depend du film) : ce sont les
// trois comptes de `deltaWitnessFrozen` qui gardent la marche.
func deltaWitnessMeasure(t *testing.T, dir string) (deltaWitnessCounts, map[uint32]int) {
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
	parTI := map[uint32]int{}
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
			br := LecteurSur(pk.Payload(data))
			recs, _ := DecodeFrameRecords(br, w, cfg)
			for i := range recs {
				out.records++
				if recs[i].Type == recNew {
					parTI[recs[i].TypeIndex]++
				}
				if recs[i].DesyncAt == -1 {
					out.walked++
				}
			}
		}
	}
	return out, parTI
}

// deltaWitnessHisto rend l'histogramme par archetype, trie, avec son total — la forme est faite
// pour etre COLLEE telle quelle dans un journal de lot.
func deltaWitnessHisto(parTI map[uint32]int) string {
	tis := make([]int, 0, len(parTI))
	total := 0
	for ti, n := range parTI {
		tis = append(tis, int(ti)) //nolint:gosec // TypeIndex tient sur 6 bits (0..63)
		total += n
	}
	sort.Ints(tis)
	var b strings.Builder
	for _, ti := range tis {
		fmt.Fprintf(&b, "ti=%d:%d ", ti, parTI[uint32(ti)]) //nolint:gosec // ti vient d'une cle uint32
	}
	fmt.Fprintf(&b, "| total NEW %d", total)
	return b.String()
}

// deltaWitnessPct : pourcentage a denominateur jamais nul.
func deltaWitnessPct(num, den int) float64 {
	if den <= 0 {
		return 0
	}
	return 100 * float64(num) / float64(den)
}
