"""Reads a recipe out of plain text, such as a scanned page's OCR text.

Simple rules that cover the usual printed layout: a title, an "Ingredients"
heading with one ingredient per line, then a "Method" heading with the
steps. Two-column pages and odd layouts confuse them; the result says
whether it looks complete, so the app can offer the AI instead.
"""

import re

AMOUNT = re.compile(r"^(\d|[½⅓⅔¼¾⅛]|a |an |one |two |three |half |pinch |handful )", re.IGNORECASE)
INGREDIENTS = re.compile(r"^(ingredients?|you(?:'| wi)ll need|what you need)\b", re.IGNORECASE)
METHOD = re.compile(r"^(method|directions?|instructions?|preparation|steps|how to (?:make|cook)(?: it)?)\b", re.IGNORECASE)
NOTES = re.compile(r"^(notes?|tips?|cook'?s'? (?:notes?|tips?)|variations?)\b", re.IGNORECASE)
STEP_NUMBER = re.compile(r"^(?:step\s*)?(\d{1,2})\s*[.):]?\s+(?=\S)", re.IGNORECASE)
SERVES = re.compile(r"\b(?:serves|servings|makes|feeds)\b\s*:?\s*(\d+)(?:\s*[-–]\s*(\d+))?", re.IGNORECASE)
TIME = re.compile(
    r"\b(prep(?:aration)?|cook(?:ing)?|total|ready in)\b(?:\s*time)?\s*:?\s*"
    r"((?:\d+\s*(?:hours?|hrs?|h)\b\s*)?(?:\d+\s*(?:minutes?|mins?|m)\b)?)",
    re.IGNORECASE,
)
# A title that's really a file name: "scan_0012", "IMG 2041", "2024-03-01 1432".
GENERATED_TITLE = re.compile(r"^(scan|img|image|doc|document|page)[\s_-]*\d|^[\d\s_.-]+$|_", re.IGNORECASE)


SMALL_WORDS = {"a", "an", "and", "as", "at", "but", "by", "for", "in", "of", "on", "or", "the", "to", "with"}


def _title_case(text: str) -> str:
    """"BEEF AND GUINNESS PIE" → "Beef and Guinness Pie"."""
    words = text.lower().split()
    return " ".join(w if i and w in SMALL_WORDS else w[:1].upper() + w[1:] for i, w in enumerate(words))


def _minutes(text: str) -> int:
    hours = re.search(r"(\d+)\s*(?:hours?|hrs?|h)\b", text, re.IGNORECASE)
    mins = re.search(r"(\d+)\s*(?:minutes?|mins?|m)\b", text, re.IGNORECASE)
    return (int(hours.group(1)) * 60 if hours else 0) + (int(mins.group(1)) if mins else 0)


def _is_heading(line: str, pattern: re.Pattern) -> bool:
    return len(line) <= 40 and bool(pattern.match(line.rstrip(":").strip()))


def _is_page_header(line: str) -> bool:
    """A running header with a page number: "42 MAINS", "Desserts 117"."""
    return len(line) <= 40 and bool(re.match(r"^\d{1,3}\s+\D+$|^\D+\s+\d{1,3}$", line))


META_START = re.compile(r"^(serves|servings|makes|feeds|prep|preparation|cook|cooking|total|ready in)\b", re.IGNORECASE)


def _is_meta(line: str) -> bool:
    """A "Serves 4" or "Prep 10 mins" line, not part of the recipe text.
    It has to start that way: "cook for 5 minutes" in a step is not one."""
    return len(line) <= 80 and bool(META_START.match(line)) and bool(re.search(r"\d", line))


def _is_group(line: str) -> bool:
    """A heading inside the ingredients: "Sauce:", "FOR THE ICING", "To serve"."""
    if AMOUNT.match(line) or len(line) > 40:
        return False
    return line.endswith(":") or (line.isupper() and len(line) > 2) or line.lower().startswith(("for the ", "to serve"))


def parse_text(text: str, fallback_title: str = "") -> dict:
    """A recipe draft from text: the same fields as a web import, with the
    ingredients as (group, [lines]), plus "complete" when it looks right."""
    lines = [" ".join(line.split()) for line in text.splitlines()]

    ing_at = next((i for i, line in enumerate(lines) if _is_heading(line, INGREDIENTS)), -1)
    method_at = next((i for i, line in enumerate(lines) if i > ing_at and _is_heading(line, METHOD)), -1)
    notes_at = next((i for i, line in enumerate(lines) if i > max(method_at, ing_at) and _is_heading(line, NOTES)), -1)

    draft = {
        "title": "",
        "description": "",
        "servings": 0.0,
        "yields": "",
        "prep_minutes": 0,
        "cook_minutes": 0,
        "total_minutes": 0,
        "ingredient_groups": [],
        "steps": [],
        "notes": "",
        "tags": [],
    }

    # Serves and times can be anywhere on the page, on lines of their own.
    for line in filter(_is_meta, lines):
        if not draft["servings"] and (m := SERVES.search(line)):
            draft["servings"] = float(m.group(1))
            if not line.lower().startswith(("serves", "servings")):
                draft["yields"] = m.group(0)
        for m in TIME.finditer(line):
            field = {"prep": "prep_minutes", "cook": "cook_minutes"}.get(m.group(1).lower()[:4], "total_minutes")
            if not draft[field]:
                draft[field] = _minutes(m.group(2))

    # Title: the document's own name unless it's a file name, else the first
    # line that reads like one. Anything else before the ingredients is the intro.
    intro_end = ing_at if ing_at >= 0 else len(lines)
    intro = [
        line for line in lines[:intro_end]
        if line and not _is_meta(line) and not _is_page_header(line) and re.search(r"[A-Za-z]", line)
    ]
    if fallback_title and not GENERATED_TITLE.search(fallback_title):
        draft["title"] = fallback_title.strip()
        if intro and intro[0].lower() == draft["title"].lower():
            intro = intro[1:]
    elif intro:
        draft["title"] = _title_case(intro.pop(0)) if intro[0].isupper() else intro[0]
    else:
        draft["title"] = fallback_title.strip()
    draft["description"] = " ".join(intro)[:1000]

    # Ingredients: one per line; a line that starts in lower case continues
    # the one before (OCR wraps long lines).
    if ing_at >= 0:
        end = method_at if method_at >= 0 else (notes_at if notes_at >= 0 else len(lines))
        group, current = "", []
        groups = [(group, current)]
        for line in lines[ing_at + 1:end]:
            if not line or _is_meta(line):
                continue
            if _is_group(line):
                group = line.rstrip(":").strip()
                current = []
                groups.append((_title_case(group) if group.isupper() else group, current))
            elif current and line[0].islower() and not AMOUNT.match(line):
                current[-1] += " " + line
            else:
                current.append(line)
        draft["ingredient_groups"] = [(g, ls) for g, ls in groups if ls]

    # Steps: numbered ("1.", "Step 2") when the page numbers them, otherwise
    # sentences run on until a line ends with a full stop.
    if method_at >= 0:
        end = notes_at if notes_at >= 0 else len(lines)
        body = [line for line in lines[method_at + 1:end] if not _is_meta(line)]
        numbered = sum(1 for line in body if STEP_NUMBER.match(line)) >= 2
        steps, current = [], ""
        for line in body:
            if not line:
                if not numbered and current:
                    steps.append(current)
                    current = ""
                continue
            m = STEP_NUMBER.match(line) if numbered else None
            if m:
                if current:
                    steps.append(current)
                current = line[m.end():]
            else:
                current = f"{current} {line}".strip()
                if not numbered and current.endswith((".", "!", ")")):
                    steps.append(current)
                    current = ""
        if current:
            steps.append(current)
        draft["steps"] = steps

    if notes_at >= 0:
        draft["notes"] = "\n".join(line for line in lines[notes_at + 1:] if line)[:5000]

    ingredient_count = sum(len(ls) for _, ls in draft["ingredient_groups"])
    draft["complete"] = bool(draft["title"] and ingredient_count >= 2 and draft["steps"])
    return draft
