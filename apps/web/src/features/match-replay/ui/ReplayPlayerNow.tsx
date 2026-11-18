/**
 * ReplayPlayerNow — LE BLOC « À L'INSTANT » DE LA FICHE D'UN JOUEUR : ce qu'il est et porte à
 * l'image lue, vivant ou mort.
 *
 * DE HAUT EN BAS : le titre et l'instant lu ; le bouclier puis la santé (libellé, piste, valeur en
 * pour-cent du plein) ; les deux armes — celle en main, puis celle en réserve, éteinte — chacune avec
 * ses munitions ; les types de grenade portés avec leur stock ; l'équipement porté avec ses charges.
 * Un joueur mort à l'instant lu n'a ni jauges ni armes : l'encadré « Éliminé » de la fiche de la
 * colonne, avec son décompte, remplace tout sous le titre.
 *
 * LE BLOC NE LIT RIEN ET NE CALCULE RIEN : `model/playerNowReadings.ts` relit, par les fonctions de
 * la fiche de la colonne, l'état à l'image donnée ; ce composant rend ce qu'on lui dit. Il reçoit
 * tout par props — joueur, document, image, horloge écrite, âges de lecture — et ne dépend ni du
 * panneau ni d'un conteneur : il se monte tel quel dans n'importe quelle surface.
 *
 * LES RÈGLES DES FICHES : une valeur non lue n'est PAS rendue (rangée absente, jamais un zéro ni un
 * mot d'absence) ; une lecture ancienne PÂLIT et dit son âge en infobulle, chaque groupe sur l'âge de
 * SA lecture et jamais emboîté (les opacités se multiplieraient) ; aucun littéral de couleur — les
 * rôles passent par les jetons du thème. Les cotes sont `COTES_BLOC_INSTANT`.
 */
import { WeaponIcon } from '@/components/ui/WeaponIcon'
import { tokenCssVar } from '@/lib/accessibility/semantic-tokens'

import type { ReadingAges } from '../hooks/useCardScene'
import { REPLAY_TEXT, type ReplayLocale } from '../i18n/i18n'
import { COTES_BLOC_INSTANT as C } from '../model/cardGabarit'
import { grenadeBoxHint } from '../model/inventoryReading'
import {
  playerNowReadings,
  readingAgeTitle,
  type NowAbility,
  type NowGauge,
  type NowGrenade,
  type NowWeapon,
  type PlayerNowReadings,
} from '../model/playerNowReadings'
import { MIRROR_STYLE } from '../model/weaponFullIcon'
import { formatSeconds, frameToMs, freshness, READING_FADE } from '../../../lib/replay/replayLogic'
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import type { ReplayPlayer } from '../../../lib/replay/rosterLogic'
import { EliminatedBox } from './ReplayVitality'

type ReplayText = (typeof REPLAY_TEXT)[ReplayLocale]

export interface ReplayPlayerNowProps {
  player: ReplayPlayer
  doc: ReplayDocumentReady
  /** L'image lue. */
  frame: number
  /** L'instant lu, déjà écrit (« 3:51 ») : le titre le porte. */
  clock: string
  /** Les vitalités publiées et les âges d'estompage du document (`useReadingAges`). */
  ages: ReadingAges
  locale: ReplayLocale
}

/** Tout ce que les rangées partagent : le document, les textes, les âges et les lectures de l'image. */
interface Ctx {
  doc: ReplayDocumentReady
  t: ReplayText
  ages: ReadingAges
  read: PlayerNowReadings
}

const ROW = 'flex items-center text-[12px]'

/** L'encre d'une rangée : celle du texte, ou l'encre atténuée d'une arme rangée et d'un type de grenade non désigné. */
function ink(stowed: boolean): string {
  return stowed ? 'text-muted-foreground' : 'text-foreground'
}

/** L'opacité d'une lecture selon son âge ; sans âge (rien de lu), pleine. */
function fade(age: number | null, full: number): number {
  return age === null ? 1 : freshness(age, full, READING_FADE)
}

function GaugeRow({ label, gauge, color, vitalityFade }: {
  label: string
  gauge: NowGauge | null
  color: string
  vitalityFade: number
}) {
  if (!gauge) return null
  return (
    <div
      className="grid items-center text-[11.5px] text-muted-foreground"
      style={{
        gridTemplateColumns: `${C.jaugeLibelleW}px minmax(0, 1fr) ${C.jaugeValeurW}px`,
        columnGap: C.ecart,
        opacity: fade(gauge.age, vitalityFade),
      }}
    >
      <span>{label}</span>
      <span
        role="meter"
        aria-label={label}
        aria-valuemin={0}
        aria-valuemax={100}
        aria-valuenow={gauge.pct}
        className="relative block overflow-hidden"
        style={{
          height: C.jaugePx,
          borderRadius: C.jaugeRayonPx,
          background: `color-mix(in srgb, var(--foreground) ${C.pistePct}%, transparent)`,
        }}
      >
        <span
          className="absolute inset-y-0 left-0 block"
          style={{ width: `${gauge.pct}%`, background: color, borderRadius: C.jaugeRayonPx }}
        />
      </span>
      <span className="text-right tabular-nums text-foreground">{gauge.pct}</span>
    </div>
  )
}

/** Les munitions d'une arme : le compte et sa réserve, ou la charge restante ; « pleines » ne s'écrit qu'en infobulle. */
function ammoParts(w: NowWeapon): { value: string; reserve: string } | null {
  if (!w.ammo || w.ammo.kind === 'full') return null
  if (w.ammo.kind === 'charge') return { value: `${w.ammo.pct} %`, reserve: '' }
  return { value: `${w.ammo.mag}`, reserve: w.ammo.res === undefined ? '' : `/${w.ammo.res}` }
}

function WeaponRow({ w, ctx }: { w: NowWeapon; ctx: Ctx }) {
  const { doc, t, ages, read } = ctx
  const ammo = ammoParts(w)
  const place = w.place === 'hand' ? t.playerInstant.weaponInHand : w.place === 'stowed' ? t.playerInstant.weaponStowed : null
  const weaponsFade = fade(read.weaponsAge, ages.readingFull)
  const weaponsTitle =
    read.weaponsAge === null
      ? undefined
      : [read.weaponsBirth ? t.loadoutBirth : null, readingAgeTitle(t.loadoutAge, t.loadoutAhead, read.weaponsAge, doc)]
          .filter((p): p is string => !!p)
          .join(' · ')
  const ammoTitle = [
    w.ammo?.kind === 'full' ? t.ammoFullLabel : null,
    read.ammoAge === null ? null : readingAgeTitle(t.inventoryAge, t.inventoryAhead, read.ammoAge, doc),
  ]
    .filter((p): p is string => !!p)
    .join(' · ')
  return (
    <div className={`${ROW} ${ink(w.place === 'stowed')}`} style={{ gap: C.ecart }}>
      <span className="flex min-w-0 items-center" style={{ gap: C.ecart, opacity: weaponsFade }} title={weaponsTitle}>
        {w.icon && (
          <WeaponIcon
            imageUrl={w.icon.url}
            tinted={w.icon.tinted}
            label={w.name}
            width={C.armeIconeW}
            height={C.armeIconeH}
            style={w.icon.mirrored ? MIRROR_STYLE : undefined}
          />
        )}
        <span className="min-w-0 truncate">{w.name}</span>
      </span>
      {ammo && (
        <span className="shrink-0" style={{ opacity: fade(read.ammoAge, ages.readingFull) }} title={ammoTitle || undefined}>
          <span className="tabular-nums">{ammo.value}</span>
          {ammo.reserve && <small className="text-muted-foreground tabular-nums">{ammo.reserve}</small>}
        </span>
      )}
      {place && (
        <small className="shrink-0 text-muted-foreground" style={{ opacity: weaponsFade }}>
          {place}
        </small>
      )}
    </div>
  )
}

function GrenadeRow({ g, ctx }: { g: NowGrenade; ctx: Ctx }) {
  const { doc, t, ages, read } = ctx
  const title = read.grenadeBox ? grenadeBoxHint(t, read.grenadeBox, read.grenades, doc) : undefined
  return (
    <div
      className={`${ROW} ${ink(g.stowed)}`}
      style={{ gap: C.ecart, opacity: fade(read.grenadesAge, ages.readingFull) }}
      title={title}
    >
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
    </div>
  )
}

function AbilityRow({ a, ctx }: { a: NowAbility; ctx: Ctx }) {
  const { doc, t, ages } = ctx
  const full = a.charges?.kind === 'full'
  const title = [
    `${t.abilityLabel} — ${a.name}`,
    readingAgeTitle(t.abilityAge, t.abilityAhead, a.age, doc),
    full ? t.abilityChargesFullHint : null,
  ]
    .filter((p): p is string => !!p)
    .join(' · ')
  return (
    <div className={`${ROW} ${ink(false)}`} style={{ gap: C.ecart }}>
      <span className="flex min-w-0 items-center" style={{ gap: C.ecart, opacity: fade(a.age, ages.readingFull) }} title={title}>
        {a.icon && (
          <WeaponIcon
            imageUrl={a.icon.url}
            tinted={a.icon.tinted}
            label={a.name}
            width={C.equipementIconePx}
            height={C.equipementIconePx}
          />
        )}
        <span className="min-w-0 truncate">{a.name}</span>
      </span>
      {a.charges?.kind === 'count' && (
        <span
          className="shrink-0 tabular-nums"
          style={{ opacity: fade(a.charges.age, ages.readingFull) }}
          title={`${t.abilityChargesAge} ${formatSeconds(frameToMs(a.charges.age, doc))}`}
        >
          · {t.playerInstant.equipmentCharges(a.charges.charges)}
        </span>
      )}
    </div>
  )
}

export function ReplayPlayerNow({ player, doc, frame, clock, ages, locale }: ReplayPlayerNowProps) {
  const t = REPLAY_TEXT[locale]
  const read = playerNowReadings({ player, doc, frame, presence: ages.presence, locale })
  const ctx: Ctx = { doc, t, ages, read }
  return (
    <section
      aria-label={t.playerInstant.now}
      data-replay-player-now=""
      className="flex flex-col"
      style={{ gap: C.ecart }}
    >
      <div
        className="flex items-baseline text-[11px] font-semibold uppercase tracking-[.2px] text-muted-foreground"
        style={{ gap: C.ecart }}
      >
        {t.playerInstant.now}
        <span className="font-normal normal-case tabular-nums">{clock}</span>
      </div>
      {read.eliminated ? (
        <div style={{ height: C.corpsEliminePx }}>
          <EliminatedBox state={read.state} doc={doc} frame={frame} locale={locale} bodyPx={C.corpsEliminePx} />
        </div>
      ) : (
        <>
          <GaugeRow label={t.shieldLabel} gauge={read.shield} color={tokenCssVar('info')} vitalityFade={ages.vitalityFade} />
          <GaugeRow label={t.healthLabel} gauge={read.health} color="var(--foreground)" vitalityFade={ages.vitalityFade} />
          {read.weapons.map((w, k) => (
            <WeaponRow key={`${w.id}-${k}`} w={w} ctx={ctx} />
          ))}
          {read.grenades.map((g) => (
            <GrenadeRow key={g.rank} g={g} ctx={ctx} />
          ))}
          {read.ability && <AbilityRow a={read.ability} ctx={ctx} />}
        </>
      )}
    </section>
  )
}
