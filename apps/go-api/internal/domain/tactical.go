package domain

import "errors"

// Les trois refus de l'onglet Tactique. Ils sont dans `domain` parce que le
// handler doit les TRADUIRE en statut HTTP (404 / 400) sans dependre du service.
var (
	// ErrTacticalCarteInconnue : aucune carte de ce nom dans les matchs retenus du
	// joueur. Ce n'est pas une carte vide, c'est une carte qu'il n'a pas jouee
	// SOUS CE FILTRE — 404, jamais une lecture a zero cellule qui se lirait comme
	// « rien ne s'y passe ».
	ErrTacticalCarteInconnue = errors.New("tactique: carte inconnue pour ce joueur")
	// ErrTacticalQuestionInconnue : question hors du vocabulaire servi.
	ErrTacticalQuestionInconnue = errors.New("tactique: question inconnue")
	// ErrTacticalQuiInconnu : axe « qui » hors du vocabulaire servi.
	ErrTacticalQuiInconnu = errors.New("tactique: axe qui inconnu")
	// ErrTacticalEscouadeSansComposition : l'axe `escouade` a ete demande sans
	// composition. Depuis le 2026-09-06 (arbitrage utilisateur), « Escouade » designe
	// LA COMPOSITION CHOISIE, pas « mes coequipiers du match » : sans elle, l'axe n'a
	// aucun contenu. Retomber en silence sur les coequipiers du match repondrait a une
	// AUTRE question que celle posee — 400, et le client ne propose pas l'axe.
	ErrTacticalEscouadeSansComposition = errors.New("tactique: axe escouade demande sans composition")
)

// Types de l'onglet Tactique — lectures de PLACEMENT agregees par carte (ou je meurs, ou
// je tue, ou je gagne). Structs purs : aucune I/O, aucun SQL, aucune dependance au
// document de rejeu.
//
// FRONTIERE VOULUE : `internal/analysis/tactical` ne connait QUE ces types. Il n'importe
// ni `analysis/replay` (le document de rejeu) ni `platform/duckdb` — c'est l'appelant qui
// PROJETTE ce qu'il a (positions de `kill_positions`, points des pistes d'un artefact)
// vers `PositionSample`. Le rasterisage reste donc consommable sans artefact.

// PositionSample est l'entree minimale du rasterisage : un point du monde, en METRES, et
// le match qui l'a produit.
//
// Le match est porte par le point (et non par l'appel) parce que le plancher de rarete se
// compte en MATCHS DISTINCTS par cellule : sans l'identifiant, un joueur immobile gonfle
// une cellule sans rien prouver de plus (mesure de cmd/mappos-build, 2026-08-30).
//
// Z est volontairement absent : toutes les lectures de l'onglet sont des vues du dessus.
type PositionSample struct {
	MatchID string
	X, Y    float64
}

// BornesMonde est le rectangle englobant d'une lecture, en metres monde. `Valide` est faux
// tant qu'aucun point n'a ete vu : un rectangle vide n'est pas un rectangle a l'origine.
type BornesMonde struct {
	MinX float64 `json:"min_x"`
	MinY float64 `json:"min_y"`
	MaxX float64 `json:"max_x"`
	MaxY float64 `json:"max_y"`

	Valide bool `json:"valide"`
}

// CelluleTactique est une cellule ALIMENTEE d'une lecture agregee. Une cellule jamais
// atteinte n'existe pas : elle n'apparait dans aucune liste (decision produit 2026-09-05,
// « cellule jamais atteinte = VIDE, jamais peinte en froid »).
type CelluleTactique struct {
	// Col, Lig : l'adresse entiere de la cellule sur la grille, ancree sur l'ORIGINE DU
	// MONDE (et non sur les bornes de la lecture) — deux lectures filtrees differemment
	// nomment donc la meme cellule pareil, et deux rasters de matchs differents se
	// somment sans re-projection.
	Col int `json:"col"`
	Lig int `json:"lig"`

	// CentreX, CentreY : le centre de la cellule en metres monde, pour le peintre.
	CentreX float64 `json:"centre_x"`
	CentreY float64 `json:"centre_y"`

	// Valeur est la valeur PAR MATCH (decision produit : jamais un cumul brut, qui ne se
	// compare pas d'un filtre a l'autre). Lecture simple : occurrences / nombre de matchs
	// retenus. Lecture signee : taux du cote victoire moins taux du cote defaite.
	Valeur float64 `json:"valeur"`

	// Brut est le cumul non normalise qui a produit `Valeur` — servi AVEC elle, jamais a
	// sa place (doctrine « jamais un taux seul »).
	Brut float64 `json:"brut"`

	// Matchs est le nombre de matchs DISTINCTS ayant alimente la cellule ; MatchsVictoire
	// et MatchsDefaite le detaillent par cote (non nuls seulement en lecture signee).
	Matchs         int `json:"matchs"`
	MatchsVictoire int `json:"matchs_victoire"`
	MatchsDefaite  int `json:"matchs_defaite"`
}

// EchelleTactique porte les reperes de coloration d'une lecture. Les quantiles sont
// calcules sur les cellules ALIMENTEES uniquement : inclure des zeros implicites
// ecraserait toute la dynamique vers le bas.
type EchelleTactique struct {
	// P50, P95 : les quantiles des valeurs des cellules. En lecture signee, ils portent
	// sur la VALEUR ABSOLUE — un quantile sur le signe n'a pas de sens (il depend de la
	// proportion de cellules favorables, pas de l'intensite).
	P50 float64 `json:"p50"`
	P95 float64 `json:"p95"`

	// Borne est le haut de l'echelle. En lecture signee, l'echelle est SYMETRIQUE et va
	// de -Borne a +Borne : sans cela, un cote parait plus intense que l'autre a valeur
	// egale.
	Borne float64 `json:"borne"`

	// Symetrique dit laquelle des deux lectures ci-dessus s'applique.
	Symetrique bool `json:"symetrique"`

	// NCellules est le nombre de cellules ayant servi au calcul.
	NCellules int `json:"n_cellules"`
}

// ---------------------------------------------------------------------------
// Lecture tactique — ce qu'un lecteur de base rend, et ce qu'une page en publie.
//
// Les types ci-dessous vivent ICI et pas dans `internal/analysis/tactical` :
// ils traversent la frontiere port -> service -> handler, et un DTO de reponse
// est un type de `domain`, jamais un type d'`analysis` (un algo qui exporte sa
// forme de sortie fige son appelant sur son implementation).
// ---------------------------------------------------------------------------

// ListeBlancheMatchs est le PERIMETRE de matchs d'une lecture tactique.
//
// POURQUOI UN TYPE ET PAS UN `[]string` (phase 4 bis, 2026-09-06). Deux appelants
// ont des besoins OPPOSES sur la meme absence de valeur :
//
//	l'onglet Tactique   passe la liste resolue par service.FilteredMatchIDs — et une
//	                    liste VIDE veut dire AUCUN MATCH (le filtre n'a rien retenu),
//	                    jamais « tous » ;
//	la page Escouade    ne passe AUCUNE liste : elle lit le journal des morts sur tout
//	                    l'historique du joueur, puis resserre en Go.
//
// Avec un `[]string` nu, ces deux etats sont le meme `len() == 0` — et le jour ou un
// appelant oublie sa liste, il obtient l'historique ENTIER en silence. Le zero-value
// de ce type-ci est l'absence de restriction (le seul etat qu'on peut construire par
// accident) et TOUTE liste, vide comprise, vient de RestreindreAux.
type ListeBlancheMatchs struct {
	restreint bool
	ids       []string
}

// RestreindreAux borne une lecture aux match_id donnes. `ids` VIDE = aucun match.
func RestreindreAux(ids []string) ListeBlancheMatchs {
	return ListeBlancheMatchs{restreint: true, ids: ids}
}

// Restreint dit si une liste blanche a ete posee (fut-elle vide).
func (l ListeBlancheMatchs) Restreint() bool { return l.restreint }

// IDs rend les match_id de la liste blanche (nil si aucune n'a ete posee).
func (l ListeBlancheMatchs) IDs() []string { return l.ids }

// TacticalQuery est la demande adressee au lecteur tactique.
//
// LE PERIMETRE EST UNE LISTE BLANCHE DE match_id, pas un jeu d'axes de filtre
// (phase 4 bis, 2026-09-06). Les axes de l'Explorateur — periode, sessions epinglees,
// contexte solo/escouade, cascade — sont resolus EN AMONT par
// `service.FilteredMatchIDs`, sur la base JOUEUR, qui est la seule a porter les
// sessions. Le lecteur, lui, ne connait que des identifiants. C'est ce qui fait
// MARCHER le filtre de session sur cet onglet (arbitrage utilisateur du 2026-09-06),
// la ou un `MatchFilterSpec` le rangeait dans les filtres IGNORES.
type TacticalQuery struct {
	// PlayerXUID est le joueur dont on lit les matchs. L'univers est TOUJOURS le
	// sien : la portee « tout le monde » n'existe pas en V1.
	PlayerXUID string

	// MapID restreint a une carte. Vide = toutes les cartes — c'est le cas de
	// l'ecran d'entree (MapsPlayed) ET du journal des morts lu par la page
	// Escouade (KillEvents), qui mesure l'echange d'une COMPOSITION et non d'une
	// carte. Seule la lecture SPATIALE (KillPositions) l'exige : une grille de
	// 0,5 m n'a de sens que carte par carte.
	MapID string

	// Matchs est la liste blanche du perimetre (cf. ListeBlancheMatchs).
	Matchs ListeBlancheMatchs

	// Coequipiers restreint aux matchs ou TOUS ces xuids etaient dans MON equipe —
	// la COMPOSITION choisie dans la barre de filtres, meme notion que la page
	// Escouade. Vide = aucune contrainte de composition.
	Coequipiers []string
}

// TacticalScope est le perimetre demande par la PAGE : les match_id que le client
// a fait resoudre, et la composition choisie. Un struct plutot que deux `[]string`
// adjacents — deux listes de chaines de suite s'inversent sans que le compilateur
// le voie.
type TacticalScope struct {
	// MatchIDs : la liste blanche resolue par le client. VIDE = aucun match.
	MatchIDs []string
	// Coequipiers : les xuids de la composition (0 a 3). Vide = pas de composition.
	Coequipiers []string
}

// TacticalRasterRequest est la demande d'une lecture de placement.
type TacticalRasterRequest struct {
	MapID    string
	Question string
	Qui      string
	Scope    TacticalScope
}

// TacticalMatch est un match RETENU par le filtre : l'unite de l'univers.
type TacticalMatch struct {
	MatchID string

	// Mesure dit que le journal des morts de ce match est LISIBLE (au moins une
	// ligne publiable dans match_kill_events_latest).
	//
	// UN MATCH NON MESURE N'EST PAS UN MATCH A ZERO MORT, c'est un match ILLISIBLE :
	// son film n'a jamais ete decode, ou il a EXPIRE cote serveur. Il ne peut
	// alimenter aucun numerateur ; le compter au denominateur « par match » ferait
	// varier la grandeur avec la COUVERTURE DE FILM au lieu du jeu — deux filtres a
	// couverture differente (20 matchs sur 20 decodes contre 2 sur 20) rendraient
	// 0,20 et 0,02 pour exactement le meme jeu. C'est le pendant du defaut P0 de la
	// phase 1 : le zero LEGITIME compte au denominateur, l'ILLISIBLE est compte a
	// part (correction G2, revue du 2026-09-06).
	Mesure bool

	// Outcome porte OutcomeWin / OutcomeLoss / OutcomeDraw / OutcomeDNF, ou
	// OutcomeUnknown quand le substrat ne le sait pas. Un resultat inconnu compte
	// au denominateur « par match » et dans aucun des deux cotes de la lecture
	// signee (cf. analysis/tactical.RasteriseAvecResultats).
	Outcome int
}

// TacticalUnivers est L'ENSEMBLE DES MATCHS RETENUS par le filtre, avec la
// composition des equipes de chacun.
//
// POURQUOI IL VOYAGE AVEC LES POINTS ET N'EN EST JAMAIS DEDUIT : un match retenu
// peut n'avoir AUCUN point sur la lecture courante (aucune mort, aucun kill), et
// c'est un zero legitime qui doit compter au denominateur. Le deduire des points
// l'effacerait — c'est le defaut corrige en phase 1 (12 victoires dont 2 muettes
// lues +0,10 au lieu de 0,00).
type TacticalUnivers struct {
	Matchs []TacticalMatch

	// Equipes : matchID -> xuid -> numero d'equipe. La composition change d'un
	// match a l'autre ; une table globale melangerait deux compositions au premier
	// joueur ayant change de camp.
	Equipes EquipesParMatch
}

// TacticalKillPosition est UNE mort mesuree : la position du tueur et celle de la
// victime, en metres monde, avec les deux identites.
//
// Z est absent : toutes les lectures de l'onglet sont des vues du dessus. Une
// ligne n'existe que si les DEUX positions sont connues — une position partielle
// n'est jamais approchee (meme prudence que KillDistanceRepo).
type TacticalKillPosition struct {
	MatchID string

	// KillerXUID et VictimXUID peuvent etre VIDES, et ce sont deux cas TRES
	// inegalement probables — la doc d'origine les mettait sur le meme plan, a tort.
	//
	//	KillerXUID vide  N'ARRIVE PAS en sortie de KillPositions. La jointure est une
	//	                 EGALITE sur `kill_positions.killer_xuid`, et le persister
	//	                 REFUSE toute ligne sans tueur (persist/kill_position_persister.go).
	//	                 Le champ reste defensif : un scan qui rendrait une chaine vide
	//	                 ne doit pas etre range dans un axe par accident.
	//	VictimXUID vide  ARRIVE. Le collecteur du film n'ecrit de position que pour les
	//	                 morts dont LES DEUX identites sont resolues
	//	                 (sync/killcollector/positions.go, killRefsFromDeaths), mais le
	//	                 producteur NATIF de Halo 5 ne pose que le tueur
	//	                 (games/halo_5/ingest/positions.go) : sa ligne peut donc joindre
	//	                 un kill-event dont la victime est un BOT (victim_xuid NULL).
	//
	// Dans les deux cas, une identite vide n'appartient a AUCUN axe « qui » : elle n'a
	// pas d'equipe, et lui en deviner une serait une invention.
	KillerXUID string
	VictimXUID string

	KillerX, KillerY float64
	VictimX, VictimY float64
}

// TacticalPositions : l'univers ET les positions mesurees de ses matchs.
type TacticalPositions struct {
	Univers TacticalUnivers
	Points  []TacticalKillPosition
}

// TacticalKillEvents : l'univers ET le journal des morts de ses matchs, sous la
// forme que `analysis/coordination` consomme.
type TacticalKillEvents struct {
	Univers TacticalUnivers
	Events  []KillEvent
}

// TacticalMapRow est une carte JOUEE, telle que le lecteur la rend : le compte de
// matchs et sa decomposition en victoires / defaites. Le plancher de lisibilite
// est pose par le service, pas par le lecteur — une regle produit n'appartient pas
// a une requete SQL.
type TacticalMapRow struct {
	MapID     string
	MapName   string
	MapNameFR string

	Matchs    int
	Victoires int
	Defaites  int
}

// ---------------------------------------------------------------------------
// Ce que la page publie
// ---------------------------------------------------------------------------

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

	// MatchsFiltres est le nombre de match_id que le FILTRE a retenus — la taille de
	// la liste blanche recue, TOUTES CARTES CONFONDUES (phase 4 bis, 2026-09-06).
	// C'est la definition du perimetre cote client : ce que la barre de filtres a
	// selectionne, avant meme de regarder cette carte-ci.
	//
	// ⚠ IL NE SE LIT PAS « M matchs de cette carte ». L'intersection avec la carte
	// (et avec la composition) est faite ensuite ; le denominateur de la lecture,
	// c'est MatchsRetenus ci-dessous. Le pied de carte doit donc dire les deux
	// grandeurs pour ce qu'elles sont — « N matchs mesures sur cette carte, sur M
	// matchs filtres » — et jamais les presenter comme un rapport.
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

	// Echange est le taux de morts vengees de mon equipe SUR CETTE CARTE. nil quand
	// le titre ne sait pas lire la source des morts (capability `film.kill_source`
	// absente) : la lecture de placement reste servie, le KPI est simplement
	// silencieux — jamais un zero, qui se lirait comme une contre-performance.
	Echange *Couverture `json:"echange,omitempty"`
}

// ---------------------------------------------------------------------------
// Vocabulaire et bornes de l'onglet
// ---------------------------------------------------------------------------

// Les QUESTIONS servies par la phase 2. Elles se lisent toutes sur le meme
// substrat — les positions mesurees de `kill_positions` — parce que c'est le seul
// qui existe sans artefact de rejeu. Les questions d'occupation (ou je passe mon
// temps, mes routes de spawn) arrivent avec les rasters de la cuisson.
const (
	// TacticalQuestionMorts : ou JE MEURS — la position de la VICTIME.
	TacticalQuestionMorts = "morts"
	// TacticalQuestionKills : ou JE TUE — la position du TUEUR.
	TacticalQuestionKills = "kills"
	// TacticalQuestionGagne : ou JE GAGNE — lecture SIGNEE, sur les ENGAGEMENTS
	// (mes kills ET mes morts), chaque cote ramene a son propre nombre de matchs.
	//
	// POURQUOI LES ENGAGEMENTS ET PAS LES SEULS KILLS : la question demande ou ma
	// presence correle avec la victoire. La seule presence mesurable avant les
	// rasters d'occupation est le combat, et il a DEUX faces — ne garder que les
	// kills confondrait « ou je gagne » avec « ou je tue », qui est deja une
	// question a part. Substitution prevue : l'occupation, quand elle existera.
	TacticalQuestionGagne = "gagne"
)

// L'axe QUI.
//
// `escouade` designe LA COMPOSITION CHOISIE dans la barre de filtres — les xuids que
// l'utilisateur a nommes, et eux seuls (arbitrage utilisateur du 2026-09-06, qui
// REMPLACE « mes coequipiers du match »). Le perimetre de matchs garantit deja que ces
// joueurs etaient dans mon equipe (cf. TacticalQuery.Coequipiers) ; sans composition,
// l'axe est REFUSE (ErrTacticalEscouadeSansComposition) plutot que redefini en douce.
//
// `adv` reste l'autre equipe DU MATCH : elle change a chaque partie et ne se nomme pas.
const (
	TacticalQuiMoi         = "moi"
	TacticalQuiEscouade    = "escouade"
	TacticalQuiAdversaires = "adv"
)

// PlancherMatchsParCarte est le nombre de matchs en dessous duquel une carte
// n'est pas ouvrable : 10 (plan tactique, 2026-09-05). Une lecture de placement
// sur trois matchs ne mesure pas un placement, elle mesure trois parties.
const PlancherMatchsParCarte = 10
