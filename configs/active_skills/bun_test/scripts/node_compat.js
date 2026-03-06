// 测试 Node.js 兼容 API
const fs = require('fs');
const path = require('path');

console.log("=== Node.js 兼容性测试 ===");

// 测试 fs 模块
console.log("\n1. 测试 fs 模块:");
const testFile = '/tmp/bun_test.txt';
fs.writeFileSync(testFile, 'Hello from Bun!');
const content = fs.readFileSync(testFile, 'utf8');
console.log("   ✅ 文件写入/读取成功:", content);
fs.unlinkSync(testFile);

// 测试 path 模块
console.log("\n2. 测试 path 模块:");
const joinedPath = path.join('/home', 'user', 'test');
console.log("   ✅ 路径拼接成功:", joinedPath);

// 测试 process 对象
console.log("\n3. 测试 process 对象:");
console.log("   ✅ Platform:", process.platform);
console.log("   ✅ Arch:", process.arch);
console.log("   ✅ CWD:", process.cwd());

console.log("\n=== 所有 Node.js 兼容性测试通过 ✅ ===");
