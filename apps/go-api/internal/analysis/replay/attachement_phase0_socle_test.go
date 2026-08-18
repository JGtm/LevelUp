package replay

// attachement_phase0_socle_test.go — PHASE 0 du plan
// `.ai/V7.5/replay2d/PLAN_ATTACHEMENT_PARENT_STATE.md` : LE SOCLE DE LECTURE D'i10.
//
// CE QUE LA PHASE 0 CHERCHE. Le composant `object-parent-state-component` (i10) est porté
// par le bipède (ti=35), l'équipement (37), le corps rigide (38), le véhicule (40), l'arme
// au sol (42) et trois archétypes inconnus. Son déserialiseur EXISTE et est bit-exact
// (`filmdec.consumeObjectParentState`, miroir de `FUN_140c1e4d0`) — mais il consommait et
// JETAIT chacune de ses valeurs. La phase 0 les fait sortir et demande à deux oracles si
// l'une d'elles est le lien parent-enfant : le drapeau de CTF tenu par son porteur, le
// Spartan assis dans un véhicule.
//
// LE CHEMIN EST LE CHEMIN DELTA, ET C'EST UNE DÉCISION DU PLAN (décision 1). On ne relit
// aucune image-clé : la marche stateful `DecodeFrameViews` est celle de la production, elle
// résout l'archétype de chaque record par le World, et elle date chaque lecture à l'instant
// du paquet. Les images-clés servent seulement à réamorcer le World, comme partout ailleurs.
//
// LE RATTACHEMENT D'UNE LECTURE À SON RECORD EST EXACT, PAS APPROCHÉ. La sonde publie la
// position de bit à laquelle i10 a commencé ; la trace du record publie, pour chaque
// composant, son `StartBit`. Les deux se joignent par égalité — jamais par voisinage. C'est
// ce qui rend la mesure insensible aux essais de décodage que l'inférence de chaîne fait et
// jette : une lecture d'essai n'a aucun record propre qui la réclame, et elle est comptée à
// part (« orphelines »).
//
// GARDE : `ATT_FILM` porte la RACINE du cache film (le répertoire qui contient
// `film_chunks/` et `film_manifests/`). Sans elle, toute la phase 0 se saute proprement —
// les films ne sont pas versionnés, rien de ceci ne peut tourner en CI.
//
//	CGO_ENABLED=0 ATT_FILM=<depot>/data/cache \
//	  go test ./internal/analysis/replay/ -run Attachement -v -timeout 60m
//
// LECTURE SEULE : aucun de ces fichiers n'écrit quoi que ce soit, n'ouvre aucune base et ne
// touche à aucun chemin de production. La seule modification de production de la phase 0 est
// la sonde elle-même (`filmdec.SetObjectParentStateHook`), qui ne lit pas un bit de plus.

import (
	"fmt"
	"os"
	"sort"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

// attFilmEnv — la garde d'environnement de toute la phase 0 attachement.
const attFilmEnv = "ATT_FILM"

// attI10Component est l'étiquette de registre d'i10, celle par laquelle le traverseur le
// dispatche (`consumeByName`, traverse.go) et celle que porte `CompResult.Name`.
const attI10Component = "object-parent-state-component"

// attVuesParPaquet : nombre de vues de réplication déroulées par paquet, même réglage que la
// marche de production du `killsource` et que l'item 0.2 des objectifs.
const attVuesParPaquet = 4

// attRequireRoot rend la racine du cache film, ou saute le test.
func attRequireRoot(t *testing.T) string {
	t.Helper()
	root := os.Getenv(attFilmEnv)
	if root == "" {
		t.Skipf("mesure non demandée : %s vide (racine du cache film)", attFilmEnv)
	}
	return root
}

// attI10 est UNE lecture d'i10, rattachée à son record.
type attI10 struct {
	// TS est l'horloge du FILM en microsecondes (celle du paquet delta).
	TS uint64
	// Chunk / Packet localisent le paquet ; BitPos le composant dans son payload.
	Chunk, Packet int
	// Slot / Gen identifient la vie de l'entité — LA PAIRE, le pool de slots reboucle.
	Slot, Gen uint32
	// TI est l'archétype résolu par le World pour ce slot.
	TI uint32
	// St est la lecture brute publiée par la sonde.
	St filmdec.ObjectParentState

	// ParentSlot est `Quant16 & 0x1FFF` : le champ de 13 bits de `readQuantStat`, lu comme un
	// slot d'entité. C'est L'HYPOTHÈSE, pas un fait — elle vaut par ce qui suit.
	ParentSlot uint32
	// ParentTI / ParentLie disent si ce slot était LIÉ dans le World au moment de la lecture,
	// et à quel archétype. C'est le contrôle qui juge l'hypothèse SANS oracle : un champ qui
	// désigne vraiment une entité pointe sur une entité vivante ; un champ mal lu tombe sur
	// des slots morts à la fréquence du hasard.
	ParentTI  uint32
	ParentLie bool
	// Lies : nombre de slots liés dans le World à l'instant de la lecture. Sans lui, « le
	// handle tombe sur une entité vivante » n'a pas de taux de hasard auquel se comparer.
	Lies int
	// TemoinLie est le MÊME test appliqué à un slot DÉCORRÉLÉ du champ lu, dans le MÊME
	// World, par le MÊME appel : c'est le témoin de l'item 0.1, et il passe par le même
	// code que la mesure — sans quoi il contrôlerait autre chose qu'elle.
	TemoinLie bool
	// Propre dit que la lecture vient d'un paquet BIT-EXACT : toutes ses vues de réplication
	// ont atteint leur marqueur de fin et aucun de ses records n'a désynchronisé. C'est la
	// SEULE garantie d'alignement disponible hors ligne — un record dont tous les composants
	// sont portés peut consommer la mauvaise largeur sans que rien ne le signale, et la
	// preuve n'arrive qu'au marqueur de fin de la vue, qui ne tombe juste que si TOUTES les
	// largeurs qui le précèdent étaient justes.
	Propre bool
}

// attStat compte ce que le balayage a rencontré. Sans ces dénominateurs, aucune
// distribution de valeurs ne se juge.
type attStat struct {
	// Paquets delta déroulés, records rendus, records sans désynchronisation.
	Paquets, Records, RecordsPropres int
	// Lectures rattachées à un record propre ; Orphelines : publiées par la sonde et
	// réclamées par aucun record propre (essais de décodage de l'inférence de chaîne).
	Lectures, Orphelines int
	// ParTI / AttachesParTI : lectures et lectures à porte ouverte, par archétype.
	ParTI, AttachesParTI map[uint32]int
	// Desync : records désynchronisés.
	Desync int
	// PaquetsPropres : paquets dont toutes les vues ont atteint leur marqueur de fin sans
	// qu'aucun record ne désynchronise ; LecturesPropres : les lectures qui en viennent.
	PaquetsPropres, LecturesPropres int
}

// attNewStat rend un compteur prêt à l'emploi.
func attNewStat() attStat {
	return attStat{ParTI: map[uint32]int{}, AttachesParTI: map[uint32]int{}}
}

// attScanI10 déroule la marche stateful sur tout le film et rend chaque lecture d'i10.
//
// UN SEUL DÉCODAGE filmdec À LA FOIS PAR PROCESS : la sonde est un global de paquet. Elle
// est posée et retirée ici, sous `LockProcessDecode`.
func attScanI10(dir string) ([]attI10, attStat, error) {
	st := attNewStat()
	brut, err := filmdec.ReadFilmChunk(dir, 0)
	if err != nil {
		return nil, st, fmt.Errorf("registre (chunk_00) illisible : %w", err)
	}
	reg, err := filmdec.ParseRegistryChunk(brut)
	if err != nil {
		return nil, st, fmt.Errorf("registre illisible : %w", err)
	}
	release := filmdec.LockProcessDecode()
	defer release()

	// La sonde écrit dans `vues`, indexée PAR POSITION DE BIT : une position réécrite est
	// un record re-décodé (réparation de composant non porté), et c'est la DERNIÈRE lecture
	// qui vaut — celle dont l'alignement a été retenu.
	vues := map[int]filmdec.ObjectParentState{}
	filmdec.SetObjectParentStateHook(func(s filmdec.ObjectParentState) { vues[s.StartBit] = s })
	defer filmdec.SetObjectParentStateHook(nil)

	cfg := filmdec.DefaultFrameConfig()
	w := filmdec.NewWorld(reg)
	var out []attI10
	n := filmdec.CountFilmChunks(dir)
	for c := 1; c <= n; c++ {
		data, err := filmdec.ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, p := range filmdec.WalkPackets(data) {
			pay := p.Payload(data)
			if p.Type == filmdec.PacketTypeKeyframe {
				w = filmdec.WorldFromKeyframe(reg, pay)
				continue
			}
			if p.Type != filmdec.PacketTypeDelta {
				continue
			}
			st.Paquets++
			for k := range vues {
				delete(vues, k)
			}
			recs, vues0 := filmdec.DecodeFrameViews(pay, w, cfg, attVuesParPaquet, cfg.PacketPreambleBits)
			out = append(out, attCollecte(recs, vues, &st, p, c, w, attPaquetPropre(recs, vues0))...)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].TS < out[j].TS })
	return out, st, nil
}

// attCollecte rattache les lectures publiées d'UN paquet aux records propres qui les
// réclament, et compte tout ce qui ne trouve pas preneur.
//
// LE PARENT EST RÉSOLU DANS L'ÉTAT DU WORLD À LA FIN DU PAQUET, pas au bit près de la
// lecture. C'est une approximation, et elle est dans le sens PRUDENT : une entité créée
// entre la lecture et la fin du paquet ne peut que faire passer un handle pour lié alors
// qu'il ne l'était pas encore — jamais l'inverse. L'écart est d'un paquet (~0,5 s mesuré).
func attCollecte(
	recs []filmdec.FrameRecord, vues map[int]filmdec.ObjectParentState,
	st *attStat, p filmdec.FilmPacket, chunk int, w *filmdec.World, propre bool,
) []attI10 {
	var out []attI10
	reclamees := 0
	if propre {
		st.PaquetsPropres++
	}
	for _, r := range recs {
		st.Records++
		if r.DesyncAt != -1 {
			st.Desync++
			continue
		}
		st.RecordsPropres++
		for _, comp := range r.Trace.Comps {
			if comp.Name != attI10Component || !comp.Ported {
				continue
			}
			s, ok := vues[comp.StartBit]
			if !ok {
				continue
			}
			reclamees++
			st.Lectures++
			st.ParTI[r.TypeIndex]++
			if s.Attached {
				st.AttachesParTI[r.TypeIndex]++
			}
			if propre {
				st.LecturesPropres++
			}
			l := attI10{
				TS: p.TimestampUS, Chunk: chunk, Packet: p.Index,
				Slot: r.Slot, Gen: r.ID >> 30, TI: r.TypeIndex, St: s,
				ParentSlot: s.Quant16 & attSlotMask, Lies: w.Bound(), Propre: propre,
			}
			if s.Attached {
				l.ParentTI, l.ParentLie = w.ArchetypeForSlot(l.ParentSlot)
				_, l.TemoinLie = w.ArchetypeForSlot((l.ParentSlot + attDecalageTemoin) & attSlotMask)
			}
			out = append(out, l)
		}
	}
	st.Orphelines += len(vues) - reclamees
	return out
}

// attSlotMask est le masque du champ de slot d'un identifiant de record : `IDLowBits` = 13
// bits (cf. `filmdec.DefaultFrameConfig`), les deux bits de poids fort étant la génération.
const attSlotMask = uint32(1<<13) - 1

// attPaquetPropre dit qu'un paquet est BIT-EXACT : au moins une vue a atteint son marqueur
// de fin et aucun record n'a désynchronisé.
//
// LE MARQUEUR DE FIN EST LA SEULE PREUVE D'ALIGNEMENT dont on dispose hors ligne. Un record
// dont tous les composants ont un déser porté « réussit » même s'il consomme la mauvaise
// largeur : rien dans le record lui-même ne le dit. Le marqueur de fin de vue, lui, ne tombe
// à sa place que si TOUTES les largeurs qui le précèdent étaient justes — un seul bit de
// trop ou de moins et le décodeur lit un type de record au lieu du marqueur.
func attPaquetPropre(recs []filmdec.FrameRecord, vuesDone int) bool {
	if vuesDone < 1 {
		return false
	}
	for _, r := range recs {
		if r.DesyncAt != -1 {
			return false
		}
	}
	return true
}

// attDecalageTemoin décale le slot du témoin. Un NOMBRE PREMIER, et grand devant la bande
// des slots vivants : le témoin doit tomber ailleurs que sur le voisinage immédiat du slot
// lu, sinon il mesurerait la densité locale des entités et non le hasard.
const attDecalageTemoin = uint32(4093)

// attTIsTries rend les archétypes rencontrés, dans l'ordre croissant.
func attTIsTries(m map[uint32]int) []uint32 {
	out := make([]uint32, 0, len(m))
	for ti := range m {
		out = append(out, ti)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// attPart rend un taux, 0 quand le dénominateur est nul.
func attPart(n, d int) float64 {
	if d == 0 {
		return 0
	}
	return float64(n) / float64(d)
}

// attScanMemo mémorise le balayage d'i10 par film : il déroule TOUT le film et plusieurs
// items de la phase 0 en ont besoin.
var attScanMemo = map[string][]attI10{}
var attStatMemo = map[string]attStat{}

// attScanOf rend les lectures d'i10 d'un film, balayées une seule fois par process.
func attScanOf(t *testing.T, root, id string) ([]attI10, attStat) {
	t.Helper()
	if v, ok := attScanMemo[id]; ok {
		return v, attStatMemo[id]
	}
	v, st, err := attScanI10(objChunkDir(root, id))
	if err != nil {
		t.Fatalf("%s : balayage i10 : %v", id, err)
	}
	attScanMemo[id], attStatMemo[id] = v, st
	return v, st
}

// TestAttachementPhase0Recensement — ITEM 0.1 : la sonde rend-elle quelque chose, et quoi.
//
// C'EST LE PRÉALABLE DES DEUX ORACLES, et il se publie séparément : si i10 n'était jamais
// lu sur le chemin delta, ou jamais à porte ouverte, les items 0.2 et 0.3 mesureraient le
// vide sans le dire. Les dénominateurs sont donc rendus AVANT toute confrontation.
func TestAttachementPhase0Recensement(t *testing.T) {
	root := attRequireRoot(t)
	joues := 0
	for _, id := range attTousFilms() {
		if _, ok := objOpenFilm(t, root, id); !ok {
			t.Logf("%s : film absent du cache — sauté", id)
			continue
		}
		joues++
		lectures, st := attScanOf(t, root, id)
		t.Logf("%s : %d paquets delta · %d records (%d propres, %d désynchronisés) · "+
			"%d lectures d'i10 rattachées, %d orphelines (essais de décodage)",
			id, st.Paquets, st.Records, st.RecordsPropres, st.Desync, st.Lectures, st.Orphelines)
		for _, ti := range attTIsTries(st.ParTI) {
			t.Logf("%s :   ti=%-3d — %6d lectures, %6d à porte OUVERTE (%.2f %%)",
				id, ti, st.ParTI[ti], st.AttachesParTI[ti],
				100*attPart(st.AttachesParTI[ti], st.ParTI[ti]))
		}
		attLogChamps(t, id, lectures)
	}
	if joues == 0 {
		t.Skipf("aucun film du corpus dans le cache (%s=%q)", attFilmEnv, root)
	}
}

// attTousFilms rend le corpus complet de la phase 0 : les trois films CTF (item 0.2) et le
// film à véhicules (item 0.3).
func attTousFilms() []string {
	return append(append([]string{}, objCTFFilms...), attFilmsVehicules()...)
}

// attLogChamps publie la distribution des champs de la branche ATTACHÉE, par archétype.
// Un champ dont une seule valeur sature toutes les lectures est une constante de structure,
// pas un lien ; un champ qui prend autant de valeurs qu'il y a de lectures est du bruit ou
// une donnée continue. C'est cette forme-là qui oriente les hypothèses de l'item 0.2.
func attLogChamps(t *testing.T, id string, lectures []attI10) {
	t.Helper()
	parTI := map[uint32][]attI10{}
	for _, l := range lectures {
		if l.St.Attached {
			parTI[l.TI] = append(parTI[l.TI], l)
		}
	}
	for _, ti := range attTIsTries(attCompteParTI(parTI)) {
		ls := parTI[ti]
		t.Logf("%s :   ti=%-3d ATTACHÉ (%d lectures) — quant16 %s · word16 %s · opt16 %s · "+
			"mtx0 %s · byte8 %s · slots %d",
			id, ti, len(ls),
			attCardinal(ls, func(l attI10) (uint32, bool) { return l.St.Quant16, true }),
			attCardinal(ls, func(l attI10) (uint32, bool) { return l.St.Word16, true }),
			attCardinal(ls, func(l attI10) (uint32, bool) { return l.St.Opt16, l.St.HasOpt16 }),
			attCardinal(ls, func(l attI10) (uint32, bool) { return l.St.Mtx[0], true }),
			attCardinal(ls, func(l attI10) (uint32, bool) { return l.St.Byte8, true }),
			len(attSlotsDe(ls)))
	}
}

// attCompteParTI projette une table de lectures par archétype en table de comptes.
func attCompteParTI(m map[uint32][]attI10) map[uint32]int {
	out := make(map[uint32]int, len(m))
	for ti, ls := range m {
		out[ti] = len(ls)
	}
	return out
}

// attSlotsDe rend les slots distincts d'un lot de lectures.
func attSlotsDe(ls []attI10) map[uint32]bool {
	out := map[uint32]bool{}
	for _, l := range ls {
		out[l.Slot] = true
	}
	return out
}

// attCardinal met en forme la cardinalité d'un champ et sa valeur dominante.
func attCardinal(ls []attI10, get func(attI10) (uint32, bool)) string {
	compte := map[uint32]int{}
	presents := 0
	for _, l := range ls {
		v, ok := get(l)
		if !ok {
			continue
		}
		presents++
		compte[v]++
	}
	if presents == 0 {
		return "absent"
	}
	var top uint32
	best := -1
	for v, n := range compte {
		if n > best || (n == best && v < top) {
			top, best = v, n
		}
	}
	return fmt.Sprintf("%d val./%d lect., dom. 0x%X x%d", len(compte), presents, top, best)
}
