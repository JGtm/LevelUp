/**
 * ReplayPlayerNow.test.tsx — LE BLOC « À L'INSTANT » DE LA FICHE DU JOUEUR.
 *
 * Ce qu'ils protègent :
 *  1. un joueur VIVANT et complet : titre et instant, bouclier et santé (piste et valeur en pour-cent du
 *     plein), l'arme en main nommée avec ses munitions, l'arme en réserve éteinte, les grenades, l'équipement
 *     avec ses charges ;
 *  2. un joueur MORT à l'instant lu : le titre, l'encadré « Éliminé » et son décompte — ni jauge, ni arme ;
 *  3. un joueur SANS lecture d'inventaire : seules les valeurs lues sont rendues, jamais un zéro ni un mot
 *     d'absence ;
 *  4. un BOT : la même fiche, par la clé de roster synthétique ;
 *  5. LE BLOC SUIT L'IMAGE LUE : deux images, deux rendus.
 */
import { describe, expect, it } from 'vitest'
import { render, screen, within } from '@testing-library/react'

import { buildPlayers, type ReplayPlayer } from '@/lib/replay/rosterLogic'
import { vitalityPresence } from '@/lib/replay/playerState'

import { testReplayDoc } from '../test/testDoc'
import type { ReadingAges } from '../hooks/useCardScene'
import { ReplayPlayerNow } from './ReplayPlayerNow'

const ICONE_FUSIL = '/static/weapons-assets/halo_infinite/jeu/contour-01.png'
const ICONE_FRAG = '/static/weapons-assets/halo_infinite/hud/Frag.png'
const ICONE_GRAPPIN = '/static/weapons-assets/halo_infinite/hud/Grapple.png'

/** Alpha vit sur [0, 100] : bouclier entamé à l'image 40, santé à 70 %. Echo meurt à 30 et revient à 130. */
function documentComplet() {
  return testReplayDoc({
    frameIntervalMs: 100,
    roster: [
      { xuid: 'A', filmIndex: 0, name: 'Alpha', team: 0 },
      { xuid: 'E', filmIndex: 1, name: 'Echo', team: 1 },
      { xuid: '', filmIndex: 2, name: 'Marc [bot]', bot: true, team: 1 },
    ],
    tracks: [
      {
        slot: 512, team: -1, xuid: 'A', startFrame: 0, endFrame: 100,
        points: [{ t: 0, x: 1, y: 1, sh: 1, hp: 1 }, { t: 40, x: 2, y: 2, sh: 0.6, hp: 0.7 }],
      },
      { slot: 513, team: -1, xuid: 'E', startFrame: 0, endFrame: 30, points: [{ t: 0, x: 3, y: 3, sh: 1, hp: 1 }] },
      { slot: 514, team: -1, xuid: 'E', startFrame: 130, endFrame: 200, points: [{ t: 130, x: 3, y: 3, sh: 1, hp: 1 }] },
      { slot: 515, team: -1, xuid: 'bot:Marc', startFrame: 0, endFrame: 100, points: [{ t: 0, x: 4, y: 4, sh: 0.5, hp: 1 }] },
    ],
    weaponLabels: {
      '0xAAAA': { fr: "Fusil d'assaut", en: 'Assault rifle', img: ICONE_FUSIL, tinted: true },
      '0xBBBB': { fr: 'SPNKr', en: 'SPNKr' },
    },
    loadouts: [
      { t: 0, slot: 512, w: ['0xAAAA', '0xBBBB'] },
      { t: 0, slot: 515, w: ['0xAAAA', '0xBBBB'] },
    ],
    grenadeLabels: [
      { fr: 'Fragmentation', en: 'Frag', img: ICONE_FRAG, tinted: true },
      { fr: 'Plasma', en: 'Plasma' },
    ],
    inventory: [
      { t: 0, slot: 512, d: 0, am: [{ mag: 36, res: 198 }, { mag: 2, res: 4 }], g: [2, 0] },
      // Alpha, plus tard : la main change et le chargeur se vide.
      { t: 60, slot: 512, d: 1, am: [{ mag: 10, res: 90 }, { mag: 1, res: 3 }], g: [1, 1], gs: 1 },
      { t: 0, slot: 515, d: 0, am: [{ mag: 8, res: 24 }, { mag: 3, res: 6 }], g: [0, 0] },
    ],
    abilityLabels: { '20': { fr: 'Grappin', en: 'Grappleshot', img: ICONE_GRAPPIN, tinted: true } },
    abilities: [{ t: 0, slot: 512, r: 20, src: 'i48' }],
    abilityCharges: [{ t: 30, slot: 512, family: 'grapple', charges: 1 }],
  })
}

const AGES = (doc: ReturnType<typeof documentComplet>): ReadingAges => ({
  presence: vitalityPresence(doc),
  vitalityFade: 60,
  readingFull: 200,
})

function joueur(doc: ReturnType<typeof documentComplet>, xuid: string): ReplayPlayer {
  const p = buildPlayers(doc, []).find((x) => x.xuid === xuid)
  if (!p) throw new Error(`joueur ${xuid} absent du document de test`)
  return p
}

function monter(xuid: string, frame: number, locale: 'fr' | 'en' = 'fr', doc = documentComplet()) {
  return render(
    <ReplayPlayerNow player={joueur(doc, xuid)} doc={doc} frame={frame} clock="3:51" ages={AGES(doc)} locale={locale} />,
  )
}

const bloc = () => screen.getByRole('region', { name: "À l'instant" })

describe('ReplayPlayerNow — joueur vivant et complet', () => {
  it('le titre porte l’instant lu', () => {
    monter('A', 50)
    expect(within(bloc()).getByText('3:51')).toBeTruthy()
    expect(bloc().textContent).toContain("À l'instant")
  })

  it('le bouclier puis la santé : piste et valeur en pour-cent du plein', () => {
    monter('A', 50)
    const jauges = within(bloc()).getAllByRole('meter')
    expect(jauges.map((j) => j.getAttribute('aria-label'))).toEqual(['Bouclier', 'Santé'])
    expect(jauges.map((j) => j.getAttribute('aria-valuenow'))).toEqual(['60', '70'])
    const lignes = jauges.map((j) => j.parentElement?.textContent)
    expect(lignes).toEqual(['Bouclier60', 'Santé70'])
  })

  it('l’arme en main est nommée avec ses munitions, la réserve suit avec son compte', () => {
    monter('A', 10)
    const texte = bloc().textContent ?? ''
    expect(texte).toContain("Fusil d'assaut")
    expect(texte).toContain('36/198')
    expect(texte).toContain('en main')
    expect(texte).toContain('SPNKr')
    expect(texte).toContain('2/4')
    expect(texte).toContain('en réserve')
    // La main d'abord, la réserve ensuite.
    expect(texte.indexOf("Fusil d'assaut")).toBeLessThan(texte.indexOf('SPNKr'))
  })

  it('l’arme en réserve est éteinte : l’encre atténuée, pas celle de la main', () => {
    monter('A', 10)
    const reserve = within(bloc()).getByText('SPNKr').closest('div') as HTMLElement
    const main = within(bloc()).getByText("Fusil d'assaut").closest('div') as HTMLElement
    expect(reserve.className).toContain('text-muted-foreground')
    expect(main.className).not.toContain('text-muted-foreground')
  })

  it('les grenades portées : nom et stock ; un type désigné éteint les autres', () => {
    const { unmount } = monter('A', 10)
    expect(within(bloc()).getByText('Fragmentation ×2')).toBeTruthy()
    expect(within(bloc()).queryByText(/Plasma/)).toBeNull()
    unmount()
    monter('A', 70)
    const frag = within(bloc()).getByText('Fragmentation ×1').closest('div') as HTMLElement
    const plasma = within(bloc()).getByText('Plasma ×1').closest('div') as HTMLElement
    // Le sélecteur lu désigne le plasma : la fragmentation s'éteint.
    expect(frag.className).toContain('text-muted-foreground')
    expect(plasma.className).not.toContain('text-muted-foreground')
  })

  it('l’équipement : son nom et ses charges restantes', () => {
    monter('A', 50)
    expect(within(bloc()).getByText('Grappin')).toBeTruthy()
    expect(within(bloc()).getByText('· 1 charge')).toBeTruthy()
  })

  it('en anglais, les mêmes rangées avec leurs libellés', () => {
    monter('A', 10, 'en')
    const region = screen.getByRole('region', { name: 'Right now' })
    expect(within(region).getByText('Shield')).toBeTruthy()
    expect(within(region).getByText('Health')).toBeTruthy()
    expect(region.textContent).toContain('Assault rifle')
    expect(region.textContent).toContain('in hand')
    expect(region.textContent).toContain('in reserve')
  })

  it('une lecture d’inventaire ancienne pâlit et dit son âge', () => {
    monter('A', 50)
    const ligne = within(bloc()).getByText("Fusil d'assaut").closest('span[title]') as HTMLElement
    expect(ligne.getAttribute('title')).toMatch(/Armes lues il y a 5,0 s|Armes lues il y a 5\.0 s/)
    expect(Number(ligne.style.opacity)).toBeLessThan(1)
  })
})

describe('ReplayPlayerNow — joueur mort à l’instant lu', () => {
  it('le titre et l’encadré « Éliminé » avec son décompte ; ni jauge, ni arme', () => {
    monter('E', 50)
    expect(bloc().textContent).toContain("À l'instant")
    expect(within(bloc()).getByText('Éliminé')).toBeTruthy()
    // Retour à l'image 130, lu à 50 : 8,0 s (pas de 100 ms).
    expect(within(bloc()).getByText('8.0 s')).toBeTruthy()
    expect(within(bloc()).queryAllByRole('meter')).toHaveLength(0)
    expect(bloc().textContent).not.toContain('en main')
  })
})

describe('ReplayPlayerNow — rien d’inventé', () => {
  it('un document sans inventaire ni capacité : seules les jauges lues sont rendues', () => {
    const doc = testReplayDoc({
      frameIntervalMs: 100,
      roster: [{ xuid: 'A', filmIndex: 0, name: 'Alpha', team: 0 }],
      tracks: [{ slot: 512, team: -1, xuid: 'A', startFrame: 0, endFrame: 100, points: [{ t: 0, x: 1, y: 1, sh: 0.4, hp: 1 }] }],
    })
    monter('A', 10, 'fr', doc)
    expect(within(bloc()).getAllByRole('meter')).toHaveLength(2)
    const texte = bloc().textContent ?? ''
    expect(texte).not.toMatch(/en main|en réserve|charge|×/)
    expect(texte).not.toMatch(/inconnu|non lu|indisponible/i)
  })

  it('un document sans vitalité publiée : ni piste ni valeur, jamais un plein inventé', () => {
    const doc = testReplayDoc({
      frameIntervalMs: 100,
      roster: [{ xuid: 'A', filmIndex: 0, name: 'Alpha', team: 0 }],
      tracks: [{ slot: 512, team: -1, xuid: 'A', startFrame: 0, endFrame: 100, points: [{ t: 0, x: 1, y: 1 }] }],
    })
    monter('A', 10, 'fr', doc)
    expect(within(bloc()).queryAllByRole('meter')).toHaveLength(0)
    expect(bloc().textContent).toBe("À l'instant3:51")
  })

  it('un joueur absent à l’image lue (aucune vie) : le titre seul', () => {
    const doc = documentComplet()
    const absent: ReplayPlayer = { xuid: 'Z', filmName: 'Zulu', team: 1, lives: [] }
    render(<ReplayPlayerNow player={absent} doc={doc} frame={50} clock="3:51" ages={AGES(doc)} locale="fr" />)
    expect(bloc().textContent).toBe("À l'instant3:51")
  })

  it('une arme sans nom au catalogue garde son identifiant, jamais le nom d’une autre', () => {
    const doc = testReplayDoc({
      frameIntervalMs: 100,
      roster: [{ xuid: 'A', filmIndex: 0, name: 'Alpha', team: 0 }],
      tracks: [{ slot: 512, team: -1, xuid: 'A', startFrame: 0, endFrame: 100, points: [{ t: 0, x: 1, y: 1 }] }],
      loadouts: [{ t: 0, slot: 512, w: ['0xCCCC'] }],
    })
    monter('A', 10, 'fr', doc)
    expect(within(bloc()).getByText('0xCCCC')).toBeTruthy()
    // Le sélecteur n'est pas lu : l'arme n'est dite ni en main ni en réserve.
    expect(bloc().textContent).not.toMatch(/en main|en réserve/)
  })
})

describe('ReplayPlayerNow — un bot', () => {
  it('la même fiche, par la clé de roster synthétique', () => {
    monter('bot:Marc', 10)
    expect(within(bloc()).getAllByRole('meter').map((j) => j.getAttribute('aria-valuenow'))).toEqual(['50', '100'])
    expect(bloc().textContent).toContain('8/24')
    expect(bloc().textContent).toContain('en main')
    // Aucun type de grenade porté (compteurs à zéro) : aucune rangée de grenade.
    expect(bloc().textContent).not.toContain('×')
  })
})

describe('ReplayPlayerNow — le bloc suit l’image lue', () => {
  it('deux images, deux rendus : la main, le chargeur, les grenades et le bouclier changent', () => {
    const doc = documentComplet()
    const joueurA = joueur(doc, 'A')
    const ages = AGES(doc)
    const { rerender } = render(<ReplayPlayerNow player={joueurA} doc={doc} frame={10} clock="0:01" ages={ages} locale="fr" />)
    const avant = bloc().textContent
    expect(avant).toContain('36/198')
    expect(avant).toContain('Fragmentation ×2')
    expect(within(bloc()).getAllByRole('meter')[0].getAttribute('aria-valuenow')).toBe('100')

    rerender(<ReplayPlayerNow player={joueurA} doc={doc} frame={70} clock="0:07" ages={ages} locale="fr" />)
    const apres = bloc().textContent
    expect(apres).not.toBe(avant)
    // L'instant écrit suit celui qu'on donne.
    expect(apres).toContain('0:07')
    // La main est désormais le SPNKr (sélecteur D = 1) et son chargeur est celui de l'emplacement 1.
    expect(apres).toContain('1/3')
    expect(apres).toContain('10/90')
    expect(apres).toContain('Fragmentation ×1')
    expect(within(bloc()).getAllByRole('meter')[0].getAttribute('aria-valuenow')).toBe('60')
    const main = within(bloc()).getByText('SPNKr').closest('div') as HTMLElement
    expect(main.className).not.toContain('text-muted-foreground')
  })
})
