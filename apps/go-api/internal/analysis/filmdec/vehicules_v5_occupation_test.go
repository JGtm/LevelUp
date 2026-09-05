package filmdec

// vehicules_v5_occupation_test.go — INSTRUMENT DU LOT V5 : « où est l'ÉTAT D'OCCUPATION
// COURANT d'un véhicule ? » (2026-09-02).
//
// LA QUESTION. Les événements board/exit sont décodés, mais ce sont des TRANSITIONS. Le mode
// Théâtre affiche conducteur / passager / artilleur À TOUT INSTANT et permet de sauter dans le
// temps : l'état d'occupation est donc SÉRIALISÉ, pas seulement événementiel.
//
// LA MÉTHODE : CORRÉLATION SUPERVISÉE. On dispose d'une vérité terrain — les sorties
// (`ScanFilmVehicleEvents`, occupant validé 100 % en bande, fermeture de trou 90,7 %) et les
// TROUS du flux de position du bipède (V1a.4 : l'occupant attaché cesse de répliquer). Un
// épisode attesté = [début du trou, instant de la sortie] pour un slot bipède donné. On
// balaie ensuite TOUS les champs de référence d'entité lus par les composants `unit-*` des
// records DELTA (sonde `SetUnitRefHook`) et on demande : lequel vaut le slot de l'occupant
// PENDANT l'épisode, et rien hors épisode ?
//
// LECTURE SEULE, hors ligne, sous garde d'environnement. Aucun fichier écrit, aucune base.
//
//	CGO_ENABLED=0 V5_ROOT=<cache> V5_FILMS=0d76e8f1,fccc61cd \
//	  go test ./internal/analysis/filmdec/ -run TestV5 -v -timeout 60m

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"
)

const (
	v5RootEnv  = "V5_ROOT"  // racine du cache (contient film_chunks/)
	v5FilmsEnv = "V5_FILMS" // short8 séparés par des virgules
)

// v5VehiculeTI est l'archétype véhicule ; v5BipedeTI celui du bipède joueur.
const (
	v5VehiculeTI = 40
	v5BipedeTI   = 35
)

// v5VuesParPaquet : nombre de vues décodées par paquet delta (même valeur que les
// instruments d'attachement — un paquet porte jusqu'à 3 vues plus une marge).
const v5VuesParPaquet = 4

// v5Films rend les répertoires de chunks demandés (short8 -> chemin complet).
func v5Films(t *testing.T) []string {
	t.Helper()
	root := os.Getenv(v5RootEnv)
	films := os.Getenv(v5FilmsEnv)
	if root == "" || films == "" {
		t.Skipf("mesure non demandée : %s ou %s vide", v5RootEnv, v5FilmsEnv)
	}
	var out []string
	for _, s := range strings.Split(films, ",") {
		if s = strings.TrimSpace(s); s != "" {
			out = append(out, root+"/film_chunks/"+s)
		}
	}
	return out
}

// v5Ref est UNE lecture de champ de référence, rattachée au record et au composant qui la
// contiennent.
type v5Ref struct {
	TS        uint64 // horodatage du paquet
	Chunk     int
	RecTI     uint32 // archétype du record porteur
	RecSlot   uint32 // slot du record porteur
	Comp      string // nom du composant contenant la lecture
	CompIdx   int    // index du composant dans l'archétype
	Kind      UnitRefKind
	Present   bool
	Val, Tail uint32
	// Ord est le rang de la lecture DANS son composant (0, 1, 2...) : deux champs du même
	// composant ne doivent pas être confondus.
	Ord int
}

// v5Slot rend le slot candidat porté par la lecture : l'index brut pour la forme à largeur
// variable, les 30 bits bas pour un handle 32 bits.
func (r v5Ref) Slot() uint32 {
	if r.Kind == UnitRefVarWidth {
		return r.Val
	}
	return r.Val & 0x3FFFFFFF
}

// v5Cle identifie un CANAL de lecture : le même champ, du même composant, du même archétype.
// C'est l'unité de décision — un signal doit être porté par UN canal, pas par « une lecture
// quelque part dans le record ».
type v5Cle struct {
	RecTI   uint32
	Comp    string
	CompIdx int
	Kind    UnitRefKind
	Ord     int
}

func (c v5Cle) String() string {
	return fmt.Sprintf("ti=%d i%d %s [%s#%d]", c.RecTI, c.CompIdx, c.Comp, c.Kind, c.Ord)
}

// v5Scan décode tout le film en marche stateful et rend TOUTES les lectures de champ de
// référence des records propres, plus les compteurs de qualité de la marche.
func v5Scan(dir string) ([]v5Ref, map[string]int, error) {
	brut, err := ReadFilmChunk(dir, 0)
	if err != nil {
		return nil, nil, fmt.Errorf("chunk_00 illisible : %w", err)
	}
	reg, err := ParseRegistryChunk(brut)
	if err != nil {
		return nil, nil, fmt.Errorf("registre illisible : %w", err)
	}
	release := LockProcessDecode()
	defer release()

	// La sonde écrit dans `vues`, indexée par position de bit : une position réécrite est un
	// record re-décodé, et c'est la DERNIÈRE lecture qui vaut.
	vues := map[int]UnitRefRead{}
	SetUnitRefHook(func(r UnitRefRead) { vues[r.StartBit] = r })
	defer SetUnitRefHook(nil)

	stat := map[string]int{}
	cfg := DefaultFrameConfig()
	w := NewWorld(reg)
	var out []v5Ref
	for c := 1; c <= CountFilmChunks(dir); c++ {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, p := range WalkPackets(data) {
			pay := p.Payload(data)
			if p.Type == PacketTypeKeyframe {
				w = WorldFromKeyframe(reg, pay)
				stat["keyframes"]++
				continue
			}
			if p.Type != PacketTypeDelta {
				continue
			}
			stat["paquets"]++
			for k := range vues {
				delete(vues, k)
			}
			recs, _ := DecodeFrameViews(pay, w, cfg, v5VuesParPaquet, cfg.PacketPreambleBits)
			out = append(out, v5Collecte(recs, vues, stat, p.TimestampUS, c)...)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].TS < out[j].TS })
	return out, stat, nil
}

// v5Collecte rattache les lectures publiées d'UN paquet aux records propres qui les
// réclament : une lecture appartient au composant dont l'emprise en bits la contient.
func v5Collecte(
	recs []FrameRecord, vues map[int]UnitRefRead, stat map[string]int, ts uint64, chunk int,
) []v5Ref {
	var out []v5Ref
	for _, r := range recs {
		stat["records"]++
		if r.DesyncAt != -1 {
			// UN RECORD DÉSYNCHRONISÉ N'EST PAS UN RECORD PERDU. `DesyncAt` est l'index du
			// PREMIER composant présent NON PORTÉ : tout ce qui le précède a été consommé
			// par la grammaire portée, au même titre que dans un record propre. Les jeter
			// priverait la mesure de l'essentiel du corpus véhicule — `ti=40` porte 13
			// composants `vehicle-*` non portés (i30..i42), TOUS situés APRÈS les
			// composants `unit-*` qui nous intéressent (i18..i26).
			stat["desync"]++
			stat[fmt.Sprintf("desync_ti%d", r.TypeIndex)]++
		} else {
			stat["records_propres"]++
			stat[fmt.Sprintf("records_propres_ti%d", r.TypeIndex)]++
		}
		comps := r.Trace.Comps
		for ci, comp := range comps {
			if !comp.Ported {
				continue
			}
			fin := r.Trace.EndBit
			if ci+1 < len(comps) {
				fin = comps[ci+1].StartBit
			}
			ord := 0
			for b := comp.StartBit; b < fin; b++ {
				v, ok := vues[b]
				if !ok {
					continue
				}
				out = append(out, v5Ref{
					TS: ts, Chunk: chunk, RecTI: r.TypeIndex, RecSlot: r.Slot,
					Comp: comp.Name, CompIdx: comp.Index, Kind: v.Kind,
					Present: v.Present, Val: v.Val, Tail: v.Tail, Ord: ord,
				})
				ord++
				stat["lectures"]++
			}
		}
	}
	return out
}

// TestV5Recensement — ÉTAPE 1, sans gate : QUELS canaux de référence existent, sur quels
// archétypes, et leurs valeurs tombent-elles dans la bande bipède ? Un canal qui ne rend
// jamais un slot bipède ne peut pas nommer un occupant.
func TestV5Recensement(t *testing.T) {
	for _, dir := range v5Films(t) {
		v5RecensementUnFilm(t, dir)
	}
}

func v5RecensementUnFilm(t *testing.T, dir string) {
	t.Helper()
	refs, stat, err := v5Scan(dir)
	if err != nil {
		t.Logf("V5 %s : %v", dir, err)
		return
	}
	chunks := v5TousChunks(dir)
	bande := bipedSlotBandDir(dir, chunks)
	bandeV := worldObjectSlotBandDir(dir, CountFilmChunks(dir), v5VehiculeTI)
	t.Logf("V5 RECENSEMENT %s — paquets=%d records=%d propres=%d desync=%d lectures=%d "+
		"| bande bipède=%d slots, bande véhicule=%d slots",
		dir, stat["paquets"], stat["records"], stat["records_propres"], stat["desync"],
		stat["lectures"], bande.Count(), len(bandeV))
	for _, ti := range []uint32{v5BipedeTI, v5VehiculeTI} {
		t.Logf("    records ti=%d : propres=%d tronqués(desync)=%d", ti,
			stat[fmt.Sprintf("records_propres_ti%d", ti)], stat[fmt.Sprintf("desync_ti%d", ti)])
	}

	type agg struct {
		n, present, enBandeB, enBandeV, zero int
		vals                                 map[uint32]int
	}
	par := map[v5Cle]*agg{}
	for _, r := range refs {
		c := v5Cle{RecTI: r.RecTI, Comp: r.Comp, CompIdx: r.CompIdx, Kind: r.Kind, Ord: r.Ord}
		a := par[c]
		if a == nil {
			a = &agg{vals: map[uint32]int{}}
			par[c] = a
		}
		a.n++
		if !r.Present {
			continue
		}
		a.present++
		s := r.Slot()
		a.vals[s]++
		switch {
		case s == 0:
			a.zero++
		case bande.Has(s):
			a.enBandeB++
		case bandeV[s]:
			a.enBandeV++
		}
	}
	cles := make([]v5Cle, 0, len(par))
	for c := range par {
		cles = append(cles, c)
	}
	sort.Slice(cles, func(i, j int) bool {
		if cles[i].RecTI != cles[j].RecTI {
			return cles[i].RecTI < cles[j].RecTI
		}
		if cles[i].CompIdx != cles[j].CompIdx {
			return cles[i].CompIdx < cles[j].CompIdx
		}
		return cles[i].Ord < cles[j].Ord
	})
	for _, c := range cles {
		a := par[c]
		if c.RecTI != v5BipedeTI && c.RecTI != v5VehiculeTI {
			continue
		}
		t.Logf("    %-52s n=%-7d ouverte=%-7d val!=0=%-6d enBandeBipède=%-6d enBandeVéhic=%-5d distinctes=%-5d top=%s",
			c.String(), a.n, a.present, a.present-a.zero, a.enBandeB, a.enBandeV, len(a.vals),
			v5Top(a.vals, 5))
	}
}

// v5Top rend les n valeurs les plus fréquentes, sous la forme `valeur x compte`.
func v5Top(vals map[uint32]int, n int) string {
	type kv struct {
		v uint32
		c int
	}
	l := make([]kv, 0, len(vals))
	for v, c := range vals {
		l = append(l, kv{v, c})
	}
	sort.Slice(l, func(i, j int) bool {
		if l[i].c != l[j].c {
			return l[i].c > l[j].c
		}
		return l[i].v < l[j].v
	})
	var b strings.Builder
	for i := 0; i < n && i < len(l); i++ {
		fmt.Fprintf(&b, "%d x%d ", l[i].v, l[i].c)
	}
	return b.String()
}

// v5TousChunks rend la liste des chunks 1..N.
func v5TousChunks(dir string) []int {
	n := CountFilmChunks(dir)
	out := make([]int, 0, n)
	for c := 1; c <= n; c++ {
		out = append(out, c)
	}
	return out
}
