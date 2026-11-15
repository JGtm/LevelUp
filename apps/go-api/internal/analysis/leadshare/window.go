package leadshare

// Clock réunit ce que le serveur sait du temps d'un match : l'artefact de rejeu pour l'axe du
// film, le registre pour la durée du match.
type Clock struct {
	// OriginMS : instant de l'image zéro du film sur l'axe du match. Nil = non établie.
	OriginMS *int64
	// T0FilmMS : coup d'envoi MESURÉ dans le film, sur l'axe du match. Nil = non mesuré.
	T0FilmMS *int64
	// DurationSeconds : durée du match au registre, countdown compris, sur l'axe du match.
	DurationSeconds *int64
	// FrameIntervalMS : durée d'une image du document.
	FrameIntervalMS int
	// FrameCount : nombre d'images du document.
	FrameCount int
}

// PlayedWindow rend la partie JOUÉE du match sur l'axe du film, ou la raison pour laquelle elle
// n'est pas établie. Même fenêtre que celle du rejeu (apps/web/src/features/match-replay/model/
// replayWindow.ts) : le film déborde le match de son countdown avant et d'une queue de quelques
// secondes après.
//
//	début = max(0, T0 film - origine)         le coup d'envoi mesuré dans le film
//	fin   = min(durée du match - origine, dernière image)
//
// La fin ne vient jamais du score : le dernier point tombe parfois deux minutes avant la fin
// d'un match terminé au temps. Zéro n'est pas une origine neutre : sans origine, pas de fenêtre.
func PlayedWindow(c Clock) (Window, Reason) {
	switch {
	case c.FrameIntervalMS <= 0:
		return Window{}, ReasonClockStepMissing
	case c.OriginMS == nil:
		return Window{}, ReasonClockOriginMissing
	case c.T0FilmMS == nil:
		return Window{}, ReasonKickoffUnmeasured
	case c.DurationSeconds == nil:
		return Window{}, ReasonDurationUnknown
	}
	start := max(0, *c.T0FilmMS-*c.OriginMS)
	end := min(*c.DurationSeconds*1000-*c.OriginMS, int64(c.FrameCount-1)*int64(c.FrameIntervalMS))
	if c.FrameCount < 2 || end <= start {
		return Window{}, ReasonWindowIncoherent
	}
	return Window{StartMS: start, EndMS: end}, ReasonNone
}
