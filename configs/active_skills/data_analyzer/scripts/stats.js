// 数据统计
const fs = require('fs');
const initSqlJs = require('sql.js');

const args = process.argv.slice(2);
if (args.length < 2) {
  console.log("用法: bun stats.js <database.db> <table_name>");
  process.exit(1);
}

const [dbFile, tableName] = args;

async function generateStats() {
  try {
    console.log(`统计表: ${tableName}\n`);
    
    const SQL = await initSqlJs();
    const fileBuffer = fs.readFileSync(dbFile);
    const db = new SQL.Database(fileBuffer);
    
    // 获取行数
    const countResult = db.exec(`SELECT COUNT(*) as count FROM ${tableName}`);
    const totalRows = countResult[0].values[0][0];
    
    // 获取列信息
    const columnsResult = db.exec(`PRAGMA table_info(${tableName})`);
    const columns = columnsResult[0].values.map(row => ({
      name: row[1],
      type: row[2]
    }));
    
    console.log(`=== 表统计 ===`);
    console.log(`表名: ${tableName}`);
    console.log(`总行数: ${totalRows}`);
    console.log(`列数: ${columns.length}\n`);
    
    console.log(`=== 列统计 ===\n`);
    
    columns.forEach(col => {
      console.log(`列: ${col.name} (${col.type || 'TEXT'})`);
      
      // NULL 值统计
      const nullResult = db.exec(`SELECT COUNT(*) FROM ${tableName} WHERE "${col.name}" IS NULL`);
      const nullCount = nullResult[0].values[0][0];
      console.log(`  NULL 值: ${nullCount} (${((nullCount / totalRows) * 100).toFixed(1)}%)`);
      
      // 唯一值统计
      const distinctResult = db.exec(`SELECT COUNT(DISTINCT "${col.name}") FROM ${tableName}`);
      const distinctCount = distinctResult[0].values[0][0];
      console.log(`  唯一值: ${distinctCount}`);
      
      // 如果看起来是数值，计算统计信息
      const sampleResult = db.exec(`SELECT "${col.name}" FROM ${tableName} WHERE "${col.name}" IS NOT NULL LIMIT 10`);
      const samples = sampleResult[0].values.map(row => parseFloat(row[0]));
      
      if (samples.every(v => !isNaN(v))) {
        const numericResult = db.exec(`
          SELECT 
            MIN("${col.name}") as min,
            MAX("${col.name}") as max,
            AVG("${col.name}") as avg
          FROM ${tableName}
          WHERE "${col.name}" IS NOT NULL
        `);
        
        const [min, max, avg] = numericResult[0].values[0];
        console.log(`  最小值: ${min}`);
        console.log(`  最大值: ${max}`);
        console.log(`  平均值: ${parseFloat(avg).toFixed(2)}`);
      }
      
      console.log();
    });
    
    console.log(`✅ 统计完成`);
    
  } catch (error) {
    console.error("❌ 统计失败:", error.message);
    process.exit(1);
  }
}

generateStats();
