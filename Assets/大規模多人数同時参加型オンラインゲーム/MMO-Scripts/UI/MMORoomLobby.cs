using System;
using System.Linq;
using TMPro;
using UnityEngine;
using UnityEngine.AI;
using UnityEngine.UI;

// Builds its UI at runtime so the feature is also present in existing MMO scenes.
[RequireComponent(typeof(MMOWebSocketClient))]
public class MMORoomLobby : MonoBehaviour {
    private static MMORoomLobby current;
    public static bool MovementAllowed => current != null && current.client != null && current.client.CanMove;
    public static bool InRoom => current != null && current.client != null && current.client.Room != null;
    private MMOWebSocketClient client;
    private MMOChatUI chat;
    private NavMeshAgent agent;
    private readonly MMORoomClock clock = new();
    private MMORoomSummary[] rooms = Array.Empty<MMORoomSummary>();
    private Canvas canvas;
    private RectTransform panel, browser, roomView, list, toolbar;
    private TMP_Text status, countdown, members, playingLabel, pageLabel;
    private TMP_InputField roomName;
    private Button readyButton, previous, next;
    private TMP_FontAsset font;
    private bool wasPlaying;
    private float nextPing;
    private int page;

    private void Awake() {
        current = this; client = GetComponent<MMOWebSocketClient>();
        client.OnRoomMessage += Receive;
        client.OnWorldReset += ResetWorld;
    }
    private void Start() {
        chat = FindObjectOfType<MMOChatUI>();
        var movement = FindObjectOfType<MMOClickMove>();
        if (movement != null) agent = movement.GetComponent<NavMeshAgent>();
        font = FindObjectsOfType<TMP_Text>(true).Select(t => t.font)
            .FirstOrDefault(f => f != null && f.name.IndexOf("Noto", StringComparison.OrdinalIgnoreCase) >= 0);
        if (font == null) font = FindObjectsOfType<TMP_Text>(true).Select(t => t.font).FirstOrDefault(f => f != null);
        BuildUI(); Refresh();
    }
    private void OnDestroy() {
        if (current == this) current = null;
        if (client != null) { client.OnRoomMessage -= Receive; client.OnWorldReset -= ResetWorld; }
        if (canvas != null) Destroy(canvas.gameObject);
    }
    private void ResetWorld() {
        clock.Reset(); page = 0;
        if (chat != null) chat.ClearHistory();
        if (status != null) status.text = client.IsConnected ? "部屋を作成、または参加してください。" : "再接続中です。接続後に部屋を選び直してください。";
        Refresh();
    }
    private void Receive(MMORoomPacket packet) {
        clock.Observe(packet.server_unix_ms, Time.realtimeSinceStartupAsDouble * 1000, packet.client_unix_ms);
        if (packet.type == "room.list") { rooms = packet.rooms ?? Array.Empty<MMORoomSummary>(); RenderRooms(); }
        if (packet.type == "room.error" && status != null) status.text = packet.message;
        if (packet.type == "room.state" && status != null) status.text = "";
        Refresh();
    }
    private void Update() {
        bool playing = client != null && client.CanMove;
        if (agent != null && agent.enabled && agent.isOnNavMesh) {
            if (!playing && (!agent.isStopped || agent.hasPath)) { agent.isStopped = true; agent.ResetPath(); }
            if (playing && !wasPlaying) agent.isStopped = false;
        }
        wasPlaying = playing;
        if (client.IsConnected && Time.unscaledTime >= nextPing) {
            nextPing = Time.unscaledTime + 10; client.SendRoom("room.ping");
        }
        if (countdown != null && client.Room != null && client.Room.phase == "countdown") {
            double remaining = (client.Room.start_at_unix_ms - clock.ServerNow(Time.realtimeSinceStartupAsDouble * 1000)) / 1000;
            countdown.text = remaining > 0 ? "開始まで " + Math.Ceiling(remaining).ToString("0") : "開始を待っています…";
        }
    }
    private void Refresh() {
        if (panel == null) return;
        var room = client.Room;
        bool playing = client.CanMove;
        panel.gameObject.SetActive(!playing);
        toolbar.gameObject.SetActive(playing);
        browser.gameObject.SetActive(room == null);
        roomView.gameObject.SetActive(room != null);
        if (room == null) return;
        playingLabel.text = "部屋：" + room.name + "　" + (room.members?.Length ?? 0) + "/4人";
        members.text = "部屋：" + room.name + "\n\n" + string.Join("\n", (room.members ?? Array.Empty<MMORoomMember>())
            .Select(m => m.login_name + "　：　" + (m.ready ? "準備完了" : "準備中")));
        var me = (room.members ?? Array.Empty<MMORoomMember>()).FirstOrDefault(m => MMOSession.Instance != null && m.account_id == MMOSession.Instance.AccountId);
        readyButton.GetComponentInChildren<TMP_Text>().text = me != null && me.ready ? "準備を取り消す" : "準備完了";
        readyButton.interactable = client.IsConnected;
        if (room.phase != "countdown") countdown.text = (room.members?.Length ?? 0) < 2 ? "開始には2人以上必要です。" : "全員が準備完了になると開始します。";
    }
    private RectTransform Box(string name, Transform parent) {
        var go = new GameObject(name, typeof(RectTransform)); go.transform.SetParent(parent, false); return (RectTransform)go.transform;
    }
    private void Fixed(RectTransform rect, float width, float height) { rect.sizeDelta = new Vector2(width, height); }
    private TMP_Text Text(Transform parent, string text, int size = 26) {
        var t = Box("Text", parent).gameObject.AddComponent<TextMeshProUGUI>();
        t.font = font; t.fontSize = size; t.text = text; t.color = Color.white; t.richText = false;
        t.alignment = TextAlignmentOptions.MidlineLeft; t.enableWordWrapping = true;
        return t;
    }
    private void RowSize(Component item, float height) { item.gameObject.AddComponent<LayoutElement>().preferredHeight = height; }
    private Button Button(Transform parent, string label, UnityEngine.Events.UnityAction action) {
        var rect = Box(label, parent); rect.gameObject.AddComponent<Image>().color = new Color(.02f,.46f,.50f);
        var b = rect.gameObject.AddComponent<Button>(); b.onClick.AddListener(action); b.targetGraphic = rect.GetComponent<Image>();
        var t = Text(rect, label, 24); t.alignment = TextAlignmentOptions.Center; Stretch(t.rectTransform, 8);
        RowSize(b, 48); return b;
    }
    private void Stretch(RectTransform rect, float inset = 0) {
        rect.anchorMin = Vector2.zero; rect.anchorMax = Vector2.one; rect.offsetMin = Vector2.one * inset; rect.offsetMax = -Vector2.one * inset;
    }
    private void Vertical(RectTransform rect, int padding = 0) {
        var l = rect.gameObject.AddComponent<VerticalLayoutGroup>(); l.padding = new RectOffset(padding,padding,padding,padding);
        l.spacing = 12; l.childControlWidth = true; l.childControlHeight = true; l.childForceExpandWidth = true; l.childForceExpandHeight = false;
    }
    private RectTransform Horizontal(Transform parent, float height) {
        var r = Box("Row", parent); RowSize(r,height);
        var l = r.gameObject.AddComponent<HorizontalLayoutGroup>(); l.spacing = 12;
        l.childControlWidth = true; l.childControlHeight = true; l.childForceExpandWidth = true;
        return r;
    }
    private void RenderRooms() {
        if (list == null) return;
        foreach (Transform child in list) { child.gameObject.SetActive(false); Destroy(child.gameObject); }
        int pages = Math.Max(1,(rooms.Length + 4) / 5); page = Mathf.Clamp(page,0,pages-1);
        pageLabel.text = (page+1) + " / " + pages;
        previous.interactable = page > 0; next.interactable = page < pages-1;
        if (rooms.Length == 0) { RowSize(Text(list,"部屋がありません。新しく作成できます。",24),60); return; }
        foreach (var room in rooms.Skip(page*5).Take(5)) {
            var entry = room;
            var b = Button(list, room.name + "　" + room.count + "/" + room.capacity + "人　" +
                (room.phase == "waiting" ? "参加する" : room.phase == "countdown" ? "開始準備中" : "プレイ中"),
                () => client.SendRoom("room.join",roomId:entry.id));
            b.interactable = client.IsConnected && room.phase == "waiting" && room.count < room.capacity;
        }
    }
    private void BuildUI() {
        canvas = new GameObject("Room UI", typeof(RectTransform),typeof(Canvas),typeof(CanvasScaler),typeof(GraphicRaycaster)).GetComponent<Canvas>();
        canvas.renderMode = RenderMode.ScreenSpaceOverlay; canvas.sortingOrder = 100;
        var scaler = canvas.GetComponent<CanvasScaler>(); scaler.uiScaleMode = CanvasScaler.ScaleMode.ScaleWithScreenSize;
        scaler.referenceResolution = new Vector2(1920,1080); scaler.screenMatchMode = CanvasScaler.ScreenMatchMode.Expand;
        panel = Box("Lobby",canvas.transform); Fixed(panel,820,850);
        panel.gameObject.AddComponent<Image>().color = new Color(.06f,.11f,.16f,.98f); Vertical(panel,28);
        RowSize(Text(panel,"MMO ルーム",34),48);
        status = Text(panel,"部屋を作成、または参加してください。",22); RowSize(status,56);
        browser = Box("Browser",panel); Vertical(browser); RowSize(browser,640);
        var inputRect = Box("Room name",browser); RowSize(inputRect,48);
        inputRect.gameObject.AddComponent<Image>().color = Color.white;
        roomName = inputRect.gameObject.AddComponent<TMP_InputField>(); roomName.characterLimit = 20;
        var inputText = Text(inputRect,"",24); inputText.color = Color.black; Stretch(inputText.rectTransform,10);
        roomName.textComponent = inputText; roomName.textViewport = inputRect;
        var placeholder = Text(inputRect,"部屋名（1～20文字）",24); placeholder.color = Color.gray; Stretch(placeholder.rectTransform,10); roomName.placeholder = placeholder;
        var actions = Horizontal(browser,48);
        Button(actions,"部屋を作成",() => client.SendRoom("room.create",roomName.text));
        Button(actions,"一覧を更新",() => client.SendRoom("room.list"));
        list = Box("Rooms",browser); Vertical(list); RowSize(list,320);
        var pagination = Horizontal(browser,44);
        previous = Button(pagination,"前へ",() => {page--;RenderRooms();});
        pageLabel = Text(pagination,"1 / 1",22); pageLabel.alignment = TextAlignmentOptions.Center;
        next = Button(pagination,"次へ",() => {page++;RenderRooms();});
        roomView = Box("Room",panel); Vertical(roomView); RowSize(roomView,600);
        members = Text(roomView,"",28); RowSize(members,280);
        countdown = Text(roomView,"",30); RowSize(countdown,90);
        readyButton = Button(roomView,"準備完了",() => {
            var me = client.Room?.members?.FirstOrDefault(m => MMOSession.Instance != null && m.account_id == MMOSession.Instance.AccountId);
            client.SendRoom("room.ready",ready:me == null || !me.ready);
        });
        Button(roomView,"部屋から退出",() => client.SendRoom("room.leave"));
        toolbar = Box("Playing room",canvas.transform); Fixed(toolbar,820,64); toolbar.anchorMin = toolbar.anchorMax = new Vector2(.5f,1); toolbar.pivot = new Vector2(.5f,1); toolbar.anchoredPosition = new Vector2(0,-12);
        toolbar.gameObject.AddComponent<Image>().color = new Color(.06f,.11f,.16f,.96f);
        var layout = toolbar.gameObject.AddComponent<HorizontalLayoutGroup>(); layout.padding = new RectOffset(12,12,8,8); layout.spacing = 12; layout.childForceExpandWidth = true; layout.childControlWidth = true; layout.childControlHeight = true;
        playingLabel = Text(toolbar,""); Button(toolbar,"部屋から退出",() => client.SendRoom("room.leave"));
        RenderRooms();
    }
}
