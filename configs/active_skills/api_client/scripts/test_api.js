// API 测试工具
const args = process.argv.slice(2);
if (args.length < 1) {
  console.log("用法: bun test_api.js <url> [method] [data]");
  console.log("示例:");
  console.log("  bun test_api.js https://api.example.com/test");
  console.log("  bun test_api.js https://api.example.com/test POST '{\"key\":\"value\"}'");
  process.exit(1);
}

const [url, method = 'GET', dataStr] = args;

async function testAPI() {
  try {
    console.log(`\n=== API 测试 ===`);
    console.log(`URL: ${url}`);
    console.log(`方法: ${method}`);
    
    const startTime = Date.now();
    const controller = new AbortController();
    const timeout = setTimeout(() => controller.abort(), 10000);
    
    const options = {
      method: method.toUpperCase(),
      headers: {},
      signal: controller.signal
    };
    
    if (dataStr && ['POST', 'PUT', 'PATCH'].includes(options.method)) {
      try {
        const data = JSON.parse(dataStr);
        options.body = JSON.stringify(data);
        options.headers['Content-Type'] = 'application/json';
        console.log(`数据:`, data);
      } catch {
        console.error("❌ JSON 数据格式错误");
        process.exit(1);
      }
    }
    
    console.log(`\n发送请求...`);
    const response = await fetch(url, options);
    clearTimeout(timeout);
    
    const duration = Date.now() - startTime;
    
    console.log(`\n=== 响应 ===`);
    console.log(`状态码: ${response.status} ${response.statusText}`);
    console.log(`耗时: ${duration}ms`);
    console.log(`\n响应头:`);
    response.headers.forEach((value, key) => {
      console.log(`  ${key}: ${value}`);
    });
    
    const contentType = response.headers.get('content-type');
    let result;
    
    if (contentType && contentType.includes('application/json')) {
      result = await response.json();
      console.log(`\n响应体 (JSON):`);
      console.log(JSON.stringify(result, null, 2));
    } else {
      result = await response.text();
      console.log(`\n响应体 (Text):`);
      console.log(result.substring(0, 500));
    }
    
    console.log(`\n✅ API 测试完成`);
    
  } catch (error) {
    if (error.name === 'AbortError') {
      console.error("\n❌ 请求超时（10秒）");
    } else {
      console.error("\n❌ 测试失败:", error.message);
    }
    process.exit(1);
  }
}

testAPI();
