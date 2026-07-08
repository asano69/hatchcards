# Hatchards

[![Ask DeepWiki](https://deepwiki.com/badge.svg)](https://deepwiki.com/asano69/hatchards)

## Introduction 🐝

<img src="frontend/public/favicon.svg" width="100" align="right" />

A flashcard should always be stored together with the context from which it was derived. (Context Preservation Principle) Flashcards are generated automatically, with Markdown notebooks serving as their sole source of truth. Users only need to focus on building and refining a knowledge base composed of Markdown files.

This application is programmable and automates flashcard generation from Q&A lists in arbitrary formats through JSON intermediate files. It stores performance data in SQLite3 and supports efficient spaced repetition through scheduling based on the FSRS algorithm.



### Features
- **Content Addressable:** cards are identified by the hash of their text. This
  means a card's progress is reset when the card is edited.
- **Efficient:** uses [FSRS] for scheduling reviews, maximizing learning while
  minimizing time spent reviewing.
- **Visibility**: Provides a detailed view of the cards' current status and offers clear visualization of the learning schedule. 
- **Git mirroring**: Because hooks are fully programmable, flashcards can, in principle, be generated automatically from any repository. Repositories containing structured Q&A data can therefore be transformed into hashcrds decks.



### Tech Stack
- Go
- [SolidJS](https://github.com/solidjs/solid)
- [PocketBase](https://github.com/pocketbase/pocketbase)
- [Vega-Lite](https://github.com/vega/vega-lite)


## Workflow 🍯
1. In a Markdown editor such as Obsidian, take notes on what you have understood about topics that interest you.
2. In question sections, write the information you want to memorize as a list of Q&A pairs. (e.g. `## Question::DeckName {{Q&A}} ---`)
3. Push the Markdown notebooks to a remote repository.
4. If the repository mirroring settings are configured correctly, flashcards will be generated automatically.

## Screenshot

>[!WARNING]
>This app is still under development.

<img src=".github/assets/sample-01.png" width="300" /><img src=".github/assets/sample-02.png" width="300" />

## Example

The following JSON file is a valid hatchards deck:

```json
[
  { "deckName": "Neurophysiology",
    "kind": "basic",
    "question": "How many neurons are there in the human brain?",
    "answer": "~80 billion."
  },
  {
    "deckName": "Neurophysiology",
    "kind": "cloze",
    "text": "An [agonist] is a ligand that binds to a receptor and [activates it]."
  },
  {
    "deckName": "Neurophysiology",
    "kind": "basic",
    "question": "How many synapses are there in a human brain?",
    "answer": "~100 trillion"
  },
  {
    "deckName": "Neurophysiology",
    "kind": "cloze",
    "text": "In the nervous system, [chemical] communication happens [between] neurons."
  }
]
```
Please use the script of your choice to convert your flashcards into this format. A sample Python script is provided in the scripts directory.

## Tutorial

>[!CAUTION]
>While the databases for the Rust and Go port versions of hashcards are compatible, you cannot migrate by simply copying the file. The Go port version includes additional management ID columns used by PocketBase, as well as view tables for statistics. To migrate from the Rust version, you need to use a script to insert the data via the API.

```sh
# On a running server
$ uv run --with requests import_rust_hashcards.py ~/data/hashcards.db http://127.0.0.1:3000 admin@mail.internal password
```

Create a directory for your flashcards, and add a JOSN file with some cards:

```bash
$ mkdir -p cards
$ cd cards
$ cat > example-notebook-1970-01.json << 'EOF'
[
  {
    "deckName": "Geography",
    "kind": "basic",
    "question": "What is Coulomb's constant?",
    "answer": "The proportionality constant of the electric force."
  },
  {
    "deckName": "Geography",
    "kind": "basic",
    "question": "What is an object with zero net charge called?",
    "answer": "Neutral."
  }
]
EOF
```

The name of the JSON file can be anything you like, but it is generally easier to manage if you create a separate file for each resource in the Markdown notebook from which the Q&A flashcards are generated. Note that the value of the `deckName` key in the JSON object is used as the playlist name displayed when reviewing the flashcards.

Start drilling:

```bash
$ hatchards serve
```

This opens a web interface at `http://localhost:3000` where you can review your
cards. The interface is simple: you read the question, mentally recall the
answer, and click reveal (or press space). Then you grade yourself on how you
did, with one of four choices:

1. Forgot (shortcut: `1`)
2. Hard (shortcut: `2`)
3. Good (shortcut: `3`)
4. Easy (shortcut: `4`)

Be honest. If you got the answer almost right, press "Forgot". If you mis-grade
something, you can undo (shortcut: `u`). The session ends when every card has
been graded "Good" or higher. You can end the session prematurely by clicking
"End", this will save your changes.



## Commands

This section documents the hatchards command line interface.

### `serve`

Start a drilling session.

```bash
$ hatchards serve --dir [DIRECTORY]
```

>[!NOTE]
>- Note: your progress is not saved until the session ends, either when you run out of cards, or when you click "End".
>- Settings such as the card location can be changed using environment variables.


### `stats`

Print collection statistics to standard output.

```bash
$ hatchards stats [DIRECTORY]
```

### `check`

Check the integrity of a collection.

```bash
$ hatchards check [DIRECTORY]
```

### `orphans`

Manage orphan cards (cards that exist in the database, but not in the
collection, i.e., cards that were deleted from the collection).

```bash
$ hatchards orphans list [DIRECTORY]
$ hatchards orphans delete [DIRECTORY]
```

Example:

```
$ hatchards orphans list Cards
04effc035b71692b66a90a622559479516526e7720c41afa22b29562915d58af
059e4e0fd5c3d0ab7ef0cc902cdc402a555ec4152b842fe584109de6c8082ce3
061b8c27e0f437d0c6ae735e829b39cc3bf0ad8218cb16387dcb4271c20b244d
$ hatchards orphans delete Cards
04effc035b71692b66a90a622559479516526e7720c41afa22b29562915d58af
059e4e0fd5c3d0ab7ef0cc902cdc402a555ec4152b842fe584109de6c8082ce3
061b8c27e0f437d0c6ae735e829b39cc3bf0ad8218cb16387dcb4271c20b244d
$ hatchards orphans list Cards
# no output
```

## Environment Variables

`hatchards serve` reads its configuration from environment variables. All are optional; defaults are shown below.

| Variable              | Default   | Description                                                                 |
|------------------------|-----------|------------------------------------------------------------------------------|
| `SERVER_HOST`          | `0.0.0.0` | Host address the HTTP server binds to.                                       |
| `SERVER_PORT`          | `3000`    | Port the HTTP server listens on.                                             |
| `DATA_ROOT`            | `.`       | Path to the collection directory (where your deck `.json` files live).       |
| `FSRS_TARGET_RECALL`   | `0.9`     | Target recall probability (0–1) used to compute review intervals.            |
| `FSRS_MIN_INTERVAL`    | `1.0`     | Minimum interval (in days) between reviews.                                  |
| `FSRS_MAX_INTERVAL`    | `256.0`   | Maximum interval (in days) between reviews.                                  |

Example:

```sh
SERVER_HOST=0.0.0.0
SERVER_PORT=3000
DATA_ROOT=./cards
FSRS_TARGET_RECALL=0.9
FSRS_MIN_INTERVAL=1.0
FSRS_MAX_INTERVAL=256.0
```

These can be set directly, via `.envrc`/`direnv`, or (as in `hashacrds.env` and `compose.yaml`) loaded from a file / passed to the Docker container.

## Database

hatchards stores card performance data and review history in the SQLite3 database managed by PocketBase.

The `cards` table has the following schema:

| Column | Type | Description |
| :--- | :--- | :--- |
| **`id`** | **`text primary key`** | **`Randomly generated unique ID (e.g., 'r...').`** |
| `card_hash` | ~~`text primary key`~~ **`text not null default ''`** | The hash of the card. **`Has a UNIQUE index constraint.`** |
| `added_at` | `text not null` | The timestamp when the card was first added to the database, in timestamp format. |
| `last_reviewed_at` | ~~`text`~~ **`text not null default ''`** | The timestamp when the card was most recently reviewed. ~~`null`~~ **`Empty string`** if the card is new. |
| `stability` | ~~`real`~~ **`numeric not null default 0`** | The card's stability. ~~`null`~~ **`0`** if the card is new. |
| `difficulty` | ~~`real`~~ **`numeric not null default 0`** | The card's difficulty. ~~`null`~~ **`0`** if the card is new. |
| `interval_raw` | ~~`real`~~ **`numeric not null default 0`** | The FSRS-calculated interval, before rounding and clamping. A real number of days until the next review. ~~`null`~~ **`0`** if the card is new. |
| `interval_days` | ~~`real`~~ **`numeric not null default 0`** | The interval as an integer number of days, after rounding and clamping. ~~`null`~~ **`0`** if the card is new. |
| `due_date` | ~~`text`~~ **`text not null default ''`** | The date when the card is next due, in `YYYY-MM-DD` format. ~~`null`~~ **`Empty string`** if the card is new. |
| `review_count` | `integer not null` | The number of times the card has been reviewed. |

The `sessions` table has the following schema:

| Column | Type | Description |
| :--- | :--- | :--- |
| ~~`session_id`~~ **`id`** | ~~`integer primary key`~~ **`text primary key default ('r'\|\|lower(hex(randomblob(7))))`** | The ID of the session. |
| `started_at` | ~~`text not null`~~ **`text not null default ''`** | The timestamp when the session started, in timestamp format. **`Has an index constraint.`** |
| `ended_at` | ~~`text not null`~~ **`text not null default ''`** | The timestamp when the session ended, in timestamp format. |

The `reviews` table has the following schema:

| Column | Type | Description |
| :--- | :--- | :--- |
| **`id`** | **`text primary key default ('r'\|\|lower(hex(randomblob(7))))`** | **`The review ID. Randomly generated unique ID.`** |
| ~~`review_id`~~ | ~~`integer primary key`~~ | *(This column was replaced by `id`)* |
| `session_id` | ~~`integer not null`~~ **`text not null default ''`** | The ID of the session this review was performed in, a foreign key. **`Has an index constraint.`** |
| `card_hash` | `text not null` **`default ''`** | The hash of the card that was reviewed, a foreign key. **`Has an index constraint.`** |
| `reviewed_at` | ~~`text not null`~~ **`text not null default ''`** | The timestamp when the review was performed (i.e., when the user submitted a grade). |
| `grade` | ~~`text not null`~~ **`text not null default ''`** | One of `forgot`, `hard`, `good`, or `easy`. |
| `stability` | ~~`real not null`~~ **`numeric not null default 0`** | The card's stability after this review. |
| `difficulty` | ~~`real not null`~~ **`numeric not null default 0`** | The card's difficulty after this review. |
| `interval_raw` | ~~`real`~~ **`numeric not null default 0`** | The FSRS-calculated interval, before rounding and clamping. A real number of days until the next review. ~~`null`~~ **`0`** if the card is new. |
| `interval_days` | ~~`real`~~ **`numeric not null default 0`** | The interval as an integer number of days, after rounding and clamping. ~~`null`~~ **`0`** if the card is new. |
| `due_date` | ~~`text not null`~~ **`text not null default ''`** | The date, in the user's local time, when the card is next due, in `YYYY-MM-DD` format. |

>[!NOTE]
>- "timestamp format" is `YYYY-MM-DDTHH:MM:SS.MMM`, e.g. `2025-10-04T17:09:51.517`.



## Prior Art

- [eudoxia0/hashcards](https://github.com/eudoxia0/hashcards)
- [org-fc](https://github.com/l3kn/org-fc)
- [org-drill](https://orgmode.org/worg/org-contrib/org-drill.html)
- [hascard](https://hackage.haskell.org/package/hascard)
- [carddown](https://github.com/martintrojer/carddown)
- [My implementation of a personal mnemonic medium](https://notes.andymatuschak.org/My_implementation_of_a_personal_mnemonic_medium)

[FSRS]: https://github.com/open-spaced-repetition/fsrs4anki
[blog]: https://borretti.me/article/hashcards-plain-text-spaced-repetition
[cargo]: https://doc.rust-lang.org/cargo/
[esr]: https://borretti.me/article/effective-spaced-repetition
[fc]: https://github.com/eudoxia0/flashcards
[rustup]: https://rustup.rs/


## License

© 2026- by asano69. Licensed under the Apache 2.0 license.  
© 2025–2026 by Fernando Borretti. Licensed under the Apache 2.0 license.  

---


To learn how to write good flashcards, read [Effective Spaced Repetition][esr].  
=> https://gutenberg.org/cache/epub/47748/pg47748-images.html  
=> https://archive.org/details/reasonwhynathist00philrich/page/n5/mode/2up  





