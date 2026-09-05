package filmdec

// r11_journal_research_test.go — LE JOURNAL D'UNE VIE, AVEC UNE ANCRE, ET SA COLLECTE.
//
// CE QUI EST NEUF, ET POURQUOI CET INSTRUMENT EXISTE. R8 et R9 ont juge huit canaux sans
// disposer d'un seul instant d'usage certain du repulseur. L'utilisateur en a livre un, et il
// est NUMERIQUE : sur `72b0a25e`, JGtm ramasse un repulseur a 2:46 AVEC TROIS CHARGES. On ne
// cherche donc plus « un signal quelque part » mais une VALEUR qui vaut 3 au ramassage et
// decroit par pas de 1. Cet instrument ne juge rien : il PUBLIE, minute par minute, tout ce
// que les composants de capacite du bipede transmettent pour un joueur donne.
//
// CE QU'IL LIT, ET PAR QUI. Toutes les valeurs viennent des DESERIALISEURS DE PRODUCTION, par
// leurs hooks (`SetAbilitySetHook` i48, `SetAbilityEnergyHook` i56, `SetSpartanAbilityHook`
// i57, `SetAbilityNonPredictedHook` i59). Aucune relecture posee a cote d'eux : deux lecteurs
// du meme champ divergent le jour ou l'un des deux est corrige.
//
// LA LECTURE QUI MOTIVE LA MESURE. `ability_energy.go` porte i56 comme `R(3)` MASQUE + 7 bits
// PAR EMPLACEMENT ARME — TROIS emplacements. Et l'en-tete d'`i56_drops_test.go` consigne, du
// consommateur `FUN_140F8F300`, que la meme valeur 7 bits se lit de deux facons : continu
// `v / 127.0f`, ou discret `(v >> 4) & 0xF` charges ENTIERES + `(v & 0xF)` de recharge
// fractionnaire. Trois emplacements, un quartet de charges entieres, une ancre a 3 : le
// journal publie donc chaque valeur AUSSI en quartets, sans prejuger de l'encodage.
//
// PIEGE HERITE DE R8, RECOPIE EXPRES : `WorldObjectPrecision` est un GLOBAL DE PAQUET. Un
// instrument qui oublie `SetWorldObjectPrecisionFromLayout` desaligne les desers sans lever
// la moindre erreur.
//
// RESOLUTION PAR LA PALETTE, JAMAIS PAR UN NUMERO FIGE : `1cd3848a` porte le propulseur au
// rang 21 (famille B) la ou les films de R8 le portent au rang 5 (famille A). Le journal
// nomme les rangs par `abilityLabels` de l'artefact.
//
// GARDES : `R9_FILMS`, `R9_ARTIFACTS`, `R8_BOUNDS`, `R11_IDS` (obligatoire), `R11_XUID`
// (defaut JGtm), `R11_ALL` (1 = tous les joueurs). Aucune ecriture, aucune DuckDB,
// `CGO_ENABLED=0`. USAGE (depuis apps/go-api) :
//
//	CGO_ENABLED=0 R9_FILMS=<repo>/data/cache/film_chunks \
//	  R9_ARTIFACTS=<repo>/data/cache/replays/halo_infinite \
//	  R8_BOUNDS=<wt>/data/titles/halo_infinite/reference/map_quant_bounds.json \
//	  R11_IDS=72b0a25e go test ./internal/analysis/filmdec/ \
//	  -run '^TestR11Journal$' -count=1 -timeout 60m -v

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

const (
	r11IDsEnv  = "R11_IDS"
	r11XUIDEnv = "R11_XUID"
	r11AllEnv  = "R11_ALL"
)

// r11I48Names / r11I56Names : les deux etiquettes possibles de chaque composant (les films
// portent l'une OU l'autre, meme dualite que le dispatch de consumeByName).
var (
	r11I48Names = []string{"biped-desired-ability-set-component", "biped-desired-ability-set"}
	r11I56Names = []string{
		"biped-spartan-ability-energy-component", "biped-spartan-ability-energy",
	}
)

// r11RankRead est UNE lecture d'i48 : le compteur de rotation, et le rang de palette (ou
// AbilitySetNoRank quand la porte est ouverte — « ce joueur ne porte rien »).
type r11RankRead struct {
	Slot    uint32
	TSUS    uint64
	Counter uint32
	Rank    int
}

// r11EnergyRead est UNE lecture d'i56 : le masque R(3) et les trois emplacements
// (AbilityEnergyUnarmed quand le film ne transmet rien pour l'emplacement).
type r11EnergyRead struct {
	Slot uint32
	TSUS uint64
	Mask uint32
	Ch   [AbilityEnergyCharges]int
}

// r11Imp est UNE impulsion : le `tag == 1` d'i57 ou d'i59, le canal du propulseur (R8).
type r11Imp struct {
	Slot uint32
	TSUS uint64
	Src  string
}

// r11Reads est la recolte d'UN passage sur les paquets delta d'un film.
type r11Reads struct {
	Ranks  []r11RankRead
	Energy []r11EnergyRead
	Imp    []r11Imp
	Stat   map[string]int
}

// r11FilmDirs rend les dossiers de film a balayer. `R11_IDS` est OBLIGATOIRE : un film coute
// des dizaines de secondes de decodage, un balayage non borne serait un piege.
func r11FilmDirs(t *testing.T) []string {
	t.Helper()
	root := os.Getenv(r9FilmsEnv)
	if root == "" {
		t.Skipf("%s absent : instrument saute", r9FilmsEnv)
	}
	var out []string
	for _, s := range strings.Split(os.Getenv(r11IDsEnv), ",") {
		if s = strings.TrimSpace(s); s != "" {
			out = append(out, filepath.Join(root, s))
		}
	}
	if len(out) == 0 {
		t.Fatalf("%s obligatoire avec %s (un film coute cher a decoder)", r11IDsEnv, r9FilmsEnv)
	}
	return out
}

// r11XUID rend le xuid suivi (defaut : l'utilisateur, seul joueur dont le Theater montre les
// matchs).
func r11XUID() string {
	if v := strings.TrimSpace(os.Getenv(r11XUIDEnv)); v != "" {
		return v
	}
	return r9JGXUID
}

// r11Nib met une valeur 7 bits en quartets : `(v>>4)&0xF` charges entieres sous la lecture
// discrete, `v & 0xF` la recharge fractionnaire. Publie les DEUX lectures sans trancher.
func r11Nib(v int) string {
	if v == AbilityEnergyUnarmed {
		return "  ----"
	}
	return fmt.Sprintf("%3d=%d/%X", v, (v>>4)&0xF, v&0xF)
}

// r11EnergyTxt met en forme une lecture d'i56 : le masque R(3) puis les trois emplacements.
func r11EnergyTxt(mask uint32, ch [AbilityEnergyCharges]int) string {
	var b strings.Builder
	fmt.Fprintf(&b, "masque=%d%d%d", (mask>>2)&1, (mask>>1)&1, mask&1)
	for i := 0; i < AbilityEnergyCharges; i++ {
		fmt.Fprintf(&b, " e%d[%s]", i, r11Nib(ch[i]))
	}
	return b.String()
}

// r11Collect est LE SEUL balayage de ce lot (regle des <= 2 copies) : le journal et la mesure
// des charges le partagent. Un passage, quatre composants, les desers de PRODUCTION.
// L'appelant detient LockProcessDecode : les hooks sont des globaux de paquet.
func r11Collect(s r8MobSetup) r11Reads {
	idx := map[string]int{
		"i48": r8IndexOfAny(s.arch, r11I48Names),
		"i56": r8IndexOfAny(s.arch, r11I56Names),
		"i57": r8IndexOfAny(s.arch, r8I57Names),
		"i59": r8IndexOfAny(s.arch, r8I59Names),
	}
	out := r11Reads{Stat: map[string]int{}}
	var cur struct {
		slot uint32
		ts   uint64
	}
	prev48, prev56 := abilitySetHook, abilityEnergyHook
	prev57, prev59 := spartanAbilityHook, abilityNonPredictedHook
	SetAbilitySetHook(func(counter uint64, rank, _ int) {
		out.Stat["i48"]++
		out.Ranks = append(out.Ranks, r11RankRead{cur.slot, cur.ts, uint32(counter), rank})
	})
	SetAbilityEnergyHook(func(mask uint32, ch [AbilityEnergyCharges]int) {
		out.Stat["i56"]++
		out.Energy = append(out.Energy, r11EnergyRead{cur.slot, cur.ts, mask, ch})
	})
	SetSpartanAbilityHook(func(tag, _, _ uint64, _ bool) {
		out.Stat[fmt.Sprintf("i57tag%d", tag)]++
		if tag == 1 {
			out.Imp = append(out.Imp, r11Imp{cur.slot, cur.ts, "i57"})
		}
	})
	SetAbilityNonPredictedHook(func(st AbilityNonPredictedState) {
		out.Stat[fmt.Sprintf("i59tag%d", st.Tag)]++
		if st.Tag == 1 {
			out.Imp = append(out.Imp, r11Imp{cur.slot, cur.ts, "i59"})
		}
	})
	defer func() {
		SetAbilitySetHook(prev48)
		SetAbilityEnergyHook(prev56)
		SetSpartanAbilityHook(prev57)
		SetAbilityNonPredictedHook(prev59)
	}()
	r11Walk(s, idx, out.Stat, func(slot uint32, ts uint64) { cur.slot, cur.ts = slot, ts })
	return out
}

// r11Walk marche les records delta de bipede. Il ne parcourt QUE les records dont le masque
// annonce l'un des quatre composants suivis : parcourir les autres couterait le meme prix
// pour rien. `set` publie a l'appelant le slot et l'instant du record avant la marche — ce
// sont les hooks, declenches DANS la marche, qui remplissent la recolte.
func r11Walk(s r8MobSetup, idx, stat map[string]int, set func(uint32, uint64)) {
	minRecord := bipedHeaderBits + bipedIndexBits*bipedMinMaskCnt + s.lay.TotalBits()
	for _, c := range s.chunks {
		data, err := ReadFilmChunk(s.dir, c)
		if err != nil {
			continue
		}
		for _, pk := range WalkPackets(data) {
			if pk.Type != PacketTypeDelta {
				continue
			}
			pay := pk.Payload(data)
			total := len(pay) * 8
			for p := 0; p+minRecord <= total; {
				i0, slot, ids, ok := matchBipedHeader(pay, p, total, s.slots, true, s.lay)
				if !ok {
					p++
					continue
				}
				stat["records"]++
				if r11Announces(ids, idx, stat) {
					set(slot, pk.TimestampUS)
					walkRecordComponents(pay, i0, total, ids, s.lay, s.arch,
						func(int) bool { return true })
				}
				p = i0 + s.lay.TotalBits()
			}
		}
	}
}

// r11Announces dit si le masque annonce au moins un des composants suivis, et compte les
// annonces par composant — ce sont les DENOMINATEURS sans lesquels un compte de lectures ne
// se juge pas (une lecture manquante peut etre une non-annonce ou une marche interrompue).
func r11Announces(ids []int, idx, stat map[string]int) bool {
	any := false
	for _, k := range []string{"i48", "i56", "i57", "i59"} {
		if idx[k] >= 0 && maskHas(ids, idx[k]) {
			stat["masque"+k]++
			any = true
		}
	}
	return any
}

// r11Setup porte le contexte d'un film prepare : l'artefact, l'origine d'horloge et le
// balayage. Les deux instruments du lot le partagent.
type r11Setup struct {
	id     string
	art    *r9Art
	origin uint64
	scan   r8MobSetup
}

// r11Prepare resout un film : artefact, bornes de carte, precision monde, origine d'horloge.
// L'appelant DOIT detenir LockProcessDecode et restaurer WorldObjectPrecision en sortie.
func r11Prepare(t *testing.T, dir string) r11Setup {
	t.Helper()
	id := filepath.Base(dir)
	art := r9LoadArt(t, id)
	entry := r8MapEntry(t, dir)
	SetWorldObjectPrecisionFromLayout(entry.Layout())
	origin, ok := r9FirstPacketUS(dir, 1)
	if !ok {
		t.Fatalf("%s : chunk 1 illisible, aucune origine d'horloge", id)
	}
	return r11Setup{id: id, art: art, origin: origin, scan: r8MobResolve(t, dir)}
}

// ms convertit un horodatage moteur en temps du visionneur.
func (s r11Setup) ms(ts uint64) int64 { return (int64(ts) - int64(s.origin)) / 1000 }

func TestR11Journal(t *testing.T) {
	for _, dir := range r11FilmDirs(t) {
		r11JournalOneFilm(t, dir)
	}
}

// r11JLine est une ligne de journal prete a etre publiee.
type r11JLine struct {
	ms   int64
	slot uint32
	kind string
	txt  string
}

func r11JournalOneFilm(t *testing.T, dir string) {
	t.Helper()
	release := LockProcessDecode()
	defer release()
	saved := WorldObjectPrecision
	defer func() { WorldObjectPrecision = saved }()
	s := r11Prepare(t, dir)
	rd := r11Collect(s.scan)
	xuid, all := r11XUID(), os.Getenv(r11AllEnv) == "1"

	var lines []r11JLine
	add := func(slot uint32, ts uint64, kind, txt string) {
		ms := s.ms(ts)
		if !all && !r11IsTarget(s.art, slot, ms, xuid) {
			return
		}
		lines = append(lines, r11JLine{ms, slot, kind, txt})
	}
	for _, r := range rd.Ranks {
		add(r.Slot, r.TSUS, "i48", fmt.Sprintf("compteur=%d rang=%d (%s)",
			r.Counter, r.Rank, r11RankLabel(s.art, r.Rank)))
	}
	for _, e := range rd.Energy {
		add(e.Slot, e.TSUS, "i56", r11EnergyTxt(e.Mask, e.Ch))
	}
	for _, i := range rd.Imp {
		add(i.Slot, i.TSUS, i.Src, "tag=1 (impulsion)")
	}
	r11LogHeader(t, s, rd.Stat)
	sort.SliceStable(lines, func(a, b int) bool { return lines[a].ms < lines[b].ms })
	t.Logf("  JOURNAL — %d lignes", len(lines))
	for _, l := range lines {
		t.Logf("    %-7s slot=%-4d %-4s %s", r9MMSS(l.ms), l.slot, l.kind, l.txt)
	}
}

// r11RankLabel nomme un rang par la palette du film. Le rang AbilitySetNoRank est la PORTE
// OUVERTE : le film dit « ce joueur ne porte pas de capacite », ce n'est pas un defaut.
func r11RankLabel(art *r9Art, rank int) string {
	if rank == AbilitySetNoRank {
		return "porte ouverte : rien de porte"
	}
	return art.r9LabelOf(rank)
}

// r11IsTarget dit si le slot appartient au joueur suivi A CET INSTANT. Le slot MIGRE aux
// reapparitions : la jointure se fait slot x FRAME, jamais slot seul.
func r11IsTarget(art *r9Art, slot uint32, ms int64, xuid string) bool {
	frame := 0
	if art.FrameIntervalMs > 0 {
		frame = int((ms - art.OriginMs) / int64(art.FrameIntervalMs))
	}
	_, x, _ := art.r9WhoAt(slot, frame)
	return x == xuid
}

// r11LogHeader publie les denominateurs et la palette : sans eux, un journal vide ne se
// distingue pas d'un journal qu'on n'a pas su lire.
func r11LogHeader(t *testing.T, s r11Setup, stat map[string]int) {
	t.Helper()
	var keys []string
	for k := range stat {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, k := range keys {
		fmt.Fprintf(&b, " %s=%d", k, stat[k])
	}
	t.Logf("%s (%s) : originMs=%d pas=%d ms |%s", s.id, s.art.MatchID, s.art.OriginMs,
		s.art.FrameIntervalMs, b.String())
	t.Logf("  palette du film (par NOM, jamais par numero fige) : %s", r11Palette(s.art))
}

// r11Palette rend la palette du film, rang par rang, telle que l'artefact la nomme.
func r11Palette(art *r9Art) string {
	type kv struct {
		r int
		s string
	}
	var all []kv
	for k, lab := range art.AbilityLabels {
		if n, ok := r11Atoi(k); ok {
			all = append(all, kv{n, lab.EN})
		}
	}
	sort.Slice(all, func(a, b int) bool { return all[a].r < all[b].r })
	var parts []string
	for _, e := range all {
		parts = append(parts, fmt.Sprintf("%d=%s", e.r, e.s))
	}
	return strings.Join(parts, " ")
}

// r11Atoi lit un entier positif sans importer strconv pour une cle de map.
func r11Atoi(s string) (int, bool) {
	if s == "" {
		return 0, false
	}
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, false
		}
		n = n*10 + int(c-'0')
	}
	return n, true
}
