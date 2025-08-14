<?php
$data = json_decode(file_get_contents("php://input"), true);
if (!isset($data["date"])) {
    http_response_code(400);
    exit("Missing date");
}

$inputDate = $data["date"]; // z. B. "2025-03-01"
$timestamp = strtotime($inputDate);
$formatted = date("d.m.Y", $timestamp); // -> "01.03.2025"

$html = <<<HTML
<!DOCTYPE html>
<html lang="de">
<head>
  <meta charset="UTF-8">
  <meta http-equiv="refresh" content="1">
  <style>
    body {
      margin: 0;
      padding: 0;
      background: transparent;
      font-family: "Segoe UI Emoji", sans-serif;
      display: flex;
      justify-content: flex-start;
      align-items: flex-end;
      height: 100vh;
    }

    .bubble {
      margin: 20px;
      background: rgba(0, 0, 0, 0.6); /* halbtransparentes Schwarz */
      color: white;
      font-size: 32px;
      padding: 10px 20px;
      border-radius: 20px;
      max-width: 90%;
      box-shadow: 0 4px 10px rgba(0, 0, 0, 0.3);
    }
  </style>
</head>
<body>
  <div class="bubble">📅 Clip vom $formatted</div>
</body>
</html>
HTML;

file_put_contents("clip_date.html", $html);
