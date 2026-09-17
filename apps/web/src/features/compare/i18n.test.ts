/**
 * Garde-rails du dictionnaire compare (lot Go-struct, 2026-08-04).
 *
 * Le contrat backend ne transporte plus de libellé de métrique (`label_fr`
 * retiré de CompareMetricRow). La résolution est donc entièrement côté front :
 * registre canonique de champs → dictionnaire de feature → clé humanisée.
 *
 * Ces tests figent les trois invariants qui rendent cette chaîne sûre :
 *   1. le registre PRIME sur le dictionnaire local ;
 *   2. aucune clé n'est déclarée des deux côtés (sinon le dictionnaire local
 *      re-diverge en silence du registre — la panne de 2026-08-02) ;
 *   3. FR et EN exposent exactement les mêmes clés locales.
 */
import { describe, it, expect } from 'vitest'

import { getCompareText, resolveMetricLabel, METRIC_TO_FIELD_KEY } from './i18n'

const REGISTRY = {
  fields: {
    win_rate: { label: 'Taux de victoire' },
    avg_damage_taken: { label: 'Dégâts subis / match' },
    max_killing_spree: { label: 'Meilleure série' },
  },
}

describe('compare — résolution des libellés de métrique', () => {
  it('le registre canonique prime sur toute valeur locale', () => {
    const text = getCompareText('fr', REGISTRY)
    expect(resolveMetricLabel(text, 'win_rate')).toBe('Taux de victoire')
    expect(resolveMetricLabel(text, 'max_killing_spree')).toBe('Meilleure série')
  })

  it('damage_taken_per_game est résolu par le registre (avg_damage_taken)', () => {
    expect(METRIC_TO_FIELD_KEY['damage_taken_per_game']).toBe('avg_damage_taken')
    const text = getCompareText('fr', REGISTRY)
    expect(resolveMetricLabel(text, 'damage_taken_per_game')).toBe('Dégâts subis / match')
  })

  it('les métriques sans FieldKey canonique gardent le dictionnaire de feature', () => {
    expect(resolveMetricLabel(getCompareText('fr', REGISTRY), 'csr')).toBe('CSR (saison actuelle)')
    expect(resolveMetricLabel(getCompareText('en', REGISTRY), 'csr')).toBe('CSR (current season)')
  })

  it('repli sur la clé humanisée quand le registre ne déclare pas la clé', () => {
    const text = getCompareText('fr', { fields: {} })
    expect(resolveMetricLabel(text, 'avg_life_secs')).toBe('Avg life secs')
    expect(resolveMetricLabel(text, 'metrique_inconnue')).toBe('Metrique inconnue')
  })

  it('aucune clé mappée sur le registre ne subsiste dans le dictionnaire local', () => {
    const fr = getCompareText('fr')
    const en = getCompareText('en')
    const duplicated = Object.keys(METRIC_TO_FIELD_KEY).filter(
      (k) => k in fr.metrics || k in en.metrics,
    )
    expect(
      duplicated,
      'Une clé mappée sur fields.toml ne doit pas avoir de libellé local : ' +
        'le doublon re-diverge du registre en silence.',
    ).toEqual([])
  })

  it('parité stricte des clés FR / EN du dictionnaire local', () => {
    const fr = Object.keys(getCompareText('fr').metrics).sort()
    const en = Object.keys(getCompareText('en').metrics).sort()
    expect(fr).toEqual(en)
    expect(fr.length).toBeGreaterThan(0)
  })
})

/**
 * Parité FR/EN du dictionnaire ENTIER — pas seulement de `metrics` (2026-09-17, lot 4).
 *
 * Le test historique ne comparait que les clés de `metrics`, parce que c'était la seule table
 * ouverte du dictionnaire. Le profil d'armes y ajoute onze entrées de premier niveau, dont
 * quatre FONCTIONS : une clé oubliée en anglais serait alors `undefined` à l'appel, donc un
 * plantage de rendu — et rien ne l'aurait vu. Le typage `Record<Locale, CompareText>` couvre
 * les champs manquants ; ce test couvre ce qu'il ne voit pas, un champ présent mais VIDE.
 */
describe('compare — parité FR/EN du dictionnaire entier', () => {
  it('mêmes clés de premier niveau, mêmes types, aucune valeur vide', () => {
    const fr = getCompareText('fr') as unknown as Record<string, unknown>
    const en = getCompareText('en') as unknown as Record<string, unknown>
    expect(Object.keys(fr).sort()).toEqual(Object.keys(en).sort())
    for (const key of Object.keys(fr)) {
      expect(typeof fr[key], `type de ${key}`).toBe(typeof en[key])
      if (typeof fr[key] === 'string') {
        expect((fr[key] as string).trim(), `valeur FR de ${key}`).not.toBe('')
        expect((en[key] as string).trim(), `valeur EN de ${key}`).not.toBe('')
      }
    }
  })

  it('les onze clés du profil d’armes existent dans les deux langues', () => {
    const attendues = [
      'catWeapons',
      'weaponsClassesTitle',
      'weaponsRangeKills',
      'weaponsRangeDeaths',
      'weaponsTopTitle',
      'weaponsCoverage',
      'weaponsBelowThreshold',
      'weaponsObserved',
      'weaponsNoRange',
      'weaponsPercentiles',
      'weaponsNoMeasure',
      'weaponsMatches',
    ]
    for (const locale of ['fr', 'en'] as const) {
      const t = getCompareText(locale) as unknown as Record<string, unknown>
      for (const key of attendues) expect(t[key], `${locale}.${key}`).toBeDefined()
    }
  })

  /**
   * FR SANS ANGLICISME (règle n°1 du dépôt). Le dictionnaire `compare` n'est PAS dans le
   * périmètre du garde-rail global `lib/i18n/no-anglicisms.guard.test.ts` (consigné en
   * Découverte au plan) : ce témoin local couvre au moins les libellés du profil d'armes,
   * où la tentation est maximale (« kills », « range », « top weapons »).
   */
  it('les libellés FR du profil d’armes n’emploient aucun anglicisme', () => {
    const fr = getCompareText('fr') as unknown as Record<string, unknown>
    const interdits = /\b(kills?|assists?|range|top weapons?|streak|win rate)\b/i
    const valeurs = [
      fr.catWeapons,
      fr.weaponsClassesTitle,
      fr.weaponsRangeKills,
      fr.weaponsRangeDeaths,
      fr.weaponsTopTitle,
      fr.weaponsObserved,
      fr.weaponsNoRange,
      fr.weaponsPercentiles,
      fr.weaponsNoMeasure,
      (fr.weaponsCoverage as (a: number, b: number) => string)(1, 2),
      (fr.weaponsBelowThreshold as (s: string) => string)('X'),
      (fr.weaponsMatches as (n: number) => string)(3),
    ] as string[]
    for (const v of valeurs) expect(v, `« ${v} »`).not.toMatch(interdits)
  })
})
