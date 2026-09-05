package replay

// bomb_armings.go — LE CÂBLAGE du calque de L'ARMEMENT DE LA BOMBE : ce que l'appelant
// fournit, ce que `BuildFromFilm` décode, et ce que `BuildFromPositions` en assemble.
//
// Il vit à part de `build.go` pour la même raison que `build_zones.go` et
// `build_objectives_live.go` : l'assemblage garde UNE ligne par calque, le détail vit à côté
// de la donnée qu'il produit. La FORME publiée vit dans document_bomb_armings.go, avec la
// provenance de la mesure.
//
// # DEUX LECTURES, ET UNE SEULE EST FAITE ICI
//
//	l'ANNEAU `ti=12 i14`    ICI, dans `BuildFromFilm`, comme les autres balayages filmdec ;
//	l'HORLOGE du manifeste  l'appelant (`replaybuild`) — ce paquet est title-agnostic et ne
//	                        connaît pas le cache film du titre ;
//	les EXPLOSIONS          déjà posées dans `doc.Objectives` (`bomb_detonations`, statborg) —
//	                        un seul décodage du statborg, jamais un second lecteur.
//
// # LA LECTURE EST « MÈCHE PAUSABLE », ET C'EST CE QUI A LEVÉ LA GARDE DE NOM
//
// Jusqu'au 2026-09-04 ce fichier portait une garde de mode DOUBLE, dont la première écartait
// One Bomb PAR SON NOM (`replaybuild.isArmableBombVariant`) : sous la lecture SIMPLE — montée
// contiguë, mèche fixe de 4,93 s — le signal n'y tenait pas (CV 0,725 ; 87/1000 tirages nuls
// faisaient aussi bien). **CETTE GARDE EST LEVÉE** : la lecture pausable du 2026-09-01,
// portée ici depuis l'instrument (`filmdec/navpoint_ti12_meche_test.go`), explique One Bomb
// sans rien casser ailleurs — 9/9 explosions portées, médiane 16,18 s, CV 0,017, 0/1000
// tirages nuls, TÉMOINS Neutral Bomb 13/13 et Husky Raid 4/4 inchangés.
//
// Ce que la lecture ajoute, et d'où vient chaque terme (toutes MESURES du 2026-09-01) :
//
//	SEGMENT     le fait brut (`filmdec.NavpointSegments`) : suite contiguë d'un slot, SANS
//	            exigence de monotonie. Le cycle de RECHARGE du marqueur (130 -> 253 -> 127)
//	            finit à son minimum et sort de lui-même — un découpage en montées le prenait
//	            pour un armement.
//	ARMEMENT    segment qui FINIT À SON SOMMET (`EndsAtSummit`) ET dont le sommet atteint le
//	            QUANTUM PLEIN (`bombArmedFullQuantum`).
//	PAUSE       tenue de DÉSARMEMENT (`IsDisarmHold`) : elle SUSPEND la mèche. Le compte à
//	            rebours reprend où il en était quand la tenue s'interrompt.
//	DÉLAI       (explosion − fin d'armement) − somme des pauses du MÊME slot entre les deux.
//
// # LA GARDE 2 RESTE, ET C'EST ELLE QUI PROTÈGE DÉSORMAIS SEULE
//
// CONFRONTATION LOCALE, TOUT-OU-RIEN PAR FILM : chaque explosion visible du film doit être
// précédée d'un armement dans la fenêtre de sens (`BombFuseSenseWindowMS`), délai corrigé des
// pauses, ET les délais du film doivent S'ACCORDER ENTRE EUX (dispersion `bombFuseMaxCV`).
// Une seule explosion orpheline — ou des délais qui se contredisent — retient le calque
// ENTIER : sur un film qui contredit la lecture, rien n'est publié plutôt que des comptes à
// rebours faux. Jamais un filtre par événement : filtrer les armements sur les explosions
// rendrait la validation circulaire.
//
// LA MÈCHE N'EST PLUS UNE CONSTANTE UNIQUE : elle est MESURÉE SUR LE FILM (médiane des délais
// corrigés) et publiée avec chaque événement. C'est ce qui rend la lecture title-agnostic —
// 4,93 s en Neutral Bomb, 5,1 s en Husky Raid, 16,2 s en One Bomb sortent de la MÊME règle,
// sans que rien ne branche sur le nom de la variante. `BombFuseMS` ne subsiste que comme
// valeur de RÉFÉRENCE, DÉDUITE, pour le seul film qui ne porte aucune explosion à mesurer.
//
// Un armement SANS explosion reste publié quand la confrontation tient : la bombe peut être
// désamorcée pendant la mèche, et ce silence-là est le comportement du jeu, pas un défaut.

import (
	"log/slog"
	"math"
	"sort"

	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/analysis/objectiveevents"
)

// BombFuseMS est la mèche de RÉFÉRENCE : celle des variantes à mèche courte, mesurée à 4,93 s
// (délai médian, CV 0,016 sur les 13 explosions Neutral Bomb, confirmé 4/4 sur Husky Raid).
//
// ELLE N'EST PLUS LA MÈCHE DU CALQUE : depuis le 2026-09-04 la mèche est MESURÉE sur chaque
// film (cf. `measureBombFuse`). Cette constante ne sert plus que de valeur DÉDUITE quand le
// film ne porte AUCUNE explosion — il n'y a alors rien à mesurer, et publier un compte à
// rebours de référence vaut mieux que n'en publier aucun. `Coverage.Detonations == 0` dit
// exactement ce cas.
const BombFuseMS = 4930

// BombFuseSenseWindowMS est la FENÊTRE DE SENS de la confrontation : au-delà, une explosion
// n'est plus rapportée à un armement. C'est la fenêtre du protocole du 2026-09-01
// (`tpSensMaxMS`), sous laquelle les mèches mesurées (4,9 s à 16,2 s) tiennent toutes très
// largement. Elle remplace la demi-fenêtre fixe de ±600 ms autour de 4 930 ms, qui supposait
// la mèche connue d'avance — c'est-à-dire la variante reconnue.
const BombFuseSenseWindowMS = 120000

// bombFuseMaxCV est la dispersion maximale tolérée entre les délais corrigés d'un même film :
// la mèche est une constante de mode, ses mesures doivent s'accorder. Seuil du protocole
// (`tpCVSeuil`), et la marge est d'un ordre de grandeur : 0,016 mesuré en Neutral Bomb, 0,016
// en Husky Raid, 0,017 en One Bomb sous la lecture pausable — contre 0,725 pour la lecture
// SIMPLE que ce seuil a précisément réfutée sur One Bomb.
const bombFuseMaxCV = 0.20

// bombPairToleranceMS borne la déduplication des paires : les navpoints vont PAR PAIRES
// (+12 d'écart de slot, un par camp) et répliquent le MÊME anneau — deux fins d'armement à
// moins de cette tolérance sont UN armement. Deux armements réels ne peuvent pas être si
// proches : le hold le plus court mesuré (~0,9 s, Husky Raid) plus la mèche les séparent
// d'au moins ~6 s.
const bombPairToleranceMS = 500

// bombArmedFullQuantum est le QUANTUM PLEIN : un segment n'est un ARMEMENT que si son sommet
// l'atteint. Mesure du gate (2026-09-01, diagnostic sur les trois films du portage) : les 7
// montées confirmées par une explosion à la mèche finissent TOUTES à q=254 (7/7, Neutral et
// Husky), et l'inspection One Bomb du même jour mesure la même fin pleine (131 -> 254). Les
// segments plafonnés à q=253 sont l'ANIMATION DE RECHARGE du marqueur, jamais un armement
// (0/12) ; en dessous, des holds relâchés. Le plein est PAIR — cohérent avec la réserve
// « quantum toujours pair » du chantier ti=11. Si un armement réel finissait sous le plein
// (échantillon final perdu), il manquerait et la confrontation locale retiendrait le calque
// du film : la défaillance dégrade vers le silence, jamais vers un faux événement.
const bombArmedFullQuantum = 254

// BombInput est CE QUE L'APPELANT FOURNIT de l'armement, plus ce que `BuildFromFilm` y dépose.
type BombInput struct {
	// Scanned est la GARDE DE MODE de L'ARMEMENT, posée par l'appelant selon
	// `game_variant_name` : vraie sur TOUTE variante de la famille bomb, One Bomb COMPRISE
	// depuis le 2026-09-04 (la garde de NOM est levée, cf. l'en-tête). Faux : ni balayage,
	// ni calque, ni couverture. C'est désormais la seule chose que le NOM décide — la
	// famille, jamais la variante.
	Scanned bool
	// CarryScanned est la GARDE DE MODE du PORTAGE (`bombCarries`, schéma 30) : vraie sur
	// TOUTE variante de la famille bomb elle aussi. Les deux gardes portent maintenant le
	// même prédicat ; elles restent DEUX champs parce qu'elles arment deux balayages
	// distincts, et qu'un canal peut retomber sans l'autre.
	CarryScanned bool
	// ChunkStartMS est l'horloge du manifeste (start_ms par index de chunk), lue par
	// l'appelant au cache film du titre — la même base que les explosions du statborg.
	ChunkStartMS map[int]int
	// Reads est déposé par `BuildFromFilm` : les lectures de l'anneau. L'appelant ne le
	// remplit pas.
	Reads []filmdec.NavpointRadialRead
}

// bombFuseVerdict est ce que la confrontation locale a MESURÉ sur le film : la mèche retenue,
// sa dispersion, et si elle vient d'une mesure ou de la référence. Il ne va pas au document
// (la mèche y est publiée avec chaque événement) — il sert le journal et le gate, pour qu'un
// chiffre de production soit lisible sans réinstrumenter.
type bombFuseVerdict struct {
	FuseMS       int
	CV           float64
	Measured     bool
	Inconsistent bool
}

// decodeFilmBombReads balaye l'anneau ti=12 et JOURNALISE ce qu'il en est.
//
// IL NE BALAYE QUE LES FILMS DE LA FAMILLE BOMB, reconnus par l'appelant (même règle de coût
// que le marqueur de portage CTF) : le balayage est une marche de tous les paquets delta du
// film, et hors Assaut le calque est vide de toute façon. L'échec n'est pas fatal : le rejeu
// sort sans compte à rebours, jamais avec un compte à rebours deviné.
//
// HORS LIGNE — appelée par BuildFromFilm, sous LockProcessDecode.
func decodeFilmBombReads(filmDir, matchID string, in BombInput) []filmdec.NavpointRadialRead {
	if !in.Scanned {
		return nil
	}
	if len(in.ChunkStartMS) == 0 {
		slog.Warn("armement : film armable sans horloge de manifeste — calque non construit",
			"match_id", matchID, "filmDir", filmDir)
		return nil
	}
	sc, err := filmdec.ScanFilmNavpointRadial(filmDir, in.ChunkStartMS)
	if err != nil {
		slog.Warn("armement : anneau ti=12 illisible — rejeu sans compte a rebours",
			"err", err, "match_id", matchID, "filmDir", filmDir)
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
	armings, cov, verdict := buildBombArmings(opt.Bomb.Reads, bombDetonationTimes(doc.Objectives), c)
	doc.BombArmings = armings
	if doc.Coverage != nil {
		doc.Coverage.BombArmings = cov
	}
	logBombArmings(doc.MatchID, cov, verdict)
}

// buildBombArmings applique la chaîne mesurée : segments -> armements pleins et pauses ->
// déduplication de paire -> confrontation locale (mèche MESURÉE) -> grille de frames. Pur,
// testable sans film.
func buildBombArmings(reads []filmdec.NavpointRadialRead, detonations []int,
	c scoreClock) ([]BombArming, *BombArmingsCoverage, bombFuseVerdict) {
	cov := &BombArmingsCoverage{Scanned: true, Reads: len(reads), Detonations: len(detonations)}
	segments := filmdec.NavpointSegments(reads)
	cov.Rises = len(segments)
	full, pauses := classifyBombSegments(segments, cov)
	armed := dedupPairedSegments(full, cov)
	cov.Armed = len(armed)
	// La confrontation juge les armements NON dédupliqués — l'instrument qui a mesuré la
	// lecture pausable travaillait sur les deux miroirs de la paire, et les pauses sont
	// indexées PAR SLOT : fondre les miroirs avant de corriger perdrait celles de l'autre.
	verdict, ok := measureBombFuse(full, pauses, detonations, cov)
	if !ok {
		// LA CONFRONTATION LOCALE ÉCHOUE : ce film contredit la lecture, soit qu'une
		// explosion n'ait aucun armement dans la fenêtre de sens, soit que ses délais se
		// contredisent entre eux. Tout-ou-rien.
		cov.Suppressed = true
		return nil, cov, verdict
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
			StartT: startT, StartMS: int(r.StartMS), FuseMS: verdict.FuseMS})
	}
	cov.Published = len(out)
	return out, cov, verdict
}

// classifyBombSegments trie les segments en ARMEMENTS (finissent à leur sommet ET atteignent
// le quantum plein) et en PAUSES (tenues de désarmement, indexées par slot).
//
// L'ordre du `switch` n'est pas indifférent : un segment qui aurait les deux formes est un
// armement. C'est celui de l'instrument de mesure, gardé à l'identique.
func classifyBombSegments(segments []filmdec.NavpointSegment, cov *BombArmingsCoverage,
) ([]filmdec.NavpointSegment, map[uint32][]filmdec.NavpointSegment) {
	armed := make([]filmdec.NavpointSegment, 0, len(segments))
	pauses := map[uint32][]filmdec.NavpointSegment{}
	for _, g := range segments {
		switch {
		case g.EndsAtSummit():
			if int(g.QMax) < bombArmedFullQuantum {
				cov.BelowFull++
				continue
			}
			armed = append(armed, g)
		case g.IsDisarmHold():
			pauses[g.Slot] = append(pauses[g.Slot], g)
		}
	}
	return armed, pauses
}

// dedupPairedSegments fond les fins d'armement à moins de bombPairToleranceMS l'une de l'autre
// en UN armement : le début retenu est le plus tôt (le hold a commencé là), la fin la plus
// tardive (la statistique du protocole — « dernière fin d'armement avant l'explosion » — date
// la mèche sur le miroir le plus tardif de la paire). L'entrée arrive triée par (EndMS, Slot).
func dedupPairedSegments(armed []filmdec.NavpointSegment,
	cov *BombArmingsCoverage) []filmdec.NavpointSegment {
	var out []filmdec.NavpointSegment
	for _, r := range armed {
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

// measureBombFuse est LA GARDE 2 : elle confronte chaque explosion à l'armement qui la
// précède, MESURE la mèche du film sur les délais corrigés des pauses, et refuse le calque
// entier si une explosion reste orpheline ou si les délais se contredisent.
//
// Rend le verdict (mèche retenue, dispersion, mesurée ou déduite) et s'il tient.
func measureBombFuse(armed []filmdec.NavpointSegment, pauses map[uint32][]filmdec.NavpointSegment,
	detonations []int, cov *BombArmingsCoverage) (bombFuseVerdict, bool) {
	delays := make([]float64, 0, len(detonations))
	for _, det := range detonations {
		d, ok := bombCorrectedDelay(armed, pauses, int32(det))
		if !ok {
			continue
		}
		cov.DetonationsCovered++
		delays = append(delays, float64(d))
	}
	if cov.DetonationsCovered < cov.Detonations {
		// Le calque est retenu, mais le verdict porte quand même ce qui a été mesuré sur les
		// explosions COUVERTES : un diagnostic muet ne dit pas de combien on est passé à côté.
		// `Measured` reste faux — rien n'est publié, la valeur ne sert qu'au journal.
		if len(delays) == 0 {
			return bombFuseVerdict{}, false
		}
		med, cv := bombMedianCV(delays)
		return bombFuseVerdict{FuseMS: int(med), CV: cv}, false
	}
	if len(delays) == 0 {
		// Aucune explosion à mesurer : la mèche publiée est DÉDUITE de la référence, et
		// `cov.Detonations == 0` le dit au lecteur.
		return bombFuseVerdict{FuseMS: BombFuseMS}, true
	}
	med, cv := bombMedianCV(delays)
	v := bombFuseVerdict{FuseMS: int(med), CV: cv, Measured: true}
	// Une seule explosion ne mesure aucune dispersion : le critère porte sur les films qui
	// ont de quoi se contredire. La limite est écrite, pas contournée.
	if len(delays) >= 2 && cv > bombFuseMaxCV {
		v.Inconsistent = true
		return v, false
	}
	return v, true
}

// bombCorrectedDelay rend le délai entre la dernière fin d'armement AVANT la cible et la
// cible, DIMINUÉ des pauses du même slot strictement entre les deux — la mèche est suspendue
// pendant une tenue de désarmement. Faux si aucun armement ne précède la cible dans la
// fenêtre de sens, ou si les pauses consomment tout le délai.
func bombCorrectedDelay(armed []filmdec.NavpointSegment,
	pauses map[uint32][]filmdec.NavpointSegment, target int32) (int32, bool) {
	i := sort.Search(len(armed), func(k int) bool { return armed[k].EndMS >= target })
	if i == 0 {
		return 0, false
	}
	a := armed[i-1]
	d := target - a.EndMS
	if d <= 0 || d > BombFuseSenseWindowMS {
		return 0, false
	}
	for _, p := range pauses[a.Slot] {
		if p.StartMS > a.EndMS && p.EndMS < target {
			d -= p.EndMS - p.StartMS
		}
	}
	if d <= 0 {
		return 0, false
	}
	return d, true
}

// bombMedianCV rend la médiane et le coefficient de variation (écart-type sur médiane) d'une
// série — la statistique EXACTE du protocole du 2026-09-01 (`tpMedCV`), pour que le seuil
// livré juge la même grandeur que celui qui a été mesuré.
func bombMedianCV(xs []float64) (float64, float64) {
	if len(xs) == 0 {
		return 0, math.Inf(1)
	}
	tri := append([]float64(nil), xs...)
	sort.Float64s(tri)
	med := tri[len(tri)/2]
	if med == 0 {
		return med, math.Inf(1)
	}
	var s float64
	for _, x := range xs {
		s += (x - med) * (x - med)
	}
	return med, math.Sqrt(s/float64(len(xs))) / med
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
func logBombArmings(matchID string, cov *BombArmingsCoverage, v bombFuseVerdict) {
	if cov.Suppressed {
		slog.Warn("rejeu : armement RETENU A LA SOURCE — le film contredit la lecture "+
			"(explosion sans armement, ou meches qui se contredisent)",
			"match_id", matchID, "explosions", cov.Detonations,
			"couvertes", cov.DetonationsCovered, "armements", cov.Armed, "segments", cov.Rises,
			"mecheIncoherente", v.Inconsistent, "meche_ms", v.FuseMS, "cv", v.CV)
		return
	}
	slog.Info("rejeu : armement de la bombe",
		"match_id", matchID, "lectures", cov.Reads, "segments", cov.Rises,
		"sousLePlein", cov.BelowFull, "armements", cov.Armed, "paireFondue", cov.PairMerged,
		"publies", cov.Published, "horsFenetre", cov.OutOfWindow,
		"explosions", cov.Detonations, "couvertes", cov.DetonationsCovered,
		"meche_ms", v.FuseMS, "mecheMesuree", v.Measured, "cv", v.CV)
}
