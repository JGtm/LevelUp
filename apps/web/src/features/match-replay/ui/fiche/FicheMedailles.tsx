/**
 * FicheMedailles — « MÉDAILLES » gagnées jusqu'à l'instant lu, comme sur les tuiles de match : l'icône de
 * 32 px (`MedalIcon`, image ou découpe de sprites) avec le halo de sa rareté, son nom dessous sur une ligne
 * tronquée, « ×n ». Une rangée centrée qui passe à la ligne.
 *
 * Une médaille sans visuel au référentiel garde son nom : aucune image n'est inventée. Le rendu ne lit rien.
 */
import { MedalIcon } from '@/components/ui/MedalIcon'
import { dropShadowForDifficulty } from '@/lib/medalDifficulty'

import { REPLAY_TEXT, type ReplayLocale } from '../../i18n/i18n'
import { COTES_FICHE_INSTANT } from '../../model/cotesFicheInstant'
import type { MedalGroup } from '../../model/playerInstantMedals'
import { FicheSection } from './ficheParts'

const C = COTES_FICHE_INSTANT.medailles

function Medal({ medal }: { medal: MedalGroup }) {
  const glow = dropShadowForDifficulty(medal.difficulty ?? undefined)
  return (
    <div
      data-fiche-medaille={medal.name}
      title={medal.description || medal.label}
      className="flex cursor-default flex-col items-center"
      style={{ width: C.largeurPx, gap: C.ecartVerticalPx }}
    >
      <MedalIcon
        imageUrl={medal.imageUrl}
        spriteSheet={medal.sprite?.sheet}
        spriteLeft={medal.sprite?.left}
        spriteTop={medal.sprite?.top}
        spriteWidth={medal.sprite?.width}
        spriteHeight={medal.sprite?.height}
        label={medal.label}
        size={C.iconePx}
        className="object-contain"
        style={glow ? { filter: glow } : undefined}
      />
      <span className="max-w-full truncate text-center leading-tight text-muted-foreground" style={{ fontSize: C.taillePx }}>
        {medal.label}
      </span>
      <span className="font-semibold leading-none text-foreground/70" style={{ fontSize: C.taillePx }}>
        ×{medal.count}
      </span>
    </div>
  )
}

export function FicheMedailles({ medals, locale }: { medals: readonly MedalGroup[]; locale: ReplayLocale }) {
  const t = REPLAY_TEXT[locale].playerInstant
  return (
    <FicheSection title={t.medals}>
      <div className="flex flex-wrap justify-center" style={{ gap: C.ecartPx, paddingTop: C.margeHautePx }}>
        {medals.map((m) => (
          <Medal key={m.name} medal={m} />
        ))}
      </div>
    </FicheSection>
  )
}
