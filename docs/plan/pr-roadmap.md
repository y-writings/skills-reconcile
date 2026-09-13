<!-- markdownlint-disable MD013 -->

# PR ロードマップ

## 行数ルール

各 PR は、手書きによる非テスト実装の追加行と削除行の合計を 500 行以下にする。通常の Go コード、
workflow、shell script、mise など実行動作を変える設定は、このハードゲートに含める。

`*_test.go`、手書き fixture、明示した再生成可能な生成物、文書はそれぞれ別集計する。対応する実装と
テストは同じ PR に置き、テスト量だけを理由に分割しない。手書き総差分（実装、テスト、fixture）が
1,000 行を超えた場合は作業を停止し、レビューで分割要否を判断する。バイナリや巨大な生成物は
追加しない。

表の「目安」はハードゲート対象となる実装差分の見積もりである。実装差分が 450 行を超える見込みに
なった時点で分割を検討し、500 行を超えた PR はレビューへ回さない。分割時にも、各 PR はビルド可能で、
公開済み機能を壊してはならない。

以下の項目は順序と責務を示す初期候補であり、同数のPRを必須とするものではない。同じ一つの
利用者向け機能を完成させる隣接項目は、実装差分が 500 行以下なら着手前の合意で統合できる。別の
利用者向け機能や仕様判断は、行数に余裕があっても同居させない。

`K`で始まる項目は文書だけのCLI契約PRとする。grammar、全flag、終了status、stdout/stderr、JSON
schema、副作用、未対応入力を[CLI契約](cli-contract.md)へ確定し、対応する実装項目より先に承認する。
explicitなstackで進める場合も、契約PRを対応する実装PRの直接のbaseにする。実装PRで未承認の
interfaceを追加または変更しない。

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

| ID  | PR の責務                                    | 主な成果物                                                 |     目安 | 完了条件                                                     |
| --- | -------------------------------------------- | ---------------------------------------------------------- | -------: | ------------------------------------------------------------ |
| C01 | workspace と manifest の場所を解決する       | 優先順位、絶対パス検証、設定 fixture                       |  250–450 | flag、環境変数、設定、cwd の各経路を副作用なしで検証できる   |
| J01 | JSON 文書の完全性を共通化する                | UTF-8、単一値、全階層の重複 member 拒否                    |   50–100 | workspace と後続 reader が同じ事前検証を利用できる           |
| J02 | canonical object decodeを共通化する          | object必須、known fieldのcase alias拒否                    |    50–90 | unknown field policyをcallerに残して型decodeできる           |
| C02 | workspace設定をforward compatibleにする      | canonicalな`workspace`、unknown field許容                  |    20–50 | 将来fieldと既存workspace設定を同時に読み取れる               |
| C03 | 公開された `skills` CLI を観測する           | version確認、global list decode、agent正規化、fake process |  300–450 | private lockを読まず、CLI失敗と曖昧な一覧を区別できる        |
| C04 | manifest のモデルと strict decode を実装する | schema 型、未知 field・trailing JSON の拒否                |  250–400 | 合成 manifest を読み取れ、ファイル更新はまだ行わない         |
| C05 | schema、default、名前を検証する              | version、agent、install name の規則                        |  250–430 | schema と名前衝突を実行前に拒否できる                        |
| C06 | remote宣言の自前policyを検証する             | 空値、制御文字、credential、local path拒否                 |  180–300 | provider構文を解釈せず、manifest固有の危険だけ拒否できる     |
| K01 | `doctor` のCLI契約を承認する                 | grammar、flag、終了status、text/JSON schema                | 文書のみ | C07が参照する完全なinterfaceがCLI契約へ登録される            |
| C07 | remote 範囲の `doctor` を公開する            | 入力・公開CLI観測の診断、CLI テスト                        |  250–450 | container fixture に対して読み取り専用で成功・失敗を説明する |

J01はUTF-8、単一のJSON値、全階層の重複member拒否だけを共有する。J02はobject必須とknown fieldの
canonical spellingを共有し、unknown field、型、null、domainのpolicyは各readerに残す。C03が依存するのは
固定した `skills` の公開commandとmachine-readable outputだけとし、private module、private source parser、
global lockのpathまたはschemaを実装しない。C06はsourceをopaqueな実行宣言として保持し、外部CLIが
受理するprovider構文や異なる表記の意味的同一性を判定しない。

C04からC06の間では、不完全なmanifestをCLIの通常経路へ通さない。C07で公開する時点では、remote entryに
必要な自前policyと公開CLI観測の不足を説明する。ownershipの判定と変更可否はR01以降で追加する。

## フェーズ 3: remote のownershipと計画

| ID  | PR の責務                       | 主な成果物                                    |     目安 | 完了条件                                                    |
| --- | ------------------------------- | --------------------------------------------- | -------: | ----------------------------------------------------------- |
| R01 | ownership receiptを読み取る     | schema、XDG path、missing/invalid、宣言digest |  250–400 | private lockなしで合成receiptを読み取れる                   |
| R02 | installed treeをfingerprintする | path、file type、symlink、content digest      |  300–450 | receiptと現在のinstallを副作用なしで比較できる              |
| R03 | remote の基本状態を分類する     | install、unchanged、reconfigure               |  300–450 | desired、公開CLI観測、receipt、fingerprintの状態表が通る    |
| R04 | 異常・所有権状態を分類する      | conflict、untracked、曖昧な観測、名前正規化   |  300–450 | 証明できない対象を変更せず、明示的なstatusにする            |
| K02 | `plan` のCLI契約を承認する      | grammar、flag、終了status、text/JSON schema   | 文書のみ | R05が参照する完全なinterfaceがCLI契約へ登録される           |
| R05 | `plan` を公開する               | text/JSON 出力、終了条件、CLI テスト          |  250–450 | remote fixture に対して決定的なplanを読み取り専用で出力する |

receiptなしで存在するinstallはuntrackedとする。source宣言は完全一致またはcanonical manifestから計算した
digestで比較し、aliasやprovider URLのsemantic identityを導入しない。公開CLIのsource情報がnullまたは
曖昧な場合も、原因をprivate lockから推測せずconflict側へ閉じる。

## フェーズ 4: manifest 更新と remote apply

| ID  | PR の責務                           | 主な成果物                                     |     目安 | 完了条件                                                                 |
| --- | ----------------------------------- | ---------------------------------------------- | -------: | ------------------------------------------------------------------------ |
| M01 | 原子的な状態ファイル更新を実装する  | directory lock、temp file、snapshot 比較       |  350–480 | 同時更新、mode維持、途中失敗をunit testで検証する                        |
| K03 | `add`と`remove`のCLI契約を承認する  | grammar、flag、終了status、出力、副作用        | 文書のみ | M02とM03が参照する完全なinterfaceがCLI契約へ登録される                   |
| M02 | `add` を公開する                    | dry-run、`--yes`、remote entry更新             |  250–450 | 全入力検証後だけmanifestを置換し、外部installは行わない                  |
| K04 | `apply`のCLI契約を承認する          | grammar、flag、終了status、出力、prune確認     | 文書のみ | A01、A04、D03が参照する完全なinterfaceがCLI契約へ登録される              |
| A01 | install 引数とprocess境界を実装する | shellを介さないrunner、opaque sourceと引数生成 |  250–400 | manifest宣言を解釈せず配列引数で正しく渡す                               |
| A02 | 実行前intentを永続化する            | operation ID、事前観測、atomic write、recovery |  300–450 | process開始前に中断判定可能なintentが残る                                |
| A03 | install後のreceiptを確定する        | install、再観測、fingerprint、receipt commit   |  300–450 | 一意なpostcondition確認後だけownershipを確定する                         |
| A04 | pruneなしの `apply` を公開する      | `--yes`、preflight、process-wide apply lock    |  300–480 | fake CLIでremote installが収束し、中断後の再実行を安全に分類できる       |
| M03 | desired の `remove` を公開する      | dry-run、`--yes`、manifest entry除外           |  220–400 | workspace sourceやインストール済み内容を削除せず、manifestだけを更新する |

`add`、`remove` は desired state の編集、`apply` は observed state の変更として責務を分ける。

## フェーズ 5: prune

| ID  | PR の責務                       | 主な成果物                               |    目安 | 完了条件                                                |
| --- | ------------------------------- | ---------------------------------------- | ------: | ------------------------------------------------------- |
| D01 | remote 削除候補を安全に計画する | receipt/fingerprint guard、remove status | 250–450 | untracked、drift、pending intentをremoveにしない        |
| D02 | 削除と再観測を実装する          | remove intent、対象再確認、結果検証      | 300–450 | 対象が変化した後は削除せず、消失確認後だけreceiptを除く |
| D03 | `apply --prune` を公開する      | 明示 flag、確認、統合テスト              | 250–450 | `--prune --yes` の組み合わせだけが隔離環境で削除を行う  |

## フェーズ 6: workspace Skill

| ID  | PR の責務                                  | 主な成果物                                          |     目安 | 完了条件                                                              |
| --- | ------------------------------------------ | --------------------------------------------------- | -------: | --------------------------------------------------------------------- |
| W01 | Skill ツリーを安全に列挙する               | path、file type、symlink、digest、mode              |  300–450 | 一時ツリーだけを読み、installer 除外対象を同じ規則で扱う              |
| W02 | frontmatter の境界と name を検証する       | header、終端、name scalar                           |  250–420 | CRLF、未終端、非文字列 name の境界を fixture で固定する               |
| W03 | description と追加 field を検証する        | quoted/block scalar、top-level 制約                 |  300–450 | description の形式と unsupported nested 値を明示的に拒否する          |
| W04 | workspace path と copy を安全に扱う        | root containment、staging copy、mode                |  280–450 | root 外、symlink parent、特殊ファイルを copy しない                   |
| W05 | workspace manifest entry を有効にする      | `kind=workspace`、固定相対 path、directory 検査     |  250–430 | Skill 本体を移行先へ追加せず、合成 workspace を検証できる             |
| W06 | projection を読み書きする                  | XDG state path、strict decode、private atomic write |  300–450 | 実 HOME を使わず、missing・競合・壊れた state を検証できる            |
| W07 | インストールされた全 copy を検証する       | agent path 解決、symlink target、digest 比較        |  350–480 | canonical copy だけを信頼せず、観測可能な copy の一致を確認する       |
| W08 | workspace の基本 plan を追加する           | install、unchanged、content update                  |  300–450 | desired tree と install tree の状態表テストが通る                     |
| W09 | workspace の所有権 plan を追加する         | ownership conflict、unregistered、遷移              |  300–450 | manifest、receipt、projectionの曖昧な組み合わせを変更対象にしない     |
| W10 | workspace install を追加する               | source 再検証、外部 CLI 引数、失敗集約              |  280–450 | plan 後に source が変わった場合、process 実行前に停止する             |
| W11 | install 前の projection intent を記録する  | crash recovery、状態更新、失敗注入                  |  300–450 | 外部 process より前に再実行可能な所有権 intent が残る                 |
| W12 | remote/workspace 遷移を検証する            | 再観測、receipt確定、projection 除去                |  300–450 | remote の新しい所有権を確認する前に workspace 記録を捨てない          |
| W13 | workspace 削除後の状態を検証する           | prune 後の再観測、state cleanup                     |  300–450 | install が残る場合や対象が変わった場合は ownership を保持する         |
| K05 | workspace向けCLI拡張契約を承認する         | 既存commandの入力、status、text/JSON schema拡張     | 文書のみ | W14とW15が参照するinterface拡張がCLI契約へ登録される                  |
| W14 | `doctor` と `plan` を workspace 対応にする | 警告、未登録 directory、JSON 出力                   |  220–400 | schema v2 の remote/workspace 混在 fixture を読み取り専用で診断できる |
| W15 | workspace 対応 `apply` を公開する          | process-wide apply lock、phase結合、CLI test        |  280–450 | 途中失敗から再実行でき、同じ workspace の再実行が収束する             |

W01からW04は、Skill treeの独立した安全契約を一つずつ固定するため別PRとする。W15が終わるまでは、
workspace entryを含む `apply` を明示的に拒否する。remoteとworkspaceという二つのkindがそろうW15で、
共通pipelineを維持したままkind固有のplan/apply処理だけをStrategyとして抽出する。providerやsource形式を
Strategyのdispatch keyにしない。

## フェーズ 7: adopt

| ID  | PR の責務                                 | 主な成果物                                    |     目安 | 完了条件                                                  |
| --- | ----------------------------------------- | --------------------------------------------- | -------: | --------------------------------------------------------- |
| K06 | `adopt`のCLI契約を承認する                | grammar、全mode・flag、終了status、出力       | 文書のみ | T01からT06が参照する完全なinterfaceがCLI契約へ登録される  |
| T01 | installed Skill の採用入力を検証する      | kind、明示source、name正規化、agent override  |  280–430 | 推測せずremote/workspaceの採用要求を区別する              |
| T02 | `adopt PATH` の copy 計画を移す           | source 検証、dry-run、staging copy            |  300–450 | destination 上書きや symlink を拒否し、まだ commit しない |
| T03 | adopt 用 transaction を移す               | rename、snapshot 比較、rollback               |  300–450 | 失敗注入で manifest と tree の片方だけを commit しない    |
| T04 | `adopt PATH` を公開する                   | CLI flag、projection 更新、統合テスト         |  280–450 | copy と ownership 記録が同じ隔離 transaction で完了する   |
| T05 | `adopt --installed` の remote を公開する  | 明示source、fingerprint、manifest/receipt更新 |  250–430 | sourceを明示したuntracked remoteだけを採用する            |
| T06 | `adopt --installed --as workspace` を公開 | local copy、projection transaction            |  300–450 | 明示指定なしに local Skill を Git 側へコピーしない        |

## フェーズ 8: v1 互換

| ID  | PR の責務                         | 主な成果物                                   |     目安 | 完了条件                                               |
| --- | --------------------------------- | -------------------------------------------- | -------: | ------------------------------------------------------ |
| K07 | v1互換CLI契約を承認する           | `migrate`/`capture`のgrammar、flag、status   | 文書のみ | L01とL02が参照する完全なinterfaceがCLI契約へ登録される |
| L01 | `migrate` を移す                  | v1 読み取り、v2 candidate、dry-run、`--yes`  |  300–450 | workspace directory と衝突する場合は書き込まず停止する |
| L02 | schema v1 の `capture` を実装する | 明示sourceを持つcandidate、warning、競合検出 |  350–480 | private lockを参照せず、v1 fixtureだけを更新できる     |

## フェーズ 9: contract確認と切り替え

| ID  | PR の責務                                 | 主な成果物                                   |    目安 | 完了条件                                                 |
| --- | ----------------------------------------- | -------------------------------------------- | ------: | -------------------------------------------------------- |
| F01 | cross-machine相当の受け入れ試験を実装する | machine fixture、receipt/apply/pruneの収束   | 300–480 | 端末状態を模した二つの隔離rootでシナリオが通る           |
| F02 | CLI契約と配布物を監査する                 | 承認済みentry一覧、container smoke、利用文書 | 250–450 | 新規契約を追加せず、未実装command・flagがない            |
| F03 | 正式な切り替えを行う                      | release/cutover手順、rollback参照の終了条件  | 150–350 | contract確認の承認後にのみ配布し、rollback手順を確認する |

F03まで `.worktrees/skills` は変更・削除しない。rollback参照を除去する場合も、切り替え後の別PRとする。

## 各 PR の共通チェックリスト

- [ ] この PR が追加する利用者向けの振る舞いを一文で説明できる。
- [ ] 承認済みの製品契約と、利用した外部CLIの公開境界をPR本文へ記載した。
- [ ] 利用者向けcommandを変更するPRは、先に承認されたCLI契約entryを参照している。
- [ ] private module、private source parser、private global lockに依存していない。
- [ ] 未対応の入力を無視せず、明示的に拒否する。
- [ ] 手書きによる非テスト実装の追加行＋削除行が 500 行以下である。
- [ ] 実装と対応テストを同じ PR に含め、実装、テスト、fixture、生成物、文書を別集計した。
- [ ] 手書き総差分が 1,000 行を超える場合、分割要否をレビューした。
- [ ] `skills/**`、実 manifest、端末固有 state、認証情報を含まない。
- [ ] コンテナ内で format、unit test、対象となる integration test が成功する。
- [ ] 書き込みテストは一時 HOME と一時 XDG directory だけを使う。
- [ ] 既存 workflow と、直前までに移行済みの機能が成功する。
- [ ] 意図的な差異、保留事項、次の PR への引き継ぎを PR 本文へ記載した。
