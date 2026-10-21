"""Tests for the Paperless import and the AI, against stand-ins for both."""

import io
import json

import httpx
import pytest
from fastapi.testclient import TestClient
from PIL import Image

import ai
import app as scraper_app
import paperless
import settings
from test_text_recipe import COOKBOOK_PAGE

MESSY_TEXT = "Grandma's slice\nmix butter sugar flour bake 20 min"


def png_bytes() -> bytes:
    out = io.BytesIO()
    Image.new("RGB", (3000, 2000), "white").save(out, "PNG")
    return out.getvalue()


def fake_paperless(request: httpx.Request) -> httpx.Response:
    assert request.headers["Authorization"] == "Token secret"
    path = request.url.path
    if path == "/api/documents/":
        assert request.url.params["tags__name__iexact"] == "recipe"
        return httpx.Response(200, json={"results": [{"id": 7, "title": "Beef pie", "created": "2026-01-02"}]})
    if path == "/api/documents/7/":
        return httpx.Response(200, json={"id": 7, "title": "scan_0042", "content": COOKBOOK_PAGE})
    if path == "/api/documents/8/":
        return httpx.Response(200, json={"id": 8, "title": "Grandma's slice", "content": MESSY_TEXT})
    if path == "/api/documents/7/thumb/":
        return httpx.Response(200, content=b"webp", headers={"content-type": "image/webp"})
    if path == "/api/documents/7/download/":
        return httpx.Response(200, content=png_bytes())
    return httpx.Response(404)


AI_ANSWER = {
    "title": "Grandma's slice",
    "servings": 0,
    "ingredient_groups": [{"group": "", "lines": ["125 g butter", "1 cup sugar"]}],
    "steps": ["Mix.", "Bake for 20 minutes."],
    "tags": ["Baking"],
}


def fake_ollama(request: httpx.Request) -> httpx.Response:
    body = json.loads(request.content)
    assert body["model"] == "test-model"
    assert "title" in body["format"]["required"]
    return httpx.Response(200, json={"message": {"content": json.dumps(AI_ANSWER)}})


def fake_api(request: httpx.Request) -> httpx.Response:
    """Like OpenAI's Chat Completions, or Anthropic's Messages."""
    if request.url.host == "api.anthropic.com":
        return fake_anthropic(request)
    assert request.url.path == "/v1/chat/completions"
    if request.headers["Authorization"] != "Bearer good-key":
        return httpx.Response(401, json={"error": {"message": "bad key"}})
    body = json.loads(request.content)
    assert body["model"] == "api-model"
    schema = body["response_format"]["json_schema"]
    assert schema["strict"] is True
    assert set(schema["schema"]["required"]) == set(schema["schema"]["properties"])  # strict mode needs all
    answer = {**AI_ANSWER, "title": "Grandma's slice (API)"}
    return httpx.Response(200, json={"choices": [{"message": {"role": "assistant", "content": json.dumps(answer)}}]})


@pytest.fixture
def client(monkeypatch):
    monkeypatch.setenv("PAPERLESS_URL", "http://paperless:8000")
    monkeypatch.setenv("PAPERLESS_PUBLIC_URL", "https://paperless.example.com")
    monkeypatch.setenv("PAPERLESS_TOKEN", "secret")
    monkeypatch.setattr(paperless, "_client", lambda: httpx.Client(
        base_url="http://paperless:8000", transport=httpx.MockTransport(fake_paperless),
        headers={"Authorization": "Token secret"}))
    monkeypatch.setattr(ai, "_ollama_client", lambda: httpx.Client(
        base_url="http://sampc:11434", transport=httpx.MockTransport(fake_ollama)))
    monkeypatch.setattr(ai, "_api_client", lambda: httpx.Client(
        base_url=settings.ai_api_url(), transport=httpx.MockTransport(fake_api),
        headers={"Authorization": f"Bearer {settings.ai_api_key()}", "x-api-key": settings.ai_api_key()}))
    for name in ("AI_API_KEY", "AI_MODEL", "OLLAMA_URL", "OLLAMA_MODEL"):
        monkeypatch.delenv(name, raising=False)
    return TestClient(scraper_app.app)


def with_ai(monkeypatch):
    monkeypatch.setenv("OLLAMA_URL", "http://sampc:11434")
    monkeypatch.setenv("OLLAMA_MODEL", "test-model")


def test_lists_recipe_documents(client):
    response = client.get("/paperless/documents")
    assert response.status_code == 200
    assert response.json()["documents"] == [{"id": 7, "title": "Beef pie", "created": "2026-01-02"}]


def test_imports_a_printed_page_with_the_rules(client):
    recipe = client.post("/paperless/documents/7/import").json()
    assert recipe["import_method"] == "rules"
    assert recipe["import_complete"]
    assert recipe["title"] == "Beef and Guinness Pie"
    assert recipe["source_url"] == "https://paperless.example.com/documents/7/details"
    assert recipe["image_urls"] == ["paperless:documents/7/image"]
    pastry = [i for i in recipe["ingredients"] if i["group"] == "Pastry"]
    assert pastry[0]["name"] == "frozen puff pastry"


def test_a_messy_page_goes_to_the_ai(client, monkeypatch):
    with_ai(monkeypatch)
    recipe = client.post("/paperless/documents/8/import").json()
    assert recipe["import_method"] == "ai"
    assert [i["name"] for i in recipe["ingredients"]] == ["butter", "sugar"]
    assert recipe["steps"] == ["Mix.", "Bake for 20 minutes."]


def test_a_messy_page_without_ai_still_gives_the_rules_attempt(client):
    recipe = client.post("/paperless/documents/8/import").json()
    assert recipe["import_method"] == "rules"
    assert not recipe["import_complete"]
    assert not recipe["ai_available"]
    assert "isn't set up" in recipe["import_note"]


def test_ai_on_request_needs_the_ai(client):
    response = client.post("/paperless/documents/7/import?ai_first=true")
    assert response.status_code == 503


def test_a_failed_ai_says_why_and_keeps_the_rules_attempt(client, monkeypatch):
    monkeypatch.setenv("AI_API_KEY", "sk-ant-wrong")
    monkeypatch.setenv("AI_MODEL", "claude-model")
    monkeypatch.setattr(ai, "_api_client", lambda: httpx.Client(
        base_url="https://api.anthropic.com", transport=httpx.MockTransport(lambda r: httpx.Response(401))))
    recipe = client.post("/paperless/documents/8/import").json()
    assert recipe["import_method"] == "rules"
    assert recipe["import_note"] == "The AI couldn't read it: The AI service refused the API key."


def test_sampc_off(client, monkeypatch):
    with_ai(monkeypatch)

    def down(request):
        raise httpx.ConnectError("refused")

    monkeypatch.setattr(ai, "_ollama_client", lambda: httpx.Client(base_url="http://sampc:11434", transport=httpx.MockTransport(down)))
    response = client.post("/paperless/documents/7/import?ai_first=true")
    assert response.status_code == 503
    assert "sampc" in response.json()["detail"]


def test_page_image_and_thumbnail(client):
    image = client.get("/paperless/documents/7/image")
    assert image.headers["content-type"] == "image/jpeg"
    assert max(Image.open(io.BytesIO(image.content)).size) == 1600
    thumb = client.get("/paperless/documents/7/thumb")
    assert thumb.content == b"webp"


def test_not_set_up(monkeypatch):
    monkeypatch.delenv("PAPERLESS_URL", raising=False)
    response = TestClient(scraper_app.app).get("/paperless/documents")
    assert response.status_code == 503


def test_web_page_without_recipe_data_goes_to_the_ai(monkeypatch):
    with_ai(monkeypatch)
    monkeypatch.setattr(ai, "_ollama_client", lambda: httpx.Client(base_url="http://sampc:11434", transport=httpx.MockTransport(fake_ollama)))
    page = "<html><body><nav>Home</nav><article><h1>Grandma's slice</h1><p>125 g butter</p></article></body></html>"
    monkeypatch.setattr(scraper_app, "_fetch", lambda url: (page, url))
    recipe = TestClient(scraper_app.app).post("/scrape", json={"url": "https://blog.example.com/slice"}).json()
    assert recipe["import_method"] == "ai"
    assert recipe["title"] == "Grandma's slice"


def fake_anthropic(request: httpx.Request) -> httpx.Response:
    assert request.url.path == "/v1/messages"
    assert request.headers["x-api-key"] == "sk-ant-good"
    body = json.loads(request.content)
    assert body["tool_choice"] == {"type": "tool", "name": "save_recipe"}
    assert body["tools"][0]["input_schema"]["properties"]["steps"]["type"] == "array"
    answer = {**AI_ANSWER, "title": "Grandma's slice (Claude)"}
    return httpx.Response(200, json={"content": [
        {"type": "text", "text": "Saving it."},
        {"type": "tool_use", "id": "t1", "name": "save_recipe", "input": answer},
    ]})


def test_an_anthropic_key_uses_claude(client, monkeypatch):
    monkeypatch.setenv("AI_API_KEY", "sk-ant-good")
    monkeypatch.setenv("AI_MODEL", "claude-model")
    assert settings.ai_api_url() == "https://api.anthropic.com"
    recipe = client.post("/paperless/documents/8/import").json()
    assert recipe["import_method"] == "ai"
    assert recipe["title"] == "Grandma's slice (Claude)"


def test_the_api_is_used_when_set_up(client, monkeypatch):
    with_ai(monkeypatch)  # Ollama too: the API wins
    monkeypatch.setenv("AI_API_KEY", "good-key")
    monkeypatch.setenv("AI_MODEL", "api-model")
    recipe = client.post("/paperless/documents/8/import").json()
    assert recipe["import_method"] == "ai"
    assert recipe["title"] == "Grandma's slice (API)"


def test_a_bad_api_key_says_so(client, monkeypatch):
    monkeypatch.setenv("AI_API_KEY", "wrong")
    monkeypatch.setenv("AI_MODEL", "api-model")
    response = client.post("/paperless/documents/7/import?ai_first=true")
    assert response.status_code == 502
    assert "API key" in response.json()["detail"]
