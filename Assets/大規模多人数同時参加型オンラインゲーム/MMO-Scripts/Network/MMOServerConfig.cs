public static class MMOServerConfig
{
    private const string Host = "49.212.160.97";
    private const int Port = 443;
    private const string HttpScheme = "https";
    private const string WebSocketScheme = "wss";
    public static string LoginUrl => $"{HttpScheme}://{Host}:{Port}/api/accounts/login";
    public static string RegisterUrl => $"{HttpScheme}://{Host}:{Port}/api/accounts/register";
    public static string WebSocketUrl => $"{WebSocketScheme}://{Host}:{Port}/ws";
}
