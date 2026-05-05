/**
 * PlayerChips — selecteur radio-like de joueur (1 actif a la fois).
 *
 * Reference visuelle : Mock 15 v2 dans .ai/mockups/engagement/.
 * Plan : .ai/PLAN_ENGAGEMENT_IMPLEMENTATION.md §6.3
 *
 * Generic et reutilisable hors contexte engagement (selecteur de joueur
 * partout dans Squad page, comparaisons, etc.).
 *
 * Comportement :
 *   - Click chip inactive -> active (deselectionne l'autre)
 *   - Click chip active -> deselect (passe a null)
 *   - Couleurs via tokenCssVar(token), pas de hex
 *   - Hard-edge cohérent avec design system (pas de border-radius)
 *   - aria-pressed pour accessibilite
 */
import type { ReactNode } from 'react'

import { tokenCssVar, type SemanticToken } from '@/lib/accessibility'

export interface PlayerChipItem {
  /** Identifiant stable (XUID ou slug). */
  id: string
  /** Label affiche. */
  label: string
  /** Token de couleur semantique (ex. 'chart-series-1'). */
  colorToken: SemanticToken
  /** Couleur hex directe — prioritaire sur colorToken si fournie (ex. couleur joueur depuis playerColors). */
  colorHex?: string
}

export interface PlayerChipsProps {
  /** Liste des joueurs proposes. */
  players: PlayerChipItem[]
  /** ID actuellement selectionne, ou null. */
  selectedId: string | null
  /** Callback quand l'utilisateur change la selection (null = deselect). */
  onChange: (id: string | null) => void
  /** Label du groupe (precede les chips). */
  groupLabel?: string
  /** aria-label sur le conteneur. */
  ariaLabel?: string
  /** Slot enfant pour ajouter des elements custom apres les chips. */
  children?: ReactNode
}

/**
 * PlayerChips — chip selector unique-selection avec deselect au re-click.
 */
export function PlayerChips(props: PlayerChipsProps) {
  const { players, selectedId, onChange, groupLabel, ariaLabel, children } = props

  return (
    <div
      role="group"
      aria-label={ariaLabel}
      style={{
        display: 'flex',
        gap: '8px',
        flexWrap: 'wrap',
        alignItems: 'center',
      }}
    >
      {groupLabel && (
        <span
          style={{
            fontSize: '11px',
            color: 'var(--color-text-muted, rgba(255,255,255,0.45))',
            letterSpacing: '0.4px',
            textTransform: 'uppercase',
            fontWeight: 500,
            marginRight: '4px',
          }}
        >
          {groupLabel}
        </span>
      )}
      {players.map((p) => {
        const isActive = p.id === selectedId
        return (
          <PlayerChipButton
            key={p.id}
            player={p}
            isActive={isActive}
            onClick={() => onChange(isActive ? null : p.id)}
          />
        )
      })}
      {children}
    </div>
  )
}

interface PlayerChipButtonProps {
  player: PlayerChipItem
  isActive: boolean
  onClick: () => void
}

function PlayerChipButton(props: PlayerChipButtonProps) {
  const { player, isActive, onClick } = props
  const accent = player.colorHex ?? tokenCssVar(player.colorToken)

  return (
    <button
      type="button"
      onClick={onClick}
      aria-pressed={isActive}
      style={{
        background: isActive ? `color-mix(in srgb, ${accent} 18%, transparent)` : 'transparent',
        color: isActive ? '#fff' : 'var(--color-text-muted, rgba(255,255,255,0.7))', // color-allow: blanc structurel pour contraste sur pill colorée active
        border: `1px solid ${isActive ? accent : 'var(--color-border-subtle, rgba(255,255,255,0.15))'}`,
        padding: '5px 11px 5px 9px',
        fontFamily: 'inherit',
        fontSize: '11px',
        fontWeight: 500,
        cursor: 'pointer',
        userSelect: 'none',
        display: 'inline-flex',
        alignItems: 'center',
        gap: '7px',
        letterSpacing: '0.2px',
        transition: 'border-color 0.15s, background 0.15s, color 0.15s',
      }}
    >
      <span
        aria-hidden
        style={{
          display: 'inline-block',
          width: '9px',
          height: '9px',
          background: accent,
          flexShrink: 0,
        }}
      />
      <span>{player.label}</span>
    </button>
  )
}
