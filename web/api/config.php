<?php
/**
 * BitfinexBot Web API - 配置管理接口
 */

header('Content-Type: application/json; charset=utf-8');
header('Access-Control-Allow-Origin: *');
header('Access-Control-Allow-Methods: GET, POST, PUT');
header('Access-Control-Allow-Headers: Content-Type, Authorization');

require_once '../config/database.php';

class ConfigAPI {
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
                case 'PUT':
                    $this->handleUpdate();
                    break;
                default:
                    $this->sendError('Method not allowed', 405);
            }
        } catch (Exception $e) {
            $this->sendError($e->getMessage(), 500);
        }
    }
    
    private function handleGet() {
        $action = $_GET['action'] ?? 'list';
        
        switch ($action) {
            case 'list':
                $this->getConfigs();
                break;
            case 'get':
                $this->getConfig();
                break;
            default:
                $this->sendError('Invalid action', 400);
        }
    }
    
    private function handleUpdate() {
        $input = json_decode(file_get_contents('php://input'), true);
        
        if (!$input) {
            $this->sendError('Invalid JSON input', 400);
            return;
        }
        
        $action = $input['action'] ?? 'update';
        
        switch ($action) {
            case 'update':
                $this->updateConfig($input);
                break;
            case 'batch_update':
                $this->batchUpdateConfigs($input);
                break;
            default:
                $this->sendError('Invalid action', 400);
        }
    }
    
    /**
     * 获取所有配置
     */
    private function getConfigs() {
        $category = $_GET['category'] ?? '';
        $showSensitive = isset($_GET['show_sensitive']) && $_GET['show_sensitive'] === '1';
        
        $whereClause = '';
        $params = [];
        
        if ($category) {
            $whereClause = "WHERE config_key LIKE ?";
            $params[] = $category . '%';
        }
        
        if (!$showSensitive) {
            $whereClause .= $whereClause ? ' AND ' : 'WHERE ';
            $whereClause .= 'is_sensitive = 0';
        }
        
        $sql = "SELECT * FROM system_config $whereClause ORDER BY config_key";
        $configs = $this->db->fetchAll($sql, $params);
        
        // 将配置按类别分组
        $grouped = $this->groupConfigsByCategory($configs);
        
        $this->sendSuccess([
            'configs' => $configs,
            'grouped' => $grouped,
            'total' => count($configs)
        ]);
    }
    
    /**
     * 获取单个配置
     */
    private function getConfig() {
        $key = $_GET['key'] ?? '';
        
        if (!$key) {
            $this->sendError('Config key is required', 400);
            return;
        }
        
        $sql = "SELECT * FROM system_config WHERE config_key = ?";
        $config = $this->db->fetchOne($sql, [$key]);
        
        if (!$config) {
            $this->sendError('Config not found', 404);
            return;
        }
        
        // 敏感信息不返回值
        if ($config['is_sensitive']) {
            $config['config_value'] = '***';
        }
        
        $this->sendSuccess($config);
    }
    
    /**
     * 更新配置
     */
    private function updateConfig($input) {
        $required = ['config_key', 'config_value'];
        foreach ($required as $field) {
            if (!isset($input[$field])) {
                $this->sendError("Missing required field: $field", 400);
                return;
            }
        }
        
        $key = $input['config_key'];
        $value = $input['config_value'];
        $type = $input['config_type'] ?? 'string';
        $description = $input['description'] ?? '';
        $userId = $input['user_id'] ?? 1; // 默认用户ID
        
        // 验证配置类型
        $validTypes = ['string', 'number', 'boolean', 'json'];
        if (!in_array($type, $validTypes)) {
            $this->sendError('Invalid config type', 400);
            return;
        }
        
        // 验证值格式
        if (!$this->validateConfigValue($value, $type)) {
            $this->sendError("Invalid value for type: $type", 400);
            return;
        }
        
        // 检查配置是否存在
        $existingSql = "SELECT id FROM system_config WHERE config_key = ?";
        $existing = $this->db->fetchOne($existingSql, [$key]);
        
        if ($existing) {
            // 更新现有配置
            $sql = "UPDATE system_config SET 
                        config_value = ?, 
                        config_type = ?, 
                        description = ?, 
                        updated_by = ?
                    WHERE config_key = ?";
            $params = [$value, $type, $description, $userId, $key];
        } else {
            // 创建新配置
            $sql = "INSERT INTO system_config 
                        (config_key, config_value, config_type, description, updated_by) 
                    VALUES (?, ?, ?, ?, ?)";
            $params = [$key, $value, $type, $description, $userId];
        }
        
        $this->db->execute($sql, $params);
        
        // 记录操作日志
        $this->logConfigChange($key, $value, $userId, $existing ? 'update' : 'create');
        
        $this->sendSuccess([
            'message' => 'Configuration updated successfully',
            'config_key' => $key,
            'action' => $existing ? 'updated' : 'created'
        ]);
    }
    
    /**
     * 批量更新配置
     */
    private function batchUpdateConfigs($input) {
        if (!isset($input['configs']) || !is_array($input['configs'])) {
            $this->sendError('Configs array is required', 400);
            return;
        }
        
        $userId = $input['user_id'] ?? 1;
        $updated = 0;
        $errors = [];
        
        $this->db->beginTransaction();
        
        try {
            foreach ($input['configs'] as $config) {
                if (!isset($config['config_key']) || !isset($config['config_value'])) {
                    $errors[] = "Missing key or value for config";
                    continue;
                }
                
                $key = $config['config_key'];
                $value = $config['config_value'];
                $type = $config['config_type'] ?? 'string';
                
                if (!$this->validateConfigValue($value, $type)) {
                    $errors[] = "Invalid value for config: $key";
                    continue;
                }
                
                $sql = "UPDATE system_config SET 
                            config_value = ?, 
                            updated_by = ?
                        WHERE config_key = ?";
                
                $affected = $this->db->execute($sql, [$value, $userId, $key]);
                
                if ($affected > 0) {
                    $updated++;
                    $this->logConfigChange($key, $value, $userId, 'batch_update');
                }
            }
            
            $this->db->commit();
            
            $this->sendSuccess([
                'message' => 'Batch update completed',
                'updated_count' => $updated,
                'errors' => $errors
            ]);
            
        } catch (Exception $e) {
            $this->db->rollBack();
            throw $e;
        }
    }
    
    /**
     * 验证配置值
     */
    private function validateConfigValue($value, $type) {
        switch ($type) {
            case 'number':
                return is_numeric($value);
            case 'boolean':
                return in_array(strtolower($value), ['true', 'false', '1', '0']);
            case 'json':
                json_decode($value);
                return json_last_error() === JSON_ERROR_NONE;
            case 'string':
            default:
                return is_string($value);
        }
    }
    
    /**
     * 按类别分组配置
     */
    private function groupConfigsByCategory($configs) {
        $grouped = [];
        
        foreach ($configs as $config) {
            $key = $config['config_key'];
            $category = 'general';
            
            // 根据键名确定类别
            if (strpos($key, 'HIGH_HOLD_') === 0) {
                $category = 'high_hold';
            } elseif (strpos($key, 'KLINE_') === 0) {
                $category = 'kline_strategy';
            } elseif (in_array($key, ['MIN_DAILY_LEND_RATE', 'SPREAD_LEND', 'GAP_BOTTOM', 'GAP_TOP'])) {
                $category = 'lending_strategy';
            } elseif (in_array($key, ['ENABLE_SMART_STRATEGY', 'VOLATILITY_THRESHOLD'])) {
                $category = 'smart_strategy';
            } elseif (in_array($key, ['MINUTES_RUN', 'TEST_MODE'])) {
                $category = 'system';
            }
            
            if (!isset($grouped[$category])) {
                $grouped[$category] = [];
            }
            
            $grouped[$category][] = $config;
        }
        
        return $grouped;
    }
    
    /**
     * 记录配置更改日志
     */
    private function logConfigChange($key, $value, $userId, $action) {
        $sql = "INSERT INTO operation_logs 
                    (user_id, operation_type, operation_desc, result) 
                VALUES (?, 'config_change', ?, 'success')";
        
        $desc = "$action config: $key";
        $this->db->execute($sql, [$userId, $desc]);
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
$api = new ConfigAPI();
$api->handleRequest();
?>