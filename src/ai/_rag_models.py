"""Modèles de données pour le système RAG.

Dataclasses : RAGConfig, Document, SearchResult.
"""

from __future__ import annotations

import hashlib
from dataclasses import dataclass, field
from datetime import datetime
from pathlib import Path
from typing import Any


@dataclass
class RAGConfig:
    """Configuration du système RAG."""

    persist_directory: str = "data/rag"
    table_name: str = "halo_knowledge"

    chunk_size: int = 1000
    chunk_overlap: int = 200

    top_k: int = 5

    default_sources: list[str] = field(default_factory=lambda: ["docs/", ".ai/", "src/"])

    include_patterns: list[str] = field(default_factory=lambda: ["*.md", "*.py", "*.json", "*.txt"])

    exclude_patterns: list[str] = field(
        default_factory=lambda: [
            "__pycache__",
            "*.pyc",
            ".git",
            "node_modules",
            "*.db",
            "*.parquet",
        ]
    )


@dataclass
class Document:
    """Un document indexé."""

    id: str
    content: str
    metadata: dict[str, Any]

    @classmethod
    def from_file(cls, path: Path, content: str) -> Document:
        """Crée un document à partir d'un fichier."""
        doc_id = hashlib.md5(f"{path}:{content[:100]}".encode()).hexdigest()
        return cls(
            id=doc_id,
            content=content,
            metadata={
                "source": str(path),
                "filename": path.name,
                "extension": path.suffix,
                "source_type": "file",
                "indexed_at": datetime.now().isoformat(),
            },
        )

    @classmethod
    def from_github(cls, url: str, path: str, content: str) -> Document:
        """Crée un document à partir de GitHub."""
        doc_id = hashlib.md5(f"{url}:{path}:{content[:100]}".encode()).hexdigest()
        return cls(
            id=doc_id,
            content=content,
            metadata={
                "source": f"{url}/{path}",
                "source_type": "github",
                "repo_url": url,
                "file_path": path,
                "indexed_at": datetime.now().isoformat(),
            },
        )


@dataclass
class SearchResult:
    """Résultat de recherche."""

    content: str
    source: str
    score: float
    metadata: dict[str, Any]
