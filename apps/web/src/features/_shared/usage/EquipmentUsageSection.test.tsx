/**
 * EquipmentUsageSection.test.tsx — l'orchestrateur du bloc « servi ou gâché » en
 * variante COMPTES (P9), monté par la Synthèse (mode 'solo') et l'Escouade (mode
 * 'squad'), PLAN_EQUIPEMENT_GACHIS_2026-09-09, étapes E5.8-E5.10/E6.2-E6.4.
 */
import { describe, expect, it } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'

import type { EquipmentUsageBlock } from '@/lib/api/types'

import { EquipmentUsageSection } from './EquipmentUsageSection'
import { USAGE_TEXT } from './usageI18n'

const t = USAGE_TEXT.fr

/**
 * Ouvre l'infobulle (i) du titre d'une carte. Depuis le 2026-09-21, aide de lecture, notes
 * de mesure et couverture y vivent ensemble : fermée, aucune de ces phrases n'est dans le
 * DOM — c'est exactement ce que l'ajustement voulait.
 */
function ouvrirAideDe(container: HTMLElement, titre: string) {
  const section = container.querySelector(`section[aria-label="${titre}"]`)
  const bouton = section?.querySelector('h3 button')
  if (bouton == null) throw new Error(`aucune infobulle (i) sur la carte « ${titre} »`)
  fireEvent.click(bouton)
}

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
    // La couverture ne s'ecrit NULLE PART sur ces deux rangees (2026-09-19) : ni dans les
    // bandeaux de titre (retiree le 2026-09-13), ni en pied de rangee. `measuredFmt` a
    // disparu du dictionnaire le 2026-09-21 : plus aucune carte ne l'ecrit.
    expect(screen.queryByText(t.measuredFooterFmt(8, 8))).not.toBeInTheDocument()
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

  it('sans *_parties : le donut equipement disparait, la carte des armes speciales RESTE (titree, etat vide)', () => {
    const usage: EquipmentUsageBlock = {
      available: true,
      matches_measured: 3,
      matches_total: 3,
      families: [{ family_key: 'wall', taken: 10, used: 5, kept: 3, dropped: 2 }],
    }
    render(<EquipmentUsageSection usage={usage} mode="solo" t={t} locale="fr" />)
    // Pas de sous-total "Mon équipe" : aucun donut ne s'est rendu.
    expect(screen.queryByText(t.rowMyTeam)).not.toBeInTheDocument()
    // La carte de part des armes speciales se rend TOUJOURS, avec son titre et son texte
    // d'absence — escamotee, la rangee se lisait comme un bug (2026-09-19).
    expect(screen.getAllByText(t.viewWeaponPartsSolo).length).toBeGreaterThan(0)
    // D8 (2026-09-21) : le bloc NOMME sa cause. Des films lus sans aucun socle, ce n'est
    // pas « aucun film » — c'est un mode qui n'allume aucun socle.
    expect(screen.getAllByText(t.emptyNoPads).length).toBeGreaterThan(0)
  })
})

// `measuredFooterFmt` a QUITTE les rangees equipement et armes speciales le 2026-09-19 ;
// il sert encore le pied de la rangee des NIVEAUX D'ARMES (`padTiersCoverage`), dont la
// couverture est propre et vient d'une autre passe. Son accord en nombre reste garde ici.
describe('couverture de mesure — accord en nombre (finitions 2026-09-13)', () => {
  it('un seul match : « match » au singulier, en FR comme en EN', () => {
    expect(USAGE_TEXT.fr.measuredFooterFmt(1, 1)).toBe('Mesuré sur 1 match sur 1')
    expect(USAGE_TEXT.en.measuredFooterFmt(1, 1)).toBe('Measured on 1 match out of 1')
  })

  it('plusieurs matchs : pluriel', () => {
    expect(USAGE_TEXT.fr.measuredFooterFmt(104, 1147)).toBe('Mesuré sur 104 matchs sur 1147')
    expect(USAGE_TEXT.en.measuredFooterFmt(104, 1147)).toBe('Measured on 104 matches out of 1147')
  })

  it('aucun match mesuré : singulier en FR (« 0 match »), pluriel en EN', () => {
    expect(USAGE_TEXT.fr.measuredFooterFmt(0, 12)).toBe('Mesuré sur 0 match sur 12')
    expect(USAGE_TEXT.en.measuredFooterFmt(0, 12)).toBe('Measured on 0 matches out of 12')
  })
})

/**
 * LA RANGÉE « CONTRÔLE DES ARMES PAR NIVEAU » (2026-09-14).
 *
 * Elle rend les MÊMES prises de socle rangées par niveau d'arme. Ce que ces tests verrouillent :
 * la rangée est ABSENTE sans bloc (« pas encore mesuré » ne se dessine pas comme « aucune
 * prise »), son ordre est celui, ÉCRIT, du contrat, et les notes de mesure ne s'écrivent que
 * quand il y a quelque chose à signaler.
 */
describe('EquipmentUsageSection — les niveaux d’armes', () => {
  /** Un bloc mesuré, avec ou sans niveaux. */
  function blocAvecNiveaux(padTiers?: EquipmentUsageBlock['pad_tiers']): EquipmentUsageBlock {
    return {
      available: true,
      matches_measured: 4,
      matches_total: 5,
      players: [{ xuid: 'moi', taken: 10, used: 5, kept: 3, dropped: 2, pad_pickups: 7 }],
      tracked_players: [],
      pad_tiers: padTiers,
    } as unknown as EquipmentUsageBlock
  }

  const niveaux = {
    matches_measured: 4,
    matches_with_pads: 3,
    matches_tiers_established: 3,
    matches_random_starts: 1,
    tiers: [
      {
        tier: 'puissance',
        player_total: 5,
        lobby_total: 20,
        weapons: [
          { family_key: '9d6aaed2', family_label: 'S7 Sniper', player_pickups: 5, lobby_pickups: 20 },
        ],
      },
      { tier: 'terrain', player_total: 2, lobby_total: 12, weapons: [] },
    ],
  } as unknown as NonNullable<EquipmentUsageBlock['pad_tiers']>

  it('n’affiche AUCUNE rangée quand le bloc des niveaux est absent', () => {
    render(<EquipmentUsageSection usage={blocAvecNiveaux()} mode="solo" t={t} locale="fr" />)
    expect(screen.queryByText(t.blockPadTiers)).not.toBeInTheDocument()
  })

  it('affiche une ligne par niveau, dans l’ordre écrit, et le total du niveau', () => {
    render(<EquipmentUsageSection usage={blocAvecNiveaux(niveaux)} mode="solo" t={t} locale="fr" />)
    expect(screen.getByText(t.blockPadTiers)).toBeInTheDocument()
    const terrain = screen.getByText(t.padTierLabels.terrain)
    const puissance = screen.getByText(t.padTierLabels.puissance)
    // La puissance PRÉCÈDE le terrain depuis le 2026-09-21 (D2) : l'ordre est celui de la
    // LECTURE — le plus lourd en tête —, jamais le volume mesuré de la session.
    expect(puissance.compareDocumentPosition(terrain)).toBe(Node.DOCUMENT_POSITION_FOLLOWING)
    // Un niveau que le bloc ne publie pas n'a pas de ligne à zéro.
    expect(screen.queryByText(t.padTierLabels.base)).not.toBeInTheDocument()
  })

  /**
   * LES ARMES DE BASE SONT REPLIÉES, FERMÉES PAR DÉFAUT (D2, 2026-09-21) : elles pèsent
   * l'essentiel du volume et écrasaient les deux niveaux qui disent le contrôle.
   */
  it('replie les armes de base derrière un dépliable fermé, déployable', () => {
    const avecBase = {
      ...niveaux,
      tiers: [
        ...(niveaux.tiers ?? []),
        { tier: 'base', player_total: 30, lobby_total: 90, weapons: [] },
      ],
    } as unknown as NonNullable<EquipmentUsageBlock['pad_tiers']>
    render(<EquipmentUsageSection usage={blocAvecNiveaux(avecBase)} mode="solo" t={t} locale="fr" />)
    const bouton = screen.getByRole('button', { name: t.padTierBaseToggleFmt(1) })
    expect(bouton).toHaveAttribute('aria-expanded', 'false')
    expect(screen.queryByText(t.padTierLabels.base)).not.toBeInTheDocument()
    fireEvent.click(bouton)
    expect(bouton).toHaveAttribute('aria-expanded', 'true')
    expect(screen.getByText(t.padTierLabels.base)).toBeInTheDocument()
  })

  it('porte les notes de mesure dans l’infobulle du titre, pas sous la grille', () => {
    const { container } = render(
      <EquipmentUsageSection usage={blocAvecNiveaux(niveaux)} mode="solo" t={t} locale="fr" />,
    )
    // UNE SEULE AIDE PAR CARTE, VISIBLE (2026-09-21) : fermée, aucune note dans le DOM.
    expect(screen.queryByText(t.padTierNoPadsFmt(1))).not.toBeInTheDocument()
    ouvrirAideDe(container, t.blockPadTiers)
    // 4 matchs mesurés, 3 avec socles : un match sans socle.
    expect(screen.getByText(t.padTierNoPadsFmt(1))).toBeInTheDocument()
    // 3 avec socles, 3 à niveaux établis : aucune carte hors référence, donc aucune note.
    expect(screen.queryByText(t.padTierUnmeasuredFmt(1))).not.toBeInTheDocument()
    expect(screen.getByText(t.padTierRandomStartsFmt(1))).toBeInTheDocument()
  })

  it('publie la rangée en anglais aussi', () => {
    render(
      <EquipmentUsageSection
        usage={blocAvecNiveaux(niveaux)}
        mode="squad"
        t={USAGE_TEXT.en}
        locale="en"
      />,
    )
    expect(screen.getByText(USAGE_TEXT.en.blockPadTiers)).toBeInTheDocument()
    expect(screen.getByText(USAGE_TEXT.en.padTierLabels.puissance)).toBeInTheDocument()
  })
})
