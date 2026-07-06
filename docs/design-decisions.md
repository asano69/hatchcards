### 設計上の決定（Design Decisions）

#### なぜカードをデータベースに保存しないのか？

従来のフラッシュカードアプリでは、すべての内容をデータベースに保存するのが一般的ですが、これにはいくつかの問題があります。

* ベンダーロックイン — データの移行や再利用が難しい
* 不透明な形式 — 標準的なツールで扱えない
* バージョン管理が困難 — Git と連携しづらい
* 共有しにくい — ファイル単体で簡単に受け渡せない

hashcards は、カードを Markdown ファイルとして保存することでこれらの問題を解決しています。SQLite に保存されるのは、各カードの次回復習日時などの**スケジューリング状態**のみで、これらは必要に応じて再構築可能な一時的データです。 ([Fernando Borretti][1])

#### なぜコンテンツアドレッシング（Content-addressable）を採用しているのか？

カードはデータベースの ID ではなく、内容のハッシュ値によって識別されます。これにより次の利点があります。

* 自動重複排除 — 同じ内容なら同じカードになる
* 安全な編集 — カードを変更すると新しいカードとして扱われる
* Git との親和性 — 内容の変更履歴を追跡しやすい
* ID の衝突がない — マージ時の競合は本質的に内容の競合になる

#### なぜ SM-2 ではなく FSRS を採用しているのか？

FSRS（Free Spaced Repetition Scheduler）は、現在最先端の間隔反復学習アルゴリズムの一つです。

* SM-2 より正確に記憶の保持率を予測できる
* 忘れてしまったカードへの対応が優れている
* カードごとの難易度に適応できる
* 現在の Anki でも採用されている

単純な「復習間隔を倍増させる」方式とは異なり、FSRS は人間の記憶の忘却曲線をモデル化してスケジュールを決定します。 ([hashcards.app][2])

[1]: https://borretti.me/article/hashcards-plain-text-spaced-repetition?utm_source=chatgpt.com "Hashcards: A Plain-Text Spaced Repetition System"
[2]: https://hashcards.app/blog/how-hashcards-works/?utm_source=chatgpt.com "Building Hashcards Mobile App | Hashcards Blog"
