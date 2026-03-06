// YAML → JSON
const fs = require('fs');
const YAML = require('yaml');

const args = process.argv.slice(2);
if (args.length < 2) {
  console.log("用法: bun yaml2json.js <input.yaml> <output.json>");
  process.exit(1);
}

const [input, output] = args;

try {
  console.log(`转换: ${input} → ${output}`);
  
  const yamlStr = fs.readFileSync(input, 'utf8');
  const data = YAML.parse(yamlStr);
  
  const jsonStr = JSON.stringify(data, null, 2);
  fs.writeFileSync(output, jsonStr);
  
  console.log("✅ 转换完成");
} catch (error) {
  console.error("❌ 转换失败:", error.message);
  process.exit(1);
}
