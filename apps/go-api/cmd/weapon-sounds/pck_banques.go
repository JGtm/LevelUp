package main

// pck_banques.go — mode `pck-banques` : rattache PLUSIEURS `.pck` a leur banque `sbnk`,
// en UNE SEULE lecture du module (7,24 Go).
//
// POURQUOI CE MODE. `banks-noms` nomme une banque quand son identifiant Wwise (chunk BKHD)
// est le FNV-1 du nom de son `.pck` : 580 banques sur 1305. La convention TOMBE pour toute
// une famille — mesure du 2026-09-02 : les packs covenant/banished (`veh_cv_ghost`,
// `veh_cv_wraith`, `veh_bt_chopper`, `exp_vehicle_*_covenant`) n'ont AUCUNE banque a ce
// hachage, alors que leur banque existe bel et bien dans le module (le Ghost, par exemple,
// est `sbnk 01862ab3`). Le pont qui tient pour eux est l'appartenance des `.wem`, deja
// utilisee par le mode `map` — mais `map` ne traite qu'un pack a la fois, donc une charge de
// 7,24 Go par pack. Ce mode extrait chaque `sbnk` UNE fois et score tous les packs contre lui.
//
// DEUX LIENS DISTINCTS, MESURES SEPAREMENT (c'est le piege du lot) :
//
//	REF   la banque REFERENCE (chunk HIRC, `sourceID`) des `.wem` qui vivent dans le pack.
//	      C'est le lien qui vaut pour les vehicules : mesure du 2026-09-02, l'intersection
//	      entre le pck du Ghost (248 medias) et les medias EMBARQUES de sa banque 01862ab3
//	      (65) est VIDE — les deux jeux d'identifiants sont disjoints. Seul REF les relie.
//	EMB   la banque EMBARQUE (chunk DIDX) des medias du pack. Vrai pour les familles UNSC
//	      (`exp_vehicle_*_unsc`, `veh_un_*` : 31/31, 78/79...), faux pour tout le reste.
//
// Le rattachement retient REF ; EMB est imprime a cote, parce qu'un pack a EMB eleve donne
// directement les wems de prefetch, et qu'un pack a EMB nul impose de passer par `pck-dump`.
//
// COUT DE REF. Un `sourceID` n'est pas aligne dans la banque : il faut lire un u32 a CHAQUE
// offset. Un filtre de 65 536 bits sur les 16 bits bas des identifiants cibles rejette la
// quasi-totalite des offsets avant tout acces a la table — sans lui le mode ne finit pas.
//
// Usage : -mode pck-banques -pck <dossier ou motif glob> [-json <sortie>]

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"levelup/go-api/internal/himodule"
)

// PackBanque est le rattachement d'un `.pck` a sa banque.
type PackBanque struct {
	Pack     string `json:"pack"`
	Sbnk     string `json:"sbnk_gid,omitempty"`
	Ref      int    `json:"wem_references"` // medias du pack REFERENCES par la banque (HIRC)
	Embarque int    `json:"wem_embarques"`  // medias du pack EMBARQUES par la banque (DIDX)
	Wem      int    `json:"wem_pck"`        // medias du pack
}

// ciblePck est un pack a rattacher : son chemin et l'ensemble de ses identifiants `.wem`.
type ciblePck struct {
	chemin string
	ids    map[uint32]bool
}

// scoreBanque est le meilleur rattachement trouve pour un pack.
type scoreBanque struct {
	ref, emb int
	gid      uint32
}

// balayagePacks porte l'etat du balayage : les packs, l'index inverse des `.wem`, le filtre
// 16 bits, le tampon de travail et les meilleurs scores. Il existe pour tenir la signature
// du scoring a un seul parametre utile (le corps de la banque).
type balayagePacks struct {
	cibles       []ciblePck
	appartenance map[uint32][]int // id .wem -> index des packs qui le portent
	filtre       *[1 << 16]bool
	vus          []map[uint32]bool // tampon reutilise : evite une allocation par banque
	scores       []scoreBanque
}

// banquesDesPacks est le mode `pck-banques`.
func banquesDesPacks(cheminModule, motif, sortie string) error {
	chemins, err := cheminsDePacks(motif)
	if err != nil {
		return err
	}
	fmt.Printf("packs a rattacher : %d\n", len(chemins))
	b := lireCibles(chemins)

	m, err := himodule.Open(cheminModule)
	if err != nil {
		return err
	}
	rapporterMemoire("module charge")

	banques := m.Files("sbnk")
	for _, f := range banques {
		data, err := m.Extract(f)
		if err != nil {
			continue
		}
		debut := indexBKHD(data)
		if debut < 0 {
			continue
		}
		b.scorerUneBanque(data[debut:], f.GlobalID)
	}
	fmt.Printf("banques sbnk parcourues : %d\n\n", len(banques))
	return b.ecrireRattachement(sortie)
}

// lireCibles lit les identifiants de chaque pack et prepare l'index inverse + le filtre.
func lireCibles(chemins []string) *balayagePacks {
	b := &balayagePacks{appartenance: map[uint32][]int{}, filtre: new([1 << 16]bool)}
	for _, c := range chemins {
		ids, err := idsPck(c)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  %s illisible : %v\n", filepath.Base(c), err)
			continue
		}
		i := len(b.cibles)
		b.cibles = append(b.cibles, ciblePck{chemin: c, ids: ids})
		for id := range ids {
			b.appartenance[id] = append(b.appartenance[id], i)
			b.filtre[id&0xFFFF] = true
		}
	}
	b.vus = make([]map[uint32]bool, len(b.cibles))
	b.scores = make([]scoreBanque, len(b.cibles))
	return b
}

// scorerUneBanque confronte le corps d'une banque a tous les packs et retient les meilleurs.
func (b *balayagePacks) scorerUneBanque(corps []byte, gid uint32) {
	for i := range b.vus {
		b.vus[i] = nil
	}
	// REF : balayage non aligne, filtre 16 bits en amont.
	for o := 0; o+4 <= len(corps); o++ {
		v := binary.LittleEndian.Uint32(corps[o:])
		if !b.filtre[v&0xFFFF] {
			continue
		}
		for _, i := range b.appartenance[v] {
			if b.vus[i] == nil {
				b.vus[i] = map[uint32]bool{}
			}
			b.vus[i][v] = true
		}
	}
	didx := mediasEmbarques(chunks(corps))
	for i := range b.cibles {
		ref := len(b.vus[i])
		if ref <= b.scores[i].ref {
			continue
		}
		emb := 0
		for id := range didx {
			if b.cibles[i].ids[id] {
				emb++
			}
		}
		b.scores[i] = scoreBanque{ref: ref, emb: emb, gid: gid}
	}
}

// ecrireRattachement imprime le tableau et, si demande, ecrit le JSON.
func (b *balayagePacks) ecrireRattachement(sortie string) error {
	out := make([]PackBanque, 0, len(b.cibles))
	for i, c := range b.cibles {
		pb := PackBanque{
			Pack:     nomFichierSansExt(c.chemin),
			Ref:      b.scores[i].ref,
			Embarque: b.scores[i].emb,
			Wem:      len(c.ids),
		}
		if b.scores[i].ref > 0 {
			pb.Sbnk = fmt.Sprintf("%08x", b.scores[i].gid)
		}
		out = append(out, pb)
		fmt.Printf("  %-52s -> sbnk %-8s  REF %3d / %3d wem du pack, EMB %3d\n",
			pb.Pack, orNone(pb.Sbnk), pb.Ref, pb.Wem, pb.Embarque)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Pack < out[j].Pack })
	if sortie == "" {
		return nil
	}
	j, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(sortie, j, 0o644)
}

// cheminsDePacks accepte un dossier ou un motif glob et rend les `.pck` correspondants.
func cheminsDePacks(motif string) ([]string, error) {
	if motif == "" {
		return nil, fmt.Errorf("le mode pck-banques exige -pck (dossier ou motif glob)")
	}
	if st, err := os.Stat(motif); err == nil && st.IsDir() {
		motif = filepath.Join(motif, "*.pck")
	}
	chemins, err := filepath.Glob(motif)
	if err != nil {
		return nil, err
	}
	if len(chemins) == 0 {
		return nil, fmt.Errorf("aucun .pck ne correspond a %q", motif)
	}
	sort.Strings(chemins)
	return chemins, nil
}

// orNone rend un tiret quand la chaine est vide (lisibilite du tableau).
func orNone(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
