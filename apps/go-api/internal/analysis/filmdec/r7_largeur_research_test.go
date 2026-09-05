package filmdec

// r7_largeur_research_test.go — lot R7 : L'EPREUVE DECISIVE DES LARGEURS DE CHARGE.
//
// POURQUOI CELLE-CI ET PAS UNE AUTRE. Toutes les mesures precedentes melangent deux choses :
// la largeur de charge d'un type, et la derive accumulee par les evenements qui le precedent
// dans la liste. On ne peut pas incriminer un type sur une liste ou il arrive en 4e position.
//
// L'ISOLEMENT. Sur une liste d'UN SEUL evenement, le cadrage de cet evenement est CERTAIN
// (position 1 : bit de config, bit de continuation, R(7) type — aucune ambiguite), et le bit
// ou commence la trame ne depend QUE de la largeur de sa charge. La profondeur de trame
// mesure donc EXACTEMENT la justesse de cette largeur, et rien d'autre.
//
// SEUILS ECRITS AVANT LA MESURE :
//   - un type est JUSTE si la profondeur de ses listes a un seul evenement atteint au moins
//     70 % de la profondeur mediane des types mesures, avec >= 30 trames ;
//   - un type est FAUX si elle tombe sous 30 % de cette mediane ;
//   - entre les deux : DOUTEUX.
// Le temoin negatif est fourni par construction : le meme decodage decale de +3 bits.
//
// LECTURE SEULE, skip par defaut, CGO_ENABLED=0.
//
//	CGO_ENABLED=0 R7_ROOT=... R7_CAT=... R7_MAPS=... R7_IDS=... \
//	  go test ./internal/analysis/filmdec/ -run '^TestR7Largeur$' -count=1 -timeout 120m -v

import (
	"path/filepath"
	"sort"
	"testing"
)

// r7MesureLargeurs rejoue la mesure « listes a un seul evenement » et rend, par type, la
// profondeur de trame au bon cadrage et au temoin +3 bits.
func r7MesureLargeurs(t *testing.T, root string, ids []string,
	cartes map[string]r7Ctx) (map[int]*r7TrameStat, map[int]*r7TrameStat) {
	t.Helper()
	juste := map[int]*r7TrameStat{}
	temoin := map[int]*r7TrameStat{}
	for _, id := range ids {
		reg, chunks, err := r7Chargements(filepath.Join(root, id))
		if err != nil || len(chunks) == 0 {
			t.Logf("film %s : illisible (%v) — ignore", id, err)
			continue
		}
		cfg := DefaultFrameConfig()
		cfg.IDLowBits, _ = r7CalibreIDLow(reg, chunks)
		ctx := cartes[id]
		for _, data := range chunks {
			wBase := NewWorld(reg)
			pks := WalkPackets(data)
			for _, pk := range pks {
				if pk.Type != PacketTypeKeyframe {
					continue
				}
				for _, r := range WalkKeyframeWorld(pk.Payload(data)) {
					wBase.BindFull(uint32((r.Gen<<30)|r.Slot), uint32(r.TI))
				}
			}
			for _, pk := range pks {
				if pk.Type != PacketTypeDelta || pk.Size < 1 {
					continue
				}
				if pay := pk.Payload(data); pay[0]&0x40 == 0 {
					_, _ = DecodeFrameRecords(NewBitReader(pay), wBase, cfg)
				}
			}
			snap := wBase.Snapshot()
			for _, pk := range pks {
				if pk.Type != PacketTypeDelta || pk.Size < 2 {
					continue
				}
				pay := pk.Payload(data)
				if pay[0]&0x40 == 0 {
					continue
				}
				evs, stop, _, fin := r7Marche(pay, ctx)
				if stop != r7StopFin || len(evs) != 1 {
					continue // SEULES les listes a un evenement isolent la largeur
				}
				typ := evs[0].Typ
				if juste[typ] == nil {
					juste[typ], temoin[typ] = &r7TrameStat{}, &r7TrameStat{}
				}
				r7Juge(reg, snap, pay, fin, cfg, juste[typ])
				r7Juge(reg, snap, pay, fin+3, cfg, temoin[typ])
			}
		}
	}
	return juste, temoin
}

// TestR7Largeur juge la largeur de charge de chaque type sur les listes a UN SEUL evenement.
func TestR7Largeur(t *testing.T) {
	root, ids := r7Films(t)
	cartes := r7Cartes(t)
	release := LockProcessDecode()
	defer release()
	juste, temoin := r7MesureLargeurs(t, root, ids, cartes)
	// Mediane des profondeurs des types suffisamment observes : la reference du seuil.
	var profs []float64
	for typ, s := range juste {
		if s.paquets >= 30 {
			profs = append(profs, s.profondeur())
			_ = typ
		}
	}
	sort.Float64s(profs)
	ref := 0.0
	if len(profs) > 0 {
		ref = profs[len(profs)/2]
	}
	var types []int
	for typ := range juste {
		types = append(types, typ)
	}
	sort.Slice(types, func(i, j int) bool { return juste[types[i]].paquets > juste[types[j]].paquets })
	t.Logf("mediane de reference : %.3f records/paquet (%d types a >= 30 trames)", ref, len(profs))
	t.Logf("")
	t.Logf("%-4s %-38s %7s %11s %11s %10s", "type", "nom", "trames", "profondeur", "temoin+3", "verdict")
	for _, typ := range types {
		s, tm := juste[typ], temoin[typ]
		p := s.profondeur()
		verdict := "DOUTEUX"
		switch {
		case s.paquets < 30:
			verdict = "trop peu"
		case p >= 0.7*ref:
			verdict = "JUSTE"
		case p < 0.3*ref:
			verdict = "FAUX"
		}
		t.Logf("%-4d %-38s %7d %11.3f %11.3f %10s", typ, r7Noms[typ], s.paquets, p,
			tm.profondeur(), verdict)
	}
}

// r7T5Candidats : les largeurs candidates des deux `FUN_14076dc04` du type 5. Le jeu vient
// des immediats reellement presents dans le corps de la fonction (0x8, 0xF) elargi aux
// largeurs de direction usuelles du moteur (10, 19, 24) — jamais un balayage libre.
var r7T5Candidats = []int{8, 10, 15, 19, 24}

// TestR7CalibreType5 cherche le couple (r7T5DirA, r7T5DirB) qui rend la largeur du type 5
// juste, mesuree sur les listes a UN SEUL evenement de type 5. SEUIL ECRIT D'AVANCE : le
// couple retenu doit atteindre au moins 1,4 record/paquet (le niveau des types JUSTE) et
// depasser le deuxieme meilleur d'au moins 30 %.
func TestR7CalibreType5(t *testing.T) {
	root, ids := r7Films(t)
	cartes := r7Cartes(t)
	release := LockProcessDecode()
	defer release()
	origA, origB := r7T5DirA, r7T5DirB
	defer func() { r7T5DirA, r7T5DirB = origA, origB }()
	type res struct {
		a, b int
		prof float64
		n    int
	}
	var tous []res
	for _, a := range r7T5Candidats {
		for _, b := range r7T5Candidats {
			r7T5DirA, r7T5DirB = a, b
			juste, _ := r7MesureLargeurs(t, root, ids, cartes)
			s := juste[5]
			if s == nil {
				continue
			}
			tous = append(tous, res{a, b, s.profondeur(), s.paquets})
		}
	}
	sort.Slice(tous, func(i, j int) bool { return tous[i].prof > tous[j].prof })
	for i, r := range tous {
		if i >= 8 {
			break
		}
		t.Logf("  A=%2d B=%2d : profondeur %.3f (n=%d)", r.a, r.b, r.prof, r.n)
	}
	if len(tous) < 2 {
		t.Logf("VERDICT : pas assez de combinaisons mesurees")
		return
	}
	best, second := tous[0], tous[1]
	if best.prof >= 1.4 && best.prof >= 1.3*second.prof {
		t.Logf("VERDICT : couple (%d, %d) retenu — profondeur %.3f contre %.3f au deuxieme",
			best.a, best.b, best.prof, second.prof)
	} else {
		t.Logf("VERDICT : NON CONCLUANT — meilleur %.3f (A=%d B=%d), deuxieme %.3f : le type 5 "+
			"reste FAUX et hors de la liste des largeurs validees",
			best.prof, best.a, best.b, second.prof)
	}
}
