<?php
$input = file_get_contents("php://input");
file_put_contents("clip_history.json", $input);
echo "OK";
