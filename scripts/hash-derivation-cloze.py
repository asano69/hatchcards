#!/usr/bin/env python3
# uv run --with blake3 ./hash-derivation-2.py

import blake3
import struct


def extract_cloze_spans(raw: str):
    # Python port of internal/parser/parser.go's extractClozeSpans.
    # Parses cloze text containing [deletion] markers and returns the clean
    # text (without brackets) plus the byte-position spans of each deletion
    # within that clean text.
    # Special cases handled, matching the Go implementation:
    #   - ![...] Markdown image syntax: brackets are kept in clean text
    #   - \[ and \] escaped brackets: treated as literal text, not markers
    raw = raw.strip()
    data = raw.encode("utf-8")
    clean = bytearray()
    spans = []

    image_mode = False
    escape_mode = False
    start_idx = None

    i = 0
    n = len(data)
    while i < n:
        c = data[i]
        if c == ord("["):
            if image_mode:
                clean.append(c)
            elif escape_mode:
                escape_mode = False
                clean.append(c)
            else:
                start_idx = len(clean)
        elif c == ord("]"):
            if image_mode:
                image_mode = False
                clean.append(c)
            elif escape_mode:
                escape_mode = False
                clean.append(c)
            elif start_idx is not None:
                end = len(clean) - 1
                spans.append((start_idx, end))
                start_idx = None
        elif c == ord("!"):
            if not image_mode and i + 1 < n and data[i + 1] == ord("["):
                image_mode = True
            clean.append(c)
        elif c == ord("\\"):
            if not escape_mode and i + 1 < n and data[i + 1] in (ord("["), ord("]")):
                escape_mode = True
            else:
                clean.append(c)
        else:
            clean.append(c)
        i += 1

    return clean.decode("utf-8"), spans


def cloze_hash(text: str, start: int, end: int) -> str:
    data = (
        b"Cloze"
        + text.encode("utf-8")
        + struct.pack("<Q", start)
        + struct.pack("<Q", end)
    )
    return blake3.blake3(data).hexdigest()


def cloze_family_hash(text: str) -> str:
    data = b"Cloze" + text.encode("utf-8")
    return blake3.blake3(data).hexdigest()


raw_text = (
    "Better is the sight of the eyes than the wandering of the\n"
    "desire: this is also vanity and vexation of spirit.\n\n"
    "— [Ecclesiastes] [6]:[9]"
)

clean_text, spans = extract_cloze_spans(raw_text)

family = cloze_family_hash(clean_text)

for start, end in spans:
    h = cloze_hash(clean_text, start, end)
    print(h)
