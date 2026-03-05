"""Découpage de texte en chunks pour indexation RAG."""

from __future__ import annotations

import re


class TextChunker:
    """Découpe le texte en chunks pour indexation."""

    def __init__(self, chunk_size: int = 1000, overlap: int = 200):
        self.chunk_size = chunk_size
        self.overlap = overlap

    def chunk_text(self, text: str) -> list[str]:
        """Découpe le texte en chunks avec overlap."""
        if len(text) <= self.chunk_size:
            return [text]

        chunks = []
        start = 0

        while start < len(text):
            end = start + self.chunk_size

            if end < len(text):
                last_newline = text.rfind("\n", start, end)
                if last_newline > start + self.chunk_size // 2:
                    end = last_newline + 1
                else:
                    last_period = text.rfind(". ", start, end)
                    if last_period > start + self.chunk_size // 2:
                        end = last_period + 2

            chunk = text[start:end].strip()
            if chunk:
                chunks.append(chunk)

            start = end - self.overlap

        return chunks

    def chunk_code(self, code: str, language: str = "python") -> list[str]:
        """Découpe du code en chunks intelligents (par fonction/classe)."""
        if language == "python":
            return self._chunk_python(code)
        return self.chunk_text(code)

    def _chunk_python(self, code: str) -> list[str]:
        """Découpe du code Python par fonction/classe."""
        chunks = []
        current_chunk: list[str] = []
        current_size = 0

        for line in code.split("\n"):
            if (
                re.match(r"^(def |class |async def )", line)
                and current_chunk
                and current_size > 100
            ):
                chunks.append("\n".join(current_chunk))
                current_chunk = []
                current_size = 0

            current_chunk.append(line)
            current_size += len(line)

            if current_size >= self.chunk_size:
                chunks.append("\n".join(current_chunk))
                current_chunk = []
                current_size = 0

        if current_chunk:
            chunks.append("\n".join(current_chunk))

        return [c for c in chunks if c.strip()]
