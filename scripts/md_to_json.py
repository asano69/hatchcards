#!/usr/bin/env python3
import json
import os
import sys


def parse_text_to_json(text_data):
    # 改行コードを統一
    text_data = text_data.replace("\r\n", "\n").replace("\r", "\n")

    # [修正ポイント] 単純な空行分割ではなく、Q:, A:, C: の始まりを基準に分割します
    # これにより、ブロック内に空行が含まれていても正しく切り分けられます
    raw_blocks = text_data.split("\n")
    blocks = []
    current_block = []

    for line in raw_blocks:
        stripped = line.strip()
        # 新しい要素の始まり（Q:, A:, C:）を見つけたら、ここまでの塊を保存
        if (
            stripped.startswith("Q:")
            or stripped.startswith("A:")
            or stripped.startswith("C:")
        ):
            if current_block:
                blocks.append("\n".join(current_block).strip())
            current_block = [line]
        else:
            # 要素の続き（または空行）なら今の塊に追加
            if current_block or stripped:  # 最初の空行は無視
                current_block.append(line)

    if current_block:
        blocks.append("\n".join(current_block).strip())

    result = []
    current_q = None

    for block in blocks:
        # --- C:（コンセプト） ---
        if block.startswith("C:"):
            content = block[2:].strip()
            result.append({"kind": "cloze", "text": content})

        # --- Q:（質問） ---
        elif block.startswith("Q:"):
            current_q = block[2:].strip()

        # --- A:（回答） ---
        elif block.startswith("A:"):
            answer = block[2:].strip()
            if current_q:
                result.append({"kind": "basic", "question": current_q, "answer": answer})
                current_q = None

    return result


def main():
    if len(sys.argv) < 2:
        print(
            "使用方法: python txt_to_json.py <入力テキストファイルパス>",
            file=sys.stderr,
        )
        sys.exit(1)

    file_path = sys.argv[1]

    if not os.path.exists(file_path):
        print(f"エラー: ファイルが見つかりません: {file_path}", file=sys.stderr)
        sys.exit(1)

    try:
        with open(file_path, "r", encoding="utf-8") as f:
            raw_data = f.read()

        json_data = parse_text_to_json(raw_data)

        # json.dumps が自動的に適切な改行記号 (\n) に変換して出力してくれます
        print(json.dumps(json_data, indent=2, ensure_ascii=False))

    except Exception as e:
        print(f"エラー: 処理に失敗しました: {e}", file=sys.stderr)
        sys.exit(1)


if __name__ == "__main__":
    main()
