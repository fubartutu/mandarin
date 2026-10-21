"""Reads recipe scans from Paperless-ngx through its REST API.

Paperless has already OCR'd every document, so a scan's text comes from
there; this module only adds a picture of the first page.
"""

import io

import httpx
import pypdfium2
from fastapi import HTTPException
from PIL import Image

import settings

PAGE_MAX_EDGE = 1600


def _client() -> httpx.Client:
    """A client logged in to Paperless. Tests replace this."""
    return httpx.Client(
        base_url=settings.paperless_url(),
        headers={"Authorization": f"Token {settings.paperless_token()}", "Accept": "application/json"},
        timeout=30,
        follow_redirects=True,
    )


def _get(path: str, **params) -> httpx.Response:
    if not settings.paperless_enabled():
        raise HTTPException(503, "Paperless isn't set up. Add PAPERLESS_URL and PAPERLESS_TOKEN to the scraper.")
    try:
        with _client() as client:
            response = client.get(path, params=params)
    except httpx.HTTPError as e:
        raise HTTPException(502, "Couldn't reach Paperless.") from e
    if response.status_code in (401, 403):
        raise HTTPException(502, "Paperless refused the API token.")
    if response.status_code == 404:
        raise HTTPException(404, "That document isn't in Paperless.")
    if response.status_code >= 400:
        raise HTTPException(502, f"Paperless answered {response.status_code}.")
    return response


def list_documents(query: str = "") -> list[dict]:
    """Documents tagged as recipes, newest first; query searches their text."""
    params = {
        "tags__name__iexact": settings.paperless_tag(),
        "ordering": "-created",
        "page_size": 100,
        "fields": "id,title,created",  # skip the OCR text: it can be long
    }
    if query.strip():
        params["query"] = query.strip()
    results = _get("/api/documents/", **params).json().get("results", [])
    return [
        {"id": d["id"], "title": d.get("title") or f"Document {d['id']}", "created": str(d.get("created") or "")}
        for d in results
        if isinstance(d, dict) and "id" in d
    ]


def get_document(doc_id: int) -> dict:
    """The document's title and OCR text, and a link to it in Paperless."""
    d = _get(f"/api/documents/{doc_id}/").json()
    return {
        "id": doc_id,
        "title": d.get("title") or "",
        "content": d.get("content") or "",
        "url": f"{settings.paperless_public_url()}/documents/{doc_id}/details",
    }


def thumbnail(doc_id: int) -> tuple[bytes, str]:
    """Paperless's small preview image (WebP or PNG) and its type."""
    response = _get(f"/api/documents/{doc_id}/thumb/")
    return response.content, response.headers.get("content-type", "image/webp")


def page_image(doc_id: int) -> bytes:
    """The first page as a JPEG, from the original scan (a PDF or an image)."""
    original = _get(f"/api/documents/{doc_id}/download/", original="true").content
    try:
        if original.startswith(b"%PDF"):
            pdf = pypdfium2.PdfDocument(original)
            try:
                image = pdf[0].render(scale=2).to_pil()  # about 144 dpi
            finally:
                pdf.close()
        else:
            image = Image.open(io.BytesIO(original))
            image.load()
    except Exception as e:  # noqa: BLE001 - any unreadable file is "no picture"
        raise HTTPException(422, "Couldn't make a picture from that document.") from e
    image = image.convert("RGB")
    image.thumbnail((PAGE_MAX_EDGE, PAGE_MAX_EDGE))
    out = io.BytesIO()
    image.save(out, "JPEG", quality=85)
    return out.getvalue()
