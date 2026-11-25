/**
 * Tests — le fil : les répliques des Spartans.
 *
 * Ce qu'ils protègent :
 *  1. UNE LIGNE PAR RÉPLIQUE affichable, avec le joueur, la phrase entre guillemets dans la langue de l'interface, et un
 *     haut-parleur seulement quand la voix existe dans cette langue ;
 *  2. LE FILTRE « Répliques » du menu du fil, avec son compte, absent quand le réglage « Répliques dans le fil » est éteint ;
 *  3. LES FAMILLES ET LA FRÉQUENCE filtrent la liste, et les joueurs masqués emportent leurs répliques ;
 *  4. UNE RÉPLIQUE SANS NOM, SANS JOUEUR OU HORS TABLE (un cri) n'est jamais affichée.
 */
import { fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

vi.mock('@/lib/accessibility', () => ({
  resolveToken: (token: string) => `var(${token})`,
  tokenCssVar: (token: string) => `var(--ac-${token})`,
}))

const hoisted = vi.hoisted(() => ({ reply: null as unknown }))

vi.mock('@/lib/api/client', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/lib/api/client')>()),
  api: { getStatic: vi.fn(async () => hoisted.reply) },
}))

import { filmAllegianceOf } from '@/lib/replay/filmAllegiance'

import { buildFeedEntries } from '../model/killFeedLogic'
import { FEED_FILTER_KEY, persistPreference } from '../settings/replayPreferences'
import { testReplayDoc } from '../test/testDoc'
import { ReplayKillFeed } from './ReplayKillFeed'
import type { ReplayPlayer } from '../../../lib/replay/rosterLogic'

const TABLE = {
  version: 1,
  sons: {
    '0000000a': { texte: { fr: 'Ennemi aperçu !', en: 'Enemy there.' }, audio: ['fr'] },
    '0000000b': { texte: { fr: 'Boucliers à zéro !', en: 'Shields down.' }, audio: [] },
    '0000000c': { texte: { fr: 'Bien joué !', en: 'Nice one.' }, audio: ['fr'] },
  },
}

const ROSTER = [
  { xuid: 'a', filmIndex: 0, name: 'Alpha', team: 0 },
  { xuid: 'b', filmIndex: 1, name: 'Bravo', team: 1 },
]
const META = new Map([
  ['a', { gamertag: 'Alpha' }],
  ['b', { gamertag: 'Bravo' }],
])

let slug = 0
function docWithChatter() {
  slug += 1
  return testReplayDoc({
    titleSlug: `chatter_feed_${slug}`,
    frameIntervalMs: 100,
    roster: ROSTER,
    chatter: [
      { t: 10, player: 'a', sound: '0000000a', name: 'seeingenemynearby' },
      { t: 20, player: 'b', sound: '0000000b', name: 'shieldsdepleted' },
      { t: 30, player: 'a', sound: '0000000c', name: 'praisesquadkill' },
      { t: 40, sound: '0000000a', name: 'seeingenemynearby' }, // sans joueur
      { t: 50, player: 'a', sound: '0000000a' }, // sans nom
      { t: 60, player: 'a', sound: '000000ff', name: 'landhard' }, // un cri : hors table
    ],
  })
}

function renderFeed(doc: ReturnType<typeof docWithChatter>, over: { locale?: 'fr' | 'en'; hidden?: readonly ReplayPlayer[] | null } = {}) {
  return render(
    <ReplayKillFeed
      entries={buildFeedEntries([], [], 0, null)}
      nowMs={100_000}
      playWindow={null}
      scoreboard={[]}
      xuidMeta={META}
      allegiance={filmAllegianceOf(doc, [], 'a')}
      locale={over.locale ?? 'fr'}
      doc={doc}
      hiddenPlayers={over.hidden ?? null}
    />,
  )
}

beforeEach(() => {
  hoisted.reply = TABLE
  localStorage.clear()
})
afterEach(() => localStorage.clear())

const rows = () => screen.queryAllByRole('listitem').filter((r) => r.getAttribute('data-feed-family') === 'replies')

describe('les répliques dans le fil', () => {
  it("affiche une ligne par réplique nommée, parlée et portée par un joueur, dans la langue de l'interface", async () => {
    renderFeed(docWithChatter())
    await waitFor(() => expect(rows()).toHaveLength(3))
    const texts = rows().map((r) => r.textContent ?? '')
    expect(texts.some((t) => t.includes('Alpha') && t.includes('« Ennemi aperçu ! »'))).toBe(true)
    expect(texts.some((t) => t.includes('Bravo') && t.includes('Boucliers à zéro'))).toBe(true)
    expect(screen.queryByText(/landhard|cri/i)).toBeNull()
  })

  it("en anglais, texte anglais, sans voix (seule la voix française est versée) donc sans haut-parleur", async () => {
    renderFeed(docWithChatter(), { locale: 'en' })
    await waitFor(() => expect(rows()).toHaveLength(3))
    const enemy = rows().find((r) => (r.textContent ?? '').includes('“Enemy there.”'))
    expect(enemy).toBeDefined()
    expect(within(enemy as HTMLElement).queryByRole('img')).toBeNull()
  })

  it('un haut-parleur sur la ligne dont la voix existe dans la langue, aucun sur les autres', async () => {
    renderFeed(docWithChatter())
    await waitFor(() => expect(rows()).toHaveLength(3))
    const withVoice = rows().filter((r) => within(r).queryByRole('img') !== null)
    expect(withVoice).toHaveLength(2) // Ennemi aperçu et Bien joué ; Boucliers n'a pas de voix versée
  })

  it('le menu propose « Répliques » avec son compte, et ce filtre ne montre que les répliques', async () => {
    renderFeed(docWithChatter())
    await waitFor(() => expect(rows()).toHaveLength(3))
    fireEvent.click(screen.getByRole('button', { name: /Filtrer le fil/ }))
    const item = within(screen.getByRole('menu')).getByRole('menuitemradio', { name: /Répliques/ })
    expect(item.textContent).toMatch(/3$/)
    fireEvent.click(item)
    expect(rows()).toHaveLength(3)
    expect(screen.getAllByRole('listitem')).toHaveLength(3)
  })

  it('« Répliques dans le fil » éteint : aucune ligne, et le menu ne propose plus le filtre', async () => {
    persistPreference('replay-show-chatter-feed', 'false')
    persistPreference(FEED_FILTER_KEY, 'replies')
    renderFeed(docWithChatter())
    await waitFor(() => expect(screen.getByRole('button', { name: /Filtrer le fil/ })).toBeTruthy())
    expect(rows()).toHaveLength(0)
    fireEvent.click(screen.getByRole('button', { name: /Filtrer le fil/ }))
    expect(within(screen.getByRole('menu')).queryByRole('menuitemradio', { name: /Répliques/ })).toBeNull()
  })

  it('une famille éteinte retire ses lignes', async () => {
    persistPreference('replay-chatter-family-shields', 'false')
    renderFeed(docWithChatter())
    await waitFor(() => expect(rows()).toHaveLength(2))
    expect(rows().some((r) => (r.textContent ?? '').includes('Boucliers'))).toBe(false)
  })

  it('« rares seulement » garde les phrases dites au plus deux fois', async () => {
    persistPreference('replay-chatter-frequency', 'rare')
    renderFeed(docWithChatter())
    await waitFor(() => expect(rows()).toHaveLength(3)) // chaque phrase n'est dite qu'une fois dans ce match
  })

  it('les joueurs masqués emportent leurs répliques du fil', async () => {
    const doc = docWithChatter()
    const hidden = [{ xuid: 'a', lives: [], board: undefined }] as unknown as ReplayPlayer[]
    renderFeed(doc, { hidden })
    await waitFor(() => expect(screen.getAllByRole('listitem').length).toBeGreaterThan(0))
    const owners = rows().map((r) => r.textContent ?? '')
    expect(owners).toHaveLength(1)
    expect(owners[0]).toContain('Bravo')
  })
})
