/**
 * objectiveSound.test.ts — les règles des sons d'objectif : la STATISTIQUE désigne le geste,
 * le CAMP désigne le fichier, et un camp inconnu se TAIT plutôt que d'en choisir un.
 *
 * Ce que ces tests protègent, et qui ne s'entend pas à l'écoute : qu'une capture adverse ne
 * joue jamais le jingle de la sienne (l'erreur serait invisible sur un match qu'on regarde
 * seul), et qu'une statistique inconnue reste muette au lieu de retomber sur une voisine.
 */
import { describe, expect, it } from 'vitest'

import {
  objectiveSoundEvents,
  objectiveSoundStem,
  sideResolverFromScoreboard,
} from './objectiveSound'
import { buildSoundTimeline, SOUND_CATEGORIES_DEFAULT } from './replaySound'
import { testReplayDoc } from './test/testDoc'

describe('objectiveSoundStem — la statistique désigne le geste, le camp désigne le fichier', () => {
  it('une capture joue DEUX fichiers différents selon le camp', () => {
    const ally = objectiveSoundStem('flag_captures', 'ally')
    const enemy = objectiveSoundStem('flag_captures', 'enemy')
    expect(ally).toBe('objective_flag_scored_team')
    expect(enemy).toBe('objective_flag_scored_enemy')
    expect(ally).not.toBe(enemy)
  })

  it('camp inconnu = silence sur les gestes à deux variantes (jamais un camp supposé)', () => {
    expect(objectiveSoundStem('flag_captures', 'unknown')).toBeUndefined()
    expect(objectiveSoundStem('flag_grabs', 'unknown')).toBeUndefined()
    expect(objectiveSoundStem('flag_steals', 'unknown')).toBeUndefined()
  })

  it('le retour de drapeau n a PAS de variante d équipe dans le jeu : il sonne pour tous', () => {
    for (const side of ['ally', 'enemy', 'unknown'] as const) {
      expect(objectiveSoundStem('flag_returns', side)).toBe('objective_flag_returned')
    }
  })

  it('une statistique hors table est MUETTE, jamais le son d une voisine', () => {
    expect(objectiveSoundStem('zone_secures', 'ally')).toBeUndefined()
    expect(objectiveSoundStem('flag_capture_assists', 'ally')).toBeUndefined()
    expect(objectiveSoundStem('kills', 'ally')).toBeUndefined()
  })

  it('la capture de zone a ses DEUX camps depuis le 2026-08-27', () => {
    // La paire était à moitié vide : seul le côté allié était désigné à l oreille. Le côté
    // adverse l a été le 2026-08-27 (événements `4ebe99d6` / `8594aef7` / `9fad450d`).
    expect(objectiveSoundStem('zone_captures', 'ally')).toBe('objective_zone_captured_team')
    expect(objectiveSoundStem('zone_captures', 'enemy')).toBe('objective_zone_captured_enemy')
  })

  it('un camp INCONNU se tait sur une action à deux variantes, même paire complète', () => {
    // C est la règle qui survit à la complétion des paires, et c est elle qu il faut épingler :
    // sans ligne « moi » au tableau de score, choisir un camp serait l affirmer. Le rejeu se
    // tait — annoncer un gain quand on perd une base est la pire erreur possible ici.
    expect(objectiveSoundStem('zone_captures', 'unknown')).toBeUndefined()
    expect(objectiveSoundStem('flag_captures', 'unknown')).toBeUndefined()
  })
})

describe('sideResolverFromScoreboard — le camp vient du tableau de score, comme les calques', () => {
  const board = [
    { xuid: 'MOI', team_side: 't0', is_me: true },
    { xuid: 'MATE', team_side: 't0' },
    { xuid: 'FOE', team_side: 't1' },
  ]

  it('la ligne « moi » donne l équipe alliée ; les autres se comparent à elle', () => {
    const side = sideResolverFromScoreboard(board)
    expect(side('MOI')).toBe('ally')
    expect(side('MATE')).toBe('ally')
    expect(side('FOE')).toBe('enemy')
  })

  it('sans ligne « moi », TOUT est inconnu — le rejeu ne choisit pas un camp par défaut', () => {
    const side = sideResolverFromScoreboard([{ xuid: 'A', team_side: 't0' }])
    expect(side('A')).toBe('unknown')
  })

  it('un xuid absent du tableau est inconnu, et un tableau absent aussi', () => {
    expect(sideResolverFromScoreboard(board)('AUTRE')).toBe('unknown')
    expect(sideResolverFromScoreboard(undefined)('MOI')).toBe('unknown')
  })
})

describe('objectiveSoundEvents — les actions posées sur l horloge du rejeu', () => {
  const doc = testReplayDoc({
    frameIntervalMs: 100,
    objectives: [
      { t: 10, xuid: 'MOI', stat: 'flag_captures', timeMs: 1000 },
      { t: 20, xuid: 'FOE', stat: 'flag_captures', timeMs: 2000 },
      { t: 30, xuid: 'FOE', stat: 'flag_returns', timeMs: 3000 },
      { t: 40, xuid: 'MOI', stat: 'zone_captures', timeMs: 4000 },
    ],
  })
  const side = sideResolverFromScoreboard([
    { xuid: 'MOI', team_side: 't0', is_me: true },
    { xuid: 'FOE', team_side: 't1' },
  ])

  it('chaque action sonne à SA frame, dans le camp de son auteur', () => {
    expect(objectiveSoundEvents(doc, side)).toEqual([
      { ms: 1000, stem: 'objective_flag_scored_team' },
      { ms: 2000, stem: 'objective_flag_scored_enemy' },
      { ms: 3000, stem: 'objective_flag_returned' },
      { ms: 4000, stem: 'objective_zone_captured_team' },
    ])
  })

  it('sans résolveur de camp, seules les actions sans variante d équipe sonnent', () => {
    expect(objectiveSoundEvents(doc)).toEqual([{ ms: 3000, stem: 'objective_flag_returned' }])
  })

  it('la catégorie « objectifs » du tiroir les coupe À LA CONSTRUCTION', () => {
    const sansObjectifs = buildSoundTimeline(
      doc,
      [],
      0,
      { ...SOUND_CATEGORIES_DEFAULT, objective: false },
      side,
    )
    expect(sansObjectifs).toEqual([])
    expect(buildSoundTimeline(doc, [], 0, SOUND_CATEGORIES_DEFAULT, side)).toHaveLength(4)
  })
})
