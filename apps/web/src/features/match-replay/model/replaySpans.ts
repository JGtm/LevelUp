/**
 * replaySpans — « CET INTERVALLE COUVRE-T-IL CETTE IMAGE ? », écrit une fois.
 *
 * POURQUOI CE MODULE EXISTE (registre 2026-09-05, K5). Le film date tout ce qui DURE par un
 * couple d'images `{t0, t1}` : un portage de bombe, de crâne ou de drapeau, une période de
 * VIP, une occupation de véhicule, une station de faille. Le prédicat « cet intervalle couvre
 * l'image courante » était écrit DIX fois, dans deux orthographes selon l'auteur :
 *
 *     frame >= s.t0 && frame <= s.t1        s.t0 <= frame && frame <= s.t1
 *
 * Les dix étaient d'accord — mesuré en vérification adverse (V-WEB-2) : même convention,
 * fermée aux deux bouts. C'est précisément ce qui rendait la dixième copie facile à écrire
 * sans y penser, et la onzième facile à écrire FAUX : `<` au lieu de `<=` sur `t1` fait
 * disparaître un glyphe à sa dernière image, ce qui ne se voit sur aucun écran.
 *
 * FERMÉ AUX DEUX BOUTS, ET C'EST UNE DÉCISION. `t1` est la DERNIÈRE image où l'état vaut,
 * pas la première où il ne vaut plus : un portage d'une seule image a `t0 === t1` et doit se
 * peindre. La convention est celle du film, pas un choix de commodité.
 */

/** Ce que ce prédicat demande d'un intervalle, et rien de plus. */
export interface FrameSpan {
  t0: number
  t1: number
}

/** covers — l'intervalle contient-il l'image ? Fermé aux deux bouts (cf. l'en-tête). */
export function covers(span: FrameSpan, frame: number): boolean {
  return frame >= span.t0 && frame <= span.t1
}

/** spansAt — les intervalles d'une liste qui couvrent l'image, dans l'ordre de la liste. */
export function spansAt<T extends FrameSpan>(spans: readonly T[], frame: number): T[] {
  return spans.filter((s) => covers(s, frame))
}
