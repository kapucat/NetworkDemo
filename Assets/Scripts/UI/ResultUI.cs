using UnityEngine;
using TMPro;

public class ResultUI : MonoBehaviour
{
    [SerializeField] private TextMeshProUGUI clearTimeText;

    private void Start()
    {
        clearTimeText.text = "Time: " + GameResultData.ClearTime.ToString("F2");
    }
}