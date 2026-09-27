/**
 * SquadPlayerSheet — LA COQUILLE DES FICHES PAR JOUEUR de l'Escouade (spec S7 du plan
 * PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26), extraite de `MedalDigest.tsx` au lot L3 :
 * liseré gauche de 4 px à la couleur du joueur, emblème de 32 px, nom, pastille « … dominante »
 * en haut à droite, sections à étiquette en capitales de 9 px, pied de fiche.
 *
 * Une seule coquille pour les fiches de médailles (Contributions), de l'objectif
 * (Contributions) et des prises (Emprise) : trois copies auraient divergé au premier réglage.
 * Le contenu de chaque section appartient à l'appelant.
 */
import type { ReactNode } from 'react'

/** Emblème rond du joueur : l'URL fournie par le serveur, sinon son initiale sur sa couleur. */
export function SquadSheetAvatar({
  label,
  color,
  emblemUrl,
  initial,
}: {
  label: string
  color: string
  emblemUrl?: string
  /** Le signe affiché sans emblème (défaut : l'initiale du nom). */
  initial?: string
}) {
  if (emblemUrl) {
    return (
      <img
        src={emblemUrl}
        alt={label}
        className="h-8 w-8 rounded-full object-cover flex-shrink-0"
        style={{ boxShadow: `0 0 0 2px ${color}` }}
      />
    )
  }

  return (
    <span
      className="h-8 w-8 rounded-full flex items-center justify-center flex-shrink-0 text-xs font-bold"
      style={{ background: color, color: '#fff' }} // color-allow: blanc structurel sur fond joueur
    >
      {initial ?? label.charAt(0).toUpperCase()}
    </span>
  )
}

/** Une section de fiche : son étiquette en capitales de 9 px, puis son contenu. */
export function SquadSheetSection({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div>
      <p className="text-[9px] uppercase tracking-wider text-muted-foreground mb-1.5">{label}</p>
      {children}
    </div>
  )
}

export interface SquadPlayerSheetProps {
  /** Couleur du joueur (liseré, pastille). */
  color: string
  name: string
  avatar: ReactNode
  /** La pastille en haut à droite — absente quand rien ne domine. */
  dominant?: {
    caption: string
    label: string
    /** Fond de la pastille (défaut : la couleur du joueur à 14 %). */
    background?: string
  }
  /** Les sections, dans l'ordre. */
  children?: ReactNode
  /** Le pied de fiche (sous un filet). */
  footer?: ReactNode
  /** Ce qui suit le pied (ex. le dépliage de toutes les médailles). */
  afterFooter?: ReactNode
  testId?: string
  /** Classes ajoutées au cadre (ex. `min-w-0` dans une grille à colonnes contraintes). */
  className?: string
}

export function SquadPlayerSheet({
  color,
  name,
  avatar,
  dominant,
  children,
  footer,
  afterFooter,
  testId,
  className,
}: SquadPlayerSheetProps) {
  return (
    <div
      className={`flex flex-col gap-3 rounded-lg border border-border bg-card p-4${className ? ` ${className}` : ''}`}
      style={{ borderLeft: `4px solid ${color}` }}
      data-testid={testId}
    >
      <div className="flex items-center justify-between gap-2">
        <div className="flex items-center gap-2 min-w-0">
          {avatar}
          <span className="font-semibold text-sm text-foreground truncate">{name}</span>
        </div>
        {dominant && (
          <div className="flex flex-col items-end shrink-0">
            <span className="text-[9px] uppercase tracking-wider text-muted-foreground leading-none">
              {dominant.caption}
            </span>
            <span
              className="text-xs font-semibold mt-0.5 rounded px-1.5 py-0.5"
              style={{ background: dominant.background ?? `color-mix(in oklab, ${color} 14%, transparent)`, color }}
            >
              {dominant.label}
            </span>
          </div>
        )}
      </div>

      {children}

      {footer != null && (
        <div className="flex flex-wrap gap-x-4 gap-y-1 text-xs text-muted-foreground border-t border-border pt-2">
          {footer}
        </div>
      )}

      {afterFooter}
    </div>
  )
}
