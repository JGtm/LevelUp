"""Indexeur de repositories GitHub pour le RAG."""

from __future__ import annotations

import re
from pathlib import Path

from src.ai._rag_models import Document

try:
    import httpx

    HTTPX_AVAILABLE = True
except ImportError:  # pragma: no cover
    HTTPX_AVAILABLE = False
    httpx = None  # type: ignore


class GitHubIndexer:
    """Indexe un repository GitHub."""

    INCLUDE_EXTENSIONS = {".md", ".py", ".ts", ".js", ".json", ".cs", ".txt"}

    EXCLUDE_PATTERNS = {
        "node_modules",
        "__pycache__",
        ".git",
        "dist",
        "build",
        "package-lock.json",
        "yarn.lock",
        ".env",
    }

    def __init__(self, repo_url: str):
        """Initialise l'indexeur avec l'URL du repo."""
        self.repo_url = repo_url.rstrip("/")
        self._parse_repo_info()

    def _parse_repo_info(self) -> None:
        """Parse owner/repo depuis l'URL."""
        match = re.match(r"https://github\.com/([^/]+)/([^/]+)", self.repo_url)
        if not match:
            raise ValueError(f"URL GitHub invalide: {self.repo_url}")
        self.owner = match.group(1)
        self.repo = match.group(2)
        self.api_base = f"https://api.github.com/repos/{self.owner}/{self.repo}"

    def fetch_tree(self, branch: str = "master") -> list[dict]:
        """Récupère l'arbre des fichiers du repo."""
        if not HTTPX_AVAILABLE:
            raise ModuleNotFoundError(
                "Le module 'httpx' est requis pour indexer un repo GitHub. "
                "Installez-le avec: pip install httpx"
            )
        url = f"{self.api_base}/git/trees/{branch}?recursive=1"

        with httpx.Client() as client:
            resp = client.get(url, headers={"Accept": "application/vnd.github.v3+json"})
            resp.raise_for_status()
            data = resp.json()

        return [
            item
            for item in data.get("tree", [])
            if item["type"] == "blob" and self._should_include(item["path"])
        ]

    def _should_include(self, path: str) -> bool:
        """Vérifie si un fichier doit être indexé."""
        ext = Path(path).suffix.lower()
        if ext not in self.INCLUDE_EXTENSIONS:
            return False
        return all(pattern not in path for pattern in self.EXCLUDE_PATTERNS)

    def fetch_file_content(self, path: str, branch: str = "master") -> str | None:
        """Récupère le contenu d'un fichier."""
        if not HTTPX_AVAILABLE:
            return None
        url = f"https://raw.githubusercontent.com/{self.owner}/{self.repo}/{branch}/{path}"
        try:
            with httpx.Client() as client:
                resp = client.get(url)
                resp.raise_for_status()
                return resp.text
        except Exception:
            return None

    def fetch_all_documents(self, branch: str = "master") -> list[Document]:
        """Récupère tous les documents du repo."""
        documents = []
        tree = self.fetch_tree(branch)
        for item in tree:
            content = self.fetch_file_content(item["path"], branch)
            if content:
                doc = Document.from_github(self.repo_url, item["path"], content)
                documents.append(doc)
        return documents
