/**
 * CreateArcForm — formulaire de création d'un arc libre (non preset).
 *
 * Un arc libre regroupe plusieurs objectifs sous un fil narratif. La création
 * ne demande que 2 champs (cf. `CreateArcBody` : title requis, description
 * optionnelle) ; le rattachement des objectifs se fait ensuite via leur
 * `arc_id`. Backend : POST /arcs → prestige.CreateArc.
 */
import { useState } from 'react'
import { useAppShellStore } from '@/stores/appShellStore'
import type { CreateArcBody } from '@/lib/prestige'
import { getPrestigeText } from '../i18n'
import { useCreateArc } from '../hooks'

interface CreateArcFormProps {
  userId: string
  titleSlug: string
  onSuccess?: () => void
  onCancel?: () => void
}

export function CreateArcForm({ userId, titleSlug, onSuccess, onCancel }: CreateArcFormProps) {
  const locale = useAppShellStore((s) => s.locale)
  const t = getPrestigeText(locale)
  const [title, setTitle] = useState('')
  const [description, setDescription] = useState('')
  const create = useCreateArc(userId, titleSlug)

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    const trimmed = title.trim()
    if (!trimmed) return
    const body: CreateArcBody = {
      user_id: userId,
      title_slug: titleSlug,
      title: trimmed,
      description: description.trim() || undefined,
    }
    create.mutate(body, { onSuccess })
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-3">
      <header className="flex items-center justify-between">
        <h3 className="text-sm font-semibold">{t.arcFormNew}</h3>
        {onCancel && (
          <button
            type="button"
            onClick={onCancel}
            className="text-xs text-muted-foreground hover:text-foreground"
          >
            {t.formCancel}
          </button>
        )}
      </header>

      <label className="block">
        <span className="mb-1 block text-xs font-medium text-muted-foreground">{t.arcFormTitle}</span>
        <input
          type="text"
          value={title}
          onChange={(e) => setTitle(e.target.value)}
          required
          maxLength={128}
          placeholder={t.arcFormTitlePlaceholder}
          className="w-full rounded-md border border-border bg-background px-3 py-1.5 text-sm text-foreground"
        />
      </label>

      <label className="block">
        <span className="mb-1 block text-xs font-medium text-muted-foreground">{t.arcFormDescription}</span>
        <textarea
          value={description}
          onChange={(e) => setDescription(e.target.value)}
          maxLength={280}
          rows={2}
          placeholder={t.arcFormDescriptionPlaceholder}
          className="w-full resize-none rounded-md border border-border bg-background px-3 py-1.5 text-sm text-foreground"
        />
      </label>

      <div className="space-y-2">
        {create.error != null && (
          <p className="text-xs text-destructive">{(create.error as Error).message}</p>
        )}
        <button
          type="submit"
          disabled={create.isPending || !title.trim()}
          className="w-full rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
        >
          {create.isPending ? t.formCreating : t.arcFormCreate}
        </button>
      </div>
    </form>
  )
}
