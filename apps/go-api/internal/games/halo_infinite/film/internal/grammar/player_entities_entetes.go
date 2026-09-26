package grammar

// player_entities_entetes.go — CE QUI PROUVE UNE ABSENCE : L'EN-TETE EXACT, CHERCHE PARTOUT (lot
// D-fix de la campagne « retours rejeu », reprise apres la revue adverse du 2026-09-24, constat
// DFIX-R2).
//
// # POURQUOI LA MARCHE NE SUFFIT PAS
//
// Les records d'une image-cle viennent de la marche d'ancres, et la marche peut perdre un vrai
// record par PLUSIEURS chemins : l'election de repli (le candidat ecarte, que la premiere version
// du lot D-fix notait), mais aussi le SAUT DE LARGEUR (il ne regarde pas les bits qu'il enjambe),
// le FAUX VOISIN (le premier `slot+1` de generation 1 de la fenetre, meme pris dans le corps du
// record courant : les candidats qu'il enjambe ne se notaient pas) et l'election dont les vrais
// records sont AU-DELA de la fenetre (jamais vus). Enumerer les chemins de la marche pour savoir
// ce qu'elle a pu perdre, c'est suivre ses heuristiques ; le prochain chemin oublie redonnerait un
// depart fantome.
//
// # LA REGLE, ET ELLE VIENT DE LA GRAMMAIRE DU RECORD
//
// Un record d'image-cle commence par un en-tete EXACT (cf. `keyframe_world.go`) : l'identifiant
// sur 32 bits (generation 1 a 3 en bits 30-31, slot en dessous, borne par la table) puis le mot de
// 32 bits `[field:26][ti:6]`, dont `field` est nul a la replication — c'est le filtre fort que la
// marche applique a TOUTE ancre. Si l'entite du slot `S` est dans l'image-cle, son record y est,
// donc la suite de 64 bits `[gen|S][ti]` y est, a UNE position de bit au moins.
//
// Donc : l'absence de `S` a une image-cle n'est PROUVEE que si aucune position de bit du payload
// ne porte cet en-tete. Un en-tete trouve mais non lu par la marche (record perdu, record illisible)
// laisse un DOUTE. Aucun seuil, aucun chemin de marche : une recherche EXHAUSTIVE du motif, qui ne
// peut pas manquer un record present. Le prix est un faux doute quand ces 64 bits apparaissent par
// hasard dans le corps d'un autre record : il ne fait jamais conclure un depart, il le DIFFERE, et
// il se compte (`coverage.seats.imagesClesDouteuses`, `bornesDifferees`).

// slotsDEntetesExacts rend les slots dont l'en-tete EXACT d'un record de JOUEUR GERE (`ti=9`)
// apparait a au moins une position de bit du payload d'image-cle `pay` (cf. l'en-tete du fichier).
func slotsDEntetesExacts(pay []byte) map[int]bool {
	out := map[int]bool{}
	total := len(pay) * 8
	for q := 0; q+64 <= total; q++ {
		id := kfReadBits(pay, q, 32)
		if id>>30 == 0 {
			continue
		}
		slot := int(id & 0x3FFFFFFF)
		if slot >= kfTableCap {
			continue
		}
		if kfReadBits(pay, q+32, 32) == managedPlayerTypeIndex {
			out[slot] = true
		}
	}
	return out
}
