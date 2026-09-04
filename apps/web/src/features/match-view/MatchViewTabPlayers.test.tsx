/**
 * Tests — MatchViewTabPlayers, la RANGÉE « Antagonistes | Assistances ».
 *
 * CE QU'ILS PROTÈGENT. Les deux graphes sont des miroirs (qui m'a tué / qui a aidé à tuer)
 * et se lisent côte à côte depuis le 2026-09-03. Mais « Assistances » ne rend RIEN quand le
 * match n'a aucune ligne de film — le cas de la quasi-totalité des matchs. Une grille à deux
 * colonnes laisserait alors les antagonistes sur une demi-largeur avec un vide à droite :
 * une CELLULE FANTÔME, qui se lit « il manque quelque chose ». Ces tests vérifient les deux
 * états de la rangée, pas seulement le nominal.
 *
 * Les feuilles lourdes (charts ECharts, tables) sont mockées : seule la MISE EN PAGE est
 * testée ici — même parti pris que MatchViewTabs.test.tsx.
 */
import { describe, expect, it, vi } from 'vitest'
import { render } from '@testing-library/react'

import { MATCH_VIEW_TEXT } from './i18n'
import { MatchViewTabPlayers } from './MatchViewTabPlayers'

import type { MatchAssistPairs, MatchViewHeader, MatchViewRank } from '@/lib/api/types'

vi.mock('./MatchNemesisCards', () => ({
  MatchNemesisCards: () => <div data-testid="nemesis" />,
}))
vi.mock('./MatchAntagonistChart', () => ({
  MatchAntagonistChart: () => <div data-testid="antagonistes" />,
}))
vi.mock('./MatchFragDiffChart', () => ({ MatchFragDiffChart: () => <div /> }))
vi.mock('./MatchScoreboard', () => ({ MatchScoreboard: () => <div /> }))
vi.mock('./MatchEncountersTable', () => ({ MatchEncountersTable: () => <div /> }))
// `MatchAssistChart` n'est PAS mocké : c'est sa porte 1 (bloc absent -> rien) que la rangée
// doit accompagner, et la mocker reviendrait à tester la mise en page contre une fiction.
vi.mock('@/components/charts/BarStackedChart', () => ({
  BarStackedChart: ({ title }: { title: string }) => <div data-testid="assistances">{title}</div>,
}))

const BLOC_ASSISTANCES = { pairs: [], measured_deaths: 12 } as unknown as MatchAssistPairs

function afficher(assistPairs: MatchAssistPairs | undefined) {
  return render(
    <MatchViewTabPlayers
      header={{} as MatchViewHeader}
      rank={{} as MatchViewRank}
      scoreboard={[]}
      roster={[]}
      nemesis={[]}
      killerVictim={[]}
      assistPairs={assistPairs}
      highlightEvents={[]}
      citations={[]}
      encounters={[]}
      meXUID="me"
      friendGamertags={[]}
      locale="fr"
      t={MATCH_VIEW_TEXT.fr}
    />,
  )
}

/** La rangée est le PARENT commun des deux graphes — ou l'absence de parent commun. */
function rangeeDe(vue: ReturnType<typeof afficher>): HTMLElement | null {
  return vue.getByTestId('antagonistes').parentElement
}

describe('MatchViewTabPlayers — Antagonistes et Assistances sur une rangée', () => {
  it('pose les deux graphes CÔTE À CÔTE quand le bloc d’assistances existe', () => {
    const vue = afficher(BLOC_ASSISTANCES)
    expect(vue.getByTestId('assistances')).toBeTruthy()
    const rangee = rangeeDe(vue)
    expect(rangee?.className).toContain('grid-cols-1')
    expect(rangee?.className).toContain('lg:grid-cols-2')
    // Les deux occupent bien la MÊME rangée.
    expect(rangee?.contains(vue.getByTestId('assistances'))).toBe(true)
  })

  it('SANS bloc d’assistances : aucune grille, donc aucune cellule fantôme à droite', () => {
    const vue = afficher(undefined)
    expect(vue.queryByTestId('assistances')).toBeNull()
    // Le parent des antagonistes est la colonne de la section, pas une grille à 2 colonnes.
    expect(rangeeDe(vue)?.className).not.toContain('lg:grid-cols-2')
  })

  it('« Némésis » reste AU-DESSUS, hors de la rangée, sur toute la largeur', () => {
    const vue = afficher(BLOC_ASSISTANCES)
    const nemesis = vue.getByTestId('nemesis')
    expect(rangeeDe(vue)?.contains(nemesis)).toBe(false)
  })
})
