package replay

// document_vehicles.go — LA FORME PUBLIEE du calque des VEHICULES, et ce que le film prouve de
// chacun de ses champs.
//
// L OBJET EST LA VIE D UN VEHICULE : ou il nait, ou il passe, qui le conduit, et jusqu a quand le
// montrer. Il est le pendant, pour `ti=40`, de ce que `groundWeapons` est pour `ti=42` — meme
// bornage par le recensement des images-cles, meme refus de publier une fin que le film ne
// montre pas.
//
// CE QUE LE CALQUE REFUSE DE DIRE, ET C EST LE POINT LE PLUS IMPORTANT DE CE FICHIER : POURQUOI
// un vehicule cesse d exister. Le lot V3 a mesure la voie « destruction datee par la mort du
// conducteur » sur 460 vies et 12 films, avec huit gates ecrits avant mesure
// (`V3_DESTRUCTION_DATEE_2026-09-02.md`). Sept echouent, et le seul qui passe est le CONTROLE.
// Les trois faits qui ferment la question :
//
//	1. AUCUNE des 460 vies n a d occupant encore a bord a la fin serree de son flux (0 sur 64
//	   vies a candidat) — un vehicule replique sa position 13 a 36 s (mediane par lot) APRES
//	   avoir ete quitte. La fin de trajectoire est une MISE AU REPOS, pas une disparition ;
//	2. la mort de l occupant PENDANT son episode est ANTI-correlee : 3 cas sur 80 (3,8 %) contre
//	   17 sur 80 (21,3 %) pour le temoin a occupant decale sur le MEME intervalle ;
//	3. la mort JUSTE APRES la sortie ne depasse pas le hasard : +7,5 points sur le corpus, sous
//	   le seuil de 10, et un seul lot sur trois au-dessus.
//
// `VehicleTrack.End` vaut donc `unknown`, et RIEN d autre. Publier `destruction` sur ce corpus
// serait affirmer ce qui a ete refute. La valeur est ANGLAISE comme toutes les enumerations du
// contrat (`pickup`/`seen`/`open`, `OriginUnknown = "unknown"`, `event`/`mixed`/`gap`) — revue
// adversariale 2026-09-02 : une seule valeur francaise aurait coute un changement de contrat
// apres backfill.

import "log/slog"

// VehicleEndUnknown est la SEULE valeur que `VehicleTrack.End` prend aujourd hui.
//
// LE CHAMP EXISTE QUAND MEME, et ce n est pas un champ mort : il est le contrat par lequel le
// client sait qu il ne doit PAS lire la disparition du sprite comme une destruction. Sans lui,
// « le sprite s efface » se lirait naturellement « le vehicule a explose » — exactement l erreur
// que la mesure interdit. D autres valeurs s ajouteront le jour ou un signal les datera : les
// deux voies restent ouvertes et sont ecrites au rapport V3 § 6 (la grammaire de bits d `i2`/`i3`
// pour `ti=40`, et l evenement de degat de type 0 de la liste, jamais decode).
const VehicleEndUnknown = "unknown"

// VehicleTrack est LA VIE D UN VEHICULE, de sa naissance a la derniere preuve de sa presence.
//
// LA VIE, PAS LE SLOT : le pool de slots reboucle et la generation ne fait que 2 bits, donc
// `(slot, gen)` est la seule cle. Deux vies d un meme slot sont deux vehicules differents.
type VehicleTrack struct {
	// Slot / Gen identifient la vie dans le film. Publies ensemble, jamais l un sans l autre.
	Slot uint32 `json:"slot"`
	Gen  uint32 `json:"gen"`
	// Chassis est le mot d identite `MPPWord32` du record de creation, en hexadecimal 8 chiffres
	// — MEME convention d ecriture que `Loadout.W` et `GroundWeapon.W`. Vide quand le record de
	// creation n a pas ete lu (la vie sort alors avec sa seule trajectoire).
	//
	// IL RESTE A COTE DE LA FAMILLE, jamais a sa place : regle du depot, on ne stocke jamais une
	// resolution qui peut s ameliorer. Un chassis absent de la table garde donc son hexadecimal a
	// l ecran, et n emprunte pas le sprite d un voisin.
	Chassis string `json:"chassis,omitempty"`
	// Family est la famille de chassis resolue par `vehicle_families.go` (`warthog`, `ghost`,
	// `mongoose`...). Vide = chassis inconnu de la table : le client dessine un marqueur neutre.
	// C est elle, et elle seule, qui nomme le sprite servi (cf. `VehicleLabel`).
	Family string `json:"family,omitempty"`
	// T0 / T1 / T1Max bornent l affichage, sur le meme axe que `Point.T`.
	//
	// MEME DOCTRINE QUE LES ARMES AU SOL : `T1` est la DERNIERE PREUVE de presence (dernier
	// echantillon de position, ou derniere image-cle qui recense la vie — le plus tardif des
	// deux), `T1Max` la PREMIERE PREUVE d absence (premiere image-cle qui ne la recense plus).
	// Entre les deux il y a ~20 s que le film ne documente pas : le client choisit son rendu dans
	// cet intervalle, mais il choisit dans du MESURE. Sans preuve d absence, `T1Max` vaut la
	// derniere frame du document — le vehicule est encore la a la fin.
	//
	// `T0` prefere l instant du record de CREATION (date a la milliseconde) au premier
	// recensement (borne a ~20 s pres) quand ce record a ete lu.
	T0    int `json:"t0"`
	T1    int `json:"t1"`
	T1Max int `json:"t1max"`
	// End est la CAUSE de la fin de vie. Voir `VehicleEndUnknown` : une seule valeur aujourd hui,
	// et c est une mesure, pas une lacune.
	End string `json:"end"`
	// Spawn est la position de NAISSANCE, en metres monde, lue dans le record de creation.
	// Absente quand ce record n a pas ete lu.
	//
	// ELLE VAUT PLUS QU UN PREMIER POINT DE TRAJECTOIRE : les vehicules naissent a des
	// emplacements FIXES et EXACTS (rayon d amas 0,00 m ; 6 emplacements sur Behemoth en grille
	// 2x3, 4-5 sur Launch Site — V2_SPAWNS_COOLDOWNS_2026-09-01 § 1), et un vehicule que personne
	// n a conduit n a AUCUN echantillon de trajectoire : sa naissance est tout ce qu on sait de
	// lui, et elle suffit a le dessiner.
	Spawn *VehicleSpawn `json:"spawn,omitempty"`
	// Samples est la trajectoire, projetee sur la grille de frames du document (un point par
	// frame, le premier observe gagne). Vide = le vehicule n a jamais bouge, ou son flux n a pas
	// ete lu — `Spawn` reste alors la seule position.
	Samples []VehicleSample `json:"samples,omitempty"`
	// Rides sont les EPISODES D OCCUPATION de cette vie, dans l ordre. Un meme vehicule en porte
	// plusieurs : la vie `slot=771` de `0d76e8f1` a connu trois occupants successifs. Vide =
	// aucun episode mesure — ce qui n est PAS « personne ne l a conduit » : la primitive
	// n attribue que 15,6 a 21,1 % des vies (limite publiee au rapport V1, item 1).
	Rides []VehicleRide `json:"rides,omitempty"`
}

// VehicleSpawn est la naissance d un vehicule : ou, et sous quel cap.
type VehicleSpawn struct {
	// X / Y / Z en metres monde, memes axes que `Point.X`/`Y`.
	X float32 `json:"x"`
	Y float32 `json:"y"`
	Z float32 `json:"z,omitempty"`
	// H serait le CAP de naissance. IL EST TOUJOURS ABSENT, et le champ est la pour le dire.
	//
	// L orientation d un vehicule vit dans la feuille 4 du default-state de `ti=40` — un
	// quaternion derriere une porte de flux, dont la largeur depend de globaux de CONFIGURATION
	// RUNTIME illisibles dans l image statique (`filmdec/default_state_ti40.go`, § 3 du dossier
	// RE). Le port la modelise ABSENTE. Un cap de naissance suppose ferait tourner le sprite dans
	// une direction que rien ne mesure ; le client oriente donc le vehicule sur son PREMIER
	// echantillon mobile, ou pas du tout.
	H *float32 `json:"h,omitempty"`
}

// VehicleSample est une position du vehicule sur l axe de frames.
type VehicleSample struct {
	// T est l index de frame, sur le meme axe que `Point.T`.
	T int `json:"t"`
	// X / Y / Z en metres monde.
	X float32 `json:"x"`
	Y float32 `json:"y"`
	Z float32 `json:"z,omitempty"`
	// H est le CAP en degres dans le plan XY, MEME convention que `Point.H` (0 = +X, 90 = +Y,
	// meme origine et meme sens que `atan2(Y, X)`).
	//
	// D OU IL VIENT, ET POURQUOI PAS D AILLEURS. Il est la direction de la VELOCITE `i1` du meme
	// record que la position. C est la SEULE orientation de vehicule que la mesure valide : `i2`
	// est REFUTE (medianes d ecart 40 a 137 deg sur quatre films, et le temoin par permutation
	// rend la meme chose — il n y a aucune association) et `i21` est ABSENT de `ti=40` (0,00 % du
	// masque sur 81 540 records, quand le meme deserialiseur le capture sur 48 a 55 % des records
	// de bipede). `i1`, lui, s accorde au deplacement a 1,7-2,1 deg de mediane (R = 0,992 a
	// 0,997), temoin par melange 51 a 88 deg.
	//
	// UN VEHICULE A L ARRET GARDE LE CAP SOUS LEQUEL IL S EST ARRETE : sous 5 m/s la direction
	// d une velocite quasi nulle est du bruit, le dernier cap connu est donc reporte. Absent
	// tant qu aucun echantillon mobile n a ete vu depuis le debut de la vie.
	//
	// PIEGE omitempty evite a l ecriture, comme `Point.H` : un cap qui s arrondit a 0 est publie
	// comme 360 (le meme angle), sans quoi il serait omis et relu comme « pas d orientation ».
	H float32 `json:"h,omitempty"`
}

// VehicleRide est un EPISODE D OCCUPATION : un joueur a bord de ce vehicule, de `T0` a `T1`.
//
// CE QU IL GARANTIT : entre `T0` et `T1`, le flux de position de ce bipede est INTERROMPU et sa
// derniere position connue etait a moins de 1,5 m de ce vehicule. C est la signature d un occupant
// attache — signal a x20,3 et x30,5 le hasard sur les deux films du gate V1, temoin fantome NUL.
//
// CE QU IL NE GARANTIT PAS : l exhaustivite (la primitive n attribue que 15,6 a 21,1 % des vies
// de vehicule), ni l unicite (une vie peut porter deux episodes chevauchants — conducteur et
// passager ne se departagent pas par la geometrie ; l ambiguite est PUBLIEE, pas cachee).
type VehicleRide struct {
	// T0 / T1 bornent l episode sur l axe de frames.
	T0 int `json:"t0"`
	T1 int `json:"t1"`
	// Slot est le slot du BIPEDE occupant : il designe la Track concernee.
	Slot uint32 `json:"slot"`
	// XUID est l identite de l occupant, en decimal (meme forme que `Track.XUID`). Vide quand le
	// pont du fil des morts n a pas nomme ce slot : l episode reste publie — le vehicule EST
	// occupe, c est son occupant qui est inconnu.
	//
	// C EST LUI QUI DONNE SA COULEUR AU VEHICULE : le client joint xuid -> equipe -> couleur,
	// exactement comme pour une trajectoire de joueur.
	//
	// ET IL N Y A PAS DE CHAMP `team` A COTE, ce qui a ete demande et mesure le 2026-09-02.
	// L EQUIPE N EST PAS DANS LE FILM — c est le meme fait que celui ecrit sur `Track.Team`, qui
	// vaut -1 pour tout le monde : le film ne porte ni camp, ni couleur, ni score par camp. Elle
	// vit dans la BASE, avec le gamertag, et le document est HORS LIGNE par construction (il se
	// cuit depuis les seuls chunks, sans ouvrir la moindre DuckDB — c est ce qui le rend
	// backfillable). Publier `team` ici obligerait soit a ouvrir la base dans le chemin de
	// cuisson (ce que tout le calque refuse), soit a la remplir A LA REQUETE cote service, ou le
	// CLIENT tient deja le scoreboard : ce serait une seconde source pour la meme jointure, et
	// c est l anti-patron n°8 du depot. LA JOINTURE RESTE DONC CHEZ LE CLIENT — et pour qu elle
	// tienne la promesse ci-dessus, elle passe par le XUID (`colorByXuidResolver`) et non plus par
	// le pont slot -> joueur, qui est MUET pendant l episode puisque le bipede ne replique plus.
	XUID string `json:"xuid,omitempty"`
	// Seat est le siege lu dans l evenement (`R(6)`), 0 = conducteur. POINTEUR : le siege 0 est
	// la valeur la PLUS frequente et la plus utile du champ (93,8 % des sorties, 21 des
	// 22 embarquements mesures), et `omitempty` sur un entier l effacerait exactement comme une
	// absence de lecture. Nil = aucun evenement apparie, ou charge trop courte.
	Seat *int `json:"seat,omitempty"`
	// Src dit D OU viennent les bornes : `event` (les deux datees a la milliseconde par la liste
	// d evenements), `mixed` (une des deux), `gap` (aucune — le seul trou de position, borne a la
	// demi-seconde du flux).
	//
	// IL EST PUBLIE PARCE QUE LES TROIS N ONT PAS LA MEME PRECISION, et qu un client qui anime
	// une montee en vehicule doit pouvoir le savoir.
	Src string `json:"src"`
	// Aim est LA VISEE DE CET OCCUPANT pendant son episode, echantillonnee sur la grille de
	// frames du document (schema 31). Vide = aucune lecture dans la fenetre.
	//
	// CE QU ELLE EST : la visee de l HOMME, pas l orientation du vehicule ni celle de la
	// tourelle. Chaque occupant — conducteur, artilleur, passager — garde son slot bipede
	// pendant tout l episode et continue d y emettre `i21`, dans des records qui ne portent
	// AUCUNE position (`filmdec.ScanFilmBipedAimOnly`). Un vehicule a donc autant de visees que
	// d occupants, et elles sont independantes.
	//
	// CE QU ELLE REMPLACE : le cap du CHASSIS, que le client employait faute de mieux pour le
	// cone du conducteur. La mesure V11 chiffre l ecart entre les deux — mediane 15,7 a
	// 21,8 deg, q3 39,6 a 52,9 deg sur 4 films — et l artilleur comme le passager n avaient,
	// eux, aucun cone du tout.
	//
	// Provenance, gates et convention d echantillonnage : `vehicle_rides_aim.go`.
	Aim []VehicleAim `json:"aim,omitempty"`
}

// VehicleAim est UNE lecture de visee d occupant, posee sur l axe de frames.
//
// LES DEUX ANGLES SONT CEUX DU PION (`Point.H` / `Point.P`), au bit pres : ils sortent du MEME
// composant `i21` et du MEME accesseur (`filmdec.aimHeadingDegFromRaw` / `aimPitchDegFromRaw`,
// detenteur unique depuis le lot V11). Le client n a donc qu une convention d angle a connaitre,
// qu il dessine le cone d un pion a pied ou celui d un occupant de vehicule.
type VehicleAim struct {
	// T est l index de frame, sur le meme axe que `Point.T` et `VehicleSample.T`.
	T int `json:"t"`
	// H est le CAP de visee en degres dans le plan XY, MEME convention que `Point.H` (0 = +X,
	// 90 = +Y, sens `atan2(Y, X)`).
	//
	// PIEGE omitempty evite a l ecriture, exactement comme `Point.H` et `VehicleSample.H` : un
	// cap qui s arrondit a 0 est publie comme 360 (le meme angle), sans quoi il serait omis et
	// relu comme « pas de visee » (cf. `headingForJSON`).
	H float32 `json:"h,omitempty"`
	// P est l ELEVATION de visee en degres, positif = vers le HAUT, MEME convention et MEME
	// reserve que `Point.P`.
	//
	// CONTRAT DE L ABSENCE, identique a celui de `Point.P` : `p` absent avec `h` present se lit
	// « A PLAT », jamais « inconnu » — publie quand |p| >= 0,05 deg, omis en dessous (cf.
	// `pitchForJSON`). Le client en fait la LONGUEUR du cone, signe compris.
	P float32 `json:"p,omitempty"`
}

// VehicleCoverage dit ce que le calque a vu, resolu, et refuse de dire.
//
// ELLE EST PUBLIEE MEME QUAND AUCUN VEHICULE NE L EST, meme raison que `placements` et
// `groundWeapons` : un film d arene sans vehicule, un film dont la bande `ti=40` est vide et un
// film qu on n a pas su balayer rendent tous trois zero vehicule — seuls ces compteurs les
// distinguent. Son ABSENCE dit encore autre chose : l appelant n a rien fourni a lire.
type VehicleCoverage struct {
	// Scanned dit que le film a ete balaye jusqu au bout (les trois lectures ont abouti).
	Scanned bool `json:"scanned"`
	// Lives est le nombre de vies RECENSEES aux images-cles ; Published celles qui sortent ;
	// NoPosition celles qui n avaient ni naissance lue ni echantillon — rien a dessiner.
	Lives      int `json:"lives"`
	Published  int `json:"published"`
	NoPosition int `json:"noPosition"`
	// Merged compte les vies FONDUES dans leur precedente (cf. `mergeVehicleRelays`) : le film
	// RE-CREE un vehicule sous un nouveau slot au lieu de le deplacer, et sans cette fusion
	// l ancienne vie restait a l ecran comme un DOUBLE, a l ancienne place, pendant l intervalle
	// non observe (~20 s). `Published` vaut donc « vies assemblees - Merged » : le compteur est
	// publie pour que la baisse du nombre de vehicules affiches soit LISIBLE et non suspecte.
	Merged int `json:"merged"`
	// WithSpawn / WithChassis : vies publiees dont le record de creation a ete lu, et parmi
	// elles celles dont le mot d identite a ete transmis.
	WithSpawn   int `json:"withSpawn"`
	WithChassis int `json:"withChassis"`
	// FamilyResolved / FamilyUnknown ventilent `WithChassis` selon que la table nomme la famille
	// ou non. LEUR SOMME EST LE DENOMINATEUR HONNETE du calque de sprites : publier
	// « N vehicules dessines » sans dire combien restent sans sprite laisserait croire a
	// l exhaustivite.
	FamilyResolved int `json:"familyResolved"`
	FamilyUnknown  int `json:"familyUnknown"`
	// UnknownChassis compte les vies par mot d identite NON RESOLU (hexadecimal 8 chiffres).
	// C est le TEMOIN qui rompt le silence : sans lui, un chassis frequent absent de la table
	// ferait disparaitre un tiers des vehicules d une carte sans que rien ne le dise.
	UnknownChassis map[string]int `json:"unknownChassis,omitempty"`
	// Samples est le nombre total de points de trajectoire publies ; WithHeading ceux qui portent
	// un cap. Un vehicule jamais conduit n a ni l un ni l autre.
	Samples     int `json:"samples"`
	WithHeading int `json:"withHeading"`
	// Rides est le nombre d episodes d occupation ; VehiclesRidden les vies qui en portent au
	// moins un ; RidesNamed ceux dont l occupant est nomme par le pont.
	Rides          int `json:"rides"`
	VehiclesRidden int `json:"vehiclesRidden"`
	RidesNamed     int `json:"ridesNamed"`
	// RidesFromEvent / RidesMixed / RidesFromGap ventilent les episodes par PRECISION de leurs
	// bornes (cf. `VehicleRide.Src`). Somme == Rides.
	RidesFromEvent int `json:"ridesFromEvent"`
	RidesMixed     int `json:"ridesMixed"`
	RidesFromGap   int `json:"ridesFromGap"`
	// RidesWithSeat : episodes dont le siege a ete lu dans un evenement.
	RidesWithSeat int `json:"ridesWithSeat"`
	// AimReads / RidesWithAim / AimSamples / AimRideFrames sont LA COUVERTURE DE LA VISEE
	// D OCCUPANT (schema 31), et les quatre sont necessaires parce qu ils distinguent quatre
	// pannes differentes que « 0 visee publiee » confondrait :
	//
	//	AimReads       lectures BRUTES rendues par le balayage du film, tous slots bipedes
	//	               confondus (`filmdec.ScanFilmBipedAimOnly`). A zero alors que des episodes
	//	               existent : c est le DECODEUR qui n a rien lu — grammaire d en-tete qui a
	//	               bouge, ou bande de slots vide —, pas le film qui serait muet. Ordre de
	//	               grandeur mesure : 4 832 a 24 050 par film (5 films, lot V11).
	//	RidesWithAim   episodes portant au moins une lecture. Le denominateur est `Rides` ; la
	//	               mesure V11 rend 35/35 sur les episodes attestes par la sortie.
	//	AimSamples     points PUBLIES (un par frame au plus, cf. `vehicleRideAimOf`).
	//	AimRideFrames  frames couvertes par les episodes, toutes vies confondues. C est le
	//	               DENOMINATEUR de `AimSamples` : sans lui, « 4 120 points de visee » ne dit
	//	               pas si la serie est continue ou trouee.
	AimReads      int `json:"aimReads"`
	RidesWithAim  int `json:"ridesWithAim"`
	AimSamples    int `json:"aimSamples"`
	AimRideFrames int `json:"aimRideFrames"`
	// Ambiguous compte les vies portant DEUX episodes qui se chevauchent : conducteur et passager
	// ne se departagent pas par la geometrie. Publie, jamais cache (regle du lot V1).
	Ambiguous int `json:"ambiguous"`
	// Shots / ShotsAmbiguous / ShotsUnplaced / ShotsNoRide sont la SECONDE PORTE DES TIRS
	// (`vehicle_shots.go`) : les tirs des joueurs EMBARQUES, que la porte du bipede ecarte
	// puisqu un occupant attache ne replique plus sa position.
	//
	// LEUR DENOMINATEUR EST `ShotsNoRide` : les orphelins qu AUCUN episode ne couvre. Sans lui,
	// « N tirs en vehicule » ne se juge pas — on ne saurait pas si la porte en a laisse dix ou
	// mille. `Shots` compte les tirs POSES (donc publies dans `shots` avec leur marqueur `v`),
	// `ShotsAmbiguous` ceux que DEUX vehicules distincts se disputent au meme instant (artefact
	// du pont, jamais tranche), `ShotsUnplaced` ceux dont la vie de vehicule n avait ni
	// echantillon ni naissance ou poser le tir.
	Shots          int `json:"shots"`
	ShotsAmbiguous int `json:"shotsAmbiguous"`
	ShotsUnplaced  int `json:"shotsUnplaced"`
	ShotsNoRide    int `json:"shotsNoRide"`
	// ShotsVehicleWeapon compte, PARMI `Shots`, ceux dont l arme n est PAS de la famille
	// personnelle (cf. `vehicleWeaponLowHalf`) — une arme qu on ne porte pas a pied.
	//
	// C EST LE TEMOIN DU CALQUE, et il est independant de tout ce qui a servi a rattacher : ni
	// l instant, ni l identite, ni la geometrie n entrent dans la lecture d un identifiant
	// d arme. Un rapport `ShotsVehicleWeapon / Shots` qui s effondrerait dirait que la seconde
	// porte s est mise a ramasser des tirs a pied. Le reste (`Shots - ShotsVehicleWeapon`) n est
	// PAS du bruit par construction : un passager tire son propre fusil depuis le vehicule.
	ShotsVehicleWeapon int `json:"shotsVehicleWeapon"`
}

// tallyVehicleCoverage compte, sur les vies PUBLIEES, ce que la couverture annonce. Un compteur
// qui se remplirait ailleurs qu ici finirait par diverger du tableau qu il decrit.
func tallyVehicleCoverage(tracks []VehicleTrack, cov *VehicleCoverage) {
	cov.Published = len(tracks)
	for _, tr := range tracks {
		if tr.Spawn != nil {
			cov.WithSpawn++
		}
		if tr.Chassis != "" {
			cov.WithChassis++
			if tr.Family != "" {
				cov.FamilyResolved++
			} else {
				cov.FamilyUnknown++
				cov.UnknownChassis[tr.Chassis]++
			}
		}
		cov.Samples += len(tr.Samples)
		for _, s := range tr.Samples {
			if s.H != 0 {
				cov.WithHeading++
			}
		}
		tallyVehicleRides(tr.Rides, cov)
	}
}

// tallyVehicleRides compte les episodes d UNE vie et releve leurs chevauchements.
func tallyVehicleRides(rides []VehicleRide, cov *VehicleCoverage) {
	if len(rides) == 0 {
		return
	}
	cov.VehiclesRidden++
	cov.Rides += len(rides)
	for i, r := range rides {
		if r.XUID != "" {
			cov.RidesNamed++
		}
		if r.Seat != nil {
			cov.RidesWithSeat++
		}
		// LA FENETRE EST INCLUSIVE aux deux bouts (`T0` et `T1` sont deux frames affichees) :
		// un episode d une seule frame en couvre UNE, pas zero.
		cov.AimRideFrames += r.T1 - r.T0 + 1
		if len(r.Aim) > 0 {
			cov.RidesWithAim++
			cov.AimSamples += len(r.Aim)
		}
		switch r.Src {
		case VehicleRideSrcEvent:
			cov.RidesFromEvent++
		case VehicleRideSrcMixed:
			cov.RidesMixed++
		default:
			cov.RidesFromGap++
		}
		// Les episodes d une vie sont TRIES par T0 : un chevauchement se voit sur le voisin.
		if i > 0 && r.T0 <= rides[i-1].T1 {
			cov.Ambiguous++
		}
	}
}

// logVehicleCoverage journalise le calque avec les MEMES denominateurs que l artefact.
//
// LE SILENCE QU IL FAUT ROMPRE : des vies publiees dont AUCUNE ne resout de famille n est pas
// « un film sans vehicule reconnaissable », c est une lecture qui a echoue en bloc — largeurs du
// bloc MPP non reinstallees, ou grammaire du default-state qui a bouge. Sans ce warn, un film
// entier sortirait avec zero sprite sans que rien ne le signale.
func logVehicleCoverage(c *VehicleCoverage) {
	if c == nil {
		return
	}
	slog.Info("rejeu : vehicules",
		"balaye", c.Scanned, "viesRecensees", c.Lives, "publiees", c.Published,
		"relaisFusionnes", c.Merged,
		"sansPosition", c.NoPosition, "avecNaissance", c.WithSpawn, "avecChassis", c.WithChassis,
		"famillesResolues", c.FamilyResolved, "famillesInconnues", c.FamilyUnknown,
		"echantillons", c.Samples, "avecCap", c.WithHeading)
	slog.Info("rejeu : occupation des vehicules",
		"episodes", c.Rides, "vehiculesOccupes", c.VehiclesRidden, "occupantsNommes", c.RidesNamed,
		"bornesParEvenement", c.RidesFromEvent, "bornesMixtes", c.RidesMixed,
		"bornesParTrou", c.RidesFromGap, "avecSiege", c.RidesWithSeat, "ambigus", c.Ambiguous,
		"lecturesDeViseeBrutes", c.AimReads, "episodesAvecVisee", c.RidesWithAim,
		"pointsDeVisee", c.AimSamples, "framesDEpisode", c.AimRideFrames)
	// LE SILENCE QU IL FAUT ROMPRE, et il est le pendant exact du warn de `logVehicleCoverage` :
	// des episodes publies dont AUCUN ne porte de visee n est pas « un film ou personne ne
	// regardait », c est le balayage `i21` sans position qui n a rien rendu. La mesure V11 rend
	// 35 episodes attestes sur 35 porteurs d au moins une lecture, sur 5 films.
	if c.Rides > 0 && c.RidesWithAim == 0 {
		slog.Warn("rejeu : AUCUN episode d occupation ne porte de visee alors que des episodes"+
			" existent — le balayage des records de visee SANS position n a rien rendu, le cone"+
			" retombe partout sur le cap du chassis",
			"episodes", c.Rides, "lecturesBrutes", c.AimReads)
	}
	for id, n := range c.UnknownChassis {
		slog.Info("rejeu : chassis de vehicule NON RESOLU — vies publiees sans sprite",
			"chassis", id, "vies", n)
	}
	if c.WithChassis > 0 && c.FamilyResolved == 0 {
		slog.Warn("rejeu : AUCUN chassis de vehicule resolu alors que le mot d identite a ete lu"+
			" — table de familles a completer, ou lecture du bloc MPP a verifier",
			"chassisLus", c.WithChassis)
	}
}
