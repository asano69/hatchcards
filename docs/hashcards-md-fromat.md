## Format

This section describes the text format used by hashcards.

### Basic Cards

Question-answer flashcards are written like this:

```
Q: What are the possible values of electric charge?
A: Any integer multiple of the fundamental charge.
```

Both the question and the answer can span multiple lines:

```
Q: List the elements of the Platinum group.
A:

- ruthenium
- rhodium
- palladium
- osmium
- iridium
- platinum
```

### Cloze Cards

Cloze cards start with the `C:` tag, and use square brackets to denote cloze
deletions:

```
C: The [order] of a group is [the cardinality of its underlying set].
```

Again, cloze cards can span multiple lines:

```
C:
Better is the sight of the eyes than the wandering of the
desire: this is also vanity and vexation of spirit.

— [Ecclesiastes] [6]:[9]
```

### Separators

Optionally, cards can be separated by horizontal rules, like so:

```
C: A semigroup with an identity element is called a [monoid].

---

C: A semigroup without associativity is called a [magma].

---

C: A magma where the operation is [associative] is called a [semigroup].
```

This can help visually separate the cards better.

## Features

This section documents specific hashcards features.

### LaTeX Support

Cards support LaTeX math via KaTeX.

Use `$...$` for inline math:

```
Q: What is the combinatorial meaning of $\binom{n}{k}$?
A: From a set of size $n$, we can choose $\binom{n}{k}$ subsets of size $k$.
```

And `$$...$$` for display math:

```
C: The [amount of substance] of a sample, denoted $n$, is defined as:

$$
n = \frac{N}{N_A}
$$

where $N$ is [the number of elementary entities] and $N_A$ is [Avogadro's constant].
```

You can define custom LaTeX macros by creating a `macros.tex` file in your
collection root:

```
\C \mathbb{C}
\R \mathbb{R}
```

Macro definitions can refer to arguments: `#1` for the first, `#2` for the
second and so on.

### Images

Ordinary Markdown image syntax works:

```
Q: Identify this painting:

![](art/diagram.png)

A: _The Siren_, by John William Waterhouse.
```

By default, image paths are resolved relative to the deck (the Markdown file)
that contains the flashcard. For example, if you have:

```
cards/
  Art Theory/
    Art.md
    Images/
      TheMermaid.jpg
      Circe.jpg
      Odysseus.jpg
```

Then flashcards in `Art.md` can reference images with paths like
`Images/Circe.jpg`.

By prefixing a path with `@/`, you can point to images relative to the
collection root directory, e.g., a path like `@/Art Theory/Images/Circe.jpg`
will always resolve to the same path, even if the deck is moved around within
the collection.

### Audio

Works like images:

```
Q: How do you pronounce "پرنده" in Persian?
A: ![](audio/parande.mp3)
```

### Deck Names

By default, the filename of a deck is the name of a deck, e.g. a file
`Medicine.md` will be parsed as a deck called `Medicine`. It is possible to
override the name using [TOML](https://toml.io/en/) frontmatter, like so:

```
---
name = "Medicine"
---

C: The mitochondria is the [powerhouse] of the cell.
```

Regardless of the filename, cards in this deck will have `Medicine` as their
deck name. This is particularly useful when you want to organize a large number
of cards into different files, while keeping their deck name the same. For
example, when taking notes from a textbook, you might have something like so:

```
Principles of Neural Science/
  Ch1.md
  Ch2.md
  ...
```

But you don't want the cards in those Markdown files to have `Ch1`, `Ch2`, etc.
as their deck name. TOML frontmatter allows you to give each chapter deck the same
deck name.

### Sibling Burial

A single cloze card in the Markdown text with _n_ cloze deletions corresponds to _n_ distinct cloze cards in the database, one per deletion. These cards are called "siblings". 

Hashcards supports "sibling burial": by default, within a session, only one sibling in a particular sibling group will be shown. This is to prevent the text of one card spoiling the answer of another card. The idea is you might do multiple sessions in a single day, and each session shows a different sibling, until you run out of siblings for all cards due today.

You can turn this off by passing `--bury-siblings=false` to the `drill` command.