// GET 请求
const args = process.argv.slice(2);
if (args.length < 1) {
  console.log("用法: bun get.js <url>");
  console.log("示例: bun get.js https://api.example.com/data");
  process.exit(1);
}

const url = args[0];

async function getRequest() {
  try {
    console.log(`GET 请求: ${url}`);
    
    const controller = new AbortController();
    const timeout = setTimeout(() => controller.abort(), 10000); // 10秒超时
    
    const response = await fetch(url, {
      method: 'GET',
      signal: controller.signal
    });
    
    clearTimeout(timeout);
    
    console.log(`状态码: ${response.status} ${response.statusText}`);
    console.log(`响应头:`, Object.fromEntries(response.headers));
    
    const contentType = response.headers.get('content-type');
    let data;
    
    if (contentType && contentType.includes('application/json')) {
      data = await response.json();
      console.log('\n响应数据 (JSON):');
      console.log(JSON.stringify(data, null, 2));
    } else {
      data = await response.text();
      console.log('\n响应数据 (Text):');
      console.log(data.substring(0, 500)); // 限制输出长度
    }
    
    console.log('\n✅ GET 请求成功');
    
  } catch (error) {
    if (error.name === 'AbortError') {
      console.error("❌ 请求超时");
    } else {
      console.error("❌ 请求失败:", error.message);
    }
    process.exit(1);
  }
}

getRequest();
