using System.Collections;
using TMPro;
using UnityEngine;
using UnityEngine.UI;

public class LoginUI : MonoBehaviour
{
    [Header("入力欄")]
    [SerializeField] private TMP_InputField loginNameInputField;
    [SerializeField] private TMP_InputField passwordInputField;

    [Header("表示")]
    [SerializeField] private TMP_Text messageText;
    [SerializeField] private Button loginButton;

    [Header("API")]
    [SerializeField] private LoginApiClient loginApiClient;

    private bool isSending;

    public void OnLoginButtonClicked()
    {
        if (isSending)
        {
            return;
        }

        string loginName = loginNameInputField.text.Trim();
        string password = passwordInputField.text;

        if (string.IsNullOrWhiteSpace(loginName))
        {
            ShowMessage("ログイン名を入力してください");
            return;
        }

        if (string.IsNullOrEmpty(password))
        {
            ShowMessage("パスワードを入力してください");
            return;
        }

        StartCoroutine(Login(loginName, password));
    }

    private IEnumerator Login(string loginName, string password)
    {
        if (loginApiClient == null)
        {
            ShowMessage("APIの設定が見つかりません");
            yield break;
        }

        SetSending(true);
        ShowMessage("ログインしています...");

        bool success = false;
        long accountId = 0;
        string returnedLoginName = "";
        string serverMessage = "";

        yield return loginApiClient.Login(
            loginName,
            password,
            (result, id, name, message) =>
            {
                success = result;
                accountId = id;
                returnedLoginName = name;
                serverMessage = message;
            });

        SetSending(false);

        if (success)
        {
            ShowMessage(
                $"ログイン成功：{returnedLoginName} ID: {accountId}");

            passwordInputField.text = "";
            yield break;
        }

        switch (serverMessage)
        {
            case "invalid login_name or password":
                ShowMessage(
                    "ログイン名またはパスワードが違います");
                break;

            case "account is unavailable":
                ShowMessage(
                    "このアカウントは利用できません");
                break;

            case "connection failed":
                ShowMessage(
                    "サーバーへ接続できませんでした");
                break;

            default:
                ShowMessage("ログインに失敗しました");
                break;
        }
    }

    private void SetSending(bool sending)
    {
        isSending = sending;

        if (loginButton != null)
        {
            loginButton.interactable = !sending;
        }
    }

    private void ShowMessage(string message)
    {
        if (messageText != null)
        {
            messageText.text = message;
        }
    }
}