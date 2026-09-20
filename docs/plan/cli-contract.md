<!-- markdownlint-disable MD013 -->

# CLI 契約

## 位置づけ

この文書は、`skills-reconcile` の利用者向け interface について、承認済みの決定を置く場所である。
ここにない利用者向け挙動を実装者が補わない。

## 契約のリセット

旧ロードマップで定義していた `doctor`、`plan`、`add`、`remove`、`apply`、`prune`、`adopt`、
`migrate`、`capture` の契約と追加順序はすべて破棄する。

現在のソースコードに存在する最小 `--help` 挙動は、旧計画の途中状態であり、新製品の将来契約を
決める根拠にはしない。新しい一覧 command の契約が承認されるまでは、新しい利用者向け interface を
追加しない。

## 次に決める契約

### N01: `$HOME/.agents/skills` の一覧表示

#### 決定済み

- 一覧 command の名前は `list` とする。
- argument vector の先頭が `list` の場合に一覧処理を選択する。
- `list` より後ろの argument は、最初の実装では解釈も検証もしない。
- 一覧処理はファイルを一切変更しない。

#### 未決事項

実装前に、次のうち実装へ必要な挙動を順次決める。

- `$HOME/.agents/skills` の解決方法
- 一覧上で一つの Skill と認識する条件
- 通常ディレクトリ、symlink、壊れた symlink、`SKILL.md` がない entry の扱い
- 読み取り不能、対象ディレクトリなし、一覧が空の場合の扱い
- 表示する field と決定的な並び順
- stdout と stderr の使い分け
- 成功と失敗の終了 status
- text output の安定性と、machine-readable output を最初から提供するかどうか

ロードマップは上記の論点を列挙するだけで、答えを暗黙に決めない。N01 の契約が承認された後にだけ、
対応する CLI 実装へ進む。

## 後で決める契約

次の interface は、一覧表示の完成後まで設計しない。

- 複数の Skill 配置先を走査する interface
- TUI の検索、フィルタリング、選択操作
- コピー元とコピー先を指定する interface
- 衝突、上書き、更新、削除の扱い
- リポジトリから各 Agent へ反映する interface

command 名だけをロードマップへ先に置いて、grammar や副作用を実装者に推測させない。
