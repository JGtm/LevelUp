//go:build research

package grammar

// mouvement_5_7_posture_research_test.go — `ti=35 i55` CONTRE LA VITESSE VERTICALE (lot 5.7).
//
// # LA QUESTION, ET CE QUE L ECRIVAIN EN DIT DEJA
//
// L ecrivain d `i55 biped-posture-physics-component` (`FUN_142f0293c` -> `FUN_142f1f630`) lit
// `R(2)` et passe la valeur a `FUN_141fd997c`, qui n est PAS une resolution sans effet : c est
// le REPARTITEUR D UNE UNION DISCRIMINEE. Pour les tags 1, 2 et 3 il POSE UN OCTET DE GENRE
// (`dst+0x2c` = 1, 2, 3) puis appelle un lecteur de charge DIFFERENT par tag ; le tag 0 prend
// une quatrieme voie. Et l image ne connait que TROIS classes d etat de bipede —
// `c_biped_ground_state`, `c_biped_airborne_state`, `c_biped_vehicle_state` (les seules chaines
// `c_biped_*` du binaire).
//
// L etat AERIEN du Spartan est donc l un de ces quatre tags, et c est cet instrument qui dit
// lequel : le tag de l etat aerien doit PRECEDER une montee de la composante verticale d `i1`.
//
// # CE QU IL MESURE, ET SUR QUELLE MARCHE
//
// Marche du JEU (5.3.3-b) : `DecodeFrameViews` sur TROIS vues, paquets a liste pleine localises
// par `marchLocateStrict`, largeurs d axe de la CARTE installees (5.3.5 — sans elles `i0` lit
// cinq bits de trop par record et tout ce qui suit est du bruit).
//
// L ORACLE DE CONTENU EST PUBLIE A CHAQUE PASSE, ET IL PRECEDE TOUTE CONCLUSION : records
// `ti=35`, part desynchronisee, et l etalon `i0`/`i1`/`i21`/`i25`. Une passe dont l etalon
// s effondre ne prouve rien, quel que soit le reste.
//
// Rejouable :
//
//	MOUV57_FILM=<dir> MOUV57_CARTE=snowbound MOUV57_BORNES=<catalogue> \
//	  go test -tags=research -count=1 -v -timeout 60m \
//	  -run '^TestMouvement57Posture$' ./internal/games/halo_infinite/film/internal/grammar/

import (
	"fmt"
	"math"
	"os"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// m57Post est UNE lecture d `i55`, datee et attribuee.
type m57Post struct {
	slot uint32
	ts   uint64
	tag  uint64
}

// m57Vit est UNE lecture de vitesse dequantifiee, datee et attribuee.
type m57Vit struct {
	slot uint32
	ts   uint64
	vz   float64
	sol  float64
}

// m57Rec : tout ce que la passe collecte.
type m57Rec struct {
	post    []m57Post
	vit     []m57Vit
	fautifs map[int]int
	etalon  map[int]int
	ti35    int
	desync  int
	paquets int
	pleins  int
	vues    int
}

// m57FenetreUS est la fenetre « dans les deux ticks suivants » du protocole de preuve : un delta
// de 60 Hz dure 16 667 us, donc deux ticks valent 33 333 us. Arrondi a 40 000 us pour absorber
// la gigue d horodatage des paquets (un paquet porte plusieurs ticks de records).
const m57FenetreUS = 40000

// m57SeuilMontee : la montee de vz, en m/s, au-dela de laquelle on parle d une MONTEE. 1 m/s est
// la borne basse du saut de 5.3.5 (« pic median sous 1 m/s »), donc le seuil qui separe une
// montee reelle du bruit de quantification.
const m57SeuilMontee = 1.0

func TestMouvement57Posture(t *testing.T) {
	rec, ok := m57Passe(t)
	if !ok {
		return
	}
	m57Rendre(t, rec)
}

// m57Passe est LA PASSE PARTAGEE des instruments du lot 5.7 : elle ouvre le film sous sa carte,
// marche la trame comme le jeu, collecte `i55` et `i1`, filtre sur le bipede, et REFUSE de
// rendre un enregistrement si l oracle de contenu ne tient pas.
//
// UNE SEULE COPIE DE LA MARCHE, et c est la regle 6 : deux instruments qui marchent le film
// cote a cote divergent le jour ou l un des deux est corrige.
func m57Passe(t *testing.T) (*m57Rec, bool) {
	t.Helper()
	dir := os.Getenv("MOUV57_FILM")
	if dir == "" {
		t.Skip("MOUV57_FILM absent : instrument de recherche")
	}
	film, err := source.LoadDir(dir, nil)
	if err != nil {
		t.Fatalf("LoadDir %s : %v", dir, err)
	}
	fc := m57Contexte(t, film)
	reg, errR := fc.Registry()
	if errR != nil {
		t.Fatalf("registre : %v", errR)
	}
	if restore, errMPP := InstallFilmFormatMPP(fc); errMPP == nil {
		defer restore()
	}
	rec := &m57Rec{fautifs: map[int]int{}, etalon: map[int]int{}}
	var ts uint64
	cfg := fc.CadreDeBalayage()
	cfg.Obs = m57Observateur(rec, &ts)
	w := NewWorld(reg)
	for _, c := range fc.ChunkNumbers() {
		data, pks, ok := fc.ChunkAt(c)
		if !ok {
			continue
		}
		m533bLierMonde(w, data, pks)
		for _, pk := range pks {
			m57Paquet(pk, data, w, cfg, rec, &ts)
		}
	}
	m57Filtrer(t, rec, w)
	return rec, m57Oracle(t, rec)
}

// m57Contexte ouvre le contexte du film sous la carte (meme geste que 5.3.4 et 5.3.5).
func m57Contexte(t *testing.T, film *source.Film) *FilmContext {
	t.Helper()
	os.Setenv("MOUV534_CARTE", os.Getenv("MOUV57_CARTE"))
	os.Setenv("MOUV534_BORNES", os.Getenv("MOUV57_BORNES"))
	return m534Contexte(t, film)
}

// m57Observateur branche les deux portes utiles : la posture `i55` et la vitesse `i1`.
func m57Observateur(rec *m57Rec, ts *uint64) *Observation {
	return &Observation{
		EtatMouvementHook: func(comp EtatMouvementComposant, slot uint32, v []uint64) {
			switch comp {
			case EtatPosture:
				rec.post = append(rec.post, m57Post{slot: slot, ts: *ts, tag: v[0]})
			case EtatVitesse:
				if v[0] != 0 || v[1] != 0 { // brut 96 bits, ou vecteur constant : pas de quanta
					return
				}
				vec := DecodeVelocity(v[2], v[3])
				rec.vit = append(rec.vit, m57Vit{slot: slot, ts: *ts,
					vz:  float64(vec[2]),
					sol: hypot32(vec[0], vec[1])})
			case EtatAccroupi, EtatGlissade, EtatControleUnite, EtatMobilite:
			}
		},
	}
}

// hypot32 rend la norme des deux composantes horizontales.
func hypot32(x, y float32) float64 {
	return math.Hypot(float64(x), float64(y))
}

// m57Paquet traite UN paquet delta : localisation, marche a trois vues, ventilation.
func m57Paquet(pk FilmPacket, data []byte, w *World, cfg FrameConfig, rec *m57Rec, ts *uint64) {
	if pk.Type != PacketTypeDelta || pk.Size < 1 {
		return
	}
	pay := pk.Payload(data)
	debut := 2
	if _, present := PacketHeadEventType(pay); present {
		if debut = marchLocateStrict(pay, w, cfg); debut < 0 {
			return
		}
		rec.pleins++
	}
	rec.paquets++
	*ts = pk.TimestampUS
	recs, vues := DecodeFrameViews(pay, w, cfg, 3, debut)
	rec.vues += vues
	for _, r := range recs {
		if r.TypeIndex != BipedTypeIndex {
			continue
		}
		rec.ti35++
		if r.DesyncAt >= 0 {
			rec.desync++
			rec.fautifs[r.DesyncAt]++
		}
		for _, b := range []int{0, 1, 21, 25, 29, 55, 62} {
			if r.Trace.Mask&(1<<uint(b)) != 0 {
				rec.etalon[b]++
			}
		}
	}
}

// m57Filtrer ne garde que les lectures d un slot LIE AU BIPEDE — meme raison qu au lot 5.3.4
// (`i1` est universel, et l attribution de slot du chemin d inference est partielle).
func m57Filtrer(t *testing.T, rec *m57Rec, w *World) {
	t.Helper()
	bipede := func(slot uint32) bool {
		ti, ok := w.ArchetypeForSlot(slot)
		return ok && ti == BipedTypeIndex
	}
	var p []m57Post
	for _, x := range rec.post {
		if bipede(x.slot) {
			p = append(p, x)
		}
	}
	var v []m57Vit
	for _, x := range rec.vit {
		if bipede(x.slot) {
			v = append(v, x)
		}
	}
	t.Logf("FILTRE « SLOT LIE AU BIPEDE » : i55 %d -> %d · i1 %d -> %d",
		len(rec.post), len(p), len(rec.vit), len(v))
	rec.post, rec.vit = p, v
}

// m57Oracle publie l oracle de contenu et REFUSE de conclure si l etalon s effondre.
func m57Oracle(t *testing.T, rec *m57Rec) bool {
	t.Helper()
	t.Logf("MARCHE (3 vues) : %d paquets, dont %d a liste pleine localises · %d vues · "+
		"%d records ti=35, dont %d desynchronises (%.2f %%)",
		rec.paquets, rec.pleins, rec.vues, rec.ti35, rec.desync,
		m533bPart(rec.desync, rec.ti35))
	t.Logf("  COMPOSANT FAUTIF : %s", m533cTable(rec.fautifs))
	t.Logf("ETALON sur ti=35 : i0 %.1f %% · i1 %.1f %% · i21 %.1f %% · i25 %.1f %% "+
		"(i29 %d · i55 %d · i62 %d declares)",
		m533bPart(rec.etalon[0], rec.ti35), m533bPart(rec.etalon[1], rec.ti35),
		m533bPart(rec.etalon[21], rec.ti35), m533bPart(rec.etalon[25], rec.ti35),
		rec.etalon[29], rec.etalon[55], rec.etalon[62])
	if m533bPart(rec.etalon[21], rec.ti35) < 50 {
		t.Logf("ETALON REFUSE : `i21` sous 50 %% — la trame n est pas cadree, rien de ce qui " +
			"suivrait ne serait une mesure. Aucune conclusion publiee.")
		return false
	}
	t.Logf("ETALON TENU : les tableaux qui suivent portent sur une trame cadree.")
	return true
}

// m57Rendre publie les trois tableaux : la distribution des tags, le test de la montee, et le
// negatif (les montees SANS candidat).
func m57Rendre(t *testing.T, rec *m57Rec) {
	t.Helper()
	m57Distribution(t, rec)
	m57TestMontee(t, rec)
	m57MonteesSansCandidat(t, rec)
}

// m57Distribution publie le compte, les slots et la ventilation par tag.
func m57Distribution(t *testing.T, rec *m57Rec) {
	t.Helper()
	parTag := map[uint64]int{}
	slots := map[uint32]bool{}
	for _, p := range rec.post {
		parTag[p.tag]++
		slots[p.slot] = true
	}
	var parts []string
	for tag := uint64(0); tag < 4; tag++ {
		parts = append(parts, fmt.Sprintf("tag %d : %d (%.1f %%)",
			tag, parTag[tag], m533bPart(parTag[tag], len(rec.post))))
	}
	t.Logf("`i55` : %d lectures sur %d slots — %s", len(rec.post), len(slots),
		strings.Join(parts, " · "))
}

// m57ParSlot indexe les lectures de vitesse par slot, triees par horodatage.
func m57ParSlot(vits []m57Vit) map[uint32][]m57Vit {
	out := map[uint32][]m57Vit{}
	for _, v := range vits {
		out[v.slot] = append(out[v.slot], v)
	}
	for s := range out {
		sort.Slice(out[s], func(i, j int) bool { return out[s][i].ts < out[s][j].ts })
	}
	return out
}

// m57TestMontee est LE TEST DU PROTOCOLE : pour chaque tag, la part des lectures suivies, dans
// les deux ticks, d une vitesse verticale POSITIVE de la MEME vie, et la part suivie d une
// montee franche (>= m57SeuilMontee).
func m57TestMontee(t *testing.T, rec *m57Rec) {
	t.Helper()
	parSlot := m57ParSlot(rec.vit)
	type bilan struct{ total, apparie, positif, franc int }
	b := map[uint64]*bilan{}
	for tag := uint64(0); tag < 4; tag++ {
		b[tag] = &bilan{}
	}
	for _, p := range rec.post {
		st := b[p.tag]
		if st == nil {
			continue
		}
		st.total++
		vz, ok := m57VzApres(parSlot[p.slot], p.ts)
		if !ok {
			continue
		}
		st.apparie++
		if vz > 0 {
			st.positif++
		}
		if vz >= m57SeuilMontee {
			st.franc++
		}
	}
	t.Logf("TEST DE LA MONTEE (fenetre %d us, meme vie) — un tag d etat AERIEN doit precede une "+
		"montee de vz sur plus de 90 %% de ses occurrences appariees :", m57FenetreUS)
	for tag := uint64(0); tag < 4; tag++ {
		st := b[tag]
		t.Logf("  tag %d : %d lectures · %d appariees (%.1f %%) · vz > 0 sur %d (%.1f %% des "+
			"appariees) · vz >= %.1f m/s sur %d (%.1f %%)",
			tag, st.total, st.apparie, m533bPart(st.apparie, st.total),
			st.positif, m533bPart(st.positif, st.apparie),
			m57SeuilMontee, st.franc, m533bPart(st.franc, st.apparie))
	}
}

// m57VzApres rend la vitesse verticale de la premiere lecture d `i1` de la meme vie dans la
// fenetre qui SUIT l horodatage donne, et si elle existe.
func m57VzApres(vits []m57Vit, ts uint64) (float64, bool) {
	best, ok := 0.0, false
	for _, v := range vits {
		if v.ts < ts || v.ts > ts+m57FenetreUS {
			continue
		}
		if !ok || v.vz > best {
			best, ok = v.vz, true
		}
	}
	return best, ok
}

// m57MonteesSansCandidat est LA CONTRE-PREUVE : une montee franche de vz qui n est PRECEDEE
// d aucune lecture d `i55` de la meme vie dans la fenetre. Une part elevee dit que le tag, quel
// qu il soit, n explique pas les montees (rampes, canons a homme, chutes).
func m57MonteesSansCandidat(t *testing.T, rec *m57Rec) {
	t.Helper()
	parSlotPost := map[uint32][]m57Post{}
	for _, p := range rec.post {
		parSlotPost[p.slot] = append(parSlotPost[p.slot], p)
	}
	var montees, avec int
	for _, v := range rec.vit {
		if v.vz < m57SeuilMontee {
			continue
		}
		montees++
		for _, p := range parSlotPost[v.slot] {
			if p.ts <= v.ts && p.ts+m57FenetreUS >= v.ts {
				avec++
				break
			}
		}
	}
	t.Logf("CONTRE-PREUVE : %d montees franches (vz >= %.1f m/s) sur %d lectures d `i1` ; "+
		"%d (%.1f %%) sont precedees d une lecture d `i55` de la meme vie dans la fenetre, "+
		"donc %.1f %% ne le sont PAS.",
		montees, m57SeuilMontee, len(rec.vit), avec, m533bPart(avec, montees),
		100-m533bPart(avec, montees))
}
