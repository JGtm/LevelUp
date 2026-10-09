import { describe, expect, it } from 'vitest'

import { impactHistory3 } from './impactHistory.fixtures'
import { buildImpactHistoryView, impactRoleToken, pointKey, scaleSentence, showAxisLabel } from './impactHistory.logic'
import { IMPACT_HISTORY_TEXT } from './impactHistoryStrings'

const fr = IMPACT_HISTORY_TEXT.fr
const en = IMPACT_HISTORY_TEXT.en

function view() {
  const v = buildImpactHistoryView(impactHistory3, 'fr', fr)
  if (!v) throw new Error('vue attendue')
  return v
}

describe('buildImpactHistoryView', () => {
  it('trois barres par soirée, dans l’ordre des joueurs', () => {
    const v = view()
    expect(v.bars).toHaveLength(9)
    expect(v.bars.slice(0, 3).map((b) => [b.evening, b.player])).toEqual([[0, 0], [0, 1], [0, 2]])
    expect(v.players).toEqual(['JGtm', 'Chocoboflor', 'Madina97294'])
  })

  it('reprend les nets servis, sans recalcul', () => {
    expect(view().nets).toEqual([
      [-9, -2, 0],
      [3.5, -5, -4],
      [1.5, -4, 3],
    ])
  })

  it('empile du plus fort barème contre l’axe vers le plus faible, les rôles à −1 en un bloc', () => {
    const jgtm = view().bars[0]
    expect(jgtm.gains).toEqual([
      { token: 'impact-gain-1', from: 0, to: 2 },
      { token: 'impact-gain-4', from: 2, to: 3 },
    ])
    expect(jgtm.losses).toEqual([
      { token: 'impact-loss-1', from: 0, to: -4 },
      { token: 'impact-loss-2', from: -4, to: -7 },
      { token: 'impact-loss-3', from: -7, to: -12 },
    ])
    expect(jgtm.gainTotal).toBe(3)
    expect(jgtm.lossTotal).toBe(-12)
  })

  it('écrit seulement les deux nets extrêmes', () => {
    expect([...view().extremes].sort()).toEqual([pointKey(0, 0), pointKey(1, 0)])
  })

  it('date seule, l’heure en seconde ligne quand deux soirées partagent la date', () => {
    const v = view()
    expect(v.twoLineAxis).toBe(true)
    expect(v.axisLabels[0]).not.toContain('\n')
    expect(v.axisLabels[1]).toContain('\n')
    expect(v.axisLabels[2]).toContain('\n')
    expect(v.axisLabels[1].split('\n')[0]).toBe(v.axisLabels[2].split('\n')[0])
    expect(v.axisLabels[1]).not.toBe(v.axisLabels[2])
  })

  it('infobulle : joueur, soirée, matchs et victoires, rôles « ×n · ±points », net', () => {
    const v = view()
    const choco = v.bars[7].tip
    expect(choco.player).toBe('Chocoboflor')
    expect(choco.evening).toMatch(/^Soirée du \d\d\/\d\d, /)
    expect(choco.matches).toBe('1 match, aucune victoire')
    expect(choco.roles.map((r) => `${r.label} ×${r.count} · ${r.points}`)).toEqual([
      'Boulet ×1 · −2',
      'Première victime ×1 · −1',
      'Voleur ×1 · −1',
    ])
    expect(choco.net).toBe('−4')
    expect(v.bars[0].tip.matches).toBe('11 matchs, 5 victoires')
    expect(v.bars[6].tip.roles).toEqual([])
  })

  it('légende : gains un par un, pertes à −1 réunies', () => {
    const { gains, losses } = view().legend
    expect(gains.map((g) => g.label)).toEqual(['Finisseur +2', 'Premier sang +2', 'Héros silencieux +1,5', 'Bourreau +1'])
    expect(losses.map((l) => l.label)).toEqual([
      'Boulet −2',
      'Faux-frère −1,5',
      'Autres rôles à −1 (Première victime, Touriste, Kamikaze, Voleur)',
    ])
    expect(losses[2].token).toBe('impact-loss-3')
  })

  it('aide et résumé accessible portent le barème et les nets', () => {
    const v = view()
    expect(v.info).toContain('JGtm, Chocoboflor et Madina97294')
    expect(v.info).toContain(
      'Barème : Finisseur et Premier sang +2, Héros silencieux +1,5, Bourreau +1 ; Boulet −2, Faux-frère −1,5, ' +
        'Première victime, Touriste, Kamikaze et Voleur −1.',
    )
    expect(v.aria).toContain('JGtm : −9, −2, 0')
  })

  it('anglais : barème à point décimal', () => {
    expect(scaleSentence(impactHistory3.scale ?? [], en)).toBe(
      'Finisher and First blood +2, Silent hero +1.5, Top killer +1 ; Last casualty −2, False brother −1.5, ' +
        'First down, Late starter, Kamikaze and Thief −1',
    )
  })

  it('rien à tracer sans soirée', () => {
    expect(buildImpactHistoryView({ ...impactHistory3, evenings: [] }, 'fr', fr)).toBeNull()
    expect(buildImpactHistoryView({ ...impactHistory3, evenings: null }, 'fr', fr)).toBeNull()
  })
})

describe('impactRoleToken', () => {
  it('un rôle inconnu prend le bout de la rampe de son signe', () => {
    expect(impactRoleToken('nouveau', 1)).toBe('impact-gain-4')
    expect(impactRoleToken('nouveau', -1)).toBe('impact-loss-3')
  })
})

describe('showAxisLabel', () => {
  it('sur téléphone, une date sur deux, la dernière toujours écrite', () => {
    expect([0, 1, 2, 3, 4].map((i) => showAxisLabel(i, 5, true))).toEqual([true, false, true, false, true])
    expect([0, 1, 2, 3].map((i) => showAxisLabel(i, 4, true))).toEqual([false, true, false, true])
    expect([0, 1].map((i) => showAxisLabel(i, 2, false))).toEqual([true, true])
  })
})

describe('textes', () => {
  // Aucun possessif ni pronom de personne dans les textes du graphe (titres, aide, légende).
  const INTERDITS_FR = /\b(mon|ma|mes|ton|ta|tes|son|sa|ses|notre|nos|votre|vos|leur|leurs|moi|toi|je|tu|nous|vous)\b/i
  const INTERDITS_EN = /\b(my|your|his|her|its|our|their|me|you|we|us|i)\b/i
  it.each([['fr', fr, INTERDITS_FR], ['en', en, INTERDITS_EN]] as const)('%s', (_l, t, re) => {
    const textes = [t.title, t.info('A, B et C', 'X +1'), t.legendNet, t.legendGrouped('−1', ['a']), t.noRole]
    for (const s of textes) expect(s, s).not.toMatch(re)
  })
})
