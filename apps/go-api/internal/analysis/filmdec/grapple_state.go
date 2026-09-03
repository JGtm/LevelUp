package filmdec

// grapple_state.go — LES ÉVÉNEMENTS DE GRAPPIN, lus dans les paquets delta.
//
// CE QUE CE BALAYAGE LIT, ET D'OÙ VIENT LA RÈGLE. Le corps tag==3 du composant i59
// `biped-spartan-ability-non-predicted-state-component` (components_biped_anchor.go) est
// l'événement du GRAPPIN : 115 des 117 lectures à porteur identifié tombent sur des vies
// rang 20 (mesure du 2026-08-16, phase E de PLAN_ETAT_ACTIF_EQUIPEMENT), par PAIRES à
// 0,150 s — un corps LÉGER (le tir) puis un corps LOURD (l'accroche). Les deux portent
// une POSITION ABSOLUE quantifiée aux largeurs d'axe de la carte, et les contrôles du
// plan PLAN_GRAPPIN_LIGNE (phase 0.3, trois films, trois cartes) établissent que cette
// position est L'ANCRE : 100 % dans l'emprise du nuage des bipèdes, distance
// joueur->ancre décroissante après l'accroche (témoins mélangés effondrés), point FIXE
// entre les deux membres d'une paire (|P1-P2| médian 0,05-0,07 u quand le joueur bouge
// de 0,42 u).
//
// LES LARGEURS DE POSITION SONT CELLES DE LA CARTE : l'appelant doit avoir installé
// `WorldObjectPrecision` (BuildFromFilm le fait via installWorldObjectPrecision, sous le
// verrou). Les QUANTA BRUTS sont publiés ; la déquantification exige les bornes de la
// carte et vit chez l'assembleur (replay/grapple_lines.go) — pas de bornes, pas de
// coordonnée monde (règle map_bounds.go).
//
// HORS LIGNE (I/O disque sur tout le film) — jamais depuis un chemin de requête.
// L'appelant doit détenir LockProcessDecode (BuildFromFilm le fait) : le hook installé
// est un global de paquet.

import (
	"fmt"

	"levelup/go-api/internal/analysis/filmsource"
)

// grappleComponentName / grappleComponentNameAlt : les deux étiquettes de registre d'i59
// — les films portent l'une OU l'autre (avec ou sans le suffixe `-component`, même
// dualité que le dispatch de consumeByName). L'index d'itérateur est résolu PAR NOM dans
// le registre du film, jamais câblé.
const (
	grappleComponentName    = "biped-spartan-ability-non-predicted-state-component"
	grappleComponentNameAlt = "biped-spartan-ability-non-predicted-state"
)

// GrappleRead est UNE lecture d'événement de grappin, localisée dans le film.
type GrappleRead struct {
	// Slot est l'identifiant bas du biped porteur — le même que celui des trajectoires,
	// donc UNE VIE et non un joueur (le slot migre aux réapparitions).
	Slot uint32
	// Chunk / PacketIndex localisent la lecture dans le film.
	Chunk, PacketIndex int
	// TimestampUS est l'horodatage du paquet porteur — MÊME horloge que BipedPosition.
	TimestampUS uint64
	// Heavy dit si la lecture est le corps LOURD (l'accroche, second membre de la paire).
	// Le corps léger est le tir.
	Heavy bool
	// PosQ : les trois quanta de la position de l'ancre, aux largeurs d'axe de la carte
	// installées au moment du balayage (WorldObjectPrecision.AxisW).
	PosQ [3]uint32
}

// GrappleStats compte ce que la marche a rencontré. Sans ces dénominateurs, une liste de
// lectures ne se juge pas.
type GrappleStats struct {
	// Records est le nombre de records delta biped reconnus.
	Records int
	// WithI59 est le nombre de ces records dont le masque annonce i59.
	WithI59 int
	// Read / Unread : lectures abouties, et records dont la marche n'a pas atteint i59.
	Read, Unread int
	// Tag3 est le nombre de lectures dont le tag externe vaut 3 (les seules qui portent
	// un corps de grappin).
	Tag3 int
	// BodyBroken : corps tag==3 NON décodables (drapeaux != 000 ou valeur interne jamais
	// observée) — comptés et écartés, jamais devinés (cf. components_biped_anchor.go).
	BodyBroken int
}

// ScanFilmGrappleReads décode les événements de grappin (corps tag==3 d'i59) dans les
// paquets delta du film de dir.
//
// UN SEUL DÉCODAGE filmdec À LA FOIS PAR PROCESS : ce balayage installe
// `abilityNonPredictedHook`, qui est un global de paquet. L'appelant doit détenir
// LockProcessDecode (BuildFromFilm le fait). Le hook est restauré à la sortie, y compris
// en cas d'erreur.
//
// ScanFilmGrappleReads est l'ENVELOPPE D2, HORS PRODUCTION ; la cuisson appelle
// [ScanGrappleReads].
func ScanFilmGrappleReads(dir string) ([]GrappleRead, GrappleStats, error) {
	film, err := filmsource.LoadDir(dir, nil)
	if err != nil {
		return nil, GrappleStats{}, err
	}
	return ScanGrappleReads(NewFilmContext(film))
}

// ScanGrappleReads décode les événements de grappin d'un film DEJA CHARGE.
func ScanGrappleReads(fc *FilmContext) ([]GrappleRead, GrappleStats, error) {
	var st GrappleStats
	chunks := fc.ChunkNumbers()
	if len(chunks) == 0 {
		return nil, st, ErrNoFilmChunk
	}
	slots := fc.BipedSlots()
	if slots.Count() == 0 {
		return nil, st, fmt.Errorf("aucun slot biped (ti=%d) dans les keyframes du film", BipedTypeIndex)
	}
	lay, err := fc.I0Layout()
	if err != nil {
		return nil, st, fmt.Errorf("découpage i0 illisible : %w", err)
	}
	arch, err := fc.bipedArchetype()
	if err != nil {
		return nil, st, err
	}
	i59idx := -1
	for _, name := range []string{grappleComponentName, grappleComponentNameAlt} {
		if ids := arch.indicesOf(name); len(ids) > 0 {
			i59idx = ids[0]
			break
		}
	}
	if i59idx < 0 {
		return nil, st, fmt.Errorf("composant %q absent de l'archétype biped du film", grappleComponentName)
	}

	// Le hook est LA grammaire : c'est le déserialiseur lui-même qui publie, on ne relit
	// pas les bits à côté de lui (même règle que ScanFilmAbilityRanks et ScanFilmCamoStates).
	sc := &grappleScanner{st: &st, lay: lay, arch: arch, i59idx: i59idx}
	prev := abilityNonPredictedHook
	SetAbilityNonPredictedHook(func(s AbilityNonPredictedState) { sc.last, sc.got = s, true })
	defer SetAbilityNonPredictedHook(prev)

	minRecord := bipedHeaderBits + bipedIndexBits*bipedMinMaskCnt + lay.TotalBits()
	for _, c := range chunks {
		data, pks, ok := fc.ChunkAt(c)
		if !ok {
			continue
		}
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta {
				continue
			}
			pay := pk.Payload(data)
			total := len(pay) * 8
			for p := 0; p+minRecord <= total; {
				i0, slot, idx, ok := matchBipedHeader(pay, p, total, slots, true, lay)
				if !ok {
					p++
					continue
				}
				st.Records++
				if maskHas(idx, i59idx) {
					sc.account(pay, i0, total, idx, slot, c, pk)
				}
				p = i0 + lay.TotalBits()
			}
		}
	}
	return sc.out, st, nil
}

// grappleScanner porte l'état du balayage : compteurs, capture du hook, et sortie.
type grappleScanner struct {
	st     *GrappleStats
	out    []GrappleRead
	lay    I0Layout
	arch   Archetype
	i59idx int
	last   AbilityNonPredictedState
	got    bool
}

// account marche UN record annonçant i59 et impute la lecture aux compteurs. La marche
// vers la cible tolère un corps cassé (le tag et le début du corps restent publiés par le
// hook — c'est la SUITE du record qui n'est plus digne de confiance).
func (sc *grappleScanner) account(pay []byte, i0, total int, idx []int,
	slot uint32, chunk int, pk FilmPacket) {
	sc.st.WithI59++
	sc.got = false
	walkRecordTo(pay, i0, total, idx, sc.lay, sc.arch, sc.i59idx)
	if !sc.got {
		sc.st.Unread++
		return
	}
	sc.st.Read++
	if sc.last.Tag != 3 {
		return
	}
	sc.st.Tag3++
	if !sc.last.BodyOK {
		sc.st.BodyBroken++
		return
	}
	heavy := sc.last.Inner == anchorInnerHeavy
	if !heavy && sc.last.Inner != anchorInnerLight {
		return // valeur interne connue mais hors des deux corps de grappin : rien à publier
	}
	sc.out = append(sc.out, GrappleRead{
		Slot: slot, Chunk: chunk, PacketIndex: pk.Index, TimestampUS: pk.TimestampUS,
		Heavy: heavy, PosQ: sc.last.PosQ,
	})
}
