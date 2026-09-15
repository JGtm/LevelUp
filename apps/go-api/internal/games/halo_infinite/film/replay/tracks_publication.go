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
	"sort"

	"levelup/go-api/internal/games/halo_infinite/film/filmdec"
	"levelup/go-api/internal/games/halo_infinite/film/replay/fallback"
)

// DefaultMinPoints est le nombre minimal de points pour qu'une vie soit publiée.
//
// IL VIT ICI ET PLUS DANS `build.go` (lot 1.6.5, déplacement pur) : ce fichier EST celui qui décide
// quelles vies sont publiées et qui tient les comptes de ce qu'il refuse ; laisser le seuil chez
// l'assembleur y aurait porté la doctrine et le fichier, déjà au-delà du seuil de 500 lignes du
// dépôt, l'aurait payée.
//
// IL VAUT 1 DEPUIS LE 2026-09-14, PAR DÉCISION UTILISATEUR : « si le film le dit, on publie ». Il
// valait 2 depuis l'origine du calque, sous la phrase « une vie d'un seul échantillon n'est pas une
// trajectoire » — qui décrivait un RENDU, pas une donnée : le film écrit une position pour un
// joueur à un instant, et l'artefact la taisait. Le compteur `coverage.tracks.refusedMinPoints`,
// posé au schéma 55 précisément pour poser la question avec un chiffre, tombe donc à 0.
//
// CE QUE LA MESURE DIT DE CES VIES (2026-09-14, `vies_un_echantillon_test.go`, 20 vies sur les huit
// builds) : UNE seule porte une mort ÉCRITE à son instant, CINQ sont la dernière image d'un slot que
// la réplication n'a plus jamais repris, QUATORZE sont ORPHELINES — leur vie se ferme sur un trou de
// réplication et la mort la plus proche du même joueur est à 0,95 s à 300 s. Ce n'est pas une raison
// de les filtrer (décision utilisateur) : c'est un DÉFAUT DE LECTURE nommé, consigné au plan §4, et
// le publier est ce qui le rend visible.
const DefaultMinPoints = 1

// decoupeDesTraces porte ce dont la publication des traces a besoin. Une structure plutôt que
// des paramètres de plus : le dépôt en borne cinq.
type decoupeDesTraces struct {
	// origin / step : l'axe de frames, en microsecondes de l'horloge du film.
	origin, step uint64
	// minPoints : le seuil de publication d'une vie (cf. DefaultMinPoints).
	minPoints int
	// scoped : l'état de lunette à un instant, d'une AUTRE source que la position.
	scoped func(slot uint32, tsUS uint64) int
	// vies : LA DÉCOUPE, telle que le registre d'identité l'a établie sur ce que le film ÉCRIT
	// (cf. lives_decoupe.go). Vide = le film ne nomme aucune vie : la publication se rabat alors
	// sur le seuil de trou, et ce repli est COMPTÉ.
	vies []lifeSpan
	// fb : le compteur de replis de la cuisson (nil-safe).
	fb *fallback.Compteur
}

// decimateTracks projette les positions sur la grille de frames (un point par slot et par
// frame, le premier observé gagne) et produit UNE TRACK PAR VIE.
//
// # LA DÉCOUPE NE SE DÉCIDE PLUS ICI (lot 1.9.13)
//
// Elle s'appliquait ici au SEUIL DE TROU (`lifeGapUS`), la même règle que `buildLifeSpans`
// écrivait de son côté — deux écritures de la même règle, qui ne pouvaient que diverger. La
// découpe vient désormais des VIES du registre, c'est-à-dire de ce que le film ÉCRIT : une vie
// finit à une mort écrite, à une apparition de corps, à une fin de manche ou à la fin du film.
// Un trou de réplication n'y est plus une fin : c'est une LACUNE, comptée
// (`coverage.tracks.gaps`) et portée par le point qui la suit (`Point.G`), pour que le client ne
// trace aucun segment au travers.
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
func decimateTracks(sorted []filmdec.BipedPosition, in decoupeDesTraces) ([]Track, TrackCoverage) {
	origin, step := in.origin, in.step
	bornes := bornesDesVies(sorted, in)
	type acc struct {
		done      [][]Point // les vies CLOSES de ce slot, dans l'ordre
		pts       []Point
		lastFrame int
		lastUS    uint64
		vie       int // l'indice, dans `bornes[slot]`, de la vie en cours
		lacuneMS  int // la lacune que le PROCHAIN point publié portera, en ms ; 0 = aucune
	}
	accs := map[uint32]*acc{}
	var order []uint32
	cov := TrackCoverage{MinPoints: in.minPoints}
	for _, p := range sorted {
		if !p.HasWorld { // quantum sans bornes de carte : pas une coordonnée, on ne publie pas
			continue
		}
		frame := int((p.TimestampUS - origin) / step)
		a := accs[p.Slot]
		if a == nil {
			a = &acc{lastFrame: -1, vie: -1}
			accs[p.Slot] = a
			order = append(order, p.Slot)
		}
		vie := vieDuPoint(bornes[p.Slot], a.vie, p.TimestampUS)
		switch {
		case len(a.pts) > 0 && vie != a.vie:
			// FIN DE VIE ÉCRITE : la track courante se clôt, la suivante s'ouvre.
			a.done = append(a.done, a.pts)
			a.pts, a.lastFrame = nil, -1
		case len(a.pts) > 0 && int64(p.TimestampUS)-int64(a.lastUS) > lifeGapUS:
			// LACUNE : la réplication s'est tue au-delà du seuil SANS que le film ferme la vie.
			// Le point qui rouvre la piste la porte, pour que le client ne trace rien au travers.
			cov.Gaps++
			ms := int((int64(p.TimestampUS) - int64(a.lastUS)) / 1000)
			cov.GapMS += ms
			a.lacuneMS = ms
		}
		a.vie, a.lastUS = vie, p.TimestampUS
		if frame == a.lastFrame {
			continue
		}
		a.lastFrame = frame
		pt := Point{T: frame, X: round2(p.X), Y: round2(p.Y), Z: round2(p.Z), G: a.lacuneMS}
		a.lacuneMS = 0
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
		if in.scoped != nil {
			pt.S = in.scoped(p.Slot, p.TimestampUS)
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
	for _, slot := range order {
		a := accs[slot]
		for _, pts := range append(a.done, a.pts) {
			if len(pts) < in.minPoints {
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

// bornesDesVies groupe par slot la DÉCOUPE que le registre a établie, triée chronologiquement.
//
// LE REPLI EST ICI, ET IL EST COMPTÉ (D14) : quand le registre ne rend AUCUNE vie — un film dont
// la table des joueurs est vide, donc aucun lien à poser — la publication n'a rien à suivre et se
// rabat sur le seuil de trou. Ce n'est pas un silence : `repli_vie_coupee_au_trou_de_replication`
// se déclenche, une fois par cuisson.
func bornesDesVies(sorted []filmdec.BipedPosition, in decoupeDesTraces) map[uint32][]lifeSpan {
	vies := in.vies
	if len(vies) == 0 {
		in.fb.Declenche(fallback.NomVieCoupeeAuTrouDeReplication)
		vies = buildLifeSpans(indexBySlot(sorted))
	}
	out := map[uint32][]lifeSpan{}
	for _, l := range vies {
		out[l.slot] = append(out[l.slot], l)
	}
	for s := range out {
		sort.SliceStable(out[s], func(i, j int) bool { return out[s][i].from < out[s][j].from })
	}
	return out
}

// vieDuPoint rend l'indice de la vie du slot qui couvre cet instant.
//
// LA RECHERCHE REPART DE LA VIE COURANTE : les points d'un slot arrivent triés et les vies sont
// disjointes, donc l'indice ne recule jamais. Aucun instant hors de toute vie n'est possible (les
// vies sont bâties sur ces mêmes points) ; si cela arrivait, la vie courante est conservée —
// jamais une vie neuve ouverte sur une borne qu'on n'a pas su lire.
func vieDuPoint(spans []lifeSpan, courant int, tsUS uint64) int {
	t := int64(tsUS)
	for i := max(courant, 0); i < len(spans); i++ {
		if t <= spans[i].to {
			return i
		}
	}
	return courant
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
	// Gaps est le nombre de LACUNES des traces publiées, et GapMS leur durée totale en
	// millisecondes (lot 1.9.13). Une lacune est un silence de réplication de plus de
	// `lifeGapUS` (5 s) À L'INTÉRIEUR d'une vie — le film ne ferme pas la vie là, il se tait.
	//
	// LES DEUX, parce qu'ils ne disent pas la même chose : vingt lacunes de 6 s sont une
	// réplication hachée, une lacune de trois minutes est un joueur que le film cesse de suivre.
	// Avant ce lot ces silences DÉCOUPAIENT la vie et ne se comptaient nulle part : une vie
	// coupée en quatre était indistinguable de quatre vies (mesure : 208 des 212 coupures des
	// 8 builds n'étaient justifiées par RIEN — cf. lives_decoupe.go).
	Gaps  int `json:"gaps"`
	GapMS int `json:"gapMs"`
}

// logTrackCoverage journalise ce que la publication des traces a refusé et ce que le film a tu.
//
// JOURNALISE, JAMAIS AVALÉ (règle n° 3 du dépôt) : le refus du seuil était muet des DEUX côtés —
// ni compteur publié, ni ligne de journal. Les LACUNES l'étaient plus encore : elles découpaient
// les vies, donc elles ne ressemblaient même pas à un silence.
func logTrackCoverage(matchID string, c TrackCoverage) {
	if c.Gaps > 0 {
		slog.Info("rejeu : lacunes de replication DANS une vie",
			"match_id", matchID, "lacunes", c.Gaps, "duree_ms", c.GapMS,
			"vies", c.Published, "seuil_ms", lifeGapUS/1000)
	}
	if c.RefusedMinPoints == 0 {
		return
	}
	slog.Info("rejeu : vies refusees par le seuil de publication",
		"match_id", matchID, "vies", c.RefusedMinPoints, "points", c.RefusedPoints,
		"seuil", c.MinPoints, "viesPubliees", c.Published)
}
