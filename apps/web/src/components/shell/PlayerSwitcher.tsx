/**
 * PlayerSwitcher — sélecteur de joueur actif de la NavL1, avec présence en jeu.
 *
 * Remplace le `<select>` natif d'origine : une `<option>` ne peut porter ni SVG
 * ni badge, or c'est exactement ce que le point Notion 4 demande — une manette à
 * côté des joueurs en partie et un compteur d'amis en jeu. Le dropdown reprend
 * le gabarit des split buttons de NavL1.tsx (même wrapper `relative`, même
 * panneau `role="menu"` absolute/bg-popover/z-50, même click-outside par
 * mousedown), enrichi de la navigation clavier attendue d'un menu ARIA.
 *
 * La bascule de joueur reste celle de la NavL1 (`onPlayerChange` →
 * handlePlayerChange → resolvePlayerSwitch) : ce composant ne décide de rien,
 * il affiche et déclenche.
 *
 * Extrait de NavL1.tsx (449 lignes) plutôt qu'ajouté dedans : le dropdown y
 * aurait fait dépasser le seuil des 500 lignes.
 */
import { useEffect, useRef, useState, type KeyboardEvent } from 'react'

import { formatMessage } from '@/lib/i18n/format'
import { commonManifest, type CommonManifestKey } from '@/lib/i18n/generated/common'
import type { PlayerSummary } from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'

import { usePresence } from './usePresence'

export interface PlayerSwitcherProps {
  players: PlayerSummary[]
  currentPlayer: PlayerSummary | null
  locale: Locale
  onPlayerChange: (playerSlug: string) => void
}

/** Manette — marqueur « ce joueur est en partie ». SVG inline (aucune dépendance d'icônes). */
function GamepadIcon({ className }: { className?: string }) {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      className={className}
      viewBox="0 0 20 20"
      fill="currentColor"
      aria-hidden="true"
    >
      <path d="M6.5 5h7a4.5 4.5 0 0 1 4.4 3.6l.8 4a3 3 0 0 1-5.2 2.6l-1-1.1a1 1 0 0 0-.8-.3H8.3a1 1 0 0 0-.8.3l-1 1.1A3 3 0 0 1 1.3 12.6l.8-4A4.5 4.5 0 0 1 6.5 5Zm-.6 2.6a.9.9 0 0 0-.9.9v.6h-.6a.9.9 0 1 0 0 1.8H5v.6a.9.9 0 1 0 1.8 0v-.6h.6a.9.9 0 1 0 0-1.8h-.6v-.6a.9.9 0 0 0-.9-.9Zm7.3.2a1 1 0 1 0 0 2 1 1 0 0 0 0-2Zm1.8 2.4a1 1 0 1 0 0 2 1 1 0 0 0 0-2Z" />
    </svg>
  )
}

/**
 * Pastille « N amis en jeu » — rendue seulement si N > 0 (un « 0 » permanent
 * serait du bruit). Le nombre seul étant cryptique, la manette l'accompagne et
 * le libellé accessible porte la phrase complète.
 */
function FriendsBadge({ count, label }: { count: number; label: string }) {
  return (
    <span
      className="inline-flex shrink-0 items-center gap-0.5 rounded-full bg-sidebar-accent px-1.5 py-0.5 text-2xs font-medium leading-none text-sidebar-accent-foreground"
      title={label}
      aria-label={label}
      data-testid="friends-in-game-badge"
    >
      <GamepadIcon className="h-3 w-3" />
      {count}
    </span>
  )
}

export function PlayerSwitcher({ players, currentPlayer, locale, onPlayerChange }: PlayerSwitcherProps) {
  const [open, setOpen] = useState(false)
  const ref = useRef<HTMLDivElement>(null)
  const triggerRef = useRef<HTMLButtonElement>(null)
  const menuRef = useRef<HTMLDivElement>(null)
  const t = (key: CommonManifestKey, vars?: Record<string, unknown>) =>
    formatMessage(commonManifest, key, locale, vars)

  const presence = usePresence()

  function menuItems(): HTMLButtonElement[] {
    return Array.from(menuRef.current?.querySelectorAll<HTMLButtonElement>('[role="menuitem"]') ?? [])
  }

  useEffect(() => {
    function handleClickOutside(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false)
    }
    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [])

  // À l'ouverture, le focus part sur le premier joueur : un menu ouvert au
  // clavier qui laisse le focus sur son déclencheur n'est pas navigable.
  useEffect(() => {
    if (!open) return
    menuItems()[0]?.focus()
  }, [open])

  function moveFocus(delta: number) {
    const items = menuItems()
    if (items.length === 0) return
    const current = items.indexOf(document.activeElement as HTMLButtonElement)
    const next = (current + delta + items.length) % items.length
    items[next]?.focus()
  }

  function handleMenuKeyDown(e: KeyboardEvent<HTMLDivElement>) {
    if (e.key === 'Escape') {
      setOpen(false)
      triggerRef.current?.focus()
      return
    }
    if (e.key === 'ArrowDown' || e.key === 'ArrowUp') {
      e.preventDefault()
      moveFocus(e.key === 'ArrowDown' ? 1 : -1)
    }
  }

  function handleTriggerKeyDown(e: KeyboardEvent<HTMLButtonElement>) {
    if (e.key === 'ArrowDown' && !open) {
      e.preventDefault()
      setOpen(true)
    }
  }

  function select(slug: string) {
    setOpen(false)
    onPlayerChange(slug)
  }

  const friendsLabel = t('common.shell.friends_in_game', { n: presence.friendsInGame })
  const friendsBadge =
    presence.friendsInGame > 0 ? (
      <FriendsBadge count={presence.friendsInGame} label={friendsLabel} />
    ) : null

  /**
   * Manette du joueur donné, ou null s'il n'est pas en partie. Le libellé est
   * porté par le `<span>` et non par le SVG : c'est lui que lisent les lecteurs
   * d'écran comme les tests.
   */
  function inGameBadge(playerSlug: string) {
    const state = presence.byPlayerSlug.get(playerSlug)
    if (!state?.in_game) return null
    // Le titre est nommé quand on le connaît : un joueur peut être en jeu sur un
    // AUTRE titre que celui affiché, et c'est précisément ce qu'il faut dire.
    const label = state.title_name
      ? t('common.shell.player_in_game_on', { title: state.title_name })
      : t('common.shell.player_in_game')
    return (
      <span
        className="inline-flex shrink-0 items-center text-sidebar-foreground/70"
        title={label}
        aria-label={label}
        data-testid={`in-game-${playerSlug}`}
      >
        <GamepadIcon className="h-3.5 w-3.5" />
      </span>
    )
  }

  // Un seul joueur : pas de menu à ouvrir (comportement d'origine préservé), mais
  // la présence a autant de sens — manette et compteur d'amis restent affichés.
  if (players.length <= 1) {
    if (!currentPlayer) return null
    return (
      <div className="flex min-w-0 items-center gap-1.5">
        <span className="max-w-[32vw] truncate text-sm font-medium text-sidebar-foreground/70 sm:max-w-none">
          {currentPlayer.gamertag}
        </span>
        {inGameBadge(currentPlayer.player_slug)}
        {friendsBadge}
      </div>
    )
  }

  return (
    <div ref={ref} className="relative flex min-w-0 items-center gap-1.5">
      <button
        ref={triggerRef}
        type="button"
        onClick={() => setOpen((v) => !v)}
        onKeyDown={handleTriggerKeyDown}
        aria-label={t('common.shell.player_select')}
        aria-haspopup="menu"
        aria-expanded={open}
        className="flex max-w-[32vw] cursor-pointer items-center gap-1.5 rounded-md border border-sidebar-border bg-sidebar-accent px-2 py-1 text-sm text-sidebar-foreground transition-colors hover:bg-sidebar-accent/80 focus:border-ring focus:outline-none focus:ring-2 focus:ring-ring/30 sm:max-w-none"
      >
        {/* Largeur STABLE, comme le <select> natif d'origine : un select se
            dimensionne sur sa plus LONGUE option, donc changer de joueur ne
            bougeait rien. Le calibre invisible (h-0) rend tous les gamertags en
            bloc — le plus large fixe la largeur — et le gamertag courant se
            tronque dedans. Gate visuel du 25/08 : sans lui, le bouton respirait
            à chaque bascule. */}
        <span className="min-w-0">
          <span aria-hidden className="pointer-events-none invisible block h-0 overflow-hidden" data-testid="player-switcher-sizer">
            {players.map((p) => (
              <span key={p.player_slug} className="block">
                {p.gamertag}
              </span>
            ))}
          </span>
          <span className="block truncate">{currentPlayer?.gamertag ?? ''}</span>
        </span>
        {/* Emplacement de la manette RÉSERVÉ même hors jeu : son apparition ne
            doit pas élargir le bouton. */}
        <span className="inline-flex w-3.5 shrink-0 justify-center">
          {currentPlayer && inGameBadge(currentPlayer.player_slug)}
        </span>
        <svg
          xmlns="http://www.w3.org/2000/svg"
          className={`h-3 w-3 shrink-0 transition-transform ${open ? 'rotate-180' : ''}`}
          viewBox="0 0 12 12"
          fill="currentColor"
          aria-hidden="true"
        >
          <path d="M6 8L1 3h10z" />
        </svg>
      </button>

      {friendsBadge}

      {open && (
        <div
          ref={menuRef}
          role="menu"
          onKeyDown={handleMenuKeyDown}
          className="absolute right-0 top-full z-50 mt-1 min-w-[12rem] rounded-md border border-border bg-popover py-1 shadow-lg"
        >
          {players.map((p) => (
            <button
              key={p.player_slug}
              type="button"
              role="menuitem"
              onClick={() => select(p.player_slug)}
              aria-current={p.player_slug === currentPlayer?.player_slug ? 'true' : undefined}
              className="flex w-full cursor-pointer items-center justify-between gap-3 px-3 py-1.5 text-left text-sm whitespace-nowrap text-popover-foreground hover:bg-accent hover:text-accent-foreground"
            >
              <span className="truncate">{p.gamertag}</span>
              {inGameBadge(p.player_slug)}
            </button>
          ))}
        </div>
      )}
    </div>
  )
}
