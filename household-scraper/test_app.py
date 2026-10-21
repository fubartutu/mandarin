"""Tests for the scraper. They use a saved page, so they run offline:

    pip install -r requirements-dev.txt && pytest
"""

import json

import pytest
from fastapi import HTTPException
from fastapi.testclient import TestClient

import app as scraper_app
from ingredients import parse_line

RECIPE = {
    "@context": "https://schema.org",
    "@type": "Recipe",
    "name": "Weeknight Pasta",
    "description": "Quick tomato pasta.",
    "recipeYield": "4 servings",
    "prepTime": "PT10M",
    "cookTime": "PT20M",
    "totalTime": "PT30M",
    "recipeCategory": "Mains",
    "recipeCuisine": "Italian",
    "image": ["https://example.com/pasta.jpg", "https://example.com/pasta-2.jpg"],
    "recipeIngredient": [
        "400 g spaghetti",
        "2 x 400g cans crushed tomatoes",
        "2-3 cloves garlic, crushed",
        "Salt and pepper to taste",
    ],
    "recipeInstructions": [
        {"@type": "HowToStep", "text": "Boil the pasta."},
        {"@type": "HowToStep", "text": "Simmer the sauce."},
    ],
}
PAGE = f'<html><head><script type="application/ld+json">{json.dumps(RECIPE)}</script></head><body></body></html>'


@pytest.fixture
def client(monkeypatch):
    def fake_fetch(url):
        if "norecipe" in url:
            return "<html><body>Hello</body></html>", url
        return PAGE, url

    monkeypatch.setattr(scraper_app, "_fetch", fake_fetch)
    return TestClient(scraper_app.app)


def test_scrape_reads_schema_org(client):
    response = client.post("/scrape", json={"url": "https://blog.example.com/pasta"})
    assert response.status_code == 200
    recipe = response.json()
    assert recipe["title"] == "Weeknight Pasta"
    assert recipe["servings"] == 4
    assert (recipe["prep_minutes"], recipe["cook_minutes"], recipe["total_minutes"]) == (10, 20, 30)
    assert recipe["steps"] == ["Boil the pasta.", "Simmer the sauce."]
    assert recipe["tags"] == ["Mains", "Italian"]
    assert recipe["image_urls"] == ["https://example.com/pasta.jpg", "https://example.com/pasta-2.jpg"]
    first = recipe["ingredients"][0]
    assert (first["quantity"], first["unit"], first["name"]) == (400, "g", "spaghetti")
    assert first["text"] == "400 g spaghetti"


def test_scrape_without_a_recipe_is_422(client):
    response = client.post("/scrape", json={"url": "https://example.com/norecipe"})
    assert response.status_code == 422


def test_fetch_rejects_other_schemes():
    with pytest.raises(HTTPException) as e:
        scraper_app._fetch("file:///etc/passwd")
    assert e.value.status_code == 422


@pytest.mark.parametrize(
    ("line", "quantity", "unit", "name", "note"),
    [
        ("2 cups (250g) plain flour, sifted", 250, "g", "plain flour", "sifted"),  # metric wins
        ("2 x 400g cans crushed tomatoes", 2, "cans", "crushed tomatoes", ""),  # buy cans
        ("2-3 cloves garlic, crushed", 3, "cloves", "garlic", "crushed"),  # the top of a range
        ("½ tsp salt", 0.5, "tsp", "salt", ""),
        ("1 L milk", 1, "L", "milk", ""),
        ("180g chicken breast (, thinly sliced)", 180, "g", "chicken breast", "thinly sliced"),
        ("1 tbsp Chinese Cooking Wine (OR Mirin)", 1, "tbsp", "Chinese Cooking Wine OR Mirin", ""),
        ("Salt and pepper to taste", 0, "", "Salt and pepper", "to taste"),
    ],
)
def test_parse_line(line, quantity, unit, name, note):
    parsed = parse_line(line)
    assert (parsed["quantity"], parsed["unit"], parsed["name"], parsed["note"]) == (quantity, unit, name, note)
    assert parsed["text"] == line


def test_parse_endpoint(client):
    response = client.post("/parse", json={"lines": ["1 kg flour", ""]})
    assert response.status_code == 200
    lines = response.json()["ingredients"]
    assert lines[0]["name"] == "flour"
    assert lines[1] == {"text": "", "quantity": 0, "unit": "", "name": "", "note": ""}
