package filmdec

// killhealth.go — METRIQUE DE SANTE DU DECODAGE DE LA SOURCE DE DEGAT (arme par kill).
//
// A QUOI ELLE SERT. Le decodeur repose sur des hypotheses qui peuvent cesser d'etre vraies sans
// produire la moindre erreur : un catalogue de tags perime par une saison neuve, un roster plus
// grand que prevu, un mode de jeu inconnu. Aucune de ces situations ne leve d'exception — elles
// se paient en COUVERTURE SILENCIEUSE. Cette structure expose les quantites qui les rendent
// visibles, et les seuils qui disent quand s'en inquieter.
//
// LE MODE DE DEGRADATION LE PLUS DANGEREUX EST LE CATALOGUE PERIME. Le scan n'accepte un
// enregistrement que si son tag est une entree `jpt!` connue ; un tag introduit par une mise a
// jour de contenu est donc INVISIBLE, pas rejete. `TagOutOfCatalogue` est mesure par une sonde
// dont la porte du catalogue est retiree, restreinte aux candidats dont le couple exact existe au
// kill-feed : c'est la seule facon de voir ce que la porte cache.
//
// CE QUE CETTE STRUCTURE N'EST PAS : une mesure de qualite des etiquettes publiees. Les candidats
// inexpliques NE SORTENT PAS ; ils ne coutent rien au consommateur. C'est leur TAUX qui informe,
// par comparaison avec la distribution observee — jamais leur existence.

import "fmt"

// KillSourceHealth : l'etat de sante d'une passe de decodage, pour un film.
//
// Toutes les quantites sont des COMPTES BRUTS : aucun ratio n'est stocke, pour qu'une agregation
// multi-films additionne des numerateurs et des denominateurs, jamais des pourcentages.
type KillSourceHealth struct {
	Film string

	// Population consultee par le decodeur (marche + scan non redondant).
	Candidates int
	Published  int

	// Ventilation des candidats que rien ne publie. LES TROIS SE COMPTENT SUR `Candidates`, et
	// c'est ce qui autorise a en faire un ratio.
	UnexplainedPair   int // `v != k`, aucun couple exact au feed dans la fenetre
	UnexplainedSelf   int // `v == k` refuse par les discriminants, ou mort deja couverte
	UnexplainedBotIdx int // portant un indice epingle sur un bot, non resolu

	// OutOfRoster : dead-state de la marche dont un indice DEPASSE le roster retenu.
	//
	// IL N'EST PAS DANS `Candidates`, ET IL N'ENTRE DONC PAS DANS `UnexplainedRatio` — c'est le
	// defaut de denominateur corrige le 2026-07-27 (RE_LOG 7ter.73 (4e)). Le filtre de
	// credibilite de la marche exclut deja tout indice `>= nPlayers` : une ligne comptee ici a
	// ete ecartee AVANT de devenir un candidat. L'additionner a un numerateur dont le
	// denominateur ne la contient pas fabriquait un ratio entre deux populations differentes.
	// Mesure du defaut : sur le BTB, l'ancien ratio annoncait 27.0 % = 82/304 avec TROIS unites
	// de numerateur absentes des 304 ; le ratio corrige vaut 79/304 = 26.0 %. Sur les quatre
	// films de reference le compteur vaut 0, donc AUCUN seuil de cette page ne bouge — le defaut
	// etait invisible exactement la ou la distribution a ete tiree.
	//
	// CE QU'IL DEVIENT : un compteur INDEPENDANT, avec sa propre alerte dure dans [Alerts]. Il ne
	// perd rien a sortir du ratio ; il gagne d'etre lisible seul.
	OutOfRoster int

	// TagOutOfCatalogueWalk : enregistrements de la MARCHE apparies au kill-feed par le couple
	// exact et dont le tag est ABSENT du catalogue. Signature exacte d'une table perimee.
	//
	// SON BRUIT EST MESURE NUL SUR CINQ FILMS — pas << nul par construction >>, et la nuance est
	// mesuree (7ter.73 (4d)). L'argument structurel est fort (la marche n'a pas de porte de
	// catalogue : elle lit `SrcTag0` quel qu'il soit, et l'appariement par couple exact a deja
	// prouve que l'enregistrement est reel), mais sur le corpus de reference AUCUN enregistrement
	// de la marche ne porte un tag hors catalogue : la porte << couple exact >> n'a donc jamais eu
	// a rejeter quoi que ce soit. Le bruit est nul PAR ABSENCE DE CAS, et cet appariement herite
	// du plancher de sur-ajustement de 9.6 % (7ter.53 (3)).
	//
	// SON POINT AVEUGLE est mesure : voir le bloc qui suit les seuils.
	TagOutOfCatalogueWalk int
	// TagOutOfCatalogueScan : complement pour les morts que la marche n'atteint pas, mesure par
	// une sonde a porte de catalogue retiree, RESTREINTE aux morts non couvertes. Vaut zero par
	// construction quand la couverture est complete — ce n'est donc jamais le compteur principal.
	TagOutOfCatalogueScan int

	DeathsReal    int // couples REELS (couple fabrique retire, 7ter.66)
	DeathsCovered int
}

// UnexplainedTotal : la somme des ventilations QUI SE COMPTENT SUR `Candidates`. `OutOfRoster`
// n'y figure pas, et c'est deliberé : voir le commentaire de ce champ.
func (h KillSourceHealth) UnexplainedTotal() int {
	return h.UnexplainedPair + h.UnexplainedSelf + h.UnexplainedBotIdx
}

// UnexplainedRatio : part des candidats consultes que rien ne publie, dans [0,1].
//
// UN SEUL DENOMINATEUR, ET IL CONTIENT TOUT LE NUMERATEUR. C'est la regle que RE_LOG 7ter.66 a
// payee d'une section entiere et que 7ter.73 (4e) a fait appliquer ici.
func (h KillSourceHealth) UnexplainedRatio() float64 {
	if h.Candidates <= 0 {
		return 0
	}
	return float64(h.UnexplainedTotal()) / float64(h.Candidates)
}

// CoverageRatio : morts couvertes / couples REELS, dans [0,1]. Le denominateur est NOMME dans le
// nom du champ : `DeathsReal`. Les autres denominateurs (morts du feed, morts de l'API) ne sont
// pas dans cette structure parce qu'ils exigent une source externe — les melanger ici produirait
// exactement l'ambiguite que 7ter.66 interdit.
func (h KillSourceHealth) CoverageRatio() float64 {
	if h.DeathsReal <= 0 {
		return 0
	}
	return float64(h.DeathsCovered) / float64(h.DeathsReal)
}

// SEUILS — ILS SORTENT DE LA DISTRIBUTION OBSERVEE, PAS D'UNE INTUITION.
//
// Mesure du 2026-07-27, mode `hyhealth`, CINQ films (section 7ter.72 du journal) :
//
//	film        mode        inexpliques   couverture   hors roster   tag hors catalogue
//	000d5950    Fiesta          7.0 %       100.0 %         0                0
//	9b191a7f    standard        6.3 %       100.0 %         0                0
//	78919882    Forge          11.6 %       100.0 %         0                0
//	fccc61cd    Fiesta         16.9 %       100.0 %         0                0
//	4f77afc1    BTB:CTF        26.0 %        76.5 %         3                0
//
// DEUX DE CES TAUX ONT BAISSE LE 2026-07-27 AVEC RE_LOG 7ter.79, ET LES SEUILS N ONT PAS ETE
// RE-DERIVES. `UnexplainedBotIdx` comptait comme inexpliques les dead-states portant un indice de
// bot en TUEUR ; ils sont desormais publies (`killsource.OriginBotKiller`), donc expliques :
// 9.4 -> 6.3 % sur `9b191a7f` (3 unites) et 17.8 -> 16.9 % sur `fccc61cd` (1 unite). Les deux
// films SANS bot sont inchanges, et le BTB aussi — aucune de ses morts orphelines ne s y resout.
//
// POURQUOI NE PAS RE-DERIVER : un seuil re-calcule a chaque amelioration du decodeur SUIT le
// decodeur au lieu de le SURVEILLER, et rendrait << hors domaine >> un simple retour a l etat
// d hier. `UnexplainedWarnRatio` reste donc a 18.0 %, desormais 1.1 point AU-DESSUS du maximum
// observe. C est un CHOIX, il est ecrit, et il rend le seuil legerement plus permissif que sa
// regle de derivation d origine.
//
// LA COLONNE << inexpliques >> DU BTB A CHANGE DE VALEUR LE 2026-07-27, et pas le decodeur : elle
// valait 27.0 % tant que `OutOfRoster` etait au numerateur sans etre au denominateur (7ter.73
// (4e)). CE CORRECTIF-LA n'a PAS touche les quatre films a 8 joueurs — leur compteur hors roster
// vaut zero — donc les seuils ci-dessous, tires de leur MAXIMUM, sont ceux d'avant la correction.
// (Deux d'entre eux ont bouge depuis, pour une raison SANS RAPPORT : voir le bloc 7ter.79
// ci-dessus. Les seuils n'ont pas suivi, deliberement.)
//
// LA SERIE DE REFERENCE EST CELLE DES QUATRE FILMS A 8 JOUEURS — le seul domaine ou la
// publication ligne par ligne est validee (la bijection du BTB a une marge NULLE, 7ter.53 (4)).
// Le BTB sert de CONTROLE POSITIF de domaine : il doit sortir HORS DOMAINE, et il en sort par
// trois criteres independants. Regle de derivation, ecrite pour qu'un futur ajustement soit un
// choix et non une derive :
//
//	HORS DOMAINE = au-dessus du MAXIMUM observe sur la serie de reference. Le film n'est pas
//	               forcement casse ; il n'est plus dans le domaine mesure, et ses lignes se
//	               ponderent en consequence.
//	ALERTE       = au-dessus du DOUBLE de ce maximum, ou l'un des deux compteurs a distribution
//	               nulle est non nul.
const (
	// UnexplainedWarnRatio : maximum observe sur la serie de reference AU MOMENT DE LA
	// DERIVATION (fccc61cd, 17.8 %). Le maximum vaut 16.9 % depuis 7ter.79 ; le seuil n a
	// deliberement PAS ete re-derive (voir le bloc ci-dessus).
	UnexplainedWarnRatio = 0.180
	// UnexplainedAlertRatio : le double. Le BTB (26.0 %) est HORS DOMAINE sans etre en ALERTE
	// par ce critere — c'est voulu : son taux s'explique par 38 candidats a indice de bot non
	// resolus, pas par un decodage casse.
	UnexplainedAlertRatio = 0.360
	// CoverageWarnRatio : plancher observe sur la serie de reference. Il vaut EXACTEMENT 1.00
	// (371/371 couples REELS), donc toute mort manquee fait sortir du domaine.
	//
	// CE SEUIL EST PORTEUR, PAS DECORATIF — ET C'EST LE POINT AVEUGLE CI-DESSOUS QUI LE PROUVE.
	// Il a longtemps ete presente comme << strict par choix >> et pose en question ouverte ;
	// 7ter.73 (4a) a renverse ce jugement en le mesurant. NE PAS L'ASSOUPLIR sans avoir d'abord
	// donne un filet de rechange au mode que `TagOutOfCatalogueWalk` ne voit pas.
	CoverageWarnRatio = 1.00
)

// POINT AVEUGLE MESURE DU COMPTEUR PRINCIPAL — LIRE AVANT DE TOUCHER AUX SEUILS.
//
// `TagOutOfCatalogueWalk` ne voit un tag perime QUE SI LA MARCHE PORTE CE TAG. Un tag dont la
// seule occurrence publiee est servie par le SCAN ne le fait pas sonner, et ce n'est pas une
// inconnue : c'est mesure (RE_LOG 7ter.73 (4a), la 21e ablation).
//
//	ablation de `0c64b43f` (1 ligne sur 78919882, servie par le SCAN SEUL) :
//	   TagOutOfCatalogueWalk = 0        -> AUCUNE ALERTE
//	   couverture 99 -> 98 = 99.0 %     -> verdict << HORS DOMAINE MESURE >> et rien de plus
//	   sonde a T4 relache : 0 -> 3      -> elle LE VOIT, mais elle est exclue des alertes (bruit)
//
// CONSEQUENCE, ET C'EST LA RAISON D'ETRE DE `CoverageWarnRatio = 1.00` : dans ce mode de
// degradation, LE PLANCHER DE COUVERTURE EST LE SEUL FILET. Il n'y a pas de severite a assouplir
// ici, il y a un compteur qui ne voit pas tout et un second qui rattrape.
//
// PORTEE JUSTE DU CONTROLE POSITIF, a citer telle quelle : *20 ablations sur 20 font sonner
// l'alerte, sur des tags QUE LA MARCHE PORTE AUSSI* (les cinq plus frequents de chaque film) —
// et non *20/20 sur un catalogue perime*. Le regime scan-seul n'est pas couvert par ce chiffre.
//
// CE QUE CELA COUTE EN ANCRES DE VERITE TERRAIN : 5 des 30 ancres Theater sont servies par le
// SCAN SEUL (4 sur `9b191a7f` — 01:00, 03:41, 04:50, 05:12 — et 1 sur `78919882`, 09:18) ; sous
// ablation l'une d'elles tombe. Le << 11.2x plus robuste >> de l'hybride vaut en LIGNES PUBLIEES,
// il ne vaut PAS en ancres (7ter.73 (4b)(4c)).

// LES TROIS VERDICTS, NOMMES. Ils sont compares par des appelants et par des tests : une chaine
// litterale recopiee ailleurs se desynchroniserait en silence le jour ou l'un d'eux change.
const (
	// VerdictNominal : le film ressemble a ceux sur lesquels le decodeur a ete mesure.
	VerdictNominal = "NOMINAL"
	// VerdictHorsDomaine : il n'y ressemble plus. IL N'EST PAS CASSE — ses lignes se ponderent.
	VerdictHorsDomaine = "HORS DOMAINE MESURE"
	// VerdictAlerte : une condition dure de [KillSourceHealth.Alerts] est franchie.
	VerdictAlerte = "ALERTE"
)

// Verdict : trois etats et pas un de plus. Un verdict n'est PAS un jugement sur les etiquettes
// publiees, c'est un jugement sur le DOMAINE : le film ressemble-t-il a ceux sur lesquels le
// decodeur a ete mesure ?
func (h KillSourceHealth) Verdict() string {
	if len(h.Alerts()) > 0 {
		return VerdictAlerte
	}
	if h.UnexplainedRatio() > UnexplainedWarnRatio || h.CoverageRatio() < CoverageWarnRatio {
		return VerdictHorsDomaine
	}
	return VerdictNominal
}

// Alerts : les conditions dures, chacune avec le chiffre qui la declenche et ce qu'elle signifie.
//
// `TagOutOfCatalogueScan` N'Y FIGURE PAS, ET C'EST MESURE : la sonde a T4 relache a un rapport
// signal/hasard de 1.10 a 1.72 selon le film (10 078 appariements contre ~9 200 a horloge decalee
// sur le BTB). Elle informe, elle n'alerte pas.
//
// Seul le compteur de la MARCHE alerte, pour deux raisons qu'il faut citer avec leur portee :
// son bruit est MESURE NUL sur cinq films (et non nul par construction, cf. son champ), et son
// controle positif est passe 20 fois sur 20 SUR LES TAGS QUE LA MARCHE PORTE. Hors de cette
// population il a un point aveugle mesure, et c'est `CoverageRatio` qui prend le relais.
func (h KillSourceHealth) Alerts() []string {
	var out []string
	if h.TagOutOfCatalogueWalk > 0 {
		out = append(out, fmt.Sprintf(
			"%d enregistrement(s) de la marche portent un tag HORS catalogue : la table `jpt!` est PERIMEE, regenerer (recette : paquet damagetag)",
			h.TagOutOfCatalogueWalk))
	}
	if h.OutOfRoster > 0 {
		out = append(out, fmt.Sprintf(
			"%d dead-state(s) a tag `jpt!` valide portent un indice hors du roster retenu : un participant n'est pas compte (bots non declares ? roster plus grand ?)",
			h.OutOfRoster))
	}
	if h.UnexplainedRatio() > UnexplainedAlertRatio {
		out = append(out, fmt.Sprintf(
			"taux d'inexpliques %.1f%% au-dela du DOUBLE du maximum observe (%.1f%%)",
			100*h.UnexplainedRatio(), 100*UnexplainedWarnRatio))
	}
	return out
}

// ExpvarPair : un compteur, pret pour `observability.AddInt`.
type ExpvarPair struct {
	Name  string
	Value int64
}

// ExpvarPairs : la publication, alignee sur ADR 0009 — compteurs entiers, snake_case,
// prefixe de categorie. Aucun ratio n'est publie : les ratios se calculent a la lecture, sinon
// une agregation multi-films moyennerait des pourcentages.
//
// Le paquet ne depend PAS de `internal/observability` : c'est l'appelant qui cable, ce qui garde
// `filmdec` sans dependance interne et cette fonction testable sans expvar.
func (h KillSourceHealth) ExpvarPairs() []ExpvarPair {
	return []ExpvarPair{
		{"killsource_candidates_total", int64(h.Candidates)},
		{"killsource_published_total", int64(h.Published)},
		{"killsource_unexplained_pair", int64(h.UnexplainedPair)},
		{"killsource_unexplained_self", int64(h.UnexplainedSelf)},
		{"killsource_unexplained_botidx", int64(h.UnexplainedBotIdx)},
		// PAS de prefixe `unexplained_` ici : ce compteur ne se compte pas sur la meme population
		// que les trois precedents, et un nom qui les rassemble inviterait a les additionner.
		{"killsource_out_of_roster", int64(h.OutOfRoster)},
		{"killsource_tag_out_of_catalogue_walk", int64(h.TagOutOfCatalogueWalk)},
		{"killsource_tag_out_of_catalogue_scan", int64(h.TagOutOfCatalogueScan)},
		{"killsource_deaths_real", int64(h.DeathsReal)},
		{"killsource_deaths_covered", int64(h.DeathsCovered)},
	}
}
