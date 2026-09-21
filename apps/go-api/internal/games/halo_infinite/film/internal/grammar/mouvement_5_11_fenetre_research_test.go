//go:build research

package grammar

// mouvement_5_11_fenetre_research_test.go — LA FENETRE DU SAUT, RECORD PAR RECORD (lot 5.11).
//
// Scission par DEPLACEMENT PUR de `mouvement_5_11_temoin_research_test.go` : le seuil de 500
// lignes de CLAUDE.md, et `plafondsParFichier` est datee et fermee (piege 5 de la passation
// 5.3.3). Aucune ligne n est reecrite ici.
//
// Les bornes se donnent en SECONDES depuis le premier paquet delta du film (`MOUV511_T0` /
// `MOUV511_T1`), ce qui laisse la fenetre TEMOIN (sans saut) se demander avec le meme
// instrument.

import (
	"fmt"
	"math"
	"os"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
)

// TestMouvement511Registre2 dumpe le registre du film : combien de composants par archetype, et
// la liste de ceux des archetypes qui portent le gros de la population.
func TestMouvement511Registre2(t *testing.T) {
	film := m511Film(t)
	fc := NewFilmContext(film)
	reg, err := fc.Registry()
	if err != nil {
		t.Fatalf("registre : %v", err)
	}
	t.Logf("REGISTRE : %d archetypes", len(reg.Archetypes))
	for _, idx := range []int{0, 2, 4, 5, 6, 9, 22, 35, 37, 47} {
		a, ok := reg.Archetype(idx)
		if !ok {
			t.Logf("  ti=%d : absent", idx)
			continue
		}
		n := len(a.Components)
		var tete []string
		for i := 0; i < n && i < 6; i++ {
			tete = append(tete, a.Components[i])
		}
		t.Logf("  ti=%2d : %3d composants — %s", idx, n, strings.Join(tete, ", "))
	}
	a, _ := reg.Archetype(BipedTypeIndex)
	for i, c := range a.Components {
		t.Logf("    biped i%-2d %s (niveau %d)", i, c, a.Level(i))
	}
}

// TestMouvement511Bipede publie TOUS les records de bipede du film temoin : instant relatif au
// premier paquet, slot, composants declares, et les bits NON LUS du paquet.
func TestMouvement511Bipede(t *testing.T) {
	film := m511Film(t)
	entry := m511Entree(t)
	fc := m511Contexte(film, entry)
	reg, err := fc.Registry()
	if err != nil {
		t.Fatalf("registre : %v", err)
	}
	if restore, errMPP := InstallFilmFormatMPP(fc); errMPP == nil {
		defer restore()
	}
	cfg := fc.CadreDeBalayage()
	w := NewWorld(reg)
	var t0 uint64
	var n, resteTotal, paquets int
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
			if t0 == 0 {
				t0 = pk.TimestampUS
			}
			pay := pk.Payload(data)
			debut := 2
			if _, present := PacketHeadEventType(pay); present {
				if debut = marchLocateStrict(pay, w, cfg); debut < 0 {
					continue
				}
			}
			recs, _ := DecodeFrameViews(pay, w, cfg, 3, debut)
			paquets++
			if len(recs) > 0 {
				reste := len(pay)*8 - recs[len(recs)-1].Trace.EndBit
				if reste > 0 {
					resteTotal += reste
				}
			}
			for _, r := range recs {
				if r.TypeIndex != BipedTypeIndex {
					continue
				}
				n++
				var noms []string
				for _, cp := range r.Trace.Comps {
					marque := ""
					if !cp.Ported {
						marque = "!"
					}
					noms = append(noms, fmt.Sprintf("i%d%s", cp.Index, marque))
				}
				t.Logf("  t=%9.3f s · slot %5d · type %d · desync %3d · %2d comps : %s",
					float64(pk.TimestampUS-t0)/1e6, r.Slot, r.Type, r.DesyncAt,
					len(r.Trace.Comps), strings.Join(noms, " "))
			}
		}
	}
	t.Logf("TOTAL : %d records de bipede sur %d paquets · %d bits NON LUS en queue de paquet "+
		"(%.1f bits par paquet)", n, paquets, resteTotal, float64(resteTotal)/float64(paquets))
}

// m511Entree rend l entree de catalogue de la carte donnee par l environnement.
func m511Entree(t *testing.T) profile.MapQuantEntry {
	t.Helper()
	chemin, nom := os.Getenv("MOUV511_BORNES"), os.Getenv("MOUV511_CARTE")
	if chemin == "" || nom == "" {
		t.Skip("MOUV511_BORNES / MOUV511_CARTE absents")
	}
	cat, err := profile.LoadMapQuantCatalog(chemin)
	if err != nil {
		t.Fatalf("catalogue : %v", err)
	}
	e, err := cat.Lookup(nom)
	if err != nil {
		t.Fatalf("carte %q : %v", nom, err)
	}
	return e
}

// m511Bits rend, en hexadecimal MSB-first, les `n` bits de `pay` a partir de `at`.
func m511Bits(pay []byte, at, n int) string {
	if n <= 0 {
		return ""
	}
	if fin := len(pay)*8 - at; n > fin {
		n = fin
	}
	if n <= 0 {
		return ""
	}
	br := LecteurSur(pay)
	br.Skip(at)
	var sb strings.Builder
	for reste := n; reste > 0; {
		w := reste
		if w > 32 {
			w = 32
		}
		fmt.Fprintf(&sb, "%0*b ", w, br.ReadBits(uint(w))) //nolint:gosec // largeur bornee a 32
		reste -= w
	}
	return strings.TrimSpace(sb.String())
}

// TestMouvement511Fenetre est LA MESURE DU LOT : tout ce qui se lit du bipede dans une fenetre de
// temps, record par record et composant par composant — valeurs publiees ET bits bruts, y compris
// pour les composants que le port SAUTE — plus les evenements de tete des paquets de la fenetre.
//
// Les bornes se donnent en SECONDES depuis le premier paquet delta du film
// (`MOUV511_T0` / `MOUV511_T1`), ce qui laisse le temoin (une fenetre sans saut) se demander avec
// le meme instrument.
func TestMouvement511Fenetre(t *testing.T) {
	film := m511Film(t)
	entry := m511Entree(t)
	fc := m511Contexte(film, entry)
	reg, err := fc.Registry()
	if err != nil {
		t.Fatalf("registre : %v", err)
	}
	if restore, errMPP := InstallFilmFormatMPP(fc); errMPP == nil {
		defer restore()
	}
	borneA, borneB := m511Bornes()
	cfg := fc.CadreDeBalayage()
	var ts uint64
	var pos []PositionSample
	var etats []string
	obs := NouvelleObservation()
	obs.PosCaptureHook = func(s PositionSample) { pos = append(pos, s) }
	obs.EtatMouvementHook = func(c EtatMouvementComposant, slot uint32, v []uint64) {
		etats = append(etats, fmt.Sprintf("%s slot %d %v", c.String(), slot, v))
	}
	cfg.Obs = obs
	w := NewWorld(reg)
	var t0 uint64
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
			if t0 == 0 {
				t0 = pk.TimestampUS
			}
			ts = pk.TimestampUS
			rel := float64(pk.TimestampUS-t0) / 1e6
			pay := pk.Payload(data)
			debut := 2
			typ, present := PacketHeadEventType(pay)
			if present {
				if debut = marchLocateStrict(pay, w, cfg); debut < 0 {
					if rel >= borneA && rel <= borneB {
						t.Logf("t=%8.3f s · EVENEMENT DE TETE type %d · NON LOCALISE · tete %s",
							rel, typ, m511Bits(pay, 0, 96))
					}
					continue
				}
				if rel >= borneA && rel <= borneB {
					t.Logf("t=%8.3f s · EVENEMENT DE TETE type %d · corps localise au bit %d · "+
						"liste %s", rel, typ, debut, m511Bits(pay, 9, debut-9))
				}
			}
			pos, etats = pos[:0], etats[:0]
			recs, _ := DecodeFrameViews(pay, w, cfg, 3, debut)
			if rel < borneA || rel > borneB {
				continue
			}
			for _, r := range recs {
				if os.Getenv("MOUV511_TOUS") == "" && r.TypeIndex != BipedTypeIndex {
					continue
				}
				m511Record(t, pay, r, rel)
			}
			m511Sondes(t, rel, pos, etats)
		}
	}
	_ = ts
}

// m511Bornes rend la fenetre de temps demandee, en secondes depuis le premier paquet delta.
func m511Bornes() (float64, float64) {
	a, b := 0.0, 1e9
	if v := os.Getenv("MOUV511_T0"); v != "" {
		fmt.Sscanf(v, "%g", &a)
	}
	if v := os.Getenv("MOUV511_T1"); v != "" {
		fmt.Sscanf(v, "%g", &b)
	}
	return a, b
}

// m511Record publie UN record de bipede : ses composants, leurs largeurs, et leurs BITS BRUTS.
func m511Record(t *testing.T, pay []byte, r FrameRecord, rel float64) {
	t.Helper()
	debut := 0
	if len(r.Trace.Comps) > 0 {
		debut = r.Trace.Comps[0].StartBit
	}
	t.Logf("t=%8.3f s · RECORD ti=%d slot %d type %d desync %d · %d composants · bits %d..%d",
		rel, r.TypeIndex, r.Slot, r.Type, r.DesyncAt, len(r.Trace.Comps), debut,
		r.Trace.EndBit)
	comps := r.Trace.Comps
	for i, c := range comps {
		fin := r.Trace.EndBit
		if i+1 < len(comps) {
			fin = comps[i+1].StartBit
		}
		largeur := fin - c.StartBit
		etat := "porte"
		if !c.Ported {
			etat = "NON PORTE"
		}
		t.Logf("    i%-2d %-52s %-9s %3d bits @%6d : %s", c.Index, c.Name, etat, largeur,
			c.StartBit, m511Bits(pay, c.StartBit, min(largeur, 128)))
	}
}

// m511Sondes publie les valeurs que les portes de publication ont rendues pour ce paquet.
func m511Sondes(t *testing.T, rel float64, pos []PositionSample, etats []string) {
	t.Helper()
	for _, p := range pos {
		t.Logf("    POSITION slot %d genre %v @bit %d : %v", p.Slot, p.Kind, p.BitPos, p.Vec)
	}
	for _, e := range etats {
		t.Logf("    ETAT %s", e)
	}
	_ = rel
}

// TestMouvement511Trajectoire publie Z(t) (depuis `i0`) et vz(t) (depuis `i1`) du bipede actif,
// et la hauteur INTEGREE de chaque montee — la verite terrain physique du 5.7.5 appliquee au
// film temoin.
func TestMouvement511Trajectoire(t *testing.T) {
	film := m511Film(t)
	entry := m511Entree(t)
	fc := m511Contexte(film, entry)
	reg, err := fc.Registry()
	if err != nil {
		t.Fatalf("registre : %v", err)
	}
	if restore, errMPP := InstallFilmFormatMPP(fc); errMPP == nil {
		defer restore()
	}
	type pt struct {
		rel        float64
		z          float64
		vx, vy, vz float64
		aZ, aV     bool
	}
	var pts []pt
	var cur pt
	cfg := fc.CadreDeBalayage()
	obs := NouvelleObservation()
	obs.PosCaptureHook = func(s PositionSample) {
		if s.Slot == 512 {
			cur.z, cur.aZ = float64(s.Vec[2]), true
		}
	}
	obs.EtatMouvementHook = func(c EtatMouvementComposant, slot uint32, v []uint64) {
		if c != EtatVitesse || slot != 512 || len(v) < 4 || v[0] != 0 || v[1] != 0 {
			return
		}
		vec := DecodeVelocity(v[2], v[3])
		cur.vx, cur.vy, cur.vz, cur.aV = float64(vec[0]), float64(vec[1]), float64(vec[2]), true
	}
	cfg.Obs = obs
	w := NewWorld(reg)
	var t0 uint64
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
			if t0 == 0 {
				t0 = pk.TimestampUS
			}
			pay := pk.Payload(data)
			debut := 2
			if _, present := PacketHeadEventType(pay); present {
				if debut = marchLocateStrict(pay, w, cfg); debut < 0 {
					continue
				}
			}
			cur = pt{rel: float64(pk.TimestampUS-t0) / 1e6}
			DecodeFrameViews(pay, w, cfg, 3, debut)
			if cur.aZ || cur.aV {
				pts = append(pts, cur)
			}
		}
	}
	var haut, zMin, zMax float64
	for i, p := range pts {
		if i == 0 || p.z < zMin {
			zMin = p.z
		}
		if i == 0 || p.z > zMax {
			zMax = p.z
		}
		if i+1 < len(pts) && p.vz > 0 {
			d := pts[i+1].rel - p.rel
			if d < 0.25 {
				haut += p.vz * d
			}
		}
		t.Logf("  t=%8.3f s · Z %10.5f · v (%+8.4f, %+8.4f, %+8.4f) m/s · |vsol| %6.3f",
			p.rel, p.z, p.vx, p.vy, p.vz, math.Hypot(p.vx, p.vy))
	}
	t.Logf("BILAN : %d instants · Z de %.5f a %.5f (amplitude %.5f) · integrale des vz positifs "+
		"%.4f m", len(pts), zMin, zMax, zMax-zMin, haut)
}

// m511Cle identifie UN champ replique : l archetype et le composant.
