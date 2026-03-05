# src/ai/rag.py
"""
RAG (Retrieval-Augmented Generation) pour OpenSpartan Graph.

Façade re-exportant les classes depuis les sous-modules :
- _rag_models : RAGConfig, Document, SearchResult
- _rag_chunker : TextChunker
- _rag_github : GitHubIndexer

Conserve HaloKnowledgeBase et create_knowledge_base localement.

Usage:
    from src.ai.rag import HaloKnowledgeBase, RAGConfig

    config = RAGConfig(persist_directory="data/rag")
    kb = HaloKnowledgeBase(config)
    kb.index_directory("docs/")
    results = kb.search("Comment fonctionne l'authentification?")
"""

from __future__ import annotations

import hashlib
import json
import logging
from datetime import datetime
from pathlib import Path
from typing import Any

# ── Re-exports depuis sous-modules ──────────────────────────────────────────
from src.ai._rag_chunker import TextChunker  # noqa: F401
from src.ai._rag_github import GitHubIndexer  # noqa: F401
from src.ai._rag_models import (  # noqa: F401
    Document,
    RAGConfig,
    SearchResult,
)

logger = logging.getLogger(__name__)

try:
    import lancedb

    LANCEDB_AVAILABLE = True
except ImportError:
    LANCEDB_AVAILABLE = False
    lancedb = None  # type: ignore


class HaloKnowledgeBase:
    """Base de connaissances Halo avec RAG (LanceDB)."""

    def __init__(self, config: RAGConfig | None = None):
        """Initialise la base de connaissances."""
        if not LANCEDB_AVAILABLE:
            raise ImportError("LanceDB n'est pas installé. Installez-le avec: pip install lancedb")

        self.config = config or RAGConfig()
        self.chunker = TextChunker(
            chunk_size=self.config.chunk_size, overlap=self.config.chunk_overlap
        )

        persist_path = Path(self.config.persist_directory)
        persist_path.mkdir(parents=True, exist_ok=True)
        self.db = lancedb.connect(str(persist_path))
        self._init_table()
        self._indexed_count = 0

    def _init_table(self) -> None:
        """Initialise la table LanceDB."""
        table_name = self.config.table_name
        if table_name in self.db.list_tables():
            self.table = self.db.open_table(table_name)
        else:
            self.table = None

    def _ensure_table(self, data: list[dict]) -> None:
        """S'assure que la table existe et y ajoute les données."""
        table_name = self.config.table_name
        if self.table is None:
            self.table = self.db.create_table(table_name, data=data, mode="overwrite")
        else:
            self.table.add(data)

    @property
    def document_count(self) -> int:
        """Nombre de documents indexés."""
        if self.table is None:
            return 0
        return self.table.count_rows()

    def index_file(self, path: Path | str) -> int:
        """Indexe un fichier. Retourne le nombre de chunks indexés."""
        path = Path(path)
        if not path.exists():
            raise FileNotFoundError(f"Fichier non trouvé: {path}")

        content = path.read_text(encoding="utf-8", errors="ignore")
        if path.suffix == ".py":
            chunks = self.chunker.chunk_code(content, "python")
        else:
            chunks = self.chunker.chunk_text(content)

        records = []
        for i, chunk in enumerate(chunks):
            doc = Document.from_file(path, chunk)
            records.append(
                {
                    "id": f"{doc.id}_{i}",
                    "text": chunk,
                    "source": str(path),
                    "source_type": "file",
                    "file_path": str(path),
                    "chunk_index": i,
                    "total_chunks": len(chunks),
                    "indexed_at": datetime.now().isoformat(),
                }
            )

        if records:
            self._ensure_table(records)

        self._indexed_count += len(chunks)
        return len(chunks)

    def index_directory(self, directory: Path | str, recursive: bool = True) -> dict[str, int]:
        """Indexe un répertoire. Retourne {fichier: nb_chunks}."""
        directory = Path(directory)
        results: dict[str, int] = {}
        if not directory.exists():
            raise FileNotFoundError(f"Répertoire non trouvé: {directory}")

        for pattern in self.config.include_patterns:
            files = directory.rglob(pattern) if recursive else directory.glob(pattern)
            for file_path in files:
                if any(excl in str(file_path) for excl in self.config.exclude_patterns):
                    continue
                try:
                    chunks = self.index_file(file_path)
                    results[str(file_path)] = chunks
                except Exception as e:
                    logger.warning("Erreur indexation %s: %s", file_path, e)

        return results

    def index_github_repo(self, repo_url: str, branch: str = "master") -> dict[str, int]:
        """Indexe un repository GitHub. Retourne {fichier: nb_chunks}."""
        indexer = GitHubIndexer(repo_url)
        documents = indexer.fetch_all_documents(branch)
        results: dict[str, int] = {}

        for doc in documents:
            ext = Path(doc.metadata.get("file_path", "")).suffix
            if ext == ".py":
                chunks = self.chunker.chunk_code(doc.content, "python")
            else:
                chunks = self.chunker.chunk_text(doc.content)

            records = []
            for i, chunk in enumerate(chunks):
                records.append(
                    {
                        "id": f"{doc.id}_{i}",
                        "text": chunk,
                        "source": doc.metadata.get("source", ""),
                        "source_type": "github",
                        "file_path": doc.metadata.get("file_path", ""),
                        "chunk_index": i,
                        "total_chunks": len(chunks),
                        "indexed_at": datetime.now().isoformat(),
                    }
                )

            if records:
                self._ensure_table(records)
            results[doc.metadata.get("file_path", doc.id)] = len(chunks)

        return results

    def index_text(
        self,
        text: str,
        source: str = "manual",
        metadata: dict[str, Any] | None = None,  # noqa: ARG002
    ) -> int:
        """Indexe du texte brut. Retourne le nombre de chunks."""
        chunks = self.chunker.chunk_text(text)
        records = []
        for i, chunk in enumerate(chunks):
            chunk_id = hashlib.md5(f"{source}:{i}:{chunk[:50]}".encode()).hexdigest()
            records.append(
                {
                    "id": chunk_id,
                    "text": chunk,
                    "source": source,
                    "source_type": "text",
                    "file_path": source,
                    "chunk_index": i,
                    "total_chunks": len(chunks),
                    "indexed_at": datetime.now().isoformat(),
                }
            )
        if records:
            self._ensure_table(records)
        return len(chunks)

    def search(  # noqa: C901
        self, query: str, top_k: int | None = None, source_type: str | None = None
    ) -> list[SearchResult]:
        """Recherche textuelle dans la base de connaissances."""
        if self.table is None:
            return []

        k = top_k or self.config.top_k
        all_data = self.table.to_pandas()

        if source_type and "source_type" in all_data.columns:
            all_data = all_data[all_data["source_type"] == source_type]

        query_lower = query.lower()
        query_terms = query_lower.split()
        scores: list[tuple[int, float, Any]] = []

        for idx, row in all_data.iterrows():
            text = str(row.get("text", "")).lower()
            term_matches = sum(1 for term in query_terms if term in text)
            exact_match = 1.0 if query_lower in text else 0.0
            score = (term_matches / max(len(query_terms), 1)) * 0.7 + exact_match * 0.3

            if score > 0:
                scores.append((idx, score, row))

        scores.sort(key=lambda x: x[1], reverse=True)

        return [
            SearchResult(
                content=row.get("text", ""),
                source=row.get("source", "unknown"),
                score=score,
                metadata={
                    "source_type": row.get("source_type", ""),
                    "chunk_index": row.get("chunk_index", 0),
                    "indexed_at": row.get("indexed_at", ""),
                },
            )
            for _idx, score, row in scores[:k]
        ]

    def search_by_source(
        self, query: str, source_type: str, top_k: int | None = None
    ) -> list[SearchResult]:
        """Recherche filtrée par type de source."""
        return self.search(query, top_k=top_k, source_type=source_type)

    def get_context_for_prompt(
        self, query: str, max_tokens: int = 4000, include_sources: bool = True
    ) -> str:
        """Génère un contexte formaté pour un prompt LLM."""
        results = self.search(query, top_k=10)
        context_parts: list[str] = []
        current_length = 0
        char_per_token = 4

        for result in results:
            chunk_length = len(result.content)
            if current_length + chunk_length > max_tokens * char_per_token:
                break
            if include_sources:
                context_parts.append(f"\n[Source: {result.source}]\n{result.content}")
            else:
                context_parts.append(result.content)
            current_length += chunk_length

        return "\n\n---\n\n".join(context_parts)

    def clear(self) -> None:
        """Supprime tous les documents de la table."""
        table_name = self.config.table_name
        if table_name in self.db.list_tables():
            self.db.drop_table(table_name)
        self.table = None
        self._indexed_count = 0

    def get_stats(self) -> dict[str, Any]:
        """Retourne les statistiques de la base."""
        return {
            "total_documents": self.document_count,
            "table_name": self.config.table_name,
            "persist_directory": self.config.persist_directory,
            "chunk_size": self.config.chunk_size,
            "backend": "lancedb",
        }

    def export_index_manifest(self, path: Path | str) -> None:
        """Exporte un manifest des documents indexés."""
        path = Path(path)

        if self.table is None:
            manifest = {
                "total_chunks": 0,
                "sources": {},
                "config": {
                    "chunk_size": self.config.chunk_size,
                    "chunk_overlap": self.config.chunk_overlap,
                },
                "exported_at": datetime.now().isoformat(),
            }
        else:
            all_data = self.table.to_pandas()
            sources: dict[str, dict[str, Any]] = {}
            for _, row in all_data.iterrows():
                source = row.get("source", "unknown")
                if source not in sources:
                    sources[source] = {
                        "count": 0,
                        "source_type": row.get("source_type", "unknown"),
                    }
                sources[source]["count"] += 1

            manifest = {
                "total_chunks": self.document_count,
                "sources": sources,
                "config": {
                    "chunk_size": self.config.chunk_size,
                    "chunk_overlap": self.config.chunk_overlap,
                },
                "exported_at": datetime.now().isoformat(),
            }

        path.write_text(json.dumps(manifest, indent=2, ensure_ascii=False))


def create_knowledge_base(
    persist_dir: str = "data/rag", index_defaults: bool = True
) -> HaloKnowledgeBase:
    """Crée une base de connaissances avec les sources par défaut."""
    config = RAGConfig(persist_directory=persist_dir)
    kb = HaloKnowledgeBase(config)

    if index_defaults:
        for source in config.default_sources:
            if Path(source).exists():
                try:
                    kb.index_directory(source)
                except Exception as e:
                    logger.warning("Erreur indexation %s: %s", source, e)

    return kb


__all__ = [
    "Document",
    "GitHubIndexer",
    "HaloKnowledgeBase",
    "RAGConfig",
    "SearchResult",
    "TextChunker",
    "create_knowledge_base",
]
