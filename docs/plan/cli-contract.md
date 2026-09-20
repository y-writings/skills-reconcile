<!-- markdownlint-disable MD013 -->

# CLI 契約

## 位置づけ

この文書は、`skills-reconcile` の利用者向け interface を実装する前に確定する契約の置き場所である。
完全な grammar、終了 status、stdout と stderr、副作用、未対応入力がここで承認されるまで、対応する
command を実装しない。

## 契約のリセット

旧ロードマップで定義していた `doctor`、`plan`、`add`、`remove`、`apply`、`prune`、`adopt`、
`migrate`、`capture` の契約と追加順序はすべて破棄する。

現在のソースコードに存在する最小 `--help` 挙動は、旧計画の途中状態であり、新製品の将来契約を
決める根拠にはしない。新しい一覧 command の契約が承認されるまでは、新しい利用者向け interface を
追加しない。

## 次に決める契約

### N01: `$HOME/.agents/skills` の一覧表示

最初の実装 PR より前に、文書だけの変更で次を確定する。

- command 名と完全な argument grammar
- `$HOME/.agents/skills` の解決方法
- 一覧上で一つの Skill と認識する条件
- 通常ディレクトリ、symlink、壊れた symlink、`SKILL.md` がない entry の扱い
- 読み取り不能、対象ディレクトリなし、一覧が空の場合の扱い
- 表示する field と決定的な並び順
- stdout と stderr の使い分け
- 成功と失敗の終了 status
- text output の安定性と、machine-readable output を最初から提供するかどうか
- 未対応の flag、positional argument、環境変数
- ファイルを一切変更しないこと

ロードマップは上記の論点を列挙するだけで、答えを暗黙に決めない。N01 の exact head が承認された後に
だけ、対応する CLI 実装へ進む。

## 後で決める契約

次の interface は、一覧表示の完成後まで設計しない。

- 複数の Skill 配置先を走査する interface
- TUI の検索、フィルタリング、選択操作
- コピー元とコピー先を指定する interface
- 衝突、上書き、更新、削除の扱い
- リポジトリから各 Agent へ反映する interface

command 名だけをロードマップへ先に置いて、grammar や副作用を実装者に推測させない。
