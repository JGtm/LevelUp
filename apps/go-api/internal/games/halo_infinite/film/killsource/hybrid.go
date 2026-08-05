package killsource

// hybrid.go — L HYBRIDE : LA MARCHE DECIDE, LE SCAN RATTRAPE.
//
// CINQ TEMPS, dans cet ordre, chacun n intervenant que sur les instants que les precedents
// n ont pas couverts :
//
//	1. MARCHE            couple exact au kill-feed, sur les dead-states CREDIBLES.
//	2. SCAN, RATTRAPAGE  meme critere, sur les candidats du scan que la marche N A PAS produits.
//	3. SOURCE AUTO-INFLIGEE  `victime == tueur` + les deux discriminants, apparie sur la VICTIME
//	                     SEULE. LA MARCHE Y A AUSSI LA PRIORITE.
//	4. MORT DE BOT       population NEUVE. Voie SCAN seule, et ce n est pas un choix : la marche
//	                     borne ses indices au roster credible et n en produit pas.
//	5. MORT PAR UN BOT   seconde population NEUVE, SYMETRIQUE de la precedente (RE_LOG 7ter.79).
//	                     Le feed porte la MORT, pas le kill. Les deux voies y participent, dans
//	                     l ordre de priorite habituel.
//
// LES DEUX POPULATIONS DE BOT NE SE CHEVAUCHENT PAS, ET CE N EST PAS UNE PRECAUTION MAIS UNE
// PROPRIETE : la 4e part des KILLS du feed que rien ne consomme, la 5e des MORTS du feed que rien
// ne consomme. Un evenement du feed ne peut pas etre les deux. Elles sont neanmoins publiees sous
// deux origines DISTINCTES, parce que ce qu elles empruntent au feed n est pas la meme moitie.
//
// LA PREFERENCE EST LEGITIME PARCE QUE LES DEUX VOIES NE SE CONTREDISENT JAMAIS : sur les
// instants ou les deux repondent, ACCORD 334, DESACCORD 0, et zero mort a plus d un candidat en
// concurrence. L hybride PREFERE, il n arbitre pas — le mecanisme d arbitrage existe, il n a
// jamais eu a servir.
//
// HYPOTHESE MESUREE COMME FAUSSE, a ne pas reposer : << la marche d abord fait remonter le gate
// (b) GLOBAL >>. Ecart mesure : +0.00 point sur les quatre films, et la raison est structurelle —
// le gate (b) est une propriete de l ENSEMBLE des enregistrements consultes, pas de l ORDRE dans
// lequel on les lit ; les 346 enregistrements partages sont les MEMES objets au MEME bit.
//
// CE QUE L HYBRIDE APPORTE VRAIMENT, et c est mesure : (1) le cout est LOCALISE ET ETIQUETE —
// 98.2 % pour la marche, 78.4 % pour le rattrapage, chaque ligne portant sa voie ; (2) une
// ROBUSTESSE 11.2 fois superieure a un catalogue perime ; (3) un gain de JUSTESSE — sous
// l hybride la ligne servie par la marche publie le bon tag sans dependre d aucun filtre.
//
// AUCUNE FUSION DE VALEURS, aucune moyenne, aucun vote : si les deux voies decodaient un tag
// different au meme instant, la marche gagnerait et l ecart serait COMPTE, jamais lisse.

import "sort"

// sourcedCandidate : un candidat, avec la voie qui l a produit.
type sourcedCandidate struct {
	candidate
	path Path
}

// pass : l etat d une passe hybride.
type pass struct {
	ctx      *decodeCtx
	all      []sourcedCandidate
	mult     map[multKey]int
	byTime   map[int]Kill
	walk     PathStats
	scan     PathStats
	selfWalk PathStats
	selfScan PathStats
	botStats PathStats
	// botKillerStats : les morts INFLIGEES PAR un bot (temps 5).
	botKillerStats PathStats
	// botUsed : les positions de candidat consommees par les DEUX temps de bot. Elle sert au
	// seul comptage des inexpliques : un candidat a indice de bot qui a servi n en est pas un.
	botUsed   map[[3]int]bool
	redundant int
	noBit     int
	agree     int
	disagree  int
	multiCand int

	unexpPair int
	unexpSelf int
	unexpBot  int
}

// population : les deux voies ramenees a une liste unique, DANS L ORDRE DE PRIORITE.
//
// Un candidat du scan est REDONDANT s il porte la meme position (chunk, paquet, bit) qu un
// enregistrement de la marche : c est alors le MEME enregistrement physique, lu deux fois.
//
// LE POINT DE METHODE. Les redondants sont retires de la population du SCAN, jamais de celle de
// la marche — sinon on choisirait le denominateur qui arrange. Et ils sont retires QUELLE QUE
// SOIT leur issue : retirer les seuls doublons qui echouent serait une manipulation de
// denominateur.
func (p *pass) population(walkCands []candidate, scanCands []candidate, noBit int) {
	p.noBit = noBit
	at := map[[3]int]bool{}
	for _, w := range walkCands {
		if w.bit >= 0 {
			at[[3]int{w.chunk, w.pidx, w.bit}] = true
		}
		p.all = append(p.all, sourcedCandidate{w, PathWalk})
	}
	for _, s := range scanCands {
		if at[[3]int{s.chunk, s.pidx, s.bit}] {
			p.redundant++
			continue
		}
		p.all = append(p.all, sourcedCandidate{s, PathScan})
	}
	cs := make([]candidate, 0, len(p.all))
	for _, a := range p.all {
		cs = append(cs, a.candidate)
	}
	// La multiplicite se calcule sur l UNION : un champ fixe lu comme un dead-state se repete
	// au meme bit quelle que soit la voie qui le lit. Elle serait aveugle a la moitie de la
	// population si elle ne voyait qu une voie.
	p.mult = multiplicity(cs)
	p.ctx.mult = p.mult
}

// runStrong : temps 1 et 2 en une seule boucle — l ordre de `all` SUFFIT a donner la priorite.
//
// Le gate (b) est compte pour TOUS les candidats de chaque voie, y compris ceux dont l instant
// etait deja pris : sinon la seconde voie serait jugee sur une population mutilee par la
// premiere, et son chiffre ne voudrait plus rien dire.
func (p *pass) runStrong() {
	for _, cd := range p.all {
		if cd.victim == cd.killer || p.ctx.isBotSide(cd.candidate) {
			continue
		}
		st := &p.walk
		if cd.path == PathScan {
			st = &p.scan
		}
		st.Population++
		e := p.ctx.matchExact(cd.candidate)
		if e == nil {
			p.unexpPair++
			continue
		}
		st.Matched++
		if _, seen := p.byTime[e.timeMS]; seen {
			continue
		}
		st.Published++
		p.byTime[e.timeMS] = p.ctx.buildKill(killDraft{timeMS: e.timeMS, victim: e.victim,
			killer: e.killer, inFeed: true, origin: OriginCredit}, cd)
	}
}

// runSelfSource : temps 3 — les morts dont la SOURCE APPARTIENT A LA VICTIME.
//
// LE FILTRE `victime != tueur` REPOSAIT SUR UNE PREMISSE INCOMPLETE. << `victime == tueur` est
// une signature de dechet >> est VRAI mais ne dit pas << donc tout `victime == tueur` est du
// dechet >> : c est AUSSI l encodage legitime d une mort dont la source appartient a la victime
// (roquette tiree trop pres, baril lance trop pres, chute). Deux phenomenes partagent une
// signature, et le filtre jetait les deux.
//
// LES DEUX DISCRIMINANTS SONT STRUCTURELS ET MESURES : la MULTIPLICITE de position (les faux se
// repetent au meme bit, les vrais sont a des bits varies) et la MAGNITUDE du tag. Verifie 8/8 en
// Theater.
func (p *pass) runSelfSource() {
	if !p.ctx.opts.SelfSource {
		return
	}
	for _, cd := range p.all {
		if cd.victim != cd.killer {
			continue
		}
		st := &p.selfWalk
		if cd.path == PathScan {
			st = &p.selfScan
		}
		st.Population++
		e := p.ctx.matchVictim(cd.candidate)
		if e == nil || !p.ctx.selfSourceOK(cd.candidate, p.mult) {
			p.unexpSelf++
			continue
		}
		st.Matched++
		if _, seen := p.byTime[e.timeMS]; seen {
			p.unexpSelf++
			continue
		}
		st.Published++
		// Le feed credite un AUTRE joueur : on publie SON credit tel quel, et on leve le
		// drapeau de divergence. Masquer l un ou l autre detruirait l information.
		p.byTime[e.timeMS] = p.ctx.buildKill(killDraft{timeMS: e.timeMS, victim: e.victim,
			killer: e.killer, inFeed: true, origin: OriginSelfSource, diverges: true}, cd)
	}
}

// runBots : temps 4 — la premiere population NEUVE, la mort DU bot.
func (p *pass) runBots() {
	for _, m := range p.ctx.resolveBotDeaths() {
		p.botStats.Population++
		if !m.found {
			continue
		}
		p.botStats.Matched++
		k := [3]int{m.cand.chunk, m.cand.pidx, m.cand.bit}
		if p.botUsed[k] {
			continue
		}
		p.botUsed[k] = true
		p.botStats.Published++
		// `inFeed = false` : le kill est au feed, la MORT n y est pas. La victime vient du
		// roster de replication, pas du kill-feed — et le consommateur doit pouvoir le savoir.
		p.byTime[m.event.timeMS] = p.ctx.buildKill(killDraft{timeMS: m.event.timeMS,
			victim: p.ctx.roster.nameOf(m.cand.victim), killer: m.event.killer,
			origin: OriginBot}, sourcedCandidate{m.cand, PathScan})
	}
}

// runBotKillers : temps 5 — la seconde population NEUVE, la mort infligee PAR un bot.
//
// `inFeed = true`, ET C EST LA MOITIE INVERSE DU TEMPS 4 : le kill-feed porte bien cette MORT
// (la victime est humaine, elle a un XUID) ; ce qu il ne porte pas, c est le KILL. Le nom du
// tueur vient donc du roster de replication — [OriginBotKiller] est ce qui le dit au lecteur, et
// c est la seule chose qui le dise.
//
// L INSTANT PUBLIE EST CELUI DU KILL-FEED, jamais celui du candidat : c est le meme choix que
// partout ailleurs dans ce fichier, et il garde les instants publies homogenes.
//
// DEUX GARDES, ET AUCUN N EST DECORATIF. Sans le premier, une mort orpheline tombant a la
// milliseconde d une mort deja publiee l ECRASERAIT en silence ; sans le second — le meme que le
// temps 4 — un SEUL dead-state servirait DEUX morts orphelines et publierait deux fois la meme
// source. Les deux valent zero sur le corpus, et ce zero est FIGE par l egalite
// `botKiller.Population == Matched == Published` exigee dans la non-regression : si l un d eux se
// declenchait, le test tomberait au lieu de laisser passer un doublon.
func (p *pass) runBotKillers() {
	for _, m := range p.ctx.resolveBotKillerDeaths(p.all) {
		p.botKillerStats.Population++
		if !m.found {
			continue
		}
		p.botKillerStats.Matched++
		k := [3]int{m.cand.chunk, m.cand.pidx, m.cand.bit}
		if _, deja := p.byTime[m.event.timeMS]; deja || p.botUsed[k] {
			continue
		}
		p.botUsed[k] = true
		p.botKillerStats.Published++
		p.byTime[m.event.timeMS] = p.ctx.buildKill(killDraft{timeMS: m.event.timeMS,
			victim: m.event.victim, killer: p.ctx.roster.nameOf(m.cand.killer),
			inFeed: true, origin: OriginBotKiller}, m.cand)
	}
}

// countUnexplainedBot : les candidats a indice de BOT que NI le temps 4 NI le temps 5 n ont
// servis. Il se compte APRES les deux, sinon il compterait comme inexplique ce que le temps
// suivant explique — exactement le genre de compteur qui reste vrai en apparence tout en
// mesurant autre chose.
func (p *pass) countUnexplainedBot() {
	for _, cd := range p.ctx.scanCands {
		if !p.ctx.isBotSide(cd) {
			continue
		}
		if !p.botUsed[[3]int{cd.chunk, cd.pidx, cd.bit}] {
			p.unexpBot++
		}
	}
}

// concordance : les deux voies se contredisent-elles ?
//
// PORTEE, et elle est essentielle : la comparaison se fait sur le SCAN ENTIER, redondances
// COMPRISES — pas sur la population amputee que l hybride consulte. Comparer apres avoir retire
// les redondants ne mesurerait rien, puisque le redondant est PRECISEMENT l enregistrement que
// les deux voies partagent.
//
// `disagree` doit valoir ZERO. S il bouge, l hybride n est plus une PREFERENCE mais un
// ARBITRAGE, et il faut le documenter comme tel.
func (p *pass) concordance(walkCands []candidate) {
	byT := map[int]*atInstant{}
	for _, cd := range walkCands {
		p.noteInstant(byT, cd, PathWalk)
	}
	for _, cd := range p.ctx.scanCands {
		p.noteInstant(byT, cd, PathScan)
	}
	for _, b := range byT {
		if b.n > 2 || (b.n == 2 && (!b.hasWalk || !b.hasScan)) {
			p.multiCand++ // plusieurs candidats DISTINCTS, pas un simple doublon de voie
		}
		if !b.hasWalk || !b.hasScan {
			continue
		}
		if b.walk.tag == b.scan.tag && b.walk.cat == b.scan.cat {
			p.agree++
		} else {
			p.disagree++
		}
	}
}

// atInstant : ce que les deux voies disent d un meme instant.
type atInstant struct {
	walk, scan       candidate
	hasWalk, hasScan bool
	n                int
}

// noteInstant : range un candidat apparie sous son instant, avec la voie qui l a produit.
func (p *pass) noteInstant(byT map[int]*atInstant, cd candidate, path Path) {
	if cd.victim == cd.killer {
		return
	}
	e := p.ctx.matchExact(cd)
	if e == nil {
		return
	}
	b := byT[e.timeMS]
	if b == nil {
		b = &atInstant{}
		byT[e.timeMS] = b
	}
	b.n++
	if path == PathWalk {
		b.walk, b.hasWalk = cd, true
		return
	}
	b.scan, b.hasScan = cd, true
}

// stats : ce que la passe a mesure, mis en forme pour le consommateur.
func (p *pass) stats(w *walkResult) Stats {
	return Stats{
		Walk: p.walk, Scan: p.scan,
		SelfWalk: p.selfWalk, SelfScan: p.selfScan,
		Bot:               p.botStats,
		BotKiller:         p.botKillerStats,
		Redundant:         p.redundant,
		NoBit:             p.noBit,
		Agree:             p.agree,
		Disagree:          p.disagree,
		MultiCandidate:    p.multiCand,
		PacketsWithEvents: w.withEv,
		PacketsLocated:    w.located,
	}
}

// kills : les lignes publiees, triees par instant.
func (p *pass) kills() []Kill {
	out := make([]Kill, 0, len(p.byTime))
	for _, k := range p.byTime {
		out = append(out, k)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].TimeMS < out[j].TimeMS })
	return out
}

// run : la passe complete.
func (c *decodeCtx) run() *pass {
	walkCands, noBit := c.walkRes.candidates()
	sortCandidates(walkCands)
	p := &pass{ctx: c, byTime: map[int]Kill{}, botUsed: map[[3]int]bool{}}
	p.population(walkCands, c.scanCands, noBit)
	p.runStrong()
	p.runSelfSource()
	p.runBots()
	p.runBotKillers()
	p.countUnexplainedBot()
	p.concordance(walkCands)
	return p
}
