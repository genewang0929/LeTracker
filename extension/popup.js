document.getElementById("syncBtn").addEventListener("click", async () => {
  const statusDiv = document.getElementById("status");
  statusDiv.textContent = "Fetching from LeetCode...";

  try {
    // 1. 呼叫 LeetCode API (它會自動帶上你瀏覽器的 Cookie，所以不用擔心登入問題)
    // limit=1000 應該夠抓很久的紀錄，不夠的話要寫迴圈 fetch offset
    const response = await fetch(
      "https://leetcode.com/api/submissions/?offset=0&limit=1000",
    );

    if (!response.ok) {
      throw new Error("Failed to fetch from LeetCode. Are you logged in?");
    }

    const data = await response.json();
    const submissions = data.submissions_dump; // LeetCode 回傳的陣列 key 叫這個

    statusDiv.textContent = `Fetched ${submissions.length} records. Sending to Backend...`;

    // 2. 轉換格式以符合我們 Go Backend 的需求
    // LeetCode 格式: { title_slug: "two-sum", status_display: "Accepted", timestamp: 167... }
    // Go Backend 格式: { slug, status, timestamp, title }
    console.log(submissions);
    const formattedHistory = submissions.map((sub) => ({
      title: sub.title,
      slug: sub.title_slug,
      status: sub.status_display, // "Accepted", "Wrong Answer", "Runtime Error"
      timestamp: sub.timestamp,
    }));

    // 3. 傳送給你的 Go Backend
    const backendResp = await fetch("http://localhost:8080/api/v1/history", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({ history: formattedHistory }),
    });

    if (!backendResp.ok) {
      const errText = await backendResp.text();
      throw new Error("Backend Error: " + errText);
    }

    const result = await backendResp.json();
    statusDiv.textContent = `Success! Imported ${result.count} records.`;
    statusDiv.style.color = "green";
  } catch (err) {
    console.error(err);
    statusDiv.textContent = "Error: " + err.message;
    statusDiv.style.color = "red";
  }
});
