package replay

// bomb_armings.go — LE CÂBLAGE du calque de L'ARMEMENT DE LA BOMBE : ce que l'appelant
// fournit, ce que `BuildFromFilm` décode, et ce que `BuildFromPositions` en assemble.
//
// Il vit à part de `build.go` pour la même raison que `build_zones.go` et
// `build_objectives_live.go` : l'assemblage garde UNE ligne par calque, le détail vit à côté
// de la donnée qu'il produit. La FORME publiée vit dans document_bomb_armings.go, avec la
// provenance de la mesure (protocole du 2026-09-01, 0/1000).
//
// # DEUX LECTURES, ET UNE SEULE EST FAITE ICI
//
//	l'ANNEAU `ti=12 i14`    ICI, dans `BuildFromFilm`, comme les autres balayages filmdec ;
//	l'HORLOGE du manifeste  l'appelant (`replaybuild`) — ce paquet est title-agnostic et ne
//	                        connaît pas le cache film du titre ;
//	les EXPLOSIONS          déjà posées dans `doc.Objectives` (`bomb_detonations`, statborg) —
//	                        un seul décodage du statborg, jamais un second lecteur.
//
// # LA GARDE DE MODE EST DOUBLE, ET C'EST LE CŒUR DE LA DÉGRADATION PROPRE
//
// Le signal est prouvé sur NEUTRAL BOMB (13/13, CV 0,016) et HUSKY RAID (4/4, CV 0,016) ; il
// NE TIENT PAS sur ONE BOMB (CV 0,725, 87/1000 tirages nuls font aussi bien — mesure du
// 2026-09-01, réserve explicite du portage).
//
//	GARDE 1 — LE NOM. L'appelant ne pose `Scanned` que sur une variante d'Assaut dont le nom
//	  ne porte pas « one bomb » (`replaybuild.isArmableBombVariant`). Le balayage n'est même
//	  pas payé ailleurs — même règle de coût que le marqueur CTF et les zones.
//	GARDE 2 — LA CONFRONTATION LOCALE. Chaque explosion visible du film doit être précédée
//	  d'un armement dans la fenêtre de la mèche (4,93 s ± 0,6 s). UNE SEULE explosion
//	  orpheline retient le calque ENTIER à la source : sur un film qui contredit la lecture,
//	  rien n'est publié plutôt que des comptes à rebours faux. Elle est TOUT-OU-RIEN par
//	  film, jamais un filtre par événement — filtrer les armements sur les explosions
//	  rendrait la validation circulaire.
//
// Un armement SANS explosion reste publié quand la confrontation tient : la bombe peut être
// désamorcée pendant la mèche, et ce silence-là est le comportement du jeu, pas un défaut.

import (
	"log/slog"
	"sort"

	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/analysis/filmsource"
	"levelup/go-api/internal/analysis/objectiveevents"
)

// BombFuseMS est la MÈCHE : le délai entre l'instant armé (fin de montée de l'anneau) et
// l'explosion. Constante MOTEUR, pas un réglage de mode — protocole du 2026-09-01 : délai
// médian 4,93 s, CV 0,016 sur les 13 explosions Neutral Bomb, confirmé 4/4 sur Husky Raid
// (hold six fois plus court, même mèche), 0/1000 tirages nuls aussi bien.
const BombFuseMS = 4930

// bombFuseWindowMS est la demi-fenêtre de la confrontation locale : une explosion est
// COUVERTE si un armement la précède de BombFuseMS ± cette tolérance. Elle absorbe la
// dispersion mesurée (écart-type ~80 ms = CV 0,016) avec une marge large, et reste très en
// dessous de ce qui séparerait la mèche d'un autre mécanisme (~17 s observés sur One Bomb).
const bombFuseWindowMS = 600

// bombPairToleranceMS borne la déduplication des paires : les navpoints vont PAR PAIRES
// (+12 d'écart de slot, un par camp) et répliquent le MÊME anneau — deux fins de montée à
// moins de cette tolérance sont UN armement. Deux armements réels ne peuvent pas être si
// proches : le hold le plus court mesuré (~0,9 s, Husky Raid) plus la mèche les séparent
// d'au moins ~6 s.
const bombPairToleranceMS = 500

// bombArmedFullQuantum est le QUANTUM PLEIN : une montée n'est un ARMEMENT que si son dernier
// échantillon l'atteint. Mesure du gate (2026-09-01, diagnostic sur les trois films du
// portage) : les 7 montées confirmées par une explosion à la mèche finissent TOUTES à q=254
// (7/7, Neutral et Husky) ; les montées plafonnées à q=253 (~4,9 s, 130 -> 253, après chaque
// explosion et à chaque apparition de bombe) sont l'ANIMATION DE RECHARGE du marqueur, jamais
// un armement (0/12) ; en dessous, des holds relâchés. Le plein est PAIR — cohérent avec la
// réserve « quantum toujours pair » du chantier ti=11. Si une montée réelle finissait sous le
// plein (échantillon final perdu), l'armement manquerait et la confrontation locale retiendrait
// le calque du film : la défaillance dégrade vers le silence, jamais vers un faux événement.
const bombArmedFullQuantum = 254

// BombInput est CE QUE L'APPELANT FOURNIT de l'armement, plus ce que `BuildFromFilm` y dépose.
type BombInput struct {
	// Scanned est la GARDE DE MODE de L'ARMEMENT, posée par l'appelant selon
	// `game_variant_name` : vraie sur les seules variantes d'Assaut où le canal de l'anneau
	// est prouvé (jamais One Bomb — cf. l'en-tête). Faux : ni balayage, ni calque, ni
	// couverture.
	Scanned bool
	// CarryScanned est la GARDE DE MODE du PORTAGE (`bombCarries`, schéma 30) : vraie sur
	// TOUTE variante de la famille bomb, One Bomb COMPRISE — le canal des armes tenues n'est
	// pas celui de l'anneau, et le négatif One Bomb ne le concerne pas (cf.
	// document_bomb_carries.go). Faux : ni calque ni couverture de portage.
	CarryScanned bool
	// ChunkStartMS est l'horloge du manifeste (start_ms par index de chunk), lue par
	// l'appelant au cache film du titre — la même base que les explosions du statborg.
	ChunkStartMS map[int]int
	// Reads est déposé par `BuildFromFilm` : les lectures de l'anneau. L'appelant ne le
	// remplit pas.
	Reads []filmdec.NavpointRadialRead
}

// decodeFilmBombReads balaye l'anneau ti=12 et JOURNALISE ce qu'il en est.
//
// IL NE BALAYE QUE LES FILMS RECONNUS ARMABLES par l'appelant (même règle de coût que le
// marqueur de portage CTF) : le balayage est une marche de tous les paquets delta du film,
// et hors Assaut le calque est vide de toute façon. L'échec n'est pas fatal : le rejeu sort
// sans compte à rebours, jamais avec un compte à rebours deviné.
//
// HORS LIGNE — appelée par BuildFromFilm, sous LockProcessDecode.
func decodeFilmBombReads(film *filmsource.Film, matchID string, in BombInput) []filmdec.NavpointRadialRead {
	if !in.Scanned {
		return nil
	}
	if len(in.ChunkStartMS) == 0 {
		slog.Warn("armement : film armable sans horloge de manifeste — calque non construit",
			"match_id", matchID)
		return nil
	}
	sc, err := filmdec.ScanNavpointRadial(film, in.ChunkStartMS)
	if err != nil {
		slog.Warn("armement : anneau ti=12 illisible — rejeu sans compte a rebours",
			"err", err, "match_id", matchID)
		return nil
	}
	if sc.Truncated {
		slog.Warn("armement : recolte TRONQUEE au plafond de lectures — armements tardifs possibles manquants",
			"match_id", matchID)
	}
	// paquetsSansHorloge est un compte NOMINAL non nul (le footer n'a pas de paquet delta de
	// base) : il s'imprime avec le bilan, il n'alarme pas.
	slog.Info("armement : anneau ti=12 balaye",
		"match_id", matchID, "slots", sc.SlotsObserved, "records", sc.Records,
		"marches", sc.Walked, "chainees", sc.Chained, "lectures", len(sc.Reads),
		"paquetsSansHorloge", sc.PacketsNoClock)
	return sc.Reads
}

// attachBombArmings pose le calque de l'armement sur le document, avec sa couverture et son
// journal. Les explosions de la confrontation locale viennent de `doc.Objectives`, DÉJÀ
// posées — l'appel vient donc après `attachObjectiveActions`.
func attachBombArmings(doc *ReplayDocument, opt Options, c scoreClock) {
	if !opt.Bomb.Scanned {
		return
	}
	armings, cov := buildBombArmings(opt.Bomb.Reads, bombDetonationTimes(doc.Objectives), c)
	doc.BombArmings = armings
	if doc.Coverage != nil {
		doc.Coverage.BombArmings = cov
	}
	logBombArmings(doc.MatchID, cov)
}

// buildBombArmings applique la chaîne mesurée : montées contiguës -> déduplication de paire
// -> confrontation locale -> grille de frames. Pur, testable sans film.
func buildBombArmings(reads []filmdec.NavpointRadialRead, detonations []int,
	c scoreClock) ([]BombArming, *BombArmingsCoverage) {
	cov := &BombArmingsCoverage{Scanned: true, Reads: len(reads), Detonations: len(detonations)}
	rises := filmdec.NavpointContiguousRises(reads)
	cov.Rises = len(rises)
	full := fullQuantumRises(rises, cov)
	armed := dedupPairedRises(full, cov)
	cov.Armed = len(armed)
	cov.DetonationsCovered = countCoveredDetonations(armed, detonations)
	if cov.DetonationsCovered < cov.Detonations {
		// LA CONFRONTATION LOCALE ÉCHOUE : ce film contredit la lecture Neutral Bomb (cf.
		// l'en-tête — One Bomb mal nommé, variante inconnue, film dégradé). Tout-ou-rien.
		cov.Suppressed = true
		return nil, cov
	}
	out := make([]BombArming, 0, len(armed))
	for _, r := range armed {
		t, ok := c.frameOf(int(r.EndMS))
		if !ok {
			cov.OutOfWindow++
			continue
		}
		startT, ok := c.frameOf(int(r.StartMS))
		if !ok {
			startT = 0 // début de hold avant la frame 0 : le compte à rebours, lui, tient
		}
		out = append(out, BombArming{T: t, TimeMS: int(r.EndMS),
			StartT: startT, StartMS: int(r.StartMS), FuseMS: BombFuseMS})
	}
	cov.Published = len(out)
	return out, cov
}

// fullQuantumRises ne garde que les montées dont le dernier échantillon atteint le QUANTUM
// PLEIN — la définition de « bombe armée » (cf. bombArmedFullQuantum). Le filtre passe AVANT
// la déduplication : sur une paire dont un miroir manquerait son dernier échantillon, c'est le
// miroir PLEIN qui survit et date l'armement.
func fullQuantumRises(rises []filmdec.NavpointRise, cov *BombArmingsCoverage) []filmdec.NavpointRise {
	out := make([]filmdec.NavpointRise, 0, len(rises))
	for _, r := range rises {
		if r.QEnd < bombArmedFullQuantum {
			cov.BelowFull++
			continue
		}
		out = append(out, r)
	}
	return out
}

// dedupPairedRises fond les fins de montée à moins de bombPairToleranceMS l'une de l'autre en
// UN armement : le début retenu est le plus tôt (le hold a commencé là), la fin la plus
// tardive (la statistique du protocole — « dernière fin de montée avant l'explosion » — date
// la mèche sur le miroir le plus tardif de la paire). L'entrée arrive triée par (EndMS, Slot).
func dedupPairedRises(rises []filmdec.NavpointRise, cov *BombArmingsCoverage) []filmdec.NavpointRise {
	var out []filmdec.NavpointRise
	for _, r := range rises {
		if n := len(out); n > 0 && r.EndMS-out[n-1].EndMS <= bombPairToleranceMS {
			cov.PairMerged++
			if r.StartMS < out[n-1].StartMS {
				out[n-1].StartMS = r.StartMS
			}
			out[n-1].EndMS = r.EndMS
			continue
		}
		out = append(out, r)
	}
	return out
}

// countCoveredDetonations compte les explosions précédées d'un armement dans la fenêtre de la
// mèche : délai (explosion - armé) dans [BombFuseMS - bombFuseWindowMS, BombFuseMS + bombFuseWindowMS].
func countCoveredDetonations(armed []filmdec.NavpointRise, detonations []int) int {
	covered := 0
	for _, det := range detonations {
		lo, hi := int32(det-BombFuseMS-bombFuseWindowMS), int32(det-BombFuseMS+bombFuseWindowMS)
		i := sort.Search(len(armed), func(k int) bool { return armed[k].EndMS >= lo })
		if i < len(armed) && armed[i].EndMS <= hi {
			covered++
		}
	}
	return covered
}

// bombDetonationTimes rend les instants (horloge du film) des explosions déjà publiées dans
// le calque des actions d'objectif, triés.
func bombDetonationTimes(actions []ObjectiveAction) []int {
	var out []int
	for _, a := range actions {
		if a.Stat == objectiveevents.StatBombDetonations {
			out = append(out, a.TimeMS)
		}
	}
	sort.Ints(out)
	return out
}

// logBombArmings journalise ce que le calque publie — et pourquoi il peut se taire.
func logBombArmings(matchID string, cov *BombArmingsCoverage) {
	if cov.Suppressed {
		slog.Warn("rejeu : armement RETENU A LA SOURCE — une explosion sans armement a la meche "+
			"contredit la lecture (variante non couverte ?)",
			"match_id", matchID, "explosions", cov.Detonations,
			"couvertes", cov.DetonationsCovered, "armements", cov.Armed, "montees", cov.Rises)
		return
	}
	slog.Info("rejeu : armement de la bombe",
		"match_id", matchID, "lectures", cov.Reads, "montees", cov.Rises,
		"sousLePlein", cov.BelowFull, "armements", cov.Armed, "paireFondue", cov.PairMerged,
		"publies", cov.Published, "horsFenetre", cov.OutOfWindow,
		"explosions", cov.Detonations, "couvertes", cov.DetonationsCovered)
}
