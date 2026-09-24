package replay

// sieges.go — LA PLACE D'UN OCCUPANT, SA PRESENCE, ET CE QUI LES DECIDE (lot 1.9.14 ; refondu au
// lot M2.3 de la campagne « retours rejeu », 2026-09-23).
//
// # LA REGLE DES PLACES (utilisateur, 2026-09-23 — elle prime sur toute autre lecture)
//
// « quand un joueur part, il libère la place de sa fiche de joueur pour son remplaçant ok ? [...]
// le nombre de joueur dans un match est fini, il y a un maximum. » Une equipe a un nombre FINI de
// places ; une fiche = une place ; un partant LIBERE sa place et son remplacant (bot ou humain)
// prend CETTE place (Q23 : un bot remplace le partant, un humain qui arrive remplace le bot) ;
// jamais plus de fiches que de places ; un joueur parti ne reste jamais affiche.
//
// # CE QU'ETAIT LE DEFAUT (rapport `RAPPORT_equipes_b1ad85eb.md`, C1 et C2)
//
// Le siege publie etait l'INDEX de participant, et un remplacant n'en herite presque jamais (le
// film reprend l'index d'un partant 2 fois sur 35 arrivees) ; l'appariement ordinal de repli ne
// chainait que les partants de la table ayant une equipe et des vies. Sur `b1ad85eb` : Eagle a 5
// sieges, Cobra a 5, et le client gardait un parti affiche faute de successeur sur SON siege.
//
// # CE QUI EST UNE PLACE, ET CE QUI LA DECIDE, DANS L'ORDRE
//
// Les PLACES sont les sieges de la table du DEBUT du film (`chunk_00`, lot 1.5) — WNBA Fan A5,
// assis au siege 5 de `b1ad85eb` et parti avant le coup d'envoi, y laisse une place que les bots
// puis Claudors tiennent. L'equipe d'une place est celle de ses occupants.
//
//	LECTURE `lu`       une entree dont l'index EST un siege de la table l'occupe : les occupants
//	                   du depart, et l'arrivant qui REPREND l'index d'un partant ;
//	LECTURE `tirs`     l'index de tireur d'un tir est la PLACE, et le remplacant en herite (rapport,
//	                   3 remplacements sur 3 ; sonde P4 : les 84 tirs de la place 5 tombent tous dans
//	                   les vies de Claudors, index 10) : un arrivant dont les tirs non couverts
//	                   designent A L'UNANIMITE une place, libre pendant sa presence, la lit ;
//	REPLI `apparie`    CHAINAGE PAR EQUIPE, a presences disjointes : la place de son equipe liberee
//	                   le plus tot, puis une place de son equipe sans occupant anterieur, puis une
//	                   place de la table jamais tenue — tant que l'equipe n'a pas sa capacite. Nomme
//	                   et compte (`repli_place_du_remplacant_par_chainage_d_equipe`) : les bots
//	                   n'ecrivent aucun tir long (sonde P4), leur place ne se lit pas encore ;
//	REPLI `ouverte`    aucune place libre, mais l'equipe n'a pas sa capacite : l'arrivant OUVRE la
//	                   place que la table du debut ne portait pas (`e5adf7b2` : 23 sieges pour
//	                   12 contre 12, le douzieme arrive a la 64e seconde). Nomme et compte
//	                   (`repli_place_ouverte_sous_la_capacite_estimee`) : la capacite est ESTIMEE,
//	                   la taille d'equipe du mode n'est pas lue dans le film (cf. sieges_places.go) ;
//	AUCUNE `index`     rien de ce qui precede (equipe inconnue, ou equipe pleine dont aucune place
//	                   n'est libre pendant sa presence — deux lectures qui se contredisent) : le
//	                   siege reste l'index, COMPTE (`sansPlace`) — c'est le seul chemin par lequel une
//	                   equipe depasserait ses places, et `depassements` le mesure.
//
// # LA PRESENCE SE BORNE AU SUCCESSEUR
//
// Sur une place, l'affichage d'un occupant s'arrete la veille de l'arrivee du suivant (une
// entite ne se voit qu'aux images-cles : sans cette borne, un partant et son remplacant
// tiendraient la meme place vingt secondes). Entre les deux, la place est VIDE (Q20).
//
// SANS ENTITE LUE (cf. occupants.go), la presence est l'enveloppe des vies, et le DERNIER occupant
// de chaque place reste affiche jusqu'a la fin — la regle d'avant, « mourir n'est pas partir »,
// que sans entite rien ne permet de trancher. Repli nomme et compte
// (`repli_presence_par_enveloppe_des_vies`).
//
// # CE FICHIER NE DESSINE RIEN
//
// Il publie `roster[].seat`, `roster[].seatSource` et `roster[].presence`. La regle d'affichage —
// une tuile par place, son occupant a l'instant lu, sinon vide — vit cote web (`seatLogic.ts`).

import (
	"sort"

	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/fallback"
)

// SeatSourceLu / SeatSourceTirs / SeatSourceApparie / SeatSourceOuverte / SeatSourceIndex : les
// cinq provenances d'un siege publie (cf. l'en-tete).
//
// ELLES NE SONT PAS DECORATIVES. Un client qui dessine une place tenue par deux joueurs
// successifs doit pouvoir distinguer ce que le film ECRIT (`lu`, `tirs`) de ce que la pose a
// DEDUIT (`apparie`, `ouverte`) et de ce qui n'a pas trouve de place (`index`).
const (
	// SeatSourceLu : le siege est l'index que le film ecrit pour cette entree — un siege de la
	// table du debut.
	SeatSourceLu = "lu"
	// SeatSourceTirs : la place est lue dans l'index de tireur des tirs de l'arrivant.
	SeatSourceTirs = "tirs"
	// SeatSourceApparie : la place vient du chainage par equipe, un REPLI (cf. l'en-tete).
	SeatSourceApparie = "apparie"
	// SeatSourceOuverte : l'arrivant a ouvert une place que la table du debut ne portait pas, son
	// equipe etant sous sa capacite estimee — un REPLI (cf. l'en-tete). Le siege est son index,
	// ou un numero au-dela des index quand une autre place porte deja le sien.
	SeatSourceOuverte = "ouverte"
	// SeatSourceIndex : aucune place : le siege est l'index de l'entree, hors de la table.
	SeatSourceIndex = "index"
)

// Presences / sources d'une presence publiee (`coverage.seats.presences`).
const (
	// PresencesDuFilm : les presences viennent des entites ti=9 et de BOT_METADATA.
	PresencesDuFilm = "film"
	// PresencesDesVies : repli — enveloppe des vies, dernier occupant d'une place jusqu'a la fin.
	PresencesDesVies = "vies"
)

// SeatCoverage est ce que la pose des places a lu, deduit, borne, et laisse sans place.
//
// SES CHIFFRES CENTRAUX SONT `depassements` (0 attendu : une equipe n'affiche jamais plus
// d'occupants que de places) et le couple `entrees` / `occupantsMax`.
type SeatCoverage struct {
	// Entrees est le denominateur : les entrees de roster publiees.
	Entrees int `json:"entrees"`
	// Sieges est le nombre de places que la colonne AFFICHE : les sieges DISTINCTS des entrees
	// qu'une presence couvre (les places tenues, plus l'index d'une entree presente restee sans
	// place). Une entree que rien ne montre n'a de fiche a aucune image, et ne compte pas.
	Sieges int `json:"sieges"`
	// Lus : les entrees dont le siege est l'index que le film ecrit (siege de la table).
	Lus int `json:"lus"`
	// PlacesTirs : les entrees dont la place est LUE dans leurs tirs.
	PlacesTirs int `json:"placesTirs"`
	// Apparies : les entrees dont la place vient du chainage par equipe (REPLI).
	Apparies int `json:"apparies"`
	// PlacesOuvertes : les entrees qui ont ouvert une place hors de la table du debut, leur equipe
	// etant sous sa capacite estimee (REPLI).
	PlacesOuvertes int `json:"placesOuvertes"`
	// SansPlace : les entrees presentes qu'aucune voie n'a placees (siege = leur index).
	SansPlace int `json:"sansPlace"`
	// ReprisesEcrites : les sieges que le film donne a PLUSIEURS entrees par leur index.
	ReprisesEcrites int `json:"reprisesEcrites"`
	// Arrivants : les entrees dont l'index n'a pas de siege dans la table du DEBUT du film.
	Arrivants int `json:"arrivants"`
	// PresencesCloses : les entrees dont la presence publiee s'acheve AVANT la derniere frame.
	// Avec `presences = film` c'est un DEPART lu ; avec `vies`, un depart OU une mort de fin de
	// partie, que rien ne distingue.
	PresencesCloses int `json:"presencesCloses"`
	// SansPresence : les entrees qu'aucune presence ne couvre, a aucun instant (ni entite, ni
	// declaration, ni vie) : elles n'ont de fiche a aucun T.
	SansPresence int `json:"sansPresence"`
	// OccupantsMax : le plus grand nombre d'entrees affichees a une meme frame.
	OccupantsMax int `json:"occupantsMax"`
	// Capacite : la plus grande capacite ESTIMEE d'une equipe (cf. sieges_places.go,
	// [poseDesPlaces.capaciteDe]) — table du debut et entites lues, jamais les places posees. 0
	// sans table.
	Capacite int `json:"capacite"`
	// Depassements : les couples (frame, equipe) ou une equipe affiche plus d'occupants que sa
	// CAPACITE (revue M2-R6 : le plafond etait les places que la pose elle-meme avait attribuees,
	// une mesure circulaire). Les entrees sans place (`index`) y comptent. 0 attendu.
	Depassements int `json:"depassements"`
	// PlacesEnTrop : par equipe, les places AFFICHEES au-dela de sa capacite, sommees. Le web rend
	// une tuile par place a chaque image (vide ou non) : c'est le nombre de tuiles de trop. 0
	// attendu.
	PlacesEnTrop int `json:"placesEnTrop"`
	// SansEquipe : les entrees presentes sans equipe lue. Leur tuile se range par la feuille de
	// match (cote web), hors de toute capacite : chacune est a lire.
	SansEquipe int `json:"sansEquipe"`
	// IdentitesHorsRoster : les identites qui nomment une vie publiee sans entree de roster (revue
	// M2-R1). Sans place ni presence, le web ne leur rend aucune tuile. 0 attendu.
	IdentitesHorsRoster int `json:"identitesHorsRoster"`
	// BotsSuccesseurs : les bots entres au roster sur l'index d'un humain dont ils sont les
	// successeurs LUS (entites disjointes, cf. roster_bots_successeurs.go).
	BotsSuccesseurs int `json:"botsSuccesseurs"`
	// PresencesParLesVies : sur un film balaye, les entrees presentes qu'aucune entite ni
	// declaration ne porte — leur presence vient de leurs seules vies (REPLI par entree,
	// `repli_presence_d_une_entree_par_ses_vies`).
	PresencesParLesVies int `json:"presencesParLesVies"`
	// RelaisBornes : les presences dont l'affichage a ete borne par l'arrivee du successeur sur
	// la meme place.
	RelaisBornes int `json:"relaisBornes"`
	// Chevauchements : les couples d'occupants d'une meme place dont les presences CERTAINES se
	// recouvrent — une contradiction des lectures, comptee.
	Chevauchements int `json:"chevauchements"`
	// TirsContestes : les arrivants dont les tirs designent plusieurs places, ou une place prise.
	TirsContestes int `json:"tirsContestes"`
	// TirsIndexTronque : l'index de tireur persiste (4 bits) ne distingue pas les places au-dela
	// de 15 (BTB) — la lecture par les tirs s'est abstenue sur tout le film.
	TirsIndexTronque bool `json:"tirsIndexTronque,omitempty"`
	// Presences dit d'ou viennent les presences publiees : `film` ou `vies` (repli).
	Presences string `json:"presences"`
	// EntitesNonLiees / EntitesContestees / TrousDEntite : ce que la liaison aux entites ti=9 n'a
	// pas pose (cf. occupants.go).
	EntitesNonLiees   int `json:"entitesNonLiees"`
	EntitesContestees int `json:"entitesContestees"`
	TrousDEntite      int `json:"trousDEntite"`
	// ImagesClesDouteuses : les images-cles porteuses ou l'ABSENCE d'au moins une entite lue N'EST PAS
	// PROUVEE — la marche de leur table a ecarte par repli un candidat ti=9 de son slot, ou a atteint
	// son record sans pouvoir le lire (lot D-fix, 2026-09-24). Aucune arrivee tardive ni aucun depart
	// ne s'y conclut. 0 attendu.
	ImagesClesDouteuses int `json:"imagesClesDouteuses"`
	// BornesDifferees : les entites dont une borne de presence (arrivee ou depart) n'est PAS posee
	// sur l'image-cle voisine de leur fenetre, douteuse, mais recule jusqu'a la premiere absence
	// prouvee — ou jusqu'au bord du film. 0 attendu.
	BornesDifferees int `json:"bornesDifferees"`
	// SansTableDuFilm : le film ne porte pas sa table de depart, donc aucune place n'est
	// decidable. Ce n'est pas un repli : c'est une abstention, et elle se lit ici.
	SansTableDuFilm bool `json:"sansTableDuFilm,omitempty"`
}

// entreesDesPlaces porte ce que la pose consomme hors du roster et des occupants.
type entreesDesPlaces struct {
	table FilmPlayerTable
	fire  []FireEventRef
	// horloge porte la grille du document et le compteur de replis de la cuisson.
	horloge replayClock
}

// poserLesSieges ECRIT la place, la provenance et la presence de chaque entree du roster, EN
// PLACE, et rend sa couverture. PURE au sens du decodage : ni octet de film, ni base.
func poserLesSieges(roster []RosterEntry, occ occupants, in entreesDesPlaces) SeatCoverage {
	cov := SeatCoverage{Entrees: len(roster), Presences: PresencesDesVies,
		EntitesNonLiees: occ.entitesNonLiees, EntitesContestees: occ.entitesContestees,
		TrousDEntite: occ.trous, ImagesClesDouteuses: occ.imagesDouteuses,
		BornesDifferees: occ.bornesDifferees}
	if occ.balaye {
		cov.Presences = PresencesDuFilm
	}
	if len(roster) == 0 {
		return cov
	}
	pp := nouvellePoseDesPlaces(roster, &occ, in)
	cov.SansTableDuFilm = pp.places == nil
	for i := range roster {
		roster[i].Seat, roster[i].SeatSource = roster[i].FilmIndex, SeatSourceLu
	}
	if pp.places != nil {
		pp.poserLesOrigines()
		pp.ouvrirAuCoupDEnvoi()
		pp.estimerLaCapacite()
		cov.PlacesTirs, cov.TirsContestes, cov.TirsIndexTronque = pp.lireLesPlacesDansLesTirs(in.fire)
		cov.Apparies, cov.PlacesOuvertes, cov.SansPlace = pp.chainerLesArrivants()
		in.horloge.fb.DeclencheN(fallback.NomPlaceDuRemplacantParChainageDEquipe, cov.Apparies)
		in.horloge.fb.DeclencheN(fallback.NomPlaceOuverteSousLaCapaciteEstimee, cov.PlacesOuvertes)
		pp.marquerLesArrivantsSansPresence()
	}
	cov.RelaisBornes, cov.Chevauchements = pp.bornerAuSuccesseur()
	pp.retirerLesAffichagesVides()
	if !occ.balaye {
		in.horloge.fb.DeclencheN(fallback.NomPresenceParEnveloppeDesVies, pp.tenirLesDerniersJusquALaFin())
	} else {
		cov.PresencesParLesVies = occ.presencesParLesVies()
		in.horloge.fb.DeclencheN(fallback.NomPresenceDUneEntreeParSesVies, cov.PresencesParLesVies)
	}
	pp.publierLesPresences()
	cov.IdentitesHorsRoster = occ.horsRoster
	cov.compterLesEntrees(roster, pp)
	return cov
}

// presencesParLesVies compte les entrees presentes dont la presence ne vient que de leurs vies.
func (o *occupants) presencesParLesVies() int {
	n := 0
	for _, e := range o.parEntree {
		if !e.lue && len(e.presence) > 0 {
			n++
		}
	}
	return n
}

// compterLesEntrees pose les compteurs que le roster publie renseigne.
func (c *SeatCoverage) compterLesEntrees(roster []RosterEntry, pp *poseDesPlaces) {
	partages, sieges := map[int]int{}, map[int]bool{}
	derniere := pp.in.horloge.frames - 1
	for i, e := range roster {
		if len(pp.occ.parEntree[i].presence) > 0 {
			sieges[e.Seat] = true
		}
		if e.SeatSource == SeatSourceLu {
			c.Lus++
			partages[e.FilmIndex]++
		}
		if pp.origines != nil && !pp.origines[e.FilmIndex] {
			c.Arrivants++
		}
		ivs := pp.occ.parEntree[i].presence
		switch {
		case len(ivs) == 0:
			c.SansPresence++
		case ivs[len(ivs)-1].aMax < derniere:
			c.PresencesCloses++
		}
	}
	for idx, n := range partages {
		if n > 1 && pp.estUnePlace(idx) {
			c.ReprisesEcrites++
		}
	}
	c.Sieges = len(sieges)
	m := pp.mesurerLAffichage()
	c.OccupantsMax, c.Depassements, c.Capacite = m.occupantsMax, m.depassements, m.capacite
	c.PlacesEnTrop, c.SansEquipe = m.placesEnTrop, m.sansEquipe
}

// siegesDuDebut rend les index que la table de `chunk_00` occupe, ou NIL quand le film ne porte
// pas sa table — nil et une table vide ne disent pas la meme chose, et tout ce fichier en
// depend.
func siegesDuDebut(table FilmPlayerTable) map[int]bool {
	if !table.Lue() {
		return nil
	}
	out := make(map[int]bool, len(table.Seats))
	for _, s := range table.Seats {
		out[s.FilmIndex] = true
	}
	return out
}

// cleDeRoster / cleDePiste : LA MEME CLE DES DEUX COTES. Un bot n'a pas de xuid (schema 36) :
// son identite est son nom. Les deux fonctions existent parce que les deux types portent cette
// identite sous des champs differents ; elles ne divergent pas.
func cleDeRoster(e RosterEntry) string {
	if e.XUID != "" {
		return e.XUID
	}
	if e.Bot && e.Name != "" {
		return botIdentityKey(e.Name)
	}
	return ""
}

func cleDePiste(tr Track) string {
	if tr.XUID != "" {
		return tr.XUID
	}
	if tr.Bot != "" {
		return botIdentityKey(tr.Bot)
	}
	return ""
}

// botIdentityKey est la forme de l'identite d'un bot, la meme que le client emploie.
func botIdentityKey(nom string) string { return "bot:" + nom }

// ordreDesArrivants trie des indices d'entree par arrivee, puis par index et par identite : l'ordre
// d'iteration d'une map Go est aleatoire, et un artefact qui change d'octets sans changer de
// contenu est indiffable.
func ordreDesArrivants(roster []RosterEntry, occ *occupants, ids []int) {
	sort.SliceStable(ids, func(a, b int) bool {
		da, db := occ.parEntree[ids[a]].presence[0].de, occ.parEntree[ids[b]].presence[0].de
		if da != db {
			return da < db
		}
		if roster[ids[a]].FilmIndex != roster[ids[b]].FilmIndex {
			return roster[ids[a]].FilmIndex < roster[ids[b]].FilmIndex
		}
		return cleDeRoster(roster[ids[a]]) < cleDeRoster(roster[ids[b]])
	})
}
