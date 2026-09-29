using System.Collections;
using System.Collections.Generic;
using TMPro;
using UnityEngine;
using UnityEngine.EventSystems;
using UnityEngine.InputSystem;
using UnityEngine.UI;

public class MMOChatUI : MonoBehaviour
{
    [Header("チャットUI")]
    [SerializeField]
    private GameObject chatInputArea;

    [SerializeField]
    private TMP_InputField chatInputField;

    [SerializeField]
    private TMP_Text chatLogText;

    [SerializeField]
    private ScrollRect chatScrollRect;

    [Header("通信")]
    [SerializeField]
    private MMOWebSocketClient webSocketClient;

    [Header("表示設定")]
    [SerializeField]
    private int maxMessageCount = 50;

    [SerializeField]
    private float sendInterval = 0.15f;

    private readonly List<string> messages = new();

    private Coroutine scrollCoroutine;
    private float nextSendTime;

    public bool IsInputActive { get; private set; }

    private void Awake()
    {
        if (chatLogText != null)
        {
            chatLogText.text = "";
            chatLogText.richText = false;
        }

        SetInputVisible(false);
    }

    private void OnEnable()
    {
        if (webSocketClient != null)
        {
            webSocketClient.OnChatMessageReceived +=
                HandleChatMessageReceived;
        }

        if (chatInputField != null)
        {
            chatInputField.onSubmit.AddListener(
                HandleInputSubmit);
        }
    }

    private void OnDisable()
    {
        if (webSocketClient != null)
        {
            webSocketClient.OnChatMessageReceived -=
                HandleChatMessageReceived;
        }

        if (chatInputField != null)
        {
            chatInputField.onSubmit.RemoveListener(
                HandleInputSubmit);
        }
    }

    private void Update()
    {
        if (Keyboard.current == null)
        {
            return;
        }

        if (!IsInputActive &&
            (Keyboard.current.enterKey.wasPressedThisFrame ||
             Keyboard.current.numpadEnterKey.wasPressedThisFrame))
        {
            OpenChatInput();
            return;
        }

        if (IsInputActive &&
            Keyboard.current.escapeKey.wasPressedThisFrame)
        {
            CloseChatInput();
        }
    }

    private void OpenChatInput()
    {
        if (chatInputField == null)
        {
            return;
        }

        IsInputActive = true;

        SetInputVisible(true);

        chatInputField.text = "";

        chatInputField.Select();
        chatInputField.ActivateInputField();
    }

    private void CloseChatInput()
    {
        IsInputActive = false;

        if (chatInputField != null)
        {
            chatInputField.text = "";
            chatInputField.DeactivateInputField();
        }

        SetInputVisible(false);

        if (EventSystem.current != null)
        {
            EventSystem.current.SetSelectedGameObject(null);
        }
    }

    private void SetInputVisible(bool visible)
    {
        if (chatInputArea != null)
        {
            chatInputArea.SetActive(visible);
        }
    }

    private void HandleInputSubmit(string inputText)
    {
        string message = inputText.Trim();

        if (string.IsNullOrWhiteSpace(message))
        {
            CloseChatInput();
            return;
        }

        if (TrySendChat(message))
        {
            CloseChatInput();
        }
    }

    private bool TrySendChat(string message)
    {
        if (Time.unscaledTime < nextSendTime)
        {
            return false;
        }

        if (message.Length > 200)
        {
            AppendSystemMessage(
                "メッセージは200文字以内で入力してください");
            return false;
        }

        if (webSocketClient == null ||
            !webSocketClient.IsConnected)
        {
            AppendSystemMessage(
                "サーバーへ接続されていません");
            return false;
        }

        nextSendTime =
            Time.unscaledTime + sendInterval;

        webSocketClient.SendChat(message);

        return true;
    }

    private void HandleChatMessageReceived(
        string loginName,
        string message,
        string sentAt)
    {
        string displayName =
            string.IsNullOrWhiteSpace(loginName)
                ? "Unknown"
                : loginName;

        AppendMessage(
            $"{displayName}：{message}");
    }

    private void AppendSystemMessage(string message)
    {
        AppendMessage(
            $"[SYSTEM] {message}");
    }

    private void AppendMessage(string message)
    {
        bool shouldAutoScroll = IsNearBottom();

        messages.Add(message);

        while (messages.Count > maxMessageCount)
        {
            messages.RemoveAt(0);
        }

        if (chatLogText != null)
        {
            chatLogText.text =
                string.Join("\n", messages);
        }

        if (shouldAutoScroll &&
            chatScrollRect != null)
        {
            if (scrollCoroutine != null)
            {
                StopCoroutine(scrollCoroutine);
            }

            scrollCoroutine =
                StartCoroutine(
                    ScrollToBottom());
        }
    }

    private bool IsNearBottom()
    {
        if (chatScrollRect == null ||
            chatScrollRect.content == null ||
            chatScrollRect.viewport == null)
        {
            return true;
        }

        bool contentFits =
            chatScrollRect.content.rect.height <=
            chatScrollRect.viewport.rect.height + 1f;

        return contentFits ||
               chatScrollRect.verticalNormalizedPosition <= 0.05f;
    }

    private IEnumerator ScrollToBottom()
    {
        yield return null;

        Canvas.ForceUpdateCanvases();

        LayoutRebuilder.ForceRebuildLayoutImmediate(
            chatScrollRect.content);

        Canvas.ForceUpdateCanvases();

        chatScrollRect.verticalNormalizedPosition = 0f;

        scrollCoroutine = null;
    }
}