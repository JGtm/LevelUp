/**
 * BadgeIcon — picto Fluent Emoji Flat (Microsoft) pour les badges d'impact.
 *
 * Source : SVG vendorés depuis github.com/microsoft/fluentui-emoji (variante
 * Flat) dans `apps/web/src/assets/badges/fluent-flat/`. Style 2D plat avec
 * dégradés doux, choisi pour réduire l'effet de volume des emojis natifs.
 *
 * Mapping des aliases (clés backend antérieures au portage analysis Go) :
 *   tourist      → last_group_kill (snail)
 *   finisher     → clutch_finisher (bullseye)
 *   first_victim → first_group_death (headstone)
 */
import firstBlood from '@/assets/badges/fluent-flat/first_blood.svg'
import clutchFinisher from '@/assets/badges/fluent-flat/clutch_finisher.svg'
import lastCasualty from '@/assets/badges/fluent-flat/last_casualty.svg'
import lastGroupKill from '@/assets/badges/fluent-flat/last_group_kill.svg'
import firstGroupDeath from '@/assets/badges/fluent-flat/first_group_death.svg'
import silentHero from '@/assets/badges/fluent-flat/silent_hero.svg'
import falseBrother from '@/assets/badges/fluent-flat/false_brother.svg'
import topKiller from '@/assets/badges/fluent-flat/top_killer.svg'
import topGun from '@/assets/badges/fluent-flat/top_gun.svg'
import kamikaze from '@/assets/badges/fluent-flat/kamikaze.svg'
import champion from '@/assets/badges/fluent-flat/champion.svg'
import maillonFaible from '@/assets/badges/fluent-flat/maillon_faible.svg'
import passagerClandestin from '@/assets/badges/fluent-flat/passager_clandestin.svg'

const BADGE_SVG: Record<string, string> = {
  first_blood: firstBlood,
  clutch_finisher: clutchFinisher,
  last_casualty: lastCasualty,
  last_group_kill: lastGroupKill,
  first_group_death: firstGroupDeath,
  silent_hero: silentHero,
  false_brother: falseBrother,
  top_killer: topKiller,
  top_gun: topGun,
  kamikaze,
  champion,
  maillon_faible: maillonFaible,
  passager_clandestin: passagerClandestin,
  // Aliases backend
  tourist: lastGroupKill,
  finisher: clutchFinisher,
  first_victim: firstGroupDeath,
}

export interface BadgeIconProps {
  badgeKey: string
  /** Taille en pixels (carrée). Défaut : 16. */
  size?: number
  /** Tooltip natif (déjà localisé). */
  title?: string
  /** ClassName additionnel pour le wrapper <img>. */
  className?: string
}

export function BadgeIcon({ badgeKey, size = 16, title, className }: BadgeIconProps) {
  const src = BADGE_SVG[badgeKey]
  if (!src) {
    return (
      <span
        aria-label={badgeKey}
        title={title ?? badgeKey}
        className={className}
        style={{ display: 'inline-block', width: size, height: size, textAlign: 'center', lineHeight: `${size}px` }}
      >
        ·
      </span>
    )
  }
  return (
    <img
      src={src}
      alt={badgeKey}
      title={title}
      width={size}
      height={size}
      className={className}
      style={{ display: 'inline-block', verticalAlign: 'middle' }}
      data-badge-key={badgeKey}
    />
  )
}
