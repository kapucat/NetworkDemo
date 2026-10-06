using UnityEngine;

public class MMORemotePlayer : MonoBehaviour
{
    [Header("移動補間")]
    [SerializeField]
    private float moveSmooth = 12f;

    [SerializeField]
    private float rotationSmooth = 12f;

    [Header("アニメーション")]
    [SerializeField]
    private Animator animator;

    private Vector3 targetPosition;
    private Quaternion targetRotation;

    private bool hasNetworkState;

    private static readonly int SpeedHash =
        Animator.StringToHash("Speed");

    private static readonly int IsGroundedHash =
        Animator.StringToHash("IsGrounded");

    private static readonly int IsJumpingHash =
        Animator.StringToHash("IsJumping");

    private static readonly int IsFallingHash =
        Animator.StringToHash("IsFalling");

    private void Awake()
    {
        if (animator == null)
        {
            animator =
                GetComponentInChildren<Animator>(true);
        }

        if (animator != null)
        {
            animator.applyRootMotion = false;

            animator.SetBool(
                IsGroundedHash,
                true);

            animator.SetBool(
                IsJumpingHash,
                false);

            animator.SetBool(
                IsFallingHash,
                false);
        }
    }

    private void Update()
    {
        if (!hasNetworkState)
        {
            SetAnimationSpeed(0f);
            return;
        }

        Vector3 beforePosition =
            transform.position;

        float moveT =
            1f - Mathf.Exp(
                -moveSmooth * Time.deltaTime);

        float rotationT =
            1f - Mathf.Exp(
                -rotationSmooth * Time.deltaTime);

        transform.position =
            Vector3.Lerp(
                transform.position,
                targetPosition,
                moveT);

        transform.rotation =
            Quaternion.Slerp(
                transform.rotation,
                targetRotation,
                rotationT);

        Vector3 movement =
            transform.position - beforePosition;

        movement.y = 0f;

        float speed = 0f;

        if (Time.deltaTime > 0f)
        {
            speed =
                movement.magnitude /
                Time.deltaTime;
        }

        SetAnimationSpeed(speed);
    }

    public void SetNetworkState(
        Vector3 position,
        float rotationY)
    {
        if (!hasNetworkState)
        {
            transform.position =
                position;

            transform.rotation =
                Quaternion.Euler(
                    0f,
                    rotationY,
                    0f);

            targetPosition =
                position;

            targetRotation =
                transform.rotation;

            hasNetworkState = true;

            return;
        }

        targetPosition =
            position;

        targetRotation =
            Quaternion.Euler(
                0f,
                rotationY,
                0f);
    }

    private void SetAnimationSpeed(float speed)
    {
        if (animator == null)
        {
            return;
        }

        animator.SetFloat(
            SpeedHash,
            speed,
            0.1f,
            Time.deltaTime);

        animator.SetBool(
            IsGroundedHash,
            true);

        animator.SetBool(
            IsJumpingHash,
            false);

        animator.SetBool(
            IsFallingHash,
            false);
    }
}