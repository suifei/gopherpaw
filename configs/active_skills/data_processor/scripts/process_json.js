// JSON 数据处理示例
const fs = require('fs');

// 从命令行参数获取输入输出文件
const args = process.argv.slice(2);
if (args.length < 2) {
  console.log("用法: bun process_json.js <input.json> <output.json>");
  process.exit(1);
}

const inputFile = args[0];
const outputFile = args[1];

console.log(`处理文件: ${inputFile} -> ${outputFile}`);

try {
  // 读取 JSON 文件
  const data = JSON.parse(fs.readFileSync(inputFile, 'utf8'));
  
  // 示例处理：添加处理时间戳
  data.processedAt = new Date().toISOString();
  data.processor = "Bun " + Bun.version;
  
  // 写入输出文件
  fs.writeFileSync(outputFile, JSON.stringify(data, null, 2));
  
  console.log("✅ 处理完成");
  console.log(`   输入: ${inputFile}`);
  console.log(`   输出: ${outputFile}`);
  console.log(`   处理时间: ${data.processedAt}`);
  
} catch (error) {
  console.error("❌ 处理失败:", error.message);
  process.exit(1);
}
