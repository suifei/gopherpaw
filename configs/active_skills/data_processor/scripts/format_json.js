// JSON 格式化工具
const fs = require('fs');

const args = process.argv.slice(2);
if (args.length < 1) {
  console.log("用法: bun format_json.js <input.json>");
  process.exit(1);
}

const inputFile = args[0];

try {
  // 读取并解析 JSON
  const data = JSON.parse(fs.readFileSync(inputFile, 'utf8'));
  
  // 格式化输出
  const formatted = JSON.stringify(data, null, 2);
  
  console.log("✅ JSON 格式化结果:\n");
  console.log(formatted);
  console.log("\n✅ 格式化完成");
  
} catch (error) {
  console.error("❌ 格式化失败:", error.message);
  process.exit(1);
}
