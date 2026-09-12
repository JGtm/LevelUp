package killsource

// minibobine_v40_test.go — LA BOBINE DE VERSION 40, ET LE GARDE QUI MORD SUR LE DECOUPAGE DECALE.
//
// # LE DEFAUT QU IL FERME
//
// Le correctif du 2026-09-12 fait LIRE la version du film dans l en-tete de son registre au lieu
// de passer 0 en dur au parseur d events (.ai/RAPPORT_BTB_2025_ABSTENTION_2026-09-12.md). Revue
// adversariale du meme jour, constat P1-1 : REPASSER 0 a la place de `f.majorVersion` dans
// [loadKillFeed] laissait toute la CI verte. La raison est mecanique — la seule bobine versionnee
// etait `minibobine_000d5950`, un film de version 41, et sur une version >= 41 le decoupage 0 et
// le decoupage 41 sont LE MEME (gamertag a l octet 0). Le correctif n avait donc aucun temoin.
//
// # CE QUE CETTE BOBINE AJOUTE
//
// Un film de version 40, versionne, ou les deux decoupages DIVERGENT franchement :
//
//	version lue (40)  roster de 26 noms, 26 gamertags distincts
//	version 0         roster de 11 entrees, 2 gamertags distincts (le reste retombe sur `xuid:`)
//
// Il n y a pas de zone grise : le plancher de ce test est a 20, et la mutation rend 11.
//
// # CE QU ELLE CONTIENT, ET POURQUOI EXACTEMENT CELA
//
//	chunk_00.bin  LE REGISTRE — c est lui qui PORTE la version (u32 LE en tete). Sans lui la
//	              bobine ne prouverait rien : `filmdec.FilmMajorVersion` rendrait `ok=false`.
//	chunk_01.bin  LE PREMIER CHUNK DE DONNEES — [loadFilm] refuse un film sans paquet de
//	              replication (`ErrNoPacket`) ; c est le plus petit chunk qui lui en donne.
//	chunk_02.bin  LE CHUNK HIGHLIGHT du film (n30), trouve PAR SON CONTENU comme partout ailleurs.
//
// Elle ne cherche PAS a publier des lignes de source de degat — c est le role de
// `minibobine_000d5950`, qui garde un prefixe contigu long. Ici on garde le fil du kill-feed et
// rien d autre : trois chunks, 876 Kio zlib, contre 3,8 Mio pour la bobine de rejeu.
//
// # LES OCTETS SONT COMPRESSES, ET C EST LE SEUL ECART AVEC LA RECETTE DE `minibobine_000d5950`
//
// Le cache de films local stocke desormais les chunks DECOMPRESSES (mesure du 2026-09-12 : 0
// registre compresse sur 1 351). Les recopier tels quels pesait 3,0 Mio ; recompresses en zlib
// ils pesent 876 Kio, et `filmsource` les inflate au chargement exactement comme il inflate ceux
// de `minibobine_000d5950`, qui sont zlib eux aussi. Aucun octet DECOMPRESSE ne change.
//
// # REGENERATION DE LA BOBINE (fixture requise, jamais d edition a la main)
//
//	KILLSOURCE_FIXTURES=<racine des films> \
//	  go test ./internal/games/halo_infinite/film/killsource/ -run TestMiniBobineV40Regenerer -update

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/analysis/filmsource"
)

// miniBobineV40Dir : la bobine de version 40, relative au paquet.
//
// ELLE A DEUX LECTEURS : ce paquet, et `internal/analysis/replay` dont `ScanDeaths` est le
// deuxieme des trois appelants qui passaient 0 (constat P1-2) — il la designe par un chemin
// relatif, comme `filmdec` designe deja `replay/testdata/minifilm_000d5950` dans l autre sens.
// Un second exemplaire de 876 Kio pour la meme preuve serait de la dette.
const miniBobineV40Dir = "testdata/minibobine_e5adf7b2"

// miniBobineV40Film : le film dont la bobine est tiree (2025-07, Fragmentation, Big Team Battle).
const miniBobineV40Film = "e5adf7b2"

// miniBobineV40Version : la version que son registre declare. C est la valeur qui rend la bobine
// utile — sur une version 39 ou 40, et sur elles seules, le gamertag vit a l octet 12 du bloc
// d event.
const miniBobineV40Version = 40

// miniBobineV40Plancher : le nombre MINIMAL de gamertags distincts attendu sous la version lue.
//
// Mesure du 2026-09-12 : 26 sous la version 40, 2 sous la version 0. Le plancher est pose a 20 —
// assez haut pour que la mutation « version 0 » (11 entrees de roster, 2 gamertags) le creve,
// assez bas pour ne pas figer un compte exact que la moindre correction de parseur ferait bouger.
const miniBobineV40Plancher = 20

// miniBobineV40Chunks : nombre de fichiers attendus dans la bobine.
const miniBobineV40Chunks = 3

// TestMiniBobineV40RosterSuitLaVersionLue — LE GARDE. Il tourne en CI, sans variable et sans
// fixture hors depot.
//
// IL VERIFIE DEUX CHOSES, ET LES DEUX SONT NECESSAIRES : que [loadFilm] LIT bien la version dans
// le registre (sans quoi le reste ne prouverait rien), et que [loadKillFeed] la PASSE au parseur
// (le geste que la mutation defait).
func TestMiniBobineV40RosterSuitLaVersionLue(t *testing.T) {
	src := chargerMiniBobineV40(t)
	f, err := loadFilm(src)
	if err != nil {
		t.Fatalf("loadFilm sur la bobine v40 : %v", err)
	}
	if !f.versionLue {
		t.Fatalf("la bobine v40 ne porte pas son registre : `chunk_00.bin` est-il present ? "+
			"(%d chunk(s) lus)", src.NumChunks())
	}
	if f.majorVersion != miniBobineV40Version {
		t.Fatalf("version lue %d, %d attendue — la bobine a change, ou l en-tete du registre "+
			"n est plus lu au meme endroit", f.majorVersion, miniBobineV40Version)
	}
	kf, err := loadKillFeed(f)
	if err != nil {
		t.Fatalf("loadKillFeed sur la bobine v40 : %v", err)
	}
	gamertags := 0
	for _, n := range kf.names {
		if !strings.HasPrefix(n, XUIDNamePrefix) {
			gamertags++
		}
	}
	if len(kf.names) < miniBobineV40Plancher || gamertags < miniBobineV40Plancher {
		t.Fatalf(`LE DECOUPAGE DU GAMERTAG NE SUIT PLUS LA VERSION DU FILM.

  bobine    : %s (film %s, version %d)
  roster    : %d entree(s), dont %d gamertag(s) — plancher %d
  attendu   : 26 et 26 (mesure du 2026-09-12)

Sur un film de version 39-40 le gamertag vit a l OCTET 12 du bloc d event, pas a l octet 0.
Passer 0 au lieu de la version lue rend ici 11 entrees et 2 gamertags : le roster humain
s effondre, et les portes « indice < nPlay » du decodeur de source de degat rejettent les trois
quarts des morts (.ai/RAPPORT_BTB_2025_ABSTENTION_2026-09-12.md).

Verifier que loadFilm pose majorVersion/versionLue et que loadKillFeed passe f.majorVersion
a analysis.ParseHighlightEvents.`,
			miniBobineV40Dir, miniBobineV40Film, f.majorVersion,
			len(kf.names), gamertags, miniBobineV40Plancher)
	}
}

// chargerMiniBobineV40 : la bobine, avec le message qu il faut quand elle manque. Elle est
// VERSIONNEE — son absence est une erreur, jamais une raison de se skipper.
func chargerMiniBobineV40(t *testing.T) *filmsource.Film {
	t.Helper()
	src, err := filmsource.LoadDir(miniBobineV40Dir, nil)
	if err != nil {
		t.Fatalf("bobine v40 illisible sous %s : %v", miniBobineV40Dir, err)
	}
	if n := src.NumChunks(); n != miniBobineV40Chunks {
		t.Fatalf("bobine v40 incomplete : %d chunk(s) sous %s, %d attendus (registre, premier "+
			"chunk de donnees, chunk HIGHLIGHT)", n, miniBobineV40Dir, miniBobineV40Chunks)
	}
	return src
}

// TestMiniBobineV40Regenerer : la RECETTE DE FABRICATION, executable — meme doctrine que
// [TestMiniBobineRegenerer] : elle vit avec ce qu elle produit, et elle exige la fixture ET
// `-update` parce qu elle ECRASE des octets versionnes.
func TestMiniBobineV40Regenerer(t *testing.T) {
	dir := os.Getenv("KILLSOURCE_FIXTURES")
	if dir == "" || !*updateGolden {
		t.Skip("regeneration de la bobine v40 : exige KILLSOURCE_FIXTURES et -update (elle " +
			"ecrase des octets versionnes)")
	}
	source := filepath.Join(dir, miniBobineV40Film)
	src, err := filmsource.LoadDir(source, nil)
	if err != nil {
		t.Fatalf("film source illisible : %v", err)
	}
	if v, ok := filmdec.FilmMajorVersion(src); !ok || v != miniBobineV40Version {
		t.Fatalf("le film %s declare la version %d (lue=%v) : la bobine doit venir d un film de "+
			"version %d, sinon elle ne prouve rien", miniBobineV40Film, v, ok, miniBobineV40Version)
	}
	hi := chunkHighlight(t, src)
	if err := os.MkdirAll(miniBobineV40Dir, 0o750); err != nil {
		t.Fatalf("creation de %s : %v", miniBobineV40Dir, err)
	}
	for i, from := range []int{0, 1, hi} {
		copierChunkCompresse(t, source, from, i)
	}
	ecrireProvenanceV40(t, hi)
	t.Logf("bobine v40 regeneree : registre + chunk 01 + chunk HIGHLIGHT (n%d du film)", hi)
}

// copierChunkCompresse : recopie un chunk du film en le normalisant en zlib.
//
// L INFLATE PRECEDE LA COMPRESSION, et il le faut : le cache local stocke aujourd hui des chunks
// DECOMPRESSES, mais rien ne garantit qu une racine de fixture plus ancienne ne porte pas encore
// les octets zlib du CDN. `filmsource.Inflate` rend le tampon inchange quand il n est pas zlib :
// la bobine est donc la meme dans les deux cas.
func copierChunkCompresse(t *testing.T, source string, from, to int) {
	t.Helper()
	name := fmt.Sprintf("chunk_%02d.bin", from)
	raw, err := os.ReadFile(filepath.Join(source, name)) //nolint:gosec // chemin construit d un index
	if err != nil {
		t.Fatalf("lecture de %s : %v", name, err)
	}
	var buf bytes.Buffer
	w, err := zlib.NewWriterLevel(&buf, zlib.BestCompression)
	if err != nil {
		t.Fatalf("compresseur zlib : %v", err)
	}
	if _, err := w.Write(filmsource.Inflate(raw)); err != nil {
		t.Fatalf("compression de %s : %v", name, err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("fermeture du compresseur sur %s : %v", name, err)
	}
	dst := filepath.Join(miniBobineV40Dir, fmt.Sprintf("chunk_%02d.bin", to))
	if err := os.WriteFile(dst, buf.Bytes(), 0o600); err != nil {
		t.Fatalf("ecriture de %s : %v", dst, err)
	}
}

// ecrireProvenanceV40 : la provenance de chaque octet, versionnee AVEC la bobine.
func ecrireProvenanceV40(t *testing.T, hi int) {
	t.Helper()
	var b strings.Builder
	fmt.Fprintf(&b, "MINI-BOBINE killsource — VERSION %d — provenance\n\n", miniBobineV40Version)
	fmt.Fprintf(&b, "Film source        : %s (data/cache/film_chunks/%s), 2025-07, Big Team Battle\n",
		miniBobineV40Film, miniBobineV40Film)
	fmt.Fprintf(&b, "chunk_00.bin       : le REGISTRE du film — il porte la version (u32 LE en tete)\n")
	fmt.Fprintf(&b, "chunk_01.bin       : le premier chunk de DONNEES — loadFilm refuse un film\n")
	fmt.Fprintf(&b, "                     sans paquet de replication (ErrNoPacket)\n")
	fmt.Fprintf(&b, "chunk_02.bin       : le chunk HIGHLIGHT du film (n%d), trouve PAR SON CONTENU\n", hi)
	fmt.Fprintf(&b, "\nPOURQUOI UNE BOBINE DE VERSION 40. Sur les versions 39-40 le gamertag vit a\n"+
		"l octet 12 du bloc d event, pas a l octet 0 ; sur les versions >= 41 les deux decoupages\n"+
		"se confondent. La bobine 000d5950 (v41) ne pouvait donc pas prouver que la version lue\n"+
		"est bien PASSEE au parseur — celle-ci le peut (26 gamertags contre 2).\n")
	fmt.Fprintf(&b, "\nLes octets sont recompresses en zlib (le cache local les stocke decompresses) ;\n"+
		"leur contenu DECOMPRESSE est celui du film, inchange. Regeneration : voir\n"+
		"minibobine_v40_test.go.\n")
	p := filepath.Join(miniBobineV40Dir, "PROVENANCE.txt")
	if err := os.WriteFile(p, []byte(b.String()), 0o600); err != nil {
		t.Fatalf("ecriture de %s : %v", p, err)
	}
}
