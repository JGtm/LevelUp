/**
 * ReplayKillLineParts — les morceaux d'une ligne de FRAG du fil des éliminations : le lieu de la
 * victime et l'arme (chacun dans sa colonne fixe, cotes de COTES_LIGNE_FIL), et le pictogramme d'arme partagé
 * avec la ligne de destruction d'un véhicule. Chaque morceau ne connaît que ce qu'il dessine ; la
 * grille, l'instant et les noms restent à la ligne.
 */
import { WeaponIcon } from '@/components/ui/WeaponIcon'

import { REPLAY_TEXT, type ReplayLocale } from '../i18n/i18n'
import { COTES_LIGNE_FIL } from '../model/cotesFeedLine'
import type { ReplayKill } from '../model/killFeedLogic'

/** Gabarit d'une icône d'arme hors de la grille d'élimination : le format bandeau de l'atlas kill feed (22x10). */
const ICON_W = 22
const ICON_H = 10
/** Diamètre du repère servi quand l'arme est inconnue — jamais l'icône d'une autre arme. */
export const DOT_PX = 7

/**
 * KillZone — LA COLONNE DU LIEU d'une ligne de frag : la zone nommée où se trouvait la victime, à
 * l'instant du frag, tronquée si elle déborde (le nom complet est dans l'infobulle). Zone inconnue : la
 * colonne reste, VIDE — aucun texte, jamais un nom de repli.
 */
export function KillZone({ name, locale }: { name: string; locale: ReplayLocale }) {
  const t = REPLAY_TEXT[locale]
  return (
    <span
      data-feed-col="lieu"
      className="truncate text-muted-foreground"
      style={{ fontSize: COTES_LIGNE_FIL.tailleSecondairePx }}
      title={name ? t.killFeedVictimZone(name) : undefined}
    >
      {name}
    </span>
  )
}

/**
 * FeedWeaponIcon — le pictogramme d'une arme au gabarit du fil : masque teint par color (l'encre du
 * camp de celui qui agit) ou image finie. Partagé par la ligne de frag et la ligne de destruction,
 * pour qu'une arme se dessine de la même façon partout. Sans URL, il ne rend rien. size : le
 * gabarit de la colonne d'arme d'une élimination ; absent, celui du bandeau.
 */
export function FeedWeaponIcon({
  imageUrl,
  tinted,
  label,
  color,
  size = { largeurPx: ICON_W, hauteurPx: ICON_H },
}: {
  imageUrl: string
  tinted: boolean
  label: string
  color: string
  size?: { largeurPx: number; hauteurPx: number }
}) {
  return (
    <WeaponIcon
      imageUrl={imageUrl}
      tinted={tinted}
      label={label}
      width={size.largeurPx}
      height={size.hauteurPx}
      style={{ color }}
    />
  )
}

/**
 * KillWeapon — LA COLONNE DE L'ARME, entre le tueur et la victime : elle remplace la croix (POC). L'icône
 * extraite du jeu est un masque teint par currentColor (cf. WeaponIcon) : la couleur d'équipe du
 * TUEUR est posée sur l'icône elle-même. Arme inconnue : un repère rond, jamais l'icône d'une autre.
 */
export function KillWeapon({ kill: k, color, locale }: { kill: ReplayKill; color: string; locale: ReplayLocale }) {
  const t = REPLAY_TEXT[locale]
  return (
    <span data-feed-col="arme" className="flex items-center justify-center">
      {k.weaponImageUrl ? (
        <FeedWeaponIcon
          imageUrl={k.weaponImageUrl}
          tinted={k.weaponTinted}
          label={k.weaponLabel || t.killFeedUnknownWeapon}
          color={color}
          size={COTES_LIGNE_FIL.arme}
        />
      ) : (
        <span
          aria-hidden
          className="rounded-full"
          style={{ width: DOT_PX, height: DOT_PX, backgroundColor: color, opacity: 0.7, flex: 'none' }}
        />
      )}
    </span>
  )
}
