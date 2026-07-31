package killsource

// assist.go — L ASSISTANT : LA TROISIEME PERSONNE D UNE MORT.
//
// L API ne rend qu un TOTAL d assists par joueur et par match. Le film, lui, dit QUI a assiste
// QUEL kill. C est donc une donnee que l API ne fournit PAS.
//
// CE QUI VALIDE CES ASSISTANTS N EST PAS LE TAUX D ATTACHEMENT (voir plus bas) MAIS LE MULTISET :
// les comptes par joueur, tries et PRIVES DES NOMS, confrontes a ceux de l API. Deux raisons
// STRUCTURELLES rendent cette confrontation non auto-confirmante — la bijection ne fait que
// renommer, et le multiset ignore les noms ; l assistant n entre PAS dans l objectif que la
// bijection optimise ([quadScore] ne regarde que le couple tueur/victime). Le test est
// `TestAssistMultisetParJoueur` ; il REMPLACE l ancienne validation par joueur, qui comparait des
// comptes NOMMES — donc exactement ce que la bijection produit.
//
// D OU IL VIENT, ET IL NE VIENT PAS D OU VIENT LA SOURCE : l assistant est un champ du
// KILL-EVENT, dans la liste d evenements en tete de paquet. La source du degat, elle, vit dans le
// DEAD-STATE de la victime, dans la boucle de records. Ce sont deux structures differentes du
// meme paquet. L assistant n est donc PAS une troisieme verite : c est un attribut de la verite
// KILL-FEED — le jeu credite un tueur, et il declare un assistant a cote.
//
// L APPARIEMENT EST CONTRAINT : un kill-event n est attache a une mort publiee que si SON COUPLE
// (tueur, victime), passe par la bijection indice -> joueur, est EXACTEMENT celui du kill-feed a
// cet instant.
//
// CE QUE LE TAUX D ATTACHEMENT N EST PAS, ET LA PHRASE QUI ETAIT ICI L AFFIRMAIT : ce n est PAS
// une mesure du fait que les indices du kill-event vivent dans le meme espace que ceux du
// dead-state. `bijection.go` fixe la permutation en MAXIMISANT le nombre de candidats dont le
// couple se retrouve au kill-feed ([quadScore]) — mot pour mot le critere d attachement. Un taux
// eleve est donc la VALEUR D UNE FONCTION OBJECTIF OPTIMISEE, pas une mesure independante. Le
// raisonnement etait CIRCULAIRE et il est retire, pas nuance.
//
// CE QUE L ATTACHEMENT RESTE, ET C EST TOUT CE QU ON LUI DEMANDE ICI : un SELECTEUR DE POPULATION.
// Il decide sur QUELLES morts un kill-event a le droit de parler. Ni l assistant, ni aucune des
// deux parts de degats n entrent dans l objectif optimise — le selecteur est donc non circulaire
// POUR CE QU ON LUI DEMANDE, et c est la seule portee qu il faut lui preter.
//
// TROIS REJETS CONNUS. Le troisieme est ne de la correction d ordre des champs (RE_LOG 7ter.75
// (2)) : ce que 7ter.24ter appelait << assistant == tueur >> portait en fait sur la VICTIME, et
// les deux rejets coexistent desormais parce qu ils ne disent pas la meme chose.
//
//	[AssistRejectSelf]    le champ ne designe personne d autre que le TUEUR
//	[AssistRejectVictim]  il designe la VICTIME
//	[AssistRejectRoster]  il designe une entite qui n est pas un joueur du film
//
// Ils sont COMPTES, pas dissimules : un rejet qui ne se compte pas est une perte silencieuse.
//
// PERIMETRE DE PUBLICATION — DECISION PRISE, ET ELLE EST CELLE DE LA SOURCE DU DEGAT : l assistant
// et les deux parts de degats ne se publient LIGNE PAR LIGNE que sous [Result.LineByLinePublishable].
// Aucune porte separee n est ajoutee : la mesure BTB la justifie a elle seule — 62 assistants
// nommes pour 122 a l API (51 %), contre 17/17 et 29/29 en Arena. Ce n est pas un decodage plus
// mauvais, c est la marge de bijection NULLE du BTB : les lignes y sont exactes en AGREGAT et
// fausses individuellement.

import "sort"

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

// DamageShare : une part de degats en POURCENTAGE ENTIER, avec son etat de mesure.
//
// TROIS ETATS, JAMAIS DEUX — exactement comme [Assist.Known], et pour la meme raison : NULL n est
// jamais zero. `Known == false` veut dire QU ON N A PAS MESURE, jamais << 0 % de degats >>. Une
// part de 0 % mesuree existe et se dit `{Pct: 0, Known: true}`.
//
// AUCUN PLAFOND A 100, ET C EST UNE DECISION FONDEE SUR UNE MESURE : plafonner rejetterait 1.7 %
// des kill-events attaches a de VRAIES morts nommees, avec des valeurs mesurees jusqu a 228.
// L hypothese naturelle est le degat EXCEDENTAIRE (une roquette qui retire plus que la vie
// restante) ; elle n est PAS etablie. Ce qui l est : une valeur > 100 est une DONNEE REELLE, pas
// une lecture ratee, et la jeter serait fabriquer une absence de mesure.
//
// RESERVE, HERITEE DE [killEventFields] ET A PORTER AVEC TOUTE CITATION DE CES DEUX NOMBRES : le
// chemin de donnees entre le kill-event du film et les champs `KillerPercentageDamageDone` /
// `AssistantPercentageDamageDone` du modele de recap N EST PAS DEMONTRE. Quatre jambes
// convergentes, PAS une chaine d appels prouvee.
type DamageShare struct {
	// Pct : la part, en pourcentage entier. A ne lire que si `Known`.
	Pct int
	// Known : la part a-t-elle ete MESUREE sur cette mort ?
	Known bool
}

// Les deux motifs de refus.
const (
	// AssistRejectSelf : le champ designe le tueur lui-meme.
	AssistRejectSelf = "assistant==tueur"
	// AssistRejectVictim : le champ designe la victime. C EST LE REJET QUE LE CHANTIER DE
	// RETRO-INGENIERIE APPELAIT << assistant==tueur >> : il lisait les deux premiers champs dans
	// le mauvais ordre (voir [readKillEvent]).
	AssistRejectVictim = "assistant==victime"
	// AssistRejectRoster : l indice ne designe aucun joueur du roster retenu.
	AssistRejectRoster = "hors-roster"
)

// RESERVE ARITHMETIQUE SUR `9b191a7f`. **TOUJOURS OUVERTE, ET DEUX DE SES UNITES SONT RENTREES
// AVEC RE_LOG 7ter.79.** Elle est consignee ici parce qu elle porte sur la distinction que TOUT LE
// SCHEMA protege : << on ne sait pas >> / << pas d assistant >>.
//
//	l API compte                                        30 assistances
//	le decodeur attache                                 87 morts sur 90
//	il y lit                                            24 assistants NOMMES
//	et                                                  63 << pas d assistant >> (mesures)
//	restent                                              3 morts NON LUES = les 3 morts DU bot
//
// Sous l hypothese << au plus un assistant par mort >>, le maximum atteignable vaut 24 + 3 = 27,
// STRICTEMENT INFERIEUR a 30. TROIS ASSISTANCES MANQUANTES QUI NE PEUVENT PAS VENIR DES MORTS
// RATEES. Il ne reste donc que deux issues, et aucune n est tranchee :
//
//	(a) une mort porte DEUX assistants — l hypothese du champ simple tombe (voir [Assist.Extra],
//	    qui est structurellement aveugle a ce cas s il ne se manifeste pas par un second
//	    kill-event attache) ;
//	(b) certains << pas d assistant >> sont FAUX — et c est alors la distinction elle-meme qui
//	    fuit, en fabriquant des << 0 assist >> jamais observes.
//
// SON HISTORIQUE EST ECRIT ICI POUR QUE PERSONNE NE RECITE UN CHIFFRE PERIME — c est ce qui est
// arrive au << 31 contre 30 >> de 7ter.24ter, recopie des semaines dans un brief :
//
//	7ter.76  22 nommes + 6 morts non lues = 28 < 30                                     ecart 2
//	7ter.77  les 3 kills DU BOT portent un kill-event credible dont DEUX nomment un assistant,
//	         mais aucun ne s attache : plafond 22 + 2 + 3 = 27. 3e ISSUE, un assistant peut
//	         etre LU sans etre PUBLIABLE.                                                ecart 3
//	7ter.79  la 3e issue est FERMEE, pas deplacee : ces 3 morts sont publiees
//	         ([OriginBotKiller]) et leurs 2 assistants avec. Plafond INCHANGE.           ecart 3
//
// L ECART DE 3 NE VIENT PLUS D UNE POPULATION EXCLUE, il se LOCALISE : Chocoboflor 6 pour 9,
// Madina97294 3 pour 5, Duke748biposto 2 pour 3 — LES ISSUES (a) ET (b) RESTENT ENTIERES. Et il ne
// se generalise pas : sur les DEUX films ou TOUTES les morts sont attachees, le decodage reproduit
// le total de l API A L UNITE (17/17 et 29/29). Sur `fccc61cd`, 7ter.79 n a rien comble — la mort
// infligee par `bid(7.0)` NE NOMME AUCUN assistant.

// killEventRec : un kill-event localise, avec sa position et l instant de son paquet.
type killEventRec struct {
	ms          int
	chunk, pidx int
	bit         int // position du code R(7), pour dedoublonner
	fields      killEventFields
	chain       int
}

// assistScan : ce que la passe de kill-events a produit et ce qu elle a coute.
type assistScan struct {
	recs []killEventRec
	// gate15 : l etat runtime retenu pour ce film (voir [pickGate15]).
	gate15 bool
	// packetsWithEvents, packetsWithKill : denominateurs de la passe.
	packetsWithEvents, packetsWithKill int
}

// minChain : profondeur de chaine exigee pour retenir un candidat. Trois evenements — le seuil
// mesure. NE PAS l abaisser sans re-mesurer : c est lui qui elimine 99.4 % des faux.
const minChain = 3

// maxChainProbe : borne de la marche avant. Au-dela le critere ne discrimine plus rien et le
// cout devient inutile.
const maxChainProbe = 12

// scanKillEvents : localise les kill-events de tous les paquets a evenements.
func scanKillEvents(f *film) *assistScan {
	s := &assistScan{}
	s.gate15 = pickGate15(f)
	for i := range f.t0 {
		p := &f.t0[i]
		if !hasEvents(p) {
			continue
		}
		s.packetsWithEvents++
		recs := killEventsIn(p.payload, s.gate15)
		if len(recs) > 0 {
			s.packetsWithKill++
		}
		ms := f.ms(p)
		for _, r := range recs {
			r.ms, r.chunk, r.pidx = ms, p.chunk, p.idx
			s.recs = append(s.recs, r)
		}
	}
	sort.Slice(s.recs, func(i, j int) bool { return s.recs[i].ms < s.recs[j].ms })
	return s
}

// killEventsIn : les kill-events d un paquet. Le motif R(7) == 85 precede d un bit de
// continuation a 1 est un GENERATEUR de candidats ; c est la CHAINE qui tranche.
func killEventsIn(pl []byte, gate15 bool) []killEventRec {
	var out []killEventRec
	nb := len(pl) * 8
	for x := 1; x+8 <= nb; x++ {
		if bitAt(pl, x-1) != 1 || bitsN(pl, x, 7) != killEventCode {
			continue
		}
		r := &evReader{pl: pl, bp: x + 7}
		if !evPresence(r, killEventCode) {
			continue
		}
		k := readKillEvent(pl, r.bp)
		if !killEventPlausible(k) {
			continue
		}
		n := evChainLen(pl, k.end, gate15, maxChainProbe)
		if n < minChain {
			continue
		}
		out = append(out, killEventRec{bit: x, fields: k, chain: n})
	}
	return out
}

// pickGate15 : `gate15` est un etat RUNTIME du jeu, absent du flux de bits — il ne se lit pas, il
// se TRANCHE par film. Critere : celui des deux qui enchaine le plus d evenements sur les
// paquets a evenements. Ce n est pas un reglage libre : les deux valeurs sont essayees et la
// mesure decide, sur une quantite (longueur de chaine) qui ne regarde AUCUN resultat publie.
func pickGate15(f *film) bool {
	score := [2]int{}
	for i := range f.t0 {
		p := &f.t0[i]
		if !hasEvents(p) || len(p.payload) < 64 {
			continue
		}
		for g := 0; g < 2; g++ {
			score[g] += len(killEventsIn(p.payload, g == 1))
		}
	}
	return score[1] > score[0]
}

// attachAssists : accroche a chaque mort publiee l assistant declare par le kill-event dont le
// COUPLE correspond exactement. Rend les compteurs de la passe.
//
// UNE MORT SANS KILL-EVENT ATTACHE GARDE `Known = false`. Elle n est PAS publiee comme << sans
// assistant >> : le decodeur ne sait pas, et le dire est la seule sortie honnete.
func (c *decodeCtx) attachAssists(kills []Kill, s *assistScan) AssistStats {
	var st AssistStats
	st.KillEvents = len(s.recs)
	st.Gate15 = s.gate15
	used := make([]bool, len(s.recs))
	for i := range kills {
		k := &kills[i]
		k.Assist.Index = -1
		hits := c.killEventsFor(k, s, used)
		if len(hits) == 0 {
			continue
		}
		// LE SURPLUS D ASSISTANTS SE PORTE SUR LA LIGNE, pas seulement dans l agregat : c est ce
		// qui rend `assist_extra_count` alimentable en base. Il n est calculable que quand
		// PLUSIEURS kill-events sont attaches — un kill-event seul n expose qu un emplacement.
		pick := hits[0]
		if len(hits) > 1 {
			st.Multi++
			var desaccord bool
			pick, desaccord = pickAssistHit(s.recs, hits)
			if desaccord {
				st.AssistFieldDisagree++
			}
			if n := distinctAssists(s.recs, hits); n > 1 {
				k.Assist.Extra = n - 1
				st.AssistMulti++
				st.AssistExtraTotal += n - 1
			}
		}
		used[pick] = true
		st.Attached++
		c.fillAssist(k, s.recs[pick].fields, &st)
	}
	return st
}

// pickAssistHit : parmi les kill-events attaches a une MEME mort, celui qui PARLE — le premier
// qui porte un champ assistant, a defaut le premier tout court. Rend aussi le DESACCORD : au moins
// un enregistrement porteur ET au moins un muet.
//
// POURQUOI ELLE EXISTE : `hits[0]` etait pris sans regarder. Deux enregistrements attaches a la
// meme mort, le premier sans champ assistant et le second en nommant un, faisaient publier
// << PAS D ASSISTANT, MESURE >> — une mesure d absence FABRIQUEE A PARTIR D UN DESACCORD, et
// aucun compteur ne la voyait.
//
// CE QU ELLE N EST PAS : une mesure. Le corpus ne tranche pas, il ne contient AUCUN desaccord
// (RE_LOG 7ter.77). C est une DECISION, fondee sur le seul argument disponible — la grammaire
// n expose qu UN emplacement d assistant par enregistrement, donc un porteur dit strictement plus
// qu un muet, et aucun mecanisme connu n EFFACERAIT un assistant reel d un second enregistrement.
// ET C EST POURQUOI LE COMPTEUR EST OBLIGATOIRE : une decision non surveillee redevient une
// hypothese dormante, un desaccord doit etre VISIBLE et pas seulement resolu. Son zero sur les
// quatre films est MESURE, pas construit — `TestAssistDesaccordAsymetrique` fabrique le cas et
// exige qu il bouge.
func pickAssistHit(recs []killEventRec, hits []int) (pick int, desaccord bool) {
	pick = hits[0]
	porteur, muet := false, false
	for _, j := range hits {
		if recs[j].fields.assist >= 0 {
			if !porteur {
				pick = j
			}
			porteur = true
			continue
		}
		muet = true
	}
	return pick, porteur && muet
}

// distinctAssists : nombre d assistants DISTINCTS et exploitables parmi des kill-events attaches
// a la meme mort.
//
// PORTEE, ET ELLE EST PLUS ETROITE QUE << UN ASSISTANT OU PLUSIEURS >> : le denominateur de cette
// mesure est LES KILL-EVENTS ATTACHES, jamais les morts. Deux kill-events attaches a la meme mort
// sont necessaires pour qu elle puisse valoir plus de 1 — un kill-event seul n expose qu un
// emplacement d assistant. Elle tranche donc << deux ENREGISTREMENTS nomment-ils deux assistants
// differents >>, pas << cette mort porte-t-elle deux assistants >>.
func distinctAssists(recs []killEventRec, hits []int) int {
	seen := map[int]bool{}
	for _, j := range hits {
		f := recs[j].fields
		if f.assist >= 0 && f.assist != f.killer {
			seen[f.assist] = true
		}
	}
	return len(seen)
}

// killEventsFor : les indices des kill-events dont le couple correspond exactement a la mort.
func (c *decodeCtx) killEventsFor(k *Kill, s *assistScan, used []bool) []int {
	var hits []int
	for j := range s.recs {
		if used[j] {
			continue
		}
		r := &s.recs[j]
		if dt := k.TimeMS - r.ms; dt < -tolMS || dt > tolMS {
			continue
		}
		if c.roster.nameOf(r.fields.victim) != k.Victim {
			continue
		}
		if k.Feed.Present && c.roster.nameOf(r.fields.killer) != k.Feed.Killer {
			continue
		}
		hits = append(hits, j)
	}
	return hits
}

// fillAssist : remplit l assistant ET LES DEUX PARTS DE DEGATS, en comptant CHAQUE issue.
//
// LA REGLE DE PUBLICATION DES DEUX PARTS TIENT EN TROIS LIGNES, ET CHACUNE EST FONDEE SUR UNE
// MESURE :
//
//	(1) elles sont NON MESUREES quand aucun kill-event n est attache — cette fonction n est alors
//	    pas appelee, et `DamageShare.Known` reste faux ;
//	(2) la part du TUEUR est publiee des qu un kill-event est attache ;
//	(3) la part de l ASSISTANT n existe QUE si le CHAMP assistant est PRESENT. Sans lui, le bloc
//	    porte une CONSTANTE PAR FILM qui ne dit rien du kill (149, 70, 20, 197 selon le film) — et
//	    442 des 595 lignes attachees, soit 74 %, sont dans ce cas. Publier la valeur brute
//	    remplirait les TROIS QUARTS de la colonne de bruit, avec des nombres parfaitement
//	    credibles (20 est un pourcentage plausible). Sur les quatre films de reference, les
//	    compteurs figes dans `assist_test.go` donnent le meme ordre de grandeur : 286 morts sans
//	    assistant sur 370 attachees, 77 %.
//
// PIEGE POUR LE COLLECTEUR : un assistant REFUSE (`Rejected` non vide) laisse `Assist.Name` VIDE
// alors que `AssistDamage.Known` vaut vrai — le champ etait present, la part est mesuree, c est
// son PORTEUR qu on refuse de nommer. `persist.KillEventInsert.AssistDamagePct` doit alors rester
// nil : le persister refuse une part sans assistant nomme, et il a raison. Les trois rejets valent
// zero sur le corpus de reference ; ce cas est theorique, pas hypothetique.
func (c *decodeCtx) fillAssist(k *Kill, f killEventFields, st *AssistStats) {
	k.Assist.Known = true
	k.Assist.Index = f.assist
	k.KillerDamage = DamageShare{Pct: int(f.killerPct), Known: true}
	if k.KillerDamage.Pct > 100 {
		st.KillerPctOver100++
	}
	if f.assist >= 0 {
		k.AssistDamage = DamageShare{Pct: int(f.assistPct), Known: true}
		if k.AssistDamage.Pct > 100 {
			st.AssistPctOver100++
		}
	}
	switch {
	case f.assist < 0:
		st.NoAssist++
	case f.assist == f.killer:
		k.Assist.Rejected = AssistRejectSelf
		st.RejectedSelf++
	case f.assist == f.victim:
		k.Assist.Rejected = AssistRejectVictim
		st.RejectedVictim++
	case c.roster.nameOf(f.assist) == "?":
		k.Assist.Rejected = AssistRejectRoster
		st.RejectedRoster++
	default:
		k.Assist.Name = c.roster.nameOf(f.assist)
		st.Named++
	}
	if f.flag != 0 {
		st.FlagSet++
	}
}

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
	// AssistExtraTotal : somme des [Assist.Extra] de la passe. Meme portee que `AssistMulti`, mais
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
	// Gate15 : l etat runtime retenu pour ce film.
	Gate15 bool
}
