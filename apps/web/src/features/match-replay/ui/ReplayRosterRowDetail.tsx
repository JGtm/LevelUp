/**
 * ReplayRosterRowDetail — LE DÉPLI DE LA LIGNE DU JOUEUR SUIVI : sous sa ligne, sur deux colonnes, les
 * munitions de l'arme en main, l'arme en réserve éteinte avec les siennes, chaque type de grenade
 * porté avec son stock, et l'équipement avec ses charges.
 *
 * LES LECTURES SONT CELLES DU BLOC « À L'INSTANT » (`playerNowReadings`) : mêmes armes ordonnées,
 * mêmes munitions, mêmes grenades, même équipement, mêmes âges — la ligne ne relit rien d'autre.
 * LES RÈGLES DES FICHES VALENT ICI : une valeur non lue n'est pas rendue (rangée absente), une lecture
 * ancienne pâlit sur l'âge de SA lecture et le dit en infobulle. Un joueur mort, ou sans rien de lu,
 * n'a pas de dépli. Les cotes sont `COTES_DEPLI_LIGNE`.
 *
 * LE DÉPLI NE DESSINE PAS DE BARRE DE CHARGE : le document donne le nombre de charges restantes, pas le
 * plein de l'équipement, et une barre en inventerait la fraction.
 */
import type { ReactNode } from 'react'

import { WeaponIcon } from '@/components/ui/WeaponIcon'

import type { CardScene } from '../hooks/useCardScene'
import { COTES_DEPLI_LIGNE as C } from '../model/cardGabarit'
import {
  playerNowReadings,
  readingAgeTitle,
  type NowAbility,
  type NowGrenade,
  type NowWeapon,
  type PlayerNowReadings,
} from '../model/playerNowReadings'
import { ammoParts } from '../model/rosterRowReadings'
import { MIRROR_STYLE } from '../model/weaponFullIcon'
import { REPLAY_TEXT, type ReplayLocale } from '../i18n/i18n'
import { freshness, READING_FADE } from '../../../lib/replay/replayLogic'
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import type { ReplayPlayer } from '../../../lib/replay/rosterLogic'

type ReplayText = (typeof REPLAY_TEXT)[ReplayLocale]

/** Tout ce que les rangées du dépli partagent. */
interface Ctx {
  doc: ReplayDocumentReady
  t: ReplayText
  readingFull: number
  read: PlayerNowReadings
}

/** L'opacité d'une lecture selon son âge ; sans âge (rien de lu), pleine. */
function fade(age: number | null, full: number): number {
  return age === null ? 1 : freshness(age, full, READING_FADE)
}

/** Une rangée du dépli : vignette puis texte, sur une ligne, à l'écart de la vignette. */
function Item({ children, muted = false, opacity = 1, title }: {
  children: ReactNode
  muted?: boolean
  opacity?: number
  title?: string
}) {
  return (
    <span
      className={`flex min-w-0 items-center whitespace-nowrap ${muted ? 'text-muted-foreground' : 'text-foreground'}`}
      style={{ gap: C.ecartIcone, opacity }}
      title={title}
    >
      {children}
    </span>
  )
}

function WeaponItem({ w, ctx }: { w: NowWeapon; ctx: Ctx }) {
  const { doc, t, read, readingFull } = ctx
  const ammo = ammoParts(w)
  const place = w.place === 'hand' ? t.playerInstant.weaponInHand : w.place === 'stowed' ? t.playerInstant.weaponStowed : null
  const title = [
    w.name,
    place,
    w.ammo?.kind === 'full' ? t.ammoFullLabel : null,
    read.weaponsBirth ? t.loadoutBirth : null,
    read.weaponsAge === null ? null : readingAgeTitle(t.loadoutAge, t.loadoutAhead, read.weaponsAge, doc),
    read.ammoAge === null ? null : readingAgeTitle(t.inventoryAge, t.inventoryAhead, read.ammoAge, doc),
  ]
    .filter((p): p is string => !!p)
    .join(' · ')
  return (
    <Item muted={w.place === 'stowed'} opacity={fade(read.weaponsAge, readingFull)} title={title}>
      {w.icon ? (
        <WeaponIcon
          imageUrl={w.icon.url}
          tinted={w.icon.tinted}
          label={w.name}
          width={C.armeIconeW}
          height={C.armeIconeH}
          style={w.icon.mirrored ? MIRROR_STYLE : undefined}
        />
      ) : (
        <span className="truncate">{w.name}</span>
      )}
      {ammo && (
        <span className="tabular-nums" style={{ opacity: fade(read.ammoAge, readingFull) }}>
          {ammo.value}
          {ammo.reserve && (
            <small className="text-muted-foreground" style={{ fontSize: C.petitPx }}>
              {ammo.reserve}
            </small>
          )}
        </span>
      )}
    </Item>
  )
}

function GrenadeItem({ g, ctx }: { g: NowGrenade; ctx: Ctx }) {
  const { doc, t, read, readingFull } = ctx
  const title = read.grenadesAge === null ? undefined : readingAgeTitle(t.grenadeAge, t.grenadeAhead, read.grenadesAge, doc)
  return (
    <Item muted={g.stowed} opacity={fade(read.grenadesAge, readingFull)} title={title}>
      {g.icon && (
        <WeaponIcon
          imageUrl={g.icon.url}
          tinted={g.icon.tinted}
          label={g.name}
          width={C.grenadeIconePx}
          height={C.grenadeIconePx}
        />
      )}
      <span className="min-w-0 truncate">{`${g.name} ×${g.count}`}</span>
    </Item>
  )
}

function AbilityItem({ a, ctx }: { a: NowAbility; ctx: Ctx }) {
  const { doc, t, readingFull } = ctx
  const charges = a.charges?.kind === 'count' ? a.charges : null
  const title = [
    `${t.abilityLabel} — ${a.name}`,
    charges ? t.playerInstant.equipmentCharges(charges.charges) : null,
    a.charges?.kind === 'full' ? t.abilityChargesFullHint : null,
    readingAgeTitle(t.abilityAge, t.abilityAhead, a.age, doc),
  ]
    .filter((p): p is string => !!p)
    .join(' · ')
  return (
    <Item opacity={fade(a.age, readingFull)} title={title}>
      {a.icon && (
        <WeaponIcon
          imageUrl={a.icon.url}
          tinted={a.icon.tinted}
          label={a.name}
          width={C.equipementIconePx}
          height={C.equipementIconePx}
        />
      )}
      {charges ? <span className="tabular-nums" style={{ opacity: fade(charges.age, readingFull) }}>{charges.charges}</span> : <span className="truncate">{a.name}</span>}
    </Item>
  )
}

export interface ReplayRosterRowDetailProps {
  player: ReplayPlayer
  doc: ReplayDocumentReady
  frame: number
  locale: ReplayLocale
  scene: CardScene
}

export function ReplayRosterRowDetail({ player, doc, frame, locale, scene }: ReplayRosterRowDetailProps) {
  const t = REPLAY_TEXT[locale]
  const read = playerNowReadings({ player, doc, frame, presence: scene.presence, locale })
  if (read.eliminated || !read.state.alive) return null
  const { weapons, grenades, ability } = read
  if (weapons.length === 0 && grenades.length === 0 && !ability) return null
  const ctx: Ctx = { doc, t, readingFull: scene.readingFull, read }
  return (
    <div
      data-replay-roster-detail=""
      aria-label={t.playerInstant.now}
      className="grid text-muted-foreground"
      style={{
        gridTemplateColumns: `repeat(${C.colonnes}, minmax(0, 1fr))`,
        columnGap: C.ecartH,
        rowGap: C.ecartV,
        paddingTop: C.margeHaute,
        paddingBottom: C.margeBasse,
        fontSize: C.textePx,
      }}
    >
      {weapons.map((w, k) => (
        <WeaponItem key={`${w.id}-${k}`} w={w} ctx={ctx} />
      ))}
      {grenades.map((g) => (
        <GrenadeItem key={g.rank} g={g} ctx={ctx} />
      ))}
      {ability && <AbilityItem a={ability} ctx={ctx} />}
    </div>
  )
}
