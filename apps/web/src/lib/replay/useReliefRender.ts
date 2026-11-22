/**
 * useReliefRender — LE RELIEF D'UNE CARTE, CHARGÉ DANS LE CALCULATEUR ET RENDU À LA DEMANDE.
 *
 * Le hook range les deux gestes que toute vue du relief refait : poser la carte dans un calculateur
 * (`createReliefComputer` : un worker, ou son repli synchrone), puis redemander un rendu à chaque
 * changement de la demande (orientation, coupe, base de teinte, traits, masques). Il ne sait rien du
 * dessin : il rend la dernière `ReliefFrame` et l'état de la lecture.
 *
 *   - `idle`    : rien à calculer (pas de fichiers, ou aucune commande ne se sert du relief) ;
 *   - `loading` : l'image se décode et se charge dans le calculateur ;
 *   - `ready`   : la carte est chargée, `frame` est la dernière image rendue (la précédente reste en
 *                 place pendant qu'une nouvelle se calcule : un curseur qui glisse ne clignote pas) ;
 *   - `failed`  : une `ReliefError` (ou une panne du calculateur) a eu lieu ; `error` la porte. Elle est
 *                 JOURNALISÉE une fois, et l'appelant retombe sur le plan : l'état reste `failed` jusqu'à
 *                 ce que les fichiers changent. Une erreur qui arrive après le démontage (ou le changement de
 *                 carte) n'est ni journalisée ni retenue : le calculateur fermé rejette, aucun repli n'a lieu.
 *
 * L'IMAGE DES HAUTEURS EST DÉCODÉE À CHAQUE CHARGEMENT dans le calculateur (ses tableaux lui sont
 * CÉDÉS, `ReliefComputer.load`) : le cache de requêtes ne garde que le `Blob`.
 */
import { useEffect, useRef, useState } from 'react'

import {
  createReliefComputer,
  readReliefImage,
  type ReliefComputer,
  type ReliefData,
  type ReliefFrame,
  type ReliefInfo,
  type ReliefRenderRequest,
} from './relief'

/** Où en est la lecture du relief. */
export type ReliefRenderStatus = 'idle' | 'loading' | 'ready' | 'failed'

/** Ce que le hook rend. */
export interface ReliefRender {
  status: ReliefRenderStatus
  /** Ce que le moteur dit de la carte chargée (`hasMesh`) ; `null` avant le chargement. */
  info: ReliefInfo | null
  /** La dernière image rendue ; `null` avant la première, ou après un changement de carte. */
  frame: ReliefFrame | null
  /** L'erreur d'un état `failed`. */
  error: unknown
}

const IDLE: ReliefRender = { status: 'idle', info: null, frame: null, error: null }

/** Le journal d'un repli, au préfixe des autres journaux du rejeu. */
function journal(what: string, err: unknown): void {
  console.warn(`[replay-relief] ${what}, repli sur le plan : ${err instanceof Error ? err.message : String(err)}`)
}

/**
 * useReliefRender charge `data` + `blob` dans un calculateur et rend la frame de `request`.
 * `request = null` : aucun rendu n'est demandé (le chargement a lieu quand même, si les fichiers sont là).
 */
export function useReliefRender(
  data: ReliefData | null,
  blob: Blob | null,
  request: ReliefRenderRequest | null,
): ReliefRender {
  const computerRef = useRef<ReliefComputer | null>(null)
  const [loaded, setLoaded] = useState<{ info: ReliefInfo; computer: ReliefComputer } | null>(null)
  const [frame, setFrame] = useState<ReliefFrame | null>(null)
  const [failure, setFailure] = useState<{ error: unknown; source: ReliefData | null } | null>(null)

  // LE CHARGEMENT : un calculateur par carte, jeté avec elle.
  useEffect(() => {
    if (!data || !blob) return
    let alive = true
    const computer = createReliefComputer()
    computerRef.current = computer
    void (async () => {
      try {
        const image = await readReliefImage(blob, data.calibration, data.height)
        const info = await computer.load({ image, data })
        if (alive) setLoaded({ info, computer })
      } catch (err) {
        // Une vue quittée ferme le calculateur : la lecture en cours rejette, ce n'est pas un repli.
        if (!alive) return
        journal('chargement du relief impossible', err)
        setFailure({ error: err, source: data })
      }
    })()
    return () => {
      alive = false
      computer.dispose()
      computerRef.current = null
      setLoaded(null)
      setFrame(null)
    }
  }, [data, blob])

  // LE RENDU : la dernière demande gagne, une demande remplacée rend `null`.
  const { turns, cut, base, lines, masks } = request ?? { turns: 0, cut: null, base: null, lines: false, masks: false }
  const asked = request !== null
  useEffect(() => {
    if (!loaded || !asked) return
    let alive = true
    loaded.computer
      .render({ turns, cut, base, lines, masks })
      .then((next) => {
        if (alive && next) setFrame(next)
      })
      .catch((err: unknown) => {
        if (!alive) return
        journal('rendu du relief impossible', err)
        setFailure({ error: err, source: data })
      })
    return () => {
      alive = false
    }
  }, [loaded, asked, turns, cut, base, lines, masks, data])

  if (failure !== null && failure.source === data) return { status: 'failed', info: null, frame: null, error: failure.error }
  if (!data || !blob) return IDLE
  if (!loaded) return { status: 'loading', info: null, frame: null, error: null }
  return { status: 'ready', info: loaded.info, frame, error: null }
}
