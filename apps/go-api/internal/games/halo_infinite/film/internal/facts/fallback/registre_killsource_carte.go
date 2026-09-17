package fallback

// registre_killsource_carte.go — LES REPLIS DE LA RESOLUTION DE CARTE DU COLLECTEUR.
//
// # POURQUOI CES DEUX ENTREES ONT LEUR FICHIER
//
// SCISSION DE TAILLE, PAS DE SENS (lot 1.9.4, 2026-09-15). `registre_killsource.go` franchissait
// les 500 lignes du depot (492 -> 523) parce que ce lot a REECRIT ces deux entrees-la : leurs
// cibles de retrait ont divergé de la constante `lot194` qui les portait toutes, et chacune a
// désormais sa propre justification datée. Le registre etait deja decoupe en cinq fichiers pour
// cette seule raison — `registre.go` le dit : « le decoupage en cinq fichiers ne suit que la
// limite de 500 lignes du depot et le paquet des sites ».
//
// AUCUNE ENTREE N'EST MODIFIEE PAR CETTE SCISSION, aucun champ, aucun ordre : [Table] trie par
// nom, et `registre` concatene les tranches. Le deplacement est PUR.
//
// # CE QUI LES REUNIT
//
// Les deux repondent a la meme question — QUELLE CARTE decrit ce match, et que publie-t-on quand
// on ne le sait pas. Depuis le lot 1.9.4 elles pointent le meme site, `killcollector/map_identity.go`,
// que les DEUX passes du collecteur (les positions et les distances de touche) partagent : la
// passe des touches devinait jusque-la sa carte par une signature de largeurs d'axe.

var registreKillsourceCarte = []Repli{
	{
		Nom:       "repli_carte_absente_largeurs_par_defaut",
		Fait:      "les largeurs d axe et la largeur d index de plage du chemin absolu de position, pour la marche des morts",
		Mecanisme: "aucune entree de catalogue n a ete passee a `killsource.Decode` : l invariant du profil est conserve, c est-a-dire les largeurs d UNE carte (`cliffhanger`, 13/13/14) appliquees a celle du match",
		Condition: CondSectionAbsente,
		Ordre:     OrdreDevantLaLecture,
		Sites: []Site{{
			Fichier: pkgKillsource + "decode.go",
			Ancre:   "if !c.calib.CarteLue {",
		}},
		DatePose: "2026-09-17",
		// POSE PAR LE LOT 3.4.1, QUI FERME LE DEFAUT DONT IL EST LE RESTE. Jusqu a ce lot,
		// `killsource` ne recevait AUCUNE entree de catalogue et INFERAIT ces largeurs par
		// balayage ; l inference est devenue ORACLE (V17, M3-Q8 : la valeur LUE prime), et la
		// valeur lue arrive desormais par `Options.Carte` — depuis `replaybuild.BuildBytes` et
		// depuis `killcollector`. Ce repli nomme ce qui reste : les appelants qui n ont pas de
		// base sous la main (CLI unitaire, ouvrier distant, collecteur sans
		// `WithPositionCapture`). Il n est PAS neutre — l invariant est l entree `cliffhanger` du
		// catalogue — et c est pourquoi il est AVERTI par film, en plus d etre inscrit ici.
		//
		// L ORACLE EST SON CONTROLE : sur un film dont la carte manque, `calibration.Desaccords`
		// compte ce que le balayage aurait designe contre ce qui est applique.
		CibleRetrait:    "lot qui rendra l entree de catalogue obligatoire a `killsource.Decode` (un film dont la carte est inconnue serait alors mis de cote plutot que decode aux largeurs d une autre — cf. D1 (cloture M2), question ouverte au pilote)",
		CritereRetrait:  "0 appel de production sans `Options.Carte` ; la CLI et les instruments la resolvent eux aussi, ou disent pourquoi ils ne le peuvent pas",
		CompteurBranche: false,
		CibleComptage:   comptageFamille19,
	},
	{
		Nom:       "repli_distances_de_touche_desactivees",
		Fait:      "la distance tireur -> victime de chaque touche",
		Mecanisme: "carte hors catalogue, ou positions de bipedes indisponibles : les distances sont desactivees, les touches restent comptees",
		Condition: CondSectionAbsente,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{
			Fichier: pkgKillcollector + "hits.go",
			Ancre:   "carte hors catalogue de bornes, distances desactivees",
		}},
		DatePose: dateAudit0E,
		// LE LOT 1.9.4 (2026-09-15) A TENU SA PART, ET L'ENTREE NE SORT PAS POUR AUTANT.
		//
		// Ce qu'il a ferme : la carte ne se DEVINE plus par une signature de largeurs d'axe, elle
		// se lit au NOM DE MATCH (`killcollector/map_identity.go`). Le critere des « cartes
		// jumelles » est donc satisfait — et la mesure a montre qu'elles etaient bien plus
		// nombreuses qu'annonce : 68 des 79 cartes du catalogue partagent leur signature avec une
		// autre, et sur les deux films Live Fire la signature designait `aquarius`, une AUTRE
		// carte, avec un seul candidat.
		//
		// Ce qui reste, et qui est LEGITIME : la degradation elle-meme. Une carte reellement
		// absente du catalogue de bornes, ou des positions de bipedes illisibles, n'ont pas de
		// distance a offrir — les touches restent comptees, seule leur distance manque. Ce repli
		// n'invente rien ; il NOMME un refus. Il se compte en production par TROIS compteurs
		// expvar depuis 1.9.4, un par cause (`killsource_hits_carte_non_cablee`,
		// `killsource_hits_matchs_sans_nom_de_carte`, `killsource_hits_cartes_hors_catalogue`) —
		// pas encore par `fallback.Compteur` : la passe de touches ne porte aucune cuisson, donc
		// aucun compteur par cuisson ou publier son compte.
		// LA CIBLE NE NOMME AUCUN LOT, ET C'EST EXACT : le lot qui rallumera la passe n'est pas
		// ecrit au plan. La decouverte qui l'etablit est « D1 (1.9.2) » en §4 du plan
		// (`match.weapon.accuracy` est `not_exposed` pour Infinite) — reference deplacee ici le
		// 2026-09-16, hors du champ : une cible se lit comme une echeance, pas comme une note de
		// bas de page, et un numero de decouverte s'y confond avec un numero de lot.
		CibleRetrait:    "le lot qui rallume la precision par arme : tant que la passe ne tourne pas, ses trois compteurs restent a zero par construction et ne prouvent rien",
		CritereRetrait:  "passe rallumee, puis 0 match a distances desactivees sur le parc pour les trois causes",
		CompteurBranche: false,
		CibleComptage:   "compteurs expvar deja cables (trois causes) ; `fallback.Compteur` au pas 2 de M2 si la passe rejoint une cuisson",
	},
	{
		Nom:       "repli_carte_premier_nom_resolu",
		Fait:      "quelle carte du catalogue decrit CE match",
		Mecanisme: "les identites candidates sont essayees dans l'ordre et la PREMIERE qui resout gagne, sans arbitrage",
		Condition: CondNonResolu,
		Ordre:     OrdreApresLecture,
		// SITE UNIQUE DEPUIS LE LOT 1.9.4 (2026-09-15), ET C'EST UN GAIN. La resolution vivait
		// dans `positions.go` et la passe des TOUCHES en avait une seconde, par signature de
		// largeurs d'axe. Les deux passes partagent desormais cette fonction : une seule regle
		// pour une seule question, donc un seul site a convertir le jour ou l'arbitrage existera.
		Sites: []Site{{
			Fichier: pkgKillcollector + "map_identity.go",
			Ancre:   "if entry, err := c.mapBounds.Lookup(name); err == nil {",
		}},
		DatePose: dateAudit0E,
		// POURQUOI LE LOT 1.9.4 NE LE RETIRE PAS, ALORS QU'IL EN ETAIT LA CIBLE.
		//
		// La cible ecrite au lot 1.9.0 disait « le nom de carte est resolu une fois et passe en
		// override ». Il l'est desormais — et cela ne retire pas ce repli-ci, qui porte une AUTRE
		// question : quand la base rend PLUSIEURS noms candidats, lequel decide ? Verifie sur
		// pieces (`platform/duckdb/replay_map_repo.go`, `MapKeysForMatch`) : l'ordre n'est pas
		// arbitraire, il est documente « du plus fiable au moins fiable » — le nom d'asset
		// canonique d'abord (il survit a un `map_name` reduit a un UUID), le libelle brut du
		// registre ensuite (le catalogue de modules est indexe en anglais, la cascade de langues
		// de `asset_translations` peut rendre un libelle traduit). Retirer le repli demanderait
		// d'ARBITRER entre ces deux sources, c'est-a-dire de trancher laquelle ment quand elles
		// divergent — une question de qualite du REGISTRE DES MATCHS, pas du decodeur de film,
		// et hors du perimetre d'un lot de la famille 1.9 (regle 7 : la decouverte se consigne,
		// elle ne se traite pas).
		CibleRetrait:    "un lot de qualite du registre des matchs (arbitrage entre `asset_translations` et `match_registry.map_name`), ou lot 3.x si le profil par carte rend l'identite sans la base",
		CritereRetrait:  "0 match du parc ou deux noms candidats resolvent DES ENTREES DIFFERENTES du catalogue — mesure a faire avant tout arbitrage",
		CompteurBranche: false,
		CibleComptage:   comptageFamille19,
	},
}
