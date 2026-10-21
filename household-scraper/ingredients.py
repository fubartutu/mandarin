"""Splits ingredient lines into quantity, unit, name and note.

Uses ingredient-parser-nlp, a model trained on recipe ingredient lines, so
"2 cups (250g) plain flour, sifted" becomes 250 g / plain flour / sifted.
"""

import re
from fractions import Fraction

from ingredient_parser import parse_ingredient

# pint's unit names, shortened the way an Australian recipe writes them.
SHORT_UNITS = {
    "gram": "g",
    "kilogram": "kg",
    "milligram": "mg",
    "liter": "L",
    "milliliter": "mL",
    "teaspoon": "tsp",
    "tablespoon": "tbsp",
    "cup": "cup",
    "ounce": "oz",
    "fluid_ounce": "fl oz",
    "pound": "lb",
    "pint": "pint",
    "quart": "qt",
    "gallon": "gal",
    "centimeter": "cm",
    "millimeter": "mm",
    "inch": "inch",
}


def _unit(unit) -> str:
    if isinstance(unit, str):
        return unit  # not a measure pint knows, e.g. "can", "cloves"
    name = str(unit)
    return SHORT_UNITS.get(name, name.replace("_", " "))


def _number(value) -> float:
    if isinstance(value, Fraction):
        return float(value)
    try:
        return float(value)
    except (TypeError, ValueError):
        return 0.0


def _pick_amount(amounts):
    """The amount to shop by. "1 x 400g can" gives "1 can" and a per-can
    "400 g" (SINGULAR); the can is what you buy. Between "2 cups" and
    "250 g" for the same thing, metric wins."""
    usable = [a for a in amounts if not getattr(a, "SINGULAR", False)] or amounts
    for amount in usable:
        system = getattr(amount, "unit_system", None)
        if system is not None and getattr(system, "value", "") == "metric":
            return amount
    return usable[0] if usable else None


def parse_line(text: str) -> dict:
    """One ingredient line as {text, quantity, unit, name, note}."""
    text = " ".join(text.split())
    result = {"text": text, "quantity": 0.0, "unit": "", "name": text, "note": ""}
    if not text:
        return result
    try:
        # One name per line: "wine OR mirin" stays a single name.
        parsed = parse_ingredient(text, separate_names=False)
    except Exception:  # noqa: BLE001 - a line we can't parse is kept as text
        return result

    names = [n.text for n in (parsed.name or [])]
    if names:
        result["name"] = _tidy(" ".join(names))

    amount = _pick_amount(list(parsed.amount or []))
    if amount is not None:
        # A range like "2-3 cloves": shop for the larger amount.
        quantity = getattr(amount, "quantity_max", None) or amount.quantity
        result["quantity"] = _number(quantity)
        result["unit"] = _unit(amount.unit)

    notes = [_tidy(part.text) for part in (parsed.size, parsed.preparation, parsed.comment) if part]
    result["note"] = ", ".join(n for n in notes if n)
    return result


def _tidy(text: str) -> str:
    """Drops the brackets and stray commas blogs leave around notes:
    "(, thinly sliced)" becomes "thinly sliced"."""
    text = re.sub(r"[()]", " ", text)
    text = re.sub(r"\s+,", ",", " ".join(text.split()))
    return text.strip(" ,;")
