//go:build research

package grammar

// p3_armes_naissance_research_test.go — SONDE P3 DE LA CAMPAGNE « RETOURS REJEU » (2026-09-23) :
// LES ARMES DE NAISSANCE sont-elles dans le record NEW du bipede (ti=35) ? (mesure seule, aucun
// code de production, aucun artefact.)
//
// FAIT DE DEPART (`.ai/V7.5/retours_rejeu_2026-09-23/RAPPORT_fiche_armes.md` §2.3) : le flux delta
// n emet jamais i43..i46 a la naissance (0/106 en Super Fiesta, etalon i48 94/106) ; la marche du
// depot traverse le record NEW du bipede et lit l arme via `consumeWeaponStateTypeInfoVariant`, que
// personne ne branche.
//
// MESURE. La marche de production de la vue B (`decodeInferLoop`, via `t519Marcher`, monde lie par
// les images-cles + la table de datums + la table anticipee, comme la sonde P1) avec un
// `HeldWeaponHook` pose sur l observateur du cadre. Pour chaque record NEW ti=35 : (slot, generation,
// horodatage), et pour chaque emplacement `weapon-state-type-info` lu : porte, moitie haute (la
// FAMILLE, celle des `loadouts` du document) et moitie basse. La moitie haute est RELUE au bit de
// debut du composant (`CompResult.StartBit`) et la moitie basse relue doit egaler `CompResult.Variant`
// (controle de cadrage) ; le crochet doit avoir vu la meme paire (controle du branchement).
//
// ORACLES (document publie du meme match, lecture seule) :
//   - O1 image-cle : premiere entree `loadouts` de la vie, sans prise intermediaire (aucun
//     `weaponChanges` du slot dans [naissance, image-cle], aucun `pickups` d arme apres la frame de
//     naissance + 1) ; accord = MEME ENSEMBLE de familles (et, a part, meme ordre) ;
//   - O2 tirs : premier tir de la vie sans prise intermediaire ; accord = famille du tir dans
//     l ensemble lu a la naissance ;
//   - TEMOIN : la lecture de naissance d une vie contre l oracle de la vie SUIVANTE (autre slot) ;
//     HASARD = accord moyen sur toutes les paires (i != j) de la population.
//
// L horloge du document (pas de 100 ms) : origine = premier paquet du chunk 1 + `originMs` du
// document (la formule de `resolveOriginMs`).
//
// SUITE (fichiers `p3_armes_naissance_{recherche,catalogue,rapport}_research_test.go`) : la marche
// ne rend presque aucune naissance (P3.2) ; la RECHERCHE D EN-TETE EXACT trouve le record NEW de
// chaque vie (P3.5) ; la traversee de production y arrive desalignee sur i43 (P3.3, P3.6) ; les
// familles des oracles sont localisees dans le record (P3.6) et se lisent en ENCHAINANT les
// emplacements par le deserialiseur de production (P3.7) ; la LECTURE PAR CATALOGUE, sans oracle de
// vie, donne les taux et les accords (P3.8, P3.2, P3.3).
//
//	MOUV511_FILM=<depot>/data/cache/film_chunks/<id> MOUV511_BORNES=<catalogue> MOUV511_CARTE=<carte> \
//	P3_DOC=<depot>/data/cache/replays/halo_infinite/<id>.json \
//	  go test -tags=research -count=1 -v -timeout 60m -run '^TestP3ArmesNaissance$' \
//	  ./internal/games/halo_infinite/film/internal/grammar/

import (
	"encoding/json"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// p3Arme est UN emplacement d arme lu dans un record NEW.
type p3Arme struct {
	idx      int
	present  bool
	hi, lo   uint32
	cadreOK  bool // moitie basse relue == CompResult.Variant
	crochet  bool // le crochet a vu la meme paire
	startBit int
	horsCat  bool // lecture par catalogue : famille presente absente du catalogue (exclue des familles)
}

// p3Naissance est UN record NEW ti=35 de la marche.
type p3Naissance struct {
	ts        uint64
	chunk     int
	slot, gen uint32
	desync    int
	annoncees int // emplacements d arme au masque
	armes     []p3Arme
	trame     int
	rattachee bool
	bit       int  // bit de l en-tete dans le paquet (recherche)
	finMarche int  // curseur de fin de vue B de la marche sur ce paquet (recherche)
	hitEndB   bool // la marche a clos sa vue B sur ce paquet
}

// familles rend l ensemble des familles presentes, trie.
func (n p3Naissance) familles() []uint32 {
	var out []uint32
	for _, a := range n.armes {
		if a.present && !a.horsCat {
			out = append(out, a.hi)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// ordonnees rend les familles presentes dans l ordre des emplacements.
func (n p3Naissance) ordonnees() []uint32 {
	var out []uint32
	for _, a := range n.armes {
		if a.present && !a.horsCat {
			out = append(out, a.hi)
		}
	}
	return out
}

// p3Passe porte ce que la marche rend.
type p3Passe struct {
	naissances                     []p3Naissance
	paquets, nonLocalises          int
	news                           map[uint32]int
	crochetAppels, crochetPresents int
	desyncNew35                    map[int]int
}

// p3Marcher decode le film UNE fois.
func p3Marcher(tc t516Temoin) *p3Passe {
	p := &p3Passe{news: map[uint32]int{}, desyncNew35: map[int]int{}}
	arch, _ := tc.reg.Archetype(BipedTypeIndex)
	armeIdx := map[int]bool{}
	for i, nom := range arch.Components {
		if nom == compWeaponStateTypeInfo {
			armeIdx[i] = true
		}
	}
	type paire struct{ hi, lo uint32 }
	var vues []paire
	obs := NouvelleObservation()
	obs.HeldWeaponHook = func(h, l uint32) {
		p.crochetAppels++
		if h != noVariant {
			p.crochetPresents++
		}
		vues = append(vues, paire{h, l})
	}
	cfg := tc.cfg
	cfg.Obs = obs
	w := NewWorld(tc.reg)
	w.PoserTableAnticipee(ConstruireTableAnticipee(tc.fc))
	for _, c := range tc.fc.ChunkNumbers() {
		data, pks, ok := tc.fc.ChunkAt(c)
		if !ok {
			continue
		}
		w.PoserChunkCourant(c)
		t525Lier(w, data, pks)
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta || pk.Size < 1 {
				continue
			}
			pay := pk.Payload(data)
			p.paquets++
			debut := DefaultPacketPreambleBits
			if _, present := PacketHeadEventType(pay); present {
				if debut = marchLocate(pay, w, cfg); debut < 0 {
					p.nonLocalises++
					continue
				}
			}
			vues = vues[:0]
			mar := t519Marcher(pay, w, cfg, debut)
			curseur := 0
			for _, r := range mar.recs {
				if r.Type != recNew {
					continue
				}
				p.news[r.TypeIndex]++
				if r.TypeIndex != BipedTypeIndex {
					continue
				}
				p.desyncNew35[r.DesyncAt]++
				n := p3Naissance{ts: pk.TimestampUS, chunk: c, slot: r.Slot, gen: r.ID >> 30,
					desync: r.DesyncAt}
				for _, i := range tcgIndices(r.Trace.Mask) {
					if armeIdx[i] {
						n.annoncees++
					}
				}
				for _, cr := range r.Trace.Comps {
					if !armeIdx[cr.Index] {
						continue
					}
					a := p3Relire(pay, cr)
					for k := curseur; k < len(vues); k++ {
						if vues[k].hi == a.hi && vues[k].lo == a.lo {
							a.crochet, curseur = true, k+1
							break
						}
					}
					n.armes = append(n.armes, a)
				}
				p.naissances = append(p.naissances, n)
			}
		}
	}
	return p
}

// p3Relire relit un emplacement d arme a son bit de debut : porte R(1), moitie haute R(32), moitie
// basse R(32) — la forme de `consumeWeaponStateTypeInfoVariant`.
func p3Relire(pay []byte, cr CompResult) p3Arme {
	a := p3Arme{idx: cr.Index, hi: noVariant, lo: noVariant, startBit: cr.StartBit}
	br := LecteurSur(pay)
	br.Skip(cr.StartBit)
	if br.ReadBit() {
		a.present = true
		a.hi = uint32(br.ReadBits(32))
		a.lo = uint32(br.ReadBits(32))
	}
	a.cadreOK = a.lo == cr.Variant
	return a
}

// --- LE DOCUMENT (oracles) ---

type p3Doc struct {
	FrameCount int   `json:"frameCount"`
	OriginMs   int64 `json:"originMs"`
	Tracks     []struct {
		Slot   uint32 `json:"slot"`
		Start  *int   `json:"startFrame"`
		End    *int   `json:"endFrame"`
		Xuid   string `json:"xuid"`
		Points []struct {
			T int `json:"t"`
		} `json:"points"`
	} `json:"tracks"`
	Loadouts []struct {
		T    int      `json:"t"`
		Slot uint32   `json:"slot"`
		W    []string `json:"w"`
	} `json:"loadouts"`
	Shots []struct {
		T    int    `json:"t"`
		Slot uint32 `json:"slot"`
		W    string `json:"w"`
	} `json:"shots"`
	WeaponChanges []struct {
		T    int    `json:"t"`
		Slot uint32 `json:"slot"`
		Kind string `json:"kind"`
	} `json:"weaponChanges"`
	Pickups []struct {
		T    int    `json:"t"`
		Slot uint32 `json:"slot"`
		Kind string `json:"kind"`
	} `json:"pickups"`
}

// p3Vie est UNE vie du document et ses oracles.
type p3Vie struct {
	slot            uint32
	xuid            string
	start, end      int
	debutConnu      bool
	o1              []uint32 // familles de la premiere image-cle sans prise intermediaire (triees)
	o1Ordre         []uint32
	o1T             int
	o2              uint32 // famille du premier tir sans prise intermediaire
	o2T             int
	aO1, aO2        bool
	prisesNaissance int           // ramassages d arme dans [start, start+1]
	naissance       *p3Naissance  // rattachee depuis la marche de la vue B
	trouvee         *p3Naissance  // trouvee par la recherche d en-tete exact
	candidats       []p3Naissance // tous les en-tetes de la fenetre, dans l ordre
}

// p3Hex lit « 0x48C19D2D » ou « 0x48C19D2D42C9679F » : rend la moitie HAUTE (8 premiers chiffres).
func p3Hex(s string) uint32 {
	s = strings.TrimPrefix(strings.ToLower(s), "0x")
	if len(s) > 8 {
		s = s[:8]
	}
	v, _ := strconv.ParseUint(s, 16, 32)
	return uint32(v)
}

// p3Vies lit le document et construit les vies et leurs oracles.
func p3Vies(t *testing.T, chemin string) ([]*p3Vie, int64) {
	t.Helper()
	blob, err := os.ReadFile(chemin)
	if err != nil {
		t.Fatalf("document : %v", err)
	}
	var d p3Doc
	if err := json.Unmarshal(blob, &d); err != nil {
		t.Fatalf("document : %v", err)
	}
	var vies []*p3Vie
	for _, tr := range d.Tracks {
		v := &p3Vie{slot: tr.Slot, xuid: tr.Xuid}
		if tr.Start != nil {
			v.start, v.debutConnu = *tr.Start, *tr.Start > 0
		}
		v.end = d.FrameCount
		if tr.End != nil {
			v.end = *tr.End
		} else if n := len(tr.Points); n > 0 {
			v.end = tr.Points[n-1].T
		}
		vies = append(vies, v)
	}
	for _, v := range vies {
		premierePrise := 1 << 30
		for _, c := range d.WeaponChanges {
			if c.Slot == v.slot && c.T >= v.start && c.T <= v.end && c.T < premierePrise {
				premierePrise = c.T
			}
		}
		for _, pk := range d.Pickups {
			if pk.Slot != v.slot || pk.Kind != "weapon" || pk.T < v.start || pk.T > v.end {
				continue
			}
			if pk.T <= v.start+1 {
				v.prisesNaissance++
				continue
			}
			if pk.T < premierePrise {
				premierePrise = pk.T
			}
		}
		p3OraclesDeVie(v, &d, premierePrise)
	}
	return vies, d.OriginMs
}

// p3OraclesDeVie pose O1 et O2 d une vie : avant toute prise intermediaire.
func p3OraclesDeVie(v *p3Vie, d *p3Doc, premierePrise int) {
	meilleur := -1
	for i, l := range d.Loadouts {
		if l.Slot == v.slot && l.T >= v.start && l.T <= v.end && l.T <= premierePrise &&
			(meilleur < 0 || l.T < d.Loadouts[meilleur].T) {
			meilleur = i
		}
	}
	if meilleur >= 0 {
		l := d.Loadouts[meilleur]
		v.aO1, v.o1T = true, l.T
		for _, s := range l.W {
			v.o1Ordre = append(v.o1Ordre, p3Hex(s))
		}
		v.o1 = append([]uint32(nil), v.o1Ordre...)
		sort.Slice(v.o1, func(i, j int) bool { return v.o1[i] < v.o1[j] })
	}
	meilleur = -1
	for i, s := range d.Shots {
		if s.Slot == v.slot && s.T >= v.start && s.T <= v.end && s.T <= premierePrise &&
			(meilleur < 0 || s.T < d.Shots[meilleur].T) {
			meilleur = i
		}
	}
	if meilleur >= 0 {
		v.aO2, v.o2T, v.o2 = true, d.Shots[meilleur].T, p3Hex(d.Shots[meilleur].W)
	}
}
