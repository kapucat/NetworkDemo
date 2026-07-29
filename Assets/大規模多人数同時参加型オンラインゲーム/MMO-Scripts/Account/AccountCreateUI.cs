using TMPro;
using UnityEngine;

public class AccountCreateUI : MonoBehaviour
{
    [SerializeField] private TMP_InputField loginNameInputField;
    [SerializeField] private TMP_InputField passwordInputField;
    [SerializeField] private TMP_InputField confirmPasswordInputField;
    [SerializeField] private TMP_Text messageText;

    public void OnCreateButtonClicked()
    {
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

        ShowMessage("入力チェック成功");
    }

    private void ShowMessage(string message)
    {
        if (messageText != null)
        {
            messageText.text = message;
        }
    }
}