package grammar

// player_entities.go — LES OCCUPANTS DU MATCH, UN PAR ENTITE `ti=9` (campagne « retours rejeu »,
// lot M2.1, 2026-09-23).
//
// # CE QUE LE FILM ECRIT, ET QUE LA TABLE PAR INDEX PERDAIT
//
// Chaque occupant du match — humain ou bot, present au coup d'envoi ou arrive en cours — a SON
// entite `ti=9` (« managed-player ») : un slot de replication, l'INDEX DE JOUEUR dans l'etat par
// defaut, le DESIGNATEUR D'EQUIPE dans le composant i0 (cf. player_teams.go). Mesures qui fondent
// ce fichier :
//
//	lot 1.7 (2026-09-14)   35 arrivees sur 18 films : une entite n'est JAMAIS reutilisee, et son
//	                       designateur est STABLE (88/88 entites) ;
//	sonde P4 (2026-09-23)  `b1ad85eb` : trois entites d'index 8 (designateurs 0, 0, 1 — trois
//	                       bots de DEUX equipes qui se relaient sur le meme index), l'entite de
//	                       l'index 1 absente des l'image-cle qui suit le depart de son occupant,
//	                       4 + 4 occupants a chacune des 25 images-cles pleines.
//
// `ScanPlayerTeams` accumulait le designateur PAR INDEX : un index repris par des occupants de
// deux equipes devenait « divergent » et n'etait pas publie — les trois bots de `b1ad85eb`
// perdaient leur equipe, et la PRESENCE de chaque occupant (premiere et derniere image-cle de son
// entite) n'etait publiee nulle part. Ce fichier rend l'ENTITE ; la table par index reste, comme
// CONTROLE (ses divergences se comptent).
//
// # CE QUE L'ENTITE DIT, ET CE QU'ELLE NE DIT PAS
//
// Elle dit la presence AU PAS DES IMAGES-CLES (une toutes les ~20 s) : l'occupant est la a chacune
// des images-cles qui portent son record. Un TROU entre la premiere et la derniere n'est pas un
// depart (sonde P4 : MONEY absent de l'image-cle f613 par perte de marche, present avant et apres
// sur le meme slot) — il se COMPTE ([PlayerEntityScan.Holes]) et ne coupe rien. L'arrivee et le
// depart ne se datent qu'a l'image-cle pres : c'est la publication qui les affine par les vies et
// par BOT_METADATA, jamais ce fichier.
//
// # UNE ABSENCE NE SE CONCLUT QUE SI LA MARCHE LA PROUVE (lot D-fix, 2026-09-24)
//
// Les records d'une image-cle viennent de la marche d'ancres, et son REPLI (recalage, election)
// ecarte des candidats : un vrai record ecarte est PERDU, et son occupant manque a l'image-cle
// sans l'avoir quittee. Mesure : l'image-cle d'avant-match de `bcb6d393` et de `fb1a1a72`, que la
// marche glissante du lot M3.1 atteint desormais, perdait le joueur gere de l'index 0 (slot 1297)
// sur une fausse ancre elue — le lot M2 le lisait ARRIVE plus tard (`000d5950` : absent 16,3 s au
// depart). La marche ne perd plus ce record (keyframe_world_preuve.go) ; LE PRINCIPE, lui, vaut
// pour tout film : une absence que la marche ne prouve pas ne conclut NI une arrivee tardive NI un
// depart. Elle se lit dans [PlayerEntityScan.Doutes], image-cle par image-cle, et les bornes de
// presence ne se posent que sur une absence PROUVEE ([PlayerEntityScan.AbsenceProuveeAvant] /
// [PlayerEntityScan.AbsenceProuveeApres]) — sinon l'occupant reste present jusqu'au bord du film.
//
// HORS LIGNE, comme player_teams.go : aucune ecriture, aucun schema.

import "sort"

// PlayerEntity est UNE entite `ti=9` : un occupant du match, lu dans la trame d'etat.
type PlayerEntity struct {
	// Slot est le slot de replication de l'entite — son identite dans le film. Une entite n'est
	// jamais reutilisee : deux occupants successifs d'un meme index ont deux slots.
	Slot int
	// Index est l'INDEX DE JOUEUR que l'etat par defaut porte (cf. [readManagedPlayerDefaultState]).
	// Majoritaire sur les lectures ; [PlayerEntity.Unstable] dit s'il a change.
	Index int
	// Team est le DESIGNATEUR D'EQUIPE (valeur du jeu : [TeamNone] ou 0..8), majoritaire.
	Team int
	// FirstKF / LastKF sont les RANGS, dans [PlayerEntityScan.KeyframesUS], de la premiere et de
	// la derniere image-cle porteuse ou l'entite est lue.
	FirstKF, LastKF int
	// Seen est le nombre d'images-cles porteuses ou elle est lue. `LastKF - FirstKF + 1 - Seen`
	// est son nombre de TROUS.
	Seen int
	// Unstable dit que l'index OU le designateur a change d'une lecture a l'autre. Mesure : zero
	// sur 22 films (note equipe) et sur les sept bobines. Une telle entite n'est pas une lecture
	// sure : la publication la COMPTE et ne la lie a personne.
	Unstable bool
}

// PlayerEntityScan est ce que le balayage `ti=9` rend PAR ENTITE.
type PlayerEntityScan struct {
	// Scanned dit que le balayage a tourne sur un registre qui porte le designateur en i0 de
	// ti=9. Faux = aucune lecture (registre absent, composant inattendu, faits anterieurs a ce
	// lot) : une liste vide ne dit alors RIEN de l'absence des occupants.
	Scanned bool
	// KeyframesUS porte l'horodatage des images-cles PORTEUSES — celles qui contiennent au moins
	// un record ti=9 —, dans l'ordre du film. La PREMIERE image-cle d'un film peut ne rien porter
	// (preambule, sonde P4 : f-186 sur `b1ad85eb`) : elle n'est pas porteuse, et « present au
	// depart » se lit a la premiere image-cle qui l'est.
	KeyframesUS []uint64
	// Entities : une entree par entite, triees par (FirstKF, Slot).
	Entities []PlayerEntity
	// Doutes : les couples (image-cle porteuse, entite) ou l'ABSENCE de l'entite N'EST PAS
	// PROUVEE — la marche de cette image-cle a ecarte par REPLI un candidat ti=9 de son slot, ou a
	// atteint son record sans pouvoir le lire (lot D-fix, 2026-09-24 ; cf. l'en-tete du fichier).
	// Tries par (Rang, Slot), restreints aux slots des entites lues. Vide : toutes les absences
	// sont prouvees.
	Doutes []DouteDAbsence
}

// DouteDAbsence est UNE absence non prouvee : a l'image-cle porteuse de rang `Rang`, l'entite du
// slot `Slot` n'a pas ete lue, et la marche ne prouve pas qu'elle n'y etait pas.
type DouteDAbsence struct {
	Rang, Slot int
}

// Holes rend le nombre total de TROUS : les couples (entite, image-cle porteuse) ou l'entite
// manque entre sa premiere et sa derniere lecture. Aucun n'est un depart (cf. l'en-tete).
func (s PlayerEntityScan) Holes() int {
	n := 0
	for _, e := range s.Entities {
		n += e.LastKF - e.FirstKF + 1 - e.Seen
	}
	return n
}

// UnstableEntities rend le nombre d'entites dont l'index ou le designateur a change.
func (s PlayerEntityScan) UnstableEntities() int {
	n := 0
	for _, e := range s.Entities {
		if e.Unstable {
			n++
		}
	}
	return n
}

// KeyframeUS rend l'horodatage de l'image-cle porteuse de rang `rang`, et faux hors des bornes.
func (s PlayerEntityScan) KeyframeUS(rang int) (uint64, bool) {
	if rang < 0 || rang >= len(s.KeyframesUS) {
		return 0, false
	}
	return s.KeyframesUS[rang], true
}

// AtStart dit que l'occupant de l'entite etait la au coup d'envoi du film : aucune image-cle
// porteuse ANTERIEURE a sa premiere lecture ne PROUVE son absence (cf. l'en-tete du fichier). Sans
// doute, c'est « lue a la premiere image-cle porteuse ».
func (s PlayerEntityScan) AtStart(e PlayerEntity) bool {
	_, ok := s.AbsenceProuveeAvant(e)
	return !ok
}

// AtEnd dit que l'occupant de l'entite etait encore la a la fin du film : aucune image-cle
// porteuse POSTERIEURE a sa derniere lecture ne prouve son absence.
func (s PlayerEntityScan) AtEnd(e PlayerEntity) bool {
	_, ok := s.AbsenceProuveeApres(e)
	return !ok
}

// AbsenceProuvee dit si l'absence de l'entite du slot `slot` a l'image-cle porteuse de rang `rang`
// est PROUVEE : aucun doute n'y porte sur son slot. Elle ne dit rien d'une image-cle ou l'entite
// est lue.
func (s PlayerEntityScan) AbsenceProuvee(slot, rang int) bool {
	i := sort.Search(len(s.Doutes), func(k int) bool {
		d := s.Doutes[k]
		return d.Rang > rang || (d.Rang == rang && d.Slot >= slot)
	})
	return i >= len(s.Doutes) || s.Doutes[i] != DouteDAbsence{Rang: rang, Slot: slot}
}

// AbsenceProuveeAvant rend le rang de la DERNIERE image-cle porteuse, anterieure a la premiere
// lecture de l'entite, qui prouve son absence — faux s'il n'y en a aucune.
func (s PlayerEntityScan) AbsenceProuveeAvant(e PlayerEntity) (int, bool) {
	for r := e.FirstKF - 1; r >= 0; r-- {
		if s.AbsenceProuvee(e.Slot, r) {
			return r, true
		}
	}
	return -1, false
}

// AbsenceProuveeApres rend le rang de la PREMIERE image-cle porteuse, posterieure a la derniere
// lecture de l'entite, qui prouve son absence — faux s'il n'y en a aucune.
func (s PlayerEntityScan) AbsenceProuveeApres(e PlayerEntity) (int, bool) {
	for r := e.LastKF + 1; r < len(s.KeyframesUS); r++ {
		if s.AbsenceProuvee(e.Slot, r) {
			return r, true
		}
	}
	return -1, false
}

// ImagesClesDouteuses rend le nombre d'images-cles porteuses ou l'absence d'au moins une entite
// lue n'est pas prouvee.
func (s PlayerEntityScan) ImagesClesDouteuses() int {
	n, dernier := 0, -1
	for _, d := range s.Doutes {
		if d.Rang != dernier {
			n, dernier = n+1, d.Rang
		}
	}
	return n
}

// BornesDifferees rend le nombre d'entites dont une borne de presence (arrivee ou depart) N'EST
// PAS posee sur l'image-cle voisine de sa fenetre, parce que celle-ci ne prouve pas son absence :
// la borne recule a la premiere absence prouvee, ou jusqu'au bord du film.
func (s PlayerEntityScan) BornesDifferees() int {
	n := 0
	for _, e := range s.Entities {
		avant := e.FirstKF > 0 && !s.AbsenceProuvee(e.Slot, e.FirstKF-1)
		apres := e.LastKF < len(s.KeyframesUS)-1 && !s.AbsenceProuvee(e.Slot, e.LastKF+1)
		if avant || apres {
			n++
		}
	}
	return n
}

// entiteEnCours accumule les lectures d'UNE entite pendant le balayage.
type entiteEnCours struct {
	slot        int
	index       map[int]int
	designateur map[int]int
	first, last int
	seen        int
}

// accumulateurDEntites porte le balayage par entite, image-cle par image-cle.
type accumulateurDEntites struct {
	keyframes []uint64
	entites   map[int]*entiteEnCours
	// doutes : par rang d'image-cle porteuse, les slots dont l'absence n'y est pas prouvee.
	doutes map[int]map[int]bool
}

func nouvelAccumulateurDEntites() *accumulateurDEntites {
	return &accumulateurDEntites{entites: map[int]*entiteEnCours{}, doutes: map[int]map[int]bool{}}
}

// douterDe note, a l'image-cle porteuse de rang `rang`, les slots ti=9 dont l'absence n'est pas
// prouvee : les records atteints mais illisibles, et les candidats ti=9 que le repli de la marche
// a ecartes — sauf ceux qu'elle a lus ailleurs dans la meme image-cle.
func (a *accumulateurDEntites) douterDe(rang int, lus map[int]bool, illisibles []int,
	ecartes []KeyframeRec) {
	douter := func(slot int) {
		if lus[slot] {
			return
		}
		if a.doutes[rang] == nil {
			a.doutes[rang] = map[int]bool{}
		}
		a.doutes[rang][slot] = true
	}
	for _, s := range illisibles {
		douter(s)
	}
	for _, r := range ecartes {
		if r.TI == managedPlayerTypeIndex {
			douter(r.Slot)
		}
	}
}

// doutesPublies rend les doutes restreints aux slots des entites LUES (un candidat ecarte d'un
// slot qu'aucune image-cle ne lit n'est l'occupant de personne), tries par (Rang, Slot).
func (a *accumulateurDEntites) doutesPublies() []DouteDAbsence {
	var out []DouteDAbsence
	for rang, slots := range a.doutes {
		for s := range slots {
			if _, connue := a.entites[s]; connue {
				out = append(out, DouteDAbsence{Rang: rang, Slot: s})
			}
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Rang != out[j].Rang {
			return out[i].Rang < out[j].Rang
		}
		return out[i].Slot < out[j].Slot
	})
	return out
}

// ouvrirImageCle inscrit une image-cle PORTEUSE et rend son rang.
func (a *accumulateurDEntites) ouvrirImageCle(ts uint64) int {
	a.keyframes = append(a.keyframes, ts)
	return len(a.keyframes) - 1
}

// noter inscrit UNE lecture reussie (index et designateur valides) d'une entite, a l'image-cle de
// rang `rang`. Deux records du meme slot dans la meme image-cle ne comptent qu'UNE presence.
func (a *accumulateurDEntites) noter(rang, slot, index, designateur int) {
	e := a.entites[slot]
	if e == nil {
		e = &entiteEnCours{slot: slot, index: map[int]int{}, designateur: map[int]int{},
			first: rang, last: rang}
		a.entites[slot] = e
	}
	e.index[index]++
	e.designateur[designateur]++
	if e.seen == 0 || rang != e.last {
		e.seen++
	}
	e.last = rang
}

// divergences rend le nombre d'entites dont le DESIGNATEUR a change — le compteur historique
// `TeamScanReport.EntityDivergences`, calcule sur les memes lectures.
func (a *accumulateurDEntites) divergences() int {
	n := 0
	for _, e := range a.entites {
		if len(e.designateur) > 1 {
			n++
		}
	}
	return n
}

// publier rend le balayage par entite, trie de facon reproductible.
func (a *accumulateurDEntites) publier() PlayerEntityScan {
	out := PlayerEntityScan{Scanned: true, KeyframesUS: a.keyframes}
	if len(a.entites) == 0 {
		return out
	}
	out.Doutes = a.doutesPublies()
	out.Entities = make([]PlayerEntity, 0, len(a.entites))
	for _, e := range a.entites {
		idx, idxInstable := majoritaire(e.index)
		des, desInstable := majoritaire(e.designateur)
		out.Entities = append(out.Entities, PlayerEntity{
			Slot: e.slot, Index: idx, Team: des, FirstKF: e.first, LastKF: e.last,
			Seen: e.seen, Unstable: idxInstable || desInstable,
		})
	}
	// L'ORDRE EST IMPOSE : l'ordre d'iteration d'une map Go est aleatoire, et des faits qui
	// changent d'octets sans changer de contenu ne se comparent plus.
	sort.Slice(out.Entities, func(i, j int) bool {
		if out.Entities[i].FirstKF != out.Entities[j].FirstKF {
			return out.Entities[i].FirstKF < out.Entities[j].FirstKF
		}
		return out.Entities[i].Slot < out.Entities[j].Slot
	})
	return out
}

// majoritaire rend la valeur la plus lue (la plus petite a egalite, pour que le choix soit
// reproductible) et dit si plus d'une valeur a ete lue.
func majoritaire(comptes map[int]int) (int, bool) {
	best, n := 0, -1
	for v, c := range comptes {
		if c > n || (c == n && v < best) {
			best, n = v, c
		}
	}
	return best, len(comptes) > 1
}
