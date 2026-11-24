using System;
using System.Collections.Generic;
using System.Text;

// Character ranges use UTF-16 indices, matching TMP_CharacterInfo.index.
public sealed class MMOChatHistory
{
    public struct NoticeRange
    {
        public int Start;
        public int End;
    }

    private struct Entry
    {
        public string Text;
        public bool IsNotice;
    }

    private readonly List<Entry> entries = new List<Entry>();
    private readonly List<NoticeRange> noticeRanges = new List<NoticeRange>();

    public string Text { get; private set; } = "";
    public IReadOnlyList<NoticeRange> NoticeRanges => noticeRanges;

    public void Add(string message, bool isNotice, int maxCount)
    {
        entries.Add(new Entry { Text = message ?? "", IsNotice = isNotice });
        maxCount = Math.Max(1, maxCount);
        if (entries.Count > maxCount)
            entries.RemoveRange(0, entries.Count - maxCount);

        var text = new StringBuilder();
        noticeRanges.Clear();
        foreach (var entry in entries)
        {
            if (text.Length > 0) text.Append('\n');
            int start = text.Length;
            text.Append(entry.Text);
            if (entry.IsNotice)
                noticeRanges.Add(new NoticeRange { Start = start, End = text.Length });
        }
        Text = text.ToString();
    }

    public void Clear()
    {
        entries.Clear();
        noticeRanges.Clear();
        Text = "";
    }
}
