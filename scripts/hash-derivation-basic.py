#!/usr/bin/env python3
# uv run --with blake3 ./hash-derivation.py

import blake3

question = "谢谢"
answer = "ありがとう"

# "Basic" + question + answer を UTF-8 バイト列として単純連結（区切り文字なし）
data = b"Basic" + question.encode("utf-8") + answer.encode("utf-8")

h = blake3.blake3(data).hexdigest()

print(h)
