package killsource

// options.go — LA CONFIGURATION GELEE.
//
// Elle n est PAS un espace de reglage : c est la configuration qui a produit les chiffres
// publies, et chacun de ses points a ete mesure en A/B. Les champs existent pour qu un
// operateur puisse REJOUER une ligne de base, jamais pour qu un appelant << ajuste >>.
//
// L outil de RE (`cmd/tmp_deadstate`) pilote les memes bascules par variables d environnement ;
// ce paquet n en lit AUCUNE — un decodeur de production dont le comportement depend de
// l environnement du process n est pas reproductible.

import (
	"math/rand"

	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
)

// Options : les quatre bascules mesurables, plus la CARTE du match (une donnee, pas une
// bascule). Le zero-value N EST PAS la configuration retenue :
// passer `nil` a [Decode], ou partir de [DefaultOptions], est la bonne facon de faire.
type Options struct {
	// Bots : lire BOT_METADATA (paquet type 12) pour etendre le roster et epingler les slots
	// de bot. Sans lui, `nPlayers` reste le nombre de joueurs du kill-feed et les morts de bot
	// sont invisibles. DEFAUT : true.
	Bots bool

	// SelfSource : accepter les morts dont la source appartient a la victime
	// (`victime == tueur` dans le dead-state). Sans lui, le filtre historique `v != k` les
	// jette — et il jetait AUSSI les 8 morts auto-infligees confirmees en Theater.
	// DEFAUT : true.
	SelfSource bool

	// MultiplicityMax : seuil de MULTIPLICITE au-dela duquel un candidat `victime == tueur` est
	// refuse. DERIVE D UNE MESURE, pas ajuste : sur les candidats apparies des quatre films la
	// multiplicite vaut 1, sans exception. Le seuil est donc la plus petite valeur possible et
	// son cout en vrais positifs est ZERO par construction, a tous les seuils 2..6.
	//
	// PORTEE, a ecrire avec le resultat : le test ne SEPARE quelque chose que sur deux films
	// sur quatre (histogrammes m=1:4 / m=1:4 / m=1:9 m=2:2 / m=1:9 m=11:11). Son COUT est nul
	// partout, son POUVOIR DISCRIMINANT ne l est pas. DEFAUT : 2.
	MultiplicityMax int

	// StrongTagRequired : exiger `tag >= 0x10000` pour une mort dont la source appartient a la
	// victime.
	//
	// CE FILTRE ACHETE DE LA JUSTESSE, JAMAIS DE LA COUVERTURE — et sur CE corpus, sous CETTE
	// architecture, il n achete rien du tout. Il faut ecrire les deux moities, parce que la
	// premiere seule a deja induit en erreur :
	//
	//	SOUS L ARCHITECTURE SCAN-D-ABORD (mesure 7ter.70, verifiee 7ter.71) le filtre est
	//	PORTEUR : sans lui la ligne `fccc61cd 01:25` publie `00000028` — etiquete << degat
	//	global >>, statut VALIDE — au lieu de `acd1cff4` (M41 SPNKr, confirme en Theater). Les
	//	deux candidats sont dans le MEME paquet, et la couverture ne bouge pas d un pouce : un
	//	controle qui ne regarde que les comptes ne le voit pas.
	//
	//	SOUS L HYBRIDE — ce que ce paquet implemente — il est INERTE : la MARCHE sert cette
	//	ligne et publie `acd1cff4` que le filtre soit actif ou non. Zero etiquette changee sur
	//	les quatre films (7ter.72 (0)(C), re-mesure a l A/B par
	//	`TestLigneDiscriminanteEstServieParLaMarche`).
	//
	// IL RESTE DONC ACTIF COMME FILET, pas comme correcteur : le jour ou la marche cesse
	// d atteindre une de ces lignes — catalogue perime, calibration en echec — le scan la
	// reprend, et le regime porteur revient. Son cout en vrais positifs est nul.
	//
	// PORTEE : il ne vaut QUE dans la population `victime == tueur`. Applique a toute la
	// population il couterait 47 vrais positifs sur 82 sur un film, le BR75 `0000b29c` vivant
	// sous 0x10000. DEFAUT : true.
	StrongTagRequired bool

	// BijectionRestarts : redemarrages aleatoires de la montee locale qui resout la bijection
	// indice -> joueur. La graine est FIXE (voir [bijectionSeed]) : deux passes sur le meme
	// film rendent la meme bijection, sinon la non-regression serait invérifiable.
	// DEFAUT : 40.
	BijectionRestarts int

	// Views : nombre de vues de replication marchees par paquet. DEFAUT : 8.
	Views int

	// Carte : L ENTREE DE CATALOGUE DE LA CARTE DU MATCH, et c est une DONNEE, pas une bascule.
	//
	// POURQUOI ELLE EST ICI, ET PAS DEVINEE. Les largeurs d axe du chemin absolu de position et
	// la largeur d index de plage sont installees AU CHARGEMENT DE LA CARTE par le moteur
	// (`FUN_140be9a14`) et ne se lisent NULLE PART dans le film. Jusqu au lot 3.4.1 ce paquet
	// les INFERAIT par balayage — il etait le seul a le faire, parce qu il etait le seul chemin
	// de decodage a ne recevoir aucune entree de catalogue, quand `replay.BuildFromFilm` la
	// recoit depuis le 2026-08-15.
	//
	// L INFERENCE EST DEVENUE ORACLE (V17, M3-Q8 : « la valeur LUE prime sur la valeur
	// mesuree ») : sans cette entree elle ne serait remplacee par RIEN — le decodeur retomberait
	// sur l invariant du profil, c est-a-dire sur les largeurs d UNE carte (`cliffhanger`,
	// 13/13/14) appliquees a toutes, quand les quatre films de reference de ce paquet rendent
	// QUATRE couples distincts. La demotion de l inference et l arrivee de la carte sont donc le
	// MEME geste ; les separer aurait ete une regression sur toute carte dont les largeurs ne
	// sont pas l invariant.
	//
	// NIL EST LICITE, ET COMPTE : le decodeur retombe sur l invariant, le DIT
	// (`repli_carte_absente_largeurs_par_defaut` au registre, avertissement par film) et
	// [Result.Calibration] porte la mention. C est le cas d un appelant sans base sous la main
	// (CLI unitaire, ouvrier distant, collecteur sans `WithPositionCapture`).
	//
	// PROVENANCE DES VALEURS : [profile.MapQuantEntry.PrecisionAbsolue] — les trois largeurs
	// d axe derivees des bornes par la loi du moteur (accord 79 cartes sur 79), la largeur
	// d index de plage et l index de la plage jouee.
	Carte *profile.MapQuantEntry
}

// bijectionSeed : graine FIXE de la montee locale. Elle ne doit pas bouger : c est elle qui
// rend le resultat reproductible d une execution a l autre.
const bijectionSeed int64 = 20260726

// DefaultOptions : LA CONFIGURATION GELEE, celle qui a produit 371/371 couples REELS et les
// 30 ancres Theater conformes (RE_LOG 7ter.70, verifie 7ter.71 ; hybride 7ter.72).
func DefaultOptions() Options {
	return Options{
		Bots:              true,
		SelfSource:        true,
		MultiplicityMax:   2,
		StrongTagRequired: true,
		BijectionRestarts: 40,
		Views:             8,
	}
}

// normalize : remplace les valeurs absentes par celles du defaut. Un appelant qui ne renseigne
// que `Bots` ne doit pas se retrouver avec zero redemarrage et zero vue.
func (o *Options) normalize() {
	d := DefaultOptions()
	if o.MultiplicityMax <= 0 {
		o.MultiplicityMax = d.MultiplicityMax
	}
	if o.BijectionRestarts <= 0 {
		o.BijectionRestarts = d.BijectionRestarts
	}
	if o.Views <= 0 {
		o.Views = d.Views
	}
}

// newRNG : le generateur de la montee locale, toujours a graine fixe.
func newRNG() *rand.Rand { return rand.New(rand.NewSource(bijectionSeed)) } //nolint:gosec // reproductibilite, pas de cryptographie

// tolMS : demi-fenetre d appariement entre un dead-state et un evenement du kill-feed.
// Valeur historique du chantier, employee par TOUTES les mesures publiees : la changer
// invaliderait la comparabilite avec la serie de reference.
const tolMS = 2500
