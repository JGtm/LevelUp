/**
 * ReplayKillLineParts — les trois morceaux d'une ligne de FRAG du fil des éliminations : la zone
 * du tueur, l'arme, l'assistance. Extraits de `KillLine` (ReplayKillFeed.tsx) quand la zone du
 * tueur est arrivée : la fonction franchissait déjà le seuil des 80 lignes et le fichier celui
 * des 500, et un morceau de plus dans la ligne les aurait creusés. Chaque morceau ne connaît que
 * ce qu'il dessine ; la rangée, l'horloge et les noms restent à la ligne.
 */
import { WeaponIcon } from '@/components/ui/WeaponIcon'
import { displayPlayerName, normalizeGamertagKey } from '@/lib/players/displayName'

import { REPLAY_TEXT, type ReplayLocale } from '../i18n/i18n'
import type { ReplayKill } from '../model/killFeedLogic'
import type { PlayerMarkKind } from '../../../lib/replay/playerMarks'
import { AssistMark } from './ReplayAssistMark'
import { FeedName } from './ReplayFeedName'

/** Gabarit d'une icône d'arme : le format bandeau de l'atlas kill feed (option 2a : 22×10). */
const ICON_W = 22
const ICON_H = 10
/** Diamètre du repère servi quand l'arme est inconnue — jamais l'icône d'une autre arme. */
export const DOT_PX = 7

/**
 * KillZone — LA GOUTTIÈRE DE ZONE d'une ligne de frag : la zone nommée où se trouvait le tueur, à
 * l'instant du frag, tronquée si elle déborde (le nom complet est dans l'infobulle). Largeur FIXE
 * (~5,5 rem), pour que les colonnes des lignes de frag s'alignent d'un regard. Zone inconnue : la
 * gouttière reste, VIDE — aucun texte, jamais un nom de repli.
 */
export function KillZone({ name, locale }: { name: string; locale: ReplayLocale }) {
  const t = REPLAY_TEXT[locale]
  return (
    <span
      className="w-[5.5rem] shrink-0 truncate text-3xs text-muted-foreground"
      title={name ? t.killFeedKillerZone(name) : undefined}
    >
      {name}
    </span>
  )
}

/**
 * KillWeapon — L'ARME entre le tueur et la victime : elle remplace la croix (POC). L'icône
 * extraite du jeu est un masque teint par currentColor (cf. WeaponIcon) : la couleur d'équipe du
 * TUEUR est posée sur l'icône elle-même.
 */
export function KillWeapon({ kill: k, color, locale }: { kill: ReplayKill; color: string; locale: ReplayLocale }) {
  const t = REPLAY_TEXT[locale]
  return k.weaponImageUrl ? (
    <WeaponIcon
      imageUrl={k.weaponImageUrl}
      tinted={k.weaponTinted}
      label={k.weaponLabel || t.killFeedUnknownWeapon}
      width={ICON_W}
      height={ICON_H}
      style={{ color }}
    />
  ) : (
    /* Repli assumé : la source du dégât n'est pas identifiable sans ambiguïté. */
    <span
      aria-hidden
      className="rounded-full"
      style={{ width: DOT_PX, height: DOT_PX, backgroundColor: color, opacity: 0.7, flex: 'none' }}
    />
  )
}

/**
 * KillAssist — L'ASSISTANCE dans la même rangée : la marque à l'encre `stat-assists` (option 2a — la
 * couleur d'équipe de l'assistant colorait une DEUXIÈME fois ce que son nom dit déjà), l'assistant,
 * SA part seule — la part du tueur est sortie de la rangée (demande utilisateur du 2026-08-24 :
 * « celui de l'assistant suffit »). Le nom de l'assistant est le seul élément qui cède — comme celui
 * du tueur et de la victime.
 */
export function KillAssist({
  kill: k,
  marksByGamertag,
  locale,
}: {
  kill: ReplayKill
  marksByGamertag: ReadonlyMap<string, PlayerMarkKind>
  locale: ReplayLocale
}) {
  const t = REPLAY_TEXT[locale]
  return (
    <span className="flex min-w-0 items-center gap-1 text-[10.5px] text-muted-foreground" title={t.killFeedAssistHint}>
      <AssistMark label={t.killFeedAssistMark} />
      <FeedName
        name={displayPlayerName(k.assistGamertag)}
        locale={locale}
        kind={marksByGamertag.get(normalizeGamertagKey(k.assistGamertag))}
      />
      {k.assistDamagePct != null && (
        <span className="shrink-0 font-mono tabular-nums">{t.killFeedAssistShare(k.assistDamagePct)}</span>
      )}
    </span>
  )
}
