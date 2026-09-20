//go:build research

package replaybuild

// ventilation_vies_research_test.go — LOT 5.1.7 (2) : POURQUOI 159 VIES DE VEHICULE NE SONT PAS
// PUBLIEES, UNE CAUSE PAR VIE.
//
// # LA QUESTION, ET POURQUOI ELLE EST L OBJECTIF DU LOT
//
// Sur `4f77afc1`, le calque recense 256 vies de vehicule et en publie 97. Sans naissance ni fin
// de vie, aucun cycle de reapparition ne s etablit — c est le prealable de 5.1.5. Le premier
// niveau de ventilation est DEJA publie par la couverture : `sansPosition = 103` et
// `relaisFusionnes = 56`, et 103 + 56 = 159. Ce que la couverture ne dit pas, c est POURQUOI ces
// 103 n ont ni naissance ni position.
//
// # CE QU IL MESURE, ET SUR QUELLES PIECES
//
// Il CUIT LE FILM PAR LE CHEMIN DE PRODUCTION (`Builder.BuildBytes`) et capte le `VehicleScan`
// REEL par l observateur (`WithObserver`, etape `vehicles`) : aucune largeur redevinee, aucune
// reconstitution. Il ventile ensuite sur les SEULES DONNEES EXPORTEES du scan — le recensement
// d images-cles, les creations, le nuage de positions — et sur les vies que le document publie.
//
// APPROXIMATION ASSUMEE ET ECRITE : la fenetre d appartenance d une position a une vie est ici
// `[premier recensement, dernier recensement]`, la fenetre de CENSUS. La production y ajoute une
// tolerance (`vehicleLife.loUS/hiUS`) et decoupe les vies successives d un meme slot. Cette
// ventilation ne pretend donc pas reproduire `sansPosition` au record pres : elle dit, pour
// chaque vie, si son slot est REPLIQUE quelque part dans le film et si sa naissance est LUE.
// C est exactement ce que la question demande.
//
//	VENT_FILM=4f77afc1 VENT_CARTE="Flood Gulch" VENT_NOFACTS=1 VENT_OUT=<tsv> \
//	  go test -tags=research ./internal/replaybuild/ -run '^TestVentilationDesVies$' -v -timeout 60m

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/halo_infinite/film/filmcache"
	"levelup/go-api/internal/games/halo_infinite/film/replay"
	"levelup/go-api/internal/games/halo_infinite/film/types"
	"levelup/go-api/internal/testutil"
)

// Les causes, une par vie.
const (
	ventPubliee     = "publiee"
	ventAssemblable = "assemblable_mais_absente_du_document__relais_fusionne"
	ventFenetre     = "slot_replique_mais_aucune_position_dans_la_fenetre_de_la_vie"
	ventSansRien    = "aucune_naissance_lue_et_slot_jamais_replique"
)

// ventLigne est UNE vie recensee, avec sa cause et les denominateurs qui la justifient.
type ventLigne struct {
	slot, gen           uint32
	census              int
	firstUS, lastUS     uint64
	naissance           bool
	posSlot, posFenetre int
	cause               string
}

func TestVentilationDesVies(t *testing.T) {
	court, carte, sortie := os.Getenv("VENT_FILM"), os.Getenv("VENT_CARTE"), os.Getenv("VENT_OUT")
	if court == "" || sortie == "" {
		t.Skip("instrument de mesure : VENT_FILM et VENT_OUT requis")
	}
	scan, doc := ventCuire(t, court, carte)
	t.Logf("DENOMINATEURS : doc.Vehicles=%d recensees=%d creations=%d positions=%d",
		len(doc.Vehicles), len(scan.Keyframes.SeenUS), len(scan.Creations), len(scan.Positions))
	ventPublier(t, court, ventiler(scan, doc), sortie)
}

// ventCuire cuit le film par le chemin de production et capte l etape `vehicles`.
func ventCuire(t *testing.T, court, carte string) (replay.VehicleScan, replay.ReplayDocument) {
	t.Helper()
	repoRoot, err := testutil.RepoRoot()
	if err != nil {
		t.Fatalf("racine repo : %v", err)
	}
	matchID, cartes, faits := veh51Entrees(t, repoRoot, court, carte, os.Getenv("VENT_NOFACTS") != "")
	b, err := NewBuilder(repoRoot, title.DefaultSlug)
	if err != nil {
		t.Fatalf("preparation du builder : %v", err)
	}
	b.SansFaitsPersistes()
	var scan replay.VehicleScan
	vu := false
	b.WithObserver(func(step string, v any) {
		if step != "vehicles" {
			return
		}
		if s, ok := v.(replay.VehicleScan); ok {
			scan, vu = s, true
		}
	})
	cacheRoot := title.NewPathResolver(repoRoot).CacheRootDir()
	built, err := b.BuildBytes(matchID, cartes, filmcache.ChunkDir(cacheRoot, court), faits)
	if err != nil {
		t.Fatalf("cuisson de %s : %v", court, err)
	}
	if !vu {
		t.Fatalf("FILM %s : l etape `vehicles` n a rien rendu — le calque n a pas tourne", court)
	}
	var doc replay.ReplayDocument
	if err := json.Unmarshal(built.Blob, &doc); err != nil {
		t.Fatalf("relecture du document de %s : %v", court, err)
	}
	return scan, doc
}

// ventiler rend une ligne par vie recensee.
func ventiler(scan replay.VehicleScan, doc replay.ReplayDocument) []ventLigne {
	publiees := map[[2]uint32]bool{}
	for _, tr := range doc.Vehicles {
		publiees[[2]uint32{tr.Slot, tr.Gen}] = true
	}
	naissances := map[[2]uint32]bool{}
	for _, c := range scan.Creations {
		naissances[[2]uint32{c.Slot, c.Gen}] = true
	}
	posParSlot := map[uint32][]uint64{}
	for _, p := range scan.Positions {
		posParSlot[p.Slot] = append(posParSlot[p.Slot], p.TimestampUS)
	}
	out := make([]ventLigne, 0, len(scan.Keyframes.SeenUS))
	for key, seen := range scan.Keyframes.SeenUS {
		if len(seen) == 0 {
			continue
		}
		out = append(out, ventUneVie(key, seen, publiees, naissances, posParSlot))
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].slot != out[j].slot {
			return out[i].slot < out[j].slot
		}
		return out[i].gen < out[j].gen
	})
	return out
}

// ventUneVie statue UNE vie : sa cause, et une seule.
func ventUneVie(key types.EquipmentLifeKey, seen []uint64, publiees, naissances map[[2]uint32]bool,
	posParSlot map[uint32][]uint64) ventLigne {
	k := [2]uint32{key.Slot, key.Gen}
	g := ventLigne{
		slot: key.Slot, gen: key.Gen, census: len(seen),
		firstUS: seen[0], lastUS: seen[len(seen)-1], naissance: naissances[k],
		posSlot: len(posParSlot[key.Slot]),
	}
	for _, ts := range posParSlot[key.Slot] {
		if ts >= g.firstUS && ts <= g.lastUS {
			g.posFenetre++
		}
	}
	switch {
	case publiees[k]:
		g.cause = ventPubliee
	case g.naissance || g.posFenetre > 0:
		g.cause = ventAssemblable
	case g.posSlot > 0:
		g.cause = ventFenetre
	default:
		g.cause = ventSansRien
	}
	return g
}

// ventPublier ecrit le tableau et son bilan par cause.
func ventPublier(t *testing.T, court string, lignes []ventLigne, chemin string) {
	t.Helper()
	parCause := map[string]int{}
	var b strings.Builder
	b.WriteString("slot\tgen\trecensements\tpremiereUS\tderniereUS\tnaissanceLue\t" +
		"positionsDuSlot\tpositionsDansLaFenetre\tcause\n")
	for _, g := range lignes {
		parCause[g.cause]++
		fmt.Fprintf(&b, "%d\t%d\t%d\t%d\t%d\t%t\t%d\t%d\t%s\n", g.slot, g.gen, g.census,
			g.firstUS, g.lastUS, g.naissance, g.posSlot, g.posFenetre, g.cause)
	}
	if err := os.WriteFile(chemin, []byte(b.String()), 0o600); err != nil {
		t.Fatalf("ecriture de %s : %v", chemin, err)
	}
	causes := make([]string, 0, len(parCause))
	for c := range parCause {
		causes = append(causes, c)
	}
	sort.Strings(causes)
	t.Logf("FILM %s — %d vies recensees, une cause par vie :", court, len(lignes))
	for _, c := range causes {
		t.Logf("    %-62s %d", c, parCause[c])
	}
}
