#!/usr/bin/env python3
"""Peuple la table citation_mappings dans metadata.duckdb.

Ce script est idempotent : il crée la table si absente, ajoute les colonnes
manquantes si besoin, et insère/met à jour toutes les citations
(PVP + PVE + composites) avec leurs métadonnées UI (image, category, tiers…).

Usage :
    python scripts/populate_citation_mappings.py
    python scripts/populate_citation_mappings.py --reset   # DROP + recréation
"""

from __future__ import annotations

import argparse
import sys
from pathlib import Path

import duckdb

# ── Chemin vers metadata.duckdb ─────────────────────────────────────────────
METADATA_DB = Path("data/warehouse/metadata.duckdb")

# ── DDL ─────────────────────────────────────────────────────────────────────
CREATE_TABLE_DDL = """
CREATE TABLE IF NOT EXISTS citation_mappings (
    citation_name_norm    VARCHAR PRIMARY KEY,
    citation_name_display VARCHAR NOT NULL,
    mapping_type          VARCHAR NOT NULL,
    medal_id              BIGINT,
    medal_ids             VARCHAR,
    stat_name             VARCHAR,
    award_name            VARCHAR,
    award_category        VARCHAR,
    custom_function       VARCHAR,
    composite_children    VARCHAR,
    confidence            VARCHAR,
    notes                 VARCHAR,
    enabled               BOOLEAN DEFAULT TRUE,
    created_at            TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at            TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    image_path            VARCHAR,
    category              VARCHAR,
    description           VARCHAR,
    tier_targets          VARCHAR,
    subcategory           VARCHAR
);
"""

# ── Citations PVP (45) ──────────────────────────────────────────────────────
# Tuple: (norm, display, type, medal_id, medal_ids, stat_name, award_name,
#          award_cat, custom_fn, composite_children, confidence, notes, enabled,
#          image_path, category, description, tier_targets, subcategory)
# fmt: off
PVP_CITATIONS: list[tuple] = [
    # ── GROUPE 1 : Mode de jeu (11) ────────────────────────────────────────
    ("charge", "À la charge", "award", None, None, None, "zone_captured", "objective", None, None, "high", "Mode de jeu — Zone capturée", True,
     "static/commendations/h5g/H5G_citation_%C3%80_la_charge.png", "Mode de jeu", "Prenez le contrôle d'une base dans n'importe quelle partie matchmaking Bases.", "10,20,30,50,100", None),
    ("forced_annexation", "Annexion forcée", "custom", None, None, None, None, None, "compute_annexion_forcee", None, "medium", "Mode de jeu — 3 Zone Capture consécutives", True,
     "static/commendations/h5g/H5G_citation_Annexion_forc%C3%A9e.png", "Mode de jeu", "Prenez le contrôle de 3 bases sans mourir dans n'importe quelle partie matchmaking Bases.", "3,6,9,15,30", None),
    ("assistant", "Assistant", "stat", None, None, "assists", None, None, None, None, "high", "Mode de jeu — Total assists", True,
     "static/commendations/h5g/H5G_citation_Assistant.png", "Mode de jeu", "Remportez n'importe quelle médaille d'assistance en Zone de combat.", "25,50,75,125,250", None),
    ("bulldozer", "Bulldozer", "custom", None, None, None, None, None, "compute_bulldozer", None, "high", "Mode de jeu — Parties Assassin KD > 8", True,
     "static/commendations/h5g/H5G_citation_Bulldozer.png", "Mode de jeu", "Terminez n'importe quelle partie matchmaking Assassin avec un FDA supérieur à 8.", "3,6,9,15,30", None),
    ("flag_defender", "Défenseur du drapeau", "award", None, None, None, "carrier_killed", "objective", None, None, "high", "Mode de jeu — Porteur tué", True,
     "static/commendations/h5g/H5G_citation_D%C3%A9fenseur_du_drapeau.png", "Mode de jeu", "Protégez le drapeau de votre équipe dans n'importe quelle partie matchmaking Capture du drapeau.", "10,20,30,50,100", None),
    ("got_you", "Je te tiens !", "award", None, None, None, "flag_returned", "objective", None, None, "high", "Mode de jeu — Drapeau ramené", True,
     "static/commendations/h5g/H5G_citation_Je_te_tiens_%21.png", "Mode de jeu", "Rapportez le drapeau de votre équipe dans n'importe quelle partie matchmaking Capture du drapeau.", "10,20,30,50,100", None),
    ("stakeholder", "Partie prenante", "award", None, None, None, "zone_secured", "objective", None, None, "high", "Mode de jeu — Zone sécurisée", True,
     "static/commendations/h5g/H5G_citation_Partie_prenante.png", "Mode de jeu", "Défendez une base appartenant à votre équipe dans n'importe quelle partie matchmaking Bases.", "10,20,30,50,100", None),
    ("flag_carrier_hunter", "Sus au porteur du drapeau", "award", None, None, None, "carrier_killed", "objective", None, None, "high", "Mode de jeu — Porteur tué", True,
     "static/commendations/h5g/H5G_citation_Sus_au_porteur_du_drapeau.png", "Mode de jeu", "Tuez un porte-drapeau ennemi dans n'importe quelle partie matchmaking Capture du drapeau.", "10,20,30,50,100", None),
    ("flag_victory", "Victoire au drapeau", "custom", None, None, None, None, None, "compute_wins_ctf", None, "high", "Mode de jeu — Victoires CTF", True,
     "static/commendations/h5g/H5G_citation_Victoire_au_drapeau.png", "Mode de jeu", "Remportez n'importe quelle partie matchmaking Capture du drapeau.", "5,10,15,25,50", None),
    ("slayer_victory", "Victoire en assassin", "custom", None, None, None, None, None, "compute_wins_slayer", None, "high", "Mode de jeu — Victoires Slayer", True,
     "static/commendations/h5g/H5G_citation_Victoire_en_Assassin.png", "Mode de jeu", "Remportez une partie d'Assassin en équipe.", "5,10,15,25,50", None),
    ("strongholds_victory", "Victoire en bases", "custom", None, None, None, None, None, "compute_wins_strongholds", None, "high", "Mode de jeu — Victoires Strongholds", True,
     "static/commendations/h5g/H5G_citation_Victoire_en_Bases.png", "Mode de jeu", "Remportez n'importe quelle partie matchmaking Bases.", "5,10,15,25,50", None),
    # ── GROUPE 2 : Arme — Véhicule (2) ─────────────────────────────────────
    ("splatter", "Écrasement", "medal", 221693153, None, None, None, None, None, None, "high", "Véhicule — Médaille Splatter/Écrasement", True,
     "static/commendations/h5g/H5G_citation_%C3%89crasement.png", "Véhicule", "Écrasez un Spartan adverse avec un véhicule.", "10,20,30,50,100", "Général"),
    ("driver", "Pilote", "medal", 3169118333, None, None, None, None, None, None, "high", "Véhicule — Médaille Violence routière", True,
     "static/commendations/h5g/H5G_citation_Pilote.png", "Véhicule", "Décrochez des médailles de pilote.", "10,20,30,50,100", "Général"),
    # ── GROUPE 2b : Arme — Grenade (2) ─────────────────────────────────────
    ("frag_grenade", "Grenade à fragmentation", "medal", 2648272972, None, None, None, None, None, None, "high", "Arme — Médaille Grenadier (frag)", True,
     "static/commendations/h5g/H5G_citation_Grenade_%C3%A0_fragmentation.png", "Arme", "Tuez un Spartan adverse à l'aide d'une grenade à fragmentation.", "5,10,15,25,50", "Grenade"),
    ("plasma_grenade", "Grenade à plasma", "medal", 3655682764, None, None, None, None, None, None, "high", "Arme — Médaille Collage (stick plasma)", True,
     "static/commendations/h5g/H5G_citation_Grenade_%C3%A0_plasma.png", "Arme", "Tuez un Spartan adverse à l'aide d'une grenade à plasma.", "2,4,6,10,20", "Grenade"),
    # ── GROUPE 3 : Multijoueur (10) ────────────────────────────────────────
    ("assassin", "Assassin", "medal", 548533137, None, None, None, None, None, None, "high", "Multijoueur — Médaille Par derrière", True,
     "static/commendations/h5g/H5G_citation_Assassin.png", "Multijoueur", "Assassinez des Spartans adverses.", "5,10,15,25,50", None),
    ("spartan_carnage", "Carnage de Spartans", "stat", None, None, "max_killing_spree", None, None, None, None, "high", "Multijoueur — max killing spree", True,
     "static/commendations/h5g/H5G_citation_Carnage_de_Spartans.png", "Multijoueur", "Tuez plusieurs Spartans adverses sans mourir.", "3,6,9,15,30", None),
    ("close_combat", "Combat rapproché", "stat", None, None, "melee_kills", None, None, None, None, "high", "Multijoueur — melee kills", True,
     "static/commendations/h5g/H5G_citation_Combat_rapproch%C3%A9.png", "Multijoueur", "Remportez n'importe quelle médaille de combat rapproché.", "10,20,30,50,100", None),
    ("opportunist", "Combattant opportuniste", "medal", None, "622331684,2063152177,4261842076,2137071619,1486797009,1430343434,2242633421", None, None, None, None, None, "high", "Multijoueur — Somme multi-kill medals", True,
     "static/commendations/h5g/H5G_citation_Combattant_opportuniste.png", "Multijoueur", "Remportez n'importe quelle médaille d'aptitude au combat.", "10,20,30,50,100", None),
    ("multikill", "Multifrag", "medal", 622331684, None, None, None, None, None, None, "high", "Multijoueur — Médaille Double frag", True,
     "static/commendations/h5g/H5G_citation_Multifrag.png", "Multijoueur", "Tuez rapidement plusieurs Spartans adverses.", "3,6,9,15,30", None),
    ("melee_fighter", "Pugilat", "stat", None, None, "melee_kills", None, None, None, None, "high", "Multijoueur — melee kills", True,
     "static/commendations/h5g/H5G_citation_Pugilat.png", "Multijoueur", "Tuez un Spartan ennemi d'une attaque rapprochée.", "5,10,15,25,50", None),
    ("headshot", "Tir à la tête", "stat", None, None, "headshot_kills", None, None, None, None, "high", "Multijoueur — headshot kills", True,
     "static/commendations/h5g/H5G_citation_Tir_%C3%A0_la_t%C3%AAte.png", "Multijoueur", "Tuez un Spartan ennemi d'un tir à la tête.", "10,20,30,50,100", None),
    ("spartan_killer", "Tueur de Spartans", "stat", None, None, "kills", None, None, None, None, "high", "Multijoueur — total kills", True,
     "static/commendations/h5g/H5G_citation_Tueur_de_Spartans.png", "Multijoueur", "Éliminez les Spartans ennemis.", "20,40,60,100,200", None),
    ("eagle_eye", "\u0152il de lynx", "medal", 1512363953, None, None, None, None, None, None, "high", "Multijoueur — Médaille Parfait", True,
     "static/commendations/h5g/H5G_citation_%C5%92il_de_lynx.png", "Multijoueur", "Tuez un Spartan adverse en pleine santé à l'aide d'une arme de précision sans manquer un seul coup.", "10,20,30,50,100", None),
    ("avenger", "Vengeur", "medal", 9000000001, None, None, None, None, None, None, "high", "Multijoueur — Médaille Vengeur", True,
     "static/commendations/h5g/Assassin_commendation.png", "Multijoueur", "Tuez l'ennemi responsable de votre mort précédente.", "5,15,30,55,105", None),
    # ── GROUPE 4 : Spartan Companies (16) ───────────────────────────────────
    ("flag_em_down", "Sors les drapeaux", "custom", None, None, None, None, None, "compute_flag_em_down", None, "high", "SC — Combine CTF awards", True,
     "static/commendations/h5g/H5G_citation_Flag_%27em_down.png", "Spartan Companies", "Obtenir une des médailles suivantes : Interception, Bataille de drapeaux, Frag du porteur, Retour du drapeau, Défense du drapeau", "1000,2000,3000,4800,9700", None),
    ("grand_theft", "Vol à la tire", "custom", None, None, None, None, None, "compute_hijack", None, "high", "SC — Combine HIJACKED_* awards", True,
     "static/commendations/h5g/H5G_citation_Grand_Theft.png", "Spartan Companies", "Aborder un véhicule ou un véhicule aérien", "200,400,600,960,1940", None),
    ("helping_hand", "Coup de main", "stat", None, None, "assists", None, None, None, None, "high", "SC — Total assists", True,
     "static/commendations/h5g/H5G_citation_Helping_Hand.png", "Spartan Companies", "Obtenir n'importe quelle assistance", "20000,40000,60000,96000,194400", None),
    ("im_just_perfect", "Zéro défaut", "medal", 1512363953, None, None, None, None, None, None, "high", "SC — Médaille Parfait", True,
     "static/commendations/h5g/H5G_citation_I%27m_just_perfect.png", "Spartan Companies", "Tuer un joueur avec une arme de précision sans manquer un tir", "2000,4000,6000,9600,19400", None),
    ("lawnmower", "Tondeuse", "medal", 221693153, None, None, None, None, None, None, "high", "SC — Médaille Écrasement", True,
     "static/commendations/h5g/H5G_citation_Lawnmower.png", "Spartan Companies", "Tuer un adversaire en l'écrasant avec un véhicule", "500,1000,1500,2400,4900", None),
    ("look_ma_no_pin", "Regarde maman, sans goupille", "stat", None, None, "grenade_kills", None, None, None, None, "high", "SC — grenade kills", True,
     "static/commendations/h5g/H5G_citation_Look_ma_no_pin.png", "Spartan Companies", "Tuer un Spartan ennemi avec n'importe quelle grenade", "4000,8000,12000,19200,38900", None),
    ("lucky", "Lucky", "medal", None, "3905838030,3091261182", None, None, None, None, None, "high", "SC — Médailles La chance + Chargeur vide", True,
     "static/commendations/h5g/H5G_citation_Lucky.png", "Spartan Companies", "Obtenir les médailles La chance, Chargeur vide, Sayonara ou Boulet de canon", "400,800,1200,1920,3880", None),
    ("no_hard_feelings", "Sans rancune", "stat", None, None, "kills", None, None, None, None, "high", "SC — total kills", True,
     "static/commendations/h5g/H5G_citation_No_Hard_Feelings.png", "Spartan Companies", "Tuer des Spartans ennemis", "50000,100000,150000,240000,486000", None),
    ("positive_contribution", "Positive contribution", "custom", None, None, None, None, None, "compute_bulldozer", None, "high", "SC — KD > 8 en Assassin", True,
     "static/commendations/h5g/H5G_citation_Positive_contribution.png", "Spartan Companies", "Finir avec FDA supérieur à 8 dans n'importe quelle partie Assassin en matchmaking", "300,600,900,1440,2960", None),
    ("power_play", "Coup de force", "stat", None, None, "power_weapon_kills", None, None, None, None, "high", "SC — power weapon kills", True,
     "static/commendations/h5g/H5G_citation_Power_play.png", "Spartan Companies", "Tuer un Spartan ennemi avec une arme puissante", "10000,20000,30000,48000,97200", None),
    ("road_trip", "Virée sur la route", "medal", 3169118333, None, None, None, None, None, None, "high", "SC — Médaille Violence routière", True,
     "static/commendations/h5g/H5G_citation_Road_Trip.png", "Spartan Companies", "Tuer un Spartan ennemi avec un véhicule terrestre", "3000,6000,9000,14400,29200", None),
    ("sting_like_a_bee", "Pique comme une abeille", "stat", None, None, "melee_kills", None, None, None, None, "high", "SC — melee kills", True,
     "static/commendations/h5g/H5G_citation_Sting_like_a_bee.png", "Spartan Companies", "Tuer un Spartan ennemi en combat rapproché", "5000,10000,15000,24000,48600", None),
    ("the_reaper", "Le faucheur", "medal", 2625820422, None, None, None, None, None, None, "high", "SC — Médaille Frag d'outre-tombe", True,
     "static/commendations/h5g/H5G_citation_The_Reaper.png", "Spartan Companies", "Tuer un adversaire d'outre-tombe", "500,1000,1500,2400,4850", None),
    ("too_fast_for_you", "Trop rapide pour toi", "medal", 2123530881, None, None, None, None, None, None, "high", "SC — Médaille Revirement", True,
     "static/commendations/h5g/H5G_citation_Too_fast_for_you.png", "Spartan Companies", "Tuer un adversaire qui vous a tiré dessus en premier", "2000,4000,6000,9600,19400", None),
    ("vandalism", "Vandalisme", "custom", None, None, None, None, None, "compute_vandalism", None, "high", "SC — Somme DESTROYED_* awards", True,
     "static/commendations/h5g/H5G_citation_Vandalism.png", "Spartan Companies", "Détruire un véhicule ennemi", "1200,2400,3600,5760,11640", None),
    # ── GROUPE 5 : Véhicule — UNSC (4) + Covenant (3) ─────────────────────
    ("wraith_destroyer", "Destructeur d'apparitions", "custom", None, None, None, None, None, "compute_wraith_destroyer", None, "high", "Véhicule — Wraith Covenant (Master: 30)", True,
     "static/commendations/h5g/H5G_citation_Destructeur_d%27apparitions.png", "Véhicule", "Détruisez les apparitions occupées par l'adversaire.", "3,6,9,15,30", "Covenant"),
    ("banshee_destroyer", "Destructeur de banshees", "award", None, None, None, "destroyed_banshee", "vehicle", None, None, "high", "Véhicule — Banshee Covenant (Master: 30)", True,
     "static/commendations/h5g/H5G_citation_Destructeur_de_banshees.png", "Véhicule", "Détruisez les banshees occupés par l'adversaire.", "3,6,9,15,30", "Covenant"),
    ("ghost_destroyer", "Destructeur de ghosts", "award", None, None, None, "destroyed_ghost", "vehicle", None, None, "high", "Véhicule — Ghost Covenant (Master: 50)", True,
     "static/commendations/h5g/H5G_citation_Destructeur_de_ghosts.png", "Véhicule", "Détruisez les ghosts occupés par l'adversaire.", "5,10,15,25,50", "Covenant"),
    ("mongoose_destroyer", "Destructeur de mongooses", "custom", None, None, None, None, None, "compute_mongoose_destroyer", None, "high", "Véhicule — Mongoose UNSC (Master: 50)", True,
     "static/commendations/h5g/H5G_citation_Destructeur_de_mongooses.png", "Véhicule", "Détruisez les mongooses occupées par l'adversaire.", "5,10,15,25,50", "UNSC"),
    ("scorpion_destroyer", "Destructeur de scorpions", "award", None, None, None, "destroyed_scorpion", "vehicle", None, None, "high", "Véhicule — Scorpion UNSC (Master: 10)", True,
     "static/commendations/h5g/H5G_citation_Destructeur_de_scorpions.png", "Véhicule", "Détruisez les scorpions occupés par l'adversaire.", "1,3,5,7,10", "UNSC"),
    ("warthog_destroyer", "Destructeur de warthogs", "custom", None, None, None, None, None, "compute_warthog_destroyer", None, "high", "Véhicule — Warthog UNSC (Master: 50)", True,
     "static/commendations/h5g/H5G_citation_Destructeur_de_warthogs.png", "Véhicule", "Détruisez les warthogs occupés par l'adversaire.", "5,10,15,25,50", "UNSC"),
    ("wasp_destroyer", "Destructeur de wasps", "award", None, None, None, "destroyed_wasp", "vehicle", None, None, "high", "Véhicule — Wasp UNSC (Master: 30)", True,
     "static/commendations/h5g/H5G_citation_Destructeur_de_wasps.png", "Véhicule", "Détruisez les wasps occupés par l'adversaire.", "3,6,9,15,30", "UNSC"),
]
# fmt: on

# ── Citations PVE unitaires ────────────────────────────────────────────────
# fmt: off
PVE_CITATIONS: list[tuple] = [
    ("grunt_slayer", "Tueur de Grognards", "pve_stat", None, None, "grunt_kills", None, None, None, None, "high", "Kills Grunt en Firefight", True,
     "static/commendations/h5g/H5G_citation_Tueur_de_Grognards.png", "Ennemi", "Tuez des Grognards.", "10,20,30,50,100", "Covenant"),
    ("elite_slayer", "Tueur d'Élites", "pve_stat", None, None, "elite_kills", None, None, None, None, "high", "Kills Élite en Firefight", True,
     "static/commendations/h5g/H5G_citation_Tueur_d%27%C3%89lites.png", "Ennemi", "Tuez des Élites.", "10,20,30,50,100", "Covenant"),
    ("jackal_slayer", "Tueur de Rapaces", "pve_stat", None, None, "jackal_kills", None, None, None, None, "high", "Kills Jackal en Firefight", True,
     "static/commendations/h5g/H5G_citation_Tueur_de_Rapaces.png", "Ennemi", "Tuez des Rapaces.", "5,10,15,25,50", "Covenant"),
    ("hunter_slayer", "Tueur de Chasseurs", "pve_stat", None, None, "hunter_kills", None, None, None, None, "high", "Kills Hunter en Firefight", True,
     "static/commendations/h5g/H5G_citation_Tueur_de_Chasseurs.png", "Ennemi", "Tuez des Chasseurs.", "2,4,6,10,20", "Covenant"),
    ("sentinel_slayer", "Tueur de sentinelles", "pve_stat", None, None, "sentinel_kills", None, None, None, None, "high", "Kills Sentinelle en Firefight — désactivé", False,
     "static/commendations/h5g/H5G_citation_Tueur_de_sentinelles.png", "Ennemi", "Tuez des sentinelles.", "5,10,15,25,50", "Banished"),
    ("like_a_boss", "Comme un Boss", "pve_stat", None, None, "boss_kills", None, None, None, None, "high", "Boss kills en Firefight", True,
     "static/commendations/h5g/H5G_citation_Like_a_boss.png", "Spartan Companies", "Tuer un boss dans une partie Baptême du feu zone de combat", "250,500,750,1200,2400", None),
    ("player_vs_everything", "Éliminations Firefight", "pve_stat", None, None, "total_enemy_kills", None, None, None, None, "high", "Total kills en Firefight", True,
     "static/commendations/h5g/H5G_citation_Player_vs_Everything.png", "Spartan Companies", "Gagner des parties en Baptême du feu", "200,400,600,960,1940", None),
    # Désactivées (pas d'image H5G / pertinence douteuse)
    ("brute_slayer", "Tueur de Brutes", "pve_stat", None, None, "brute_kills", None, None, None, None, "high", "Kills Brute en Firefight — pas d'image H5G", False,
     None, "Ennemi", "Tuez des Brutes.", None, "Banished"),
    ("skimmer_slayer", "Tueur de Skimmers", "pve_stat", None, None, "skimmer_kills", None, None, None, None, "high", "Kills Skimmer en Firefight — pas d'image H5G", False,
     None, "Ennemi", "Tuez des Skimmers.", None, "Banished"),
    ("marine_slayer", "Tueur de Marines", "pve_stat", None, None, "marine_kills", None, None, None, None, "high", "Kills Marine en Firefight — désactivé (alliés)", False,
     "static/commendations/h5g/H5G_citation_Tueur_de_r%C3%A9pliques_de_Marines.png", "Ennemi", "Tuez les Marines ennemis.", "20,40,60,100,200", "Banished"),
]
# fmt: on

# ── Citation composite : Destructeur de Covenants ──────────────────────────
_DESTRUCTEUR_CHILDREN = [
    "grunt_slayer",
    "elite_slayer",
    "jackal_slayer",
    "hunter_slayer",
    "like_a_boss",
    "brute_slayer",
    "skimmer_slayer",
]

# ── Citation composite : Maîtrise des grenades ────────────────────────────
_GRENADE_CHILDREN = [
    "frag_grenade",
    "plasma_grenade",
]

# ── Citation composite : Maîtrise de véhicule ─────────────────────────────
_VEHICLE_CHILDREN = [
    "splatter",
    "driver",
    "wraith_destroyer",
    "banshee_destroyer",
    "ghost_destroyer",
    "mongoose_destroyer",
    "scorpion_destroyer",
    "warthog_destroyer",
    "wasp_destroyer",
]

# ── Citations armes (v5.5 — noms canoniques v5.6 per-kill) ─────────────────
# stat_name = "weapon_kills:<weapon_name>" tel que stocké dans shared.weapon_kills
# Les variantes (alt) ne sont PAS fusionnées ici (à valider séparément).
# fmt: off
_WP = "static/commendations/h5g/"   # préfixe commun images H5G (conservé pour les armes gardées)
_WP_HI = "static/commendations/hi/"  # préfixe images Halo Infinite natives

WEAPON_CITATIONS: list[tuple] = [
    # UNSC
    ("br75_mastery", "Maîtrise du BR75", "weapon_stat", None, None, "weapon_kills:BR75", None, None, None, None, "high", "Kills avec BR75", True,
     _WP_HI + "HI_Commendations_BR75.png", "Arme", "Éliminez des Spartans avec le BR75.", "25,50,100,200,500", "UNSC"),
    ("ma40_mastery", "Maîtrise du MA40 AR", "weapon_stat", None, None, "weapon_kills:MA40 AR", None, None, None, None, "high", "Kills avec MA40 AR", True,
     _WP_HI + "HI_Commendations_MA40.png", "Arme", "Éliminez des Spartans avec le MA40 AR.", "25,50,100,200,500", "UNSC"),
    ("sidekick_mastery", "Maîtrise du MK50 Sidekick", "weapon_stat", None, None, "weapon_kills:Mk51 Sidekick", None, None, None, None, "high", "Kills avec MK50 Sidekick", True,
     _WP_HI + "HI_Commendations_Sidekick.png", "Arme", "Éliminez des Spartans avec le MK50 Sidekick.", "10,25,50,100,250", "UNSC"),
    ("commando_mastery", "Maîtrise du VK78 Commando", "weapon_stat", None, None, "weapon_kills:VK78 Commando", None, None, None, None, "high", "Kills avec VK78 Commando", True,
     _WP_HI + "HI_Commendations_Commando.png", "Arme", "Éliminez des Spartans avec le VK78 Commando.", "10,25,50,100,250", "UNSC"),
    ("sniper_mastery", "Maîtrise du S7 Sniper", "weapon_stat", None, None, "weapon_kills:S7 Sniper", None, None, None, None, "high", "Kills avec S7 Sniper", True,
     _WP_HI + "HI_Commendations_Sniper-S7.png", "Arme", "Éliminez des Spartans avec le S7 Sniper.", "10,20,40,80,200", "UNSC"),
    ("spnkr_mastery", "Maîtrise du M41 SPNKr", "weapon_stat", None, None, "weapon_kills:M41 SPNKr", None, None, None, None, "high", "Kills avec M41 SPNKr", True,
     _WP + "H5G_citation_SPNKR.png", "Arme", "Éliminez des Spartans avec le M41 SPNKr.", "10,20,40,80,200", "UNSC"),
    ("bulldog_mastery", "Maîtrise du CQS48 Bulldog", "weapon_stat", None, None, "weapon_kills:CQS48 Bulldog", None, None, None, None, "high", "Kills avec CQS48 Bulldog", True,
     _WP_HI + "HI_Commendations_Bulldog.png", "Arme", "Éliminez des Spartans avec le CQS48 Bulldog.", "10,25,50,100,250", "UNSC"),
    ("bandit_mastery", "Maîtrise du Bandit EVO", "weapon_stat", None, None, "weapon_kills:Bandit Evo", None, None, None, None, "high", "Kills avec Bandit EVO", True,
     _WP_HI + "HI_Commendations_Bandit.png", "Arme", "Éliminez des Spartans avec le Bandit EVO.", "10,25,50,100,250", "UNSC"),
    ("hydra_mastery", "Maîtrise du MLRS-2 Hydra", "weapon_stat", None, None, "weapon_kills:MLRS-2 Hydra", None, None, None, None, "high", "Kills avec MLRS-2 Hydra", True,
     _WP_HI + "HI_Commendations_Hydra.png", "Arme", "Éliminez des Spartans avec le MLRS-2 Hydra.", "5,10,20,40,100", "UNSC"),
    ("mutilator_mastery", "Maîtrise du Mutilateur", "weapon_stat", None, None, "weapon_kills:Mutilator", None, None, None, None, "high", "Kills avec Mutilateur", True,
     _WP_HI + "HI_Commendations_Mutilator.png", "Arme", "Éliminez des Spartans avec le Mutilateur.", "10,25,50,100,250", "UNSC"),
    # Paria (armes Covenant + Banished fusionnées)
    ("stalker_mastery", "Maîtrise du Fusil traqueur", "weapon_stat", None, None, "weapon_kills:Stalker Rifle", None, None, None, None, "high", "Kills avec Fusil traqueur", True,
     _WP_HI + "HI_Commendations_Stalker.png", "Arme", "Éliminez des Spartans avec le Fusil traqueur.", "10,25,50,100,250", "Paria"),
    ("needler_mastery", "Maîtrise du Needler", "weapon_stat", None, None, "weapon_kills:Needler", None, None, None, None, "high", "Kills avec Needler", True,
     _WP + "H5G_citation_Needler.png", "Arme", "Éliminez des Spartans avec le Needler.", "10,25,50,100,250", "Paria"),
    ("energy_sword_mastery", "Maîtrise de l'Épée à énergie", "weapon_stat", None, None, "weapon_kills:Energy Sword", None, None, None, None, "high", "Kills avec Épée à énergie", True,
     _WP + "H5G_citation_%C3%89p%C3%A9e_%C3%A0_%C3%A9nergie.png", "Arme", "Éliminez des Spartans avec l'Épée à énergie.", "10,20,40,80,200", "Paria"),
    ("mangler_mastery", "Maîtrise du Déchiqueteur", "weapon_stat", None, None, "weapon_kills:Mangler", None, None, None, None, "high", "Kills avec Déchiqueteur", True,
     _WP_HI + "HI_Commendations_Mangler.png", "Arme", "Éliminez des Spartans avec le Déchiqueteur.", "10,25,50,100,250", "Paria"),
    ("skewer_mastery", "Maîtrise de l'Empaleur", "weapon_stat", None, None, "weapon_kills:Skewer", None, None, None, None, "high", "Kills avec Empaleur", True,
     _WP_HI + "HI_Commendations_Skewer.png", "Arme", "Éliminez des Spartans avec l'Empaleur.", "5,10,20,40,100", "Paria"),
    ("gravity_hammer_mastery", "Maîtrise du Marteau antigravité", "weapon_stat", None, None, "weapon_kills:Gravity Hammer", None, None, None, None, "high", "Kills avec Marteau antigravité", True,
     _WP + "H5G_citation_Marteau_antigrav.png", "Arme", "Éliminez des Spartans avec le Marteau antigravité.", "10,20,40,80,200", "Paria"),
    ("pulse_carbine_mastery", "Maîtrise de la Carabine à impulsion", "weapon_stat", None, None, "weapon_kills:Pulse Carbine", None, None, None, None, "high", "Kills avec Carabine à impulsion", True,
     _WP_HI + "HI_Commendations_Carabine.png", "Arme", "Éliminez des Spartans avec la Carabine à impulsion.", "10,25,50,100,250", "Paria"),
    ("ravager_mastery", "Maîtrise du Ravageur", "weapon_stat", None, None, "weapon_kills:Ravager", None, None, None, None, "high", "Kills avec Ravageur", True,
     _WP_HI + "HI_Commendations_Ravager.png", "Arme", "Éliminez des Spartans avec le Ravageur.", "5,10,20,40,100", "Paria"),
    ("plasma_pistol_mastery", "Maîtrise du Pistolet à plasma", "weapon_stat", None, None, "weapon_kills:Plasma Pistol", None, None, None, None, "high", "Kills avec Pistolet à plasma", True,
     _WP_HI + "HI_Commendations_Plasma.png", "Arme", "Éliminez des Spartans avec le Pistolet à plasma.", "10,25,50,100,250", "Paria"),
    # Forerunner
    ("heatwave_mastery", "Maîtrise du Calcineur", "weapon_stat", None, None, "weapon_kills:Heatwave", None, None, None, None, "high", "Kills avec Calcineur", True,
     _WP_HI + "HI_Commendations_Heatwave.png", "Arme", "Éliminez des Spartans avec le Calcineur.", "10,25,50,100,250", "Forerunner"),
    ("cindershot_mastery", "Maîtrise du Crémateur", "weapon_stat", None, None, "weapon_kills:Cindershot", None, None, None, None, "high", "Kills avec Crémateur", True,
     _WP_HI + "HI_Commendations_Cremator.png", "Arme", "Éliminez des Spartans avec le Crémateur.", "10,20,40,80,200", "Forerunner"),
    ("sentinel_beam_mastery", "Maîtrise du Rayon de Sentinelle", "weapon_stat", None, None, "weapon_kills:Sentinel Beam", None, None, None, None, "high", "Kills avec Rayon de Sentinelle", True,
     _WP_HI + "HI_Commendations_Sentinel.png", "Arme", "Éliminez des Spartans avec le Rayon de Sentinelle.", "10,25,50,100,250", "Forerunner"),
    ("disruptor_mastery", "Maîtrise du Disrupteur", "weapon_stat", None, None, "weapon_kills:Disruptor", None, None, None, None, "high", "Kills avec Disrupteur", True,
     _WP_HI + "HI_Commendations_Disruptor.png", "Arme", "Éliminez des Spartans avec le Disrupteur.", "10,25,50,100,250", "Forerunner"),
    ("shock_rifle_mastery", "Maîtrise du Fusil électrique", "weapon_stat", None, None, "weapon_kills:Shock Rifle", None, None, None, None, "high", "Kills avec Fusil électrique", True,
     _WP_HI + "HI_Commendations_Shock.png", "Arme", "Éliminez des Spartans avec le Fusil électrique.", "10,25,50,100,250", "Forerunner"),
]
# fmt: on

# ── Composites faction armes (v5.5) ───────────────────────────────────────
_HUMAN_WEAPON_CHILDREN = [
    "br75_mastery",
    "ma40_mastery",
    "sidekick_mastery",
    "commando_mastery",
    "sniper_mastery",
    "spnkr_mastery",
    "bulldog_mastery",
    "bandit_mastery",
    "hydra_mastery",
    "mutilator_mastery",
]
_PARIA_WEAPON_CHILDREN = [
    "stalker_mastery",
    "needler_mastery",
    "energy_sword_mastery",
    "mangler_mastery",
    "skewer_mastery",
    "gravity_hammer_mastery",
    "pulse_carbine_mastery",
    "ravager_mastery",
    "plasma_pistol_mastery",
]
_FORERUNNER_WEAPON_CHILDREN = [
    "heatwave_mastery",
    "cindershot_mastery",
    "sentinel_beam_mastery",
    "disruptor_mastery",
    "shock_rifle_mastery",
]
_ALL_WEAPON_MASTERY_CHILDREN = [
    "human_weapons_mastery",
    "paria_weapons_mastery",
    "forerunner_weapons_mastery",
    "grenade_mastery",
]

# fmt: off
COMPOSITE_CITATIONS: list[tuple] = [
    ("covenant_destroyer", "Destructeur de Covenants", "composite", None, None, None, None, None, None,
     "[" + ", ".join(f'"{c}"' for c in _DESTRUCTEUR_CHILDREN) + "]",
     "high", "Obtenez toutes les citations d'élimination de Covenants", True,
     "static/commendations/h5g/H5G_ma\u00eetrise_Destructeur_de_Covenants.png", "Ennemi", "Obtenez toutes les citations d'élimination de Covenants", None, "Covenant"),
    ("grenade_mastery", "Maîtrise des grenades", "composite", None, None, None, None, None, None,
     "[" + ", ".join(f'"{c}"' for c in _GRENADE_CHILDREN) + "]",
     "high", "Obtenez toutes les citations de grenade", True,
     "static/commendations/h5g/H5G_Ma\u00eetrise_des_grenades.png", "Arme", "Obtenez toutes les citations de grenade.", None, "Grenade"),
    ("vehicle_mastery", "Maîtrise de véhicule", "composite", None, None, None, None, None, None,
     "[" + ", ".join(f'"{c}"' for c in _VEHICLE_CHILDREN) + "]",
     "high", "Obtenez toutes les citations de véhicule", True,
     "static/commendations/h5g/Vehicle_Mastery.png", "Véhicule", "Obtenez toutes les citations de véhicule.", None, "Général"),
    ("human_weapons_mastery", "Maîtrise des armes UNSC", "composite", None, None, None, None, None, None,
     "[" + ", ".join(f'"{c}"' for c in _HUMAN_WEAPON_CHILDREN) + "]",
     "high", "Obtenez toutes les citations d'armes UNSC", True,
     "static/commendations/h5g/H5G_Ma\u00eetrise_en_armes_UNSC.png", "Arme", "Obtenez toutes les citations de maîtrise d'armes UNSC.", None, "UNSC"),
    ("paria_weapons_mastery", "Maîtrise des armes Parias", "composite", None, None, None, None, None, None,
     "[" + ", ".join(f'"{c}"' for c in _PARIA_WEAPON_CHILDREN) + "]",
     "high", "Obtenez toutes les citations d'armes Parias", True,
     _WP_HI + "HI_Ma\u00eetrise_en_armes_lourdes_Parias.png", "Arme", "Obtenez toutes les citations de maîtrise d'armes Parias.", None, "Paria"),
    ("forerunner_weapons_mastery", "Maîtrise des armes Forerunner", "composite", None, None, None, None, None, None,
     "[" + ", ".join(f'"{c}"' for c in _FORERUNNER_WEAPON_CHILDREN) + "]",
     "high", "Obtenez toutes les citations d'armes Forerunner", True,
     "static/commendations/h5g/H5G_Maîtrise_en_armes_lourdes_forerunners.png", "Arme", "Obtenez toutes les citations de maîtrise d'armes Forerunner.", None, "Forerunner"),
    ("all_weapons_mastery", "Maîtrise en armement", "composite", None, None, None, None, None, None,
     "[" + ", ".join(f'"{c}"' for c in _ALL_WEAPON_MASTERY_CHILDREN) + "]",
     "high", "Obtenez toutes les citations d'armement", True,
     "static/commendations/h5g/H5G_Maîtrise_en_armement.png", "Arme", "Obtenez toutes les citations d'armement.", None, "Général"),
]
# fmt: on

ALL_CITATIONS = PVP_CITATIONS + PVE_CITATIONS + WEAPON_CITATIONS + COMPOSITE_CITATIONS

UPSERT_SQL = """
    INSERT INTO citation_mappings (
        citation_name_norm, citation_name_display, mapping_type,
        medal_id, medal_ids, stat_name, award_name, award_category,
        custom_function, composite_children, confidence, notes, enabled,
        image_path, category, description, tier_targets, subcategory
    ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
    ON CONFLICT (citation_name_norm)
    DO UPDATE SET
        citation_name_display = EXCLUDED.citation_name_display,
        mapping_type = EXCLUDED.mapping_type,
        medal_id = EXCLUDED.medal_id,
        medal_ids = EXCLUDED.medal_ids,
        stat_name = EXCLUDED.stat_name,
        award_name = EXCLUDED.award_name,
        award_category = EXCLUDED.award_category,
        custom_function = EXCLUDED.custom_function,
        composite_children = EXCLUDED.composite_children,
        confidence = EXCLUDED.confidence,
        notes = EXCLUDED.notes,
        enabled = EXCLUDED.enabled,
        image_path = EXCLUDED.image_path,
        category = EXCLUDED.category,
        description = EXCLUDED.description,
        tier_targets = EXCLUDED.tier_targets,
        subcategory = EXCLUDED.subcategory,
        updated_at = now()
"""


def ensure_schema(conn: duckdb.DuckDBPyConnection) -> None:
    """Crée la table et ajoute les colonnes manquantes (idempotent)."""
    conn.execute(CREATE_TABLE_DDL)

    cols = {
        row[0]
        for row in conn.execute(
            "SELECT column_name FROM information_schema.columns "
            "WHERE table_name = 'citation_mappings'"
        ).fetchall()
    }
    migrations = {
        "composite_children": "VARCHAR",
        "image_path": "VARCHAR",
        "category": "VARCHAR",
        "description": "VARCHAR",
        "tier_targets": "VARCHAR",
        "subcategory": "VARCHAR",
    }
    for col_name, col_type in migrations.items():
        if col_name not in cols:
            conn.execute(f"ALTER TABLE citation_mappings ADD COLUMN {col_name} {col_type}")
            print(f"  ✅ Colonne {col_name} ajoutée")


def populate(conn: duckdb.DuckDBPyConnection) -> int:
    """Insère/met à jour toutes les citations. Retourne le nombre total."""
    conn.executemany(UPSERT_SQL, ALL_CITATIONS)
    return len(ALL_CITATIONS)


def cleanup_obsolete(conn: duckdb.DuckDBPyConnection) -> int:
    """Supprime les citations qui ne font plus partie du référentiel."""
    known_norms = {c[0] for c in ALL_CITATIONS}
    existing = {
        row[0]
        for row in conn.execute("SELECT citation_name_norm FROM citation_mappings").fetchall()
    }
    obsolete = existing - known_norms
    if obsolete:
        placeholders = ", ".join(["?"] * len(obsolete))
        conn.execute(
            f"DELETE FROM citation_mappings WHERE citation_name_norm IN ({placeholders})",
            list(obsolete),
        )
    return len(obsolete)


def main() -> None:
    """Point d'entrée principal."""
    parser = argparse.ArgumentParser(description="Peuple citation_mappings dans metadata.duckdb")
    parser.add_argument(
        "--reset",
        action="store_true",
        help="DROP la table et la recrée from scratch",
    )
    args = parser.parse_args()

    if not METADATA_DB.exists():
        print(f"❌ metadata.duckdb introuvable : {METADATA_DB}")
        sys.exit(1)

    conn = duckdb.connect(str(METADATA_DB))
    try:
        if args.reset:
            conn.execute("DROP TABLE IF EXISTS citation_mappings")
            print("🗑️  Table citation_mappings supprimée")

        ensure_schema(conn)

        total = populate(conn)
        n_obsolete = cleanup_obsolete(conn)

        enabled = conn.execute(
            "SELECT COUNT(*) FROM citation_mappings WHERE enabled IS NOT FALSE"
        ).fetchone()[0]
        disabled = conn.execute(
            "SELECT COUNT(*) FROM citation_mappings WHERE enabled = FALSE"
        ).fetchone()[0]
        with_image = conn.execute(
            "SELECT COUNT(*) FROM citation_mappings WHERE image_path IS NOT NULL"
        ).fetchone()[0]
        with_tiers = conn.execute(
            "SELECT COUNT(*) FROM citation_mappings WHERE tier_targets IS NOT NULL"
        ).fetchone()[0]

        print(f"✅ {total} citations insérées/mises à jour")
        print(f"   → {enabled} activées, {disabled} désactivées")
        print(f"   → {with_image} avec image, {with_tiers} avec tiers")
        if n_obsolete:
            print(f"   → {n_obsolete} obsolètes supprimées")

        print("\n📋 Résumé par type :")
        for row in conn.execute(
            "SELECT mapping_type, COUNT(*), SUM(CASE WHEN enabled THEN 1 ELSE 0 END) "
            "FROM citation_mappings GROUP BY mapping_type ORDER BY mapping_type"
        ).fetchall():
            print(f"   {row[0]:12s} : {row[1]:2d} total, {row[2]:2d} activées")

        print("\n📋 Résumé par catégorie :")
        for row in conn.execute(
            "SELECT category, COUNT(*) "
            "FROM citation_mappings WHERE category IS NOT NULL "
            "GROUP BY category ORDER BY category"
        ).fetchall():
            print(f"   {row[0]:20s} : {row[1]:2d}")

    finally:
        conn.close()


if __name__ == "__main__":
    main()
