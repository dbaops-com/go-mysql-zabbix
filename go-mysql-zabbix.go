package main

import (
    "fmt"
    //"log"
    "time"
    "flag"
    "database/sql"
    _ "github.com/go-sql-driver/mysql"
    "github.com/gomodule/redigo/redis"
    "strconv"
    "strings"
    "github.com/shopspring/decimal"
    "regexp"
    "os"
    "bufio"
)

var (
    db_host         string
    db_port         string
    db_user         string
    db_pass         string
    db_name         string
    mo_item         string
    mo_item_strings []string
    mo_value        string
    timeDura        float64
    xyz_value       string
    hisMapName      string
    curMapName      string
    sqlType         string
    calcType        string
    line_seg        string
    MySQLMonitor    map[string]string
    his_map         map[string]string
    cur_map         map[string]string
    conf_map        map[string]string

    variablesCurMap map[string]string
    statusCurMap    map[string]string
    processCurMap   map[string]string
    subordinateCurMap     map[string]string
    binlogCurMap    map[string]string
    engineCurMap    map[string]string
)

const (
    redisSTR        string = "127.0.0.1:6379"
    //statusSQL       string = "SHOW GLOBAL STATUS;"
    statusSQL       string = "SHOW GLOBAL STATUS WHERE variable_name IN ('Aborted_clients','Aborted_connects','Binlog_cache_disk_use','Binlog_cache_use','Bytes_received','Bytes_sent','Com_delete','Com_delete_multi','Com_insert','Com_insert_select','Com_load','Com_replace','Com_replace_select','Com_select','Com_update','Com_update_multi','Connections','Created_tmp_disk_tables','Created_tmp_files','Created_tmp_tables','Handler_commit','Handler_delete','Handler_read_first','Handler_read_key','Handler_read_next','Handler_read_prev','Handler_read_rnd','Handler_read_rnd_next','Handler_rollback','Handler_savepoint','Handler_savepoint_rollback','Handler_update','Handler_write','Innodb_buffer_pool_reads', 'Innodb_buffer_pool_read_requests','Innodb_DEL','Innodb_QPS','Innodb_TPS','Innodb_rows_inserted','Innodb_rows_updated','Innodb_rows_deleted','Innodb_rows_read','Innodb_row_lock_time','Innodb_row_lock_waits','Key_blocks_not_flushed','Key_blocks_unused','Key_read_requests','Key_reads','Key_write_requests','Key_writes','Max_used_connections','Open_files','Open_tables','Opened_tables','Qcache_free_blocks','Qcache_free_memory','Qcache_hits','Qcache_inserts','Qcache_lowmem_prunes','Qcache_not_cached','Qcache_queries_in_cache','Qcache_total_blocks','Query_time_status','Query_time_binlog','Query_time_process','Query_time_engine','Query_time_subordinate','Query_time_variables','Query_time_avg','Questions','Rpl_semi_sync_main_clients','Rpl_semi_sync_main_net_avg_wait_time','Rpl_semi_sync_main_net_wait_time','Rpl_semi_sync_main_net_waits','Rpl_semi_sync_main_no_times','Rpl_semi_sync_main_no_tx','Rpl_semi_sync_main_status','Rpl_semi_sync_main_timefunc_failures','Rpl_semi_sync_main_tx_avg_wait_time','Rpl_semi_sync_main_tx_wait_time','Rpl_semi_sync_main_tx_waits','Rpl_semi_sync_main_wait_pos_backtraverse','Rpl_semi_sync_main_wait_sessions','Rpl_semi_sync_main_yes_tx','Select_full_join','Select_full_range_join','Select_range','Select_range_check','Select_scan','Subordinate_open_temp_tables','Slow_queries','Sort_merge_passes','Sort_range','Sort_rows','Sort_scan','Table_locks_immediate','Table_locks_waited','Threads_cached','Threads_connected','Threads_created','Threads_running','binary_log_space');"
    //variablesSQL    string = "SHOW GLOBAL VARIABLES;"
    variablesSQL    string = "SHOW GLOBAL VARIABLES WHERE variable_name IN ('version_comment', 'version','log_bin', 'innodb_log_buffer_size','key_buffer_size','key_cache_block_size','max_connections','query_cache_size','table_open_cache','thread_cache_size');"
    subordinateSQL        string = "SHOW SLAVE  STATUS;"
    //processSQL      string = "SHOW PROCESSLIST;"
    //processSQL      string = "SELECT id, IF(state IS NULL OR LENGTH(state)=0, 'none', state) FROM INFORMATION_SCHEMA.PROCESSLIST;"
    perconaSQL      string = "SELECT CONCAT(id,'-',if(command='Query', time_ms/1000, 0)) sqlAvg, IF(state IS NULL OR LENGTH(state)=0, 'none', state) sqlType FROM INFORMATION_SCHEMA.PROCESSLIST;"
    processSQL      string = "SELECT CONCAT(id,'-',if(command='Query', time,         0)) sqlAvg, IF(state IS NULL OR LENGTH(state)=0, 'none', state) sqlType FROM INFORMATION_SCHEMA.PROCESSLIST;"
    engineSQL       string = "SHOW ENGINE INNODB STATUS;"
    binlogSQL       string = "SHOW MASTER LOGS;"
    timeFMT         string = "2006-01-02 15:04:05.000000"
)

type MStatus struct {
    mKey            string
    mValue          string
}

var db          *sql.DB
//var rdb         *redis.Client

func checkError(err error, msg string) {
    if err != nil{
        //log.Fatalln(err)
        //fmt.Println("OUTPUT: ", db_host, msg)
        //fmt.Println(err)
        fmt.Println("-0")
        os.Exit(1)
    }
}

// 定义一个初始化数据库的函数
func initMySQL() (err error) {
    // DSN:Data Source Name
    // dsn := "root:123456@tcp(127.0.0.1:3306)/dba_info"
    dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8",db_user,db_pass,db_host,db_port,db_name)
    // 不会校验账号密码是否正确
    // 注意！！！这里不要使用:=，我们是给全局变量赋值，然后在main函数中使用全局变量db
    db, err = sql.Open("mysql", dsn)
    checkError(err, "MySQL open conncection Failed:" + db_host)
    // 尝试与数据库建立连接（校验dsn是否正确）
    err = db.Ping()
    checkError(err, "Try to connct MySQL Server Failed")
    return nil
}

// 查询多条数据示例
func queryMultiRow(tmpType string, sqlStr string) (map[string]string) {
    err := initMySQL() // 调用输出化数据库的函数
    checkError(err, "Init MySQL server connection failed")

    rows, err := db.Query(sqlStr)
    checkError(err, "Query MySQL server failed")

    // 非常重要：关闭rows释放持有的数据库链接
    defer rows.Close()
    // 循环读取结果集中的数据
    vmap := make(map[string]string)
    for rows.Next() {
        var m MStatus
        err := rows.Scan(&m.mKey, &m.mValue)
        checkError(err, "Scan rows failed")

        //fmt.Printf("%s: %-50s= %-20s\n", curMapName, m.mKey, m.mValue)
        //fmt.Printf("%s\n", m.mKey)
        m.mKey=strings.Replace(m.mKey, "_", "-", -1)
        vmap[m.mKey] = m.mValue
    }
    db.Close()
    return vmap
}

// 查询多字段数据示例
func queryMultiColumn(tmpType string, sqlStr string) (map[string]string) {
    err := initMySQL() // 调用输出化数据库的函数
    checkError(err, "Init MySQL server connection failed")

    rows, err := db.Query(sqlStr)
    checkError(err, "Query MySQL server failed")

    // 非常重要：关闭rows释放持有的数据库链接
    defer rows.Close()
    // 循环读取结果集中的数据
    vmap := make(map[string]string)

    cols, _ := rows.Columns()
    buff := make([]interface{}, len(cols))  // 临时slice
    data := make([]string, len(cols))       // 存数据slice
    for i, _ := range buff {
        buff[i] = &data[i]
    }
    for rows.Next() {
        rows.Scan(buff...)                  // ...是必须的
    }

    for k, col := range data {
        var m MStatus
        m.mKey=strings.Replace(cols[k], "_", "-", -1)
        vmap[m.mKey] = col
    }
    db.Close()
    return vmap

}

func hmsetRedis(tmpName string, tmpMap map[string]string ){
    conn, err := redis.Dial("tcp", redisSTR)
    checkError(err, "Redis conn Failed")

    defer conn.Close()
    _, err1 := conn.Do("HMSet", redis.Args{}.Add(tmpName).AddFlat(tmpMap)...)
    checkError(err1, "Redis HMSet Failed")

}

func hgetallRedis(tmpName string) (map[string]string){
    conn, err := redis.Dial("tcp", redisSTR)
    checkError(err, "Redis conn Failed")

    defer conn.Close()
    tmpMap, err := redis.StringMap(conn.Do("HGetAll", tmpName, ))
    checkError(err, "Redis HGetAll Failed")
    return tmpMap
}

func hmgetRedis(tmpName string, tmpKey string) ([]string){
    conn, err := redis.Dial("tcp", redisSTR)
    checkError(err, "Redis conn Failed")

    defer conn.Close()
    tmpMap, err := redis.Strings(conn.Do("HMGet", tmpName, sqlType+"CreateTime", tmpKey))
    checkError(err, "Redis HMGet Failed")
    return tmpMap
}

func hmgetRedisStrs(tmpName string, tmpKey []string) ([]string){
    conn, err := redis.Dial("tcp", redisSTR)
    checkError(err, "Redis conn Failed")

    defer conn.Close()

    tmpArgs := make([]interface{}, len(tmpKey)+1)
    tmpArgs[0] = tmpName
    for i, v := range tmpKey {
        tmpArgs[i+1] = v
    }

    tmpMap,err := redis.Strings(conn.Do("HMGET", tmpArgs...))
    checkError(err, "Redis HMGet Failed")
    return tmpMap
}


func strToTime(tmpTimeStr string) (time.Time){
    tmpTime, err := time.Parse(timeFMT, tmpTimeStr);
    checkError(err, "Parse time Failed")
    return tmpTime
}

func strToFloat64(tmpStr string) (float64) {
    reg := regexp.MustCompile("[ |,|\\[|\\]|;|-]")
    tmpStr = reg.ReplaceAllString(tmpStr, "")
    //fmt.Println(tmpStr)
    tmpFloat64, err := strconv.ParseFloat(tmpStr, 64)
    checkError(err, "Parse Float64 Failed")
    return tmpFloat64
}

func floatToStr(tmpFloat float64) string {

    // convert float number to string
    tmpDecimal := decimal.NewFromFloat(tmpFloat)
    tmpStr := tmpDecimal.String()
    if strings.Index(tmpStr, ".") > 0 {
        return fmt.Sprintf("%.6f", tmpFloat)
    } else {
        return tmpStr
    }
 }

func mathData() (string){
    // Current Time
    //cTime   := time.Now()
    //curTime := cTime.Format(timeFMT)

    // get his map from Redis
    // his_map = hgetallRedis(hisMapName)
    // get cur map from Redis
    //cur_map = hgetallRedis(curMapName)
    // fmt.Println(hisTime.Format(timeFMT))
    // fmt.Println(curTime)
    // fmt.Println(timeDura)

    // get his map from Redis
    tmpHisMap := hmgetRedis(hisMapName, mo_item)
    // get cur map from Redis
    tmpCurMap := hmgetRedis(curMapName, mo_item)
    if len(tmpHisMap) > 0 && len(tmpCurMap) > 0 {
        hisTime := strToTime(tmpHisMap[0])
        curTime := strToTime(tmpCurMap[0])
        timeDura  = float64(curTime.Sub(hisTime)/time.Millisecond)/1000.0

        if       calcType == "0" {
            switch mo_item {
            case "binary-log-space":
                mo_value = floatToStr(strToFloat64(tmpCurMap[1])/1024.0/1024.0/1024.0)
            case "Relay-Log-Space":
                mo_value = floatToStr(strToFloat64(tmpCurMap[1])/1024.0/1024.0/1024.0)
            default:
                mo_value = tmpCurMap[1]
            }

        }else if calcType == "1" {

            tmp_value := (strToFloat64(tmpCurMap[1]) - strToFloat64(tmpHisMap[1]))/timeDura
            // fmt.Println(tmpCurMap[mo_item], tmpHisMap[mo_item], timeDura)
            mo_value  = floatToStr(tmp_value)

        }else if calcType == "2" {
            switch mo_item {
            case "MySQL-TPS":
                mo_item_strings = []string{sqlType+"CreateTime", "Com-insert", "Com-update", "Com-delete"}
                tmpHisMap := hmgetRedisStrs(hisMapName, mo_item_strings)
                tmpCurMap := hmgetRedisStrs(curMapName, mo_item_strings)
                mo_value = floatToStr(( strToFloat64(tmpCurMap[1]) + strToFloat64(tmpCurMap[2]) + strToFloat64(tmpCurMap[3]) -
                                        strToFloat64(tmpHisMap[1]) - strToFloat64(tmpHisMap[2]) - strToFloat64(tmpHisMap[3]))/timeDura)
            case "MySQL-QPS":
                mo_item_strings = []string{sqlType+"CreateTime", "Com-select"}
                tmpHisMap := hmgetRedisStrs(hisMapName, mo_item_strings)
                tmpCurMap := hmgetRedisStrs(curMapName, mo_item_strings)
                mo_value = floatToStr((strToFloat64(tmpCurMap[1]) - strToFloat64(tmpHisMap[1]))/timeDura)
            case "Innodb-TPS":
                mo_item_strings = []string{sqlType+"CreateTime", "Innodb-rows-deleted", "Innodb-rows-updated", "Innodb-rows-inserted"}
                tmpHisMap := hmgetRedisStrs(hisMapName, mo_item_strings)
                tmpCurMap := hmgetRedisStrs(curMapName, mo_item_strings)
                mo_value = floatToStr(( strToFloat64(tmpCurMap[1]) + strToFloat64(tmpCurMap[2]) + strToFloat64(tmpCurMap[3]) -
                                        strToFloat64(tmpHisMap[1]) - strToFloat64(tmpHisMap[2]) - strToFloat64(tmpHisMap[3]))/timeDura)
            case "Innodb-QPS":
                mo_item_strings = []string{sqlType+"CreateTime", "Innodb-rows-read"}
                tmpCurMap := hmgetRedisStrs(curMapName, mo_item_strings)
                tmpHisMap := hmgetRedisStrs(hisMapName, mo_item_strings)
                mo_value = floatToStr((strToFloat64(tmpCurMap[1]) - strToFloat64(tmpHisMap[1]))/timeDura)
            case "Innodb-DEL":
                mo_item_strings = []string{sqlType+"CreateTime", "Innodb-rows-deleted"}
                tmpCurMap := hmgetRedisStrs(curMapName, mo_item_strings)
                tmpHisMap := hmgetRedisStrs(hisMapName, mo_item_strings)
                mo_value = floatToStr((strToFloat64(tmpCurMap[1]) - strToFloat64(tmpHisMap[1]))/timeDura)
            case "Subordinate-isYes":
                mo_item_strings = []string{sqlType+"CreateTime", "Subordinate-IO-Running", "Subordinate-SQL-Running"}
                tmpCurMap := hmgetRedisStrs(curMapName, mo_item_strings)
                if tmpCurMap[1] == "Yes" && tmpCurMap[2] == "Yes" {
                    mo_value = "1"
                }
            case "semi-sync":
                mo_item_strings = []string{sqlType+"CreateTime", "Rpl-semi-sync-main-status"}
                tmpCurMap := hmgetRedisStrs(curMapName, mo_item_strings)
                if tmpCurMap[1] == "ON" {
                    mo_value = "100"
                } else {
                    mo_value = "60"
                }
            case "MySQL-isAlive":
                mo_item_strings = []string{"StatusCreateTime", "VariablesCreateTime"}
                tmpCurMap := hmgetRedisStrs(curMapName, mo_item_strings)
                curTime := strToTime(getTime())
                statusCreateTime    := strToTime(tmpCurMap[0])
                variablesCreateTime := strToTime(tmpCurMap[1])
                statusDura    := float64(curTime.Sub(statusCreateTime)/time.Millisecond)/1000.0
                variablesDura := float64(curTime.Sub(variablesCreateTime)/time.Millisecond)/1000.0
                //fmt.Println(statusDura, variablesDura)
                if statusDura >= 30 || variablesDura >= 30 {
                    mo_value = "0"
                } else {
                    mo_value = "1"
                }
            }

        }
        nowTime := strToTime(getTime())
        statusCreateTime    := strToTime(tmpCurMap[0])
        if float64(nowTime.Sub(statusCreateTime)/time.Millisecond)/1000.0 < 15 {
            return mo_value
        } else {
            return ""
        }
    } else {
        return ""
    }
}

func saveProcess(tmpMap map[string]string) map[string]string {
    results      := map[string]float64 {
      "State_closing_tables"       : 0,
      "State_copying_to_tmp_table" : 0,
      "State_end"                  : 0,
      "State_freeing_items"        : 0,
      "State_init"                 : 0,
      "State_locked"               : 0,
      "State_login"                : 0,
      "State_preparing"            : 0,
      "State_reading_from_net"     : 0,
      "State_sending_data"         : 0,
      "State_sorting_result"       : 0,
      "State_statistics"           : 0,
      "State_updating"             : 0,
      "State_writing_to_net"       : 0,
      "State_none"                 : 0,
      "State_other"                : 0,
      "SQL_times"                  : 0,
      "SQL_count"                  : 0,
      "SQL_avgtime"                : 0,
    }
    tmpResults   := make(map[string]string)
    for key, value := range tmpMap{
        if len(strings.Replace(value, " ", "", -1)) > 0 {
            reg      := regexp.MustCompile("/^(Table lock|Waiting for .*lock)$/")
            regValue := reg.ReplaceAllString(value, "Locked")
            tmpValue := "State_" + strings.ToLower(strings.Replace(regValue, " ", "_", -1))
            if _, ok := results[tmpValue]; ok {
                results[tmpValue] += 1
                if tmpValue != "State_none" {
                    results["SQL_times"] += strToFloat64(strings.Split(key, "-")[1])
                    results["SQL_count"] += 1
                }

            } else {
                results["State_other"] += 1
            }
        }
    }
    if results["SQL_times"] >= 0 && results["SQL_count"] > 0 {
        results["SQL_avgtime"] = results["SQL_times"] * 1000.0 / results["SQL_count"]
    } else {
        results["SQL_avgtime"] = 0
    }

    for key, value := range results{
        key = strings.Replace(key, "_", "-", -1)
        tmpResults[key]=floatToStr(value)
    }
    return tmpResults
}

func saveInnodb(tmpText string) map[string]string{
    rowText      := strings.Split(tmpText, "\n")
    results      := make(map[string]float64)
    tmpResults   := make(map[string]string)
    reg          := regexp.MustCompile("[.|\\-|log]")
    mysqlVersion := reg.ReplaceAllString(his_map["version"], "")
    lastLine     := ""
    for i, _ := range rowText {
        // replace multi space to only one in middle
        reg  := regexp.MustCompile("\\s+")
        line := reg.ReplaceAllString(rowText[i], " ")
        // replace Prefix space
        reg  = regexp.MustCompile("^\\s+")
        line = reg.ReplaceAllString(line, "")

        row  := strings.Split(line ," ")

        if line == "BACKGROUND THREAD" {
            line_seg="+++BTH:"
        } else if line == "SEMAPHORES" {
            line_seg="+++SEM:"
        } else if line == "LATEST DETECTED DEADLOCK" {
            line_seg="+++LDD:"
        } else if line == "TRANSACTIONS" {
            line_seg="+++TRX:"
        } else if line == "FILE I/O" {
            line_seg="+++FIO:"
        } else if line == "INSERT BUFFER AND ADAPTIVE HASH INDEX" {
            line_seg="+++IBA:"
        } else if line == "LOG" {
            line_seg="+++LOG:"
        } else if line == "BUFFER POOL AND MEMORY" {
            line_seg="+++BPA:"
        } else if line == "INDIVIDUAL BUFFER POOL INFO" {
            line_seg="+++IBP:"
        } else if line == "ROW OPERATIONS" {
            line_seg="+++ROP:"
        }
        newLine := line_seg + line
        //fmt.Println(newLine)
        //fmt.Println(line)
        // -----------------
        // BACKGROUND THREAD
        // -----------------
        // newLine
        if line_seg == "+++BTH:" {

            if strings.HasPrefix(newLine, line_seg + "srv_main_thread loops: "){
                // srv_main_thread loops: 35406745 srv_active, 0 srv_shutdown, 593 srv_idle
                // srv_main_thread log flush and writes: 35402807
                results["main_thread_active"]   = strToFloat64(row[2])
                results["main_thread_shutdown"] = strToFloat64(row[4])
                results["main_thread_idle"]     = strToFloat64(row[6])
            } else if strings.HasPrefix(newLine, line_seg + "srv_main_thread log flush and writes: ") {
                results["main_thread_log_fwrites"]  = strToFloat64(row[5])
            }
        }
        // ----------cat
        // SEMAPHORES
        // ----------
        // newLine
        if line_seg == "+++SEM:" {
            if strings.HasPrefix(newLine, line_seg + "Mutex spin waits") {
                // Mutex spin waits 79626940, rounds 157459864, OS waits 698719
                // Mutex spin waits 0, rounds 247280272495, OS waits 316513438
                results["spin_waits"]  += strToFloat64(row[3])
                results["spin_rounds"] += strToFloat64(row[5])
                results["os_waits"]    += strToFloat64(row[8])
            } else if strings.HasPrefix(newLine, line_seg + "RW-shared spins") {
                // # RW-shared spins 0, rounds 9714605, OS waits 1646243
                // # RW-excl spins 0, rounds 100072735, OS waits 1270363
                // # RW-sx spins 1132098, rounds 21086697, OS waits 273760
                // # Spin rounds per wait: 9714605.00 RW-shared, 100072735.00 RW-excl, 18.63 RW-sx
                results["spin_waits"]  += strToFloat64(strings.Replace(row[2], ",", "", -1))
                results["spin_rounds"] += strToFloat64(strings.Replace(row[4], ",", "", -1))
                results["os_waits"]    += strToFloat64(row[7])
            } else if strings.HasPrefix(newLine, line_seg + "RW-excl spins") {
                // # RW-shared spins 0, rounds 9714605, OS waits 1646243
                // # RW-excl spins 0, rounds 100072735, OS waits 1270363
                // # RW-sx spins 1132098, rounds 21086697, OS waits 273760
                // # Spin rounds per wait: 9714605.00 RW-shared, 100072735.00 RW-excl, 18.63 RW-sx
                results["spin_waits"]  += strToFloat64(strings.Replace(row[2], ",", "", -1))
                results["spin_rounds"] += strToFloat64(strings.Replace(row[4], ",", "", -1))
                results["os_waits"]    += strToFloat64(row[7])
            } else if strings.HasPrefix(newLine, line_seg + "RW-sx spins") {
                // # RW-shared spins 0, rounds 9714605, OS waits 1646243
                // # RW-excl spins 0, rounds 100072735, OS waits 1270363
                // # RW-sx spins 1132098, rounds 21086697, OS waits 273760
                // # Spin rounds per wait: 9714605.00 RW-shared, 100072735.00 RW-excl, 18.63 RW-sx
                results["spin_waits"]  += strToFloat64(strings.Replace(row[2], ",", "", -1))
                results["spin_rounds"] += strToFloat64(strings.Replace(row[4], ",", "", -1))
                results["os_waits"]    += strToFloat64(row[7])
            }
        }
        // ------------
        // TRANSACTIONS
        // ------------
        // newLine
        if line_seg == "+++TRX:" {
            if strings.HasPrefix(newLine, line_seg + "Trx id counter") {
                // Trx id counter 561197120268
                results["innodb_transactions"] = strToFloat64(row[3])
            } else if strings.HasPrefix(newLine, line_seg + "History list length") {
                // History list length 46
                results["history_list"] = strToFloat64(row[3])
            } else if strings.HasPrefix(newLine, line_seg + "---TRANSACTION") {
                // ---TRANSACTION 421492345098080, not started
                results["current_transactions"] += 1
                if strings.Index(line, "ACTIVE") > 0 {
                    results["active_transactions"] += 1
                }
            } else if strings.HasPrefix(newLine, line_seg + "------- TRX HAS BEEN") {
                // ------- TRX HAS BEEN WAITING 32 SEC FOR THIS LOCK TO BE GRANTED:
                results["innodb_lock_wait_secs"] = strToFloat64(row[5])
            } else if strings.HasPrefix(newLine, line_seg) && strings.Index(line, "read views open inside InnoDB") > 0 {
                // 0 read views open inside InnoDB
                results["read_views"] = strToFloat64(row[0])
            } else if strings.HasPrefix(newLine, line_seg + "mysql tables in use") {
                // Trx id counter 439808148
                results["innodb_tables_in_use"] += strToFloat64(row[4])
                results["innodb_locked_tables"] += strToFloat64(row[6])
            } else if strings.HasPrefix(newLine, line_seg) && strings.Index(line, "lock struct(s)") > 0 {

                if strings.HasPrefix(newLine, line_seg + "LOCK WAIT") || strings.HasPrefix(newLine, line_seg + "ROLLING"){
                        results["innodb_lock_structs"] += strToFloat64(row[2])
                        results["locked_transactions"] += 1
                }else{
                        results["innodb_lock_structs"] += strToFloat64(row[0])
                        results["locked_transactions"] += 0
                }
            } else if strings.HasPrefix(newLine, line_seg) && strings.Index(line, "OS file reads, ") > 0 {
                // 8782182 OS file reads, 15635445 OS file writes, 947800 OS fsyncs
                results["file_reads"]  = strToFloat64(row[0])
                results["file_writes"] = strToFloat64(row[4])
                results["file_fsyncs"] = strToFloat64(row[8])
            }
        }
        // --------
        // FILE I/O
        // --------
        // newLine
        if line_seg == "+++FIO:" {
            if strings.HasPrefix(newLine, line_seg + "Pending normal aio reads:") && strings.Index(line, "]") == 0 {
                // Trx id counter 439808148
                results["pending_normal_aio_reads"]  = strToFloat64(row[4])
                results["pending_normal_aio_writes"] = strToFloat64(row[7])
            } else if strings.HasPrefix(newLine, line_seg + "Pending normal aio reads:") && strings.Index(line, "]") > 0 {
                // Pending normal aio reads: [0, 0, 0, 0, 0, 0, 0, 0] , aio writes: [0, 0, 0, 0] ,
                re := regexp.MustCompile("[\\[|\\]]")
                line_columns := re.Split(line, -1)
                aio_reads  := strings.Split(line_columns[1], ",")
                aio_writes := strings.Split(line_columns[3], ",")
                for _, values := range aio_reads {
                    results["pending_normal_aio_reads"]  += strToFloat64(values)
                }
                for _, values := range aio_writes {
                    results["pending_normal_aio_writes"] += strToFloat64(values)
                }
                // results["pending_normal_aio_reads"]  = strToFloat64(row[4]) + strToFloat64(row[5]) + strToFloat64(row[6]) + strToFloat64(row[7])
                // results["pending_normal_aio_writes"] = strToFloat64(row[11]) + strToFloat64(row[12]) + strToFloat64(row[13]) + strToFloat64(row[14])
            } else if strings.HasPrefix(newLine, line_seg + "ibuf aio reads") && strings.Index(line, ":,") == 0 {
                //  ibuf aio reads:, log i/o's:, sync i/o's:
                results["pending_ibuf_aio_reads"]  = strToFloat64(row[3])
                results["pending_aio_log_ios"]     = strToFloat64(row[6])
                results["pending_aio_sync_ios"]    = strToFloat64(row[9])

            } else if strings.HasPrefix(newLine, line_seg + "ibuf aio reads") && strings.Index(line, ":,") > 0 {
                //  ibuf aio reads:, log i/o's:, sync i/o's:
                results["pending_ibuf_aio_reads"]  = 0
                results["pending_aio_log_ios"]     = 0
                results["pending_aio_sync_ios"]    = 0

            } else if strings.HasPrefix(newLine, line_seg + "Pending flushes (fsync)") {
                // Pending flushes (fsync) log: 0; buffer pool: 0
                results["pending_log_flushes"]       = strToFloat64(row[4])
                results["pending_buf_pool_flushes"]  = strToFloat64(row[7])
            } else if strings.HasPrefix(newLine, line_seg) && strings.Index(line, "OS file reads, ") > 0 {
                // 8782182 OS file reads, 15635445 OS file writes, 947800 OS fsyncs
                results["file_reads"]  = strToFloat64(row[0])
                results["file_writes"] = strToFloat64(row[4])
                results["file_fsyncs"] = strToFloat64(row[8])
            }
        }
        // -------------------------------------
        // INSERT BUFFER AND ADAPTIVE HASH INDEX
        // -------------------------------------
        // newLine
        if line_seg == "+++IBA:" {
            if strings.HasPrefix(newLine, line_seg + "Ibuf for space 0: size ") {
                // Older InnoDB code seemed to be ready for an ibuf per tablespace.  It
                // had two lines in the output.  Newer has just one line, see below.
                // Ibuf for space 0: size 1, free list len 887, seg size 889, is not empty
                // Ibuf for space 0: size 1, free list len 887, seg size 889,
                results["ibuf_used_cells"]  = strToFloat64(row[5])
                results["ibuf_free_cells"]  = strToFloat64(row[9])
                results["ibuf_cell_count"]  = strToFloat64(row[12])
            } else if strings.HasPrefix(newLine, line_seg + "Ibuf: size ") {
                // Ibuf: size 1, free list len 4634, seg size 4636,
                results["ibuf_used_cells"]  = strToFloat64(row[2])
                results["ibuf_free_cells"]  = strToFloat64(row[6])
                results["ibuf_cell_count"]  = strToFloat64(row[9])
                if strings.Index(line, "merges") > 0 {
                   results["ibuf_merges"]  = strToFloat64(row[10])
                }
            } else if strings.HasPrefix(lastLine, line_seg + "merged operations:") && strings.Index(line, ", delete mark ") > 0 {
                // Output of show engine innodb status has changed in 5.5
                // merged operations:
                // insert 593983, delete mark 387006, delete 73092
                results["ibuf_inserts"] = strToFloat64(row[1])
                results["ibuf_merged"]  = strToFloat64(row[1]) + strToFloat64(row[4]) + strToFloat64(row[6])
            } else if strings.HasPrefix(newLine, line_seg) && strings.Index(line, "merged recs, ") > 0 {
                // 19817685 inserts, 19817684 merged recs, 3552620 merges
                results["ibuf_inserts"] = strToFloat64(row[0])
                results["ibuf_merged"]  = strToFloat64(row[2])
                results["ibuf_merges"]  = strToFloat64(row[5])

            } else if strings.HasPrefix(newLine, line_seg + "Hash table size ") {
                // In some versions of InnoDB, the used cells is omitted.
                // Hash table size 4425293, used cells 4229064, ....
                // Hash table size 57374437, node heap has 72964 buffer(s) <-- no used cells
                results["hash_index_cells_total"]  = strToFloat64(row[3])
                if strings.Index(line, "used cells") > 0 {
                    results["hash_index_cells_used"]  = strToFloat64(row[6])
                } else {
                    results["hash_index_cells_used"]  = 0.0
                }
            }
        }
        // ---
        // LOG
        // ---
        // newLine
        if line_seg == "+++LOG:" {

            if strings.HasPrefix(newLine, line_seg) && strings.Index(line, " log i/o's done, ") > 0 {
                // 3430041 log i/o's done, 17.44 log i/o's/second
                // 520835887 log i/o's done, 17.28 log i/o's/second, 518724686 syncs, 2980893 checkpoints
                // TODO: graph syncs and checkpoints
                results["log_writes"]  = strToFloat64(row[0])
            } else if strings.HasPrefix(newLine, line_seg) && mysqlVersion < "50700" && strings.Index(line, "pending log writes, ") > 0 {
                // 0 pending log writes, 0 pending chkp writes
                results["pending_log_writes"]  = strToFloat64(row[0])
                results["pending_chkp_writes"] = strToFloat64(row[4])

            } else if strings.HasPrefix(newLine, line_seg) && mysqlVersion > "50700" && strings.Index(line, "pending log flushes, ") > 0 {
                // Post 5.7.x SHOW ENGINE INNODB STATUS syntax
                // 0 pending log flushes, 0 pending chkp writes
                results["pending_log_writes"]  = strToFloat64(row[0])
                results["pending_chkp_writes"] = strToFloat64(row[4])

            } else if strings.HasPrefix(newLine, line_seg + "Log sequence number") {
                // This number is NOT printed in hex in InnoDB plugin.
                // Log sequence number 13093949495856 //plugin
                // Log sequence number 125 3934414864 //normal
                if len(row) == 4 {
                    results["log_bytes_written"] = strToFloat64(row[3])
                } else if len(row) == 5 {
                    results["log_bytes_written"] = strToFloat64(row[3]) + strToFloat64(row[4])
                }
            } else if strings.HasPrefix(newLine, line_seg + "Log flushed up to") {
                // This number is NOT printed in hex in InnoDB plugin.
                // Log flushed up to   13093948219327
                // Log flushed up to   125 3934414864
                if len(row) == 5 {
                    results["log_bytes_flushed"] = strToFloat64(row[4])
                } else if len(row) == 6 {
                    results["log_bytes_flushed"] = strToFloat64(row[4]) + strToFloat64(row[5])
                }
            } else if strings.HasPrefix(newLine, line_seg + "Last checkpoint at") {
                //  Last checkpoint at  125 3934293461
                if len(row) == 4 {
                    results["last_checkpoint"] = strToFloat64(row[3])
                } else if len(row) == 5 {
                    results["last_checkpoint"] = strToFloat64(row[3]) + strToFloat64(row[4])
                }
            }
        }
        // ----------------------
        // BUFFER POOL AND MEMORY
        // ----------------------
        // newLine
        if line_seg == "+++BPA:" {
            if mysqlVersion < "50700" && strings.HasPrefix(newLine, line_seg + "Total memory allocated") && strings.Index(line, "in additional pool allocated") > 0 {
                // Total memory allocated 29642194944; in additional pool allocated 0
                // Total memory allocated by read views 96
                results["total_mem_alloc"]       = strToFloat64(row[3])
                results["additional_pool_alloc"] = strToFloat64(row[8])
            } else if mysqlVersion > "50700" && strings.HasPrefix(newLine, line_seg + "Total large memory allocated") {
                // Post 5.7.x SHOW ENGINE INNODB STATUS syntax
                // Total large memory allocated 2198863872
                results["total_mem_alloc"]       = strToFloat64(row[4])
                results["additional_pool_alloc"] = 0.0
            } else if strings.HasPrefix(newLine, line_seg + "Adaptive hash index ") {
                // Adaptive hash index 1538240664     (186998824 + 1351241840)
                results["adaptive_hash_memory"] = strToFloat64(row[3])
            } else if strings.HasPrefix(newLine, line_seg + "Page hash ") {
                // Page hash           11688584
                results["page_hash_memory"] = strToFloat64(row[2])
            } else if strings.HasPrefix(newLine, line_seg + "Dictionary cache ") {
                // Dictionary cache    145525560     (140250984 + 5274576)
                results["dictionary_cache_memory"] = strToFloat64(row[2])
            } else if strings.HasPrefix(newLine, line_seg + "Dictionary memory allocated ") {
                // Dictionary memory allocated 2046666
                results["dictionary_cache_memory"] = strToFloat64(row[3])
            } else if strings.HasPrefix(newLine, line_seg + "File system ") {
                // File system         313848     (82672 + 231176)
                results["file_system_memory"] = strToFloat64(row[2])
            } else if strings.HasPrefix(newLine, line_seg + "Lock system ") {
                // Lock system         29232616     (29219368 + 13248)
                results["lock_system_memory"] = strToFloat64(row[2])
            } else if strings.HasPrefix(newLine, line_seg + "Recovery system ") {
                // Recovery system     0     (0 + 0)
                results["recovery_system_memory"] = strToFloat64(row[2])
            } else if strings.HasPrefix(newLine, line_seg + "Threads ") {
                // Threads             409336     (406936 + 2400)
                results["thread_hash_memory"] = strToFloat64(row[1])
            } else if strings.HasPrefix(newLine, line_seg + "innodb_io_pattern ") {
                // innodb_io_pattern   0     (0 + 0)
                results["innodb_io_pattern_memory"] = strToFloat64(row[1])

            } else if strings.HasPrefix(newLine, line_seg + "Buffer pool size ") {
                // The " " after size is necessary to avoid matching the wrong line:
                // Buffer pool size        1769471
                // Buffer pool size, bytes 28991012864
                results["pool_size"] = strToFloat64(row[3])
            } else if strings.HasPrefix(newLine, line_seg + "Free buffers ") {
                // Free buffers            0
                results["free_pages"] = strToFloat64(row[2])
            } else if strings.HasPrefix(newLine, line_seg + "Database pages ") {
                // Database pages          1696503
                results["database_pages"] = strToFloat64(row[2])
            } else if strings.HasPrefix(newLine, line_seg + "Modified db pages ") {
                // Modified db pages       160602
                results["modified_pages"] = strToFloat64(row[3])
            } else if strings.HasPrefix(newLine, line_seg + "Pages read ahead ") {
                // Must do this BEFORE the next test, otherwise it'll get fooled by this
                // line from the new plugin (see samples/innodb-015.txt):
                // Pages read ahead 0.00/s, evicted without access 0.06/s
                // TODO: No-op for now, see issue 134.
            } else if strings.HasPrefix(newLine, line_seg + "Pages read ") {
                // Pages read 15240822, created 1770238, written 21705836
                results["pages_read"]    = strToFloat64(row[2])
                results["pages_created"] = strToFloat64(row[4])
                results["pages_written"] = strToFloat64(row[6])
            }
        }
        // --------------
        // ROW OPERATIONS
        // --------------
        // newLine
        if line_seg == "+++ROP:" {
            if strings.HasPrefix(newLine, line_seg + "Number of rows inserted ") {
                // Number of rows inserted 50678311, updated 66425915, deleted 20605903, read 454561562
                results["rows_inserted"] = strToFloat64(row[4])
                results["rows_updated"]  = strToFloat64(row[6])
                results["rows_deleted"]  = strToFloat64(row[8])
                results["rows_read"]     = strToFloat64(row[10])
            } else if strings.HasPrefix(newLine, line_seg) && strings.Index(line, "queries inside InnoDB, ") > 0 {
                // 0 queries inside InnoDB, 0 queries in queue
                results["queries_inside"] = strToFloat64(row[0])
                results["queries_queued"] = strToFloat64(row[4])
            } else if strings.HasPrefix(newLine, line_seg) && strings.Index(line, "read views open inside InnoDB") > 0 {
                // 0 read views open inside InnoDB
                results["read_views"] = strToFloat64(row[0])
            }
        }

        lastLine = newLine
    }
    results["unflushed_log"]        = results["log_bytes_written"] - results["log_bytes_flushed"]
    results["uncheckpointed_bytes"] = results["log_bytes_written"] - results["last_checkpoint"]
    results["Key_buf_bytes_used"]   = strToFloat64(variablesCurMap["key-buffer-size"]) - strToFloat64(statusCurMap["Key-blocks-unused"]) * strToFloat64(variablesCurMap["key-cache-block-size"])
    results["Key_blocks_not_flushed"]   = strToFloat64(statusCurMap["Key-blocks-not-flushed"]) - strToFloat64(variablesCurMap["key-cache-block-size"])
    for k, v := range results {
        k = strings.Replace(k, "_", "-", -1)
        tmpResults[k] = floatToStr(v)
    }

    //fmt.Println(tmpResults)
    return tmpResults
}

func saveBinlog(tmpMap map[string]string) map[string]string {
    results      := map[string]float64 {
      "binary-log-space"       : 0,
    }
    tmpResults   := make(map[string]string)
    if len(tmpMap) > 0 {
        for key, value := range tmpMap{
            if key != "BinlogCreateTime" {
                results["binary-log-space"] += strToFloat64(value)
            }
        }
        tmpResults["binary-log-space"] = floatToStr(results["binary-log-space"])
    } else {
        tmpResults["binary-log-space"] = "0"
    }

    return tmpResults
}

func addTime (tmpType string, cTime string, tmpMap map[string]string) map[string]string {
    tmpMap[tmpType+"CreateTime"]=cTime
    //fmt.Println(cTime)
    tmpCommitTime := time.Now()
    tmpMap[tmpType+"CommitTime"]=tmpCommitTime.Format(timeFMT)
    return tmpMap
}

func getTime () string {
    tmpCreateTime := time.Now()
    tmpCTime := tmpCreateTime.Format(timeFMT)
    return tmpCTime
}
func saveData() (string) {
    responseTime := make(map[string]string)

    // #
    // # History Map Save
    // #
    // Save curMap to ParameterMap
    his_map = hgetallRedis(curMapName)

    //fmt.Println(len(his_map), len(cur_map))
    //cur_map = addTime("VariablesCreate", cur_map)
    variablesCTime  := getTime()
    variablesCurMap = queryMultiRow("Variables", variablesSQL)
    variablesCurMap =  addTime("Variables", variablesCTime, variablesCurMap)
    hmsetRedis(curMapName, variablesCurMap)

    //cur_map = addTime("StatusCreate", cur_map)
    statusCTime  := getTime()
    statusCurMap = queryMultiRow("Status", statusSQL)
    statusCurMap =  addTime("Status", statusCTime, statusCurMap)
    hmsetRedis(curMapName, statusCurMap)

    //cur_map = addTime("ProcessCreate", cur_map)
    processCTime  := getTime()
    if strings.Split(variablesCurMap["version-comment"], " ")[0] == "Percona"{
        processCurMap = queryMultiRow("Process", perconaSQL)
    } else {
        processCurMap = queryMultiRow("Process", processSQL)
    }
    processCurMap =  saveProcess(processCurMap)
    processCurMap =  addTime("Process", processCTime, processCurMap)
    //fmt.Println(cur_map)
    hmsetRedis(curMapName, processCurMap)

    //cur_map = addTime("SubordinateCreate", cur_map)
    subordinateCTime  := getTime()
    subordinateCurMap = queryMultiColumn("Subordinate", subordinateSQL)
    if len(subordinateCurMap["Subordinate-SQL-Running"]) > 0 && len(subordinateCurMap["Subordinate-IO-Running"]) > 0 {
        subordinateCurMap["MySQL-Role"] = "0"
    } else {
        subordinateCurMap["MySQL-Role"] = "1"
    }
    subordinateCurMap = addTime("Subordinate", subordinateCTime, subordinateCurMap)
    hmsetRedis(curMapName, subordinateCurMap)

    // #
    // # History Map Save
    // #
    // Save ParameterMap to Redis
    if len(his_map) > 0 {
        hmsetRedis(hisMapName, his_map)
    }

    // IF log-bin is enabled
    if variablesCurMap["log-bin"] == "ON" {
        //cur_map = addTime("BinlogCreate", cur_map)
        binlogCTime  := getTime()
        binlogCurMap := queryMultiRow("Binlog", binlogSQL)
        binlogCurMap = saveBinlog(binlogCurMap)
        binlogCurMap = addTime("Binlog", binlogCTime, binlogCurMap)
        //fmt.Println(cur_map)
        hmsetRedis(curMapName, binlogCurMap)
    }

    //cur_map = addTime("EngineCreate", cur_map)
    engineCTime  := getTime()
    engineCurMap := queryMultiColumn("Engine", engineSQL)
    engineCurMap = saveInnodb(engineCurMap["Status"])
    engineCurMap = addTime("Engine", engineCTime, engineCurMap)
    hmsetRedis(curMapName, engineCurMap)

    if len(statusCurMap) > 0 {
        statusCreateTime := strToTime(statusCurMap["StatusCreateTime"])
        statusCommitTime := strToTime(statusCurMap["StatusCommitTime"])
        statusAvgTime    := floatToStr(float64(statusCommitTime.Sub(statusCreateTime)/time.Millisecond/1.00))
        responseTime["Query-time-status"] = statusAvgTime
        //fmt.Println(statusAvgTime)
    }
    if len(processCurMap) > 0 {
        processCreateTime := strToTime(processCurMap["ProcessCreateTime"])
        processCommitTime := strToTime(processCurMap["ProcessCommitTime"])
        processAvgTime    := floatToStr(float64(processCommitTime.Sub(processCreateTime)/time.Millisecond/1.0))
        responseTime["Query-time-process"] = processAvgTime
        //fmt.Println(processAvgTime)
    }
    if len(engineCurMap) > 0 {
        engineCreateTime := strToTime(engineCurMap["EngineCreateTime"])
        engineCommitTime := strToTime(engineCurMap["EngineCommitTime"])
        engineAvgTime    := floatToStr(float64(engineCommitTime.Sub(engineCreateTime)/time.Millisecond/1.0))
        responseTime["Query-time-engine"] = engineAvgTime
        //fmt.Println(engineAvgTime)
    }
    if len(subordinateCurMap) > 0 {
        subordinateCreateTime := strToTime(subordinateCurMap["SubordinateCreateTime"])
        subordinateCommitTime := strToTime(subordinateCurMap["SubordinateCommitTime"])
        subordinateAvgTime    := floatToStr(float64(subordinateCommitTime.Sub(subordinateCreateTime)/time.Millisecond/1.0))
        responseTime["Query-time-subordinate"] = subordinateAvgTime
        //fmt.Println(subordinateAvgTime)
    }
    if len(variablesCurMap) > 0 {
        variablesCreateTime := strToTime(variablesCurMap["VariablesCreateTime"])
        variablesCommitTime := strToTime(variablesCurMap["VariablesCommitTime"])
        variablesAvgTime    := floatToStr(float64(variablesCommitTime.Sub(variablesCreateTime)/time.Millisecond/1.0))
        responseTime["Query-time-variables"] = variablesAvgTime
        //fmt.Println(variablesAvgTime)
    }
    if len(binlogCurMap) > 0 {
        binlogCreateTime := strToTime(binlogCurMap["BinlogCreateTime"])
        binlogCommitTime := strToTime(binlogCurMap["BinlogCommitTime"])
        binlogAvgTime    := floatToStr(float64(binlogCommitTime.Sub(binlogCreateTime)/time.Millisecond/1.0))
        responseTime["Query-time-binlog"] = binlogAvgTime
        //fmt.Println(binlogAvgTime)
    }

    if len(statusCurMap) > 0 && len(variablesCurMap) > 0 {
        hisTmpTime  := strToTime(statusCurMap["StatusCreateTime"])
        curTmpTime  := strToTime(engineCurMap["EngineCommitTime"])
        avgTime := floatToStr(float64(curTmpTime.Sub(hisTmpTime)/time.Millisecond/6))
        responseTime["Query-time-avg"] = avgTime
        hmsetRedis(curMapName, responseTime)
        //fmt.Println(responseTime)
        TimeDura := floatToStr(float64(curTmpTime.Sub(hisTmpTime)/time.Millisecond))
        return TimeDura
    } else {
        return "-0"
    }
}
func confInit(tmpStr string) (string, string) {
    var tmpUser string
    var tmpPass string
    file ,err := os.Open("/etc/zabbix/script/RDS_INFO.conf")
    checkError(err, "Conf File Init Failed")
    defer file.Close()
    scanner := bufio.NewScanner(file)
    scanner.Split(bufio.ScanLines)
    for scanner.Scan() {
        reg  := regexp.MustCompile("\\s+")
        line := reg.ReplaceAllString(scanner.Text(), " ")
        rows := strings.Split(line, " ")
        //fmt.Println(rows[0], rows[1])
        if rows[0] == tmpStr {
            tmpUser = rows[1]
            tmpPass = rows[2]
        }
    }
    // fmt.Println(tmpMap)
    if len(tmpUser) > 0 {
        return tmpUser, tmpPass
    } else {
        return "", ""
    }
}

func paraInit() {
    // mysql -usys_root -psys_root.5mY2fxd9EsspZ8zAUixCgrF6prQ2aLtA
    flag_host := flag.String("host", "127.0.0.1",         "MySQL Server's Host")
    flag_port := flag.String("port", "3306",              "MySQL Server's Port")
    flag_user := flag.String("user", "root",              "MySQL Server's User")
    flag_pass := flag.String("pass", "123456",            "MySQL Server's Pass")
    flag_name := flag.String("name", "information_schema","MySQL Database Name")
    flag_item := flag.String("item", "MySQL-TPS",         "MySQL Template Item")
    flag_conf := flag.String("conf", "8.8.8.8_3306",      "MySQL Server's Conf")
    flag.Parse()
    db_host = *flag_host
    db_port = *flag_port
    db_user = *flag_user
    db_pass = *flag_pass
    db_name = *flag_name
    mo_item = *flag_item
    if *flag_conf != "8.8.8.8_3306"  {
        db_host_string := strings.Split(*flag_conf, "_")
        db_host = db_host_string[0]
        db_port = db_host_string[1]
        if mo_item == "SaveData" {
            db_user, db_pass = confInit(*flag_conf)
        }

    }

}

func main() {
    MySQLItemInit()
    paraInit()
    hisMapName="goMySQL_his_" + db_host + "_" + db_port
    curMapName="goMySQL_cur_" + db_host + "_" + db_port
    //fmt.Println(MySQLMonitor[mo_item])
    switch mo_item {
    case "pool-reads":
        mo_item = "Innodb-buffer-pool-reads"
    case "pool-read-requests":
        mo_item = "Innodb-buffer-pool-read-requests"
    case "table-cache":
        mo_item = "table-open-cache"
    case "subordinate-running":
        mo_item = "Subordinate-isYes"
    case "response-time":
        mo_item = "Query-time-avg"
    case "subordinate-lag":
        mo_item = "Seconds-Behind-Main"
    case "relay-log-space":
        mo_item = "Relay-Log-Space"
    case "semisync-clients":
        mo_item = "Rpl-semi-sync-main-clients"
    case "semisync-net-avg-wait-time":
        mo_item = "Rpl-semi-sync-main-net-avg-wait-time"
    case "semisync-net-wait-time":
        mo_item = "Rpl-semi-sync-main-net-wait-time"
    case "semisync-net-waits":
        mo_item = "Rpl-semi-sync-main-net-waits"
    case "semisync-no-times":
        mo_item = "Rpl-semi-sync-main-no-times"
    case "semisync-no-tx":
        mo_item = "Rpl-semi-sync-main-no-tx"
    //    mo_item = "Rpl-semi-sync-main-status"
    case "semisync-timefunc-failures":
        mo_item = "Rpl-semi-sync-main-timefunc-failures"
    case "semisync-tx-avg-wait-time":
        mo_item = "Rpl-semi-sync-main-tx-avg-wait-time"
    case "semisync-tx-wait-time":
        mo_item = "Rpl-semi-sync-main-tx-wait-time"
    case "semisync-tx-waits":
        mo_item = "Rpl-semi-sync-main-tx-waits"
    case "semisync-wait-pos-backtraverse":
        mo_item = "Rpl-semi-sync-main-wait-pos-backtraverse"
    case "semisync-wait-sessions":
        mo_item = "Rpl-semi-sync-main-wait-sessions"
    case "semisync-yes-tx":
        mo_item = "Rpl-semi-sync-main-yes-tx"
    }
    if _, ok := MySQLMonitor[mo_item]; ok {
        sqlType  = strings.Split(MySQLMonitor[mo_item], "_")[0]
        calcType = strings.Split(MySQLMonitor[mo_item], "_")[1]
        //fmt.Println(sqlType, calcType)
        //fmt.Println(db_host, db_port, db_user, db_pass, db_name)
        if sqlType == "saveData" {
            xyz_value = saveData()
        } else {
            xyz_value = mathData()
        }
        if len(xyz_value) > 0 {
            fmt.Println(xyz_value)
        } else {
            fmt.Println("-0")
        }

    } else {
        fmt.Println("MySQLMonitor's Item not exists: " + mo_item)
    }
}

func MySQLItemInit(){
    MySQLMonitor = map[string]string {
        // alias
        "Query-time-status"                                  : "Status_0",
        "Query-time-binlog"                                  : "Status_0",
        "Query-time-process"                                 : "Status_0",
        "Query-time-engine"                                  : "Status_0",
        "Query-time-subordinate"                                   : "Status_0",
        "Query-time-variables"                               : "Status_0",
        "Query-time-avg"                                     : "Status_0",
        "pool-reads"                                         : "Status_0",
        "pool-read-requests"                                 : "Status_0",
        "table-cache"                                        : "Status_0",
        "subordinate-running"                                      : "Status_0",
        //"subordinate-stopped"                                      : "Status_0",
        // saveData
        "SaveData"                                           : "saveData_0",
        "semisync-clients"                                   : "Status_0",
        "semisync-net-avg-wait-time"                         : "Status_0",
        "semisync-net-wait-time"                             : "Status_0",
        "semisync-net-waits"                                 : "Status_0",
        "semisync-no-times"                                  : "Status_0",
        "semisync-no-tx"                                     : "Status_0",
        "semisync-status"                                    : "Status_0",
        "semisync-timefunc-failures"                         : "Status_0",
        "semisync-tx-avg-wait-time"                          : "Status_0",
        "semisync-tx-wait-time"                              : "Status_0",
        "semisync-tx-waits"                                  : "Status_0",
        "semisync-wait-pos-backtraverse"                     : "Status_0",
        "semisync-wait-sessions"                             : "Status_0",
        "semisync-yes-tx"                                    : "Status_0",

        // caluSQL
        "MySQL-isAlive"                                      : "Status_2",
        "SQL-avgtime"                                        : "Status_0",
        "MySQL-TPS"                                          : "Status_2",
        "MySQL-QPS"                                          : "Status_2",
        "Innodb-TPS"                                         : "Status_2",
        "Innodb-QPS"                                         : "Status_2",
        "Innodb-DEL"                                         : "Status_2",
        "Subordinate-isYes"                                        : "Status_2",
        "semi-sync"                                          : "Status_2",
        "subordinate-lag"                                          : "Status_2",
        "response-time"                                      : "Status_2",
        "relay-log-space"                                    : "Status_2",
        // binlogSQL
        "binary-log-space"                                   : "Status_0",
        // subordinateSQL
        "MySQL-Role"                                         : "Subordinate_0",
        "Relay-Log-Space"                                    : "Subordinate_0",
        "Subordinate-SQL-Running"                                  : "Subordinate_0",
        "Subordinate-IO-Running"                                   : "Subordinate_0",
        "Seconds-Behind-Main"                              : "Subordinate_0",
        "Rpl-semi-sync-main-status"                        : "Subordinate_0",
        // statusSQL   string = "SHOW GLOBAL STATUS;"
        "Aborted-clients"                                    : "Status_0",
        "Aborted-connects"                                   : "Status_0",
        "Binlog-cache-disk-use"                              : "Status_0",
        "Binlog-cache-use"                                   : "Status_0",
        "Binlog-stmt-cache-disk-use"                         : "Status_0",
        "Binlog-stmt-cache-use"                              : "Status_0",
        "Bytes-received"                                     : "Status_0",
        "Bytes-sent"                                         : "Status_0",
        "Com-admin-commands"                                 : "Status_0",
        "Com-assign-to-keycache"                             : "Status_0",
        "Com-alter-db"                                       : "Status_0",
        "Com-alter-db-upgrade"                               : "Status_0",
        "Com-alter-event"                                    : "Status_0",
        "Com-alter-function"                                 : "Status_0",
        "Com-alter-instance"                                 : "Status_0",
        "Com-alter-procedure"                                : "Status_0",
        "Com-alter-server"                                   : "Status_0",
        "Com-alter-table"                                    : "Status_0",
        "Com-alter-tablespace"                               : "Status_0",
        "Com-alter-user"                                     : "Status_0",
        "Com-analyze"                                        : "Status_0",
        "Com-begin"                                          : "Status_0",
        "Com-binlog"                                         : "Status_0",
        "Com-call-procedure"                                 : "Status_0",
        "Com-change-db"                                      : "Status_0",
        "Com-change-main"                                  : "Status_0",
        "Com-change-repl-filter"                             : "Status_0",
        "Com-check"                                          : "Status_0",
        "Com-checksum"                                       : "Status_0",
        "Com-commit"                                         : "Status_0",
        "Com-create-db"                                      : "Status_0",
        "Com-create-event"                                   : "Status_0",
        "Com-create-function"                                : "Status_0",
        "Com-create-index"                                   : "Status_0",
        "Com-create-procedure"                               : "Status_0",
        "Com-create-server"                                  : "Status_0",
        "Com-create-table"                                   : "Status_0",
        "Com-create-trigger"                                 : "Status_0",
        "Com-create-udf"                                     : "Status_0",
        "Com-create-user"                                    : "Status_0",
        "Com-create-view"                                    : "Status_0",
        "Com-dealloc-sql"                                    : "Status_0",
        "Com-delete"                                         : "Status_0",
        "Com-delete-multi"                                   : "Status_0",
        "Com-do"                                             : "Status_0",
        "Com-drop-db"                                        : "Status_0",
        "Com-drop-event"                                     : "Status_0",
        "Com-drop-function"                                  : "Status_0",
        "Com-drop-index"                                     : "Status_0",
        "Com-drop-procedure"                                 : "Status_0",
        "Com-drop-server"                                    : "Status_0",
        "Com-drop-table"                                     : "Status_0",
        "Com-drop-trigger"                                   : "Status_0",
        "Com-drop-user"                                      : "Status_0",
        "Com-drop-view"                                      : "Status_0",
        "Com-empty-query"                                    : "Status_0",
        "Com-execute-sql"                                    : "Status_0",
        "Com-explain-other"                                  : "Status_0",
        "Com-flush"                                          : "Status_0",
        "Com-get-diagnostics"                                : "Status_0",
        "Com-grant"                                          : "Status_0",
        "Com-ha-close"                                       : "Status_0",
        "Com-ha-open"                                        : "Status_0",
        "Com-ha-read"                                        : "Status_0",
        "Com-help"                                           : "Status_0",
        "Com-insert"                                         : "Status_0",
        "Com-insert-select"                                  : "Status_0",
        "Com-install-plugin"                                 : "Status_0",
        "Com-kill"                                           : "Status_0",
        "Com-load"                                           : "Status_0",
        "Com-lock-tables"                                    : "Status_0",
        "Com-optimize"                                       : "Status_0",
        "Com-preload-keys"                                   : "Status_0",
        "Com-prepare-sql"                                    : "Status_0",
        "Com-purge"                                          : "Status_0",
        "Com-purge-before-date"                              : "Status_0",
        "Com-release-savepoint"                              : "Status_0",
        "Com-rename-table"                                   : "Status_0",
        "Com-rename-user"                                    : "Status_0",
        "Com-repair"                                         : "Status_0",
        "Com-replace"                                        : "Status_0",
        "Com-replace-select"                                 : "Status_0",
        "Com-reset"                                          : "Status_0",
        "Com-resignal"                                       : "Status_0",
        "Com-revoke"                                         : "Status_0",
        "Com-revoke-all"                                     : "Status_0",
        "Com-rollback"                                       : "Status_0",
        "Com-rollback-to-savepoint"                          : "Status_0",
        "Com-savepoint"                                      : "Status_0",
        "Com-select"                                         : "Status_0",
        "Com-set-option"                                     : "Status_0",
        "Com-signal"                                         : "Status_0",
        "Com-show-binlog-events"                             : "Status_0",
        "Com-show-binlogs"                                   : "Status_0",
        "Com-show-charsets"                                  : "Status_0",
        "Com-show-collations"                                : "Status_0",
        "Com-show-create-db"                                 : "Status_0",
        "Com-show-create-event"                              : "Status_0",
        "Com-show-create-func"                               : "Status_0",
        "Com-show-create-proc"                               : "Status_0",
        "Com-show-create-table"                              : "Status_0",
        "Com-show-create-trigger"                            : "Status_0",
        "Com-show-databases"                                 : "Status_0",
        "Com-show-engine-logs"                               : "Status_0",
        "Com-show-engine-mutex"                              : "Status_0",
        "Com-show-engine-status"                             : "Status_0",
        "Com-show-events"                                    : "Status_0",
        "Com-show-errors"                                    : "Status_0",
        "Com-show-fields"                                    : "Status_0",
        "Com-show-function-code"                             : "Status_0",
        "Com-show-function-status"                           : "Status_0",
        "Com-show-grants"                                    : "Status_0",
        "Com-show-keys"                                      : "Status_0",
        "Com-show-main-status"                             : "Status_0",
        "Com-show-open-tables"                               : "Status_0",
        "Com-show-plugins"                                   : "Status_0",
        "Com-show-privileges"                                : "Status_0",
        "Com-show-procedure-code"                            : "Status_0",
        "Com-show-procedure-status"                          : "Status_0",
        "Com-show-processlist"                               : "Status_0",
        "Com-show-profile"                                   : "Status_0",
        "Com-show-profiles"                                  : "Status_0",
        "Com-show-relaylog-events"                           : "Status_0",
        "Com-show-subordinate-hosts"                               : "Status_0",
        "Com-show-subordinate-status"                              : "Status_0",
        "Com-show-status"                                    : "Status_0",
        "Com-show-storage-engines"                           : "Status_0",
        "Com-show-table-status"                              : "Status_0",
        "Com-show-tables"                                    : "Status_0",
        "Com-show-triggers"                                  : "Status_0",
        "Com-show-variables"                                 : "Status_0",
        "Com-show-warnings"                                  : "Status_0",
        "Com-show-create-user"                               : "Status_0",
        "Com-shutdown"                                       : "Status_0",
        "Com-subordinate-start"                                    : "Status_0",
        "Com-subordinate-stop"                                     : "Status_0",
        "Com-group-replication-start"                        : "Status_0",
        "Com-group-replication-stop"                         : "Status_0",
        "Com-stmt-execute"                                   : "Status_0",
        "Com-stmt-close"                                     : "Status_0",
        "Com-stmt-fetch"                                     : "Status_0",
        "Com-stmt-prepare"                                   : "Status_0",
        "Com-stmt-reset"                                     : "Status_0",
        "Com-stmt-send-long-data"                            : "Status_0",
        "Com-truncate"                                       : "Status_0",
        "Com-uninstall-plugin"                               : "Status_0",
        "Com-unlock-tables"                                  : "Status_0",
        "Com-update"                                         : "Status_0",
        "Com-update-multi"                                   : "Status_0",
        "Com-xa-commit"                                      : "Status_0",
        "Com-xa-end"                                         : "Status_0",
        "Com-xa-prepare"                                     : "Status_0",
        "Com-xa-recover"                                     : "Status_0",
        "Com-xa-rollback"                                    : "Status_0",
        "Com-xa-start"                                       : "Status_0",
        "Com-stmt-reprepare"                                 : "Status_0",
        "Connection-errors-accept"                           : "Status_0",
        "Connection-errors-internal"                         : "Status_0",
        "Connection-errors-max-connections"                  : "Status_0",
        "Connection-errors-peer-address"                     : "Status_0",
        "Connection-errors-select"                           : "Status_0",
        "Connection-errors-tcpwrap"                          : "Status_0",
        "Connections"                                        : "Status_0",
        "Created-tmp-disk-tables"                            : "Status_0",
        "Created-tmp-files"                                  : "Status_0",
        "Created-tmp-tables"                                 : "Status_0",
        "Delayed-errors"                                     : "Status_0",
        "Delayed-insert-threads"                             : "Status_0",
        "Delayed-writes"                                     : "Status_0",
        "Flush-commands"                                     : "Status_0",
        "Handler-commit"                                     : "Status_0",
        "Handler-delete"                                     : "Status_0",
        "Handler-discover"                                   : "Status_0",
        "Handler-external-lock"                              : "Status_0",
        "Handler-mrr-init"                                   : "Status_0",
        "Handler-prepare"                                    : "Status_0",
        "Handler-read-first"                                 : "Status_0",
        "Handler-read-key"                                   : "Status_0",
        "Handler-read-last"                                  : "Status_0",
        "Handler-read-next"                                  : "Status_0",
        "Handler-read-prev"                                  : "Status_0",
        "Handler-read-rnd"                                   : "Status_0",
        "Handler-read-rnd-next"                              : "Status_0",
        "Handler-rollback"                                   : "Status_0",
        "Handler-savepoint"                                  : "Status_0",
        "Handler-savepoint-rollback"                         : "Status_0",
        "Handler-update"                                     : "Status_0",
        "Handler-write"                                      : "Status_0",
        "Innodb-buffer-pool-dump-status"                     : "Status_0",
        "Innodb-buffer-pool-load-status"                     : "Status_0",
        "Innodb-buffer-pool-resize-status"                   : "Status_0",
        "Innodb-buffer-pool-pages-data"                      : "Status_0",
        "Innodb-buffer-pool-bytes-data"                      : "Status_0",
        "Innodb-buffer-pool-pages-dirty"                     : "Status_0",
        "Innodb-buffer-pool-bytes-dirty"                     : "Status_0",
        "Innodb-buffer-pool-pages-flushed"                   : "Status_0",
        "Innodb-buffer-pool-pages-free"                      : "Status_0",
        "Innodb-buffer-pool-pages-misc"                      : "Status_0",
        "Innodb-buffer-pool-pages-total"                     : "Status_0",
        "Innodb-buffer-pool-read-ahead-rnd"                  : "Status_0",
        "Innodb-buffer-pool-read-ahead"                      : "Status_0",
        "Innodb-buffer-pool-read-ahead-evicted"              : "Status_0",
        "Innodb-buffer-pool-read-requests"                   : "Status_0",
        "Innodb-buffer-pool-reads"                           : "Status_0",
        "Innodb-buffer-pool-wait-free"                       : "Status_0",
        "Innodb-buffer-pool-write-requests"                  : "Status_0",
        "Innodb-data-fsyncs"                                 : "Status_0",
        "Innodb-data-pending-fsyncs"                         : "Status_0",
        "Innodb-data-pending-reads"                          : "Status_0",
        "Innodb-data-pending-writes"                         : "Status_0",
        "Innodb-data-read"                                   : "Status_0",
        "Innodb-data-reads"                                  : "Status_0",
        "Innodb-data-writes"                                 : "Status_0",
        "Innodb-data-written"                                : "Status_0",
        "Innodb-dblwr-pages-written"                         : "Status_0",
        "Innodb-dblwr-writes"                                : "Status_0",
        "Innodb-log-waits"                                   : "Status_0",
        "Innodb-log-write-requests"                          : "Status_0",
        "Innodb-log-writes"                                  : "Status_0",
        "Innodb-os-log-fsyncs"                               : "Status_0",
        "Innodb-os-log-pending-fsyncs"                       : "Status_0",
        "Innodb-os-log-pending-writes"                       : "Status_0",
        "Innodb-os-log-written"                              : "Status_0",
        "Innodb-page-size"                                   : "Status_0",
        "Innodb-pages-created"                               : "Status_0",
        "Innodb-pages-read"                                  : "Status_0",
        "Innodb-pages-written"                               : "Status_0",
        "Innodb-row-lock-current-waits"                      : "Status_0",
        "Innodb-row-lock-time"                               : "Status_0",
        "Innodb-row-lock-time-avg"                           : "Status_0",
        "Innodb-row-lock-time-max"                           : "Status_0",
        "Innodb-row-lock-waits"                              : "Status_0",
        "Innodb-rows-deleted"                                : "Status_1",
        "Innodb-rows-inserted"                               : "Status_1",
        "Innodb-rows-read"                                   : "Status_1",
        "Innodb-rows-updated"                                : "Status_1",
        "Innodb-num-open-files"                              : "Status_0",
        "Innodb-truncated-status-writes"                     : "Status_0",
        "Innodb-available-undo-logs"                         : "Status_0",
        "Key-blocks-not-flushed"                             : "Status_0",
        "Key-blocks-unused"                                  : "Status_0",
        "Key-blocks-used"                                    : "Status_0",
        "Key-read-requests"                                  : "Status_0",
        "Key-reads"                                          : "Status_0",
        "Key-write-requests"                                 : "Status_0",
        "Key-writes"                                         : "Status_0",
        "Locked-connects"                                    : "Status_0",
        "Max-execution-time-exceeded"                        : "Status_0",
        "Max-execution-time-set"                             : "Status_0",
        "Max-execution-time-set-failed"                      : "Status_0",
        "Max-used-connections"                               : "Status_0",
        "Max-used-connections-time"                          : "Status_0",
        "Not-flushed-delayed-rows"                           : "Status_0",
        "Ongoing-anonymous-transaction-count"                : "Status_0",
        "Open-files"                                         : "Status_0",
        "Open-streams"                                       : "Status_0",
        "Open-table-definitions"                             : "Status_0",
        "Open-tables"                                        : "Status_0",
        "Opened-files"                                       : "Status_0",
        "Opened-table-definitions"                           : "Status_0",
        "Opened-tables"                                      : "Status_0",
        "Performance-schema-accounts-lost"                   : "Status_0",
        "Performance-schema-cond-classes-lost"               : "Status_0",
        "Performance-schema-cond-instances-lost"             : "Status_0",
        "Performance-schema-digest-lost"                     : "Status_0",
        "Performance-schema-file-classes-lost"               : "Status_0",
        "Performance-schema-file-handles-lost"               : "Status_0",
        "Performance-schema-file-instances-lost"             : "Status_0",
        "Performance-schema-hosts-lost"                      : "Status_0",
        "Performance-schema-index-stat-lost"                 : "Status_0",
        "Performance-schema-locker-lost"                     : "Status_0",
        "Performance-schema-memory-classes-lost"             : "Status_0",
        "Performance-schema-metadata-lock-lost"              : "Status_0",
        "Performance-schema-mutex-classes-lost"              : "Status_0",
        "Performance-schema-mutex-instances-lost"            : "Status_0",
        "Performance-schema-nested-statement-lost"           : "Status_0",
        "Performance-schema-prepared-statements-lost"        : "Status_0",
        "Performance-schema-program-lost"                    : "Status_0",
        "Performance-schema-rwlock-classes-lost"             : "Status_0",
        "Performance-schema-rwlock-instances-lost"           : "Status_0",
        "Performance-schema-session-connect-attrs-lost"      : "Status_0",
        "Performance-schema-socket-classes-lost"             : "Status_0",
        "Performance-schema-socket-instances-lost"           : "Status_0",
        "Performance-schema-stage-classes-lost"              : "Status_0",
        "Performance-schema-statement-classes-lost"          : "Status_0",
        "Performance-schema-table-handles-lost"              : "Status_0",
        "Performance-schema-table-instances-lost"            : "Status_0",
        "Performance-schema-table-lock-stat-lost"            : "Status_0",
        "Performance-schema-thread-classes-lost"             : "Status_0",
        "Performance-schema-thread-instances-lost"           : "Status_0",
        "Performance-schema-users-lost"                      : "Status_0",
        "Prepared-stmt-count"                                : "Status_0",
        "Qcache-free-blocks"                                 : "Status_0",
        "Qcache-free-memory"                                 : "Status_0",
        "Qcache-hits"                                        : "Status_0",
        "Qcache-inserts"                                     : "Status_0",
        "Qcache-lowmem-prunes"                               : "Status_0",
        "Qcache-not-cached"                                  : "Status_0",
        "Qcache-queries-in-cache"                            : "Status_0",
        "Qcache-total-blocks"                                : "Status_0",
        "Queries"                                            : "Status_0",
        "Questions"                                          : "Status_0",
        "Rpl-semi-sync-main-clients"                       : "Status_0",
        "Rpl-semi-sync-main-net-avg-wait-time"             : "Status_0",
        "Rpl-semi-sync-main-net-wait-time"                 : "Status_0",
        "Rpl-semi-sync-main-net-waits"                     : "Status_0",
        "Rpl-semi-sync-main-no-times"                      : "Status_0",
        "Rpl-semi-sync-main-no-tx"                         : "Status_0",
//        "Rpl-semi-sync-main-status"                        : "Status_0",
        "Rpl-semi-sync-main-timefunc-failures"             : "Status_0",
        "Rpl-semi-sync-main-tx-avg-wait-time"              : "Status_0",
        "Rpl-semi-sync-main-tx-wait-time"                  : "Status_0",
        "Rpl-semi-sync-main-tx-waits"                      : "Status_0",
        "Rpl-semi-sync-main-wait-pos-backtraverse"         : "Status_0",
        "Rpl-semi-sync-main-wait-sessions"                 : "Status_0",
        "Rpl-semi-sync-main-yes-tx"                        : "Status_0",
        "Select-full-join"                                   : "Status_0",
        "Select-full-range-join"                             : "Status_0",
        "Select-range"                                       : "Status_0",
        "Select-range-check"                                 : "Status_0",
        "Select-scan"                                        : "Status_0",
        "Subordinate-open-temp-tables"                             : "Status_0",
        "Subordinate-retried-transactions"                         : "Status_0",
        "Slow-launch-threads"                                : "Status_0",
        "Slow-queries"                                       : "Status_0",
        "Sort-merge-passes"                                  : "Status_0",
        "Sort-range"                                         : "Status_0",
        "Sort-rows"                                          : "Status_0",
        "Sort-scan"                                          : "Status_0",
        "Ssl-accept-renegotiates"                            : "Status_0",
        "Ssl-accepts"                                        : "Status_0",
        "Ssl-callback-cache-hits"                            : "Status_0",
        "Ssl-cipher"                                         : "Status_0",
        "Ssl-cipher-list"                                    : "Status_0",
        "Ssl-client-connects"                                : "Status_0",
        "Ssl-connect-renegotiates"                           : "Status_0",
        "Ssl-ctx-verify-depth"                               : "Status_0",
        "Ssl-ctx-verify-mode"                                : "Status_0",
        "Ssl-default-timeout"                                : "Status_0",
        "Ssl-finished-accepts"                               : "Status_0",
        "Ssl-finished-connects"                              : "Status_0",
        "Ssl-server-not-after"                               : "Status_0",
        "Ssl-server-not-before"                              : "Status_0",
        "Ssl-session-cache-hits"                             : "Status_0",
        "Ssl-session-cache-misses"                           : "Status_0",
        "Ssl-session-cache-mode"                             : "Status_0",
        "Ssl-session-cache-overflows"                        : "Status_0",
        "Ssl-session-cache-size"                             : "Status_0",
        "Ssl-session-cache-timeouts"                         : "Status_0",
        "Ssl-sessions-reused"                                : "Status_0",
        "Ssl-used-session-cache-entries"                     : "Status_0",
        "Ssl-verify-depth"                                   : "Status_0",
        "Ssl-verify-mode"                                    : "Status_0",
        "Ssl-version"                                        : "Status_0",
        "Table-locks-immediate"                              : "Status_0",
        "Table-locks-waited"                                 : "Status_0",
        "Table-open-cache-hits"                              : "Status_0",
        "Table-open-cache-misses"                            : "Status_0",
        "Table-open-cache-overflows"                         : "Status_0",
        "Tc-log-max-pages-used"                              : "Status_0",
        "Tc-log-page-size"                                   : "Status_0",
        "Tc-log-page-waits"                                  : "Status_0",
        "Threads-cached"                                     : "Status_0",
        "Threads-connected"                                  : "Status_0",
        "Threads-created"                                    : "Status_0",
        "Threads-running"                                    : "Status_0",
        "Uptime"                                             : "Status_0",
        "Uptime-since-flush-status"                          : "Status_0",

        // processSQL      string = "SELECT ID, STATE FROM INFORMATION_SCHEMA.PROCESSLIST;"
        "State-closing-tables"                               : "Process_0",
        "State-copying-to-tmp-table"                         : "Process_0",
        "State-end"                                          : "Process_0",
        "State-freeing-items"                                : "Process_0",
        "State-init"                                         : "Process_0",
        "State-locked"                                       : "Process_0",
        "State-login"                                        : "Process_0",
        "State-preparing"                                    : "Process_0",
        "State-reading-from-net"                             : "Process_0",
        "State-sending-data"                                 : "Process_0",
        "State-sorting-result"                               : "Process_0",
        "State-statistics"                                   : "Process_0",
        "State-updating"                                     : "Process_0",
        "State-writing-to-net"                               : "Process_0",
        "State-none"                                         : "Process_0",
        "State-other"                                        : "Process_0",
        // engineSQL       string = "SHOW ENGINE InnoDB STATUS;"
        "spin-waits"                                               : "Engine_0",
        "spin-rounds"                                              : "Engine_0",
        "os-waits"                                                 : "Engine_0",
        "innodb-sem-waits"                                         : "Engine_0",
        "innodb-sem-wait-time-ms"                                  : "Engine_0",
        "innodb-transactions"                                      : "Engine_0",
        "history-list"                                             : "Engine_0",
        "current-transactions"                                     : "Engine_0",
        "active-transactions"                                      : "Engine_0",
        "innodb-lock-wait-secs"                                    : "Engine_0",
        "read-views"                                               : "Engine_0",
        "innodb-tables-in-use"                                     : "Engine_0",
        "innodb-locked-tables"                                     : "Engine_0",
        "innodb-lock-structs"                                      : "Engine_0",
        "locked-transactions"                                      : "Engine_0",
        "file-reads"                                               : "Engine_0",
        "file-writes"                                              : "Engine_0",
        "file-fsyncs"                                              : "Engine_0",
        "pending-normal-aio-reads"                                 : "Engine_0",
        "pending-normal-aio-writes"                                : "Engine_0",
        "pending-ibuf-aio-reads"                                   : "Engine_0",
        "pending-aio-log-ios"                                      : "Engine_0",
        "pending-aio-sync-ios"                                     : "Engine_0",
        "pending-log-flushes"                                      : "Engine_0",
        "ibuf-used-cells"                                          : "Engine_0",
        "ibuf-free-cells"                                          : "Engine_0",
        "ibuf-cell-count"                                          : "Engine_0",
        "ibuf-merges"                                              : "Engine_0",
        "ibuf-inserts"                                             : "Engine_0",
        "ibuf-merged"                                              : "Engine_0",
        "hash-index-cells-total"                                   : "Engine_0",
        "hash-index-cells-used"                                    : "Engine_0",
        "log-writes"                                               : "Engine_0",
        "pending-log-writes"                                       : "Engine_0",
        "pending-chkp-writes"                                      : "Engine_0",
        "log-bytes-written"                                        : "Engine_0",
        "log-bytes-flushed"                                        : "Engine_0",
        "last-checkpoint"                                          : "Engine_0",
        "total-mem-alloc"                                          : "Engine_0",
        "additional-pool-alloc"                                    : "Engine_0",
        "pending-buf-pool-flushes"                                 : "Engine_0",
        "adaptive-hash-memory"                                     : "Engine_0",
        "page-hash-memory"                                         : "Engine_0",
        "dictionary-cache-memory"                                  : "Engine_0",
        "file-system-memory"                                       : "Engine_0",
        "lock-system-memory"                                       : "Engine_0",
        "recovery-system-memory"                                   : "Engine_0",
        "thread-hash-memory"                                       : "Engine_0",
        "innodb-io-pattern-memory"                                 : "Engine_0",
        "pool-size"                                                : "Engine_0",
        "free-pages"                                               : "Engine_0",
        "database-pages"                                           : "Engine_0",
        "modified-pages"                                           : "Engine_0",
        "pages-read"                                               : "Engine_0",
        "pages-created"                                            : "Engine_0",
        "pages-written"                                            : "Engine_0",
        "rows-inserted"                                            : "Engine_0",
        "rows-updated"                                             : "Engine_0",
        "rows-deleted"                                             : "Engine_0",
        "rows-read"                                                : "Engine_0",
        "queries-inside"                                           : "Engine_0",
        "queries-queued"                                           : "Engine_0",
        "Key-buf-bytes-unflushed"                                  : "Engine_0",
        "Key-buf-bytes-used"                                       : "Engine_0",
        "uncheckpointed-bytes"                                     : "Engine_0",
        "unflushed-log"                                            : "Engine_0",

        // variablesSQL    string = "SHOW GLOBAL VARIABLES;"
        "autocommit"                                                     : "Variables_0",
        "automatic-sp-privileges"                                        : "Variables_0",
        "auto-generate-certs"                                            : "Variables_0",
        "auto-increment-increment"                                       : "Variables_0",
        "auto-increment-offset"                                          : "Variables_0",
        "avoid-temporal-upgrade"                                         : "Variables_0",
        "back-log"                                                       : "Variables_0",
        "basedir"                                                        : "Variables_0",
        "big-tables"                                                     : "Variables_0",
        "bind-address"                                                   : "Variables_0",
        "binlog-cache-size"                                              : "Variables_0",
        "binlog-checksum"                                                : "Variables_0",
        "binlog-direct-non-transactional-updates"                        : "Variables_0",
        "binlog-error-action"                                            : "Variables_0",
        "binlog-format"                                                  : "Variables_0",
        "binlog-group-commit-sync-delay"                                 : "Variables_0",
        "binlog-group-commit-sync-no-delay-count"                        : "Variables_0",
        "binlog-gtid-simple-recovery"                                    : "Variables_0",
        "binlog-max-flush-queue-time"                                    : "Variables_0",
        "binlog-order-commits"                                           : "Variables_0",
        "binlog-rows-query-log-events"                                   : "Variables_0",
        "binlog-row-image"                                               : "Variables_0",
        "binlog-skip-flush-commands"                                     : "Variables_0",
        "binlog-space-limit"                                             : "Variables_0",
        "binlog-stmt-cache-size"                                         : "Variables_0",
        "binlog-transaction-dependency-history-size"                     : "Variables_0",
        "binlog-transaction-dependency-tracking"                         : "Variables_0",
        "block-encryption-mode"                                          : "Variables_0",
        "bulk-insert-buffer-size"                                        : "Variables_0",
        "character-sets-dir"                                             : "Variables_0",
        "character-set-client"                                           : "Variables_0",
        "character-set-connection"                                       : "Variables_0",
        "character-set-database"                                         : "Variables_0",
        "character-set-filesystem"                                       : "Variables_0",
        "character-set-results"                                          : "Variables_0",
        "character-set-server"                                           : "Variables_0",
        "character-set-system"                                           : "Variables_0",
        "check-proxy-users"                                              : "Variables_0",
        "collation-connection"                                           : "Variables_0",
        "collation-database"                                             : "Variables_0",
        "collation-server"                                               : "Variables_0",
        "completion-type"                                                : "Variables_0",
        "concurrent-insert"                                              : "Variables_0",
        "connect-timeout"                                                : "Variables_0",
        "core-file"                                                      : "Variables_0",
        "csv-mode"                                                       : "Variables_0",
        "datadir"                                                        : "Variables_0",
        "datetime-format"                                                : "Variables_0",
        "date-format"                                                    : "Variables_0",
        "default-authentication-plugin"                                  : "Variables_0",
        "default-password-lifetime"                                      : "Variables_0",
        "default-storage-engine"                                         : "Variables_0",
        "default-tmp-storage-engine"                                     : "Variables_0",
        "default-week-format"                                            : "Variables_0",
        "delayed-insert-limit"                                           : "Variables_0",
        "delayed-insert-timeout"                                         : "Variables_0",
        "delayed-queue-size"                                             : "Variables_0",
        "delay-key-write"                                                : "Variables_0",
        "disabled-storage-engines"                                       : "Variables_0",
        "disconnect-on-expired-password"                                 : "Variables_0",
        "div-precision-increment"                                        : "Variables_0",
        "encrypt-binlog"                                                 : "Variables_0",
        "encrypt-tmp-files"                                              : "Variables_0",
        "end-markers-in-json"                                            : "Variables_0",
        "enforce-gtid-consistency"                                       : "Variables_0",
        "enforce-storage-engine"                                         : "Variables_0",
        "eq-range-index-dive-limit"                                      : "Variables_0",
        "event-scheduler"                                                : "Variables_0",
        "expand-fast-index-creation"                                     : "Variables_0",
        "expire-logs-days"                                               : "Variables_0",
        "explicit-defaults-for-timestamp"                                : "Variables_0",
        "extra-max-connections"                                          : "Variables_0",
        "extra-port"                                                     : "Variables_0",
        "flush"                                                          : "Variables_0",
        "flush-time"                                                     : "Variables_0",
        "foreign-key-checks"                                             : "Variables_0",
        "ft-boolean-syntax"                                              : "Variables_0",
        "ft-max-word-len"                                                : "Variables_0",
        "ft-min-word-len"                                                : "Variables_0",
        "ft-query-expansion-limit"                                       : "Variables_0",
        "ft-query-extra-word-chars"                                      : "Variables_0",
        "ft-stopword-file"                                               : "Variables_0",
        "general-log"                                                    : "Variables_0",
        "general-log-file"                                               : "Variables_0",
        "group-concat-max-len"                                           : "Variables_0",
        "gtid-executed"                                                  : "Variables_0",
        "gtid-executed-compression-period"                               : "Variables_0",
        "gtid-mode"                                                      : "Variables_0",
        "gtid-owned"                                                     : "Variables_0",
        "gtid-purged"                                                    : "Variables_0",
        "have-backup-locks"                                              : "Variables_0",
        "have-backup-safe-binlog-info"                                   : "Variables_0",
        "have-compress"                                                  : "Variables_0",
        "have-crypt"                                                     : "Variables_0",
        "have-dynamic-loading"                                           : "Variables_0",
        "have-geometry"                                                  : "Variables_0",
        "have-openssl"                                                   : "Variables_0",
        "have-profiling"                                                 : "Variables_0",
        "have-query-cache"                                               : "Variables_0",
        "have-rtree-keys"                                                : "Variables_0",
        "have-snapshot-cloning"                                          : "Variables_0",
        "have-ssl"                                                       : "Variables_0",
        "have-statement-timeout"                                         : "Variables_0",
        "have-symlink"                                                   : "Variables_0",
        "hostname"                                                       : "Variables_0",
        "host-cache-size"                                                : "Variables_0",
        "ignore-builtin-innodb"                                          : "Variables_0",
        "ignore-db-dirs"                                                 : "Variables_0",
        "init-connect"                                                   : "Variables_0",
        "init-file"                                                      : "Variables_0",
        "init-subordinate"                                                     : "Variables_0",
        "innodb-adaptive-flushing"                                       : "Variables_0",
        "innodb-adaptive-flushing-lwm"                                   : "Variables_0",
        "innodb-adaptive-hash-index"                                     : "Variables_0",
        "innodb-adaptive-hash-index-parts"                               : "Variables_0",
        "innodb-adaptive-max-sleep-delay"                                : "Variables_0",
        "innodb-api-bk-commit-interval"                                  : "Variables_0",
        "innodb-api-disable-rowlock"                                     : "Variables_0",
        "innodb-api-enable-binlog"                                       : "Variables_0",
        "innodb-api-enable-mdl"                                          : "Variables_0",
        "innodb-api-trx-level"                                           : "Variables_0",
        "innodb-autoextend-increment"                                    : "Variables_0",
        "innodb-autoinc-lock-mode"                                       : "Variables_0",
        "innodb-background-scrub-data-check-interval"                    : "Variables_0",
        "innodb-background-scrub-data-compressed"                        : "Variables_0",
        "innodb-background-scrub-data-interval"                          : "Variables_0",
        "innodb-background-scrub-data-uncompressed"                      : "Variables_0",
        "innodb-buffer-pool-chunk-size"                                  : "Variables_0",
        "innodb-buffer-pool-dump-at-shutdown"                            : "Variables_0",
        "innodb-buffer-pool-dump-now"                                    : "Variables_0",
        "innodb-buffer-pool-dump-pct"                                    : "Variables_0",
        "innodb-buffer-pool-filename"                                    : "Variables_0",
        "innodb-buffer-pool-instances"                                   : "Variables_0",
        "innodb-buffer-pool-load-abort"                                  : "Variables_0",
        "innodb-buffer-pool-load-at-startup"                             : "Variables_0",
        "innodb-buffer-pool-load-now"                                    : "Variables_0",
        "innodb-buffer-pool-size"                                        : "Variables_0",
        "innodb-change-buffering"                                        : "Variables_0",
        "innodb-change-buffer-max-size"                                  : "Variables_0",
        "innodb-checksums"                                               : "Variables_0",
        "innodb-checksum-algorithm"                                      : "Variables_0",
        "innodb-cleaner-lsn-age-factor"                                  : "Variables_0",
        "innodb-cmp-per-index-enabled"                                   : "Variables_0",
        "innodb-commit-concurrency"                                      : "Variables_0",
        "innodb-compressed-columns-threshold"                            : "Variables_0",
        "innodb-compressed-columns-zip-level"                            : "Variables_0",
        "innodb-compression-failure-threshold-pct"                       : "Variables_0",
        "innodb-compression-level"                                       : "Variables_0",
        "innodb-compression-pad-pct-max"                                 : "Variables_0",
        "innodb-concurrency-tickets"                                     : "Variables_0",
        "innodb-corrupt-table-action"                                    : "Variables_0",
        "innodb-data-file-path"                                          : "Variables_0",
        "innodb-data-home-dir"                                           : "Variables_0",
        "innodb-deadlock-detect"                                         : "Variables_0",
        "innodb-default-encryption-key-id"                               : "Variables_0",
        "innodb-default-row-format"                                      : "Variables_0",
        "innodb-disable-sort-file-cache"                                 : "Variables_0",
        "innodb-doublewrite"                                             : "Variables_0",
        "innodb-empty-free-list-algorithm"                               : "Variables_0",
        "innodb-encryption-rotate-key-age"                               : "Variables_0",
        "innodb-encryption-rotation-iops"                                : "Variables_0",
        "innodb-encryption-threads"                                      : "Variables_0",
        "innodb-encrypt-online-alter-logs"                               : "Variables_0",
        "innodb-encrypt-tables"                                          : "Variables_0",
        "innodb-fast-shutdown"                                           : "Variables_0",
        "innodb-file-format"                                             : "Variables_0",
        "innodb-file-format-check"                                       : "Variables_0",
        "innodb-file-format-max"                                         : "Variables_0",
        "innodb-file-per-table"                                          : "Variables_0",
        "innodb-fill-factor"                                             : "Variables_0",
        "innodb-flushing-avg-loops"                                      : "Variables_0",
        "innodb-flush-log-at-timeout"                                    : "Variables_0",
        "innodb-flush-log-at-trx-commit"                                 : "Variables_0",
        "innodb-flush-method"                                            : "Variables_0",
        "innodb-flush-neighbors"                                         : "Variables_0",
        "innodb-flush-sync"                                              : "Variables_0",
        "innodb-force-load-corrupted"                                    : "Variables_0",
        "innodb-force-recovery"                                          : "Variables_0",
        "innodb-ft-aux-table"                                            : "Variables_0",
        "innodb-ft-cache-size"                                           : "Variables_0",
        "innodb-ft-enable-diag-print"                                    : "Variables_0",
        "innodb-ft-enable-stopword"                                      : "Variables_0",
        "innodb-ft-ignore-stopwords"                                     : "Variables_0",
        "innodb-ft-max-token-size"                                       : "Variables_0",
        "innodb-ft-min-token-size"                                       : "Variables_0",
        "innodb-ft-num-word-optimize"                                    : "Variables_0",
        "innodb-ft-result-cache-limit"                                   : "Variables_0",
        "innodb-ft-server-stopword-table"                                : "Variables_0",
        "innodb-ft-sort-pll-degree"                                      : "Variables_0",
        "innodb-ft-total-cache-size"                                     : "Variables_0",
        "innodb-ft-user-stopword-table"                                  : "Variables_0",
        "innodb-immediate-scrub-data-uncompressed"                       : "Variables_0",
        "innodb-io-capacity"                                             : "Variables_0",
        "innodb-io-capacity-max"                                         : "Variables_0",
        "innodb-kill-idle-transaction"                                   : "Variables_0",
        "innodb-large-prefix"                                            : "Variables_0",
        "innodb-locks-unsafe-for-binlog"                                 : "Variables_0",
        "innodb-lock-wait-timeout"                                       : "Variables_0",
        "innodb-log-buffer-size"                                         : "Variables_0",
        "innodb-log-checksums"                                           : "Variables_0",
        "innodb-log-compressed-pages"                                    : "Variables_0",
        "innodb-log-files-in-group"                                      : "Variables_0",
        "innodb-log-file-size"                                           : "Variables_0",
        "innodb-log-group-home-dir"                                      : "Variables_0",
        "innodb-log-write-ahead-size"                                    : "Variables_0",
        "innodb-lru-scan-depth"                                          : "Variables_0",
        "innodb-max-bitmap-file-size"                                    : "Variables_0",
        "innodb-max-changed-pages"                                       : "Variables_0",
        "innodb-max-dirty-pages-pct"                                     : "Variables_0",
        "innodb-max-dirty-pages-pct-lwm"                                 : "Variables_0",
        "innodb-max-purge-lag"                                           : "Variables_0",
        "innodb-max-purge-lag-delay"                                     : "Variables_0",
        "innodb-max-undo-log-size"                                       : "Variables_0",
        "innodb-monitor-disable"                                         : "Variables_0",
        "innodb-monitor-enable"                                          : "Variables_0",
        "innodb-monitor-reset"                                           : "Variables_0",
        "innodb-monitor-reset-all"                                       : "Variables_0",
        "innodb-numa-interleave"                                         : "Variables_0",
        "innodb-old-blocks-pct"                                          : "Variables_0",
        "innodb-old-blocks-time"                                         : "Variables_0",
        "innodb-online-alter-log-max-size"                               : "Variables_0",
        "innodb-open-files"                                              : "Variables_0",
        "innodb-optimize-fulltext-only"                                  : "Variables_0",
        "innodb-page-cleaners"                                           : "Variables_0",
        "innodb-page-size"                                               : "Variables_0",
        "innodb-parallel-dblwr-encrypt"                                  : "Variables_0",
        "innodb-parallel-doublewrite-path"                               : "Variables_0",
        "innodb-print-all-deadlocks"                                     : "Variables_0",
        "innodb-print-lock-wait-timeout-info"                            : "Variables_0",
        "innodb-purge-batch-size"                                        : "Variables_0",
        "innodb-purge-rseg-truncate-frequency"                           : "Variables_0",
        "innodb-purge-threads"                                           : "Variables_0",
        "innodb-random-read-ahead"                                       : "Variables_0",
        "innodb-read-ahead-threshold"                                    : "Variables_0",
        "innodb-read-io-threads"                                         : "Variables_0",
        "innodb-read-only"                                               : "Variables_0",
        "innodb-redo-log-encrypt"                                        : "Variables_0",
        "innodb-replication-delay"                                       : "Variables_0",
        "innodb-rollback-on-timeout"                                     : "Variables_0",
        "innodb-rollback-segments"                                       : "Variables_0",
        "innodb-scrub-log"                                               : "Variables_0",
        "innodb-scrub-log-speed"                                         : "Variables_0",
        "innodb-show-locks-held"                                         : "Variables_0",
        "innodb-show-verbose-locks"                                      : "Variables_0",
        "innodb-sort-buffer-size"                                        : "Variables_0",
        "innodb-spin-wait-delay"                                         : "Variables_0",
        "innodb-stats-auto-recalc"                                       : "Variables_0",
        "innodb-stats-include-delete-marked"                             : "Variables_0",
        "innodb-stats-method"                                            : "Variables_0",
        "innodb-stats-on-metadata"                                       : "Variables_0",
        "innodb-stats-persistent"                                        : "Variables_0",
        "innodb-stats-persistent-sample-pages"                           : "Variables_0",
        "innodb-stats-sample-pages"                                      : "Variables_0",
        "innodb-stats-transient-sample-pages"                            : "Variables_0",
        "innodb-status-output"                                           : "Variables_0",
        "innodb-status-output-locks"                                     : "Variables_0",
        "innodb-strict-mode"                                             : "Variables_0",
        "innodb-support-xa"                                              : "Variables_0",
        "innodb-sync-array-size"                                         : "Variables_0",
        "innodb-sync-spin-loops"                                         : "Variables_0",
        "innodb-sys-tablespace-encrypt"                                  : "Variables_0",
        "innodb-table-locks"                                             : "Variables_0",
        "innodb-temp-data-file-path"                                     : "Variables_0",
        "innodb-temp-tablespace-encrypt"                                 : "Variables_0",
        "innodb-thread-concurrency"                                      : "Variables_0",
        "innodb-thread-sleep-delay"                                      : "Variables_0",
        "innodb-tmpdir"                                                  : "Variables_0",
        "innodb-track-changed-pages"                                     : "Variables_0",
        "innodb-undo-directory"                                          : "Variables_0",
        "innodb-undo-logs"                                               : "Variables_0",
        "innodb-undo-log-encrypt"                                        : "Variables_0",
        "innodb-undo-log-truncate"                                       : "Variables_0",
        "innodb-undo-tablespaces"                                        : "Variables_0",
        "innodb-use-global-flush-log-at-trx-commit"                      : "Variables_0",
        "innodb-use-native-aio"                                          : "Variables_0",
        "innodb-version"                                                 : "Variables_0",
        "innodb-write-io-threads"                                        : "Variables_0",
        "interactive-timeout"                                            : "Variables_0",
        "internal-tmp-disk-storage-engine"                               : "Variables_0",
        "join-buffer-size"                                               : "Variables_0",
        "keep-files-on-create"                                           : "Variables_0",
        "keyring-operations"                                             : "Variables_0",
        "key-buffer-size"                                                : "Variables_0",
        "key-cache-age-threshold"                                        : "Variables_0",
        "key-cache-block-size"                                           : "Variables_0",
        "key-cache-division-limit"                                       : "Variables_0",
        "kill-idle-transaction"                                          : "Variables_0",
        "large-files-support"                                            : "Variables_0",
        "large-pages"                                                    : "Variables_0",
        "large-page-size"                                                : "Variables_0",
        "lc-messages"                                                    : "Variables_0",
        "lc-messages-dir"                                                : "Variables_0",
        "lc-time-names"                                                  : "Variables_0",
        "license"                                                        : "Variables_0",
        "local-infile"                                                   : "Variables_0",
        "locked-in-memory"                                               : "Variables_0",
        "lock-wait-timeout"                                              : "Variables_0",
        "log-bin"                                                        : "Variables_0",
        "log-bin-basename"                                               : "Variables_0",
        "log-bin-index"                                                  : "Variables_0",
        "log-bin-trust-function-creators"                                : "Variables_0",
        "log-bin-use-v1-row-events"                                      : "Variables_0",
        "log-builtin-as-identified-by-password"                          : "Variables_0",
        "log-error"                                                      : "Variables_0",
        "log-error-verbosity"                                            : "Variables_0",
        "log-output"                                                     : "Variables_0",
        "log-queries-not-using-indexes"                                  : "Variables_0",
        "log-subordinate-updates"                                              : "Variables_0",
        "log-slow-admin-statements"                                      : "Variables_0",
        "log-slow-filter"                                                : "Variables_0",
        "log-slow-rate-limit"                                            : "Variables_0",
        "log-slow-rate-type"                                             : "Variables_0",
        "log-slow-subordinate-statements"                                      : "Variables_0",
        "log-slow-sp-statements"                                         : "Variables_0",
        "log-slow-verbosity"                                             : "Variables_0",
        "log-statements-unsafe-for-binlog"                               : "Variables_0",
        "log-syslog"                                                     : "Variables_0",
        "log-syslog-facility"                                            : "Variables_0",
        "log-syslog-include-pid"                                         : "Variables_0",
        "log-syslog-tag"                                                 : "Variables_0",
        "log-throttle-queries-not-using-indexes"                         : "Variables_0",
        "log-timestamps"                                                 : "Variables_0",
        "log-warnings"                                                   : "Variables_0",
        "long-query-time"                                                : "Variables_0",
        "lower-case-file-system"                                         : "Variables_0",
        "lower-case-table-names"                                         : "Variables_0",
        "low-priority-updates"                                           : "Variables_0",
        "main-info-repository"                                         : "Variables_0",
        "main-verify-checksum"                                         : "Variables_0",
        "max-allowed-packet"                                             : "Variables_0",
        "max-binlog-cache-size"                                          : "Variables_0",
        "max-binlog-files"                                               : "Variables_0",
        "max-binlog-size"                                                : "Variables_0",
        "max-binlog-stmt-cache-size"                                     : "Variables_0",
        "max-connections"                                                : "Variables_0",
        "max-connect-errors"                                             : "Variables_0",
        "max-delayed-threads"                                            : "Variables_0",
        "max-digest-length"                                              : "Variables_0",
        "max-error-count"                                                : "Variables_0",
        "max-execution-time"                                             : "Variables_0",
        "max-heap-table-size"                                            : "Variables_0",
        "max-insert-delayed-threads"                                     : "Variables_0",
        "max-join-size"                                                  : "Variables_0",
        "max-length-for-sort-data"                                       : "Variables_0",
        "max-points-in-geometry"                                         : "Variables_0",
        "max-prepared-stmt-count"                                        : "Variables_0",
        "max-relay-log-size"                                             : "Variables_0",
        "max-seeks-for-key"                                              : "Variables_0",
        "max-slowlog-files"                                              : "Variables_0",
        "max-slowlog-size"                                               : "Variables_0",
        "max-sort-length"                                                : "Variables_0",
        "max-sp-recursion-depth"                                         : "Variables_0",
        "max-tmp-tables"                                                 : "Variables_0",
        "max-user-connections"                                           : "Variables_0",
        "max-write-lock-count"                                           : "Variables_0",
        "metadata-locks-cache-size"                                      : "Variables_0",
        "metadata-locks-hash-instances"                                  : "Variables_0",
        "min-examined-row-limit"                                         : "Variables_0",
        "multi-range-count"                                              : "Variables_0",
        "myisam-data-pointer-size"                                       : "Variables_0",
        "myisam-max-sort-file-size"                                      : "Variables_0",
        "myisam-mmap-size"                                               : "Variables_0",
        "myisam-recover-options"                                         : "Variables_0",
        "myisam-repair-threads"                                          : "Variables_0",
        "myisam-sort-buffer-size"                                        : "Variables_0",
        "myisam-stats-method"                                            : "Variables_0",
        "myisam-use-mmap"                                                : "Variables_0",
        "mysql-native-password-proxy-users"                              : "Variables_0",
        "net-buffer-length"                                              : "Variables_0",
        "net-read-timeout"                                               : "Variables_0",
        "net-retry-count"                                                : "Variables_0",
        "net-write-timeout"                                              : "Variables_0",
        "new"                                                            : "Variables_0",
        "ngram-token-size"                                               : "Variables_0",
        "offline-mode"                                                   : "Variables_0",
        "old"                                                            : "Variables_0",
        "old-alter-table"                                                : "Variables_0",
        "old-passwords"                                                  : "Variables_0",
        "open-files-limit"                                               : "Variables_0",
        "optimizer-prune-level"                                          : "Variables_0",
        "optimizer-search-depth"                                         : "Variables_0",
        "optimizer-switch"                                               : "Variables_0",
        "optimizer-trace"                                                : "Variables_0",
        "optimizer-trace-features"                                       : "Variables_0",
        "optimizer-trace-limit"                                          : "Variables_0",
        "optimizer-trace-max-mem-size"                                   : "Variables_0",
        "optimizer-trace-offset"                                         : "Variables_0",
        "parser-max-mem-size"                                            : "Variables_0",
        "performance-schema"                                             : "Variables_0",
        "performance-schema-accounts-size"                               : "Variables_0",
        "performance-schema-digests-size"                                : "Variables_0",
        "performance-schema-events-stages-history-long-size"             : "Variables_0",
        "performance-schema-events-stages-history-size"                  : "Variables_0",
        "performance-schema-events-statements-history-long-size"         : "Variables_0",
        "performance-schema-events-statements-history-size"              : "Variables_0",
        "performance-schema-events-transactions-history-long-size"       : "Variables_0",
        "performance-schema-events-transactions-history-size"            : "Variables_0",
        "performance-schema-events-waits-history-long-size"              : "Variables_0",
        "performance-schema-events-waits-history-size"                   : "Variables_0",
        "performance-schema-hosts-size"                                  : "Variables_0",
        "performance-schema-max-cond-classes"                            : "Variables_0",
        "performance-schema-max-cond-instances"                          : "Variables_0",
        "performance-schema-max-digest-length"                           : "Variables_0",
        "performance-schema-max-file-classes"                            : "Variables_0",
        "performance-schema-max-file-handles"                            : "Variables_0",
        "performance-schema-max-file-instances"                          : "Variables_0",
        "performance-schema-max-index-stat"                              : "Variables_0",
        "performance-schema-max-memory-classes"                          : "Variables_0",
        "performance-schema-max-metadata-locks"                          : "Variables_0",
        "performance-schema-max-mutex-classes"                           : "Variables_0",
        "performance-schema-max-mutex-instances"                         : "Variables_0",
        "performance-schema-max-prepared-statements-instances"           : "Variables_0",
        "performance-schema-max-program-instances"                       : "Variables_0",
        "performance-schema-max-rwlock-classes"                          : "Variables_0",
        "performance-schema-max-rwlock-instances"                        : "Variables_0",
        "performance-schema-max-socket-classes"                          : "Variables_0",
        "performance-schema-max-socket-instances"                        : "Variables_0",
        "performance-schema-max-sql-text-length"                         : "Variables_0",
        "performance-schema-max-stage-classes"                           : "Variables_0",
        "performance-schema-max-statement-classes"                       : "Variables_0",
        "performance-schema-max-statement-stack"                         : "Variables_0",
        "performance-schema-max-table-handles"                           : "Variables_0",
        "performance-schema-max-table-instances"                         : "Variables_0",
        "performance-schema-max-table-lock-stat"                         : "Variables_0",
        "performance-schema-max-thread-classes"                          : "Variables_0",
        "performance-schema-max-thread-instances"                        : "Variables_0",
        "performance-schema-session-connect-attrs-size"                  : "Variables_0",
        "performance-schema-setup-actors-size"                           : "Variables_0",
        "performance-schema-setup-objects-size"                          : "Variables_0",
        "performance-schema-users-size"                                  : "Variables_0",
        "pid-file"                                                       : "Variables_0",
        "plugin-dir"                                                     : "Variables_0",
        "port"                                                           : "Variables_0",
        "preload-buffer-size"                                            : "Variables_0",
        "profiling"                                                      : "Variables_0",
        "profiling-history-size"                                         : "Variables_0",
        "protocol-version"                                               : "Variables_0",
        "proxy-protocol-networks"                                        : "Variables_0",
        "query-alloc-block-size"                                         : "Variables_0",
        "query-cache-limit"                                              : "Variables_0",
        "query-cache-min-res-unit"                                       : "Variables_0",
        "query-cache-size"                                               : "Variables_0",
        "query-cache-strip-comments"                                     : "Variables_0",
        "query-cache-type"                                               : "Variables_0",
        "query-cache-wlock-invalidate"                                   : "Variables_0",
        "query-prealloc-size"                                            : "Variables_0",
        "range-alloc-block-size"                                         : "Variables_0",
        "range-optimizer-max-mem-size"                                   : "Variables_0",
        "rbr-exec-mode"                                                  : "Variables_0",
        "read-buffer-size"                                               : "Variables_0",
        "read-only"                                                      : "Variables_0",
        "read-rnd-buffer-size"                                           : "Variables_0",
        "relay-log"                                                      : "Variables_0",
        "relay-log-basename"                                             : "Variables_0",
        "relay-log-index"                                                : "Variables_0",
        "relay-log-info-file"                                            : "Variables_0",
        "relay-log-info-repository"                                      : "Variables_0",
        "relay-log-purge"                                                : "Variables_0",
        "relay-log-recovery"                                             : "Variables_0",
        "relay-log-space-limit"                                          : "Variables_0",
        "report-host"                                                    : "Variables_0",
        "report-password"                                                : "Variables_0",
        "report-port"                                                    : "Variables_0",
        "report-user"                                                    : "Variables_0",
        "require-secure-transport"                                       : "Variables_0",
        "rpl-semi-sync-main-enabled"                                   : "Variables_0",
        "rpl-semi-sync-main-timeout"                                   : "Variables_0",
        "rpl-semi-sync-main-trace-level"                               : "Variables_0",
        "rpl-semi-sync-main-wait-for-subordinate-count"                      : "Variables_0",
        "rpl-semi-sync-main-wait-no-subordinate"                             : "Variables_0",
        "rpl-semi-sync-main-wait-point"                                : "Variables_0",
        "rpl-semi-sync-subordinate-enabled"                                    : "Variables_0",
        "rpl-semi-sync-subordinate-trace-level"                                : "Variables_0",
        "rpl-stop-subordinate-timeout"                                         : "Variables_0",
        "secure-auth"                                                    : "Variables_0",
        "secure-file-priv"                                               : "Variables_0",
        "server-id"                                                      : "Variables_0",
        "server-id-bits"                                                 : "Variables_0",
        "server-uuid"                                                    : "Variables_0",
        "session-track-gtids"                                            : "Variables_0",
        "session-track-schema"                                           : "Variables_0",
        "session-track-state-change"                                     : "Variables_0",
        "session-track-system-variables"                                 : "Variables_0",
        "session-track-transaction-info"                                 : "Variables_0",
        "sha256-password-auto-generate-rsa-keys"                         : "Variables_0",
        "sha256-password-private-key-path"                               : "Variables_0",
        "sha256-password-proxy-users"                                    : "Variables_0",
        "sha256-password-public-key-path"                                : "Variables_0",
        "show-compatibility-56"                                          : "Variables_0",
        "show-create-table-verbosity"                                    : "Variables_0",
        "show-old-temporals"                                             : "Variables_0",
        "skip-external-locking"                                          : "Variables_0",
        "skip-name-resolve"                                              : "Variables_0",
        "skip-networking"                                                : "Variables_0",
        "skip-show-database"                                             : "Variables_0",
        "subordinate-allow-batching"                                           : "Variables_0",
        "subordinate-checkpoint-group"                                         : "Variables_0",
        "subordinate-checkpoint-period"                                        : "Variables_0",
        "subordinate-compressed-protocol"                                      : "Variables_0",
        "subordinate-exec-mode"                                                : "Variables_0",
        "subordinate-load-tmpdir"                                              : "Variables_0",
        "subordinate-max-allowed-packet"                                       : "Variables_0",
        "subordinate-net-timeout"                                              : "Variables_0",
        "subordinate-parallel-type"                                            : "Variables_0",
        "subordinate-parallel-workers"                                         : "Variables_0",
        "subordinate-pending-jobs-size-max"                                    : "Variables_0",
        "subordinate-preserve-commit-order"                                    : "Variables_0",
        "subordinate-rows-search-algorithms"                                   : "Variables_0",
        "subordinate-skip-errors"                                              : "Variables_0",
        "subordinate-sql-verify-checksum"                                      : "Variables_0",
        "subordinate-transaction-retries"                                      : "Variables_0",
        "subordinate-type-conversions"                                         : "Variables_0",
        "slow-launch-time"                                               : "Variables_0",
        "slow-query-log"                                                 : "Variables_0",
        "slow-query-log-always-write-time"                               : "Variables_0",
        "slow-query-log-file"                                            : "Variables_0",
        "slow-query-log-use-global-control"                              : "Variables_0",
        "socket"                                                         : "Variables_0",
        "sort-buffer-size"                                               : "Variables_0",
        "sql-auto-is-null"                                               : "Variables_0",
        "sql-big-selects"                                                : "Variables_0",
        "sql-buffer-result"                                              : "Variables_0",
        "sql-log-bin"                                                    : "Variables_0",
        "sql-log-off"                                                    : "Variables_0",
        "sql-mode"                                                       : "Variables_0",
        "sql-notes"                                                      : "Variables_0",
        "sql-quote-show-create"                                          : "Variables_0",
        "sql-safe-updates"                                               : "Variables_0",
        "sql-select-limit"                                               : "Variables_0",
        "sql-subordinate-skip-counter"                                         : "Variables_0",
        "sql-warnings"                                                   : "Variables_0",
        "ssl-ca"                                                         : "Variables_0",
        "ssl-capath"                                                     : "Variables_0",
        "ssl-cert"                                                       : "Variables_0",
        "ssl-cipher"                                                     : "Variables_0",
        "ssl-crl"                                                        : "Variables_0",
        "ssl-crlpath"                                                    : "Variables_0",
        "ssl-key"                                                        : "Variables_0",
        "stored-program-cache"                                           : "Variables_0",
        "super-read-only"                                                : "Variables_0",
        "sync-binlog"                                                    : "Variables_0",
        "sync-frm"                                                       : "Variables_0",
        "sync-main-info"                                               : "Variables_0",
        "sync-relay-log"                                                 : "Variables_0",
        "sync-relay-log-info"                                            : "Variables_0",
        "system-time-zone"                                               : "Variables_0",
        "table-definition-cache"                                         : "Variables_0",
        "table-open-cache"                                               : "Variables_0",
        "table-open-cache-instances"                                     : "Variables_0",
        "thread-cache-size"                                              : "Variables_0",
        "thread-handling"                                                : "Variables_0",
        "thread-pool-high-prio-mode"                                     : "Variables_0",
        "thread-pool-high-prio-tickets"                                  : "Variables_0",
        "thread-pool-idle-timeout"                                       : "Variables_0",
        "thread-pool-max-threads"                                        : "Variables_0",
        "thread-pool-oversubscribe"                                      : "Variables_0",
        "thread-pool-size"                                               : "Variables_0",
        "thread-pool-stall-limit"                                        : "Variables_0",
        "thread-stack"                                                   : "Variables_0",
        "thread-statistics"                                              : "Variables_0",
        "time-format"                                                    : "Variables_0",
        "time-zone"                                                      : "Variables_0",
        "tls-version"                                                    : "Variables_0",
        "tmpdir"                                                         : "Variables_0",
        "tmp-table-size"                                                 : "Variables_0",
        "tokudb-alter-print-error"                                       : "Variables_0",
        "tokudb-analyze-delete-fraction"                                 : "Variables_0",
        "tokudb-analyze-in-background"                                   : "Variables_0",
        "tokudb-analyze-mode"                                            : "Variables_0",
        "tokudb-analyze-throttle"                                        : "Variables_0",
        "tokudb-analyze-time"                                            : "Variables_0",
        "tokudb-auto-analyze"                                            : "Variables_0",
        "tokudb-block-size"                                              : "Variables_0",
        "tokudb-bulk-fetch"                                              : "Variables_0",
        "tokudb-cachetable-pool-threads"                                 : "Variables_0",
        "tokudb-cache-size"                                              : "Variables_0",
        "tokudb-cardinality-scale-percent"                               : "Variables_0",
        "tokudb-checkpointing-period"                                    : "Variables_0",
        "tokudb-checkpoint-lock"                                         : "Variables_0",
        "tokudb-checkpoint-on-flush-logs"                                : "Variables_0",
        "tokudb-checkpoint-pool-threads"                                 : "Variables_0",
        "tokudb-check-jemalloc"                                          : "Variables_0",
        "tokudb-cleaner-iterations"                                      : "Variables_0",
        "tokudb-cleaner-period"                                          : "Variables_0",
        "tokudb-client-pool-threads"                                     : "Variables_0",
        "tokudb-commit-sync"                                             : "Variables_0",
        "tokudb-compress-buffers-before-eviction"                        : "Variables_0",
        "tokudb-create-index-online"                                     : "Variables_0",
        "tokudb-data-dir"                                                : "Variables_0",
        "tokudb-debug"                                                   : "Variables_0",
        "tokudb-directio"                                                : "Variables_0",
        "tokudb-dir-cmd"                                                 : "Variables_0",
        "tokudb-dir-cmd-last-error"                                      : "Variables_0",
        "tokudb-dir-cmd-last-error-string"                               : "Variables_0",
        "tokudb-dir-per-db"                                              : "Variables_0",
        "tokudb-disable-hot-alter"                                       : "Variables_0",
        "tokudb-disable-prefetching"                                     : "Variables_0",
        "tokudb-disable-slow-alter"                                      : "Variables_0",
        "tokudb-empty-scan"                                              : "Variables_0",
        "tokudb-enable-fast-update"                                      : "Variables_0",
        "tokudb-enable-fast-upsert"                                      : "Variables_0",
        "tokudb-enable-native-partition"                                 : "Variables_0",
        "tokudb-enable-partial-eviction"                                 : "Variables_0",
        "tokudb-fanout"                                                  : "Variables_0",
        "tokudb-fsync-log-period"                                        : "Variables_0",
        "tokudb-fs-reserve-percent"                                      : "Variables_0",
        "tokudb-hide-default-row-format"                                 : "Variables_0",
        "tokudb-killed-time"                                             : "Variables_0",
        "tokudb-last-lock-timeout"                                       : "Variables_0",
        "tokudb-loader-memory-size"                                      : "Variables_0",
        "tokudb-load-save-space"                                         : "Variables_0",
        "tokudb-lock-timeout"                                            : "Variables_0",
        "tokudb-lock-timeout-debug"                                      : "Variables_0",
        "tokudb-log-dir"                                                 : "Variables_0",
        "tokudb-max-lock-memory"                                         : "Variables_0",
        "tokudb-optimize-index-fraction"                                 : "Variables_0",
        "tokudb-optimize-index-name"                                     : "Variables_0",
        "tokudb-optimize-throttle"                                       : "Variables_0",
        "tokudb-prelock-empty"                                           : "Variables_0",
        "tokudb-read-block-size"                                         : "Variables_0",
        "tokudb-read-buf-size"                                           : "Variables_0",
        "tokudb-read-status-frequency"                                   : "Variables_0",
        "tokudb-row-format"                                              : "Variables_0",
        "tokudb-rpl-check-readonly"                                      : "Variables_0",
        "tokudb-rpl-lookup-rows"                                         : "Variables_0",
        "tokudb-rpl-lookup-rows-delay"                                   : "Variables_0",
        "tokudb-rpl-unique-checks"                                       : "Variables_0",
        "tokudb-rpl-unique-checks-delay"                                 : "Variables_0",
        "tokudb-strip-frm-data"                                          : "Variables_0",
        "tokudb-support-xa"                                              : "Variables_0",
        "tokudb-tmp-dir"                                                 : "Variables_0",
        "tokudb-version"                                                 : "Variables_0",
        "tokudb-write-status-frequency"                                  : "Variables_0",
        "transaction-alloc-block-size"                                   : "Variables_0",
        "transaction-isolation"                                          : "Variables_0",
        "transaction-prealloc-size"                                      : "Variables_0",
        "transaction-read-only"                                          : "Variables_0",
        "transaction-write-set-extraction"                               : "Variables_0",
        "tx-isolation"                                                   : "Variables_0",
        "tx-read-only"                                                   : "Variables_0",
        "unique-checks"                                                  : "Variables_0",
        "updatable-views-with-limit"                                     : "Variables_0",
        "userstat"                                                       : "Variables_0",
        "validate-password-check-user-name"                              : "Variables_0",
        "validate-password-dictionary-file"                              : "Variables_0",
        "validate-password-length"                                       : "Variables_0",
        "validate-password-mixed-case-count"                             : "Variables_0",
        "validate-password-number-count"                                 : "Variables_0",
        "validate-password-policy"                                       : "Variables_0",
        "validate-password-special-char-count"                           : "Variables_0",
        "version"                                                        : "Variables_0",
        "version-comment"                                                : "Variables_0",
        "version-compile-machine"                                        : "Variables_0",
        "version-compile-os"                                             : "Variables_0",
        "version-suffix"                                                 : "Variables_0",
        "wait-timeout"                                                   : "Variables_0",
    }
    //fmt.Println(len(MySQLMonitor))
}
