using UnityEngine;
using UnityEngine.SceneManagement;

public class SceneLoader : MonoBehaviour
{
    public void LoadTitle()
    {
        SceneManager.LoadScene("TitleScene");
    }

    public void LoadGameStage01()
    {
        SceneManager.LoadScene("GameScene_Stage01");
    }

    public void LoadResult()
    {
        SceneManager.LoadScene("ResultScene");
    }

    public void LoadRanking()
    {
        SceneManager.LoadScene("RankingScene");
    }

    public void QuitGame()
    {
        Application.Quit();
    }
}