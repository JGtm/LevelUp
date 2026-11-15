/**
 * pickFolder — la boîte « Parcourir » du système, demandée à la coquille de bureau (ADR 0045).
 *
 * La page servie par le sidecar appelle UNE commande de la coquille, `pick_folder`, par le pont que
 * Tauri injecte dans la page (`window.__TAURI_INTERNALS__.invoke`) : aucune dépendance `@tauri-apps/api`.
 * La coquille n'autorise cet appel qu'à cette page (origine `http://127.0.0.1:*`, fenêtre principale).
 *
 * Le chemin rendu est une chaîne OPAQUE, telle que le système l'a donnée : le web ne la découpe, ne la
 * normalise ni ne la compare. Le serveur la valide à l'enregistrement, comme une saisie.
 *
 * Ce module ne décide pas s'il faut l'offrir : le bouton se branche sur la capacité `can_pick_folder`
 * du bootstrap (`FolderPickButton`), jamais sur le système ni sur la présence du pont.
 */

/** Nom de la commande de la coquille (`pick_folder` côté Rust). */
export const PICK_FOLDER_COMMAND = 'pick_folder'

type Invoke = (command: string, args?: Record<string, unknown>) => Promise<unknown>

/** Le pont de la coquille, ou rien quand la page est ouverte hors de l'application de bureau. */
function shellInvoke(): Invoke | null {
  const internals: unknown = Reflect.get(window, '__TAURI_INTERNALS__')
  if (typeof internals !== 'object' || internals === null) return null
  const invoke: unknown = Reflect.get(internals, 'invoke')
  return typeof invoke === 'function' ? (invoke as Invoke) : null
}

/**
 * Ouvre la boîte de choix de dossier. `initial` : la valeur courante du champ, d'où la boîte part.
 * Rend le chemin choisi, ou `null` quand l'utilisateur annule ou que l'appel échoue (l'échec est
 * journalisé) : dans les deux cas le champ reste inchangé.
 */
export async function pickFolder(initial: string): Promise<string | null> {
  const invoke = shellInvoke()
  if (!invoke) {
    console.warn('[pickFolder] le pont de la coquille est absent : page ouverte hors de l’application de bureau')
    return null
  }
  try {
    const chosen = await invoke(PICK_FOLDER_COMMAND, { initial })
    return typeof chosen === 'string' && chosen !== '' ? chosen : null
  } catch (err) {
    console.error('[pickFolder] la coquille a refusé ou échoué', err)
    return null
  }
}
