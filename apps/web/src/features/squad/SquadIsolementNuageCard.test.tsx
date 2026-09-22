/**
 * Le NUAGE « Pourquoi la riposte ne vient pas » — ce qu'il montre et ce qu'il tait.
 *
 * Ce que ces tests cadenassent : un état vide n'est jamais un nuage à zéro point ; un petit
 * point par MORT et un gros point par JOUEUR ; les deux bandes nommées portent les valeurs
 * absentes (hors de vue, jamais vengée) ; l'infobulle d'une mort dit la COUVERTURE en
 * portée du radar ; et les deux langues rendent deux textes.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, screen } from '@testing-library/react'

import { renderWithProviders } from '@/test/render-utils'
import { useAppShellStore } from '@/stores/appShellStore'
import type {
  SquadEchangeJoueur,
  SquadIsolementMort,
  SquadIsolementRepere,
  SquadNuageIsolement,
} from '@/lib/api/types'

import { SquadIsolementNuageCard } from './SquadIsolementNuageCard'
import { SYMBOLE_MORT, SYMBOLE_MORT_HORS_FENETRE } from './squadIsolementNuageOption'

// jsdom n'a pas de canvas : on mocke echarts-for-react (comme FirstBloodLanes) pour
// capturer l'option ECharts construite, notamment le formatter de tooltip.
const captured: Array<Record<string, unknown>> = []
vi.mock('echarts-for-react', () => ({
  default: (props: Record<string, unknown>) => {
    captured.push(props)
    return <div data-testid="isolement-nuage-stub" />
  },
}))

beforeEach(() => {
  useAppShellStore.setState({ locale: 'fr' })
  captured.length = 0
})
afterEach(() => useAppShellStore.setState({ locale: 'fr' }))

const joueurs: SquadEchangeJoueur[] = [
  { xuid: 'x1', gamertag: 'Alice' },
  { xuid: 'x2', gamertag: 'Bob' },
]

function isoCouverture(brut: number, n: number, echantillonFaible: boolean) {
  return {
    taux: n > 0 ? brut / n : 0,
    brut,
    par_match: n > 0 ? brut / 3 : 0,
    n,
    echantillon_faible: echantillonFaible,
  }
}

function mort(over: Partial<SquadIsolementMort> = {}): SquadIsolementMort {
  return {
    xuid: 'x1',
    gamertag: 'Alice',
    match_id: 'm1',
    time_ms: 10_000,
    distance_ratio: 0.6,
    hors_de_vue: false,
    vengee: true,
    hors_fenetre: false,
    delai_ms: 3_000,
    ...over,
  } as SquadIsolementMort
}

/** Une mort dont le tueur est tombé APRÈS la fenêtre : délai publié, jamais une riposte. */
function mortHorsFenetre(delaiMs: number): SquadIsolementMort {
  return mort({ vengee: false, hors_fenetre: true, delai_ms: delaiMs })
}

function repere(over: Partial<SquadIsolementRepere> = {}): SquadIsolementRepere {
  return {
    xuid: 'x1',
    gamertag: 'Alice',
    nb_morts: 12,
    mediane_distance_ratio: 0.8,
    mediane_delai_ms: 4_000,
    part_isolee: isoCouverture(4, 10, true),
    couverture: isoCouverture(6, 10, true),
    ...over,
  } as SquadIsolementRepere
}

function nuageDe(over: Partial<SquadNuageIsolement> = {}): SquadNuageIsolement {
  return {
    morts: [mort(), mort({ xuid: 'x2', gamertag: 'Bob', distance_ratio: 1.4 })],
    reperes: [repere(), repere({ xuid: 'x2', gamertag: 'Bob', nb_morts: 20 })],
    plancher_echantillon_faible: 30,
    fenetre_ms: 5_000,
    plafond_ms: 60_000,
    ...over,
  } as SquadNuageIsolement
}

/** L'option ECharts du dernier rendu. */
function derniereOption<T>(): T {
  return captured[captured.length - 1].option as T
}

describe('SquadIsolementNuageCard', () => {
  it('ÉTAT VIDE, jamais un nuage à zéro point, quand aucune mort n’est mesurée', () => {
    renderWithProviders(
      <SquadIsolementNuageCard nuage={nuageDe({ morts: [], reperes: [] })} joueurs={joueurs} />,
    )
    expect(screen.getByText(/Aucune mort mesurée/i)).toBeTruthy()
    expect(screen.queryByTestId('chart-card')).toBeNull()
  })

  it('rend le nuage dès qu’une mort est publiée', () => {
    renderWithProviders(<SquadIsolementNuageCard nuage={nuageDe()} joueurs={joueurs} />)
    expect(screen.getByTestId('squad-isolement-nuage')).toBeTruthy()
    expect(screen.getByTestId('chart-card')).toBeTruthy()
  })

  // L'INFOBULLE PORTE DÉSORMAIS AUSSI CE QUE LA FIGURE DÉNOMBRE (2026-09-21, lot A1) :
  // cette phrase vivait en gris au-dessus du graphe. Une carte n'a qu'une infobulle, et
  // l'aide de lecture y reste concise — trois phrases, plus la légende de la figure.
  it('l’aide ⓘ reste CONCISE et porte la portée du radar comme la légende de la figure', () => {
    renderWithProviders(<SquadIsolementNuageCard nuage={nuageDe()} joueurs={joueurs} />)
    fireEvent.mouseEnter(screen.getByRole('button', { name: /info/i }))
    const aide = screen.getByRole('tooltip').textContent ?? ''
    expect(aide).toContain('radar')
    expect(aide).toContain('un gros point par joueur')
    expect(aide.split('.').filter((p) => p.trim().length > 0).length).toBeLessThanOrEqual(4)
  })

  it('PARITÉ FR/EN : les deux langues rendent un titre, et deux titres différents', () => {
    const { unmount } = renderWithProviders(
      <SquadIsolementNuageCard nuage={nuageDe()} joueurs={joueurs} />,
    )
    const fr = screen.getByText('Frags non ripostés').textContent ?? ''
    unmount()
    useAppShellStore.setState({ locale: 'en' })
    renderWithProviders(<SquadIsolementNuageCard nuage={nuageDe()} joueurs={joueurs} />)
    const en = screen.getByText('Unriposted kills').textContent ?? ''
    expect(fr.length).toBeGreaterThan(0)
    expect(en.length).toBeGreaterThan(0)
    expect(en).not.toBe(fr)
  })

  it('un point par MORT, un gros point par JOUEUR', async () => {
    renderWithProviders(
      <SquadIsolementNuageCard
        nuage={nuageDe({
          morts: [mort(), mort({ time_ms: 20_000 }), mort({ xuid: 'x2', gamertag: 'Bob' })],
        })}
        joueurs={joueurs}
      />,
    )
    await screen.findByTestId('isolement-nuage-stub')
    const option = derniereOption<{
      series: Array<{ name: string; data: Array<Record<string, unknown>> }>
    }>()
    const sériesMorts = option.series.filter((s) => 'raw' in s.data[0])
    const sériesRepères = option.series.filter((s) => 'repereRaw' in s.data[0])
    expect(sériesMorts.map((s) => s.data.length)).toEqual([2, 1])
    expect(sériesRepères).toHaveLength(2)
  })

  it('une mort HORS DE VUE se pose dans la bande de droite, jamais à une distance inventée', async () => {
    renderWithProviders(
      <SquadIsolementNuageCard
        nuage={nuageDe({
          morts: [mort({ distance_ratio: undefined, hors_de_vue: true })],
          reperes: [repere()],
        })}
        joueurs={joueurs}
      />,
    )
    await screen.findByTestId('isolement-nuage-stub')
    const option = derniereOption<{
      xAxis: { max: number }
      series: Array<{ data: Array<{ value: [number, number] }> }>
    }>()
    // Aucun ratio observé → zone mesurée bornée au minimum (2), bande au-delà de 2,5.
    expect(option.series[0].data[0].value[0]).toBeGreaterThan(2.5)
    expect(option.series[0].data[0].value[0]).toBeLessThanOrEqual(option.xAxis.max)
  })

  it('une mort JAMAIS VENGÉE se pose dans la bande haute, jamais à un délai inventé', async () => {
    renderWithProviders(
      <SquadIsolementNuageCard
        nuage={nuageDe({
          morts: [mort({ vengee: false, hors_fenetre: false, delai_ms: undefined })],
          reperes: [repere()],
        })}
        joueurs={joueurs}
      />,
    )
    await screen.findByTestId('isolement-nuage-stub')
    const option = derniereOption<{
      yAxis: { max: number }
      series: Array<{ data: Array<{ value: [number, number] }> }>
    }>()
    // La bande est AU-DESSUS du plafond du contrat (60 s), dans l'espace réservé.
    expect(option.series[0].data[0].value[1]).toBeGreaterThan(60)
    expect(option.series[0].data[0].value[1]).toBeLessThanOrEqual(option.yAxis.max)
  })

  it('le repère de la PORTÉE DU RADAR est tracé à 1,0', async () => {
    renderWithProviders(<SquadIsolementNuageCard nuage={nuageDe()} joueurs={joueurs} />)
    await screen.findByTestId('isolement-nuage-stub')
    const option = derniereOption<{
      series: Array<{ markLine?: { data: Array<{ xAxis: number }> } }>
    }>()
    const ligne = option.series.find((s) => s.markLine)?.markLine
    expect(ligne?.data[0].xAxis).toBe(1)
  })

  // ─── LA RÈGLE DES 5 s SE TRACE, ELLE NE COUPE PAS (décision du 2026-09-22) ───────

  it('le repère de la FENÊTRE DE RIPOSTE vient du contrat, jamais d’un 5 en dur', async () => {
    renderWithProviders(
      <SquadIsolementNuageCard nuage={nuageDe({ fenetre_ms: 8_000 })} joueurs={joueurs} />,
    )
    await screen.findByTestId('isolement-nuage-stub')
    const option = derniereOption<{
      series: Array<{ markLine?: { data: Array<{ xAxis?: number; yAxis?: number; name?: string }> } }>
    }>()
    const ligne = option.series.find((s) => s.markLine)?.markLine
    const fenetre = ligne?.data.find((d) => d.yAxis != null)
    expect(fenetre?.yAxis).toBe(8)
    expect(fenetre?.name).toContain('Fenêtre de riposte')
  })

  it('une mort HORS FENÊTRE reste VISIBLE à son délai, sous la bande réservée', async () => {
    renderWithProviders(
      <SquadIsolementNuageCard
        nuage={nuageDe({ morts: [mortHorsFenetre(30_000)], reperes: [repere()] })}
        joueurs={joueurs}
      />,
    )
    await screen.findByTestId('isolement-nuage-stub')
    const option = derniereOption<{
      yAxis: { max: number }
      series: Array<{ data: Array<{ value: [number, number] }> }>
    }>()
    expect(option.series[0].data[0].value[1]).toBe(30)
    expect(option.yAxis.max).toBeGreaterThan(60)
  })

  // ─── L'AXE DES DÉLAIS EST LOGARITHMIQUE ET BORNÉ (décision du 2026-09-22) ──────

  it('l’axe des délais est en LOG, du plancher 0,2 s au plafond DU CONTRAT', async () => {
    renderWithProviders(
      <SquadIsolementNuageCard nuage={nuageDe({ plafond_ms: 30_000 })} joueurs={joueurs} />,
    )
    await screen.findByTestId('isolement-nuage-stub')
    const option = derniereOption<{
      yAxis: { type: string; min: number; max: number; axisLabel: { customValues: number[] } }
    }>()
    expect(option.yAxis.type).toBe('log')
    expect(option.yAxis.min).toBe(0.2)
    // Plafond 30 s : l'axe ne monte pas jusqu'à 60, et aucune graduation ne dépasse.
    expect(option.yAxis.axisLabel.customValues).toEqual([0.2, 0.5, 1, 2, 5, 10, 30])
    expect(option.yAxis.max).toBeGreaterThan(30)
  })

  it('les graduations s’écrivent en secondes DANS LA LOCALE, jamais en « 0.2 » brut', async () => {
    renderWithProviders(<SquadIsolementNuageCard nuage={nuageDe()} joueurs={joueurs} />)
    await screen.findByTestId('isolement-nuage-stub')
    const option = derniereOption<{
      yAxis: { axisLabel: { formatter: (v: number) => string } }
    }>()
    expect(option.yAxis.axisLabel.formatter(0.2)).toBe('0,2 s')
    expect(option.yAxis.axisLabel.formatter(60)).toBe('60 s')
  })

  // LA FORME DIT L'ÉTAT, ET L'ENCRE RESTE PLEINE (décision utilisateur du 2026-09-22).

  it('une mort HORS FENÊTRE se peint en LOSANGE PLEIN, dans l’encre du joueur', async () => {
    renderWithProviders(
      <SquadIsolementNuageCard
        nuage={nuageDe({ morts: [mort(), mortHorsFenetre(60_000)], reperes: [repere()] })}
        joueurs={joueurs}
      />,
    )
    await screen.findByTestId('isolement-nuage-stub')
    const option = derniereOption<{
      series: Array<{
        data: Array<{ symbol: string; symbolSize: number; itemStyle: Record<string, unknown> }>
      }>
    }>()
    const [dans, hors] = option.series[0].data
    expect(dans.symbol).toBe(SYMBOLE_MORT)
    expect(hors.symbol).toBe(SYMBOLE_MORT_HORS_FENETRE)
    expect(hors.symbol).toBe('diamond')
    // MÊME encre, MÊME taille, aucun remplissage retiré : seule la silhouette change.
    expect(hors.itemStyle.color).toBe(dans.itemStyle.color)
    expect(hors.itemStyle.color).not.toBe('transparent')
    expect(hors.symbolSize).toBe(dans.symbolSize)
  })

  it('AUCUNE opacité sous 1 dans l’option — ni point, ni repère, ni bande', async () => {
    renderWithProviders(
      <SquadIsolementNuageCard
        nuage={nuageDe({
          morts: [mort(), mortHorsFenetre(60_000), mort({ xuid: 'x2', gamertag: 'Bob' })],
        })}
        joueurs={joueurs}
      />,
    )
    await screen.findByTestId('isolement-nuage-stub')
    const option = derniereOption<Record<string, unknown>>()
    const attenuations: Array<[string, number]> = []
    const balaye = (noeud: unknown, chemin: string) => {
      if (Array.isArray(noeud)) {
        noeud.forEach((v, i) => balaye(v, `${chemin}[${i}]`))
        return
      }
      if (noeud == null || typeof noeud !== 'object') return
      for (const [cle, valeur] of Object.entries(noeud as Record<string, unknown>)) {
        if (cle === 'opacity' && typeof valeur === 'number' && valeur < 1) {
          attenuations.push([`${chemin}.${cle}`, valeur])
        }
        balaye(valeur, `${chemin}.${cle}`)
      }
    }
    balaye(option, 'option')
    expect(attenuations).toEqual([])
  })

  it('l’infobulle d’une mort hors fenêtre dit le délai ET le verdict « sans riposte »', async () => {
    renderWithProviders(
      <SquadIsolementNuageCard
        nuage={nuageDe({ morts: [mortHorsFenetre(60_000)], reperes: [repere()] })}
        joueurs={joueurs}
      />,
    )
    await screen.findByTestId('isolement-nuage-stub')
    const option = derniereOption<{
      tooltip: { formatter: (p: unknown) => string }
      series: Array<{ data: Array<Record<string, unknown>> }>
    }>()
    const html = option.tooltip.formatter({ data: option.series[0].data[0] })
    expect(html).toContain('60,0')
    expect(html).toContain('hors fenêtre')
    expect(html).toContain('sans riposte')
    expect(html).not.toContain('Ripostée en')
  })

  it('la légende NOMME le losange — un encodage qu’on ne nomme pas ne se lit pas', () => {
    renderWithProviders(<SquadIsolementNuageCard nuage={nuageDe()} joueurs={joueurs} />)
    const pastille = screen.getByTestId('squad-isolement-legende-hors-fenetre')
    // Un LOSANGE PLEIN : un carré tourné d'un quart de tour, jamais un cercle évidé.
    expect(pastille.getAttribute('style')).toContain('rotate(45deg)')
    expect(pastille.className).not.toContain('rounded-full')
    expect(pastille.className).not.toContain('border')
    expect(screen.getByText(/tueur tombé hors fenêtre/i)).toBeTruthy()
  })

  it('l’infobulle d’une mort dit la COUVERTURE en portée du radar, et son délai', async () => {
    renderWithProviders(<SquadIsolementNuageCard nuage={nuageDe()} joueurs={joueurs} />)
    await screen.findByTestId('isolement-nuage-stub')
    const option = derniereOption<{
      tooltip: { formatter: (p: unknown) => string }
      series: Array<{ data: Array<Record<string, unknown>> }>
    }>()
    const html = option.tooltip.formatter({ data: option.series[0].data[0] })
    expect(html).toContain('Couverture')
    expect(html).toContain('portée du radar')
    expect(html).toContain('Ripostée en')
  })

  it('l’infobulle d’une mort hors de vue ne cite aucune distance', async () => {
    renderWithProviders(
      <SquadIsolementNuageCard
        nuage={nuageDe({
          morts: [
            mort({
              distance_ratio: undefined,
              hors_de_vue: true,
              vengee: false,
              hors_fenetre: false,
              delai_ms: undefined,
            }),
          ],
          reperes: [repere()],
        })}
        joueurs={joueurs}
      />,
    )
    await screen.findByTestId('isolement-nuage-stub')
    const option = derniereOption<{
      tooltip: { formatter: (p: unknown) => string }
      series: Array<{ data: Array<Record<string, unknown>> }>
    }>()
    const html = option.tooltip.formatter({ data: option.series[0].data[0] })
    expect(html).toContain('aucun coéquipier en vue')
    expect(html).toContain('Jamais ripostée')
  })

  it('la réserve « échantillon faible » se lit dans l’infobulle du repère, pas seulement dans sa bordure', async () => {
    renderWithProviders(
      <SquadIsolementNuageCard
        nuage={nuageDe({
          morts: [mort()],
          reperes: [repere({ part_isolee: isoCouverture(4, 10, true) })],
        })}
        joueurs={joueurs}
      />,
    )
    await screen.findByTestId('isolement-nuage-stub')
    const option = derniereOption<{
      tooltip: { formatter: (p: unknown) => string }
      series: Array<{ data: Array<Record<string, unknown>> }>
    }>()
    const repereDatum = option.series.find((s) => 'repereRaw' in s.data[0])?.data[0]
    const html = option.tooltip.formatter({ data: repereDatum })
    expect(html).toContain('échantillon faible')
    expect(html).toContain("Taux de riposte")
  })

  it('échantillon suffisant : le repère est plein, aucune bordure pointillée', async () => {
    renderWithProviders(
      <SquadIsolementNuageCard
        nuage={nuageDe({
          morts: [mort()],
          reperes: [repere({ part_isolee: isoCouverture(12, 40, false) })],
        })}
        joueurs={joueurs}
      />,
    )
    await screen.findByTestId('isolement-nuage-stub')
    const option = derniereOption<{
      series: Array<{ data: Array<{ itemStyle: Record<string, unknown> }> }>
    }>()
    const repereDatum = option.series.find((s) => 'repereRaw' in s.data[0])?.data[0]
    expect(repereDatum?.itemStyle.borderType).toBeUndefined()
  })

  it('un petit point se peint à ENCRE PLEINE, sans opacité posée', async () => {
    renderWithProviders(<SquadIsolementNuageCard nuage={nuageDe()} joueurs={joueurs} />)
    await screen.findByTestId('isolement-nuage-stub')
    const option = derniereOption<{
      series: Array<{ data: Array<{ itemStyle: Record<string, unknown> }> }>
    }>()
    // jsdom ne résout pas les CSS vars : la couleur y est vide, c'est l'ABSENCE d'opacité
    // et la PRÉSENCE d'un remplissage (jamais « transparent ») qui se vérifient ici.
    expect(option.series[0].data[0].itemStyle.opacity).toBeUndefined()
    expect(option.series[0].data[0].itemStyle).toHaveProperty('color')
    expect(option.series[0].data[0].itemStyle.color).not.toBe('transparent')
  })
})
