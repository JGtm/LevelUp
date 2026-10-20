/**
 * WeaponRecordsGapCard — « Records de distance par arme » quand le film du périmètre ne donne aucun frag
 * mesuré : la carte garde son titre et dit pourquoi (« Analyse du film en attente » ou « Film non analysé »)
 * dans son bloc, au lieu de disparaître. Le serveur nomme la cause (`weapon_records_gap`, règle unique
 * d'analyse en cours) ; un titre qui ne mesure pas de positions (capability `weapon_range` absente) n'en
 * a pas et la page n'affiche rien.
 */
import { SectionCard } from '@/components/ui/section-card'
import { FilmGapState } from '@/features/_shared/filmGap/FilmGapSection'
import type { FilmGap } from '@/features/_shared/filmGap/filmGap.logic'
import { FILM_GAP_TEXT } from '@/features/_shared/filmGap/filmGapStrings'
import { formatMessage } from '@/lib/i18n/format'
import { synthesisManifest } from '@/lib/i18n/generated/synthesis'
import { useAppShellStore } from '@/stores/appShellStore'

export function WeaponRecordsGapCard({ gap }: { gap: FilmGap }) {
  const locale = useAppShellStore((s) => s.locale)
  const title = formatMessage(synthesisManifest, 'synthesis.weapon_records.title', locale)
  return (
    <SectionCard title={title} label={title}>
      <div className="px-3 py-4" data-testid="weapon-records-gap">
        <FilmGapState gap={gap} t={FILM_GAP_TEXT[locale]} />
      </div>
    </SectionCard>
  )
}
