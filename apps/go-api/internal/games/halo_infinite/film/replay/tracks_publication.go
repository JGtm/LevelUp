package replay

// tracks_publication.go — LA PUBLICATION DES TRACES, ET CE QU ELLE REFUSE.
//
// SORTI DE `build.go` le 2026-09-14 (lot 1.0, revue R1, constat R1-4). `build.go` pesait 607
// lignes AVANT ce lot — deja au-dela du seuil du depot, et c est le lot 2.7 qui le scindera :
// y poser le comptage de refus, son type et son journal l aurait grossi pour rien. La coupure
// suit une responsabilite : ici vit tout ce qui decide QUELLES vies sont publiees et tient les
// comptes de ce qui ne l est pas ; `build.go` garde l assemblage du document.
//
// DEPLACEMENT PUR pour `decimateTracks` : aucune ligne de logique ne change.

import (
	"log/slog"

	"levelup/go-api/internal/games/halo_infinite/film/filmdec"
)

// decimateTracks projette les positions sur la grille de frames (un point par slot et par
// frame, le premier observé gagne) et produit UNE TRACK PAR VIE — un slot qui disparaît plus
// de `lifeGapUS` puis revient ouvre une nouvelle track, la MÊME règle de découpe que
// `buildLifeSpans` (lot identité des vies, 2026-09-02).
//
// POURQUOI PAR VIE ET PLUS PAR SLOT. Une track unique par slot fusionnait les vies d'un slot
// RECYCLÉ (partant remplacé par un arrivant ou un bot) : le premier porteur nommé gardait
// tout l'intervalle, le second n'avait aucune vie — sa fiche restait « Éliminé /
// Réapparition ? » pendant que son corps se déplaçait sous le nom du premier. Le contrat
// client (buildSlotOwnership, résolveurs frame-aware par slot) attend des vies disjointes.
// L'ordre reste celui de première apparition du slot, les vies d'un slot en ordre
// chronologique — déterministe, artefact diffable.
//
// ELLE REND CE QU'ELLE REFUSE (schéma 55, lot 1.0.4). Le seuil `minPoints` écartait des vies
// SANS AUCUN COMPTEUR : un artefact publiant 90 traces là où le film en portait 95 était
// indistinguable d'un film à 90 vies. Les deux compteurs disent combien de VIES et combien de
// POINTS le seuil a retirés — cf. TrackCoverage, publié en `coverage.tracks`.
func decimateTracks(sorted []filmdec.BipedPosition, origin, step uint64, minPoints int,
	scoped func(slot uint32, tsUS uint64) int) ([]Track, TrackCoverage) {
	type acc struct {
		done      [][]Point // les vies CLOSES de ce slot, dans l'ordre
		pts       []Point
		lastFrame int
		lastUS    uint64
	}
	accs := map[uint32]*acc{}
	var order []uint32
	for _, p := range sorted {
		if !p.HasWorld { // quantum sans bornes de carte : pas une coordonnée, on ne publie pas
			continue
		}
		frame := int((p.TimestampUS - origin) / step)
		a := accs[p.Slot]
		if a == nil {
			a = &acc{lastFrame: -1}
			accs[p.Slot] = a
			order = append(order, p.Slot)
		}
		// Trou au-delà de lifeGapUS = NOUVELLE VIE : la track courante se clôt, la suivante
		// s'ouvre. Même seuil que buildLifeSpans — deux découpes divergentes rendraient le
		// nommage par vie inappariable.
		if len(a.pts) > 0 && int64(p.TimestampUS)-int64(a.lastUS) > lifeGapUS {
			a.done = append(a.done, a.pts)
			a.pts = nil
			a.lastFrame = -1
		}
		a.lastUS = p.TimestampUS
		if frame == a.lastFrame {
			continue
		}
		a.lastFrame = frame
		pt := Point{T: frame, X: round2(p.X), Y: round2(p.Y), Z: round2(p.Z)}
		if h, ok := p.AimHeadingDeg(); ok { // cap de visée du MÊME record (i21), si répliqué
			pt.H = headingForJSON(h)
		}
		// ÉLÉVATION du MÊME record et du MÊME composant que le cap (le R(11) qui suit le
		// R(12) d'i21) : les deux angles arrivent ensemble ou pas du tout, `AimPitchDeg`
		// partageant la validité `HasYaw` avec `AimHeadingDeg`. Publier l'un sans l'autre
		// n'a donc aucun sens — et l'absence de `p` sur un point qui porte `h` dit « à
		// plat », pas « inconnu » (cf. Point.P).
		if pitch, ok := p.AimPitchDeg(); ok {
			pt.P = pitchForJSON(pitch)
		}
		// LUNETTE : etat a bascule, d'une AUTRE source que les deux angles ci-dessus — on le
		// consulte a l'instant du point au lieu de le lire dedans (cf. Point.S, zoom_state.go).
		if scoped != nil {
			pt.S = scoped(p.Slot, p.TimestampUS)
		}
		// Vitalité du MÊME record que la position (i4 / i5). La décimation garde le PREMIER
		// échantillon de chaque frame : si deux records du même slot tombent dans la même
		// frame de 100 ms et que seul le second porte le bouclier, il est perdu. Cela
		// n'invente rien — c'est une perte, pas une erreur — et le témoin publié est mesuré
		// sur les positions NON décimées.
		// Témoin : P(bouclier nul | 500 ms avant une mort connue) = 50,49 % contre 38,18 %
		// chez un vivant à plus de 5 s d'une mort, soit un rapport de 1,32x — FAIBLE, et
		// c'est normal : le film ne réplique le bouclier que lorsqu'il CHANGE, donc une
		// mesure de bouclier est déjà une mesure de combat. Ce qui porte le rendu est le
		// témoin de FORME (27 404/27 404 quanta dans [0,64]), pas ce rapport.
		if sh, ok := p.ShieldAt(); ok {
			pt.Sh = fractionForJSON(sh)
		}
		if hp, ok := p.HealthAt(); ok {
			pt.Hp = fractionForJSON(hp)
		}
		a.pts = append(a.pts, pt)
	}
	tracks := make([]Track, 0, len(order))
	cov := TrackCoverage{MinPoints: minPoints}
	for _, slot := range order {
		a := accs[slot]
		for _, pts := range append(a.done, a.pts) {
			if len(pts) < minPoints {
				// LE REFUS SE COMPTE, ET IL NE SE DÉCLENCHE PLUS AU SEUIL PAR DÉFAUT : celui-ci
				// vaut 1 depuis le lot 1.6.5 (« si le film le dit, on publie »), donc ce compteur
				// est à 0 sur toute cuisson qui ne règle pas `Options.MinPoints`. Il reste pour
				// l'appelant qui le règle, et parce qu'un compteur à 0 qui pourrait monter est ce
				// qui rend un retour en arrière visible.
				cov.RefusedMinPoints++
				cov.RefusedPoints += len(pts)
				continue
			}
			tracks = append(tracks, Track{
				Slot:       slot,
				Team:       -1,
				Points:     pts,
				StartFrame: pts[0].T,
				EndFrame:   pts[len(pts)-1].T,
			})
			cov.Published++
			cov.PublishedPoints += len(pts)
		}
	}
	return tracks, cov
}

// TrackCoverage est ce que le SEUIL DE PUBLICATION des traces retient et refuse.
//
// # LE SILENCE QU'ELLE ROMPT (schéma 55, lot 1.0.4 du PLAN_DECODEUR_FILM)
//
// `decimateTracks` écarte toute vie dont la trajectoire décimée porte moins de `MinPoints`
// échantillons — une vie d'un seul point n'est pas une trajectoire. Le refus était MUET depuis
// l'origine du calque : ni compteur publié, ni ligne de journal. Un artefact publiant 90 traces
// là où le film en porte 95 était donc indistinguable d'un film à 90 vies, et TOUT lecteur qui
// rapporte un compte de vies au film (le registre d'identité, la couverture du pont, l'écran)
// travaillait sur un dénominateur amputé sans le savoir.
//
// # CE QUE LE CHIFFRE A PERMIS DE TRANCHER (lot 1.6.5, 2026-09-14)
//
// La question « faut-il publier les vies d'un seul échantillon ? » appartenait à l'utilisateur, et
// ces compteurs existaient pour la lui poser avec un chiffre : 20 vies sur les huit builds, 0 à 6
// par film. Il a tranché — « si le film le dit, on publie » — et `DefaultMinPoints` est passé de 2
// à 1. Ces compteurs restent : à 0 sur toute cuisson au seuil par défaut, ils rendent visible un
// appelant qui règle `Options.MinPoints` et tout retour en arrière.
type TrackCoverage struct {
	// Published / PublishedPoints : les traces publiées et leurs points, c'est-à-dire le
	// DÉNOMINATEUR sans lequel un compte de refus ne se juge pas.
	Published       int `json:"published"`
	PublishedPoints int `json:"publishedPoints"`
	// RefusedMinPoints est le nombre de VIES que le seuil a écartées, et RefusedPoints le
	// nombre de points qu'elles portaient. Les deux, parce qu'ils ne disent pas la même chose :
	// dix vies d'un point sont un pool de slots qui s'ouvre et se referme, une vie de dix points
	// refusée serait un seuil mal réglé.
	RefusedMinPoints int `json:"refusedMinPoints"`
	RefusedPoints    int `json:"refusedPoints"`
	// MinPoints est le seuil APPLIQUÉ sur ce document. Il voyage avec ses conséquences : un
	// compte de refus ne se relit pas sans savoir contre quoi il a été mesuré, et l'appelant
	// peut le régler (`Options.MinPoints`).
	MinPoints int `json:"minPoints"`
}

// logTrackCoverage journalise ce que le seuil a refusé, quand il a refusé quelque chose.
//
// JOURNALISE, JAMAIS AVALÉ (règle n° 3 du dépôt) : ce refus était muet des DEUX côtés — ni
// compteur publié, ni ligne de journal. Le seuil ne bouge pas ; ce qu'il retire se voit
// désormais dans l'artefact ET dans les logs de la cuisson.
func logTrackCoverage(matchID string, c TrackCoverage) {
	if c.RefusedMinPoints == 0 {
		return
	}
	slog.Info("rejeu : vies refusees par le seuil de publication",
		"match_id", matchID, "vies", c.RefusedMinPoints, "points", c.RefusedPoints,
		"seuil", c.MinPoints, "viesPubliees", c.Published)
}
