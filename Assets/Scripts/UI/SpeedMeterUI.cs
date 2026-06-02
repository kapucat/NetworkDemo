using UnityEngine;
using UnityEngine.UI;
using TMPro;

public class SpeedMeterUI : MonoBehaviour
{
    [SerializeField] private Rigidbody playerRigidbody;
    [SerializeField] private TextMeshProUGUI speedValueText;
    [SerializeField] private Image speedGaugeFill;

    [SerializeField] private float maxDisplaySpeed = 50f;
    [SerializeField] private float smoothSpeed = 10f;

    private float currentDisplaySpeed;

    private void Update()
    {
        if (playerRigidbody == null)
        {
            return;
        }

        Vector3 velocity = playerRigidbody.velocity;
        velocity.y = 0f;

        float targetSpeed = velocity.magnitude;

        currentDisplaySpeed = Mathf.Lerp(
            currentDisplaySpeed,
            targetSpeed,
            smoothSpeed * Time.deltaTime
        );

        if (speedValueText != null)
        {
            speedValueText.text = Mathf.RoundToInt(currentDisplaySpeed).ToString();
        }

        if (speedGaugeFill != null)
        {
            speedGaugeFill.fillAmount = Mathf.Clamp01(currentDisplaySpeed / maxDisplaySpeed);
        }
    }
}