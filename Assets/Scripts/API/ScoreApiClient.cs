using System.Collections;
using System.Text;
using TMPro;
using UnityEngine;
using UnityEngine.Networking;

public class ScoreApiClient : MonoBehaviour
{
    [SerializeField] private string baseUrl = "http://localhost:8080";
    [SerializeField] private bool checkHealthOnStart = false;
    [SerializeField] private bool sendScoreOnStart = false;
    [SerializeField] private TextMeshProUGUI bestTimeText;

    [System.Serializable]
    private class ScoreRequest
    {
        public string player_name;
        public int stage_id;
        public float clear_time;
        public int death_count;
    }

    [System.Serializable]
    private class BestResponse
    {
        public string player_name;
        public int stage_id;
        public float best_time;
        public int death_count;
    }

    private void Start()
    {
        if (checkHealthOnStart)
        {
            StartCoroutine(CheckHealth());
        }

        if (sendScoreOnStart)
        {
            SendCurrentScore();
        }
    }

    public void SendCurrentScore()
    {
        ScoreRequest score = new ScoreRequest
        {
            player_name = PlayerNameManager.GetPlayerName(),
            stage_id = 1,
            clear_time = GameResultData.ClearTime,
            death_count = GameResultData.DeathCount
        };

        StartCoroutine(PostScore(score));
    }

    private IEnumerator CheckHealth()
    {
        string url = baseUrl + "/health";

        using UnityWebRequest request = UnityWebRequest.Get(url);

        yield return request.SendWebRequest();

        if (request.result == UnityWebRequest.Result.Success)
        {
            Debug.Log("Health check success: " + request.downloadHandler.text);
        }
        else
        {
            Debug.LogError("Health check failed: " + request.error);
        }
    }

    private IEnumerator PostScore(ScoreRequest score)
    {
        string url = baseUrl + "/api/scores";
        string json = JsonUtility.ToJson(score);

        byte[] bodyRaw = Encoding.UTF8.GetBytes(json);

        using UnityWebRequest request = new UnityWebRequest(url, "POST");
        request.uploadHandler = new UploadHandlerRaw(bodyRaw);
        request.downloadHandler = new DownloadHandlerBuffer();
        request.SetRequestHeader("Content-Type", "application/json");

        Debug.Log("Send score JSON: " + json);

        yield return request.SendWebRequest();

        if (request.result == UnityWebRequest.Result.Success)
        {
            Debug.Log("Score send success: " + request.downloadHandler.text);

            StartCoroutine(GetBestScore(score.player_name, score.stage_id));
        }
        else
        {
            Debug.LogError("Score send failed: " + request.error);

            if (bestTimeText != null)
            {
                bestTimeText.text = "Best: Load failed";
            }
        }
    }

    private IEnumerator GetBestScore(string playerName, int stageId)
    {
        string url = baseUrl + "/api/best?player_name=" + UnityWebRequest.EscapeURL(playerName) + "&stage_id=" + stageId;

        using UnityWebRequest request = UnityWebRequest.Get(url);

        yield return request.SendWebRequest();

        if (request.result == UnityWebRequest.Result.Success)
        {
            Debug.Log("Best score get success: " + request.downloadHandler.text);

            BestResponse best = JsonUtility.FromJson<BestResponse>(request.downloadHandler.text);

            if (bestTimeText != null)
            {
                bestTimeText.text = "Best: " + best.best_time.ToString("F2") + "s";
            }
        }
        else
        {
            Debug.LogError("Best score get failed: " + request.error);

            if (bestTimeText != null)
            {
                bestTimeText.text = "Best: None";
            }
        }
    }
}