package domain

// Package domain — squad_echange.go : L'ECHANGE SUR LA PAGE ESCOUADE.
//
// UN ECHANGE est une mort vengee : un coequipier abat le tueur dans les 5 secondes
// (analysis/coordination.FenetreEchangeMs, valeur arretee par l'utilisateur le 2026-09-05).
// Ces types sont la forme sous laquelle la mesure entre dans le `pageData` de la page
// Escouade — une section de plus, servie par le meme appel que les autres.
//
// TROIS PERIMETRES COEXISTENT ICI, ET ILS NE SE CONFONDENT PAS :
//
//	MON CAMP     le denominateur du KPI : moi ET mes coequipiers DU MATCH. Decision
//	             produit de l'utilisateur (2026-09-06) — c'est le denominateur le moins
//	             biaise, et le meme que celui de l'onglet Tactique. Un taux sur mes seules
//	             morts dirait « on me venge », pas « on echange ».
//	LE ROSTER    les axes de la MATRICE : le joueur principal et les coequipiers
//	             SELECTIONNES, les seuls que la page sait nommer. Un vengeur de passage
//	             compte au KPI (il est de mon camp) et n'a pas de ligne dans la matrice —
//	             afficher un xuid nu serait pire que l'ecarter (doctrine
//	             SquadAssistPairsTable).
//	L'HABITUEL   la reference : la MEME mesure sur tout l'historique de la composition,
//	             filtres de la page mis a part. C'est la baseline du briefing de
//	             l'Explorateur (le scope est toujours un sous-ensemble de l'habituel).

// SquadEchangeJoueur nomme un axe de la matrice. L'ordre de la tranche est l'ordre du
// roster (joueur principal d'abord), le meme que celui des autres blocs de la page.
type SquadEchangeJoueur struct {
	XUID     string `json:"xuid"`
	Gamertag string `json:"gamertag"`
}

// SquadEchangeCell est UNE case de la matrice « qui echange pour qui » : le vengeur en
// LIGNE, le venge en COLONNE — la meme orientation que le tableau « qui assiste qui »
// (Assistant / Beneficiaire), son voisin immediat sur la page.
//
// La diagonale n'existe pas : personne ne se venge soi-meme (une mort dont le vengeur est
// la victime est ecartee par analysis/coordination).
type SquadEchangeCell struct {
	VengeurXUID     string `json:"vengeur_xuid"`
	VengeurGamertag string `json:"vengeur_gamertag"`
	VengeXUID       string `json:"venge_xuid"`
	VengeGamertag   string `json:"venge_gamertag"`

	// Nombre est le compte BRUT d'echanges de ce couple sur le perimetre filtre.
	Nombre int `json:"nombre"`

	// ParMatch est ce meme compte ramene aux matchs MESURES — la grandeur qui se
	// compare d'un filtre a l'autre (doctrine « jamais un compte sans son
	// denominateur »). Ce n'est PAS un taux : c'est une quantite par match, sans
	// borne haute.
	//
	// LE DENOMINATEUR EST `MatchsMesures`, PAS `MatchsTotal` (correction G2,
	// 2026-09-06). Le numerateur ne peut venir que des matchs dont le journal des
	// morts est lisible ; diviser par tous les matchs du filtre ferait varier la
	// grandeur avec la COUVERTURE DE FILM au lieu du jeu — deux filtres a 20/20 et
	// a 2/20 matchs decodes rendraient 0,20 et 0,02 pour exactement le meme jeu.
	ParMatch float64 `json:"par_match"`
}

// SquadEchangeBucket est un intervalle de la distribution des delais, PRE-BINNE COTE
// SERVEUR (ADR 0010) : le client peint des barres, il ne decide pas des bornes.
//
// Les cinq premiers intervalles couvrent la fenetre (0-1, 1-2, 2-3, 3-4, 4-5 s, la borne
// haute de 5 s COMPRISE). Les deux derniers (5-7 s puis au-dela) sont HORS FENETRE : ils
// sont MONTRES et ne sont COMPTES DANS AUCUN TAUX. Sans eux, la distribution s'arreterait
// net a 5 s sans dire si la fenetre coupe une population dense ou du vide.
type SquadEchangeBucket struct {
	// DebutMs et FinMs bornent l'intervalle en millisecondes. FinMs vaut 0 sur le
	// dernier intervalle, qui est OUVERT (cf. Ouvert).
	DebutMs int64 `json:"debut_ms"`
	FinMs   int64 `json:"fin_ms"`

	// Ouvert : l'intervalle n'a pas de borne haute (« au-dela de N s »).
	Ouvert bool `json:"ouvert"`

	// HorsFenetre : l'intervalle est au-dela de la fenetre d'echange. Ses ripostes sont
	// affichees et n'entrent dans aucun taux.
	HorsFenetre bool `json:"hors_fenetre"`

	// Nombre est le compte de morts de MON CAMP dont la riposte tombe dans cet
	// intervalle.
	Nombre int `json:"nombre"`
}

// SquadEchange est la section « echange » du pageData de la page Escouade.
//
// Absente (nil) quand le titre ne sait pas lire la source des morts
// (games.JournalDesMortsFiable). C'est une OMISSION, jamais des zeros : un taux nul se
// lirait comme une contre-performance quand la verite est « ce titre ne mesure pas ca ».
type SquadEchange struct {
	// Joueurs donne l'ordre des axes de la matrice (roster).
	Joueurs []SquadEchangeJoueur `json:"joueurs"`

	// Cellules ne porte QUE les couples ayant au moins un echange : une case vide de la
	// matrice n'existe pas dans le contrat, le client la peint a zero.
	Cellules []SquadEchangeCell `json:"cellules"`

	// Delais est la distribution complete (fenetre + hors fenetre), toujours servie avec
	// ses sept intervalles, meme a zero — une distribution amputee de ses intervalles
	// vides ne se lit pas.
	Delais []SquadEchangeBucket `json:"delais"`

	// FenetreMs est la fenetre d'echange en millisecondes. Publiee parce que la page doit
	// pouvoir la NOMMER a l'utilisateur, pas la recopier.
	FenetreMs int64 `json:"fenetre_ms"`

	// Couverture est le taux d'echange de MON CAMP sur le perimetre filtre, sous la seule
	// forme dont un taux sort du domaine (taux + brut + par match + N + echantillon
	// faible). Sa quantite PAR MATCH se divise par MatchsMesures (cf. G2), jamais par
	// MatchsTotal.
	Couverture Couverture `json:"couverture"`

	// Habituel est la MEME mesure sur tout l'historique de la composition — la reference
	// dont l'ecart se lit en points. Le perimetre filtre est toujours un SOUS-ENSEMBLE de
	// celui-ci : des comptes de matchs egaux valent « aucun filtre ne retrecit », et
	// l'ecart est alors nul par construction (a masquer cote client).
	Habituel Couverture `json:"habituel"`

	// MatchsHabituel est le nombre de matchs de la reference.
	MatchsHabituel int `json:"matchs_habituel"`

	// MatchsMesures et MatchsTotal sont LE BANDEAU DE COUVERTURE : combien des matchs du
	// perimetre portent un journal des morts lisible, sur combien au total.
	//
	// CE N'EST PAS DECORATIF. Le journal vient du film, et les films Theater EXPIRENT
	// cote serveur : le manque est DEFINITIF, pas un retard. Un taux calcule sur une
	// fraction de la selection SANS dire laquelle serait un chiffre non reproductible.
	MatchsMesures int `json:"matchs_mesures"`
	MatchsTotal   int `json:"matchs_total"`
}
