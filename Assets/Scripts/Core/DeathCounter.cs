using UnityEngine;

public class DeathCounter : MonoBehaviour
{
    private int deathCount;

    private void Start()
    {
        deathCount = 0;
        GameResultData.DeathCount = 0;
    }

    public void AddDeath()
    {
        deathCount++;
        GameResultData.DeathCount = deathCount;
    }
}