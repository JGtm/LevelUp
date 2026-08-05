// wrap.cpp — pont extern "C" entre cgo et Kraken_Decompress (linkage C++,
// défini dans kraken.cpp). Décodeur Oodle clean-room (ooz, Powzix, GPLv3).
#include "ooz_compat.h"

// Déclaration (le corps est dans kraken.cpp).
int Kraken_Decompress(const byte *src, size_t src_len, byte *dst, size_t dst_len);

extern "C" int ooz_decompress(const unsigned char *src, long src_len,
                              unsigned char *dst, long dst_len) {
	return Kraken_Decompress((const byte *)src, (size_t)src_len,
	                         (byte *)dst, (size_t)dst_len);
}
