/**
 * Tests — le pied « Éliminations » de la fiche du Standard.
 *
 * Ce qu'ils protègent :
 *  1. LE TITRE ET LE LIEN : « Éliminations » et « Fil complet » (qui rend la colonne), dans chaque langue.
 *  2. LES DEUX DERNIÈRES LIGNES DU FIL DU MATCH, tous joueurs confondus, de la plus récente à la plus ancienne,
 *     lues par `useFeedView` : la MÊME liste que la colonne, coupée à l'instant lu.
 *  3. LE FILTRE COURANT du fil s'applique à l'extrait.
 *  4. LE JOUEUR DE LA FICHE est en gras là où son nom apparaît (acteur ou victime), et lui seul.
 *  5. AUCUNE LIGNE INVENTÉE : sans ligne à cet instant, la liste est vide, le titre et le lien restent.
 */
import { cleanup, fireEvent, render, renderHook, screen, within } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import type { KillEvent } from '@/features/match-view/_momentum'
import { filmAllegianceOf } from '@/lib/replay/filmAllegiance'
import type { ReplayPlayer } from '@/lib/replay/rosterLogic'

import { useFeedView } from '../hooks/useFeedView'
import { COTES_FICHE_STANDARD } from '../model/cotesFicheStandard'
import { buildFeedEntries } from '../model/killFeedLogic'
import { FEED_FILTER_KEY } from '../settings/replayPreferences'
import { testReplayDoc } from '../test/testDoc'
import { ReplayFeedExtract, type ReplayFeedExtractProps } from './ReplayFeedExtract'

vi.mock('@/lib/accessibility', () => ({
  resolveToken: (token: string) => `var(${token})`,
  tokenCssVar: (token: string) => `var(--ac-${token})`,
}))

const META = new Map([
  ['me', { gamertag: 'JGtm' }],
  ['foe', { gamertag: 'Cobra01' }],
  ['pal', { gamertag: 'Alpha' }],
])

const ROSTER = [
  { xuid: 'me', filmIndex: 0, name: 'JGtm', team: 0 },
  { xuid: 'foe', filmIndex: 1, name: 'Cobra01', team: 1 },
  { xuid: 'pal', filmIndex: 2, name: 'Alpha', team: 0 },
]

function killEvent(over: Partial<KillEvent>): KillEvent {
  return {
    tMs: 1_000, xuid: 'me', ally: true, teamID: 0, weaponKey: '', weaponLabel: '', weaponImageUrl: '',
    weaponTinted: false, assistState: '', assistGamertag: '', assistTeamID: null, killerDamagePct: null,
    assistDamagePct: null, victimXuid: 'foe', victimGamertag: 'Cobra01', ...over,
  }
}

/** Trois éliminations ; JGtm tue en 2 s, Alpha tue en 5 s, Cobra01 tue JGtm en 9 s. */
const KILLS = [
  killEvent({ tMs: 2_000 }),
  killEvent({ tMs: 5_000, xuid: 'pal', victimXuid: 'foe', victimGamertag: 'Cobra01' }),
  killEvent({ tMs: 9_000, xuid: 'foe', victimXuid: 'me', victimGamertag: 'JGtm' }),
]

/** Une prise d'objectif plus récente que la dernière élimination : « Tout » la montre, « Éliminations » non. */
const DOC = testReplayDoc({
  frameIntervalMs: 100,
  roster: ROSTER,
  objectives: [
    { stat: 'flag_grabs', t: 40, xuid: 'me', timeMs: 4_000 },
    { stat: 'flag_captures', t: 110, xuid: 'foe', timeMs: 11_000 },
  ] as never,
})
const ALLEGIANCE = filmAllegianceOf(DOC, [], 'me')
const ENTRIES = buildFeedEntries(KILLS, [], 0, null)

function player(xuid: string, name: string): ReplayPlayer {
  return { xuid, filmName: name, team: 0, lives: [] }
}

function setup(over: { nowMs?: number; locale?: 'fr' | 'en'; who?: ReplayPlayer } = {}) {
  const onFullFeed = vi.fn()
  const props: Omit<ReplayFeedExtractProps, 'player' | 'onFullFeed'> = {
    entries: ENTRIES,
    nowMs: over.nowMs ?? 100_000,
    playWindow: null,
    scoreboard: [],
    xuidMeta: META,
    allegiance: ALLEGIANCE,
    locale: over.locale ?? 'fr',
    doc: DOC,
  }
  render(<ReplayFeedExtract {...props} player={over.who ?? player('me', 'JGtm')} onFullFeed={onFullFeed} />)
  return { onFullFeed, props }
}

const rows = () => screen.getAllByRole('listitem')

beforeEach(() => window.localStorage.setItem(FEED_FILTER_KEY, 'kills'))
afterEach(() => {
  cleanup()
  window.localStorage.removeItem(FEED_FILTER_KEY)
})

describe('ReplayFeedExtract — le titre et le lien', () => {
  it('« Éliminations » et « Fil complet » ; le lien rend la colonne', () => {
    const { onFullFeed } = setup()
    const pied = screen.getByRole('region', { name: 'Éliminations' })
    expect(within(pied).getByText('Éliminations')).toBeTruthy()
    fireEvent.click(within(pied).getByRole('button', { name: 'Fil complet' }))
    expect(onFullFeed).toHaveBeenCalledTimes(1)
  })

  it('en anglais', () => {
    setup({ locale: 'en' })
    expect(screen.getByRole('button', { name: 'Full feed' })).toBeTruthy()
    expect(screen.getByRole('region', { name: 'Kills' })).toBeTruthy()
  })

  it('le pied ne se comprime pas et un filet le sépare de la zone qui défile', () => {
    setup()
    const pied = screen.getByRole('region', { name: 'Éliminations' })
    expect(pied.className).toContain('shrink-0')
    expect(pied.className).toContain('border-t')
    expect(pied.style.paddingTop).toBe(`${COTES_FICHE_STANDARD.pied.margeHautePx}px`)
  })
})

describe('ReplayFeedExtract — les deux dernières lignes', () => {
  it('deux lignes, la plus récente en tête : celles de useFeedView, coupées à l’instant lu', () => {
    const { props } = setup()
    const { result } = renderHook(() =>
      useFeedView({ entries: ENTRIES, nowMs: props.nowMs, doc: DOC, locale: 'fr', allegiance: ALLEGIANCE }),
    )
    // La liste de la colonne a ses trois éliminations, la plus récente d'abord.
    expect(result.current.lines).toHaveLength(3)
    const lignes = rows()
    expect(lignes).toHaveLength(COTES_FICHE_STANDARD.pied.lignes)
    expect(lignes).toHaveLength(2)
    // 9 s : Cobra01 tue JGtm ; 5 s : Alpha tue Cobra01 ; celle de 2 s est hors de l'extrait.
    expect(lignes[0].textContent).toMatch(/Cobra01.*JGtm/)
    expect(lignes[1].textContent).toMatch(/Alpha.*Cobra01/)
    expect(screen.queryAllByText('JGtm')).toHaveLength(1)
  })

  it('l’instant lu décide : à 6 s, la dernière est celle de 5 s, et il n’y en a que deux', () => {
    setup({ nowMs: 6_000 })
    const lignes = rows()
    expect(lignes).toHaveLength(2)
    expect(lignes[0].textContent).toMatch(/Alpha.*Cobra01/)
    expect(lignes[1].textContent).toMatch(/JGtm.*Cobra01/)
  })

  it('à 3 s, une seule ligne : rien n’est inventé pour en faire deux', () => {
    setup({ nowMs: 3_000 })
    expect(rows()).toHaveLength(1)
  })

  it('avant la première élimination, la liste est vide ; le titre et le lien restent', () => {
    setup({ nowMs: 500 })
    expect(screen.queryAllByRole('listitem')).toHaveLength(0)
    expect(screen.getByRole('button', { name: 'Fil complet' })).toBeTruthy()
  })

  it('l’extrait suit le filtre courant : « Tout » montre la prise d’objectif plus récente que les éliminations', () => {
    window.localStorage.setItem(FEED_FILTER_KEY, 'tout')
    setup()
    const lignes = rows()
    expect(lignes).toHaveLength(2)
    // La capture de 11 s est la ligne la plus récente ; la dernière élimination (9 s) vient ensuite.
    expect(lignes[0].textContent).not.toMatch(/Cobra01.*JGtm/)
    expect(lignes[1].textContent).toMatch(/Cobra01.*JGtm/)
  })
})

describe('ReplayFeedExtract — le joueur de la fiche en gras', () => {
  it('en gras comme victime et comme acteur ; les autres noms ne le sont pas', () => {
    setup({ nowMs: 6_000 })
    const [recente, ancienne] = rows()
    // 5 s : Alpha tue Cobra01 ; 2 s : JGtm tue Cobra01 (celle de 9 s est hors de l'instant lu).
    expect(within(ancienne).getByText('JGtm').className).toContain('font-bold')
    expect(within(ancienne).getByText('Cobra01').className).not.toContain('font-bold')
    expect(within(recente).getByText('Alpha').className).not.toContain('font-bold')
    expect(within(recente).getByText('Cobra01').className).not.toContain('font-bold')
  })

  it('un autre joueur : en gras comme tueur sur une ligne et comme victime sur l’autre', () => {
    setup({ who: player('foe', 'Cobra01') })
    const [recente, ancienne] = rows()
    // 9 s : Cobra01 tue JGtm ; 5 s : Alpha tue Cobra01.
    expect(within(recente).getByText('Cobra01').className).toContain('font-bold')
    expect(within(recente).getByText('JGtm').className).not.toContain('font-bold')
    expect(within(ancienne).getByText('Cobra01').className).toContain('font-bold')
    expect(within(ancienne).getByText('Alpha').className).not.toContain('font-bold')
  })

  it('un joueur absent des deux lignes n’en met aucun en gras', () => {
    setup({ who: player('inconnu', 'Inconnu') })
    for (const nom of document.querySelectorAll('li span.truncate')) {
      expect(nom.className).not.toContain('font-bold')
    }
  })

  it('un joueur désigné par la clé de sa feuille (xuid de base d’un bot) est reconnu', () => {
    const bot: ReplayPlayer = {
      xuid: 'bot:Alpha', filmName: 'Alpha', team: 0, lives: [],
      board: { xuid: 'pal', gamertag: 'Alpha' } as ReplayPlayer['board'],
    }
    setup({ nowMs: 6_000, who: bot })
    expect(within(rows()[0]).getByText('Alpha').className).toContain('font-bold')
  })
})
