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
}

// trackFrameWindows indexe les fenêtres [StartFrame, EndFrame] des vies publiées. Ce
// n'est PAS l'ensemble booléen des slots publiés (published_tracks.go reste son seul
// constructeur) : la fenêtre sert à FERMER un épisode à la mort et à borner ses frames.
func trackFrameWindows(tracks []Track) map[uint32][2]int {
	out := make(map[uint32][2]int, len(tracks))
	for _, t := range tracks {
		out[t.Slot] = [2]int{t.StartFrame, t.EndFrame}
	}
	return out
}

// episodeAccum accumule les épisodes d'UNE famille pour UN slot : machine à deux états
// (ouvert / fermé), fermeture à la fin de vie si rien ne l'a mesurée.
type episodeAccum struct {
	slot      uint32
	fam       string
	window    [2]int
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

// close émet l'épisode borné à la fenêtre de la vie. Un épisode entièrement hors de la
// fenêtre publiée est écarté : il n'a aucune fiche où s'afficher.
func (a *episodeAccum) close(endFrame int, endRead bool) {
	a.open = false
	t0, t1 := a.openFrame, endFrame
	if t0 < a.window[0] {
		t0 = a.window[0]
	}
	if t1 > a.window[1] {
		t1 = a.window[1]
	}
	if t1 < t0 {
		return
	}
	*a.out = append(*a.out, EquipmentEpisode{Slot: a.slot, Fam: a.fam, T0: t0, T1: t1, EndRead: endRead})
}

// finish ferme un épisode resté ouvert À LA FIN DE LA VIE : la fin de piste date la mort
// (EndRead=false — rien n'a mesuré une désactivation).
func (a *episodeAccum) finish() {
	if a.open {
		a.close(a.window[1], false)
	}
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
) ([]EquipmentEpisode, int) {
	if len(tracks) == 0 || step == 0 {
		return nil, 0
	}
	windows := trackFrameWindows(tracks)
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
	camo []filmdec.CamoRead, origin, step uint64, windows map[uint32][2]int, out *[]EquipmentEpisode,
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
		acc := episodeAccum{slot: s, fam: EquipFamilyCamo, window: windows[s], out: out}
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
	sorted []filmdec.BipedPosition, origin, step uint64, windows map[uint32][2]int, out *[]EquipmentEpisode,
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
			a = &episodeAccum{slot: p.Slot, fam: EquipFamilyOvershield, window: w, out: out}
			accs[p.Slot] = a
			order = append(order, p.Slot)
		}
		a.sample(frameOf(p.TimestampUS, origin, step), p.Shield.Q > filmdec.OvershieldFullQ)
	}
	for _, s := range order {
		accs[s].finish()
	}
}

// equipmentCoverage compte, par famille, les vies porteuses et les épisodes. Les
// ensembles de slots sont des sets struct{} : ce ne sont PAS des ensembles de slots
// publiés (published_tracks.go reste le seul constructeur de ceux-là), ce sont les vies
// PORTEUSES d'épisodes.
func equipmentCoverage(eps []EquipmentEpisode, tracks []Track) *EquipmentCoverage {
	cov := &EquipmentCoverage{TracksTotal: len(tracks)}
	camoSlots := map[uint32]struct{}{}
	osSlots := map[uint32]struct{}{}
	for _, e := range eps {
		switch e.Fam {
		case EquipFamilyCamo:
			cov.CamoEpisodes++
			camoSlots[e.Slot] = struct{}{}
		case EquipFamilyOvershield:
			cov.OvershieldEpisodes++
			osSlots[e.Slot] = struct{}{}
		}
	}
	cov.CamoLives = len(camoSlots)
	cov.OvershieldLives = len(osSlots)
	return cov
}
