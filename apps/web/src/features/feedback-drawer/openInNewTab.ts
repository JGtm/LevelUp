/**
 * Ouvre une URL dans un nouvel onglet par un lien `target="_blank"` cliqué par programme.
 *
 * Pas de `window.open(url, '_blank', 'noopener')` : avec `noopener` il rend toujours `null`, ce qui ne
 * permet pas de savoir si l'onglet s'est ouvert, et dans l'application de bureau la fenêtre refuse la
 * seconde fenêtre (le lien part dans le navigateur par défaut, voir la politique de navigation de la
 * coquille). Un lien cliqué dans le geste de l'utilisateur n'est pas pris pour une fenêtre surgissante.
 */
export function openInNewTab(url: string): void {
  const link = document.createElement('a')
  link.href = url
  link.target = '_blank'
  link.rel = 'noopener noreferrer'
  link.hidden = true
  document.body.appendChild(link)
  link.click()
  link.remove()
}
