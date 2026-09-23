/**
 * Tests — ReplayTeams et la RÈGLE DES PLACES (lot M2.4, 2026-09-23).
 *
 * Ce que la colonne doit rendre d'une place, image par image, quand le document publie la
 * présence de ses occupants (schéma 69) : la fiche de l'occupant présent ; « pas encore apparu »
 * sous son nom quand il tient la place sans corps (Q21) ; la place VIDE, sans aucun nom, entre
 * un partant et son remplaçant (Q20). Un joueur parti n'est jamais affiché, et la colonne garde
 * une tuile par place — ni plus, ni moins.
 */
import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'

import { ReplayTeams } from './ReplayTeams'
import { REPLAY_TEXT } from '../i18n/i18n'
import { testReplayDoc } from '../test/testDoc'

/** Une vie du joueur `xuid` sur [debut, fin]. */
function vie(xuid: string, slot: number, debut: number, fin: number) {
  return { slot, team: 0, xuid, startFrame: debut, endFrame: fin, points: [{ t: debut, x: 0, y: 0 }] }
}

/**
 * Le document : la place 0 tenue par le Partant (certain jusqu'à 50, peut-être jusqu'à 59),
 * VIDE de 60 à 99, puis tenue par l'Arrivant dès 100 — qui n'apparaît qu'à 120. La place 1
 * est tenue d'un bout à l'autre par le Titulaire.
 */
function documentDesPlaces() {
  return testReplayDoc({
    frameCount: 200,
    frameIntervalMs: 100,
    originMs: 0,
    roster: [
      { xuid: 'P', filmIndex: 0, seat: 0, seatSource: 'lu', name: 'Partant', team: 0, presence: [{ from: 0, to: 50, toMax: 59 }] },
      { xuid: 'A', filmIndex: 9, seat: 0, seatSource: 'apparie', name: 'Arrivant', team: 0, presence: [{ from: 100, to: 199 }] },
      { xuid: 'T', filmIndex: 1, seat: 1, seatSource: 'lu', name: 'Titulaire', team: 0, presence: [{ from: 0, to: 199 }] },
    ],
    tracks: [vie('P', 512, 0, 50), vie('A', 530, 120, 199), vie('T', 513, 0, 199)],
  })
}

function colonne(frame: number, locale: 'fr' | 'en' = 'fr') {
  return render(<ReplayTeams doc={documentDesPlaces()} scoreboard={[]} frame={frame} locale={locale} />)
}

describe('ReplayTeams — la règle des places', () => {
  it('le partant tient sa fiche dans sa présence', () => {
    colonne(55)
    expect(screen.getByText('Partant')).toBeTruthy()
    expect(screen.queryByText(REPLAY_TEXT.fr.seatVacant)).toBeNull()
  })

  it('Q20 : entre le partant et son remplaçant, la place est VIDE — et le parti n’est plus nommé', () => {
    colonne(80)
    expect(screen.getByText(REPLAY_TEXT.fr.seatVacant)).toBeTruthy()
    expect(screen.getByTitle(REPLAY_TEXT.fr.seatVacantHint)).toBeTruthy()
    expect(screen.queryByText('Partant')).toBeNull()
    expect(screen.queryByText('Arrivant')).toBeNull()
    expect(screen.getByText('Titulaire')).toBeTruthy()
  })

  it('Q21 : le remplaçant tient la place avant sa première apparition, « pas encore apparu »', () => {
    colonne(110)
    expect(screen.getByText('Arrivant')).toBeTruthy()
    expect(screen.getByText(REPLAY_TEXT.fr.seatNotSpawned)).toBeTruthy()
    expect(screen.getByTitle(REPLAY_TEXT.fr.seatNotSpawnedHint)).toBeTruthy()
  })

  it('une fois apparu, sa fiche remplace la tuile d’attente', () => {
    colonne(150)
    expect(screen.getByText('Arrivant')).toBeTruthy()
    expect(screen.queryByText(REPLAY_TEXT.fr.seatNotSpawned)).toBeNull()
    expect(screen.queryByText('Partant')).toBeNull()
  })

  it('les deux tuiles parlent anglais en anglais', () => {
    const vue = colonne(80, 'en')
    expect(vue.getByText('Open slot')).toBeTruthy()
    vue.rerender(<ReplayTeams doc={documentDesPlaces()} scoreboard={[]} frame={110} locale="en" />)
    expect(vue.getByText('Not spawned yet')).toBeTruthy()
  })

  it('une tuile par place, à chaque image : jamais plus de tuiles que de places', () => {
    for (const frame of [0, 55, 80, 110, 150, 199]) {
      const vue = colonne(frame)
      // Les tuiles sont les ENFANTS DIRECTS du conteneur des places d'un camp (la colonne qui
      // défile) — le seul endroit où leur nombre se lit sans dépendre de leur contenu.
      const conteneurs = [...vue.container.querySelectorAll('.overflow-y-auto')]
      const tuiles = conteneurs.reduce((n, c) => n + c.children.length, 0)
      expect(conteneurs.length, `image ${frame}`).toBe(1)
      expect(tuiles, `image ${frame}`).toBe(2)
      vue.unmount()
    }
  })
})
