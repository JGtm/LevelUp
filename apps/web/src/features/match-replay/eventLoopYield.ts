/**
 * eventLoopYield.ts — RENDRE LA MAIN À L'ÉVÉNEMENTIEL SANS SE FAIRE BRIDER.
 *
 * # LE PIÈGE, MESURÉ PLUTÔT QUE SUPPOSÉ (2026-08-28)
 *
 * Une boucle de calcul long doit rendre la main entre deux lots, sinon l'onglet se fige. Le
 * geste évident — `await new Promise(r => setTimeout(r, 0))` — est un PIÈGE dès que l'onglet
 * passe en arrière-plan, ce qui est précisément ce que fait un utilisateur pendant un export
 * de plusieurs minutes.
 *
 * Mesure faite dans l'onglet du rejeu, `document.hidden === true`, 20 tirages chacun :
 *
 * | Primitive         | Latence moyenne | Pire cas |
 * |-------------------|-----------------|----------|
 * | `setTimeout(…, 0)`| 673,53 ms       | 1009 ms  |
 * | `MessageChannel`  | 0,06 ms         | 1 ms     |
 *
 * Les navigateurs brident les minuteurs des onglets cachés à une seconde. `requestAnimationFrame`
 * est pire encore : il ne se déclenche PLUS DU TOUT tant que l'onglet n'est pas visible. Le
 * message d'un `MessageChannel`, lui, n'est pas un minuteur : c'est une tâche, et les tâches ne
 * sont pas bridées.
 *
 * CE QUE ÇA COÛTAIT CONCRÈTEMENT : le premier export d'essai — 30 images sur une toile de test —
 * a pris 10,2 secondes, soit dix fois le temps réel de ce qu'il encodait. Tout ce temps était
 * passé dans les attentes de contre-pression. Avec cette primitive, la même attente est gratuite.
 *
 * # POURQUOI UN MODULE, POUR SI PEU DE LIGNES
 *
 * Deux consommateurs : la contre-pression de l'encodeur (`replayVideoEncoder.ts`) et la boucle
 * d'export elle-même (`useReplayExport.ts`). Recopier le geste dans les deux, c'est garantir
 * qu'un des deux redeviendra un `setTimeout` à la première relecture distraite — et le bridage
 * ne se voit PAS en développement, où l'onglet est au premier plan.
 */

/**
 * yieldToEvents rend la main à la boucle d'événements pour une tâche, sans bridage.
 *
 * Repli sur `setTimeout` là où `MessageChannel` n'existe pas (jsdom ancien, environnements de
 * test) : le repli est bridé, mais un test ne tourne pas en onglet caché.
 */
export function yieldToEvents(): Promise<void> {
  if (typeof MessageChannel !== 'function') {
    return new Promise((resolve) => setTimeout(resolve, 0))
  }
  return new Promise((resolve) => {
    const channel = new MessageChannel()
    channel.port1.onmessage = () => {
      // LES DEUX PORTS SE FERMENT : un canal laissé ouvert par attente, sur dix-huit mille
      // images, retient dix-huit mille paires de ports jusqu'au ramasse-miettes.
      channel.port1.close()
      channel.port2.close()
      resolve()
    }
    channel.port2.postMessage(0)
  })
}
