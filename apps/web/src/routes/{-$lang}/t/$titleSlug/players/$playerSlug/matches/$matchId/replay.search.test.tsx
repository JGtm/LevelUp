/**
 * replay.search.test.tsx — LE PARAMÈTRE D'INSTANT `?t=` DU REJEU SURVIT À UN LIEN ÉCRIT À LA MAIN.
 *
 * Le routeur lit chaque valeur de recherche en JSON : `?t=612000` arrive comme le NOMBRE 612000,
 * alors que les liens de l'app (`MatchReplayLink`) arrivent cités (`?t=%224200%22`, la chaîne
 * "4200"). Les deux formes doivent ouvrir le rejeu au même instant ; seule une valeur
 * inexploitable retombe sur « pas d'instant » (rejeu ouvert au début, jamais une page en erreur).
 *
 * Le test passe par le VRAI chemin : l'analyseur de recherche par défaut du routeur, puis le
 * `validateSearch` de la route elle-même — pas une copie du schéma.
 */
import { describe, expect, it } from 'vitest'
import { defaultParseSearch, defaultStringifySearch } from '@tanstack/react-router'

import { Route } from './replay'

/** Ce que la page lit via `Route.useSearch()` pour une chaîne de recherche d'URL donnée. */
function rechercheValidee(chaine: string) {
  const schema = Route.options.validateSearch as { parse: (v: unknown) => unknown }
  return schema.parse(defaultParseSearch(chaine))
}

describe('rejeu — paramètre d’instant `t`', () => {
  it('un `t` numérique écrit à la main est conservé et converti en chaîne', () => {
    expect(defaultParseSearch('?t=612000&clock=match')).toEqual({ t: 612000, clock: 'match' })
    expect(rechercheValidee('?t=612000&clock=match')).toEqual({ t: '612000', clock: 'match' })
  })

  it('le lien produit par l’app (chaîne citée) donne la même valeur', () => {
    const chaine = defaultStringifySearch({ t: '612000', clock: 'match' })
    expect(rechercheValidee(chaine)).toEqual({ t: '612000', clock: 'match' })
  })

  it('une valeur inexploitable retombe sur « pas d’instant », sans erreur', () => {
    expect(rechercheValidee('?t=true&clock=video')).toEqual({ t: undefined, clock: undefined })
    expect(rechercheValidee('')).toEqual({})
  })
})
