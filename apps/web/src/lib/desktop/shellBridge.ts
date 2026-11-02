/**
 * shellBridge — le seul canal entre la page du serveur et la coquille de bureau (ADR 0045).
 *
 * La page tourne dans WebView2 sans aucune commande Tauri : la coquille n'expose à la page que le
 * protocole embarqué `levelup-shell`, que Windows adresse par `http://levelup-shell.localhost`. Une
 * demande est un GET sans lecture de réponse (`no-cors`) : la coquille agit — boîte native, mise à
 * jour — et la page reste où elle est (aucune navigation, donc aucun rechargement ni cookie à
 * rejouer).
 */

/** Origine du protocole embarqué de la coquille, côté WebView2 (Windows seul). */
const SHELL_ORIGIN = 'http://levelup-shell.localhost'

/** Chemin qui lance la recherche de mise à jour dans la coquille. */
export const SHELL_CHECK_UPDATE_PATH = '/check-update'

/**
 * Demande à la coquille de chercher une mise à jour. Le résultat (à jour, proposée, échec) est
 * montré par la coquille dans une boîte de dialogue native. Rend vrai quand la demande est partie,
 * faux quand la coquille est injoignable (page ouverte hors de l'application de bureau).
 */
export async function requestUpdateCheck(): Promise<boolean> {
  try {
    await fetch(`${SHELL_ORIGIN}${SHELL_CHECK_UPDATE_PATH}`, { mode: 'no-cors', cache: 'no-store' })
    return true
  } catch (err) {
    console.warn('[shellBridge] la coquille est injoignable', err)
    return false
  }
}
