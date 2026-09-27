package grammar

import (
	"fmt"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/weaponv3"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// player_index.go — L'INDEX DE JOUEUR SE LIT DANS LE FILM.
//
// CE QUE CE FICHIER REMPLACE. Le rejeu résolvait le lien « index de joueur du film ->
// identité » par une AFFECTATION DE COÛT MINIMAL sur les 8! permutations : on gardait celle
// qui se contredisait le moins (un joueur ne tire pas pendant qu'il est mort). Cela marchait
// — la table obtenue est la bonne — mais c'était un CHOIX, et sa marge était étroite : 32
// contradictions contre 39 pour la deuxième permutation.
//
// LE LIEN EST ÉCRIT, ET IL SUFFIT DE LE LIRE. Le xuid d'un joueur figure dans le film sur
// 8 octets petit-boutiste, et les CINQ BITS qui le précèdent immédiatement portent son index.
// C'est la méthode établie par le chantier voisin (`filmdec-killweapon`, lot `co.pi`), qui la
// mesure sur 116 films sur 116 et l'a portée dans `weaponv3.ResolveXuidToPI`. Elle vivait déjà
// dans ce dépôt ; elle n'avait jamais été branchée sur le rejeu.
//
// LE PIÈGE, DOCUMENTÉ PAR LE VOISIN ET REPRODUIT ICI. Appliqué au chunk 0 (le registre) ou au
// chunk des highlights, le résolveur rend 0 pour tous les xuids — le motif s'y trouve dans un
// contexte qui n'est pas celui d'un enregistrement de joueur. Mesuré sur `000d5950` : les
// chunks 1 à 26 donnent TOUS la même table, le 0 et le 27 rendent 0 pour les huit. On ne lit
// donc que les chunks de réplication, et on EXIGE qu'ils concordent.
//
// POURQUOI EXIGER LA CONCORDANCE PLUTÔT QUE PRENDRE UNE MAJORITÉ. Une majorité serait un vote,
// et c'est exactement ce que ce chantier a retiré. Ici les 26 lectures sont 26 occurrences du
// MÊME fait écrit dans le film : si deux d'entre elles divergeaient, ce ne serait pas un
// désaccord à arbitrer, ce serait le signe que la lecture est fausse — et il faut alors ne
// rien publier plutôt que trancher.
//
// DESCENDU DE `film/replay` AU LOT J4.2 (2026-09-26, PLAN_SUITE_AUDIT_DECODEUR_FILM, DU-3 = S1) :
// c est une LECTURE du film. Deplacement pur ; `PlayerIndexTable` vit en `types`. Ce que la
// publication fait de la table — le roster qu on lui donne, l injectivite exigee — reste en
// `replay` (`rosterOf`, `injectiveOrEmpty`).

// ScanFilmPlayerIndices lit l'index de joueur de chaque xuid du roster dans les chunks de
// réplication du film.
//
// HORS LIGNE (I/O disque sur tout le film) — jamais depuis un chemin de requête.
// ENVELOPPE D2, HORS PRODUCTION ; la cuisson appelle [ScanPlayerIndices].
func ScanFilmPlayerIndices(filmDir string, roster []uint64) (types.PlayerIndexTable, error) {
	film, err := source.LoadDir(filmDir, nil)
	if err != nil {
		return types.PlayerIndexTable{ByXUID: map[uint64]int{}}, err
	}
	return ScanPlayerIndices(film, roster)
}

// ScanPlayerIndices lit l'index de joueur de chaque xuid du roster dans un film DEJA CHARGE.
func ScanPlayerIndices(film *source.Film, roster []uint64) (types.PlayerIndexTable, error) {
	out, _, err := scanPlayerIndices(film, roster)
	return out, err
}

// scanPlayerIndices est [ScanPlayerIndices], plus le nombre de chunks de replication SAUTES faute
// d etre lisibles — le compte de `repli_chunk_de_replication_saute`, que l etage du pont d identite
// verse au rapport de son contexte (lot J8.7).
func scanPlayerIndices(film *source.Film, roster []uint64) (types.PlayerIndexTable, int, error) {
	out := types.PlayerIndexTable{ByXUID: map[uint64]int{}}
	if len(roster) == 0 {
		return out, 0, fmt.Errorf("roster vide : rien à résoudre")
	}
	nums := FilmChunkNumbers(film)
	if len(nums) == 0 {
		return out, 0, ErrNoReadableFilmChunk
	}
	// Chunks de RÉPLICATION seulement : le 0 est le registre, le dernier porte les highlights.
	// Les deux rendent une table nulle, et l'inclure écraserait la bonne.
	seen := map[uint64]map[int]int{}
	sautes := 0
	for _, c := range nums[:len(nums)-1] {
		raw, _, ok := FilmChunkAt(film, c)
		if !ok {
			sautes++
			continue
		}
		got := weaponv3.ResolveXuidToPI(roster, raw)
		if len(got) == 0 {
			sautes++ // resolution vide : sautee comme un chunk illisible, et comptee avec lui
			continue
		}
		out.Readings++
		for x, pi := range got {
			if seen[x] == nil {
				seen[x] = map[int]int{}
			}
			seen[x][pi]++
		}
	}
	if out.Readings == 0 {
		return out, sautes, fmt.Errorf("aucun chunk de réplication n'a livré d'index de joueur")
	}
	for x, byIdx := range seen {
		if len(byIdx) > 1 {
			out.Disagreements++
			continue // une identité lue de deux façons n'est pas publiable
		}
		for pi := range byIdx {
			out.ByXUID[x] = pi
		}
	}
	return out, sautes, nil
}
