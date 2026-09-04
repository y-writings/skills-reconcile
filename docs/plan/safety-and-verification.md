<!-- markdownlint-disable MD013 -->

# 安全性と検証

## 実行境界

移行作業中の機能検証はコンテナ内で行う。リポジトリは `/workspace` へ mount してよいが、利用者の
ホームディレクトリ、`.agents`、各 agent の設定 directory、実際の XDG directory、Docker socket は
mount しない。

コンテナ内では、少なくとも次の値を専用の一時領域へ固定する。

| 状態       | コンテナ内の例                     | host から引き継がないもの             |
| ---------- | ---------------------------------- | ------------------------------------- |
| HOME       | `/tmp/skills-reconcile-home`       | `$HOME/.agents` と agent ごとの Skill |
| XDG config | `/tmp/skills-reconcile-xdg/config` | 実 workspace 設定                     |
| XDG state  | `/tmp/skills-reconcile-xdg/state`  | `.skill-lock.json` と projection      |
| XDG cache  | `/tmp/skills-reconcile-xdg/cache`  | npm や CLI の既存 cache               |
| manifest   | テストごとの一時 workspace         | 移行元の `skills-manifest.json`       |

テストは開始時に空の一時 root を作り、終了時はコンテナごと破棄する。固定した `skills` CLI の
install/remove を使う統合テストも、この root の外へ書き込めない構成にする。

## コンテナ方針

- 移行元と同じ Go、Node.js、`skills` CLI のバージョンから開始する。
- base image と GitHub Actions は移行先の既存方針に合わせて immutable な digest/SHA で固定する。
- `npm ci --ignore-scripts` で lockfile から依存関係を再現する。
- `SKILLS_RECONCILE_EXECUTABLE` はコンテナ内の固定済み executable だけを指す。
- `npx skills` や host の `PATH` 上に偶然存在する executable は使わない。
- ローカルと CI は同じ build/test entrypoint を使い、CI 専用の別手順を増やさない。
- 開発用 image に認証情報を bake せず、ネットワークを必要とするのは image build 時の依存取得に
  限定する。

## Nixとコンテナの責務

Nixは `skills-reconcile` を再現可能にbuild・install・実行する配布経路とする。コンテナはHOMEやXDG
stateへの副作用を隔離し、実際のinstall、prune、adoptを検証する実行境界とする。Nix buildがsandbox
内で成功することは、CLIをhostの実状態に対して安全に実行できることを意味しないため、書き込み系の
統合テストは引き続きコンテナ内だけで行う。

Nix packageでは次を検証する。

- `nix flake check`
- `nix build .#skills-reconcile`
- build結果の `bin/skills-reconcile` に対する最小smoke test
- `nix run .#skills-reconcile -- --help` 相当のapp output
- package内で固定したnpm版 `skills` とNode.jsを解決でき、repoの `node_modules` に依存しないこと
- package closureにSkill本体、実manifest、端末固有stateが含まれないこと

LinuxとDarwin、x86_64とaarch64のpackage/app outputをflakeで定義する。必須CIでは少なくともLinuxの
buildを実行し、利用対象となるDarwinについてもrunnerまたは合意したローカルNix環境で確認する。
全system向けoutputの評価と、現在のsystemで実際にbuildできることを混同しない。

## テスト層

### 1. Unit test

manifest、lock、path、status、引数生成など、process や実 HOME を必要としない規則を検証する。
境界値と拒否すべき入力を table-driven test で表す。

### 2. Component test

外部 `skills` CLI は fake executable または `Runner` で置き換える。呼び出し回数よりも、shell を
介さない引数、失敗時の停止位置、再観測の順序を確認する。

### 3. Container integration test

合成した Skill/manifest と protocol を再現する fake CLI を使い、隔離 HOME 内で install、再実行、
reconfigure、prune、adopt をネットワーク非依存で検証する。固定済みの実 `skills` CLI は、local fixture
で再現できる範囲と `--version`、一覧形式の互換確認に使う。公開 remote へのアクセスは必須 CI に
しない。機能がまだ未移行なら、そのシナリオは追加しない。

### 4. Read-only shadow test

切り替え直前に限り、利用者が明示的に選んだ実 workspace に対して新旧の `doctor` と `plan` を
読み取り専用で実行し、正規化した結果を人が比較する。エージェントが実環境で `apply` や
`--prune` を自動実行しない。

## GitHub Actions

既存の `00-security-scan.yaml`、`00-semantic-pr-check.yaml`、
`00-weekly-merged-prs-report.yaml`、`00-entire-checkpoint-collection.yaml` はそのまま維持し、ツール用の
workflow を独立して追加する。最初は次の job に分ける。既存 workflow の変更が必要になった場合は、
機能移行と混ぜずに独立した CI PR とする。

| Job         | 役割                                             | 必須条件                             |
| ----------- | ------------------------------------------------ | ------------------------------------ |
| diff-policy | 実装 500 行、差分内訳、禁止パスの確認            | checkout 以外の write 権限を持たない |
| nix-package | flake check、package build、app smoke            | `flake.lock` とNix sandboxを使う     |
| build-unit  | container build、format、static check、unit test | 固定 image とlockfileを使う          |
| integration | 現在までに公開したシナリオだけを隔離 root で実行 | 実 secret とhost stateを使わない     |

推奨する基本検査は `gofmt` 差分なし、`go vet ./...`、`go test ./...`、必要な段階から
`go test -race ./...` である。workflow 自体は `actionlint` と既存の pin 管理対象に含める。

PR の `pull_request` workflow は `contents: read` を基本とし、PR から package 公開、release 作成、
manifest 書き換えを行わない。配布が必要になる F03 は、テスト workflow と分け、承認された tag
または手動 trigger だけを入口にする。

## 500 行 gate

集計は PR の merge base と head の diff に対して行い、次の内訳を CI と PR 本文へ表示する。

| 区分           | 例                                         | 判定                                     |
| -------------- | ------------------------------------------ | ---------------------------------------- |
| 手書き実装     | 非 test Go、workflow、script、実行設定     | 追加＋削除 500 行以下を必須とする        |
| テスト         | `*_test.go`、テスト用 helper・fake         | 別集計し、実装と同じ PR に含める         |
| 手書き fixture | `testdata`、合成 manifest・Skill           | 別集計し、テストと同じ PR に含める       |
| 生成物         | npm/Nix lockfile、再生成可能なsource・設定 | 明示したpathだけを別集計する             |
| 文書           | `docs/**`、明示した README など            | 別集計する                               |
| 純粋な rename  | 内容を変更しない path 移動                 | 別集計し、変更された行だけを実装へ数える |

手書き総差分は、手書き実装、テスト、手書き fixture の合計とする。これが 1,000 行を超えた場合、
CI は警告を出すが自動失敗にはしない。レビューでは、テストが一つの振る舞いを検証するための反復的な
case なのか、新しいテスト基盤や複数の仕様を含むのかを確認し、後者なら分割する。

生成物を別集計できるのは、生成元が同じ PR にあり、固定した手順で再生成でき、CI で差分なしを
確認できる場合だけとする。それ以外の生成 source・設定は手書き実装として数える。`package.json` と
`package-lock.json`、`flake.nix` と `flake.lock` のように原子的であるべき組み合わせは分離せず、
同じ PR に含める。

自動集計が 500 行以下でも、一つの振る舞いとしてレビューしにくい場合はさらに分ける。一方、実装と
対応テストを別 PR にしたり、テスト量だけを理由に一つの振る舞いを分割したりしない。

## 禁止パス gate

少なくとも次を検出する。

- ルートの `skills/**`
- ルートの実運用用 `skills-manifest.json`
- `state/**`、`.skill-lock.json`、`workspace-projections.json`
- `node_modules/**`、ビルド済み `skills-reconcile`
- private key、token、認証情報を含む URL、利用者の絶対パス

テスト fixture のファイル名が状態ファイルと同じになる場合は、`testdata` の下に合成データである
ことを明示し、secret scan も通す。

## 1 PR ごとの協業フロー

1. ロードマップから次の一項目だけを選ぶ。
2. 実装前に、入力、期待する出力、副作用、未対応範囲を短い表として PR 本文または Draft に置く。
3. 移行元の該当コードとテストを確認し、採用する契約を列挙する。
4. 合成 fixture と失敗系テストを先に用意し、コンテナで対象範囲だけを反復する。
5. 対象の振る舞いを移し、全テスト、差分内訳、500 行 gate、禁止パス gate を実行する。
6. Draft PR で差分と意図的な未対応範囲を一緒に確認する。
7. CI とレビューが完了してから squash merge し、ロードマップへ実測と引き継ぎを反映する。
8. `main` の成功を確認してから次の機能に着手する。

この流れにより、仕様判断が必要な箇所を実装前または小さな差分の段階で相談できる。バグらしき
挙動を発見した場合は、[対象範囲と互換性](scope-and-compatibility.md) の判断情報を提示し、指示が
あるまで当該機能の公開を進めない。

## PR の停止条件

次のいずれかに当たる場合はマージせず、分割または相談する。

- 手書きによる非テスト実装の差分が 500 行を超えた。
- 実装と対応テストが別 PR になっている。
- 手書き総差分が 1,000 行を超え、分割要否をレビューしていない。
- 既存機能の仕様変更が、対象機能の移行に混ざった。
- 実 HOME、実 Skill、実 manifest、認証情報がテストに必要になった。
- 移行元のコード、テスト、README が同じ入力に異なる期待値を示した。
- 削除対象または workspace 所有権を、観測状態から一意に証明できない。
- 前の PR が未マージまたは `main` の CI が失敗している。
- コンテナと CI で結果が一致しない。

## 切り替え手順

1. F01 までの合成環境テストをすべて成功させる。
2. コマンド・flag・status・出力形式の parity 表を完成させる。
3. 新旧をそれぞれ同じ manifest snapshot と観測 snapshot に対して読み取り専用で実行する。
4. 差分を「合意済み」「未解決」に分類し、未解決をゼロにする。
5. 実 workspace では `doctor`、次に `plan` だけを実行し、結果を人が承認する。
6. 独立した F03 PR で build・配布・利用文書を確定する。
7. 新実装による最初の書き込み操作は prune なしで行い、再度 `plan` して収束を確認する。
8. `--prune` は別の明示作業として扱い、削除対象をレビューしてから実行する。

## ロールバック

- 機能移行中は、問題のある最新 PR を revert し、直前の `main` へ戻す。
- 移行元 `.worktrees/skills` は F03 完了まで変更しないため、比較と読み取り専用運用に使える。
- 移行先を配布しても、旧実装を即座に削除しない。少なくとも最初の install と再 plan が収束する
  までは旧実装の基準コミットを保持する。
- 外部 install が一部成功した場合は、無条件の逆操作をしない。新旧いずれかの `plan` と観測状態を
  保存し、所有権を確認してから再実行または手動復旧を選ぶ。
- prune 後の自動復元は計画しない。削除前レビューと manifest/lock の provenance を復旧根拠にする。
