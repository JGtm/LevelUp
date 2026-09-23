package fallback

// registre_replay_places.go — les replis des PLACES et des PRÉSENCES du roster du rejeu
// (`internal/games/halo_infinite/film/replay/sieges.go`, `successions.go`), posés par le lot M2.3
// de la campagne « retours rejeu » (2026-09-23) sur la RÈGLE DES PLACES de l'utilisateur.
//
// UN FICHIER NEUF, ET C'EST LA LIMITE DE 500 LIGNES : `registre_replay_identites.go` y était à
// trois lignes près. L'entrée `repli_siege_du_remplacant_par_appariement_ordinal` (lot 1.9.14) en
// sort : le chaînage par équipe la REMPLACE — il la généralise à tout arrivant (bots compris) et
// ne passe plus qu'après les deux lectures de la place (l'index de la table, les tirs).

// dateM2RetoursRejeu : le jour du lot M2.3 — quatre entrées y naissent, et `goconst` refuse à juste
// titre une troisième occurrence du littéral. Une DATE, pas un lot (cf. [dateVague2]).
const dateM2RetoursRejeu = "2026-09-23"

var registreReplayPlaces = []Repli{
	{
		Nom: "repli_place_du_remplacant_par_chainage_d_equipe",
		Fait: "quelle place de la table du debut (quelle fiche) un arrivant tient quand ni son " +
			"index ni ses tirs ne la disent",
		Mecanisme: "chainage PAR EQUIPE a presences certaines disjointes : la place de son equipe " +
			"liberee le plus tot avant son arrivee, puis une place de son equipe sans occupant " +
			"anterieur, puis une place de la table jamais tenue tant que l'equipe n'a pas sa " +
			"capacite estimee (cf. `repli_place_ouverte_sous_la_capacite_estimee`) ; sinon le repli " +
			"suivant, ou aucune place et le siege reste l'index (compte `coverage.seats.sansPlace`)",
		// FILM MUET, ET LE NEGATIF EST MESURE (sonde P4, `b1ad85eb`) : les bots n'ecrivent AUCUN
		// tir long (record 105) — 0 tir d'index >= 8, 0 tir de la place 5 pendant les presences
		// de 343 Hundy et 343 PardonMy, 0 tir de la place 1 pendant les 170 s de 343 Brew Dog. La
		// place d'un bot, et celle d'un arrivant qui ne tire pas, ne se LIT donc nulle part.
		Condition: CondFilmMuet,
		// APRES LES DEUX LECTURES : l'index de la table (`lu`) puis les tirs (`tirs`) posent leurs
		// places d'abord ; le chainage ne se sert que de ce qui reste libre.
		Ordre: OrdreApresLecture,
		Sites: []Site{{
			Fichier: pkgReplay + "sieges.go",
			Ancre:   "in.horloge.fb.DeclencheN(fallback.NomPlaceDuRemplacantParChainageDEquipe, cov.Apparies)",
		}, {
			Fichier: pkgReplay + "sieges_places.go",
			Ancre:   "pp.asseoir(i, p, SeatSourceApparie)",
		}},
		DatePose: dateM2RetoursRejeu,
		CibleRetrait: "le lot qui lira la place d'un bot dans le film (le canal de ses tirs — la " +
			"variante courte du record 105, a instruire — ou un autre champ de place)",
		CritereRetrait: "coverage.seats.apparies a 0 sur `b1ad85eb`, `43e96765` et `0d265ab0` " +
			"(bots et arrivants sans tir) : chaque place y est `lu` ou `tirs`",
		CompteurBranche: true,
	},
	{
		Nom: "repli_place_ouverte_sous_la_capacite_estimee",
		Fait: "combien de places (de fiches) une equipe peut tenir, quand un arrivant ne trouve " +
			"aucune place libre de la table du debut",
		Mecanisme: "capacite ESTIMEE de l'equipe = la plus haute de trois bornes basses : les " +
			"sieges de la table repartis entre les equipes lues, la plus grande equipe de la table " +
			"apres la pose des origines (equipes d'un mode de meme taille), le plus grand nombre de " +
			"ses entites ti=9 lues a une meme image-cle ; sous cette capacite, l'arrivant OUVRE une " +
			"place (son index, ou un numero au-dela des 64 index) ; a la capacite, aucune place",
		// LA LECTURE A TOURNE ET N'A PAS TRANCHE : l'index (`lu`), les tirs (`tirs`) et le chainage
		// n'ont trouve aucune place libre pour cet arrivant. La TAILLE D'EQUIPE DU MODE n'est pas
		// lue : que le film l'ecrive (variante de jeu) est a instruire — rien ne l'a mesure.
		Condition: CondNonResolu,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{
			Fichier: pkgReplay + "sieges.go",
			Ancre:   "in.horloge.fb.DeclencheN(fallback.NomPlaceOuverteSousLaCapaciteEstimee, cov.PlacesOuvertes)",
		}, {
			Fichier: pkgReplay + "sieges_places.go",
			Ancre:   "pp.asseoir(i, pp.ouvrirUnePlace(i), SeatSourceOuverte)",
		}},
		DatePose: dateM2RetoursRejeu,
		CibleRetrait: "le lot qui lira la taille d'equipe du mode dans le film (proprietes de la " +
			"variante de jeu), si le film l'ecrit — sinon le negatif mesure fait passer ce repli en " +
			"`film_muet`",
		CritereRetrait: "sur le parc republie, la capacite lue egale la capacite estimee sur chaque " +
			"document (111), et `coverage.seats.placesOuvertes` ne compte que des arrivants dont " +
			"l'equipe etait incomplete a la table du debut",
		CompteurBranche: true,
	},
	{
		Nom:  "repli_presence_par_enveloppe_des_vies",
		Fait: "de quand a quand un joueur tient sa place quand le film n'a pas ete lu par entite ti=9",
		Mecanisme: "presence = enveloppe de ses vies publiees (premiere vie -> fin de la derniere), " +
			"et le DERNIER occupant de chaque place reste affiche jusqu'a la fin du film — la regle " +
			"d'avant, « mourir n'est pas partir », que sans entite rien ne permet de trancher",
		// LECTURE NON PORTEE : les entites ti=9 se lisent (lot M2.1), mais pas sur un registre sans
		// ti=9 ou sans designateur en i0, ni sur un chemin qui assemble depuis des positions deja
		// decodees (`BuildFromPositions`, collecteur de sync) ou depuis des faits anterieurs au
		// schema de faits 4 (qui se redecodent).
		Condition: CondLectureNonPortee,
		Ordre:     OrdreSansLecture,
		Sites: []Site{{
			Fichier: pkgReplay + "sieges.go",
			Ancre:   "in.horloge.fb.DeclencheN(fallback.NomPresenceParEnveloppeDesVies, pp.tenirLesDerniersJusquALaFin())",
		}},
		DatePose: dateM2RetoursRejeu,
		CibleRetrait: "chaque chemin qui publie un roster porte les entites ti=9 (le collecteur de " +
			"sync y compris) ; ce qui reste est un film sans ti=9, a mesurer",
		CritereRetrait: "0 declenchement sur le parc republie apres re-decodage (111 documents) : " +
			"`coverage.seats.presences` vaut `film` partout",
		CompteurBranche: true,
	},
	{
		Nom: "repli_vie_de_bot_par_relais_de_la_base",
		Fait: "a quel bot appartient une vie restee anonyme apres la lecture des corps, des " +
			"entites et des sieges de bot",
		Mecanisme: "chaine de fenetres autour de l'heure d'arrivee que la BASE donne a un bot arrive " +
			"en cours (`joined_in_progress`) : l'unique vie anonyme nee dans [arrivee - 2 s, " +
			"arrivee + 20 s], puis l'unique vie nee dans [fin de la precedente + 2 s, fin + 25 s] ; " +
			"deux candidates se departagent par un tir de l'index du bot, sinon la chaine s'arrete",
		// FILM MUET : le relais n'est lu nulle part pour une vie que le lien corps -> entite n'a pas
		// nommee (bot present moins d'une image-cle, corps sans record de creation). La base est
		// en RETARD d'environ 20 s sur le film (mediane -22,3 s sur 45 relais, rapport equipes) :
		// ce repli manque la premiere vie d'un bot remplacant, et c'est ce que le lien par entite
		// ferme en amont.
		Condition: CondFilmMuet,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{
			Fichier: pkgReplay + "successions.go",
			Ancre:   "fb.DeclencheN(fallback.NomVieDeBotParRelaisDeLaBase, claimed)",
		}},
		DatePose: dateM2RetoursRejeu,
		CibleRetrait: "le lot qui datera la presence d'un bot de moins d'une image-cle par le film " +
			"(BOT_METADATA + creation du corps) sans passer par la base",
		CritereRetrait: "0 vie attribuee par relais sur le parc republie : toute vie de bot est " +
			"nommee par son corps (creation dans la fenetre de son entite)",
		CompteurBranche: true,
	},
}
