// JSON → CSV (仅支持数组)
const fs = require('fs');
const { stringify } = require('csv-stringify/sync');

const args = process.argv.slice(2);
if (args.length < 2) {
  console.log("用法: bun json2csv.js <input.json> <output.csv>");
  process.exit(1);
}

const [input, output] = args;

try {
  console.log(`转换: ${input} → ${output}`);
  
  const jsonStr = fs.readFileSync(input, 'utf8');
  const data = JSON.parse(jsonStr);
  
  if (!Array.isArray(data)) {
    throw new Error("JSON 数据必须是数组");
  }
  
  if (data.length === 0) {
    throw new Error("JSON 数组为空");
  }
  
  const csvStr = stringify(data, {
    header: true,
    columns: Object.keys(data[0])
  });
  
  fs.writeFileSync(output, csvStr);
  
  console.log("✅ 转换完成");
} catch (error) {
  console.error("❌ 转换失败:", error.message);
  process.exit(1);
}
