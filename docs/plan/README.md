<!-- markdownlint-disable MD013 -->

# skills-sync から skills-reconcile への移行計画

## 目的

合意した製品契約に基づいて、このリポジトリに `skills-reconcile` を安全に実装する。
`.worktrees/skills` にある `skills-sync` は対象機能の所在を調べるための inventory と、切り替え時の
rollback 参照に限定し、その実装、テスト、出力を仕様または設計判断の根拠にしない。
移行は小さな Pull Request（以下 PR）を順番にマージして進め、ローカルの Skill、グローバルな
インストール状態、既存の GitHub Actions を壊さないことを最優先とする。

この計画が扱うのは移行の順序、PR の境界、検証方法、切り替え方法である。既知・未知を問わず、
個別バグの修正内容やリファクタリング内容は扱わない。移行中に問題を発見した場合の判断手順だけを
定める。

## 前提

- 承認済みの製品契約は `docs/plan/**` を正本とする。
- `.worktrees/skills` のコミット `3c15f60` は、変更しない inventory と rollback 参照として固定する。
- 移行先の開始点は、この計画作成時点の `main`、コミット `b6732ef` とする。
- Go モジュール名は移行先に合わせて `github.com/y-writings/skills-reconcile` へ変更する。
- 移行先の CLI 名はリポジトリ名に合わせて `skills-reconcile` とする。既存利用者はいないため、
  `skills-sync` という CLI 名の互換 alias は設けない。
- Go 1.24.0 と `skills` 1.5.23 を固定する。コンテナでは
  Node.js 22.20.0 を維持し、Nix package では `skills` が要求する Node.js 22.20.0 以上を
  `flake.lock` で固定する。依存関係の更新は移行と同時に行わない。
- `skills` について依存する契約は、固定versionが公開するCLI commandとmachine-readable outputに
  限定する。private module、private source parser、global lockの内部schemaには依存しない。
- 機能 PR は直列にマージし、原則として未マージの機能 PR の上へ次の機能 PR を積まない。
- 各 PR の手書きによる非テスト実装は、追加行と削除行の合計で 500 行以下にする。テスト、
  手書き fixture、生成物、文書は別集計する。

inventory の参照revisionは自動では動かさない。参照revisionの変更は、対象機能の所在を調べる必要が
生じた場合だけ独立して合意する。revisionを変更しても製品契約は変わらない。

## 完了時の状態

- 管理ツールのソース、テスト、固定された外部 CLI、開発コンテナ、CI が移行先だけで完結する。
- `nix run`、`nix build`、`nix profile install` から `skills-reconcile` を利用できる。
- Skill 本体や実環境の状態をリポジトリへ持ち込まず、合成した fixture で検証できる。
- 読み取り系、書き込み系、削除系の順に機能が移行され、各段階が単独でレビュー可能である。
- inventory を変更せずに残し、最終確認が終わるまで rollback 参照として利用できる。
- 最終的な切り替えは、全機能を移し終えた後の独立した PR として扱う。

## 対象外

- `.worktrees/skills/skills/**` にある Skill 本体
- `.worktrees/skills/skills-manifest.json` にある実 Skill の一覧
- `state/.skill-lock.json`、workspace projection、グローバルインストールなどの端末固有状態
- 認証情報、絶対パス、タイムスタンプ、利用者のホームディレクトリ内の設定
- この計画内での個別バグ修正、仕様変更、依存関係更新、一般的なコード整理
- 移行完了前の配布、実環境へのインストール、実データに対する `apply` や `--prune`

テストに必要な manifest と Skill は `testdata` またはテストごとの一時ディレクトリへ、実データと
無関係な内容で作る。移行先のルートへ実運用用 `skills-manifest.json` は追加しない。

## 全体フロー

| 段階            | 到達点                                                | 書き込み範囲     |
| --------------- | ----------------------------------------------------- | ---------------- |
| 0. 計画固定     | 対象、基準コミット、PR 制約を合意する                 | ドキュメントのみ |
| 1. 安全な土台   | Nixで導入でき、同じコンテナをローカルとCIから使える   | リポジトリ内のみ |
| 2. 最小コア     | CLI、workspace、manifest、公開CLI観測を読み取れる     | なし             |
| 3. 読み取り計画 | remote Skill の差分を `plan` で説明できる             | なし             |
| 4. remote 更新  | `add` と prune なしの `apply` を検証付きで実行できる  | 隔離環境のみ     |
| 5. 削除         | manifest からの除外と明示的な `--prune` を扱える      | 隔離環境のみ     |
| 6. workspace    | Skill ツリーと端末固有 projection を安全に扱える      | 隔離環境のみ     |
| 7. adopt        | パスまたはインストール済み Skill を明示的に採用できる | 隔離環境のみ     |
| 8. 互換機能     | 承認したv1の `migrate` と `capture` を実装する        | 隔離環境のみ     |
| 9. 切り替え     | 製品contractを確認し、移行先を正式な管理元にする      | 合意した環境のみ |

最小コアは「`skills-reconcile` の最小 CLI を Nix とコンテナの両方でビルドでき、読み取り専用の
入力検証までを CI で再現できる状態」とする。この段階では Skill のインストール、manifest 更新、
削除を行わない。

## 進め方の原則

1. 合意した一つの振る舞いを移行先の責務境界に合わせて実装し、その振る舞いを固定するテストを
   同じ PR に置く。inventory のファイルを実装単位としてコピーしない。
2. まだ移していない入力やフラグは、無視せず「未対応」として失敗させる。
3. 読み取り、追加・更新、削除の順序を守る。特に削除は `--prune` と明示確認を維持する。
4. 実環境の `$HOME`、`XDG_CONFIG_HOME`、`XDG_STATE_HOME` をテストや開発コンテナへ渡さない。
5. 各 PR は `main` へマージされた直前の PR だけに依存し、単独でビルド・テスト可能にする。
6. inventory のコードとテストは機能や依存関係の所在を調べる用途に限定する。そこから仕様、互換性、
   責務境界を推論しない。
7. バグ修正または仕様変更が必要になった場合は、機能移行 PR に暗黙に混ぜない。
8. Nix package は利用・配布の経路、コンテナは副作用を隔離する開発・統合テストの境界として扱い、
   どちらか一方で他方を代替しない。
9. remote source は外部 CLI へ渡す opaque な宣言として保持する。provider、alias、SCP、port、
   well-known URL の解釈や意味的同一性を再実装しない。
10. 外部 CLI の private lock を所有権の根拠にしない。書き込み前の intent と、成功後の公開CLI観測・
    installed tree fingerprintを結び付けたmachine-local receiptで所有権を記録する。

## 計画書の構成

- [対象範囲と互換性](scope-and-compatibility.md): 実装対象、責務、機能ごとの完了条件
- [PR ロードマップ](pr-roadmap.md): 実装 500 行以下を前提とした具体的なマージ順
- [安全性と検証](safety-and-verification.md): コンテナ、CI、協業、切り替え、ロールバック

## 計画の変更方法

実装開始後は、完了した PR をロードマップ上でチェックし、実測行数と次の PR への引き継ぎを
追記する。PR の分割・統合が必要な場合は、次の条件をすべて守る範囲で計画を更新する。

- 一つの PR が一つのレビュー可能な振る舞いを持つ。
- 手書きによる非テスト実装の差分が 500 行以下である。
- 対応するテストを同じ PR に含め、テスト量だけを理由に分割していない。
- 削除や実環境への書き込みを前倒ししない。
- 未解決の仕様判断を仮定で実装しない。
