<!-- markdownlint-disable MD013 -->

# 安全性と検証

## 現在の安全境界

最初のマイルストーンは読み取り専用である。`$HOME/.agents/skills` の一覧表示は、Skill、設定、lock、
state、Git repository を作成、変更、移動、削除してはならない。

テストは実 HOME を使用せず、テストごとの一時ディレクトリに合成した `.agents/skills` を作成する。
実在する Skill 本体、利用者固有 path、認証情報、実 lockfile を fixture としてコミットしない。

## テスト層

### Unit test

filesystem discovery を一時ディレクトリに対して検証する。N01 で決めた通常 entry、空 directory、
対象なし、読み取り不能、symlink、壊れた entry、決定的な並び順を、必要な範囲だけ table-driven test に
する。

### CLI test

process を起動し、argument、終了 status、stdout、stderr が承認済み契約と一致することを確認する。
実 HOME ではなく、テストが所有する root を使える内部境界を設ける。利用者向けに未承認の flag や
環境変数を公開してテスト注入を実現しない。

### Package smoke test

Go build、Nix package、コンテナから同じ CLI を起動し、読み取り専用の合成 fixture に対して結果が
一致することを確認する。

### 実環境での確認

N01 と実装が承認された後に限り、利用者が明示的に実行した一覧 command で実 `.agents/skills` を
読み取れる。自動テスト、CI、エージェントによる検証では実 HOME を走査しない。

## symlink と path

symlink を一覧へ含めるか、どこまで解決するか、壊れた symlink をどう報告するかは N01 で決める。
決定前に実装の偶然の挙動へ依存しない。

どの契約を選んでも、一覧表示のために root 外へ書き込まない。path を出力する場合は、利用者向けの
表示とテスト・ログでの絶対 path 露出を分けて検討する。

## 将来のコピー機能

コピー機能は、一覧表示や TUI と同じ安全契約にしない。destination 構造と衝突規則が決まった後に、
次の安全条件を具体化する。

- dry-run または同等の事前確認
- destination containment
- symlink と特殊ファイルの扱い
- staging と原子的な確定
- 既存 destination の保護
- 部分失敗と再実行

ライセンス適合性をツールが判定しないことと、意図しない path をコピーしないことは別の責務である。

## 基本検証

現在利用可能な経路について、少なくとも次を実行する。

- `gofmt` の差分がないこと
- `go vet ./...`
- `go test ./...`
- `nix flake check`
- `nix build .#skills-reconcile`
- コンテナ build と、合成 HOME に対する smoke test

Nix やコンテナを N04、N05 で変更する場合も、変更前後で利用可能な検証経路を明示し、壊れた状態を
次の PR に持ち越さない。

## 停止条件

次の場合は実装または merge を止め、契約または PR 境界を見直す。

- 一覧表示の仕様を実装者が推測する必要がある。
- 実 HOME や実 Skill がテストに必要になる。
- 読み取り機能がファイルを変更する。
- TUI、コピー、作者区分、manifest など未決の責務が最初の実装へ混ざる。
- symlink や読み取り不能 entry の扱いが契約とテストで一致しない。
- 手書き非テスト実装が 500 行を超える。
- 実装、テスト、fixture の手書き総差分が 1,000 行を超え、責務の再確認をしていない。
- Go、Nix、コンテナの現在有効な経路が同じ公開済み挙動を提供しない。
