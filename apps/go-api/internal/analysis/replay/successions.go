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
	// FilmIndex est l'indice de RÉPLICATION du remplaçant (BOT_METADATA) : c'est lui qui
	// DIFFÉRENCIE deux remplaçants simultanés — leurs TIRS sont indexés, et un tir de cet
	// indice tombé dans une vie candidate la corrobore (cf. la levée de contestation).
	FilmIndex int
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
// n'est attribué — les deux horloges ne se parlent pas. `fire` (les tirs, indexés par
// joueur de film) sert à LEVER une contestation : quand deux vies candidates naissent dans
// la même fenêtre (deux remplaçants simultanés), celle qui CONTIENT un tir de l'indice du
// remplaçant est la sienne — un tir est une lecture, pas une devinette. Deux candidates
// tirées, ou aucune : la chaîne s'arrête.
func attributeSuccessions(tracks []Track, successions []Succession,
	origin, step uint64, deathOffsetMS int64, offsetMatches int, fire []FireEventRef) {
	if len(successions) == 0 || offsetMatches == 0 {
		return
	}
	claimed, halted, liftedByFire := 0, 0, 0
	for _, s := range successions {
		switchUS := (s.SwitchMatchMS + deathOffsetMS) * 1000
		from, to := switchUS-successionLeadUS, switchUS+successionFirstUS
		for {
			i, lifted := candidateIn(tracks, origin, step, from, to, s.FilmIndex, fire)
			if i < 0 {
				halted++
				break
			}
			if lifted {
				liftedByFire++
			}
			tracks[i].Bot = s.BotName
			claimed++
			endUS := int64(origin) + int64(tracks[i].EndFrame)*int64(step)
			from, to = endUS+successionGapMinUS, endUS+successionGapMaxUS
		}
	}
	slog.Info("rejeu : fermetures par relais", "successions", len(successions),
		"viesAttribuees", claimed, "chainesArretees", halted, "contestationsLeveesParTir", liftedByFire)
}

// candidateIn rend l'index de la track anonyme retenue dans [fromUS, toUS] : l'unique
// candidate, ou — à plusieurs — l'unique candidate CORROBORÉE par un tir de l'indice du
// remplaçant. -1 sinon (le second retour dit si un tir a départagé).
func candidateIn(tracks []Track, origin, step uint64, fromUS, toUS int64,
	filmIndex int, fire []FireEventRef) (int, bool) {
	var cands []int
	for i := range tracks {
		if tracks[i].XUID != "" || tracks[i].Bot != "" {
			continue
		}
		startUS := int64(origin) + int64(tracks[i].StartFrame)*int64(step)
		if startUS < fromUS || startUS > toUS {
			continue
		}
		cands = append(cands, i)
	}
	if len(cands) == 1 {
		return cands[0], false
	}
	if len(cands) < 2 {
		return -1, false
	}
	// UN TIR NE VOTE QUE S'IL TOMBE DANS EXACTEMENT UNE CANDIDATE : deux vies qui se
	// chevauchent contiennent les mêmes instants, et un tir couvert par les deux ne dit
	// rien. Des votes pour DEUX candidates différentes = contesté (un tir mal daté ne doit
	// pas pouvoir mentir).
	voted := -1
	for _, f := range fire {
		if f.FilmIndex != filmIndex {
			continue
		}
		holder := -1
		for _, i := range cands {
			if !trackContains(tracks[i], origin, step, f.TimestampUS) {
				continue
			}
			if holder >= 0 {
				holder = -2 // couvert par deux candidates : ce tir ne vote pas
				break
			}
			holder = i
		}
		if holder < 0 {
			continue
		}
		if voted >= 0 && voted != holder {
			return -1, false
		}
		voted = holder
	}
	if voted >= 0 {
		return voted, true
	}
	return -1, false
}

// trackContains dit si l'instant tombe dans la fenêtre de la vie.
func trackContains(tr Track, origin, step uint64, tsUS uint64) bool {
	fromUS := origin + uint64(tr.StartFrame)*step
	toUS := origin + uint64(tr.EndFrame+1)*step
	return tsUS >= fromUS && tsUS < toUS
}
