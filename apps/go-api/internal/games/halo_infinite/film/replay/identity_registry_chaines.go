package replay

// identity_registry_chaines.go — LA CHAINE DE CONTRADICTIONS : LES CHUNKS FONT FOI POUR SES MEMBRES.
//
// # LE FAIT
//
// Une CHAINE est une contradiction entre la table du film et la lecture des chunks de replication
// dont l'index des chunks est le SIEGE DE TABLE d'un autre joueur, lui-meme contredit : la table
// assied `V` a `j`, les chunks lisent `W` a `j` et `V` a `k != j`. Deux lectures du film (celle de
// `W` et celle de `V`) dementent alors le siege de table de `V`. Sur ces films, les corps que la
// table attribue aux membres de la chaine portent l'equipe du joueur que les CHUNKS y lisent, pas
// celle du titulaire de table ; et les corps de l'index des chunks d'un membre restent sans nom.
// Mesures et temoins : journal d'E8.4 du plan `.ai/archive/V7.5/PLAN_BACKLOG_2026-10-16.md`.
//
// # LA REGLE
//
// Pour les membres d'une chaine, la table effective prend leur index des chunks. Leur ancien siege
// de table, quand plus personne ne le tient, reste un index SUPPLEMENTAIRE de la lecture des corps
// (identity_registry_index_des_chunks.go), sous la meme garde du chevauchement : un corps de cet
// index qui chevauche un corps du meme joueur nomme par son index des chunks le dement, et le siege
// est retire. La chaine ne se resout que si l'index des chunks de chaque membre est libre de tout
// joueur que les chunks ne contredisent pas ; sinon rien ne bouge et la chaine se compte.
//
// C'est la meme provenance que le complement par les chunks de la composition (voie
// `PlayerIndexTable`, repli nomme et compte) : la table du film se tait la ou deux lectures la
// dementent.

import "levelup/go-api/internal/games/halo_infinite/film/identity"

// membresDesChaines rend les joueurs contredits qui forment une chaine : ceux dont l'index des
// chunks est le siege d'un autre joueur contredit, ces titulaires, et le titulaire d'un siege
// contredit que les chunks donnent a un joueur que la table n'assied pas (le complement par les
// chunks y poserait sinon une collision qui retirerait les deux liens).
func membresDesChaines(cs []contradictionDeSiege, in IdentityInput) map[uint64]bool {
	assisA := make(map[int]uint64, len(cs))
	for _, c := range cs {
		assisA[c.siege] = c.xuid
	}
	membres := map[uint64]bool{}
	for _, c := range cs {
		if v, tenu := assisA[c.controle]; tenu && v != c.xuid {
			membres[c.xuid], membres[v] = true, true
		}
	}
	assis := map[uint64]bool{}
	for _, s := range in.FilmTable.Seats {
		assis[s.XUID] = s.XUID != 0
	}
	for x, j := range in.PlayerIndices.ByXUID {
		if v, tenu := assisA[j]; tenu && !assis[x] {
			membres[v] = true
		}
	}
	return membres
}

// resoudreLesChaines porte les membres d'une chaine a leur index des chunks dans la table effective,
// et rend les membres effectivement deplaces (vide quand la chaine ne se resout pas). Elle tourne
// AVANT le retrait des collisions : c'est elle qui leve celles que la chaine explique.
//
// TOUT OU RIEN : un index des chunks tenu par un joueur qui n'est pas membre laisserait deux joueurs
// au meme index ; la chaine reste alors telle que la table l'ecrit, et `chaines` le dit.
func resoudreLesChaines(out *filmTableLinks, cs []contradictionDeSiege, in IdentityInput) map[uint64]bool {
	membres := membresDesChaines(cs, in)
	if len(membres) == 0 {
		return nil
	}
	for _, c := range cs {
		if !membres[c.xuid] {
			continue
		}
		for x, i := range out.table.ByXUID {
			if i == c.controle && !membres[x] {
				out.chaines++
				return nil
			}
		}
	}
	for _, c := range cs {
		if !membres[c.xuid] {
			continue
		}
		out.table.ByXUID[c.xuid] = c.controle
		out.voie[c.xuid] = identity.MethodPlayerIndexTable
		out.couverture.Direct--
		out.couverture.Fallback++
	}
	out.membresDeChaine = len(membres)
	return membres
}
