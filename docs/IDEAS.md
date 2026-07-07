## Card Stages

Like in Anki. New, learning, mature, with a state machine.

Arguably, the first time a card is reviewed, "forgetting" should not adjust the
FSRS parameters.

See:

- <https://docs.ankiweb.net/getting-started.html#card-states>


## 可視性

- 過去から現在のデッキごとの安定性のグラフ
- 現在の、カードステータス（期限切れ、新規カード、期日カード）
- 未来の、確定された期限
- “Statistics - Anki Manual”. docs.ankiweb.net, [https://docs.ankiweb.net/stats.html](https://docs.ankiweb.net/stats.html), (Accessed 2026-07-04)

![](https://docs.ankiweb.net/media/Statistics.png)

## Term-Definition Cards

A shorthand. Writing:

```
T: lithification
D: The process of turning loose sediment into rock.
```

Is equivalent to writing this:

```
Q: Define: lithification
A: The process of turning loose sediment into rock.

Q: Term for: The process of turning loose sediment into rock.
A: lithification
```

## Preview Command

Right now the only way to see how a card renders is to run the `drill` command
and hope you see it first. Instead, there should be a `preview` command that
opens a web interface that lets you navigate the flashcards, either all of them,
or one deck at a time, and see how they render.



