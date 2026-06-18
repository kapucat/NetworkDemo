using System.Collections;
using UnityEngine;
using UnityEngine.UI;

public class SpeedLineEffect : MonoBehaviour
{
    [SerializeField] private Image speedLineImage;
    [SerializeField] private float maxAlpha = 0.6f;
    [SerializeField] private float fadeSpeed = 8f;

    private Coroutine effectCoroutine;

    private void Awake()
    {
        if (speedLineImage == null)
        {
            speedLineImage = GetComponent<Image>();
        }

        if (speedLineImage != null)
        {
            Color color = speedLineImage.color;
            color.a = 0f;
            speedLineImage.color = color;
        }
    }

    public void Play(float duration)
    {
        if (speedLineImage == null)
        {
            return;
        }

        if (effectCoroutine != null)
        {
            StopCoroutine(effectCoroutine);
        }

        effectCoroutine = StartCoroutine(PlayRoutine(duration));
    }

    private IEnumerator PlayRoutine(float duration)
    {
        while (speedLineImage.color.a < maxAlpha - 0.01f)
        {
            SetAlpha(Mathf.Lerp(speedLineImage.color.a, maxAlpha, Time.deltaTime * fadeSpeed));
            yield return null;
        }

        SetAlpha(maxAlpha);

        float timer = 0f;

        while (timer < duration)
        {
            timer += Time.deltaTime;
            yield return null;
        }

        while (speedLineImage.color.a > 0.01f)
        {
            SetAlpha(Mathf.Lerp(speedLineImage.color.a, 0f, Time.deltaTime * fadeSpeed));
            yield return null;
        }

        SetAlpha(0f);
        effectCoroutine = null;
    }

    private void SetAlpha(float alpha)
    {
        Color color = speedLineImage.color;
        color.a = alpha;
        speedLineImage.color = color;
    }
}