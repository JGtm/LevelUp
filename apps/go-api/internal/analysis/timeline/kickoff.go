package timeline

import (
	"time"

	"levelup/go-api/internal/domain"
)

// KickoffRev est la révision de la résolution du coup d'envoi et du temps de jeu qui en
// découle (ordre des sources, estimation et ses seuils, formule du temps de jeu). Une base
// partagée range la dernière révision dont le rattrapage complet a réussi ; une valeur
// différente relance ce rattrapage au démarrage (ADR 0050). Le garde-rail
// TestKickoffParametersMatchRevision échoue quand un seuil change sans que la révision change ;
// un changement de logique sans changement de nombre se signale en changeant cette valeur. Les notes
// lisent ce que cette résolution écrit : l'empreinte des révisions de notation hache KickoffRev
// (TestScoringParametersMatchRevision, paquet sync), donc la changer impose de les monter aussi.
const KickoffRev = "kickoff-2026-11-24"

// KickoffSource dit d'où vient le coup d'envoi rangé d'un match. Valeurs rangées dans
// `match_registry.real_start_source` : `film`, `api`, ou vide (coup d'envoi inconnu).
type KickoffSource string

const (
	// KickoffSourceFilm : le coup d'envoi publié par le document de rejeu du match.
	KickoffSourceFilm KickoffSource = "film"
	// KickoffSourceAPI : l'estimation par les arrivées des joueurs présents au début.
	KickoffSourceAPI KickoffSource = "api"
	// KickoffSourceUnknown : ni rejeu ni estimation acceptée.
	KickoffSourceUnknown KickoffSource = ""
)

// Kickoff est le coup d'envoi résolu d'un match.
type Kickoff struct {
	// Source : la source retenue ; inconnue = aucune valeur.
	Source KickoffSource
	// OffsetMs : le coup d'envoi en millisecondes depuis le début canonique du match. Sans sens
	// quand Source est inconnue.
	OffsetMs int64
	// Estimate : l'issue de l'estimation quand elle a été jouée (diagnostic journalisé et compté,
	// jamais rangé) ; vide quand le rejeu a décidé.
	Estimate T0Quality
	// FilmOutOfBounds : le rejeu publiait un coup d'envoi hors de la fenêtre du match (négatif, ou
	// au-delà de sa durée) ; il a été refusé comme un rejeu qui n'en publie pas.
	FilmOutOfBounds bool
}

// Known dit si le match a un coup d'envoi.
func (k Kickoff) Known() bool { return k.Source != KickoffSourceUnknown }

// RealStart rend l'instant du coup d'envoi en UTC (début canonique + décalage), nil quand il
// est inconnu.
func (k Kickoff) RealStart(startUTC time.Time) *time.Time {
	if !k.Known() {
		return nil
	}
	t := startUTC.UTC().Add(time.Duration(k.OffsetMs) * time.Millisecond)
	return &t
}

// KickoffInput porte ce que la résolution lit pour un match.
type KickoffInput struct {
	// StartUTC : début canonique du match (`start_time_utc`, sinon `start_time` lu en UTC).
	StartUTC time.Time
	// FilmT0Ms : le coup d'envoi tel que le document de rejeu du match le publie (`t0FilmMs`),
	// en ms depuis le début canonique. Nil : pas de rejeu, ou rejeu sans coup d'envoi.
	FilmT0Ms *int64
	// DurationSeconds : la durée du match ; nil ou nulle = inconnue (seule la borne zéro du coup
	// d'envoi du rejeu s'applique alors).
	DurationSeconds *int
	// LongestPlayedSeconds : le plus long temps de jeu rangé des joueurs du match ; nil = inconnu.
	// Quand la durée du registre est plus courte (durée manifestement fausse, cf.
	// [DurationUnreliable]), il borne la fenêtre du match à sa place.
	LongestPlayedSeconds *int
	// Participations : les arrivées des joueurs, lues seulement quand le rejeu ne décide pas.
	Participations []ParticipationT0Input
}

// ResolveKickoff est le SEUL point où se décide le coup d'envoi d'un match, dans l'ordre fixe
// des sources : le rejeu, sinon l'estimation par les arrivées (UN appel de ComputeT0, refusée
// quand elle est négative ou au-delà de son seuil), sinon inconnu. Le rejeu est pris tel que le
// document le publie : aucun détecteur n'est rejoué ici ; seule sa plausibilité se vérifie, par la
// fenêtre du match elle-même (ni négatif, ni au-delà de la durée), et un coup d'envoi hors de
// cette fenêtre compte comme un rejeu qui n'en publie pas. La durée du registre est parfois fausse
// (21 s pour un match de 7 minutes) : quand elle est plus courte que le plus long temps de jeu d'un
// joueur, c'est ce temps de jeu qui borne la fenêtre ([MatchWindowSeconds]), et un coup d'envoi juste
// n'est pas refusé pour cette raison.
func ResolveKickoff(in KickoffInput) Kickoff {
	outOfBounds := false
	if in.FilmT0Ms != nil {
		window := MatchWindowSeconds(in.DurationSeconds, in.LongestPlayedSeconds)
		if FilmWithinMatch(*in.FilmT0Ms, window) {
			return Kickoff{Source: KickoffSourceFilm, OffsetMs: *in.FilmT0Ms}
		}
		outOfBounds = true
	}
	t0, q := ComputeT0(in.Participations, in.StartUTC)
	if !q.Computed() {
		return Kickoff{Source: KickoffSourceUnknown, Estimate: q, FilmOutOfBounds: outOfBounds}
	}
	return Kickoff{Source: KickoffSourceAPI, OffsetMs: t0, Estimate: q, FilmOutOfBounds: outOfBounds}
}

// DurationUnreliable dit si la durée du registre est manifestement fausse : connue, et plus courte
// que le plus long temps de jeu d'un joueur du match (un joueur ne joue pas plus longtemps que le
// match). Faux quand l'une des deux valeurs manque : rien ne contredit alors la durée.
func DurationUnreliable(durationSeconds, longestPlayedSeconds *int) bool {
	return durationSeconds != nil && *durationSeconds > 0 &&
		longestPlayedSeconds != nil && *longestPlayedSeconds > *durationSeconds
}

// MatchWindowSeconds rend la durée qui borne la fenêtre du match : celle du registre, sauf quand elle
// est manifestement fausse ([DurationUnreliable]) — c'est alors le plus long temps de jeu. C'est la
// seule règle qui accepte ou refuse un coup d'envoi du rejeu (ResolveKickoff, et le coup d'envoi
// du rejeu gardé par sync/kickoff.Resolve).
func MatchWindowSeconds(durationSeconds, longestPlayedSeconds *int) *int {
	if DurationUnreliable(durationSeconds, longestPlayedSeconds) {
		return longestPlayedSeconds
	}
	return durationSeconds
}

// FilmWithinMatch dit si un coup d'envoi du rejeu tombe dans la fenêtre du match : [0, durée], ou
// [0, ∞[ quand la durée est inconnue. C'est la règle de la résolution ([ResolveKickoff]) ; tout
// lecteur qui prend le coup d'envoi directement dans le document (fenêtre jouée de la Mainmise)
// l'applique aussi, pour ne jamais retenir une valeur que le registre a refusée.
func FilmWithinMatch(t0Ms int64, durationSeconds *int) bool {
	if t0Ms < 0 {
		return false
	}
	return durationSeconds == nil || *durationSeconds <= 0 || t0Ms <= int64(*durationSeconds)*1000
}

// GameplayTimeline rend la chronologie de jeu d'un match résolu : début canonique, durée du
// match et coup d'envoi, par la règle de lecture [ForMatch] (coup d'envoi inconnu : la
// chronologie part du début du match). ok=false quand la durée est absente ou nulle : le match
// n'a pas de fenêtre de jeu.
func GameplayTimeline(startUTC time.Time, durationSeconds *int, k Kickoff) (domain.MatchTimeline, bool) {
	if durationSeconds == nil || *durationSeconds <= 0 {
		return domain.MatchTimeline{}, false
	}
	var off *int64
	if k.Known() {
		off = &k.OffsetMs
	}
	return ForMatch(startUTC, int64(*durationSeconds)*1000, off), true
}

// PlayedSeconds rend le temps de jeu à ranger pour un joueur sur la chronologie du match, par la
// formule ComputeTimePlayedFor (troncature à la seconde comprise). ok=false quand la valeur rangée
// reste : présence inconnue (aucune heure d'arrivée), match sans fenêtre de jeu, ou joueur parti
// avant le coup d'envoi (fenêtre vide) — la dernière valeur rangée est alors conservée telle quelle
// (celle de l'API pour une présence inconnue, qui n'a jamais été recalculée).
func PlayedSeconds(in TimePlayedInput, tl domain.MatchTimeline, hasWindow bool) (int, bool) {
	if !hasWindow {
		return 0, false
	}
	secs, q := ComputeTimePlayedFor(in, tl)
	if q != TimePlayedOK {
		return 0, false
	}
	return int(secs), true
}

// HumanXUID dit si un identifiant est celui d'un joueur (entier) : les bots portent un
// identifiant non numérique (`bid(N.M)`) et n'entrent pas dans l'estimation.
func HumanXUID(xuid string) bool {
	if xuid == "" {
		return false
	}
	for _, c := range xuid {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
