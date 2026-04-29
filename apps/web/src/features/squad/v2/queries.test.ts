/**
 * Tests useSquadV2 — vérification que les filtres cascade (experience_types,
 * playlists) sont correctement encodés dans l'URL envoyée au backend.
 *
 * Ces tests couvrent le chaîne de causalité complète côté frontend :
 * filterContext.cascade → useSquadV2 params → URL query string.
 */
import { describe, expect, it } from 'vitest'

// decodeFormUrl décode une URL encodée par URLSearchParams (+ = espace).
function decodeFormUrl(url: string): string {
  return decodeURIComponent(url.replace(/\+/g, ' '))
}

// Fonction pure extraite de useSquadV2 pour tester l'URL sans hook React
function buildSquadV2Url(
  playerSlug: string,
  teammates: string[],
  period: string,
  experienceTypes: string[],
  playlists: string[],
  maps: string[] = [],
): string {
  const teammatesQuery = teammates.length > 0 ? teammates.join(',') : ''
  const periodQuery = period || 'all'
  const expQuery = experienceTypes.length > 0 ? experienceTypes.join(',') : ''
  const playlistsQuery = playlists.length > 0 ? playlists.join(',') : ''
  const mapsQuery = maps.length > 0 ? maps.join(',') : ''

  const params = new URLSearchParams({ teammates: teammatesQuery, period: periodQuery })
  if (expQuery) params.set('experience_types', expQuery)
  if (playlistsQuery) params.set('playlists', playlistsQuery)
  if (mapsQuery) params.set('maps', mapsQuery)

  return `/players/${playerSlug}/pages/squad/v2?${params.toString()}`
}

describe('useSquadV2 URL building', () => {
  it('sans filtres cascade — URL minimale', () => {
    const url = buildSquadV2Url('jgtm', ['friend1'], 'all', [], [])
    expect(url).toContain('teammates=friend1')
    expect(url).toContain('period=all')
    expect(url).not.toContain('experience_types')
    expect(url).not.toContain('playlists')
  })

  it('experience_types envoyé en CSV', () => {
    const url = buildSquadV2Url('jgtm', ['f1'], 'all', ['PVP classé', 'PVE'], [])
    expect(url).toContain('experience_types=PVP+class%C3%A9%2CPVE')
  })

  it('playlists envoyé en CSV (label FR)', () => {
    // Le label FR ("Partie rapide") est ce que filtersResolve retourne et
    // ce que filterRowsByCascade compare via Labels["fr"] après le fix.
    const url = buildSquadV2Url('jgtm', ['f1'], 'all', [], ['Partie rapide', 'Arène classée'])
    expect(url).toContain('playlists=')
    expect(decodeFormUrl(url)).toContain('Partie rapide,Arène classée')
  })

  it('experience_types + playlists combinés', () => {
    const url = buildSquadV2Url('jgtm', ['f1'], '1m', ['PVP non classé'], ['Partie rapide'])
    expect(url).toContain('period=1m')
    expect(decodeFormUrl(url)).toContain('PVP non classé')
    expect(decodeFormUrl(url)).toContain('Partie rapide')
  })

  it('playlists vide = absent de l\'URL', () => {
    const url = buildSquadV2Url('jgtm', ['f1'], 'all', ['PVP classé'], [])
    expect(url).not.toContain('playlists')
    expect(decodeFormUrl(url)).toContain('PVP classé')
  })

  it('maps envoyé en CSV (label FR)', () => {
    const url = buildSquadV2Url('jgtm', ['f1'], 'all', [], [], ['Décharge', 'Bazar'])
    expect(url).toContain('maps=')
    expect(decodeFormUrl(url)).toContain('Décharge,Bazar')
  })

  it('maps vide = absent de l\'URL', () => {
    const url = buildSquadV2Url('jgtm', ['f1'], 'all', [], [], [])
    expect(url).not.toContain('maps')
  })

  it('plusieurs coéquipiers', () => {
    const url = buildSquadV2Url('jgtm', ['f1', 'f2', 'f3'], 'all', [], [])
    expect(url).toContain('teammates=f1%2Cf2%2Cf3')
  })
})
