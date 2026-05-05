import { useEffect, useState } from 'react'

/**
 * useThemeVersion — renvoie un compteur qui s'incrémente quand l'attribut
 * `data-theme` change sur `<html>`. À utiliser comme dépendance d'un
 * `useMemo` qui génère une option ECharts pour forcer le recalcul (et donc
 * le re-render canvas) lors d'un toggle de thème light/dark.
 *
 * @example
 *   const themeVersion = useThemeVersion()
 *   const option = useMemo(() => buildChartOption(data, getEChartsThemeColors()), [data, themeVersion])
 */
export function useThemeVersion(): number {
  const [version, setVersion] = useState(0)
  useEffect(() => {
    if (typeof document === 'undefined') return
    const observer = new MutationObserver((mutations) => {
      for (const m of mutations) {
        if (m.type === 'attributes' && m.attributeName === 'data-theme') {
          setVersion((v) => v + 1)
          return
        }
      }
    })
    observer.observe(document.documentElement, { attributes: true })
    return () => observer.disconnect()
  }, [])
  return version
}
