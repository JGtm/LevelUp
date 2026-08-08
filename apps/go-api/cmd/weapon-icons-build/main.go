// Command weapon-icons-build extrait des archives `.module` d'Halo Infinite les ICÔNES
// D'ARMES du jeu et les écrit en PNG dans static/, prêtes à être revues puis branchées.
//
// POURQUOI CE BINAIRE EXISTE, ET POURQUOI IL N'EST PAS JETABLE. Il produit un artefact
// VERSIONNÉ (les PNG de static/weapons-assets/halo_infinite/jeu/) : il tombe donc du bon
// côté du critère de purge appliqué en J3 lot A — un outil qui n'établit qu'une grammaire
// est jetable, un outil qui régénère un asset versionné reste. Il est le seul moyen de
// reproduire ces images après une mise à jour de contenu du jeu.
//
// OÙ. Sur une machine qui a Halo Infinite INSTALLÉ. Il lit les archives en LECTURE SEULE et
// n'écrit que dans le dossier de sortie. cgo requis (décompression Kraken via internal/ooz).
//
//	go run ./cmd/weapon-icons-build                       # racine du jeu auto-détectée
//	go run ./cmd/weapon-icons-build -deploy "D:/.../deploy" -out ../../static/...
//
// CE QUI EST ÉTABLI SUR PIÈCES (aucune de ces étapes n'est postulée) :
//
//  1. Les 29 `weap` du registre référencent tous DEUX bitmaps communs — mesuré sur les 29 :
//     bc17adf1 et e39747c8. Le troisième bitm varie mais se PARTAGE entre armes par groupes
//     (Mangler+Needler, MA40+MA5K, six armes Covenant ensemble) : c'est un réticule, pas une
//     icône. Les deux bitmaps communs, eux, déclarent 39 images de tailles TOUTES
//     différentes — un jeu d'icônes, et le seul candidat possible.
//  2. Le lien tag -> pixels : l'entrée .module porte à +0x10 un index dans la table de
//     ressources du module (0 dans la variante `ds/`, qui n'a pas les pixels ; non nul dans
//     `pc/`). Les 39 ressources qui suivent cet index correspondent une à une aux 39 images.
//  3. Le format : contrôle ARITHMÉTIQUE exact. Pour l'image 0 (333x117), la ressource pèse
//     53372 octets = en-tête 212 + données 72 + 53088, et 53088 = 40320 + 10080 + 2688,
//     soit mip0+mip1+mip2 en blocs 4x4 sur 16 octets. À l'octet près, sur toutes les images.
//  4. Le contenu : R constant à 255, G constant à 0, B = rampe verticale de teinte appliquée
//     au rendu, A = LE DESSIN. Seul l'alpha est extrait, rendu en blanc sur fond transparent
//     — même convention que static/abilities-assets.
//
// CE QUI N'EST PAS ÉTABLI, ET QUI EST DONC LAISSÉ AU GATE HUMAIN : quelle image désigne
// quelle arme. L'index d'icône n'est pas un petit entier à offset constant dans le `weap`
// (mesuré : 0 candidat sur les 29 armes), et l'appariement automatique par silhouette contre
// les icônes déjà nommées du dépôt N'EST PAS DISCRIMINANT (marges de 0,00 à 0,10 pour des
// scores de 0,44 à 0,90 — des armes toutes « longues et horizontales » se ressemblent trop
// une fois remplies). Les fichiers sont donc nommés par leur INDEX dans l'atlas, jamais par
// un nom d'arme deviné : afficher un mauvais fusil est pire qu'un libellé.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// atlasTags : les deux bitmaps d'icônes communs à toutes les armes, et le style que chacun
// porte. Mesuré : même nombre d'images, mêmes dimensions index par index — ce sont les deux
// rendus du MÊME jeu d'icônes.
var atlasTags = []struct {
	ID    uint32
	Style string
	// CleanThrough : dernier index dont le decodage a ete CONFIRME A L OEIL. Au-dela, les
	// images sont quand meme livrees mais marquees suspectes dans index.json — rien n est
	// coupe, le doute est etiquete. C est un constat visuel, pas une mesure : la densite de
	// transitions d opacite a ete essayee et NE SEPARE PAS (l icone d explosion, legitime,
	// sort en tete du classement de bruit).
	CleanThrough int
}{
	{0xbc17adf1, "contour", 38},
	{0xe39747c8, "silhouette", 38},
	// Atlas « sandbox » : armes, vehicules, grenades lancees, et pictogrammes de mort. Le tag
	// en declare 88 ; les images au-dela de l index 72 sortent RAYEES (des descripteurs faux
	// positifs y tombent sur une ressource du bon poids, donc le controle arithmetique les
	// laisse passer). La coupe est ASSUMEE et bornee ici plutot que servie corrompue.
	{0x0302cad3, "sandbox", 72},
}

// iconEntry décrit une icône extraite, telle qu'écrite dans index.json.
type iconEntry struct {
	Index      int    `json:"index"`
	Style      string `json:"style"`
	File       string `json:"file"`
	SourceTag  string `json:"source_tag"`
	SourceW    int    `json:"source_w"`
	SourceH    int    `json:"source_h"`
	CroppedW   int    `json:"cropped_w"`
	CroppedH   int    `json:"cropped_h"`
	BC7Format  int    `json:"bc7_format"`
	Verified   bool   `json:"align_verified"`
	Suspect    bool   `json:"decode_suspect"`
	Noise      string `json:"alpha_noise"`
	RebuiltPc  string `json:"bc7_rebuilt_pct"`
	OpaquePc   string `json:"bc7_opaque_pct"`
	DegradedPc string `json:"bc7_degraded_pct"`
}

func main() {
	deploy := flag.String("deploy", "", "racine `deploy` des archives du jeu (auto-détectée si vide)")
	out := flag.String("out", filepath.Join("..", "..", "static", "weapons-assets", "halo_infinite", "jeu"),
		"dossier de sortie des PNG")
	probe := flag.Int("probe", 6, "profondeur de la sonde de recalage descripteur -> ressource")
	maxIdx := flag.Int("max", 120, "nombre d'images à extraire par atlas")
	flag.Parse()
	probeWindow = *probe

	if *deploy != "" {
		if err := os.Setenv("HALO_DEPLOY", *deploy); err != nil {
			fmt.Fprintln(os.Stderr, "HALO_DEPLOY:", err)
			os.Exit(1)
		}
	}
	root := moduleRoot()
	if root == "" {
		fmt.Fprintln(os.Stderr, "archives du jeu introuvables : passer -deploy <chemin>")
		os.Exit(1)
	}
	if err := os.MkdirAll(*out, 0o750); err != nil {
		fmt.Fprintln(os.Stderr, "création du dossier de sortie:", err)
		os.Exit(1)
	}
	fmt.Printf("archives : %s\nsortie   : %s\n\n", root, *out)

	ix := buildIndex("")
	fmt.Printf("index : %d archives, %d entrées\n\n", ix.nMods, ix.nEntry)

	var entries []iconEntry
	for _, at := range atlasTags {
		n := 0
		limit := *maxIdx
		for idx := 0; idx < limit; idx++ {
			img, st, im, err := decodeAlphaGlyph(ix, at.ID, idx)
			if err != nil {
				fmt.Printf("%-11s arrêt à #%d : %v\n", at.Style, idx, err)
				break
			}
			name := fmt.Sprintf("%s-%02d.png", at.Style, idx)
			if err := writePNG(filepath.Join(*out, name), img); err != nil {
				fmt.Fprintln(os.Stderr, "écriture:", err)
				os.Exit(1)
			}
			entries = append(entries, iconEntry{
				Index: idx, Style: at.Style, File: name,
				SourceTag: fmt.Sprintf("%08x", at.ID),
				SourceW:   im.W, SourceH: im.H,
				CroppedW: img.Bounds().Dx(), CroppedH: img.Bounds().Dy(),
				BC7Format: im.Format, Verified: imageVerified(ix, at.ID, idx),
				Suspect:    idx > at.CleanThrough,
				Noise:      fmt.Sprintf("%.4f", alphaNoise(img)),
				RebuiltPc:  pct(st.Rebuilt, st.Blocks),
				OpaquePc:   pct(st.Opaque, st.Blocks),
				DegradedPc: pct(st.Degraded, st.Blocks),
			})
			n++
		}
		fmt.Printf("%-11s %d icônes écrites\n", at.Style, n)
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Index != entries[j].Index {
			return entries[i].Index < entries[j].Index
		}
		return entries[i].Style < entries[j].Style
	})
	blob, err := json.MarshalIndent(entries, "", " ")
	if err != nil {
		fmt.Fprintln(os.Stderr, "json:", err)
		os.Exit(1)
	}
	if err := os.WriteFile(filepath.Join(*out, "index.json"), append(blob, '\n'), 0o600); err != nil {
		fmt.Fprintln(os.Stderr, "index.json:", err)
		os.Exit(1)
	}
	fmt.Printf("\n%d entrées -> %s\n", len(entries), filepath.Join(*out, "index.json"))
}

// pct rend un pourcentage lisible, ou "0.00" si le dénominateur est nul.
func pct(n, total int) string {
	if total <= 0 {
		return "0.00"
	}
	return fmt.Sprintf("%.2f", 100*float64(n)/float64(total))
}
