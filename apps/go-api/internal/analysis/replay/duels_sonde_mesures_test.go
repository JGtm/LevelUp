package replay

// duels_sonde_mesures_test.go — les mesures M1 a M4 de la sonde duels (scinde de
// duels_sonde_research_test.go pour le seuil de 500 lignes). Voir l'en-tete de ce fichier-la
// pour la question posee, les criteres et les seuils ecrits avant la mesure.

import (
	"fmt"
	"sort"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

// duelsPct formate un taux en gardant le couple brut visible — un pourcentage seul ne
// s'additionne pas d'un film a l'autre.
func duelsPct(n, d int) string {
	if d == 0 {
		return fmt.Sprintf("%d/0 (n.d.)", n)
	}
	return fmt.Sprintf("%d/%d = %.1f %%", n, d, 100*float64(n)/float64(d))
}

// duelsFatal rend le DERNIER degat recu par slot dans [ts-duelsKillWindowUS, ts], et sa presence.
// Le tueur est le responsable de ce degat : c'est la seule definition qui n'exige pas le
// kill-feed, donc la seule qui garde la sonde hors ligne.
func duelsFatal(dmg []duelDmg, slot uint32, ts uint64) (duelDmg, bool) {
	var best duelDmg
	found := false
	lo := uint64(0)
	if ts > duelsKillWindowUS {
		lo = ts - duelsKillWindowUS
	}
	for _, d := range dmg {
		if d.ts > ts {
			break // le flux est trie
		}
		if d.ts >= lo && d.victim == slot {
			best, found = d, true
		}
	}
	return best, found
}

// duelsExisteDegat dit si un degat de `de` vers `vers` existe dans [lo, hi].
func duelsExisteDegat(dmg []duelDmg, de, vers uint32, lo, hi uint64) bool {
	for _, d := range dmg {
		if d.ts > hi {
			break
		}
		if d.ts >= lo && d.attacker == de && d.victim == vers {
			return true
		}
	}
	return false
}

// duelsExisteDegatVersQuiconque dit si `de` a inflige un degat a n'importe qui dans [lo, hi].
func duelsExisteDegatVersQuiconque(dmg []duelDmg, de uint32, lo, hi uint64) bool {
	for _, d := range dmg {
		if d.ts > hi {
			break
		}
		if d.ts >= lo && d.attacker == de {
			return true
		}
	}
	return false
}

// duelsMesureReciprocite produit M1 et M2.
//
// M1 : part des morts dont la victime avait touche son tueur dans la fenetre.
// M2 : le meme compte avec la fenetre DEPLACEE de duelsShiftUS vers le passe — si la
// reciprocite tenait aussi la, elle ne dirait rien de l'echange. Un second controle,
// « la victime a-t-elle touche QUELQU'UN », mesure la SPECIFICITE : c'est lui qui distingue
// « elle ripostait » de « elle ripostait a SON TUEUR ».
func duelsMesureReciprocite(t *testing.T, morts []duelMort, dmg []duelDmg) {
	t.Helper()
	avecFatal, ecarts := 0, make([]int64, 0, len(morts))
	for _, m := range morts {
		if f, ok := duelsFatal(dmg, m.slot, m.ts); ok {
			avecFatal++
			ecarts = append(ecarts, int64(m.ts)-int64(f.ts))
		}
	}
	t.Logf("M1.0 morts avec un degat fatal attribue : %s", duelsPct(avecFatal, len(morts)))
	if len(ecarts) > 0 {
		sort.Slice(ecarts, func(i, j int) bool { return ecarts[i] < ecarts[j] })
		t.Logf("M1.0bis ecart degat fatal -> fin de vie : mediane %d ms, p90 %d ms "+
			"(temoin de la jointure : un ecart stable et court = les deux flux parlent du meme fait)",
			ecarts[len(ecarts)/2]/1000, ecarts[len(ecarts)*9/10]/1000)
	}

	for _, w := range duelsRiposteWindowsUS {
		duels, quiconque, temoin := 0, 0, 0
		for _, m := range morts {
			f, ok := duelsFatal(dmg, m.slot, m.ts)
			if !ok {
				continue
			}
			lo := duelsBorneBasse(m.ts, w)
			if duelsExisteDegat(dmg, m.slot, f.attacker, lo, m.ts) {
				duels++
			}
			if duelsExisteDegatVersQuiconque(dmg, m.slot, lo, m.ts) {
				quiconque++
			}
			if m.ts > duelsShiftUS {
				tHi := m.ts - duelsShiftUS
				if duelsExisteDegat(dmg, m.slot, f.attacker, duelsBorneBasse(tHi, w), tHi) {
					temoin++
				}
			}
		}
		t.Logf("M1 riposte a %d s : duels %s | riposte vers quiconque %s | M2 temoin decale %s",
			w/1_000_000, duelsPct(duels, avecFatal),
			duelsPct(quiconque, avecFatal), duelsPct(temoin, avecFatal))
	}
}

func duelsBorneBasse(ts, w uint64) uint64 {
	if ts > w {
		return ts - w
	}
	return 0
}

// duelsPremierEchange rend l'instant du PREMIER degat de l'echange entre a et b dans [lo, hi],
// dans un sens ou dans l'autre. C'est l'instant qui caracterise l'OUVERTURE du duel — celui ou
// la distance a un sens tactique, contrairement a l'instant du coup fatal.
func duelsPremierEchange(dmg []duelDmg, a, b uint32, lo, hi uint64) (uint64, bool) {
	for _, d := range dmg {
		if d.ts > hi {
			break
		}
		if d.ts < lo {
			continue
		}
		if (d.attacker == a && d.victim == b) || (d.attacker == b && d.victim == a) {
			return d.ts, true
		}
	}
	return 0, false
}

// duelsDistance rend la distance monde (metres) entre deux slots a l'instant ts, si les deux
// echantillons tombent dans la tolerance. Rend aussi le DENIVELE |dz| entre les deux.
//
// POURQUOI |dz| ET PAS « 3D moins plan » : la premiere version de cette sonde mesurait l'ecart
// entre distance 3D et distance plane, et concluait a un axe vertical nul (0,0-0,2 m). C'etait
// la mesure qui etait aveugle, pas l'axe : a 6 m de distance, un denivele d'UN METRE ne change la
// distance 3D que de 8 cm (sqrt(36+1) - 6). L'avantage de hauteur se lit sur |dz| lui-meme.
func duelsDistance(
	tracks map[uint32]slotTrack, a, b uint32, ts uint64,
) (d3, dz float64, ok bool) {
	ta, oka := tracks[a]
	tb, okb := tracks[b]
	if !oka || !okb {
		return 0, 0, false
	}
	pa, da := ta.at(ts)
	pb, db := tb.at(ts)
	if da > duelsPosTolUS || db > duelsPosTolUS || !pa.HasWorld || !pb.HasWorld {
		return 0, 0, false
	}
	// LES DEUX FORMULES SONT CELLES DU PAQUET (`dist3`, `planDist` — geometry.go), jamais une
	// copie : le garde-rail TestUneSeuleFormuleDeDistance3D couvre aussi les instruments,
	// parce que la copie qui divergerait viendrait justement d'un instrument.
	d3 = dist3([3]float32{pa.X, pa.Y, pa.Z}, [3]float32{pb.X, pb.Y, pb.Z})
	return d3, absF(float64(pa.Z - pb.Z)), true
}

func absF(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// duelsMesureDistance produit M3 : la resolvabilite de la distance d'OUVERTURE, son
// histogramme sur les bornes de production, et la comparaison duel <-> elimination.
//
// LA COMPARAISON EST L'ORACLE INDEPENDANT : si les eliminations (aucune riposte) mesurent
// SYSTEMATIQUEMENT plus court que les duels, la classification decrit bien un fait de jeu —
// on n'execute pas a 30 m, on execute au contact.
func duelsMesureDistance(t *testing.T, morts []duelMort, dmg []duelDmg, tracks map[uint32]slotTrack) {
	t.Helper()
	const w = 3_000_000
	var duels, elims []float64
	tentees, resolues, denivele, hauteurs := 0, 0, 0, make([]float64, 0, len(morts))
	buckets := make([]int, len(filmdec.WeaponHitDistanceEdges)+1)
	for _, m := range morts {
		f, ok := duelsFatal(dmg, m.slot, m.ts)
		if !ok {
			continue
		}
		lo := duelsBorneBasse(m.ts, w)
		estDuel := duelsExisteDegat(dmg, m.slot, f.attacker, lo, m.ts)
		at, ok := duelsPremierEchange(dmg, m.slot, f.attacker, lo, m.ts)
		if !ok {
			at = f.ts
		}
		tentees++
		d3, dz, ok := duelsDistance(tracks, m.slot, f.attacker, at)
		if !ok {
			continue
		}
		resolues++
		hauteurs = append(hauteurs, dz)
		if dz > 1.0 {
			denivele++
		}
		if estDuel {
			duels = append(duels, d3)
			buckets[filmdec.WeaponHitBucket(d3)]++
		} else {
			elims = append(elims, d3)
		}
	}
	t.Logf("M3 distance d'ouverture resolue : %s (tolerance %d ms)",
		duelsPct(resolues, tentees), duelsPosTolUS/1000)
	t.Logf("M3 mediane duels %s | mediane eliminations %s | denivele |dz| median %s | "+
		"engagements a plus d'1 m de denivele %s",
		duelsMediane(duels), duelsMediane(elims), duelsMediane(hauteurs),
		duelsPct(denivele, resolues))
	t.Logf("M3 histogramme des duels sur %v m : %v", filmdec.WeaponHitDistanceEdges, buckets)
}

// duelsMediane rend la mediane formatee, ou « n.d. » sur un echantillon vide.
func duelsMediane(v []float64) string {
	if len(v) == 0 {
		return "n.d. (0)"
	}
	s := append([]float64(nil), v...)
	sort.Float64s(s)
	return fmt.Sprintf("%.1f m (n=%d)", s[len(s)/2], len(s))
}

// duelsMesureTrio produit M4 : combien de duels restent un tete-a-tete, et combien voient un
// TIERS intervenir. Le tiers est defini par le DEGAT, pas par la proximite : un joueur qui
// passe a cote sans tirer n'a pas participe.
//
// RESERVE ASSUMEE : sans l'equipe, « un tiers est intervenu » ne se scinde pas en « renfort
// ami » et « renfort adverse ». C'est la mesure qui manque, pas la donnee — le pont slot ->
// xuid la rendrait, au prix du roster (donc de la base), que cette sonde s'interdit.
func duelsMesureTrio(t *testing.T, morts []duelMort, dmg []duelDmg) {
	t.Helper()
	const w = 3_000_000
	repartition := map[int]int{}
	total := 0
	for _, m := range morts {
		f, ok := duelsFatal(dmg, m.slot, m.ts)
		if !ok {
			continue
		}
		lo := duelsBorneBasse(m.ts, w)
		if !duelsExisteDegat(dmg, m.slot, f.attacker, lo, m.ts) {
			continue // pas un duel : hors population de M4
		}
		total++
		repartition[len(duelsTiers(dmg, m.slot, f.attacker, lo, m.ts))]++
	}
	t.Logf("M4 duels analyses : %d — tete-a-tete pur %s",
		total, duelsPct(repartition[0], total))
	for _, n := range duelsClesTriees(repartition) {
		if n == 0 {
			continue
		}
		t.Logf("M4 duels avec %d tiers implique(s) : %s", n, duelsPct(repartition[n], total))
	}
}

// duelsTiers rend les slots, autres que a et b, qui ont inflige ou subi un degat de l'un des
// deux dans [lo, hi].
func duelsTiers(dmg []duelDmg, a, b uint32, lo, hi uint64) map[uint32]bool {
	out := map[uint32]bool{}
	for _, d := range dmg {
		if d.ts > hi {
			break
		}
		if d.ts < lo {
			continue
		}
		for _, s := range [2]uint32{d.attacker, d.victim} {
			autre := d.attacker
			if s == d.attacker {
				autre = d.victim
			}
			if (s == a || s == b) && autre != a && autre != b {
				out[autre] = true
			}
		}
	}
	return out
}

// duelsClesTriees rend les cles d'une repartition dans l'ordre croissant (sortie reproductible).
func duelsClesTriees(m map[int]int) []int {
	out := make([]int, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Ints(out)
	return out
}

// duelsRecensementC0 compte les paquets 0xC0 par SOUS-TYPE (les 7 bits qui suivent les 2 bits
// d'en-tete). Cette mesure ne sert pas les duels : elle dit si la maigreur du flux de degats est
// une propriete DU FILM ou une propriete de NOTRE PORTE.
//
// POURQUOI ELLE EXISTE. ScanFilmWeaponDamages ne retient que le sous-type 0 (damage_aftermath)
// et jette explicitement le sous-type 1 (damage_section_response). Si le sous-type 1 est
// abondant, il y a un RESERVOIR non lu ; s'il est aussi rare que le 0, le film ne replique
// simplement pas chaque coup au but, et aucune amelioration de decodage n'y changera rien.
func duelsRecensementC0(t *testing.T, dir string) {
	t.Helper()
	n := filmdec.CountFilmChunks(dir)
	parType := map[uint64]int{}
	deltas, c0 := 0, 0
	for c := 1; c <= n; c++ {
		data, err := filmdec.ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, pk := range filmdec.WalkPackets(data) {
			if pk.Type != filmdec.PacketTypeDelta {
				continue
			}
			deltas++
			if pk.Size < 2 {
				continue
			}
			pay := pk.Payload(data)
			if pay[0] != 0xC0 {
				continue
			}
			c0++
			br := filmdec.NewBitReader(pay)
			br.Skip(2)
			parType[br.ReadBits(7)]++
		}
	}
	t.Logf("M0bis recensement : %d chunks, %d paquets delta, %d paquets 0xC0", n, deltas, c0)
	for _, k := range duelsClesTrieesU64(parType) {
		t.Logf("M0bis   0xC0 sous-type %d : %d paquets", k, parType[k])
	}
}

// duelsClesTrieesU64 rend les cles d'un recensement dans l'ordre croissant.
func duelsClesTrieesU64(m map[uint64]int) []uint64 {
	out := make([]uint64, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// duelsChuteEps : baisse minimale de fraction de bouclier retenue comme une CHUTE. Le quantum
// de la source vaut 1/64 = 0,0156 ; 0,02 est juste au-dessus, donc aucune vraie baisse n'est
// perdue et aucun aller-retour de quantification n'est compte.
const duelsChuteEps = 0.02

// duelChute : une baisse de bouclier, datee, sur un slot.
type duelChute struct {
	slot uint32
	ts   uint64
}

// duelsChutesBouclier rend les baisses de bouclier de chaque slot.
//
// POURQUOI CETTE MESURE EXISTE, ET C'EST LE COEUR DE LA SONDE. Le flux `damage_aftermath` est
// mesure ICI a 1 a 5 evenements par mort : trop maigre pour voir un ECHANGE, qui en demande
// DEUX. Le bouclier, lui, est replique DANS LE RECORD DE POSITION a chaque fois qu'il CHANGE —
// deux ordres de grandeur plus dense. Une chute de bouclier EST un degat subi ; elle ne dit pas
// QUI l'a inflige, mais elle dit QUE le joueur en a pris, et c'est la moitie manquante de la
// reciprocite. La question que M5 tranche : cette voie est-elle assez dense et assez fiable ?
func duelsChutesBouclier(
	positions []filmdec.BipedPosition, vies map[uint32][]lifeSpan,
) ([]duelChute, int) {
	parSlot := map[uint32][]filmdec.BipedPosition{}
	for _, p := range positions {
		if _, ok := p.ShieldAt(); ok {
			parSlot[p.Slot] = append(parSlot[p.Slot], p)
		}
	}
	var out []duelChute
	lectures := 0
	for slot, ps := range parSlot {
		sort.Slice(ps, func(i, j int) bool { return ps[i].TimestampUS < ps[j].TimestampUS })
		lectures += len(ps)
		prec, precTS, ouvert := float32(0), uint64(0), false
		for _, p := range ps {
			v, _ := p.ShieldAt()
			// Un trou superieur a lifeGapUS = nouvelle vie : le bouclier repart plein, la
			// comparaison avec la valeur d'avant serait une fausse chute (cf. lifeGapUS).
			if ouvert && p.TimestampUS-precTS <= lifeGapUS && float64(prec-v) > duelsChuteEps {
				out = append(out, duelChute{slot: slot, ts: p.TimestampUS})
			}
			prec, precTS, ouvert = v, p.TimestampUS, true
		}
		_ = vies
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ts < out[j].ts })
	return out, lectures
}

// duelsMesureBouclier produit M5 : la voie de reprise. Trois quantites, dans cet ordre.
//
//	densite      — chutes par mort, a comparer aux 1-5 degats par mort du flux 0xC0.
//	oracle       — part des morts dont LA VICTIME a une chute dans les 3 s. C'est un controle
//	               de validite, pas un resultat : un mourant a forcement perdu son bouclier.
//	               S'il est bas, le canal n'est pas exploitable et le reste ne vaut rien.
//	discrimination — combien d'AUTRES slots vivants ont une chute dans la meme fenetre. C'est
//	               la mesure decisive : a 1 ou 2, une chute designe presque un adversaire ; a 6,
//	               elle ne designe personne et il faudra un second signal pour trancher.
func duelsMesureBouclier(
	t *testing.T, morts []duelMort, chutes []duelChute, lectures int, vies map[uint32][]lifeSpan,
) {
	t.Helper()
	const w = 3_000_000
	t.Logf("M5 densite : %d lectures de bouclier, %d chutes, %d morts -> %.1f chutes par mort",
		lectures, len(chutes), len(morts), float64(len(chutes))/float64(max1(len(morts))))

	oracle, repartition := 0, map[int]int{}
	for _, m := range morts {
		lo := duelsBorneBasse(m.ts, w)
		autres := map[uint32]bool{}
		for _, c := range chutes {
			if c.ts > m.ts {
				break
			}
			if c.ts < lo {
				continue
			}
			if c.slot == m.slot {
				continue
			}
			if duelsVivant(vies, c.slot, c.ts) {
				autres[c.slot] = true
			}
		}
		if duelsChuteDe(chutes, m.slot, lo, m.ts) {
			oracle++
		}
		repartition[len(autres)]++
	}
	t.Logf("M5 oracle : la victime a une chute de bouclier dans les 3 s avant sa mort : %s",
		duelsPct(oracle, len(morts)))
	for _, n := range duelsClesTriees(repartition) {
		t.Logf("M5 discrimination : %d autre(s) slot(s) en chute dans la fenetre : %s",
			n, duelsPct(repartition[n], len(morts)))
	}
}

// duelsChuteDe dit si le slot a une chute de bouclier dans [lo, hi].
func duelsChuteDe(chutes []duelChute, slot uint32, lo, hi uint64) bool {
	for _, c := range chutes {
		if c.ts > hi {
			break
		}
		if c.ts >= lo && c.slot == slot {
			return true
		}
	}
	return false
}

// max1 evite une division par zero dans les densites.
func max1(n int) int {
	if n <= 0 {
		return 1
	}
	return n
}
