package tokencipher

// dpapi_windows.go : le coffre Windows (DPAPI), portée utilisateur courant. Un scellé n'est lisible
// que par le même compte Windows sur la même machine : le fichier de jetons copié sur un autre poste
// ou lu depuis un autre compte est illisible, c'est le but (plan D-8).

import (
	"errors"
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

// dpapiEntropy : entropie facultative ajoutée au scellé. Elle lie le scellé à LevelUp : un autre
// programme du même compte qui appelle CryptUnprotectData sans elle n'ouvre pas le fichier. Constante
// de format : la changer rend illisibles tous les fichiers déjà scellés.
var dpapiEntropy = []byte("levelup.watcher_tokens.v1")

// dpapiDescription : description attachée au scellé (visible de DPAPI, jamais secrète).
const dpapiDescription = "LevelUp watcher token"

type dpapiCipher struct{}

func newDPAPI() (Cipher, error) { return dpapiCipher{}, nil }

func (dpapiCipher) Name() string { return NameDPAPI }

// blobOf expose b comme DataBlob sans le copier. Un tampon vide donne un blob vide.
func blobOf(b []byte) windows.DataBlob {
	if len(b) == 0 {
		return windows.DataBlob{}
	}
	return windows.DataBlob{Size: uint32(len(b)), Data: &b[0]}
}

// takeBlob copie le contenu d'un blob alloué par le système puis le libère.
func takeBlob(b *windows.DataBlob) []byte {
	if b.Data == nil {
		return []byte{}
	}
	out := append([]byte(nil), unsafe.Slice(b.Data, b.Size)...)
	_, _ = windows.LocalFree(windows.Handle(unsafe.Pointer(b.Data)))
	return out
}

func (dpapiCipher) Seal(plain []byte) ([]byte, error) {
	desc, err := windows.UTF16PtrFromString(dpapiDescription)
	if err != nil {
		return nil, fmt.Errorf("tokencipher: description DPAPI: %w", err)
	}
	in, entropy := blobOf(plain), blobOf(dpapiEntropy)
	var out windows.DataBlob
	if err := windows.CryptProtectData(&in, desc, &entropy, 0, nil, windows.CRYPTPROTECT_UI_FORBIDDEN, &out); err != nil {
		return nil, fmt.Errorf("tokencipher: CryptProtectData: %w", err)
	}
	return takeBlob(&out), nil
}

func (dpapiCipher) Open(sealed []byte) ([]byte, error) {
	if len(sealed) == 0 {
		return nil, errors.New("tokencipher: scellé DPAPI vide")
	}
	in, entropy := blobOf(sealed), blobOf(dpapiEntropy)
	var out windows.DataBlob
	if err := windows.CryptUnprotectData(&in, nil, &entropy, 0, nil, windows.CRYPTPROTECT_UI_FORBIDDEN, &out); err != nil {
		return nil, fmt.Errorf("tokencipher: CryptUnprotectData: %w", err)
	}
	return takeBlob(&out), nil
}
