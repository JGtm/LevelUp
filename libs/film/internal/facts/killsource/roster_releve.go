package killsource

// roster_releve.go — UN INDICE DE BOT N EST PLUS UN BOT APRES LA FIN DE SA DERNIERE DECLARATION.
//
// BOT_METADATA date chaque declaration d un bot ([BotDeclaration]) : du premier paquet qui le
// declare au premier paquet complet qui ne le declare plus. La fin est lue a la frame ; le debut
// n est qu une borne ([roster.botTientA]). Quand le dernier bot d un indice est parti, un humain
// peut reprendre l indice : la lecture du motif de xuid ([lireIndexParMotif]) le nomme alors a cet indice, et
// cette lecture, qui ne date rien, ne contredit pas le bot — elle nomme son SUCCESSEUR.
//
// LA RELEVE : sur un indice epingle par BOT_METADATA dont toutes les declarations sont closes, le
// nom que le motif lit (s il n est epingle nulle part ailleurs) devient le joueur de l indice a
// partir de la fin de la derniere declaration ([roster.pinUnMotif], compte `MotifReleve`). Avant
// cette fin, l indice reste au bot.
//
// LES LECTURES DATEES : toute decision prise sur un indice a un instant passe par [roster.isBotIndexA],
// [roster.nomEpingleA] et [roster.nameOfA]. Un indice de bot apres la fin de sa derniere
// declaration et sans releve n est ni un bot ni un nom lu : la lecture se tait, et aucune ligne de
// bot (mort DU bot ou mort infligee PAR le bot) ne s y publie.

// releve : le successeur humain d un indice de bot.
type releve struct {
	// finUS : la fin de la derniere declaration BOT_METADATA de l indice, horloge du film.
	finUS uint64
	// pos : la position du successeur dans `names`.
	pos int
}

// finDesDeclarations rend la fin de la derniere declaration des bots declares sur `slot`, et faux
// quand l un d eux est encore declare au dernier paquet du film (`ToUS == 0`) ou qu aucune
// declaration n est connue : l indice n est alors jamais libre.
func (r *roster) finDesDeclarations(slot int) (uint64, bool) {
	var fin uint64
	vu := false
	for _, b := range r.bots.Bots {
		if b.Slot != slot {
			continue
		}
		for _, d := range b.declarations {
			if d.ToUS == 0 {
				return 0, false
			}
			fin, vu = max(fin, d.ToUS), true
		}
	}
	return fin, vu
}

// relever pose `nom` en successeur du bot de l indice `idx` quand les declarations de ce bot sont
// toutes closes. Faux quand l indice n est pas un indice de bot, que le bot est declare jusqu a
// la fin du film, ou que le nom est deja epingle a un autre indice (un joueur, un indice).
func (r *roster) relever(idx int, nom string, posDuNom map[string]int, prises map[int]bool) bool {
	if !r.isBotIndex(idx) {
		return false
	}
	fin, closes := r.finDesDeclarations(idx)
	if !closes {
		return false
	}
	pos, connu := posDuNom[nom]
	if connu && prises[pos] {
		return false
	}
	if !connu {
		r.names = append(r.names, nom)
		pos = len(r.names) - 1
		posDuNom[nom] = pos
	}
	prises[pos] = true
	if r.releves == nil {
		r.releves = map[int]releve{}
	}
	r.releves[idx] = releve{finUS: fin, pos: pos}
	r.table.MotifReleve++
	return true
}

// instantUS : l instant `ms` de killsource sur l horloge du film, celle des declarations.
func (r *roster) instantUS(ms int) uint64 { return r.tsBase + uint64(max(ms, 0))*1000 }

// successeurA rend la position du successeur qui tient l indice `i` a l instant `ms`.
func (r *roster) successeurA(i, ms int) (int, bool) {
	rl, ok := r.releves[i]
	if !ok || r.instantUS(ms) < rl.finUS {
		return 0, false
	}
	return rl.pos, true
}

// botTientA : un bot tient-il encore l indice `i` a l instant `ms` ? Oui jusqu a la fin de la derniere
// declaration de ses bots, et toujours quand l une reste ouverte au dernier paquet du film.
//
// LE DEBUT D UNE DECLARATION N EST QU UNE BORNE : le paquet qui la porte peut etre celui de la tete
// du chunk suivant l arrivee du bot (botmeta.go), et le parc porte des bots tues une seconde avant
// leur premiere declaration. Avant elle, l indice est donc deja au bot ; c est la FIN qui est lue a
// la frame (premier paquet complet qui ne le declare plus).
func (r *roster) botTientA(i, ms int) bool {
	fin, closes := r.finDesDeclarations(i)
	return !closes || r.instantUS(ms) < fin
}

// isBotIndexA : l indice `i` est-il tenu par un bot a l instant `ms` ? Epingle par BOT_METADATA ET
// tenu a cet instant ([roster.botTientA]).
func (r *roster) isBotIndexA(i, ms int) bool {
	return r.isBotIndex(i) && r.botTientA(i, ms)
}

// nomEpingleA : [roster.nomEpingle] a l instant `ms`. Le successeur apres la releve ; rien sur un
// indice de bot apres sa derniere declaration sans successeur (la lecture se tait) ; le nom epingle sinon.
func (r *roster) nomEpingleA(i, ms int) (string, bool) {
	if pos, ok := r.successeurA(i, ms); ok {
		return r.names[pos], true
	}
	if r.isBotIndex(i) && !r.botTientA(i, ms) {
		return "", false
	}
	return r.nomEpingle(i)
}

// nameOfA : [roster.nameOf] a l instant `ms` — le successeur apres la releve, le nom de la
// bijection sinon.
func (r *roster) nameOfA(i, ms int) string {
	if pos, ok := r.successeurA(i, ms); ok {
		return r.names[pos]
	}
	return r.nameOf(i)
}
