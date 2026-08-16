/**
 * Tests — ReplayKillFeed (fil des éliminations de la page de rejeu).
 *
 * Ce qu'ils protègent, dans l'ordre :
 *  1. LA SYNCHRONISATION. Un kill ne s'affiche pas avant son instant dans le rejeu.
 *  2. LA PERMANENCE (verdict user 2026-08-13) : les lignes passées RESTENT — le fil se
 *     lit en entier, il ne s'évapore plus après 8 s.
 *  3. LE RECALAGE DES DEUX HORLOGES. Les events arrivent sur l'horloge du gameplay, le
 *     rejeu tourne sur celle du film : sans `t0_ms`, le feed a ~18 s de retard.
 *  4. TUEUR / ARME / VICTIME : la victime est nommée et colorée par SON équipe ; une
 *     arme non résolue rend un repère neutre, jamais l'icône d'un autre kill.
 *  5. LES MÉDAILLES : badge en image + libellé/description en infobulle, rattachées au
 *     kill ; une médaille sans visuel garde son texte.
 */
import { describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'

import type { KillEvent } from '@/features/match-view/_momentum'
import type { MatchScoreboardRow } from '@/lib/api/types'

import type { MedalEvent } from './killFeedLogic'
import { ReplayKillFeed } from './ReplayKillFeed'
import { testReplayDoc } from './test/testDoc'
import type { ReplayDocumentReady } from './replayNormalize'

vi.mock('@/lib/accessibility', () => ({
  resolveToken: (token: string) => `var(${token})`,
  tokenCssVar: (token: string) => `var(--ac-${token})`,
}))

const T0 = 18_465

function kill(over: Partial<KillEvent>): KillEvent {
  return {
    tMs: 1_000,
    xuid: 'me',
    ally: true,
    teamID: 0,
    weaponKey: '',
    weaponLabel: '',
    weaponImageUrl: '',
    weaponTinted: false,
    assistState: '',
    assistGamertag: '',
    assistTeamID: null,
    killerDamagePct: null,
    assistDamagePct: null,
    victimXuid: '',
    victimGamertag: '',
    victimTeamID: null,
    ...over,
  }
}

function medal(over: Partial<MedalEvent>): MedalEvent {
  return {
    tMs: 1_000,
    xuid: 'me',
    gamertag: 'JGtm',
    teamID: 0,
    name: 'No Scope',
    label: 'Sans lunette',
    description: 'Tuer au sniper sans lunette.',
    imageUrl: '/static/medals/1.png',
    ...over,
  }
}

const META = new Map([
  ['me', { gamertag: 'JGtm', ally: true }],
  ['foe', { gamertag: 'Cobra01', ally: false }],
])

const SCOREBOARD: MatchScoreboardRow[] = []

function renderFeed(
  kills: KillEvent[],
  nowMs: number,
  t0Ms = T0,
  medals: MedalEvent[] = [],
  doc: ReplayDocumentReady | null = null,
  scoreboard: MatchScoreboardRow[] = SCOREBOARD,
) {
  return render(
    <ReplayKillFeed
      kills={kills}
      medals={medals}
      t0Ms={t0Ms}
      nowMs={nowMs}
      doc={doc}
      scoreboard={scoreboard}
      xuidMeta={META}
      locale="fr"
    />,
  )
}

describe('ReplayKillFeed — synchronisation et permanence', () => {
  const kills = [kill({ tMs: 16_841, xuid: 'me' }), kill({ tMs: 40_000, xuid: 'foe' })]

  it("n'affiche pas un kill qui n'a pas encore eu lieu", () => {
    // 16 841 ms d'horloge gameplay = 35 306 ms d'horloge rejeu. À 30 s de rejeu, rien.
    renderFeed(kills, 30_000)
    expect(screen.queryByText('JGtm')).toBeNull()
    expect(screen.getByText(/Rien à cet instant/)).toBeTruthy()
  })

  it("l'affiche dès que le rejeu atteint son instant", () => {
    renderFeed(kills, 35_400)
    expect(screen.getByText('JGtm')).toBeTruthy()
    // Et pas encore l'autre, 40 s plus loin sur l'horloge gameplay.
    expect(screen.queryByText('Cobra01')).toBeNull()
  })

  it('LE GARDE ensuite — le fil est permanent, aucune fenêtre (verdict user 2026-08-13)', () => {
    renderFeed(kills, 120_000)
    expect(screen.getByText('JGtm')).toBeTruthy()
    expect(screen.getByText('Cobra01')).toBeTruthy()
  })

  it('dit le compte : lignes affichées / total du match', () => {
    renderFeed(kills, 35_400)
    expect(screen.getByText('1 / 2')).toBeTruthy()
  })

  it('SANS RECALAGE, le même kill serait affiché ~18 s trop tôt — le témoin le montre', () => {
    // Le contre-test qui départage : avec t0 = 0, le kill apparaît à 16,8 s de rejeu, un
    // instant où il n'a pas eu lieu. C'est exactement le défaut que `t0_ms` corrige.
    renderFeed(kills, 17_000, 0)
    expect(screen.getByText('JGtm')).toBeTruthy()
  })
})

describe('ReplayKillFeed — arme du kill', () => {
  it("sert l'icône quand le backend a résolu l'arme", () => {
    renderFeed(
      [
        kill({
          tMs: 1_000,
          weaponImageUrl: '/static/weapons/br75.png',
          weaponLabel: 'BR75',
          weaponTinted: true,
        }),
      ],
      20_000,
    )
    expect(screen.getByRole('img', { name: 'BR75' })).toBeTruthy()
  })

  it("ne sert AUCUNE icône quand l'arme n'est pas résolue", () => {
    const { container } = renderFeed([kill({ tMs: 1_000 })], 20_000)
    expect(screen.getByText('JGtm')).toBeTruthy()
    expect(screen.queryAllByRole('img')).toEqual([])
    // Le repère neutre, lui, est bien là : la ligne existe, seule l'arme manque.
    expect(container.querySelector('.rounded-full')).toBeTruthy()
  })

  it("n'écrit aucun hex de couleur (règle color-tokens)", () => {
    const { container } = renderFeed([kill({ tMs: 1_000 })], 20_000)
    expect(container.innerHTML).not.toMatch(/#[0-9a-fA-F]{6}/)
  })

  it("TEINTE l'icône à la couleur d'équipe du TUEUR (constat gate 2026-08-13)", () => {
    // La couleur d'identité vient de la cascade du scoreboard (team_color prioritaire) :
    // l'icône-masque doit la porter, comme dans le kill feed de la carte « Dominance ».
    const sb = [
      { xuid: 'me', gamertag: 'JGtm', team_side: 't0', team_color: 'var(--team-témoin)' },
    ] as MatchScoreboardRow[]
    renderFeed(
      [
        kill({
          tMs: 1_000,
          teamID: 0,
          weaponImageUrl: '/static/weapons/br75.png',
          weaponLabel: 'BR75',
          weaponTinted: true,
        }),
      ],
      20_000,
      T0,
      [],
      null,
      sb,
    )
    const icon = screen.getByRole('img', { name: 'BR75' })
    expect(icon.getAttribute('style') ?? '').toContain('var(--team-témoin)')
  })
})

describe('ReplayKillFeed — le fil sur le référentiel des pistes (document fourni)', () => {
  /**
   * Un document 10 Hz : la victime « foe » meurt à la frame 20 (2 000 ms) puis, sur une
   * seconde vie, à la frame 80 (8 000 ms) sans qu'aucun kill ne la revendique. Le kill
   * arrive avec le décalage d'origine de l'artefact (+3 000 ms ici, +3 678 ms mesurés
   * sur le témoin 000d5950).
   */
  const docSpec = {
    frameIntervalMs: 100,
    tracks: [
      { slot: 2, team: -1, xuid: 'foe', points: [{ t: 0, x: 0, y: 0 }], startFrame: 0, endFrame: 20 },
      { slot: 2, team: -1, xuid: 'foe', points: [{ t: 40, x: 0, y: 0 }], startFrame: 40, endFrame: 80 },
    ],
  }
  const doc = testReplayDoc(docSpec)
  const kills = [kill({ tMs: 5_000, xuid: 'me', victimXuid: 'foe', victimGamertag: 'Cobra01' })]

  it("la ligne part au MÊME instant que le flash de fiche — la fin de vie, pas l'event brut", () => {
    // À 2 100 ms de rejeu, la fin de vie (2 000 ms) est passée : la ligne est là, alors
    // que l'horloge brute (5 000 ms) l'aurait fait attendre 3 s après le flash.
    renderFeed(kills, 2_100, 0, [], doc)
    expect(screen.getByText('JGtm')).toBeTruthy()
  })

  it('la mort sans kill fait sa ligne NEUTRE : le défunt, « mort », pas d\'arme ni de tueur', () => {
    renderFeed(kills, 9_000, 0, [], doc)
    expect(screen.getByText('mort')).toBeTruthy()
    // Le défunt est nommé sur SA ligne — « Cobra01 » vit aussi en victime du kill.
    expect(screen.getAllByText('Cobra01')).toHaveLength(2)
    expect(screen.getAllByRole('listitem')).toHaveLength(2)
  })

  it('avant la fin de vie orpheline, la ligne neutre n\'existe pas encore', () => {
    renderFeed(kills, 7_000, 0, [], doc)
    expect(screen.queryByText('mort')).toBeNull()
  })

  it('SANS type établi, la ligne neutre ne porte AUCUNE icône — jamais celle d\'une autre mort', () => {
    renderFeed(kills, 9_000, 0, [], doc)
    expect(screen.getByText('mort')).toBeTruthy()
    // Aucune icône dans tout le fil : ni celle d'une arme, ni celle d'un autre type de mort.
    expect(screen.queryAllByRole('img')).toHaveLength(0)
  })

  it('AVEC un type établi, la ligne porte le pictogramme du TYPE de mort, à l\'encre neutre', () => {
    // Le document date la mort sur l'horloge du FIL : la fin de vie est à 8 000 ms sur
    // l'axe du rejeu, l'origine publiée vaut 3 000 ms, donc 11 000 ms côté fil.
    const typé = testReplayDoc({
      ...docSpec,
      originMs: 3_000,
      neutralDeaths: [
        { xuid: 'foe', feedMs: 11_000, kind: 'suicide', img: '/s/suicide.png', tinted: true },
      ],
    })
    renderFeed(kills, 9_000, 0, [], typé)
    const icone = screen.getByRole('img', { name: 'Tué par sa propre arme' })
    // Masque teint (technique des icônes d'arme du fil), pas une image finie.
    const style = icone.getAttribute('style') ?? ''
    expect(style).toContain('/s/suicide.png')
    // ENCRE NEUTRE, pas une couleur d'équipe : personne n'a tué sur cette ligne.
    expect(style).toContain('divergent-neutral')
  })
})

describe('ReplayKillFeed — la victime, servie par le backend', () => {
  it('nomme la victime et la colore par SON équipe, pas celle du tueur', () => {
    renderFeed(
      [kill({ tMs: 1_000, teamID: 0, victimXuid: 'foe', victimGamertag: 'Cobra01', victimTeamID: 1 })],
      20_000,
    )
    const tueur = screen.getByText('JGtm')
    const victime = screen.getByText('Cobra01')
    expect(victime).toBeTruthy()
    expect(victime.getAttribute('style')).not.toBe(tueur.getAttribute('style'))
  })

  it('sans paire, la ligne vit sans victime — rien n’est inventé', () => {
    renderFeed([kill({ tMs: 1_000 })], 20_000)
    expect(screen.getByText('JGtm')).toBeTruthy()
    expect(screen.queryByText('Cobra01')).toBeNull()
  })
})

describe('ReplayKillFeed — les médailles du fil', () => {
  it('badge en IMAGE, libellé + description en infobulle, rattaché au kill (±500 ms)', () => {
    renderFeed([kill({ tMs: 1_000, xuid: 'me' })], 20_000, T0, [
      medal({ tMs: 1_200, xuid: 'me' }),
    ])
    const badge = screen.getByRole('img', { name: 'Sans lunette' })
    expect(badge.getAttribute('title')).toBe('Sans lunette — Tuer au sniper sans lunette.')
    // Rattachée : une seule ligne, pas de ligne médaille séparée.
    expect(screen.getAllByRole('listitem')).toHaveLength(1)
  })

  it('médaille SEULE (aucun kill proche du même acteur) : sa propre ligne, au nom du décoré', () => {
    renderFeed([], 60_000, T0, [medal({ tMs: 10_000, xuid: 'me', gamertag: 'JGtm' })])
    expect(screen.getByText('JGtm')).toBeTruthy()
    expect(screen.getByRole('img', { name: 'Sans lunette' })).toBeTruthy()
  })

  it('médaille sans visuel : son TEXTE — jamais le badge d’une autre', () => {
    renderFeed([], 60_000, T0, [
      medal({ tMs: 10_000, imageUrl: '', label: '', description: '' }),
    ])
    expect(screen.queryAllByRole('img')).toEqual([])
    expect(screen.getByText('No Scope')).toBeTruthy()
  })
})

describe('ReplayKillFeed — les TROIS états de l’assistance, jamais confondus', () => {
  it('assistant NOMMÉ : le nom, sa part, la part du tueur — et le fond affirme la contribution', () => {
    const { container } = renderFeed(
      [
        kill({
          tMs: 1_000,
          assistState: 'named',
          assistGamertag: 'Aidant77',
          assistTeamID: 0,
          killerDamagePct: 63,
          assistDamagePct: 37,
        }),
      ],
      20_000,
    )
    expect(screen.getByText('Aidant77')).toBeTruthy()
    // La part de l'assistant s'écrit « - 37 % » depuis la planche du 16/08, et la ligne
    // n'écrit PLUS « assisté par » : une MARQUE le dit, en pictogramme.
    expect(screen.getByText('- 37 %')).toBeTruthy()
    expect(screen.getByRole('img', { name: 'Assistance' })).toBeTruthy()
    expect(container.textContent).not.toMatch(/assist[ée]/i)
    expect(screen.getByText(/tueur 63 %/)).toBeTruthy()
    expect(container.querySelector('li')?.getAttribute('style')).toContain('color-mix')
  })

  it('« aucun » MESURÉ : rien d’affiché — l’information vit en infobulle, distincte d’« inconnu »', () => {
    const { container } = renderFeed(
      [kill({ tMs: 1_000, assistState: 'none', killerDamagePct: 100 })],
      20_000,
    )
    expect(screen.queryByText(/assistant inconnu/)).toBeNull()
    const line = container.querySelector('li')
    expect(line?.getAttribute('title')).toMatch(/MESURÉ/)
    expect(line?.getAttribute('style') ?? '').not.toContain('color-mix')
  })

  it('INCONNU : RIEN d’affiché — l’assistance ne se précise que quand on a l’info', () => {
    // Décision utilisateur (2026-08-12) : la plupart des kills n'ont pas d'assistant ;
    // marteler « assistant inconnu » ne renseignait personne. L'état reste distinct dans
    // la donnée (assistState: ''), l'écran n'en dit rien.
    const { container } = renderFeed([kill({ tMs: 1_000, assistState: '' })], 20_000)
    expect(screen.queryByText(/assistant inconnu|unknown assist/)).toBeNull()
    expect(container.querySelector('li')?.getAttribute('title')).toBeNull()
  })
})
