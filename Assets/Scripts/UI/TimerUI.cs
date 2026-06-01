using UnityEngine;
using TMPro;

public class TimerUI : MonoBehaviour
{
    private float time;

    private void Update()
    {
        time += Time.deltaTime;
    }

    public float GetTime()
    {
        return time;
    }
}