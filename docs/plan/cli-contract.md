<!-- markdownlint-disable MD013 -->

# CLI 契約

## 位置づけ

この文書は、`skills-reconcile` の利用者向け interface について、承認済みの決定を置く場所である。
ここにない利用者向け挙動を実装者が補わない。

## 契約のリセット

旧ロードマップで定義していた `doctor`、`plan`、`add`、`remove`、`apply`、`prune`、`adopt`、
`migrate`、`capture` の契約と追加順序はすべて破棄する。

現在のソースコードに存在する最小 `--help` 挙動は、旧計画の途中状態であり、新製品の将来契約を
決める根拠にはしない。新しい利用者向け interface は、以下で承認した一覧 command に限定する。

## 承認済みの契約

### N01: `$HOME/.agents/skills` の一覧表示

#### Command の選択

- 一覧 command の名前は `list` とする。
- argument vector の先頭が `list` の場合に一覧処理を選択する。
- `list` より後ろの argument は、最初の実装では解釈も検証もしない。
- argument がない場合と、先頭の argument が `list` または単独の `--help` ではない場合は失敗する。
- `--help` は単独で指定された場合だけ成功する。`list --help` は一覧処理を選択し、`--help` を解釈しない。

#### Scan root の解決

- process 起動時の `HOME` 環境変数を利用者の home directory とする。
- `HOME` は空でない絶対 path でなければならない。未設定、空、相対 path の場合は失敗する。
- scan root は `HOME` に `.agents/skills` を連結した path とする。
- 設定ファイル、XDG directory、現在の working directory、外部 CLI から scan root を補完しない。
- `HOME` または scan root 自体が symlink を含む場合は、OS の通常の path 解決に従う。
- scan root が存在しない、directory ではない、読み取れない場合は失敗する。

#### Skill の認識

- scan root の直下だけを走査し、再帰的に探索しない。
- root 直下の entry 名は 1 文字以上 64 文字以下とし、ASCII 小文字 `a-z`、数字 `0-9`、
  hyphen `-` だけを認める。hyphen は先頭と末尾に置かず、連続させない。
- この名前条件を満たさない entry は Skill として扱わず、entry の種類、symlink target、
  `SKILL.md` の状態を調べる前に認識対象から外す。その entry の状態は一覧処理を失敗させない。
- 名前条件を満たす root 直下の entry が directory であり、その直下に symlink ではない
  通常ファイルの `SKILL.md` が存在する場合、その entry を一つの Skill と認識する。
- `SKILL.md` の内容、frontmatter 内の `name`、空かどうかは検証しない。空の `SKILL.md` も認識条件を満たす。
- 通常ファイル、特殊ファイル、directory 以外を指す symlink は Skill として扱わない。
- `SKILL.md` が存在しない directory、または `SKILL.md` が通常ファイルではない directory は Skill として
  扱わない。

#### Symlink と異常 entry

- root 直下の名前条件を満たす symlink が directory を指し、その directory が Skill の認識条件を
  満たす場合は Skill として扱う。名前条件と一覧表示には symlink target の名前ではなく、root 直下の
  entry 名を使用する。
- root 直下の symlink target が scan root 外にあっても、一覧のための読み取りに限って認識対象にする。
  この決定は、将来のコピー処理で scan root 外の target を許可する根拠にしない。
- root 直下の名前条件を満たす entry に、壊れた symlink、symlink loop、entry の種類または
  `SKILL.md` の状態を確認できない読み取り error が一つでもある場合は、一覧処理全体を失敗させる。
- discovery が失敗した場合は、確認済みの Skill を部分結果として stdout に出力しない。

#### 成功時の出力

- 表示 field は root 直下の entry 名だけとし、絶対 path と symlink target は表示しない。
- entry 名を byte 列として昇順に並べ、それぞれを変更せず一行ずつ stdout に出力する。
- scan root が存在して読み取り可能だが、認識条件を満たす Skill がない場合は成功し、stdout には何も
  出力しない。scan root が空の場合と、認識対象外の entry だけがある場合を同じ結果として扱う。
- 成功時の終了 status は `0` とし、stderr には何も出力しない。
- 上記の text output を最初の安定 interface とする。machine-readable output と出力形式を切り替える
  option は提供しない。

#### 失敗時の出力

- 利用者が対処できる診断を stderr に出力し、終了 status は `1` とする。
- discovery の失敗では stdout に何も出力しない。
- 診断文の具体的な wording は安定 interface としない。

#### Help

単独の `--help` は、次の text を stdout に出力して成功する。

```text
Usage:
  skills-reconcile list
  skills-reconcile --help

Commands:
  list    List Skills in $HOME/.agents/skills.
```

#### 副作用

- 一覧処理は Skill、設定、lock、state、Git repository を作成、変更、移動、削除しない。
- symlink target が scan root 外にある場合も、root 内外を問わずファイルへ書き込まない。
- 外部 CLI や network を使用しない。

## 後で決める契約

次の interface は、一覧表示の完成後まで設計しない。

- 複数の Skill 配置先を走査する interface
- TUI の検索、フィルタリング、選択操作
- コピー元とコピー先を指定する interface
- 衝突、上書き、更新、削除の扱い
- リポジトリから各 Agent へ反映する interface

command 名だけをロードマップへ先に置いて、grammar や副作用を実装者に推測させない。
