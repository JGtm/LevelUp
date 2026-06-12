/**
 * InlineResolveForm — formulaire de résolution inline (pattern reset-password
 * de la section Users) : N champs texte + Enregistrer/Annuler, rendu dans une
 * rangée bordée sous la ligne à résoudre.
 */
import { useState } from 'react'

import { Button } from '@/components/ui/button'
import { useAdminT } from '../useAdminText'

interface ResolveField {
  key: string
  label: string
  initial?: string
  placeholder?: string
}

interface InlineResolveFormProps {
  /** Contexte affiché en tête (ex. l'ID de l'asset, la clé du mode). */
  subject: string
  fields: ResolveField[]
  busy?: boolean
  /** Valide la saisie ; retourne false pour bloquer le submit (champ requis vide). */
  onSubmit: (values: Record<string, string>) => void
  onCancel: () => void
}

export function InlineResolveForm({ subject, fields, busy, onSubmit, onCancel }: InlineResolveFormProps) {
  const tA = useAdminT()
  const [values, setValues] = useState<Record<string, string>>(() => {
    const init: Record<string, string> = {}
    for (const f of fields) init[f.key] = f.initial ?? ''
    return init
  })
  const hasValue = fields.some((f) => (values[f.key] ?? '').trim().length > 0)

  return (
    <div className="flex flex-wrap items-center gap-2 rounded-md border border-dashed px-4 py-3">
      <span className="font-mono text-xs text-muted-foreground" title={subject}>
        {subject}
      </span>
      {fields.map((f) => (
        <label key={f.key} className="flex items-center gap-1.5 text-xs text-muted-foreground">
          {f.label}
          <input
            type="text"
            value={values[f.key] ?? ''}
            maxLength={128}
            onChange={(e) => setValues((prev) => ({ ...prev, [f.key]: e.target.value }))}
            className="w-48 rounded-md border border-input bg-background px-2 py-1 text-sm text-foreground"
            placeholder={f.placeholder}
          />
        </label>
      ))}
      <Button size="sm" disabled={busy || !hasValue} onClick={() => onSubmit(values)}>
        {tA('admin.dq.form_submit')}
      </Button>
      <Button size="sm" variant="outline" onClick={onCancel} disabled={busy}>
        {tA('admin.dq.form_cancel')}
      </Button>
    </div>
  )
}
