/**
 * Garde-rail (CLAUDE.md n°6) : le bandeau « titre + aide ⓘ » ne se réécrit pas à la main.
 *
 * Contexte : les lots A1 et D (2026-09-19/21) ont fait passer les notes de pied de carte en
 * infobulle de titre. Le motif
 * `titleAdornment={(label) => <span className="flex items-center gap-1.5">{label}<InfoTooltip …/></span>}`
 * s'est alors recopié dans une douzaine de fichiers. Il est centralisé le 2026-09-21 dans
 * `components/ui/title-with-info.tsx` (`titleWithInfo`) ; une factorisation sans garde-rail
 * re-diverge (leçon du prédicat bot, 8 → 36 copies).
 *
 * CE QUE CE TEST SURVEILLE, ET SEULEMENT ÇA : un `<InfoTooltip` écrit À L'INTÉRIEUR d'une
 * valeur de prop `titleAdornment`. `InfoTooltip` reste parfaitement légitime partout
 * ailleurs — en-tête de colonne, tuile, corps de carte.
 */
import { describe, it, expect } from 'vitest'

const sources = import.meta.glob('/src/features/**/*.{ts,tsx}', {
  query: '?raw',
  import: 'default',
  eager: true,
}) as Record<string, string>

/**
 * AUCUNE ALLOWLIST, et c'est vérifié : les deux surfaces du lot A2 (`_shared/usage/`,
 * `session-detail/`) posent leurs propres helpers `cardTitleWithHint` /
 * `cardTitleAdornment`, mais sur `HeaderLabelTooltip` — un libellé porteur, PAS l'icône ⓘ.
 * Ce n'est donc pas une copie du motif surveillé ici, et rien n'a besoin d'être exempté.
 */

/** `titleAdornment={…}` et tout ce qu'il contient, accolades équilibrées. */
function titleAdornmentValues(source: string): string[] {
  const out: string[] = []
  const marker = 'titleAdornment={'
  let from = 0
  for (;;) {
    const start = source.indexOf(marker, from)
    if (start === -1) return out
    let depth = 0
    let i = start + marker.length - 1
    for (; i < source.length; i += 1) {
      if (source[i] === '{') depth += 1
      else if (source[i] === '}') {
        depth -= 1
        if (depth === 0) break
      }
    }
    out.push(source.slice(start, i + 1))
    from = i + 1
  }
}

describe('garde-rail bandeau titre + aide ⓘ (titleWithInfo source unique)', () => {
  it('le glob voit bien les sources des features', () => {
    // Un glob qui ne matche plus rien rendrait le test vert pour de mauvaises raisons.
    expect(Object.keys(sources).length).toBeGreaterThan(100)
  })

  it('aucun titleAdornment ne monte un <InfoTooltip> à la main', () => {
    const offenders = Object.entries(sources)
      .filter(([path]) => !/\.test\.tsx?$/.test(path))
      .filter(([, source]) => titleAdornmentValues(source).some((v) => v.includes('<InfoTooltip')))
      .map(([path]) => path)

    expect(offenders).toEqual([])
  })
})
