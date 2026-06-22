using UnityEngine;

public class PlayerController : MonoBehaviour
{
    [Header("Move")]
    [SerializeField] private float groundAcceleration = 35f;
    [SerializeField] private float airAcceleration = 15f;
    [SerializeField] private float maxGroundControlSpeed = 18f;
    [SerializeField] private float maxAirControlSpeed = 35f;
    [SerializeField] private float groundFriction = 8f;
    [SerializeField] private float absoluteMaxSpeed = 50f;

    [Header("Jump")]
    [SerializeField] private float jumpPower = 7f;

    [Header("References")]
    [SerializeField] private Transform cameraTransform;



    [Header("Gravity")]
    [SerializeField] private float fallMultiplier = 3.0f;
    [SerializeField] private float lowJumpMultiplier = 2.0f;

    [Header("Air Control")]
    [SerializeField] private float airFriction = 0.8f;


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

        ApplyExtraGravity(isWallRunning);
        ApplyAirFriction(isGrappling, isWallRunning);
        LimitAbsoluteSpeed();
    }

    private void Move()
    {
        Vector3 forward = cameraTransform.forward;
        Vector3 right = cameraTransform.right;

        forward.y = 0f;
        right.y = 0f;

        forward.Normalize();
        right.Normalize();

        Vector3 inputDirection = forward * zInput + right * xInput;

        Vector3 horizontalVelocity = new Vector3(rb.velocity.x, 0f, rb.velocity.z);

        if (inputDirection.sqrMagnitude > 0.01f)
        {
            inputDirection.Normalize();

            float acceleration = isGrounded ? groundAcceleration : airAcceleration;
            float maxControlSpeed = isGrounded ? maxGroundControlSpeed : maxAirControlSpeed;

            float speedInInputDirection = Vector3.Dot(horizontalVelocity, inputDirection);

            // ÇªÇÃï˚å¸Ç…Ç‹Çæâ¡ë¨Ç≈Ç´ÇÈéûÇæÇØóÕÇë´Ç∑
            // Ç∑Ç≈Ç…ë¨Ç¢èÍçáÇÕë¨ìxÇè¡Ç≥Ç∏ÅAè„èëÇ´Ç‡ÇµÇ»Ç¢
            if (speedInInputDirection < maxControlSpeed)
            {
                rb.AddForce(inputDirection * acceleration, ForceMode.Acceleration);
            }
        }
        else if (isGrounded)
        {
            // ì¸óÕÇµÇƒÇ¢Ç»Ç¢éûÇæÇØínè„Ç≈è≠Çµå∏ë¨
            rb.AddForce(-horizontalVelocity * groundFriction, ForceMode.Acceleration);
        }
    }

    private void Jump()
    {
        rb.velocity = new Vector3(rb.velocity.x, 0f, rb.velocity.z);
        rb.AddForce(Vector3.up * jumpPower, ForceMode.Impulse);
        isGrounded = false;
    }

    private void LimitAbsoluteSpeed()
    {
        Vector3 horizontalVelocity = new Vector3(rb.velocity.x, 0f, rb.velocity.z);

        if (horizontalVelocity.magnitude > absoluteMaxSpeed)
        {
            Vector3 limitedVelocity = horizontalVelocity.normalized * absoluteMaxSpeed;
            rb.velocity = new Vector3(limitedVelocity.x, rb.velocity.y, limitedVelocity.z);
        }
    }

    private void OnCollisionStay(Collision collision)
    {
        isGrounded = true;
    }

    private void OnCollisionExit(Collision collision)
    {
        isGrounded = false;
    }



    private void ApplyExtraGravity(bool isWallRunning)
    {
        if (isWallRunning)
        {
            return;
        }

        if (rb.velocity.y < 0f)
        {
            rb.AddForce(
                Vector3.up * Physics.gravity.y * (fallMultiplier - 1f),
                ForceMode.Acceleration
            );
        }
        else if (rb.velocity.y > 0f && !Input.GetKey(KeyCode.Space))
        {
            rb.AddForce(
                Vector3.up * Physics.gravity.y * (lowJumpMultiplier - 1f),
                ForceMode.Acceleration
            );
        }
    }

    private void ApplyAirFriction(bool isGrappling, bool isWallRunning)
    {
        if (isGrounded || isGrappling || isWallRunning)
        {
            return;
        }

        Vector3 horizontalVelocity = new Vector3(rb.velocity.x, 0f, rb.velocity.z);

        rb.AddForce(
            -horizontalVelocity * airFriction,
            ForceMode.Acceleration
        );
    }


    public void SetCameraTransform(Transform newCameraTransform)
    {
        cameraTransform = newCameraTransform;
    }

}