/**
 * Mesures de piste de carrousel pour jsdom, qui ne mesure rien : pose `scrollWidth`, `clientWidth`,
 * `offsetWidth`, `scrollLeft` (lus à chaque accès dans `state`, donc modifiables en cours de test)
 * et un `offsetLeft` de 300 px par rang d'enfant.
 */
export interface TrackMetricsState {
  scrollWidth: number
  clientWidth: number
  scrollLeft: number
}

export function mockTrackMetrics(state: TrackMetricsState): () => void {
  const proto = HTMLElement.prototype as unknown as Record<string, unknown>
  const getters: Record<string, () => unknown> = {
    scrollWidth: () => state.scrollWidth,
    clientWidth: () => state.clientWidth,
    offsetWidth: () => state.clientWidth,
    scrollLeft: () => state.scrollLeft,
  }
  const saved = new Map<string, PropertyDescriptor | undefined>()
  const define = (key: string, get: (this: HTMLElement) => unknown) => {
    saved.set(key, Object.getOwnPropertyDescriptor(proto, key))
    Object.defineProperty(proto, key, { configurable: true, get, set: () => {} })
  }
  for (const [key, get] of Object.entries(getters)) define(key, get)
  define('offsetLeft', function (this: HTMLElement) {
    const parent = this.parentElement
    return parent ? Array.from(parent.children).indexOf(this) * 300 : 0
  })
  return () => {
    for (const [key, desc] of saved) {
      if (desc) Object.defineProperty(proto, key, desc)
      else delete proto[key]
    }
  }
}
