/**
 * useColorPaletteVersion — retourne un entier qui s'incrémente à chaque
 * changement de palette CSS (MutationObserver sur :root style).
 *
 * Usage dans les composants Plotly :
 *   const paletteVersion = useColorPaletteVersion()
 *   const { traces, layout } = useMemo(() => buildLayout(), [rows, paletteVersion])
 *
 * Cela force le recalcul des colorscales (resolveToken) quand la palette change.
 */
import { useState, useEffect } from 'react'

export function useColorPaletteVersion(): number {
  const [version, setVersion] = useState(0)

  useEffect(() => {
    const root = document.documentElement
    const observer = new MutationObserver((mutations) => {
      for (const m of mutations) {
        if (m.type === 'attributes' && m.attributeName === 'style') {
          setVersion((v) => v + 1)
          break
        }
      }
    })
    observer.observe(root, { attributes: true, attributeFilter: ['style'] })
    return () => observer.disconnect()
  }, [])

  return version
}
