/**
 * CoverFlowModal — modale plein écran avec carrousel coverflow pour les médias.
 *
 * P8.5 (revue 2026-04-29 gap #14) : promu de `components/ui/` vers
 * `features/media/` car le composant est strictement media-specific
 * (importait MediaLikeButton + getMediaModalsText). La frontière inversée
 * `components/ → features/` est désormais respectée.
 */
'use client'

import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import Hls from 'hls.js'
import type { MediaItemRow } from '@/lib/api/types'
import { useAppShellStore } from '@/stores/appShellStore'
import { intlLocale } from '@/lib/formatters'
import type { ManifestLocale } from '@/lib/i18n/format'
import { MediaLikeButton } from './MediaViewer'
import { getMediaModalsText } from './i18n-modals'
import { log } from './_logger'

interface CoverFlowModalProps {
  items: MediaItemRow[]
  onToggleLike: (item: MediaItemRow) => void
  startIndex: number
  onClose: () => void
  likeDisabled?: boolean
  hasNextPage?: boolean
  onLoadNextPage?: () => void
  globalIndexOffset?: number
  globalTotal?: number
  /** Callback ouvrant le picker d'(ré)association. Le label affiché passe
   *  automatiquement de "Associer" à "Réassocier" selon `item.match_id`. */
  onReassociate?: (item: MediaItemRow) => void
  /** Gamertag du joueur actif — masque le bouton (ré)association sur les médias
   *  qui ne lui appartiennent pas. */
  currentPlayerGamertag?: string
  /** Slug du joueur courant pour construire le lien "Voir le match". */
  playerSlug?: string
  /** Si renseigné et égal à `currentItem.match_id`, supprime le lien
   *  "Voir le match" (on est déjà sur la page de ce match). */
  currentMatchId?: string | null
  /** Callback pour naviguer vers la page du match. Si fourni, remplace le
   *  comportement par défaut (lien `<a>`) pour permettre au consommateur
   *  d'attacher un MatchNavContext (cf. useNavigateToMatch). */
  onOpenMatch?: (matchId: string) => void
  autoChain?: boolean
  onToggleAutoChain?: () => void
}

type SlotPos = { x: string; scale: number; opacity: number }
const POSITIONS: Record<number, SlotPos> = {
  [-2]: { x: '-115%', scale: 0.55, opacity: 0 },
  [-1]: { x: '-55%', scale: 0.7, opacity: 0.45 },
  [0]: { x: '0%', scale: 1, opacity: 1 },
  [1]: { x: '55%', scale: 0.7, opacity: 0.45 },
  [2]: { x: '115%', scale: 0.55, opacity: 0 },
}

const ANIM_MS = 500
const ANIM_EASE = 'cubic-bezier(0.32, 0.72, 0, 1)'
const WINDOW_RADIUS = 2
const IMAGE_AUTOCHAIN_DELAY_MS = 7000

function formatHeading(item: MediaItemRow, index: number, total: number, locale: ManifestLocale) {
  // Format court HH:MM JJ/MM/AA (cohérent avec formatMediaDate des thumbnails).
  const raw = item.capture_end_utc ?? item.match_start_time
  let dateStr: string | null = null
  if (raw) {
    const d = new Date(raw)
    if (!Number.isNaN(d.getTime())) {
      const loc = intlLocale(locale)
      const datePart = d.toLocaleDateString(loc, { day: '2-digit', month: '2-digit', year: '2-digit' })
      const timePart = d.toLocaleTimeString(loc, { hour: '2-digit', minute: '2-digit' })
      dateStr = `${timePart} ${datePart}`
    }
  }
  return [
    `${index + 1}/${total}`,
    item.map_name,
    dateStr,
  ].filter(Boolean).join(' · ')
}

interface ClipPlayerProps {
  filePath: string
  basename: string | null
  isCenter: boolean
  relPos: number
  videoRef: (el: HTMLVideoElement | null) => void
  onEnded: (() => void) | undefined
  /** Libellés des deux interrupteurs audio (layout multipiste game/voices/full). */
  audioLabels: { game: string; voice: string; group: string }
}

/** Une source est HLS si son chemin (hors query string) se termine par .m3u8. */
function isHlsSource(path: string): boolean {
  return path.split('?')[0].toLowerCase().endsWith('.m3u8')
}

/**
 * Wrapper <video> qui lit :
 *   - les flux HLS (.m3u8) via hls.js (ou le lecteur natif Safari/iOS), avec
 *     sélecteur de piste audio quand le master expose plusieurs pistes ;
 *   - les fichiers web-natifs (mp4/webm/mov) en lecture directe.
 * Affiche un message clair en cas d'erreur plutôt qu'un cadre noir vide.
 */
function ClipPlayer({ filePath, basename, isCenter, relPos, videoRef, onEnded, audioLabels }: ClipPlayerProps) {
  const [error, setError] = useState<string | null>(null)
  const [lastFilePath, setLastFilePath] = useState(filePath)
  const videoElRef = useRef<HTMLVideoElement | null>(null)
  const hlsRef = useRef<Hls | null>(null)
  const [audioTracks, setAudioTracks] = useState<{ id: number; name: string }[]>([])
  const [activeAudio, setActiveAudio] = useState(-1)
  // Deux interrupteurs indépendants Jeu/Voix (les deux ON par défaut). N'ont de
  // sens que sur le layout multipiste game/voices/full. ATTENTION : la key est
  // portée par la div de SLOT (côté parent), pas par ClipPlayer — l'instance
  // PERSISTE donc tant que le clip reste dans la fenêtre de proximité (±2), et
  // l'état des interrupteurs NE se réinitialise PAS à la navigation de voisinage.
  // Il est réappliqué au recentrage par l'effet ci-dessous (dep isCenter) et le
  // parent respecte le mute "deux OFF" via le marqueur data-audio-off.
  const [gameOn, setGameOn] = useState(true)
  const [voiceOn, setVoiceOn] = useState(true)

  if (filePath !== lastFilePath) {
    setLastFilePath(filePath)
    setError(null)
  }

  const isHls = isHlsSource(filePath)

  // Callback ref : alimente la ref locale (pour attacher hls.js) ET le callback
  // parent (qui pilote play/pause/mute via sa Map de refs).
  const setRefs = useCallback(
    (el: HTMLVideoElement | null) => {
      videoElRef.current = el
      videoRef(el)
    },
    [videoRef],
  )

  // Attache hls.js pour les sources .m3u8 (sauf Safari/iOS qui lisent HLS nativement).
  useEffect(() => {
    if (!isHls) return
    const video = videoElRef.current
    if (!video) return

    // Préférer hls.js dès que MSE est disponible (Chrome/Firefox/Edge) : c'est le
    // SEUL chemin qui peuple le sélecteur de pistes audio (via AUDIO_TRACKS_UPDATED).
    // Le natif ne sert que de repli (Safari/iOS sans MSE). PIÈGE (incident 2026-06-14) :
    // Chrome renvoie "maybe" à canPlayType('application/vnd.apple.mpegurl') — truthy —
    // MAIS n'expose pas video.audioTracks. Prendre le natif en premier lisait la vidéo
    // sans jamais afficher le sélecteur de pistes (game/voix ou Track1..N).
    if (!Hls.isSupported()) {
      if (video.canPlayType('application/vnd.apple.mpegurl')) {
        video.src = filePath
        return
      }
      log.warn('hls:unsupported', 'Lecture HLS non supportée par ce navigateur', { filePath })
      setError('Lecture HLS non supportée par ce navigateur')
      return
    }

    // autoStartLoad:false → loadSource charge le manifest (ce qui peuple le
    // sélecteur de pistes via AUDIO_TRACKS_UPDATED, y compris pour les voisins)
    // mais NE télécharge AUCUN segment tant que startLoad() n'est pas appelé.
    // Le chargement des segments est réservé au clip centré (effet dédié
    // ci-dessous) : sinon les jusqu'à 5 slots HLS rendus (±2) préchargeraient
    // ~30 s de segments en parallèle (l'attribut preload du <video> est sans
    // effet en MSE).
    const hls = new Hls({ enableWorker: true, autoStartLoad: false })
    hlsRef.current = hls
    hls.loadSource(filePath)
    hls.attachMedia(video)
    // Les pistes audio alternées sont peuplées via AUDIO_TRACKS_UPDATED (à
    // MANIFEST_PARSED, hls.audioTracks est encore vide — confirmé en navigateur).
    hls.on(Hls.Events.AUDIO_TRACKS_UPDATED, (_evt, data) => {
      const tracks = data.audioTracks.map((t, i) => ({ id: i, name: t.name || t.lang || `Audio ${i + 1}` }))
      setAudioTracks(tracks)
      setActiveAudio(hls.audioTrack)
      log.debug('pistes audio reçues', { count: tracks.length, names: tracks.map((t) => t.name) })
    })
    hls.on(Hls.Events.ERROR, (_evt, data) => {
      if (data.fatal) {
        log.error('hls:fatal', 'Erreur fatale du flux HLS', {
          filePath, type: data.type, details: data.details,
        })
        setError('Erreur de lecture du flux HLS')
      }
    })
    return () => {
      hls.destroy()
      hlsRef.current = null
      setAudioTracks([])
      setActiveAudio(-1)
    }
  }, [filePath, isHls])

  // Chargement des segments réservé au clip centré : startLoad() au centrage,
  // stopLoad() au décentrage. Les instances hls.js sont créées autoStartLoad:false
  // (cf. effet d'attache, déclaré AVANT celui-ci → hlsRef.current est déjà posé
  // quand cet effet s'exécute au montage). Sans ce pilotage, tous les slots HLS
  // rendus (±2) chargeraient leurs segments en parallèle.
  useEffect(() => {
    const hls = hlsRef.current
    if (!hls) return
    if (isCenter) hls.startLoad()
    else hls.stopLoad()
  }, [isHls, isCenter])

  function selectAudioTrack(id: number) {
    if (hlsRef.current) {
      hlsRef.current.audioTrack = id
      setActiveAudio(id)
    }
  }

  // Index des renditions par slug (game/voices/full). Présence des 3 = layout
  // "2 toggles" produit par le transcodage récent ; sinon clip legacy (pistes
  // brutes) → sélecteur par-piste en repli.
  const bySlug = useMemo(() => {
    const m: Record<string, number> = {}
    for (const t of audioTracks) m[t.name] = t.id
    return m
  }, [audioTracks])
  const isToggleLayout =
    bySlug.game !== undefined && bySlug.voices !== undefined && bySlug.full !== undefined

  // Applique l'état des deux interrupteurs à la rendition jouée, et le RÉAPPLIQUE
  // au recentrage (dep isCenter) car l'instance persiste dans la fenêtre ±2.
  // Combinaisons : both→full, jeu seul→game, voix seule→voices, aucun→muet.
  // Le marqueur data-audio-off signale au parent (effet currentItem) de NE PAS
  // démuter ce clip : les effets du parent s'exécutent APRÈS ceux de l'enfant
  // dans un même commit, donc réappliquer `.muted` ici ne suffirait pas (le
  // parent le démuterait derrière) — c'est au parent de s'abstenir.
  useEffect(() => {
    if (!isToggleLayout) return
    const video = videoElRef.current
    if (!video) return
    if (!gameOn && !voiceOn) {
      video.muted = true
      video.dataset.audioOff = '1'
      return
    }
    delete video.dataset.audioOff
    video.muted = false
    const slug = gameOn && voiceOn ? 'full' : gameOn ? 'game' : 'voices'
    const idx = bySlug[slug]
    if (hlsRef.current && idx !== undefined) {
      hlsRef.current.audioTrack = idx
      setActiveAudio(idx)
    }
  }, [isToggleLayout, gameOn, voiceOn, bySlug, isCenter])

  if (error) {
    return (
      <div className="flex h-full w-full flex-col items-center justify-center bg-card p-4 text-center text-foreground/80">
        <svg className="mb-3 h-10 w-10 text-muted-foreground/60" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
          <path strokeLinecap="round" strokeLinejoin="round" d="M12 9v3.75m9-.75a9 9 0 11-18 0 9 9 0 0118 0zm-9 3.75h.008v.008H12v-.008z" />
        </svg>
        {/* eslint-disable-next-line @levelup/no-hardcoded-strings */}
        <p className="text-sm font-medium">Lecture impossible</p>
        <p className="mt-1 text-xs text-muted-foreground">{error}</p>
        {basename && <p className="mt-2 text-xs text-muted-foreground/70">{basename}</p>}
      </div>
    )
  }

  return (
    <div className="relative h-full w-full">
      <video
        ref={setRefs}
        src={isHls ? undefined : filePath}
        controls={isCenter}
        // On retire le bouton plein écran NATIF du <video> : le plein écran
        // passe par notre bouton (sur le conteneur stage). Sinon le natif
        // re-piégerait l'élément et l'autoChain casserait (image figée +
        // audio hors champ sur l'enchaînement).
        controlsList="nofullscreen"
        disablePictureInPicture
        muted={!isCenter}
        preload={Math.abs(relPos) <= 1 ? 'auto' : 'metadata'}
        onEnded={onEnded}
        onError={(e) => {
          if (isHls) return // les erreurs HLS passent par Hls.Events.ERROR
          const code = e.currentTarget.error?.code
          const msg =
            code === 4 ? 'Format vidéo non supporté par le navigateur'
            : code === 3 ? 'Erreur de décodage de la vidéo'
            : code === 2 ? 'Erreur réseau lors du chargement'
            : 'Vidéo inaccessible'
          log.warn('video:decode_error', msg, { filePath, code })
          setError(msg)
        }}
        className="h-full w-full bg-card object-contain"
      />
      {isCenter && audioTracks.length > 1 && (
        <div
          className="absolute bottom-16 right-3 z-10 flex items-center gap-1 rounded-md border border-border bg-card/90 p-1 backdrop-blur-sm"
          role="group"
          aria-label={audioLabels.group}
        >
          <svg className="ml-1 h-3.5 w-3.5 text-muted-foreground" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2} aria-hidden="true">
            <path strokeLinecap="round" strokeLinejoin="round" d="M3 10v4m4-7v10m4-13v16m4-11v6m4-8v10" />
          </svg>
          {isToggleLayout
            ? [
                { on: gameOn, set: setGameOn, label: audioLabels.game },
                { on: voiceOn, set: setVoiceOn, label: audioLabels.voice },
              ].map(({ on, set, label }) => (
                <button
                  key={label}
                  type="button"
                  aria-pressed={on}
                  onClick={(e) => {
                    e.stopPropagation()
                    log.debug('bascule interrupteur audio', { label, next: !on })
                    set((v) => !v)
                  }}
                  className={`rounded px-2.5 py-0.5 text-xs font-medium transition-colors ${
                    on
                      ? 'bg-primary text-primary-foreground'
                      : 'text-muted-foreground hover:bg-accent hover:text-foreground'
                  }`}
                >
                  {label}
                </button>
              ))
            : audioTracks.map((t) => (
                <button
                  key={t.id}
                  type="button"
                  aria-pressed={activeAudio === t.id}
                  onClick={(e) => {
                    e.stopPropagation()
                    selectAudioTrack(t.id)
                  }}
                  className={`rounded px-2 py-0.5 text-xs transition-colors ${
                    activeAudio === t.id ? 'bg-primary/20 text-primary' : 'text-muted-foreground hover:text-foreground'
                  }`}
                >
                  {t.name}
                </button>
              ))}
        </div>
      )}
    </div>
  )
}

export function CoverFlowModal({
  items,
  onToggleLike,
  startIndex,
  onClose,
  likeDisabled = false,
  hasNextPage = false,
  onLoadNextPage,
  globalIndexOffset = 0,
  globalTotal,
  onReassociate,
  currentPlayerGamertag,
  playerSlug,
  currentMatchId,
  onOpenMatch,
  autoChain = false,
  onToggleAutoChain,
}: CoverFlowModalProps) {
  const locale = useAppShellStore((s) => s.locale)
  const text = getMediaModalsText(locale)
  // L'identité de l'item courant est suivie via son file_path (stable),
  // pas via un index (qui peut changer si l'array items est réordonné).
  const [currentFilePath, setCurrentFilePath] = useState<string | null>(items[startIndex]?.file_path ?? null)
  const [pendingPageAdvance, setPendingPageAdvance] = useState(false)
  const animatingRef = useRef(false)
  const videoRefs = useRef<Map<string, HTMLVideoElement>>(new Map())
  const autoChainTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  // Plein écran : on met en fullscreen le CONTENEUR (stage) du carrousel, pas
  // le <video> lui-même. Comme le stage ne change pas quand l'autoChain
  // enchaîne, le plein écran persiste tout seul → fini l'image figée + audio
  // hors champ (un requestFullscreen sur le nouveau <video> serait bloqué car
  // hors d'un geste utilisateur lors d'un enchaînement automatique).
  const stageRef = useRef<HTMLDivElement>(null)
  const [isFullscreen, setIsFullscreen] = useState(false)

  useEffect(() => {
    function onFsChange() {
      setIsFullscreen(document.fullscreenElement === stageRef.current)
    }
    document.addEventListener('fullscreenchange', onFsChange)
    return () => document.removeEventListener('fullscreenchange', onFsChange)
  }, [])

  function toggleFullscreen() {
    const el = stageRef.current
    if (!el) return
    if (document.fullscreenElement) {
      void document.exitFullscreen().catch(() => undefined)
    } else {
      void el.requestFullscreen().catch(() => undefined)
    }
  }

  // Garde la dernière position valide pour éviter les sauts visuels si
  // l'item disparaît temporairement (refetch entre 2 vues).
  const [lastValidIdx, setLastValidIdx] = useState<number>(startIndex)

  // L'index courant est calculé dynamiquement depuis le file_path. Si l'item
  // a disparu, on retombe sur la dernière position connue clampée.
  const { idx: committedIdx, fromFallback } = useMemo(() => {
    if (!currentFilePath) {
      return { idx: Math.min(startIndex, Math.max(0, items.length - 1)), fromFallback: false }
    }
    const found = items.findIndex((item) => item.file_path === currentFilePath)
    if (found >= 0) return { idx: found, fromFallback: false }
    return { idx: Math.min(lastValidIdx, Math.max(0, items.length - 1)), fromFallback: true }
  }, [items, currentFilePath, startIndex, lastValidIdx])

  // Render-time state sync (pattern recommandé React quand l'état dépend des
  // props : https://react.dev/learn/you-might-not-need-an-effect). Évite la
  // ref-during-render flaggée par react-hooks/refs sans introduire de useEffect.
  if (!fromFallback && committedIdx !== lastValidIdx) {
    setLastValidIdx(committedIdx)
  }

  // Quand startIndex change (nouvelle ouverture de lightbox), reset le file_path.
  // Reste en useEffect car on doit lire animatingRef (non lisible en render).
  // Attention : seul `startIndex` est listé en dep — `items` ne doit PAS être
  // ajouté sinon on perd l'item courant à chaque refetch (cf. tests « reste sur
  // le même item si l'array est réordonné »).
  useEffect(() => {
    if (animatingRef.current) return
    setCurrentFilePath(items[startIndex]?.file_path ?? null)
    // eslint-disable-next-line react-hooks/exhaustive-deps -- voir commentaire ci-dessus
  }, [startIndex])

  // Pagination async : quand une nouvelle page d'items est livrée alors qu'un
  // advance est en attente, sauter à l'item 0 de la nouvelle page. Reste en
  // useEffect car la trigger est l'arrivée async de données externes (refetch
  // page suivante), pas un changement de prop synchrone.
  useEffect(() => {
    if (pendingPageAdvance && items.length > 0) {
      // eslint-disable-next-line react-hooks/set-state-in-effect -- resync sur arrivée async d'une nouvelle page (pas un dérivé synchrone), cf. commentaire ci-dessus (2026-07-22)
      setCurrentFilePath(items[0]?.file_path ?? null)
      setPendingPageAdvance(false)
    }
  }, [items, pendingPageAdvance])

  const canPrev = committedIdx > 0
  const canNext = committedIdx < items.length - 1
  // "Peut enchaîner" = il existe un item suivant local OU une page suivante à
  // charger. Symétrique du bouton next (disabled={!canNext && !hasNextPage}).
  const canAdvanceFurther = canNext || hasNextPage
  const currentItem = items[committedIdx] ?? null
  const isCurrentClip = currentItem?.kind === 'clip'

  const total = globalTotal ?? items.length
  const globalIndex = globalIndexOffset + committedIdx
  const heading = currentItem ? formatHeading(currentItem, globalIndex, total, locale) : ''
  const currentHasMatch = Boolean(currentItem?.match_id)
  const isOnCurrentMatchPage = currentHasMatch && currentMatchId === currentItem?.match_id
  const showViewMatchLink = currentHasMatch && !isOnCurrentMatchPage && Boolean(playerSlug) && Boolean(currentItem?.match_id)
  const reassociateLabel = currentHasMatch ? text.coverFlow.reassociateButton : text.coverFlow.associateButton
  const reassociateTitle = currentHasMatch ? text.coverFlow.reassociateTitle : text.coverFlow.associateTitle

  function navigate(dir: 'next' | 'prev') {
    if (animatingRef.current) return
    if (dir === 'next' && !canNext) {
      if (hasNextPage && onLoadNextPage && !pendingPageAdvance) {
        setPendingPageAdvance(true)
        onLoadNextPage()
      }
      return
    }
    if (dir === 'prev' && !canPrev) return

    const target = committedIdx + (dir === 'next' ? 1 : -1)
    const targetItem = items[target]
    if (!targetItem) return

    animatingRef.current = true
    setCurrentFilePath(targetItem.file_path)

    window.setTimeout(() => {
      animatingRef.current = false
    }, ANIM_MS)
  }

  useEffect(() => {
    videoRefs.current.forEach((vid, path) => {
      if (path === currentItem?.file_path) {
        // Ne PAS démuter un clip dont les deux interrupteurs Jeu/Voix sont OFF :
        // ClipPlayer pose data-audio-off="1" dans ce cas (l'état persiste car
        // l'instance survit à la navigation de voisinage). Cet effet parent
        // s'exécute APRÈS l'effet enfant du même commit — sans ce garde il
        // rétablirait le son alors que l'UI affiche Jeu OFF / Voix OFF.
        if (vid.dataset.audioOff !== '1') vid.muted = false
        if (vid.paused) {
          const p = vid.play()
          if (p && typeof p.catch === 'function') p.catch(() => undefined)
        }
      } else {
        vid.muted = true
        if (!vid.paused) vid.pause()
        if (vid.currentTime > 0) vid.currentTime = 0
      }
    })
  }, [currentItem])

  useEffect(() => {
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape') onClose()
      if (e.key === 'ArrowLeft') navigate('prev')
      if (e.key === 'ArrowRight') navigate('next')
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
    // eslint-disable-next-line react-hooks/exhaustive-deps -- navigate volontairement non mémoïsé (cf. l.554) ; closure rafraîchie via committedIdx/items.length (2026-07-22)
  }, [onClose, committedIdx, items.length])

  useEffect(() => {
    if (autoChainTimerRef.current) clearTimeout(autoChainTimerRef.current)
    if (!autoChain || !currentItem || isCurrentClip || !canAdvanceFurther || pendingPageAdvance) {
      return
    }
    // NB : `setTimeout` (pas `window.setTimeout`) pour que le type de retour
    // s'accorde à `ReturnType<typeof setTimeout>` du ref. Les 2 garde-rails node
    // (calendar/keys .guard.test.ts) tirent `@types/node` dans le programme tsc,
    // ce qui bascule `setTimeout` sur la surcharge Node (retour `Timeout`, pas
    // `number`). Runtime identique en navigateur (V1c, DC-2).
    autoChainTimerRef.current = setTimeout(() => navigate('next'), IMAGE_AUTOCHAIN_DELAY_MS)
    return () => {
      if (autoChainTimerRef.current) clearTimeout(autoChainTimerRef.current)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps -- navigate volontairement non mémoïsé (cf. l.554) ; closure rafraîchie via currentItem (2026-07-22)
  }, [autoChain, currentItem, isCurrentClip, canAdvanceFurther, pendingPageAdvance])

  const slots = useMemo(() => {
    const result: { item: MediaItemRow; relPos: number; absIdx: number }[] = []
    for (let off = -WINDOW_RADIUS; off <= WINDOW_RADIUS; off++) {
      const idx = committedIdx + off
      if (idx >= 0 && idx < items.length) {
        result.push({ item: items[idx], relPos: off, absIdx: idx })
      }
    }
    return result
  }, [items, committedIdx])

  function setVideoRef(filePath: string) {
    return (el: HTMLVideoElement | null) => {
      if (el) videoRefs.current.set(filePath, el)
      else videoRefs.current.delete(filePath)
    }
  }

  // Pas de useCallback : `navigate` est recréé à chaque render et capture
  // `committedIdx`. Mémoïser ce handler figerait un `navigate` périmé (deps
  // stables une fois l'enchaînement lancé) → re-navigation vers l'item courant
  // dès le 2e clip. Le handler n'est passé qu'au prop onEnded (aucun deps array
  // ne le référence, ClipPlayer n'est pas mémoïsé → recréation sans coût).
  const handleVideoEnded = () => {
    if (autoChain && canAdvanceFurther && !pendingPageAdvance) {
      navigate('next')
    }
  }

  if (!currentItem) {
    return null
  }

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-background/90"
      onClick={onClose}
    >
      <div
        className="relative mx-4 flex max-h-[90vh] w-full max-w-5xl flex-col"
        onClick={(event) => event.stopPropagation()}
      >
        <div className="flex items-center justify-between border-b border-border bg-card/95 px-4 py-2 text-foreground rounded-t-xl overflow-hidden">
          <div className="flex min-w-0 items-center gap-3">
            <span className="truncate text-sm text-foreground/80">{heading}</span>
            {showViewMatchLink && playerSlug && currentItem.match_id && (
              onOpenMatch ? (
                <button
                  type="button"
                  onClick={(e) => {
                    e.stopPropagation()
                    onOpenMatch(currentItem.match_id as string)
                  }}
                  title={text.coverFlow.viewMatchTitle}
                  aria-label={text.coverFlow.viewMatchTitle}
                  className="shrink-0 text-foreground/70 hover:text-foreground transition-colors"
                >
                  {/* Icône "ouvrir le match" — alignée sur celle de MatchCard pour cohérence visuelle */}
                  <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" fill="currentColor" className="h-4 w-4" aria-hidden="true">
                    <path d="M6.22 8.72a.75.75 0 0 0 1.06 1.06l5.22-5.22v1.69a.75.75 0 0 0 1.5 0v-3.5a.75.75 0 0 0-.75-.75h-3.5a.75.75 0 0 0 0 1.5h1.69L6.22 8.72Z" />
                    <path d="M3.5 6.75c0-.69.56-1.25 1.25-1.25H7A.75.75 0 0 0 7 4H4.75A2.75 2.75 0 0 0 2 6.75v4.5A2.75 2.75 0 0 0 4.75 14h4.5A2.75 2.75 0 0 0 12 11.25V9a.75.75 0 0 0-1.5 0v2.25c0 .69-.56 1.25-1.25 1.25h-4.5c-.69 0-1.25-.56-1.25-1.25v-4.5Z" />
                  </svg>
                </button>
              ) : (
                <a
                  href={`/players/${playerSlug}/matches/${currentItem.match_id}`}
                  onClick={(e) => e.stopPropagation()}
                  title={text.coverFlow.viewMatchTitle}
                  aria-label={text.coverFlow.viewMatchTitle}
                  className="shrink-0 text-foreground/70 hover:text-foreground transition-colors"
                >
                  {/* Icône "ouvrir le match" — alignée sur celle de MatchCard pour cohérence visuelle */}
                  <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" fill="currentColor" className="h-4 w-4" aria-hidden="true">
                    <path d="M6.22 8.72a.75.75 0 0 0 1.06 1.06l5.22-5.22v1.69a.75.75 0 0 0 1.5 0v-3.5a.75.75 0 0 0-.75-.75h-3.5a.75.75 0 0 0 0 1.5h1.69L6.22 8.72Z" />
                    <path d="M3.5 6.75c0-.69.56-1.25 1.25-1.25H7A.75.75 0 0 0 7 4H4.75A2.75 2.75 0 0 0 2 6.75v4.5A2.75 2.75 0 0 0 4.75 14h4.5A2.75 2.75 0 0 0 12 11.25V9a.75.75 0 0 0-1.5 0v2.25c0 .69-.56 1.25-1.25 1.25h-4.5c-.69 0-1.25-.56-1.25-1.25v-4.5Z" />
                  </svg>
                </a>
              )
            )}
            {/* Bouton réassocier : visible sauf si on PROUVE que le média est à un
                autre joueur (owner_gamertag connu ET différent du gamertag de
                référence). On ne se rabat PAS sur playerSlug ici (slug ≠ gamertag
                ⇒ masquait le bouton à tort dans MatchMediaTab/RecentMediaRail qui
                ne passent pas currentPlayerGamertag — cf. régression 8edd5c784). */}
            {onReassociate &&
              (!currentPlayerGamertag ||
                !currentItem.owner_gamertag ||
                currentItem.owner_gamertag.toLowerCase() === currentPlayerGamertag.toLowerCase()) && (
              <button
                type="button"
                onClick={(e) => {
                  e.stopPropagation()
                  onReassociate(currentItem)
                }}
                className="shrink-0 rounded border border-border px-2 py-0.5 text-xs text-foreground/80 hover:border-foreground/50 hover:text-foreground whitespace-nowrap"
                title={reassociateTitle}
              >
                {reassociateLabel}
              </button>
            )}
          </div>
          <div className="flex items-center gap-3 shrink-0">
            <div onClick={(e) => e.stopPropagation()}>
              <MediaLikeButton
                isLiked={currentItem.liked}
                likeCount={currentItem.like_count}
                onToggle={() => onToggleLike(currentItem)}
                disabled={likeDisabled}
              />
            </div>
            {onToggleAutoChain && (
              <button
                onClick={onToggleAutoChain}
                title={autoChain ? text.coverFlow.disableChaining : text.coverFlow.enableChaining}
                className={`flex items-center gap-1.5 rounded-full border px-2.5 py-1 text-xs transition-colors ${
                  autoChain
                    ? 'bg-success/20 border-success/50 text-success'
                    : 'bg-destructive/20 border-destructive/50 text-destructive'
                }`}
              >
                <svg
                  className="h-3.5 w-3.5"
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                  strokeWidth={2}
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    d="M13 5l7 7-7 7M5 5l7 7-7 7"
                  />
                </svg>
                {text.coverFlow.chainButton}
              </button>
            )}
            <button
              type="button"
              onClick={toggleFullscreen}
              title={isFullscreen ? text.coverFlow.exitFullscreen : text.coverFlow.enterFullscreen}
              aria-label={isFullscreen ? text.coverFlow.exitFullscreen : text.coverFlow.enterFullscreen}
              className="flex h-7 w-7 items-center justify-center rounded-full border border-border bg-card/90 text-muted-foreground transition-colors hover:text-foreground"
            >
              <svg
                className="h-3.5 w-3.5"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
                strokeWidth={2}
              >
                {isFullscreen ? (
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    d="M9 9L4 4m0 0v4m0-4h4m7 5l5-5m0 0v4m0-4h-4M9 15l-5 5m0 0v-4m0 4h4m7-5l5 5m0 0v-4m0 4h-4"
                  />
                ) : (
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    d="M4 8V4m0 0h4M4 4l5 5m11-5h-4m4 0v4m0-4l-5 5M4 16v4m0 0h4m-4 0l5-5m11 5l-5-5m5 5v-4m0 4h-4"
                  />
                )}
              </svg>
            </button>
            <button
              type="button"
              onClick={onClose}
              className="text-xl leading-none text-muted-foreground hover:text-foreground"
              aria-label={text.coverFlow.closeAriaLabel}
            >
              ✕
            </button>
          </div>
        </div>

        <div className="relative w-full flex-1 overflow-visible bg-card rounded-b-xl">
          {/* C'est CE conteneur qu'on met en plein écran (stageRef) → la vidéo
              centrale remplit l'écran et l'enchaînement reste fullscreen, sans
              transfert de fullscreen sur le nouveau <video>. */}
          <div
            ref={stageRef}
            className={`relative mx-auto w-full ${
              isFullscreen
                ? 'flex h-full items-center justify-center overflow-hidden bg-black'
                : 'aspect-video overflow-visible'
            }`}
          >
            <button
              onClick={(e) => {
                e.stopPropagation()
                navigate('prev')
              }}
              disabled={!canPrev}
              className="absolute left-3 top-1/2 z-20 -translate-y-1/2 rounded-full border border-border bg-card/90 p-2 text-foreground transition-colors hover:bg-card disabled:cursor-not-allowed disabled:opacity-30 md:left-6"
            >
              <svg
                className="h-6 w-6"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
                strokeWidth={2}
              >
                <path strokeLinecap="round" strokeLinejoin="round" d="M15 19l-7-7 7-7" />
              </svg>
            </button>
            {slots.map(({ item, relPos }) => {
              const pos = POSITIONS[relPos]
              const isCenter = relPos === 0
              return (
                <div
                  key={item.file_path}
                  className="absolute inset-0 overflow-hidden rounded-xl"
                  style={{
                    transform: `translateX(${pos.x}) scale(${pos.scale})`,
                    // En plein écran : on masque les voisins, seul le média
                    // central est visible (il remplit l'écran).
                    opacity: isFullscreen ? (isCenter ? 1 : 0) : pos.opacity,
                    transition: `transform ${ANIM_MS}ms ${ANIM_EASE}, opacity ${ANIM_MS}ms ${ANIM_EASE}`,
                    pointerEvents: isCenter ? 'auto' : 'none',
                    zIndex: 10 - Math.abs(relPos),
                    willChange: 'transform, opacity',
                  }}
                  onClick={
                    isCenter
                      ? undefined
                      : (e) => {
                          e.stopPropagation()
                          if (relPos < 0) navigate('prev')
                          else navigate('next')
                        }
                  }
                >
                  {item.kind === 'clip' ? (
                    <ClipPlayer
                      filePath={item.file_path}
                      basename={item.basename}
                      isCenter={isCenter}
                      relPos={relPos}
                      videoRef={setVideoRef(item.file_path)}
                      onEnded={isCenter && autoChain && canAdvanceFurther ? handleVideoEnded : undefined}
                      audioLabels={{
                        game: text.coverFlow.audioGame,
                        voice: text.coverFlow.audioVoice,
                        group: text.coverFlow.audioGroupLabel,
                      }}
                    />
                  ) : (
                    <img
                      src={item.file_path}
                      alt={item.basename}
                      className="h-full w-full object-contain bg-card"
                    />
                  )}
                </div>
              )
            })}

            <button
              onClick={(e) => {
                e.stopPropagation()
                navigate('next')
              }}
              disabled={!canNext && !hasNextPage}
              className="absolute right-3 top-1/2 z-20 -translate-y-1/2 rounded-full border border-border bg-card/90 p-2 text-foreground transition-colors hover:bg-card disabled:cursor-not-allowed disabled:opacity-30 md:right-6"
          >
            <svg
              className="h-6 w-6"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
              strokeWidth={2}
            >
              <path strokeLinecap="round" strokeLinejoin="round" d="M9 5l7 7-7 7" />
            </svg>
            </button>

            {/* Sortie plein écran — overlay DANS le stage (seul élément visible
                en fullscreen ; le header avec les boutons est hors champ). */}
            {isFullscreen && (
              <button
                type="button"
                onClick={(e) => {
                  e.stopPropagation()
                  toggleFullscreen()
                }}
                aria-label={text.coverFlow.exitFullscreen}
                title={text.coverFlow.exitFullscreen}
                className="absolute right-4 top-4 z-30 flex h-10 w-10 items-center justify-center rounded-full bg-black/60 text-foreground transition-colors hover:bg-black/80"
              >
                <svg
                  className="h-5 w-5"
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                  strokeWidth={2}
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    d="M9 9L4 4m0 0v4m0-4h4m7 5l5-5m0 0v4m0-4h-4M9 15l-5 5m0 0v-4m0 4h4m7-5l5 5m0 0v-4m0 4h-4"
                  />
                </svg>
              </button>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}
