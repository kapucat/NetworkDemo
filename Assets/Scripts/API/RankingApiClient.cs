using System.Collections;
using System.Text;
using TMPro;
using UnityEngine;
using UnityEngine.Networking;

public class RankingApiClient : MonoBehaviour
{
    [SerializeField] private string baseUrl = "http://localhost:8080";
    [SerializeField] private TextMeshProUGUI rankingListText;
    [SerializeField] private int stageId = 1;

    [System.Serializable]
    private class RankingData
    {
        public int rank;
        public string player_name;
        public float clear_time;
        public int death_count;
    }

    [System.Serializable]
    private class RankingList
    {
        public RankingData[] items;
    }

    private void Start()
    {
        StartCoroutine(GetRankings());
    }

    private IEnumerator GetRankings()
    {
        string url = baseUrl + "/api/rankings?stage_id=" + stageId;

        using UnityWebRequest request = UnityWebRequest.Get(url);

        yield return request.SendWebRequest();

        if (request.result == UnityWebRequest.Result.Success)
        {
            Debug.Log("Ranking get success: " + request.downloadHandler.text);

            string json = request.downloadHandler.text;

            // UnityÇÃJsonUtilityÇÕîzóÒÇæÇØÇÃJSONÇíºê⁄ì«ÇﬂÇ»Ç¢ÇÃÇ≈ÅAitemsÇ≈ïÔÇﬁ
            string wrappedJson = "{\"items\":" + json + "}";

            RankingList rankingList = JsonUtility.FromJson<RankingList>(wrappedJson);

            ShowRankings(rankingList.items);
        }
        else
        {
            Debug.LogError("Ranking get failed: " + request.error);

            if (rankingListText != null)
            {
                rankingListText.text = "Ranking load failed";
            }
        }
    }

    private void ShowRankings(RankingData[] rankings)
    {
        if (rankingListText == null)
        {
            Debug.LogError("rankingListText is not assigned.");
            return;
        }

        StringBuilder builder = new StringBuilder();

        foreach (RankingData ranking in rankings)
        {
            builder.AppendLine(
                ranking.rank + ". " +
                ranking.player_name + "  " +
                ranking.clear_time.ToString("F2") + "s  " +
                "Death: " + ranking.death_count
            );
        }

        rankingListText.text = builder.ToString();
    }
}