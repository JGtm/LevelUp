package sync

// durable_progress.go — invariant "durable-avant-progrès".
//
// Contrat : ne jamais avancer un marqueur de progrès (watermark, offset) AVANT
// que l'écriture durable associée ait commité. Sinon un échec d'écriture laisse
// le progrès en avance → l'item est considéré "déjà vu" et n'est jamais ré-écrit
// (perte silencieuse). Cf. incident LUSR JGtm 2026-06-07 (fix canonicalGate) et
// l'article DuckDB+Kafka "commit offset only after durability".
//
// Sites cross-store réels : LUSR v2 shadow (ligne match_skill_rank player DB
// AVANT le watermark player_skill_state_v2 shared). Le helper centralise et teste
// ce pattern ; `held` (anti-gap de groupe ordonné) est optionnel (nil = pas de
// notion de groupe).

import "context"

// DurableStep décrit une unité "écris durable, puis avance le progrès".
//   - GroupKey : clé d'un groupe ORDONNÉ (chrono). Si un item du groupe échoue son
//     écriture durable, le groupe est "tenu" → les items plus récents ne doivent
//     pas avancer le progrès par-dessus (anti-gap). Vide/held nil = pas de groupe.
//   - WriteDurable : l'écriture durable (nil = rien à écrire → considéré réussi).
//   - AdvanceProgress : avance le marqueur de progrès (nil = rien à avancer).
type DurableStep struct {
	GroupKey        string
	WriteDurable    func(context.Context) error
	AdvanceProgress func(context.Context) error
}

// DurableOutcome est le résultat de CommitThenAdvance.
//   - Advanced : l'écriture durable a réussi ET le progrès a été avancé.
//   - Held     : le progrès N'A PAS été avancé et le groupe est tenu (skip ordonné
//     ou échec d'écriture durable). Les items plus récents du groupe doivent être
//     sautés ce cycle.
//   - Err      : erreur d'écriture durable (avec Held) ou d'avance (sans Held —
//     l'écriture durable a réussi, donc un retry idempotent ré-écrira sans perte).
type DurableOutcome struct {
	Advanced bool
	Held     bool
	Err      error
}

// CommitThenAdvance applique l'invariant durable-avant-progrès pour UN item.
//
// Règles :
//  1. groupe déjà tenu → DurableOutcome{Held:true} sans rien tenter (skip ordonné).
//  2. WriteDurable échoue → tient le groupe (held[GroupKey]=true) + {Held:true, Err}.
//     AdvanceProgress N'EST PAS appelé.
//  3. WriteDurable OK puis AdvanceProgress échoue → {Err} SANS Held (retry idempotent :
//     l'écriture durable étant faite, ré-écrire est sûr et l'avance retentera).
//  4. Tout OK → {Advanced:true}.
//
// held peut être nil (pas de notion de groupe ordonné) ; GroupKey est alors ignoré.
func CommitThenAdvance(ctx context.Context, held map[string]bool, step DurableStep) DurableOutcome {
	if held != nil && step.GroupKey != "" && held[step.GroupKey] {
		return DurableOutcome{Held: true}
	}
	if step.WriteDurable != nil {
		if err := step.WriteDurable(ctx); err != nil {
			if held != nil && step.GroupKey != "" {
				held[step.GroupKey] = true
			}
			return DurableOutcome{Held: true, Err: err}
		}
	}
	if step.AdvanceProgress != nil {
		if err := step.AdvanceProgress(ctx); err != nil {
			return DurableOutcome{Err: err}
		}
	}
	return DurableOutcome{Advanced: true}
}
