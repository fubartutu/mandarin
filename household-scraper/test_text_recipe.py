"""Tests for the simple rules that read recipes from OCR text."""

from text_recipe import parse_text

# Roughly what Paperless's OCR gives for a printed cookbook page.
COOKBOOK_PAGE = """
42 MAINS

BEEF AND GUINNESS PIE
A proper winter pie with a flaky lid.

Serves 6 Prep 30 mins Cook 2 hours 30 mins

INGREDIENTS
2 tbsp olive oil
1.2 kg chuck steak, cut into 3 cm
pieces
2 onions, chopped
440 ml can Guinness
PASTRY
2 sheets frozen puff pastry, thawed
1 egg, lightly beaten

METHOD
1. Heat the oil in a large saucepan over high heat. Brown the beef in batches.
2. Add the onions and cook for 5 minutes until
soft.
3. Pour in the Guinness, cover and simmer for 2 hours.
4. Fill a pie dish, top with pastry and bake for 30 minutes.

TIP
Make the filling a day ahead.
"""


def test_reads_a_cookbook_page():
    draft = parse_text(COOKBOOK_PAGE, "scan_0042")
    assert draft["complete"]
    assert draft["title"] == "Beef and Guinness Pie"  # the file name is ignored
    assert draft["description"] == "A proper winter pie with a flaky lid."
    assert (draft["servings"], draft["prep_minutes"], draft["cook_minutes"]) == (6, 30, 150)
    assert draft["ingredient_groups"] == [
        ("", ["2 tbsp olive oil", "1.2 kg chuck steak, cut into 3 cm pieces", "2 onions, chopped", "440 ml can Guinness"]),
        ("Pastry", ["2 sheets frozen puff pastry, thawed", "1 egg, lightly beaten"]),
    ]
    assert draft["steps"] == [
        "Heat the oil in a large saucepan over high heat. Brown the beef in batches.",
        "Add the onions and cook for 5 minutes until soft.",
        "Pour in the Guinness, cover and simmer for 2 hours.",
        "Fill a pie dish, top with pastry and bake for 30 minutes.",
    ]
    assert draft["notes"] == "Make the filling a day ahead."


def test_prefers_a_real_document_title():
    assert parse_text(COOKBOOK_PAGE, "Nan's beef pie")["title"] == "Nan's beef pie"


def test_unnumbered_steps_split_at_full_stops():
    text = "Scones\nIngredients\n3 cups self-raising flour\n1 cup cream\n1 cup lemonade\nMethod\nMix everything into a soft\ndough.\nPat out and cut into rounds.\nBake at 220C for 15 minutes."
    draft = parse_text(text)
    assert draft["title"] == "Scones"
    assert draft["steps"] == ["Mix everything into a soft dough.", "Pat out and cut into rounds.", "Bake at 220C for 15 minutes."]
    assert draft["complete"]


def test_without_headings_it_says_so():
    draft = parse_text("Grandma's slice\nmix butter sugar flour bake 20 min", "Grandma's slice")
    assert not draft["complete"]
    assert draft["title"] == "Grandma's slice"
