<!-- markdownlint-disable MD013 -->

# PR ロードマップ

## 行数ルール

各 PR は、手書きによる非テスト実装の追加行と削除行の合計を 500 行以下にする。通常の Go コード、
workflow、shell script、mise など実行動作を変える設定は、このハードゲートに含める。

`*_test.go`、手書き fixture、明示した再生成可能な生成物、文書はそれぞれ別集計する。対応する実装と
テストは同じ PR に置き、テスト量だけを理由に分割しない。手書き総差分（実装、テスト、fixture）が
1,000 行を超えた場合は CI で警告し、自動失敗ではなくレビューで分割要否を判断する。バイナリや
巨大な生成物は追加しない。

表の「目安」はハードゲート対象となる実装差分の見積もりである。実装差分が 450 行を超える見込みに
なった時点で分割を検討し、500 行を超えた PR はレビューへ回さない。分割時にも、各 PR はビルド可能で、
公開済み機能を壊してはならない。

以下の 49 項目は順序と責務を示す初期候補であり、49 個の PR を必須とするものではない。同じ一つの
利用者向け機能を完成させる隣接項目は、実装差分が 500 行以下なら着手前の合意で統合できる。別の
利用者向け機能や仕様判断は、行数に余裕があっても同居させない。

## フェーズ 0: 計画

| ID  | PR の責務          | 主な成果物     |     目安 | 完了条件                                     |
| --- | ------------------ | -------------- | -------: | -------------------------------------------- |
| P00 | 移行計画を固定する | `docs/plan/**` | 文書のみ | 対象、順序、500 行ルール、安全条件を合意する |

## フェーズ 1: 開発と CI の土台

| ID  | PR の責務                      | 主な成果物                                               |    目安 | 完了条件                                                     |
| --- | ------------------------------ | -------------------------------------------------------- | ------: | ------------------------------------------------------------ |
| P01 | Nix導入可能な最小CLIを起動する | `go.mod`、entrypoint、flake/lock、npm版`skills`、wrapper | 300–450 | `nix build`と`nix run`が成功し、書き込みコマンドは存在しない |
| P02 | 固定toolchainをコンテナ化する  | `Dockerfile`、`.dockerignore`、ignore、mise task         | 150–300 | Skill本体なしでイメージを再現でき、hostのホームをmountしない |
| P03 | build経路をCIで検証する        | Go/Nix/container jobs                                    |   40–80 | 読み取り権限だけのPR workflowで三つのbuild経路が成功する     |

P01 の終了時点を「最小コアの実行可能な骨格」かつ「Nixから導入可能な最初の到達点」とする。P03 までに、
Goの直接build、Nix package、開発コンテナを同じCLI契約に対して検証できる状態にする。以後はこの骨格へ
一つずつ振る舞いを追加し、各機能PRでNix packageも壊れていないことを確認する。

## フェーズ 2: 読み取りコア

| ID  | PR の責務                                | 主な成果物                                              |    目安 | 完了条件                                                     |
| --- | ---------------------------------------- | ------------------------------------------------------- | ------: | ------------------------------------------------------------ |
| C01 | workspace と manifest の場所を解決する   | 優先順位、絶対パス検証、設定 fixture                    | 250–450 | flag、環境変数、設定、cwd の各経路を副作用なしで検証できる   |
| C02 | グローバル lock を読み取る               | lock decode、missing と invalid の区別、source identity | 300–450 | source provenance と復元可否を fixture で判定できる          |
| C03 | agent 名と `skills ls` JSON を読む       | agent 正規化、一覧 decode、fake process                 | 300–450 | 不明 agent、壊れた JSON、外部コマンド失敗を区別できる        |
| C04 | manifest のモデルと strict decode を移す | schema 型、未知 field・trailing JSON の拒否             | 250–400 | 合成 manifest を読み取れ、ファイル更新はまだ行わない         |
| C05 | schema、default、名前を検証する          | version、agent、install name の規則                     | 250–430 | schema と名前衝突を実行前に拒否できる                        |
| C06 | remote source を検証する                 | source、ref、skillPath、credential の規則               | 280–450 | portable でない remote entry を実行前に拒否できる            |
| C07 | remote 範囲の `doctor` を公開する        | version 確認、入力・観測診断、CLI テスト                | 250–450 | container fixture に対して読み取り専用で成功・失敗を説明する |

C04 から C06 の間では、不完全な manifest を CLI の通常経路へ通さない。C07 で公開する時点で、
remote entry に必要な検証がすべて有効になっていることを確認する。

## フェーズ 3: remote の計画

| ID  | PR の責務                   | 主な成果物                             |    目安 | 完了条件                                                      |
| --- | --------------------------- | -------------------------------------- | ------: | ------------------------------------------------------------- |
| R01 | remote の基本状態を分類する | install、unchanged、reconfigure        | 300–450 | desired、lock、installed の状態表テストが通る                 |
| R02 | 異常・所有権状態を分類する  | conflict、stale、untracked、名前正規化 | 300–450 | 曖昧な状態を変更対象にせず、明示的な status にする            |
| R03 | `plan` を公開する           | text/JSON 出力、終了条件、CLI テスト   | 250–450 | remote fixture に対して決定的な plan を読み取り専用で出力する |

## フェーズ 4: manifest 更新と remote apply

| ID  | PR の責務                         | 主な成果物                               |    目安 | 完了条件                                                                    |
| --- | --------------------------------- | ---------------------------------------- | ------: | --------------------------------------------------------------------------- |
| M01 | 原子的な状態ファイル更新を移す    | directory lock、temp file、snapshot 比較 | 350–480 | 同時更新、mode 維持、途中失敗を unit test で検証する                        |
| M02 | `add` を公開する                  | dry-run、`--yes`、remote entry 更新      | 250–450 | 全入力検証後だけ manifest を置換し、外部 install は行わない                 |
| A01 | install 引数と process 境界を移す | shell を介さない runner、引数生成        | 250–450 | ref、skillPath、agent を配列引数で正しく渡す                                |
| A02 | install 後の収束制御を移す        | install、再観測、失敗集約                | 300–450 | install 失敗時に prune せず、最終観測結果を返す                             |
| A03 | prune なしの `apply` を公開する   | `--yes`、preflight、global lock          | 300–480 | container の fake CLI で remote install が収束し、再実行が unchanged になる |
| M03 | desired の `remove` を公開する    | dry-run、`--yes`、manifest entry 除外    | 220–400 | workspace source やインストール済み内容を削除せず、manifest だけを更新する  |

`add`、`remove` は desired state の編集、`apply` は observed state の変更として責務を分ける。

## フェーズ 5: prune

| ID  | PR の責務                       | 主な成果物                          |    目安 | 完了条件                                               |
| --- | ------------------------------- | ----------------------------------- | ------: | ------------------------------------------------------ |
| D01 | remote 削除候補を安全に計画する | ownership guard、remove status      | 250–450 | untracked または復元不能な対象を remove にしない       |
| D02 | 削除と再観測を移す              | remove runner、対象再確認、結果検証 | 300–450 | install が収束する前や対象が変化した後は削除しない     |
| D03 | `apply --prune` を公開する      | 明示 flag、確認、統合テスト         | 250–450 | `--prune --yes` の組み合わせだけが隔離環境で削除を行う |

## フェーズ 6: workspace Skill

| ID  | PR の責務                                  | 主な成果物                                          |    目安 | 完了条件                                                              |
| --- | ------------------------------------------ | --------------------------------------------------- | ------: | --------------------------------------------------------------------- |
| W01 | Skill ツリーを安全に列挙する               | path、file type、symlink、digest、mode              | 300–450 | 一時ツリーだけを読み、installer 除外対象を同じ規則で扱う              |
| W02 | frontmatter の境界と name を検証する       | header、終端、name scalar                           | 250–420 | CRLF、未終端、非文字列 name の境界を fixture で固定する               |
| W03 | description と追加 field を検証する        | quoted/block scalar、top-level 制約                 | 300–450 | description の形式と unsupported nested 値を明示的に拒否する          |
| W04 | workspace path と copy を安全に扱う        | root containment、staging copy、mode                | 280–450 | root 外、symlink parent、特殊ファイルを copy しない                   |
| W05 | workspace manifest entry を有効にする      | `kind=workspace`、固定相対 path、directory 検査     | 250–430 | Skill 本体を移行先へ追加せず、合成 workspace を検証できる             |
| W06 | projection を読み書きする                  | XDG state path、strict decode、private atomic write | 300–450 | 実 HOME を使わず、missing・競合・壊れた state を検証できる            |
| W07 | インストールされた全 copy を検証する       | agent path 解決、symlink target、digest 比較        | 350–480 | canonical copy だけを信頼せず、観測可能な copy の一致を確認する       |
| W08 | workspace の基本 plan を追加する           | install、unchanged、content update                  | 300–450 | desired tree と install tree の状態表テストが通る                     |
| W09 | workspace の所有権 plan を追加する         | ownership conflict、unregistered、遷移              | 300–450 | manifest、projection、lock の曖昧な組み合わせを変更対象にしない       |
| W10 | workspace install を追加する               | source 再検証、外部 CLI 引数、失敗集約              | 280–450 | plan 後に source が変わった場合、process 実行前に停止する             |
| W11 | install 前の projection intent を記録する  | crash recovery、状態更新、失敗注入                  | 300–450 | 外部 process より前に再実行可能な所有権 intent が残る                 |
| W12 | remote/workspace 遷移を検証する            | 再観測、provenance 確認、projection 除去            | 300–450 | remote の新しい所有権を確認する前に workspace 記録を捨てない          |
| W13 | workspace 削除後の状態を検証する           | prune 後の再観測、state cleanup                     | 300–450 | install が残る場合や対象が変わった場合は ownership を保持する         |
| W14 | `doctor` と `plan` を workspace 対応にする | 警告、未登録 directory、JSON 出力                   | 220–400 | schema v2 の remote/workspace 混在 fixture を読み取り専用で診断できる |
| W15 | workspace 対応 `apply` を公開する          | global lock、phase 結合、CLI test                   | 280–450 | 途中失敗から再実行でき、同じ workspace の再実行が収束する             |

W01 から W04 は、移行元の大きな `internal/workspace` を安全に分割するため別 PR とする。W15 が
終わるまでは、workspace entry を含む `apply` を明示的に拒否する。

## フェーズ 7: adopt

| ID  | PR の責務                                 | 主な成果物                                   |    目安 | 完了条件                                                  |
| --- | ----------------------------------------- | -------------------------------------------- | ------: | --------------------------------------------------------- |
| T01 | installed Skill の採用方法を分類する      | provenance 判定、name 正規化、agent override | 280–430 | local/untracked と restorable remote を区別する           |
| T02 | `adopt PATH` の copy 計画を移す           | source 検証、dry-run、staging copy           | 300–450 | destination 上書きや symlink を拒否し、まだ commit しない |
| T03 | adopt 用 transaction を移す               | rename、snapshot 比較、rollback              | 300–450 | 失敗注入で manifest と tree の片方だけを commit しない    |
| T04 | `adopt PATH` を公開する                   | CLI flag、projection 更新、統合テスト        | 280–450 | copy と ownership 記録が同じ隔離 transaction で完了する   |
| T05 | `adopt --installed` の remote を公開する  | lock provenance、dry-run、manifest 更新      | 250–430 | 復元可能な remote だけを暗黙の既定値として採用する        |
| T06 | `adopt --installed --as workspace` を公開 | local copy、projection transaction           | 300–450 | 明示指定なしに local Skill を Git 側へコピーしない        |

## フェーズ 8: v1 互換

| ID  | PR の責務                     | 主な成果物                                   |    目安 | 完了条件                                               |
| --- | ----------------------------- | -------------------------------------------- | ------: | ------------------------------------------------------ |
| L01 | `migrate` を移す              | v1 読み取り、v2 candidate、dry-run、`--yes`  | 300–450 | workspace directory と衝突する場合は書き込まず停止する |
| L02 | schema v1 の `capture` を移す | observed からの candidate、warning、競合検出 | 350–480 | schema v2 では拒否し、v1 fixture だけを更新できる      |

## フェーズ 9: parity と切り替え

| ID  | PR の責務                              | 主な成果物                                  |    目安 | 完了条件                                                  |
| --- | -------------------------------------- | ------------------------------------------- | ------: | --------------------------------------------------------- |
| F01 | cross-machine 相当の受け入れ試験を移す | machine fixture、capture/apply/prune の収束 | 300–480 | 端末状態を模した二つの隔離 root でシナリオが通る          |
| F02 | CLI parity と配布物を確定する          | command matrix、container smoke、利用文書   | 250–450 | 合意済み差異以外の未移行コマンド・flag がない             |
| F03 | 正式な切り替えを行う                   | release/cutover 手順、旧実装の参照終了条件  | 150–350 | 読み取り比較の承認後にのみ配布し、rollback 手順を確認する |

F03 まで `.worktrees/skills` は変更・削除しない。旧実装の除去が必要になった場合も、切り替え後の
別 PR とする。

## 各 PR の共通チェックリスト

- [ ] この PR が追加する利用者向けの振る舞いを一文で説明できる。
- [ ] 移行元の基準コミットから参照した関数・テストを PR 本文へ記載した。
- [ ] 未対応の入力を無視せず、明示的に拒否する。
- [ ] 手書きによる非テスト実装の追加行＋削除行が 500 行以下である。
- [ ] 実装と対応テストを同じ PR に含め、実装、テスト、fixture、生成物、文書を別集計した。
- [ ] 手書き総差分が 1,000 行を超える場合、分割要否をレビューした。
- [ ] `skills/**`、実 manifest、端末固有 state、認証情報を含まない。
- [ ] コンテナ内で format、unit test、対象となる integration test が成功する。
- [ ] 書き込みテストは一時 HOME と一時 XDG directory だけを使う。
- [ ] 既存 workflow と、直前までに移行済みの機能が成功する。
- [ ] 意図的な差異、保留事項、次の PR への引き継ぎを PR 本文へ記載した。
