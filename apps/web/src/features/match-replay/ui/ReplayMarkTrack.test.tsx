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
    teamID: 0,
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
