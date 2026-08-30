package main

// banks_noms.go — NOMMER LES 1305 BANQUES DU JEU (mode `banks-noms`).
//
// CE QUE CE MODE DEBLOQUE. Toute la chaine des sons part d'une banque, et jusqu'ici une
// banque ne se nommait que par RICOCHET : si un de ses `.wem` vivait dans un `.pck` au nom
// explicite, elle heritait de ce nom. La mesure du lot des equipements manquants a chiffre la
// portee de ce pont : 3 banques d'equipement sur 17 seulement touchent un pack nomme. Les
// 14 autres — et toutes les banques de MODE (drapeau, bastion, extraction, bombe), dont
// AUCUNE n'a de pack — restaient anonymes.
//
// LA VOIE : L'IDENTIFIANT EST LE HACHAGE DU NOM. Le chunk `BKHD` d'une banque porte son
// identifiant Wwise, et cet identifiant est le FNV-1 32 bits du nom de fichier en minuscules.
// Ce n'est pas une hypothese : `calibrerNommage` (mapping.go) confronte deja les deux sur les
// banques qui ont un pack. Ce mode generalise ce controle a TOUT le module, puis attaque les
// banques restantes au dictionnaire (`banks_dico.go`).
//
// L'ORDRE DES CONTROLES EST IMPOSE, et il est ecrit ici pour ne pas pouvoir etre saute :
//
//	1. CALIBRATION : combien de banques du module portent l'identifiant d'un `.pck` du jeu ?
//	   C'est la preuve de la convention, sur un temoin de plusieurs centaines. Si elle est
//	   basse, tout le reste est nul et le mode le dit.
//	2. ESPERANCE DE COLLISION : `candidats x cibles / 2^32` doit rester sous 0,10 (le seuil
//	   que le lot des equipements s'etait donne pour le murmur3). Elle est imprimee AVANT les
//	   resultats, jamais apres.
//	3. RESULTATS : un nom par banque, avec sa PROVENANCE (`pck` = lu sur le disque,
//	   `dictionnaire` = casse par hachage), jamais melangees.
//
// MEMOIRE : ce mode ouvre `pc/globals` (7,24 Go). Jamais en parallele d'un autre chargement.

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"levelup/go-api/internal/himodule"
)

// BanqueNommee est une banque du module, son identifiant Wwise et son nom quand il est etabli.
type BanqueNommee struct {
	Gid        string `json:"sbnk_gid"`
	IDWwise    uint32 `json:"id_wwise"`
	Nom        string `json:"nom,omitempty"`
	Provenance string `json:"provenance,omitempty"` // "pck" | "dictionnaire"
	Wem        int    `json:"wem_embarques"`
	Octets     int    `json:"octets"`
}

// RapportBanques est le JSON de sortie du mode.
type RapportBanques struct {
	Module          string         `json:"module"`
	Banques         int            `json:"banques"`
	NommeesParPck   int            `json:"nommees_par_pck"`
	NommeesParDico  int            `json:"nommees_par_dictionnaire"`
	Anonymes        int            `json:"anonymes"`
	Candidats       int            `json:"candidats_dictionnaire"`
	Esperance       float64        `json:"esperance_collision"`
	PacksSansBanque []string       `json:"packs_sans_banque_dans_ce_module,omitempty"`
	Liste           []BanqueNommee `json:"banques_detail"`
}

// nommerBanques est le mode `banks-noms`.
func nommerBanques(cheminModule, dossierSFX, sortie string) error {
	packs, err := nomsDePacks(dossierSFX)
	if err != nil {
		return err
	}
	fmt.Printf("packs lus sur le disque : %d\n", len(packs))

	m, err := himodule.Open(cheminModule)
	if err != nil {
		return err
	}
	rapporterMemoire("module charge")

	banques, err := lireIdentifiantsBanques(m)
	if err != nil {
		return err
	}
	fmt.Printf("banques du module : %d (dont %d sans chunk BKHD lisible)\n",
		len(banques), compterSansID(banques))

	rap := apparier(cheminModule, banques, packs)
	afficherRapportBanques(rap)
	if sortie == "" {
		return nil
	}
	b, err := json.MarshalIndent(rap, "", " ")
	if err != nil {
		return err
	}
	return os.WriteFile(sortie, b, 0o644)
}

// nomsDePacks rend les noms de fichiers `.pck` sans extension, en minuscules.
func nomsDePacks(dossier string) ([]string, error) {
	chemins, err := filepath.Glob(filepath.Join(dossier, "*.pck"))
	if err != nil {
		return nil, err
	}
	if len(chemins) == 0 {
		return nil, fmt.Errorf("aucun .pck dans %s", dossier)
	}
	out := make([]string, 0, len(chemins))
	for _, c := range chemins {
		out = append(out, strings.ToLower(nomFichierSansExt(c)))
	}
	sort.Strings(out)
	return out, nil
}

// lireIdentifiantsBanques decompresse chaque tag `sbnk` et lit l'identifiant du chunk BKHD.
func lireIdentifiantsBanques(m *himodule.Module) ([]BanqueNommee, error) {
	fichiers := m.Files("sbnk")
	out := make([]BanqueNommee, 0, len(fichiers))
	for _, f := range fichiers {
		b := BanqueNommee{Gid: fmt.Sprintf("%08x", f.GlobalID)}
		data, err := m.Extract(f)
		if err != nil {
			out = append(out, b)
			continue
		}
		b.Octets = len(data)
		debut := indexBKHD(data)
		if debut < 0 {
			out = append(out, b)
			continue
		}
		ch := chunks(data[debut:])
		if tete, ok := ch["BKHD"]; ok && len(tete) >= 8 {
			b.IDWwise = binary.LittleEndian.Uint32(tete[4:])
		}
		if didx, ok := ch["DIDX"]; ok {
			b.Wem = len(didx) / 12
		}
		out = append(out, b)
	}
	return out, nil
}

// compterSansID compte les banques dont le chunk BKHD n'a pas ete lu.
func compterSansID(banques []BanqueNommee) int {
	n := 0
	for _, b := range banques {
		if b.IDWwise == 0 {
			n++
		}
	}
	return n
}

// apparier associe un nom a chaque banque : d'abord par les packs du disque, puis par le
// dictionnaire sur les seules banques restantes (c'est ce qui garde l'esperance basse).
func apparier(module string, banques []BanqueNommee, packs []string) RapportBanques {
	parPck := make(map[uint32]string, len(packs))
	for _, p := range packs {
		parPck[fnv1(p)] = p
	}
	vus := map[uint32]bool{}
	restantes := 0
	for i := range banques {
		id := banques[i].IDWwise
		if id == 0 {
			continue
		}
		if nom, ok := parPck[id]; ok {
			banques[i].Nom, banques[i].Provenance = nom, "pck"
			vus[id] = true
			continue
		}
		restantes++
	}

	dico := candidatsBanques(packs)
	rap := RapportBanques{
		Module:    module,
		Banques:   len(banques),
		Candidats: len(dico),
		Esperance: esperanceCollision(len(dico), restantes),
		Liste:     banques,
	}
	for i := range banques {
		if banques[i].Nom != "" || banques[i].IDWwise == 0 {
			continue
		}
		if nom, ok := dico[banques[i].IDWwise]; ok {
			banques[i].Nom, banques[i].Provenance = nom, "dictionnaire"
		}
	}
	for _, b := range banques {
		switch b.Provenance {
		case "pck":
			rap.NommeesParPck++
		case "dictionnaire":
			rap.NommeesParDico++
		default:
			rap.Anonymes++
		}
	}
	for _, p := range packs {
		if !vus[fnv1(p)] {
			rap.PacksSansBanque = append(rap.PacksSansBanque, p)
		}
	}
	return rap
}

// afficherRapportBanques imprime la calibration AVANT les resultats, puis les noms casses.
func afficherRapportBanques(rap RapportBanques) {
	fmt.Println()
	fmt.Println("=== 1. CALIBRATION — la convention de nommage tient-elle ? ===")
	fmt.Printf("  banques dont l'identifiant est le FNV-1 d'un nom de pack : %d / %d\n",
		rap.NommeesParPck, rap.Banques)
	fmt.Printf("  packs du disque sans banque dans ce module : %d\n", len(rap.PacksSansBanque))
	if rap.NommeesParPck < 50 {
		fmt.Println("  ATTENTION : temoin trop maigre — le dictionnaire ci-dessous ne prouve rien.")
	}

	fmt.Println()
	fmt.Println("=== 2. ESPERANCE DE COLLISION ===")
	cibles := rap.Banques - rap.NommeesParPck
	fmt.Println("  " + formaterEsperance(rap.Candidats, cibles))

	fmt.Println()
	fmt.Println("=== 3. NOMS CASSES PAR DICTIONNAIRE (banques sans pack) ===")
	casses := make([]BanqueNommee, 0, rap.NommeesParDico)
	for _, b := range rap.Liste {
		if b.Provenance == "dictionnaire" {
			casses = append(casses, b)
		}
	}
	sort.Slice(casses, func(i, j int) bool { return casses[i].Nom < casses[j].Nom })
	for _, b := range casses {
		fmt.Printf("  sbnk %s  id %08x  %-44s  %d wem embarques\n", b.Gid, b.IDWwise, b.Nom, b.Wem)
	}
	if len(casses) == 0 {
		fmt.Println("  (aucun)")
	}
	fmt.Printf("\nbilan : %d nommees par pck, %d par dictionnaire, %d anonymes\n",
		rap.NommeesParPck, rap.NommeesParDico, rap.Anonymes)
}
