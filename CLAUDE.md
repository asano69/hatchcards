## Overview

hashcards is a plain-text spaced repetition system written in Rust. It parses Markdown files containing flashcards, stores performance data in SQLite, and presents cards through a web interface using the FSRS algorithm for scheduling.


# Design and Internals

- Cards are content addressed.
- Media files are referenced in markdown using standard image syntax: `![](path/to/file.ext)`. Standard image and AV formats are supported.

# Rules

- When fixing bugs, add a failing regression test first.
- All errors are user-facing, so messages should be clear.
- Keep functions small and focused.
- Module files should re-export what's needed, hide implementation details.
- Don't persist changes to the database during drilling. Use the cache.
- Don't use timezones: dates are naive for a reason. Due dates etc. are more like the dates in a journal entry than precise points in time.


# Work in progress

- SSRからSPA/CSRに段階的に移行中。これまで、go-templateのみのアーキテクチャだったがPocketBaseやSolid.jsをつかう設計にかえた。その変更に伴い、PocketBaseが提供する仕組みをつかってDBを操作するように書き換えが必要になっている。

