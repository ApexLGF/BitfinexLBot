<?php
/**
 * BitfinexBot Web API - 主入口文件
 */

header('Content-Type: application/json; charset=utf-8');
header('Access-Control-Allow-Origin: *');
header('Access-Control-Allow-Methods: GET, POST, PUT, DELETE');
header('Access-Control-Allow-Headers: Content-Type, Authorization');

// 处理 OPTIONS 请求（CORS 预检）
if ($_SERVER['REQUEST_METHOD'] === 'OPTIONS') {
    http_response_code(200);
    exit();
}

// API 路由配置
$routes = [
    'commands' => 'commands.php',
    'status' => 'status.php',
    'config' => 'config.php',
];

// 获取路由参数
$endpoint = $_GET['endpoint'] ?? '';

if (!$endpoint) {
    sendError('Missing endpoint parameter', 400);
    exit();
}

if (!isset($routes[$endpoint])) {
    sendError('Invalid endpoint', 404);
    exit();
}

// 包含对应的API文件
$apiFile = __DIR__ . '/' . $routes[$endpoint];

if (!file_exists($apiFile)) {
    sendError('API file not found', 500);
    exit();
}

// 移除endpoint参数，避免传递给子API
unset($_GET['endpoint']);

try {
    require_once $apiFile;
} catch (Exception $e) {
    sendError('Internal server error: ' . $e->getMessage(), 500);
}

function sendError($message, $code = 400) {
    http_response_code($code);
    echo json_encode([
        'success' => false,
        'error' => $message,
        'timestamp' => time()
    ], JSON_UNESCAPED_UNICODE);
}
?>