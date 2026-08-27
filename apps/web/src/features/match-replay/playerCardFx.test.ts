/**
 * Tests — playerCardFx (la composition des effets d'une fiche joueur).
 *
 * CE QU'ILS PROTÈGENT : la grammaire des effets — un état se porte par le fond ou par un
 * cadre sur la COUCHE en retrait, un événement par un éclat à délai négatif, les ombres
 * s'accumulent, le fond se compose — et la règle « une fiche morte ne porte que la mort ».
 * Les couleurs sont des tokens, jamais des littéraux : les assertions le vérifient au
 * passage.
 */
import { describe, expect, it } from 'vitest'

import { NO_ZONES, type ZonePresence } from './equipmentZones'
import { REPLAY_TEXT } from './i18n'
import { hasUnderLayer, playerCardFx, type CardFxInput } from './playerCardFx'

function input(over: Partial<CardFxInput> = {}): CardFxInput {
  return {
    alive: true,
    deathAge: -1,
    lifeAge: -1,
    teleportAge: -1,
    flashFrames: 14,
    equipment: null,
    zones: NO_ZONES,
    text: REPLAY_TEXT.fr,
    ...over,
  }
}

const zones = (over: Partial<ZonePresence>): ZonePresence => ({ ...NO_ZONES, ...over })

describe('playerCardFx — la mort, redessinée', () => {
  it('la fiche s’éteint (voile sombre), lavis rouge dégradé, fin cadre — pas un aplat', () => {
    const fx = playerCardFx(input({ alive: false, deathAge: 100 }))
    expect(fx.underStyle.backgroundColor).toContain('var(--replay-label-stroke)')
    expect(fx.underStyle.backgroundImage).toContain('linear-gradient(90deg')
    expect(fx.underStyle.backgroundImage).toContain('var(--ac-destructive)')
    expect(fx.underStyle.boxShadow).toContain('var(--ac-destructive)')
    expect(fx.flashClass).toBe('')
    expect(fx.title).toBeUndefined()
    expect(fx.translocationDelay).toBeNull()
  })

  it('dans la fenêtre du coup fatal : l’éclat, à délai négatif proportionnel à l’âge', () => {
    const fx = playerCardFx(input({ alive: false, deathAge: 7 }))
    expect(fx.flashClass).toBe('replay-flash-death')
    expect(fx.underStyle.animationDelay).toBe('-0.930s')
  })
})

describe('playerCardFx — les éclats d’une fiche vivante', () => {
  it('réapparition : classe et délai négatif sur la couche d’effets', () => {
    const fx = playerCardFx(input({ lifeAge: 7 }))
    expect(fx.flashClass).toBe('replay-flash-respawn')
    expect(fx.underStyle.animationDelay).toBe('-0.600s')
  })

  it('translocation : le FOURREAU (incrustation) porte son délai — dans la fenêtre seulement', () => {
    const fx = playerCardFx(input({ teleportAge: 7 }))
    expect(fx.translocationDelay).toBe('-0.600s')
    expect(fx.flashClass).toBe('')
    expect(fx.title).toBe(REPLAY_TEXT.fr.translocationFlash)
    expect(playerCardFx(input({ teleportAge: 15 })).translocationDelay).toBeNull()
  })

  it('réapparition et translocation COEXISTENT : deux couches, deux animations', () => {
    const fx = playerCardFx(input({ lifeAge: 5, teleportAge: 2 }))
    expect(fx.flashClass).toBe('replay-flash-respawn')
    expect(fx.translocationDelay).toBe('-0.171s')
  })
})

describe('playerCardFx — verre trempé du camouflage', () => {
  const camo = { camo: true, overshield: false }

  it('flou, voile au foreground sur le fond de carte, et REFLETS diagonaux en couche image', () => {
    const fx = playerCardFx(input({ equipment: camo }))
    expect(fx.underStyle.backdropFilter).toBe('blur(6px)')
    expect(fx.underStyle.backgroundColor).toContain('var(--foreground)')
    expect(fx.underStyle.backgroundColor).toContain('var(--card)')
    expect(fx.underStyle.backgroundImage).toContain('linear-gradient(115deg')
    // Le liseré ET la tranche haute éclairée — les deux signes du verre.
    expect(fx.underStyle.boxShadow).toContain('var(--border)')
    expect(fx.underStyle.boxShadow).toContain('inset 0 1px 0')
    // Jamais une opacité réduite sur toute la fiche.
    expect(fx.underStyle.opacity).toBeUndefined()
  })
})

describe('playerCardFx — encadrés (surbouclier, champ de réparation)', () => {
  it('surbouclier : cadre et halo au token legendary, aucun fond propre', () => {
    const fx = playerCardFx(input({ equipment: { camo: false, overshield: true } }))
    expect(fx.underStyle.boxShadow).toContain('var(--ac-legendary)')
    expect(fx.underStyle.backgroundColor).toBeUndefined()
  })

  it('champ de réparation : cadre et halo au token success — celui de la jauge de santé', () => {
    const fx = playerCardFx(input({ zones: zones({ repair: true }) }))
    expect(fx.underStyle.boxShadow).toContain('var(--ac-success)')
    expect(fx.title).toBe(REPLAY_TEXT.fr.zonePresence.field)
  })

  it('les cadres se COMPOSENT avec le verre : les ombres s’accumulent', () => {
    const fx = playerCardFx(input({
      equipment: { camo: true, overshield: true },
      zones: zones({ repair: true }),
    }))
    expect(fx.underStyle.boxShadow).toContain('var(--border)')
    expect(fx.underStyle.boxShadow).toContain('var(--ac-legendary)')
    expect(fx.underStyle.boxShadow).toContain('var(--ac-success)')
  })
})

describe('playerCardFx — écran occultant', () => {
  it('sur la couche d’effets : flou léger, voile sombre discret, contour de la même encre', () => {
    // Le NUAGE NOIR, lui, vit sur l'incrustation (classe replay-zone-cloud) : la
    // couche du dessous ne porte que le versant discret.
    const fx = playerCardFx(input({ zones: zones({ shroud: true }) }))
    expect(fx.underStyle.backdropFilter).toBe('blur(3px)')
    expect(fx.underStyle.backgroundColor).toContain('var(--replay-label-stroke)')
    expect(fx.underStyle.boxShadow).toContain('var(--replay-label-stroke)')
    expect(fx.underStyle.backgroundImage).toBeUndefined()
  })

  it('composé au camouflage : le flou le plus fort, le voile sombre TEINTE le verre', () => {
    const fx = playerCardFx(input({
      equipment: { camo: true, overshield: false },
      zones: zones({ shroud: true }),
    }))
    expect(fx.underStyle.backdropFilter).toBe('blur(6px)')
    expect(fx.underStyle.backgroundColor).toContain('var(--replay-label-stroke)')
    expect(fx.underStyle.backgroundColor).toContain('var(--foreground)')
    expect(fx.underStyle.backgroundImage).toContain('linear-gradient')
  })
})

describe('playerCardFx — capteur adverse, infobulle, couche vide', () => {
  it('le capteur ne pose RIEN sur la couche d’effets (l’incrustation s’en charge), mais se dit', () => {
    const fx = playerCardFx(input({ zones: zones({ sensorSincePingMs: 200 }) }))
    expect(fx.underStyle).toEqual({})
    expect(hasUnderLayer(fx)).toBe(false)
    expect(fx.title).toBe(REPLAY_TEXT.fr.zonePresence.sensor)
  })

  it('plusieurs états : l’infobulle les enchaîne dans un ordre stable ; aucun état : rien', () => {
    const fx = playerCardFx(input({
      equipment: { camo: true, overshield: false },
      zones: zones({ repair: true, sensorSincePingMs: 10 }),
    }))
    expect(fx.title).toBe(
      [
        REPLAY_TEXT.fr.equipmentActive.camo,
        REPLAY_TEXT.fr.zonePresence.field,
        REPLAY_TEXT.fr.zonePresence.sensor,
      ].join(' · '),
    )
    const rien = playerCardFx(input())
    expect(rien.title).toBeUndefined()
    expect(hasUnderLayer(rien)).toBe(false)
  })
})
