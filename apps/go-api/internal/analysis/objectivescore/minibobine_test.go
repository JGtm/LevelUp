package objectivescore

// minibobine_test.go — LE GARDE QUI TOURNE EN CI, SANS RIEN DEMANDER À PERSONNE.
//
// POURQUOI IL EXISTE. Les deux autres gardes de ce paquet qui touchent des octets réels —
// `TestGoldenFilmsReels` et les tests cache-backed — exigent `FILM_CACHE_ROOT`, parce que
// les films pèsent des dizaines de Mo et ne sont pas versionnés. Ils se skippent donc en CI.
// Un gate optionnel est un garde qui ne peut pas échouer, et un garde qui ne peut pas
// échouer ne garde rien.
//
// CE MOTIF A DÉJÀ MORDU DEUX FOIS SUR CE CHANTIER. Le 2026-08-02, le commit `47c9e72ac` a pu
// annoncer « killsource (golden) vert » en toute bonne foi alors que la sortie du décodeur
// avait bougé sur les quatre films : le golden se skippait sans `KILLSOURCE_FIXTURES`. Le
// 2026-08-06, l'audit a trouvé le seul test de CE paquet-ci sur film réel éteint depuis des
// mois par un chemin absolu mort. Deux fois la même forme : un vert qui ne prouve rien.
// `killsource/minibobine_test.go` a été la réponse là-bas ; ce fichier est la même réponse
// ici, et il en reprend la doctrine.
//
// CE QUE LA BOBINE EST. Pour deux films, un petit nombre de chunks TYPE-2, chacun TRONQUÉ
// juste après son premier paquet TYPE_2 complet, puis rangé en zlib. Aucun octet n'est
// réécrit : ce qui est versionné est un PRÉFIXE d'octets bruts du chunk d'origine, sous la
// forme comprimée que le cache film emploie lui-même (le harnais décompresse au magic 0x78,
// comme les décodeurs). La troncature est possible ici — et ne l'était pas pour killsource,
// dont le monde s'accumule depuis l'en-tête — parce que ce décodeur est SANS ÉTAT d'un chunk
// à l'autre : `anchoredPayload` ne regarde que le premier paquet TYPE_2 du chunk qu'on lui
// donne.
//
// LES DEUX TRANSFORMATIONS SONT PROUVÉES NON FAUSSANTES, pas supposées. La recette de
// fabrication (`TestMiniBobineObjectifsRegenerer`) refuse d'écrire un chunk si le préfixe ne
// décode pas EXACTEMENT comme le chunk entier (même position de token, mêmes varints), et si
// la décompression de ce qu'elle s'apprête à écrire ne rend pas octet pour octet le préfixe
// d'origine. Les 8 Mo de la version non comprimée tenaient à des plages de zéros : 340 Ko
// suffisent aux mêmes octets.
//
// CE QU'ELLE VERROUILLE :
//
//	LA POSITION DE BIT DU TOKEN     donc `scoreToken`, `tokenWinLo`, `tokenWinHi`.
//	LES VALEURS BRUTES PAR CHUNK    donc `shOffTeam0`, `shOffTeam1`, et les quatre offsets
//	                                KOTH. C'est LE canal de détection de ce garde : décaler
//	                                un offset change ces colonnes.
//	LES VALEURS PUBLIÉES            ce que l'appelant recevrait.
//	DEUX PLANCHERS                  un nombre minimal de frames ancrées ET une courbe qui
//	                                monte réellement. Sans eux, une bobine tronquée ou un
//	                                décodeur muet régénéreraient un golden vide, vert et
//	                                creux — le défaut corrigé en J4.0 sur le garde
//	                                feedback-drawer.
//
// CE QU'ELLE NE VERROUILLE PAS, ET IL FAUT LE DIRE. La calibration Strongholds
// (`calibrateByFinal`) remet la dernière frame sur le final passé en argument : sur une
// bobine tronquée, les colonnes calibrées ne sont donc PAS le score du match, seulement la
// mise à l'échelle de la portion retenue. La justesse du décodeur, elle, n'est verrouillée
// nulle part — elle est RÉFUTÉE, et de longue date : voir l'en-tête du paquet et
// `golden_films_test.go`.
//
// RÉGÉNÉRATION DU GOLDEN (aucune fixture requise) :
//
//	go test ./internal/analysis/objectivescore/ -run TestGoldenMiniBobineObjectifs -update
//
// RÉGÉNÉRATION DE LA BOBINE ELLE-MÊME (fixture requise, jamais d'édition à la main) :
//
//	FILM_CACHE_ROOT=<repo>/data/cache \
//	  go test ./internal/analysis/objectivescore/ -run TestMiniBobineObjectifsRegenerer -update

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// miniBobineRacine : la bobine versionnée, relative au paquet.
const miniBobineRacine = "testdata/minibobine_objectifs"

// miniBobineChunks : nombre de chunks TYPE-2 conservés par film.
//
// Ce sont les huit DERNIERS chunks ancrés, pas les huit premiers, et ce choix est le fruit
// d'une mesure : sur les huit premiers chunks de `0a247154` le score KOTH publié vaut encore
// 0-0 (les points n'arrivent qu'à 240 s), et une bobine qui fige des zéros ne distingue pas
// un décodeur juste d'un décodeur muet. Les huit derniers portent la fin de match — donc
// 4-2 pour KOTH, le SEUL résultat que la rétro-ingénierie ait validé exact.
const miniBobineChunks = 8

// miniBobinePlancherFrames : nombre MINIMAL de frames ancrées attendu, par bobine. Il rend
// le garde auto-vérifiant : sans lui, une bobine absente ou vidée passerait dès que
// quelqu'un régénérerait le golden sur une sortie vide.
const miniBobinePlancherFrames = 6

// bobineObjectifs : une bobine et ce qu'on en attend.
type bobineObjectifs struct {
	Dossier string // sous miniBobineRacine
	Film    string // film d'origine
	Variant string // game_variant_name à passer au dispatch
	FinalT0 int
	FinalT1 int
}

// corpusMiniBobines : les deux modes que ce décodeur sait traiter. Strongholds pour le
// chemin varint bit-aligné, KOTH `0a247154` pour le chemin byte-aligné — c'est le SEUL cas
// que la rétro-ingénierie a validé exact (koth.go), donc celui dont une dérive serait la
// plus coûteuse.
var corpusMiniBobines = []bobineObjectifs{
	{"strongholds_7344d24f", "7344d24f", "Arena:Strongholds", 193, 112},
	{"koth_0a247154", "0a247154", "Ranked:King of the Hill", 4, 2},
}

// TestGoldenMiniBobineObjectifs : LE GOLDEN INCONDITIONNEL. Aucune variable d'environnement,
// aucune fixture hors dépôt — il tourne partout où `go test ./...` tourne, donc en CI.
func TestGoldenMiniBobineObjectifs(t *testing.T) {
	var b strings.Builder
	b.WriteString(enteteMiniBobine)
	for _, bo := range corpusMiniBobines {
		chunks := chargerBobine(t, bo)
		frames := DecodeScoreTimeline(bo.Variant, chunks, bo.FinalT0, bo.FinalT1)
		if len(frames) < miniBobinePlancherFrames {
			t.Fatalf("%s : %d frame(s) ancrée(s), plancher %d — le garde ne garderait plus "+
				"rien : bobine tronquée, ou décodeur cassé au point de ne plus rien ancrer",
				bo.Dossier, len(frames), miniBobinePlancherFrames)
		}
		verifierBobineNonTriviale(t, bo, chunks, frames)
		rendreBobine(&b, bo, chunks, frames)
	}
	comparerGolden(t, "minibobine_objectifs", b.String())
}

// verifierBobineNonTriviale : le second plancher, et il vaut pour les deux modes. Un golden
// fige une sortie ; il ne dit pas qu'elle est NON TRIVIALE. Si le décodeur se mettait à lire
// des zéros partout, toutes les colonnes resteraient cohérentes entre elles et le golden
// régénéré serait vert, creux, et indistinguable d'un garde qui marche.
func verifierBobineNonTriviale(t *testing.T, bo bobineObjectifs, chunks []ChunkInput, frames []ScoreFrame) {
	t.Helper()
	if dernier := frames[len(frames)-1]; dernier.Team0 == 0 && dernier.Team1 == 0 {
		t.Fatalf("%s : la bobine publie 0-0 a la derniere frame — elle ne distingue plus un "+
			"decodeur juste d'un decodeur muet", bo.Dossier)
	}
	if len(signaturesLues(chunks)) < 2 {
		t.Fatalf("%s : le decodeur lit la MEME chose sur les %d chunks de la bobine — l'ancre "+
			"ne suit plus le bloc de score, ou les octets versionnes sont identiques",
			bo.Dossier, len(chunks))
	}
}

// signaturesLues : l'ensemble des valeurs distinctes que le décodeur lit sur la bobine, tous
// offsets confondus. Deux signatures suffisent à prouver qu'il lit quelque chose qui bouge.
func signaturesLues(chunks []ChunkInput) map[string]struct{} {
	_, raw0, raw1 := collectStrongholdsRaw(chunks)
	vues := make(map[string]struct{}, len(raw0))
	for i := range raw0 {
		o12, o13, o14, o16 := octetsKOTH(chunks, i)
		vues[fmt.Sprintf("%d/%d/%d,%d,%d,%d", raw0[i], raw1[i], o12, o13, o14, o16)] = struct{}{}
	}
	return vues
}

const enteteMiniBobine = `# GOLDEN objectivescore — MINI-BOBINES VERSIONNEES
#
# Fige la sortie du decodeur sur des PAQUETS REELS VERSIONNES. Ne PAS editer a la main.
#
# CE GOLDEN TOURNE EN CI, SANS FIXTURE ET SANS VARIABLE D ENVIRONNEMENT. C est sa raison
# d etre : TestGoldenFilmsReels et les tests cache-backed, eux, se skippent sans
# FILM_CACHE_ROOT. Deux fois deja sur ce chantier un vert a ete annonce sur un garde qui ne
# tournait pas (killsource 2026-08-02 ; le test cache-backed de ce paquet, mort depuis des
# mois, trouve par l audit du 2026-08-06).
#
# CHAQUE FICHIER DE LA BOBINE est un PREFIXE d octets bruts d un chunk TYPE-2, tronque juste
# apres son premier paquet TYPE_2 complet. Aucun octet n est reecrit. La troncature est
# licite parce que ce decodeur est SANS ETAT d un chunk a l autre, et la recette de
# fabrication REFUSE d ecrire si le chunk tronque ne decode pas comme le chunk entier.
#
# LES BRUTES SONT LA SUBSTANCE : ce sont bitpos, brut0/brut1 (Strongholds) et les octets
# lus (KOTH) qui repondent du token, de sa fenetre et des offsets. Les colonnes calibrees
# retombent sur le final PAR CONSTRUCTION et ne prouvent aucune position de bit.
#
# REGENERATION :
#   go test ./internal/analysis/objectivescore/ -run TestGoldenMiniBobineObjectifs -update
`

// rendreBobine : la section figée d'une bobine.
func rendreBobine(b *strings.Builder, bo bobineObjectifs, chunks []ChunkInput, frames []ScoreFrame) {
	type2, ancres := comptesAncrage(chunks)
	fmt.Fprintf(b, "\n## %s — film %s — %s (final passe au decodeur : %d-%d)\n",
		bo.Dossier, bo.Film, bo.Variant, bo.FinalT0, bo.FinalT1)
	fmt.Fprintf(b, "chunks=%d type2=%d ancres=%d token=0x%X fenetre=[%d,%d)\n",
		len(chunks), type2, ancres, scoreToken, tokenWinLo, tokenWinHi)
	_, raw0, raw1 := collectStrongholdsRaw(chunks)
	fmt.Fprintf(b, "# instant_ms  bitpos  brut0(+%d)  brut1(+%d)  octets[+12,+13,+14,+16]  publie\n",
		shOffTeam0, shOffTeam1)
	for i, f := range frames {
		o12, o13, o14, o16 := octetsKOTH(chunks, i)
		fmt.Fprintf(b, "%d  %d  %d  %d  %d,%d,%d,%d  %d-%d\n", f.TimeMS, bitposAt(chunks, i),
			raw0[i], raw1[i], o12, o13, o14, o16, f.Team0, f.Team1)
	}
	fmt.Fprintf(b, "source=%s conf=%s  fin publiee %d-%d\n", frames[0].Source,
		frames[0].Confidence, frames[len(frames)-1].Team0, frames[len(frames)-1].Team1)
}

// chargerBobine lit une bobine versionnée. Son absence est une ERREUR, jamais un skip.
func chargerBobine(t *testing.T, bo bobineObjectifs) []ChunkInput {
	t.Helper()
	dir := filepath.Join(miniBobineRacine, bo.Dossier)
	brut, err := os.ReadFile(filepath.Join(dir, "manifeste.json")) //nolint:gosec // constante de test
	if err != nil {
		t.Fatalf("bobine %s illisible : %v — elle est VERSIONNEE, son absence est une erreur, "+
			"pas une raison d'ignorer le test", dir, err)
	}
	var mf testManifest
	if err := json.Unmarshal(brut, &mf); err != nil {
		t.Fatalf("manifeste de %s illisible : %v", dir, err)
	}
	if len(mf.Chunks) != miniBobineChunks {
		t.Fatalf("bobine %s incomplete : %d chunk(s) au manifeste, %d attendus",
			dir, len(mf.Chunks), miniBobineChunks)
	}
	chunks := make([]ChunkInput, 0, len(mf.Chunks))
	for _, c := range mf.Chunks {
		d, err := os.ReadFile(filepath.Join(dir, nomChunk(c.Index))) //nolint:gosec // index du manifeste
		if err != nil {
			t.Fatalf("chunk %d de %s illisible : %v", c.Index, dir, err)
		}
		chunks = append(chunks, ChunkInput{
			Data: decompresser(d), StartMS: c.StartMS, ChunkType: c.ChunkType,
		})
	}
	return chunks
}

// TestMiniBobineObjectifsRegenerer : la RECETTE DE FABRICATION, exécutable.
//
// Elle est un test et non un `cmd/` parce qu'elle vit avec ce qu'elle produit : une recette
// rangée ailleurs se périme sans que rien ne le signale. Elle exige la fixture ET `-update`
// — deux verrous, parce qu'elle écrase des octets versionnés.
func TestMiniBobineObjectifsRegenerer(t *testing.T) {
	if !*majGolden {
		t.Skip("regeneration de la bobine : exige -update (elle ecrase des octets versionnes) " +
			"et FILM_CACHE_ROOT")
	}
	racine := racineCacheFilm(t)
	for _, bo := range corpusMiniBobines {
		regenererBobine(t, racine, bo)
	}
}

// regenererBobine écrit une bobine : les N premiers chunks TYPE-2 ancrés du film, tronqués.
func regenererBobine(t *testing.T, racine string, bo bobineObjectifs) {
	t.Helper()
	dir := filepath.Join(miniBobineRacine, bo.Dossier)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatalf("creation de %s : %v", dir, err)
	}
	var mf testManifest
	total := 0
	for _, c := range derniersChunksAncres(t, chargerChunksFilm(t, racine, bo.Film)) {
		court := comprimerFidele(t, tronquerApresType2(t, c))
		idx := len(mf.Chunks)
		if err := os.WriteFile(filepath.Join(dir, nomChunk(idx)), court, 0o600); err != nil {
			t.Fatalf("ecriture du chunk %d : %v", idx, err)
		}
		mf.Chunks = append(mf.Chunks, struct {
			Index     int `json:"index"`
			ChunkType int `json:"chunk_type"`
			StartMS   int `json:"start_ms"`
		}{Index: idx, ChunkType: c.ChunkType, StartMS: c.StartMS})
		total += len(court)
	}
	if len(mf.Chunks) != miniBobineChunks {
		t.Fatalf("%s : seulement %d chunk(s) ancre(s) trouve(s), %d demandes",
			bo.Film, len(mf.Chunks), miniBobineChunks)
	}
	ecrireManifeste(t, dir, mf)
	ecrireProvenanceBobine(t, dir, bo, total)
	t.Logf("bobine %s regeneree : %d chunks, %d octets", bo.Dossier, len(mf.Chunks), total)
}

// derniersChunksAncres : les `miniBobineChunks` derniers chunks du film qui portent l'ancre,
// dans l'ordre. Voir la constante pour la raison du « derniers » plutôt que « premiers ».
func derniersChunksAncres(t *testing.T, chunks []ChunkInput) []ChunkInput {
	t.Helper()
	var ancres []ChunkInput
	for _, c := range chunks {
		if p, tb := anchoredPayload(c); p != nil && tb >= 0 {
			ancres = append(ancres, c)
		}
	}
	if len(ancres) < miniBobineChunks {
		t.Fatalf("film source : %d chunk(s) ancre(s) seulement, %d demandes", len(ancres), miniBobineChunks)
	}
	return ancres[len(ancres)-miniBobineChunks:]
}

// tronquerApresType2 rend le PRÉFIXE du chunk qui s'arrête à la fin de son premier paquet
// TYPE_2, et REFUSE de le rendre si le préfixe ne décode pas exactement comme le chunk
// entier. C'est cette vérification qui autorise la troncature : sans elle, la bobine serait
// une fixture retaillée sur mesure, c'est-à-dire le défaut que ce lot corrige.
func tronquerApresType2(t *testing.T, c ChunkInput) []byte {
	t.Helper()
	fin := finPremierType2(c.Data)
	if fin <= 0 {
		t.Fatalf("aucun paquet TYPE_2 complet dans un chunk annonce ancre")
	}
	court := ChunkInput{Data: c.Data[:fin], StartMS: c.StartMS, ChunkType: c.ChunkType}
	pEntier, tbEntier := anchoredPayload(c)
	pCourt, tbCourt := anchoredPayload(court)
	if pCourt == nil || tbCourt != tbEntier {
		t.Fatalf("troncature faussante : token a %d sur le chunk entier, %d sur le prefixe",
			tbEntier, tbCourt)
	}
	if varAtBit(pEntier, tbEntier+shOffTeam0) != varAtBit(pCourt, tbCourt+shOffTeam0) ||
		varAtBit(pEntier, tbEntier+shOffTeam1) != varAtBit(pCourt, tbCourt+shOffTeam1) {
		t.Fatalf("troncature faussante : les varints lus different entre le chunk entier et son prefixe")
	}
	return c.Data[:fin]
}

// comprimerFidele rend le préfixe en zlib, APRÈS avoir vérifié que le décomprimer rend
// octet pour octet ce qu'on lui a donné. Sans cette relecture, « compression sans perte »
// serait une croyance sur une bibliothèque plutôt qu'une propriété de la bobine versionnée.
func comprimerFidele(t *testing.T, brut []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zlib.NewWriter(&buf)
	if _, err := w.Write(brut); err != nil {
		t.Fatalf("compression du chunk : %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("fermeture du compresseur : %v", err)
	}
	comprime := buf.Bytes()
	if relu := decompresser(comprime); !bytes.Equal(relu, brut) {
		t.Fatalf("compression faussante : %d octet(s) relus pour %d ecrits", len(relu), len(brut))
	}
	return comprime
}

// finPremierType2 : l'offset de fin du premier paquet TYPE_2 du chunk, ou -1. Même parcours
// de conteneur que `type2Payload` — l'en-tête fait 16 octets, la taille est en LE@4.
func finPremierType2(d []byte) int {
	off := 0
	for off+packetHdr <= len(d) {
		typ := int(binary.LittleEndian.Uint16(d[off:]))
		size := int(binary.LittleEndian.Uint32(d[off+4:]))
		if size < 0 || off+packetHdr+size > len(d) {
			return -1
		}
		if typ == packetType2 {
			return off + packetHdr + size
		}
		off += packetHdr + size
		if typ == packetEnd {
			return -1
		}
	}
	return -1
}

// ecrireManifeste : les métadonnées dont le décodeur a besoin et que les octets ne portent
// pas (instant de départ, type du chunk). Recopiées du manifest du cache, pas inventées.
func ecrireManifeste(t *testing.T, dir string, mf testManifest) {
	t.Helper()
	brut, err := json.MarshalIndent(mf, "", "  ")
	if err != nil {
		t.Fatalf("serialisation du manifeste : %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "manifeste.json"), append(brut, '\n'), 0o600); err != nil {
		t.Fatalf("ecriture du manifeste : %v", err)
	}
}

// ecrireProvenanceBobine : la provenance de chaque octet, versionnée AVEC la bobine.
func ecrireProvenanceBobine(t *testing.T, dir string, bo bobineObjectifs, total int) {
	t.Helper()
	var b strings.Builder
	fmt.Fprintf(&b, "MINI-BOBINE objectivescore — provenance\n\n")
	fmt.Fprintf(&b, "Film source      : %s (data/cache/film_chunks/%s)\n", bo.Film, bo.Film)
	fmt.Fprintf(&b, "Mode             : %s\n", bo.Variant)
	fmt.Fprintf(&b, "Chunks conserves : les %d premiers chunks TYPE-2 ANCRES du film\n", miniBobineChunks)
	fmt.Fprintf(&b, "Forme            : prefixe d octets BRUTS, tronque a la fin du premier\n"+
		"                   paquet TYPE_2 complet, range en zlib (forme employee par le\n"+
		"                   cache film lui-meme). Aucun octet n est modifie.\n")
	fmt.Fprintf(&b, "Poids            : %d octets comprimes\n", total)
	fmt.Fprintf(&b, "\nPOURQUOI UNE TRONCATURE EST LICITE ICI. Le decodeur est SANS ETAT d un "+
		"chunk a\nl autre : il ne lit que le premier paquet TYPE_2 du chunk qu on lui donne. La "+
		"recette\nverifie chunk par chunk que le prefixe decode EXACTEMENT comme le chunk entier "+
		"(meme\nposition de token, memes varints) et refuse d ecrire sinon. Elle verifie de meme "+
		"que\nla decompression rend octet pour octet le prefixe d origine.\n")
	fmt.Fprintf(&b, "\nRegeneration : voir minibobine_test.go.\n")
	if err := os.WriteFile(filepath.Join(dir, "PROVENANCE.txt"), []byte(b.String()), 0o600); err != nil {
		t.Fatalf("ecriture de la provenance : %v", err)
	}
}
