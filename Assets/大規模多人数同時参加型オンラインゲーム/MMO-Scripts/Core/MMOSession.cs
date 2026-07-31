using UnityEngine;

public class MMOSession : MonoBehaviour
{
    public static MMOSession Instance { get; private set; }

    public long AccountId { get; private set; }
    public string LoginName { get; private set; } = "";

    public bool IsLoggedIn =>
        AccountId > 0 &&
        !string.IsNullOrWhiteSpace(LoginName);

    private void Awake()
    {
        if (Instance != null && Instance != this)
        {
            Destroy(gameObject);
            return;
        }

        Instance = this;
        DontDestroyOnLoad(gameObject);
    }

    public void SetAccount(long accountId, string loginName)
    {
        AccountId = accountId;
        LoginName = loginName;
    }

    public void Clear()
    {
        AccountId = 0;
        LoginName = "";
    }
}