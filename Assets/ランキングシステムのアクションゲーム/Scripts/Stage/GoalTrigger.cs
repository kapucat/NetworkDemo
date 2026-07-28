using UnityEngine;
using UnityEngine.SceneManagement;

public class GoalTrigger : MonoBehaviour
{
    [SerializeField] private GameTimer gameTimer;

    private void OnTriggerEnter(Collider other)
    {
        if (!other.CompareTag("Player"))
        {
            return;
        }

        GameResultData.ClearTime = gameTimer.GetCurrentTime();
        gameTimer.StopTimer();

        SceneManager.LoadScene("ResultScene");
    }
}