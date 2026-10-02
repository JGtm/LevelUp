//go:build research

package grammar

// campagne_blocs_research_test.go — MESURES CIBLEES (etape 4) : ce que le film dit de ses slots,
// hors de la marche des trames — blocs de type 1 par chunk, declarations des images-cles,
// allocateur (T1-3) — et les tables de comptes que les sondes remplissent.

import (
	"fmt"
	"sort"
	"strings"
)

// cmCompte : un compte de la campagne. `n` compte l objet de la table (eid, record, paquet...).
type cmCompte struct{ n, paquets, horsCadre, fermes, enJeu int }

// cmTables : table -> cle -> compte.
type cmTables map[string]map[string]*cmCompte

// add ajoute a une cle.
func (t cmTables) add(table, cle string, c cmCompte) {
	if t[table] == nil {
		t[table] = map[string]*cmCompte{}
	}
	x := t[table][cle]
	if x == nil {
		x = &cmCompte{}
		t[table][cle] = x
	}
	x.n += c.n
	x.paquets += c.paquets
	x.horsCadre += c.horsCadre
	x.fermes += c.fermes
	x.enJeu += c.enJeu
}

// un : un objet compte une fois.
func (t cmTables) un(table, cle string) { t.add(table, cle, cmCompte{n: 1}) }

// cmCles rend les cles triees d une table.
func cmCles(m map[string]*cmCompte) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// cmBlocs : ce que le film dit de ses slots hors de la marche.
type cmBlocs struct {
	entrees map[int][]DatumEntry
	queue   map[int][datumQueueMots]uint32
	suivant map[int]int
	// plafond : le plus grand slot ALLOUE (generation ou drapeaux poses) par un bloc du film.
	plafond uint32
	// gens : par slot, les generations vues allouees dans un bloc du film (bit g).
	gens map[uint32]uint8
	// kf : les cles (slot, tete) qu une image-cle du film declare ; kfSlots, les slots.
	kf      map[uint64]bool
	kfSlots map[uint32]bool
	// declares : par chunk, l archetype que la PREMIERE image-cle donne a chaque slot (le verdict
	// des NEW refuses, `jugerLesNeufsRefuses`).
	declares map[int]map[uint32]uint32
	table    *TableAnticipee
	// dico : le pont masque de composants du bloc -> archetype (NOTE 5.21 §3.2 : constant par
	// archetype), appris sur les slots vivants que la premiere image-cle du chunk nomme.
	dico map[[4]uint64]map[uint32]bool
}

// cmLireBlocs lit tout ce que le film dit de ses slots.
func cmLireBlocs(f *cmFilm) *cmBlocs {
	b := &cmBlocs{entrees: map[int][]DatumEntry{}, queue: map[int][datumQueueMots]uint32{},
		suivant: map[int]int{}, gens: map[uint32]uint8{}, kf: map[uint64]bool{},
		kfSlots: map[uint32]bool{}, declares: map[int]map[uint32]uint32{},
		dico: map[[4]uint64]map[uint32]bool{}}
	nums := f.fc.ChunkNumbers()
	for i := 0; i+1 < len(nums); i++ {
		b.suivant[nums[i]] = nums[i+1]
	}
	marche := f.fc.MarcheDImageCle()
	for _, c := range nums {
		data, pks, ok := f.fc.ChunkAt(c)
		if !ok {
			continue
		}
		premiere := true
		for _, pk := range pks {
			switch pk.Type {
			case PacketTypeDatums:
				bl, err := LireBlocDeDatums(pk.Payload(data))
				if err != nil {
					continue
				}
				b.entrees[c], b.queue[c] = bl.Entrees, bl.Queue
			case PacketTypeKeyframe:
				if premiere {
					pay := pk.Payload(data)
					table, _ := TableDeDatums(pay)
					b.declares[c] = archetypesDeclares(marche.Marcher(pay).Records, table)
					premiere = false
				}
			}
		}
		for s, e := range b.entrees[c] {
			if ti, ok := b.declares[c][uint32(s)]; ok && e.Vivante() && e.Composants != ([4]uint64{}) { //nolint:gosec // slot
				if b.dico[e.Composants] == nil {
					b.dico[e.Composants] = map[uint32]bool{}
				}
				b.dico[e.Composants][ti] = true
			}
			if e.Gen != 0 || e.Drapeaux != 0 {
				b.gens[uint32(s)] |= 1 << e.Gen //nolint:gosec // slot < 8191
				if uint32(s) > b.plafond {      //nolint:gosec // idem
					b.plafond = uint32(s) //nolint:gosec // idem
				}
			}
		}
	}
	b.table = ConstruireTableAnticipee(f.fc)
	for cle := range b.table.entrees {
		b.kf[uint64(cle.slot)<<2|uint64(cle.tete)] = true
		b.kfSlots[cle.slot] = true
	}
	return b
}

// entree rend l entree d un slot au bloc du chunk `c`, et si le bloc la porte.
func (b *cmBlocs) entree(c int, slot uint32) (DatumEntry, bool) {
	e := b.entrees[c]
	if int(slot) >= len(e) {
		return DatumEntry{}, false
	}
	return e[slot], true
}

// cmAlloueSous : l entree porte une allocation (vivante ou liberee) sous la generation `gen`.
func cmAlloueSous(e DatumEntry, gen uint8) bool {
	return e.Gen == gen && (e.Gen != 0 || e.Drapeaux != 0)
}

// naissance : la classe de la carte v2 (`v2_datums.go`) pour un eid rejete dans le chunk `c`.
func (b *cmBlocs) naissance(c int, eid uint32, neufLu bool) string {
	slot, gen := eid&0x3fffffff, uint8(eid>>30) //nolint:gosec // deux bits
	if neufLu {
		return "NEW lu dans le chunk"
	}
	n, ok := b.suivant[c]
	if !ok || b.entrees[n] == nil {
		return "non mesurable"
	}
	cur, _ := b.entree(c, slot)
	nx, _ := b.entree(n, slot)
	switch nxA, curA := cmAlloueSous(nx, gen), cmAlloueSous(cur, gen); {
	case nxA && !curA:
		return "naissance non lue"
	case gen == 0 && nx.Gen == 0 && nx.Drapeaux == 0 && cur.Gen != 0:
		return "naissance non lue, generation 0" // FUN_142f2e598 : gen = (gen + 1) & 3 a l allocation
	case curA && cur.Vivante():
		return "vivant au bloc du chunk"
	case curA:
		return "libere avant le chunk"
	case nx.Gen != cur.Gen || nx.Drapeaux != cur.Drapeaux:
		return "realloue sous une autre generation"
	}
	return "aucune allocation"
}

// plausibilite : la ventilation de T1-1 (correction 3) d un eid rejete.
func (b *cmBlocs) plausibilite(classeNaissance string, eid uint32) string {
	slot, gen := eid&0x3fffffff, uint8(eid>>30) //nolint:gosec // deux bits
	switch {
	case strings.HasPrefix(classeNaissance, "naissance non lue"):
		return "atteste : naissance au bloc suivant"
	case b.gens[slot]&(1<<gen) != 0:
		return "atteste : un bloc du film (meme tete)"
	case b.kf[uint64(slot)<<2|uint64(gen)]:
		return "atteste : une image-cle (meme tete)"
	case slot > b.plafond:
		return "slot au-dela du plafond du film"
	}
	return "indetermine"
}

// Les pools de l allocateur (T1 §2.2, table statique 0x143cefd78) : par plage de slots.
var cmBasesDePool = [5]int{0, 512, 768, 1024, 1280}

// cmPoolDe rend le pool d un slot (par plage).
func cmPoolDe(slot uint32) int {
	for p := 4; p >= 0; p-- {
		if int(slot) >= cmBasesDePool[p] {
			return p
		}
	}
	return 0
}

// cmProfondeurPrediction : nombre d allocations predites par pool.
const cmProfondeurPrediction = 64

// predictions rend, pour le chunk `c`, le rang de chaque slot dans la suite des allocations
// predites de son pool (next-fit depuis le curseur de queue, libre = `(f&1)==0 || (f&2)!=0`), sans
// liberation pendant le chunk. Nil sans bloc.
func (b *cmBlocs) predictions(c int) map[uint32]int {
	e := b.entrees[c]
	if e == nil {
		return nil
	}
	q := b.queue[c]
	out := map[uint32]int{}
	for p := 0; p < 5; p++ {
		base, fin := cmBasesDePool[p], len(e)
		if p < 4 {
			fin = cmBasesDePool[p+1]
		}
		if fin > len(e) {
			fin = len(e)
		}
		if base >= fin {
			continue
		}
		cur := int(q[p])
		if cur < base || cur >= fin {
			cur = base
		}
		rang := 0
		for k := 0; k < fin-base && rang < cmProfondeurPrediction; k++ {
			s := base + (cur-base+k)%(fin-base)
			if d := e[s].Drapeaux; d&1 == 0 || d&2 != 0 {
				out[uint32(s)] = rang //nolint:gosec // slot < 8191
				rang++
			}
		}
	}
	return out
}

// classeDeRang classe un rang de prediction, tete comprise.
func (b *cmBlocs) classeDeRang(c int, pred map[uint32]int, eid uint32) string {
	if pred == nil {
		return "sans bloc"
	}
	slot, gen := eid&0x3fffffff, uint8(eid>>30) //nolint:gosec // deux bits
	r, ok := pred[slot]
	if !ok {
		if int(slot) >= len(b.entrees[c]) {
			return fmt.Sprintf("pool %d · au-dela de la table", cmPoolDe(slot))
		}
		return fmt.Sprintf("pool %d · hors des %d predits", cmPoolDe(slot), cmProfondeurPrediction)
	}
	e, _ := b.entree(c, slot)
	tete := "tete predite"
	if (e.Gen+1)&3 != gen {
		tete = "tete autre"
	}
	var rg string
	switch {
	case r == 0:
		rg = "rang 0"
	case r <= 3:
		rg = "rang 1-3"
	case r <= 15:
		rg = "rang 4-15"
	default:
		rg = "rang 16-63"
	}
	return fmt.Sprintf("pool %d · %s · %s", cmPoolDe(slot), rg, tete)
}

// cmTemoin choisit un eid TEMOIN pour un eid rejete : meme tete, slot jamais alloue dans les blocs
// du chunk et du suivant, jamais declare par une image-cle, distinct des autres temoins.
func (b *cmBlocs) cmTemoin(c int, eid uint32, pris map[uint32]bool) (uint32, bool) {
	e, n := b.entrees[c], b.entrees[b.suivant[c]]
	if e == nil || n == nil {
		return 0, false
	}
	lim := len(e)
	if len(n) < lim {
		lim = len(n)
	}
	slot := eid & 0x3fffffff
	for k := 0; k < lim; k++ {
		s := uint32((int(slot)*7919 + 4099 + k) % lim) //nolint:gosec // borne par lim
		if s == slot || pris[s] || b.kfSlots[s] || b.gens[s] != 0 {
			continue
		}
		if x := e[s]; x.Gen != 0 || x.Drapeaux != 0 {
			continue
		}
		pris[s] = true
		return eid&0xc0000000 | s, true
	}
	return 0, false
}

// cmJoindre assemble des dimensions en cle.
func cmJoindre(parts ...string) string { return strings.Join(parts, " · ") }

// archetypeDuBloc rend l archetype que le pont du bloc SUIVANT donne a un slot ne dans le chunk
// `c` : « resolu », « ambigu » (plusieurs archetypes pour ce masque) ou « inconnu ».
func (b *cmBlocs) archetypeDuBloc(c int, slot uint32) (uint32, string) {
	n, ok := b.suivant[c]
	if !ok {
		return 0, "sans bloc suivant"
	}
	e, ok := b.entree(n, slot)
	if !ok || e.Composants == ([4]uint64{}) {
		return 0, "masque vide au bloc suivant"
	}
	tis := b.dico[e.Composants]
	switch len(tis) {
	case 0:
		return 0, "masque inconnu"
	case 1:
		for ti := range tis {
			return ti, "resolu"
		}
	}
	return 0, "ambigu"
}
