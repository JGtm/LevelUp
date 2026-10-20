package main

// final.go — mode `final` : la liste des `.wem` de TIR d'une arme, et rien d'autre.
//
// Assemble les deux moities de la preuve, toutes deux etablies sans aucun nom Wwise :
//
//	1. `weap.xml` nomme le champ « Weapon Fire Sound » -> N tags de son (mode `tir`) ;
//	2. ces tags portent des identifiants d'evenements Wwise, et le rapport du mode `map`
//	   donne les `.wem` de chaque evenement.
//
// L'intersection rend les `.wem` de tir. Tout ce qui n'est pas atteint par cette chaine
// est explicitement laisse de cote — on ne complete jamais au jugé.

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"levelup/go-api/internal/himodule"
)

type rapportTir struct {
	Arme         string   `json:"arme"`
	WeapTag      string   `json:"weap_tag"`
	SonsDeTir    []string `json:"tags_son_de_tir"`
	EventsDeTir  []string `json:"events_de_tir"`
	WemsDeTir    []uint32 `json:"wems_de_tir"`
	WemsDuPck    int      `json:"wems_du_pck"`
	PartDuPckPct int      `json:"part_du_pck_pct"`
}

// livrerTir produit la liste des `.wem` de tir a partir du rapport du mode `map`.
func livrerTir(cheminModule, cheminJSON string, gidWeap uint32, sortie string) error {
	rap, err := chargerRapport(cheminJSON)
	if err != nil {
		return err
	}
	o, err := calculerOffsets()
	if err != nil {
		return err
	}
	m, err := himodule.Open(cheminModule)
	if err != nil {
		return err
	}
	rapporterMemoire("module charge")

	groupes := make(map[uint32]string, 1<<17)
	for _, f := range m.Files("") {
		groupes[f.GlobalID] = f.Group
	}
	sonsTir, err := tagsSonDeTir(m, gidWeap, o, groupes)
	if err != nil {
		return err
	}
	if len(sonsTir) == 0 {
		return fmt.Errorf("aucun tag « Weapon Fire Sound » pour le weap %08x", gidWeap)
	}
	fmt.Printf("arme      : %s (weap %08x)\n", rap.Arme, gidWeap)
	fmt.Printf("sons de tir declares par weap.xml : %d tag(s)\n", len(sonsTir))

	events := evenementsDesTags(m, sonsTir, rap)
	fmt.Printf("evenements portes par ces tags    : %d\n", len(events))

	wems := unionWems(rap, events)
	part := 0
	if rap.WemsDuPck > 0 {
		part = len(wems) * 100 / rap.WemsDuPck
	}
	fmt.Printf("\n=== %d .wem DE TIR sur %d dans le pck (%d %%) ===\n", len(wems), rap.WemsDuPck, part)

	out := rapportTir{
		Arme: rap.Arme, WeapTag: fmt.Sprintf("%08x", gidWeap),
		WemsDeTir: wems, WemsDuPck: rap.WemsDuPck, PartDuPckPct: part,
	}
	for _, g := range sonsTir {
		out.SonsDeTir = append(out.SonsDeTir, fmt.Sprintf("%08x", g))
	}
	for _, e := range events {
		out.EventsDeTir = append(out.EventsDeTir, fmt.Sprintf("%08x", e))
	}
	for _, e := range events {
		for _, ev := range rap.Evenements {
			if ev.IDEvent == e {
				fmt.Printf("  event %08x : %3d .wem\n", e, ev.Nombre)
			}
		}
	}
	if sortie == "" {
		return nil
	}
	blob, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}
	fmt.Printf("\necrit : %s\n", sortie)
	return os.WriteFile(sortie, blob, 0o644)
}

// tagsSonDeTir rend les tags designes par le champ nomme « Weapon Fire Sound ».
func tagsSonDeTir(m *himodule.Module, gidWeap uint32, o offsetsTir, groupes map[uint32]string) ([]uint32, error) {
	for _, f := range m.Files("weap") {
		if f.GlobalID != gidWeap {
			continue
		}
		data, err := m.Extract(f)
		if err != nil {
			return nil, err
		}
		t, err := ouvrirTagWeap(data)
		if err != nil {
			return nil, err
		}
		racine, err := t.blocRacine()
		if err != nil {
			return nil, err
		}
		var tous []uint32
		for _, bloc := range t.enfantsDe(racine) {
			sous := t.enfantsDe(bloc)
			blocVar, ok := sous[o.variations]
			if !ok {
				continue
			}
			absV, tailleV := t.blocAbs(blocVar)
			if absV < 0 || tailleV < tailleRef {
				continue
			}
			if gids, tousSons := refsDeSons(data, absV, tailleV, groupes); tousSons {
				tous = append(tous, gids...)
			}
		}
		sort.Slice(tous, func(i, j int) bool { return tous[i] < tous[j] })
		return tous, nil
	}
	return nil, fmt.Errorf("tag weap %08x absent du module", gidWeap)
}

// evenementsDesTags rend les identifiants d'evenements portes par ces tags de son.
func evenementsDesTags(m *himodule.Module, tags []uint32, rap rapportArme) []uint32 {
	voulu := map[uint32]bool{}
	for _, g := range tags {
		voulu[g] = true
	}
	set := map[uint32]bool{}
	var aiguille [4]byte
	for _, f := range append(m.Files("lsnd"), m.Files("snd!")...) {
		if !voulu[f.GlobalID] {
			continue
		}
		data, err := m.Extract(f)
		if err != nil {
			continue
		}
		for _, ev := range rap.Evenements {
			binary.LittleEndian.PutUint32(aiguille[:], ev.IDEvent)
			if bytes.Contains(data, aiguille[:]) {
				set[ev.IDEvent] = true
			}
		}
	}
	out := make([]uint32, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// unionWems rend l'union des `.wem` des evenements retenus.
func unionWems(rap rapportArme, events []uint32) []uint32 {
	garde := map[uint32]bool{}
	for _, e := range events {
		garde[e] = true
	}
	set := map[uint32]bool{}
	for _, ev := range rap.Evenements {
		if !garde[ev.IDEvent] {
			continue
		}
		for _, w := range ev.Wems {
			set[w] = true
		}
	}
	out := make([]uint32, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}
