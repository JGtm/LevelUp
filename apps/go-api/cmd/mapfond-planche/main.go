// cmd/mapfond-planche — LA PLANCHE DE REVUE DES FONDS DE CARTE.
//
// CE QU'ELLE RESOUT. Le gate visuel des fonds se jugeait sur des PNG copies dans un dossier
// du Bureau : l'utilisateur ouvrait 33 fichiers un par un, et son verdict ne laissait aucune
// trace ecrite. Cette planche rend UNE page autonome — vignettes cote a cote, chiffres sous
// chaque carte — que l'on regarde en une passe et dont le verdict entre au registre.
//
// CE QU'ELLE MONTRE ET QUI NE SE VOIT PAS AUTREMENT : le CADRE de chaque image est trace,
// et la matiere non transparente est encadree en pointilles. Un fond dont l'arene n'occupe
// que la moitie du cadre se lit d'un coup d'oeil — c'est precisement le defaut que l'oracle
// des ancres ne voit pas.
//
// ENTREE : un manifeste TSV, une ligne par IMAGE, groupees par cle de carte :
//
//	cle <TAB> libelle <TAB> sous-titre <TAB> statut <TAB> colonne <TAB> chemin.png
//
// Plusieurs lignes de meme `cle` deviennent les colonnes d'une meme fiche (avant / apres /
// temoin). L'ordre des cles suit leur premiere apparition dans le fichier.
//
// SORTIE : un seul fichier HTML, images comprises (data URI). Aucune ressource externe —
// c'est la contrainte des artefacts publies, et elle vaut aussi hors ligne.
//
// Usage :
//
//	go run ./cmd/mapfond-planche --manifeste M.tsv --sortie planche.html [--titre "..."]
//	                             [--intro "..."] [--cote 380]
package main

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"flag"
	"fmt"
	"html"
	"image"
	"image/draw"
	"image/png"
	"log/slog"
	"os"
	"strings"
)

// coteMaxParDefaut : cote maximal d'une vignette, en pixels. 380 px laisse lire l'architecture
// d'une arene tout en gardant la page sous le plafond de 16 Mo pour une soixantaine de cartes.
const coteMaxParDefaut = 380

// coteLoupeParDefaut : cote de l image servie par la LOUPE, en pixels. La vignette de la
// grille ne sert qu a reperer la carte ; le verdict se prend sur cette image-la. 900 px tient
// sur un ecran courant sans que soixante cartes fassent exploser le plafond de 16 Mo de la
// page (mesure : ~150 Ko par carte a ce cote, contre ~35 Ko pour la vignette).
const coteLoupeParDefaut = 900

type vignette struct {
	colonne   string
	dataURI   string
	loupeURI  string
	largeur   int
	hauteur   int
	occupL    float64
	occupA    float64
	matiereOK bool
}

type fiche struct {
	cle       string
	libelle   string
	sousTitre string
	statut    string
	vignettes []vignette
}

func main() {
	manifeste := flag.String("manifeste", "", "TSV : cle, libelle, sous-titre, statut, colonne, chemin png")
	sortie := flag.String("sortie", "planche.html", "fichier HTML a ecrire")
	titre := flag.String("titre", "Planche des fonds de carte", "titre de la page")
	intro := flag.String("intro", "", "paragraphe d'introduction (texte brut)")
	cote := flag.Int("cote", coteMaxParDefaut, "cote maximal d'une vignette, en pixels")
	coteLoupe := flag.Int("cote-loupe", coteLoupeParDefaut, "cote de l'image servie par la loupe, en pixels ; 0 = pas de loupe")
	flag.Parse()

	if *manifeste == "" {
		slog.Error("manifeste requis")
		os.Exit(1)
	}
	fiches, err := lisManifeste(*manifeste, *cote, *coteLoupe)
	if err != nil {
		slog.Error("lecture du manifeste", "err", err, "path", *manifeste)
		os.Exit(1)
	}
	page := rendPage(*titre, *intro, fiches)
	if err := os.WriteFile(*sortie, []byte(page), 0o644); err != nil {
		slog.Error("ecriture de la planche", "err", err, "path", *sortie)
		os.Exit(1)
	}
	n := 0
	for _, f := range fiches {
		n += len(f.vignettes)
	}
	slog.Info("planche ecrite", "path", *sortie, "fiches", len(fiches), "vignettes", n,
		"octets", len(page))
}

// lisManifeste lit le TSV et cuit une vignette par ligne. Une ligne illisible est SIGNALEE et
// sautee : une planche amputee et qui le dit vaut mieux qu'un arret sur la 40e carte.
func lisManifeste(chemin string, cote, coteLoupe int) ([]fiche, error) {
	f, err := os.Open(chemin)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var out []fiche
	index := map[string]int{}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		ligne := strings.TrimRight(sc.Text(), "\r")
		if ligne == "" || strings.HasPrefix(ligne, "#") {
			continue
		}
		c := strings.Split(ligne, "\t")
		if len(c) < 6 {
			slog.Warn("ligne ignoree : 6 colonnes attendues", "ligne", ligne)
			continue
		}
		v, err := cuitVignette(c[5], cote, coteLoupe)
		if err != nil {
			slog.Warn("vignette ignoree", "err", err, "path", c[5])
			continue
		}
		v.colonne = c[4]
		i, vu := index[c[0]]
		if !vu {
			out = append(out, fiche{cle: c[0], libelle: c[1], sousTitre: c[2], statut: c[3]})
			i = len(out) - 1
			index[c[0]] = i
		}
		out[i].vignettes = append(out[i].vignettes, v)
	}
	return out, sc.Err()
}

// cuitVignette reduit un fond a la taille d'affichage et mesure son occupation. La reduction
// est une MOYENNE DE BLOC (pas un echantillonnage) : sur une carte faite de traits fins, un
// plus-proche-voisin en perdrait la moitie et donnerait a juger une image qui n'existe pas.
func cuitVignette(chemin string, cote, coteLoupe int) (vignette, error) {
	blob, err := os.ReadFile(chemin)
	if err != nil {
		return vignette{}, err
	}
	src, err := png.Decode(bytes.NewReader(blob))
	if err != nil {
		return vignette{}, err
	}
	b := src.Bounds()
	v := vignette{largeur: b.Dx(), hauteur: b.Dy()}
	v.occupL, v.occupA, v.matiereOK = mesureOccupation(src)

	rgba := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(rgba, rgba.Bounds(), src, b.Min, draw.Src)
	petit := reduitParBloc(rgba, cote)

	buf, err := encodePNG(petit)
	if err != nil {
		return vignette{}, err
	}
	v.dataURI = buf
	if coteLoupe > cote {
		grande, err := encodePNG(reduitParBloc(rgba, coteLoupe))
		if err != nil {
			return vignette{}, err
		}
		v.loupeURI = grande
	}
	return v, nil
}

// encodePNG encode une image en data URI PNG, compression maximale : la page embarque tout,
// chaque kilo-octet compte contre le plafond de 16 Mo.
func encodePNG(img *image.RGBA) (string, error) {
	var buf bytes.Buffer
	if err := (&png.Encoder{CompressionLevel: png.BestCompression}).Encode(&buf, img); err != nil {
		return "", err
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

// mesureOccupation rend la part de la LARGEUR et de l'AIRE du cadre reellement couvertes par
// de la matiere. Meme seuil d'alpha que `cmd/mapfond-cadrage`.
func mesureOccupation(img image.Image) (occupL, occupA float64, ok bool) {
	b := img.Bounds()
	minX, maxX, n := b.Max.X, b.Min.X-1, 0
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if _, _, _, a := img.At(x, y).RGBA(); a>>8 < 8 {
				continue
			}
			n++
			if x < minX {
				minX = x
			}
			if x > maxX {
				maxX = x
			}
		}
	}
	if n == 0 {
		return 0, 0, false
	}
	return float64(maxX-minX+1) / float64(b.Dx()), float64(n) / float64(b.Dx()*b.Dy()), true
}

// reduitParBloc rend l'image ramenee a `cote` sur son plus grand cote, par moyenne de bloc en
// alpha PRE-MULTIPLIE : moyenner des couleurs sans tenir compte de leur opacite ferait baver
// le noir du vide transparent sur les bords de la matiere.
func reduitParBloc(src *image.RGBA, cote int) *image.RGBA {
	b := src.Bounds()
	if b.Dx() <= cote && b.Dy() <= cote {
		return src
	}
	f := float64(cote) / float64(max(b.Dx(), b.Dy()))
	w, h := max(1, int(float64(b.Dx())*f)), max(1, int(float64(b.Dy())*f))
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		y0, y1 := y*b.Dy()/h, (y+1)*b.Dy()/h
		for x := 0; x < w; x++ {
			x0, x1 := x*b.Dx()/w, (x+1)*b.Dx()/w
			var sr, sg, sb, sa, n float64
			for sy := y0; sy < max(y1, y0+1); sy++ {
				for sx := x0; sx < max(x1, x0+1); sx++ {
					i := src.PixOffset(sx, sy)
					a := float64(src.Pix[i+3]) / 255
					sr += float64(src.Pix[i]) * a
					sg += float64(src.Pix[i+1]) * a
					sb += float64(src.Pix[i+2]) * a
					sa += a
					n++
				}
			}
			i := dst.PixOffset(x, y)
			if sa > 0 {
				dst.Pix[i] = uint8(sr / sa)
				dst.Pix[i+1] = uint8(sg / sa)
				dst.Pix[i+2] = uint8(sb / sa)
			}
			dst.Pix[i+3] = uint8(255 * sa / n)
		}
	}
	return dst
}

func rendPage(titre, intro string, fiches []fiche) string {
	var b strings.Builder
	b.WriteString(`<title>` + html.EscapeString(titre) + `</title>` + "\n<style>\n" + feuilleDeStyle + "\n</style>\n")
	b.WriteString(`<header class="tete">` + "\n")
	b.WriteString(`<p class="surtitre">Rejeu 2D &middot; revue carte par carte</p>` + "\n")
	b.WriteString(`<h1>` + html.EscapeString(titre) + `</h1>` + "\n")
	if intro != "" {
		for _, p := range strings.Split(intro, "\n\n") {
			b.WriteString(`<p class="intro">` + html.EscapeString(p) + `</p>` + "\n")
		}
	}
	b.WriteString(`<p class="legende"><span class="ex"></span>Le trait plein est le CADRE de l'image ; le trait pointille borne la matiere dessinee. L'ecart entre les deux est le defaut de cadrage.</p>` + "\n")
	b.WriteString("</header>\n<main class=\"planche\">\n")
	for _, f := range fiches {
		b.WriteString(rendFiche(f))
	}
	b.WriteString("</main>\n")
	b.WriteString(calqueLoupe)
	return b.String()
}

func rendFiche(f fiche) string {
	var b strings.Builder
	b.WriteString(`<article class="fiche">` + "\n<div class=\"entete\">\n")
	b.WriteString(`<h2>` + html.EscapeString(f.libelle) + `</h2>` + "\n")
	if f.statut != "" {
		b.WriteString(`<span class="statut ` + classeStatut(f.statut) + `">` + html.EscapeString(f.statut) + `</span>` + "\n")
	}
	b.WriteString("</div>\n")
	if f.sousTitre != "" {
		b.WriteString(`<p class="sous">` + html.EscapeString(f.sousTitre) + `</p>` + "\n")
	}
	b.WriteString(`<div class="colonnes">` + "\n")
	for _, v := range f.vignettes {
		b.WriteString(`<figure>` + "\n")
		alt := html.EscapeString(f.libelle + " — " + v.colonne)
		if v.loupeURI != "" {
			// La vignette devient un BOUTON : la loupe se prend a la souris comme au clavier.
			b.WriteString(`<button class="cadre loupe" type="button" data-grande="` + v.loupeURI +
				`" data-titre="` + alt + `"><img alt="` + alt + `" src="` + v.dataURI +
				`"><span class="loupe-ind" aria-hidden="true">agrandir</span></button>` + "\n")
		} else {
			b.WriteString(`<div class="cadre"><img alt="` + alt + `" src="` + v.dataURI + `"></div>` + "\n")
		}
		b.WriteString(`<figcaption><span class="col">` + html.EscapeString(v.colonne) + `</span>`)
		b.WriteString(fmt.Sprintf(`<span class="chiffres">%d&times;%d px &middot; largeur utile <b>%.0f&nbsp;%%</b> &middot; aire <b>%.0f&nbsp;%%</b></span>`,
			v.largeur, v.hauteur, 100*v.occupL, 100*v.occupA))
		b.WriteString("</figcaption>\n</figure>\n")
	}
	b.WriteString("</div>\n</article>\n")
	return b.String()
}

func classeStatut(s string) string {
	switch {
	case strings.HasPrefix(s, "VALID"):
		return "ok"
	case strings.HasPrefix(s, "REFUS"):
		return "ko"
	default:
		return "attente"
	}
}
