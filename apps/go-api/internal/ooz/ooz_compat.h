// ooz_compat.h — shim de compatibilité pour compiler kraken.cpp (ooz, Powzix,
// écrit pour MSVC/Windows.h) sous mingw-w64 g++ via cgo. Remplace stdafx.h.
#pragma once

#include <stdint.h>
#include <stdlib.h>
#include <string.h>
#include <stdio.h>
#include <assert.h>
#include <intrin.h>    // mingw-w64 : _BitScanReverse, _byteswap_*, _rotl, __popcnt
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
