package tactical

// sequence.go — LA SEQUENCE D'UNE MORT (vue D du detail d'une zone) : ou etaient la victime, son tueur
// et ses coequipiers autour de l'instant de sa mort, lus sur les TRAJECTOIRES du rejeu 2D.
//
// Pur : les pistes decodees de l'artefact, l'identite des acteurs (venue de la BASE, jamais de
// l'equipe du film), le rayon et l'horloge sont donnes. Aucun fichier, aucune base.
//
// LES TEMPS. L'axe de travail est celui du FILM (ms depuis l'image zero) ; la mort y est
// `matchMs − originMs` (la soustraction unique de `matchClock.filmMsOfMatchMs`, cote web). La reponse
// parle en DECALAGE a la mort : −8 s a +3 s pour les traces, −4 s a +3 s pour les etapes.
//
// LES TRACES sont reechantillonnees toutes les 200 ms. PAS D'INTERPOLATION A TRAVERS UNE LACUNE : un
// point qui porte `G > 0` (le film n'a rien replique pendant ce temps) n'est jamais relie au
// precedent — l'echantillon qui le suit est marque `Coupure`, et ceux qui tombent dans la lacune
// n'existent pas. Un acteur sans position a un instant n'a pas d'echantillon : jamais (0, 0).
//
// LE CERCLE DE PORTEE suit la victime avant sa mort (offset < 0) et RESTE AU POINT DE MORT ensuite.
// Le compte « dans le rayon » est celui des coequipiers positionnes a cet instant dont la distance
// au centre est a la portee (`coordination.APortee`, borne inclusive).

import (
	"errors"
	"sort"

	"levelup/go-api/internal/analysis/coordination"
	"levelup/go-api/internal/domain"
)

// Les durees de la sequence, en millisecondes.
const (
	SequenceAvantMs       = 8000 // debut des traces, avant la mort
	SequenceApresMs       = 3000 // fin des traces et des etapes, apres la mort
	SequencePasMs         = 200  // pas des traces
	SequenceEtapeDebut    = -4000
	SequenceEtapePasMs    = 1000
	SequenceTraineMs      = 4000 // duree de la traine derriere chaque acteur
	toleranceDeBordImages = 1    // image(s) de tolerance avant le premier point et apres le dernier
)

// ErrSequenceSansMort : la victime n'a aucune position a l'instant de sa mort — la sequence n'a pas
// de point d'ancrage et n'est pas rendue.
var ErrSequenceSansMort = errors.New("sequence: la victime n'a pas de position a l'instant de sa mort")

// ActeurSequence est un acteur dont l'identite vient de la base, et sa piste dans le film (vide si le
// film ne la porte pas).
type ActeurSequence struct {
	Role     string
	XUID     string
	Gamertag string
	Points   []domain.TacticalPistePoint
}

// EntreeSequence porte tout ce que la sequence lit.
type EntreeSequence struct {
	Victime     ActeurSequence
	Tueur       *ActeurSequence
	Coequipiers []ActeurSequence
	// FrameIntervalMs : la duree d'une image de l'artefact.
	FrameIntervalMs int
	// MortFilmMs : l'instant de la mort sur l'axe du film.
	MortFilmMs int64
	// MortMatchMs et T0Ms : l'instant de la mort sur l'axe du match et le coup d'envoi du match
	// (nil = inconnu), pour l'horloge affichee des etapes.
	MortMatchMs int64
	T0Ms        *int64
	RayonM      float64
}

// pos est une position du plan, en metres monde.
type pos struct{ x, y float64 }

// positionA rend la position interpolee de la piste a `filmMs`, ou faux : avant le premier point ou
// apres le dernier (a une image de tolerance), ou dans une lacune (`G > 0` sur le point suivant).
func positionA(points []domain.TacticalPistePoint, intervalMs int, filmMs int64) (pos, bool) {
	n := len(points)
	if n == 0 || intervalMs <= 0 {
		return pos{}, false
	}
	ms := func(i int) int64 { return int64(points[i].T) * int64(intervalMs) }
	tolerance := int64(toleranceDeBordImages * intervalMs)
	i := sort.Search(n, func(k int) bool { return ms(k) >= filmMs })
	switch {
	case i == n: // apres le dernier point
		if filmMs-ms(n-1) <= tolerance {
			return pos{float64(points[n-1].X), float64(points[n-1].Y)}, true
		}
		return pos{}, false
	case ms(i) == filmMs:
		return pos{float64(points[i].X), float64(points[i].Y)}, true
	case i == 0: // avant le premier point
		if ms(0)-filmMs <= tolerance {
			return pos{float64(points[0].X), float64(points[0].Y)}, true
		}
		return pos{}, false
	}
	p0, p1 := points[i-1], points[i]
	if p1.G > 0 {
		return pos{}, false
	}
	f := float64(filmMs-ms(i-1)) / float64(ms(i)-ms(i-1))
	return pos{
		x: float64(p0.X) + f*float64(p1.X-p0.X),
		y: float64(p0.Y) + f*float64(p1.Y-p0.Y),
	}, true
}

// lacuneEntre dit qu'un point a lacune tombe dans ]t0, t1] : la trace ne se relie pas.
func lacuneEntre(points []domain.TacticalPistePoint, intervalMs int, t0, t1 int64) bool {
	for _, p := range points {
		t := int64(p.T) * int64(intervalMs)
		if t > t0 && t <= t1 && p.G > 0 {
			return true
		}
	}
	return false
}

// traceDe echantillonne un acteur toutes les SequencePasMs de −8 s a +3 s autour de la mort.
func traceDe(a ActeurSequence, in EntreeSequence) domain.TacticalSequenceActeur {
	out := domain.TacticalSequenceActeur{Role: a.Role, XUID: a.XUID, Gamertag: a.Gamertag,
		Points: []domain.TacticalSequencePoint{}}
	precedentPresent := false
	for off := -SequenceAvantMs; off <= SequenceApresMs; off += SequencePasMs {
		t := in.MortFilmMs + int64(off)
		p, ok := positionA(a.Points, in.FrameIntervalMs, t)
		if !ok {
			precedentPresent = false
			continue
		}
		coupure := (!precedentPresent && len(out.Points) > 0) ||
			(precedentPresent && lacuneEntre(a.Points, in.FrameIntervalMs, t-SequencePasMs, t))
		out.Points = append(out.Points, domain.TacticalSequencePoint{
			OffsetMs: off, X: Arrondi2(p.x), Y: Arrondi2(p.y), Coupure: coupure,
		})
		precedentPresent = true
	}
	return out
}

// Sequence assemble la sequence d'une mort, ou ErrSequenceSansMort.
func Sequence(in EntreeSequence) (domain.TacticalSequence, error) {
	mort, ok := positionA(in.Victime.Points, in.FrameIntervalMs, in.MortFilmMs)
	if !ok {
		return domain.TacticalSequence{}, ErrSequenceSansMort
	}
	pointMort := domain.TacticalPoint{X: Arrondi2(mort.x), Y: Arrondi2(mort.y)}
	seq := domain.TacticalSequence{
		RayonM: in.RayonM, TraineMs: SequenceTraineMs, PasMs: SequencePasMs, Mort: pointMort,
		Acteurs: []domain.TacticalSequenceActeur{},
	}
	acteurs := append([]ActeurSequence{in.Victime}, in.Coequipiers...)
	if in.Tueur != nil {
		acteurs = append(acteurs, *in.Tueur)
	}
	for _, a := range acteurs {
		if t := traceDe(a, in); len(t.Points) > 0 {
			seq.Acteurs = append(seq.Acteurs, t)
		}
	}
	for off := SequenceEtapeDebut; off <= SequenceApresMs; off += SequenceEtapePasMs {
		seq.Etapes = append(seq.Etapes, etapeA(off, in, pointMort))
	}
	return seq, nil
}

// etapeA rend l'etape a `off` ms de la mort : le centre du cercle et ce qu'il contient.
func etapeA(off int, in EntreeSequence, pointMort domain.TacticalPoint) domain.TacticalSequenceEtape {
	t := in.MortFilmMs + int64(off)
	centre := pointMort
	if off < 0 {
		if p, ok := positionA(in.Victime.Points, in.FrameIntervalMs, t); ok {
			centre = domain.TacticalPoint{X: Arrondi2(p.x), Y: Arrondi2(p.y)}
		}
	}
	e := domain.TacticalSequenceEtape{OffsetMs: off, Centre: centre, DansLeRayonXUIDs: []string{}}
	if in.T0Ms != nil {
		v := max(in.MortMatchMs+int64(off)-*in.T0Ms, 0)
		e.InstantAfficheMs = &v
	}
	for _, c := range in.Coequipiers {
		p, ok := positionA(c.Points, in.FrameIntervalMs, t)
		if !ok {
			continue
		}
		e.Positionnes++
		d := DistanceM(centre, domain.TacticalPoint{X: p.x, Y: p.y})
		if coordination.APortee(&d, in.RayonM) {
			e.DansLeRayon++
			e.DansLeRayonXUIDs = append(e.DansLeRayonXUIDs, c.XUID)
		}
	}
	return e
}
