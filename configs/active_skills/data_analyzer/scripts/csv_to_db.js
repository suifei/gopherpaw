// CSV 转数据库
const fs = require('fs');
const { parse } = require('csv-parse/sync');
const initSqlJs = require('sql.js');

const args = process.argv.slice(2);
if (args.length < 2) {
  console.log("用法: bun csv_to_db.js <data.csv> <output.db>");
  process.exit(1);
}

const [csvFile, dbFile] = args;

async function csvToDatabase() {
  try {
    console.log(`导入 CSV: ${csvFile} → ${dbFile}`);
    
    // 读取 CSV
    const csvContent = fs.readFileSync(csvFile, 'utf8');
    const records = parse(csvContent, {
      columns: true,
      skip_empty_lines: true
    });
    
    if (records.length === 0) {
      throw new Error("CSV 文件为空");
    }
    
    console.log(`读取 ${records.length} 条记录`);
    
    // 初始化 SQL.js
    const SQL = await initSqlJs();
    const db = new SQL.Database();
    
    // 获取列名
    const columns = Object.keys(records[0]);
    
    // 创建表
    const createTableSQL = `
      CREATE TABLE data (
        ${columns.map(col => `"${col}" TEXT`).join(',\n')}
      )
    `;
    db.run(createTableSQL);
    
    // 插入数据
    const insertSQL = `
      INSERT INTO data (${columns.map(c => `"${c}"`).join(', ')})
      VALUES (${columns.map(() => '?').join(', ')})
    `;
    
    records.forEach(record => {
      const values = columns.map(col => record[col]);
      db.run(insertSQL, values);
    });
    
    // 保存到文件
    const data = db.export();
    const buffer = Buffer.from(data);
    fs.writeFileSync(dbFile, buffer);
    
    console.log(`✅ 数据库创建成功: ${dbFile}`);
    console.log(`   表名: data`);
    console.log(`   列数: ${columns.length}`);
    console.log(`   行数: ${records.length}`);
    
  } catch (error) {
    console.error("❌ 转换失败:", error.message);
    process.exit(1);
  }
}

csvToDatabase();
