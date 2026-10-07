//go:build research

package grammar

// ri27d0_monde_research_test.go — LA MESURE DE 2.7.d0, volet des objets du monde (plan de l etape 2
// de la representation intermediaire) : les records que la couche de recuperation des objets du monde
// lit aujourd hui — les echantillons de pistes (equipement, armes au sol, projectiles ;
// [echantillonsDesBandes]) et les creations (equipement, armes au sol, vehicules ;
// [releverLesCreations]) — contre ceux que la marche des trames lit dans les memes bandes : records
// DELTA dont le composant i0 se dequantifie ([decodeWorldObjectPos]), records NEW. Les records sont
// compares par (chunk, paquet, archetype, slot, generation) puis par le bit de leur i0 (pistes) ou de
// leur en-tete (creations). Aucun fichier de production n est touche.
//
// Sortie (RI27C_OUT) : `monde_d0.tsv`
//
//	S  film  classe  compte
//	D  film  classe  chunk  paquet  ti  slot  gen  detail   (au plus 30 par classe et par film)
//
//	RI27C_FILMS=<id,...> RI27C_RACINE=<film_chunks> RI27C_OUT=<dossier> [RI27C_CARTES=<id=Carte;...>] \
//	  go test -tags=research -count=1 -run '^TestRI27d0Monde$' -timeout 120m \
//	  ./internal/games/halo_infinite/film/internal/grammar/

import (
	"fmt"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/lecture"
	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
)

// ri27d0CleMonde designe un record d objet du monde.
type ri27d0CleMonde struct {
	chunk, paquet, ti int
	slot, gen         uint32
}

// ri27d0Monde est le canal des trames de la mesure : il releve les records d objets du monde que la
// marche lit, et ce que chaque trame prouve.
type ri27d0Monde struct {
	m          *MarcheDistribuee
	wr         profile.Vec3Range
	lg         profile.PrecisionDescriptor
	bandes     map[int]map[uint32]bool
	bandesCrea map[int]map[uint32]bool
	pistes     map[ri27d0CleMonde]int // bit d i0 des records delta dequantifies
	neufs      map[ri27d0CleMonde]int // bit d en-tete des records NEW
	horsBande  map[string]int
	trames     map[paquetDuFlux]trameDuCanal
	// vus : ce que la marche a lu de chaque slot dans chaque paquet.
	vus     map[ri27d0CleSlot]string
	classes map[string]int
	ech     map[string]int
	lignes  []string
	court   string
}

func (*ri27d0Monde) Interets() []Interet { return nil }
func (*ri27d0Monde) Clore(BilanDeMarche) {}

func (c *ri27d0Monde) Brancher(_ *Observation, m *MarcheDistribuee) { c.m = m }

// Trame releve les records d objets du monde de la trame marchee `p`.
func (c *ri27d0Monde) Trame(p *lecture.Paquet) {
	recs, lus := c.m.recordsDeLaTrame()
	cle := paquetDuFlux{p.Chunk, p.Index}
	if !lus {
		c.trames[cle] = trameDuCanal{prouveeDes: rienDeProuve}
		return
	}
	t := trameDuCanal{prouveeDes: preuveDeLaTrame(p)}
	for i := range recs {
		r := &recs[i]
		t.slots = append(t.slots, r.Slot)
		c.vus[ri27d0CleSlot{p.Chunk, p.Index, r.Slot}] = ri27d0Decrire(p.Payload, r, &c.wr, c.lg)
		ti := int(r.TypeIndex)
		k := ri27d0CleMonde{p.Chunk, p.Index, ti, r.Slot, r.ID >> 30}
		switch r.Type {
		case recDelta:
			band, ok := c.bandes[ti]
			if !ok {
				continue
			}
			if !band[r.Slot] {
				c.horsBande[fmt.Sprintf("delta_ti%d", ti)]++
				continue
			}
			for _, co := range r.Trace.Comps {
				if co.Index == 0 && co.Ported {
					if _, okp := decodeWorldObjectPos(p.Payload, co.StartBit, &c.wr, c.lg); okp {
						c.pistes[k] = co.StartBit
					}
				}
			}
		case recNew:
			band, ok := c.bandesCrea[ti]
			if !ok {
				continue
			}
			if !band[r.Slot] {
				c.horsBande[fmt.Sprintf("neuf_ti%d", ti)]++
				continue
			}
			c.neufs[k] = r.HeaderBit
		}
	}
	c.trames[cle] = t
}

func (c *ri27d0Monde) classer(classe string, k ri27d0CleMonde, detail string) {
	c.classes[classe]++
	if c.ech[classe] >= 30 {
		return
	}
	c.ech[classe]++
	c.lignes = append(c.lignes, fmt.Sprintf("D\t%s\t%s\t%d\t%d\t%d\t%d\t%d\t%s", c.court, classe, k.chunk, k.paquet,
		k.ti, k.slot, k.gen, detail))
}

// ri27d0PistesAnciennes rend les records que la passe des pistes accepte ([echantillonsDesBandes]),
// avec le bit de leur i0, pour les bandes des archetypes `tis`.
func ri27d0PistesAnciennes(fc *FilmContext, wr profile.Vec3Range, lg profile.PrecisionDescriptor,
	tis []int, bandes [][]uint32) map[ri27d0CleMonde]int {
	app := appartenanceDesBandes(bandes)
	out := map[ri27d0CleMonde]int{}
	posBits := projPosBits(lg)
	for _, ch := range fc.ChunkNumbers() {
		data, pks, ok := fc.ChunkAt(ch)
		if !ok {
			continue
		}
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta {
				continue
			}
			pay := pk.Payload(data)
			limit := len(pay)*8 - (worldObjectHeaderBits + worldObjectIndexBits + posBits)
			var curseurs [maxBandesParPasse]int
			for p := 0; p <= limit; p++ {
				h, ok := enteteDObjetDuMonde(pay, p)
				if !ok || int(h.Slot) >= len(app) || app[h.Slot] == 0 {
					continue
				}
				lu, accepte, after := false, false, 0
				for i := range bandes {
					if app[h.Slot]&(1<<uint(i)) == 0 || curseurs[i] > p {
						continue
					}
					if !lu {
						lu = true
						rec, okm := masqueDObjetDuMonde(pay, p, h)
						if okm && rec.Idx[0] == 0 {
							_, accepte = decodeWorldObjectPos(pay, rec.After, &wr, lg)
							after = rec.After
						}
					}
					if !accepte {
						break
					}
					out[ri27d0CleMonde{ch, pk.Index, tis[i], h.Slot, h.Gen}] = after
					curseurs[i] = p + posBits + 1
				}
			}
		}
	}
	return out
}

// comparer range chaque record des deux lectures, pour un genre (`pistes` ou `creations`).
func (c *ri27d0Monde) comparer(genre string, anciens, neufs map[ri27d0CleMonde]int) {
	for k, b := range anciens {
		nb, ok := neufs[k]
		switch {
		case ok && nb == b:
			c.classes[fmt.Sprintf("%s:ti%d:commun_meme_bit", genre, k.ti)]++
		case ok:
			c.classer(fmt.Sprintf("%s:ti%d:commun_bit_different", genre, k.ti), k, fmt.Sprintf("ancien=%d marche=%d", b, nb))
		default:
			t, vu := c.trames[paquetDuFlux{k.chunk, k.paquet}]
			raison := "recupere_derriere_la_marche"
			switch {
			case !vu:
				raison = "trame_non_rendue"
			case slices.Contains(t.slots, k.slot):
				raison = "marche_lit_le_slot:" + ri27d0Classe(c.vus[ri27d0CleSlot{k.chunk, k.paquet, k.slot}])
			case int64(b) >= int64(t.prouveeDes):
				raison = "prouve_par_la_fermeture"
			}
			c.classer(fmt.Sprintf("%s:ti%d:ancien_seul:%s", genre, k.ti, raison), k, fmt.Sprintf("bit=%d prouvee_des=%d marche=%s", b, t.prouveeDes, c.vus[ri27d0CleSlot{k.chunk, k.paquet, k.slot}]))
		}
	}
	for k, nb := range neufs {
		if _, ok := anciens[k]; ok {
			continue
		}
		t := c.trames[paquetDuFlux{k.chunk, k.paquet}]
		etat := "trame_fermee"
		if t.prouveeDes == rienDeProuve {
			etat = "trame_non_prouvee"
		}
		c.classer(fmt.Sprintf("%s:ti%d:marche_seule:%s", genre, k.ti, etat), k, fmt.Sprintf("bit=%d", nb))
	}
}

func TestRI27d0Monde(t *testing.T) {
	films, racine, sortie := ri27cEnv(t)
	cat, err := profile.LoadMapQuantCatalog(e191bCatalogue())
	if err != nil {
		t.Fatalf("catalogue : %v", err)
	}
	var lignes []string
	for _, court := range films {
		fc := ri27cContexte(t, filepath.Join(racine, court), court, true)
		entree, err := cat.Lookup(ri27cCarte(t, court))
		if err != nil {
			t.Fatalf("carte de %s : %v", court, err)
		}
		wr, lg := entree.Range(), fc.ProfilDeBalayage().LargeursObjetDuMonde()
		c := &ri27d0Monde{wr: wr, lg: lg, court: court, bandes: map[int]map[uint32]bool{},
			bandesCrea: map[int]map[uint32]bool{}, pistes: map[ri27d0CleMonde]int{}, neufs: map[ri27d0CleMonde]int{},
			horsBande: map[string]int{}, trames: map[paquetDuFlux]trameDuCanal{}, classes: map[string]int{},
			ech: map[string]int{}, vus: map[ri27d0CleSlot]string{}}
		tis := archetypesAPistes()
		var bandes [][]uint32
		for _, ti := range tis {
			c.bandes[ti] = worldObjectSlotBand(fc, ti)
			bandes = append(bandes, slotsDeLaBande(c.bandes[ti]))
		}
		for _, ti := range archetypesDeCreation() {
			c.bandesCrea[int(ti)] = worldObjectSlotBand(fc, int(ti))
		}
		anciennes := ri27d0PistesAnciennes(fc, wr, lg, tis, bandes)
		creations := map[ri27d0CleMonde]int{}
		// LA PASSE SEULE (depuis 2.7.d3, les points d entree de production font passer la marche devant
		// elle) : chaque archetype de creation marche par [releverLesCreations].
		for ti, band := range c.bandesCrea {
			w, err := fc.marcheDeCreation(uint32(ti), &wr, band) //nolint:gosec // archetype de creation
			if err != nil {
				t.Logf("%s : creations ti=%d : %v", court, ti, err)
				continue
			}
			cres, _ := releverLesCreations(fc, []equipCreationWalk{w})
			for _, x := range cres[0] {
				creations[ri27d0CleMonde{x.Chunk, x.PacketIndex, ti, x.Slot, x.Gen}] = x.BitPos
			}
		}
		if err := Distribuer(fc, c); err != nil {
			t.Fatalf("%s : distribution : %v", court, err)
		}
		c.comparer("pistes", anciennes, c.pistes)
		c.comparer("creations", creations, c.neufs)
		for k, n := range c.horsBande {
			c.classes["marche_hors_bande:"+k] = n
		}
		lignes = append(lignes, ri27d0Trier(court, "S", c.classes)...)
		lignes = append(lignes, c.lignes...)
		t.Logf("%s : pistes anciennes %d / marche %d ; creations anciennes %d / marche %d", court,
			len(anciennes), len(c.pistes), len(creations), len(c.neufs))
		runtime.GC()
	}
	ri27cEcrire(t, filepath.Join(sortie, "monde_d0.tsv"), lignes)
}

// ri27d0CleSlot designe un slot dans un paquet.
type ri27d0CleSlot struct {
	chunk, paquet int
	slot          uint32
}

// ri27d0Decrire decrit un record de la marche : genre, archetype, generation, en-tete, et ce qu il
// fait de son i0 (absent, non porte, dequantifie, refuse).
func ri27d0Decrire(pay []byte, r *FrameRecord, wr *profile.Vec3Range, lg profile.PrecisionDescriptor) string {
	i0 := "sans_i0"
	for _, co := range r.Trace.Comps {
		if co.Index != 0 {
			continue
		}
		switch _, ok := decodeWorldObjectPos(pay, co.StartBit, wr, lg); {
		case !co.Ported:
			i0 = "i0_non_porte"
		case ok:
			i0 = fmt.Sprintf("i0_lu@%d", co.StartBit)
		default:
			i0 = fmt.Sprintf("i0_refuse@%d", co.StartBit)
		}
	}
	return fmt.Sprintf("%s/ti%d/gen%d/@%d/%s", nomDeTypeDeRecord(r.Type), r.TypeIndex, r.ID>>30, r.HeaderBit, i0)
}

// ri27d0Classe reduit une description a sa classe (genre, archetype, sort d i0).
func ri27d0Classe(d string) string {
	parts := strings.Split(d, "/")
	if len(parts) < 5 {
		return "inconnu"
	}
	i0 := parts[4]
	if k := strings.IndexByte(i0, '@'); k >= 0 {
		i0 = i0[:k]
	}
	return parts[0] + "_" + parts[1] + "_" + i0
}
