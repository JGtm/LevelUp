package anonymisegamertags

// conteneur_test.go — OUVRIR ET REFERMER UN FICHIER DE FIXTURE : zlib, gzip ou brut.
//
// La regle de detection est celle du decodeur (`source.Inflate`) : un premier octet 0x78 suivi
// d un flux zlib qui se decompresse est un chunk compresse ; tout le reste traverse tel quel.

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"errors"
	"fmt"
	"io"
)

// forme : la facon dont un fichier porte ses octets clairs.
type forme string

const (
	formeBrute forme = "brut"
	formeZlib  forme = "zlib"
	formeGzip  forme = "gzip"
)

// contenu : les octets clairs d un fichier, et de quoi le refermer sous la meme forme.
type contenu struct {
	forme  forme
	clair  []byte
	niveau int         // zlib : niveau deduit de l en-tete
	entete gzip.Header // gzip : en-tete d origine (nom, date) conserve
}

// ouvrir rend le contenu clair d un fichier. Un flux zlib ou gzip qui casse en cours de route est
// REFUSE : on ne saurait pas le refermer a l identique.
func ouvrir(brut []byte) (contenu, error) {
	if len(brut) >= 2 && brut[0] == 0x1f && brut[1] == 0x8b {
		zr, err := gzip.NewReader(bytes.NewReader(brut))
		if err != nil {
			return contenu{}, fmt.Errorf("en-tete gzip : %w", err)
		}
		clair, err := io.ReadAll(zr)
		if err != nil {
			return contenu{}, fmt.Errorf("flux gzip tronque ou casse : %w", err)
		}
		return contenu{forme: formeGzip, clair: clair, entete: zr.Header}, nil
	}
	if len(brut) >= 2 && brut[0] == 0x78 {
		zr, err := zlib.NewReader(bytes.NewReader(brut))
		if err == nil {
			clair, errLecture := io.ReadAll(zr)
			if errLecture != nil {
				return contenu{}, fmt.Errorf("flux zlib tronque ou casse : %w", errLecture)
			}
			return contenu{forme: formeZlib, clair: clair, niveau: niveauZlib(brut[1])}, nil
		}
		if !errors.Is(err, zlib.ErrHeader) {
			return contenu{}, fmt.Errorf("en-tete zlib : %w", err)
		}
	}
	return contenu{forme: formeBrute, clair: append([]byte(nil), brut...)}, nil
}

// niveauZlib traduit le champ FLEVEL de l en-tete zlib (RFC 1950) en niveau de compression.
func niveauZlib(flg byte) int {
	switch flg >> 6 {
	case 0:
		return zlib.BestSpeed
	case 1:
		return 5
	case 3:
		return zlib.BestCompression
	default:
		return zlib.DefaultCompression
	}
}

// refermer rend les octets du fichier sous sa forme d origine.
func refermer(c contenu) ([]byte, error) {
	var buf bytes.Buffer
	switch c.forme {
	case formeBrute:
		return c.clair, nil
	case formeZlib:
		w, err := zlib.NewWriterLevel(&buf, c.niveau)
		if err != nil {
			return nil, fmt.Errorf("compresseur zlib : %w", err)
		}
		if _, err := w.Write(c.clair); err != nil {
			return nil, fmt.Errorf("compression zlib : %w", err)
		}
		if err := w.Close(); err != nil {
			return nil, fmt.Errorf("compression zlib : %w", err)
		}
	case formeGzip:
		w := gzip.NewWriter(&buf)
		w.Header = c.entete
		if _, err := w.Write(c.clair); err != nil {
			return nil, fmt.Errorf("compression gzip : %w", err)
		}
		if err := w.Close(); err != nil {
			return nil, fmt.Errorf("compression gzip : %w", err)
		}
	default:
		return nil, fmt.Errorf("forme inconnue %q", c.forme)
	}
	return buf.Bytes(), nil
}

// anonymiserOctets rend le fichier reecrit et les occurrences remplacees. Un fichier sans
// occurrence est rendu TEL QUEL (pas de recompression qui changerait ses octets pour rien), et un
// fichier reecrit est RELU pour verifier que son contenu clair est bien celui qu on a produit.
func anonymiserOctets(brut []byte, ps []paire) ([]byte, map[string]int, forme, error) {
	c, err := ouvrir(brut)
	if err != nil {
		return nil, nil, "", err
	}
	compte := remplacer(c.clair, ps)
	if len(compte) == 0 {
		return brut, compte, c.forme, nil
	}
	sortie, err := refermer(c)
	if err != nil {
		return nil, nil, "", err
	}
	relu, err := ouvrir(sortie)
	if err != nil {
		return nil, nil, "", fmt.Errorf("relecture : %w", err)
	}
	if relu.forme != c.forme || !bytes.Equal(relu.clair, c.clair) {
		return nil, nil, "", fmt.Errorf("relecture : le fichier reecrit ne rend pas le contenu produit")
	}
	return sortie, compte, c.forme, nil
}
