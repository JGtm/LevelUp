package replay

// identity_registry_elimination.go — L'IDENTITE PAR ELIMINATION SUR LE ROSTER.
//
// # LE FAIT QU'ELLE FERME (lot R2 du plan v2, temoin `d9781168`)
//
// `d9781168` (Oddball, a manches) publiait 19 vies sans nom, TOUTES sur UN slot dont aucune vie
// n'etait nommee : un joueur qui ne meurt JAMAIS de tout le match. Le pont par morts ne peut
// rien pour lui — il nomme une vie par la mort qui la TERMINE, et il n'y en a aucune. Les
// fermetures non plus : elles raisonnent sur des reapparitions, qui n'existent pas davantage.
//
// # POURQUOI L'ELIMINATION N'EST PAS UNE DEVINETTE
//
// Elle ne suppose RIEN du contenu : ni ressemblance de compteurs, ni proximite geometrique, ni
// ordre d'iteration. Elle CONSTATE qu'il ne reste qu'une affectation possible — un seul xuid du
// roster ne nomme aucune vie, un seul slot de bipede n'a aucune vie nommee. C'est un
// appariement FORCE, du meme ordre que la fermeture B (« un autre corps est reapparu, donc
// celui-ci etait celui-la »), et il porte la meme provenance : `deduit`, jamais `direct`.
//
// # DEUX CANDIDATS = ON SE TAIT, ET ON COMPTE
//
// Des qu'il reste deux xuids libres ou deux slots muets, l'unicite disparait et avec elle la
// preuve. On ne choisit pas « le premier » : ce serait un choix par l'ordre, celui-la meme que
// `SlotAmbiguous` existe pour signaler. Le residu est publie « non resolu » et compte.
//
// # ELLE AJOUTE UNE PRESENCE, ELLE NE FABRIQUE PAS UNE MORT
//
// Les vies nommees ici gardent leur `cause` de decoupe (`film_end`, `cut`) : rien dans
// l'elimination ne dit COMMENT une vie s'est terminee. Confondre les deux axes est le P0 de la
// ronde 2 du 2026-09-07, qui avait coute toute une lecture d'isolement.

import "log/slog"

// NomParElimination : la vie a ete nommee par ELIMINATION sur le roster — une deduction, et
// surtout PAS une mort. Troisieme valeur de l'axe `nomPar` (cf. lives.go).
const NomParElimination = "elimination"

// resolveByRosterElimination nomme les vies du seul slot muet quand il ne reste qu'un seul xuid
// du roster sans aucune vie. Ne fait RIEN dans tout autre cas.
func (r *IdentityRegistry) resolveByRosterElimination(in IdentityInput) {
	if r.own.DeathsNamed == 0 || len(r.own.lives) == 0 {
		return
	}
	libres := rosterSansVie(r.own.lives, rosterCandidat(in))
	muets := slotsSansVieNommee(r.own.lives)
	if len(libres) != 1 || len(muets) != 1 {
		if len(libres) > 0 && len(muets) > 0 {
			slog.Info("rejeu : elimination sur le roster impossible — l'unicite manque",
				"match_id", in.MatchID, "xuidsLibres", len(libres), "slotsMuets", len(muets))
		}
		return
	}
	xuid, slot := libres[0], muets[0]
	for i := range r.own.lives {
		if r.own.lives[i].slot != slot || r.own.lives[i].xuid != 0 {
			continue
		}
		r.own.lives[i].xuid = xuid
		r.own.lives[i].nomPar = NomParElimination
		r.deducedLives[i] = true
		r.eliminated++
	}
	if r.eliminated == 0 {
		return
	}
	r.eliminatedSlot, r.eliminatedXUID = slot, xuid
	// LE PONT APLATI SUIT LES VIES, sans quoi les deux tables du meme registre diraient deux
	// choses du meme slot (c'est l'invariant qu'`ownersFromLives` impose deja aux lectures).
	if r.own.SlotXUID != nil {
		r.own.SlotXUID[slot] = xuid
	}
	if pi, connu := in.PlayerIndices.ByXUID[xuid]; connu && r.own.Owner != nil {
		if _, deja := r.own.Owner[slot]; !deja {
			r.own.Owner[slot] = pi
		}
	}
	slog.Info("rejeu : identite posee par elimination sur le roster",
		"match_id", in.MatchID, "slot", slot, "xuid", xuid, "vies", r.eliminated)
}

// rosterCandidat rend les xuids que l'elimination peut attribuer : ceux que le film NOMME
// (`PlayerIndexTable`, un lien direct) completes par ceux que la base declare.
//
// UN XUID SANS INDEX DE JOUEUR RESTE CANDIDAT : il est au roster du match, donc il a joue, et
// c'est precisement le cas d'un joueur qui ne meurt jamais — le fil des morts ne l'a pas fait
// entrer dans la table d'index.
func rosterCandidat(in IdentityInput) []uint64 {
	seen := map[uint64]bool{}
	out := make([]uint64, 0, len(in.PlayerIndices.ByXUID)+len(in.RosterXUIDs))
	ajouter := func(x uint64) {
		if x == 0 || seen[x] {
			return
		}
		seen[x] = true
		out = append(out, x)
	}
	for x := range in.PlayerIndices.ByXUID {
		ajouter(x)
	}
	for _, x := range in.RosterXUIDs {
		ajouter(x)
	}
	// ORDRE STABLE : `ByXUID` est une map, et l'unique candidat retenu ne doit pas dependre de
	// l'ordre d'iteration — meme si, par construction, il n'y en a qu'un quand on l'emploie.
	trierUint64(out)
	return out
}

// rosterSansVie rend les xuids du roster qu'AUCUNE vie ne nomme.
func rosterSansVie(lives []lifeSpan, roster []uint64) []uint64 {
	nommes := map[uint64]bool{}
	for _, l := range lives {
		if l.xuid != 0 {
			nommes[l.xuid] = true
		}
	}
	out := make([]uint64, 0, len(roster))
	for _, x := range roster {
		if !nommes[x] {
			out = append(out, x)
		}
	}
	return out
}

// slotsSansVieNommee rend les slots de bipede dont AUCUNE vie n'est nommee, en ordre croissant.
func slotsSansVieNommee(lives []lifeSpan) []uint32 {
	nomme := map[uint32]bool{}
	tous := map[uint32]bool{}
	for _, l := range lives {
		tous[l.slot] = true
		if l.xuid != 0 {
			nomme[l.slot] = true
		}
	}
	out := make([]uint32, 0, len(tous))
	for s := range tous {
		if !nomme[s] {
			out = append(out, s)
		}
	}
	trierUint32(out)
	return out
}

// trierUint64 / trierUint32 : tris par insertion, sur des tranches de taille de roster (<= 16).
// Une dependance a `sort` pour seize elements couterait plus en lecture qu'elle ne rapporte.
func trierUint64(v []uint64) {
	for i := 1; i < len(v); i++ {
		for j := i; j > 0 && v[j] < v[j-1]; j-- {
			v[j], v[j-1] = v[j-1], v[j]
		}
	}
}

func trierUint32(v []uint32) {
	for i := 1; i < len(v); i++ {
		for j := i; j > 0 && v[j] < v[j-1]; j-- {
			v[j], v[j-1] = v[j-1], v[j]
		}
	}
}
