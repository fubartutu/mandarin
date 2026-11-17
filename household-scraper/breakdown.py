"""Breaks a task down with the AI (ai.py), for when a task is too big to start.

Two kinds:
- "steps": a project into its main stages ("Do lawn renovation" into "Scalp
  the lawn", "Dethatch the lawn" ...). Mandarin makes each one a subtask.
- "chunks": one stage into small actions that are easy to start and done in
  a few minutes. Mandarin adds them as a checklist on the task.

The prompts are written for ADHD task paralysis: every item starts with a
verb, is something you can physically do, and the first chunk is trivially
easy so starting doesn't need willpower.
"""

from bs4 import BeautifulSoup
from fastapi import HTTPException

import ai

MAX_STEPS = 15
MAX_TITLE = 200
MAX_DESCRIPTION = 4000

SCHEMA = {
    "type": "object",
    "properties": {
        "steps": {
            "type": "array",
            "items": {
                "type": "object",
                "properties": {"title": {"type": "string"}, "minutes": {"type": "integer"}},
                "required": ["title", "minutes"],
                "additionalProperties": False,
            },
        },
    },
    "required": ["steps"],
    "additionalProperties": False,
}

COMMON = """Rules for every item:
- Start with a verb and say exactly what to do ("Mow the lawn on the lowest setting", not "Mowing").
- Concrete and physical: something you could start right now, no vague items like "Prepare" or "Get organised".
- Short: under 12 words. Australian English spelling.
- In the order they should be done.
- minutes: a rough, honest estimate of how long it takes.
- If the task is already tiny or makes no sense, return an empty list.
"""

STEPS = """Someone with ADHD finds this task too big to start. Break it into its main stages: 3 to 8 steps, each a meaningful part of the job that could be a task of its own.
Leave out planning or research steps unless the job really can't be done without them.
""" + COMMON

CHUNKS = """Someone with ADHD finds this task overwhelming. Break it into small chunks that are easy to start: 4 to 10 chunks of about 5 to 15 minutes each.
- Make the first chunk trivially easy (like "Put on your gardening shoes" or "Get the rake from the shed"), so starting takes no willpower.
- Include getting things out and packing them away when the task needs it.
- Only cover this task, not the other stages listed below.
""" + COMMON


def _plain(html: str) -> str:
    """A task description (Vikunja keeps HTML) as plain text."""
    text = BeautifulSoup(html or "", "html.parser").get_text("\n")
    lines = (line.strip() for line in text.splitlines())
    return "\n".join(line for line in lines if line)[:MAX_DESCRIPTION]


def prompt(mode: str, title: str, description: str = "", project: str = "", parent: str = "", siblings: list[str] | None = None) -> str:
    """The prompt for one breakdown; separate from suggest() so tests can read it."""
    parts = [CHUNKS if mode == "chunks" else STEPS, "", f"Task: {title.strip()}"]
    if project.strip():
        parts.append(f"Project: {project.strip()}")
    if parent.strip():
        parts.append(f"It is one stage of: {parent.strip()}")
    others = [s.strip() for s in siblings or [] if s.strip() and s.strip() != title.strip()]
    if others:
        parts.append("Other stages (done separately): " + "; ".join(others))
    notes = _plain(description)
    if notes:
        parts += ["Notes on the task:", notes]
    return "\n".join(parts)


def suggest(mode: str, title: str, description: str = "", project: str = "", parent: str = "", siblings: list[str] | None = None) -> dict:
    """{"steps": [{"title", "minutes"}]} for the task, cleaned up."""
    if mode not in ("steps", "chunks"):
        raise HTTPException(422, "mode must be steps or chunks.")
    if not title.strip():
        raise HTTPException(422, "The task has no title.")
    data = ai.ask(
        prompt(mode, title, description, project, parent, siblings),
        SCHEMA, "breakdown", "Save the steps the task is broken down into.",
    )
    steps = []
    for item in data.get("steps") or []:
        if not isinstance(item, dict):
            continue
        text = " ".join(str(item.get("title") or "").split())[:MAX_TITLE]
        if not text:
            continue
        try:
            minutes = max(0, min(int(item.get("minutes") or 0), 24 * 60))
        except (TypeError, ValueError):
            minutes = 0
        steps.append({"title": text, "minutes": minutes})
    if not steps:
        raise HTTPException(422, "The AI didn't find a way to break that down.")
    return {"steps": steps[:MAX_STEPS]}
