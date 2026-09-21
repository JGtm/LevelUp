//go:build research

package grammar

// mouvement_5_7_retenus_research_test.go — LES ETATS DE MOUVEMENT DES RECORDS RETENUS, ET EUX
// SEULS (lot 5.7).
//
// # LE DEFAUT D INSTRUMENT QUE CE FICHIER CONTOURNE, ET QU IL FAUT LIRE AVANT LES CHIFFRES
//
// La porte de publication des etats de mouvement (`etats_mouvement_hooks.go`) tire depuis
// `traverseComponentLoop`, et DEUX chemins de la marche appellent cette boucle sur des
// alignements CANDIDATS, dont ils jettent ensuite la quasi-totalite :
//
//	`marchLocateStrict`  — la localisation du premier record d un paquet a liste d evenements :
//	                       elle essaie des offsets jusqu a ce qu un decodage tienne ;
//	`deltaBodyTrial`     — l inference de chaine (`frame_chain_infer.go`), qui decode un corps
//	                       candidat CONTRE CHAQUE ARCHETYPE du registre, sous budget d essais.
//
// Mesure sur `bfecd02b`, marche du jeu a trois vues, largeurs de carte installees :
//
//	composant                        | dans les records RENDUS | porte, localisation | porte, vues
//	unit-crouch-component            |                      60 |               6 358 |       2 776
//	biped-posture-physics-component  |                      52 |               2 359 |       1 425
//	biped-slide-component            |                      56 |               2 656 |       1 398
//	biped-mobility-action-component  |                     321 |               3 059 |       1 555
//	unit-control-component           |                     101 |               4 678 |       2 614
//	object-translational-velocity-*  |                  75 488 |              13 221 |      82 401
//
// Autrement dit : pour un composant declare sur 77 % des records (`i1`) le bruit d essais ne
// pese que 1,27 fois le signal ; pour les composants d etat, declares sur moins d un record sur
// mille, il pese de 14 a 152 FOIS le signal. Toute statistique tiree de la porte brute mesure
// donc les essais, pas le film.
//
// # CE QUE CE FICHIER MESURE A LA PLACE
//
// Il ne branche AUCUN hook pendant la marche. Il garde les records RENDUS par
// [DecodeFrameViews], et pour chacun il relit ses composants d etat A LEUR `StartBit`, qui est
// celui que la boucle de composants a consigne dans `Trace.Comps`. La relecture emploie le MEME
// profil et le MEME deserialiseur : elle repasse sur les memes bits, au meme endroit, et
// l egalite de largeur est verifiee (`m57ReluLargeur`).
//
// C est une SECONDE LECTURE, et c est dit : l idiome du depot veut que le hook soit la
// grammaire. Ici la porte ne peut pas distinguer un essai d un record retenu, et un instrument
// qui publierait ses chiffres sans le dire publierait le bruit des essais.
//
// Rejouable : memes variables que `TestMouvement57Posture`.

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// m57RLu est UNE lecture relue sur un record RETENU.
type m57RLu struct {
	comp EtatMouvementComposant
	slot uint32
	ts   uint64
	v    []uint64
}

// m57RRec : ce que la passe des retenus collecte.
type m57RRec struct {
	lus      []m57RLu
	ti35     int
	desync   int
	largeurs map[string]int
	ecarts   int
}

// m57CompsEtat lie le NOM de registre d un composant d etat a son enumere de publication. La
// cle est le NOM : le registre du film numerote les composants, et l index n est pas un
// invariant (une premiere sonde a compte l index 55 et rendu 52 lectures au lieu de 3 784).
var m57CompsEtat = map[string]EtatMouvementComposant{
	"unit-crouch-component":           EtatAccroupi,
	"unit-crouch":                     EtatAccroupi,
	"biped-slide-component":           EtatGlissade,
	"biped-slide":                     EtatGlissade,
	"biped-posture-physics-component": EtatPosture,
	"biped-mobility-action-component": EtatMobilite,
	"biped-mobility-action":           EtatMobilite,
	"object-translational-velocity-dynamic-precision-component": EtatVitesse,
}

// TestMouvement57Retenus publie les etats de mouvement des records retenus.
func TestMouvement57Retenus(t *testing.T) {
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
	rec := &m57RRec{largeurs: map[string]int{}}
	cfg := fc.CadreDeBalayage()
	cfg.Obs = nil // AUCUN hook pendant la marche : c est tout le point de ce fichier.
	w := NewWorld(reg)
	for _, c := range fc.ChunkNumbers() {
		data, pks, ok := fc.ChunkAt(c)
		if !ok {
			continue
		}
		m533bLierMonde(w, data, pks)
		for _, pk := range pks {
			m57RPaquet(pk, data, w, cfg, rec)
		}
	}
	m57RRendre(t, rec, w)
}

// m57RPaquet marche UN paquet et relit les composants d etat de ses records de bipede.
func m57RPaquet(pk FilmPacket, data []byte, w *World, cfg FrameConfig, rec *m57RRec) {
	if pk.Type != PacketTypeDelta || pk.Size < 1 {
		return
	}
	pay := pk.Payload(data)
	debut := 2
	if _, present := PacketHeadEventType(pay); present {
		if debut = marchLocateStrict(pay, w, cfg); debut < 0 {
			return
		}
	}
	recs, _ := DecodeFrameViews(pay, w, cfg, 3, debut)
	for _, r := range recs {
		if r.TypeIndex != BipedTypeIndex {
			continue
		}
		rec.ti35++
		if r.DesyncAt >= 0 {
			rec.desync++
		}
		m57RRelire(pay, r, cfg, rec, pk.TimestampUS)
	}
}

// m57RRelire relit, pour UN record retenu, ses composants d etat a leur `StartBit`.
func m57RRelire(pay []byte, r FrameRecord, cfg FrameConfig, rec *m57RRec, ts uint64) {
	comps := r.Trace.Comps
	for i, c := range comps {
		comp, ok := m57CompsEtat[c.Name]
		if !ok || !c.Ported {
			continue
		}
		fin := r.Trace.EndBit
		if i+1 < len(comps) {
			fin = comps[i+1].StartBit
		}
		v, largeur := m57ReluLargeur(pay, c.StartBit, comp, cfg)
		if v == nil {
			continue
		}
		rec.largeurs[c.Name] += largeur
		if fin > 0 && c.StartBit+largeur != fin {
			// L ECART EST COMPTE, PAS TU : si la relecture ne consomme pas exactement ce que la
			// boucle a consomme, la valeur relue n est pas celle du record.
			rec.ecarts++
			continue
		}
		rec.lus = append(rec.lus, m57RLu{comp: comp, slot: r.Slot, ts: ts, v: v})
	}
}

// m57ReluLargeur relit UN composant d etat a `depart` et rend ses valeurs et sa largeur.
func m57ReluLargeur(pay []byte, depart int, comp EtatMouvementComposant,
	cfg FrameConfig) ([]uint64, int) {
	var out []uint64
	obs := NouvelleObservation()
	obs.EtatMouvementHook = func(c EtatMouvementComposant, _ uint32, v []uint64) {
		if c == comp && out == nil {
			out = append([]uint64(nil), v...)
		}
	}
	br := LecteurSur(pay)
	br.PoserContexte(ContexteDeLecture{Profil: cfg.Profil, Obs: obs})
	br.Skip(depart)
	switch comp {
	case EtatAccroupi:
		consumeUnitCrouch(br)
	case EtatGlissade:
		consumeBipedSlide(br, m57NiveauGlissade)
	case EtatPosture:
		consumeBipedPosturePhysics(br)
	case EtatMobilite:
		consumeBipedMobilityAction(br)
	case EtatVitesse:
		consumeObjectTranslationalVelocity(br)
	case EtatControleUnite:
		return nil, 0
	}
	return out, br.BitPos() - depart
}

// m57NiveauGlissade est le `level` que le registre du film donne a `i62` : la boucle de
// composants le passe (`arch.Level(i)`), et la relecture doit le passer aussi — sinon le second
// R(8) de la queue manque.
const m57NiveauGlissade = 1

// m57RRendre publie les comptes par composant et la ventilation des tags de posture.
func m57RRendre(t *testing.T, rec *m57RRec, w *World) {
	t.Helper()
	t.Logf("RECORDS RETENUS : %d ti=35 (%d desynchronises) · %d relectures de largeur egale · "+
		"%d ECARTS DE LARGEUR ecartes", rec.ti35, rec.desync, len(rec.lus), rec.ecarts)
	parComp := map[EtatMouvementComposant]int{}
	slots := map[EtatMouvementComposant]map[uint32]bool{}
	for _, l := range rec.lus {
		parComp[l.comp]++
		if slots[l.comp] == nil {
			slots[l.comp] = map[uint32]bool{}
		}
		slots[l.comp][l.slot] = true
	}
	for _, c := range []EtatMouvementComposant{EtatAccroupi, EtatGlissade, EtatPosture,
		EtatMobilite, EtatVitesse} {
		t.Logf("  %-46s : %6d lectures sur %3d slots", c.String(), parComp[c], len(slots[c]))
	}
	m57RPosture(t, rec, w)
}

// m57RPosture publie la ventilation des tags d `i55` et la vitesse verticale TENUE, sur les
// seules lectures de records retenus.
func m57RPosture(t *testing.T, rec *m57RRec, w *World) {
	t.Helper()
	var vits []m57Vit
	var posts []m57Post
	for _, l := range rec.lus {
		lie := func(s uint32) bool {
			ti, ok := w.ArchetypeForSlot(s)
			return ok && ti == BipedTypeIndex
		}
		if !lie(l.slot) {
			continue
		}
		switch l.comp {
		case EtatVitesse:
			if l.v[0] != 0 || l.v[1] != 0 {
				continue
			}
			vec := DecodeVelocity(l.v[2], l.v[3])
			vits = append(vits, m57Vit{slot: l.slot, ts: l.ts, vz: float64(vec[2]),
				sol: hypot32(vec[0], vec[1])})
		case EtatPosture:
			posts = append(posts, m57Post{slot: l.slot, ts: l.ts, tag: l.v[0]})
		case EtatAccroupi, EtatGlissade, EtatControleUnite, EtatMobilite:
		}
	}
	parSlot := m57ParSlot(vits)
	parTag := map[uint64][]float64{}
	for _, p := range posts {
		v, ok := m57VitTenue(parSlot[p.slot], p.ts)
		if !ok {
			continue
		}
		parTag[p.tag] = append(parTag[p.tag], v.vz)
	}
	var parts []string
	for tag := uint64(0); tag < 4; tag++ {
		vz := parTag[tag]
		sort.Float64s(vz)
		var monte int
		for _, x := range vz {
			if x >= m57SeuilMontee {
				monte++
			}
		}
		parts = append(parts, fmt.Sprintf("tag %d : %d avec vitesse tenue, vz mediane %+.3f, "+
			"vz >= %.1f sur %d (%.1f %%)", tag, len(vz), m57Quantile(vz, 0.50), m57SeuilMontee,
			monte, m533bPart(monte, len(vz))))
	}
	t.Logf("`i55` SUR LES RECORDS RETENUS : %d lectures liees au bipede, %d vitesses",
		len(posts), len(vits))
	for _, p := range parts {
		t.Logf("  %s", p)
	}
	t.Logf("  (denominateurs volontairement bruts : sur ce film la population retenue est de " +
		"l ordre de la cinquantaine de lectures, et aucune conclusion ne s y tient)")
	_ = strings.TrimSpace("")
}
