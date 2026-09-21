/**
 * SessionCoordination.test — LE LOT O sur la colonne de session : les deux cartes de la
 * section « Coordination » (D22-1, D22-6) et la carte « Portée des engagements » (D22-4).
 *
 * Ce que ces cas verrouillent, et pourquoi chacun compte :
 *
 *   - DEUX JAUGES, DEUX EN-TÊTES PROPRES, UN SEUL TRAIT DE PARITÉ. `UsageGaugeGrid` lisait
 *     ses en-têtes dans une table de TROIS dénominateurs du bloc « usages » et reconnaissait
 *     la colonne d'équipe par sa POSITION : montée telle quelle, la carte Riposte aurait
 *     affiché « Mon équipe dans le lobby » au-dessus de « Je suis couvert » et rayé une
 *     colonne sur deux. Le cas fixe les en-têtes ET l'absence de hachure ;
 *   - UNE CASE SANS DÉNOMINATEUR RESTE GRISE. Un match sans mort de camp ne vaut pas 0 % de
 *     ripostes — il n'est pas mesuré. Le tri par `tone` est la seule chose qui distingue les
 *     deux à l'écran ;
 *   - `available = false` GARDE LA RANGÉE et nomme sa cause (D8) : un bloc escamoté laisse
 *     la rangée bancale et se lit comme un bug ;
 *   - PORTÉE : les bâtons suivent l'ordre des matchs, un bâton sous le plancher est CREUX
 *     (contour pointillé, aucun remplissage), la médiane de session est une markLine, et
 *     sous trois bâtons pleins il n'y a PAS de bandes de rôle — trois bandes assises sur
 *     deux points ne séparent rien.
 */
import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'

import type { CoordinationBlock, MatchRangeBlock } from '@/lib/api/types'

import { SessionCoordinationSection } from './SessionCoordinationSection'
import { buildSessionRangeOption } from './charts/sessionRangeChart'
import { COORDINATION_TEXT } from './coordinationI18n'
import {
  bandCaption,
  buildAppuiGaugeRows,
  buildRiposteBand,
  buildRiposteGaugeRows,
} from './coordinationModel'
import { batonsPortee, medianeSession, seuilsSession } from './sessionRange.logic'

const t = COORDINATION_TEXT.fr

function couverture(taux: number, brut: number, n: number, faible = false) {
  return { taux, brut, n, par_match: brut / Math.max(1, n), echantillon_faible: faible }
}

function matchPoint(id: string, over: Partial<Record<string, number>> = {}) {
  return {
    match_id: id,
    assists_to_me: 2,
    my_assisted_kills: 4,
    my_deaths: 5,
    my_deaths_avenged: 3,
    my_measured_kills: 10,
    my_ripostes: 3,
    team_assists: 12,
    team_deaths: 20,
    team_deaths_avenged: 12,
    parity_pct: 25,
    riposte_share_pct: 40,
    assist_share_of_team_pct: 25,
    team_size: 4,
    ...over,
  }
}

const BLOC: CoordinationBlock = {
  available: true,
  fenetre_ms: 5000,
  matches_measured: 3,
  matches_total: 4,
  riposte: {
    je_suis_couvert: couverture(0.62, 31, 50),
    je_riposte: couverture(0.31, 18, 58),
    parity_pct: 25,
    team_deaths: 58,
    team_deaths_avenged: 30,
    delai_median_ms: 4200,
  },
  appui: {
    on_me_prepare: couverture(0.44, 22, 50),
    ma_part_des_appuis: couverture(0.34, 17, 50),
    parity_pct: 25,
  },
  per_match: [
    matchPoint('m1'),
    // Aucune mort de camp : la case doit rester GRISE malgré une part servie.
    matchPoint('m2', { team_deaths: 0, riposte_share_pct: 0 }),
    matchPoint('m3', { riposte_share_pct: 10 }),
  ],
}

describe('Section Coordination (D22-1 / D22-6)', () => {
  it('rend deux jauges nommées par le lot, sans hachure de lobby, et un seul trait de parité', () => {
    const { container } = render(<SessionCoordinationSection coordination={BLOC} />)

    // Les en-têtes sont ceux du lot, pas les trois dénominateurs du bloc « usages ».
    expect(screen.getByText(t.gaugeCovered)).toBeInTheDocument()
    expect(screen.getByText(t.gaugeIRiposte)).toBeInTheDocument()
    expect(screen.getByText(t.gaugePrepared)).toBeInTheDocument()
    expect(screen.getByText(t.gaugeAssistShare)).toBeInTheDocument()

    // Aucune colonne rapportée au lobby → aucune tranche hachurée.
    expect(container.querySelectorAll('[data-gauge-denominator="lobby"]')).toHaveLength(0)

    // Le chiffre d'appel (délai médian) est écrit ; aucune autre phrase (D22-verbosité).
    expect(container.querySelectorAll('[data-coordination-callout]')).toHaveLength(1)
    expect(screen.getByText(`${t.delaiMedian} : 4,2 s`)).toBeInTheDocument()
  })

  it('grise la case d’un match sans mort de camp et ne la compte pas dans le pied', () => {
    const cells = buildRiposteBand(BLOC.per_match ?? [], t, 'fr')
    expect(cells.map((c) => c.tone)).toEqual(['above', 'unmeasured', 'below'])
    // 1 case au-dessus sur 2 MESURÉES — la case grise sort du dénominateur.
    expect(bandCaption(cells, t)).toBe('1/2')
  })

  it('garde la rangée et nomme la cause quand le bloc est indisponible (D8)', () => {
    const { container } = render(
      <SessionCoordinationSection
        coordination={{ ...BLOC, available: false, matches_measured: 0 }}
      />,
    )
    expect(container.querySelector('[data-session-coordination="empty"]')).not.toBeNull()
    // Les deux cartes restent, avec leur titre et une cause nommée.
    expect(screen.getByText(t.cardRiposte)).toBeInTheDocument()
    expect(screen.getByText(t.cardAppui)).toBeInTheDocument()
    expect(container.querySelectorAll('[data-usage-empty]')).toHaveLength(2)
  })
})

describe('Repère d’habituel des jauges sans parité (lot S)', () => {
  it('pose l’habituel de la période sur « je suis couvert » et « on me prépare »', () => {
    const bloc: CoordinationBlock = {
      ...BLOC,
      riposte: { ...BLOC.riposte, habituel_pct: 55 },
      appui: { ...BLOC.appui, habituel_pct: 38 },
    }
    const [riposte] = buildRiposteGaugeRows(bloc, t, 'fr')
    const [appui] = buildAppuiGaugeRows(bloc, t, 'fr')

    // Le trait des deux jauges sans parité EST l'habituel...
    expect(riposte.gauges[0].parityPct).toBe(55)
    expect(appui.gauges[0].parityPct).toBe(38)
    // ...et l'infobulle le NOMME, faute de quoi il se lirait comme une parité.
    expect(riposte.gauges[0].tooltip).toContain('habituel')
    expect(appui.gauges[0].tooltip).toContain('habituel')
    // La jauge voisine garde SA parité, et son infobulle ne parle pas d'habituel.
    expect(riposte.gauges[1].parityPct).toBe(25)
    expect(riposte.gauges[1].tooltip).not.toContain('habituel')
  })

  it('n’invente aucun repère quand le contrat ne sert pas d’habituel', () => {
    const [riposte] = buildRiposteGaugeRows(BLOC, t, 'fr')
    const [appui] = buildAppuiGaugeRows(BLOC, t, 'fr')
    expect(riposte.gauges[0].parityPct).toBeNull()
    expect(appui.gauges[0].parityPct).toBeNull()
    expect(riposte.gauges[0].tooltip).not.toContain('habituel')
  })
})

const profil = (id: string, ordreIso: string, mediane: number, lobby: number, mesures: number) => ({
  match_id: id,
  played_at: ordreIso,
  map_name: 'Streets',
  lobby_median_m: lobby,
  lobby_measured: 40,
  players: [
    {
      xuid: 'x1',
      gamertag: 'Moi',
      median_m: mediane,
      lobby_delta_m: mediane - lobby,
      measured: mesures,
    },
  ],
})

/** Quatre matchs, servis EN DÉSORDRE : le tri chronologique doit les remettre en place. */
const BLOC_PORTEE: MatchRangeBlock = {
  kills_measured: 38,
  kills_total: 50,
  profiles: [
    profil('m3', '2026-04-21T21:00:00Z', 22, 20, 9),
    profil('m1', '2026-04-21T19:30:00Z', 14, 20, 12),
    profil('m4', '2026-04-21T21:30:00Z', 30, 20, 3), // sous le plancher → creux
    profil('m2', '2026-04-21T20:15:00Z', 18, 20, 8),
  ],
}

describe('Carte Portée des engagements (D22-4)', () => {
  it('range les bâtons du plus ancien au plus récent et étiquette « #N · carte »', () => {
    const { batons, categories } = batonsPortee(BLOC_PORTEE, 'moi')
    expect(batons.map((b) => b.matchId)).toEqual(['m1', 'm2', 'm3', 'm4'])
    expect(batons.map((b) => b.ecartM)).toEqual([-6, -2, 2, 10])
    expect(categories[0]).toBe('#1 · Streets')
  })

  it('dessine creux le bâton sous le plancher et plein les autres', () => {
    const { batons, categories } = batonsPortee(BLOC_PORTEE, 'moi')
    expect(batons.map((b) => b.plein)).toEqual([true, true, true, false])
    const option = buildSessionRangeOption(batons, {
      categories,
      seuils: seuilsSession(batons),
      medianeM: medianeSession(batons),
      libelles: {
        yAxis: t.rangeAxis,
        lobbyLine: t.rangeLobbyLine,
        medianLine: t.rangeSessionMedian,
        bandes: { front: t.roleFront, polyvalent: t.roleVersatile, sniper: t.roleSniper },
        tooltip: () => '',
      },
    }) as {
      series: [{ data: { value: number; itemStyle: Record<string, unknown> }[]; markLine: { data: unknown[] }; markArea?: { data: unknown[] } }]
    }
    const data = option.series[0].data
    expect(data.map((d) => d.value)).toEqual([-6, -2, 2, 10])
    // Bâton creux : transparent + contour pointillé. Bâtons pleins : une encre, pas de contour.
    expect(data[3].itemStyle.color).toBe('transparent')
    expect(data[3].itemStyle.borderType).toBe('dashed')
    expect(data[0].itemStyle.color).not.toBe('transparent')
    expect(data[0].itemStyle.borderType).toBeUndefined()
    // La ligne du lobby (0) ET la médiane de session (-2, sur les trois bâtons pleins).
    expect(option.series[0].markLine.data).toHaveLength(2)
    expect(medianeSession(batons)).toBe(-2)
    // Trois bâtons pleins ⇒ les bandes de rôle existent.
    expect(option.series[0].markArea?.data).toHaveLength(3)
  })

  it('n’assied aucune bande de rôle sous trois bâtons pleins', () => {
    const maigre: MatchRangeBlock = {
      kills_measured: 14,
      kills_total: 30,
      profiles: [
        profil('m1', '2026-04-21T19:30:00Z', 14, 20, 8),
        profil('m2', '2026-04-21T20:15:00Z', 18, 20, 6),
        profil('m3', '2026-04-21T21:00:00Z', 22, 20, 2),
      ],
    }
    const { batons, categories } = batonsPortee(maigre, 'moi')
    expect(batons.filter((b) => b.plein)).toHaveLength(2)
    expect(seuilsSession(batons)).toBeNull()
    const option = buildSessionRangeOption(batons, {
      categories,
      seuils: seuilsSession(batons),
      medianeM: medianeSession(batons),
      libelles: {
        yAxis: t.rangeAxis,
        lobbyLine: t.rangeLobbyLine,
        medianLine: t.rangeSessionMedian,
        bandes: { front: t.roleFront, polyvalent: t.roleVersatile, sniper: t.roleSniper },
        tooltip: () => '',
      },
    }) as { series: [{ markArea?: unknown }] }
    expect(option.series[0].markArea).toBeUndefined()
  })
})
