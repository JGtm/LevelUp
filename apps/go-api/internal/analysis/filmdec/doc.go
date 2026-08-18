// Package filmdec decodes the Halo Infinite Theater film replication stream
// (the ECS component wire format), reverse-engineered statically from
// HaloInfinite.exe via Ghidra. It provides the bit-exact BitReader plus value
// codecs (layers L1/L2) and per-component record parsers (layer L4).
//
// Milestone M1 (this file set): the BitReader and the signed variable-width codec.
//
// IT NO LONGER HOLDS A STATBORG (SCORE) RECORD PARSER, and the removal is a measured
// decision, not housekeeping (D1 of PLAN_EXPLOITATION_REGISTRE_FILM, 2026-08-18). The parser
// that lived in statborg.go mirrored FUN_140c18794 faithfully, but it was reached through the
// component chain at an offset the measurement disproved: 841 readings out of 841 wrong. The
// score of the match is decoded by ANCHORING instead, in analysis/objectiveevents, whose
// grammar was calibrated against a Cheat Engine capture. Two decoders of the same fact
// diverge; this one is gone rather than kept "just in case". The ECS deserializer of the
// statborg archetype (consumeStatborgValueStat, components_world.go) is untouched — it
// consumes bits so the chain stays aligned, which is a different job.
//
// The engine's accumulator/refill state
// machine (FUN_1406d6c7c / FUN_1406cf008 / FUN_140c18a1c) is an optimization; its
// observable output is a plain MSB-first big-endian bitstream, which is what this
// package implements. See .ai/V7.5/film_re/PLAN_FILM_ECS_DECODER.md and .ai/V7.5/dumps/m1_funcs.c.
package filmdec
