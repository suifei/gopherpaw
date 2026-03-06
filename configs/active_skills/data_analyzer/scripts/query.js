// 执行 SQL 查询
const fs = require('fs');
const initSqlJs = require('sql.js');

const args = process.argv.slice(2);
if (args.length < 2) {
  console.log("用法: bun query.js <database.db> <sql_query>");
  console.log("示例: bun query.js data.db 'SELECT * FROM data LIMIT 10'");
  process.exit(1);
}

const [dbFile, sqlQuery] = args;

async function executeQuery() {
  try {
    console.log(`执行查询: ${sqlQuery}`);
    
    // 加载数据库
    const SQL = await initSqlJs();
    const fileBuffer = fs.readFileSync(dbFile);
    const db = new SQL.Database(fileBuffer);
    
    // 执行查询
    const startTime = Date.now();
    const result = db.exec(sqlQuery);
    const duration = Date.now() - startTime;
    
    if (result.length === 0) {
      console.log("查询结果: 无数据");
      return;
    }
    
    const { columns, values } = result[0];
    
    console.log(`\n查询结果 (${values.length} 行, ${duration}ms):\n`);
    
    // 打印表头
    console.log(columns.map(c => c.padEnd(20)).join(' | '));
    console.log(columns.map(() => '-'.repeat(20)).join('-+-'));
    
    // 打印数据（限制前 20 行）
    values.slice(0, 20).forEach(row => {
      console.log(row.map(v => String(v || 'NULL').padEnd(20)).join(' | '));
    });
    
    if (values.length > 20) {
      console.log(`\n... 还有 ${values.length - 20} 行`);
    }
    
    console.log(`\n✅ 查询完成`);
    
  } catch (error) {
    console.error("❌ 查询失败:", error.message);
    process.exit(1);
  }
}

executeQuery();
