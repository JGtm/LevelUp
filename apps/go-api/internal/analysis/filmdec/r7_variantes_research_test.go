package filmdec

// r7_variantes_research_test.go — lot R7 : CALIBRATION SUR PIECES DES VARIANTES DE BUILD.
//
// Trois champs de la grammaire sont gardes par des globales du moteur ecrites au demarrage
// (`FUN_1404f25f4` pour le prefixe du type 15, une garde de session pour la queue du type 85,
// `FUN_14076f91c` pour le tag 7 du type 82). Elles valent zero dans l'image du binaire :
// STATIQUEMENT ON NE PEUT PAS TRANCHER. Plutot que de deviner, on TRANCHE SUR PIECES avec
// l'oracle de trame, restreint aux listes qui contiennent le type concerne : la variante
// juste fait aller la trame LOIN, la fausse la fait desynchroniser tout de suite.
//
// SEUIL ECRIT AVANT LA MESURE : on retient une variante si sa profondeur de trame depasse
// celle de l'autre d'un facteur >= 2 ; sinon on declare la mesure NON CONCLUANTE et le type
// reste sous reserve.
//
// LECTURE SEULE, skip par defaut, CGO_ENABLED=0.

import (
	"fmt"
	"path/filepath"
	"testing"
)

// r7ContientType rend une garde de filtrage : la liste contient au moins un evenement du type.
func r7ContientType(typ int) func([]r7Ev) bool {
	return func(evs []r7Ev) bool {
		for _, e := range evs {
			if e.Typ == typ {
				return true
			}
		}
		return false
	}
}

// r7VarianteCas decrit une variante binaire a calibrer.
type r7VarianteCas struct {
	nom     string
	typ     int
	drapeau *bool
	libelle [2]string // [false, true]
}

// TestR7Variantes calibre les variantes de build sur l'oracle de trame.
func TestR7Variantes(t *testing.T) {
	root, ids := r7Films(t)
	cartes := r7Cartes(t)
	release := LockProcessDecode()
	defer release()
	cas := []r7VarianteCas{
		{"type 15 Script — prefixe R(15)", 15, &r7Var15Prefixe,
			[2]string{"sans prefixe", "avec prefixe R(15)"}},
		{"type 85 PlayerKilledEvent — queue 68 bits", 85, &r7Var85Queue,
			[2]string{"sans queue", "avec queue R(32)+R(32)+R(4)"}},
		{"type 82 tag 7 — R(96) brut vs vecteur quantifie", 82, &r7Var82Tag7Brut,
			[2]string{"vecteur quantifie", "R(96) brut"}},
	}
	type charge struct {
		reg    *Registry
		chunks [][]byte
		cfg    FrameConfig
		ctx    r7Ctx
		id     string
	}
	var films []charge
	for _, id := range ids {
		dir := filepath.Join(root, id)
		reg, chunks, err := r7Chargements(dir)
		if err != nil || len(chunks) == 0 {
			t.Logf("film %s : illisible (%v) — ignore", id, err)
			continue
		}
		cfg := DefaultFrameConfig()
		cfg.IDLowBits, _ = r7CalibreIDLow(reg, chunks)
		films = append(films, charge{reg, chunks, cfg, cartes[id], id})
	}
	for _, c := range cas {
		t.Logf("")
		t.Logf("######## %s ########", c.nom)
		orig := *c.drapeau
		var stats [2]r7TrameStat
		for i, v := range []bool{false, true} {
			*c.drapeau = v
			for _, f := range films {
				st, _ := r7OracleFilm(f.reg, f.chunks, f.ctx, f.cfg, r7ContientType(c.typ), 0)
				stats[i].cumule(st)
			}
			r7RapportTrame(t, "  "+c.libelle[i]+" ", stats[i])
		}
		*c.drapeau = orig
		a, b := stats[0].profondeur(), stats[1].profondeur()
		switch {
		case stats[0].paquets == 0 && stats[1].paquets == 0:
			t.Logf("  VERDICT : aucune liste contenant le type %d n'a ete marchee — NON CONCLUANT", c.typ)
		case a >= 2*b && a > 0:
			t.Logf("  VERDICT : %q retenu (profondeur %.3f contre %.3f, facteur %.2f)",
				c.libelle[0], a, b, a/maxNonNul(b))
		case b >= 2*a && b > 0:
			t.Logf("  VERDICT : %q retenu (profondeur %.3f contre %.3f, facteur %.2f)",
				c.libelle[1], b, a, b/maxNonNul(a))
		default:
			t.Logf("  VERDICT : NON CONCLUANT (profondeurs %.3f et %.3f, facteur < 2) — le type reste sous reserve",
				a, b)
		}
	}
}

func maxNonNul(v float64) float64 {
	if v < 0.0001 {
		return 0.0001
	}
	return v
}

// TestR7CalibreCarte identifie, film par film, la SOMME des largeurs d'axe de la carte par
// l'oracle de trame. Candidats : les sommes distinctes du catalogue de production (15 sur 79
// cartes reelles). Mesure restreinte aux listes qui contiennent un vecteur quantifie (types
// 117 ou 82), les seules ou la valeur change quelque chose.
func TestR7CalibreCarte(t *testing.T) {
	root, ids := r7Films(t)
	profils := r7ProfilsCarte(t)
	if len(profils) == 0 {
		t.Skipf("definir %s (catalogue) pour la calibration de carte", r7CatEnv)
	}
	release := LockProcessDecode()
	defer release()
	t.Logf("%d profils de carte distincts au catalogue", len(profils))
	garde := func(evs []r7Ev) bool {
		for _, e := range evs {
			if e.Typ == 117 || e.Typ == 82 {
				return true
			}
		}
		return false
	}
	for _, id := range ids {
		reg, chunks, err := r7Chargements(filepath.Join(root, id))
		if err != nil || len(chunks) == 0 {
			t.Logf("film %s : illisible (%v) — ignore", id, err)
			continue
		}
		cfg := DefaultFrameConfig()
		cfg.IDLowBits, _ = r7CalibreIDLow(reg, chunks)
		best := ""
		bestProf, second := -1.0, -1.0
		var detail []string
		for _, p := range profils {
			st, _ := r7OracleFilm(reg, chunks, p.Ctx, cfg, garde, 0)
			prof := st.profondeur()
			detail = append(detail, fmt.Sprintf("%s:%.2f(n=%d)", p.Nom, prof, st.paquets))
			if prof > bestProf {
				second, bestProf, best = bestProf, prof, p.Nom
			} else if prof > second {
				second = prof
			}
		}
		t.Logf("film %s : carte retenue %s (profondeur %.3f, 2e meilleure %.3f)",
			id, best, bestProf, second)
		t.Logf("  detail : %v", detail)
	}
	t.Logf("Reporter les cartes retenues dans R7_MAPS=\"id8=nomCarte,...\".")
}

// TestR7Tag7 recense les etiquettes d'union reellement rencontrees dans les sacs de
// proprietes du type 82. Objet : dire si la branche « tag 7 » — la seule que l'exe ne ferme
// pas statiquement — est empruntee dans le parc, ou si la reserve est theorique.
func TestR7Tag7(t *testing.T) {
	root, ids := r7Films(t)
	cartes := r7Cartes(t)
	r7CompteTags = true
	defer func() {
		r7CompteTags = false
		r7TagsA, r7TagsB = map[uint64]int{}, map[uint64]int{}
	}()
	sacs, evts82 := 0, 0
	for _, id := range ids {
		dir := filepath.Join(root, id)
		ctx := cartes[id]
		for c, n := 1, r7Chunks(dir); c <= n; c++ {
			data, err := ReadFilmChunk(dir, c)
			if err != nil {
				continue
			}
			for _, pk := range WalkPackets(data) {
				if pk.Type != PacketTypeDelta || pk.Size < 2 {
					continue
				}
				pay := pk.Payload(data)
				if pay[0]&0x40 == 0 {
					continue
				}
				evs, _, _, _ := r7Marche(pay, ctx)
				for _, e := range evs {
					if e.Typ == 82 {
						evts82++
					}
				}
				sacs++
			}
		}
	}
	t.Logf("=== %d listes marchees · %d evenements de type 82 traverses ===", sacs, evts82)
	t.Logf("etiquettes du SAC PRINCIPAL (union 0x14080eff0) : %s", r7CompteTag(r7TagsA))
	t.Logf("etiquettes du SOUS-SAC (union 0x1407f0ebc)      : %s", r7CompteTag(r7TagsB))
	if r7TagsA[7] == 0 {
		t.Logf("VERDICT : le tag 7 (seule branche non fermee statiquement) N'APPARAIT PAS — "+
			"la reserve est theorique sur ce parc (%d valeurs lues)", sommeMapU(r7TagsA))
	} else {
		t.Logf("VERDICT : le tag 7 apparait %d fois — la reserve est REELLE, il faut trancher "+
			"R(96) brut contre vecteur quantifie", r7TagsA[7])
	}
}

func r7CompteTag(m map[uint64]int) string {
	out := ""
	for tag := uint64(0); tag < 8; tag++ {
		if out != "" {
			out += " · "
		}
		out += fmt.Sprintf("%d:%d", tag, m[tag])
	}
	return out
}

func sommeMapU(m map[uint64]int) int {
	s := 0
	for _, v := range m {
		s += v
	}
	return s
}
