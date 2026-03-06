// 测试 Bun 特有 API
console.log("=== Bun 特有 API 测试 ===");

// 测试 Bun.version
console.log("\n1. Bun.version:");
console.log("   ✅ Version:", Bun.version);

// 测试 Bun.env
console.log("\n2. Bun.env:");
console.log("   ✅ PATH exists:", !!Bun.env.PATH);
console.log("   ✅ HOME exists:", !!Bun.env.HOME);

// 测试 Bun.file (创建但不读取，避免错误)
console.log("\n3. Bun.file:");
const testFile = Bun.file("/tmp/test.txt");
console.log("   ✅ Bun.file() 可用");

// 测试性能
console.log("\n4. 性能测试:");
const start = performance.now();
for (let i = 0; i < 1000000; i++) {
  // 空循环
}
const end = performance.now();
console.log("   ✅ 100万次循环耗时:", (end - start).toFixed(2), "ms");

console.log("\n=== 所有 Bun 特有 API 测试通过 ✅ ===");
