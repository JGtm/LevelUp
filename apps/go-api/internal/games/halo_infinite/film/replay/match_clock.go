package replay

// match_clock.go — L'HORLOGE MATCH <-> FRAMES, ÉCRITE UNE SEULE FOIS.
//
// # Le défaut que ce fichier ferme
//
// La conversion « instant du MATCH (horloge du fil des morts) -> index de frame » était
// écrite TROIS fois, à l'identique : `flagCarryCtx` (drapeau), `skullCarryCtx` (crâne),
// `vipCrownCtx` (couronne). La règle du dépôt (CLAUDE.md n° 6) tolère deux copies ; à la
// troisième, on centralise ET on pose un garde-rail — c'est ce que l'arrivée du QUATRIÈME
// consommateur (le portage de la bombe d'Assaut, schéma 30) déclenche ici. Le garde-rail :
// `match_clock_guard_test.go`, qui interdit toute autre déclaration de `frameOfMatchMS`.
//
// # La convention, mesurée et déjà en production
//
//	horlogeFilm(ms) = horlogeMatch(ms) + deathOffsetMS      (cf. OwnerReport.DeathOffsetMS)
//	frame           = (horlogeFilm(µs) − origin) / step     (origin/step : axe des positions)
//
// `origin` est l'horodatage du PREMIER PAQUET DE POSITION (µs, horloge moteur), `step` la
// durée d'une frame en µs, `deathOffsetMS` le calage mesuré par `bestDeathOffset` (owners.go).

// matchClock pose les instants de l'horloge du MATCH (celle du fil des morts et des
// enregistrements du statborg) sur l'axe de frames publié du document.
type matchClock struct {
	// origin / step : l'axe de temps du rejeu, en microsecondes de l'horloge du FILM.
	origin, step uint64
	frames       int
	// deathOffsetMS : horlogeFilm = horlogeMatch + deathOffsetMS (cf. OwnerReport.DeathOffsetMS).
	deathOffsetMS int64
}

// frameOfMatchMS pose un instant de l'horloge du MATCH sur l'axe de frames du rejeu.
// -1 : l'instant précède la frame 0 ou l'axe n'a pas d'échelle.
func (c matchClock) frameOfMatchMS(matchMS int64) int {
	if c.step == 0 {
		return -1
	}
	filmUS := (matchMS + c.deathOffsetMS) * 1000
	if filmUS < int64(c.origin) {
		return -1
	}
	return int((uint64(filmUS) - c.origin) / c.step)
}

// matchMSOfFrame est l'inverse de [matchClock.frameOfMatchMS] : l'instant du MATCH où
// commence une frame. Il borne les périodes que RIEN ne ferme — sans lui, une période
// ouverte dans les dernières secondes s'arrêterait au dernier événement daté plutôt qu'à
// la fin du rejeu.
func (c matchClock) matchMSOfFrame(frame int) int64 {
	if frame < 0 {
		return 0
	}
	return int64(c.origin+uint64(frame)*c.step)/1000 - c.deathOffsetMS
}

// slackFrames traduit une durée en millisecondes en nombre de frames, arrondi vers le
// haut. Zéro quand l'axe n'a pas d'échelle — le seuil dégénère alors en « rien n'est
// proche du bord », le comportement le plus prudent.
func (c matchClock) slackFrames(ms int) int {
	if c.step == 0 {
		return 0
	}
	return int((uint64(ms)*1000 + c.step - 1) / c.step)
}
