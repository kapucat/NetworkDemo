using UnityEngine;
using TMPro;

public class GameTimer : MonoBehaviour
{
    [SerializeField] private TextMeshProUGUI timerText;

    private float currentTime;
    private bool isRunning = true;

    private void Update()
    {
        if (!isRunning)
        {
            return;
        }

        currentTime += Time.deltaTime;

        if (timerText != null)
        {
            timerText.text = currentTime.ToString("F2");
        }
    }

    public float GetCurrentTime()
    {
        return currentTime;
    }

    public void StopTimer()
    {
        isRunning = false;
    }
}