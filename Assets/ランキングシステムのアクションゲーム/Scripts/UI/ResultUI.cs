using UnityEngine;
using TMPro;

public class ResultUI : MonoBehaviour
{
    [SerializeField] private TextMeshProUGUI clearTimeText;
    [SerializeField] private TextMeshProUGUI deathCountText;

    private void Start()
    {
        Cursor.lockState = CursorLockMode.None;
        Cursor.visible = true;

        clearTimeText.text = "Time: " + GameResultData.ClearTime.ToString("F2");
        deathCountText.text = "Death: " + GameResultData.DeathCount;
    }
}