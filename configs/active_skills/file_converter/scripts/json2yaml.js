// JSON → YAML
const fs = require('fs');
const YAML = require('yaml');

const args = process.argv.slice(2);
if (args.length < 2) {
  console.log("用法: bun json2yaml.js <input.json> <output.yaml>");
  process.exit(1);
}

const [input, output] = args;

try {
  console.log(`转换: ${input} → ${output}`);
  
  const jsonStr = fs.readFileSync(input, 'utf8');
  const data = JSON.parse(jsonStr);
  
  const yamlStr = YAML.stringify(data);
  fs.writeFileSync(output, yamlStr);
  
  console.log("✅ 转换完成");
} catch (error) {
  console.error("❌ 转换失败:", error.message);
  process.exit(1);
}
