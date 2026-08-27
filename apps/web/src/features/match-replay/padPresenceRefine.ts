/**
 * padPresenceRefine.ts — L'ÉTAT INCERTAIN D'UN SOCLE, RAMENÉ À CE QU'UN JOUEUR A PU FAIRE.
 *
 * LA DÉCISION EST DE L'UTILISATEUR (2e passe du 2026-08-27) : « j'ai vu un rejeu où un socle
 * était incertain alors que tous les joueurs étaient à l'autre bout de la map. Pas logique. »
 * Personne ne ramasse une arme sans passer dessus : tant qu'AUCUN joueur n'est passé près du
 * socle, il n'y a aucune raison d'afficher un doute — le socle reste PLEIN.
 *
 * CE QUE CE FICHIER FAIT, ET RIEN D'AUTRE : il REMONTE `tLow`, la borne à partir de laquelle
 * l'écran cesse d'affirmer la présence, jusqu'à la première APPROCHE d'un joueur dans la fenêtre
 * d'incertitude. `t0`, `tHigh`, `spawns` et `cycle` sont INTOUCHÉS — la preuve d'absence, le
 * basculement « vide » et le compte à rebours restent exactement ceux de la mesure.
 *
 * POURQUOI LA MESURE A-2 NE RÉFUTAIT PAS CETTE RÈGLE, alors qu'elle la contredisait à 31 %.
 * L'oracle était CIRCULAIRE. A-2 comparait la règle à la preuve d'absence de l'ARTEFACT, or
 * cette preuve est le RECENSEMENT des images-clés — et le recensement est lui-même vraisembla-
 * blement borné par la proximité des joueurs. Dans une fenêtre sans aucune approche, ce qui
 * disparaît est probablement l'objet DU RECENSEMENT, pas l'objet du terrain. La répartition le
 * dit : les deux films qui portent ~75 % des contradictions sont les deux BTB à 24-25 joueurs,
 * avec des distances minimales médianes de 18 à 25 m ; les arènes sont entre 0 et 10 %. Le
 * négatif d'A-2 reste vrai tel qu'il est écrit — l'artefact ne confirme pas la règle là-bas —
 * mais il ne la réfute pas.
 *
 * LE COÛT EST ASSUMÉ, et il faut l'écrire : une VRAIE disparition sans approche (mécanisme
 * inconnu — objet retiré par le moteur, arme ramassée par un joueur dont la trace manque) sera
 * désormais affichée PLEINE jusqu'à `tHigh`, là où « incertain » n'affirmait rien. On échange un
 * doute honnête mais illisible contre une affirmation utile et rarement fausse. C'est une
 * décision produit, pas un résultat de mesure.
 *
 * CONDITION DE REPRISE : l'enquête sur la population de socles des artefacts BTB (les deux films
 * suspects, socles jamais approchés de tout le film) est ouverte dans les Découvertes du plan
 * `.ai/V7.5/replay2d/PLAN_RETOURS_REJEU_2026-08-27.md`. Si elle montre que ces socles sont des
 * artefacts de mesure, ce fichier n'aura plus rien à corriger ; si elle montre l'inverse, c'est
 * le rayon ou la règle qui devront bouger.
 *
 * Pur : aucune dépendance React, aucun canvas. La mémoïsation est chez l'appelant.
 */
import type { ReplayTrackReady, ReplayWeaponPadReady } from './replayNormalize'

/**
 * LE RAYON D'APPROCHE, en mètres monde, et d'où il vient.
 *
 * MESURÉ (instrument A-2, `pads_proximity_research_test.go`), et TOUS LES CHIFFRES QUI SUIVENT
 * VIENNENT DU MÊME DÉNOMINATEUR — les 1 064 occupations achevées que l'artefact publie
 * (`padPickups`, 30 artefacts). Les mélanger avec ceux de la base de contrôle reconstruite
 * depuis `Presence` (992 occupations, plus étroite) est l'erreur que la revue de la mesure a
 * déjà fait corriger une fois : à 2 m, **69,1 %** des occupations achevées portent au moins une
 * approche (100 − 30,92), et le passage le plus proche a une distance médiane de **0,36 m**.
 *
 * LA DISTRIBUTION EST BIMODALE, et c'est elle qui rend le choix PEU SENSIBLE : soit un joueur
 * passe à moins d'un mètre, soit il n'y a personne à des dizaines de mètres. Ouvrir le rayon de
 * 1 m à 10 m ne gagne qu'une DIZAINE de points (contradiction 33,18 % → 23,21 %) : il n'existe
 * pas de rayon « bien réglé » à trouver, tout l'intervalle raisonnable donne le même résultat.
 * 2 m est assez large pour couvrir un ramassage en courant (un joueur parcourt ~0,6 m entre
 * deux images) sans jamais atteindre le second mode.
 */
export const PAD_APPROACH_RADIUS_M = 2

/**
 * LA LAME « PERSONNE N'EST PASSÉ » : de combien la borne se pose AVANT `tHigh`, en images.
 *
 * POURQUOI PAS `tLow = tHigh` TOUT ROND, qui serait l'écriture évidente : `padStateAt` lit déjà
 * `tHigh <= tLow` comme le SENTINEL « aucune absence n'a jamais été prouvée », et rend alors
 * PLEIN pour toujours — y compris APRÈS `tHigh`. Le socle ne basculerait plus jamais à vide et
 * son compte à rebours disparaîtrait, alors que la preuve d'absence, elle, n'a pas bougé. Il
 * faut donc que `tLow` reste STRICTEMENT sous `tHigh`, d'où cette lame.
 *
 * ELLE EST VISIBLE, ET IL FAUT L'ÉCRIRE (revue adversariale du 2026-08-27). On a d'abord cru
 * qu'une lame d'une demi-image ne coûtait rien parce que « les images sont entières » : c'est
 * FAUX au tracé. `useReplayPlayback` avance l'image de `dt × fps` SANS arrondi, et le calque
 * comme le survol lisent cette valeur FRACTIONNAIRE. L'intervalle `[tHigh - lame ; tHigh)` est
 * donc atteignable : à une demi-image il durait 50 ms de film (100 ms de montre à 0,5x), soit
 * environ trois images peintes de halo « incertain » juste avant chaque bascule — et l'état s'y
 * FIGEAIT si le lecteur mettait pause dedans.
 *
 * `1/64` D'IMAGE RAMÈNE LA LAME À ~1,6 ms de film (une image de 100 ms) : au pire UNE image
 * peinte isolée de temps en temps, et une pause qui tomberait pile dedans devient de l'ordre de
 * 0,008 % de l'axe. La lame ne DISPARAÎT pas — elle ne peut pas, `tHigh > tLow` est la
 * condition qui garde le sentinel libre — elle devient invisible en pratique. Puissance de deux
 * choisie exprès : exacte en binaire, donc aucune dérive d'arrondi en comparant à une image
 * fractionnaire.
 *
 * EXPORTÉE parce qu'elle est une CLAUSE de ce module, pas un détail : un test qui la recopierait
 * en littéral ne verrait pas son changement.
 */
export const PAD_BETWEEN_FRAMES = 1 / 64

/** Le socle et la fenêtre d'incertitude d'une de ses occupations : ce qu'on interroge. */
interface PadWindow {
  cx: number
  cy: number
  lo: number
  hi: number
}

/**
 * refinePadPresence — les mêmes socles, avec leur incertitude ramenée aux instants où quelqu'un
 * a PU prendre l'arme.
 *
 * RÉFÉRENCE PRÉSERVÉE QUAND RIEN NE CHANGE : un socle dont aucune occupation ne bouge est rendu
 * TEL QUEL, et le tableau lui-même n'est recréé que si au moins un socle a changé. Les appelants
 * mémoïsent sur l'identité du tableau ; en recréer un à chaque appel ferait repeindre la scène
 * pour rien.
 */
export function refinePadPresence(
  pads: readonly ReplayWeaponPadReady[],
  tracks: readonly ReplayTrackReady[],
): readonly ReplayWeaponPadReady[] {
  if (pads.length === 0 || tracks.length === 0) return pads
  let touche = false
  const out = pads.map((pad) => {
    const presence = refineOccupancies(pad, tracks)
    if (presence === pad.presence) return pad
    touche = true
    return { ...pad, presence }
  })
  return touche ? out : pads
}

/**
 * refineOccupancies rend les occupations d'UN socle, ou le tableau d'origine si aucune ne bouge.
 *
 * UNE FENÊTRE JAMAIS FERMÉE RESTE INTOUCHÉE (`tHigh <= tLow`) : il n'y a rien à raffiner là où
 * aucune absence n'a été prouvée — le socle est déjà plein jusqu'au bout, et y toucher
 * réécrirait le sentinel que `padStateAt` lit.
 */
function refineOccupancies(
  pad: ReplayWeaponPadReady,
  tracks: readonly ReplayTrackReady[],
): ReplayWeaponPadReady['presence'] {
  let touche = false
  const out = pad.presence.map((occ) => {
    if (occ.tHigh <= occ.tLow) return occ
    const win = { cx: pad.x, cy: pad.y, lo: occ.tLow, hi: occ.tHigh }
    const tLow = refinedLow(win, tracks)
    if (tLow === occ.tLow) return occ
    touche = true
    return { ...occ, tLow }
  })
  return touche ? out : pad.presence
}

/**
 * refinedLow — la nouvelle borne de présence prouvée pour une fenêtre ouverte.
 *
 * AUCUNE APPROCHE : la borne se pose juste avant `tHigh` (cf. PAD_BETWEEN_FRAMES) — le socle
 * reste plein jusqu'à la preuve d'absence, qui garde tous ses droits.
 *
 * UNE APPROCHE À `t` : la borne devient la DERNIÈRE IMAGE AVANT `t`. Elle ne DESCEND jamais —
 * raffiner ne peut qu'ajouter de la certitude, jamais en retirer.
 */
function refinedLow(win: PadWindow, tracks: readonly ReplayTrackReady[]): number {
  const t = firstApproach(win, tracks)
  if (t === null) return win.hi - PAD_BETWEEN_FRAMES
  // `Math.ceil(t) - 1` est le plus grand entier STRICTEMENT inférieur à `t` : à cette image, le
  // joueur n'est pas encore au socle. C'est une image de prudence (l'approche a lieu APRÈS
  // elle), et elle va dans le bon sens : on n'affirme jamais une présence à une image où le
  // ramassage a pu commencer.
  //
  // AUCUN PLAFOND À ÉCRIRE, et c'en est un de moins qui mentirait (revue adversariale du
  // 2026-08-27) : `t` vient d'un segment COUPÉ à la fenêtre, donc `t <= hi` ; les bornes étant
  // entières au contrat, `ceil(t) - 1 <= hi - 1`. Un `Math.min(..., win.hi)` était donc
  // inatteignable — et pire, il aurait silencieusement rendu `hi` (le sentinel « jamais vidé »)
  // le jour où le clip disparaîtrait, au lieu de laisser la régression se voir. Le cas de test
  // « un segment qui ne touche qu'APRÈS la fenêtre » est le filet qui garde le clip en place.
  return Math.max(win.lo, Math.ceil(t) - 1)
}

/**
 * firstApproach — l'instant (en images, interpolé) où un joueur entre le premier dans le rayon
 * du socle pendant la fenêtre, ou null si personne n'y entre.
 *
 * LES POINTS D'UNE TRACE SONT ÉCHANTILLONNÉS, et c'est tout le piège : tester la seule distance
 * AUX POINTS laisse passer le joueur qui traverse le socle entre deux échantillons. On teste
 * donc le SEGMENT entre deux points consécutifs — la trajectoire telle que le rejeu la dessine.
 *
 * LE TROU DE RÉPLICATION TOMBE SOUS LE MÊME ARGUMENT (revue adversariale du 2026-08-27). Quand
 * une vie survit à un long trou (jusqu'à ~5 s sans position répliquée), le segment trace une
 * CORDE DROITE que le joueur n'a pas forcément parcourue — et cette corde peut frôler un socle
 * sans passage réel. On ne corrige pas, et la raison est la cohérence de l'écran : le rejeu
 * DESSINE cette même corde, le marqueur du joueur la parcourt sous les yeux du lecteur. Dire
 * « incertain » là où l'écran montre quelqu'un qui passe est cohérent ; l'inverse serait un
 * doute invisible. Fréquence non mesurée — consignée en Découvertes.
 *
 * LA RECHERCHE EST BORNÉE PAR LA FENÊTRE : les points d'une trace sont triés, une dichotomie
 * donne le premier de la fenêtre, et on remonte d'UN point pour attraper le segment qui y ENTRE.
 * C'est ce qui rend la passe tenable — une fenêtre dure quelques secondes, une trace tout un
 * match.
 */
function firstApproach(win: PadWindow, tracks: readonly ReplayTrackReady[]): number | null {
  let best: number | null = null
  for (const track of tracks) {
    const pts = track.points
    if (pts.length === 0) continue
    if (pts.length === 1) {
      const only = pts[0]
      if (only.t >= win.lo && only.t <= win.hi && withinRadius(only.x - win.cx, only.y - win.cy)) {
        best = best === null ? only.t : Math.min(best, only.t)
      }
      continue
    }
    let i = lowerBound(pts, win.lo)
    if (i > 0) i -= 1 // le segment qui ENTRE dans la fenêtre part du point précédent
    for (; i + 1 < pts.length && pts[i].t <= win.hi; i++) {
      const t = segmentTouch(pts[i], pts[i + 1], win)
      if (t !== null && (best === null || t < best)) best = t
    }
  }
  return best
}

/** lowerBound — le rang du premier point dont l'instant atteint `t` (dichotomie). */
function lowerBound(pts: ReplayTrackReady['points'], t: number): number {
  let lo = 0
  let hi = pts.length
  while (lo < hi) {
    const mid = (lo + hi) >> 1
    if (pts[mid].t < t) lo = mid + 1
    else hi = mid
  }
  return lo
}

/** withinRadius — le point est-il dans le rayon d'approche (distance 2D, monde) ? */
function withinRadius(dx: number, dy: number): boolean {
  return dx * dx + dy * dy <= PAD_APPROACH_RADIUS_M * PAD_APPROACH_RADIUS_M
}

/**
 * segmentTouch — le premier instant où le segment [a, b], COUPÉ à la fenêtre, entre dans le
 * rayon du socle, ou null.
 *
 * LE SEGMENT EST COUPÉ AVANT D'ÊTRE TESTÉ : un joueur qui frôle le socle une seconde APRÈS la
 * preuve d'absence ne dit rien de la disparition, et le compter avancerait la borne à tort.
 */
function segmentTouch(
  a: ReplayTrackReady['points'][number],
  b: ReplayTrackReady['points'][number],
  win: PadWindow,
): number | null {
  const [p, q] = a.t <= b.t ? [a, b] : [b, a]
  const d0 = Math.max(p.t, win.lo)
  const d1 = Math.min(q.t, win.hi)
  if (d1 < d0) return null
  const [ax, ay] = pointAt(p, q, d0)
  const [bx, by] = pointAt(p, q, d1)
  const u = firstWithin(ax - win.cx, ay - win.cy, bx - ax, by - ay)
  return u === null ? null : d0 + u * (d1 - d0)
}

/** pointAt — la position sur le segment [p, q] à l'instant `t`. Durée nulle : le départ. */
function pointAt(
  p: ReplayTrackReady['points'][number],
  q: ReplayTrackReady['points'][number],
  t: number,
): [number, number] {
  if (q.t <= p.t) return [p.x, p.y]
  const u = (t - p.t) / (q.t - p.t)
  return [p.x + u * (q.x - p.x), p.y + u * (q.y - p.y)]
}

/**
 * firstWithin — le plus petit `u` de [0, 1] pour lequel `p + u·d` est dans le rayon, ou null.
 *
 * C'EST UNE ÉQUATION DU SECOND DEGRÉ, et il faut sa PREMIÈRE racine, pas la distance minimale :
 * ce qu'on date est l'ENTRÉE dans le rayon, pas le point le plus proche du passage.
 */
function firstWithin(px: number, py: number, dx: number, dy: number): number | null {
  const a = dx * dx + dy * dy
  const c = px * px + py * py - PAD_APPROACH_RADIUS_M * PAD_APPROACH_RADIUS_M
  if (a === 0) return c <= 0 ? 0 : null
  const b = 2 * (px * dx + py * dy)
  const disc = b * b - 4 * a * c
  if (disc < 0) return null
  const root = Math.sqrt(disc)
  const u0 = (-b - root) / (2 * a)
  const u1 = (-b + root) / (2 * a)
  if (u1 < 0 || u0 > 1) return null
  return Math.max(u0, 0)
}
