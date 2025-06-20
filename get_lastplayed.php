<?php
header("Content-Type: application/json");
$filename = "clip_history.json";
if (file_exists($filename)) {
    echo file_get_contents($filename);
} else {
    echo "{}";
}
