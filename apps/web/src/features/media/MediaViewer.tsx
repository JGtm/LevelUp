import { useState } from 'react'
import { Badge } from '@/components/ui/badge'
import { GifHoverThumbnail } from '@/components/ui/gif-hover-thumbnail'
import { CoverFlowModal } from './CoverFlowModal'
import { getMediaText } from './i18n'
import { getMediaModalsText } from './i18n-modals'
import { useAppShellStore } from '@/stores/appShellStore'
import type { MediaItemRow } from '@/lib/api/types'

export { CoverFlowModal as MediaLightbox }

function formatMediaDate(value: string | null | undefined) {
  if (!value) {
    return null
  }
  const d = new Date(value)
  if (Number.isNaN(d.getTime())) {
    return null
  }
  const datePart = d.toLocaleDateString('fr-FR', { day: '2-digit', month: '2-digit', year: '2-digit' })
  const timePart = d.toLocaleTimeString('fr-FR', { hour: '2-digit', minute: '2-digit' })
  return `${timePart} ${datePart}`
}

const MAP_NAME_MAX_CHARS = 13

/** Tronque le nom de map à 13 caractères max (12 + "...") pour rester compact
 *  dans le footer de la vignette à côté du badge clip/screenshot. */
function truncateMapName(name: string): string {
  if (name.length <= MAP_NAME_MAX_CHARS) return name
  return `${name.slice(0, 12)}...`
}

/** Icône "ouvrir le match" — alignée sur le SVG de MatchCard pour cohérence
 *  visuelle entre les surfaces qui invitent à ouvrir un match. */
function OpenMatchIcon({ className }: { className?: string }) {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 16 16"
      fill="currentColor"
      className={className}
      aria-hidden="true"
    >
      <path d="M6.22 8.72a.75.75 0 0 0 1.06 1.06l5.22-5.22v1.69a.75.75 0 0 0 1.5 0v-3.5a.75.75 0 0 0-.75-.75h-3.5a.75.75 0 0 0 0 1.5h1.69L6.22 8.72Z" />
      <path d="M3.5 6.75c0-.69.56-1.25 1.25-1.25H7A.75.75 0 0 0 7 4H4.75A2.75 2.75 0 0 0 2 6.75v4.5A2.75 2.75 0 0 0 4.75 14h4.5A2.75 2.75 0 0 0 12 11.25V9a.75.75 0 0 0-1.5 0v2.25c0 .69-.56 1.25-1.25 1.25h-4.5c-.69 0-1.25-.56-1.25-1.25v-4.5Z" />
    </svg>
  )
}

// LikersLine — affiche "Alice, Bob et 3 autres ♥" sous le bouton like
function LikersLine({ likers, totalLikers }: { likers?: string[]; totalLikers?: number }) {
  if (!totalLikers || totalLikers === 0) return null
  const names = likers ?? []
  const rest = totalLikers - names.length
  let label: string
  if (names.length === 0) {
    label = `${totalLikers} ♥`
  } else if (rest <= 0) {
    label = `${names.join(', ')} ♥`
  } else {
    label = `${names.join(', ')} et ${rest} autre${rest > 1 ? 's' : ''} ♥`
  }
  return <p className="text-3xs text-rose-400 leading-tight">{label}</p> // color-allow: rose pour like indicator — CLAUDE.md §20
}

function HeartIcon({ filled, className }: { filled: boolean; className?: string }) {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 24 24"
      fill={filled ? 'currentColor' : 'none'}
      stroke="currentColor"
      strokeWidth={2}
      className={className}
      aria-hidden="true"
    >
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        d="M21 8.25c0-2.485-2.099-4.5-4.688-4.5-1.935 0-3.597 1.126-4.312 2.733-.715-1.607-2.377-2.733-4.313-2.733C5.1 3.75 3 5.765 3 8.25c0 7.22 9 12 9 12s9-4.78 9-12z"
      />
    </svg>
  )
}

interface MediaLikeButtonProps {
  isLiked: boolean
  likeCount: number
  onToggle: () => void
  compact?: boolean
  disabled?: boolean
}

export function MediaLikeButton({
  isLiked,
  likeCount,
  onToggle,
  compact = false,
  disabled = false,
}: MediaLikeButtonProps) {
  const [isAnimating, setIsAnimating] = useState(false)

  function handleClick(event: React.MouseEvent) {
    event.stopPropagation()
    onToggle()
    setIsAnimating(true)
    setTimeout(() => setIsAnimating(false), 250)
  }

  return (
    <button
      type="button"
      disabled={disabled}
      onClick={handleClick}
      // color-allow: rose pour le bouton like (heart) — CLAUDE.md §20 tolère rose pour liked
      className={compact
        ? `absolute right-1.5 top-1.5 flex h-7 min-w-7 items-center justify-center gap-1 rounded-full px-2 text-3xs font-semibold transition-colors disabled:cursor-not-allowed disabled:opacity-60 ${isLiked ? 'bg-card/90 text-rose-400' : 'bg-card/70 text-muted-foreground hover:text-rose-300'}` // color-allow: rose like button compact
        : `inline-flex items-center gap-2 rounded-full border px-3 py-1.5 text-xs font-medium transition-colors disabled:cursor-not-allowed disabled:opacity-60 ${isLiked ? 'border-rose-500/40 bg-rose-500/10 text-rose-400' : 'border-border bg-card/80 text-foreground hover:border-rose-400/40 hover:text-rose-300'}`} // color-allow: rose like button
      aria-label={isLiked ? 'Retirer le like' : 'Liker'}
    >
      <HeartIcon
        filled={isLiked}
        className={`transition-transform duration-200 ${compact ? 'h-4 w-4' : 'h-3.5 w-3.5'} ${isAnimating ? 'scale-125' : 'scale-100'}`}
      />
      {compact
        ? likeCount > 0 && <span>{likeCount}</span>
        : <span>{likeCount}</span>}
    </button>
  )
}

interface MediaThumbnailCardProps {
  item: MediaItemRow
  onToggleLike: (item: MediaItemRow) => void
  onOpen: () => void
  likeDisabled?: boolean
  /** Slug du joueur courant pour construire le lien vers la page du match. */
  playerSlug?: string
  /** Si renseigné et égal à `item.match_id`, supprime le lien "Voir le match"
   *  (on est déjà sur la page du match associé). */
  currentMatchId?: string | null
  /** Callback pour ouvrir le picker d'association (Associer ou Réassocier).
   *  Si absent, aucun lien d'(ré)association n'est rendu. */
  onAssociate?: (item: MediaItemRow) => void
  /** Callback pour naviguer vers la page du match. Si fourni, remplace le
   *  comportement par défaut (lien `<a>`) pour permettre au consommateur
   *  d'attacher un MatchNavContext (cf. useNavigateToMatch). */
  onOpenMatch?: (matchId: string) => void
}

export function MediaThumbnailCard({
  item,
  onToggleLike,
  onOpen,
  likeDisabled = false,
  playerSlug,
  currentMatchId,
  onAssociate,
  onOpenMatch,
}: MediaThumbnailCardProps) {
  const [isHovering, setIsHovering] = useState(false)
  const locale = useAppShellStore((s) => s.locale)
  const text = getMediaText(locale)
  const modals = getMediaModalsText(locale)
  const dateStr = formatMediaDate(item.capture_end_utc ?? item.match_start_time)
  const hasMatch = Boolean(item.match_id)
  const isCurrentMatch = hasMatch && currentMatchId === item.match_id
  const showViewMatchLink = hasMatch && !isCurrentMatch && Boolean(playerSlug)
  const showAssociateLink = !hasMatch && Boolean(onAssociate)

  return (
    <article
      className="group relative overflow-hidden rounded-lg border bg-muted transition-shadow hover:shadow-md"
      onClick={onOpen}
      onMouseEnter={() => setIsHovering(true)}
      onMouseLeave={() => setIsHovering(false)}
      onFocus={() => setIsHovering(true)}
      onBlur={() => setIsHovering(false)}
      onKeyDown={(event) => {
        if (event.key === 'Enter' || event.key === ' ') {
          event.preventDefault()
          onOpen()
        }
      }}
      role="button"
      tabIndex={0}
    >
      <div className="relative aspect-video overflow-hidden bg-muted">
        {item.kind === 'clip' && item.thumbnail_path ? (
          <GifHoverThumbnail
            src={item.thumbnail_path}
            isActive={isHovering}
            className="absolute inset-0 h-full w-full"
          />
        ) : item.thumbnail_path ? (
          <img
            src={item.thumbnail_path}
            alt={item.basename}
            className="h-full w-full object-cover transition-transform duration-200 group-hover:scale-105"
          />
        ) : item.kind === 'screenshot' ? (
          <img
            src={item.file_path}
            alt={item.basename}
            className="h-full w-full object-cover transition-transform duration-200 group-hover:scale-105"
          />
        ) : item.kind === 'clip' ? (
          // Bug #8 : clips sans thumbnail backfillé (cas typique des médias
          // de coéquipiers). Le browser affiche la première frame avec
          // `preload="metadata"` + fragment `#t=0.5` qui force un seek dès
          // le chargement metadata — pas de download de la vidéo entière.
          <video
            src={`${item.file_path}#t=0.5`}
            preload="metadata"
            muted
            playsInline
            className="h-full w-full object-cover transition-transform duration-200 group-hover:scale-105"
          />
        ) : (
          <div className="flex h-full w-full items-center justify-center text-2xl text-muted-foreground">
            ▶
          </div>
        )}
        {item.kind === 'clip' && (
          <div className="pointer-events-none absolute inset-0 flex items-center justify-center">
            <svg
              className="h-12 w-12 text-foreground/60 drop-shadow-md transition-opacity duration-200 group-hover:opacity-0"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth={1.5}
              aria-hidden="true"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
              />
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                d="M15.91 11.672a.375.375 0 010 .656l-5.603 3.113a.375.375 0 01-.557-.328V8.887c0-.286.307-.466.557-.327l5.603 3.112z"
              />
            </svg>
          </div>
        )}
        <MediaLikeButton
          compact
          isLiked={item.liked}
          likeCount={item.like_count}
          onToggle={() => onToggleLike(item)}
          disabled={likeDisabled}
        />
      </div>

      <div className="flex flex-col gap-1 p-2">
        <div className="flex items-center gap-1 min-w-0">
          <Badge variant={item.kind === 'clip' ? 'default' : 'secondary'} className="text-xs shrink-0">
            {item.kind}
          </Badge>
          {item.map_name ? (
            <span className="truncate text-xs text-muted-foreground min-w-0">{truncateMapName(item.map_name)}</span>
          ) : showAssociateLink && onAssociate ? (
            <button
              type="button"
              onClick={(e) => {
                e.stopPropagation()
                onAssociate(item)
              }}
              title={modals.coverFlow.associateTitle}
              className="truncate text-xs italic text-muted-foreground hover:text-foreground hover:underline min-w-0"
            >
              + {modals.coverFlow.associateButton}
            </button>
          ) : !hasMatch ? (
            <span className="truncate text-xs text-muted-foreground/50 min-w-0 italic">{text.thumbnail.noMatchAssociated}</span>
          ) : null}
          {showViewMatchLink && playerSlug && item.match_id && (
            onOpenMatch ? (
              <button
                type="button"
                onClick={(e) => {
                  e.stopPropagation()
                  onOpenMatch(item.match_id as string)
                }}
                title={modals.coverFlow.viewMatchTitle}
                aria-label={modals.coverFlow.viewMatchTitle}
                className="shrink-0 text-muted-foreground hover:text-foreground transition-colors"
              >
                <OpenMatchIcon className="h-3 w-3" />
              </button>
            ) : (
              <a
                href={`/players/${playerSlug}/matches/${item.match_id}`}
                onClick={(e) => e.stopPropagation()}
                title={modals.coverFlow.viewMatchTitle}
                aria-label={modals.coverFlow.viewMatchTitle}
                className="shrink-0 text-muted-foreground hover:text-foreground transition-colors"
              >
                <OpenMatchIcon className="h-3 w-3" />
              </a>
            )
          )}
          {dateStr && <span className="ml-auto shrink-0 text-xs text-muted-foreground">{dateStr}</span>}
        </div>
        <LikersLine likers={item.likers} totalLikers={item.total_likers} />
      </div>
    </article>
  )
}
