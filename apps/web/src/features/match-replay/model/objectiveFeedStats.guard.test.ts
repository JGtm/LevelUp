/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail : LA LISTE FERMÉE DES ACTIONS D'OBJECTIF DU FIL EST CELLE DU DÉCODEUR.
 *
 * `OBJECTIVE_FEED_STATS` reprend les statistiques d'objectif que `named.go` nomme (constantes
 * `Stat*`, hors `kills` et `assists`, qui ne sont pas des objectifs). Une statistique ajoutée au
 * décodeur sans libellé ici rendrait la ligne muette sans que rien le dise : ce test le dit.
 * Et chaque action de la liste a son libellé dans CHAQUE langue du rejeu.
 */
import { readFileSync } from 'node:fs'

import { describe, expect, it } from 'vitest'

import { KNOWN_LOCALES } from '@/lib/i18n/locale'

import { REPLAY_TEXT } from '../i18n/i18n'
import { instantaneFilm } from '../test/featureFiles'
import { OBJECTIVE_FEED_STATS, objectiveFamilyOfFeedStat } from './objectiveFeedLogic'
import { OBJECTIVE_FAMILIES } from './objectiveFamilies'

const NAMED_GO = instantaneFilm('internal/facts/objectives/named.go.txt')

/** Les statistiques qui ne sont pas des objectifs : ancre d'identité du balayage et son contrôle croisé. */
const NOT_OBJECTIVES = new Set(['kills', 'assists'])

function namedStats(): string[] {
  const src = readFileSync(NAMED_GO, 'utf8')
  return [...src.matchAll(/^\s*Stat[A-Za-z]+\s*=\s*"([a-z_]+)"/gm)]
    .map((m) => m[1])
    .filter((s) => !NOT_OBJECTIVES.has(s))
}

describe('garde-rail : les actions d’objectif du fil suivent named.go', () => {
  it('la liste fermée est exactement l’ensemble des statistiques d’objectif nommées par le décodeur', () => {
    const go = namedStats()
    expect(go.length, 'named.go doit rester lisible : aucune constante Stat* trouvée').toBeGreaterThan(0)
    expect([...OBJECTIVE_FEED_STATS].sort()).toEqual([...go].sort())
  })

  it('chaque action de la liste porte un préfixe de famille connu', () => {
    for (const stat of OBJECTIVE_FEED_STATS) {
      expect(OBJECTIVE_FAMILIES).toContain(objectiveFamilyOfFeedStat(stat))
    }
  })

  it('chaque action a un libellé non vide dans chaque langue du rejeu', () => {
    for (const locale of KNOWN_LOCALES) {
      const labels = REPLAY_TEXT[locale].feedObjectiveAction
      for (const stat of OBJECTIVE_FEED_STATS) {
        expect(labels[stat], `${locale} : libellé manquant pour ${stat}`).toBeTruthy()
      }
    }
  })
})
