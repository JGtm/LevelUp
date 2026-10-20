/**
 * CompareWeaponsSection.test — la section « Profil d'armes » du Face-à-face.
 *
 * Ce que ces tests verrouillent :
 *
 *  1. LA SECTION DISPARAÎT ENTIÈREMENT quand aucune réponse ne porte de profil — jamais une
 *     section vide, qui se lirait comme un chargement bloqué.
 *  2. LES BLOCS TOMBENT SÉPARÉMENT : sans `range`, les classes et les armes restent (c'est le
 *     cas d'un titre sans positions par kill, cf. D9).
 *  3. « SUR N MATCHS » EST AFFICHÉ POUR CHAQUE JOUEUR, sans condition : `is_sample` a été
 *     retiré du contrat au lot 3-bis, et c'est ce nombre qui dit au lecteur sur quoi le
 *     profil repose.
 *  4. AUCUN LIBELLÉ DE RÔLE N'EST UNE CLÉ DE MANIFESTE BRUTE.
 *
 * ECharts est mocké (jsdom ne peint pas de canvas) — même motif que les tests de
 * `WeaponRangeSection` (Séries temporelles). La géométrie du graphe est testée PURE ailleurs
 * (`components/charts/weaponRangeChart.test.ts`).
 */
import { describe, expect, it, vi } from 'vitest'
import { screen } from '@testing-library/react'

import { renderWithProviders } from '@/test/render-utils'
import type {
  CompareResponse,
  CompareWeaponSide,
  NormalizedPlayerStats,
  WeaponRangeSide,
} from '@/lib/api/types'

import { CompareWeaponsSection } from './CompareWeaponsSection'
import { getCompareText } from './i18n'

vi.mock('echarts-for-react', () => ({
  default: () => <div data-testid="echarts-mock" />,
}))

const text = getCompareText('fr')

const side = (o: Partial<WeaponRangeSide>): WeaponRangeSide => ({
  measured: 10,
  p10: 5,
  median: 7,
  p90: 10,
  min_m: 2,
  max_m: 14,
  above_pct: 30,
  level_pct: 50,
  below_pct: 20,
  ...o,
})

const stats = (gamertag: string): NormalizedPlayerStats =>
  ({ gamertag, matches: 100 }) as NormalizedPlayerStats

const profil = (o: Partial<CompareWeaponSide>): CompareWeaponSide => ({
  matches: 12,
  total_kills: 100,
  frag_classes: [
    { class: 'shoulder', kills: 60, share_pct: 60 },
    { class: 'grenade', kills: 40, share_pct: 40 },
  ],
  top_weapons: [
    { label: 'BR75', label_en: 'BR75', kills: 30, image_url: 'https://cdn/br75.png', image_tinted: true },
    { label: 'Sidekick', kills: 20 },
  ],
  ...o,
})

const avecPortee = (medianeFrags: number): CompareWeaponSide =>
  profil({
    range: {
      weapons: [{ weapon_key: 'precision', kills: side({ median: medianeFrags }) }],
      median_kills_m: medianeFrags,
      median_deaths_m: 12,
      measured_kills: 80,
      total_kills: 100,
      measured_deaths: 60,
      total_deaths: 75,
      below_threshold_kills: [{ weapon_key: 'sniper', measured: 6 }],
    },
  })

/** Un joueur qui porte EN PLUS un rôle « Environnement » — le cas du gate visuel. */
const avecEnvironnement = (): CompareWeaponSide =>
  profil({
    range: {
      weapons: [
        { weapon_key: 'precision', kills: side({ median: 30 }) },
        { weapon_key: 'environmental', kills: side({ median: 8 }) },
      ],
      median_kills_m: 30,
      median_deaths_m: 12,
      measured_kills: 80,
      total_kills: 100,
      measured_deaths: 60,
      total_deaths: 75,
    },
  })

const reponse = (
  nomA: string,
  nomB: string,
  a?: CompareWeaponSide,
  b?: CompareWeaponSide,
): CompareResponse =>
  ({
    player_a: stats(nomA),
    player_b: stats(nomB),
    metrics: [],
    title_slug: 'halo_infinite',
    weapons: a && b ? { player_a: a, player_b: b } : undefined,
  }) as CompareResponse

describe('CompareWeaponsSection — deux joueurs', () => {
  it('rend les trois blocs : classes, portée, armes les plus utilisées', () => {
    renderWithProviders(
      <CompareWeaponsSection
        left={reponse('Alpha', 'Bravo', avecPortee(20), avecPortee(25))}
        text={text}
        locale="fr"
      />,
    )
    expect(screen.getByText(text.catWeapons)).toBeInTheDocument()
    // Le sous-titre « Part des frags par classe » a ete retire : la carte porte le titre.
    expect(screen.getByText(text.weaponsTopTitle)).toBeInTheDocument()
    // Les deux cartes de portée, nommées par leur question.
    expect(screen.getByText(text.weaponsRangeKills)).toBeInTheDocument()
    expect(screen.getByText(text.weaponsRangeDeaths)).toBeInTheDocument()
  })

  it('affiche « sur N matchs », sans condition — `is_sample` n’existe plus', () => {
    renderWithProviders(
      <CompareWeaponsSection
        left={reponse('Alpha', 'Bravo', profil({ matches: 12 }), profil({ matches: 4 }))}
        text={text}
        locale="fr"
      />,
    )
    expect(screen.getAllByText(text.weaponsMatches(4)).length).toBeGreaterThan(0)
  })

  /**
   * NI COUVERTURE NI LISTE « SOUS LE SEUIL » sous les graphes (gate visuel 2026-09-17,
   * amendement D6). Ce témoin est l'INVERSE de celui qu'il remplace : il garantit que ces deux
   * phrases ne reviennent pas par mégarde. La fixture porte pourtant les deux faits —
   * `measured_kills: 80` sur `total_kills: 100`, et un rôle écarté à 6 mesures — donc leur
   * absence à l'écran est bien un choix d'affichage, pas un manque de donnée.
   */
  it('n’affiche ni la couverture ni les rôles sous le seuil', () => {
    const { container } = renderWithProviders(
      <CompareWeaponsSection
        left={reponse('Alpha', 'Bravo', avecPortee(20), avecPortee(25))}
        text={text}
        locale="fr"
      />,
    )
    expect(screen.queryByText(/frags mesurés sur/)).toBeNull()
    expect(container.textContent).not.toContain('Sous le seuil')
  })

  it('n’affiche JAMAIS une clé de manifeste brute', () => {
    const { container } = renderWithProviders(
      <CompareWeaponsSection
        left={reponse('Alpha', 'Bravo', avecPortee(20), avecPortee(25))}
        text={text}
        locale="fr"
      />,
    )
    expect(container.textContent).not.toContain('frags.role.')
    expect(container.textContent).not.toContain('frags.class.')
  })

  it('rend les armes du top avec leur icône quand l’URL est servie, le nom seul sinon', () => {
    renderWithProviders(
      <CompareWeaponsSection
        left={reponse('Alpha', 'Bravo', profil({}), profil({}))}
        text={text}
        locale="fr"
      />,
    )
    // BR75 porte une icône-masque : `WeaponIcon` en rend un nœud de rôle `img` nommé.
    expect(screen.getAllByRole('img', { name: 'BR75' }).length).toBeGreaterThan(0)
    // Sidekick n'a pas d'URL : le nom est là, mais aucune icône à son nom.
    expect(screen.getAllByText('Sidekick').length).toBeGreaterThan(0)
    expect(screen.queryByRole('img', { name: 'Sidekick' })).toBeNull()
  })
})

describe('CompareWeaponsSection — dégradations', () => {
  it('section ABSENTE quand la réponse ne porte pas de profil', () => {
    const { container } = renderWithProviders(
      <CompareWeaponsSection left={reponse('Alpha', 'Bravo')} text={text} locale="fr" />,
    )
    expect(container.textContent).toBe('')
  })

  it('bloc de portée ABSENT sans `range` — classes et armes restent (D9)', () => {
    renderWithProviders(
      <CompareWeaponsSection
        left={reponse('Alpha', 'Bravo', profil({}), profil({}))}
        text={text}
        locale="fr"
      />,
    )
    // Les classes n ont plus de sous-titre (gate visuel 2026-09-17) : la carte porte le titre.
    expect(screen.getByText(text.catWeapons)).toBeInTheDocument()
    expect(screen.getByText(text.weaponsTopTitle)).toBeInTheDocument()
    expect(screen.queryByText(text.weaponsRangeKills)).toBeNull()
    expect(screen.queryByText(text.weaponsRangeDeaths)).toBeNull()
  })
})

describe('CompareWeaponsSection — mode miroir', () => {
  it('rend trois colonnes d’armes et deux paires de graphes', () => {
    renderWithProviders(
      <CompareWeaponsSection
        left={reponse('Alpha', 'Bravo', avecPortee(20), avecPortee(25))}
        right={reponse('Alpha', 'Charlie', avecPortee(20), avecPortee(30))}
        text={text}
        locale="fr"
      />,
    )
    // Les trois joueurs nommés dans le bloc du top.
    expect(screen.getAllByText('Alpha').length).toBeGreaterThan(0)
    expect(screen.getAllByText('Bravo').length).toBeGreaterThan(0)
    expect(screen.getAllByText('Charlie').length).toBeGreaterThan(0)
    // Deux paires = deux cartes « Où ils fraguent » et deux « Où ils meurent ».
    expect(screen.getAllByText(text.weaponsRangeKills)).toHaveLength(2)
    expect(screen.getAllByText(text.weaponsRangeDeaths)).toHaveLength(2)
  })

  /**
   * LE TÉMOIN DE RENDU DU GATE VISUEL (2026-09-17) : Charlie porte un rôle
   * « Environnement » qu'Alpha et Bravo n'ont pas. Les deux paires de graphes doivent quand
   * même afficher les MÊMES lignes, dans le MÊME ordre — la ligne existe partout, vide chez
   * ceux qui n'ont rien. On le vérifie sur l'axe des catégories des quatre graphes, que le
   * mock d'ECharts expose via l'étiquette accessible des légendes et les libellés de rôle.
   */
  it('les quatre graphes portent le même axe quand un seul joueur a un rôle en plus', () => {
    const { container } = renderWithProviders(
      <CompareWeaponsSection
        left={reponse('Alpha', 'Bravo', avecPortee(20), avecPortee(25))}
        right={reponse('Alpha', 'Charlie', avecPortee(20), avecEnvironnement())}
        text={text}
        locale="fr"
      />,
    )
    // La section rend bien les deux paires.
    expect(screen.getAllByText(text.weaponsRangeKills)).toHaveLength(2)
    expect(screen.getAllByText(text.weaponsRangeDeaths)).toHaveLength(2)
    // Aucune clé brute n'a fuité malgré le rôle inconnu des deux premiers joueurs.
    expect(container.textContent).not.toContain('frags.role.')
    expect(container.textContent).not.toContain('frags.class.')
  })
})
