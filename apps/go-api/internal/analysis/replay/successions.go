package replay

// successions.go — LA FERMETURE PAR RELAIS : le remplaçant hérite des vies anonymes qui
// suivent son arrivée.
//
// LE CAS QU'ELLE RÉSOUT (retour user 2026-09-02, le match témoin 1b2d9e08) : un joueur
// quitte, un BOT le remplace au même siège. Le fil des morts ne porte AUCUNE mort de bot
// (mesuré : 77 morts, 0 sans xuid — le parseur ancre sur les xuid), donc ni le nommage par
// mort ni les fermetures A/B ne peuvent lui attribuer ses vies : son pion restait invisible
// et sa fiche vide. Or la BASE date l'arrivée du remplaçant À LA SECONDE
// (`joined_in_progress` + `first_joined_time`, axe du match) — et le pont sait déjà caler
// cet axe sur l'horloge du film (`DeathOffsetMS`).
//
// LA RÈGLE EST UNE CHAÎNE DE FENÊTRES, à candidat unique — le standard des fermetures :
//
//	première vie   l'UNIQUE track anonyme qui naît dans [arrivée − 2 s, arrivée + 20 s]
//	vies suivantes l'UNIQUE track anonyme qui naît dans [fin de la précédente + 2 s,
//	               fin + 25 s] — la fenêtre d'une réapparition (8-10 s mesurés) avec marge
//
// Zéro candidat = la chaîne s'arrête (le bot a survécu jusqu'au bout, ou le film ne réplique
// plus). Deux candidats = CONTESTÉ, la chaîne s'arrête aussi : on ne tranche pas. Les vies
// non couvertes restent anonymes — une caméra de fin de partie ne peut pas devenir un bot
// par accident, elle ne naît pas dans une fenêtre de réapparition chaînée depuis l'arrivée.
//
// PLUSIEURS RELAIS DANS UN MATCH (« ça peut arriver des dizaines de fois ») : les
// successions se traitent par instant d'arrivée croissant, chaque vie réclamée sort du
// pot commun — deux chaînes ne peuvent pas revendiquer la même vie.

import "log/slog"

// Succession est un RELAIS lu dans la base : un remplaçant (bot) arrive à cet instant de
// l'axe du match. L'assembleur (replaybuild) la construit depuis les faits de participation
// et le roster BOT_METADATA du décodage killsource — ce paquet-ci n'ouvre aucune base.
type Succession struct {
	// BotName est le nom d'affichage du remplaçant, suffixe « [bot] » compris — la même
	// forme que Track.Bot et RosterEntry.Name.
	BotName string
	// SwitchMatchMS est l'instant d'arrivée sur l'axe du MATCH (le même que Death.TimeMS).
	SwitchMatchMS int64
}

// Fenêtres de la chaîne, en microsecondes de film.
const (
	successionLeadUS   = 2_000_000  // tolérance AVANT l'arrivée déclarée (granularité API : la seconde)
	successionFirstUS  = 20_000_000 // l'arrivant apparaît vite ; au-delà, on n'affirme rien
	successionGapMinUS = 2_000_000  // une réapparition ne suit jamais la mort de moins de 2 s
	successionGapMaxUS = 25_000_000 // réapparition mesurée 8-10 s ; 25 s couvre les modes lents
)

// attributeSuccessions pose le nom du remplaçant sur les vies que la chaîne lui rend.
// `deathOffsetMS` est le calage axe-match -> axe-film du pont ; sans lui (0 apparié), rien
// n'est attribué — les deux horloges ne se parlent pas.
func attributeSuccessions(tracks []Track, successions []Succession,
	origin, step uint64, deathOffsetMS int64, offsetMatches int) {
	if len(successions) == 0 || offsetMatches == 0 {
		return
	}
	claimed, halted := 0, 0
	for _, s := range successions {
		switchUS := (s.SwitchMatchMS + deathOffsetMS) * 1000
		from, to := switchUS-successionLeadUS, switchUS+successionFirstUS
		for {
			i := uniqueAnonymousIn(tracks, origin, step, from, to)
			if i < 0 {
				halted++
				break
			}
			tracks[i].Bot = s.BotName
			claimed++
			endUS := int64(origin) + int64(tracks[i].EndFrame)*int64(step)
			from, to = endUS+successionGapMinUS, endUS+successionGapMaxUS
		}
	}
	slog.Info("rejeu : fermetures par relais",
		"successions", len(successions), "viesAttribuees", claimed, "chainesArretees", halted)
}

// uniqueAnonymousIn rend l'index de l'UNIQUE track anonyme qui naît dans [fromUS, toUS],
// -1 sinon (aucune, ou plus d'une — les deux arrêtent la chaîne).
func uniqueAnonymousIn(tracks []Track, origin, step uint64, fromUS, toUS int64) int {
	found := -1
	for i := range tracks {
		if tracks[i].XUID != "" || tracks[i].Bot != "" {
			continue
		}
		startUS := int64(origin) + int64(tracks[i].StartFrame)*int64(step)
		if startUS < fromUS || startUS > toUS {
			continue
		}
		if found >= 0 {
			return -1 // contesté : on ne tranche pas
		}
		found = i
	}
	return found
}
