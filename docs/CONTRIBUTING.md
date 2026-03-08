# Contributing to LevelUp

French version: [FR/CONTRIBUTING.md](FR/CONTRIBUTING.md)

Thank you for your interest in contributing to LevelUp! This document explains how to participate in the project.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [How to Contribute](#how-to-contribute)
- [Environment Setup](#environment-setup)
- [Code Standards](#code-standards)
- [Pull Request Process](#pull-request-process)
- [Reporting a Bug](#reporting-a-bug)
- [Proposing a Feature](#proposing-a-feature)
- [Open Source Credits](#open-source-credits)

---

## Code of Conduct

This project follows a respectful and inclusive code of conduct. Be kind to other contributors.

---

## How to Contribute

### 1. Fork the Project

```bash
# Fork via GitHub then clone
git clone https://github.com/your-username/levelup-halo.git
cd levelup-halo
```

### 2. Create a Branch

```bash
git checkout -b feature/my-feature
# or
git checkout -b fix/my-bug
```

### 3. Develop

Make your changes following the code standards.

### 4. Test

```bash
pytest
```

### 5. Commit

```bash
git add .
git commit -m "feat(scope): description"
```

### 6. Push and Pull Request

```bash
git push origin feature/my-feature
```

Create a Pull Request on GitHub.

---

## Environment Setup

### Prerequisites

- **Python 3.10 to 3.12** (python.org or Windows Store). **Do not use MSYS2 Python** (Git Bash): DuckDB has no wheels for MINGW, and pip would attempt a source build that fails.
- Git

### Installation (Git Bash)

```bash
bash scripts/setup_env.sh
source scripts/activate_env.sh
```

The script uses `py` (Python Launcher for Windows) to create a venv with the Windows Python, in order to get pre-built DuckDB wheels.

### Development Tools

The following tools are included in `[dev]`:

| Tool | Usage |
|------|-------|
| `pytest` | Unit tests |
| `black` | Code formatting |
| `isort` | Import sorting |
| `ruff` | Linting |
| `mypy` | Type checking |

---

## Code Standards

### Formatting

Before each commit:

```bash
# Formatting
black .
isort .

# Linting
ruff check --fix .
```

### Type Hints

All public functions must have type hints:

```python
def compute_kd_ratio(kills: int, deaths: int) -> float:
    """Calcule le ratio kills/deaths."""
    if deaths == 0:
        return float(kills)
    return kills / deaths
```

### Docstrings

Use docstrings in French:

```python
def load_matches(self, limit: int = 100) -> pl.DataFrame:
    """
    Charge les matchs depuis la base de données.
    
    Args:
        limit: Nombre maximum de matchs à charger.
        
    Returns:
        DataFrame Polars avec les statistiques des matchs.
    """
```

### Data Access

**ALWAYS** use `DuckDBRepository`:

```python
from src.data.repositories import DuckDBRepository

repo = DuckDBRepository(db_path, xuid)
matches = repo.load_matches()
```

### Data Backfill

**ALWAYS** use `scripts/backfill_data.py` for backfilling or creating new backfill functions. Do not create separate backfill scripts; add a dedicated option in `backfill_data.py` (e.g. `--sessions`, `--killer-victim`). See the script's docstring for the pattern to follow.

### Performance Benchmarking

Use `scripts/benchmark_pages.py` to measure data loading times:

```bash
# Create a baseline before a change
python scripts/benchmark_pages.py --baseline --output .ai/reports/benchmark_baseline.json

# Run a standard benchmark (5 iterations)
python scripts/benchmark_pages.py --runs 5

# Compare against an existing baseline
python scripts/benchmark_pages.py --compare .ai/reports/benchmark_baseline.json
```

The script automatically measures: cold/warm load, medals, teammates, Polars filtering, Pandas conversion.
Variability (CV) should remain < 10% for reliable results.

---

## Pull Request Process

### Checklist

Before submitting a PR, verify:

- [ ] Tests pass (`pytest`)
- [ ] Code is formatted (`black`, `isort`)
- [ ] No linting errors (`ruff check`)
- [ ] New tests cover the changes
- [ ] Documentation is updated if necessary
- [ ] Commit message follows the Conventional Commits format

### Commit Format

```
<type>(<scope>): <description>

[optional body]
```

Types:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation
- `refactor`: Refactoring
- `test`: Adding tests
- `chore`: Maintenance

Examples:
```
feat(ui): add radar chart for performance stats
fix(sync): fix Firefight mode parsing
docs: update installation guide
```

### Review

A maintainer will get back to you for:
- Clarification questions
- Improvement suggestions
- Validation and merge

---

## Reporting a Bug

### Before Reporting

1. Check that the bug has not already been reported
2. Test with the latest version

### Create an Issue

Include:
- **Description**: Observed vs. expected behaviour
- **Reproduction**: Steps to reproduce
- **Environment**: OS, Python version
- **Logs**: Full error messages

```markdown
## Bug

### Description
The dashboard does not load matches for player X.

### Reproduction
1. Open the dashboard
2. Select player X
3. Observe the error

### Environment
- OS: Windows 11
- Python: 3.11.4
- Version: 3.0.0

### Logs
```
Error: DuckDB file not found...
```
```

---

## Open Source Credits

This project relies on several community components. Credits are centralised in [ACKNOWLEDGMENTS.md](ACKNOWLEDGMENTS.md).

Before adding a major external dependency, document it in that file as well to keep attribution clear.

---

## Proposing a Feature

### Before Proposing

1. Check that the feature has not already been proposed or is in progress
2. Think through the implementation

### Create an Issue

Include:
- **Description**: What does the feature do?
- **Motivation**: Why is it useful?
- **Implementation**: How to implement it (optional)

```markdown
## Feature Request

### Description
Add a CSV export of statistics.

### Motivation
Allow users to analyse their stats in Excel.

### Suggested Implementation
- Add an "Export CSV" button on the History page
- Use Polars for the conversion
```

---

## Questions?

If you have questions, open an issue with the `question` tag.

---

**Thank you for contributing to LevelUp!**
