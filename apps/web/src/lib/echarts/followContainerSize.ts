/**
 * Un graphe ECharts suit la taille de son conteneur, sans exception.
 *
 * `echarts-for-react` fige la largeur mesurée une image après le montage (deuxième
 * `init` avec `width`/`height` explicites), puis confie le suivi à `size-sensor`, dont
 * les notifications sont regroupées (60 ms) et dont la PREMIÈRE est ignorée pour ne pas
 * couper l'animation d'entrée. Un graphe monté pendant que son conteneur change de
 * largeur (ouverture de la comparaison des Sessions : la colonne passe de la pleine page
 * à la moitié en 300 ms) voit tout ce changement fondu dans cette première notification,
 * donc ignoré : il reste à sa largeur de montage et déborde de sa carte.
 *
 * Ici, chaque changement de taille du conteneur compare la taille du DOM à celle du
 * graphe et ne redimensionne que si elles diffèrent : la première observation (tailles
 * égales) laisse l'animation d'entrée intacte, un écart réel est toujours rattrapé.
 */

/** Surface minimale d'une instance ECharts : les paquets `echarts` et `echarts-for-react` déclarent chacun leur `EChartsType`. */
export interface ResizableChart {
  getDom(): HTMLElement
  getWidth(): number
  getHeight(): number
  isDisposed(): boolean
  resize(opts: { width: 'auto'; height: 'auto' }): void
}

/** Branche le suivi de taille ; renvoie la fonction qui le débranche. */
export function followContainerSize(chart: ResizableChart): () => void {
  if (typeof ResizeObserver === 'undefined') return () => {}
  const dom = chart.getDom()
  let frame = 0
  const sync = () => {
    if (chart.isDisposed()) return
    if (dom.clientWidth !== chart.getWidth() || dom.clientHeight !== chart.getHeight()) {
      chart.resize({ width: 'auto', height: 'auto' })
    }
  }
  const observer = new ResizeObserver(() => {
    cancelAnimationFrame(frame)
    frame = requestAnimationFrame(sync)
  })
  observer.observe(dom)
  return () => {
    cancelAnimationFrame(frame)
    observer.disconnect()
  }
}
