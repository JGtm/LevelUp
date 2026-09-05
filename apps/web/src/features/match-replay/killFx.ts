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
 * LES DEUX POSITIONS SONT CELLES DU RÉSOLVEUR DE PORTEUR (`carrierPosition.ts`, 2026-09-05) :
 * un joueur tué au volant, ou en passager, explose SUR SON VÉHICULE. Un bipède attaché ne
 * réplique plus sa position monde et sa trace s'interpole en ligne droite à travers le décor —
 * l'effet de mort le mieux daté du match se posait donc, pour ces morts-là, à un endroit où
 * personne n'était. La primitive de relecture (`posOfPlayerAt`) a été rapatriée dans son module
 * canonique (`livesPosition.ts`) le même jour : ce fichier n'entretient plus AUCUNE copie de
 * l'index des vies, et le cycle d'imports qui justifiait cette copie n'existe plus.
 *
 * Pas de React, pas de canvas : logique pure, testée (killFx.test.ts).
 */
import type { KillEvent } from '@/features/match-view/_momentum'

import { buildCarrierPosAt } from './carrierPosition'
import { alignFeed } from './killFeedLogic'
import { buildLivesByXuid, deathWindowFrames } from './livesPosition'
import { familyOf, type ShotFamily } from './shotEffects'
import { isAliveAt, msToFrames, trackWindow } from './replayLogic'
import type { ReplayDocumentReady, ReplayTrackReady } from './replayNormalize'

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
  // LES POSITIONS PASSENT PAR LE RÉSOLVEUR DE PORTEUR (carrierPosition.ts, 2026-09-05) : un
  // joueur tué AU VOLANT — ou en passager — explose là où roule son véhicule, pas sur la ligne
  // droite qu'interpolait sa trace de bipède pendant qu'il ne répliquait plus. La fenêtre
  // après-mort du repli reste celle de la victime, qui vient de mourir par construction.
  const posOf = buildCarrierPosAt(doc)
  // L'INDEX DES VIES NE SERT PLUS QU'AU SLOT (la couleur de l'effet) : le slot est une
  // propriété de la VIE DU BIPÈDE, qu'aucun véhicule ne déplace. Index et fenêtre viennent de
  // livesPosition.ts — plus aucune copie locale depuis que la primitive y a été rapatriée.
  const lives = buildLivesByXuid(doc.tracks)
  const deathFrames = deathWindowFrames(doc)
  const out: KillFxEntry[] = []
  for (const k of alignFeed(kills, t0Ms, doc).kills) {
    const frame = Math.round(msToFrames(k.replayMs, doc))
    const killer = posOf(k.xuid, frame)
    const victim = k.victimXuid ? posOf(k.victimXuid, frame) : null
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
      slot: slotOfPlayerAt(lives.get(k.xuid), frame, deathFrames),
      seed: (frame * 2654435761) % 100003,
    })
  }
  return out
}
