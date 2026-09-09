package replay

import (
	"sort"

	"levelup/go-api/internal/analysis/filmdec"
)

// equipment_episodes.go — L'ÉTAT ACTIF D'UN ÉQUIPEMENT, daté PAR VIE sur l'axe du rejeu.
//
// DEUX FAMILLES, DEUX CANAUX MESURÉS (2026-08-16, PLAN_ETAT_ACTIF_EQUIPEMENT, gates A et
// B passés) — et rien d'autre : les déployables ne se datent pas par les canaux mesurés,
// la mobilité n'a pas d'instant d'usage par i54. On publie ce qui est mesuré, l'absence
// reste une absence.
//
//	camo        i28 queue[1], INTERRUPTEUR binaire 0/4095 (cf. filmdec/camo_state.go).
//	            L'activation se date au passage à 4095, la désactivation au retour à 0.
//	overshield  i5 NON clampé, règle `q > 64` (0 faux positif sur ~150 000 mesures hors
//	            porteurs). Le quantum est capturé dans le MÊME record que la position
//	            (ScanFilmBipedPositions, CaptureDirs) : aucune relecture à côté du déser.
//
// POURQUOI UN CANAL SÉPARÉ ET PAS `Point.sh` : `ShieldFraction` CLAMPE à 1.0 avant la
// sérialisation — l'artefact ne peut pas discriminer un surbouclier d'un bouclier plein,
// et changer l'échelle de `sh` changerait toutes les jauges de vitalité du rejeu. Les
// épisodes sont donc un calque à part, à l'échelle près de rien d'autre.
//
// LA FIN D'UN ÉPISODE EST SOIT MESURÉE, SOIT LA MORT. Un épisode encore ouvert à la fin
// de la vie se ferme à la fin de la piste (le fil des morts date la fin de vie — c'est le
// même instant que le flash de mort des fiches). Le champ EndRead dit lequel des deux :
// un client qui sonne la désactivation ne doit la sonner QUE sur une fin mesurée.
//
// LES DEUX BORNES SONT « À LA PRÉCISION DE LA RETRANSMISSION PRÈS » : le film ne
// retransmet un canal que lorsqu'il change (i28) ou que le record le porte (i5). Pour le
// surbouclier, la mesure dit de surcroît que l'épisode commence à la PREMIÈRE mesure de
// bouclier de la vie qui dépasse — la datation fine du ramassage n'est pas établie.

// Familles d'équipement publiées. Identifiants STABLES, jamais des libellés (même règle
// que NeutralDeath.Kind) : les libellés vivent côté client, dans les deux langues.
const (
	// EquipFamilyCamo : camouflage actif (rang 8 de la palette famille A).
	EquipFamilyCamo = "camo"
	// EquipFamilyOvershield : surbouclier (rang 9 de la palette famille A).
	EquipFamilyOvershield = "overshield"
)

// EquipmentEpisode est un épisode daté d'état ACTIF d'un équipement, porté par une vie.
type EquipmentEpisode struct {
	// Slot désigne la Track concernée — donc une VIE, pas un joueur (même règle que les
	// autres calques : le slot migre aux réapparitions).
	Slot uint32 `json:"slot"`
	// Fam est la famille d'équipement (EquipFamilyCamo / EquipFamilyOvershield). Une
	// famille inconnue du client ne reçoit AUCUN effet ni son — jamais ceux d'une voisine.
	Fam string `json:"fam"`
	// T0 / T1 bornent l'épisode sur le même axe que Point.T. T1 est soit l'instant de la
	// transition mesurée de fin, soit la fin de la vie (cf. EndRead).
	T0 int `json:"t0"`
	T1 int `json:"t1"`
	// EndRead dit si la fin est une transition MESURÉE (retour à 0 pour le camo, retour
	// sous le plein pour le surbouclier) ou la fermeture à la fin de la vie. Publié parce
	// qu'un consommateur (le son) doit distinguer « l'équipement s'éteint » de « le
	// porteur meurt » — le second n'a pas de son de désactivation mesuré.
	EndRead bool `json:"endRead,omitempty"`
	// K / A : les frags et assistances DU PORTEUR pendant [T0, T1] (cf.
	// equipment_episode_kills.go). Omis quand nuls — le zéro DE CE CHAMP n'est distinguable
	// d'une mesure non tentée QUE via EquipmentCoverage.KillsRead, publié à côté : lui seul
	// dit si la jointure a eu lieu pour ce match.
	K int `json:"k,omitempty"`
	A int `json:"a,omitempty"`
}

// EquipmentCoverage dit combien de vies publiées portent au moins un épisode, par
// famille — le dénominateur sans lequel « N épisodes » ne se juge pas. Une couverture
// partielle est un résultat, pas un échec : la plupart des vies ne portent NI camouflage
// NI surbouclier, et zéro épisode sur un film sans porteur est la valeur juste.
type EquipmentCoverage struct {
	// TracksTotal est le nombre de vies PUBLIÉES du document (les seules où un épisode
	// peut s'afficher).
	TracksTotal int `json:"tracksTotal"`
	// CamoLives / CamoEpisodes : vies portant au moins un épisode de camouflage, et le
	// compte total d'épisodes.
	CamoLives    int `json:"camoLives"`
	CamoEpisodes int `json:"camoEpisodes"`
	// OvershieldLives / OvershieldEpisodes : idem pour le surbouclier.
	OvershieldLives    int `json:"overshieldLives"`
	OvershieldEpisodes int `json:"overshieldEpisodes"`
	// KillsRead dit si `EquipmentEpisode.K`/`.A` ont été MESURÉS pour ce match — jamais
	// « il n'y avait rien à joindre ». Faux quand killsource n'a pas pu être décodé, quand
	// sa porte de publication ligne-par-ligne était fermée, ou quand l'origine d'horloge du
	// document n'est pas établie (cf. equipment_episode_kills.go). SANS CE CHAMP, un match
	// sans aucun frag sous effet actif (K=0 partout, mesuré) est indiscernable d'un match où
	// la mesure a simplement échoué — exactement le piège que Coverage existe pour fermer.
	KillsRead bool `json:"killsRead"`
}

// trackFrameWindows indexe les fenêtres [StartFrame, EndFrame] des vies publiées. Ce
// n'est PAS l'ensemble booléen des slots publiés (published_tracks.go reste son seul
// constructeur) : la fenêtre sert à FERMER un épisode à la mort et à borner ses frames.
//
// UN SLOT PORTE TOUTES SES VIES, PAS SEULEMENT LA DERNIÈRE. Depuis le schéma 36 (« une
// track = une vie ») un slot recyclé publie plusieurs pistes ; n'en garder qu'une bornait
// les épisodes des vies antérieures hors de leur fenêtre, et `close` les jetait
// (`t1 < t0`). Mesure du balayage du parc : `82f29378` perdait son UNIQUE épisode de
// surbouclier — il est revenu —, `084a804d` 2 épisodes de camouflage sur 21.
// Correctif du 2026-09-06.
//
// `13d92593` PERDAIT AUSSI SON ÉPISODE DE SURBOUCLIER, ET IL NE REVIENT PAS : ce n'est pas
// le même défaut. Il durait zéro image (t0 = t1 = 3603) et s'ancrait sur le seul point de
// trajectoire qui plaçait le joueur à 267 u de sa position précédente — celui-là même qui
// donnait au document des bornes de scène fausses, et que l'assainissement a supprimé.
// Cuisson de contrôle : 0 épisode, document identique à la base hors `schemaVersion`.
// (Constat C4 de la revue REG-R1 : ce commentaire le citait parmi les films restitués.)
//
// LES FENETRES SONT TRIEES ET PORTENT L'IDENTITE DE LEUR VIE (revue DUREES-R1, constat C2) :
// `spanFor` a besoin de savoir ce que SEPARE un trou entre deux vies — un silence de
// replication, ou une mort.
//
// LA BORNE EST LA MORT, ET C'EST LE REGISTRE QUI LA DIT (correctif E2-bis, 2026-09-08).
// `closedByDeath` porte les indices des pistes dont la vie se termine par une mort LUE
// (`IdentityRegistry.TracesCloturesParMort`, cause `CauseVieMort`). Ce fichier lisait cette
// frontiere dans « la vie porte un nom » (`XUID != ""`, deduction retiree) : proxy exact tant que
// le fil des morts etait la seule voie de nommage — il posait le xuid de la victime sur la vie
// que sa mort acheve. Depuis le lot E2, le FILM nomme les vies a leur CREATION ; toutes portent
// un nom, aucune n'est plus « deduite », et le proxy declarait une mort a CHAQUE trou de
// replication. Mesure : `084a804d` slot 620, camo `[3105..3672]` (568 frames, la valeur meme que
// la chronique du v45 publie) retombe a `[3105..3120]`, 16 frames, soit 552 perdues sur le
// TEMOIN de ce bornage.
func trackFrameWindows(tracks []Track, closedByDeath map[int]bool) map[uint32][]lifeWindow {
	out := make(map[uint32][]lifeWindow, len(tracks))
	for i, t := range tracks {
		out[t.Slot] = append(out[t.Slot], lifeWindow{
			from: t.StartFrame, to: t.EndFrame, named: closedByDeath[i]})
	}
	for s := range out {
		w := out[s]
		sort.Slice(w, func(i, j int) bool { return w[i].from < w[j].from })
	}
	return out
}

// lifeWindow est la fenetre d UNE vie publiee, plus ce que sa FIN signifie.
type lifeWindow struct {
	from, to int
	// named dit que la vie se termine par une MORT LUE — la seule frontiere que `spanFor` ne
	// franchit pas. Une vie fermee par un TROU DE REPLICATION (`CauseVieCoupure`) ou par la fin
	// du film ne borne rien : rien n y meurt, et c est exactement la couture que `spanFor` existe
	// pour refaire.
	//
	// LA LECTURE EST CONSERVATRICE, ET C EST VOULU : un episode trop court est une mesure
	// incomplete ; un episode qui enjambe une mort est une mesure FAUSSE, peinte sur une vie ou
	// rien ne l a lue.
	named bool
}

// windowFor rend la vie du slot qui recouvre le plus l'intervalle d'un épisode. ok=false
// quand aucune ne l'intersecte : l'épisode n'a alors aucune fiche où s'afficher.
//
// DEUX APPELANTS, ET C'EST TOUT : `finish` (la vie qui contient l'OUVERTURE, celle dont la fin
// date la mort) et `equipmentCoverage` (la clé de vie du dénominateur). Le BORNAGE de l'épisode,
// lui, passe par `spanFor` — cf. sa raison ci-dessous.
//
// LE RECOUVREMENT PLUTÔT QUE L'APPARTENANCE : une lecture peut tomber dans un trou de
// réplication (les vies d'un slot ne se touchent pas), et exiger que `from` soit DANS une
// fenêtre y perdrait l'épisode. Le recouvrement maximal ne dépend d'aucun ordre.
func windowFor(windows []lifeWindow, from, to int) (lifeWindow, bool) {
	best, bestOv, found := lifeWindow{}, 0, false
	for _, w := range windows {
		lo, hi := from, to
		if lo < w.from {
			lo = w.from
		}
		if hi > w.to {
			hi = w.to
		}
		if ov := hi - lo + 1; ov > 0 && (!found || ov > bestOv) {
			best, bestOv, found = w, ov, true
		}
	}
	return best, found
}

// spanFor rend les bornes de l'UNION des vies du slot que l'intervalle mesuré recouvre — mais
// SANS JAMAIS FRANCHIR UNE MORT. ok=false quand aucune vie ne l'intersecte : l'épisode n'a alors
// aucune fiche où s'afficher, et il est écarté comme avant.
//
// POURQUOI L'UNION, ET PAS LA VIE QUI RECOUVRE LE PLUS. Un état actif se mesure PAR SLOT (i28
// pour le camo, i5 pour le surbouclier) : ses deux bornes sont des transitions LUES, elles ne
// savent rien du découpage en vies. Depuis le schéma 36 (« une track = une vie ») un trou de
// réplication de plus de `lifeGapUS` coupe une piste en deux — et un état actif est
// précisément ce qui PROVOQUE ce trou pour le camouflage : un porteur invisible et immobile
// cesse d'être répliqué. Borner à la seule vie de recouvrement maximal jetait alors la part de
// l'épisode couverte par l'autre vie, DONT SON INSTANT D'ACTIVATION MESURÉ. Mesure du corpus
// témoin : `084a804d`, slot 620, camo [3105..3672] lu (568 frames), publié [3173..3672] (500) —
// 68 frames perdues dont 16 à l'intérieur d'une vie publiée, l'activation sonnée 6,8 s en
// retard.
//
// POURQUOI LA MORT ARRÊTE L'UNION (revue DUREES-R1, constat C2). L'union recoud un SILENCE, pas
// une vie. Sans cette borne, trois vies NOMMÉES d'un même slot avec une activation dans la
// première rendaient un épisode `[20..450]` qui enjambait deux morts — `equipmentFx.ts` aurait
// peint l'effet sur des vies où rien ne l'a jamais lu. La couture ne traverse donc que les
// frontières qu'AUCUNE identité ne date (cf. `lifeWindow.named`), et s'arrête à la première fin
// de vie nommée rencontrée depuis l'ancre. La règle vaut dans les deux sens ; en pratique le
// clamp de `close` ne peut que RÉTRÉCIR l'intervalle mesuré, donc seule l'extension vers l'avant
// change quelque chose.
//
// L'ANCRE est la vie qui CONTIENT l'ouverture ; à défaut (activation antérieure à la première
// vie publiée, cf. `frameOf` en signé) la première vie recouverte.
func spanFor(windows []lifeWindow, from, to int) (lifeWindow, bool) {
	couvertes := overlapping(windows, from, to)
	if len(couvertes) == 0 {
		return lifeWindow{}, false
	}
	ancre := 0
	for i, w := range couvertes {
		if from >= w.from && from <= w.to {
			ancre = i
			break
		}
	}
	lo, hi := ancre, ancre
	for lo > 0 && !couvertes[lo-1].named {
		lo--
	}
	for hi < len(couvertes)-1 && !couvertes[hi].named {
		hi++
	}
	return lifeWindow{from: couvertes[lo].from, to: couvertes[hi].to, named: couvertes[hi].named}, true
}

// overlapping rend, dans l'ordre chronologique, les vies du slot que `[from, to]` recouvre.
// L'ordre vient de `trackFrameWindows`, qui trie : la contiguïté testée par `spanFor` est donc
// celle du temps, jamais celle de l'itération d'une map.
func overlapping(windows []lifeWindow, from, to int) []lifeWindow {
	out := make([]lifeWindow, 0, len(windows))
	for _, w := range windows {
		lo, hi := from, to
		if lo < w.from {
			lo = w.from
		}
		if hi > w.to {
			hi = w.to
		}
		if hi < lo {
			continue // cette vie ne recouvre pas l'intervalle
		}
		out = append(out, w)
	}
	return out
}

// episodeAccum accumule les épisodes d'UNE famille pour UN slot : machine à deux états
// (ouvert / fermé), fermeture à la fin de vie si rien ne l'a mesurée.
type episodeAccum struct {
	slot uint32
	fam  string
	// windows porte TOUTES les vies publiées du slot : l'épisode est borné à celle qu'il
	// recouvre, jamais à la dernière du slot (cf. trackFrameWindows).
	windows   []lifeWindow
	openFrame int
	open      bool
	out       *[]EquipmentEpisode
}

// sample avance la machine : active ouvre (au premier instant actif), inactive ferme
// (transition mesurée). Une lecture qui ne change pas l'état ne produit rien.
func (a *episodeAccum) sample(frame int, active bool) {
	switch {
	case active && !a.open:
		a.open, a.openFrame = true, frame
	case !active && a.open:
		a.close(frame, true)
	}
}

// close émet l'épisode borné aux vies publiées du slot qu'il recouvre (leur UNION, cf.
// spanFor). Un épisode entièrement hors des fenêtres publiées est écarté : il n'a aucune fiche
// où s'afficher.
func (a *episodeAccum) close(endFrame int, endRead bool) {
	a.open = false
	w, ok := spanFor(a.windows, a.openFrame, endFrame)
	if !ok {
		return // aucune vie publiée ne recouvre l'épisode
	}
	t0, t1 := a.openFrame, endFrame
	if t0 < w.from {
		t0 = w.from
	}
	if t1 > w.to {
		t1 = w.to
	}
	if t1 < t0 {
		return
	}
	*a.out = append(*a.out, EquipmentEpisode{Slot: a.slot, Fam: a.fam, T0: t0, T1: t1, EndRead: endRead})
}

// finish ferme un épisode resté ouvert À LA FIN DE LA VIE : la fin de piste date la mort
// (EndRead=false — rien n'a mesuré une désactivation). La vie est celle qui contient
// l'ouverture, pas la dernière du slot.
func (a *episodeAccum) finish() {
	if !a.open {
		return
	}
	if w, ok := windowFor(a.windows, a.openFrame, a.openFrame); ok {
		a.close(w.to, false)
		return
	}
	a.open = false
}

// frameOf projette un horodatage film sur la grille, en SIGNÉ (division plancher) : une
// lecture antérieure à l'origine rend une frame négative, que le clamp de fenêtre ramène
// au début de la vie — l'état était déjà actif quand le rejeu commence, on ne l'invente
// pas plus tard.
func frameOf(ts, origin, step uint64) int {
	if ts >= origin {
		return int((ts - origin) / step)
	}
	return -int((origin - ts + step - 1) / step)
}

// buildEquipmentEpisodes assemble les épisodes des deux familles. Rend aussi le compte
// des lectures camo NON BINAIRES (ni 0 ni 4095, jamais observées sur le corpus mesuré) :
// elles ne changent pas l'état — on n'interprète pas un troisième niveau d'un
// interrupteur — mais elles se COMPTENT, pour que leur apparition se voie au journal.
func buildEquipmentEpisodes(
	sorted []filmdec.BipedPosition, camo []filmdec.CamoRead, origin, step uint64, tracks []Track,
	closedByDeath map[int]bool,
) ([]EquipmentEpisode, int) {
	if len(tracks) == 0 || step == 0 {
		return nil, 0
	}
	windows := trackFrameWindows(tracks, closedByDeath)
	var out []EquipmentEpisode
	nonBinary := buildCamoEpisodes(camo, origin, step, windows, &out)
	buildOvershieldEpisodes(sorted, origin, step, windows, &out)
	if len(out) == 0 {
		return nil, nonBinary
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].T0 != out[j].T0 {
			return out[i].T0 < out[j].T0
		}
		if out[i].Slot != out[j].Slot {
			return out[i].Slot < out[j].Slot
		}
		return out[i].Fam < out[j].Fam
	})
	return out, nonBinary
}

// buildCamoEpisodes déroule l'interrupteur i28 queue[1] par vie. Les lectures sont
// regroupées par slot puis rejouées en ordre de temps — l'ordre du balayage suit déjà les
// chunks, le tri est là pour que la machine ne dépende pas d'un ordre d'itération.
func buildCamoEpisodes(
	camo []filmdec.CamoRead, origin, step uint64, windows map[uint32][]lifeWindow, out *[]EquipmentEpisode,
) int {
	bySlot := map[uint32][]filmdec.CamoRead{}
	for _, r := range camo {
		if _, ok := windows[r.Slot]; !ok {
			continue // vie non publiée : aucune fiche où poser l'épisode
		}
		bySlot[r.Slot] = append(bySlot[r.Slot], r)
	}
	slots := make([]uint32, 0, len(bySlot))
	for s := range bySlot {
		slots = append(slots, s)
	}
	sort.Slice(slots, func(i, j int) bool { return slots[i] < slots[j] })
	nonBinary := 0
	for _, s := range slots {
		list := bySlot[s]
		sort.SliceStable(list, func(i, j int) bool { return list[i].TimestampUS < list[j].TimestampUS })
		acc := episodeAccum{slot: s, fam: EquipFamilyCamo, windows: windows[s], out: out}
		for _, r := range list {
			switch r.Q {
			case filmdec.CamoActiveQ:
				acc.sample(frameOf(r.TimestampUS, origin, step), true)
			case filmdec.CamoInactiveQ:
				acc.sample(frameOf(r.TimestampUS, origin, step), false)
			default:
				nonBinary++
			}
		}
		acc.finish()
	}
	return nonBinary
}

// buildOvershieldEpisodes déroule la règle `q > 64` sur les mesures de bouclier du MÊME
// balayage que les positions (le quantum brut voyage dans BipedPosition.Shield.Q). Les
// positions arrivent DÉJÀ triées par temps (BuildFromPositions trie avant d'assembler).
func buildOvershieldEpisodes(
	sorted []filmdec.BipedPosition, origin, step uint64, windows map[uint32][]lifeWindow, out *[]EquipmentEpisode,
) {
	accs := map[uint32]*episodeAccum{}
	var order []uint32
	for _, p := range sorted {
		if !p.HasShield {
			continue
		}
		w, ok := windows[p.Slot]
		if !ok {
			continue
		}
		a := accs[p.Slot]
		if a == nil {
			a = &episodeAccum{slot: p.Slot, fam: EquipFamilyOvershield, windows: w, out: out}
			accs[p.Slot] = a
			order = append(order, p.Slot)
		}
		a.sample(frameOf(p.TimestampUS, origin, step), p.Shield.Q > filmdec.OvershieldFullQ)
	}
	for _, s := range order {
		accs[s].finish()
	}
}

// equipmentCoverage compte, par famille, les VIES porteuses et les épisodes. Ce ne sont
// PAS des ensembles de slots publiés (published_tracks.go reste le seul constructeur de
// ceux-là), ce sont les vies PORTEUSES d'épisodes.
//
// LA CLÉ EST (slot, image de début de la vie), PAS LE SLOT. Le compteur indexait par slot
// sous ce commentaire-ci : tant que seuls les épisodes de la DERNIÈRE vie survivaient, un
// slot valait une vie et l'écart ne se voyait pas ; le correctif du 2026-09-06, qui publie
// les épisodes de toutes les vies, le rend visible — deux épisodes d'un même slot recyclé
// comptaient pour une seule vie. C'est le défaut symétrique de celui corrigé le même jour
// pour `coverage.grapple.pullLives` (constat C2 de la revue REG-R1).
func equipmentCoverage(eps []EquipmentEpisode, tracks []Track, closedByDeath map[int]bool) *EquipmentCoverage {
	cov := &EquipmentCoverage{TracksTotal: len(tracks)}
	windows := trackFrameWindows(tracks, closedByDeath)
	camoLives := map[[2]int]struct{}{}
	osLives := map[[2]int]struct{}{}
	for _, e := range eps {
		// À défaut de fenêtre (épisode d'un slot sans piste publiée — impossible par
		// construction, l'assembleur les écarte), le slot lui-même sert de clé : mieux vaut
		// compter une vie de trop que perdre l'épisode dans le dénominateur.
		cle := [2]int{int(e.Slot), -1}
		if w, ok := windowFor(windows[e.Slot], e.T0, e.T1); ok {
			cle[1] = w.from
		}
		switch e.Fam {
		case EquipFamilyCamo:
			cov.CamoEpisodes++
			camoLives[cle] = struct{}{}
		case EquipFamilyOvershield:
			cov.OvershieldEpisodes++
			osLives[cle] = struct{}{}
		}
	}
	cov.CamoLives = len(camoLives)
	cov.OvershieldLives = len(osLives)
	return cov
}
