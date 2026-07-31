using System.Collections;
using System.Text.RegularExpressions;
using TMPro;
using UnityEngine;
using UnityEngine.UI;

public class AccountCreateUI : MonoBehaviour
{
    [Header("入力欄")]
    [SerializeField] private TMP_InputField loginNameInputField;
    [SerializeField] private TMP_InputField passwordInputField;
    [SerializeField] private TMP_InputField confirmPasswordInputField;

    [Header("表示")]
    [SerializeField] private TMP_Text messageText;
    [SerializeField] private Button createButton;

    [Header("API")]
    [SerializeField] private AccountApiClient accountApiClient;

    private bool isSending;

    public void OnCreateButtonClicked()
    {
        if (isSending)
        {
            return;
        }

        string loginName = loginNameInputField.text.Trim();
        string password = passwordInputField.text;
        string confirmPassword = confirmPasswordInputField.text;

        if (string.IsNullOrWhiteSpace(loginName))
        {
            ShowMessage("ログイン名を入力してください");
            return;
        }

        if (loginName.Length < 3 || loginName.Length > 20)
        {
            ShowMessage("ログイン名は3～20文字で入力してください");
            return;
        }

        if (!Regex.IsMatch(loginName, @"^[a-zA-Z0-9_]+$"))
        {
            ShowMessage(
                "ログイン名は半角英数字とアンダースコアのみ使用できます");
            return;
        }

        if (string.IsNullOrEmpty(password))
        {
            ShowMessage("パスワードを入力してください");
            return;
        }

        if (password.Length < 8 || password.Length > 64)
        {
            ShowMessage("パスワードは8～64文字で入力してください");
            return;
        }

        if (password != confirmPassword)
        {
            ShowMessage("確認用パスワードが一致しません");
            return;
        }

        StartCoroutine(RegisterAccount(loginName, password));
    }

    private IEnumerator RegisterAccount(
        string loginName,
        string password)
    {
        if (accountApiClient == null)
        {
            ShowMessage("APIの設定が見つかりません");
            yield break;
        }

        SetSending(true);
        ShowMessage("アカウントを作成しています...");

        bool success = false;
        long accountId = 0;
        string serverMessage = "";

        yield return accountApiClient.RegisterAccount(
            loginName,
            password,
            (result, id, message) =>
            {
                success = result;
                accountId = id;
                serverMessage = message;
            });

        SetSending(false);

        if (success)
        {
            ShowMessage(
                $"アカウントを作成しました ID: {accountId}");

            passwordInputField.text = "";
            confirmPasswordInputField.text = "";
            yield break;
        }

        switch (serverMessage)
        {
            case "login_name already exists":
                ShowMessage(
                    "そのログイン名はすでに使用されています");
                break;

            case "connection failed":
                ShowMessage(
                    "サーバーへ接続できませんでした");
                break;

            default:
                ShowMessage(
                    "アカウント作成に失敗しました");
                break;
        }
    }

    private void SetSending(bool sending)
    {
        isSending = sending;

        if (createButton != null)
        {
            createButton.interactable = !sending;
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