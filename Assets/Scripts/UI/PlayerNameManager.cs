using UnityEngine;
using TMPro;

public class PlayerNameManager : MonoBehaviour
{
    [SerializeField] private TMP_InputField playerNameInput;

    private const string PlayerNameKey = "PlayerName";

    private void Start()
    {
        string savedName = PlayerPrefs.GetString(PlayerNameKey, "Player01");

        if (playerNameInput != null)
        {
            playerNameInput.text = savedName;
        }
    }

    public void SavePlayerName()
    {
        if (playerNameInput == null)
        {
            return;
        }

        string playerName = playerNameInput.text.Trim();

        if (string.IsNullOrEmpty(playerName))
        {
            playerName = "Player01";
        }

        PlayerPrefs.SetString(PlayerNameKey, playerName);
        PlayerPrefs.Save();
    }

    public static string GetPlayerName()
    {
        return PlayerPrefs.GetString(PlayerNameKey, "Player01");
    }
}