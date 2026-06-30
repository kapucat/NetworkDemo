using UnityEngine;

[RequireComponent(typeof(Rigidbody))]
public class PlayerAnimationController : MonoBehaviour
{
    [Header("References")]
    [SerializeField] private Animator animator;
    [SerializeField] private CapsuleCollider capsuleCollider;

    [Header("Ground Check")]
    [SerializeField] private LayerMask groundLayer = ~0;
    [SerializeField] private float groundCheckRadius = 0.25f;
    [SerializeField] private float groundCheckDistance = 0.3f;
    [SerializeField] private bool drawDebug = true;

    private Rigidbody rb;
    private readonly RaycastHit[] groundHits = new RaycastHit[8];

    private static readonly int SpeedHash = Animator.StringToHash("Speed");
    private static readonly int IsGroundedHash = Animator.StringToHash("IsGrounded");
    private static readonly int IsJumpingHash = Animator.StringToHash("IsJumping");
    private static readonly int IsFallingHash = Animator.StringToHash("IsFalling");

    private void Awake()
    {
        rb = GetComponent<Rigidbody>();

        if (animator == null)
        {
            animator = GetComponentInChildren<Animator>();
        }

        if (capsuleCollider == null)
        {
            capsuleCollider = GetComponent<CapsuleCollider>();
        }

        if (animator != null)
        {
            animator.applyRootMotion = false;
        }
    }

    private void Update()
    {
        if (animator == null || rb == null)
        {
            return;
        }

        Vector3 velocity = rb.velocity;
        Vector3 horizontalVelocity = new Vector3(velocity.x, 0f, velocity.z);

        float speed = horizontalVelocity.magnitude;
        bool isGrounded = CheckGrounded();
        bool isJumping = velocity.y > 0.1f;
        bool isFalling = !isGrounded && velocity.y < -0.1f;

        animator.SetFloat(SpeedHash, speed);
        animator.SetBool(IsGroundedHash, isGrounded);
        animator.SetBool(IsJumpingHash, isJumping);
        animator.SetBool(IsFallingHash, isFalling);
    }

    private bool CheckGrounded()
    {
        Vector3 origin;

        if (capsuleCollider != null)
        {
            Bounds bounds = capsuleCollider.bounds;
            origin = new Vector3(
                bounds.center.x,
                bounds.min.y + groundCheckRadius + 0.05f,
                bounds.center.z
            );
        }
        else
        {
            origin = transform.position + Vector3.up * 0.2f;
        }

        int hitCount = Physics.SphereCastNonAlloc(
            origin,
            groundCheckRadius,
            Vector3.down,
            groundHits,
            groundCheckDistance,
            groundLayer,
            QueryTriggerInteraction.Ignore
        );

        bool grounded = false;

        for (int i = 0; i < hitCount; i++)
        {
            Collider hitCollider = groundHits[i].collider;

            if (hitCollider == null)
            {
                continue;
            }

            if (hitCollider.transform == transform || hitCollider.transform.IsChildOf(transform))
            {
                continue;
            }

            grounded = true;
            break;
        }

        if (drawDebug)
        {
            Debug.DrawRay(
                origin,
                Vector3.down * groundCheckDistance,
                grounded ? Color.green : Color.red
            );
        }

        return grounded;
    }
}