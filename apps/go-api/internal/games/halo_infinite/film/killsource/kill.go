package killsource

// kill.go — LE TYPE DE SORTIE. C est ici que vit la decision << garder LES DEUX VERITES >>.
//
// REGLE DE CONCEPTION, et elle explique toute la forme de ce fichier : un consommateur qui ne
// veut QUE le kill-feed doit pouvoir ignorer entierement la source, et INVERSEMENT. Les deux
// sous-structures sont donc COMPLETES chacune de son cote, et aucune ne se derive de l autre.
// Il n existe volontairement AUCUN champ << arme corrigee >> ni << vrai tueur >> : ce serait
// presenter une verite comme l amendement de l autre, ce que la doctrine interdit.

import (
	"fmt"

	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/games/halo_infinite/film/damagetag"
)

// Kill : une mort, et LES DEUX REPONSES a << qu est-ce qui l a tuee >>.
type Kill struct {
	// TimeMS : instant de la mort, en millisecondes depuis le debut du film, tel que le
	// kill-feed le date. Pour une mort de BOT (absente du kill-feed) c est l instant du kill
	// correspondant.
	TimeMS int

	// Victim : la victime. Elle vient du kill-feed quand il porte la mort, du roster de
	// replication (BOT_METADATA) quand la victime est un bot.
	Victim string

	// Feed : LA VERITE KILL-FEED — qui recoit le CREDIT. C est ce que le jeu affiche et ce sur
	// quoi les statistiques officielles du joueur sont baties.
	Feed FeedTruth

	// Source : LA VERITE SOURCE DE DEGAT — d ou vient le degat fatal. Lue dans le dead-state de
	// la victime, jamais dans l arme tenue, jamais dans le credit.
	Source SourceTruth

	// Diverges : la source du degat appartient a la VICTIME alors que le feed credite un autre
	// joueur. LES DEUX SONT VRAIS ; c est la divergence qui est l information. 8 cas sur les
	// quatre films de reference, confirmes 8/8 en Theater (RE_LOG 7ter.63).
	Diverges bool

	// Read : d ou vient techniquement cette ligne. A PONDERER, pas a afficher : la voie MARCHE
	// vaut 98.2 % de justesse de couple, le RATTRAPAGE du scan 78.4 % (RE_LOG 7ter.72).
	Read Provenance

	// Assist : l assistant declare par le KILL-EVENT. Ce n est PAS une troisieme verite : c est
	// un attribut de la verite KILL-FEED — le jeu credite un tueur et declare un assistant a
	// cote. `Assist.Known = false` veut dire QU ON NE SAIT PAS, jamais << pas d assistant >>.
	Assist Assist

	// KillerDamage : la part de degats du TUEUR, en pourcentage entier, lue dans le MEME
	// kill-event que l assistant. `Known = false` veut dire NON MESURE — c est le cas des morts
	// auxquelles aucun kill-event n a pu etre attache.
	KillerDamage DamageShare
	// AssistDamage : la part de degats de l ASSISTANT.
	//
	// ⚠ ELLE N EXISTE QUE SI LE CHAMP ASSISTANT EST PRESENT. Sans lui, le bloc du film porte une
	// CONSTANTE PAR FILM qui ne dit rien du kill — elle est alors publiee << non mesure >>, pas a
	// zero. Voir [decodeCtx.fillAssist] pour la regle et sa mesure.
	//
	// AUCUN PLAFOND A 100 n est impose ici ni sur `KillerDamage` : voir [DamageShare].
	AssistDamage DamageShare
}

// FeedTruth : ce que le JEU affiche. Se suffit a elle-meme.
type FeedTruth struct {
	// Killer : le joueur credite par le jeu. Vide quand la mort n est pas au kill-feed.
	Killer string
	// Present : le chunk HIGHLIGHT porte-t-il cette mort ? FAUX pour une mort de BOT — le
	// kill-feed du film est HUMAIN SEUL (un bot n a pas de XUID, sa mort ne produit aucun
	// event `death`), mesure sans aucune ancre en RE_LOG 7ter.59.
	Present bool
}

// SourceTruth : d ou vient le degat fatal. Se suffit a elle-meme.
type SourceTruth struct {
	// Tag : l identifiant `jpt!` (damage_effect) lu dans le dead-state de la victime. C est la
	// quantite BRUTE, celle qui ne depend d aucune table de nommage.
	Tag uint32

	// Display : l etiquette a afficher. Nom propre quand il y en a un et qu il est publiable,
	// [LabelOther] sinon. La REGLE est dans [labelOf] et elle est volontairement simple :
	// l encodage des sources sans nom n est pas la priorite du lot.
	Display string
	// Named : `Display` est-il un vrai nom, ou le repli [LabelOther] ?
	Named bool

	// Class : nature de la source (arme, melee, grenade, vehicule, objet explosif, degat
	// global, inconnu). Toujours renseignee, meme quand `Named` est faux — c est l information
	// que le repli [LabelOther] ne doit pas faire perdre.
	Class damagetag.Class
	// Status : ce que le consommateur a le droit de faire de l etiquette.
	//
	//	VALIDE        publiable telle quelle
	//	SOUS_RESERVE  la CLASSE est sure, le NOM PROPRE ne l est pas (cas mesure : un marteau de
	//	              campagne publie a la place du marteau multijoueur, RE_LOG 7ter.50)
	//	AMBIGU        NE PAS PUBLIER le nom : il serait affirmatif et faux
	//	INCONNU       tag `jpt!` valide, aucune source remontee
	Status damagetag.Status
	// Detail : la qualification mesuree (entree de grenade, chassis, etat de degat, effet).
	Detail string
	// Reserve : ce que l etiquette NE DIT PAS. Vide quand il n y a pas de reserve.
	Reserve string

	// Category : le modificateur de rapport de degat lu sur 4 bits a `comp+0x0c`. Il ne
	// discrimine PAS grenade et melee (les deux sortent `None`) : c est le TAG qui les porte.
	Category Category
}

// LabelOther : l etiquette de repli quand aucun nom propre n est publiable. Decision
// utilisateur : *<< on dira que c est Autres pour le moment >>*. Un consommateur qui veut
// traduire ou raffiner dispose de `Class`, `Status` et `Detail` a cote.
const LabelOther = "Autres"

// Path : quelle VOIE a lu l enregistrement. Les deux lisent le MEME champ et, quand elles
// repondent toutes les deux, elles repondent AU MEME BIT (346/346, desaccord 0 — RE_LOG
// 7ter.72). Ce n est donc pas un arbitrage entre deux mesures, c est une PREFERENCE entre deux
// localisateurs du meme enregistrement.
type Path string

// Les deux voies de lecture.
const (
	// PathWalk : la MARCHE deroule les records depuis le debut du paquet. Rappel plus faible,
	// mais une chaine complete l a atteint : gate (b) 98.2 %. Elle n a AUCUNE porte de
	// catalogue, ce qui la rend seule capable de detecter un catalogue perime.
	PathWalk Path = "marche"
	// PathScan : le SCAN DIRECT balaie les positions de bit et retient ce qui passe quatre
	// tests. Rappel superieur, precision inferieure : gate (b) 78.4 % en rattrapage.
	PathScan Path = "scan"
)

// Origin : COMMENT la ligne a ete appariee. Ce n est pas la meme chose que la voie : l origine
// dit ce que la ligne AFFIRME, la voie dit avec quelle precision.
type Origin string

// Les CINQ origines d appariement. Les troisieme et quatrieme sont les deux populations de bot,
// et elles sont SYMETRIQUES : le feed porte le kill sans la mort, ou la mort sans le kill. La
// cinquieme ne qualifie PAS un [Kill] : c est celle des morts que personne ne revendique.
const (
	// OriginCredit : le couple (tueur, victime) du dead-state est EXACTEMENT celui du
	// kill-feed dans la fenetre. Les deux verites concordent.
	OriginCredit Origin = "credit-concordant"
	// OriginSelfSource : la source appartient a la victime (`victime == tueur` dans le
	// dead-state) alors que le feed credite un autre joueur. Apparie sur la VICTIME SEULE.
	OriginSelfSource Origin = "source-victime"
	// OriginBot : la victime est un BOT. Population NEUVE : elle n a JAMAIS ete au
	// denominateur, le kill-feed etant humain-seul.
	OriginBot Origin = "bot"
	// OriginBotKiller : le TUEUR est un BOT. Population NEUVE elle aussi, et SYMETRIQUE de la
	// precedente : le kill-feed porte la MORT (la victime est humaine, elle a un XUID) mais
	// aucun evenement `kill` en face (le bot n en a pas). La mort ne se recolle donc a rien et
	// n a JAMAIS pu entrer dans les couples reconstruits.
	//
	// CE QUE LA LIGNE CHANGE DE PROVENANCE, ET IL FAUT LE LIRE AVANT DE CONSOMMER : [Kill.Victim]
	// et [Kill.TimeMS] viennent du KILL-FEED comme partout, mais [FeedTruth.Killer] vient du
	// ROSTER DE REPLICATION (BOT_METADATA), pas du feed. [FeedTruth.Present] reste VRAI parce
	// qu il qualifie la MORT, et la mort est bien au feed. C est exactement l inverse d
	// [OriginBot], ou c est la victime qui vient de la replication.
	OriginBotKiller Origin = "tueur-bot"
	// OriginUnclaimed : PERSONNE ne revendique cette mort. Le kill-feed la porte sans aucun
	// kill en face, aucun bot ne l explique, et le dead-state designe la VICTIME elle-meme en
	// tueur. C est la provenance des lignes de [Result.UnclaimedDeaths] — elles ne sont JAMAIS
	// des [Kill] (il n y a pas de credit a porter, et en inventer un serait un mensonge).
	OriginUnclaimed Origin = "sans-revendication"
)

// Provenance : d ou vient techniquement une ligne publiee.
type Provenance struct {
	Path   Path
	Origin Origin
	// Multiplicity : nombre de paquets DISTINCTS portant un candidat au meme triplet
	// (bit, tag, indice victime). Un vrai dead-state tombe a un bit quelconque ; un champ fixe
	// lu comme un dead-state se repete AU MEME BIT (11 fois au bit 746 sur un film). C est le
	// discriminant structurel des morts auto-infligees (RE_LOG 7ter.61 C1).
	Multiplicity int
}

func (p Provenance) String() string {
	return fmt.Sprintf("%s / %s", p.Origin, p.Path)
}

// Category : modificateur de rapport de degat (enum de 10 valeurs, 4 bits).
type Category int

// Les valeurs de l enum, telles que le moteur les nomme. NE PAS TRADUIRE ICI : ce sont des
// identifiants stables, la traduction FR/EN appartient a la couche d affichage (regle i18n du
// depot : aucun libelle en dur cote Go).
const (
	CategoryNone Category = iota
	CategoryHeadshot
	CategoryHeadshotMultiplier
	// CategorySilentMelee : assassinat. La melee ORDINAIRE sort `None` — c est le TAG qui la
	// porte, pas la categorie (RE_LOG 7ter.37).
	CategorySilentMelee
	CategoryCollisionDamage
	// CategoryAttachedDamage : projectile FIXE sur la cible — grenade collee ET
	// supercombinaison Needler.
	CategoryAttachedDamage
	CategoryWeakSpot
	CategoryChainedProjectile
	CategorySweetHeat
	CategoryVehicleTransferDamage
)

// categoryNames : identifiants du moteur, indexes par la valeur de l enum.
var categoryNames = [...]string{
	"None", "Headshot", "HeadshotMultiplier", "SilentMelee", "CollisionDamage",
	"AttachedDamage", "WeakSpot", "ChainedProjectile", "SweetHeat", "VehicleTransferDamage",
}

// Name : l identifiant du moteur, ou "?<n>" hors enum.
func (c Category) Name() string {
	if c < 0 || int(c) >= len(categoryNames) {
		return fmt.Sprintf("?%d", int(c))
	}
	return categoryNames[c]
}

func (c Category) String() string { return c.Name() }

// Coverage : LES DENOMINATEURS, NOMMES. Il n en existe pas un seul, il en existe quatre, et les
// confondre est la faute que RE_LOG 7ter.66 a corrigee au prix d une section entiere.
type Coverage struct {
	// Covered : morts publiees.
	Covered int
	// RealPairs : couples REELS = couples reconstruits MOINS les couples FABRIQUES par la
	// reconstruction de feed. C EST LE DENOMINATEUR DE REFERENCE. Un couple fabrique n est pas
	// une mort manquee : c est une mort qui n existe pas (la vraie victime est un bot).
	RealPairs int
	// ReconstructedPairs : couples reconstruits, couples fabriques COMPRIS.
	ReconstructedPairs int
	// GhostPairs : les couples FABRIQUES, retires de `RealPairs`. Publie pour que le retrait
	// soit visible et non dissimule.
	GhostPairs int
	// SameInstantPairs : couples portant kill ET death au MEME instant. Les autres couples
	// reconstruits sont RECOLLES sur un voisin — un recollage legitime et mesure (ces couples
	// echouent MOINS que les autres, p = 0.886), mais c est lui qui fabrique les fantomes.
	SameInstantPairs int
	// FeedKills, FeedDeaths : evenements bruts du kill-feed. NE CONTIENNENT AUCUNE MORT DE BOT.
	FeedKills  int
	FeedDeaths int
	// BotDeaths : morts de bot decodees. POPULATION NEUVE : jamais au denominateur, donc jamais
	// a additionner a `Covered` pour fabriquer un taux.
	BotDeaths int
	// BotKillerDeaths : morts INFLIGEES PAR un bot. DEUXIEME POPULATION NEUVE, et elle n est ni
	// `Covered` ni `BotDeaths` :
	//
	//	`Covered`         humain tue par un humain — le feed porte le kill ET la mort.
	//	`BotDeaths`       BOT tue par un humain    — le feed porte le kill, pas la mort.
	//	`BotKillerDeaths` humain tue par un BOT    — le feed porte la mort, pas le kill.
	//
	// Elle n a JAMAIS pu etre au denominateur des couples : une mort que rien ne consomme ne
	// produit aucun couple. NE PAS L ADDITIONNER A `Covered` : le seul denominateur qui
	// l accueille est celui des morts de l API, la seule reference complete.
	BotKillerDeaths int
}

// UnclaimedDeath : UNE MORT QUE PERSONNE NE REVENDIQUE, et la source qui l a causee.
//
// # POURQUOI CE N EST PAS UN [Kill], ET POURQUOI CE N EN SERA JAMAIS UN
//
// Un [Kill] porte DEUX verites dont l une est un CREDIT ; ici il n y a pas de credit a porter.
// Le kill-feed porte la MORT et AUCUN kill en face, et le dead-state de la victime designe la
// VICTIME ELLE-MEME comme source du degat. Ranger ces lignes parmi les kills obligerait a
// inventer un tueur — exactement ce que la doctrine du paquet interdit.
//
// # LA POPULATION, ET SA GARDE
//
// Elle part de `feed.orphD` (les morts que la reconstruction de couples n a jamais consommees,
// cf. [killFeed.split]) MOINS celles que le temps 5 explique par un TUEUR BOT : celles-la ont un
// tueur, elles sortent en [Kill] avec [OriginBotKiller].
//
// Ce qui reste n est publie QUE si un dead-state de la fenetre nomme la victime ET porte le MEME
// indice en tueur. Sans cette egalite on ne publie rien : un dead-state qui designerait un autre
// joueur decrirait un kill que le feed ne porte pas, et le presenter comme une mort de soi serait
// affirmatif et faux. C est la meme regle que partout : un couple contraint des deux cotes.
//
// # CE QUE LE CONSOMMATEUR EN FAIT
//
// [SourceTruth.Class] est la NATURE du degat (`DEGAT_GLOBAL` = chute / environnement /
// hors-limites, `ARME` = sa propre arme, ...). Elle est etablie meme quand le nom sort en
// [LabelOther] — c est elle qui distingue une chute d un tir de roquette trop pres d un mur.
// La porte [Result.LineByLinePublishable] vaut ici comme pour le reste : ces lignes sont nommees
// par la MEME bijection indice -> joueur.
type UnclaimedDeath struct {
	// TimeMS : instant de la mort, tel que le kill-feed le date.
	TimeMS int
	// Victim : la victime, telle que le kill-feed la nomme.
	Victim string
	// VictimXUID : le xuid de la victime. C est la SEULE cle de jointure fiable pour un
	// consommateur qui rapproche ces morts d une autre source (pistes de film, scoreboard).
	VictimXUID uint64
	// Source : d ou vient le degat fatal, lu dans le dead-state de la victime.
	Source SourceTruth
	// Read : la provenance technique de la ligne (voie de lecture, multiplicite).
	Read Provenance
}

// Result : tout ce qu une passe rend.
type Result struct {
	// Kills : les morts publiees, triees par instant.
	Kills []Kill
	// UnclaimedDeaths : les morts que le kill-feed porte SANS aucun tueur, et dont le
	// dead-state designe la victime elle-meme (suicides, chutes, hors-limites). Triees par
	// instant. Population RARE mais COMPLETE, mesure du 2026-08-14 : 5 publiees sur 5
	// orphelines (64e8adfa 2, e94163af 1, 606d9844 2, 000d5950 0), 0 inexpliquee, ecart au
	// feed de 0 a 4 ms. Instrument rejouable : `neutral_death_research_test.go`.
	UnclaimedDeaths []UnclaimedDeath
	// Coverage : les denominateurs.
	Coverage Coverage
	// Health : la metrique de sante, prete pour `expvar` via `Health.ExpvarPairs()`.
	// Le type est PUR (zero dependance interne) et vit dans `internal/analysis/filmdec`.
	Health filmdec.KillSourceHealth
	// Stats : ce qu il faut pour PONDERER et pour VERIFIER que l hybride est bien une
	// preference et non un arbitrage.
	Stats Stats
	// Roster : les noms retenus, bots compris, et la bijection indice -> joueur.
	Roster Roster
	// Calibration : la calibration map-dependante retenue, sous forme lisible. Aucune valeur
	// n est a fournir a la main : les quatre films de reference rendent quatre couples
	// DISTINCTS (RE_LOG 7ter.40).
	Calibration string
	// BijectionMargin : ecart de score entre la meilleure bijection et la meilleure a UNE
	// transposition pres. ZERO = au moins deux joueurs sont interchangeables.
	BijectionMargin int
	// Probe : la sonde a porte de catalogue RELACHEE. NIL quand la couverture est complete —
	// c est le seul regime ou elle porterait de l information, et elle coute cher.
	Probe *RelaxedProbe
}

// LineByLinePublishable : les attributions ligne par ligne sont-elles publiables ?
//
// DEUX CONDITIONS, et la seconde est celle qui a interdit le BTB : la bijection doit avoir une
// MARGE STRICTEMENT POSITIVE (marge nulle = deux joueurs interchangeables, donc les lignes sont
// exactes en AGREGAT et fausses individuellement, RE_LOG 7ter.53), et la sante ne doit pas etre
// en ALERTE. Un `false` ne dit pas que le decodage est faux : il dit que seul l agregat l est.
//
// CETTE PORTE VAUT POUR TOUT CE QUE LA LIGNE PORTE — la source du degat, LE CREDIT, L ASSISTANT et
// LES DEUX PARTS DE DEGATS. C est une DECISION, et elle refuse deliberement une porte separee par
// donnee : l assistant est nomme par la MEME bijection indice -> joueur que le reste, il tombe
// donc exactement au meme endroit. La mesure BTB le confirme sans ambiguite — 62 assistants nommes
// pour 122 a l API (51 %), contre 17/17 et 29/29 en Arena.
func (r *Result) LineByLinePublishable() bool {
	return r.BijectionMargin > 0 && len(r.Health.Alerts()) == 0
}

// Stats : les quantites qui permettent de PONDERER une sortie, et de verifier les proprietes sur
// lesquelles l architecture repose.
type Stats struct {
	// Walk, Scan : le gate (b) PAR VOIE, sur la population `victime != tueur`. C est ce qui
	// permet a un consommateur de ponderer : 98.2 % contre 78.4 % sur la serie de reference.
	Walk, Scan PathStats
	// SelfWalk, SelfScan : idem pour les morts dont la source appartient a la victime. SIX des
	// HUIT morts auto-infligees de la serie de reference sont atteintes par la MARCHE, donc par
	// la voie a 98.2 % — un renforcement direct de ce resultat.
	SelfWalk, SelfScan PathStats
	// Bot : les morts de bot. `Population` = kills du feed sans victime humaine.
	Bot PathStats
	// BotKiller : les morts INFLIGEES PAR un bot. `Population` = morts du feed que la
	// reconstruction de couples n a jamais consommees, donc sans tueur humain en face.
	BotKiller PathStats
	// Unclaimed : les morts que PERSONNE ne revendique (cf. [UnclaimedDeath]). `Population` est
	// la MEME que celle de `BotKiller` — les deux temps se partagent les morts orphelines du
	// feed, l un les explique par un tueur bot, l autre par la victime elle-meme. Un ecart
	// entre la somme des deux `Published` et la population dit ce qui reste inexplique.
	Unclaimed PathStats
	// Redundant : candidats du scan portant la MEME position qu un enregistrement de la marche.
	// Ce sont les memes objets lus deux fois : 346 sur la serie de reference.
	Redundant int
	// NoBit : enregistrements de la marche sans position enregistree. Ils restent utilisables
	// pour l appariement, seulement inaptes au test de redondance — et ils sont COMPTES.
	NoBit int
	// Agree, Disagree : instants ou LES DEUX voies repondent. `Disagree` DOIT valoir zero : c est
	// ce qui rend l hybride legitime comme PREFERENCE. S il bouge, c est devenu un ARBITRAGE et
	// il faut le documenter comme tel.
	Agree, Disagree int
	// MultiCandidate : morts ayant PLUS D UN candidat distinct en concurrence. Zero sur la serie
	// de reference : le mecanisme d arbitrage existe, il n a jamais eu a servir.
	MultiCandidate int
	// PacketsWithEvents, PacketsLocated : les morts sont TOUTES dans les paquets a events ; la
	// part localisee (95.8 a 97.6 % sur la serie de reference) borne ce que la marche peut lire.
	PacketsWithEvents, PacketsLocated int
	// Assist : les denominateurs de la passe d assistants. Elle est INDEPENDANTE de la source du
	// degat : elle lit la liste d evenements, pas la boucle de records.
	Assist AssistStats
}

// PathStats : le gate (b) d une voie. `Population` est ce qu elle a propose, `Matched` ce dont
// le couple exact existe au kill-feed dans la fenetre, `Published` ce qui a effectivement ete
// publie (une mort deja couverte par une voie plus contrainte ne l est pas deux fois).
type PathStats struct{ Population, Matched, Published int }

// Ratio : `Matched / Population`, dans [0,1]. Zero quand la population est vide.
func (s PathStats) Ratio() float64 {
	if s.Population <= 0 {
		return 0
	}
	return float64(s.Matched) / float64(s.Population)
}
