/**
 * lib/sessions/sessionLabels — l'identite d'un LABEL DE SESSION, et sa reconciliation.
 *
 * ─── POURQUOI CE MODULE EXISTE, ET POURQUOI IL EST DANS `lib/` ────────────────
 *
 * Un label de session est fabrique cote Go (`buildSessionLabel`) et embarque son
 * COMPTE DE MATCHS : « 02/04/2026 19:00–23:45 (13) ». Le backend, lui, filtre par
 * EGALITE STRICTE sur ce label. Deux matchs de plus a la prochaine synchronisation, et
 * un label persiste (URL partagee, miroir localStorage) devient un ZOMBIE : il ne
 * correspond plus a aucune session, la case a cocher n'existe plus, et la lecture
 * revient vide sans que rien ne l'explique.
 *
 * La reconciliation vient de la page Escouade, qui a rencontre le probleme la premiere.
 * Elle a ete DEPLACEE ici le 2026-09-06 parce que l'onglet Tactique persiste lui aussi
 * des labels de session : deux consommateurs, une seule definition — la recopier aurait
 * donne deux notions d'identite d'une session (CLAUDE.md n 6).
 */
import type { SessionLabelEntry } from '@/lib/api/types'

/**
 * Retire le suffixe " (N)" (match-count figé au sync, cf. buildSessionLabel
 * côté Go) d'un label de session pour obtenir une clé d'identité stable.
 */
export function stripSessionCountSuffix(label: string): string {
  return label.replace(/\s*\(\d+\)\s*$/, '').trim()
}

/**
 * Réconcilie les labels de sessions pickés contre la liste de sessions courante.
 *
 * Les labels backend embarquent un suffixe " (N)" qui change au gré des syncs.
 * Un label persisté en localStorage avec un ancien compte devient un "zombie" :
 * compté par le rail mais sans case à cocher correspondante (donc indécochable)
 * et filtré à 0 match côté backend. On remappe chaque label pické vers sa forme
 * courante (matching sur la clé sans suffixe) et on droppe les zombies
 * introuvables + les doublons. L'ordre des labels valides est préservé.
 */
export function reconcileSquadSessionLabels(
  picked: string[],
  sessions: SessionLabelEntry[],
): string[] {
  if (picked.length === 0 || sessions.length === 0) return picked
  const currentByKey = new Map(sessions.map((s) => [stripSessionCountSuffix(s.label), s.label]))
  const reconciled: string[] = []
  for (const label of picked) {
    const current = currentByKey.get(stripSessionCountSuffix(label))
    if (current && !reconciled.includes(current)) reconciled.push(current)
  }
  return reconciled
}
