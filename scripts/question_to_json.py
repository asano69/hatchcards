#!/usr/bin/env python3
"""
Convert `## question` ... `---` blocks embedded in Markdown files into JSON decks.

A question block starts with a line like `## question` or `## question::DeckName`
(case-insensitive "question") and ends at the next line containing only `---`.
Using a heading + `---` terminator (instead of a code fence) means the
block's content can safely contain its own code fences.

Pipeline (small, single-purpose functions composed together):
    find_markdown_files
        -> extract_question_blocks   (per file: locate ## question ... --- blocks)
        -> parse_question_text       (per block: turn text into entries, log syntax issues)
        -> group_question_blocks     (accumulate entries per deck name, across all files)
        -> dedupe_entries        (drop exact duplicates within a deck)
        -> write_decks           (write <deck>.json to the output directory)

Usage:
    question_to_json.py <input_dir> <output_dir>
"""

import json
import logging
import re
import sys
from dataclasses import dataclass
from pathlib import Path

logger = logging.getLogger("question_to_json")

DEFAULT_DECK_NAME = "Default"

# Matches a question header, e.g. "## question", "## Question", or "## question::Test"
HEADER_RE = re.compile(r"^##\s+[Qq]uestion(?:::(\S+))?\s*$")

# Matches the terminator line that closes a question block: a line of only dashes.
TERMINATOR_RE = re.compile(r"^-{3,}\s*$")


@dataclass
class QuestionBlock:
    """A single ## question ... --- block extracted from a Markdown file."""

    deck_name: str
    content: str
    source_file: Path
    start_line: int  # 1-indexed line number of the header line


def sanitize_deck_name(name):
    """
    Sanitize a deck name coming from untrusted Markdown content.

    Strips path-separator-like characters so a crafted ## question::../../x
    block can't be used to write outside the output directory.
    """
    cleaned = re.sub(r'[\\/:*?"<>|]', "_", name).strip()
    return cleaned or DEFAULT_DECK_NAME


def find_markdown_files(input_dir):
    """Recursively find all Markdown files under input_dir, sorted for stable output."""
    return sorted(input_dir.rglob("*.md"))


def extract_question_blocks(text, source_file):
    """
    Scan a Markdown file's text for "## question" / "## question::Name" blocks.

    Each block runs from its header line to the next line containing only
    dashes (e.g. "---"). Unlike a code-fence-delimited block, this format
    is not broken by code fences (or anything else) inside the content.
    """
    lines = text.replace("\r\n", "\n").replace("\r", "\n").split("\n")
    blocks = []
    i = 0
    while i < len(lines):
        match = HEADER_RE.match(lines[i].strip())
        if not match:
            i += 1
            continue

        raw_name = match.group(1)
        deck_name = sanitize_deck_name(raw_name) if raw_name else DEFAULT_DECK_NAME
        start_line = i + 1  # 1-indexed, for logging

        content_lines = []
        i += 1
        closed = False
        while i < len(lines):
            if TERMINATOR_RE.match(lines[i].strip()):
                closed = True
                break
            content_lines.append(lines[i])
            i += 1

        if not closed:
            logger.warning(
                "%s:%d: question block '%s' has no closing '---'; "
                "rest of file after this point is ignored",
                source_file,
                start_line,
                deck_name,
            )
            break

        blocks.append(
            QuestionBlock(
                deck_name=deck_name,
                content="\n".join(content_lines),
                source_file=source_file,
                start_line=start_line,
            )
        )
        i += 1  # move past the terminator line

    return blocks


def parse_question_text(text_data, context):
    """
    Parse the text content of one question block into JSON-ready entries.

    Recognizes three line prefixes:
        C:  a standalone concept entry
        Q:  the start of a question (must be followed by a matching A:)
        A:  the answer to the preceding Q:

    A block may span multiple (including blank) lines; a new block starts
    only at the next Q:/A:/C: prefix.

    Returns (entries, warnings). Warnings describe any broken syntax found
    (e.g. a Q: without a matching A:, or an unrecognized line) so callers
    can log them without silently dropping data.
    """
    text_data = text_data.replace("\r\n", "\n").replace("\r", "\n")
    raw_lines = text_data.split("\n")

    blocks = []
    current_block = []
    for line in raw_lines:
        stripped = line.strip()
        if (
            stripped.startswith("Q:")
            or stripped.startswith("A:")
            or stripped.startswith("C:")
        ):
            if current_block:
                blocks.append("\n".join(current_block).strip())
            current_block = [line]
        else:
            if current_block or stripped:  # ignore leading blank lines
                current_block.append(line)
    if current_block:
        blocks.append("\n".join(current_block).strip())

    entries = []
    warnings = []
    current_q = None
    current_q_pos = None

    for pos, block in enumerate(blocks):
        if block.startswith("C:"):
            if current_q is not None:
                warnings.append(
                    f"{context}: 'Q:' (entry {current_q_pos}) has no matching 'A:' before a 'C:' entry"
                )
                current_q = None
            content = block[2:].strip()
            if not content:
                warnings.append(f"{context}: empty 'C:' entry (entry {pos})")
            entries.append({"kind": "cloze", "text": content})

        elif block.startswith("Q:"):
            if current_q is not None:
                warnings.append(
                    f"{context}: 'Q:' (entry {current_q_pos}) has no matching 'A:'"
                )
            current_q = block[2:].strip()
            current_q_pos = pos
            if not current_q:
                warnings.append(f"{context}: empty 'Q:' entry (entry {pos})")

        elif block.startswith("A:"):
            if current_q is None:
                warnings.append(
                    f"{context}: 'A:' (entry {pos}) has no preceding 'Q:'; ignored"
                )
                continue
            answer = block[2:].strip()
            entries.append({"kind": "basic", "question": current_q, "answer": answer})
            current_q = None
            current_q_pos = None

        else:
            warnings.append(
                f"{context}: unrecognized content (entry {pos}): {block[:50]!r}"
            )

    if current_q is not None:
        warnings.append(
            f"{context}: trailing 'Q:' (entry {current_q_pos}) has no matching 'A:'"
        )

    return entries, warnings


def group_question_blocks(md_files):
    """Extract and parse every question block across all files, grouped by deck name."""
    decks = {}
    for md_file in md_files:
        try:
            text = md_file.read_text(encoding="utf-8")
        except OSError as e:
            logger.error("Failed to read %s: %s", md_file, e)
            continue

        for block in extract_question_blocks(text, md_file):
            context = (
                f"{block.source_file}:{block.start_line} (question:{block.deck_name})"
            )
            entries, warnings = parse_question_text(block.content, context)
            for warning in warnings:
                logger.warning(warning)
            decks.setdefault(block.deck_name, []).extend(entries)
    return decks


def dedupe_entries(entries):
    """Remove exact duplicate entries (same type and same field values), keeping first occurrence."""
    seen = set()
    unique = []
    for entry in entries:
        key = json.dumps(entry, sort_keys=True, ensure_ascii=False)
        if key in seen:
            continue
        seen.add(key)
        unique.append(entry)
    return unique


def write_decks(decks, output_dir):
    """Write each deck's deduplicated entries to <output_dir>/<deck_name>.json."""
    output_dir.mkdir(parents=True, exist_ok=True)
    for deck_name, entries in decks.items():
        unique_entries = dedupe_entries(entries)
        out_path = output_dir / f"{deck_name}.json"
        with out_path.open("w", encoding="utf-8") as f:
            json.dump(unique_entries, f, indent=2, ensure_ascii=False)
        logger.info("Wrote %d entries to %s", len(unique_entries), out_path)


def main():
    logging.basicConfig(level=logging.INFO, format="%(levelname)s: %(message)s")

    if len(sys.argv) != 3:
        print("Usage: question_to_json.py <input_dir> <output_dir>", file=sys.stderr)
        sys.exit(1)

    input_dir = Path(sys.argv[1])
    output_dir = Path(sys.argv[2])

    if not input_dir.is_dir():
        print(f"Error: input directory not found: {input_dir}", file=sys.stderr)
        sys.exit(1)

    md_files = find_markdown_files(input_dir)
    decks = group_question_blocks(md_files)
    write_decks(decks, output_dir)


if __name__ == "__main__":
    main()
