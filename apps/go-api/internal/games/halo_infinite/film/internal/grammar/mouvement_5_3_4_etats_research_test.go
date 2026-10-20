//go:build research

package grammar

// mouvement_5_3_4_etats_research_test.go — LA MESURE FINALE DES ETATS DE MOUVEMENT (lot 5.3.4).
//
// # CE QU IL MESURE, ET SUR QUELLE MARCHE
//
// La marche est celle du JEU, etablie au point 5.3.3-b : `DecodeFrameViews` sur TROIS vues (ce
// que le frame-processor `FUN_142987460` deroule), et les paquets a liste d evenements pleine
// localises par la signature du depot (`marchLocateStrict`). Elle rend 31 530 records `ti=35`
// sur `bfecd02b` contre 11 228 a la marche d une seule vue.
//
// Les valeurs viennent des portes de publication du lot 5.3.4 (`etats_mouvement_hooks.go`) :
// aucun bit n est lu autrement, et l instrument ne decode rien lui-meme.
//
// # LA DOCTRINE, ET CE QU ELLE IMPOSE A LA LECTURE DES CHIFFRES
//
// Ces etats sont a L INSTANT. Un delta ne porte `i29` que quand l accroupi CHANGE : compter des
// lectures, c est compter des TRANSITIONS. Une « cadence » se lit donc en records par seconde et
// par slot, et un « intervalle » est le temps entre deux transitions de la MEME vie.
//
// Rejouable :
//
//	MOUV534_FILM=<dir> MOUV534_CARTE=snowbound MOUV534_BORNES=<catalogue> \
//	  go test -tags=research -count=1 -v -timeout 60m \
//	  -run '^TestMouvement534Etats$' ./internal/games/halo_infinite/film/internal/grammar/

import (
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// m534Ev est UNE lecture d etat, datee et attribuee a un slot.
type m534Ev struct {
	slot uint32
	ts   uint64
	v    []uint64
}

// m534Rec : tout ce que la passe collecte.
type m534Rec struct {
	accroupi, glissade, vitesse []m534Ev
	posture                     map[uint64]int
	postureSlots                map[uint32]bool
	ctrlPorte, ctrlTotal        int
	ctrlF1, ctrlF2              map[uint64]int
	mobil                       []m534Ev // i54 : pas de slot (le hook n en porte pas), date seul
	fautifs                     map[int]int
	ti35, ti35Desync            int
	paquets, vues, pleinsVus    int
	tmin, tmax                  uint64
}

func TestMouvement534Etats(t *testing.T) {
	dir := os.Getenv("MOUV534_FILM")
	if dir == "" {
		t.Skip("MOUV534_FILM absent : instrument de recherche")
	}
	film, err := source.LoadDir(dir, nil)
	if err != nil {
		t.Fatalf("LoadDir %s : %v", dir, err)
	}
	fc := m534Contexte(t, film)
	reg, err := fc.Registry()
	if err != nil {
		t.Fatalf("registre : %v", err)
	}
	if restore, errMPP := InstallFilmFormatMPP(fc); errMPP == nil {
		defer restore()
	}
	rec := &m534Rec{posture: map[uint64]int{}, postureSlots: map[uint32]bool{},
		ctrlF1: map[uint64]int{}, ctrlF2: map[uint64]int{}, fautifs: map[int]int{}}
	var ts uint64
	cfg := fc.CadreDeBalayage()
	cfg.Obs = m534Observateur(rec, &ts)
	w := NewWorld(reg)
	for _, c := range fc.ChunkNumbers() {
		data, pks, ok := fc.ChunkAt(c)
		if !ok {
			continue
		}
		m533bLierMonde(w, data, pks)
		for _, pk := range pks {
			m534Paquet(pk, data, w, cfg, rec, &ts)
		}
	}
	m534Filtrer(t, rec, w)
	m534Rendre(t, rec)
	m534Roster(t, fc)
}

// m534Filtrer NE GARDE QUE LES LECTURES D UN SLOT LIE AU BIPEDE, et dit combien il jette.
//
// DEUX RAISONS, ET LES DEUX SONT MESUREES.
//
//	i1 EST UNIVERSEL. `object-translational-velocity` est lu par TOUS les archetypes, projectiles
//	compris : sans ce filtre la vitesse « au sol » monte a 343 m/s (une balle), et l histogramme
//	ne dit plus rien du Spartan.
//	L ATTRIBUTION DE SLOT EST PARTIELLE SUR LE CHEMIN D INFERENCE. `decodeDelta` pose le slot de
//	capture (`poserSlotDeCapture`), mais `decodeInferLoop` — la boucle que `DecodeFrameViews`
//	emploie — ne le pose PAS pour un record NEW : la lecture herite alors du slot precedent, ou
//	de zero au premier record d un paquet. Ce filtre ecarte ces lectures plutot que de les
//	attribuer a tort. Decouverte consignee au § 4 du plan.
func m534Filtrer(t *testing.T, rec *m534Rec, w *World) {
	t.Helper()
	garde := func(evs []m534Ev) ([]m534Ev, int) {
		out := make([]m534Ev, 0, len(evs))
		var jetees int
		for _, e := range evs {
			if ti, ok := w.ArchetypeForSlot(e.slot); ok && ti == BipedTypeIndex {
				out = append(out, e)
				continue
			}
			jetees++
		}
		return out, jetees
	}
	var jA, jG, jV int
	rec.accroupi, jA = garde(rec.accroupi)
	rec.glissade, jG = garde(rec.glissade)
	rec.vitesse, jV = garde(rec.vitesse)
	t.Logf("FILTRE « SLOT LIE AU BIPEDE » : lectures ecartees — i29 %d · i62 %d · i1 %d "+
		"(slot non lie, lie a un autre archetype, ou herite du record precedent sur le chemin "+
		"d inference)", jA, jG, jV)
}

// m534Vues : TROIS, ce que le frame-processor deroule. `MOUV534_VUES` le surcharge.
func m534Vues() int {
	if v := os.Getenv("MOUV534_VUES"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return 3
}

// m534Contexte ouvre le contexte du film sous la carte.
func m534Contexte(t *testing.T, film *source.Film) *FilmContext {
	t.Helper()
	nom := os.Getenv("MOUV534_CARTE")
	if nom == "" {
		return NewFilmContext(film)
	}
	chemin := os.Getenv("MOUV534_BORNES")
	if chemin == "" {
		t.Fatalf("MOUV534_BORNES attendu avec MOUV534_CARTE")
	}
	cat, err := profile.LoadMapQuantCatalog(chemin)
	if err != nil {
		t.Fatalf("catalogue de bornes : %v", err)
	}
	entry, err := cat.Lookup(nom)
	if err != nil {
		t.Fatalf("carte %q : %v", nom, err)
	}
	t.Logf("CARTE : %s (axes %v)", nom, entry.AxisWidths)
	fc := NewFilmContextForMap(film, &entry, nil)
	// LES LARGEURS D AXE DE LA CARTE S INSTALLENT ICI, ET C EST OBLIGATOIRE — c est le geste que
	// `replay.installWorldObjectPrecision` fait en production juste apres le constructeur.
	// Sans lui, le contexte garde le descripteur PAR DEFAUT, qui est l entree `cliffhanger` du
	// catalogue : sur `snowbound` (15/15/17) le chemin absolu d `i0` lirait ses trois axes aux
	// largeurs 13/13/14, donc CINQ BITS DE TROP PAR RECORD, et tout ce qui suit dans le record
	// est du bruit. L instrument de reference de 5.3.2 ne le faisait pas — cause mesuree au
	// lot 5.3.5.
	bal := fc.ProfilDeBalayage()
	bal.PoserLargeursObjetDuMondeDepuisDecoupage(entry.Layout())
	fc.PoserProfilDeBalayage(bal)
	t.Logf("LARGEURS WORLD-OBJECT INSTALLEES : %v (index %d, region %d)",
		fc.LargeursObjetDuMonde().AxisW, fc.LargeursObjetDuMonde().IndexW,
		fc.LargeursObjetDuMonde().Region)
	return fc
}

// m534Observateur branche les deux hooks utiles. Le pointeur `ts` porte l horodatage du paquet
// en cours : le hook n en a pas, et c est la seule facon de DATER une lecture.
func m534Observateur(rec *m534Rec, ts *uint64) *Observation {
	return &Observation{
		EtatMouvementHook: func(comp EtatMouvementComposant, slot uint32, v []uint64) {
			ev := m534Ev{slot: slot, ts: *ts, v: v}
			switch comp {
			case EtatAccroupi:
				rec.accroupi = append(rec.accroupi, ev)
			case EtatGlissade:
				rec.glissade = append(rec.glissade, ev)
			case EtatVitesse:
				rec.vitesse = append(rec.vitesse, ev)
			case EtatPosture:
				rec.posture[v[0]]++
				rec.postureSlots[slot] = true
			case EtatControleUnite:
				rec.ctrlTotal++
				if v[0] != 0 {
					rec.ctrlPorte++
					rec.ctrlF1[v[1]]++
					if v[2] != 0 {
						rec.ctrlF2[v[3]]++
					}
				}
			}
		},
		MobilityActionHook: func(f1, f2 bool) {
			rec.mobil = append(rec.mobil, m534Ev{ts: *ts, v: []uint64{bit2u(f1), bit2u(f2)}})
		},
	}
}

// m534Paquet traite UN paquet delta : localisation, marche a trois vues, ventilation.
func m534Paquet(pk FilmPacket, data []byte, w *World, cfg FrameConfig, rec *m534Rec, ts *uint64) {
	if pk.Type != PacketTypeDelta || pk.Size < 1 {
		return
	}
	pay := pk.Payload(data)
	debut := 2
	if _, present := PacketHeadEventType(pay); present {
		if debut = marchLocateStrict(pay, w, cfg); debut < 0 {
			return
		}
		rec.pleinsVus++
	}
	rec.paquets++
	*ts = pk.TimestampUS
	if rec.tmin == 0 || pk.TimestampUS < rec.tmin {
		rec.tmin = pk.TimestampUS
	}
	if pk.TimestampUS > rec.tmax {
		rec.tmax = pk.TimestampUS
	}
	recs, vues := DecodeFrameViews(pay, w, cfg, m534Vues(), debut)
	rec.vues += vues
	for _, r := range recs {
		if r.TypeIndex != BipedTypeIndex {
			continue
		}
		rec.ti35++
		if r.DesyncAt >= 0 {
			rec.ti35Desync++
			rec.fautifs[r.DesyncAt]++
		}
	}
}

// m534Roster publie ce que LA TABLE D IDENTITE DU FILM donne, sans aucune base : les gamertags
// du roster. LE JOIN SLOT -> JOUEUR N EST PAS ICI, et c est dit : il demande le registre
// d identite (lien record de creation de bipede -> joueur), qui vit dans `killsource` et dans la
// cuisson du rejeu. Les instants ci-dessus sont donc nommes par SLOT (la vie), pas par joueur.
func m534Roster(t *testing.T, fc *FilmContext) {
	t.Helper()
	reg, ok := FilmRegistryChunk(fc.Film())
	if !ok {
		t.Logf("ROSTER : le film ne porte pas son `chunk_00` — aucune table d identite")
		return
	}
	ident, err := ReadFilmIdentity(reg)
	if err != nil {
		t.Logf("ROSTER : identite du film illisible (%v)", err)
		return
	}
	slots, _, err := ReadPlayerTable(reg, ident)
	if err != nil {
		t.Logf("ROSTER : table des joueurs illisible (%v)", err)
		return
	}
	var noms []string
	for _, s := range slots {
		if s.Gamertag != "" {
			noms = append(noms, s.Gamertag)
		}
	}
	t.Logf("ROSTER LU DANS LE FILM (%d joueurs) : %s", len(noms), strings.Join(noms, ", "))
	t.Logf("  LE JOIN SLOT -> JOUEUR N EST PAS FAIT ICI : il demande le registre d identite " +
		"(creation de bipede -> joueur), qui vit dans `killsource` et dans la cuisson. Les " +
		"instants sont donc nommes par SLOT.")
}

// m534Duree rend la duree couverte, en secondes.
func m534Duree(rec *m534Rec) float64 {
	if rec.tmax <= rec.tmin {
		return 0
	}
	return float64(rec.tmax-rec.tmin) / 1e6
}

// m534Rendre publie les tableaux, dans l ordre du plan.
func m534Rendre(t *testing.T, rec *m534Rec) {
	t.Helper()
	d := m534Duree(rec)
	t.Logf("MARCHE (%d vues) : %d paquets lus, dont %d a liste pleine localises · %d vues "+
		"franchies · %d records ti=35, dont %d desynchronises (%.2f %%) · %.1f s couvertes",
		m534Vues(), rec.paquets, rec.pleinsVus, rec.vues, rec.ti35, rec.ti35Desync,
		m533bPart(rec.ti35Desync, rec.ti35), d)
	t.Logf("  COMPOSANT FAUTIF : %s", m533cTable(rec.fautifs))
	m534RendreAccroupi(t, rec, d)
	m534RendreGlissade(t, rec)
	m534RendreMobilite(t, rec)
	m534RendreVentilations(t, rec)
	m534RendreVitesse(t, rec)
}

// m534ParSlot regroupe des lectures par slot, chaque groupe trie par horodatage.
func m534ParSlot(evs []m534Ev) map[uint32][]m534Ev {
	m := map[uint32][]m534Ev{}
	for _, e := range evs {
		m[e.slot] = append(m[e.slot], e)
	}
	for s := range m {
		g := m[s]
		sort.Slice(g, func(i, j int) bool { return g[i].ts < g[j].ts })
		m[s] = g
	}
	return m
}

// m534RendreAccroupi : la cadence par slot, et les intervalles de PROGRESSION > 0,5.
func m534RendreAccroupi(t *testing.T, rec *m534Rec, duree float64) {
	t.Helper()
	parSlot := m534ParSlot(rec.accroupi)
	var actifs, prog5 int
	for _, e := range rec.accroupi {
		if e.v[0] != 0 {
			actifs++
		}
		if float64(e.v[1])/1023 > 0.5 {
			prog5++
		}
	}
	t.Logf("i29 ACCROUPI : %d lectures sur %d slots · %d avec le booleen POSE (%.1f %%) · "+
		"%d avec une progression > 0,5 (%.1f %%)", len(rec.accroupi), len(parSlot), actifs,
		m533bPart(actifs, len(rec.accroupi)), prog5, m533bPart(prog5, len(rec.accroupi)))
	t.Logf("  CADENCE PAR SLOT (records/s, les 8 plus actifs) : %s",
		m534Cadences(parSlot, duree))
	t.Logf("  INTERVALLES ENTRE TRANSITIONS (progression > 0,5) : %s",
		m534Intervalles(parSlot, func(e m534Ev) bool { return float64(e.v[1])/1023 > 0.5 }))
	t.Logf("  CINQ INSTANTS (temps de barre Theater) : %s",
		m534Instants(parSlot, rec.tmin, func(e m534Ev) bool { return e.v[0] != 0 }))
}

// m534RendreGlissade : compte, intervalles, instants.
func m534RendreGlissade(t *testing.T, rec *m534Rec) {
	t.Helper()
	parSlot := m534ParSlot(rec.glissade)
	var actives int
	for _, e := range rec.glissade {
		if e.v[0] != 0 {
			actives++
		}
	}
	t.Logf("i62 GLISSADE : %d lectures sur %d slots · %d avec la porte OUVERTE (%.1f %%)",
		len(rec.glissade), len(parSlot), actives, m533bPart(actives, len(rec.glissade)))
	t.Logf("  INTERVALLES ENTRE GLISSADES : %s",
		m534Intervalles(parSlot, func(e m534Ev) bool { return e.v[0] != 0 }))
	t.Logf("  CINQ INSTANTS (temps de barre Theater) : %s",
		m534Instants(parSlot, rec.tmin, func(e m534Ev) bool { return e.v[0] != 0 }))
}

// m534RendreMobilite : `i54`, datee. Le hook ne porte pas de slot : l instant est tout ce qu on a.
func m534RendreMobilite(t *testing.T, rec *m534Rec) {
	t.Helper()
	var amorces int
	var parts []string
	var dernier uint64
	for _, e := range rec.mobil {
		if e.v[0] == 0 {
			continue
		}
		amorces++
		if len(parts) < 5 && (dernier == 0 || e.ts > dernier+2_000_000) {
			dernier = e.ts
			parts = append(parts, m532Barre(e.ts, rec.tmin))
		}
	}
	t.Logf("i54 ACTION DE MOBILITE : %d lectures · %d avec le drapeau d amorce POSE (%.1f %%)",
		len(rec.mobil), amorces, m533bPart(amorces, len(rec.mobil)))
	t.Logf("  CINQ INSTANTS (temps de barre Theater ; pas de slot — le hook n en porte pas) : %s",
		strings.Join(parts, " · "))
}

// m534RendreVentilations : `i55` et la tete d `i18`.
func m534RendreVentilations(t *testing.T, rec *m534Rec) {
	t.Helper()
	total := 0
	for _, n := range rec.posture {
		total += n
	}
	t.Logf("i55 POSTURE : %d lectures sur %d slots · tags %s", total, len(rec.postureSlots),
		m533cTableU64(rec.posture))
	t.Logf("i18 CONTROLE D UNITE : %d lectures · %d avec la porte de tete POSEE (%.1f %%) · "+
		"premier index %s · second index %s", rec.ctrlTotal, rec.ctrlPorte,
		m533bPart(rec.ctrlPorte, rec.ctrlTotal), m533cTableU64(rec.ctrlF1),
		m533cTableU64(rec.ctrlF2))
}

// m534RendreVitesse : la composante VERTICALE (candidat du saut) et la vitesse AU SOL.
func m534RendreVitesse(t *testing.T, rec *m534Rec) {
	t.Helper()
	parSlot := m534ParSlot(rec.vitesse)
	var sol []float64
	var montees int
	type mont struct {
		slot uint32
		ts   uint64
		vz   float64
	}
	var mts []mont
	var pleine, absente int
	for _, e := range rec.vitesse {
		if e.v[0] != 0 {
			pleine++
			continue
		}
		if e.v[1] != 0 {
			absente++
			continue
		}
		v := DecodeVelocity(e.v[2], e.v[3])
		h := math.Hypot(float64(v[0]), float64(v[1]))
		sol = append(sol, h)
		if float64(v[2]) > m534SeuilMontee {
			montees++
			mts = append(mts, mont{e.slot, e.ts, float64(v[2])})
		}
	}
	t.Logf("i1 VITESSE : %d lectures sur %d slots · %d en pleine precision (non dequantifiees) · "+
		"%d absentes derriere leur porte · %d dequantifiees", len(rec.vitesse), len(parSlot),
		pleine, absente, len(sol))
	t.Logf("  VITESSE AU SOL (m/s) : %s", m534Quantiles(sol))
	t.Logf("  HISTOGRAMME AU SOL (classes de 1 m/s) : %s", m534Histo(sol))
	t.Logf("  MONTEES (composante verticale > %.1f m/s) : %d (%.1f %% des lectures dequantifiees)",
		m534SeuilMontee, montees, m533bPart(montees, len(sol)))
	sort.Slice(mts, func(i, j int) bool { return mts[i].ts < mts[j].ts })
	var parts []string
	var dernier uint64
	for _, m := range mts {
		if len(parts) == 5 {
			break
		}
		if dernier != 0 && m.ts < dernier+2_000_000 {
			continue
		}
		dernier = m.ts
		parts = append(parts, fmt.Sprintf("%s (slot %d, vz %.1f m/s)",
			m532Barre(m.ts, rec.tmin), m.slot, m.vz))
	}
	t.Logf("  CINQ INSTANTS DE MONTEE (temps de barre Theater) : %s", strings.Join(parts, " · "))
}

// m534SeuilMontee : le seuil de composante verticale au-dela duquel une lecture est comptee
// comme une MONTEE. 2 m/s — au-dessus du bruit de la marche en pente, sous la vitesse
// d ascension d un saut de Spartan. C est un SEUIL D INSTRUMENT, pas une grammaire.
const m534SeuilMontee = 2.0
