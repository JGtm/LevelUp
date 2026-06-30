/**
 * Libellés FR/EN du sélecteur d'escouades (presets + gestion).
 *
 * Dict local par feature (suffixe `.i18n.ts`) : source de vérité des chaînes UI
 * de useSquadPresets, hors scan FieldKey (lint-no-hardcoded-fields).
 */
export const SQUAD_PRESETS_STRINGS = {
  fr: {
    squadsHeader: 'Mes escouades',
    groupsHeader: 'Mes groupes',
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
    squadsHeader: 'My squads',
    groupsHeader: 'My groups',
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

export type SquadPresetsStrings = (typeof SQUAD_PRESETS_STRINGS)['fr']
