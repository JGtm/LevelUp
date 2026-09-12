package filmdec

// rtpc_ti10_census_test.go — LE JOURNAL PAR FILM, L'INVENTAIRE DES IDENTIFIANTS (gate 2) et
// L'EPREUVE D'EGALITE contre les identifiants attendus (gate 3). Le CRITERE est ecrit dans
// l'en-tete de `rtpc_ti10_assaut_test.go` ; ce fichier ne fait que publier ce qui a ete lu.

import (
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"
)

// ti10RobusteSlots / ti10RobusteFilms : le filtre anti-bruit d'ancrage, fige avant la mesure
// (lecon du lot C : 811 identifiants rares s'y etaient reveles etre du bruit).
const (
	ti10RobusteSlots = 2
	ti10RobusteFilms = 2
)

// ti10AttendusBanque : les identifiants ATTENDUS, hachages FNV-1 32 bits des noms d'evenements
// de la banque `sb_004_mod_mp_assault` (2b01f208), plus les cinq oracles deja casses par la
// reconnaissance et les deux identifiants mesures en Strongholds au lot C.
//
// LES CINQ PREMIERS SONT DES ORACLES, PAS DES CANDIDATS : leurs noms et leurs identifiants sont
// tous deux connus, et `TestRtpcTi10Hachage` s'en sert pour prouver que le hacheur de ce fichier
// est bien `FUN_140871f08`. Les onze suivants sont les candidats d'armement.
//
// CE QUE VAUT UNE ABSENCE : RIEN. Ce sont des identifiants d'EVENEMENTS Wwise, pas de RTPC.
// L'egalite est testee parce qu'elle est gratuite et qu'une prise nommerait le canal.
var ti10AttendusBanque = []struct {
	id  uint32
	nom string
}{
	{0x984f65e5, "play_004_mod_mp_assault_bomb_detonated (ORACLE)"},
	{0xa38d8b3e, "play_004_mod_mp_assault_bomb_planted_loop (ORACLE)"},
	{0xb57933f2, "play_004_mod_mp_assault_bomb_disarm_loop (ORACLE)"},
	{0xe8ca00b8, "play_004_mod_mp_assault_bomb_spawn (ORACLE)"},
	{0x4cf90163, "play_004_mod_mp_assault_bomb_despawn (ORACLE)"},
	{0x3380093c, "play_004_mod_mp_assault_bomb_arm_loop"},
	{0x1c84ced0, "play_004_mod_mp_assault_bomb_arm_loop_team"},
	{0x23717795, "play_004_mod_mp_assault_bomb_arm_loop_enemy"},
	{0x510bb0a6, "play_004_mod_mp_assault_bomb_arming_loop"},
	{0x910fe602, "play_004_mod_mp_assault_bomb_arming_loop_team"},
	{0xc0b4a7d7, "play_004_mod_mp_assault_bomb_arming_loop_enemy"},
	{0xdf97d304, "play_004_mod_mp_assault_bomb_armed"},
	{0xc0a161ea, "play_004_mod_mp_assault_bomb_arm_start"},
	{0x963bf341, "play_004_mod_mp_assault_bomb_countdown_loop"},
	{0xbf87d5e1, "play_004_mod_mp_assault_bomb_reset_loop"},
	{0x6df703dd, "play_004_mod_mp_assault_bomb_resetting_loop"},
	{0x06854540, "identifiant mesure en Strongholds (lot C, nom inconnu)"},
	{0x7cbf0066, "identifiant mesure en Strongholds (lot C, nom inconnu)"},
}

// ti10FNV1 rend `FNV-1` 32 bits du nom MINUSCULE — la fabrique d'identifiants Wwise lue dans le
// binaire (`FUN_140871f08` : base 0x811C9DC5, puis h = h*0x01000193 ^ c, multiplication PUIS
// xor). C'est litteralement `AK::SoundEngine::GetIDFromString`.
func ti10FNV1(nom string) uint32 {
	h := uint32(0x811C9DC5)
	for i := 0; i < len(nom); i++ {
		c := nom[i]
		if c >= 'A' && c <= 'Z' {
			c += 32
		}
		h = h*0x01000193 ^ uint32(c)
	}
	return h
}

// TestRtpcTi10Hachage prouve le hacheur sur les CINQ oracles dont le nom ET l'identifiant sont
// connus. PUR : aucun film, aucune garde d'environnement. Si ce test rougit, la table
// `ti10AttendusBanque` ne vaut rien et le gate 3 non plus.
func TestRtpcTi10Hachage(t *testing.T) {
	for _, a := range ti10AttendusBanque {
		if !strings.HasSuffix(a.nom, "(ORACLE)") {
			continue
		}
		nom := strings.TrimSuffix(a.nom, " (ORACLE)")
		if got := ti10FNV1(nom); got != a.id {
			t.Errorf("FNV-1(%q) = %08x, attendu %08x", nom, got, a.id)
		}
	}
}

// ti10JournalFilm publie ce qu'un film a rendu.
func ti10JournalFilm(t *testing.T, b *ti10FilmBilan) {
	t.Helper()
	sc := b.sc
	t.Logf("%-9s %-26s slots observes %d (bande comblee %d) · %d vie(s) recensee(s)",
		b.id, b.mode, sc.SlotsObserves, sc.SlotsBande, sc.KeyCensus)
	t.Logf("           DELTA %d ancres, %d marches, %d cassees, %d chainees (%.1f %%) | "+
		"IMAGE-CLE %d records, %d marches, %d cassees, %d chainees",
		sc.Records, sc.Walked, sc.Broken, sc.Chained, ti11Part(sc.Chained, sc.Walked),
		sc.KeyRecords, sc.KeyWalked, sc.KeyBroken, sc.KeyChained)
	t.Logf("           LECTURES rtpc : %d dont %d chainees · %d a identifiant NUL (emplacement "+
		"libere) · %d identifiant(s) distinct(s) · %d serie(s) (slot, id) · %d montee(s)",
		b.lectures, b.chainees, b.nuls, len(b.ids), b.seriesN, len(b.montees))
	for _, l := range ti10LignesID(b) {
		t.Logf("           %s", l)
	}
	if sc.Tronque {
		t.Logf("           ATTENTION : plafond de %d lectures ATTEINT — la recolte est tronquee",
			ti10MaxLectures)
	}
	if sc.PaquetsSansHorloge > 0 {
		t.Logf("           %d paquet(s) sans horloge (chunk absent du manifeste) — ecartes",
			sc.PaquetsSansHorloge)
	}
	t.Logf("           PREMIER COMPOSANT BLOQUANT : %s · balayage en %s",
		ti12Bloquants(sc.Bloque), b.duree.Round(time.Second))
	for _, e := range b.extraits {
		t.Logf("           %s", e)
	}
}

// ti10LignesID rend au plus huit lignes d'identifiants, les plus fournis d'abord.
func ti10LignesID(b *ti10FilmBilan) []string {
	ids := make([]uint32, 0, len(b.ids))
	for id := range b.ids {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool {
		if b.ids[ids[i]].lectures != b.ids[ids[j]].lectures {
			return b.ids[ids[i]].lectures > b.ids[ids[j]].lectures
		}
		return ids[i] < ids[j]
	})
	out := make([]string, 0, 9)
	for i, id := range ids {
		if i == 8 {
			out = append(out, fmt.Sprintf("ID (+%d identifiant(s) de moindre trafic)", len(ids)-8))
			break
		}
		st := b.ids[id]
		out = append(out, fmt.Sprintf("ID %08x : %d lecture(s) (%d chainees) · %d slot(s) · "+
			"comp %s · quanta [%d, %d] = [%.2f, %.2f] · %d distinct(s) · %d montee(s)",
			id, st.lectures, st.chainees, len(st.slots), ti10Comps(st.comps),
			st.qMin, st.qMax, ManagedObjectRTPCValue(uint64(st.qMin)),
			ManagedObjectRTPCValue(uint64(st.qMax)), len(st.distinct), st.montees))
	}
	return out
}

// ti10Comps rend la liste triee des index de composant porteurs (-1 = voie image-cle).
func ti10Comps(m map[int16]bool) string {
	ks := make([]int, 0, len(m))
	for k := range m {
		ks = append(ks, int(k))
	}
	sort.Ints(ks)
	var sb strings.Builder
	for i, k := range ks {
		if i > 0 {
			sb.WriteString("/")
		}
		if k < 0 {
			sb.WriteString("cle")
			continue
		}
		fmt.Fprintf(&sb, "i%d", k)
	}
	return sb.String()
}

// ti10Extraits rend EN CLAIR les series des trois couples les plus fournis. Une jauge se
// reconnait a l'oeil ; un bruit uniforme aussi.
func ti10Extraits(series map[ti10Cle][]ti10Ech) []string {
	type l struct {
		cle ti10Cle
		n   int
	}
	ls := make([]l, 0, len(series))
	for c, v := range series {
		ls = append(ls, l{c, len(v)})
	}
	sort.Slice(ls, func(a, c int) bool {
		if ls[a].n != ls[c].n {
			return ls[a].n > ls[c].n
		}
		if ls[a].cle.slot != ls[c].cle.slot {
			return ls[a].cle.slot < ls[c].cle.slot
		}
		return ls[a].cle.id < ls[c].cle.id
	})
	out := make([]string, 0, 3)
	for i, x := range ls {
		if i == 3 {
			break
		}
		s := series[x.cle]
		sort.Slice(s, func(a, c int) bool { return s[a].tMS < s[c].tMS })
		out = append(out, fmt.Sprintf("SERIE slot %d id %08x (%d ech.) : %s",
			x.cle.slot, x.cle.id, x.n, ti10EnClair(s)))
	}
	return out
}

// ti10EnClair rend au plus 20 echantillons d'une serie, « t_s=quantum ».
func ti10EnClair(s []ti10Ech) string {
	var sb strings.Builder
	pas := 1
	if len(s) > 20 {
		pas = len(s) / 20
	}
	n := 0
	for i := 0; i < len(s) && n < 20; i += pas {
		if n > 0 {
			sb.WriteString(" ")
		}
		fmt.Fprintf(&sb, "%.1fs=%d", float64(s[i].tMS)/1000, s[i].q)
		n++
	}
	if len(s) > 20 {
		sb.WriteString(" ...")
	}
	return sb.String()
}

// ti10IDGlobal agrege UN identifiant sur tout le corpus.
type ti10IDGlobal struct {
	id                             uint32
	lecturesAssaut, lecturesTemoin int
	monteesAssaut                  int
	filmsAssaut, filmsTemoin       []string
	slotsAssaut                    map[uint32]bool
	comps                          map[int16]bool
	qMin, qMax                     uint32
}

// ti10Inventaire est le resultat du gate 2 : le tri des identifiants en trois colonnes.
type ti10Inventaire struct {
	tous map[uint32]*ti10IDGlobal
	// propres : vus en Assaut, ABSENTS des quatre temoins, et robustes (>= 2 slots, >= 2 films).
	propres []uint32
	// communs : vus en Assaut ET sur au moins un temoin.
	communs []uint32
	// rares : vus en Assaut seulement, mais sous le filtre de robustesse.
	rares []uint32
}

// ti10Gate2 inventorie les identifiants et les trie en propres / communs / rares.
func ti10Gate2(t *testing.T, bs []*ti10FilmBilan) *ti10Inventaire {
	t.Helper()
	inv := &ti10Inventaire{tous: map[uint32]*ti10IDGlobal{}}
	for _, b := range bs {
		temoin := strings.HasPrefix(b.mode, "TEMOIN")
		for id, st := range b.ids {
			inv.ajouter(b.id, id, st, temoin)
		}
	}
	inv.trier()
	t.Logf("########## GATE 2 INVENTAIRE — %d identifiant(s) distinct(s) sur le corpus : "+
		"%d PROPRE(S) A L'ASSAUT (robustes), %d commun(s) avec un temoin, %d rare(s) sous le "+
		"filtre (>= %d slots ET >= %d films)", len(inv.tous), len(inv.propres), len(inv.communs),
		len(inv.rares), ti10RobusteSlots, ti10RobusteFilms)
	ti10JournalInventaire(t, inv, "PROPRES A L'ASSAUT", inv.propres)
	ti10JournalInventaire(t, inv, "COMMUNS AVEC UN TEMOIN", inv.communs)
	ti10JournalInventaire(t, inv, "RARES (sous le filtre — bruit d'ancrage presume)", inv.rares)
	switch {
	case len(inv.tous) == 0:
		t.Logf("VERDICT GATE 2 : NEGATIF NET — aucun identifiant de rtpc lu, la piste se ferme.")
	case len(inv.propres) == 0:
		t.Logf("VERDICT GATE 2 : NEGATIF PROPRE — aucun identifiant robuste n'est propre a " +
			"l'Assaut ; les rtpc lus sont des canaux generiques d'objet gere.")
	default:
		t.Logf("VERDICT GATE 2 : %d canal(aux) de mode candidat(s) — gate 4 leur est applique.",
			len(inv.propres))
	}
	return inv
}

// ajouter range un identifiant d'un film dans l'inventaire.
func (inv *ti10Inventaire) ajouter(film string, id uint32, st *ti10IDStat, temoin bool) {
	g, ok := inv.tous[id]
	if !ok {
		g = &ti10IDGlobal{id: id, slotsAssaut: map[uint32]bool{}, comps: map[int16]bool{},
			qMin: ^uint32(0)}
		inv.tous[id] = g
	}
	for c := range st.comps {
		g.comps[c] = true
	}
	if st.qMin < g.qMin {
		g.qMin = st.qMin
	}
	if st.qMax > g.qMax {
		g.qMax = st.qMax
	}
	if temoin {
		g.lecturesTemoin += st.lectures
		g.filmsTemoin = append(g.filmsTemoin, film)
		return
	}
	g.lecturesAssaut += st.lectures
	g.monteesAssaut += st.montees
	g.filmsAssaut = append(g.filmsAssaut, film)
	for s := range st.slots {
		g.slotsAssaut[s] = true
	}
}

// trier repartit les identifiants en trois colonnes, chacune triee par trafic decroissant.
func (inv *ti10Inventaire) trier() {
	for id, g := range inv.tous {
		switch {
		case len(g.filmsAssaut) == 0:
			continue // vu sur un temoin seulement — hors sujet, pas publie
		case len(g.filmsTemoin) > 0:
			inv.communs = append(inv.communs, id)
		case len(g.slotsAssaut) >= ti10RobusteSlots && len(g.filmsAssaut) >= ti10RobusteFilms:
			inv.propres = append(inv.propres, id)
		default:
			inv.rares = append(inv.rares, id)
		}
	}
	tri := func(s []uint32) {
		sort.Slice(s, func(i, j int) bool {
			a, b := inv.tous[s[i]], inv.tous[s[j]]
			if a.lecturesAssaut != b.lecturesAssaut {
				return a.lecturesAssaut > b.lecturesAssaut
			}
			return s[i] < s[j]
		})
	}
	tri(inv.propres)
	tri(inv.communs)
	tri(inv.rares)
}

// ti10JournalInventaire publie une colonne de l'inventaire (au plus douze lignes).
func ti10JournalInventaire(t *testing.T, inv *ti10Inventaire, titre string, ids []uint32) {
	t.Helper()
	t.Logf("--- %s : %d identifiant(s)", titre, len(ids))
	for i, id := range ids {
		if i == 12 {
			t.Logf("    (+%d identifiant(s) de moindre trafic)", len(ids)-12)
			return
		}
		g := inv.tous[id]
		t.Logf("    %08x : Assaut %d lecture(s) sur %d film(s), %d slot(s), %d montee(s) · "+
			"temoin %d lecture(s) sur %d film(s) · comp %s · quanta [%d, %d]",
			id, g.lecturesAssaut, len(g.filmsAssaut), len(g.slotsAssaut), g.monteesAssaut,
			g.lecturesTemoin, len(g.filmsTemoin), ti10Comps(g.comps), g.qMin, g.qMax)
	}
}

// ti10Gate3 confronte les identifiants LUS aux identifiants ATTENDUS, par simple egalite.
func ti10Gate3(t *testing.T, inv *ti10Inventaire) {
	t.Helper()
	prises := 0
	for _, a := range ti10AttendusBanque {
		g, ok := inv.tous[a.id]
		if !ok {
			continue
		}
		prises++
		t.Logf("    PRISE %08x = %s — Assaut %d lecture(s) sur %d film(s), temoin %d sur %d",
			a.id, a.nom, g.lecturesAssaut, len(g.filmsAssaut), g.lecturesTemoin,
			len(g.filmsTemoin))
	}
	t.Logf("########## GATE 3 EGALITE — %d prise(s) sur %d identifiant(s) attendus confrontes a "+
		"%d identifiant(s) lus", prises, len(ti10AttendusBanque), len(inv.tous))
	if prises == 0 {
		t.Logf("VERDICT GATE 3 : AUCUNE PRISE — et cela NE PROUVE RIEN. Les attendus sont des " +
			"identifiants d'EVENEMENTS Wwise, pas de RTPC ; le nommage du canal reste a faire " +
			"par le hachage du pool Lua des tags hsc*, hors perimetre de ce lot.")
	}
}
