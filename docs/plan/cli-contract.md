<!-- markdownlint-disable MD013 -->

# CLI契約

## 位置づけ

この文書は `skills-reconcile` の利用者向けinterfaceの正本である。ここへ明記され、先行する文書PRで
承認されたentryだけを実装できる。PRロードマップにあるcommand名は実装範囲を示すlabelであり、
grammar、flag、終了status、出力または副作用を定義しない。

CLI entryを追加または変更するPRは文書だけで作成し、対応する実装PRより先にmergeする。explicitな
stackでは、契約PRを実装PRの直接のbaseにできる。契約と実装を同じPRで新規決定しない。

各entryは、少なくとも次をすべて定義する。

- 完全なcommand grammarとpositional argument
- global flagとsubcommand flag、そのdefault、排他条件
- 成功、差分あり、入力不正、競合、外部process失敗の終了status
- stdoutとstderrの使い分け、およびtext outputの安定性
- machine-readable outputのschema、順序、nullability、未知field方針
- filesystemと外部processの副作用、確認条件、dry-runの意味
- 未対応の入力と、互換性を持たせない入力

entryにないcommand、flag、argument、status、fieldを実装者が補ってはならない。必要になった場合は、
対応する契約entryを先に承認する。

## 現在承認済みのinterface

現時点で承認済みなのは、フェーズ1で実装した最小CLIだけである。

### `skills-reconcile --help`

- argument vectorが正確に `--help` 一つの場合だけ成功する。
- 終了statusは0、stderrは空とし、stdoutへ次を出力する。

```text
Usage: skills-reconcile --help

No commands are available in this migration phase.
```

### 引数なし

- 終了statusは1、stdoutは空とする。
- stderrへ `skills-reconcile: usage: skills-reconcile --help` と改行を出力する。

### その他の入力

- `--version`、将来command名、`--help`の後続argumentを含め、その他のargument vectorは未対応とする。
- 終了statusは1、stdoutは空とする。
- stderrへ `skills-reconcile: unsupported command or option %q` の `%q` を先頭argumentのGo quoted stringで
  置換し、改行を付けて出力する。

## 契約を追加する順序

| ID  | 対象interface                   | 実装前の承認期限 | 必須内容                                                      |
| --- | ------------------------------- | ---------------- | ------------------------------------------------------------- |
| K01 | `doctor`                        | C07より前        | grammar、全flag、status、診断code、text/JSON schema           |
| K02 | `plan`                          | R05より前        | grammar、全flag、status、action/status集合、text/JSON schema  |
| K03 | `add`、`remove`                 | M02より前        | grammar、全flag、status、dry-run、manifest更新、text/JSON出力 |
| K04 | `apply`、`apply --prune`        | A01より前        | grammar、全flag、status、確認条件、外部process、副作用、出力  |
| K05 | workspace向け既存command拡張    | W14より前        | 入力kind、status追加、text/JSON schema拡張、互換性            |
| K06 | `adopt`の全mode                 | T01より前        | grammar、全flag、status、copy/ownership副作用、dry-run、出力  |
| K07 | `migrate`、schema v1の`capture` | L01より前        | grammar、全flag、status、入力schema、変換結果、副作用、出力   |

K01からK07は契約を決めるPRであり、command実装を含めない。各実装PRは対応entryへlinkし、black-box testで
grammar、終了status、stdout/stderr、machine-readable schema、副作用を固定する。

F02は、K01からK07で承認済みのentryと実装・利用文書が一致することを監査する。F02で新しいcommand、
flag、status、output fieldを初めて決めない。
