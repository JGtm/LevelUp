// Package grammar is the GRAMMAR layer of the film decoder (ADR 0034 D-1:
// source -> profile -> grammar -> facts -> replay). It decodes the Halo Infinite Theater
// film replication stream (the ECS component wire format), reverse-engineered statically
// from HaloInfinite.exe via Ghidra: the [Lecteur] (the bit reader of the `source` layer,
// [source.Bits], decorated with a scan profile, a capture state and an observer), the
// engine's value codecs ([Lecteur.ReadSignedVarWidth]) and the per-component record parsers.
// Outside `film/` it is reached through the `film/decfilm` facade only (the compiler enforces
// it: the package lives under `film/internal/`).
//
// IT HOLDS NO STATBORG (SCORE) RECORD PARSER, and the removal is a measured decision, not
// housekeeping (D1 of PLAN_EXPLOITATION_REGISTRE_FILM, 2026-08-18). The parser mirrored
// FUN_140c18794 faithfully, but it was reached through the component chain at an offset the
// measurement disproved: 841 readings out of 841 wrong. The score of the match is decoded by
// ANCHORING instead, in `film/internal/facts/objectives`, whose grammar was calibrated against
// a Cheat Engine capture. The ECS deserializer of the statborg archetype
// (consumeStatborgValueStat, components_world.go) stays — it consumes bits so the chain stays
// aligned, which is a different job.
//
// The engine's accumulator/refill state machine (FUN_1406d6c7c / FUN_1406cf008 /
// FUN_140c18a1c) is an optimization; its observable output is a plain MSB-first big-endian
// bitstream, which is what the reader implements. See .ai/V7.5/film_re/PLAN_FILM_ECS_DECODER.md
// and .ai/V7.5/dumps/m1_funcs.c.
package grammar
