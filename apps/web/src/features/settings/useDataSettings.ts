/**
 * useDataSettings — état de saisie de la carte « Données » (lot L22) : brouillons des champs texte,
 * enregistrement par PATCH /settings, refus du serveur rattaché au champ qui l'a causé.
 *
 * Un champ texte se valide à la perte du focus ou par son bouton ; une liste se valide au choix.
 * Le refus arrive avec un `code` fixe, dont le texte est écrit dans le dictionnaire de la carte.
 */
import { useState } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { apiErrorCode } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'
import type { UpdateSettingsRequest } from '@/lib/api/types'
import { useUpdateSettings } from './queries'
import { errorText, fieldOfCode, parseWholeNumber, type DataField, type DataSettingKey } from './dataLogic'
import type { DataText } from './dataText'
import { flashSaveStatus } from './saveFeedbackStore'

type Errors = Partial<Record<DataField, string>>
type Drafts = Partial<Record<DataSettingKey, string>>

/** Les valeurs enregistrées des quatre réglages, en texte (ce que montre le champ au repos). */
export type SavedDataSettings = Record<DataSettingKey, string>

/** Code de refus d'une saisie qui n'est pas un entier : la requête ne part pas. */
const NOT_A_NUMBER_CODE: Partial<Record<DataSettingKey, string>> = {
  backup_keep: 'backup_keep_invalid',
  memory_limit_mb: 'memory_limit_invalid',
}

/** Le brouillon d'un champ, converti en requête ; `null` quand un nombre attendu n'en est pas un. */
function requestOf(key: DataSettingKey, raw: string): UpdateSettingsRequest | null {
  const text = raw.trim()
  if (key === 'backup_dir' || key === 'backup_interval') return { [key]: text }
  const n = parseWholeNumber(text)
  return n === null ? null : { [key]: n }
}

export interface DataSettingsForm {
  /** Valeur affichée d'un champ : le brouillon en cours, sinon le réglage enregistré. */
  value: (key: DataSettingKey) => string
  /** Le brouillon diffère du réglage enregistré. */
  dirty: (key: DataSettingKey) => boolean
  edit: (key: DataSettingKey, raw: string) => void
  /** Enregistre le brouillon du champ, ou `raw` quand la valeur vient d'un choix (rien à faire si elle n'a pas changé). */
  commit: (key: DataSettingKey, raw?: string) => void
  /** Pose un refus hors champ ; `undefined` l'efface. */
  setGeneralError: (message: string | undefined) => void
  errors: Errors
  saving: boolean
}

export function useDataSettings(t: DataText, current: SavedDataSettings): DataSettingsForm {
  const qc = useQueryClient()
  const update = useUpdateSettings()
  const [drafts, setDrafts] = useState<Drafts>({})
  const [errors, setErrors] = useState<Errors>({})

  const setDraft = (key: DataSettingKey, raw: string | undefined) =>
    setDrafts((prev) => ({ ...prev, [key]: raw }))
  const setError = (field: DataField, message: string | undefined) =>
    setErrors((prev) => ({ ...prev, [field]: message }))

  const value = (key: DataSettingKey) => drafts[key] ?? current[key]
  const dirty = (key: DataSettingKey) => drafts[key] !== undefined && drafts[key].trim() !== current[key]

  function commit(key: DataSettingKey, raw?: string) {
    const draft = raw ?? drafts[key]
    if (draft === undefined || update.isPending) return
    if (draft.trim() === current[key]) return setDraft(key, undefined)
    const req = requestOf(key, draft)
    setError(key, undefined)
    if (req === null) return setError(key, errorText(t, NOT_A_NUMBER_CODE[key]) ?? t.saveFailed)
    update.mutate(req, {
      onSuccess: () => {
        flashSaveStatus('saved')
        setDraft(key, undefined)
        void qc.invalidateQueries({ queryKey: queryKeys.dataStatus })
      },
      onError: (err) => {
        flashSaveStatus('error')
        const code = apiErrorCode(err)
        setError(fieldOfCode(code) ?? key, errorText(t, code) ?? t.saveFailed)
      },
    })
  }

  return {
    value,
    dirty,
    edit: setDraft,
    commit,
    setGeneralError: (message) => setError('general', message),
    errors,
    saving: update.isPending,
  }
}
