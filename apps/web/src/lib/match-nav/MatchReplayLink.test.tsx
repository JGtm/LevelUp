/**
 * MatchReplayLink.test.tsx — LA PORTE DE TITRE DU COMPOSANT PARTAGÉ, testée pour elle-même.
 *
 * Ce composant est le point unique de l'icône « Rejeu » : Explorer, Synergies escouade et
 * carte de match. Il porte DEUX portes — la capability `replay` du titre, et `available`
 * (`has_replay`) de la ligne. Jusqu'au 2026-09-06, aucune n'était couverte ici : les tests
 * des deux tableaux assertaient l'absence du LIEN, que ce composant masque déjà tout seul,
 * si bien que les portes se couvraient mutuellement (revue C-R1, constat C2 : la mutation
 * qui retirait `!titreARejeu` laissait 388 tests verts).
 *
 * La carte de match n'a, elle, AUCUNE colonne à masquer : sans cette porte-ci, elle
 * afficherait un lien vers une page de rejeu que le titre ne sert pas (503).
 */
import { describe, expect, it, vi } from 'vitest'
import { screen } from '@testing-library/react'

// TanStack Router : <Link> exige un RouterProvider. Même stub que les tests de tableaux —
// on veut la route ciblée, pas le rendu du routeur.
vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  type LinkStubProps = {
    children?: React.ReactNode
    to: string
    params?: Record<string, string>
  } & React.AnchorHTMLAttributes<HTMLAnchorElement>
  return {
    ...actual,
    Link: ({ children, to, params, ...rest }: LinkStubProps) => {
      let href = to
      for (const [key, value] of Object.entries(params ?? {})) {
        href = href.replace(`$${key}`, value)
      }
      return (
        <a href={href} {...rest}>
          {children}
        </a>
      )
    },
  }
})

import { renderWithProviders } from '@/test/render-utils'
import { useAppShellStore } from '@/stores/appShellStore'

import { MatchReplayLink } from './MatchReplayLink'

const LABEL = 'Ouvrir le rejeu 2D du match'

function setTitleCaps(caps: string[]) {
  useAppShellStore.setState({
    currentTitleSlug: 'titre_test',
    availableTitles: [
      {
        slug: 'titre_test', name: 'Test', status: 'active', capabilities: caps, is_default: true,
        effective_hp_to_kill: 225, provides_damage_taken: true, provides_team_mmr: true,
        provides_max_killing_spree: true, offensive_conversion_p80: 0.9,
        defensive_resistance_p80: 1.65,
      },
    ],
  })
}

function rendre(available: boolean) {
  renderWithProviders(
    <MatchReplayLink available={available} matchId="match-1" playerSlug="me" label={LABEL} />,
  )
}

describe('MatchReplayLink — les deux portes, chacune pour elle-même', () => {
  it('titre AVEC `replay` et ligne avec artefact : le lien est rendu', () => {
    setTitleCaps(['replay'])
    rendre(true)
    expect(screen.getByRole('link', { name: LABEL }).getAttribute('href')).toContain(
      '/matches/match-1/replay',
    )
  })

  // PORTE DE TITRE seule : l'artefact existe pour CE match, mais le titre n'a pas de page de
  // rejeu (ses routes /replay* rendent 503). Un lien y mènerait à une panne.
  it('titre SANS `replay` : rien, même avec un artefact sur la ligne', () => {
    setTitleCaps(['ranked', 'waypoint_match_url'])
    rendre(true)
    expect(screen.queryByRole('link', { name: LABEL })).not.toBeInTheDocument()
  })

  // PORTE DE LIGNE seule : le titre sert le rejeu, mais pas pour ce match-là (404).
  it('titre AVEC `replay` mais ligne sans artefact : rien', () => {
    setTitleCaps(['replay'])
    rendre(false)
    expect(screen.queryByRole('link', { name: LABEL })).not.toBeInTheDocument()
  })
})
