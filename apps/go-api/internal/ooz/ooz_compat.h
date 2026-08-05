// ooz_compat.h — shim de compatibilité pour compiler kraken.cpp (ooz, Powzix,
// écrit pour MSVC/Windows.h) sous g++ via cgo. Remplace stdafx.h.
//
// PORTABLE À DESSEIN (2026-07-31) : le shim n'incluait que <intrin.h>, en-tête
// MSVC/mingw-w64 qui n'existe pas sous gcc Linux. Le paquet compilait donc sur le
// poste de développement et sur le runner Windows, mais faisait échouer TOUTE la
// suite Go du runner Linux — `go test ./...` s'arrête sur « fatal error: intrin.h:
// No such file or directory », et les 8 786 tests de la baseline sont alors déclarés
// absents. Le décodeur des .module (internal/himodule, géométrie des cartes) n'a
// aucune raison d'être Windows-only : ce qui manquait était trois noms d'intrinsèques.
#pragma once

#include <stdint.h>
#include <stdlib.h>
#include <string.h>
#include <stdio.h>
#include <assert.h>

#if defined(_WIN32)
#include <intrin.h> // MSVC / mingw-w64 : _byteswap_*, _rotl
#else
// gcc/clang ailleurs : x86intrin.h porte les _mm_* ET _rotl ; les _byteswap_* et les
// _BitScan* sont des noms MSVC sans équivalent d'en-tête, rendus par les builtins GNU
// — mêmes instructions, aucune conversion de valeur.
#include <x86intrin.h>
#define _byteswap_ushort(x) __builtin_bswap16(x)
#define _byteswap_ulong(x) __builtin_bswap32(x)
#define _byteswap_uint64(x) __builtin_bswap64(x)

// Contrat MSVC reproduit à l'identique : 0 rendu et *index INTACT quand le masque est
// nul (les builtins GNU, eux, sont indéfinis en 0), sinon 1 et *index = rang du bit.
// Les builtins « l » suivent la largeur d'unsigned long, qui n'est pas la même ici
// (64 bits) et sous Windows (32) : passer par une largeur fixe tronquerait.
static inline unsigned char _BitScanReverse(unsigned long *index, unsigned long mask) {
  if (mask == 0) return 0;
  *index = (unsigned long)(sizeof(unsigned long) * 8 - 1) - (unsigned long)__builtin_clzl(mask);
  return 1;
}

static inline unsigned char _BitScanForward(unsigned long *index, unsigned long mask) {
  if (mask == 0) return 0;
  *index = (unsigned long)__builtin_ctzl(mask);
  return 1;
}
#endif

#include <emmintrin.h> // SSE2 _mm_*

#ifndef __forceinline
#define __forceinline inline
#endif

typedef unsigned char byte;
typedef unsigned char uint8;
typedef unsigned int uint32;
typedef unsigned long long uint64;
typedef long long int64;
typedef int int32;
typedef unsigned short uint16;
typedef short int16;
typedef unsigned int uint;
