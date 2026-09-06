package domain

// tactical_page.go — CE QUE LA PAGE TACTIQUE PUBLIE : la grille des cartes et la lecture
// d'une carte.
//
// Extrait de tactical.go le 2026-09-06 (phase 7). Ce fichier-la avait franchi le seuil de
// 500 lignes en phase 4 bis, et la phase 7 l'y enfoncait : la regle du depot est de ne pas
// accroitre une dette gelee (CLAUDE.md n 5). La coupure suit la frontiere que le fichier
// declarait deja par un commentaire de section — CE QU'ON DEMANDE au lecteur d'un cote, CE
// QU'ON PUBLIE de l'autre.

// TacticalMapCard est une carte de l'ecran d'entree : la ligne du lecteur, plus
// le verdict de lisibilite.
type TacticalMapCard struct {
	MapID     string `json:"map_id"`
	MapName   string `json:"map_name"`
	MapNameFR string `json:"map_name_fr"`

	Matchs    int `json:"matchs"`
	Victoires int `json:"victoires"`
	Defaites  int `json:"defaites"`

	// SousPlancher : la carte compte moins de matchs que le plancher par carte.
	// Elle reste affichee (le joueur doit voir qu'il y a joue) mais desaturee et
	// non ouvrable — une lecture de placement sur trois matchs est du bruit.
	SousPlancher bool `json:"sous_plancher"`
}

// TacticalMapsPage est la reponse de l'ecran d'entree.
type TacticalMapsPage struct {
	Cartes []TacticalMapCard `json:"cartes"`

	// PlancherMatchs est le seuil qui a decide de `SousPlancher`. Publie parce que
	// l'ecran doit pouvoir le NOMMER a l'utilisateur, pas le recopier.
	PlancherMatchs int `json:"plancher_matchs"`
}

// TacticalRaster est la reponse d'une lecture de placement sur une carte.
type TacticalRaster struct {
	MapID    string `json:"map_id"`
	Question string `json:"question"`
	Qui      string `json:"qui"`

	// MatchsFiltres est le nombre de matchs du perimetre joues SUR CETTE CARTE :
	// l'univers du lecteur, mesures ET non mesures (liste blanche x carte x
	// composition, exclusion Campagne comprise). Publie pour que le pied de carte
	// puisse dire « N mesures sur M » — sans lui, l'ecart entre ce que le joueur a
	// joue ici et ce que la carte peut montrer serait invisible.
	//
	// LES DEUX GRANDEURS SE COMPARENT, ET C'EST VOULU (decision superviseur du
	// 2026-09-06, apres un aller-retour) : MatchsRetenus est un SOUS-ENSEMBLE de
	// MatchsFiltres. Une version intermediaire y avait mis la taille de la liste
	// blanche recue — toutes cartes confondues —, ce qui donnait deux grandeurs sans
	// denominateur commun sous des noms qui invitaient a en faire un rapport.
	// SOUS UN FILTRE `spawn`, C'EST L'UNIVERS DEJA RESTREINT : les matchs dont la premiere
	// vie du joueur part de la grappe demandee. Ils sont mesures par construction — un
	// spawn de depart ne se connait que par un sidecar.
	MatchsFiltres int `json:"matchs_filtres"`

	// MatchsRetenus est le DENOMINATEUR de la lecture : les matchs du filtre dont le
	// journal des morts est LISIBLE (cf. TacticalMatch.Mesure). Un match jamais
	// decode n'y entre pas — il ne peut alimenter aucune cellule, et l'y compter
	// ferait varier l'intensite avec la couverture de film au lieu du jeu
	// (correction G2, 2026-09-06). Publie AVEC les cellules : une intensite sans son
	// denominateur ne se compare pas d'un filtre a l'autre.
	//
	// ⚠ CE N'EST LE DENOMINATEUR DIRECT QUE DE LA LECTURE NON SIGNEE. La lecture
	// signee normalise CHAQUE COTE par le sien (occV/nbV - occD/nbD, cf.
	// analysis/tactical.CellulesSignees) : ses deux denominateurs sont
	// MatchsVictoire et MatchsDefaite ci-dessous, et leur somme est en general
	// INFERIEURE a MatchsRetenus (les nuls et les matchs de resultat inconnu
	// comptent dans l'univers, dans aucun des deux cotes).
	MatchsRetenus int `json:"matchs_retenus"`

	// MatchsVictoire et MatchsDefaite sont les deux denominateurs de la lecture
	// SIGNEE, sur l'univers entier. Nuls sur une lecture non signee, ou ils
	// n'auraient aucun role. Publies parce que le pied de carte doit pouvoir dire
	// sur quoi la difference est calculee — « 12 victoires contre 8 defaites » — au
	// lieu de laisser croire que c'est MatchsRetenus des deux cotes.
	MatchsVictoire int `json:"matchs_victoire"`
	MatchsDefaite  int `json:"matchs_defaite"`

	// PasM est le pas de la grille en metres, et Bornes le rectangle englobant les
	// cellules LISIBLES — le cadre que le peintre doit couvrir.
	PasM   float64     `json:"pas_m"`
	Bornes BornesMonde `json:"bornes"`

	Cellules []CelluleTactique `json:"cellules"`
	Echelle  EchelleTactique   `json:"echelle"`

	// PointsIgnores : les positions ecartees faute de coordonnees finies. Publie
	// plutot qu'avale — un decodage qui derape se voit ici.
	PointsIgnores int `json:"points_ignores"`

	// EvenementsJournal et EvenementsLocalises disent CE QUE LA CARTE NE MONTRE PAS
	// (ajout 2026-09-06) : combien d'evenements de la cible le journal des morts
	// compte sur l'univers (morts pour « ou je meurs », kills pour « ou je tue »,
	// les deux pour « ou je gagne »), et combien d'entre eux ont une position
	// mesuree. Le pied de carte les rend en clair — « N morts, M localisees ».
	//
	// POURQUOI C'EST OBLIGATOIRE. Une position n'existe que si le producteur a su
	// resoudre les deux identites et si l'instant n'etait pas ambigu (double kill) ;
	// une carte muette sur un pan entier de la partie ressemble sinon a un pan de
	// terrain ou il ne se passe rien. L'ecart est une PROPRIETE DE LA MESURE, pas
	// un detail d'implementation.
	//
	// EvenementsJournal vaut 0 quand le journal n'a pas pu etre lu : le pied de
	// carte doit alors taire la couverture plutot qu'annoncer 0 sur M.
	EvenementsJournal   int `json:"evenements_journal"`
	EvenementsLocalises int `json:"evenements_localises"`

	// Grappes sont les amas de REAPPARITION du joueur sur cette carte, calcules a la
	// lecture depuis les spawns de depart des sidecars. Servis avec toutes les lectures
	// d'artefact : ce sont eux que le filtre `spawn` designe.
	Grappes []TacticalGrappe `json:"grappes,omitempty"`

	// MatchsSansRayon : les matchs ECARTES de la lecture « isole » parce que leur variante
	// n'a pas de portee de radar mesuree. Publie plutot qu'avale — sans lui, une lecture
	// amputee ressemblerait a une lecture complete.
	MatchsSansRayon int `json:"matchs_sans_rayon,omitempty"`

	// MortsIndeterminees : les morts ECARTEES de la lecture « isole » parce qu'au moins un
	// coequipier etait INVISIBLE a cet instant (en vehicule non attribue, ou survivant
	// anonyme) et qu'aucun coequipier vu n'etait a portee. Ni isolees ni accompagnees :
	// les compter isolees rendait des morts « seules » a trois metres d'un coequipier.
	MortsIndeterminees int `json:"morts_indeterminees,omitempty"`

	// MortsPositionInconnue : les morts dont le film ne dit pas OU elles ont eu lieu
	// (embarquement sans point de vehicule). Ni peintes ni examinees.
	MortsPositionInconnue int `json:"morts_position_inconnue,omitempty"`

	// Isolement est la part des morts SANS coequipier vivant a portee, sous la forme
	// canonique (taux + brut + par match + N + echantillon faible). nil hors de la lecture
	// « isole ».
	Isolement *Couverture `json:"isolement,omitempty"`

	// Echange est le taux de morts vengees de mon equipe SUR CETTE CARTE. nil quand
	// le titre ne sait pas lire la source des morts (capability `film.kill_source`
	// absente) : la lecture de placement reste servie, le KPI est simplement
	// silencieux — jamais un zero, qui se lirait comme une contre-performance.
	Echange *Couverture `json:"echange,omitempty"`
}
