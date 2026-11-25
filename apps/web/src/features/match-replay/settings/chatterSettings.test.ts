/**
 * chatterSettings.test.ts — LES RÉGLAGES DES RÉPLIQUES dans la table des réglages : où ils sont rangés, ce qu'ils valent à
 * l'ouverture, ce que les préréglages en font, quand le match les propose, et le lien à leur état. Logique pure.
 */
import { describe, expect, it, vi } from 'vitest'

import { CHATTER_FILM, CHATTER_FILM_SILENT, FULL_FILM, makeSettingsContext } from '../test/settingsContextFixtures'
import { CHATTER_FREQUENCIES } from '../model/chatterLines'
import { isProposed, isShown, masterState, pinStatus, shownOfRubric } from './settingsAvailability'
import { bindSettings, type BindingInput } from './settingsBinding'
import { settingText, choiceText } from './settingsText'
import { DEFAULT_PINS, isPinnable } from './quickAccess'
import { defOf, defsOfRubric, factoryValues, PRESET_COVERED, type SettingId } from './settingsTable'
import { changedFrom, presetValues, writesToReach } from './viewPresets'

const CHATTER_IDS: readonly SettingId[] = [
  'chatterBubbles', 'chatterIcon', 'chatterFeed', 'famEnemy', 'famShields', 'famSquadDown', 'famPraise', 'famOther',
  'chatterFreq', 'chatterVoice',
]
const FAMILY_IDS: readonly SettingId[] = ['famEnemy', 'famShields', 'famSquadDown', 'famPraise', 'famOther']

describe('les réglages des répliques dans la table', () => {
  it('sont rangés là où la maquette les met : Joueurs, Fil, Son', () => {
    expect(defsOfRubric('joueurs').filter((d) => d.sub === 'chatter').map((d) => d.id)).toEqual(['chatterBubbles', 'chatterIcon'])
    expect(defsOfRubric('fil').map((d) => d.id)).toEqual([
      'feedFilter', 'planFeed', 'chatterFeed', 'famEnemy', 'famShields', 'famSquadDown', 'famPraise', 'famOther', 'chatterFreq',
    ])
    expect(defsOfRubric('son').filter((d) => d.sub === 'chatter').map((d) => d.id)).toEqual(['chatterVoice'])
  })

  it("sont tous allumés à l'ouverture, toutes familles, toutes fréquences", () => {
    const v = factoryValues()
    for (const id of CHATTER_IDS.filter((i) => i !== 'chatterFreq')) expect(v[id]).toBe(true)
    expect(v.chatterFreq).toBe('all')
  })

  it('offrent les trois fréquences, hors préréglage ; les familles ne sont pas épinglables', () => {
    expect(defOf('chatterFreq').options).toEqual([...CHATTER_FREQUENCIES])
    expect(defOf('chatterFreq').preset).toBe(false)
    for (const id of FAMILY_IDS) expect(isPinnable(id)).toBe(false)
    for (const id of ['chatterBubbles', 'chatterIcon', 'chatterFeed', 'chatterVoice'] as const) expect(isPinnable(id)).toBe(true)
  })

  it('les bulles de répliques sont épinglées d’usine, avec les quatre autres', () => {
    expect(DEFAULT_PINS).toEqual(['aim', 'trail', 'heatmap', 'planFeed', 'chatterBubbles'])
  })

  it('un libellé fr et en pour chaque réglage et chaque option de fréquence', () => {
    for (const locale of ['fr', 'en'] as const) {
      for (const id of CHATTER_IDS) expect(settingText(locale, id).label).not.toBe('')
      for (const o of CHATTER_FREQUENCIES) expect(choiceText(locale, 'chatterFreq', o).label).not.toBe(o)
    }
    expect(settingText('fr', 'famShields').label).toBe('Boucliers')
    expect(settingText('en', 'famShields').label).toBe('Shields')
  })
})

describe('ce que le match propose', () => {
  it("un film sans réplique n'en propose aucun réglage", () => {
    const c = makeSettingsContext({ film: FULL_FILM })
    for (const id of CHATTER_IDS) expect(isProposed(id, c)).toBe(false)
  })

  it("un film avec des répliques propose l'affichage, le fil, les familles, la fréquence ; l'icône et la voix s'il y a une voix", () => {
    const c = makeSettingsContext({ film: CHATTER_FILM })
    for (const id of CHATTER_IDS) expect(isProposed(id, c)).toBe(true)
  })

  it("sans voix dans la langue, ni l'icône ni la voix ne se proposent : le reste reste proposé", () => {
    const c = makeSettingsContext({ film: CHATTER_FILM_SILENT })
    expect(isProposed('chatterIcon', c)).toBe(false)
    expect(isProposed('chatterVoice', c)).toBe(false)
    for (const id of ['chatterBubbles', 'chatterFeed', 'famEnemy', 'chatterFreq'] as const) expect(isProposed(id, c)).toBe(true)
  })

  it('la voix demande aussi un son à régler', () => {
    const c = makeSettingsContext({ film: CHATTER_FILM, soundAvailable: false })
    expect(isProposed('chatterVoice', c)).toBe(false)
  })

  it("la ligne épinglée d'un match sans réplique reste atteignable mais grisée, avec sa raison", () => {
    const c = makeSettingsContext({ film: FULL_FILM })
    expect(pinStatus('chatterBubbles', c)).toEqual({ enabled: false, reason: 'match' })
  })

  it('les familles et la fréquence se rangent sous leur sous-titre, dans la rubrique Fil', () => {
    const c = makeSettingsContext({ film: CHATTER_FILM })
    const shown = shownOfRubric('fil', c, factoryValues()).map((d) => d.id)
    expect(shown).toContain('chatterFreq')
    expect(isShown('famOther', c, factoryValues())).toBe(true)
  })

  it('compte les répliques dans les « N sur M » des rubriques', () => {
    const c = makeSettingsContext({ film: CHATTER_FILM })
    const m = masterState('fil', c, factoryValues())
    expect(m.total).toBe(2 + 5) // fil posé sur le plan, répliques dans le fil, cinq familles
    expect(m.on).toBe(m.total)
  })
})

describe('les préréglages et les répliques', () => {
  const c = makeSettingsContext({ film: CHATTER_FILM })

  it('les neuf interrupteurs de répliques sont couverts, la fréquence non', () => {
    for (const id of CHATTER_IDS.filter((i) => i !== 'chatterFreq')) expect(PRESET_COVERED).toContain(id)
    expect(PRESET_COVERED).not.toContain('chatterFreq')
  })

  it("Complet allume tout, comme l'état d'ouverture", () => {
    expect(changedFrom(presetValues('complet'), factoryValues(), c)).toEqual([])
  })

  it('Combat ne garde que les familles ennemi, boucliers et équipier tombé ; ni bulle, ni icône, ni fil, ni voix', () => {
    const p = presetValues('combat')
    expect(FAMILY_IDS.filter((id) => p[id] === true)).toEqual(['famEnemy', 'famShields', 'famSquadDown'])
    for (const id of ['chatterBubbles', 'chatterIcon', 'chatterFeed', 'chatterVoice'] as const) expect(p[id]).toBe(false)
  })

  it('Déplacements garde les cinq familles sans bulle ni voix', () => {
    const p = presetValues('deplacements')
    for (const id of FAMILY_IDS) expect(p[id]).toBe(true)
    expect(p.chatterBubbles).toBe(false)
    expect(p.chatterVoice).toBe(false)
  })

  it('Objectif montre les répliques (bulles, icône, fil, voix) des familles Ennemi repéré et Autres', () => {
    const p = presetValues('objectif')
    for (const id of ['chatterBubbles', 'chatterIcon', 'chatterFeed', 'chatterVoice', 'famEnemy', 'famOther'] as const) expect(p[id]).toBe(true)
    for (const id of ['famShields', 'famSquadDown', 'famPraise'] as const) expect(p[id]).toBe(false)
  })

  it("un préréglage n'écrit pas un réglage que le match ne propose pas (film sans réplique)", () => {
    const sans = makeSettingsContext({ film: FULL_FILM })
    const ecrits = writesToReach(presetValues('combat'), factoryValues(), sans).map(([id]) => id)
    for (const id of CHATTER_IDS) expect(ecrits).not.toContain(id)
  })
})

describe('le lien aux états des réglages', () => {
  function binding(over: { toggleFamily?: ReturnType<typeof vi.fn>; setFrequency?: ReturnType<typeof vi.fn>; toggleCategory?: ReturnType<typeof vi.fn> } = {}) {
    const toggleFamily = over.toggleFamily ?? vi.fn()
    const setFrequency = over.setFrequency ?? vi.fn()
    const toggleCategory = over.toggleCategory ?? vi.fn()
    const toggleBubbles = vi.fn()
    const input = {
      settings: {
        showBubbles: true, toggleBubbles, showIcon: false, toggleIcon: vi.fn(), showFeed: true, toggleFeed: vi.fn(),
        families: { enemy: true, shields: false, squadDown: true, praise: true, other: true }, toggleFamily,
        frequency: 'rare', setFrequency,
      },
      relief: { selected: 'plan', lines: { show: true, onToggle: vi.fn() }, floor: { cut: null, follow: { on: false, onToggle: vi.fn() }, selectAll: vi.fn(), setHeight: vi.fn() } },
      orientation: { retained: null, setTurns: vi.fn() },
      sound: { on: false, volume: 0.7, toggle: vi.fn(), setVolume: vi.fn(), categories: { weapon: true, grenade: true, melee: true, equipment: true, objective: true, voice: false }, toggleCategory },
      feed: { value: 'tout', choose: vi.fn() },
      heatMode: 'presence',
      followedKnown: true,
    } as unknown as BindingInput
    return { b: bindSettings(input), toggleBubbles, toggleFamily, setFrequency, toggleCategory }
  }

  it('lit chaque réglage dans son état', () => {
    const { b } = binding()
    expect(b.values).toMatchObject({
      chatterBubbles: true, chatterIcon: false, chatterFeed: true, famEnemy: true, famShields: false,
      chatterFreq: 'rare', chatterVoice: false,
    })
  })

  it("n'écrit un interrupteur que si la valeur voulue diffère", () => {
    const { b, toggleBubbles, toggleFamily } = binding()
    b.set('chatterBubbles', true)
    expect(toggleBubbles).not.toHaveBeenCalled()
    b.set('chatterBubbles', false)
    expect(toggleBubbles).toHaveBeenCalledTimes(1)
    b.set('famShields', true)
    expect(toggleFamily).toHaveBeenCalledWith('shields')
  })

  it('écrit la fréquence, et la voix par la catégorie « voix » du son', () => {
    const { b, setFrequency, toggleCategory } = binding()
    b.set('chatterFreq', 'noFrequent')
    expect(setFrequency).toHaveBeenCalledWith('noFrequent')
    b.set('chatterVoice', true)
    expect(toggleCategory).toHaveBeenCalledWith('voice')
  })
})
