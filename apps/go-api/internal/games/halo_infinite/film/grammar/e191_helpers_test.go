package grammar

import "path/filepath"

// e191_helpers_test.go — les helpers des instruments E191b / E191c que des tests du build PAR
// DEFAUT emploient aussi (`build_profile_test.go`, `i0_catalogue_mutation_test.go`).
//
// POURQUOI CE FICHIER EXISTE (2026-09-16). Les instruments `e191b_carte_ti37_*` et `e191c_*`
// rebalayent les 7 bobines a chaque execution : 72 s des 89,9 s du paquet le jour de la fusion
// du lot 1.9.1 bis. Ils sont passes derriere `//go:build research` (joues a la demande :
// `go test -tags research ./internal/games/halo_infinite/film/filmdec/ -run 'TestE191'`, compiles
// en CI par `go vet -tags research`). Les quatre symboles ci-dessous etaient definis DANS ces
// instruments et lus par des tests ordinaires ; ils vivent ici, sans tag, pour que le build par
// defaut compile. Aucun corps n'est modifie — deplacement pur.

// e191cAncre est un record deja localise : son payload et son premier bit. Les ancres sont
// calculees UNE FOIS — le balayage rejoue seulement l etat par defaut, pas le scan d ancres.
type e191cAncre struct {
	Pay []byte
	Bit int
}

// e191cN2Part rend la part des records dont `n2` prend la valeur modale, et cette valeur.
func e191cN2Part(ancres []e191cAncre, ti int, ctx ContexteDeLecture) (float64, uint64, int) {
	hist := map[uint64]int{}
	total := 0
	{
		for _, a := range ancres {
			p, b := a.Pay, keyframeBorne{Bit: a.Bit, TI: ti}
			total++
			br := LecteurSur(p)
			br.PoserContexte(ctx)
			br.SetBitPos(b.Bit + keyframeFullStateHeaderBits)
			n1 := int32(br.ReadBits(keyframeFullStateSizeBits)) //nolint:gosec // 32 bits
			if n1 > 0 {
				consumeKeyframeDefaultState(br, uint32(ti)) //nolint:gosec // index d archetype
			}
			hist[br.ReadBits(keyframeFullStateSizeBits)]++
		}
	}
	meilleure, n := uint64(0), 0
	for v, c := range hist {
		if c > n || (c == n && v < meilleure) {
			meilleure, n = v, c
		}
	}
	if total == 0 {
		return 0, 0, 0
	}
	return float64(n) / float64(total), meilleure, total
}

// e191cPayloads rend tous les payloads d image-cle d une bobine.
func e191cPayloads(fc *FilmContext) [][]byte {
	var out [][]byte
	for _, num := range fc.ChunkNumbers() {
		data, packets, ok := fc.ChunkAt(num)
		if !ok {
			continue
		}
		for _, pk := range packets {
			if pk.Type == PacketTypeKeyframe {
				out = append(out, pk.Payload(data))
			}
		}
	}
	return out
}

// e191bCatalogue est le chemin du catalogue de bornes versionne, depuis ce paquet.
func e191bCatalogue() string {
	return filepath.Join("..", "..", "..", "..", "..", "..", "..", "data", "titles",
		"halo_infinite", "reference", "map_quant_bounds.json")
}
