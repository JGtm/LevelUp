package grammar

// object_deaths.go — LA MORT ÉCRITE D'UNE ENTITÉ DU MONDE, lue au composant
// `object-dead-state-component` par la marche.
//
// LE FAIT. Le moteur ne déclare AUCUN composant « véhicule détruit » : relu chez l'écrivain
// (Ghidra, 2026-09-05), il ne connaît que des composants d'OBJET — `object-body-vitality`,
// `object-shield-vitality`, `object-damage-sections`, `object-dead-state`. Le dead-state est
// donc le SEUL signal d'état gravé à la mort, et il est GÉNÉRIQUE : une destruction de véhicule
// passe par le même code de mort qu'un kill de joueur (`FUN_1404d9828` -> `FUN_140adefbc` ->
// `FUN_142c4e850`, le classifieur qui choisit `enemy_vehicle_kill` quand la victime n'a pas
// d'index de joueur).
//
// LE VERROU QUI A CACHÉ CE FAIT PENDANT TOUT UN CHANTIER, et qui est la raison d'être de ce
// fichier : le filtre `DesyncAt == -1`. `DesyncAt` est l'index du PREMIER composant présent NON
// PORTÉ ; tout ce qui le précède a été consommé dans l'ordre. Sur `ti=40`, 65 des 69 records qui
// DÉCLARENT le dead-state rompent à `i30`..`i36` (`vehicle-auto-turret-triggers`,
// `vehicle-auto-turret-aiming-vector`, `vehicle-transformed-or-desired-open-state-changed`,
// `vehicle-type-state`) — TOUS après `i11`. Les bits du dead-state avaient donc été lus au bon
// endroit ; seule la QUEUE du record était inconnue. Le filtre strict, hérité de `killsource`
// (où le risque de rupture est en AMONT, chez le bipède), jetait des morts de véhicule
// PARFAITEMENT LUES : `ti=40` rendait 0 dead-state là où le film en écrit 21 à 27 par film.
//
// LA RÈGLE POSÉE ICI : un dead-state est accepté quand `DesyncAt == -1` (record entièrement
// porté) OU quand `DesyncAt > index(dead-state)` (queue inconnue, tête lue). Les deux qualités
// sont comptées À PART (`TailDesync`) — elles ne se mélangent jamais dans une mesure.
//
// CE QUE CE FICHIER NE FAIT PAS : grouper les dead-states d'une même vie en UN épisode de mort.
// Le film re-réplique le dead-state sur plusieurs ticks (le bipède a le même profil : 84
// dead-states pour ~60 morts réelles) ; le groupement est de l'ASSEMBLAGE et vit chez
// l'appelant, avec sa notion de vie.
//
// HORS LIGNE par construction (le film entier est parcouru) — jamais depuis un chemin de requête.
//
// Mesure et gates d'origine : `.ai/V7.5/GATE_V13_DEADSTATE_MARCHE_2026-09-05.md` et
// `.ai/V7.5/film_re/NOTE_V13_DEADSTATE_VEHICULE_2026-09-05.md`.

import (
	"sort"

	"levelup/go-api/internal/analysis/filmsource"
)

// deadStateComponentName est le nom du composant `i11` dans le registre ECS du film. Il est
// NOMMÉ par le film lui-même (chunk 0) : l'index se RÉSOUT par ce nom, archétype par archétype,
// et n'est jamais écrit en dur — deux archétypes ne le portent pas au même rang.
const deadStateComponentName = "object-dead-state-component"

// ObjectDeath est UNE mort écrite : le composant dead-state d'une entité, porté à `Mort`.
type ObjectDeath struct {
	// TimestampUS est l'instant du PAQUET qui la porte, sur l'horloge du film.
	TimestampUS uint64
	// Slot / Gen identifient la VIE de l'entité (le pool de slots reboucle, la génération fait
	// 2 bits) ; TypeIndex est son archétype.
	Slot, Gen, TypeIndex uint32
	// Dead est le composant capturé. Chez le bipède il porte le couple victime / tueur ; sur
	// les autres archétypes seul le drapeau `Mort` est établi (les champs restent lus, leur
	// SENS ne l'est pas — cf. la réserve 1 de la note V13).
	Dead DeadState
	// TailDesync dit que le record a rompu APRÈS le dead-state : la tête est lue au bon
	// endroit, la queue du record n'est pas modélisée. Compté à part, jamais confondu avec un
	// record entièrement porté.
	TailDesync bool
}

// ObjectDeathStats porte les DÉNOMINATEURS sans lesquels aucun compte ne se publie : un compte
// faible sous une couverture faible ne conclut pas à l'absence, il conclut « sous-instrumenté ».
type ObjectDeathStats struct {
	// Config est le cadre RETENU par la calibration : il se publie, il ne se suppose pas.
	Config FrameConfig
	// CadreParDefaut dit que le profil de calibration etait PLAT — aucune largeur candidate n a
	// domine son dauphin — et que le cadre rendu est celui par defaut. La marche a tourne, mais
	// sur une largeur que RIEN n a confirmee : c est un repli
	// (`repli_cadre_de_marche_par_defaut_conserve`), et il se compte.
	CadreParDefaut bool
	// CadreLocalises / CadreDauphin / CadreEvenements sont les DENOMINATEURS de cette decision :
	// paquets a evenements localises par le candidat retenu, par son dauphin, et leur total sur
	// l echantillon de calibrage. Sans eux, « cadre idLow=13 » ne dit pas s il a ete choisi par
	// une marge de six ou par une marge de rien.
	CadreLocalises, CadreDauphin, CadreEvenements int
	// Keyframes / Deltas : ce que le film a offert à la marche.
	Keyframes, Deltas int
	// Packets / EventPackets / LocatedPackets : paquets marchés, dont porteurs d'une liste
	// d'événements, dont effectivement localisés. Un écart entre les deux derniers est la part
	// du film que la marche n'a PAS lue.
	Packets, EventPackets, LocatedPackets int
	// Records / CleanRecords : par archétype, records atteints et records entièrement portés.
	Records, CleanRecords map[uint32]int
	// MaskDeclared / MaskDeclaredDesync : par archétype, records dont le MASQUE déclare le
	// dead-state, et ceux d'entre eux qui ont désynchronisé. C'est LE contrôle qui sépare
	// « cette entité ne meurt pas dans le film » de « ses morts sont dans la fraction qu'on
	// jette » — le masque se lit AVANT toute consommation de corps.
	MaskDeclared, MaskDeclaredDesync map[uint32]int
}

// newObjectDeathStats rend des compteurs prêts à l'emploi (les cartes ne sont jamais nil : un
// lecteur de couverture ne doit pas avoir à tester).
func newObjectDeathStats() ObjectDeathStats {
	return ObjectDeathStats{
		Records: map[uint32]int{}, CleanRecords: map[uint32]int{},
		MaskDeclared: map[uint32]int{}, MaskDeclaredDesync: map[uint32]int{},
	}
}

// ScanFilmObjectDeaths est l'ENVELOPPE HORS PRODUCTION (charge le film depuis `dir`) ; la
// cuisson appelle [ScanObjectDeaths].
func ScanFilmObjectDeaths(dir string) ([]ObjectDeath, ObjectDeathStats, error) {
	film, err := filmsource.LoadDir(dir, nil)
	if err != nil {
		return nil, newObjectDeathStats(), err
	}
	return ScanObjectDeaths(contexteDeBobine(film))
}

// ScanObjectDeaths marche les paquets delta d'un film DÉJÀ CHARGÉ et rend toutes les morts
// écrites, TOUS archétypes confondus, triées par instant puis par slot.
//
// AUCUN FILTRE DE BANDE — et c'est un acquis de la mesure : la marche range par ARCHÉTYPE
// (`FrameRecord.TypeIndex`), jamais par bande de slots dérivée des images-clés. Le filtre de
// bande aurait perdu 2 à 5 morts de bipède par film, le film liant aussi des entités par
// records NEW en cours de flux.
func ScanObjectDeaths(fc *FilmContext) ([]ObjectDeath, ObjectDeathStats, error) {
	st := newObjectDeathStats()
	reg, err := fc.Registry()
	if err != nil {
		return nil, st, err
	}
	kfs, deltas := marchPacketsOf(fc)
	st.Keyframes, st.Deltas = len(kfs), len(deltas)
	if len(deltas) == 0 {
		return nil, st, nil
	}
	cfg, parDefaut, meilleur, dauphin := calibrateFrameConfig(reg, kfs, deltas, fc.CadreDeBalayage())
	st.Config, st.CadreParDefaut = cfg, parDefaut
	st.CadreLocalises, st.CadreDauphin, st.CadreEvenements = meilleur.located, dauphin.located, meilleur.events
	h := &objectDeathHarvest{reg: reg, idx: map[uint32]int{}, st: &st}
	tl := newMarchTimeline(reg, kfs)
	for _, d := range deltas {
		w := tl.advanceTo(d.timestampUS)
		start, withEvents, ok := marchStartOf(d.payload, w, cfg)
		if withEvents {
			st.EventPackets++
		}
		if !ok {
			continue
		}
		if withEvents {
			st.LocatedPackets++
		}
		st.Packets++
		h.harvest(marchRecordsOf(d.payload, w, cfg, start), d.timestampUS)
	}
	return dedupObjectDeaths(h.out), st, nil
}

// marchPacketsOf relève les images-clés décodées et les paquets delta du film, TRIÉS par
// instant — le curseur de la timeline exige des appels croissants.
//
// Les payloads sont des VUES sur les octets du film, déjà résidents : les retenir ne recopie
// rien.
func marchPacketsOf(fc *FilmContext) ([]marchKeyframe, []marchDelta) {
	var kfs []marchKeyframe
	var deltas []marchDelta
	for _, c := range fc.ChunkNumbers() {
		data, pks, ok := fc.ChunkAt(c)
		if !ok {
			continue
		}
		for _, pk := range pks {
			switch pk.Type {
			case PacketTypeKeyframe:
				kfs = append(kfs, marchKeyframe{pk.TimestampUS, WalkKeyframeWorld(pk.Payload(data))})
			case PacketTypeDelta:
				deltas = append(deltas, marchDelta{pk.TimestampUS, pk.Payload(data)})
			}
		}
	}
	sort.SliceStable(deltas, func(i, j int) bool {
		return deltas[i].timestampUS < deltas[j].timestampUS
	})
	return kfs, deltas
}

// objectDeathHarvest porte ce que la récolte doit connaître (règle des 5 paramètres) : le
// registre, le mémo des index de dead-state par archétype, les compteurs et la sortie.
type objectDeathHarvest struct {
	reg *Registry
	idx map[uint32]int
	st  *ObjectDeathStats
	out []ObjectDeath
}

// deadStateIndex rend l'index du composant dead-state dans l'archétype `ti`, résolu PAR LE NOM
// du registre, ou -1 quand l'archétype ne le porte pas.
func (h *objectDeathHarvest) deadStateIndex(ti uint32) int {
	if i, ok := h.idx[ti]; ok {
		return i
	}
	out := -1
	if a, ok := h.reg.Archetype(int(ti)); ok {
		for i, c := range a.Components {
			if c == deadStateComponentName {
				out = i
				break
			}
		}
	}
	h.idx[ti] = out
	return out
}

// harvest range les records d'un paquet : compteurs de couverture pour tous, morts pour ceux
// qui en portent une lisible.
func (h *objectDeathHarvest) harvest(recs []FrameRecord, atUS uint64) {
	for i := range recs {
		r := &recs[i]
		h.note(r)
		if r.Trace.Dead == nil || !r.Trace.Dead.Mort {
			continue
		}
		tail, ok := h.accept(r)
		if !ok {
			continue
		}
		h.out = append(h.out, ObjectDeath{
			TimestampUS: atUS, Slot: r.Slot, Gen: r.ID >> 30, TypeIndex: r.TypeIndex,
			Dead: *r.Trace.Dead, TailDesync: tail,
		})
	}
}

// accept applique LA règle du lot : record entièrement porté, ou rupture strictement APRÈS le
// dead-state. Rend aussi la qualité retenue.
func (h *objectDeathHarvest) accept(r *FrameRecord) (tailDesync, ok bool) {
	if r.DesyncAt == -1 {
		return false, true
	}
	di := h.deadStateIndex(r.TypeIndex)
	if di >= 0 && r.DesyncAt > di {
		return true, true
	}
	return false, false
}

// note alimente les dénominateurs de couverture, dont le CONTRÔLE DE MASQUE.
func (h *objectDeathHarvest) note(r *FrameRecord) {
	h.st.Records[r.TypeIndex]++
	if di := h.deadStateIndex(r.TypeIndex); di >= 0 && di < 64 && r.Trace.Mask&(1<<uint(di)) != 0 {
		h.st.MaskDeclared[r.TypeIndex]++
		if r.DesyncAt != -1 {
			h.st.MaskDeclaredDesync[r.TypeIndex]++
		}
	}
	if r.DesyncAt == -1 {
		h.st.CleanRecords[r.TypeIndex]++
	}
}

// dedupObjectDeaths trie et déduplique : une entité ne meurt qu'une fois à un instant donné, et
// plusieurs VUES de réplication d'un même paquet republient le même dead-state.
//
// LA QUALITÉ LA MEILLEURE GAGNE : si le même instant est vu par un record entièrement porté ET
// par un record à queue inconnue, c'est le premier qui est retenu — la marche ne doit pas
// dégrader une lecture propre parce qu'une vue ultérieure a rompu.
func dedupObjectDeaths(in []ObjectDeath) []ObjectDeath {
	if len(in) == 0 {
		return nil
	}
	sort.SliceStable(in, func(i, j int) bool {
		switch {
		case in[i].TimestampUS != in[j].TimestampUS:
			return in[i].TimestampUS < in[j].TimestampUS
		case in[i].Slot != in[j].Slot:
			return in[i].Slot < in[j].Slot
		case in[i].Gen != in[j].Gen:
			return in[i].Gen < in[j].Gen
		default:
			return !in[i].TailDesync && in[j].TailDesync
		}
	})
	type key struct {
		slot, gen uint32
		at        uint64
	}
	seen := map[key]bool{}
	out := make([]ObjectDeath, 0, len(in))
	for _, d := range in {
		k := key{d.Slot, d.Gen, d.TimestampUS}
		if seen[k] {
			continue
		}
		seen[k] = true
		out = append(out, d)
	}
	return out
}
