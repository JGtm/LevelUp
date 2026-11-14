/**
 * Logique pure (DOM seul, sans React) de la couche des infobulles `title`.
 *
 * Pourquoi : l'infobulle native d'un attribut `title` est dessinée par le navigateur ou WebView2 dans une
 * fenêtre système séparée, que le partage d'une FENÊTRE (Discord, Teams) ne capture pas. La couche retire
 * le `title` d'un élément pendant qu'il est survolé ou focalisé au clavier, et le fait afficher par une
 * infobulle de l'application (visible dans le partage).
 *
 * Le texte retiré vit dans `data-lu-title` pendant la prise en charge : l'attribut sert aussi de marque, ce
 * qui permet de retrouver la source d'un élément déjà pris en charge quand le pointeur passe d'un de ses
 * enfants à l'autre.
 */

/** Marque posée sur le conteneur d'une infobulle JS (`Tooltip`, `InfoTooltip`) : un `title` en dessous ne la double pas. */
export const TOOLTIP_OWNER_ATTRIBUTE = 'data-lu-tooltip'

/** Texte du `title` pendant sa prise en charge. */
export const STASH_ATTRIBUTE = 'data-lu-title'

/** Marque l'`aria-label` posé temporairement pour garder le nom accessible d'un élément sans autre nom. */
const NAME_FALLBACK_ATTRIBUTE = 'data-lu-title-name'

/** Élément qui porte une infobulle native : l'élément lui-même, et le texte de son `title` (HTML) ou de son `<title>` (SVG). */
export interface TitleSource {
  owner: Element
  text: string
}

/** Enfant `<title>` direct d'un élément SVG, s'il existe. */
function svgTitleChild(el: Element): Element | null {
  if (!(el instanceof SVGElement)) return null
  for (const child of Array.from(el.children)) {
    if (child instanceof SVGElement && child.localName === 'title') return child
  }
  return null
}

/**
 * L'élément qui porterait l'infobulle native pour `start` : le plus proche ancêtre (lui compris) qui a un
 * `title` non vide, ou un enfant SVG `<title>` non vide. Comme le navigateur, un `title=""` sur un élément
 * plus proche supprime l'infobulle de ses ancêtres (résultat `null`). Un élément déjà pris en charge est
 * reconnu à sa marque `data-lu-title`.
 */
export function findTitleSource(start: Element | null): TitleSource | null {
  for (let el: Element | null = start; el; el = el.parentElement) {
    const stash = el.getAttribute(STASH_ATTRIBUTE)
    if (stash !== null) return { owner: el, text: stash }
    if (el instanceof SVGElement) {
      const text = svgTitleChild(el)?.textContent ?? ''
      if (text.trim() !== '') return { owner: el, text }
    } else {
      const title = el.getAttribute('title')
      if (title !== null) return title.trim() === '' ? null : { owner: el, text: title }
    }
  }
  return null
}

/** Un `title` sous une infobulle JS déjà en place : à retirer, jamais à afficher une seconde fois. */
export function isCoveredByTooltip(owner: Element): boolean {
  return owner.closest(`[${TOOLTIP_OWNER_ATTRIBUTE}]`) !== null
}

/**
 * Faut-il poser un `aria-label` le temps de la prise en charge ? Oui quand le `title` est le SEUL nom de
 * l'élément (bouton icône, SVG) : le retirer le rendrait muet pour un lecteur d'écran. Non quand l'élément a
 * déjà un nom (`aria-label`, `aria-labelledby`, texte, image décrite, étiquette de champ).
 */
function needsNameFallback(owner: Element): boolean {
  if (owner.hasAttribute('aria-label') || owner.hasAttribute('aria-labelledby')) return false
  if (owner instanceof SVGElement) return true
  if (
    (owner instanceof HTMLInputElement || owner instanceof HTMLSelectElement || owner instanceof HTMLTextAreaElement) &&
    owner.labels !== null &&
    owner.labels.length > 0
  ) {
    return false
  }
  if ((owner.textContent ?? '').trim() !== '') return false
  return owner.querySelector('[alt]:not([alt=""]), [aria-label]') === null
}

/**
 * Prend en charge le `title` de `source` : le texte passe en `data-lu-title`, l'infobulle native disparaît.
 * Idempotent (un élément déjà pris en charge n'est pas retouché).
 */
export function claimTitle(source: TitleSource): void {
  const { owner, text } = source
  if (owner.hasAttribute(STASH_ATTRIBUTE)) return
  const nameless = needsNameFallback(owner)
  owner.setAttribute(STASH_ATTRIBUTE, text)
  const svgTitle = svgTitleChild(owner)
  if (svgTitle) svgTitle.textContent = ''
  else owner.removeAttribute('title')
  if (nameless) {
    owner.setAttribute('aria-label', text.trim())
    owner.setAttribute(NAME_FALLBACK_ATTRIBUTE, '')
  }
}

/**
 * Rend à `owner` son `title`. Si l'application a réécrit le titre entre-temps (rendu React pendant le
 * survol), la valeur neuve gagne : le texte retiré n'écrase jamais un titre posé après lui.
 */
export function releaseTitle(owner: Element): void {
  const stash = owner.getAttribute(STASH_ATTRIBUTE)
  if (stash === null) return
  const svgTitle = svgTitleChild(owner)
  if (svgTitle) {
    if ((svgTitle.textContent ?? '') === '') svgTitle.textContent = stash
  } else if (!owner.hasAttribute('title')) {
    owner.setAttribute('title', stash)
  }
  if (owner.hasAttribute(NAME_FALLBACK_ATTRIBUTE)) {
    if (owner.getAttribute('aria-label') === stash.trim()) owner.removeAttribute('aria-label')
    owner.removeAttribute(NAME_FALLBACK_ATTRIBUTE)
  }
  owner.removeAttribute(STASH_ATTRIBUTE)
}

/** Relie `owner` à l'infobulle affichée (`aria-describedby`), sans perdre les références déjà posées. */
export function addDescribedBy(owner: Element, id: string): void {
  const tokens = (owner.getAttribute('aria-describedby') ?? '').split(/\s+/).filter(Boolean)
  if (!tokens.includes(id)) tokens.push(id)
  owner.setAttribute('aria-describedby', tokens.join(' '))
}

/** Retire la référence posée par `addDescribedBy`, et l'attribut s'il ne reste rien. */
export function removeDescribedBy(owner: Element, id: string): void {
  const tokens = (owner.getAttribute('aria-describedby') ?? '').split(/\s+/).filter((t) => t !== '' && t !== id)
  if (tokens.length === 0) owner.removeAttribute('aria-describedby')
  else owner.setAttribute('aria-describedby', tokens.join(' '))
}
