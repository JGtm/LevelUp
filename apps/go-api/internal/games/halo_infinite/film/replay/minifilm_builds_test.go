package replay

// minifilm_builds_test.go — UNE MINI-BOBINE PAR BUILD (lot 0.A.2).
//
// CE QUE CELLE-CI AJOUTE A `minifilm_000d5950`. La bobine historique (cf. minifilm_test.go) n a
// PAS de `chunk_00` : c est un choix, et `zero_disque_test.go` verrouille l erreur exacte que son
// absence produit. Elle ne peut donc rien dire du REGISTRE, de la TABLE DES JOUEURS ni du BUILD —
// c est-a-dire de tout ce que le chantier du decodeur doit rendre explicite. Les bobines de ce
// fichier portent `chunk_00` par construction, une par build present au cache.
//
// CE QU UNE BOBINE PAR BUILD CONTIENT, ET POURQUOI CES TROIS-LA :
//
//	chunk_00.bin  le chunk de REGISTRE du film (type 1), tel quel — identite, registre des
//	              composants, table des joueurs, build en clair. C est le seul chunk qui porte la
//	              grammaire du film ; sans lui une bobine ne peut pas etre lue PAR SON PROFIL.
//	chunk_01.bin  les paquets d IMAGE-CLE (type 2) des chunks de replication : l etat complet de
//	              toutes les entites toutes les vingt secondes. C est la matiere du lot 0.A.3
//	              (fermeture par archetype) et du lot 3.6 (port des composants manquants).
//	chunk_02.bin  le PIED du film (type 3), tel quel — le fil des morts.
//
// LE CONDITIONNEMENT EST DU ZLIB, ET C EST CE QUI TIENT LE BUDGET. Les chunks du cache sont
// stockes DECOMPRESSES (entetes `0900` / `0100`, jamais `78 9c`) : `chunk_00` pese a lui seul
// 1,85 a 1,88 Mio, soit DEJA plus que le plafond de 1 Mio par build (arbitrage V7). Le chemin de
// lecture du depot inflate de maniere transparente et laisse passer ce qui ne l est pas
// (`filmsource.Inflate`, « une entree deja decompressee traverse telle quelle ») : stocker les
// chunks en zlib rend donc EXACTEMENT les memes octets au decodeur, et ramene `chunk_00` a
// 0,39-0,48 Mio. Aucun octet n est modifie, seul le conditionnement change.
//
// CE QUE CES BOBINES NE SONT PAS : des films valides. Les paquets d image-cle sont concatenes
// hors de leur continuite, donc les POSITIONS de biped — qui s accumulent par deltas — y sont
// sans signification, exactement comme dans la bobine historique.
//
// REGENERATION (jamais d edition a la main) :
//
//	REPLAY_FILM_CACHE=<repo>/data/cache/film_chunks \
//	  go test ./internal/games/halo_infinite/film/replay/ -run MiniFilmBuildsRegenerate -update

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/filmdec"
)

// miniFilmBuildBudget : le plafond par bobine, arbitrage V7 du plan (1 Mio).
const miniFilmBuildBudget = 1 << 20

// miniFilmCacheEnv : la variable qui porte la racine du cache de films pour la regeneration.
const miniFilmCacheEnv = "REPLAY_FILM_CACHE"

// buildMiniFilm decrit une bobine de build : le film dont elle sort, et ce qui le caracterise.
// Le BUILD est la cle du chantier (lot H : « la cle de dechiffrage est le BUILD, pas la version
// majeure ») ; la version et la carte sont la pour que la provenance se lise sans requete.
type buildMiniFilm struct {
	Short8  string
	Version int
	Build   string
	Map     string
	Mode    string
}

// miniFilmBuilds : UN FILM PAR BUILD PRESENT AU CACHE, le plus leger de chacun (cf. l entete de
// `testdata/equivalence/CORPUS.txt`, colonne version / build).
//
// `000d5950` N Y FIGURE PAS : sa bobine historique est sans `chunk_00` et se garde telle quelle.
// Il reste une entree des tables de goldens, ou il porte le meme build que `fb1a1a72`.
func miniFilmBuilds() []buildMiniFilm {
	return []buildMiniFilm{
		{"a521164d", 33, "HI_1_4_1", "Fragmentation Heavies", "BTB Heavies:Total Control"},
		{"60ae07c4", 37, "HI_1_8_0", "Live Fire - Ranked", "Ranked:Oddball"},
		{"11de8353", 38, "HI_1_9_0", "Thunderhead", "BTB:Fiesta Slayer"},
		{"111fa685", 39, "HI_1_10_0", "Command", "BTB:Fiesta Slayer"},
		{"e5adf7b2", 40, "HI_1_11_0", "Fragmentation", "BTB:Fiesta Slayer"},
		{"bcb6d393", 40, "HI_1_12_0", "Cliffhanger", "CTF:Arena"},
		{"fb1a1a72", 41, "HI_1_13_0", "Banished Narrows", "CTF:Arena"},
	}
}

// Dir rend le repertoire de la bobine, relatif au paquet.
func (b buildMiniFilm) Dir() string { return "testdata/minifilm_" + b.Short8 }

// TestMiniFilmBuildsRegenerate : LA SEULE PORTE D ECRITURE DES BOBINES PAR BUILD.
//
// Deux conditions explicites, comme pour la bobine historique et pour le fixture d entrees :
// `-update` et la racine du cache. Une regeneration accidentelle transformerait n importe quelle
// regression en « nouvelle reference ».
func TestMiniFilmBuildsRegenerate(t *testing.T) {
	cache := os.Getenv(miniFilmCacheEnv)
	switch {
	case !*updateGolden:
		t.Skip("regeneration des bobines par build : passer -update (et " + miniFilmCacheEnv + ")")
	case cache == "":
		t.Skip("regeneration des bobines par build : " + miniFilmCacheEnv + " non defini")
	}
	for _, b := range miniFilmBuilds() {
		t.Run(b.Build, func(t *testing.T) {
			n, err := writeBuildMiniFilm(b, filepath.Join(cache, b.Short8))
			if err != nil {
				t.Fatalf("%s (%s) : %v", b.Short8, b.Build, err)
			}
			t.Logf("bobine %s ecrite : %d octets (%.0f %% du budget)",
				b.Dir(), n, 100*float64(n)/miniFilmBuildBudget)
		})
	}
}

// writeBuildMiniFilm ecrit les trois chunks et la provenance, et rend le poids total.
//
// LE BUDGET EST VERIFIE AVANT D ECRIRE : une bobine hors budget n est pas ecrite du tout, et
// l erreur porte la taille mesuree. Depasser en silence ferait grossir le depot sans que
// personne ne l ait decide.
func writeBuildMiniFilm(b buildMiniFilm, src string) (int, error) {
	registre, err := filmdec.ReadFilmChunk(src, 0)
	if err != nil {
		return 0, fmt.Errorf("chunk_00 (registre) : %w", err)
	}
	pied := filmdec.CountFilmChunks(src)
	if pied < 2 {
		return 0, fmt.Errorf("film a %d chunk(s) : ni replication ni pied", pied)
	}
	brutPied, err := filmdec.ReadFilmChunk(src, pied)
	if err != nil {
		return 0, fmt.Errorf("chunk %d (pied) : %w", pied, err)
	}
	// LES DEUX CHUNKS QUI NE SE COUPENT PAS D ABORD : le registre et le pied entrent ENTIERS (un
	// demi-registre ne se lit pas). Ce qui reste du budget est la RESERVE des images-cles, seule
	// matiere qui se coupe sans rendre la bobine illisible.
	z0, err := zlibBytes(registre)
	if err != nil {
		return 0, fmt.Errorf("zlib chunk_00 : %w", err)
	}
	z2, err := zlibBytes(brutPied)
	if err != nil {
		return 0, fmt.Errorf("zlib chunk_02 : %w", err)
	}
	reserve := miniFilmBuildBudget - len(z0) - len(z2)
	if reserve <= 0 {
		return len(z0) + len(z2), fmt.Errorf("registre (%d) et pied (%d) compresses saturent deja "+
			"le budget de %d octets : aucune image-cle ne tiendrait",
			len(z0), len(z2), miniFilmBuildBudget)
	}
	kf, z1, err := keyframePacketsOf(src, pied, reserve)
	if err != nil {
		return 0, err
	}
	if kf.count == 0 {
		return 0, fmt.Errorf("aucun paquet d image-cle sous la reserve de %d octets", reserve)
	}
	total := len(z0) + len(z1) + len(z2)
	if total > miniFilmBuildBudget {
		return total, fmt.Errorf("bobine a %d octets, budget %d (V7)", total, miniFilmBuildBudget)
	}
	if err := os.MkdirAll(b.Dir(), 0o750); err != nil {
		return total, err
	}
	for nom, z := range map[string][]byte{
		"chunk_00.bin": z0, "chunk_01.bin": z1, "chunk_02.bin": z2,
	} {
		if err := os.WriteFile(filepath.Join(b.Dir(), nom), z, 0o600); err != nil {
			return total, fmt.Errorf("ecriture %s : %w", nom, err)
		}
	}
	prov := b.provenance(kf, len(z0), len(z1), len(z2), len(registre), len(brutPied), pied)
	if err := os.WriteFile(filepath.Join(b.Dir(), miniFilmManifest), []byte(prov), 0o600); err != nil {
		return total, fmt.Errorf("ecriture de la provenance : %w", err)
	}
	return total, nil
}

// keyframeSelection porte les paquets d image-cle retenus et de quoi ecrire leur provenance.
type keyframeSelection struct {
	body     []byte
	count    int
	chunks   []int
	tousVus  int
	tronquee bool
}

// keyframePacketsOf concatene les paquets d image-cle des chunks de REPLICATION (1..pied-1),
// CHUNK PAR CHUNK, et s arrete quand le chunk suivant ferait deborder la reserve.
//
// POURQUOI COUPER PAR CHUNK ET NON PAR PAQUET. Une image-cle porte l etat COMPLET de toutes les
// entites : pour la fermeture par archetype (lot 0.A.3), dix images-cles disent la meme chose que
// cinquante — chaque archetype y est represente des la premiere. Couper au chunk garde donc des
// images ENTIERES et une frontiere lisible (« les images-cles des chunks 1 a K »), la ou couper
// au paquet produirait une derniere image tronquee, c est-a-dire un record que le decodeur lirait
// jusqu au bout d un tampon qui s arrete.
//
// Le critere de selection est celui du decodeur (`filmdec.PacketTypeKeyframe`), pas une
// heuristique : la meme porte que `miniPacketKind` applique a la bobine historique.
func keyframePacketsOf(src string, pied, reserve int) (keyframeSelection, []byte, error) {
	var sel keyframeSelection
	var retenu []byte
	for c := 1; c < pied; c++ {
		chunk, err := filmdec.ReadFilmChunk(src, c)
		if err != nil {
			continue
		}
		duChunk, n := keyframesDuChunk(chunk)
		sel.tousVus += n
		if n == 0 || sel.tronquee {
			continue
		}
		z, errZ := zlibBytes(append(append([]byte{}, retenu...), duChunk...))
		if errZ != nil {
			return sel, nil, fmt.Errorf("zlib chunk_01 : %w", errZ)
		}
		if len(z) > reserve {
			// Ce chunk ferait deborder : on s arrete AVANT de l ajouter, et on continue la
			// boucle pour compter les images-cles restantes (la provenance dit combien).
			sel.tronquee = true
			continue
		}
		retenu = append(retenu, duChunk...)
		sel.count += n
		sel.chunks = append(sel.chunks, c)
	}
	sel.body = retenu
	z, err := zlibBytes(retenu)
	if err != nil {
		return sel, nil, fmt.Errorf("zlib chunk_01 : %w", err)
	}
	return sel, z, nil
}

// keyframesDuChunk rend les octets des paquets d image-cle d un chunk, en-tete compris, et leur
// nombre. Un paquet dont l en-tete deborde du chunk est ignore : la bobine ne porte que des
// paquets ENTIERS.
func keyframesDuChunk(chunk []byte) ([]byte, int) {
	var out []byte
	n := 0
	for _, p := range filmdec.WalkPackets(chunk) {
		if p.Type != filmdec.PacketTypeKeyframe {
			continue
		}
		start := p.Start - packetHeaderSizeMini
		if start < 0 || p.Start+p.Size > len(chunk) {
			continue
		}
		out = append(out, chunk[start:p.Start+p.Size]...)
		n++
	}
	return out, n
}

// provenance rend le texte de `PROVENANCE.txt`. Il dit d ou vient chaque octet ET la commande qui
// le refabrique : une fixture binaire sans provenance ecrite est un fait sans source.
func (b buildMiniFilm) provenance(kf keyframeSelection, z0, z1, z2, brut0, brut2, pied int) string {
	coupe := "toutes les images-cles du film"
	if kf.tronquee {
		coupe = fmt.Sprintf("COUPEE au budget : %d image(s)-cle retenue(s) sur les %d du film",
			kf.count, kf.tousVus)
	}
	return fmt.Sprintf(`MINI-BOBINE PAR BUILD — film %s (%s, %s, version %d, build %s)

CE FICHIER DIT D OU VIENT CHAQUE OCTET. Elle se REGENERE, elle ne s edite pas :

    %s=<repo>/data/cache/film_chunks \
      go test ./internal/games/halo_infinite/film/replay/ -run MiniFilmBuildsRegenerate -update

chunk_00.bin  %d octets (zlib) — LE CHUNK DE REGISTRE du film (type 1), %d octets une fois
              decompresse, OCTET POUR OCTET. Il porte l identite, le registre des composants, la
              table des joueurs et le build en clair (%s). C est ce chunk qui manque a la bobine
              historique %s, et sans lequel aucune bobine ne peut etre lue par son PROFIL.
chunk_01.bin  %d octets (zlib) — %d paquet(s) d IMAGE-CLE (type 2) REELS, %d octets une fois
              decompresses, extraits des chunks de replication %v.
              %s.
              Une image-cle porte l etat COMPLET de toutes les entites : c est la matiere de la
              fermeture par archetype (lot 0.A.3) et du port des composants manquants (lot 3.6).
chunk_02.bin  %d octets (zlib) — LE PIED du film (chunk %d, type 3), %d octets une fois
              decompresse, OCTET POUR OCTET : le fil des morts.

LE CONDITIONNEMENT EST DU ZLIB ET NE CHANGE AUCUN OCTET. Les chunks du cache sont stockes
DECOMPRESSES ; l inflate du depot decompresse ce qui est compresse et laisse passer ce qui ne
l est pas, donc le decodeur recoit exactement les memes octets. Sans ce conditionnement, chunk_00
pesait a lui seul %.2f Mio — deja plus que le plafond de 1 Mio par build (arbitrage V7 du plan).

CE QUE CETTE BOBINE N EST PAS : un film valide. Les paquets d image-cle sont concatenes hors de
leur continuite, donc les POSITIONS de biped — qui s accumulent par deltas — y sont sans
signification, comme dans la bobine historique.
`, b.Short8, b.Map, b.Mode, b.Version, b.Build,
		miniFilmCacheEnv,
		z0, brut0, b.Build, goldenFilm,
		z1, kf.count, len(kf.body), kf.chunks, coupe,
		z2, pied, brut2,
		float64(brut0)/(1<<20))
}
