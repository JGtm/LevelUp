/**
 * Tests — LE TEMPS MORT SUR LA FICHE JOUEUR (`ReplayTeams`, rangée `DeadTimeRow`).
 *
 * FICHIER SÉPARÉ DE `ReplayTeams.test.tsx` LE 2026-08-24 (ronde 1b de la revue) : le fichier
 * de tests des fiches était déjà à 882 lignes pour un seuil de dépôt à 500, et ce lot allait
 * l'emmener à 982. On n'accroît pas une dette gelée. La découpe suit une frontière nette — une
 * SEULE rangée de la fiche, celle dont ce lot est responsable — et `ReplayTeams.test.tsx`
 * retrouve exactement sa taille d'avant-lot.
 *
 * CE QU'ILS TIENNENT, LES QUATRE CHOSES QUI DISTINGUENT CETTE RANGÉE DES AUTRES :
 *   1. c'est un TOTAL DE MATCH — il ne tique pas avec la lecture, contrairement à tout ce qui
 *      l'entoure sur la fiche ;
 *   2. il s'écrit `mm:ss`, « 00:00 » compris : mort zéro fois est une mesure ;
 *   3. il s'écrit d'un TIRET quand le film ne permet pas de conclure, et l'infobulle dit alors
 *      pourquoi — sans elle, un tiret se lirait comme une panne d'affichage ;
 *   4. la fiche compacte ne le porte pas.
 *
 * La règle du refus elle-même (quelle trace anonyme invalide quel trou) est éprouvée dans
 * `deadTimeLogic.test.ts` : ici on ne teste que ce que l'écran en FAIT.
 */
import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'

import type { ReplayDocument } from '@/lib/api/types'

import { ReplayTeams } from './ReplayTeams'
import { testReplayDoc } from './test/testDoc'

/**
 * Une vie vivante sur [0,100] pour le slot 512, rattachée au joueur A.
 *
 * Copie locale de la constante homonyme de `ReplayTeams.test.tsx` (deuxième et dernière copie,
 * règle « <= 2 » du CLAUDE.md) : l'exporter depuis un fichier de test pour l'importer dans un
 * autre créerait une dépendance entre suites, pire que huit lignes recopiées.
 */
const TRACK = {
  slot: 512,
  team: -1,
  xuid: 'A',
  startFrame: 0,
  endFrame: 100,
  points: [{ t: 0, x: 0, y: 0 }],
}

/** Deux vies séparées de 80 images ; une image = une seconde, donc 80 s de temps mort. */
const AVEC_TROU = {
  frameCount: 1000,
  frameIntervalMs: 1000,
  roster: [{ xuid: 'A', filmIndex: 0, name: 'Alpha' }],
  tracks: [TRACK, { ...TRACK, slot: 513, startFrame: 180, endFrame: 300 }],
}

/** Le document minimal des autres suites : une seule vie, donc aucun trou. */
function renderTeams(over: Partial<ReplayDocument>, frame = 10) {
  const doc = testReplayDoc({
    roster: [{ xuid: 'A', filmIndex: 0, name: 'Alpha' }],
    tracks: [TRACK],
    ...over,
  })
  return render(<ReplayTeams doc={doc} scoreboard={[]} frame={frame} locale="fr" />)
}

describe('ReplayTeams — temps mort cumulé', () => {
  it('écrit le cumul en mm:ss, minutes complétées', () => {
    const doc = testReplayDoc(AVEC_TROU)
    const view = render(<ReplayTeams doc={doc} scoreboard={[]} frame={10} locale="fr" />)
    expect(view.getByTitle('Temps mort').textContent).toContain('01:20')
  })

  it('ne TIQUE pas avec la lecture : c’est un total de match, pas une valeur à l’instant lu', () => {
    const doc = testReplayDoc(AVEC_TROU)
    const tot = render(<ReplayTeams doc={doc} scoreboard={[]} frame={10} locale="fr" />)
    expect(tot.getByTitle('Temps mort').textContent).toContain('01:20')
    tot.unmount()
    // Image 140 : le joueur est justement DANS son trou, et le total reste le même.
    const tard = render(<ReplayTeams doc={doc} scoreboard={[]} frame={140} locale="fr" />)
    expect(tard.getByTitle('Temps mort').textContent).toContain('01:20')
  })

  it('un joueur jamais mort affiche 00:00 — une mesure, pas une lacune', () => {
    renderTeams({})
    expect(screen.getByTitle('Temps mort').textContent).toContain('00:00')
  })

  it('EN : « Time dead » porte la même valeur', () => {
    const doc = testReplayDoc(AVEC_TROU)
    const view = render(<ReplayTeams doc={doc} scoreboard={[]} frame={10} locale="en" />)
    expect(view.getByTitle('Time dead').textContent).toContain('01:20')
    expect(view.queryByTitle('Temps mort')).toBeNull()
  })

  it('la fiche COMPACTE ne le porte pas : elle n’a pas de rangée libre', () => {
    const doc = testReplayDoc(AVEC_TROU)
    const view = render(
      <ReplayTeams doc={doc} scoreboard={[]} frame={10} locale="fr" compact />,
    )
    expect(view.queryByTitle('Temps mort')).toBeNull()
    expect(view.queryByTitle(/Non mesurable/)).toBeNull()
  })
})

/**
 * LA MESURE REFUSÉE À L'ÉCRAN (revue adversariale du 24/08, constats 1 et 2) : une vie que le
 * film ne rattache à personne vit dans le trou du joueur. La fiche écrit un tiret et DIT
 * pourquoi — sans l'infobulle, le tiret se lirait comme une panne d'affichage.
 */
describe('ReplayTeams — temps mort non mesurable', () => {
  it('vie non rattachée CONTENUE dans le trou : un TIRET, et l’infobulle en donne la raison', () => {
    const doc = testReplayDoc({
      ...AVEC_TROU,
      tracks: [
        TRACK,
        { ...TRACK, slot: 900, xuid: undefined, startFrame: 120, endFrame: 160 },
        { ...TRACK, slot: 513, startFrame: 180, endFrame: 300 },
      ],
    })
    const view = render(<ReplayTeams doc={doc} scoreboard={[]} frame={10} locale="fr" />)
    const ligne = view.getByTitle(/^Non mesurable/)
    expect(ligne.textContent).toContain('—')
    expect(ligne.textContent).not.toContain('01:20')
    // La rangée existe toujours : le refus ne fait pas sauter la fiche.
    expect(ligne.textContent).toContain('Temps mort')
  })

  it('caméra de fin de match (trace qui DÉBORDE du trou) : la fiche garde sa mesure', () => {
    // Le pendant à l'écran de la preuve d'exclusion : une trace qui court encore après le
    // retour du joueur ne peut pas être sa vie manquante. Sans cet affinement, la rangée
    // affichait « — » quatre fois sur cinq sur les artefacts servis.
    const doc = testReplayDoc({
      ...AVEC_TROU,
      tracks: [
        TRACK,
        { ...TRACK, slot: 900, xuid: undefined, startFrame: 120, endFrame: 999 },
        { ...TRACK, slot: 513, startFrame: 180, endFrame: 300 },
      ],
    })
    const view = render(<ReplayTeams doc={doc} scoreboard={[]} frame={10} locale="fr" />)
    expect(view.getByTitle('Temps mort').textContent).toContain('01:20')
    expect(view.queryByTitle(/^Non mesurable/)).toBeNull()
  })

  it('joueur du roster sans aucune vie : le même tiret, jamais 00:00', () => {
    const doc = testReplayDoc({
      ...AVEC_TROU,
      roster: [
        { xuid: 'A', filmIndex: 0, name: 'Alpha' },
        { xuid: 'Z', filmIndex: 1, name: 'Zoulou' },
      ],
    })
    const view = render(<ReplayTeams doc={doc} scoreboard={[]} frame={10} locale="fr" />)
    // Alpha garde sa mesure ; Zoulou, que le film n'a jamais situé, affiche le refus.
    expect(view.getByTitle('Temps mort').textContent).toContain('01:20')
    expect(view.getByTitle(/^Non mesurable/).textContent).toContain('—')
  })

  it('EN : le refus parle anglais lui aussi', () => {
    const doc = testReplayDoc({
      ...AVEC_TROU,
      roster: [{ xuid: 'Z', filmIndex: 1, name: 'Zoulou' }],
      tracks: [],
    })
    const view = render(<ReplayTeams doc={doc} scoreboard={[]} frame={10} locale="en" />)
    expect(view.getByTitle(/^Not measurable/).textContent).toContain('—')
  })
})
