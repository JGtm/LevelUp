/**
 * La logique PURE de la grille des cartes.
 *
 * Ce que ces tests cadenassent :
 *   - le tri par nombre de matchs, et son départage STABLE par le nom ;
 *   - l'état sous plancher lu sur le VERDICT DU SERVEUR, aux deux bornes (9 et 10) ;
 *   - la barre victoires / défaites quand V + D est INFÉRIEUR aux matchs (le cas nominal :
 *     nuls, abandons, résultat inconnu) — la barre ne doit jamais remplir le cadre ;
 *   - les deux protections d'affichage (zéro match, contrat incohérent) ;
 *   - LE PÉRIMÈTRE (phase 4 bis) : le contexte de filtre que la barre L2 fait résoudre,
 *     les sessions proposées selon la composition, et la traduction gamertag → xuid —
 *     dont le refus de traduire, qui est ce qui empêche un filtre d'être élargi en
 *     silence.
 */
import { describe, expect, it } from 'vitest'

import { EXPERIENCE_TO_CASCADE } from '@/features/_shared/experienceCascade'
import type { TacticalMapCard } from '@/lib/api/types'

import {
  barreResultats,
  couvertureGrille,
  estOuvrable,
  nomCarte,
  trierCartes,
  contexteFiltre,
  resoudreComposition,
  sessionsProposees,
} from './tacticalLogic'
import { TACTICAL_SCOPE_DEFAUT, type TacticalScope } from './tacticalScope'

function carte(over: Partial<TacticalMapCard> = {}): TacticalMapCard {
  return {
    map_id: 'm',
    map_name: 'Map',
    map_name_fr: '',
    matchs: 10,
    victoires: 5,
    defaites: 5,
    sous_plancher: false,
    ...over,
  }
}

describe('trierCartes', () => {
  it('classe de la carte la plus jouée à la moins jouée', () => {
    const tri = trierCartes([
      carte({ map_id: 'a', map_name: 'Aquarius', matchs: 4 }),
      carte({ map_id: 'b', map_name: 'Bazaar', matchs: 31 }),
      carte({ map_id: 'c', map_name: 'Catalyst', matchs: 12 }),
    ])
    expect(tri.map((c) => c.map_id)).toEqual(['b', 'c', 'a'])
  })

  it('départage par le NOM à nombre de matchs égal — deux affichages, un seul ordre', () => {
    const tri = trierCartes([
      carte({ map_id: 'z', map_name: 'Streets', matchs: 10 }),
      carte({ map_id: 'a', map_name: 'Aquarius', matchs: 10 }),
      carte({ map_id: 'm', map_name: 'Live Fire', matchs: 10 }),
    ])
    expect(tri.map((c) => c.map_name)).toEqual(['Aquarius', 'Live Fire', 'Streets'])
  })

  it("ne filtre RIEN : une carte sous le plancher reste dans la liste", () => {
    const tri = trierCartes([
      carte({ map_id: 'a', matchs: 20 }),
      carte({ map_id: 'b', matchs: 2, sous_plancher: true }),
    ])
    expect(tri).toHaveLength(2)
    expect(tri[1].map_id).toBe('b')
  })

  it("ne mute pas la liste reçue", () => {
    const source = [carte({ map_id: 'a', matchs: 1 }), carte({ map_id: 'b', matchs: 9 })]
    trierCartes(source)
    expect(source.map((c) => c.map_id)).toEqual(['a', 'b'])
  })
})

describe('estOuvrable — les deux bornes du plancher', () => {
  // Le seuil est décidé PAR LE SERVEUR (`sous_plancher`) : le client ne le recalcule pas.
  // Ces deux cas posent la borne exacte telle que le contrat la sert.
  it('9 matchs, sous le plancher de 10 : la carte ne s’ouvre pas', () => {
    expect(estOuvrable(carte({ matchs: 9, sous_plancher: true }))).toBe(false)
  })

  it('10 matchs pile, plancher atteint : la carte s’ouvre', () => {
    expect(estOuvrable(carte({ matchs: 10, sous_plancher: false }))).toBe(true)
  })
})

describe('barreResultats', () => {
  it('V + D INFÉRIEUR aux matchs : le reste est du neutre, jamais du V ni du D', () => {
    const parts = barreResultats(carte({ matchs: 10, victoires: 6, defaites: 3 }))
    expect(parts.victoires).toBeCloseTo(0.6)
    expect(parts.defaites).toBeCloseTo(0.3)
    expect(parts.autres).toBeCloseTo(0.1)
  })

  it('V + D égal aux matchs : aucun reste', () => {
    const parts = barreResultats(carte({ matchs: 8, victoires: 5, defaites: 3 }))
    expect(parts.victoires + parts.defaites).toBeCloseTo(1)
    expect(parts.autres).toBeCloseTo(0)
  })

  it('aucun match : barre vide, jamais une division par zéro', () => {
    const parts = barreResultats(carte({ matchs: 0, victoires: 0, defaites: 0 }))
    expect(parts).toEqual({ victoires: 0, defaites: 0, autres: 0 })
    expect(Number.isNaN(parts.victoires)).toBe(false)
  })

  it('contrat incohérent (V + D > matchs) : la barre ne déborde pas du cadre', () => {
    const parts = barreResultats(carte({ matchs: 4, victoires: 5, defaites: 3 }))
    expect(parts.victoires + parts.defaites + parts.autres).toBeLessThanOrEqual(1.0000001)
    expect(parts.victoires).toBeCloseTo(5 / 8)
    expect(parts.defaites).toBeCloseTo(3 / 8)
  })

  it('valeurs négatives : ramenées à zéro plutôt que peintes à l’envers', () => {
    const parts = barreResultats(carte({ matchs: 10, victoires: -3, defaites: 4 }))
    expect(parts.victoires).toBe(0)
    expect(parts.defaites).toBeCloseTo(0.4)
  })
})

describe('couvertureGrille', () => {
  it('compte les cartes ET la somme de leurs matchs', () => {
    expect(
      couvertureGrille([carte({ matchs: 12 }), carte({ matchs: 3 }), carte({ matchs: 30 })]),
    ).toEqual({ cartes: 3, matchs: 45 })
  })

  it('grille vide : deux zéros', () => {
    expect(couvertureGrille([])).toEqual({ cartes: 0, matchs: 0 })
  })
})

describe('nomCarte', () => {
  it('le français vient du contrat', () => {
    expect(nomCarte(carte({ map_name: 'Streets', map_name_fr: 'Ruelles' }), 'fr')).toBe('Ruelles')
  })

  it('nom FR vide : on retombe sur le nom canonique plutôt que sur du blanc', () => {
    expect(nomCarte(carte({ map_name: 'Streets', map_name_fr: '   ' }), 'fr')).toBe('Streets')
  })

  it('en anglais, le nom canonique', () => {
    expect(nomCarte(carte({ map_name: 'Streets', map_name_fr: 'Ruelles' }), 'en')).toBe('Streets')
  })

  it('aucun nom : le map_id plutôt qu’une vignette anonyme', () => {
    expect(nomCarte(carte({ map_id: 'asset-42', map_name: '', map_name_fr: '' }), 'fr')).toBe(
      'asset-42',
    )
  })
})

// ─── LE PERIMETRE ET LA COMPOSITION (phase 4 bis) ─────────────────────────────

describe('contexteFiltre', () => {
  const base: TacticalScope = { ...TACTICAL_SCOPE_DEFAUT }

  it('sans session : mode « period », les bornes et la cascade de la barre', () => {
    const ctx = contexteFiltre({
      ...base,
      debut: '2026-01-01',
      fin: '2026-02-01',
      playlists: ['Ranked Arena'],
      modes: ['Slayer'],
      experience: 'ranked',
    })
    expect(ctx.filter_mode).toBe('period')
    expect(ctx.period).toEqual({ start_date: '2026-01-01', end_date: '2026-02-01' })
    expect(ctx.cascade?.playlists).toEqual(['Ranked Arena'])
    expect(ctx.cascade?.modes).toEqual(['Slayer'])
    expect(ctx.cascade?.experience_types).toEqual(EXPERIENCE_TO_CASCADE.ranked)
    expect(ctx.match_context).toBeUndefined()
  })

  it('avec une session epinglee : mode « sessions », et le label transmis', () => {
    const ctx = contexteFiltre({ ...base, sessions: ['Session du 3 mars'], debut: '2026-01-01' })
    expect(ctx.filter_mode).toBe('sessions')
    expect(ctx.sessions?.picked_sessions).toEqual(['Session du 3 mars'])
    // La periode reste PUBLIEE (le backend l'ignore tant qu'une session est pickee) :
    // la retirer ferait perdre la borne au retour en mode periode.
    expect(ctx.period?.start_date).toBe('2026-01-01')
  })

  it('la vue ne descend en match_context que si elle n’est pas « all »', () => {
    expect(contexteFiltre({ ...base, vue: 'squad' }).match_context).toBe('squad')
    expect(contexteFiltre({ ...base, vue: 'all' }).match_context).toBeUndefined()
  })

  it('un label de session vide n’ouvre PAS le mode sessions', () => {
    const ctx = contexteFiltre({ ...base, sessions: ['  '] })
    expect(ctx.filter_mode).toBe('period')
    expect(ctx.sessions?.picked_sessions).toEqual([])
  })
})

describe('sessionsProposees', () => {
  const options = [
    { label: 'Solo 1', session_id: 's1', match_count: 4, match_count_filtered: 4, is_squad: false, started_at_utc: 'A', ended_at_utc: 'B' },
    { label: 'Escouade 1', session_id: 's2', match_count: 6, match_count_filtered: 6, is_squad: true, started_at_utc: 'C', ended_at_utc: 'D' },
  ]

  it('sans composition : les sessions SOLO', () => {
    const out = sessionsProposees(options, false)
    expect(out.map((s) => s.label)).toEqual(['Solo 1'])
  })

  it('avec une composition : les sessions d’ESCOUADE', () => {
    const out = sessionsProposees(options, true)
    expect(out.map((s) => s.label)).toEqual(['Escouade 1'])
    // La forme attendue par SessionMultiSelect, pas celle de /filters/resolve.
    expect(out[0].started_at).toBe('C')
    expect(out[0].ended_at).toBe('D')
    expect(out[0].match_count).toBe(6)
  })
})

describe('resoudreComposition', () => {
  const options = [
    { gamertag: 'Ami', xuid: 'xuid(1)', encounter_count: 10 },
    { gamertag: 'SansXuid', encounter_count: 3 },
  ]

  it('traduit les gamertags en xuids, dans l’ordre de la composition', () => {
    expect(resoudreComposition(['Ami'], options)).toEqual({ xuids: ['xuid(1)'], inconnus: [] })
  })

  it('la casse ne compte pas — un gamertag se saisit comme il se prononce', () => {
    expect(resoudreComposition(['ami'], options).xuids).toEqual(['xuid(1)'])
  })

  it('un nom introuvable est SIGNALE, jamais ignore', () => {
    const { xuids, inconnus } = resoudreComposition(['Ami', 'Fantome'], options)
    expect(xuids).toEqual(['xuid(1)'])
    expect(inconnus).toEqual(['Fantome'])
  })

  it('une option SANS xuid ne resout rien : elle ne peut pas filtrer', () => {
    expect(resoudreComposition(['SansXuid'], options).inconnus).toEqual(['SansXuid'])
  })
})
