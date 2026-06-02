using UnityEngine;

public class WallRunController : MonoBehaviour
{
    [Header("References")]
    [SerializeField] private Transform cameraTransform;

    [Header("Wall Check")]
    [SerializeField] private float wallCheckDistance = 0.9f;
    [SerializeField] private LayerMask wallMask;

    [Header("Wall Run")]
    [SerializeField] private float minStartSpeed = 5f;
    [SerializeField] private float wallRunSpeed = 14f;
    [SerializeField] private float wallRunGravity = 2f;
    [SerializeField] private float wallStickForce = 6f;
    [SerializeField] private float maxWallRunTime = 1.8f;

    [Header("Wall Jump")]
    [SerializeField] private float wallJumpSideForce = 9f;
    [SerializeField] private float wallJumpUpForce = 7f;
    [SerializeField] private float wallJumpForwardForce = 6f;
    [SerializeField] private float wallJumpCooldown = 0.25f;

    private Rigidbody rb;

    private bool isWallRunning;
    private bool wallRight;
    private bool wallLeft;

    private Vector3 wallNormal;
    private Vector3 wallForward;

    private float wallRunTimer;
    private float lastWallJumpTime;

    public bool IsWallRunning => isWallRunning;

    private void Awake()
    {
        rb = GetComponent<Rigidbody>();
    }

    private void Update()
    {
        if (isWallRunning && Input.GetKeyDown(KeyCode.Space))
        {
            WallJump();
        }
    }

    private void FixedUpdate()
    {
        CheckWall();

        if (CanStartOrContinueWallRun())
        {
            if (!isWallRunning)
            {
                StartWallRun();
            }

            WallRunMovement();
        }
        else
        {
            StopWallRun();
        }
    }

    private void CheckWall()
    {
        wallRight = Physics.Raycast(transform.position, transform.right, out RaycastHit rightHit, wallCheckDistance, wallMask);
        wallLeft = Physics.Raycast(transform.position, -transform.right, out RaycastHit leftHit, wallCheckDistance, wallMask);

        if (wallRight)
        {
            wallNormal = rightHit.normal;
        }
        else if (wallLeft)
        {
            wallNormal = leftHit.normal;
        }
        else
        {
            wallNormal = Vector3.zero;
        }
    }

    private bool CanStartOrContinueWallRun()
    {
        bool hasWall = wallRight || wallLeft;
        bool isPressingForward = Input.GetKey(KeyCode.W);
        bool isInAir = !Physics.Raycast(transform.position, Vector3.down, 1.25f);
        bool hasEnoughSpeed = rb.velocity.magnitude >= minStartSpeed;
        bool cooldownFinished = Time.time >= lastWallJumpTime + wallJumpCooldown;

        return hasWall && isPressingForward && isInAir && hasEnoughSpeed && cooldownFinished;
    }

    private void StartWallRun()
    {
        isWallRunning = true;
        wallRunTimer = 0f;
        rb.useGravity = false;
    }

    private void WallRunMovement()
    {
        wallRunTimer += Time.fixedDeltaTime;

        if (wallRunTimer > maxWallRunTime)
        {
            StopWallRun();
            return;
        }

        wallForward = Vector3.Cross(wallNormal, Vector3.up);

        if (Vector3.Dot(wallForward, cameraTransform.forward) < 0f)
        {
            wallForward = -wallForward;
        }

        Vector3 currentVelocity = rb.velocity;

        float currentSpeed = new Vector3(currentVelocity.x, 0f, currentVelocity.z).magnitude;
        float targetSpeed = Mathf.Max(currentSpeed, wallRunSpeed);

        Vector3 targetVelocity = wallForward.normalized * targetSpeed;
        targetVelocity.y = -wallRunGravity;

        rb.velocity = Vector3.Lerp(rb.velocity, targetVelocity, 0.25f);

        rb.AddForce(-wallNormal * wallStickForce, ForceMode.Acceleration);
    }

    private void StopWallRun()
    {
        if (!isWallRunning)
        {
            return;
        }

        isWallRunning = false;
        rb.useGravity = true;
    }

    private void WallJump()
    {
        Vector3 jumpDirection =
            wallNormal * wallJumpSideForce +
            Vector3.up * wallJumpUpForce +
            cameraTransform.forward * wallJumpForwardForce;

        isWallRunning = false;
        rb.useGravity = true;
        lastWallJumpTime = Time.time;

        rb.velocity = new Vector3(rb.velocity.x, 0f, rb.velocity.z);
        rb.AddForce(jumpDirection, ForceMode.Impulse);
    }
}