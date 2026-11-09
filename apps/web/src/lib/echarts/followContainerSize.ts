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
 *
 * Pendant une transition de mise en page (`lib/layout/layoutSettling` : ouverture ou fermeture du
 * volet ancré), les mesures sont ignorées : la largeur change à chaque image et redessiner le graphe
 * à chacune coûte cher pour rien. À la fin de la transition, une seule synchronisation rattrape la
 * taille finale.
 */
import { isLayoutSettling, onLayoutSettled } from '@/lib/layout/layoutSettling'

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
    if (chart.isDisposed() || isLayoutSettling()) return
    if (dom.clientWidth !== chart.getWidth() || dom.clientHeight !== chart.getHeight()) {
      chart.resize({ width: 'auto', height: 'auto' })
    }
  }
  const schedule = () => {
    cancelAnimationFrame(frame)
    frame = requestAnimationFrame(sync)
  }
  const observer = new ResizeObserver(schedule)
  observer.observe(dom)
  const offSettled = onLayoutSettled(schedule)
  return () => {
    cancelAnimationFrame(frame)
    observer.disconnect()
    offSettled()
  }
}
