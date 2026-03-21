"""Traductions FR des rangs Halo Infinite (career + CSR)."""

from __future__ import annotations

# ── Rangs career (Spartan Rank) ──────────────────────────────────────────────
CAREER_RANK_NAMES_FR: dict[str, str] = {
    "Recruit": "Recrue",
    "Cadet": "Cadet",
    "Private": "Soldat",
    "Lance Corporal": "Caporal suppléant",
    "Corporal": "Caporal",
    "Sergeant": "Sergent",
    "Staff Sergeant": "Sergent-chef",
    "Gunnery Sergeant": "Sergent d'artillerie",
    "Master Sergeant": "Sergent-major",
    "Lieutenant": "Lieutenant",
    "Captain": "Capitaine",
    "Major": "Commandant",
    "Lt Colonel": "Lieutenant-colonel",
    "Colonel": "Colonel",
    "Brigadier General": "Général de brigade",
    "General": "Général",
    "Hero": "Héros",
}

# ── Rangs CSR (compétitif) ───────────────────────────────────────────────────
CSR_TIER_NAMES_FR: dict[str, str] = {
    "Bronze": "Bronze",
    "Silver": "Argent",
    "Gold": "Or",
    "Platinum": "Platine",
    "Diamond": "Diamant",
    "Onyx": "Onyx",
}

# ── Combiné ──────────────────────────────────────────────────────────────────
RANK_NAMES_FR: dict[str, str] = {**CAREER_RANK_NAMES_FR, **CSR_TIER_NAMES_FR}


def translate_rank(en_name: str) -> str:
    """Traduit un nom de rang EN → FR. Retourne l'original si inconnu."""
    return RANK_NAMES_FR.get(en_name, en_name)
