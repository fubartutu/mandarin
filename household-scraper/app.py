"""Mandarin's recipe scraper: a small web service the Mandarin server calls.

POST /scrape {"url": ...}   fetches a recipe page and returns the recipe
POST /parse  {"lines": [...]} splits ingredient lines into amount and name
GET  /health

Only the Mandarin server talks to it, over the Docker network; it has no
login of its own, so don't publish its port.

How a page is read: recipe-scrapers has site-specific code for a few hundred
sites, and otherwise reads the schema.org Recipe data that most recipe blogs
publish for search engines. An AI fallback for pages with neither is planned.
"""

import re
from urllib.parse import urlparse

from curl_cffi import requests
from curl_cffi.requests.exceptions import RequestException
from fastapi import FastAPI, HTTPException
from pydantic import BaseModel, Field
from recipe_scrapers import scrape_html

from ingredients import parse_line

MAX_PAGE_BYTES = 5 * 1024 * 1024
MAX_LINES = 300

# Many recipe sites sit behind bot protection that checks the TLS handshake,
# not just the headers. curl_cffi makes the same handshake Chrome does.
BROWSER = "chrome"

app = FastAPI(title="Mandarin recipe scraper")


class ScrapeRequest(BaseModel):
    url: str = Field(max_length=2000)


class ParseRequest(BaseModel):
    lines: list[str] = Field(max_length=MAX_LINES)


def _fetch(url: str) -> tuple[str, str]:
    """Returns the page's HTML and its final URL (after redirects)."""
    parts = urlparse(url)
    if parts.scheme not in ("http", "https") or not parts.netloc:
        raise HTTPException(422, "That isn't a web address.")
    try:
        response = requests.get(url, impersonate=BROWSER, timeout=20, stream=True,
                                headers={"Accept-Language": "en-AU,en;q=0.9"})
        try:
            if response.status_code >= 400:
                raise HTTPException(502, f"The site answered {response.status_code}.")
            body = b""
            for chunk in response.iter_content():
                body += chunk
                if len(body) > MAX_PAGE_BYTES:
                    raise HTTPException(502, "The page is too big.")
        finally:
            response.close()
    except RequestException as e:
        raise HTTPException(502, "Couldn't load the page.") from e
    return body.decode(response.encoding or "utf-8", errors="replace"), str(response.url)


def _try(method, default=None):
    """Calls a scraper field; sites leave many of them out, which raises."""
    try:
        value = method()
    except Exception:  # noqa: BLE001 - every missing field raises its own type
        return default
    return default if value is None else value


def _minutes(value) -> int:
    try:
        return max(0, int(value))
    except (TypeError, ValueError):
        return 0


def _servings(yields: str) -> float:
    match = re.search(r"\d+(?:\.\d+)?", yields or "")
    return float(match.group()) if match else 0.0


def _images(scraper) -> list[str]:
    """Up to two image URLs: the main one, then any others in the schema data."""
    urls = []
    main = _try(scraper.image)
    if isinstance(main, str) and main:
        urls.append(main)
    schema_image = _try(lambda: scraper.schema.data.get("image"))
    candidates = schema_image if isinstance(schema_image, list) else [schema_image]
    for item in candidates:
        url = item.get("url") if isinstance(item, dict) else item
        if isinstance(url, str) and url.startswith("http") and url not in urls:
            urls.append(url)
    return urls[:2]


def _tags(scraper) -> list[str]:
    """Category and cuisine; keywords are mostly search-engine filler."""
    tags = []
    for method in (scraper.category, scraper.cuisine):
        value = _try(method, "")
        for tag in str(value).split(","):
            tag = tag.strip()
            if tag and tag.lower() not in (t.lower() for t in tags):
                tags.append(tag)
    return tags[:8]


def _ingredients(scraper) -> list[dict]:
    """Ingredient lines, parsed, each with the heading of its group if any."""
    groups = _try(scraper.ingredient_groups, [])
    if not groups:
        groups = [type("Group", (), {"purpose": None, "ingredients": _try(scraper.ingredients, [])})]
    lines = []
    for group in groups:
        for text in group.ingredients or []:
            line = parse_line(text)
            if line["text"]:
                line["group"] = (group.purpose or "").strip().rstrip(":").strip()
                lines.append(line)
    return lines[:MAX_LINES]


@app.get("/health")
def health():
    return {"ok": True}


@app.post("/scrape")
def scrape(request: ScrapeRequest):
    html, final_url = _fetch(request.url.strip())
    try:
        scraper = scrape_html(html, org_url=final_url, supported_only=False)
    except Exception as e:  # noqa: BLE001 - "no recipe here" comes in several types
        raise HTTPException(422, "No recipe found on that page.") from e

    title = _try(scraper.title, "")
    ingredients = _ingredients(scraper)
    if not title or not ingredients:
        raise HTTPException(422, "No recipe found on that page.")

    yields = str(_try(scraper.yields, ""))
    steps = [s.strip() for s in _try(scraper.instructions_list, []) if s and s.strip()]
    return {
        "title": title.strip(),
        "source_url": final_url,
        "description": str(_try(scraper.description, "")).strip(),
        "servings": _servings(yields),
        "yields": yields,
        "prep_minutes": _minutes(_try(scraper.prep_time)),
        "cook_minutes": _minutes(_try(scraper.cook_time)),
        "total_minutes": _minutes(_try(scraper.total_time)),
        "ingredients": ingredients,
        "steps": steps,
        "tags": _tags(scraper),
        "image_urls": _images(scraper),
    }


@app.post("/parse")
def parse(request: ParseRequest):
    return {"ingredients": [parse_line(line) for line in request.lines]}
