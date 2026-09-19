<!-- markdownlint-disable MD013 -->

# 対象範囲と互換性

## 実装単位

移行単位はinventoryのファイルやpackageではなく、合意した製品責務と利用者から見える振る舞いにする。
inventoryにある約4,300行の実装と約4,590行のテストは、対象機能や依存関係の所在を調べる用途に限る。
テスト量だけを理由に一つの振る舞いを複数PRへ分けない。

| 責務                             | 移行先のowner候補      | 主な依存先                                  |
| -------------------------------- | ---------------------- | ------------------------------------------- |
| CLI の組み立て                   | `cmd/skills-reconcile` | すべての内部package                         |
| manifest のモデル・検証・保存    | `internal/manifest`    | agent検証、ファイルロック                   |
| workspace 解決・Skill ツリー検証 | `internal/workspace`   | ファイルシステム                            |
| 外部CLIとinstall状態の観測       | `internal/skillscli`   | 公開された `skills` CLI                     |
| 所有権のintentとreceipt          | `internal/receipt`     | manifest宣言、公開CLI観測、tree fingerprint |
| workspace projection             | `internal/projection`  | receipt、ファイルロック                     |
| desired と observed の比較       | `internal/planner`     | manifest、公開CLI観測、receipt、workspace   |
| 外部 CLI の実行                  | `internal/executor`    | planner、manifest、workspace                |
| install・再観測・prune の制御    | `internal/reconcile`   | planner、executor、receipt                  |
| 既存 Skill の明示的な採用        | `internal/adopt`       | manifest、公開CLI観測、receipt、workspace   |
| 明示されたv1入力の取り込み       | `internal/capture`     | manifest、公開CLI観測、receipt              |
| 排他制御                         | `internal/filelock`    | ファイルシステム                            |

この依存方向に従い、下位の読み取り責務から移す。CLI へ公開するのは、対象機能の失敗条件まで
テストできた後とする。

## 実装するもの

- 合意済みの `skills-reconcile` CLI 契約
- 合意済みの管理ロジックと、その振る舞いを証明する移行先固有のテスト
- `go.mod`
- `tools/skills/package.json` と `package-lock.json`
- `flake.nix` と `flake.lock`。移行元からのコピーではなく、移行先の package 定義として作成する
- 開発・テストに必要な `Dockerfile`、`.dockerignore`、関連する ignore 設定
- ツールの利用方法と安全上の注意を説明するドキュメント

移行先は `github.com/y-writings/skills-reconcile` のpackageとして実装する。inventoryのpackage境界や
関数を互換対象にせず、上表の責務に沿って必要最小限のAPIを定義する。

## コピーしないもの

次のパスまたはデータは、管理ツールの実装ではないため移行しない。

| 移行元                           | 理由                                |
| -------------------------------- | ----------------------------------- |
| `skills/**`                      | Skill 本体であり、今回の対象外      |
| `skills-manifest.json`           | 移行元の実 Skill 一覧を参照している |
| `state/**`                       | 特定端末で観測された状態である      |
| `.git/**`                        | 移行元リポジトリの履歴・設定である  |
| 移行元のビルド済み `skills-sync` | 移行先のソースから再現する          |
| `tools/skills/node_modules/**`   | lockfile からコンテナ内で再構築する |
| `$HOME` 以下の Skill・設定・lock | 利用者固有の状態である              |

`skills/**`、実運用 manifest、状態ファイルの混入は、merge baseからの最終差分監査とPRレビューで
確認する。機密情報の機械的な検出は既存security scanに委ねる。テスト fixture は
`internal/**/testdata` または一時ディレクトリだけに置く。

## 製品契約

実装の基準は、次の利用者向け契約と、固定した `skills` が公開するCLI境界とする。CLIの具体的な
grammar、flag、終了status、output schemaは[CLI契約](cli-contract.md)の承認済みentryだけを正本とする。

| 領域           | 契約                                                                       | 検証                        |
| -------------- | -------------------------------------------------------------------------- | --------------------------- |
| CLI            | CLI契約で承認したcommand、全flag、終了status、stdout/stderr、output schema | black-box テスト            |
| workspace 選択 | `--workspace`、環境変数、設定、カレントディレクトリの優先順                | table-driven test           |
| manifest       | JSON完全性、schema、名前、opaqueなsource宣言、agent、決定的な出力          | fixture と unit test        |
| 観測           | `skills --version` と `skills list -g --json`、installed treeを安全に読む  | fake CLI と fixture         |
| ownership      | intent、宣言digest、公開CLI観測、tree fingerprintをreceiptへ結び付ける     | 状態表と失敗注入            |
| plan           | install、reconfigure、unchanged、conflict、untrackedなどを区別する         | 状態表テスト                |
| apply          | intent記録後に実行し、再観測とreceipt確定後だけ成功とする                  | コンテナ統合テスト          |
| prune          | 明示指定時だけ、receiptと現在のfingerprintが一致する対象を削除する         | 負のテストを含む統合テスト  |
| workspace      | Skill ツリーの内容、実行 bit、symlink、パス、digest を検証する             | 一時ツリーのテスト          |
| machine state  | receiptとprojectionをGit管理せず、競合しない原子的更新を行う               | 分離した XDG state のテスト |
| adopt          | remoteのsource宣言とkindを明示させ、上書き前に停止する                     | dry-run と失敗系テスト      |
| v1 互換        | 明示されたv1入力だけをschema v2の通常経路から分離して扱う                  | 互換 fixture                |

読み取り対象のJSONは、UTF-8、単一値、全階層の重複memberがないことを共通検証する。JSON objectは
`encoding/json`で型decodeし、各readerが必要とするstruct fieldだけを読み取る。未知fieldは読み飛ばし、
field名の照合とJSON tagの解釈には`encoding/json`の標準動作を使う。大小文字に対する追加制約や独自の
tag schema検証は設けない。fieldの必須性、null、domainのpolicyは各readerが検証する。

workspace configはtop-level objectとし、`workspace`を文字列として読み取る。未知fieldは読み飛ばす。
`workspace`がない場合と空文字列の場合は未設定として扱い、`null`と文字列以外の値は拒否する。

remote source は外部CLIへ渡す宣言全体をそのまま保持し、完全一致と宣言digestだけを比較する。
`skills-reconcile` は空値、制御文字、credential、remote kindでのlocal pathなど、自身のmanifestと
実行安全性に属する規則だけを検証する。provider、alias、SCP、port、well-known URLの構文、または異なる
表記の意味的同一性は判定しない。固定CLIがsourceを受理できるかどうかはapply時の実行結果で確定する。

`skills-reconcile` は外部CLIのprivate global lockを直接読まない。公開一覧でsource情報が欠ける場合は
原因を推測せずprovenance不明として扱い、receiptで所有権を証明できないinstallを変更または削除しない。
receiptは外部CLI実行前のintentと事前観測を持ち、実行後の公開CLI観測とtree fingerprintが一意に
確認できた場合だけ確定する。

## 意図的に変更する名前

既存利用者との後方互換性は不要なため、移行先では製品名に連動する識別子を
`skills-reconcile` に統一する。旧名との fallback や二重読み取りは追加しない。

| 対象                | 移行元                   | 移行先                        |
| ------------------- | ------------------------ | ----------------------------- |
| executable          | `skills-sync`            | `skills-reconcile`            |
| command directory   | `cmd/skills-sync`        | `cmd/skills-reconcile`        |
| workspace 環境変数  | `SKILLS_SYNC_WORKSPACE`  | `SKILLS_RECONCILE_WORKSPACE`  |
| executable 環境変数 | `SKILLS_SYNC_EXECUTABLE` | `SKILLS_RECONCILE_EXECUTABLE` |
| XDG namespace       | `skills-sync`            | `skills-reconcile`            |
| global apply lock   | `.skills-sync-apply`     | `.skills-reconcile-apply`     |

tree digest の識別文字列のように永続データの計算結果へ影響する内部識別子は、単なる名称変更として
一括置換しない。該当機能を移す PR で、互換性が不要であることと計算結果への影響を確認して決める。

## Nix package の契約

[driftline の flake](https://github.com/y-writings/driftline/blob/main/flake.nix) と
[Nixによる導入例](https://github.com/y-writings/driftline/blob/main/README.md#install) を構成の参考にし、
次の output と利用経路を初期段階から提供する。

- `packages.<system>.skills-reconcile` と `packages.<system>.default`
- `apps.<system>.skills-reconcile` と `apps.<system>.default`
- `x86_64-linux`、`aarch64-linux`、`x86_64-darwin`、`aarch64-darwin`
- `nix build .#skills-reconcile`
- `nix run .#skills-reconcile -- ARGS...`
- `nix profile install github:y-writings/skills-reconcile#skills-reconcile`

driftline は `buildGoModule` で Go CLI を作り、runtime dependency を wrapper の `PATH` へ加えている。
`skills-reconcile` では同じ考え方を用いるが、Go binary だけでは完全な package にならない。固定済みの
npm版 `skills`、それを実行する Node.js、Go binary を同じ Nix closure から利用できるようにし、
wrapper から `SKILLS_RECONCILE_EXECUTABLE` を Nix store 内の executable へ固定する。

Nix package は Skill 本体、実 manifest、端末固有 state を含めない。build と install は利用者の
Skill を変更せず、`skills-reconcile` を明示的に実行したときだけ通常の CLI 処理が始まる。

`flake.lock` は移行先で生成してnixpkgsを固定し、npm依存は `package-lock.json` で固定する。
Nix package の更新と無関係なGo、Node.js、`skills` の更新を同じPRに含めない。

## 機能の公開順

各節のcommand名は実装範囲を示し、interfaceを定義しない。対応するCLI契約PRを先に承認し、そのentryを
参照する実装PRだけがcommandを公開できる。CLI契約と実装を同じPRで新規決定しない。

### 読み取り専用

最初に workspace の場所、manifest、machine-local receipt、公開CLIによるインストール済み一覧を
読み取る。次に `doctor` と remote Skill の `plan` を公開する。この間は外部CLIのバージョン確認と
一覧取得以外のサブコマンドを呼ばない。

### 追加・更新

原子的な manifest 更新と排他制御を先に実装し、`add` を公開する。その後、外部CLIの引数生成、
intent記録、install、再観測、receipt確定を順番に実装し、最後にpruneなしの `apply` を公開する。

### 削除

manifest から desired entry を外す `remove` と、実インストールを削除する `apply --prune` は別の
機能として扱う。後者は所有権確認、install の収束、削除後の再観測がそろうまで公開しない。

### workspace と adopt

workspace Skill の読み取り、内容検証、projection、plan、install を順番に実装する。remoteとworkspace
という二つ目のinstall kindがそろった時点で、共通実行pipelineからkind固有処理を分離する。
workspaceの通常経路が収束してから `adopt PATH`、明示sourceを要求する `adopt --installed` を追加する。

### v1 互換

schema v2 の主要経路が完成した後で、独立して承認された `migrate` と `capture` を実装する。
互換コマンドは対象schemaとsource宣言を明示して検証し、private lockから値を推測しない。

## バグまたは仕様差を見つけた場合

具体的な修正内容はこの計画では決めない。機能スライスの実装中に、承認済みの製品契約と固定した
外部CLIの公開動作が両立しない場合は、PR作成前ならcommit、push、PR作成を行わずに止める。
PR作成後のCIまたはレビューで見つかった場合は、PRをopenのまま残してmergeせず、次の情報だけを
提示して判断を依頼する。

1. 再現に必要な最小入力
2. 承認済みの製品契約
3. 固定した外部CLIの公開境界で実際に起きること
4. 契約または依存境界を変更する場合の影響範囲
5. 手書きによる非テスト実装の 500 行制限内で分離可能かどうか

判断後は、製品契約の変更、dependency adapterの変更、実装修正のいずれかを明示する。合意されて
いない挙動変更や一般的なリファクタリングを、行数調整のために混ぜない。

## 完了条件

完了時には次を満たす。

- `skills-reconcile` のサブコマンドと主要フラグが、承認済みの製品契約と一致する。
- 合意した契約テストが、移行先のコンテナで成功する。
- remote、workspace、adopt、prune、v1 互換の代表シナリオが合成環境で収束する。
- 禁止対象の Skill 本体、実 manifest、端末固有状態を Git が追跡していない。
- 固定した外部CLIの公開境界に対するcontract testが成功する。
- 実環境へ切り替える前に、読み取り専用の `doctor` と `plan` の結果を人が承認している。
