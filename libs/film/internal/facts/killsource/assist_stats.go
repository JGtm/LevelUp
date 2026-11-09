package killsource

// assist_stats.go — LES DENOMINATEURS DE LA PASSE D ASSISTANTS : [AssistStats].
//
// SORTI D `assist.go` PAR DEPLACEMENT PUR au lot J7.6 (2026-09-26, PLAN_SUITE_AUDIT_DECODEUR_FILM) :
// le fichier atteignait le plafond de 500 lignes (`archlint/film_file_size_test.go`) au moment ou le
// compte des doublons d enregistrement s y ajoutait. Aucun octet de code n est change.

// AssistStats : les denominateurs de la passe d assistants. AUCUN RATIO — les taux se calculent
// chez le lecteur, avec le denominateur qu il nomme.
type AssistStats struct {
	// KillEvents : kill-events localises dans le film, toutes morts confondues.
	KillEvents int
	// Attached : morts publiees auxquelles un kill-event a ete attache par COUPLE EXACT. C est LE
	// DENOMINATEUR de tout le reste de cette structure.
	//
	// CE QU IL NE MESURE PAS : la bijection indice -> joueur est ajustee en MAXIMISANT ce meme
	// critere de couple ([quadScore]). Ce nombre est donc la valeur d une fonction objectif
	// optimisee, PAS un test independant de l espace d indices. Il sert de SELECTEUR DE
	// POPULATION, et a rien d autre.
	Attached int
	// Multi : morts pour lesquelles PLUSIEURS kill-events non consommes correspondaient. Non nul
	// = l appariement n est plus unique et les lignes concernees sont a regarder.
	//
	// C EST UNE QUANTITE VIVANTE, DESORMAIS FIGEE PAR FILM (0 / 1 / 0 / 1, voir `assistAttendu`) :
	// elle valait deja 1 sur `9b191a7f` et 1 sur `fccc61cd` sans que rien ne l ancre, alors que
	// c est elle qui ouvre le domaine de [pickAssistHit] et [distinctAssists].
	Multi int
	// Named : assistants publies avec un nom.
	Named int
	// NoAssist : kill-events attaches dont le champ assistant est ABSENT. C est un << pas
	// d assistant >> mesure, a ne pas confondre avec `Known = false`. C est aussi la population ou
	// la part de degats de l assistant N EST PAS PUBLIEE : le bloc y porte une constante par film.
	// La RESERVE ARITHMETIQUE sur `9b191a7f` (plus haut) donne une raison de ne pas tenir CHAQUE
	// ligne de ce compteur pour acquise.
	NoAssist int
	// RejectedSelf, RejectedVictim, RejectedRoster : les rejets, chacun compte a part.
	RejectedSelf, RejectedVictim, RejectedRoster int
	// FlagSet : kill-events attaches dont le bit R(1) qui precede le champ assistant vaut 1. Sa
	// semantique n est PAS etablie ; le compteur existe pour qu on puisse la chercher sans
	// re-decoder.
	FlagSet int
	// AssistMulti : morts pour lesquelles PLUSIEURS KILL-EVENTS ATTACHES nomment des assistants
	// DISTINCTS.
	//
	// SON DENOMINATEUR EST << LES KILL-EVENTS ATTACHES >>, PAS << LES MORTS >>. Il n est calcule
	// que si DEUX kill-events distincts sont attaches a la meme mort, et la grammaire ne lit qu UN
	// champ d assistant par kill-event. Sa nullite prouve donc *aucune mort n a recu deux
	// ENREGISTREMENTS nommant des assistants differents* — elle NE prouve PAS *aucune mort ne
	// porte deux assistants*. La seconde affirmation n est mesuree par personne, et la RESERVE
	// ARITHMETIQUE sur `9b191a7f` (plus haut dans ce fichier) donne une raison arithmetique de ne
	// pas la tenir pour acquise.
	AssistMulti int
	// AssistExtraTotal : somme des [types.Assist.Extra] de la passe. Meme portee que `AssistMulti`, mais
	// il compte les assistants en surplus et non les morts — une mort pourrait en porter deux.
	// C est la quantite qui alimente `assist_extra_count` en base.
	AssistExtraTotal int
	// AssistFieldDisagree : morts ou les kill-events attaches NE S ACCORDENT PAS sur la PRESENCE
	// du champ assistant — au moins un le porte, au moins un ne le porte pas.
	//
	// IL SURVEILLE UNE DECISION, PAS UNE MESURE. `AssistMulti` compte les desaccords sur l IDENTITE
	// de l assistant ; celui-ci compte les desaccords sur son EXISTENCE, que [pickAssistHit] tranche
	// en faveur du porteur. Sans lui, un << pas d assistant >> publie ne se distinguerait pas d un
	// << pas d assistant >> ARBITRE. Il vaut ZERO sur les quatre films, et ce zero est MESURE : les
	// seules morts a multi-attachement du corpus (une sur `9b191a7f`, une sur `fccc61cd`) portent
	// DEUX FOIS LE MEME enregistrement, a deux positions de bit distantes de 15, champs identiques
	// (RE_LOG 7ter.77).
	AssistFieldDisagree int
	// KillerPctOver100, AssistPctOver100 : parts de degats publiees STRICTEMENT SUPERIEURES a 100.
	//
	// Ce ne sont PAS des rejets : aucun plafond n est impose, et ces lignes sont publiees. Le
	// compteur existe pour que la population concernee (1.7 % des kill-events attaches sur le
	// corpus large, valeurs jusqu a 228) reste VISIBLE par film au lieu d etre du folklore. Son
	// interpretation — degat excedentaire — n est PAS etablie.
	KillerPctOver100, AssistPctOver100 int
	// ParLaFenetre : morts dont le kill-event a ete attache par la FENETRE de 2,5 s et non par
	// l identite de paquet — le repli `repli_appariement_par_fenetre_temporelle` (lot 1.9.7).
	// Mesure du lot : ZERO sur les 21 films entiers ; c est le compte qui permettra de le retirer
	// d ici (D14 d).
	ParLaFenetre int
	// Gate15 : l etat d execution que le rattrapage des kills a tranche pour ce film ; faux quand
	// aucune trame ne l a demande.
	Gate15 bool
	// Doublons : kill-events RETIRES parce qu ils repetent, dans le meme paquet, un enregistrement
	// deja lu a un bit anterieur ([assistScan.dedoublonner], lot J7.6, FK-6). Hors de `KillEvents`.
	Doublons int
	// Fil : la lecture de l assistant des morts sans kill-event au fil des evenements
	// ([decodeCtx.attachAssistsDuFil]). Ses publications sont HORS de `Attached` et de `Named`.
	Fil AssistFilStats
}
