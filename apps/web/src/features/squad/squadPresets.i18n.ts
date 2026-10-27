/**
 * Libellés FR/EN du sélecteur d'escouades (presets + gestion).
 *
 * Dict local par feature (suffixe `.i18n.ts`) : source de vérité des chaînes UI
 * de useSquadPresets, hors scan FieldKey (lint-no-hardcoded-fields).
 */
import type { Locale } from '@/lib/i18n/locale'

export interface SquadPresetsStrings {
  squadsHeader: string
  groupsHeader: string
  save: string
  saving: string
  saved: string
  saveSuccess: string
  saveError: string
  manage: string
  done: string
  rename: string
  ok: string
  del: string
  confirmDelete: string
  usualPrefix: string
}

export const SQUAD_PRESETS_STRINGS: Record<Locale, SquadPresetsStrings> = {
  fr: {
    squadsHeader: 'Escouades enregistrées',
    groupsHeader: 'Groupes',
    save: 'Enregistrer la compo',
    saving: 'Enregistrement…',
    saved: 'Compo déjà enregistrée',
    saveSuccess: 'Escouade enregistrée',
    saveError: "Échec de l'enregistrement",
    manage: 'Gérer',
    done: 'Terminé',
    rename: 'Renommer',
    ok: 'OK',
    del: 'Suppr.',
    confirmDelete: 'Confirmer ?',
    usualPrefix: 'surtout',
  },
  en: {
    squadsHeader: 'Saved squads',
    groupsHeader: 'Groups',
    save: 'Save lineup',
    saving: 'Saving…',
    saved: 'Lineup already saved',
    saveSuccess: 'Squad saved',
    saveError: 'Save failed',
    manage: 'Manage',
    done: 'Done',
    rename: 'Rename',
    ok: 'OK',
    del: 'Delete',
    confirmDelete: 'Confirm?',
    usualPrefix: 'mostly',
  },
}

