package killsource

// hybrid_bots.go — LES DEUX TEMPS DE BOT DE L HYBRIDE (temps 4 et 5, cf. `hybrid.go`), et le NOM
// qu un bot porte A L INSTANT d une ligne publiee.
//
// # UN INDICE, PLUSIEURS BOTS : LE NOM SE LIT A L INSTANT
//
// BOT_METADATA declare parfois plusieurs bots sur le MEME indice de replication, l un apres
// l autre (une succession dans le meme siege, cf. [roster.pinBots]), et il DATE chaque declaration
// ([BotDeclaration]). Le roster ne garde qu un nom par indice pour tout le film — le dernier
// declare —, parce que la bijection et le gate des indices raisonnent par indice. Une ligne
// publiee, elle, a un instant : le bot qu elle nomme est celui que BOT_METADATA declare sur cet
// indice A CET INSTANT ([decodeCtx.nomDuBotA]). Hors de toute declaration, le nom du roster reste
// celui de la ligne : aucune regle ne decide la ou le film se tait.

// runBots : temps 4 — la premiere population NEUVE, la mort DU bot. La premiere passe publie ce que
// chaque kill trouve au premier candidat de sa fenetre ; le complement ([decodeCtx.completerLesMortsDeBot])
// n ajoute que des lignes, aux kills qu elle laisse sans ligne et avec les dead-states qu elle n a pas
// servis.
func (p *pass) runBots() {
	ms := p.ctx.resolveBotDeaths()
	trouves, aCompleter := make(map[int]bool, len(ms)), make(map[int]bool, len(ms))
	for _, m := range ms {
		p.botStats.Population++
		if !m.found {
			aCompleter[m.event.timeMS] = true
			continue
		}
		trouves[m.event.timeMS] = true
		p.botStats.Matched++
		if !p.publierMortDeBot(m) {
			aCompleter[m.event.timeMS] = true
		}
	}
	sansLigne := func(t int) bool { _, deja := p.byTime[t]; return aCompleter[t] && !deja }
	for _, m := range p.ctx.completerLesMortsDeBot(ms, sansLigne, copieDesServis(p.botUsed)) {
		if !m.found {
			continue
		}
		if !trouves[m.event.timeMS] {
			p.botStats.Matched++
		}
		p.publierMortDeBot(m)
	}
}

// publierMortDeBot publie la mort de bot `m` — sauf un instant deja publie que la mort de bot n a
// pas le droit de remplacer ([autoSurCoupleFabrique]), et un nom de remplissage. Rend faux pour un
// dead-state deja servi, seul refus que le complement peut reprendre avec un autre dead-state.
func (p *pass) publierMortDeBot(m botMatch) bool {
	k := positionDuCandidat(m.cand)
	if p.botUsed[k] {
		return false
	}
	remplace := false
	if prev, deja := p.byTime[m.event.timeMS]; deja {
		if !autoSurCoupleFabrique(m, prev) {
			p.collisionsBot++ // FK-4 (lot J7.4) : un instant publie ne se reecrit jamais
			return true
		}
		remplace = true
	}
	victime, publiable := p.nomPubliableA(m.cand.victim, m.event.timeMS)
	if !publiable {
		return true
	}
	if remplace {
		p.autoSurFabriqueRemplacees++
		p.retirerLigne(m.event.timeMS) // une ligne publiee, une provenance
	}
	p.botUsed[k] = true
	if m.fab {
		p.fantomes[m.event.timeMS] = true // le couple recolle etait une mort de bot : fantome
	}
	p.botStats.Published++
	p.noterLigne(m.event.timeMS, m.parLaFenetre, &p.appar.BotFenetre)
	// `inFeed = false` : le kill est au feed, la MORT n y est pas. La victime vient du roster de
	// replication, pas du kill-feed — et le consommateur doit pouvoir le savoir. Une source qui
	// appartient au bot lui-meme leve la divergence, comme au temps 3 : le feed credite un autre
	// joueur que celui que le dead-state designe.
	p.byTime[m.event.timeMS] = p.ctx.buildKill(killDraft{timeMS: m.event.timeMS,
		victim: victime, killer: m.event.killer,
		origin: OriginBot, diverges: m.sourceDeLaVictime}, sourcedCandidate{m.cand, PathScan})
	return true
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
// source. Les morts que la premiere passe laisse sans ligne se completent ensuite, avec les seuls
// dead-states qu aucun temps n a servis : le complement ajoute des lignes, il n en retire aucune.
func (p *pass) runBotKillers() {
	ms := p.ctx.resolveBotKillerDeaths(p.all)
	trouves := make(map[int]bool, len(ms))
	var restantes []feedEvent
	for _, m := range ms {
		p.botKillerStats.Population++
		if !m.found {
			restantes = append(restantes, m.event)
			continue
		}
		trouves[m.event.timeMS] = true
		p.botKillerStats.Matched++
		if !p.publierMortParUnBot(m) {
			restantes = append(restantes, m.event)
		}
	}
	for _, m := range p.ctx.apparierMortsParUnBot(restantes, p.all, copieDesServis(p.botUsed)) {
		if !m.found {
			continue
		}
		if !trouves[m.event.timeMS] {
			p.botKillerStats.Matched++
		}
		p.publierMortParUnBot(m)
	}
}

// publierMortParUnBot publie la mort `m` infligee par un bot — sauf un instant deja publie, un
// dead-state deja servi et un nom de remplissage. Rend faux pour un dead-state deja servi a un
// instant libre, seul refus que le complement peut reprendre avec un autre dead-state.
func (p *pass) publierMortParUnBot(m botKillerMatch) bool {
	k := positionDuCandidat(m.cand.candidate)
	if _, deja := p.byTime[m.event.timeMS]; deja {
		return true
	}
	if p.botUsed[k] {
		return false
	}
	tueur, publiable := p.nomPubliableA(m.cand.killer, m.event.timeMS)
	if !publiable {
		return true
	}
	p.botUsed[k] = true
	p.botKillerStats.Published++
	p.noterLigne(m.event.timeMS, m.parLaFenetre, &p.appar.BotFenetre)
	p.byTime[m.event.timeMS] = p.ctx.buildKill(killDraft{timeMS: m.event.timeMS,
		victim: m.event.victim, killer: tueur,
		inFeed: true, origin: OriginBotKiller}, m.cand)
	return true
}

// nomPubliableA : [pass.nomPubliable], au nom que l indice porte A L INSTANT `ms` de la ligne
// ([decodeCtx.nomDuBotA]).
func (p *pass) nomPubliableA(i, ms int) (string, bool) {
	if nom, ok := p.ctx.nomDuBotA(i, ms); ok {
		return nom, true
	}
	return p.nomPubliable(i)
}

// nomDuBotA : le nom du bot que BOT_METADATA declare sur l indice `i` a l instant `ms` (millisecondes
// de killsource), suffixe [BotSuffix] compris. Faux quand l indice n est pas epingle sur un bot ou
// n en porte qu un, ou qu aucune declaration ne couvre l instant : le nom du roster s applique alors,
// tel quel. Les bots d un indice epingle le sont tous ([roster.pinBots] decide par indice).
func (c *decodeCtx) nomDuBotA(i, ms int) (string, bool) {
	if ms < 0 || !c.roster.isBotIndex(i) {
		return "", false
	}
	var declares []bot
	for _, b := range c.roster.bots.Bots {
		if b.Slot == i {
			declares = append(declares, b)
		}
	}
	if len(declares) < 2 {
		return "", false
	}
	t := c.film.tsBase + uint64(ms)*1000 // l horloge du film, celle des declarations
	for _, b := range declares {
		for _, d := range b.declarations {
			if d.FromUS <= t && (d.ToUS == 0 || t < d.ToUS) {
				return b.Name + BotSuffix, true
			}
		}
	}
	return "", false
}
