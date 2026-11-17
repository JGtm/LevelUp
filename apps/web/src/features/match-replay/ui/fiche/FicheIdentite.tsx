/**
 * FicheIdentite — L'IDENTITÉ DU JOUEUR EN TÊTE DE LA FICHE : la pastille de son camp, son nom en gras, le nom de
 * son camp, puis ses frags / morts / assistances (ceux du film à l'instant lu, à défaut les totaux de la feuille). Ce
 * que la surface pose à droite (capsule « Échap », croix de la fenêtre) suit, à la même rangée.
 *
 * Aucune bordure colorée : le camp se dit par la pastille et par son nom. Le nom est à 17 px dans la colonne du
 * Standard, à 16 px dans la fenêtre. Le rendu ne lit rien : tout est donné par props.
 */
import type { ReactNode } from 'react'

import { REPLAY_TEXT, type ReplayLocale } from '../../i18n/i18n'
import { COTES_FICHE_INSTANT, type FicheSurface } from '../../model/cotesFicheInstant'
import type { IdentityCounters } from '../../model/playerInstantView'

const C = COTES_FICHE_INSTANT.identite

export interface FicheIdentiteProps {
  name: string
  /** Le nom du camp du joueur dans le film ; `null` quand le film ne l'écrit pas. */
  camp: string | null
  /** L'encre de la pastille : celle du camp (`campInk`). */
  ink: string
  counters: IdentityCounters | null
  surface: FicheSurface
  locale: ReplayLocale
  /** Ce que la surface pose à droite de l'identité. */
  headEnd?: ReactNode
}

export function FicheIdentite({ name, camp, ink, counters, surface, locale, headEnd }: FicheIdentiteProps) {
  const t = REPLAY_TEXT[locale]
  return (
    <div className="flex items-center" style={{ gap: C.ecartPx }} data-fiche-identite="">
      <span
        aria-hidden
        className="inline-block shrink-0"
        style={{ width: C.pastillePx, height: C.pastillePx, borderRadius: C.pastilleRayonPx, background: ink }}
      />
      <b className="min-w-0 truncate font-bold text-foreground" style={{ fontSize: C.nomPx[surface] }} title={name}>
        {name}
      </b>
      {camp && (
        <small className="shrink-0 text-muted-foreground" style={{ fontSize: C.campPx }}>
          {camp}
        </small>
      )}
      <div className="ml-auto flex shrink-0 items-center" style={{ gap: C.ecartPx }}>
        {counters && (
          <span
            className="font-semibold tabular-nums text-foreground"
            style={{ fontSize: C.triplePx }}
            title={counters.fromFilm ? t.countersLive : t.countersMatch}
          >
            {counters.kills}
            <Separator />
            {counters.deaths}
            <Separator />
            {counters.assists}
          </span>
        )}
        {headEnd}
      </div>
    </div>
  )
}

function Separator() {
  return (
    <span className="font-normal text-muted-foreground" style={{ marginInline: C.separateurMargePx }}>
      /
    </span>
  )
}
