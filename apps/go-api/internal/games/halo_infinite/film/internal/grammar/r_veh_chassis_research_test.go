//go:build research

package grammar

// r_veh_chassis_research_test.go — CAMPAGNE DE GRAMMAIRE, RECHERCHE R-L4 (b) (2026-10-02) : les
// chassis `ti=40` « inconnus » (`77ef810a`, `4118381d`, `d0b40d0a`, et tous ceux des builds
// anciens) sont-ils de vrais chassis ou des lectures fausses ? Mesure seulement.
//
// L HYPOTHESE MESUREE : sur les formats de film <= 25, le premier champ du bloc MPP
// (`FUN_141fd72c0`) fait 8 bits et l index inline 3 (decoupage 8/3 que l oracle n2 designe,
// `profile.MPPPourFormat`), alors que le depot, faute de largeur posee pour ces formats, lit au
// defaut 9/5. Lu a 9 bits, le mot de 32 bits `MPPWord32` qui suit est decale d UN bit : le
// chassis lu est `(vrai << 1) | bit suivant`.
//
// Pour chaque record NEW `ti=40` de la marche des trames (la marche de la carte v2, decoupage du
// film), l instrument relit le bloc MPP a 9/5 ET a 8/3 depuis l en-tete du record ; pour chaque
// slot, il releve le chassis des records d image-cle (decoupage 8/3 et 9/5). Il rend, par film
// et par (chassis 9/5, chassis 8/3), le nombre de NEW, et si le chassis 8/3 est connu de la table
// des types de physique (CAMPAGNE_PHYSIQUE) et egal au chassis d image-cle du meme slot.
//
//	go test -tags=research -count=1 -run '^TestRVehChassisNeufs$' \
//	  ./internal/games/halo_infinite/film/internal/grammar/

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"

	"levelup/go-api/internal/filmproc"
	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
)

// rvcPhysique lit CAMPAGNE_PHYSIQUE (`chassis:type,...`), sans dependre de la surcouche.
func rvcPhysique() map[uint32]int {
	out := map[uint32]int{}
	for _, x := range strings.Split(os.Getenv("CAMPAGNE_PHYSIQUE"), ",") {
		g, ty, ok := strings.Cut(strings.TrimSpace(x), ":")
		if !ok {
			continue
		}
		gid, err1 := strconv.ParseUint(g, 16, 32)
		v, err2 := strconv.Atoi(ty)
		if err1 == nil && err2 == nil {
			out[uint32(gid)] = v
		}
	}
	return out
}

// rvcMPPDuNeuf relit le MPPWord32 d un record NEW depuis son en-tete, au decoupage `w`.
func rvcMPPDuNeuf(pay []byte, headerBit int, cfg FrameConfig, w profile.MPPWidths) (uint32, uint32, bool) {
	obs := NouvelleObservation()
	var mpp uint32
	vu := false
	obs.MppHook = func(f MPPField, v uint64, present bool) {
		if f == MPPWord32 && present && !vu {
			mpp, vu = uint32(v), true //nolint:gosec // mot de 32 bits
		}
	}
	c := cfg
	c.Obs = obs
	c.Profil.MPP = w
	br := LecteurSur(pay)
	br.poserCadre(c)
	defer obs.neutraliserCaptures()()
	br.SetBitPos(headerBit)
	if c.HasExtraFields {
		br.Skip(32)
	}
	if readRecordType(br) != recNew {
		return 0, 0, false
	}
	readRecordID(br, c.IDLowBits, c.IDBase)
	ti := uint32(br.ReadBits(6))
	if ti != VehicleTypeIndex {
		return 0, ti, false
	}
	consumeVersionPrefix(br)
	consumeMultiplayerPropertiesBlock(br)
	return mpp, ti, vu
}

// rvcEcouteur releve les NEW `ti=40` de la marche.
type rvcEcouteur struct {
	cfg   FrameConfig
	neufs []rvcNeuf
}

// rvcNeuf : un record NEW `ti=40` et ses deux lectures.
type rvcNeuf struct {
	slot     uint32
	m95, m83 uint32
	ferme    bool // le paquet du NEW est ferme (carte v2)
	propre   bool // le NEW est lu jusqu au bout (aucune desynchronisation)
}

func (e *rvcEcouteur) debutDeChunk(int, []byte, []FilmPacket, *World) {}
func (e *rvcEcouteur) finDeFilm()                                     {}

func (e *rvcEcouteur) paquet(_ int, p *cmPaquet, _ *World) {
	// Les records de la marche detaillee ne portent pas leur en-tete (HeaderBit nul) : on le
	// retrouve en relisant la vue B dans l ordre, comme `campagne_invariants_research_test.go`.
	pos := p.d.DebutVueB
	for _, r := range p.recs {
		br := LecteurSur(p.pay)
		br.poserCadre(e.cfg)
		br.SetBitPos(pos)
		if e.cfg.HasExtraFields {
			br.Skip(32)
		}
		if readRecordType(br) != r.Type || readRecordID(br, e.cfg.IDLowBits, e.cfg.IDBase) != r.ID {
			return
		}
		if r.Type == recNew && r.TypeIndex == VehicleTypeIndex {
			m95, _, ok95 := rvcMPPDuNeuf(p.pay, pos, e.cfg, profile.MPPWidths{Lead: 9, Index: 5})
			m83, _, ok83 := rvcMPPDuNeuf(p.pay, pos, e.cfg, profile.MPPWidths{Lead: 8, Index: 3})
			if ok95 || ok83 {
				e.neufs = append(e.neufs, rvcNeuf{slot: r.Slot, m95: m95, m83: m83, ferme: p.d.Fermee, propre: r.DesyncAt < 0})
			}
		}
		switch {
		case r.Type == recDel:
			br.Skip(32)
			pos = br.BitPos()
		case r.DesyncAt >= 0 || (r.Trace.EndBit == 0 && len(r.Trace.Comps) == 0):
			return
		default:
			pos = r.Trace.EndBit
		}
	}
}

// rvcChassisDImageCle : par slot, les chassis `ti=40` lus en image-cle au decoupage `w`.
func rvcChassisDImageCle(f *cmFilm, w profile.MPPWidths) map[uint32]map[uint32]bool {
	out := map[uint32]map[uint32]bool{}
	prec := f.fc.PoserMPP(w)
	defer f.fc.PoserMPP(prec)
	ctx := f.fc.ContexteDeLecture()
	marche := f.fc.MarcheDImageCle()
	for _, num := range f.fc.ChunkNumbers() {
		data, pks, ok := f.fc.ChunkAt(num)
		if !ok {
			continue
		}
		for _, pk := range pks {
			if pk.Type != PacketTypeKeyframe {
				continue
			}
			pay := pk.Payload(data)
			for _, r := range marche.Records(pay) {
				if r.TI != VehicleTypeIndex {
					continue
				}
				vu := false
				c := ctx
				c.Obs = &Observation{MppHook: func(fl MPPField, v uint64, present bool) {
					if fl == MPPWord32 && present && !vu {
						vu = true
						if out[uint32(r.Slot)] == nil { //nolint:gosec // slot < 2^30
							out[uint32(r.Slot)] = map[uint32]bool{} //nolint:gosec // idem
						}
						out[uint32(r.Slot)][uint32(v)] = true //nolint:gosec // idem
					}
				}}
				WalkKeyframeFullState(pay, r.Bit, f.reg, c)
			}
		}
	}
	return out
}

// rvcCle : une ligne de sortie.
type rvcCle struct {
	source   string
	m95, m83 uint32
}

// rvcCompte : ce qu une ligne cumule.
type rvcCompte struct {
	n, egal, fermes, propres int
	imageCle                 map[uint32]bool
}

// rvcCreations : le balayeur de creations de production (`ScanVehicleCreations`) au decoupage
// `w`, cle (slot, chunk, paquet) -> chassis.
func rvcCreations(f *cmFilm, w profile.MPPWidths) map[[3]int]uint32 {
	prec := f.fc.PoserMPP(w)
	defer f.fc.PoserMPP(prec)
	wr := profile.QuantRangeCEBiped()
	out := map[[3]int]uint32{}
	cre, _, err := ScanVehicleCreations(f.fc, &wr)
	if err != nil {
		return out
	}
	for _, c := range cre {
		out[[3]int{int(c.Slot), c.Chunk, c.PacketIndex}] = uint32(c.MPPVal[MPPWord32]) //nolint:gosec // 32 bits
	}
	return out
}

// TestRVehChassisNeufs : chassis des NEW `ti=40` (marche des trames, balayeur de creations) a 9/5
// et a 8/3, confrontes au chassis d image-cle (8/3) du meme slot.
func TestRVehChassisNeufs(t *testing.T) {
	racine, sortie, films := b2Env(t)
	physique := rvcPhysique()
	utiles := cmUtiles(t)
	lignes := []string{"film\tbuild\tsource\tchassis_9_5\ttype_9_5\tchassis_8_3\ttype_8_3\tneufs\t" +
		"egal_image_cle_du_slot\tpaquets_fermes\tneufs_lus_au_bout\tchassis_image_cle_du_slot"}
	ty := func(m uint32) string {
		if v, ok := physique[m]; ok {
			return strconv.Itoa(v)
		}
		return "inconnu"
	}
	for _, id := range films {
		garde := filmproc.Arm("campagne/r-veh-chassis", 4, func(pic uint64) {
			fmt.Fprintf(os.Stderr, "%s : plafond memoire franchi (%d octets)\n", id, pic)
			os.Exit(3)
		})
		f, ok := cmOuvrir(t, racine, id, utiles)
		if !ok {
			garde.Disarm()
			continue
		}
		kf83 := rvcChassisDImageCle(f, profile.MPPWidths{Lead: 8, Index: 3})
		acc := map[rvcCle]*rvcCompte{}
		noter := func(k rvcCle, slot uint32, ferme, propre bool) {
			c := acc[k]
			if c == nil {
				c = &rvcCompte{imageCle: map[uint32]bool{}}
				acc[k] = c
			}
			c.n++
			if kf83[slot][k.m83] {
				c.egal++
			}
			if ferme {
				c.fermes++
			}
			if propre {
				c.propres++
			}
			for m := range kf83[slot] {
				c.imageCle[m] = true
			}
		}
		e := &rvcEcouteur{cfg: f.cfg}
		cmMarcher(f, cmVariante{}, e)
		for _, x := range e.neufs {
			noter(rvcCle{"marche-NEW", x.m95, x.m83}, x.slot, x.ferme, x.propre)
		}
		c95 := rvcCreations(f, profile.MPPWidths{Lead: 9, Index: 5})
		c83 := rvcCreations(f, profile.MPPWidths{Lead: 8, Index: 3})
		for k, m95 := range c95 {
			m83, ok := c83[k]
			if !ok {
				m83 = 0xffffffff
			}
			noter(rvcCle{"balayeur-creations", m95, m83}, uint32(k[0]), false, false) //nolint:gosec // slot
		}
		for k, m83 := range c83 {
			if _, ok := c95[k]; !ok {
				noter(rvcCle{"balayeur-creations", 0xffffffff, m83}, uint32(k[0]), false, false) //nolint:gosec // slot
			}
		}
		for k, c := range acc {
			var kfs []string
			for m := range c.imageCle {
				kfs = append(kfs, fmt.Sprintf("%08x", m))
			}
			sort.Strings(kfs)
			lignes = append(lignes, fmt.Sprintf("%s\t%s\t%s\t%08x\t%s\t%08x\t%s\t%d\t%d\t%d\t%d\t%s", id, f.build, k.source,
				k.m95, ty(k.m95), k.m83, ty(k.m83), c.n, c.egal, c.fermes, c.propres, strings.Join(kfs, ",")))
		}
		t.Logf("%s %s : %d NEW ti=40 relus, %d / %d creations (9/5 / 8/3), pic %d Mio", id, f.build, len(e.neufs),
			len(c95), len(c83), garde.Peak()>>20)
		garde.Disarm()
	}
	b2Ecrire(t, sortie, "r_veh_chassis_neufs.tsv", lignes)
}
