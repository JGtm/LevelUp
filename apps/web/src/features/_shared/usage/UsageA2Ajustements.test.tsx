/**
 * UsageA2Ajustements.test.tsx — LES AJUSTEMENTS DU LOT A2 (2026-09-21) sur les formes
 * partagées du bloc « usages d'équipement, armes spéciales et objectifs ».
 *
 * Ce que ces cas verrouillent, et pourquoi chacun compte :
 *
 *   - D7 — LA LÉGENDE DE TEXTURE EST SOUS CHAQUE FORME qui porte la texture. Elle vivait
 *     dans une phrase de l'infobulle d'en-tête, donc nulle part : le lecteur voyait deux
 *     colonnes de jauge sur trois rayées sans savoir ce que la rayure disait. Un
 *     `UsageHatchLegend` retiré de `UsageGaugeGrid` ou de `UsageLobbyTrack` fait rougir ;
 *   - ÉTIQUETTES BLANCHES ET SEGMENTS DE MÊME ÉPAISSEUR sur la piste du lobby. La hachure
 *     de « eux » portait `opacity: .45` SUR le segment : elle délavait l'étiquette et, sans
 *     assise, le segment se lisait plus mince que ses voisins pleins. Ce cas fixe les deux
 *     — même classe de hauteur, même `text-white`, sur segments pleins ET hachés ;
 *   - D2 — LE DÉPLIABLE DES ARMES DE BASE EST FERMÉ PAR DÉFAUT, et `aria-expanded` le dit ;
 *   - D8 — CHAQUE CAUSE D'ÉTAT VIDE A SA PHRASE : « aucune donnée » n'en distinguait aucune.
 */
import { describe, expect, it } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'

import { UsageEmptyNotice } from './UsageEmptyNotice'
import { UsageGaugeGrid } from './UsageForms'
import { UsageLobbyTrack } from './UsageLobbyTrack'
import { buildGaugeRow } from './usageGaugeModel'
import type { UsageTrackSegment } from './usageLobbyTrackModel'
import { USAGE_TEXT } from './usageI18n'
import type { UsageEmptyReason } from './usageAvailability'

const t = USAGE_TEXT.fr

const shares = {
  player_total: 9,
  team_total: 20,
  lobby_total: 43,
  team_share_of_lobby_pct: 45.6,
  player_share_of_team_pct: 20.5,
  player_share_of_lobby_pct: 9.3,
}

function ligne(key: string, label: string) {
  return buildGaugeRow({
    key,
    label,
    shares,
    teamParityPct: 25,
    lobbyParityPct: 12.5,
    teamOfLobbyParityPct: 50,
    t,
    locale: 'fr',
  })
}

describe('UsageHatchLegend — D7, la légende est sous la forme, pas dans une infobulle', () => {
  it('la grille de jauges pose sa légende plein / hachuré (rapporté au lobby)', () => {
    render(<UsageGaugeGrid rows={[ligne('camo', 'Camouflage')]} t={t} />)
    expect(screen.getByText(t.hatchLegendSolid)).toBeInTheDocument()
    expect(screen.getByText(t.hatchLegendLobby)).toBeInTheDocument()
    // La piste, elle, légende « eux » — pas la grille.
    expect(screen.queryByText(t.hatchLegendEnemy)).not.toBeInTheDocument()
  })

  it('la piste du lobby pose sa légende nous / hachure neutre = adversaires', () => {
    render(<UsageLobbyTrack segments={segments()} label={t.viewLobbyTrack} t={t} />)
    expect(screen.getByText(t.hatchLegendSolid)).toBeInTheDocument()
    expect(screen.getByText(t.hatchLegendEnemy)).toBeInTheDocument()
  })
})

/** Trois segments assez lourds pour porter chacun son étiquette (seuil : 12 %). */
function segments(): UsageTrackSegment[] {
  return [
    { key: 'me', label: 'moi', kind: 'me', count: 9, pctText: '20,9 %', tooltip: 'moi : 9' },
    {
      key: 'rest',
      label: t.segTeamRest,
      kind: 'team-rest',
      count: 11,
      pctText: '25,6 %',
      tooltip: 'reste : 11',
    },
    {
      key: 'enemy',
      label: t.segEnemy,
      kind: 'enemy',
      count: 23,
      pctText: '53,5 %',
      tooltip: 'eux : 23',
    },
  ]
}

describe('UsageLobbyTrack — étiquettes blanches et segments de même épaisseur', () => {
  it('TOUTES les étiquettes sont blanches, y compris sur le segment haché', () => {
    const { container } = render(
      <UsageLobbyTrack segments={segments()} label={t.viewLobbyTrack} t={t} />,
    )
    const cellules = Array.from(container.querySelectorAll('[role="img"][tabindex="0"]'))
    expect(cellules).toHaveLength(3)
    for (const cellule of cellules) {
      expect(cellule.className).toContain('text-white')
      // Et l'étiquette est écrite : un segment à 53 % qui n'écrirait rien serait le
      // symptôme d'un seuil cassé.
      expect(cellule.textContent).not.toBe('')
    }
  })

  it('le segment haché a la MÊME hauteur que les pleins (aucune opacité sur le segment)', () => {
    const { container } = render(
      <UsageLobbyTrack segments={segments()} label={t.viewLobbyTrack} t={t} />,
    )
    const cellules = Array.from(
      container.querySelectorAll<HTMLElement>('[role="img"][tabindex="0"]'),
    )
    const hauteurs = new Set(cellules.map((c) => c.className.match(/h-\S+/)?.[0]))
    expect(hauteurs).toEqual(new Set(['h-full']))
    // L'opacité qui délavait l'étiquette et amincissait le segment n'est plus posée.
    for (const cellule of cellules) expect(cellule.style.opacity).toBe('')
  })
})

describe('UsageGaugeGrid — D2, le dépliable des armes de base', () => {
  it('est FERMÉ par défaut, et le dit par aria-expanded', () => {
    render(
      <UsageGaugeGrid
        rows={[ligne('tier-puissance', t.padTierLabels.puissance)]}
        collapsedRows={[ligne('tier-base', t.padTierLabels.base)]}
        collapsedLabel={t.padTierBaseToggleFmt(1)}
        t={t}
      />,
    )
    const bouton = screen.getByRole('button', { name: t.padTierBaseToggleFmt(1) })
    expect(bouton).toHaveAttribute('aria-expanded', 'false')
    expect(screen.queryByText(t.padTierLabels.base)).not.toBeInTheDocument()
    fireEvent.click(bouton)
    expect(bouton).toHaveAttribute('aria-expanded', 'true')
    expect(screen.getByText(t.padTierLabels.base)).toBeInTheDocument()
  })
})

describe('UsageEmptyNotice — D8, une phrase par cause', () => {
  const attendu: Record<UsageEmptyReason, string> = {
    'no-film': t.emptyNoFilm,
    'no-pads': t.emptyNoPads,
    'no-objectives': t.emptyNoObjectives,
    'load-failed': t.unavailableLoadFailed,
  }

  it('nomme la cause, et les quatre phrases sont distinctes', () => {
    for (const [reason, phrase] of Object.entries(attendu)) {
      const { unmount } = render(
        <UsageEmptyNotice reason={reason as UsageEmptyReason} t={t} />,
      )
      expect(screen.getByText(phrase)).toBeInTheDocument()
      unmount()
    }
    expect(new Set(Object.values(attendu)).size).toBe(4)
  })

  it('porte sa cause en attribut, pour que la capture dise laquelle', () => {
    const { container } = render(<UsageEmptyNotice reason="no-pads" t={t} />)
    expect(container.querySelector('[data-usage-empty="no-pads"]')).not.toBeNull()
  })
})
