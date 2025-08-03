<?php
/**
 * BitfinexBot Web API - 命令处理接口
 */

header('Content-Type: application/json; charset=utf-8');
header('Access-Control-Allow-Origin: *');
header('Access-Control-Allow-Methods: GET, POST');
header('Access-Control-Allow-Headers: Content-Type, Authorization');

require_once '../config/database.php';

class CommandAPI {
    private $db;
    
    public function __construct() {
        $this->db = new Database();
    }
    
    public function handleRequest() {
        $method = $_SERVER['REQUEST_METHOD'];
        
        try {
            switch ($method) {
                case 'GET':
                    $this->handleGet();
                    break;
                case 'POST':
                    $this->handlePost();
                    break;
                default:
                    $this->sendError('Method not allowed', 405);
            }
        } catch (Exception $e) {
            $this->sendError($e->getMessage(), 500);
        }
    }
    
    private function handleGet() {
        $action = $_GET['action'] ?? '';
        
        switch ($action) {
            case 'list':
                $this->getCommands();
                break;
            case 'status':
                $this->getCommandStatus();
                break;
            default:
                $this->sendError('Invalid action', 400);
        }
    }
    
    private function handlePost() {
        $input = json_decode(file_get_contents('php://input'), true);
        
        if (!$input) {
            $this->sendError('Invalid JSON input', 400);
            return;
        }
        
        $action = $input['action'] ?? '';
        
        switch ($action) {
            case 'submit':
                $this->submitCommand($input);
                break;
            default:
                $this->sendError('Invalid action', 400);
        }
    }
    
    /**
     * 提交新命令
     */
    private function submitCommand($input) {
        $required = ['command_type'];
        foreach ($required as $field) {
            if (!isset($input[$field])) {
                $this->sendError("Missing required field: $field", 400);
                return;
            }
        }
        
        $commandType = $input['command_type'];
        $commandData = isset($input['command_data']) ? json_encode($input['command_data']) : null;
        $userId = $input['user_id'] ?? 1; // 默认用户ID，后续实现认证后修改
        
        // 验证命令类型
        $validCommands = [
            'start', 'stop', 'restart', 'status', 'balance', 'rates', 
            'orders', 'credits', 'config', 'cancel_all', 'lending_check', 'rate_check',
            'set_config', 'get_config', 'update_all_config', 'wallets', 'daily_earnings'
        ];
        
        if (!in_array($commandType, $validCommands)) {
            $this->sendError('Invalid command type', 400);
            return;
        }
        
        $sql = "INSERT INTO web_commands (user_id, command_type, command_data) VALUES (?, ?, ?)";
        $params = [$userId, $commandType, $commandData];
        
        $this->db->execute($sql, $params);
        $commandId = $this->db->lastInsertId();
        
        $this->sendSuccess([
            'message' => 'Command submitted successfully',
            'command_id' => $commandId,
            'command_type' => $commandType,
            'timestamp' => date('Y-m-d H:i:s')
        ]);
    }
    
    /**
     * 获取命令列表
     */
    private function getCommands() {
        $limit = (int)($_GET['limit'] ?? 20);
        $offset = (int)($_GET['offset'] ?? 0);
        $status = $_GET['status'] ?? '';
        
        $whereClause = '';
        $params = [];
        
        if ($status) {
            $whereClause = 'WHERE status = ?';
            $params[] = $status;
        }
        
        $sql = "SELECT * FROM web_commands $whereClause ORDER BY created_at DESC LIMIT ? OFFSET ?";
        $params[] = $limit;
        $params[] = $offset;
        
        $commands = $this->db->fetchAll($sql, $params);
        
        // 获取总数
        $countSql = "SELECT COUNT(*) as total FROM web_commands $whereClause";
        $countParams = $status ? [$status] : [];
        $total = $this->db->fetchOne($countSql, $countParams)['total'];
        
        $this->sendSuccess([
            'commands' => $commands,
            'total' => (int)$total,
            'limit' => $limit,
            'offset' => $offset
        ]);
    }
    
    /**
     * 获取特定命令状态
     */
    private function getCommandStatus() {
        $commandId = $_GET['id'] ?? '';
        
        if (!$commandId) {
            $this->sendError('Command ID is required', 400);
            return;
        }
        
        $sql = "SELECT * FROM web_commands WHERE id = ?";
        $command = $this->db->fetchOne($sql, [$commandId]);
        
        if (!$command) {
            $this->sendError('Command not found', 404);
            return;
        }
        
        $this->sendSuccess($command);
    }
    
    private function sendSuccess($data) {
        echo json_encode([
            'success' => true,
            'data' => $data,
            'timestamp' => time()
        ], JSON_UNESCAPED_UNICODE);
    }
    
    private function sendError($message, $code = 400) {
        http_response_code($code);
        echo json_encode([
            'success' => false,
            'error' => $message,
            'timestamp' => time()
        ], JSON_UNESCAPED_UNICODE);
    }
}

// 处理请求
$api = new CommandAPI();
$api->handleRequest();
?>