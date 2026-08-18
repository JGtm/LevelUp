package replay

// document_score.go — LA COURBE DU SCORE, ET CE QU'ELLE GARANTIT.
//
// # CE QUE LE CALQUE APPORTE
//
// Les positions disent ou les joueurs etaient, les actions d'objectif ce qu'ils ont accompli.
// Celui-ci dit OU EN ETAIT LE SCORE a cet instant — pour les deux camps et pour chaque
// joueur, a la milliseconde ou le jeu l'a change, sur la meme grille de frames que le reste
// du document.
//
// # L'ORACLE EST LE SCORE AFFICHE, ET CE N'EST PAS TOUJOURS CELUI DE L'API
//
// Le film porte le score que l'ECRAN montre. `Teams[].Stats.CoreStats.Score` de l'API porte
// autre chose dans deux modes, et c'est mesure (phase 0-ter du lot A, 2026-08-18) :
//
//	Strongholds  l'API compte des TICKS (emissions - 1 : 193/174/132 la ou le film dit 200,
//	             c'est-a-dire le plafond, c'est-a-dire la victoire) ;
//	KOTH         l'API compte des secondes de colline sur 2 films du corpus, des collines
//	             sur les 2 autres.
//
// Sur les 16 films ou l'API EST le score affiche : 16/16 exact, 5 modes sur 5. C'est pourquoi
// `coverage.score.oracle` vaut `displayed` et le dit : la courbe publiee ici ne se compare
// pas terme a terme au `team_0_score` du registre sur ces deux modes-la.
//
// # LES MANCHES SONT PUBLIEES SEPAREMENT, ET LE TOTAL AVEC
//
// Le score de mode REPART DE ZERO a chaque manche (l'en-tete de 5 bits d'un composant est le
// numero de manche, 0-based). Publier la seule derniere valeur donnerait la derniere manche
// pour le match : sur l'Oddball `24dbb67d`, 100/78 au lieu de 200/121. Le document porte donc
// les deux formes — `rounds` (ce que l'ecran affiche PENDANT la manche) et `total` (ce que le
// match retient) — parce qu'aucune ne se deduit de l'autre sans connaitre le decoupage.
//
// # CE QUI N'EST PAS PUBLIE, ET POURQUOI
//
// Les DECREMENTS du score personnel. `self_destruction` et `betrayed_player` valent -100, et
// le filtre de non-decroissance qui rend les manches cumulables les ecarte. Le prix est
// mesure et faible (3 ancrages parasites pour 377 lectures reelles sur le film de reference),
// mais il est reel : un score personnel publie ici ne redescend jamais. Le dire vaut mieux
// que publier une courbe dont personne ne sait si elle peut reculer.

// Identites d'equipe possibles pour `coverage.score.teamIdentity` : la METHODE qui a
// rattache un slot d'entite (6 / 8) a un camp du registre (D3 du plan registre-film).
//
// L'ORDRE EST CELUI DE LA FORCE DE PREUVE, et la troisieme valeur est un refus explicite :
// une courbe `unresolved` est publiee SANS `teamId`, jamais avec un camp devine. Sur une
// carte, l'erreur serait invisible et credible.
const (
	// ScoreIdentityFinal (a) : le score FINAL de chaque slot d'equipe designe un camp sans
	// ambiguite parce que `team_0_score` et `team_1_score` different.
	ScoreIdentityFinal = "a"
	// ScoreIdentityFrags (b) : les scores du registre sont egaux (ou absents) ; c'est alors
	// la somme des FRAGS des joueurs identifies de chaque camp qui designe le slot — le
	// statborg replique le total de frags du camp en `comp 2 A` du slot d'equipe.
	ScoreIdentityFrags = "b"
	// ScoreIdentityUnresolved (c) : ni l'un ni l'autre. Les courbes sortent sans `teamId`.
	ScoreIdentityUnresolved = "unresolved"
)

// ScoreOracleDisplayed nomme l'oracle de la courbe : le score AFFICHE par le jeu.
//
// La valeur est ecrite dans le document plutot que sous-entendue : un client (ou une mesure
// ulterieure) qui compare cette courbe au `team_0_score` du registre doit pouvoir lire, dans
// l'artefact lui-meme, que les deux ne mesurent pas la meme chose en Strongholds et en KOTH.
const ScoreOracleDisplayed = "displayed"

// ScoreTick est un point de courbe : un instant sur l'axe du rejeu et une valeur.
//
// T EST UNE FRAME DU DOCUMENT, comme Point.T, Shot.T et ObjectiveAction.T — l'instant du film
// moins l'origine de la frame 0, divise par le pas de la grille. Les emissions du statborg
// sont datees depuis le PREMIER PAQUET DU FILM, la grille compte depuis le premier paquet de
// POSITION : publier le quotient brut decalerait toute la courbe de 3,6 s a 50,8 s selon le
// match (le meme defaut que le calque d'objectifs a paye, registre `:123`).
type ScoreTick struct {
	T int `json:"t"`
	V int `json:"v"`
}

// ScoreRound est la courbe d'UNE manche : les valeurs propres a la manche, non cumulees.
type ScoreRound struct {
	// Round est le numero de manche, 0-based (0 = premiere manche).
	Round int `json:"round"`
	// Points sont les emissions retenues, aux CHANGEMENTS seulement.
	Points []ScoreTick `json:"points"`
}

// ScoreSeries est un compteur suivi dans le temps, sous ses deux formes.
type ScoreSeries struct {
	// Rounds est la courbe manche par manche (valeurs propres a la manche).
	Rounds []ScoreRound `json:"rounds,omitempty"`
	// Total est la courbe cumulee sur le match : les manches finalisees plus la courante.
	// Croissante de bout en bout, son dernier point vaut le total du match.
	Total []ScoreTick `json:"total,omitempty"`
}

// TeamScore est la courbe de score d'un camp.
type TeamScore struct {
	// TeamID est le camp du registre (0 ou 1). ABSENT quand l'identite n'est pas resolue
	// (`coverage.score.teamIdentity` vaut alors `unresolved`) : la courbe reste publiee, mais
	// le client ne peut pas la colorer — c'est voulu, l'inverse serait une devinette.
	//
	// POINTEUR, PAS int : le camp 0 existe, et `omitempty` sur un entier l'effacerait.
	TeamID *int `json:"teamId,omitempty"`
	// Rounds et Total : cf. ScoreSeries. Ecrits a plat ici parce qu'une equipe n'a qu'un
	// seul compteur — son score de mode.
	Rounds []ScoreRound `json:"rounds,omitempty"`
	Total  []ScoreTick  `json:"total,omitempty"`
}

// PlayerScore porte les compteurs vivants d'un joueur.
//
// LA CLE EST LE XUID, comme partout ailleurs dans le document : le slot d'entite du statborg
// et le slot de biped du rejeu sont DEUX espaces differents, et les confondre poserait les
// compteurs sur les mauvais joueurs sans que rien ne le signale. Un slot que le pont
// d'identite n'a pas apparie sans ambiguite n'est PAS publie.
type PlayerScore struct {
	XUID string `json:"xuid"`
	// Score est le score PERSONNEL (le chiffre des recompenses : 100 par frag, 50 par
	// assistance, 300 par capture de drapeau...). Il ne recule jamais ici — cf. l'en-tete.
	Score ScoreSeries `json:"score"`
	// Kills, Deaths, Assists sont les trois compteurs de base, repliques quel que soit le
	// mode (c'est ce qui rend le calque publiable en Slayer comme en CTF).
	Kills   ScoreSeries `json:"kills"`
	Deaths  ScoreSeries `json:"deaths"`
	Assists ScoreSeries `json:"assists"`
}

// ScoreTimeline est le calque du score dans le temps.
type ScoreTimeline struct {
	// Teams porte les deux camps quand le film les replique. Vide en solo, ou quand aucun
	// slot d'equipe n'emet le score de mode.
	Teams []TeamScore `json:"teams,omitempty"`
	// Players porte les joueurs dont le slot d'entite a ete apparie a une ligne de match.
	// Vide quand l'appelant n'a pas fourni les lignes (le pont d'identite passe par elles).
	Players []PlayerScore `json:"players,omitempty"`
}

// ScoreCoverage dit ce que vaut le calque du score — et ce qu'il ne vaut pas.
//
// MEME REGLE QUE LES AUTRES COUVERTURES : publier une courbe sans dire d'ou vient l'identite
// des camps, combien de manches ont ete lues et si la lecture a ete tronquee laisserait croire
// a une exhaustivite que la mesure ne garantit pas.
type ScoreCoverage struct {
	// TeamIdentity nomme la METHODE qui a rattache les slots d'equipe aux camps :
	// `a` (score final), `b` (somme des frags) ou `unresolved`.
	TeamIdentity string `json:"teamIdentity"`
	// Rounds est le nombre de manches RETENUES par le decodeur (les manches fantomes,
	// ancrages fortuits, sont ecartees). 1 sur un match sans manches multiples.
	Rounds int `json:"rounds"`
	// ModeSupported dit si le film replique le score de MODE sur au moins un slot d'equipe.
	// Faux = les compteurs de joueur peuvent exister sans qu'aucune courbe d'equipe n'existe.
	ModeSupported bool `json:"modeSupported"`
	// Truncated dit que la lecture des enregistrements a atteint son plafond (film anormal :
	// un seul du corpus de 22 l'a fait). Les courbes publiees s'arretent alors avant la fin
	// du match — le dire est le minimum, publier un score tronque en silence serait un
	// mensonge.
	Truncated bool `json:"truncated"`
	// Oracle nomme ce que la courbe MESURE : `displayed`, le score affiche par le jeu.
	Oracle string `json:"oracle"`
	// Points est le nombre total de points publies, equipes et joueurs confondus — le
	// denominateur de volume du calque.
	Points int `json:"points"`
}
