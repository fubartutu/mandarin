"""Reads a recipe out of text with a language model: Anthropic's API or an
OpenAI-compatible one when a key is set up, else Ollama on sampc.

Used when the simple rules (text_recipe.py) can't make sense of a scan, and
for web pages that publish no recipe data. The model only sorts the text into
fields and must keep ingredient lines as written; amounts are split out by
the same parser as everything else.
"""

import json

import httpx
from fastapi import HTTPException

import settings

MAX_TEXT = 20_000  # characters; a recipe page is well under this

SCHEMA = {
    "type": "object",
    "properties": {
        "title": {"type": "string"},
        "description": {"type": "string"},
        "servings": {"type": "number"},
        "yields": {"type": "string"},
        "prep_minutes": {"type": "integer"},
        "cook_minutes": {"type": "integer"},
        "total_minutes": {"type": "integer"},
        "ingredient_groups": {
            "type": "array",
            "items": {
                "type": "object",
                "properties": {"group": {"type": "string"}, "lines": {"type": "array", "items": {"type": "string"}}},
                "required": ["group", "lines"],
                "additionalProperties": False,
            },
        },
        "steps": {"type": "array", "items": {"type": "string"}},
        "notes": {"type": "string"},
        "tags": {"type": "array", "items": {"type": "string"}},
    },
    "additionalProperties": False,
}
# OpenAI's strict structured outputs want every field required; the prompt
# says what to put when the page doesn't give one.
SCHEMA["required"] = list(SCHEMA["properties"])

PROMPT = """Extract the recipe from the text below into JSON.

Rules:
- Copy each ingredient line exactly as written, one per entry, including its amount. Join lines that were split across a line break.
- Put ingredients under their heading (e.g. "Sauce") in "group"; use "" when there is no heading.
- One cooking step per entry in "steps", without step numbers.
- servings is a number (0 if not given). Times are whole minutes (0 if not given).
- tags: at most 3 short ones, like the course or cuisine.
- notes: tips or variations printed with the recipe, else "".
- Leave out anything that isn't part of this recipe (adverts, page numbers, other recipes).
- Use "" or 0 for anything the text doesn't give.
- If there is no recipe in the text, return an empty title.

Text:
"""


def _ollama_client() -> httpx.Client:
    """A client for Ollama. Loading a model on sampc can take a while. Tests replace this."""
    return httpx.Client(base_url=settings.ollama_url(), timeout=240)


def _api_client() -> httpx.Client:
    """A client for the AI API (Anthropic or OpenAI-compatible). Tests replace this."""
    if settings.ai_is_anthropic():
        headers = {"x-api-key": settings.ai_api_key(), "anthropic-version": "2023-06-01"}
    else:
        headers = {"Authorization": f"Bearer {settings.ai_api_key()}"}
    return httpx.Client(base_url=settings.ai_api_url(), headers=headers, timeout=120)


def _api_error(response: httpx.Response) -> None:
    """Turns an API's refusal into a message for the app."""
    if response.status_code in (401, 403):
        raise HTTPException(502, "The AI service refused the API key.")
    if response.status_code in (429, 529):
        raise HTTPException(503, "The AI service is busy or out of credit. Try again later.")
    if response.status_code >= 400:
        raise HTTPException(502, f"The AI service answered {response.status_code}. Check AI_MODEL.")


def _ask_anthropic(prompt: str) -> str:
    """The reply's JSON text from Anthropic's Messages API. The model must
    call a "save_recipe" tool whose input follows the schema, which makes
    the reply's shape reliable."""
    body = {
        "model": settings.ai_model(),
        "max_tokens": 4096,
        "tools": [{"name": "save_recipe", "description": "Save the recipe found in the text.", "input_schema": SCHEMA}],
        "tool_choice": {"type": "tool", "name": "save_recipe"},
        "messages": [{"role": "user", "content": prompt}],
    }
    try:
        with _api_client() as client:
            response = client.post("/v1/messages", json=body)
    except httpx.HTTPError as e:
        raise HTTPException(503, "Couldn't reach the AI service.") from e
    _api_error(response)
    try:
        call = next(block for block in response.json()["content"] if block.get("type") == "tool_use")
        return json.dumps(call["input"])
    except (KeyError, StopIteration, ValueError, TypeError, AttributeError) as e:
        raise HTTPException(502, "The AI's answer couldn't be read.") from e


def _ask_api(prompt: str) -> str:
    """The reply's JSON text from the Chat Completions API, in strict JSON-schema mode."""
    body = {
        "model": settings.ai_model(),
        "messages": [{"role": "user", "content": prompt}],
        "response_format": {"type": "json_schema", "json_schema": {"name": "recipe", "strict": True, "schema": SCHEMA}},
    }
    try:
        with _api_client() as client:
            response = client.post("/chat/completions", json=body)
    except httpx.HTTPError as e:
        raise HTTPException(503, "Couldn't reach the AI service.") from e
    _api_error(response)
    try:
        return response.json()["choices"][0]["message"]["content"]
    except (KeyError, IndexError, ValueError, TypeError) as e:
        raise HTTPException(502, "The AI's answer couldn't be read.") from e


def _ask_ollama(prompt: str) -> str:
    """The reply's JSON text from Ollama, with its structured outputs."""
    body = {
        "model": settings.ollama_model(),
        "messages": [{"role": "user", "content": prompt}],
        "format": SCHEMA,
        "stream": False,
        "options": {"temperature": 0},
    }
    try:
        with _ollama_client() as client:
            response = client.post("/api/chat", json=body)
    except httpx.HTTPError as e:
        raise HTTPException(503, "Couldn't reach the AI on sampc. Is sampc switched on?") from e
    if response.status_code >= 400:
        raise HTTPException(502, f"The AI answered {response.status_code}. Is the model pulled on sampc?")
    try:
        return response.json()["message"]["content"]
    except (KeyError, ValueError, TypeError) as e:
        raise HTTPException(502, "The AI's answer couldn't be read.") from e


def extract(text: str) -> dict:
    """A recipe draft in the same shape as text_recipe.parse_text."""
    prompt = PROMPT + text[:MAX_TEXT]
    if settings.api_enabled():
        reply = _ask_anthropic(prompt) if settings.ai_is_anthropic() else _ask_api(prompt)
    elif settings.ollama_enabled():
        reply = _ask_ollama(prompt)
    else:
        raise HTTPException(503, "The AI isn't set up. Add AI_API_KEY and AI_MODEL (or OLLAMA_URL and OLLAMA_MODEL) to the scraper.")
    try:
        data = json.loads(reply)
    except (ValueError, TypeError) as e:
        raise HTTPException(502, "The AI's answer couldn't be read.") from e
    if not str(data.get("title") or "").strip():
        raise HTTPException(422, "The AI found no recipe in it.")

    groups = []
    for g in data.get("ingredient_groups") or []:
        lines = [str(line).strip() for line in g.get("lines") or [] if str(line).strip()]
        if lines:
            groups.append((str(g.get("group") or "").strip().rstrip(":"), lines))
    number = lambda key: max(0, int(data.get(key) or 0))  # noqa: E731 - tiny local helper
    return {
        "title": str(data.get("title")).strip(),
        "description": str(data.get("description") or "").strip(),
        "servings": max(0.0, float(data.get("servings") or 0)),
        "yields": str(data.get("yields") or "").strip(),
        "prep_minutes": number("prep_minutes"),
        "cook_minutes": number("cook_minutes"),
        "total_minutes": number("total_minutes"),
        "ingredient_groups": groups,
        "steps": [str(s).strip() for s in data.get("steps") or [] if str(s).strip()],
        "notes": str(data.get("notes") or "").strip(),
        "tags": [str(t).strip() for t in data.get("tags") or [] if str(t).strip()][:3],
        "complete": True,
    }
