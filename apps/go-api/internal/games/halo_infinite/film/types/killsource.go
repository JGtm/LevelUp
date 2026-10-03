package types

// killsource.go — LES TYPES DE CONTRAT DE LA COUCHE DES FAITS, PART `killsource`.
//
// Deplaces de `film/facts/killsource/` au lot 2.6.2 (2026-09-16), SANS reecriture. TROIS des six
// types que la note de preparation §3.2 avait comptes : `Kill` porte un champ NON EXPORTE
// (`paquet`) que seul son producteur peut ecrire, `Result` nomme deux types de `grammar`, et
// `FilmTablePinning` porte une REGLE (`AffectationUnique`) — les trois restent ou ils sont, et
// le volet grammaire/rejeu du lot 2.6 dira ce qu il en fait.

// ApparStats : D OU VIENT L APPARIEMENT `dead-state <-> kill-feed` de chaque ligne PUBLIEE
// (lot 1.9.7).
//
// AUCUN RATIO — les taux se calculent chez le lecteur, avec le denominateur qu il nomme. Les
// quatre premiers champs se somment exactement aux lignes publiees par les temps 1 a 6 de
// l hybride ; le cinquieme est un DIAGNOSTIC et n en fait pas partie.
type ApparStats struct {
	// Identite : l appariement a ete decide par l IDENTITE DE PAQUET — le film a ecrit le
	// dead-state et le kill-event 85 dans le MEME paquet. C est la part LUE de la publication.
	Identite int
	// Fenetre : LE REPLI `repli_appariement_par_fenetre_temporelle` aux temps 1 a 3 (couple
	// exact et source auto-infligee). Tant qu il monte, des lignes sont encore appariees par une
	// coincidence temporelle de 2,5 s : c est lui qui dira quand le repli pourra etre retire
	// (D14 d).
	Fenetre int
	// BotFenetre : LE REPLI `repli_mort_de_bot_premier_candidat` aux temps 4 et 5. Le kill-feed
	// etant humain-seul, ces instants ne portent presque jamais d identite : le repli y est la
	// voie NORMALE, et son compte le mesure au lieu de le supposer.
	BotFenetre int
	// NonRevendiqueeFenetre : LE REPLI `repli_mort_non_revendiquee_la_plus_proche` au temps 6.
	NonRevendiqueeFenetre int
	// CouplesSansIdentite : couples publies du kill-feed auxquels AUCUN kill-event 85 ne s est
	// attache. C est le diagnostic typé qui OUVRE le repli de la fenetre (D14 b) — sans lui, un
	// compte de replis ne designerait aucune correction.
	CouplesSansIdentite int
}

// Assist : ce que le kill-event declare a cote du tueur.
//
// UN SEUL ASSISTANT — ET LA PORTEE DE CE CONSTAT EST PLUS ETROITE QU IL N Y PARAIT. La grammaire
// n expose qu un emplacement d assistant PAR KILL-EVENT ; le seul surplus qu on sache observer est
// donc celui de DEUX KILL-EVENTS ATTACHES A LA MEME MORT et nommant des assistants differents.
// Ce que `Extra == 0` etablit est exactement cela, et rien de plus : *aucune mort n a recu deux
// ENREGISTREMENTS nommant des assistants distincts*. Ce n est PAS << aucune mort ne porte deux
// assistants >> — cette seconde affirmation-la n est mesuree par personne (voir la RESERVE
// ARITHMETIQUE sur `9b191a7f`, plus bas).
type Assist struct {
	// Name : l assistant. Vide quand le kill-event n en declare pas, ou quand il en declare un
	// que l on refuse (voir `Rejected`).
	Name string
	// Index : l indice de replication brut, -1 quand le champ est absent. C est la quantite qui
	// ne depend d aucune bijection — a citer si un nom surprend.
	Index int
	// Rejected : le motif de refus, vide quand il n y en a pas. `AssistRejectSelf` ou
	// `AssistRejectRoster`.
	Rejected string
	// Known : un kill-event a-t-il ete attache a cette mort ? FAUX ne veut pas dire << pas
	// d assistant >> : cela veut dire QU ON NE SAIT PAS. La distinction est la raison d etre de
	// ce champ, et elle doit survivre jusqu en base.
	Known bool
	// Extra : nombre d assistants DISTINCTS EN SURPLUS observes sur cette mort, au-dela de celui
	// qui est publie dans `Name`.
	//
	// C EST LE GARDE-FOU DE L HYPOTHESE << UN SEUL ASSISTANT >>, ET IL DOIT ETRE PORTE PAR LA
	// LIGNE, PAS SEULEMENT PAR L AGREGAT. Tant que ce champ n existait pas, la colonne
	// `assist_extra_count` de `match_kill_events` n etait alimentable par personne :
	// `SELECT SUM(assist_extra_count)` — que la documentation presente comme LE DECLENCHEUR DE
	// MIGRATION vers une table fille — valait zero PAR CONSTRUCTION, jamais par mesure. Un
	// garde-fou muet est pire que pas de garde-fou : il rassure.
	//
	// PORTEE : voir le commentaire du type. Ce compteur ne voit qu un surplus porte par un SECOND
	// KILL-EVENT ATTACHE ; il est structurellement aveugle a un second assistant qui serait
	// declare autrement.
	Extra int
}

// CoupleStats : d ou vient le couple (tueur, victime) de chaque instant du kill-feed.
//
// AUCUN RATIO — les taux se calculent chez le lecteur, avec le denominateur qu il nomme. Les
// trois premiers champs se somment exactement aux kills du feed que la structure a classes.
type CoupleStats struct {
	// MemeInstant : le kill-feed porte le kill ET la mort au meme instant. Le couple est ECRIT
	// par le feed ; aucune lecture ni aucun repli n intervient.
	MemeInstant int
	// Lus : couples dont le KILL-EVENT 85 a decide la victime. C est la part de la publication
	// qui vient d une LECTURE et non d une reconstruction.
	Lus int
	// Recolles : LE REPLI (`repli_couple_recolle_sur_le_voisin`) — la victime vient de la mort
	// d un instant voisin parce que le film s est tu. Tant qu il monte, des couples sont encore
	// DEVINES : c est lui qui dira quand le repli pourra etre retire (D14 d).
	Recolles int
	// Perdus : kills que ni la lecture ni le repli n ont su appareiller (`orphK`). La victime
	// n est pas au feed et le film ne la nomme pas : c est le dead-state qui tranchera.
	Perdus int
	// VictimesBotLues : kills dont le film NOMME un bot en victime. Population qui n entre dans
	// aucun couple ; avant ce lot elle etait RECOLLEE sur la mort d un humain.
	VictimesBotLues int
	// Muet : instants ou la lecture s est tue — aucun kill-event de la fenetre ne nomme ce tueur
	// par deux indices EPINGLES. C est le diagnostic typé qui ouvre le repli (D14 b).
	Muet int
	// Ambigu : instants ou DEUX kill-events non consommes nomment ce tueur et DES VICTIMES
	// DIFFERENTES. La lecture ne tranche pas ; le repli reprend, et le compteur le dit.
	Ambigu int
	// Accord / Contradiction : LE CONTROLE de la lecture par le recollage, jamais l inverse
	// (D14 b). `Accord` = le voisin immediat porte bien la mort du joueur que le film nomme ;
	// `Contradiction` = le film nomme un joueur dont AUCUN voisin immediat ne porte la mort. La
	// valeur publiee ne bouge pas — la lecture prime —, la contradiction se compte.
	Accord, Contradiction int
}
