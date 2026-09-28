using UnityEngine;

public class MMOSession : MonoBehaviour
{
    public static MMOSession Instance { get; private set; }

    public long AccountId { get; private set; }
    public string LoginName { get; private set; } = "";
    public string SessionToken { get; private set; } = "";

    public bool IsLoggedIn =>
        AccountId > 0 &&
        !string.IsNullOrWhiteSpace(LoginName) &&
        !string.IsNullOrWhiteSpace(SessionToken);

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

    public void SetAccount(
        long accountId,
        string loginName,
        string sessionToken)
    {
        AccountId = accountId;
        LoginName = loginName;
        SessionToken = sessionToken;
    }

    public void Clear()
    {
        AccountId = 0;
        LoginName = "";
        SessionToken = "";
    }
}