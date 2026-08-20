/**
 * killFx.ts — LES EFFETS DE MORT SUR LA CARTE, orientés tueur -> victime.
 *
 * POURQUOI CE CALQUE EXISTE, ET POURQUOI IL EST SÉPARÉ DES TIRS (règle du POC, conservée
 * telle quelle) : un événement de TIR ne porte pas de victime — son trait part dans la
 * direction visée, quand elle est lisible. Le fil des MORTS, lui, nomme le tueur ET la
 * victime, et leurs deux positions se relisent dans les trajectoires à l'instant de la
 * mort. C'est donc ici, et ici seulement, qu'un effet peut pointer une vraie victime.
 *
 * LA RÈGLE 89/93 (mesure du POC, portée ici) : l'effet n'est ORIENTÉ que lorsque le couple
 * est COMPLET — les deux positions relues. Victime seule, tueur seul : un marqueur non
 * orienté, jamais un axe inventé. Sur le film de référence : 89 morts sur 93 avec couple
 * complet, 3 victime seule, 1 tueur seul, 0 sans position.
 *
 * LA FAMILLE vient de `doc.killEffects` (weapon_key -> famille, table du titre) : les kills
 * du feed portent un weapon_key résolu côté base, pas un identifiant d'arme film. Une clé
 * absente (mêlée générique, source inconnue) tombe sur le rendu neutre — jamais celui d'une
 * arme voisine.
 *
 * Pas de React, pas de canvas : logique pure, testée (killFx.test.ts).
 */
import type { KillEvent } from '@/features/match-view/_momentum'

import { alignFeed } from './killFeedLogic'
import { familyOf, type ShotFamily } from './shotEffects'
import { isAliveAt, msToFrames, positionAt, trackWindow, type XY } from './replayLogic'
import type { ReplayDocumentReady, ReplayTrackReady } from './replayNormalize'

/**
 * Fenêtre d'acceptation d'une DERNIÈRE position après la fin d'une vie, en ms. La victime,
 * par construction, vient de mourir — sa trace est close, sa dernière position reste vraie.
 *
 * ELLE VAUT LA FENÊTRE DEATH D'ORIGINE (1,5 s) MAIS N'EN DÉPEND PLUS : la croix de mort est
 * passée à 2,5 s le 2026-08-18, celle-ci non. Ce sont deux questions différentes — combien de
 * temps un repère reste LISIBLE d'un côté, jusqu'où une position reste VRAIE de l'autre — et
 * les lier ferait bouger la seconde à chaque réglage d'écran de la première.
 */
export const KILLPOS_WINDOW_MS = 1_500

/**
 * Seuil MÊLÉE, en mètres monde : en deçà, la victime est à portée du geste et l'arc de
 * liaison se dessine. C'est un CHOIX adossé à une mesure (POC) : médiane 0,4-0,5 m pour
 * l'épée et la mêlée, 23 m pour le sniper — le seuil est posé dans l'intervalle qui sépare
 * ces deux populations.
 */
export const MELEE_LINK_MAX_M = 8

/** Un effet de mort, précalculé en coordonnées MONDE (la conversion px dépend du cadrage). */
export interface KillFxEntry {
  /** Frame du kill sur la grille du rejeu. */
  frame: number
  /** Origine de l'effet (le tueur — ou la victime quand lui seul manque). */
  x: number
  y: number
  /** Position de la victime, SEULEMENT quand le couple est complet. */
  vx: number | null
  vy: number | null
  /**
   * Le LIEU DE LA MORT : la position de la victime dès qu'elle est relue, que le tueur
   * l'ait été ou non. Distinct de `vx`/`vy`, qui n'existent QUE pour ORIENTER l'effet et
   * exigent donc le couple complet (règle 89/93) — la carte de chaleur des éliminations,
   * elle, n'a pas besoin du tueur pour savoir où l'on est mort. Mesure du POC : sur 93
   * morts, le couple complet en couvre 89 et les « victime seule » 3 de plus, que ce champ
   * récupère. null = victime non localisée : aucune position devinée.
   */
  deathX: number | null
  deathY: number | null
  /** Distance tueur-victime en mètres monde (null sans couple complet). */
  dist: number | null
  fam: ShotFamily
  /** Slot de la vie du tueur à cet instant (pour sa couleur) ; null si introuvable. */
  slot: number | null
  /** Germe stable : deux lectures du même instant redonnent la même forme. */
  seed: number
}

/**
 * posOfPlayerAt — position d'un joueur (xuid) à une frame, relue dans ses trajectoires.
 *
 * Patron `posOfNameAt` du POC : une vie VIVANTE à cette frame rend sa position ; sinon on
 * accepte la DERNIÈRE position d'une vie close depuis moins de `deathFrames` — c'est le cas
 * de la victime, par construction. Au-delà : null, aucune position inventée.
 */
export function posOfPlayerAt(
  lives: ReplayTrackReady[] | undefined,
  frame: number,
  deathFrames: number,
): XY | null {
  if (!lives || lives.length === 0) return null
  for (const l of lives) {
    if (isAliveAt(l, frame)) {
      const p = positionAt(l.points, frame)
      if (p) return p
    }
  }
  let best: ReplayTrackReady | null = null
  let bestD = Number.POSITIVE_INFINITY
  for (const l of lives) {
    const d = frame - trackWindow(l).end
    if (d >= 0 && d <= deathFrames && d < bestD) {
      bestD = d
      best = l
    }
  }
  if (!best) return null
  const last = best.points[best.points.length - 1]
  return last ? { x: last.x, y: last.y } : null
}

/** slotOfPlayerAt — le slot de la vie du joueur couvrant la frame (ou la dernière close). */
function slotOfPlayerAt(
  lives: ReplayTrackReady[] | undefined,
  frame: number,
  deathFrames: number,
): number | null {
  if (!lives) return null
  for (const l of lives) {
    if (isAliveAt(l, frame)) return l.slot
  }
  for (const l of lives) {
    const d = frame - trackWindow(l).end
    if (d >= 0 && d <= deathFrames) return l.slot
  }
  return null
}

/**
 * buildKillFx précalcule les effets de mort d'un document : positions monde résolues une
 * fois au chargement (patron du POC — 93 morts, aucune recherche pendant la lecture).
 *
 * L'horloge est celle du fil (`alignFeed` : l'origine publiée par l'artefact, cf. l'en-tête
 * de killFeedLogic.ts) : une seule règle de recalage pour le feed et la carte. C'est aussi
 * ce qui rend la fenêtre DEATH efficace — mesure témoin 000d5950 : 1/93 victimes relues
 * avec le recalage brut `+t0`, 90/93 une fois aligné.
 */
export function buildKillFx(
  doc: ReplayDocumentReady,
  kills: KillEvent[],
  t0Ms: number,
): KillFxEntry[] {
  if (kills.length === 0 || doc.tracks.length === 0) return []
  const deathFrames = Math.max(1, Math.round(msToFrames(KILLPOS_WINDOW_MS, doc)))
  const livesByXuid = new Map<string, ReplayTrackReady[]>()
  for (const t of doc.tracks) {
    if (!t.xuid) continue
    const list = livesByXuid.get(t.xuid)
    if (list) list.push(t)
    else livesByXuid.set(t.xuid, [t])
  }
  const out: KillFxEntry[] = []
  for (const k of alignFeed(kills, t0Ms, doc).kills) {
    const frame = Math.round(msToFrames(k.replayMs, doc))
    const killer = posOfPlayerAt(livesByXuid.get(k.xuid), frame, deathFrames)
    const victim = k.victimXuid
      ? posOfPlayerAt(livesByXuid.get(k.victimXuid), frame, deathFrames)
      : null
    const origin = killer ?? victim
    if (!origin) continue // ni tueur ni victime localisable : on ne dessine rien
    const complete = killer !== null && victim !== null
    out.push({
      frame,
      x: origin.x,
      y: origin.y,
      vx: complete ? victim.x : null,
      vy: complete ? victim.y : null,
      deathX: victim ? victim.x : null,
      deathY: victim ? victim.y : null,
      dist: complete ? Math.hypot(victim.x - killer.x, victim.y - killer.y) : null,
      fam: familyOf(k.weaponKey ? doc.killEffects?.[k.weaponKey] : undefined),
      slot: slotOfPlayerAt(livesByXuid.get(k.xuid), frame, deathFrames),
      seed: (frame * 2654435761) % 100003,
    })
  }
  return out
}
