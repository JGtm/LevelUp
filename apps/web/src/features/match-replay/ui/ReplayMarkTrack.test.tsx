/**
 * Tests — ReplayMarkTrack (la piste de marques de la frise : kills, morts, médailles).
 *
 * DÉPLACÉS ICI DEPUIS `ReplayTimelineTracks.presence.test.tsx` LE 2026-09-09 (lot C4 de
 * `PLAN_RETOURS_VAGUE_C_FORMES_2026-09-08.md`, décision D7). Le dessin d'une médaille — un
 * anneau d'un pixel autour d'une marque de trois par huit, mesuré illisible en pratique — devient
 * un BADGE EN IMAGE du jeu (`ui/MedalBadges.tsx`, déjà employé par le fil des éliminations), et
 * ce composant en est désormais le seul foyer : le tester au travers de toute la frise
 * (`ReplayTimelineTracks`) n'apportait rien que ce montage direct ne couvre pas plus simplement.
 *
 * Ce qu'ils protègent :
 *  1. UN KILL MÉDAILLÉ GARDE SA SILHOUETTE NUE et reçoit le badge EN SURIMPRESSION, jamais un
 *     contour posé sur la marque elle-même — c'est tout le changement de ce lot.
 *  2. L'INFOBULLE DU BADGE PORTE LE TITRE ET LA DESCRIPTION (décision D7), pas seulement le nom :
 *     `MedalBadges` les compose déjà, ce composant ne fait que lui transmettre l'identité
 *     complète de la médaille au lieu d'un simple libellé.
 *  3. UNE MÉDAILLE D'OBJECTIF (`kind: 'medal'`, sans kill à proximité) N'A PLUS DE MARQUE NUE :
 *     son seul dessin est désormais le badge — la fusionner avec la marque de kill aurait
 *     réintroduit un second repère au même endroit (décision 9 du plan « frise, point de vue »,
 *     toujours valide).
 *  4. UNE MARQUE SANS MÉDAILLE N'A AUCUN BADGE : la décoration ne s'invite pas sans donnée.
 */
import { describe, expect, it } from 'vitest'
import { render } from '@testing-library/react'

import type { MedalEvent } from '../model/killFeedLogic'
import { ReplayMarkTrack } from './ReplayMarkTrack'
import type { TrackMark } from '../model/replayTimelineTracksLogic'

function medal(over: Partial<MedalEvent> = {}): MedalEvent {
  return {
    tMs: 1_000,
    xuid: 'me',
    gamertag: 'JGtm',
    name: 'Double Kill',
    label: 'Doublé',
    description: 'Deux éliminations en quatre secondes.',
    imageUrl: '/static/medals/double.png',
    ...over,
  }
}

function mark(over: Partial<TrackMark> = {}): TrackMark {
  return { key: 'm1', ratio: 0.5, kind: 'kill', clock: '2:30', medals: [], friend: false, ...over }
}

describe('ReplayMarkTrack — le badge de médaille (décision D7, 2026-09-09)', () => {
  it('un kill MÉDAILLÉ garde sa silhouette nue et reçoit le badge en image', () => {
    const { container, getByRole } = render(
      <ReplayMarkTrack
        marks={[mark({ key: 'k1', clock: '1:12', medals: [medal()] })]}
        height="h-[24px]"
        tall
      />,
    )
    // La silhouette du kill : aucun contour, aucune trace d'anneau.
    const silhouette = container.querySelector('[title="1:12 — Doublé"]')
    expect(silhouette).toBeTruthy()
    expect(silhouette?.className).not.toContain('ring-1')
    expect(silhouette?.className).not.toContain('ring-foreground')
    // Le badge, EN SURIMPRESSION : une image, pas un contour.
    const badge = getByRole('img', { name: 'Doublé' })
    expect(badge.tagName).toBe('IMG')
    expect(badge.getAttribute('src')).toBe('/static/medals/double.png')
  })

  it("l'infobulle du badge porte le TITRE ET LA DESCRIPTION, pas seulement le nom", () => {
    const { getByRole } = render(
      <ReplayMarkTrack
        marks={[mark({ medals: [medal({ label: 'Doublé', description: 'Deux éliminations en quatre secondes.' })] })]}
        height="h-[24px]"
        tall
      />,
    )
    const badge = getByRole('img', { name: 'Doublé' })
    expect(badge.getAttribute('title')).toBe('Doublé — Deux éliminations en quatre secondes.')
  })

  it('un kill ORDINAIRE (sans médaille) ne reçoit aucun badge', () => {
    const { container, queryByRole } = render(
      <ReplayMarkTrack marks={[mark({ key: 'k2', clock: '2:20' })]} height="h-[24px]" tall />,
    )
    expect(queryByRole('img')).toBeNull()
    expect(container.querySelector('[title="2:20"]')).toBeTruthy()
  })

  it("une médaille D'OBJECTIF (kind medal) n'a plus de marque nue : le badge est son seul dessin", () => {
    const { container, getByRole } = render(
      <ReplayMarkTrack
        marks={[mark({ key: 'obj', kind: 'medal', clock: '4:04', medals: [medal({ label: 'Capture' })] })]}
        height="h-[24px]"
        tall
      />,
    )
    // Aucune marque nue pour ce repère : ni titre isolé, ni contour à chercher.
    expect(container.querySelector('[title="4:04 — Capture"]')).toBeNull()
    expect(getByRole('img', { name: 'Capture' })).toBeTruthy()
  })

  it('une marque AMIE médaillée garde son losange ET reçoit le badge', () => {
    const { container, getByRole } = render(
      <ReplayMarkTrack
        marks={[mark({ key: 'ami', clock: '5:05', friend: true, medals: [medal({ label: 'Doublé' })] })]}
        height="h-[24px]"
        tall
      />,
    )
    const losange = container.querySelector('[title="5:05 — Doublé"]')
    expect(losange?.className).toContain('rotate-45')
    expect(losange?.className).not.toContain('ring-1')
    expect(getByRole('img', { name: 'Doublé' })).toBeTruthy()
  })

  it("la piste des COÉQUIPIERS (tall=false) ne dessine pas de badge, même si on lui en donne un", () => {
    // Cas de garde : `buildEventTracks` ne pose jamais de médaille sur cette piste en pratique
    // (cf. son en-tête), mais le composant ne doit pas en dépendre pour rester correct.
    const { queryByRole } = render(
      <ReplayMarkTrack marks={[mark({ medals: [medal()] })]} height="h-3.5" tall={false} />,
    )
    expect(queryByRole('img')).toBeNull()
  })
})

/**
 * C.1 DU PLAN DES ANTAGONISTES (2026-11-22, D-5 et DU-4) — LE SENS SE LIT À LA POSITION.
 *
 * Sur la rangée du joueur regardé (`tall`, vingt-quatre pixels), une élimination se pose dans la
 * moitié HAUTE et une mort dans la moitié BASSE : la position dit le sens sans l'aide de la
 * couleur. La rangée des coéquipiers (quatorze pixels) ne change pas.
 *
 * Le test lit la bande verticale dans les classes de la marque (`top-*` et `h-*`, échelle Tailwind
 * de quatre pixels par unité, ou valeur entre crochets) : jsdom ne calcule pas de feuille de style.
 * La boîte lue est celle de la marque avant rotation ; le losange tourné occupe sa diagonale, qui
 * tient aussi dans la moitié (6 px de côté, 8,5 px en travers, centré à 6 ou à 18).
 */
const TALL_PX = 24

function px(className: string, prefix: 'top' | 'h'): number {
  const nombre = String.raw`(\d+(?:\.\d+)?)`
  const m = new RegExp(String.raw`(?:^|\s)${prefix}-(?:\[${nombre}px\]|${nombre})(?:\s|$)`).exec(className)
  if (!m) throw new Error(`classe ${prefix}-* absente de « ${className} »`)
  return m[1] !== undefined ? Number(m[1]) : Number(m[2]) * 4
}

/** La bande verticale [haut, bas] d'une marque, en pixels depuis le haut de la rangée. */
function bande(el: Element | null): readonly [number, number] {
  const cls = el?.getAttribute('class') ?? ''
  const top = px(cls, 'top')
  return [top, top + px(cls, 'h')]
}

function enHaut([a, b]: readonly [number, number]): boolean {
  return a >= 0 && b <= TALL_PX / 2
}
function enBas([a, b]: readonly [number, number]): boolean {
  return a >= TALL_PX / 2 && b <= TALL_PX
}

describe('ReplayMarkTrack — élimination en haut, mort en bas (C.1, D-5)', () => {
  it('sur la rangée regardée, une élimination se pose dans la moitié haute', () => {
    const { container } = render(
      <ReplayMarkTrack marks={[mark({ key: 'k', kind: 'kill', clock: '1:00' })]} height="h-[24px]" tall />,
    )
    expect(enHaut(bande(container.querySelector('[title="1:00"]')))).toBe(true)
  })

  it('sur la rangée regardée, une mort se pose dans la moitié basse', () => {
    const { container } = render(
      <ReplayMarkTrack marks={[mark({ key: 'd', kind: 'death', clock: '1:10' })]} height="h-[24px]" tall />,
    )
    expect(enBas(bande(container.querySelector('[title="1:10"]')))).toBe(true)
  })

  it('un losange ami suit la même règle : haut pour une élimination, bas pour une mort', () => {
    const { container } = render(
      <ReplayMarkTrack
        marks={[
          mark({ key: 'ka', kind: 'kill', clock: '2:00', friend: true }),
          mark({ key: 'da', kind: 'death', clock: '2:10', friend: true }),
        ]}
        height="h-[24px]"
        tall
      />,
    )
    const haut = container.querySelector('[title="2:00"]')
    const bas = container.querySelector('[title="2:10"]')
    expect(haut?.className).toContain('rotate-45')
    expect(enHaut(bande(haut))).toBe(true)
    expect(enBas(bande(bas))).toBe(true)
  })

  it("l'élimination et la mort ne se recouvrent pas : deux bandes distinctes", () => {
    const { container } = render(
      <ReplayMarkTrack
        marks={[
          mark({ key: 'k', kind: 'kill', clock: '3:00' }),
          mark({ key: 'd', kind: 'death', clock: '3:00 ' }),
        ]}
        height="h-[24px]"
        tall
      />,
    )
    const [, finHaut] = bande(container.querySelector('[title="3:00"]'))
    const [debutBas] = bande(container.querySelector('[title="3:00 "]'))
    expect(finHaut).toBeLessThan(debutBas)
  })

  it("l'encre reste celle d'aujourd'hui : alliée pour une élimination, adverse pour une mort", () => {
    const { container } = render(
      <ReplayMarkTrack
        marks={[
          mark({ key: 'k', kind: 'kill', clock: '4:00' }),
          mark({ key: 'd', kind: 'death', clock: '4:10' }),
        ]}
        height="h-[24px]"
        tall
      />,
    )
    expect(container.querySelector<HTMLElement>('[title="4:00"]')?.style.background).toContain('--ac-team-ally')
    expect(container.querySelector<HTMLElement>('[title="4:10"]')?.style.background).toContain('--ac-team-enemy')
  })

  it('la rangée des COÉQUIPIERS ne change pas (DU-4) : mêmes bandes pour une élimination et une mort', () => {
    const { container } = render(
      <ReplayMarkTrack
        marks={[
          mark({ key: 'k', kind: 'kill', clock: '5:00' }),
          mark({ key: 'd', kind: 'death', clock: '5:10' }),
          mark({ key: 'f', kind: 'kill', clock: '5:20', friend: true }),
        ]}
        height="h-3.5"
        tall={false}
      />,
    )
    expect(container.querySelector('[title="5:00"]')?.className).toBe(
      'pointer-events-none absolute -translate-x-1/2 top-1 h-1.5 w-[2px] rounded-[2px] opacity-65',
    )
    expect(container.querySelector('[title="5:10"]')?.className).toBe(
      'pointer-events-none absolute -translate-x-1/2 top-1 h-1.5 w-[2px] rounded-[2px] opacity-65',
    )
    expect(container.querySelector('[title="5:20"]')?.className).toBe(
      'pointer-events-none absolute -translate-x-1/2 top-[4px] h-1.5 w-1.5 rotate-45 rounded-[2px] opacity-65',
    )
  })

  it("le badge de médaille reste à côté de sa marque d'élimination : même abscisse, et la marque tient dans la hauteur du badge", () => {
    const { container, getByRole } = render(
      <ReplayMarkTrack
        marks={[mark({ key: 'k', kind: 'kill', clock: '6:00', ratio: 0.4, medals: [medal({ label: 'Doublé' })] })]}
        height="h-[24px]"
        tall
      />,
    )
    const marque = container.querySelector<HTMLElement>('[title="6:00 — Doublé"]')
    const conteneur = getByRole('img', { name: 'Doublé' }).parentElement as HTMLElement
    expect(conteneur.style.left).toBe(marque?.style.left)
    // Badge de seize pixels, posé à `top-*` : il enveloppe la bande de la marque.
    const topBadge = px(conteneur.className, 'top')
    const [a, b] = bande(marque)
    expect(topBadge).toBeLessThanOrEqual(a)
    expect(topBadge + 16).toBeGreaterThanOrEqual(b)
    // Et il suit la marque vers le haut : son centre est dans la moitié haute de la rangée.
    expect(topBadge + 8).toBeLessThan(TALL_PX / 2)
  })
})
