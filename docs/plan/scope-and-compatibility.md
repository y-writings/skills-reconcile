<!-- markdownlint-disable MD013 -->

# 対象範囲と互換性

## 移行単位

移行元は Go の CLI と、固定された npm パッケージとしての `skills` CLI から構成されている。
現状は実装が約 4,300 行、テストが約 4,590 行ある。特に `cmd/skills-sync/main.go`、
`internal/workspace`、`internal/planner` は大きいため、移行単位はファイルではなく責務と振る舞いに
する。テスト量だけを理由に一つの振る舞いを複数 PR へ分けない。

| 責務                             | 移行元                | 主な依存先                                     |
| -------------------------------- | --------------------- | ---------------------------------------------- |
| CLI の組み立て                   | `cmd/skills-sync`     | すべての内部パッケージ                         |
| manifest のモデル・検証・保存    | `internal/manifest`   | agent 検証、ファイルロック                     |
| workspace 解決・Skill ツリー検証 | `internal/workspace`  | ファイルシステム                               |
| グローバル lock の観測           | `internal/lock`       | ファイルシステム                               |
| インストール状態の観測           | `internal/inspect`    | workspace、外部 `skills` CLI                   |
| workspace projection             | `internal/projection` | ファイルロック                                 |
| desired と observed の比較       | `internal/planner`    | manifest、lock、inspect、workspace、projection |
| 外部 CLI の実行                  | `internal/executor`   | planner、manifest、inspect、workspace          |
| install・再観測・prune の制御    | `internal/reconcile`  | planner、executor                              |
| 既存 Skill の採用                | `internal/adopt`      | manifest、lock、inspect、workspace             |
| v1 状態の取り込み                | `internal/capture`    | manifest、lock、inspect                        |
| 排他制御                         | `internal/filelock`   | ファイルシステム                               |

この依存方向に従い、下位の読み取り責務から移す。CLI へ公開するのは、対象機能の失敗条件まで
テストできた後とする。

## コピーするもの

- `cmd/skills-sync` のサブコマンドを、移行先の `cmd/skills-reconcile` へ移した CLI 契約
- `internal/**` の管理ロジックと対応するテストのうち、合意済みの振る舞い
- `go.mod`
- `tools/skills/package.json` と `package-lock.json`
- `flake.nix` と `flake.lock`。移行元からのコピーではなく、移行先の package 定義として作成する
- 開発・テストに必要な `Dockerfile`、`.dockerignore`、関連する ignore 設定
- ツールの利用方法と安全上の注意を説明するドキュメント

import path は移行先のモジュール名へ機械的に変更する。大きなファイルは必要に応じて責務別に
分割できるが、その PR では外部仕様を変えない。

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

CI には禁止パス検査を置き、`skills/**`、実運用 manifest、状態ファイルが誤って追加された場合は
失敗させる。テスト fixture は `internal/**/testdata` または一時ディレクトリだけに置く。

## 維持する契約

最終的な互換性の基準は、移行元のファイル構造ではなく次の利用者向け契約とする。

| 領域           | 維持する契約                                                               | 移行時の確認                |
| -------------- | -------------------------------------------------------------------------- | --------------------------- |
| CLI            | `skills-reconcile` のサブコマンド、主要フラグ、終了コード、出力の役割      | black-box テスト            |
| workspace 選択 | `--workspace`、環境変数、設定、カレントディレクトリの優先順                | table-driven test           |
| manifest       | strict decode、schema、名前、source、ref、agent、決定的な出力              | fixture と unit test        |
| 観測           | lock がない場合と壊れている場合を区別し、`skills ls` を安全に解釈する      | fake CLI と fixture         |
| plan           | install、reconfigure、unchanged、conflict、stale、untracked などを区別する | 状態表テスト                |
| apply          | 全体を事前検証し、install 後に再観測してから成功とする                     | コンテナ統合テスト          |
| prune          | 明示指定時だけ実行し、所有権と再観測で削除対象を確認する                   | 負のテストを含む統合テスト  |
| workspace      | Skill ツリーの内容、実行 bit、symlink、パス、digest を検証する             | 一時ツリーのテスト          |
| projection     | 端末固有状態を Git 管理せず、競合しない原子的更新を行う                    | 分離した XDG state のテスト |
| adopt          | remote と workspace を暗黙に混同せず、上書き前に停止する                   | dry-run と失敗系テスト      |
| v1 互換        | `migrate` と schema v1 の `capture` を schema v2 の通常経路から分離する    | 互換 fixture                |

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

### 読み取り専用

最初に workspace の場所、manifest、グローバル lock、インストール済み一覧を読み取る。次に
`doctor` と remote Skill の `plan` を公開する。この間は外部 CLI のバージョン確認と一覧取得以外の
サブコマンドを呼ばない。

### 追加・更新

原子的な manifest 更新と排他制御を先に移し、`add` を公開する。その後、外部 CLI の引数生成、
install、再観測、収束判定を順番に移し、最後に prune なしの `apply` を公開する。

### 削除

manifest から desired entry を外す `remove` と、実インストールを削除する `apply --prune` は別の
機能として扱う。後者は所有権確認、install の収束、削除後の再観測がそろうまで公開しない。

### workspace と adopt

workspace Skill の読み取り、内容検証、projection、plan、install を順番に移す。workspace の
通常経路が収束してから `adopt PATH`、`adopt --installed` を追加する。

### v1 互換

schema v2 の主要経路が完成した後で `migrate` と `capture` を移す。互換コマンドは通常の
schema v2 経路へ機能を混ぜず、対象 schema を明示して検証する。

## バグまたは仕様差を見つけた場合

具体的な修正内容はこの計画では決めない。機能スライスの実装中に、移行元のコード、テスト、README
の間で矛盾が見つかった場合は、PR 作成前なら commit、push、PR 作成を行わずに止める。PR 作成後の
CI またはレビューで見つかった場合は、通常の PR を open のまま残して merge せず、次の情報だけを
提示して判断を依頼する。

1. 再現に必要な最小入力
2. 移行元で実際に起きること
3. テストまたは文書から読み取れる期待値
4. そのまま互換にする場合と、先に直す場合の影響範囲
5. 手書きによる非テスト実装の 500 行制限内で分離可能かどうか

判断後は、互換移行、修正を含む移行、独立した先行修正のいずれかを明示する。合意されていない
挙動変更や一般的なリファクタリングを、行数調整のために混ぜない。

## 最終 parity の定義

完了時には次を満たす。

- `skills-reconcile` のサブコマンドと主要フラグの一覧が、合意した移行元機能と一致する。
- 合意して移した契約テストが、移行先のコンテナで成功する。
- remote、workspace、adopt、prune、v1 互換の代表シナリオが合成環境で収束する。
- 禁止対象の Skill 本体、実 manifest、端末固有状態を Git が追跡していない。
- 意図的な差異はすべて文書化され、未判断の差異が残っていない。
- 実環境へ切り替える前に、読み取り専用の `doctor` と `plan` の結果を人が比較している。
