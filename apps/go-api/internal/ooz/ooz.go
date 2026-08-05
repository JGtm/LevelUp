// Package ooz expose le décodeur Oodle clean-room « ooz » (Powzix, GPLv3) pour
// décompresser les blocs Kraken/Mermaid/Selkie/Leviathan des .module Halo Infinite.
// Offline-pur, universel, aucune DLL propriétaire du jeu requise.
//
// Nécessite CGO_ENABLED=1 + un toolchain C++ (mingw-w64 g++).
package ooz

/*
#cgo CXXFLAGS: -O2 -msse4.1 -std=c++14 -w
#cgo LDFLAGS: -static-libstdc++ -static-libgcc
extern int ooz_decompress(const unsigned char* src, long src_len, unsigned char* dst, long dst_len);
*/
import "C"

import (
	"fmt"
	"unsafe"
)

// safeSpace : marge OBLIGATOIRE après le buffer de sortie. Les copieurs SSE d'ooz
// (Kraken/Mermaid) écrivent par blocs de 16/32 octets et débordent donc au-delà de la
// dernière position décodée. Sans cette marge, un bloc dont la taille décompressée
// tombe près d'une limite de page fait crasher le process (0xC0000005 reproductible
// sur ridgeline-rtx-new.module, entrée #1082, rawLen=0x100000, faute à dst+0xFFFEC).
const safeSpace = 128

// Decompress décompresse un bloc Oodle. rawLen = taille décompressée attendue
// (connue du conteneur module). Renvoie une erreur si la taille décodée diffère.
func Decompress(comp []byte, rawLen int) ([]byte, error) {
	if len(comp) == 0 || rawLen <= 0 {
		return nil, fmt.Errorf("ooz: entrée invalide (comp=%d rawLen=%d)", len(comp), rawLen)
	}
	dst := make([]byte, rawLen+safeSpace)
	n := C.ooz_decompress(
		(*C.uchar)(unsafe.Pointer(&comp[0])), C.long(len(comp)),
		(*C.uchar)(unsafe.Pointer(&dst[0])), C.long(rawLen),
	)
	if int(n) != rawLen {
		return nil, fmt.Errorf("ooz: décompression échouée (retour=%d, attendu=%d)", int(n), rawLen)
	}
	return dst[:rawLen:rawLen], nil
}
