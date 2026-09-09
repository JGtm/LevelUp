/**
 * EquipmentUsageSection.test.tsx — l'orchestrateur du bloc « servi ou gâché » en
 * variante COMPTES (P9), monté par la Synthèse (mode 'solo') et l'Escouade (mode
 * 'squad'), PLAN_EQUIPEMENT_GACHIS_2026-09-09, étapes E5.8-E5.10/E6.2-E6.4.
 */
import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'

import type { EquipmentUsageBlock } from '@/lib/api/types'

import { EquipmentUsageSection } from './EquipmentUsageSection'
import { USAGE_TEXT } from './usageI18n'

const t = USAGE_TEXT.fr

describe('EquipmentUsageSection', () => {
  it('bloc absent : rien ne se rend (jamais un graphe fantôme)', () => {
    const { container } = render(
      <EquipmentUsageSection usage={undefined} mode="solo" t={t} locale="fr" />,
    )
    expect(container.firstChild).toBeNull()
  })

  it('available=false avec raison unsupported : rien ne se rend (titre sans decodeur)', () => {
    const usage: EquipmentUsageBlock = {
      available: false,
      unavailable_reason: 'unsupported',
      matches_measured: 0,
      matches_total: 0,
    }
    const { container } = render(<EquipmentUsageSection usage={usage} mode="solo" t={t} locale="fr" />)
    expect(container.firstChild).toBeNull()
  })

  it('available=false avec raison load_failed : etat vide avec la raison', () => {
    const usage: EquipmentUsageBlock = {
      available: false,
      unavailable_reason: 'load_failed',
      matches_measured: 0,
      matches_total: 12,
    }
    render(<EquipmentUsageSection usage={usage} mode="solo" t={t} locale="fr" />)
    expect(screen.getByText(t.unavailableLoadFailed)).toBeInTheDocument()
  })

  it('mode solo : une ligne par famille (equipement), triee du plus pris au moins pris', () => {
    const usage: EquipmentUsageBlock = {
      available: true,
      matches_measured: 8,
      matches_total: 8,
      families: [
        { family_key: 'sensor', taken: 40, used: 4, kept: 16, dropped: 20 },
        { family_key: 'wall', taken: 138, used: 79, kept: 5, dropped: 16 },
      ],
      players: [{ xuid: 'me', taken: 178, used: 83, kept: 21, dropped: 36, pad_pickups: 12 }],
    }
    render(<EquipmentUsageSection usage={usage} mode="solo" t={t} locale="fr" />)
    // "Mur de protection" (wall) doit apparaitre AVANT "Capteur de menaces" (sensor).
    const labels = screen.getAllByText(/Mur de protection|Capteur de menaces/)
    expect(labels[0]).toHaveTextContent('Mur de protection')
    expect(labels[1]).toHaveTextContent('Capteur de menaces')
    // Le bandeau "Matchs mesures" (une occurrence par carte)
    expect(screen.getAllByText(t.measuredFmt(8, 8)).length).toBe(2)
    // La barre "armes speciales" (pad_pickups) rend "Moi" pour le joueur de la route.
    expect(screen.getByText('Moi')).toBeInTheDocument()
    expect(screen.getByText('12 prises')).toBeInTheDocument()
  })

  it('mode squad : une ligne par coequipier (equipement ET armes), pas par famille', () => {
    const usage: EquipmentUsageBlock = {
      available: true,
      matches_measured: 42,
      matches_total: 42,
      tracked_players: [{ xuid: 'f1', gamertag: 'Madina' }],
      players: [
        { xuid: 'me', taken: 88, used: 55, kept: 9, dropped: 24, pad_pickups: 41 },
        { xuid: 'f1', taken: 74, used: 60, kept: 4, dropped: 10, pad_pickups: 31 },
      ],
    }
    render(<EquipmentUsageSection usage={usage} mode="squad" t={t} locale="fr" />)
    expect(screen.getAllByText('Moi').length).toBeGreaterThan(0)
    expect(screen.getAllByText('Madina').length).toBeGreaterThan(0)
    expect(screen.getByText('88 pris')).toBeInTheDocument()
    expect(screen.getByText('74 pris')).toBeInTheDocument()
    expect(screen.getByText('41 prises')).toBeInTheDocument()
    expect(screen.getByText('31 prises')).toBeInTheDocument()
  })

  it('donuts absents quand *_parties ne sont pas fournies (scope sans camp connu)', () => {
    const usage: EquipmentUsageBlock = {
      available: true,
      matches_measured: 3,
      matches_total: 3,
      families: [{ family_key: 'wall', taken: 10, used: 5, kept: 3, dropped: 2 }],
    }
    render(<EquipmentUsageSection usage={usage} mode="solo" t={t} locale="fr" />)
    // Pas de sous-total "Mon équipe" : le donut ne s'est pas rendu.
    expect(screen.queryByText(t.rowMyTeam)).not.toBeInTheDocument()
  })
})
