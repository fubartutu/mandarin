"""Tests for task breakdown, against a stand-in AI."""

import json

import httpx
import pytest
from fastapi.testclient import TestClient

import ai
import app as scraper_app
import breakdown
import settings

LAWN_STEPS = {"steps": [
    {"title": "Scalp the lawn", "minutes": 90},
    {"title": "  Dethatch   the lawn ", "minutes": 60},
    {"title": "", "minutes": 5},
    {"title": "Core aerate the lawn", "minutes": "lots"},
]}

seen: list[dict] = []


def fake_ollama(request: httpx.Request) -> httpx.Response:
    body = json.loads(request.content)
    seen.append(body)
    assert body["format"]["required"] == ["steps"]
    return httpx.Response(200, json={"message": {"content": json.dumps(LAWN_STEPS)}})


def fake_anthropic(request: httpx.Request) -> httpx.Response:
    body = json.loads(request.content)
    assert body["tool_choice"] == {"type": "tool", "name": "save_breakdown"}
    return httpx.Response(200, json={"content": [{"type": "tool_use", "name": "save_breakdown", "input": LAWN_STEPS}]})


@pytest.fixture
def client(monkeypatch):
    for name in ("AI_API_KEY", "AI_MODEL", "OLLAMA_URL", "OLLAMA_MODEL"):
        monkeypatch.delenv(name, raising=False)
    seen.clear()
    monkeypatch.setattr(ai, "_ollama_client", lambda: httpx.Client(
        base_url="http://sampc:11434", transport=httpx.MockTransport(fake_ollama)))
    monkeypatch.setattr(ai, "_api_client", lambda: httpx.Client(
        base_url=settings.ai_api_url(), transport=httpx.MockTransport(fake_anthropic)))
    return TestClient(scraper_app.app)


def with_ollama(monkeypatch):
    monkeypatch.setenv("OLLAMA_URL", "http://sampc:11434")
    monkeypatch.setenv("OLLAMA_MODEL", "test-model")


def test_steps_are_cleaned_up(client, monkeypatch):
    with_ollama(monkeypatch)
    response = client.post("/breakdown", json={"mode": "steps", "title": "Do lawn renovation"})
    assert response.status_code == 200
    assert response.json() == {"steps": [
        {"title": "Scalp the lawn", "minutes": 90},
        {"title": "Dethatch the lawn", "minutes": 60},
        {"title": "Core aerate the lawn", "minutes": 0},
    ]}
    assert "main stages" in seen[0]["messages"][0]["content"]


def test_chunks_prompt_has_the_context(client, monkeypatch):
    with_ollama(monkeypatch)
    client.post("/breakdown", json={
        "mode": "chunks", "title": "Scalp the lawn", "description": "<p>Use the <b>old</b> mower</p>",
        "project": "Garden", "parent": "Do lawn renovation", "siblings": ["Scalp the lawn", "Dethatch the lawn"],
    })
    text = seen[0]["messages"][0]["content"]
    assert "trivially easy" in text
    assert "It is one stage of: Do lawn renovation" in text
    assert "Other stages (done separately): Dethatch the lawn" in text
    assert "Use the\nold\nmower" in text
    assert "Project: Garden" in text


def test_claude_works_too(client, monkeypatch):
    monkeypatch.setenv("AI_API_KEY", "sk-ant-good")
    monkeypatch.setenv("AI_MODEL", "claude-model")
    response = client.post("/breakdown", json={"mode": "steps", "title": "Do lawn renovation"})
    assert response.json()["steps"][0]["title"] == "Scalp the lawn"


def test_no_ai(client):
    response = client.post("/breakdown", json={"mode": "steps", "title": "Do lawn renovation"})
    assert response.status_code == 503
    assert "isn't set up" in response.json()["detail"]


def test_an_empty_answer(client, monkeypatch):
    with_ollama(monkeypatch)
    monkeypatch.setattr(breakdown.ai, "ask", lambda *args: {"steps": []})
    response = client.post("/breakdown", json={"mode": "chunks", "title": "Breathe"})
    assert response.status_code == 422


def test_bad_mode(client):
    assert client.post("/breakdown", json={"mode": "everything", "title": "x"}).status_code == 422
