package domain

import "time"

// settings_film_archive.go — les réglages de l'ARCHIVE DES FILMS (lot L14, ADR 0045 D-12) et la
// mesure de son occupation. Les trois types d'écriture/lecture des réglages l'EMBARQUENT (champs
// aplatis dans le JSON) : la forme de ces deux clés se déclare ici, une fois.

// FilmArchiveSettings : ce que app_settings.json et GET /settings portent de l'archive.
type FilmArchiveSettings struct {
	// ArchiveFilms : conserver les chunks des films après leur cuisson. Défaut OUI (clé absente
	// = vrai) : un film expire côté serveur Halo, l'archive est la seule copie. Non : les faits
	// et le rejeu sont produits, les chunks sont retirés après la cuisson, et la rétention des
	// rejeux est forcée illimitée (la purge suppose une recuisson depuis l'archive).
	ArchiveFilms bool `json:"archive_films" doc:"Conserver les films (chunks) après la cuisson du rejeu. Non : les chunks sont retirés après cuisson et la rétention des rejeux est illimitée."`
	// FilmArchiveDir : dossier de l'archive, chemin absolu. Vide = <données>/cache. Changé à chaud
	// tant que l'archive est vide ; sinon le changement est refusé (commande de déplacement).
	FilmArchiveDir string `json:"film_archive_dir" doc:"Dossier de l'archive des films (chemin absolu). Vide = emplacement par défaut. Appliqué tout de suite tant que l'archive est vide."`
}

// FilmArchiveUpdate : les deux clés, en écriture partielle (PATCH /settings).
type FilmArchiveUpdate struct {
	ArchiveFilms   *bool   `json:"archive_films,omitempty"`
	FilmArchiveDir *string `json:"film_archive_dir,omitempty"`
}

// FilmArchiveStatus est la réponse de GET /settings/film-archive : où est l'archive, ce qu'elle
// occupe et ce qu'il reste sur son volume.
type FilmArchiveStatus struct {
	// Dir : le dossier EFFECTIF de l'archive (celui que le processus a résolu au démarrage).
	Dir string `json:"dir" doc:"Dossier effectif de l'archive des films."`
	// ConfiguredDir : la valeur du réglage. Différente de Dir tant que le serveur n'a pas redémarré.
	ConfiguredDir string `json:"configured_dir" doc:"Valeur enregistrée du réglage ; vide = emplacement par défaut."`
	// RestartRequired : le réglage enregistré n'est pas encore celui qui sert.
	RestartRequired bool  `json:"restart_required" doc:"Vrai quand le dossier enregistré diffère du dossier effectif (redémarrage nécessaire)."`
	ArchiveFilms    bool  `json:"archive_films" doc:"Réglage « archiver les films »."`
	Films           int   `json:"films" doc:"Nombre de films dans l'archive."`
	Bytes           int64 `json:"bytes" doc:"Taille occupée par l'archive, en octets."`
	// Partial : la mesure a été interrompue (délai) ; Films et Bytes sont des minorants.
	Partial bool `json:"partial" doc:"Vrai si la mesure a été interrompue : les totaux sont des minorants."`
	// MeasuredAt : instant de la mesure (elle est mise en cache, jamais refaite à chaque requête).
	MeasuredAt time.Time `json:"measured_at" doc:"Instant de la mesure (mise en cache)."`
	// Volume : l'espace du volume qui porte l'archive. Absent quand la mesure est indisponible.
	Volume *FilmArchiveVolume `json:"volume,omitempty"`
}

// FilmArchiveVolume : l'espace disque du volume de l'archive.
type FilmArchiveVolume struct {
	FreeBytes   uint64 `json:"free_bytes"`
	TotalBytes  uint64 `json:"total_bytes"`
	UsedPercent int    `json:"used_percent"`
	// Status : ok | warn | critical, selon les seuils de la surveillance disque (80 / 90 %).
	Status string `json:"status" enum:"ok,warn,critical,unknown"`
}
