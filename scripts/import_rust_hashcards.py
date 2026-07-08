#!/usr/bin/env python3
"""
Import a legacy (Rust-format) hashcards.db into a running PocketBase-backed
hatchards server via its REST API.

Usage:
    python3 import_rust_hashcards.py hashcards.db http://127.0.0.1:3000 admin@mail.internal password
"""

import sqlite3
import sys

import requests


def authenticate(base_url, email, password):
    """Log in as superuser and return the auth token."""
    resp = requests.post(
        f"{base_url}/api/collections/_superusers/auth-with-password",
        json={"identity": email, "password": password},
    )
    resp.raise_for_status()
    return resp.json()["token"]


def find_record_id(base_url, headers, collection, filter_expr):
    """Return the id of the first record matching filter_expr, or None."""
    resp = requests.get(
        f"{base_url}/api/collections/{collection}/records",
        headers=headers,
        params={"filter": filter_expr, "perPage": 1},
    )
    resp.raise_for_status()
    items = resp.json()["items"]
    return items[0]["id"] if items else None


def upsert_record(base_url, headers, collection, filter_expr, data):
    """Create a record, or overwrite it if one already matches filter_expr."""
    existing_id = find_record_id(base_url, headers, collection, filter_expr)
    if existing_id:
        resp = requests.patch(
            f"{base_url}/api/collections/{collection}/records/{existing_id}",
            headers=headers,
            json=data,
        )
    else:
        resp = requests.post(
            f"{base_url}/api/collections/{collection}/records",
            headers=headers,
            json=data,
        )
    if resp.status_code >= 400:
        print(f"skip {collection} record {data}: {resp.text}", file=sys.stderr)
        return None
    return resp.json()["id"]


def import_cards(conn, base_url, headers):
    """Create or overwrite a bare 'new card' record for every card_hash in the legacy db."""
    rows = conn.execute("SELECT card_hash, added_at FROM cards").fetchall()
    for card_hash, added_at in rows:
        upsert_record(
            base_url,
            headers,
            "cards",
            f'card_hash = "{card_hash}"',
            {"card_hash": card_hash, "added_at": added_at, "review_count": 0},
        )


def import_sessions(conn, base_url, headers):
    """Import sessions and return a map of legacy session_id -> new PocketBase id."""
    rows = conn.execute(
        "SELECT session_id, started_at, ended_at FROM sessions"
    ).fetchall()
    session_map = {}
    for legacy_id, started_at, ended_at in rows:
        # Sessions have no unique key in the legacy schema, so a rerun would
        # duplicate them; match on started_at+ended_at to keep this idempotent.
        record_id = upsert_record(
            base_url,
            headers,
            "sessions",
            f'started_at = "{started_at}" && ended_at = "{ended_at}"',
            {"started_at": started_at, "ended_at": ended_at},
        )
        if record_id:
            session_map[legacy_id] = record_id
    return session_map


def import_reviews(conn, base_url, headers, session_map):
    """Import reviews in chronological order so cards end up in their final state.

    Each insert triggers the 'reviews' OnRecordCreate hook server-side, which
    copies stability/difficulty/interval/due_date onto the matching card.
    """
    rows = conn.execute(
        """
        SELECT session_id, card_hash, reviewed_at, grade, stability,
               difficulty, interval_raw, interval_days, due_date
        FROM reviews
        ORDER BY reviewed_at
        """
    ).fetchall()

    for (
        session_id,
        card_hash,
        reviewed_at,
        grade,
        stability,
        difficulty,
        interval_raw,
        interval_days,
        due_date,
    ) in rows:
        if session_id not in session_map:
            print(
                f"skip review {card_hash}@{reviewed_at}: unknown session {session_id}",
                file=sys.stderr,
            )
            continue
        # A review is uniquely identified by (card_hash, reviewed_at), so
        # matching on those keeps reruns idempotent instead of duplicating history.
        upsert_record(
            base_url,
            headers,
            "reviews",
            f'card_hash = "{card_hash}" && reviewed_at = "{reviewed_at}"',
            {
                "session_id": session_map[session_id],
                "card_hash": card_hash,
                "reviewed_at": reviewed_at,
                "grade": grade,
                "stability": stability,
                "difficulty": difficulty,
                "interval_raw": interval_raw,
                "interval_days": interval_days,
                "due_date": due_date,
            },
        )


def main():
    if len(sys.argv) != 5:
        print(
            "Usage: import_legacy.py <legacy.db> <base_url> <admin_email> <admin_password>"
        )
        sys.exit(1)

    legacy_path, base_url, email, password = sys.argv[1:5]

    token = authenticate(base_url, email, password)
    headers = {"Authorization": token}

    conn = sqlite3.connect(legacy_path)
    import_cards(conn, base_url, headers)
    session_map = import_sessions(conn, base_url, headers)
    import_reviews(conn, base_url, headers, session_map)
    conn.close()


if __name__ == "__main__":
    main()
