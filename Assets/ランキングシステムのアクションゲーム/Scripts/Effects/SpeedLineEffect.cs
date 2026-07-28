using System.Collections;
using UnityEngine;
using UnityEngine.UI;

public class SpeedLineEffect : MonoBehaviour
{
    [SerializeField] private Image speedLineImage;
    [SerializeField] private float maxAlpha = 0.4f;
    [SerializeField] private float fadeSpeed = 8f;

    [Header("Move Effect")]
    [SerializeField] private bool animateScale = true;
    [SerializeField] private float minScale = 1.0f;
    [SerializeField] private float maxScale = 1.12f;
    [SerializeField] private float scaleSpeed = 6f;

    private Coroutine effectCoroutine;
    private RectTransform rectTransform;

    private void Awake()
    {
        if (speedLineImage == null)
        {
            speedLineImage = GetComponent<Image>();
        }

        rectTransform = GetComponent<RectTransform>();

        if (speedLineImage != null)
        {
            Color color = speedLineImage.color;
            color.a = 0f;
            speedLineImage.color = color;
        }

        if (rectTransform != null)
        {
            rectTransform.localScale = Vector3.one * minScale;
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
        float timer = 0f;

        while (timer < duration)
        {
            timer += Time.deltaTime;

            float currentAlpha = Mathf.Lerp(
                speedLineImage.color.a,
                maxAlpha,
                Time.deltaTime * fadeSpeed
            );

            SetAlpha(currentAlpha);

            if (animateScale && rectTransform != null)
            {
                float scale = Mathf.Lerp(
                    minScale,
                    maxScale,
                    Mathf.PingPong(Time.time * scaleSpeed, 1f)
                );

                rectTransform.localScale = Vector3.one * scale;
            }

            yield return null;
        }

        while (speedLineImage.color.a > 0.01f)
        {
            float currentAlpha = Mathf.Lerp(
                speedLineImage.color.a,
                0f,
                Time.deltaTime * fadeSpeed
            );

            SetAlpha(currentAlpha);

            if (animateScale && rectTransform != null)
            {
                rectTransform.localScale = Vector3.one * minScale;
            }

            yield return null;
        }

        SetAlpha(0f);

        if (rectTransform != null)
        {
            rectTransform.localScale = Vector3.one * minScale;
        }

        effectCoroutine = null;
    }

    private void SetAlpha(float alpha)
    {
        Color color = speedLineImage.color;
        color.a = alpha;
        speedLineImage.color = color;
    }
}