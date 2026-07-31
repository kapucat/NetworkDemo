using System;
using System.Collections;
using System.Text;
using UnityEngine;
using UnityEngine.Networking;

public class LoginApiClient : MonoBehaviour
{
    [SerializeField]
    private string loginUrl =
        "http://localhost:8080/api/accounts/login";

    [Serializable]
    private class LoginRequest
    {
        public string login_name;
        public string password;
    }

    [Serializable]
    private class LoginResponse
    {
        public string message;
        public long account_id;
        public string login_name;
    }

    public IEnumerator Login(
        string loginName,
        string password,
        Action<bool, long, string, string> onCompleted)
    {
        LoginRequest requestData = new LoginRequest
        {
            login_name = loginName,
            password = password
        };

        string json = JsonUtility.ToJson(requestData);
        byte[] body = Encoding.UTF8.GetBytes(json);

        using UnityWebRequest request =
            new UnityWebRequest(loginUrl, UnityWebRequest.kHttpVerbPOST);

        request.uploadHandler = new UploadHandlerRaw(body);
        request.downloadHandler = new DownloadHandlerBuffer();
        request.SetRequestHeader("Content-Type", "application/json");
        request.timeout = 10;

        yield return request.SendWebRequest();

        string responseText = request.downloadHandler.text;
        LoginResponse response = null;

        if (!string.IsNullOrWhiteSpace(responseText))
        {
            try
            {
                response = JsonUtility.FromJson<LoginResponse>(responseText);
            }
            catch (Exception exception)
            {
                Debug.LogWarning(
                    $"ƒƒOƒCƒ“API‰ž“š‚Ì‰ðÍ‚ÉŽ¸”s‚µ‚Ü‚µ‚½: {exception.Message}");
            }
        }

        bool success =
            request.responseCode >= 200 &&
            request.responseCode < 300;

        long accountId = response != null
            ? response.account_id
            : 0;

        string returnedLoginName = response != null
            ? response.login_name
            : "";

        string message = response != null
            ? response.message
            : "connection failed";

        if (!success)
        {
            Debug.LogWarning(
                $"ƒƒOƒCƒ“Ž¸”s: HTTP {request.responseCode} / {message}");
        }

        onCompleted?.Invoke(
            success,
            accountId,
            returnedLoginName,
            message);
    }
}