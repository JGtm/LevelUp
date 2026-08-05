// Auto-genere par scripts/build_i18n_manifests.mjs - NE PAS EDITER A LA MAIN.
// Source : apps/web/src/lib/i18n/manifests/feedback_drawer.toml

export const feedbackDrawerManifest = {
  "feedback_drawer.attach.label": { fr: "Joindre les infos techniques", en: "Attach technical info" },
  "feedback_drawer.attach.preview_summary": { fr: "Aperçu Markdown", en: "Markdown preview" },
  "feedback_drawer.classification.preview": { fr: "Type {type} · Sévérité {severity} · Zone {area}", en: "Type {type} · Severity {severity} · Area {area}" },
  "feedback_drawer.field.description": { fr: "Description", en: "Description" },
  "feedback_drawer.field.description_placeholder": { fr: "Ce qui s'est passé, ce que tu attendais, étapes pour reproduire…", en: "What happened, what you expected, steps to reproduce…" },
  "feedback_drawer.field.title": { fr: "Titre", en: "Title" },
  "feedback_drawer.field.title_placeholder": { fr: "Résume ton retour en quelques mots", en: "Sum up your feedback in a few words" },
  "feedback_drawer.mini_tab.aria_close": { fr: "Fermer le panneau de retour", en: "Close feedback panel" },
  "feedback_drawer.mini_tab.aria_open": { fr: "Envoyer un retour", en: "Send feedback" },
  "feedback_drawer.popup_blocked": { fr: "Lien copié dans le presse-papier — collez-le dans un onglet pour ouvrir GitHub.", en: "Link copied to clipboard — paste it in a tab to open GitHub." },
  "feedback_drawer.rate_limit": { fr: "Merci, 5 retours ont déjà été envoyés dans la dernière heure.", en: "Thanks, 5 feedback submissions were already sent in the last hour." },
  "feedback_drawer.similar.label": { fr: "Une issue existe peut-être déjà :", en: "A similar issue may already exist:" },
  "feedback_drawer.submit": { fr: "Ouvrir sur GitHub", en: "Open on GitHub" },
  "feedback_drawer.submit_note": { fr: "Redirection vers GitHub pour finaliser l'envoi. Une analyse automatique enrichira l'issue.", en: "Redirecting to GitHub to finalize the submission. An automated analysis will enrich the issue." },
  "feedback_drawer.title": { fr: "Envoyer un retour", en: "Send feedback" },
  "feedback_drawer.type.aria": { fr: "Type de retour", en: "Feedback type" },
  "feedback_drawer.type.bug": { fr: "Bug", en: "Bug" },
  "feedback_drawer.type.idea": { fr: "Idée", en: "Idea" },
  "feedback_drawer.type.question": { fr: "Question", en: "Question" },
} as const

export type FeedbackDrawerManifestKey = keyof typeof feedbackDrawerManifest
