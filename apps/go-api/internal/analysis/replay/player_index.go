package replay

import (
	"fmt"
	"sort"

	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/analysis/filmsource"
	"levelup/go-api/internal/analysis/weaponv3"
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

// PlayerIndexTable est le lien identité -> index de joueur du film, lu et contrôlé.
type PlayerIndexTable struct {
	// ByXUID est la table elle-même.
	ByXUID map[uint64]int
	// Readings est le nombre de chunks de réplication qui ont produit la table.
	Readings int
	// Disagreements est le nombre de xuids pour lesquels deux chunks ont lu des index
	// DIFFÉRENTS. Non nul, la table n'est pas publiée : ce n'est pas un désaccord à
	// arbitrer, c'est le symptôme d'une lecture fausse.
	Disagreements int
}

// ScanFilmPlayerIndices lit l'index de joueur de chaque xuid du roster dans les chunks de
// réplication du film.
//
// HORS LIGNE (I/O disque sur tout le film) — jamais depuis un chemin de requête.
// ENVELOPPE D2, HORS PRODUCTION ; la cuisson appelle [ScanPlayerIndices].
func ScanFilmPlayerIndices(filmDir string, roster []uint64) (PlayerIndexTable, error) {
	film, err := filmsource.LoadDir(filmDir, nil)
	if err != nil {
		return PlayerIndexTable{ByXUID: map[uint64]int{}}, err
	}
	return ScanPlayerIndices(film, roster)
}

// ScanPlayerIndices lit l'index de joueur de chaque xuid du roster dans un film DEJA CHARGE.
func ScanPlayerIndices(film *filmsource.Film, roster []uint64) (PlayerIndexTable, error) {
	out := PlayerIndexTable{ByXUID: map[uint64]int{}}
	if len(roster) == 0 {
		return out, fmt.Errorf("roster vide : rien à résoudre")
	}
	nums := filmdec.FilmChunkNumbers(film)
	if len(nums) == 0 {
		return out, filmdec.ErrNoReadableFilmChunk
	}
	// Chunks de RÉPLICATION seulement : le 0 est le registre, le dernier porte les highlights.
	// Les deux rendent une table nulle, et l'inclure écraserait la bonne.
	seen := map[uint64]map[int]int{}
	for _, c := range nums[:len(nums)-1] {
		raw, _, ok := filmdec.FilmChunkAt(film, c)
		if !ok {
			continue
		}
		got := weaponv3.ResolveXuidToPI(roster, raw)
		if len(got) == 0 {
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
		return out, fmt.Errorf("aucun chunk de réplication n'a livré d'index de joueur")
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
	return out, nil
}

// rosterFromDeaths rend les xuids distincts du fil des morts, en ordre stable. C'est le seul
// roster dont le rejeu dispose sans base de données — et il est déjà, lui aussi, une lecture.
func rosterFromDeaths(deaths []Death) []uint64 {
	seen := map[uint64]bool{}
	out := make([]uint64, 0, 8)
	for _, d := range deaths {
		if !seen[d.XUID] {
			seen[d.XUID] = true
			out = append(out, d.XUID)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// injectiveOrEmpty rend la table si elle est injective (deux joueurs ne peuvent pas partager
// un index), et une table VIDE sinon.
//
// L'injectivité n'est pas une préférence esthétique : un index partagé placerait les tirs de
// deux joueurs sur la même trace, sans que rien ne le signale.
func injectiveOrEmpty(t PlayerIndexTable) (PlayerIndexTable, int) {
	byIdx := map[int]int{}
	for _, pi := range t.ByXUID {
		byIdx[pi]++
	}
	collisions := 0
	for _, n := range byIdx {
		if n > 1 {
			collisions++
		}
	}
	if collisions > 0 {
		return PlayerIndexTable{ByXUID: map[uint64]int{}, Readings: t.Readings,
			Disagreements: t.Disagreements}, collisions
	}
	return t, 0
}
