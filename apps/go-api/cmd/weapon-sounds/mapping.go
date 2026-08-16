package main

// mapping.go — ETAPE 2 : d'un `.pck` d'arme vers ses evenements Wwise et leurs `.wem`.
//
// Le `.pck` est le point d'entree parce qu'il porte le nom de l'arme ET l'ensemble de ses
// `.wem`. Cet ensemble sert deux fois : il DESIGNE le `sbnk` de l'arme parmi 1305 (celui
// qui contient le plus de ses IDs), et il VALIDE les lectures ambigues du parseur de bank.

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"levelup/go-api/internal/himodule"
)

// magicBKHD marque le debut de la bank Wwise dans les octets du tag `sbnk`.
var magicBKHD = []byte("BKHD")

// indexBKHD rend l'offset du debut de la bank dans les octets d'un tag `sbnk`.
func indexBKHD(b []byte) int { return bytes.Index(b, magicBKHD) }

type evenementRendu struct {
	IDEvent uint32   `json:"id_event"`
	Nom     string   `json:"nom,omitempty"` // retrouve par hachage FNV-1, vide si inconnu
	Nombre  int      `json:"nombre_wem"`
	Wems    []uint32 `json:"wems"`
	// Couches : une entree par ACTION de l'evenement, donc par couche jouee EN PARALLELE.
	// Un coup entendu est la somme d'une variante tiree dans chaque couche — pas un `.wem`.
	Couches []brancheRendue `json:"couches,omitempty"`
	// Variation : la fourchette de la COUCHE DOMINANTE, c'est-a-dire celle qui porte le
	// coup. C'est cette valeur que l'app applique a chaque lecture.
	Variation *variationRendue `json:"variation,omitempty"`
}

type rapportArme struct {
	Arme        string           `json:"arme"`
	Pck         string           `json:"pck"`
	SbnkGlobal  uint32           `json:"sbnk_global_id"`
	WemsDuPck   int              `json:"wems_du_pck"`
	Evenements  []evenementRendu `json:"evenements"`
	SonsResolus int              `json:"sons_resolus"`
}

// cartographier resout une arme : `.pck` -> `sbnk` -> evenements -> `.wem`.
func cartographier(cheminModule, cheminPck, sortieJSON string) error {
	ids, err := idsPck(cheminPck)
	if err != nil {
		return err
	}
	arme := nomArme(cheminPck)
	fmt.Printf("arme     : %s\n", arme)
	fmt.Printf("pck      : %d .wem\n", len(ids))

	m, err := himodule.Open(cheminModule)
	if err != nil {
		return err
	}
	f, brut, score, err := trouverSbnk(m, ids)
	if err != nil {
		return err
	}
	fmt.Printf("sbnk     : gid %08x (%d .wem du pck retrouves dans ses octets)\n", f.GlobalID, score)

	calibrerNommage(brut, cheminPck)

	b, err := parserBank(brut, validateurWem(ids))
	if err != nil {
		return err
	}
	fmt.Printf("hierarchie: %d objets, %d Sound resolus, %d Events, %d Actions\n\n",
		len(b.Objets), len(b.Sons), len(b.Events), len(b.Actions))

	rap := rapportArme{
		Arme: arme, Pck: cheminPck, SbnkGlobal: f.GlobalID,
		WemsDuPck: len(ids), SonsResolus: len(b.Sons),
	}
	for id := range b.Events {
		w := b.wemsDeEvent(id)
		if len(w) == 0 {
			continue
		}
		rap.Evenements = append(rap.Evenements, evenementRendu{IDEvent: id, Nombre: len(w), Wems: w})
	}
	idsEvents := make([]uint32, 0, len(rap.Evenements))
	for _, e := range rap.Evenements {
		idsEvents = append(idsEvents, e.IDEvent)
	}
	noms := resoudreNoms(arme, idsEvents)
	for i := range rap.Evenements {
		rap.Evenements[i].Nom = noms[rap.Evenements[i].IDEvent]
	}
	sort.Slice(rap.Evenements, func(i, j int) bool {
		return rap.Evenements[i].Nombre > rap.Evenements[j].Nombre
	})
	afficherEvenements(rap, ids)
	if sortieJSON == "" {
		return nil
	}
	blob, err := json.MarshalIndent(rap, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(sortieJSON, blob, 0o644)
}

// trouverSbnk designe le `sbnk` de l'arme : celui dont les octets portent le plus d'IDs
// du `.pck`. Aucun nom n'intervient — c'est l'intersection des donnees qui tranche.
func trouverSbnk(m *himodule.Module, ids map[uint32]bool) (himodule.File, []byte, int, error) {
	temoins := echantillon(ids, 24)
	var meilleur himodule.File
	var brutMeilleur []byte
	meilleurScore := 0
	for _, f := range m.Files("sbnk") {
		data, err := m.Extract(f)
		if err != nil {
			continue
		}
		score := compterTemoins(data, temoins)
		if score > meilleurScore {
			meilleurScore, meilleur, brutMeilleur = score, f, data
		}
	}
	if meilleurScore == 0 {
		return meilleur, nil, 0, fmt.Errorf("aucun sbnk ne porte les .wem de ce pck")
	}
	debut := indexBKHD(brutMeilleur)
	if debut < 0 {
		return meilleur, nil, 0, fmt.Errorf("sbnk %08x sans chunk BKHD", meilleur.GlobalID)
	}
	return meilleur, brutMeilleur[debut:], meilleurScore, nil
}

// calibrerNommage confronte l'ID de la bank (chunk BKHD) au hachage de son nom de fichier.
//
// C'EST LE POINT D'ETALONNAGE de l'etape 3 : la bank est le SEUL objet dont on connaisse
// a la fois l'identifiant Wwise et un nom candidat solide (le nom du `.pck`). S'ils
// coincident, la convention de nommage est etablie et l'echec sur les evenements ne vient
// que de la liste de verbes. S'ils divergent, c'est la forme des noms qu'il faut revoir.
func calibrerNommage(brut []byte, cheminPck string) {
	ch := chunks(brut)
	tete, ok := ch["BKHD"]
	if !ok || len(tete) < 8 {
		fmt.Println("calibrage : chunk BKHD illisible")
		return
	}
	idBank := binary.LittleEndian.Uint32(tete[4:])
	base := nomFichierSansExt(cheminPck)
	fmt.Printf("calibrage: id de bank %08x", idBank)
	for _, cand := range []string{base, "wea_" + nomArme(cheminPck), nomArme(cheminPck)} {
		if fnv1(cand) == idBank {
			fmt.Printf(" == fnv1(%q) -- convention CONFIRMEE\n", cand)
			return
		}
	}
	fmt.Printf(" ; fnv1(%q) = %08x -- NE CORRESPOND PAS\n", base, fnv1(base))
}

// echantillon prend au plus n IDs, tries pour etre reproductibles d'un run a l'autre.
func echantillon(ids map[uint32]bool, n int) []uint32 {
	out := make([]uint32, 0, len(ids))
	for k := range ids {
		out = append(out, k)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	if len(out) > n {
		out = out[:n]
	}
	return out
}

// compterTemoins compte les IDs presents en clair dans les octets d'un tag.
func compterTemoins(data []byte, temoins []uint32) int {
	var aiguille [4]byte
	n := 0
	for _, t := range temoins {
		binary.LittleEndian.PutUint32(aiguille[:], t)
		if bytes.Contains(data, aiguille[:]) {
			n++
		}
	}
	return n
}

// afficherEvenements rend compte, et controle que tout `.wem` rendu vient bien du `.pck`.
func afficherEvenements(rap rapportArme, ids map[uint32]bool) {
	fmt.Println("--- evenements, du plus fourni au plus rare ---")
	total := map[uint32]bool{}
	horsPck := 0
	nommes := 0
	for i, e := range rap.Evenements {
		if e.Nom != "" {
			nommes++
		}
		if i < 15 {
			nom := e.Nom
			if nom == "" {
				nom = "(nom non retrouve)"
			}
			fmt.Printf("  event %08x : %3d .wem  %s\n", e.IDEvent, e.Nombre, nom)
		}
		for _, w := range e.Wems {
			total[w] = true
			if !ids[w] {
				horsPck++
			}
		}
	}
	if len(rap.Evenements) > 15 {
		fmt.Printf("  ... et %d autres evenements\n", len(rap.Evenements)-15)
	}
	fmt.Printf("\ncouverture : %d .wem distincts atteints sur %d dans le pck\n",
		len(total), rap.WemsDuPck)
	fmt.Printf("controle croise : %d .wem hors du pck (attendu 0)\n", horsPck)
	fmt.Printf("noms retrouves  : %d / %d evenements\n", nommes, len(rap.Evenements))
}
