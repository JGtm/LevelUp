/**
 * loadedImage.ts — LES IMAGES DES VIGNETTES DU REJEU, chargées une fois par URL.
 *
 * # POURQUOI (2026-09-16, décision D9 du plan « formats vidéo »)
 *
 * Les vignettes de grenade, d'arme au sol et de socle sont TEINTES à une encre du thème, et se
 * reteignent quand l'encre change. Elles le faisaient en RECHARGEANT l'image : la vignette
 * manquait le temps d'un `onload`. Sans conséquence tant que l'encre ne changeait qu'avec le
 * thème ; mais l'export bascule les encres au thème sombre à son entrée, et ses premières images
 * seraient sorties SANS vignettes — ou avec, selon la vitesse du cache du navigateur.
 *
 * Une image déjà chargée est donc rendue SYNCHRONEMENT : la reteinte a lieu dans le même passage
 * d'effets que le reste du changement d'encres, avant que l'export ne peigne sa première image.
 *
 * # LE SEUL `new Image()` DES CALQUES
 *
 * Quatre hooks recopiaient ce chargement (grenades, armes au sol, socles, véhicules). Il vit ici,
 * et le garde-rail `loadedImage.guard.test.ts` interdit d'en réécrire une copie.
 */

const images = new Map<string, HTMLImageElement>()

/**
 * withLoadedImage — appelle `ready` avec l'image de `url` : tout de suite si elle est déjà
 * chargée, à son chargement sinon. Une image qui échoue n'appelle que `failed` s'il est fourni
 * (le calque garde son tracé de repli) ; elle n'est pas mise en cache, un appel suivant la retente.
 */
export function withLoadedImage(
  url: string,
  ready: (image: HTMLImageElement) => void,
  failed?: () => void,
): void {
  const cached = images.get(url)
  if (cached) {
    ready(cached)
    return
  }
  const im = new Image()
  im.onload = () => {
    images.set(url, im)
    ready(im)
  }
  if (failed) im.onerror = failed
  im.src = url
}
