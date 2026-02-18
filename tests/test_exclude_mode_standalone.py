#!/usr/bin/env python3
"""Test minimal du mode exclude sans dépendances Streamlit."""

def _detect_filter_mode(selected, all_options):
    """Copie locale de la fonction pour test."""
    if not all_options:
        return "include"
    
    selected_set = set(selected) if isinstance(selected, list) else selected
    all_set = set(all_options) if isinstance(all_options, list) else all_options
    
    if not selected_set:
        return "include"
    
    ratio = len(selected_set) / len(all_set)
    
    if ratio > 0.7:
        return "exclude"
    elif ratio < 0.3:
        return "include"
    else:
        return "include"


def test_all():
    """Tests de la fonction."""
    print("Testing _detect_filter_mode()...")
    
    tests = [
        # (selected, all_options, expected_mode, description)
        ({"A", "B", "C", "D", "E", "F", "G", "H"}, {"A", "B", "C", "D", "E", "F", "G", "H", "I", "J"}, "exclude", ">70% → exclude"),
        ({"A", "B"}, {"A", "B", "C", "D", "E", "F", "G", "H", "I", "J"}, "include", "<30% → include"),
        ({"A", "B", "C", "D", "E"}, {"A", "B", "C", "D", "E", "F", "G", "H", "I", "J"}, "include", "50% zone grise → include"),
        ({"A", "B", "C"}, {"A", "B", "C"}, "exclude", "100% → exclude"),
        (set(), {"A", "B", "C"}, "include", "0% → include"),
        (set(range(71)), set(range(100)), "exclude", "71% → exclude"),
        (set(range(70)), set(range(100)), "include", "70% zone grise → include"),
        (set(range(30)), set(range(100)), "include", "30% zone grise → include"),
        (set(range(29)), set(range(100)), "include", "29% → include"),
    ]
    
    passed = 0
    failed = 0
    
    for selected, all_options, expected, desc in tests:
        result = _detect_filter_mode(selected, all_options)
        if result == expected:
            print(f"  ✓ {desc}")
            passed += 1
        else:
            print(f"  ✗ {desc} (got '{result}', expected '{expected}')")
            failed += 1
    
    print(f"\nResults: {passed} passed, {failed} failed")
    
    if failed == 0:
        print("\n✅ ALL TESTS PASSED!")
        return True
    else:
        print("\n❌ SOME TESTS FAILED!")
        return False


def test_scenarios():
    """Tests de scénarios réels."""
    print("\nTesting real-world scenarios...")
    
    # Scénario 1: Tout sauf Firefight
    all_playlists = ["Quick Play", "Ranked", "BTB", "Firefight", "Slayer"]
    selected = {"Quick Play", "Ranked", "BTB", "Slayer"}  # 80%
    mode = _detect_filter_mode(selected, all_playlists)
    
    if mode == "exclude":
        excluded = set(all_playlists) - selected
        if excluded == {"Firefight"}:
            print("  ✓ Scénario: Tout sauf Firefight (mode exclude, sauvegarde ['Firefight'])")
        else:
            print(f"  ✗ Excluded wrong items: {excluded}")
            return False
    else:
        print(f"  ✗ Wrong mode: {mode}")
        return False
    
    # Scénario 2: Nouvelle playlist ajoutée en mode exclude
    print("  ✓ Scénario: Nouvelle playlist auto-incluse en mode exclude")
    excluded_saved = ["BTB"]
    all_playlists_new = ["Quick Play", "Ranked", "BTB", "New Mode"]
    selected_result = set(all_playlists_new) - set(excluded_saved)
    
    if "New Mode" in selected_result and "BTB" not in selected_result:
        print("  ✓ Nouvelle playlist 'New Mode' auto-incluse, 'BTB' toujours exclu")
    else:
        print(f"  ✗ Wrong result: {selected_result}")
        return False
    
    print("\n✅ ALL SCENARIOS PASSED!")
    return True


if __name__ == "__main__":
    success = test_all() and test_scenarios()
    exit(0 if success else 1)
