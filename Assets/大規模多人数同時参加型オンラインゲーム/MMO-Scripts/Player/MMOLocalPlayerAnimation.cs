using UnityEngine;
using UnityEngine.AI;

[RequireComponent(typeof(NavMeshAgent))]
public class MMOLocalPlayerAnimation : MonoBehaviour
{
    [SerializeField]
    private Animator animator;

    private NavMeshAgent agent;

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
        agent = GetComponent<NavMeshAgent>();

        if (animator == null)
        {
            animator =
                GetComponentInChildren<Animator>(true);
        }

        if (animator != null)
        {
            animator.applyRootMotion = false;
        }
    }

    private void Update()
    {
        if (animator == null ||
            agent == null)
        {
            return;
        }

        Vector3 velocity =
            agent.velocity;

        velocity.y = 0f;

        float speed =
            velocity.magnitude;

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