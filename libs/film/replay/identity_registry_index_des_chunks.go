package replay

// identity_registry_index_des_chunks.go — L'INDEX QUE LES CHUNKS LISENT POUR UN JOUEUR QUE LA TABLE
// DU FILM PLACE AILLEURS.
//
// # LE FAIT
//
// Le record de creation d'un corps porte un index de participant que la table effective ne nomme pas
// (`index_hors_table`), alors que la lecture des chunks de replication (`grammar.ScanPlayerIndices`,
// concordante sur tous les chunks) donne CET index a un joueur que la table du film assied a un
// autre index. Les corps de ce joueur portent alors l'index des chunks — seul, ou en alternance avec
// celui de la table, sans jamais se chevaucher. Mesures et temoins : journal d'E8.4 du plan
// `.ai/archive/V7.5/PLAN_BACKLOG_2026-10-16.md`.
//
// # LA REGLE
//
// La table du film reste la source premiere de la table effective (identity_registry_film_table.go)
// et rien de ce qu'elle assied n'est retire. L'index `j` que les chunks lisent pour un joueur `W`
// que la table assied a `i != j` nomme EN PLUS les corps qui le portent, a trois conditions :
//
//  1. aucun joueur de la table effective ne tient `j` : un joueur que les chunks contredisent aussi
//     y forme une CHAINE (identity_registry_chaines.go) ; un joueur qu'ils ne contredisent pas
//     l'interdit, deux lectures se disputent l'index, on se tait et on compte ;
//  2. aucun bot declare par BOT_METADATA ne tient `j` (l'ordre des voies de [creationReport.poser]
//     le garantit : le bot est lu avant le xuid) ;
//  3. aucun corps nomme `W` par cet index ne CHEVAUCHE dans le temps un corps nomme `W` autrement :
//     un joueur n'occupe qu'un corps a la fois. Un chevauchement dit que l'une des deux lectures
//     est fausse ; l'index des chunks est alors retire pour ce joueur, et le refus se compte.
//
// C'est une LECTURE du film (l'index ecrit dans le corps, le couple index-xuid ecrit dans les
// chunks), employee la ou la table du film se tait sur cet index : le meme statut que le complement
// par les chunks de la composition, repli nomme et compte (`repli_index_des_chunks_hors_table`).

import (
	"maps"
	"slices"

	"github.com/JGtm/LevelUp/libs/film/types"
)

// contradictionDeSiege : un joueur que la table du film assied a `siege` et que les chunks lisent
// a `controle`.
type contradictionDeSiege struct {
	xuid            uint64
	siege, controle int
}

// contradictionsAssises rend les contradictions des sieges que la table du film pose, dans l ordre des
// sieges du film. Appelee AVANT le retrait des collisions : la chaine en leve certaines, et le
// complement (indexDesChunksHorsTable) saute les sieges que le retrait a ensuite supprimes.
//
// `in` est l'entree AVANT composition : `in.PlayerIndices` y est la lecture brute des chunks.
func contradictionsAssises(table types.PlayerIndexTable, in IdentityInput) []contradictionDeSiege {
	if !in.FilmTable.Lue() {
		return nil
	}
	var out []contradictionDeSiege
	for _, s := range in.FilmTable.Seats {
		if s.XUID == 0 {
			continue
		}
		j, lu := in.PlayerIndices.ByXUID[s.XUID]
		if !lu || j == s.FilmIndex {
			continue
		}
		if siege, assis := table.ByXUID[s.XUID]; !assis || siege != s.FilmIndex {
			continue
		}
		out = append(out, contradictionDeSiege{xuid: s.XUID, siege: s.FilmIndex, controle: j})
	}
	return out
}

// indexDesChunksHorsTable rend les index SUPPLEMENTAIRES de la lecture des corps (index -> xuid) :
// pour un joueur contredit hors chaine, l'index des chunks quand aucun joueur de la table effective
// ne le tient ; pour un membre de chaine (identity_registry_chaines.go), son ancien siege de table
// quand plus personne ne le tient. Rend aussi le nombre de contradictions hors chaine refusees
// parce que leur index des chunks est tenu par un joueur que les chunks ne contredisent pas.
func indexDesChunksHorsTable(table types.PlayerIndexTable, cs []contradictionDeSiege,
	membres map[uint64]bool) (map[int]uint64, int) {
	tenus := indexToXUIDOf(table.ByXUID)
	out := map[int]uint64{}
	refusees := 0
	for _, c := range cs {
		if _, assis := table.ByXUID[c.xuid]; !assis {
			// RETIRE PAR UNE COLLISION D'INDEX : plus de lien direct, rien a completer.
			continue
		}
		index := c.controle
		if membres[c.xuid] {
			index = c.siege
		}
		if t, tenu := tenus[index]; tenu && t != c.xuid {
			if !membres[c.xuid] {
				refusees++
			}
			continue
		}
		out[index] = c.xuid
	}
	return out, refusees
}

// nommerViesParCreations pose le lien DIRECT sur les vies, AVANT toute autre voie, avec l'index des
// chunks hors table (cf. l'en-tete de ce fichier) sous la garde du chevauchement.
//
// Elle n'ecrase jamais rien : elle est la PREMIERE etape de [BuildIdentityRegistry], les vies y
// arrivent anonymes. Elle rend le rapport, jamais une decision — le registre en tire la
// couverture et les alarmes. Un index des chunks refuse fait rejouer la lecture sans lui, a partir
// des vies telles qu'elles sont arrivees : la boucle termine, l'ensemble des index ne fait que
// decroitre.
func nommerViesParCreations(lives []lifeSpan, in IdentityInput) creationReport {
	chunks := maps.Clone(in.indexDesChunks)
	if len(chunks) == 0 {
		return nommerAvecLesTables(lives, in, nil)
	}
	arrivees := slices.Clone(lives)
	for {
		rep := nommerAvecLesTables(lives, in, chunks)
		enConflit := indexDesChunksEnConflit(lives, rep.parIndexDesChunks)
		if len(enConflit) == 0 {
			rep.IndexDesChunksRefuses = len(in.indexDesChunks) - len(chunks)
			return rep
		}
		for _, j := range enConflit {
			delete(chunks, j)
		}
		copy(lives, arrivees)
	}
}

// indexDesChunksEnConflit rend les index des chunks dont un corps nomme `W` chevauche dans le temps
// un corps nomme `W` par une autre voie, en ordre croissant.
func indexDesChunksEnConflit(lives []lifeSpan, parChunks map[int]int) []int {
	conflit := map[int]bool{}
	for i, j := range parChunks {
		if conflit[j] {
			continue
		}
		for k := range lives {
			if k == i || lives[k].xuid != lives[i].xuid {
				continue
			}
			if jk, viaChunks := parChunks[k]; viaChunks && jk == j {
				continue
			}
			if seChevauchent(lives[i], lives[k]) {
				conflit[j] = true
				break
			}
		}
	}
	out := slices.Collect(maps.Keys(conflit))
	slices.Sort(out)
	return out
}

// garderLIndexDesCorpsParChunks rend au pont slot -> index l'index que le corps PORTE quand c'est
// l'index des chunks qui l'a nomme : `ownersFromLives` deduit l'index du xuid par la table
// effective, qui donne le siege du joueur et non l'index de ce corps. Le pont sert l'equipe lue par
// index et l'occupant d'un slot : il doit dire l'index ecrit dans le record de creation. Seul le
// slot dont la PREMIERE vie nommee (celle que `ownersFromLives` retient) vient de cet index change.
func garderLIndexDesCorpsParChunks(owners map[uint32]int, lives []lifeSpan, xuidToIndex map[uint64]int,
	parChunks map[int]int) {
	if len(parChunks) == 0 {
		return
	}
	vus := map[uint32]bool{}
	for i, l := range lives {
		if l.xuid == 0 || vus[l.slot] {
			continue
		}
		if _, ok := xuidToIndex[l.xuid]; !ok {
			continue
		}
		vus[l.slot] = true
		if j, viaChunks := parChunks[i]; viaChunks && owners[l.slot] == xuidToIndex[l.xuid] {
			owners[l.slot] = j
		}
	}
}
