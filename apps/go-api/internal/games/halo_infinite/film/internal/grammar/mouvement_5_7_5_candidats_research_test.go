//go:build research

package grammar

// mouvement_5_7_5_candidats_research_test.go — LES CHAMPS REPLIQUES, NOTES CONTRE LES DEUX
// VERITES TERRAIN PHYSIQUES (lot 5.7.5, volet C).
//
// # LA METHODE, ET POURQUOI ELLE EST DANS CE SENS
//
// Le volet A/B (`mouvement_5_7_5_oracles_research_test.go`) etiquette les SAUTS par leur hauteur
// et les plateaux de vitesse par leur valeur. Ces etiquettes sont la VERITE TERRAIN. Ce fichier
// prend chaque champ replique du bipede, en fait une suite d instants (slot, horodatage), et
// demande : ces instants tombent-ils la ou la physique dit qu un saut commence, ou dans un
// plateau de vitesse donne ?
//
// LE SCORE EST DOUBLE, ET LES DEUX SENS COMPTENT. La PRECISION dit quelle part des instants du
// candidat tombe sur l etiquette ; le RAPPEL dit quelle part des etiquettes porte un instant du
// candidat. Un champ qui ne s allume qu une fois par match aurait une precision de 100 % sans
// rien expliquer ; un champ toujours allume aurait un rappel de 100 % sans rien distinguer. Un
// candidat n est NOMME qu au-dela de 90 % dans les deux sens.
//
// # POURQUOI LES CANDIDATS SONT RELUS A LEUR `StartBit`
//
// Les portes de publication d `i18`, d `i57` et d `i59` ne portent PAS le slot — or une
// coincidence par VIE l exige. Plutot que d ajouter trois portes de production pour une mesure,
// l instrument relit chaque composant A SON `StartBit`, celui que la boucle de composants a
// consigne dans `Trace.Comps` du record RETENU : il connait donc le slot (celui du record) et
// l instant (celui du paquet). La technique est validee au § 5.7.2.d — 0 ecart de largeur sur
// 75 977 relectures.
//
// Rejouable : memes variables que `TestMouvement575Oracles`.

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// m575Obs est UN instant ou un candidat est vrai, attribue a une vie.
type m575Obs struct {
	slot uint32
	ts   uint64
}

// m575FenetreAmorceUS est la tolerance autour de l amorce d un saut. Deux ticks de 60 Hz de part
// et d autre : un etat replique ne voyage pas forcement sur le tick exact du decollage.
const m575FenetreAmorceUS = 33333

// m575CandidatsComposants : les composants relus, et le nom de registre par lequel on les
// reconnait dans `Trace.Comps`.
var m575CandidatsComposants = []string{
	compUnitControl,
	"biped-spartan-ability-component",
	"biped-spartan-ability-non-predicted-state",
	"biped-posture-physics-component",
	"biped-mobility-action-component",
}

// TestMouvement575Candidats note chaque champ replique contre les deux etiquettes physiques.
func TestMouvement575Candidats(t *testing.T) {
	rec, cands, ok := m575PasseCandidats(t)
	if !ok {
		return
	}
	t.Logf("CANDIDATS RELUS : %d champs distincts, %d instants au total",
		len(cands), m575TotalObs(cands))
	m575NoterSauts(t, rec, cands)
	m575NoterPlateaux(t, rec, cands)
}

// m575PasseCandidats marche le film, garde les records RETENUS, et relit leurs composants
// candidats a leur `StartBit`.
func m575PasseCandidats(t *testing.T) (*m57Rec, map[string][]m575Obs, bool) {
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
	cands := map[string][]m575Obs{}
	var ts uint64
	cfg := fc.CadreDeBalayage()
	cfg.Obs = m57Observateur(rec, &ts)
	w := NewWorld(reg)
	for _, c := range fc.ChunkNumbers() {
		data, pks, okC := fc.ChunkAt(c)
		if !okC {
			continue
		}
		m533bLierMonde(w, data, pks)
		for _, pk := range pks {
			m575PaquetCandidats(pk, data, w, cfg, rec, &ts, cands)
		}
	}
	m57Filtrer(t, rec, w)
	return rec, cands, m57Oracle(t, rec)
}

// m575PaquetCandidats marche UN paquet et relit les candidats de ses records de bipede retenus.
func m575PaquetCandidats(pk FilmPacket, data []byte, w *World, cfg FrameConfig, rec *m57Rec,
	ts *uint64, cands map[string][]m575Obs) {
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
		if ti, okB := w.ArchetypeForSlot(r.Slot); !okB || ti != BipedTypeIndex {
			continue
		}
		m575RelireCandidats(pay, r, cfg, pk.TimestampUS, cands)
	}
}

// m575RelireCandidats relit les composants candidats d UN record retenu.
func m575RelireCandidats(pay []byte, r FrameRecord, cfg FrameConfig, ts uint64,
	cands map[string][]m575Obs) {
	for _, c := range r.Trace.Comps {
		if !c.Ported || !m575EstCandidat(c.Name) {
			continue
		}
		for _, nom := range m575LireUnCandidat(pay, c.StartBit, c.Name, cfg) {
			cands[nom] = append(cands[nom], m575Obs{slot: r.Slot, ts: ts})
		}
	}
}

// m575EstCandidat dit si le composant fait partie des candidats relus.
func m575EstCandidat(nom string) bool {
	for _, n := range m575CandidatsComposants {
		if n == nom {
			return true
		}
	}
	return false
}

// m575LireUnCandidat relit UN composant a `depart` et rend les NOMS DE CANDIDAT vrais a cet
// instant (un bit pose, une etiquette prise, un drapeau leve).
func m575LireUnCandidat(pay []byte, depart int, nom string, cfg FrameConfig) []string {
	var out []string
	obs := NouvelleObservation()
	obs.UnitRefHook = func(u UnitRefRead) {
		if u.Kind != UnitRefWord32 || !u.Present {
			return
		}
		out = append(out, "i18.mot32.porte")
		for b := 0; b < 32; b++ {
			if u.Val&(1<<uint(b)) != 0 {
				out = append(out, fmt.Sprintf("i18.mot32.bit%02d", b))
			}
		}
	}
	obs.SpartanAbilityHook = func(tag, sub, ref uint64, hasRef bool) {
		out = append(out, fmt.Sprintf("i57.etiquette=%d", int(tag)-1))
		if hasRef {
			out = append(out, fmt.Sprintf("i57.sous=%d", sub), "i57.reference")
			_ = ref
		}
	}
	obs.AbilityNonPredictedHook = func(st AbilityNonPredictedState) {
		out = append(out, fmt.Sprintf("i59.tag=%d", st.Tag))
		if st.BodyWalked {
			out = append(out, fmt.Sprintf("i59.interne=%d", st.Inner))
		}
	}
	obs.EtatMouvementHook = func(comp EtatMouvementComposant, _ uint32, v []uint64) {
		switch comp {
		case EtatPosture:
			out = append(out, fmt.Sprintf("i55.tag=%d", v[0]))
		case EtatMobilite:
			if v[0] != 0 {
				out = append(out, "i54.amorce")
			}
			if len(v) > 1 && v[1] != 0 {
				out = append(out, "i54.second")
			}
		case EtatAccroupi:
			if v[0] != 0 {
				out = append(out, "i29.accroupi")
			}
		case EtatGlissade, EtatControleUnite, EtatVitesse:
		}
	}
	br := LecteurSur(pay)
	br.PoserContexte(ContexteDeLecture{Profil: cfg.Profil, Obs: obs})
	br.Skip(depart)
	m575Consommer(br, nom, cfg)
	return out
}

// m575Consommer appelle le deserialiseur du composant nomme.
func m575Consommer(br *Lecteur, nom string, cfg FrameConfig) {
	switch nom {
	case compUnitControl:
		consumeUnitControl(br)
	case "biped-spartan-ability-component":
		consumeBipedSpartanAbility(br)
	case "biped-spartan-ability-non-predicted-state":
		consumeBipedSpartanAbilityNonPredictedState(br, 2)
	case "biped-posture-physics-component":
		consumeBipedPosturePhysics(br)
	case "biped-mobility-action-component":
		consumeBipedMobilityAction(br)
	}
	_ = cfg
}

// m575TotalObs compte tous les instants de candidat.
func m575TotalObs(cands map[string][]m575Obs) int {
	var n int
	for _, v := range cands {
		n += len(v)
	}
	return n
}

// m575NoterSauts note chaque candidat contre les AMORCES DE SAUT etiquetees par la hauteur.
func m575NoterSauts(t *testing.T, rec *m57Rec, cands map[string][]m575Obs) {
	t.Helper()
	eps := m575Episodes(rec)
	h, ok := m575HauteurPic(eps)
	if !ok {
		t.Logf("SCORES SAUT : aucun pic de hauteur — rien a noter")
		return
	}
	parSlot := map[uint32][]uint64{}
	var total int
	for _, e := range eps {
		if e.hauteur < h*(1-m575ToleranceH) || e.hauteur > h*(1+m575ToleranceH) {
			continue
		}
		parSlot[e.slot] = append(parSlot[e.slot], e.t0)
		total++
	}
	t.Logf("SCORES CONTRE LE SAUT (H = %.2f m, %d amorces etiquetees, fenetre +/- %d us) :",
		h, total, m575FenetreAmorceUS)
	m575Tableau(t, cands, total, func(o m575Obs) bool {
		for _, t0 := range parSlot[o.slot] {
			if m575Proximite(o.ts, t0) <= m575FenetreAmorceUS {
				return true
			}
		}
		return false
	}, func(o m575Obs) uint64 {
		for _, t0 := range parSlot[o.slot] {
			if m575Proximite(o.ts, t0) <= m575FenetreAmorceUS {
				return t0
			}
		}
		return 0
	})
}

// m575HauteurPic rend la hauteur du pic au-dessus du plancher de bruit.
func m575HauteurPic(eps []m575Episode) (float64, bool) {
	casiers := map[int]int{}
	for _, e := range eps {
		casiers[int(e.hauteur/0.1)]++
	}
	pic, n := -1, 0
	for k, c := range casiers {
		if float64(k)*0.1 < m575HauteurMiniM {
			continue
		}
		if c > n {
			pic, n = k, c
		}
	}
	if pic < 0 {
		return 0, false
	}
	return (float64(pic) + 0.5) * 0.1, true
}

// m575Proximite rend l ecart absolu entre deux horodatages.
func m575Proximite(a, b uint64) uint64 {
	if a > b {
		return a - b
	}
	return b - a
}

// m575NoterPlateaux note chaque candidat contre les DEUX plateaux de vitesse les plus lourds.
func m575NoterPlateaux(t *testing.T, rec *m57Rec, cands map[string][]m575Obs) {
	t.Helper()
	pls := m575Plateaux(rec)
	casiers := map[int]float64{}
	for _, p := range pls {
		casiers[int(p.valeur/0.25)] += p.dureeS
	}
	var cles []int
	for k := range casiers {
		cles = append(cles, k)
	}
	sort.Slice(cles, func(i, j int) bool { return casiers[cles[i]] > casiers[cles[j]] })
	for i, k := range cles {
		if i >= 2 {
			break
		}
		parSlot := map[uint32][][2]uint64{}
		var total int
		for _, p := range pls {
			if int(p.valeur/0.25) != k {
				continue
			}
			parSlot[p.slot] = append(parSlot[p.slot], [2]uint64{p.t0, p.t1})
			total++
		}
		t.Logf("SCORES CONTRE LE PLATEAU %.2f-%.2f m/s (%d plateaux, %.0f s) :",
			float64(k)*0.25, float64(k+1)*0.25, total, casiers[k])
		m575Tableau(t, cands, total, func(o m575Obs) bool {
			for _, iv := range parSlot[o.slot] {
				if o.ts >= iv[0] && o.ts <= iv[1] {
					return true
				}
			}
			return false
		}, func(o m575Obs) uint64 {
			for _, iv := range parSlot[o.slot] {
				if o.ts >= iv[0] && o.ts <= iv[1] {
					return iv[0]
				}
			}
			return 0
		})
	}
}

// m575Tableau publie, pour chaque candidat, sa precision et son rappel contre une etiquette.
func m575Tableau(t *testing.T, cands map[string][]m575Obs, totalEtiquettes int,
	dedans func(m575Obs) bool, cle func(m575Obs) uint64) {
	t.Helper()
	type ligne struct {
		nom              string
		n, bons, couvres int
		prec, rapp       float64
	}
	var ls []ligne
	for nom, obs := range cands {
		if len(obs) < 5 {
			continue // un candidat a moins de cinq instants ne se note pas
		}
		var bons int
		vus := map[uint64]bool{}
		for _, o := range obs {
			if !dedans(o) {
				continue
			}
			bons++
			vus[cle(o)] = true
		}
		l := ligne{nom: nom, n: len(obs), bons: bons, couvres: len(vus)}
		l.prec = m533bPart(bons, len(obs))
		l.rapp = m533bPart(len(vus), totalEtiquettes)
		ls = append(ls, l)
	}
	sort.Slice(ls, func(i, j int) bool {
		if ls[i].prec != ls[j].prec {
			return ls[i].prec > ls[j].prec
		}
		return ls[i].rapp > ls[j].rapp
	})
	var nomme []string
	for i, l := range ls {
		if i < 12 {
			t.Logf("  %-22s : %5d instants · %5d dans l etiquette (precision %5.1f %%) · "+
				"%4d etiquettes couvertes (rappel %5.1f %%)", l.nom, l.n, l.bons, l.prec,
				l.couvres, l.rapp)
		}
		if l.prec >= 90 && l.rapp >= 90 {
			nomme = append(nomme, l.nom)
		}
	}
	if len(ls) > 12 {
		t.Logf("  ... %d candidats de moins bonne precision non listes", len(ls)-12)
	}
	if len(nomme) == 0 {
		t.Logf("  AUCUN CANDIDAT NOMME : aucun champ n atteint 90 %% dans les DEUX sens. "+
			"Ce n est pas un negatif — c est le score de %d candidats mesures.", len(ls))
		return
	}
	t.Logf("  CANDIDAT(S) NOMME(S) : %s", strings.Join(nomme, " · "))
}

// m575Densite compte, par NOM de composant, les records retenus qui le declarent.
func m575Densite(recs []FrameRecord) (map[string]int, int) {
	parNom := map[string]int{}
	var total int
	for _, r := range recs {
		if r.TypeIndex != BipedTypeIndex {
			continue
		}
		total++
		for _, c := range r.Trace.Comps {
			parNom[c.Name]++
		}
	}
	return parNom, total
}

// TestMouvement575DensiteRecords marche le film et publie le recensement, trie par densite.
func TestMouvement575DensiteRecords(t *testing.T) {
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
	cfg := fc.CadreDeBalayage()
	cfg.Obs = nil
	w := NewWorld(reg)
	parNom := map[string]int{}
	var total int
	for _, c := range fc.ChunkNumbers() {
		data, pks, okC := fc.ChunkAt(c)
		if !okC {
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
			recs, _ := DecodeFrameViews(pay, w, cfg, 3, debut)
			n, tt := m575Densite(recs)
			for k, v := range n {
				parNom[k] += v
			}
			total += tt
		}
	}
	noms := make([]string, 0, len(parNom))
	for k := range parNom {
		noms = append(noms, k)
	}
	sort.Slice(noms, func(i, j int) bool { return parNom[noms[i]] > parNom[noms[j]] })
	t.Logf("DENSITE DES COMPOSANTS DU BIPEDE SUR %d RECORDS RETENUS (un etat par instant exige "+
		"un composant DENSE) :", total)
	for _, n := range noms {
		t.Logf("  %-56s : %7d (%5.2f %%)", n, parNom[n], m533bPart(parNom[n], total))
	}
}
