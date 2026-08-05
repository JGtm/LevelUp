// Package filmdec decodes the Halo Infinite Theater film replication stream
// (the ECS component wire format), reverse-engineered statically from
// HaloInfinite.exe via Ghidra. It provides the bit-exact BitReader plus value
// codecs (layers L1/L2) and per-component record parsers (layer L4).
//
// Milestone M1 (this file set): the BitReader, the signed variable-width codec,
// and the statborg (score) record parser. The engine's accumulator/refill state
// machine (FUN_1406d6c7c / FUN_1406cf008 / FUN_140c18a1c) is an optimization; its
// observable output is a plain MSB-first big-endian bitstream, which is what this
// package implements. See .ai/V7.5/film_re/PLAN_FILM_ECS_DECODER.md and .ai/V7.5/dumps/m1_funcs.c.
package filmdec
