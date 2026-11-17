# Unity WebSocket切断処理の回帰テスト

リポジトリのルートで `./tests/unity-disconnect/Test.ps1` を実行する。
Visual Studio 2022のC#コンパイラーを使い、実際のMMOWebSocketClient.csとMMOSession.csをテスト用のUnity・WebSocket代替型と組み合わせて実行する。Unity本体や素材は不要。

管理者切断コード1008、予約済み再接続の抑止、メインスレッド上のログイン画面への遷移要求、ローカル認証情報の消去、通常切断の再接続、古い接続イベントの無視、破棄後の再接続抑止を確認する。
UnityStubs.csはこのコンソールテスト専用で、ゲームのAssetsには入れない。
テスト出力は一時フォルダに作成し、終了後に削除する。

実際のUnityシーン遷移・描画・ビルド版の確認は [確認手順](../../docs/MMO管理者切断.md) を参照。
