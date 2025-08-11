<?php
/**
 * BitfinexBot Web API - 收益数据接口
 */

// 数据库连接配置
$db_host = 'mysql';
$db_name = 'bitfinex_bot_db';
$db_user = 'bitfinex_bot';
$db_pass = 'BitfinexBot@2025';

try {
    $pdo = new PDO("mysql:host=$db_host;dbname=$db_name;charset=utf8mb4", $db_user, $db_pass);
    $pdo->setAttribute(PDO::ATTR_ERRMODE, PDO::ERRMODE_EXCEPTION);
} catch (PDOException $e) {
    sendResponse(false, null, 'Database connection failed: ' . $e->getMessage());
    exit();
}

// 获取操作类型
$action = $_GET['action'] ?? '';

switch ($action) {
    case 'chart':
        handleChartData();
        break;
    
    case 'summary':
        handleSummaryData();
        break;
        
    default:
        sendResponse(false, null, 'Invalid action');
        break;
}

/**
 * 处理图表数据请求
 */
function handleChartData() {
    global $pdo;
    
    $days = intval($_GET['days'] ?? 30);
    $currency = $_GET['currency'] ?? 'USD';
    
    // 限制天数范围
    if ($days < 1) $days = 30;
    if ($days > 365) $days = 365;
    
    try {
        // 获取最近N天的收益数据
        $sql = "SELECT 
                    summary_date as date,
                    total_earnings,
                    total_trades,
                    total_amount
                FROM daily_earnings_summary 
                WHERE currency = :currency 
                AND summary_date >= DATE_SUB(CURDATE(), INTERVAL :days DAY)
                ORDER BY summary_date ASC";
        
        $stmt = $pdo->prepare($sql);
        $stmt->bindParam(':currency', $currency);
        $stmt->bindParam(':days', $days, PDO::PARAM_INT);
        $stmt->execute();
        
        $earnings = $stmt->fetchAll(PDO::FETCH_ASSOC);
        
        // 计算统计信息
        $statistics = calculateStatistics($earnings);
        
        // 构造响应数据
        $data = [
            'earnings' => $earnings,
            'statistics' => $statistics,
            'period' => [
                'days' => $days,
                'currency' => $currency,
                'from' => date('Y-m-d', strtotime("-$days days")),
                'to' => date('Y-m-d')
            ]
        ];
        
        sendResponse(true, $data);
        
    } catch (PDOException $e) {
        sendResponse(false, null, 'Failed to fetch chart data: ' . $e->getMessage());
    }
}

/**
 * 处理汇总数据请求
 */
function handleSummaryData() {
    global $pdo;
    
    $currency = $_GET['currency'] ?? 'USD';
    
    try {
        // 获取各种时间段的汇总数据
        $summaryData = [];
        
        // 今日收益
        $sql = "SELECT COALESCE(SUM(total_earnings), 0) as earnings, COUNT(*) as days 
                FROM daily_earnings_summary 
                WHERE currency = :currency AND summary_date = CURDATE()";
        $stmt = $pdo->prepare($sql);
        $stmt->bindParam(':currency', $currency);
        $stmt->execute();
        $summaryData['today'] = $stmt->fetch(PDO::FETCH_ASSOC);
        
        // 本周收益
        $sql = "SELECT COALESCE(SUM(total_earnings), 0) as earnings, COUNT(*) as days 
                FROM daily_earnings_summary 
                WHERE currency = :currency 
                AND summary_date >= DATE_SUB(CURDATE(), INTERVAL 7 DAY)";
        $stmt = $pdo->prepare($sql);
        $stmt->bindParam(':currency', $currency);
        $stmt->execute();
        $summaryData['week'] = $stmt->fetch(PDO::FETCH_ASSOC);
        
        // 本月收益
        $sql = "SELECT COALESCE(SUM(total_earnings), 0) as earnings, COUNT(*) as days 
                FROM daily_earnings_summary 
                WHERE currency = :currency 
                AND summary_date >= DATE_SUB(CURDATE(), INTERVAL 30 DAY)";
        $stmt = $pdo->prepare($sql);
        $stmt->bindParam(':currency', $currency);
        $stmt->execute();
        $summaryData['month'] = $stmt->fetch(PDO::FETCH_ASSOC);
        
        sendResponse(true, $summaryData);
        
    } catch (PDOException $e) {
        sendResponse(false, null, 'Failed to fetch summary data: ' . $e->getMessage());
    }
}

/**
 * 计算统计信息
 */
function calculateStatistics($earnings) {
    if (empty($earnings)) {
        return [
            'total_earnings' => 0,
            'avg_daily_earnings' => 0,
            'max_daily_earnings' => 0,
            'min_daily_earnings' => 0,
            'trading_days' => 0,
            'total_trades' => 0
        ];
    }
    
    $totalEarnings = 0;
    $maxEarnings = 0;
    $minEarnings = PHP_FLOAT_MAX;
    $tradingDays = 0;
    $totalTrades = 0;
    
    foreach ($earnings as $earning) {
        $dailyEarnings = floatval($earning['total_earnings']);
        $totalEarnings += $dailyEarnings;
        $totalTrades += intval($earning['total_trades'] ?? 0);
        
        if ($dailyEarnings > 0) {
            $tradingDays++;
            $maxEarnings = max($maxEarnings, $dailyEarnings);
            $minEarnings = min($minEarnings, $dailyEarnings);
        }
    }
    
    // 如果没有交易天数，重置最小值
    if ($tradingDays === 0) {
        $minEarnings = 0;
    }
    
    $avgDailyEarnings = $tradingDays > 0 ? $totalEarnings / $tradingDays : 0;
    
    return [
        'total_earnings' => round($totalEarnings, 6),
        'avg_daily_earnings' => round($avgDailyEarnings, 6),
        'max_daily_earnings' => round($maxEarnings, 6),
        'min_daily_earnings' => round($minEarnings, 6),
        'trading_days' => $tradingDays,
        'total_trades' => $totalTrades,
        'data_points' => count($earnings)
    ];
}

/**
 * 发送JSON响应
 */
function sendResponse($success, $data = null, $error = null) {
    $response = [
        'success' => $success,
        'timestamp' => time(),
        'datetime' => date('Y-m-d H:i:s')
    ];
    
    if ($success && $data !== null) {
        $response['data'] = $data;
    }
    
    if (!$success && $error !== null) {
        $response['error'] = $error;
    }
    
    echo json_encode($response, JSON_UNESCAPED_UNICODE | JSON_PRETTY_PRINT);
}
?>