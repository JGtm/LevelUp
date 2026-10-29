package halo_infinite

// film_highlight_events.go — LE PONT ENTRE L EVENEMENT DE TEMPS FORT DU DECODEUR ET CELUI DE
// LEVELUP (plan D2, decision D2-B ; ADR 0012).
//
// Le decodeur de film porte SA forme (`film/highlightevent`) : il ne doit importer aucun paquet de
// LevelUp. LevelUp garde la sienne (`domain/highlightevent`), remplie aussi par l import
// OpenSpartan et le rattrapage des medailles, et lue par `internal/analysis`. Les deux formes ont
// les memes champs ; la conversion vit ICI, dans l adaptateur de titre, et nulle part ailleurs :
// tout lecteur du chunk des temps forts hors du film appelle [ParseHighlightEvents].

import (
	"levelup/go-api/internal/domain/highlightevent"

	"github.com/JGtm/LevelUp/libs/film/decfilm"
	filmhighlight "github.com/JGtm/LevelUp/libs/film/highlightevent"
)

// ParseHighlightEvents lit le chunk des temps forts d un film par le decodeur et rend les
// evenements sous la forme de LevelUp. Meme contrat que `decfilm.ParseHighlightEvents` : memes
// evenements dans le meme ordre, meme erreur, `nil` quand le decodeur rend `nil`.
func ParseHighlightEvents(data []byte, filmMajorVersion int) ([]highlightevent.HighlightEvent, error) {
	evs, err := decfilm.ParseHighlightEvents(data, filmMajorVersion)
	return highlightEventsDepuisFilm(evs), err
}

// HighlightEventDepuisFilm convertit un evenement du decodeur en evenement de LevelUp, champ pour
// champ. Fonction pure ; `TestHighlightEventDepuisFilm_MemesChamps` exige que les deux formes
// gardent les memes champs (noms, types, ordre).
func HighlightEventDepuisFilm(e filmhighlight.HighlightEvent) highlightevent.HighlightEvent {
	return highlightevent.HighlightEvent{
		XUID:      e.XUID,
		Gamertag:  e.Gamertag,
		EventType: e.EventType,
		TypeHint:  e.TypeHint,
		IsMedal:   e.IsMedal,
		TimeMS:    e.TimeMS,
		MedalType: e.MedalType,
	}
}

// highlightEventsDepuisFilm convertit une tranche ; `nil` reste `nil`.
func highlightEventsDepuisFilm(evs []filmhighlight.HighlightEvent) []highlightevent.HighlightEvent {
	if evs == nil {
		return nil
	}
	out := make([]highlightevent.HighlightEvent, len(evs))
	for i, e := range evs {
		out[i] = HighlightEventDepuisFilm(e)
	}
	return out
}
