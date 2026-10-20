//go:build research

package grammar

// mouvement_5_3_5_vitesse_research_test.go — LA LOI DE `i1`, ET SON ORACLE INDEPENDANT (lot 5.3.5).
//
// # CE QUE L ECRIVAIN DIT, RELU A L OCTET (2026-09-21)
//
//	FUN_14076d45c  (l entree d i1)
//	    b = R(1) ; FUN_14076d4d0(lecteur, dst, b*2)
//	FUN_14076d4d0  (le repartiteur, PARTAGE avec i62)
//	    mode 0 (b == 0) : FUN_14076d528(..., min = DAT_143cd88f8, max = DAT_143cd88fc, 10, 0x13)
//	    mode 2 (b == 1) : FUN_1406d676c(..., 0x60)        -> vec3 BRUT de 96 bits
//	FUN_14076d528  (le vec3 a precision dynamique)
//	    g = R(1) ; si g != 0 -> VECTEUR CONSTANT (*PTR_DAT_14474c2f0), AUCUN bit de charge
//	    sinon : dir = R(0x13 = 19) ; FUN_1406d8288(dir, &u, 19)   -> vecteur unitaire cubemap
//	            m   = FUN_14076d6dc(lecteur, ..., min, max, 10)   -> LE SCALAIRE, LU APRES
//	            out = u * m
//	FUN_14076d6dc  (la loi du scalaire)
//	    raw = R(w) ; n = 1 << w
//	    raw == 0    -> min
//	    raw >= n-1  -> max
//	    sinon       -> exp(raw * step + 0.5 * step) - (1 - min),  step = log((1 - min) + max) / n
//
// CONSTANTES RELUES DANS LE BINAIRE (`/read_memory`, float32 little-endian) :
//
//	DAT_143cd88f8 = 8fc2f53c = 0.029999999   (min)
//	DAT_143cd88fc = 0000af43 = 350.0         (max)
//	DAT_143cd8374 = 0000803f = 1.0           (le 1 de « 1 - min »)
//	DAT_143cd84b0 = 0000003f = 0.5           (le demi-pas)
//
// **LE PORT DU DEPOT EST EXACT** : `DecodeVelocityMagnitude` transcrit cette loi terme pour
// terme, `velMagMin`/`velMagMax`/`velScaleBitsDef` valent ces constantes, l ordre direction puis
// scalaire est le bon, et la polarite de la porte l est aussi (bit a 1 = vecteur constant, 0 bit
// de charge). **LA LOI N EST DONC PAS LA CAUSE** des 347 m/s mesures au lot 5.3.4.
//
// # L ORACLE QUE CE FICHIER POSE, ET POURQUOI IL EST INDEPENDANT
//
// Une vitesse decodee se verifie contre le DEPLACEMENT : deux positions successives d une MEME
// vie, divisees par leur ecart de temps. Si le decodage est juste, le RAPPORT
// (deplacement par seconde) / (vitesse decodee) est CONSTANT — c est le facteur d unite entre la
// boite de replication et le metre — et sa dispersion est etroite. S il est du bruit, le rapport
// s etale sur des ordres de grandeur.
//
// L oracle ne suppose AUCUNE unite : c est la DISPERSION du rapport qui tranche, pas sa valeur.
//
// Rejouable :
//
//	MOUV535_FILM=<dir> MOUV535_CARTE=snowbound MOUV535_BORNES=<catalogue> \
//	  go test -tags=research -count=1 -v -timeout 60m \
//	  -run '^TestMouvement535Vitesse$' ./internal/games/halo_infinite/film/internal/grammar/

import (
	"fmt"
	"math"
	"os"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// m535Pos est UN echantillon de position, date et attribue.
type m535Pos struct {
	slot uint32
	ts   uint64
	kind PosKind
	vec  [3]float32
}

// m535Vit est UNE lecture de vitesse dequantifiee, datee et attribuee.
type m535Vit struct {
	slot uint32
	ts   uint64
	v    [3]float32
	sol  float64
	vz   float64
}

// m535Rec : ce que la passe collecte.
type m535Rec struct {
	pos   []m535Pos
	vit   []m535Vit
	modes map[string]int
}

func TestMouvement535Vitesse(t *testing.T) {
	dir := os.Getenv("MOUV535_FILM")
	if dir == "" {
		t.Skip("MOUV535_FILM absent : instrument de recherche")
	}
	film, err := source.LoadDir(dir, nil)
	if err != nil {
		t.Fatalf("LoadDir %s : %v", dir, err)
	}
	fc := m535Contexte(t, film)
	reg, err := fc.Registry()
	if err != nil {
		t.Fatalf("registre : %v", err)
	}
	if restore, errMPP := InstallFilmFormatMPP(fc); errMPP == nil {
		defer restore()
	}
	rec := &m535Rec{modes: map[string]int{}}
	var ts uint64
	cfg := fc.CadreDeBalayage()
	cfg.Obs = m535Observateur(rec, &ts)
	w := NewWorld(reg)
	for _, c := range fc.ChunkNumbers() {
		data, pks, ok := fc.ChunkAt(c)
		if !ok {
			continue
		}
		m533bLierMonde(w, data, pks)
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta || pk.Size < 1 {
				continue
			}
			pay := pk.Payload(data)
			debut := 2
			if _, present := PacketHeadEventType(pay); present {
				if debut = marchLocateStrict(pay, w, cfg); debut < 0 {
					continue
				}
			}
			ts = pk.TimestampUS
			DecodeFrameViews(pay, w, cfg, 3, debut)
		}
	}
	m535Filtrer(t, rec, w)
	m535Rendre(t, rec)
}

// m535Contexte ouvre le contexte du film sous la carte (meme geste que 5.3.4).
func m535Contexte(t *testing.T, film *source.Film) *FilmContext {
	t.Helper()
	os.Setenv("MOUV534_CARTE", os.Getenv("MOUV535_CARTE"))
	os.Setenv("MOUV534_BORNES", os.Getenv("MOUV535_BORNES"))
	return m534Contexte(t, film)
}

// m535Observateur branche les deux portes : les positions `i0` et les vitesses `i1`.
func m535Observateur(rec *m535Rec, ts *uint64) *Observation {
	return &Observation{
		PosCaptureHook: func(s PositionSample) {
			rec.modes["pos:"+s.Kind.String()]++
			rec.pos = append(rec.pos, m535Pos{slot: s.Slot, ts: *ts, kind: s.Kind, vec: s.Vec})
		},
		EtatMouvementHook: func(comp EtatMouvementComposant, slot uint32, v []uint64) {
			if comp != EtatVitesse {
				return
			}
			switch {
			case v[0] != 0:
				rec.modes["vit:brut96"]++
				return
			case v[1] != 0:
				rec.modes["vit:constant"]++
				return
			}
			rec.modes["vit:dequantifiee"]++
			vec := DecodeVelocity(v[2], v[3])
			rec.vit = append(rec.vit, m535Vit{slot: slot, ts: *ts, v: vec,
				sol: math.Hypot(float64(vec[0]), float64(vec[1])), vz: float64(vec[2])})
		},
	}
}

// m535Filtrer ne garde que les lectures d un slot LIE AU BIPEDE (meme raison qu au lot 5.3.4 :
// `i0` et `i1` sont universels, et l attribution du chemin d inference est partielle).
func m535Filtrer(t *testing.T, rec *m535Rec, w *World) {
	t.Helper()
	bipede := func(slot uint32) bool {
		ti, ok := w.ArchetypeForSlot(slot)
		return ok && ti == BipedTypeIndex
	}
	var p []m535Pos
	for _, x := range rec.pos {
		if bipede(x.slot) {
			p = append(p, x)
		}
	}
	var v []m535Vit
	for _, x := range rec.vit {
		if bipede(x.slot) {
			v = append(v, x)
		}
	}
	t.Logf("FILTRE « SLOT LIE AU BIPEDE » : positions %d -> %d · vitesses %d -> %d",
		len(rec.pos), len(p), len(rec.vit), len(v))
	rec.pos, rec.vit = p, v
}

// m535Rendre publie les trois verdicts : le recensement des modes, l oracle, la distribution.
func m535Rendre(t *testing.T, rec *m535Rec) {
	t.Helper()
	t.Logf("MODES RECENSES : %s", m535Modes(rec.modes))
	m535Oracle(t, rec)
	m535Distribution(t, rec)
	m535Impulsions(t, rec)
}

// m535Modes rend le recensement, trie.
func m535Modes(m map[string]int) string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	var out []string
	for _, k := range ks {
		out = append(out, fmt.Sprintf("%s:%d", k, m[k]))
	}
	return strings.Join(out, " · ")
}
