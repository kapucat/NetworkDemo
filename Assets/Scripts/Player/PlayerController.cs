using UnityEngine;

public class PlayerController : MonoBehaviour
{
    [SerializeField] private float moveSpeed = 8f;
    [SerializeField] private float jumpPower = 7f;
    [SerializeField] private Transform cameraTransform;

    private Rigidbody rb;
    private bool isGrounded;

    private GrappleController grappleController;
    private WallRunController wallRunController;

    private float xInput;
    private float zInput;
    private bool jumpRequested;

    private void Awake()
    {
        rb = GetComponent<Rigidbody>();
        rb.freezeRotation = true;

        grappleController = GetComponent<GrappleController>();
        wallRunController = GetComponent<WallRunController>();
    }

    private void Update()
    {
        xInput = Input.GetAxisRaw("Horizontal");
        zInput = Input.GetAxisRaw("Vertical");

        if (Input.GetKeyDown(KeyCode.Space) && isGrounded)
        {
            jumpRequested = true;
        }
    }

    private void FixedUpdate()
    {
        bool isGrappling = grappleController != null && grappleController.IsGrappling;
        bool isWallRunning = wallRunController != null && wallRunController.IsWallRunning;

        if (!isGrappling && !isWallRunning)
        {
            Move();
        }

        if (jumpRequested)
        {
            Jump();
            jumpRequested = false;
        }
    }

    private void Move()
    {
        Vector3 forward = cameraTransform.forward;
        Vector3 right = cameraTransform.right;

        forward.y = 0f;
        right.y = 0f;

        forward.Normalize();
        right.Normalize();

        Vector3 moveDirection = forward * zInput + right * xInput;
        Vector3 velocity = moveDirection.normalized * moveSpeed;

        rb.velocity = new Vector3(velocity.x, rb.velocity.y, velocity.z);
    }

    private void Jump()
    {
        rb.velocity = new Vector3(rb.velocity.x, 0f, rb.velocity.z);
        rb.AddForce(Vector3.up * jumpPower, ForceMode.Impulse);
        isGrounded = false;
    }

    private void OnCollisionStay(Collision collision)
    {
        isGrounded = true;
    }

    private void OnCollisionExit(Collision collision)
    {
        isGrounded = false;
    }
}