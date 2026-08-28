/**
 * useHlsVideo — BRANCHER UN <video> SUR UN FLUX HLS, une fois pour tout le dépôt.
 *
 * POURQUOI UN HELPER ET PAS UNE SECONDE COPIE. Les clips transcodés voient leur `file_path`
 * muer vers un `master.m3u8` : un `<video src>` nu ne les lit PAS sur Chrome/Firefox, qui
 * n'ont pas de lecteur HLS natif. La galerie avait résolu le problème (coverflow, 2026-06) ;
 * la lightbox du rejeu se posait exactement la même question. Deux copies de cette attache
 * auraient divergé sur le premier quirk de navigateur corrigé d'un seul côté — et le quirk
 * ci-dessous prouve qu'il y en a.
 *
 * LE QUIRK CHROME (incident 2026-06-14) : `canPlayType('application/vnd.apple.mpegurl')`
 * renvoie « maybe » — truthy — alors que Chrome n'expose PAS `video.audioTracks`. Prendre le
 * lecteur natif en premier lisait donc la vidéo sans jamais peupler le sélecteur de pistes.
 * hls.js prime dès que MSE est disponible ; le natif n'est qu'un repli (Safari/iOS).
 *
 * LES RAPPELS SONT LUS DANS UNE REF, et l'effet ne dépend que de la source : un appelant qui
 * passe une lambda inline reconstruirait sinon l'instance hls.js à chaque rendu — c'est-à-dire
 * détruirait le flux en cours de lecture.
 *
 * Garde-rail : `hlsSingleImport.guard.test.ts` interdit tout autre import de `hls.js`.
 */
import Hls from 'hls.js'
import { useEffect, useRef, type RefObject } from 'react'

/** Une piste audio alternée du master, telle que l'appelant l'affiche. */
export interface HlsAudioTrack {
  id: number
  name: string
}

/** Ce qui peut empêcher la lecture : navigateur sans MSE ni HLS natif, ou flux mort. */
export type HlsFailure = 'unsupported' | 'fatal'

/** Détail d'une erreur fatale, tel que hls.js le rapporte (pour le journal). */
export interface HlsFailureDetail {
  type?: string
  details?: string
}

export interface UseHlsVideoOptions {
  /** L'élément à alimenter. Le hook ne fait rien tant qu'il est absent. */
  videoRef: RefObject<HTMLVideoElement | null>
  /** La source du média : tout ce qui n'est pas un `.m3u8` est laissé au `<video src>`. */
  src: string
  /**
   * `false` : le manifest est lu (les pistes audio remontent) mais AUCUN segment n'est
   * téléchargé avant un `startLoad()` de l'appelant — ce que fait le coverflow, qui monte
   * jusqu'à cinq lecteurs et n'en écoute qu'un. Défaut `true` : un lecteur seul charge.
   */
  autoStartLoad?: boolean
  /** Pistes audio du master (vide au démontage). Non appelé quand le master n'en expose pas. */
  onAudioTracks?: (tracks: HlsAudioTrack[], active: number) => void
  /** Échec de lecture. L'appelant décide du message et du journal. */
  onFailure?: (kind: HlsFailure, detail?: HlsFailureDetail) => void
}

/** Une source est HLS si son chemin (hors query string) se termine par .m3u8. */
export function isHlsSource(path: string): boolean {
  return path.split('?')[0].toLowerCase().endsWith('.m3u8')
}

/**
 * Attache hls.js au `<video>` quand la source est un flux, et le détache proprement. Rend
 * l'instance (pour piloter `startLoad`/`stopLoad` ou changer de piste audio) et le verdict
 * `isHls`, dont l'appelant a besoin pour NE PAS poser d'attribut `src` sur son élément.
 */
export function useHlsVideo(o: UseHlsVideoOptions): {
  isHls: boolean
  hlsRef: RefObject<Hls | null>
} {
  const { videoRef, src, autoStartLoad = true } = o
  const hlsRef = useRef<Hls | null>(null)
  const isHls = isHlsSource(src)

  // Les rappels changent d'identité à chaque rendu chez la plupart des appelants ; les lire
  // dans une ref garde l'effet d'attache dépendant de la SOURCE, et d'elle seule.
  const callbacks = useRef({ onAudioTracks: o.onAudioTracks, onFailure: o.onFailure })
  // La mise à jour se fait dans un effet, PAS pendant le rendu (écrire une ref pendant le
  // rendu est un effet de bord sur un rendu que React peut rejouer). Déclaré AVANT l'effet
  // d'attache : les effets s'exécutent dans l'ordre, les rappels sont donc à jour quand une
  // source change. Au tout premier rendu, ce sont ceux passés à `useRef` qui servent.
  useEffect(() => {
    callbacks.current = { onAudioTracks: o.onAudioTracks, onFailure: o.onFailure }
  })

  useEffect(() => {
    if (!isHls) return
    const video = videoRef.current
    if (!video) return

    if (!Hls.isSupported()) {
      // Repli natif (Safari/iOS) : le lecteur du système sait lire un master.
      if (video.canPlayType('application/vnd.apple.mpegurl')) {
        video.src = src
        return
      }
      callbacks.current.onFailure?.('unsupported')
      return
    }

    const hls = new Hls({ enableWorker: true, autoStartLoad })
    hlsRef.current = hls
    hls.loadSource(src)
    hls.attachMedia(video)
    // Les pistes alternées arrivent par AUDIO_TRACKS_UPDATED : à MANIFEST_PARSED,
    // `hls.audioTracks` est encore vide (vérifié en navigateur).
    hls.on(Hls.Events.AUDIO_TRACKS_UPDATED, (_evt, data) => {
      const tracks = data.audioTracks.map((t, i) => ({
        id: i,
        name: t.name || t.lang || `Audio ${i + 1}`,
      }))
      callbacks.current.onAudioTracks?.(tracks, hls.audioTrack)
    })
    hls.on(Hls.Events.ERROR, (_evt, data) => {
      if (data.fatal) callbacks.current.onFailure?.('fatal', { type: data.type, details: data.details })
    })
    return () => {
      hls.destroy()
      hlsRef.current = null
      // Plus d'instance, plus de pistes : l'appelant qui les affiche doit les oublier.
      callbacks.current.onAudioTracks?.([], -1)
    }
  }, [videoRef, src, isHls, autoStartLoad])

  return { isHls, hlsRef }
}
