//go:build research

package grammar

// masque_5_10_5_research_test.go — LES 32 DRAPEAUX DE `PlayerGameEventSmall`, NOTES CONTRE LES
// ETIQUETTES PHYSIQUES DU BIPEDE (lot 5.10.5).
//
// LA QUESTION, ET POURQUOI ELLE SE POSE MAINTENANT. Le type 82 (`0xE9`) se termine par un
// masque de 32 `R(1)` INCONDITIONNELS (`FUN_14080add8` -> `FUN_14080ae28`, boucle de 32
// lectures d un bit). Le decodeur le SAUTAIT depuis le lot 1 — personne ne l a jamais note
// contre un fait physique. D4 (5.7) l a consigne comme candidat au DECLENCHEUR du saut, que le
// lot 5.9 n a publie que DERIVE de la physique (`jumpDerived`, H = 0,85 m).
//
// LA MESURE, ET SES TROIS ETIQUETTES. Chaque bit est note contre les DEBUTS d etat que la
// production lit deja (`ScanMovementStates`, la fonction de production, pas une copie) :
//
//	(a) SAUT      `jumpDerived` — l episode derive de la vitesse verticale (lot 5.9)
//	(b) SPRINT    `sprint` — `i57` brut 2, la fente 1 nommee par l image (lot 5.9.5)
//	(c) ACCROUPI  `crouch` — `i29`, le booleen d accroupissement
//
// LE JUGE EST ECRIT AVANT LA MESURE : un bit est NOMME s il depasse 90 % en precision ET en
// rappel contre UNE etiquette. Sinon, le tableau des 32 scores est publie tel quel — c est le
// resultat, pas un echec.
//
// LA FENETRE EST DE DEUX TICKS, et le tick est MESURE sur le film (ecart median entre deux
// paquets delta), jamais suppose.
//
// LE PONT VERS LE SLOT est celui du lot 1 : `lot1chReferenceBase` = 512, la base de la
// categorie 4 du flux. Les deux premieres references d en-tete sont notees SEPAREMENT — la
// mesure dit laquelle porte le joueur, elle ne le suppose pas.
//
// ENV : `SIEGE510_FILM`, `SIEGE510_CARTE`, `SIEGE510_BORNES` (memes portes que le reste du lot).

import (
	"fmt"
	"os"
	"sort"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// m5105Seuil est le juge, ecrit avant la mesure : precision ET rappel au-dessus de 90 %.
const m5105Seuil = 90.0

// m5105Evt est UN evenement de type 82 : son instant, son masque, et les deux slots candidats.
type m5105Evt struct {
	ts           uint64
	mask         uint64
	slot0, slot1 int64
	slot2        int64
}

// m5105Debut est UN debut d etat physique : le slot du bipede et l instant.
type m5105Debut struct {
	slot uint32
	ts   uint64
}

// TestMasque5105 publie la table des 32 bits, notes contre les trois etiquettes.
func TestMasque5105(t *testing.T) {
	dir := os.Getenv("SIEGE510_FILM")
	if dir == "" {
		t.Skip("SIEGE510_FILM absent : instrument de recherche")
	}
	film, err := source.LoadDir(dir, nil)
	if err != nil {
		t.Fatalf("LoadDir %s : %v", dir, err)
	}
	fc := s510Contexte(t, film)
	evts, tick := m5105Evenements(t, fc)
	debuts := m5105Debuts(t, fc)
	if len(evts) == 0 {
		t.Log("AUCUN evenement de type 82 : rien a noter.")
		return
	}
	fenetre := 2 * tick
	t.Logf("POPULATION : %d evenements de type 82 · tick median %d us · fenetre +/- %d us",
		len(evts), tick, fenetre)
	m5105Bandes(t, evts, debuts)
	for _, ref := range []int{0, 1} {
		t.Logf("=== REFERENCE D EN-TETE %d (slot = %d + index) ===", ref, lot1chReferenceBase)
		m5105Table(t, evts, debuts, ref, fenetre)
	}
	m5105TempsSeul(t, evts, debuts, fenetre)
}

// m5105TempsSeulDecalage est le TEMOIN de la mesure sans slot : le meme calcul, les etiquettes
// decalees de trois secondes. Ce que le hasard rend, il le rend aussi decale.
const m5105TempsSeulDecalage = uint64(3_000_000)

// m5105TempsSeul note chaque bit SANS le pont de slot : un evenement compte s il tombe a moins
// d une fenetre d un debut, QUEL QUE SOIT le joueur. C est la seule mesure disponible quand les
// references d en-tete sont absentes — et elle porte son temoin decale, sans lequel une
// coincidence de population dense se lirait comme un signal.
func m5105TempsSeul(t *testing.T, evts []m5105Evt, debuts map[string][]m5105Debut, fenetre uint64) {
	t.Helper()
	t.Log("=== SANS PONT DE SLOT (coincidence de temps seule, avec temoin decale de 3 s) ===")
	for _, g := range []string{types.MovementJumpDerived, types.MovementSprint, types.MovementCrouch} {
		var tris, decales []uint64
		for _, d := range debuts[g] {
			tris = append(tris, d.ts)
			decales = append(decales, d.ts+m5105TempsSeulDecalage)
		}
		sort.Slice(tris, func(i, j int) bool { return tris[i] < tris[j] })
		sort.Slice(decales, func(i, j int) bool { return decales[i] < decales[j] })
		for bit := 0; bit < 32; bit++ {
			var n, justes, temoin int
			for _, e := range evts {
				if e.mask&(1<<uint(bit)) == 0 {
					continue
				}
				n++
				if _, ok := m5105Proche(tris, e.ts, fenetre); ok {
					justes++
				}
				if _, ok := m5105Proche(decales, e.ts, fenetre); ok {
					temoin++
				}
			}
			if n < 10 {
				continue
			}
			t.Logf("  %s · bit %2d : %d evenements · coincidence %.1f %% · temoin decale %.1f %%",
				m5105Court(g), bit, n, m533bPart(justes, n), m533bPart(temoin, n))
		}
	}
}

// m5105Bandes publie les BANDES DE SLOTS des deux cotes. Sans elles, un score nul ne se lit pas :
// il peut venir d un pont de slot faux autant que d une absence de correlation.
func m5105Bandes(t *testing.T, evts []m5105Evt, debuts map[string][]m5105Debut) {
	t.Helper()
	cnt0, cnt1 := map[int]int{}, map[int]int{}
	var absents int
	for _, e := range evts {
		if e.slot0 < 0 {
			absents++
		} else {
			cnt0[int(e.slot0)+lot1chReferenceBase]++
		}
		if e.slot1 >= 0 {
			cnt1[int(e.slot1)+lot1chReferenceBase]++
		}
	}
	var p0, p1, p2 int
	for _, e := range evts {
		if e.slot0 >= 0 {
			p0++
		}
		if e.slot1 >= 0 {
			p1++
		}
		if e.slot2 >= 0 {
			p2++
		}
	}
	t.Logf("  REFERENCES D EN-TETE PRESENTES : ref0 %d/%d · ref1 %d · ref2 %d",
		p0, len(evts), p1, p2)
	t.Logf("  SLOTS des evenements : ref0 %s (%d sans ref0) · ref1 %s",
		m533cTable(cnt0), absents, m533cTable(cnt1))
	for _, g := range []string{types.MovementJumpDerived, types.MovementSprint, types.MovementCrouch} {
		cnt := map[int]int{}
		for _, d := range debuts[g] {
			cnt[int(d.slot)]++
		}
		t.Logf("  SLOTS des debuts %s : %s", m5105Court(g), m533cTable(cnt))
	}
}

// m5105Evenements marche tous les paquets et rend les evenements de type 82 avec leur masque,
// plus le tick MESURE (ecart median entre deux paquets delta consecutifs).
func m5105Evenements(t *testing.T, fc *FilmContext) ([]m5105Evt, uint64) {
	t.Helper()
	var out []m5105Evt
	var ecarts []uint64
	var prev uint64
	for _, c := range fc.ChunkNumbers() {
		data, pks, ok := fc.ChunkAt(c)
		if !ok {
			continue
		}
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta || pk.Size < 2 {
				continue
			}
			if prev > 0 && pk.TimestampUS > prev {
				ecarts = append(ecarts, pk.TimestampUS-prev)
			}
			prev = pk.TimestampUS
			pay := pk.Payload(data)
			if pay[0] != 0xE9 {
				continue
			}
			r, estType82 := pgesDecodePacket(pay, pk.TimestampUS)
			if !estType82 {
				continue
			}
			out = append(out, m5105Evt{
				ts: r.ts, mask: r.payload.mask, slot0: r.ref0, slot1: r.ref1, slot2: r.ref2,
			})
		}
	}
	sort.Slice(ecarts, func(i, j int) bool { return ecarts[i] < ecarts[j] })
	tick := uint64(16_667)
	if len(ecarts) > 0 {
		tick = ecarts[len(ecarts)/2]
	}
	return out, tick
}

// m5105Debuts rend les DEBUTS d etat par genre, lus par la fonction de PRODUCTION.
func m5105Debuts(t *testing.T, fc *FilmContext) map[string][]m5105Debut {
	t.Helper()
	reads, st, err := ScanMovementStates(fc)
	if err != nil {
		t.Fatalf("ScanMovementStates : %v", err)
	}
	out := map[string][]m5105Debut{}
	for _, r := range reads {
		if r.Kind != types.MovementJumpDerived && !r.On {
			continue // un ARRET n est pas un debut
		}
		out[r.Kind] = append(out[r.Kind], m5105Debut{slot: r.Slot, ts: r.TimestampUS})
	}
	t.Logf("ETIQUETTES (`ScanMovementStates`, %d records, %d lectures) : saut %d · sprint %d · "+
		"accroupi %d", st.Records, st.Read, len(out[types.MovementJumpDerived]),
		len(out[types.MovementSprint]), len(out[types.MovementCrouch]))
	return out
}

// m5105Table note les 32 bits contre les trois etiquettes et publie une ligne par bit.
func m5105Table(t *testing.T, evts []m5105Evt, debuts map[string][]m5105Debut, ref int,
	fenetre uint64) {
	t.Helper()
	genres := []string{types.MovementJumpDerived, types.MovementSprint, types.MovementCrouch}
	var nommes []string
	for bit := 0; bit < 32; bit++ {
		var poses []m5105Evt
		for _, e := range evts {
			if e.mask&(1<<uint(bit)) != 0 {
				poses = append(poses, e)
			}
		}
		if len(poses) == 0 {
			continue
		}
		ligne := fmt.Sprintf("  bit %2d : %5d evenements", bit, len(poses))
		for _, g := range genres {
			p, r := m5105Score(poses, debuts[g], ref, fenetre)
			ligne += fmt.Sprintf(" · %s P %.1f %% R %.1f %%", m5105Court(g), p, r)
			if p >= m5105Seuil && r >= m5105Seuil {
				nommes = append(nommes, fmt.Sprintf("bit %d -> %s (P %.1f %%, R %.1f %%)",
					bit, g, p, r))
			}
		}
		t.Log(ligne)
	}
	if len(nommes) == 0 {
		t.Logf("AUCUN BIT NOMME (juge : P et R au-dessus de %.0f %%) — la table ci-dessus EST le "+
			"resultat.", m5105Seuil)
		return
	}
	for _, n := range nommes {
		t.Logf("BIT NOMME : %s", n)
	}
}

// m5105Score rend la precision et le rappel d un bit contre une etiquette.
//
// PRECISION : la part des evenements porteurs du bit qui tombent a moins d une fenetre d un
// debut du MEME slot. RAPPEL : la part des debuts couverts par un tel evenement.
func m5105Score(poses []m5105Evt, debuts []m5105Debut, ref int, fenetre uint64) (float64, float64) {
	if len(poses) == 0 || len(debuts) == 0 {
		return 0, 0
	}
	parSlot := map[uint32][]uint64{}
	for _, d := range debuts {
		parSlot[d.slot] = append(parSlot[d.slot], d.ts)
	}
	for s := range parSlot {
		sort.Slice(parSlot[s], func(i, j int) bool { return parSlot[s][i] < parSlot[s][j] })
	}
	couverts := map[string]bool{}
	var justes int
	for _, e := range poses {
		slot, ok := m5105Slot(e, ref)
		if !ok {
			continue
		}
		if ts, trouve := m5105Proche(parSlot[slot], e.ts, fenetre); trouve {
			justes++
			couverts[fmt.Sprintf("%d:%d", slot, ts)] = true
		}
	}
	return m533bPart(justes, len(poses)), m533bPart(len(couverts), len(debuts))
}

// m5105Slot resout le slot du joueur d un evenement par la base de la categorie 4.
func m5105Slot(e m5105Evt, ref int) (uint32, bool) {
	idx := e.slot0
	if ref == 1 {
		idx = e.slot1
	}
	if idx < 0 {
		return 0, false
	}
	return uint32(int64(lot1chReferenceBase) + idx), true //nolint:gosec // index du flux
}

// m5105Proche rend l instant du debut le plus proche dans la fenetre, s il existe.
func m5105Proche(tris []uint64, at, fenetre uint64) (uint64, bool) {
	i := sort.Search(len(tris), func(k int) bool { return tris[k]+fenetre >= at })
	for ; i < len(tris) && tris[i] <= at+fenetre; i++ {
		return tris[i], true
	}
	return 0, false
}

// m5105Court abrege un genre pour la table.
func m5105Court(genre string) string {
	switch genre {
	case types.MovementJumpDerived:
		return "saut"
	case types.MovementSprint:
		return "sprint"
	}
	return "accroupi"
}
