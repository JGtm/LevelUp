package filmdec

// equipment_creation_width.go — LA LARGEUR du premier champ du bloc `object-multiplayer-
// properties`, MESURÉE dans le film au lieu d'être devinée.
//
// LE PROBLÈME, ET IL EST MESURÉ. Le default-state de l'archétype d'équipement (ti=37) ne fait
// pas la même largeur d'un film à l'autre : 60 bits sur les films d'arène, 57 sur d'autres
// (2026-08-17, PLAN_IDENTITE_TI37 phase 2). L'écart de 3 bits est porté par le PREMIER champ du
// bloc MPP (FUN_141fd72c0, R(9) au décompile) — une largeur de configuration de RÉPLICATION, du
// même genre que les largeurs d'axe du chemin world-object : posée au chargement de la carte,
// absente de l'exécutable. Balayer à la mauvaise largeur ne rend pas une mesure fausse, il rend
// du BRUIT : le curseur est décalé, le masque de présence ne se lit plus, et les quelques
// records qui survivent portent des identifiants tous différents.
//
// L'ORACLE, ET POURQUOI C'EN EST UN. La première détection (2026-08-17) retenait la largeur qui
// maximisait le nombre de records acceptés, en exigeant que leurs identifiants se répètent.
// C'est un critère NÉGATIF (« ça ne se casse pas »), et il a échoué sur les gros films faute de
// records dans la fenêtre. Celui-ci est POSITIF et indépendant du décodage du record : les vies
// d'objet ti=37 sont déjà décodées par les paquets delta (ScanFilmWorldObjects), donc le premier
// point de chaque vie (slot, génération) est CONNU. Un en-tête NEW annonce sa vie ; à la bonne
// largeur, la position lue à la fin du record retombe sur ce premier point, à quelques quanta
// près. Cette coïncidence-là ne s'obtient pas par hasard — trois axes quantifiés, quelques
// centaines de cibles dans un espace de 2^45 — et elle ne dépend d'AUCUNE hypothèse sur la
// grammaire du bloc MPP.
//
// L'ARRÊT EST ANTICIPÉ, et c'est ce qui règle le cas des gros films : on lit autant de chunks
// qu'il en faut pour que le verdict soit net (une largeur nettement devant les autres), au lieu
// de borner la fenêtre à l'aveugle et de ne rien conclure quand elle est trop pauvre.
//
// HORS LIGNE (I/O disque) — jamais depuis un chemin de requête. L'APPELANT DOIT DÉTENIR
// LockProcessDecode : la calibration écrit le global `mppLeadBits` ; elle le restaure à la
// sortie, et c'est à l'appelant d'installer la largeur retenue pour son propre décodage.

import (
	"fmt"
	"sort"

	"levelup/go-api/internal/analysis/filmsource"
)

// EquipmentLifeKey identifie une vie d'objet du monde : LA PAIRE (slot, génération), jamais le
// slot seul — le pool de slots reboucle et la génération ne fait que 2 bits.
type EquipmentLifeKey struct{ Slot, Gen uint32 }

// EquipmentLifeSpan est UNE vie d'objet telle que les paquets delta l'ont décodée : ses bornes
// temporelles et son premier point. C'est à la fois l'ORACLE de la calibration et la matière de
// la pose publiée.
//
// T1US EST UNE MISE AU REPOS, PAS UNE DISPARITION (mesuré le 2026-08-18, cf. splitLives) : le
// décodage ne suit que les records qui portent une position, et un objet posé cesse d'en
// émettre dès qu'il s'immobilise. Le film ne date PAS la disparition d'un objet d'équipement —
// ni par un record de suppression isolable, ni par une queue de records sans position (les deux
// témoins sont du bruit, `equipment_life_end_test.go`).
type EquipmentLifeSpan struct {
	T0US, T1US uint64
	First      [3]float32
	Points     int
}

// EquipmentLifeSpans indexe les vies d'objet par clé.
//
// PLUSIEURS VIES PAR CLÉ, ET C'EST LA DONNÉE : la génération ne fait que 2 bits, donc un slot
// repasse par la même paire (slot, génération) au cours d'un match — d'autant plus souvent que
// le match est long et l'objet volatil. Ne garder que la première (ce que faisait la version
// initiale de cet oracle) perdait la moitié des poses sur les films BTB : un record de création
// tardif ne retombait sur AUCUN premier point connu. On garde donc toutes les vies de la clé,
// triées par instant, et c'est la POSITION qui départage.
func EquipmentLifeSpans(tracks []ProjectileTrack) map[EquipmentLifeKey][]EquipmentLifeSpan {
	out := make(map[EquipmentLifeKey][]EquipmentLifeSpan, len(tracks))
	for _, tr := range tracks {
		if len(tr.Pts) == 0 {
			continue
		}
		k := EquipmentLifeKey{tr.Slot, tr.Gen}
		p := tr.Pts[0]
		out[k] = append(out[k], EquipmentLifeSpan{
			T0US:   p.TimestampUS,
			T1US:   tr.Pts[len(tr.Pts)-1].TimestampUS,
			First:  [3]float32{p.X, p.Y, p.Z},
			Points: len(tr.Pts),
		})
	}
	for k := range out {
		sort.Slice(out[k], func(i, j int) bool { return out[k][i].T0US < out[k][j].T0US })
	}
	return out
}

// MatchEquipmentLife rend la vie de la clé dont le premier point coïncide avec `pos`, à `eps`
// près sur chaque axe. C'est LE contrôle qui sépare une création réelle d'un faux positif du
// balayage bit à bit : l'en-tête NEW n'est pas assez sélectif à lui seul (mesuré le 2026-08-17 —
// sur un film BTB, 20 657 ancres pour ~20 créations réelles), mais la coïncidence de trois axes
// quantifiés avec le premier point de la vie que l'en-tête ANNONCE ne s'obtient pas par hasard.
//
// À égalité, la vie la plus proche en temps du record l'emporte : deux vies de la même clé au
// même endroit (un objet qui réapparaît à son socle) ne se départagent que par l'horloge.
func MatchEquipmentLife(
	spans []EquipmentLifeSpan, pos [3]float32, eps [3]float32, atUS uint64,
) (EquipmentLifeSpan, bool) {
	var best EquipmentLifeSpan
	found := false
	for _, s := range spans {
		if !equipPosNear(s.First, pos, eps) {
			continue
		}
		if !found || equipTimeGap(s.T0US, atUS) < equipTimeGap(best.T0US, atUS) {
			best, found = s, true
		}
	}
	return best, found
}

func equipPosNear(a, b [3]float32, eps [3]float32) bool {
	return absF32(a[0]-b[0]) <= eps[0] &&
		absF32(a[1]-b[1]) <= eps[1] &&
		absF32(a[2]-b[2]) <= eps[2]
}

func equipTimeGap(a, b uint64) uint64 {
	if a > b {
		return a - b
	}
	return b - a
}

func absF32(f float32) float32 {
	if f < 0 {
		return -f
	}
	return f
}

// EquipmentPosEps rend le rayon d'accord de l'oracle sur chaque axe, pour les bornes données.
func EquipmentPosEps(wr *Vec3Range) [3]float32 {
	return [3]float32{
		mppCalibPosEps * (wr[0].Max - wr[0].Min),
		mppCalibPosEps * (wr[1].Max - wr[1].Min),
		mppCalibPosEps * (wr[2].Max - wr[2].Min),
	}
}

// mppLeadCandidates / mppIndexCandidates bornent la calibration. Le décompile donne 9 et 5 ; la
// mesure du 2026-08-17 observe un default-state 3 bits plus court sur certains films, et ces deux
// champs sont les SEULS champs inconditionnels du chemin minimal dont la largeur puisse
// l'expliquer (l'identifiant de 32 bits ne bouge pas, le compte de 3 bits non plus). Les plages
// encadrent le décompile avec de la marge, sans jamais permettre une largeur absurde. L'ordre
// place les valeurs du décompile en tête : à égalité d'accords, c'est la première qui gagne.
var (
	mppLeadCandidates  = []int{9, 6, 5, 7, 8, 10, 11, 12, 13}
	mppIndexCandidates = []int{5, 2, 3, 4, 6, 7, 8}
)

// mppCandidates énumère les découpages testés, dans l'ordre de préférence.
func mppCandidates() []MPPWidths {
	out := make([]MPPWidths, 0, len(mppLeadCandidates)*len(mppIndexCandidates))
	for _, idx := range mppIndexCandidates {
		for _, lead := range mppLeadCandidates {
			out = append(out, MPPWidths{Lead: lead, Index: idx})
		}
	}
	return out
}

const (
	// mppCalibPosEps est le rayon d'accord, en FRACTION de l'étendue de chaque axe. La position
	// du record de création n'est pas identique au premier point transmis en delta (écart médian
	// de quelques quanta) : une égalité stricte ne retrouverait presque aucun record. 0,002 de
	// l'emprise vaut une poignée de quanta, et reste mille fois plus petit que la carte.
	mppCalibPosEps = 0.002
	// mppCalibMinAgree est le nombre d'accords en deçà duquel on ne conclut pas. Douze positions
	// qui retombent sur la bonne vie ne sont pas un hasard, mais moins ne se distingue pas d'une
	// coïncidence sur un film pauvre en poses.
	mppCalibMinAgree = 12
	// mppCalibMargin est le facteur qui sépare la largeur retenue de sa concurrente. Une largeur
	// juste ne devance pas l'autre de peu : elle l'écrase. Exiger un facteur 3 est ce qui
	// interdit de trancher sur du bruit.
	mppCalibMargin = 3
)

// MPPCalibration porte ce que la calibration a mesuré. Sans ces dénominateurs, une largeur
// retenue n'est qu'une affirmation : c'est le nombre d'accords ET l'écart à la concurrente qui
// disent si le film a tranché ou non.
type MPPCalibration struct {
	// Widths est le découpage retenu ; non Valid() si la calibration n'a pas conclu.
	Widths MPPWidths
	// Agree est le nombre de records dont la position retombe sur le premier point de la vie
	// que leur en-tête annonce, au découpage retenu.
	Agree int
	// Runner / RunnerAgree : le meilleur découpage CONCURRENT et son score. C'est l'écart entre
	// les deux qui fait la preuve, pas le score absolu.
	Runner      MPPWidths
	RunnerAgree int
	// Anchors est le nombre d'en-têtes NEW ti=37 dont la vie est connue des paquets delta.
	Anchors int
	// Chunks est le nombre de chunks lus avant l'arrêt (anticipé ou en fin de film).
	Chunks int
	// Lives est le nombre de vies d'objet que l'oracle connaissait.
	Lives int
	// ByWidths est le score de CHAQUE découpage candidat : la pièce justificative.
	ByWidths map[MPPWidths]int
}

// String rend la calibration en une ligne, pour le journal de production.
func (c MPPCalibration) String() string {
	if !c.Widths.Valid() {
		return fmt.Sprintf("NON CONCLUANTE (%d ancres, %d vies, %d chunks)",
			c.Anchors, c.Lives, c.Chunks)
	}
	return fmt.Sprintf("%s (%d accords contre %d à %s ; %d ancres, %d vies, %d chunks)",
		c.Widths, c.Agree, c.RunnerAgree, c.Runner, c.Anchors, c.Lives, c.Chunks)
}

// CalibrateMPPWidths mesure le découpage des deux champs de largeur variable du bloc MPP sur le
// film de dir, en confrontant chaque découpage candidat à l'oracle de position.
//
// POURQUOI DEUX CHAMPS ET NON UN. Rétrécir le premier champ du bloc ou celui qui suit
// l'identifiant donne le même TOTAL, donc la même position — l'oracle seul ne les sépare pas sur
// les records du chemin minimal. Il les sépare sur TOUS LES AUTRES : un découpage faux lit les
// portes du bloc à des bits décalés, prend des branches qui n'existent pas, et perd les records
// dont une porte est ouverte. C'est le NOMBRE d'accords qui tranche, et l'écart est massif.
//
// Rend (calibration, true) quand un découpage devance nettement les autres ; (calibration, false)
// sinon — et dans ce cas l'appelant NE PUBLIE PAS, il le dit. Un défaut silencieux servirait des
// poses inventées, et un identifiant lu 3 bits trop tôt est une invention.
// ENVELOPPE D2, HORS PRODUCTION ; la cuisson appelle [CalibrateMPPWidthsOf].
func CalibrateMPPWidths(
	dir string, wr *Vec3Range, band map[uint32]bool, spans map[EquipmentLifeKey][]EquipmentLifeSpan,
) (MPPCalibration, bool) {
	film, err := filmsource.LoadDir(dir, nil)
	if err != nil {
		return MPPCalibration{ByWidths: map[MPPWidths]int{}, Lives: len(spans)}, false
	}
	return CalibrateMPPWidthsOf(NewFilmContext(film), wr, band, spans)
}

// CalibrateMPPWidthsOf mesure le découpage du bloc MPP sur un film DEJA CHARGE.
func CalibrateMPPWidthsOf(
	fc *FilmContext, wr *Vec3Range, band map[uint32]bool,
	spans map[EquipmentLifeKey][]EquipmentLifeSpan,
) (MPPCalibration, bool) {
	cal := MPPCalibration{ByWidths: map[MPPWidths]int{}, Lives: len(spans)}
	if wr == nil || len(band) == 0 || len(spans) == 0 {
		return cal, false
	}
	arch, err := fc.EquipmentArchetype()
	if err != nil {
		return cal, false
	}
	nums := fc.ChunkNumbers()
	if len(nums) == 0 {
		return cal, false
	}
	defer SetMPPWidths(CurrentMPPWidths())
	var cur equipCreationRead
	defer installCreationHooks(&cur)()

	pr := mppCalibProbe{
		walk:  equipCreationWalk{comps: len(arch.Components), wr: wr, band: band, cur: &cur},
		spans: spans,
		eps:   EquipmentPosEps(wr),
		cal:   &cal,
	}
	for _, c := range nums {
		data, pks, ok := fc.ChunkAt(c)
		if !ok {
			continue
		}
		cal.Chunks++
		for _, pk := range pks {
			if pk.Type == PacketTypeDelta {
				pr.scanPayload(pk.Payload(data), pk.TimestampUS)
			}
		}
		if _, ok := mppCalibVerdict(cal.ByWidths); ok {
			break // le verdict est net : lire plus de chunks ne l'affinerait pas
		}
	}
	best, ok := mppCalibVerdict(cal.ByWidths)
	cal.Widths, cal.Agree = best.widths, best.agree
	cal.Runner, cal.RunnerAgree = best.runner, best.runnerAgree
	if !ok {
		cal.Widths = MPPWidths{}
	}
	return cal, ok
}

// mppCalibProbe porte ce que la marche d'un payload doit connaître (règle des 5 paramètres).
type mppCalibProbe struct {
	walk  equipCreationWalk
	spans map[EquipmentLifeKey][]EquipmentLifeSpan
	eps   [3]float32
	cal   *MPPCalibration
}

// scanPayload compte, pour CHAQUE découpage candidat, les records dont la position retombe sur
// le premier point de la vie que leur en-tête annonce.
//
// L'ANCRE EST CHERCHÉE UNE SEULE FOIS pour tous les découpages : l'en-tête NEW ne dépend pas du
// bloc MPP, qui vit derrière lui. Soixante-trois balayages complets du film coûteraient
// soixante-trois fois le prix de celui-ci pour la même information.
func (pr *mppCalibProbe) scanPayload(pay []byte, atUS uint64) {
	var st EquipmentCreationStats
	total := len(pay) * 8
	limit := total - woNewHeaderBits
	cands := mppCandidates()
	for p := 0; p <= limit; p++ {
		slot, gen, ok := matchEquipmentNewHeader(pay, p, pr.walk.band)
		if !ok {
			continue
		}
		spans := pr.spans[EquipmentLifeKey{slot, gen}]
		if len(spans) == 0 {
			continue // une vie que les paquets delta n'ont pas vue ne peut rien arbitrer
		}
		pr.cal.Anchors++
		for _, w := range cands {
			SetMPPWidths(w)
			cre, ok := pr.walk.readCreation(pay, p, total, &st)
			if !ok {
				continue
			}
			if _, hit := MatchEquipmentLife(spans, [3]float32{cre.X, cre.Y, cre.Z}, pr.eps, atUS); hit {
				pr.cal.ByWidths[w]++
			}
		}
	}
}

// mppCalibResult est le classement des découpages à un instant du balayage.
type mppCalibResult struct {
	widths      MPPWidths
	agree       int
	runner      MPPWidths
	runnerAgree int
}

// mppCalibVerdict classe les découpages et dit si l'écart tranche. Les deux conditions sont
// cumulatives et énoncées avant la mesure : assez d'accords ET une avance qui écrase.
func mppCalibVerdict(byWidths map[MPPWidths]int) (mppCalibResult, bool) {
	var r mppCalibResult
	for _, w := range mppCandidates() {
		n := byWidths[w]
		switch {
		case n > r.agree:
			r.runner, r.runnerAgree = r.widths, r.agree
			r.widths, r.agree = w, n
		case n > r.runnerAgree:
			r.runner, r.runnerAgree = w, n
		}
	}
	return r, r.agree >= mppCalibMinAgree && r.agree >= mppCalibMargin*r.runnerAgree
}
