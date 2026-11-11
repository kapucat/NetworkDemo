public static class MMOServerConfig
{
#if UNITY_EDITOR
    private const string Host = "localhost";
    private const int Port = 8081;
    private const string HttpScheme = "http";
    private const string WebSocketScheme = "ws";
#else
    private const string Host = "49.212.160.97";
    private const int Port = 443;
    private const string HttpScheme = "https";
    private const string WebSocketScheme = "wss";
#endif
    public static string LoginUrl => $"{HttpScheme}://{Host}:{Port}/api/accounts/login";
    public static string RegisterUrl => $"{HttpScheme}://{Host}:{Port}/api/accounts/register";
    public static string WebSocketUrl => $"{WebSocketScheme}://{Host}:{Port}/ws";
}
