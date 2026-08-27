/**
 * MedalBadges — les badges de médaille EN IMAGES (POC) : le badge du jeu se reconnaît d'un
 * coup d'œil, le libellé et la description vivent dans l'infobulle où ils ne coûtent rien.
 * PAS d'inversion ni de teinte : une médaille est une pièce EN COULEUR.
 *
 * REPLI : une médaille sans visuel garde son TEXTE — jamais le badge d'une autre.
 *
 * Extrait de `ReplayKillFeed.tsx` le 2026-08-16, quand le fil a franchi le seuil de taille
 * du dépôt en recevant les marques d'identité : ce bloc n'a aucun lien avec la ligne qui le
 * porte (il ignore tueur, victime, couleurs d'équipe), c'est donc lui qui sort.
 */
import type { MedalEvent } from './killFeedLogic'

/** Côté d'un badge de médaille, en px (option 2a du handoff 2026-08-27 : 16 px). */
const MEDAL_PX = 16

export function MedalBadges({ medals }: { medals: MedalEvent[] }) {
  return (
    <>
      {medals.map((m, i) => {
        const label = m.label || m.name
        const tooltip = m.description ? `${label} — ${m.description}` : label
        return m.imageUrl ? (
          <img
            key={`${m.name}-${m.tMs}-${i}`}
            src={m.imageUrl}
            alt={label}
            title={tooltip}
            width={MEDAL_PX}
            height={MEDAL_PX}
            className="inline-block object-contain"
            loading="lazy"
          />
        ) : (
          <span
            key={`${m.name}-${m.tMs}-${i}`}
            title={tooltip}
            className="rounded-sm border border-border px-1 text-3xs text-muted-foreground"
          >
            {label}
          </span>
        )
      })}
    </>
  )
}
