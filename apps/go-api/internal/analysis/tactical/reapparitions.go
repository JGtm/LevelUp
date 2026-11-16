package tactical

// reapparitions.go — QUELLE VIE EST UNE REAPPARITION, ET QUI ETAIT OU QUAND ELLE COMMENCE.
//
// # LA GRAMMAIRE DU FILM PRIME SUR L'HEURISTIQUE
//
// Une piste n'est pas toujours une vie, et une vie n'est pas toujours precedee d'une mort : un
// trou de replication coupe une vie en deux pistes, et le debut d'une manche fait naitre tout
// le monde sans que personne n'ait ete tue. La decision « cette piste est une reapparition »
// se lit donc, quand le document la porte, dans la FIN que le film ecrit pour la piste
// precedente du joueur :
//
//  1. mort : le corps est libere (RaisonLiberee) a au plus quelques frames de la fin de la
//     piste, OU un compte a rebours de reapparition du joueur COMMENCE a ces frames-la. La piste
//     suivante est une reapparition apres mort ;
//  2. fin de manche : le corps est supprime alors que le joueur le commande encore
//     (RaisonSupprimee). La piste suivante est un spawn de DEBUT DE MANCHE : ni une
//     reapparition, ni une entree du modele, au meme titre qu'une premiere vie ;
//  3. aucune de ces lectures pour cette piste : pas une reapparition. Un document qui ne porte
//     AUCUNE lecture de fin de vie du film (LectureDuFilm : artefact anterieur a ces champs) n'a
//     donc aucune reapparition ; l'ecart entre deux pistes ne decide jamais (c'est un trou de
//     replication autant qu'une mort).
//
// # L'INSTANTANE
//
// A la frame de la reapparition, on garde la position de chaque AUTRE joueur vivant. Les
// joueurs dont la piste commence dans la meme seconde sont exclus : leur premier point est un
// spawn, pas une position de jeu (ils naissent avec le joueur, par exemple au debut d'une
// manche). Une piste sans xuid est vivante et compte (bot, vie non nommee).
//
// PUR : aucune I/O.

import (
	"math"
	"sort"
)

// Tolerances de la preuve de fin de vie, en millisecondes, converties avec le pas du document :
// la liberation du corps ou le debut du compte a rebours peuvent tomber de ToleranceFinAvantMs
// avant a ToleranceFinApresMs apres la derniere frame de la piste. Memes valeurs que la preuve de
// fin de vie du rejeu web : 2 images avant et 6 images apres, a 33 ms par image.
const (
	ToleranceFinAvantMs = 66
	ToleranceFinApresMs = 198
)

// FenetreApparitionSimultaneeMs : de part et d'autre de la frame de la reapparition, une
// piste qui COMMENCE dans cette fenetre est un spawn simultane et n'entre pas dans
// l'instantane (1 s au total).
const FenetreApparitionSimultaneeMs = 500

// EquipeInconnue est la valeur d'Equipe quand le film ne nomme aucune equipe.
const EquipeInconnue = -1

// vieIndexeeReapp est une piste prete a etre interrogee.
type vieIndexeeReapp struct {
	index  int // rang dans EntreeOccupation.Pistes
	xuid   string
	equipe int
	points []PointPiste
	debut  int
	fin    int
	finLue *FinLue
}

// IndexVies repond, pour un match, a « qui etait vivant ou a cette frame » et a « cette
// piste est-elle une reapparition ». Un index se construit une fois par match.
type IndexVies struct {
	intervalleMs int
	vies         []vieIndexeeReapp
	debuts       map[string][]int
	// filmLu : le document porte au moins une lecture de fin de vie du film (LectureDuFilm).
	filmLu bool
}

// LectureDuFilm dit si l'entree porte au moins une lecture de fin de vie du film : un debut de
// compte a rebours rattache a un xuid, ou la fin lue d'une piste qui a des points. Sans elle, aucune
// piste n'est une reapparition (cf. l'en-tete).
func LectureDuFilm(e EntreeOccupation) bool {
	for _, d := range e.DebutsDeCompte {
		if d.XUID != "" {
			return true
		}
	}
	for _, p := range e.Pistes {
		if p.Fin != nil && len(pointsDansLaFenetre(p)) > 0 {
			return true
		}
	}
	return false
}

// NouvelIndexVies indexe les pistes d'un match. Une piste sans point est ecartee.
func NouvelIndexVies(e EntreeOccupation) *IndexVies {
	ix := &IndexVies{intervalleMs: e.IntervalleFrameMs, debuts: map[string][]int{}}
	for _, d := range e.DebutsDeCompte {
		if d.XUID == "" {
			continue
		}
		ix.debuts[d.XUID] = append(ix.debuts[d.XUID], d.Frame)
	}
	ix.filmLu = LectureDuFilm(e)
	for i, p := range e.Pistes {
		points := pointsDansLaFenetre(p)
		if len(points) == 0 {
			continue
		}
		fin := points[len(points)-1].T
		if p.EndFrame > p.StartFrame && p.EndFrame > fin {
			// La fin declaree fait foi : le film a pu cesser de repliquer la position avant.
			fin = p.EndFrame
		}
		ix.vies = append(ix.vies, vieIndexeeReapp{
			index: i, xuid: p.XUID, equipe: p.Equipe, points: points,
			debut: points[0].T, fin: fin, finLue: p.Fin,
		})
	}
	return ix
}

// Vivants rend les joueurs vivants a `frame`, hors ceux du xuid donne (le sujet, toutes ses
// vies) et hors les pistes qui commencent dans la meme seconde. L'ordre est stable.
func (ix *IndexVies) Vivants(xuid string, frame int) []Vivant {
	var out []Vivant
	for _, v := range ix.vies {
		if v.xuid == xuid || v.debut > frame || v.fin < frame {
			continue
		}
		if math.Abs(float64(v.debut-frame))*float64(ix.intervalleMs) < FenetreApparitionSimultaneeMs {
			continue
		}
		p, _ := dernierPointAvantOuPremier(v.points, frame)
		out = append(out, Vivant{XUID: v.xuid, Equipe: v.equipe, X: p.X, Y: p.Y, Z: p.Z})
	}
	sort.SliceStable(out, func(a, b int) bool { return vivantAvant(out[a], out[b]) })
	return out
}

// vivantAvant est l'ordre total qui rend un instantane deterministe.
func vivantAvant(x, y Vivant) bool {
	switch {
	case x.XUID != y.XUID:
		return x.XUID < y.XUID
	case x.Equipe != y.Equipe:
		return x.Equipe < y.Equipe
	case x.X != y.X:
		return x.X < y.X
	case x.Y != y.Y:
		return x.Y < y.Y
	}
	return x.Z < y.Z
}

// EquipeDuJoueur rend l'equipe de la premiere vie nommee du joueur qui en porte une, sinon
// EquipeInconnue.
func (ix *IndexVies) EquipeDuJoueur(xuid string) int {
	best, bestDebut := EquipeInconnue, math.MaxInt
	for _, v := range ix.vies {
		if v.xuid == xuid && v.equipe >= 0 && v.debut < bestDebut {
			best, bestDebut = v.equipe, v.debut
		}
	}
	return best
}

// Spawns rend, par rang de piste dans l'entree, le SpawnPiste de chaque piste NOMMEE (sans
// la marque PremiereVie, posee a l'assemblage). Une reapparition porte sa mort et, quand son
// equipe est connue, l'instantane des autres joueurs.
func (ix *IndexVies) Spawns() map[int]SpawnPiste {
	parJoueur := map[string][]int{}
	for k, v := range ix.vies {
		if v.xuid != "" {
			parJoueur[v.xuid] = append(parJoueur[v.xuid], k)
		}
	}
	out := make(map[int]SpawnPiste, len(ix.vies))
	for xuid, ks := range parJoueur {
		sort.SliceStable(ks, func(a, b int) bool { return ix.vies[ks[a]].debut < ix.vies[ks[b]].debut })
		for r, k := range ks {
			v := ix.vies[k]
			s := SpawnPiste{Frame: v.debut, X: v.points[0].X, Y: v.points[0].Y, Z: v.points[0].Z}
			if r > 0 {
				ix.qualifier(&s, xuid, v, ix.vies[ks[r-1]])
			}
			out[v.index] = s
		}
	}
	return out
}

// qualifier decide si la vie `v` est une reapparition apres `prev` et la remplit.
func (ix *IndexVies) qualifier(s *SpawnPiste, xuid string, v, prev vieIndexeeReapp) {
	if !ix.filmLu {
		return
	}
	mort, manche := ix.lireFin(xuid, prev)
	s.DebutDeManche = manche && !mort
	if !mort {
		return
	}
	s.Reapparition = true
	equipe := v.equipe
	if equipe < 0 {
		equipe = ix.EquipeDuJoueur(xuid)
	}
	dernier := prev.points[len(prev.points)-1]
	s.Mort = &PointMort{Frame: prev.fin, X: dernier.X, Y: dernier.Y, Z: dernier.Z}
	if equipe >= 0 {
		s.Instantane = ix.Vivants(xuid, v.debut)
	}
}

// lireFin dit comment le film clot la vie `prev` : par une mort (liberation du corps ou
// compte a rebours qui commence), ou par une fin de manche (corps supprime). La mort l'emporte :
// un compte a rebours prouve que le joueur attendait.
func (ix *IndexVies) lireFin(xuid string, prev vieIndexeeReapp) (mort, manche bool) {
	dansLaTolerance := func(frame int) bool {
		d := (frame - prev.fin) * ix.intervalleMs
		return d >= -ToleranceFinAvantMs && d <= ToleranceFinApresMs
	}
	for _, f := range ix.debuts[xuid] {
		if dansLaTolerance(f) {
			return true, false
		}
	}
	if f := prev.finLue; f != nil && dansLaTolerance(f.Frame) {
		switch f.Raison {
		case RaisonLiberee:
			return true, false
		case RaisonSupprimee:
			return false, true
		}
	}
	return false, false
}
