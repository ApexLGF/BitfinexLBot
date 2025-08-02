<?php
/**
 * BitfinexBot Web API - 状态查询接口
 */

header('Content-Type: application/json; charset=utf-8');
header('Access-Control-Allow-Origin: *');
header('Access-Control-Allow-Methods: GET');
header('Access-Control-Allow-Headers: Content-Type, Authorization');

require_once '../config/database.php';

class StatusAPI {
    private $db;
    
    public function __construct() {
        $this->db = new Database();
    }
    
    public function handleRequest() {
        if ($_SERVER['REQUEST_METHOD'] !== 'GET') {
            $this->sendError('Method not allowed', 405);
            return;
        }
        
        $action = $_GET['action'] ?? '';
        
        try {
            switch ($action) {
                case 'bot':
                    $this->getBotStatus();
                    break;
                case 'system':
                    $this->getSystemStatus();
                    break;
                case 'notifications':
                    $this->getNotifications();
                    break;
                case 'stats':
                    $this->getStatistics();
                    break;
                default:
                    $this->sendError('Invalid action', 400);
            }
        } catch (Exception $e) {
            $this->sendError($e->getMessage(), 500);
        }
    }
    
    /**
     * 获取机器人状态
     */
    private function getBotStatus() {
        $sql = "SELECT * FROM bot_realtime_status ORDER BY id DESC LIMIT 1";
        $status = $this->db->fetchOne($sql);
        
        if (!$status) {
            // 如果没有状态记录，创建默认状态
            $this->createDefaultStatus();
            $status = $this->db->fetchOne($sql);
        }
        
        // 计算运行时间
        if ($status['last_execution']) {
            $lastExecution = new DateTime($status['last_execution']);
            $now = new DateTime();
            $status['uptime'] = $now->getTimestamp() - $lastExecution->getTimestamp();
        } else {
            $status['uptime'] = 0;
        }
        
        // 添加状态描述
        $statusMessages = [
            'running' => '运行中',
            'stopped' => '已停止',
            'error' => '错误状态',
            'maintenance' => '维护中'
        ];
        
        $status['status_text'] = $statusMessages[$status['status']] ?? '未知状态';
        
        $this->sendSuccess($status);
    }
    
    /**
     * 获取系统状态
     */
    private function getSystemStatus() {
        // 获取最近的命令统计
        $commandStats = $this->getCommandStatistics();
        
        // 获取数据库连接状态
        $dbStatus = $this->checkDatabaseStatus();
        
        // 获取最近的错误
        $recentErrors = $this->getRecentErrors();
        
        $systemStatus = [
            'database' => $dbStatus,
            'commands' => $commandStats,
            'errors' => $recentErrors,
            'server_time' => date('Y-m-d H:i:s'),
            'timezone' => date_default_timezone_get()
        ];
        
        $this->sendSuccess($systemStatus);
    }
    
    /**
     * 获取通知列表
     */
    private function getNotifications() {
        $limit = (int)($_GET['limit'] ?? 10);
        $unreadOnly = isset($_GET['unread_only']) && $_GET['unread_only'] === '1';
        
        $whereClause = $unreadOnly ? 'WHERE is_read = 0' : '';
        
        $sql = "SELECT * FROM notifications $whereClause ORDER BY created_at DESC LIMIT ?";
        $notifications = $this->db->fetchAll($sql, [$limit]);
        
        // 获取未读通知数量
        $unreadCount = $this->db->fetchOne("SELECT COUNT(*) as count FROM notifications WHERE is_read = 0")['count'];
        
        $this->sendSuccess([
            'notifications' => $notifications,
            'unread_count' => (int)$unreadCount,
            'total_shown' => count($notifications)
        ]);
    }
    
    /**
     * 获取统计信息
     */
    private function getStatistics() {
        $stats = [];
        
        // 命令统计
        $stats['commands'] = [
            'total' => $this->getCommandCount(),
            'pending' => $this->getCommandCount('pending'),
            'processing' => $this->getCommandCount('processing'),
            'completed' => $this->getCommandCount('completed'),
            'failed' => $this->getCommandCount('failed')
        ];
        
        // 通知统计
        $stats['notifications'] = [
            'total' => $this->getNotificationCount(),
            'unread' => $this->getNotificationCount(false),
            'by_type' => $this->getNotificationsByType()
        ];
        
        // 最近24小时活动
        $stats['recent_activity'] = $this->getRecentActivity();
        
        $this->sendSuccess($stats);
    }
    
    /**
     * 创建默认状态
     */
    private function createDefaultStatus() {
        $sql = "INSERT INTO bot_realtime_status (status) VALUES ('stopped') 
                ON DUPLICATE KEY UPDATE status = status";
        $this->db->execute($sql);
    }
    
    /**
     * 获取命令统计
     */
    private function getCommandStatistics() {
        $sql = "SELECT status, COUNT(*) as count FROM web_commands 
                WHERE created_at >= DATE_SUB(NOW(), INTERVAL 24 HOUR) 
                GROUP BY status";
        $results = $this->db->fetchAll($sql);
        
        $stats = [];
        foreach ($results as $row) {
            $stats[$row['status']] = (int)$row['count'];
        }
        
        return $stats;
    }
    
    /**
     * 检查数据库状态
     */
    private function checkDatabaseStatus() {
        try {
            $this->db->fetchOne("SELECT 1");
            return [
                'status' => 'connected',
                'message' => '数据库连接正常'
            ];
        } catch (Exception $e) {
            return [
                'status' => 'error',
                'message' => '数据库连接异常: ' . $e->getMessage()
            ];
        }
    }
    
    /**
     * 获取最近错误
     */
    private function getRecentErrors() {
        $sql = "SELECT * FROM notifications WHERE level = 'error' 
                ORDER BY created_at DESC LIMIT 5";
        return $this->db->fetchAll($sql);
    }
    
    /**
     * 获取命令数量
     */
    private function getCommandCount($status = null) {
        if ($status) {
            $sql = "SELECT COUNT(*) as count FROM web_commands WHERE status = ?";
            $result = $this->db->fetchOne($sql, [$status]);
        } else {
            $sql = "SELECT COUNT(*) as count FROM web_commands";
            $result = $this->db->fetchOne($sql);
        }
        return (int)$result['count'];
    }
    
    /**
     * 获取通知数量
     */
    private function getNotificationCount($read = null) {
        if ($read !== null) {
            $sql = "SELECT COUNT(*) as count FROM notifications WHERE is_read = ?";
            $result = $this->db->fetchOne($sql, [$read ? 1 : 0]);
        } else {
            $sql = "SELECT COUNT(*) as count FROM notifications";
            $result = $this->db->fetchOne($sql);
        }
        return (int)$result['count'];
    }
    
    /**
     * 按类型获取通知
     */
    private function getNotificationsByType() {
        $sql = "SELECT type, COUNT(*) as count FROM notifications GROUP BY type";
        $results = $this->db->fetchAll($sql);
        
        $types = [];
        foreach ($results as $row) {
            $types[$row['type']] = (int)$row['count'];
        }
        
        return $types;
    }
    
    /**
     * 获取最近活动
     */
    private function getRecentActivity() {
        $sql = "SELECT 
                    DATE(created_at) as date,
                    COUNT(*) as commands,
                    SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END) as completed,
                    SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) as failed
                FROM web_commands 
                WHERE created_at >= DATE_SUB(NOW(), INTERVAL 7 DAY)
                GROUP BY DATE(created_at)
                ORDER BY date DESC";
        
        return $this->db->fetchAll($sql);
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
$api = new StatusAPI();
$api->handleRequest();
?>