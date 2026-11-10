package decfilm

// bots_sans_kill_feed.go — LE ROSTER DE BOTS D UN FILM QUI N A PAS DE KILL-FEED.
//
// `killsource.Decode` rend `ErrNoKillFeed` AVANT de construire son roster : sur un match sans aucun
// frag (`5da6fd30`, trois bots declares, zero kill), la cuisson recevait un resultat nil, donc aucun
// bot, et ses vies de bot restaient sans identite alors que BOT_METADATA les nomme. Ce fichier lit ce
// que le film declare deja, par la MEME lecture de paquets que `killsource`
// ([grammar.PaquetsBotMetadata]), et en rend un resultat MINIMAL : le roster de bots et le profil de
// depart, rien d autre. Aucune mort, aucune bijection : [killsource.Result.LineByLinePublishable] y
// reste faux, comme sur le resultat nil d avant.
//
// # UN JUMEAU DE `killsource/botmeta.go` ET `botmeta_equipe.go`, ET POURQUOI
//
// L agregat des declarations et le releve des equipes sont prives a `killsource`, dont le code ne
// bouge pas ici : toute modification de la couche fait monter `killsource.Rev`, qui perime les faits
// de TOUT le parc pour les six vies d un film. Ce fichier en reprend donc les regles, a l identique
// et pour les seules entrees dont un film sans kill-feed a besoin ; il vit hors couche (la facade n
// entre dans l empreinte d aucune), et le jour ou `killsource` exporte son agregat il se retire.
// Les regles : un bot se cle par (slot, bid) ; il est declare du premier paquet qui le porte au
// premier paquet COMPLET qui ne le porte plus ; un paquet dont le balayage ne retrouve pas `nbBots`
// entrees ne ferme rien ; une equipe ne se lit que dans un paquet dont la marche ferme, et deux
// paquets qui s en disputent la valeur la refusent. Un bot est NON EPINGLE quand son slot tombe sur
// un siege que la table du film nomme, ou hors des 32 indices.

import (
	"cmp"
	"slices"

	"github.com/JGtm/halo-infinite-film-decoder/internal/facts/killsource"
	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar"
	"github.com/JGtm/halo-infinite-film-decoder/internal/source"
)

// Bornes de `killsource` : un paquet annonce au plus 64 bots, un slot de bot tient dans 32 indices,
// et une equipe du jeu va de [grammar.TeamNone] a 8.
const (
	botsParPaquetMax = 64
	slotsDeBotMax    = 32
	equipeDeBotMax   = 8
)

// botLu : un bot que le film declare, avec ce que ses paquets en disent.
type botLu struct {
	slot, bid    int
	nom          string
	declarations []killsource.BotDeclaration
	equipe       int
	equipeLue    bool
}

// resultatDeBotsSansKillFeed rend le resultat minimal d un film dont le kill-feed manque : son roster
// de bots, ses bots non epingles et le bilan de leur lecture, sous le profil de depart de `killsource`
// (celui que la cuisson porte deja sur un resultat nil). Nil quand le film ne declare aucun bot : le
// resultat nil d avant reste alors exact.
func resultatDeBotsSansKillFeed(film *source.Film) *killsource.Result {
	if film == nil {
		return nil
	}
	// L erreur n accompagne que le refus de la table, deja porte par `table.Refusal` et sans effet ici :
	// sans table lue, aucune borne des humains n existe et `killsource` epingle tous les bots.
	table, _ := grammar.ScanFilmPlayerTable(film)
	lecture := grammar.LectureDesBotsPour(table.Build)
	paquets := grammar.PaquetsBotMetadata(film, lecture)
	bots, incomplets := agregerLesBots(paquets)
	bots, bilan := releverLesEquipes(paquets, bots, lecture)
	if len(bots) == 0 {
		return nil
	}
	roster := killsource.Roster{BotPaquetsIncomplets: incomplets, BotEquipes: bilan}
	contredit := contreditLaTable(table)
	for _, b := range bots {
		roster.Bots = append(roster.Bots, b.entree())
		if b.slot >= slotsDeBotMax || contredit(b.slot) {
			roster.UnpinnedBots = append(roster.UnpinnedBots, b.entree())
		}
	}
	return &killsource.Result{Roster: roster, ProfilCalibre: killsource.ProfilDeDepart()}
}

// entree rend la forme publiee d un bot.
func (b botLu) entree() killsource.BotEntry {
	e := killsource.BotEntry{Slot: b.slot, BotID: b.bid, Name: b.nom,
		Declarations: append([]killsource.BotDeclaration(nil), b.declarations...)}
	if b.equipeLue {
		v := b.equipe
		e.Team = &v
	}
	return e
}

// contreditLaTable rend le predicat « le slot d un bot tombe dans l espace des humains ET sur un siege
// qu un humain tient » : l espace des humains est le NOMBRE de sieges que la table du film nomme, et
// un siege sans gamertag ne contredit aucun bot. Sans table lue, aucun bot ne contredit rien.
func contreditLaTable(t grammar.FilmPlayerTable) func(slot int) bool {
	nommes := map[int]bool{}
	if t.Refusal == grammar.FilmTableRead {
		for _, s := range t.Seats {
			if s.Gamertag != "" {
				nommes[s.FilmIndex] = true
			}
		}
	}
	borne := len(nommes)
	return func(slot int) bool { return slot < borne && nommes[slot] }
}

// agregerLesBots agrege les bots des paquets BOT_METADATA, dans l ordre du film, avec leurs
// intervalles de declaration, puis les range par slot (tri stable). Rend aussi le nombre de paquets
// incomplets.
func agregerLesBots(paquets []grammar.PaquetBotMetadata) ([]botLu, int) {
	var bots []botLu
	incomplets := 0
	rang := map[[2]int]int{}       // (slot, bid) -> position dans bots
	ouverts := map[[2]int]uint64{} // declares au dernier paquet lu -> debut de leur declaration
	for _, p := range paquets {
		if p.NBots < 0 || p.NBots > botsParPaquetMax {
			continue
		}
		declares := make(map[[2]int]bool, len(p.Balayees))
		for _, e := range p.Balayees {
			k := [2]int{e.Slot, e.BotID}
			declares[k] = true
			if _, vu := rang[k]; !vu {
				rang[k] = len(bots)
				bots = append(bots, botLu{slot: e.Slot, bid: e.BotID, nom: e.Nom})
			}
			if _, ouvert := ouverts[k]; !ouvert {
				ouverts[k] = p.Instant
			}
		}
		if len(p.Balayees) != p.NBots {
			incomplets++ // un bot manque a la lecture n est pas un bot parti : rien ne se ferme
			continue
		}
		for _, k := range clesDeclarees(ouverts) {
			if !declares[k] {
				bots[rang[k]].declarations = append(bots[rang[k]].declarations,
					killsource.BotDeclaration{FromUS: ouverts[k], ToUS: p.Instant})
				delete(ouverts, k)
			}
		}
	}
	for _, k := range clesDeclarees(ouverts) {
		bots[rang[k]].declarations = append(bots[rang[k]].declarations, killsource.BotDeclaration{FromUS: ouverts[k]})
	}
	slices.SortStableFunc(bots, func(a, b botLu) int { return cmp.Compare(a.slot, b.slot) })
	return bots, incomplets
}

// clesDeclarees rend les cles des declarations ouvertes, triees : l ordre d iteration d une map est
// aleatoire, et les faits doivent se reproduire a l octet.
func clesDeclarees(m map[[2]int]uint64) [][2]int {
	out := make([][2]int, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	slices.SortFunc(out, func(a, b [2]int) int { return cmp.Or(cmp.Compare(a[0], b[0]), cmp.Compare(a[1], b[1])) })
	return out
}

// releverLesEquipes donne a chaque bot l equipe que ses paquets FERMES lui donnent quand ils s
// accordent, retire du roster les noms que la grammaire de l ecrivain ne lit pas, et rend le bilan.
// Sans largeur de personnalisation ou sans position de l equipe au profil du build, aucune equipe ne
// se lit et rien n est retire.
func releverLesEquipes(paquets []grammar.PaquetBotMetadata, bots []botLu, lecture grammar.LectureDesBots) ([]botLu, killsource.EquipesDesBots) {
	fragments := 0
	for _, p := range paquets {
		fragments += p.FragmentsEcartes
	}
	if !lecture.PersoConnue || !lecture.EquipeConnue {
		return bots, killsource.EquipesDesBots{Illisibles: len(bots), FragmentsEcartes: fragments,
			PersoInconnue:    !lecture.PersoConnue && len(bots) > 0,
			PositionInconnue: lecture.PersoConnue && len(bots) > 0}
	}
	parBot := map[[2]int]map[int]bool{}
	refuses := map[[2]int]bool{}
	horsBalayage := map[[2]int]bool{}
	var bilan killsource.EquipesDesBots
	for _, p := range paquets {
		noterLesEquipes(p, parBot, refuses, horsBalayage, &bilan)
	}
	bilan.FragmentsEcartes = fragments
	gardes := bots[:0]
	for _, b := range bots {
		k := [2]int{b.slot, b.bid}
		eqs := parBot[k]
		switch {
		case len(eqs) == 1:
			for v := range eqs {
				b.equipe, b.equipeLue = v, true
			}
			bilan.Lues++
		case len(eqs) > 1:
			bilan.Contradictoires++
		case refuses[k]:
			bilan.Illisibles++
		default:
			bilan.HorsGrammaire++
			continue
		}
		gardes = append(gardes, b)
	}
	bilan.EntreesHorsBalayage = len(horsBalayage)
	return gardes, bilan
}

// noterLesEquipes releve UN paquet : les entrees que la grammaire de l ecrivain y lit, confrontees a
// celles que le balayage des noms y a trouvees.
func noterLesEquipes(p grammar.PaquetBotMetadata, parBot map[[2]int]map[int]bool,
	refuses, horsBalayage map[[2]int]bool, bilan *killsource.EquipesDesBots) {
	if !p.Ferme {
		bilan.PaquetsNonFermes++
		for _, b := range p.Balayees {
			refuses[[2]int{b.Slot, b.BotID}] = true
		}
		return
	}
	noms := make(map[[2]int]string, len(p.Balayees))
	for _, b := range p.Balayees {
		noms[[2]int{b.Slot, b.BotID}] = b.Nom
	}
	for _, e := range p.Ecrites {
		k := [2]int{e.Slot, e.BotID}
		if nom, ok := noms[k]; !ok || nom != e.Nom {
			horsBalayage[k] = true
			continue
		}
		if !e.EquipeLue { // aucune position de l equipe pour ce build : refusee, jamais 0
			refuses[k] = true
			continue
		}
		if e.Equipe < grammar.TeamNone || e.Equipe > equipeDeBotMax {
			bilan.HorsDomaine++
			refuses[k] = true
			continue
		}
		if e.Jumeau != e.Equipe {
			bilan.JumeauxDiscordants++
		}
		if parBot[k] == nil {
			parBot[k] = map[int]bool{}
		}
		parBot[k][e.Equipe] = true
	}
}
