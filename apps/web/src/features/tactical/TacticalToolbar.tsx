/**
 * TacticalToolbar — question / qui / spawn de départ, au-dessus de la vue d'analyse
 * (item 5.2). TROIS CONTRÔLES, AUCUN NOUVEAU COMPOSANT : le `Select` et le `Button`
 * existants du dépôt suffisent — la segmentation « qui » est trois boutons `variant`
 * (sélectionné en `default`, les autres en `outline`), pas un composant Segmented dédié
 * pour trois valeurs qui n'existe nulle part ailleurs dans le dépôt.
 *
 * « Escouade » est DÉSACTIVÉ sans coéquipier dans la composition (même règle que le
 * contrat : `qui: escouade` exige des coéquipiers) — un bouton actif qui ne changerait
 * rien à la lecture mentirait sur ce qu'il fait.
 */
import { Button } from '@/components/ui/button'
import { Select } from '@/components/ui/select'
import type { TacticalGrappe } from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'

import type { TacticalText } from './i18n'
import type { TacticalQuestion, TacticalQui } from './tacticalView.logic'

export interface TacticalToolbarProps {
  t: TacticalText
  locale: Locale
  question: TacticalQuestion
  onQuestionChange: (question: TacticalQuestion) => void
  qui: TacticalQui
  onQuiChange: (qui: TacticalQui) => void
  escouadeDisponible: boolean
  spawn: string
  onSpawnChange: (spawn: string) => void
  grappes: readonly TacticalGrappe[]
}

const QUI_VALEURS: readonly TacticalQui[] = ['moi', 'escouade', 'adv']

export function TacticalToolbar({
  t,
  locale,
  question,
  onQuestionChange,
  qui,
  onQuiChange,
  escouadeDisponible,
  spawn,
  onSpawnChange,
  grappes,
}: TacticalToolbarProps) {
  const libelleQui: Record<TacticalQui, string> = {
    moi: t.whoMe,
    escouade: t.whoSquad,
    adv: t.whoOpponents,
  }

  return (
    <div
      className="flex flex-wrap items-end gap-4 border-b border-border px-3 py-2"
      data-testid="tactical-toolbar"
    >
      <label className="flex flex-col gap-1 text-xs text-muted-foreground">
        {t.questionLabel}
        <Select
          value={question}
          onChange={(e) => onQuestionChange(e.target.value as TacticalQuestion)}
          className="w-48"
          aria-label={t.questionLabel}
        >
          {t.analysisQuestions.map((q) => (
            <option key={q.id} value={q.id}>
              {q.label}
            </option>
          ))}
        </Select>
      </label>

      <div className="flex flex-col gap-1 text-xs text-muted-foreground">
        {t.whoLabel}
        <div className="flex gap-1" role="group" aria-label={t.whoLabel}>
          {QUI_VALEURS.map((valeur) => (
            <Button
              key={valeur}
              type="button"
              size="sm"
              variant={qui === valeur ? 'default' : 'outline'}
              aria-pressed={qui === valeur}
              disabled={valeur === 'escouade' && !escouadeDisponible}
              onClick={() => onQuiChange(valeur)}
            >
              {libelleQui[valeur]}
            </Button>
          ))}
        </div>
      </div>

      <label className="flex flex-col gap-1 text-xs text-muted-foreground">
        {t.spawnLabel}
        <Select
          value={spawn}
          onChange={(e) => onSpawnChange(e.target.value)}
          className="w-48"
          aria-label={t.spawnLabel}
        >
          <option value="">{t.spawnAll}</option>
          {grappes.map((g) => (
            <option key={g.id} value={g.id}>
              {locale === 'fr' ? g.nom_fr : g.nom_en}
            </option>
          ))}
        </Select>
      </label>
    </div>
  )
}
