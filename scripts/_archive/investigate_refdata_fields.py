#!/usr/bin/env python3
"""Investigation des champs refdata disponibles dans l'API.

Vérifie la présence de :
- GameVariantCategory dans MatchInfo
- PersonalScoreAwards dans CoreStats
- Breakdowns.PersonalScore dans CoreStats

Usage:
    python scripts/investigate_refdata_fields.py [match_id] [gamertag]

Note: Ce script nécessite les dépendances installées (pip install -r requirements.txt)
      notamment pandas pour les imports du projet.
"""

import asyncio
import json
import os
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).parent.parent))


def _load_dotenv():
    """Charge les variables d'environnement depuis .env.local ou .env"""
    repo_root = Path(__file__).resolve().parent.parent
    for name in (".env.local", ".env"):
        p = repo_root / name
        if not p.exists():
            continue
        content = p.read_text(encoding="utf-8")
        for line in content.splitlines():
            line = line.strip()
            if not line or line.startswith("#") or "=" not in line:
                continue
            k, v = line.split("=", 1)
            if os.environ.get(k.strip()) is None:
                os.environ[k.strip()] = v.strip().strip('"')


async def main():
    _load_dotenv()

    # Import avec gestion d'erreur claire
    try:
        from src.data.sync.api_client import SPNKrAPIClient, get_tokens_from_env
    except ImportError as e:
        print("=" * 80)
        print("ERREUR: Impossible d'importer les modules nécessaires")
        print("=" * 80)
        print(f"Details: {e}")
        print("\nSolution:")
        print("  1. Installez les dependances: pip install -r requirements.txt")
        print("  2. Ou utilisez l'environnement virtuel du projet")
        print("  3. Ou executez via streamlit: streamlit run streamlit_app.py")
        return

    # Arguments
    match_id = sys.argv[1] if len(sys.argv) > 1 else None
    gamertag = sys.argv[2] if len(sys.argv) > 2 else "SpartanC"

    tokens = await get_tokens_from_env()
    if not tokens:
        print("❌ Erreur : Tokens API non disponibles")
        print("   Configurez SPNKR_SPARTAN_TOKEN et SPNKR_CLEARANCE_TOKEN")
        return

    async with SPNKrAPIClient(tokens=tokens) as client:
        # Si pas de match_id, récupérer le plus récent
        if not match_id:
            print(f"📥 Récupération de l'historique pour {gamertag}...")
            history = await client.get_match_history(gamertag, count=1)
            if not history:
                print("❌ Aucun match trouvé")
                return
            match_id = history[0].match_id
            print(f"✅ Match trouvé : {match_id}")

        print(f"\n🔍 Analyse du match {match_id}...\n")

        # Récupérer les stats
        stats_json = await client.get_match_stats(match_id)
        if not stats_json:
            print("❌ Impossible de récupérer les stats")
            return

        # Sauvegarder le JSON complet pour inspection
        output_dir = Path(__file__).parent.parent / "data" / "investigation"
        output_dir.mkdir(parents=True, exist_ok=True)
        output_file = output_dir / f"refdata_investigation_{match_id}.json"
        with open(output_file, "w", encoding="utf-8") as f:
            json.dump(stats_json, f, indent=2, ensure_ascii=False)
        print(f"💾 JSON complet sauvegardé : {output_file}\n")

        # Analyser MatchInfo
        print("=" * 80)
        print("📋 ANALYSE MatchInfo")
        print("=" * 80)
        match_info = stats_json.get("MatchInfo", {})
        if not match_info:
            print("❌ MatchInfo non trouvé")
        else:
            print(f"✅ MatchInfo trouvé ({len(match_info)} clés)")
            print("\nClés disponibles dans MatchInfo:")
            for key in sorted(match_info.keys()):
                value = match_info[key]
                if isinstance(value, dict):
                    print(f"  📦 {key}: dict avec {len(value)} clés")
                elif isinstance(value, list):
                    print(f"  📋 {key}: list[{len(value)}]")
                else:
                    print(f"  🔹 {key}: {type(value).__name__} = {str(value)[:60]}")

            # Chercher GameVariantCategory
            print("\n🔍 Recherche de GameVariantCategory:")
            if "GameVariantCategory" in match_info:
                category = match_info["GameVariantCategory"]
                print(f"  ✅ TROUVÉ : {category} (type: {type(category).__name__})")
            else:
                print("  ❌ NON TROUVÉ directement dans MatchInfo")
                # Chercher dans les assets
                print("\n  🔍 Recherche dans les assets:")
                for asset_key in ["UgcGameVariant", "GameVariant", "Playlist"]:
                    asset = match_info.get(asset_key)
                    if isinstance(asset, dict):
                        print(f"    📦 {asset_key}:")
                        for k, v in asset.items():
                            print(f"      - {k}: {type(v).__name__} = {str(v)[:50]}")

        # Analyser Players[].CoreStats
        print("\n" + "=" * 80)
        print("📊 ANALYSE CoreStats (premier joueur)")
        print("=" * 80)
        players = stats_json.get("Players", [])
        if not players:
            print("❌ Aucun joueur trouvé")
        else:
            print(f"✅ {len(players)} joueurs trouvés")
            first_player = players[0]
            core_stats = None

            # Chercher CoreStats
            if "CoreStats" in first_player:
                core_stats = first_player["CoreStats"]
            elif "Stats" in first_player and isinstance(first_player["Stats"], dict):
                core_stats = first_player["Stats"].get("CoreStats")

            if not core_stats:
                print("❌ CoreStats non trouvé")
            else:
                print(f"✅ CoreStats trouvé ({len(core_stats)} clés)")
                print("\nClés disponibles dans CoreStats:")
                for key in sorted(core_stats.keys()):
                    value = core_stats[key]
                    if isinstance(value, dict):
                        print(f"  📦 {key}: dict avec {len(value)} clés")
                        # Afficher les sous-clés
                        if key == "Breakdowns":
                            print("    Sous-clés Breakdowns:")
                            for sub_key in value:
                                print(f"      - {sub_key}")
                    elif isinstance(value, list):
                        print(f"  📋 {key}: list[{len(value)}]")
                        # Afficher le premier élément si pertinent
                        if value and isinstance(value[0], dict):
                            print(f"    Premier élément: {list(value[0].keys())[:5]}")
                    else:
                        val_str = str(value)
                        if len(val_str) > 60:
                            val_str = val_str[:60] + "..."
                        print(f"  🔹 {key}: {type(value).__name__} = {val_str}")

                # Chercher PersonalScoreAwards
                print("\n🔍 Recherche de PersonalScoreAwards:")
                if "PersonalScoreAwards" in core_stats:
                    awards = core_stats["PersonalScoreAwards"]
                    print(f"  ✅ TROUVÉ : {type(awards).__name__}")
                    if isinstance(awards, list):
                        print(f"    Nombre d'awards : {len(awards)}")
                        if awards:
                            print(f"    Premier award : {awards[0]}")
                    elif isinstance(awards, dict):
                        print(f"    Clés : {list(awards.keys())}")
                else:
                    print("  ❌ NON TROUVÉ directement dans CoreStats")

                # Chercher Breakdowns.PersonalScore
                print("\n🔍 Recherche de Breakdowns.PersonalScore:")
                breakdowns = core_stats.get("Breakdowns")
                if isinstance(breakdowns, dict):
                    if "PersonalScore" in breakdowns:
                        ps_breakdown = breakdowns["PersonalScore"]
                        print(f"  ✅ TROUVÉ dans Breakdowns : {type(ps_breakdown).__name__}")
                        if isinstance(ps_breakdown, list):
                            print(f"    Nombre d'éléments : {len(ps_breakdown)}")
                            if ps_breakdown:
                                print(f"    Premier élément : {ps_breakdown[0]}")
                        elif isinstance(ps_breakdown, dict):
                            print(f"    Clés : {list(ps_breakdown)}")
                    else:
                        print("  ❌ NON TROUVÉ dans Breakdowns")
                        print(f"    Clés disponibles dans Breakdowns : {list(breakdowns.keys())}")
                else:
                    print("  ❌ Breakdowns non trouvé ou pas un dict")

                # Vérifier PersonalScore total
                print("\n🔍 Vérification PersonalScore total:")
                if "PersonalScore" in core_stats:
                    ps_total = core_stats["PersonalScore"]
                    print(f"  ✅ PersonalScore total : {ps_total}")
                else:
                    print("  ❌ PersonalScore total non trouvé")

        # Résumé
        print("\n" + "=" * 80)
        print("📝 RÉSUMÉ")
        print("=" * 80)
        print("\n✅ Disponible :")
        print("  - MatchInfo (structure complète)")
        print("  - CoreStats (structure complète)")
        print("  - PersonalScore (total)")

        print("\n❓ À vérifier :")
        print("  - GameVariantCategory dans MatchInfo")
        print("  - PersonalScoreAwards dans CoreStats")
        print("  - Breakdowns.PersonalScore dans CoreStats")

        print(f"\n💾 JSON complet sauvegardé dans : {output_file}")
        print("   Inspectez ce fichier pour plus de détails.")


if __name__ == "__main__":
    asyncio.run(main())
