/**
 * playerCampLabel.test.ts — LE NOM DU CAMP D'UN JOUEUR : celui du camp du film, nommé par la feuille de ses
 * membres ; aucun nom pour un joueur sans équipe au film.
 */
import { describe, expect, it } from 'vitest'

import type { ReplayPlayer } from '@/lib/replay/rosterLogic'

import { scoreboardRow } from '../test/scoreboardRow'
import { playerCampLabel } from './playerCampLabel'

const TEXT = { teamLabelFmt: (n: string) => `Équipe ${n}`, teamNumberedFmt: (n: number) => `Équipe ${n}` }

const eagle = (xuid: string): ReplayPlayer => ({
  xuid, team: 0, lives: [], board: scoreboardRow(xuid, xuid, 't0', { team_name: 'Eagle' }),
})
const cobra: ReplayPlayer = { xuid: 'C', team: 1, lives: [], board: scoreboardRow('C', 'C', 't1', { team_name: 'Cobra' }) }

describe('playerCampLabel', () => {
  it('le nom du camp du film, lu sur la feuille de ses membres', () => {
    const players = [eagle('A'), eagle('B'), cobra]
    expect(playerCampLabel(players[0], players, TEXT)).toBe('Équipe Eagle')
    expect(playerCampLabel(cobra, players, TEXT)).toBe('Équipe Cobra')
  })

  it('un joueur sans feuille prend le nom que ses coéquipiers donnent à son camp', () => {
    const sansFeuille: ReplayPlayer = { xuid: 'D', team: 0, lives: [] }
    expect(playerCampLabel(sansFeuille, [eagle('A'), sansFeuille], TEXT)).toBe('Équipe Eagle')
  })

  it('un camp que la feuille ne nomme pas : « Équipe N » du désignateur du film', () => {
    const seul: ReplayPlayer = { xuid: 'D', team: 3, lives: [] }
    expect(playerCampLabel(seul, [seul], TEXT)).toBe('Équipe 3')
  })

  it('un joueur sans équipe au film, ou en mode sans camps : aucun nom', () => {
    const sans: ReplayPlayer = { xuid: 'E', lives: [] }
    const horsCamps: ReplayPlayer = { xuid: 'F', team: -1, lives: [] }
    expect(playerCampLabel(sans, [sans, horsCamps], TEXT)).toBeNull()
    expect(playerCampLabel(horsCamps, [sans, horsCamps], TEXT)).toBeNull()
  })
})
