/**
 * weaponBurstSpecs.guard.test.ts — CE QU'UNE LIGNE DE LA TABLE DES RAFALES DOIT TENIR.
 *
 * La table (`weaponBurstSpecs.ts`) grandira d'une ligne par vote d'écoute de l'utilisateur, et
 * chacune de ces lignes est écrite à la main. Les trois façons de la casser ne s'entendent pas
 * — elles rendent un SILENCE ou un son tronqué, ce qui passe pour « pas de tir » :
 *
 *  - une clé qui n'est pas un stem livré : le lecteur ne trouve aucune URL, la rafale ne part
 *    jamais, et rien ne dit pourquoi ;
 *  - un nombre de coups < 2 : la ligne ne fait rien du tout (le lecteur reprend le chemin
 *    simple dès `count <= 1`) — une table qui ment sur son effet est pire qu'une table vide ;
 *  - une rafale plus longue que le plafond de coupe : l'enveloppe l'écrête et les dernières
 *    balles sont mangées, sans erreur ni trace.
 */
import { describe, expect, it } from 'vitest'

import { SOUND_CUT_MAX_S } from './replayAudio'
import { WEAPON_SOUND_STEMS } from './replaySound'
import { WEAPON_BURST_SPECS } from './weaponBurstSpecs'

const lignes = Object.entries(WEAPON_BURST_SPECS)

/**
 * Durée MAXIMALE d'un son d'arme livré, en secondes.
 *
 * C'est une RÈGLE DE LIVRAISON, pas une mesure d'ici : « armes, lancers et mêlée à 1,2 s »
 * (décision utilisateur du 2026-08-16, lot R2.1 — rappelée en tête de `replayAudio.ts` et de
 * `replaySound.ts`). Les 26 assets d'arme livrés la respectent, la MA40 étant même plus courte
 * (1,055 s).
 *
 * ELLE EST INDISPENSABLE ICI parce que l'enveloppe d'une rafale ne couvre PAS seulement les
 * écarts entre balles : elle vaut `(coups-1) * ecart + durée DU FICHIER` (`playBurst`). Une
 * borne posée sur les seuls écarts laisserait passer une ligne dont la dernière balle serait
 * mangée par le plafond de coupe — sans erreur, sans trace, et en silence.
 */
const DUREE_MAX_ASSET_ARME_S = 1.2

/** La durée réellement enveloppée par une rafale, dans le pire cas d'asset livré. */
function dureeRafaleS(spec: { coups: number; ecartMs: number }): number {
  return ((spec.coups - 1) * spec.ecartMs) / 1000 + DUREE_MAX_ASSET_ARME_S
}

describe('WEAPON_BURST_SPECS — garde-rail de la table', () => {
  it('la table est non vide (une table morte ne se remarquerait pas)', () => {
    expect(lignes.length).toBeGreaterThan(0)
  })

  it.each(lignes)('%s : la clé est un stem d arme LIVRÉ', (stem) => {
    // Les valeurs de WEAPON_SOUND_STEMS sont les stems de fichier ; les clés sont les
    // weapon_key. La rafale se déclenche sur le STEM TIRÉ à la lecture — c'est donc parmi les
    // valeurs qu'elle doit exister.
    expect(Object.values(WEAPON_SOUND_STEMS)).toContain(stem)
  })

  it.each(lignes)('%s : au moins 2 coups, et un écart strictement positif', (_stem, spec) => {
    expect(spec.coups).toBeGreaterThanOrEqual(2)
    expect(Number.isInteger(spec.coups)).toBe(true)
    expect(spec.ecartMs).toBeGreaterThan(0)
  })

  it.each(lignes)('%s : la rafale entière tient sous le plafond de coupe, FICHIER COMPRIS', (_stem, spec) => {
    expect(dureeRafaleS(spec)).toBeLessThanOrEqual(SOUND_CUT_MAX_S)
  })

  /**
   * LA GARDE DOIT MORDRE, et rien ne le prouve tant qu'aucune ligne ne la déclenche : la table
   * n'en porte qu'une, très en dessous du plafond. Ce cas est la ligne qu'on n'écrira jamais —
   * il montre que la borne REFUSE ce qu'elle prétend refuser.
   *
   * `{coups: 3, ecartMs: 1500}` fait 3,0 s d'écarts : une borne posée sur les seuls écarts la
   * laisserait passer, alors que l'enveloppe réelle atteint 4,2 s et fait manger la dernière
   * balle par le plafond de 4,0 s.
   */
  it('REFUSE une ligne dont le FICHIER fait déborder la rafale (3 coups à 1500 ms)', () => {
    const trop = { coups: 3, ecartMs: 1500 }
    expect(((trop.coups - 1) * trop.ecartMs) / 1000).toBeLessThan(SOUND_CUT_MAX_S) // écarts seuls : passe
    expect(dureeRafaleS(trop)).toBeGreaterThan(SOUND_CUT_MAX_S) // enveloppe réelle : refusée
  })
})
