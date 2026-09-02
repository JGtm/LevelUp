package replay

import (
	"context"
	"log/slog"
	"sort"
)

// t0_film.go — LE COUP D'ENVOI, DATE PAR LE PREMIER MOUVEMENT DU FILM.
//
// # 1. LE DEFAUT QUE CE FICHIER FERME
//
// Le T0 d'un match (fin du decompte pre-match, coup d'envoi du gameplay) est ESTIME cote
// sync depuis les `first_joined_time` de l'API (`analysis/timeline/compute_t0.go`). Sur une
// part des matchs, l'API rend des `first_joined_time` colles au `start_time` : le T0 calcule
// vaut alors ~0 alors que le decompte dure une vingtaine de secondes. Le rejeu 2D demarre sur
// des joueurs statufies, et tout ce qui se date sur l'axe du match (premier frag...) est
// decale d'autant.
//
// Le film, lui, porte le coup d'envoi directement : la grille se leve d'un coup et tout le
// monde part quasi simultanement. Le PREMIER MOUVEMENT lu dans les positions DATE donc le
// coup d'envoi, sans base, sans horloge murale et sans appariement.
//
// # 2. CE QUE LA MESURE A ETABLI (2026-09-02, 106 artefacts, 101 retenus)
//
// La recherche fondatrice est `t0_mouvement_research_test.go` — elle appelle ce detecteur, et
// ses trois temoins sont INTERNES au film (aucun etalon en jeu) :
//
//	RAFALE   66,7 % de l'effectif (mediane 6 joueurs sur 8-9) part dans la SECONDE du premier.
//	ACCORD   l'ecart entre le 1er et le 3e partant vaut 100 ms de mediane, 500 ms au p95 —
//	         le choix de la variante ne change donc rien, et le 1er suffit.
//	MARGE    le premier mouvement tombe a 22 700 ms de la frame 0 du film, ECART-TYPE 299 ms,
//	         CV 0,013, etendue 21 500 - 23 000 ms sur 83 matchs.
//
// Confrontation des DEUX horloges sur les 49 matchs au T0-API sain : ecart-type 9 752 ms pour
// le film contre 12 764 ms pour l'API, et l'API descend a 17 804 ms la ou le film ne descend
// jamais sous 25 907 ms. Sur les 11 matchs au T0-API degenere, le detecteur rend 10 decomptes
// sur 11 dans la plage plausible 15-45 s.
//
// AUCUNE CONSTANTE ABSOLUE N'EST ECRITE ICI (decision D1 du plan). La marge de 22 700 ms est
// un temoin de plausibilite documentaire, jamais une valeur servie : le detecteur MESURE par
// match, il ne rejoue pas une constante.
//
// # 3. LES DEUX PIEGES QUE LE DETECTEUR EVITE
//
// Par piste (une piste = UNE VIE, cf. `Track.XUID`), sur la serie de points :
//
//  1. le premier point d'une piste n'ouvre aucun pas (rien a soustraire) ;
//  2. un pas dont les deux points sont separes de plus de `t0FilmWindowMS` est une RUPTURE,
//     pas un deplacement — le film ne replique la position que lorsqu'elle CHANGE, donc un
//     joueur immobile pendant le decompte n'a AUCUN point entre la frame 0 et son depart
//     (temoin sur 1b2d9e08 : le slot 512 a un point a t=0 puis plus rien jusqu'a t=227).
//     Compter ce pas comme un deplacement daterait le coup d'envoi a la frame 0 de tout film ;
//  3. un pas de plus de `t0FilmJumpM` est une TELEPORTATION (apparition, arrivee tardive), pas
//     une locomotion : rupture aussi.
//
// MOUVEMENT = le cumul des pas CONTIGUS depasse `t0FilmCumulM` dans une fenetre glissante de
// `t0FilmWindowMS`. Un jitter d'une seule image ne suffit donc pas.
//
// # 4. LE REFUS EST EXPLICITE (decision D6 du plan)
//
// Trois causes rendent nil — jamais un zero ambigu, jamais une valeur douteuse en silence :
// aucune piste ne bouge, la rafale de depart compte moins de `t0FilmMinBurst` partants, ou le
// premier mouvement arrive plus de `t0FilmMaxDelayMS` apres la frame 0. Chaque refus est
// journalise avec son match et sa raison, et publie dans `coverage.t0Film`.

// --- Seuils du detecteur, ecrits AVANT la mesure et confirmes par elle ---
const (
	// t0FilmJumpM : au-dela, le pas est une teleportation, pas une locomotion.
	t0FilmJumpM = 5.0
	// t0FilmCumulM : le deplacement cumule qui fait un mouvement. Le plancher de bruit mesure
	// sur la fenetre de decompte des matchs sains donne un cumul maximal median de 0,06 m —
	// ce seuil est huit fois au-dessus de ce bruit.
	t0FilmCumulM = 0.5
	// t0FilmWindowMS : largeur de la fenetre glissante de cumul, ET duree maximale d'un pas
	// exploitable (au-dela, le film n'a rien replique : c'est une rupture).
	t0FilmWindowMS = 1000
	// t0FilmBurstMS : la fenetre dans laquelle on compte les partants qui accompagnent le
	// premier. Mesuree : l'ecart 1er -> 3e partant vaut 100 ms de mediane, 500 ms au p95.
	t0FilmBurstMS = 1000
	// t0FilmMinBurst : en deca, un « premier mouvement » est un artefact isole (un point
	// aberrant, une piste de spectateur) et non la levee de la grille. Deux, parce que la
	// rafale mesuree porte 6 partants de mediane : exiger deux est le minimum qui distingue
	// une grille d'un accident.
	t0FilmMinBurst = 2
	// t0FilmMaxDelayMS : au-dela de deux minutes apres la frame 0, ce qui a bouge n'est plus
	// un coup d'envoi mais du jeu deja en cours (film qui demarre tard, positions manquantes).
	// La borne reprend `t0MaxPlausibleMs` de la production (`analysis/timeline/compute_t0.go`).
	t0FilmMaxDelayMS = 120000
)

// T0FilmPoint est un point de piste reduit a ce que le detecteur lit : son instant sur la
// grille de frames et sa position monde.
type T0FilmPoint struct {
	T       int
	X, Y, Z float32
}

// T0FilmTrack est une piste — UNE VIE, pas un joueur (cf. `Track.XUID`) — et l'identite de
// son porteur, qui sert a compter la rafale par JOUEUR et non par vie.
type T0FilmTrack struct {
	XUID   string
	Points []T0FilmPoint
}

// T0FilmCoverage est le VERDICT du detecteur, publie dans l'artefact a cote du champ.
//
// ELLE EST PUBLIEE MEME QUAND LE DETECTEUR REFUSE, et c'est le point : un film sans mouvement,
// un film dont un seul joueur part et un film qui demarre trop tard rendent tous trois un
// `t0FilmMs` absent — seule la raison les distingue.
type T0FilmCoverage struct {
	// Detected dit si un coup d'envoi a ete date. Faux = `t0FilmMs` est absent.
	Detected bool `json:"detected"`
	// Reason nomme la cause du refus (`noMovement` / `burstTooSmall` / `tooLate` /
	// `noFrameInterval`). Vide quand la detection a abouti.
	Reason string `json:"reason,omitempty"`
	// Tracks est le nombre de pistes exploitables (au moins deux points) — le denominateur.
	Tracks int `json:"tracks"`
	// Moving est le nombre de pistes qui portent un mouvement.
	Moving int `json:"moving"`
	// Burst est le nombre de partants distincts dans la seconde du premier. C'est le TEMOIN
	// DIRECT de l'hypothese : si la grille se leve d'un coup, l'essentiel de l'effectif part
	// ensemble ; un partant isole est un artefact.
	Burst int `json:"burst"`
	// MarginMs est l'ecart entre le premier mouvement et la FRAME 0 du film. Une marge
	// quasi nulle dit que le film commence SUR le coup d'envoi (18 films sur 101 mesures) —
	// la valeur reste bonne, le decompte n'est simplement pas filme.
	MarginMs int64 `json:"marginMs"`
}

// t0FilmReason* nomment les refus. Constantes plutot que litteraux : la raison sort dans
// l'artefact ET dans le journal, et deux ecritures divergentes rendraient le champ inutile.
const (
	t0FilmReasonNoMovement  = "noMovement"
	t0FilmReasonSmallBurst  = "burstTooSmall"
	t0FilmReasonTooLate     = "tooLate"
	t0FilmReasonNoFrameStep = "noFrameInterval"
)

// t0FilmStep est un deplacement entre deux points CONSECUTIFS d'une meme piste.
type t0FilmStep struct {
	startFrame, endFrame int
	dist                 float64
	// rupture : le pas ne peut pas se lire comme une locomotion (trou de replication ou
	// teleportation). Il vide l'accumulateur au lieu de l'alimenter.
	rupture bool
}

// t0FilmSteps ramene une piste a la suite de ses pas. Le premier point n'ouvre aucun pas.
func t0FilmSteps(pts []T0FilmPoint, frameIntervalMS int) []t0FilmStep {
	if len(pts) < 2 || frameIntervalMS <= 0 {
		return nil
	}
	tri := append([]T0FilmPoint(nil), pts...)
	sort.Slice(tri, func(i, j int) bool { return tri[i].T < tri[j].T })
	out := make([]t0FilmStep, 0, len(tri)-1)
	for i := 1; i < len(tri); i++ {
		a, b := tri[i-1], tri[i]
		d := dist3([3]float32{a.X, a.Y, a.Z}, [3]float32{b.X, b.Y, b.Z})
		out = append(out, t0FilmStep{
			startFrame: a.T,
			endFrame:   b.T,
			dist:       d,
			rupture:    (b.T-a.T)*frameIntervalMS > t0FilmWindowMS || d > t0FilmJumpM,
		})
	}
	return out
}

// t0FilmWindow est LA FENETRE GLISSANTE DE CUMUL, et elle n'existe qu'ici : la recherche
// fondatrice l'emprunte au lieu d'en garder une copie (regle du depot : au plus deux copies
// d'un meme motif, et une factorisation sans reemploi re-diverge).
type t0FilmWindow struct {
	// widthFrames est la largeur de la fenetre, exprimee dans l'unite des `Point.T`.
	widthFrames int
	starts      []int
	dists       []float64
	sum         float64
}

func newT0FilmWindow(frameIntervalMS int) t0FilmWindow {
	return t0FilmWindow{widthFrames: t0FilmWindowMS / frameIntervalMS}
}

// push absorbe un pas et rend le cumul CONTIGU de la fenetre apres absorption. Un pas de
// rupture la vide et rend zero.
func (w *t0FilmWindow) push(s t0FilmStep) float64 {
	if s.rupture {
		w.starts, w.dists, w.sum = w.starts[:0], w.dists[:0], 0
		return 0
	}
	w.starts = append(w.starts, s.startFrame)
	w.dists = append(w.dists, s.dist)
	w.sum += s.dist
	// La fenetre se mesure du DEBUT du pas le plus ancien a la FIN du pas courant.
	for len(w.dists) > 1 && (s.endFrame-w.starts[0]) > w.widthFrames {
		w.sum -= w.dists[0]
		w.starts, w.dists = w.starts[1:], w.dists[1:]
	}
	return w.sum
}

// t0FilmFirstMovement rend la frame du PREMIER MOUVEMENT d'une piste, ou -1 quand elle n'en
// porte aucun.
func t0FilmFirstMovement(steps []t0FilmStep, frameIntervalMS int) int {
	if frameIntervalMS <= 0 {
		return -1
	}
	w := newT0FilmWindow(frameIntervalMS)
	for _, s := range steps {
		if w.push(s) > t0FilmCumulM {
			return s.endFrame
		}
	}
	return -1
}

// t0FilmDeparture est le premier mouvement d'une piste : sa frame et l'identite de son porteur.
type t0FilmDeparture struct {
	frame int
	xuid  string
}

// t0FilmDepartures rend, triee par frame, la premiere frame de mouvement de chaque piste qui
// en porte une, et le nombre de pistes exploitables (au moins deux points).
func t0FilmDepartures(tracks []T0FilmTrack, frameIntervalMS int) (deps []t0FilmDeparture, usable int) {
	for i := range tracks {
		if len(tracks[i].Points) < 2 {
			continue
		}
		usable++
		f := t0FilmFirstMovement(t0FilmSteps(tracks[i].Points, frameIntervalMS), frameIntervalMS)
		if f >= 0 {
			deps = append(deps, t0FilmDeparture{frame: f, xuid: tracks[i].XUID})
		}
	}
	sort.Slice(deps, func(a, b int) bool { return deps[a].frame < deps[b].frame })
	return deps, usable
}

// t0FilmBurst compte les PARTANTS DISTINCTS dont le premier mouvement tombe a moins de
// `windowMS` du tout premier. `deps` doit etre trie par frame.
//
// PAR XUID, pas par piste : deux vies du meme joueur ne font pas deux partants. Une piste que
// le fil des morts n'a pas nommee compte pour UN partant — le film n'offre aucun moyen de la
// replier sur un joueur, et deux vies anonymes qui partent dans la meme seconde du coup
// d'envoi sont deux joueurs (une seconde vie suppose une mort, impossible avant le depart).
func t0FilmBurst(deps []t0FilmDeparture, frameIntervalMS int, windowMS int64) int {
	if len(deps) == 0 {
		return 0
	}
	base := int64(deps[0].frame) * int64(frameIntervalMS)
	seen := map[string]bool{}
	n := 0
	for _, d := range deps {
		if int64(d.frame)*int64(frameIntervalMS)-base > windowMS {
			break
		}
		if d.xuid == "" {
			n++
			continue
		}
		if !seen[d.xuid] {
			seen[d.xuid] = true
			n++
		}
	}
	return n
}

// DetectT0Film date le coup d'envoi du match sur l'horloge du fil des eliminations, ou REFUSE.
//
// `originMs` est l'instant de la frame 0 sur cette meme horloge (cf. origin.go) : le resultat
// est `originMs + frameDuPremierMouvement * frameIntervalMS`, une addition, pas un recalage.
//
// PUR au sens du calcul — la seule sortie hors valeur de retour est le journal du refus, qui
// suit la regle du depot : jamais de degradation silencieuse.
func DetectT0Film(tracks []T0FilmTrack, frameIntervalMS int, originMs int64,
	matchID string) (*int64, *T0FilmCoverage) {
	cov := &T0FilmCoverage{}
	if frameIntervalMS <= 0 {
		return t0FilmRefuse(cov, t0FilmReasonNoFrameStep, matchID)
	}
	deps, usable := t0FilmDepartures(tracks, frameIntervalMS)
	cov.Tracks, cov.Moving = usable, len(deps)
	if len(deps) == 0 {
		return t0FilmRefuse(cov, t0FilmReasonNoMovement, matchID)
	}
	cov.MarginMs = int64(deps[0].frame) * int64(frameIntervalMS)
	cov.Burst = t0FilmBurst(deps, frameIntervalMS, t0FilmBurstMS)
	if cov.Burst < t0FilmMinBurst {
		return t0FilmRefuse(cov, t0FilmReasonSmallBurst, matchID)
	}
	if cov.MarginMs > t0FilmMaxDelayMS {
		return t0FilmRefuse(cov, t0FilmReasonTooLate, matchID)
	}
	cov.Detected = true
	t0 := originMs + cov.MarginMs
	return &t0, cov
}

// t0FilmRefuse journalise le refus et rend l'absence de valeur. Le contexte est vide, et c'est
// exact : cet assembleur est HORS LIGNE, il n'y a aucun contexte de requete a propager.
func t0FilmRefuse(cov *T0FilmCoverage, reason, matchID string) (*int64, *T0FilmCoverage) {
	cov.Detected, cov.Reason = false, reason
	slog.WarnContext(context.Background(),
		"rejeu : coup d'envoi NON date par le film — le T0 de l'API reste la source",
		"match_id", matchID, "raison", reason, "pistes", cov.Tracks, "pistesEnMouvement",
		cov.Moving, "rafale", cov.Burst, "margeMs", cov.MarginMs)
	return nil, cov
}

// t0FilmTracksOf ramene les pistes publiees du document a ce que le detecteur lit.
func t0FilmTracksOf(tracks []Track) []T0FilmTrack {
	out := make([]T0FilmTrack, 0, len(tracks))
	for i := range tracks {
		pts := make([]T0FilmPoint, len(tracks[i].Points))
		for j, p := range tracks[i].Points {
			pts[j] = T0FilmPoint{T: p.T, X: p.X, Y: p.Y, Z: p.Z}
		}
		out = append(out, T0FilmTrack{XUID: tracks[i].XUID, Points: pts})
	}
	return out
}
