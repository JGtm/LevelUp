package grammar

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
// est un global de paquet.

import (
	"fmt"

	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// grappleComponentName / grappleComponentNameAlt : les deux étiquettes de registre d'i59
// — les films portent l'une OU l'autre (avec ou sans le suffixe `-component`, même
// dualité que le dispatch de consumeByName). L'index d'itérateur est résolu PAR NOM dans
// le registre du film, jamais câblé.
const (
	grappleComponentName    = "biped-spartan-ability-non-predicted-state-component"
	grappleComponentNameAlt = "biped-spartan-ability-non-predicted-state"
)

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
// ScanFilmGrappleReads est l'ENVELOPPE D2, HORS PRODUCTION ; la cuisson appelle
// [ScanGrappleReads].
func ScanFilmGrappleReads(dir string) ([]types.GrappleRead, GrappleStats, error) {
	film, err := source.LoadDir(dir, nil)
	if err != nil {
		return nil, GrappleStats{}, err
	}
	return ScanGrappleReads(contexteDeBobine(film))
}

// ScanGrappleReads décode les événements de grappin d'un film DEJA CHARGE.
func ScanGrappleReads(fc *FilmContext) ([]types.GrappleRead, GrappleStats, error) {
	var st GrappleStats
	chunks := fc.ChunkNumbers()
	if len(chunks) == 0 {
		return nil, st, ErrNoFilmChunk
	}
	slots := fc.BipedSlots()
	if slots.Count() == 0 {
		return nil, st, fmt.Errorf("aucun slot biped (ti=%d) dans les keyframes du film", BipedTypeIndex)
	}
	if _, err := fc.I0Layout(); err != nil {
		return nil, st, fmt.Errorf("découpage i0 illisible : %w", err)
	}
	arch, err := fc.bipedArchetype()
	if err != nil {
		return nil, st, err
	}
	i59idx := componentIndexOfAny(arch, grappleComponentName, grappleComponentNameAlt)
	if i59idx < 0 {
		return nil, st, fmt.Errorf("composant %q absent de l'archétype biped du film", grappleComponentName)
	}
	lu, err := fc.lecturesBipedes()
	if err != nil {
		return nil, st, err
	}

	// Le hook est LA grammaire : c'est le déserialiseur lui-même qui a publié pendant la marche,
	// et la lecture rejoue sa publication ; on ne relit pas les bits (même règle que
	// ScanFilmAbilityRanks et ScanFilmCamoStates).
	sc := &grappleScanner{st: &st, i59idx: i59idx, obs: NouvelleObservation()}
	sc.obs.AbilityNonPredictedHook = func(s AbilityNonPredictedState) { sc.last, sc.got = s, true }
	st.Records = lu.examines
	for i := range lu.records {
		if r := &lu.records[i]; r.annonce(i59idx) {
			sc.account(r)
		}
	}
	return sc.out, st, nil
}

// grappleScanner porte l'état du balayage : compteurs, l'observation sur laquelle les lectures se
// rejouent et la capture de son hook, et sortie.
type grappleScanner struct {
	st     *GrappleStats
	out    []types.GrappleRead
	obs    *Observation
	i59idx int
	last   AbilityNonPredictedState
	got    bool
}

// account rejoue UN record annonçant i59 et impute la lecture aux compteurs. La lecture vers la
// cible tolère un corps cassé (le tag et le début du corps restent publiés par le hook — c'est la
// SUITE du record qui n'est plus digne de confiance).
func (sc *grappleScanner) account(r *recordBipedeLu) {
	slot, chunk, pk := r.Slot, r.Chunk, r.Packet
	sc.st.WithI59++
	sc.got = false
	r.parcourirJusqua(sc.obs, sc.i59idx)
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
	if !sc.last.PosCarte {
		return // pas de position aux largeurs de la carte : aucun point d'ancre à publier
	}
	sc.out = append(sc.out, types.GrappleRead{
		Slot: slot, Chunk: chunk, PacketIndex: pk.Index, TimestampUS: pk.TimestampUS,
		Heavy: heavy, PosQ: sc.last.PosQ,
	})
}
