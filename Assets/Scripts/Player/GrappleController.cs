using UnityEngine;

public class GrappleController : MonoBehaviour
{
    [Header("References")]
    [SerializeField] private Transform cameraTransform;
    [SerializeField] private LineRenderer lineRenderer;

    [Header("Grapple Settings")]
    [SerializeField] private bool canUseGrapple = true; //グラップルのオンオフ
    [SerializeField] private float maxDistance = 60f;
    [SerializeField] private float pullForce = 38f;
    [SerializeField] private float swingForce = 28f;
    [SerializeField] private float ropeTightness = 10f;
    [SerializeField] private float startBoost = 8f;
    [SerializeField] private float stopDistance = 3f;
    [SerializeField] private float maxSpeed = 35f;
    [SerializeField] private LayerMask grappleMask = ~0;

    private Rigidbody rb;
    private bool isGrappling;
    private Vector3 grapplePoint;
    private float ropeLength;

    public bool IsGrappling => isGrappling;

    private void Awake()
    {
        rb = GetComponent<Rigidbody>();
        rb.freezeRotation = true;

        if (lineRenderer != null)
        {
            lineRenderer.enabled = false;
            lineRenderer.positionCount = 0;
        }
    }

    private void Update()
    {
        if (Input.GetMouseButtonDown(0))
        {
            StartGrapple();
        }

        if (Input.GetMouseButtonUp(0))
        {
            StopGrapple();
        }

        if (isGrappling)
        {
            Debug.DrawLine(cameraTransform.position, grapplePoint, Color.green);
            UpdateRopeLine();
        }
    }

    private void FixedUpdate()
    {
        if (!isGrappling)
        {
            return;
        }

        ApplyGrappleMovement();
        LimitSpeed();
    }

    private void StartGrapple() // グラップル開始
    {
        if (!canUseGrapple)
        {
            return;
        }

        Ray ray = new Ray(cameraTransform.position, cameraTransform.forward);

        if (!Physics.Raycast(ray, out RaycastHit hit, maxDistance, grappleMask))
        {
            return;
        }

        grapplePoint = hit.point;

        float distance = Vector3.Distance(transform.position, grapplePoint);

        // 少し短めにすると、支点に引っ張られてスイングしやすい
        ropeLength = distance * 0.8f;

        isGrappling = true;

        // 掴んだ瞬間に少し勢いを足す
        Vector3 boostDirection = (cameraTransform.forward + Vector3.up * 0.25f).normalized;
        rb.AddForce(boostDirection * startBoost, ForceMode.Impulse);

        if (lineRenderer != null)
        {
            lineRenderer.enabled = true;
            lineRenderer.positionCount = 2;
            // ロープの始点をカメラの前に少しオフセットして設定
            Vector3 ropeStartPosition =
                cameraTransform.position
                + cameraTransform.forward * 0.4f
                + cameraTransform.right * 0.25f
                - cameraTransform.up * 0.2f;

            lineRenderer.SetPosition(0, ropeStartPosition);
            lineRenderer.SetPosition(1, grapplePoint);
        }
    }

    private void ApplyGrappleMovement()
    {
        Vector3 toPoint = grapplePoint - transform.position;
        float distance = toPoint.magnitude;

        if (distance <= stopDistance)
        {
            StopGrapple();
            return;
        }

        Vector3 directionToPoint = toPoint.normalized;

        // 基本の引っ張り
        rb.AddForce(directionToPoint * pullForce, ForceMode.Acceleration);

        // ロープ長より離れたら、ロープが張っている感じを出す
        if (distance > ropeLength)
        {
            float stretch = distance - ropeLength;
            rb.AddForce(directionToPoint * stretch * ropeTightness, ForceMode.Acceleration);

            // 支点から離れる方向の速度を少し削る
            Vector3 velocity = rb.velocity;
            float awaySpeed = Vector3.Dot(velocity, -directionToPoint);

            if (awaySpeed > 0f)
            {
                velocity += directionToPoint * awaySpeed * 0.7f;
                rb.velocity = velocity;
            }
        }

        // 入力方向にスイング加速
        float horizontal = Input.GetAxisRaw("Horizontal");
        float vertical = Input.GetAxisRaw("Vertical");

        Vector3 forwardOnSwingPlane = Vector3.ProjectOnPlane(cameraTransform.forward, directionToPoint).normalized;
        Vector3 rightOnSwingPlane = Vector3.ProjectOnPlane(cameraTransform.right, directionToPoint).normalized;

        Vector3 swingDirection = forwardOnSwingPlane * vertical + rightOnSwingPlane * horizontal;

        if (swingDirection.sqrMagnitude > 0.01f)
        {
            rb.AddForce(swingDirection.normalized * swingForce, ForceMode.Acceleration);
        }
    }

    private void LimitSpeed()
    {
        if (rb.velocity.magnitude > maxSpeed)
        {
            rb.velocity = rb.velocity.normalized * maxSpeed;
        }
    }

    private void StopGrapple()
    {
        isGrappling = false;

        if (lineRenderer != null)
        {
            lineRenderer.enabled = false;
            lineRenderer.positionCount = 0;
        }
    }

    private void UpdateRopeLine()
    {
        if (lineRenderer == null)
        {
            return;
        }

        Vector3 ropeStartPosition =
            cameraTransform.position
            + cameraTransform.forward * 0.4f
            + cameraTransform.right * 0.25f
            - cameraTransform.up * 0.2f;

        lineRenderer.SetPosition(0, ropeStartPosition);
        lineRenderer.SetPosition(1, grapplePoint);
    }

    public void SetCameraTransform(Transform newCameraTransform)
    {
        cameraTransform = newCameraTransform;
    }

}