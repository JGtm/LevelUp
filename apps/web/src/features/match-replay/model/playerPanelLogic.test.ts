/**
 * Tests — la logique pure du panneau de joueur (lot 4 du plan « maquettes du rejeu »).
 *
 * Ce qu'ils protègent :
 *  1. LE JOUEUR CHOISI : un clic sur une fiche ouvre le panneau, un autre joueur REMPLACE le
 *     contenu (actions remises à zéro), recliquer la fiche ouverte referme.
 *  2. LES EXCLUSIONS (D-7) : Seul coupe Masquer ; Suivre coupe Masquer ; Masquer coupe Seul et
 *     Suivre. « Retour au fil » remet tout à zéro, Suivre compris.
 *  3. CE QUE LA CARTE MONTRE : sans Seul ni Masquer, aucun filtre (`null`, rien à brancher) ;
 *     sinon le prédicat désigne les vies du joueur choisi, bot compris (par ses vies, jamais par
 *     un xuid qu'il n'a pas).
 *  4. CE QUE LE PANNEAU AFFICHE : les totaux de FIN DE MATCH lus sur la feuille, jamais un zéro
 *     inventé — une valeur non lue est une lacune, un joueur sans feuille n'a aucune section.
 */
import { describe, expect, it } from 'vitest'

import type { MatchScoreboardRow } from '@/lib/api/types'
import type { ReplayPlayer } from '@/lib/replay/rosterLogic'

import {
  NO_PLAYER_PANEL,
  panelFocusOf,
  playerPanelSheet,
  reducePlayerPanel,
  shownSlotPredicate,
  type PlayerPanelState,
} from './playerPanelLogic'
import { scoreboardRow } from '../test/scoreboardRow'

const OPEN_A: PlayerPanelState = { open: true, xuid: 'A', follow: false, solo: false, hide: false }

describe('reducePlayerPanel — le joueur choisi', () => {
  it('un clic sur une fiche ouvre le panneau sur ce joueur, actions éteintes', () => {
    expect(reducePlayerPanel(NO_PLAYER_PANEL, { type: 'card', xuid: 'A' })).toEqual(OPEN_A)
  })

  it('une AUTRE fiche remplace le joueur et remet les actions à zéro', () => {
    const busy: PlayerPanelState = { ...OPEN_A, follow: true, solo: true }
    expect(reducePlayerPanel(busy, { type: 'card', xuid: 'B' })).toEqual({ ...OPEN_A, xuid: 'B' })
  })

  it('recliquer la fiche OUVERTE referme et remet tout à zéro', () => {
    const busy: PlayerPanelState = { ...OPEN_A, hide: true }
    expect(reducePlayerPanel(busy, { type: 'card', xuid: 'A' })).toEqual(NO_PLAYER_PANEL)
  })

  it('« Retour au fil » referme et arrête Suivre, Seul et Masquer', () => {
    const busy: PlayerPanelState = { ...OPEN_A, follow: true, solo: true }
    expect(reducePlayerPanel(busy, { type: 'close' })).toEqual(NO_PLAYER_PANEL)
  })

  it('fermer un panneau déjà fermé rend le même objet (personne n’est réveillé)', () => {
    expect(reducePlayerPanel(NO_PLAYER_PANEL, { type: 'close' })).toBe(NO_PLAYER_PANEL)
  })
})

describe('reducePlayerPanel — les exclusions de D-7', () => {
  it('Seul coupe Masquer', () => {
    const s = reducePlayerPanel({ ...OPEN_A, hide: true }, { type: 'toggle', action: 'solo' })
    expect(s).toMatchObject({ solo: true, hide: false })
  })

  it('Suivre coupe Masquer et laisse Seul', () => {
    const s = reducePlayerPanel({ ...OPEN_A, hide: true }, { type: 'toggle', action: 'follow' })
    expect(s).toMatchObject({ follow: true, hide: false, solo: false })
    const keep = reducePlayerPanel({ ...OPEN_A, solo: true }, { type: 'toggle', action: 'follow' })
    expect(keep).toMatchObject({ follow: true, solo: true })
  })

  it('Masquer coupe Seul ET Suivre', () => {
    const s = reducePlayerPanel({ ...OPEN_A, solo: true, follow: true }, { type: 'toggle', action: 'hide' })
    expect(s).toMatchObject({ hide: true, solo: false, follow: false })
  })

  it('rebasculer une action allumée l’éteint, sans toucher aux autres', () => {
    const s = reducePlayerPanel({ ...OPEN_A, solo: true, follow: true }, { type: 'toggle', action: 'solo' })
    expect(s).toMatchObject({ solo: false, follow: true })
  })

  it('une action sans panneau ouvert est sans effet', () => {
    expect(reducePlayerPanel(NO_PLAYER_PANEL, { type: 'toggle', action: 'solo' })).toBe(NO_PLAYER_PANEL)
  })

  it('un geste manuel sur la carte arrête Suivre seulement', () => {
    const s = reducePlayerPanel({ ...OPEN_A, follow: true, solo: true }, { type: 'stopFollow' })
    expect(s).toMatchObject({ follow: false, solo: true, open: true })
    expect(reducePlayerPanel(OPEN_A, { type: 'stopFollow' })).toBe(OPEN_A)
  })
})

function player(xuid: string, lives: Array<{ slot: number; start: number; end: number }>, extra: Partial<ReplayPlayer> = {}): ReplayPlayer {
  return {
    xuid,
    lives: lives.map((l) => ({
      slot: l.slot, team: 0, xuid, startFrame: l.start, endFrame: l.end,
      points: [{ t: l.start, x: 0, y: 0 }],
    })) as never,
    ...extra,
  }
}

const A = player('A', [{ slot: 1, start: 0, end: 50 }, { slot: 3, start: 60, end: 100 }])
const BOT = player('bot:Marc', [{ slot: 7, start: 0, end: 100 }], { bot: true })

describe('panelFocusOf — le joueur dont le panneau est ouvert', () => {
  it('rend le joueur et ses actions, ou null panneau fermé / joueur parti du roster', () => {
    expect(panelFocusOf(OPEN_A, [A, BOT])).toEqual({ player: A, follow: false, solo: false, hide: false })
    expect(panelFocusOf(NO_PLAYER_PANEL, [A])).toBeNull()
    expect(panelFocusOf({ ...OPEN_A, xuid: 'Z' }, [A])).toBeNull()
  })

  it('un bot est un joueur comme les autres : sa clé de roster le retrouve', () => {
    const s: PlayerPanelState = { ...OPEN_A, xuid: 'bot:Marc', follow: true }
    expect(panelFocusOf(s, [A, BOT])?.player).toBe(BOT)
  })
})

describe('shownSlotPredicate — ce que la carte montre', () => {
  it('ni Seul ni Masquer : aucun filtre à brancher', () => {
    expect(shownSlotPredicate({ player: A, follow: true, solo: false, hide: false })).toBeNull()
    expect(shownSlotPredicate(null)).toBeNull()
  })

  it('Masquer retire les vies du joueur, et seulement elles', () => {
    const shown = shownSlotPredicate({ player: A, follow: false, solo: false, hide: true })!
    expect(shown(1, 10)).toBe(false) // sa première vie
    expect(shown(3, 70)).toBe(false) // sa seconde vie, autre slot
    expect(shown(2, 10)).toBe(true) // un autre joueur
  })

  it('le slot réattribué : la vie d’un AUTRE joueur sur le même slot reste affichée', () => {
    // Le slot 1 appartient à A jusqu'à l'image 50 ; à l'image 80 il est tenu par un autre.
    const shown = shownSlotPredicate({ player: A, follow: false, solo: false, hide: true })!
    expect(shown(1, 80)).toBe(true)
  })

  it('Seul n’affiche que les vies du joueur', () => {
    const shown = shownSlotPredicate({ player: A, follow: false, solo: true, hide: false })!
    expect(shown(1, 10)).toBe(true)
    expect(shown(3, 70)).toBe(true)
    expect(shown(2, 10)).toBe(false)
  })

  it('un bot se filtre par ses vies : il n’a pas de xuid', () => {
    const shown = shownSlotPredicate({ player: BOT, follow: false, solo: false, hide: true })!
    expect(shown(7, 20)).toBe(false)
    expect(shown(1, 20)).toBe(true)
  })
})

function sheet(over: Partial<MatchScoreboardRow> = {}): MatchScoreboardRow {
  return scoreboardRow('A', 'Alpha', 'a', over)
}

describe('playerPanelSheet — les totaux de fin de match, sans rien inventer', () => {
  it('sans ligne de feuille : aucune section, et il le dit', () => {
    expect(playerPanelSheet(undefined)).toEqual({ hasSheet: false, sections: [] })
  })

  it('une feuille entièrement non lue ne produit AUCUNE section — jamais de zéro', () => {
    const r = playerPanelSheet(sheet({ score: undefined, headshot_kills: undefined, power_weapon_kills: undefined, melee_kills: undefined }))
    expect(r.hasSheet).toBe(true)
    expect(r.sections).toEqual([])
  })

  it('la précision : pourcentage, touches et tirs', () => {
    const r = playerPanelSheet(sheet({ accuracy: 42.5, shots_hit: 85, shots_fired: 200 }))
    expect(r.sections[0]).toEqual({
      id: 'precision',
      rows: [
        { key: 'precision', format: 'percent', value: 42.5 },
        { key: 'shotsHit', format: 'integer', value: 85 },
        { key: 'shotsFired', format: 'integer', value: 200 },
      ],
    })
  })

  it('une valeur non lue AU MILIEU d’une section est une lacune (null), pas un zéro', () => {
    const r = playerPanelSheet(sheet({ accuracy: 40, shots_hit: undefined, shots_fired: 100 }))
    expect(r.sections[0].rows.map((x) => x.value)).toEqual([40, null, 100])
  })

  it('un vrai zéro MESURÉ reste un zéro', () => {
    const r = playerPanelSheet(sheet({ damage_dealt: 0, damage_taken: 120 }))
    const damage = r.sections.find((s) => s.id === 'damage')!
    expect(damage.rows.map((x) => x.value)).toEqual([0, 120])
  })

  it('les dégâts sont arrondis à l’entier', () => {
    const r = playerPanelSheet(sheet({ damage_dealt: 3456.7, damage_taken: 99.2 }))
    const damage = r.sections.find((s) => s.id === 'damage')!
    expect(damage.rows.map((x) => x.value)).toEqual([3457, 99])
  })

  it('le détail du score : score, tête, arme de puissance, corps à corps ; la grenade seulement si la feuille la porte', () => {
    const r = playerPanelSheet(sheet({ score: 1800, headshot_kills: 4, power_weapon_kills: 2, melee_kills: 1 }))
    const score = r.sections.find((s) => s.id === 'score')!
    expect(score.rows.map((x) => x.key)).toEqual(['score', 'headshotKills', 'powerWeaponKills', 'meleeKills'])
    const withGrenade = playerPanelSheet(sheet({ score: 1800, grenade_kills: 3 }))
    expect(withGrenade.sections.find((s) => s.id === 'score')!.rows.map((x) => x.key)).toContain('grenadeKills')
  })

  it('l’objectif : seuls les champs de la table fermée que la feuille porte, zéro mesuré compris', () => {
    const r = playerPanelSheet(sheet({ objective: { flag_captures: 2, flag_returns: 0, mystery_stat: 9 } as never }))
    const obj = r.sections.find((s) => s.id === 'objective')!
    expect(obj.rows).toEqual([
      { key: 'flag_captures', format: 'integer', value: 2 },
      { key: 'flag_returns', format: 'integer', value: 0 },
    ])
  })

  it('sans objectif sur la feuille, pas de section d’objectif', () => {
    expect(playerPanelSheet(sheet({ objective: undefined })).sections.find((s) => s.id === 'objective')).toBeUndefined()
  })

  it('les armes : les cinq plus meurtrières, les zéros écartés', () => {
    const weapon = (id: number, kills: number) => ({ weapon_id: id, label: `W${id}`, kills })
    const r = playerPanelSheet(sheet({
      weapon_kills: [weapon(1, 2), weapon(2, 9), weapon(3, 0), weapon(4, 5), weapon(5, 1), weapon(6, 3), weapon(7, 4)],
    }))
    const w = r.sections.find((s) => s.id === 'weapons')!
    expect(w.weapons?.map((x) => x.weaponId)).toEqual([2, 4, 7, 6, 1])
  })

  it('l’ordre des sections est celui de la maquette : précision, dégâts, détail du score, objectif, armes', () => {
    const r = playerPanelSheet(sheet({
      accuracy: 50, shots_hit: 1, shots_fired: 2, damage_dealt: 10, damage_taken: 5, score: 100,
      objective: { flag_captures: 1 } as never, weapon_kills: [{ weapon_id: 1, kills: 1 }],
    }))
    expect(r.sections.map((s) => s.id)).toEqual(['precision', 'damage', 'score', 'objective', 'weapons'])
  })
})
