using System;
using System.Collections;
using System.Text;
using UnityEngine;
using UnityEngine.Networking;

public class AccountApiClient : MonoBehaviour
{
    [SerializeField]
    private string registerUrl =
        "http://localhost:8080/api/accounts/register";

    [Serializable]
    private class RegisterRequest
    {
        public string login_name;
        public string password;
    }

    [Serializable]
    private class RegisterResponse
    {
        public string message;
        public long account_id;
    }

    public IEnumerator RegisterAccount(
        string loginName,
        string password,
        Action<bool, long, string> onCompleted)
    {
        RegisterRequest requestData = new RegisterRequest
        {
            login_name = loginName,
            password = password
        };

        string json = JsonUtility.ToJson(requestData);
        byte[] body = Encoding.UTF8.GetBytes(json);

        using (UnityWebRequest request =
               new UnityWebRequest(registerUrl, UnityWebRequest.kHttpVerbPOST))
        {
            request.uploadHandler = new UploadHandlerRaw(body);
            request.downloadHandler = new DownloadHandlerBuffer();
            request.SetRequestHeader("Content-Type", "application/json");
            request.timeout = 10;

            yield return request.SendWebRequest();

            RegisterResponse response = null;
            string responseText = request.downloadHandler.text;

            if (!string.IsNullOrWhiteSpace(responseText))
            {
                try
                {
                    response =
                        JsonUtility.FromJson<RegisterResponse>(responseText);
                }
                catch (Exception exception)
                {
                    Debug.LogWarning(
                        $"“o˜^API‚Ì‰ž“š‰ðÍ‚ÉŽ¸”s‚µ‚Ü‚µ‚½: {exception.Message}");
                }
            }

            bool success =
                request.responseCode >= 200 &&
                request.responseCode < 300;

            long accountId = response != null
                ? response.account_id
                : 0;

            string message = response != null
                ? response.message
                : "connection failed";

            if (!success)
            {
                Debug.LogWarning(
                    $"ƒAƒJƒEƒ“ƒg“o˜^Ž¸”s: " +
                    $"HTTP {request.responseCode} / {message}");
            }

            onCompleted?.Invoke(success, accountId, message);
        }
    }
}