"""Settings, read from the environment (set them in the compose file).

PAPERLESS_URL         Paperless-ngx as this container reaches it, e.g. http://paperless:8000
PAPERLESS_TOKEN       an API token (Paperless: your profile → API auth token)
PAPERLESS_TAG         the tag on recipe scans; default "recipe"
PAPERLESS_PUBLIC_URL  Paperless as your browser reaches it, for the link on a
                      recipe; defaults to PAPERLESS_URL
OLLAMA_URL            Ollama on sampc, e.g. http://192.168.1.20:11434
OLLAMA_MODEL          the model to use, e.g. one you've pulled with `ollama pull`

Paperless import is off without PAPERLESS_URL and PAPERLESS_TOKEN; the AI is
off without OLLAMA_URL and OLLAMA_MODEL.
"""

import os


def env(name: str, default: str = "") -> str:
    return os.environ.get(name, default).strip()


def paperless_url() -> str:
    return env("PAPERLESS_URL").rstrip("/")


def paperless_public_url() -> str:
    return (env("PAPERLESS_PUBLIC_URL") or paperless_url()).rstrip("/")


def paperless_token() -> str:
    return env("PAPERLESS_TOKEN")


def paperless_tag() -> str:
    return env("PAPERLESS_TAG", "recipe")


def paperless_enabled() -> bool:
    return bool(paperless_url() and paperless_token())


def ollama_url() -> str:
    return env("OLLAMA_URL").rstrip("/")


def ollama_model() -> str:
    return env("OLLAMA_MODEL")


def ai_enabled() -> bool:
    return bool(ollama_url() and ollama_model())
