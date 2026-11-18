/**
 * Tests — ReplayPlayerCard (tuile compacte de la rangée) : les EFFETS de la fiche.
 *
 * Un état d'équipement se dit par un effet dédié sur la couche en retrait, jamais en faisant fondre la
 * fiche entière : camouflage (verre trempé), surbouclier (encadré `legendary`), les deux se composent. Les
 * effets de ZONE disent quand le joueur se tient dans le disque d'un objet posé — champ de réparation, écran
 * occultant, capteur adverse — par les MÊMES portes que le calque de la carte (`equipmentZones.ts`) ; l'éclat
 * de TRANSLOCATION vient des passages mesurés (`translocations[]`). Une fiche morte ne porte jamais un effet
 * d'équipement ni de zone.
 *
 * Le montage est la rangée des équipes, qui exige deux camps : Alpha est inspecté (en (0,0)), Bravo tient le
 * camp d'en face, loin de toute pose.
 */
import { describe, expect, it } from 'vitest'
import { render } from '@testing-library/react'

import type { ReplayDocument, ReplayEquipmentPlacement } from '@/lib/api/types'
import { NO_ALLEGIANCE } from '@/lib/replay/filmAllegiance'

import { ReplayRowFromBoard } from '../test/replayRowFromBoard'
import { scoreboardRow } from '../test/scoreboardRow'
import { testReplayDoc } from '../test/testDoc'

/** Une vie vivante sur [0,100] pour le slot 512, rattachée au joueur A, en (0,0). */
const TRACK = { slot: 512, team: -1, xuid: 'A', startFrame: 0, endFrame: 100, points: [{ t: 0, x: 0, y: 0 }] }
const TRACK_B = { slot: 513, team: -1, xuid: 'B', startFrame: 0, endFrame: 100, points: [{ t: 0, x: 90, y: 90 }] }

function renderCard(over: Partial<ReplayDocument>, frame = 10) {
  const doc = testReplayDoc({
    roster: [
      { xuid: 'A', filmIndex: 0, seat: 0, name: 'Alpha', team: 0 },
      { xuid: 'B', filmIndex: 1, seat: 1, name: 'Bravo', team: 1 },
    ],
    tracks: [TRACK, TRACK_B],
    ...over,
  })
  return render(<ReplayRowFromBoard allegiance={NO_ALLEGIANCE} doc={doc} scoreboard={[]} frame={frame} locale="fr" />)
}

const cardOf = (view: ReturnType<typeof render>) => view.getByText('Alpha').parentElement?.parentElement as HTMLElement
// LES EFFETS VIVENT SUR LA COUCHE EN RETRAIT (« pas de padding, tout est bord à bord ») : c'est elle que ces
// tests inspectent, jamais la fiche.
const layerOf = (view: ReturnType<typeof render>) => cardOf(view).querySelector('.replay-card-fx') as HTMLElement

describe('ReplayPlayerCard — état actif équipement (camouflage, surbouclier)', () => {
  it('camouflage actif : VERRE TREMPÉ (flou + voile + reflets), JAMAIS une opacité réduite', () => {
    // Un état se dit par un effet dédié, jamais en faisant fondre la fiche entière. Le verre porte ses REFLETS
    // diagonaux en couche image — le fond passe en longhands.
    const view = renderCard({ equipmentEpisodes: [{ slot: 512, fam: 'camo', t0: 0, t1: 50, endRead: true }] })
    const layer = layerOf(view)
    expect(layer.style.backdropFilter).toContain('blur')
    expect(layer.style.backgroundColor).toContain('var(--card)')
    expect(layer.style.backgroundImage).toContain('linear-gradient')
    expect(cardOf(view).style.opacity).toBe('')
  })

  it('surbouclier actif : ENCADRÉ au token `legendary`, jamais `info` (réservé à la jauge de bouclier)', () => {
    const view = renderCard({ equipmentEpisodes: [{ slot: 512, fam: 'overshield', t0: 0, t1: 50, endRead: true }] })
    const layer = layerOf(view)
    expect(layer.style.boxShadow).toContain('var(--ac-legendary)')
    expect(layer.style.boxShadow).not.toContain('var(--ac-info)')
    // Le cadre n'a pas de fond propre : cf. la composition avec le verre ci-dessous.
    expect(layer.style.backgroundColor).toBe('')
  })

  it('les deux effets se COMPOSENT : les ombres s’accumulent, le fond reste celui du verre', () => {
    const view = renderCard({
      equipmentEpisodes: [
        { slot: 512, fam: 'camo', t0: 0, t1: 50, endRead: true },
        { slot: 512, fam: 'overshield', t0: 0, t1: 50, endRead: true },
      ],
    })
    const layer = layerOf(view)
    expect(layer.style.backdropFilter).toContain('blur')
    expect(layer.style.backgroundColor).toContain('var(--card)')
    expect(layer.style.boxShadow).toContain('var(--ac-legendary)')
    expect(layer.style.boxShadow).toContain('var(--border)')
  })

  it('hors de la fenêtre [t0,t1] mesurée : AUCUNE couche d’effets — la fiche reste normale', () => {
    const view = renderCard({ equipmentEpisodes: [{ slot: 512, fam: 'camo', t0: 60, t1: 90, endRead: true }] }, 10)
    expect(layerOf(view)).toBeNull()
  })

  it('une fiche MORTE ne porte jamais un effet d’équipement (un épisode se ferme à la mort au plus tard)', () => {
    const view = renderCard({ equipmentEpisodes: [{ slot: 512, fam: 'camo', t0: 0, t1: 99, endRead: false }] }, 140)
    // La couche ne porte que l'ÉCLAT du coup fatal (encore dans sa fenêtre à l'image 140) : aucun verre — la
    // mort vit sur la tuile, et un épisode qui n'est plus mesuré n'a pas à survivre à la vie.
    const layer = layerOf(view)
    expect(layer.className).toContain('replay-flash-death')
    expect(layer.style.backdropFilter).toBe('')
    expect(layer.style.backgroundImage).toBe('')
    expect(layer.style.boxShadow).toBe('')
  })
})

describe('ReplayPlayerCard — zones d’équipement et translocation', () => {
  const field = (over: Partial<ReplayEquipmentPlacement> = {}) => ({
    family: 'repair_field',
    id: '0x32d97758',
    owner: 9,
    origin: 'deployed',
    t0: 0,
    t1: 50,
    x: 1,
    y: 0,
    ...over,
  })

  it('champ de réparation sous le joueur : contour vert, croix flottantes, et la phrase', () => {
    const view = renderCard({ equipmentPlacements: [field()] })
    const card = cardOf(view)
    expect(layerOf(view).style.boxShadow).toContain('var(--ac-success)')
    expect(card.title).toContain('champ de réparation')
    expect(card.querySelectorAll('.replay-zone-cross')).toHaveLength(3)
  })

  it('objet LÂCHÉ : aucune zone — la porte est celle du calque, un lâcher n’agit pas', () => {
    const view = renderCard({ equipmentPlacements: [field({ origin: 'dropped' })] })
    expect(layerOf(view)).toBeNull()
    expect(cardOf(view).querySelector('.replay-zone-cross')).toBeNull()
  })

  it('écran occultant : NUAGE NOIR par-dessus les infos, voile sombre discret, flou, contour', () => {
    const view = renderCard({ equipmentPlacements: [field({ family: 'shroud_screen', id: '0x5eeb1a13', x: 4 })] })
    const card = cardOf(view)
    const layer = layerOf(view)
    expect(card.querySelector('.replay-zone-cloud')).toBeTruthy()
    expect(layer.style.backdropFilter).toContain('blur')
    expect(layer.style.backgroundColor).toContain('var(--replay-label-stroke)')
    expect(layer.style.boxShadow).toContain('var(--replay-label-stroke)')
    expect(card.title).toContain('écran occultant')
  })

  it('capteur ADVERSE : le contour « détecté » pulse sur l’incrustation ; même camp : rien', () => {
    const deuxCamps = (side: string) => {
      const doc = testReplayDoc({
        roster: [
          { xuid: 'A', filmIndex: 0, name: 'Alpha', team: 0 },
          { xuid: 'B', filmIndex: 1, name: 'Bravo', team: side === 't1' ? 1 : 0 },
          { xuid: 'C', filmIndex: 2, name: 'Charlie', team: 1 },
        ],
        tracks: [
          TRACK,
          { ...TRACK, slot: 513, xuid: 'B', points: [{ t: 0, x: 9, y: 9 }] },
          { ...TRACK, slot: 514, xuid: 'C', points: [{ t: 0, x: 90, y: 90 }] },
        ],
        equipmentPlacements: [field({ family: 'sensor', id: '0x72b63d69', owner: 513, x: 0, y: 0 })],
      })
      const board = [
        scoreboardRow('A', 'Alpha', 't0'),
        scoreboardRow('B', 'Bravo', side),
        scoreboardRow('C', 'Charlie', 't1'),
      ]
      return render(<ReplayRowFromBoard allegiance={NO_ALLEGIANCE} doc={doc} scoreboard={board} frame={10} locale="fr" />)
    }
    const adverse = deuxCamps('t1')
    const card = cardOf(adverse)
    expect(card.querySelector('.replay-zone-sensor')).toBeTruthy()
    expect(card.title).toContain('capteur de menaces adverse')
    // Bravo est HORS de portée (12,7 m > 4,25 m) : sa fiche ne porte rien.
    const bravo = adverse.getByText('Bravo').parentElement?.parentElement as HTMLElement
    expect(bravo.querySelector('.replay-zone-sensor')).toBeNull()
    adverse.unmount()
    expect(cardOf(deuxCamps('t0')).querySelector('.replay-zone-sensor')).toBeNull()
  })

  it('translocation : le FOURREAU sur la bordure à l’instant de l’ÉVÉNEMENT, à délai négatif', () => {
    // L'ÉCLAT VIENT DE `translocations[]` : l'événement 117 du film date l'usage lui-même. La piste ne porte
    // aucune discontinuité ici — c'est le point : le saut de 3,24 m mesuré sur le corpus n'en produirait aucune
    // de lisible non plus.
    const view = renderCard({ translocations: [{ slot: 512, t: 21 }] }, 22)
    const card = cardOf(view)
    const fourreau = card.querySelector('.replay-flash-translocation') as HTMLElement
    expect(fourreau).toBeTruthy()
    expect(fourreau.style.animationDelay.startsWith('-')).toBe(true)
    expect(card.title).toContain('translocateur')
  })

  it('une fiche MORTE ne porte ni zone ni croix — même posée en plein champ', () => {
    const view = renderCard({ equipmentPlacements: [field({ x: 0 })] }, 140)
    const card = cardOf(view)
    // La couche ne porte que l'éclat du coup fatal — jamais un cadre de zone (success) — et aucune
    // incrustation : ni croix, ni nuage, ni éclair.
    expect(layerOf(view).style.boxShadow).toBe('')
    expect(card.querySelector('.replay-zone-cross')).toBeNull()
    expect(card.querySelector('.replay-zone-cloud')).toBeNull()
    expect(card.querySelector('.replay-zone-bolt')).toBeNull()
  })
})
