package killsource

// minibobine_test.go — LE GOLDEN QUI TOURNE EN CI, SANS RIEN DEMANDER A PERSONNE.
//
// POURQUOI IL EXISTE. [TestGoldenFilms] fige les chiffres du chantier sur les quatre films de
// reference — 107 Mo qui ne sont pas versionnes — et il se met donc en `t.Skip` sans
// `KILLSOURCE_FIXTURES`. Consequence mesuree le 2026-08-02 : le commit `47c9e72ac` a pu annoncer
// << killsource (golden) vert >> en toute bonne foi alors que la sortie du decodeur avait bouge
// sur les quatre films. Personne n a menti ; le gate etait une option.
//
// C EST LA TROISIEME FOIS que le meme motif passe : le garde feedback-drawer grepait un chemin
// inexistant (J4.0), le ratchet knip ne tournait que sous condition de chemin (J3.3), et ce
// golden-ci se skippe sur une variable d environnement. **Un gate optionnel est un garde qui ne
// peut pas echouer, et un garde qui ne peut pas echouer ne garde rien.**
//
// CE QUE CE FICHIER AJOUTE. Une bobine de PAQUETS REELS, versionnee (3,8 Mo), et un golden qui
// tourne INCONDITIONNELLEMENT — donc en CI, dans `go test ./...`, sans fixture et sans variable.
//
// CE QUE LA BOBINE EST : un PREFIXE CONTIGU du film `000d5950` (chunks 00 a 05, en-tete compris)
// suivi de son chunk HIGHLIGHT. Le prefixe est contigu et part du debut PARCE QUE LE DECODEUR
// L EXIGE : le monde se construit par accumulation depuis l en-tete, et une bobine faite de
// paquets cherry-pickes — le patron de la mini-bobine du rejeu, `internal/analysis/replay` — ne
// decode ici plus AUCUNE mort. Mesure a l appui : en-tete + chunk 01 + highlight (644 Ko) rend
// ZERO candidat ; le prefixe 00-03 en rend 6 ; le prefixe 00-05 en rend 10. C est ce dernier qui
// est retenu, parce que c est lui qui atteint la ligne `01:12`.
//
// CE QU ELLE VERROUILLE, ET C EST DELIBERE :
//
//	LES 10 LIGNES PUBLIEES, EN ENTIER   tag, etiquette, categorie, credit, voie, parts de degats.
//	                                    Un compte peut rester juste pendant qu une etiquette
//	                                    devient fausse : ce sont les lignes qui l attrapent.
//	TROIS ANCRES THEATER                00:35 Disruptor, 00:44 Skewer, 01:12 M41 SPNKr — trois des
//	                                    onze confrontations confirmees de `000d5950` tombent dans
//	                                    le prefixe. Le garde protege donc de la verite terrain,
//	                                    pas seulement de la reproductibilite.
//	LA LIGNE `01:12`                    LA divergence : la source appartient a la victime et le
//	                                    jeu credite un autre joueur. La doctrine des deux verites
//	                                    se verifie la, et nulle part ailleurs dans cette bobine.
//	LA CALIBRATION ET `recordStateParam` le canal de detection PROUVE (voir ci-dessous).
//
// CE QU ELLE NE VERROUILLE PAS, ET IL FAUT LE DIRE. La calibration automatique tombe ici en
// PROFIL PLAT : l echantillon de paquets type-0 sans event du prefixe ne rend aucun record de
// biped, le balayage ne departage rien et les valeurs par defaut sont conservees. Le couple
// (axisW, indexW) de ce golden est donc le DEFAUT, pas un resultat de balayage — seul
// [TestGoldenFilms] verrouille le balayage lui-meme, sur les films entiers.
//
// TEMOIN DE DETECTION, JOUE LE 2026-08-02 — sans lui ce fichier ne serait qu une esperance.
// En neutralisant le `br.ReadBits(2)` de `consumeAbsoluteWithGate` (la cause etablie de la
// derive des quatre goldens), la sortie de CETTE bobine CHANGE : `recordStateParam` passe de 0
// a 2 et le rapport de croissance de 1.000 a 1.004. Les dix lignes publiees, elles, ne bougent
// pas — ce qui est coherent avec la mesure du meme jour (371 lignes sur 371 identiques sur les
// films entiers). **Le canal de detection de ce garde est la ligne de calibration, pas les
// lignes publiees.** Ecrit ici parce qu un garde dont on ignore PAR OU il detecte est un garde
// qu on croit avoir.
//
// REGENERATION DU GOLDEN (aucune fixture requise) :
//
//	go test ./internal/games/halo_infinite/film/killsource/ -run TestGoldenMiniBobine -update
//
// REGENERATION DE LA BOBINE ELLE-MEME (fixture requise, jamais d edition a la main) :
//
//	KILLSOURCE_FIXTURES=<racine des films> \
//	  go test ./internal/games/halo_infinite/film/killsource/ -run TestMiniBobineRegenerer -update

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis"
)

// miniBobineDir : la bobine versionnee, relative au paquet.
const miniBobineDir = "testdata/minibobine_000d5950"

// miniBobineFilm : le film dont la bobine est un prefixe. Sert a nommer la provenance.
const miniBobineFilm = "000d5950"

// miniBobinePrefixe : nombre de chunks du PREFIXE CONTIGU conserve, en-tete (chunk 00) compris.
// Passer de 4 a 6 fait passer la bobine de 6 a 10 morts publiees et lui apporte la ligne `01:12`,
// la seule divergence du prefixe.
const miniBobinePrefixe = 6

// miniBobineChunks : nombre total de fichiers attendus — le prefixe, plus le chunk HIGHLIGHT.
const miniBobineChunks = miniBobinePrefixe + 1

// miniBobinePlancher : nombre MINIMAL de lignes publiees attendu.
//
// C EST LUI QUI REND LE GARDE AUTO-VERIFIANT, et il n est pas decoratif : sans plancher, une
// bobine tronquee ou absente ferait passer le test des lors que quelqu un regenererait le golden
// sur une sortie vide. Un golden fige une sortie ; il ne dit pas qu elle est NON TRIVIALE. C est
// exactement le defaut corrige en J4.0 sur le garde feedback-drawer — echouer bruyamment quand
// la cible n existe pas.
const miniBobinePlancher = 10

// TestGoldenMiniBobine : LE GOLDEN INCONDITIONNEL. Aucune variable d environnement, aucune
// fixture hors depot — il tourne partout ou `go test ./...` tourne, donc en CI.
func TestGoldenMiniBobine(t *testing.T) {
	src, err := DirChunks(miniBobineDir)
	if err != nil {
		t.Fatalf("mini-bobine illisible sous %s : %v — elle est VERSIONNEE, son absence est une "+
			"erreur, pas une raison d ignorer le test", miniBobineDir, err)
	}
	if n := src.NumChunks(); n != miniBobineChunks {
		t.Fatalf("mini-bobine incomplete : %d chunk(s) sous %s, %d attendus (%d de prefixe + le "+
			"chunk HIGHLIGHT)", n, miniBobineDir, miniBobineChunks, miniBobinePrefixe)
	}
	res, err := Decode(t.Context(), miniBobineFilm, src, nil)
	if err != nil {
		t.Fatalf("Decode sur la mini-bobine : %v", err)
	}
	if len(res.Kills) < miniBobinePlancher {
		t.Fatalf("la mini-bobine ne publie que %d ligne(s), plancher %d : le garde ne garderait "+
			"plus rien — bobine tronquee, ou decodeur casse au point de ne plus rien rendre",
			len(res.Kills), miniBobinePlancher)
	}
	comparerGolden(t, "minibobine", rendreMiniBobine(res, controleNegatif(t, src, len(res.Roster.IndexToName))))
}

// rendreMiniBobine : la sortie figee de la bobine. Elle reutilise les sections du rendu des films
// entiers — meme vocabulaire, memes phrases qualifiantes — et y ajoute LES LIGNES, toutes.
func rendreMiniBobine(res *Result, neg negControle) string {
	var b strings.Builder
	fmt.Fprintf(&b, enteteMiniBobine, miniBobineFilm, miniBobinePrefixe)
	sectionCouverture(&b, res.Coverage)
	sectionLignesMiniBobine(&b, res)
	sectionAutoInfligees(&b, res)
	sectionControleNegatif(&b, neg)
	sectionVoies(&b, res)
	sectionSante(&b, res)
	return b.String()
}

// enteteMiniBobine : la recette dans le fichier, comme pour les goldens de film — un fichier fige
// qui ne dit pas comment il a ete produit devient indechiffrable des que son auteur n est plus la.
const enteteMiniBobine = `# GOLDEN killsource — MINI-BOBINE (prefixe du film %s)
#
# Fige la sortie du decodeur sur des PAQUETS REELS VERSIONNES. Ne PAS editer a la main.
#
# CE GOLDEN TOURNE EN CI, SANS FIXTURE ET SANS VARIABLE D ENVIRONNEMENT. C est sa raison
# d etre : TestGoldenFilms, lui, se skippe sans KILLSOURCE_FIXTURES, et un commit a deja pu
# annoncer un golden vert alors que la sortie avait bouge (2026-08-02, commit 47c9e72ac).
#
# LA BOBINE est le PREFIXE CONTIGU des %d premiers chunks du film, en-tete compris, suivi de son
# chunk HIGHLIGHT. Contigu et depuis le debut PARCE QUE LE DECODEUR L EXIGE : le monde
# s accumule depuis l en-tete. Une bobine de paquets cherry-pickes n y decode AUCUNE mort.
#
# PORTEE — la calibration automatique tombe ici en PROFIL PLAT (l echantillon du prefixe ne rend
# aucun record de biped) : le couple axisW/indexW ci-dessous est le DEFAUT, pas un resultat de
# balayage. Seul TestGoldenFilms verrouille le balayage, sur les films entiers.
#
# REGENERATION :
#   go test ./internal/games/halo_infinite/film/killsource/ -run TestGoldenMiniBobine -update
`

// sectionLignesMiniBobine : LES LIGNES PUBLIEES, TOUTES, avec tout ce qui les qualifie.
//
// Sur un film entier ce bloc serait illisible (371 lignes) et les goldens de film n en figent
// donc que les ancres. Ici elles sont dix : les ecrire toutes coute quinze lignes et verrouille
// l etiquette de CHAQUE mort, pas seulement celle des ancres.
func sectionLignesMiniBobine(b *strings.Builder, res *Result) {
	fmt.Fprintf(b, "\n## LES %d LIGNES PUBLIEES, EN ENTIER — un compte juste peut porter une "+
		"etiquette fausse\n", len(res.Kills))
	fmt.Fprint(b, "# format : instant  victime  tag  etiquette  categorie  CREDIT DU JEU  "+
		"assistant  voie  origine  divergence\n")
	for _, k := range res.Kills {
		fmt.Fprintf(b, "%s  %s  %08x  %q  %s  credit=%s  assistant=%s  %s  %s  divergence=%s\n",
			clock(k.TimeMS), k.Victim, k.Source.Tag, k.Source.Display, k.Source.Class,
			k.Feed.Killer, assistantGolden(k.Assist), k.Read.Path, k.Read.Origin,
			ouiNonGolden(k.Diverges))
	}
}

// TestMiniBobineRegenerer : la RECETTE DE FABRICATION de la bobine, executable.
//
// Elle est un test et non un `cmd/` parce qu elle vit avec ce qu elle produit : une recette
// rangee ailleurs se perime sans que rien ne le signale. Elle exige la fixture ET `-update` —
// deux verrous, parce qu elle ECRASE des octets versionnes.
//
// LE CHUNK HIGHLIGHT EST TROUVE PAR SON CONTENU, jamais par son numero : il est en n27 sur ce
// film, en n62 sur un BTB, et l outillage de RE a deja perdu un kill-feed entier en le bornant
// (RE_LOG 7ter.52).
func TestMiniBobineRegenerer(t *testing.T) {
	dir := os.Getenv("KILLSOURCE_FIXTURES")
	if dir == "" || !*updateGolden {
		t.Skip("regeneration de la bobine : exige KILLSOURCE_FIXTURES et -update (elle ecrase " +
			"des octets versionnes)")
	}
	source := filepath.Join(dir, miniBobineFilm)
	src, err := DirChunks(source)
	if err != nil {
		t.Fatalf("film source illisible : %v", err)
	}
	hi := chunkHighlight(t, src)
	if err := os.MkdirAll(miniBobineDir, 0o750); err != nil {
		t.Fatalf("creation de %s : %v", miniBobineDir, err)
	}
	for i := 0; i < miniBobinePrefixe; i++ {
		copierChunk(t, source, i, i)
	}
	copierChunk(t, source, hi, miniBobinePrefixe)
	ecrireProvenance(t, hi)
	t.Logf("bobine regeneree : %d chunk(s) de prefixe + le chunk HIGHLIGHT (n%d du film)",
		miniBobinePrefixe, hi)
}

// chunkHighlight : l index du chunk qui porte le plus de kills — la meme detection PAR CONTENU
// que [loadKillFeed], et pour la meme raison.
func chunkHighlight(t *testing.T, src ChunkSource) int {
	t.Helper()
	best, bestN := -1, 0
	for i := 0; i < src.NumChunks(); i++ {
		raw, err := src.Chunk(i)
		if err != nil {
			continue
		}
		evs, err := analysis.ParseHighlightEvents(raw, 0)
		if err != nil {
			continue
		}
		n := 0
		for _, e := range evs {
			if e.EventType == analysis.EventTypeKill {
				n++
			}
		}
		if n > bestN {
			best, bestN = i, n
		}
	}
	if best < 0 {
		t.Fatalf("aucun chunk HIGHLIGHT dans le film source : la bobine serait sans kill-feed")
	}
	return best
}

// copierChunk : recopie les octets BRUTS, sans les decompresser ni les toucher. La bobine doit
// etre du film, pas une re-serialisation.
func copierChunk(t *testing.T, source string, from, to int) {
	t.Helper()
	name := fmt.Sprintf("chunk_%02d.bin", from)
	raw, err := os.ReadFile(filepath.Join(source, name)) //nolint:gosec // chemin construit d un index
	if err != nil {
		t.Fatalf("lecture de %s : %v", name, err)
	}
	dst := filepath.Join(miniBobineDir, fmt.Sprintf("chunk_%02d.bin", to))
	if err := os.WriteFile(dst, raw, 0o600); err != nil {
		t.Fatalf("ecriture de %s : %v", dst, err)
	}
}

// ecrireProvenance : la provenance de chaque octet, versionnee AVEC la bobine.
func ecrireProvenance(t *testing.T, hi int) {
	t.Helper()
	var b strings.Builder
	fmt.Fprintf(&b, "MINI-BOBINE killsource — provenance\n\n")
	fmt.Fprintf(&b, "Film source        : %s (data/cache/film_chunks/%s)\n",
		miniBobineFilm, miniBobineFilm)
	fmt.Fprintf(&b, "Prefixe conserve   : chunks 00 a %02d du film, EN-TETE COMPRIS, octets bruts\n",
		miniBobinePrefixe-1)
	fmt.Fprintf(&b, "Chunk HIGHLIGHT    : n%d du film, recopie en chunk_%02d.bin\n",
		hi, miniBobinePrefixe)
	fmt.Fprintf(&b, "\nPOURQUOI UN PREFIXE CONTIGU ET NON DES PAQUETS CHOISIS. Le decodeur "+
		"construit son\nmonde par accumulation depuis l en-tete. Une bobine de paquets "+
		"cherry-pickes — le patron\nde la mini-bobine du rejeu — n y decode AUCUNE mort "+
		"(mesure : 0 candidat).\n")
	fmt.Fprintf(&b, "\nAucun octet n est modifie. Regeneration : voir minibobine_test.go.\n")
	p := filepath.Join(miniBobineDir, "PROVENANCE.txt")
	if err := os.WriteFile(p, []byte(b.String()), 0o600); err != nil {
		t.Fatalf("ecriture de %s : %v", p, err)
	}
}
