/**
 * ReplayEventLine — la ligne d'une PRISE d'arme ou de bonus (sur socle ou au sol), d'une MONTÉE en
 * véhicule ou d'une DESTRUCTION de véhicule : un pictogramme, le joueur à l'encre de son camp, et
 * une phrase. Même rangée que toutes les lignes du fil (une seule ligne, sans retour, pictogramme
 * `shrink-0`) ; contrairement à l'objectif, elle n'est pas un bandeau : ce ne sont pas des
 * tournants du match.
 *
 * UNE DESTRUCTION SANS AUTEUR n'a pas de nom de joueur : la ligne ne porte que la phrase
 * (« Ghost détruit »). Avec un auteur, il ouvre la phrase comme pour toute autre ligne ; les joueurs
 * tués avec le véhicule, quand le film en nomme, la ferment entre parenthèses (leur nom s'il n'y en
 * a qu'un, leur nombre sinon : `vehicleKilledLabel`).
 *
 * LE PICTOGRAMME DE LA CAUSE (destruction seulement) : quand la ligne porte l'arme qui a détruit le
 * véhicule, son dessin prend la place entre l'auteur et la phrase, au même gabarit et à la même encre
 * que sur une ligne d'élimination. Sans arme résolue, aucun repère n'est dessiné.
 */
import { REPLAY_TEXT, type ReplayLocale } from '../i18n/i18n'
import type { PickupEventLine, VehicleDestroyedEventLine, VehicleEventLine } from '../model/feedEventLines'
import { FEED_FILTER_GLYPH } from '../model/feedGlyphs'
import { FEED_ROW } from '../model/feedRow'
import { vehicleKilledLabel } from '../model/vehicleKilledLabel'
import type { PlayerMarkKind } from '../../../lib/replay/playerMarks'
import { FeedName } from './ReplayFeedName'
import { FeedWeaponIcon } from './ReplayKillLineParts'
import { EventGlyph } from './ReplayEventGlyph'
import { FeedClock } from './ReplayFeedClock'

interface Props {
  line: PickupEventLine | VehicleEventLine | VehicleDestroyedEventLine
  replayMs: number
  /** Encre de camp du joueur (allégeance du film), neutre quand elle est inconnue. */
  color: string
  mark: PlayerMarkKind | undefined
  /** Le joueur qui ouvre la phrase ; vide pour une destruction sans auteur. */
  name: string
  /** Les noms des joueurs tués avec le véhicule lors d'une destruction ; absent ou vide : aucun. */
  killedNames?: readonly string[]
  locale: ReplayLocale
}

/** La phrase de la ligne, après le nom du joueur (ou seule, pour une destruction sans auteur). */
function actionOf(
  line: Props['line'],
  t: (typeof REPLAY_TEXT)['fr'],
  killedNames: readonly string[],
  locale: ReplayLocale,
): string {
  if (line.family === 'pickups') return line.ground ? t.feedGroundPickupAction(line.name) : t.feedPickupAction(line.name)
  if (!('destroyed' in line)) return t.feedVehicleAction(line.name)
  const sentence = line.xuid || line.botName ? t.feedVehicleDestroyedBy(line.name) : t.feedVehicleDestroyed(line.name)
  const killed = vehicleKilledLabel(killedNames, locale)
  return killed ? `${sentence} ${killed}` : sentence
}

export function ReplayEventLine({ line, replayMs, color, mark, name, killedNames = [], locale }: Props) {
  const t = REPLAY_TEXT[locale]
  const action = actionOf(line, t, killedNames, locale)
  const hasActor = name !== ''
  const weapon = 'destroyed' in line ? line.weapon : undefined
  return (
    <li className={FEED_ROW} title={hasActor ? `${name} ${action}` : action} data-feed-family={line.family}>
      <FeedClock ms={replayMs} />
      <span className="shrink-0" style={{ color }}>
        <EventGlyph glyph={FEED_FILTER_GLYPH[line.family]} />
      </span>
      {hasActor && <FeedName kind={mark} color={color} locale={locale} className="font-medium" name={name} />}
      {weapon && (
        <FeedWeaponIcon
          imageUrl={weapon.imageUrl}
          tinted={weapon.tinted}
          label={weapon.label || t.killFeedUnknownWeapon}
          color={color}
        />
      )}
      <span className="min-w-0 truncate text-muted-foreground">{action}</span>
    </li>
  )
}
