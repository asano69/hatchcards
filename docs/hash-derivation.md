ハッシュの導出アルゴリズムとシェルからの計算方法について説明します。

## ハッシュ導出アルゴリズム

hashcardsでは、カードのハッシュは**BLAKE3**アルゴリズムを使用してカードのコンテンツから計算されます 。ハッシュは32バイトの配列として定義されています

カードはコンテンツアドレス可能（content-addressable）であり、カードのテキストが変更されるとハッシュも変更され、レビュー履歴はリセットされます 。

カードのハッシュ導出はinternal/types/card.goのHash()メソッドで定義されています。

## Basic

Basicカードの場合、以下の3つのバイト列を単純に連結してBLAKE3でハッシュ化しています。

```
goh.Update([]byte("Basic"))
h.Update([]byte(c.Question))
h.Update([]byte(c.Answer))
```

つまり blake3("Basic" + question + answer) です（UTF-8バイト列として連結、区切り文字なし）


## Cloze

