using TMPro;
using UnityEngine;
using UnityEngine.SceneManagement;

public class MMOMainUI : MonoBehaviour
{
    [SerializeField] private TMP_Text loginInfoText;

    private void Start()
    {
        if (MMOSession.Instance == null ||
            !MMOSession.Instance.IsLoggedIn)
        {
            SceneManager.LoadScene("LoginScene");
            return;
        }

        loginInfoText.text =
            $"ログイン中：{MMOSession.Instance.LoginName}\n" +
            $"Account ID：{MMOSession.Instance.AccountId}";
    }

    public void OnLogoutButtonClicked()
    {
        if (MMOSession.Instance != null)
        {
            MMOSession.Instance.Clear();
        }

        SceneManager.LoadScene("LoginScene");
    }
}