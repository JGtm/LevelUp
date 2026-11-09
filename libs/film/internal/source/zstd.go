package source

// zstd.go — LE STOCKAGE ZSTD DES CHUNKS DU CACHE (decision D-18, 2026-11-01).
//
// UN CHUNK DU CACHE A TROIS FORMES POSSIBLES SUR DISQUE, et [Inflate] / [Load] les lisent toutes :
//
//	brut   la forme ecrite jusqu'ici par le writer du cache (le CDN sert du zlib, `filmfetch` le
//	       decompresse avant d'ecrire) ;
//	zlib   les chunks historiques du cache herite (premier octet 0x78) ;
//	zstd   la forme ecrite par le writer depuis D-18, niveau 3 ([NiveauZstd]) : en-tete magique
//	       `28 b5 2f fd`, taille du contenu inscrite dans l'en-tete de trame.
//
// LE FORMAT SE RECONNAIT A L'EN-TETE, jamais au nom du fichier ni a une colonne du manifeste : un
// cache a moitie recompresse (la recompression est une commande de l'utilisateur, jamais
// automatique) se lit donc sans table de correspondance. Aucun chunk brut ne peut commencer par la
// magie zstd : le premier champ d'un paquet est un u16 de type dont la valeur 0xB528 n'existe pas.
//
// UNE SEULE PORTE, comme la decompression zlib : ce fichier est le seul du depot a importer
// `klauspost/compress/zstd` (ratchet `archlint/no_raw_film_bytes_outside_source_test.go`, motif
// `decompression`). Le writer et le recompresseur du cache passent par [CompresserZstd] ; tout
// lecteur par [Inflate] ou [DecompresserZstd].
//
// LA TAILLE LOGIQUE D'UN CHUNK est celle de ses octets DECOMPRESSES. Le manifeste inscrit
// `size_bytes` = taille de ce que le writer a recu, et le controle de troncature du cache compare
// cette taille a la taille logique du fichier ([TailleLogiqueZstd] pour un fichier zstd, la taille
// du fichier sinon). Ainsi un manifeste ecrit avant D-18 reste valide apres la recompression d'un
// chunk brut, sans que personne ne le reecrive.

import (
	"bytes"
	"errors"
	"fmt"
	"sync"

	"github.com/klauspost/compress/zstd"
)

// NiveauZstd : le niveau de compression des chunks du cache, zstd-3 (« defaut » de la
// bibliotheque, `SpeedDefault`). Mesure du 2026-11-01 sur le cache de l'auteur : -75 % pour 3 s
// de compression sur 20 films.
const NiveauZstd = zstd.SpeedDefault

// tailleMaxEnteteZstd : taille maximale de la magie et de l'en-tete de trame zstd (4 + 14 octets).
// Lire autant d'octets au debut d'un fichier suffit pour [TailleLogiqueZstd].
const tailleMaxEnteteZstd = 18

// decompressionZstdMax : borne memoire d'une decompression. Le plus gros chunk connu pese
// 89 Mio comprime ; 1 Gio laisse une marge sans laisser un fichier corrompu reserver la machine.
const decompressionZstdMax = 1 << 30

// TailleEnteteZstd : le nombre d'octets a lire au debut d'un fichier pour y retrouver la taille
// logique d'un chunk zstd.
const TailleEnteteZstd = tailleMaxEnteteZstd

var magieZstd = [4]byte{0x28, 0xb5, 0x2f, 0xfd}

// ErrZstdTailleInconnue : l'en-tete de trame ne declare pas la taille du contenu (trame ecrite par
// un autre compresseur que le notre).
var ErrZstdTailleInconnue = errors.New("source: la trame zstd ne declare pas sa taille")

// EstZstd dit si `octets` COMMENCE par la magie d'une trame zstd.
func EstZstd(octets []byte) bool {
	return len(octets) >= len(magieZstd) && bytes.Equal(octets[:len(magieZstd)], magieZstd[:])
}

var (
	encodeurZstd = sync.OnceValues(func() (*zstd.Encoder, error) {
		// Concurrence bornee : un encodeur par appel simultane coute ~50 Mio sur un gros chunk, et
		// le writer ecrit un film (quelques dizaines de chunks) a la fois.
		return zstd.NewWriter(nil, zstd.WithEncoderLevel(NiveauZstd), zstd.WithEncoderConcurrency(2))
	})
	decodeurZstd = sync.OnceValues(func() (*zstd.Decoder, error) {
		return zstd.NewReader(nil, zstd.WithDecoderConcurrency(2), zstd.WithDecoderMaxMemory(decompressionZstdMax))
	})
)

// CompresserZstd compresse `octets` en une trame zstd-3 qui declare sa taille. Une entree vide
// est rendue telle quelle (il n'y a rien a comprimer, et une trame vide serait plus longue que
// l'entree).
func CompresserZstd(octets []byte) ([]byte, error) {
	if len(octets) == 0 {
		return octets, nil
	}
	enc, err := encodeurZstd()
	if err != nil {
		return nil, fmt.Errorf("source: encodeur zstd: %w", err)
	}
	return enc.EncodeAll(octets, make([]byte, 0, len(octets)/3+64)), nil
}

// DecompresserZstd decompresse une trame zstd. Un fichier qui n'en est pas une, ou une trame
// abimee, rend une erreur : l'appelant qui accepte le partiel ([Inflate]) le dit lui-meme.
func DecompresserZstd(trame []byte) ([]byte, error) {
	dec, err := decodeurZstd()
	if err != nil {
		return nil, fmt.Errorf("source: decodeur zstd: %w", err)
	}
	hint := 0
	if taille, err := TailleLogiqueZstd(trame); err == nil && taille <= decompressionZstdMax {
		hint = int(taille)
	}
	out, err := dec.DecodeAll(trame, make([]byte, 0, hint))
	if err != nil {
		return out, fmt.Errorf("source: decompression zstd: %w", err)
	}
	return out, nil
}

// TailleLogiqueZstd rend la taille des octets DECOMPRESSES d'une trame, lue dans son seul en-tete
// (les [TailleEnteteZstd] premiers octets suffisent). [ErrZstdTailleInconnue] quand l'en-tete ne
// la declare pas ; toute autre erreur dit que `debut` n'est pas un en-tete de trame valide.
func TailleLogiqueZstd(debut []byte) (int64, error) {
	var h zstd.Header
	if err := h.Decode(debut); err != nil {
		return 0, fmt.Errorf("source: en-tete zstd: %w", err)
	}
	if !h.HasFCS {
		return 0, ErrZstdTailleInconnue
	}
	return int64(h.FrameContentSize), nil
}
