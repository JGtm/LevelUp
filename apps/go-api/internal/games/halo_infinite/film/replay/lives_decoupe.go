package replay

// lives_decoupe.go — OU FINIT UNE VIE : CE QUE LE FILM ECRIT, ET PLUS UN SEUIL (lot 1.9.13).
//
// # LE DEFAUT, MESURE
//
// `buildLifeSpans` ferme une vie des qu'un trou de positions depasse `lifeGapUS` (5 s). C'est une
// HEURISTIQUE au sens de D13 : la grammaire dit qu'une vie commence a une APPARITION et finit a
// une MORT ECRITE, a une FIN DE MANCHE ou a la FIN DU FILM — un trou de replication n'est pas une
// mort, c'est une LACUNE.
//
// Mesure du 2026-09-15 (`decoupe_des_vies_mesure_test.go`, les 8 builds) sur les 212 coupures que
// ce seuil decide : 4 sont appariees a une mort ecrite, 0 a un record de creation de bipede, 0 a
// une frontiere de manche, 0 a la fin du film — et 208 (98,1 %) ne sont justifiees PAR RIEN. La
// consequence etait deja mesuree au lot 1.6.5 (D1 (1.6)) : 14 des 20 vies d'un seul echantillon
// des 8 builds sont ORPHELINES, et toutes se ferment sur un de ces trous.
//
// # LA CONVERSION
//
// La decoupe par le seuil devient un ECHAFAUDAGE : elle borne les sejours de replication, et rien
// de plus. [decouperAuxFaitsEcrits] la relit et REFERME toute coupure que le film ne justifie
// pas ; les deux sejours n'en font plus qu'un, separes par une lacune. Une coupure ne survit que
// si le film l'ECRIT :
//
//	mort ecrite     le fil des morts apparie la fin du sejour (`CauseVieMort`, pose par
//	                `marquerCauseDeMort` — la SEULE cause qui dise « ce joueur est mort »)
//	apparition      un record de CREATION de bipede pour ce slot tombe dans le trou : le film
//	                ecrit qu'un corps NEUF commence ici. C'est aussi le signal d'un slot RECYCLE
//	                (le moteur cree l'entite PUIS replique ses positions, cf.
//	                identity_registry_creation.go) : sans lui, deux occupants d'un meme slot
//	                fusionneraient en une seule vie
//	fin de manche   une frontiere de manche tombe dans le trou. Une fin de manche fait
//	                REAPPARAITRE tout le monde SANS mort ecrite : elle ferme donc une vie la ou
//	                le fil des morts se tait
//
// # LE REPLI QUI RESTE, ET LUI SEUL
//
// `repli_vie_coupee_au_trou_de_replication` (D14) : quand le film n'ecrit AUCUNE mort du joueur
// de tout le film, aucune des trois lectures ci-dessus ne peut borner ses vies — le seuil reste
// alors la seule voie, et il est COMPTE. Il passe `devant_la_lecture` -> `apres_lecture` : la
// lecture tourne d'abord (les trois causes ci-dessus), le repli n'entre qu'apres son silence.
// Mesure du 2026-09-15 : 0 declenchement sur les 8 builds — toute vie coupee y appartient a un
// joueur qui meurt au moins une fois.

import (
	"sort"

	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/fallback"
	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/objectives"
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// buildLifeSpans découpe les trajectoires en SÉJOURS DE RÉPLICATION : un slot qui disparaît plus
// de lifeGapUS puis revient ouvre un nouveau séjour.
//
// CE N'EST PLUS LA DÉCOUPE DES VIES DEPUIS LE LOT 1.9.13 (2026-09-15), C'EST SON ÉCHAFAUDAGE.
// Un trou de réplication n'est pas une mort (D13) : la découpe publiée vient de ce que le film
// ÉCRIT, et `decouperAuxFaitsEcrits` referme ici toute coupure qu'aucun fait ne justifie. Ce
// découpage-ci garde deux rôles, et deux seulement : il est la grille sur laquelle
// [bestDeathOffset] mesure le calage du fil des morts (elle n'a pas d'autre grille avant que le
// calage existe), et il est le repli compté quand le registre ne rend aucune vie.
//
// Mesure du 2026-09-15 sur les 8 builds : 212 coupures décidées ici, dont 208 qu'AUCUN fait du
// film ne justifiait.
func buildLifeSpans(tracks map[uint32]slotTrack) []lifeSpan {
	slots := make([]uint32, 0, len(tracks))
	for s := range tracks {
		slots = append(slots, s)
	}
	sort.Slice(slots, func(i, j int) bool { return slots[i] < slots[j] })
	var out []lifeSpan
	for _, s := range slots {
		pts := tracks[s].pts
		if len(pts) == 0 {
			continue
		}
		start, last := int64(pts[0].TimestampUS), int64(pts[0].TimestampUS)
		for _, p := range pts[1:] {
			t := int64(p.TimestampUS)
			if t-last > lifeGapUS {
				// TROU AU-DELÀ DU SEUIL : la vie se ferme ici. C'est la cause STRUCTURELLE,
				// que le nommage écrasera s'il sait mieux (une mort, une fermeture).
				out = append(out, lifeSpan{slot: s, from: start, to: last, cause: CauseVieCoupure})
				start = t
			}
			last = t
		}
		// LA DERNIÈRE VIE DU SLOT N'EST FERMÉE PAR AUCUN TROU : ses points sont simplement
		// épuisés. C'est la fin de ce que le film montre de ce slot — pas une coupure.
		out = append(out, lifeSpan{slot: s, from: start, to: last, cause: CauseVieFinFilm})
	}
	return out
}

// faitsQuiBornentUneVie porte ce que le film ECRIT et qui ferme legitimement une vie. Une
// structure plutot que des parametres de plus : le depot en borne cinq.
type faitsQuiBornentUneVie struct {
	// creations : les records de creation de bipede (`grammar.ScanBipedCreations`).
	creations []grammar.BipedCreation
	// manches : les frontieres de manche, sur l'horloge du FILM, en microsecondes.
	manches []int64
	// mortsParJoueur : les instants des morts ECRITES du fil, par joueur, sur l'horloge du FILM
	// (microsecondes). Un joueur absent de cette table n'a AUCUNE mort du film, donc aucune
	// lecture ne borne ses vies.
	mortsParJoueur map[uint64][]int64
	// fb : le compteur de replis de la cuisson (nil-safe).
	fb *fallback.Compteur
}

// decouperAuxFaitsEcrits relit l'echafaudage de [buildLifeSpans] et ne garde que les coupures
// que le film ECRIT. Les sejours d'une coupure refermee fusionnent en UNE vie.
//
// L'ORDRE DE SORTIE EST CELUI DE L'ENTREE — par slot croissant puis chronologique
// (cf. buildLifeSpans) — parce que `nommerViesParCreations` s'appuie dessus pour dire QUELLE vie
// un record ouvre, et qu'un artefact dont les vies changent d'ordre ne se compare pas.
//
// ELLE NE NOMME RIEN ET NE POSE AUCUNE CAUSE : la vie fusionnee garde la cause du DERNIER sejour
// (c'est lui qui la termine) et la premiere identite non vide des sejours fusionnes — ils sont le
// meme corps, la lecture d'identite repasse ensuite sur le resultat.
func decouperAuxFaitsEcrits(lives []lifeSpan, in faitsQuiBornentUneVie) []lifeSpan {
	if len(lives) == 0 {
		return lives
	}
	creaParSlot := creationsParSlot(in.creations)
	out := make([]lifeSpan, 0, len(lives))
	for i := range lives {
		l := lives[i]
		n := len(out)
		if n > 0 && out[n-1].slot == l.slot {
			cause := causeDeLaCoupure(out[n-1], l, creaParSlot[l.slot], in)
			if cause == "" {
				// LA COUPURE N'EST PAS ECRITE : le trou devient une LACUNE de la MEME vie. Le
				// sejour qui suit prolonge celui qui precede — meme corps, meme vie.
				out[n-1].to, out[n-1].cause = l.to, l.cause
				if out[n-1].xuid == 0 && out[n-1].bid == "" {
					out[n-1].xuid, out[n-1].bid, out[n-1].nomPar = l.xuid, l.bid, l.nomPar
				}
				continue
			}
			// LA COUPURE EST ECRITE : la vie qui s'y ferme porte CE QUI L'A FERMEE. Une mort du
			// joueur dans la fenetre du trou EST une mort ecrite, que l'appariement 1:1 du pont
			// l'ait retenue ou non — lui est glouton sur le TEMPS SEUL, et il peut avoir donne
			// cette fin de vie a la mort d'un autre joueur.
			out[n-1].cause = cause
		}
		out = append(out, l)
	}
	return out
}

// causeDeLaCoupure rend CE QUI FERME la vie en cours (`prec`, dont `to` est la fin du dernier
// sejour ferme) devant le sejour `suiv` — chaine vide quand RIEN ne la ferme, et le trou est
// alors une lacune.
//
// L'ORDRE EST FIXE (D14 b) : les trois lectures d'abord, le repli ensuite — et ce repli-ci ne
// s'ouvre que sur un SILENCE mesure (aucune mort ecrite de ce joueur dans tout le film), jamais
// sur un desaccord avec une lecture.
func causeDeLaCoupure(prec, suiv lifeSpan, creations []uint64, in faitsQuiBornentUneVie) string {
	if mortEcriteDansLeTrou(prec, suiv, in) {
		return CauseVieMort
	}
	for _, t := range creations {
		// APPARITION ECRITE : un corps NEUF commence dans le trou. Le record precede la premiere
		// position repliquee du sejour qu'il ouvre (mesure du lot E2), il tombe donc dans le trou.
		if int64(t) > prec.to && int64(t) <= suiv.from {
			return CauseVieCoupure
		}
	}
	for _, m := range in.manches {
		// FIN DE MANCHE : tout le monde reapparait SANS qu'aucune mort soit ecrite.
		if m >= prec.to && m <= suiv.from {
			return CauseVieCoupure
		}
	}
	if len(in.mortsParJoueur[prec.xuid]) == 0 {
		// REPLI NOMME ET COMPTE (D14) : le film n'ecrit AUCUNE mort de ce joueur — rien ne borne
		// ses vies, et le seuil de trou reste la seule voie. Une vie SANS identite lue tombe ici
		// aussi, et pour la meme raison : sans joueur, pas de mort a lui rapporter.
		in.fb.Declenche(fallback.NomVieCoupeeAuTrouDeReplication)
		return CauseVieCoupure
	}
	return ""
}

// mortEcriteDansLeTrou dit si le fil des morts porte une mort DU JOUEUR DE CETTE VIE dans la
// fenetre du trou.
//
// L'IDENTITE FAIT PARTIE DU CRITERE, et ce n'est pas un detail : l'appariement du pont
// (`apparierMortsEtVies`) est glouton SUR LE TEMPS SEUL, donc il peut poser sur une fin de sejour
// la mort d'un AUTRE joueur — c'est la discordance que `verifierParLesMorts` compte deja, « le
// film fait foi ». Mesure du 2026-09-15 : accepter cette cause sans regarder le xuid laissait
// 5 coupures sans aucune mort du joueur concerne. Le lien vient du registre d'identite (lot 1.6).
//
// FAUTE D'IDENTITE LUE, LA CAUSE POSEE PAR LE PONT EST LA SEULE LECTURE DISPONIBLE : on la prend
// telle quelle plutot que de fermer une vie qu'une mort borne peut-etre.
func mortEcriteDansLeTrou(prec, suiv lifeSpan, in faitsQuiBornentUneVie) bool {
	if prec.xuid == 0 {
		return prec.cause == CauseVieMort
	}
	// La fenetre remonte de la tolerance d'appariement : une mort tombe a l'instant de la
	// DERNIERE position repliquee, pas apres elle.
	de, a := prec.to-deathMatchWindowMS*1000, suiv.from
	for _, t := range in.mortsParJoueur[prec.xuid] {
		if t >= de && t <= a {
			return true
		}
	}
	return false
}

// creationsParSlot groupe les instants des records de creation par slot, tries.
func creationsParSlot(creations []grammar.BipedCreation) map[uint32][]uint64 {
	out := map[uint32][]uint64{}
	for _, c := range creations {
		out[c.Slot] = append(out[c.Slot], c.TimestampUS)
	}
	for s := range out {
		sort.Slice(out[s], func(i, j int) bool { return out[s][i] < out[s][j] })
	}
	return out
}

// mortsParJoueur groupe les instants des morts ECRITES du fil par joueur, sur l'horloge du FILM
// (microsecondes) — `horlogeFilm = horlogeFil + deathOffsetMS`, la convention de match_clock.go.
func mortsParJoueur(deaths []Death, offsetMS int64) map[uint64][]int64 {
	out := make(map[uint64][]int64, len(deaths))
	for _, d := range deaths {
		out[d.XUID] = append(out[d.XUID], (d.TimeMS+offsetMS)*1000)
	}
	return out
}

// manchesEnFilmUS pose les frontieres de manche sur l'horloge du FILM, en microsecondes.
//
// LES BORNES VIENNENT D'`objectives`, QUI LES MESURE DEJA (`ResolveRoundBounds`) : les
// re-mesurer ici ferait deux mesures de la meme grandeur, qui divergeraient.
func manchesEnFilmUS(records []types.StatRecord, offsetMS int64) []int64 {
	if len(records) == 0 {
		return nil
	}
	var out []int64
	for _, ms := range objectives.ResolveRoundBounds(records).Starts() {
		out = append(out, (int64(ms)+offsetMS)*1000)
	}
	return out
}
